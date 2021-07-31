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

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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
	getTrades            = "/markets/%s/trades"
	getHistoricalData    = "/markets/%s/candles"
	getFutures           = "/futures"
	getFuture            = "/futures/"
	getFutureStats       = "/futures/%s/stats"
	getFundingRates      = "/funding_rates"
	getIndexWeights      = "/indexes/%s/weights"
	getAllWalletBalances = "/wallet/all_balances"
	getIndexCandles      = "/indexes/%s/candles"

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
	getOpenOrders            = "/orders"
	getOrderHistory          = "/orders/history"
	getOpenTriggerOrders     = "/conditional_orders"
	getTriggerOrderTriggers  = "/conditional_orders/%s/triggers"
	getTriggerOrderHistory   = "/conditional_orders/history"
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
	getFills                 = "/fills"
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
	subaccounts              = "/subaccounts"
	subaccountsUpdateName    = "/subaccounts/update_name"
	subaccountsBalance       = "/subaccounts/%s/balances"
	subaccountsTransfer      = "/subaccounts/transfer"

	// Margin Endpoints
	marginBorrowRates    = "/spot_margin/borrow_rates"
	marginLendingRates   = "/spot_margin/lending_rates"
	marginLendingHistory = "/spot_margin/history"
	dailyBorrowedAmounts = "/spot_margin/borrow_summary"
	marginMarketInfo     = "/spot_margin/market_info?market=%s"
	marginBorrowHistory  = "/spot_margin/borrow_history"
	marginLendHistory    = "/spot_margin/lending_history"
	marginLendingOffers  = "/spot_margin/offers"
	marginLendingInfo    = "/spot_margin/lending_info"
	submitLendingOrder   = "/spot_margin/offers"

	// Staking endpoints
	stakes          = "/staking/stakes"
	unstakeRequests = "/staking/unstake_requests"
	stakeBalances   = "/staking/balances"
	stakingRewards  = "/staking/staking_rewards"
	serumStakes     = "/srm_stakes/stakes"

	// Other Consts
	trailingStopOrderType = "trailingStop"
	takeProfitOrderType   = "takeProfit"
	closedStatus          = "closed"
	spotString            = "spot"
	futuresString         = "future"

	ratePeriod = time.Second
	rateLimit  = 30
)

var (
	errInvalidOrderID                                    = errors.New("invalid order ID")
	errStartTimeCannotBeAfterEndTime                     = errors.New("start timestamp cannot be after end timestamp")
	errSubaccountNameMustBeSpecified                     = errors.New("a subaccount name must be specified")
	errSubaccountUpdateNameInvalid                       = errors.New("invalid subaccount old/new name")
	errCoinMustBeSpecified                               = errors.New("a coin must be specified")
	errSubaccountTransferSizeGreaterThanZero             = errors.New("transfer size must be greater than 0")
	errSubaccountTransferSourceDestinationMustNotBeEqual = errors.New("subaccount transfer source and destination must not be the same value")
	errUnrecognisedOrderStatus                           = errors.New("unrecognised order status received")
	errInvalidOrderAmounts                               = errors.New("filled amount should not exceed order amount")

	validResolutionData = []int64{15, 60, 300, 900, 3600, 14400, 86400}
)

// GetHistoricalIndex gets historical index data
func (f *FTX) GetHistoricalIndex(indexName string, resolution int64, startTime, endTime time.Time) ([]OHLCVData, error) {
	params := url.Values{}
	if indexName == "" {
		return nil, errors.New("indexName is a mandatory field")
	}
	params.Set("index_name", indexName)
	err := checkResolution(resolution)
	if err != nil {
		return nil, err
	}
	params.Set("resolution", strconv.FormatInt(resolution, 10))
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errStartTimeCannotBeAfterEndTime
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	resp := struct {
		Data []OHLCVData `json:"result"`
	}{}
	endpoint := common.EncodeURLValues(fmt.Sprintf(getIndexCandles, indexName), params)
	return resp.Data, f.SendHTTPRequest(exchange.RestSpot, endpoint, &resp)
}

func checkResolution(res int64) error {
	for x := range validResolutionData {
		if validResolutionData[x] == res {
			return nil
		}
	}
	return errors.New("resolution data is a mandatory field and the data provided is invalid")
}

