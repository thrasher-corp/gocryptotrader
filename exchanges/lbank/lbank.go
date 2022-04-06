package lbank

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	gctcrypto "github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// Lbank is the overarching type across this package
type Lbank struct {
	exchange.Base
	privateKey    *rsa.PrivateKey
	WebsocketConn *stream.WebsocketConnection
}

const (
	lbankAPIURL      = "https://api.lbkex.com"
	lbankAPIVersion  = "1"
	lbankAPIVersion2 = "2"
	lbankFeeNotFound = 0.0

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
)

// GetTicker returns a ticker for the specified symbol
// symbol: eth_btc
func (l *Lbank) GetTicker(ctx context.Context, symbol string) (TickerResponse, error) {
	var t TickerResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := fmt.Sprintf("/v%s/%s?%s", lbankAPIVersion, lbankTicker, params.Encode())
	return t, l.SendHTTPRequest(ctx, exchange.RestSpot, path, &t)
}

// GetTickers returns all tickers
func (l *Lbank) GetTickers(ctx context.Context) ([]TickerResponse, error) {
	var t []TickerResponse
	params := url.Values{}
	params.Set("symbol", "all")
	path := fmt.Sprintf("/v%s/%s?%s", lbankAPIVersion, lbankTicker, params.Encode())
	return t, l.SendHTTPRequest(ctx, exchange.RestSpot, path, &t)
}

// GetCurrencyPairs returns a list of supported currency pairs by the exchange
func (l *Lbank) GetCurrencyPairs(ctx context.Context) ([]string, error) {
	path := fmt.Sprintf("/v%s/%s", lbankAPIVersion,
		lbankCurrencyPairs)
	var result []string
	return result, l.SendHTTPRequest(ctx, exchange.RestSpot, path, &result)
}

// GetMarketDepths returns arrays of asks, bids and timestamp
func (l *Lbank) GetMarketDepths(ctx context.Context, symbol, size, merge string) (MarketDepthResponse, error) {
	var m MarketDepthResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("size", size)
	params.Set("merge", merge)
	path := fmt.Sprintf("/v%s/%s?%s", lbankAPIVersion2, lbankMarketDepths, params.Encode())
	return m, l.SendHTTPRequest(ctx, exchange.RestSpot, path, &m)
}

// GetTrades returns an array of available trades regarding a particular exchange
func (l *Lbank) GetTrades(ctx context.Context, symbol string, limit, time int64) ([]TradeResponse, error) {
	var g []TradeResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("size", strconv.FormatInt(limit, 10))
	}
	if time > 0 {
		params.Set("time", strconv.FormatInt(time, 10))
	}
	path := fmt.Sprintf("/v%s/%s?%s", lbankAPIVersion, lbankTrades, params.Encode())
	return g, l.SendHTTPRequest(ctx, exchange.RestSpot, path, &g)
}

// GetKlines returns kline data
func (l *Lbank) GetKlines(ctx context.Context, symbol, size, klineType, tm string) ([]KlineResponse, error) {
	var klineTemp [][]float64
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("size", size)
	params.Set("type", klineType)
	params.Set("time", tm)
	path := fmt.Sprintf("/v%s/%s?%s", lbankAPIVersion, lbankKlines, params.Encode())
	err := l.SendHTTPRequest(ctx, exchange.RestSpot, path, &klineTemp)
	if err != nil {
		return nil, err
	}

	var k []KlineResponse
	for x := range klineTemp {
		if len(klineTemp[x]) < 6 {
			return nil, errors.New("unexpected kline data length")
		}
		k = append(k, KlineResponse{
			TimeStamp:     time.Unix(int64(klineTemp[x][0]), 0).UTC(),
			OpenPrice:     klineTemp[x][1],
			HigestPrice:   klineTemp[x][2],
			LowestPrice:   klineTemp[x][3],
			ClosePrice:    klineTemp[x][4],
			TradingVolume: klineTemp[x][5],
		})
	}
	return k, nil
}

