package json

import (
	"bytes"
	"io"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// number mirrors types.Number, which cannot be imported here without a cycle
type number float64

func (n *number) UnmarshalJSON(data []byte) error {
	if data[0] == '"' {
		data = data[1 : len(data)-1]
	}
	if len(data) == 0 {
		*n = 0
		return nil
	}
	v, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return err
	}
	*n = number(v)
	return nil
}

// Positional &[N]any{...} decoding needs MergeWithLegacySemantics under json/v2.
func TestPositionalDecoding(t *testing.T) {
	t.Parallel()
	var ts int64
	var open, high number
	require.NoError(t, Unmarshal([]byte(`[1700000000,"42000.1","42500.2"]`), &[3]any{&ts, &open, &high}), "Unmarshal must not error")
	assert.Equal(t, int64(1700000000), ts, "timestamp should decode into the supplied pointer")
	assert.InDelta(t, 42000.1, float64(open), 1e-9, "open should decode into the supplied pointer")
	assert.InDelta(t, 42500.2, float64(high), 1e-9, "high should decode into the supplied pointer")
}

// Exchange payloads need not match the destination array length.
func TestPositionalDecodingAnyLength(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, data string
		wantA      string
		wantB      string
		wantC      string
	}{
		{"exact", `["a","b","c"]`, "a", "b", "c"},
		{"short", `["a"]`, "a", "untouched", "untouched"},
		{"long", `["a","b","c","d","e"]`, "a", "b", "c"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a, b, c := "untouched", "untouched", "untouched"
			require.NoError(t, Unmarshal([]byte(tc.data), &[3]any{&a, &b, &c}), "Unmarshal must not error")
			assert.Equal(t, tc.wantA, a, "first element should decode")
			assert.Equal(t, tc.wantB, b, "second element should match expectation")
			assert.Equal(t, tc.wantC, c, "third element should match expectation")
		})
	}
}

// Outbound order requests send quoted booleans, which json/v2 restricts the `,string` tag from.
func TestStringTagOnBool(t *testing.T) {
	t.Parallel()
	var in struct {
		ReduceOnly bool `json:"reduceOnly,string"`
	}
	require.NoError(t, Unmarshal([]byte(`{"reduceOnly":"true"}`), &in), "Unmarshal must accept a quoted bool")
	assert.True(t, in.ReduceOnly, "quoted bool should decode to true")

	out, err := Marshal(&in)
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"reduceOnly":"true"}`, string(out), "bool should marshal back out quoted")
}

// json/v2 has no default time.Duration representation; config.json nanoseconds must round-trip.
func TestDuration(t *testing.T) {
	t.Parallel()
	var v struct {
		Timeout time.Duration `json:"timeout"`
	}
	require.NoError(t, Unmarshal([]byte(`{"timeout":30000000000}`), &v), "Unmarshal must not error")
	assert.Equal(t, 30*time.Second, v.Timeout, "duration should decode from nanoseconds")

	out, err := Marshal(&v)
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"timeout":30000000000}`, string(out), "duration should marshal back to nanoseconds")
}

// A type parsing its own input needs no format tag to carry NaN and Inf.
func TestNumberNonFinite(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		in   string
		want float64
	}{
		{`"NaN"`, math.NaN()},
		{`"Inf"`, math.Inf(1)},
		{`"-Infinity"`, math.Inf(-1)},
	} {
		var n number
		require.NoErrorf(t, Unmarshal([]byte(tc.in), &n), "Unmarshal must not error for %s", tc.in)
		if math.IsNaN(tc.want) {
			assert.Truef(t, math.IsNaN(float64(n)), "%s should decode to NaN", tc.in)
			continue
		}
		assert.Equalf(t, tc.want, float64(n), "%s should decode to the expected infinity", tc.in)
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	type payload struct {
		Name   string     `json:"name"`
		Age    int        `json:"age"`
		Extra  RawMessage `json:"extra"`
		Nested []string   `json:"nested"`
	}
	src := `{"name":"Wednesday","age":6,"extra":{"a":1},"nested":["Gomez","Morticia"]}`

	var p payload
	require.NoError(t, Unmarshal([]byte(src), &p), "Unmarshal must not error")
	assert.Equal(t, "Wednesday", p.Name, "name should decode")
	assert.Equal(t, 6, p.Age, "age should decode")
	assert.JSONEq(t, `{"a":1}`, string(p.Extra), "RawMessage should be captured verbatim")

	out, err := Marshal(&p)
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, src, string(out), "payload should round-trip")
}

