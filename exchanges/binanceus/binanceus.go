package binanceus

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Binanceus
type Exchange struct {
	exchange.Base
	obm *orderbookManager
}

const (
	tradeBaseURL = "https://www.binance.us/spot-trade/"
	// General Data Endpoints
	serverTime   = "/api/v3/time"
	systemStatus = "/sapi/v1/system/status"

	// Public endpoints
	exchangeInfo     = "/api/v3/exchangeInfo"
	recentTrades     = "/api/v3/trades"
	aggregatedTrades = "/api/v3/aggTrades"
	orderBookDepth   = "/api/v3/depth"
	candleStick      = "/api/v3/klines"
	tickerPrice      = "/api/v3/ticker/price"
	averagePrice     = "/api/v3/avgPrice"
	bestPrice        = "/api/v3/ticker/bookTicker"
	priceChange      = "/api/v3/ticker/24hr"
	historicalTrades = "/api/v3/historicalTrades"

	// Withdraw API endpoints
	tradingStatus = "/sapi/v3/apiTradingStatus"
	tradeFee      = "/wapi/v3/tradeFee.html"

	// Subaccounts
	subaccountsInformation    = "/sapi/v3/sub-account/list"
	subaccountTransferHistory = "/sapi/v3/sub-account/transfer/history"
	subaccountTransfer        = "/sapi/v3/sub-account/transfer"
	subaccountAssets          = "/sapi/v3/sub-account/assets"

	// Account Endpoint
	accountInfo                            = "/api/v3/account"
	accountStatus                          = "/sapi/v3/accountStatus"
	accountEnableCryptoWithdrawalEndpoint  = "/sapi/v1/account/quickEnableWithdrawal"
	accountDisableCryptoWithdrawalEndpoint = "/sapi/v1/account/quickDisableWithdrawal"
	masterAccounts                         = "/sapi/v1/sub-account/spotSummary"
	subAccountStatusList                   = "/sapi/v1/sub-account/status"
	usersSpotAssetsSnapshot                = "/sapi/v1/accountSnapshot"

	// Trade Order Endpoints
	orderRateLimit     = "/api/v3/rateLimit/order"
	testCreateNeworder = "/api/v3/order/test" // Method: POST
	orderRequest       = "/api/v3/order"      // Used in Create {Method: POST}, Cancel {DELETE}, and get{GET} OrderRequest
	openOrders         = "/api/v3/openOrders"
	myTrades           = "/api/v3/myTrades"

	// One-Cancels-the-Other Orders (OCO Orders)
	ocoOrder        = "/api/v3/order/oco"
	ocoOrderList    = "/api/v3/orderList"
	ocoAllOrderList = "/api/v3/allOrderList"
	ocoOpenOrders   = "/api/v3/openOrderList"

	// OTC Endpoints
	// Over-The-Counter Endpoints
	otcSelectors    = "/sapi/v1/otc/coinPairs"
	otcQuotes       = "/sapi/v1/otc/quotes"
	otcTradeOrder   = "/sapi/v1/otc/orders"
	otcTradeOrders  = "/sapi/v1/otc/orders/"
	ocbsTradeOrders = "/sapi/v1/ocbs/orders"

	// Wallet endpoints
	assetDistributionHistory = "/sapi/v1/asset/assetDistributionHistory"
	assetFeeAndWalletStatus  = "/sapi/v1/capital/config/getall"
	applyWithdrawal          = "/sapi/v1/capital/withdraw/apply"
	withdrawalHistory        = "/sapi/v1/capital/withdraw/history"
	withdrawFiat             = "/sapi/v1/fiatpayment/apply/withdraw"
	fiatWithdrawalHistory    = "/sapi/v1/fiatpayment/query/withdraw/history"
	fiatDepositHistory       = "/sapi/v1/fiatpayment/query/deposit/history"
	depositAddress           = "/sapi/v1/capital/deposit/address"
	depositHistory           = "/sapi/v1/capital/deposit/hisrec"
	subAccountDepositAddress = "/sapi/v1/capital/sub-account/deposit/address"
	subAccountDepositHistory = "/sapi/v1/capital/sub-account/deposit/history"

	// Referral Reward Endpoints
	referralRewardHistory = "/sapi/v1/marketing/referral/reward/history"

	// Web socket related route
	userAccountStream = "/api/v3/userDataStream"

	// Other Consts
	defaultRecvWindow = 5 * time.Second

	// recvWindowSize5000
	recvWindowSize5000 = 5000
)

const recvWindowSize5000String = "5000"

// This is a list of error Messages to be returned by binanceus endpoint methods.
var (
	errNotValidEmailAddress                   = errors.New("invalid email address")
	errUnacceptableSenderEmail                = errors.New("senders address email is missing")
	errUnacceptableReceiverEmail              = errors.New("receiver address email is missing")
	errInvalidAssetValue                      = errors.New("invalid asset ")
	errInvalidAssetAmount                     = errors.New("invalid asset amount")
	errIncompleteArguments                    = errors.New("missing required argument")
	errMissingRequiredArgumentCoin            = errors.New("missing required argument,coin")
	errMissingRequiredArgumentNetwork         = errors.New("missing required argument,network")
	errAmountValueMustBeGreaterThan0          = errors.New("amount must be greater than 0")
	errMissingPaymentAccountInfo              = errors.New("error: missing payment account")
	errMissingRequiredParameterAddress        = errors.New("missing required parameter \"address\"")
	errMissingCurrencySymbol                  = errors.New("missing currency symbol")
	errEitherOrderIDOrClientOrderIDIsRequired = errors.New("either order id or client order id is required")
	errMissingRequestAmount                   = errors.New("missing required value \"requestAmount\"")
	errMissingRequestCoin                     = errors.New("missing required value \"requestCoin\" name")
	errMissingToCoinName                      = errors.New("missing required value \"toCoin\" name")
	errMissingFromCoinName                    = errors.New("missing required value \"fromCoin\" name")
	errMissingQuoteID                         = errors.New("missing quote id")
	errMissingSubAccountEmail                 = errors.New("missing sub-account email address")
	errMissingCurrencyCoin                    = errors.New("missing currency coin")
	errInvalidUserBusinessType                = errors.New("only 0: referrer and 1: referee are allowed")
	errMissingPageNumber                      = errors.New("missing page number")
	errInvalidRowNumber                       = errors.New("invalid row number")
)

// General Data Endpoints

// GetServerTime this endpoint returns the exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	var response ServerTime
	err := e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, serverTime, spotDefaultRate, &response)
	return response.Timestamp.Time(), err
}

// GetSystemStatus endpoint to fetch whether the system status is normal or under maintenance.
func (e *Exchange) GetSystemStatus(ctx context.Context) (int, error) {
	resp := struct {
		Status int `json:"status"`
	}{}
	return resp.Status, e.SendAuthHTTPRequest(
		ctx, exchange.RestSpotSupplementary,
		http.MethodGet, systemStatus,
		nil, spotDefaultRate, &resp)
}

// GetExchangeInfo to get the current exchange trading rules and trading pair information.
func (e *Exchange) GetExchangeInfo(ctx context.Context) (ExchangeInfo, error) {
	var respo ExchangeInfo
	return respo, e.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		exchangeInfo, spotExchangeInfo, &respo)
}

