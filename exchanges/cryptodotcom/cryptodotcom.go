package cryptodotcom

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Cryptodotcom is the overarching type across this package
type Cryptodotcom struct {
	exchange.Base
}

const (
	// cryptodotcom API endpoints.
	cryptodotcomUATSandboxAPIURL = "https://uat-api.3ona.co"
	cryptodotcomAPIURL           = "https://api.crypto.com"

	// cryptodotcom websocket endpoints.
	cryptodotcomUATSandboxWebsocketURL = "wss://uat-stream.3ona.co"
	cryptodotcomWebsocketURL           = "wss://stream.crypto.com"

	cryptodotcomAPIVersion = "/v2/"

	// Public endpoints
	publicAuth                      = "public/auth"
	publicInstruments               = "public/get-instruments"
	publicOrderbook                 = "public/get-book"
	publicCandlestick               = "public/get-candlestick"
	publicTicker                    = "public/get-ticker"
	publicTrades                    = "public/get-trades"
	publicGetValuations             = "public/get-valuations"               // skipped
	publicGetExpiredSettlementPrice = "public/get-expired-settlement-price" // skipped
	publicGetInsurance              = "public/get-insurance"                // skipped

	// Authenticated endpoints
	privateSetCancelOnDisconnect = "private/set-cancel-on-disconnect"
	privateGetCancelOnDisconnect = "private/get-cancel-on-disconnect"

	privateUserBalance              = "private/user-balance"
	privateUserBalanceHistory       = "private/user-balance-history"
	privateCreateSubAccountTransfer = "private/create-sub-account-transfer"
	privateGetSubAccountBalances    = "private/get-sub-account-balances"
	privateGetPositions             = "private/get-positions"

	privateCreateOrder           = "private/create-order"
	privateCancelOrder           = "private/cancel-order"
	privateCreateOrderList       = "private/create-order-list"
	privateCancelOrderList       = "private/cancel-order-list"
	privateGetOrderList          = "private/get-order-list"
	privateCancelAllOrders       = "private/cancel-all-orders"
	privateClosePosition         = "private/close-position"
	privateGetOrderHistory       = "private/get-order-history"
	privateGetOpenOrders         = "private/get-open-orders"
	privateGetOrderDetail        = "private/get-order-detail"
	privateGetTrades             = "private/get-trades"
	privateChangeAccountLeverage = "private/change-account-leverage"
	privateGetTransactions       = "private/get-transactions"

	// Wallet management API
	postWithdrawal = "private/create-withdrawal"

	privateGetCurrencyNetworks = "private/get-currency-networks"
	privategetDepositAddress   = "private/get-deposit-address"
	privateGetAccounts         = "private/get-accounts"

	privateGetWithdrawalHistory = "private/get-withdrawal-history"
	privateGetDepositHistory    = "private/get-deposit-history"

	// Spot Trading API
	privateGetAccountSummary = "private/get-account-summary"
)

// GetSymbols provides information on all supported instruments
func (cr *Cryptodotcom) GetInstruments(ctx context.Context) ([]Instrument, error) {
	resp := &struct {
		Instruments []Instrument `json:"instruments"`
	}{}
	return resp.Instruments, cr.SendHTTPRequest(ctx, exchange.RestSpot, publicInstruments, request.Unset, &resp)
}

// GetOrderbook retches the public order book for a particular instrument and depth.
func (cr *Cryptodotcom) GetOrderbook(ctx context.Context, instrumentName string, depth int64) (*OrderbookDetail, error) {
	params := url.Values{}
	if instrumentName == "" {
		return nil, errSymbolIsRequired
	}
	params.Set("instrument_name", instrumentName)
	if depth != 0 {
		params.Set("depth", strconv.FormatInt(depth, 10))
	}
	var resp *OrderbookDetail
	return resp, cr.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(publicOrderbook, params), request.Unset, &resp)
}

