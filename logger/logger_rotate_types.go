package logger

import (
	"os"
	"sync"
)

const (
	defaultMaxSize = 250
	megabyte       = 1024 * 1024
)

type Rotate struct {
	FileName string
	Rotate   *bool
	MaxSize  int64

	size   int64
	output *os.File
	mu     sync.Mutex
}
