package cryptodotcom

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Cryptodotcom is the overarching type across this package
type Cryptodotcom struct {
	exchange.Base
}

const (
	// cryptodotcom API endpoints.
	cryptodotcomAPIURLV1 = "https://api.crypto.com/exchange/v1/"
	cryptodotcomAPIURLV2 = "https://api.crypto.com/v2/"

	// cryptodotcom websocket endpoints.
	cryptodotcomWebsocketUserAPI   = "wss://stream.crypto.com/v2/user"
	cryptodotcomWebsocketMarketAPI = "wss://stream.crypto.com/v2/market"

	// Public endpoints
	publicAuth        = "public/auth"
	publicInstruments = "public/get-instruments"
	publicOrderbook   = "public/get-book"
	publicCandlestick = "public/get-candlestick"
	publicTickers     = "public/get-ticker"
	publicTrades      = "public/get-trades"

	// Authenticated endpoints
	privateSetCancelOnDisconnect = "private/set-cancel-on-disconnect"
	privateGetCancelOnDisconnect = "private/get-cancel-on-disconnect"

	privateCreateSubAccountTransfer = "private/create-sub-account-transfer"

	privateCreateOrder     = "private/create-order"
	privateCancelOrder     = "private/cancel-order"
	privateCreateOrderList = "private/create-order-list"
	privateCancelOrderList = "private/cancel-order-list"
	privateCancelAllOrders = "private/cancel-all-orders"
	privateGetOrderHistory = "private/get-order-history"
	privateGetOpenOrders   = "private/get-open-orders"
	privateGetOrderDetail  = "private/get-order-detail"
	privateGetTrades       = "private/get-trades"
	privateGetTransactions = "private/get-transactions"

	// Wallet management API
	privateWithdrawal = "private/create-withdrawal"

	privateGetCurrencyNetworks = "private/get-currency-networks"
	privateGetDepositAddress   = "private/get-deposit-address"
	privateGetAccounts         = "private/get-accounts"
	privateCreateSubAccount    = "private/create-subaccount-transfer"

	// OTC Trading API
	privateGetOTCUser         = "private/otc/get-otc-user"
	privateGetOTCInstruments  = "private/otc/get-instruments"
	privateOTCRequestQuote    = "private/otc/request-quote"
	privateOTCAcceptQuote     = "private/otc/accept-quote"
	privateGetOTCQuoteHistory = "private/otc/get-quote-history"
	privateGetOTCTradeHistory = "private/otc/get-trade-history"

	privateCreateOTCOrder = "private/otc/create-order"

	privateGetWithdrawalHistory = "private/get-withdrawal-history"
	privateGetDepositHistory    = "private/get-deposit-history"

	// Spot Trading API
	privateGetAccountSummary = "private/get-account-summary"
)

// GetRiskParameters provides information on risk parameter settings for Smart Cross Margin.
func (cr *Cryptodotcom) GetRiskParameters(ctx context.Context) (*SmartCrossMarginRiskParameter, error) {
	var resp *SmartCrossMarginRiskParameter
	return resp, cr.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, "public/get-risk-parameters", request.UnAuth, &resp)
}

// GetInstruments provides information on all supported instruments
func (cr *Cryptodotcom) GetInstruments(ctx context.Context) (*InstrumentList, error) {
	var resp *InstrumentList
	err := cr.SendHTTPRequest(ctx, exchange.RestSpot, publicInstruments, publicInstrumentsRate, &resp)
	switch {
	case err != nil:
		return nil, err
	case resp == nil:
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOrderbook retches the public order book for a particular instrument and depth.
func (cr *Cryptodotcom) GetOrderbook(ctx context.Context, symbol string, depth int64) (*OrderbookDetail, error) {
	params, err := checkInstrumentName(symbol)
	if err != nil {
		return nil, err
	}
	if depth != 0 {
		params.Set("depth", strconv.FormatInt(depth, 10))
	}
	var resp *OrderbookDetail
	return resp, cr.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(publicOrderbook, params), publicOrderbookRate, &resp)
}

func checkInstrumentName(symbol string) (url.Values, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("instrument_name", symbol)
	return params, nil
}

// GetCandlestickDetail retrieves candlesticks (k-line data history) over a given period for an instrument
func (cr *Cryptodotcom) GetCandlestickDetail(ctx context.Context, symbol string, interval kline.Interval) (*CandlestickDetail, error) {
	params, err := checkInstrumentName(symbol)
	if err != nil {
		return nil, err
	}
	if intervalString, err := intervalToString(interval); err == nil {
		params.Set("timeframe", intervalString)
	}
	var resp *CandlestickDetail
	return resp, cr.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(publicCandlestick, params), publicCandlestickRate, &resp)
}

