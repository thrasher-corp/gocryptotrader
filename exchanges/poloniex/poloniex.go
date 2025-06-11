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

// Poloniex is the overarching type across the poloniex package
type Poloniex struct {
	exchange.Base
	details CurrencyDetails
}

// GetTicker returns current ticker information
func (p *Poloniex) GetTicker(ctx context.Context) (map[string]Ticker, error) {
	type response struct {
		Data map[string]Ticker
	}

	resp := response{}
	path := "/public?command=returnTicker"

	return resp.Data, p.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp.Data)
}

// GetVolume returns a list of currencies with associated volume
func (p *Poloniex) GetVolume(ctx context.Context) (any, error) {
	var resp any
	path := "/public?command=return24hVolume"

	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetOrderbook returns the full orderbook from poloniex
func (p *Poloniex) GetOrderbook(ctx context.Context, currencyPair string, depth int) (OrderbookAll, error) {
	vals := url.Values{}

	if depth != 0 {
		vals.Set("depth", strconv.Itoa(depth))
	}

	oba := OrderbookAll{Data: make(map[string]Orderbook)}
	if currencyPair != "" {
		vals.Set("currencyPair", currencyPair)
		resp := OrderbookResponse{}
		path := "/public?command=returnOrderBook&" + vals.Encode()
		err := p.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
		if err != nil {
			return oba, err
		}
		if resp.Error != "" {
			return oba, fmt.Errorf("%s GetOrderbook() error: %s", p.Name, resp.Error)
		}
		ob := Orderbook{
			Bids: make([]OrderbookItem, len(resp.Bids)),
			Asks: make([]OrderbookItem, len(resp.Asks)),
		}
		for x := range resp.Asks {
			ob.Asks[x] = OrderbookItem{
				Price:  resp.Asks[x][0].Float64(),
				Amount: resp.Asks[x][1].Float64(),
			}
		}
		for x := range resp.Bids {
			ob.Bids[x] = OrderbookItem{
				Price:  resp.Bids[x][0].Float64(),
				Amount: resp.Bids[x][1].Float64(),
			}
		}
		oba.Data[currencyPair] = ob
	} else {
		vals.Set("currencyPair", "all")
		resp := OrderbookResponseAll{}
		path := "/public?command=returnOrderBook&" + vals.Encode()
		err := p.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp.Data)
		if err != nil {
			return oba, err
		}
		for currency, orderbook := range resp.Data {
			ob := Orderbook{
				Bids: make([]OrderbookItem, len(orderbook.Bids)),
				Asks: make([]OrderbookItem, len(orderbook.Asks)),
			}
			for x := range orderbook.Asks {
				ob.Asks[x] = OrderbookItem{
					Price:  orderbook.Asks[x][0].Float64(),
					Amount: orderbook.Asks[x][1].Float64(),
				}
			}
			for x := range orderbook.Bids {
				ob.Bids[x] = OrderbookItem{
					Price:  orderbook.Bids[x][0].Float64(),
					Amount: orderbook.Bids[x][1].Float64(),
				}
			}
			oba.Data[currency] = ob
		}
	}
	return oba, nil
}

