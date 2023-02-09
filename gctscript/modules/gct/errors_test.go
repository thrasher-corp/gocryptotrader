package gct

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common"
)

func TestErrorResponse(t *testing.T) {
	t.Parallel()
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

func TestConstructRuntimeError(t *testing.T) {
	t.Parallel()
	err := constructRuntimeError(0, "", "", nil)
	if !errors.Is(err, common.ErrTypeAssertFailure) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrTypeAssertFailure)
	}
}
