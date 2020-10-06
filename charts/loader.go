package charts

import (
	"errors"
	"io/ioutil"
	"os"
)

func writeTemplate(input []byte) (*os.File, error) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}

	n, err := f.Write(input)
	if err != nil {
		return nil, err
	}
	if n != len(input) {
		return nil, errors.New("data length mismatch")
	}
	return f, nil
}
