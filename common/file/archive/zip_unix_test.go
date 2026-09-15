//go:build unix

package archive

import (
	"archive/zip"
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// filepath.Dir cleans, so a ".." ahead of a link collapses before the kernel resolves the link and
// the archive holds a different file than the caller named
func TestZipFileSourceThroughLinkAndDotDot(t *testing.T) {
	t.Parallel()
	skipWithoutSymlinks(t)
	root := t.TempDir()
	left := filepath.Join(root, "left")
	right := filepath.Join(root, "right")
	require.NoError(t, os.MkdirAll(filepath.Join(right, "sub"), 0o755), "MkdirAll must not error")
	require.NoError(t, os.MkdirAll(left, 0o755), "MkdirAll must not error")
	require.NoError(t, os.WriteFile(filepath.Join(left, "payload"), []byte("the wrong one"), 0o600), "WriteFile must not error")
	require.NoError(t, os.WriteFile(filepath.Join(right, "payload"), []byte("the one asked for"), 0o600), "WriteFile must not error")
	require.NoError(t, os.Symlink(filepath.Join(right, "sub"), filepath.Join(left, "link")), "Symlink must not error")

	sep := string(filepath.Separator)
	src := filepath.Join(left, "link") + sep + ".." + sep + "payload"
	dest := filepath.Join(root, "out.zip")
	require.NoError(t, Zip(src, dest), "Zip must not error")

	o, err := UnZip(dest, filepath.Join(root, "extracted"))
	require.NoError(t, err, "UnZip must not error")
	require.Len(t, o, 1, "UnZip must extract the single file")
	b, err := os.ReadFile(o[0])
	require.NoError(t, err, "ReadFile must not error")
	assert.Equal(t, "the one asked for", string(b), "the archive should hold the file the kernel resolves the source to")
}

// A directory name need not be valid UTF-8 either, and reading one through Root.FS would put it
// through fs.ValidPath, failing the whole archive rather than recording it
func TestZipDirectoryWithAwkwardName(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := filepath.Join(root, "src")
	name := string([]byte{0xff, 0xfe}) + "-dir"
	if err := os.MkdirAll(filepath.Join(src, name), 0o755); err != nil {
		t.Skipf("filesystem will not hold a name that is not valid UTF-8: %v", err)
	}
	require.NoError(t, os.WriteFile(filepath.Join(src, name, "inner.txt"), []byte("payload"), 0o600), "WriteFile must not error")

	dest := filepath.Join(root, "out.zip")
	require.NoError(t, Zip(src, dest), "Zip must not error for a directory whose name is not valid UTF-8")
	o, err := UnZip(dest, filepath.Join(root, "extracted"))
	require.NoError(t, err, "UnZip must not error")
	require.Len(t, o, 1, "UnZip must extract the file beneath it")
	assert.Equal(t, name, filepath.Base(filepath.Dir(o[0])), "the directory should keep the name it was given")
}

// A filename need not be valid UTF-8 on Unix, and reaching a lone file through Root.FS would put
// it through fs.ValidPath, which rejects one that is not
func TestZipFileSourceWithAwkwardName(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "src")
	require.NoError(t, os.Mkdir(dir, 0o755), "Mkdir must not error")
	name := string([]byte{0xff, 0xfe}) + "-name.txt"
	src := filepath.Join(dir, name)
	if err := os.WriteFile(src, []byte("payload"), 0o600); err != nil {
		t.Skipf("filesystem will not hold a name that is not valid UTF-8: %v", err)
	}

	dest := filepath.Join(root, "out.zip")
	require.NoError(t, Zip(src, dest), "Zip must not error for a source whose name is not valid UTF-8")
	o, err := UnZip(dest, filepath.Join(root, "extracted"))
	require.NoError(t, err, "UnZip must not error")
	require.Len(t, o, 1, "UnZip must extract the single file")
	assert.Equal(t, name, filepath.Base(o[0]), "the entry should carry the name the file was given")
}

