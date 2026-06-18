package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/dklinux7/devkit/internal/config"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "git pull + push on ~/.devkit/ (multi-machine sync)",
	RunE:  runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	gitDir := dataDir + "/.git"
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("%s is not a git repository\n\nTo set up sync:\n  cd %s\n  git init && git remote add origin <your-private-repo-url>\n  git add -A && git commit -m \"initial\"\n  git push -u origin main", dataDir, dataDir)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Syncing %s...\n", dataDir)

	pullCmd := exec.Command("git", "-C", dataDir, "pull", "--rebase")
	pullCmd.Stdout = cmd.OutOrStdout()
	pullCmd.Stderr = cmd.ErrOrStderr()
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	pushCmd := exec.Command("git", "-C", dataDir, "push")
	pushCmd.Stdout = cmd.OutOrStdout()
	pushCmd.Stderr = cmd.ErrOrStderr()
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ ~/.devkit/ synced\n")
	return nil
}
