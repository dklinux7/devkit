package main

import (
	"fmt"
	"path/filepath"

	"github.com/dklinux7/devkit/internal/config"
	dkfs "github.com/dklinux7/devkit/internal/fs"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage contexts",
}

var contextLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List contexts with size and last-modified date",
	RunE:  runContextLs,
}

func init() {
	contextCmd.AddCommand(contextLsCmd)
	rootCmd.AddCommand(contextCmd)
}

func runContextLs(cmd *cobra.Command, args []string) error {
	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	fsys := dkfs.NewOsFS()

	ws, err := config.Load(fsys, dataDir)
	if err != nil {
		return err
	}

	contextsDir := filepath.Join(dataDir, "contexts")
	entries, err := fsys.ReadDir(contextsDir)
	if err != nil {
		return fmt.Errorf("reading contexts/: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  contexts/\n")

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() {
			// Folder context: aggregate size of all .md files inside.
			dirPath := filepath.Join(contextsDir, name)
			subEntries, err := fsys.ReadDir(dirPath)
			if err != nil {
				continue
			}
			var totalSize int64
			var latestMod string
			for _, sub := range subEntries {
				if sub.IsDir() {
					continue
				}
				subPath := filepath.Join(dirPath, sub.Name())
				info, err := fsys.Stat(subPath)
				if err != nil {
					continue
				}
				totalSize += info.Size()
				modStr := info.ModTime().Format("2006-01-02")
				if modStr > latestMod {
					latestMod = modStr
				}
			}
			active := ""
			if name == ws.ActiveContext {
				active = "  *active*"
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    %-20s %s  %s%s\n",
				name+"/", formatContextSize(totalSize), latestMod, active)
			continue
		}

		// Flat file context.
		if filepath.Ext(name) != ".md" {
			continue
		}
		filePath := filepath.Join(contextsDir, name)
		info, err := fsys.Stat(filePath)
		if err != nil {
			continue
		}
		baseName := name[:len(name)-3] // strip .md
		active := ""
		if baseName == ws.ActiveContext {
			active = "  *active*"
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    %-20s %s  %s%s\n",
			name, formatContextSize(info.Size()), info.ModTime().Format("2006-01-02"), active)
	}

	return nil
}

func formatContextSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%4dB", n)
	}
	return fmt.Sprintf("%.1fKB", float64(n)/1024.0)
}
