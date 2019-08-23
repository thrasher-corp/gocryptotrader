package huobihadax

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	huobihadaxAPIURL     = "https://api.hadax.com"
	huobihadaxAPIVersion = "1"
	huobihadaxAPIName    = "hadax"

	huobihadaxMarketHistoryKline    = "market/history/kline"
	huobihadaxMarketDetail          = "market/detail"
	huobihadaxMarketDetailMerged    = "market/detail/merged"
	huobihadaxMarketDepth           = "market/depth"
	huobihadaxMarketTrade           = "market/trade"
	huobihadaxMarketTradeHistory    = "market/history/trade"
	huobihadaxSymbols               = "common/symbols"
	huobihadaxCurrencies            = "common/currencys"
	huobihadaxTimestamp             = "common/timestamp"
	huobihadaxAccounts              = "account/accounts"
	huobihadaxAggregatedBalance     = "subuser/aggregate-balance"
	huobihadaxAccountBalance        = "account/accounts/%s/balance"
	huobihadaxOrderPlace            = "order/orders/place"
	huobihadaxOrderCancel           = "order/orders/%s/submitcancel"
	huobihadaxGetOpenOrders         = "order/openOrders"
	huobihadaxOrderCancelBatch      = "order/orders/batchcancel"
	huobiHadaxBatchCancelOpenOrders = "order/orders/batchCancelOpenOrders"
	huobihadaxGetOrder              = "order/orders/%s"
	huobihadaxGetOrderMatch         = "order/orders/%s/matchresults"
	huobihadaxGetOrders             = "order/orders"
	huobihadaxGetOrdersMatch        = "orders/matchresults"
	huobihadaxMarginTransferIn      = "dw/transfer-in/margin"
	huobihadaxMarginTransferOut     = "dw/transfer-out/margin"
	huobihadaxMarginOrders          = "margin/orders"
	huobihadaxMarginRepay           = "margin/orders/%s/repay"
	huobihadaxMarginLoanOrders      = "margin/loan-orders"
	huobihadaxMarginAccountBalance  = "margin/accounts/balance"
	huobihadaxWithdrawCreate        = "dw/withdraw/api/create"
	huobihadaxWithdrawCancel        = "dw/withdraw-virtual/%s/cancel"
	huobiHadaxDepositAddress        = "query/deposit-withdraw"

	huobihadaxAuthRate   = 100
	huobihadaxUnauthRate = 100
)

// HUOBIHADAX is the overarching type across this package
type HUOBIHADAX struct {
	WebsocketConn              *wshandler.WebsocketConnection
	AuthenticatedWebsocketConn *wshandler.WebsocketConnection
	exchange.Base
}

