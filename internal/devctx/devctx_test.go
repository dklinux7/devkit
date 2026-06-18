package devctx

import (
	"testing"

	"github.com/dklinux7/devkit/internal/fs"
)

func TestStripFrontmatter(t *testing.T) {
	input := []byte("---\ntitle: test\ndate: 2024-01-01\n---\n\n# Hello\nBody text")
	got := string(StripFrontmatter(input))
	want := "# Hello\nBody text"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestStripFrontmatter_NoFrontmatter(t *testing.T) {
	input := []byte("# Just content\nNo frontmatter here")
	got := string(StripFrontmatter(input))
	if got != string(input) {
		t.Fatalf("should not modify content without frontmatter")
	}
}

func TestLoad_Basic(t *testing.T) {
	m := fs.NewMemFS()
	m.WriteFile("/dk/identity/ai.md", []byte("---\ntitle: ai\n---\nBe concise."), 0644)
	m.WriteFile("/dk/identity/engineering.md", []byte("Use Go."), 0644)
	m.WriteFile("/dk/contexts/work.md", []byte("---\ncompany: acme\n---\nAcme Corp context."), 0644)
	m.WriteFile("/dk/donts.md", []byte("Never commit secrets."), 0644)

	src, err := Load(m, "/dk", "work", false)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(src.Identity) != 2 {
		t.Fatalf("got %d identity files, want 2", len(src.Identity))
	}
	if string(src.Identity[0]) != "Be concise." {
		t.Fatalf("identity[0] = %q, want frontmatter stripped", string(src.Identity[0]))
	}
	if string(src.Context) != "Acme Corp context." {
		t.Fatalf("context = %q", string(src.Context))
	}
	if string(src.Donts) != "Never commit secrets." {
		t.Fatalf("donts = %q", string(src.Donts))
	}
	if src.Lessons != nil {
		t.Fatal("lessons should be nil without --include-lessons")
	}
}

func TestLoad_WithLessons(t *testing.T) {
	m := fs.NewMemFS()
	m.WriteFile("/dk/identity/ai.md", []byte("Be concise."), 0644)
	m.WriteFile("/dk/contexts/work.md", []byte("Context."), 0644)
	m.WriteFile("/dk/donts.md", []byte("Donts."), 0644)
	m.WriteFile("/dk/lessons/acme-2024.md", []byte("---\ncompany: acme\n---\nLesson 1."), 0644)

	src, err := Load(m, "/dk", "work", true)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(src.Lessons) != 1 {
		t.Fatalf("got %d lessons, want 1", len(src.Lessons))
	}
	if string(src.Lessons[0]) != "Lesson 1." {
		t.Fatalf("lesson = %q", string(src.Lessons[0]))
	}
}

func TestLoad_ContextFolder(t *testing.T) {
	m := fs.NewMemFS()
	m.WriteFile("/dk/identity/ai.md", []byte("AI rules."), 0644)
	m.WriteFile("/dk/contexts/bigco/main.md", []byte("Main context."), 0644)
	m.WriteFile("/dk/contexts/bigco/services.md", []byte("Services list."), 0644)
	m.WriteFile("/dk/donts.md", []byte("Donts."), 0644)
	m.MkdirAll("/dk/contexts/bigco", 0755)

	src, err := Load(m, "/dk", "bigco", false)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(src.Context) == 0 {
		t.Fatal("expected non-empty context from folder")
	}
}

func TestLoad_MissingContext(t *testing.T) {
	m := fs.NewMemFS()
	m.WriteFile("/dk/identity/ai.md", []byte("AI rules."), 0644)
	m.WriteFile("/dk/donts.md", []byte("Donts."), 0644)

	src, err := Load(m, "/dk", "nonexistent", false)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(src.Context) != 0 {
		t.Fatalf("expected empty context, got %q", string(src.Context))
	}
}
