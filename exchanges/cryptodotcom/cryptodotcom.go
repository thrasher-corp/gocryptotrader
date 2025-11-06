package cryptodotcom

import (
	"bytes"
	"context"
	"encoding/hex"
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
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Exchange is the overarching type across this package
type Exchange struct {
	exchange.Base
}

const (
	// cryptodotcom API endpoints.
	apiURLV1              = "https://api.crypto.com/exchange/v1/"
	apiURLV1Supplementary = "https://api.crypto.com/v1/"

	// cryptodotcom websocket endpoints.
	cryptodotcomWebsocketUserAPI   = "wss://stream.crypto.com/v1/user"
	cryptodotcomWebsocketMarketAPI = "wss://stream.crypto.com/v1/market"

	// Public endpoints
	publicAuth = "public/auth"

	privateCreateOrder          = "private/create-order"
	privateCancelOrder          = "private/cancel-order"
	privateCreateOrderList      = "private/create-order-list"
	privateCancelOrderList      = "private/cancel-order-list"
	privateCancelAllOrders      = "private/cancel-all-orders"
	privateGetOrderHistory      = "private/get-order-history"
	privateGetOpenOrders        = "private/get-open-orders"
	privateGetTrades            = "private/get-trades"
	privateWithdrawal           = "private/create-withdrawal"
	privateGetWithdrawalHistory = "private/get-withdrawal-history"

	// Spot Trading API
	privateGetAccountSummary = "private/get-account-summary"
)

// GetRiskParameters provides information on risk parameter settings for Smart Cross Margin.
func (e *Exchange) GetRiskParameters(ctx context.Context) (*SmartCrossMarginRiskParameter, error) {
	var resp *SmartCrossMarginRiskParameter
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, "public/get-risk-parameters", request.UnAuth, &resp)
}

// GetInstruments provides information on all supported instruments
func (e *Exchange) GetInstruments(ctx context.Context) (*AllInstruments, error) {
	var resp *AllInstruments
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "public/get-instruments", publicInstrumentsRate, &resp)
}

// GetOrderbook retches the public order book for a particular instrument and depth.
func (e *Exchange) GetOrderbook(ctx context.Context, symbol string, depth int64) (*OrderbookDetail, error) {
	params, err := checkInstrumentName(symbol)
	if err != nil {
		return nil, err
	}
	if depth != 0 {
		params.Set("depth", strconv.FormatInt(depth, 10))
	}
	var resp *OrderbookDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("public/get-book", params), publicOrderbookRate, &resp)
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
func (e *Exchange) GetCandlestickDetail(ctx context.Context, symbol string, interval kline.Interval, count int64, startTime, endTime time.Time) (*CandlestickDetail, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params, err := checkInstrumentName(symbol)
	if err != nil {
		return nil, err
	}
	if intervalString, err := intervalToString(interval); err == nil {
		params.Set("timeframe", intervalString)
	}
	if count >= 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if !startTime.IsZero() {
		params.Set("start_ts", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("end_ts", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp *CandlestickDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("public/get-candlestick", params), publicCandlestickRate, &resp)
}

// GetTickers fetches the public tickers for an instrument.
func (e *Exchange) GetTickers(ctx context.Context, symbol string) (*TickersResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("instrument_name", symbol)
	}
	var resp *TickersResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("public/get-tickers", params), publicTickerRate, &resp)
}

