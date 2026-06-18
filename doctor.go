package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/dklinux7/devkit/internal/config"
	dkfs "github.com/dklinux7/devkit/internal/fs"
	"github.com/dklinux7/devkit/internal/registry"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check which generated project files are stale",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	fsys := dkfs.NewOsFS()

	ws, err := config.Load(fsys, dataDir)
	if err != nil {
		return err
	}

	// Gather source file mtimes.
	sourceMtime := time.Time{}

	identityFiles, err := fsys.Glob(filepath.Join(dataDir, "identity", "*.md"))
	if err != nil {
		return fmt.Errorf("reading identity/: %w", err)
	}
	for _, f := range identityFiles {
		info, err := fsys.Stat(f)
		if err != nil {
			continue
		}
		if info.ModTime().After(sourceMtime) {
			sourceMtime = info.ModTime()
		}
	}

	// Context mtime.
	ctxPath := filepath.Join(dataDir, "contexts", ws.ActiveContext+".md")
	if fsys.Exists(ctxPath) {
		info, err := fsys.Stat(ctxPath)
		if err == nil && info.ModTime().After(sourceMtime) {
			sourceMtime = info.ModTime()
		}
	} else {
		ctxDir := filepath.Join(dataDir, "contexts", ws.ActiveContext)
		if fsys.Exists(ctxDir) {
			ctxFiles, err := fsys.Glob(filepath.Join(ctxDir, "*.md"))
			if err == nil {
				for _, f := range ctxFiles {
					info, err := fsys.Stat(f)
					if err == nil && info.ModTime().After(sourceMtime) {
						sourceMtime = info.ModTime()
					}
				}
			}
		}
	}

	// donts.md mtime.
	dontsPath := filepath.Join(dataDir, "donts.md")
	if fsys.Exists(dontsPath) {
		info, err := fsys.Stat(dontsPath)
		if err == nil && info.ModTime().After(sourceMtime) {
			sourceMtime = info.ModTime()
		}
	}

	paths, err := registry.ReadAll(fsys, dataDir)
	if err != nil {
		return fmt.Errorf("reading projects registry: %w", err)
	}

	if len(paths) == 0 {
		return fmt.Errorf("no projects tracked — run devkit generate <path> first")
	}

	var allGood = true

	for _, p := range paths {
		if !fsys.Exists(p) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "? missing       %s\n", p)
			allGood = false
			continue
		}

		claudeMD := filepath.Join(p, "CLAUDE.md")
		if !fsys.Exists(claudeMD) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "⚠ not generated  %s\n", p)
			allGood = false
			continue
		}

		info, err := fsys.Stat(claudeMD)
		if err != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "⚠ unreadable     %s\n", p)
			allGood = false
			continue
		}

		if sourceMtime.After(info.ModTime()) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✗ stale          %s\n", p)
			allGood = false
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ up-to-date     %s\n", p)
		}
	}

	if allGood {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nAll projects up to date.")
	}

	return nil
}