// GetMostRecentTrades to get older trades. maximum limit in the RecentTradeRequestParams is 1,000 trades.
func (e *Exchange) GetMostRecentTrades(ctx context.Context, rtr RecentTradeRequestParams) ([]RecentTrade, error) {
	params := url.Values{}
	symbol, err := e.FormatSymbol(rtr.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.FormatInt(rtr.Limit, 10))
	path := common.EncodeURLValues(recentTrades, params)
	var resp []RecentTrade
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// GetHistoricalTrades returns historical trade activity
// symbol: string of currency pair
// limit: Optional. Default 500; max 1000.
func (e *Exchange) GetHistoricalTrades(ctx context.Context, hist HistoricalTradeParams) ([]HistoricalTrade, error) {
	var resp []HistoricalTrade
	params := url.Values{}
	params.Set("symbol", hist.Symbol)
	params.Set("limit", strconv.FormatInt(hist.Limit, 10))
	if hist.FromID > 0 {
		params.Set("fromId", strconv.FormatUint(hist.FromID, 10))
	}
	path := common.EncodeURLValues(historicalTrades, params)
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotHistoricalTradesRate, &resp)
}

// GetAggregateTrades to get compressed, aggregate trades. Trades that fill at the time, from the same order, with the same price will have the quantity aggregated.
func (e *Exchange) GetAggregateTrades(ctx context.Context, agg *AggregatedTradeRequestParams) ([]AggregatedTrade, error) {
	params := url.Values{}
	s, err := e.FormatSymbol(agg.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", s)
	// If the user request is directly not supported by the exchange, we might be able to fulfill it
	// by merging results from multiple API requests
	needBatch := true // Need to batch unless user has specified a limit
	if agg.Limit > 0 && agg.Limit <= 1000 {
		needBatch = false
		params.Set("limit", strconv.Itoa(agg.Limit))
	}
	if agg.FromID != 0 {
		params.Set("fromId", strconv.FormatInt(agg.FromID, 10))
	}
	if !agg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(agg.StartTime.UnixMilli(), 10))
	}
	if !agg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(agg.EndTime.UnixMilli(), 10))
	}

	// startTime and endTime are set and time between startTime and endTime is more than 1 hour
	needBatch = needBatch || (!agg.StartTime.IsZero() && !agg.EndTime.IsZero() && agg.EndTime.Sub(agg.StartTime) > time.Hour)
	// Fall back to batch requests, if possible and necessary
	if needBatch {
		// fromId or start time must be set
		canBatch := agg.FromID == 0 != agg.StartTime.IsZero()
		if canBatch {
			// Split the request into multiple
			return e.batchAggregateTrades(ctx, agg, params)
		}

		// Can't handle this request locally or remotely
		// We would receive {"code":-1128,"msg":"Combination of optional parameters invalid."}
		return nil, errors.New("please set StartTime or FromId, but not both")
	}
	var resp []AggregatedTrade
	path := aggregatedTrades + "?" + params.Encode()
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// batchAggregateTrades fetches trades in multiple requests   <-- copied and amended from the  binance
// first phase, hourly requests until the first trade (or end time) is reached
// second phase, limit requests from previous trade until end time (or limit) is reached
func (e *Exchange) batchAggregateTrades(ctx context.Context, args *AggregatedTradeRequestParams, params url.Values) ([]AggregatedTrade, error) {
	var resp []AggregatedTrade
	// prepare first request with only first hour and max limit
	if args.Limit == 0 || args.Limit > 1000 {
		// Extend from the default of 500
		params.Set("limit", "1000")
	}

	var fromID int64
	if args.FromID > 0 {
		fromID = args.FromID
	} else {
		// Only 10 seconds is used to prevent limit of 1000 being reached in the first request, cutting off trades for high activity pairs
		// If we don't find anything we keep increasing the window by doubling the interval and scanning again
		increment := time.Second * 10
		for start := args.StartTime; len(resp) == 0; start, increment = start.Add(increment), increment*2 {
			if !args.EndTime.IsZero() && start.After(args.EndTime) || increment <= 0 {
				// All requests returned empty or we searched until increment overflowed
				return nil, nil
			}
			params.Set("startTime", strconv.FormatInt(start.UnixMilli(), 10))
			end := start.Add(increment)
			if !args.EndTime.IsZero() && end.After(args.EndTime) {
				end = args.EndTime
			}
			params.Set("endTime", strconv.FormatInt(end.UnixMilli(), 10))
			path := aggregatedTrades + "?" + params.Encode()
			err := e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
			if err != nil {
				return resp, fmt.Errorf("%w %v", err, args.Symbol)
			}
		}
		fromID = resp[len(resp)-1].ATradeID
	}

	// other requests follow from the last aggregate trade id and have no time window
	params.Del("startTime")
	params.Del("endTime")
outer:
	for ; args.Limit == 0 || len(resp) < args.Limit; fromID = resp[len(resp)-1].ATradeID {
		// Keep requesting new data after last retrieved trade
		params.Set("fromId", strconv.FormatInt(fromID, 10))
		path := aggregatedTrades + "?" + params.Encode()
		var additionalTrades []AggregatedTrade
		if err := e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &additionalTrades); err != nil {
			return resp, fmt.Errorf("%w %v", err, args.Symbol)
		}
		switch len(additionalTrades) {
		case 0, 1:
			break outer // We only got the one we already have
		default:
			additionalTrades = additionalTrades[1:] // Remove the record we already have
		}
		if !args.EndTime.IsZero() {
			// Check if only some of the results are before EndTime
			if afterEnd := sort.Search(len(additionalTrades), func(i int) bool {
				return args.EndTime.Before(additionalTrades[i].TimeStamp.Time())
			}); afterEnd < len(additionalTrades) {
				resp = append(resp, additionalTrades[:afterEnd]...)
				break
			}
		}
		resp = append(resp, additionalTrades...)
	}
	if args.Limit > 0 && len(resp) > args.Limit {
		resp = resp[:args.Limit]
	}
	return resp, nil
}

// GetOrderBookDepth to get the order book depth. Please note the limits in the table below.
func (e *Exchange) GetOrderBookDepth(ctx context.Context, pair currency.Pair, limit uint64) (*OrderBook, error) {
	params := url.Values{}
	symbol, err := e.FormatSymbol(pair, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.FormatUint(limit, 10))

	var resp *OrderBookData
	if err := e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues(orderBookDepth, params), orderbookLimit(limit), &resp); err != nil {
		return nil, err
	}

	return &OrderBook{
		Bids:         resp.Bids.Levels(),
		Asks:         resp.Asks.Levels(),
		LastUpdateID: resp.LastUpdateID,
	}, nil
}

// GetIntervalEnum allowed interval params by Binanceus
func (e *Exchange) GetIntervalEnum(interval kline.Interval) string {
	switch interval {
	case kline.OneMin:
		return "1m"
	case kline.ThreeMin:
		return "3m"
	case kline.FiveMin:
		return "5m"
	case kline.FifteenMin:
		return "15m"
	case kline.ThirtyMin:
		return "30m"
	case kline.OneHour:
		return "1h"
	case kline.TwoHour:
		return "2h"
	case kline.FourHour:
		return "4h"
	case kline.SixHour:
		return "6h"
	case kline.EightHour:
		return "8h"
	case kline.TwelveHour:
		return "12h"
	case kline.OneDay:
		return "1d"
	case kline.ThreeDay:
		return "3d"
	case kline.OneWeek:
		return "1w"
	case kline.OneMonth:
		return "1M"
	default:
		return "notfound"
	}
}