// GetTrades fetches the public trades for a particular instrument.
func (e *Exchange) GetTrades(ctx context.Context, symbol string, count int64, startTime, endTime time.Time) (*TradesResponse, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params, err := checkInstrumentName(symbol)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if !startTime.IsZero() {
		params.Set("start_ts", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("end_ts", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp *TradesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("public/get-trades", params), publicTradesRate, &resp)
}

// GetValuations fetches certain valuation type data for a particular instrument.
// Valuation type possible values: index_price, mark_price, funding_hist, funding_rate, and estimated_funding_rate
func (e *Exchange) GetValuations(ctx context.Context, symbol, valuationType string, count int64, startTime, endTime time.Time) (*InstrumentValuation, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if valuationType == "" {
		return nil, errValuationTypeUnset
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("instrument_name", symbol)
	params.Set("valuation_type", valuationType)
	if count > 0 {
		params.Set("count", strconv.FormatInt(count, 10))
	}
	if !startTime.IsZero() {
		params.Set("start_ts", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("end_ts", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp *InstrumentValuation
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("public/get-valuations", params), getValuationsRate, &resp)
}

// Private endpoints

// WithdrawFunds creates a withdrawal request. Withdrawal setting must be enabled for your API Key. If you do not see the option when viewing your API Key, this feature is not yet available for you.
// Withdrawal addresses must first be whitelisted in your account’s Withdrawal Whitelist page.
// Withdrawal fees and minimum withdrawal amount can be found on the Fees & Limits page on the Exchange website.
func (e *Exchange) WithdrawFunds(ctx context.Context, ccy currency.Code, amount float64, address, addressTag, networkID, clientWithdrawalID string) (*WithdrawalItem, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, fmt.Errorf("%w, withdrawal amount provided: %f", order.ErrAmountIsInvalid, amount)
	}
	if address == "" {
		return nil, errAddressRequired
	}
	params := make(map[string]any)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, postWithdrawalRate, privateWithdrawal, params, &resp)
}

// GetCurrencyNetworks retrieves the symbol network mapping.
func (e *Exchange) GetCurrencyNetworks(ctx context.Context) (*CurrencyNetworkResponse, error) {
	var resp *CurrencyNetworkResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetCurrencyNetworksRate, "private/get-currency-networks", nil, &resp)
}

// GetWithdrawalHistory retrieves accounts withdrawal history.
func (e *Exchange) GetWithdrawalHistory(ctx context.Context) (*WithdrawalResponse, error) {
	var resp *WithdrawalResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetWithdrawalHistoryRate, privateGetWithdrawalHistory, nil, &resp)
}

// GetDepositHistory retrieves deposit history. Withdrawal setting must be enabled for your API Key. If you do not see the option when viewing your API Keys, this feature is not yet available for you.
func (e *Exchange) GetDepositHistory(ctx context.Context, ccy currency.Code, startTimestamp, endTimestamp time.Time, pageSize, page, status int64) (*DepositResponse, error) {
	params := make(map[string]any)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetDepositHistoryRate, "private/get-deposit-history", params, &resp)
}

// GetPersonalDepositAddress fetches deposit address. Withdrawal setting must be enabled for your API Key. If you do not see the option when viewing your API Keys, this feature is not yet available for you.
func (e *Exchange) GetPersonalDepositAddress(ctx context.Context, ccy currency.Code) (*DepositAddresses, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := make(map[string]any)
	params["currency"] = ccy.String()
	var resp *DepositAddresses
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privategetDepositAddressRate, "private/get-deposit-address", params, &resp)
}

// CreateExportRequest creates a new export
// requested_data possible values: SPOT_ORDER, SPOT_TRADE, MARGIN_ORDER, MARGIN_TRADE , OEX_ORDER, OEX_TRADE
func (e *Exchange) CreateExportRequest(ctx context.Context, symbol, clientRequestID string, startTime, endTime time.Time, requestedData []string) (*ExportRequestResponse, error) {
	params := make(map[string]any)
	if symbol != "" {
		params["instrument_names"] = symbol
	}
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, createExportRequestRate, "private/export/create-export-request", params, &resp)
}

// GetExportRequests retrieves an export requests
func (e *Exchange) GetExportRequests(ctx context.Context, symbol string, startTime, endTime time.Time, requestedData []string, pageSize, page int64) (*ExportRequests, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := make(map[string]any)
	if symbol != "" {
		params["instrument_names"] = symbol
	}
	if !startTime.IsZero() {
		params["start_ts"] = startTime.UnixMilli()
	}
	if !endTime.IsZero() {
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getExportRequestRate, "private/export/get-export-requests", params, &resp)
}

// SPOT Trading API endpoints.

// GetAccountSummary returns the account balance of a user for a particular token.
func (e *Exchange) GetAccountSummary(ctx context.Context, ccy currency.Code) (*Accounts, error) {
	params := make(map[string]any)
	if !ccy.IsEmpty() {
		params["currency"] = ccy.String()
	}
	var resp *Accounts
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetAccountSummaryRate, privateGetAccountSummary, params, &resp)
}

// CreateOrder created a new BUY or SELL order on the Exchange.
func (e *Exchange) CreateOrder(ctx context.Context, arg *OrderParam) (*CreateOrderResponse, error) {
	params, err := arg.getCreateParamMap()
	if err != nil {
		return nil, err
	}
	var resp *CreateOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCreateOrderRate, privateCreateOrder, params, &resp)
}

