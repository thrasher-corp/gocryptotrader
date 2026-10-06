package request

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
)

var sensitiveLogKeys = []string{
	"Authorization",
	"Cookie",
	"Key",
	"OK-ACCESS-KEY",
	"OK-ACCESS-PASSPHRASE",
	"X-BAPI-SIGN",
	"Btse-Api",
	"X-Auth",
	"Btse-Sign",
	"Kc-Api-Sign",
	"Api-Sign",
	"Authent",
	"listenKey",
	"tfa",
	"X-USER",
	"AccessKeyId",
	"refresh_token",
}

func TestHeaderValuesForLog(t *testing.T) {
	t.Parallel()

	values := []string{"sensitive-value"}
	for _, header := range sensitiveLogKeys {
		assert.Equalf(t, []string{"[REDACTED]"}, headerValuesForLog(header, values), "%s should be redacted", header)
	}
	for _, header := range []string{"Content-Type", "KC-API-KEY-VERSION", "SignatureMethod", "signTimestamp", "X-Auth-Nonce", "gAuth"} {
		assert.Equalf(t, values, headerValuesForLog(header, values), "%s should remain available for diagnostics", header)
	}
}

func BenchmarkHeaderValuesForLog(b *testing.B) {
	values := []string{"sensitive-value"}
	for _, header := range append(sensitiveLogKeys, "Content-Type") {
		b.Run(header, func(b *testing.B) {
			for b.Loop() {
				_ = headerValuesForLog(header, values)
			}
		})
	}
}

func TestIsSensitiveLogKey(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		sensitive bool
	}{
		{name: ""},
		{name: "KEY", sensitive: true},
		{name: "listenKey", sensitive: true},
		{name: "OK-ACCESS-KEY", sensitive: true},
		{name: "prefix_refresh_TOKEN", sensitive: true},
		{name: "Btse-Api", sensitive: true},
		{name: "X-Auth", sensitive: true},
		{name: "prefix_AuTh", sensitive: true},
		{name: "_api", sensitive: true},
		{name: "api"},
		{name: "auth"},
		{name: "gAuth"},
		{name: "X-Auth-Nonce"},
		{name: "KC-API-KEY-VERSION"},
		{name: "SignatureMethod"},
		{name: "signTimestamp"},
		{name: "key-"},
		{name: "token_"},
		{name: "pass-word"},
		{name: "prefix_KEY", sensitive: true},
		{name: "prefix_ſign"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.sensitive, isSensitiveLogKey(tc.name), "key matching should preserve suffix and separator rules")
		})
	}
}

func TestRedactEncodedValues(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		encoded  string
		expected string
	}{
		{name: "empty"},
		{name: "public", encoded: "symbol=BTC%2FUSD&limit=100", expected: "symbol=BTC%2FUSD&limit=100"},
		{name: "malformed name", encoded: "%ZZ=secret-value&nonce=1", expected: "%ZZ=[REDACTED]&nonce=1"},
		{name: "first field", encoded: "key=secret&nonce=1", expected: "key=[REDACTED]&nonce=1"},
		{name: "last field", encoded: "nonce=1&key=secret", expected: "nonce=1&key=[REDACTED]"},
		{name: "consecutive fields", encoded: "key=one&signature=two&nonce=1", expected: "key=[REDACTED]&signature=[REDACTED]&nonce=1"},
		{name: "separated fields", encoded: "key=one&nonce=1&key=two", expected: "key=[REDACTED]&nonce=1&key=[REDACTED]"},
		{name: "empty values", encoded: "key=&token=&nonce=", expected: "key=[REDACTED]&token=[REDACTED]&nonce="},
		{name: "empty fields", encoded: "&key=secret&&nonce=1&", expected: "&key=[REDACTED]&&nonce=1&"},
		{name: "no equals", encoded: "key&nonce=1&token", expected: "key&nonce=1&token"},
		{name: "embedded equals", encoded: "key=one=two&note=a=b", expected: "key=[REDACTED]&note=a=b"},
		{name: "encoded name", encoded: "api%5Fkey=secret&note=a+b", expected: "api%5Fkey=[REDACTED]&note=a+b"},
		{name: "encoded separators", encoded: "note=a%26b%3Dc&key=secret", expected: "note=a%26b%3Dc&key=[REDACTED]"},
		{name: "already redacted", encoded: "key=[REDACTED]&nonce=1", expected: "key=[REDACTED]&nonce=1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, redactEncodedValues(tc.encoded), "redaction should preserve field order and non-sensitive bytes")
		})
	}
}

