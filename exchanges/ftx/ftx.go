package ftx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
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
	collateralWeight CollateralWeightHolder
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
	getExpiredFutures    = "/expired_futures"
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
	collateral               = "/wallet/collateral"
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
	errCollateralCurrencyNotFound                        = errors.New("no collateral scaling information found")
	errCollateralInitialMarginFractionMissing            = errors.New("cannot scale collateral, missing initial margin fraction information")

	validResolutionData = []int64{15, 60, 300, 900, 3600, 14400, 86400}
)

// GetHistoricalIndex gets historical index data
func (f *FTX) GetHistoricalIndex(ctx context.Context, indexName string, resolution int64, startTime, endTime time.Time) ([]OHLCVData, error) {
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
	return resp.Data, f.SendHTTPRequest(ctx, exchange.RestSpot, endpoint, &resp)
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
func (f *FTX) GetMarkets(ctx context.Context) ([]MarketData, error) {
	resp := struct {
		Data []MarketData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(ctx, exchange.RestSpot, getMarkets, &resp)
}

// GetMarket gets market data for a provided asset type
func (f *FTX) GetMarket(ctx context.Context, marketName string) (MarketData, error) {
	resp := struct {
		Data MarketData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(ctx, exchange.RestSpot, getMarket+marketName,
		&resp)
}

// GetOrderbook gets orderbook for a given market with a given depth (default depth 20)
func (f *FTX) GetOrderbook(ctx context.Context, marketName string, depth int64) (OrderbookData, error) {
	result := struct {
		Data TempOBData `json:"result"`
	}{}

	strDepth := "20" // If we send a zero value we get zero asks from the
	// endpoint
	if depth != 0 {
		strDepth = strconv.FormatInt(depth, 10)
	}

	var resp OrderbookData
	err := f.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf(getOrderbook, marketName, strDepth), &result)
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
func (f *FTX) GetTrades(ctx context.Context, marketName string, startTime, endTime, limit int64) ([]TradeData, error) {
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
	return resp.Data, f.SendHTTPRequest(ctx, exchange.RestSpot, endpoint, &resp)
}

// GetHistoricalData gets historical OHLCV data for a given market pair
func (f *FTX) GetHistoricalData(ctx context.Context, marketName string, timeInterval, limit int64, startTime, endTime time.Time) ([]OHLCVData, error) {
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
	return resp.Data, f.SendHTTPRequest(ctx, exchange.RestSpot, endpoint, &resp)
}

// GetFutures gets data on futures
func (f *FTX) GetFutures(ctx context.Context) ([]FuturesData, error) {
	resp := struct {
		Data []FuturesData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(ctx, exchange.RestSpot, getFutures, &resp)
}

// GetFuture gets data on a given future
func (f *FTX) GetFuture(ctx context.Context, futureName string) (FuturesData, error) {
	resp := struct {
		Data FuturesData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(ctx, exchange.RestSpot, getFuture+futureName, &resp)
}

// GetFutureStats gets data on a given future's stats
func (f *FTX) GetFutureStats(ctx context.Context, futureName string) (FutureStatsData, error) {
	resp := struct {
		Data FutureStatsData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf(getFutureStats, futureName), &resp)
}

// GetExpiredFuture returns information on an expired futures contract
func (f *FTX) GetExpiredFuture(ctx context.Context, pair currency.Pair) (FuturesData, error) {
	p, err := f.FormatSymbol(pair, asset.Futures)
	if err != nil {
		return FuturesData{}, err
	}
	resp, err := f.GetExpiredFutures(ctx)
	if err != nil {
		return FuturesData{}, err
	}
	for i := range resp {
		if resp[i].Name == p {
			return resp[i], nil
		}
	}
	return FuturesData{}, fmt.Errorf("%s %s %w", f.Name, p, currency.ErrPairNotFound)
}

// GetExpiredFutures returns information on expired futures contracts
func (f *FTX) GetExpiredFutures(ctx context.Context) ([]FuturesData, error) {
	resp := struct {
		Data []FuturesData `json:"result"`
	}{}
	return resp.Data, f.SendHTTPRequest(ctx, exchange.RestSpot, getExpiredFutures, &resp)
}

// GetFundingRates gets data on funding rates
func (f *FTX) GetFundingRates(ctx context.Context, startTime, endTime time.Time, future string) ([]FundingRatesData, error) {
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
	return resp.Data, f.SendHTTPRequest(ctx, exchange.RestSpot, endpoint, &resp)
}

// GetIndexWeights gets index weights
func (f *FTX) GetIndexWeights(ctx context.Context, index string) (IndexWeights, error) {
	var resp IndexWeights
	return resp, f.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf(getIndexWeights, index), &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (f *FTX) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result interface{}) error {
	endpoint, err := f.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       f.Verbose,
		HTTPDebugging: f.HTTPDebugging,
		HTTPRecording: f.HTTPRecording,
	}
	return f.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	})
}

// GetMarginBorrowRates gets borrowing rates for margin trading
func (f *FTX) GetMarginBorrowRates(ctx context.Context) ([]MarginFundingData, error) {
	r := struct {
		Data []MarginFundingData `json:"result"`
	}{}
	return r.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginBorrowRates, nil, &r)
}

// GetMarginLendingRates gets lending rates for margin trading
func (f *FTX) GetMarginLendingRates(ctx context.Context) ([]MarginFundingData, error) {
	r := struct {
		Data []MarginFundingData `json:"result"`
	}{}
	return r.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginLendingRates, nil, &r)
}

// MarginDailyBorrowedAmounts gets daily borrowed amounts for margin
func (f *FTX) MarginDailyBorrowedAmounts(ctx context.Context) ([]MarginDailyBorrowStats, error) {
	r := struct {
		Data []MarginDailyBorrowStats `json:"result"`
	}{}
	return r.Data, f.SendHTTPRequest(ctx, exchange.RestSpot, dailyBorrowedAmounts, &r)
}

