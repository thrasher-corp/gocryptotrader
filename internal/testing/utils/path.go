package path

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Exported public errors
var (
	ErrRootNotFound = errors.New("could not find root of gocryptotrader")
)

// RootPathFromCWD returns the system path to GoCryptoTrader from the current working directory
// Expects to find LICENSE file
func RootPathFromCWD() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return RootPath(wd)
}

// RootPath returns the system path to GoCryptoTrader from a sub-directory path
func RootPath(p string) (string, error) {
	parts := strings.Split(p, string(filepath.Separator))
	for i := len(parts); i > 0; i-- {
		dir := strings.Join(parts[:i], string(filepath.Separator))
		if _, err := os.Stat(filepath.Join(dir, "LICENSE")); err == nil {
			return dir, nil
		}
	}
	return "", ErrRootNotFound
}