// GetSpotKline to get Kline/candlestick bars for a token symbol. Klines are uniquely identified by their open time.
func (e *Exchange) GetSpotKline(ctx context.Context, arg *KlinesRequestParams) ([]CandleStick, error) {
	symbol, err := e.FormatSymbol(arg.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", arg.Interval)
	if arg.Limit != 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if !arg.StartTime.IsZero() && arg.StartTime.Unix() != 0 {
		params.Set("startTime", strconv.FormatInt((arg.StartTime).UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() && arg.EndTime.Unix() != 0 {
		params.Set("endTime", strconv.FormatInt((arg.EndTime).UnixMilli(), 10))
	}
	path := common.EncodeURLValues(candleStick, params)
	var resp []CandleStick
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// GetSinglePriceData to get the latest price for a token symbol or symbols.
func (e *Exchange) GetSinglePriceData(ctx context.Context, symbol currency.Pair) (SymbolPrice, error) {
	var res SymbolPrice
	params := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return res, err
	}
	params.Set("symbol", symbolValue)
	path := common.EncodeURLValues(tickerPrice, params)
	return res, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &res)
}

// GetPriceDatas to get the latest price for symbols.
func (e *Exchange) GetPriceDatas(ctx context.Context) (SymbolPrices, error) {
	var res SymbolPrices
	return res, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, tickerPrice, spotSymbolPriceAllRate, &res)
}

// GetAveragePrice returns current average price for a symbol.
//
// symbol: string of currency pair
func (e *Exchange) GetAveragePrice(ctx context.Context, symbol currency.Pair) (AveragePrice, error) {
	resp := AveragePrice{}
	params := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)

	path := common.EncodeURLValues(averagePrice, params)

	return resp, e.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// GetBestPrice returns the latest best price for symbol
// symbol: string of currency pair
func (e *Exchange) GetBestPrice(ctx context.Context, symbol currency.Pair) (BestPrice, error) {
	resp := BestPrice{}
	params := url.Values{}
	rateLimit := spotOrderbookTickerAllRate
	if !symbol.IsEmpty() {
		rateLimit = spotDefaultRate
		symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	path := common.EncodeURLValues(bestPrice, params)

	return resp,
		e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, rateLimit, &resp)
}

// GetPriceChangeStats returns price change statistics for the last 24 hours
// symbol: string of currency pair
func (e *Exchange) GetPriceChangeStats(ctx context.Context, symbol currency.Pair) (PriceChangeStats, error) {
	resp := PriceChangeStats{}
	params := url.Values{}
	rateLimit := spotPriceChangeAllRate
	if !symbol.IsEmpty() {
		rateLimit = spotDefaultRate
		symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	path := common.EncodeURLValues(priceChange, params)

	return resp, e.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, path, rateLimit, &resp)
}

// GetTickers returns the ticker data for the last 24 hrs
func (e *Exchange) GetTickers(ctx context.Context) ([]PriceChangeStats, error) {
	var resp []PriceChangeStats
	return resp, e.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, priceChange, spotPriceChangeAllRate, &resp)
}

// GetAccount returns binance user accounts
func (e *Exchange) GetAccount(ctx context.Context) (*Account, error) {
	type response struct {
		Response
		Account
	}
	var resp response
	params := url.Values{}
	if err := e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, accountInfo,
		params, spotAccountInformationRate,
		&resp); err != nil {
		return &resp.Account, err
	}

	if resp.Code != 0 {
		return &resp.Account, errors.New(resp.Msg)
	}

	return &resp.Account, nil
}

// GetUserAccountStatus  to fetch account status detail.
func (e *Exchange) GetUserAccountStatus(ctx context.Context, recvWindow uint64) (*AccountStatusResponse, error) {
	var resp AccountStatusResponse
	params := url.Values{}
	timestamp := time.Now().UnixMilli()
	params.Set("timestamp", strconv.FormatInt(timestamp, 10))
	if recvWindow > 0 && recvWindow < 60000 {
		if recvWindow < 2000 {
			recvWindow += 1500
		}
		params.Set("recvWindow", strconv.FormatUint(recvWindow, 10))
	}

	return &resp,
		e.SendAuthHTTPRequest(ctx,
			exchange.RestSpotSupplementary,
			http.MethodGet,
			accountStatus,
			params,
			spotDefaultRate,
			&resp)
}

// GetUserAPITradingStatus to fetch account API trading status details.
func (e *Exchange) GetUserAPITradingStatus(ctx context.Context, recvWindow uint64) (*TradeStatus, error) {
	type response struct {
		Success bool        `json:"success"`
		TC      TradeStatus `json:"status"`
	}
	var resp response
	params := url.Values{}
	timestamp := time.Now().UnixMilli()
	params.Set("timestamp", strconv.FormatInt(timestamp, 10))
	if recvWindow > 0 && recvWindow < 2000 {
		recvWindow += 1500
	}
	params.Set("recvWindow", strconv.FormatUint(recvWindow, 10))
	return &resp.TC,
		e.SendAuthHTTPRequest(ctx,
			exchange.RestSpotSupplementary,
			http.MethodGet,
			tradingStatus,
			params,
			spotDefaultRate,
			&resp)
}

// GetFee to fetch trading fees.
func (e *Exchange) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		multiplier, er := e.getMultiplier(ctx, feeBuilder.IsMaker, feeBuilder)
		if er != nil {
			return 0, er
		}
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount, multiplier)
	case exchange.CryptocurrencyWithdrawalFee:
		wallet, er := e.GetAssetFeesAndWalletStatus(ctx)
		if er != nil {
			return fee, er
		}
		for x := range wallet {
			for y := range wallet[x].NetworkList {
				if wallet[x].NetworkList[y].IsDefault {
					return wallet[x].NetworkList[y].WithdrawFee, nil
				}
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

// getMultiplier retrieves account based taker/maker fees
func (e *Exchange) getMultiplier(ctx context.Context, isMaker bool, feeBuilder *exchange.FeeBuilder) (float64, error) {
	symbol, er := e.FormatSymbol(feeBuilder.Pair, asset.Spot)
	if er != nil {
		return 0, er
	}
	trades, er := e.GetTradeFee(ctx, 0, symbol)
	if er != nil {
		return 0, er
	}
	for x := range trades.TradeFee {
		if trades.TradeFee[x].Symbol == symbol {
			if isMaker {
				return trades.TradeFee[x].Maker, nil
			}
			return trades.TradeFee[x].Taker, nil
		}
	}
	return 0, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.001 * price * amount
}

// calculateTradingFee returns the fee for trading any currency on Binanceus
func calculateTradingFee(purchasePrice, amount, multiplier float64) float64 {
	return (multiplier / 100) * purchasePrice * amount
}

// GetTradeFee to fetch trading fees.
func (e *Exchange) GetTradeFee(ctx context.Context, recvWindow uint64, symbol string) (TradeFeeList, error) {
	timestamp := time.Now().UnixMilli()
	params := url.Values{}
	var resp TradeFeeList
	params.Set("timestamp", strconv.FormatInt(timestamp, 10))
	if recvWindow > 0 {
		if recvWindow < 2000 {
			recvWindow += 3000
		} else if recvWindow > 60000 {
			recvWindow = recvWindowSize5000
		}
		params.Set("recvWindow", strconv.FormatUint(recvWindow, 10))
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	return resp, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		tradeFee,
		params,
		spotDefaultRate,
		&resp)
}

// GetAssetDistributionHistory this endpoint to query
// asset distribution records, including for staking, referrals and airdrops etc.
//
// INPUTS:
// asset: string , startTime & endTime unix time in Milli seconds, recvWindow(duration in milli seconds > 2000 to < 6000)
func (e *Exchange) GetAssetDistributionHistory(ctx context.Context, a string, startTime, endTime int64, recvWindow uint64) (*AssetDistributionHistories, error) {
	params := url.Values{}
	timestamp := time.Now().UnixMilli()
	var resp AssetDistributionHistories
	params.Set("timestamp", strconv.FormatInt(timestamp, 10))
	if startTime > 0 && time.UnixMilli(startTime).Before(time.Now()) {
		params.Set("startTime", strconv.FormatInt(startTime, 10))
	}
	if startTime > 0 {
		params.Set("endTime", strconv.FormatInt(endTime, 10))
	}
	if recvWindow > 0 && recvWindow < 60000 {
		if recvWindow < 2000 {
			recvWindow += 2000
		} else if recvWindow > 6000 {
			recvWindow = recvWindowSize5000
		}
		params.Set("recvWindow", strconv.FormatUint(recvWindow, 10))
	}

	if a != "" {
		params.Set("asset", a)
	}
	return &resp, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, assetDistributionHistory,
		params,
		spotDefaultRate, &resp)
}

// QuickEnableCryptoWithdrawal use this endpoint to enable crypto withdrawals.
func (e *Exchange) QuickEnableCryptoWithdrawal(ctx context.Context) error {
	params := url.Values{}
	response := struct {
		Data any
	}{}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary,
		http.MethodPost,
		accountEnableCryptoWithdrawalEndpoint, params, spotDefaultRate, &(response.Data))
}

