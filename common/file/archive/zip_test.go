package archive

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

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
	assert.Len(t, o, 1, "UnZip should extract 1 file")

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
	assert.True(t, found, "UnZip should find a gctscript in the zip")
	assert.GreaterOrEqual(t, len(o), 6, "UnZip should extract at least 6 files")

	folder = filepath.Join("..", "..", "..", "testdata", "invalid_file.json")
	err = Zip(folder, filepath.Join(tempDir, "invalid.zip"))
	assert.ErrorIs(t, err, fs.ErrNotExist, "Zip should error correctly")

	t.Cleanup(func() { addFilesToZip = addFilesToZipWrapper })
	addFilesToZip = func(*zip.Writer, string, os.FileInfo, output) error {
		return errors.New("specific error")
	}
	folder = filepath.Join("..", "..", "..", "testdata", "http_mock")
	outFolderZip = filepath.Join(tempDir, "error_zip.zip")
	err = Zip(folder, outFolderZip)
	assert.ErrorContains(t, err, "specific error", "Zip should error correctly")
}

func TestZipRejectsSymlinks(t *testing.T) {
	t.Parallel()
	skipWithoutSymlinks(t)

	// a tree per case, so a link added by one is never seen by another
	newLinkedTree := func(t *testing.T) (root, src, outside string) {
		t.Helper()
		root = t.TempDir()
		src = filepath.Join(root, "src")
		require.NoError(t, os.MkdirAll(filepath.Join(src, "sub"), 0o750), "MkdirAll must not error")
		require.NoError(t, os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0o600), "WriteFile must not error")
		outside = filepath.Join(root, "outside.txt")
		require.NoError(t, os.WriteFile(outside, []byte("secret"), 0o600), "WriteFile must not error")
		return root, src, outside
	}

	for _, tc := range []struct {
		name   string
		link   string
		target func(src, outside string) string
	}{
		{
			name:   "to a directory inside the tree",
			link:   "dirlink",
			target: func(string, string) string { return "sub" },
		},
		{
			name:   "to a file outside the tree",
			link:   "escape.txt",
			target: func(_, outside string) string { return outside },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, src, outside := newLinkedTree(t)
			require.NoError(t, os.Symlink(tc.target(src, outside), filepath.Join(src, tc.link)), "Symlink must not error")

			dest := filepath.Join(root, "out.zip")
			assert.ErrorIs(t, Zip(src, dest), errUnsupportedFileType, "Zip should reject a symlink under the source")
			assert.NoFileExists(t, dest, "the failed archive should be removed")
		})
	}

	t.Run("symlinked source", func(t *testing.T) {
		t.Parallel()
		root, src, _ := newLinkedTree(t)
		srcLink := filepath.Join(root, "srclink")
		require.NoError(t, os.Symlink(src, srcLink), "Symlink must not error")

		dest := filepath.Join(root, "out.zip")
		assert.ErrorIs(t, Zip(srcLink, dest), errUnsupportedFileType, "Zip should reject a symlinked source")
		assert.NoFileExists(t, dest, "a rejected source should not create the destination")
	})

	// a trailing separator names the directory the link points at, so it is archived rather than
	// rejected, and the entries inside it are still checked
	t.Run("symlinked source with a trailing separator", func(t *testing.T) {
		t.Parallel()
		root, src, _ := newLinkedTree(t)
		srcLink := filepath.Join(root, "srclink") + string(filepath.Separator)
		require.NoError(t, os.Symlink(src, filepath.Join(root, "srclink")), "Symlink must not error")

		dest := filepath.Join(root, "out.zip")
		require.NoError(t, Zip(srcLink, dest), "Zip must not error for a symlinked source named with a trailing separator")
		o, err := UnZip(dest, filepath.Join(root, "dereferenced"))
		require.NoError(t, err, "UnZip must not error")
		require.Len(t, o, 1, "UnZip must extract the file behind the link")
		b, err := os.ReadFile(o[0])
		require.NoError(t, err, "ReadFile must not error")
		assert.Equal(t, "hello", string(b), "the dereferenced source should carry its own contents")

		require.NoError(t, os.Symlink("a.txt", filepath.Join(src, "inner.txt")), "Symlink must not error")
		assert.ErrorIs(t, Zip(srcLink, filepath.Join(root, "deref_link.zip")), errUnsupportedFileType,
			"Zip should still reject a symlink under a dereferenced source")
	})
}

