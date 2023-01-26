package cryptodotcom

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
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
	cryptodotcomUATSandboxAPIURL = "https://uat-api.3ona.co"
	cryptodotcomAPIURL           = "https://api.crypto.com"

	// cryptodotcom websocket endpoints.
	cryptodotcomWebsocketUserAPI   = "wss://stream.crypto.com/v2/user"
	cryptodotcomWebsocketMarketAPI = "wss://stream.crypto.com/v2/market"

	cryptodotcomAPIVersion  = "/v2/"
	cryptodotcomAPIVersion1 = "/v1/"

	// Public endpoints
	publicAuth                      = "public/auth"
	publicInstruments               = "public/get-instruments"
	publicOrderbook                 = "public/get-book"
	publicCandlestick               = "public/get-candlestick"
	publicTicker                    = "public/get-ticker"
	publicTrades                    = "public/get-trades"
	publicGetValuations             = "public/get-valuations"               // skipped
	publicGetExpiredSettlementPrice = "public/get-expired-settlement-price" // skipped
	publicGetInsurance              = "public/get-insurance"                // skipped

	// Authenticated endpoints
	privateSetCancelOnDisconnect = "private/set-cancel-on-disconnect"
	privateGetCancelOnDisconnect = "private/get-cancel-on-disconnect"

	privateUserBalance              = "private/user-balance"
	privateUserBalanceHistory       = "private/user-balance-history"
	privateCreateSubAccountTransfer = "private/create-sub-account-transfer"
	privateGetSubAccountBalances    = "private/get-sub-account-balances"
	privateGetPositions             = "private/get-positions"

	privateCreateOrder           = "private/create-order"
	privateCancelOrder           = "private/cancel-order"
	privateCreateOrderList       = "private/create-order-list"
	privateCancelOrderList       = "private/cancel-order-list"
	privateCancelAllOrders       = "private/cancel-all-orders" // TODO:
	privateClosePosition         = "private/close-position"
	privateGetOrderHistory       = "private/get-order-history"
	privateGetOpenOrders         = "private/get-open-orders"
	privateGetOrderDetail        = "private/get-order-detail"
	privateGetTrades             = "private/get-trades"
	privateChangeAccountLeverage = "private/change-account-leverage"
	privateGetTransactions       = "private/get-transactions"

	// Wallet management API
	postWithdrawal = "private/create-withdrawal"

	privateGetCurrencyNetworks = "private/get-currency-networks"
	privategetDepositAddress   = "private/get-deposit-address"
	privateGetAccounts         = "private/get-accounts"

	// OTC Trading API
	privateGetOTCUser         = "private/otc/get-otc-user"
	privateGetOTCInstruments  = "private/otc/get-instruments"
	privateOTCRequestQuote    = "private/otc/request-quote"
	privateOTCAcceptQuote     = "private/otc/accept-quote"
	privateGetOTCQuoteHistory = "private/otc/get-quote-history"
	privateGetOTCTradeHistory = "private/otc/get-trade-history"

	privateGetWithdrawalHistory = "private/get-withdrawal-history"
	privateGetDepositHistory    = "private/get-deposit-history"

	// Spot Trading API
	privateGetAccountSummary = "private/get-account-summary"
)

// GetInstruments provides information on all supported instruments
func (cr *Cryptodotcom) GetInstruments(ctx context.Context) ([]Instrument, error) {
	var resp *InstrumentList
	return resp.Instruments, cr.SendHTTPRequest(ctx, exchange.RestSpot, publicInstruments, request.Unset, &resp)
}

// GetOrderbook retches the public order book for a particular instrument and depth.
func (cr *Cryptodotcom) GetOrderbook(ctx context.Context, instrumentName string, depth int64) (*OrderbookDetail, error) {
	params := url.Values{}
	if instrumentName == "" {
		return nil, errSymbolIsRequired
	}
	params.Set("instrument_name", instrumentName)
	if depth != 0 {
		params.Set("depth", strconv.FormatInt(depth, 10))
	}
	var resp *OrderbookDetail
	return resp, cr.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(publicOrderbook, params), request.Unset, &resp)
}

