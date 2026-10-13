package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Bot request validation errors
var (
	errBotIDRequired                 = errors.New("bot ID required")
	errGridIDRequired                = errors.New("grid ID required")
	errBotInitialMarginRequired      = errors.New("bot initial margin required")
	errAdjustPositionModeRequired    = errors.New("adjust position mode required")
	errSymbolSettingsRequired        = errors.New("symbol settings required")
	errTargetPositionPercentRequired = errors.New("target position percent required")
	errGridCellNumberRequired        = errors.New("grid cell number required")
	errGridMinPriceRequired          = errors.New("grid min price required")
	errGridMaxPriceRequired          = errors.New("grid max price required")
	errGridTypeRequired              = errors.New("grid type required")
	errGridModeRequired              = errors.New("grid mode required")
	errGridTotalInvestmentRequired   = errors.New("grid total investment required")
	errMartingaleModeRequired        = errors.New("martingale mode required")
	errPriceFloatPercentRequired     = errors.New("price float percent required")
	errAddPositionPercentRequired    = errors.New("add position percent required")
	errAddPositionNumberRequired     = errors.New("add position number required")
	errRoundTPPercentRequired        = errors.New("round take-profit percent required")
	errBotCloseModeRequired          = errors.New("bot close mode required")
	errDCAFrequencyRequired          = errors.New("DCA frequency of at least a second required")
	errDCAPairsRequired              = errors.New("DCA pairs required")
)

// Bot results report their outcome in a status code beside the envelope's retCode; these errors match the codes the
// Bot Error Codes page documents
var (
	errBotRequestFailed     = errors.New("bot request failed")
	errBotInvalidParameters = errors.New("invalid bot request parameters")
	errBotUserBanned        = errors.New("user banned from bots")
	errBotRateLimitExceeded = errors.New("bot rate limit exceeded")
	errBotNotFound          = errors.New("bot not found")
	errBotNotClosable       = errors.New("bot cannot be closed in its current state")
)

// botStatusErrors maps the bot status codes that mean the same thing for every bot type. 401 is left out: it means
// authentication required for futures bots but insufficient balance when creating a spot grid bot
var botStatusErrors = map[uint64]error{
	400: errBotInvalidParameters,
	404: errBotNotFound,
	405: errBotNotClosable, // spot grid bot already cancelling or completed
	421: errBotUserBanned,
	429: errBotRateLimitExceeded,
	503: errBotNotClosable, // DCA bot in the middle of an investment cycle
}

// botStatusError returns nil when a bot result's status code reports success, and otherwise an error wrapping
// errBotRequestFailed, the code's documented error if it has one, and the details the result carries
func botStatusError(statusCode uint64, details ...string) error {
	if statusCode == 0 || statusCode == http.StatusOK {
		return nil
	}
	parts := make([]string, 0, len(details)+1)
	parts = append(parts, "status code "+strconv.FormatUint(statusCode, 10))
	for _, d := range details {
		if d != "" {
			parts = append(parts, d)
		}
	}
	msg := strings.Join(parts, ": ")
	if mapped, ok := botStatusErrors[statusCode]; ok {
		return fmt.Errorf("%w: %w: %s", errBotRequestFailed, mapped, msg)
	}
	return fmt.Errorf("%w: %s", errBotRequestFailed, msg)
}

