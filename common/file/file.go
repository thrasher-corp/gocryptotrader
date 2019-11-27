package file

import (
	"io"
	"io/ioutil"
	"os"
)

// Write writes selected data to a file or returns an error if it fails. This
// func also ensures that all files are set to this permission (only rw access
// for the running user and the group the user is a member of)
func Write(file string, data []byte) error {
	return ioutil.WriteFile(file, data, 0770)
}

// Move moves a file from a source path to a destination path
// This must be used across the codebase for compatibility with Docker volumes
// and Golang (fixes Invalid cross-device link when using os.Rename)
func Move(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}

	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return err
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return err
	}

	return os.Remove(sourcePath)
}
