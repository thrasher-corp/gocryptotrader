package poloniex

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	poloniexAPIURL               = "https://poloniex.com"
	tradeSpot                    = "/trade/"
	tradeFutures                 = "/futures" + tradeSpot
	poloniexAltAPIUrl            = "https://api.poloniex.com"
	poloniexAPITradingEndpoint   = "tradingApi"
	poloniexAPIVersion           = "1"
	poloniexBalances             = "returnBalances"
	poloniexBalancesComplete     = "returnCompleteBalances"
	poloniexDepositAddresses     = "returnDepositAddresses"
	poloniexGenerateNewAddress   = "generateNewAddress"
	poloniexDepositsWithdrawals  = "returnDepositsWithdrawals"
	poloniexOrders               = "returnOpenOrders"
	poloniexTradeHistory         = "returnTradeHistory"
	poloniexOrderTrades          = "returnOrderTrades"
	poloniexOrderStatus          = "returnOrderStatus"
	poloniexOrderCancel          = "cancelOrder"
	poloniexOrderMove            = "moveOrder"
	poloniexWithdraw             = "withdraw"
	poloniexFeeInfo              = "returnFeeInfo"
	poloniexAvailableBalances    = "returnAvailableAccountBalances"
	poloniexTradableBalances     = "returnTradableBalances"
	poloniexTransferBalance      = "transferBalance"
	poloniexMarginAccountSummary = "returnMarginAccountSummary"
	poloniexMarginBuy            = "marginBuy"
	poloniexMarginSell           = "marginSell"
	poloniexMarginPosition       = "getMarginPosition"
	poloniexMarginPositionClose  = "closeMarginPosition"
	poloniexCreateLoanOffer      = "createLoanOffer"
	poloniexCancelLoanOffer      = "cancelLoanOffer"
	poloniexOpenLoanOffers       = "returnOpenLoanOffers"
	poloniexActiveLoans          = "returnActiveLoans"
	poloniexLendingHistory       = "returnLendingHistory"
	poloniexAutoRenew            = "toggleAutoRenew"
	poloniexCancelByIDs          = "/orders/cancelByIds"
	poloniexTimestamp            = "/timestamp"
	poloniexWalletActivity       = "/wallets/activity"
	poloniexMaxOrderbookDepth    = 100
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Poloniex
type Exchange struct {
	exchange.Base
	details CurrencyDetails
}

// GetTicker returns current ticker information
func (e *Exchange) GetTicker(ctx context.Context) (map[string]Ticker, error) {
	type response struct {
		Data map[string]Ticker
	}

	resp := response{}
	path := "/public?command=returnTicker"

	return resp.Data, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp.Data)
}

// GetVolume returns a list of currencies with associated volume
func (e *Exchange) GetVolume(ctx context.Context) (any, error) {
	var resp any
	path := "/public?command=return24hVolume"

	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetOrderbook returns the full orderbook from poloniex
func (e *Exchange) GetOrderbook(ctx context.Context, currencyPair string, depth int) (OrderbookAll, error) {
	vals := url.Values{}

	if depth != 0 {
		vals.Set("depth", strconv.Itoa(depth))
	}

	oba := OrderbookAll{Data: make(map[string]Orderbook)}
	if currencyPair != "" {
		vals.Set("currencyPair", currencyPair)
		resp := OrderbookResponse{}
		path := "/public?command=returnOrderBook&" + vals.Encode()
		if err := e.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp); err != nil {
			return oba, err
		}
		if resp.Error != "" {
			return oba, fmt.Errorf("%s GetOrderbook() error: %s", e.Name, resp.Error)
		}
		oba.Data[currencyPair] = Orderbook{
			Bids: resp.Bids.Levels(),
			Asks: resp.Asks.Levels(),
		}
	} else {
		vals.Set("currencyPair", "all")
		resp := OrderbookResponseAll{}
		path := "/public?command=returnOrderBook&" + vals.Encode()
		if err := e.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp.Data); err != nil {
			return oba, err
		}
		for currency, orderbook := range resp.Data {
			oba.Data[currency] = Orderbook{
				Bids: orderbook.Bids.Levels(),
				Asks: orderbook.Asks.Levels(),
			}
		}
	}
	return oba, nil
}

