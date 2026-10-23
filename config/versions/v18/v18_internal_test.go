package v18

import (
	"encoding/json" //nolint:depguard // Matches the encoding/json the migration uses
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw json.RawMessage
		exp time.Duration
		ok  bool
	}{
		{raw: nil, ok: true},
		{raw: json.RawMessage(`null`)},
		{raw: json.RawMessage(`100000000`), exp: 100 * time.Millisecond, ok: true},
		{raw: json.RawMessage(`"100ms"`), exp: 100 * time.Millisecond, ok: true},
		{raw: json.RawMessage(`"0.1s"`), exp: 100 * time.Millisecond, ok: true},
		{raw: json.RawMessage(`"0s"`), ok: true},
		{raw: json.RawMessage(`"100000000"`)},
		{raw: json.RawMessage(`"wibble"`)},
		{raw: json.RawMessage(`true`)},
	} {
		got, ok := parseInterval(tc.raw)
		assert.Equalf(t, tc.ok, ok, "parseInterval should report whether %s parses", tc.raw)
		assert.Equalf(t, tc.exp, got, "parseInterval should return the duration of %s", tc.raw)
	}
}
