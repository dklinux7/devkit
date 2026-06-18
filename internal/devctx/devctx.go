package devctx

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dklinux7/devkit/internal/fs"
)

var frontmatterRe = regexp.MustCompile(`(?s)\A---\n.*?\n---\n*`)

func StripFrontmatter(content []byte) []byte {
	return frontmatterRe.ReplaceAll(content, nil)
}

type Sources struct {
	Identity   [][]byte
	Context    []byte
	RawContext []byte
	Donts      []byte
	Lessons    [][]byte
}

func Load(fsys fs.FS, dataDir string, activeContext string, includeLessons bool) (*Sources, error) {
	if strings.Contains(activeContext, "..") || filepath.IsAbs(activeContext) {
		return nil, fmt.Errorf("invalid active_context %q: must be a simple name", activeContext)
	}
	contextsDir := filepath.Join(dataDir, "contexts")
	ctxPath := filepath.Join(contextsDir, activeContext+".md")
	if !strings.HasPrefix(filepath.Clean(ctxPath), filepath.Clean(contextsDir)+string(os.PathSeparator)) {
		return nil, fmt.Errorf("active_context %q escapes contexts directory", activeContext)
	}

	s := &Sources{}

	identityFiles, err := fsys.Glob(filepath.Join(dataDir, "identity", "*.md"))
	if err != nil {
		return nil, err
	}
	for _, f := range identityFiles {
		data, err := fsys.ReadFile(f)
		if err != nil {
			return nil, err
		}
		s.Identity = append(s.Identity, StripFrontmatter(data))
	}

	if fsys.Exists(ctxPath) {
		data, err := fsys.ReadFile(ctxPath)
		if err != nil {
			return nil, err
		}
		s.RawContext = data
		s.Context = StripFrontmatter(data)
	} else {
		ctxDir := filepath.Join(dataDir, "contexts", activeContext)
		if fsys.Exists(ctxDir) {
			files, err := fsys.Glob(filepath.Join(ctxDir, "*.md"))
			if err != nil {
				return nil, err
			}
			var parts [][]byte
			for _, f := range files {
				data, err := fsys.ReadFile(f)
				if err != nil {
					return nil, err
				}
				parts = append(parts, StripFrontmatter(data))
			}
			s.Context = []byte(joinSections(parts))
		}
	}

	dontsPath := filepath.Join(dataDir, "donts.md")
	if fsys.Exists(dontsPath) {
		data, err := fsys.ReadFile(dontsPath)
		if err != nil {
			return nil, err
		}
		s.Donts = StripFrontmatter(data)
	}

	if includeLessons {
		lessonFiles, err := fsys.Glob(filepath.Join(dataDir, "lessons", "*.md"))
		if err != nil {
			return nil, err
		}
		for _, f := range lessonFiles {
			data, err := fsys.ReadFile(f)
			if err != nil {
				return nil, err
			}
			s.Lessons = append(s.Lessons, StripFrontmatter(data))
		}
	}

	return s, nil
}

func joinSections(parts [][]byte) string {
	strs := make([]string, len(parts))
	for i, p := range parts {
		strs[i] = strings.TrimSpace(string(p))
	}
	return strings.Join(strs, "\n\n")
}
