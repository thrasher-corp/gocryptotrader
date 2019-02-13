package okgroup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	// just your average return type from okex
	returnTypeOne     = "map[string]interface {}"
	okGroupAuthRate   = 0
	okGroupUnauthRate = 0
	// OkGroupAPIPath const to help with api url formatting
	OkGroupAPIPath = "api/"
	// API subsections
	okGroupAccountSubsection       = "account"
	okGroupTokenSubsection         = "spot"
	okGroupMarginTradingSubsection = "margin"
	okGroupFuturesSubsection       = "futures"
	// Common endpoints
	okGroupTradingAccounts         = "accounts"
	okGroupLedger                  = "ledger"
	okGroupOrders                  = "orders"
	okGroupBatchOrders             = "batch_orders"
	okGroupCancelOrders            = "cancel_orders"
	okGroupCancelOrder             = "cancel_order"
	okGroupCancelBatchOrders       = "cancel_batch_orders"
	okGroupPendingOrders           = "orders_pending"
	okGroupTrades                  = "trades"
	okGroupTicker                  = "ticker"
	okGroupGetSpotTokenPairDetails = "instruments"
	// Account based endpoints
	okGroupGetAccountCurrencies        = "currencies"
	okGroupGetAccountWalletInformation = "wallet"
	okGroupFundsTransfer               = "transfer"
	okGroupWithdraw                    = "withdrawal"
	okGroupGetWithdrawalFees           = "withdrawal/fee"
	okGroupGetWithdrawalHistory        = "withdrawal/history"
	okGroupGetDepositAddress           = "deposit/address"
	okGroupGetAccountDepositHistory    = "deposit/history"
	// Token based endpoints
	okGroupGetSpotTransactionDetails = "fills"
	okGroupGetSpotOrderBook          = "book"
	okGroupGetSpotMarketData         = "candles"
	// Margin based endpoints
	okGroupGetMarketAvailability = "availability"
	okGroupGetLoanHistory        = "borrowed"
	okGroupGetLoan               = "borrow"
	okGroupGetRepayment          = "repayment"
	// Futures based endpoints
	okGroupFuturePosition = "position"
	okGroupFutureLeverage = "leverage"
	okGroupFutureOrder    = "order"
	okGroupFutureHolds    = "holds"
	okGroupIndices        = "index"
	okGroupRate           = "rate"
	okGroupEsimtatedPrice = "estimated_price"
	okGroupOpenInterest   = "open_interest"
	okGroupPriceLimit     = "price_limit"
	okGroupMarkPrice      = "mark_price"
	okGroupLiquidation    = "liquidation"
	okGroupTagPrice       = "tag_price"
)

var errMissValue = errors.New("warning - resp value is missing from exchange")

// OKGroup is the overaching type across the all of OKEx's exchange methods
type OKGroup struct {
	exchange.Base
	ExchangeName  string
	WebsocketConn *websocket.Conn
	mu            sync.Mutex

	// Spot and contract market error codes as per https://www.okex.com/rest_request.html
	ErrorCodes map[string]error

	// Stores for corresponding variable checks
	ContractTypes    []string
	CurrencyPairs    []string
	ContractPosition []string
	Types            []string

	// URLs to be overridden by implementations of OKGroup
	APIURL       string
	APIVersion   string
	WebsocketURL string
}

// SetDefaults method assignes the default values for Bittrex
func (o *OKGroup) SetDefaults() {
	o.SetErrorDefaults()
	o.SetCheckVarDefaults()
	o.Name = o.ExchangeName
	o.Enabled = false
	o.Verbose = false
	o.RESTPollingDelay = 10
	o.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.NoFiatWithdrawals
	o.RequestCurrencyPairFormat.Delimiter = "_"
	o.RequestCurrencyPairFormat.Uppercase = false
	o.ConfigCurrencyPairFormat.Delimiter = "_"
	o.ConfigCurrencyPairFormat.Uppercase = true
	o.SupportsAutoPairUpdating = true
	o.SupportsRESTTickerBatching = false
	o.Requester = request.New(o.Name,
		request.NewRateLimit(time.Second, okGroupAuthRate),
		request.NewRateLimit(time.Second, okGroupUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	o.APIUrlDefault = o.APIURL
	o.APIUrl = o.APIUrlDefault
	o.AssetTypes = []string{ticker.Spot}
	o.WebsocketInit()
	o.Websocket.Functionality = exchange.WebsocketTickerSupported |
		exchange.WebsocketTradeDataSupported |
		exchange.WebsocketKlineSupported |
		exchange.WebsocketOrderbookSupported
}

// Setup method sets current configuration details if enabled
func (o *OKGroup) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		o.SetEnabled(false)
	} else {
		o.Name = exch.Name
		o.Enabled = true
		o.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		o.SetAPIKeys(exch.APIKey, exch.APISecret, exch.ClientID, false)
		o.SetHTTPClientTimeout(exch.HTTPTimeout)
		o.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		o.RESTPollingDelay = exch.RESTPollingDelay
		o.Verbose = exch.Verbose
		o.Websocket.SetEnabled(exch.Websocket)
		o.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		o.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		o.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := o.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
		err = o.WebsocketSetup(o.WsConnect,
			exch.Name,
			exch.Websocket,
			okexDefaultWebsocketURL,
			exch.WebsocketURL)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// -------------------------------------Account------------------------------------------

// GetAccountCurrencies returns a list of tradable spot instruments and their properties
func (o *OKGroup) GetAccountCurrencies() ([]GetAccountCurrenciesResponse, error) {
	var resp []GetAccountCurrenciesResponse
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupAccountSubsection, okGroupGetAccountCurrencies, nil, &resp, true)
}

// GetAccountWalletInformation returns a list of wallets and their properties
func (o *OKGroup) GetAccountWalletInformation(currency string) ([]WalletInformationResponse, error) {
	var resp []WalletInformationResponse
	var requestURL string
	if len(currency) > 0 {
		requestURL = fmt.Sprintf("%v/%v", okGroupGetAccountWalletInformation, currency)
	} else {
		requestURL = okGroupGetAccountWalletInformation
	}

	return resp, o.SendHTTPRequest(http.MethodGet, okGroupAccountSubsection, requestURL, nil, &resp, true)
}

// TransferAccountFunds  the transfer of funds between wallet, trading accounts, main account and sub accounts.
func (o *OKGroup) TransferAccountFunds(request TransferAccountFundsRequest) (TransferAccountFundsResponse, error) {
	var resp TransferAccountFundsResponse
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupAccountSubsection, okGroupFundsTransfer, request, &resp, true)
}

// AccountWithdraw withdrawal of tokens to OKCoin International, other OKEx accounts or other addresses.
func (o *OKGroup) AccountWithdraw(request AccountWithdrawRequest) (AccountWithdrawResponse, error) {
	var resp AccountWithdrawResponse
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupAccountSubsection, okGroupWithdraw, request, &resp, true)
}

// GetAccountWithdrawalFee retrieves the information about the recommended network transaction fee for withdrawals to digital asset addresses. The higher the fees are, the sooner the confirmations you will get.
func (o *OKGroup) GetAccountWithdrawalFee(currency string) ([]GetAccountWithdrawalFeeResponse, error) {
	var resp []GetAccountWithdrawalFeeResponse
	var requestURL string
	if len(currency) > 0 {
		requestURL = fmt.Sprintf("%v?currency=%v", okGroupGetWithdrawalFees, currency)
	} else {
		requestURL = okGroupGetAccountWalletInformation
	}

	return resp, o.SendHTTPRequest(http.MethodGet, okGroupAccountSubsection, requestURL, nil, &resp, true)
}

