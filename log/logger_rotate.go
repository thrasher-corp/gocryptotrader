package log

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/file"
)

// Write implementation to satisfy io.Writer handles length check and rotation
func (r *Rotate) Write(output []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	outputLen := int64(len(output))

	if outputLen > r.maxSize() {
		return 0, fmt.Errorf(
			"write length %v exceeds max file size %v", outputLen, r.maxSize(),
		)
	}

	if r.output == nil {
		err = r.openOrCreateFile(outputLen)
		if err != nil {
			return 0, err
		}
	}

	if *r.Rotate {
		if r.size+outputLen > r.maxSize() {
			err = r.rotateFile()
			if err != nil {
				return 0, err
			}
		}
	}

	n, err = r.output.Write(output)
	r.size += int64(n)

	return n, err
}

func (r *Rotate) openOrCreateFile(n int64) error {
	logFile := filepath.Join(LogPath, r.FileName)

	info, err := os.Stat(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return r.openNew()
		}
		return fmt.Errorf("error opening log file info: %s", err)
	}

	if *r.Rotate {
		if info.Size()+n >= r.maxSize() {
			return r.rotateFile()
		}
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
	name := filepath.Join(LogPath, r.FileName)
	_, err := os.Stat(name)

	if err == nil {
		timestamp := time.Now().Format("2006-01-02T15-04-05")
		newName := filepath.Join(LogPath, timestamp+"-"+r.FileName)

		err = file.Move(name, newName)
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

// Close handler for open file
func (r *Rotate) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.close()
}

func (r *Rotate) rotateFile() (err error) {
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

func (r *Rotate) maxSize() int64 {
	if r.MaxSize == 0 {
		return int64(defaultMaxSize * megabyte)
	}
	return r.MaxSize * int64(megabyte)
}