// GetCandlestickDetail retrieves candlesticks (k-line data history) over a given period for an instrument
func (cr *Cryptodotcom) GetCandlestickDetail(ctx context.Context, instrumentName string, interval kline.Interval) (*CandlestickDetail, error) {
	if instrumentName == "" {
		return nil, errSymbolIsRequired
	}
	params := url.Values{}
	params.Set("instrument_name", instrumentName)
	if intervalString, err := intervalToString(interval); err == nil {
		params.Set("timeframe", intervalString)
	}
	var resp *CandlestickDetail
	return resp, cr.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(publicCandlestick, params), request.Unset, &resp)
}

// GetTicker fetches the public tickers for an instrument.
func (cr *Cryptodotcom) GetTicker(ctx context.Context, instrumentName string) (*TickersResponse, error) {
	params := url.Values{}
	if instrumentName != "" {
		params.Set("instrument_name", instrumentName)
	}
	var resp *TickersResponse
	return resp, cr.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(publicTicker, params), request.Unset, &resp)
}

// GetTrades fetches the public trades for a particular instrument.
func (cr *Cryptodotcom) GetTrades(ctx context.Context, instrumentName string) (*TradesResponse, error) {
	if instrumentName == "" {
		return nil, errSymbolIsRequired
	}
	params := url.Values{}
	params.Set("instrument_name", instrumentName)
	var resp *TradesResponse
	return resp, cr.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(publicTrades, params), request.Unset, &resp)
}

// Private endpoints

// WithdrawFunds creates a withdrawal request. Withdrawal setting must be enabled for your API Key. If you do not see the option when viewing your API Key, this feature is not yet available for you.
// Withdrawal addresses must first be whitelisted in your account’s Withdrawal Whitelist page.
// Withdrawal fees and minimum withdrawal amount can be found on the Fees & Limits page on the Exchange website.
func (cr *Cryptodotcom) WithdrawFunds(ctx context.Context, ccy currency.Code, amount float64, address, addressTag, networkID, clientWithdrawalID string) (*WithdrawalItem, error) {
	if ccy.IsEmpty() {
		return nil, errInvalidCurrency
	}
	if amount <= 0 {
		return nil, fmt.Errorf("%w, withdrawal amount provided: %f", errInvalidAmount, amount)
	}
	if address == "" {
		return nil, errors.New("address is required")
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
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, postWithdrawalRate, postWithdrawal, params, &resp)
}

// GetCurrencyNetworks retrives the symbol network mapping.
func (cr *Cryptodotcom) GetCurrencyNetworks(ctx context.Context) (*CurrencyNetworkResponse, error) {
	var resp *CurrencyNetworkResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetCurrencyNetworksRate, privateGetCurrencyNetworks, nil, &resp)
}

// GetWithdrawalHistory retrives accounts withdrawal history.
func (cr *Cryptodotcom) GetWithdrawalHistory(ctx context.Context) (*WithdrawalResponse, error) {
	var resp *WithdrawalResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetWithdrawalHistoryRate, privateGetWithdrawalHistory, nil, &resp)
}

// GetDepositHistory retrives deposit history. Withdrawal setting must be enabled for your API Key. If you do not see the option when viewing your API Keys, this feature is not yet available for you.
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
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privategetDepositAddressRate, privategetDepositAddress, params, &resp)
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
func (cr *Cryptodotcom) CreateOrder(ctx context.Context, arg CreateOrderParam) (*CreateOrderResponse, error) {
	params, err := cr.getCreateParamMap(arg)
	if err != nil {
		return nil, err
	}
	var resp *CreateOrderResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCreateOrderRate, privateCreateOrder, params, &resp)
}

// CancelExistingOrder cancels and existing open order.
func (cr *Cryptodotcom) CancelExistingOrder(ctx context.Context, instrumentName, orderID string) error {
	params := make(map[string]interface{})
	if instrumentName == "" {
		return errSymbolIsRequired
	}
	if orderID == "" {
		return order.ErrOrderIDNotSet
	}
	params["instrument_name"] = instrumentName
	params["order_id"] = orderID
	return cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCancelOrderRate, privateCancelOrder, params, nil)
}