// GetMarkets gets market data
func (f *FTX) GetMarkets() ([]MarketData, error) {
	resp := struct {
		Data []MarketData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(exchange.RestSpot, getMarkets, &resp)
}

// GetMarket gets market data for a provided asset type
func (f *FTX) GetMarket(marketName string) (MarketData, error) {
	resp := struct {
		Data MarketData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(exchange.RestSpot, getMarket+marketName,
		&resp)
}

// GetOrderbook gets orderbook for a given market with a given depth (default depth 20)
func (f *FTX) GetOrderbook(marketName string, depth int64) (OrderbookData, error) {
	result := struct {
		Data TempOBData `json:"result"`
	}{}

	strDepth := "20" // If we send a zero value we get zero asks from the
	// endpoint
	if depth != 0 {
		strDepth = strconv.FormatInt(depth, 10)
	}

	var resp OrderbookData
	err := f.SendHTTPRequest(exchange.RestSpot, fmt.Sprintf(getOrderbook, marketName, strDepth), &result)
	if err != nil {
		return resp, err
	}
	resp.MarketName = marketName
	for x := range result.Data.Asks {
		resp.Asks = append(resp.Asks, OData{
			Price: result.Data.Asks[x][0],
			Size:  result.Data.Asks[x][1],
		})
	}
	for y := range result.Data.Bids {
		resp.Bids = append(resp.Bids, OData{
			Price: result.Data.Bids[y][0],
			Size:  result.Data.Bids[y][1],
		})
	}
	return resp, nil
}

// GetTrades gets trades based on the conditions specified
func (f *FTX) GetTrades(marketName string, startTime, endTime, limit int64) ([]TradeData, error) {
	if marketName == "" {
		return nil, errors.New("a market pair must be specified")
	}

	params := url.Values{}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if startTime > 0 && endTime > 0 {
		if startTime >= (endTime) {
			return nil, errStartTimeCannotBeAfterEndTime
		}
		params.Set("start_time", strconv.FormatInt(startTime, 10))
		params.Set("end_time", strconv.FormatInt(endTime, 10))
	}
	resp := struct {
		Data []TradeData `json:"result"`
	}{}
	endpoint := common.EncodeURLValues(fmt.Sprintf(getTrades, marketName), params)
	return resp.Data, f.SendHTTPRequest(exchange.RestSpot, endpoint, &resp)
}

// GetHistoricalData gets historical OHLCV data for a given market pair
func (f *FTX) GetHistoricalData(marketName string, timeInterval, limit int64, startTime, endTime time.Time) ([]OHLCVData, error) {
	if marketName == "" {
		return nil, errors.New("a market pair must be specified")
	}

	err := checkResolution(timeInterval)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("resolution", strconv.FormatInt(timeInterval, 10))
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errStartTimeCannotBeAfterEndTime
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	resp := struct {
		Data []OHLCVData `json:"result"`
	}{}
	endpoint := common.EncodeURLValues(fmt.Sprintf(getHistoricalData, marketName), params)
	return resp.Data, f.SendHTTPRequest(exchange.RestSpot, endpoint, &resp)
}

// GetFutures gets data on futures
func (f *FTX) GetFutures() ([]FuturesData, error) {
	resp := struct {
		Data []FuturesData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(exchange.RestSpot, getFutures, &resp)
}

// GetFuture gets data on a given future
func (f *FTX) GetFuture(futureName string) (FuturesData, error) {
	resp := struct {
		Data FuturesData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(exchange.RestSpot, getFuture+futureName, &resp)
}

// GetFutureStats gets data on a given future's stats
func (f *FTX) GetFutureStats(futureName string) (FutureStatsData, error) {
	resp := struct {
		Data FutureStatsData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(exchange.RestSpot, fmt.Sprintf(getFutureStats, futureName), &resp)
}

// GetFundingRates gets data on funding rates
func (f *FTX) GetFundingRates(startTime, endTime time.Time, future string) ([]FundingRatesData, error) {
	resp := struct {
		Data []FundingRatesData `json:"result"`
	}{}
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp.Data, errStartTimeCannotBeAfterEndTime
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	if future != "" {
		params.Set("future", future)
	}
	endpoint := common.EncodeURLValues(getFundingRates, params)
	return resp.Data, f.SendHTTPRequest(exchange.RestSpot, endpoint, &resp)
}

// GetIndexWeights gets index weights
func (f *FTX) GetIndexWeights(index string) (IndexWeights, error) {
	var resp IndexWeights
	return resp, f.SendHTTPRequest(exchange.RestSpot, fmt.Sprintf(getIndexWeights, index), &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (f *FTX) SendHTTPRequest(ep exchange.URL, path string, result interface{}) error {
	endpoint, err := f.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	return f.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       f.Verbose,
		HTTPDebugging: f.HTTPDebugging,
		HTTPRecording: f.HTTPRecording,
	})
}

// GetMarginBorrowRates gets borrowing rates for margin trading
func (f *FTX) GetMarginBorrowRates() ([]MarginFundingData, error) {
	r := struct {
		Data []MarginFundingData `json:"result"`
	}{}
	return r.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, marginBorrowRates, nil, &r)
}

// GetMarginLendingRates gets lending rates for margin trading
func (f *FTX) GetMarginLendingRates() ([]MarginFundingData, error) {
	r := struct {
		Data []MarginFundingData `json:"result"`
	}{}
	return r.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, marginLendingRates, nil, &r)
}

// MarginDailyBorrowedAmounts gets daily borrowed amounts for margin
func (f *FTX) MarginDailyBorrowedAmounts() ([]MarginDailyBorrowStats, error) {
	r := struct {
		Data []MarginDailyBorrowStats `json:"result"`
	}{}
	return r.Data, f.SendHTTPRequest(exchange.RestSpot, dailyBorrowedAmounts, &r)
}

// GetMarginMarketInfo gets margin market data
func (f *FTX) GetMarginMarketInfo(market string) ([]MarginMarketInfo, error) {
	r := struct {
		Data []MarginMarketInfo `json:"result"`
	}{}
	return r.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, fmt.Sprintf(marginMarketInfo, market), nil, &r)
}

// GetMarginBorrowHistory gets the margin borrow history data
func (f *FTX) GetMarginBorrowHistory(startTime, endTime time.Time) ([]MarginTransactionHistoryData, error) {
	r := struct {
		Data []MarginTransactionHistoryData `json:"result"`
	}{}

	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errStartTimeCannotBeAfterEndTime
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	endpoint := common.EncodeURLValues(marginBorrowHistory, params)
	return r.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, endpoint, nil, &r)
}

// GetMarginMarketLendingHistory gets the markets margin lending rate history
func (f *FTX) GetMarginMarketLendingHistory(coin currency.Code, startTime, endTime time.Time) ([]MarginTransactionHistoryData, error) {
	r := struct {
		Data []MarginTransactionHistoryData `json:"result"`
	}{}
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.Upper().String())
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errStartTimeCannotBeAfterEndTime
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	endpoint := common.EncodeURLValues(marginLendingHistory, params)
	return r.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, endpoint, params, &r)
}