func TestPathForLog(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		path     string
		expected string
	}{
		{path: "https://example.com/api", expected: "https://example.com/api"},
		{path: "https://example.com/api?", expected: "https://example.com/api?"},
		{path: "https://example.com/api?symbol=BTC%2FUSD&limit=100", expected: "https://example.com/api?symbol=BTC%2FUSD&limit=100"},
		{path: "https://example.com/api?nonce=1&key=secret", expected: "https://example.com/api?nonce=1&key=[REDACTED]"},
		{path: "https://example.com/api?key=[REDACTED]", expected: "https://example.com/api?key=[REDACTED]"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, pathForLog(tc.path), "logged path should preserve non-sensitive bytes")
		})
	}
}

type partialErrorReader struct {
	payload []byte
	err     error
	read    bool
}

type closeErrorReader struct {
	io.Reader
	err error
}

func (c closeErrorReader) Close() error {
	return c.err
}

func (p *partialErrorReader) Read(b []byte) (int, error) {
	if p.read {
		return 0, p.err
	}
	p.read = true
	return copy(b, p.payload), nil
}

func TestDumpRequestForLog(t *testing.T) {
	t.Parallel()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com/api?AccessKeyId=secret-key&timestamp=1", strings.NewReader("key=secret-body&nonce=1"))
	require.NoError(t, err, "NewRequestWithContext must not error")

	dump, err := dumpRequestForLog(req, "")
	require.NoError(t, err, "dumpRequestForLog must not error")
	assert.Contains(t, string(dump), "AccessKeyId=[REDACTED]&timestamp=1", "request dump should redact query credentials")
	assert.Contains(t, string(dump), "key=[REDACTED]&nonce=1", "request dump should treat a body without a content type as form-encoded")
	assert.NotContains(t, string(dump), "secret-key", "request dump should not contain the query credential")
	assert.NotContains(t, string(dump), "secret-body", "request dump should not contain the body credential")
	assert.Equal(t, "AccessKeyId=secret-key&timestamp=1", req.URL.RawQuery, "request dump redaction should not mutate the request URL")

	streamingBody := struct{ io.Reader }{strings.NewReader("key=streaming-secret&nonce=2")}
	req, err = http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com/api", streamingBody)
	require.NoError(t, err, "NewRequestWithContext must not error for a streaming body")
	require.Nil(t, req.GetBody, "streaming request must not have a replayable body")
	dump, err = dumpRequestForLog(req, "")
	require.NoError(t, err, "dumpRequestForLog must not error for a streaming body")
	assert.Contains(t, string(dump), "key=[REDACTED]&nonce=2", "request dump should redact a streaming form body")
	restoredBody, err := io.ReadAll(req.Body)
	require.NoError(t, err, "ReadAll must read the restored streaming body")
	assert.Equal(t, "key=streaming-secret&nonce=2", string(restoredBody), "request dump should restore a non-replayable body")

	req, err = http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com/api", strings.NewReader(`{"note":"a&key=b","nested":{"passphrase":"secret"}}`))
	require.NoError(t, err, "NewRequestWithContext must not error for a JSON body")
	dump, err = dumpRequestForLog(req, "application/json")
	require.NoError(t, err, "dumpRequestForLog must not error for a JSON body")
	assert.Contains(t, string(dump), `{"note":"a&key=b","nested":{"passphrase":"[REDACTED]"}}`, "request dump should redact nested JSON credentials")
	assert.NotContains(t, string(dump), "secret", "request dump should not expose JSON credentials")

	req, err = http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com/api", strings.NewReader(`{"password":"secret"`))
	require.NoError(t, err, "NewRequestWithContext must not error for invalid JSON")
	dump, err = dumpRequestForLog(req, "application/json")
	require.NoError(t, err, "dumpRequestForLog must fail closed for invalid JSON")
	assert.Contains(t, string(dump), "[REDACTED INVALID JSON BODY]", "request dump should replace invalid JSON")
	assert.NotContains(t, string(dump), "secret", "request dump should not expose invalid JSON contents")

	req, err = http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com/api", http.NoBody)
	require.NoError(t, err, "NewRequestWithContext must not error for sensitive headers")
	req.Header.Set("Authorization", "secret-header")
	dump, err = dumpRequestForLog(req, "")
	require.NoError(t, err, "dumpRequestForLog must not error for sensitive headers")
	assert.Contains(t, string(dump), "Authorization: [REDACTED]", "request dump should redact sensitive headers")
	assert.NotContains(t, string(dump), "secret-header", "request dump should not expose sensitive headers")
	assert.Equal(t, "secret-header", req.Header.Get("Authorization"), "request dump should not mutate the original headers")

	readErr := errors.New("body read failure")
	partialBody := &partialErrorReader{payload: []byte("partial-body"), err: readErr}
	req, err = http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com/api", partialBody)
	require.NoError(t, err, "NewRequestWithContext must not error for a partially readable body")
	_, err = dumpRequestForLog(req, "application/json")
	require.ErrorIs(t, err, readErr, "dumpRequestForLog must return the body read error")
	restoredBody, err = io.ReadAll(req.Body)
	require.ErrorIs(t, err, readErr, "the restored body must retain its read error")
	assert.Equal(t, "partial-body", string(restoredBody), "the restored body should retain bytes read before the error")

	req, err = http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com/api", http.NoBody)
	require.NoError(t, err, "NewRequestWithContext must not error for http.NoBody")
	dump, err = dumpRequestForLog(req, "")
	require.NoError(t, err, "dumpRequestForLog must not error for http.NoBody")
	assert.Equal(t, http.NoBody, req.Body, "dumpRequestForLog should preserve an explicit http.NoBody")
	assert.Zero(t, req.ContentLength, "dumpRequestForLog should preserve the zero content length")
	assert.NotContains(t, string(dump), "Transfer-Encoding: chunked", "dumpRequestForLog should not change framing for http.NoBody")
}

