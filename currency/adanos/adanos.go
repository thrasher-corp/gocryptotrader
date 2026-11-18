// Package adanos provides an optional, on-demand client for Adanos Reddit crypto
// sentiment. It does not register an engine provider or supply market prices.
package adanos

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Validation errors can be matched with errors.Is.
var (
	ErrInvalidAPIKey   = errors.New("adanos API key is missing or invalid")
	ErrInvalidRequest  = errors.New("adanos context and token request are required")
	ErrInvalidSymbol   = errors.New("adanos symbol must contain 1 to 20 ASCII letters or digits with an optional leading dollar sign")
	ErrInvalidPeriod   = errors.New("adanos bounds must be valid inclusive UTC dates with from not after to")
	ErrInvalidResponse = errors.New("adanos response does not contain a token report")

	symbolPattern = regexp.MustCompile(`^\$?[A-Za-z0-9]{1,20}$`)
)

// New constructs a client without network I/O. Requests have a 30-second timeout
// and never follow redirects, so the API key cannot reach a redirect destination.
func New(apiKey string) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" || strings.ContainsAny(apiKey, "\r\n") {
		return nil, ErrInvalidAPIKey
	}
	apiKey = strings.TrimSpace(apiKey)
	httpClient := common.NewHTTPClientWithTimeout(30 * time.Second)
	transport := httpClient.Transport.(*http.Transport) //nolint:forcetypeassert // The native helper always creates an *http.Transport.
	// Prevent HTTP1 reused-connection replay and HTTP2 refused-stream replay.
	transport.DisableKeepAlives = true
	transport.Protocols = new(http.Protocols)
	transport.Protocols.SetHTTP1(true)
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	requester, err := request.New("Adanos", httpClient)
	if err != nil {
		return nil, fmt.Errorf("construct Adanos requester: %w", err)
	}
	return &Client{apiKey: apiKey, baseURL: "https://api.adanos.com", httpClient: httpClient, requester: requester}, nil
}

// Shutdown releases the client after all requests have finished.
// Do not use the client during or after Shutdown. Call it once when retiring it.
func (c *Client) Shutdown() error {
	c.httpClient.CloseIdleConnections()
	return c.requester.Shutdown()
}

// GetToken returns a report for one symbol using inclusive UTC date bounds.
// Each call makes at most one request, including on HTTP 429.
func (c *Client) GetToken(ctx context.Context, params *TokenRequest) (*TokenResponse, error) {
	if ctx == nil || params == nil {
		return nil, ErrInvalidRequest
	}
	if !symbolPattern.MatchString(params.Symbol) {
		return nil, ErrInvalidSymbol
	}
	values := url.Values{}
	for name, bound := range map[string]time.Time{"from": params.From, "to": params.To} {
		if bound.IsZero() {
			continue
		}
		bound = bound.UTC()
		if bound.Year() < 1 || bound.Year() > 9999 {
			return nil, ErrInvalidPeriod
		}
		values.Set(name, bound.Format(time.DateOnly))
	}
	if values.Has("from") && values.Has("to") && values.Get("from") > values.Get("to") {
		return nil, ErrInvalidPeriod
	}
	var response *TokenResponse
	err := c.requester.SendPayload(request.WithRetryNotAllowed(ctx), request.Unset, func() (*request.Item, error) {
		return &request.Item{
			Method: http.MethodGet,
			Path:   common.EncodeURLValues(c.baseURL+"/reddit/crypto/v1/token/"+params.Symbol, values),
			Headers: map[string]string{
				"X-API-Key": c.apiKey,
				"Accept":    "application/json",
			},
			Result: &response,
		}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return nil, &tokenError{err: err, apiKey: c.apiKey}
	}
	if response == nil || response.Symbol == "" {
		return nil, ErrInvalidResponse
	}
	return response, nil
}

func (e *tokenError) Error() string {
	// Provider error detail can echo credentials outside recognised secret fields.
	return "adanos get token: " + strings.ReplaceAll(e.err.Error(), e.apiKey, "[REDACTED]")
}

func (e *tokenError) Unwrap() error {
	return e.err
}
