package binanceus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Binanceus is the overarching type across this package
type Binanceus struct {
	validLimits []int
	exchange.Base
}

const (
	binanceusAPIURL     = "https://api.binance.us"
	binanceusAPIVersion = "/v3"

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
	accountStatus = "/wapi/v3/accountStatus.html"
	tradingStatus = "/wapi/v3/apiTradingStatus.html"
	tradeFee      = "/wapi/v3/tradeFee.html"

	// Wallet endpoints
	assetDistribution = "/sapi/v1/asset/assetDividend"

	// subaccounts ...
	subaccountsInformation    = "/wapi/v3/sub-account/list.html"
	subaccountTransferHistory = "/wapi/v3/sub-account/transfer/history.html"
	subaccountTransfer        = "/wapi/v3/sub-account/transfer.html" // Not Implemented Yet
	subaccountAssets          = "/wapi/v3/sub-account/assets.html"   // Not Implemented Yet

	// Trade Order Endpoints
	orderRateLimit = "/api/v3/rateLimit/order"

	testCreateNeworder = "/api/v3/order/test" // Method: POST
	orderRequest       = "/api/v3/order"      // Used in Create {Method: POST}, Cancel {DELTE}, and get{GET} OrderRequest
	openOrders         = "/api/v3/openOrders"
	myTrades           = "/api/v3/myTrades"

	accountInfo = "/api/v3/account"

	// OCO Orders
	// one-cancels-the-other
	ocoOrder        = "/api/v3/order/oco"
	ocoOrderList    = "/api/v3/orderList"
	ocoAllOrderList = "/api/v3/allOrderList"
	ocoOpenOrders   = "/api/v3/openOrderList"
	// Authenticated endpoints

	//
	otcSelectors   = "/sapi/v1/otc/selectors"
	otcQuotes      = "/sapi/v1/otc/quotes"
	otcTradeOrder  = "/sapi/v1/otc/orders"
	otcTradeOrders = "/sapi/v1/otc/orders/"

	// Wallet endpoints
	assetFeeAndWalletStatus = "/sapi/v1/capital/config/getall"
	applyWithdrawal         = "/sapi/v1/capital/withdraw/apply"
	withdrawalHistory       = "/sapi/v1/capital/withdraw/history"
	withdrawFiat            = "/sapi/v1/fiatpayment/apply/withdraw"
	fiatWithdrawalhistory   = "/sapi/v1/fiatpayment/query/withdraw/history"
	fiatDepositHistory      = "/sapi/v1/fiatpayment/query/deposit/history"
	depositAddress          = "/sapi/v1/capital/deposit/address"
	depositHistory          = "/sapi/v1/capital/deposit/hisrec"

	// Other Consts
	defaultRecvWindow      = 5 * time.Second
	binanceUSAPITimeLayout = "2006-01-02 15:04:05"
)

// EmailRX represents email address maching pattern
var (
	EmailRX                 = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	errNotValidEmailAddress = errors.New("invalid email address")
	// errNoParentUser              = errors.New("error Not parent user")
	errUnacceptableSenderEmail   = errors.New("senders address email is missing")
	errUnacceptableReceiverEmail = errors.New("receiver address email is missing")
	errInvalidAssetValue         = errors.New("invalid asset ")
	errInvalidAssetAmount        = errors.New("invalid asset amount")
	errIncompleteArguments       = errors.New("missing required argument")
)

// MatchesPattern PATTERN , REGULAR EXPRESION
func MatchesEmailPattern(value string) bool {
	if value == "" {
		return false
	}
	if !EmailRX.MatchString(value) {
		return false
	}
	return true
}

// SetValues sets the default valid values
func (b *Binanceus) SetValues() {
	b.validLimits = []int{5, 10, 20, 50, 100, 500, 1000, 5000}
}

// Start implementing public and private exchange API funcs below
func (b *Binanceus) GetExchangeInfo(ctx context.Context) (ExchangeInfo, error) {
	var respo ExchangeInfo
	return respo, b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, exchangeInfo, spotExchangeInfo, &respo)
}