// GetTickers fetches the public tickers for an instrument.
func (cr *Cryptodotcom) GetTickers(ctx context.Context, symbol string) (*TickersResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("instrument_name", symbol)
	}
	var resp *TickersResponse
	return resp, cr.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(publicTickers, params), publicTickerRate, &resp)
}

// GetTrades fetches the public trades for a particular instrument.
func (cr *Cryptodotcom) GetTrades(ctx context.Context, symbol string) (*TradesResponse, error) {
	params, err := checkInstrumentName(symbol)
	if err != nil {
		return nil, err
	}
	var resp *TradesResponse
	return resp, cr.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(publicTrades, params), publicTradesRate, &resp)
}

// Private endpoints

// WithdrawFunds creates a withdrawal request. Withdrawal setting must be enabled for your API Key. If you do not see the option when viewing your API Key, this feature is not yet available for you.
// Withdrawal addresses must first be whitelisted in your account’s Withdrawal Whitelist page.
// Withdrawal fees and minimum withdrawal amount can be found on the Fees & Limits page on the Exchange website.
func (cr *Cryptodotcom) WithdrawFunds(ctx context.Context, ccy currency.Code, amount float64, address, addressTag, networkID, clientWithdrawalID string) (*WithdrawalItem, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, fmt.Errorf("%w, withdrawal amount provided: %f", order.ErrAmountBelowMin, amount)
	}
	if address == "" {
		return nil, errAddressRequired
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
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, postWithdrawalRate, privateWithdrawal, params, &resp)
}

// GetCurrencyNetworks retrieves the symbol network mapping.
func (cr *Cryptodotcom) GetCurrencyNetworks(ctx context.Context) (*CurrencyNetworkResponse, error) {
	var resp *CurrencyNetworkResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetCurrencyNetworksRate, privateGetCurrencyNetworks, nil, &resp)
}

// GetWithdrawalHistory retrieves accounts withdrawal history.
func (cr *Cryptodotcom) GetWithdrawalHistory(ctx context.Context) (*WithdrawalResponse, error) {
	var resp *WithdrawalResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetWithdrawalHistoryRate, privateGetWithdrawalHistory, nil, &resp)
}

// GetDepositHistory retrieves deposit history. Withdrawal setting must be enabled for your API Key. If you do not see the option when viewing your API Keys, this feature is not yet available for you.
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
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetDepositHistoryRate, privateGetDepositHistory, params, &resp)
}

// GetPersonalDepositAddress fetches deposit address. Withdrawal setting must be enabled for your API Key. If you do not see the option when viewing your API Keys, this feature is not yet available for you.
func (cr *Cryptodotcom) GetPersonalDepositAddress(ctx context.Context, ccy currency.Code) (*DepositAddresses, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := make(map[string]interface{})
	params["currency"] = ccy.String()
	var resp *DepositAddresses
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privategetDepositAddressRate, privateGetDepositAddress, params, &resp)
}

// CreateExportRequest creates a new export
// requested_data possible values: SPOT_ORDER, SPOT_TRADE, MARGIN_ORDER, MARGIN_TRADE , OEX_ORDER, OEX_TRADE
func (cr *Cryptodotcom) CreateExportRequest(ctx context.Context, symbol, clientRequestID string, startTime, endTime time.Time, requestedData []string) (*ExportRequestResponse, error) {
	params := make(map[string]interface{})
	if symbol != "" {
		params["instrument_names"] = symbol
	}
	err := common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	if len(requestedData) == 0 {
		return nil, errRequestedDataTypesRequired
	}
	params["start_ts"] = startTime.UnixMilli()
	params["end_ts"] = endTime.UnixMilli()
	params["requested_data"] = requestedData
	if clientRequestID != "" {
		params["client_request_id"] = clientRequestID
	}
	var resp *ExportRequestResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, createExportRequestRate, "private/export/create-export-request", params, &resp)
}

// GetExportRequests retrieves an export requests
func (cr *Cryptodotcom) GetExportRequests(ctx context.Context, symbol string, startTime, endTime time.Time, requestedData []string, pageSize, page int64) (*ExportRequests, error) {
	params := make(map[string]interface{})
	if symbol != "" {
		params["instrument_names"] = symbol
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params["start_ts"] = startTime.UnixMilli()
		params["end_ts"] = endTime.UnixMilli()
	}
	if len(requestedData) != 0 {
		params["requested_data"] = requestedData
	}
	if pageSize > 0 {
		params["page_size"] = pageSize
	}
	if page > 0 {
		params["page"] = page
	}
	var resp *ExportRequests
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, getExportRequestRate, "private/export/get-export-requests", params, &resp)
}