// GetAccountWithdrawalHistory retrieves all recent withdrawal records.
func (o *OKGroup) GetAccountWithdrawalHistory(currency string) ([]WithdrawalHistoryResponse, error) {
	var resp []WithdrawalHistoryResponse
	var requestURL string
	if len(currency) > 0 {
		requestURL = fmt.Sprintf("%v/%v", okGroupGetWithdrawalHistory, currency)
	} else {
		requestURL = okGroupGetWithdrawalHistory
	}
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupAccountSubsection, requestURL, nil, &resp, true)
}

// GetAccountBillDetails retrieves the bill details of the wallet. All the information will be paged and sorted in reverse chronological order,
// which means the latest will be at the top. Please refer to the pagination section for additional records after the first page.
// 3 months recent records will be returned at maximum
func (o *OKGroup) GetAccountBillDetails(request GetAccountBillDetailsRequest) ([]GetAccountBillDetailsResponse, error) {
	var resp []GetAccountBillDetailsResponse
	urlValues := url.Values{}
	if request.Type > 0 {
		urlValues.Set("type", strconv.FormatInt(request.Type, 10))
	}
	if len(request.Currency) > 0 {
		urlValues.Set("currency", request.Currency)
	}
	if request.From > 0 {
		urlValues.Set("from", strconv.FormatInt(request.From, 10))
	}
	if request.To > 0 {
		urlValues.Set("to", strconv.FormatInt(request.To, 10))
	}
	if request.Limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(request.Limit, 10))
	}

	parameters := ""
	urlEncodedValues := urlValues.Encode()
	if len(urlEncodedValues) > 0 {
		parameters = fmt.Sprintf("?%v", urlEncodedValues)
	}
	requestURL := fmt.Sprintf("%v%v", okGroupLedger, parameters)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupAccountSubsection, requestURL, nil, &resp, true)
}

// GetAccountDepositAddressForCurrency retrieves the deposit addresses of different tokens, including previously used addresses.
func (o *OKGroup) GetAccountDepositAddressForCurrency(currency string) ([]GetDepositAddressResponse, error) {
	var resp []GetDepositAddressResponse
	urlValues := url.Values{}
	urlValues.Set("currency", currency)
	requestURL := fmt.Sprintf("%v?%v", okGroupGetDepositAddress, urlValues.Encode())
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupAccountSubsection, requestURL, nil, &resp, true)
}

// GetAccountDepositHistory retrieves the deposit history of all tokens.100 recent records will be returned at maximum
func (o *OKGroup) GetAccountDepositHistory(currency string) ([]GetAccountDepositHistoryResponse, error) {
	var resp []GetAccountDepositHistoryResponse
	var requestURL string
	if len(currency) > 0 {
		requestURL = fmt.Sprintf("%v/%v", okGroupGetAccountDepositHistory, currency)
	} else {
		requestURL = okGroupGetWithdrawalHistory
	}
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupAccountSubsection, requestURL, nil, &resp, true)
}

// -------------------------------------Spot------------------------------------------

// GetSpotTradingAccounts retrieves the list of assets(only show pairs with balance larger than 0), the balances, amount available/on hold in spot accounts.
func (o *OKGroup) GetSpotTradingAccounts() ([]GetSpotTradingAccountResponse, error) {
	var resp []GetSpotTradingAccountResponse
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupTokenSubsection, okGroupTradingAccounts, nil, &resp, true)
}

// GetSpotTradingAccountForCurrency This endpoint supports getting the balance, amount available/on hold of a token in spot account.
func (o *OKGroup) GetSpotTradingAccountForCurrency(currency string) (GetSpotTradingAccountResponse, error) {
	var resp GetSpotTradingAccountResponse
	requestURL := fmt.Sprintf("%v/%v", okGroupTradingAccounts, currency)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, true)
}

// GetSpotBillDetailsForCurrency This endpoint supports getting the balance, amount available/on hold of a token in spot account.
func (o *OKGroup) GetSpotBillDetailsForCurrency(request GetSpotBillDetailsForCurrencyRequest) ([]GetSpotBillDetailsForCurrencyResponse, error) {
	var resp []GetSpotBillDetailsForCurrencyResponse
	urlValues := url.Values{}
	if request.From > 0 {
		urlValues.Set("from", strconv.FormatInt(request.From, 10))
	}
	if request.To > 0 {
		urlValues.Set("to", strconv.FormatInt(request.To, 10))
	}
	if request.Limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(request.Limit, 10))
	}

	requestURL := fmt.Sprintf("%v/%v/%v?%v", okGroupTradingAccounts, request.Currency, okGroupLedger, urlValues.Encode())
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, true)
}

// PlaceSpotOrder token trading only supports limit and market orders (more order types will become available in the future). You can place an order only if you have enough funds.
// Once your order is placed, the amount will be put on hold.
func (o *OKGroup) PlaceSpotOrder(request PlaceSpotOrderRequest) (PlaceSpotOrderResponse, error) {
	var resp PlaceSpotOrderResponse
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupTokenSubsection, okGroupOrders, request, &resp, true)
}

// PlaceMultipleSpotOrders supports placing multiple orders for specific trading pairs
// up to 4 trading pairs, maximum 4 orders for each pair
func (o *OKGroup) PlaceMultipleSpotOrders(request []PlaceSpotOrderRequest) (map[string][]PlaceSpotOrderResponse, []error) {
	currencyPairOrders := make(map[string]int)
	resp := make(map[string][]PlaceSpotOrderResponse)
	for _, order := range request {
		currencyPairOrders[order.InstrumentID]++
	}
	if len(currencyPairOrders) > 4 {
		return resp, []error{errors.New("up to 4 trading pairs")}
	}
	for _, orderCount := range currencyPairOrders {
		if orderCount > 4 {
			return resp, []error{errors.New("maximum 4 orders for each pair")}
		}
	}

	err := o.SendHTTPRequest(http.MethodPost, okGroupTokenSubsection, okGroupBatchOrders, request, &resp, true)
	if err != nil {
		return resp, []error{err}
	}

	orderErrors := []error{}
	for currency, orderResponse := range resp {
		for _, order := range orderResponse {
			if !order.Result {
				orderErrors = append(orderErrors, fmt.Errorf("Order for currency %v failed to be placed", currency))
			}
		}
	}
	if len(orderErrors) <= 0 {
		orderErrors = nil
	}

	return resp, orderErrors
}

// CancelSpotOrder Cancelling an unfilled order.
func (o *OKGroup) CancelSpotOrder(request CancelSpotOrderRequest) (CancelSpotOrderResponse, error) {
	var resp CancelSpotOrderResponse
	requestURL := fmt.Sprintf("%v/%v", okGroupCancelOrders, request.OrderID)
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupTokenSubsection, requestURL, request, &resp, true)
}

// CancelMultipleSpotOrders Cancelling multiple unfilled orders.
func (o *OKGroup) CancelMultipleSpotOrders(request CancelMultipleSpotOrdersRequest) (map[string][]CancelMultipleSpotOrdersResponse, []error) {
	resp := make(map[string][]CancelMultipleSpotOrdersResponse)
	if len(request.OrderIDs) > 4 {
		return resp, []error{errors.New("maximum 4 order cancellations for each pair")}
	}

	err := o.SendHTTPRequest(http.MethodPost, okGroupTokenSubsection, okGroupCancelBatchOrders, []CancelMultipleSpotOrdersRequest{request}, &resp, true)
	if err != nil {
		return resp, []error{err}
	}

	orderErrors := []error{}
	for currency, orderResponse := range resp {
		for _, order := range orderResponse {
			if !order.Result {
				orderErrors = append(orderErrors, fmt.Errorf("Order %v for currency %v failed to be cancelled", order.OrderID, currency))
			}
		}
	}
	if len(orderErrors) <= 0 {
		orderErrors = nil
	}

	return resp, orderErrors
}

