package deribit

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	apiURL              = "https://www.deribit.com"
	testnetAPIURL       = "https://test.deribit.com"
	websocketURL        = "wss://www.deribit.com/ws/api/v2"
	testnetWebsocketURL = "wss://test.deribit.com/ws/api/v2"

	apiPath             = "/api/v2/"
	jsonRPCVersion      = "2.0"
	privateMethodPrefix = "private/"
	resultOK            = "ok"

	// errorCodeRetry asks the client to send the request again, as Deribit could not process it yet
	errorCodeRetry = 10040
	// errorCodeOrderNotFound reports that no order has the ID or label a request names
	errorCodeOrderNotFound = 10004
	// maximumRequestAttempts bounds how often a request Deribit asks to retry, or a book subscription Deribit refuses to
	// renew, is sent
	maximumRequestAttempts = 5
)

var (
	errInstrumentNameRequired    = errors.New("instrument name is required")
	errOrderIDRequired           = errors.New("order ID is required")
	errLabelRequired             = errors.New("label is required")
	errInvalidKind               = errors.New("invalid instrument kind")
	errInvalidDirection          = errors.New("invalid direction")
	errInvalidAmount             = errors.New("amount must be greater than zero")
	errInvalidCount              = errors.New("invalid count")
	errUnexpectedResult          = errors.New("unexpected result")
	errWebsocketNotAuthenticated = errors.New("websocket connection is not authenticated")
)

// SendHTTPRequest sends an unauthenticated JSON-RPC request for a public method and decodes its result
func (e *Exchange) SendHTTPRequest(ctx context.Context, epl request.EndpointLimit, method string, params, result any) error {
	return e.sendHTTPRequest(ctx, epl, method, params, result, nil)
}

// SendAuthenticatedHTTPRequest sends a JSON-RPC request signed with the configured credentials and decodes its result
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, epl request.EndpointLimit, method string, params, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, err)
	}
	return e.sendHTTPRequest(ctx, epl, method, params, result, creds)
}

// sendHTTPRequest posts a JSON-RPC request to the method's path, signing it when credentials are given, and returns the
// error object Deribit sends in place of a result
func (e *Exchange) sendHTTPRequest(ctx context.Context, epl request.EndpointLimit, method string, params, result any, creds *accounts.Credentials) error {
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestFutures)
	if err != nil {
		return err
	}
	if creds != nil {
		if params, err = authorisedParams(ctx, params); err != nil {
			return err
		}
	}
	body, err := json.Marshal(&JSONRPCRequest{JSONRPC: jsonRPCVersion, Method: method, Params: params})
	if err != nil {
		return err
	}
	path := apiPath + method
	requestType := request.AuthType(request.UnauthenticatedRequest)
	if creds != nil {
		requestType = request.AuthenticatedRequest
	}
	if !resendable(method) {
		// Deribit may have carried out a request whose response timed out, so the requester must not send it again;
		// retryRequest still does when Deribit answers that it did not process the request
		ctx = request.WithRetryNotAllowed(ctx)
	}
	return retryRequest(ctx, func() error {
		resp := &JSONRPCResponse{Result: result}
		var challenged privateResult
		if creds != nil {
			challenged.result = result
			resp.Result = &challenged
		}
		err := e.SendPayload(ctx, epl, func() (*request.Item, error) {
			return e.httpRequestItem(endpoint, path, body, resp, creds)
		}, requestType)
		// Deribit answers a failed request with an error status, so its error object explains the failure better; an
		// authenticated request's failure is still reported as one, as the requester reports it
		if resp.Error != nil {
			if creds != nil {
				return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, resp.Error)
			}
			return resp.Error
		}
		if err != nil {
			return err
		}
		if challenged.challenge != nil {
			return challenged.challenge
		}
		return nil
	})
}

// httpRequestItem returns the item posting a JSON-RPC body to an endpoint's path, signed when credentials are given; a
// signed item is nonce enabled, as taking its nonce already holds the requester's timed lock, which the requester would
// otherwise wait out before sending it
func (e *Exchange) httpRequestItem(endpoint, path string, body []byte, result any, creds *accounts.Credentials) (*request.Item, error) {
	headers := map[string]string{"Content-Type": "application/json"}
	if creds != nil {
		authorisation, err := signHTTPRequest(creds, path, body, time.Now(), e.Requester.GetNonce(nonce.UnixNano).String())
		if err != nil {
			return nil, err
		}
		headers["Authorization"] = authorisation
	}
	return &request.Item{
		Method:                 http.MethodPost,
		Path:                   endpoint + path,
		Headers:                headers,
		Body:                   bytes.NewReader(body),
		Result:                 result,
		NonceEnabled:           creds != nil,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}, nil
}

