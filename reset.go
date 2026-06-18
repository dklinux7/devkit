package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/dklinux7/devkit/internal/config"
	"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Delete ~/.devkit/ and re-initialize with starter templates",
	RunE:  runReset,
}

func init() {
	rootCmd.AddCommand(resetCmd)
}

func runReset(cmd *cobra.Command, args []string) error {
	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

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
