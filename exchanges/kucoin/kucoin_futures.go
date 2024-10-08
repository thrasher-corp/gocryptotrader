package kucoin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	kucoinFuturesAPIURL = "https://api-futures.kucoin.com/api"
	kucoinWebsocketURL  = "wss://ws-api.kucoin.com/endpoint"

	kucoinFuturesOrder     = "/v1/orders"
	kucoinFuturesStopOrder = "/v1/stopOrders"
)

// GetFuturesOpenContracts gets all open futures contract with its details
func (ku *Kucoin) GetFuturesOpenContracts(ctx context.Context) ([]Contract, error) {
	var resp []Contract
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresOpenContractsEPL, "/v1/contracts/active", &resp)
}

// GetFuturesContract get contract details
func (ku *Kucoin) GetFuturesContract(ctx context.Context, symbol string) (*Contract, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *Contract
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresContractEPL, "/v1/contracts/"+symbol, &resp)
}

// GetFuturesTicker get real time ticker
func (ku *Kucoin) GetFuturesTicker(ctx context.Context, symbol string) (*FuturesTicker, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *FuturesTicker
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresTickerEPL, "/v1/ticker?symbol="+symbol, &resp)
}

// GetFuturesTickers does n * REST requests based on enabled pairs of the futures asset type
func (ku *Kucoin) GetFuturesTickers(ctx context.Context) ([]*ticker.Price, error) {
	pairs, err := ku.GetEnabledPairs(asset.Futures)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	tickersC := make(chan *ticker.Price, len(pairs))
	errC := make(chan error, len(pairs))

	for i := range pairs {
		var p currency.Pair
		if p, err = ku.FormatExchangeCurrency(pairs[i], asset.Futures); err != nil {
			errC <- err
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()

			if tick, err2 := ku.GetFuturesTicker(ctx, p.String()); err2 != nil {
				errC <- err2
			} else {
				tickersC <- &ticker.Price{
					Last:         tick.Price.Float64(),
					Bid:          tick.BestBidPrice.Float64(),
					Ask:          tick.BestAskPrice.Float64(),
					BidSize:      tick.BestBidSize,
					AskSize:      tick.BestAskSize,
					Volume:       tick.Size,
					Pair:         p,
					LastUpdated:  tick.FilledTime.Time(),
					ExchangeName: ku.Name,
					AssetType:    asset.Futures,
				}
			}
		}()
	}

	wg.Wait()
	close(tickersC)
	close(errC)
	var errs error
	for err := range errC {
		errs = common.AppendError(errs, err)
	}
	if errs != nil {
		return nil, errs
	}

	tickers := make([]*ticker.Price, 0, len(pairs))
	for tick := range tickersC {
		tickers = append(tickers, tick)
	}
	return tickers, nil
}

// GetFuturesOrderbook gets full orderbook for a specified symbol
func (ku *Kucoin) GetFuturesOrderbook(ctx context.Context, symbol string) (*Orderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var o futuresOrderbookResponse
	err := ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresOrderbookEPL, common.EncodeURLValues("/v1/level2/snapshot", params), &o)
	if err != nil {
		return nil, err
	}
	return constructFuturesOrderbook(&o), nil
}

// GetFuturesPartOrderbook20 gets orderbook for a specified symbol with depth 20
func (ku *Kucoin) GetFuturesPartOrderbook20(ctx context.Context, symbol string) (*Orderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var o futuresOrderbookResponse
	err := ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresPartOrderbookDepth20EPL, common.EncodeURLValues("/v1/level2/depth20", params), &o)
	if err != nil {
		return nil, err
	}
	return constructFuturesOrderbook(&o), nil
}

// GetFuturesPartOrderbook100 gets orderbook for a specified symbol with depth 100
func (ku *Kucoin) GetFuturesPartOrderbook100(ctx context.Context, symbol string) (*Orderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var o futuresOrderbookResponse
	err := ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresPartOrderbookDepth100EPL, common.EncodeURLValues("/v1/level2/depth100", params), &o)
	if err != nil {
		return nil, err
	}
	return constructFuturesOrderbook(&o), nil
}