func TestMarshalIndent(t *testing.T) {
	t.Parallel()
	out, err := MarshalIndent(map[string]int{"a": 1}, "", "  ")
	require.NoError(t, err, "MarshalIndent must not error")
	assert.Contains(t, string(out), "\n  ", "output should be indented")
	assert.JSONEq(t, `{"a":1}`, string(out), "indented output should still be equivalent JSON")
}

func TestValid(t *testing.T) {
	t.Parallel()
	assert.True(t, Valid([]byte(`{"a":1}`)), "well-formed JSON should report valid")
	assert.False(t, Valid([]byte(`{"a":}`)), "malformed JSON should report invalid")
	assert.True(t, Valid([]byte(`{"a":1,"a":2}`)), "duplicate names should report valid, matching what Unmarshal accepts")
}

func TestEncoderDecoder(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	require.NoError(t, NewEncoder(&buf).Encode(map[string]int{"a": 1}), "Encode must not error")
	assert.JSONEq(t, `{"a":1}`, buf.String(), "encoded value should match")

	dec := NewDecoder(bytes.NewReader([]byte(`{"a":1} trailing`)))
	var got map[string]int
	require.NoError(t, dec.Decode(&got), "Decode must not error")
	assert.Equal(t, map[string]int{"a": 1}, got, "decoded value should match")

	rest, err := io.ReadAll(dec.Buffered())
	require.NoError(t, err, "reading the buffered remainder must not error")
	assert.Contains(t, string(rest), "trailing", "Buffered should expose the unread remainder")
}

func TestUnmarshalError(t *testing.T) {
	t.Parallel()
	var v struct {
		A int `json:"a"`
	}
	assert.Error(t, Unmarshal([]byte(`{"a":"not-a-number"}`), &v), "type mismatch should error")
	assert.Error(t, Unmarshal([]byte(`{"a":`), &v), "malformed JSON should error")
}

