package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dklinux7/devkit/internal/config"
	dkfs "github.com/dklinux7/devkit/internal/fs"
	"github.com/dklinux7/devkit/internal/search"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search across all ~/.devkit/ markdown",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	fsys := dkfs.NewOsFS()
	matches, err := search.Search(fsys, dataDir, query)
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No matches for %q in %s\n", query, dataDir)
		return nil
	}

	for _, m := range matches {
		rel, _ := filepath.Rel(dataDir, m.File)
		if rel == "" {
			rel = m.File
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s:%d: %s\n", rel, m.Line, m.Text)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n%d matches\n", len(matches))

	return nil
}