func TestZipDestinationIsSource(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), "payload.txt")
	require.NoError(t, os.WriteFile(p, []byte("keep me"), 0o600), "WriteFile must not error")

	err := Zip(p, p)
	assert.ErrorIs(t, err, fs.ErrExist, "Zip should refuse to write over its own source")

	b, err := os.ReadFile(p)
	require.NoError(t, err, "ReadFile must not error")
	assert.Equal(t, "keep me", string(b), "source should be left intact")

	t.Run("destination hard linked to the source", func(t *testing.T) {
		t.Parallel()
		skipWithoutHardLinks(t)
		alias := filepath.Join(filepath.Dir(p), "hardlink")
		require.NoError(t, os.Link(p, alias), "Link must not error")
		assert.ErrorIs(t, Zip(p, alias), fs.ErrExist, "Zip should refuse a destination hard linked to its source")

		b, err := os.ReadFile(p)
		require.NoError(t, err, "ReadFile must not error")
		assert.Equal(t, "keep me", string(b), "source should survive a hard linked destination")
	})

	t.Run("destination symlink", func(t *testing.T) {
		t.Parallel()
		skipWithoutSymlinks(t)
		dangling := filepath.Join(filepath.Dir(p), "dangling")
		target := filepath.Join(filepath.Dir(p), "created.zip")
		require.NoError(t, os.Symlink(target, dangling), "Symlink must not error")

		assert.ErrorIs(t, Zip(p, dangling), fs.ErrExist, "Zip should refuse a dangling symlinked destination")
		assert.NoFileExists(t, target, "the link target should not be created")

		require.NoError(t, os.WriteFile(target, []byte("keep target"), 0o600), "WriteFile must not error")
		assert.ErrorIs(t, Zip(p, dangling), fs.ErrExist, "Zip should refuse a symlink to an existing target")
		b, err := os.ReadFile(target)
		require.NoError(t, err, "ReadFile must not error")
		assert.Equal(t, "keep target", string(b), "the existing symlink target should be preserved")

		require.NoError(t, os.Remove(dangling), "Remove must not error")
		require.NoError(t, os.Symlink(p, dangling), "Symlink must not error")
		assert.ErrorIs(t, Zip(p, dangling), fs.ErrExist, "Zip should refuse a destination symlinked to its source")
		b, err = os.ReadFile(p)
		require.NoError(t, err, "ReadFile must not error")
		assert.Equal(t, "keep me", string(b), "the source should survive a symlinked destination")
	})
}

// newSourceTree builds a source directory holding a file that a rejected destination must leave
// intact
func newSourceTree(t *testing.T) (root, tree, victim string) {
	t.Helper()
	root = t.TempDir()
	tree = filepath.Join(root, "tree")
	require.NoError(t, os.MkdirAll(filepath.Join(tree, "sub"), 0o755), "MkdirAll must not error")
	victim = filepath.Join(tree, "sub", "victim.txt")
	require.NoError(t, os.WriteFile(victim, []byte("do not destroy"), 0o600), "WriteFile must not error")
	return root, tree, victim
}

func assertVictimIntact(t *testing.T, victim string) {
	t.Helper()
	b, err := os.ReadFile(victim)
	require.NoError(t, err, "ReadFile must not error")
	assert.Equal(t, "do not destroy", string(b), "a file inside the source should be left intact")
}

func TestZipDestinationWithinSource(t *testing.T) {
	t.Parallel()
	root, tree, victim := newSourceTree(t)

	// an existing file inside the source is refused by the exclusive create, before the walk
	assert.ErrorIs(t, Zip(tree, victim), fs.ErrExist, "Zip should refuse a destination beneath the source")
	assertVictimIntact(t, victim)

	// a new one is caught by identity when the walk opens it, so no spelling of the path hides it
	dests := []string{filepath.Join(tree, "out.zip"), filepath.Join(tree, "sub", "out.zip")}
	// Windows collapses the ".." lexically, so there that spelling names a path outside the source
	// and the case being covered does not arise
	if runtime.GOOS != "windows" && symlinksAvailable(t) {
		alias := filepath.Join(root, "alias")
		require.NoError(t, os.Symlink(filepath.Join(tree, "sub"), alias), "Symlink must not error")
		sep := string(filepath.Separator)
		dests = append(dests, alias+sep+".."+sep+"out.zip")
	}
	for _, dest := range dests {
		err := Zip(tree, dest)
		assert.ErrorIsf(t, err, errDestinationWithinSource, "Zip should refuse to archive into the source at %q", dest)
		assert.ErrorContainsf(t, err, dest, "the refusal should name the destination the caller passed, %q", dest)
		assert.NoFileExistsf(t, dest, "the partial archive at %q should be removed", dest)
	}

	// a destination alongside the source is the normal case and must still work
	assert.NoError(t, Zip(tree, filepath.Join(root, "out.zip")), "Zip should accept a destination beside the source")
}

// TestZipRefusesExistingDestination pins the exclusive create. Nothing before it can say what a
// pathname resolves to, so an alias reaching a file inside the source passes every check; refusing
// to open an existing file is what keeps that file intact, whichever way the path is spelled
func TestZipRefusesExistingDestination(t *testing.T) {
	t.Parallel()
	t.Run("a hard link into the source", func(t *testing.T) {
		t.Parallel()
		skipWithoutHardLinks(t)
		root, tree, victim := newSourceTree(t)
		dest := filepath.Join(root, "out.zip")
		require.NoError(t, os.Link(victim, dest), "Link must not error")

		assert.ErrorIs(t, Zip(tree, dest), fs.ErrExist, "Zip should refuse a destination that already exists")
		assertVictimIntact(t, victim)
	})

	t.Run("a previous archive beside the source", func(t *testing.T) {
		t.Parallel()
		root, tree, _ := newSourceTree(t)
		dest := filepath.Join(root, "out.zip")
		require.NoError(t, os.WriteFile(dest, []byte("previous archive"), 0o600), "WriteFile must not error")

		assert.ErrorIs(t, Zip(tree, dest), fs.ErrExist, "Zip should refuse a destination that already exists")
		b, err := os.ReadFile(dest)
		require.NoError(t, err, "ReadFile must not error")
		assert.Equal(t, "previous archive", string(b), "an existing destination should be left intact")
	})
}