// GetFuturesTradeHistory get last 100 trades for symbol
func (ku *Kucoin) GetFuturesTradeHistory(ctx context.Context, symbol string) ([]FuturesTrade, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp []FuturesTrade
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresTransactionHistoryEPL, "/v1/trade/history?symbol="+symbol, &resp)
}

// GetFuturesInterestRate get interest rate
func (ku *Kucoin) GetFuturesInterestRate(ctx context.Context, symbol string, startAt, endAt time.Time, reverse, forward bool, offset, maxCount int64) (*FundingInterestRateResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	params.Set("reverse", strconv.FormatBool(reverse))
	params.Set("forward", strconv.FormatBool(forward))
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount != 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	var resp *FundingInterestRateResponse
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresInterestRateEPL, common.EncodeURLValues("/v1/interest/query", params), &resp)
}

// GetFuturesIndexList retrieves futures index information for a symbol
func (ku *Kucoin) GetFuturesIndexList(ctx context.Context, symbol string, startAt, endAt time.Time, reverse, forward bool, offset, maxCount int64) (*FuturesIndexResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	params.Set("reverse", strconv.FormatBool(reverse))
	params.Set("forward", strconv.FormatBool(forward))
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount != 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	var resp *FuturesIndexResponse
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresIndexListEPL, common.EncodeURLValues("/v1/index/query", params), &resp)
}

// GetFuturesCurrentMarkPrice get current mark price
func (ku *Kucoin) GetFuturesCurrentMarkPrice(ctx context.Context, symbol string) (*FuturesMarkPrice, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *FuturesMarkPrice
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresCurrentMarkPriceEPL, "/v1/mark-price/"+symbol+"/current", &resp)
}

// GetFuturesPremiumIndex get premium index
func (ku *Kucoin) GetFuturesPremiumIndex(ctx context.Context, symbol string, startAt, endAt time.Time, reverse, forward bool, offset, maxCount int64) (*FuturesInterestRateResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	params.Set("reverse", strconv.FormatBool(reverse))
	params.Set("forward", strconv.FormatBool(forward))
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount != 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	var resp *FuturesInterestRateResponse
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresPremiumIndexEPL, common.EncodeURLValues("/v1/premium/query", params), &resp)
}

// Get24HourFuturesTransactionVolume retrieves a 24 hour transaction volume
func (ku *Kucoin) Get24HourFuturesTransactionVolume(ctx context.Context) (*TransactionVolume, error) {
	var resp *TransactionVolume
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresTransactionVolumeEPL, http.MethodGet, "/v1/trade-statistics", nil, &resp)
}

// GetFuturesCurrentFundingRate get current funding rate
func (ku *Kucoin) GetFuturesCurrentFundingRate(ctx context.Context, symbol string) (*FuturesFundingRate, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *FuturesFundingRate
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresCurrentFundingRateEPL, "/v1/funding-rate/"+symbol+"/current", &resp)
}

// GetPublicFundingRate query the funding rate at each settlement time point within a certain time range of the corresponding contract
func (ku *Kucoin) GetPublicFundingRate(ctx context.Context, symbol string, from, to time.Time) ([]FundingHistoryItem, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	err := common.StartEndTimeCheck(from, to)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	var resp []FundingHistoryItem
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresPublicFundingRateEPL, common.EncodeURLValues("/v1/contract/funding-rates", params), &resp)
}

// GetFuturesServerTime get server time
func (ku *Kucoin) GetFuturesServerTime(ctx context.Context) (time.Time, error) {
	resp := struct {
		Data types.Time `json:"data"`
		Error
	}{}
	err := ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresServerTimeEPL, "/v1/timestamp", &resp)
	if err != nil {
		return time.Time{}, err
	}
	return resp.Data.Time(), nil
}

// GetFuturesServiceStatus get service status
func (ku *Kucoin) GetFuturesServiceStatus(ctx context.Context) (*FuturesServiceStatus, error) {
	var resp *FuturesServiceStatus
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresServiceStatusEPL, "/v1/status", &resp)
}