// GetMarginMarketInfo gets margin market data
func (f *FTX) GetMarginMarketInfo(ctx context.Context, market string) ([]MarginMarketInfo, error) {
	r := struct {
		Data []MarginMarketInfo `json:"result"`
	}{}
	return r.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, fmt.Sprintf(marginMarketInfo, market), nil, &r)
}

// GetMarginBorrowHistory gets the margin borrow history data
func (f *FTX) GetMarginBorrowHistory(ctx context.Context, startTime, endTime time.Time) ([]MarginTransactionHistoryData, error) {
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
	return r.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, endpoint, nil, &r)
}

// GetMarginMarketLendingHistory gets the markets margin lending rate history
func (f *FTX) GetMarginMarketLendingHistory(ctx context.Context, coin currency.Code, startTime, endTime time.Time) ([]MarginTransactionHistoryData, error) {
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
	return r.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, endpoint, params, &r)
}

// GetMarginLendingHistory gets margin lending history
func (f *FTX) GetMarginLendingHistory(ctx context.Context, coin currency.Code, startTime, endTime time.Time) ([]MarginTransactionHistoryData, error) {
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
	return r.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginLendHistory, endpoint, &r)
}

// GetMarginLendingOffers gets margin lending offers
func (f *FTX) GetMarginLendingOffers(ctx context.Context) ([]LendingOffersData, error) {
	r := struct {
		Data []LendingOffersData `json:"result"`
	}{}
	return r.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginLendingOffers, nil, &r)
}

// GetLendingInfo gets margin lending info
func (f *FTX) GetLendingInfo(ctx context.Context) ([]LendingInfoData, error) {
	r := struct {
		Data []LendingInfoData `json:"result"`
	}{}
	return r.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginLendingInfo, nil, &r)
}

// SubmitLendingOffer submits an offer for margin lending
func (f *FTX) SubmitLendingOffer(ctx context.Context, coin currency.Code, size, rate float64) error {
	resp := struct {
		Result  string `json:"result"`
		Success bool   `json:"success"`
	}{}
	req := make(map[string]interface{})
	req["coin"] = coin.Upper().String()
	req["size"] = size
	req["rate"] = rate

	if err := f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, marginLendingOffers, req, &resp); err != nil {
		return err
	}

	if !resp.Success {
		return errors.New(resp.Result)
	}
	return nil
}

// GetAccountInfo gets account info
func (f *FTX) GetAccountInfo(ctx context.Context) (AccountInfoData, error) {
	resp := struct {
		Data AccountInfoData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getAccountInfo, nil, &resp)
}

// GetPositions gets the users positions
func (f *FTX) GetPositions(ctx context.Context) ([]PositionData, error) {
	resp := struct {
		Data []PositionData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getPositions, nil, &resp)
}

// ChangeAccountLeverage changes default leverage used by account
func (f *FTX) ChangeAccountLeverage(ctx context.Context, leverage float64) error {
	req := make(map[string]interface{})
	req["leverage"] = leverage
	return f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, setLeverage, req, nil)
}

// GetCoins gets coins' data in the account wallet
func (f *FTX) GetCoins(ctx context.Context) ([]WalletCoinsData, error) {
	resp := struct {
		Data []WalletCoinsData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getCoins, nil, &resp)
}

// GetBalances gets balances of the account
func (f *FTX) GetBalances(ctx context.Context, includeLockedBreakdown, includeFreeIgnoringCollateral bool) ([]WalletBalance, error) {
	resp := struct {
		Data []WalletBalance `json:"result"`
	}{}
	vals := url.Values{}
	if includeLockedBreakdown {
		vals.Set("includeLockedBreakdown", strconv.FormatBool(includeLockedBreakdown))
	}
	if includeFreeIgnoringCollateral {
		vals.Set("includeFreeIgnoringCollateral", strconv.FormatBool(includeFreeIgnoringCollateral))
	}
	balanceURL := common.EncodeURLValues(getBalances, vals)
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, balanceURL, nil, &resp)
}

// GetAllWalletBalances gets all wallets' balances
func (f *FTX) GetAllWalletBalances(ctx context.Context) (AllWalletBalances, error) {
	resp := struct {
		Data AllWalletBalances `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getAllWalletBalances, nil, &resp)
}

// FetchDepositAddress gets deposit address for a given coin
func (f *FTX) FetchDepositAddress(ctx context.Context, coin currency.Code, chain string) (*DepositData, error) {
	resp := struct {
		Data DepositData `json:"result"`
	}{}
	vals := url.Values{}
	if chain != "" {
		vals.Set("method", strings.ToLower(chain))
	}
	path := common.EncodeURLValues(getDepositAddress+coin.Upper().String(), vals)
	return &resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// FetchDepositHistory gets deposit history
func (f *FTX) FetchDepositHistory(ctx context.Context) ([]DepositItem, error) {
	resp := struct {
		Data []DepositItem `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getDepositHistory, nil, &resp)
}