// StructToMap converts an interface to a map[string]any representation for SendAuthHTTPRequest to use it for signture generation.
func StructToMap(s any) (map[string]any, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	err = json.Unmarshal(data, &result)
	return result, err
}

// AmendOrder updates an open order given the order id, price, and quantity information
func (e *Exchange) AmendOrder(ctx context.Context, arg *AmendOrderParam) (*CreateOrderResponse, error) {
	if arg.OriginalClientOrderID == "" && arg.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.NewPrice <= 0 {
		return nil, order.ErrPriceMustBeSetIfLimitOrder
	}
	if arg.NewQuantity <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	params, err := StructToMap(arg)
	if err != nil {
		return nil, err
	}
	var resp *CreateOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateAmendOrderRate, "private/amend-order", params, &resp)
}

// CancelExistingOrder cancels and existing open order.
func (e *Exchange) CancelExistingOrder(ctx context.Context, symbol, orderID string) error {
	if symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	if orderID == "" {
		return order.ErrOrderIDNotSet
	}
	params := make(map[string]any)
	params["instrument_name"] = symbol
	params["order_id"] = orderID
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCancelOrderRate, privateCancelOrder, params, nil)
}

// CreateOrderList create a list of orders on the Exchange.
// contingency_type must be LIST, for list of orders creation.
// This call is asynchronous, so the response is simply a confirmation of the request.
func (e *Exchange) CreateOrderList(ctx context.Context, contingencyType string, arg []OrderParam) (*OrderCreationResponse, error) {
	orderParams := make([]map[string]any, len(arg))
	for x := range arg {
		p, err := arg[x].getCreateParamMap()
		if err != nil {
			return nil, err
		}
		orderParams[x] = p
	}
	params := make(map[string]any)
	if contingencyType == "" {
		contingencyType = "LIST"
	}
	params["order_list"] = orderParams
	params["contingency_type"] = contingencyType
	var resp *OrderCreationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCreateOrderListRate, privateCreateOrderList, params, &resp)
}

// CancelOrderList cancel a list of orders on the Exchange.
func (e *Exchange) CancelOrderList(ctx context.Context, args []CancelOrderParam) (*CancelOrdersResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrNilPointer
	}
	cancelOrderList := make([]map[string]any, 0, len(args))
	for x := range args {
		if args[x].InstrumentName == "" && args[x].OrderID == "" {
			return nil, errInstrumentNameOrOrderIDRequired
		}
		result := make(map[string]any)
		if args[x].InstrumentName != "" {
			result["instrument_name"] = args[x].InstrumentName
		}
		if args[x].OrderID != "" {
			result["order_id"] = args[x].OrderID
		}
		cancelOrderList = append(cancelOrderList, result)
	}
	params := make(map[string]any)
	params["order_list"] = cancelOrderList
	var resp *CancelOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCancelOrderListRate, privateCancelOrderList, params, &resp)
}

// CancelAllPersonalOrders cancels all orders for a particular instrument/pair (asynchronous)
// This call is asynchronous, so the response is simply a confirmation of the request.
func (e *Exchange) CancelAllPersonalOrders(ctx context.Context, symbol, orderType string) error {
	params := make(map[string]any)
	if symbol != "" {
		params["instrument_name"] = symbol
	}
	if orderType != "" {
		params["type"] = orderType
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCancelAllOrdersRate, privateCancelAllOrders, params, nil)
}

// GetAccounts retrieves Account and its Sub Accounts
func (e *Exchange) GetAccounts(ctx context.Context) (*AccountResponse, error) {
	var resp *AccountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, privateGetAccountsRate, "private/get-accounts", nil, &resp)
}

// SubAccountTransfer transfer between subaccounts (and master account).
// Possible value for 'from' and 'to' : the master account UUID, or a sub-account UUID.
func (e *Exchange) SubAccountTransfer(ctx context.Context, from, to string, ccy currency.Code, amount float64) error {
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
		return order.ErrAmountIsInvalid
	}
	params := make(map[string]any)
	params["from"] = from
	params["to"] = to
	params["currency"] = ccy.String()
	params["amount"] = amount
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, privateCreateSubAccountRate, "private/create-subaccount-transfer", params, nil)
}