// GetTradeHistory returns trades history from poloniex
func (p *Poloniex) GetTradeHistory(ctx context.Context, currencyPair string, start, end int64) ([]TradeHistory, error) {
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
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetChartData returns chart data for a specific currency pair
func (p *Poloniex) GetChartData(ctx context.Context, currencyPair string, start, end time.Time, period string) ([]ChartData, error) {
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
	err := p.SendHTTPRequest(ctx, exchange.RestSpot, path, &temp)
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
func (p *Poloniex) GetCurrencies(ctx context.Context) (map[string]*Currencies, error) {
	type Response struct {
		Data map[string]*Currencies
	}
	resp := Response{}
	return resp.Data, p.SendHTTPRequest(ctx,
		exchange.RestSpot,
		"/public?command=returnCurrencies&includeMultiChainCurrencies=true",
		&resp.Data,
	)
}

// GetTimestamp returns server time
func (p *Poloniex) GetTimestamp(ctx context.Context) (time.Time, error) {
	var resp TimeStampResponse
	err := p.SendHTTPRequest(ctx,
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
func (p *Poloniex) GetLoanOrders(ctx context.Context, currency string) (LoanOrders, error) {
	resp := LoanOrders{}
	path := "/public?command=returnLoanOrders&currency=" + currency
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetBalances returns balances for your account.
func (p *Poloniex) GetBalances(ctx context.Context) (Balance, error) {
	var result any
	if err := p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexBalances, url.Values{}, &result); err != nil {
		return Balance{}, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return Balance{}, common.GetTypeAssertError("map[string]any", result, "balance result")
	}

	balance := Balance{
		Currency: make(map[string]float64),
	}

	for x, y := range data {
		bal, ok := y.(string)
		if !ok {
			return Balance{}, common.GetTypeAssertError("string", y, "balance amount")
		}

		var err error
		balance.Currency[x], err = strconv.ParseFloat(bal, 64)
		if err != nil {
			return Balance{}, err
		}
	}

	return balance, nil
}

// GetCompleteBalances returns complete balances from your account.
func (p *Poloniex) GetCompleteBalances(ctx context.Context) (CompleteBalances, error) {
	var result CompleteBalances
	vals := url.Values{}
	vals.Set("account", "all")
	err := p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot,
		http.MethodPost,
		poloniexBalancesComplete,
		vals,
		&result)
	return result, err
}

// GetDepositAddresses returns deposit addresses for all enabled cryptos.
func (p *Poloniex) GetDepositAddresses(ctx context.Context) (DepositAddresses, error) {
	var result any
	addresses := DepositAddresses{}

	err := p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexDepositAddresses, url.Values{}, &result)
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
func (p *Poloniex) GenerateNewAddress(ctx context.Context, curr string) (string, error) {
	type Response struct {
		Success  int
		Error    string
		Response string
	}
	resp := Response{}
	values := url.Values{}
	values.Set("currency", curr)

	err := p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexGenerateNewAddress, values, &resp)
	if err != nil {
		return "", err
	}

	if resp.Error != "" {
		return "", errors.New(resp.Error)
	}

	return resp.Response, nil
}

// GetDepositsWithdrawals returns a list of deposits and withdrawals
func (p *Poloniex) GetDepositsWithdrawals(ctx context.Context, start, end string) (DepositsWithdrawals, error) {
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

	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexDepositsWithdrawals, values, &resp)
}

// GetOpenOrders returns current unfilled opened orders
func (p *Poloniex) GetOpenOrders(ctx context.Context, currency string) (OpenOrdersResponse, error) {
	values := url.Values{}
	values.Set("currencyPair", currency)
	result := OpenOrdersResponse{}
	return result, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexOrders, values, &result.Data)
}

// GetOpenOrdersForAllCurrencies returns all open orders
func (p *Poloniex) GetOpenOrdersForAllCurrencies(ctx context.Context) (OpenOrdersResponseAll, error) {
	values := url.Values{}
	values.Set("currencyPair", "all")
	result := OpenOrdersResponseAll{}
	return result, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexOrders, values, &result.Data)
}

// GetAuthenticatedTradeHistoryForCurrency returns account trade history
func (p *Poloniex) GetAuthenticatedTradeHistoryForCurrency(ctx context.Context, currency string, start, end, limit int64) (AuthenticatedTradeHistoryResponse, error) {
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

	values.Set("currencyPair", currency)
	result := AuthenticatedTradeHistoryResponse{}
	return result, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexTradeHistory, values, &result.Data)
}

// GetAuthenticatedTradeHistory returns account trade history
func (p *Poloniex) GetAuthenticatedTradeHistory(ctx context.Context, start, end, limit int64) (AuthenticatedTradeHistoryAll, error) {
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

	err := p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexTradeHistory, values, &result)
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
func (p *Poloniex) GetAuthenticatedOrderStatus(ctx context.Context, orderID string) (o OrderStatusData, err error) {
	values := url.Values{}

	if orderID == "" {
		return o, errors.New("no orderID passed")
	}

	values.Set("orderNumber", orderID)
	var rawOrderStatus OrderStatus
	err = p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexOrderStatus, values, &rawOrderStatus)
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
func (p *Poloniex) GetAuthenticatedOrderTrades(ctx context.Context, orderID string) (o []OrderTrade, err error) {
	values := url.Values{}

	if orderID == "" {
		return nil, errors.New("no orderID passed")
	}

	values.Set("orderNumber", orderID)
	var result json.RawMessage
	err = p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexOrderTrades, values, &result)
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
func (p *Poloniex) PlaceOrder(ctx context.Context, currency string, rate, amount float64, immediate, fillOrKill, buy bool) (OrderResponse, error) {
	result := OrderResponse{}
	values := url.Values{}

	var orderType string
	if buy {
		orderType = order.Buy.Lower()
	} else {
		orderType = order.Sell.Lower()
	}

	values.Set("currencyPair", currency)
	values.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	if immediate {
		values.Set("immediateOrCancel", "1")
	}

	if fillOrKill {
		values.Set("fillOrKill", "1")
	}

	return result, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, orderType, values, &result)
}