// QuickDisableCryptoWithdrawal use this endpoint to disable crypto withdrawals.
func (e *Exchange) QuickDisableCryptoWithdrawal(ctx context.Context) error {
	params := url.Values{}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary,
		http.MethodPost,
		accountDisableCryptoWithdrawalEndpoint, params, spotDefaultRate, nil)
}

// GetUsersSpotAssetSnapshot retrieves a snapshot of list of assets in the account.
func (e *Exchange) GetUsersSpotAssetSnapshot(ctx context.Context, startTime, endTime time.Time, limit, offset uint64) (*SpotAssetsSnapshotResponse, error) {
	params := url.Values{}
	params.Set("type", "SPOT")
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	var resp SpotAssetsSnapshotResponse
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary,
		http.MethodGet, usersSpotAssetsSnapshot,
		params, spotDefaultRate, &resp)
}

// GetSubaccountInformation to fetch your sub-account list.
func (e *Exchange) GetSubaccountInformation(ctx context.Context, page, limit uint64, status, email string) ([]SubAccount, error) {
	params := url.Values{}
	type response struct {
		Success     bool         `json:"success"`
		Subaccounts []SubAccount `json:"subAccounts"`
	}
	var resp response

	if email != "" {
		params.Set("email", email)
	}
	if status != "" && (status == "enabled" || status == "disabled") {
		params.Set("status", status)
	}
	if page != 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	timestamp := time.Now().UnixMilli()
	params.Set("timestamp", strconv.FormatInt(timestamp, 10))
	return resp.Subaccounts, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		subaccountsInformation,
		params,
		spotDefaultRate,
		resp)
}

// GetSubaccountTransferHistory to fetch sub-account asset transfer history.
func (e *Exchange) GetSubaccountTransferHistory(ctx context.Context, email string, startTime, endTime, page, limit int64) ([]TransferHistory, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errNotValidEmailAddress
	}

	params := url.Values{}
	params.Set("email", email)
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	if page != 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	startTimeT := time.UnixMilli(startTime)
	endTimeT := time.UnixMilli(endTime)

	hundredDayBefore := time.Now().Add(-time.Hour * 24 * 100).Truncate(time.Hour)
	if !(startTimeT.Before(hundredDayBefore)) || startTimeT.Before(time.Now()) {
		params.Set("startTime", strconv.FormatInt(startTime, 10))
	}
	if !(endTimeT.Before(hundredDayBefore)) || endTimeT.Before(time.Now()) {
		params.Set("endTime", strconv.FormatInt(endTime, 10))
	}

	var resp struct {
		Success   bool              `json:"success"`
		Transfers []TransferHistory `json:"transfers"`
	}
	return resp.Transfers, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		subaccountTransferHistory,
		params,
		spotDefaultRate,
		resp)
}

// ExecuteSubAccountTransfer to execute sub-account asset transfers.
func (e *Exchange) ExecuteSubAccountTransfer(ctx context.Context, arg *SubAccountTransferRequestParams) (*SubAccountTransferResponse, error) {
	params := url.Values{}
	var response SubAccountTransferResponse
	if !common.MatchesEmailPattern(arg.FromEmail) {
		return nil, errUnacceptableSenderEmail
	}
	if !common.MatchesEmailPattern(arg.ToEmail) {
		return nil, errUnacceptableReceiverEmail
	}
	if len(arg.Asset) <= 2 {
		return nil, errInvalidAssetValue
	}
	if arg.Amount <= 0.0 {
		return nil, errInvalidAssetAmount
	}
	params.Set("fromEmail", arg.FromEmail)
	params.Set("toEmail", arg.ToEmail)
	params.Set("asset", arg.Asset)
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', 0, 64))
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	return &response, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, subaccountTransfer, params, spotDefaultRate, &response)
}

// GetSubaccountAssets to fetch sub-account assets.
func (e *Exchange) GetSubaccountAssets(ctx context.Context, email string) (*SubAccountAssets, error) {
	var resp SubAccountAssets
	if !common.MatchesEmailPattern(email) {
		return nil, errNotValidEmailAddress
	}
	params := url.Values{}
	timestamp := time.Now().UnixMilli()
	params.Set("timestamp", strconv.FormatInt(timestamp, 10))
	params.Set("email", email)
	//
	return &resp, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary, http.MethodGet,
		subaccountAssets, params,
		spotDefaultRate,
		&resp)
}

// GetMasterAccountTotalUSDValue this endpoint to get the total value of assets in the master account in USD.
func (e *Exchange) GetMasterAccountTotalUSDValue(ctx context.Context, email string, page, size int) (*SpotUSDMasterAccounts, error) {
	var response SpotUSDMasterAccounts
	params := url.Values{}
	if email != "" {
		params.Set("email", email)
	}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if size > 0 {
		params.Set("size", strconv.Itoa(size))
	}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	return &response, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary,
		http.MethodGet, masterAccounts, params,
		spotDefaultRate, &response)
}

// GetSubaccountStatusList this endpoint retrieves a status list of sub-accounts.
func (e *Exchange) GetSubaccountStatusList(ctx context.Context, email string) ([]SubAccountStatus, error) {
	params := url.Values{}
	if !common.MatchesEmailPattern(email) {
		return nil, errMissingSubAccountEmail
	}
	params.Set("email", email)
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	var response []SubAccountStatus
	return response, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary,
		http.MethodGet, subAccountStatusList, params,
		spotDefaultRate, &response)
}

// Trade Order Endpoints

// GetOrderRateLimits get the current trade order count rate limits for all time intervals.
// INPUTS: recvWindow <= 60000
func (e *Exchange) GetOrderRateLimits(ctx context.Context, recvWindow uint) ([]OrderRateLimit, error) {
	params := url.Values{}
	timestamp := time.Now().UnixMilli()
	params.Set("timestamp", strconv.FormatInt(timestamp, 10))
	if recvWindow > 1000 && recvWindow < 60000 {
		params.Set("recvWindow", strconv.Itoa(int(recvWindow)))
	} else {
		params.Set("recvWindow", strconv.Itoa(30000))
	}
	var resp []OrderRateLimit
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, orderRateLimit, params, spotOrderRateLimitRate, &resp)
}

// NewOrder sends a new order to Binanceus
func (e *Exchange) NewOrder(ctx context.Context, o *NewOrderRequest) (NewOrderResponse, error) {
	var resp NewOrderResponse
	if err := e.newOrder(ctx, orderRequest, o, &resp); err != nil {
		return resp, err
	}
	if resp.Code != 0 {
		return resp, errors.New(resp.Msg)
	}
	return resp, nil
}

