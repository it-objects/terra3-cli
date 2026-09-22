package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/it-objects/terra3-cli/internal/auth"
	"github.com/it-objects/terra3-cli/internal/config"
	"github.com/spf13/cobra"
)

var platformLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Terra3 via Entra ID SSO (opens browser)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if cfg.CognitoDomain == "" || cfg.CognitoClientID == "" {
			return fmt.Errorf("cognito not configured — set cognito_domain and cognito_client_id in ~/.terra3/config.yaml")
		}

		verifier, err := auth.GenerateVerifier()
		if err != nil {
			return fmt.Errorf("failed to generate PKCE verifier: %w", err)
		}
		challenge := auth.GenerateChallenge(verifier)

		port, resultCh, shutdown := auth.StartCallbackServer()
		defer shutdown()

		redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)
		tokenEndpoint := fmt.Sprintf("%s/oauth2/token", cfg.CognitoDomain)

		params := url.Values{
			"response_type":         {"code"},
			"client_id":             {cfg.CognitoClientID},
			"redirect_uri":          {redirectURI},
			"scope":                 {"openid profile email"},
			"code_challenge":        {challenge},
			"code_challenge_method": {"S256"},
		}
		if cfg.IdentityProvider != "" {
			params.Set("identity_provider", cfg.IdentityProvider)
		}
		authorizeURL := fmt.Sprintf("%s/oauth2/authorize?%s", cfg.CognitoDomain, params.Encode())

		fmt.Println("Opening browser for login...")
		fmt.Printf("If the browser doesn't open, visit:\n  %s\n\n", authorizeURL)
		openPlatformBrowser(authorizeURL)
		fmt.Println("Waiting for authentication...")

		select {
		case result := <-resultCh:
			if result.Error != "" {
				return fmt.Errorf("login failed: %s", result.Error)
			}

			creds, err := auth.ExchangeCode(tokenEndpoint, cfg.CognitoClientID, result.Code, redirectURI, verifier)
			if err != nil {
				return fmt.Errorf("token exchange failed: %w", err)
			}

			name := parsePlatformIDTokenClaim(creds.IDToken, "email")
			if name == "" {
				name = parsePlatformIDTokenClaim(creds.IDToken, "preferred_username")
			}
			if name == "" {
				name = "authenticated user"
			}

			fmt.Printf("\nLogged in as: %s\n", name)
			fmt.Printf("Token expires: %s\n", creds.ExpiresAt)
			fmt.Printf("API: %s\n", cfg.ApiUrl)

		case <-time.After(2 * time.Minute):
			return fmt.Errorf("login timed out — no callback received within 2 minutes")
		}

		return nil
	},
}

func openPlatformBrowser(rawURL string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	}
	if cmd != nil {
		_ = cmd.Start()
	}
}

func parsePlatformIDTokenClaim(idToken, claim string) string {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	if v, ok := claims[claim].(string); ok {
		return v
	}
	return ""
}

func init() {
	platformCmd.AddCommand(platformLoginCmd)
}
