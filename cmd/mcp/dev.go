package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

func runDevServer() error {
	// Get current directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Check if this is an MCP project
	if _, err := os.Stat(filepath.Join(wd, "go.mod")); err != nil {
		return fmt.Errorf("go.mod not found, are you in an MCP project directory?")
	}

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer watcher.Close()

	// Watch Go files
	if err := filepath.Walk(wd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			return watcher.Add(filepath.Dir(path))
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to set up file watching: %w", err)
	}

	// Start initial server
	cmd := startServer()
	defer func() {
		if err := cmd.Process.Kill(); err != nil {
			log.Printf("Error killing process: %v", err)
		}
	}()

	// Watch for changes
	fmt.Println("Development server started. Press Ctrl+C to exit.")
	debounce := time.NewTimer(0)
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				debounce.Reset(100 * time.Millisecond)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("Error watching files: %v", err)
		case <-debounce.C:
			// Kill old server
			if cmd != nil && cmd.Process != nil {
				if err := cmd.Process.Kill(); err != nil {
					log.Printf("Error killing process: %v", err)
				}
				if err := cmd.Wait(); err != nil && err.Error() != "signal: killed" {
					log.Printf("Error waiting for process: %v", err)
				}
			}
			// Start new server
			cmd = startServer()
		}
	}
}

func startServer() *exec.Cmd {
	cmd := exec.Command("go", "run", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start server: %v", err)
	}
	return cmd
}
