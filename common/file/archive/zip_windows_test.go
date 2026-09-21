package archive

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLstatSource(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(file, []byte("original"), 0o600), "WriteFile must not error")
	info, err := lstatSource(file)
	require.NoError(t, err, "lstatSource must not error")
	assert.True(t, info.Mode().IsRegular(), "lstatSource should identify a regular file")
	require.NoError(t, os.Rename(file, filepath.Join(dir, "original.txt")), "Rename must not error")
	require.NoError(t, os.WriteFile(file, []byte("replacement"), 0o600), "WriteFile must not error")
	replacement, err := lstatSource(file)
	require.NoError(t, err, "lstatSource must not error")
	assert.False(t, os.SameFile(info, replacement), "lstatSource should retain identity after a pathname is replaced")

	info, err = lstatSource(dir + string(filepath.Separator))
	require.NoError(t, err, "lstatSource must accept a trailing separator")
	assert.True(t, info.IsDir(), "lstatSource should identify a directory")
	_, err = lstatSource(filepath.Join(dir, "absent"))
	assert.ErrorIs(t, err, fs.ErrNotExist, "lstatSource should report a missing source")
	_, err = lstatSource("")
	assert.ErrorIs(t, err, fs.ErrNotExist, "lstatSource should reject an empty source")

	t.Run("symlinks", func(t *testing.T) {
		t.Parallel()
		skipWithoutSymlinks(t)
		link := filepath.Join(t.TempDir(), "link")
		require.NoError(t, os.Symlink(dir, link), "Symlink must not error")
		info, err := lstatSource(link)
		require.NoError(t, err, "lstatSource must inspect a link")
		assert.NotZero(t, info.Mode()&os.ModeSymlink, "lstatSource should identify the link itself")
		info, err = lstatSource(link + string(filepath.Separator))
		require.NoError(t, err, "lstatSource must follow a link with a trailing separator")
		assert.True(t, info.IsDir(), "a trailing separator should name the linked directory")
	})
}