// GetTransactions fetches recent transactions
func (e *Exchange) GetTransactions(ctx context.Context, symbol, journalType string, startTimestamp, endTimestamp time.Time, limit int64) (*TransactionResponse, error) {
	params := make(map[string]any)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, privateGetTransactionsRate, "private/get-transactions", params, &resp)
}

// GetUserAccountFeeRate get fee rates for user’s account.
func (e *Exchange) GetUserAccountFeeRate(ctx context.Context) (*FeeRate, error) {
	var resp *FeeRate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/get-fee-rate", nil, &resp)
}

// GetInstrumentFeeRate get the instrument fee rate.
func (e *Exchange) GetInstrumentFeeRate(ctx context.Context, symbol string) (*InstrumentFeeRate, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := make(map[string]any)
	params["instrument_name"] = symbol
	var resp *InstrumentFeeRate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/get-instrument-fee-rate", params, &resp)
}

// CreateSubAccountTransfer transfer between subaccounts (and master account).
func (e *Exchange) CreateSubAccountTransfer(ctx context.Context, from, to string, ccy currency.Code, amount float64) error {
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
		return order.ErrAmountIsInvalid
	}
	params := make(map[string]any)
	params["from"] = from
	params["to"] = to
	params["currency"] = ccy.String()
	params["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCreateSubAccountTransferRate, "private/create-sub-account-transfer", params, nil)
}

// GetPersonalOrderHistory gets the order history for a particular instrument
//
// If paging is used, enumerate each page (starting with 0) until an empty order_list array appears in the response.
func (e *Exchange) GetPersonalOrderHistory(ctx context.Context, symbol string, startTimestamp, endTimestamp time.Time, pageSize, page int64) (*PersonalOrdersResponse, error) {
	params := make(map[string]any)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOrderHistoryRate, privateGetOrderHistory, params, &resp)
}

// GetPersonalOpenOrders retrieves all open orders of particular instrument.
func (e *Exchange) GetPersonalOpenOrders(ctx context.Context, symbol string) (*PersonalOrdersResponse, error) {
	params := make(map[string]any)
	if symbol != "" {
		params["instrument_name"] = symbol
	}
	var resp *PersonalOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOpenOrdersRate, privateGetOpenOrders, params, &resp)
}

// GetOrderDetail retrieves details on a particular order ID
func (e *Exchange) GetOrderDetail(ctx context.Context, orderID, clientOrderID string) (*OrderDetail, error) {
	if orderID == "" && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := make(map[string]any)
	if orderID != "" {
		params["order_id"] = orderID
	} else {
		params["client_oid"] = clientOrderID
	}
	var resp *OrderDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOrderDetailRate, "private/get-order-detail", params, &resp)
}

// GetPrivateTrades gets all executed trades for a particular instrument.
//
// If paging is used, enumerate each page (starting with 0) until an empty trade_list array appears in the response.
// Users should use user.trade to keep track of real-time trades, and private/get-trades should primarily be used for recovery; typically when the websocket is disconnected.
func (e *Exchange) GetPrivateTrades(ctx context.Context, symbol string, startTimestamp, endTimestamp time.Time, pageSize, page int64) (*PersonalTrades, error) {
	params := make(map[string]any)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetTradesRate, privateGetTrades, params, &resp)
}

// GetOTCUser retrieves OTC User.
func (e *Exchange) GetOTCUser(ctx context.Context) (*OTCTrade, error) {
	var resp *OTCTrade
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOTCUserRate, "private/otc/get-otc-user", nil, &resp)
}

// GetOTCInstruments retrieve tradable OTC instruments.
func (e *Exchange) GetOTCInstruments(ctx context.Context) (*OTCInstrumentsResponse, error) {
	var resp *OTCInstrumentsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOTCInstrumentsRate, "private/otc/get-instruments", nil, &resp)
}

// RequestOTCQuote request a quote to buy or sell with either base currency or quote currency.
// direction represents the order side enum with value of BUY, SELL, or TWO-WAY
func (e *Exchange) RequestOTCQuote(ctx context.Context, currencyPair currency.Pair, baseCurrencySize, quoteCurrencySize float64, direction string) (*OTCQuoteResponse, error) {
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
	params := make(map[string]any)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateOTCRequestQuoteRate, "private/otc/request-quote", params, &resp)
}

