package archive

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// ErrUnableToCloseFile message to display when file handler is unable to be closed normally
	ErrUnableToCloseFile string = "Unable to close file %v %v"
)

var (
	// errDestinationWithinSource is returned when the walk opens the archive being written, which
	// would otherwise add a partial copy of the output to its own output
	errDestinationWithinSource = errors.New("archive destination is within the source directory")
	// errEntryReplaced is returned when what was opened is not what the walk found, which is the
	// window the walk cannot close by looking at the path again
	errEntryReplaced = errors.New("entry replaced during the walk")
	// errUnnameableSource is returned for a source that cannot name an entry, every entry being
	// recorded beneath the source's own name
	errUnnameableSource = errors.New("source cannot name an archive entry")
	// errUnsupportedFileType is returned rather than archiving anything but a regular file or a
	// directory. UnZip does not recreate special file types, so nothing else survives a round
	// trip, and the entry a symlink produced carried the link's mode over the target's bytes,
	// which a conformant extractor restores as a link whose target is those bytes
	errUnsupportedFileType = errors.New("unsupported file type")
)

// entry is one file the walk found: the path to open it by, the name it takes in the archive, and
// what the walk's Lstat reported, which the file opened later is then checked against
type entry struct {
	path string
	name string
	info os.FileInfo
}

// output identifies the archive being written: the info the walk recognises it by, and the path
// the caller passed, so a refusal names something the caller can place
type output struct {
	path string
	info os.FileInfo
}

var addFilesToZip = addFilesToZipWrapper

// UnZip extracts input zip into dest path
func UnZip(src, dest string) (fileList []string, err error) {
	z, err := zip.OpenReader(src)
	if err != nil {
		return fileList, err
	}

	for x := range z.File {
		fPath := filepath.Join(dest, z.File[x].Name) //nolint // We ignore
		// gosec linter above because the code below files the file traversal
		// bug when extracting archives
		if !strings.HasPrefix(fPath, filepath.Clean(dest)+string(os.PathSeparator)) {
			err = z.Close()
			if err != nil {
				log.Errorf(log.Global, ErrUnableToCloseFile, z, err)
			}
			err = fmt.Errorf("%s: illegal file path", fPath)
			return fileList, err
		}

		if z.File[x].FileInfo().IsDir() {
			err = os.MkdirAll(fPath, os.ModePerm)
			if err != nil {
				return fileList, err
			}
			continue
		}

		err = os.MkdirAll(filepath.Dir(fPath), file.DefaultPermissionOctal)
		if err != nil {
			return fileList, err
		}

		var outFile *os.File
		outFile, err = os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, z.File[x].Mode())
		if err != nil {
			return fileList, err
		}

		var eFile io.ReadCloser
		eFile, err = z.File[x].Open()
		if err != nil {
			errCls := outFile.Close()
			if errCls != nil {
				log.Errorf(log.Global, ErrUnableToCloseFile, outFile, errCls)
			}
			return fileList, err
		}

		_, errIOCopy := io.Copy(outFile, eFile)
		if errIOCopy != nil {
			err = z.Close()
			if err != nil {
				log.Errorf(log.Global, ErrUnableToCloseFile, z, err)
			}
			err = outFile.Close()
			if err != nil {
				log.Errorf(log.Global, ErrUnableToCloseFile, outFile, err)
			}
			err = eFile.Close()
			if err != nil {
				log.Errorf(log.Global, ErrUnableToCloseFile, eFile, err)
			}
			return fileList, errIOCopy
		}
		err = outFile.Close()
		if err != nil {
			log.Errorf(log.Global, ErrUnableToCloseFile, outFile, err)
		}
		err = eFile.Close()
		if err != nil {
			log.Errorf(log.Global, ErrUnableToCloseFile, eFile, err)
		}
		if err != nil {
			return fileList, err
		}

		fileList = append(fileList, fPath)
	}
	return fileList, z.Close()
}

