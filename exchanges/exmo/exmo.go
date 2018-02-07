package exmo

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	exmoAPIURL     = "https://api.exmo.com"
	exmoAPIVersion = "1"

	exmoTrades          = "trades"
	exmoOrderbook       = "order_book"
	exmoTicker          = "ticker"
	exmoPairSettings    = "pair_settings"
	exmoCurrency        = "currency"
	exmoUserInfo        = "user_info"
	exmoOrderCreate     = "order_create"
	exmoOrderCancel     = "order_cancel"
	exmoOpenOrders      = "user_open_orders"
	exmoUserTrades      = "user_trades"
	exmoCancelledOrders = "user_cancelled_orders"
	exmoOrderTrades     = "order_trades"
	exmoRequiredAmount  = "required_amount"
	exmoDepositAddress  = "deposit_address"
	exmoWithdrawCrypt   = "withdraw_crypt"
	exmoGetWithdrawTXID = "withdraw_get_txid"
	exmoExcodeCreate    = "excode_create"
	exmoExcodeLoad      = "excode_load"
	exmoWalletHistory   = "wallet_history"
)

// EXMO exchange struct
type EXMO struct {
	exchange.Base
}

// Rate limit: 180 per/minute

// SetDefaults sets the basic defaults for exmo
func (e *EXMO) SetDefaults() {
	e.Name = "EXMO"
	e.Enabled = false
	e.Verbose = false
	e.Websocket = false
	e.RESTPollingDelay = 10
	e.RequestCurrencyPairFormat.Delimiter = "_"
	e.RequestCurrencyPairFormat.Uppercase = true
	e.RequestCurrencyPairFormat.Separator = ","
	e.ConfigCurrencyPairFormat.Delimiter = "_"
	e.ConfigCurrencyPairFormat.Uppercase = true
	e.AssetTypes = []string{ticker.Spot}
}

