package gateio

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errBotStrategyIDRequired     = errors.New("bot strategy ID required")
	errBotStrategyTypeRequired   = errors.New("bot strategy type required")
	errBotMarketRequired         = errors.New("bot market (trading pair) required")
	errBotMoneyRequired          = errors.New("bot investment amount required")
	errBotLowPriceRequired       = errors.New("bot low price required")
	errBotHighPriceRequired      = errors.New("bot high price required")
	errBotGridNumRequired        = errors.New("bot grid number must be greater than zero")
	errBotPriceFloorRequired     = errors.New("bot price floor required")
	errBotProfitPerGridRequired  = errors.New("bot profit per grid required")
	errBotInvestAmountRequired   = errors.New("bot invest amount required")
	errBotPriceDeviationRequired = errors.New("bot price deviation required")
	errBotMaxOrdersRequired      = errors.New("bot max orders must be greater than zero")
	errBotTakeProfitRequired     = errors.New("bot take profit ratio required")
	errBotDirectionRequired      = errors.New("bot direction required")
)

// botResponseError converts a non-zero bot API response code into an error.
func botResponseError(code int32, message string) error {
	if code != 0 && code != 200 {
		return fmt.Errorf("bot api error code: %d message: %s", code, message)
	}
	return nil
}

// GetBotStrategyRecommendations retrieves AIHub strategy recommendations.
// It is the only formal interface for the discover domain and supports top1, bundle, filter, and refresh scenes.
func (e *Exchange) GetBotStrategyRecommendations(ctx context.Context, arg *GetBotStrategyRecommendationsRequest) (*BotDiscoverData, error) {
	params := url.Values{}
	if arg != nil {
		if arg.Market != "" {
			params.Set("market", arg.Market)
		}
		if arg.StrategyType != "" {
			params.Set("strategy_type", arg.StrategyType)
		}
		if arg.Direction != "" {
			params.Set("direction", arg.Direction)
		}
		if arg.InvestAmount != "" {
			params.Set("invest_amount", arg.InvestAmount)
		}
		if arg.Scene != "" {
			params.Set("scene", arg.Scene)
		}
		if arg.RefreshRecommendationID != "" {
			params.Set("refresh_recommendation_id", arg.RefreshRecommendationID)
		}
		if arg.Limit > 0 {
			params.Set("limit", strconv.Itoa(int(arg.Limit)))
		}
		if arg.MaxDrawdownLTE != "" {
			params.Set("max_drawdown_lte", arg.MaxDrawdownLTE)
		}
		if arg.BacktestAPRGTE != "" {
			params.Set("backtest_apr_gte", arg.BacktestAPRGTE)
		}
	}
	var resp BotDiscoverResponse
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "bot/strategy/recommend", params, nil, &resp)
}

// CreateSpotGridBot creates a spot grid strategy based on the incoming parameters.
func (e *Exchange) CreateSpotGridBot(ctx context.Context, arg *SpotGridCreateRequest) (*BotCreateData, error) {
	if arg.Market == "" {
		return nil, errBotMarketRequired
	}
	if arg.CreateParams.Money <= 0 {
		return nil, errBotMoneyRequired
	}
	if arg.CreateParams.LowPrice <= 0 {
		return nil, errBotLowPriceRequired
	}
	if arg.CreateParams.HighPrice <= 0 {
		return nil, errBotHighPriceRequired
	}
	if arg.CreateParams.GridNum <= 0 {
		return nil, errBotGridNumRequired
	}
	arg.StrategyType = BotStrategySpotGrid
	var resp BotCreateResponse
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "bot/spot-grid/create", nil, arg, &resp)
}

// CreateMarginGridBot creates a leverage (margin) grid strategy based on the passed parameters.
func (e *Exchange) CreateMarginGridBot(ctx context.Context, arg *MarginGridCreateRequest) (*BotCreateData, error) {
	if arg.Market == "" {
		return nil, errBotMarketRequired
	}
	if arg.CreateParams.Money == "" {
		return nil, errBotMoneyRequired
	}
	if arg.CreateParams.LowPrice <= 0 {
		return nil, errBotLowPriceRequired
	}
	if arg.CreateParams.HighPrice <= 0 {
		return nil, errBotHighPriceRequired
	}
	if arg.CreateParams.GridNum <= 0 {
		return nil, errBotGridNumRequired
	}
	if arg.CreateParams.Leverage == "" {
		return nil, order.ErrSubmitLeverageNotSupported
	}
	arg.StrategyType = BotStrategyMarginGrid
	var resp BotCreateResponse
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "bot/margin-grid/create", nil, arg, &resp)
}

// CreateInfiniteGridBot creates an infinite grid strategy based on the passed parameters.
func (e *Exchange) CreateInfiniteGridBot(ctx context.Context, arg *InfiniteGridCreateRequest) (*BotCreateData, error) {
	if arg.Market == "" {
		return nil, errBotMarketRequired
	}
	if arg.CreateParams.Money == "" {
		return nil, errBotMoneyRequired
	}
	if arg.CreateParams.PriceFloor == "" {
		return nil, errBotPriceFloorRequired
	}
	if arg.CreateParams.ProfitPerGrid == "" {
		return nil, errBotProfitPerGridRequired
	}
	arg.StrategyType = BotStrategyInfiniteGrid
	var resp BotCreateResponse
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "bot/infinite-grid/create", nil, arg, &resp)
}

