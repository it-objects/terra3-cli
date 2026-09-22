package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Credentials struct {
	AccessToken   string `json:"access_token"`
	IDToken       string `json:"id_token"`
	RefreshToken  string `json:"refresh_token"`
	ExpiresAt     string `json:"expires_at"`
	TokenEndpoint string `json:"token_endpoint"`
	ClientID      string `json:"client_id"`
}

func CredentialsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".terra3", "credentials.json")
}

func LoadCredentials() (*Credentials, error) {
	data, err := os.ReadFile(CredentialsPath())
	if err != nil {
		return nil, fmt.Errorf("not logged in — run 'terra3 platform login' first")
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("corrupted credentials file: %w", err)
	}
	return &creds, nil
}

func SaveCredentials(creds *Credentials) error {
	path := CredentialsPath()
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func ClearCredentials() error {
	return os.Remove(CredentialsPath())
}

func (c *Credentials) IsExpired() bool {
	t, err := time.Parse(time.RFC3339, c.ExpiresAt)
	if err != nil {
		return true
	}
	return time.Now().After(t.Add(-30 * time.Second))
}

func (c *Credentials) Refresh() error {
	if c.RefreshToken == "" || c.TokenEndpoint == "" || c.ClientID == "" {
		return fmt.Errorf("cannot refresh — missing refresh token or config")
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {c.ClientID},
		"refresh_token": {c.RefreshToken},
	}

	resp, err := http.PostForm(c.TokenEndpoint, data)
	if err != nil {
		return fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("refresh failed (HTTP %d) — run 'terra3 platform login' again", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to parse refresh response: %w", err)
	}

	c.AccessToken = tokenResp.AccessToken
	if tokenResp.IDToken != "" {
		c.IDToken = tokenResp.IDToken
	}
	c.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Format(time.RFC3339)

	return SaveCredentials(c)
}

func GetValidToken() (string, error) {
	creds, err := LoadCredentials()
	if err != nil {
		return "", err
	}

	if creds.IsExpired() {
		if err := creds.Refresh(); err != nil {
			return "", err
		}
	}

	return creds.AccessToken, nil
}

func ExchangeCode(tokenEndpoint, clientID, code, redirectURI, verifier string) (*Credentials, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	}

	resp, err := http.PostForm(tokenEndpoint, data)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		return nil, fmt.Errorf("token exchange failed (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(body[:n])))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	creds := &Credentials{
		AccessToken:   tokenResp.AccessToken,
		IDToken:       tokenResp.IDToken,
		RefreshToken:  tokenResp.RefreshToken,
		ExpiresAt:     time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Format(time.RFC3339),
		TokenEndpoint: tokenEndpoint,
		ClientID:      clientID,
	}

	if err := SaveCredentials(creds); err != nil {
		return nil, fmt.Errorf("failed to save credentials: %w", err)
	}

	return creds, nil
}