func (b *Binanceus) GetMostRecentTrades(ctx context.Context, rtr RecentTradeRequestParams) ([]RecentTrade, error) {
	params := url.Values{}
	symbol, err := b.FormatSymbol(rtr.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	params.Set("limit", fmt.Sprintf("%d", rtr.Limit))
	path := recentTrades + "?" + params.Encode()
	var resp []RecentTrade
	return resp, b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// GetHistoricalTrades returns historical trade activity
//
// symbol: string of currency pair
// limit: Optional. Default 500; max 1000.
func (b *Binanceus) GetHistoricalTrades(ctx context.Context, hist HistoricalTradeParams) ([]HistoricalTrade, error) {
	var resp []HistoricalTrade
	params := url.Values{}
	params.Set("symbol", hist.Symbol)
	params.Set("limit", fmt.Sprintf("%d", hist.Limit))
	if hist.FromID > 0 {
		params.Set("fromId", fmt.Sprintf("%d", hist.FromID))
	}
	path := historicalTrades + "?" + params.Encode()
	return resp, b.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

func (b *Binanceus) GetAggregateTrades(ctx context.Context, agg *AggregatedTradeRequestParams) ([]AggregatedTrade, error) {
	params := url.Values{}
	symbol, err := b.FormatSymbol(agg.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	needBatch := false
	if agg.Limit > 0 {
		if agg.Limit > 1000 {
			needBatch = true
		} else {
			params.Set("limit", strconv.Itoa(agg.Limit))
		}
	}
	if agg.FromID != 0 {
		params.Set("fromId", strconv.FormatInt(agg.FromID, 10))
	}
	startTime := time.UnixMilli(int64(agg.StartTime))
	endTime := time.UnixMilli(int64(agg.EndTime))

	if (endTime.UnixNano() - startTime.UnixNano()) >= int64(time.Hour) {
		endTime = startTime.Add(time.Minute * 59)
	}

	if !startTime.IsZero() {
		params.Set("startTime", strconv.Itoa(int(agg.StartTime)))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.Itoa(int(agg.EndTime)))
	}
	needBatch = needBatch || (!startTime.IsZero() && !endTime.IsZero() && endTime.Sub(startTime) > time.Hour)
	if needBatch {
		// fromId xor start time must be set
		canBatch := (agg.FromID == 0) != (startTime.IsZero())
		if canBatch {
			return b.batchAggregateTrades(ctx, agg, params)
		}
		// Can't handle this request locally or remotely
		// We would receive {"code":-1128,"msg":"Combination of optional parameters invalid."}
		return nil, errors.New("please set StartTime or FromId, but not both")
	}
	var resp []AggregatedTrade
	path := aggregatedTrades + "?" + params.Encode()
	return resp, b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// batchAggregateTrades fetches trades in multiple requests   <-- copied and amended from the  binance
// first phase, hourly requests until the first trade (or end time) is reached
// second phase, limit requests from previous trade until end time (or limit) is reached
func (b *Binanceus) batchAggregateTrades(ctx context.Context, arg *AggregatedTradeRequestParams, params url.Values) ([]AggregatedTrade, error) {
	var resp []AggregatedTrade
	// prepare first request with only first hour and max limit
	if arg.Limit == 0 || arg.Limit > 1000 {
		// Extend from the default of 500
		params.Set("limit", "1000")
	}
	startTime := time.UnixMilli(int64(arg.StartTime))
	endTime := time.UnixMilli(int64(arg.EndTime))
	var fromID int64
	if arg.FromID > 0 {
		fromID = arg.FromID
	} else {
		// Only 10 seconds is used to prevent limit of 1000 being reached in the first request,
		// cutting off trades for high activity pairs
		increment := time.Second * 10
		for len(resp) == 0 {
			startTime = startTime.Add(increment)
			if !endTime.IsZero() && !startTime.Before(endTime) {
				// All requests returned empty
				return nil, nil
			}
			params.Set("startTime", strconv.Itoa(int(startTime.UnixMilli())))
			params.Set("endTime", strconv.Itoa(int(startTime.Add(increment).UnixMilli())))
			path := aggregatedTrades + "?" + params.Encode()
			println(path)
			err := b.SendHTTPRequest(ctx,
				exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
			if err != nil {
				log.Warn(log.ExchangeSys, err.Error())
				return resp, err
			}
		}
		fromID = resp[len(resp)-1].ATradeID
	}

	// other requests follow from the last aggregate trade id and have no time window
	params.Del("startTime")
	params.Del("endTime")
	// while we haven't reached the limit
	for ; arg.Limit == 0 || len(resp) < arg.Limit; fromID = resp[len(resp)-1].ATradeID {
		// Keep requesting new data after last retrieved trade
		params.Set("fromId", strconv.FormatInt(fromID, 10))
		path := aggregatedTrades + "?" + params.Encode()
		var additionalTrades []AggregatedTrade
		err := b.SendHTTPRequest(ctx,
			exchange.RestSpotSupplementary,
			path,
			spotDefaultRate,
			&additionalTrades)
		if err != nil {
			return resp, err
		}
		lastIndex := len(additionalTrades)
		if !endTime.IsZero() {
			// get index for truncating to end time
			lastIndex = sort.Search(len(additionalTrades), func(i int) bool {
				return endTime.Before(additionalTrades[i].TimeStamp)
			})
		}
		// don't include the first as the request was inclusive from last ATradeID
		resp = append(resp, additionalTrades[1:lastIndex]...)
		// If only the starting trade is returned or if we received trades after end time
		if len(additionalTrades) == 1 || lastIndex < len(additionalTrades) {
			// We found the end
			break
		}
	}
	// Truncate if necessary
	if arg.Limit > 0 && len(resp) > arg.Limit {
		resp = resp[:arg.Limit]
	}
	return resp, nil
}

// GetOrderBookDepth ...
func (b *Binanceus) GetOrderBookDepth(ctx context.Context, arg *OrderBookDataRequestParams) (*OrderBook, error) {
	if err := b.CheckLimit(arg.Limit); err != nil {
		return nil, err
	}
	params := url.Values{}
	symbol, err := b.FormatSymbol(arg.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	params.Set("limit", fmt.Sprintf("%d", arg.Limit))
	var resp OrderBookData
	if err := b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		orderBookDepth+"?"+params.Encode(),
		orderbookLimit(arg.Limit), &resp); err != nil {
		return nil, err
	}
	orderbook := OrderBook{
		Bids:         make([]OrderbookItem, len(resp.Bids)),
		Asks:         make([]OrderbookItem, len(resp.Asks)),
		LastUpdateID: resp.LastUpdateID,
	}
	for x := range resp.Bids {
		price, err := strconv.ParseFloat(resp.Bids[x][0], 64)
		if err != nil {
			return nil, err
		}
		amount, err := strconv.ParseFloat(resp.Bids[x][1], 64)
		if err != nil {
			return nil, err
		}
		orderbook.Bids[x] = OrderbookItem{
			Price:    price,
			Quantity: amount,
		}
	}
	for x := range resp.Asks {
		price, err := strconv.ParseFloat(resp.Asks[x][0], 64)
		if err != nil {
			return nil, err
		}
		amount, err := strconv.ParseFloat(resp.Asks[x][1], 64)
		if err != nil {
			return nil, err
		}
		orderbook.Asks[x] = OrderbookItem{
			Price:    price,
			Quantity: amount,
		}
	}
	return &orderbook, nil
}

// CheckLimit checks value against a variable list
func (b *Binanceus) CheckLimit(limit int) error {
	for x := range b.validLimits {
		if b.validLimits[x] == limit {
			return nil
		}
	}
	return errors.New("incorrect limit values - valid values are 5, 10, 20, 50, 100, 500, 1000")
}

func (b *Binanceus) GetSpotKline(ctx context.Context, arg *KlinesRequestParams) ([]CandleStick, error) {
	symbol, err := b.FormatSymbol(arg.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", arg.Interval)
	if arg.Limit != 0 {
		params.Set("limit", strconv.Itoa(arg.Limit))
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt((arg.StartTime).UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt((arg.EndTime).UnixMilli(), 10))
	}

	path := candleStick + "?" + params.Encode()
	var resp interface{}

	err = b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		path,
		spotDefaultRate,
		&resp)
	if err != nil {
		return nil, err
	}
	responseData, ok := resp.([]interface{})
	if !ok {
		return nil, errors.New("unable to type assert responseData")
	}

	klineData := make([]CandleStick, len(responseData))
	for x := range responseData {
		individualData, ok := responseData[x].([]interface{})
		if !ok {
			return nil, errors.New("unable to type assert individualData")
		}
		if len(individualData) != 12 {
			return nil, errors.New("unexpected kline data length")
		}
		var candle CandleStick
		candle.OpenTime = time.UnixMilli(int64(individualData[0].(float64)))
		if candle.Open, err = convert.FloatFromString(individualData[1]); err != nil {
			return nil, err
		}
		if candle.High, err = convert.FloatFromString(individualData[2]); err != nil {
			return nil, err
		}
		if candle.Low, err = convert.FloatFromString(individualData[3]); err != nil {
			return nil, err
		}
		if candle.Close, err = convert.FloatFromString(individualData[4]); err != nil {
			return nil, err
		}
		if candle.Volume, err = convert.FloatFromString(individualData[5]); err != nil {
			return nil, err
		}
		candle.CloseTime = time.UnixMilli(int64(individualData[6].(float64)))
		if candle.QuoteAssetVolume, err = convert.FloatFromString(individualData[7]); err != nil {
			return nil, err
		}
		if candle.TradeCount, ok = individualData[8].(float64); !ok {
			return nil, errors.New("unable to type assert trade count")
		}
		if candle.TakerBuyAssetVolume, err = convert.FloatFromString(individualData[9]); err != nil {
			return nil, err
		}
		if candle.TakerBuyQuoteAssetVolume, err = convert.FloatFromString(individualData[10]); err != nil {
			return nil, err
		}
		klineData[x] = candle
	}
	return klineData, nil
}

func (b *Binanceus) GetSinglePriceData(ctx context.Context, symbol currency.Pair) (SymbolPrice, error) {
	var res SymbolPrice
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return res, err
	}
	params.Set("symbol", symbolValue)
	path := tickerPrice + "?" + params.Encode()
	return res, b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &res)
}

func (b *Binanceus) GetPriceDatas(ctx context.Context) (SymbolPrices, error) {
	var res SymbolPrices
	return res, b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, tickerPrice, spotDefaultRate, &res)
}

// GetAveragePrice returns current average price for a symbol.
//
// symbol: string of currency pair
func (b *Binanceus) GetAveragePrice(ctx context.Context, symbol currency.Pair) (AveragePrice, error) {
	resp := AveragePrice{}
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)

	path := averagePrice + "?" + params.Encode()

	return resp, b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// GetBestPrice returns the latest best price for symbol
//
// symbol: string of currency pair
func (b *Binanceus) GetBestPrice(ctx context.Context, symbol currency.Pair) (BestPrice, error) {
	resp := BestPrice{}
	params := url.Values{}
	rateLimit := spotOrderbookTickerAllRate
	if !symbol.IsEmpty() {
		rateLimit = spotDefaultRate
		symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	path := bestPrice + "?" + params.Encode()

	return resp,
		b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, rateLimit, &resp)
}

// GetPriceChangeStats returns price change statistics for the last 24 hours
//
// symbol: string of currency pair
func (b *Binanceus) GetPriceChangeStats(ctx context.Context, symbol currency.Pair) (PriceChangeStats, error) {
	resp := PriceChangeStats{}
	params := url.Values{}
	rateLimit := spotPriceChangeAllRate
	if !symbol.IsEmpty() {
		rateLimit = spotDefaultRate
		symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	path := priceChange + "?" + params.Encode()

	return resp, b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, path, rateLimit, &resp)
}

// GetTickers returns the ticker data for the last 24 hrs
func (b *Binanceus) GetTickers(ctx context.Context) ([]PriceChangeStats, error) {
	var resp []PriceChangeStats
	return resp, b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, priceChange, spotPriceChangeAllRate, &resp)
}

// GetAccount returns binance user accounts
func (b *Binanceus) GetAccount(ctx context.Context) (*Account, error) {
	type response struct {
		Response
		Account
	}

	var resp response
	params := url.Values{}

	if err := b.SendAuthHTTPRequest(ctx,
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

func (b *Binanceus) GetUserAccountStatus(ctx context.Context, recvWindow uint) (*AccountStatusResponse, error) {
	var resp AccountStatusResponse
	params := url.Values{}
	timestamp := time.Now().UnixMilli()
	params.Set("timestamp", strconv.Itoa(int(timestamp)))
	if recvWindow > 0 {
		if recvWindow < 2000 {
			recvWindow += 1500
		}
	} else {
		recvWindow = uint(defaultRecvWindow)
	}
	params.Set("recvWindow", strconv.Itoa(int(recvWindow)))
	return &resp,
		b.SendAuthHTTPRequest(ctx,
			exchange.RestSpotSupplementary,
			http.MethodGet,
			accountStatus,
			params,
			spotAccountInformationRate,
			&resp)
}

func (b *Binanceus) GetUserAPITradingStatus(ctx context.Context, recvWindow uint) (*TradeStatus, error) {
	type response struct {
		Success bool        `json:"success"`
		TC      TradeStatus `json:"status"`
	}
	var resp response
	params := url.Values{}
	timestamp := time.Now().UnixMilli()
	params.Set("timestamp", strconv.Itoa(int(timestamp)))
	if recvWindow > 0 {
		if recvWindow < 2000 {
			recvWindow += 1500
		}
	} else {
		recvWindow = uint(defaultRecvWindow)
	}
	params.Set("recvWindow", strconv.Itoa(int(recvWindow)))
	return &(resp.TC),
		b.SendAuthHTTPRequest(ctx,
			exchange.RestSpotSupplementary,
			http.MethodGet,
			tradingStatus,
			params,
			spotAccountInformationRate,
			&resp)
}

func (b *Binanceus) GetTradeFee(ctx context.Context, recvWindow uint, symbol string) (*TradeFeeList, error) {
	timestamp := time.Now().UnixMilli()
	params := url.Values{}
	var resp TradeFeeList
	params.Set("timestamp", strconv.Itoa(int(timestamp)))
	if recvWindow > 0 {
		if recvWindow < 2000 {
			recvWindow += 2000
		} else if recvWindow > 6000 {
			recvWindow = 5000
		}
	} else {
		recvWindow = uint(defaultRecvWindow)
	}
	params.Set("recvWindow", strconv.Itoa(int(recvWindow)))
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	return &resp, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		tradeFee,
		params,
		spotDefaultRate,
		&resp)
}

// GetAssetDistributionHistory this endpoint to query
// asset distribution records, including for staking, referrals and airdrops etc.
// INPUTS:
//       asset: string , startTime & endTime unix time in Milli seconds, recvWindow(duration in milli seconds > 2000 to < 6000)
func (b *Binanceus) GetAssetDistributionHistory(ctx context.Context, asset string, startTime, endTime uint64, recvWindow uint) (*AssetDistributionHistories, error) {
	params := url.Values{}
	timestamp := time.Now().UnixMilli()
	var resp AssetDistributionHistories
	params.Set("timestamp", strconv.Itoa(int(timestamp)))
	// assetDistribution
	if startTime > 0 && time.UnixMilli(int64(startTime)).Before(time.Now()) {
		params.Set("startTime", strconv.Itoa(int(startTime)))
	}
	if startTime > 0 {
		params.Set("endTime", strconv.Itoa(int(endTime)))
	}
	if recvWindow > 0 {
		if recvWindow < 2000 {
			recvWindow += 2000
		} else if recvWindow > 6000 {
			recvWindow = 5000
		}
	} else {
		recvWindow = uint(defaultRecvWindow)
	}
	params.Set("recvWindow", strconv.Itoa(int(recvWindow)))
	if asset != "" {
		params.Set("asset", asset)
	}
	return &resp, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, assetDistribution,
		params,
		spotDefaultRate, &resp)
}

func (b *Binanceus) GetSubaccountInformation(ctx context.Context, page uint, limit uint, status string, email string) ([]*SubAccount, error) {
	params := url.Values{}
	type response struct {
		Success     bool          `json:"success"`
		Subaccounts []*SubAccount `json:"subAccounts"`
	}
	var resp response

	if email != "" {
		params.Set("email", email)
	}
	if status != "" && (status == "enabled" || status == "disabled") {
		params.Set("status", status)
	}
	if page != 0 {
		params.Set("page", strconv.Itoa(int(page)))
	}
	if limit != 0 {
		params.Set("limit", strconv.Itoa(int(limit)))
	}
	// params.Set("recvWindow", strconv.Itoa(5000))
	timestamp := time.Now().UnixMilli()
	params.Set("timestamp", strconv.Itoa(int(timestamp)))
	return resp.Subaccounts, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		subaccountsInformation,
		params,
		spotDefaultRate,
		resp)
}

