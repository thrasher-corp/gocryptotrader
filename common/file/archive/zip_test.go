package archive

import (
	"archive/zip"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnZip(t *testing.T) {
	tempDir := t.TempDir()
	zipFile := filepath.Join("..", "..", "..", "testdata", "testdata.zip")
	files, err := UnZip(zipFile, tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files to be extracted received: %v ", len(files))
	}

	zipFile = filepath.Join("..", "..", "..", "testdata", "zip-slip.zip")
	_, err = UnZip(zipFile, tempDir)
	if err == nil {
		t.Fatal("Zip() expected to error due to ZipSlip detection but extracted successfully")
	}

	zipFile = filepath.Join("..", "..", "..", "testdata", "configtest.json")
	_, err = UnZip(zipFile, tempDir)
	if err == nil {
		t.Fatal("Zip() expected to error due to invalid zipfile")
	}
}

func TestZip(t *testing.T) {
	tempDir := t.TempDir()
	outFile := filepath.Join(tempDir, "out.zip")
	err := Zip(filepath.Join("..", "..", "..", "testdata", "configtest.json"), outFile)
	require.NoError(t, err, "Zip must not error")
	o, err := UnZip(outFile, tempDir)
	require.NoError(t, err, "UnZip must not error")
	assert.Len(t, o, 1, "Should extract 1 file")

	folder := filepath.Join("..", "..", "..", "testdata", "gctscript")
	outFolderZip := filepath.Join(tempDir, "out_folder.zip")
	err = Zip(folder, outFolderZip)
	require.NoError(t, err, "Zip must not error")
	o, err = UnZip(outFolderZip, tempDir)
	require.NoError(t, err, "UnZip must not error")
	var found bool
	for i := range o {
		if filepath.Base(o[i]) == "timer.gct" {
			found = true
		}
	}
	assert.True(t, found, "Should find a gctscript in the zip")
	assert.GreaterOrEqual(t, len(o), 6, "Should extract at least 6 files")

	folder = filepath.Join("..", "..", "..", "testdata", "invalid_file.json")
	err = Zip(folder, filepath.Join(tempDir, "invalid.zip"))
	assert.ErrorIs(t, err, fs.ErrNotExist, "Zip should error correctly")

	t.Cleanup(func() { addFilesToZip = addFilesToZipWrapper })
	addFilesToZip = addFilesToZipTestWrapper
	folder = filepath.Join("..", "..", "..", "testdata", "http_mock")
	outFolderZip = filepath.Join(tempDir, "error_zip.zip")
	err = Zip(folder, outFolderZip)
	assert.ErrorContains(t, err, "specific error", "Zip should error correctly")
}

func addFilesToZipTestWrapper(_ *zip.Writer, _ string, _ bool) error {
	return errors.New("specific error")
}

func TestZipSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on Windows")
	}
	tempDir := t.TempDir()
	src := filepath.Join(tempDir, "src")
	require.NoError(t, os.Mkdir(src, 0o750), "Mkdir must not error")
	require.NoError(t, os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0o600), "WriteFile must not error")
	require.NoError(t, os.Symlink("a.txt", filepath.Join(src, "inner.txt")), "Symlink must not error")

	inRootZip := filepath.Join(tempDir, "in_root.zip")
	require.NoError(t, Zip(src, inRootZip), "Zip must not error for a symlink resolving inside the tree")
	o, err := UnZip(inRootZip, filepath.Join(tempDir, "extracted"))
	require.NoError(t, err, "UnZip must not error")
	assert.Len(t, o, 2, "Should archive the file and the symlink target contents")

	outside := filepath.Join(tempDir, "outside.txt")
	require.NoError(t, os.WriteFile(outside, []byte("secret"), 0o600), "WriteFile must not error")
	require.NoError(t, os.Symlink(outside, filepath.Join(src, "escape.txt")), "Symlink must not error")
	err = Zip(src, filepath.Join(tempDir, "escaped.zip"))
	assert.ErrorContains(t, err, "escapes from parent", "Zip should reject a symlink resolving outside the tree")
}

func TestZipDestinationIsSource(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), "payload.txt")
	require.NoError(t, os.WriteFile(p, []byte("keep me"), 0o600), "WriteFile must not error")

	err := Zip(p, p)
	assert.ErrorIs(t, err, errDestinationIsSource, "Zip should refuse to write over its own source")

	b, err := os.ReadFile(p)
	require.NoError(t, err, "ReadFile must not error")
	assert.Equal(t, "keep me", string(b), "source should be left intact")

	alias := filepath.Join(filepath.Dir(p), "alias")
	require.NoError(t, os.Symlink(p, alias), "Symlink must not error")
	err = Zip(p, alias)
	assert.ErrorIs(t, err, errDestinationIsSource, "Zip should refuse a destination symlinked to its source")

	b, err = os.ReadFile(p)
	require.NoError(t, err, "ReadFile must not error")
	assert.Equal(t, "keep me", string(b), "source should survive a symlinked destination")
}