// SPOT Trading API endpoints.

// GetAccountSummary returns the account balance of a user for a particular token.
func (cr *Cryptodotcom) GetAccountSummary(ctx context.Context, ccy currency.Code) (*Accounts, error) {
	params := make(map[string]interface{})
	if !ccy.IsEmpty() {
		params["currency"] = ccy.String()
	}
	var resp *Accounts
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetAccountSummaryRate, privateGetAccountSummary, params, &resp)
}

// CreateOrder created a new BUY or SELL order on the Exchange.
func (cr *Cryptodotcom) CreateOrder(ctx context.Context, arg *CreateOrderParam) (*CreateOrderResponse, error) {
	params, err := arg.getCreateParamMap()
	if err != nil {
		return nil, err
	}
	var resp *CreateOrderResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCreateOrderRate, privateCreateOrder, params, &resp)
}

// CancelExistingOrder cancels and existing open order.
func (cr *Cryptodotcom) CancelExistingOrder(ctx context.Context, symbol, orderID string) error {
	if symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	if orderID == "" {
		return order.ErrOrderIDNotSet
	}
	params := make(map[string]interface{})
	params["instrument_name"] = symbol
	params["order_id"] = orderID
	return cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCancelOrderRate, privateCancelOrder, params, nil)
}

// CreateOrderList create a list of orders on the Exchange.
// contingency_type must be LIST, for list of orders creation.
// This call is asynchronous, so the response is simply a confirmation of the request.
func (cr *Cryptodotcom) CreateOrderList(ctx context.Context, contingencyType string, arg []CreateOrderParam) (*OrderCreationResponse, error) {
	orderParams := make([]map[string]interface{}, len(arg))
	for x := range arg {
		p, err := arg[x].getCreateParamMap()
		if err != nil {
			return nil, err
		}
		orderParams[x] = p
	}
	params := make(map[string]interface{})
	if contingencyType == "" {
		contingencyType = "LIST"
	}
	params["order_list"] = orderParams
	params["contingency_type"] = contingencyType
	var resp *OrderCreationResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCreateOrderListRate, privateCreateOrderList, params, &resp)
}

// CancelOrderList cancel a list of orders on the Exchange.
func (cr *Cryptodotcom) CancelOrderList(ctx context.Context, args []CancelOrderParam) (*CancelOrdersResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrNilPointer
	}
	cancelOrderList := make([]map[string]interface{}, 0, len(args))
	for x := range args {
		if args[x].InstrumentName == "" && args[x].OrderID == "" {
			return nil, errInstrumentNameOrOrderIDRequired
		}
		result := make(map[string]interface{})
		if args[x].InstrumentName != "" {
			result["instrument_name"] = args[x].InstrumentName
		}
		if args[x].OrderID != "" {
			result["order_id"] = args[x].OrderID
		}
		cancelOrderList = append(cancelOrderList, result)
	}
	params := make(map[string]interface{})
	params["order_list"] = cancelOrderList
	var resp *CancelOrdersResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCancelOrderListRate, privateCancelOrderList, params, &resp)
}

// CancelAllPersonalOrders cancels all orders for a particular instrument/pair (asynchronous)
// This call is asynchronous, so the response is simply a confirmation of the request.
func (cr *Cryptodotcom) CancelAllPersonalOrders(ctx context.Context, symbol string) error {
	if symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	params := make(map[string]interface{})
	params["instrument_name"] = symbol
	return cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCancelAllOrdersRate, privateCancelAllOrders, params, nil)
}

// GetAccounts retrieves Account and its Sub Accounts
func (cr *Cryptodotcom) GetAccounts(ctx context.Context) (*AccountResponse, error) {
	var resp *AccountResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, privateGetAccountsRate, privateGetAccounts, nil, &resp)
}

// SubAccountTransfer transfer between subaccounts (and master account).
// Possible value for 'from' and 'to' : the master account UUID, or a sub-account UUID.
func (cr *Cryptodotcom) SubAccountTransfer(ctx context.Context, from, to string, ccy currency.Code, amount float64) error {
	if from == "" {
		return fmt.Errorf("%w source address, 'from', is missing", errSubAccountAddressRequired)
	}
	if to == "" {
		return fmt.Errorf("%w destination address, 'to', is missing", errSubAccountAddressRequired)
	}
	if ccy.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return order.ErrAmountBelowMin
	}
	params := make(map[string]interface{})
	params["from"] = from
	params["to"] = to
	params["currency"] = ccy.String()
	params["amount"] = amount
	return cr.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, privateCreateSubAccountRate, privateCreateSubAccount, params, nil)
}