func (b *Binanceus) GetSubaccountTransferhistory(ctx context.Context,
	email string,
	startTime uint64,
	endTime uint64,
	page, limit int) ([]*TransferHistory, error) {
	timestamp := time.Now().UnixMilli()
	params := url.Values{}
	type response struct {
		Success   bool               `json:"success"`
		Transfers []*TransferHistory `json:"transfers"`
	}
	var resp response
	if !MatchesEmailPattern(email) {
		return nil, errNotValidEmailAddress
	}
	params.Set("email", email)
	params.Set("timestamp", strconv.Itoa(int(timestamp)))
	if page != 0 {
		params.Set("page", strconv.Itoa(int(page)))
	}
	if limit != 0 {
		params.Set("limit", strconv.Itoa(int(limit)))
	}
	startTimeT := time.UnixMilli(int64(startTime))
	endTimeT := time.UnixMilli(int64(endTime))

	hundredDayBefore := time.Now()
	hundredDayBefore.Sub(time.UnixMilli(int64((time.Hour * 24 * 10) / time.Millisecond)))
	if !(startTimeT.Before(hundredDayBefore)) || !startTimeT.After(time.Now()) {
		params.Set("startTime", strconv.Itoa(int(startTime)))
	}
	if !(endTimeT.Before(hundredDayBefore)) || !endTimeT.After(time.Now()) {
		params.Set("startTime", strconv.Itoa(int(endTime)))
	}
	return resp.Transfers, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		subaccountTransferHistory,
		params,
		spotDefaultRate,
		resp)
}

