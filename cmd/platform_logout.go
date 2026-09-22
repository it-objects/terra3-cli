package cmd

import (
	"fmt"

	"github.com/it-objects/terra3-cli/internal/auth"
	"github.com/spf13/cobra"
)

var platformLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored Terra3 platform credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := auth.ClearCredentials(); err != nil {
			fmt.Println("No credentials to clear.")
			return nil
		}
		fmt.Println("Logged out. Credentials cleared.")
		return nil
	},
}

func init() {
	platformCmd.AddCommand(platformLogoutCmd)
}
