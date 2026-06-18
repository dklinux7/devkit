package search

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dklinux7/devkit/internal/fs"
)

type Match struct {
	File string
	Line int
	Text string
}

func Search(fsys fs.FS, dataDir string, query string) ([]Match, error) {
	if rgPath, err := exec.LookPath("rg"); err == nil {
		return searchRipgrep(rgPath, dataDir, query)
	}
	return searchNative(fsys, dataDir, query)
}

func searchRipgrep(rgPath string, dataDir string, query string) ([]Match, error) {
	cmd := exec.Command(rgPath, "--line-number", "--no-heading", "--glob", "*.md", "--", query, dataDir)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("ripgrep: %w", err)
	}

	var matches []Match
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		lineNum := 0
		_, _ = fmt.Sscanf(parts[1], "%d", &lineNum)
		matches = append(matches, Match{
			File: parts[0],
			Line: lineNum,
			Text: parts[2],
		})
	}
	return matches, nil
}

func searchNative(fsys fs.FS, dataDir string, query string) ([]Match, error) {
	re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(query))
	if err != nil {
		return nil, err
	}

	var matches []Match
	var walkErr error

	mdFiles, err := findMarkdownFiles(fsys, dataDir)
	if err != nil {
		return nil, err
	}

	for _, path := range mdFiles {
		data, err := fsys.ReadFile(path)
		if err != nil {
			walkErr = err
			continue
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if re.MatchString(line) {
				matches = append(matches, Match{
					File: path,
					Line: i + 1,
					Text: line,
				})
			}
		}
	}

	if walkErr != nil && len(matches) == 0 {
		return nil, walkErr
	}
	return matches, nil
}

func findMarkdownFiles(fsys fs.FS, dir string) ([]string, error) {
	var files []string

	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		if e.IsDir() {
			sub, err := findMarkdownFiles(fsys, path)
			if err != nil {
				continue
			}
			files = append(files, sub...)
		} else if strings.HasSuffix(e.Name(), ".md") {
			files = append(files, path)
		}
	}

	return files, nil
}