// AcceptOTCQuote accept a quote from request quote.
func (e *Exchange) AcceptOTCQuote(ctx context.Context, quoteID, direction string) (*AcceptQuoteResponse, error) {
	if quoteID == "" {
		return nil, errQuoteIDRequired
	}
	params := make(map[string]any)
	if direction != "" {
		params["direction"] = direction
	}
	params["quote_id"] = quoteID
	var resp *AcceptQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateOTCAcceptQuoteRate, "private/otc/accept-quote", params, &resp)
}

// GetOTCQuoteHistory retrieves quote history.
func (e *Exchange) GetOTCQuoteHistory(ctx context.Context, currencyPair currency.Pair, startTimestamp, endTimestamp time.Time, pageSize, page int64) (*QuoteHistoryResponse, error) {
	params := make(map[string]any)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOTCTradeHistoryRate, "private/otc/get-quote-history", params, &resp)
}

// GetOTCTradeHistory retrieves otc trade history
func (e *Exchange) GetOTCTradeHistory(ctx context.Context, currencyPair currency.Pair, startTimestamp, endTimestamp time.Time, pageSize, page int64) (*OTCTradeHistoryResponse, error) {
	params := make(map[string]any)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOTCTradeHistoryRate, "private/otc/get-trade-history", params, &resp)
}

// CreateOTCOrder creates a new BUY or SELL OTC order.
//
// Subscribe to otc_book.{instrument_name}
// Receive otc_book.{instrument_name} response
// Send private/otc/create-order with price obtained from step 2.
// If receive PENDING status, keep sending private/otc/get-trade-history till status FILLED or REJECTED
func (e *Exchange) CreateOTCOrder(ctx context.Context, symbol, side, clientOrderID string, quantity, price float64, settleLater bool) (*OTCOrderResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if quantity <= 0 {
		return nil, fmt.Errorf("%w, 'quantity' is %f", order.ErrAmountIsInvalid, quantity)
	}
	if price <= 0 {
		return nil, errPriceBelowMin
	}
	if side != "BUY" && side != "SELL" {
		return nil, fmt.Errorf("%w, side=%s", order.ErrSideIsInvalid, side)
	}
	params := make(map[string]any)
	params["instrument_name"] = symbol
	params["quantity"] = quantity
	params["price"] = price
	params["side"] = side
	if clientOrderID != "" {
		params["client_oid"] = clientOrderID
	}
	params["settle_later"] = settleLater
	var resp *OTCOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCreateOTCOrderRate, "private/otc/create-order", params, &resp)
}

var intervalMap = map[kline.Interval]string{kline.OneMin: "1m", kline.FiveMin: "5m", kline.FifteenMin: "15m", kline.ThirtyMin: "30m", kline.OneHour: "1h", kline.FourHour: "4h", kline.SixHour: "6h", kline.TwelveHour: "12h", kline.OneDay: "1D", kline.SevenDay: "7D", kline.TwoWeek: "14D", kline.OneMonth: "1M"}

// intervalToString returns a string representation of interval.
func intervalToString(interval kline.Interval) (string, error) {
	intervalString, okay := intervalMap[interval]
	if !okay {
		return "", fmt.Errorf("%v interval:%v", kline.ErrUnsupportedInterval, interval)
	}
	return intervalString, nil
}

var intervalStringList = []struct {
	String   string
	Interval kline.Interval
}{
	{"1m", kline.OneMin}, {"5m", kline.FiveMin}, {"15m", kline.FifteenMin}, {"30m", kline.ThirtyMin}, {"1h", kline.OneHour}, {"4h", kline.FourHour}, {"6h", kline.SixHour}, {"12h", kline.TwelveHour}, {"1D", kline.OneDay}, {"7D", kline.SevenDay}, {"14D", kline.TwoWeek}, {"1M", kline.OneMonth},
}

// stringToInterval converts a string representation to kline.Interval instance.
func stringToInterval(interval string) (kline.Interval, error) {
	for i := range intervalStringList {
		if intervalStringList[i].String == interval {
			return intervalStringList[i].Interval, nil
		}
	}
	return 0, fmt.Errorf("%w %s", kline.ErrUnsupportedInterval, interval)
}

// -------- Staking Endpoints ------------------------------------------------------------------------

