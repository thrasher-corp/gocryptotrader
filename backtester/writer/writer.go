package writer

import (
	"errors"
	"fmt"
	"strings"
)

const wStr = "%s-%s"

var errIDNotSet = errors.New("id not set")

// Writer is a custom writer for the backtester which parses log.MultiWriter logs
// and filters responses by a runID. This allows simultaneous strategy executions
// do not write logs
type Writer struct {
	runID    string
	logs     []string
	isActive bool
}

// SetupWriter returns a writer to store logs
func SetupWriter(id string) (*Writer, error) {
	if id == "" {
		return nil, errIDNotSet
	}
	return &Writer{
		runID:    id,
		isActive: true,
	}, nil
}

// DeActivate prevents any new logs being written to the writer
func (w *Writer) DeActivate() {
	w.isActive = false
}

// Write writes logs to the logger
func (w *Writer) Write(p []byte) (n int, err error) {
	if !w.isActive {
		return 0, nil
	}
	if len(p) == 0 {
		return 0, nil
	}
	str := fmt.Sprintf(wStr, w.runID, p)
	w.logs = append(w.logs, str)
	return len(p), nil
}

// String returns the accumulated string.
func (w *Writer) String() string {
	var resp string
	for i := range w.logs {
		split := strings.Split(w.logs[i], w.runID+"-")
		if len(split) == 1 {
			continue
		}
		resp += split[1]
	}
	return resp
}