// GetTransactions fetches recent transactions
func (cr *Cryptodotcom) GetTransactions(ctx context.Context, symbol, journalType string, startTimestamp, endTimestamp time.Time, limit int64) (*TransactionResponse, error) {
	params := make(map[string]interface{})
	if symbol != "" {
		params["instrument_name"] = symbol
	}
	if journalType != "" {
		params["journal_type"] = journalType
	}
	if !startTimestamp.IsZero() {
		params["start_time"] = startTimestamp.UnixMilli()
	}
	if !endTimestamp.IsZero() {
		params["end_time"] = endTimestamp.UnixMilli()
	}
	if limit > 0 {
		params["limit"] = limit
	}
	var resp *TransactionResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, privateGetTransactionsRate, privateGetTransactions, params, &resp)
}

// GetUserAccountFeeRate get fee rates for user’s account.
func (cr *Cryptodotcom) GetUserAccountFeeRate(ctx context.Context) (*FeeRate, error) {
	var resp *FeeRate
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/get-fee-rate", nil, &resp)
}

// GetInstrumentFeeRate get the instrument fee rate.
func (cr *Cryptodotcom) GetInstrumentFeeRate(ctx context.Context, symbol string) (*InstrumentFeeRate, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := make(map[string]interface{})
	params["instrument_name"] = symbol
	var resp *InstrumentFeeRate
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/get-instrument-fee-rate", params, &resp)
}

// CreateSubAccountTransfer transfer between subaccounts (and master account).
func (cr *Cryptodotcom) CreateSubAccountTransfer(ctx context.Context, from, to string, ccy currency.Code, amount float64) error {
	if from == "" {
		return fmt.Errorf("%w, 'from' is empty", errSubAccountAddressRequired)
	}
	if to == "" {
		return fmt.Errorf("%w, 'to' is empty", errSubAccountAddressRequired)
	}
	if ccy.IsEmpty() {
		return fmt.Errorf("%w Currency: %v", currency.ErrCurrencyCodeEmpty, ccy)
	}
	if amount <= 0 {
		return order.ErrAmountBelowMin
	}
	params := make(map[string]interface{})
	params["from"] = from
	params["to"] = to
	params["currency"] = ccy.String()
	params["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	return cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCreateSubAccountTransferRate, privateCreateSubAccountTransfer, params, nil)
}

// GetPersonalOrderHistory gets the order history for a particular instrument
//
// If paging is used, enumerate each page (starting with 0) until an empty order_list array appears in the response.
func (cr *Cryptodotcom) GetPersonalOrderHistory(ctx context.Context, symbol string, startTimestamp, endTimestamp time.Time, pageSize, page int64) (*PersonalOrdersResponse, error) {
	params := make(map[string]interface{})
	if symbol != "" {
		params["instrument_name"] = symbol
	}
	if !startTimestamp.IsZero() {
		params["start_ts"] = strconv.FormatInt(startTimestamp.UnixMilli(), 10)
	}
	if !endTimestamp.IsZero() {
		params["end_ts"] = strconv.FormatInt(endTimestamp.UnixMilli(), 10)
	}
	if pageSize > 0 {
		params["page_size"] = pageSize
	}
	if page > 0 {
		params["page"] = page
	}
	var resp *PersonalOrdersResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOrderHistoryRate, privateGetOrderHistory, params, &resp)
}

// GetPersonalOpenOrders retrieves all open orders of particular instrument.
func (cr *Cryptodotcom) GetPersonalOpenOrders(ctx context.Context, symbol string, pageSize, page int64) (*PersonalOrdersResponse, error) {
	params := make(map[string]interface{})
	if symbol != "" {
		params["instrument_name"] = symbol
	}
	if pageSize > 0 {
		params["page_size"] = pageSize
	}
	if page > 0 {
		params["page"] = page
	}
	var resp *PersonalOrdersResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOpenOrdersRate, privateGetOpenOrders, params, &resp)
}

// GetOrderDetail retrieves details on a particular order ID
func (cr *Cryptodotcom) GetOrderDetail(ctx context.Context, orderID string) (*OrderDetail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := make(map[string]interface{})
	params["order_id"] = orderID
	var resp *OrderDetail
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOrderDetailRate, privateGetOrderDetail, params, &resp)
}

