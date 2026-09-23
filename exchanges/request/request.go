package request

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/timedmutex"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// UnsetRequest is an unset request authentication level
	UnsetRequest AuthType = 0
	// UnauthenticatedRequest denotes a request with no credentials
	UnauthenticatedRequest = iota << 1
	// AuthenticatedRequest denotes a request using API credentials
	AuthenticatedRequest
)

// AuthType helps distinguish the purpose of a HTTP request
type AuthType uint8

// Public errors
var (
	ErrRequestSystemIsNil = errors.New("request system is nil")
	ErrAuthRequestFailed  = errors.New("authenticated request failed")
	ErrBadStatus          = errors.New("unsuccessful HTTP status code")
)

var (
	errRequestFunctionIsNil   = errors.New("request function is nil")
	errRequestItemNil         = errors.New("request item is nil")
	errInvalidPath            = errors.New("invalid path")
	errHeaderResponseMapIsNil = errors.New("header response map is nil")
	errFailedToRetryRequest   = errors.New("failed to retry request")
	errContextRequired        = errors.New("context is required")
	errTransportNotSet        = errors.New("transport not set, cannot set timeout")
	errRequestTypeUnpopulated = errors.New("request type bool is not populated")
	errExceedsMaxRetries      = errors.New("exceeds maximum retry attempts")
)

// New returns a new Requester
func New(name string, httpRequester *http.Client, opts ...RequesterOption) (*Requester, error) {
	protectedClient, err := newProtectedClient(httpRequester)
	if err != nil {
		return nil, fmt.Errorf("cannot set up a new requester for %s: %w", name, err)
	}
	r := &Requester{
		_HTTPClient: protectedClient,
		name:        name,
		backoff:     DefaultBackoff(),
		retryPolicy: DefaultRetryPolicy,
		maxRetries:  MaxRetryAttempts,
		timedLock:   timedmutex.NewTimedMutex(DefaultMutexLockTimeout),
		reporter:    globalReporter,
	}

	for _, o := range opts {
		o(r)
	}

	return r, nil
}

// SendPayload handles sending HTTP/HTTPS requests
func (r *Requester) SendPayload(ctx context.Context, ep EndpointLimit, newRequest Generate, requestType AuthType) error {
	if r == nil {
		return ErrRequestSystemIsNil
	}

	if ctx == nil {
		return errContextRequired
	}
	if requestType == UnsetRequest {
		return errRequestTypeUnpopulated
	}

	defer r.timedLock.UnlockIfLocked()

	if newRequest == nil {
		return errRequestFunctionIsNil
	}

	err := r.doRequest(ctx, ep, newRequest)
	if err != nil && requestType == AuthenticatedRequest {
		err = common.AppendError(err, ErrAuthRequestFailed)
	}
	return err
}

// validateRequest validates the requester item fields
func (i *Item) validateRequest(ctx context.Context, r *Requester) (*http.Request, error) {
	if i == nil {
		return nil, errRequestItemNil
	}

	if i.Path == "" {
		return nil, errInvalidPath
	}

	if i.HeaderResponse != nil && *i.HeaderResponse == nil {
		return nil, errHeaderResponseMapIsNil
	}

	if !i.NonceEnabled {
		r.timedLock.LockForDuration()
	}
	req, err := http.NewRequestWithContext(ctx, i.Method, i.Path, i.Body)
	if err != nil {
		return nil, err
	}

	applyStringHeaders(req.Header, i.Headers)

	if r.userAgent != "" && req.Header.Get(userAgent) == "" {
		req.Header.Add(userAgent, r.userAgent)
	}
	applyHeaders(req.Header, headersFromContext(ctx))

	if i.HTTPDebugging {
		if dump, err := dumpRequestForLog(req, req.Header.Get("Content-Type")); err != nil {
			log.Errorf(log.RequestSys, "%s DumpRequest invalid request: %v", r.name, err)
		} else {
			log.Debugf(log.RequestSys, "DumpRequest:\n%s", dump)
		}
	}

	return req, nil
}