// CreateStaking create a request to earn token rewards by staking on-chain in the Exchange.
func (e *Exchange) CreateStaking(ctx context.Context, symbol string, quantity float64) (*StakingResp, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if quantity <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	params := make(map[string]any)
	params["instrument_name"] = symbol
	params["quantity"] = quantity
	var resp *StakingResp
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/stake", params, &resp)
}

// Unstake create a request to unlock staked token.
func (e *Exchange) Unstake(ctx context.Context, symbol string, quantity float64) (*StakingResp, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if quantity <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	params := make(map[string]any)
	params["instrument_name"] = symbol
	params["quantity"] = quantity
	var resp *StakingResp
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/unstake", params, &resp)
}

// GetStakingPosition get the total staking position for a user/token
func (e *Exchange) GetStakingPosition(ctx context.Context, symbol string) (*StakingPosition, error) {
	params := make(map[string]any)
	if symbol != "" {
		params["instrument_name"] = symbol
	}
	var resp *StakingPosition
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/get-staking-position", params, &resp)
}

// GetStakingInstruments get staking instruments information
func (e *Exchange) GetStakingInstruments(ctx context.Context) (*StakingInstrumentsResponse, error) {
	var resp *StakingInstrumentsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/get-staking-instruments", nil, &resp)
}

// GetOpenStakeUnStakeRequests get stake/unstake requests that status is not in final state.
func (e *Exchange) GetOpenStakeUnStakeRequests(ctx context.Context, symbol string, startTime, endTime time.Time, limit int64) (*StakingRequestsResponse, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := make(map[string]any)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/get-open-stake", params, &resp)
}

// GetStakingHistory get stake/unstake request history
func (e *Exchange) GetStakingHistory(ctx context.Context, symbol string, startTime, endTime time.Time, limit int64) (*StakingRequestsResponse, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := make(map[string]any)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/get-stake-history", params, &resp)
}

// GetStakingRewardHistory get stake/unstake request history
func (e *Exchange) GetStakingRewardHistory(ctx context.Context, symbol string, startTime, endTime time.Time, limit int64) (*StakingRewardHistory, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := make(map[string]any)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/get-reward-history", params, &resp)
}

// ConvertStakedToken create a request to convert between staked token with liquid staking token.
func (e *Exchange) ConvertStakedToken(ctx context.Context, fromSymbol, toSymbol string, expectedRate, fromQuantity, slippageToleranceBasisPoints float64) (*StakingTokenConversionResponse, error) {
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
		return nil, order.ErrAmountIsInvalid
	}
	if slippageToleranceBasisPoints <= 0 {
		return nil, errInvalidSlippageToleraceBPs
	}
	params := make(map[string]any)
	params["from_instrument_name"] = fromSymbol
	params["to_instrument_name"] = toSymbol
	params["expected_rate"] = expectedRate
	params["from_quantity"] = fromQuantity
	params["slippage_tolerance_bps"] = slippageToleranceBasisPoints
	var resp *StakingTokenConversionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/convert", params, &resp)
}

// GetOpenStakingConverts get convert request that status is not in final state.
func (e *Exchange) GetOpenStakingConverts(ctx context.Context, startTime, endTime time.Time, limit int64) (*StakingConvertsHistory, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := make(map[string]any)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/get-open-convert", params, &resp)
}

// GetStakingConvertHistory get convert request history
func (e *Exchange) GetStakingConvertHistory(ctx context.Context, startTime, endTime time.Time, limit int64) (*StakingConvertsHistory, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := make(map[string]any)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "private/staking/get-convert-history", params, &resp)
}

// StakingConversionRate get conversion rate between staked token and liquid staking token
func (e *Exchange) StakingConversionRate(ctx context.Context, symbol string) (*StakingConversionRate, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := make(map[string]any)
	params["instrument_name"] = symbol
	var resp *StakingConversionRate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "public/staking/get-conversion-rate", params, &resp)
}