// v1 marshalled nil slices and maps as null; exchange request bodies are signed as sent.
func TestNilSliceAndMapAsNull(t *testing.T) {
	t.Parallel()
	out, err := Marshal(&struct {
		Slice []string          `json:"slice"`
		Map   map[string]string `json:"map"`
		Bytes []byte            `json:"bytes"`
	}{})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"slice":null,"map":null,"bytes":null}`, string(out), "nil slice, map and byte slice should marshal as null")
}

// v1 escaped HTML characters and the JS line separators by default.
func TestEscaping(t *testing.T) {
	t.Parallel()
	out, err := Marshal(map[string]string{"a": "a<b>c&d\u2028\u2029"})
	require.NoError(t, err, "Marshal must not error")
	//nolint:testifylint // JSONEq compares semantically, which would pass with or without the escaping under test
	assert.Equal(t, `{"a":"a\u003cb\u003ec\u0026d\u2028\u2029"}`, string(out), "HTML characters and JS line separators should be escaped")
}

// v1 emitted a RawMessage with the escaping it arrived with; mock recordings are re-encoded verbatim.
func TestRawMessagePreservesEscaping(t *testing.T) {
	t.Parallel()
	out, err := Marshal(map[string]any{"r": RawMessage(`{"a":"\u0041"}`)})
	require.NoError(t, err, "Marshal must not error")
	//nolint:testifylint // JSONEq compares semantically, which would pass with or without the escaping under test
	assert.Equal(t, `{"r":{"a":"\u0041"}}`, string(out), "pre-escaped sequences should be preserved rather than canonicalised")
}

// v1 substituted U+FFFD for invalid UTF-8; v2 fails the whole decode instead.
func TestInvalidUTF8(t *testing.T) {
	t.Parallel()
	const payload = "{\"a\":\"\xff\"}"
	var v struct {
		A string `json:"a"`
	}
	require.NoError(t, Unmarshal([]byte(payload), &v), "Unmarshal must not error on invalid UTF-8")
	assert.Equal(t, "\uFFFD", v.A, "invalid UTF-8 should decode to the replacement character")
	assert.True(t, Valid([]byte(payload)), "invalid UTF-8 should still report valid")
}

// Binance sends "e" and "E" in one object; an exact match must beat the case insensitive fallback.
func TestExactCaseMatchWins(t *testing.T) {
	t.Parallel()
	type payload struct {
		EventType string `json:"e"`
		EventTime int64  `json:"E"`
	}
	for _, in := range []string{
		`{"e":"executionReport","E":1700000000123}`,
		`{"E":1700000000123,"e":"executionReport"}`,
	} {
		var p payload
		require.NoErrorf(t, Unmarshal([]byte(in), &p), "Unmarshal must not error for %s", in)
		assert.Equalf(t, "executionReport", p.EventType, "e should decode into the exactly matching field for %s", in)
		assert.Equalf(t, int64(1700000000123), p.EventTime, "E should decode into the exactly matching field for %s", in)
	}

	var p payload
	require.NoError(t, Unmarshal([]byte(`{"E":1700000000123}`), &p), "Unmarshal must not error")
	assert.Empty(t, p.EventType, "the case insensitive fallback should not claim E when a field tagged E exists")
	assert.Equal(t, int64(1700000000123), p.EventTime, "E should still decode")
}

// v1 took these as setters where json/v2 takes them as construction options.
func TestEncoderDecoderSettings(t *testing.T) {
	t.Parallel()
	t.Run("SetIndent", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		enc := NewEncoder(&buf)
		require.NoError(t, enc.Encode(map[string]int{"a": 1}), "Encode must not error")
		enc.SetIndent("", "  ")
		require.NoError(t, enc.Encode(map[string]int{"b": 2}), "Encode must not error")
		assert.Equal(t, "{\"a\":1}\n{\n  \"b\": 2\n}\n", buf.String(), "SetIndent should apply from the next value on, as it did in v1")
	})

	t.Run("SetEscapeHTML", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		enc := NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		require.NoError(t, enc.Encode(map[string]string{"a": "a<b>c&d\u2028"}), "Encode must not error")
		//nolint:testifylint // JSONEq compares semantically, which would pass with or without the escaping under test
		assert.Equal(t, "{\"a\":\"a<b>c&d\\u2028\"}\n", buf.String(), "disabling HTML escaping should leave the JS line separators escaped, as it did in v1")
	})

	t.Run("DisallowUnknownFields", func(t *testing.T) {
		t.Parallel()
		var v struct {
			A int `json:"a"`
		}
		dec := NewDecoder(bytes.NewReader([]byte(`{"a":1,"b":2}`)))
		require.NoError(t, dec.Decode(&v), "an unknown member must decode by default")

		dec = NewDecoder(bytes.NewReader([]byte(`{"a":1,"b":2}`)))
		dec.DisallowUnknownFields()
		assert.Error(t, dec.Decode(&v), "an unknown member should error once disallowed")
	})

	t.Run("More", func(t *testing.T) {
		t.Parallel()
		dec := NewDecoder(bytes.NewReader([]byte(`{"a":1} {"b":2}`)))
		var m map[string]int
		assert.True(t, dec.More(), "a fresh stream should report more")
		require.NoError(t, dec.Decode(&m), "Decode must not error")
		assert.True(t, dec.More(), "a stream with a value left should report more")
		require.NoError(t, dec.Decode(&m), "Decode must not error")
		assert.False(t, dec.More(), "an exhausted stream should not report more")
	})
}
