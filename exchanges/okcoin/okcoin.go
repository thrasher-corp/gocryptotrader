package okcoin

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	okCoinRateInterval        = time.Second
	okCoinStandardRequestRate = 6
	apiPath                   = "api/"
	okCoinAPIURL              = "https://www.okcoin.com/" + apiPath
	okCoinAPIVersion          = "/v3/"
	okCoinExchangeName        = "OKCOIN International"
	okCoinWebsocketURL        = "wss://real.okcoin.com:8443/ws/v3"
)

// OKCoin bases all methods off oKCoin implementation
type OKCoin struct {
	exchange.Base
	// Spot and contract market error codes
	ErrorCodes map[string]error
}

const (
	accountSubsection         = "account"
	tokenSubsection           = "spot"
	marginTradingSubsection   = "margin"
	accounts                  = "accounts"
	ledger                    = "ledger"
	orders                    = "orders"
	batchOrders               = "batch_orders"
	cancelOrders              = "cancel_orders"
	cancelOrder               = "cancel_order"
	cancelBatchOrders         = "cancel_batch_orders"
	pendingOrders             = "orders_pending"
	trades                    = "trades"
	tickerData                = "ticker"
	instruments               = "instruments"
	getAccountDepositHistory  = "deposit/history"
	getSpotTransactionDetails = "fills"
	getSpotOrderBook          = "book"
	getSpotMarketData         = "candles"
	// Account based endpoints
	getAccountCurrencies        = "currencies"
	getAccountWalletInformation = "wallet"
	fundsTransfer               = "transfer"
	withdrawRequest             = "withdrawal"
	getWithdrawalFees           = "withdrawal/fee"
	getWithdrawalHistory        = "withdrawal/history"
	getDepositAddress           = "deposit/address"
	// Margin based endpoints
	getMarketAvailability = "availability"
	getLoan               = "borrow"
	getRepayment          = "repayment"
)

// GetAccountCurrencies returns a list of tradable spot instruments and their properties
func (o *OKCoin) GetAccountCurrencies(ctx context.Context) ([]GetAccountCurrenciesResponse, error) {
	var respData []struct {
		Name          string `json:"name"`
		Currency      string `json:"currency"`
		Chain         string `json:"chain"`
		CanInternal   int64  `json:"can_internal,string"`
		CanWithdraw   int64  `json:"can_withdraw,string"`
		CanDeposit    int64  `json:"can_deposit,string"`
		MinWithdrawal string `json:"min_withdrawal"`
	}
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, accountSubsection, getAccountCurrencies, nil, &respData, true)
	if err != nil {
		return nil, err
	}
	resp := make([]GetAccountCurrenciesResponse, len(respData))
	for i := range respData {
		var mw float64
		mw, err = strconv.ParseFloat(respData[i].MinWithdrawal, 64)
		if err != nil {
			return nil, err
		}
		resp[i] = GetAccountCurrenciesResponse{
			Name:          respData[i].Name,
			Currency:      respData[i].Currency,
			Chain:         respData[i].Chain,
			CanInternal:   respData[i].CanInternal == 1,
			CanWithdraw:   respData[i].CanWithdraw == 1,
			CanDeposit:    respData[i].CanDeposit == 1,
			MinWithdrawal: mw,
		}
	}
	return resp, nil
}

// GetAccountWalletInformation returns a list of wallets and their properties
func (o *OKCoin) GetAccountWalletInformation(ctx context.Context, currency string) ([]WalletInformationResponse, error) {
	var requestURL string
	if currency != "" {
		requestURL = fmt.Sprintf("%v/%v", getAccountWalletInformation, currency)
	} else {
		requestURL = getAccountWalletInformation
	}

	var resp []WalletInformationResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, accountSubsection, requestURL, nil, &resp, true)
}

// TransferAccountFunds  the transfer of funds between wallet, trading accounts, main account and sub-accounts.
func (o *OKCoin) TransferAccountFunds(ctx context.Context, request TransferAccountFundsRequest) (TransferAccountFundsResponse, error) {
	var resp TransferAccountFundsResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, accountSubsection, fundsTransfer, request, &resp, true)
}

