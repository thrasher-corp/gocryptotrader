package poloniex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	poloniexAPIURL               = "https://poloniex.com"
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
	poloniexMaxOrderbookDepth    = 100
)

// Poloniex is the overarching type across the poloniex package
type Poloniex struct {
	exchange.Base
	details CurrencyDetails
}

// GetTicker returns current ticker information
func (p *Poloniex) GetTicker() (map[string]Ticker, error) {
	type response struct {
		Data map[string]Ticker
	}

	resp := response{}
	path := "/public?command=returnTicker"

	return resp.Data, p.SendHTTPRequest(exchange.RestSpot, path, &resp.Data)
}

// GetVolume returns a list of currencies with associated volume
func (p *Poloniex) GetVolume() (interface{}, error) {
	var resp interface{}
	path := "/public?command=return24hVolume"

	return resp, p.SendHTTPRequest(exchange.RestSpot, path, &resp)
}

// GetOrderbook returns the full orderbook from poloniex
func (p *Poloniex) GetOrderbook(currencyPair string, depth int) (OrderbookAll, error) {
	vals := url.Values{}

	if depth != 0 {
		vals.Set("depth", strconv.Itoa(depth))
	}

	oba := OrderbookAll{Data: make(map[string]Orderbook)}
	if currencyPair != "" {
		vals.Set("currencyPair", currencyPair)
		resp := OrderbookResponse{}
		path := fmt.Sprintf("/public?command=returnOrderBook&%s", vals.Encode())
		err := p.SendHTTPRequest(exchange.RestSpot, path, &resp)
		if err != nil {
			return oba, err
		}
		if resp.Error != "" {
			return oba, fmt.Errorf("%s GetOrderbook() error: %s", p.Name, resp.Error)
		}
		var ob Orderbook
		for x := range resp.Asks {
			price, err := strconv.ParseFloat(resp.Asks[x][0].(string), 64)
			if err != nil {
				return oba, err
			}
			ob.Asks = append(ob.Asks, OrderbookItem{
				Price:  price,
				Amount: resp.Asks[x][1].(float64),
			})
		}

		for x := range resp.Bids {
			price, err := strconv.ParseFloat(resp.Bids[x][0].(string), 64)
			if err != nil {
				return oba, err
			}
			ob.Bids = append(ob.Bids, OrderbookItem{
				Price:  price,
				Amount: resp.Bids[x][1].(float64),
			})
		}
		oba.Data[currencyPair] = ob
	} else {
		vals.Set("currencyPair", "all")
		resp := OrderbookResponseAll{}
		path := fmt.Sprintf("/public?command=returnOrderBook&%s", vals.Encode())
		err := p.SendHTTPRequest(exchange.RestSpot, path, &resp.Data)
		if err != nil {
			return oba, err
		}
		for currency, orderbook := range resp.Data {
			var ob Orderbook
			for x := range orderbook.Asks {
				price, err := strconv.ParseFloat(orderbook.Asks[x][0].(string), 64)
				if err != nil {
					return oba, err
				}
				ob.Asks = append(ob.Asks, OrderbookItem{
					Price:  price,
					Amount: orderbook.Asks[x][1].(float64),
				})
			}

			for x := range orderbook.Bids {
				price, err := strconv.ParseFloat(orderbook.Bids[x][0].(string), 64)
				if err != nil {
					return oba, err
				}
				ob.Bids = append(ob.Bids, OrderbookItem{
					Price:  price,
					Amount: orderbook.Bids[x][1].(float64),
				})
			}
			oba.Data[currency] = ob
		}
	}
	return oba, nil
}

// GetTradeHistory returns trades history from poloniex
func (p *Poloniex) GetTradeHistory(currencyPair string, start, end int64) ([]TradeHistory, error) {
	vals := url.Values{}
	vals.Set("currencyPair", currencyPair)

	if start > 0 {
		vals.Set("start", strconv.FormatInt(start, 10))
	}

	if end > 0 {
		vals.Set("end", strconv.FormatInt(end, 10))
	}

	var resp []TradeHistory
	path := fmt.Sprintf("/public?command=returnTradeHistory&%s", vals.Encode())

	return resp, p.SendHTTPRequest(exchange.RestSpot, path, &resp)
}

// GetChartData returns chart data for a specific currency pair
func (p *Poloniex) GetChartData(currencyPair string, start, end time.Time, period string) ([]ChartData, error) {
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
	var resp []ChartData
	path := "/public?command=returnChartData&" + vals.Encode()
	err := p.SendHTTPRequest(exchange.RestSpot, path, &temp)
	if err != nil {
		return nil, err
	}

	tempUnmarshal := json.Unmarshal(temp, &resp)
	if tempUnmarshal != nil {
		var errResp struct {
			Error string `json:"error"`
		}
		errRet := json.Unmarshal(temp, &errResp)
		if errRet != nil {
			return nil, err
		}
		if errResp.Error != "" {
			return nil, errors.New(errResp.Error)
		}
	}

	return resp, nil
}

