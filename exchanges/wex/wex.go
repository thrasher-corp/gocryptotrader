package wex

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
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	wexAPIPublicURL       = "https://wex.nz/api"
	wexAPIPrivateURL      = "https://wex.nz/tapi"
	wexAPIPublicVersion   = "3"
	wexAPIPrivateVersion  = "1"
	wexInfo               = "info"
	wexTicker             = "ticker"
	wexDepth              = "depth"
	wexTrades             = "trades"
	wexAccountInfo        = "getInfo"
	wexTrade              = "Trade"
	wexActiveOrders       = "ActiveOrders"
	wexOrderInfo          = "OrderInfo"
	wexCancelOrder        = "CancelOrder"
	wexTradeHistory       = "TradeHistory"
	wexTransactionHistory = "TransHistory"
	wexWithdrawCoin       = "WithdrawCoin"
	wexCoinDepositAddress = "CoinDepositAddress"
	wexCreateCoupon       = "CreateCoupon"
	wexRedeemCoupon       = "RedeemCoupon"
)

// WEX is the overarching type across the wex package
type WEX struct {
	exchange.Base
	Ticker map[string]Ticker
}

// SetDefaults sets current default value for WEX
func (w *WEX) SetDefaults() {
	w.Name = "WEX"
	w.Enabled = false
	w.Fee = 0.2
	w.Verbose = false
	w.Websocket = false
	w.RESTPollingDelay = 10
	w.Ticker = make(map[string]Ticker)
	w.RequestCurrencyPairFormat.Delimiter = "_"
	w.RequestCurrencyPairFormat.Uppercase = false
	w.RequestCurrencyPairFormat.Separator = "-"
	w.ConfigCurrencyPairFormat.Delimiter = ""
	w.ConfigCurrencyPairFormat.Uppercase = true
	w.AssetTypes = []string{ticker.Spot}
}

// Setup sets exchange configuration parameters for WEX
func (w *WEX) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		w.SetEnabled(false)
	} else {
		w.Enabled = true
		w.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		w.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		w.RESTPollingDelay = exch.RESTPollingDelay
		w.Verbose = exch.Verbose
		w.Websocket = exch.Websocket
		w.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		w.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		w.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := w.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = w.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns the exchange fee
func (w *WEX) GetFee() float64 {
	return w.Fee
}

// GetInfo returns the WEX info
func (w *WEX) GetInfo() (Info, error) {
	resp := Info{}
	req := fmt.Sprintf("%s/%s/%s/", wexAPIPublicURL, wexAPIPublicVersion, wexInfo)

	return resp, common.SendHTTPGetRequest(req, true, w.Verbose, &resp)
}

// GetTicker returns a ticker for a specific currency
func (w *WEX) GetTicker(symbol string) (map[string]Ticker, error) {
	type Response struct {
		Data map[string]Ticker
	}

	response := Response{}
	req := fmt.Sprintf("%s/%s/%s/%s", wexAPIPublicURL, wexAPIPublicVersion, wexTicker, symbol)

	return response.Data, common.SendHTTPGetRequest(req, true, w.Verbose, &response.Data)
}

// GetDepth returns the depth for a specific currency
func (w *WEX) GetDepth(symbol string) (Orderbook, error) {
	type Response struct {
		Data map[string]Orderbook
	}

	response := Response{}
	req := fmt.Sprintf("%s/%s/%s/%s", wexAPIPublicURL, wexAPIPublicVersion, wexDepth, symbol)

	return response.Data[symbol],
		common.SendHTTPGetRequest(req, true, w.Verbose, &response.Data)
}

// GetTrades returns the trades for a specific currency
func (w *WEX) GetTrades(symbol string) ([]Trades, error) {
	type Response struct {
		Data map[string][]Trades
	}

	response := Response{}
	req := fmt.Sprintf("%s/%s/%s/%s", wexAPIPublicURL, wexAPIPublicVersion, wexTrades, symbol)

	return response.Data[symbol],
		common.SendHTTPGetRequest(req, true, w.Verbose, &response.Data)
}

// GetAccountInfo returns a users account info
func (w *WEX) GetAccountInfo() (AccountInfo, error) {
	var result AccountInfo

	return result,
		w.SendAuthenticatedHTTPRequest(wexAccountInfo, url.Values{}, &result)
}

// GetActiveOrders returns the active orders for a specific currency
func (w *WEX) GetActiveOrders(pair string) (map[string]ActiveOrders, error) {
	req := url.Values{}
	req.Add("pair", pair)

	var result map[string]ActiveOrders

	return result, w.SendAuthenticatedHTTPRequest(wexActiveOrders, req, &result)
}

// GetOrderInfo returns the order info for a specific order ID
func (w *WEX) GetOrderInfo(OrderID int64) (map[string]OrderInfo, error) {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(OrderID, 10))

	var result map[string]OrderInfo

	return result, w.SendAuthenticatedHTTPRequest(wexOrderInfo, req, &result)
}