// Zip merges the error from closing the zip.Writer, which writes the central directory, so a
// failure there is not reported as a successful archive. Failing the last entry's compressor is
// what reaches it: an earlier one would surface from the next CreateHeader instead
func TestZipReportsWriterCloseFailure(t *testing.T) {
	// replaces the package level addFilesToZip, so it cannot be combined with t.Parallel
	root := t.TempDir()
	src := filepath.Join(root, "src")
	require.NoError(t, os.MkdirAll(src, 0o755), "MkdirAll must not error")
	require.NoError(t, os.WriteFile(filepath.Join(src, "only.txt"), []byte("payload"), 0o600), "WriteFile must not error")

	orig := addFilesToZip
	t.Cleanup(func() { addFilesToZip = orig })
	addFilesToZip = func(z *zip.Writer, s string, srcInfo os.FileInfo, out output) error {
		z.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
			return closeFailingWriteCloser{Writer: w}, nil
		})
		return addFilesToZipWrapper(z, s, srcInfo, out)
	}

	dest := filepath.Join(root, "out.zip")
	assert.ErrorIs(t, Zip(src, dest), errCloseFailed, "Zip should report a failure to close the writer")
	assert.NoFileExists(t, dest, "the archive should be removed when it could not be completed")
}

// TestZipRelativeSourceAbsoluteDestination keeps the mixed-spelling case pinned now that nothing
// compares paths: the destination is recognised by identity when the walk opens it, whichever way
// either side is spelled
func TestZipRelativeSourceAbsoluteDestination(t *testing.T) {
	// t.Chdir cannot be combined with t.Parallel
	root, tree, _ := newSourceTree(t)
	t.Chdir(root)

	assert.ErrorIs(t, Zip("tree", filepath.Join(tree, "out.zip")), errDestinationWithinSource,
		"a relative source should not let an absolute destination inside it through")
	assert.ErrorIs(t, Zip(tree, filepath.Join("tree", "sub", "out.zip")), errDestinationWithinSource,
		"an absolute source should not let a relative destination inside it through")
}

// TestZipFileSourceWithoutDirectory covers a src carrying no directory part, which roots the walk
// at the working directory rather than at a named parent
func TestZipFileSourceWithoutDirectory(t *testing.T) {
	// t.Chdir cannot be combined with t.Parallel
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("payload"), 0o600), "WriteFile must not error")

	t.Chdir(dir)

	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	require.NoError(t, addFilesToZipWrapper(z, "a.txt", statSource(t, "a.txt"), newDestInfo(t)), "addFilesToZipWrapper must not error")
	require.NoError(t, z.Close(), "Close must not error")
	assert.Equal(t, map[string]string{"a.txt": "payload"}, zipEntries(t, &buf), "a bare filename should root the walk at the working directory")

	require.NoError(t, Zip("a.txt", "out.zip"), "Zip must not error for a source with no directory part")
	o, err := UnZip("out.zip", filepath.Join(dir, "extracted"))
	require.NoError(t, err, "UnZip must not error")
	require.Len(t, o, 1, "UnZip must extract the single file")
	b, err := os.ReadFile(o[0])
	require.NoError(t, err, "ReadFile must not error")
	assert.Equal(t, "payload", string(b), "the archived file should carry its contents")
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

// skipWithoutHardLinks skips a test that cannot set up without creating a hard link
func skipWithoutHardLinks(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	require.NoError(t, os.WriteFile(target, nil, 0o600), "WriteFile must not error")
	if os.Link(target, filepath.Join(dir, "link")) != nil {
		t.Skip("hard link creation unavailable on this platform")
	}
}

// skipWithoutSymlinks skips a test that cannot set up without creating a symlink
func skipWithoutSymlinks(t *testing.T) {
	t.Helper()
	if !symlinksAvailable(t) {
		t.Skip("symlink creation unavailable on this platform")
	}
}

// zipEntries decodes an archive a test wrote, mapping each entry name to its contents
func zipEntries(t *testing.T, buf *bytes.Buffer) map[string]string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err, "NewReader must not error")

	got := make(map[string]string, len(r.File))
	for _, f := range r.File {
		require.NotContainsf(t, got, f.Name, "the archive must not repeat entry %q", f.Name)
		if f.FileInfo().IsDir() {
			got[f.Name] = ""
			continue
		}
		rc, err := f.Open()
		require.NoError(t, err, "Open must not error")
		b, err := io.ReadAll(rc)
		require.NoError(t, err, "ReadAll must not error")
		require.NoError(t, rc.Close(), "Close must not error")
		got[f.Name] = string(b)
	}
	return got
}

// newDestInfo stands in for the archive Zip would be writing, at a path outside any source tree
func newDestInfo(t *testing.T) output {
	t.Helper()
	dest := filepath.Join(t.TempDir(), "out.zip")
	require.NoError(t, os.WriteFile(dest, nil, 0o600), "WriteFile must not error")
	i, err := os.Stat(dest)
	require.NoError(t, err, "Stat must not error")
	return output{
		path: dest,
		info: i,
	}
}