// doRequest performs a HTTP/HTTPS request with the supplied params
func (r *Requester) doRequest(ctx context.Context, endpoint EndpointLimit, newRequest Generate) error {
	for attempt := 1; ; attempt++ {
		// Check if context has finished before executing new attempt.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if r.limiter != nil {
			// Initiate a rate limit reservation and sleep on requested endpoint
			err := r.InitiateRateLimit(ctx, endpoint)
			if err != nil {
				return fmt.Errorf("failed to rate limit HTTP request: %w", err)
			}
		} else if err := WaitForRateLimitBarrier(ctx); err != nil {
			return fmt.Errorf("failed to coordinate HTTP request: %w", err)
		}

		p, err := newRequest()
		if err != nil {
			return err
		}

		req, err := p.validateRequest(ctx, r)
		if err != nil {
			return err
		}

		verbose := IsVerbose(ctx, p.Verbose)
		retry, err := r.executeRequest(ctx, p, req, attempt, verbose)
		if err != nil {
			return err
		}
		if retry {
			continue
		}
		return nil
	}
}

// executeRequest performs one HTTP request attempt and reports whether the
// caller should retry. Any response body is closed before this method returns.
func (r *Requester) executeRequest(ctx context.Context, p *Item, req *http.Request, attempt int, verbose bool) (bool, error) {
	if verbose {
		log.Debugf(log.RequestSys, "%s attempt %d request path: %s", r.name, attempt, pathForLog(p.Path))
		for k, d := range req.Header {
			log.Debugf(log.RequestSys, "%s request header [%s]: %s", r.name, k, headerValuesForLog(k, d))
		}
		log.Debugf(log.RequestSys, "%s request type: %s", r.name, p.Method)
		if req.GetBody != nil {
			bodyCopy, bodyErr := req.GetBody()
			if bodyErr != nil {
				return false, bodyErr
			}
			payload, bodyErr := io.ReadAll(bodyCopy)
			if closeErr := bodyCopy.Close(); closeErr != nil {
				log.Errorf(log.RequestSys, "%s failed to close request body %s", r.name, closeErr)
			}
			if bodyErr != nil {
				return false, bodyErr
			}
			payload = bodyForLog(payload, req.Header.Get("Content-Type"))
			log.Debugf(log.RequestSys, "%s request body: %s", r.name, payload)
		}
	}

	start := time.Now()

	resp, requestErr := r._HTTPClient.do(req)

	if r.reporter != nil && requestErr == nil {
		r.reporter.Latency(r.name, p.Method, p.Path, time.Since(start))
	}

	if retry, err := r.evaluateRetry(ctx, resp, requestErr, attempt, verbose); err != nil {
		return false, err
	} else if retry {
		return true, nil
	}

	contents, readErr := io.ReadAll(resp.Body)
	if closeErr := resp.Body.Close(); closeErr != nil {
		log.Errorf(log.RequestSys, "%s failed to close response body %s", r.name, closeErr)
	}
	if readErr != nil {
		return false, readErr
	}
	// Even in the case of an erroneous condition below, yield the parsed
	// response to caller. Skip unmarshalling if there is no body content
	// (e.g. HTTP 204 No Content) to avoid a spurious syntax error.
	var unmarshallError error
	if p.Result != nil && resp.StatusCode != http.StatusNoContent {
		unmarshallError = json.Unmarshal(contents, p.Result)
	}

	if p.HTTPRecording {
		// This dumps http responses for future mocking implementations
		if err := mock.HTTPRecord(resp, r.name, contents, p.HTTPMockDataSliceLimit); err != nil {
			return false, fmt.Errorf("mock recording failure %w, request %v: resp: %v", err, req, resp)
		}
	}

	if p.HeaderResponse != nil {
		maps.Copy(*p.HeaderResponse, resp.Header)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusNoContent {
		return false, fmt.Errorf("%s %w: %d raw response: %s", r.name, ErrBadStatus, resp.StatusCode, bodyForLog(contents, resp.Header.Get("Content-Type")))
	}

	var contentsForLog []byte
	if p.HTTPDebugging || verbose {
		contentsForLog = bodyForLog(contents, resp.Header.Get("Content-Type"))
	}
	if p.HTTPDebugging {
		respForLog := *resp
		respForLog.Header = make(http.Header, len(resp.Header))
		for name, values := range resp.Header {
			respForLog.Header[name] = headerValuesForLog(name, values)
		}
		dump, dumpErr := httputil.DumpResponse(&respForLog, false)
		if dumpErr != nil {
			log.Errorf(log.RequestSys, "DumpResponse invalid response: %v:", dumpErr)
		} else {
			log.Debugf(log.RequestSys, "DumpResponse (%v):\n%s", pathForLog(p.Path), dump)
		}
		log.Debugf(log.RequestSys, "DumpResponse Body (%v):\n %s", pathForLog(p.Path), contentsForLog)
	}

	if verbose {
		for k, d := range resp.Header {
			log.Debugf(log.RequestSys, "%s response header [%s]: %s", r.name, k, headerValuesForLog(k, d))
		}
		log.Debugf(log.RequestSys, "HTTP status: %s, Code: %v", resp.Status, resp.StatusCode)
		if !p.HTTPDebugging {
			log.Debugf(log.RequestSys, "%s raw response: %s", r.name, contentsForLog)
		}
	}
	return false, unmarshallError
}

func headerValuesForLog(name string, values []string) []string {
	if isSensitiveLogKey(name) {
		return []string{"[REDACTED]"}
	}
	return values
}

func applyStringHeaders(destination http.Header, headers map[string]string) {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		destination.Set(key, headers[key])
	}
}