func TestBodyForLogContentTypeAndUnchangedJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		contentType string
		body        string
		expected    string
	}{
		{name: "unchanged JSON", contentType: "application/json", body: `{"orderId":1234567890123456789,"note":"a&b"}`, expected: `{"orderId":1234567890123456789,"note":"a&b"}`},
		{name: "nested credential", contentType: "application/json", body: `{"nested":[{"password":"secret"}]}`, expected: `{"nested":[{"password":"[REDACTED]"}]}`},
		{name: "every credential in a collection", contentType: "application/json", body: `[{"secret":"a"},{"nested":{"secret":"b"},"other":{"secret":"c"}},1]`, expected: `[{"secret":"[REDACTED]"},{"nested":{"secret":"[REDACTED]"},"other":{"secret":"[REDACTED]"}},1]`},
		{name: "declared form resembling JSON string", contentType: "application/x-www-form-urlencoded", body: `"password=secret"`, expected: `"password=[REDACTED]`},
		{name: "declared form resembling JSON object", contentType: "application/x-www-form-urlencoded", body: `{"a":1}`, expected: `{"a":1}`},
		{name: "declared form carrying unchanged JSON", contentType: "application/x-www-form-urlencoded", body: `{"orderId":1234567890123456789,"note":"a\u0026b"}`, expected: `{"orderId":1234567890123456789,"note":"a\u0026b"}`},
		{name: "declared form carrying JSON credentials", contentType: "application/x-www-form-urlencoded", body: `{"password":"secret"}`, expected: `{"password":"[REDACTED]"}`},
		{name: "declared form readable as both", contentType: "application/x-www-form-urlencoded", body: `{"password":"secret","note":"&key=form-secret&x="}`, expected: `{"password":"[REDACTED]","note":"&key=[REDACTED]&x="}`},
		{name: "declared form whose form pass breaks its JSON", contentType: "application/x-www-form-urlencoded", body: `{"passphrase":"secret","redirect":"https://example.com/?api_key=1"}`, expected: "[REDACTED INVALID JSON BODY]"},
		{name: "declared form carrying a number outside float64's range", contentType: "application/x-www-form-urlencoded", body: `{"password":"secret","n":1e9999}`, expected: "[REDACTED INVALID JSON BODY]"},
		{name: "non-form body", contentType: "text/plain", body: "upstream unavailable", expected: "[REDACTED NON-FORM BODY]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, string(bodyForLog([]byte(tc.body), tc.contentType)), "bodyForLog should follow the declared content type and preserve unchanged JSON")
		})
	}
}

func TestDumpRequestForLogReplaysCloseFailure(t *testing.T) {
	t.Parallel()
	for _, closeErr := range []error{errors.New("close failure"), io.EOF, io.ErrUnexpectedEOF} {
		body := closeErrorReader{Reader: strings.NewReader("key=secret-body&nonce=1"), err: closeErr}
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com/api", body)
		require.NoError(t, err, "NewRequestWithContext must not error")
		require.Nil(t, req.GetBody, "a plain ReadCloser must not be replayable")

		_, err = dumpRequestForLog(req, "")
		require.ErrorIs(t, err, closeErr, "dumpRequestForLog must return the body close error")

		restored, readErr := io.ReadAll(req.Body)
		assert.Equal(t, "key=secret-body&nonce=1", string(restored), "the restored body should retain the bytes read")
		if errors.Is(closeErr, io.EOF) {
			assert.NoError(t, readErr, "io.ReadAll should treat a replayed EOF as successful completion")
		} else {
			assert.ErrorIs(t, readErr, closeErr, "the restored body should replay the close failure")
		}
	}
}

type cyclicError struct {
	next error
}

func (*cyclicError) Error() string {
	return "cyclic"
}

func (c *cyclicError) Unwrap() error {
	return c.next
}

