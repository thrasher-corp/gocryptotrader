package request

import (
	"bytes"
	jsonv1 "encoding/json"   //nolint:depguard // Decodes with the standard library, whatever the build's JSON backend.
	"encoding/json/jsontext" //nolint:depguard // Tests the token-based redactor directly.
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{"document in a string value", `{"code":40101,"original":"{\"api_key\":\"k\",\"sig\":\"s\",\"params\":{}}"}`, `{"code":40101,"original":"{\"api_key\":\"[REDACTED]\",\"sig\":\"[REDACTED]\",\"params\":{}}"}`},
		{"public document in a string value", `{"original":"{\"method\":\"public/get-book\",\"params\":{}}"}`, `{"original":"{\"method\":\"public/get-book\",\"params\":{}}"}`},
		{"array document in a string value", `["[{\"secret\":1}]",2]`, `["[{\"secret\":\"[REDACTED]\"}]",2]`},
		{"escaped document", `{"note":"\u007b\"key\":1}"}`, `{"note":"{\"key\":\"[REDACTED]\"}"}`},
		{"document after a space", `{"note":" {\"key\":1}"}`, `{"note":" {\"key\":\"[REDACTED]\"}"}`},
		{"document after escaped whitespace", `{"note":"\t\r\n {\"key\":1}"}`, `{"note":"\t\r\n {\"key\":\"[REDACTED]\"}"}`},
		{"document as the whole body", `"{\"api_key\":\"k\",\"n\":1}"`, `"{\"api_key\":\"[REDACTED]\",\"n\":1}"`},
		{"escaped public document kept byte for byte", `{"note":"\u007b\"n\":\"a\/b\"}"}`, `{"note":"\u007b\"n\":\"a\/b\"}"}`},
		{"escaped text", `{"note":"\tplain"}`, `{"note":"\tplain"}`},
		{"text starting with a brace", `{"note":"{not a document"}`, `{"note":"[REDACTED]"}`},
		{"empty string", `{"note":""}`, `{"note":""}`},
		{"credential holding a document", `{"key":"{\"a\":1}"}`, `{"key":"[REDACTED]"}`},
		{"name holding a document", `{"{\"key\":1}":2}`, `{"{\"key\":1}":2}`},
		{"document with invalid UTF8", "{\"original\":\"{\\\"note\\\":\\\"\xff\\\",\\\"key\\\":1}\"}", "{\"original\":\"{\\\"note\\\":\\\"\uFFFD\\\",\\\"key\\\":\\\"[REDACTED]\\\"}\"}"},
		{"documents at the nesting limit", jsonDocumentInStringsForTest(t, `{"key":1}`, maxJSONDocumentLogDepth), jsonDocumentInStringsForTest(t, `{"key":"[REDACTED]"}`, maxJSONDocumentLogDepth)},
		{"documents beyond the nesting limit", jsonDocumentInStringsForTest(t, `{"n":1}`, maxJSONDocumentLogDepth+1), jsonDocumentInStringsForTest(t, `{"a":"[REDACTED]"}`, maxJSONDocumentLogDepth)},
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
	for _, seed := range []string{`{}`, `[]`, `{"password":"example"}`, `{"pass\u0077ord":{}}`, `[{"key":1},{"nonce":2}]`, `{"key":1,"key":2}`, `{"value":1e9999}`, `{"original":"{\"api_key\":\"k\",\"n\":[1]}"}`, `["[{\"secret\":1}]"]`, `{"note":"\u007b\"key\":1}"}`, `{"note":"{not a document"}`} {
		f.Add([]byte(seed))
	}
	// Sonic rejects nesting this deep, which jsontext accepts.
	f.Add([]byte(strings.Repeat("[", 5000) + "0" + strings.Repeat("]", 5000)))
	f.Fuzz(func(t *testing.T, payload []byte) {
		t.Parallel()
		got, err := redactJSONBody(payload)
		if err != nil {
			return
		}
		var original, filtered any
		// The build's JSON backend may reject input that jsontext accepts.
		require.NoError(t, jsonv1.Unmarshal(payload, &original), "accepted input must decode")
		require.NoError(t, jsonv1.Unmarshal(got, &filtered), "filtered output must decode")
		assert.Equal(t, redactJSONValueForTest(original, nil, maxJSONDocumentLogDepth), expandJSONDocumentsForTest(filtered), "raw redaction should match the decoded traversal")
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

// jsonDocumentForTest is a JSON document a string value held, decoded in the string's place.
type jsonDocumentForTest struct {
	value any
}

// holdsJSONDocumentForTest reports whether a decoded string value holds a JSON object or array.
func holdsJSONDocumentForTest(s string) bool {
	s = strings.TrimLeft(s, " \t\r\n")
	return s != "" && (s[0] == '{' || s[0] == '[')
}

// jsonDocumentInStringsForTest returns a JSON document held in a string value levels times over.
func jsonDocumentInStringsForTest(tb testing.TB, document string, levels int) string {
	tb.Helper()
	for range levels {
		quoted, err := jsontext.AppendQuote(nil, document)
		require.NoError(tb, err, "AppendQuote must quote the document")
		document = `{"a":` + string(quoted) + `}`
	}
	return document
}

// expandJSONDocumentsForTest decodes each JSON document a decoded value's strings hold in the string's place.
func expandJSONDocumentsForTest(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			typed[key] = expandJSONDocumentsForTest(nested)
		}
	case []any:
		for i := range typed {
			typed[i] = expandJSONDocumentsForTest(typed[i])
		}
	case string:
		var document any
		if holdsJSONDocumentForTest(typed) && jsonv1.Unmarshal([]byte(typed), &document) == nil {
			return jsonDocumentForTest{expandJSONDocumentsForTest(document)}
		}
	}
	return value
}

// redactJSONValueForTest returns a decoded value redacted as redactJSONBody redacts its encoding, with each JSON
// document its strings hold, up to depth deep, decoded in the string's place as expandJSONDocumentsForTest decodes it.
func redactJSONValueForTest(value any, sensitiveKeys map[string]bool, depth int) any {
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
				continue
			}
			typed[key] = redactJSONValueForTest(nested, sensitiveKeys, depth)
		}
	case []any:
		if len(typed) > 1 && sensitiveKeys == nil {
			sensitiveKeys = make(map[string]bool)
		}
		for i := range typed {
			typed[i] = redactJSONValueForTest(typed[i], sensitiveKeys, depth)
		}
	case string:
		if !holdsJSONDocumentForTest(typed) {
			return typed
		}
		var document any
		if depth == 0 || jsonv1.Unmarshal([]byte(typed), &document) != nil {
			return "[REDACTED]"
		}
		return jsonDocumentForTest{redactJSONValueForTest(document, nil, depth-1)}
	}
	return value
}
