package lbank

import (
	"bytes"
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

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Lbank is the overarching type across this package
type Lbank struct {
	exchange.Base
	privateKey    *rsa.PrivateKey
	WebsocketConn *wshandler.WebsocketConnection
}

const (
	lbankAPIURL          = "https://api.lbkex.com"
	lbankAPIVersion      = "1"
	lbankAuthRateLimit   = 0
	lbankUnAuthRateLimit = 0
	lbankFeeNotFound     = 0.0

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

// SetDefaults sets the basic defaults for Lbank
func (l *Lbank) SetDefaults() {
	l.Name = "Lbank"
	l.RESTPollingDelay = 10
	l.RequestCurrencyPairFormat.Delimiter = "_"
	l.ConfigCurrencyPairFormat.Delimiter = "_"
	l.AssetTypes = []string{ticker.Spot}
	l.SupportsAutoPairUpdating = true
	l.APIWithdrawPermissions = exchange.AutoWithdrawCryptoWithAPIPermission | exchange.NoFiatWithdrawals
	l.Requester = request.New(l.Name,
		request.NewRateLimit(time.Second, lbankAuthRateLimit),
		request.NewRateLimit(time.Second, lbankUnAuthRateLimit),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	l.APIUrlDefault = lbankAPIURL
	l.APIUrl = l.APIUrlDefault
	l.Websocket = wshandler.New()
}

// Setup takes in the supplied exchange configuration details and sets params
func (l *Lbank) Setup(exch *config.ExchangeConfig) {
	if !exch.Enabled {
		l.SetEnabled(false)
	} else {
		l.Enabled = true
		l.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		l.AuthenticatedWebsocketAPISupport = exch.AuthenticatedWebsocketAPISupport
		l.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		l.SetHTTPClientTimeout(exch.HTTPTimeout)
		l.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		l.RESTPollingDelay = exch.RESTPollingDelay
		l.Verbose = exch.Verbose
		l.Websocket.SetWsStatusAndConnection(exch.Websocket)
		l.BaseCurrencies = exch.BaseCurrencies
		l.AvailablePairs = exch.AvailablePairs
		l.EnabledPairs = exch.EnabledPairs
		err := l.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = l.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = l.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = l.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = l.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
		if l.AuthenticatedAPISupport {
			err = l.loadPrivKey()
			if err != nil {
				l.AuthenticatedAPISupport = false
				log.Errorf("couldnt load private key, setting authenticated support to false")
			}
		}
	}
}

// GetTicker returns a ticker for the specified symbol
// symbol: eth_btc
func (l *Lbank) GetTicker(symbol string) (TickerResponse, error) {
	var t TickerResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := fmt.Sprintf("%s/v%s/%s?%s", l.APIUrl, lbankAPIVersion, lbankTicker, params.Encode())
	return t, l.SendHTTPRequest(path, &t)
}

// GetCurrencyPairs returns a list of supported currency pairs by the exchange
func (l *Lbank) GetCurrencyPairs() ([]string, error) {
	path := fmt.Sprintf("%s/v%s/%s", l.APIUrl, lbankAPIVersion,
		lbankCurrencyPairs)
	var result []string
	return result, l.SendHTTPRequest(path, &result)
}

// GetMarketDepths returns arrays of asks, bids and timestamp
func (l *Lbank) GetMarketDepths(symbol, size, merge string) (MarketDepthResponse, error) {
	var m MarketDepthResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("size", size)
	params.Set("merge", merge)
	path := fmt.Sprintf("%s/v%s/%s?%s", l.APIUrl, lbankAPIVersion, lbankMarketDepths, params.Encode())
	return m, l.SendHTTPRequest(path, &m)
}

// GetTrades returns an array of available trades regarding a particular exchange
func (l *Lbank) GetTrades(symbol, size, time string) ([]TradeResponse, error) {
	var g []TradeResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("size", size)
	params.Set("time", time)
	path := fmt.Sprintf("%s/v%s/%s?%s", l.APIUrl, lbankAPIVersion, lbankTrades, params.Encode())
	return g, l.SendHTTPRequest(path, &g)
}

// GetKlines returns kline data
func (l *Lbank) GetKlines(symbol, size, klineType, time string) ([]KlineResponse, error) {
	var klineTemp interface{}
	var k []KlineResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("size", size)
	params.Set("type", klineType)
	params.Set("time", time)
	path := fmt.Sprintf("%s/v%s/%s?%s", l.APIUrl, lbankAPIVersion, lbankKlines, params.Encode())
	err := l.SendHTTPRequest(path, &klineTemp)
	if err != nil {
		return k, err
	}

	resp, ok := klineTemp.([]interface{})
	if !ok {
		return nil, errors.New("response received is invalid")
	}

	for i := range resp {
		resp2, ok := resp[i].([]interface{})
		if !ok {
			return nil, errors.New("response received is invalid")
		}
		var tempResp KlineResponse
		for x := range resp2 {
			switch x {
			case 0:
				tempResp.TimeStamp = int64(resp2[x].(float64))
			case 1:
				if val, ok := resp2[x].(int64); ok {
					tempResp.OpenPrice = float64(val)
				} else {
					tempResp.OpenPrice = resp2[x].(float64)
				}
			case 2:
				if val, ok := resp2[x].(int64); ok {
					tempResp.HigestPrice = float64(val)
				} else {
					tempResp.HigestPrice = resp2[x].(float64)
				}
			case 3:
				if val, ok := resp2[x].(int64); ok {
					tempResp.LowestPrice = float64(val)
				} else {
					tempResp.LowestPrice = resp2[x].(float64)
				}

			case 4:
				if val, ok := resp2[x].(int64); ok {
					tempResp.ClosePrice = float64(val)
				} else {
					tempResp.ClosePrice = resp2[x].(float64)
				}

			case 5:
				if val, ok := resp2[x].(int64); ok {
					tempResp.TradingVolume = float64(val)
				} else {
					tempResp.TradingVolume = resp2[x].(float64)
				}

			}
		}
		k = append(k, tempResp)
	}
	return k, nil
}

// GetUserInfo gets users account info
func (l *Lbank) GetUserInfo() (InfoFinalResponse, error) {
	var resp InfoFinalResponse
	path := fmt.Sprintf("%s/v%s/%s?", l.APIUrl, lbankAPIVersion, lbankUserInfo)
	err := l.SendAuthHTTPRequest(http.MethodPost, path, nil, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// CreateOrder creates an order
func (l *Lbank) CreateOrder(pair, side string, amount, price float64) (CreateOrderResponse, error) {
	var resp CreateOrderResponse
	if !strings.EqualFold(side, "buy") && !strings.EqualFold(side, "sell") {
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
	params.Set("type", common.StringToLower(side))
	params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	path := fmt.Sprintf("%s/v%s/%s?", l.APIUrl, lbankAPIVersion, lbankPlaceOrder)
	err := l.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// RemoveOrder cancels a given order
func (l *Lbank) RemoveOrder(pair, orderID string) (RemoveOrderResponse, error) {
	var resp RemoveOrderResponse
	params := url.Values{}
	params.Set("symbol", pair)
	params.Set("order_id", orderID)
	path := fmt.Sprintf("%s/v%s/%s", l.APIUrl, lbankAPIVersion, lbankCancelOrder)
	err := l.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
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
func (l *Lbank) QueryOrder(pair, orderIDs string) (QueryOrderFinalResponse, error) {
	var resp QueryOrderFinalResponse
	var tempResp QueryOrderResponse
	params := url.Values{}
	params.Set("symbol", pair)
	params.Set("order_id", orderIDs)
	path := fmt.Sprintf("%s/v%s/%s?", l.APIUrl, lbankAPIVersion, lbankQueryOrder)
	err := l.SendAuthHTTPRequest(http.MethodPost, path, params, &tempResp)
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
func (l *Lbank) QueryOrderHistory(pair, pageNumber, pageLength string) (OrderHistoryFinalResponse, error) {
	var resp OrderHistoryFinalResponse
	var tempResp OrderHistoryResponse
	params := url.Values{}
	params.Set("symbol", pair)
	params.Set("current_page", pageNumber)
	params.Set("page_length", pageLength)
	path := fmt.Sprintf("%s/v%s/%s?", l.APIUrl, lbankAPIVersion, lbankQueryHistoryOrder)
	err := l.SendAuthHTTPRequest(http.MethodPost, path, params, &tempResp)
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
func (l *Lbank) GetPairInfo() ([]PairInfoResponse, error) {
	var resp []PairInfoResponse
	path := fmt.Sprintf("%s/v%s/%s?", l.APIUrl, lbankAPIVersion, lbankPairInfo)
	return resp, l.SendHTTPRequest(path, &resp)
}

// OrderTransactionDetails gets info about transactions
func (l *Lbank) OrderTransactionDetails(symbol, orderID string) (TransactionHistoryResp, error) {
	var resp TransactionHistoryResp
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("order_id", orderID)
	path := fmt.Sprintf("%s/v%s/%s?", l.APIUrl, lbankAPIVersion, lbankOrderTransactionDetails)
	err := l.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// TransactionHistory stores info about transactions
func (l *Lbank) TransactionHistory(symbol, transactionType, startDate, endDate, from, direct, size string) (TransactionHistoryResp, error) {
	var resp TransactionHistoryResp
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("type", transactionType)
	params.Set("start_date", startDate)
	params.Set("end_date", endDate)
	params.Set("from", from)
	params.Set("direct", direct)
	params.Set("size", size)
	path := fmt.Sprintf("%s/v%s/%s?", l.APIUrl, lbankAPIVersion, lbankPastTransactions)
	err := l.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
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
func (l *Lbank) GetOpenOrders(pair, pageNumber, pageLength string) (OpenOrderFinalResponse, error) {
	var resp OpenOrderFinalResponse
	var tempResp OpenOrderResponse
	params := url.Values{}
	params.Set("symbol", pair)
	params.Set("current_page", pageNumber)
	params.Set("page_length", pageLength)
	path := fmt.Sprintf("%s/v%s/%s", l.APIUrl, lbankAPIVersion, lbankOpeningOrders)
	err := l.SendAuthHTTPRequest(http.MethodPost, path, params, &tempResp)
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
func (l *Lbank) USD2RMBRate() (ExchangeRateResponse, error) {
	var resp ExchangeRateResponse
	path := fmt.Sprintf("%s/v%s/%s", l.APIUrl, lbankAPIVersion, lbankUSD2CNYRate)
	return resp, l.SendHTTPRequest(path, &resp)
}

// GetWithdrawConfig gets information about withdrawals
func (l *Lbank) GetWithdrawConfig(assetCode string) ([]WithdrawConfigResponse, error) {
	var resp []WithdrawConfigResponse
	params := url.Values{}
	params.Set("assetCode", assetCode)
	path := fmt.Sprintf("%s/v%s/%s?%s", l.APIUrl, lbankAPIVersion, lbankWithdrawConfig, params.Encode())
	return resp, l.SendHTTPRequest(path, &resp)
}

// Withdraw sends a withdrawal request
func (l *Lbank) Withdraw(account, assetCode, amount, memo, mark string) (WithdrawResponse, error) {
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
	path := fmt.Sprintf("%s/v%s/%s", l.APIUrl, lbankAPIVersion, lbankWithdraw)
	err := l.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// RevokeWithdraw cancels the withdrawal given the withdrawalID
func (l *Lbank) RevokeWithdraw(withdrawID string) (RevokeWithdrawResponse, error) {
	var resp RevokeWithdrawResponse
	params := url.Values{}
	if withdrawID != "" {
		params.Set("withdrawId", withdrawID)
	}
	path := fmt.Sprintf("%s/v%s/%s?", l.APIUrl, lbankAPIVersion, lbankRevokeWithdraw)
	err := l.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != 0 {
		return resp, ErrorCapture(resp.Error)
	}

	return resp, nil
}

// GetWithdrawalRecords gets withdrawal records
func (l *Lbank) GetWithdrawalRecords(assetCode, status, pageNo, pageSize string) (WithdrawalResponse, error) {
	var resp WithdrawalResponse
	params := url.Values{}
	params.Set("assetCode", assetCode)
	params.Set("status", status)
	params.Set("pageNo", pageNo)
	params.Set("pageSize", pageSize)
	path := fmt.Sprintf("%s/v%s/%s", l.APIUrl, lbankAPIVersion, lbankWithdrawalRecords)
	err := l.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
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
func (l *Lbank) SendHTTPRequest(path string, result interface{}) error {
	return l.SendPayload(http.MethodGet,
		path,
		nil,
		nil,
		&result,
		false,
		false,
		l.Verbose,
		l.HTTPDebugging,
		l.HTTPRecording)
}

func (l *Lbank) loadPrivKey() error {
	key := strings.Join([]string{
		"-----BEGIN RSA PRIVATE KEY-----",
		l.APISecret,
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
	md5hash := common.GetMD5([]byte(data))
	m := common.StringToUpper(common.HexEncodeToString(md5hash))
	s := common.GetSHA256([]byte(m))
	r, err := rsa.SignPKCS1v15(rand.Reader, l.privateKey, crypto.SHA256, s)
	if err != nil {
		return "", err
	}
	return common.Base64Encode(r), nil
}

// SendAuthHTTPRequest sends an authenticated request
func (l *Lbank) SendAuthHTTPRequest(method, endpoint string, vals url.Values, result interface{}) error {
	if vals == nil {
		vals = url.Values{}
	}

	vals.Set("api_key", l.APIKey)
	sig, err := l.sign(vals.Encode())
	if err != nil {
		return err
	}

	vals.Set("sign", sig)
	payload := vals.Encode()
	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	return l.SendPayload(method,
		endpoint,
		headers,
		bytes.NewBufferString(payload),
		&result,
		true,
		false,
		l.Verbose,
		l.HTTPDebugging,
		l.HTTPRecording)
}
