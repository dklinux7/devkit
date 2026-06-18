package search

import (
	"testing"

	"github.com/dklinux7/devkit/internal/fs"
)

func TestSearchNative_Basic(t *testing.T) {
	m := fs.NewMemFS()
	m.MkdirAll("/dk", 0755)
	m.WriteFile("/dk/notes.md", []byte("line one\nretry logic here\nline three"), 0644)
	m.WriteFile("/dk/other.md", []byte("no match here"), 0644)

	matches, err := searchNative(m, "/dk", "retry")
	if err != nil {
		t.Fatalf("searchNative: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("got %d matches, want 1", len(matches))
	}
	if matches[0].Line != 2 {
		t.Fatalf("line = %d, want 2", matches[0].Line)
	}
	if matches[0].File != "/dk/notes.md" {
		t.Fatalf("file = %q", matches[0].File)
	}
}

func TestSearchNative_CaseInsensitive(t *testing.T) {
	m := fs.NewMemFS()
	m.MkdirAll("/dk", 0755)
	m.WriteFile("/dk/test.md", []byte("Kubernetes cluster\nKUBERNETES pods"), 0644)

	matches, err := searchNative(m, "/dk", "kubernetes")
	if err != nil {
		t.Fatalf("searchNative: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("got %d matches, want 2", len(matches))
	}
}

func TestSearchNative_Recursive(t *testing.T) {
	m := fs.NewMemFS()
	m.MkdirAll("/dk/sub", 0755)
	m.WriteFile("/dk/top.md", []byte("found here"), 0644)
	m.WriteFile("/dk/sub/nested.md", []byte("also found here"), 0644)

	matches, err := searchNative(m, "/dk", "found")
	if err != nil {
		t.Fatalf("searchNative: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("got %d matches, want 2", len(matches))
	}
}

func TestSearchNative_NoMatch(t *testing.T) {
	m := fs.NewMemFS()
	m.MkdirAll("/dk", 0755)
	m.WriteFile("/dk/test.md", []byte("nothing relevant"), 0644)

	matches, err := searchNative(m, "/dk", "xyzzy")
	if err != nil {
		t.Fatalf("searchNative: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("got %d matches, want 0", len(matches))
	}
}
