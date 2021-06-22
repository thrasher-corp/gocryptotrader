package okgroup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	okGroupAuthRate   = 0
	okGroupUnauthRate = 0
	// OKGroupAPIPath const to help with api url formatting
	OKGroupAPIPath = "api/"
	// API subsections
	okGroupAccountSubsection        = "account"
	okGroupTokenSubsection          = "spot"
	okGroupMarginTradingSubsection  = "margin"
	okGroupFuturesTradingSubSection = "futures"
	oKGroupSwapTradingSubSection    = "swap"
	// OKGroupAccounts common api endpoint
	OKGroupAccounts = "accounts"
	// OKGroupLedger common api endpoint
	OKGroupLedger = "ledger"
	// OKGroupOrders common api endpoint
	OKGroupOrders = "orders"
	// OKGroupBatchOrders common api endpoint
	OKGroupBatchOrders = "batch_orders"
	// OKGroupCancelOrders common api endpoint
	OKGroupCancelOrders = "cancel_orders"
	// OKGroupCancelOrder common api endpoint
	OKGroupCancelOrder = "cancel_order"
	// OKGroupCancelBatchOrders common api endpoint
	OKGroupCancelBatchOrders = "cancel_batch_orders"
	// OKGroupPendingOrders common api endpoint
	OKGroupPendingOrders = "orders_pending"
	// OKGroupTrades common api endpoint
	OKGroupTrades = "trades"
	// OKGroupTicker common api endpoint
	OKGroupTicker = "ticker"
	// OKGroupInstruments common api endpoint
	OKGroupInstruments = "instruments"
	// OKGroupLiquidation common api endpoint
	OKGroupLiquidation = "liquidation"
	// OKGroupMarkPrice common api endpoint
	OKGroupMarkPrice = "mark_price"
	// OKGroupGetAccountDepositHistory common api endpoint
	OKGroupGetAccountDepositHistory = "deposit/history"
	// OKGroupGetSpotTransactionDetails common api endpoint
	OKGroupGetSpotTransactionDetails = "fills"
	// OKGroupGetSpotOrderBook common api endpoint
	OKGroupGetSpotOrderBook = "book"
	// OKGroupGetSpotMarketData common api endpoint
	OKGroupGetSpotMarketData = "candles"
	// OKGroupPriceLimit common api endpoint
	OKGroupPriceLimit = "price_limit"
	// Account based endpoints
	okGroupGetAccountCurrencies        = "currencies"
	okGroupGetAccountWalletInformation = "wallet"
	okGroupFundsTransfer               = "transfer"
	okGroupWithdraw                    = "withdrawal"
	okGroupGetWithdrawalFees           = "withdrawal/fee"
	okGroupGetWithdrawalHistory        = "withdrawal/history"
	okGroupGetDepositAddress           = "deposit/address"
	// Margin based endpoints
	okGroupGetMarketAvailability = "availability"
	okGroupGetLoanHistory        = "borrowed"
	okGroupGetLoan               = "borrow"
	okGroupGetRepayment          = "repayment"
)

var errMissValue = errors.New("warning - resp value is missing from exchange")

// OKGroup is the overaching type across the all of OKEx's exchange methods
type OKGroup struct {
	exchange.Base
	ExchangeName string
	// Spot and contract market error codes as per https://www.okex.com/rest_request.html
	ErrorCodes map[string]error
	// Stores for corresponding variable checks
	ContractTypes         []string
	CurrencyPairsDefaults []string
	ContractPosition      []string
	Types                 []string
	// URLs to be overridden by implementations of OKGroup
	APIURL       string
	APIVersion   string
	WebsocketURL string
}

// GetAccountCurrencies returns a list of tradable spot instruments and their properties
func (o *OKGroup) GetAccountCurrencies() (resp []GetAccountCurrenciesResponse, _ error) {
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupAccountSubsection, okGroupGetAccountCurrencies, nil, &resp, true)
}