// futuresComboBotBody validates the settings of a futures combo bot and returns their body
func futuresComboBotBody(arg *CreateFuturesComboBotRequest) (map[string]any, error) {
	if arg.Leverage <= 0 {
		return nil, errInvalidLeverage
	}
	if arg.InitialMargin <= 0 {
		return nil, errBotInitialMarginRequired
	}
	if arg.AdjustPositionMode == 0 {
		return nil, errAdjustPositionModeRequired
	}
	if len(arg.SymbolSettings) == 0 {
		return nil, errSymbolSettingsRequired
	}
	settings := make([]map[string]any, len(arg.SymbolSettings))
	for i := range arg.SymbolSettings {
		s := &arg.SymbolSettings[i]
		if s.Symbol == "" {
			return nil, fmt.Errorf("symbol setting %d: %w", i, errSymbolMissing)
		}
		if s.TargetPositionPercent <= 0 {
			return nil, fmt.Errorf("symbol setting %d: %w", i, errTargetPositionPercentRequired)
		}
		if s.Side == 0 {
			return nil, fmt.Errorf("symbol setting %d: %w", i, order.ErrSideIsInvalid)
		}
		item := make(map[string]any)
		item["symbol"] = s.Symbol
		item["target_position_percent"] = strconv.FormatFloat(s.TargetPositionPercent, 'f', -1, 64)
		item["side"] = s.Side
		settings[i] = item
	}
	body := make(map[string]any)
	body["leverage"] = strconv.FormatFloat(arg.Leverage, 'f', -1, 64)
	body["init_margin"] = strconv.FormatFloat(arg.InitialMargin, 'f', -1, 64)
	body["adjust_position_mode"] = arg.AdjustPositionMode
	body["symbol_settings"] = settings
	if arg.AdjustPositionPercent != 0 {
		body["adjust_position_percent"] = strconv.FormatFloat(arg.AdjustPositionPercent, 'f', -1, 64)
	}
	if arg.AdjustPositionTimeInterval != 0 {
		body["adjust_position_time_interval"] = int64(arg.AdjustPositionTimeInterval / time.Second)
	}
	if arg.SLPercent != 0 {
		body["sl_percent"] = strconv.FormatFloat(arg.SLPercent, 'f', -1, 64)
	}
	if arg.TPPercent != 0 {
		body["tp_percent"] = strconv.FormatFloat(arg.TPPercent, 'f', -1, 64)
	}
	if arg.TrailingStopPercent != 0 {
		body["trailing_stop_percent"] = strconv.FormatFloat(arg.TrailingStopPercent, 'f', -1, 64)
	}
	return body, nil
}