// GetMarginLendingHistory gets margin lending history
func (f *FTX) GetMarginLendingHistory(coin currency.Code, startTime, endTime time.Time) ([]MarginTransactionHistoryData, error) {
	r := struct {
		Data []MarginTransactionHistoryData `json:"result"`
	}{}
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.Upper().String())
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errStartTimeCannotBeAfterEndTime
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	endpoint := common.EncodeURLValues(marginLendHistory, params)
	return r.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, marginLendHistory, endpoint, &r)
}

// GetMarginLendingOffers gets margin lending offers
func (f *FTX) GetMarginLendingOffers() ([]LendingOffersData, error) {
	r := struct {
		Data []LendingOffersData `json:"result"`
	}{}
	return r.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, marginLendingOffers, nil, &r)
}

// GetLendingInfo gets margin lending info
func (f *FTX) GetLendingInfo() ([]LendingInfoData, error) {
	r := struct {
		Data []LendingInfoData `json:"result"`
	}{}
	return r.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, marginLendingInfo, nil, &r)
}

// SubmitLendingOffer submits an offer for margin lending
func (f *FTX) SubmitLendingOffer(coin currency.Code, size, rate float64) error {
	resp := struct {
		Result  string `json:"result"`
		Success bool   `json:"success"`
	}{}
	req := make(map[string]interface{})
	req["coin"] = coin.Upper().String()
	req["size"] = size
	req["rate"] = rate

	if err := f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, marginLendingOffers, req, &resp); err != nil {
		return err
	}

	if !resp.Success {
		return errors.New(resp.Result)
	}
	return nil
}

