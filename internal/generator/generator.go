package generator

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/dklinux7/devkit/internal/config"
	"github.com/dklinux7/devkit/internal/fs"
)

var MarkdownTargets = []string{
	"CLAUDE.md",
	"AGENTS.md",
	"GEMINI.md",
	".cursorrules",
	".windsurfrules",
	".github/copilot-instructions.md",
}

var MDCTargets = []string{
	".cursor/rules/devkit-context.mdc",
}

const mdcFrontmatter = "---\ndescription: devkit identity and context\nalwaysApply: true\n---\n\n"

var StructuredTargets = []string{
	"opencode.json",
	".claude/settings.json",
}

type Result struct {
	Written     []string
	Overwritten []string
}

type TemplateData struct {
	Workspace *config.Workspace
	Content   string
}

func Generate(fsys fs.FS, targetDir string, content string, ws *config.Workspace, templateDir string) (*Result, error) {
	r := &Result{}

	allMarkdownTargets := append(MarkdownTargets, ws.ExtraTargets...)

	for _, name := range allMarkdownTargets {
		path := filepath.Join(targetDir, name)
		if fsys.Exists(path) {
			existing, err := fsys.ReadFile(path)
			if err == nil && string(existing) != content {
				r.Overwritten = append(r.Overwritten, name)
			}
		}
		if err := ensureParentDir(fsys, path); err != nil {
			return nil, err
		}
		if err := fsys.WriteFile(path, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", name, err)
		}
		r.Written = append(r.Written, name)
	}

	mdcContent := mdcFrontmatter + content
	for _, name := range MDCTargets {
		path := filepath.Join(targetDir, name)
		if fsys.Exists(path) {
			existing, err := fsys.ReadFile(path)
			if err == nil && string(existing) != mdcContent {
				r.Overwritten = append(r.Overwritten, name)
			}
		}
		if err := ensureParentDir(fsys, path); err != nil {
			return nil, err
		}
		if err := fsys.WriteFile(path, []byte(mdcContent), 0644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", name, err)
		}
		r.Written = append(r.Written, name)
	}

	for _, name := range StructuredTargets {
		tmplPath := filepath.Join(templateDir, name+".tmpl")
		if !fsys.Exists(tmplPath) {
			continue
		}
		tmplData, err := fsys.ReadFile(tmplPath)
		if err != nil {
			return nil, fmt.Errorf("reading template %s: %w", name, err)
		}
		tmpl, err := template.New(name).Parse(string(tmplData))
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", name, err)
		}

		var buf strings.Builder
		data := TemplateData{Workspace: ws, Content: content}
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("executing template %s: %w", name, err)
		}

		path := filepath.Join(targetDir, name)
		rendered := buf.String()
		if fsys.Exists(path) {
			existing, err := fsys.ReadFile(path)
			if err == nil && string(existing) != rendered {
				r.Overwritten = append(r.Overwritten, name)
			}
		}
		if err := ensureParentDir(fsys, path); err != nil {
			return nil, err
		}
		if err := fsys.WriteFile(path, []byte(rendered), 0644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", name, err)
		}
		r.Written = append(r.Written, name)
	}

	return r, nil
}

func ensureParentDir(fsys fs.FS, path string) error {
	dir := filepath.Dir(path)
	if !fsys.Exists(dir) {
		return fsys.MkdirAll(dir, 0755)
	}
	return nil
}