func TestURLErrorForLogRedactsNestedURLs(t *testing.T) {
	t.Parallel()
	err := &url.Error{
		Op:  http.MethodGet,
		URL: "https://example.com/api?signature=outer-secret",
		Err: &url.Error{
			Op:  http.MethodGet,
			URL: "https://example.com/api?signature=inner-secret",
			Err: errors.New("transport failure"),
		},
	}

	redacted := urlErrorForLog(err)
	assert.NotContains(t, redacted.Error(), "outer-secret", "redacted error should not contain the outer URL credential")
	assert.NotContains(t, redacted.Error(), "inner-secret", "redacted error should not contain the nested URL credential")
	assert.Contains(t, redacted.Error(), "signature=[REDACTED]", "redacted error should retain diagnostic query structure")
	assert.Contains(t, err.Error(), "outer-secret", "redaction should not mutate the original outer error")
	assert.Contains(t, err.Error(), "inner-secret", "redaction should not mutate the original nested error")

	var deep error
	deep = errors.New("transport failure")
	for depth := 0; depth <= maxURLErrorLogDepth; depth++ {
		deep = &url.Error{Op: http.MethodGet, URL: fmt.Sprintf("https://example.com/api?signature=deep-secret-%d", depth), Err: deep}
	}
	redacted = urlErrorForLog(deep)
	assert.ErrorIs(t, redacted, errTruncatedErrorChain, "deep URL errors should be truncated with a safe sentinel")
	assert.NotContains(t, redacted.Error(), "deep-secret", "truncating a deep error should not expose URL credentials")

	cyclicRemainder := new(cyclicError)
	cyclicRemainder.next = cyclicRemainder
	var bounded error
	bounded = cyclicRemainder
	for depth := range maxURLErrorLogDepth {
		bounded = &url.Error{Op: http.MethodGet, URL: fmt.Sprintf("https://example.com/api?signature=bounded-secret-%d", depth), Err: bounded}
	}
	redacted = urlErrorForLog(bounded)
	assert.ErrorIs(t, redacted, errTruncatedErrorChain, "the depth cap should truncate before traversing a cyclic remainder")
	assert.NotContains(t, redacted.Error(), "bounded-secret", "bounded cyclic errors should not expose URL credentials")

	cyclicURLRemainder := &url.Error{Op: http.MethodGet, URL: "https://example.com/api?signature=remainder-secret"}
	cyclicURLRemainder.Err = cyclicURLRemainder
	bounded = cyclicURLRemainder
	for depth := 1; depth < maxURLErrorLogDepth; depth++ {
		bounded = &url.Error{Op: http.MethodGet, URL: fmt.Sprintf("https://example.com/api?signature=near-secret-%d", depth), Err: bounded}
	}
	redacted = urlErrorForLog(bounded)
	assert.ErrorIs(t, redacted, errTruncatedErrorChain, "a URL cycle just inside the depth cap should be truncated")
	assert.NotContains(t, redacted.Error(), "remainder-secret", "a URL cycle should not expose credentials")

	nonURLCycle := new(cyclicError)
	nonURLCycle.next = nonURLCycle
	result := make(chan error, 1)
	go func() {
		result <- urlErrorForLog(&url.Error{Op: http.MethodGet, URL: "https://example.com/api?signature=cycle-secret", Err: nonURLCycle})
	}()
	select {
	case redacted = <-result:
	case <-time.After(time.Second):
		t.Fatal("a non-URL error cycle must not hang redaction")
	}
	assert.NotContains(t, redacted.Error(), "cycle-secret", "a non-URL cycle should not expose URL credentials")
}

func TestSendPayloadRedactsInvalidSignedURL(t *testing.T) {
	t.Parallel()
	r, err := New("test", &http.Client{})
	require.NoError(t, err, "New must not error")
	err = r.SendPayload(t.Context(), Unset, func() (*Item, error) {
		return &Item{
			Method: http.MethodGet,
			Path:   "https://example.com/v1/accounts/1%zz/balance?AccessKeyId=fake-key&Signature=fake-signature",
		}, nil
	}, AuthenticatedRequest)
	require.ErrorIs(t, err, ErrAuthRequestFailed, "SendPayload must report the failed authenticated request")
	assert.Contains(t, err.Error(), "invalid URL escape", "error should retain the parse failure")
	assert.Contains(t, err.Error(), "AccessKeyId=[REDACTED]", "error should retain the redacted query field")
	assert.Contains(t, err.Error(), "Signature=[REDACTED]", "error should retain the redacted signature field")
	assert.NotContains(t, err.Error(), "fake-key", "error should not expose the access key")
	assert.NotContains(t, err.Error(), "fake-signature", "error should not expose the signature")
}