func TestZipDestinationWithinSource(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tree := filepath.Join(root, "tree")
	require.NoError(t, os.MkdirAll(filepath.Join(tree, "sub"), 0o755), "MkdirAll must not error")
	victim := filepath.Join(tree, "sub", "victim.txt")
	require.NoError(t, os.WriteFile(victim, []byte("do not destroy"), 0o600), "WriteFile must not error")

	assert.ErrorIs(t, Zip(tree, victim), errDestinationWithinSource, "Zip should refuse a destination beneath the source")
	b, err := os.ReadFile(victim)
	require.NoError(t, err, "ReadFile must not error")
	assert.Equal(t, "do not destroy", string(b), "a file beneath the source should be left intact")

	assert.ErrorIs(t, Zip(tree, filepath.Join(tree, "out.zip")), errDestinationWithinSource, "Zip should refuse to archive into the source")

	// a destination alongside the source is the normal case and must still work
	assert.NoError(t, Zip(tree, filepath.Join(root, "out.zip")), "Zip should accept a destination beside the source")
}

func TestZipDirectorySymlink(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tree := filepath.Join(root, "tree")
	require.NoError(t, os.MkdirAll(filepath.Join(tree, "sub"), 0o755), "MkdirAll must not error")
	require.NoError(t, os.WriteFile(filepath.Join(tree, "sub", "f.txt"), []byte("hello"), 0o600), "WriteFile must not error")
	require.NoError(t, os.Symlink("sub", filepath.Join(tree, "link")), "Symlink must not error")

	dest := filepath.Join(root, "out.zip")
	require.NoError(t, Zip(tree, dest), "Zip must not error for a directory symlink resolving inside the source")

	r, err := zip.OpenReader(dest)
	require.NoError(t, err, "OpenReader must not error")
	defer r.Close()
	got := make(map[string]uint64, len(r.File))
	for _, f := range r.File {
		got[f.Name] = f.UncompressedSize64
	}
	assert.Contains(t, got, "tree/link", "the link should be recorded as an entry")
	assert.Equal(t, uint64(5), got["tree/sub/f.txt"], "the linked directory contents should be archived by the walk")
}

func TestZipRelativeSourceAbsoluteDestination(t *testing.T) {
	// t.Chdir cannot be combined with t.Parallel
	root := t.TempDir()
	tree := filepath.Join(root, "tree")
	require.NoError(t, os.MkdirAll(tree, 0o755), "MkdirAll must not error")
	require.NoError(t, os.WriteFile(filepath.Join(tree, "a.txt"), []byte("payload"), 0o600), "WriteFile must not error")

	t.Chdir(root)
	assert.ErrorIs(t, Zip("tree", filepath.Join(tree, "out.zip")), errDestinationWithinSource,
		"a relative source must not let an absolute destination inside it through the guard")
}

func TestCheckDestination(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tree := filepath.Join(root, "tree")
	require.NoError(t, os.MkdirAll(filepath.Join(tree, "sub"), 0o755), "MkdirAll must not error")
	file := filepath.Join(root, "a.txt")
	require.NoError(t, os.WriteFile(file, []byte("payload"), 0o600), "WriteFile must not error")
	alias := filepath.Join(root, "alias")
	require.NoError(t, os.Symlink(file, alias), "Symlink must not error")
	inside := filepath.Join(tree, "sub", "inside.txt")
	require.NoError(t, os.WriteFile(inside, []byte("keep"), 0o600), "WriteFile must not error")
	insideAlias := filepath.Join(root, "inside-alias")
	require.NoError(t, os.Symlink(inside, insideAlias), "Symlink must not error")

	treeInfo, err := os.Stat(tree)
	require.NoError(t, err, "Stat must not error")
	fileInfo, err := os.Stat(file)
	require.NoError(t, err, "Stat must not error")

	for _, tc := range []struct {
		name     string
		src      string
		dest     string
		srcInfo  os.FileInfo
		expected error
	}{
		{"dest is src", file, file, fileInfo, errDestinationIsSource},
		{"dest symlinked to src", file, alias, fileInfo, errDestinationIsSource},
		{"dest inside dir src", tree, filepath.Join(tree, "out.zip"), treeInfo, errDestinationWithinSource},
		{"dest below dir src", tree, filepath.Join(tree, "sub", "out.zip"), treeInfo, errDestinationWithinSource},
		{"dest symlinked into dir src", tree, insideAlias, treeInfo, errDestinationWithinSource},
		{"dest beside dir src", tree, filepath.Join(root, "out.zip"), treeInfo, nil},
		{"dest beside file src", file, filepath.Join(root, "a.zip"), fileInfo, nil},
		{"dest parent missing", tree, filepath.Join(root, "absent", "out.zip"), treeInfo, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := checkDestination(tc.src, tc.dest, tc.srcInfo)
			if tc.expected == nil {
				assert.NoError(t, err, "checkDestination should accept the destination")
				return
			}
			assert.ErrorIs(t, err, tc.expected, "checkDestination should reject the destination")
		})
	}
}