// GetAccountWalletInformation returns a list of wallets and their properties
func (o *OKGroup) GetAccountWalletInformation(currency string) (resp []WalletInformationResponse, _ error) {
	var requestURL string
	if currency != "" {
		requestURL = fmt.Sprintf("%v/%v", okGroupGetAccountWalletInformation, currency)
	} else {
		requestURL = okGroupGetAccountWalletInformation
	}

	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupAccountSubsection, requestURL, nil, &resp, true)
}

// TransferAccountFunds  the transfer of funds between wallet, trading accounts, main account and sub accounts.
func (o *OKGroup) TransferAccountFunds(request TransferAccountFundsRequest) (resp TransferAccountFundsResponse, _ error) {
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodPost, okGroupAccountSubsection, okGroupFundsTransfer, request, &resp, true)
}

// AccountWithdraw withdrawal of tokens to OKCoin International, other OKEx accounts or other addresses.
func (o *OKGroup) AccountWithdraw(request AccountWithdrawRequest) (resp AccountWithdrawResponse, _ error) {
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodPost, okGroupAccountSubsection, okGroupWithdraw, request, &resp, true)
}

// GetAccountWithdrawalFee retrieves the information about the recommended network transaction fee for withdrawals to digital asset addresses. The higher the fees are, the sooner the confirmations you will get.
func (o *OKGroup) GetAccountWithdrawalFee(currency string) (resp []GetAccountWithdrawalFeeResponse, _ error) {
	var requestURL string
	if currency != "" {
		requestURL = fmt.Sprintf("%v?currency=%v", okGroupGetWithdrawalFees, currency)
	} else {
		requestURL = okGroupGetAccountWalletInformation
	}

	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupAccountSubsection, requestURL, nil, &resp, true)
}

// GetAccountWithdrawalHistory retrieves all recent withdrawal records.
func (o *OKGroup) GetAccountWithdrawalHistory(currency string) (resp []WithdrawalHistoryResponse, _ error) {
	var requestURL string
	if currency != "" {
		requestURL = fmt.Sprintf("%v/%v", okGroupGetWithdrawalHistory, currency)
	} else {
		requestURL = okGroupGetWithdrawalHistory
	}
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupAccountSubsection, requestURL, nil, &resp, true)
}

// GetAccountBillDetails retrieves the bill details of the wallet. All the information will be paged and sorted in reverse chronological order,
// which means the latest will be at the top. Please refer to the pagination section for additional records after the first page.
// 3 months recent records will be returned at maximum
func (o *OKGroup) GetAccountBillDetails(request GetAccountBillDetailsRequest) (resp []GetAccountBillDetailsResponse, _ error) {
	requestURL := fmt.Sprintf("%v%v", OKGroupLedger, FormatParameters(request))
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupAccountSubsection, requestURL, nil, &resp, true)
}

// GetAccountDepositAddressForCurrency retrieves the deposit addresses of different tokens, including previously used addresses.
func (o *OKGroup) GetAccountDepositAddressForCurrency(currency string) (resp []GetDepositAddressResponse, _ error) {
	urlValues := url.Values{}
	urlValues.Set("currency", currency)
	requestURL := fmt.Sprintf("%v?%v", okGroupGetDepositAddress, urlValues.Encode())
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupAccountSubsection, requestURL, nil, &resp, true)
}

// GetAccountDepositHistory retrieves the deposit history of all tokens.100 recent records will be returned at maximum
func (o *OKGroup) GetAccountDepositHistory(currency string) (resp []GetAccountDepositHistoryResponse, _ error) {
	var requestURL string
	if currency != "" {
		requestURL = fmt.Sprintf("%v/%v", OKGroupGetAccountDepositHistory, currency)
	} else {
		requestURL = OKGroupGetAccountDepositHistory
	}
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupAccountSubsection, requestURL, nil, &resp, true)
}

// GetSpotTradingAccounts retrieves the list of assets(only show pairs with balance larger than 0), the balances, amount available/on hold in spot accounts.
func (o *OKGroup) GetSpotTradingAccounts() (resp []GetSpotTradingAccountResponse, _ error) {
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupTokenSubsection, OKGroupAccounts, nil, &resp, true)
}