func applyHeaders(destination, headers http.Header) {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		destination.Del(key)
		for _, value := range headers[key] {
			destination.Add(key, value)
		}
	}
}

// isSensitiveLogKey matches name suffixes rather than substrings, so OK-ACCESS-KEY, X-BAPI-SIGN and listenKey are
// caught while KC-API-KEY-VERSION, SignatureMethod and signTimestamp stay readable. BTSE sends its API key as btse-api.
func isSensitiveLogKey(name string) bool {
	name = strings.ToLower(name)
	if i := strings.LastIndexAny(name, "-_"); i >= 0 && name[i+1:] == "api" {
		return true
	}
	for _, suffix := range sensitiveLogKeySuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

var sensitiveLogKeySuffixes = []string{"key", "keyid", "sign", "signature", "authent", "authorization", "passphrase", "password", "secret", "token", "cookie", "tfa", "user"}

// redactEncodedValues keeps field order and non-sensitive values so a redacted query or form body still reads like the request that was sent.
func redactEncodedValues(encoded string) string {
	fields := strings.Split(encoded, "&")
	for i, field := range fields {
		name, _, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		unescaped, err := url.QueryUnescape(name)
		if err != nil || isSensitiveLogKey(unescaped) {
			fields[i] = name + "=[REDACTED]"
		}
	}
	return strings.Join(fields, "&")
}

func pathForLog(path string) string {
	base, query, ok := strings.Cut(path, "?")
	if !ok {
		return path
	}
	return base + "?" + redactEncodedValues(query)
}

func isFormEncoded(contentType string) bool {
	mediaType, _, _ := strings.Cut(contentType, ";")
	mediaType = strings.TrimSpace(mediaType)
	return mediaType == "" || strings.EqualFold(mediaType, "application/x-www-form-urlencoded")
}

func isJSONEncoded(contentType string) bool {
	mediaType, _, _ := strings.Cut(contentType, ";")
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

func bodyForLog(payload []byte, contentType string) []byte {
	if len(payload) == 0 {
		return payload
	}
	if strings.TrimSpace(contentType) != "" && isFormEncoded(contentType) {
		return []byte(redactEncodedValues(string(payload)))
	}
	trimmed := bytes.TrimSpace(payload)
	looksLikeJSON := len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')
	if isJSONEncoded(contentType) || looksLikeJSON || json.Valid(payload) {
		var value any
		if err := json.Unmarshal(payload, &value); err == nil {
			if !redactJSONValue(value) {
				return payload
			}
			if redacted, err := json.Marshal(value); err == nil {
				return redacted
			}
		}
		return []byte("[REDACTED INVALID JSON BODY]")
	}
	if isFormEncoded(contentType) {
		return []byte(redactEncodedValues(string(payload)))
	}
	return []byte("[REDACTED NON-FORM BODY]")
}

func redactJSONValue(value any) bool {
	redacted := false
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if isSensitiveLogKey(key) {
				typed[key] = "[REDACTED]"
				redacted = true
				continue
			}
			redacted = redactJSONValue(nested) || redacted
		}
	case []any:
		for _, nested := range typed {
			redacted = redactJSONValue(nested) || redacted
		}
	}
	return redacted
}

