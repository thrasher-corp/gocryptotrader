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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// FTX is the overarching type across this package
type FTX struct {
	exchange.Base
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
	getIndexWeights      = "/indexes/%s/weights"
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
	getFundingPayments       = "/funding_payments?"
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
	endpointQuote            = "/options/requests/%s/quotes"
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
	spotString            = "spot"
	futuresString         = "future"

	ratePeriod = time.Second
	rateLimit  = 30
)

// GetMarkets gets market data
func (f *FTX) GetMarkets() ([]MarketData, error) {
	resp := struct {
		Data []MarketData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(ftxAPIURL+getMarkets, &resp)
}

// GetMarket gets market data for a provided asset type
func (f *FTX) GetMarket(marketName string) (MarketData, error) {
	resp := struct {
		Data MarketData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(ftxAPIURL+getMarket+marketName,
		&resp)
}

// GetOrderbook gets orderbook for a given market with a given depth (default depth 20)
func (f *FTX) GetOrderbook(marketName string, depth int64) (OrderbookData, error) {
	result := struct {
		Data TempOBData `json:"result"`
	}{}
	strDepth := strconv.FormatInt(depth, 10)
	var resp OrderbookData
	err := f.SendHTTPRequest(fmt.Sprintf(ftxAPIURL+getOrderbook, marketName, strDepth), &result)
	if err != nil {
		return resp, err
	}
	resp.MarketName = marketName
	for x := range result.Data.Asks {
		resp.Asks = append(resp.Asks, OData{Price: result.Data.Asks[x][0],
			Size: result.Data.Asks[x][1],
		})
	}
	for y := range result.Data.Bids {
		resp.Bids = append(resp.Bids, OData{Price: result.Data.Bids[y][0],
			Size: result.Data.Bids[y][1],
		})
	}
	return resp, nil
}

// GetTrades gets trades based on the conditions specified
func (f *FTX) GetTrades(marketName string, startTime, endTime, limit int64) ([]TradeData, error) {
	strLimit := strconv.FormatInt(limit, 10)
	params := url.Values{}
	params.Set("limit", strLimit)
	resp := struct {
		Data []TradeData `json:"result"`
	}{}
	if startTime > 0 && endTime > 0 {
		if startTime >= (endTime) {
			return resp.Data, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime, 10))
		params.Set("end_time", strconv.FormatInt(endTime, 10))
	}
	return resp.Data, f.SendHTTPRequest(fmt.Sprintf(ftxAPIURL+getTrades, marketName)+params.Encode(),
		&resp)
}

// GetHistoricalData gets historical OHLCV data for a given market pair
func (f *FTX) GetHistoricalData(marketName, timeInterval, limit string, startTime, endTime time.Time) ([]OHLCVData, error) {
	resp := struct {
		Data []OHLCVData `json:"result"`
	}{}
	params := url.Values{}
	params.Set("resolution", timeInterval)
	if limit != "" {
		params.Set("limit", limit)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp.Data, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp.Data, f.SendHTTPRequest(fmt.Sprintf(ftxAPIURL+getHistoricalData, marketName)+params.Encode(), &resp)
}

// GetFutures gets data on futures
func (f *FTX) GetFutures() ([]FuturesData, error) {
	resp := struct {
		Data []FuturesData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(ftxAPIURL+getFutures, &resp)
}

// GetFuture gets data on a given future
func (f *FTX) GetFuture(futureName string) (FuturesData, error) {
	resp := struct {
		Data FuturesData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(ftxAPIURL+getFuture+futureName, &resp)
}

// GetFutureStats gets data on a given future's stats
func (f *FTX) GetFutureStats(futureName string) (FutureStatsData, error) {
	resp := struct {
		Data FutureStatsData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(fmt.Sprintf(ftxAPIURL+getFutureStats, futureName), &resp)
}

// GetFundingRates gets data on funding rates
func (f *FTX) GetFundingRates() ([]FundingRatesData, error) {
	resp := struct {
		Data []FundingRatesData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(ftxAPIURL+getFundingRates, &resp)
}

// GetIndexWeights gets index weights
func (f *FTX) GetIndexWeights(index string) (IndexWeights, error) {
	var resp IndexWeights
	return resp, f.SendHTTPRequest(ftxAPIURL+fmt.Sprintf(getIndexWeights, index), &resp)
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
func (f *FTX) GetAccountInfo() (AccountInfoData, error) {
	resp := struct {
		Data AccountInfoData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getAccountInfo, nil, &resp)
}

// GetPositions gets the users positions
func (f *FTX) GetPositions() ([]PositionData, error) {
	resp := struct {
		Data []PositionData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getPositions, nil, &resp)
}

// ChangeAccountLeverage changes default leverage used by account
func (f *FTX) ChangeAccountLeverage(leverage float64) error {
	req := make(map[string]interface{})
	req["leverage"] = leverage
	return f.SendAuthHTTPRequest(http.MethodPost, setLeverage, req, nil)
}

// GetCoins gets coins' data in the account wallet
func (f *FTX) GetCoins() ([]WalletCoinsData, error) {
	resp := struct {
		Data []WalletCoinsData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getCoins, nil, &resp)
}

// GetBalances gets balances of the account
func (f *FTX) GetBalances() ([]BalancesData, error) {
	resp := struct {
		Data []BalancesData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getBalances, nil, &resp)
}

// GetAllWalletBalances gets all wallets' balances
func (f *FTX) GetAllWalletBalances() (AllWalletAccountData, error) {
	resp := struct {
		Data AllWalletAccountData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getAllWalletBalances, nil, &resp)
}

// FetchDepositAddress gets deposit address for a given coin
func (f *FTX) FetchDepositAddress(coin string) (DepositData, error) {
	resp := struct {
		Data DepositData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getDepositAddress+coin, nil, &resp)
}

// FetchDepositHistory gets deposit history
func (f *FTX) FetchDepositHistory() ([]TransactionData, error) {
	resp := struct {
		Data []TransactionData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getDepositHistory, nil, &resp)
}

// FetchWithdrawalHistory gets withdrawal history
func (f *FTX) FetchWithdrawalHistory() ([]TransactionData, error) {
	resp := struct {
		Data []TransactionData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getWithdrawalHistory, nil, &resp)
}

// Withdraw sends a withdrawal request
func (f *FTX) Withdraw(coin, address, tag, password, code string, size float64) (TransactionData, error) {
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
	resp := struct {
		Data TransactionData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodPost, withdrawRequest, req, &resp)
}

// GetOpenOrders gets open orders
func (f *FTX) GetOpenOrders(marketName string) ([]OrderData, error) {
	params := url.Values{}
	if marketName != "" {
		params.Set("market", marketName)
	}
	resp := struct {
		Data []OrderData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getOpenOrders+params.Encode(), nil, &resp)
}

// FetchOrderHistory gets order history
func (f *FTX) FetchOrderHistory(marketName string, startTime, endTime time.Time, limit string) ([]OrderData, error) {
	resp := struct {
		Data []OrderData `json:"result"`
	}{}
	params := url.Values{}
	if marketName != "" {
		params.Set("market", marketName)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp.Data, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	if limit != "" {
		params.Set("limit", limit)
	}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getOrderHistory+params.Encode(), nil, &resp)
}

// GetOpenTriggerOrders gets trigger orders that are currently open
func (f *FTX) GetOpenTriggerOrders(marketName, orderType string) ([]TriggerOrderData, error) {
	params := url.Values{}
	if marketName != "" {
		params.Set("market", marketName)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	resp := struct {
		Data []TriggerOrderData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getOpenTriggerOrders+params.Encode(), nil, &resp)
}

// GetTriggerOrderTriggers gets trigger orders that are currently open
func (f *FTX) GetTriggerOrderTriggers(orderID string) ([]TriggerData, error) {
	resp := struct {
		Data []TriggerData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, fmt.Sprintf(getTriggerOrderTriggers, orderID), nil, &resp)
}

// GetTriggerOrderHistory gets trigger orders that are currently open
func (f *FTX) GetTriggerOrderHistory(marketName string, startTime, endTime time.Time, side, orderType, limit string) ([]TriggerOrderData, error) {
	resp := struct {
		Data []TriggerOrderData `json:"result"`
	}{}
	params := url.Values{}
	if marketName != "" {
		params.Set("market", marketName)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp.Data, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
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
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getTriggerOrderHistory+params.Encode(), nil, &resp)
}

// Order places an order
func (f *FTX) Order(marketName, side, orderType, reduceOnly, ioc, postOnly, clientID string, price, size float64) (OrderData, error) {
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
		req["clientId"] = clientID
	}
	resp := struct {
		Data OrderData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodPost, placeOrder, req, &resp)
}

// TriggerOrder places an order
func (f *FTX) TriggerOrder(marketName, side, orderType, reduceOnly, retryUntilFilled string, size, triggerPrice, orderPrice, trailValue float64) (TriggerOrderData, error) {
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
	resp := struct {
		Data TriggerOrderData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodPost, placeTriggerOrder, req, &resp)
}

// ModifyPlacedOrder modifies a placed order
func (f *FTX) ModifyPlacedOrder(orderID, clientID string, price, size float64) (OrderData, error) {
	req := make(map[string]interface{})
	req["price"] = price
	req["size"] = size
	if clientID != "" {
		req["clientID"] = clientID
	}
	resp := struct {
		Data OrderData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodPost, fmt.Sprintf(modifyOrder, orderID), req, &resp)
}

// ModifyOrderByClientID modifies a placed order via clientOrderID
func (f *FTX) ModifyOrderByClientID(clientOrderID, clientID string, price, size float64) (OrderData, error) {
	req := make(map[string]interface{})
	req["price"] = price
	req["size"] = size
	if clientID != "" {
		req["clientID"] = clientID
	}
	resp := struct {
		Data OrderData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodPost, fmt.Sprintf(modifyOrderByClientID, clientOrderID), req, &resp)
}

// ModifyTriggerOrder modifies an existing trigger order
// Choices for ordertype include stop, trailingStop, takeProfit
func (f *FTX) ModifyTriggerOrder(orderID, orderType string, size, triggerPrice, orderPrice, trailValue float64) (TriggerOrderData, error) {
	req := make(map[string]interface{})
	req["size"] = size
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
	resp := struct {
		Data TriggerOrderData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodPost, fmt.Sprintf(modifyTriggerOrder, orderID), req, &resp)
}

// GetOrderStatus gets the order status of a given orderID
func (f *FTX) GetOrderStatus(orderID string) (OrderData, error) {
	resp := struct {
		Data OrderData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getOrderStatus+orderID, nil, &resp)
}

// GetOrderStatusByClientID gets the order status of a given clientOrderID
func (f *FTX) GetOrderStatusByClientID(clientOrderID string) (OrderData, error) {
	resp := struct {
		Data OrderData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getOrderStatusByClientID+clientOrderID, nil, &resp)
}

// DeleteOrder deletes an order
func (f *FTX) DeleteOrder(orderID string) (string, error) {
	resp := struct {
		Result  string `json:"result"`
		Success bool   `json:"success"`
	}{}
	if err := f.SendAuthHTTPRequest(http.MethodDelete, deleteOrder+orderID, nil, &resp); err != nil {
		return "", err
	}
	if !resp.Success {
		return resp.Result, errors.New("delete order request by ID unsuccessful")
	}
	return resp.Result, nil
}

// DeleteOrderByClientID deletes an order
func (f *FTX) DeleteOrderByClientID(clientID string) (string, error) {
	resp := struct {
		Result  string `json:"result"`
		Success bool   `json:"success"`
	}{}

	if err := f.SendAuthHTTPRequest(http.MethodDelete, deleteOrderByClientID+clientID, nil, &resp); err != nil {
		return "", err
	}
	if !resp.Success {
		return resp.Result, errors.New("delete order request by client ID unsuccessful")
	}
	return resp.Result, nil
}

// DeleteTriggerOrder deletes an order
func (f *FTX) DeleteTriggerOrder(orderID string) (string, error) {
	resp := struct {
		Result  string `json:"result"`
		Success bool   `json:"success"`
	}{}

	if err := f.SendAuthHTTPRequest(http.MethodDelete, cancelTriggerOrder+orderID, nil, &resp); err != nil {
		return "", err
	}
	if !resp.Success {
		return resp.Result, errors.New("delete trigger order request unsuccessful")
	}
	return resp.Result, nil
}

// GetFills gets fills' data
func (f *FTX) GetFills(market, limit string, startTime, endTime time.Time) ([]FillsData, error) {
	resp := struct {
		Data []FillsData `json:"result"`
	}{}
	params := url.Values{}
	if market != "" {
		params.Set("market", market)
	}
	if limit != "" {
		params.Set("limit", limit)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp.Data, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getFills+params.Encode(), nil, &resp)
}

// GetFundingPayments gets funding payments
func (f *FTX) GetFundingPayments(startTime, endTime time.Time, future string) ([]FundingPaymentsData, error) {
	resp := struct {
		Data []FundingPaymentsData `json:"result"`
	}{}
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp.Data, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	if future != "" {
		params.Set("future", future)
	}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getFundingPayments+params.Encode(), nil, &resp)
}

// ListLeveragedTokens lists leveraged tokens
func (f *FTX) ListLeveragedTokens() ([]LeveragedTokensData, error) {
	resp := struct {
		Data []LeveragedTokensData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getLeveragedTokens, nil, &resp)
}

// GetTokenInfo gets token info
func (f *FTX) GetTokenInfo(tokenName string) ([]LeveragedTokensData, error) {
	resp := struct {
		Data []LeveragedTokensData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getTokenInfo+tokenName, nil, &resp)
}

// ListLTBalances gets leveraged tokens' balances
func (f *FTX) ListLTBalances() ([]LTBalanceData, error) {
	resp := struct {
		Data []LTBalanceData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getLTBalances, nil, &resp)
}

// ListLTCreations lists the leveraged tokens' creation requests
func (f *FTX) ListLTCreations() ([]LTCreationData, error) {
	resp := struct {
		Data []LTCreationData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getLTCreations, nil, &resp)
}

// RequestLTCreation sends a request to create a leveraged token
func (f *FTX) RequestLTCreation(tokenName string, size float64) (RequestTokenCreationData, error) {
	req := make(map[string]interface{})
	req["size"] = size
	resp := struct {
		Data RequestTokenCreationData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodPost, fmt.Sprintf(requestLTCreation, tokenName), req, &resp)
}

// ListLTRedemptions lists the leveraged tokens' redemption requests
func (f *FTX) ListLTRedemptions() ([]LTRedemptionData, error) {
	resp := struct {
		Data []LTRedemptionData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getLTRedemptions, nil, &resp)
}

// RequestLTRedemption sends a request to redeem a leveraged token
func (f *FTX) RequestLTRedemption(tokenName string, size float64) (LTRedemptionRequestData, error) {
	req := make(map[string]interface{})
	req["size"] = size
	resp := struct {
		Data LTRedemptionRequestData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodPost, fmt.Sprintf(requestLTRedemption, tokenName), req, &resp)
}

// GetQuoteRequests gets a list of quote requests
func (f *FTX) GetQuoteRequests() ([]QuoteRequestData, error) {
	resp := struct {
		Data []QuoteRequestData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getListQuotes, nil, &resp)
}

// GetYourQuoteRequests gets a list of your quote requests
func (f *FTX) GetYourQuoteRequests() ([]PersonalQuotesData, error) {
	resp := struct {
		Data []PersonalQuotesData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getMyQuotesRequests, nil, &resp)
}

// CreateQuoteRequest sends a request to create a quote
func (f *FTX) CreateQuoteRequest(underlying, optionType, side string, expiry int64, requestExpiry string, strike, size, limitPrice, counterParyID float64, hideLimitPrice bool) (CreateQuoteRequestData, error) {
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
	resp := struct {
		Data CreateQuoteRequestData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodPost, createQuoteRequest, req, &resp)
}

// DeleteQuote sends request to cancel a quote
func (f *FTX) DeleteQuote(requestID string) (CancelQuoteRequestData, error) {
	resp := struct {
		Data CancelQuoteRequestData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodDelete, deleteQuote+requestID, nil, &resp)
}

// GetQuotesForYourQuote gets a list of quotes for your quote
func (f *FTX) GetQuotesForYourQuote(requestID string) (QuoteForQuoteData, error) {
	var resp QuoteForQuoteData
	return resp, f.SendAuthHTTPRequest(http.MethodGet, fmt.Sprintf(endpointQuote, requestID), nil, &resp)
}

// MakeQuote makes a quote for a quote
func (f *FTX) MakeQuote(requestID, price string) ([]QuoteForQuoteData, error) {
	params := url.Values{}
	params.Set("price", price)
	resp := struct {
		Data []QuoteForQuoteData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodPost, fmt.Sprintf(endpointQuote, requestID), nil, &resp)
}

// MyQuotes gets a list of my quotes for quotes
func (f *FTX) MyQuotes() ([]QuoteForQuoteData, error) {
	resp := struct {
		Data []QuoteForQuoteData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getMyQuotes, nil, &resp)
}

// DeleteMyQuote deletes my quote for quotes
func (f *FTX) DeleteMyQuote(quoteID string) ([]QuoteForQuoteData, error) {
	resp := struct {
		Data []QuoteForQuoteData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodDelete, deleteMyQuote+quoteID, nil, &resp)
}

// AcceptQuote accepts the quote for quote
func (f *FTX) AcceptQuote(quoteID string) ([]QuoteForQuoteData, error) {
	resp := struct {
		Data []QuoteForQuoteData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodPost, fmt.Sprintf(acceptQuote, quoteID), nil, &resp)
}

// GetAccountOptionsInfo gets account's options' info
func (f *FTX) GetAccountOptionsInfo() (AccountOptionsInfoData, error) {
	resp := struct {
		Data AccountOptionsInfoData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getOptionsInfo, nil, &resp)
}

// GetOptionsPositions gets options' positions
func (f *FTX) GetOptionsPositions() ([]OptionsPositionsData, error) {
	resp := struct {
		Data []OptionsPositionsData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getOptionsPositions, nil, &resp)
}

// GetPublicOptionsTrades gets options' trades from public
func (f *FTX) GetPublicOptionsTrades(startTime, endTime time.Time, limit string) ([]OptionsTradesData, error) {
	resp := struct {
		Data []OptionsTradesData `json:"result"`
	}{}
	req := make(map[string]interface{})
	if !startTime.IsZero() && !endTime.IsZero() {
		req["start_time"] = strconv.FormatInt(startTime.Unix(), 10)
		req["end_time"] = strconv.FormatInt(endTime.Unix(), 10)
		if startTime.After(endTime) {
			return resp.Data, errors.New("startTime cannot be after endTime")
		}
	}
	if limit != "" {
		req["limit"] = limit
	}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getPublicOptionsTrades, req, &resp)
}

// GetOptionsFills gets fills data for options
func (f *FTX) GetOptionsFills(startTime, endTime time.Time, limit string) ([]OptionFillsData, error) {
	resp := struct {
		Data []OptionFillsData `json:"result"`
	}{}
	req := make(map[string]interface{})
	if !startTime.IsZero() && !endTime.IsZero() {
		req["start_time"] = strconv.FormatInt(startTime.Unix(), 10)
		req["end_time"] = strconv.FormatInt(endTime.Unix(), 10)
		if startTime.After(endTime) {
			return resp.Data, errors.New("startTime cannot be after endTime")
		}
	}
	if limit != "" {
		req["limit"] = limit
	}
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getOptionsFills, req, &resp)
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
			fee = feeData.MakerFee * feeBuilder.Amount * feeBuilder.PurchasePrice
		case false:
			fee = feeData.TakerFee * feeBuilder.Amount * feeBuilder.PurchasePrice
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
	if strings.EqualFold(orderType, order.Limit.String()) {
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
func (f *FTX) RequestForQuotes(base, quote string, amount float64) (RequestQuoteData, error) {
	resp := struct {
		Data RequestQuoteData `json:"result"`
	}{}
	req := make(map[string]interface{})
	req["fromCoin"] = base
	req["toCoin"] = quote
	req["size"] = amount
	return resp.Data, f.SendAuthHTTPRequest(http.MethodPost, requestOTCQuote, req, &resp)
}

// GetOTCQuoteStatus gets quote status of a quote
func (f *FTX) GetOTCQuoteStatus(marketName, quoteID string) ([]QuoteStatusData, error) {
	resp := struct {
		Data []QuoteStatusData `json:"result"`
	}{}
	params := url.Values{}
	params.Set("market", marketName)
	return resp.Data, f.SendAuthHTTPRequest(http.MethodGet, getOTCQuoteStatus+quoteID, params, &resp)
}

// AcceptOTCQuote requests for otc quotes
func (f *FTX) AcceptOTCQuote(quoteID string) error {
	return f.SendAuthHTTPRequest(http.MethodPost, fmt.Sprintf(acceptOTCQuote, quoteID), nil, nil)
}
