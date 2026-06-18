package registry

import (
	"bufio"
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	dkfs "github.com/dklinux7/devkit/internal/fs"
)

// Append adds targetPath to projects.txt if not already present.
func Append(fsys dkfs.FS, dataDir, targetPath string) error {
	filePath := filepath.Join(dataDir, "projects.txt")

	existing, err := fsys.ReadFile(filePath)
	if err != nil {
		// File doesn't exist yet — create it with just this path.
		return fsys.WriteFile(filePath, []byte(targetPath+"\n"), 0644)
	}

	scanner := bufio.NewScanner(bytes.NewReader(existing))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == targetPath {
			return nil // already present
		}
	}

	updated := append(existing, []byte(targetPath+"\n")...)
	return fsys.WriteFile(filePath, updated, 0644)
}

// ReadAll returns all paths from projects.txt. Missing file returns empty slice, not error.
func ReadAll(fsys dkfs.FS, dataDir string) ([]string, error) {
	filePath := filepath.Join(dataDir, "projects.txt")

	data, err := fsys.ReadFile(filePath)
	if err != nil {
		if !fsys.Exists(filePath) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("reading projects.txt: %w", err)
	}

	var paths []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			paths = append(paths, line)
		}
	}
	return paths, nil
}