// GetPrivateTrades gets all executed trades for a particular instrument.
//
// If paging is used, enumerate each page (starting with 0) until an empty trade_list array appears in the response.
// Users should use user.trade to keep track of real-time trades, and private/get-trades should primarily be used for recovery; typically when the websocket is disconnected.
func (cr *Cryptodotcom) GetPrivateTrades(ctx context.Context, symbol string, startTimestamp, endTimestamp time.Time, pageSize, page int64) (*PersonalTrades, error) {
	params := make(map[string]interface{})
	if symbol != "" {
		params["instrument_name"] = symbol
	}
	if !startTimestamp.IsZero() {
		params["start_ts"] = strconv.FormatInt(startTimestamp.UnixMilli(), 10)
	}
	if !endTimestamp.IsZero() {
		params["end_ts"] = strconv.FormatInt(endTimestamp.UnixMilli(), 10)
	}
	if pageSize != 0 {
		params["page_size"] = pageSize
	}
	if page != 0 {
		params["page"] = page
	}
	var resp *PersonalTrades
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetTradesRate, privateGetTrades, params, &resp)
}

// GetOTCUser retrieves OTC User.
func (cr *Cryptodotcom) GetOTCUser(ctx context.Context) (*OTCTrade, error) {
	var resp *OTCTrade
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOTCUserRate, privateGetOTCUser, nil, &resp)
}

// GetOTCInstruments retrieve tradable OTC instruments.
func (cr *Cryptodotcom) GetOTCInstruments(ctx context.Context) (*OTCInstrumentsResponse, error) {
	var resp *OTCInstrumentsResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOTCInstrumentsRate, privateGetOTCInstruments, nil, &resp)
}

// RequestOTCQuote request a quote to buy or sell with either base currency or quote currency.
// direction represents the order side enum with value of BUY, SELL, or TWO-WAY
func (cr *Cryptodotcom) RequestOTCQuote(ctx context.Context, currencyPair currency.Pair,
	baseCurrencySize, quoteCurrencySize float64, direction string) (*OTCQuoteResponse, error) {
	if !currencyPair.IsPopulated() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if baseCurrencySize <= 0 && quoteCurrencySize <= 0 {
		return nil, fmt.Errorf("%w, either base_currency_size or quote_currency_size is required", order.ErrAmountMustBeSet)
	}
	direction = strings.ToUpper(direction)
	if direction != "BUY" && direction != "SELL" && direction != "TWO-WAY" {
		return nil, fmt.Errorf("%w, invalid order direction; must be BUY, SELL, or TWO-WAY", order.ErrSideIsInvalid)
	}
	params := make(map[string]interface{})
	params["direction"] = direction
	params["base_currency"] = currencyPair.Base.String()
	params["quote_currency"] = currencyPair.Quote.String()
	if baseCurrencySize != 0 {
		params["base_currency_size"] = strconv.FormatFloat(baseCurrencySize, 'f', -1, 64)
	}
	if quoteCurrencySize != 0 {
		params["quote_currency_size"] = strconv.FormatFloat(quoteCurrencySize, 'f', -1, 64)
	}
	var resp *OTCQuoteResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateOTCRequestQuoteRate, privateOTCRequestQuote, params, &resp)
}

// AcceptOTCQuote accept a quote from request quote.
func (cr *Cryptodotcom) AcceptOTCQuote(ctx context.Context, quoteID, direction string) (*AcceptQuoteResponse, error) {
	if quoteID == "" {
		return nil, errQuoteIDRequired
	}
	params := make(map[string]interface{})
	if direction != "" {
		params["direction"] = direction
	}
	params["quote_id"] = quoteID
	var resp *AcceptQuoteResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateOTCAcceptQuoteRate, privateOTCAcceptQuote, params, &resp)
}

// GetOTCQuoteHistory retrieves quote history.
func (cr *Cryptodotcom) GetOTCQuoteHistory(ctx context.Context, currencyPair currency.Pair,
	startTimestamp, endTimestamp time.Time, pageSize, page int64) (*QuoteHistoryResponse, error) {
	params := make(map[string]interface{})
	if !currencyPair.Base.IsEmpty() {
		params["base_currency"] = currencyPair.Base.String()
	}
	if !currencyPair.Quote.IsEmpty() {
		params["quote_currency"] = currencyPair.Quote.String()
	}
	if !startTimestamp.IsZero() {
		params["start_ts"] = startTimestamp.UnixMilli()
	}
	if !endTimestamp.IsZero() {
		params["end_ts"] = endTimestamp.UnixMilli()
	}
	if pageSize > 0 {
		params["page_size"] = pageSize
	}
	if page > 0 {
		params["page"] = page
	}
	var resp *QuoteHistoryResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOTCTradeHistoryRate, privateGetOTCQuoteHistory, params, &resp)
}