// SetDefaults sets default values for the exchange
func (h *HUOBIHADAX) SetDefaults() {
	h.Name = "HuobiHadax"
	h.Enabled = false
	h.Fee = 0
	h.Verbose = false
	h.RESTPollingDelay = 10
	h.APIWithdrawPermissions = exchange.AutoWithdrawCryptoWithSetup |
		exchange.NoFiatWithdrawals
	h.RequestCurrencyPairFormat.Delimiter = ""
	h.RequestCurrencyPairFormat.Uppercase = false
	h.ConfigCurrencyPairFormat.Delimiter = "-"
	h.ConfigCurrencyPairFormat.Uppercase = true
	h.AssetTypes = []string{ticker.Spot}
	h.SupportsAutoPairUpdating = true
	h.SupportsRESTTickerBatching = false
	h.Requester = request.New(h.Name,
		request.NewRateLimit(time.Second*10, huobihadaxAuthRate),
		request.NewRateLimit(time.Second*10, huobihadaxUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	h.APIUrlDefault = huobihadaxAPIURL
	h.APIUrl = h.APIUrlDefault
	h.Websocket = wshandler.New()
	h.Websocket.Functionality = wshandler.WebsocketKlineSupported |
		wshandler.WebsocketTradeDataSupported |
		wshandler.WebsocketOrderbookSupported |
		wshandler.WebsocketSubscribeSupported |
		wshandler.WebsocketUnsubscribeSupported |
		wshandler.WebsocketAuthenticatedEndpointsSupported |
		wshandler.WebsocketAccountDataSupported |
		wshandler.WebsocketMessageCorrelationSupported
	h.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	h.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	h.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit

}

// Setup sets user configuration
func (h *HUOBIHADAX) Setup(exch *config.ExchangeConfig) {
	if !exch.Enabled {
		h.SetEnabled(false)
	} else {
		h.Enabled = true
		h.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		h.AuthenticatedWebsocketAPISupport = exch.AuthenticatedWebsocketAPISupport
		h.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		h.APIAuthPEMKeySupport = exch.APIAuthPEMKeySupport
		h.APIAuthPEMKey = exch.APIAuthPEMKey
		h.SetHTTPClientTimeout(exch.HTTPTimeout)
		h.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		h.RESTPollingDelay = exch.RESTPollingDelay
		h.Verbose = exch.Verbose
		h.HTTPDebugging = exch.HTTPDebugging
		h.BaseCurrencies = exch.BaseCurrencies
		h.AvailablePairs = exch.AvailablePairs
		h.EnabledPairs = exch.EnabledPairs
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
		err = h.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = h.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
		err = h.Websocket.Setup(h.WsConnect,
			h.Subscribe,
			h.Unsubscribe,
			exch.Name,
			exch.Websocket,
			exch.Verbose,
			HuobiHadaxSocketIOAddress,
			exch.WebsocketURL,
			exch.AuthenticatedWebsocketAPISupport)
		if err != nil {
			log.Fatal(err)
		}
		h.WebsocketConn = &wshandler.WebsocketConnection{
			ExchangeName:         h.Name,
			URL:                  HuobiHadaxSocketIOAddress,
			ProxyURL:             h.Websocket.GetProxyAddress(),
			Verbose:              h.Verbose,
			RateLimit:            rateLimit,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		}
		h.AuthenticatedWebsocketConn = &wshandler.WebsocketConnection{
			ExchangeName:         h.Name,
			URL:                  wsAccountsOrdersURL,
			ProxyURL:             h.Websocket.GetProxyAddress(),
			Verbose:              h.Verbose,
			RateLimit:            rateLimit,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		}
		h.Websocket.Orderbook.Setup(
			exch.WebsocketOrderbookBufferLimit,
			false,
			false,
			false,
			false,
			exch.Name)
	}
}

// GetSpotKline returns kline data
// KlinesRequestParams holds the Kline request params
func (h *HUOBIHADAX) GetSpotKline(arg KlinesRequestParams) ([]KlineItem, error) {
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
	urlPath := fmt.Sprintf("%s/%s", h.APIUrl, huobihadaxMarketHistoryKline)

	err := h.SendHTTPRequest(common.EncodeURLValues(urlPath, vals), &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Data, err
}

// GetMarketDetailMerged returns the ticker for the specified symbol
func (h *HUOBIHADAX) GetMarketDetailMerged(symbol string) (DetailMerged, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)

	type response struct {
		Response
		Tick DetailMerged `json:"tick"`
	}

	var result response
	urlPath := fmt.Sprintf("%s/%s", h.APIUrl, huobihadaxMarketDetailMerged)

	err := h.SendHTTPRequest(common.EncodeURLValues(urlPath, vals), &result)
	if result.ErrorMessage != "" {
		return result.Tick, errors.New(result.ErrorMessage)
	}
	return result.Tick, err
}

// GetDepth returns the depth for the specified symbol
func (h *HUOBIHADAX) GetDepth(symbol, depthType string) (Orderbook, error) {
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
	urlPath := fmt.Sprintf("%s/%s", h.APIUrl, huobihadaxMarketDepth)

	err := h.SendHTTPRequest(common.EncodeURLValues(urlPath, vals), &result)
	if result.ErrorMessage != "" {
		return result.Depth, errors.New(result.ErrorMessage)
	}
	return result.Depth, err
}

// GetTrades returns the trades for the specified symbol
func (h *HUOBIHADAX) GetTrades(symbol string) ([]Trade, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)

	type response struct {
		Response
		Tick struct {
			Data []Trade `json:"data"`
		} `json:"tick"`
	}

	var result response
	urlPath := fmt.Sprintf("%s/%s", h.APIUrl, huobihadaxMarketTrade)

	err := h.SendHTTPRequest(common.EncodeURLValues(urlPath, vals), &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Tick.Data, err
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
func (h *HUOBIHADAX) GetLatestSpotPrice(symbol string) (float64, error) {
	list, err := h.GetTradeHistory(symbol, "1")

	if err != nil {
		return 0, err
	}
	if len(list) == 0 {
		return 0, errors.New("the length of the list is 0")
	}

	return list[0].Trades[0].Price, nil
}

// GetTradeHistory returns the trades for the specified symbol
func (h *HUOBIHADAX) GetTradeHistory(symbol, size string) ([]TradeHistory, error) {
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
	urlPath := fmt.Sprintf("%s/%s", h.APIUrl, huobihadaxMarketTradeHistory)

	err := h.SendHTTPRequest(common.EncodeURLValues(urlPath, vals), &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.TradeHistory, err
}

// GetMarketDetail returns the ticker for the specified symbol
func (h *HUOBIHADAX) GetMarketDetail(symbol string) (Detail, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)

	type response struct {
		Response
		Tick Detail `json:"tick"`
	}

	var result response
	urlPath := fmt.Sprintf("%s/%s", h.APIUrl, huobihadaxMarketDetail)

	err := h.SendHTTPRequest(common.EncodeURLValues(urlPath, vals), &result)
	if result.ErrorMessage != "" {
		return result.Tick, errors.New(result.ErrorMessage)
	}
	return result.Tick, err
}

// GetSymbols returns an array of symbols supported by Huobi
func (h *HUOBIHADAX) GetSymbols() ([]Symbol, error) {
	type response struct {
		Response
		Symbols []Symbol `json:"data"`
	}

	var result response
	urlPath := fmt.Sprintf("%s/v%s/%s/%s", h.APIUrl, huobihadaxAPIVersion, huobihadaxAPIName, huobihadaxSymbols)

	err := h.SendHTTPRequest(urlPath, &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Symbols, err
}

// GetCurrencies returns a list of currencies supported by Huobi
func (h *HUOBIHADAX) GetCurrencies() ([]string, error) {
	type response struct {
		Response
		Currencies []string `json:"data"`
	}

	var result response
	urlPath := fmt.Sprintf("%s/v%s/%s/%s", h.APIUrl, huobihadaxAPIVersion, huobihadaxAPIName, huobihadaxCurrencies)

	err := h.SendHTTPRequest(urlPath, &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Currencies, err
}

// GetTimestamp returns the Huobi server time
func (h *HUOBIHADAX) GetTimestamp() (int64, error) {
	type response struct {
		Response
		Timestamp int64 `json:"data"`
	}

	var result response
	urlPath := fmt.Sprintf("%s/v%s/%s", h.APIUrl, huobihadaxAPIVersion, huobihadaxTimestamp)

	err := h.SendHTTPRequest(urlPath, &result)
	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.Timestamp, err
}

// GetAccounts returns the Huobi user accounts
func (h *HUOBIHADAX) GetAccounts() ([]Account, error) {
	type response struct {
		Response
		AccountData []Account `json:"data"`
	}

	var result response
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, huobihadaxAccounts, url.Values{}, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.AccountData, err
}

// GetAccountBalance returns the users Huobi account balance
func (h *HUOBIHADAX) GetAccountBalance(accountID string) ([]AccountBalanceDetail, error) {
	type response struct {
		Response
		AccountBalanceData AccountBalance `json:"data"`
	}

	var result response
	endpoint := fmt.Sprintf("%s/%s", huobihadaxAPIName, fmt.Sprintf(huobihadaxAccountBalance, accountID))
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, endpoint, url.Values{}, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.AccountBalanceData.AccountBalanceDetails, err
}

// GetAggregatedBalance returns the balances of all the sub-account aggregated.
func (h *HUOBIHADAX) GetAggregatedBalance() ([]AggregatedBalance, error) {
	type response struct {
		Response
		AggregatedBalances []AggregatedBalance `json:"data"`
	}

	var result response

	err := h.SendAuthenticatedHTTPRequest(
		http.MethodGet,
		huobihadaxAggregatedBalance,
		url.Values{},
		&result,
	)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.AggregatedBalances, err
}

// SpotNewOrder submits an order to Huobi
func (h *HUOBIHADAX) SpotNewOrder(arg SpotNewOrderRequestParams) (int64, error) {
	vals := make(map[string]string)
	vals["account-id"] = fmt.Sprintf("%d", arg.AccountID)
	vals["amount"] = strconv.FormatFloat(arg.Amount, 'f', -1, 64)

	// Only set price if order type is not equal to buy-market or sell-market
	if arg.Type != SpotNewOrderRequestTypeBuyMarket && arg.Type != SpotNewOrderRequestTypeSellMarket {
		vals["price"] = strconv.FormatFloat(arg.Price, 'f', -1, 64)
	}

	if arg.Source != "" {
		vals["source"] = arg.Source
	}

	vals["symbol"] = arg.Symbol
	vals["type"] = string(arg.Type)

	type response struct {
		Response
		OrderID int64 `json:"data,string"`
	}

	// The API indicates that for the POST request, the parameters of each method are not signed and authenticated. That is, only the AccessKeyId, SignatureMethod, SignatureVersion, and Timestamp parameters are required for the POST request. The other parameters are placed in the body.
	// So re-encode the Post parameter
	bytesParams, _ := json.Marshal(vals)
	postBodyParams := string(bytesParams)
	if h.Verbose {
		fmt.Println("Post params:", postBodyParams)
	}

	var result response
	endpoint := fmt.Sprintf("%s/%s", huobihadaxAPIName, huobihadaxOrderPlace)
	err := h.SendAuthenticatedHTTPPostRequest(http.MethodPost, endpoint, postBodyParams, &result)

	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.OrderID, err
}

// CancelExistingOrder cancels an order on Huobi
func (h *HUOBIHADAX) CancelExistingOrder(orderID int64) (int64, error) {
	type response struct {
		Response
		OrderID int64 `json:"data,string"`
	}

	var result response
	endpoint := fmt.Sprintf(huobihadaxOrderCancel, strconv.FormatInt(orderID, 10))
	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, endpoint, url.Values{}, &result)

	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.OrderID, err
}

// CancelOrderBatch cancels a batch of orders -- to-do
func (h *HUOBIHADAX) CancelOrderBatch(orderIDs []int64) (CancelOrderBatch, error) {
	type response struct {
		Status string           `json:"status"`
		Data   CancelOrderBatch `json:"data"`
	}

	// Used to send param formatting
	type postBody struct {
		List []int64 `json:"order-ids"`
	}

	// Format to JSON
	bytesParams, _ := common.JSONEncode(&postBody{List: orderIDs})
	postBodyParams := string(bytesParams)

	var result response
	err := h.SendAuthenticatedHTTPPostRequest(http.MethodPost, huobihadaxOrderCancelBatch, postBodyParams, &result)

	if len(result.Data.Failed) != 0 {
		errJSON, _ := common.JSONEncode(result.Data.Failed)
		return CancelOrderBatch{}, errors.New(string(errJSON))
	}
	return result.Data, err
}

// CancelOpenOrdersBatch cancels a batch of orders -- to-do
func (h *HUOBIHADAX) CancelOpenOrdersBatch(accountID, symbol string) (CancelOpenOrdersBatch, error) {
	params := url.Values{}

	params.Set("account-id", accountID)
	var result CancelOpenOrdersBatch

	data := struct {
		AccountID string `json:"account-id"`
		Symbol    string `json:"symbol"`
	}{
		AccountID: accountID,
		Symbol:    symbol,
	}

	bytesParams, _ := common.JSONEncode(data)
	postBodyParams := string(bytesParams)

	err := h.SendAuthenticatedHTTPPostRequest(http.MethodPost, huobiHadaxBatchCancelOpenOrders, postBodyParams, &result)

	if result.Data.FailedCount > 0 {
		return result, fmt.Errorf("there were %v failed order cancellations", result.Data.FailedCount)
	}

	return result, err
}

// GetOpenOrders returns a list of orders
func (h *HUOBIHADAX) GetOpenOrders(accountID, symbol, side string, size int) ([]OrderInfo, error) {
	type response struct {
		Response
		Orders []OrderInfo `json:"data"`
	}

	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("accountID", accountID)
	if len(side) > 0 {
		vals.Set("side", side)
	}
	vals.Set("size", fmt.Sprintf("%v", size))

	var result response
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, huobihadaxGetOpenOrders, vals, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}

	return result.Orders, err
}

// GetOrder returns order information for the specified order
func (h *HUOBIHADAX) GetOrder(orderID int64) (OrderInfo, error) {
	type response struct {
		Response
		Order OrderInfo `json:"data"`
	}

	var result response
	endpoint := fmt.Sprintf(huobihadaxGetOrder, strconv.FormatInt(orderID, 10))
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, endpoint, url.Values{}, &result)

	if result.ErrorMessage != "" {
		return result.Order, errors.New(result.ErrorMessage)
	}
	return result.Order, err
}

// GetOrderMatchResults returns matched order info for the specified order
func (h *HUOBIHADAX) GetOrderMatchResults(orderID int64) ([]OrderMatchInfo, error) {
	type response struct {
		Response
		Orders []OrderMatchInfo `json:"data"`
	}

	var result response
	endpoint := fmt.Sprintf(huobihadaxGetOrderMatch, strconv.FormatInt(orderID, 10))
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, endpoint, url.Values{}, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Orders, err
}

// GetOrders returns a list of orders
func (h *HUOBIHADAX) GetOrders(symbol, types, start, end, states, from, direct, size string) ([]OrderInfo, error) {
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
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, huobihadaxGetOrders, vals, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Orders, err
}

// GetOrdersMatch returns a list of matched orders
func (h *HUOBIHADAX) GetOrdersMatch(symbol, types, start, end, from, direct, size string) ([]OrderMatchInfo, error) {
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
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, huobihadaxGetOrdersMatch, vals, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Orders, err
}

// MarginTransfer transfers assets into or out of the margin account
func (h *HUOBIHADAX) MarginTransfer(symbol, currency string, amount float64, in bool) (int64, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("currency", currency)
	vals.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	path := huobihadaxMarginTransferIn
	if !in {
		path = huobihadaxMarginTransferOut
	}

	type response struct {
		Response
		TransferID int64 `json:"data"`
	}

	var result response
	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, path, vals, &result)

	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.TransferID, err
}

// MarginOrder submits a margin order application
func (h *HUOBIHADAX) MarginOrder(symbol, currency string, amount float64) (int64, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("currency", currency)
	vals.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	type response struct {
		Response
		MarginOrderID int64 `json:"data"`
	}

	var result response
	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, huobihadaxMarginOrders, vals, &result)

	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.MarginOrderID, err
}

