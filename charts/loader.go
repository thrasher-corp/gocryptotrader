package charts

import (
	"errors"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
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

func readTemplateToByte(input string) ([]byte, error) {
	return ioutil.ReadFile(input)
}

func ReadTemplate(input string) (string, error) {
	b, err := readTemplateToByte(input)
	if err != nil {
		return "", err
	}

	s := make([]string, 0, len(b))
	for i := range b {
		s = append(s, strconv.Itoa(int(b[i])))
	}

	output := "\"" + input + "\": {" + strings.Join(s, ",") + "},"
	return output, nil
}
