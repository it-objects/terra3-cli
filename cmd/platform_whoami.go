package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/it-objects/terra3-cli/internal/auth"
	"github.com/spf13/cobra"
)

var platformWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current Terra3 platform identity",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.LoadCredentials()
		if err != nil || creds.AccessToken == "" {
			fmt.Println("Not authenticated.")
			fmt.Println("  Run 'terra3 platform login' to sign in.")
			return nil
		}

		claims := parsePlatformJWTClaims(creds.IDToken)
		email, _ := claims["email"].(string)
		name, _ := claims["preferred_username"].(string)
		if name == "" {
			name, _ = claims["cognito:username"].(string)
		}
		sub, _ := claims["sub"].(string)
		groups := parsePlatformJWTGroups(claims)

		role := "developer"
		if platformContainsGroup(groups, "admin") {
			role = "admin"
		}

		fmt.Printf("Auth: Cognito (%s)\n", role)
		if email != "" {
			fmt.Printf("Email: %s\n", email)
		}
		if name != "" {
			fmt.Printf("User: %s\n", name)
		}
		if sub != "" {
			fmt.Printf("Sub: %s\n", sub)
		}
		if len(groups) > 0 {
			fmt.Printf("Groups: %s\n", strings.Join(groups, ", "))
		}
		if creds.IsExpired() {
			fmt.Println("Token: expired (will auto-refresh)")
		} else {
			fmt.Printf("Token: valid until %s\n", creds.ExpiresAt)
		}
		return nil
	},
}

func parsePlatformJWTClaims(token string) map[string]interface{} {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims map[string]interface{}
	_ = json.Unmarshal(payload, &claims)
	return claims
}

func parsePlatformJWTGroups(claims map[string]interface{}) []string {
	if claims == nil {
		return nil
	}
	raw, ok := claims["cognito:groups"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []interface{}:
		groups := make([]string, 0, len(v))
		for _, g := range v {
			if s, ok := g.(string); ok {
				groups = append(groups, s)
			}
		}
		return groups
	case string:
		trimmed := strings.Trim(v, "[]")
		if trimmed == "" {
			return nil
		}
		return strings.Fields(trimmed)
	}
	return nil
}

func platformContainsGroup(groups []string, target string) bool {
	for _, g := range groups {
		if g == target {
			return true
		}
	}
	return false
}

func init() {
	platformCmd.AddCommand(platformWhoamiCmd)
}
