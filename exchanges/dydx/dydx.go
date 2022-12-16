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
	usernameExists                 = "usernames"
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
	errInvalidPeriod           = errors.New("invalid period specified")
	errSortByIsRequired        = errors.New("parameter \"sortBy\" is required")
	errMissingPublicID         = errors.New("missing user public id")
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

// GetGlobalConfigurationVariables retrives any global configuration variables for the exchange as a whole.
func (dy *DYDX) GetGlobalConfigurationVariables(ctx context.Context) (*ConfigurationVariableResponse, error) {
	var resp ConfigurationVariableResponse
	return &resp, dy.SendHTTPRequest(ctx, exchange.RestSpot, globalConfigurations, &resp)
}

// CheckIfUserExists checks if a user exists for a given Ethereum address.
func (dy *DYDX) CheckIfUserExists(ctx context.Context, etheriumAddress string) (bool, error) {
	resp := &struct {
		Exists bool `json:"exists"`
	}{}
	return resp.Exists, dy.SendHTTPRequest(ctx, exchange.RestSpot, usersExists+"?ethereumAddress="+etheriumAddress, resp)
}

// CheckIfUsernameExists check if a username has been taken by a user.
func (dy *DYDX) CheckIfUsernameExists(ctx context.Context, username string) (bool, error) {
	resp := &struct {
		Exists bool `json:"exists"`
	}{}
	return resp.Exists, dy.SendHTTPRequest(ctx, exchange.RestSpot, usernameExists+"?username="+username, resp)
}

// GetAPIServerTime get the current time of the API server.
func (dy *DYDX) GetAPIServerTime(ctx context.Context) (*APIServerTime, error) {
	var resp APIServerTime
	return &resp, dy.SendHTTPRequest(ctx, exchange.RestSpot, apiServerTime, &resp)
}

// GetPublicLeaderboardPNLs retrives the top PNLs for a specified period and how they rank against each other.
func (dy *DYDX) GetPublicLeaderboardPNLs(ctx context.Context, period, sortBy string, startingBeforeOrAt time.Time, limit int64) (*LeaderboardPNLs, error) {
	params := url.Values{}
	if period == "" {
		return nil, fmt.Errorf("%w \"period\" is required", errInvalidPeriod)
	}
	params.Set("period", period)
	if !startingBeforeOrAt.IsZero() {
		params.Set("startingBeforeOrAt", startingBeforeOrAt.Format("2022-02-02T15:31:10.813Z"))
	}
	if sortBy == "" {
		return nil, errSortByIsRequired
	}
	params.Set("sortBy", sortBy)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp LeaderboardPNLs
	return &resp, dy.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(leaderboardPNL, params), &resp)
}

// GetPublicRetroactiveMiningReqards retrives the retroactive mining rewards for an ethereum address.
func (dy *DYDX) GetPublicRetroactiveMiningReqards(ctx context.Context, ethereumAddress string) (*RetroactiveMiningReward, error) {
	var resp RetroactiveMiningReward
	return &resp, dy.SendHTTPRequest(ctx, exchange.RestSpot, publicRetroactiveMiningRewards+"?ethereumAddress="+ethereumAddress, &resp)
}

// VerifyEmailAddress verify an email address by providing the verification token sent to the email address.
func (dy *DYDX) VerifyEmailAddress(ctx context.Context, token string) (interface{}, error) {
	var response interface{}
	return response, dy.SendHTTPRequest(ctx, exchange.RestSpot, verifyEmailAddress+"?token="+token, response)
}

// GetCurrentlyRevealedHedgies retrives the currently revealed Hedgies for competition distribution.
func (dy *DYDX) GetCurrentlyRevealedHedgies(ctx context.Context, daily, weekly string) (*CurrentRevealedHedgies, error) {
	params := url.Values{}
	if daily != "" {
		params.Set("daily", daily)
	}
	if weekly != "" {
		params.Set("weekly", weekly)
	}
	var resp CurrentRevealedHedgies
	return &resp, dy.SendHTTPRequest(ctx, exchange.RestSpot, currentHedgies, &resp)
}

// GetHistoricallyRevealedHedgies retrives the historically revealed Hedgies from competition distributions.
func (dy *DYDX) GetHistoricallyRevealedHedgies(ctx context.Context, nftRevealType string, start, end int64) (*HistoricalRevealedHedgies, error) {
	params := url.Values{}
	if nftRevealType != "" {
		params.Set("nftRevealType", nftRevealType)
	}
	if start != 0 {
		params.Set("start", strconv.FormatInt(start, 10))
	}
	if end != 0 {
		params.Set("end", strconv.FormatInt(end, 10))
	}
	var resp HistoricalRevealedHedgies
	return &resp, dy.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(historicalHedgies, params), &resp)
}

// GetInsuranceFundBalance retrives the balance of dydx insurance fund.
func (dy *DYDX) GetInsuranceFundBalance(ctx context.Context) (*InsuranceFundBalance, error) {
	var resp InsuranceFundBalance
	return &resp, dy.SendHTTPRequest(ctx, exchange.RestSpot, insuranceFundBalance, &resp)
}

// GetPublicProfile retrives the public profile of a user given their public id.
func (dy *DYDX) GetPublicProfile(ctx context.Context, publicID string) (*PublicProfile, error) {
	var resp PublicProfile
	if publicID == "" {
		return nil, errMissingPublicID
	}
	return &resp, dy.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf(publicProifle, publicID), &resp)
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