// GetSpotTradingAccountForCurrency This endpoint supports getting the balance, amount available/on hold of a token in spot account.
func (o *OKGroup) GetSpotTradingAccountForCurrency(currency string) (resp GetSpotTradingAccountResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", OKGroupAccounts, currency)
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, true)
}

// GetSpotBillDetailsForCurrency This endpoint supports getting the balance, amount available/on hold of a token in spot account.
func (o *OKGroup) GetSpotBillDetailsForCurrency(request GetSpotBillDetailsForCurrencyRequest) (resp []GetSpotBillDetailsForCurrencyResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", OKGroupAccounts, request.Currency, OKGroupLedger, FormatParameters(request))
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, true)
}

// PlaceSpotOrder token trading only supports limit and market orders (more order types will become available in the future).
// You can place an order only if you have enough funds.
// Once your order is placed, the amount will be put on hold.
func (o *OKGroup) PlaceSpotOrder(request *PlaceOrderRequest) (resp PlaceOrderResponse, _ error) {
	if request.OrderType == "" {
		request.OrderType = strconv.Itoa(NormalOrder)
	}
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodPost, okGroupTokenSubsection, OKGroupOrders, request, &resp, true)
}

// PlaceMultipleSpotOrders supports placing multiple orders for specific trading pairs
// up to 4 trading pairs, maximum 4 orders for each pair
func (o *OKGroup) PlaceMultipleSpotOrders(request []PlaceOrderRequest) (map[string][]PlaceOrderResponse, []error) {
	currencyPairOrders := make(map[string]int)
	resp := make(map[string][]PlaceOrderResponse)

	for i := range request {
		if request[i].OrderType == "" {
			request[i].OrderType = strconv.Itoa(NormalOrder)
		}
		currencyPairOrders[request[i].InstrumentID]++
	}

	if len(currencyPairOrders) > 4 {
		return resp, []error{errors.New("up to 4 trading pairs")}
	}
	for _, orderCount := range currencyPairOrders {
		if orderCount > 4 {
			return resp, []error{errors.New("maximum 4 orders for each pair")}
		}
	}

	err := o.SendHTTPRequest(exchange.RestSpot, http.MethodPost, okGroupTokenSubsection, OKGroupBatchOrders, request, &resp, true)
	if err != nil {
		return resp, []error{err}
	}

	var orderErrors []error
	for currency, orderResponse := range resp {
		for i := range orderResponse {
			if !orderResponse[i].Result {
				orderErrors = append(orderErrors, fmt.Errorf("order for currency %v failed to be placed", currency))
			}
		}
	}

	return resp, orderErrors
}

// CancelSpotOrder Cancelling an unfilled order.
func (o *OKGroup) CancelSpotOrder(request CancelSpotOrderRequest) (resp CancelSpotOrderResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", OKGroupCancelOrders, request.OrderID)
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodPost, okGroupTokenSubsection, requestURL, request, &resp, true)
}

// CancelMultipleSpotOrders Cancelling multiple unfilled orders.
func (o *OKGroup) CancelMultipleSpotOrders(request CancelMultipleSpotOrdersRequest) (resp map[string][]CancelMultipleSpotOrdersResponse, err error) {
	resp = make(map[string][]CancelMultipleSpotOrdersResponse)
	if len(request.OrderIDs) > 4 {
		return resp, errors.New("maximum 4 order cancellations for each pair")
	}

	err = o.SendHTTPRequest(exchange.RestSpot, http.MethodPost, okGroupTokenSubsection, OKGroupCancelBatchOrders, []CancelMultipleSpotOrdersRequest{request}, &resp, true)
	if err != nil {
		return
	}

	for currency, orderResponse := range resp {
		for i := range orderResponse {
			cancellationResponse := CancelMultipleSpotOrdersResponse{
				OrderID:   orderResponse[i].OrderID,
				Result:    orderResponse[i].Result,
				ClientOID: orderResponse[i].ClientOID,
			}

			if !orderResponse[i].Result {
				cancellationResponse.Error = fmt.Errorf("order %v for currency %v failed to be cancelled", orderResponse[i].OrderID, currency)
			}

			resp[currency] = append(resp[currency], cancellationResponse)
		}
	}

	return
}