// GetFuturesComboBotLimit returns the accepted range of each futures combo bot setting and whether the given settings
// fall within them (Get Bot Parameter Limit)
func (e *Exchange) GetFuturesComboBotLimit(ctx context.Context, arg *GetFuturesComboBotLimitRequest) (*FuturesComboBotLimitResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	body, err := futuresComboBotBody(&arg.CreateFuturesComboBotRequest)
	if err != nil {
		return nil, err
	}
	if arg.NeedToSlippage {
		body["need_to_slippage"] = true
	}
	var resp *FuturesComboBotLimitResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/fcombobot/getlimit", nil, body, &resp, getFuturesComboBotLimitEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.CheckCode, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateFuturesComboBot creates a futures combo bot, which holds a portfolio of futures positions and rebalances it
// (Create Bot)
func (e *Exchange) CreateFuturesComboBot(ctx context.Context, arg *CreateFuturesComboBotRequest) (*CreateBotResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	body, err := futuresComboBotBody(arg)
	if err != nil {
		return nil, err
	}
	var resp *CreateBotResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/fcombobot/create", nil, body, &resp, createFuturesComboBotEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.BanReasonText, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// CloseFuturesComboBot closes a running futures combo bot, cancelling its orders and closing its positions (Close Bot)
func (e *Exchange) CloseFuturesComboBot(ctx context.Context, botID string) (*CloseBotResponse, error) {
	if botID == "" {
		return nil, errBotIDRequired
	}
	body := make(map[string]any)
	body["bot_id"] = botID
	var resp *CloseBotResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/fcombobot/close", nil, body, &resp, closeFuturesComboBotEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetFuturesComboBotDetail returns a futures combo bot's settings, state, balances and profit and loss (Get Bot Detail)
func (e *Exchange) GetFuturesComboBotDetail(ctx context.Context, botID string) (*FuturesComboBotDetailResponse, error) {
	if botID == "" {
		return nil, errBotIDRequired
	}
	body := make(map[string]any)
	body["bot_id"] = botID
	var resp *FuturesComboBotDetailResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/fcombobot/detail", nil, body, &resp, getFuturesComboBotDetailEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// ValidateFuturesGridInput returns the accepted range of each futures grid bot setting and whether the given settings
// fall within them (Validate Grid Input)
func (e *Exchange) ValidateFuturesGridInput(ctx context.Context, arg *ValidateFuturesGridInputRequest) (*ValidateFuturesGridInputResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol == "" {
		return nil, errSymbolMissing
	}
	if arg.CellNumber == 0 {
		return nil, errGridCellNumberRequired
	}
	if arg.MinPrice <= 0 {
		return nil, errGridMinPriceRequired
	}
	if arg.MaxPrice <= 0 {
		return nil, errGridMaxPriceRequired
	}
	if arg.Leverage <= 0 {
		return nil, errInvalidLeverage
	}
	if arg.GridType == 0 {
		return nil, errGridTypeRequired
	}
	if arg.GridMode == 0 {
		return nil, errGridModeRequired
	}
	body := make(map[string]any)
	body["symbol"] = arg.Symbol
	body["cell_number"] = arg.CellNumber
	body["min_price"] = strconv.FormatFloat(arg.MinPrice, 'f', -1, 64)
	body["max_price"] = strconv.FormatFloat(arg.MaxPrice, 'f', -1, 64)
	body["leverage"] = strconv.FormatFloat(arg.Leverage, 'f', -1, 64)
	body["grid_type"] = arg.GridType
	body["grid_mode"] = arg.GridMode
	if arg.StopLossPrice != 0 {
		body["stop_loss_price"] = strconv.FormatFloat(arg.StopLossPrice, 'f', -1, 64)
	}
	if arg.TakeProfitPrice != 0 {
		body["take_profit_price"] = strconv.FormatFloat(arg.TakeProfitPrice, 'f', -1, 64)
	}
	if arg.TPSLType != 0 {
		body["tp_sl_type"] = arg.TPSLType
	}
	if arg.EntryPrice != 0 {
		body["entry_price"] = strconv.FormatFloat(arg.EntryPrice, 'f', -1, 64)
	}
	if arg.StopLossPercent != 0 {
		body["stop_loss_per"] = strconv.FormatFloat(arg.StopLossPercent, 'f', -1, 64)
	}
	if arg.TakeProfitPercent != 0 {
		body["take_profit_per"] = strconv.FormatFloat(arg.TakeProfitPercent, 'f', -1, 64)
	}
	if arg.TrailingStopPercent != 0 {
		body["trailing_stop_per"] = strconv.FormatFloat(arg.TrailingStopPercent, 'f', -1, 64)
	}
	if arg.InitialMargin != 0 {
		body["init_margin"] = strconv.FormatFloat(arg.InitialMargin, 'f', -1, 64)
	}
	if arg.MoveUpPrice != 0 {
		body["move_up_price"] = strconv.FormatFloat(arg.MoveUpPrice, 'f', -1, 64)
	}
	if arg.MoveDownPrice != 0 {
		body["move_down_price"] = strconv.FormatFloat(arg.MoveDownPrice, 'f', -1, 64)
	}
	var resp *ValidateFuturesGridInputResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/fgridbot/validate", nil, body, &resp, validateFuturesGridInputEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.CheckCode, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateFuturesGridBot creates a futures grid bot, which places grid orders across a price range (Create Grid Bot)
func (e *Exchange) CreateFuturesGridBot(ctx context.Context, arg *CreateFuturesGridBotRequest) (*CreateFuturesGridBotResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol == "" {
		return nil, errSymbolMissing
	}
	if arg.CellNumber == 0 {
		return nil, errGridCellNumberRequired
	}
	if arg.MinPrice <= 0 {
		return nil, errGridMinPriceRequired
	}
	if arg.MaxPrice <= 0 {
		return nil, errGridMaxPriceRequired
	}
	if arg.Leverage <= 0 {
		return nil, errInvalidLeverage
	}
	if arg.GridType == 0 {
		return nil, errGridTypeRequired
	}
	if arg.GridMode == 0 {
		return nil, errGridModeRequired
	}
	if arg.TotalInvestment <= 0 {
		return nil, errGridTotalInvestmentRequired
	}
	body := make(map[string]any)
	body["symbol"] = arg.Symbol
	body["cell_number"] = arg.CellNumber
	body["min_price"] = strconv.FormatFloat(arg.MinPrice, 'f', -1, 64)
	body["max_price"] = strconv.FormatFloat(arg.MaxPrice, 'f', -1, 64)
	body["leverage"] = strconv.FormatFloat(arg.Leverage, 'f', -1, 64)
	body["grid_type"] = arg.GridType
	body["grid_mode"] = arg.GridMode
	body["total_investment"] = strconv.FormatFloat(arg.TotalInvestment, 'f', -1, 64)
	if arg.TakeProfitPercent != 0 {
		body["take_profit_per"] = strconv.FormatFloat(arg.TakeProfitPercent, 'f', -1, 64)
	}
	if arg.StopLossPercent != 0 {
		body["stop_loss_per"] = strconv.FormatFloat(arg.StopLossPercent, 'f', -1, 64)
	}
	if arg.TakeProfitPrice != 0 {
		body["take_profit_price"] = strconv.FormatFloat(arg.TakeProfitPrice, 'f', -1, 64)
	}
	if arg.StopLossPrice != 0 {
		body["stop_loss_price"] = strconv.FormatFloat(arg.StopLossPrice, 'f', -1, 64)
	}
	if arg.TPSLType != 0 {
		body["tp_sl_type"] = arg.TPSLType
	}
	if arg.EntryPrice != 0 {
		body["entry_price"] = strconv.FormatFloat(arg.EntryPrice, 'f', -1, 64)
	}
	if arg.TrailingStopPercent != 0 {
		body["trailing_stop_per"] = strconv.FormatFloat(arg.TrailingStopPercent, 'f', -1, 64)
	}
	if arg.MoveUpPrice != 0 {
		body["move_up_price"] = strconv.FormatFloat(arg.MoveUpPrice, 'f', -1, 64)
	}
	if arg.MoveDownPrice != 0 {
		body["move_down_price"] = strconv.FormatFloat(arg.MoveDownPrice, 'f', -1, 64)
	}
	var resp *CreateFuturesGridBotResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/fgridbot/create", nil, body, &resp, createFuturesGridBotEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.CheckCode, resp.BanReasonText, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// CloseFuturesGridBot closes a running futures grid bot, cancelling its orders and closing its position (Close Grid
// Bot)
func (e *Exchange) CloseFuturesGridBot(ctx context.Context, botID string) (*CloseFuturesGridBotResponse, error) {
	if botID == "" {
		return nil, errBotIDRequired
	}
	body := make(map[string]any)
	body["bot_id"] = botID
	var resp *CloseFuturesGridBotResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/fgridbot/close", nil, body, &resp, closeFuturesGridBotEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetFuturesGridBotDetail returns a futures grid bot's settings, state, position and profit and loss (Get Grid Bot
// Detail)
func (e *Exchange) GetFuturesGridBotDetail(ctx context.Context, botID string) (*FuturesGridBotDetailResponse, error) {
	if botID == "" {
		return nil, errBotIDRequired
	}
	body := make(map[string]any)
	body["bot_id"] = botID
	var resp *FuturesGridBotDetailResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/fgridbot/detail", nil, body, &resp, getFuturesGridBotDetailEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetFuturesMartingaleBotLimit returns the accepted range of each futures Martingale bot setting and whether the
// given settings fall within them (Get Bot Parameter Limit)
func (e *Exchange) GetFuturesMartingaleBotLimit(ctx context.Context, arg *GetFuturesMartingaleBotLimitRequest) (*FuturesMartingaleBotLimitResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol == "" {
		return nil, errSymbolMissing
	}
	if arg.MartingaleMode == 0 {
		return nil, errMartingaleModeRequired
	}
	if arg.Leverage <= 0 {
		return nil, errInvalidLeverage
	}
	body := make(map[string]any)
	body["symbol"] = arg.Symbol
	body["martingale_mode"] = arg.MartingaleMode
	body["leverage"] = strconv.FormatFloat(arg.Leverage, 'f', -1, 64)
	if arg.PriceFloatPercent != 0 {
		body["price_float_percent"] = strconv.FormatFloat(arg.PriceFloatPercent, 'f', -1, 64)
	}
	if arg.AddPositionPercent != 0 {
		body["add_position_percent"] = strconv.FormatFloat(arg.AddPositionPercent, 'f', -1, 64)
	}
	if arg.AddPositionNumber != 0 {
		body["add_position_num"] = arg.AddPositionNumber
	}
	if arg.InitialMargin != 0 {
		body["init_margin"] = strconv.FormatFloat(arg.InitialMargin, 'f', -1, 64)
	}
	if arg.RoundTPPercent != 0 {
		body["round_tp_percent"] = strconv.FormatFloat(arg.RoundTPPercent, 'f', -1, 64)
	}
	if arg.SLPercent != 0 {
		body["sl_percent"] = strconv.FormatFloat(arg.SLPercent, 'f', -1, 64)
	}
	if arg.EntryPrice != 0 {
		body["entry_price"] = strconv.FormatFloat(arg.EntryPrice, 'f', -1, 64)
	}
	if arg.NeedToSlippage {
		body["need_to_slippage"] = true
	}
	var resp *FuturesMartingaleBotLimitResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/fmartingalebot/getlimit", nil, body, &resp, getFuturesMartingaleBotLimitEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.CheckCode, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateFuturesMartingaleBot creates a futures Martingale bot, which adds to its position as the price moves against
// it and takes profit when the price reverses (Create Martingale Bot)
func (e *Exchange) CreateFuturesMartingaleBot(ctx context.Context, arg *CreateFuturesMartingaleBotRequest) (*CreateBotResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol == "" {
		return nil, errSymbolMissing
	}
	if arg.MartingaleMode == 0 {
		return nil, errMartingaleModeRequired
	}
	if arg.Leverage <= 0 {
		return nil, errInvalidLeverage
	}
	if arg.PriceFloatPercent <= 0 {
		return nil, errPriceFloatPercentRequired
	}
	if arg.AddPositionPercent <= 0 {
		return nil, errAddPositionPercentRequired
	}
	if arg.AddPositionNumber == 0 {
		return nil, errAddPositionNumberRequired
	}
	if arg.InitialMargin <= 0 {
		return nil, errBotInitialMarginRequired
	}
	if arg.RoundTPPercent <= 0 {
		return nil, errRoundTPPercentRequired
	}
	body := make(map[string]any)
	body["symbol"] = arg.Symbol
	body["martingale_mode"] = arg.MartingaleMode
	body["leverage"] = strconv.FormatFloat(arg.Leverage, 'f', -1, 64)
	body["price_float_percent"] = strconv.FormatFloat(arg.PriceFloatPercent, 'f', -1, 64)
	body["add_position_percent"] = strconv.FormatFloat(arg.AddPositionPercent, 'f', -1, 64)
	body["add_position_num"] = arg.AddPositionNumber
	body["init_margin"] = strconv.FormatFloat(arg.InitialMargin, 'f', -1, 64)
	body["round_tp_percent"] = strconv.FormatFloat(arg.RoundTPPercent, 'f', -1, 64)
	if arg.AutoCycleToggle != 0 {
		body["auto_cycle_toggle"] = arg.AutoCycleToggle
	}
	if arg.SLPercent != 0 {
		body["sl_percent"] = strconv.FormatFloat(arg.SLPercent, 'f', -1, 64)
	}
	if arg.EntryPrice != 0 {
		body["entry_price"] = strconv.FormatFloat(arg.EntryPrice, 'f', -1, 64)
	}
	var resp *CreateBotResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/fmartingalebot/create", nil, body, &resp, createFuturesMartingaleBotEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.BanReasonText, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// CloseFuturesMartingaleBot closes a running futures Martingale bot, cancelling its orders and closing its position
// (Close Martingale Bot)
func (e *Exchange) CloseFuturesMartingaleBot(ctx context.Context, botID string) (*CloseBotResponse, error) {
	if botID == "" {
		return nil, errBotIDRequired
	}
	body := make(map[string]any)
	body["bot_id"] = botID
	var resp *CloseBotResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/fmartingalebot/close", nil, body, &resp, closeFuturesMartingaleBotEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetFuturesMartingaleBotDetail returns a futures Martingale bot's settings, round progress, position and profit and
// loss (Get Martingale Bot Detail)
func (e *Exchange) GetFuturesMartingaleBotDetail(ctx context.Context, botID string) (*FuturesMartingaleBotDetailResponse, error) {
	if botID == "" {
		return nil, errBotIDRequired
	}
	body := make(map[string]any)
	body["bot_id"] = botID
	var resp *FuturesMartingaleBotDetailResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/fmartingalebot/detail", nil, body, &resp, getFuturesMartingaleBotDetailEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// ValidateSpotGridInput returns the accepted range of each spot grid bot setting and whether the given settings fall
// within them (Validate Grid Input). The docs call it a guest endpoint, but Bybit answers an unsigned request with
// retCode 10001 "apiKey is missing"
func (e *Exchange) ValidateSpotGridInput(ctx context.Context, arg *ValidateSpotGridInputRequest) (*ValidateSpotGridInputResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol == "" {
		return nil, errSymbolMissing
	}
	if arg.CellNumber == 0 {
		return nil, errGridCellNumberRequired
	}
	if arg.MinPrice <= 0 {
		return nil, errGridMinPriceRequired
	}
	if arg.MaxPrice <= 0 {
		return nil, errGridMaxPriceRequired
	}
	body := make(map[string]any)
	body["symbol"] = arg.Symbol
	body["cell_number"] = arg.CellNumber
	body["min_price"] = strconv.FormatFloat(arg.MinPrice, 'f', -1, 64)
	body["max_price"] = strconv.FormatFloat(arg.MaxPrice, 'f', -1, 64)
	if arg.InvestMode != 0 {
		body["invest_mode"] = arg.InvestMode
	}
	if arg.BaseInvestment != 0 {
		body["base_investment"] = strconv.FormatFloat(arg.BaseInvestment, 'f', -1, 64)
	}
	if arg.QuoteInvestment != 0 {
		body["quote_investment"] = strconv.FormatFloat(arg.QuoteInvestment, 'f', -1, 64)
	}
	if arg.StopLoss != 0 {
		body["stop_loss"] = strconv.FormatFloat(arg.StopLoss, 'f', -1, 64)
	}
	if arg.TakeProfit != 0 {
		body["take_profit"] = strconv.FormatFloat(arg.TakeProfit, 'f', -1, 64)
	}
	if arg.EntryPrice != 0 {
		body["entry_price"] = strconv.FormatFloat(arg.EntryPrice, 'f', -1, 64)
	}
	if arg.TrailingStopPercent != 0 {
		body["ts_percent"] = strconv.FormatFloat(arg.TrailingStopPercent, 'f', -1, 64)
	}
	if arg.EnableTrailing {
		body["enable_trailing"] = true
	}
	if arg.LimitUpPrice != 0 {
		body["limit_up_price"] = strconv.FormatFloat(arg.LimitUpPrice, 'f', -1, 64)
	}
	var resp *ValidateSpotGridInputResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/grid/validate-input", nil, body, &resp, validateSpotGridInputEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.CheckCode, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateSpotGridBot creates a spot grid bot, which places grid orders across a price range (Create Grid Bot)
func (e *Exchange) CreateSpotGridBot(ctx context.Context, arg *CreateSpotGridBotRequest) (*CreateSpotGridBotResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol == "" {
		return nil, errSymbolMissing
	}
	if arg.CellNumber == 0 {
		return nil, errGridCellNumberRequired
	}
	if arg.MinPrice <= 0 {
		return nil, errGridMinPriceRequired
	}
	if arg.MaxPrice <= 0 {
		return nil, errGridMaxPriceRequired
	}
	body := make(map[string]any)
	body["symbol"] = arg.Symbol
	body["cell_number"] = arg.CellNumber
	body["min_price"] = strconv.FormatFloat(arg.MinPrice, 'f', -1, 64)
	body["max_price"] = strconv.FormatFloat(arg.MaxPrice, 'f', -1, 64)
	if arg.InvestMode != 0 {
		body["invest_mode"] = arg.InvestMode
	}
	if arg.BaseInvestment != 0 {
		body["base_investment"] = strconv.FormatFloat(arg.BaseInvestment, 'f', -1, 64)
	}
	if arg.QuoteInvestment != 0 {
		body["quote_investment"] = strconv.FormatFloat(arg.QuoteInvestment, 'f', -1, 64)
	}
	if arg.EntryPrice != 0 {
		body["entry_price"] = strconv.FormatFloat(arg.EntryPrice, 'f', -1, 64)
	}
	if arg.StopLossPrice != 0 {
		body["stop_loss_price"] = strconv.FormatFloat(arg.StopLossPrice, 'f', -1, 64)
	}
	if arg.TakeProfitPrice != 0 {
		body["take_profit_price"] = strconv.FormatFloat(arg.TakeProfitPrice, 'f', -1, 64)
	}
	if arg.TrailingStopPercent != 0 {
		body["ts_percent"] = strconv.FormatFloat(arg.TrailingStopPercent, 'f', -1, 64)
	}
	if arg.EnableTrailing {
		body["enable_trailing"] = true
	}
	if arg.LimitUpPrice != 0 {
		body["limit_up_price"] = strconv.FormatFloat(arg.LimitUpPrice, 'f', -1, 64)
	}
	var resp *CreateSpotGridBotResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/grid/create-grid", nil, body, &resp, createSpotGridBotEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.BanReasonText, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// CloseSpotGridBot closes a new or running spot grid bot, settling its assets by closeMode: 2 base token, 3 quote
// token or 4 both as held (Close Grid Bot). close_mode is sent as a number although the docs type it as a string:
// bot enums parse from numbers, as Close DCA Bot's request example shows, but may not parse from numeric strings
func (e *Exchange) CloseSpotGridBot(ctx context.Context, gridID string, closeMode uint64) (*CloseBotResponse, error) {
	if gridID == "" {
		return nil, errGridIDRequired
	}
	if closeMode == 0 {
		return nil, errBotCloseModeRequired
	}
	body := make(map[string]any)
	body["grid_id"] = gridID
	body["close_mode"] = closeMode
	var resp *CloseBotResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/grid/close-grid", nil, body, &resp, closeSpotGridBotEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetSpotGridBotDetail returns a spot grid bot's settings, state and profit (Get Grid Bot Detail)
func (e *Exchange) GetSpotGridBotDetail(ctx context.Context, gridID string) (*SpotGridBotDetailResponse, error) {
	if gridID == "" {
		return nil, errGridIDRequired
	}
	body := make(map[string]any)
	body["grid_id"] = gridID
	var resp *SpotGridBotDetailResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/grid/query-grid-detail", nil, body, &resp, getSpotGridBotDetailEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateDCABot creates a DCA bot, which buys a fixed quote amount of each of its pairs at every interval (Create DCA
// Bot)
func (e *Exchange) CreateDCABot(ctx context.Context, arg *CreateDCABotRequest) (*CreateBotResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Frequency < time.Second {
		return nil, errDCAFrequencyRequired
	}
	if arg.QuoteCoin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if len(arg.Pairs) == 0 {
		return nil, errDCAPairsRequired
	}
	pairs := make([]map[string]any, len(arg.Pairs))
	for i := range arg.Pairs {
		if arg.Pairs[i].Base.IsEmpty() {
			return nil, fmt.Errorf("pair %d: %w", i, currency.ErrCurrencyCodeEmpty)
		}
		if arg.Pairs[i].Amount <= 0 {
			return nil, fmt.Errorf("pair %d: %w", i, order.ErrAmountIsInvalid)
		}
		item := make(map[string]any)
		item["base"] = arg.Pairs[i].Base.Upper().String()
		item["amount"] = strconv.FormatFloat(arg.Pairs[i].Amount, 'f', -1, 64)
		pairs[i] = item
	}
	parameters := make(map[string]any)
	parameters["frequency_in_second"] = int64(arg.Frequency / time.Second)
	parameters["quote_coin"] = arg.QuoteCoin.Upper().String()
	parameters["pairs"] = pairs
	if arg.MaxInvestAmount != 0 {
		parameters["max_invest_amount"] = strconv.FormatFloat(arg.MaxInvestAmount, 'f', -1, 64)
	}
	body := make(map[string]any)
	body["parameters"] = parameters
	var resp *CreateBotResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/dca/create-bot", nil, body, &resp, createDCABotEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.BanReasonText, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}

// CloseDCABot closes a DCA bot, settling its assets by closeMode: 2 base token or 3 quote token (Close DCA Bot).
// close_mode is sent as a number although the docs type it as a string: their request example sends a number, and
// the result names the mode's enum, which parses from a number but may not parse from a numeric string
func (e *Exchange) CloseDCABot(ctx context.Context, botID string, closeMode uint64) (*CloseDCABotResponse, error) {
	if botID == "" {
		return nil, errBotIDRequired
	}
	if closeMode == 0 {
		return nil, errBotCloseModeRequired
	}
	body := make(map[string]any)
	body["bot_id"] = botID
	body["close_mode"] = closeMode
	var resp *CloseDCABotResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/dca/close-bot", nil, body, &resp, closeDCABotEPL); err != nil {
		return nil, err
	}
	if err := botStatusError(resp.StatusCode, resp.DebugMessage); err != nil {
		return nil, err
	}
	return resp, nil
}
