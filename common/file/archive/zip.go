package archive

import (
	"archive/zip"
	"fmt"
	"io"
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

var addFilesToZip func(z *zip.Writer, src string, isDir bool) error

func init() {
	addFilesToZip = addFilesToZipWrapper
}

// UnZip extracts input zip into dest path
func UnZip(src, dest string) (fileList []string, err error) {
	z, err := zip.OpenReader(src)
	if err != nil {
		return
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
			return
		}

		if z.File[x].FileInfo().IsDir() {
			err = os.MkdirAll(fPath, os.ModePerm)
			if err != nil {
				return
			}
			continue
		}

		err = os.MkdirAll(filepath.Dir(fPath), file.DefaultPermissionOctal)
		if err != nil {
			return
		}

		var outFile *os.File
		outFile, err = os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, z.File[x].Mode())
		if err != nil {
			return
		}

		var eFile io.ReadCloser
		eFile, err = z.File[x].Open()
		if err != nil {
			errCls := outFile.Close()
			if errCls != nil {
				log.Errorf(log.Global, ErrUnableToCloseFile, outFile, errCls)
			}
			return
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
			return
		}

		fileList = append(fileList, fPath)
	}
	return fileList, z.Close()
}

// Zip archives requested file or folder
func Zip(src, dest string) error {
	i, err := os.Stat(src)
	if err != nil {
		return err
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}

	z := zip.NewWriter(f)

	err = addFilesToZip(z, src, i.IsDir())
	if err != nil {
		z.Close()
		errCls := f.Close()
		if errCls != nil {
			log.Errorf(log.Global, "Failed to close file handle, manual deletion required: %v", errCls)
			return err
		}
		errRemove := os.Remove(dest)
		if errRemove != nil {
			log.Errorf(log.Global, "Failed to remove archive, manual deletion required: %v", errRemove)
		}
		return err
	}

	z.Close()
	f.Close()
	return nil
}

func addFilesToZipWrapper(z *zip.Writer, src string, isDir bool) error {
	return filepath.Walk(src, func(path string, i os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		h, err := zip.FileInfoHeader(i)
		if err != nil {
			return err
		}

		if isDir {
			h.Name = filepath.Join(filepath.Base(src), strings.TrimPrefix(path, src))
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

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, f)
		if err != nil {
			log.Errorf(log.Global, "Failed to Copy data: %v", err)
		}

		return f.Close()
	})
}