// FetchWithdrawalHistory gets withdrawal history
func (f *FTX) FetchWithdrawalHistory(ctx context.Context) ([]WithdrawItem, error) {
	resp := struct {
		Data []WithdrawItem `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getWithdrawalHistory, nil, &resp)
}

// Withdraw sends a withdrawal request
func (f *FTX) Withdraw(ctx context.Context, coin currency.Code, address, tag, password, chain, code string, size float64) (*WithdrawItem, error) {
	if coin.IsEmpty() || address == "" || size == 0 {
		return nil, errors.New("coin, address and size must be specified")
	}

	req := make(map[string]interface{})
	req["coin"] = coin.Upper().String()
	req["size"] = size
	req["address"] = address
	if code != "" {
		req["code"] = code
	}
	if tag != "" {
		req["tag"] = tag
	}
	if password != "" {
		req["password"] = password
	}
	if chain != "" {
		req["method"] = chain
	}
	resp := struct {
		Data WithdrawItem `json:"result"`
	}{}
	return &resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, withdrawRequest, req, &resp)
}

// GetOpenOrders gets open orders
func (f *FTX) GetOpenOrders(ctx context.Context, marketName string) ([]OrderData, error) {
	params := url.Values{}
	if marketName != "" {
		params.Set("market", marketName)
	}
	resp := struct {
		Data []OrderData `json:"result"`
	}{}
	endpoint := common.EncodeURLValues(getOpenOrders, params)
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, endpoint, nil, &resp)
}

// FetchOrderHistory gets order history
func (f *FTX) FetchOrderHistory(ctx context.Context, marketName string, startTime, endTime time.Time, limit string) ([]OrderData, error) {
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
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, endpoint, nil, &resp)
}

// GetOpenTriggerOrders gets trigger orders that are currently open
func (f *FTX) GetOpenTriggerOrders(ctx context.Context, marketName, orderType string) ([]TriggerOrderData, error) {
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
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, endpoint, nil, &resp)
}

// GetTriggerOrderTriggers gets trigger orders that are currently open
func (f *FTX) GetTriggerOrderTriggers(ctx context.Context, orderID string) ([]TriggerData, error) {
	resp := struct {
		Data []TriggerData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, fmt.Sprintf(getTriggerOrderTriggers, orderID), nil, &resp)
}

// GetTriggerOrderHistory gets trigger orders that are currently open
func (f *FTX) GetTriggerOrderHistory(ctx context.Context, marketName string, startTime, endTime time.Time, side, orderType, limit string) ([]TriggerOrderData, error) {
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
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, endpoint, nil, &resp)
}

// Order places an order
func (f *FTX) Order(
	ctx context.Context,
	marketName, side, orderType string,
	reduceOnly, ioc, postOnly bool,
	clientID string,
	price, size float64,
) (OrderData, error) {
	req := make(map[string]interface{})
	req["market"] = marketName
	req["side"] = side
	req["price"] = price
	req["type"] = orderType
	req["size"] = size
	if reduceOnly {
		req["reduceOnly"] = reduceOnly
	}
	if ioc {
		req["ioc"] = ioc
	}
	if postOnly {
		req["postOnly"] = postOnly
	}
	if clientID != "" {
		req["clientId"] = clientID
	}
	resp := struct {
		Data OrderData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, placeOrder, req, &resp)
}

// TriggerOrder places an order
func (f *FTX) TriggerOrder(ctx context.Context, marketName, side, orderType, reduceOnly, retryUntilFilled string, size, triggerPrice, orderPrice, trailValue float64) (TriggerOrderData, error) {
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
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, placeTriggerOrder, req, &resp)
}

// ModifyPlacedOrder modifies a placed order
func (f *FTX) ModifyPlacedOrder(ctx context.Context, orderID, clientID string, price, size float64) (OrderData, error) {
	req := make(map[string]interface{})
	req["price"] = price
	req["size"] = size
	if clientID != "" {
		req["clientID"] = clientID
	}
	resp := struct {
		Data OrderData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, fmt.Sprintf(modifyOrder, orderID), req, &resp)
}

// ModifyOrderByClientID modifies a placed order via clientOrderID
func (f *FTX) ModifyOrderByClientID(ctx context.Context, clientOrderID, clientID string, price, size float64) (OrderData, error) {
	req := make(map[string]interface{})
	req["price"] = price
	req["size"] = size
	if clientID != "" {
		req["clientID"] = clientID
	}
	resp := struct {
		Data OrderData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, fmt.Sprintf(modifyOrderByClientID, clientOrderID), req, &resp)
}

// ModifyTriggerOrder modifies an existing trigger order
// Choices for ordertype include stop, trailingStop, takeProfit
func (f *FTX) ModifyTriggerOrder(ctx context.Context, orderID, orderType string, size, triggerPrice, orderPrice, trailValue float64) (TriggerOrderData, error) {
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
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, fmt.Sprintf(modifyTriggerOrder, orderID), req, &resp)
}

// GetOrderStatus gets the order status of a given orderID
func (f *FTX) GetOrderStatus(ctx context.Context, orderID string) (OrderData, error) {
	resp := struct {
		Data OrderData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getOrderStatus+orderID, nil, &resp)
}

// GetOrderStatusByClientID gets the order status of a given clientOrderID
func (f *FTX) GetOrderStatusByClientID(ctx context.Context, clientOrderID string) (OrderData, error) {
	resp := struct {
		Data OrderData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getOrderStatusByClientID+clientOrderID, nil, &resp)
}

func (f *FTX) deleteOrderByPath(ctx context.Context, path string) (string, error) {
	resp := struct {
		Result  string `json:"result"`
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}{}
	err := f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, &resp)
	// If there is an error reported, but the resp struct reports one of a very few
	// specific error causes, we still consider this a successful cancellation.
	if err != nil && !resp.Success && (resp.Error == "Order already closed" || resp.Error == "Order already queued for cancellation") {
		return resp.Error, nil
	}
	return resp.Result, err
}

// DeleteOrder deletes an order
func (f *FTX) DeleteOrder(ctx context.Context, orderID string) (string, error) {
	if orderID == "" {
		return "", errInvalidOrderID
	}
	return f.deleteOrderByPath(ctx, deleteOrder+orderID)
}

// DeleteOrderByClientID deletes an order
func (f *FTX) DeleteOrderByClientID(ctx context.Context, clientID string) (string, error) {
	if clientID == "" {
		return "", errInvalidOrderID
	}
	return f.deleteOrderByPath(ctx, deleteOrderByClientID+clientID)
}

// DeleteTriggerOrder deletes an order
func (f *FTX) DeleteTriggerOrder(ctx context.Context, orderID string) (string, error) {
	if orderID == "" {
		return "", errInvalidOrderID
	}
	return f.deleteOrderByPath(ctx, cancelTriggerOrder+orderID)
}

// GetFills gets order fills data and ensures that all
// fills are retrieved from the supplied timeframe
func (f *FTX) GetFills(ctx context.Context, market currency.Pair, item asset.Item, startTime, endTime time.Time) ([]FillsData, error) {
	var resp []FillsData
	var nextEnd = endTime
	limit := 200
	for {
		data := struct {
			Data []FillsData `json:"result"`
		}{}
		params := url.Values{}
		params.Add("limit", strconv.FormatInt(int64(limit), 10))
		if !market.IsEmpty() {
			fp, err := f.FormatExchangeCurrency(market, item)
			if err != nil {
				return nil, err
			}
			params.Set("market", fp.String())
		}
		if !startTime.IsZero() && !endTime.IsZero() {
			if startTime.After(endTime) {
				return data.Data, errStartTimeCannotBeAfterEndTime
			}
			params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
			params.Set("end_time", strconv.FormatInt(nextEnd.Unix(), 10))
		}
		endpoint := common.EncodeURLValues(getFills, params)
		err := f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, endpoint, nil, &data)
		if err != nil {
			return nil, err
		}
		if len(data.Data) == 0 ||
			data.Data[len(data.Data)-1].Time.Equal(nextEnd) {
			break
		}
	data:
		for i := range data.Data {
			for j := range resp {
				if resp[j].ID == data.Data[i].ID {
					continue data
				}
			}
			resp = append(resp, data.Data[i])
		}
		if len(data.Data) < limit {
			break
		}
		nextEnd = data.Data[len(data.Data)-1].Time
	}
	sort.Slice(resp, func(i, j int) bool {
		return resp[i].Time.Before(resp[j].Time)
	})
	return resp, nil
}

// GetFundingPayments gets funding payments
func (f *FTX) GetFundingPayments(ctx context.Context, startTime, endTime time.Time, future string) ([]FundingPaymentsData, error) {
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
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, endpoint, nil, &resp)
}

// ListLeveragedTokens lists leveraged tokens
func (f *FTX) ListLeveragedTokens(ctx context.Context) ([]LeveragedTokensData, error) {
	resp := struct {
		Data []LeveragedTokensData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getLeveragedTokens, nil, &resp)
}

// GetTokenInfo gets token info
func (f *FTX) GetTokenInfo(ctx context.Context, tokenName string) ([]LeveragedTokensData, error) {
	resp := struct {
		Data []LeveragedTokensData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getTokenInfo+tokenName, nil, &resp)
}

// ListLTBalances gets leveraged tokens' balances
func (f *FTX) ListLTBalances(ctx context.Context) ([]LTBalanceData, error) {
	resp := struct {
		Data []LTBalanceData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getLTBalances, nil, &resp)
}

// ListLTCreations lists the leveraged tokens' creation requests
func (f *FTX) ListLTCreations(ctx context.Context) ([]LTCreationData, error) {
	resp := struct {
		Data []LTCreationData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getLTCreations, nil, &resp)
}

// RequestLTCreation sends a request to create a leveraged token
func (f *FTX) RequestLTCreation(ctx context.Context, tokenName string, size float64) (RequestTokenCreationData, error) {
	req := make(map[string]interface{})
	req["size"] = size
	resp := struct {
		Data RequestTokenCreationData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, fmt.Sprintf(requestLTCreation, tokenName), req, &resp)
}

// ListLTRedemptions lists the leveraged tokens' redemption requests
func (f *FTX) ListLTRedemptions(ctx context.Context) ([]LTRedemptionData, error) {
	resp := struct {
		Data []LTRedemptionData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getLTRedemptions, nil, &resp)
}

// RequestLTRedemption sends a request to redeem a leveraged token
func (f *FTX) RequestLTRedemption(ctx context.Context, tokenName string, size float64) (LTRedemptionRequestData, error) {
	req := make(map[string]interface{})
	req["size"] = size
	resp := struct {
		Data LTRedemptionRequestData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, fmt.Sprintf(requestLTRedemption, tokenName), req, &resp)
}

// GetQuoteRequests gets a list of quote requests
func (f *FTX) GetQuoteRequests(ctx context.Context) ([]QuoteRequestData, error) {
	resp := struct {
		Data []QuoteRequestData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getListQuotes, nil, &resp)
}

// GetYourQuoteRequests gets a list of your quote requests
func (f *FTX) GetYourQuoteRequests(ctx context.Context) ([]PersonalQuotesData, error) {
	resp := struct {
		Data []PersonalQuotesData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getMyQuotesRequests, nil, &resp)
}

// CreateQuoteRequest sends a request to create a quote
func (f *FTX) CreateQuoteRequest(ctx context.Context, underlying currency.Code, optionType, side string, expiry int64, requestExpiry string, strike, size, limitPrice, counterPartyID float64, hideLimitPrice bool) (CreateQuoteRequestData, error) {
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
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, createQuoteRequest, req, &resp)
}

// DeleteQuote sends request to cancel a quote
func (f *FTX) DeleteQuote(ctx context.Context, requestID string) (CancelQuoteRequestData, error) {
	resp := struct {
		Data CancelQuoteRequestData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, deleteQuote+requestID, nil, &resp)
}

// GetQuotesForYourQuote gets a list of quotes for your quote
func (f *FTX) GetQuotesForYourQuote(ctx context.Context, requestID string) (QuoteForQuoteData, error) {
	var resp QuoteForQuoteData
	return resp, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, fmt.Sprintf(endpointQuote, requestID), nil, &resp)
}

// MakeQuote makes a quote for a quote
func (f *FTX) MakeQuote(ctx context.Context, requestID, price string) ([]QuoteForQuoteData, error) {
	params := url.Values{}
	params.Set("price", price)
	resp := struct {
		Data []QuoteForQuoteData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, fmt.Sprintf(endpointQuote, requestID), nil, &resp)
}

// MyQuotes gets a list of my quotes for quotes
func (f *FTX) MyQuotes(ctx context.Context) ([]QuoteForQuoteData, error) {
	resp := struct {
		Data []QuoteForQuoteData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getMyQuotes, nil, &resp)
}

// DeleteMyQuote deletes my quote for quotes
func (f *FTX) DeleteMyQuote(ctx context.Context, quoteID string) ([]QuoteForQuoteData, error) {
	resp := struct {
		Data []QuoteForQuoteData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, deleteMyQuote+quoteID, nil, &resp)
}

// AcceptQuote accepts the quote for quote
func (f *FTX) AcceptQuote(ctx context.Context, quoteID string) ([]QuoteForQuoteData, error) {
	resp := struct {
		Data []QuoteForQuoteData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, fmt.Sprintf(acceptQuote, quoteID), nil, &resp)
}

// GetAccountOptionsInfo gets account's options' info
func (f *FTX) GetAccountOptionsInfo(ctx context.Context) (AccountOptionsInfoData, error) {
	resp := struct {
		Data AccountOptionsInfoData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getOptionsInfo, nil, &resp)
}

// GetOptionsPositions gets options' positions
func (f *FTX) GetOptionsPositions(ctx context.Context) ([]OptionsPositionsData, error) {
	resp := struct {
		Data []OptionsPositionsData `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getOptionsPositions, nil, &resp)
}

