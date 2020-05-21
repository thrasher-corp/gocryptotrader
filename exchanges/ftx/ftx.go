package ftx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
)

// FTX is the overarching type across this package
type FTX struct {
	exchange.Base
	WebsocketConn *wshandler.WebsocketConnection
}

const (
	ftxAPIURL = "https://ftx.com/api"

	// Public endpoints
	getMarkets           = "/markets"
	getMarket            = "/markets/"
	getOrderbook         = "/markets/%s/orderbook?depth=%s"
	getTrades            = "/markets/%s/trades?"
	getHistoricalData    = "/markets/%s/candles?"
	getFutures           = "/futures"
	getFuture            = "/futures/"
	getFutureStats       = "/futures/%s/stats"
	getFundingRates      = "/funding_rates"
	getAllWalletBalances = "/wallet/all_balances"

	// Authenticated endpoints
	getAccountInfo           = "/account"
	getPositions             = "/positions"
	setLeverage              = "/account/leverage"
	getCoins                 = "/wallet/coins"
	getBalances              = "/wallet/balances"
	getDepositAddress        = "/wallet/deposit_address/"
	getDepositHistory        = "/wallet/deposits"
	getWithdrawalHistory     = "/wallet/withdrawals"
	withdrawRequest          = "/wallet/withdrawals"
	getOpenOrders            = "/orders?"
	getOrderHistory          = "/orders/history?"
	getOpenTriggerOrders     = "/conditional_orders?"
	getTriggerOrderTriggers  = "/conditional_orders/%s/triggers"
	getTriggerOrderHistory   = "/conditional_orders/history?"
	placeOrder               = "/orders"
	placeTriggerOrder        = "/conditional_orders"
	modifyOrder              = "/orders/%s/modify"
	modifyOrderByClientID    = "/orders/by_client_id/%s/modify"
	modifyTriggerOrder       = "/conditional_orders/%s/modify"
	getOrderStatus           = "/orders/"
	getOrderStatusByClientID = "/orders/by_client_id/"
	deleteOrder              = "/orders/"
	deleteOrderByClientID    = "/orders/by_client_id/"
	cancelTriggerOrder       = "/conditional_orders/"
	getFills                 = "/fills?"
	getFundingPayments       = "/funding_payments"
	getLeveragedTokens       = "/lt/tokens"
	getTokenInfo             = "/lt/"
	getLTBalances            = "/lt/balances"
	getLTCreations           = "/lt/creations"
	requestLTCreation        = "/lt/%s/create"
	getLTRedemptions         = "/lt/redemptions"
	requestLTRedemption      = "/lt/%s/redeem"
	getListQuotes            = "/options/requests"
	getMyQuotesRequests      = "/options/my_requests"
	createQuoteRequest       = "/options/requests"
	deleteQuote              = "/options/requests/"
	getQuotesForQuote        = "/options/requests/%s/quotes"
	createQuote              = "/options/requests/%s/quotes?"
	getMyQuotes              = "/options/my_quotes"
	deleteMyQuote            = "/options/quotes/"
	acceptQuote              = "/options/quotes/%s/accept"
	getOptionsInfo           = "/options/account_info"
	getOptionsPositions      = "/options/positions"
	getPublicOptionsTrades   = "/options/trades"
	getOptionsFills          = "/options/fills"
	requestOTCQuote          = "/otc/quotes"
	getOTCQuoteStatus        = "/otc/quotes/"
	acceptOTCQuote           = "/otc/quotes/%s/accept"

	// Other Consts
	trailingStopOrderType = "trailingStop"
	takeProfitOrderType   = "takeProfit"
	closedStatus          = "closed"
	defaultTime           = "0001-01-01 00:00:00 +0000 UTC"
	spotString            = "spot"
	futuresString         = "future"
)

// Start implementing public and private exchange API funcs below

// GetMarkets gets market data
func (f *FTX) GetMarkets() (Markets, error) {
	var resp Markets
	return resp, f.SendHTTPRequest(ftxAPIURL+getMarkets, &resp)
}

// GetMarket gets market data for a provided asset type
func (f *FTX) GetMarket(marketName string) (Market, error) {
	var market Market
	return market, f.SendHTTPRequest(ftxAPIURL+getMarket+marketName,
		&market)
}