// GetFuturesKline get contract's kline data
func (ku *Kucoin) GetFuturesKline(ctx context.Context, granularity int64, symbol string, from, to time.Time) ([]FuturesKline, error) {
	if granularity == 0 {
		return nil, kline.ErrInvalidInterval
	}
	if !slices.Contains(validGranularity, strconv.FormatInt(granularity, 10)) {
		return nil, fmt.Errorf("%w, invalid granularity", kline.ErrUnsupportedInterval)
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	// The granularity (granularity parameter of K-line) represents the number of minutes, the available granularity scope is: 1,5,15,30,60,120,240,480,720,1440,10080. Requests beyond the above range will be rejected
	params.Set("granularity", strconv.FormatInt(granularity, 10))
	params.Set("symbol", symbol)
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	var resp [][6]float64
	err := ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresKlineEPL, common.EncodeURLValues("/v1/kline/query", params), &resp)
	if err != nil {
		return nil, err
	}
	kline := make([]FuturesKline, len(resp))
	for i := range resp {
		kline[i] = FuturesKline{
			StartTime: time.UnixMilli(int64(resp[i][0])),
			Open:      resp[i][1],
			High:      resp[i][2],
			Low:       resp[i][3],
			Close:     resp[i][4],
			Volume:    resp[i][5],
		}
	}
	return kline, nil
}

// PostFuturesOrder used to place two types of futures orders: limit and market
func (ku *Kucoin) PostFuturesOrder(ctx context.Context, arg *FuturesOrderParam) (string, error) {
	err := ku.FillFuturesPostOrderArgumentFilter(arg)
	if err != nil {
		return "", err
	}
	resp := struct {
		OrderID string `json:"orderId"`
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresPlaceOrderEPL, http.MethodPost, kucoinFuturesOrder, &arg, &resp)
}

// PostFuturesOrderTest a test endpoint to place a single futures order
func (ku *Kucoin) PostFuturesOrderTest(ctx context.Context, arg *FuturesOrderParam) (string, error) {
	err := ku.FillFuturesPostOrderArgumentFilter(arg)
	if err != nil {
		return "", err
	}
	resp := struct {
		OrderID string `json:"orderId"`
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresPlaceOrderEPL, http.MethodPost, kucoinFuturesOrder+"/test", &arg, &resp)
}

// FillFuturesPostOrderArgumentFilter verifies futures order request parameters
func (ku *Kucoin) FillFuturesPostOrderArgumentFilter(arg *FuturesOrderParam) error {
	if *arg == (FuturesOrderParam{}) {
		return common.ErrNilPointer
	}
	if arg.Leverage <= 0 {
		return errInvalidLeverage
	}
	if arg.ClientOrderID == "" {
		return order.ErrClientOrderIDMustBeSet
	}
	if arg.Side == "" {
		return fmt.Errorf("%w, empty order side", order.ErrSideIsInvalid)
	}
	if arg.Symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if arg.Stop != "" {
		if arg.StopPriceType == "" {
			return errInvalidStopPriceType
		}
		if arg.StopPrice <= 0 {
			return fmt.Errorf("%w, stopPrice is required", order.ErrPriceBelowMin)
		}
	}
	switch arg.OrderType {
	case "limit", "":
		if arg.Price <= 0 {
			return fmt.Errorf("%w %f", order.ErrPriceBelowMin, arg.Price)
		}
		if arg.Size <= 0 {
			return fmt.Errorf("%w, must be non-zero positive value", order.ErrAmountBelowMin)
		}
		if arg.VisibleSize < 0 {
			return fmt.Errorf("%w, visible size must be non-zero positive value", order.ErrAmountBelowMin)
		}
	case "market":
		if arg.Size <= 0 {
			return fmt.Errorf("%w, market size must be > 0", order.ErrAmountBelowMin)
		}
	default:
		return fmt.Errorf("%w, order type= %s", order.ErrTypeIsInvalid, arg.OrderType)
	}
	return nil
}

// PlaceMultipleFuturesOrders used to place multiple futures orders
// The maximum limit orders for a single contract is 100 per account, and the maximum stop orders for a single contract is 50 per account
func (ku *Kucoin) PlaceMultipleFuturesOrders(ctx context.Context, args []FuturesOrderParam) ([]FuturesOrderRespItem, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%w, not order to place", common.ErrEmptyParams)
	}
	var err error
	for x := range args {
		err = ku.FillFuturesPostOrderArgumentFilter(&args[x])
		if err != nil {
			return nil, err
		}
	}
	var resp []FuturesOrderRespItem
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, multipleFuturesOrdersEPL, http.MethodPost, "/v1/orders/multi", args, &resp)
}