// GetAccountInfo gets account info
func (f *FTX) GetAccountInfo() (AccountInfoData, error) {
	resp := struct {
		Data AccountInfoData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getAccountInfo, nil, &resp)
}

// GetPositions gets the users positions
func (f *FTX) GetPositions() ([]PositionData, error) {
	resp := struct {
		Data []PositionData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getPositions, nil, &resp)
}

// ChangeAccountLeverage changes default leverage used by account
func (f *FTX) ChangeAccountLeverage(leverage float64) error {
	req := make(map[string]interface{})
	req["leverage"] = leverage
	return f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, setLeverage, req, nil)
}

// GetCoins gets coins' data in the account wallet
func (f *FTX) GetCoins() ([]WalletCoinsData, error) {
	resp := struct {
		Data []WalletCoinsData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getCoins, nil, &resp)
}

// GetBalances gets balances of the account
func (f *FTX) GetBalances() ([]WalletBalance, error) {
	resp := struct {
		Data []WalletBalance `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getBalances, nil, &resp)
}

// GetAllWalletBalances gets all wallets' balances
func (f *FTX) GetAllWalletBalances() (AllWalletBalances, error) {
	resp := struct {
		Data AllWalletBalances `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getAllWalletBalances, nil, &resp)
}

// FetchDepositAddress gets deposit address for a given coin
func (f *FTX) FetchDepositAddress(coin currency.Code) (DepositData, error) {
	resp := struct {
		Data DepositData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getDepositAddress+coin.Upper().String(), nil, &resp)
}

// FetchDepositHistory gets deposit history
func (f *FTX) FetchDepositHistory() ([]TransactionData, error) {
	resp := struct {
		Data []TransactionData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getDepositHistory, nil, &resp)
}

// FetchWithdrawalHistory gets withdrawal history
func (f *FTX) FetchWithdrawalHistory() ([]TransactionData, error) {
	resp := struct {
		Data []TransactionData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getWithdrawalHistory, nil, &resp)
}

// Withdraw sends a withdrawal request
func (f *FTX) Withdraw(coin currency.Code, address, tag, password, code string, size float64) (TransactionData, error) {
	req := make(map[string]interface{})
	req["coin"] = coin.Upper().String()
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
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, withdrawRequest, req, &resp)
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
	endpoint := common.EncodeURLValues(getOpenOrders, params)
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, endpoint, nil, &resp)
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
			return resp.Data, errStartTimeCannotBeAfterEndTime
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	if limit != "" {
		params.Set("limit", limit)
	}
	endpoint := common.EncodeURLValues(getOrderHistory, params)
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, endpoint, nil, &resp)
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
	endpoint := common.EncodeURLValues(getOpenTriggerOrders, params)
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, endpoint, nil, &resp)
}

// GetTriggerOrderTriggers gets trigger orders that are currently open
func (f *FTX) GetTriggerOrderTriggers(orderID string) ([]TriggerData, error) {
	resp := struct {
		Data []TriggerData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, fmt.Sprintf(getTriggerOrderTriggers, orderID), nil, &resp)
}

// GetTriggerOrderHistory gets trigger orders that are currently open
func (f *FTX) GetTriggerOrderHistory(marketName string, startTime, endTime time.Time, side, orderType, limit string) ([]TriggerOrderData, error) {
	params := url.Values{}
	if marketName != "" {
		params.Set("market", marketName)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errStartTimeCannotBeAfterEndTime
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
	resp := struct {
		Data []TriggerOrderData `json:"result"`
	}{}
	endpoint := common.EncodeURLValues(getTriggerOrderHistory, params)
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, endpoint, nil, &resp)
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
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, placeOrder, req, &resp)
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
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, placeTriggerOrder, req, &resp)
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
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, fmt.Sprintf(modifyOrder, orderID), req, &resp)
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
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, fmt.Sprintf(modifyOrderByClientID, clientOrderID), req, &resp)
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
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, fmt.Sprintf(modifyTriggerOrder, orderID), req, &resp)
}

