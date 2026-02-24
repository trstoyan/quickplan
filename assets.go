package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed scripts/*
var embeddedScripts embed.FS

// ExtractScripts extracts the embedded scripts to a temporary directory
// Returns the path to the directory containing the scripts
func ExtractScripts() (string, error) {
	// Create a temp directory for our binaries
	tmpDir := filepath.Join(os.TempDir(), "quickplan-bins")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Walk through the embedded scripts and extract them
	err := fs.WalkDir(embeddedScripts, "scripts", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		content, err := embeddedScripts.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		destPath := filepath.Join(tmpDir, filepath.Base(path))
		if err := os.WriteFile(destPath, content, 0755); err != nil {
			return fmt.Errorf("failed to write script %s: %w", destPath, err)
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return tmpDir, nil
}