func (cr *Cryptodotcom) getCreateParamMap(arg CreateOrderParam) (map[string]interface{}, error) {
	if arg.InstrumentName == "" {
		return nil, errSymbolIsRequired
	}
	if arg.Side != order.Sell && arg.Side != order.Buy {
		return nil, fmt.Errorf("%w, side: %s", order.ErrSideIsInvalid, arg.Side)
	}
	if arg.OrderType == order.UnknownType || arg.OrderType == order.AnyType {
		return nil, fmt.Errorf("%w, Order Type: %v", order.ErrTypeIsInvalid, arg.OrderType)
	}
	switch arg.OrderType {
	case order.Limit, order.StopLimit, order.TakeProfitLimit:
		if arg.Price <= 0 { // Unit price
			return nil, fmt.Errorf("%w, price must be non-zero positive decimal value", order.ErrPriceBelowMin)
		}
		if arg.Quantity <= 0 {
			return nil, fmt.Errorf("quantity must be non-zero positive decimal value")
		}
		switch arg.OrderType {
		case order.StopLimit, order.TakeProfitLimit:
			if arg.TriggerPrice <= 0 {
				return nil, fmt.Errorf("trigger price is required for Order Type: %v", arg.OrderType)
			}
		}
	case order.Market:
		if arg.Side == order.Buy {
			if arg.Notional <= 0 && arg.Quantity <= 0 {
				return nil, fmt.Errorf("either notional or quantity must be non-zero value for order type: %v and order side: %v", arg.OrderType, arg.Side)
			}
		} else {
			if arg.Quantity <= 0 {
				return nil, fmt.Errorf("quantity must be non-zero positive decimal value for order type: %v and order side: %v", arg.OrderType, arg.Side)
			}
		}
	case order.StopLoss, order.TakeProfit:
		if arg.Side == order.Sell {
			if arg.Quantity <= 0 {
				return nil, fmt.Errorf("quantity must be non-zero positive decimal value for order type: %v and order side: %v", arg.OrderType, arg.Side)
			}
		} else {
			if arg.Notional <= 0 {
				return nil, fmt.Errorf("quantity must be non-zero positive decimal value for order type: %v", arg.OrderType)
			}
		}
		if arg.TriggerPrice <= 0 {
			return nil, fmt.Errorf("trigger price is required for Order Type: %v", arg.OrderType)
		}
	}
	params := make(map[string]interface{})
	params["instrument_name"] = arg.InstrumentName
	params["side"] = arg.Side.String()
	params["type"] = arg.OrderType.String()
	params["price"] = arg.Price
	if arg.Quantity > 0 {
		params["quantity"] = arg.Quantity
	}
	if arg.Notional > 0 {
		params["notional"] = arg.Notional
	}
	if arg.ClientOrderID != "" {
		params["client_oid"] = arg.ClientOrderID
	}
	if arg.TimeInForce != "" {
		params["time_in_force"] = arg.TimeInForce
	}
	if arg.PostOnly {
		params["exec_inst"] = "POST_ONLY"
	}
	if arg.TriggerPrice > 0 {
		params["trigger_price"] = arg.TriggerPrice
	}
	return params, nil
}

// CreateOrderList create a list of orders on the Exchange.
// contingency_type must be LIST, for list of orders creation.
// This call is asynchronous, so the response is simply a confirmation of the request.
func (cr *Cryptodotcom) CreateOrderList(ctx context.Context, contingencyType string, arg []CreateOrderParam) (*OrderCreationResponse, error) {
	params := make(map[string]interface{})
	orderParams := make([]map[string]interface{}, len(arg))
	for x := range arg {
		p, err := cr.getCreateParamMap(arg[x])
		if err != nil {
			return nil, err
		}
		orderParams[x] = p
	}
	params["order_list"] = orderParams
	var resp *OrderCreationResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCreateOrderListRate, privateCreateOrderList, params, &resp)
}

// CancelOrderList cancel a list of orders on the Exchange.
func (cr *Cryptodotcom) CancelOrderList(ctx context.Context, args []CancelOrderParam) (*CancelOrdersResponse, error) {
	if len(args) == 0 {
		return nil, errNoArgumentPassed
	}
	params := make(map[string]interface{})
	cancelOrderList := []map[string]interface{}{}
	for x := range args {
		if args[x].InstrumentName == "" && args[x].OrderID == "" {
			return nil, errors.New("either InstrumentName or OrderID is required")
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
	params["order_list"] = cancelOrderList
	var resp *CancelOrdersResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCancelOrderListRate, privateCancelOrderList, params, &resp)
}

// CancelAllPersonalOrders cancels all orders for a particular instrument/pair (asynchronous)
// This call is asynchronous, so the response is simply a confirmation of the request.
func (cr *Cryptodotcom) CancelAllPersonalOrders(ctx context.Context, instrumentName string) error {
	if instrumentName == "" {
		return errSymbolIsRequired
	}
	params := make(map[string]interface{})
	params["instrument_name"] = instrumentName
	return cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCancelAllOrdersRate, privateCancelAllOrders, params, nil)
}

