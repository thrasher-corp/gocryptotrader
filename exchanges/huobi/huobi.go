package huobi

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	huobiAPIURL     = "https://api.huobi.pro"
	huobiAPIVersion = "1"

	huobiMarketHistoryKline   = "market/history/kline"
	huobiMarketDetail         = "market/detail"
	huobiMarketDetailMerged   = "market/detail/merged"
	huobiMarketDepth          = "market/depth"
	huobiMarketTrade          = "market/trade"
	huobiMarketTradeHistory   = "market/history/trade"
	huobiSymbols              = "common/symbols"
	huobiCurrencies           = "common/currencys"
	huobiTimestamp            = "common/timestamp"
	huobiAccounts             = "account/accounts"
	huobiAccountBalance       = "account/accounts/%s/balance"
	huobiOrderPlace           = "order/orders/place"
	huobiOrderCancel          = "order/orders/%s/submitcancel"
	huobiOrderCancelBatch     = "order/orders/batchcancel"
	huobiGetOrder             = "order/orders/%s"
	huobiGetOrderMatch        = "order/orders/%s/matchresults"
	huobiGetOrders            = "order/orders"
	huobiGetOrdersMatch       = "orders/matchresults"
	huobiMarginTransferIn     = "dw/transfer-in/margin"
	huobiMarginTransferOut    = "dw/transfer-out/margin"
	huobiMarginOrders         = "margin/orders"
	huobiMarginRepay          = "margin/orders/%s/repay"
	huobiMarginLoanOrders     = "margin/loan-orders"
	huobiMarginAccountBalance = "margin/accounts/balance"
	huobiWithdrawCreate       = "dw/withdraw/api/create"
	huobiWithdrawCancel       = "dw/withdraw-virtual/%s/cancel"
)

// HUOBI is the overarching type across this package
type HUOBI struct {
	exchange.Base
}

// SetDefaults sets default values for the exchange
func (h *HUOBI) SetDefaults() {
	h.Name = "Huobi"
	h.Enabled = false
	h.Fee = 0
	h.Verbose = false
	h.Websocket = false
	h.RESTPollingDelay = 10
	h.RequestCurrencyPairFormat.Delimiter = ""
	h.RequestCurrencyPairFormat.Uppercase = false
	h.ConfigCurrencyPairFormat.Delimiter = "-"
	h.ConfigCurrencyPairFormat.Uppercase = true
	h.AssetTypes = []string{ticker.Spot}
}

// Setup sets user configuration
func (h *HUOBI) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		h.SetEnabled(false)
	} else {
		h.Enabled = true
		h.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		h.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		h.RESTPollingDelay = exch.RESTPollingDelay
		h.Verbose = exch.Verbose
		h.Websocket = exch.Websocket
		h.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		h.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		h.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := h.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = h.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns Huobi fee
func (h *HUOBI) GetFee() float64 {
	return h.Fee
}

// GetKline returns kline data
func (h *HUOBI) GetKline(symbol, period, size string) ([]Klines, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)

	if period != "" {
		vals.Set("period", period)
	}

	if size != "" {
		vals.Set("size", size)
	}

	type response struct {
		Response
		Data []Klines `json:"data"`
	}

	var result response
	url := fmt.Sprintf("%s/%s", huobiAPIURL, huobiMarketHistoryKline)
	err := common.SendHTTPGetRequest(common.EncodeURLValues(url, vals), true, h.Verbose, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Data, err
}

// GetMarketDetailMerged returns the ticker for the specified symbol
func (h *HUOBI) GetMarketDetailMerged(symbol string) (DetailMerged, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)

	type response struct {
		Response
		Tick DetailMerged `json:"tick"`
	}

	var result response
	url := fmt.Sprintf("%s/%s", huobiAPIURL, huobiMarketDetailMerged)
	err := common.SendHTTPGetRequest(common.EncodeURLValues(url, vals), true, h.Verbose, &result)

	if result.ErrorMessage != "" {
		return result.Tick, errors.New(result.ErrorMessage)
	}
	return result.Tick, err
}

// GetDepth returns the depth for the specified symbol
func (h *HUOBI) GetDepth(symbol, depthType string) (Orderbook, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)

	if depthType != "" {
		vals.Set("type", depthType)
	}

	type response struct {
		Response
		Depth Orderbook `json:"tick"`
	}

	var result response
	url := fmt.Sprintf("%s/%s", huobiAPIURL, huobiMarketDepth)
	err := common.SendHTTPGetRequest(common.EncodeURLValues(url, vals), true, h.Verbose, &result)

	if result.ErrorMessage != "" {
		return result.Depth, errors.New(result.ErrorMessage)
	}
	return result.Depth, err
}

