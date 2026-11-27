package okx

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Okx
type Exchange struct {
	exchange.Base

	instrumentsInfoMapLock sync.Mutex
	instrumentsInfoMap     map[string][]InstrumentResponse
	// instrumentIDCodeMap caches instrument ID codes by instrument ID for
	// websocket order operations which identify instruments by code.
	instrumentIDCodeMap map[string]uint64
	// instrumentTradingStates holds the trading state of each instrument OKX
	// last listed or pushed as pre-open or with a time it switches to
	// continuous trading, which decides whether its tickers may quote their
	// best ask below their best bid
	instrumentTradingStates map[string]instrumentTradingState
	// instrumentGeneration counts the instrument listings requested and the
	// instruments channel pushes cached. instrumentGenerations holds the count
	// of the listing or push which last set each instrument, and
	// instrumentListGenerations of the listing which last set each instrument
	// type's list, so neither a listing requested before a push nor an older
	// listing replaces what a newer one set.
	instrumentGeneration      uint64
	instrumentGenerations     map[string]uint64
	instrumentListGenerations map[string]uint64
	// accountPositionModes caches each account's contract position mode, and
	// accountLevels its account level, by API key: the mode decides the
	// placement of futures and perpetual swap orders, the level whether a cross
	// MARGIN order sends its margin currency. A request can carry another
	// account's credentials in its context, so one account's modes must not
	// answer for another's. Both come from the account configuration, fetched
	// on first use, and are refreshed by SetPositionMode and SetAccountMode,
	// which count in accountPositionModeSwitches every switch that may have
	// changed them.
	accountPositionModeMu       sync.RWMutex
	accountPositionModes        map[string]string
	accountLevels               map[string]uint64
	accountPositionModeSwitches uint64
	// accountModeSwitchGate serialises SetPositionMode and SetAccountMode, each
	// holding it from its request to its cache update, so that a slower reply
	// to an earlier switch cannot overwrite a later switch's mode or level. A
	// switch waiting for it gives up when its context ends.
	accountModeSwitchGate     chan struct{}
	accountModeSwitchGateOnce sync.Once
	// websocketLogins holds the credentials each authenticated websocket
	// connection logged in with: OKX acts for that account on every operation
	// the connection carries, whatever credentials the request names, and
	// pushes that account's private data on it.
	websocketLoginsMu sync.Mutex
	websocketLogins   map[websocket.Connection]websocketLogin
	// websocketReconnecting is set while a reconnection OKX's service upgrade
	// notice asked for is under way, so that the notices of several connections
	// reconnect the websocket once
	websocketReconnecting atomic.Bool
	// subscriptionRequests holds the websocket subscribe and unsubscribe
	// requests sent until their outcome is tracked
	subscriptionRequests subscriptionRequests
	// orderBookStates orders the pushes applied to each websocket order book and
	// holds the channel each was loaded from and the recoveries waiting for a
	// push of their channel
	orderBookStates orderBookStates
}

const baseURL = "https://www.okx.com/"

const apiURL = baseURL + apiPath

const apiPath = "api/v5/"

const websocketURL = "wss://ws.okx.com/ws/v5/"

const apiWebsocketPublicURL = websocketURL + "public"

const apiWebsocketPrivateURL = websocketURL + "private"

const expiryDateLayout = "20060102"