// TODO: Execute Sub-account Transfer
// POST /wapi/v3/sub-account/transfer.html (HMAC SHA256)
// Use this endpoint to execute sub-account asset transfers.
func (b *Binanceus) ExecuteSubAccountTransfer(ctx context.Context, arg *SubaccountTransferRequestParams) (*SubaccountTransferResponse, error) {
	params := url.Values{}
	var response SubaccountTransferResponse
	if !MatchesEmailPattern(arg.FromEmail) {
		return nil, errUnacceptableSenderEmail
	}
	if !MatchesEmailPattern(arg.ToEmail) {
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
	params.Set("amount", fmt.Sprintf("%f", arg.Amount))
	params.Set("recvWindow", "5000")
	params.Set("timestamp", fmt.Sprintf("%d", time.Now().UnixMilli()))
	return &response, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, subaccountTransfer, params, spotDefaultRate, &response)
}

// ---

// TODO: Get Sub-account Assets
// GET /wapi/v3/sub-account/assets.html (HMAC SHA256)
// Use this endpoint to fetch sub-account assets.
func (b *Binanceus) GetSubaccountAssets(ctx context.Context, email string) (*SubAccountAssets, error) {
	var resp SubAccountAssets
	if !MatchesEmailPattern(email) {
		return nil, errNotValidEmailAddress
	}
	params := url.Values{}
	timestamp := time.Now().UnixMilli()
	params.Set("timestamp", fmt.Sprintf("%d", timestamp))
	params.Set("email", email)
	//
	return &resp, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary, http.MethodGet,
		subaccountAssets, params,
		spotDefaultRate,
		&resp)
}

// Trade Order Endpoints