// NewOrderTest sends a new test order to Binanceus
// to test new order creation and signature/recvWindow long. The endpoint creates and validates a new order but does not send it into the matching engine.
func (e *Exchange) NewOrderTest(ctx context.Context, o *NewOrderRequest) (*NewOrderResponse, error) {
	var resp NewOrderResponse
	return &resp, e.newOrder(ctx, testCreateNeworder, o, &resp)
}

// newOrder this endpoint is used by both new order and NewOrderTest passing their route and order information to send new order.
func (e *Exchange) newOrder(ctx context.Context, api string, o *NewOrderRequest, resp *NewOrderResponse) error {
	params := url.Values{}
	symbol, err := e.FormatSymbol(o.Symbol, asset.Spot)
	if err != nil {
		return err
	}
	params.Set("symbol", symbol)
	params.Set("side", o.Side)
	params.Set("type", string(o.TradeType))
	if o.QuoteOrderQty > 0 {
		params.Set("quoteOrderQty", strconv.FormatFloat(o.QuoteOrderQty, 'f', -1, 64))
	} else {
		params.Set("quantity", strconv.FormatFloat(o.Quantity, 'f', -1, 64))
	}
	if o.TradeType == BinanceRequestParamsOrderLimit {
		params.Set("price", strconv.FormatFloat(o.Price, 'f', -1, 64))
	}
	if o.TimeInForce != "" {
		params.Set("timeInForce", o.TimeInForce)
	}
	if o.NewClientOrderID != "" {
		params.Set("newClientOrderId", o.NewClientOrderID)
	}
	if o.StopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(o.StopPrice, 'f', -1, 64))
	}
	if o.IcebergQty != 0 {
		params.Set("icebergQty", strconv.FormatFloat(o.IcebergQty, 'f', -1, 64))
	}
	if o.NewOrderRespType != "" {
		params.Set("newOrderRespType", o.NewOrderRespType)
	}
	return e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodPost, api, params,
		spotOrderRate, resp)
}

// GetOrder to check a trade order's status.
func (e *Exchange) GetOrder(ctx context.Context, arg *OrderRequestParams) (*Order, error) {
	var resp Order
	params := url.Values{}
	if arg.Symbol == "" {
		return nil, errIncompleteArguments
	}
	params.Set("symbol", strings.ToUpper(arg.Symbol))
	if arg.OrderID > 0 {
		params.Set("orderId", strconv.FormatUint(arg.OrderID, 10))
	}
	timestamp := time.Now().UnixMilli()
	params.Set("timestamp", strconv.FormatInt(timestamp, 10))
	if arg.OrigClientOrderID != "" {
		params.Set("origClientOrderId", arg.OrigClientOrderID)
	}
	if arg.recvWindow > 200 && arg.recvWindow <= 6000 {
		params.Set("recvWindow", strconv.Itoa(int(arg.recvWindow)))
	}
	return &resp, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, orderRequest,
		params, spotOrderQueryRate,
		&resp)
}

// GetAllOpenOrders to get all open trade orders on a token symbol. Do not access this without a token symbol as this would return all pair data.
func (e *Exchange) GetAllOpenOrders(ctx context.Context, symbol string) ([]Order, error) {
	var response []Order
	params := url.Values{}

	timestamp := time.Now().UnixMilli()
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	params.Set("timestamp", strconv.FormatInt(timestamp, 10))
	params.Set("recvWindow", recvWindowSize5000String)
	var rateLimit request.EndpointLimit
	if symbol != "" {
		rateLimit = spotOpenOrdersSpecificRate
	} else {
		rateLimit = spotOpenOrdersAllRate
	}
	return response, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary, http.MethodGet,
		openOrders, params,
		rateLimit, &response)
}

// CancelExistingOrder to cancel an active trade order.
func (e *Exchange) CancelExistingOrder(ctx context.Context, arg *CancelOrderRequestParams) (*Order, error) {
	params := url.Values{}
	var response Order
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	symbolValue, err := e.FormatSymbol(arg.Symbol, asset.Spot)
	if err != nil || symbolValue == "" {
		return nil, errMissingCurrencySymbol
	}
	params.Set("symbol", symbolValue)
	if arg.OrderID == "" && arg.ClientSuppliedOrderID == "" {
		return nil, errEitherOrderIDOrClientOrderIDIsRequired
	}
	if arg.ClientSuppliedOrderID != "" {
		params.Set("origClientOrderId", arg.ClientSuppliedOrderID)
	} else {
		params.Set("orderId", arg.OrderID)
	}
	params.Set("recvWindow", recvWindowSize5000String)
	return &response, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodDelete, orderRequest,
		params, spotOrderRate, &response)
}

// CancelOpenOrdersForSymbol request to cancel an open orders.
func (e *Exchange) CancelOpenOrdersForSymbol(ctx context.Context, symbol string) ([]Order, error) {
	params := url.Values{}
	if symbol == "" || len(symbol) < 4 {
		return nil, errMissingCurrencySymbol
	}
	params.Set("symbol", symbol)
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", "5000")
	var response []Order
	return response, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary,
		http.MethodDelete, openOrders,
		params, spotOrderRate, response)
}

// GetTrades to get trade data for a specific account and token symbol.
func (e *Exchange) GetTrades(ctx context.Context, arg *GetTradesParams) ([]Trade, error) {
	var resp []Trade
	params := url.Values{}
	if arg.Symbol == "" || len(arg.Symbol) <= 2 {
		return nil, errIncompleteArguments
	}
	params.Set("symbol", arg.Symbol)
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	if arg.RecvWindow > 3000 {
		params.Set("recvWindow", strconv.FormatUint(arg.RecvWindow, 10))
	}
	if arg.StartTime != nil {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if arg.EndTime != nil {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.FromID > 0 {
		params.Set("fromId", strconv.FormatUint(arg.FromID, 10))
	}
	if arg.Limit > 0 && arg.Limit < 1000 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	} else if arg.Limit > 1000 {
		params.Set("limit", strconv.Itoa(1000))
	}
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, myTrades, params, spotTradesQueryRate, &resp)
}

// OCO Orders

// CreateNewOCOOrder o place a new OCO(one-cancels-the-other) order.
func (e *Exchange) CreateNewOCOOrder(ctx context.Context, arg *OCOOrderInputParams) (*OCOFullOrderResponse, error) {
	params := url.Values{}
	if arg == nil || arg.Symbol == "" || len(arg.Symbol) <= 2 || arg.Quantity == 0 || arg.Side == "" || arg.Price == 0 || arg.StopPrice == 0 {
		return nil, errIncompleteArguments
	}
	params.Set("symbol", arg.Symbol)
	params.Set("quantity", strconv.FormatFloat(arg.Quantity, 'f', 5, 64))
	params.Set("side", arg.Side)
	params.Set("price", strconv.FormatFloat(arg.Price, 'f', 5, 64))
	params.Set("stopPrice", strconv.FormatFloat(arg.StopPrice, 'f', 5, 64))
	if arg.ListClientOrderID != "" {
		params.Set("listClientOrderId", arg.ListClientOrderID)
	}
	if arg.LimitClientOrderID != "" {
		params.Set("limitClientOrderId", arg.LimitClientOrderID)
	}
	if arg.LimitIcebergQty > 0 {
		params.Set("limitIcebergQty", strconv.FormatFloat(arg.LimitIcebergQty, 'f', 5, 64))
	}
	if arg.StopClientOrderID != "" {
		params.Set("stopClientOrderId", arg.StopClientOrderID)
	}
	if arg.StopLimitPrice > 0.0 {
		params.Set("stopLimitPrice", strconv.FormatFloat(arg.StopLimitPrice, 'f', 5, 64))
	}
	if arg.StopIcebergQty > 0.0 {
		params.Set("stopIcebergQty", strconv.FormatFloat(arg.StopIcebergQty, 'f', 5, 64))
	}
	if arg.StopLimitTimeInForce != "" {
		params.Set("stopLimitTimeInForce", arg.StopLimitTimeInForce)
	}
	if arg.NewOrderRespType != "" {
		params.Set("newOrderRespType", arg.NewOrderRespType)
	}
	if arg.RecvWindow > 200 {
		params.Set("recvWindow", strconv.FormatUint(arg.RecvWindow, 10))
	} else {
		params.Set("recvWindow", "6000")
	}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	var response OCOFullOrderResponse
	return &response, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodPost, ocoOrder, params,
		spotOrderRate, &response)
}

