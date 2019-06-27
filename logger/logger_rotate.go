package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	defaultMaxSize = 100
	megabyte       = 1024 * 1024
)

type Rotate struct {
	Filename string
	MaxSize  int

	Compress bool

	size   int64
	output *os.File
	mu     sync.Mutex
}

func (r *Rotate) Write(output []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	writeLen := int64(len(output))

	if writeLen > r.max() {
		return 0, fmt.Errorf(
			"write length %v exceeds max file size %v", writeLen, r.max(),
		)
	}

	if r.output == nil {
		err = r.openOrCreateFile(writeLen)
		if err != nil {
			return 0, err
		}
	}

	if r.size+writeLen > r.max() {
		err = r.rotate()
		if err != nil {
			return 0, err
		}
	}

	n, err = r.output.Write(output)
	r.size += int64(n)

	return n, err
}

func (r *Rotate) openOrCreateFile(n int64) error {

	logFile := filepath.Join(LogPath, r.Filename)

	info, err := os.Stat(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return r.openNew()
		}
		return fmt.Errorf("error opening log file info: %s", err)
	}

	if info.Size()+n >= r.max() {
		return r.rotate()
	}

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return r.openNew()
	}

	r.output = file
	r.size = info.Size()

	return nil
}

func (r *Rotate) openNew() error {
	name := filepath.Join(LogPath, r.Filename)
	_, err := os.Stat(name)

	t := time.Now()
	timestamp := t.Format("2006-01-02T15-04-05.000")
	newName := filepath.Join(LogPath, r.Filename, timestamp)
	if err == nil {
		err = os.Rename(name, newName)
		if err != nil {
			return fmt.Errorf("can't rename log file: %s", err)
		}
	}

	file, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("can't open new logfile: %s", err)
	}

	r.output = file
	r.size = 0

	return nil
}

func (r *Rotate) close() (err error) {
	if r.output == nil {
		return nil
	}
	err = r.output.Close()
	r.output = nil
	return err
}

func (r *Rotate) rotate() (err error) {
	err = r.close()
	if err != nil {
		return
	}

	err = r.openNew()
	if err != nil {
		return
	}
	return nil
}

func (r *Rotate) max() int64 {
	if r.MaxSize == 0 {
		return int64(defaultMaxSize * megabyte)
	}
	return int64(r.MaxSize) * int64(megabyte)
}