// CancelFuturesOrderByOrderID used to cancel single order previously placed including a stop order
func (ku *Kucoin) CancelFuturesOrderByOrderID(ctx context.Context, orderID string) ([]string, error) {
	return ku.cancelFuturesOrderByID(ctx, orderID, "/v1/orders/", "")
}

// CancelFuturesOrderByClientOrderID cancels a futures order by using client order ID
func (ku *Kucoin) CancelFuturesOrderByClientOrderID(ctx context.Context, symbol, clientOrderID string) ([]string, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	return ku.cancelFuturesOrderByID(ctx, clientOrderID, "/v1/orders/client-order/", symbol)
}

func (ku *Kucoin) cancelFuturesOrderByID(ctx context.Context, id, path, symbol string) ([]string, error) {
	if id == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	resp := struct {
		CancelledOrderIDs []string `json:"cancelledOrderIds"`
	}{}
	return resp.CancelledOrderIDs, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresCancelAnOrderEPL, http.MethodDelete, common.EncodeURLValues(path+id, params), nil, &resp)
}

// CancelMultipleFuturesLimitOrders used to cancel all futures order excluding stop orders
func (ku *Kucoin) CancelMultipleFuturesLimitOrders(ctx context.Context, symbol string) ([]string, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	resp := struct {
		CancelledOrderIDs []string `json:"cancelledOrderIds"`
	}{}
	return resp.CancelledOrderIDs, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresLimitOrderMassCancelationEPL, http.MethodDelete, common.EncodeURLValues(kucoinFuturesOrder, params), nil, &resp)
}

// CancelAllFuturesStopOrders used to cancel all untriggered stop orders
func (ku *Kucoin) CancelAllFuturesStopOrders(ctx context.Context, symbol string) ([]string, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	resp := struct {
		CancelledOrderIDs []string `json:"cancelledOrderIds"`
	}{}
	return resp.CancelledOrderIDs, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresCancelMultipleLimitOrdersEPL, http.MethodDelete, common.EncodeURLValues(kucoinFuturesStopOrder, params), nil, &resp)
}

// GetFuturesOrders gets the user current futures order list
func (ku *Kucoin) GetFuturesOrders(ctx context.Context, status, symbol, side, orderType string, startAt, endAt time.Time) (*FutureOrdersResponse, error) {
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	var resp *FutureOrdersResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresRetrieveOrderListEPL, http.MethodGet, common.EncodeURLValues(kucoinFuturesOrder, params), nil, &resp)
}

// GetUntriggeredFuturesStopOrders gets the untriggered stop orders list
func (ku *Kucoin) GetUntriggeredFuturesStopOrders(ctx context.Context, symbol, side, orderType string, startAt, endAt time.Time) (*FutureOrdersResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	var resp *FutureOrdersResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, cancelUntriggeredFuturesStopOrdersEPL, http.MethodGet, common.EncodeURLValues(kucoinFuturesStopOrder, params), nil, &resp)
}

// GetFuturesRecentCompletedOrders gets list of recent 1000 orders in the last 24 hours
func (ku *Kucoin) GetFuturesRecentCompletedOrders(ctx context.Context, symbol string) ([]FuturesOrder, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []FuturesOrder
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresRecentCompletedOrdersEPL, http.MethodGet, common.EncodeURLValues("/v1/recentDoneOrders", params), nil, &resp)
}

// GetFuturesOrderDetails gets single order details by order ID
func (ku *Kucoin) GetFuturesOrderDetails(ctx context.Context, orderID, clientOrderID string) (*FuturesOrder, error) {
	path := "/v1/orders/"
	if orderID == "" && clientOrderID == "" {
		return nil, fmt.Errorf("%w either client order ID or order id required", order.ErrOrderIDNotSet)
	}
	if orderID == "" {
		path = "/v1/orders/byClientOid"
	}
	params := url.Values{}
	if clientOrderID != "" {
		params.Set("clientOid", clientOrderID)
	}
	var resp *FuturesOrder
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresOrdersByIDEPL, http.MethodGet, common.EncodeURLValues(path+orderID, params), nil, &resp)
}