// GetOrderRateLimits
// GET /api/v3/rateLimit/order (HMAC SHA256)
// Get the current trade order count rate limits for all time intervals.
// INPUTS: recvWindow <= 60000
func (b *Binanceus) GetOrderRateLimits(ctx context.Context, recvWindow uint) ([]OrderRateLimit, error) {
	params := url.Values{}
	timestamp := time.Now().UnixMilli()
	params.Set("timestamp", strconv.Itoa(int(timestamp)))
	// if !(recvWindow > 60000 || recvWindow <= 0) {
	// 	params.Set("recvWindow", strconv.Itoa(int(recvWindow)))
	// } else {
	params.Set("recvWindow", strconv.Itoa(30000))
	// }
	var resp []OrderRateLimit
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, orderRateLimit, params, spotDefaultRate, &resp)
}

// NewOrder sends a new order to Binance
func (b *Binanceus) NewOrder(ctx context.Context, o *NewOrderRequest) (NewOrderResponse, error) {
	var resp NewOrderResponse
	if err := b.newOrder(ctx, orderRequest, o, &resp); err != nil {
		return resp, err
	}
	if resp.Code != 0 {
		return resp, errors.New(resp.Msg)
	}
	return resp, nil
}

// NewOrderTest sends a new test order to Binance
func (b *Binanceus) NewOrderTest(ctx context.Context, o *NewOrderRequest) (*NewOrderResponse, error) {
	var resp NewOrderResponse
	return &resp, b.newOrder(ctx, testCreateNeworder, o, &resp)
}
func (b *Binanceus) newOrder(ctx context.Context, api string, o *NewOrderRequest, resp *NewOrderResponse) error {
	params := url.Values{}
	symbol, err := b.FormatSymbol(o.Symbol, asset.Spot)
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
		params.Set("timeInForce", string(o.TimeInForce))
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
	return b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, api, params, spotOrderRate, resp)
}

func (b *Binanceus) GetOrder(ctx context.Context, arg OrderRequestParams) (*Order, error) {
	var resp Order
	params := url.Values{}
	if arg.Symbol == "" {
		return nil, errIncompleteArguments
	}
	params.Set("symbol", strings.ToUpper(arg.Symbol))
	if arg.OrderID > 0 {
		params.Set("orderId", fmt.Sprintf("%d", arg.OrderID))
	}
	timestamp := time.Now().UnixMilli()
	params.Set("timestamp", fmt.Sprintf("%d", timestamp))
	if arg.OrigClientOrderId != "" {
		params.Set("origClientOrderId", arg.OrigClientOrderId)
	}
	//  else {
	// 	return nil, errIncompleteArguments
	// }
	if arg.RecvWindow > 200 && arg.RecvWindow <= 6000 {
		params.Set("recvWindow", fmt.Sprint(arg.RecvWindow))
	}
	return &resp, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, orderRequest,
		params, spotDefaultRate,
		&resp)
}

func (b *Binanceus) GetAllOpenOrders(ctx context.Context, symbol string) ([]*Order, error) {
	var response []*Order
	params := url.Values{}

	timestamp := time.Now().UnixMilli()
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	params.Set("timestamp", fmt.Sprint(timestamp))
	params.Set("recvWindow", fmt.Sprint(5000))
	return response, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary, http.MethodGet,
		openOrders, params,
		spotDefaultRate, &response)
}

func (b *Binanceus) CancelExistingOrder(ctx context.Context, arg CancelOrderRequestParams) (*Order, error) {
	params := url.Values{}
	var response Order
	symbolValue, err := b.FormatSymbol(arg.Symbol, asset.Spot)
	if err != nil || symbolValue == "" {
		return nil, errIncompleteArguments
	}
	params.Set("symbol", symbolValue)
	if arg.OrigClientOrderID != "" {
		params.Set("origClientOrderID", arg.OrigClientOrderID)
	}
	if arg.OrderID != 0 {
		params.Set("orderID", fmt.Sprint(arg.OrderID))
	}
	return &response, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodDelete, orderRequest,
		params, spotDefaultRate, &response)
}

// CancelOpenOrdersForSymbol request to cancel an open orders.
func (b *Binanceus) CancelOpenOrdersForSymbol(ctx context.Context, symbol string) ([]Order, error) {
	params := url.Values{}
	if symbol == "" || len(symbol) < 4 {
		return nil, errIncompleteArguments
	}
	params.Set("symbol", symbol)
	params.Set("timestamp", fmt.Sprintf("%d", time.Now().UnixMilli()))
	params.Set("recvWindow", "5000")
	var response []Order
	return response, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary,
		http.MethodDelete, openOrders,
		params, spotDefaultRate, response)
}

func (b *Binanceus) GetTrades(ctx context.Context, arg GetTradesParams) ([]*Trade, error) {
	var resp []*Trade
	params := url.Values{}
	if arg.Symbol == "" || len(arg.Symbol) <= 2 {
		return nil, errIncompleteArguments
	}
	params.Set("symbol", arg.Symbol)
	params.Set("timestamp", fmt.Sprint(time.Now().UnixMilli()))
	if arg.RecvWindow > 3000 {
		params.Set("recvWindow", fmt.Sprint(arg.RecvWindow))
	}
	if arg.StartTime != nil {
		params.Set("startTime", fmt.Sprint(arg.StartTime.UnixMilli()))
	}
	if arg.EndTime != nil {
		params.Set("endTime", fmt.Sprint(arg.EndTime.UnixMilli()))
	}
	if arg.FromID > 0 {
		params.Set("fromId", fmt.Sprint(arg.FromID))
	}
	if arg.Limit > 0 && arg.Limit < 1000 {
		params.Set("limit", fmt.Sprint(arg.Limit))
	} else if arg.Limit > 1000 {
		params.Set("limit", fmt.Sprint(1000))
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, myTrades, params, spotDefaultRate, &resp)
}

// OCO Orders
//

