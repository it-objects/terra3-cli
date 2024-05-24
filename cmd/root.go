/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "terra3",
	Short: "A tool that makes it easy to interact with Terra3 provisioned enviroments.",
	Long: `Welcome to Terra3 CLI - A tool that makes it easy to interact with Terra3 provisioned enviroments.
	
	Hold on. What is Terra3, anyway? Terra3 is an opinionated Terraform module for 
	ramping-up 3-tier architectures in AWS in no time! Goto https://terra3.io for further information.
	
	The Terra3 CLI provides features to manage Terra3 stacks from the command line. It can
	
	* create a secure port-forward to the private RDS database using SSM 
	* manage environment hibernation (start/stop/status)
	* manage AWS secrets related to the environment
	* comfortably shelling into a container (if ECS exec is activated for the cluster)
	* and much more to come! 
	`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(dbCmd)
	rootCmd.AddCommand(loginCmd)
}