// GetFuturesOrderDetailsByClientOrderID gets single order details by client ID
func (ku *Kucoin) GetFuturesOrderDetailsByClientOrderID(ctx context.Context, clientOrderID string) (*FuturesOrder, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	var resp *FuturesOrder
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresOrderDetailsByClientOrderIDEPL, http.MethodGet, "/v1/orders/byClientOid?clientOid="+clientOrderID, nil, &resp)
}

// GetFuturesFills gets list of recent fills
func (ku *Kucoin) GetFuturesFills(ctx context.Context, orderID, symbol, side, orderType string, startAt, endAt time.Time) (*FutureFillsResponse, error) {
	params := url.Values{}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	var resp *FutureFillsResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresRetrieveFillsEPL, http.MethodGet, common.EncodeURLValues("/v1/fills", params), nil, &resp)
}

// GetFuturesRecentFills gets list of 1000 recent fills in the last 24 hrs
func (ku *Kucoin) GetFuturesRecentFills(ctx context.Context) ([]FuturesFill, error) {
	var resp []FuturesFill
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresRecentFillsEPL, http.MethodGet, "/v1/recentFills", nil, &resp)
}

// GetFuturesOpenOrderStats gets the total number and value of the all your active orders
func (ku *Kucoin) GetFuturesOpenOrderStats(ctx context.Context, symbol string) (*FuturesOpenOrderStats, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *FuturesOpenOrderStats
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresOpenOrderStatsEPL, http.MethodGet, "/v1/openOrderStatistics?symbol="+symbol, nil, &resp)
}

// GetFuturesPosition gets the position details of a specified position
func (ku *Kucoin) GetFuturesPosition(ctx context.Context, symbol string) (*FuturesPosition, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *FuturesPosition
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresPositionEPL, http.MethodGet, "/v1/position?symbol="+symbol, nil, &resp)
}

// GetFuturesPositionList gets the list of position with details
func (ku *Kucoin) GetFuturesPositionList(ctx context.Context) ([]FuturesPosition, error) {
	var resp []FuturesPosition
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresPositionListEPL, http.MethodGet, "/v1/positions", nil, &resp)
}

// SetAutoDepositMargin enable/disable of auto-deposit margin
func (ku *Kucoin) SetAutoDepositMargin(ctx context.Context, symbol string, status bool) (bool, error) {
	if symbol == "" {
		return false, currency.ErrSymbolStringEmpty
	}
	params := make(map[string]interface{})
	params["symbol"] = symbol
	params["status"] = status
	var resp bool
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, setAutoDepositMarginEPL, http.MethodPost, "/v1/position/margin/auto-deposit-status", params, &resp)
}

// GetMaxWithdrawMargin query the maximum amount of margin that the current position supports withdrawal
func (ku *Kucoin) GetMaxWithdrawMargin(ctx context.Context, symbol string) (float64, error) {
	if symbol == "" {
		return 0, currency.ErrSymbolStringEmpty
	}
	var resp types.Number
	return resp.Float64(), ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, maxWithdrawMarginEPL, http.MethodGet, "/v1/margin/maxWithdrawMargin?symbol="+symbol, nil, &resp)
}

// RemoveMarginManually removes a margin manually
func (ku *Kucoin) RemoveMarginManually(ctx context.Context, arg *WithdrawMarginResponse) (*MarginRemovingResponse, error) {
	if *arg == (WithdrawMarginResponse{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.WithdrawAmount <= 0 {
		return nil, fmt.Errorf("%w, withdrawAmount must be greater than 0", order.ErrAmountBelowMin)
	}
	var resp *MarginRemovingResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, removeMarginManuallyEPL, http.MethodPost, "/v1/margin/withdrawMargin", arg, &resp)
}

// AddMargin is used to add margin manually
func (ku *Kucoin) AddMargin(ctx context.Context, symbol, uniqueID string, margin float64) (*FuturesPosition, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := make(map[string]interface{})
	params["symbol"] = symbol
	if uniqueID == "" {
		return nil, errors.New("uniqueID cannot be empty")
	}
	params["bizNo"] = uniqueID
	if margin <= 0 {
		return nil, errors.New("margin cannot be zero or negative")
	}
	params["margin"] = strconv.FormatFloat(margin, 'f', -1, 64)
	var resp *FuturesPosition
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresAddMarginManuallyEPL, http.MethodPost, "/v1/position/margin/deposit-margin", params, &resp)
}

