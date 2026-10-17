package bitfinex

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var (
	errInvalidBookPrecision        = errors.New("invalid book precision, want 0 to 4 for P0 to P4")
	errInvalidBookLength           = errors.New("invalid book length, want 1, 25, 100 or 250")
	errInvalidStatsKey             = errors.New("invalid stats key")
	errInvalidStatsSymbol          = errors.New("invalid stats symbol")
	errInvalidStatsSide            = errors.New("invalid stats side")
	errInvalidStatsPair            = errors.New("invalid stats pair")
	errCandlesPeriodRequired       = errors.New("funding currency candles need a period, or an aggregation with a period start and end")
	errCandlesPeriodUnsupported    = errors.New("candles period unsupported")
	errInvalidCandlesAggregation   = errors.New("invalid candles aggregation, want 10 or 30")
	errInvalidLeaderboardKey       = errors.New("invalid leaderboard key")
	errMarketAveragePeriodRequired = errors.New("funding market average price needs a period")
)

// statsSizes holds the one size Bitfinex keeps each stats key at; any other size returns nothing
var statsSizes = map[string]string{
	StatsKeyPositionSize:               "1m",
	StatsKeyFundingSize:                "1m",
	StatsKeyCreditsSize:                "1m",
	StatsKeyCreditsSizeByPair:          "1m",
	StatsKeyVolume1Day:                 "30m",
	StatsKeyVolume7Day:                 "30m",
	StatsKeyVolume30Day:                "30m",
	StatsKeyVolumeWeightedAveragePrice: "1d",
}

// candleTimeFrames holds the time frame Bitfinex names each candle interval by, case sensitively
var candleTimeFrames = map[kline.Interval]string{
	kline.OneMin:     "1m",
	kline.FiveMin:    "5m",
	kline.FifteenMin: "15m",
	kline.ThirtyMin:  "30m",
	kline.OneHour:    "1h",
	kline.ThreeHour:  "3h",
	kline.SixHour:    "6h",
	kline.TwelveHour: "12h",
	kline.OneDay:     "1D",
	kline.OneWeek:    "1W",
	kline.TwoWeek:    "14D",
	kline.OneMonth:   "1M",
}

// leaderboardTimeFrames holds the time frame Bitfinex names each leaderboard interval by; unlike the candles' week, the
// leaderboards' is lower case
var leaderboardTimeFrames = map[kline.Interval]string{
	kline.ThreeHour: "3h",
	kline.OneWeek:   "1w",
	kline.OneMonth:  "1M",
}