// GetOrderbook gets orderbook for a given market with a given depth (default depth 20)
func (f *FTX) GetOrderbook(marketName string, depth int64) (OrderbookData, error) {
	strDepth := strconv.FormatInt(depth, 10)
	var resp OrderbookData
	var tempOB TempOrderbook
	err := f.SendHTTPRequest(fmt.Sprintf(ftxAPIURL+getOrderbook, marketName, strDepth), &tempOB)
	if err != nil {
		return resp, err
	}
	resp.MarketName = marketName
	for x := range tempOB.Result.Asks {
		resp.Asks = append(resp.Asks, OData{Price: tempOB.Result.Asks[x][0],
			Size: tempOB.Result.Asks[x][1],
		})
	}
	for y := range tempOB.Result.Bids {
		resp.Bids = append(resp.Bids, OData{Price: tempOB.Result.Bids[y][0],
			Size: tempOB.Result.Bids[y][1],
		})
	}
	return resp, nil
}

// GetTrades gets trades based on the conditions specified
func (f *FTX) GetTrades(marketName string, startTime, endTime time.Time, limit int64) (Trades, error) {
	strLimit := strconv.FormatInt(limit, 10)
	params := url.Values{}
	params.Set("limit", strLimit)
	var resp Trades
	if !startTime.IsZero() && !endTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
		if !startTime.Before(endTime) {
			return resp, errors.New("startTime cannot be bigger than endTime")
		}
	}
	return resp, f.SendHTTPRequest(fmt.Sprintf(ftxAPIURL+getTrades, marketName)+params.Encode(),
		&resp)
}

// GetHistoricalData gets historical OHLCV data for a given market pair
func (f *FTX) GetHistoricalData(marketName, timeInterval, limit string, startTime, endTime time.Time) (HistoricalData, error) {
	var resp HistoricalData
	params := url.Values{}
	params.Set("resolution", timeInterval)
	if limit != "" {
		params.Set("limit", limit)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
		if !startTime.Before(endTime) {
			return resp, errors.New("startTime cannot be bigger than endTime")
		}
	}
	return resp, f.SendHTTPRequest(fmt.Sprintf(ftxAPIURL+getHistoricalData, marketName)+params.Encode(), &resp)
}

// GetFutures gets data on futures
func (f *FTX) GetFutures() (Futures, error) {
	var resp Futures
	return resp, f.SendHTTPRequest(ftxAPIURL+getFutures, &resp)
}

// GetFuture gets data on a given future
func (f *FTX) GetFuture(futureName string) (Future, error) {
	var resp Future
	return resp, f.SendHTTPRequest(ftxAPIURL+getFuture+futureName, &resp)
}

// GetFutureStats gets data on a given future's stats
func (f *FTX) GetFutureStats(futureName string) (FutureStats, error) {
	var resp FutureStats
	return resp, f.SendHTTPRequest(fmt.Sprintf(ftxAPIURL+getFutureStats, futureName), &resp)
}

// GetFundingRates gets data on funding rates
func (f *FTX) GetFundingRates() (FundingRates, error) {
	var resp FundingRates
	return resp, f.SendHTTPRequest(ftxAPIURL+getFundingRates, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (f *FTX) SendHTTPRequest(path string, result interface{}) error {
	return f.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          path,
		Result:        result,
		Verbose:       f.Verbose,
		HTTPDebugging: f.HTTPDebugging,
		HTTPRecording: f.HTTPRecording,
	})
}

// GetAccountInfo gets account info
func (f *FTX) GetAccountInfo() (AccountData, error) {
	var resp AccountData
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getAccountInfo, nil, &resp)
}

// GetPositions gets the users positions
func (f *FTX) GetPositions() (Positions, error) {
	var resp Positions
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getPositions, nil, &resp)
}

// ChangeAccountLeverage changes default leverage used by account
func (f *FTX) ChangeAccountLeverage(leverage float64) error {
	req := make(map[string]interface{})
	req["leverage"] = leverage
	return f.SendAuthHTTPRequest(http.MethodPost, setLeverage, req, nil)
}

// GetCoins gets coins' data in the account wallet
func (f *FTX) GetCoins() (WalletCoins, error) {
	var resp WalletCoins
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getCoins, nil, &resp)
}

// GetBalances gets balances of the account
func (f *FTX) GetBalances() (WalletBalances, error) {
	var resp WalletBalances
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getBalances, nil, &resp)
}