func TestAddFilesToZipWrapper(t *testing.T) {
	t.Parallel()
	_, _, victim := newSourceTree(t)

	t.Run("directory source", func(t *testing.T) {
		t.Parallel()
		// its own tree, given a mode no plausible hardcoded default would match, so that the
		// permission assertion below can tell a copied mode from an invented one
		_, tree, _ := newSourceTree(t)
		require.NoError(t, os.Chmod(tree, 0o750), "Chmod must not error")

		var buf bytes.Buffer
		z := zip.NewWriter(&buf)
		require.NoError(t, addFilesToZipWrapper(z, tree, statSource(t, tree), newDestInfo(t)), "addFilesToZipWrapper must not error")
		require.NoError(t, z.Close(), "Close must not error")

		assert.Equal(t, map[string]string{"tree/": "", "tree/sub/": "", "tree/sub/victim.txt": "do not destroy"},
			zipEntries(t, &buf), "the walk should record the source directory and everything under it")

		r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		require.NoError(t, err, "NewReader must not error")
		i, err := os.Stat(tree)
		require.NoError(t, err, "Stat must not error")
		for _, f := range r.File {
			if f.Name == "tree/" {
				assert.Equal(t, i.Mode(), f.Mode(), "a directory entry should carry the directory's permissions")
			}
		}
	})

	t.Run("file source", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		z := zip.NewWriter(&buf)
		require.NoError(t, addFilesToZipWrapper(z, victim, statSource(t, victim), newDestInfo(t)), "addFilesToZipWrapper must not error")
		require.NoError(t, z.Close(), "Close must not error")

		assert.Equal(t, map[string]string{"victim.txt": "do not destroy"}, zipEntries(t, &buf),
			"a lone file should be recorded under its own name")
	})

	t.Run("symlink under the source", func(t *testing.T) {
		t.Parallel()
		skipWithoutSymlinks(t)
		_, linked, _ := newSourceTree(t)
		require.NoError(t, os.Symlink("sub", filepath.Join(linked, "link")), "Symlink must not error")

		err := addFilesToZipWrapper(zip.NewWriter(&bytes.Buffer{}), linked, statSource(t, linked), newDestInfo(t))
		assert.ErrorIs(t, err, errUnsupportedFileType, "addFilesToZipWrapper should reject a symlink under the source")
	})

	// an entry that cannot be finished fails the next CreateHeader, and the walk reaches names in
	// lexical order, so "second" decides whether that lands on the file branch or the directory one
	for _, second := range []string{"b.txt", "sub"} {
		t.Run("header that cannot be created before "+second, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o600), "WriteFile must not error")
			if filepath.Ext(second) == "" {
				require.NoError(t, os.Mkdir(filepath.Join(dir, second), 0o755), "Mkdir must not error")
			} else {
				require.NoError(t, os.WriteFile(filepath.Join(dir, second), []byte("b"), 0o600), "WriteFile must not error")
			}

			z := zip.NewWriter(&bytes.Buffer{})
			z.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
				return closeFailingWriteCloser{Writer: w}, nil
			})

			err := addFilesToZipWrapper(z, dir, statSource(t, dir), newDestInfo(t))
			assert.ErrorIsf(t, err, errCloseFailed, "addFilesToZipWrapper should report a header it cannot create before %q", second)
		})
	}

	t.Run("source that cannot be opened", func(t *testing.T) {
		t.Parallel()
		err := addFilesToZipWrapper(zip.NewWriter(&bytes.Buffer{}), filepath.Join(t.TempDir(), "absent"), statSource(t, t.TempDir()), newDestInfo(t))
		assert.ErrorIs(t, err, fs.ErrNotExist, "addFilesToZipWrapper should report a source it cannot open")
	})

	// Zip checks src too, so without this case either check alone passes the suite and neither is
	// shown to be load-bearing
	t.Run("symlink as a lone source", func(t *testing.T) {
		t.Parallel()
		skipWithoutSymlinks(t)
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "target.txt"), []byte("payload"), 0o600), "WriteFile must not error")
		link := filepath.Join(dir, "link.txt")
		require.NoError(t, os.Symlink("target.txt", link), "Symlink must not error")

		err := addFilesToZipWrapper(zip.NewWriter(&bytes.Buffer{}), link, statSource(t, link), newDestInfo(t))
		assert.ErrorIs(t, err, errUnsupportedFileType, "addFilesToZipWrapper should reject a symlinked lone source")
	})

	// the parent opens, so this reaches the walk itself rather than stopping at the root
	t.Run("source whose parent opens but which does not exist", func(t *testing.T) {
		t.Parallel()
		err := addFilesToZipWrapper(zip.NewWriter(&bytes.Buffer{}), filepath.Join(t.TempDir(), "absent.txt"), statSource(t, victim), newDestInfo(t))
		assert.ErrorIs(t, err, fs.ErrNotExist, "addFilesToZipWrapper should report an error the walk hands it")
	})
}

// newEntry describes a file the way the walk would have found it
func newEntry(t *testing.T, root *os.Root, path, name string) entry {
	t.Helper()
	i, err := root.Lstat(path)
	require.NoError(t, err, "Lstat must not error")
	return entry{
		path: path,
		name: name,
		info: i,
	}
}