// GetTradeHistory returns trades history from poloniex
func (e *Exchange) GetTradeHistory(ctx context.Context, currencyPair string, start, end int64) ([]TradeHistory, error) {
	vals := url.Values{}
	vals.Set("currencyPair", currencyPair)

	if start > 0 {
		vals.Set("start", strconv.FormatInt(start, 10))
	}

	if end > 0 {
		vals.Set("end", strconv.FormatInt(end, 10))
	}

	var resp []TradeHistory
	path := "/public?command=returnTradeHistory&" + vals.Encode()
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetChartData returns chart data for a specific currency pair
func (e *Exchange) GetChartData(ctx context.Context, currencyPair string, start, end time.Time, period string) ([]ChartData, error) {
	vals := url.Values{}
	vals.Set("currencyPair", currencyPair)

	if !start.IsZero() {
		vals.Set("start", strconv.FormatInt(start.Unix(), 10))
	}

	if !end.IsZero() {
		vals.Set("end", strconv.FormatInt(end.Unix(), 10))
	}

	if period != "" {
		vals.Set("period", period)
	}

	var temp json.RawMessage
	path := "/public?command=returnChartData&" + vals.Encode()
	err := e.SendHTTPRequest(ctx, exchange.RestSpot, path, &temp)
	if err != nil {
		return nil, err
	}

	var resp []ChartData
	err = json.Unmarshal(temp, &resp)
	if err != nil {
		var errResp struct {
			Error string `json:"error"`
		}
		if errRet := json.Unmarshal(temp, &errResp); errRet != nil {
			return nil, errRet
		}
		if errResp.Error != "" {
			return nil, errors.New(errResp.Error)
		}
	}

	return resp, err
}

// GetCurrencies returns information about currencies
func (e *Exchange) GetCurrencies(ctx context.Context) (map[string]*Currencies, error) {
	type Response struct {
		Data map[string]*Currencies
	}
	resp := Response{}
	return resp.Data, e.SendHTTPRequest(ctx,
		exchange.RestSpot,
		"/public?command=returnCurrencies&includeMultiChainCurrencies=true",
		&resp.Data,
	)
}

// GetTimestamp returns server time
func (e *Exchange) GetTimestamp(ctx context.Context) (time.Time, error) {
	var resp TimeStampResponse
	err := e.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		poloniexTimestamp,
		&resp,
	)
	if err != nil {
		return time.Time{}, err
	}
	return resp.ServerTime.Time(), nil
}

// GetLoanOrders returns the list of loan offers and demands for a given
// currency, specified by the "currency" GET parameter.
func (e *Exchange) GetLoanOrders(ctx context.Context, ccy string) (LoanOrders, error) {
	resp := LoanOrders{}
	path := "/public?command=returnLoanOrders&currency=" + ccy
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetBalances returns balances for your account.
func (e *Exchange) GetBalances(ctx context.Context) (map[currency.Code]float64, error) {
	var result any
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexBalances, url.Values{}, &result); err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, common.GetTypeAssertError("map[string]any", result, "balance result")
	}

	bals := make(map[currency.Code]float64, len(data))

	for x, y := range data {
		bal, ok := y.(string)
		if !ok {
			return nil, common.GetTypeAssertError("string", y, "balance amount")
		}

		var err error
		if bals[currency.NewCode(x)], err = strconv.ParseFloat(bal, 64); err != nil {
			return nil, err
		}
	}

	return bals, nil
}

// GetCompleteBalances returns complete balances from your account.
func (e *Exchange) GetCompleteBalances(ctx context.Context) (CompleteBalances, error) {
	var result CompleteBalances
	vals := url.Values{}
	vals.Set("account", "all")
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot,
		http.MethodPost,
		poloniexBalancesComplete,
		vals,
		&result)
	return result, err
}

// GetDepositAddresses returns deposit addresses for all enabled cryptos.
func (e *Exchange) GetDepositAddresses(ctx context.Context) (DepositAddresses, error) {
	var result any
	addresses := DepositAddresses{}

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexDepositAddresses, url.Values{}, &result)
	if err != nil {
		return addresses, err
	}

	addresses.Addresses = make(map[string]string)
	data, ok := result.(map[string]any)
	if !ok {
		return addresses, errors.New("return val not map[string]any")
	}

	for x, y := range data {
		addresses.Addresses[x], ok = y.(string)
		if !ok {
			return addresses, common.GetTypeAssertError("string", y, "address")
		}
	}

	return addresses, nil
}