// GetPublicOptionsTrades gets options' trades from public
func (f *FTX) GetPublicOptionsTrades(ctx context.Context, startTime, endTime time.Time, limit string) ([]OptionsTradesData, error) {
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
	return resp.Data, f.SendHTTPRequest(ctx, exchange.RestSpot, endpoint, &resp)
}

// GetOptionsFills gets fills data for options
func (f *FTX) GetOptionsFills(ctx context.Context, startTime, endTime time.Time, limit string) ([]OptionFillsData, error) {
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
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getOptionsFills, req, &resp)
}

// GetStakes returns a list of staked assets
func (f *FTX) GetStakes(ctx context.Context) ([]Stake, error) {
	resp := struct {
		Data []Stake `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, stakes, nil, &resp)
}

// GetUnstakeRequests returns a collection of unstake requests
func (f *FTX) GetUnstakeRequests(ctx context.Context) ([]UnstakeRequest, error) {
	resp := struct {
		Data []UnstakeRequest `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, unstakeRequests, nil, &resp)
}

// GetStakeBalances returns a collection of staked coin balances
func (f *FTX) GetStakeBalances(ctx context.Context) ([]StakeBalance, error) {
	resp := struct {
		Data []StakeBalance `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, stakeBalances, nil, &resp)
}

// UnstakeRequest unstakes an existing staked coin
func (f *FTX) UnstakeRequest(ctx context.Context, coin currency.Code, size float64) (*UnstakeRequest, error) {
	resp := struct {
		Data UnstakeRequest `json:"result"`
	}{}
	req := make(map[string]interface{})
	req["coin"] = coin.Upper().String()
	req["size"] = strconv.FormatFloat(size, 'f', -1, 64)
	return &resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, unstakeRequests, req, &resp)
}

// CancelUnstakeRequest cancels a pending unstake request
func (f *FTX) CancelUnstakeRequest(ctx context.Context, requestID int64) (bool, error) {
	resp := struct {
		Result string
	}{}
	path := unstakeRequests + "/" + strconv.FormatInt(requestID, 10)
	if err := f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, &resp); err != nil {
		return false, err
	}

	if resp.Result != "Cancelled" {
		return false, errors.New("failed to cancel unstake request")
	}
	return true, nil
}

// GetStakingRewards returns a collection of staking rewards
func (f *FTX) GetStakingRewards(ctx context.Context) ([]StakeReward, error) {
	resp := struct {
		Data []StakeReward `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, stakingRewards, nil, &resp)
}