// GetSpotOrders List your orders. Cursor pagination is used. All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetSpotOrders(request GetSpotOrdersRequest) ([]GetSpotOrderResponse, error) {
	var resp []GetSpotOrderResponse
	urlValues := url.Values{}
	urlValues.Set("instrument_id", request.InstrumentID)
	urlValues.Set("status", request.Status)
	if request.From > 0 {
		urlValues.Set("from", strconv.FormatInt(request.From, 10))
	}
	if request.To > 0 {
		urlValues.Set("to", strconv.FormatInt(request.To, 10))
	}
	if request.Limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(request.Limit, 10))
	}

	requestURL := fmt.Sprintf("%v?%v", okGroupOrders, urlValues.Encode())
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, true)
}

// GetSpotOpenOrders List all your current open orders. Cursor pagination is used. All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetSpotOpenOrders(request GetSpotOpenOrdersRequest) ([]GetSpotOrderResponse, error) {
	var resp []GetSpotOrderResponse
	urlValues := url.Values{}
	if len(request.InstrumentID) > 0 {
		urlValues.Set("instrument_id", request.InstrumentID)
	}
	if request.From > 0 {
		urlValues.Set("from", strconv.FormatInt(request.From, 10))
	}
	if request.To > 0 {
		urlValues.Set("to", strconv.FormatInt(request.To, 10))
	}
	if request.Limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(request.Limit, 10))
	}

	parameters := ""
	urlEncodedValues := urlValues.Encode()
	if len(urlEncodedValues) > 0 {
		parameters = fmt.Sprintf("?%v", urlEncodedValues)
	}
	requestURL := fmt.Sprintf("%v%v", okGroupPendingOrders, parameters)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, true)
}

// GetSpotOrder Get order details by order ID.
func (o *OKGroup) GetSpotOrder(request GetSpotOrderRequest) (GetSpotOrderResponse, error) {
	var resp GetSpotOrderResponse
	urlValues := url.Values{}
	urlValues.Set("instrument_id", request.InstrumentID)
	requestURL := fmt.Sprintf("%v/%v?%v", okGroupOrders, request.OrderID, urlValues.Encode())
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupTokenSubsection, requestURL, request, &resp, true)
}

// GetSpotTransactionDetails Get details of the recent filled orders. Cursor pagination is used. All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetSpotTransactionDetails(request GetSpotTransactionDetailsRequest) ([]GetSpotTransactionDetailsResponse, error) {
	var resp []GetSpotTransactionDetailsResponse
	urlValues := url.Values{}
	urlValues.Set("order_id", strconv.FormatInt(request.OrderID, 10))
	urlValues.Set("instrument_id", request.InstrumentID)
	if request.From > 0 {
		urlValues.Set("from", strconv.FormatInt(request.From, 10))
	}
	if request.To > 0 {
		urlValues.Set("to", strconv.FormatInt(request.To, 10))
	}
	if request.Limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(request.Limit, 10))
	}

	requestURL := fmt.Sprintf("%v?%v", okGroupGetSpotTransactionDetails, urlValues.Encode())
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, true)
}

// GetSpotTokenPairDetails Get market data. This endpoint provides the snapshots of market data and can be used without verifications.
// List trading pairs and get the trading limit, price, and more information of different trading pairs.
func (o *OKGroup) GetSpotTokenPairDetails() ([]GetSpotTokenPairDetailsResponse, error) {
	var resp []GetSpotTokenPairDetailsResponse
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupTokenSubsection, okGroupGetSpotTokenPairDetails, nil, &resp, true)
}

// GetSpotOrderBook Getting the order book of a trading pair. Pagination is not supported here. The whole book will be returned for one request. WebSocket is recommended here.
func (o *OKGroup) GetSpotOrderBook(request GetSpotOrderBookRequest) (GetSpotOrderBookResponse, error) {
	var resp GetSpotOrderBookResponse
	urlValues := url.Values{}
	if request.Depth > 0 {
		urlValues.Set("depth", fmt.Sprintf("%v", request.Depth))
	}
	if request.Size > 0 {
		urlValues.Set("size", strconv.FormatInt(request.Size, 10))
	}

	parameters := ""
	urlEncodedValues := urlValues.Encode()
	if len(urlEncodedValues) > 0 {
		parameters = fmt.Sprintf("?%v", urlEncodedValues)
	}

	requestURL := fmt.Sprintf("%v/%v/%v%v", okGroupGetSpotTokenPairDetails, request.InstrumentID, okGroupGetSpotOrderBook, parameters)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, true)
}

// GetSpotAllTokenPairsInformation Get the last traded price, best bid/ask price, 24 hour trading volume and more info of all trading pairs.
func (o *OKGroup) GetSpotAllTokenPairsInformation() ([]GetSpotTokenPairsInformationResponse, error) {
	var resp []GetSpotTokenPairsInformationResponse
	requestURL := fmt.Sprintf("%v/%v", okGroupGetSpotTokenPairDetails, okGroupTicker)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, true)
}

// GetSpotAllTokenPairsInformationForCurrency Get the last traded price, best bid/ask price, 24 hour trading volume and more info of a currency
func (o *OKGroup) GetSpotAllTokenPairsInformationForCurrency(currency string) (GetSpotTokenPairsInformationResponse, error) {
	var resp GetSpotTokenPairsInformationResponse
	requestURL := fmt.Sprintf("%v/%v/%v", okGroupGetSpotTokenPairDetails, currency, okGroupTicker)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, true)
}

// GetSpotFilledOrdersInformation Get the recent 60 transactions of all trading pairs. Cursor pagination is used. All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetSpotFilledOrdersInformation(request GetSpotFilledOrdersInformationRequest) ([]GetSpotFilledOrdersInformationResponse, error) {
	var resp []GetSpotFilledOrdersInformationResponse
	urlValues := url.Values{}
	if request.From > 0 {
		urlValues.Set("from", strconv.FormatInt(request.From, 10))
	}
	if request.To > 0 {
		urlValues.Set("to", strconv.FormatInt(request.To, 10))
	}
	if request.Limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(request.Limit, 10))
	}

	parameters := ""
	urlEncodedValues := urlValues.Encode()
	if len(urlEncodedValues) > 0 {
		parameters = fmt.Sprintf("?%v", urlEncodedValues)
	}
	requestURL := fmt.Sprintf("%v/%v/%v%v", okGroupGetSpotTokenPairDetails, request.InstrumentID, okGroupTrades, parameters)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, true)
}

// GetSpotMarketData Get the charts of the trading pairs. Charts are returned in grouped buckets based on requested granularity.
func (o *OKGroup) GetSpotMarketData(request GetSpotMarketDataRequest) (GetSpotMarketDataResponse, error) {
	var resp GetSpotMarketDataResponse
	urlValues := url.Values{}
	if len(request.Start) > 0 {
		urlValues.Set("start", request.Start)
	}
	if len(request.End) > 0 {
		urlValues.Set("end", request.End)
	}
	if request.Granularity > 0 {
		urlValues.Set("granularity", strconv.FormatInt(request.Granularity, 10))
	}

	parameters := ""
	urlEncodedValues := urlValues.Encode()
	if len(urlEncodedValues) > 0 {
		parameters = fmt.Sprintf("?%v", urlEncodedValues)
	}
	requestURL := fmt.Sprintf("%v/%v/%v%v", okGroupGetSpotTokenPairDetails, request.InstrumentID, okGroupGetSpotMarketData, parameters)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, true)
}

// -------------------------------------Margin------------------------------------------

// GetMarginTradingAccounts List all assets under token margin trading account, including information such as balance, amount on hold and more.
func (o *OKGroup) GetMarginTradingAccounts() ([]GetMarginAccountsResponse, error) {
	var resp []GetMarginAccountsResponse
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupMarginTradingSubsection, okGroupTradingAccounts, nil, &resp, true)
}