// GetAccounts retrives Account and its Sub Accounts
func (cr *Cryptodotcom) GetAccounts(ctx context.Context) (*AccountResponse, error) {
	var resp *AccountResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetAccountsRate, privateGetAccounts, nil, &resp)
}

// GetTransactions fetches recent transactions
func (cr *Cryptodotcom) GetTransactions(ctx context.Context, instrumentName, journalType string, startTimestamp, endTimestamp time.Time, limit int64) (*TransactionResponse, error) {
	params := make(map[string]interface{})
	if instrumentName != "" {
		params["instrument_name"] = instrumentName
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
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetTransactionsRate, privateGetTransactions, params, &resp)
}

// CreateSubAccountTransfer transfer between subaccounts (and master account).
func (cr *Cryptodotcom) CreateSubAccountTransfer(ctx context.Context, from, to string, ccy currency.Code, amount float64) error {
	params := make(map[string]interface{})
	if from == "" {
		return errors.New("'from' subaccount address is required")
	}
	if to == "" {
		return errors.New("'to' subaccount address is required")
	}
	if ccy.IsEmpty() {
		return fmt.Errorf("%w Currency: %v", currency.ErrCurrencyCodeEmpty, ccy)
	}
	if amount <= 0 {
		return errors.New("'amount' must be greater than 0")
	}
	params["from"] = from
	params["to"] = to
	params["currency"] = ccy.String()
	params["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	return cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateCreateSubAccountTransferRate, privateCreateSubAccountTransfer, params, nil)
}

//	GetPersonalOrderHistory gets the order history for a particular instrument
//
// If paging is used, enumerate each page (starting with 0) until an empty order_list array appears in the response.
func (cr *Cryptodotcom) GetPersonalOrderHistory(ctx context.Context, instrumentName string, startTimestamp, endTimestamp time.Time, pageSize, page int64) (*PersonalOrdersResponse, error) {
	params := make(map[string]interface{})
	if instrumentName != "" {
		params["instrument_name"] = instrumentName
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

// GetPersonalOpenOrders retrives all open orders of particular instrument.
func (cr *Cryptodotcom) GetPersonalOpenOrders(ctx context.Context, instrumentName string, pageSize, page int64) (*PersonalOrdersResponse, error) {
	params := make(map[string]interface{})
	if instrumentName != "" {
		params["instrument_name"] = instrumentName
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

// GetOrderDetail retrives details on a particular order ID
func (cr *Cryptodotcom) GetOrderDetail(ctx context.Context, orderID string) (*OrderDetail, error) {
	var resp *OrderDetail
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := make(map[string]interface{})
	params["order_id"] = orderID
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOrderDetailRate, privateGetOrderDetail, params, &resp)
}

//	GetPersonalTrades gets all executed trades for a particular instrument.
//
// If paging is used, enumerate each page (starting with 0) until an empty trade_list array appears in the response.
// Users should use user.trade to keep track of real-time trades, and private/get-trades should primarily be used for recovery; typically when the websocket is disconnected.
func (cr *Cryptodotcom) GetPersonalTrades(ctx context.Context, instrumentName string, startTimestamp, endTimestamp time.Time, pageSize, page int64) (*PersonalTrades, error) {
	params := make(map[string]interface{})
	if instrumentName != "" {
		params["instrument_name"] = instrumentName
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

// SetCancelOnDisconnect cancel on Disconnect is an optional feature that will cancel all open orders created by the connection upon loss of connectivity between client or server.
func (cr *Cryptodotcom) SetCancelOnDisconnect(ctx context.Context, scope string) (*CancelOnDisconnectScope, error) {
	if scope != "ACCOUNT" && scope != "CONNECTION" {
		return nil, errInvalidOrderCancellationScope
	}
	params := make(map[string]interface{})
	params["scope"] = scope
	var resp *CancelOnDisconnectScope
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateSetCancelOnDisconnectRate, privateSetCancelOnDisconnect, params, &resp)
}

// GetOTCUser retrives OTC User.
func (cr *Cryptodotcom) GetOTCUser(ctx context.Context) (*OTCTrade, error) {
	var resp *OTCTrade
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOTCUserRate, privateGetOTCUser, nil, &resp)
}

// GetOTCInstruments retrive tradable OTC instruments.
func (cr *Cryptodotcom) GetOTCInstruments(ctx context.Context) (*OTCInstrumentsResponse, error) {
	var resp *OTCInstrumentsResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateGetOTCInstrumentsRate, privateGetOTCInstruments, nil, &resp)
}

// RequestOTCQuote request a quote to buy or sell with either base currency or quote currency.
// direction represents the order side enum with value of BUY, SELL, or TWO-WAY
func (cr *Cryptodotcom) RequestOTCQuote(ctx context.Context, baseCurrency, quoteCurrency currency.Code,
	baseCurrencySize, quoteCurrencySize float64, direction string) (*OTCQuoteResponse, error) {
	if baseCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, empty base_currency", currency.ErrCurrencyCodeEmpty)
	}
	if quoteCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, empty quote_currency", currency.ErrCurrencyCodeEmpty)
	}
	if baseCurrencySize <= 0 && quoteCurrencySize <= 0 {
		return nil, errors.New("either base_currency_size or quote_currency_size is required")
	}
	direction = strings.ToUpper(direction)
	if direction != "BUY" && direction != "SELL" && direction != "TWO-WAY" {
		return nil, errors.New("invalid order direction; must be BUY, SELL, or TWO-WAY")
	}
	params := make(map[string]interface{})
	params["base_currency"] = baseCurrency.String()
	params["quote_currency"] = quoteCurrency.String()
	if baseCurrencySize != 0 {
		params["base_currency_size"] = strconv.FormatFloat(baseCurrencySize, 'f', -1, 64)
	}
	if quoteCurrencySize != 0 {
		params["quote_currency_size"] = strconv.FormatFloat(quoteCurrencySize, 'f', -1, 64)
	}
	params["direction"] = direction
	var resp *OTCQuoteResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateOTCRequestQuoteRate, privateOTCRequestQuote, params, &resp)
}