// GetAllWalletBalances gets all wallets' balances
func (f *FTX) GetAllWalletBalances() (AllWalletBalances, error) {
	var resp AllWalletBalances
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getAllWalletBalances, nil, &resp)
}

// FetchDepositAddress gets deposit address for a given coin
func (f *FTX) FetchDepositAddress(coin string) (DepositAddress, error) {
	var resp DepositAddress
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getDepositAddress+coin, nil, &resp)
}

// FetchDepositHistory gets deposit history
func (f *FTX) FetchDepositHistory() (DepositHistory, error) {
	var resp DepositHistory
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getDepositHistory, nil, &resp)
}

// FetchWithdrawalHistory gets withdrawal history
func (f *FTX) FetchWithdrawalHistory() (WithdrawalHistory, error) {
	var resp WithdrawalHistory
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getWithdrawalHistory, nil, &resp)
}

// Withdraw sends a withdrawal request
func (f *FTX) Withdraw(coin, address, tag, password, code string, size float64) (WithdrawData, error) {
	req := make(map[string]interface{})
	req["coin"] = coin
	req["address"] = address
	req["size"] = size
	if code != "" {
		req["code"] = code
	}
	if tag != "" {
		req["tag"] = tag
	}
	if password != "" {
		req["password"] = password
	}
	var resp WithdrawData
	return resp, f.SendAuthHTTPRequest(http.MethodPost, withdrawRequest, req, &resp)
}

// GetOpenOrders gets open orders
func (f *FTX) GetOpenOrders(marketName string) (OpenOrders, error) {
	params := url.Values{}
	if marketName != "" {
		params.Set("market", marketName)
	}
	var resp OpenOrders
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getOpenOrders+params.Encode(), nil, &resp)
}

// FetchOrderHistory gets order history
func (f *FTX) FetchOrderHistory(marketName string, startTime, endTime time.Time, limit string) (OrderHistory, error) {
	var resp OrderHistory
	params := url.Values{}
	if marketName != "" {
		params.Set("market", marketName)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
		if !startTime.Before(endTime) {
			return resp, errors.New("startTime cannot be bigger than endTime")
		}
	}
	if limit != "" {
		params.Set("limit", limit)
	}
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getOrderHistory+params.Encode(), nil, &resp)
}

// GetOpenTriggerOrders gets trigger orders that are currently open
func (f *FTX) GetOpenTriggerOrders(marketName, orderType string) (OpenTriggerOrders, error) {
	params := url.Values{}
	if marketName != "" {
		params.Set("market", marketName)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	var resp OpenTriggerOrders
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getOpenTriggerOrders+params.Encode(), nil, &resp)
}

// GetTriggerOrderTriggers gets trigger orders that are currently open
func (f *FTX) GetTriggerOrderTriggers(orderID string) (Triggers, error) {
	var resp Triggers
	return resp, f.SendAuthHTTPRequest(http.MethodGet, fmt.Sprintf(getTriggerOrderTriggers, orderID), nil, &resp)
}

// GetTriggerOrderHistory gets trigger orders that are currently open
func (f *FTX) GetTriggerOrderHistory(marketName string, startTime, endTime time.Time, side, orderType, limit string) (TriggerOrderHistory, error) {
	var resp TriggerOrderHistory
	params := url.Values{}
	if marketName != "" {
		params.Set("market", marketName)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
		if !startTime.Before(endTime) {
			return resp, errors.New("startTime cannot be bigger than endTime")
		}
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if limit != "" {
		params.Set("limit", limit)
	}
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getTriggerOrderHistory+params.Encode(), nil, &resp)
}

// Order places an order
func (f *FTX) Order(marketName, side, orderType, reduceOnly, ioc, postOnly, clientID string, price, size float64) (PlaceOrder, error) {
	req := make(map[string]interface{})
	req["market"] = marketName
	req["side"] = side
	req["price"] = price
	req["type"] = orderType
	req["size"] = size
	if reduceOnly != "" {
		req["reduceOnly"] = reduceOnly
	}
	if ioc != "" {
		req["ioc"] = ioc
	}
	if postOnly != "" {
		req["postOnly"] = postOnly
	}
	if clientID != "" {
		req["clientID"] = clientID
	}
	var resp PlaceOrder
	return resp, f.SendAuthHTTPRequest(http.MethodPost, placeOrder, req, &resp)
}

