package archive

import (
	"archive/zip"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnZip(t *testing.T) {
	tempDir := t.TempDir()
	files, err := UnZip(filepath.Join("..", "..", "..", "testdata", "testdata.zip"), tempDir)
	require.NoError(t, err, "UnZip must not error")
	assert.Len(t, files, 2, "UnZip should extract both files")

	_, err = UnZip(filepath.Join("..", "..", "..", "testdata", "zip-slip.zip"), tempDir)
	assert.ErrorContains(t, err, "illegal file path", "UnZip should reject an entry escaping the destination")

	_, err = UnZip(filepath.Join("..", "..", "..", "testdata", "configtest.json"), tempDir)
	assert.ErrorIs(t, err, zip.ErrFormat, "UnZip should error on a file that is not an archive")
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
	skipWithoutSymlinks(t)
	tempDir := t.TempDir()
	src := filepath.Join(tempDir, "src")
	require.NoError(t, os.Mkdir(src, 0o750), "Mkdir must not error")
	require.NoError(t, os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0o600), "WriteFile must not error")
	require.NoError(t, os.Symlink("a.txt", filepath.Join(src, "inner.txt")), "Symlink must not error")

	inRootZip := filepath.Join(tempDir, "in_root.zip")
	require.NoError(t, Zip(src, inRootZip), "Zip must not error for a symlink resolving inside the tree")
	extracted := filepath.Join(tempDir, "extracted")
	o, err := UnZip(inRootZip, extracted)
	require.NoError(t, err, "UnZip must not error")
	require.Len(t, o, 2, "UnZip must extract the file and the symlink")
	// the entry carries the target's bytes, so the count alone would pass on an empty one
	inner, err := os.ReadFile(filepath.Join(extracted, "src", "inner.txt"))
	require.NoError(t, err, "ReadFile must not error")
	assert.Equal(t, "hello", string(inner), "the symlink should be archived with its target's contents")

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

	t.Run("destination symlinked to the source", func(t *testing.T) {
		t.Parallel()
		skipWithoutSymlinks(t)
		alias := filepath.Join(filepath.Dir(p), "alias")
		require.NoError(t, os.Symlink(p, alias), "Symlink must not error")
		assert.ErrorIs(t, Zip(p, alias), errDestinationIsSource, "Zip should refuse a destination symlinked to its source")

		b, err := os.ReadFile(p)
		require.NoError(t, err, "ReadFile must not error")
		assert.Equal(t, "keep me", string(b), "source should survive a symlinked destination")
	})
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
	skipWithoutSymlinks(t)
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

// symlinksAvailable reports whether this platform lets the test create a symlink. Windows needs
// Developer Mode or SeCreateSymbolicLinkPrivilege, which CI holds and a stock developer machine
// does not, so callers feature-detect rather than skipping the whole GOOS and losing the coverage
// wherever it does run
func symlinksAvailable(t *testing.T) bool {
	t.Helper()
	dir := t.TempDir()
	return os.Symlink(filepath.Join(dir, "target"), filepath.Join(dir, "link")) == nil
}

// skipWithoutSymlinks skips a test that cannot set up without creating a symlink
func skipWithoutSymlinks(t *testing.T) {
	t.Helper()
	if !symlinksAvailable(t) {
		t.Skip("symlink creation unavailable on this platform")
	}
}

// TestZipRelativePathsThroughSymlinkedWorkingDirectory roots the tree under a symlink so the guard
// is reached wherever symlinks can be created, rather than only where the temp directory is already
// indirect. TestZipRelativeSourceAbsoluteDestination above depends on that indirection: macOS
// resolves t.TempDir under /var, a symlink to private/var, and the Windows CI runner's temp path
// carries the 8.3 alias RUNNER~1, which only EvalSymlinks expands
func TestZipRelativePathsThroughSymlinkedWorkingDirectory(t *testing.T) {
	// t.Chdir cannot be combined with t.Parallel
	skipWithoutSymlinks(t)
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(target, link), "Symlink must not error")
	require.NoError(t, os.MkdirAll(filepath.Join(link, "tree", "sub"), 0o755), "MkdirAll must not error")
	victim := filepath.Join(link, "tree", "sub", "victim.txt")
	require.NoError(t, os.WriteFile(victim, []byte("do not destroy"), 0o600), "WriteFile must not error")

	t.Chdir(link)
	assert.ErrorIs(t, Zip("tree", victim), errDestinationWithinSource,
		"a relative source must not let an absolute destination inside it through the guard")
	assert.ErrorIs(t, Zip(filepath.Join(target, "tree"), filepath.Join("tree", "sub", "victim.txt")), errDestinationWithinSource,
		"an absolute source must not let a relative destination inside it through the guard")

	b, err := os.ReadFile(victim)
	require.NoError(t, err, "ReadFile must not error")
	assert.Equal(t, "do not destroy", string(b), "a file beneath the source should survive both attempts")
}

