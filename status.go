package main

import (
	"fmt"
	"path/filepath"

	"github.com/dklinux7/devkit/internal/registry"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync state for all tracked project paths",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx, err := resolveComposed(false, true)
	if err != nil {
		return err
	}

	paths, err := registry.ReadAll(ctx.fsys, ctx.dataDir)
	if err != nil {
		return fmt.Errorf("reading projects registry: %w", err)
	}

	if len(paths) == 0 {
		return fmt.Errorf("no projects tracked — run devkit generate <path> first")
	}

	var inSync, stale, missing int

	for _, p := range paths {
		if !ctx.fsys.Exists(p) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "? missing       %s\n", p)
			missing++
			continue
		}

		claudeMD := filepath.Join(p, "CLAUDE.md")
		if !ctx.fsys.Exists(claudeMD) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "⚠ not generated  %s\n", p)
			stale++
			continue
		}

		existing, err := ctx.fsys.ReadFile(claudeMD)
		if err != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "⚠ not generated  %s\n", p)
			stale++
			continue
		}

		if string(existing) == ctx.result.Content {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ in-sync    %s\n", p)
			inSync++
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✗ stale      %s\n", p)
			stale++
		}
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n%d in-sync, %d stale, %d missing\n", inSync, stale, missing)
	return nil
}