// GetTrades returns the trades for the specified symbol
func (h *HUOBI) GetTrades(symbol string) ([]Trade, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)

	type response struct {
		Response
		Tick struct {
			Data []Trade `json:"data"`
		} `json:"tick"`
	}

	var result response
	url := fmt.Sprintf("%s/%s", huobiAPIURL, huobiMarketTrade)
	err := common.SendHTTPGetRequest(common.EncodeURLValues(url, vals), true, h.Verbose, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Tick.Data, err
}

// GetTradeHistory returns the trades for the specified symbol
func (h *HUOBI) GetTradeHistory(symbol, size string) ([]TradeHistory, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)

	if size != "" {
		vals.Set("size", size)
	}

	type response struct {
		Response
		TradeHistory []TradeHistory `json:"data"`
	}

	var result response
	url := fmt.Sprintf("%s/%s", huobiAPIURL, huobiMarketTradeHistory)
	err := common.SendHTTPGetRequest(common.EncodeURLValues(url, vals), true, h.Verbose, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.TradeHistory, err
}

// GetMarketDetail returns the ticker for the specified symbol
func (h *HUOBI) GetMarketDetail(symbol string) (Detail, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)

	type response struct {
		Response
		Tick Detail `json:"tick"`
	}

	var result response
	url := fmt.Sprintf("%s/%s", huobiAPIURL, huobiMarketDetail)
	err := common.SendHTTPGetRequest(common.EncodeURLValues(url, vals), true, h.Verbose, &result)

	if result.ErrorMessage != "" {
		return result.Tick, errors.New(result.ErrorMessage)
	}
	return result.Tick, err
}