// GetCandlestickDetail retrieves candlesticks (k-line data history) over a given period for an instrument
func (cr *Cryptodotcom) GetCandlestickDetail(ctx context.Context, instrumentName string, interval kline.Interval) (*CandlestickDetail, error) {
	if instrumentName == "" {
		return nil, errSymbolIsRequired
	}
	params := url.Values{}
	params.Set("instrument_name", instrumentName)
	if intervalString, err := intervalToString(interval); err == nil {
		params.Set("timeframe", intervalString)
	}
	var resp *CandlestickDetail
	return resp, cr.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(publicCandlestick, params), request.Unset, &resp)
}

// GetTicker fetches the public tickers for an instrument.
func (cr *Cryptodotcom) GetTicker(ctx context.Context, instrumentName string) (*TickersResponse, error) {
	params := url.Values{}
	if instrumentName == "" {
		return nil, errSymbolIsRequired
	}
	if instrumentName != "" {
		params.Set("instrument_name", instrumentName)
	}
	var resp *TickersResponse
	return resp, cr.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(publicTicker, params), request.Unset, &resp)
}

// GetTrades fetches the public trades for a particular instrument.
func (cr *Cryptodotcom) GetTrades(ctx context.Context, instrumentName string) (*TradesResponse, error) {
	if instrumentName == "" {
		return nil, errSymbolIsRequired
	}
	params := url.Values{}
	params.Set("instrument_name", instrumentName)
	var resp *TradesResponse
	return resp, cr.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(publicTrades, params), request.Unset, &resp)
}

// Private endpoints

// WithdrawFunds creates a withdrawal request. Withdrawal setting must be enabled for your API Key. If you do not see the option when viewing your API Key, this feature is not yet available for you.
// Withdrawal addresses must first be whitelisted in your account’s Withdrawal Whitelist page.
// Withdrawal fees and minimum withdrawal amount can be found on the Fees & Limits page on the Exchange website.
func (cr *Cryptodotcom) WithdrawFunds(ctx context.Context, ccy currency.Code, amount float64, address, addressTag, networkID, clientWithdrawalID string) (*WithdrawalItem, error) {
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	if amount <= 0 {
		return nil, fmt.Errorf("%w, withdrawal amount provided: %f", errInvalidAmount, amount)
	}
	if address == "" {
		return nil, errors.New("address is required")
	}
	params := make(map[string]interface{})
	params["currency"] = ccy.String()
	params["amount"] = amount
	params["address"] = address
	if clientWithdrawalID != "" {
		params["client_wid"] = clientWithdrawalID
	}
	if addressTag != "" {
		params["address_tag"] = addressTag
	}
	if networkID != "" {
		params["network_id"] = networkID
	}
	var resp *WithdrawalItem
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.Unset, postWithdrawal, params, &resp)
}

// GetCurrencyNetworks retrives the symbol network mapping.
func (cr *Cryptodotcom) GetCurrencyNetworks(ctx context.Context) (*CurrencyNetworkResponse, error) {
	var resp *CurrencyNetworkResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.Unset, privateGetCurrencyNetworks, nil, &resp)
}

// GetWithdrawalHistory retrives accounts withdrawal history.
func (cr *Cryptodotcom) GetWithdrawalHistory(ctx context.Context) (*WithdrawalResponse, error) {
	var resp *WithdrawalResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.Unset, privateGetWithdrawalHistory, nil, &resp)
}

// GetDepositHistory retrives deposit history. Withdrawal setting must be enabled for your API Key. If you do not see the option when viewing your API Keys, this feature is not yet available for you.
func (cr *Cryptodotcom) GetDepositHistory(ctx context.Context, ccy currency.Code, startTimestamp, endTimestamp time.Time, pageSize, page, status int64) (*DepositResponse, error) {
	params := make(map[string]interface{})
	if ccy.IsEmpty() {
		params["currency"] = ccy.String()
	}
	if !startTimestamp.IsZero() {
		params["start_ts"] = strconv.FormatInt(startTimestamp.UnixMilli(), 10)
	}
	if !endTimestamp.IsZero() {
		params["end_ts"] = strconv.FormatInt(endTimestamp.UnixMilli(), 10)
	}
	// Page size (Default: 20, Max: 200)
	if pageSize != 0 {
		params["page_size"] = strconv.FormatInt(pageSize, 10)
	}
	if page != 0 {
		params["page"] = strconv.FormatInt(page, 10)
	}
	// 0 - Pending, 1 - Processing, 2 - Rejected, 3 - Payment In-progress, 4 - Payment Failed, 5 - Completed, 6 - Cancelled
	if status != 0 {
		params["status"] = status
	}
	var resp *DepositResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.Unset, privateGetDepositHistory, params, &resp)
}

