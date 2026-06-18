package registry

import (
	"testing"

	"github.com/dklinux7/devkit/internal/fs"
)

func TestAppend_FirstEntry(t *testing.T) {
	m := fs.NewMemFS()
	if err := m.MkdirAll("/home/.devkit", 0755); err != nil {
		t.Fatal(err)
	}

	if err := Append(m, "/home/.devkit", "/home/projects/myapp"); err != nil {
		t.Fatalf("Append: %v", err)
	}

	paths, err := ReadAll(m, "/home/.devkit")
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("len(paths) = %d, want 1", len(paths))
	}
	if paths[0] != "/home/projects/myapp" {
		t.Fatalf("paths[0] = %q, want %q", paths[0], "/home/projects/myapp")
	}
}

func TestAppend_Deduplication(t *testing.T) {
	m := fs.NewMemFS()
	if err := m.MkdirAll("/home/.devkit", 0755); err != nil {
		t.Fatal(err)
	}

	if err := Append(m, "/home/.devkit", "/home/projects/myapp"); err != nil {
		t.Fatalf("first Append: %v", err)
	}
	if err := Append(m, "/home/.devkit", "/home/projects/myapp"); err != nil {
		t.Fatalf("second Append: %v", err)
	}

	paths, err := ReadAll(m, "/home/.devkit")
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("len(paths) = %d, want 1 (no duplicates)", len(paths))
	}
}

func TestReadAll_Empty(t *testing.T) {
	m := fs.NewMemFS()

	paths, err := ReadAll(m, "/home/.devkit")
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(paths) != 0 {
		t.Fatalf("len(paths) = %d, want 0 for missing file", len(paths))
	}
}

func TestReadAll_MultipleEntries(t *testing.T) {
	m := fs.NewMemFS()
	if err := m.MkdirAll("/home/.devkit", 0755); err != nil {
		t.Fatal(err)
	}

	entries := []string{"/home/projects/app1", "/home/projects/app2", "/home/work/api"}
	for _, e := range entries {
		if err := Append(m, "/home/.devkit", e); err != nil {
			t.Fatalf("Append(%s): %v", e, err)
		}
	}

	paths, err := ReadAll(m, "/home/.devkit")
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(paths) != len(entries) {
		t.Fatalf("len(paths) = %d, want %d", len(paths), len(entries))
	}
	for i, want := range entries {
		if paths[i] != want {
			t.Fatalf("paths[%d] = %q, want %q", i, paths[i], want)
		}
	}
}
