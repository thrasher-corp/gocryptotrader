package lbank

import (
	"bytes"
	"context"
	"crypto"
	"crypto/md5" //nolint:gosec // Used for this exchange
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Lbank
type Exchange struct {
	exchange.Base
	privateKey *rsa.PrivateKey
}

const (
	lbankAPIURL      = "https://api.lbkex.com"
	lbankAPIVersion1 = "1"
	lbankAPIVersion2 = "2"
	lbankFeeNotFound = 0.0
	tradeBaseURL     = "https://www.lbank.com/trade/"

	// Public endpoints
	lbankTicker         = "ticker.do"
	lbankCurrencyPairs  = "currencyPairs.do"
	lbankMarketDepths   = "depth.do"
	lbankTrades         = "trades.do"
	lbankKlines         = "kline.do"
	lbankPairInfo       = "accuracy.do"
	lbankUSD2CNYRate    = "usdToCny.do"
	lbankWithdrawConfig = "withdrawConfigs.do"

	// Authenticated endpoints
	lbankUserInfo                = "user_info.do"
	lbankPlaceOrder              = "create_order.do"
	lbankCancelOrder             = "cancel_order.do"
	lbankQueryOrder              = "orders_info.do"
	lbankQueryHistoryOrder       = "orders_info_history.do"
	lbankOrderTransactionDetails = "order_transaction_detail.do"
	lbankPastTransactions        = "transaction_history.do"
	lbankOpeningOrders           = "orders_info_no_deal.do"
	lbankWithdrawalRecords       = "withdraws.do"
	lbankWithdraw                = "withdraw.do"
	lbankRevokeWithdraw          = "withdrawCancel.do"
	lbankTimestamp               = "timestamp.do"
)

var (
	errPEMBlockIsNil           = errors.New("pem block is nil")
	errUnableToParsePrivateKey = errors.New("unable to parse private key")
	errPrivateKeyNotLoaded     = errors.New("private key not loaded")
)

// GetTicker returns a ticker for the specified symbol
// symbol: eth_btc
func (e *Exchange) GetTicker(ctx context.Context, symbol string) (*TickerResponse, error) {
	var t *TickerResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := fmt.Sprintf("/v%s/%s?%s", lbankAPIVersion1, lbankTicker, params.Encode())
	return t, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &t)
}

// GetTimestamp returns a timestamp
func (e *Exchange) GetTimestamp(ctx context.Context) (time.Time, error) {
	var resp TimestampResponse
	path := fmt.Sprintf("/v%s/%s", lbankAPIVersion2, lbankTimestamp)
	err := e.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
	if err != nil {
		return time.Time{}, err
	}
	return resp.Timestamp.Time(), nil
}

// GetTickers returns all tickers
func (e *Exchange) GetTickers(ctx context.Context) ([]TickerResponse, error) {
	var t []TickerResponse
	params := url.Values{}
	params.Set("symbol", "all")
	path := fmt.Sprintf("/v%s/%s?%s", lbankAPIVersion1, lbankTicker, params.Encode())
	return t, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &t)
}

// GetCurrencyPairs returns a list of supported currency pairs by the exchange
func (e *Exchange) GetCurrencyPairs(ctx context.Context) ([]string, error) {
	path := fmt.Sprintf("/v%s/%s", lbankAPIVersion1,
		lbankCurrencyPairs)
	var result []string
	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &result)
}

// GetMarketDepths returns arrays of asks, bids and timestamp
func (e *Exchange) GetMarketDepths(ctx context.Context, symbol string, size uint64) (*MarketDepthResponse, error) {
	var m MarketDepthResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("size", strconv.FormatUint(size, 10))
	path := fmt.Sprintf("/v%s/%s?%s", lbankAPIVersion2, lbankMarketDepths, params.Encode())
	return &m, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &m)
}

// GetTrades returns an array of available trades regarding a particular exchange
// The time parameter is optional, if provided it will return trades after the given time
func (e *Exchange) GetTrades(ctx context.Context, symbol string, limit uint64, tm time.Time) ([]TradeResponse, error) {
	var g []TradeResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("size", strconv.FormatUint(limit, 10))
	}
	if !tm.IsZero() {
		params.Set("time", strconv.FormatInt(tm.Unix(), 10))
	}
	path := fmt.Sprintf("/v%s/%s?%s", lbankAPIVersion1, lbankTrades, params.Encode())
	return g, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &g)
}