// GetMarginTradingAccountsForCurrency Get the balance, amount on hold and more useful information.
func (o *OKGroup) GetMarginTradingAccountsForCurrency(currency string) (GetMarginAccountsResponse, error) {
	var resp GetMarginAccountsResponse
	requestURL := fmt.Sprintf("%v/%v", okGroupTradingAccounts, currency)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupMarginTradingSubsection, requestURL, nil, &resp, true)
}

// GetMarginBillDetails List all bill details. Pagination is used here. before and after cursor arguments should not be confused with before and after in chronological time.
// Most paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetMarginBillDetails(request GetAccountBillDetailsRequest) ([]GetSpotBillDetailsForCurrencyResponse, error) {
	var resp []GetSpotBillDetailsForCurrencyResponse
	urlValues := url.Values{}
	if request.Type > 0 {
		urlValues.Set("type", strconv.FormatInt(request.Type, 10))
	}
	if len(request.Currency) > 0 {
		urlValues.Set("currency", request.Currency)
	}
	if request.From > 0 {
		urlValues.Set("from", strconv.FormatInt(request.From, 10))
	}
	if request.To > 0 {
		urlValues.Set("to", strconv.FormatInt(request.To, 10))
	}
	if request.Limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(request.Limit, 10))
	}

	parameters := ""
	urlEncodedValues := urlValues.Encode()
	if len(urlEncodedValues) > 0 {
		parameters = fmt.Sprintf("?%v", urlEncodedValues)
	}
	requestURL := fmt.Sprintf("%v/%v/%v%v", okGroupTradingAccounts, request.Currency, okGroupLedger, parameters)

	return resp, o.SendHTTPRequest(http.MethodGet, okGroupMarginTradingSubsection, requestURL, nil, &resp, true)
}

// GetMarginAccountSettings Get all information of the margin trading account, including the maximum loan amount, interest rate, and maximum leverage.
func (o *OKGroup) GetMarginAccountSettings(currency string) ([]GetMarginAccountSettingsResponse, error) {
	var resp []GetMarginAccountSettingsResponse
	var requestURL string
	if len(currency) > 0 {
		requestURL = fmt.Sprintf("%v/%v/%v", okGroupTradingAccounts, currency, okGroupGetMarketAvailability)
	} else {
		requestURL = fmt.Sprintf("%v/%v", okGroupTradingAccounts, okGroupGetMarketAvailability)
	}
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupMarginTradingSubsection, requestURL, nil, &resp, true)
}

// GetMarginLoanHistory Get loan history of the margin trading account. Pagination is used here. before and after cursor arguments should not be confused with before and after in chronological time.
// Most paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetMarginLoanHistory(request GetMarginLoanHistoryRequest) ([]GetMarginLoanHistoryResponse, error) {
	var resp []GetMarginLoanHistoryResponse
	var requestURL string
	if len(request.InstrumentID) > 0 {
		requestURL = fmt.Sprintf("%v/%v/%v", okGroupTradingAccounts, request.InstrumentID, okGroupGetLoan)
	} else {
		requestURL = fmt.Sprintf("%v/%v", okGroupTradingAccounts, okGroupGetLoan)
	}
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupMarginTradingSubsection, requestURL, nil, &resp, true)
}

// OpenMarginLoan Borrowing tokens in a margin trading account.
func (o *OKGroup) OpenMarginLoan(request OpenMarginLoanRequest) (OpenMarginLoanResponse, error) {
	var resp OpenMarginLoanResponse
	requestURL := fmt.Sprintf("%v/%v", okGroupTradingAccounts, okGroupGetLoan)
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupMarginTradingSubsection, requestURL, request, &resp, true)
}

// RepayMarginLoan Repaying tokens in a margin trading account.
func (o *OKGroup) RepayMarginLoan(request RepayMarginLoanRequest) (RepayMarginLoanResponse, error) {
	var resp RepayMarginLoanResponse
	requestURL := fmt.Sprintf("%v/%v", okGroupTradingAccounts, okGroupGetRepayment)
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupMarginTradingSubsection, requestURL, request, &resp, true)
}

// PlaceMarginOrder OKEx API only supports limit and market orders (more orders will become available in the future).
// You can place an order only if you have enough funds. Once your order is placed, the amount will be put on hold.
func (o *OKGroup) PlaceMarginOrder(request PlaceSpotOrderRequest) (PlaceSpotOrderResponse, error) {
	var resp PlaceSpotOrderResponse
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupMarginTradingSubsection, okGroupOrders, request, &resp, true)
}

// PlaceMultipleMarginOrders Place multiple orders for specific trading pairs (up to 4 trading pairs, maximum 4 orders each)
func (o *OKGroup) PlaceMultipleMarginOrders(request []PlaceSpotOrderRequest) (map[string][]PlaceSpotOrderResponse, []error) {
	currencyPairOrders := make(map[string]int)
	resp := make(map[string][]PlaceSpotOrderResponse)
	for _, order := range request {
		currencyPairOrders[order.InstrumentID]++
	}
	if len(currencyPairOrders) > 4 {
		return resp, []error{errors.New("up to 4 trading pairs")}
	}
	for _, orderCount := range currencyPairOrders {
		if orderCount > 4 {
			return resp, []error{errors.New("maximum 4 orders for each pair")}
		}
	}

	err := o.SendHTTPRequest(http.MethodPost, okGroupMarginTradingSubsection, okGroupBatchOrders, request, &resp, true)
	if err != nil {
		return resp, []error{err}
	}

	orderErrors := []error{}
	for currency, orderResponse := range resp {
		for _, order := range orderResponse {
			if !order.Result {
				orderErrors = append(orderErrors, fmt.Errorf("Order for currency %v failed to be placed", currency))
			}
		}
	}
	if len(orderErrors) <= 0 {
		orderErrors = nil
	}

	return resp, orderErrors
}

// CancelMarginOrder Cancelling an unfilled order.
func (o *OKGroup) CancelMarginOrder(request CancelSpotOrderRequest) (CancelSpotOrderResponse, error) {
	var resp CancelSpotOrderResponse
	requestURL := fmt.Sprintf("%v/%v", okGroupCancelOrders, request.OrderID)
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupMarginTradingSubsection, requestURL, request, &resp, true)
}

// CancelMultipleMarginOrders Cancelling multiple unfilled orders.
func (o *OKGroup) CancelMultipleMarginOrders(request CancelMultipleSpotOrdersRequest) (map[string][]CancelMultipleSpotOrdersResponse, []error) {
	resp := make(map[string][]CancelMultipleSpotOrdersResponse)
	if len(request.OrderIDs) > 4 {
		return resp, []error{errors.New("maximum 4 order cancellations for each pair")}
	}

	err := o.SendHTTPRequest(http.MethodPost, okGroupMarginTradingSubsection, okGroupCancelBatchOrders, []CancelMultipleSpotOrdersRequest{request}, &resp, true)
	if err != nil {
		return resp, []error{err}
	}

	orderErrors := []error{}
	for currency, orderResponse := range resp {
		for _, order := range orderResponse {
			if !order.Result {
				orderErrors = append(orderErrors, fmt.Errorf("Order %v for currency %v failed to be cancelled", order.OrderID, currency))
			}
		}
	}
	if len(orderErrors) <= 0 {
		orderErrors = nil
	}

	return resp, orderErrors
}

// GetMarginOrders List your orders. Cursor pagination is used. All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetMarginOrders(request GetSpotOrdersRequest) ([]GetSpotOrderResponse, error) {
	var resp []GetSpotOrderResponse
	urlValues := url.Values{}
	urlValues.Set("instrument_id", request.InstrumentID)
	urlValues.Set("status", request.Status)
	if request.From > 0 {
		urlValues.Set("from", strconv.FormatInt(request.From, 10))
	}
	if request.To > 0 {
		urlValues.Set("to", strconv.FormatInt(request.To, 10))
	}
	if request.Limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(request.Limit, 10))
	}

	requestURL := fmt.Sprintf("%v?%v", okGroupOrders, urlValues.Encode())
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupMarginTradingSubsection, requestURL, nil, &resp, true)
}