func TestAddFileToZip(t *testing.T) {
	t.Parallel()
	_, tree, victim := newSourceTree(t)
	root, err := os.OpenRoot(filepath.Join(tree, "sub"))
	require.NoError(t, err, "OpenRoot must not error")
	t.Cleanup(func() { assert.NoError(t, root.Close(), "Close should not error") })

	t.Run("regular file", func(t *testing.T) {
		t.Parallel()
		// a mode and a modification time no plausible hardcoded value would match, so the
		// assertions below can tell a copied header field from an invented one
		require.NoError(t, os.Chmod(victim, 0o640), "Chmod must not error")
		modified := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
		require.NoError(t, os.Chtimes(victim, modified, modified), "Chtimes must not error")

		var buf bytes.Buffer
		z := zip.NewWriter(&buf)
		require.NoError(t, addFileToZip(z, root, newEntry(t, root, "victim.txt", "renamed.txt"), newDestInfo(t)), "addFileToZip must not error")
		require.NoError(t, z.Close(), "Close must not error")

		assert.Equal(t, map[string]string{"renamed.txt": "do not destroy"}, zipEntries(t, &buf),
			"the entry should carry the name it was given and the file's contents")

		r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		require.NoError(t, err, "NewReader must not error")
		require.Len(t, r.File, 1, "the archive must hold the one entry")
		assert.Equal(t, zip.Deflate, r.File[0].Method, "the entry should be deflated rather than stored")

		i, err := os.Stat(victim)
		require.NoError(t, err, "Stat must not error")
		assert.Equal(t, i.Mode(), r.File[0].Mode(), "the entry should carry the file's permissions")
		assert.WithinDuration(t, i.ModTime(), r.File[0].Modified, 2*time.Second, "the entry should carry the file's modification time")
	})

	// the walk only reaches this helper for a directory entry that was not a directory, so the
	// type has to be tested again on what the open actually returned
	t.Run("opened file that is not regular", func(t *testing.T) {
		t.Parallel()
		err := addFileToZip(zip.NewWriter(&bytes.Buffer{}), root, newEntry(t, root, ".", "dir"), newDestInfo(t))
		assert.ErrorIs(t, err, errUnsupportedFileType, "addFileToZip should reject an opened file that is not regular")
	})

	// the entry before this one wrote fine but cannot be finished, which is what fails this
	// header. Reached through the wrapper today, so this covers the branch directly
	t.Run("header that cannot be created", func(t *testing.T) {
		t.Parallel()
		z := zip.NewWriter(&bytes.Buffer{})
		z.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
			return closeFailingWriteCloser{Writer: w}, nil
		})
		require.NoError(t, addFileToZip(z, root, newEntry(t, root, "victim.txt", "first.txt"), newDestInfo(t)),
			"addFileToZip must not error for the entry that cannot be finished")

		err := addFileToZip(z, root, newEntry(t, root, "victim.txt", "second.txt"), newDestInfo(t))
		assert.ErrorIs(t, err, errCloseFailed, "addFileToZip should report a header it cannot create")
	})

	// the walk refuses a link it sees, so this covers one appearing between that check and the
	// open. The entry describes the file the link points at, so the identity check below cannot
	// catch it either: opening through the root is the only thing that keeps the archive inside src
	t.Run("entry naming a link out of the root", func(t *testing.T) {
		t.Parallel()
		skipWithoutSymlinks(t)
		// stated while the file is inside the root, then that file is moved out and a link to it
		// left at the old name, so the recorded identity still matches what an ordinary open finds
		inside := filepath.Join(tree, "sub", "escape.txt")
		require.NoError(t, os.WriteFile(inside, []byte("not in the source"), 0o600), "WriteFile must not error")
		e := newEntry(t, root, "escape.txt", "escape.txt")
		outside := filepath.Join(tree, "escape-target.txt")
		require.NoError(t, os.Rename(inside, outside), "Rename must not error")
		require.NoError(t, os.Symlink(outside, inside), "Symlink must not error")

		var buf bytes.Buffer
		z := zip.NewWriter(&buf)
		err := addFileToZip(z, root, e, newDestInfo(t))
		require.Error(t, err, "addFileToZip must refuse an entry resolving outside the root")
		assert.ErrorContains(t, err, "opening "+filepath.Join(root.Name(), "escape.txt"), "the refusal should say what it was opening")

		require.NoError(t, z.Close(), "Close must not error")
		assert.Empty(t, zipEntries(t, &buf), "nothing outside the root should reach the archive")
	})

	// the walk states a name and this helper opens it; between the two the name can come to refer
	// to a different file, which the entry it was given no longer describes
	t.Run("file replaced since the walk stated it", func(t *testing.T) {
		t.Parallel()
		other := filepath.Join(tree, "sub", "other.txt")
		require.NoError(t, os.WriteFile(other, []byte("a different file"), 0o600), "WriteFile must not error")

		e := newEntry(t, root, "victim.txt", "victim.txt")
		e.info = newEntry(t, root, "other.txt", "other.txt").info

		err := addFileToZip(zip.NewWriter(&bytes.Buffer{}), root, e, newDestInfo(t))
		assert.ErrorIs(t, err, errEntryReplaced, "addFileToZip should refuse a file that is not the one the walk stated")
	})

	t.Run("file that is the archive being written", func(t *testing.T) {
		t.Parallel()
		destInfo, err := os.Stat(victim)
		require.NoError(t, err, "Stat must not error")

		err = addFileToZip(zip.NewWriter(&bytes.Buffer{}), root, newEntry(t, root, "victim.txt", "victim.txt"), output{
			path: victim,
			info: destInfo,
		})
		assert.ErrorIs(t, err, errDestinationWithinSource, "addFileToZip should refuse to archive its own output")
		assert.ErrorContains(t, err, victim, "the refusal should name the destination the caller passed")
	})

	t.Run("compressor that cannot write", func(t *testing.T) {
		t.Parallel()
		z := zip.NewWriter(&bytes.Buffer{})
		z.RegisterCompressor(zip.Deflate, func(io.Writer) (io.WriteCloser, error) { return failingWriteCloser{}, nil })

		err := addFileToZip(z, root, newEntry(t, root, "victim.txt", "victim.txt"), newDestInfo(t))
		assert.ErrorIs(t, err, errWriteFailed, "addFileToZip should report a copy failure")
		assert.ErrorContains(t, err, filepath.Join(root.Name(), "victim.txt"), "the failure should name the entry being copied")
	})

	t.Run("file that cannot be opened", func(t *testing.T) {
		t.Parallel()
		err := addFileToZip(zip.NewWriter(&bytes.Buffer{}), root, entry{
			path: "absent.txt",
			name: "absent.txt",
		}, newDestInfo(t))
		assert.ErrorIs(t, err, fs.ErrNotExist, "addFileToZip should report a file it cannot open")
	})
}

