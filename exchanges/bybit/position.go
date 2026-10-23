package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Position errors
var (
	errMarginNotSet      = errors.New("margin not set")
	errFromUIDRequired   = errors.New("from UID required")
	errToUIDRequired     = errors.New("to UID required")
	errNoPositionsToMove = errors.New("no positions to move")
)

// GetPositionInfo returns real-time position data, such as size and cumulative realised PNL (Get Position Info)
func (e *Exchange) GetPositionInfo(ctx context.Context, r *PositionInfoRequest) (*PositionInfoResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Category == "" {
		return nil, errCategoryNotSet
	}
	params := url.Values{}
	params.Set("category", r.Category)
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if !r.BaseCoin.IsEmpty() {
		params.Set("baseCoin", r.BaseCoin.Upper().String())
	}
	if !r.SettleCoin.IsEmpty() {
		params.Set("settleCoin", r.SettleCoin.Upper().String())
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	if r.Cursor != "" {
		params.Set("cursor", r.Cursor)
	}
	var resp *PositionInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/position/list", params, nil, &resp, getPositionInfoEPL)
}

// SetLeverageLevel sets a symbol's leverage, from 1 to the maximum its risk limit allows (Set Leverage)
func (e *Exchange) SetLeverageLevel(ctx context.Context, r *SetLeverageRequest) error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	switch {
	case r.Category == "":
		return errCategoryNotSet
	case r.Symbol == "":
		return errSymbolMissing
	case r.BuyLeverage <= 0:
		return fmt.Errorf("%w: buy leverage %v", errInvalidLeverage, r.BuyLeverage)
	case r.SellLeverage <= 0:
		return fmt.Errorf("%w: sell leverage %v", errInvalidLeverage, r.SellLeverage)
	}
	body := make(map[string]any)
	body["category"] = r.Category
	body["symbol"] = r.Symbol
	body["buyLeverage"] = strconv.FormatFloat(r.BuyLeverage, 'f', -1, 64)
	body["sellLeverage"] = strconv.FormatFloat(r.SellLeverage, 'f', -1, 64)
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/position/set-leverage", nil, body, nil, setLeverageLevelEPL)
}

// GetFuturesLeverage returns the leverage set for futures symbols, whether or not they have a position
// (Get Futures Leverage)
func (e *Exchange) GetFuturesLeverage(ctx context.Context, category, symbol string) (*FuturesLeverageResponse, error) {
	if category == "" {
		return nil, errCategoryNotSet
	}
	params := url.Values{}
	params.Set("category", category)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *FuturesLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/position/symbol-info", params, nil, &resp, defaultEPL)
}

// SwitchPositionMode switches a symbol, or every symbol of a settle coin, between one-way and hedge mode
// (Switch Position Mode)
func (e *Exchange) SwitchPositionMode(ctx context.Context, r *SwitchPositionModeRequest) error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	switch {
	case r.Category == "":
		return errCategoryNotSet
	case r.Symbol == "" && r.Coin.IsEmpty():
		return errEitherSymbolOrCoinRequired
	}
	body := make(map[string]any)
	body["category"] = r.Category
	if r.Symbol != "" {
		body["symbol"] = r.Symbol
	}
	if !r.Coin.IsEmpty() {
		body["coin"] = r.Coin.Upper().String()
	}
	body["mode"] = r.Mode
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/position/switch-mode", nil, body, nil, switchPositionModeEPL)
}

