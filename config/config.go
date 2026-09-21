package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configFileName = ".zt-config.json"

	// Environment variable overrides, useful on hosts where writing
	// ~/.zt-config.json isn't desirable (ephemeral CI runners, containers)
	// or where the token needs to be swapped without re-running `zt init`.
	// Each one, if set, takes precedence over the corresponding value in
	// the config file. If the file doesn't exist at all, the three
	// together are enough to run without ever calling `zt init`.
	envAPIToken  = "ZT_API_TOKEN"
	envAccountID = "ZT_ACCOUNT_ID"
	envDomain    = "ZT_DOMAIN"
)

type Config struct {
	APIToken  string `json:"api_token"`
	AccountID string `json:"account_id"`
	Domain    string `json:"domain"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configFileName), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		// No config file — fine, as long as the environment fills in
		// everything needed. Checked below.
	} else {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("malformed config: %w", err)
		}
	}

	if v := os.Getenv(envAPIToken); v != "" {
		cfg.APIToken = v
	}
	if v := os.Getenv(envAccountID); v != "" {
		cfg.AccountID = v
	}
	if v := os.Getenv(envDomain); v != "" {
		cfg.Domain = v
	}

	if cfg.APIToken == "" || cfg.AccountID == "" || cfg.Domain == "" {
		return nil, fmt.Errorf(
			"config incomplete — run `zt init`, or set %s/%s/%s",
			envAPIToken, envAccountID, envDomain,
		)
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

func ConfigFilePath() string {
	path, _ := configPath()
	return path
}