// GetSpotOrders List your orders. Cursor pagination is used.
// All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetSpotOrders(request GetSpotOrdersRequest) (resp []GetSpotOrderResponse, _ error) {
	requestURL := fmt.Sprintf("%v%v", OKGroupOrders, FormatParameters(request))
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, true)
}

// GetSpotOpenOrders List all your current open orders. Cursor pagination is used.
// All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetSpotOpenOrders(request GetSpotOpenOrdersRequest) (resp []GetSpotOrderResponse, _ error) {
	requestURL := fmt.Sprintf("%v%v", OKGroupPendingOrders, FormatParameters(request))
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, true)
}

// GetSpotOrder Get order details by order ID.
func (o *OKGroup) GetSpotOrder(request GetSpotOrderRequest) (resp GetSpotOrderResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v%v", OKGroupOrders, request.OrderID, FormatParameters(request))
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupTokenSubsection, requestURL, request, &resp, true)
}

// GetSpotTransactionDetails Get details of the recent filled orders. Cursor pagination is used.
// All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetSpotTransactionDetails(request GetSpotTransactionDetailsRequest) (resp []GetSpotTransactionDetailsResponse, _ error) {
	requestURL := fmt.Sprintf("%v%v", OKGroupGetSpotTransactionDetails, FormatParameters(request))
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, false)
}

// GetSpotTokenPairDetails Get market data. This endpoint provides the snapshots of market data and can be used without verifications.
// List trading pairs and get the trading limit, price, and more information of different trading pairs.
func (o *OKGroup) GetSpotTokenPairDetails() (resp []GetSpotTokenPairDetailsResponse, _ error) {
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupTokenSubsection, OKGroupInstruments, nil, &resp, false)
}

// GetOrderBook Getting the order book of a trading pair. Pagination is not
// supported here. The whole book will be returned for one request. Websocket is
// recommended here.
func (o *OKGroup) GetOrderBook(request GetOrderBookRequest, a asset.Item) (resp GetOrderBookResponse, _ error) {
	var requestType, endpoint string
	switch a {
	case asset.Spot:
		endpoint = OKGroupGetSpotOrderBook
		requestType = okGroupTokenSubsection
	case asset.Futures:
		endpoint = OKGroupGetSpotOrderBook
		requestType = "futures"
	case asset.PerpetualSwap:
		endpoint = "depth"
		requestType = "swap"
	default:
		return resp, errors.New("unhandled asset type")
	}
	requestURL := fmt.Sprintf("%v/%v/%v%v",
		OKGroupInstruments,
		request.InstrumentID,
		endpoint,
		FormatParameters(request))
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet,
		requestType,
		requestURL,
		nil,
		&resp,
		false)
}

// GetSpotAllTokenPairsInformation Get the last traded price, best bid/ask price, 24 hour trading volume and more info of all trading pairs.
func (o *OKGroup) GetSpotAllTokenPairsInformation() (resp []GetSpotTokenPairsInformationResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", OKGroupInstruments, OKGroupTicker)
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, false)
}

// GetSpotAllTokenPairsInformationForCurrency Get the last traded price, best bid/ask price, 24 hour trading volume and more info of a currency
func (o *OKGroup) GetSpotAllTokenPairsInformationForCurrency(currency string) (resp GetSpotTokenPairsInformationResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v", OKGroupInstruments, currency, OKGroupTicker)
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, false)
}

// GetSpotFilledOrdersInformation Get the recent 60 transactions of all trading pairs.
// Cursor pagination is used. All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetSpotFilledOrdersInformation(request GetSpotFilledOrdersInformationRequest) (resp []GetSpotFilledOrdersInformationResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", OKGroupInstruments, request.InstrumentID, OKGroupTrades, FormatParameters(request))
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupTokenSubsection, requestURL, nil, &resp, false)
}