// SetTradingStop sets the take profit, stop loss or trailing stop of a position (Set Trading Stop)
func (e *Exchange) SetTradingStop(ctx context.Context, r *TradingStopRequest) error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	switch {
	case r.Category == "":
		return errCategoryNotSet
	case r.Symbol == "":
		return errSymbolMissing
	case r.TPSLMode == "":
		return errTakeProfitOrStopLossModeMissing
	}
	body := make(map[string]any)
	body["category"] = r.Category
	body["symbol"] = r.Symbol
	body["tpslMode"] = r.TPSLMode
	body["positionIdx"] = r.PositionIndex
	if r.TakeProfit != nil {
		body["takeProfit"] = strconv.FormatFloat(*r.TakeProfit, 'f', -1, 64)
	}
	if r.StopLoss != nil {
		body["stopLoss"] = strconv.FormatFloat(*r.StopLoss, 'f', -1, 64)
	}
	if r.TrailingStop != nil {
		body["trailingStop"] = strconv.FormatFloat(*r.TrailingStop, 'f', -1, 64)
	}
	if r.TPTriggerBy != "" {
		body["tpTriggerBy"] = r.TPTriggerBy
	}
	if r.SLTriggerBy != "" {
		body["slTriggerBy"] = r.SLTriggerBy
	}
	if r.ActivePrice != 0 {
		body["activePrice"] = strconv.FormatFloat(r.ActivePrice, 'f', -1, 64)
	}
	if r.TPSize != 0 {
		body["tpSize"] = strconv.FormatFloat(r.TPSize, 'f', -1, 64)
	}
	if r.SLSize != 0 {
		body["slSize"] = strconv.FormatFloat(r.SLSize, 'f', -1, 64)
	}
	if r.TPLimitPrice != 0 {
		body["tpLimitPrice"] = strconv.FormatFloat(r.TPLimitPrice, 'f', -1, 64)
	}
	if r.SLLimitPrice != 0 {
		body["slLimitPrice"] = strconv.FormatFloat(r.SLLimitPrice, 'f', -1, 64)
	}
	if r.TPOrderType != "" {
		body["tpOrderType"] = r.TPOrderType
	}
	if r.SLOrderType != "" {
		body["slOrderType"] = r.SLOrderType
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/position/trading-stop", nil, body, nil, setTradingStopEPL)
}

// SetAutoAddMargin turns automatically adding margin on or off for an isolated margin position (Set Auto Add Margin)
func (e *Exchange) SetAutoAddMargin(ctx context.Context, r *SetAutoAddMarginRequest) error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	switch {
	case r.Category == "":
		return errCategoryNotSet
	case r.Symbol == "":
		return errSymbolMissing
	}
	body := make(map[string]any)
	body["category"] = r.Category
	body["symbol"] = r.Symbol
	body["autoAddMargin"] = 0
	if r.AutoAddMargin {
		body["autoAddMargin"] = 1
	}
	if r.PositionIndex != 0 {
		body["positionIdx"] = r.PositionIndex
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/position/set-auto-add-margin", nil, body, nil, setAutoAddMarginEPL)
}

// AddOrReduceMargin adds margin to, or removes margin from, an isolated margin position (Add Or Reduce Margin)
func (e *Exchange) AddOrReduceMargin(ctx context.Context, r *AddOrReduceMarginRequest) (*AddOrReduceMarginResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.Category == "":
		return nil, errCategoryNotSet
	case r.Symbol == "":
		return nil, errSymbolMissing
	case r.Margin == 0:
		return nil, errMarginNotSet
	}
	body := make(map[string]any)
	body["category"] = r.Category
	body["symbol"] = r.Symbol
	body["margin"] = strconv.FormatFloat(r.Margin, 'f', -1, 64)
	if r.PositionIndex != 0 {
		body["positionIdx"] = r.PositionIndex
	}
	var resp *AddOrReduceMarginResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/position/add-margin", nil, body, &resp, addOrReduceMarginEPL)
}

// GetClosedPNL returns the profit and loss of orders that closed positions in the last two years, newest first
// (Get Closed PnL)
func (e *Exchange) GetClosedPNL(ctx context.Context, r *ClosedPNLRequest) (*ClosedPNLResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Category == "" {
		return nil, errCategoryNotSet
	}
	if !r.StartTime.IsZero() && !r.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(r.StartTime, r.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", r.Category)
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if !r.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(r.StartTime.UnixMilli(), 10))
	}
	if !r.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(r.EndTime.UnixMilli(), 10))
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	if r.Cursor != "" {
		params.Set("cursor", r.Cursor)
	}
	var resp *ClosedPNLResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/position/closed-pnl", params, nil, &resp, getClosedPNLEPL)
}