// AccountWithdraw withdrawal of tokens to OKCoin International or other addresses.
func (o *OKCoin) AccountWithdraw(ctx context.Context, request AccountWithdrawRequest) (AccountWithdrawResponse, error) {
	var resp AccountWithdrawResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, accountSubsection, withdrawRequest, request, &resp, true)
}

// GetAccountWithdrawalFee retrieves the information about the recommended network transaction fee for withdrawals to digital asset addresses. The higher the fees are, the sooner the confirmations you will get.
func (o *OKCoin) GetAccountWithdrawalFee(ctx context.Context, currency string) ([]GetAccountWithdrawalFeeResponse, error) {
	var requestURL string
	if currency != "" {
		requestURL = fmt.Sprintf("%v?currency=%v", getWithdrawalFees, currency)
	} else {
		requestURL = getAccountWalletInformation
	}

	var resp []GetAccountWithdrawalFeeResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, accountSubsection, requestURL, nil, &resp, true)
}

// GetAccountWithdrawalHistory retrieves all recent withdrawal records.
func (o *OKCoin) GetAccountWithdrawalHistory(ctx context.Context, currency string) ([]WithdrawalHistoryResponse, error) {
	var requestURL string
	if currency != "" {
		requestURL = fmt.Sprintf("%v/%v", getWithdrawalHistory, currency)
	} else {
		requestURL = getWithdrawalHistory
	}
	var resp []WithdrawalHistoryResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, accountSubsection, requestURL, nil, &resp, true)
}

// GetAccountBillDetails retrieves the bill details of the wallet. All the information will be paged and sorted in reverse chronological order,
// which means the latest will be at the top. Please refer to the pagination section for additional records after the first page.
// 3 months recent records will be returned at maximum
func (o *OKCoin) GetAccountBillDetails(ctx context.Context, request GetAccountBillDetailsRequest) ([]GetAccountBillDetailsResponse, error) {
	requestURL := fmt.Sprintf("%v%v", ledger, FormatParameters(request))
	var resp []GetAccountBillDetailsResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, accountSubsection, requestURL, nil, &resp, true)
}

// GetAccountDepositAddressForCurrency retrieves the deposit addresses of different tokens, including previously used addresses.
func (o *OKCoin) GetAccountDepositAddressForCurrency(ctx context.Context, currency string) ([]GetDepositAddressResponse, error) {
	urlValues := url.Values{}
	urlValues.Set("currency", currency)
	requestURL := fmt.Sprintf("%v?%v", getDepositAddress, urlValues.Encode())
	var resp []GetDepositAddressResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, accountSubsection, requestURL, nil, &resp, true)
}

// GetAccountDepositHistory retrieves the deposit history of all tokens.100 recent records will be returned at maximum
func (o *OKCoin) GetAccountDepositHistory(ctx context.Context, currency string) ([]GetAccountDepositHistoryResponse, error) {
	var requestURL string
	if currency != "" {
		requestURL = fmt.Sprintf("%v/%v", getAccountDepositHistory, currency)
	} else {
		requestURL = getAccountDepositHistory
	}
	var resp []GetAccountDepositHistoryResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, accountSubsection, requestURL, nil, &resp, true)
}

// GetSpotTradingAccounts retrieves the list of assets(only show pairs with balance larger than 0), the balances, amount available/on hold in spot accounts.
func (o *OKCoin) GetSpotTradingAccounts(ctx context.Context) ([]GetSpotTradingAccountResponse, error) {
	var resp []GetSpotTradingAccountResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, accounts, nil, &resp, true)
}

// GetSpotTradingAccountForCurrency This endpoint supports getting the balance, amount available/on hold of a token in spot account.
func (o *OKCoin) GetSpotTradingAccountForCurrency(ctx context.Context, currency string) (GetSpotTradingAccountResponse, error) {
	requestURL := fmt.Sprintf("%v/%v", accounts, currency)
	var resp GetSpotTradingAccountResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, true)
}

// GetSpotBillDetailsForCurrency This endpoint supports getting the balance, amount available/on hold of a token in spot account.
func (o *OKCoin) GetSpotBillDetailsForCurrency(ctx context.Context, request GetSpotBillDetailsForCurrencyRequest) ([]GetSpotBillDetailsForCurrencyResponse, error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", accounts, request.Currency, ledger, FormatParameters(request))
	var resp []GetSpotBillDetailsForCurrencyResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, true)
}

