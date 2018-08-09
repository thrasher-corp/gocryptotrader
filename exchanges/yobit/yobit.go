package yobit

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	apiPublicURL                  = "https://yobit.net/api"
	apiPrivateURL                 = "https://yobit.net/tapi"
	apiPublicVersion              = "3"
	publicInfo                    = "info"
	publicTicker                  = "ticker"
	publicDepth                   = "depth"
	publicTrades                  = "trades"
	privateAccountInfo            = "getInfo"
	privateTrade                  = "Trade"
	privateActiveOrders           = "ActiveOrders"
	privateOrderInfo              = "OrderInfo"
	privateCancelOrder            = "CancelOrder"
	privateTradeHistory           = "TradeHistory"
	privateGetDepositAddress      = "GetDepositAddress"
	privateWithdrawCoinsToAddress = "WithdrawCoinsToAddress"
	privateCreateCoupon           = "CreateYobicode"
	privateRedeemCoupon           = "RedeemYobicode"

	yobitAuthRate   = 0
	yobitUnauthRate = 0
)

// Yobit is the overarching type across the Yobit package
type Yobit struct {
	exchange.Base
	Ticker map[string]Ticker
}

// SetDefaults sets current default value for Yobit
func (y *Yobit) SetDefaults() {
	y.Name = "Yobit"
	y.Enabled = true
	y.Fee = 0.2
	y.Verbose = false
	y.Websocket = false
	y.RESTPollingDelay = 10
	y.APIUrl = apiPublicURL
	y.AuthenticatedAPISupport = true
	y.Ticker = make(map[string]Ticker)
	y.RequestCurrencyPairFormat.Delimiter = "_"
	y.RequestCurrencyPairFormat.Uppercase = false
	y.RequestCurrencyPairFormat.Separator = "-"
	y.ConfigCurrencyPairFormat.Delimiter = "_"
	y.ConfigCurrencyPairFormat.Uppercase = true
	y.AssetTypes = []string{ticker.Spot}
	y.SupportsAutoPairUpdating = false
	y.SupportsRESTTickerBatching = true
	y.Requester = request.New(y.Name, request.NewRateLimit(time.Second, yobitAuthRate), request.NewRateLimit(time.Second, yobitUnauthRate), common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
}

// Setup sets exchange configuration parameters for Yobit
func (y *Yobit) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		y.SetEnabled(false)
	} else {
		y.Enabled = true
		y.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		y.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		y.RESTPollingDelay = exch.RESTPollingDelay
		y.Verbose = exch.Verbose
		y.Websocket = exch.Websocket
		y.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		y.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		y.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		y.SetHTTPClientTimeout(exch.HTTPTimeout)
		y.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		err := y.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = y.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = y.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns the exchange fee
func (y *Yobit) GetFee() float64 {
	return y.Fee
}

// GetInfo returns the Yobit info
func (y *Yobit) GetInfo() (Info, error) {
	resp := Info{}
	path := fmt.Sprintf("%s/%s/%s/", apiPublicURL, apiPublicVersion, publicInfo)

	return resp, y.SendHTTPRequest(path, &resp)
}

// GetTicker returns a ticker for a specific currency
func (y *Yobit) GetTicker(symbol string) (map[string]Ticker, error) {
	type Response struct {
		Data map[string]Ticker
	}

	response := Response{}
	path := fmt.Sprintf("%s/%s/%s/%s", apiPublicURL, apiPublicVersion, publicTicker, symbol)

	return response.Data, y.SendHTTPRequest(path, &response.Data)
}

// GetDepth returns the depth for a specific currency
func (y *Yobit) GetDepth(symbol string) (Orderbook, error) {
	type Response struct {
		Data map[string]Orderbook
	}

	response := Response{}
	path := fmt.Sprintf("%s/%s/%s/%s", apiPublicURL, apiPublicVersion, publicDepth, symbol)

	return response.Data[symbol],
		y.SendHTTPRequest(path, &response.Data)
}

// GetTrades returns the trades for a specific currency
func (y *Yobit) GetTrades(symbol string) ([]Trades, error) {
	type Response struct {
		Data map[string][]Trades
	}

	response := Response{}
	path := fmt.Sprintf("%s/%s/%s/%s", apiPublicURL, apiPublicVersion, publicTrades, symbol)

	return response.Data[symbol], y.SendHTTPRequest(path, &response.Data)
}

// GetAccountInfo returns a users account info
func (y *Yobit) GetAccountInfo() (AccountInfo, error) {
	result := AccountInfo{}

	err := y.SendAuthenticatedHTTPRequest(privateAccountInfo, url.Values{}, &result)
	if err != nil {
		return result, err
	}
	if result.Error != "" {
		return result, errors.New(result.Error)
	}
	return result, nil
}

// Trade places an order and returns the order ID if successful or an error
func (y *Yobit) Trade(pair, orderType string, amount, price float64) (int64, error) {
	req := url.Values{}
	req.Add("pair", pair)
	req.Add("type", orderType)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("rate", strconv.FormatFloat(price, 'f', -1, 64))

	result := Trade{}

	err := y.SendAuthenticatedHTTPRequest(privateTrade, req, &result)
	if err != nil {
		return int64(result.OrderID), err
	}
	if result.Error != "" {
		return int64(result.OrderID), errors.New(result.Error)
	}
	return int64(result.OrderID), nil
}

// GetActiveOrders returns the active orders for a specific currency
func (y *Yobit) GetActiveOrders(pair string) (map[string]ActiveOrders, error) {
	req := url.Values{}
	req.Add("pair", pair)

	result := map[string]ActiveOrders{}

	return result, y.SendAuthenticatedHTTPRequest(privateActiveOrders, req, &result)
}

// GetOrderInfo returns the order info for a specific order ID
func (y *Yobit) GetOrderInfo(OrderID int64) (map[string]OrderInfo, error) {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(OrderID, 10))

	result := map[string]OrderInfo{}

	return result, y.SendAuthenticatedHTTPRequest(privateOrderInfo, req, &result)
}