// GetCurrencies returns information about currencies
func (p *Poloniex) GetCurrencies() (map[string]Currencies, error) {
	type Response struct {
		Data map[string]Currencies
	}
	resp := Response{}
	return resp.Data, p.SendHTTPRequest(exchange.RestSpot, "/public?command=returnCurrencies", &resp.Data)
}

// GetLoanOrders returns the list of loan offers and demands for a given
// currency, specified by the "currency" GET parameter.
func (p *Poloniex) GetLoanOrders(currency string) (LoanOrders, error) {
	resp := LoanOrders{}
	path := fmt.Sprintf("/public?command=returnLoanOrders&currency=%s", currency)

	return resp, p.SendHTTPRequest(exchange.RestSpot, path, &resp)
}

// GetBalances returns balances for your account.
func (p *Poloniex) GetBalances() (Balance, error) {
	var result interface{}

	err := p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexBalances, url.Values{}, &result)
	if err != nil {
		return Balance{}, err
	}

	data := result.(map[string]interface{})
	balance := Balance{}
	balance.Currency = make(map[string]float64)

	for x, y := range data {
		balance.Currency[x], _ = strconv.ParseFloat(y.(string), 64)
	}

	return balance, nil
}

// GetCompleteBalances returns complete balances from your account.
func (p *Poloniex) GetCompleteBalances() (CompleteBalances, error) {
	var result CompleteBalances
	vals := url.Values{}
	vals.Set("account", "all")
	err := p.SendAuthenticatedHTTPRequest(exchange.RestSpot,
		http.MethodPost,
		poloniexBalancesComplete,
		vals,
		&result)
	return result, err
}

// GetDepositAddresses returns deposit addresses for all enabled cryptos.
func (p *Poloniex) GetDepositAddresses() (DepositAddresses, error) {
	var result interface{}
	addresses := DepositAddresses{}

	err := p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexDepositAddresses, url.Values{}, &result)
	if err != nil {
		return addresses, err
	}

	addresses.Addresses = make(map[string]string)
	data, ok := result.(map[string]interface{})
	if !ok {
		return addresses, errors.New("return val not map[string]interface{}")
	}

	for x, y := range data {
		addresses.Addresses[x] = y.(string)
	}

	return addresses, nil
}

// GenerateNewAddress generates a new address for a currency
func (p *Poloniex) GenerateNewAddress(currency string) (string, error) {
	type Response struct {
		Success  int
		Error    string
		Response string
	}
	resp := Response{}
	values := url.Values{}
	values.Set("currency", currency)

	err := p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexGenerateNewAddress, values, &resp)
	if err != nil {
		return "", err
	}

	if resp.Error != "" {
		return "", errors.New(resp.Error)
	}

	return resp.Response, nil
}

// GetDepositsWithdrawals returns a list of deposits and withdrawals
func (p *Poloniex) GetDepositsWithdrawals(start, end string) (DepositsWithdrawals, error) {
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

	return resp, p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexDepositsWithdrawals, values, &resp)
}

// GetOpenOrders returns current unfilled opened orders
func (p *Poloniex) GetOpenOrders(currency string) (OpenOrdersResponse, error) {
	values := url.Values{}
	values.Set("currencyPair", currency)
	result := OpenOrdersResponse{}
	return result, p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexOrders, values, &result.Data)
}

// GetOpenOrdersForAllCurrencies returns all open orders
func (p *Poloniex) GetOpenOrdersForAllCurrencies() (OpenOrdersResponseAll, error) {
	values := url.Values{}
	values.Set("currencyPair", "all")
	result := OpenOrdersResponseAll{}
	return result, p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexOrders, values, &result.Data)
}

// GetAuthenticatedTradeHistoryForCurrency returns account trade history
func (p *Poloniex) GetAuthenticatedTradeHistoryForCurrency(currency string, start, end, limit int64) (AuthenticatedTradeHistoryResponse, error) {
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
	return result, p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexTradeHistory, values, &result.Data)
}

