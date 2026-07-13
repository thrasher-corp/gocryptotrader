package gateio

import (
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// BotStrategyType represents the complete enumeration of policy types supported by AIHub.
const (
	BotStrategySpotGrid           = "spot_grid"
	BotStrategyMarginGrid         = "margin_grid"
	BotStrategyInfiniteGrid       = "infinite_grid"
	BotStrategyFuturesGrid        = "futures_grid"
	BotStrategySpotMartingale     = "spot_martingale"
	BotStrategyContractMartingale = "contract_martingale"
)

// Enumeration of scenarios supported by the policy recommendation interface.
const (
	BotSceneTop1    = "top1"
	BotSceneBundle  = "bundle"
	BotSceneFilter  = "filter"
	BotSceneRefresh = "refresh"
)

// GetBotStrategyRecommendationsRequest holds parameters for the AIHub strategy recommendation endpoint.
type GetBotStrategyRecommendationsRequest struct {
	Market                  string `json:"market,omitempty"`
	StrategyType            string `json:"strategy_type,omitempty"`
	Direction               string `json:"direction,omitempty"`
	InvestAmount            string `json:"invest_amount,omitempty"`
	Scene                   string `json:"scene,omitempty"`
	RefreshRecommendationID string `json:"refresh_recommendation_id,omitempty"`
	Limit                   int32  `json:"limit,omitempty"`
	MaxDrawdownLTE          string `json:"max_drawdown_lte,omitempty"`
	BacktestAPRGTE          string `json:"backtest_apr_gte,omitempty"`
}

// BotRecommendation holds a single strategy recommendation.
type BotRecommendation struct {
	RecommendationID      string `json:"recommendation_id"`
	Market                string `json:"market"`
	StrategyType          string `json:"strategy_type"`
	StrategyName          string `json:"strategy_name"`
	BacktestAPR           string `json:"backtest_apr"`
	MaxDrawdown           string `json:"max_drawdown"`
	Summary               string `json:"summary"`
	StrategyParamsPreview string `json:"strategy_params_preview"`
}

// BotDiscoverData holds the strategy recommendation result data.
type BotDiscoverData struct {
	Scene              string               `json:"scene"`
	Recommendations    []*BotRecommendation `json:"recommendations"`
	UnsupportedFilters []string             `json:"unsupported_filters"`
}

// BotDiscoverResponse is the unified response for the strategy recommendation endpoint.
type BotDiscoverResponse struct {
	Code    int32            `json:"code"`
	Message string           `json:"message"`
	Data    *BotDiscoverData `json:"data"`
	TraceID string           `json:"trace_id"`
}

func (b *BotDiscoverResponse) Error() error { return botResponseError(b.Code, b.Message) }

// BotCreateData holds the policy information returned after a strategy is successfully created.
type BotCreateData struct {
	StrategyID   string `json:"strategy_id"`
	StrategyType string `json:"strategy_type"`
	Market       string `json:"market"`
	Status       string `json:"status"`
	JumpURL      string `json:"jump_url"`
}

// BotCreateResponse is the unified response returned when a strategy is created.
type BotCreateResponse struct {
	Code    int32         `json:"code"`
	Message string        `json:"message"`
	Data    BotCreateData `json:"data"`
	TraceID string        `json:"trace_id"`
}

func (b *BotCreateResponse) Error() error { return botResponseError(b.Code, b.Message) }

// SpotGridCreateParams holds the creation parameters for a spot grid strategy.
type SpotGridCreateParams struct {
	Money              types.Number `json:"money"`
	LowPrice           types.Number `json:"low_price"`
	HighPrice          types.Number `json:"high_price"`
	GridNumber         int32        `json:"grid_num"`
	PriceType          order.Side   `json:"price_type"`
	TriggerPrice       string       `json:"trigger_price,omitempty"`
	StopProfit         types.Number `json:"stop_profit,omitempty"`
	StopLoss           types.Number `json:"stop_loss,omitempty"`
	ProfitSharingRatio types.Number `json:"profit_sharing_ratio,omitempty"`
	IsUseBase          bool         `json:"is_use_base,omitempty"`
}

// SpotGridCreateRequest is the request body for creating a spot grid strategy.
type SpotGridCreateRequest struct {
	StrategyType string               `json:"strategy_type"`
	Market       string               `json:"market"`
	CreateParams SpotGridCreateParams `json:"create_params"`
}

// MarginGridCreateParams holds the creation parameters for a leverage grid strategy.
type MarginGridCreateParams struct {
	Money              string       `json:"money"`
	LowPrice           types.Number `json:"low_price"`
	HighPrice          types.Number `json:"high_price"`
	GridNum            int32        `json:"grid_num"`
	PriceType          order.Side   `json:"price_type"`
	Leverage           types.Number `json:"leverage"`
	Direction          string       `json:"direction,omitempty"` // possible values: 'long', 'short', and 'neutral'
	TriggerPrice       types.Number `json:"trigger_price,omitempty"`
	StopProfit         types.Number `json:"stop_profit,omitempty"`
	StopLoss           types.Number `json:"stop_loss,omitempty"`
	ProfitSharingRatio types.Number `json:"profit_sharing_ratio,omitempty"`
	IsUseBase          bool         `json:"is_use_base,omitempty"`
}

// MarginGridCreateRequest is the request body for creating a leverage grid strategy.
type MarginGridCreateRequest struct {
	StrategyType string                 `json:"strategy_type"`
	Market       string                 `json:"market"`
	CreateParams MarginGridCreateParams `json:"create_params"`
}

// InfiniteGridCreateParams holds the creation parameters for an infinite grid strategy.
type InfiniteGridCreateParams struct {
	Money              string     `json:"money"`
	PriceFloor         string     `json:"price_floor"`
	ProfitPerGrid      string     `json:"profit_per_grid"`
	GridNum            int32      `json:"grid_num,omitempty"`
	PriceType          order.Side `json:"price_type,omitempty"`
	TriggerPrice       string     `json:"trigger_price,omitempty"`
	StopProfit         string     `json:"stop_profit,omitempty"`
	StopLoss           string     `json:"stop_loss,omitempty"`
	ProfitSharingRatio string     `json:"profit_sharing_ratio,omitempty"`
	IsUseBase          bool       `json:"is_use_base,omitempty"`
}

// InfiniteGridCreateRequest is the request body for creating an infinite grid strategy.
type InfiniteGridCreateRequest struct {
	StrategyType string                   `json:"strategy_type"`
	Market       string                   `json:"market"`
	CreateParams InfiniteGridCreateParams `json:"create_params"`
}

// FuturesGridCreateParams holds the creation parameters for a futures (contract) grid strategy.
type FuturesGridCreateParams struct {
	Money              types.Number `json:"money"`
	LowPrice           types.Number `json:"low_price,omitempty"`
	HighPrice          types.Number `json:"high_price,omitempty"`
	GridNum            int32        `json:"grid_num"`
	PriceType          order.Side   `json:"price_type"`
	Leverage           types.Number `json:"leverage"`
	Direction          string       `json:"direction,omitempty"` // possible values: 'long', 'short', and 'neutral'
	TriggerPrice       types.Number `json:"trigger_price,omitempty"`
	StopProfit         types.Number `json:"stop_profit,omitempty"`
	StopLoss           types.Number `json:"stop_loss,omitempty"`
	ProfitSharingRatio types.Number `json:"profit_sharing_ratio,omitempty"`
	IsUseBase          bool         `json:"is_use_base,omitempty"`
}

// FuturesGridCreateRequest is the request body for creating a futures grid strategy.
type FuturesGridCreateRequest struct {
	StrategyType string                  `json:"strategy_type"`
	Market       string                  `json:"market"`
	CreateParams FuturesGridCreateParams `json:"create_params"`
}

// SpotMartingaleCreateParams holds the creation parameters for a spot martingale strategy.
type SpotMartingaleCreateParams struct {
	InvestAmount       types.Number `json:"invest_amount"`
	PriceDeviation     types.Number `json:"price_deviation"`
	MaxOrders          int32        `json:"max_orders"`
	TakeProfitRatio    types.Number `json:"take_profit_ratio"`
	StopLossPerCycle   types.Number `json:"stop_loss_per_cycle,omitempty"`
	TriggerPrice       types.Number `json:"trigger_price,omitempty"`
	ProfitSharingRatio types.Number `json:"profit_sharing_ratio,omitempty"`
}

// SpotMartingaleCreateRequest is the request body for creating a spot martingale strategy.
type SpotMartingaleCreateRequest struct {
	StrategyType string                     `json:"strategy_type"`
	Market       string                     `json:"market"`
	CreateParams SpotMartingaleCreateParams `json:"create_params"`
}

// ContractMartingaleCreateParams holds the creation parameters for a contract martingale strategy.
type ContractMartingaleCreateParams struct {
	InvestAmount       string `json:"invest_amount"`
	PriceDeviation     string `json:"price_deviation"`
	MaxOrders          int32  `json:"max_orders"`
	TakeProfitRatio    string `json:"take_profit_ratio"`
	Direction          string `json:"direction"`
	Leverage           string `json:"leverage"`
	StopLossPrice      string `json:"stop_loss_price,omitempty"`
	ProfitSharingRatio string `json:"profit_sharing_ratio,omitempty"`
}

// ContractMartingaleCreateRequest is the request body for creating a contract martingale strategy.
type ContractMartingaleCreateRequest struct {
	StrategyType string                         `json:"strategy_type"`
	Market       string                         `json:"market"`
	CreateParams ContractMartingaleCreateParams `json:"create_params"`
}

// BotPortfolioRunningItem holds a single record in the list of running policies.
type BotPortfolioRunningItem struct {
	StrategyID   string       `json:"strategy_id"`
	StrategyType string       `json:"strategy_type"`
	StrategyName string       `json:"strategy_name"`
	Market       string       `json:"market"`
	Status       string       `json:"status"`
	PNL          types.Number `json:"pnl"`
	PNLRate      types.Number `json:"pnl_rate"`
	InvestAmount types.Number `json:"invest_amount"`
	CreatedAt    types.Time   `json:"created_at"`
}

// BotPortfolioRunningData holds the running policy list data.
type BotPortfolioRunningData struct {
	Items    []*BotPortfolioRunningItem `json:"items"`
	Page     int32                      `json:"page"`
	PageSize int32                      `json:"page_size"`
	Total    int32                      `json:"total"`
}

// BotPortfolioRunningResponse is the unified response for querying the running policy list.
type BotPortfolioRunningResponse struct {
	Code    int32                   `json:"code"`
	Message string                  `json:"message"`
	Data    BotPortfolioRunningData `json:"data"`
	TraceID string                  `json:"trace_id"`
}

func (b *BotPortfolioRunningResponse) Error() error { return botResponseError(b.Code, b.Message) }

// BotPortfolioBaseInfo holds base information about a portfolio strategy.
type BotPortfolioBaseInfo struct {
	StrategyName    string       `json:"strategy_name"`
	CreatedAt       types.Time   `json:"created_at"`
	RunningDuration int64        `json:"running_duration"`
	InvestAmount    types.Number `json:"invest_amount"`
	TotalProfit     types.Number `json:"total_profit"`
	ProfitRate      types.Number `json:"profit_rate"`
}

// BotPortfolioMetrics holds the metrics for a portfolio strategy detail.
type BotPortfolioMetrics struct {
	GridProfit                string       `json:"grid_profit"`
	FloatingPNL               types.Number `json:"floating_pnl"`
	ArbitrageCount            int64        `json:"arbitrage_count"`
	PriceRange                string       `json:"price_range"`
	GridCount                 int64        `json:"grid_count"`
	EstimatedLiquidationPrice types.Number `json:"estimated_liquidation_price"`
	PriceFloor                types.Number `json:"price_floor"`
	GridProfitRate            types.Number `json:"grid_profit_rate"`
	RealizedPNL               types.Number `json:"realized_pnl"`
	FinishedRounds            int64        `json:"finished_rounds"`
	AverageCost               string       `json:"avg_cost"`
	TakeProfitPrice           types.Number `json:"take_profit_price"`
	MaintenanceMarginRatio    types.Number `json:"maintenance_margin_ratio"`
}

// BotPortfolioPosition holds the position information for a portfolio strategy detail.
type BotPortfolioPosition struct {
	Amount        types.Number `json:"amount"`
	EntryPrice    types.Number `json:"entry_price"`
	QuoteAmount   types.Number `json:"quote_amount"`
	PositionValue types.Number `json:"position_value"`
	Margin        string       `json:"margin"`
	Side          string       `json:"side"`
}

// BotPortfolioDetailData holds the detail of a single running strategy.
type BotPortfolioDetailData struct {
	StrategyID    string                `json:"strategy_id"`
	StrategyType  string                `json:"strategy_type"`
	Market        string                `json:"market"`
	Status        string                `json:"status"`
	BaseInfo      *BotPortfolioBaseInfo `json:"base_info"`
	Metrics       *BotPortfolioMetrics  `json:"metrics"`
	Position      *BotPortfolioPosition `json:"position"`
	StopSupported bool                  `json:"stop_supported"`
}

// BotPortfolioDetailResponse is the unified response for querying strategy details.
type BotPortfolioDetailResponse struct {
	Code    int32                  `json:"code"`
	Message string                 `json:"message"`
	Data    BotPortfolioDetailData `json:"data"`
	TraceID string                 `json:"trace_id"`
}

func (b *BotPortfolioDetailResponse) Error() error { return botResponseError(b.Code, b.Message) }

// BotPortfolioStopRequest is the request body for terminating a running strategy.
type BotPortfolioStopRequest struct {
	StrategyID   string `json:"strategy_id"`
	StrategyType string `json:"strategy_type"`
}

// BotPortfolioStopData holds the result information returned after a strategy is terminated.
type BotPortfolioStopData struct {
	StrategyID    string `json:"strategy_id"`
	StrategyType  string `json:"strategy_type"`
	Status        string `json:"status"`
	ResultMessage string `json:"result_message"`
}

// BotPortfolioStopResponse is the unified response for terminating a running strategy.
type BotPortfolioStopResponse struct {
	Code    int32                 `json:"code"`
	Message string                `json:"message"`
	Data    *BotPortfolioStopData `json:"data"`
	TraceID string                `json:"trace_id"`
}

// Error implements the error check interface for use by requester.
func (b *BotPortfolioStopResponse) Error() error { return botResponseError(b.Code, b.Message) }