func TestExecuteRequestVerboseRedactsCredentials(t *testing.T) {
	// Not parallel: the log hook and log config are process-wide, and cleanup returns both to the package defaults.
	var mu sync.Mutex
	var logged strings.Builder
	require.NoError(t, gctlog.SetGlobalLogConfig(gctlog.GenDefaultSettings()), "SetGlobalLogConfig must not error")
	gctlog.SetCustomLogHook(func(_, _ string, a ...any) bool {
		mu.Lock()
		fmt.Fprintln(&logged, a...)
		mu.Unlock()
		return true
	})
	t.Cleanup(func() {
		gctlog.SetCustomLogHook(nil)
		assert.NoError(t, gctlog.SetGlobalLogConfig(&gctlog.Config{}), "SetGlobalLogConfig should not error")
	})

	var calls int
	var sent *http.Request
	var sentBody []byte
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("transport failure")
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		sent, sentBody = req, body
		return &http.Response{
			Status:     "200 OK",
			StatusCode: http.StatusOK,
			Header:     http.Header{"Set-Cookie": []string{"session=secret-cookie"}},
			Body:       io.NopCloser(strings.NewReader(`{"nested":{"password":"response-secret"},"value":1}`)),
			Request:    req,
		}, nil
	})}
	r, err := New(
		"test", httpClient,
		WithBackoff(func(int) time.Duration { return 0 }),
		WithRetryPolicy(func(_ *http.Response, err error) (bool, error) { return err != nil, nil }),
	)
	require.NoError(t, err, "New must not error")

	const path = "https://example.com/api?timestamp=1&signature=secret-signature"
	for attempt := 1; attempt <= 2; attempt++ {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, path, strings.NewReader("key=secret-key&nonce=1"))
		require.NoError(t, err, "NewRequestWithContext must not error")
		if attempt == 1 {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
		}
		req.Header.Set("OK-ACCESS-KEY", "secret-header")
		retry, err := r.executeRequest(t.Context(), &Item{Method: http.MethodPost, Path: path}, req, attempt, true)
		require.NoError(t, err, "executeRequest must not error")
		require.Equal(t, attempt == 1, retry, "executeRequest must retry only the failed attempt")
	}

	require.NotNil(t, sent, "transport must receive the retried request")
	assert.Equal(t, path, sent.URL.String(), "redaction should not change the URL sent")
	assert.Equal(t, "secret-header", sent.Header.Get("OK-ACCESS-KEY"), "redaction should not change the headers sent")
	assert.Equal(t, "key=secret-key&nonce=1", string(sentBody), "redaction should not change the body sent")

	const jsonBody = `{"note":"a&key=b","password":"request-secret","nonce":1}`
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com/api", strings.NewReader(jsonBody))
	require.NoError(t, err, "NewRequestWithContext must not error for a JSON request")
	req.Header.Set("Content-Type", "application/json")
	retry, err := r.executeRequest(t.Context(), &Item{Method: http.MethodPost, Path: "https://example.com/api"}, req, 1, true)
	require.NoError(t, err, "executeRequest must not error for a JSON request")
	assert.False(t, retry, "successful JSON request should not retry")

	mu.Lock()
	out := logged.String()
	mu.Unlock()
	for _, line := range []string{
		`test attempt 1 request path: https://example.com/api?timestamp=1&signature=[REDACTED]`,
		`test request header [Ok-Access-Key]: [[REDACTED]]`,
		`test request body: key=[REDACTED]&nonce=1`,
		`cause: Post "https://example.com/api?timestamp=1&signature=[REDACTED]": transport failure`,
		`test response header [Set-Cookie]: [[REDACTED]]`,
		`{"nested":{"password":"[REDACTED]"},"value":1}`,
		`{"note":"a&key=b","password":"[REDACTED]","nonce":1}`,
	} {
		assert.Containsf(t, out, line, "verbose log should contain %s", line)
	}
	for _, secret := range []string{"secret-signature", "secret-key", "secret-header", "secret-cookie", "request-secret", "response-secret"} {
		assert.NotContainsf(t, out, secret, "verbose log should not contain %s", secret)
	}
}

func TestExecuteRequestBadStatusRedactsCredentials(t *testing.T) {
	t.Parallel()
	contentType := "application/json"
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Status:     "400 Bad Request",
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{contentType}},
			Body:       io.NopCloser(strings.NewReader(`{"password":"response-secret","reason":"invalid"}`)),
			Request:    req,
		}, nil
	})}
	r, err := New("test", httpClient)
	require.NoError(t, err, "New must not error")
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", http.NoBody)
	require.NoError(t, err, "NewRequestWithContext must not error")

	_, err = r.executeRequest(t.Context(), &Item{Method: http.MethodGet, Path: "https://example.com"}, req, 1, false)
	require.ErrorIs(t, err, ErrBadStatus, "executeRequest must return ErrBadStatus")
	assert.Contains(t, err.Error(), `"password":"[REDACTED]"`, "bad status error should retain redacted response structure")
	assert.NotContains(t, err.Error(), "response-secret", "bad status error should not expose response credentials")

	contentType = "application/x-www-form-urlencoded"
	_, err = r.executeRequest(t.Context(), &Item{Method: http.MethodGet, Path: "https://example.com"}, req, 1, false)
	require.ErrorIs(t, err, ErrBadStatus, "form-labelled JSON error must return ErrBadStatus")
	assert.Contains(t, err.Error(), `"password":"[REDACTED]"`, "form-labelled JSON error should retain redacted response structure")
	assert.NotContains(t, err.Error(), "response-secret", "form-labelled JSON error should not expose response credentials")
}