// GenerateNewAddress generates a new address for a currency
func (e *Exchange) GenerateNewAddress(ctx context.Context, curr string) (string, error) {
	type Response struct {
		Success  int
		Error    string
		Response string
	}
	resp := Response{}
	values := url.Values{}
	values.Set("currency", curr)

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexGenerateNewAddress, values, &resp)
	if err != nil {
		return "", err
	}

	if resp.Error != "" {
		return "", errors.New(resp.Error)
	}

	return resp.Response, nil
}

// GetDepositsWithdrawals returns a list of deposits and withdrawals
func (e *Exchange) GetDepositsWithdrawals(ctx context.Context, start, end string) (DepositsWithdrawals, error) {
	resp := DepositsWithdrawals{}
	values := url.Values{}

	if start != "" {
		values.Set("start", start)
	} else {
		values.Set("start", "0")
	}

	if end != "" {
		values.Set("end", end)
	} else {
		values.Set("end", strconv.FormatInt(time.Now().Unix(), 10))
	}

	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexDepositsWithdrawals, values, &resp)
}

// GetOpenOrders returns current unfilled opened orders
func (e *Exchange) GetOpenOrders(ctx context.Context, ccy string) (OpenOrdersResponse, error) {
	values := url.Values{}
	values.Set("currencyPair", ccy)
	result := OpenOrdersResponse{}
	return result, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexOrders, values, &result.Data)
}

// GetOpenOrdersForAllCurrencies returns all open orders
func (e *Exchange) GetOpenOrdersForAllCurrencies(ctx context.Context) (OpenOrdersResponseAll, error) {
	values := url.Values{}
	values.Set("currencyPair", "all")
	result := OpenOrdersResponseAll{}
	return result, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexOrders, values, &result.Data)
}

// GetAuthenticatedTradeHistoryForCurrency returns account trade history
func (e *Exchange) GetAuthenticatedTradeHistoryForCurrency(ctx context.Context, currencyPair string, start, end, limit int64) (AuthenticatedTradeHistoryResponse, error) {
	values := url.Values{}

	if start > 0 {
		values.Set("start", strconv.FormatInt(start, 10))
	}

	if limit > 0 {
		values.Set("limit", strconv.FormatInt(limit, 10))
	}

	if end > 0 {
		values.Set("end", strconv.FormatInt(end, 10))
	}

	values.Set("currencyPair", currencyPair)
	result := AuthenticatedTradeHistoryResponse{}
	return result, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexTradeHistory, values, &result.Data)
}

// GetAuthenticatedTradeHistory returns account trade history
func (e *Exchange) GetAuthenticatedTradeHistory(ctx context.Context, start, end, limit int64) (AuthenticatedTradeHistoryAll, error) {
	values := url.Values{}

	if start > 0 {
		values.Set("start", strconv.FormatInt(start, 10))
	}

	if limit > 0 {
		values.Set("limit", strconv.FormatInt(limit, 10))
	}

	if end > 0 {
		values.Set("end", strconv.FormatInt(end, 10))
	}

	values.Set("currencyPair", "all")
	var result json.RawMessage

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexTradeHistory, values, &result)
	if err != nil {
		return AuthenticatedTradeHistoryAll{}, err
	}

	var nodata []any
	err = json.Unmarshal(result, &nodata)
	if err == nil {
		return AuthenticatedTradeHistoryAll{}, nil
	}

	var mainResult AuthenticatedTradeHistoryAll
	return mainResult, json.Unmarshal(result, &mainResult.Data)
}

// GetAuthenticatedOrderStatus returns the status of a given orderId.
func (e *Exchange) GetAuthenticatedOrderStatus(ctx context.Context, orderID string) (o OrderStatusData, err error) {
	values := url.Values{}

	if orderID == "" {
		return o, errors.New("no orderID passed")
	}

	values.Set("orderNumber", orderID)
	var rawOrderStatus OrderStatus
	err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexOrderStatus, values, &rawOrderStatus)
	if err != nil {
		return o, err
	}

	switch rawOrderStatus.Success {
	case 0: // fail
		var errMsg GenericResponse
		err = json.Unmarshal(rawOrderStatus.Result, &errMsg)
		if err != nil {
			return o, err
		}
		return o, errors.New(errMsg.Error)
	case 1: // success
		var status map[string]OrderStatusData
		err = json.Unmarshal(rawOrderStatus.Result, &status)
		if err != nil {
			return o, err
		}

		for _, o = range status {
			return o, err
		}
	}

	return o, err
}