// GetMarketData Get the charts of the trading pairs. Charts are returned in grouped buckets based on requested granularity.
func (o *OKGroup) GetMarketData(request *GetMarketDataRequest) (resp GetMarketDataResponse, err error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", OKGroupInstruments, request.InstrumentID, OKGroupGetSpotMarketData, FormatParameters(request))
	var requestType string
	switch request.Asset {
	case asset.Spot, asset.Margin:
		requestType = okGroupTokenSubsection
	case asset.Futures:
		requestType = okGroupFuturesTradingSubSection
	case asset.PerpetualSwap:
		requestType = oKGroupSwapTradingSubSection
	default:
		return nil, errors.New("asset not supported")
	}
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, requestType, requestURL, nil, &resp, false)
}

// GetMarginTradingAccounts List all assets under token margin trading account, including information such as balance, amount on hold and more.
func (o *OKGroup) GetMarginTradingAccounts() (resp []GetMarginAccountsResponse, _ error) {
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupMarginTradingSubsection, OKGroupAccounts, nil, &resp, true)
}

// GetMarginTradingAccountsForCurrency Get the balance, amount on hold and more useful information.
func (o *OKGroup) GetMarginTradingAccountsForCurrency(currency string) (resp GetMarginAccountsResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", OKGroupAccounts, currency)
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupMarginTradingSubsection, requestURL, nil, &resp, true)
}

// GetMarginBillDetails List all bill details. Pagination is used here.
// before and after cursor arguments should not be confused with before and after in chronological time.
// Most paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetMarginBillDetails(request GetMarginBillDetailsRequest) (resp []GetSpotBillDetailsForCurrencyResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", OKGroupAccounts, request.InstrumentID, OKGroupLedger, FormatParameters(request))
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupMarginTradingSubsection, requestURL, nil, &resp, true)
}

// GetMarginAccountSettings Get all information of the margin trading account,
// including the maximum loan amount, interest rate, and maximum leverage.
func (o *OKGroup) GetMarginAccountSettings(currency string) (resp []GetMarginAccountSettingsResponse, _ error) {
	var requestURL string
	if currency != "" {
		requestURL = fmt.Sprintf("%v/%v/%v", OKGroupAccounts, currency, okGroupGetMarketAvailability)
	} else {
		requestURL = fmt.Sprintf("%v/%v", OKGroupAccounts, okGroupGetMarketAvailability)
	}
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupMarginTradingSubsection, requestURL, nil, &resp, true)
}

// GetMarginLoanHistory Get loan history of the margin trading account.
// Pagination is used here. before and after cursor arguments should not be confused with before and after in chronological time.
// Most paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetMarginLoanHistory(request GetMarginLoanHistoryRequest) (resp []GetMarginLoanHistoryResponse, _ error) {
	var requestURL string
	if len(request.InstrumentID) > 0 {
		requestURL = fmt.Sprintf("%v/%v/%v", OKGroupAccounts, request.InstrumentID, okGroupGetLoan)
	} else {
		requestURL = fmt.Sprintf("%v/%v", OKGroupAccounts, okGroupGetLoan)
	}
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupMarginTradingSubsection, requestURL, nil, &resp, true)
}

// OpenMarginLoan Borrowing tokens in a margin trading account.
func (o *OKGroup) OpenMarginLoan(request OpenMarginLoanRequest) (resp OpenMarginLoanResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", OKGroupAccounts, okGroupGetLoan)
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodPost, okGroupMarginTradingSubsection, requestURL, request, &resp, true)
}

// RepayMarginLoan Repaying tokens in a margin trading account.
func (o *OKGroup) RepayMarginLoan(request RepayMarginLoanRequest) (resp RepayMarginLoanResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", OKGroupAccounts, okGroupGetRepayment)
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodPost, okGroupMarginTradingSubsection, requestURL, request, &resp, true)
}

