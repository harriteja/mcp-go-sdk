package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func buildServer() error {
	// Get current directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Check if this is an MCP project
	if _, err := os.Stat(filepath.Join(wd, "go.mod")); err != nil {
		return fmt.Errorf("go.mod not found, are you in an MCP project directory?")
	}

	// Create build directory if it doesn't exist
	buildDir := filepath.Join(wd, "build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	// Build the server
	cmd := exec.Command("go", "build", "-o", filepath.Join(buildDir, "server"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build server: %w", err)
	}

	fmt.Println("Server built successfully!")
	fmt.Printf("Binary location: %s\n", filepath.Join(buildDir, "server"))
	return nil
}