// CreateNewOCOOrder --
func (b *Binanceus) CreateNewOCOOrder(ctx context.Context, arg OCOOrderInputParams) (*OCOFullOrderResponse, error) {
	params := url.Values{}
	if arg.Symbol == "" || len(arg.Symbol) <= 2 {
		return nil, errIncompleteArguments
	}
	if arg.Quantity == 0 {
		return nil, errIncompleteArguments
	}
	if arg.Side == "" {
		return nil, errIncompleteArguments
	}
	if arg.Price == 0 {
		return nil, errIncompleteArguments
	}
	if arg.StopPrice == 0 {
		return nil, errIncompleteArguments
	}
	params.Set("symbol", arg.Symbol)
	params.Set("quantity", fmt.Sprint(arg.Quantity))
	params.Set("side", arg.Side)
	params.Set("price", fmt.Sprint(arg.Price))
	params.Set("stopPrice", fmt.Sprint(arg.StopPrice))
	if arg.ListClientOrderID != "" {
		params.Set("listClientOrderId", fmt.Sprint(arg.ListClientOrderID))
	}
	if arg.LimitClientOrderID != "" {
		params.Set("limitClientOrderId", fmt.Sprint(arg.LimitClientOrderID))
	}
	if arg.LimitIcebergQty > 0 {
		params.Set("limitIcebergQty", fmt.Sprint(arg.LimitIcebergQty))
	}
	if arg.StopClientOrderID > "" {
		params.Set("stopClientOrderId", (arg.StopClientOrderID))
	}
	if arg.StopLimitPrice > 0.0 {
		params.Set("stopLimitPrice", fmt.Sprint(arg.StopLimitPrice))
	}
	if arg.StopIcebergQty > 0.0 {
		params.Set("stopIcebergQty", fmt.Sprint(arg.StopIcebergQty))
	}
	if arg.StopLimitTimeInForce != "" {
		params.Set("stopLimitTimeInForce", arg.StopLimitTimeInForce)
	}
	if arg.NewOrderRespType != "" {
		params.Set("newOrderRespType", arg.NewOrderRespType)
	}
	if arg.RecvWindow > 200 {
		params.Set("recvWindow", fmt.Sprint(arg.RecvWindow))
	}
	timestamp := time.Now().UnixMilli()
	params.Set("timestamp", fmt.Sprint(timestamp))
	var response OCOFullOrderResponse
	return &response, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodPost, ocoOrder, params,
		spotDefaultRate, &response)
}

// GetOCOOrder
func (b *Binanceus) GetOCOOrder(ctx context.Context, arg GetOCOPrderRequestParams) (*OCOOrderResponse, error) {
	params := url.Values{}
	params.Set("timestamp", fmt.Sprint(time.Now().UnixMilli()))
	if arg.OrderListID <= 0 && arg.OrigClientOrderID == "" {
		return nil, errIncompleteArguments
	}
	if arg.OrderListID > 0 {
		params.Set("orderListId", fmt.Sprint(arg.OrderListID))
	} else if arg.OrigClientOrderID != "" {
		params.Set("origClientOrderId", arg.OrigClientOrderID)
	} else {
		return nil, errIncompleteArguments
	}
	params.Set("recvWindow", "60000")
	var response OCOOrderResponse
	return &response, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, ocoOrderList, params, spotDefaultRate, &response)
}

// GetAllOCOOrder   ...
func (b *Binanceus) GetAllOCOOrder(ctx context.Context, arg OCOOrdersRequestParams) ([]*OCOOrderResponse, error) {
	params := url.Values{}
	params.Set("timestamp", fmt.Sprint(time.Now().UnixMilli()))
	var response []*OCOOrderResponse
	if arg.FromID > 0 {
		params.Set("fromId", fmt.Sprint(arg.FromID))
	} else {
		if arg.StartTime.Unix() > 0 && arg.StartTime.Before(arg.EndTime) {
			params.Set("startTime", fmt.Sprint(arg.StartTime.UnixMilli()))
			params.Set("endTime", fmt.Sprint(arg.EndTime.UnixMilli()))
		} else if arg.StartTime.Unix() > 0 {
			params.Set("startTime", fmt.Sprint(arg.StartTime.UnixMilli()))
		}
	}
	if arg.Limit > 0 {
		params.Set("limit", fmt.Sprint(arg.Limit))
	}
	if arg.RecvWindow > 0 {
		params.Set("recvWindow", fmt.Sprint(arg.RecvWindow))
	}
	return response, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary,
		http.MethodGet, ocoAllOrderList,
		params, spotDefaultRate,
		&response)
}

// GetOpenOCOOrders ...
func (b *Binanceus) GetOpenOCOOrders(ctx context.Context, RecvWindow uint) ([]*OCOOrderResponse, error) {
	params := url.Values{}
	params.Set("timestamp", fmt.Sprint(time.Now().UnixMilli()))
	if RecvWindow > 0 {
		params.Set("recvWindow", fmt.Sprint(RecvWindow))
	} else {
		params.Set("recvWindow", "30000")
	}
	var response []*OCOOrderResponse
	return response, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet,
		ocoOpenOrders, params,
		spotDefaultRate, &response)
}

// CancelOCOOrder ...
func (b *Binanceus) CancelOCOOrder(ctx context.Context, arg OCOOrdersDeleteRequestParams) (*OCOFullOrderResponse, error) {
	var response OCOFullOrderResponse
	params := url.Values{}
	params.Set("timestamp", fmt.Sprint(time.Now().UnixMilli()))
	if arg.Symbol == "" || len(arg.Symbol) <= 2 {
		return nil, errIncompleteArguments
	}
	if arg.OrderListID > 0 {
		params.Set("orderListId", fmt.Sprint(arg.OrderListID))
	} else if arg.ListClientOrderID != "" {
		params.Set("listClientOrderId", fmt.Sprint(arg.ListClientOrderID))
	} else {
		return nil, errIncompleteArguments
	}
	if arg.RecvWindow > 0 {
		params.Set("recvWindow", fmt.Sprint(arg.RecvWindow))
	}
	return &response, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary,
		http.MethodGet, ocoOrderList,
		params, spotDefaultRate, &response)
}

// OTC end points

func (b *Binanceus) GetSupportedCoinPairs(ctx context.Context, symbol currency.Pair) ([]*CoinPairInfo, error) {
	params := url.Values{}
	if symbol.Base.String() != "" && symbol.Quote.String() != "" {
		params.Set("fromCoin", symbol.Base.String())
		params.Set("toCoin", symbol.Quote.String())
	}
	var resp []*CoinPairInfo
	return resp, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary, http.MethodGet, otcSelectors,
		params, spotDefaultRate, &resp)
}

