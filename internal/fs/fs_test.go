package fs

import (
	"testing"
)

func TestMemFS_ReadWriteFile(t *testing.T) {
	m := NewMemFS()

	_, err := m.ReadFile("/tmp/nofile")
	if err == nil {
		t.Fatal("expected error reading nonexistent file")
	}

	err = m.WriteFile("/tmp/hello.txt", []byte("hello"), 0644)
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	data, err := m.ReadFile("/tmp/hello.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("got %q, want %q", data, "hello")
	}
}

func TestMemFS_Exists(t *testing.T) {
	m := NewMemFS()

	if m.Exists("/nope") {
		t.Fatal("expected false for nonexistent path")
	}

	m.WriteFile("/tmp/file", []byte("x"), 0644)
	if !m.Exists("/tmp/file") {
		t.Fatal("expected true for existing file")
	}
}

func TestMemFS_MkdirAll(t *testing.T) {
	m := NewMemFS()

	err := m.MkdirAll("/a/b/c", 0755)
	if err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if !m.Exists("/a/b/c") {
		t.Fatal("expected dir to exist")
	}
	if !m.Exists("/a/b") {
		t.Fatal("expected parent dir to exist")
	}
}

func TestMemFS_ReadDir(t *testing.T) {
	m := NewMemFS()
	m.WriteFile("/dir/a.md", []byte("a"), 0644)
	m.WriteFile("/dir/b.md", []byte("b"), 0644)
	m.WriteFile("/dir/sub/c.md", []byte("c"), 0644)

	entries, err := m.ReadDir("/dir")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
	if entries[0].Name() != "a.md" || entries[0].IsDir() {
		t.Fatalf("entry 0: got %s (dir=%v)", entries[0].Name(), entries[0].IsDir())
	}
	if entries[2].Name() != "sub" || !entries[2].IsDir() {
		t.Fatalf("entry 2: got %s (dir=%v)", entries[2].Name(), entries[2].IsDir())
	}
}

func TestMemFS_Glob(t *testing.T) {
	m := NewMemFS()
	m.WriteFile("/dir/a.md", []byte("a"), 0644)
	m.WriteFile("/dir/b.txt", []byte("b"), 0644)
	m.WriteFile("/dir/c.md", []byte("c"), 0644)

	matches, err := m.Glob("/dir/*.md")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("got %d matches, want 2", len(matches))
	}
}

func TestMemFS_Stat(t *testing.T) {
	m := NewMemFS()
	m.WriteFile("/file.txt", []byte("content"), 0644)
	m.MkdirAll("/mydir", 0755)

	info, err := m.Stat("/file.txt")
	if err != nil {
		t.Fatalf("Stat file: %v", err)
	}
	if info.IsDir() {
		t.Fatal("expected file, got dir")
	}
	if info.Size() != 7 {
		t.Fatalf("got size %d, want 7", info.Size())
	}

	info, err = m.Stat("/mydir")
	if err != nil {
		t.Fatalf("Stat dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected dir")
	}
}

func TestOsFS_Implements_Interface(t *testing.T) {
	var _ FS = NewOsFS()
}

func TestMemFS_Implements_Interface(t *testing.T) {
	var _ FS = NewMemFS()
}
