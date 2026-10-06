package request

import (
	"bytes"
	"encoding/json/jsontext" //nolint:depguard // Tests the token-based redactor directly.
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestRedactJSONBody(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, body, expected string }{
		{"public", ` {"orderId":1234567890123456789,"note":"a&b"} `, ` {"orderId":1234567890123456789,"note":"a&b"} `},
		{"format", " {\n\t\"password\" : 42, \"n\":1.2300e+02 }\n", " {\n\t\"password\" : \"[REDACTED]\", \"n\":1.2300e+02 }\n"},
		{"all types", `{"key":null,"key":false,"key":[],"key":{"password":1},"key":"x"}`, `{"key":"[REDACTED]","key":"[REDACTED]","key":"[REDACTED]","key":"[REDACTED]","key":"[REDACTED]"}`},
		{"escaped", `{"pass\u0077ord":"a\"b","Key":1}`, `{"pass\u0077ord":"[REDACTED]","Key":"[REDACTED]"}`},
		{"nested siblings", `[{"data":{"key":1}},{"token":2}]`, `[{"data":{"key":"[REDACTED]"}},{"token":"[REDACTED]"}]`},
		{"scalar", `"public"`, `"public"`},
		{"duplicate public", `{"n":1,"n":2}`, `{"n":1,"n":2}`},
		{"invalid UTF8", "{\"note\":\"\xff\",\"key\":0}", "{\"note\":\"\xff\",\"key\":\"[REDACTED]\"}"},
		{"deep nesting", strings.Repeat("[", 100) + `{"key":0}` + strings.Repeat("]", 100), strings.Repeat("[", 100) + `{"key":"[REDACTED]"}` + strings.Repeat("]", 100)},
		{"empty", "", ""},
		{"unfinished", `{"key":`, ""},
		{"trailing", `{} {}`, ""},
		{"bad separator", `{"key"::0}`, ""},
		{"control", "{\"\x00\":0}", ""},
		{"overflow", `{"key":[1e9999]}`, ""},
		{"public overflow", `{"value":1e9999}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			payload := []byte(tc.body)
			got, err := redactJSONBody(payload)
			if tc.expected == "" {
				assert.Error(t, err, "invalid JSON should fail closed")
				if tc.name == "empty" || tc.name == "trailing" {
					assert.ErrorIs(t, err, errInvalidJSONLogBody, "empty and multiple documents should return the validation error")
				}
				return
			}
			require.NoError(t, err, "valid JSON must redact")
			assert.Equal(t, tc.expected, string(got), "only sensitive values should change")
			assert.Equal(t, tc.body, string(payload), "input should remain unchanged")
		})
	}
}

func FuzzRedactJSONBody(f *testing.F) {
	for _, seed := range []string{`{}`, `[]`, `{"password":"example"}`, `{"pass\u0077ord":{}}`, `[{"key":1},{"nonce":2}]`, `{"key":1,"key":2}`, `{"value":1e9999}`, "{\"\x00\":0}"} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, payload []byte) {
		t.Parallel()
		got, err := redactJSONBody(payload)
		if err != nil {
			return
		}
		var original, filtered any
		require.NoError(t, json.Unmarshal(payload, &original), "accepted input must decode")
		require.NoError(t, json.Unmarshal(got, &filtered), "filtered output must decode")
		redactJSONValueForTest(original, nil)
		assert.Equal(t, original, filtered, "raw redaction should match the decoded traversal")
	})
}

func TestSkipJSONLogValue(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, value string
		invalid     bool
	}{
		{name: "string", value: `"example"`},
		{name: "number", value: `42`},
		{name: "null", value: `null`},
		{name: "boolean", value: `false`},
		{name: "empty object", value: `{}`},
		{name: "nested containers", value: `[{"key":[1,{"token":"example"}]}]`},
		{name: "overflow", value: `1e9999`, invalid: true},
		{name: "nested overflow", value: `{"key":[1e9999]}`, invalid: true},
		{name: "missing value", invalid: true},
		{name: "unfinished container", value: `{"key":`, invalid: true},
		{name: "invalid nested string", value: `{"key":"\x"}`, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			prefix := `{"secret":`
			decoder := jsontext.NewDecoder(bytes.NewBufferString(prefix + tc.value + `,"nonce":1}`))
			_, err := decoder.ReadToken()
			require.NoError(t, err, "object opening must decode")
			_, err = decoder.ReadToken()
			require.NoError(t, err, "sensitive field name must decode")
			err = skipJSONLogValue(decoder)
			if tc.invalid {
				assert.Error(t, err, "invalid values should be rejected")
				return
			}
			require.NoError(t, err, "valid sensitive value must be consumed")
			assert.Equal(t, int64(len(prefix)+len(tc.value)), decoder.InputOffset(), "only the sensitive value should be consumed")
			next, err := decoder.ReadToken()
			require.NoError(t, err, "following public name must remain readable")
			assert.Equal(t, "nonce", next.String(), "next token should be the public sibling")
		})
	}
}

func redactJSONValueForTest(value any, sensitiveKeys map[string]bool) bool {
	redacted := false
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			sensitive, checked := sensitiveKeys[key]
			if !checked {
				sensitive = isSensitiveLogKey(key)
				if sensitiveKeys != nil {
					sensitiveKeys[key] = sensitive
				}
			}
			if sensitive {
				typed[key] = "[REDACTED]"
				redacted = true
				continue
			}
			redacted = redactJSONValueForTest(nested, sensitiveKeys) || redacted
		}
	case []any:
		if len(typed) > 1 && sensitiveKeys == nil {
			sensitiveKeys = make(map[string]bool)
		}
		for _, nested := range typed {
			redacted = redactJSONValueForTest(nested, sensitiveKeys) || redacted
		}
	}
	return redacted
}
