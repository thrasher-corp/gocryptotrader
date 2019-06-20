package loggerv2

import (
	"fmt"
	"io"
)

func (mw *multiWriter) Add(writers ...io.Writer) {
	mw.mu.Lock()
	mw.writers = append(mw.writers, writers...)
	mw.mu.Unlock()
}

func (mw *multiWriter) Remove(writers ...io.Writer) {
	mw.mu.Lock()
	for i := len(mw.writers) - 1; i > 0; i-- {
		for v := range writers {
			if mw.writers[i] == writers[v] {
				fmt.Println(writers[v])
				mw.writers = append(mw.writers[:i], mw.writers[i+1:]...)
				break
			}
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

func test() {

}