func TestExecuteRequestRecordingFailureRedactsCredentials(t *testing.T) {
	t.Parallel()
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Status:     "200 OK",
			StatusCode: http.StatusOK,
			Header:     http.Header{"Set-Cookie": []string{"session=secret-cookie"}},
			Body:       io.NopCloser(strings.NewReader(`{}`)),
			Request:    req,
		}, nil
	})}
	// No mock file exists for this service, so recording fails.
	r, err := New("recordingfailure", httpClient)
	require.NoError(t, err, "New must not error")
	const path = "https://example.com/api?timestamp=1&signature=secret-signature"
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody)
	require.NoError(t, err, "NewRequestWithContext must not error")
	req.Header.Set("OK-ACCESS-KEY", "secret-header")

	_, err = r.executeRequest(t.Context(), &Item{Method: http.MethodGet, Path: path, HTTPRecording: true}, req, 1, false)
	require.ErrorContains(t, err, "mock recording failure", "executeRequest must return the recording failure")
	assert.Contains(t, err.Error(), "GET https://example.com/api?timestamp=1&signature=[REDACTED]", "recording failure should name the redacted request")
	for _, secret := range []string{"secret-signature", "secret-header", "secret-cookie"} {
		assert.NotContainsf(t, err.Error(), secret, "recording failure should not contain %s", secret)
	}
}