// TriggerOrder places an order
func (f *FTX) TriggerOrder(marketName, side, orderType, reduceOnly, retryUntilFilled string, size, triggerPrice, orderPrice, trailValue float64) (PlaceTriggerOrder, error) {
	req := make(map[string]interface{})
	req["market"] = marketName
	req["side"] = side
	req["type"] = orderType
	req["size"] = size
	if reduceOnly != "" {
		req["reduceOnly"] = reduceOnly
	}
	if retryUntilFilled != "" {
		req["retryUntilFilled"] = retryUntilFilled
	}
	if orderType == order.Stop.Lower() || orderType == "" {
		req["triggerPrice"] = triggerPrice
		req["orderPrice"] = orderPrice
	}
	if orderType == trailingStopOrderType {
		req["trailValue"] = trailValue
	}
	if orderType == takeProfitOrderType {
		req["triggerPrice"] = triggerPrice
		req["orderPrice"] = orderPrice
	}
	var resp PlaceTriggerOrder
	return resp, f.SendAuthHTTPRequest(http.MethodPost, placeTriggerOrder, req, &resp)
}

// ModifyPlacedOrder modifies a placed order
func (f *FTX) ModifyPlacedOrder(orderID, clientID string, price, size float64) (ModifyOrder, error) {
	req := make(map[string]interface{})
	req["price"] = price
	req["size"] = size
	if clientID != "" {
		req["clientID"] = clientID
	}
	var resp ModifyOrder
	return resp, f.SendAuthHTTPRequest(http.MethodPost, fmt.Sprintf(modifyOrder, orderID), req, &resp)
}

// ModifyOrderByClientID modifies a placed order via clientOrderID
func (f *FTX) ModifyOrderByClientID(clientOrderID, clientID string, price, size float64) (ModifyOrder, error) {
	req := make(map[string]interface{})
	req["price"] = price
	req["size"] = size
	if clientID != "" {
		req["clientID"] = clientID
	}
	var resp ModifyOrder
	return resp, f.SendAuthHTTPRequest(http.MethodPost, fmt.Sprintf(modifyOrderByClientID, clientOrderID), req, &resp)
}

// ModifyTriggerOrder modifies an existing trigger order
func (f *FTX) ModifyTriggerOrder(orderID, orderType string, size, triggerPrice, orderPrice, trailValue float64) (ModifyTriggerOrder, error) {
	req := make(map[string]interface{})
	if orderType == order.Stop.Lower() || orderType == "" {
		req["triggerPrice"] = triggerPrice
		req["orderPrice"] = orderPrice
	}
	if orderType == trailingStopOrderType {
		req["trailValue"] = trailValue
	}
	if orderType == takeProfitOrderType {
		req["triggerPrice"] = triggerPrice
		req["orderPrice"] = orderPrice
	}
	var resp ModifyTriggerOrder
	return resp, f.SendAuthHTTPRequest(http.MethodPost, fmt.Sprintf(modifyTriggerOrder, orderID), req, &resp)
}

// GetOrderStatus gets the order status of a given orderID
func (f *FTX) GetOrderStatus(orderID string) (OrderStatus, error) {
	var resp OrderStatus
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getOrderStatus+orderID, nil, &resp)
}

// GetOrderStatusByClientID gets the order status of a given clientOrderID
func (f *FTX) GetOrderStatusByClientID(clientOrderID string) (OrderStatus, error) {
	var resp OrderStatus
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getOrderStatusByClientID+clientOrderID, nil, &resp)
}

// DeleteOrder deletes an order
func (f *FTX) DeleteOrder(orderID string) (CancelOrderResponse, error) {
	var resp CancelOrderResponse
	return resp, f.SendAuthHTTPRequest(http.MethodGet, deleteOrder+orderID, nil, &resp)
}

// DeleteOrderByClientID deletes an order
func (f *FTX) DeleteOrderByClientID(clientID string) (CancelOrderResponse, error) {
	var resp CancelOrderResponse
	return resp, f.SendAuthHTTPRequest(http.MethodGet, deleteOrderByClientID+clientID, nil, &resp)
}

// DeleteTriggerOrder deletes an order
func (f *FTX) DeleteTriggerOrder(orderID string) (CancelOrderResponse, error) {
	var resp CancelOrderResponse
	return resp, f.SendAuthHTTPRequest(http.MethodGet, cancelTriggerOrder+orderID, nil, &resp)
}

