package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/dklinux7/devkit/internal/registry"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check which generated project files are stale (mtime-based)",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	cc, err := resolveComposed(false, true)
	if err != nil {
		return err
	}

	sourceMtime := time.Time{}

	identityFiles, err := cc.fsys.Glob(filepath.Join(cc.dataDir, "identity", "*.md"))
	if err != nil {
		return fmt.Errorf("reading identity/: %w", err)
	}
	for _, f := range identityFiles {
		info, err := cc.fsys.Stat(f)
		if err != nil {
			continue
		}
		if info.ModTime().After(sourceMtime) {
			sourceMtime = info.ModTime()
		}
	}

	ctxPath := filepath.Join(cc.dataDir, "contexts", cc.ws.ActiveContext+".md")
	if cc.fsys.Exists(ctxPath) {
		if info, err := cc.fsys.Stat(ctxPath); err == nil && info.ModTime().After(sourceMtime) {
			sourceMtime = info.ModTime()
		}
	} else {
		ctxDir := filepath.Join(cc.dataDir, "contexts", cc.ws.ActiveContext)
		if cc.fsys.Exists(ctxDir) {
			ctxFiles, _ := cc.fsys.Glob(filepath.Join(ctxDir, "*.md"))
			for _, f := range ctxFiles {
				if info, err := cc.fsys.Stat(f); err == nil && info.ModTime().After(sourceMtime) {
					sourceMtime = info.ModTime()
				}
			}
		}
	}

	dontsPath := filepath.Join(cc.dataDir, "donts.md")
	if cc.fsys.Exists(dontsPath) {
		if info, err := cc.fsys.Stat(dontsPath); err == nil && info.ModTime().After(sourceMtime) {
			sourceMtime = info.ModTime()
		}
	}

	debugf("latest source mtime: %s", sourceMtime)

	paths, err := registry.ReadAll(cc.fsys, cc.dataDir)
	if err != nil {
		return fmt.Errorf("reading projects registry: %w", err)
	}
	if len(paths) == 0 {
		return fmt.Errorf("no projects tracked — run devkit generate <path> first")
	}

	allGood := true
	for _, p := range paths {
		if !cc.fsys.Exists(p) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "? missing       %s\n", p)
			allGood = false
			continue
		}
		claudeMD := filepath.Join(p, "CLAUDE.md")
		if !cc.fsys.Exists(claudeMD) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "⚠ not generated  %s\n", p)
			allGood = false
			continue
		}
		info, err := cc.fsys.Stat(claudeMD)
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
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "All projects up to date.")
	}
	return nil
}