// errWriteFailed stands in for a compressor that cannot write
var errWriteFailed = errors.New("write failed")

type failingWriteCloser struct{}

func (failingWriteCloser) Write([]byte) (int, error) { return 0, errWriteFailed }
func (failingWriteCloser) Close() error              { return nil }

// errCloseFailed stands in for a compressor that cannot finish an entry, which is what makes the
// next CreateHeader fail
var errCloseFailed = errors.New("close failed")

type closeFailingWriteCloser struct{ io.Writer }

func (closeFailingWriteCloser) Close() error { return errCloseFailed }

// duringFirstEntry runs fn once while the first entry is being compressed, so a change to the tree
// lands after the walk has read the directory and before it reaches the entry that changed
func duringFirstEntry(t *testing.T, fn func()) {
	t.Helper()
	orig := addFilesToZip
	t.Cleanup(func() { addFilesToZip = orig })

	var once sync.Once
	addFilesToZip = func(z *zip.Writer, src string, srcInfo os.FileInfo, out output) error {
		z.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
			once.Do(fn)
			return flate.NewWriter(w, flate.DefaultCompression)
		})
		return addFilesToZipWrapper(z, src, srcInfo, out)
	}
}

// TestZipTreeChangedMidWalk covers a source changing under the archive. A directory is read once,
// so an entry can be replaced after that read and before the walk reaches it, which is why the
// entry is stated afresh rather than taken from what the walk cached
func TestZipTreeChangedMidWalk(t *testing.T) {
	// replaces the package level addFilesToZip, so it cannot be combined with t.Parallel
	newTree := func(t *testing.T) (root, src, later string) {
		t.Helper()
		root = t.TempDir()
		src = filepath.Join(root, "src")
		require.NoError(t, os.MkdirAll(src, 0o755), "MkdirAll must not error")
		require.NoError(t, os.WriteFile(filepath.Join(src, "aaa.txt"), []byte("first entry"), 0o600), "WriteFile must not error")
		later = filepath.Join(src, "zzz.txt")
		require.NoError(t, os.WriteFile(later, []byte("original"), 0o600), "WriteFile must not error")
		return root, src, later
	}

	t.Run("entry replaced by a symlink", func(t *testing.T) {
		skipWithoutSymlinks(t)
		root, src, later := newTree(t)
		outside := filepath.Join(root, "outside.txt")
		require.NoError(t, os.WriteFile(outside, []byte("secret"), 0o600), "WriteFile must not error")
		duringFirstEntry(t, func() {
			require.NoError(t, os.Remove(later), "Remove must not error")
			require.NoError(t, os.Symlink(outside, later), "Symlink must not error")
		})

		dest := filepath.Join(root, "out.zip")
		assert.ErrorIs(t, Zip(src, dest), errUnsupportedFileType, "Zip should reject an entry replaced by a symlink after the walk read the directory")
		assert.NoFileExists(t, dest, "the partial archive should be removed")
	})

	// a replacement of the same kind is archived as whatever the name refers to when it is opened.
	// Detecting it would mean comparing identity against what reading the directory reported, and
	// os.SameFile is documented to apply only to this package's Stat results
	t.Run("entry replaced by another file", func(t *testing.T) {
		root, src, later := newTree(t)
		impostor := filepath.Join(root, "impostor.txt")
		require.NoError(t, os.WriteFile(impostor, []byte("an impostor"), 0o600), "WriteFile must not error")
		duringFirstEntry(t, func() { require.NoError(t, os.Rename(impostor, later), "Rename must not error") })

		dest := filepath.Join(root, "out.zip")
		require.NoError(t, Zip(src, dest), "Zip must not error for a replacement of the same kind")
		o, err := UnZip(dest, filepath.Join(root, "extracted"))
		require.NoError(t, err, "UnZip must not error")
		require.Len(t, o, 2, "UnZip must extract both entries")
		b, err := os.ReadFile(filepath.Join(root, "extracted", "src", "zzz.txt"))
		require.NoError(t, err, "ReadFile must not error")
		assert.Equal(t, "an impostor", string(b), "the entry should carry what the name referred to when it was opened")
	})

	for _, tc := range []struct {
		name    string
		replace func(t *testing.T, path string)
	}{
		{
			name: "file replaced by a directory",
			replace: func(t *testing.T, path string) {
				t.Helper()
				require.NoError(t, os.Remove(path), "Remove must not error")
				require.NoError(t, os.Mkdir(path, 0o755), "Mkdir must not error")
			},
		},
		{
			name: "directory replaced by a file",
			replace: func(t *testing.T, path string) {
				t.Helper()
				require.NoError(t, os.Remove(path), "Remove must not error")
				require.NoError(t, os.WriteFile(path, []byte("no longer a directory"), 0o600), "WriteFile must not error")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, src, later := newTree(t)
			if tc.name == "directory replaced by a file" {
				require.NoError(t, os.Remove(later), "Remove must not error")
				require.NoError(t, os.Mkdir(later, 0o755), "Mkdir must not error")
			}
			duringFirstEntry(t, func() { tc.replace(t, later) })

			dest := filepath.Join(root, "out.zip")
			assert.ErrorIs(t, Zip(src, dest), errEntryReplaced, "Zip should refuse an entry whose kind changed after the walk read it")
			assert.NoFileExists(t, dest, "the partial archive should be removed")
		})
	}

	t.Run("entry removed", func(t *testing.T) {
		root, src, later := newTree(t)
		duringFirstEntry(t, func() { require.NoError(t, os.Remove(later), "Remove must not error") })

		dest := filepath.Join(root, "out.zip")
		assert.ErrorIs(t, Zip(src, dest), fs.ErrNotExist, "Zip should report an entry removed after the walk read the directory")
		assert.NoFileExists(t, dest, "the partial archive should be removed")
	})
}