// Zip archives requested file or folder. Only regular files and directories are archived: a
// symlink is rejected as src, as dest, and as any entry under a directory src, rather than being
// followed. Symlinks among the parent directories of either path are still resolved, as is a src
// named with a trailing separator, which names the directory the link points at. dest may not be
// src, nor sit inside a directory src, nor already exist. Those hold for a filesystem that is not
// changing underneath the archive. A file whose kind changes while the walk runs is refused, as is
// one replaced between being stated and being opened, because both are settled against the handle
// the copy reads from. A directory has no such handle to settle against, only the one it is read
// through, so one swapped for another after it is stated is walked as whatever it now refers to,
// and a name that becomes a link to the file it already named is archived, since its type and its
// identity both still hold. The opened source must still match its initial identity. Rooting a
// lone file needs read permission on its parent directory, which is more than reading the file
// requires
func Zip(src, dest string) error {
	i, err := lstatSource(src)
	if err != nil {
		return err
	}

	// before dest is created, so a rejected src leaves nothing behind
	if err := checkFileType(i.Mode(), src); err != nil {
		return err
	}

	// Exclusive because no check on a pathname can say what it resolves to: a hard link, or a ".."
	// that resolves through a symlink, reaches a file inside src under a name nothing recognises,
	// and truncating it would destroy it. Refusing to open an existing file is the one answer the
	// filesystem gives atomically on every platform. 0o666 is the mode os.Create used
	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		return err
	}

	z := zip.NewWriter(f)

	// the walk compares the files it opens against this rather than against dest's path, so no
	// spelling of that path can hide the output from the walk that is writing it
	destInfo, err := f.Stat()
	if err == nil {
		err = addFilesToZip(z, src, i, output{path: dest, info: destInfo})
	}
	// zip.Writer.Close writes the central directory, so discarding its error would report success
	// over a corrupt archive
	if closeErr := z.Close(); err == nil {
		err = closeErr
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		return nil
	}

	// Identified against the handle it was created on, since a pathname says nothing about what it
	// resolves to now: a retargeted parent would have Zip delete a file it never made
	i, statErr := os.Lstat(dest)
	switch {
	case destInfo == nil:
		log.Errorf(log.Global, "Failed to remove archive, manual deletion required: %s could not be identified when it was created", dest)
	case statErr != nil:
		log.Errorf(log.Global, "Failed to remove archive, manual deletion required: %v", statErr)
	case !os.SameFile(destInfo, i):
		log.Errorf(log.Global, "Failed to remove archive, manual deletion required: %s no longer names the file that was created", dest)
	default:
		if errRemove := os.Remove(dest); errRemove != nil {
			log.Errorf(log.Global, "Failed to remove archive, manual deletion required: %v", errRemove)
		}
	}
	return err
}

// rootFS avoids Root.FS's fs.ValidPath check, which rejects filesystem names that are not UTF-8.
// fs.ReadDir supplies sorted entries through Open
type rootFS struct{ root *os.Root }

// Open is non-blocking for the same reason the entry open is: a name stated as a directory can be a
// fifo by the time it is opened, and a blocking open would wait for a writer that never comes
func (r rootFS) Open(name string) (fs.File, error) {
	return r.root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}

// checkFileType rejects anything that is not a regular file or a directory. The type is tested
// rather than the symlink bit alone because Windows suppresses ModeDir for a reparse point that
// names another entity, reporting a junction as irregular rather than as a symlink. Every reparse
// point but a symlink, a socket or a deduplicated file is irregular too, so a cloud placeholder
// file is refused rather than downloaded, while one on a directory keeps its ModeDir and is
// walked like any other
func checkFileType(mode fs.FileMode, path string) error {
	if !mode.IsRegular() && !mode.IsDir() {
		return fmt.Errorf("%w %s: %s", errUnsupportedFileType, mode, path)
	}
	return nil
}

// archiveName is the name the source takes at the top of the archive. ".." is resolved against the
// working directory because it cannot name an entry: a source spelled with a trailing ".." would
// otherwise be recorded as "../", which a conformant extractor refuses, UnZip among them. "." is
// left alone, filepath.Join dropping it, so a source named that keeps the entries it always had
func archiveName(src string) (string, error) {
	name := filepath.Base(filepath.Clean(src))
	if name == ".." {
		abs, err := filepath.Abs(src)
		if err != nil {
			return "", err
		}
		name = filepath.Base(abs)
	}
	if name == ".." || name == string(filepath.Separator) {
		return "", fmt.Errorf("%w: %s", errUnnameableSource, src)
	}
	return name, nil
}

