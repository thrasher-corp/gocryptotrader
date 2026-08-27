//go:build !sonic_on

// Tests for the unexported helpers in json.go, which only exist in this build; json_test.go is
// untagged and compiles against sonic too, so it cannot reference them

package json

import (
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