// GetFuturesRiskLimitLevel gets information about risk limit level of a specific contract
func (ku *Kucoin) GetFuturesRiskLimitLevel(ctx context.Context, symbol string) ([]FuturesRiskLimitLevel, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp []FuturesRiskLimitLevel
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresRiskLimitLevelEPL, http.MethodGet, "/v1/contracts/risk-limit/"+symbol, nil, &resp)
}

// FuturesUpdateRiskLmitLevel is used to adjustment the risk limit level
func (ku *Kucoin) FuturesUpdateRiskLmitLevel(ctx context.Context, symbol string, level int64) (bool, error) {
	if symbol == "" {
		return false, currency.ErrSymbolStringEmpty
	}
	params := make(map[string]interface{})
	params["symbol"] = symbol
	params["level"] = strconv.FormatInt(level, 10)
	var resp bool
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresUpdateRiskLimitLevelEPL, http.MethodPost, "/v1/position/risk-limit-level/change", params, &resp)
}

// GetFuturesFundingHistory gets information about funding history
func (ku *Kucoin) GetFuturesFundingHistory(ctx context.Context, symbol string, offset, maxCount int64, reverse, forward bool, startAt, endAt time.Time) (*FuturesFundingHistoryResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	params.Set("reverse", strconv.FormatBool(reverse))
	params.Set("forward", strconv.FormatBool(forward))
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount != 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	var resp *FuturesFundingHistoryResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresFundingHistoryEPL, http.MethodGet, common.EncodeURLValues("/v1/funding-history", params), nil, &resp)
}

// GetFuturesAccountOverview gets future account overview
func (ku *Kucoin) GetFuturesAccountOverview(ctx context.Context, ccy string) (*FuturesAccount, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	var resp *FuturesAccount
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresAccountOverviewEPL, http.MethodGet, common.EncodeURLValues("/v1/account-overview", params), nil, &resp)
}

// GetFuturesTransactionHistory gets future transaction history
func (ku *Kucoin) GetFuturesTransactionHistory(ctx context.Context, ccy currency.Code, txType string, offset, maxCount int64, forward bool, startAt, endAt time.Time) (*FuturesTransactionHistoryResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if txType != "" {
		params.Set("type", txType)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	params.Set("forward", strconv.FormatBool(forward))
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount != 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	var resp *FuturesTransactionHistoryResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresRetrieveTransactionHistoryEPL, http.MethodGet, common.EncodeURLValues("/v1/transaction-history", params), nil, &resp)
}

// CreateFuturesSubAccountAPIKey is used to create Futures APIs for sub-accounts
func (ku *Kucoin) CreateFuturesSubAccountAPIKey(ctx context.Context, ipWhitelist, passphrase, permission, remark, subName string) (*APIKeyDetail, error) {
	if remark == "" {
		return nil, errRemarkIsRequired
	}
	if subName == "" {
		return nil, errInvalidSubAccountName
	}
	if passphrase == "" {
		return nil, errInvalidPassPhraseInstance
	}
	params := make(map[string]interface{})
	params["passphrase"] = passphrase
	params["remark"] = remark
	params["subName"] = subName
	if ipWhitelist != "" {
		params["ipWhitelist"] = ipWhitelist
	}
	if permission != "" {
		params["permission"] = permission
	}
	var resp *APIKeyDetail
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, createSubAccountAPIKeyEPL, http.MethodPost, "/v1/sub/api-key", params, &resp)
}

// TransferFuturesFundsToMainAccount helps in transferring funds from futures to main/trade account
func (ku *Kucoin) TransferFuturesFundsToMainAccount(ctx context.Context, amount float64, ccy currency.Code, recAccountType string) (*TransferRes, error) {
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if recAccountType == "" {
		return nil, fmt.Errorf("%w, invalid receive account type", errAccountTypeMissing)
	}
	params := make(map[string]interface{})
	params["amount"] = amount
	params["currency"] = ccy.String()
	params["recAccountType"] = recAccountType
	var resp *TransferRes
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, transferOutToMainEPL, http.MethodPost, "/v3/transfer-out", params, &resp)
}