func addFilesToZipWrapper(z *zip.Writer, src string, srcInfo os.FileInfo, out output) error {
	rootDir := src
	if !srcInfo.IsDir() {
		// Split preserves a symlink followed by ".."; Dir would clean it before the kernel resolves it
		rootDir, _ = filepath.Split(src)
		if rootDir == "" {
			rootDir = "."
		}
	}
	// On Unix a trailing slash requires directory resolution, so OpenRoot cannot block opening
	// a FIFO substituted for the source. Keep Windows drive-relative paths unchanged
	if filepath.Separator == '/' {
		rootDir += "/"
	}
	// Rooted so a mid-walk symlink swap cannot reach outside the tree being archived
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return fmt.Errorf("rooting %s: %w", src, err)
	}
	defer root.Close()

	if !srcInfo.IsDir() {
		// a lone file is the one the caller named rather than a name the walk found, so it is
		// reached through the root directly. Root.FS would take it through fs.ValidPath, which
		// rejects a name that is not valid UTF-8, and no directory has to be enumerated to find it
		base := filepath.Base(src)
		i, err := root.Lstat(base)
		if err != nil {
			return err
		}
		if err := checkFileType(i.Mode(), src); err != nil {
			return err
		}
		return addFileToZip(z, root, entry{path: base, name: i.Name(), info: srcInfo}, out)
	}

	srcName, err := archiveName(src)
	if err != nil {
		return err
	}

	return fs.WalkDir(rootFS{root}, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// what reading the directory reported is too old to decide on the type: an entry replaced
		// by a symlink would be followed, and one replaced by a fifo would block the open forever
		i, err := root.Lstat(path)
		if err != nil {
			return err
		}

		full := filepath.Join(root.Name(), path)
		if err := checkFileType(i.Mode(), full); err != nil {
			return err
		}

		// Check the opened root against the source Zip stated, and the entry kind against the
		// cached DirEntry WalkDir uses to decide whether to descend
		if path == "." && !os.SameFile(srcInfo, i) || d.IsDir() != i.IsDir() {
			return fmt.Errorf("%w: %s", errEntryReplaced, full)
		}

		// ToSlash because zip.FileHeader.Name must use forward slashes on every platform
		name := filepath.ToSlash(filepath.Join(srcName, path))

		if !i.IsDir() {
			return addFileToZip(z, root, entry{path: path, name: name, info: i}, out)
		}

		// UnZip refuses the "./" a "." source would otherwise record for its own root
		if name == "." {
			return nil
		}

		h, err := zip.FileInfoHeader(i)
		if err == nil {
			h.Name = name + "/"
			_, err = z.CreateHeader(h)
		}
		return err
	})
}

// addFileToZip writes one file entry. The file is opened before its header is written so that the
// entry describes what is actually read, and so that the checks below run against the handle the
// copy uses rather than against a pathname, which says nothing about what a later open would find
func addFileToZip(z *zip.Writer, root *os.Root, e entry, out output) (err error) {
	// named the way the caller spelled the source, since this is what a failure reports
	full := filepath.Join(root.Name(), e.path)

	// non-blocking because the type is settled on the handle below: a fifo substituted since the
	// walk stated this name would otherwise block the open until someone wrote to it
	f, err := root.OpenFile(e.path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return fmt.Errorf("opening %s: %w", full, err)
	}
	defer func() {
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
	}()

	i, err := f.Stat()
	// again, because the walk checked a path rather than this handle, and the header below is
	// built from it
	if err == nil && !i.Mode().IsRegular() {
		err = fmt.Errorf("%w %s: %s", errUnsupportedFileType, i.Mode(), full)
	}
	if err != nil {
		return err
	}
	// a replacement since the walk stated this name would otherwise be archived under it
	if !os.SameFile(e.info, i) {
		return fmt.Errorf("%w: %s", errEntryReplaced, full)
	}
	if os.SameFile(out.info, i) {
		return fmt.Errorf("%w: %s", errDestinationWithinSource, out.path)
	}

	var w io.Writer
	h, err := zip.FileInfoHeader(i)
	if err == nil {
		h.Name, h.Method = e.name, zip.Deflate
		w, err = z.CreateHeader(h)
	}
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("copying %s: %w", full, err)
	}
	return nil
}