// GetUserInfo gets users account info
func (l *Lbank) GetUserInfo(ctx context.Context) (InfoFinalResponse, error) {
	var resp InfoFinalResponse
	path := fmt.Sprintf("/v%s/%s?", lbankAPIVersion, lbankUserInfo)
	err := l.SendAuthHTTPRequest(ctx, http.MethodPost, path, nil, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// CreateOrder creates an order
func (l *Lbank) CreateOrder(ctx context.Context, pair, side string, amount, price float64) (CreateOrderResponse, error) {
	var resp CreateOrderResponse
	if !strings.EqualFold(side, order.Buy.String()) &&
		!strings.EqualFold(side, order.Sell.String()) {
		return resp, errors.New("side type invalid can only be 'buy' or 'sell'")
	}
	if amount <= 0 {
		return resp, errors.New("amount can't be smaller than or equal to 0")
	}
	if price <= 0 {
		return resp, errors.New("price can't be smaller than or equal to 0")
	}
	params := url.Values{}

	params.Set("symbol", pair)
	params.Set("type", strings.ToLower(side))
	params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	path := fmt.Sprintf("/v%s/%s?", lbankAPIVersion, lbankPlaceOrder)
	err := l.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// RemoveOrder cancels a given order
func (l *Lbank) RemoveOrder(ctx context.Context, pair, orderID string) (RemoveOrderResponse, error) {
	var resp RemoveOrderResponse
	params := url.Values{}
	params.Set("symbol", pair)
	params.Set("order_id", orderID)
	path := fmt.Sprintf("/v%s/%s", lbankAPIVersion, lbankCancelOrder)
	err := l.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &resp)
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
func (l *Lbank) QueryOrder(ctx context.Context, pair, orderIDs string) (QueryOrderFinalResponse, error) {
	var resp QueryOrderFinalResponse
	var tempResp QueryOrderResponse
	params := url.Values{}
	params.Set("symbol", pair)
	params.Set("order_id", orderIDs)
	path := fmt.Sprintf("/v%s/%s?", lbankAPIVersion, lbankQueryOrder)
	err := l.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &tempResp)
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
func (l *Lbank) QueryOrderHistory(ctx context.Context, pair, pageNumber, pageLength string) (OrderHistoryFinalResponse, error) {
	var resp OrderHistoryFinalResponse
	var tempResp OrderHistoryResponse
	params := url.Values{}
	params.Set("symbol", pair)
	params.Set("current_page", pageNumber)
	params.Set("page_length", pageLength)
	path := fmt.Sprintf("/v%s/%s?", lbankAPIVersion, lbankQueryHistoryOrder)
	err := l.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &tempResp)
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
func (l *Lbank) GetPairInfo(ctx context.Context) ([]PairInfoResponse, error) {
	var resp []PairInfoResponse
	path := fmt.Sprintf("/v%s/%s?", lbankAPIVersion, lbankPairInfo)
	return resp, l.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// OrderTransactionDetails gets info about transactions
func (l *Lbank) OrderTransactionDetails(ctx context.Context, symbol, orderID string) (TransactionHistoryResp, error) {
	var resp TransactionHistoryResp
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("order_id", orderID)
	path := fmt.Sprintf("/v%s/%s?", lbankAPIVersion, lbankOrderTransactionDetails)
	err := l.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// TransactionHistory stores info about transactions
func (l *Lbank) TransactionHistory(ctx context.Context, symbol, transactionType, startDate, endDate, from, direct, size string) (TransactionHistoryResp, error) {
	var resp TransactionHistoryResp
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("type", transactionType)
	params.Set("start_date", startDate)
	params.Set("end_date", endDate)
	params.Set("from", from)
	params.Set("direct", direct)
	params.Set("size", size)
	path := fmt.Sprintf("/v%s/%s?", lbankAPIVersion, lbankPastTransactions)
	err := l.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &resp)
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
func (l *Lbank) GetOpenOrders(ctx context.Context, pair, pageNumber, pageLength string) (OpenOrderFinalResponse, error) {
	var resp OpenOrderFinalResponse
	var tempResp OpenOrderResponse
	params := url.Values{}
	params.Set("symbol", pair)
	params.Set("current_page", pageNumber)
	params.Set("page_length", pageLength)
	path := fmt.Sprintf("/v%s/%s", lbankAPIVersion, lbankOpeningOrders)
	err := l.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &tempResp)
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
func (l *Lbank) USD2RMBRate(ctx context.Context) (ExchangeRateResponse, error) {
	var resp ExchangeRateResponse
	path := fmt.Sprintf("/v%s/%s", lbankAPIVersion, lbankUSD2CNYRate)
	return resp, l.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetWithdrawConfig gets information about withdrawals
func (l *Lbank) GetWithdrawConfig(ctx context.Context, assetCode string) ([]WithdrawConfigResponse, error) {
	var resp []WithdrawConfigResponse
	params := url.Values{}
	params.Set("assetCode", assetCode)
	path := fmt.Sprintf("/v%s/%s?%s", lbankAPIVersion, lbankWithdrawConfig, params.Encode())
	return resp, l.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// Withdraw sends a withdrawal request
func (l *Lbank) Withdraw(ctx context.Context, account, assetCode, amount, memo, mark, withdrawType string) (WithdrawResponse, error) {
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
	path := fmt.Sprintf("/v%s/%s", lbankAPIVersion,
		lbankWithdraw)
	err := l.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// RevokeWithdraw cancels the withdrawal given the withdrawalID
func (l *Lbank) RevokeWithdraw(ctx context.Context, withdrawID string) (RevokeWithdrawResponse, error) {
	var resp RevokeWithdrawResponse
	params := url.Values{}
	if withdrawID != "" {
		params.Set("withdrawId", withdrawID)
	}
	path := fmt.Sprintf("/v%s/%s?", lbankAPIVersion, lbankRevokeWithdraw)
	err := l.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// GetWithdrawalRecords gets withdrawal records
func (l *Lbank) GetWithdrawalRecords(ctx context.Context, assetCode, status, pageNo, pageSize string) (WithdrawalResponse, error) {
	var resp WithdrawalResponse
	params := url.Values{}
	params.Set("assetCode", assetCode)
	params.Set("status", status)
	params.Set("pageNo", pageNo)
	params.Set("pageSize", pageSize)
	path := fmt.Sprintf("/v%s/%s", lbankAPIVersion, lbankWithdrawalRecords)
	err := l.SendAuthHTTPRequest(ctx, http.MethodPost, path, params, &resp)
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
func (l *Lbank) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result interface{}) error {
	endpoint, err := l.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       l.Verbose,
		HTTPDebugging: l.HTTPDebugging,
		HTTPRecording: l.HTTPRecording,
	}

	return l.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	})
}

func (l *Lbank) loadPrivKey(ctx context.Context) error {
	creds, err := l.GetCredentials(ctx)
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
		return errors.New("pem block is nil")
	}

	p, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("unable to decode priv key: %s", err)
	}

	var ok bool
	l.privateKey, ok = p.(*rsa.PrivateKey)
	if !ok {
		return errors.New("unable to parse RSA private key")
	}
	return nil
}

func (l *Lbank) sign(data string) (string, error) {
	if l.privateKey == nil {
		return "", errors.New("private key not loaded")
	}
	md5hash, err := gctcrypto.GetMD5([]byte(data))
	if err != nil {
		return "", err
	}
	m := strings.ToUpper(gctcrypto.HexEncodeToString(md5hash))
	s, err := gctcrypto.GetSHA256([]byte(m))
	if err != nil {
		return "", err
	}
	r, err := rsa.SignPKCS1v15(rand.Reader, l.privateKey, crypto.SHA256, s)
	if err != nil {
		return "", err
	}
	return gctcrypto.Base64Encode(r), nil
}

// SendAuthHTTPRequest sends an authenticated request
func (l *Lbank) SendAuthHTTPRequest(ctx context.Context, method, endpoint string, vals url.Values, result interface{}) error {
	creds, err := l.GetCredentials(ctx)
	if err != nil {
		return err
	}

	if vals == nil {
		vals = url.Values{}
	}

	vals.Set("api_key", creds.Key)
	sig, err := l.sign(vals.Encode())
	if err != nil {
		return err
	}

	vals.Set("sign", sig)
	payload := vals.Encode()
	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	item := &request.Item{
		Method:        method,
		Path:          endpoint,
		Headers:       headers,
		Result:        result,
		AuthRequest:   true,
		Verbose:       l.Verbose,
		HTTPDebugging: l.HTTPDebugging,
		HTTPRecording: l.HTTPRecording,
	}

	return l.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		item.Body = bytes.NewBufferString(payload)
		return item, nil
	})
}