func TestExecuteRequestHTTPDebuggingRedactsCredentials(t *testing.T) {
	// Not parallel: the log hook and log config are process-wide, and cleanup returns both to the package defaults.
	var mu sync.Mutex
	var logged strings.Builder
	require.NoError(t, gctlog.SetGlobalLogConfig(gctlog.GenDefaultSettings()), "SetGlobalLogConfig must not error")
	gctlog.SetCustomLogHook(func(_, _ string, a ...any) bool {
		mu.Lock()
		fmt.Fprintln(&logged, a...)
		mu.Unlock()
		return true
	})
	t.Cleanup(func() {
		gctlog.SetCustomLogHook(nil)
		assert.NoError(t, gctlog.SetGlobalLogConfig(&gctlog.Config{}), "SetGlobalLogConfig should not error")
	})

	var sentBody []byte
	var sentContentTypes []string
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		sentContentTypes = append(sentContentTypes, req.Header.Get("Content-Type"))
		var err error
		sentBody, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		return &http.Response{
			Status:     "200 OK",
			StatusCode: http.StatusOK,
			Header:     http.Header{"Set-Cookie": []string{"session=secret-cookie"}},
			Body:       io.NopCloser(strings.NewReader(`{"secret":"response-secret","value":1}`)),
			Request:    req,
		}, nil
	})}
	r, err := New("test", httpClient)
	require.NoError(t, err, "New must not error")

	const path = "https://example.com/api?timestamp=1&signature=secret-signature"
	responseHeaders := make(http.Header)
	err = r.SendPayload(t.Context(), Unset, func() (*Item, error) {
		return &Item{
			Method:         http.MethodPost,
			Path:           path,
			Body:           struct{ io.Reader }{strings.NewReader("key=secret-key&nonce=1")},
			HeaderResponse: &responseHeaders,
			HTTPDebugging:  true,
		}, nil
	}, UnauthenticatedRequest)
	require.NoError(t, err, "SendPayload must not error")
	assert.Equal(t, "key=secret-key&nonce=1", string(sentBody), "dumping the request should not change the body sent")
	assert.Equal(t, "session=secret-cookie", responseHeaders.Get("Set-Cookie"), "HeaderResponse should retain the unredacted response headers")

	const itemHeaderJSON = `{"note":"a&key=b","nonce":1}`
	err = r.SendPayload(t.Context(), Unset, func() (*Item, error) {
		return &Item{
			Method:        http.MethodPost,
			Path:          "https://example.com/api",
			Body:          strings.NewReader(itemHeaderJSON),
			Headers:       map[string]string{"Content-Type": "application/json"},
			HTTPDebugging: true,
		}, nil
	}, UnauthenticatedRequest)
	require.NoError(t, err, "SendPayload must not error with an Item content type")

	const contextHeaderJSON = `{"note":"context&key=b","nonce":2}`
	ctx := WithHeaders(t.Context(), http.Header{"Content-Type": []string{"application/json"}})
	err = r.SendPayload(ctx, Unset, func() (*Item, error) {
		return &Item{
			Method:        http.MethodPost,
			Path:          "https://example.com/api",
			Body:          strings.NewReader(contextHeaderJSON),
			Headers:       map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			HTTPDebugging: true,
		}, nil
	}, UnauthenticatedRequest)
	require.NoError(t, err, "SendPayload must not error with a context content type override")

	const lowerCaseJSON = `{"note":"lowercase&key=b","nonce":3}`
	ctx = WithHeaders(t.Context(), http.Header{"content-type": []string{"application/json"}})
	err = r.SendPayload(ctx, Unset, func() (*Item, error) {
		return &Item{
			Method:        http.MethodPost,
			Path:          "https://example.com/api",
			Body:          strings.NewReader(lowerCaseJSON),
			Headers:       map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			HTTPDebugging: true,
		}, nil
	}, UnauthenticatedRequest)
	require.NoError(t, err, "SendPayload must not error with a lowercase context content type override")

	const lowerCaseForm = "key=LOWERCASEFORMSECRET&nonce=4"
	ctx = WithHeaders(t.Context(), http.Header{"content-type": []string{"application/x-www-form-urlencoded"}})
	err = r.SendPayload(ctx, Unset, func() (*Item, error) {
		return &Item{
			Method:        http.MethodPost,
			Path:          "https://example.com/api",
			Body:          strings.NewReader(lowerCaseForm),
			Headers:       map[string]string{"Content-Type": "application/json"},
			HTTPDebugging: true,
		}, nil
	}, UnauthenticatedRequest)
	require.NoError(t, err, "SendPayload must not error with a lowercase form content type override")

	const emptyOverrideForm = "key=EMPTYOVERRIDESECRET&nonce=5"
	ctx = WithHeaders(t.Context(), http.Header{"Content-Type": []string{""}})
	err = r.SendPayload(ctx, Unset, func() (*Item, error) {
		return &Item{
			Method:        http.MethodPost,
			Path:          "https://example.com/api",
			Body:          strings.NewReader(emptyOverrideForm),
			Headers:       map[string]string{"Content-Type": "application/json"},
			HTTPDebugging: true,
		}, nil
	}, UnauthenticatedRequest)
	require.NoError(t, err, "SendPayload must not error with an empty context content type override")

	const duplicateHeaderJSON = `{"password":"DUPLICATEHEADERSECRET","nonce":6}`
	err = r.SendPayload(t.Context(), Unset, func() (*Item, error) {
		return &Item{
			Method: http.MethodPost,
			Path:   "https://example.com/api",
			Body:   strings.NewReader(duplicateHeaderJSON),
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"content-type": "application/json",
			},
			HTTPDebugging: true,
		}, nil
	}, UnauthenticatedRequest)
	require.NoError(t, err, "SendPayload must not error with duplicate content type spellings")
	require.Equal(t, []string{
		"",
		"application/json",
		"application/json",
		"application/json",
		"application/x-www-form-urlencoded",
		"",
		"application/json",
	}, sentContentTypes, "the transport must receive each effective content type")

	mu.Lock()
	out := logged.String()
	mu.Unlock()
	for _, line := range []string{
		`POST /api?timestamp=1&signature=[REDACTED] HTTP/1.1`,
		`key=[REDACTED]&nonce=1`,
		`DumpResponse (https://example.com/api?timestamp=1&signature=[REDACTED])`,
		`DumpResponse Body (https://example.com/api?timestamp=1&signature=[REDACTED])`,
		`Set-Cookie: [REDACTED]`,
		itemHeaderJSON,
		contextHeaderJSON,
		lowerCaseJSON,
		"key=[REDACTED]&nonce=4",
		"key=[REDACTED]&nonce=5",
		`{"password":"[REDACTED]","nonce":6}`,
		`{"secret":"[REDACTED]","value":1}`,
	} {
		assert.Containsf(t, out, line, "HTTPDebugging log should contain %s", line)
	}
	for _, secret := range []string{"secret-signature", "secret-key", "secret-cookie", "LOWERCASEFORMSECRET", "EMPTYOVERRIDESECRET", "DUPLICATEHEADERSECRET", "response-secret"} {
		assert.NotContainsf(t, out, secret, "HTTPDebugging log should not contain %s", secret)
	}
}

func TestHTTPDebuggingDumpFailureDoesNotAbortRequest(t *testing.T) {
	t.Parallel()

	var called bool
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		called = true
		return &http.Response{
			Status:     "200 OK",
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       http.NoBody,
			Request:    req,
		}, nil
	})}
	r, err := New("test", httpClient)
	require.NoError(t, err, "New must not error")

	err = r.SendPayload(t.Context(), Unset, func() (*Item, error) {
		return &Item{Method: http.MethodGet, Path: "custom://example.com/api", HTTPDebugging: true}, nil
	}, UnauthenticatedRequest)
	require.NoError(t, err, "SendPayload must not fail when only the debug dump fails")
	assert.True(t, called, "SendPayload should execute the request when the debug dump fails")
}