// SendHTTPRequest send requests for un-authenticated market endpoints.
func (e *Exchange) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result any) error {
	endpointPath, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	response := &RespData{
		Result: result,
	}
	err = e.SendPayload(ctx, f, func() (*request.Item, error) {
		return &request.Item{
			Method:        http.MethodGet,
			Path:          endpointPath + path,
			Result:        response,
			Verbose:       e.Verbose,
			HTTPDebugging: e.HTTPDebugging,
			HTTPRecording: e.HTTPRecording,
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
func (e *Exchange) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, epl request.EndpointLimit, path string, arg map[string]any, resp any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := e.API.Endpoints.GetURL(ePath)
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
		signaturePayload := path + strconv.FormatInt(idInt, 10) + creds.Key + e.getParamString(arg) + strconv.FormatInt(timestamp.UnixMilli(), 10)
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
			Signature: hex.EncodeToString(hmac),
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
			Verbose:       e.Verbose,
			HTTPDebugging: e.HTTPDebugging,
			HTTPRecording: e.HTTPRecording,
		}, nil
	}
	err = e.SendPayload(ctx, epl, newRequest, request.AuthenticatedRequest)
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

func (e *Exchange) getParamString(params map[string]any) string {
	paramString := ""
	keys := e.sortParams(params)
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
		case map[string]any:
			paramString += keys[x] + e.getParamString(value)
		case string:
			paramString += keys[x] + value
		case []map[string]any:
			for y := range value {
				paramString += e.getParamString(value[y])
			}
		}
	}
	return paramString
}

func (e *Exchange) sortParams(params map[string]any) []string {
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
	case order.UnknownType:
		return ""
	default:
		return orderType.String()
	}
}

// StringToOrderType returns an order.Type representation from string
func StringToOrderType(orderType string) (order.Type, error) {
	orderType = strings.ToUpper(orderType)
	oType, err := order.StringToOrderType(orderType)
	if err != nil {
		return order.UnknownType, fmt.Errorf("%w, %v", order.ErrTypeIsInvalid, err)
	}
	if oType == order.UnknownType || oType == order.AnyType {
		return order.UnknownType, fmt.Errorf("%w, Order Type: %v", order.ErrTypeIsInvalid, orderType)
	}
	return oType, nil
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

// -------------------------------   Account Balance and Position endpoints ---------------------------------

// GetUserBalance returns the user's wallet balance.
func (e *Exchange) GetUserBalance(ctx context.Context) (*UserAccountBalanceDetail, error) {
	var resp *UserAccountBalanceDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.Auth, "private/user-balance", nil, &resp)
}

// GetUserBalanceHistory returns the user's balance history.
func (e *Exchange) GetUserBalanceHistory(ctx context.Context, timeFrame string, endTime time.Time, limit int64) (*UserBalanceHistory, error) {
	params := make(map[string]any)
	if timeFrame != "" {
		params["timeframe"] = timeFrame
	}
	if !endTime.IsZero() {
		params["end_time"] = endTime.UnixMilli()
	}
	if limit > 0 {
		params["limit"] = limit
	}
	var resp *UserBalanceHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.Auth, "private/user-balance-history", params, &resp)
}

// GetSubAccountBalances retrieves the user's wallet balances of all sub-accounts.
func (e *Exchange) GetSubAccountBalances(ctx context.Context) (*SubAccountBalance, error) {
	var resp *SubAccountBalance
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, request.Auth, "private/get-subaccount-balances", nil, &resp)
}

// GetPositions returns the user's position.
func (e *Exchange) GetPositions(ctx context.Context, instrumentName string) (*UsersPositions, error) {
	params := make(map[string]any)
	if instrumentName != "" {
		params["instrument_name"] = instrumentName
	}
	var resp *UsersPositions
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getPositionsRate, "private/get-positions", params, &resp)
}

// GetExpiredSettlementPrice fetches settlement price of expired instruments.
func (e *Exchange) GetExpiredSettlementPrice(ctx context.Context, instrumentype asset.Item, page int) (*ExpiredSettlementPrice, error) {
	if instrumentype == asset.Empty {
		return nil, asset.ErrInvalidAsset
	}
	params := url.Values{}
	switch instrumentype {
	case asset.Futures:
		params.Set("instrument_type", "FUTURE")
	default:
		params.Set("instrument_type", strings.ToUpper(instrumentype.String()))
	}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	var resp *ExpiredSettlementPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues("public/get-expired-settlement-price", params), expiredSettlementPriceRate, &resp)
}

// GetAnnouncements retrieves all announcements from server
func (e *Exchange) GetAnnouncements(ctx context.Context, category, productType string) (*Announcements, error) {
	params := url.Values{}
	if category != "" {
		params.Set("category", category)
	}
	if productType != "" {
		params.Set("product_type", productType)
	}
	var resp *Announcements
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("public/get-announcements", params), getAnnouncementsRate, &resp)
}