// GetOrderStatus gets the order status of a given orderID
func (f *FTX) GetOrderStatus(orderID string) (OrderData, error) {
	resp := struct {
		Data OrderData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getOrderStatus+orderID, nil, &resp)
}

// GetOrderStatusByClientID gets the order status of a given clientOrderID
func (f *FTX) GetOrderStatusByClientID(clientOrderID string) (OrderData, error) {
	resp := struct {
		Data OrderData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getOrderStatusByClientID+clientOrderID, nil, &resp)
}

func (f *FTX) deleteOrderByPath(path string) (string, error) {
	resp := struct {
		Result  string `json:"result"`
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}{}
	err := f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodDelete, path, nil, &resp)
	// If there is an error reported, but the resp struct reports one of a very few
	// specific error causes, we still consider this a successful cancellation.
	if err != nil && !resp.Success && (resp.Error == "Order already closed" || resp.Error == "Order already queued for cancellation") {
		return resp.Error, nil
	}
	return resp.Result, err
}

// DeleteOrder deletes an order
func (f *FTX) DeleteOrder(orderID string) (string, error) {
	if orderID == "" {
		return "", errInvalidOrderID
	}
	return f.deleteOrderByPath(deleteOrder + orderID)
}

// DeleteOrderByClientID deletes an order
func (f *FTX) DeleteOrderByClientID(clientID string) (string, error) {
	if clientID == "" {
		return "", errInvalidOrderID
	}
	return f.deleteOrderByPath(deleteOrderByClientID + clientID)
}

// DeleteTriggerOrder deletes an order
func (f *FTX) DeleteTriggerOrder(orderID string) (string, error) {
	if orderID == "" {
		return "", errInvalidOrderID
	}
	return f.deleteOrderByPath(cancelTriggerOrder + orderID)
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
			return resp.Data, errStartTimeCannotBeAfterEndTime
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	endpoint := common.EncodeURLValues(getFills, params)
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, endpoint, nil, &resp)
}

// GetFundingPayments gets funding payments
func (f *FTX) GetFundingPayments(startTime, endTime time.Time, future string) ([]FundingPaymentsData, error) {
	resp := struct {
		Data []FundingPaymentsData `json:"result"`
	}{}
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp.Data, errStartTimeCannotBeAfterEndTime
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	if future != "" {
		params.Set("future", future)
	}
	endpoint := common.EncodeURLValues(getFundingPayments, params)
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, endpoint, nil, &resp)
}

// ListLeveragedTokens lists leveraged tokens
func (f *FTX) ListLeveragedTokens() ([]LeveragedTokensData, error) {
	resp := struct {
		Data []LeveragedTokensData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getLeveragedTokens, nil, &resp)
}

// GetTokenInfo gets token info
func (f *FTX) GetTokenInfo(tokenName string) ([]LeveragedTokensData, error) {
	resp := struct {
		Data []LeveragedTokensData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getTokenInfo+tokenName, nil, &resp)
}

// ListLTBalances gets leveraged tokens' balances
func (f *FTX) ListLTBalances() ([]LTBalanceData, error) {
	resp := struct {
		Data []LTBalanceData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getLTBalances, nil, &resp)
}

// ListLTCreations lists the leveraged tokens' creation requests
func (f *FTX) ListLTCreations() ([]LTCreationData, error) {
	resp := struct {
		Data []LTCreationData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getLTCreations, nil, &resp)
}

// RequestLTCreation sends a request to create a leveraged token
func (f *FTX) RequestLTCreation(tokenName string, size float64) (RequestTokenCreationData, error) {
	req := make(map[string]interface{})
	req["size"] = size
	resp := struct {
		Data RequestTokenCreationData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, fmt.Sprintf(requestLTCreation, tokenName), req, &resp)
}

// ListLTRedemptions lists the leveraged tokens' redemption requests
func (f *FTX) ListLTRedemptions() ([]LTRedemptionData, error) {
	resp := struct {
		Data []LTRedemptionData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getLTRedemptions, nil, &resp)
}