// GetAuthenticatedOrderTrades returns all trades involving a given orderId.
func (e *Exchange) GetAuthenticatedOrderTrades(ctx context.Context, orderID string) (o []OrderTrade, err error) {
	values := url.Values{}

	if orderID == "" {
		return nil, errors.New("no orderID passed")
	}

	values.Set("orderNumber", orderID)
	var result json.RawMessage
	err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexOrderTrades, values, &result)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, errors.New("received unexpected response")
	}

	switch result[0] {
	case '{': // error message received
		var resp GenericResponse
		err = json.Unmarshal(result, &resp)
		if err != nil {
			return nil, err
		}
		if resp.Error != "" {
			err = errors.New(resp.Error)
		}
	case '[': // data received
		err = json.Unmarshal(result, &o)
	default:
		return nil, errors.New("received unexpected response")
	}

	return o, err
}

// PlaceOrder places a new order on the exchange
func (e *Exchange) PlaceOrder(ctx context.Context, currencyPair string, rate, amount float64, immediate, fillOrKill, buy bool) (OrderResponse, error) {
	result := OrderResponse{}
	values := url.Values{}

	var orderType string
	if buy {
		orderType = order.Buy.Lower()
	} else {
		orderType = order.Sell.Lower()
	}

	values.Set("currencyPair", currencyPair)
	values.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	if immediate {
		values.Set("immediateOrCancel", "1")
	}

	if fillOrKill {
		values.Set("fillOrKill", "1")
	}

	return result, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, orderType, values, &result)
}

// CancelExistingOrder cancels and order by orderID
func (e *Exchange) CancelExistingOrder(ctx context.Context, orderID int64) error {
	result := GenericResponse{}
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderID, 10))

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexOrderCancel, values, &result)
	if err != nil {
		return err
	}

	if result.Success != 1 {
		return errors.New(result.Error)
	}

	return nil
}

// MoveOrder moves an order
func (e *Exchange) MoveOrder(ctx context.Context, orderID int64, rate, amount float64, postOnly, immediateOrCancel bool) (MoveOrderResponse, error) {
	result := MoveOrderResponse{}
	values := url.Values{}

	if orderID == 0 {
		return result, errors.New("orderID cannot be zero")
	}

	if rate == 0 {
		return result, errors.New("rate cannot be zero")
	}

	values.Set("orderNumber", strconv.FormatInt(orderID, 10))
	values.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))

	if postOnly {
		values.Set("postOnly", "true")
	}

	if immediateOrCancel {
		values.Set("immediateOrCancel", "true")
	}

	if amount != 0 {
		values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		poloniexOrderMove,
		values,
		&result)
	if err != nil {
		return result, err
	}

	if result.Success != 1 {
		return result, errors.New(result.Error)
	}

	return result, nil
}

// Withdraw withdraws a currency to a specific delegated address.
// For currencies where there are multiple networks to choose from (like USDT or BTC),
// you can specify the chain by setting the "currency" parameter to be a multiChain currency
// name, like USDTTRON, USDTETH, or BTCTRON
func (e *Exchange) Withdraw(ctx context.Context, ccy, address string, amount float64) (*Withdraw, error) {
	result := &Withdraw{}
	values := url.Values{}

	values.Set("currency", strings.ToUpper(ccy))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("address", address)

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexWithdraw, values, &result)
	if err != nil {
		return nil, err
	}

	if result.Error != "" {
		return nil, errors.New(result.Error)
	}

	return result, nil
}

// GetFeeInfo returns fee information
func (e *Exchange) GetFeeInfo(ctx context.Context) (Fee, error) {
	result := Fee{}

	return result, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexFeeInfo, url.Values{}, &result)
}

