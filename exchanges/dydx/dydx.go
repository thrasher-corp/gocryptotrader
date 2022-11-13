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
