/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints out Terra3 CLI version.",
	Long:  `Prints out Terra3 CLI version.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Terra3 CLI v0.0.7")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
