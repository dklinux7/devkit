package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

//go:embed all:templates
var TemplateFS embed.FS

var rootCmd = &cobra.Command{
	Use:   "devkit",
	Short: "Personal dev workspace generator",
	Long: `devkit — personal dev workspace generator

Composes your identity, constraints, and company context into AI config
files for any coding tool (Claude Code, Cursor, Copilot, Windsurf, OpenCode).

One source of truth → every AI tool gets the same context.`,
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
