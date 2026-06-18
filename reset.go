package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/dklinux7/devkit/internal/config"
	dkfs "github.com/dklinux7/devkit/internal/fs"
	"github.com/spf13/cobra"
)

var resetHard bool

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Re-scaffold ~/.devkit/ (non-destructive by default; use --hard to delete everything)",
	RunE:  runReset,
}

func init() {
	resetCmd.Flags().BoolVar(&resetHard, "hard", false, "delete all of ~/.devkit/ and re-initialize (destructive)")
	rootCmd.AddCommand(resetCmd)
}

func runReset(cmd *cobra.Command, args []string) error {
	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	if resetHard {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "WARNING: This will permanently delete %s/ and all its contents.\n", dataDir)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "This cannot be undone. All identity, context, and findings files will be lost.\n\n")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Type 'yes' to confirm: ")

		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		answer := strings.TrimSpace(scanner.Text())

		if answer != "yes" {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
			return nil
		}

		if err := os.RemoveAll(dataDir); err != nil {
			return fmt.Errorf("deleting %s: %w", dataDir, err)
		}

		return runInit(cmd, args)
	}

	// Non-destructive: scaffold only files that don't already exist.
	fsys := dkfs.NewOsFS()

	// Determine which files would be scaffolded.
	var missing []string
	walkErr := fs.WalkDir(TemplateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel("templates", path)
		target := filepath.Join(dataDir, rel)
		if !fsys.Exists(target) {
			missing = append(missing, rel)
		}
		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("scanning templates: %w", walkErr)
	}

	if len(missing) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Already up to date.")
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Will scaffold %d missing files in %s/:\n", len(missing), dataDir)
	for _, f := range missing {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", f)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nType 'yes' to confirm: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	answer := strings.TrimSpace(scanner.Text())

	if answer != "yes" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
		return nil
	}

	if err := runInitMissing(cmd, dataDir, fsys); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Scaffolded %d missing files in %s/\n", len(missing), dataDir)
	return nil
}

// runInitMissing copies scaffold files only where the destination doesn't exist.
func runInitMissing(cmd *cobra.Command, dataDir string, fsys dkfs.FS) error {
	return fs.WalkDir(TemplateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel("templates", path)
		target := filepath.Join(dataDir, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		// Skip if file already exists.
		if fsys.Exists(target) {
			return nil
		}

		data, err := TemplateFS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}
