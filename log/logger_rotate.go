package log

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/file"
)

var (
	errExceedsMaxFileSize = errors.New("exceeds max file size")
	errFileNameIsEmpty    = errors.New("filename is empty")
)

// Write implementation to satisfy io.Writer handles length check and rotation
func (r *Rotate) Write(output []byte) (n int, err error) {
	outputLen := int64(len(output))
	if outputLen > r.maxSize() {
		return 0, fmt.Errorf(
			"write length %v %w %v", outputLen, errExceedsMaxFileSize, r.maxSize(),
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
	logFile := filepath.Join(GetLogPath(), r.FileName)
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

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return r.openNew()
	}

	r.output = f
	r.size = info.Size()

	return nil
}

func (r *Rotate) openNew() error {
	if r.FileName == "" {
		return fmt.Errorf("cannot open new file: %w", errFileNameIsEmpty)
	}
	name := filepath.Join(GetLogPath(), r.FileName)
	_, err := os.Stat(name)

	if err == nil {
		timestamp := time.Now().Format("2006-01-02T15-04-05")
		newName := filepath.Join(GetLogPath(), timestamp+"-"+r.FileName)

		err = file.Move(name, newName)
		if err != nil {
			return fmt.Errorf("can't rename log file: %s", err)
		}
	}

	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("can't open new logfile: %s", err)
	}

	r.output = f
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
	return r.close()
}

func (r *Rotate) rotateFile() (err error) {
	err = r.close()
	if err != nil {
		return
	}
	return r.openNew()
}

func (r *Rotate) maxSize() int64 {
	if r.MaxSize == 0 {
		return defaultMaxSize * megabyte
	}
	return r.MaxSize * megabyte
}