// StakeRequest submits a stake request based on the specified currency and size
func (f *FTX) StakeRequest(ctx context.Context, coin currency.Code, size float64) (*Stake, error) {
	resp := struct {
		Data Stake `json:"result"`
	}{}
	req := make(map[string]interface{})
	req["coin"] = coin.Upper().String()
	req["size"] = strconv.FormatFloat(size, 'f', -1, 64)
	return &resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, serumStakes, req, &resp)
}

// SendAuthHTTPRequest sends an authenticated request
func (f *FTX) SendAuthHTTPRequest(ctx context.Context, ep exchange.URL, method, path string, data, result interface{}) error {
	creds, err := f.GetCredentials(ctx)
	if err != nil {
		return err
	}

	endpoint, err := f.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	newRequest := func() (*request.Item, error) {
		ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
		var body io.Reader
		var hmac, payload []byte

		sigPayload := ts + method + "/api" + path
		if data != nil {
			payload, err = json.Marshal(data)
			if err != nil {
				return nil, err
			}
			body = bytes.NewBuffer(payload)
			sigPayload += string(payload)
		}

		hmac, err = crypto.GetHMAC(crypto.HashSHA256,
			[]byte(sigPayload),
			[]byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		headers := make(map[string]string)
		headers["FTX-KEY"] = creds.Key
		headers["FTX-SIGN"] = crypto.HexEncodeToString(hmac)
		headers["FTX-TS"] = ts
		if creds.SubAccount != "" {
			headers["FTX-SUBACCOUNT"] = url.QueryEscape(creds.SubAccount)
		}
		headers["Content-Type"] = "application/json"

		return &request.Item{
			Method:        method,
			Path:          endpoint + path,
			Headers:       headers,
			Body:          body,
			Result:        result,
			AuthRequest:   true,
			Verbose:       f.Verbose,
			HTTPDebugging: f.HTTPDebugging,
			HTTPRecording: f.HTTPRecording,
		}, nil
	}
	return f.SendPayload(ctx, request.Unset, newRequest)
}

// GetFee returns an estimate of fee based on type of transaction
func (f *FTX) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	if !f.GetAuthenticatedAPISupport(exchange.RestAuthentication) {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	switch feeBuilder.FeeType {
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder)
	default:
		feeData, err := f.GetAccountInfo(ctx)
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

func (f *FTX) compatibleOrderVars(ctx context.Context, orderSide, orderStatus, orderType string, amount, filledAmount, avgFillPrice float64) (OrderVars, error) {
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
		switch {
		case filledAmount <= 0:
			// Order is closed with a filled amount of 0, which means it's
			// cancelled.
			resp.Status = order.Cancelled
		case math.Abs(filledAmount-amount) > 1e-6:
			// Order is closed with filledAmount above 0, but not equal to the
			// full amount, which means it's partially executed and then
			// cancelled.
			resp.Status = order.PartiallyCancelled
		default:
			// Order is closed and filledAmount == amount, which means it's
			// fully executed.
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
	fee, err := f.GetFee(ctx, &feeBuilder)
	if err != nil {
		return resp, err
	}
	resp.Fee = fee
	return resp, nil
}

// RequestForQuotes requests for otc quotes
func (f *FTX) RequestForQuotes(ctx context.Context, base, quote currency.Code, amount float64) (RequestQuoteData, error) {
	resp := struct {
		Data RequestQuoteData `json:"result"`
	}{}
	req := make(map[string]interface{})
	req["fromCoin"] = base.Upper().String()
	req["toCoin"] = quote.Upper().String()
	req["size"] = amount
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, requestOTCQuote, req, &resp)
}

// GetOTCQuoteStatus gets quote status of a quote
func (f *FTX) GetOTCQuoteStatus(ctx context.Context, marketName, quoteID string) (*QuoteStatusData, error) {
	resp := struct {
		Data QuoteStatusData `json:"result"`
	}{}
	params := url.Values{}
	params.Set("market", marketName)
	return &resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, getOTCQuoteStatus+quoteID, params, &resp)
}

// AcceptOTCQuote requests for otc quotes
func (f *FTX) AcceptOTCQuote(ctx context.Context, quoteID string) error {
	return f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, fmt.Sprintf(acceptOTCQuote, quoteID), nil, nil)
}