// Setup takes in the supplied exchange configuration details and sets params
func (e *EXMO) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		e.SetEnabled(false)
	} else {
		e.Enabled = true
		e.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		e.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		e.RESTPollingDelay = exch.RESTPollingDelay
		e.Verbose = exch.Verbose
		e.Websocket = exch.Websocket
		e.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		e.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		e.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := e.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = e.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetTrades returns the trades for a symbol or symbols
func (e *EXMO) GetTrades(symbol string) (map[string][]Trades, error) {
	v := url.Values{}
	v.Set("pair", symbol)
	result := make(map[string][]Trades)
	url := fmt.Sprintf("%s/v%s/%s", exmoAPIURL, exmoAPIVersion, exmoTrades)
	err := common.SendHTTPGetRequest(common.EncodeURLValues(url, v), true, e.Verbose, &result)
	return result, err
}

// GetOrderbook returns the orderbook for a symbol or symbols
func (e *EXMO) GetOrderbook(symbol string) (map[string]Orderbook, error) {
	v := url.Values{}
	v.Set("pair", symbol)
	result := make(map[string]Orderbook)
	url := fmt.Sprintf("%s/v%s/%s", exmoAPIURL, exmoAPIVersion, exmoOrderbook)
	err := common.SendHTTPGetRequest(common.EncodeURLValues(url, v), true, e.Verbose, &result)
	return result, err
}

// GetTicker returns the ticker for a symbol or symbols
func (e *EXMO) GetTicker(symbol string) (map[string]Ticker, error) {
	v := url.Values{}
	v.Set("pair", symbol)
	result := make(map[string]Ticker)
	url := fmt.Sprintf("%s/v%s/%s", exmoAPIURL, exmoAPIVersion, exmoTicker)
	err := common.SendHTTPGetRequest(common.EncodeURLValues(url, v), true, e.Verbose, &result)
	return result, err
}

// GetPairSettings returns the pair settings for a symbol or symbols
func (e *EXMO) GetPairSettings() (map[string]PairSettings, error) {
	result := make(map[string]PairSettings)
	url := fmt.Sprintf("%s/v%s/%s", exmoAPIURL, exmoAPIVersion, exmoPairSettings)
	err := common.SendHTTPGetRequest(url, true, e.Verbose, &result)
	return result, err
}

// GetCurrency returns a list of currencies
func (e *EXMO) GetCurrency() ([]string, error) {
	result := []string{}
	url := fmt.Sprintf("%s/v%s/%s", exmoAPIURL, exmoAPIVersion, exmoCurrency)
	err := common.SendHTTPGetRequest(url, true, e.Verbose, &result)
	return result, err
}

// GetUserInfo returns the user info
func (e *EXMO) GetUserInfo() (UserInfo, error) {
	var result UserInfo
	err := e.SendAuthenticatedHTTPRequest("POST", exmoUserInfo, url.Values{}, &result)
	return result, err
}

// CreateOrder creates an order
// Params: pair, quantity, price and type
// Type can be buy, sell, market_buy, market_sell, market_buy_total and market_sell_total
func (e *EXMO) CreateOrder(pair, orderType string, price, amount float64) (int64, error) {
	type response struct {
		OrderID int64 `json:"order_id"`
	}

	v := url.Values{}
	v.Set("pair", pair)
	v.Set("type", orderType)
	v.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	v.Set("quantity", strconv.FormatFloat(amount, 'f', -1, 64))

	var result response
	err := e.SendAuthenticatedHTTPRequest("POST", exmoOrderCreate, v, &result)
	return result.OrderID, err
}

// CancelOrder cancels an order by the orderID
func (e *EXMO) CancelOrder(orderID int64) error {
	v := url.Values{}
	v.Set("order_id", strconv.FormatInt(orderID, 10))
	var result interface{}
	return e.SendAuthenticatedHTTPRequest("POST", exmoOrderCancel, v, &result)
}

// GetOpenOrders returns the users open orders
func (e *EXMO) GetOpenOrders() (map[string]OpenOrders, error) {
	result := make(map[string]OpenOrders)
	err := e.SendAuthenticatedHTTPRequest("POST", exmoOpenOrders, url.Values{}, &result)
	return result, err
}

// GetUserTrades returns the user trades
func (e *EXMO) GetUserTrades(pair, offset, limit string) (map[string][]UserTrades, error) {
	result := make(map[string][]UserTrades)
	v := url.Values{}
	v.Set("pair", pair)

	if offset != "" {
		v.Set("offset", offset)
	}

	if limit != "" {
		v.Set("limit", limit)
	}

	err := e.SendAuthenticatedHTTPRequest("POST", exmoUserTrades, v, &result)
	return result, err
}

// GetCancelledOrders returns a list of cancelled orders
func (e *EXMO) GetCancelledOrders(offset, limit string) ([]CancelledOrder, error) {
	var result []CancelledOrder
	v := url.Values{}

	if offset != "" {
		v.Set("offset", offset)
	}

	if limit != "" {
		v.Set("limit", limit)
	}

	err := e.SendAuthenticatedHTTPRequest("POST", exmoCancelledOrders, v, &result)
	return result, err
}

// GetOrderTrades returns a history of order trade details for the specific orderID
func (e *EXMO) GetOrderTrades(orderID int64) (OrderTrades, error) {
	var result OrderTrades
	v := url.Values{}
	v.Set("order_id", strconv.FormatInt(orderID, 10))

	err := e.SendAuthenticatedHTTPRequest("POST", exmoOrderTrades, v, &result)
	return result, err
}

// GetRequiredAmount calculates the sum of buying a certain amount of currency
// for the particular currency pair
func (e *EXMO) GetRequiredAmount(pair string, amount float64) (RequiredAmount, error) {
	v := url.Values{}
	v.Set("pair", pair)
	v.Set("quantity", strconv.FormatFloat(amount, 'f', -1, 64))
	var result RequiredAmount
	err := e.SendAuthenticatedHTTPRequest("POST", exmoRequiredAmount, v, &result)
	return result, err
}

// GetDepositAddress returns a list of addresses for cryptocurrency deposits
func (e *EXMO) GetDepositAddress() (map[string]string, error) {
	result := make(map[string]string)
	err := e.SendAuthenticatedHTTPRequest("POST", exmoDepositAddress, url.Values{}, &result)
	log.Println(reflect.TypeOf(result).String())
	return result, err
}

// WithdrawCryptocurrency withdraws a cryptocurrency from the exchange to the desired address
// NOTE: This API function is available only after request to their tech support team
func (e *EXMO) WithdrawCryptocurrency(currency, address, invoice string, amount float64) (int64, error) {
	type response struct {
		TaskID int64 `json:"task_id,string"`
	}

	v := url.Values{}
	v.Set("currency", currency)
	v.Set("address", address)

	if common.StringToUpper(currency) == "XRP" {
		v.Set(invoice, invoice)
	}

	v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var result response
	err := e.SendAuthenticatedHTTPRequest("POST", exmoWithdrawCrypt, v, &result)
	return result.TaskID, err
}

// GetWithdrawTXID gets the result of a withdrawal request
func (e *EXMO) GetWithdrawTXID(taskID int64) (string, error) {
	type response struct {
		Status bool   `json:"status"`
		TXID   string `json:"txid"`
	}

	v := url.Values{}
	v.Set("task_id", strconv.FormatInt(taskID, 10))

	var result response
	err := e.SendAuthenticatedHTTPRequest("POST", exmoGetWithdrawTXID, v, &result)
	return result.TXID, err
}

// ExcodeCreate creates an EXMO coupon
func (e *EXMO) ExcodeCreate(currency string, amount float64) (ExcodeCreate, error) {
	v := url.Values{}
	v.Set("currency", currency)
	v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	var result ExcodeCreate
	err := e.SendAuthenticatedHTTPRequest("POST", exmoExcodeCreate, v, &result)
	return result, err
}

// ExcodeLoad loads an EXMO coupon
func (e *EXMO) ExcodeLoad(excode string) (ExcodeLoad, error) {
	v := url.Values{}
	v.Set("code", excode)

	var result ExcodeLoad
	err := e.SendAuthenticatedHTTPRequest("POST", exmoExcodeLoad, v, &result)
	return result, err
}

// GetWalletHistory returns the users deposit/withdrawal history
func (e *EXMO) GetWalletHistory(date int64) (WalletHistory, error) {
	v := url.Values{}
	v.Set("date", strconv.FormatInt(date, 10))

	var result WalletHistory
	err := e.SendAuthenticatedHTTPRequest("POST", exmoWalletHistory, v, &result)
	return result, err
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (e *EXMO) SendAuthenticatedHTTPRequest(method, endpoint string, vals url.Values, result interface{}) error {
	if !e.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, e.Name)
	}

	if e.Nonce.Get() == 0 {
		e.Nonce.Set(time.Now().UnixNano())
	} else {
		e.Nonce.Inc()
	}
	vals.Set("nonce", e.Nonce.String())

	payload := vals.Encode()
	hash := common.GetHMAC(common.HashSHA512, []byte(payload), []byte(e.APISecret))

	if e.Verbose {
		log.Printf("Sending %s request to %s with params %s\n", method, endpoint, payload)
	}

	headers := make(map[string]string)
	headers["Key"] = e.APIKey
	headers["Sign"] = common.HexEncodeToString(hash)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	path := fmt.Sprintf("%s/v%s/%s", exmoAPIURL, exmoAPIVersion, endpoint)
	resp, err := common.SendHTTPRequest(method, path, headers, strings.NewReader(payload))
	if err != nil {
		return err
	}

	if e.Verbose {
		log.Printf("Received raw: \n%s\n", resp)
	}

	var authResp AuthResponse
	err = common.JSONDecode([]byte(resp), &authResp)
	if err != nil {
		return errors.New("unable to JSON Unmarshal auth response")
	}

	if !authResp.Result && authResp.Error != "" {
		return fmt.Errorf("auth error: %s", authResp.Error)
	}

	err = common.JSONDecode([]byte(resp), &result)
	if err != nil {
		return errors.New("unable to JSON Unmarshal response")
	}
	return nil
}