// GetOCOOrder to retrieve a specific OCO order based on provided optional parameters.
func (e *Exchange) GetOCOOrder(ctx context.Context, arg *GetOCOOrderRequestParams) (*OCOOrderResponse, error) {
	params := url.Values{}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	switch {
	case arg.OrderListID != "":
		params.Set("orderListId", arg.OrderListID)
	case arg.OrigClientOrderID != "":
		params.Set("origClientOrderId", arg.OrigClientOrderID)
	default:
		return nil, errIncompleteArguments
	}
	params.Set("recvWindow", "60000")
	var response OCOOrderResponse
	return &response, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, ocoOrderList, params, spotSingleOCOOrderRate, &response)
}

// GetAllOCOOrder to retrieve all OCO orders based on provided optional parameters. Please note the maximum limit is 1,000 orders.
func (e *Exchange) GetAllOCOOrder(ctx context.Context, arg *OCOOrdersRequestParams) ([]OCOOrderResponse, error) {
	params := url.Values{}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	var response []OCOOrderResponse
	if arg.FromID > 0 {
		params.Set("fromId", strconv.FormatUint(arg.FromID, 10))
	} else {
		if arg.StartTime.Unix() > 0 && arg.StartTime.Before(arg.EndTime) {
			params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
			params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
		} else if arg.StartTime.Unix() > 0 {
			params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
		}
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.RecvWindow > 0 {
		params.Set("recvWindow", strconv.FormatUint(arg.RecvWindow, 10))
	}
	return response, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary,
		http.MethodGet, ocoAllOrderList,
		params, spotAllOCOOrdersRate,
		&response)
}

// GetOpenOCOOrders to query open OCO orders.
func (e *Exchange) GetOpenOCOOrders(ctx context.Context, recvWindow uint64) ([]OCOOrderResponse, error) {
	params := url.Values{}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	if recvWindow > 0 {
		params.Set("recvWindow", strconv.FormatUint(recvWindow, 10))
	} else {
		params.Set("recvWindow", "30000")
	}
	var response []OCOOrderResponse
	return response, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet,
		ocoOpenOrders, params,
		spotOpenOrdersSpecificRate, &response)
}

// CancelOCOOrder to cancel an entire order list.
func (e *Exchange) CancelOCOOrder(ctx context.Context, arg *OCOOrdersDeleteRequestParams) (*OCOFullOrderResponse, error) {
	var response OCOFullOrderResponse
	params := url.Values{}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	switch {
	case arg.OrderListID > 0:
		params.Set("orderListId", strconv.FormatUint(arg.OrderListID, 10))
	case arg.ListClientOrderID != "":
		params.Set("listClientOrderId", arg.ListClientOrderID)
	default:
		return nil, errIncompleteArguments
	}
	if arg.RecvWindow > 0 {
		params.Set("recvWindow", strconv.FormatUint(arg.RecvWindow, 10))
	}
	return &response, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary,
		http.MethodGet, ocoOrderList,
		params, spotOrderRate, &response)
}

// OTC end points

// GetSupportedCoinPairs to get a list of supported coin pairs for convert.
// returns list of CoinPairInfo
func (e *Exchange) GetSupportedCoinPairs(ctx context.Context, symbol currency.Pair) ([]CoinPairInfo, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("fromCoin", symbol.Base.String())
		params.Set("toCoin", symbol.Quote.String())
	}
	var resp []CoinPairInfo
	return resp, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary, http.MethodGet, otcSelectors,
		params, spotDefaultRate, &resp)
}

// RequestForQuote endpoint to request a quote for a from-to coin pair.
func (e *Exchange) RequestForQuote(ctx context.Context, arg *RequestQuoteParams) (*Quote, error) {
	params := url.Values{}
	var resp Quote
	if arg.FromCoin == "" {
		return nil, errMissingFromCoinName
	}
	if arg.ToCoin == "" {
		return nil, errMissingToCoinName
	}
	if arg.RequestCoin == "" {
		return nil, errMissingRequestCoin
	}
	if arg.RequestAmount <= 0 {
		return nil, errMissingRequestAmount
	}
	params.Set("fromCoin", arg.FromCoin)
	params.Set("toCoin", arg.ToCoin)
	params.Set("requestAmount", strconv.FormatFloat(arg.RequestAmount, 'f', 0, 64))
	params.Set("requestCoin", arg.RequestCoin)
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	return &resp, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost, otcQuotes, params,
		spotDefaultRate, &resp)
}

// PlaceOTCTradeOrder to place an order using an acquired quote.
// returns OTCTradeOrderResponse response containing the OrderID,OrderStatus, and CreateTime information of an order.
func (e *Exchange) PlaceOTCTradeOrder(ctx context.Context, quoteID string) (*OTCTradeOrderResponse, error) {
	params := url.Values{}
	if strings.Trim(quoteID, " ") == "" {
		return nil, errMissingQuoteID
	}
	params.Set("quoteId", quoteID)
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	var response OTCTradeOrderResponse
	return &response, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpot, http.MethodPost,
		otcTradeOrder, params,
		spotOrderRate, &response)
}

// GetOTCTradeOrder returns a single OTC Trade Order instance.
func (e *Exchange) GetOTCTradeOrder(ctx context.Context, orderID uint64) (*OTCTradeOrder, error) {
	var response OTCTradeOrder
	params := url.Values{}
	if orderID <= 0 {
		return nil, errIncompleteArguments
	}
	orderIDStr := strconv.FormatUint(orderID, 10)
	params.Set("orderId", orderIDStr)
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	path := otcTradeOrders + orderIDStr
	return &response, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		path, params,
		spotOrderRate, response)
}

// GetAllOTCTradeOrders returns list of OTC Trade Orders
func (e *Exchange) GetAllOTCTradeOrders(ctx context.Context, arg *OTCTradeOrderRequestParams) ([]OTCTradeOrder, error) {
	params := url.Values{}
	if arg.OrderID != "" {
		params.Set("orderId", arg.OrderID)
	}
	if arg.FromCoin != "" {
		params.Set("fromCoin", arg.FromCoin)
	}
	if !(arg.StartTime.IsZero()) {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !(arg.EndTime.IsZero()) {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.ToCoin != "" {
		params.Set("toCoin", arg.ToCoin)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.Itoa(int(arg.Limit)))
	}
	var response []OTCTradeOrder
	return response, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, otcTradeOrder,
		params, spotOrderRate, &response)
}

// GetAllOCBSTradeOrders use this endpoint to query all OCBS orders by condition.
func (e *Exchange) GetAllOCBSTradeOrders(ctx context.Context, arg OCBSOrderRequestParams) (*OCBSTradeOrdersResponse, error) {
	var resp OCBSTradeOrdersResponse
	params := url.Values{}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	if arg.OrderID != "" {
		params.Set("orderId", arg.OrderID)
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if arg.Limit > 0 && arg.Limit < 100 {
		params.Set("limit", strconv.Itoa(int(arg.Limit)))
	}
	return &resp, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, ocbsTradeOrders,
		params, spotOrderRate, &resp)
}

// Wallet End points