// GetFills gets fills' data
func (f *FTX) GetFills(market, limit string, startTime, endTime time.Time) (Fills, error) {
	var resp Fills
	params := url.Values{}
	if market != "" {
		params.Set("market", market)
	}
	if limit != "" {
		params.Set("limit", limit)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
		if !startTime.Before(endTime) {
			return resp, errors.New("startTime cannot be bigger than endTime")
		}
	}
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getFills+params.Encode(), nil, &resp)
}

// GetFundingPayments gets funding payments
func (f *FTX) GetFundingPayments(startTime, endTime time.Time, future string) (FundingPayments, error) {
	var resp FundingPayments
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
		if !startTime.Before(endTime) {
			return resp, errors.New("startTime cannot be bigger than endTime")
		}
	}
	if future != "" {
		params.Set("future", future)
	}
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getFundingPayments+params.Encode(), nil, &resp)
}

// ListLeveragedTokens lists leveraged tokens
func (f *FTX) ListLeveragedTokens() (LeveragedTokens, error) {
	var resp LeveragedTokens
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getLeveragedTokens, nil, &resp)
}

// GetTokenInfo gets token info
func (f *FTX) GetTokenInfo(tokenName string) (TokenInfo, error) {
	var resp TokenInfo
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getTokenInfo+tokenName, nil, &resp)
}

// ListLTBalances gets leveraged tokens' balances
func (f *FTX) ListLTBalances() (LTBalances, error) {
	var resp LTBalances
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getLTBalances, nil, &resp)
}

// ListLTCreations lists the leveraged tokens' creation requests
func (f *FTX) ListLTCreations() (LTCreationList, error) {
	var resp LTCreationList
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getLTCreations, nil, &resp)
}

// RequestLTCreation sends a request to create a leveraged token
func (f *FTX) RequestLTCreation(tokenName string, size float64) (RequestTokenCreation, error) {
	req := make(map[string]interface{})
	req["size"] = size
	var resp RequestTokenCreation
	return resp, f.SendAuthHTTPRequest(http.MethodPost, fmt.Sprintf(requestLTCreation, tokenName), req, &resp)
}

// ListLTRedemptions lists the leveraged tokens' redemption requests
func (f *FTX) ListLTRedemptions(tokenName string, size float64) (LTRedemptionList, error) {
	var resp LTRedemptionList
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getLTRedemptions, nil, &resp)
}

// RequestLTRedemption sends a request to redeem a leveraged token
func (f *FTX) RequestLTRedemption(tokenName string, size float64) (LTRedemptionRequest, error) {
	req := make(map[string]interface{})
	req["size"] = size
	var resp LTRedemptionRequest
	return resp, f.SendAuthHTTPRequest(http.MethodPost, fmt.Sprintf(requestLTRedemption, tokenName), req, &resp)
}

// GetQuoteRequests gets a list of quote requests
func (f *FTX) GetQuoteRequests() (QuoteRequests, error) {
	var resp QuoteRequests
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getListQuotes, nil, &resp)
}

// GetYourQuoteRequests gets a list of your quote requests
func (f *FTX) GetYourQuoteRequests() (PersonalQuotes, error) {
	var resp PersonalQuotes
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getMyQuotesRequests, nil, &resp)
}

// CreateQuoteRequest sends a request to create a quote
func (f *FTX) CreateQuoteRequest(underlying, optionType, side, expiry, requestExpiry string, strike, size, limitPrice, counterParyID float64, hideLimitPrice bool) (CreateQuote, error) {
	req := make(map[string]interface{})
	req["underlying"] = underlying
	req["type"] = optionType
	req["side"] = side
	req["strike"] = strike
	req["expiry"] = expiry
	req["size"] = size
	if limitPrice != 0 {
		req["limitPrice"] = limitPrice
	}
	if requestExpiry != "" {
		req["requestExpiry"] = requestExpiry
	}
	if counterParyID != 0 {
		req["counterParyID"] = counterParyID
	}
	req["hideLimitPrice"] = hideLimitPrice
	var resp CreateQuote
	return resp, f.SendAuthHTTPRequest(http.MethodPost, createQuoteRequest, req, &resp)
}

// DeleteQuote sends request to cancel a quote
func (f *FTX) DeleteQuote(requestID string) (CancelQuote, error) {
	var resp CancelQuote
	return resp, f.SendAuthHTTPRequest(http.MethodDelete, deleteQuote+requestID, nil, &resp)
}

