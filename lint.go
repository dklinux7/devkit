package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dklinux7/devkit/internal/composer"
	"github.com/dklinux7/devkit/internal/config"
	"github.com/dklinux7/devkit/internal/devctx"
	dkfs "github.com/dklinux7/devkit/internal/fs"
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Validate ~/.devkit/ source files",
	RunE:  runLint,
}

func init() {
	rootCmd.AddCommand(lintCmd)
}

const (
	lintWarnSize = 8 * 1024  // 8KB per file
	lintWarnComp = 16 * 1024 // 16KB composed
	lintFailComp = 32 * 1024 // 32KB composed
)

var unexpandedVarRe = regexp.MustCompile(`\$\{[A-Z_][A-Z0-9_]*\}`)

func runLint(cmd *cobra.Command, args []string) error {
	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	fsys := dkfs.NewOsFS()

	var warnings, errors int

	// 1. workspace.yaml — required fields.
	ws, err := config.Load(fsys, dataDir)
	if err != nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✗ workspace.yaml: %v\n", err)
		errors++
		// Can't continue without a valid workspace.
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n%d warning(s), %d error(s)\n", warnings, errors)
		return fmt.Errorf("lint failed: %d error(s)", errors)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "✓ workspace.yaml valid")

	// 2. Active context exists.
	ctxPath := filepath.Join(dataDir, "contexts", ws.ActiveContext+".md")
	ctxDir := filepath.Join(dataDir, "contexts", ws.ActiveContext)
	if !fsys.Exists(ctxPath) && !fsys.Exists(ctxDir) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✗ contexts/%s: missing (neither file nor directory)\n", ws.ActiveContext)
		errors++
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ contexts/%s exists\n", ws.ActiveContext)
	}

	// 3. identity/ has at least one .md file.
	identityFiles, err := fsys.Glob(filepath.Join(dataDir, "identity", "*.md"))
	if err != nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "⚠ identity/: cannot read directory\n")
		warnings++
	} else if len(identityFiles) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "⚠ identity/: no .md files found\n")
		warnings++
	}

	// 4. Check each .md file in identity/ and active context.
	filesToCheck := append([]string{}, identityFiles...)
	if fsys.Exists(ctxPath) {
		filesToCheck = append(filesToCheck, ctxPath)
	} else if fsys.Exists(ctxDir) {
		ctxFiles, err := fsys.Glob(filepath.Join(ctxDir, "*.md"))
		if err == nil {
			filesToCheck = append(filesToCheck, ctxFiles...)
		}
	}

	for _, f := range filesToCheck {
		info, err := fsys.Stat(f)
		if err != nil {
			continue
		}
		rel, _ := filepath.Rel(dataDir, f)

		if info.Size() > lintWarnSize {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "⚠ %s: %.1fKB (may be large for some AI tools)\n",
				rel, float64(info.Size())/1024.0)
			warnings++
		}

		data, err := fsys.ReadFile(f)
		if err != nil {
			continue
		}
		if unexpandedVarRe.Find(data) != nil {
			vars := unexpandedVarRe.FindAll(data, -1)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "⚠ %s: unexpanded template variable(s): %s\n",
				rel, joinUniq(vars))
			warnings++
		}

		if info.Size() <= lintWarnSize {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ %s valid\n", rel)
		}
	}

	// 5. Estimate composed size.
	sources, err := devctx.Load(fsys, dataDir, ws.ActiveContext, false)
	if err == nil {
		result, err := composer.Compose(sources, true)
		if err == nil {
			sizeKB := float64(result.Size) / 1024.0
			if result.Size > lintFailComp {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✗ Estimated composed size: %.1fKB (exceeds 32KB hard limit)\n", sizeKB)
				errors++
			} else if result.Size > lintWarnComp {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "⚠ Estimated composed size: %.1fKB (recommended limit: 16KB)\n", sizeKB)
				warnings++
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Estimated composed size: %.1fKB (within limits)\n", sizeKB)
			}
		}
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n%d warning(s), %d error(s)\n", warnings, errors)

	if errors > 0 {
		return fmt.Errorf("lint failed: %d error(s)", errors)
	}
	return nil
}

func joinUniq(bss [][]byte) string {
	seen := make(map[string]bool)
	var parts []string
	for _, b := range bss {
		s := string(b)
		if !seen[s] {
			seen[s] = true
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, ", ")
}
