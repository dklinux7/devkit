package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/dklinux7/devkit/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up ~/.devkit/ with starter templates",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Join(dataDir, "workspace.yaml")); err == nil {
		return fmt.Errorf("%s already exists — remove it first to re-initialize", dataDir)
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("creating %s: %w", dataDir, err)
	}

	err = fs.WalkDir(TemplateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel("templates", path)
		target := filepath.Join(dataDir, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := TemplateFS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		return fmt.Errorf("scaffolding templates: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), `✓ Created %s/

Next steps:
  1. Edit your identity:
     %s/identity/ai.md          ← how AI should behave with you
     %s/identity/engineering.md  ← your coding style, git workflow, preferences

  2. Set your constraints:
     %s/donts.md                ← things AI must never do

  3. Create your first context:
     %s/contexts/work.md        ← describe your company, repos, tools, team

  4. Generate AI config for a project:
     devkit generate ~/path/to/project

Run 'devkit help' for all commands.
`, dataDir, dataDir, dataDir, dataDir, dataDir)

	return nil
}
