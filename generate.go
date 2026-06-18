package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dklinux7/devkit/internal/composer"
	"github.com/dklinux7/devkit/internal/config"
	"github.com/dklinux7/devkit/internal/devctx"
	dkfs "github.com/dklinux7/devkit/internal/fs"
	"github.com/dklinux7/devkit/internal/generator"
	"github.com/spf13/cobra"
)

var (
	dryRun         bool
	includeLessons bool
	force          bool
)

var generateCmd = &cobra.Command{
	Use:   "generate <path>",
	Short: "Compose identity + context → write AI config files to target",
	Args:  cobra.ExactArgs(1),
	RunE:  runGenerate,
}

func init() {
	generateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be generated without writing")
	generateCmd.Flags().BoolVar(&includeLessons, "include-lessons", false, "append lessons at end of output")
	generateCmd.Flags().BoolVar(&force, "force", false, "bypass 32KB size limit")
	rootCmd.AddCommand(generateCmd)
}

func runGenerate(cmd *cobra.Command, args []string) error {
	targetDir, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolving target path: %w", err)
	}

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return fmt.Errorf("target directory does not exist: %s", targetDir)
	}

	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	fsys := dkfs.NewOsFS()

	ws, err := config.Load(fsys, dataDir)
	if err != nil {
		return err
	}

	sources, err := devctx.Load(fsys, dataDir, ws.ActiveContext, includeLessons)
	if err != nil {
		return fmt.Errorf("loading context: %w", err)
	}

	result, err := composer.Compose(sources, force)
	if err != nil {
		return err
	}

	for _, w := range result.Warnings {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "⚠ %s\n", w)
	}

	if dryRun {
		printDryRun(cmd, targetDir, result, ws)
		return nil
	}

	templateDir := filepath.Join(dataDir, "templates")
	genResult, err := generator.Generate(fsys, targetDir, result.Content, ws, templateDir)
	if err != nil {
		return err
	}

	if len(genResult.Overwritten) > 0 {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Overwriting: %s\n", strings.Join(genResult.Overwritten, ", "))
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), `✓ Generated %d files in %s:
  %s

  Context: %s (from %s/contexts/%s.md)
  Size: %.1fKB

  ⚠ These files contain your private context. Add to .gitignore if repo is public.
`, len(genResult.Written), targetDir,
		strings.Join(genResult.Written, ", "),
		ws.ActiveContext, dataDir, ws.ActiveContext,
		float64(result.Size)/1024.0)

	return nil
}

func printDryRun(cmd *cobra.Command, targetDir string, result *composer.Result, ws *config.Workspace) {
	lines := strings.Split(result.Content, "\n")
	preview := lines
	if len(preview) > 20 {
		preview = preview[:20]
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would generate %d files in %s:\n\n", len(generator.MarkdownTargets), targetDir)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "--- CLAUDE.md (preview) ---\n")
	for _, line := range preview {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", line)
	}
	if len(lines) > 20 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "... (%d more lines)\n", len(lines)-20)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %.1fKB | Files: %s\n",
		float64(result.Size)/1024.0,
		strings.Join(generator.MarkdownTargets, ", "))
}