// GetQuotesForYourQuote gets a list of quotes for your quote
func (f *FTX) GetQuotesForYourQuote(requestID string) (QuoteForQuoteData, error) {
	var resp QuoteForQuoteData
	return resp, f.SendAuthHTTPRequest(http.MethodGet, fmt.Sprintf(getQuotesForQuote, requestID), nil, &resp)
}

// MakeQuote makes a quote for a quote
func (f *FTX) MakeQuote(requestID, price string) (QuoteForQuoteResponse, error) {
	params := url.Values{}
	params.Set("price", price)
	var resp QuoteForQuoteResponse
	return resp, f.SendAuthHTTPRequest(http.MethodGet, fmt.Sprintf(createQuote, requestID), nil, &resp)
}

// MyQuotes gets a list of my quotes for quotes
func (f *FTX) MyQuotes() (QuoteForQuoteResponse, error) {
	var resp QuoteForQuoteResponse
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getMyQuotes, nil, &resp)
}

// DeleteMyQuote deletes my quote for quotes
func (f *FTX) DeleteMyQuote(quoteID string) (QuoteForQuoteResponse, error) {
	var resp QuoteForQuoteResponse
	return resp, f.SendAuthHTTPRequest(http.MethodDelete, deleteMyQuote+quoteID, nil, &resp)
}

// AcceptQuote accepts the quote for quote
func (f *FTX) AcceptQuote(quoteID string) (QuoteForQuoteResponse, error) {
	var resp QuoteForQuoteResponse
	return resp, f.SendAuthHTTPRequest(http.MethodPost, fmt.Sprintf(acceptQuote, quoteID), nil, &resp)
}

// GetAccountOptionsInfo gets account's options' info
func (f *FTX) GetAccountOptionsInfo() (AccountOptionsInfo, error) {
	var resp AccountOptionsInfo
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getOptionsInfo, nil, &resp)
}

// GetOptionsPositions gets options' positions
func (f *FTX) GetOptionsPositions() (OptionsPositions, error) {
	var resp OptionsPositions
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getOptionsPositions, nil, &resp)
}

// GetPublicOptionsTrades gets options' trades from public
func (f *FTX) GetPublicOptionsTrades(startTime, endTime time.Time, limit string) (PublicOptionsTrades, error) {
	var resp PublicOptionsTrades
	req := make(map[string]interface{})
	if !startTime.IsZero() && !endTime.IsZero() {
		req["start_time"] = strconv.FormatInt(startTime.Unix(), 10)
		req["end_time"] = strconv.FormatInt(endTime.Unix(), 10)
		if !startTime.Before(endTime) {
			return resp, errors.New("startTime cannot be bigger than endTime")
		}
	}
	if limit != "" {
		req["limit"] = limit
	}
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getPublicOptionsTrades, req, &resp)
}

// GetOptionsFills gets fills data for options
func (f *FTX) GetOptionsFills(startTime, endTime time.Time, limit string) (OptionsFills, error) {
	var resp OptionsFills
	req := make(map[string]interface{})
	if !startTime.IsZero() && !endTime.IsZero() {
		req["start_time"] = strconv.FormatInt(startTime.Unix(), 10)
		req["end_time"] = strconv.FormatInt(endTime.Unix(), 10)
		if !startTime.Before(endTime) {
			return resp, errors.New("startTime cannot be bigger than endTime")
		}
	}
	if limit != "" {
		req["limit"] = limit
	}
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getOptionsFills, req, &resp)
}

// SendAuthHTTPRequest sends an authenticated request
func (f *FTX) SendAuthHTTPRequest(method, path string, data, result interface{}) error {
	ts := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
	var body io.Reader
	var hmac, payload []byte
	var err error
	if data != nil {
		payload, err = json.Marshal(data)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(payload)
		sigPayload := ts + method + "/api" + path + string(payload)
		hmac = crypto.GetHMAC(crypto.HashSHA256, []byte(sigPayload), []byte(f.API.Credentials.Secret))
	} else {
		sigPayload := ts + method + "/api" + path
		hmac = crypto.GetHMAC(crypto.HashSHA256, []byte(sigPayload), []byte(f.API.Credentials.Secret))
	}
	headers := make(map[string]string)
	headers["FTX-KEY"] = f.API.Credentials.Key
	headers["FTX-SIGN"] = crypto.HexEncodeToString(hmac)
	headers["FTX-TS"] = ts
	headers["Content-Type"] = "application/json"
	return f.SendPayload(context.Background(), &request.Item{
		Method:        method,
		Path:          ftxAPIURL + path,
		Headers:       headers,
		Body:          body,
		Result:        result,
		AuthRequest:   true,
		Verbose:       f.Verbose,
		HTTPDebugging: f.HTTPDebugging,
		HTTPRecording: f.HTTPRecording,
	})
}