// RequestLTRedemption sends a request to redeem a leveraged token
func (f *FTX) RequestLTRedemption(tokenName string, size float64) (LTRedemptionRequestData, error) {
	req := make(map[string]interface{})
	req["size"] = size
	resp := struct {
		Data LTRedemptionRequestData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, fmt.Sprintf(requestLTRedemption, tokenName), req, &resp)
}

// GetQuoteRequests gets a list of quote requests
func (f *FTX) GetQuoteRequests() ([]QuoteRequestData, error) {
	resp := struct {
		Data []QuoteRequestData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getListQuotes, nil, &resp)
}

// GetYourQuoteRequests gets a list of your quote requests
func (f *FTX) GetYourQuoteRequests() ([]PersonalQuotesData, error) {
	resp := struct {
		Data []PersonalQuotesData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getMyQuotesRequests, nil, &resp)
}

// CreateQuoteRequest sends a request to create a quote
func (f *FTX) CreateQuoteRequest(underlying currency.Code, optionType, side string, expiry int64, requestExpiry string, strike, size, limitPrice, counterPartyID float64, hideLimitPrice bool) (CreateQuoteRequestData, error) {
	req := make(map[string]interface{})
	req["underlying"] = underlying.Upper().String()
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
	if counterPartyID != 0 {
		req["counterpartyId"] = counterPartyID
	}
	req["hideLimitPrice"] = hideLimitPrice
	resp := struct {
		Data CreateQuoteRequestData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, createQuoteRequest, req, &resp)
}

// DeleteQuote sends request to cancel a quote
func (f *FTX) DeleteQuote(requestID string) (CancelQuoteRequestData, error) {
	resp := struct {
		Data CancelQuoteRequestData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodDelete, deleteQuote+requestID, nil, &resp)
}

// GetQuotesForYourQuote gets a list of quotes for your quote
func (f *FTX) GetQuotesForYourQuote(requestID string) (QuoteForQuoteData, error) {
	var resp QuoteForQuoteData
	return resp, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, fmt.Sprintf(endpointQuote, requestID), nil, &resp)
}

// MakeQuote makes a quote for a quote
func (f *FTX) MakeQuote(requestID, price string) ([]QuoteForQuoteData, error) {
	params := url.Values{}
	params.Set("price", price)
	resp := struct {
		Data []QuoteForQuoteData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, fmt.Sprintf(endpointQuote, requestID), nil, &resp)
}

// MyQuotes gets a list of my quotes for quotes
func (f *FTX) MyQuotes() ([]QuoteForQuoteData, error) {
	resp := struct {
		Data []QuoteForQuoteData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getMyQuotes, nil, &resp)
}

// DeleteMyQuote deletes my quote for quotes
func (f *FTX) DeleteMyQuote(quoteID string) ([]QuoteForQuoteData, error) {
	resp := struct {
		Data []QuoteForQuoteData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodDelete, deleteMyQuote+quoteID, nil, &resp)
}

// AcceptQuote accepts the quote for quote
func (f *FTX) AcceptQuote(quoteID string) ([]QuoteForQuoteData, error) {
	resp := struct {
		Data []QuoteForQuoteData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, fmt.Sprintf(acceptQuote, quoteID), nil, &resp)
}

// GetAccountOptionsInfo gets account's options' info
func (f *FTX) GetAccountOptionsInfo() (AccountOptionsInfoData, error) {
	resp := struct {
		Data AccountOptionsInfoData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getOptionsInfo, nil, &resp)
}

// GetOptionsPositions gets options' positions
func (f *FTX) GetOptionsPositions() ([]OptionsPositionsData, error) {
	resp := struct {
		Data []OptionsPositionsData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getOptionsPositions, nil, &resp)
}

// GetPublicOptionsTrades gets options' trades from public
func (f *FTX) GetPublicOptionsTrades(startTime, endTime time.Time, limit string) ([]OptionsTradesData, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errStartTimeCannotBeAfterEndTime
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	if limit != "" {
		params.Set("limit", limit)
	}
	resp := struct {
		Data []OptionsTradesData `json:"result"`
	}{}
	endpoint := common.EncodeURLValues(getPublicOptionsTrades, params)
	return resp.Data, f.SendHTTPRequest(exchange.RestSpot, endpoint, &resp)
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
			return resp.Data, errStartTimeCannotBeAfterEndTime
		}
	}
	if limit != "" {
		req["limit"] = limit
	}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getOptionsFills, req, &resp)
}