// GetOTCTradeHistory retrieves otc trade history
func (cr *Cryptodotcom) GetOTCTradeHistory(ctx context.Context, currencyPair currency.Pair, startTimestamp, endTimestamp time.Time, pageSize, page int64) (*OTCTradeHistoryResponse, error) {
	params := make(map[string]interface{})
	if !currencyPair.Base.IsEmpty() {
		params["base_currency"] = currencyPair.Base.String()
	}
	if !currencyPair.Base.IsEmpty() {
		params["quote_currency"] = currencyPair.Quote.String()
	}
	if !startTimestamp.IsZero() {
		params["start_ts"] = startTimestamp.UnixMilli()
	}
	if !endTimestamp.IsZero() {
		params["end_ts"] = endTimestamp.UnixMilli()
	}
	if pageSize > 0 {
		params["page_size"] = pageSize
	}
	if page > 0 {
		params["page"] = page
	}
	var resp *OTCTradeHistoryResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOTCTradeHistoryRate, privateGetOTCTradeHistory, params, &resp)
}

// CreateOTCOrder creates a new BUY or SELL OTC order.
//
// Subscribe to otc_book.{instrument_name}
// Receive otc_book.{instrument_name} response
// Send private/otc/create-order with price obtained from step 2.
// If receive PENDING status, keep sending private/otc/get-trade-history till status FILLED or REJECTED
func (cr *Cryptodotcom) CreateOTCOrder(ctx context.Context, symbol, side, clientOrderID string, quantity, price float64, settleLater bool) (*OTCOrderResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if quantity <= 0 {
		return nil, fmt.Errorf("%w, 'quantity' is %f", order.ErrAmountBelowMin, quantity)
	}
	if price <= 0 {
		return nil, order.ErrPriceBelowMin
	}
	if side != "BUY" && side != "SELL" {
		return nil, fmt.Errorf("%w, side=%s", order.ErrSideIsInvalid, side)
	}
	params := make(map[string]interface{})
	params["instrument_name"] = symbol
	params["quantity"] = quantity
	params["price"] = price
	params["side"] = side
	if clientOrderID != "" {
		params["client_oid"] = clientOrderID
	}
	params["settle_later"] = settleLater
	var resp *OTCOrderResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCreateOTCOrderRate, privateCreateOTCOrder, params, &resp)
}

// intervalToString returns a string representation of interval.
func intervalToString(interval kline.Interval) (string, error) {
	intervalMap := map[kline.Interval]string{
		kline.OneMin: "1m", kline.FiveMin: "5m", kline.FifteenMin: "15m", kline.ThirtyMin: "30m", kline.OneHour: "1h", kline.FourHour: "4h", kline.SixHour: "6h", kline.TwelveHour: "12h", kline.OneDay: "1D", kline.SevenDay: "7D", kline.TwoWeek: "14D", kline.OneMonth: "1M"}
	intervalString, okay := intervalMap[interval]
	if !okay {
		return "", fmt.Errorf("%v interval:%v", kline.ErrUnsupportedInterval, interval)
	}
	return intervalString, nil
}

var intervalStringMap = map[string]kline.Interval{
	"1m": kline.OneMin, "5m": kline.FiveMin, "15m": kline.FifteenMin, "30m": kline.ThirtyMin, "1h": kline.OneHour, "4h": kline.FourHour, "6h": kline.SixHour, "12h": kline.TwelveHour, "1D": kline.OneDay, "7D": kline.SevenDay, "14D": kline.TwoWeek, "1M": kline.OneMonth}

// stringToInterval converts a string representation to kline.Interval instance.
func stringToInterval(interval string) (kline.Interval, error) {
	klineInterval, okay := intervalStringMap[interval]
	if !okay {
		return 0, fmt.Errorf("%w %s", kline.ErrInvalidInterval, interval)
	}
	return klineInterval, nil
}

// -------- Staking Endpoints ------------------------------------------------------------------------

// CreateStaking create a request to earn token rewards by staking on-chain in the Exchange.
func (cr *Cryptodotcom) CreateStaking(ctx context.Context, symbol string, quantity float64) (*StakingResp, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if quantity <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	params := make(map[string]interface{})
	params["instrument_name"] = symbol
	params["quantity"] = quantity
	var resp *StakingResp
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/stake", params, &resp)
}

//

// Unstake create a request to unlock staked token.
func (cr *Cryptodotcom) Unstake(ctx context.Context, symbol string, quantity float64) (*StakingResp, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if quantity <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	params := make(map[string]interface{})
	params["instrument_name"] = symbol
	params["quantity"] = quantity
	var resp *StakingResp
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/unstake", params, &resp)
}

// GetStakingPosition get the total staking position for a user/token
func (cr *Cryptodotcom) GetStakingPosition(ctx context.Context, symbol string) (*StakingPosition, error) {
	params := make(map[string]interface{})
	if symbol != "" {
		params["instrument_name"] = symbol
	}
	var resp *StakingPosition
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/get-staking-position", params, &resp)
}

