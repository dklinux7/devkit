package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dklinux7/devkit/internal/composer"
	"github.com/dklinux7/devkit/internal/config"
	"github.com/dklinux7/devkit/internal/devctx"
	dkfs "github.com/dklinux7/devkit/internal/fs"
	"github.com/dklinux7/devkit/internal/generator"
	"github.com/dklinux7/devkit/internal/registry"
	"github.com/spf13/cobra"
)

var (
	dryRun         bool
	includeLessons bool
	force          bool
	generateAll    bool
	quiet          bool
)

var generateCmd = &cobra.Command{
	Use:   "generate <path>",
	Short: "Compose identity + context → write AI config files to target",
	Args:  cobra.ArbitraryArgs,
	RunE:  runGenerate,
}

func init() {
	generateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be generated without writing")
	generateCmd.Flags().BoolVar(&includeLessons, "include-lessons", false, "append lessons at end of output")
	generateCmd.Flags().BoolVar(&force, "force", false, "bypass 32KB size limit")
	generateCmd.Flags().BoolVar(&generateAll, "all", false, "regenerate all tracked project paths")
	generateCmd.Flags().BoolVar(&quiet, "quiet", false, "suppress output on success")
	rootCmd.AddCommand(generateCmd)
}

func runGenerate(cmd *cobra.Command, args []string) error {
	if !generateAll && len(args) != 1 {
		return fmt.Errorf("requires exactly 1 argument (target path), or use --all")
	}
	if generateAll && len(args) > 0 {
		return fmt.Errorf("--all does not take a path argument")
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
		targetDir := args[0]
		abs, err := filepath.Abs(targetDir)
		if err != nil {
			return fmt.Errorf("resolving target path: %w", err)
		}
		printDryRun(cmd, abs, result, ws)
		return nil
	}

	if generateAll {
		paths, err := registry.ReadAll(fsys, dataDir)
		if err != nil {
			return fmt.Errorf("reading projects registry: %w", err)
		}
		if len(paths) == 0 {
			return fmt.Errorf("no projects tracked — run devkit generate <path> first")
		}

		templateDir := filepath.Join(dataDir, "templates")
		for _, p := range paths {
			if _, err := os.Stat(p); os.IsNotExist(err) {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  skipping missing dir: %s\n", p)
				continue
			}
			if err := generateToPath(cmd, fsys, dataDir, ws, sources, result, p, templateDir); err != nil {
				return err
			}
		}
		return nil
	}

	targetDir, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolving target path: %w", err)
	}

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return fmt.Errorf("target directory does not exist: %s", targetDir)
	}

	templateDir := filepath.Join(dataDir, "templates")
	return generateToPath(cmd, fsys, dataDir, ws, sources, result, targetDir, templateDir)
}

func generateToPath(cmd *cobra.Command, fsys dkfs.FS, dataDir string, ws *config.Workspace, sources *devctx.Sources, result *composer.Result, targetDir, templateDir string) error {
	genResult, err := generator.Generate(fsys, targetDir, result.Content, ws, templateDir)
	if err != nil {
		return err
	}

	if mcpJSON := buildMCPJSON(sources); mcpJSON != "" {
		mcpPath := filepath.Join(targetDir, ".mcp.json")
		if fsys.Exists(mcpPath) {
			existing, _ := fsys.ReadFile(mcpPath)
			if string(existing) != mcpJSON {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Overwriting: .mcp.json\n")
			}
		}
		if err := fsys.WriteFile(mcpPath, []byte(mcpJSON), 0600); err != nil {
			return fmt.Errorf("writing .mcp.json: %w", err)
		}
	}

	if len(genResult.Overwritten) > 0 {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Overwriting: %s\n", strings.Join(genResult.Overwritten, ", "))
	}

	if !quiet {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), `✓ Generated %d files in %s:
  %s

  Context: %s (from %s/contexts/%s.md)
  Size: %.1fKB

  ⚠ These files contain your private context. Add to .gitignore if repo is public.
`, len(genResult.Written), targetDir,
			strings.Join(genResult.Written, ", "),
			ws.ActiveContext, dataDir, ws.ActiveContext,
			float64(result.Size)/1024.0)
	}

	// Register the path in projects.txt.
	if err := registry.Append(fsys, dataDir, targetDir); err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  warning: could not update projects registry: %v\n", err)
	}

	// Write to ~/.claude/skills/devkit-context.md if the directory exists.
	writeSkillsFile(cmd, result.Content)

	return nil
}

func writeSkillsFile(cmd *cobra.Command, content string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	skillsDir := filepath.Join(home, ".claude", "skills")
	if _, err := os.Stat(skillsDir); err != nil {
		return // directory doesn't exist, skip silently
	}
	skillsPath := filepath.Join(skillsDir, "devkit-context.md")
	if err := os.WriteFile(skillsPath, []byte(content), 0644); err != nil {
		return // skip silently on error
	}
	if !quiet {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  ↳ also wrote ~/.claude/skills/devkit-context.md\n")
	}
}

func buildMCPJSON(sources *devctx.Sources) string {
	servers := devctx.ParseMCPServers(sources.RawContext)
	if len(servers) == 0 {
		return ""
	}

	type serverEntry struct {
		Command string            `json:"command"`
		Args    []string          `json:"args,omitempty"`
		Env     map[string]string `json:"env,omitempty"`
	}
	type mcpConfig struct {
		MCPServers map[string]serverEntry `json:"mcpServers"`
	}

	cfg := mcpConfig{MCPServers: make(map[string]serverEntry)}
	for name, srv := range servers {
		cfg.MCPServers[name] = serverEntry{
			Command: srv.Command,
			Args:    srv.Args,
			Env:     srv.Env,
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return ""
	}
	return string(data) + "\n"
}

func printDryRun(cmd *cobra.Command, targetDir string, result *composer.Result, ws *config.Workspace) {
	lines := strings.Split(result.Content, "\n")
	preview := lines
	if len(preview) > 20 {
		preview = preview[:20]
	}

	allTargets := append(generator.MarkdownTargets, ws.ExtraTargets...)
	allTargets = append(allTargets, generator.MDCTargets...)

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would generate %d files in %s:\n\n", len(allTargets), targetDir)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "--- CLAUDE.md (preview) ---\n")
	for _, line := range preview {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", line)
	}
	if len(lines) > 20 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "... (%d more lines)\n", len(lines)-20)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %.1fKB | Files: %s\n",
		float64(result.Size)/1024.0,
		strings.Join(allTargets, ", "))
}