// MarginRepayment repays a margin amount for a margin ID
func (h *HUOBIHADAX) MarginRepayment(orderID int64, amount float64) (int64, error) {
	vals := url.Values{}
	vals.Set("order-id", strconv.FormatInt(orderID, 10))
	vals.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	type response struct {
		Response
		MarginOrderID int64 `json:"data"`
	}

	var result response
	endpoint := fmt.Sprintf(huobihadaxMarginRepay, strconv.FormatInt(orderID, 10))
	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, endpoint, vals, &result)

	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.MarginOrderID, err
}

// GetMarginLoanOrders returns the margin loan orders
func (h *HUOBIHADAX) GetMarginLoanOrders(symbol, currency, start, end, states, from, direct, size string) ([]MarginOrder, error) {
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
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, huobihadaxMarginLoanOrders, vals, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.MarginLoanOrders, err
}

// GetMarginAccountBalance returns the margin account balances
func (h *HUOBIHADAX) GetMarginAccountBalance(symbol string) ([]MarginAccountBalance, error) {
	type response struct {
		Response
		Balances []MarginAccountBalance `json:"data"`
	}

	vals := url.Values{}
	if symbol != "" {
		vals.Set("symbol", symbol)
	}

	var result response
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, huobihadaxMarginAccountBalance, vals, &result)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Balances, err
}

