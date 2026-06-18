package fs

import (
	"os"
	"path/filepath"
)

type OsFS struct{}

func NewOsFS() *OsFS {
	return &OsFS{}
}

func (f *OsFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (f *OsFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".devkit-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

func (f *OsFS) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (f *OsFS) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

func (f *OsFS) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (f *OsFS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (f *OsFS) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}