// RequestForQuote endpoint to request a quote for a from-to coin pair.
func (b *Binanceus) RequestForQuote(ctx context.Context, arg RequestQuoteParams) (*RequestQuote, error) {
	params := url.Values{}
	var resp RequestQuote
	if arg.FromCoin == "" {
		return nil, errIncompleteArguments
	}
	if arg.ToCoin == "" {
		return nil, errIncompleteArguments
	}
	if arg.RequestCoin == "" {
		return nil, errIncompleteArguments
	}
	if arg.RequestAmount <= 0 {
		return nil, errIncompleteArguments
	}
	params.Set("fromCoin", arg.FromCoin)
	params.Set("toCoin", arg.ToCoin)
	params.Set("requestAmount", fmt.Sprint(arg.RequestAmount))
	params.Set("requestCoin", arg.RequestCoin)
	return &resp, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost, otcQuotes, params,
		spotDefaultRate, resp)
}

// PlaceOTCTradeOrder
func (b *Binanceus) PlaceOTCTradeOrder(ctx context.Context, quoteID string) (*OTCTradeOrderResponse, error) {
	params := url.Values{}
	if quoteID == "" {
		return nil, errIncompleteArguments
	}
	params.Set("quoteId", quoteID)
	params.Set("timestamp", fmt.Sprint(time.Now().UnixMilli()))
	var response OTCTradeOrderResponse
	return &response, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpot, http.MethodPost,
		otcTradeOrder, params,
		spotDefaultRate, &response)
}

// GetOTCTradeOrder
// /sapi/v1/otc/orders/{orderId}
func (b *Binanceus) GetOTCTradeOrder(ctx context.Context, orderID uint64) (*OTCTradeOrder, error) {
	var response OTCTradeOrder
	params := url.Values{}
	if orderID <= 0 {
		return nil, errIncompleteArguments
	}
	params.Set("orderid", fmt.Sprint(orderID))
	params.Set("timestamp", fmt.Sprint(time.Now().UnixMilli()))
	path := otcTradeOrders + fmt.Sprint(orderID)
	return &response, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		path, params,
		spotDefaultRate, response)
}

// GetAllOTCTradeOrders ...
func (b *Binanceus) GetAllOTCTradeOrders(ctx context.Context, arg OTCTradeOrderRequestParams) ([]*OTCTradeOrder, error) {
	params := url.Values{}
	if arg.OrderId != "" {
		params.Set("orderId", arg.OrderId)
	}
	if arg.FromCoin != "" {
		params.Set("fromCoin", arg.FromCoin)
	}
	if (arg.StartTime) != nil {
		params.Set("startTime", fmt.Sprint(arg.StartTime.UnixMilli()))
	}
	if arg.EndTime != nil {
		params.Set("endTime", fmt.Sprint(arg.EndTime.UnixMilli()))
	}
	if arg.ToCoin != "" {
		params.Set("toCoin", fmt.Sprint(arg.ToCoin))
	}
	if arg.Limit > 0 {
		params.Set("limit", fmt.Sprint(arg.Limit))
	}
	var response []*OTCTradeOrder
	return response, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, otcTradeOrder,
		params, spotDefaultRate, &response)
}

// Wallet End points
//
//

// GetAssetFeesAndWalletStatus
func (b *Binanceus) GetAssetFeesAndWalletStatus(ctx context.Context) (AssetWalletList, error) {
	params := url.Values{}
	params.Set("timestamp", fmt.Sprint(time.Now().UnixMilli()))
	var response AssetWalletList
	return response, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary,
		http.MethodGet, assetFeeAndWalletStatus,
		params, spotDefaultRate, &response)
}

// WithdrawCrypto method to withdraw crypto
func (b *Binanceus) WithdrawCrypto(ctx context.Context, arg WithdrawalRequestParam) (string, error) {
	params := url.Values{}
	params.Set("timestamp", fmt.Sprint(time.Now().UnixMilli()))
	if arg.Coin == "" {
		return "", errors.New("Missing Required argument,Coin")
	}
	params.Set("coin", arg.Coin)
	if arg.Network == "" {
		return "", errors.New("Missing Required argument,Network")
	}
	params.Set("network", arg.Network)
	if arg.WithdrawOrderId != "" { /// Client ID for withdraw
		params.Set("withdrawOrderId", arg.WithdrawOrderId)
	}
	if arg.Address != "" {
		return "", errIncompleteArguments
	}
	params.Set("address", arg.Address)
	if arg.AddressTag != "" {
		params.Set("addressTag", arg.AddressTag)
	}
	if arg.Amount <= 0 {
		return "", errIncompleteArguments
	}
	params.Set("amount", fmt.Sprint(arg.Amount))
	if arg.RecvWindow > 2000 {
		params.Set("recvWindow", fmt.Sprint(arg.RecvWindow))
	}
	// --
	var response WithdrawalResponse
	var er = b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodPost, applyWithdrawal,
		params, spotDefaultRate, &response)
	if er != nil {
		return "", er
	}
	return response.ID, er
}