// GetPlatformStatus returns whether the platform is operative or in maintenance, during which orders can be neither
// placed nor cancelled
func (e *Exchange) GetPlatformStatus(ctx context.Context) (*PlatformStatusResponse, error) {
	var resp *PlatformStatusResponse
	if err := e.SendHTTPRequest(ctx, platformStatusEPL, http.MethodGet, "platform/status", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetTicker returns a trading pair's ticker, for a symbol such as tBTCUSD
func (e *Exchange) GetTicker(ctx context.Context, symbol string) (*Ticker, error) {
	if !isTradingSymbol(symbol) {
		return nil, errTradingSymbolRequired
	}
	var resp *Ticker
	if err := e.SendHTTPRequest(ctx, tickerEPL, http.MethodGet, "ticker/"+symbol, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetFundingTicker returns a funding currency's ticker, for a symbol such as fUSD
func (e *Exchange) GetFundingTicker(ctx context.Context, symbol string) (*FundingTicker, error) {
	if !isFundingSymbol(symbol) {
		return nil, errFundingSymbolRequired
	}
	var resp *FundingTicker
	if err := e.SendHTTPRequest(ctx, tickerEPL, http.MethodGet, "ticker/"+symbol, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetTickers returns the tickers of trading pair and funding currency symbols, such as tBTCUSD and fUSD, or of every
// pair and currency when symbols is empty
func (e *Exchange) GetTickers(ctx context.Context, symbols []string) (*TickersResponse, error) {
	params := url.Values{}
	params.Set("symbols", "ALL")
	if len(symbols) > 0 {
		params.Set("symbols", strings.Join(symbols, ","))
	}
	var resp *TickersResponse
	if err := e.SendHTTPRequest(ctx, tickersEPL, http.MethodGet, common.EncodeURLValues("tickers", params), nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetTickersHistory returns the hourly best bid and ask of trading pairs, such as tBTCUSD, or of every pair when
// Symbols is empty. Funding currencies are refused, as Bitfinex keeps no history of theirs and returns none
func (e *Exchange) GetTickersHistory(ctx context.Context, req *TickersHistoryRequest) ([]HistoricTicker, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbols", "ALL")
	if len(req.Symbols) > 0 {
		for _, symbol := range req.Symbols {
			if !isTradingSymbol(symbol) {
				return nil, fmt.Errorf("%w: %q", errTradingSymbolRequired, symbol)
			}
		}
		params.Set("symbols", strings.Join(req.Symbols, ","))
	}
	if err := setHistoryParams(params, req.Start, req.End, req.Limit, req.Sort); err != nil {
		return nil, err
	}
	var resp []HistoricTicker
	return resp, e.SendHTTPRequest(ctx, tickersHistoryEPL, http.MethodGet, common.EncodeURLValues("tickers/hist", params), nil, &resp)
}

// GetPublicTrades returns a trading pair's trades, for a symbol such as tBTCUSD; Bitfinex returns none, rather than an
// error, for a pair it does not list
func (e *Exchange) GetPublicTrades(ctx context.Context, req *PublicTradesRequest) ([]PublicTrade, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isTradingSymbol(req.Symbol) {
		return nil, errTradingSymbolRequired
	}
	params := url.Values{}
	if err := setHistoryParams(params, req.Start, req.End, req.Limit, req.Sort); err != nil {
		return nil, err
	}
	var resp []PublicTrade
	return resp, e.SendHTTPRequest(ctx, publicTradesEPL, http.MethodGet, common.EncodeURLValues("trades/"+req.Symbol+"/hist", params), nil, &resp)
}

// GetPublicFundingTrades returns a funding currency's trades, for a symbol such as fUSD; Bitfinex returns none, rather
// than an error, for a currency it does not list
func (e *Exchange) GetPublicFundingTrades(ctx context.Context, req *PublicTradesRequest) ([]PublicFundingTrade, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isFundingSymbol(req.Symbol) {
		return nil, errFundingSymbolRequired
	}
	params := url.Values{}
	if err := setHistoryParams(params, req.Start, req.End, req.Limit, req.Sort); err != nil {
		return nil, err
	}
	var resp []PublicFundingTrade
	return resp, e.SendHTTPRequest(ctx, publicTradesEPL, http.MethodGet, common.EncodeURLValues("trades/"+req.Symbol+"/hist", params), nil, &resp)
}

// GetBook returns a trading pair's book aggregated by price, bids then asks, for a symbol such as tBTCUSD
func (e *Exchange) GetBook(ctx context.Context, req *BookRequest) ([]BookLevel, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isTradingSymbol(req.Symbol) {
		return nil, errTradingSymbolRequired
	}
	precision, err := bookPrecision(req.Precision)
	if err != nil {
		return nil, err
	}
	params, err := bookParams(req.Length)
	if err != nil {
		return nil, err
	}
	var resp []BookLevel
	return resp, e.SendHTTPRequest(ctx, bookEPL, http.MethodGet, common.EncodeURLValues("book/"+req.Symbol+"/"+precision, params), nil, &resp)
}

// GetFundingBook returns a funding currency's book aggregated by rate and period, bids then offers, for a symbol such
// as fUSD
func (e *Exchange) GetFundingBook(ctx context.Context, req *BookRequest) ([]FundingBookLevel, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isFundingSymbol(req.Symbol) {
		return nil, errFundingSymbolRequired
	}
	precision, err := bookPrecision(req.Precision)
	if err != nil {
		return nil, err
	}
	params, err := bookParams(req.Length)
	if err != nil {
		return nil, err
	}
	var resp []FundingBookLevel
	return resp, e.SendHTTPRequest(ctx, bookEPL, http.MethodGet, common.EncodeURLValues("book/"+req.Symbol+"/"+precision, params), nil, &resp)
}

// GetRawBook returns a trading pair's book of individual orders, bids then asks, for a symbol such as tBTCUSD
func (e *Exchange) GetRawBook(ctx context.Context, req *RawBookRequest) ([]RawBookLevel, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isTradingSymbol(req.Symbol) {
		return nil, errTradingSymbolRequired
	}
	params, err := bookParams(req.Length)
	if err != nil {
		return nil, err
	}
	var resp []RawBookLevel
	return resp, e.SendHTTPRequest(ctx, bookEPL, http.MethodGet, common.EncodeURLValues("book/"+req.Symbol+"/R0", params), nil, &resp)
}

// GetRawFundingBook returns a funding currency's book of individual bids and offers, bids first, for a symbol such as
// fUSD
func (e *Exchange) GetRawFundingBook(ctx context.Context, req *RawBookRequest) ([]RawFundingBookLevel, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isFundingSymbol(req.Symbol) {
		return nil, errFundingSymbolRequired
	}
	params, err := bookParams(req.Length)
	if err != nil {
		return nil, err
	}
	var resp []RawFundingBookLevel
	return resp, e.SendHTTPRequest(ctx, bookEPL, http.MethodGet, common.EncodeURLValues("book/"+req.Symbol+"/R0", params), nil, &resp)
}

// bookPrecision returns the path segment of an aggregated book's precision, 0 to 4 for P0 to P4
func bookPrecision(precision uint64) (string, error) {
	if precision > 4 {
		return "", fmt.Errorf("%w: %d", errInvalidBookPrecision, precision)
	}
	return "P" + strconv.FormatUint(precision, 10), nil
}

// bookParams returns the query of a book's length, which Bitfinex refuses unless it is 1, 25, 100 or 250; unset, it is
// left to Bitfinex's default of 25
func bookParams(length uint64) (url.Values, error) {
	params := url.Values{}
	switch length {
	case 0:
	case 1, 25, 100, 250:
		params.Set("len", strconv.FormatUint(length, 10))
	default:
		return nil, fmt.Errorf("%w: %d", errInvalidBookLength, length)
	}
	return params, nil
}

// GetLastStat returns a statistic's latest value
func (e *Exchange) GetLastStat(ctx context.Context, req *StatsRequest) (*Stat, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	key, err := statsKey(req)
	if err != nil {
		return nil, err
	}
	var resp *Stat
	if err := e.SendHTTPRequest(ctx, statsEPL, http.MethodGet, "stats1/"+key+"/last", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetStatsHistory returns a statistic's values, newest first unless sorted otherwise; Bitfinex keeps a value every
// minute, every 30 minutes for the volume keys and every day for the volume weighted average price
func (e *Exchange) GetStatsHistory(ctx context.Context, req *StatsHistoryRequest) ([]Stat, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	key, err := statsKey(&req.StatsRequest)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setHistoryParams(params, req.Start, req.End, req.Limit, req.Sort); err != nil {
		return nil, err
	}
	var resp []Stat
	return resp, e.SendHTTPRequest(ctx, statsEPL, http.MethodGet, common.EncodeURLValues("stats1/"+key+"/hist", params), nil, &resp)
}

// statsKey returns the key:size:symbol path segment of a statistic, with its side or pair where its key takes one.
// Bitfinex returns nothing, rather than an error, for a symbol, side or pair a key does not take, so those are refused
func statsKey(req *StatsRequest) (string, error) {
	size, ok := statsSizes[req.Key]
	if !ok {
		return "", fmt.Errorf("%w: %q", errInvalidStatsKey, req.Key)
	}
	switch req.Key {
	case StatsKeyPositionSize, StatsKeyVolumeWeightedAveragePrice:
		if !isTradingSymbol(req.Symbol) {
			return "", fmt.Errorf("%w: key %s takes a trading pair", errTradingSymbolRequired, req.Key)
		}
	case StatsKeyFundingSize, StatsKeyCreditsSize, StatsKeyCreditsSizeByPair:
		if !isFundingSymbol(req.Symbol) {
			return "", fmt.Errorf("%w: key %s takes a funding currency", errFundingSymbolRequired, req.Key)
		}
	default:
		if req.Symbol != StatsPlatform {
			return "", fmt.Errorf("%w: %q, the volume keys take %s", errInvalidStatsSymbol, req.Symbol, StatsPlatform)
		}
	}
	key := req.Key + ":" + size + ":" + req.Symbol
	switch {
	case req.Key == StatsKeyPositionSize:
		if req.Side != StatsSideLong && req.Side != StatsSideShort {
			return "", fmt.Errorf("%w: %q, want long or short", errInvalidStatsSide, req.Side)
		}
		key += ":" + req.Side
	case req.Side != "":
		return "", fmt.Errorf("%w: only key %s takes a side", errInvalidStatsSide, StatsKeyPositionSize)
	}
	switch {
	case req.Key == StatsKeyCreditsSizeByPair:
		if !isTradingSymbol(req.Pair) {
			return "", fmt.Errorf("%w: key %s takes the pair whose positions use the funding", errTradingSymbolRequired, req.Key)
		}
		key += ":" + req.Pair
	case req.Pair != "":
		return "", fmt.Errorf("%w: only key %s takes a pair", errInvalidStatsPair, StatsKeyCreditsSizeByPair)
	}
	return key, nil
}

// GetLastCandle returns a trading pair's or funding currency's latest candle
func (e *Exchange) GetLastCandle(ctx context.Context, req *CandlesRequest) (*Candle, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	key, err := candleKey(req)
	if err != nil {
		return nil, err
	}
	var resp *Candle
	if err := e.SendHTTPRequest(ctx, candlesEPL, http.MethodGet, "candles/"+key+"/last", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetCandlesHistory returns a trading pair's or funding currency's candles, the latest 100 unless limited otherwise
func (e *Exchange) GetCandlesHistory(ctx context.Context, req *CandlesHistoryRequest) ([]Candle, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	key, err := candleKey(&req.CandlesRequest)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setHistoryParams(params, req.Start, req.End, req.Limit, req.Sort); err != nil {
		return nil, err
	}
	var resp []Candle
	return resp, e.SendHTTPRequest(ctx, candlesEPL, http.MethodGet, common.EncodeURLValues("candles/"+key+"/hist", params), nil, &resp)
}

// candleKey returns the trade:TIME_FRAME:SYMBOL path segment of candles, with a funding currency's period or aggregated
// periods. Bitfinex returns nothing, rather than an error, for a funding currency without a period and a trading pair
// with one, so those are refused
func candleKey(req *CandlesRequest) (string, error) {
	timeFrame, ok := candleTimeFrames[req.Interval]
	if !ok {
		return "", fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, req.Interval)
	}
	key := "trade:" + timeFrame + ":" + req.Symbol
	switch {
	case isTradingSymbol(req.Symbol):
		if req.Period != 0 || req.Aggregation != 0 || req.PeriodStart != 0 || req.PeriodEnd != 0 {
			return "", fmt.Errorf("%w: a trading pair's candles take no period or aggregation", errCandlesPeriodUnsupported)
		}
		return key, nil
	case !isFundingSymbol(req.Symbol):
		return "", errSymbolRequired
	}
	if req.Aggregation == 0 {
		if req.PeriodStart != 0 || req.PeriodEnd != 0 {
			return "", fmt.Errorf("%w: a period start and end need an aggregation", errInvalidCandlesAggregation)
		}
		if req.Period == 0 {
			return "", errCandlesPeriodRequired
		}
		return key + ":p" + strconv.FormatUint(req.Period, 10), nil
	}
	if req.Aggregation != 10 && req.Aggregation != 30 {
		return "", fmt.Errorf("%w: %d", errInvalidCandlesAggregation, req.Aggregation)
	}
	if req.Period != 0 {
		return "", fmt.Errorf("%w: aggregated candles take a period start and end instead", errCandlesPeriodUnsupported)
	}
	if req.PeriodStart == 0 || req.PeriodEnd == 0 {
		return "", errCandlesPeriodRequired
	}
	return key + ":a" + strconv.FormatUint(req.Aggregation, 10) + ":p" + strconv.FormatUint(req.PeriodStart, 10) + ":p" + strconv.FormatUint(req.PeriodEnd, 10), nil
}

// GetDerivativesStatus returns the status of derivative pairs, by keys such as tBTCF0:USTF0, or of every pair when keys
// is empty
func (e *Exchange) GetDerivativesStatus(ctx context.Context, keys []string) ([]DerivativeStatusEntry, error) {
	params := url.Values{}
	params.Set("keys", "ALL")
	if len(keys) > 0 {
		params.Set("keys", strings.Join(keys, ","))
	}
	var resp []DerivativeStatusEntry
	return resp, e.SendHTTPRequest(ctx, derivativesStatusEPL, http.MethodGet, common.EncodeURLValues("status/deriv", params), nil, &resp)
}

// GetDerivativesStatusHistory returns a derivative pair's status over time, for a key such as tBTCF0:USTF0, newest
// first unless sorted otherwise; Bitfinex returns none, rather than an error, for a pair it does not list
func (e *Exchange) GetDerivativesStatusHistory(ctx context.Context, req *DerivativesStatusHistoryRequest) ([]DerivativeStatus, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isTradingSymbol(req.Key) {
		return nil, errTradingSymbolRequired
	}
	params := url.Values{}
	if err := setHistoryParams(params, req.Start, req.End, req.Limit, req.Sort); err != nil {
		return nil, err
	}
	var resp []DerivativeStatus
	return resp, e.SendHTTPRequest(ctx, derivativesStatusHistoryEPL, http.MethodGet, common.EncodeURLValues("status/deriv/"+req.Key+"/hist", params), nil, &resp)
}

// GetLiquidations returns positions' liquidations, newest first unless sorted otherwise; a liquidation's trigger and
// each of its market executions are separate liquidations
func (e *Exchange) GetLiquidations(ctx context.Context, req *LiquidationsRequest) ([]Liquidation, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setHistoryParams(params, req.Start, req.End, req.Limit, req.Sort); err != nil {
		return nil, err
	}
	var resp liquidationsResponse
	if err := e.SendHTTPRequest(ctx, liquidationsEPL, http.MethodGet, common.EncodeURLValues("liquidations/hist", params), nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetLeaderboards returns a leaderboard's standings, newest first unless sorted otherwise; Bitfinex returns none,
// rather than an error, for a leaderboard that does not run
func (e *Exchange) GetLeaderboards(ctx context.Context, req *LeaderboardsRequest) ([]LeaderboardStanding, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	switch req.Key {
	case LeaderboardKeyUnrealisedProfitPeriodDelta, LeaderboardKeyUnrealisedProfitInception, LeaderboardKeyVolume, LeaderboardKeyRealisedProfit:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidLeaderboardKey, req.Key)
	}
	timeFrame, ok := leaderboardTimeFrames[req.TimeFrame]
	if !ok {
		return nil, fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, req.TimeFrame)
	}
	if !isTradingSymbol(req.Symbol) {
		return nil, errTradingSymbolRequired
	}
	params := url.Values{}
	if err := setHistoryParams(params, req.Start, req.End, req.Limit, req.Sort); err != nil {
		return nil, err
	}
	var resp []LeaderboardStanding
	return resp, e.SendHTTPRequest(ctx, leaderboardsEPL, http.MethodGet, common.EncodeURLValues("rankings/"+req.Key+":"+timeFrame+":"+req.Symbol+"/hist", params), nil, &resp)
}

// GetFundingStatistics returns a funding currency's hourly funding statistics, newest first, for a symbol such as fUSD
func (e *Exchange) GetFundingStatistics(ctx context.Context, req *FundingStatisticsRequest) ([]FundingStatistic, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isFundingSymbol(req.Symbol) {
		return nil, errFundingSymbolRequired
	}
	params := url.Values{}
	if err := setHistoryParams(params, req.Start, req.End, req.Limit, 0); err != nil {
		return nil, err
	}
	var resp []FundingStatistic
	return resp, e.SendHTTPRequest(ctx, fundingStatisticsEPL, http.MethodGet, common.EncodeURLValues("funding/stats/"+req.Symbol+"/hist", params), nil, &resp)
}

// GetCurrencySymbols returns the currencies whose symbols elsewhere differ from Bitfinex's, such as DSH, known as DASH
func (e *Exchange) GetCurrencySymbols(ctx context.Context) ([]CurrencySymbol, error) {
	return getPublicConfig[[]CurrencySymbol](ctx, e, "conf/pub:map:currency:sym")
}

// GetCurrencyLabels returns the currencies' names, such as Bitcoin for BTC
func (e *Exchange) GetCurrencyLabels(ctx context.Context) ([]CurrencyLabel, error) {
	return getPublicConfig[[]CurrencyLabel](ctx, e, "conf/pub:map:currency:label")
}

// GetCurrencyUnits returns the units of the currencies whose amounts are in units other than whole coins, such as
// MegaIOTA for IOT
func (e *Exchange) GetCurrencyUnits(ctx context.Context) ([]CurrencyUnit, error) {
	return getPublicConfig[[]CurrencyUnit](ctx, e, "conf/pub:map:currency:unit")
}

// GetCurrencyUnderlyings returns the currencies derivative currencies track, such as BTC for BTCF0
func (e *Exchange) GetCurrencyUnderlyings(ctx context.Context) ([]CurrencyUnderlying, error) {
	return getPublicConfig[[]CurrencyUnderlying](ctx, e, "conf/pub:map:currency:undl")
}

// GetCurrencyPools returns the networks currencies are carried on, such as ETH for the tokens on Ethereum
func (e *Exchange) GetCurrencyPools(ctx context.Context) ([]CurrencyPool, error) {
	return getPublicConfig[[]CurrencyPool](ctx, e, "conf/pub:map:currency:pool")
}

// GetCurrencyExplorers returns the currencies' block explorers
func (e *Exchange) GetCurrencyExplorers(ctx context.Context) ([]CurrencyExplorer, error) {
	return getPublicConfig[[]CurrencyExplorer](ctx, e, "conf/pub:map:currency:explorer")
}

// GetCurrencyWithdrawalFees returns the fee a withdrawal of each currency costs
func (e *Exchange) GetCurrencyWithdrawalFees(ctx context.Context) ([]CurrencyWithdrawalFee, error) {
	return getPublicConfig[[]CurrencyWithdrawalFee](ctx, e, "conf/pub:map:currency:tx:fee")
}

// GetTransferMethods returns the methods API withdrawals take, such as BITCOIN, with the currencies each moves; a
// currency may move by several methods, one for each network it is carried on
func (e *Exchange) GetTransferMethods(ctx context.Context) ([]TransferMethod, error) {
	return getPublicConfig[[]TransferMethod](ctx, e, "conf/pub:map:tx:method")
}

// GetSpotPairs returns the trading pairs of the exchange wallet, without their t prefix, such as BTCUSD and AAVE:USD;
// they include the securities pairs GetSecuritiesPairs returns
func (e *Exchange) GetSpotPairs(ctx context.Context) ([]string, error) {
	return getPublicConfig[[]string](ctx, e, "conf/pub:list:pair:exchange")
}

// GetMarginPairs returns the trading pairs margin trading offers, without their t prefix, such as BTCUSD
func (e *Exchange) GetMarginPairs(ctx context.Context) ([]string, error) {
	return getPublicConfig[[]string](ctx, e, "conf/pub:list:pair:margin")
}

// GetFuturesPairs returns the derivative pairs, without their t prefix, such as BTCF0:USTF0
func (e *Exchange) GetFuturesPairs(ctx context.Context) ([]string, error) {
	return getPublicConfig[[]string](ctx, e, "conf/pub:list:pair:futures")
}

// GetSecuritiesPairs returns the securities pairs, without their t prefix; GetSpotPairs includes them
func (e *Exchange) GetSecuritiesPairs(ctx context.Context) ([]string, error) {
	return getPublicConfig[[]string](ctx, e, "conf/pub:list:pair:securities")
}

// GetCurrencies returns every currency on the platform, including derivative currencies such as BTCF0; not all can be
// funded
func (e *Exchange) GetCurrencies(ctx context.Context) ([]currency.Code, error) {
	return getPublicConfig[[]currency.Code](ctx, e, "conf/pub:list:currency")
}

// GetMarginCurrencies returns the currencies margin trading takes as collateral
func (e *Exchange) GetMarginCurrencies(ctx context.Context) ([]currency.Code, error) {
	return getPublicConfig[[]currency.Code](ctx, e, "conf/pub:list:currency:margin")
}

// GetLeaderboardCompetitions returns the leaderboards that run, each the key, time frame and symbol
// LeaderboardsRequest takes, joined by colons, such as plu:3h:tBTCUSD
func (e *Exchange) GetLeaderboardCompetitions(ctx context.Context) ([]string, error) {
	return getPublicConfig[[]string](ctx, e, "conf/pub:list:competitions")
}

// GetPairInfo returns the trading pairs' market information, such as their order sizes and margin requirements
func (e *Exchange) GetPairInfo(ctx context.Context) ([]PairInfoEntry, error) {
	return getPublicConfig[[]PairInfoEntry](ctx, e, "conf/pub:info:pair")
}

// GetFuturesPairInfo returns the derivative pairs' market information, such as their order sizes and margin
// requirements
func (e *Exchange) GetFuturesPairInfo(ctx context.Context) ([]PairInfoEntry, error) {
	return getPublicConfig[[]PairInfoEntry](ctx, e, "conf/pub:info:pair:futures")
}

// GetTransferMethodStatuses returns whether each transfer method's deposits and withdrawals run, and what they need
func (e *Exchange) GetTransferMethodStatuses(ctx context.Context) ([]TransferMethodStatus, error) {
	return getPublicConfig[[]TransferMethodStatus](ctx, e, "conf/pub:info:tx:status")
}

// GetMarginSpecification returns the collateral haircuts and margin requirements of margin trading
func (e *Exchange) GetMarginSpecification(ctx context.Context) (*MarginSpecificationResponse, error) {
	resp, err := getPublicConfig[MarginSpecificationResponse](ctx, e, "conf/pub:spec:margin")
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDerivativesFees returns the fee settings of derivatives
func (e *Exchange) GetDerivativesFees(ctx context.Context) (*DerivativesFeesResponse, error) {
	resp, err := getPublicConfig[DerivativesFeesResponse](ctx, e, "conf/pub:fees")
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// getPublicConfig returns a config's value, which Bitfinex sends alone in an array, holding null for a config it has no
// value for
func getPublicConfig[T any](ctx context.Context, e *Exchange, path string) (T, error) {
	var zero T
	var resp []*T
	if err := e.SendHTTPRequest(ctx, configsEPL, http.MethodGet, path, nil, &resp); err != nil {
		return zero, err
	}
	if len(resp) == 0 || resp[0] == nil {
		return zero, common.ErrNoResponse
	}
	return *resp[0], nil
}

// GetVirtualAssetServiceProviders returns the virtual asset service providers a withdrawal can name as its recipient,
// for the Travel Rule
func (e *Exchange) GetVirtualAssetServiceProviders(ctx context.Context) ([]VirtualAssetServiceProvider, error) {
	var resp []VirtualAssetServiceProvider
	return resp, e.SendHTTPRequest(ctx, virtualAssetServiceProvidersEPL, http.MethodGet, "ext/vasps", nil, &resp)
}

// GetMarketAveragePrice returns the average price at which a trading pair's book would fill an amount, for a symbol
// such as tBTCUSD; Bitfinex returns a zero price and amount, rather than an error, for a pair it does not list
func (e *Exchange) GetMarketAveragePrice(ctx context.Context, req *MarketAveragePriceRequest) (*MarketAveragePriceResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isTradingSymbol(req.Symbol) {
		return nil, errTradingSymbolRequired
	}
	if req.Amount == 0 {
		return nil, errInvalidAmount
	}
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	if req.PriceLimit != 0 {
		params.Set("rate_limit", strconv.FormatFloat(req.PriceLimit, 'f', -1, 64))
	}
	var resp *MarketAveragePriceResponse
	if err := e.SendHTTPRequest(ctx, marketAveragePriceEPL, http.MethodPost, common.EncodeURLValues("calc/trade/avg", params), nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetFundingMarketAveragePrice returns the average daily rate at which a funding currency's book would fill an amount,
// for a symbol such as fUSD; Bitfinex returns a zero rate and amount, rather than an error, for a currency it does not
// list
func (e *Exchange) GetFundingMarketAveragePrice(ctx context.Context, req *FundingMarketAveragePriceRequest) (*FundingMarketAveragePriceResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isFundingSymbol(req.Symbol) {
		return nil, errFundingSymbolRequired
	}
	if req.Amount == 0 {
		return nil, errInvalidAmount
	}
	if req.Period == 0 {
		return nil, errMarketAveragePeriodRequired
	}
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	params.Set("period", strconv.FormatUint(req.Period, 10))
	if req.RateLimit != 0 {
		params.Set("rate_limit", strconv.FormatFloat(req.RateLimit, 'f', -1, 64))
	}
	var resp *FundingMarketAveragePriceResponse
	if err := e.SendHTTPRequest(ctx, marketAveragePriceEPL, http.MethodPost, common.EncodeURLValues("calc/trade/avg", params), nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetForeignExchangeRate returns the exchange rate between two currencies, such as BTC in USD
func (e *Exchange) GetForeignExchangeRate(ctx context.Context, req *ForeignExchangeRateRequest) (*ForeignExchangeRateResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.BaseCurrency.IsEmpty() || req.QuoteCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	body := &foreignExchangeRateBody{BaseCurrency: req.BaseCurrency.Upper().String(), QuoteCurrency: req.QuoteCurrency.Upper().String()}
	var resp *ForeignExchangeRateResponse
	if err := e.SendHTTPRequest(ctx, foreignExchangeRateEPL, http.MethodPost, "calc/fx", body, &resp); err != nil {
		return nil, err
	}
	// Bitfinex answers [null] for a currency it has no rate for, which leaves the rate zero
	if resp == nil || resp.CurrentRate == 0 {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}