// GetMarginOpenOrders List all your current open orders. Cursor pagination is used. All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetMarginOpenOrders(request GetSpotOpenOrdersRequest) ([]GetSpotOrderResponse, error) {
	var resp []GetSpotOrderResponse
	urlValues := url.Values{}
	if len(request.InstrumentID) > 0 {
		urlValues.Set("instrument_id", request.InstrumentID)
	}
	if request.From > 0 {
		urlValues.Set("from", strconv.FormatInt(request.From, 10))
	}
	if request.To > 0 {
		urlValues.Set("to", strconv.FormatInt(request.To, 10))
	}
	if request.Limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(request.Limit, 10))
	}

	parameters := ""
	urlEncodedValues := urlValues.Encode()
	if len(urlEncodedValues) > 0 {
		parameters = fmt.Sprintf("?%v", urlEncodedValues)
	}
	requestURL := fmt.Sprintf("%v%v", okGroupPendingOrders, parameters)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupMarginTradingSubsection, requestURL, nil, &resp, true)
}

// GetMarginOrder Get order details by order ID.
func (o *OKGroup) GetMarginOrder(request GetSpotOrderRequest) (GetSpotOrderResponse, error) {
	var resp GetSpotOrderResponse
	urlValues := url.Values{}
	urlValues.Set("instrument_id", request.InstrumentID)
	requestURL := fmt.Sprintf("%v/%v?%v", okGroupOrders, request.OrderID, urlValues.Encode())
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupMarginTradingSubsection, requestURL, request, &resp, true)
}

// GetMarginTransactionDetails Get details of the recent filled orders. Cursor pagination is used. All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetMarginTransactionDetails(request GetSpotTransactionDetailsRequest) ([]GetSpotTransactionDetailsResponse, error) {
	var resp []GetSpotTransactionDetailsResponse
	urlValues := url.Values{}
	urlValues.Set("order_id", strconv.FormatInt(request.OrderID, 10))
	urlValues.Set("instrument_id", request.InstrumentID)
	if request.From > 0 {
		urlValues.Set("from", strconv.FormatInt(request.From, 10))
	}
	if request.To > 0 {
		urlValues.Set("to", strconv.FormatInt(request.To, 10))
	}
	if request.Limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(request.Limit, 10))
	}

	requestURL := fmt.Sprintf("%v?%v", okGroupGetSpotTransactionDetails, urlValues.Encode())
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupMarginTradingSubsection, requestURL, nil, &resp, true)
}

// -------------------------------------Futures------------------------------------------

// GetFuturesPostions Get the information of all holding positions in futures trading.Due to high energy consumption, you are advised to capture data with the "Futures Account of a Currency" API instead.
func (o *OKGroup) GetFuturesPostions() (GetFuturesPositionsResponse, error) {
	var resp GetFuturesPositionsResponse
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, okGroupFuturePosition, nil, &resp, true)
}

// GetFuturesPostionsForCurrency Get the information of holding positions of a contract.
func (o *OKGroup) GetFuturesPostionsForCurrency(instrumentID string) (GetFuturesPositionsForCurrencyResponse, error) {
	var resp GetFuturesPositionsForCurrencyResponse
	requestURL := fmt.Sprintf("%v/%v", instrumentID, okGroupFuturePosition)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesAccountOfAllCurrencies Get the futures account info of all token.Due to high energy consumption, you are advised to capture data with the "Futures Account of a Currency" API instead.
func (o *OKGroup) GetFuturesAccountOfAllCurrencies() (FuturesAccountForAllCurrenciesResponse, error) {
	var resp FuturesAccountForAllCurrenciesResponse
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, okGroupTradingAccounts, nil, &resp, true)
}

// GetFuturesAccountOfACurrency Get the futures account info of a token.
func (o *OKGroup) GetFuturesAccountOfACurrency(instrumentID string) (FuturesCurrencyData, error) {
	var resp FuturesCurrencyData
	requestURL := fmt.Sprintf("%v/%v", okGroupTradingAccounts, instrumentID)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesLeverage Get the leverage of the futures account
func (o *OKGroup) GetFuturesLeverage(instrumentID string) (GetFuturesLeverageResponse, error) {
	var resp GetFuturesLeverageResponse
	requestURL := fmt.Sprintf("%v/%v/%v", okGroupTradingAccounts, instrumentID, okGroupFutureLeverage)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// SetFuturesLeverage Adjusting the leverage for futures account。
// Cross margin request requirements:  {"leverage":"10"}
// Fixed margin request requirements: {"instrument_id":"BTC-USD-180213","direction":"long","leverage":"10"}
func (o *OKGroup) SetFuturesLeverage(request SetFuturesLeverageRequest) (SetFuturesLeverageResponse, error) {
	var resp SetFuturesLeverageResponse
	requestURL := fmt.Sprintf("%v/%v/%v", okGroupTradingAccounts, request.Currency, okGroupFutureLeverage)
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupFuturesSubsection, requestURL, request, &resp, true)
}

// GetFuturesBillDetails Shows the account’s historical coin in flow and out flow.
// All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetFuturesBillDetails(request GetSpotBillDetailsForCurrencyRequest) ([]GetSpotBillDetailsForCurrencyResponse, error) {
	var resp []GetSpotBillDetailsForCurrencyResponse
	urlValues := url.Values{}
	if request.From > 0 {
		urlValues.Set("from", strconv.FormatInt(request.From, 10))
	}
	if request.To > 0 {
		urlValues.Set("to", strconv.FormatInt(request.To, 10))
	}
	if request.Limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(request.Limit, 10))
	}

	parameters := ""
	urlEncodedValues := urlValues.Encode()
	if len(urlEncodedValues) > 0 {
		parameters = fmt.Sprintf("?%v", urlEncodedValues)
	}
	requestURL := fmt.Sprintf("%v/%v/%v%v", okGroupTradingAccounts, request.Currency, okGroupLedger, parameters)

	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// PlaceFuturesOrder OKEx futures trading only supports limit orders.
// You can place an order only if you have enough funds. Once your order is placed, the amount will be put on hold in the order lifecycle.
// The assets and amount on hold depends on the order's specific type and parameters.
func (o *OKGroup) PlaceFuturesOrder(request PlaceFuturesOrderRequest) (PlaceFuturesOrderResponse, error) {
	var resp PlaceFuturesOrderResponse
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupFuturesSubsection, okGroupFutureOrder, request, &resp, true)
}

// PlaceFuturesOrderBatch Batch contract placing order operation.
func (o *OKGroup) PlaceFuturesOrderBatch(request PlaceFuturesOrderBatchRequest) (PlaceFuturesOrderBatchResponse, error) {
	var resp PlaceFuturesOrderBatchResponse
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupFuturesSubsection, okGroupOrders, request, &resp, true)
}

// CancelFuturesOrder Cancelling an unfilled order.
func (o *OKGroup) CancelFuturesOrder(request CancelFuturesOrderRequest) (CancelFuturesOrderResponse, error) {
	var resp CancelFuturesOrderResponse
	requestURL := fmt.Sprintf("%v/%v/%v", okGroupCancelOrder, request.InstrumentID, request.OrderID)
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupFuturesSubsection, requestURL, request, &resp, true)
}

// CancelFuturesOrderBatch With best effort, cancel all open orders.
func (o *OKGroup) CancelFuturesOrderBatch(request CancelMultipleSpotOrdersRequest) (CancelMultipleSpotOrdersResponse, error) {
	var resp CancelMultipleSpotOrdersResponse
	requestURL := fmt.Sprintf("%v/%v", okGroupCancelBatchOrders, request.InstrumentID)
	return resp, o.SendHTTPRequest(http.MethodPost, okGroupFuturesSubsection, requestURL, request, &resp, true)
}

