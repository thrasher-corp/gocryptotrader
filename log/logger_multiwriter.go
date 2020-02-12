package log

import (
	"io"
)

// Add appends a new writer to the multiwriter slice
func (mw *multiWriter) Add(writer io.Writer) {
	mw.mu.Lock()
	mw.writers = append(mw.writers, writer)
	mw.mu.Unlock()
}

// Remove removes existing writer from multiwriter slice
func (mw *multiWriter) Remove(writer io.Writer) {
	mw.mu.Lock()

	var removeIDs []int
	for i := range mw.writers {
		if mw.writers[i] == writer {
			removeIDs = append(removeIDs, i)
		}
	}

	for x := range removeIDs {
		mw.writers[x] = mw.writers[len(mw.writers)-1]
		mw.writers[len(mw.writers)-1] = nil
		mw.writers = mw.writers[:len(mw.writers)-1]
	}

	mw.mu.Unlock()
}

// Write concurrent safe Write for each writer
func (mw *multiWriter) Write(p []byte) (n int, err error) {
	type data struct {
		n   int
		err error
	}
	mw.mu.RLock()
	defer mw.mu.RUnlock()

	results := make(chan data, len(mw.writers))

	for _, wr := range mw.writers {
		go func(w io.Writer, p []byte, ch chan data) {
			n, err = w.Write(p)
			if err != nil {
				ch <- data{n, err}
				return
			}
			if n != len(p) {
				ch <- data{n, io.ErrShortWrite}
				return
			}
			ch <- data{n, nil}
		}(wr, p, results)
	}

	for range mw.writers {
		d := <-results
		if d.err != nil {
			return d.n, d.err
		}
	}
	return len(p), nil
}

// MultiWriter make and return a new copy of multiWriter
func MultiWriter(writers ...io.Writer) io.Writer {
	w := make([]io.Writer, len(writers))
	copy(w, writers)
	return &multiWriter{writers: w}
}
