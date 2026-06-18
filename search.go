package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dklinux7/devkit/internal/config"
	dkfs "github.com/dklinux7/devkit/internal/fs"
	"github.com/dklinux7/devkit/internal/search"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
)

var searchInteractive bool

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search across all ~/.devkit/ markdown",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearch,
}

func init() {
	searchCmd.Flags().BoolVar(&searchInteractive, "interactive", false, "fuzzy search results interactively (requires fzf or go-fuzzyfinder)")
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
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No matches for %q in %s\n", query, dataDir)
		return nil
	}

	if searchInteractive {
		return runSearchInteractive(cmd, matches, dataDir)
	}

	for _, m := range matches {
		rel, _ := filepath.Rel(dataDir, m.File)
		if rel == "" {
			rel = m.File
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s:%d: %s\n", rel, m.Line, m.Text)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n%d matches\n", len(matches))

	return nil
}

func runSearchInteractive(cmd *cobra.Command, matches []search.Match, dataDir string) error {
	// Try fzf first.
	if _, err := exec.LookPath("fzf"); err == nil {
		var lines []string
		for _, m := range matches {
			rel, _ := filepath.Rel(dataDir, m.File)
			if rel == "" {
				rel = m.File
			}
			lines = append(lines, fmt.Sprintf("%s:%d: %s", rel, m.Line, m.Text))
		}
		allLines := strings.Join(lines, "\n")

		fzfCmd := exec.Command("fzf", "--no-sort")
		fzfCmd.Stdin = strings.NewReader(allLines)
		fzfCmd.Stderr = os.Stderr
		out, err := fzfCmd.Output()
		if err != nil {
			return nil // user cancelled
		}
		_, _ = fmt.Fprint(cmd.OutOrStdout(), string(out))
		return nil
	}

	// Fallback: go-fuzzyfinder.
	idx, err := fuzzyfinder.Find(matches, func(i int) string {
		rel, _ := filepath.Rel(dataDir, matches[i].File)
		return fmt.Sprintf("%s:%d: %s", rel, matches[i].Line, matches[i].Text)
	})
	if err != nil {
		return nil // user cancelled
	}
	m := matches[idx]
	rel, _ := filepath.Rel(dataDir, m.File)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s:%d: %s\n", rel, m.Line, m.Text)
	return nil
}