// GetFuturesOrderList List your orders. Cursor pagination is used.
// All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetFuturesOrderList(request GetFuturesOrdersListRequest) (GetFuturesOrderListResponse, error) {
	var resp GetFuturesOrderListResponse
	urlValues := url.Values{}
	urlValues.Set("status", strconv.FormatInt(request.Status, 10))
	if request.From > 0 {
		urlValues.Set("from", strconv.FormatInt(request.From, 10))
	}
	if request.To > 0 {
		urlValues.Set("to", strconv.FormatInt(request.To, 10))
	}
	if request.Limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(request.Limit, 10))
	}

	requestURL := fmt.Sprintf("%v/%v?%v", okGroupOrders, request.InstrumentID, urlValues.Encode())
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesOrderDetails Get order details by order ID.
func (o *OKGroup) GetFuturesOrderDetails(request GetFuturesOrderDetailsRequest) (resp GetFuturesOrderDetailsResponseData, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okGroupOrders, request.InstrumentID, request.OrderID)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesTransactionDetails  Get details of the recent filled orders. Cursor pagination is used. All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetFuturesTransactionDetails(request GetFuturesTransactionDetailsRequest) ([]GetFuturesTransactionDetailsResponse, error) {
	var resp []GetFuturesTransactionDetailsResponse
	urlValues := url.Values{}
	urlValues.Set("order_id", strconv.FormatInt(request.OrderID, 10))
	urlValues.Set("instrument_id", request.InstrumentID)
	if request.From > 0 {
		urlValues.Set("from", strconv.FormatInt(request.From, 10))
	}
	if request.To > 0 {
		urlValues.Set("to", strconv.FormatInt(request.To, 10))
	}
	if request.Limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(request.Limit, 10))
	}

	requestURL := fmt.Sprintf("%v?%v", okGroupGetSpotTransactionDetails, urlValues.Encode())
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesContractInformation Get market data. This endpoint provides the snapshots of market data and can be used without verifications.
func (o *OKGroup) GetFuturesContractInformation() ([]GetFuturesContractInformationResponse, error) {
	var resp []GetFuturesContractInformationResponse
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, okGroupGetSpotTokenPairDetails, nil, &resp, false)
}

// GetFuturesOrderBook List all contracts. This request does not support pagination. The full list will be returned for a request.
func (o *OKGroup) GetFuturesOrderBook(request GetFuturesOrderBookRequest) (GetFuturesOrderBookResponse, error) {
	var resp GetFuturesOrderBookResponse
	urlValues := url.Values{}
	if request.Size > 0 {
		urlValues.Set("size", strconv.FormatInt(request.Size, 10))
	}
	urlValues.Set("instrument_id", request.InstrumentID)

	parameters := ""
	urlEncodedValues := urlValues.Encode()
	if len(urlEncodedValues) > 0 {
		parameters = fmt.Sprintf("?%v", urlEncodedValues)
	}
	requestURL := fmt.Sprintf("%v/%v/%v%v", okGroupGetSpotTokenPairDetails, request.InstrumentID, okGroupGetSpotOrderBook, parameters)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetAllFuturesTokenInfo Get the last traded price, best bid/ask price, 24 hour trading volume and more info of all contracts.
func (o *OKGroup) GetAllFuturesTokenInfo() (resp []GetFuturesTokenInfoResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", okGroupGetSpotTokenPairDetails, okGroupTicker)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesTokenInfoForCurrency Get the last traded price, best bid/ask price, 24 hour trading volume and more info of a contract.
func (o *OKGroup) GetFuturesTokenInfoForCurrency(instrumentID string) (resp GetFuturesTokenInfoResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okGroupGetSpotTokenPairDetails, instrumentID, okGroupTicker)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesFilledOrder Get the recent 300 transactions of all contracts. Pagination is not supported here.
// The whole book will be returned for one request. WebSocket is recommended here.
func (o *OKGroup) GetFuturesFilledOrder(request GetFuturesFilledOrderRequest) (resp []GetFuturesFilledOrdersResponse, _ error) {
	urlValues := url.Values{}
	if request.From > 0 {
		urlValues.Set("from", strconv.FormatInt(request.From, 10))
	}
	if request.To > 0 {
		urlValues.Set("to", strconv.FormatInt(request.To, 10))
	}
	if request.Limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(request.Limit, 10))
	}

	parameters := ""
	urlEncodedValues := urlValues.Encode()
	if len(urlEncodedValues) > 0 {
		parameters = fmt.Sprintf("?%v", urlEncodedValues)
	}
	requestURL := fmt.Sprintf("%v/%v/%v%v", okGroupGetSpotTokenPairDetails, request.InstrumentID, okGroupTrades, parameters)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesMarketData Get the charts of the trading pairs. Charts are returned in grouped buckets based on requested granularity.
func (o *OKGroup) GetFuturesMarketData(request GetFuturesMarketDateRequest) (resp GetFuturesMarketDataResponse, _ error) {
	urlValues := url.Values{}
	if len(request.Start) > 0 {
		urlValues.Set("start", request.Start)
	}
	if len(request.End) > 0 {
		urlValues.Set("end", request.End)
	}
	if request.Granularity > 0 {
		urlValues.Set("granularity", strconv.FormatInt(request.Granularity, 10))
	}

	parameters := ""
	urlEncodedValues := urlValues.Encode()
	if len(urlEncodedValues) > 0 {
		parameters = fmt.Sprintf("?%v", urlEncodedValues)
	}
	requestURL := fmt.Sprintf("%v/%v/%v%v", okGroupGetSpotTokenPairDetails, request.InstrumentID, okGroupGetSpotMarketData, parameters)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesHoldAmount Get the number of futures with hold.
func (o *OKGroup) GetFuturesHoldAmount(instrumentID string) (resp GetFuturesHoldAmountResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okGroupTradingAccounts, instrumentID, okGroupFutureHolds)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, true)
}

// GetFuturesIndices Get Indices of tokens. This is a public endpoint, no identity verification is needed.
func (o *OKGroup) GetFuturesIndices(instrumentID string) (resp GetFuturesIndicesResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okGroupGetSpotTokenPairDetails, instrumentID, okGroupIndices)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, false)
}

// GetFuturesExchangeRates Get the fiat exchange rates. This is a public endpoint, no identity verification is needed.
func (o *OKGroup) GetFuturesExchangeRates() (resp GetFuturesExchangeRatesResponse, _ error) {
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, okGroupRate, nil, &resp, false)
}

// GetFuturesEstimatedDeliveryPrice the estimated delivery price. It is available 3 hours before delivery.
// This is a public endpoint, no identity verification is needed.
func (o *OKGroup) GetFuturesEstimatedDeliveryPrice(instrumentID string) (resp GetFuturesEstimatedDeliveryPriceResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okGroupGetSpotTokenPairDetails, instrumentID, okGroupEsimtatedPrice)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, false)
}

// GetFuturesOpenInterests Get the open interest of a contract. This is a public endpoint, no identity verification is needed.
func (o *OKGroup) GetFuturesOpenInterests(instrumentID string) (resp GetFuturesOpenInterestsResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okGroupGetSpotTokenPairDetails, instrumentID, okGroupOpenInterest)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, false)
}

// GetFuturesCurrentPriceLimit The maximum buying price and the minimum selling price of the contract.
// This is a public endpoint, no identity verification is needed.
func (o *OKGroup) GetFuturesCurrentPriceLimit(instrumentID string) (resp GetFuturesCurrentPriceLimitResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okGroupGetSpotTokenPairDetails, instrumentID, okGroupPriceLimit)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, false)
}

