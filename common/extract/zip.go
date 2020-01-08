package extract

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Unzip extracts input zip into dest path
func Unzip(src, dest string) error {
	z, err := zip.OpenReader(src)
	if err != nil {
		return err
	}

	for x := range z.File {
		fPath := filepath.Join(dest, z.File[x].Name) // nolint:gosec
		// We ignore gosec linter above because the code below files the file traversal bug when extracting archives
		if !strings.HasPrefix(fPath, filepath.Clean(dest)+string(os.PathSeparator)) {
			err = z.Close()
			if err != nil {
				log.Errorf(log.Global, ErrUnableToCloseFile, z, err)
			}
			return fmt.Errorf("%s: illegal file path", fPath)
		}

		if z.File[x].FileInfo().IsDir() {
			err = os.MkdirAll(fPath, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}

		err = os.MkdirAll(filepath.Dir(fPath), 0770)
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, z.File[x].Mode())
		if err != nil {
			return err
		}

		eFile, err := z.File[x].Open()
		if err != nil {
			err = outFile.Close()
			if err != nil {
				log.Errorf(log.Global, ErrUnableToCloseFile, outFile, err)
			}
			return err
		}

		_, err = io.Copy(outFile, eFile)
		if err != nil {
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
			return err
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
			return err
		}
	}
	return z.Close()
}