// Withdraw withdraws the desired amount and currency
func (h *HUOBIHADAX) Withdraw(c currency.Code, address, addrTag string, amount, fee float64) (int64, error) {
	type response struct {
		Response
		WithdrawID int64 `json:"data"`
	}
	data := struct {
		Address  string `json:"address"`
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
		Fee      string `json:"fee,omitempty"`
		AddrTag  string `json:"addr-tag,omitempty"`
	}{
		Address:  address,
		Currency: c.Lower().String(),
		Amount:   strconv.FormatFloat(amount, 'f', -1, 64),
	}

	if fee > 0 {
		data.Fee = strconv.FormatFloat(fee, 'f', -1, 64)
	}

	if c == currency.XRP {
		data.AddrTag = addrTag
	}

	var result response
	bytesParams, _ := common.JSONEncode(data)
	postBodyParams := string(bytesParams)
	err := h.SendAuthenticatedHTTPPostRequest(http.MethodPost, huobihadaxWithdrawCreate, postBodyParams, &result)

	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.WithdrawID, err
}

// CancelWithdraw cancels a withdraw request
func (h *HUOBIHADAX) CancelWithdraw(withdrawID int64) (int64, error) {
	type response struct {
		Response
		WithdrawID int64 `json:"data"`
	}

	vals := url.Values{}
	vals.Set("withdraw-id", strconv.FormatInt(withdrawID, 10))

	var result response
	endpoint := fmt.Sprintf(huobihadaxWithdrawCancel, strconv.FormatInt(withdrawID, 10))
	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, endpoint, vals, &result)

	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.WithdrawID, err
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (h *HUOBIHADAX) SendHTTPRequest(path string, result interface{}) error {
	return h.SendPayload(http.MethodGet,
		path,
		nil,
		nil,
		result,
		false,
		false,
		h.Verbose,
		h.HTTPDebugging,
		h.HTTPRecording)
}