// GetFuturesCurrentMarkPrice The maximum buying price and the minimum selling price of the contract.
// This is a public endpoint, no identity verification is needed.
func (o *OKGroup) GetFuturesCurrentMarkPrice(instrumentID string) (resp GetFuturesCurrentMarkPriceResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", okGroupGetSpotTokenPairDetails, instrumentID, okGroupMarkPrice)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, false)
}

// GetFuturesForceLiquidatedOrders Get force liquidated orders. This is a public endpoint, no identity verification is needed.
func (o *OKGroup) GetFuturesForceLiquidatedOrders(request GetFuturesForceLiquidatedOrdersRequest) (resp []GetFuturesForceLiquidatedOrdersResponse, _ error) {
	urlValues := url.Values{}
	if len(request.Status) > 0 {
		urlValues.Set("status", request.Status)
	}
	if request.From > 0 {
		urlValues.Set("from", strconv.FormatInt(request.From, 10))
	}
	if request.To > 0 {
		urlValues.Set("to", strconv.FormatInt(request.To, 10))
	}
	if request.Limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(request.Limit, 10))
	}

	parameters := ""
	urlEncodedValues := urlValues.Encode()
	if len(urlEncodedValues) > 0 {
		parameters = fmt.Sprintf("?%v", urlEncodedValues)
	}
	requestURL := fmt.Sprintf("%v/%v/%v%v", okGroupGetSpotTokenPairDetails, request.InstrumentID, okGroupLiquidation, parameters)
	return resp, o.SendHTTPRequest(http.MethodGet, okGroupFuturesSubsection, requestURL, nil, &resp, false)
}

// GetFuturesTagPrice Get the tag price. This is a public endpoint, no identity verification is needed.
func (o *OKGroup) GetFuturesTagPrice(instrumentID string) (resp GetFuturesTagPriceResponse, _ error) {
	// OKEX documentation is missing for this endpoint. Guessing "tag_price" for the URL results in 404
	return GetFuturesTagPriceResponse{}, common.ErrNotYetImplemented
}

// -------------------------------------------------------------------------------------------------------

// GetErrorCode returns an error code
func (o *OKGroup) GetErrorCode(code interface{}) error {
	var assertedCode string

	switch reflect.TypeOf(code).String() {
	case "float64":
		assertedCode = strconv.FormatFloat(code.(float64), 'f', -1, 64)
	case "string":
		assertedCode = code.(string)
	default:
		return errors.New("unusual type returned")
	}

	if i, ok := o.ErrorCodes[assertedCode]; ok {
		return i
	}
	return errors.New("unable to find SPOT error code")
}

// SendHTTPRequest sends an authenticated http request to a desired
// path with a JSON payload (of present)
// URL arguments must be in the request path and not as url.URL values
func (o *OKGroup) SendHTTPRequest(httpMethod, requestType, requestPath string, data interface{}, result interface{}, authenticated bool) (err error) {
	if authenticated && !o.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, o.Name)
	}

	utcTime := time.Now().UTC()
	iso := utcTime.String()
	isoBytes := []byte(iso)
	iso = string(isoBytes[:10]) + "T" + string(isoBytes[11:23]) + "Z"
	payload := []byte("")

	if data != nil {
		payload, err = common.JSONEncode(data)
		if err != nil {
			return errors.New("SendHTTPRequest: Unable to JSON request")
		}

		if o.Verbose {
			log.Debugf("Request JSON: %s\n", payload)
		}
	}

	path := o.APIUrl + requestType + o.APIVersion + requestPath
	if o.Verbose {
		log.Debugf("Sending %v request to %s with params \n", requestType, path)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	if authenticated {
		signPath := fmt.Sprintf("/%v%v%v%v", OkGroupAPIPath, requestType, o.APIVersion, requestPath)
		hmac := common.GetHMAC(common.HashSHA256, []byte(iso+httpMethod+signPath+string(payload)), []byte(o.APISecret))
		base64 := common.Base64Encode(hmac)
		headers["OK-ACCESS-KEY"] = o.APIKey
		headers["OK-ACCESS-SIGN"] = base64
		headers["OK-ACCESS-TIMESTAMP"] = iso
		headers["OK-ACCESS-PASSPHRASE"] = o.ClientID
	}

	var intermediary json.RawMessage
	type errCapFormat struct {
		Error        int64  `json:"error_code,omitempty"`
		Code         int64  `json:"code,omitempty"`
		ErrorMessage string `json:"error_message,omitempty"`
		Result       bool   `json:"result,omitempty"`
	}

	errCap := errCapFormat{}
	errCap.Result = true
	err = o.SendPayload(strings.ToUpper(httpMethod), path, headers, bytes.NewBuffer(payload), &intermediary, authenticated, o.Verbose)
	if err != nil {
		return err
	}

	err = common.JSONDecode(intermediary, &errCap)
	if err == nil {
		if len(errCap.ErrorMessage) > 0 {
			return fmt.Errorf("Error: %v", errCap.ErrorMessage)
		}
		if errCap.Error > 0 {
			return fmt.Errorf("SendHTTPRequest error - %s",
				o.ErrorCodes[strconv.FormatInt(errCap.Error, 10)])
		}
		if !errCap.Result {
			return errors.New("Unspecified error occured")
		}
	}

	return common.JSONDecode(intermediary, result)
}

// SetCheckVarDefaults sets main variables that will be used in requests because
// api does not return an error if there are misspellings in strings. So better
// to check on this, this end.
func (o *OKGroup) SetCheckVarDefaults() {
	o.ContractTypes = []string{"this_week", "next_week", "quarter"}
	o.CurrencyPairs = []string{"btc_usd", "ltc_usd", "eth_usd", "etc_usd", "bch_usd"}
	o.Types = []string{"1min", "3min", "5min", "15min", "30min", "1day", "3day",
		"1week", "1hour", "2hour", "4hour", "6hour", "12hour"}
	o.ContractPosition = []string{"1", "2", "3", "4"}
}

// CheckContractPosition checks to see if the string is a valid position for OKGroup
func (o *OKGroup) CheckContractPosition(position string) error {
	if !common.StringDataCompare(o.ContractPosition, position) {
		return errors.New("invalid position string - e.g. 1 = open long position, 2 = open short position, 3 = liquidate long position, 4 = liquidate short position")
	}
	return nil
}

// CheckSymbol checks to see if the string is a valid symbol for OKGroup
func (o *OKGroup) CheckSymbol(symbol string) error {
	if !common.StringDataCompare(o.CurrencyPairs, symbol) {
		return errors.New("invalid symbol string")
	}
	return nil
}

// CheckContractType checks to see if the string is a correct asset
func (o *OKGroup) CheckContractType(contractType string) error {
	if !common.StringDataCompare(o.ContractTypes, contractType) {
		return errors.New("invalid contract type string")
	}
	return nil
}

// CheckType checks to see if the string is a correct type
func (o *OKGroup) CheckType(typeInput string) error {
	if !common.StringDataCompare(o.Types, typeInput) {
		return errors.New("invalid type string")
	}
	return nil
}

// GetFee returns an estimate of fee based on type of transaction
func (o *OKGroup) GetFee(feeBuilder exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount, feeBuilder.IsMaker)
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.FirstCurrency)
	}
	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

func calculateTradingFee(purchasePrice, amount float64, isMaker bool) (fee float64) {
	// TODO volume based fees
	if isMaker {
		fee = 0.001
	} else {
		fee = 0.0015
	}
	return fee * amount * purchasePrice
}

func getWithdrawalFee(currency string) float64 {
	return WithdrawalFees[currency]
}

