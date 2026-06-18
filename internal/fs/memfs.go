package fs

import (
	"fmt"
	"io/fs"
	"os"
	"path"
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

func normalize(p string) string {
	return filepath.ToSlash(p)
}

func (m *MemFS) ReadFile(p string) ([]byte, error) {
	p = normalize(p)
	data, ok := m.files[p]
	if !ok {
		return nil, fmt.Errorf("open %s: no such file or directory", p)
	}
	return data, nil
}

func (m *MemFS) WriteFile(p string, data []byte, perm os.FileMode) error {
	p = normalize(p)
	m.files[p] = append([]byte(nil), data...)
	m.ModTimes[p] = time.Now()
	dir := slashDir(p)
	for dir != "." && dir != "/" {
		m.dirs[dir] = true
		parent := slashDir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil
}

func slashDir(p string) string {
	idx := strings.LastIndexByte(p, '/')
	if idx < 0 {
		return "."
	}
	if idx == 0 {
		return "/"
	}
	return p[:idx]
}

func (m *MemFS) ReadDir(p string) ([]os.DirEntry, error) {
	p = normalize(p)
	var entries []os.DirEntry
	seen := make(map[string]bool)

	prefix := strings.TrimSuffix(p, "/") + "/"

	for fp := range m.files {
		if !strings.HasPrefix(fp, prefix) {
			continue
		}
		rel := strings.TrimPrefix(fp, prefix)
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
		if slashDir(d) == p {
			name := d[strings.LastIndexByte(d, '/')+1:]
			if !seen[name] {
				seen[name] = true
				entries = append(entries, &memDirEntry{name: name, isDir: true})
			}
		}
	}

	if len(entries) == 0 && !m.dirs[p] {
		hasAnyFile := false
		for fp := range m.files {
			if strings.HasPrefix(fp, prefix) {
				hasAnyFile = true
				break
			}
		}
		if !hasAnyFile {
			return nil, fmt.Errorf("open %s: no such file or directory", p)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	return entries, nil
}

func (m *MemFS) Glob(pattern string) ([]string, error) {
	pattern = normalize(pattern)
	var matches []string
	for fp := range m.files {
		matched, err := path.Match(pattern, fp)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, fp)
		}
	}
	sort.Strings(matches)
	return matches, nil
}

func (m *MemFS) Exists(p string) bool {
	p = normalize(p)
	if _, ok := m.files[p]; ok {
		return true
	}
	return m.dirs[p]
}

func (m *MemFS) MkdirAll(p string, perm os.FileMode) error {
	p = normalize(p)
	m.dirs[p] = true
	dir := slashDir(p)
	for dir != "." && dir != "/" {
		m.dirs[dir] = true
		parent := slashDir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil
}

func (m *MemFS) Stat(p string) (os.FileInfo, error) {
	p = normalize(p)
	if data, ok := m.files[p]; ok {
		mt := m.ModTimes[p]
		name := p[strings.LastIndexByte(p, '/')+1:]
		return &memFileInfo{name: name, size: int64(len(data)), modTime: mt}, nil
	}
	if m.dirs[p] {
		name := p[strings.LastIndexByte(p, '/')+1:]
		return &memFileInfo{name: name, isDir: true}, nil
	}
	return nil, fmt.Errorf("stat %s: no such file or directory", p)
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