// SendAuthenticatedHTTPPostRequest sends authenticated requests to the HUOBI API
func (h *HUOBIHADAX) SendAuthenticatedHTTPPostRequest(method, endpoint, postBodyValues string, result interface{}) error {
	if !h.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, h.Name)
	}

	signatureParams := url.Values{}
	signatureParams.Set("AccessKeyId", h.APIKey)
	signatureParams.Set("SignatureMethod", "HmacSHA256")
	signatureParams.Set("SignatureVersion", "2")
	signatureParams.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05"))

	endpoint = fmt.Sprintf("/v%s/%s", huobihadaxAPIVersion, endpoint)
	payload := fmt.Sprintf("%s\napi.hadax.com\n%s\n%s",
		method, endpoint, signatureParams.Encode())

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	headers["Accept-Language"] = "zh-cn"

	hmac := common.GetHMAC(common.HashSHA256, []byte(payload), []byte(h.APISecret))
	signatureParams.Set("Signature", common.Base64Encode(hmac))
	urlPath := common.EncodeURLValues(fmt.Sprintf("%s%s", h.APIUrl, endpoint),
		signatureParams)
	return h.SendPayload(method,
		urlPath,
		headers,
		bytes.NewBufferString(postBodyValues),
		result,
		true,
		false,
		h.Verbose,
		h.HTTPDebugging,
		h.HTTPRecording)
}

