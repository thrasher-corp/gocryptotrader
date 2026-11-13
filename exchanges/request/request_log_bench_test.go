package request

import (
	"strings"
	"testing"
)

// Compare revisions with the same Go version, JSON backend and command, for example:
// go test ./exchanges/request -run '^$' -bench 'Benchmark(RedactEncodedValues|PathForLog|BodyForLog)$' -benchmem -benchtime=200ms -count=5
// Add -tags sonic_on to compare the alternative JSON backend separately.
// Keep fixture construction outside b.Loop so only filtering is measured.
func BenchmarkRedactEncodedValues(b *testing.B) {
	for _, tc := range []struct {
		name  string
		query string
	}{
		{name: "empty"},
		{name: "public", query: "symbol=BTCUSDT&limit=100&recvWindow=5000&timestamp=1789610256254"},
		{name: "signed", query: "symbol=BTCUSDT&timestamp=1789610256254&signature=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
		{name: "form", query: "key=example-key&nonce=1789610256359875828&signature=example-signature"},
		{name: "escaped", query: "api%5Fkey=example-key&symbol=BTC%2FUSD&nonce=1"},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = redactEncodedValues(tc.query)
			}
		})
	}
}

func BenchmarkPathForLog(b *testing.B) {
	for _, path := range []string{
		"https://example.com/api",
		"https://example.com/api?symbol=BTCUSDT&limit=100",
		"https://example.com/api?timestamp=1789610256254&signature=example-signature",
	} {
		b.Run(path, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = pathForLog(path)
			}
		})
	}
}

func BenchmarkBodyForLog(b *testing.B) {
	const publicOrder = `{"orderId":1234567890123456789,"symbol":"BTCUSDT","price":"65000.00","quantity":"0.01","status":"NEW"}`
	const credential = `{"api_key":"example-key","secret":"example-secret","permissions":["read","trade"],"enabled":true}`
	for _, tc := range []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "public_object", contentType: "application/json", body: publicOrder},
		{name: "nested_credentials", contentType: "application/json", body: `{"result":{"account":` + credential + `},"success":true}`},
		{name: "public_array", contentType: "application/json", body: "[" + strings.Repeat(publicOrder+",", 99) + publicOrder + "]"},
		{name: "credential_array", contentType: "application/json", body: "[" + strings.Repeat(credential+",", 99) + credential + "]"},
		{name: "form_labelled_json", contentType: "application/x-www-form-urlencoded", body: credential},
		{name: "inferred_json", body: publicOrder},
		{name: "scalar", body: `"diagnostic message"`},
		{name: "invalid_json", contentType: "application/json", body: `{"password":"example-secret"`},
	} {
		b.Run(tc.name, func(b *testing.B) {
			payload := []byte(tc.body)
			b.ReportAllocs()
			b.SetBytes(int64(len(payload)))
			for b.Loop() {
				_ = bodyForLog(payload, tc.contentType)
			}
		})
	}
}
