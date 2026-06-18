package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dklinux7/devkit/internal/fs"
	"gopkg.in/yaml.v3"
)

type Workspace struct {
	Name          string `yaml:"name"`
	ActiveContext string `yaml:"active_context"`
}

func DataDir() (string, error) {
	if env := os.Getenv("DEVKIT_HOME"); env != "" {
		return env, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".devkit"), nil
}

func Load(fsys fs.FS, dataDir string) (*Workspace, error) {
	path := filepath.Join(dataDir, "workspace.yaml")
	data, err := fsys.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading workspace.yaml: %w", err)
	}

	var ws Workspace
	if err := yaml.Unmarshal(data, &ws); err != nil {
		return nil, fmt.Errorf("parsing workspace.yaml: %w", err)
	}

	if ws.Name == "" {
		return nil, fmt.Errorf("workspace.yaml: name is required")
	}
	if ws.ActiveContext == "" {
		return nil, fmt.Errorf("workspace.yaml: active_context is required")
	}

	return &ws, nil
}