// AcceptOTCQuote accept a quote from request quote.
func (cr *Cryptodotcom) AcceptOTCQuote(ctx context.Context, quoteID, direction string) (*AcceptQuoteResponse, error) {
	if quoteID == "" {
		return nil, errors.New("missing quote ID")
	}
	params := make(map[string]interface{})
	if direction != "" {
		params["direction"] = direction
	}
	params["quote_id"] = quoteID
	var resp *AcceptQuoteResponse
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestSpot, privateOTCAcceptQuoteRate, privateOTCAcceptQuote, params, &resp)
}

// GetOTCQuoteHistory retrives quote history.
func (cr *Cryptodotcom) GetOTCQuoteHistory(ctx context.Context, baseCurrency, quoteCurrency currency.Code,
	startTimestamp, endTimestamp time.Time, pageSize, page int64) (*QuoteHistoryResponse, error) {
	params := make(map[string]interface{})
	if !baseCurrency.IsEmpty() {
		params["base_currency"] = baseCurrency.String()
	}
	if !quoteCurrency.IsEmpty() {
		params["quote_currency"] = quoteCurrency.String()
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
	return resp, cr.SendAuthHTTPRequest(context.Background(), exchange.RestSpot, request.Unset, privateGetOTCQuoteHistory, params, &resp)
}

// GetOTCTradeHistory retrives otc trade history
func (cr *Cryptodotcom) GetOTCTradeHistory(ctx context.Context, baseCurrency, quoteCurrency currency.Code, startTimestamp, endTimestamp time.Time, pageSize, page int64) (*OTCTradeHistoryResponse, error) {
	params := make(map[string]interface{})
	if !baseCurrency.IsEmpty() {
		params["base_currency"] = baseCurrency.String()
	}
	if !quoteCurrency.IsEmpty() {
		params["quote_currency"] = quoteCurrency.String()
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

// intervalToString returns a string representation of interval.
func intervalToString(interval kline.Interval) (string, error) {
	switch interval {
	case kline.OneMin:
		return "1m", nil
	case kline.FiveMin:
		return "5m", nil
	case kline.FifteenMin:
		return "15m", nil
	case kline.ThirtyMin:
		return "30m", nil
	case kline.OneHour:
		return "1h", nil
	case kline.FourHour:
		return "4h", nil
	case kline.SixHour:
		return "6h", nil
	case kline.TwelveHour:
		return "12h", nil
	case kline.OneDay:
		return "1D", nil
	case kline.SevenDay:
		return "7D", nil
	case kline.TwoWeek:
		return "14D", nil
	case kline.OneMonth:
		return "1M", nil
	default:
		return "", fmt.Errorf("%v interval:%v", kline.ErrUnsupportedInterval, interval)
	}
}

// stringToInterval converts a string representation to kline.Interval instance.
func stringToInterval(interval string) (kline.Interval, error) {
	switch interval {
	case "1m":
		return kline.OneMin, nil
	case "5m":
		return kline.FiveMin, nil
	case "15m":
		return kline.FifteenMin, nil
	case "30m":
		return kline.ThirtyMin, nil
	case "1h":
		return kline.OneHour, nil
	case "4h":
		return kline.FourHour, nil
	case "6h":
		return kline.SixHour, nil
	case "12h":
		return kline.TwelveHour, nil
	case "1D":
		return kline.OneDay, nil
	case "7D":
		return kline.SevenDay, nil
	case "14D":
		return kline.TwoWeek, nil
	case "1M":
		return kline.OneMonth, nil
	default:
		return 0, fmt.Errorf("invalid interval string: %s", interval)
	}
}

// SendHTTPRequest send requests for un-authenticated market endpoints.
func (cr *Cryptodotcom) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result interface{}) error {
	endpointPath, err := cr.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	response := &RespData{}
	err = cr.SendPayload(ctx, f, func() (*request.Item, error) {
		return &request.Item{
			Method:        http.MethodGet,
			Path:          endpointPath + cryptodotcomAPIVersion + path,
			Result:        response,
			Verbose:       cr.Verbose,
			HTTPDebugging: cr.HTTPDebugging,
			HTTPRecording: cr.HTTPRecording,
		}, nil
	})
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
	return json.Unmarshal(response.Result, &result)
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
	response := &RespData{}
	newRequest := func() (*request.Item, error) {
		timestamp := time.Now()
		var body io.Reader
		var hmac, payload []byte
		var id string
		var idInt int64
		id, err = common.GenerateRandomString(6, common.NumberCharacters)
		if err != nil {
			return nil, err
		}
		idInt, err = strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, err
		}
		signaturePayload := path + strconv.FormatInt(idInt, 10) + creds.Key + cr.getParamString(arg) + strconv.FormatInt(timestamp.UnixMilli(), 10)
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
		payload, err = json.Marshal(req)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(payload)
		return &request.Item{
			Method:        http.MethodPost,
			Path:          endpoint + cryptodotcomAPIVersion + path,
			Headers:       headers,
			Body:          body,
			Result:        response,
			AuthRequest:   true,
			Verbose:       cr.Verbose,
			HTTPDebugging: cr.HTTPDebugging,
			HTTPRecording: cr.HTTPRecording,
		}, nil
	}
	err = cr.SendPayload(ctx, request.Unset, newRequest)
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
	return json.Unmarshal(response.Result, resp)
}

func (cr *Cryptodotcom) getParamString(params map[string]interface{}) string {
	paramString := ""
	keys := cr.sortParams(params)
	for x := range keys {
		if params[keys[x]] == nil {
			paramString += keys[x] + "null"
		}
		switch reflect.ValueOf(params[keys[x]]).Kind() {
		case reflect.Bool:
			paramString += keys[x] + strconv.FormatBool(params[keys[x]].(bool))
		case reflect.Int64:
			paramString += keys[x] + strconv.FormatInt(params[keys[x]].(int64), 10)
		case reflect.Float32:
			paramString += keys[x] + strconv.FormatFloat(params[keys[x]].(float64), 'f', -1, 64)
		case reflect.Map:
			paramString += keys[x] + cr.getParamString((params[keys[x]]).(map[string]interface{}))
		case reflect.String:
			paramString += keys[x] + params[keys[x]].(string)
		case reflect.Slice:
			listOfMaps := params[keys[x]].([]map[string]interface{})
			for y := range listOfMaps {
				paramString += cr.getParamString(listOfMaps[y])
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

func (cr *Cryptodotcom) orderTypeToString(orderType order.Type) string {
	switch orderType {
	case order.StopLimit:
		return "STOP_LIMIT"
	case order.TakeProfit:
		return "TAKE_PROFIT"
	default:
		return orderType.String()
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