// PlaceMarginOrder OKEx API only supports limit and market orders (more orders will become available in the future).
// You can place an order only if you have enough funds. Once your order is placed, the amount will be put on hold.
func (o *OKGroup) PlaceMarginOrder(request *PlaceOrderRequest) (resp PlaceOrderResponse, _ error) {
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodPost, okGroupMarginTradingSubsection, OKGroupOrders, request, &resp, true)
}

// PlaceMultipleMarginOrders Place multiple orders for specific trading pairs (up to 4 trading pairs, maximum 4 orders each)
func (o *OKGroup) PlaceMultipleMarginOrders(request []PlaceOrderRequest) (map[string][]PlaceOrderResponse, []error) {
	currencyPairOrders := make(map[string]int)
	resp := make(map[string][]PlaceOrderResponse)
	for i := range request {
		currencyPairOrders[request[i].InstrumentID]++
	}
	if len(currencyPairOrders) > 4 {
		return resp, []error{errors.New("up to 4 trading pairs")}
	}
	for _, orderCount := range currencyPairOrders {
		if orderCount > 4 {
			return resp, []error{errors.New("maximum 4 orders for each pair")}
		}
	}

	err := o.SendHTTPRequest(exchange.RestSpot, http.MethodPost, okGroupMarginTradingSubsection, OKGroupBatchOrders, request, &resp, true)
	if err != nil {
		return resp, []error{err}
	}

	var orderErrors []error
	for currency, orderResponse := range resp {
		for i := range orderResponse {
			if !orderResponse[i].Result {
				orderErrors = append(orderErrors, fmt.Errorf("order for currency %v failed to be placed", currency))
			}
		}
	}

	return resp, orderErrors
}

// CancelMarginOrder Cancelling an unfilled order.
func (o *OKGroup) CancelMarginOrder(request CancelSpotOrderRequest) (resp CancelSpotOrderResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v", OKGroupCancelOrders, request.OrderID)
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodPost, okGroupMarginTradingSubsection, requestURL, request, &resp, true)
}

// CancelMultipleMarginOrders Cancelling multiple unfilled orders.
func (o *OKGroup) CancelMultipleMarginOrders(request CancelMultipleSpotOrdersRequest) (map[string][]CancelMultipleSpotOrdersResponse, []error) {
	resp := make(map[string][]CancelMultipleSpotOrdersResponse)
	if len(request.OrderIDs) > 4 {
		return resp, []error{errors.New("maximum 4 order cancellations for each pair")}
	}

	err := o.SendHTTPRequest(exchange.RestSpot, http.MethodPost, okGroupMarginTradingSubsection, OKGroupCancelBatchOrders, []CancelMultipleSpotOrdersRequest{request}, &resp, true)
	if err != nil {
		return resp, []error{err}
	}

	var orderErrors []error
	for currency, orderResponse := range resp {
		for i := range orderResponse {
			if !orderResponse[i].Result {
				orderErrors = append(orderErrors, fmt.Errorf("order %v for currency %v failed to be cancelled", orderResponse[i].OrderID, currency))
			}
		}
	}

	return resp, orderErrors
}

// GetMarginOrders List your orders. Cursor pagination is used. All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetMarginOrders(request GetSpotOrdersRequest) (resp []GetSpotOrderResponse, _ error) {
	requestURL := fmt.Sprintf("%v%v", OKGroupOrders, FormatParameters(request))
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupMarginTradingSubsection, requestURL, nil, &resp, true)
}

// GetMarginOpenOrders List all your current open orders. Cursor pagination is used. All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetMarginOpenOrders(request GetSpotOpenOrdersRequest) (resp []GetSpotOrderResponse, _ error) {
	requestURL := fmt.Sprintf("%v%v", OKGroupPendingOrders, FormatParameters(request))
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupMarginTradingSubsection, requestURL, nil, &resp, true)
}

// GetMarginOrder Get order details by order ID.
func (o *OKGroup) GetMarginOrder(request GetSpotOrderRequest) (resp GetSpotOrderResponse, _ error) {
	requestURL := fmt.Sprintf("%v/%v%v", OKGroupOrders, request.OrderID, FormatParameters(request))
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupMarginTradingSubsection, requestURL, request, &resp, true)
}

