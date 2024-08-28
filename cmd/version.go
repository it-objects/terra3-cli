/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// var needs to be used instead of const as ldflags is used to fill this
// information in the release process
var (
	version = "unknown"
	commit  = "unknown"
	date    = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints out Terra3 CLI version.",
	Long:  `Prints out Terra3 CLI version.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Terra3 CLI " + version + " Date: " + date + " Commit: " + commit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
