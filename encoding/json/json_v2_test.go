//go:build !sonic_on

// Tests for the unexported helpers in json.go, which only exist in this build; json_test.go is
// untagged and compiles against sonic too, so it cannot reference them

package json

import (
	jsonv1 "encoding/json" //nolint:depguard // the wrapper is compared against what it stands in for
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// indentBytes is exercised directly since it carries the fallback both MarshalIndent and Encode
// rely on, including the invalid-JSON path neither of them can reach after a successful marshal.
func TestIndentBytes(t *testing.T) {
	t.Parallel()
	out, err := indentBytes([]byte(`{"a":1,"b":[2,3]}`), ">>", "  ")
	require.NoError(t, err, "indentBytes must not error on valid JSON")
	assert.Equal(t, "{\n>>  \"a\": 1,\n>>  \"b\": [\n>>    2,\n>>    3\n>>  ]\n>>}", string(out), "indentBytes should match v1's Indent")

	out, err = indentBytes([]byte("{"), "", "  ")
	require.Error(t, err, "indentBytes must error on malformed JSON")
	assert.Nil(t, out, "indentBytes should return nil bytes on error")
}

// errReadFailed stands in for an upstream that fails rather than ending
var errReadFailed = errors.New("read failed")

// failingReader serves r, then fails where the reader underneath would have reported end of input
type failingReader struct {
	r   io.Reader
	err error
}

func (f *failingReader) Read(p []byte) (int, error) {
	n, err := f.r.Read(p)
	if errors.Is(err, io.EOF) {
		if f.err != nil {
			return n, f.err
		}
		return n, errReadFailed
	}
	return n, err
}

// TestMoreMatchesStdlibOnReadFailure pins More against the encoding/json this build stands in for,
// rather than against a hand-written expectation. A reader wrapping io.EOF is the case errors.Is
// would treat as end of input where v1 reports an error
func TestMoreMatchesStdlibOnReadFailure(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "a plain failure", err: errReadFailed},
		{name: "a failure wrapping io.EOF", err: fmt.Errorf("reader: %w", io.EOF)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			want := jsonv1.NewDecoder(&failingReader{r: strings.NewReader(""), err: tc.err}).More()
			got := NewDecoder(&failingReader{r: strings.NewReader(""), err: tc.err}).More()
			assert.Equal(t, want, got, "More should agree with encoding/json on a failed read")
		})
	}
}

// TestDecodeReportsTheErrorMoreRead covers a reader that fails once and then recovers. Without
// latching, More reads the failure and the Decode that follows succeeds on the next value, so the
// error is reported to nobody
func TestDecodeReportsTheErrorMoreRead(t *testing.T) {
	t.Parallel()
	dec := NewDecoder(&failOnceReader{r: strings.NewReader(`{"a":2}`)})
	var m map[string]int
	assert.True(t, dec.More(), "a failed read should report more")
	assert.ErrorIs(t, dec.Decode(&m), errReadFailed, "Decode should report the failure More read rather than the value after it")
	assert.ErrorIs(t, dec.Decode(&m), errReadFailed, "the failure should be latched, as v1 latches it")
}

// failOnceReader fails its first read and serves r from then on
type failOnceReader struct {
	r      io.Reader
	failed bool
}

func (f *failOnceReader) Read(p []byte) (int, error) {
	if !f.failed {
		f.failed = true
		return 0, errReadFailed
	}
	return f.r.Read(p)
}

// TestDecodeErrorTypeSurvivesMore pins the error type Decode reports after More has peeked. More
// has to read a token to tell a failed read from end of input, and reporting that raw token error
// from Decode would hand back a *jsontext.SyntacticError where v1 hands back a *SyntaxError, which
// is the type ReportErrorsWithLegacySemantics exists to preserve
func TestDecodeErrorTypeSurvivesMore(t *testing.T) {
	t.Parallel()
	for _, callMore := range []bool{false, true} {
		dec := NewDecoder(strings.NewReader("xyz"))
		if callMore {
			assert.True(t, dec.More(), "malformed input should report more")
		}
		var v any
		err := dec.Decode(&v)
		var syntaxErr *SyntaxError
		assert.ErrorAsf(t, err, &syntaxErr, "Decode should report a *SyntaxError with More called first %v", callMore)
	}
}