// GetMarginTransactionDetails Get details of the recent filled orders. Cursor pagination is used.
// All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKGroup) GetMarginTransactionDetails(request GetSpotTransactionDetailsRequest) (resp []GetSpotTransactionDetailsResponse, _ error) {
	requestURL := fmt.Sprintf("%v%v", OKGroupGetSpotTransactionDetails, FormatParameters(request))
	return resp, o.SendHTTPRequest(exchange.RestSpot, http.MethodGet, okGroupMarginTradingSubsection, requestURL, nil, &resp, true)
}

// FormatParameters Formats URL parameters, useful for optional parameters due to OKEX signature check
func FormatParameters(request interface{}) (parameters string) {
	v, err := query.Values(request)
	if err != nil {
		log.Errorf(log.ExchangeSys, "Could not parse %v to URL values. Check that the type has url fields", reflect.TypeOf(request).Name())
		return
	}
	urlEncodedValues := v.Encode()
	if len(urlEncodedValues) > 0 {
		parameters = fmt.Sprintf("?%v", urlEncodedValues)
	}
	return
}

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
func (o *OKGroup) SendHTTPRequest(ep exchange.URL, httpMethod, requestType, requestPath string, data, result interface{}, authenticated bool) (err error) {
	if authenticated && !o.AllowAuthenticatedRequest() {
		return fmt.Errorf("%s %w", o.Name, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}
	endpoint, err := o.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	now := time.Now()
	utcTime := now.UTC().Format(time.RFC3339)
	payload := []byte("")

	if data != nil {
		payload, err = json.Marshal(data)
		if err != nil {
			return errors.New("sendHTTPRequest: Unable to JSON request")
		}

		if o.Verbose {
			log.Debugf(log.ExchangeSys, "Request JSON: %s\n", payload)
		}
	}

	path := endpoint + requestType + o.APIVersion + requestPath
	if o.Verbose {
		log.Debugf(log.ExchangeSys, "Sending %v request to %s \n", requestType, path)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	if authenticated {
		signPath := fmt.Sprintf("/%v%v%v%v", OKGroupAPIPath,
			requestType, o.APIVersion, requestPath)
		hmac := crypto.GetHMAC(crypto.HashSHA256,
			[]byte(utcTime+httpMethod+signPath+string(payload)),
			[]byte(o.API.Credentials.Secret))
		headers["OK-ACCESS-KEY"] = o.API.Credentials.Key
		headers["OK-ACCESS-SIGN"] = crypto.Base64Encode(hmac)
		headers["OK-ACCESS-TIMESTAMP"] = utcTime
		headers["OK-ACCESS-PASSPHRASE"] = o.API.Credentials.ClientID
	}

	// Requests that have a 30+ second difference between the timestamp and the API service time will be considered expired or rejected
	ctx, cancel := context.WithDeadline(context.Background(), now.Add(30*time.Second))
	defer cancel()
	var intermediary json.RawMessage
	type errCapFormat struct {
		Error        int64  `json:"error_code,omitempty"`
		ErrorMessage string `json:"error_message,omitempty"`
		Result       bool   `json:"result,string,omitempty"`
	}

	errCap := errCapFormat{}
	errCap.Result = true
	err = o.SendPayload(ctx, &request.Item{
		Method:        strings.ToUpper(httpMethod),
		Path:          path,
		Headers:       headers,
		Body:          bytes.NewBuffer(payload),
		Result:        &intermediary,
		AuthRequest:   authenticated,
		Verbose:       o.Verbose,
		HTTPDebugging: o.HTTPDebugging,
		HTTPRecording: o.HTTPRecording,
	})
	if err != nil {
		return err
	}

	err = json.Unmarshal(intermediary, &errCap)
	if err == nil {
		if errCap.ErrorMessage != "" {
			return fmt.Errorf("error: %v", errCap.ErrorMessage)
		}
		if errCap.Error > 0 {
			return fmt.Errorf("sendHTTPRequest error - %s",
				o.ErrorCodes[strconv.FormatInt(errCap.Error, 10)])
		}
		if !errCap.Result {
			return errors.New("unspecified error occurred")
		}
	}

	return json.Unmarshal(intermediary, result)
}

// SetCheckVarDefaults sets main variables that will be used in requests because
// api does not return an error if there are misspellings in strings. So better
// to check on this, this end.
func (o *OKGroup) SetCheckVarDefaults() {
	o.ContractTypes = []string{"this_week", "next_week", "quarter"}
	o.CurrencyPairsDefaults = []string{"btc_usd", "ltc_usd", "eth_usd", "etc_usd", "bch_usd"}
	o.Types = []string{"1min", "3min", "5min", "15min", "30min", "1day", "3day",
		"1week", "1hour", "2hour", "4hour", "6hour", "12hour"}
	o.ContractPosition = []string{"1", "2", "3", "4"}
}

// GetFee returns an estimate of fee based on type of transaction
func (o *OKGroup) GetFee(feeBuilder *exchange.FeeBuilder) (fee float64, _ error) {
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount, feeBuilder.IsMaker)
	case exchange.CryptocurrencyWithdrawalFee:
		withdrawFees, err := o.GetAccountWithdrawalFee(feeBuilder.FiatCurrency.String())
		if err != nil {
			return -1, err
		}
		for _, withdrawFee := range withdrawFees {
			if withdrawFee.Currency == feeBuilder.FiatCurrency.String() {
				fee = withdrawFee.MinFee
				break
			}
		}
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.0015 * price * amount
}

func calculateTradingFee(purchasePrice, amount float64, isMaker bool) (fee float64) {
	// TODO volume based fees
	if isMaker {
		fee = 0.0005
	} else {
		fee = 0.0015
	}
	return fee * amount * purchasePrice
}

// SetErrorDefaults sets the full error default list
func (o *OKGroup) SetErrorDefaults() {
	o.ErrorCodes = map[string]error{
		"0":     errors.New("successful"),
		"1":     errors.New("invalid parameter in url normally"),
		"30001": errors.New("request header \"OK_ACCESS_KEY\" cannot be blank"),
		"30002": errors.New("request header \"OK_ACCESS_SIGN\" cannot be blank"),
		"30003": errors.New("request header \"OK_ACCESS_TIMESTAMP\" cannot be blank"),
		"30004": errors.New("request header \"OK_ACCESS_PASSPHRASE\" cannot be blank"),
		"30005": errors.New("invalid OK_ACCESS_TIMESTAMP"),
		"30006": errors.New("invalid OK_ACCESS_KEY"),
		"30007": errors.New("invalid Content_Type, please use \"application/json\" format"),
		"30008": errors.New("timestamp request expired"),
		"30009": errors.New("system error"),
		"30010": errors.New("api validation failed"),
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
		"32008": errors.New("max. positions to open (cross margin)"),
		"32009": errors.New("max. positions to open (fixed margin)"),
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
		"34021": errors.New("not verified address"),
		"34022": errors.New("withdrawals are not available for sub accounts"),
		"35001": errors.New("contract subscribing does not exist"),
		"35002": errors.New("contract is being settled"),
		"35003": errors.New("contract is being paused"),
		"35004": errors.New("pending contract settlement"),
		"35005": errors.New("perpetual swap trading is not enabled"),
		"35008": errors.New("margin ratio too low when placing order"),
		"35010": errors.New("closing position size larger than available size"),
		"35012": errors.New("placing an order with less than 1 contract"),
		"35014": errors.New("order size is not in acceptable range"),
		"35015": errors.New("leverage level unavailable"),
		"35017": errors.New("changing leverage level"),
		"35019": errors.New("order size exceeds limit"),
		"35020": errors.New("order price exceeds limit"),
		"35021": errors.New("order size exceeds limit of the current tier"),
		"35022": errors.New("contract is paused or closed"),
		"35030": errors.New("place multiple orders"),
		"35031": errors.New("cancel multiple orders"),
		"35061": errors.New("invalid instrument_id"),
	}
}