// GetKlines returns kline data
func (e *Exchange) GetKlines(ctx context.Context, symbol, size, klineType string, tm time.Time) ([]KlineResponse, error) {
	var klineTemp [][]float64
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("size", size)
	params.Set("type", klineType)
	params.Set("time", strconv.FormatInt(tm.Unix(), 10))
	path := fmt.Sprintf("/v%s/%s?%s", lbankAPIVersion1, lbankKlines, params.Encode())
	err := e.SendHTTPRequest(ctx, exchange.RestSpot, path, &klineTemp)
	if err != nil {
		return nil, err
	}

	k := make([]KlineResponse, len(klineTemp))
	for x := range klineTemp {
		if len(klineTemp[x]) < 6 {
			return nil, errors.New("unexpected kline data length")
		}
		k[x] = KlineResponse{
			TimeStamp:     time.Unix(int64(klineTemp[x][0]), 0).UTC(),
			OpenPrice:     klineTemp[x][1],
			HighestPrice:  klineTemp[x][2],
			LowestPrice:   klineTemp[x][3],
			ClosePrice:    klineTemp[x][4],
			TradingVolume: klineTemp[x][5],
		}
	}
	return k, nil
}

// GetUserInfo gets users account info
func (e *Exchange) GetUserInfo(ctx context.Context) (InfoFinalResponse, error) {
	var resp InfoFinalResponse
	path := fmt.Sprintf("/v%s/%s?", lbankAPIVersion1, lbankUserInfo)
	err := e.SendAuthHTTPRequest(ctx, http.MethodPost, path, nil, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// CreateOrder creates an order
func (e *Exchange) CreateOrder(ctx context.Context, pair, side string, amount, price float64) (CreateOrderResponse, error) {
	var resp CreateOrderResponse
	if !strings.EqualFold(side, order.Buy.String()) && !strings.EqualFold(side, order.Sell.String()) {
		return resp, order.ErrSideIsInvalid
	}
	if amount <= 0 {
		return resp, limits.ErrAmountBelowMin
	}
	if price <= 0 {
		return resp, limits.ErrPriceBelowMin
	}

	params := url.Values{}
	params.Set("symbol", pair)
	params.Set("type", strings.ToLower(side))
	params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	path := fmt.Sprintf("/v%s/%s?", lbankAPIVersion1, lbankPlaceOrder)
	err := e.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// RemoveOrder cancels a given order
func (e *Exchange) RemoveOrder(ctx context.Context, pair, orderID string) (RemoveOrderResponse, error) {
	var resp RemoveOrderResponse
	params := url.Values{}
	params.Set("symbol", pair)
	params.Set("order_id", orderID)
	path := fmt.Sprintf("/v%s/%s", lbankAPIVersion1, lbankCancelOrder)
	err := e.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// QueryOrder finds out information about orders (can pass up to 3 comma separated values to this)
// Lbank returns an empty string as their []OrderResponse instead of returning an empty array, so when len(tempResp.Orders) > 2 its not empty and should be unmarshalled separately
func (e *Exchange) QueryOrder(ctx context.Context, pair, orderIDs string) (QueryOrderFinalResponse, error) {
	var resp QueryOrderFinalResponse
	var tempResp QueryOrderResponse
	params := url.Values{}
	params.Set("symbol", pair)
	params.Set("order_id", orderIDs)
	path := fmt.Sprintf("/v%s/%s?", lbankAPIVersion1, lbankQueryOrder)
	err := e.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &tempResp)
	if err != nil {
		return resp, err
	}

	var totalOrders []OrderResponse
	if len(tempResp.Orders) > 2 {
		err = json.Unmarshal(tempResp.Orders, &totalOrders)
		if err != nil {
			return resp, err
		}
	}
	resp.ErrCapture = tempResp.ErrCapture
	resp.Orders = totalOrders

	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// QueryOrderHistory finds order info in the past 2 days
// Lbank returns an empty string as their []OrderResponse instead of returning an empty array, so when len(tempResp.Orders) > 2 its not empty and should be unmarshalled separately
func (e *Exchange) QueryOrderHistory(ctx context.Context, pair, pageNumber, pageLength string) (OrderHistoryFinalResponse, error) {
	var resp OrderHistoryFinalResponse
	var tempResp OrderHistoryResponse
	params := url.Values{}
	params.Set("symbol", pair)
	params.Set("current_page", pageNumber)
	params.Set("page_length", pageLength)
	path := fmt.Sprintf("/v%s/%s?", lbankAPIVersion1, lbankQueryHistoryOrder)
	err := e.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &tempResp)
	if err != nil {
		return resp, err
	}

	var totalOrders []OrderResponse
	if len(tempResp.Orders) > 2 {
		err = json.Unmarshal(tempResp.Orders, &totalOrders)
		if err != nil {
			return resp, err
		}
	}
	resp.ErrCapture = tempResp.ErrCapture
	resp.PageLength = tempResp.PageLength
	resp.Orders = totalOrders
	resp.CurrentPage = tempResp.CurrentPage

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// GetPairInfo finds information about all trading pairs
func (e *Exchange) GetPairInfo(ctx context.Context) ([]PairInfoResponse, error) {
	var resp []PairInfoResponse
	path := fmt.Sprintf("/v%s/%s?", lbankAPIVersion1, lbankPairInfo)
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// OrderTransactionDetails gets info about transactions
func (e *Exchange) OrderTransactionDetails(ctx context.Context, symbol, orderID string) (TransactionHistoryResp, error) {
	var resp TransactionHistoryResp
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("order_id", orderID)
	path := fmt.Sprintf("/v%s/%s?", lbankAPIVersion1, lbankOrderTransactionDetails)
	err := e.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// TransactionHistory stores info about transactions
func (e *Exchange) TransactionHistory(ctx context.Context, symbol, transactionType, startDate, endDate, from, direct, size string) (TransactionHistoryResp, error) {
	var resp TransactionHistoryResp
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("type", transactionType)
	params.Set("start_date", startDate)
	params.Set("end_date", endDate)
	params.Set("from", from)
	params.Set("direct", direct)
	params.Set("size", size)
	path := fmt.Sprintf("/v%s/%s?", lbankAPIVersion1, lbankPastTransactions)
	err := e.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// GetOpenOrders gets opening orders
// Lbank returns an empty string as their []OrderResponse instead of returning an empty array, so when len(tempResp.Orders) > 2 its not empty and should be unmarshalled separately
func (e *Exchange) GetOpenOrders(ctx context.Context, pair, pageNumber, pageLength string) (OpenOrderFinalResponse, error) {
	var resp OpenOrderFinalResponse
	var tempResp OpenOrderResponse
	params := url.Values{}
	params.Set("symbol", pair)
	params.Set("current_page", pageNumber)
	params.Set("page_length", pageLength)
	path := fmt.Sprintf("/v%s/%s", lbankAPIVersion1, lbankOpeningOrders)
	err := e.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &tempResp)
	if err != nil {
		return resp, err
	}

	var totalOrders []OrderResponse
	if len(tempResp.Orders) > 2 {
		err = json.Unmarshal(tempResp.Orders, &totalOrders)
		if err != nil {
			return resp, err
		}
	}
	resp.ErrCapture = tempResp.ErrCapture
	resp.PageLength = tempResp.PageLength
	resp.PageNumber = tempResp.PageNumber
	resp.Orders = totalOrders

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// USD2RMBRate finds USD-CNY Rate
func (e *Exchange) USD2RMBRate(ctx context.Context) (ExchangeRateResponse, error) {
	var resp ExchangeRateResponse
	path := fmt.Sprintf("/v%s/%s", lbankAPIVersion1, lbankUSD2CNYRate)
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetWithdrawConfig gets information about withdrawals
func (e *Exchange) GetWithdrawConfig(ctx context.Context, c currency.Code) ([]WithdrawConfigResponse, error) {
	var resp []WithdrawConfigResponse
	params := url.Values{}
	params.Set("assetCode", c.Lower().String())
	path := fmt.Sprintf("/v%s/%s?%s", lbankAPIVersion1, lbankWithdrawConfig, params.Encode())
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// Withdraw sends a withdrawal request
func (e *Exchange) Withdraw(ctx context.Context, account, assetCode, amount, memo, mark, withdrawType string) (WithdrawResponse, error) {
	var resp WithdrawResponse
	params := url.Values{}
	params.Set("account", account)
	params.Set("assetCode", assetCode)
	params.Set("amount", amount)
	if memo != "" {
		params.Set("memo", memo)
	}
	if mark != "" {
		params.Set("mark", mark)
	}
	if withdrawType != "" {
		params.Set("type", withdrawType)
	}
	path := fmt.Sprintf("/v%s/%s", lbankAPIVersion1,
		lbankWithdraw)
	err := e.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// RevokeWithdraw cancels the withdrawal given the withdrawalID
func (e *Exchange) RevokeWithdraw(ctx context.Context, withdrawID string) (RevokeWithdrawResponse, error) {
	var resp RevokeWithdrawResponse
	params := url.Values{}
	if withdrawID != "" {
		params.Set("withdrawId", withdrawID)
	}
	path := fmt.Sprintf("/v%s/%s?", lbankAPIVersion1, lbankRevokeWithdraw)
	err := e.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// GetWithdrawalRecords gets withdrawal records
func (e *Exchange) GetWithdrawalRecords(ctx context.Context, assetCode string, pageNo, status, pageSize int64) (WithdrawalResponse, error) {
	var resp WithdrawalResponse
	params := url.Values{}
	params.Set("assetCode", assetCode)
	params.Set("status", strconv.FormatInt(status, 10))
	params.Set("pageNo", strconv.FormatInt(pageNo, 10))
	params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	path := fmt.Sprintf("/v%s/%s", lbankAPIVersion1, lbankWithdrawalRecords)
	err := e.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// ErrorCapture captures errors
func ErrorCapture(code int64) error {
	msg, ok := errorCodes[code]
	if !ok {
		return fmt.Errorf("undefined code please check api docs for error code definition: %v", code)
	}
	return errors.New(msg)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	item := &request.Item{
		Method:                 http.MethodGet,
		Path:                   endpoint + path,
		Result:                 result,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}

	return e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

func (e *Exchange) loadPrivKey(ctx context.Context) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	key := strings.Join([]string{
		"-----BEGIN RSA PRIVATE KEY-----",
		creds.Secret,
		"-----END RSA PRIVATE KEY-----",
	}, "\n")

	block, _ := pem.Decode([]byte(key))
	if block == nil {
		return errPEMBlockIsNil
	}

	p, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("%w: %w", errUnableToParsePrivateKey, err)
	}

	var ok bool
	e.privateKey, ok = p.(*rsa.PrivateKey)
	if !ok {
		return common.GetTypeAssertError("*rsa.PrivateKey", p)
	}
	return nil
}

func (e *Exchange) sign(data string) (string, error) {
	if e.privateKey == nil {
		return "", errPrivateKeyNotLoaded
	}
	md5sum := md5.Sum([]byte(data)) //nolint:gosec // Used for this exchange
	shasum := sha256.Sum256([]byte(strings.ToUpper(hex.EncodeToString(md5sum[:]))))
	r, err := rsa.SignPKCS1v15(rand.Reader, e.privateKey, crypto.SHA256, shasum[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(r), nil
}

// SendAuthHTTPRequest sends an authenticated request
func (e *Exchange) SendAuthHTTPRequest(ctx context.Context, method, endpoint string, vals url.Values, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}

	if vals == nil {
		vals = url.Values{}
	}

	vals.Set("api_key", creds.Key)
	sig, err := e.sign(vals.Encode())
	if err != nil {
		return err
	}

	vals.Set("sign", sig)
	payload := vals.Encode()
	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	item := &request.Item{
		Method:                 method,
		Path:                   endpoint,
		Headers:                headers,
		Result:                 result,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}

	return e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		item.Body = bytes.NewBufferString(payload)
		return item, nil
	}, request.AuthenticatedRequest)
}
