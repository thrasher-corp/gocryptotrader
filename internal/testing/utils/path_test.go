package path

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRootPathFromCWD exercises RootPathFromCWD and RootPath
func TestRootPathFromCWD(t *testing.T) {
	r, err := RootPathFromCWD()
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(r, "LICENSE"))
	require.NoError(t, err, "Must find a LICENSE file")

	// Ensure there are no other license files in sub-directories
	err = filepath.WalkDir(r, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip the root of the project
		w, _ := filepath.Split(p)
		w = filepath.Clean(w)
		if w == r {
			switch d.Name() {
			case "vendor", "web":
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && d.Name() == "LICENSE" {
			return fmt.Errorf("found an unexpected LICENSE file in a sub-directory: %s", p)
		}
		return nil
	})
	assert.NoError(t, err)
}
