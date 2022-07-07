package log

import (
	"errors"
	"fmt"
	"io"
)

var (
	errWriterAlreadyLoaded = errors.New("io.Writer already loaded")
	errWriterNotFound      = errors.New("io.Writer not found")
)

// Add appends a new writer to the multiwriter slice
func (mw *multiWriterHolder) Add(writer io.Writer) error {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	for i := range mw.writers {
		if mw.writers[i] == writer {
			return errWriterAlreadyLoaded
		}
	}
	mw.writers = append(mw.writers, writer)
	return nil
}

// Remove removes existing writer from multiwriter slice
func (mw *multiWriterHolder) Remove(writer io.Writer) error {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	for i := range mw.writers {
		if mw.writers[i] != writer {
			continue
		}
		mw.writers[i] = mw.writers[len(mw.writers)-1]
		mw.writers[len(mw.writers)-1] = nil
		mw.writers = mw.writers[:len(mw.writers)-1]
		return nil
	}
	return errWriterNotFound
}

func loggerWorker() {
	var n int
	var err error
	for j := range jobChannel {
		n, err = j.Writer.Write(j.Data)
		if err != nil {
			displayError(fmt.Errorf("%T %w", j.Writer, err))
		} else if n != len(j.Data) {
			fmt.Println("WOW")
			displayError(fmt.Errorf("%T %w", j.Writer, io.ErrShortWrite))
		}
		jobsPool.Put(j)
	}
}

// Write concurrent safe Write for each writer
func (mw *multiWriterHolder) Write(p []byte) (int, error) {
	mw.mu.RLock()
	defer mw.mu.RUnlock()
	for x := range mw.writers {
		newJob := jobsPool.Get().(*job)
		newJob.Writer = mw.writers[x]
		newJob.Data = make([]byte, len(p))
		copy(newJob.Data, p)

		select {
		case jobChannel <- newJob:
		default:
			displayError(errors.New("logger jobs channel is filled"))
			jobChannel <- newJob
		}
	}
	return len(p), nil
}

// multiWriter make and return a new copy of multiWriterHolder
func multiWriter(writers ...io.Writer) (*multiWriterHolder, error) {
	mw := &multiWriterHolder{}
	for x := range writers {
		err := mw.Add(writers[x])
		if err != nil {
			return nil, err
		}
	}
	return mw, nil
}
