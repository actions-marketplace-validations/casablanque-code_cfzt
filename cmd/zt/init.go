package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/casablanque-code/cfzt/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Configure Cloudflare credentials",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	bold := color.New(color.Bold).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	warn := color.New(color.FgYellow).SprintFunc()

	fmt.Printf("%s\n\n", bold("⚡ zt init — Cloudflare Zero Trust setup"))

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("  API Token (Cloudflare → My Profile → API Tokens): ")
	token, err := readSecret(reader)
	if err != nil {
		return fmt.Errorf("reading API token: %w", err)
	}

	fmt.Print("  Account ID (right sidebar on any CF dashboard page): ")
	accountID, err := reader.ReadString('\n')
	if err != nil && accountID == "" {
		return fmt.Errorf("reading account ID: %w", err)
	}
	accountID = strings.TrimSpace(accountID)

	fmt.Print("  Domain (e.g. example.com — must be on Cloudflare): ")
	domain, err := reader.ReadString('\n')
	if err != nil && domain == "" {
		return fmt.Errorf("reading domain: %w", err)
	}
	domain = strings.TrimSpace(domain)

	if token == "" || accountID == "" || domain == "" {
		return fmt.Errorf("all fields are required")
	}

	cfg := &config.Config{
		APIToken:  token,
		AccountID: accountID,
		Domain:    domain,
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  → Verifying API token... ")
	cf := newCFClient(token, accountID)
	if err := cf.VerifyToken(); err != nil {
		fmt.Println()
		fmt.Printf("  %s Token verification failed: %v\n", warn("!"), err)
		fmt.Printf("  %s Config saved, but check your token before running zt up\n", warn("!"))
		fmt.Println()
		fmt.Println("  Required token permissions:")
		fmt.Println("    Account / Cloudflare Tunnel / Edit")
		fmt.Println("    Zone / DNS / Edit")
		fmt.Println("    Account / Access: Apps and Policies / Edit")
		return nil
	}
	fmt.Printf("%s\n", green("✓"))

	fmt.Printf("  → Verifying domain %s... ", domain)
	if err := cf.VerifyZone(domain); err != nil {
		fmt.Println()
		fmt.Printf("  %s Domain check failed: %v\n", warn("!"), err)
		fmt.Printf("  %s Config saved, but make sure the domain is added to Cloudflare\n", warn("!"))
		return nil
	}
	fmt.Printf("%s\n", green("✓"))

	fmt.Println()
	fmt.Printf("  %s Config saved to %s\n", green("✓"), config.ConfigFilePath())
	fmt.Println()
	fmt.Println("  Next: zt up <service_name> <port>")
	fmt.Println("  Example: zt up portainer --docker --allow you@example.com")
	return nil
}

// readSecret reads the API token without echoing it to the terminal, when
// stdin is a real TTY. When stdin is redirected (piping a token in from a
// script, or CI), it falls back to a normal buffered line read — there's no
// terminal to suppress echo on, and term.ReadPassword would fail outright
// since it requires a valid terminal file descriptor.
func readSecret(reader *bufio.Reader) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return "", err
		}
		return strings.TrimSpace(line), nil
	}

	bytes, err := term.ReadPassword(fd)
	fmt.Println() // ReadPassword doesn't echo the Enter keypress
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes)), nil
}
