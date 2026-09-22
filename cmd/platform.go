package cmd

import (
	"github.com/spf13/cobra"
)

var platformCmd = &cobra.Command{
	Use:   "platform",
	Short: "Interact with the Terra3 control plane (Cognito auth, no AWS profile required)",
	Long: `Commands that authenticate against the Terra3 platform API via Entra ID / Cognito.

Unlike 'terra3 db port-forward' (which needs an AWS profile with Identity Center
access to the workload account), these commands rely only on Terra3 RBAC.`,
}

func init() {
	rootCmd.AddCommand(platformCmd)
}
