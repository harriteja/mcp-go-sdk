package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP CLI tool for Go SDK",
	Long: `MCP CLI tool provides utilities for developing MCP servers and clients using the Go SDK.
It supports project scaffolding, development server, and code generation.`,
}

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new MCP project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		if err := initProject(projectName); err != nil {
			log.Fatalf("Failed to initialize project: %v", err)
		}
	},
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run development server with hot reload",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDevServer(); err != nil {
			log.Fatalf("Failed to run dev server: %v", err)
		}
	},
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the MCP server for deployment",
	Run: func(cmd *cobra.Command, args []string) {
		if err := buildServer(); err != nil {
			log.Fatalf("Failed to build server: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(devCmd)
	rootCmd.AddCommand(buildCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