// GetTradableBalances returns tradable balances
func (e *Exchange) GetTradableBalances(ctx context.Context) (map[string]map[string]float64, error) {
	type Response struct {
		Data map[string]map[string]any
	}
	result := Response{}

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexTradableBalances, url.Values{}, &result.Data)
	if err != nil {
		return nil, err
	}

	balances := make(map[string]map[string]float64)

	for x, y := range result.Data {
		balances[x] = make(map[string]float64)
		for z, w := range y {
			bal, ok := w.(string)
			if !ok {
				return nil, common.GetTypeAssertError("string", w, "balance")
			}
			balances[x][z], err = strconv.ParseFloat(bal, 64)
			if err != nil {
				return nil, err
			}
		}
	}

	return balances, nil
}

// TransferBalance transfers balances between your accounts
func (e *Exchange) TransferBalance(ctx context.Context, ccy, from, to string, amount float64) (bool, error) {
	values := url.Values{}
	result := GenericResponse{}

	values.Set("currency", ccy)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("fromAccount", from)
	values.Set("toAccount", to)

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexTransferBalance, values, &result)
	if err != nil {
		return false, err
	}

	if result.Error != "" && result.Success != 1 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// GetMarginAccountSummary returns a summary on your margin accounts
func (e *Exchange) GetMarginAccountSummary(ctx context.Context) (Margin, error) {
	result := Margin{}
	return result, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexMarginAccountSummary, url.Values{}, &result)
}

// PlaceMarginOrder places a margin order
func (e *Exchange) PlaceMarginOrder(ctx context.Context, currencyPair string, rate, amount, lendingRate float64, buy bool) (OrderResponse, error) {
	result := OrderResponse{}
	values := url.Values{}

	var orderType string
	if buy {
		orderType = poloniexMarginBuy
	} else {
		orderType = poloniexMarginSell
	}

	values.Set("currencyPair", currencyPair)
	values.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	if lendingRate != 0 {
		values.Set("lendingRate", strconv.FormatFloat(lendingRate, 'f', -1, 64))
	}

	return result, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, orderType, values, &result)
}

// GetMarginPosition returns a position on a margin order
func (e *Exchange) GetMarginPosition(ctx context.Context, currencyPair string) (any, error) {
	values := url.Values{}

	if currencyPair != "" && currencyPair != "all" {
		values.Set("currencyPair", currencyPair)
		result := MarginPosition{}
		return result, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexMarginPosition, values, &result)
	}
	values.Set("currencyPair", "all")

	type Response struct {
		Data map[string]MarginPosition
	}
	result := Response{}
	return result, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexMarginPosition, values, &result.Data)
}

// CloseMarginPosition closes a current margin position
func (e *Exchange) CloseMarginPosition(ctx context.Context, currencyPair string) (bool, error) {
	values := url.Values{}
	values.Set("currencyPair", currencyPair)
	result := GenericResponse{}

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexMarginPositionClose, values, &result)
	if err != nil {
		return false, err
	}

	if result.Success == 0 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// CreateLoanOffer places a loan offer on the exchange
func (e *Exchange) CreateLoanOffer(ctx context.Context, ccy string, amount, rate float64, duration int, autoRenew bool) (int64, error) {
	values := url.Values{}
	values.Set("currency", ccy)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("duration", strconv.Itoa(duration))

	if autoRenew {
		values.Set("autoRenew", "1")
	} else {
		values.Set("autoRenew", "0")
	}

	values.Set("lendingRate", strconv.FormatFloat(rate, 'f', -1, 64))

	type Response struct {
		Success int    `json:"success"`
		Error   string `json:"error"`
		OrderID int64  `json:"orderID"`
	}

	result := Response{}

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexCreateLoanOffer, values, &result)
	if err != nil {
		return 0, err
	}

	if result.Success == 0 {
		return 0, errors.New(result.Error)
	}

	return result.OrderID, nil
}

// CancelLoanOffer cancels a loan offer order
func (e *Exchange) CancelLoanOffer(ctx context.Context, orderNumber int64) (bool, error) {
	result := GenericResponse{}
	values := url.Values{}
	values.Set("orderID", strconv.FormatInt(orderNumber, 10))

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexCancelLoanOffer, values, &result)
	if err != nil {
		return false, err
	}

	if result.Success == 0 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// GetOpenLoanOffers returns all open loan offers
func (e *Exchange) GetOpenLoanOffers(ctx context.Context) (map[string][]LoanOffer, error) {
	type Response struct {
		Data map[string][]LoanOffer
	}
	result := Response{}

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexOpenLoanOffers, url.Values{}, &result.Data)
	if err != nil {
		return nil, err
	}

	if result.Data == nil {
		return nil, errors.New("there are no open loan offers")
	}

	return result.Data, nil
}