func TestExecuteRequestVerboseRequestBody(t *testing.T) {
	t.Parallel()

	getBodyErr := errors.New("get request body failure")
	readErr := errors.New("read request body failure")
	closeErr := errors.New("close request body failure")

	for _, tc := range []struct {
		name                       string
		body                       io.Reader
		getBodyErr                 error
		closeErr                   error
		expectedErr                error
		expectedCloseCalls         int
		expectedTransportCalls     int
		expectedResponseCloseCalls int
	}{
		{
			name:        "get body failure",
			getBodyErr:  getBodyErr,
			expectedErr: getBodyErr,
		},
		{
			name:               "read body failure",
			body:               iotest.ErrReader(readErr),
			expectedErr:        readErr,
			expectedCloseCalls: 1,
		},
		{
			name:                       "close body failure",
			body:                       strings.NewReader(`{"request":true}`),
			closeErr:                   closeErr,
			expectedCloseCalls:         1,
			expectedTransportCalls:     1,
			expectedResponseCloseCalls: 1,
		},
		{
			name:                       "successful body copy",
			body:                       strings.NewReader(`{"request":true}`),
			expectedCloseCalls:         1,
			expectedTransportCalls:     1,
			expectedResponseCloseCalls: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			requestBody := &trackedReadCloser{
				Reader:   tc.body,
				closeErr: tc.closeErr,
			}
			responseBody := &trackedReadCloser{Reader: strings.NewReader(`{"response":true}`)}
			transportCalls := 0
			httpClient := &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					transportCalls++
					return &http.Response{
						Status:     "200 OK",
						StatusCode: http.StatusOK,
						ProtoMajor: 1,
						ProtoMinor: 1,
						Header:     make(http.Header),
						Body:       responseBody,
						Request:    req,
					}, nil
				}),
			}
			r, err := New("test", httpClient)
			require.NoError(t, err, "New must not error")
			t.Cleanup(func() {
				assert.NoError(t, r.Shutdown(), "Shutdown should not error")
			})

			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com", http.NoBody)
			require.NoError(t, err, "http.NewRequestWithContext must not error")
			req.Header.Set("X-Test", "value")
			req.GetBody = func() (io.ReadCloser, error) {
				if tc.getBodyErr != nil {
					return nil, tc.getBodyErr
				}
				return requestBody, nil
			}
			retry, err := r.executeRequest(t.Context(), &Item{
				Method: http.MethodPost,
				Path:   "https://example.com",
			}, req, 1, true)
			require.False(t, retry, "executeRequest must not retry")
			if tc.expectedErr == nil {
				require.NoError(t, err, "executeRequest must not error")
			} else {
				require.ErrorIs(t, err, tc.expectedErr, "executeRequest must return the expected error")
			}
			assert.Equal(t, tc.expectedCloseCalls, requestBody.closeCalls, "executeRequest should close requestBody the expected number of times")
			assert.Equal(t, tc.expectedTransportCalls, transportCalls, "executeRequest should execute the transport the expected number of times")
			assert.Equal(t, tc.expectedResponseCloseCalls, responseBody.closeCalls, "executeRequest should close the response body the expected number of times")
		})
	}
}

func TestEvaluateRetryRedactsURLOnAllReturns(t *testing.T) {
	t.Parallel()
	transportErr := errors.New("transport failure")
	credentialErr := &url.Error{Op: http.MethodGet, URL: "https://example.com?signature=returned-secret", Err: transportErr}
	tests := []struct {
		name            string
		retryNotAllowed bool
		defaultPolicy   bool
		policyRetries   bool
		maxRetries      int
		deadline        bool
	}{
		{name: "retry not allowed", retryNotAllowed: true},
		{name: "default policy declines non-timeout", defaultPolicy: true},
		{name: "custom policy declines"},
		{name: "maximum retries", policyRetries: true},
		{name: "deadline prevents retry", policyRetries: true, maxRetries: 1, deadline: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := Requester{maxRetries: tc.maxRetries, backoff: func(int) time.Duration { return time.Millisecond }}
			if tc.defaultPolicy {
				r.retryPolicy = DefaultRetryPolicy
			} else {
				r.retryPolicy = func(*http.Response, error) (bool, error) { return tc.policyRetries, nil }
			}
			ctx := t.Context()
			if tc.retryNotAllowed {
				ctx = WithRetryNotAllowed(ctx)
			}
			if tc.deadline {
				var cancel context.CancelFunc
				ctx, cancel = context.WithDeadline(ctx, time.Now())
				defer cancel()
			}
			retry, err := r.evaluateRetry(ctx, nil, credentialErr, 1, false)
			require.ErrorIs(t, err, transportErr, "evaluateRetry must preserve the transport cause")
			assert.False(t, retry, "evaluateRetry should not retry on this return path")
			assert.NotContains(t, err.Error(), "returned-secret", "evaluateRetry should not return URL credentials")
			assert.Contains(t, err.Error(), "signature=[REDACTED]", "evaluateRetry should retain redacted URL structure")
		})
	}
}