// CancelExistingOrder cancels and order by orderID
func (p *Poloniex) CancelExistingOrder(ctx context.Context, orderID int64) error {
	result := GenericResponse{}
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderID, 10))

	err := p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexOrderCancel, values, &result)
	if err != nil {
		return err
	}

	if result.Success != 1 {
		return errors.New(result.Error)
	}

	return nil
}

// MoveOrder moves an order
func (p *Poloniex) MoveOrder(ctx context.Context, orderID int64, rate, amount float64, postOnly, immediateOrCancel bool) (MoveOrderResponse, error) {
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

	err := p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
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
func (p *Poloniex) Withdraw(ctx context.Context, currency, address string, amount float64) (*Withdraw, error) {
	result := &Withdraw{}
	values := url.Values{}

	values.Set("currency", strings.ToUpper(currency))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("address", address)

	err := p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexWithdraw, values, &result)
	if err != nil {
		return nil, err
	}

	if result.Error != "" {
		return nil, errors.New(result.Error)
	}

	return result, nil
}

// GetFeeInfo returns fee information
func (p *Poloniex) GetFeeInfo(ctx context.Context) (Fee, error) {
	result := Fee{}

	return result, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexFeeInfo, url.Values{}, &result)
}

// GetTradableBalances returns tradable balances
func (p *Poloniex) GetTradableBalances(ctx context.Context) (map[string]map[string]float64, error) {
	type Response struct {
		Data map[string]map[string]any
	}
	result := Response{}

	err := p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexTradableBalances, url.Values{}, &result.Data)
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
func (p *Poloniex) TransferBalance(ctx context.Context, currency, from, to string, amount float64) (bool, error) {
	values := url.Values{}
	result := GenericResponse{}

	values.Set("currency", currency)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("fromAccount", from)
	values.Set("toAccount", to)

	err := p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexTransferBalance, values, &result)
	if err != nil {
		return false, err
	}

	if result.Error != "" && result.Success != 1 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// GetMarginAccountSummary returns a summary on your margin accounts
func (p *Poloniex) GetMarginAccountSummary(ctx context.Context) (Margin, error) {
	result := Margin{}
	return result, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexMarginAccountSummary, url.Values{}, &result)
}

// PlaceMarginOrder places a margin order
func (p *Poloniex) PlaceMarginOrder(ctx context.Context, currency string, rate, amount, lendingRate float64, buy bool) (OrderResponse, error) {
	result := OrderResponse{}
	values := url.Values{}

	var orderType string
	if buy {
		orderType = poloniexMarginBuy
	} else {
		orderType = poloniexMarginSell
	}

	values.Set("currencyPair", currency)
	values.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	if lendingRate != 0 {
		values.Set("lendingRate", strconv.FormatFloat(lendingRate, 'f', -1, 64))
	}

	return result, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, orderType, values, &result)
}

// GetMarginPosition returns a position on a margin order
func (p *Poloniex) GetMarginPosition(ctx context.Context, currency string) (any, error) {
	values := url.Values{}

	if currency != "" && currency != "all" {
		values.Set("currencyPair", currency)
		result := MarginPosition{}
		return result, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexMarginPosition, values, &result)
	}
	values.Set("currencyPair", "all")

	type Response struct {
		Data map[string]MarginPosition
	}
	result := Response{}
	return result, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexMarginPosition, values, &result.Data)
}

// CloseMarginPosition closes a current margin position
func (p *Poloniex) CloseMarginPosition(ctx context.Context, currency string) (bool, error) {
	values := url.Values{}
	values.Set("currencyPair", currency)
	result := GenericResponse{}

	err := p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexMarginPositionClose, values, &result)
	if err != nil {
		return false, err
	}

	if result.Success == 0 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// CreateLoanOffer places a loan offer on the exchange
func (p *Poloniex) CreateLoanOffer(ctx context.Context, currency string, amount, rate float64, duration int, autoRenew bool) (int64, error) {
	values := url.Values{}
	values.Set("currency", currency)
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

	err := p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexCreateLoanOffer, values, &result)
	if err != nil {
		return 0, err
	}

	if result.Success == 0 {
		return 0, errors.New(result.Error)
	}

	return result.OrderID, nil
}

