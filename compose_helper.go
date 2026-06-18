package main

import (
	"fmt"

	"github.com/dklinux7/devkit/internal/composer"
	"github.com/dklinux7/devkit/internal/config"
	"github.com/dklinux7/devkit/internal/devctx"
	dkfs "github.com/dklinux7/devkit/internal/fs"
)

type composedContext struct {
	fsys    dkfs.FS
	dataDir string
	ws      *config.Workspace
	result  *composer.Result
}

func resolveComposed(includeLessons bool, force bool) (*composedContext, error) {
	dataDir, err := config.DataDir()
	if err != nil {
		return nil, err
	}
	debugf("data dir: %s", dataDir)
	fsys := dkfs.NewOsFS()
	ws, err := config.Load(fsys, dataDir)
	if err != nil {
		return nil, err
	}
	debugf("active context: %s", ws.ActiveContext)
	sources, err := devctx.Load(fsys, dataDir, ws.ActiveContext, includeLessons)
	if err != nil {
		return nil, fmt.Errorf("loading context: %w", err)
	}
	result, err := composer.Compose(sources, force)
	if err != nil {
		return nil, err
	}
	debugf("composed size: %d bytes", result.Size)
	return &composedContext{fsys: fsys, dataDir: dataDir, ws: ws, result: result}, nil
}