func TestCheckFileType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mode     fs.FileMode
		expected error
	}{
		{
			mode: 0o644,
		},
		{
			mode: fs.ModeDir | 0o755,
		},
		{
			mode:     fs.ModeSymlink | 0o777,
			expected: errUnsupportedFileType,
		},
		// a Windows directory junction, which no other platform can create
		{
			mode:     fs.ModeIrregular | 0o644,
			expected: errUnsupportedFileType,
		},
		// but a directory carrying a reparse point that is not a junction is still walked
		{
			mode: fs.ModeDir | fs.ModeIrregular | 0o755,
		},
		{
			mode:     fs.ModeNamedPipe | 0o644,
			expected: errUnsupportedFileType,
		},
		{
			mode:     fs.ModeDevice | fs.ModeCharDevice | 0o666,
			expected: errUnsupportedFileType,
		},
		{
			mode:     fs.ModeSocket | 0o755,
			expected: errUnsupportedFileType,
		},
	} {
		t.Run(tc.mode.String(), func(t *testing.T) {
			t.Parallel()
			err := checkFileType(tc.mode, "entry")
			if tc.expected == nil {
				assert.NoError(t, err, "checkFileType should accept the mode")
				return
			}
			assert.ErrorIs(t, err, tc.expected, "checkFileType should reject the mode")
		})
	}
}

func TestZipInvalidDestinationParent(t *testing.T) {
	t.Parallel()
	root, tree, _ := newSourceTree(t)
	file := filepath.Join(root, "a.txt")
	require.NoError(t, os.WriteFile(file, []byte("payload"), 0o600), "WriteFile must not error")

	assert.ErrorIs(t, Zip(tree, filepath.Join(root, "absent", "out.zip")), fs.ErrNotExist,
		"Zip should reject a missing destination parent")
	assert.Error(t, Zip(tree, filepath.Join(file, "out.zip")), "Zip should reject a regular file as the destination parent")
}

func TestRootFSOpen(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "payload.txt"), []byte("payload"), 0o600), "WriteFile must not error")
	root, err := os.OpenRoot(dir)
	require.NoError(t, err, "OpenRoot must not error")
	t.Cleanup(func() { assert.NoError(t, root.Close(), "Close should not error") })
	r := rootFS{root}
	f, err := r.Open("payload.txt")
	require.NoError(t, err, "Open must open a regular file")
	b, err := io.ReadAll(f)
	require.NoError(t, err, "ReadAll must not error")
	assert.Equal(t, "payload", string(b), "Open should preserve the file contents")
	require.NoError(t, f.Close(), "Close must not error")

	f, err = r.Open(".")
	require.NoError(t, err, "Open must open a directory")
	i, err := f.Stat()
	require.NoError(t, err, "Stat must not error")
	assert.True(t, i.IsDir(), "Open should preserve the directory type")
	require.NoError(t, f.Close(), "Close must not error")

	_, err = r.Open("absent")
	assert.ErrorIs(t, err, fs.ErrNotExist, "Open should preserve a missing-file error")
}

func statSource(t *testing.T, src string) os.FileInfo {
	t.Helper()
	info, err := lstatSource(src)
	require.NoError(t, err, "Lstat must not error")
	return info
}