// CancelOrder cancels an order for a specific order ID
func (y *Yobit) CancelOrder(OrderID int64) (bool, error) {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(OrderID, 10))

	result := CancelOrder{}

	err := y.SendAuthenticatedHTTPRequest(privateCancelOrder, req, &result)
	if err != nil {
		return false, err
	}
	if result.Error != "" {
		return false, errors.New(result.Error)
	}
	return true, nil
}

// GetTradeHistory returns the trade history
func (y *Yobit) GetTradeHistory(TIDFrom, Count, TIDEnd int64, order, since, end, pair string) (map[string]TradeHistory, error) {
	req := url.Values{}
	req.Add("from", strconv.FormatInt(TIDFrom, 10))
	req.Add("count", strconv.FormatInt(Count, 10))
	req.Add("from_id", strconv.FormatInt(TIDFrom, 10))
	req.Add("end_id", strconv.FormatInt(TIDEnd, 10))
	req.Add("order", order)
	req.Add("since", since)
	req.Add("end", end)
	req.Add("pair", pair)

	result := map[string]TradeHistory{}

	return result, y.SendAuthenticatedHTTPRequest(privateTradeHistory, req, &result)
}

// GetDepositAddress returns the deposit address for a specific currency
func (y *Yobit) GetDepositAddress(coin string) (DepositAddress, error) {
	req := url.Values{}
	req.Add("coinName", coin)

	result := DepositAddress{}

	err := y.SendAuthenticatedHTTPRequest(privateGetDepositAddress, req, &result)
	if err != nil {
		return result, err
	}
	if result.Error != "" {
		return result, errors.New(result.Error)
	}
	return result, nil
}

// WithdrawCoinsToAddress initiates a withdrawal to a specified address
func (y *Yobit) WithdrawCoinsToAddress(coin string, amount float64, address string) (WithdrawCoinsToAddress, error) {
	req := url.Values{}
	req.Add("coinName", coin)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)

	result := WithdrawCoinsToAddress{}

	err := y.SendAuthenticatedHTTPRequest(privateWithdrawCoinsToAddress, req, &result)
	if err != nil {
		return result, err
	}
	if result.Error != "" {
		return result, errors.New(result.Error)
	}
	return result, nil
}

// CreateCoupon creates an exchange coupon for a sepcific currency
func (y *Yobit) CreateCoupon(currency string, amount float64) (CreateCoupon, error) {
	req := url.Values{}
	req.Add("currency", currency)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	var result CreateCoupon

	err := y.SendAuthenticatedHTTPRequest(privateCreateCoupon, req, &result)
	if err != nil {
		return result, err
	}
	if result.Error != "" {
		return result, errors.New(result.Error)
	}
	return result, nil
}

// RedeemCoupon redeems an exchange coupon
func (y *Yobit) RedeemCoupon(coupon string) (RedeemCoupon, error) {
	req := url.Values{}
	req.Add("coupon", coupon)

	result := RedeemCoupon{}

	err := y.SendAuthenticatedHTTPRequest(privateRedeemCoupon, req, &result)
	if err != nil {
		return result, err
	}
	if result.Error != "" {
		return result, errors.New(result.Error)
	}
	return result, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (y *Yobit) SendHTTPRequest(path string, result interface{}) error {
	return y.SendPayload("GET", path, nil, nil, result, false, y.Verbose)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to Yobit
func (y *Yobit) SendAuthenticatedHTTPRequest(path string, params url.Values, result interface{}) (err error) {
	if !y.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, y.Name)
	}

	if params == nil {
		params = url.Values{}
	}

	if y.Nonce.Get() == 0 {
		y.Nonce.Set(time.Now().Unix())
	} else {
		y.Nonce.Inc()
	}
	params.Set("nonce", y.Nonce.String())
	params.Set("method", path)

	encoded := params.Encode()
	hmac := common.GetHMAC(common.HashSHA512, []byte(encoded), []byte(y.APISecret))

	if y.Verbose {
		log.Printf("Sending POST request to %s calling path %s with params %s\n", apiPrivateURL, path, encoded)
	}

	headers := make(map[string]string)
	headers["Key"] = y.APIKey
	headers["Sign"] = common.HexEncodeToString(hmac)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	return y.SendPayload("POST", apiPrivateURL, headers, strings.NewReader(encoded), result, true, y.Verbose)
}
