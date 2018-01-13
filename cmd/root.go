package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "awsup",
	Short: "awsup is a transpiler for authoring AWS CloudFormation templates",
	Long:  longAppDescription,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var longAppDescription = strings.TrimSpace(`
awsup is a transpiler that generates AWS CloudFormation templates based on
a convenient, readable source language.
`)
