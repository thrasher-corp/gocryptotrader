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

	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// ErrUnableToCloseFile message to display when file handler is unable to be closed normally
	ErrUnableToCloseFile string = "Unable to close file %v %v"
)

var (
	// errDestinationIsSource is returned rather than letting the dest file be created over src,
	// which would truncate the source before the walk reads it
	errDestinationIsSource = errors.New("archive destination is the source")
	// errDestinationWithinSource is returned for a dest under a directory src, where the archive
	// would either truncate an entry it is about to read or add itself to its own output
	errDestinationWithinSource = errors.New("archive destination is within the source directory")
	// errTooManySymlinks is returned for a destination symlink chain longer than the kernel follows
	errTooManySymlinks = errors.New("too many levels of symbolic links")
)

// maxSymlinkHops matches the chain length Linux follows before returning ELOOP
const maxSymlinkHops = 40

var addFilesToZip func(z *zip.Writer, src string, isDir bool) error

func init() {
	addFilesToZip = addFilesToZipWrapper
}

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

// Zip archives requested file or folder. The walk is rooted at src, so any symlink it cannot
// resolve inside src fails the whole archive rather than being skipped: one pointing outside, a
// dangling one, or any absolute symlink even where the target is inside. One resolving to a file is
// stored under the link's own mode bits but carries the target's bytes, while one resolving to a
// directory is recorded as an empty entry, its contents archived separately by the walk. A
// directory src is resolved before the root is established, so one symlinked to a directory
// outside archives that directory, but a file src is rooted at its parent and so is checked like
// any other entry. dest may not be src nor sit inside a directory src.
func Zip(src, dest string) error {
	i, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := checkDestination(src, dest, i); err != nil {
		return err
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}

	z := zip.NewWriter(f)

	err = addFilesToZip(z, src, i.IsDir())
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

	if errRemove := os.Remove(dest); errRemove != nil {
		log.Errorf(log.Global, "Failed to remove archive, manual deletion required: %v", errRemove)
	}
	return err
}

// checkDestination rejects a dest that would destroy or include the source
func checkDestination(src, dest string, srcInfo os.FileInfo) error {
	switch di, err := os.Stat(dest); {
	case err == nil:
		if os.SameFile(srcInfo, di) {
			return fmt.Errorf("%w: %s", errDestinationIsSource, dest)
		}
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}

	if !srcInfo.IsDir() {
		return nil
	}

	// Absolute before resolved: EvalSymlinks leaves a relative path relative, so the working
	// directory prefix stays unresolved and the two sides stop being comparable whenever the
	// working directory, or any parent of it, is reached through a symlink
	srcResolved, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	srcResolved, err = filepath.EvalSymlinks(srcResolved)
	if err != nil {
		return err
	}
	destResolved, err := destWriteTarget(dest)
	if err != nil {
		// no parent directory means os.Create will fail before it can write anything
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	// paths that cannot be made relative are on different volumes, so dest is not inside src
	if rel, relErr := filepath.Rel(srcResolved, destResolved); relErr == nil &&
		(rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))) {
		return fmt.Errorf("%w: %s", errDestinationWithinSource, dest)
	}
	return nil
}

// destWriteTarget returns the path os.Create(dest) would write to, with every symlink it would
// follow resolved. EvalSymlinks alone cannot answer this: it errors on a path that does not exist
// yet, which a destination about to be created never does, and on a symlink pointing at one, which
// os.Create follows and creates at the far end
func destWriteTarget(dest string) (string, error) {
	target, err := filepath.Abs(dest)
	if err != nil {
		return "", err
	}
	// one iteration per link, plus the one that finds the non-link at the end of the chain
	for range maxSymlinkHops + 1 {
		link, err := os.Readlink(target)
		if err != nil {
			// the final element is the one os.Create makes, so only its directory can be resolved
			dir, err := filepath.EvalSymlinks(filepath.Dir(target))
			if err != nil {
				return "", err
			}
			return filepath.Join(dir, filepath.Base(target)), nil
		}
		if !filepath.IsAbs(link) {
			link = filepath.Join(filepath.Dir(target), link)
		}
		target = link
	}
	return "", fmt.Errorf("%w: %s", errTooManySymlinks, dest)
}

func addFilesToZipWrapper(z *zip.Writer, src string, isDir bool) error {
	// Rooted so a mid-walk symlink swap cannot pull in an unrelated file; OpenRoot needs a
	// directory, so a lone file is rooted at its parent
	rootDir, start := src, "."
	if !isDir {
		rootDir, start = filepath.Split(src)
		if rootDir == "" {
			rootDir = "."
		}
	}

	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return err
	}
	defer root.Close()

	return fs.WalkDir(root.FS(), start, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		i, err := d.Info()
		if err != nil {
			return err
		}

		h, err := zip.FileInfoHeader(i)
		if err != nil {
			return err
		}

		if isDir {
			// ToSlash because zip.FileHeader.Name must use forward slashes on every platform
			h.Name = filepath.ToSlash(filepath.Join(filepath.Base(src), path))
		}

		if i.IsDir() {
			h.Name += "/"
		} else {
			h.Method = zip.Deflate
		}

		w, err := z.CreateHeader(h)
		if err != nil {
			return err
		}

		if i.IsDir() {
			return nil
		}

		f, err := root.Open(path)
		if err != nil {
			return err
		}
		fi, err := f.Stat()
		if err == nil && !fi.IsDir() {
			// a symlink resolving to a directory inside the root has no contents to copy, so the
			// entry records the link alone rather than failing the archive
			_, err = io.Copy(w, f)
		}
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return fmt.Errorf("copying %s: %w", path, err)
		}

		return nil
	})
}