// PlaceSpotOrder token trading only supports limit and market orders (more order types will become available in the future).
// You can place an order only if you have enough funds.
// Once your order is placed, the amount will be put on hold.
func (o *OKCoin) PlaceSpotOrder(ctx context.Context, request *PlaceOrderRequest) (PlaceOrderResponse, error) {
	if request.OrderType == "" {
		request.OrderType = "0"
	}
	var resp PlaceOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, tokenSubsection, orders, request, &resp, true)
}

// PlaceMultipleSpotOrders supports placing multiple orders for specific trading pairs
// up to 4 trading pairs, maximum 4 orders for each pair
func (o *OKCoin) PlaceMultipleSpotOrders(ctx context.Context, request []PlaceOrderRequest) (map[string][]PlaceOrderResponse, []error) {
	currencyPairOrders := make(map[string]int)
	resp := make(map[string][]PlaceOrderResponse)

	for i := range request {
		if request[i].OrderType == "" {
			request[i].OrderType = "0"
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

	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, tokenSubsection, batchOrders, request, &resp, true)
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
func (o *OKCoin) CancelSpotOrder(ctx context.Context, request CancelSpotOrderRequest) (CancelSpotOrderResponse, error) {
	requestURL := fmt.Sprintf("%v/%v", cancelOrders, request.OrderID)
	var resp CancelSpotOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, tokenSubsection, requestURL, request, &resp, true)
}

// CancelMultipleSpotOrders Cancelling multiple unfilled orders.
func (o *OKCoin) CancelMultipleSpotOrders(ctx context.Context, request CancelMultipleSpotOrdersRequest) (map[string][]CancelMultipleSpotOrdersResponse, error) {
	if len(request.OrderIDs) > 4 {
		return nil, errors.New("maximum 4 order cancellations for each pair")
	}

	resp := make(map[string][]CancelMultipleSpotOrdersResponse)
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, tokenSubsection, cancelBatchOrders, []CancelMultipleSpotOrdersRequest{request}, &resp, true)
	if err != nil {
		return nil, err
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

	return resp, nil
}

// GetSpotOrders List your orders. Cursor pagination is used.
// All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKCoin) GetSpotOrders(ctx context.Context, request GetSpotOrdersRequest) ([]GetSpotOrderResponse, error) {
	requestURL := fmt.Sprintf("%v%v", orders, FormatParameters(request))
	var resp []GetSpotOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, true)
}

// GetSpotOpenOrders List all your current open orders. Cursor pagination is used.
// All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKCoin) GetSpotOpenOrders(ctx context.Context, request GetSpotOpenOrdersRequest) ([]GetSpotOrderResponse, error) {
	requestURL := fmt.Sprintf("%v%v", pendingOrders, FormatParameters(request))
	var resp []GetSpotOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, true)
}

// GetSpotOrder Get order details by order ID.
func (o *OKCoin) GetSpotOrder(ctx context.Context, request GetSpotOrderRequest) (GetSpotOrderResponse, error) {
	requestURL := fmt.Sprintf("%v/%v%v", orders, request.OrderID, FormatParameters(request))
	var resp GetSpotOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, request, &resp, true)
}

// GetSpotTransactionDetails Get details of the recent filled orders. Cursor pagination is used.
// All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKCoin) GetSpotTransactionDetails(ctx context.Context, request GetSpotTransactionDetailsRequest) ([]GetSpotTransactionDetailsResponse, error) {
	requestURL := fmt.Sprintf("%v%v", getSpotTransactionDetails, FormatParameters(request))
	var resp []GetSpotTransactionDetailsResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, true)
}

// GetSpotTokenPairDetails Get market data. This endpoint provides the snapshots of market data and can be used without verifications.
// List trading pairs and get the trading limit, price, and more information of different trading pairs.
func (o *OKCoin) GetSpotTokenPairDetails(ctx context.Context) ([]GetSpotTokenPairDetailsResponse, error) {
	var resp []GetSpotTokenPairDetailsResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, instruments, nil, &resp, false)
}

