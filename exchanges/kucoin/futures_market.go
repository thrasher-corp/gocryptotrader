package kucoin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var errOrderbookSizeRequired = errors.New("orderbook size is required")

// GetFuturesContract returns a tradable contract's specification and market state (Get Symbol)
func (e *Exchange) GetFuturesContract(ctx context.Context, symbol string) (*FuturesContractResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *FuturesContractResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, getFuturesContractEPL, "/v1/contracts/"+symbol, &resp)
}

// GetFuturesOpenContracts returns every tradable contract's specification and market state (Get All Symbols)
func (e *Exchange) GetFuturesOpenContracts(ctx context.Context) ([]Contract, error) {
	var resp []Contract
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, getFuturesOpenContractsEPL, "/v1/contracts/active", &resp)
}

// GetFuturesTicker returns a contract's last fill and best bid and ask (Get Ticker)
func (e *Exchange) GetFuturesTicker(ctx context.Context, symbol string) (*FuturesTickerResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesTickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, getFuturesTickerEPL, common.EncodeURLValues("/v1/ticker", params), &resp)
}

// GetLatestTickersForAllContracts returns every contract's last fill and best bid and ask (Get All Tickers)
func (e *Exchange) GetLatestTickersForAllContracts(ctx context.Context) ([]FuturesTicker, error) {
	var resp []FuturesTicker
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, getLatestTickersForAllContractsEPL, "/v1/allTickers", &resp)
}

// GetFuturesOrderbook returns a contract's full order book, aggregated by price (Get Full OrderBook)
func (e *Exchange) GetFuturesOrderbook(ctx context.Context, symbol string) (*FuturesOrderbookResponse, error) {
	// The docs mark symbol optional, but the server rejects a request without one
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesOrderbookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, getFuturesOrderbookEPL, common.EncodeURLValues("/v1/level2/snapshot", params), &resp)
}

// GetFuturesPartOrderbook returns the best size levels of each side of a contract's order book, aggregated by price;
// the server offers sizes 20 and 100 (Get Part OrderBook)
func (e *Exchange) GetFuturesPartOrderbook(ctx context.Context, symbol string, size uint64) (*FuturesOrderbookResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if size == 0 {
		return nil, errOrderbookSizeRequired
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesOrderbookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, getFuturesPartOrderbookEPL, common.EncodeURLValues("/v1/level2/depth"+strconv.FormatUint(size, 10), params), &resp)
}

// GetFuturesTradeHistory returns a contract's last 100 fills (Get Trade History)
func (e *Exchange) GetFuturesTradeHistory(ctx context.Context, symbol string) ([]FuturesTrade, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp []FuturesTrade
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, getFuturesTradeHistoryEPL, common.EncodeURLValues("/v1/trade/history", params), &resp)
}

// GetFuturesKline returns up to 200 candles of a contract, index or premium index symbol; the docs say 500, but the
// server stops at 200 (Get Klines)
func (e *Exchange) GetFuturesKline(ctx context.Context, r *FuturesKlineRequest) ([]FuturesKline, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if r.Granularity == 0 {
		return nil, kline.ErrInvalidInterval
	}
	if r.Granularity < kline.OneMin || r.Granularity.Duration()%time.Minute != 0 {
		return nil, fmt.Errorf("%w %s: granularity is sent in whole minutes", kline.ErrUnsupportedInterval, r.Granularity)
	}
	if !r.From.IsZero() && !r.To.IsZero() {
		if err := common.StartEndTimeCheck(r.From, r.To); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", r.Symbol)
	params.Set("granularity", strconv.FormatInt(int64(r.Granularity.Duration()/time.Minute), 10))
	if !r.From.IsZero() {
		params.Set("from", strconv.FormatInt(r.From.UnixMilli(), 10))
	}
	if !r.To.IsZero() {
		params.Set("to", strconv.FormatInt(r.To.UnixMilli(), 10))
	}
	var resp []FuturesKline
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, getFuturesKlineEPL, common.EncodeURLValues("/v1/kline/query", params), &resp)
}

// GetFuturesCurrentMarkPrice returns a contract's current mark price (Get Mark Price)
func (e *Exchange) GetFuturesCurrentMarkPrice(ctx context.Context, symbol string) (*FuturesMarkPriceResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *FuturesMarkPriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, getFuturesCurrentMarkPriceEPL, "/v1/mark-price/"+symbol+"/current", &resp)
}

// GetFuturesIndexList returns a page of a spot index's prices and each exchange's contribution to them; the symbol is
// a contract's index symbol, such as .KXBTUSDT (Get Spot Index Price)
func (e *Exchange) GetFuturesIndexList(ctx context.Context, r *FuturesHistoryRequest) (*FuturesSpotIndexPriceResponse, error) {
	params, err := r.params()
	if err != nil {
		return nil, err
	}
	var resp *FuturesSpotIndexPriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, getFuturesIndexListEPL, common.EncodeURLValues("/v1/index/query", params), &resp)
}

// GetFuturesInterestRate returns a page of an interest rate index's values; the symbol is a contract's funding base or
// quote symbol, such as .XBTINT (Get Interest Rate Index)
func (e *Exchange) GetFuturesInterestRate(ctx context.Context, r *FuturesHistoryRequest) (*FuturesIndexHistoryResponse, error) {
	params, err := r.params()
	if err != nil {
		return nil, err
	}
	var resp *FuturesIndexHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, getFuturesInterestRateEPL, common.EncodeURLValues("/v1/interest/query", params), &resp)
}