// CreateFuturesGridBot creates a futures (contract) grid strategy based on the incoming parameters.
func (e *Exchange) CreateFuturesGridBot(ctx context.Context, arg *FuturesGridCreateRequest) (*BotCreateData, error) {
	if arg.Market == "" {
		return nil, errBotMarketRequired
	}
	if arg.CreateParams.Money == "" {
		return nil, errBotMoneyRequired
	}
	if arg.CreateParams.LowPrice == "" {
		return nil, errBotLowPriceRequired
	}
	if arg.CreateParams.HighPrice == "" {
		return nil, errBotHighPriceRequired
	}
	if arg.CreateParams.GridNum <= 0 {
		return nil, errBotGridNumRequired
	}
	if arg.CreateParams.Leverage == "" {
		return nil, order.ErrSubmitLeverageNotSupported
	}
	arg.StrategyType = BotStrategyFuturesGrid
	var resp BotCreateResponse
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "bot/futures-grid/create", nil, arg, &resp)
}

// CreateSpotMartingaleBot creates a spot martingale strategy based on the passed parameters.
func (e *Exchange) CreateSpotMartingaleBot(ctx context.Context, arg *SpotMartingaleCreateRequest) (*BotCreateData, error) {
	if arg.Market == "" {
		return nil, errBotMarketRequired
	}
	if arg.CreateParams.InvestAmount == "" {
		return nil, errBotInvestAmountRequired
	}
	if arg.CreateParams.PriceDeviation == "" {
		return nil, errBotPriceDeviationRequired
	}
	if arg.CreateParams.MaxOrders <= 0 {
		return nil, errBotMaxOrdersRequired
	}
	if arg.CreateParams.TakeProfitRatio == "" {
		return nil, errBotTakeProfitRequired
	}
	arg.StrategyType = BotStrategySpotMartingale
	var resp BotCreateResponse
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "bot/spot-martingale/create", nil, arg, &resp)
}

// CreateContractMartingaleBot creates a contract martingale strategy based on the input parameters.
func (e *Exchange) CreateContractMartingaleBot(ctx context.Context, arg *ContractMartingaleCreateRequest) (*BotCreateData, error) {
	if arg.Market == "" {
		return nil, errBotMarketRequired
	}
	if arg.CreateParams.InvestAmount == "" {
		return nil, errBotInvestAmountRequired
	}
	if arg.CreateParams.PriceDeviation == "" {
		return nil, errBotPriceDeviationRequired
	}
	if arg.CreateParams.MaxOrders <= 0 {
		return nil, errBotMaxOrdersRequired
	}
	if arg.CreateParams.TakeProfitRatio == "" {
		return nil, errBotTakeProfitRequired
	}
	if arg.CreateParams.Direction == "" {
		return nil, errBotDirectionRequired
	}
	if arg.CreateParams.Leverage == "" {
		return nil, order.ErrSubmitLeverageNotSupported
	}
	arg.StrategyType = BotStrategyContractMartingale
	var resp BotCreateResponse
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "bot/contract-martingale/create", nil, arg, &resp)
}

// GetBotRunningStrategies queries the list of AIHub strategies currently running by the user. Supports filtering by strategy type, trading pair, and paging.
func (e *Exchange) GetBotRunningStrategies(ctx context.Context, strategyType, market string, page, pageSize int32) (*BotPortfolioRunningData, error) {
	params := url.Values{}
	if strategyType != "" {
		params.Set("strategy_type", strategyType)
	}
	if market != "" {
		params.Set("market", market)
	}
	if page > 0 {
		params.Set("page", strconv.Itoa(int(page)))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.Itoa(int(pageSize)))
	}
	var resp BotPortfolioRunningResponse
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "bot/portfolio/running", params, nil, &resp)
}

// GetBotStrategyDetail queries the detail of a single running AIHub strategy. Both strategyID and strategyType must be provided.
func (e *Exchange) GetBotStrategyDetail(ctx context.Context, strategyID, strategyType string) (*BotPortfolioDetailData, error) {
	if strategyID == "" {
		return nil, errBotStrategyIDRequired
	}
	if strategyType == "" {
		return nil, errBotStrategyTypeRequired
	}
	params := url.Values{}
	params.Set("strategy_id", strategyID)
	params.Set("strategy_type", strategyType)
	var resp BotPortfolioDetailResponse
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "bot/portfolio/detail", params, nil, &resp)
}

// StopBotStrategy terminates a single running AIHub strategy. Only one policy may be terminated per request.
func (e *Exchange) StopBotStrategy(ctx context.Context, strategyID string, strategyType string) (*BotPortfolioStopData, error) {
	if strategyID == "" {
		return nil, errBotStrategyIDRequired
	}
	if strategyType == "" {
		return nil, errBotStrategyTypeRequired
	}
	arg := &BotPortfolioStopRequest{
		StrategyID:   strategyID,
		StrategyType: strategyType,
	}
	var resp BotPortfolioStopResponse
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "bot/portfolio/stop", nil, arg, &resp)
}