// GetAuthenticatedTradeHistory returns account trade history
func (p *Poloniex) GetAuthenticatedTradeHistory(start, end, limit int64) (AuthenticatedTradeHistoryAll, error) {
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

	err := p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexTradeHistory, values, &result)
	if err != nil {
		return AuthenticatedTradeHistoryAll{}, err
	}

	var nodata []interface{}
	err = json.Unmarshal(result, &nodata)
	if err == nil {
		return AuthenticatedTradeHistoryAll{}, nil
	}

	var mainResult AuthenticatedTradeHistoryAll
	return mainResult, json.Unmarshal(result, &mainResult.Data)
}

// GetAuthenticatedOrderStatus returns the status of a given orderId.
func (p *Poloniex) GetAuthenticatedOrderStatus(orderID string) (o OrderStatusData, err error) {
	values := url.Values{}

	if orderID == "" {
		return o, fmt.Errorf("no orderID passed")
	}

	values.Set("orderNumber", orderID)
	var rawOrderStatus OrderStatus
	err = p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexOrderStatus, values, &rawOrderStatus)
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
		return o, fmt.Errorf(errMsg.Error)
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
func (p *Poloniex) GetAuthenticatedOrderTrades(orderID string) (o []OrderTrade, err error) {
	values := url.Values{}

	if orderID == "" {
		return nil, fmt.Errorf("no orderId passed")
	}

	values.Set("orderNumber", orderID)
	var result json.RawMessage
	err = p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexOrderTrades, values, &result)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("received unexpected response")
	}

	switch result[0] {
	case '{': // error message received
		var resp GenericResponse
		err = json.Unmarshal(result, &resp)
		if err != nil {
			return nil, err
		}
		if resp.Error != "" {
			err = fmt.Errorf(resp.Error)
		}
	case '[': // data received
		err = json.Unmarshal(result, &o)
	default:
		return nil, fmt.Errorf("received unexpected response")
	}

	return o, err
}

// PlaceOrder places a new order on the exchange
func (p *Poloniex) PlaceOrder(currency string, rate, amount float64, immediate, fillOrKill, buy bool) (OrderResponse, error) {
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

	return result, p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, orderType, values, &result)
}

// CancelExistingOrder cancels and order by orderID
func (p *Poloniex) CancelExistingOrder(orderID int64) error {
	result := GenericResponse{}
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderID, 10))

	err := p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexOrderCancel, values, &result)
	if err != nil {
		return err
	}

	if result.Success != 1 {
		return errors.New(result.Error)
	}

	return nil
}

// MoveOrder moves an order
func (p *Poloniex) MoveOrder(orderID int64, rate, amount float64, postOnly, immediateOrCancel bool) (MoveOrderResponse, error) {
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

	err := p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost,
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

// Withdraw withdraws a currency to a specific delegated address
func (p *Poloniex) Withdraw(currency, address string, amount float64) (*Withdraw, error) {
	result := &Withdraw{}
	values := url.Values{}

	values.Set("currency", currency)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("address", address)

	err := p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexWithdraw, values, &result)
	if err != nil {
		return nil, err
	}

	if result.Error != "" {
		return nil, errors.New(result.Error)
	}

	return result, nil
}

// GetFeeInfo returns fee information
func (p *Poloniex) GetFeeInfo() (Fee, error) {
	result := Fee{}

	return result, p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexFeeInfo, url.Values{}, &result)
}

// GetTradableBalances returns tradable balances
func (p *Poloniex) GetTradableBalances() (map[string]map[string]float64, error) {
	type Response struct {
		Data map[string]map[string]interface{}
	}
	result := Response{}

	err := p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexTradableBalances, url.Values{}, &result.Data)
	if err != nil {
		return nil, err
	}

	balances := make(map[string]map[string]float64)

	for x, y := range result.Data {
		balances[x] = make(map[string]float64)
		for z, w := range y {
			balances[x][z], _ = strconv.ParseFloat(w.(string), 64)
		}
	}

	return balances, nil
}

// TransferBalance transfers balances between your accounts
func (p *Poloniex) TransferBalance(currency, from, to string, amount float64) (bool, error) {
	values := url.Values{}
	result := GenericResponse{}

	values.Set("currency", currency)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("fromAccount", from)
	values.Set("toAccount", to)

	err := p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexTransferBalance, values, &result)
	if err != nil {
		return false, err
	}

	if result.Error != "" && result.Success != 1 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// GetMarginAccountSummary returns a summary on your margin accounts
func (p *Poloniex) GetMarginAccountSummary() (Margin, error) {
	result := Margin{}
	return result, p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexMarginAccountSummary, url.Values{}, &result)
}

// PlaceMarginOrder places a margin order
func (p *Poloniex) PlaceMarginOrder(currency string, rate, amount, lendingRate float64, buy bool) (OrderResponse, error) {
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

	return result, p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, orderType, values, &result)
}

