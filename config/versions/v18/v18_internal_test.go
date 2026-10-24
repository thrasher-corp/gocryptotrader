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
		{raw: json.RawMessage(`900000000000`), exp: 15 * time.Minute, ok: true},
		{raw: json.RawMessage(`"15m"`), exp: 15 * time.Minute, ok: true},
		{raw: json.RawMessage(`"0.25h"`), exp: 15 * time.Minute, ok: true},
		{raw: json.RawMessage(`"100ms"`), exp: 100 * time.Millisecond, ok: true},
		{raw: json.RawMessage(`"0s"`), ok: true},
		{raw: json.RawMessage(`"900000000000"`)},
		{raw: json.RawMessage(`"1d"`)},
		{raw: json.RawMessage(`true`)},
	} {
		got, ok := parseInterval(tc.raw)
		assert.Equalf(t, tc.ok, ok, "parseInterval should report whether %s parses", tc.raw)
		assert.Equalf(t, tc.exp, got, "parseInterval should return the duration of %s", tc.raw)
	}
}

func TestShortInterval(t *testing.T) {
	t.Parallel()
	for d, exp := range map[time.Duration]string{
		15 * time.Minute:        "15m",
		100 * time.Millisecond:  "100ms",
		4 * time.Hour:           "4h",
		90 * time.Second:        "1m30s",
		time.Hour + time.Minute: "1h1m",
	} {
		assert.Equalf(t, exp, shortInterval(d), "%s should be saved as the config saves it", d)
	}
}