// GetClosedOptionsPositions returns the options positions closed in the last six months, newest first
// (Get Closed Options Positions)
func (e *Exchange) GetClosedOptionsPositions(ctx context.Context, r *ClosedOptionsPositionsRequest) (*ClosedOptionsPositionsResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Category == "" {
		return nil, errCategoryNotSet
	}
	if !r.StartTime.IsZero() && !r.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(r.StartTime, r.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", r.Category)
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if !r.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(r.StartTime.UnixMilli(), 10))
	}
	if !r.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(r.EndTime.UnixMilli(), 10))
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	if r.Cursor != "" {
		params.Set("cursor", r.Cursor)
	}
	var resp *ClosedOptionsPositionsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/position/get-closed-positions", params, nil, &resp, getClosedOptionsPositionsEPL)
}

// MovePositions moves positions between a master account and its sub-accounts as a fee-free block trade. Only a master
// account's API key may call it (Move Position)
func (e *Exchange) MovePositions(ctx context.Context, r *MovePositionsRequest) (*MovePositionsResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.FromUID == 0:
		return nil, errFromUIDRequired
	case r.ToUID == 0:
		return nil, errToUIDRequired
	case len(r.List) == 0:
		return nil, errNoPositionsToMove
	}
	legs := make([]map[string]any, len(r.List))
	for i := range r.List {
		switch {
		case r.List[i].Category == "":
			return nil, fmt.Errorf("list %d: %w", i, errCategoryNotSet)
		case r.List[i].Symbol == "":
			return nil, fmt.Errorf("list %d: %w", i, errSymbolMissing)
		case r.List[i].Price <= 0:
			return nil, fmt.Errorf("list %d: %w", i, limits.ErrPriceBelowMin)
		case r.List[i].Side == "":
			return nil, fmt.Errorf("list %d: %w", i, order.ErrSideIsInvalid)
		case r.List[i].Quantity <= 0:
			return nil, fmt.Errorf("list %d: %w", i, limits.ErrAmountBelowMin)
		}
		leg := make(map[string]any)
		leg["category"] = r.List[i].Category
		leg["symbol"] = r.List[i].Symbol
		leg["price"] = strconv.FormatFloat(r.List[i].Price, 'f', -1, 64)
		leg["side"] = r.List[i].Side
		leg["qty"] = strconv.FormatFloat(r.List[i].Quantity, 'f', -1, 64)
		legs[i] = leg
	}
	body := make(map[string]any)
	body["fromUid"] = strconv.FormatUint(r.FromUID, 10)
	body["toUid"] = strconv.FormatUint(r.ToUID, 10)
	body["list"] = legs
	var resp *MovePositionsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/position/move-positions", nil, body, &resp, movePositionsEPL)
}

// GetMovePositionHistory returns the positions moved by Move Position. Only a master account's API key may call it
// (Get Move Position History)
func (e *Exchange) GetMovePositionHistory(ctx context.Context, r *MovePositionHistoryRequest) (*MovePositionHistoryResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if !r.StartTime.IsZero() && !r.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(r.StartTime, r.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if r.Category != "" {
		params.Set("category", r.Category)
	}
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if !r.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(r.StartTime.UnixMilli(), 10))
	}
	if !r.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(r.EndTime.UnixMilli(), 10))
	}
	if r.Status != "" {
		params.Set("status", r.Status)
	}
	if r.BlockTradeID != "" {
		params.Set("blockTradeId", r.BlockTradeID)
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	if r.Cursor != "" {
		params.Set("cursor", r.Cursor)
	}
	var resp *MovePositionHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/position/move-history", params, nil, &resp, getMovePositionHistoryEPL)
}

// ConfirmNewRiskLimit asks Bybit to recalculate a reduce-only position's risk limit after the account adjusted it,
// removing the reduce-only mark when it passes (Confirm New Risk Limit)
func (e *Exchange) ConfirmNewRiskLimit(ctx context.Context, category, symbol string) error {
	switch {
	case category == "":
		return errCategoryNotSet
	case symbol == "":
		return errSymbolMissing
	}
	body := make(map[string]any)
	body["category"] = category
	body["symbol"] = symbol
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/position/confirm-pending-mmr", nil, body, nil, confirmNewRiskLimitEPL)
}