// GetMarginPosition returns a position on a margin order
func (p *Poloniex) GetMarginPosition(currency string) (interface{}, error) {
	values := url.Values{}

	if currency != "" && currency != "all" {
		values.Set("currencyPair", currency)
		result := MarginPosition{}
		return result, p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexMarginPosition, values, &result)
	}
	values.Set("currencyPair", "all")

	type Response struct {
		Data map[string]MarginPosition
	}
	result := Response{}
	return result, p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexMarginPosition, values, &result.Data)
}

// CloseMarginPosition closes a current margin position
func (p *Poloniex) CloseMarginPosition(currency string) (bool, error) {
	values := url.Values{}
	values.Set("currencyPair", currency)
	result := GenericResponse{}

	err := p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexMarginPositionClose, values, &result)
	if err != nil {
		return false, err
	}

	if result.Success == 0 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// CreateLoanOffer places a loan offer on the exchange
func (p *Poloniex) CreateLoanOffer(currency string, amount, rate float64, duration int, autoRenew bool) (int64, error) {
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

	err := p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexCreateLoanOffer, values, &result)
	if err != nil {
		return 0, err
	}

	if result.Success == 0 {
		return 0, errors.New(result.Error)
	}

	return result.OrderID, nil
}

// CancelLoanOffer cancels a loan offer order
func (p *Poloniex) CancelLoanOffer(orderNumber int64) (bool, error) {
	result := GenericResponse{}
	values := url.Values{}
	values.Set("orderID", strconv.FormatInt(orderNumber, 10))

	err := p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexCancelLoanOffer, values, &result)
	if err != nil {
		return false, err
	}

	if result.Success == 0 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// GetOpenLoanOffers returns all open loan offers
func (p *Poloniex) GetOpenLoanOffers() (map[string][]LoanOffer, error) {
	type Response struct {
		Data map[string][]LoanOffer
	}
	result := Response{}

	err := p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexOpenLoanOffers, url.Values{}, &result.Data)
	if err != nil {
		return nil, err
	}

	if result.Data == nil {
		return nil, errors.New("there are no open loan offers")
	}

	return result.Data, nil
}

// GetActiveLoans returns active loans
func (p *Poloniex) GetActiveLoans() (ActiveLoans, error) {
	result := ActiveLoans{}
	return result, p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost, poloniexActiveLoans, url.Values{}, &result)
}

// GetLendingHistory returns lending history for the account
func (p *Poloniex) GetLendingHistory(start, end string) ([]LendingHistory, error) {
	vals := url.Values{}

	if start != "" {
		vals.Set("start", start)
	}

	if end != "" {
		vals.Set("end", end)
	}

	var resp []LendingHistory
	return resp, p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost,
		poloniexLendingHistory,
		vals,
		&resp)
}

// ToggleAutoRenew allows for the autorenew of a contract
func (p *Poloniex) ToggleAutoRenew(orderNumber int64) (bool, error) {
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderNumber, 10))
	result := GenericResponse{}

	err := p.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost,
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

// SendHTTPRequest sends an unauthenticated HTTP request
func (p *Poloniex) SendHTTPRequest(ep exchange.URL, path string, result interface{}) error {
	endpoint, err := p.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	return p.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       p.Verbose,
		HTTPDebugging: p.HTTPDebugging,
		HTTPRecording: p.HTTPRecording,
	})
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (p *Poloniex) SendAuthenticatedHTTPRequest(ep exchange.URL, method, endpoint string, values url.Values, result interface{}) error {
	if !p.AllowAuthenticatedRequest() {
		return fmt.Errorf("%s %w", p.Name, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}
	ePoint, err := p.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	headers["Key"] = p.API.Credentials.Key
	values.Set("nonce", strconv.FormatInt(time.Now().UnixNano(), 10))
	values.Set("command", endpoint)

	hmac := crypto.GetHMAC(crypto.HashSHA512,
		[]byte(values.Encode()),
		[]byte(p.API.Credentials.Secret))

	headers["Sign"] = crypto.HexEncodeToString(hmac)

	path := fmt.Sprintf("%s/%s", ePoint, poloniexAPITradingEndpoint)

	return p.SendPayload(context.Background(), &request.Item{
		Method:        method,
		Path:          path,
		Headers:       headers,
		Body:          bytes.NewBufferString(values.Encode()),
		Result:        result,
		AuthRequest:   true,
		Verbose:       p.Verbose,
		HTTPDebugging: p.HTTPDebugging,
		HTTPRecording: p.HTTPRecording,
	})
}

// GetFee returns an estimate of fee based on type of transaction
func (p *Poloniex) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		feeInfo, err := p.GetFeeInfo()
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