// GetStakingInstruments get staking instruments information
func (cr *Cryptodotcom) GetStakingInstruments(ctx context.Context) (*StakingInstrumentsResponse, error) {
	var resp *StakingInstrumentsResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/get-staking-instruments", nil, &resp)
}

// GetOpenStakeUnStakeRequests get stake/unstake requests that status is not in final state.
func (cr *Cryptodotcom) GetOpenStakeUnStakeRequests(ctx context.Context, symbol string, startTime, endTime time.Time, limit int64) (*StakingRequestsResponse, error) {
	params := make(map[string]interface{})
	if symbol != "" {
		params["instrument_name"] = symbol
	}
	if !startTime.IsZero() {
		params["start_time"] = startTime.UnixMilli()
	}
	if !endTime.IsZero() {
		params["end_time"] = endTime.UnixMilli()
	}
	if limit > 0 {
		params["limit"] = limit
	}
	var resp *StakingRequestsResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/get-open-stake", params, &resp)
}

// GetStakingHistory get stake/unstake request history
func (cr *Cryptodotcom) GetStakingHistory(ctx context.Context, symbol string, startTime, endTime time.Time, limit int64) (*StakingRequestsResponse, error) {
	params := make(map[string]interface{})
	if symbol != "" {
		params["instrument_name"] = symbol
	}
	if !startTime.IsZero() {
		params["start_time"] = startTime.UnixMilli()
	}
	if !endTime.IsZero() {
		params["end_time"] = endTime.UnixMilli()
	}
	if limit > 0 {
		params["limit"] = limit
	}
	var resp *StakingRequestsResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/get-stake-history", params, &resp)
}

// GetStakingRewardHistory get stake/unstake request history
func (cr *Cryptodotcom) GetStakingRewardHistory(ctx context.Context, symbol string, startTime, endTime time.Time, limit int64) (*StakingRewardHistory, error) {
	params := make(map[string]interface{})
	if symbol != "" {
		params["instrument_name"] = symbol
	}
	if !startTime.IsZero() {
		params["start_time"] = startTime.UnixMilli()
	}
	if !endTime.IsZero() {
		params["end_time"] = endTime.UnixMilli()
	}
	if limit > 0 {
		params["limit"] = limit
	}
	var resp *StakingRewardHistory
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/get-reward-history", params, &resp)
}

// ConvertStakedToken create a request to convert between staked token with liquid staking token.
func (cr *Cryptodotcom) ConvertStakedToken(ctx context.Context, fromSymbol, toSymbol string, expectedRate, fromQuantity, slippageToleranceBasisPoints float64) (*StakingTokenConversionResponse, error) {
	if fromSymbol == "" {
		return nil, fmt.Errorf("%w, fromSymbol is empty", currency.ErrSymbolStringEmpty)
	}
	if toSymbol == "" {
		return nil, fmt.Errorf("%w, toSymbol is empty", currency.ErrSymbolStringEmpty)
	}
	if expectedRate <= 0 {
		return nil, errInvalidRate
	}
	if fromQuantity <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if slippageToleranceBasisPoints <= 0 {
		return nil, errInvalidSlippageToleraceBPs
	}
	params := make(map[string]interface{})
	params["from_instrument_name"] = fromSymbol
	params["to_instrument_name"] = toSymbol
	params["expected_rate"] = expectedRate
	params["from_quantity"] = fromQuantity
	params["slippage_tolerance_bps"] = slippageToleranceBasisPoints
	var resp *StakingTokenConversionResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/convert", params, &resp)
}

// GetOpenStakingConverts get convert request that status is not in final state.
func (cr *Cryptodotcom) GetOpenStakingConverts(ctx context.Context, startTime, endTime time.Time, limit int64) (*StakingConvertsHistory, error) {
	params := make(map[string]interface{})
	if !startTime.IsZero() {
		params["start_time"] = startTime.UnixMilli()
	}
	if !endTime.IsZero() {
		params["end_time"] = endTime.UnixMilli()
	}
	if limit > 0 {
		params["limit"] = limit
	}
	var resp *StakingConvertsHistory
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/get-open-convert", params, &resp)
}

// GetStakingConvertHistory get convert request history
func (cr *Cryptodotcom) GetStakingConvertHistory(ctx context.Context, startTime, endTime time.Time, limit int64) (*StakingConvertsHistory, error) {
	params := make(map[string]interface{})
	if !startTime.IsZero() {
		params["start_time"] = startTime.UnixMilli()
	}
	if !endTime.IsZero() {
		params["end_time"] = endTime.UnixMilli()
	}
	if limit > 0 {
		params["limit"] = limit
	}
	var resp *StakingConvertsHistory
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/get-convert-history", params, &resp)
}