// GetActiveLoans returns active loans
func (e *Exchange) GetActiveLoans(ctx context.Context) (ActiveLoans, error) {
	result := ActiveLoans{}
	return result, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexActiveLoans, url.Values{}, &result)
}

// GetLendingHistory returns lending history for the account
func (e *Exchange) GetLendingHistory(ctx context.Context, start, end string) ([]LendingHistory, error) {
	vals := url.Values{}

	if start != "" {
		vals.Set("start", start)
	}

	if end != "" {
		vals.Set("end", end)
	}

	var resp []LendingHistory
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		poloniexLendingHistory,
		vals,
		&resp)
}

// ToggleAutoRenew allows for the autorenew of a contract
func (e *Exchange) ToggleAutoRenew(ctx context.Context, orderNumber int64) (bool, error) {
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderNumber, 10))
	result := GenericResponse{}

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		poloniexAutoRenew,
		values,
		&result)
	if err != nil {
		return false, err
	}

	if result.Success == 0 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// WalletActivity returns the wallet activity between set start and end time
func (e *Exchange) WalletActivity(ctx context.Context, start, end time.Time, activityType string) (*WalletActivityResponse, error) {
	values := url.Values{}
	err := common.StartEndTimeCheck(start, end)
	if err != nil {
		return nil, err
	}
	values.Set("start", strconv.FormatInt(start.Unix(), 10))
	values.Set("end", strconv.FormatInt(end.Unix(), 10))
	if activityType != "" {
		values.Set("activityType", activityType)
	}
	var resp WalletActivityResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		poloniexWalletActivity,
		values,
		&resp)
}

// CancelMultipleOrdersByIDs Batch cancel one or many smart orders in an account by IDs.
func (e *Exchange) CancelMultipleOrdersByIDs(ctx context.Context, orderIDs, clientOrderIDs []string) ([]CancelOrdersResponse, error) {
	values := url.Values{}
	if len(orderIDs) > 0 {
		values.Set("orderIds", strings.Join(orderIDs, ","))
	}
	if len(clientOrderIDs) > 0 {
		values.Set("clientOrderIds", strings.Join(clientOrderIDs, ","))
	}
	var result []CancelOrdersResponse
	return result, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete,
		poloniexCancelByIDs,
		values,
		&result)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	item := &request.Item{
		Method:                 http.MethodGet,
		Path:                   endpoint + path,
		Result:                 result,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}

	return e.SendPayload(ctx, request.UnAuth, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, endpoint string, values url.Values, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	ePoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	return e.SendPayload(ctx, request.Auth, func() (*request.Item, error) {
		headers := make(map[string]string)
		headers["Content-Type"] = "application/x-www-form-urlencoded"
		headers["Key"] = creds.Key
		values.Set("nonce", e.Requester.GetNonce(nonce.UnixNano).String())
		values.Set("command", endpoint)

		hmac, err := crypto.GetHMAC(crypto.HashSHA512, []byte(values.Encode()), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		headers["Sign"] = hex.EncodeToString(hmac)

		path := fmt.Sprintf("%s/%s", ePoint, poloniexAPITradingEndpoint)
		return &request.Item{
			Method:                 method,
			Path:                   path,
			Headers:                headers,
			Body:                   bytes.NewBufferString(values.Encode()),
			Result:                 result,
			NonceEnabled:           true,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.AuthenticatedRequest)
}

// GetFee returns an estimate of fee based on type of transaction
func (e *Exchange) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		feeInfo, err := e.GetFeeInfo(ctx)
		if err != nil {
			return 0, err
		}
		fee = calculateTradingFee(feeInfo,
			feeBuilder.PurchasePrice,
			feeBuilder.Amount,
			feeBuilder.IsMaker)

	case exchange.CryptocurrencyWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.Pair.Base)
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
	return 0.002 * price * amount
}

func calculateTradingFee(feeInfo Fee, purchasePrice, amount float64, isMaker bool) (fee float64) {
	if isMaker {
		fee = feeInfo.MakerFee
	} else {
		fee = feeInfo.TakerFee
	}
	return fee * amount * purchasePrice
}

func getWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}