type replayReader struct {
	io.Reader
	terminalErr error
}

func (r replayReader) Read(payload []byte) (int, error) {
	n, err := r.Reader.Read(payload)
	if errors.Is(err, io.EOF) {
		return n, r.terminalErr
	}
	return n, err
}

func dumpRequestForLog(req *http.Request, contentType string) ([]byte, error) {
	clone := req.Clone(req.Context())
	// The sensitive-key suffixes must cover every credential-bearing header
	// sent by exchange adapters because the dump includes their headers.
	for name, values := range clone.Header {
		clone.Header[name] = headerValuesForLog(name, values)
	}
	if req.URL != nil {
		requestURL := *req.URL
		requestURL.RawQuery = redactEncodedValues(requestURL.RawQuery)
		clone.URL = &requestURL
	}
	if req.Body != nil && req.Body != http.NoBody {
		body := req.Body
		if req.GetBody != nil {
			var err error
			body, err = req.GetBody()
			if err != nil {
				return nil, err
			}
		}
		payload, err := io.ReadAll(body)
		if closeErr := body.Close(); err == nil {
			err = closeErr
		}
		if req.GetBody == nil {
			// The reader is not replayable, so restore what the debug dump consumed.
			// A close failure is surfaced as the terminal read error; HTTP/2 may handle
			// that differently from the original Close failure.
			var restored io.Reader = bytes.NewReader(payload)
			if err != nil {
				restored = replayReader{Reader: restored, terminalErr: err}
			}
			req.Body = io.NopCloser(restored)
		}
		if err != nil {
			return nil, err
		}
		payload = bodyForLog(payload, contentType)
		clone.Body = io.NopCloser(bytes.NewReader(payload))
		clone.ContentLength = int64(len(payload))
	}
	return httputil.DumpRequestOut(clone, true)
}

// urlErrorForLog redacts the request URL that http.Client.Do embeds in its errors without mutating the original.
func urlErrorForLog(err error) error {
	return urlErrorForLogDepth(err, maxURLErrorLogDepth)
}

// maxURLErrorLogDepth prevents a cyclic error chain from exhausting the stack.
const maxURLErrorLogDepth = 8

var errTruncatedErrorChain = errors.New("error chain truncated for logging")

func urlErrorForLogDepth(err error, depth int) error {
	if depth == 0 {
		return errTruncatedErrorChain
	}
	urlErr, ok := err.(*url.Error)
	if !ok || urlErr == nil {
		return err
	}
	return &url.Error{Op: urlErr.Op, URL: pathForLog(urlErr.URL), Err: urlErrorForLogDepth(urlErr.Err, depth-1)}
}

