package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type fileWriters struct {
	fileHandle []os.File
}

var f = fileWriters{}

func createFileHandle(logger string, global bool) (io.Writer, error) {
	var logFile string
	if global {
		logFile = filepath.Join(LogPath, "log.txt")
	} else {
		logFile = filepath.Join(LogPath, logger+".txt")
	}
	fileHandle, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
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