// GetFuturesPremiumIndex returns a page of a premium index's values; the symbol is a contract's premiums symbol, such
// as .XBTUSDTMPI (Get Premium Index)
func (e *Exchange) GetFuturesPremiumIndex(ctx context.Context, r *FuturesHistoryRequest) (*FuturesIndexHistoryResponse, error) {
	params, err := r.params()
	if err != nil {
		return nil, err
	}
	var resp *FuturesIndexHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, getFuturesPremiumIndexEPL, common.EncodeURLValues("/v1/premium/query", params), &resp)
}

// params builds the query the history endpoints share
func (r *FuturesHistoryRequest) params() (url.Values, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !r.StartAt.IsZero() && !r.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(r.StartAt, r.EndAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", r.Symbol)
	if !r.StartAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(r.StartAt.UnixMilli(), 10))
	}
	if !r.EndAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(r.EndAt.UnixMilli(), 10))
	}
	if r.Reverse != nil {
		params.Set("reverse", strconv.FormatBool(*r.Reverse))
	}
	if r.Offset != 0 {
		params.Set("offset", strconv.FormatUint(r.Offset, 10))
	}
	if r.Forward != nil {
		params.Set("forward", strconv.FormatBool(*r.Forward))
	}
	if r.MaxCount != 0 {
		params.Set("maxCount", strconv.FormatUint(r.MaxCount, 10))
	}
	return params, nil
}

// Get24HourFuturesTransactionVolume returns the platform's futures turnover over the last 24 hours (Get 24hr stats)
func (e *Exchange) Get24HourFuturesTransactionVolume(ctx context.Context) (*Futures24HourStatsResponse, error) {
	var resp *Futures24HourStatsResponse
	// The docs list this endpoint as public, but the server rejects an unsigned request
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, get24HourFuturesTransactionVolumeEPL, http.MethodGet, "/v1/trade-statistics", nil, &resp)
}

// GetFuturesServerTime returns the futures API server's time (Get Server Time)
func (e *Exchange) GetFuturesServerTime(ctx context.Context) (time.Time, error) {
	var resp types.Time
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, getFuturesServerTimeEPL, "/v1/timestamp", &resp); err != nil {
		return time.Time{}, err
	}
	return resp.Time(), nil
}

// GetFuturesServiceStatus returns whether the futures service is open, under maintenance or only cancelling orders
// (Get Service Status)
func (e *Exchange) GetFuturesServiceStatus(ctx context.Context) (*FuturesServiceStatusResponse, error) {
	var resp *FuturesServiceStatusResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, getFuturesServiceStatusEPL, "/v1/status", &resp)
}

// errFuturesContractNotFound is returned when Get All Symbols does not list a contract Get All Tickers has a ticker for
var errFuturesContractNotFound = errors.New("futures contract not found")

// GetFuturesTickers returns a ticker for each enabled futures pair: its last fill and best bid and ask from Get All
// Tickers, and its 24 hour high, low and volumes, open interest and mark and index prices from Get All Symbols. A pair
// whose contract Get All Symbols does not list is left out and reported in the error
func (e *Exchange) GetFuturesTickers(ctx context.Context) ([]*ticker.Price, error) {
	if err := e.CurrencyPairs.IsAssetEnabled(asset.Futures); err != nil {
		return nil, err
	}
	tickers, err := e.GetLatestTickersForAllContracts(ctx)
	if err != nil {
		return nil, err
	}
	contracts, err := e.GetFuturesOpenContracts(ctx)
	if err != nil {
		return nil, err
	}
	contractsBySymbol := make(map[string]*Contract, len(contracts))
	for i := range contracts {
		contractsBySymbol[contracts[i].Symbol] = &contracts[i]
	}
	resp := make([]*ticker.Price, 0, len(tickers))
	var errs error
	for i := range tickers {
		pair, enabled, err := e.MatchSymbolCheckEnabled(tickers[i].Symbol, asset.Futures, false)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		if !enabled {
			continue
		}
		contract, ok := contractsBySymbol[tickers[i].Symbol]
		if !ok {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", errFuturesContractNotFound, tickers[i].Symbol))
			continue
		}
		baseVolume, quoteVolume := contractVolumes(contract)
		resp = append(resp, &ticker.Price{
			Last:         tickers[i].Price.Float64(),
			LastSize:     float64(tickers[i].Size),
			High:         contract.HighPrice,
			Low:          contract.LowPrice,
			Bid:          tickers[i].BestBidPrice.Float64(),
			BidSize:      float64(tickers[i].BestBidSize),
			Ask:          tickers[i].BestAskPrice.Float64(),
			AskSize:      float64(tickers[i].BestAskSize),
			BaseVolume:   baseVolume,
			QuoteVolume:  quoteVolume,
			OpenInterest: contract.OpenInterest.Float64(),
			MarkPrice:    contract.MarkPrice,
			IndexPrice:   contract.IndexPrice,
			Pair:         pair,
			ExchangeName: e.Name,
			AssetType:    asset.Futures,
			LastUpdated:  tickers[i].Timestamp.Time(),
		})
	}
	return resp, errs
}
