package gct

import (
	"testing"
)

func TestErrorResponse(t *testing.T) {
	errResp, err := errorResponse("error %s", "hello")
	if err != nil {
		t.Fatal(err)
	}

	if errResp.String() != `error: "error hello"` {
		t.Fatalf("received: %v but expected: %v", errResp.String(), `error: "error hello"`)
	}
}