// GetOrderBook Getting the order book of a trading pair. Pagination is not
// supported here. The whole book will be returned for one request. Websocket is
// recommended here.
func (o *OKCoin) GetOrderBook(ctx context.Context, request *GetOrderBookRequest, a asset.Item) (*GetOrderBookResponse, error) {
	var requestType, endpoint string
	var resp *GetOrderBookResponse
	if a != asset.Spot {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	switch a {
	case asset.Spot:
		endpoint = getSpotOrderBook
		requestType = tokenSubsection
	default:
		return resp, errors.New("unhandled asset type")
	}
	requestURL := fmt.Sprintf("%v/%v/%v%v", instruments, request.InstrumentID, endpoint, FormatParameters(request))
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, requestType, requestURL, nil, &resp, false)
}

// GetSpotAllTokenPairsInformation Get the last traded price, best bid/ask price, 24 hour trading volume and more info of all trading pairs.
func (o *OKCoin) GetSpotAllTokenPairsInformation(ctx context.Context) ([]GetSpotTokenPairsInformationResponse, error) {
	requestURL := fmt.Sprintf("%v/%v", instruments, tickerData)
	var resp []GetSpotTokenPairsInformationResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, false)
}

// GetSpotAllTokenPairsInformationForCurrency Get the last traded price, best bid/ask price, 24 hour trading volume and more info of a currency
func (o *OKCoin) GetSpotAllTokenPairsInformationForCurrency(ctx context.Context, currency string) (GetSpotTokenPairsInformationResponse, error) {
	requestURL := fmt.Sprintf("%v/%v/%v", instruments, currency, tickerData)
	var resp GetSpotTokenPairsInformationResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, false)
}

// GetSpotFilledOrdersInformation Get the recent 60 transactions of all trading pairs.
// Cursor pagination is used. All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKCoin) GetSpotFilledOrdersInformation(ctx context.Context, request GetSpotFilledOrdersInformationRequest) ([]GetSpotFilledOrdersInformationResponse, error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", instruments, request.InstrumentID, trades, FormatParameters(request))
	var resp []GetSpotFilledOrdersInformationResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, false)
}

// GetMarketData Get the charts of the trading pairs. Charts are returned in grouped buckets based on requested granularity.
func (o *OKCoin) GetMarketData(ctx context.Context, request *GetMarketDataRequest) (GetMarketDataResponse, error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", instruments, request.InstrumentID, getSpotMarketData, FormatParameters(request))
	if request.Asset != asset.Spot && request.Asset != asset.Margin {
		return nil, asset.ErrNotSupported
	}
	var resp GetMarketDataResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, false)
}

// GetMarginTradingAccounts List all assets under token margin trading account, including information such as balance, amount on hold and more.
func (o *OKCoin) GetMarginTradingAccounts(ctx context.Context) ([]GetMarginAccountsResponse, error) {
	var resp []GetMarginAccountsResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, accounts, nil, &resp, true)
}

// GetMarginTradingAccountsForCurrency Get the balance, amount on hold and more useful information.
func (o *OKCoin) GetMarginTradingAccountsForCurrency(ctx context.Context, currency string) (GetMarginAccountsResponse, error) {
	requestURL := fmt.Sprintf("%v/%v", accounts, currency)
	var resp GetMarginAccountsResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, requestURL, nil, &resp, true)
}

// GetMarginBillDetails List all bill details. Pagination is used here.
// before and after cursor arguments should not be confused with before and after in chronological time.
// Most paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKCoin) GetMarginBillDetails(ctx context.Context, request GetMarginBillDetailsRequest) ([]GetSpotBillDetailsForCurrencyResponse, error) {
	requestURL := fmt.Sprintf("%v/%v/%v%v", accounts, request.InstrumentID, ledger, FormatParameters(request))
	var resp []GetSpotBillDetailsForCurrencyResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, requestURL, nil, &resp, true)
}

// GetMarginAccountSettings Get all information of the margin trading account,
// including the maximum loan amount, interest rate, and maximum leverage.
func (o *OKCoin) GetMarginAccountSettings(ctx context.Context, currency string) ([]GetMarginAccountSettingsResponse, error) {
	var requestURL string
	if currency != "" {
		requestURL = fmt.Sprintf("%v/%v/%v", accounts, currency, getMarketAvailability)
	} else {
		requestURL = fmt.Sprintf("%v/%v", accounts, getMarketAvailability)
	}
	var resp []GetMarginAccountSettingsResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, requestURL, nil, &resp, true)
}