// sendHTTPQueryRequest sends a public, non-matching engine method as an HTTP GET request with its params in the query
// string, and decodes its result
func (e *Exchange) sendHTTPQueryRequest(ctx context.Context, method string, params url.Values, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestFutures)
	if err != nil {
		return err
	}
	path := common.EncodeURLValues(endpoint+apiPath+method, params)
	return retryRequest(ctx, func() error {
		resp := &JSONRPCResponse{Result: result}
		err := e.SendPayload(ctx, nonMatchingEPL, func() (*request.Item, error) {
			return &request.Item{
				Method:                 http.MethodGet,
				Path:                   path,
				Result:                 resp,
				Verbose:                e.Verbose,
				HTTPDebugging:          e.HTTPDebugging,
				HTTPRecording:          e.HTTPRecording,
				HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
			}, nil
		}, request.UnauthenticatedRequest)
		// Deribit answers a failed request with an error status, so its error object explains the failure better
		if resp.Error != nil {
			return resp.Error
		}
		return err
	})
}

// signHTTPRequest returns the Authorization header of a request, signing its timestamp, nonce, method, path and body
// with the credentials' secret
func signHTTPRequest(creds *accounts.Credentials, path string, body []byte, now time.Time, requestNonce string) (string, error) {
	timestamp := strconv.FormatInt(now.UnixMilli(), 10)
	payload := timestamp + "\n" + requestNonce + "\n" + http.MethodPost + "\n" + path + "\n" + string(body) + "\n"
	signature, err := crypto.GetHMAC(crypto.HashSHA256, []byte(payload), []byte(creds.Secret))
	if err != nil {
		return "", err
	}
	return "deri-hmac-sha256 id=" + creds.Key + ",ts=" + timestamp + ",sig=" + hex.EncodeToString(signature) + ",nonce=" + requestNonce, nil
}

// sendRequest sends a request over the websocket or HTTP, signing an HTTP request for a private method
func (e *Exchange) sendRequest(ctx context.Context, overWebsocket bool, epl request.EndpointLimit, method string, params, result any) error {
	switch {
	case overWebsocket:
		return e.SendWebsocketRequest(ctx, epl, method, params, result)
	case isPrivateMethod(method):
		return e.SendAuthenticatedHTTPRequest(ctx, epl, method, params, result)
	default:
		return e.SendHTTPRequest(ctx, epl, method, params, result)
	}
}

// sendOKRequest sends a non-matching engine request whose result is "ok" when it succeeds
func (e *Exchange) sendOKRequest(ctx context.Context, overWebsocket bool, method string, params any) error {
	var resp string
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, method, params, &resp); err != nil {
		return err
	}
	if resp != resultOK {
		return fmt.Errorf("%w: %s returned %q", errUnexpectedResult, method, resp)
	}
	return nil
}

// isPrivateMethod reports whether a method needs authentication
func isPrivateMethod(method string) bool {
	return strings.HasPrefix(method, privateMethodPrefix)
}

// resendable reports whether a request may be sent again after its response times out: a public method, or a private
// one that only reads, as Deribit may already have carried out an order, cancel, withdrawal, transfer or other change
func resendable(method string) bool {
	action, private := strings.CutPrefix(method, privateMethodPrefix)
	return !private || strings.HasPrefix(action, "get_") || strings.HasPrefix(action, "list_") ||
		strings.HasPrefix(action, "simulate_")
}

// retryRequest sends a request until Deribit stops asking for it to be retried, backing off between attempts
func retryRequest(ctx context.Context, send func() error) error {
	backoff := request.DefaultBackoff()
	for attempt := 1; ; attempt++ {
		err := send()
		var apiErr *APIError
		if attempt == maximumRequestAttempts || !errors.As(err, &apiErr) || apiErr.Code != errorCodeRetry {
			return err
		}
		timer := time.NewTimer(backoff(attempt))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
