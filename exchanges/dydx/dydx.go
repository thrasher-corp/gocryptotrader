package dydx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// DYDX is the overarching type across this package
type DYDX struct {
	exchange.Base
}

const (
	dydxAPIURL     = "https://api.dydx.exchange/" + dydxAPIVersion
	dydxAPIVersion = "v3/"
	dydxWSAPIURL   = "wss://api.dydx.exchange/" + dydxAPIVersion + "ws"

	// Public endpoints
	markets                        = "markets"
	marketOrderbook                = "orderbook/%s" // orderbook/:market
	marketTrades                   = "trades/%s"    // trades/:market
	fastWithdrawals                = "fast-withdrawals"
	marketStats                    = "stats/%s"              // stats/:market
	marketHistoricalFunds          = "historical-funding/%s" // historical-funding/:market
	marketCandles                  = "candles/%s"            // candles/:market
	globalConfigurations           = "config"
	usersExists                    = "users/exists"
	usernames                      = "usernames"
	apiServerTime                  = "time"
	leaderboardPNL                 = "leaderboard-pnl"
	publicRetroactiveMiningRewards = "rewards/public-retroactive-mining"
	verifyEmailAddress             = "emails/verify-email"
	historicalHedgies              = "hedgies/history"
	currentHedgies                 = "hedgies/current"
	insuranceFundBalance           = "insurance-fund/balance"
	publicProifle                  = "profile/%s" // profile/:publicId

	// Authenticated endpoints
)

var (
	errMissingMarketInstrument = errors.New("missing market instrument")
)

// GetMarkets retrives one or all markets as well as metadata about each retrieved market.
func (dy *DYDX) GetMarkets(ctx context.Context, instrument string) (*InstrumentDatas, error) {
	params := url.Values{}
	if instrument != "" {
		params.Set("market", instrument)
	}
	var resp InstrumentDatas
	return &resp, dy.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(markets, params), &resp)
}

// GetOrderbooks retrives  the active orderbook for a market. All bids and asks that are fillable are returned.
func (dy *DYDX) GetOrderbooks(ctx context.Context, instrument string) (*MarketOrderbook, error) {
	if instrument == "" {
		return nil, errMissingMarketInstrument
	}
	var resp MarketOrderbook
	return &resp, dy.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf(marketOrderbook, instrument), &resp)
}

// GetTrades retrives Trades by specified parameters. Passing in all query parameters to the HTTP endpoint would look like.
func (dy *DYDX) GetTrades(ctx context.Context, instrument string, startingBeforeOrAT time.Time, limit int64) ([]MarketTrade, error) {
	params := url.Values{}
	if instrument == "" {
		return nil, errMissingMarketInstrument
	}
	if !startingBeforeOrAT.IsZero() {
		params.Set("startingBeforeOrAt", startingBeforeOrAT.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp MarketTrades
	return resp.Trades, dy.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(fmt.Sprintf(marketTrades, instrument), params), &resp)
}

// GetFastWithdrawalLiquidity returns a map of all LP provider accounts that have available funds for fast withdrawals.
// Given a debitAmount and asset the user wants sent to L1, this endpoint also returns the predicted amount of the desired asset the user will be credited on L1.
// Given a creditAmount and asset the user wants sent to L1,
// this endpoint also returns the predicted amount the user will be debited on L2.
func (dy *DYDX) GetFastWithdrawalLiquidity(ctx context.Context, creditAsset string, creditAmount, debitAmount float64) (map[string]LiquidityProvider, error) {
	params := url.Values{}
	if creditAsset != "" {
		params.Set("creditAsset", creditAsset)
	}
	if (creditAmount != 0 || debitAmount != 0) && creditAsset == "" {
		return nil, errors.New("cannot find quote without creditAsset")
	} else if creditAmount != 0 && debitAmount != 0 {
		return nil, errors.New("creditAmount and debitAmount cannot both be set")
	}
	if creditAmount != 0 {
		params.Set("creditAmount", strconv.FormatFloat(creditAmount, 'f', -1, 64))
	}
	if debitAmount != 0 {
		params.Set("debitAmount", strconv.FormatFloat(debitAmount, 'f', -1, 64))
	}
	var resp WithdrawalLiquidityResponse
	return resp.LiquidityProviders, dy.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(fastWithdrawals, params), &resp)
}

// GetMarketStats retrives an individual market's statistics over a set period of time or all available periods of time.
func (dy *DYDX) GetMarketStats(ctx context.Context, instrument string, days int64) (map[string]TickerData, error) {
	params := url.Values{}
	if instrument == "" {
		return nil, errMissingMarketInstrument
	}
	if days != 0 {
		if days != 1 && days != 7 && days != 30 {
			return nil, errors.New("only 1,7, and 30 days are allowed")
		}
		params.Set("days", strconv.FormatInt(days, 10))
	}
	var resp TickerDatas
	return resp.Markets, dy.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf(marketStats, instrument), &resp)
}

// GetHistoricalFunding retrives the historical funding rates for a market.
func (dy *DYDX) GetHistoricalFunding(ctx context.Context, instrument string, effectiveBeforeOrAt time.Time) ([]HistoricalFunding, error) {
	params := url.Values{}
	if instrument == "" {
		return nil, errMissingMarketInstrument
	}
	if !effectiveBeforeOrAt.IsZero() {
		params.Set("effectiveBeforeOrAt", effectiveBeforeOrAt.String())
	}
	var resp HistoricFundingResponse
	return resp.HistoricalFundings, dy.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(fmt.Sprintf(marketHistoricalFunds, instrument), params), &resp)
}

// GetResolutionFromInterval returns the resolution(string representation of interval) from interval instance if supported by the exchange.
func (dy *DYDX) GetResolutionFromInterval(interval kline.Interval) (string, error) {
	switch interval {
	case kline.OneMin:
		return "1MIN", nil
	case kline.FiveMin:
		return "5MINS", nil
	case kline.FifteenMin:
		return "15MINS", nil
	case kline.ThirtyMin:
		return "30MINS", nil
	case kline.OneHour:
		return "1HOUR", nil
	case kline.FourHour:
		return "4HOURS", nil
	case kline.OneDay:
		return "1DAY", nil
	default:
		return "", kline.ErrUnsupportedInterval
	}
}

// GetCandlesForMarket retrives the candle statistics for a market.
func (dy *DYDX) GetCandlesForMarket(ctx context.Context, instrument string, interval kline.Interval, fromISO, toISO string, limit int64) ([]MarketCandle, error) {
	params := url.Values{}
	if instrument == "" {
		return nil, errMissingMarketInstrument
	}
	resolution, err := dy.GetResolutionFromInterval(interval)
	if err != nil {
		return nil, err
	}
	params.Set("resolution", resolution)
	if fromISO != "" {
		params.Set("fromISO", fromISO)
	}
	if toISO != "" {
		params.Set("toISO", toISO)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp MarketCandlesResponse
	return resp.Candles, dy.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(fmt.Sprintf(marketCandles, instrument), params), &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (dy *DYDX) SendHTTPRequest(ctx context.Context, endpoint exchange.URL, path string, result interface{}) error {
	urlPath, err := dy.API.Endpoints.GetURL(endpoint)
	if err != nil {
		return err
	}

	item := &request.Item{
		Method:        http.MethodGet,
		Path:          urlPath + path,
		Result:        result,
		Verbose:       dy.Verbose,
		HTTPDebugging: dy.HTTPDebugging,
		HTTPRecording: dy.HTTPRecording,
	}
	return dy.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	})
}