// GetMarginLoanHistory Get loan history of the margin trading account.
// Pagination is used here. before and after cursor arguments should not be confused with before and after in chronological time.
// Most paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKCoin) GetMarginLoanHistory(ctx context.Context, request GetMarginLoanHistoryRequest) ([]GetMarginLoanHistoryResponse, error) {
	var requestURL string
	if len(request.InstrumentID) > 0 {
		requestURL = fmt.Sprintf("%v/%v/%v", accounts, request.InstrumentID, getLoan)
	} else {
		requestURL = fmt.Sprintf("%v/%v", accounts, getLoan)
	}
	var resp []GetMarginLoanHistoryResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, requestURL, nil, &resp, true)
}

// OpenMarginLoan Borrowing tokens in a margin trading account.
func (o *OKCoin) OpenMarginLoan(ctx context.Context, request OpenMarginLoanRequest) (OpenMarginLoanResponse, error) {
	requestURL := fmt.Sprintf("%v/%v", accounts, getLoan)
	var resp OpenMarginLoanResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, marginTradingSubsection, requestURL, request, &resp, true)
}

// RepayMarginLoan Repaying tokens in a margin trading account.
func (o *OKCoin) RepayMarginLoan(ctx context.Context, request RepayMarginLoanRequest) (RepayMarginLoanResponse, error) {
	requestURL := fmt.Sprintf("%v/%v", accounts, getRepayment)
	var resp RepayMarginLoanResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, marginTradingSubsection, requestURL, request, &resp, true)
}

// PlaceMarginOrder You can place an order only if you have enough funds. Once your order is placed, the amount will be put on hold.
func (o *OKCoin) PlaceMarginOrder(ctx context.Context, request *PlaceOrderRequest) (PlaceOrderResponse, error) {
	var resp PlaceOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, marginTradingSubsection, orders, request, &resp, true)
}

// PlaceMultipleMarginOrders Place multiple orders for specific trading pairs (up to 4 trading pairs, maximum 4 orders each)
func (o *OKCoin) PlaceMultipleMarginOrders(ctx context.Context, request []PlaceOrderRequest) (map[string][]PlaceOrderResponse, []error) {
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

	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, marginTradingSubsection, batchOrders, request, &resp, true)
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
func (o *OKCoin) CancelMarginOrder(ctx context.Context, request CancelSpotOrderRequest) (CancelSpotOrderResponse, error) {
	requestURL := fmt.Sprintf("%v/%v", cancelOrders, request.OrderID)
	var resp CancelSpotOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, marginTradingSubsection, requestURL, request, &resp, true)
}

// CancelMultipleMarginOrders Cancelling multiple unfilled orders.
func (o *OKCoin) CancelMultipleMarginOrders(ctx context.Context, request CancelMultipleSpotOrdersRequest) (map[string][]CancelMultipleSpotOrdersResponse, []error) {
	resp := make(map[string][]CancelMultipleSpotOrdersResponse)
	if len(request.OrderIDs) > 4 {
		return resp, []error{errors.New("maximum 4 order cancellations for each pair")}
	}

	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, marginTradingSubsection, cancelBatchOrders, []CancelMultipleSpotOrdersRequest{request}, &resp, true)
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
func (o *OKCoin) GetMarginOrders(ctx context.Context, request GetSpotOrdersRequest) ([]GetSpotOrderResponse, error) {
	requestURL := fmt.Sprintf("%v%v", orders, FormatParameters(request))
	var resp []GetSpotOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, requestURL, nil, &resp, true)
}

// GetMarginOpenOrders List all your current open orders. Cursor pagination is used. All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKCoin) GetMarginOpenOrders(ctx context.Context, request GetSpotOpenOrdersRequest) ([]GetSpotOrderResponse, error) {
	requestURL := fmt.Sprintf("%v%v", pendingOrders, FormatParameters(request))
	var resp []GetSpotOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, requestURL, nil, &resp, true)
}

