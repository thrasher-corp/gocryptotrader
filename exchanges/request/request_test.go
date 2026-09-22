package request

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
)

const unexpected = "unexpected values"

var (
	testURL     string
	serverLimit *RateLimiterWithWeight

	sensitiveLogKeys = []string{
		"Authorization",
		"Cookie",
		"Key",
		"OK-ACCESS-KEY",
		"OK-ACCESS-PASSPHRASE",
		"X-BAPI-SIGN",
		"Btse-Api",
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
)

func TestHeaderValuesForLog(t *testing.T) {
	t.Parallel()

	values := []string{"sensitive-value"}
	for _, header := range sensitiveLogKeys {
		assert.Equalf(t, []string{"[REDACTED]"}, headerValuesForLog(header, values), "%s should be redacted", header)
	}
	for _, header := range []string{"Content-Type", "KC-API-KEY-VERSION", "SignatureMethod", "signTimestamp"} {
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

func TestRedactEncodedValuesFailsClosed(t *testing.T) {
	t.Parallel()
	redacted := redactEncodedValues("%ZZ=secret-value&nonce=1")
	expected := "%ZZ=[REDACTED]&nonce=1"
	assert.Equal(t, expected, redacted, "a malformed field name should have its value redacted")
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
	assert.Contains(t, string(dump), `{"nested":{"passphrase":"[REDACTED]"},"note":"a\u0026key=b"}`, "request dump should redact nested JSON credentials")
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
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (r roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
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
	r, err := New("test", httpClient,
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
		`{"nonce":1,"note":"a\u0026key=b","password":"[REDACTED]"}`,
	} {
		assert.Containsf(t, out, line, "verbose log should contain %s", line)
	}
	for _, secret := range []string{"secret-signature", "secret-key", "secret-header", "secret-cookie", "request-secret", "response-secret"} {
		assert.NotContainsf(t, out, secret, "verbose log should not contain %s", secret)
	}
}

func TestExecuteRequestBadStatusRedactsCredentials(t *testing.T) {
	t.Parallel()
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Status:     "400 Bad Request",
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
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
	}, sentContentTypes, "the transport should receive each effective content type")

	mu.Lock()
	out := logged.String()
	mu.Unlock()
	for _, line := range []string{
		`POST /api?timestamp=1&signature=[REDACTED] HTTP/1.1`,
		`key=[REDACTED]&nonce=1`,
		`DumpResponse (https://example.com/api?timestamp=1&signature=[REDACTED])`,
		`DumpResponse Body (https://example.com/api?timestamp=1&signature=[REDACTED])`,
		`Set-Cookie: [REDACTED]`,
		`{"nonce":1,"note":"a\u0026key=b"}`,
		`{"nonce":2,"note":"context\u0026key=b"}`,
		`{"nonce":3,"note":"lowercase\u0026key=b"}`,
		"key=[REDACTED]&nonce=4",
		"key=[REDACTED]&nonce=5",
		`{"nonce":6,"password":"[REDACTED]"}`,
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

type trackedReadCloser struct {
	io.Reader
	closeErr   error
	closeCalls int
}

func (t *trackedReadCloser) Close() error {
	t.closeCalls++
	return t.closeErr
}

type trackingReporter struct {
	calls int
}

func (t *trackingReporter) Latency(string, string, string, time.Duration) {
	t.calls++
}

func TestRoundTripFuncRoundTrip(t *testing.T) {
	t.Parallel()
	expectedErr := errors.New("transport failure")
	for _, tc := range []struct {
		name             string
		expectedResponse *http.Response
		expectedErr      error
	}{
		{
			name:             "response",
			expectedResponse: &http.Response{StatusCode: http.StatusTeapot, Body: http.NoBody},
		},
		{
			name:        "error",
			expectedErr: expectedErr,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", http.NoBody)
			var receivedRequest *http.Request
			transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
				receivedRequest = req
				return tc.expectedResponse, tc.expectedErr
			})

			response, err := transport.RoundTrip(req)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr, "RoundTrip must return the transport error")
			} else {
				require.NoError(t, err, "RoundTrip must not error for a transport response")
			}
			assert.Same(t, tc.expectedResponse, response, "RoundTrip should return the transport response")
			assert.Same(t, req, receivedRequest, "RoundTrip should pass the request to the transport")
			if response != nil {
				require.NoError(t, response.Body.Close(), "response.Body.Close must not error")
			}
		})
	}
}

func TestTrackedReadCloserClose(t *testing.T) {
	t.Parallel()
	closeErr := errors.New("close failure")
	for _, tc := range []struct {
		name     string
		closeErr error
	}{
		{
			name: "success",
		},
		{
			name:     "error",
			closeErr: closeErr,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			closer := &trackedReadCloser{closeErr: tc.closeErr}
			err := closer.Close()
			if tc.closeErr != nil {
				require.ErrorIs(t, err, tc.closeErr, "Close must return the configured error")
			} else {
				require.NoError(t, err, "Close must not error without a configured error")
			}
			assert.Equal(t, 1, closer.closeCalls, "Close should increment closeCalls exactly once")
		})
	}
}

func TestTrackingReporterLatency(t *testing.T) {
	t.Parallel()
	reporter := new(trackingReporter)
	reporter.Latency("exchange", http.MethodGet, "/path", time.Second)
	assert.Equal(t, 1, reporter.calls, "Latency should increment calls exactly once")
}

