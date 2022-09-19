package log

import (
	"errors"
	"fmt"
	"io"
)

var (
	// ErrWriterNotFound is returned when a writer is not found
	ErrWriterNotFound = errors.New("io.Writer not found")

	errWriterAlreadyLoaded = errors.New("io.Writer already loaded")
)

// AddWriter appends an additional writer to all subloggers
func AddWriter(w io.Writer) error {
	RWM.Lock()
	defer RWM.Unlock()
	var err error
	for _, v := range SubLoggers {
		var mwh *multiWriterHolder
		switch v.output.(type) {
		case *multiWriterHolder:
			mwh = v.output.(*multiWriterHolder)
		default:
			mwh, err = multiWriter(v.output)
			if err != nil {
				return err
			}
		}
		err = mwh.Add(w)
		if err != nil {
			return err
		}
		v.output = mwh
	}
	return nil
}

// RemoveWriter removes a writer from all subloggers
func RemoveWriter(w io.Writer) error {
	RWM.Lock()
	defer RWM.Unlock()
	for _, v := range SubLoggers {
		mr, ok := v.output.(*multiWriterHolder)
		if !ok {
			continue
		}
		err := mr.Remove(w)
		if err != nil {
			return err
		}
	}
	return nil
}

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
	return ErrWriterNotFound
}

// Write concurrent safe Write for each writer
func (mw *multiWriterHolder) Write(p []byte) (int, error) {
	type data struct {
		n   int
		err error
	}

	results := make(chan data, len(mw.writers))
	mw.mu.RLock()
	defer mw.mu.RUnlock()
	for x := range mw.writers {
		go func(w io.Writer, p []byte, ch chan<- data) {
			n, err := w.Write(p)
			if err != nil {
				ch <- data{n, fmt.Errorf("%T %w", w, err)}
				return
			}
			if n != len(p) {
				ch <- data{n, fmt.Errorf("%T %w", w, io.ErrShortWrite)}
				return
			}
			ch <- data{n, nil}
		}(mw.writers[x], p, results)
	}

	for range mw.writers {
		// NOTE: These results do not necessarily reflect the current io.writer
		// due to the go scheduler and writer finishing at different times, the
		// response coming from the channel might not match up with the for loop
		// writer.
		d := <-results
		if d.err != nil {
			return d.n, d.err
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