// evaluateRetry checks whether a request should be retried based on the retry
// policy and context. It propagates incoming request errors when retrying is
// declined and drains and closes response bodies before retrying or returning
// a retry-decision error.
func (r *Requester) evaluateRetry(ctx context.Context, resp *http.Response, incomingErr error, attempt int, verbose bool) (bool, error) {
	if hasRetryNotAllowed(ctx) {
		return false, urlErrorForLog(incomingErr)
	}

	retry, err := r.retryPolicy(resp, incomingErr)
	if err != nil {
		if incomingErr == nil && resp != nil {
			r.drainBody(resp.Body)
		}
		return false, urlErrorForLog(err)
	}

	if !retry {
		return false, urlErrorForLog(incomingErr)
	}

	if incomingErr == nil {
		// If the body isn't fully read, the connection cannot be reused
		r.drainBody(resp.Body)
	}

	if attempt > r.maxRetries {
		if incomingErr != nil {
			return false, fmt.Errorf("%w %w: err: %w", errFailedToRetryRequest, errExceedsMaxRetries, urlErrorForLog(incomingErr))
		}
		return false, fmt.Errorf("%w %w: status %q", errFailedToRetryRequest, errExceedsMaxRetries, resp.Status)
	}

	after := RetryAfter(resp, time.Now())
	backoff := r.backoff(attempt)
	delay := max(backoff, after)

	if dl, ok := ctx.Deadline(); ok && dl.Before(time.Now().Add(delay)) {
		if incomingErr != nil {
			return false, fmt.Errorf("%w %w: err: %w", errFailedToRetryRequest, context.DeadlineExceeded, urlErrorForLog(incomingErr))
		}
		return false, fmt.Errorf("%w %w: status %q", errFailedToRetryRequest, context.DeadlineExceeded, resp.Status)
	}

	if verbose {
		if incomingErr != nil {
			log.Errorf(log.RequestSys, "%s request has failed. Retrying request in %s, attempt %d, cause: %s", r.name, delay, attempt, urlErrorForLog(incomingErr))
		} else {
			log.Errorf(log.RequestSys, "%s request has failed. Retrying request in %s, attempt %d, status: %q", r.name, delay, attempt, resp.Status)
		}
	}

	if delay > 0 {
		// Allow for context cancellation while delaying the retry.
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return false, fmt.Errorf("%w %w", errFailedToRetryRequest, ctx.Err())
		}
	}

	return true, nil
}

func (r *Requester) drainBody(body io.ReadCloser) {
	if _, err := io.Copy(io.Discard, io.LimitReader(body, drainBodyLimit)); err != nil {
		log.Errorf(log.RequestSys, "%s failed to drain request body %s", r.name, err)
	}
	if err := body.Close(); err != nil {
		log.Errorf(log.RequestSys, "%s failed to close request body %s", r.name, err)
	}
}

// GetNonce returns a nonce for requests. This locks and enforces concurrent
// nonce FIFO on the buffered job channel
func (r *Requester) GetNonce(set nonce.Setter) nonce.Value {
	r.timedLock.LockForDuration()
	return r.Nonce.GetAndIncrement(set)
}

// SetProxy sets a proxy address for the client transport
func (r *Requester) SetProxy(p *url.URL) error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	return r._HTTPClient.setProxy(p)
}

// SetHTTPClient sets exchanges HTTP client
func (r *Requester) SetHTTPClient(newClient *http.Client) error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	protectedClient, err := newProtectedClient(newClient)
	if err != nil {
		return err
	}
	r._HTTPClient = protectedClient
	return nil
}

// SetHTTPClientTimeout sets the timeout value for the exchanges HTTP Client and
// also the underlying transports idle connection timeout
func (r *Requester) SetHTTPClientTimeout(timeout time.Duration) error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	return r._HTTPClient.setHTTPClientTimeout(timeout)
}

// SetHTTPClientUserAgent sets the exchanges HTTP user agent
func (r *Requester) SetHTTPClientUserAgent(userAgent string) error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	r.userAgent = userAgent
	return nil
}

// GetHTTPClientUserAgent gets the exchanges HTTP user agent
func (r *Requester) GetHTTPClientUserAgent() (string, error) {
	if r == nil {
		return "", ErrRequestSystemIsNil
	}
	return r.userAgent, nil
}

// Shutdown releases persistent memory for garbage collection.
func (r *Requester) Shutdown() error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	return r._HTTPClient.release()
}
