package gct

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
)

func TestErrorResponse(t *testing.T) {
	t.Parallel()
	_, err := errorResponsef("")
	require.ErrorIs(t, err, errFormatStringIsEmpty)

	_, err = errorResponsef("--")
	require.ErrorIs(t, err, errNoArguments)

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
	require.ErrorIs(t, err, common.ErrTypeAssertFailure)
}
