package common

import (
	"os"
	"path/filepath"
)

// GetExecDir return exec dir
func GetExecDir() string {
	dir, _ := os.Executable()
	exPath := filepath.Dir(dir)
	return exPath
}