func TestMain(m *testing.M) {
	serverLimitInterval := time.Millisecond * 500
	serverLimit = NewWeightedRateLimitByDuration(serverLimitInterval)
	serverLimitRetry := NewWeightedRateLimitByDuration(serverLimitInterval)
	sm := http.NewServeMux()
	sm.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"response":true}`)
		if err != nil {
			log.Fatal(err)
		}
	})
	sm.HandleFunc("/error", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err := io.WriteString(w, `{"error":true}`)
		if err != nil {
			log.Fatal(err)
		}
	})
	sm.HandleFunc("/timeout", func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(time.Millisecond * 100)
		w.WriteHeader(http.StatusGatewayTimeout)
	})
	sm.HandleFunc("/rate", func(w http.ResponseWriter, _ *http.Request) {
		if !serverLimit.limiter.Allow() {
			http.Error(w,
				http.StatusText(http.StatusTooManyRequests),
				http.StatusTooManyRequests)
			_, err := io.WriteString(w, `{"response":false}`)
			if err != nil {
				log.Fatal(err)
			}
			return
		}
		_, err := io.WriteString(w, `{"response":true}`)
		if err != nil {
			log.Fatal(err)
		}
	})
	sm.HandleFunc("/rate-retry", func(w http.ResponseWriter, _ *http.Request) {
		if !serverLimitRetry.limiter.Allow() {
			w.Header().Add("Retry-After", strconv.Itoa(int(math.Round(serverLimitInterval.Seconds()))))
			http.Error(w,
				http.StatusText(http.StatusTooManyRequests),
				http.StatusTooManyRequests)
			_, err := io.WriteString(w, `{"response":false}`)
			if err != nil {
				log.Fatal(err)
			}
			return
		}
		_, err := io.WriteString(w, `{"response":true}`)
		if err != nil {
			log.Fatal(err)
		}
	})
	sm.HandleFunc("/always-retry", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Retry-After", time.Now().Format(time.RFC1123))
		w.WriteHeader(http.StatusTooManyRequests)
		_, err := io.WriteString(w, `{"response":false}`)
		if err != nil {
			log.Fatal(err)
		}
	})
	sm.HandleFunc("/nocontent", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	server := httptest.NewServer(sm)
	testURL = server.URL
	issues := m.Run()
	server.Close()
	os.Exit(issues)
}

func TestCheckRequest(t *testing.T) {
	t.Parallel()

	r, err := New("TestRequest",
		new(http.Client))
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()

	var check *Item
	_, err = check.validateRequest(ctx, &Requester{})
	if err == nil {
		t.Fatal(unexpected)
	}

	_, err = check.validateRequest(ctx, nil)
	if err == nil {
		t.Fatal(unexpected)
	}

	_, err = check.validateRequest(ctx, r)
	if err == nil {
		t.Fatal(unexpected)
	}

	check = &Item{}
	_, err = check.validateRequest(ctx, r)
	if err == nil {
		t.Fatal(unexpected)
	}

	check.Path = testURL
	check.Method = " " // Forces method check; "" automatically converts to GET
	_, err = check.validateRequest(ctx, r)
	if err == nil {
		t.Fatal(unexpected)
	}

	check.Method = http.MethodPost
	_, err = check.validateRequest(ctx, r)
	if err != nil {
		t.Fatal(err)
	}

	var passback http.Header
	check.HeaderResponse = &passback
	_, err = check.validateRequest(ctx, r)
	if err == nil {
		t.Fatal("expected error when underlying memory is not allocated")
	}
	passback = http.Header{}

	// Test setting headers
	check.Headers = map[string]string{
		"Content-Type": "Super awesome HTTP party experience",
		"preserved":    "true",
	}

	// Test user agent set
	r.userAgent = "r00t axxs"
	req, err := check.validateRequest(ctx, r)
	if err != nil {
		t.Fatal(err)
	}

	if req.Header.Get("Content-Type") != "Super awesome HTTP party experience" {
		t.Fatal(unexpected)
	}

	if req.UserAgent() != "r00t axxs" {
		t.Fatal(unexpected)
	}

	ctx = WithHeaders(ctx, http.Header{
		"Content-Type": {"context override"},
		"User-Agent":   {"context agent"},
		"X-Values":     {"one", "two"},
	})
	req, err = check.validateRequest(ctx, r)
	require.NoError(t, err, "validateRequest must not error")
	assert.Equal(t, "context override", req.Header.Get("Content-Type"), "context header should override item header")
	assert.Equal(t, "context agent", req.UserAgent(), "context header should override requester user agent")
	assert.Equal(t, []string{"one", "two"}, req.Header.Values("X-Values"), "context header values should be preserved")
	assert.Equal(t, "true", req.Header.Get("preserved"), "item header should be preserved when not overridden by context")
}

var globalshell = RateLimitDefinitions{
	Auth:   NewWeightedRateLimitByDuration(time.Millisecond * 600),
	UnAuth: NewRateLimitWithWeight(time.Second*1, 100, 1),
}

func TestDoRequest(t *testing.T) {
	t.Parallel()
	r, err := New("test", new(http.Client), WithLimiter(globalshell))
	require.NoError(t, err, "New requester must not error")

	ctx := t.Context()
	err = (*Requester)(nil).SendPayload(ctx, Unset, nil, UnauthenticatedRequest)
	require.ErrorIs(t, err, ErrRequestSystemIsNil)

	err = r.SendPayload(ctx, Unset, nil, UnauthenticatedRequest)
	require.ErrorIs(t, err, errRequestFunctionIsNil)

	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) { return nil, nil }, UnauthenticatedRequest)
	require.ErrorIs(t, err, errRequestItemNil)

	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) { return &Item{}, nil }, UnauthenticatedRequest)
	require.ErrorIs(t, err, errInvalidPath)

	var nilHeader http.Header
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return &Item{
			Path:           testURL,
			HeaderResponse: &nilHeader,
		}, nil
	}, UnauthenticatedRequest)
	require.ErrorIs(t, err, errHeaderResponseMapIsNil)

	// Invalid/missing endpoint limit
	err = r.SendPayload(ctx, Unset, func() (*Item, error) { return &Item{Path: testURL}, nil }, UnauthenticatedRequest)
	require.ErrorIs(t, err, common.ErrNilPointer)

	// Force debug
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return &Item{
			Path: testURL,
			Headers: map[string]string{
				"test": "supertest",
			},
			Body:          strings.NewReader("test"),
			HTTPDebugging: true,
			Verbose:       true,
		}, nil
	}, UnauthenticatedRequest)
	require.NoError(t, err, "SendPayload must not error")

	// Fail new request call
	newError := errors.New("request item failure")
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return nil, newError
	}, UnauthenticatedRequest)
	require.ErrorIs(t, err, newError)

	r._HTTPClient, err = newProtectedClient(common.NewHTTPClientWithTimeout(0))
	require.NoError(t, err, "newProtectedClient must not error")

	// timeout checker
	err = r._HTTPClient.setHTTPClientTimeout(time.Millisecond * 50)
	require.NoError(t, err, "setHTTPClientTimeout must not error")
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return &Item{Path: testURL + "/timeout"}, nil
	}, UnauthenticatedRequest)
	require.ErrorIs(t, err, errFailedToRetryRequest)
	// reset timeout
	err = r._HTTPClient.setHTTPClientTimeout(0)
	require.NoError(t, err, "setHTTPClientTimeout must not error")

	// Check JSON
	var resp struct {
		Response bool `json:"response"`
	}

	// Check header contents
	passback := http.Header{}
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return &Item{
			Method:         http.MethodGet,
			Path:           testURL,
			Result:         &resp,
			HeaderResponse: &passback,
		}, nil
	}, UnauthenticatedRequest)
	require.NoError(t, err, "SendPayload must not error")

	require.Equal(t, "17", passback.Get("Content-Length"), "Content-Length must have the correct value")
	require.Equal(t, "application/json", passback.Get("Content-Type"), "Content-Type must have the correct value")

	// Check error
	var respErr struct {
		Error bool `json:"error"`
	}
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return &Item{
			Method: http.MethodGet,
			Path:   testURL,
			Result: &respErr,
		}, nil
	}, UnauthenticatedRequest)
	require.NoError(t, err, "SendPayload must not error")
	require.False(t, respErr.Error, "Error must be false")

	// Check client side rate limit
	var ec common.ErrorCollector
	for range 5 {
		ec.Go(func() error {
			var resp struct {
				Response bool `json:"response"`
			}
			if err := r.SendPayload(ctx, Auth, func() (*Item, error) {
				return &Item{
					Method: http.MethodGet,
					Path:   testURL + "/rate",
					Result: &resp,
				}, nil
			}, AuthenticatedRequest); err != nil {
				return fmt.Errorf("SendPayload error: %w", err)
			}
			if !resp.Response {
				return fmt.Errorf("unexpected response: %+v", resp)
			}
			return nil
		})
	}

	require.NoError(t, ec.Collect(), "Collect must return no errors")
}

func TestExecuteRequestClosesResponseBody(t *testing.T) {
	t.Parallel()

	readErr := errors.New("response read failure")
	closeErr := errors.New("response close failure")
	retryErr := errors.New("retry policy failure")

	for _, tc := range []struct {
		name          string
		statusCode    int
		body          io.Reader
		closeErr      error
		retryPolicy   RetryPolicy
		expectedErr   error
		errContains   string
		headers       http.Header
		trailer       http.Header
		transfer      []string
		record        bool
		debug         bool
		verbose       bool
		report        bool
		attempt       int
		expectedRetry bool
	}{
		{
			name:       "successful response",
			statusCode: http.StatusOK,
			body:       strings.NewReader(`{"response":true}`),
		},
		{
			name:        "unsuccessful response",
			statusCode:  http.StatusBadRequest,
			body:        strings.NewReader(`{"error":true}`),
			expectedErr: ErrBadStatus,
		},
		{
			name:        "response read failure",
			statusCode:  http.StatusOK,
			body:        iotest.ErrReader(readErr),
			expectedErr: readErr,
		},
		{
			name:       "response close failure",
			statusCode: http.StatusOK,
			body:       strings.NewReader(`{"response":true}`),
			closeErr:   closeErr,
		},
		{
			name:       "retry policy failure",
			statusCode: http.StatusOK,
			body:       strings.NewReader(`{"response":true}`),
			retryPolicy: func(*http.Response, error) (bool, error) {
				return false, retryErr
			},
			expectedErr: retryErr,
		},
		{
			name:        "mock recording failure",
			statusCode:  http.StatusOK,
			body:        strings.NewReader(`{"response":true}`),
			record:      true,
			errContains: "mock recording failure",
		},
		{
			name:       "debug response dump failure",
			statusCode: http.StatusOK,
			body:       strings.NewReader(`{"response":true}`),
			trailer: http.Header{
				"Content-Length": {"1"},
			},
			transfer: []string{"chunked"},
			debug:    true,
			verbose:  true,
		},
		{
			name:       "verbose response with latency reporter",
			statusCode: http.StatusOK,
			body:       strings.NewReader(`{"response":true}`),
			headers: http.Header{
				"X-Test": {"value"},
			},
			verbose: true,
			report:  true,
		},
		{
			name:          "retry response",
			statusCode:    http.StatusTooManyRequests,
			body:          strings.NewReader(`{"retry":true}`),
			expectedRetry: true,
		},
		{
			name:        "terminal retry response",
			statusCode:  http.StatusTooManyRequests,
			body:        strings.NewReader(`{"retry":true}`),
			attempt:     MaxRetryAttempts + 1,
			expectedErr: errFailedToRetryRequest,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body := &trackedReadCloser{
				Reader:   tc.body,
				closeErr: tc.closeErr,
			}
			httpClient := &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						Status:           fmt.Sprintf("%d %s", tc.statusCode, http.StatusText(tc.statusCode)),
						StatusCode:       tc.statusCode,
						ProtoMajor:       1,
						ProtoMinor:       1,
						Header:           tc.headers.Clone(),
						Trailer:          tc.trailer.Clone(),
						TransferEncoding: slices.Clone(tc.transfer),
						Body:             body,
						Request:          req,
					}, nil
				}),
			}
			options := []RequesterOption{WithBackoff(func(int) time.Duration { return 0 })}
			if tc.retryPolicy != nil {
				options = append(options, WithRetryPolicy(tc.retryPolicy))
			}
			reporter := &trackingReporter{}
			if tc.report {
				options = append(options, WithReporter(reporter))
			}
			requesterName := "test"
			if tc.record {
				requesterName = ""
			}
			r, err := New(requesterName, httpClient, options...)
			require.NoError(t, err, "New must not error")
			t.Cleanup(func() {
				assert.NoError(t, r.Shutdown(), "Shutdown should not error")
			})

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", http.NoBody)
			require.NoError(t, err, "http.NewRequestWithContext must not error")
			attempt := tc.attempt
			if attempt == 0 {
				attempt = 1
			}
			retry, err := r.executeRequest(t.Context(), &Item{
				Method:        http.MethodGet,
				Path:          "https://example.com",
				HTTPRecording: tc.record,
				HTTPDebugging: tc.debug,
			}, req, attempt, tc.verbose)
			require.Equal(t, tc.expectedRetry, retry, "executeRequest must return the expected retry decision")
			switch {
			case tc.expectedErr != nil:
				require.ErrorIs(t, err, tc.expectedErr, "executeRequest must return the expected error")
			case tc.errContains != "":
				require.ErrorContains(t, err, tc.errContains, "executeRequest must return the expected error")
			default:
				require.NoError(t, err, "executeRequest must not error")
			}
			assert.Equal(t, 1, body.closeCalls, "executeRequest should close body exactly once")
			if tc.report {
				assert.Equal(t, 1, reporter.calls, "executeRequest should report one latency observation")
			}
		})
	}
}

func TestExecuteRequestRecordsResponse(t *testing.T) {
	serviceDir, err := os.MkdirTemp("..", "request_test_http_recording_") //nolint:usetesting // mock.HTTPRecord requires a sibling fixture directory.
	require.NoError(t, err, "os.MkdirTemp must create the recording service directory")
	recordingDir := filepath.Join(serviceDir, "testdata")
	recordingFile := filepath.Join(recordingDir, "http.json")
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(serviceDir), "os.RemoveAll must remove the recording service directory")
	})
	require.NoError(t, os.Mkdir(recordingDir, 0o755), "os.Mkdir must create the recording testdata directory")
	require.NoError(t, os.WriteFile(recordingFile, []byte(`{"routes":null}`), 0o600), "os.WriteFile must create the recording fixture")

	service := filepath.Base(serviceDir)
	body := &trackedReadCloser{Reader: strings.NewReader(`{"response":true}`)}
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Status:     "200 OK",
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       body,
				Request:    req,
			}, nil
		}),
	}
	r, err := New(service, httpClient)
	require.NoError(t, err, "New must not error")
	t.Cleanup(func() {
		assert.NoError(t, r.Shutdown(), "Shutdown should not error")
	})

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com/record", http.NoBody)
	require.NoError(t, err, "http.NewRequestWithContext must not error")
	retry, err := r.executeRequest(t.Context(), &Item{
		Method:        http.MethodGet,
		Path:          "https://example.com/record",
		HTTPRecording: true,
	}, req, 1, false)
	require.NoError(t, err, "executeRequest must not error")
	require.False(t, retry, "executeRequest must not retry")
	assert.Equal(t, 1, body.closeCalls, "executeRequest should close body exactly once")
	recorded, err := os.ReadFile(recordingFile)
	require.NoError(t, err, "os.ReadFile must read the recording fixture")
	assert.Contains(t, string(recorded), `"/record"`, "recorded should contain the request path")
}

func TestDoRequestExecutesRetryDecision(t *testing.T) {
	t.Parallel()

	attempts := 0
	bodies := make([]*trackedReadCloser, 0, 2)
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			statusCode := http.StatusOK
			if attempts == 1 {
				statusCode = http.StatusTooManyRequests
			}
			body := &trackedReadCloser{Reader: strings.NewReader(`{"response":true}`)}
			bodies = append(bodies, body)
			return &http.Response{
				Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
				StatusCode: statusCode,
				Header:     make(http.Header),
				Body:       body,
				Request:    req,
			}, nil
		}),
	}
	r, err := New("test", httpClient, WithBackoff(func(int) time.Duration { return 0 }))
	require.NoError(t, err, "New must not error")
	t.Cleanup(func() {
		assert.NoError(t, r.Shutdown(), "Shutdown should not error")
	})

	err = r.doRequest(t.Context(), Unset, func() (*Item, error) {
		return &Item{
			Method:       http.MethodGet,
			Path:         "https://example.com",
			NonceEnabled: true,
		}, nil
	})
	require.NoError(t, err, "doRequest must not error")
	require.Len(t, bodies, 2, "bodies must contain one entry per request attempt")
	for _, body := range bodies {
		assert.Equal(t, 1, body.closeCalls, "doRequest should close body exactly once")
	}
}

func TestDoRequestReturnsExecutionError(t *testing.T) {
	t.Parallel()

	retryErr := errors.New("retry policy failure")
	body := &trackedReadCloser{Reader: strings.NewReader(`{"response":true}`)}
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Status:     "200 OK",
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       body,
				Request:    req,
			}, nil
		}),
	}
	r, err := New("test", httpClient, WithRetryPolicy(func(*http.Response, error) (bool, error) {
		return false, retryErr
	}))
	require.NoError(t, err, "New must not error")
	t.Cleanup(func() {
		assert.NoError(t, r.Shutdown(), "Shutdown should not error")
	})

	err = r.doRequest(t.Context(), Unset, func() (*Item, error) {
		return &Item{
			Method:       http.MethodGet,
			Path:         "https://example.com",
			NonceEnabled: true,
		}, nil
	})
	require.ErrorIs(t, err, retryErr, "doRequest must return the execution error")
	assert.Equal(t, 1, body.closeCalls, "doRequest should close body exactly once")
}

func TestExecuteRequestProcessesResponse(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name           string
		statusCode     int
		payload        string
		initialResult  bool
		expectedResult bool
		debug          bool
		expectError    bool
	}{
		{
			name:           "decoded response with headers and debugging",
			statusCode:     http.StatusOK,
			payload:        `{"response":true}`,
			expectedResult: true,
			debug:          true,
		},
		{
			name:           "no content preserves result",
			statusCode:     http.StatusNoContent,
			initialResult:  true,
			expectedResult: true,
		},
		{
			name:        "invalid response",
			statusCode:  http.StatusOK,
			payload:     `{`,
			expectError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body := &trackedReadCloser{Reader: strings.NewReader(tc.payload)}
			httpClient := &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						Status:     fmt.Sprintf("%d %s", tc.statusCode, http.StatusText(tc.statusCode)),
						StatusCode: tc.statusCode,
						Header: http.Header{
							"X-Test": {"value"},
						},
						Body:    body,
						Request: req,
					}, nil
				}),
			}
			r, err := New("test", httpClient)
			require.NoError(t, err, "New must not error")
			t.Cleanup(func() {
				assert.NoError(t, r.Shutdown(), "Shutdown should not error")
			})

			result := struct {
				Response bool `json:"response"`
			}{Response: tc.initialResult}
			headerResponse := make(http.Header)
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", http.NoBody)
			require.NoError(t, err, "http.NewRequestWithContext must not error")
			retry, err := r.executeRequest(t.Context(), &Item{
				Method:         http.MethodGet,
				Path:           "https://example.com",
				Result:         &result,
				HeaderResponse: &headerResponse,
				HTTPDebugging:  tc.debug,
			}, req, 1, false)
			require.False(t, retry, "executeRequest must not retry")
			if tc.expectError {
				require.Error(t, err, "executeRequest must return the response decoding error")
			} else {
				require.NoError(t, err, "executeRequest must not error")
			}
			assert.Equal(t, tc.expectedResult, result.Response, "executeRequest should return the expected response result")
			assert.Equal(t, []string{"value"}, headerResponse.Values("X-Test"), "executeRequest should return the expected response headers")
			assert.Equal(t, 1, body.closeCalls, "executeRequest should close body exactly once")
		})
	}
}

func TestExecuteRequestTransportError(t *testing.T) {
	t.Parallel()

	transportErr := errors.New("transport failure")
	for _, tc := range []struct {
		name          string
		retryPolicy   RetryPolicy
		expectedRetry bool
		expectedErr   error
	}{
		{
			name:        "default policy returns transport error",
			expectedErr: transportErr,
		},
		{
			name: "custom policy declines transport error",
			retryPolicy: func(*http.Response, error) (bool, error) {
				return false, nil
			},
			expectedErr: transportErr,
		},
		{
			name: "custom policy retries transport error",
			retryPolicy: func(*http.Response, error) (bool, error) {
				return true, nil
			},
			expectedRetry: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reporter := &trackingReporter{}
			httpClient := &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return nil, transportErr
				}),
			}
			options := []RequesterOption{
				WithBackoff(func(int) time.Duration { return 0 }),
				WithReporter(reporter),
			}
			if tc.retryPolicy != nil {
				options = append(options, WithRetryPolicy(tc.retryPolicy))
			}
			r, err := New("test", httpClient, options...)
			require.NoError(t, err, "New must not error")
			t.Cleanup(func() {
				assert.NoError(t, r.Shutdown(), "Shutdown should not error")
			})

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", http.NoBody)
			require.NoError(t, err, "http.NewRequestWithContext must not error")
			retry, err := r.executeRequest(t.Context(), &Item{Method: http.MethodGet, Path: "https://example.com"}, req, 1, false)
			require.Equal(t, tc.expectedRetry, retry, "executeRequest must return the correct retry decision")
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr, "executeRequest must return the transport error")
			} else {
				require.NoError(t, err, "executeRequest must not error")
			}
			assert.Zero(t, reporter.calls, "reporter.calls should remain zero")
		})
	}
}

func TestExecuteRequestRedirectError(t *testing.T) {
	t.Parallel()

	redirectErr := errors.New("redirect failure")
	body := &trackedReadCloser{Reader: strings.NewReader("redirect response")}
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Status:     "302 Found",
				StatusCode: http.StatusFound,
				Header: http.Header{
					"Location": {"https://example.net"},
				},
				Body:    body,
				Request: req,
			}, nil
		}),
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return redirectErr
		},
	}
	r, err := New("test", httpClient, WithRetryPolicy(func(*http.Response, error) (bool, error) {
		return false, nil
	}))
	require.NoError(t, err, "New must not error")
	t.Cleanup(func() {
		assert.NoError(t, r.Shutdown(), "Shutdown should not error")
	})

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", http.NoBody)
	require.NoError(t, err, "http.NewRequestWithContext must not error")
	retry, err := r.executeRequest(t.Context(), &Item{Method: http.MethodGet, Path: "https://example.com"}, req, 1, false)
	require.ErrorIs(t, err, redirectErr, "executeRequest must return the redirect error")
	require.False(t, retry, "executeRequest must not retry")
	assert.Equal(t, 1, body.closeCalls, "executeRequest should close body exactly once")
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

func TestDoRequest_NoContent(t *testing.T) {
	t.Parallel()
	newRequester := func(t *testing.T) *Requester {
		t.Helper()
		r, err := New("test", common.NewHTTPClientWithTimeout(0), WithLimiter(globalshell))
		require.NoError(t, err, "New requester must not error")
		t.Cleanup(func() { assert.NoError(t, r.Shutdown(), "Shutdown should not error") })
		return r
	}

	scenarioTests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "no content zero value result remains unchanged",
			run: func(t *testing.T) {
				t.Helper()
				r := newRequester(t)
				var resp struct {
					Response bool `json:"response"`
				}
				err := r.SendPayload(t.Context(), UnAuth, func() (*Item, error) {
					return &Item{
						Method: http.MethodPost,
						Path:   testURL + "/nocontent",
						Result: &resp,
					}, nil
				}, UnauthenticatedRequest)
				require.NoError(t, err, "SendPayload must not error on 204 No Content")
				assert.False(t, resp.Response, "result should remain zero value on 204 No Content")
			},
		},
		{
			name: "no content pre populated result remains unchanged",
			run: func(t *testing.T) {
				t.Helper()
				r := newRequester(t)
				resp := struct {
					Response bool `json:"response"`
				}{
					Response: true,
				}
				err := r.SendPayload(t.Context(), UnAuth, func() (*Item, error) {
					return &Item{
						Method: http.MethodPost,
						Path:   testURL + "/nocontent",
						Result: &resp,
					}, nil
				}, UnauthenticatedRequest)
				require.NoError(t, err, "SendPayload must not error on 204 No Content")
				assert.True(t, resp.Response, "result should remain unchanged when unmarshalling is skipped on 204 No Content")
			},
		},
		{
			name: "no content nil result must not error",
			run: func(t *testing.T) {
				t.Helper()
				r := newRequester(t)
				err := r.SendPayload(t.Context(), UnAuth, func() (*Item, error) {
					return &Item{
						Method: http.MethodPost,
						Path:   testURL + "/nocontent",
					}, nil
				}, UnauthenticatedRequest)
				require.NoError(t, err, "SendPayload must not error on 204 No Content with nil Result")
			},
		},
		{
			name: "no content header response must be copied",
			run: func(t *testing.T) {
				t.Helper()
				server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-No-Content", "true")
					w.WriteHeader(http.StatusNoContent)
				}))

				r := newRequester(t)
				require.NoError(t, r.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
				headers := http.Header{}
				var resp struct {
					Response bool `json:"response"`
				}
				err := r.SendPayload(t.Context(), UnAuth, func() (*Item, error) {
					return &Item{
						Method:         http.MethodPost,
						Path:           server.URL,
						Result:         &resp,
						HeaderResponse: &headers,
					}, nil
				}, UnauthenticatedRequest)
				require.NoError(t, err, "SendPayload must not error on 204 No Content with headers")
				assert.Equal(t, "true", headers.Get("X-No-Content"), "HeaderResponse should contain custom headers for 204 No Content")
				assert.Equal(t, "application/json", headers.Get("Content-Type"), "HeaderResponse should contain Content-Type for 204 No Content")
				assert.False(t, resp.Response, "result should remain zero value when 204 No Content has headers")
			},
		},
	}
	for _, tc := range scenarioTests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestDoRequest_Retries(t *testing.T) {
	t.Parallel()

	r, err := New("test", new(http.Client), WithBackoff(func(int) time.Duration { return 0 }))
	require.NoError(t, err, "New requester must not error")

	var ec common.ErrorCollector
	for range 4 {
		ec.Go(func() error {
			var resp struct {
				Response bool `json:"response"`
			}
			itemFn := func() (*Item, error) {
				return &Item{
					Method: http.MethodGet,
					Path:   testURL + "/rate-retry",
					Result: &resp,
				}, nil
			}

			if err := r.SendPayload(t.Context(), Auth, itemFn, AuthenticatedRequest); err != nil {
				return fmt.Errorf("SendPayload error: %w", err)
			}
			if !resp.Response {
				return fmt.Errorf("unexpected response: %+v", resp)
			}
			return nil
		})
	}

	require.NoError(t, ec.Collect(), "Collect must return no errors")
}

func TestDoRequest_RetryNonRecoverable(t *testing.T) {
	t.Parallel()

	backoff := func(int) time.Duration {
		return 0
	}
	r, err := New("test", new(http.Client), WithBackoff(backoff))
	if err != nil {
		t.Fatal(err)
	}
	err = r.SendPayload(t.Context(), Unset, func() (*Item, error) {
		return &Item{
			Method: http.MethodGet,
			Path:   testURL + "/always-retry",
		}, nil
	}, UnauthenticatedRequest)
	require.ErrorIs(t, err, errFailedToRetryRequest)
}

func TestDoRequest_NotRetryable(t *testing.T) {
	t.Parallel()

	notRetryErr := errors.New("not retryable")
	retry := func(*http.Response, error) (bool, error) {
		return false, notRetryErr
	}
	backoff := func(n int) time.Duration {
		return time.Duration(n) * time.Millisecond
	}
	r, err := New("test", new(http.Client), WithRetryPolicy(retry), WithBackoff(backoff))
	if err != nil {
		t.Fatal(err)
	}
	err = r.SendPayload(t.Context(), Unset, func() (*Item, error) {
		return &Item{
			Method: http.MethodGet,
			Path:   testURL + "/always-retry",
		}, nil
	}, UnauthenticatedRequest)
	require.ErrorIs(t, err, notRetryErr)
}

func TestEvaluateRetry(t *testing.T) {
	t.Parallel()

	r := Requester{}
	retry, err := r.evaluateRetry(WithRetryNotAllowed(t.Context()), nil, errInvalidPath, 1, false)
	require.ErrorIs(t, err, errInvalidPath, "must return incoming error when retry not allowed")
	require.False(t, retry, "must not retry when retry not allowed")

	retryPolicyErr := errors.New("retry policy failure")
	responseBody := &trackedReadCloser{Reader: strings.NewReader("response")}
	r.retryPolicy = func(*http.Response, error) (bool, error) {
		return false, retryPolicyErr
	}
	retry, err = r.evaluateRetry(t.Context(), &http.Response{Body: responseBody}, nil, 1, false)
	require.ErrorIs(t, err, retryPolicyErr, "evaluateRetry must return the retry policy error")
	require.False(t, retry, "evaluateRetry must not retry after a retry policy error")
	assert.Equal(t, 1, responseBody.closeCalls, "evaluateRetry should close responseBody exactly once after a retry policy error")

	retry, err = r.evaluateRetry(t.Context(), nil, nil, 1, false)
	require.ErrorIs(t, err, retryPolicyErr, "evaluateRetry must return the retry policy error without a response")
	require.False(t, retry, "evaluateRetry must not retry after a retry policy error without a response")

	transportErr := errors.New("transport failure")
	responseBody = &trackedReadCloser{Reader: strings.NewReader("response")}
	retry, err = r.evaluateRetry(t.Context(), &http.Response{Body: responseBody}, transportErr, 1, false)
	require.ErrorIs(t, err, retryPolicyErr, "evaluateRetry must return the retry policy error after a transport error")
	require.False(t, retry, "evaluateRetry must not retry after a retry policy and transport error")
	assert.Zero(t, responseBody.closeCalls, "evaluateRetry should leave responseBody open after a transport error")
	require.NoError(t, responseBody.Close(), "responseBody.Close must not error")

	drainErr := errors.New("response drain failure")
	drainCloseErr := errors.New("response close failure")
	responseBody = &trackedReadCloser{Reader: iotest.ErrReader(drainErr), closeErr: drainCloseErr}
	retry, err = r.evaluateRetry(t.Context(), &http.Response{Body: responseBody}, nil, 1, false)
	require.ErrorIs(t, err, retryPolicyErr, "evaluateRetry must return the retry policy error after drain and close failures")
	require.False(t, retry, "evaluateRetry must not retry after drain and close failures")
	assert.Equal(t, 1, responseBody.closeCalls, "evaluateRetry should close responseBody exactly once after drain and close failures")

	r.retryPolicy = func(*http.Response, error) (bool, error) {
		return false, nil
	}
	retry, err = r.evaluateRetry(t.Context(), nil, transportErr, 1, false)
	require.ErrorIs(t, err, transportErr, "evaluateRetry must return the transport error when retrying is declined")
	require.False(t, retry, "evaluateRetry must not retry when the retry policy declines")

	credentialErr := &url.Error{Op: http.MethodGet, URL: "https://example.com?signature=returned-secret", Err: transportErr}
	retry, err = r.evaluateRetry(t.Context(), nil, credentialErr, 1, false)
	require.ErrorIs(t, err, transportErr, "evaluateRetry must preserve the transport cause after redacting the URL")
	require.False(t, retry, "evaluateRetry must not retry a declined URL error")
	assert.NotContains(t, err.Error(), "returned-secret", "evaluateRetry should not return URL credentials to its caller")
	assert.Contains(t, err.Error(), "signature=[REDACTED]", "evaluateRetry should retain redacted URL structure")

	r.retryPolicy = DefaultRetryPolicy
	retry, err = r.evaluateRetry(t.Context(), nil, errInvalidPath, 1, false)
	require.ErrorIs(t, err, errInvalidPath, "must return incoming error when using default retry policy")
	require.False(t, retry, "must not retry when using default retry policy and the error is non-timeout")

	retry, err = r.evaluateRetry(t.Context(), &http.Response{StatusCode: http.StatusOK}, nil, 1, false)
	require.NoError(t, err, "must not error when response is OK")
	require.False(t, retry, "must not retry on 200 status")

	errTimeout := &net.DNSError{IsTimeout: true}
	retry, err = r.evaluateRetry(t.Context(), nil, errTimeout, 1, false)
	require.ErrorIs(t, err, errFailedToRetryRequest, "must return error when attempt is higher than max retries")
	require.ErrorIs(t, err, errExceedsMaxRetries, "must return error when attempt is higher than max retries")
	require.ErrorIs(t, err, errTimeout, "must wrap original error")
	require.False(t, retry, "must not retry when max attempts exceeded")

	maxRetryBody := &trackedReadCloser{Reader: strings.NewReader("")}
	retry, err = r.evaluateRetry(t.Context(), &http.Response{StatusCode: http.StatusTooManyRequests, Status: "429", Body: maxRetryBody}, nil, 1, false)
	require.ErrorContains(t, err, "failed to retry request exceeds maximum retry attempts: status \"429\"", "must return error and status code when attempt is higher than max retries")
	require.False(t, retry, "must not retry when max attempts exceeded")
	assert.Equal(t, 1, maxRetryBody.closeCalls, "evaluateRetry should close maxRetryBody exactly once after maximum retries")

	r.maxRetries = 1
	r.backoff = func(int) time.Duration { return time.Millisecond * 10 }
	ctx, cancel := context.WithDeadline(t.Context(), time.Now())
	defer cancel()
	retry, err = r.evaluateRetry(ctx, nil, errTimeout, 1, false)
	require.ErrorIs(t, err, errFailedToRetryRequest, "must return error when deadline would be exceeded")
	require.ErrorIs(t, err, context.DeadlineExceeded, "must return error when deadline would be exceeded")
	require.ErrorIs(t, err, errTimeout, "must wrap original error")
	require.False(t, retry, "must not retry when deadline would be exceeded")

	deadlineBody := &trackedReadCloser{Reader: strings.NewReader("")}
	retry, err = r.evaluateRetry(ctx, &http.Response{StatusCode: http.StatusTooManyRequests, Status: "429", Body: deadlineBody}, nil, 1, false)
	require.ErrorContains(t, err, "failed to retry request context deadline exceeded: status \"429\"", "must return error and status code when attempt is higher than max retries")
	require.False(t, retry, "must not retry when deadline would be exceeded")
	assert.Equal(t, 1, deadlineBody.closeCalls, "evaluateRetry should close deadlineBody exactly once when the deadline would be exceeded")

	ctx, cancel = context.WithCancel(t.Context())
	cancel()
	cancelledBody := &trackedReadCloser{Reader: strings.NewReader("")}
	retry, err = r.evaluateRetry(ctx, &http.Response{StatusCode: http.StatusTooManyRequests, Status: "429", Body: cancelledBody}, nil, 1, true)
	require.ErrorIs(t, err, errFailedToRetryRequest, "must return error when context is cancelled")
	require.ErrorIs(t, err, context.Canceled, "must return error when context is cancelled")
	require.False(t, retry, "must not retry when context is cancelled")
	assert.Equal(t, 1, cancelledBody.closeCalls, "evaluateRetry should close cancelledBody exactly once when the context is cancelled")

	retryBody := &trackedReadCloser{Reader: strings.NewReader("")}
	retry, err = r.evaluateRetry(t.Context(), &http.Response{StatusCode: http.StatusTooManyRequests, Status: "429", Body: retryBody}, nil, 1, true)
	require.NoError(t, err, "must not error")
	require.True(t, retry, "must retry on 429 response")
	assert.Equal(t, 1, retryBody.closeCalls, "evaluateRetry should close retryBody exactly once before retrying")

	r.backoff = func(int) time.Duration { return 0 }
	retryBody = &trackedReadCloser{Reader: iotest.ErrReader(drainErr), closeErr: drainCloseErr}
	retry, err = r.evaluateRetry(t.Context(), &http.Response{StatusCode: http.StatusTooManyRequests, Status: "429", Body: retryBody}, nil, 1, true)
	require.NoError(t, err, "evaluateRetry must not error after drain and close failures")
	require.True(t, retry, "evaluateRetry must retry after drain and close failures")
	assert.Equal(t, 1, retryBody.closeCalls, "evaluateRetry should close retryBody exactly once after drain and close failures")

	retry, err = r.evaluateRetry(t.Context(), nil, errTimeout, 1, true)
	require.NoError(t, err, "must not error")
	require.True(t, retry, "must retry on timeout error")
}

func TestGetNonce(t *testing.T) {
	t.Parallel()
	r, err := New("test", new(http.Client), WithLimiter(globalshell))
	require.NoError(t, err)
	n1 := r.GetNonce(nonce.Unix)
	assert.NotZero(t, n1)
	n2 := r.GetNonce(nonce.Unix)
	assert.NotZero(t, n2)
	assert.NotEqual(t, n1, n2)

	r2, err := New("test", new(http.Client), WithLimiter(globalshell))
	require.NoError(t, err)
	n3 := r2.GetNonce(nonce.UnixNano)
	assert.NotZero(t, n3)
	n4 := r2.GetNonce(nonce.UnixNano)
	assert.NotZero(t, n4)
	assert.NotEqual(t, n3, n4)

	assert.NotEqual(t, n1, n3)
	assert.NotEqual(t, n2, n4)
}

// 40532461	       30.29 ns/op	       0 B/op	       0 allocs/op (prev)
// 45329203	       26.53 ns/op	       0 B/op	       0 allocs/op
func BenchmarkGetNonce(b *testing.B) {
	r, err := New("test", new(http.Client), WithLimiter(globalshell))
	require.NoError(b, err)
	for b.Loop() {
		r.GetNonce(nonce.UnixNano)
		r.timedLock.UnlockIfLocked()
	}
}

func TestSetProxy(t *testing.T) {
	t.Parallel()
	var r *Requester
	err := r.SetProxy(nil)
	require.ErrorIs(t, err, ErrRequestSystemIsNil)

	r, err = New("test", &http.Client{Transport: new(http.Transport)}, WithLimiter(globalshell))
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse("http://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	err = r.SetProxy(u)
	if err != nil {
		t.Fatal(err)
	}
	u, err = url.Parse("")
	if err != nil {
		t.Fatal(err)
	}
	err = r.SetProxy(u)
	if err == nil {
		t.Fatal("error cannot be nil")
	}
}

func TestBasicLimiter(t *testing.T) {
	r, err := New("test", new(http.Client), WithLimiter(NewBasicRateLimit(time.Second, 1, 1)))
	if err != nil {
		t.Fatal(err)
	}
	i := Item{Path: "http://www.google.com", Method: http.MethodGet}
	ctx := t.Context()

	tn := time.Now()
	err = r.SendPayload(ctx, Unset, func() (*Item, error) { return &i, nil }, UnauthenticatedRequest)
	if err != nil {
		t.Fatal(err)
	}
	err = r.SendPayload(ctx, Unset, func() (*Item, error) { return &i, nil }, UnauthenticatedRequest)
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(tn) < time.Second {
		t.Error("rate limit issues")
	}

	ctx, cancel := context.WithDeadline(ctx, tn.Add(time.Nanosecond))
	defer cancel()
	err = r.SendPayload(ctx, Unset, func() (*Item, error) { return &i, nil }, UnauthenticatedRequest)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestSendPayloadRateLimitBarrierWithoutLimiter(t *testing.T) {
	t.Parallel()
	writes := make(chan struct{}, 2)
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		writes <- struct{}{}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	})}
	r, err := New("barrier", client)
	require.NoError(t, err, "New requester must not error")
	contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
	require.NoError(t, err)
	errs := make(chan error, 2)
	send := func(ctx context.Context) {
		errs <- r.SendPayload(ctx, Unset, func() (*Item, error) {
			return &Item{Method: http.MethodGet, Path: "https://example.com", Result: new(any)}, nil
		}, UnauthenticatedRequest)
	}

	go send(contexts[0])
	require.Never(t, func() bool { return len(writes) != 0 }, 10*time.Millisecond, time.Millisecond,
		"limiter-less request must wait for its peer")
	go send(contexts[1])
	require.NoError(t, <-errs)
	require.NoError(t, <-errs)
	require.Len(t, writes, 2)
}

func TestSendPayloadRateLimitBarrierAllowsRetryAfterAcceptance(t *testing.T) {
	t.Parallel()
	newRequester := func(retryFirst bool) (*Requester, *int) {
		calls := 0
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			status := http.StatusOK
			if retryFirst && calls == 1 {
				status = http.StatusTooManyRequests
			}
			return &http.Response{
				StatusCode: status,
				Header:     http.Header{"Content-Type": {"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			}, nil
		})}
		r, err := New("barrier-retry", client,
			WithLimiter(NewBasicRateLimit(time.Millisecond, 100, 1)),
			WithBackoff(func(int) time.Duration { return 0 }))
		require.NoError(t, err, "New requester must not error")
		return r, &calls
	}

	left, leftCalls := newRequester(false)
	right, rightCalls := newRequester(true)
	contexts, err := NewRateLimitBarrierContexts(t.Context(), 2)
	require.NoError(t, err)
	errs := make(chan error, 2)
	send := func(ctx context.Context, r *Requester) {
		errs <- r.SendPayload(ctx, Unset, func() (*Item, error) {
			return &Item{Method: http.MethodGet, Path: "https://example.com", Result: new(any)}, nil
		}, UnauthenticatedRequest)
	}

	go send(contexts[0], left)
	go send(contexts[1], right)
	require.NoError(t, <-errs)
	require.NoError(t, <-errs)
	assert.Equal(t, 1, *leftCalls, "accepted request should execute once")
	assert.Equal(t, 2, *rightCalls, "accepted request should pass its retry rate-limit gate")
}

func TestEnableDisableRateLimit(t *testing.T) {
	r, err := New("TestRequest", new(http.Client), WithLimiter(NewBasicRateLimit(50*time.Millisecond, 1, 1)))
	require.NoError(t, err, "New requester must not error")

	sendIt := func() error {
		return r.SendPayload(t.Context(), Auth, func() (*Item, error) {
			return &Item{
				Method: http.MethodGet,
				Path:   testURL,
				Result: new(any),
			}, nil
		}, AuthenticatedRequest)
	}

	// allow initial request
	require.NoError(t, sendIt(), "sendIt must not error")

	// error on redundant enable
	assert.ErrorIs(t, r.EnableRateLimiter(), ErrRateLimiterAlreadyEnabled)

	// error on redundant disable
	require.NoError(t, r.DisableRateLimiter(), "DisableRateLimiter must not error")
	assert.ErrorIs(t, r.DisableRateLimiter(), ErrRateLimiterAlreadyDisabled)

	// allow requests when disabled
	require.NoError(t, sendIt(), "sendIt must not error")

	// allow when re-enabled
	require.NoError(t, r.EnableRateLimiter(), "EnableRateLimiter must succeed")
	require.NoError(t, sendIt(), "sendIt must not error")

	// block excess requests
	require.NoError(t, sendIt(), "sendIt must not error") // consume the one token
	start := time.Now()
	err = sendIt() // this should block until a token is refilled
	require.NoError(t, err, "sendIt must not error")
	elapsed := time.Since(start)
	assert.GreaterOrEqualf(t, elapsed.Milliseconds(), int64(20), "Expected sendIt to block for at least 20ms, but it returned after %dms", elapsed.Milliseconds())
}

func TestSetHTTPClient(t *testing.T) {
	var r *Requester
	err := r.SetHTTPClient(nil)
	require.ErrorIs(t, err, ErrRequestSystemIsNil)

	client := new(http.Client)
	r = new(Requester)
	err = r.SetHTTPClient(client)
	require.NoError(t, err)

	err = r.SetHTTPClient(client)
	require.ErrorIs(t, err, errCannotReuseHTTPClient)
}

func TestSetHTTPClientTimeout(t *testing.T) {
	var r *Requester
	err := r.SetHTTPClientTimeout(0)
	require.ErrorIs(t, err, ErrRequestSystemIsNil)

	r = new(Requester)
	err = r.SetHTTPClient(common.NewHTTPClientWithTimeout(2))
	if err != nil {
		t.Fatal(err)
	}
	err = r.SetHTTPClientTimeout(time.Second)
	require.NoError(t, err)
}

func TestSetHTTPClientUserAgent(t *testing.T) {
	var r *Requester
	err := r.SetHTTPClientUserAgent("")
	require.ErrorIs(t, err, ErrRequestSystemIsNil)

	r = new(Requester)
	err = r.SetHTTPClientUserAgent("")
	require.NoError(t, err)
}

func TestGetHTTPClientUserAgent(t *testing.T) {
	var r *Requester
	_, err := r.GetHTTPClientUserAgent()
	require.ErrorIs(t, err, ErrRequestSystemIsNil)

	r = new(Requester)
	err = r.SetHTTPClientUserAgent("sillyness")
	require.NoError(t, err)

	ua, err := r.GetHTTPClientUserAgent()
	require.NoError(t, err)

	if ua != "sillyness" {
		t.Fatal("unexpected value")
	}
}

func TestGetRateLimiterDefinitions(t *testing.T) {
	t.Parallel()
	require.Equal(t, RateLimitDefinitions(nil), (*Requester)(nil).GetRateLimiterDefinitions())
	r, err := New("test", new(http.Client), WithLimiter(globalshell))
	require.NoError(t, err)
	require.NotEmpty(t, r.GetRateLimiterDefinitions())
	assert.Equal(t, globalshell, r.GetRateLimiterDefinitions())
}
