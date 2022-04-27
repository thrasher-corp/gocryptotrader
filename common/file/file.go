package file

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// DefaultPermissionOctal is the default file and folder permission octal used throughout GCT
const DefaultPermissionOctal fs.FileMode = 0o770

// Write writes selected data to a file or returns an error if it fails. This
// func also ensures that all files are set to this permission (only rw access
// for the running user and the group the user is a member of)
func Write(file string, data []byte) error {
	basePath := filepath.Dir(file)
	if !Exists(basePath) {
		if err := os.MkdirAll(basePath, DefaultPermissionOctal); err != nil {
			return err
		}
	}
	return os.WriteFile(file, data, DefaultPermissionOctal)
}

// Writer creates a writer to a file or returns an error if it fails. This
// func also ensures that all files are set to this permission (only rw access
// for the running user and the group the user is a member of)
func Writer(file string) (*os.File, error) {
	basePath := filepath.Dir(file)
	if !Exists(basePath) {
		if err := os.MkdirAll(basePath, DefaultPermissionOctal); err != nil {
			return nil, err
		}
	}
	return os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, DefaultPermissionOctal)
}

// Move moves a file from a source path to a destination path
// This must be used across the codebase for compatibility with Docker volumes
// and Golang (fixes Invalid cross-device link when using os.Rename)
func Move(sourcePath, destPath string) error {
	sourceAbs, err := filepath.Abs(sourcePath)
	if err != nil {
		return err
	}
	destAbs, err := filepath.Abs(destPath)
	if err != nil {
		return err
	}
	if sourceAbs == destAbs {
		return nil
	}
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}

	destDir := filepath.Dir(destPath)
	if !Exists(destDir) {
		err = os.MkdirAll(destDir, DefaultPermissionOctal)
		if err != nil {
			return err
		}
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return err
	}

	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	outputFile.Close()
	if err != nil {
		if errRem := os.Remove(destPath); errRem != nil {
			return fmt.Errorf(
				"unable to os.Remove error: %s after io.Copy error: %s",
				errRem,
				err,
			)
		}
		return err
	}

	return os.Remove(sourcePath)
}

// Exists returns whether or not a file or path exists
func Exists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

// WriteAsCSV takes a table of records and writes it as CSV
func WriteAsCSV(filename string, records [][]string) error {
	if len(records) == 0 {
		return errors.New("no records in matrix")
	}

	buf := bytes.Buffer{}
	w := csv.NewWriter(&buf)

	alignment := len(records[0])
	for i := range records {
		if len(records[i]) != alignment {
			return errors.New("incorrect alignment")
		}

		err := w.Write(records[i])
		if err != nil {
			return err
		}
	}

	w.Flush()

	if err := w.Error(); err != nil {
		return err
	}
	return Write(filename, buf.Bytes())
}