// GetFee returns an estimate of fee based on type of transaction
func (f *FTX) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder)
	default:
		feeData, err := f.GetAccountInfo()
		if err != nil {
			return 0, err
		}
		switch feeBuilder.IsMaker {
		case true:
			fee = feeData.Result.MakerFee * feeBuilder.Amount * feeBuilder.PurchasePrice
		case false:
			fee = feeData.Result.TakerFee * feeBuilder.Amount * feeBuilder.PurchasePrice
		}
		if fee < 0 {
			fee = 0
		}
	}
	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(feeBuilder *exchange.FeeBuilder) float64 {
	if feeBuilder.IsMaker {
		return 0.0002 * feeBuilder.PurchasePrice * feeBuilder.Amount
	}
	return 0.0007 * feeBuilder.PurchasePrice * feeBuilder.Amount
}

func parseInterval(in time.Duration) (TimeInterval, error) {
	switch in {
	case kline.FifteenSecond:
		return TimeIntervalFifteenSeconds, nil
	case kline.OneMin:
		return TimeIntervalMinute, nil
	case kline.FiveMin:
		return TimeIntervalFiveMinutes, nil
	case kline.FifteenMin:
		return TimeIntervalFifteenMinutes, nil
	case kline.OneHour:
		return TimeIntervalHour, nil
	case kline.FourHour:
		return TimeIntervalFourHours, nil
	case kline.OneDay:
		return TimeIntervalDay, nil
	default:
		return TimeIntervalMinute, errInvalidInterval
	}
}

func (f *FTX) compatibleOrderVars(orderSide, orderStatus, orderType string, amount, filledAmount, avgFillPrice float64) (OrderVars, error) {
	var resp OrderVars
	switch orderSide {
	case order.Buy.Lower():
		resp.Side = order.Buy
	case order.Sell.Lower():
		resp.Side = order.Sell
	}
	switch orderStatus {
	case strings.ToLower(order.New.String()):
		resp.Status = order.New
	case strings.ToLower(order.Open.String()):
		resp.Status = order.Open
	case closedStatus:
		if filledAmount != 0 && filledAmount != amount {
			resp.Status = order.PartiallyCancelled
		}
		if filledAmount == 0 {
			resp.Status = order.Cancelled
		}
		if filledAmount == amount {
			resp.Status = order.Filled
		}
	}
	var feeBuilder exchange.FeeBuilder
	feeBuilder.PurchasePrice = avgFillPrice
	feeBuilder.Amount = amount
	resp.OrderType = order.Market
	if orderType == strings.ToLower(order.Limit.String()) {
		resp.OrderType = order.Limit
		feeBuilder.IsMaker = true
	}
	fee, err := f.GetFee(&feeBuilder)
	if err != nil {
		return resp, err
	}
	resp.Fee = fee
	return resp, nil
}

// RequestForQuotes requests for otc quotes
func (f *FTX) RequestForQuotes(base, quote string, amount float64) (RequestQuote, error) {
	var resp RequestQuote
	req := make(map[string]interface{})
	req["fromCoin"] = base
	req["toCoin"] = quote
	req["size"] = amount
	return resp, f.SendAuthHTTPRequest(http.MethodPost, requestOTCQuote, req, &resp)
}

// GetOTCQuoteStatus gets quote status of a quote
func (f *FTX) GetOTCQuoteStatus(marketName, quoteID string) (QuoteStatus, error) {
	var resp QuoteStatus
	params := url.Values{}
	params.Set("market", marketName)
	return resp, f.SendAuthHTTPRequest(http.MethodGet, getOTCQuoteStatus+quoteID, params, &resp)
}

// AcceptOTCQuote requests for otc quotes
func (f *FTX) AcceptOTCQuote(quoteID string) (AcceptQuote, error) {
	var resp AcceptQuote
	return resp, f.SendAuthHTTPRequest(http.MethodPost, fmt.Sprintf(acceptOTCQuote, quoteID), nil, &resp)
}