// GetAssetFeesAndWalletStatus to fetch the details of all crypto assets, including fees, withdrawal limits and network status.
// returns the asset wallet detail as a list.
func (e *Exchange) GetAssetFeesAndWalletStatus(ctx context.Context) (AssetWalletList, error) {
	params := url.Values{}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	var response AssetWalletList
	return response, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, assetFeeAndWalletStatus,
		params, spotDefaultRate, &response)
}

// WithdrawCrypto method to withdraw crypto
func (e *Exchange) WithdrawCrypto(ctx context.Context, arg *withdraw.Request) (string, error) {
	params := url.Values{}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	if arg.Currency.String() == "" {
		return "", errMissingRequiredArgumentCoin
	}
	params.Set("coin", arg.Currency.String())
	if arg.Crypto.Chain == "" {
		return "", errMissingRequiredArgumentNetwork
	}
	params.Set("network", arg.Crypto.Chain)
	if arg.ClientOrderID != "" {
		params.Set("withdrawOrderId", arg.ClientOrderID)
	}
	if arg.Crypto.Address == "" {
		return "", errMissingRequiredParameterAddress
	}
	params.Set("address", arg.Crypto.Address)
	if arg.Crypto.AddressTag != "" {
		params.Set("addressTag", arg.Crypto.AddressTag)
	}
	if arg.Amount <= 0 {
		return "", errAmountValueMustBeGreaterThan0
	}
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', 0, 64))
	var response WithdrawalResponse
	if err := e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodPost, applyWithdrawal,
		params, spotDefaultRate, &response); err != nil {
		return "", err
	}
	return response.ID, nil
}

// WithdrawalHistory gets the status of recent withdrawals
// status `param` used as string to prevent default value 0 (for int) interpreting as EmailSent status
func (e *Exchange) WithdrawalHistory(ctx context.Context, c currency.Code, status string, startTime, endTime time.Time, offset, limit int) ([]WithdrawStatusResponse, error) {
	params := url.Values{}
	if !c.IsEmpty() {
		params.Set("coin", c.String())
	}
	if status != "" {
		i, err := strconv.Atoi(status)
		if err != nil {
			return nil, fmt.Errorf("wrong param (status): %s. Error: %v", status, err)
		}
		switch i {
		case EmailSent, Cancelled, AwaitingApproval, Rejected, Processing, Failure, Completed:
		default:
			return nil, fmt.Errorf("wrong param (status): %s", status)
		}
		params.Set("status", status)
	}
	if !startTime.IsZero() && startTime.Unix() != 0 {
		params.Set("startTime", strconv.FormatInt(startTime.Unix(), 10))
	}
	if !endTime.IsZero() && endTime.Unix() != 0 {
		params.Set("endTime", strconv.FormatInt(endTime.Unix(), 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if limit != 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	var withdrawStatus []WithdrawStatusResponse
	if err := e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		withdrawalHistory,
		params,
		spotDefaultRate,
		&withdrawStatus); err != nil {
		return nil, err
	}
	return withdrawStatus, nil
}

// FiatWithdrawalHistory to fetch your fiat (USD) withdrawal history.
// returns FiatAssetHistory containing list of fiat asset records.
func (e *Exchange) FiatWithdrawalHistory(ctx context.Context, arg *FiatWithdrawalRequestParams) (FiatAssetsHistory, error) {
	var response FiatAssetsHistory
	params := url.Values{}
	if !(arg.EndTime.IsZero()) && !(arg.EndTime.Before(time.Now())) {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if !arg.StartTime.IsZero() && !(arg.StartTime.After(time.Now())) {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if arg.FiatCurrency != "" {
		params.Set("fiatCurrency", arg.FiatCurrency)
	}
	if arg.Offset > 0 {
		params.Set("offset", strconv.FormatInt(arg.Offset, 10))
	}
	if arg.PaymentChannel != "" {
		params.Set("paymentChannel", arg.PaymentChannel)
	}
	if arg.PaymentMethod != "" {
		params.Set("paymentMethod", arg.PaymentMethod)
	}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	return response, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, fiatWithdrawalHistory,
		params, spotDefaultRate, &response)
}

// WithdrawFiat to submit a USD withdraw request via Silvergate Exchange Network (SEN).
// returns the Order ID as string
func (e *Exchange) WithdrawFiat(ctx context.Context, arg *WithdrawFiatRequestParams) (string, error) {
	params := url.Values{}
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	if arg == nil {
		return "", errIncompleteArguments
	}
	params.Set("timestamp", timestamp)
	if arg.PaymentChannel != "" {
		params.Set("paymentChannel", arg.PaymentChannel)
	}
	if arg.PaymentMethod != "" {
		params.Set("paymentMethod", arg.PaymentMethod)
	}
	if arg.PaymentAccount == "" {
		return "", errMissingPaymentAccountInfo
	}
	if arg.FiatCurrency != "" {
		params.Set("fiatCurrency", arg.FiatCurrency)
	}
	if arg.Amount <= 0 {
		return "", errAmountValueMustBeGreaterThan0
	}
	type response struct {
		OrderID string `json:"orderId"`
	}
	var resp response
	return resp.OrderID, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary,
		http.MethodPost, withdrawFiat,
		params, spotDefaultRate, &resp,
	)
}

/*
	Deposits
	Get Crypto Deposit Address
*/

// GetDepositAddressForCurrency retrieves the wallet address for a given currency
func (e *Exchange) GetDepositAddressForCurrency(ctx context.Context, coin, chain string) (*DepositAddress, error) {
	params := url.Values{}
	if coin == "" {
		return nil, errMissingRequiredArgumentCoin
	}
	params.Set("coin", coin)
	if chain != "" {
		params.Set("network", chain)
	}
	params.Set("recvWindow", "10000")
	var d DepositAddress
	return &d,
		e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, depositAddress, params, spotDefaultRate, &d)
}

// DepositHistory returns the deposit history based on the supplied params
// status `param` used as string to prevent default value 0 (for int) interpreting as EmailSent status
func (e *Exchange) DepositHistory(ctx context.Context, coin currency.Code, status uint8, startTime, endTime time.Time, offset, limit int) ([]DepositHistory, error) {
	var response []DepositHistory
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}

	if status > 0 {
		switch status {
		case 0 /*Pending*/, 1 /*Success*/, 6 /*Credited but cannot withdraw*/ :
			params.Set("status", strconv.Itoa(int(status)))
		default:
			return nil, fmt.Errorf("wrong param (status) 0 Pending, 1 success, 6 credited but cannot withdraw are allowed: %d ", status)
		}
	}
	if !startTime.IsZero() && startTime.Unix() != 0 {
		params.Set("startTime", strconv.FormatInt(startTime.Unix(), 10))
	}

	if !endTime.IsZero() && endTime.Unix() != 0 {
		params.Set("endTime", strconv.FormatInt(endTime.Unix(), 10))
	}

	if offset != 0 {
		params.Set("offset", strconv.Itoa(offset))
	}

	if limit != 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	if err := e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		depositHistory,
		params,
		spotDefaultRate,
		&response); err != nil {
		return nil, err
	}

	return response, nil
}

