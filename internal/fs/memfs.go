package fs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type MemFS struct {
	files    map[string][]byte
	dirs     map[string]bool
	ModTimes map[string]time.Time
}

func NewMemFS() *MemFS {
	return &MemFS{
		files:    make(map[string][]byte),
		dirs:     make(map[string]bool),
		ModTimes: make(map[string]time.Time),
	}
}

func (m *MemFS) ReadFile(path string) ([]byte, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, fmt.Errorf("open %s: no such file or directory", path)
	}
	return data, nil
}

func (m *MemFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	m.files[path] = append([]byte(nil), data...)
	m.ModTimes[path] = time.Now()
	dir := filepath.Dir(path)
	for dir != "." && dir != "/" {
		m.dirs[dir] = true
		dir = filepath.Dir(dir)
	}
	return nil
}

func (m *MemFS) ReadDir(path string) ([]os.DirEntry, error) {
	var entries []os.DirEntry
	seen := make(map[string]bool)

	prefix := path + "/"

	for p := range m.files {
		if !strings.HasPrefix(p, prefix) {
			continue
		}
		rel := strings.TrimPrefix(p, prefix)
		parts := strings.SplitN(rel, "/", 2)
		name := parts[0]
		if seen[name] {
			continue
		}
		seen[name] = true
		if len(parts) > 1 {
			entries = append(entries, &memDirEntry{name: name, isDir: true})
		} else {
			entries = append(entries, &memDirEntry{name: name, isDir: false})
		}
	}

	for d := range m.dirs {
		if filepath.Dir(d) == path {
			name := filepath.Base(d)
			if !seen[name] {
				seen[name] = true
				entries = append(entries, &memDirEntry{name: name, isDir: true})
			}
		}
	}

	if len(entries) == 0 && !m.dirs[path] {
		hasAnyFile := false
		for p := range m.files {
			if strings.HasPrefix(p, prefix) {
				hasAnyFile = true
				break
			}
		}
		if !hasAnyFile {
			return nil, fmt.Errorf("open %s: no such file or directory", path)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	return entries, nil
}

func (m *MemFS) Glob(pattern string) ([]string, error) {
	var matches []string
	for p := range m.files {
		matched, err := filepath.Match(pattern, p)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, p)
		}
	}
	sort.Strings(matches)
	return matches, nil
}

func (m *MemFS) Exists(path string) bool {
	if _, ok := m.files[path]; ok {
		return true
	}
	return m.dirs[path]
}

func (m *MemFS) MkdirAll(path string, perm os.FileMode) error {
	m.dirs[path] = true
	dir := filepath.Dir(path)
	for dir != "." && dir != "/" {
		m.dirs[dir] = true
		dir = filepath.Dir(dir)
	}
	return nil
}

func (m *MemFS) Stat(path string) (os.FileInfo, error) {
	if data, ok := m.files[path]; ok {
		mt := m.ModTimes[path]
		return &memFileInfo{name: filepath.Base(path), size: int64(len(data)), modTime: mt}, nil
	}
	if m.dirs[path] {
		return &memFileInfo{name: filepath.Base(path), isDir: true}, nil
	}
	return nil, fmt.Errorf("stat %s: no such file or directory", path)
}

type memDirEntry struct {
	name  string
	isDir bool
}

func (e *memDirEntry) Name() string               { return e.name }
func (e *memDirEntry) IsDir() bool                 { return e.isDir }
func (e *memDirEntry) Type() fs.FileMode           { return 0 }
func (e *memDirEntry) Info() (fs.FileInfo, error)  { return &memFileInfo{name: e.name, isDir: e.isDir}, nil }

type memFileInfo struct {
	name    string
	size    int64
	isDir   bool
	modTime time.Time
}

func (i *memFileInfo) Name() string       { return i.name }
func (i *memFileInfo) Size() int64        { return i.size }
func (i *memFileInfo) Mode() fs.FileMode  { return 0644 }
func (i *memFileInfo) ModTime() time.Time { return i.modTime }
func (i *memFileInfo) IsDir() bool        { return i.isDir }
func (i *memFileInfo) Sys() any           { return nil }
