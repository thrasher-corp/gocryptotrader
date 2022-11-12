package log

import (
	"errors"
	"fmt"
	"io"
	"log"
	"time"
)

var (
	errWriterAlreadyLoaded = errors.New("io.Writer already loaded")
	errJobsChannelIsFull   = errors.New("logger jobs channel is filled")
	errWriterIsNil         = errors.New("io writer is nil")
)

// loggerWorker handles all work staged to be written to configured io.Writer(s)
// This worker is generated in init() to handle full workload.
func loggerWorker() {
	// Localise a persistent buffer for a worker, this does not need to be
	// garbage collected.
	buffer := make([]byte, 0, defaultBufferCapacity)
	var n int
	var err error
	for j := range jobsChannel {
		if j.Passback != nil {
			j.Passback <- struct{}{}
			continue
		}
		data := j.fn()
		buffer = append(buffer, j.Header...)
		if j.ShowLogSystemName {
			buffer = append(buffer, j.Spacer...)
			buffer = append(buffer, j.SlName...)
		}
		buffer = append(buffer, j.Spacer...)
		if j.TimestampFormat != "" {
			buffer = time.Now().AppendFormat(buffer, j.TimestampFormat)
		}
		buffer = append(buffer, j.Spacer...)
		buffer = append(buffer, data...)
		if data == "" || data[len(data)-1] != '\n' {
			buffer = append(buffer, '\n')
		}

		for x := range j.Writers {
			n, err = j.Writers[x].Write(buffer)
			if err != nil {
				displayError(fmt.Errorf("%T %w", j.Writers[x], err))
			} else if n != len(buffer) {
				displayError(fmt.Errorf("%T %w", j.Writers[x], io.ErrShortWrite))
			}
		}
		buffer = buffer[:0] // Clean buffer
		jobsPool.Put(j)
	}
}

// deferral defines functionality that will capture data string processing and
// defer that to the worker pool if needed.
type deferral func() string

// StageLogEvent stages a new logger event in a jobs channel to be processed by
// a worker pool. This segregates the need to process the log string and the
// writes to the required io.Writer.
func (mw *multiWriterHolder) StageLogEvent(fn deferral, header, slName, spacer, timestampFormat string, showLogSystemName, bypassWarning bool) {
	newJob := jobsPool.Get().(*job) //nolint:forcetypeassert // Not necessary from a pool
	newJob.Writers = mw.writers
	newJob.fn = fn
	newJob.Header = header
	newJob.SlName = slName
	newJob.ShowLogSystemName = showLogSystemName
	newJob.Spacer = spacer
	newJob.TimestampFormat = timestampFormat

	select {
	case jobsChannel <- newJob:
	default:
		// This will cause temporary caller impedance, which can have a knock
		// on effect in processing.
		if !bypassWarning {
			log.Printf("Logger warning: %v\n", errJobsChannelIsFull)
		}
		jobsChannel <- newJob
	}
}

// multiWriter make and return a new copy of multiWriterHolder
func multiWriter(writers ...io.Writer) (*multiWriterHolder, error) {
	mw := &multiWriterHolder{}
	for x := range writers {
		err := mw.add(writers[x])
		if err != nil {
			return nil, err
		}
	}
	return mw, nil
}

// Add appends a new writer to the multiwriter slice
func (mw *multiWriterHolder) add(writer io.Writer) error {
	if writer == nil {
		return errWriterIsNil
	}
	for i := range mw.writers {
		if mw.writers[i] == writer {
			return errWriterAlreadyLoaded
		}
	}
	mw.writers = append(mw.writers, writer)
	return nil
}