// GetPersonalDepositAddress fetches deposit address. Withdrawal setting must be enabled for your API Key. If you do not see the option when viewing your API Keys, this feature is not yet available for you.
func (cr *Cryptodotcom) GetPersonalDepositAddress(ctx context.Context, ccy currency.Code) (*DepositAddresses, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := make(map[string]interface{})
	params["currency"] = ccy.String()
	var resp *DepositAddresses
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.Unset, privategetDepositAddress, params, &resp)
}

// SPOT Trading API endpoints.

// GetAccountSummary returns the account balance of a user for a particular token.
func (cr *Cryptodotcom) GetAccountSummary(ctx context.Context, ccy currency.Code) (*Accounts, error) {
	params := make(map[string]interface{})
	if !ccy.IsEmpty() {
		params["currency"] = ccy.String()
	}
	var resp *Accounts
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.Unset, privateGetAccountSummary, params, &resp)
}

// SetCancelOnDisconnect cancel on Disconnect is an optional feature that will cancel all open orders created by the connection upon loss of connectivity between client or server.
func (cr *Cryptodotcom) SetCancelOnDisconnect(ctx context.Context, scope string) (*CancelOnDisconnectScope, error) {
	if scope != "ACCOUNT" && scope != "CONNECTION" {
		return nil, errInvalidOrderCancellationScope
	}
	params := make(map[string]interface{})
	params["scope"] = scope
	var resp *CancelOnDisconnectScope
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.Unset, privateSetCancelOnDisconnect, params, &resp)
}

// intervalToString returns a string representation of interval.
func intervalToString(interval kline.Interval) (string, error) {
	switch interval {
	case kline.OneMin:
		return "1m", nil
	case kline.FiveMin:
		return "5m", nil
	case kline.FifteenMin:
		return "15m", nil
	case kline.ThirtyMin:
		return "30m", nil
	case kline.OneHour:
		return "1h", nil
	case kline.FourHour:
		return "4h", nil
	case kline.SixHour:
		return "6h", nil
	case kline.TwelveHour:
		return "12h", nil
	case kline.OneDay:
		return "1D", nil
	case kline.SevenDay:
		return "7D", nil
	case kline.TwoWeek:
		return "14D", nil
	case kline.OneMonth:
		return "1M", nil
	default:
		return "", fmt.Errorf("%v interval:%v", kline.ErrUnsupportedInterval, interval)
	}
}

// stringToInterval converts a string representation to kline.Interval instance.
func stringToInterval(interval string) (kline.Interval, error) {
	switch interval {
	case "1m":
		return kline.OneMin, nil
	case "5m":
		return kline.FiveMin, nil
	case "15m":
		return kline.FifteenMin, nil
	case "30m":
		return kline.ThirtyMin, nil
	case "1h":
		return kline.OneHour, nil
	case "4h":
		return kline.FourHour, nil
	case "6h":
		return kline.SixHour, nil
	case "12h":
		return kline.TwelveHour, nil
	case "1D":
		return kline.OneDay, nil
	case "7D":
		return kline.SevenDay, nil
	case "14D":
		return kline.TwoWeek, nil
	case "1M":
		return kline.OneMonth, nil
	default:
		return 0, fmt.Errorf("invalid interval string: %s", interval)
	}
}