// TransferFundsToFuturesAccount helps in transferring funds from payee account to futures account
func (ku *Kucoin) TransferFundsToFuturesAccount(ctx context.Context, amount float64, ccy currency.Code, payAccountType string) error {
	if amount <= 0 {
		return order.ErrAmountBelowMin
	}
	if ccy.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if payAccountType == "" {
		return fmt.Errorf("%w, payAccountType cannot be empty", errAccountTypeMissing)
	}
	params := make(map[string]interface{})
	params["amount"] = amount
	params["currency"] = ccy.String()
	params["payAccountType"] = payAccountType
	resp := struct {
		Error
	}{}
	return ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, transferFundToFuturesAccountEPL, http.MethodPost, "/v1/transfer-in", params, &resp)
}

// GetFuturesTransferOutList gets list of transfer out
func (ku *Kucoin) GetFuturesTransferOutList(ctx context.Context, ccy currency.Code, status string, startAt, endAt time.Time) (*TransferListsResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if status != "" {
		params.Set("status", status)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	var resp *TransferListsResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresTransferOutListEPL, http.MethodGet, common.EncodeURLValues("/v1/transfer-list", params), nil, &resp)
}

func processFuturesOB(ob [][2]float64) []orderbook.Tranche {
	o := make([]orderbook.Tranche, len(ob))
	for x := range ob {
		o[x] = orderbook.Tranche{
			Price:  ob[x][0],
			Amount: ob[x][1],
		}
	}
	return o
}

func constructFuturesOrderbook(o *futuresOrderbookResponse) *Orderbook {
	return &Orderbook{
		Bids:     processFuturesOB(o.Bids),
		Asks:     processFuturesOB(o.Asks),
		Sequence: o.Sequence,
		Time:     o.Time.Time(),
	}
}

// GetFuturesTradingPairsActualFees retrieves the actual fee rate of the trading pair. The fee rate of your sub-account is the same as that of the master account
func (ku *Kucoin) GetFuturesTradingPairsActualFees(ctx context.Context, symbol string) (*TradingPairFee, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *TradingPairFee
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresTradingPairFeeEPL, http.MethodGet, common.EncodeURLValues("/v1/trade-fees", params), nil, &resp)
}

// GetPositionHistory query position history information records
func (ku *Kucoin) GetPositionHistory(ctx context.Context, symbol string, from, to time.Time, limit, pageID int64) (*FuturesPositionHistory, error) {
	params := url.Values{}
	if !from.IsZero() && !to.IsZero() {
		err := common.StartEndTimeCheck(from, to)
		if err != nil {
			return nil, err
		}
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pageID > 0 {
		params.Set("pageId", strconv.FormatInt(pageID, 10))
	}
	var resp *FuturesPositionHistory
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresPositionHistoryEPL, http.MethodGet, common.EncodeURLValues("/v1/history-positions", params), nil, &resp)
}

// GetMaximumOpenPositionSize retrieves a maximum open position size
func (ku *Kucoin) GetMaximumOpenPositionSize(ctx context.Context, symbol string, price float64, leverage int64) (*FuturesMaxOpenPositionSize, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if price <= 0 {
		return nil, order.ErrPriceBelowMin
	}
	if leverage <= 0 {
		return nil, fmt.Errorf("%w, leverage is required", errInvalidLeverage)
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	params.Set("leverage", strconv.FormatInt(leverage, 10))
	var resp *FuturesMaxOpenPositionSize
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresMaxOpenPositionsSizeEPL, http.MethodGet, common.EncodeURLValues("/v2/getMaxOpenSize", params), nil, &resp)
}

// GetLatestTickersForAllContracts retrieves all futures instruments ticker information
func (ku *Kucoin) GetLatestTickersForAllContracts(ctx context.Context) ([]WsFuturesTicker, error) {
	var resp []WsFuturesTicker
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresAllTickersInfoEPL, http.MethodGet, "/v1/allTickers", nil, &resp)
}