// SetErrorDefaults sets the full error default list
func (o *OKGroup) SetErrorDefaults() {
	o.ErrorCodes = map[string]error{
		"0":     errors.New("successful"),
		"1":     errors.New("Invalid parameter in url normally"),
		"30001": errors.New("request header \"OK_ACCESS_KEY\" cannot be blank"),
		"30002": errors.New("request header \"OK_ACCESS_SIGN\" cannot be blank"),
		"30003": errors.New("request header \"OK_ACCESS_TIMESTAMP\" cannot be blank"),
		"30004": errors.New("request header \"OK_ACCESS_PASSPHRASE\" cannot be blank"),
		"30005": errors.New("invalid OK_ACCESS_TIMESTAMP"),
		"30006": errors.New("invalid OK_ACCESS_KEY"),
		"30007": errors.New("invalid Content_Type, please use \"application/json\" format"),
		"30008": errors.New("timestamp request expired"),
		"30009": errors.New("system error"),
		"30010": errors.New("API validation failed"),
		"30011": errors.New("invalid IP"),
		"30012": errors.New("invalid authorization"),
		"30013": errors.New("invalid sign"),
		"30014": errors.New("request too frequent"),
		"30015": errors.New("request header \"OK_ACCESS_PASSPHRASE\" incorrect"),
		"30016": errors.New("you are using v1 apiKey, please use v1 endpoint. If you would like to use v3 endpoint, please subscribe to v3 apiKey"),
		"30017": errors.New("apikey's broker id does not match"),
		"30018": errors.New("apikey's domain does not match"),
		"30020": errors.New("body cannot be blank"),
		"30021": errors.New("json data format error"),
		"30023": errors.New("required parameter cannot be blank"),
		"30024": errors.New("parameter value error"),
		"30025": errors.New("parameter category error"),
		"30026": errors.New("requested too frequent; endpoint limit exceeded"),
		"30027": errors.New("login failure"),
		"30028": errors.New("unauthorized execution"),
		"30029": errors.New("account suspended"),
		"30030": errors.New("endpoint request failed. Please try again"),
		"30031": errors.New("token does not exist"),
		"30032": errors.New("pair does not exist"),
		"30033": errors.New("exchange domain does not exist"),
		"30034": errors.New("exchange ID does not exist"),
		"30035": errors.New("trading is not supported in this website"),
		"30036": errors.New("no relevant data"),
		"30037": errors.New("endpoint is offline or unavailable"),
		"30038": errors.New("user does not exist"),
		"32001": errors.New("futures account suspended"),
		"32002": errors.New("futures account does not exist"),
		"32003": errors.New("canceling, please wait"),
		"32004": errors.New("you have no unfilled orders"),
		"32005": errors.New("max order quantity"),
		"32006": errors.New("the order price or trigger price exceeds USD 1 million"),
		"32007": errors.New("leverage level must be the same for orders on the same side of the contract"),
		"32008": errors.New("Max. positions to open (cross margin)"),
		"32009": errors.New("Max. positions to open (fixed margin)"),
		"32010": errors.New("leverage cannot be changed with open positions"),
		"32011": errors.New("futures status error"),
		"32012": errors.New("futures order update error"),
		"32013": errors.New("token type is blank"),
		"32014": errors.New("your number of contracts closing is larger than the number of contracts available"),
		"32015": errors.New("margin ratio is lower than 100% before opening positions"),
		"32016": errors.New("margin ratio is lower than 100% after opening position"),
		"32017": errors.New("no BBO"),
		"32018": errors.New("the order quantity is less than 1, please try again"),
		"32019": errors.New("the order price deviates from the price of the previous minute by more than 3%"),
		"32020": errors.New("the price is not in the range of the price limit"),
		"32021": errors.New("leverage error"),
		"32022": errors.New("this function is not supported in your country or region according to the regulations"),
		"32023": errors.New("this account has outstanding loan"),
		"32024": errors.New("order cannot be placed during delivery"),
		"32025": errors.New("order cannot be placed during settlement"),
		"32026": errors.New("your account is restricted from opening positions"),
		"32027": errors.New("cancelled over 20 orders"),
		"32028": errors.New("account is suspended and liquidated"),
		"32029": errors.New("order info does not exist"),
		"33001": errors.New("margin account for this pair is not enabled yet"),
		"33002": errors.New("margin account for this pair is suspended"),
		"33003": errors.New("no loan balance"),
		"33004": errors.New("loan amount cannot be smaller than the minimum limit"),
		"33005": errors.New("repayment amount must exceed 0"),
		"33006": errors.New("loan order not found"),
		"33007": errors.New("status not found"),
		"33008": errors.New("loan amount cannot exceed the maximum limit"),
		"33009": errors.New("user ID is blank"),
		"33010": errors.New("you cannot cancel an order during session 2 of call auction"),
		"33011": errors.New("no new market data"),
		"33012": errors.New("order cancellation failed"),
		"33013": errors.New("order placement failed"),
		"33014": errors.New("order does not exist"),
		"33015": errors.New("exceeded maximum limit"),
		"33016": errors.New("margin trading is not open for this token"),
		"33017": errors.New("insufficient balance"),
		"33018": errors.New("this parameter must be smaller than 1"),
		"33020": errors.New("request not supported"),
		"33021": errors.New("token and the pair do not match"),
		"33022": errors.New("pair and the order do not match"),
		"33023": errors.New("you can only place market orders during call auction"),
		"33024": errors.New("trading amount too small"),
		"33025": errors.New("base token amount is blank"),
		"33026": errors.New("transaction completed"),
		"33027": errors.New("cancelled order or order cancelling"),
		"33028": errors.New("the decimal places of the trading price exceeded the limit"),
		"33029": errors.New("the decimal places of the trading size exceeded the limit"),
		"34001": errors.New("withdrawal suspended"),
		"34002": errors.New("please add a withdrawal address"),
		"34003": errors.New("sorry, this token cannot be withdrawn to xx at the moment"),
		"34004": errors.New("withdrawal fee is smaller than minimum limit"),
		"34005": errors.New("withdrawal fee exceeds the maximum limit"),
		"34006": errors.New("withdrawal amount is lower than the minimum limit"),
		"34007": errors.New("withdrawal amount exceeds the maximum limit"),
		"34008": errors.New("insufficient balance"),
		"34009": errors.New("your withdrawal amount exceeds the daily limit"),
		"34010": errors.New("transfer amount must be larger than 0"),
		"34011": errors.New("conditions not met"),
		"34012": errors.New("the minimum withdrawal amount for NEO is 1, and the amount must be an integer"),
		"34013": errors.New("please transfer"),
		"34014": errors.New("transfer limited"),
		"34015": errors.New("subaccount does not exist"),
		"34016": errors.New("transfer suspended"),
		"34017": errors.New("account suspended"),
		"34018": errors.New("incorrect trades password"),
		"34019": errors.New("please bind your email before withdrawal"),
		"34020": errors.New("please bind your funds password before withdrawal"),
		"34021": errors.New("Not verified address"),
		"34022": errors.New("Withdrawals are not available for sub accounts"),
		"35001": errors.New("Contract subscribing does not exist"),
		"35002": errors.New("Contract is being settled"),
		"35003": errors.New("Contract is being paused"),
		"35004": errors.New("Pending contract settlement"),
		"35005": errors.New("Perpetual swap trading is not enabled"),
		"35008": errors.New("Margin ratio too low when placing order"),
		"35010": errors.New("Closing position size larger than available size"),
		"35012": errors.New("Placing an order with less than 1 contract"),
		"35014": errors.New("Order size is not in acceptable range"),
		"35015": errors.New("Leverage level unavailable"),
		"35017": errors.New("Changing leverage level"),
		"35019": errors.New("Order size exceeds limit"),
		"35020": errors.New("Order price exceeds limit"),
		"35021": errors.New("Order size exceeds limit of the current tier"),
		"35022": errors.New("Contract is paused or closed"),
		"35030": errors.New("Place multiple orders"),
		"35031": errors.New("Cancel multiple orders"),
		"35061": errors.New("Invalid instrument_id"),
	}
}
