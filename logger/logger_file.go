package logger

import (
	"fmt"
	"io"
	"os"
)

type fileWriters struct {
	fileHandle []os.File
}

var f = fileWriters{}

func createFileHandle(subsystem string) (io.Writer, error) {
	fileHandle, err := os.OpenFile(subsystem+".txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	addFileWriter(fileHandle)
	return fileHandle, nil
}

func addFileWriter(fileHandle *os.File) {
	f.fileHandle = append(f.fileHandle, *fileHandle)
}

func closeAllFiles() error {
	if len(f.fileHandle) > 0 {
		for x := range f.fileHandle {
			err := f.fileHandle[x].Close()
			if err != nil {
				return fmt.Errorf("failed to close file: %v", f.fileHandle[x].Name())
			}
		}
	}
	return nil
}