// CancelOrder cancels an order for a specific order ID
func (w *WEX) CancelOrder(OrderID int64) (bool, error) {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(OrderID, 10))

	var result CancelOrder
	err := w.SendAuthenticatedHTTPRequest(wexCancelOrder, req, &result)

	if err != nil {
		return false, err
	}

	return true, nil
}

// Trade places an order and returns the order ID if successful or an error
func (w *WEX) Trade(pair, orderType string, amount, price float64) (int64, error) {
	req := url.Values{}
	req.Add("pair", pair)
	req.Add("type", orderType)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("rate", strconv.FormatFloat(price, 'f', -1, 64))

	var result Trade

	return int64(result.OrderID),
		w.SendAuthenticatedHTTPRequest(wexTrade, req, &result)
}

// GetTransactionHistory returns the transaction history
func (w *WEX) GetTransactionHistory(TIDFrom, Count, TIDEnd int64, order, since, end string) (map[string]TransHistory, error) {
	req := url.Values{}
	req.Add("from", strconv.FormatInt(TIDFrom, 10))
	req.Add("count", strconv.FormatInt(Count, 10))
	req.Add("from_id", strconv.FormatInt(TIDFrom, 10))
	req.Add("end_id", strconv.FormatInt(TIDEnd, 10))
	req.Add("order", order)
	req.Add("since", since)
	req.Add("end", end)

	var result map[string]TransHistory

	return result,
		w.SendAuthenticatedHTTPRequest(wexTransactionHistory, req, &result)
}

// GetTradeHistory returns the trade history
func (w *WEX) GetTradeHistory(TIDFrom, Count, TIDEnd int64, order, since, end, pair string) (map[string]TradeHistory, error) {
	req := url.Values{}
	req.Add("from", strconv.FormatInt(TIDFrom, 10))
	req.Add("count", strconv.FormatInt(Count, 10))
	req.Add("from_id", strconv.FormatInt(TIDFrom, 10))
	req.Add("end_id", strconv.FormatInt(TIDEnd, 10))
	req.Add("order", order)
	req.Add("since", since)
	req.Add("end", end)
	req.Add("pair", pair)

	var result map[string]TradeHistory

	return result, w.SendAuthenticatedHTTPRequest(wexTradeHistory, req, &result)
}

// WithdrawCoins withdraws coins for a specific coin
func (w *WEX) WithdrawCoins(coin string, amount float64, address string) (WithdrawCoins, error) {
	req := url.Values{}
	req.Add("coinName", coin)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)

	var result WithdrawCoins

	return result, w.SendAuthenticatedHTTPRequest(wexWithdrawCoin, req, &result)
}

// CoinDepositAddress returns the deposit address for a specific currency
func (w *WEX) CoinDepositAddress(coin string) (string, error) {
	req := url.Values{}
	req.Add("coinName", coin)

	var result CoinDepositAddress

	return result.Address,
		w.SendAuthenticatedHTTPRequest(wexCoinDepositAddress, req, &result)
}

// CreateCoupon creates an exchange coupon for a sepcific currency
func (w *WEX) CreateCoupon(currency string, amount float64) (CreateCoupon, error) {
	req := url.Values{}
	req.Add("currency", currency)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	var result CreateCoupon

	return result, w.SendAuthenticatedHTTPRequest(wexCreateCoupon, req, &result)
}

// RedeemCoupon redeems an exchange coupon
func (w *WEX) RedeemCoupon(coupon string) (RedeemCoupon, error) {
	req := url.Values{}
	req.Add("coupon", coupon)

	var result RedeemCoupon

	return result, w.SendAuthenticatedHTTPRequest(wexRedeemCoupon, req, &result)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to WEX
func (w *WEX) SendAuthenticatedHTTPRequest(method string, values url.Values, result interface{}) (err error) {
	if !w.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, w.Name)
	}

	if w.Nonce.Get() == 0 {
		w.Nonce.Set(time.Now().Unix())
	} else {
		w.Nonce.Inc()
	}
	values.Set("nonce", w.Nonce.String())
	values.Set("method", method)

	encoded := values.Encode()
	hmac := common.GetHMAC(common.HashSHA512, []byte(encoded), []byte(w.APISecret))

	if w.Verbose {
		log.Printf("Sending POST request to %s calling method %s with params %s\n", wexAPIPrivateURL, method, encoded)
	}

	headers := make(map[string]string)
	headers["Key"] = w.APIKey
	headers["Sign"] = common.HexEncodeToString(hmac)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := common.SendHTTPRequest("POST", wexAPIPrivateURL, headers, strings.NewReader(encoded))
	if err != nil {
		return err
	}

	response := Response{}
	err = common.JSONDecode([]byte(resp), &response)
	if err != nil {
		return err
	}

	if response.Success != 1 {
		return errors.New(response.Error)
	}

	JSONEncoded, err := common.JSONEncode(response.Return)
	if err != nil {
		return err
	}

	return common.JSONDecode(JSONEncoded, &result)
}