// GetMarginOrder Get order details by order ID.
func (o *OKCoin) GetMarginOrder(ctx context.Context, request GetSpotOrderRequest) (GetSpotOrderResponse, error) {
	requestURL := fmt.Sprintf("%v/%v%v", orders, request.OrderID, FormatParameters(request))
	var resp GetSpotOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, requestURL, request, &resp, true)
}

// GetMarginTransactionDetails Get details of the recent filled orders. Cursor pagination is used.
// All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
func (o *OKCoin) GetMarginTransactionDetails(ctx context.Context, request GetSpotTransactionDetailsRequest) ([]GetSpotTransactionDetailsResponse, error) {
	requestURL := fmt.Sprintf("%v%v", getSpotTransactionDetails, FormatParameters(request))
	var resp []GetSpotTransactionDetailsResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, requestURL, nil, &resp, true)
}

// FormatParameters Formats URL parameters, useful for optional parameters due to OKCoin signature check
func FormatParameters(request interface{}) (parameters string) {
	v, err := query.Values(request)
	if err != nil {
		log.Errorf(log.ExchangeSys, "Could not parse %v to URL values. Check that the type has url fields", reflect.TypeOf(request).Name())
		return
	}
	if urlEncodedValues := v.Encode(); len(urlEncodedValues) > 0 {
		parameters = fmt.Sprintf("?%v", urlEncodedValues)
	}
	return
}

// GetErrorCode returns an error code
func (o *OKCoin) GetErrorCode(code interface{}) error {
	var assertedCode string

	switch d := code.(type) {
	case float64:
		assertedCode = strconv.FormatFloat(d, 'f', -1, 64)
	case string:
		assertedCode = d
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
func (o *OKCoin) SendHTTPRequest(ctx context.Context, ep exchange.URL, httpMethod, requestType, requestPath string, data, result interface{}, authenticated bool) error {
	endpoint, err := o.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	var intermediary json.RawMessage
	newRequest := func() (*request.Item, error) {
		utcTime := time.Now().UTC().Format(time.RFC3339)
		payload := []byte("")

		if data != nil {
			payload, err = json.Marshal(data)
			if err != nil {
				return nil, err
			}
		}

		path := endpoint + requestType + okCoinAPIVersion + requestPath
		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		if authenticated {
			var creds *account.Credentials
			creds, err = o.GetCredentials(ctx)
			if err != nil {
				return nil, err
			}
			signPath := fmt.Sprintf("/%v%v%v%v", apiPath,
				requestType, okCoinAPIVersion, requestPath)

			var hmac []byte
			hmac, err = crypto.GetHMAC(crypto.HashSHA256,
				[]byte(utcTime+httpMethod+signPath+string(payload)),
				[]byte(creds.Secret))
			if err != nil {
				return nil, err
			}
			headers["OK-ACCESS-KEY"] = creds.Key
			headers["OK-ACCESS-SIGN"] = crypto.Base64Encode(hmac)
			headers["OK-ACCESS-TIMESTAMP"] = utcTime
			headers["OK-ACCESS-PASSPHRASE"] = creds.ClientID
		}

		return &request.Item{
			Method:        strings.ToUpper(httpMethod),
			Path:          path,
			Headers:       headers,
			Body:          bytes.NewBuffer(payload),
			Result:        &intermediary,
			AuthRequest:   authenticated,
			Verbose:       o.Verbose,
			HTTPDebugging: o.HTTPDebugging,
			HTTPRecording: o.HTTPRecording,
		}, nil
	}

	err = o.SendPayload(ctx, request.Unset, newRequest)
	if err != nil {
		return err
	}

	type errCapFormat struct {
		Error        int64  `json:"error_code,omitempty"`
		ErrorMessage string `json:"error_message,omitempty"`
		Result       bool   `json:"result,string,omitempty"`
	}
	errCap := errCapFormat{Result: true}

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

// GetFee returns an estimate of fee based on type of transaction
func (o *OKCoin) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount, feeBuilder.IsMaker)
	case exchange.CryptocurrencyWithdrawalFee:
		withdrawFees, err := o.GetAccountWithdrawalFee(ctx, feeBuilder.FiatCurrency.String())
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