// SendAuthenticatedHTTPRequest sends authenticated requests to the HUOBI API
func (h *HUOBIHADAX) SendAuthenticatedHTTPRequest(method, endpoint string, values url.Values, result interface{}) error {
	if !h.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, h.Name)
	}

	values.Set("AccessKeyId", h.APIKey)
	values.Set("SignatureMethod", "HmacSHA256")
	values.Set("SignatureVersion", "2")
	values.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05"))

	endpoint = fmt.Sprintf("/v%s/%s", huobihadaxAPIVersion, endpoint)
	payload := fmt.Sprintf("%s\napi.hadax.com\n%s\n%s",
		method, endpoint, values.Encode())

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	hmac := common.GetHMAC(common.HashSHA256, []byte(payload), []byte(h.APISecret))
	values.Set("Signature", common.Base64Encode(hmac))

	urlPath := common.EncodeURLValues(fmt.Sprintf("%s%s", h.APIUrl, endpoint),
		values)
	return h.SendPayload(method,
		urlPath,
		headers,
		bytes.NewBufferString(""),
		result,
		true,
		false,
		h.Verbose,
		h.HTTPDebugging,
		h.HTTPRecording)
}

// GetFee returns an estimate of fee based on type of transaction
func (h *HUOBIHADAX) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	if feeBuilder.FeeType == exchange.OfflineTradeFee || feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		fee = calculateTradingFee(feeBuilder.Pair, feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

func calculateTradingFee(c currency.Pair, price, amount float64) float64 {
	if c.IsCryptoFiatPair() {
		return 0.001 * price * amount
	}
	return 0.002 * price * amount
}

// GetDepositWithdrawalHistory returns deposit or withdrawal data
func (h *HUOBIHADAX) GetDepositWithdrawalHistory(associatedID, currency string, isDeposit bool, size int64) ([]History, error) {
	var resp = struct {
		Response
		Data []History `json:"data"`
	}{}

	vals := url.Values{}

	if isDeposit {
		vals.Set("type", "deposit")
	} else {
		vals.Set("type", "withdraw")
	}

	vals.Set("from", associatedID)
	vals.Set("size", strconv.FormatInt(size, 10))
	vals.Set("currency", common.StringToLower(currency))

	err := h.SendAuthenticatedHTTPRequest(http.MethodGet,
		huobiHadaxDepositAddress,
		vals,
		&resp)

	if resp.ErrorMessage != "" {
		return resp.Data, errors.New(resp.ErrorMessage)
	}
	return resp.Data, err
}
