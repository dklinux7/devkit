package main

import (
	"fmt"
	"path/filepath"

	"github.com/dklinux7/devkit/internal/generator"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff <path>",
	Short: "Show what devkit generate would change",
	Args:  cobra.ExactArgs(1),
	RunE:  runDiff,
}

var diffCheck bool

func init() {
	diffCmd.Flags().BoolVar(&diffCheck, "check", false, "exit 1 if any files would change (for CI)")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	targetDir, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolving target path: %w", err)
	}

	cc, err := resolveComposed(false, true)
	if err != nil {
		return err
	}

	// Build list of all targets to check.
	type targetCheck struct {
		name    string
		content string
	}

	var targets []targetCheck
	for _, name := range generator.MarkdownTargets {
		targets = append(targets, targetCheck{name: name, content: cc.result.Content})
	}
	for _, name := range cc.ws.ExtraTargets {
		targets = append(targets, targetCheck{name: name, content: cc.result.Content})
	}
	mdcContent := generator.MDCFrontmatter + cc.result.Content
	for _, name := range generator.MDCTargets {
		targets = append(targets, targetCheck{name: name, content: mdcContent})
	}

	templateDir := filepath.Join(cc.dataDir, "templates")
	for _, name := range generator.StructuredTargets {
		tmplPath := filepath.Join(templateDir, name+".tmpl")
		if cc.fsys.Exists(tmplPath) {
			targets = append(targets, targetCheck{name: name, content: ""}) // content checked separately
		}
	}

	var wouldChange int

	for _, tc := range targets {
		path := filepath.Join(targetDir, tc.name)

		if tc.content == "" {
			// Structured target — just note it has a template.
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "- %s (no template)\n", tc.name)
			continue
		}

		if !cc.fsys.Exists(path) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✗ %s (would create: %s)\n", tc.name, formatSize(len(tc.content)))
			wouldChange++
			continue
		}

		existing, err := cc.fsys.ReadFile(path)
		if err != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✗ %s (would change: unreadable → %s)\n", tc.name, formatSize(len(tc.content)))
			wouldChange++
			continue
		}

		if string(existing) == tc.content {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ %s (unchanged)\n", tc.name)
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✗ %s (would change: %s → %s)\n",
				tc.name, formatSize(len(existing)), formatSize(len(tc.content)))
			wouldChange++
		}
	}

	// Also flag structured targets without templates.
	for _, name := range generator.StructuredTargets {
		tmplPath := filepath.Join(templateDir, name+".tmpl")
		if !cc.fsys.Exists(tmplPath) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "- %s (no template)\n", name)
		}
	}

	if diffCheck && wouldChange > 0 {
		return fmt.Errorf("out of sync: %d files would change", wouldChange)
	}

	return nil
}

func formatSize(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	return fmt.Sprintf("%.1fKB", float64(n)/1024.0)
}