// GetStakes returns a list of staked assets
func (f *FTX) GetStakes() ([]Stake, error) {
	resp := struct {
		Data []Stake `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, stakes, nil, &resp)
}

// GetUnstakeRequests returns a collection of unstake requests
func (f *FTX) GetUnstakeRequests() ([]UnstakeRequest, error) {
	resp := struct {
		Data []UnstakeRequest `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, unstakeRequests, nil, &resp)
}

// GetStakeBalances returns a collection of staked coin balances
func (f *FTX) GetStakeBalances() ([]StakeBalance, error) {
	resp := struct {
		Data []StakeBalance `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, stakeBalances, nil, &resp)
}

// UnstakeRequest unstakes an existing staked coin
func (f *FTX) UnstakeRequest(coin currency.Code, size float64) (*UnstakeRequest, error) {
	resp := struct {
		Data UnstakeRequest `json:"result"`
	}{}
	req := make(map[string]interface{})
	req["coin"] = coin.Upper().String()
	req["size"] = strconv.FormatFloat(size, 'f', -1, 64)
	return &resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, unstakeRequests, req, &resp)
}

// CancelUnstakeRequest cancels a pending unstake request
func (f *FTX) CancelUnstakeRequest(requestID int64) (bool, error) {
	resp := struct {
		Result string
	}{}
	path := unstakeRequests + "/" + strconv.FormatInt(requestID, 10)
	if err := f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodDelete, path, nil, &resp); err != nil {
		return false, err
	}

	if resp.Result != "Cancelled" {
		return false, errors.New("failed to cancel unstake request")
	}
	return true, nil
}

// GetStakingRewards returns a collection of staking rewards
func (f *FTX) GetStakingRewards() ([]StakeReward, error) {
	resp := struct {
		Data []StakeReward `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, stakingRewards, nil, &resp)
}

// StakeRequest submits a stake request based on the specified currency and size
func (f *FTX) StakeRequest(coin currency.Code, size float64) (*Stake, error) {
	resp := struct {
		Data Stake `json:"result"`
	}{}
	req := make(map[string]interface{})
	req["coin"] = coin.Upper().String()
	req["size"] = strconv.FormatFloat(size, 'f', -1, 64)
	return &resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, serumStakes, req, &resp)
}

// SendAuthHTTPRequest sends an authenticated request
func (f *FTX) SendAuthHTTPRequest(ep exchange.URL, method, path string, data, result interface{}) error {
	endpoint, err := f.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	ts := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
	var body io.Reader
	var hmac, payload []byte
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
	if f.API.Credentials.Subaccount != "" {
		headers["FTX-SUBACCOUNT"] = url.QueryEscape(f.API.Credentials.Subaccount)
	}
	headers["Content-Type"] = "application/json"
	return f.SendPayload(context.Background(), &request.Item{
		Method:        method,
		Path:          endpoint + path,
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
	if !f.GetAuthenticatedAPISupport(exchange.RestAuthentication) {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
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
	if filledAmount > amount {
		return OrderVars{}, fmt.Errorf("%w, amount: %f filled: %f", errInvalidOrderAmounts, amount, filledAmount)
	}
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
			break
		}
		if filledAmount == 0 {
			resp.Status = order.Cancelled
			break
		}
		if filledAmount == amount {
			resp.Status = order.Filled
		}
	default:
		return resp, fmt.Errorf("%w %s", errUnrecognisedOrderStatus, orderStatus)
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
func (f *FTX) RequestForQuotes(base, quote currency.Code, amount float64) (RequestQuoteData, error) {
	resp := struct {
		Data RequestQuoteData `json:"result"`
	}{}
	req := make(map[string]interface{})
	req["fromCoin"] = base.Upper().String()
	req["toCoin"] = quote.Upper().String()
	req["size"] = amount
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, requestOTCQuote, req, &resp)
}

// GetOTCQuoteStatus gets quote status of a quote
func (f *FTX) GetOTCQuoteStatus(marketName, quoteID string) ([]QuoteStatusData, error) {
	resp := struct {
		Data []QuoteStatusData `json:"result"`
	}{}
	params := url.Values{}
	params.Set("market", marketName)
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, getOTCQuoteStatus+quoteID, params, &resp)
}

// AcceptOTCQuote requests for otc quotes
func (f *FTX) AcceptOTCQuote(quoteID string) error {
	return f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, fmt.Sprintf(acceptOTCQuote, quoteID), nil, nil)
}

