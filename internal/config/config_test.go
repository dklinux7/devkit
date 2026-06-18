package config

import (
	"testing"

	"github.com/dklinux7/devkit/internal/fs"
)

func TestLoad_Valid(t *testing.T) {
	m := fs.NewMemFS()
	if err := m.WriteFile("/home/.devkit/workspace.yaml", []byte("name: John\nactive_context: work\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ws, err := Load(m, "/home/.devkit")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ws.Name != "John" {
		t.Fatalf("Name = %q, want %q", ws.Name, "John")
	}
	if ws.ActiveContext != "work" {
		t.Fatalf("ActiveContext = %q, want %q", ws.ActiveContext, "work")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	m := fs.NewMemFS()
	_, err := Load(m, "/home/.devkit")
	if err == nil {
		t.Fatal("expected error for missing workspace.yaml")
	}
}

func TestLoad_MissingName(t *testing.T) {
	m := fs.NewMemFS()
	if err := m.WriteFile("/home/.devkit/workspace.yaml", []byte("active_context: work\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(m, "/home/.devkit")
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestLoad_MissingActiveContext(t *testing.T) {
	m := fs.NewMemFS()
	if err := m.WriteFile("/home/.devkit/workspace.yaml", []byte("name: John\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(m, "/home/.devkit")
	if err == nil {
		t.Fatal("expected error for missing active_context")
	}
}

func TestLoad_ExtraTargets(t *testing.T) {
	m := fs.NewMemFS()
	yaml := "name: John\nactive_context: work\nextra_targets:\n  - custom/tool.md\n  - .roo/system-prompt.md\n"
	if err := m.WriteFile("/home/.devkit/workspace.yaml", []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	ws, err := Load(m, "/home/.devkit")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(ws.ExtraTargets) != 2 {
		t.Fatalf("ExtraTargets len = %d, want 2", len(ws.ExtraTargets))
	}
	if ws.ExtraTargets[0] != "custom/tool.md" {
		t.Fatalf("ExtraTargets[0] = %q, want %q", ws.ExtraTargets[0], "custom/tool.md")
	}
	if ws.ExtraTargets[1] != ".roo/system-prompt.md" {
		t.Fatalf("ExtraTargets[1] = %q, want %q", ws.ExtraTargets[1], ".roo/system-prompt.md")
	}
}

func TestDataDir_Default(t *testing.T) {
	t.Setenv("DEVKIT_HOME", "")
	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}
	if dir == "" {
		t.Fatal("DataDir returned empty string")
	}
}

func TestDataDir_EnvOverride(t *testing.T) {
	t.Setenv("DEVKIT_HOME", "/custom/path")
	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}
	if dir != "/custom/path" {
		t.Fatalf("DataDir = %q, want %q", dir, "/custom/path")
	}
}