/*
SendHTTPRequest sends an http request, optionally with a JSON payload
URL arguments must be encoded in the request path
result must be a pointer
The response data is unmarshalled directly into result first and then (if needed) from the first array element
A *RESTResponse result receives the whole envelope instead, for endpoints that send fields beside the data
*/
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, f request.EndpointLimit, httpMethod, requestPath string, data, result any, requestType request.AuthType) (err error) {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	resp := new(RESTResponse)
	newRequest := func() (*request.Item, error) {
		var payload []byte
		if data != nil {
			payload, err = json.Marshal(data)
			if err != nil {
				return nil, err
			}
		}
		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		if simulate, okay := ctx.Value(testNetVal).(bool); okay && simulate {
			headers["x-simulated-trading"] = "1"
		}
		if requestType == request.AuthenticatedRequest {
			creds, err := e.GetCredentials(ctx)
			if err != nil {
				return nil, err
			}
			signPath := "/" + apiPath + requestPath
			utcTime := time.Now().UTC().Format(time.RFC3339)
			hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(utcTime+httpMethod+signPath+string(payload)), []byte(creds.Secret))
			if err != nil {
				return nil, err
			}
			headers["OK-ACCESS-KEY"] = creds.Key
			headers["OK-ACCESS-SIGN"] = base64.StdEncoding.EncodeToString(hmac)
			headers["OK-ACCESS-TIMESTAMP"] = utcTime
			headers["OK-ACCESS-PASSPHRASE"] = creds.ClientID
		}
		return &request.Item{
			Method:                 strings.ToUpper(httpMethod),
			Path:                   endpoint + requestPath,
			Headers:                headers,
			Body:                   bytes.NewBuffer(payload),
			Result:                 resp,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}
	if err := e.SendPayload(ctx, f, newRequest, requestType); err != nil {
		return err
	}
	if resp.Code.Int64() != 0 {
		// A failed batch reply can still carry per-order results for the parts
		// that succeeded; decode best-effort so callers can report them.
		decodeErr := unmarshalResponseData(resp.Data, result)
		if requestType == request.AuthenticatedRequest {
			err = request.ErrAuthRequestFailed
		}
		// OKX sends a partial success with a message, which would return below
		// without the sentinel callers match to record the per-order results.
		// Rows that failed to decode can be half populated, so they are not
		// vouched for.
		if resp.Code.Int64() == 2 {
			if decodeErr != nil {
				return common.AppendError(err, fmt.Errorf("error code: `2`; message: %q: %w", resp.Message, decodeErr))
			}
			return common.AppendError(err, errPartialSuccess)
		}
		return common.AppendError(err, &ResponseError{Code: resp.Code.Int64(), Message: resp.Message})
	}
	if envelope, ok := result.(*RESTResponse); ok {
		*envelope = *resp
		return nil
	}
	return unmarshalResponseData(resp.Data, result)
}

// ResponseError is the error code and message OKX answers a failed request with
type ResponseError struct {
	Code    int64
	Message string
}

// Error returns OKX's message with its code, or the code's documented error
// when OKX sent no message
func (r *ResponseError) Error() string {
	if r.Message != "" {
		return fmt.Sprintf("error code: `%d`; message: %q", r.Code, r.Message)
	}
	if mErr, ok := ErrorCodes[strconv.FormatInt(r.Code, 10)]; ok {
		return mErr.Error()
	}
	return fmt.Sprintf("error code: `%d`", r.Code)
}

// Unwrap returns the code's documented error, or nil for an undocumented code
func (r *ResponseError) Unwrap() error {
	return ErrorCodes[strconv.FormatInt(r.Code, 10)]
}

// unmarshalResponseData decodes response data into result, which is usually
// the data itself; some endpoints wrap a single item in an array for a
// non-slice result.
func unmarshalResponseData(data json.RawMessage, result any) error {
	// Most endpoints can be unmarshalled directly (objects and full arrays).
	directErr := json.Unmarshal(data, result)
	if directErr == nil {
		return nil
	}

	// Some endpoints return a single item wrapped in data:[{...}] for a non-slice result.
	var items []json.RawMessage
	if sliceErr := json.Unmarshal(data, &items); sliceErr != nil {
		return fmt.Errorf("cannot unmarshal response data directly (error: %w) or as an array (error: %w)", directErr, sliceErr)
	}
	if len(items) != 1 {
		return fmt.Errorf("cannot unmarshal response data directly (error: %w) or as a single-item array (got %d items)", directErr, len(items))
	}
	if firstErr := json.Unmarshal(items[0], result); firstErr != nil {
		return fmt.Errorf("cannot unmarshal response data directly (error: %w) or as the first array item (error: %w)", directErr, firstErr)
	}
	return nil
}

func getStatusError(statusCode uint64, statusMessage string) error {
	if statusCode == 0 {
		return nil
	}
	return fmt.Errorf("status code: `%d` status message: %q", statusCode, statusMessage)
}