// TestZipDanglingDestinationSymlink covers a destination symlink whose target does not exist yet.
// EvalSymlinks reports it as not existing, which reads like a missing parent directory, but
// os.Create follows the link and creates the file at the far end
func TestZipDanglingDestinationSymlink(t *testing.T) {
	t.Parallel()
	skipWithoutSymlinks(t)
	root := t.TempDir()
	tree := filepath.Join(root, "tree")
	require.NoError(t, os.MkdirAll(tree, 0o755), "MkdirAll must not error")
	require.NoError(t, os.WriteFile(filepath.Join(tree, "payload.txt"), []byte("payload"), 0o600), "WriteFile must not error")

	inside := filepath.Join(root, "points-inside.zip")
	require.NoError(t, os.Symlink(filepath.Join(tree, "created.zip"), inside), "Symlink must not error")
	assert.ErrorIs(t, Zip(tree, inside), errDestinationWithinSource,
		"Zip should refuse a destination symlinked to a path inside the source")
	assert.NoFileExists(t, filepath.Join(tree, "created.zip"), "the link target should not be created inside the source")

	// one resolving outside the source is the normal case and must still work
	outside := filepath.Join(root, "points-outside.zip")
	require.NoError(t, os.Symlink(filepath.Join(root, "created.zip"), outside), "Symlink must not error")
	require.NoError(t, Zip(tree, outside), "Zip must not error for a destination symlinked outside the source")
	assert.FileExists(t, filepath.Join(root, "created.zip"), "the archive should be written through the link")
}

// TestDestWriteTargetSymlinkChainLimit pins the bound to one iteration per link plus the one that
// finds the non-link at the end, so an off-by-one refuses a chain the walk should reach. It does
// not compare against os.Create: MAXSYMLINKS is 40 on Linux and 32 on macOS, so agreement with the
// kernel is not portable, and the guard being the more permissive of the two is harmless because
// os.Create then refuses the write itself
func TestDestWriteTargetSymlinkChainLimit(t *testing.T) {
	t.Parallel()
	skipWithoutSymlinks(t)
	for _, hops := range []int{maxSymlinkHops, maxSymlinkHops + 1} {
		t.Run(strconv.Itoa(hops)+" hops", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			// the expectation is resolved because destWriteTarget resolves: macOS hands back
			// t.TempDir under /var, a symlink to private/var, and Windows an 8.3 alias
			resolvedRoot, err := filepath.EvalSymlinks(root)
			require.NoError(t, err, "EvalSymlinks must not error")

			target := filepath.Join(root, "end.zip")
			for i := range hops {
				link := filepath.Join(root, "l"+strconv.Itoa(i))
				require.NoError(t, os.Symlink(target, link), "Symlink must not error")
				target = link
			}
			got, err := destWriteTarget(target)
			if hops > maxSymlinkHops {
				assert.ErrorIs(t, err, errTooManySymlinks, "destWriteTarget should refuse a chain longer than the bound")
				return
			}
			require.NoError(t, err, "destWriteTarget must follow a chain the bound allows")
			assert.Equal(t, filepath.Join(resolvedRoot, "end.zip"), got, "destWriteTarget should resolve to the far end of the chain")
		})
	}
}

func TestCheckDestination(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tree := filepath.Join(root, "tree")
	require.NoError(t, os.MkdirAll(filepath.Join(tree, "sub"), 0o755), "MkdirAll must not error")
	file := filepath.Join(root, "a.txt")
	require.NoError(t, os.WriteFile(file, []byte("payload"), 0o600), "WriteFile must not error")
	inside := filepath.Join(tree, "sub", "inside.txt")
	require.NoError(t, os.WriteFile(inside, []byte("keep"), 0o600), "WriteFile must not error")
	alias := filepath.Join(root, "alias")
	insideAlias := filepath.Join(root, "inside-alias")
	// the two aliased cases are skipped rather than the whole table where links cannot be made
	links := symlinksAvailable(t)
	if links {
		require.NoError(t, os.Symlink(file, alias), "Symlink must not error")
		require.NoError(t, os.Symlink(inside, insideAlias), "Symlink must not error")
	}

	treeInfo, err := os.Stat(tree)
	require.NoError(t, err, "Stat must not error")
	fileInfo, err := os.Stat(file)
	require.NoError(t, err, "Stat must not error")

	for _, tc := range []struct {
		name         string
		src          string
		dest         string
		srcInfo      os.FileInfo
		expected     error
		needsSymlink bool
	}{
		{name: "dest is src", src: file, dest: file, srcInfo: fileInfo, expected: errDestinationIsSource},
		{name: "dest symlinked to src", src: file, dest: alias, srcInfo: fileInfo, expected: errDestinationIsSource, needsSymlink: true},
		{name: "dest inside dir src", src: tree, dest: filepath.Join(tree, "out.zip"), srcInfo: treeInfo, expected: errDestinationWithinSource},
		{name: "dest below dir src", src: tree, dest: filepath.Join(tree, "sub", "out.zip"), srcInfo: treeInfo, expected: errDestinationWithinSource},
		{name: "dest symlinked into dir src", src: tree, dest: insideAlias, srcInfo: treeInfo, expected: errDestinationWithinSource, needsSymlink: true},
		{name: "dest beside dir src", src: tree, dest: filepath.Join(root, "out.zip"), srcInfo: treeInfo},
		{name: "dest beside file src", src: file, dest: filepath.Join(root, "a.zip"), srcInfo: fileInfo},
		{name: "dest parent missing", src: tree, dest: filepath.Join(root, "absent", "out.zip"), srcInfo: treeInfo},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.needsSymlink && !links {
				t.Skip("symlink creation unavailable on this platform")
			}
			err := checkDestination(tc.src, tc.dest, tc.srcInfo)
			if tc.expected == nil {
				assert.NoError(t, err, "checkDestination should accept the destination")
				return
			}
			assert.ErrorIs(t, err, tc.expected, "checkDestination should reject the destination")
		})
	}
}