// GetSymbols returns an array of symbols supported by Huobi
func (h *HUOBI) GetSymbols() ([]Symbol, error) {
	type response struct {
		Response
		Symbols []Symbol `json:"data"`
	}

	var result response
	url := fmt.Sprintf("%s/v%s/%s", huobiAPIURL, huobiAPIVersion, huobiSymbols)
	err := common.SendHTTPGetRequest(url, true, h.Verbose, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Symbols, err
}

// GetCurrencies returns a list of currencies supported by Huobi
func (h *HUOBI) GetCurrencies() ([]string, error) {
	type response struct {
		Response
		Currencies []string `json:"data"`
	}

	var result response
	url := fmt.Sprintf("%s/v%s/%s", huobiAPIURL, huobiAPIVersion, huobiCurrencies)
	err := common.SendHTTPGetRequest(url, true, h.Verbose, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Currencies, err
}

// GetTimestamp returns the Huobi server time
func (h *HUOBI) GetTimestamp() (int64, error) {
	type response struct {
		Response
		Timestamp int64 `json:"data"`
	}

	var result response
	url := fmt.Sprintf("%s/v%s/%s", huobiAPIURL, huobiAPIVersion, huobiTimestamp)
	err := common.SendHTTPGetRequest(url, true, h.Verbose, &result)

	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.Timestamp, err
}

// GetAccounts returns the Huobi user accounts
func (h *HUOBI) GetAccounts() ([]Account, error) {
	type response struct {
		Response
		AccountData []Account `json:"data"`
	}

	var result response
	err := h.SendAuthenticatedHTTPRequest("GET", huobiAccounts, url.Values{}, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.AccountData, err
}

// GetAccountBalance returns the users Huobi account balance
func (h *HUOBI) GetAccountBalance(accountID string) ([]AccountBalance, error) {
	type response struct {
		Response
		AccountData []AccountBalance `json:"list"`
	}

	var result response
	endpoint := fmt.Sprintf(huobiAccountBalance, accountID)
	err := h.SendAuthenticatedHTTPRequest("GET", endpoint, url.Values{}, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.AccountData, err
}

// PlaceOrder submits an order to Huobi
func (h *HUOBI) PlaceOrder(symbol, source, accountID, orderType string, amount, price float64) (int64, error) {
	vals := url.Values{}
	vals.Set("account-id", accountID)
	vals.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	// Only set price if order type is not equal to buy-market or sell-market
	if orderType != "buy-market" && orderType != "sell-market" {
		vals.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}

	if source != "" {
		vals.Set("source", source)
	}

	vals.Set("symbol", symbol)
	vals.Set("type", orderType)

	type response struct {
		Response
		OrderID int64 `json:"data,string"`
	}

	var result response
	err := h.SendAuthenticatedHTTPRequest("POST", huobiOrderPlace, vals, &result)

	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.OrderID, err
}

// CancelOrder cancels an order on Huobi
func (h *HUOBI) CancelOrder(orderID int64) (int64, error) {
	type response struct {
		Response
		OrderID int64 `json:"data,string"`
	}

	var result response
	endpoint := fmt.Sprintf(huobiOrderCancel, strconv.FormatInt(orderID, 10))
	err := h.SendAuthenticatedHTTPRequest("POST", endpoint, url.Values{}, &result)

	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.OrderID, err
}

// CancelOrderBatch cancels a batch of orders -- to-do
func (h *HUOBI) CancelOrderBatch(orderIDs []int64) ([]CancelOrderBatch, error) {
	type response struct {
		Response
		Data []CancelOrderBatch `json:"data"`
	}

	var result response
	err := h.SendAuthenticatedHTTPRequest("POST", huobiOrderCancelBatch, url.Values{}, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Data, err
}

// GetOrder returns order information for the specified order
func (h *HUOBI) GetOrder(orderID int64) (OrderInfo, error) {
	type response struct {
		Response
		Order OrderInfo `json:"data"`
	}

	var result response
	endpoint := fmt.Sprintf(huobiGetOrder, strconv.FormatInt(orderID, 10))
	err := h.SendAuthenticatedHTTPRequest("GET", endpoint, url.Values{}, &result)

	if result.ErrorMessage != "" {
		return result.Order, errors.New(result.ErrorMessage)
	}
	return result.Order, err
}

// GetOrderMatchResults returns matched order info for the specified order
func (h *HUOBI) GetOrderMatchResults(orderID int64) ([]OrderMatchInfo, error) {
	type response struct {
		Response
		Orders []OrderMatchInfo `json:"data"`
	}

	var result response
	endpoint := fmt.Sprintf(huobiGetOrderMatch, strconv.FormatInt(orderID, 10))
	err := h.SendAuthenticatedHTTPRequest("GET", endpoint, url.Values{}, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Orders, err
}

// GetOrders returns a list of orders
func (h *HUOBI) GetOrders(symbol, types, start, end, states, from, direct, size string) ([]OrderInfo, error) {
	type response struct {
		Response
		Orders []OrderInfo `json:"data"`
	}

	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("states", states)

	if types != "" {
		vals.Set("types", types)
	}

	if start != "" {
		vals.Set("start-date", start)
	}

	if end != "" {
		vals.Set("end-date", end)
	}

	if from != "" {
		vals.Set("from", from)
	}

	if direct != "" {
		vals.Set("direct", direct)
	}

	if size != "" {
		vals.Set("size", size)
	}

	var result response
	err := h.SendAuthenticatedHTTPRequest("GET", huobiGetOrders, vals, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Orders, err
}

// GetOrdersMatch returns a list of matched orders
func (h *HUOBI) GetOrdersMatch(symbol, types, start, end, from, direct, size string) ([]OrderMatchInfo, error) {
	type response struct {
		Response
		Orders []OrderMatchInfo `json:"data"`
	}

	vals := url.Values{}
	vals.Set("symbol", symbol)

	if types != "" {
		vals.Set("types", types)
	}

	if start != "" {
		vals.Set("start-date", start)
	}

	if end != "" {
		vals.Set("end-date", end)
	}

	if from != "" {
		vals.Set("from", from)
	}

	if direct != "" {
		vals.Set("direct", direct)
	}

	if size != "" {
		vals.Set("size", size)
	}

	var result response
	err := h.SendAuthenticatedHTTPRequest("GET", huobiGetOrdersMatch, vals, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Orders, err
}

// MarginTransfer transfers assets into or out of the margin account
func (h *HUOBI) MarginTransfer(symbol, currency string, amount float64, in bool) (int64, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("currency", currency)
	vals.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	path := huobiMarginTransferIn
	if !in {
		path = huobiMarginTransferOut
	}

	type response struct {
		Response
		TransferID int64 `json:"data"`
	}

	var result response
	err := h.SendAuthenticatedHTTPRequest("POST", path, vals, &result)

	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.TransferID, err
}

// MarginOrder submits a margin order application
func (h *HUOBI) MarginOrder(symbol, currency string, amount float64) (int64, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("currency", currency)
	vals.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	type response struct {
		Response
		MarginOrderID int64 `json:"data"`
	}

	var result response
	err := h.SendAuthenticatedHTTPRequest("POST", huobiMarginOrders, vals, &result)

	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.MarginOrderID, err
}

// MarginRepayment repays a margin amount for a margin ID
func (h *HUOBI) MarginRepayment(orderID int64, amount float64) (int64, error) {
	vals := url.Values{}
	vals.Set("order-id", strconv.FormatInt(orderID, 10))
	vals.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	type response struct {
		Response
		MarginOrderID int64 `json:"data"`
	}

	var result response
	endpoint := fmt.Sprintf(huobiMarginRepay, strconv.FormatInt(orderID, 10))
	err := h.SendAuthenticatedHTTPRequest("POST", endpoint, vals, &result)

	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.MarginOrderID, err
}

// GetMarginLoanOrders returns the margin loan orders
func (h *HUOBI) GetMarginLoanOrders(symbol, currency, start, end, states, from, direct, size string) ([]MarginOrder, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("currency", currency)

	if start != "" {
		vals.Set("start-date", start)
	}

	if end != "" {
		vals.Set("end-date", end)
	}

	if states != "" {
		vals.Set("states", states)
	}

	if from != "" {
		vals.Set("from", from)
	}

	if direct != "" {
		vals.Set("direct", direct)
	}

	if size != "" {
		vals.Set("size", size)
	}

	type response struct {
		Response
		MarginLoanOrders []MarginOrder `json:"data"`
	}

	var result response
	err := h.SendAuthenticatedHTTPRequest("GET", huobiMarginLoanOrders, vals, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.MarginLoanOrders, err
}

// GetMarginAccountBalance returns the margin account balances
func (h *HUOBI) GetMarginAccountBalance(symbol string) ([]MarginAccountBalance, error) {
	type response struct {
		Response
		Balances []MarginAccountBalance `json:"data"`
	}

	vals := url.Values{}
	if symbol != "" {
		vals.Set("symbol", symbol)
	}

	var result response
	err := h.SendAuthenticatedHTTPRequest("GET", huobiMarginAccountBalance, vals, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Balances, err
}

// Withdraw withdraws the desired amount and currency
func (h *HUOBI) Withdraw(address, currency, addrTag string, amount, fee float64) (int64, error) {
	type response struct {
		Response
		WithdrawID int64 `json:"data"`
	}

	vals := url.Values{}
	vals.Set("address", address)
	vals.Set("currency", currency)
	vals.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	if fee != 0 {
		vals.Set("fee", strconv.FormatFloat(fee, 'f', -1, 64))
	}

	if currency == "XRP" {
		vals.Set("addr-tag", addrTag)
	}

	var result response
	err := h.SendAuthenticatedHTTPRequest("POST", huobiWithdrawCreate, vals, &result)

	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.WithdrawID, err
}

// CancelWithdraw cancels a withdraw request
func (h *HUOBI) CancelWithdraw(withdrawID int64) (int64, error) {
	type response struct {
		Response
		WithdrawID int64 `json:"data"`
	}

	vals := url.Values{}
	vals.Set("withdraw-id", strconv.FormatInt(withdrawID, 10))

	var result response
	endpoint := fmt.Sprintf(huobiWithdrawCancel, strconv.FormatInt(withdrawID, 10))
	err := h.SendAuthenticatedHTTPRequest("POST", endpoint, vals, &result)

	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.WithdrawID, err
}

// SendAuthenticatedHTTPRequest sends authenticated requests to the HUOBI API
func (h *HUOBI) SendAuthenticatedHTTPRequest(method, endpoint string, values url.Values, result interface{}) error {
	if !h.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, h.Name)
	}

	values.Set("AccessKeyId", h.APIKey)
	values.Set("SignatureMethod", "HmacSHA256")
	values.Set("SignatureVersion", "2")
	values.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05"))

	endpoint = fmt.Sprintf("/v%s/%s", huobiAPIVersion, endpoint)
	payload := fmt.Sprintf("%s\napi.huobi.pro\n%s\n%s",
		method, endpoint, values.Encode())

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	hmac := common.GetHMAC(common.HashSHA256, []byte(payload), []byte(h.APISecret))
	values.Set("Signature", common.Base64Encode(hmac))

	url := fmt.Sprintf("%s%s", huobiAPIURL, endpoint)
	url = common.EncodeURLValues(url, values)
	resp, err := common.SendHTTPRequest(method, url, headers, bytes.NewBufferString(""))

	if err != nil {
		return err
	}

	err = common.JSONDecode([]byte(resp), &result)
	if err != nil {
		return errors.New("unable to JSON Unmarshal response")
	}

	return nil
}
