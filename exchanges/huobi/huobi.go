package huobi

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
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

	huobiAuthRate   = 100
	huobiUnauthRate = 100
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
	h.SupportsAutoPairUpdating = true
	h.SupportsRESTTickerBatching = false
	h.Requester = request.New(h.Name, request.NewRateLimit(time.Second*10, huobiAuthRate), request.NewRateLimit(time.Second*10, huobiUnauthRate), common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
}

// Setup sets user configuration
func (h *HUOBI) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		h.SetEnabled(false)
	} else {
		h.Enabled = true
		h.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		h.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		h.APIAuthPEMKey = exch.APIAuthPEMKey
		h.SetHTTPClientTimeout(exch.HTTPTimeout)
		h.SetHTTPClientUserAgent(exch.HTTPUserAgent)
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
		err = h.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns Huobi fee
func (h *HUOBI) GetFee() float64 {
	return h.Fee
}

// GetSpotKline returns kline data
// KlinesRequestParams contains symbol, period and size
func (h *HUOBI) GetSpotKline(arg KlinesRequestParams) ([]KlineItem, error) {
	vals := url.Values{}
	vals.Set("symbol", arg.Symbol)
	vals.Set("period", string(arg.Period))

	if arg.Size != 0 {
		vals.Set("size", strconv.Itoa(arg.Size))
	}

	type response struct {
		Response
		Data []KlineItem `json:"data"`
	}

	var result response
	url := fmt.Sprintf("%s/%s", huobiAPIURL, huobiMarketHistoryKline)

	err := h.SendHTTPRequest(common.EncodeURLValues(url, vals), &result)
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

	err := h.SendHTTPRequest(common.EncodeURLValues(url, vals), &result)
	if result.ErrorMessage != "" {
		return result.Tick, errors.New(result.ErrorMessage)
	}
	return result.Tick, err
}

// GetDepth returns the depth for the specified symbol
func (h *HUOBI) GetDepth(obd OrderBookDataRequestParams) (Orderbook, error) {
	vals := url.Values{}
	vals.Set("symbol", obd.Symbol)

	if obd.Type != OrderBookDataRequestParamsTypeNone {
		vals.Set("type", string(obd.Type))
	}

	type response struct {
		Response
		Depth Orderbook `json:"tick"`
	}

	var result response
	url := fmt.Sprintf("%s/%s", huobiAPIURL, huobiMarketDepth)

	err := h.SendHTTPRequest(common.EncodeURLValues(url, vals), &result)
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

	err := h.SendHTTPRequest(common.EncodeURLValues(url, vals), &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Tick.Data, err
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
func (h *HUOBI) GetLatestSpotPrice(symbol string) (float64, error) {
	list, err := h.GetTradeHistory(symbol, "1")

	if err != nil {
		return 0, err
	}
	if len(list) == 0 {
		return 0, errors.New("The length of the list is 0")
	}

	return list[0].Trades[0].Price, nil
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

	err := h.SendHTTPRequest(common.EncodeURLValues(url, vals), &result)
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

	err := h.SendHTTPRequest(common.EncodeURLValues(url, vals), &result)
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

	err := h.SendHTTPRequest(url, &result)
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

	err := h.SendHTTPRequest(url, &result)
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

	err := h.SendHTTPRequest(url, &result)
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
func (h *HUOBI) GetAccountBalance(accountID string) ([]AccountBalanceDetail, error) {
	type response struct {
		Response
		AccountBalanceData AccountBalance `json:"data"`
	}

	var result response
	endpoint := fmt.Sprintf(huobiAccountBalance, accountID)
	err := h.SendAuthenticatedHTTPRequest("GET", endpoint, url.Values{}, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.AccountBalanceData.AccountBalanceDetails, err
}

// SpotNewOrder submits an order to Huobi
func (h *HUOBI) SpotNewOrder(arg SpotNewOrderRequestParams) (int64, error) {
	vals := url.Values{}
	vals.Set("account-id", fmt.Sprintf("%d", arg.AccountID))
	vals.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))

	// Only set price if order type is not equal to buy-market or sell-market
	if arg.Type != SpotNewOrderRequestTypeBuyMarket && arg.Type != SpotNewOrderRequestTypeSellMarket {
		vals.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}

	if arg.Source != "" {
		vals.Set("source", arg.Source)
	}

	vals.Set("symbol", arg.Symbol)
	vals.Set("type", string(arg.Type))

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

// SendHTTPRequest sends an unauthenticated HTTP request
func (h *HUOBI) SendHTTPRequest(path string, result interface{}) error {
	return h.SendPayload("GET", path, nil, nil, result, false, h.Verbose)
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
	signature := common.Base64Encode(hmac)
	values.Set("Signature", signature)

	pemKey := strings.NewReader(h.APIAuthPEMKey)
	pemBytes, err := ioutil.ReadAll(pemKey)
	if err != nil {
		return fmt.Errorf("Huobi unable to ioutil.ReadAll PEM key: %s", err)
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return fmt.Errorf("Huobi block is nil")
	}

	x509Encoded := block.Bytes
	privKey, err := x509.ParseECPrivateKey(x509Encoded)
	if err != nil {
		return fmt.Errorf("Huobi unable to ParseECPrivKey: %s", err)
	}

	r, s, err := ecdsa.Sign(rand.Reader, privKey, common.GetSHA256([]byte(signature)))
	if err != nil {
		return fmt.Errorf("Huobi unable to sign: %s", err)
	}

	privSig := r.Bytes()
	privSig = append(privSig, s.Bytes()...)
	values.Set("PrivateSignature", common.Base64Encode(privSig))

	url := fmt.Sprintf("%s%s", huobiAPIURL, endpoint)
	url = common.EncodeURLValues(url, values)

	return h.SendPayload(method, url, headers, bytes.NewBufferString(""), result, true, h.Verbose)
}