func TestAddFilesToZipWrapperSourceReplaced(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"directory", "symlink", "file"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			if kind == "symlink" {
				skipWithoutSymlinks(t)
			}
			dir := t.TempDir()
			src := filepath.Join(dir, "src")
			if kind == "file" {
				require.NoError(t, os.WriteFile(src, []byte("original"), 0o600), "WriteFile must not error")
			} else {
				require.NoError(t, os.Mkdir(src, 0o755), "Mkdir must not error")
			}
			info := statSource(t, src)
			require.NoError(t, os.Rename(src, filepath.Join(dir, "original")), "Rename must not error")
			switch kind {
			case "file":
				require.NoError(t, os.WriteFile(src, []byte("replacement"), 0o600), "WriteFile must not error")
			case "directory":
				require.NoError(t, os.Mkdir(src, 0o755), "Mkdir must not error")
			case "symlink":
				other := filepath.Join(dir, "other")
				require.NoError(t, os.Mkdir(other, 0o755), "Mkdir must not error")
				require.NoError(t, os.WriteFile(filepath.Join(other, "secret.txt"), []byte("outside source"), 0o600), "WriteFile must not error")
				require.NoError(t, os.Symlink(other, src), "Symlink must not error")
			}
			var buf bytes.Buffer
			z := zip.NewWriter(&buf)
			err := addFilesToZipWrapper(z, src, info, newDestInfo(t))
			assert.ErrorIs(t, err, errEntryReplaced, "addFilesToZipWrapper should reject a replaced source")
			require.NoError(t, z.Close(), "Close must not error")
			assert.Empty(t, zipEntries(t, &buf), "a replaced source should contribute no archive entries")
		})
	}
}

// TestZipSourceSpelledWithDotDot covers a source whose last element is "..". Recording entries
// beneath filepath.Base of that spelling names them "../", which UnZip and any conformant extractor
// refuse, so the name is taken from the directory the path resolves to instead
func TestZipSourceSpelledWithDotDot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := filepath.Join(root, "src")
	require.NoError(t, os.MkdirAll(filepath.Join(src, "sub"), 0o755), "MkdirAll must not error")
	require.NoError(t, os.WriteFile(filepath.Join(src, "payload.txt"), []byte("payload"), 0o600), "WriteFile must not error")

	dest := filepath.Join(root, "out.zip")
	sep := string(filepath.Separator)
	require.NoError(t, Zip(filepath.Join(src, "sub")+sep+"..", dest), "Zip must not error for a source spelled with a trailing ..")

	o, err := UnZip(dest, filepath.Join(root, "extracted"))
	require.NoError(t, err, "UnZip must accept the archive, which it refuses when entries are named \"../\"")
	require.Len(t, o, 1, "UnZip must extract the single file")
	assert.Equal(t, filepath.Join(root, "extracted", "src", "payload.txt"), o[0], "the entry should sit beneath the directory the source resolves to")
}

func TestArchiveName(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, tc := range []struct {
		name, src, exp string
		err            error
	}{
		{name: "plain directory", src: filepath.Join(dir, "src"), exp: "src"},
		{name: "source ending in ..", src: filepath.Join(dir, "src", "sub") + string(filepath.Separator) + "..", exp: "src"},
		{name: "trailing separator", src: filepath.Join(dir, "src") + string(filepath.Separator), exp: "src"},
		{name: "root", src: string(filepath.Separator), err: errUnnameableSource},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := archiveName(tc.src)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "archiveName should refuse a source that cannot name an entry")
				return
			}
			require.NoError(t, err, "archiveName must not error")
			assert.Equal(t, tc.exp, got, "archiveName should name the directory the source resolves to")
		})
	}
}

// TestZipDotSourceKeepsItsEntries pins "." keeping its entries at the top of the archive, and its
// root recorded under no name at all, the "./" it would otherwise carry being one UnZip refuses
func TestZipDotSourceKeepsItsEntries(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "sub"), 0o755), "Mkdir must not error")
	require.NoError(t, os.WriteFile(filepath.Join(root, "payload"), []byte("payload"), 0o600), "WriteFile must not error")
	require.NoError(t, os.WriteFile(filepath.Join(root, "sub", "nested"), []byte("nested"), 0o600), "WriteFile must not error")
	t.Chdir(root)

	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	require.NoError(t, addFilesToZipWrapper(z, ".", statSource(t, "."), newDestInfo(t)), "addFilesToZipWrapper must not error")
	require.NoError(t, z.Close(), "Close must not error")
	assert.Equal(t, map[string]string{"payload": "payload", "sub/": "", "sub/nested": "nested"}, zipEntries(t, &buf),
		"a source named \".\" should keep its entries at the top of the archive")

	dest := filepath.Join(t.TempDir(), "out.zip")
	require.NoError(t, Zip(".", dest), "Zip must not error for a source named \".\"")
	extracted := filepath.Join(t.TempDir(), "extracted")
	o, err := UnZip(dest, extracted)
	require.NoError(t, err, "UnZip must not error on an archive Zip wrote for a source named \".\"")
	require.Len(t, o, 2, "UnZip must extract the file at the top and the one beneath the directory")
	for _, name := range []string{"payload", filepath.Join("sub", "nested")} {
		b, err := os.ReadFile(filepath.Join(extracted, name))
		require.NoErrorf(t, err, "ReadFile must not error for %s", name)
		assert.Equalf(t, filepath.Base(name), string(b), "the extracted %s should carry its contents", name)
	}
}