// StakingConversionRate get conversion rate between staked token and liquid staking token
func (cr *Cryptodotcom) StakingConversionRate(ctx context.Context, symbol string) (*StakingConversionRate, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := make(map[string]interface{})
	params["instrument_name"] = symbol
	var resp *StakingConversionRate
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "public/staking/get-conversion-rate", params, &resp)
}

// SendHTTPRequest send requests for un-authenticated market endpoints.
func (cr *Cryptodotcom) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result interface{}) error {
	endpointPath, err := cr.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	response := &RespData{
		Result: result,
	}
	err = cr.SendPayload(ctx, f, func() (*request.Item, error) {
		return &request.Item{
			Method:        http.MethodGet,
			Path:          endpointPath + path,
			Result:        response,
			Verbose:       cr.Verbose,
			HTTPDebugging: cr.HTTPDebugging,
			HTTPRecording: cr.HTTPRecording,
		}, nil
	}, request.UnauthenticatedRequest)
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
	return nil
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
	response := &RespData{
		Result: resp,
	}
	newRequest := func() (*request.Item, error) {
		timestamp := time.Now()
		var id string
		id, err = common.GenerateRandomString(6, common.NumberCharacters)
		if err != nil {
			return nil, err
		}
		var idInt int64
		idInt, err = strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, err
		}
		signaturePayload := path + strconv.FormatInt(idInt, 10) + creds.Key + cr.getParamString(arg) + strconv.FormatInt(timestamp.UnixMilli(), 10)
		var hmac []byte
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
		var payload []byte
		payload, err = json.Marshal(req)
		if err != nil {
			return nil, err
		}
		body := bytes.NewBuffer(payload)
		return &request.Item{
			Method:        http.MethodPost,
			Path:          endpoint + path,
			Headers:       headers,
			Body:          body,
			Result:        response,
			Verbose:       cr.Verbose,
			HTTPDebugging: cr.HTTPDebugging,
			HTTPRecording: cr.HTTPRecording,
		}, nil
	}
	err = cr.SendPayload(ctx, epl, newRequest, request.AuthenticatedRequest)
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
	return nil
}

func (cr *Cryptodotcom) getParamString(params map[string]interface{}) string {
	paramString := ""
	keys := cr.sortParams(params)
	for x := range keys {
		if params[keys[x]] == nil {
			paramString += keys[x] + "null"
		}
		switch value := params[keys[x]].(type) {
		case bool:
			paramString += keys[x] + strconv.FormatBool(value)
		case int64:
			paramString += keys[x] + strconv.FormatInt(value, 10)
		case float64:
			paramString += keys[x] + strconv.FormatFloat(value, 'f', -1, 64)
		case map[string]interface{}:
			paramString += keys[x] + cr.getParamString(value)
		case string:
			paramString += keys[x] + value
		case []map[string]interface{}:
			for y := range value {
				paramString += cr.getParamString(value[y])
			}
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

// OrderTypeToString returns a string representation of order type for outbound requests
func OrderTypeToString(orderType order.Type) string {
	switch orderType {
	case order.StopLimit:
		return "STOP_LIMIT"
	case order.TakeProfit:
		return "TAKE_PROFIT"
	default:
		return orderType.String()
	}
}

// StringToOrderType returns an order.Type representation from string
func StringToOrderType(orderType string) (order.Type, error) {
	orderType = strings.ToUpper(orderType)
	switch orderType {
	case "STOP_LIMIT":
		return order.StopLimit, nil
	case "TAKE_PROFIT":
		return order.TakeProfit, nil
	default:
		oType, err := order.StringToOrderType(orderType)
		if err != nil {
			return order.UnknownType, fmt.Errorf("%w, %v", order.ErrTypeIsInvalid, err)
		}
		if oType == order.UnknownType || oType == order.AnyType {
			return order.UnknownType, fmt.Errorf("%w, Order Type: %v", order.ErrTypeIsInvalid, orderType)
		}
		return oType, nil
	}
}

func translateWithdrawalStatus(status string) string {
	switch status {
	case "0":
		return "Pending"
	case "1":
		return "Processing"
	case "2":
		return "Rejected"
	case "3":
		return "Payment In-progress"
	case "4":
		return "Payment Failed"
	case "5":
		return "Completed"
	case "6":
		return "Cancelled"
	default:
		return status
	}
}

func translateDepositStatus(status string) string {
	switch status {
	case "0":
		return "Not Arrived"
	case "1":
		return "Arrived"
	case "2":
		return "Failed"
	case "3":
		return "Pending"
	default:
		return status
	}
}
