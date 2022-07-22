package gct

import (
	"errors"
	"testing"
)

func TestErrorResponse(t *testing.T) {
	_, err := errorResponsef("")
	if !errors.Is(err, errFormatStringIsEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errFormatStringIsEmpty)
	}

	_, err = errorResponsef("--")
	if !errors.Is(err, errNoArguments) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoArguments)
	}

	errResp, err := errorResponsef("error %s", "hello")
	if err != nil {
		t.Fatal(err)
	}

	if errResp.String() != `error: "error hello"` {
		t.Fatalf("received: %v but expected: %v", errResp.String(), `error: "error hello"`)
	}
}
