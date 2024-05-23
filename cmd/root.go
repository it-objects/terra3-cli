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
	Short: "A tool that makes it easy to cope with Terra3 provisioned enviroments.",
	Long: `Welcome to Terra3 - An opinionated Terraform module for ramping-up 3-tier architectures in AWS in no time!

	This repository contains a collection of Terraform modules that aim to make it 
	easier and faster for customers to get started with a 3-tier-architecture in AWS. 
	It can be used to configure and provision a complete stack with
	
	* a static website served from S3 and AWS Cloudfront
	
	* a containerized backend/API running on AWS ECS
	
	* an AWS RDS MySQL/Postgres database
	
	* an AWS ElastiCache Redis
	
	It is the result of many projects we did for customers with similar requirements. 
	And rather than starting from scratch with every project, we've created reusable 
	Terraform modules. What started as an internal library, now evolved into a single 
	module we'd like to share and to give back to the community as open source.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
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
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