// Zip checks src before creating dest, so a rejected source touches nothing. Only a destination
// that cannot be created tells the two apart: without the check the create is reached and reports
// its own failure instead
func TestZipRejectsSourceBeforeCreatingDestination(t *testing.T) {
	t.Parallel()
	skipWithoutSymlinks(t)
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "target.txt"), []byte("payload"), 0o600), "WriteFile must not error")
	link := filepath.Join(root, "link.txt")
	require.NoError(t, os.Symlink("target.txt", link), "Symlink must not error")

	unwritable := filepath.Join(root, "unwritable")
	require.NoError(t, os.Mkdir(unwritable, 0o500), "Mkdir must not error")
	t.Cleanup(func() { assert.NoError(t, os.Chmod(unwritable, 0o700), "Chmod should not error") })

	err := Zip(link, filepath.Join(unwritable, "out.zip"))
	assert.ErrorIs(t, err, errUnsupportedFileType, "Zip should reject the source before it tries to create the destination")

	if os.Geteuid() == 0 {
		t.Skip("root can read a directory without read permission")
	}

	// and a source that cannot be rooted names the source, not the directory holding it. Search
	// permission is enough to stat and read the file; rooting its parent needs read as well
	searchOnly := filepath.Join(root, "searchonly")
	require.NoError(t, os.Mkdir(searchOnly, 0o755), "Mkdir must not error")
	inaccessible := filepath.Join(searchOnly, "a.txt")
	require.NoError(t, os.WriteFile(inaccessible, []byte("payload"), 0o600), "WriteFile must not error")
	require.NoError(t, os.Chmod(searchOnly, 0o111), "Chmod must not error")
	t.Cleanup(func() { assert.NoError(t, os.Chmod(searchOnly, 0o700), "Chmod should not error") })

	err = Zip(inaccessible, filepath.Join(root, "out.zip"))
	require.Error(t, err, "Zip must error when the source cannot be rooted")
	assert.ErrorContains(t, err, "rooting "+inaccessible, "the failure should name the source the caller passed")
}

// A fifo has to be refused before it is opened: opening one blocks until something writes to it, so
// a walk that reached the open would hang rather than error. Only Unix can create one
func TestZipRejectsFIFO(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pipe := filepath.Join(dir, "pipe")
	require.NoError(t, syscall.Mkfifo(pipe, 0o600), "Mkfifo must not error")

	t.Run("as a lone source", func(t *testing.T) {
		t.Parallel()
		err := addFilesToZipWrapper(zip.NewWriter(&bytes.Buffer{}), pipe, statSource(t, pipe), newDestInfo(t))
		assert.ErrorIs(t, err, errUnsupportedFileType, "addFilesToZipWrapper should reject a fifo named as the source")
	})

	// a directory replaced after it is stated is read, not opened for content, so the read has to
	// be non-blocking too
	t.Run("substituted for a directory before it is read", func(t *testing.T) {
		t.Parallel()
		sub := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(sub, "zzz"), 0o755), "Mkdir must not error")
		root, err := os.OpenRoot(sub)
		require.NoError(t, err, "OpenRoot must not error")
		t.Cleanup(func() { assert.NoError(t, root.Close(), "Close should not error") })
		require.NoError(t, os.Remove(filepath.Join(sub, "zzz")), "Remove must not error")
		require.NoError(t, syscall.Mkfifo(filepath.Join(sub, "zzz"), 0o600), "Mkfifo must not error")

		var f fs.File
		done := make(chan struct{})
		go func() { f, err = (rootFS{root}).Open("zzz"); close(done) }()
		select {
		case <-done:
			require.NoError(t, err, "Open must return without blocking on a fifo")
			require.NoError(t, f.Close(), "Close must not error")
		case <-time.After(3 * time.Second):
			require.FailNow(t, "Open must not block on a fifo")
		}
		_, err = fs.ReadDir(rootFS{root}, "zzz")
		assert.Error(t, err, "ReadDir should report a name that is no longer a directory")
	})

	// substituted after the walk stated the name, so only the open can catch it
	t.Run("substituted before the open", func(t *testing.T) {
		t.Parallel()
		sub := t.TempDir()
		file := filepath.Join(sub, "a.txt")
		require.NoError(t, os.WriteFile(file, []byte("payload"), 0o600), "WriteFile must not error")
		root, err := os.OpenRoot(sub)
		require.NoError(t, err, "OpenRoot must not error")
		t.Cleanup(func() { assert.NoError(t, root.Close(), "Close should not error") })

		e := newEntry(t, root, "a.txt", "a.txt")
		require.NoError(t, os.Remove(file), "Remove must not error")
		require.NoError(t, syscall.Mkfifo(file, 0o600), "Mkfifo must not error")

		out := newDestInfo(t)
		done := make(chan error, 1)
		go func() { done <- addFileToZip(zip.NewWriter(&bytes.Buffer{}), root, e, out) }()
		select {
		case err := <-done:
			assert.ErrorIs(t, err, errUnsupportedFileType, "addFileToZip should refuse a fifo rather than archive it")
		case <-time.After(3 * time.Second):
			t.Error("addFileToZip blocked opening a fifo rather than refusing it")
		}
	})
}

