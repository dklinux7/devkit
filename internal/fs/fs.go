package fs

import "os"

type FS interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	ReadDir(path string) ([]os.DirEntry, error)
	Glob(pattern string) ([]string, error)
	Exists(path string) bool
	MkdirAll(path string, perm os.FileMode) error
	Stat(path string) (os.FileInfo, error)
}
