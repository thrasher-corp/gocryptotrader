package logger

import (
	"fmt"
	"io"
)

func (mw *multiWriter) Add(writer io.Writer) {
	mw.mu.Lock()
	mw.writers = append(mw.writers, writer)
	mw.mu.Unlock()
}

func (mw *multiWriter) Remove(writer io.Writer) {
	mw.mu.Lock()
	for i := range mw.writers {
		fmt.Print(mw.writers[i])
		if mw.writers[i] == writer {
			mw.writers = append(mw.writers[:i], mw.writers[i+1:]...)
			fmt.Print(mw.writers[i])
		}
	}
	mw.mu.Unlock()
}

func (mw *multiWriter) Write(p []byte) (n int, err error) {
	type data struct {
		n   int
		err error
	}

	results := make(chan data)

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

func MultiWriter(writers ...io.Writer) io.Writer {
	w := make([]io.Writer, len(writers))
	copy(w, writers)
	return &multiWriter{writers: w}
}
