package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type TunnelSecret struct {
	TunnelSecret string `json:"tunnel_secret"`
}

// metadataManagedByKey/Value tag every tunnel cfzt creates with
// {"managed_by": "cfzt"} in the Cloudflare Tunnel metadata field. This is
// how cfzt tells its own tunnels apart from ones created some other way
// (Cloudflare dashboard, another tool) when it finds a name collision —
// see FindTunnelByNameManaged.
const (
	metadataManagedByKey   = "managed_by"
	metadataManagedByValue = "cfzt"
)

type TunnelResponse struct {
	Result struct {
		ID       string         `json:"id"`
		Name     string         `json:"name"`
		Status   string         `json:"status"`
		Metadata map[string]any `json:"metadata,omitempty"`
	} `json:"result"`
	Success bool     `json:"success"`
	Errors  []APIErr `json:"errors"`
}

type TunnelListResponse struct {
	Result []struct {
		ID       string         `json:"id"`
		Name     string         `json:"name"`
		Status   string         `json:"status"`
		Metadata map[string]any `json:"metadata,omitempty"`
	} `json:"result"`
	Success bool     `json:"success"`
	Errors  []APIErr `json:"errors"`
}

type TunnelConfigPayload struct {
	Config TunnelIngressConfig `json:"config"`
}

type TunnelIngressConfig struct {
	Ingress []IngressRule `json:"ingress"`
}

type IngressRule struct {
	Hostname string `json:"hostname,omitempty"`
	Service  string `json:"service"`
}

// CreateTunnel creates a named tunnel and returns its ID and credentials JSON.
// The tunnel is tagged with metadata {"managed_by": "cfzt"} so a later name
// collision can be told apart from a tunnel cfzt didn't create — see
// FindTunnelByNameManaged.
func (c *Client) CreateTunnel(name string) (tunnelID string, credJSON []byte, err error) {
	secret := generateSecret()
	body, _ := json.Marshal(map[string]any{
		"name":          name,
		"tunnel_secret": secret,
		"metadata": map[string]string{
			metadataManagedByKey: metadataManagedByValue,
		},
	})

	resp, err := c.do("POST",
		fmt.Sprintf("/accounts/%s/cfd_tunnel", c.AccountID),
		bytes.NewReader(body))
	if err != nil {
		return "", nil, err
	}
	defer closeBody(resp)

	var tr TunnelResponse
	if err := decode(resp, &tr); err != nil {
		return "", nil, err
	}
	if !tr.Success {
		return "", nil, apiErr(tr.Errors)
	}

	creds := map[string]string{
		"AccountTag":   c.AccountID,
		"TunnelSecret": secret,
		"TunnelID":     tr.Result.ID,
	}
	credJSON, _ = json.Marshal(creds)
	return tr.Result.ID, credJSON, nil
}

// ConfigureTunnel sets ingress rules for a tunnel.
func (c *Client) ConfigureTunnel(tunnelID, hostname, localPort string) error {
	payload := TunnelConfigPayload{
		Config: TunnelIngressConfig{
			Ingress: []IngressRule{
				{Hostname: hostname, Service: "http://localhost:" + localPort},
				{Service: "http_status:404"},
			},
		},
	}
	body, _ := json.Marshal(payload)

	resp, err := c.do("PUT",
		fmt.Sprintf("/accounts/%s/cfd_tunnel/%s/configurations", c.AccountID, tunnelID),
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer closeBody(resp)

	var result struct {
		Success bool     `json:"success"`
		Errors  []APIErr `json:"errors"`
	}
	if err := decode(resp, &result); err != nil {
		return err
	}
	if !result.Success {
		return apiErr(result.Errors)
	}
	return nil
}

// DeleteTunnel deletes a tunnel by ID.
func (c *Client) DeleteTunnel(tunnelID string) error {
	resp, err := c.do("DELETE",
		fmt.Sprintf("/accounts/%s/cfd_tunnel/%s", c.AccountID, tunnelID),
		nil)
	if err != nil {
		return err
	}
	defer closeBody(resp)

	var result struct {
		Success bool     `json:"success"`
		Errors  []APIErr `json:"errors"`
	}
	if err := decode(resp, &result); err != nil {
		return err
	}
	if !result.Success {
		return apiErr(result.Errors)
	}
	return nil
}

// ListTunnels returns all tunnels for the account.
func (c *Client) ListTunnels() ([]struct{ ID, Name, Status string }, error) {
	resp, err := c.do("GET",
		fmt.Sprintf("/accounts/%s/cfd_tunnel?status=active", c.AccountID),
		nil)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	var tr TunnelListResponse
	if err := decode(resp, &tr); err != nil {
		return nil, err
	}
	if !tr.Success {
		return nil, apiErr(tr.Errors)
	}

	out := make([]struct{ ID, Name, Status string }, len(tr.Result))
	for i, r := range tr.Result {
		out[i] = struct{ ID, Name, Status string }{r.ID, r.Name, r.Status}
	}
	return out, nil
}

func decode(resp *http.Response, v any) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// FindTunnelByName finds an existing tunnel by name and returns its ID.
// Returns empty string (no error) if not found.
func (c *Client) FindTunnelByName(name string) (string, error) {
	id, _, err := c.FindTunnelByNameManaged(name)
	return id, err
}

// FindTunnelByNameManaged finds an existing tunnel by name and reports
// whether it carries cfzt's own "managed_by": "cfzt" metadata tag — the
// marker CreateTunnel sets on every tunnel it creates. A name collision
// with a tunnel that lacks the tag means it was created some other way
// (Cloudflare dashboard, another tool, or a previous cfzt version from
// before this tag existed), and callers should treat deleting it as a
// distinct, more dangerous operation than cleaning up cfzt's own stale
// tunnel. Returns empty id (no error, managedByCfzt false) if not found.
func (c *Client) FindTunnelByNameManaged(name string) (id string, managedByCfzt bool, err error) {
	resp, err := c.do("GET",
		fmt.Sprintf("/accounts/%s/cfd_tunnel?name=%s", c.AccountID, name),
		nil)
	if err != nil {
		return "", false, err
	}
	defer closeBody(resp)

	var tr TunnelListResponse
	if err := decode(resp, &tr); err != nil {
		return "", false, err
	}
	if !tr.Success {
		return "", false, apiErr(tr.Errors)
	}
	for _, t := range tr.Result {
		if t.Name == name {
			tag, _ := t.Metadata[metadataManagedByKey].(string)
			return t.ID, tag == metadataManagedByValue, nil
		}
	}
	return "", false, nil
}

// GetTunnelStatus returns the current status of a tunnel from Cloudflare.
// Possible values: active, degraded, inactive, down.
func (c *Client) GetTunnelStatus(tunnelID string) (string, error) {
	resp, err := c.do("GET",
		fmt.Sprintf("/accounts/%s/cfd_tunnel/%s", c.AccountID, tunnelID),
		nil)
	if err != nil {
		return "", err
	}
	defer closeBody(resp)

	var result struct {
		Result struct {
			Status string `json:"status"`
		} `json:"result"`
		Success bool     `json:"success"`
		Errors  []APIErr `json:"errors"`
	}
	if err := decode(resp, &result); err != nil {
		return "", err
	}
	if !result.Success {
		return "", apiErr(result.Errors)
	}
	return result.Result.Status, nil
}