// GetSubaccounts returns the users subaccounts
func (f *FTX) GetSubaccounts(ctx context.Context) ([]Subaccount, error) {
	resp := struct {
		Data []Subaccount `json:"result"`
	}{}
	return resp.Data, f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, subaccounts, nil, &resp)
}

// CreateSubaccount creates a new subaccount
func (f *FTX) CreateSubaccount(ctx context.Context, name string) (*Subaccount, error) {
	if name == "" {
		return nil, errSubaccountNameMustBeSpecified
	}
	d := make(map[string]string)
	d["nickname"] = name

	resp := struct {
		Data Subaccount `json:"result"`
	}{}
	if err := f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, subaccounts, d, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// UpdateSubaccountName updates an existing subaccount name
func (f *FTX) UpdateSubaccountName(ctx context.Context, oldName, newName string) (*Subaccount, error) {
	if oldName == "" || newName == "" || oldName == newName {
		return nil, errSubaccountUpdateNameInvalid
	}
	d := make(map[string]string)
	d["nickname"] = oldName
	d["newNickname"] = newName

	resp := struct {
		Data Subaccount `json:"result"`
	}{}
	if err := f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, subaccountsUpdateName, d, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// DeleteSubaccount deletes the specified subaccount name
func (f *FTX) DeleteSubaccount(ctx context.Context, name string) error {
	if name == "" {
		return errSubaccountNameMustBeSpecified
	}
	d := make(map[string]string)
	d["nickname"] = name
	resp := struct {
		Data Subaccount `json:"result"`
	}{}
	return f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, subaccounts, d, &resp)
}

// SubaccountBalances returns the user's subaccount balances
func (f *FTX) SubaccountBalances(ctx context.Context, name string) ([]SubaccountBalance, error) {
	if name == "" {
		return nil, errSubaccountNameMustBeSpecified
	}
	resp := struct {
		Data []SubaccountBalance `json:"result"`
	}{}
	if err := f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, fmt.Sprintf(subaccountsBalance, name), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// SubaccountTransfer transfers a desired coin to the specified subaccount
func (f *FTX) SubaccountTransfer(ctx context.Context, coin currency.Code, source, destination string, size float64) (*SubaccountTransferStatus, error) {
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
	if err := f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, subaccountsTransfer, d, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// FetchExchangeLimits fetches spot order execution limits
func (f *FTX) FetchExchangeLimits(ctx context.Context) ([]order.MinMaxLevel, error) {
	data, err := f.GetMarkets(ctx)
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

// GetCollateral returns total collateral and the breakdown of
// collateral contributions
func (f *FTX) GetCollateral(ctx context.Context, maintenance bool) (*CollateralResponse, error) {
	resp := struct {
		Data CollateralResponse `json:"result"`
	}{}
	u := url.Values{}
	if maintenance {
		u.Add("marginType", "maintenance")
	} else {
		u.Add("marginType", "initial")
	}
	url := common.EncodeURLValues(collateral, u)
	if err := f.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, url, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// LoadCollateralWeightings sets the collateral weights for
// currencies supported by FTX
func (f *FTX) LoadCollateralWeightings(ctx context.Context) error {
	f.collateralWeight = make(map[*currency.Item]CollateralWeight)
	// taken from https://help.ftx.com/hc/en-us/articles/360031149632-Non-USD-Collateral
	// sets default, then uses the latest from FTX
	f.collateralWeight.load("1INCH", 0.9, 0.85, 0.0005)
	f.collateralWeight.load("AAPL", 0.9, 0.85, 0.005)
	f.collateralWeight.load("AAVE", 0.9, 0.85, 0.0025)
	f.collateralWeight.load("ABNB", 0.9, 0.85, 0.005)
	f.collateralWeight.load("ACB", 0.9, 0.85, 0.0025)
	f.collateralWeight.load("ALPHA", 0.9, 0.85, 0.00025)
	f.collateralWeight.load("AMC", 0.9, 0.85, 0.0025)
	f.collateralWeight.load("AMD", 0.9, 0.85, 0.005)
	f.collateralWeight.load("AMZN", 0.9, 0.85, 0.03)
	f.collateralWeight.load("APHA", 0.9, 0.85, 0.001)
	f.collateralWeight.load("ARKK", 0.9, 0.85, 0.005)
	f.collateralWeight.load("AUD", 0.99, 0.98, 0.00001)
	f.collateralWeight.load("BABA", 0.9, 0.85, 0.01)
	f.collateralWeight.load("BADGER", 0.85, 0.8, 0.0025)
	f.collateralWeight.load("BAND", 0.85, 0.8, 0.001)
	f.collateralWeight.load("BAO", 0.85, 0.8, 0.000025)
	f.collateralWeight.load("BB", 0.9, 0.85, 0.0025)
	f.collateralWeight.load("BCH", 0.95, 0.9, 0.0008)
	f.collateralWeight.load("BILI", 0.9, 0.85, 0.005)
	f.collateralWeight.load("BITW", 0.9, 0.85, 0.005)
	f.collateralWeight.load("BNB", 0.95, 0.9, 0.0005)
	f.collateralWeight.load("BNT", 0.9, 0.85, 0.0025)
	f.collateralWeight.load("BNTX", 0.9, 0.85, 0.005)
	f.collateralWeight.load("BRL", 0.99, 0.98, 0.00001)
	f.collateralWeight.load("BRZ", 0.99, 0.98, 0.00001)
	f.collateralWeight.load("BTC", 0.975, 0.95, 0.002)
	f.collateralWeight.load("BTMX", 0.7, 0.65, 0.0008)
	f.collateralWeight.load("BUSD", 1, 1, 0)
	f.collateralWeight.load("BVOL", 0.85, 0.8, 0.005)
	f.collateralWeight.load("BYND", 0.9, 0.85, 0.0075)
	f.collateralWeight.load("CAD", 0.99, 0.98, 0.00001)
	f.collateralWeight.load("CEL", 0.85, 0.8, 0.001)
	f.collateralWeight.load("CGC", 0.9, 0.85, 0.0025)
	f.collateralWeight.load("CHF", 0.99, 0.98, 0.00001)
	f.collateralWeight.load("COIN", 0.85, 0.8, 0.01)
	f.collateralWeight.load("COMP", 0.9, 0.85, 0.002)
	f.collateralWeight.load("COPE", 0.6, 0.55, 0.02)
	f.collateralWeight.load("CRON", 0.9, 0.85, 0.001)
	f.collateralWeight.load("CUSDT", 0.9, 0.85, 0.00001)
	f.collateralWeight.load("DAI", 0.9, 0.85, 0.00005)
	f.collateralWeight.load("DOGE", 0.95, 0.9, 0.00002)
	f.collateralWeight.load("ETH", 0.95, 0.9, 0.0004)
	f.collateralWeight.load("STETH", 0.9, 0.85, 0.0012)
	f.collateralWeight.load("ETHE", 0.9, 0.85, 0.0025)
	f.collateralWeight.load("EUR", 0.99, 0.98, 0.00001)
	f.collateralWeight.load("FB", 0.9, 0.85, 0.01)
	f.collateralWeight.load("FIDA", 0.85, 0.8, 0.001)
	f.collateralWeight.load("FTM", 0.85, 0.8, 0.0005)
	f.collateralWeight.load("FTT", 0.95, 0.95, 0.0005)
	f.collateralWeight.load("GBP", 0.99, 0.98, 0.00001)
	f.collateralWeight.load("GBTC", 0.9, 0.85, 0.0025)
	f.collateralWeight.load("GDX", 0.9, 0.85, 0.0025)
	f.collateralWeight.load("GDXJ", 0.9, 0.85, 0.005)
	f.collateralWeight.load("GLD", 0.9, 0.85, 0.005)
	f.collateralWeight.load("GLXY", 0.9, 0.85, 0.005)
	f.collateralWeight.load("GME", 0.9, 0.85, 0.005)
	f.collateralWeight.load("GOOGL", 0.9, 0.85, 0.025)
	f.collateralWeight.load("GRT", 0.9, 0.85, 0.00025)
	f.collateralWeight.load("HKD", 0.99, 0.98, 0.00001)
	f.collateralWeight.load("HOLY", 0.9, 0.85, 0.0005)
	f.collateralWeight.load("HOOD", 0.85, 0.8, 0.005)
	f.collateralWeight.load("HT", 0.9, 0.85, 0.0003)
	f.collateralWeight.load("HUSD", 1, 1, 0)
	f.collateralWeight.load("HXRO", 0.85, 0.8, 0.001)
	f.collateralWeight.load("IBVOL", 0.85, 0.8, 0.015)
	f.collateralWeight.load("KIN", 0.85, 0.8, 0.000008)
	f.collateralWeight.load("KNC", 0.95, 0.9, 0.001)
	f.collateralWeight.load("LEO", 0.85, 0.8, 0.001)
	f.collateralWeight.load("LINK", 0.95, 0.9, 0.0004)
	f.collateralWeight.load("LRC", 0.85, 0.8, 0.0005)
	f.collateralWeight.load("LTC", 0.95, 0.9, 0.0004)
	f.collateralWeight.load("MATIC", 0.85, 0.8, 0.00004)
	f.collateralWeight.load("MKR", 0.9, 0.85, 0.007)
	f.collateralWeight.load("MOB", 0.6, 0.55, 0.005)
	f.collateralWeight.load("MRNA", 0.9, 0.85, 0.005)
	f.collateralWeight.load("MSTR", 0.9, 0.85, 0.008)
	f.collateralWeight.load("NFLX", 0.9, 0.85, 0.01)
	f.collateralWeight.load("NIO", 0.9, 0.85, 0.004)
	f.collateralWeight.load("NOK", 0.9, 0.85, 0.001)
	f.collateralWeight.load("NVDA", 0.9, 0.85, 0.01)
	f.collateralWeight.load("OKB", 0.9, 0.85, 0.0003)
	f.collateralWeight.load("OMG", 0.85, 0.8, 0.001)
	f.collateralWeight.load("USDP", 1, 1, 0)
	f.collateralWeight.load("PAXG", 0.95, 0.9, 0.002)
	f.collateralWeight.load("PENN", 0.9, 0.85, 0.005)
	f.collateralWeight.load("PFE", 0.9, 0.85, 0.004)
	f.collateralWeight.load("PYPL", 0.9, 0.85, 0.008)
	f.collateralWeight.load("RAY", 0.85, 0.8, 0.0005)
	f.collateralWeight.load("REN", 0.9, 0.85, 0.00025)
	f.collateralWeight.load("RSR", 0.85, 0.8, 0.0001)
	f.collateralWeight.load("RUNE", 0.85, 0.8, 0.001)
	f.collateralWeight.load("SECO", 0.9, 0.85, 0.0005)
	f.collateralWeight.load("SGD", 0.99, 0.98, 0.00001)
	f.collateralWeight.load("SLV", 0.9, 0.85, 0.0025)
	f.collateralWeight.load("SNX", 0.85, 0.8, 0.001)
	f.collateralWeight.load("SOL", 0.9, 0.85, 0.0004)
	f.collateralWeight.load("STSOL", 0.9, 0.85, 0.0004)
	f.collateralWeight.load("MSOL", 0.9, 0.85, 0.0004)
	f.collateralWeight.load("SPY", 0.9, 0.85, 0.01)
	f.collateralWeight.load("SQ", 0.9, 0.85, 0.008)
	f.collateralWeight.load("SRM", 0.9, 0.85, 0.0005)
	f.collateralWeight.load("SUSHI", 0.95, 0.9, 0.001)
	f.collateralWeight.load("SXP", 0.9, 0.85, 0.0005)
	f.collateralWeight.load("TLRY", 0.9, 0.85, 0.001)
	f.collateralWeight.load("TOMO", 0.85, 0.8, 0.0005)
	f.collateralWeight.load("TRX", 0.9, 0.85, 0.00001)
	f.collateralWeight.load("TRY", 0.99, 0.98, 0.00001)
	f.collateralWeight.load("TRYB", 0.9, 0.85, 0.00001)
	f.collateralWeight.load("TSLA", 0.9, 0.85, 0.01)
	f.collateralWeight.load("TSM", 0.9, 0.85, 0.015)
	f.collateralWeight.load("TUSD", 1, 1, 0)
	f.collateralWeight.load("TWTR", 0.9, 0.85, 0.004)
	f.collateralWeight.load("UBER", 0.9, 0.85, 0.004)
	f.collateralWeight.load("UNI", 0.95, 0.9, 0.001)
	f.collateralWeight.load("USD", 1, 1, 0)
	f.collateralWeight.load("USDC", 1, 1, 0)
	f.collateralWeight.load("USDT", 0.975, 0.95, 0.00001)
	f.collateralWeight.load("USO", 0.9, 0.85, 0.0025)
	f.collateralWeight.load("WBTC", 0.975, 0.95, 0.005)
	f.collateralWeight.load("WUSDC", 1, 1, 0)
	f.collateralWeight.load("WUSDT", 0.975, 0.95, 0.00001)
	f.collateralWeight.load("XAUT", 0.95, 0.9, 0.002)
	f.collateralWeight.load("XRP", 0.95, 0.9, 0.00002)
	f.collateralWeight.load("YFI", 0.9, 0.85, 0.015)
	f.collateralWeight.load("ZAR", 0.99, 0.98, 0.00001)
	f.collateralWeight.load("ZM", 0.9, 0.85, 0.01)
	f.collateralWeight.load("ZRX", 0.85, 0.8, 0.001)

	if !f.GetAuthenticatedAPISupport(exchange.RestAuthentication) {
		return nil
	}
	coins, err := f.GetCoins(ctx)
	if err != nil {
		return err
	}
	for i := range coins {
		if !coins[i].Collateral {
			continue
		}
		f.collateralWeight.loadTotal(coins[i].ID, coins[i].CollateralWeight)
	}

	futures, err := f.GetFutures(ctx)
	if err != nil {
		return err
	}
	for i := range futures {
		f.collateralWeight.loadInitialMarginFraction(futures[i].Underlying, futures[i].InitialMarginFractionFactor)
	}

	return nil
}

func (c CollateralWeightHolder) hasData() bool {
	return len(c) > 0
}

func (c CollateralWeightHolder) loadTotal(code string, weighting float64) {
	cc := currency.NewCode(code)
	currencyCollateral := c[cc.Item]
	currencyCollateral.Total = weighting
	c[cc.Item] = currencyCollateral
}

func (c CollateralWeightHolder) loadInitialMarginFraction(code string, imf float64) {
	cc := currency.NewCode(code)
	currencyCollateral := c[cc.Item]
	currencyCollateral.InitialMarginFractionFactor = imf
	c[cc.Item] = currencyCollateral
}

func (c CollateralWeightHolder) load(code string, total, initial, imfFactor float64) {
	cc := currency.NewCode(code)
	c[cc.Item] = CollateralWeight{
		Total:                       total,
		Initial:                     initial,
		InitialMarginFractionFactor: imfFactor,
	}
}