// WithdrawHistory gets the status of recent withdrawals
// status `param` used as string to prevent default value 0 (for int) interpreting as EmailSent status
func (b *Binanceus) WithdrawalHistory(ctx context.Context, c currency.Code, status string, startTime, endTime time.Time, offset, limit int) ([]WithdrawStatusResponse, error) {
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
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UTC().Unix(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UTC().Unix(), 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if limit != 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	var withdrawStatus []WithdrawStatusResponse
	if err := b.SendAuthHTTPRequest(ctx,
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

func (b *Binanceus) FiatWithdrawalhistory(ctx context.Context, arg FiatWithdrawalRequestParams) (FiatAssetsHistory, error) {
	var response FiatAssetsHistory
	params := url.Values{}
	if !(arg.EndTime.IsZero()) && !(arg.EndTime.Before(time.Now())) {
		params.Set("endTime", fmt.Sprint(arg.EndTime.UnixMilli()))
	}
	if !(arg.StartTime.IsZero()) && !(arg.StartTime.After(time.Now())) {
		params.Set("startTime", fmt.Sprint(arg.StartTime.UnixMilli()))
	}
	if arg.FiatCurrency != "" {
		params.Set("fiatCurrency", arg.FiatCurrency)
	}
	if arg.Offset > 0 {
		params.Set("offset", fmt.Sprint(arg.Offset))
	}
	if arg.PaymentChannel != "" {
		params.Set("paymentChannel", arg.PaymentChannel)
	}
	if arg.PaymentMethod != "" {
		params.Set("paymentMethod", arg.PaymentMethod)
	}
	params.Set("timestamp", fmt.Sprint(time.Now().UnixMilli()))
	return response, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, fiatWithdrawalhistory,
		params, spotDefaultRate, &response)
}

func (b *Binanceus) WithdrawFiat(ctx context.Context, arg WithdrawFiatRequestParams) (string, error) {
	params := url.Values{}
	timestamp := fmt.Sprint(time.Now().UnixMilli())
	params.Set("timestamp", timestamp)
	if arg.PaymentChannel != "" {
		params.Set("paymentChannel", arg.PaymentChannel)
	}
	if arg.PaymentMethod != "" {
		params.Set("paymentMethod", arg.PaymentMethod)
	}
	if arg.PaymentAccount != "" {
		return "", errors.New("error: missing payment account")
	}
	if arg.FiatCurrency != "" {
		params.Set("fiatCurrency", arg.FiatCurrency)
	}
	if arg.Amount <= 0 {
		return "", errors.New("error: invalid withdrawal amount")
	}
	type response struct {
		OrderID string `json:"orderId"`
	}
	var resp response
	return resp.OrderID, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary,
		http.MethodPost, withdrawFiat,
		params, spotDefaultRate, &resp,
	)
}

/*
	Deposits
	Get Crypto Deposit Address
*/

// GetDepositAddressForCurrency retrieves the wallet address for a given currency
func (b *Binanceus) GetDepositAddressForCurrency(ctx context.Context, currency, chain string) (*DepositAddress, error) {
	params := url.Values{}
	params.Set("coin", currency)
	if chain != "" {
		params.Set("network", chain)
	}
	params.Set("recvWindow", "10000")
	var d DepositAddress
	return &d,
		b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, depositAddress, params, spotDefaultRate, &d)
}

// DepositHistory returns the deposit history based on the supplied params
// status `param` used as string to prevent default value 0 (for int) interpreting as EmailSent status
func (b *Binanceus) DepositHistory(ctx context.Context, c currency.Code, status string, startTime, endTime time.Time, offset, limit int) ([]DepositHistory, error) {
	var response []DepositHistory

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

	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UTC().Unix(), 10))
	}

	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UTC().Unix(), 10))
	}

	if offset != 0 {
		params.Set("offset", strconv.Itoa(offset))
	}

	if limit != 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	if err := b.SendAuthHTTPRequest(ctx,
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

func (b *Binanceus) FiatDepositHistory(ctx context.Context, arg FiatWithdrawalRequestParams) (FiatAssetsHistory, error) {
	params := url.Values{}
	if !(arg.EndTime.IsZero()) && !(arg.EndTime.Before(time.Now())) {
		params.Set("endTime", fmt.Sprint(arg.EndTime.UnixMilli()))
	}
	if !(arg.StartTime.IsZero()) && !(arg.StartTime.After(time.Now())) {
		params.Set("startTime", fmt.Sprint(arg.StartTime.UnixMilli()))
	}
	if arg.FiatCurrency != "" {
		params.Set("fiatCurrency", arg.FiatCurrency)
	}
	if arg.Offset > 0 {
		params.Set("offset", fmt.Sprint(arg.Offset))
	}
	if arg.PaymentChannel != "" {
		params.Set("paymentChannel", arg.PaymentChannel)
	}
	if arg.PaymentMethod != "" {
		params.Set("paymentMethod", arg.PaymentMethod)
	}
	params.Set("timestamp", fmt.Sprint(time.Now().UnixMilli()))
	var response FiatAssetsHistory
	return response, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary, http.MethodGet,
		fiatDepositHistory, params, spotDefaultRate, &response)
}

// SendHTTPRequest  ...
func (b *Binanceus) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result interface{}) error {
	endpointPath, err := b.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpointPath + path,
		Result:        result,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	}
	return b.SendPayload(ctx, f, func() (*request.Item, error) {
		return item, nil
	})
}

func (b *Binanceus) SendAPIKeyHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result interface{}) error {
	endpointPath, err := b.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return err
	}

	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = creds.Key
	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpointPath + path,
		Headers:       headers,
		Result:        result,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording}

	return b.SendPayload(ctx, f, func() (*request.Item, error) {
		return item, nil
	})
}

// SendAuthHTTPRequest sends an authenticated HTTP request
func (b *Binanceus) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, f request.EndpointLimit, result interface{}) error {
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpointPath, err := b.API.Endpoints.GetURL(ePath)
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
	err = b.SendPayload(ctx, f, func() (*request.Item, error) {
		fullPath := endpointPath + path
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		signature := params.Encode()
		var hmacSigned []byte
		hmacSigned, err = crypto.GetHMAC(crypto.HashSHA256,
			[]byte(signature),
			[]byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		hmacSignedStr := crypto.HexEncodeToString(hmacSigned)
		headers := make(map[string]string)
		headers["X-MBX-APIKEY"] = creds.Key
		fullPath = common.EncodeURLValues(fullPath, params)
		fullPath += "&signature=" + hmacSignedStr
		return &request.Item{
			Method:        method,
			Path:          fullPath,
			Headers:       headers,
			Result:        &interim,
			AuthRequest:   true,
			Verbose:       b.Verbose,
			HTTPDebugging: b.HTTPDebugging,
			HTTPRecording: b.HTTPRecording}, nil
	})
	if err != nil {
		return err
	}
	errCap := struct {
		Success bool   `json:"success"`
		Message string `json:"msg"`
		Code    int64  `json:"code"`
	}{}
	// println(string([]byte(interim)))
	if err := json.Unmarshal(interim, &errCap); err == nil {
		if !errCap.Success && errCap.Message != "" && errCap.Code != 200 {
			return errors.New(errCap.Message)
		}
	}
	return json.Unmarshal(interim, result)
}