// CancelLoanOffer cancels a loan offer order
func (p *Poloniex) CancelLoanOffer(ctx context.Context, orderNumber int64) (bool, error) {
	result := GenericResponse{}
	values := url.Values{}
	values.Set("orderID", strconv.FormatInt(orderNumber, 10))

	err := p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexCancelLoanOffer, values, &result)
	if err != nil {
		return false, err
	}

	if result.Success == 0 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// GetOpenLoanOffers returns all open loan offers
func (p *Poloniex) GetOpenLoanOffers(ctx context.Context) (map[string][]LoanOffer, error) {
	type Response struct {
		Data map[string][]LoanOffer
	}
	result := Response{}

	err := p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexOpenLoanOffers, url.Values{}, &result.Data)
	if err != nil {
		return nil, err
	}

	if result.Data == nil {
		return nil, errors.New("there are no open loan offers")
	}

	return result.Data, nil
}

// GetActiveLoans returns active loans
func (p *Poloniex) GetActiveLoans(ctx context.Context) (ActiveLoans, error) {
	result := ActiveLoans{}
	return result, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, poloniexActiveLoans, url.Values{}, &result)
}

// GetLendingHistory returns lending history for the account
func (p *Poloniex) GetLendingHistory(ctx context.Context, start, end string) ([]LendingHistory, error) {
	vals := url.Values{}

	if start != "" {
		vals.Set("start", start)
	}

	if end != "" {
		vals.Set("end", end)
	}

	var resp []LendingHistory
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		poloniexLendingHistory,
		vals,
		&resp)
}

// ToggleAutoRenew allows for the autorenew of a contract
func (p *Poloniex) ToggleAutoRenew(ctx context.Context, orderNumber int64) (bool, error) {
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderNumber, 10))
	result := GenericResponse{}

	err := p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
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
func (p *Poloniex) WalletActivity(ctx context.Context, start, end time.Time, activityType string) (*WalletActivityResponse, error) {
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
	return &resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		poloniexWalletActivity,
		values,
		&resp)
}

// CancelMultipleOrdersByIDs Batch cancel one or many smart orders in an account by IDs.
func (p *Poloniex) CancelMultipleOrdersByIDs(ctx context.Context, orderIDs, clientOrderIDs []string) ([]CancelOrdersResponse, error) {
	values := url.Values{}
	if len(orderIDs) > 0 {
		values.Set("orderIds", strings.Join(orderIDs, ","))
	}
	if len(clientOrderIDs) > 0 {
		values.Set("clientOrderIds", strings.Join(clientOrderIDs, ","))
	}
	var result []CancelOrdersResponse
	return result, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete,
		poloniexCancelByIDs,
		values,
		&result)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (p *Poloniex) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result any) error {
	endpoint, err := p.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       p.Verbose,
		HTTPDebugging: p.HTTPDebugging,
		HTTPRecording: p.HTTPRecording,
	}

	return p.SendPayload(ctx, request.UnAuth, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (p *Poloniex) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, endpoint string, values url.Values, result any) error {
	creds, err := p.GetCredentials(ctx)
	if err != nil {
		return err
	}
	ePoint, err := p.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	return p.SendPayload(ctx, request.Auth, func() (*request.Item, error) {
		headers := make(map[string]string)
		headers["Content-Type"] = "application/x-www-form-urlencoded"
		headers["Key"] = creds.Key
		values.Set("nonce", p.Requester.GetNonce(nonce.UnixNano).String())
		values.Set("command", endpoint)

		hmac, err := crypto.GetHMAC(crypto.HashSHA512, []byte(values.Encode()), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		headers["Sign"] = hex.EncodeToString(hmac)

		path := fmt.Sprintf("%s/%s", ePoint, poloniexAPITradingEndpoint)
		return &request.Item{
			Method:        method,
			Path:          path,
			Headers:       headers,
			Body:          bytes.NewBufferString(values.Encode()),
			Result:        result,
			NonceEnabled:  true,
			Verbose:       p.Verbose,
			HTTPDebugging: p.HTTPDebugging,
			HTTPRecording: p.HTTPRecording,
		}, nil
	}, request.AuthenticatedRequest)
}

// GetFee returns an estimate of fee based on type of transaction
func (p *Poloniex) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		feeInfo, err := p.GetFeeInfo(ctx)
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