// SendHTTPRequest send requests for un-authenticated market endpoints.
func (cr *Cryptodotcom) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result interface{}) error {
	endpointPath, err := cr.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	response := &struct {
		ID            int             `json:"id"`
		Method        string          `json:"method"`
		Code          int             `json:"code"`
		Message       string          `json:"message"`
		DetailCode    string          `json:"detail_code"`
		DetailMessage string          `json:"detail_message"`
		Result        json.RawMessage `json:"result"`
	}{}
	err = cr.SendPayload(ctx, f, func() (*request.Item, error) {
		return &request.Item{
			Method:        http.MethodGet,
			Path:          endpointPath + cryptodotcomAPIVersion + path,
			Result:        response,
			Verbose:       cr.Verbose,
			HTTPDebugging: cr.HTTPDebugging,
			HTTPRecording: cr.HTTPRecording,
		}, nil
	})
	if err != nil {
		return err
	}
	if response.Code != 0 {
		mes := fmt.Sprintf("error code: %d Message: %s", response.Code, response.Message)
		if response.DetailCode != "0" && response.DetailCode != "" {
			mes = fmt.Sprintf("%s Detail: %s %s", mes, response.DetailCode, response.DetailMessage)
		}
		return errors.New(mes)
	}
	return json.Unmarshal(response.Result, &result)
}

// SendAuthHTTPRequest sends an authenticated HTTP request to the server
func (cr *Cryptodotcom) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, epl request.EndpointLimit, path string, arg map[string]interface{}, resp interface{}) error {
	creds, err := cr.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := cr.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	response := &struct {
		ID            int             `json:"id"`
		Method        string          `json:"method"`
		Code          int             `json:"code"`
		Message       string          `json:"message"`
		DetailCode    string          `json:"detail_code"`
		DetailMessage string          `json:"detail_message"`
		Result        json.RawMessage `json:"result"`
	}{}
	newRequest := func() (*request.Item, error) {
		timestamp := time.Now()
		var body io.Reader
		var hmac, payload []byte
		var id string
		var idInt int64
		id, err = common.GenerateRandomString(6, common.NumberCharacters)
		if err != nil {
			return nil, err
		}
		idInt, err = strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, err
		}
		signaturePayload := path + strconv.FormatInt(idInt, 10) + creds.Key + cr.getParamString(arg) + strconv.FormatInt(timestamp.UnixMilli(), 10)
		hmac, err = crypto.GetHMAC(crypto.HashSHA256,
			[]byte(signaturePayload),
			[]byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		req := &PrivateRequestParam{
			ID:        idInt,
			Method:    path,
			APIKey:    creds.Key,
			Nonce:     timestamp.UnixMilli(),
			Params:    arg,
			Signature: crypto.HexEncodeToString(hmac),
		}
		payload, err = json.Marshal(req)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(payload)
		return &request.Item{
			Method:        http.MethodPost,
			Path:          endpoint + cryptodotcomAPIVersion + path,
			Headers:       headers,
			Body:          body,
			Result:        response,
			AuthRequest:   true,
			Verbose:       cr.Verbose,
			HTTPDebugging: cr.HTTPDebugging,
			HTTPRecording: cr.HTTPRecording,
		}, nil
	}
	err = cr.SendPayload(ctx, request.Unset, newRequest)
	if err != nil {
		return err
	}
	return json.Unmarshal(response.Result, resp)
}

func (cr *Cryptodotcom) getParamString(params map[string]interface{}) string {
	paramString := ""
	keys := cr.sortParams(params)
	for x := range keys {
		if params[keys[x]] == nil {
			paramString += keys[x] + "null"
		}
		switch reflect.ValueOf(params[keys[x]]).Kind() {
		case reflect.Bool:
			paramString += keys[x] + strconv.FormatBool(params[keys[x]].(bool))
		case reflect.Int64:
			paramString += keys[x] + strconv.FormatInt(params[keys[x]].(int64), 10)
		case reflect.Float32:
			paramString += keys[x] + strconv.FormatFloat(params[keys[x]].(float64), 'f', -1, 64)
		case reflect.Map:
			paramString += keys[x] + cr.getParamString((params[keys[x]]).(map[string]interface{}))
		case reflect.String:
			paramString += keys[x] + params[keys[x]].(string)
		}
	}
	return paramString
}

func (cr *Cryptodotcom) sortParams(params map[string]interface{}) []string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
