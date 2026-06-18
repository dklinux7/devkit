package main

import (
	"bufio"
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dklinux7/devkit/internal/config"
	dkfs "github.com/dklinux7/devkit/internal/fs"
	"github.com/spf13/cobra"
)

var untrackCmd = &cobra.Command{
	Use:   "untrack <path>",
	Short: "Remove a project path from the tracking registry",
	Args:  cobra.ExactArgs(1),
	RunE:  runUntrack,
}

func init() {
	rootCmd.AddCommand(untrackCmd)
}

func runUntrack(cmd *cobra.Command, args []string) error {
	targetDir, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	fsys := dkfs.NewOsFS()
	filePath := filepath.Join(dataDir, "projects.txt")

	data, err := fsys.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("no projects tracked")
	}

	var lines []string
	found := false
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == targetDir {
			found = true
			continue
		}
		if line != "" {
			lines = append(lines, line)
		}
	}

	if !found {
		return fmt.Errorf("%s is not tracked", targetDir)
	}

	var buf []byte
	if len(lines) > 0 {
		buf = []byte(strings.Join(lines, "\n") + "\n")
	}

	if err := fsys.WriteFile(filePath, buf, 0600); err != nil {
		return fmt.Errorf("updating registry: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Removed %s from tracking registry\n", targetDir)
	return nil
}