// GetSubaccounts returns the users subaccounts
func (f *FTX) GetSubaccounts() ([]Subaccount, error) {
	resp := struct {
		Data []Subaccount `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, subaccounts, nil, &resp)
}

// CreateSubaccount creates a new subaccount
func (f *FTX) CreateSubaccount(name string) (*Subaccount, error) {
	if name == "" {
		return nil, errSubaccountNameMustBeSpecified
	}
	d := make(map[string]string)
	d["nickname"] = name

	resp := struct {
		Data Subaccount `json:"result"`
	}{}
	if err := f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, subaccounts, d, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// UpdateSubaccountName updates an existing subaccount name
func (f *FTX) UpdateSubaccountName(oldName, newName string) (*Subaccount, error) {
	if oldName == "" || newName == "" || oldName == newName {
		return nil, errSubaccountUpdateNameInvalid
	}
	d := make(map[string]string)
	d["nickname"] = oldName
	d["newNickname"] = newName

	resp := struct {
		Data Subaccount `json:"result"`
	}{}
	if err := f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, subaccountsUpdateName, d, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// DeleteSubaccount deletes the specified subaccount name
func (f *FTX) DeleteSubaccount(name string) error {
	if name == "" {
		return errSubaccountNameMustBeSpecified
	}
	d := make(map[string]string)
	d["nickname"] = name
	resp := struct {
		Data Subaccount `json:"result"`
	}{}
	return f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodDelete, subaccounts, d, &resp)
}

// SubaccountBalances returns the user's subaccount balances
func (f *FTX) SubaccountBalances(name string) ([]SubaccountBalance, error) {
	if name == "" {
		return nil, errSubaccountNameMustBeSpecified
	}
	resp := struct {
		Data []SubaccountBalance `json:"result"`
	}{}
	if err := f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet, fmt.Sprintf(subaccountsBalance, name), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// SubaccountTransfer transfers a desired coin to the specified subaccount
func (f *FTX) SubaccountTransfer(coin currency.Code, source, destination string, size float64) (*SubaccountTransferStatus, error) {
	if coin.IsEmpty() {
		return nil, errCoinMustBeSpecified
	}
	if size <= 0 {
		return nil, errSubaccountTransferSizeGreaterThanZero
	}
	if source == destination {
		return nil, errSubaccountTransferSourceDestinationMustNotBeEqual
	}
	d := make(map[string]interface{})
	d["coin"] = coin.Upper().String()
	d["size"] = size
	if source == "" {
		source = "main"
	}
	d["source"] = source
	if destination == "" {
		destination = "main"
	}
	d["destination"] = destination
	resp := struct {
		Data SubaccountTransferStatus `json:"result"`
	}{}
	if err := f.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost, subaccountsTransfer, d, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// FetchExchangeLimits fetches spot order execution limits
func (f *FTX) FetchExchangeLimits() ([]order.MinMaxLevel, error) {
	data, err := f.GetMarkets()
	if err != nil {
		return nil, err
	}

	var limits []order.MinMaxLevel
	for x := range data {
		if !data[x].Enabled {
			continue
		}
		var cp currency.Pair
		var a asset.Item
		switch data[x].MarketType {
		case "future":
			a = asset.Futures
			cp, err = currency.NewPairFromString(data[x].Name)
			if err != nil {
				return nil, err
			}
		case "spot":
			a = asset.Spot
			cp, err = currency.NewPairFromStrings(data[x].BaseCurrency, data[x].QuoteCurrency)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unhandled data type %s, cannot process exchange limit",
				data[x].MarketType)
		}

		limits = append(limits, order.MinMaxLevel{
			Pair:       cp,
			Asset:      a,
			StepPrice:  data[x].PriceIncrement,
			StepAmount: data[x].SizeIncrement,
			MinAmount:  data[x].MinProvideSize,
		})
	}
	return limits, nil
}