// ------------------------------- Fiat Wallet API ---------------------------------------

// GetPrivateFiatDepositInformation retrieves fiat deposit information for the authenticated user. Returns bank details for depositing fiat currency with optional payment network filtering.
func (e *Exchange) GetPrivateFiatDepositInformation(ctx context.Context, paymentNetworks string) (*FiatDepositInfoDetail, error) {
	if paymentNetworks == "" {
		return nil, errPaymentNetworkIsMissing
	}
	var resp *FiatDepositInfoDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, request.UnAuth, "private/fiat/fiat-deposit-info", map[string]any{"payment_networks": paymentNetworks}, &resp)
}

// GetFiatDepositHistory retrieves paginated fiat deposit transaction history for the authenticated user.
func (e *Exchange) GetFiatDepositHistory(ctx context.Context, page, pageSize uint64, startTime, endTime time.Time, paymentNetworks string) (*FiatDepositHistory, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	param := make(map[string]any)
	param["page"] = page
	param["page_size"] = pageSize
	if !startTime.IsZero() {
		param["start_time"] = startTime.UnixMilli()
	}
	if !endTime.IsZero() {
		param["end_time"] = endTime.UnixMilli()
	}
	if paymentNetworks != "" {
		param["payment_networks"] = paymentNetworks
	}
	var resp *FiatDepositHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, request.UnAuth, "private/fiat/fiat-deposit-history", param, &resp)
}

// GetFiatWithdrawHistory retrieves paginated fiat withdrawal transaction history for the authenticated user.
func (e *Exchange) GetFiatWithdrawHistory(ctx context.Context, page, pageSize uint64, startTime, endTime time.Time, paymentNetworks string) (*FiatWithdrawalHistory, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	param := make(map[string]any)
	param["page"] = page
	param["page_size"] = pageSize
	if !startTime.IsZero() {
		param["start_time"] = startTime.UnixMilli()
	}
	if !endTime.IsZero() {
		param["end_time"] = endTime.UnixMilli()
	}
	if paymentNetworks != "" {
		param["payment_networks"] = paymentNetworks
	}
	var resp *FiatWithdrawalHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, request.UnAuth, "private/fiat/fiat-withdraw-history", param, &resp)
}

// CreateFiatWithdrawal creates a new fiat withdrawal request for the authenticated user.
func (e *Exchange) CreateFiatWithdrawal(ctx context.Context, arg *FiatCreateWithdrawl) (*FiatWithdrawalResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AccountID == "" {
		return nil, errAccountIDMissing
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Currency == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.PaymentNetwork == "" {
		return nil, errPaymentNetworkIsMissing
	}
	param, err := StructToMap(arg)
	if err != nil {
		return nil, err
	}
	var resp *FiatWithdrawalResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, request.UnAuth, "private/fiat/fiat-create-withdraw", param, &resp)
}

// GetFiatTransactionQuota retrieves transaction quota information for a specific payment network.
func (e *Exchange) GetFiatTransactionQuota(ctx context.Context, paymentNetwork string) (*FiatWithdrawalQuota, error) {
	if paymentNetwork == "" {
		return nil, errPaymentNetworkIsMissing
	}
	var resp *FiatWithdrawalQuota
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, request.UnAuth, "private/fiat/fiat-transaction-quota", map[string]any{"payment_network": paymentNetwork}, &resp)
}

// GetFiatTransactionLimit retrieves transaction limits for a specific payment network.
func (e *Exchange) GetFiatTransactionLimit(ctx context.Context, paymentNetwork string) (*FiatTransactionLimit, error) {
	if paymentNetwork == "" {
		return nil, errPaymentNetworkIsMissing
	}
	var resp *FiatTransactionLimit
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, request.UnAuth, "private/fiat/fiat-transaction-limit", map[string]any{"payment_network": paymentNetwork}, &resp)
}

// GetFiatBankAccounts retrieves user's bank accounts with optional payment network filtering.
func (e *Exchange) GetFiatBankAccounts(ctx context.Context, paymentNetworks string) (*FiatBankAccounts, error) {
	if paymentNetworks == "" {
		return nil, errPaymentNetworkIsMissing
	}
	var resp *FiatBankAccounts
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, request.UnAuth, "private/fiat/fiat-get-bank-accounts", map[string]any{"payment_networks": paymentNetworks}, &resp)
}