func TestAddFilesToZipWrapperUnreadableDirectory(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root can read a directory without read permission")
	}
	_, tree, _ := newSourceTree(t)
	sub := filepath.Join(tree, "sub")
	require.NoError(t, os.Chmod(sub, 0o111), "Chmod must not error")
	t.Cleanup(func() { assert.NoError(t, os.Chmod(sub, 0o700), "Chmod should not error") })
	err := addFilesToZipWrapper(zip.NewWriter(&bytes.Buffer{}), tree, statSource(t, tree), newDestInfo(t))
	assert.ErrorIs(t, err, fs.ErrPermission, "addFilesToZipWrapper should report a directory it cannot read")
}

func TestZipCleanupFailure(t *testing.T) {
	// Replaces the package level addFilesToZip, so it cannot be combined with t.Parallel
	orig := addFilesToZip
	t.Cleanup(func() { addFilesToZip = orig })
	expected := errors.New("archive failed")
	addFilesToZip = func(_ *zip.Writer, _ string, _ os.FileInfo, out output) error {
		// Unlinking the open output leaves cleanup with a missing path after both writers close
		require.NoError(t, os.Remove(out.path), "Remove must not error")
		return expected
	}
	root, tree, _ := newSourceTree(t)
	dest := filepath.Join(root, "out.zip")
	assert.ErrorIs(t, Zip(tree, dest), expected, "Zip should preserve the original error when cleanup fails")
	assert.NoFileExists(t, dest, "the removed output should remain absent")
}

// TestZipCleanupIdentifiesTheDestination covers the destination's parent being retargeted between
// the exclusive create and the failure, which would have Zip remove a file it never created
func TestZipCleanupIdentifiesTheDestination(t *testing.T) {
	// This replaces the package level addFilesToZip, so it cannot be combined with t.Parallel
	skipWithoutSymlinks(t)
	root := t.TempDir()
	_, tree, _ := newSourceTree(t)

	created, unrelated := filepath.Join(root, "created"), filepath.Join(root, "unrelated")
	require.NoError(t, os.Mkdir(created, 0o755), "Mkdir must not error")
	require.NoError(t, os.Mkdir(unrelated, 0o755), "Mkdir must not error")
	bystander := filepath.Join(unrelated, "out.zip")
	require.NoError(t, os.WriteFile(bystander, []byte("someone else's file"), 0o600), "WriteFile must not error")

	link := filepath.Join(root, "parent")
	require.NoError(t, os.Symlink(created, link), "Symlink must not error")
	dest := filepath.Join(link, "out.zip")

	orig := addFilesToZip
	t.Cleanup(func() { addFilesToZip = orig })
	expected := errors.New("archive failed")
	addFilesToZip = func(_ *zip.Writer, _ string, _ os.FileInfo, _ output) error {
		require.NoError(t, os.Remove(link), "Remove must not error")
		require.NoError(t, os.Symlink(unrelated, link), "Symlink must not error")
		return expected
	}

	assert.ErrorIs(t, Zip(tree, dest), expected, "Zip should preserve the original error")

	b, err := os.ReadFile(bystander)
	require.NoError(t, err, "ReadFile must not error")
	assert.Equal(t, "someone else's file", string(b), "a file the destination path came to name should not be removed")
	assert.FileExists(t, filepath.Join(created, "out.zip"), "the archive Zip created should be left for manual deletion rather than the wrong file removed")
}

func TestZipRejectsSourceReplacedWithFIFO(t *testing.T) {
	// Replaces the package level addFilesToZip, so it cannot be combined with t.Parallel
	src := filepath.Join(t.TempDir(), "src")
	require.NoError(t, os.Mkdir(src, 0o755), "Mkdir must not error")
	orig := addFilesToZip
	t.Cleanup(func() { addFilesToZip = orig })
	addFilesToZip = func(z *zip.Writer, src string, info os.FileInfo, out output) error {
		if err := os.Remove(src); err != nil {
			return err
		}
		if err := syscall.Mkfifo(src, 0o600); err != nil {
			return err
		}
		return addFilesToZipWrapper(z, src, info, out)
	}
	dest := filepath.Join(t.TempDir(), "out.zip")
	done := make(chan error, 1)
	go func() { done <- Zip(src, dest) }()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, syscall.ENOTDIR, "Zip should refuse a source replaced with a FIFO")
	case <-time.After(3 * time.Second):
		// Release a blocked reader before failing so a regression leaves no goroutine behind
		f, err := os.OpenFile(src, os.O_RDWR|syscall.O_NONBLOCK, 0)
		require.NoError(t, err, "OpenFile must unblock the FIFO reader")
		require.NoError(t, f.Close(), "Close must not error")
		<-done
		t.Fatal("Zip must not block opening a source replaced with a FIFO")
	}
	assert.NoFileExists(t, dest, "Zip should remove the rejected archive")
}
