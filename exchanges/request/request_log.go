package request

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func headerValuesForLog(name string, values []string) []string {
	if isSensitiveLogKey(name) {
		return []string{"[REDACTED]"}
	}
	return values
}

// isSensitiveLogKey matches name suffixes rather than substrings, so OK-ACCESS-KEY, X-BAPI-SIGN and listenKey are
// caught while KC-API-KEY-VERSION, SignatureMethod and signTimestamp stay readable. BTSE sends its API key as btse-api,
// and Bitstamp's v2 authentication sends it as X-Auth.
func isSensitiveLogKey(name string) bool {
	// None of the suffixes span a separator, so only normalise the final segment.
	separator := strings.LastIndexAny(name, "-_")
	name = strings.ToLower(name[separator+1:])
	if separator >= 0 && (name == "api" || name == "auth") {
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
	var redacted strings.Builder
	var offset, copied int
	for field := range strings.SplitSeq(encoded, "&") {
		start := offset
		offset += len(field) + 1
		name, _, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		unescaped, err := url.QueryUnescape(name)
		if err != nil || isSensitiveLogKey(unescaped) {
			// Copy untouched spans only when a value needs redacting.
			if copied == 0 {
				redacted.Grow(len(encoded))
			}
			redacted.WriteString(encoded[copied : start+len(name)+1])
			redacted.WriteString("[REDACTED]")
			copied = start + len(field)
		}
	}
	if copied == 0 {
		return encoded
	}
	redacted.WriteString(encoded[copied:])
	return redacted.String()
}

func pathForLog(path string) string {
	base, query, ok := strings.Cut(path, "?")
	if !ok {
		return path
	}
	redacted := redactEncodedValues(query)
	if redacted == query {
		return path
	}
	return base + "?" + redacted
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
	trimmed := bytes.TrimSpace(payload)
	looksLikeJSON := len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')
	if strings.TrimSpace(contentType) != "" && isFormEncoded(contentType) {
		// Redact as a form first because that is how the receiver interprets it,
		// then as JSON when the body also has that shape.
		redacted := []byte(redactEncodedValues(string(payload)))
		if looksLikeJSON {
			if filtered, err := redactJSONBody(redacted); err == nil {
				return filtered
			}
			// A JSON-shaped body the JSON pass cannot read may still carry credentials.
			return []byte("[REDACTED INVALID JSON BODY]")
		}
		return redacted
	}
	if isJSONEncoded(contentType) || looksLikeJSON || json.Valid(payload) {
		if redacted, err := redactJSONBody(payload); err == nil {
			return redacted
		}
		return []byte("[REDACTED INVALID JSON BODY]")
	}
	if isFormEncoded(contentType) {
		return []byte(redactEncodedValues(string(payload)))
	}
	return []byte("[REDACTED NON-FORM BODY]")
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