// FiatDepositHistory fetch your fiat (USD) deposit history as Fiat Assets History
func (e *Exchange) FiatDepositHistory(ctx context.Context, arg *FiatWithdrawalRequestParams) (FiatAssetsHistory, error) {
	params := url.Values{}
	if !(arg.EndTime.IsZero()) && !(arg.EndTime.Before(time.Now())) {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if !(arg.StartTime.IsZero()) && !(arg.StartTime.After(time.Now())) {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if arg.FiatCurrency != "" {
		params.Set("fiatCurrency", arg.FiatCurrency)
	}
	if arg.Offset > 0 {
		params.Set("offset", strconv.FormatInt(arg.Offset, 10))
	}
	if arg.PaymentChannel != "" {
		params.Set("paymentChannel", arg.PaymentChannel)
	}
	if arg.PaymentMethod != "" {
		params.Set("paymentMethod", arg.PaymentMethod)
	}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	var response FiatAssetsHistory
	return response, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary, http.MethodGet,
		fiatDepositHistory, params, spotDefaultRate, &response)
}

// GetSubAccountDepositAddress retrieves sub-account’s deposit address.
func (e *Exchange) GetSubAccountDepositAddress(ctx context.Context, arg SubAccountDepositAddressRequestParams) (*SubAccountDepositAddress, error) {
	params := url.Values{}
	if !common.MatchesEmailPattern(arg.Email) {
		return nil, errMissingSubAccountEmail
	} else if arg.Coin.String() == "" {
		return nil, errMissingCurrencyCoin
	}
	params.Set("email", arg.Email)
	params.Set("coin", arg.Coin.String())
	var response SubAccountDepositAddress
	return &response, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet,
		subAccountDepositAddress, params, spotDefaultRate, &response)
}

// GetSubAccountDepositHistory retrieves sub-account deposit history.
func (e *Exchange) GetSubAccountDepositHistory(ctx context.Context, email string, coin currency.Code, status int, startTime, endTime time.Time, limit, offset int) ([]SubAccountDepositItem, error) {
	params := url.Values{}
	if !common.MatchesEmailPattern(email) {
		return nil, errMissingSubAccountEmail
	}
	params.Set("email", email)
	if coin.String() != "" {
		params.Set("coin", coin.String())
	}
	if status == 0 || status == 6 || status == 1 {
		params.Set("status", strconv.Itoa(status))
	}
	if !startTime.IsZero() && startTime.Unix() != 0 && startTime.Before(time.Now()) {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() && endTime.Unix() != 0 && endTime.Before(time.Now()) {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	var response []SubAccountDepositItem
	return response, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet,
		subAccountDepositHistory, params, spotDefaultRate, &response)
}

// Referral Endpoints

// GetReferralRewardHistory retrieves the user’s referral reward history.
func (e *Exchange) GetReferralRewardHistory(ctx context.Context, userBusinessType, page, rows int) (*ReferralRewardHistoryResponse, error) {
	params := url.Values{}
	switch {
	case userBusinessType != 0 && userBusinessType != 1:
		return nil, errInvalidUserBusinessType
	case page == 0:
		return nil, errMissingPageNumber
	case rows < 1 || rows > 200:
		return nil, errInvalidRowNumber
	}
	params.Set("userBizType", strconv.Itoa(userBusinessType))
	params.Set("page", strconv.Itoa(page))
	params.Set("rows", strconv.Itoa(rows))
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	var response ReferralRewardHistoryResponse
	return &response, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, referralRewardHistory, params, spotDefaultRate, &response)
}

// SendHTTPRequest sends an unauthenticated request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result any) error {
	endpointPath, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	item := &request.Item{
		Method:                 http.MethodGet,
		Path:                   endpointPath + path,
		Result:                 result,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}
	return e.SendPayload(ctx, f, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAPIKeyHTTPRequest is a special API request where the api key is
// appended to the headers without a secret
func (e *Exchange) SendAPIKeyHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result any) error {
	endpointPath, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}

	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = creds.Key
	item := &request.Item{
		Method:                 http.MethodGet,
		Path:                   endpointPath + path,
		Headers:                headers,
		Result:                 result,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}

	return e.SendPayload(ctx, f, func() (*request.Item, error) {
		return item, nil
	}, request.AuthenticatedRequest)
}

// SendAuthHTTPRequest sends an authenticated HTTP request
func (e *Exchange) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, f request.EndpointLimit, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpointPath, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	if params == nil {
		params = url.Values{}
	}
	if params.Get("recvWindow") == "" {
		params.Set("recvWindow", strconv.FormatInt(defaultRecvWindow.Milliseconds(), 10))
	}
	interim := json.RawMessage{}
	err = e.SendPayload(ctx, f, func() (*request.Item, error) {
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		hmacSigned, err := crypto.GetHMAC(crypto.HashSHA256, []byte(params.Encode()), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		headers := make(map[string]string)
		headers["X-MBX-APIKEY"] = creds.Key
		fullPath := common.EncodeURLValues(endpointPath+path, params) + "&signature=" + hex.EncodeToString(hmacSigned)
		return &request.Item{
			Method:                 method,
			Path:                   fullPath,
			Headers:                headers,
			Result:                 &interim,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return err
	}
	errCap := struct {
		Success bool   `json:"success"`
		Message string `json:"msg"`
		Code    int64  `json:"code"`
	}{}
	if err := json.Unmarshal(interim, &errCap); err == nil {
		if !errCap.Success && errCap.Message != "" && errCap.Code != 200 {
			return errors.New(errCap.Message)
		}
	}
	return json.Unmarshal(interim, result)
}

// ----- Web socket related methods

// GetWsAuthStreamKey this method 'Creates User Data Stream' will retrieve a key to use for authorised WS streaming
// Same as that of Binance
// Start a new user data websocket. The stream will close after 60 minutes unless a keepalive is sent.
// If the account has an active listenKey,
// that listenKey will be returned and its validity will be extended for 60 minutes.
func (e *Exchange) GetWsAuthStreamKey(ctx context.Context) (string, error) {
	endpointPath, err := e.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
	if err != nil {
		return "", err
	}

	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return "", err
	}

	var resp UserAccountStream
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = creds.Key
	item := &request.Item{
		Method:                 http.MethodPost,
		Path:                   endpointPath + userAccountStream,
		Headers:                headers,
		Result:                 &resp,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}

	err = e.SendPayload(ctx, spotDefaultRate, func() (*request.Item, error) {
		return item, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return "", err
	}
	return resp.ListenKey, nil
}

// MaintainWsAuthStreamKey will Extend User Data Stream
// Similar functionality to the same method of Binance.
// Keepalive a user data stream to prevent a time out.
// User data streams will close after 60 minutes.
// It's recommended to send a ping about every 30 minutes.
func (e *Exchange) MaintainWsAuthStreamKey(ctx context.Context) error {
	endpointPath, err := e.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
	if err != nil {
		return err
	}
	if listenKey == "" {
		listenKey, err = e.GetWsAuthStreamKey(ctx)
		return err
	}

	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}

	path := endpointPath + userAccountStream
	params := url.Values{}
	params.Set("listenKey", listenKey)
	path = common.EncodeURLValues(path, params)
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = creds.Key
	item := &request.Item{
		Method:                 http.MethodPut,
		Path:                   path,
		Headers:                headers,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}

	return e.SendPayload(ctx, spotDefaultRate, func() (*request.Item, error) {
		return item, nil
	}, request.AuthenticatedRequest)
}

// CloseUserDataStream Close out a user data websocket.
func (e *Exchange) CloseUserDataStream(ctx context.Context) error {
	endpointPath, err := e.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
	if err != nil {
		return err
	}
	if listenKey == "" {
		listenKey, err = e.GetWsAuthStreamKey(ctx)
		return err
	}

	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}

	path := endpointPath + userAccountStream
	params := url.Values{}
	params.Set("listenKey", listenKey)
	path = common.EncodeURLValues(path, params)
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = creds.Key
	item := &request.Item{
		Method:                 http.MethodDelete,
		Path:                   path,
		Headers:                headers,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}

	return e.SendPayload(ctx, spotDefaultRate, func() (*request.Item, error) {
		return item, nil
	}, request.AuthenticatedRequest)
}
