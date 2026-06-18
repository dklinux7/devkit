package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

//go:embed all:templates
var TemplateFS embed.FS

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "devkit",
	Short: "Personal dev workspace generator",
	Long: `devkit — personal dev workspace generator

Composes your identity, constraints, and company context into AI config
files for any coding tool (Claude Code, Cursor, Copilot, Windsurf, OpenCode).

One source of truth → every AI tool gets the same context.`,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "print debug information")
}

func main() {
	os.Exit(run())
}

func run() int {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func debugf(format string, args ...any) {
	if verbose {
		_, _ = fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", args...)
	}
}
