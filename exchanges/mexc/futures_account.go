package mexc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
)

const (
	// futuresFeeDeductionDealsLimit is the most deals a fee deduction details request selects
	futuresFeeDeductionDealsLimit = 20
	// futuresHistoryWindow is the widest window the futures history endpoints serve in one request
	futuresHistoryWindow = 90 * 24 * time.Hour
)

// setFuturesTimeWindow sets a futures history request's window in Unix milliseconds after checking it; either bound
// may be left unset, and the venue serves at most 90 days at a time
func setFuturesTimeWindow(params url.Values, start, end time.Time) error {
	if !start.IsZero() && !end.IsZero() {
		if err := common.StartEndTimeCheck(start, end); err != nil {
			return err
		}
		if end.Sub(start) > futuresHistoryWindow {
			return fmt.Errorf("%w: %s to %s", errHistoryWindowTooWide, start, end)
		}
	}
	if !start.IsZero() {
		params.Set("start_time", strconv.FormatInt(start.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end_time", strconv.FormatInt(end.UnixMilli(), 10))
	}
	return nil
}

// validPositionType reports whether a position type is long or short; 0 selects both where a filter is optional
func validPositionType(positionType uint8) bool {
	return positionType == FuturesPositionTypeLong || positionType == FuturesPositionTypeShort
}

// GetFuturesAccountAssets returns the balances of every currency in the futures account
func (e *Exchange) GetFuturesAccountAssets(ctx context.Context) ([]*FuturesAccountAssetResponse, error) {
	var resp []*FuturesAccountAssetResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresAccountAssetsEPL, http.MethodGet, "private/account/assets", nil, nil, &resp, true)
}

// GetFuturesAccountAsset returns the balances of a currency in the futures account
func (e *Exchange) GetFuturesAccountAsset(ctx context.Context, ccy currency.Code) (*FuturesAccountAssetResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *FuturesAccountAssetResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresAccountAssetEPL, http.MethodGet, "private/account/asset/"+ccy.Upper().String(), nil, nil, &resp, true)
}

// GetFuturesAssetTransferRecords returns a page of transfers to and from the futures account, newest first; the page
// size is at most 100
func (e *Exchange) GetFuturesAssetTransferRecords(ctx context.Context, req *FuturesTransferRecordsRequest) (*FuturesPageResponse[FuturesTransferRecordResponse], error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	if req.State != "" {
		params.Set("state", req.State)
	}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if err := setFuturesPage(params, &req.FuturesPageRequest, futuresDefaultPageSizeLimit); err != nil {
		return nil, err
	}
	var resp *FuturesPageResponse[FuturesTransferRecordResponse]
	return resp, e.SendFuturesHTTPRequest(ctx, futuresTransferRecordsEPL, http.MethodGet, "private/account/transfer_record", params, nil, &resp, true)
}

// GetFuturesProfitRate returns the account's profit rate over a day, FuturesProfitRateDay, or a week,
// FuturesProfitRateWeek
func (e *Exchange) GetFuturesProfitRate(ctx context.Context, period uint8) (*FuturesProfitRateResponse, error) {
	if period != FuturesProfitRateDay && period != FuturesProfitRateWeek {
		return nil, fmt.Errorf("%w: %d", errInvalidProfitRateType, period)
	}
	var resp *FuturesProfitRateResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresProfitRateEPL, http.MethodGet, "private/account/profit_rate/"+strconv.FormatUint(uint64(period), 10), nil, nil, &resp, true)
}

// GetFuturesFeeDeductionConfigs returns the currencies which can pay futures fees and their discounts
func (e *Exchange) GetFuturesFeeDeductionConfigs(ctx context.Context) ([]*FuturesFeeDeductionConfigResponse, error) {
	var resp []*FuturesFeeDeductionConfigResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresFeeDeductionConfigsEPL, http.MethodGet, "private/account/feeDeductConfigs", nil, nil, &resp, true)
}

// GetFuturesSpotDiscountConfigs returns the spot holdings discount configurations which apply to futures fees
func (e *Exchange) GetFuturesSpotDiscountConfigs(ctx context.Context) ([]*FuturesDiscountConfigResponse, error) {
	var resp []*FuturesDiscountConfigResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresDiscountConfigsEPL, http.MethodGet, "private/account/config/contractFeeDiscountConfig", nil, nil, &resp, true)
}

// GetFuturesFeeDeductionDetails returns the fees of a contract's deals and how they were deducted
func (e *Exchange) GetFuturesFeeDeductionDetails(ctx context.Context, req *FuturesFeeDeductionDetailsRequest) ([]*FuturesFeeDeductionDetailResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if len(req.DealIDs) > futuresFeeDeductionDealsLimit {
		return nil, fmt.Errorf("%w: %d deals, at most %d", errTooManyOrders, len(req.DealIDs), futuresFeeDeductionDealsLimit)
	}
	params := url.Values{}
	params.Set("symbol", req.Symbol.String())
	if len(req.DealIDs) > 0 {
		params.Set("ids", strings.Join(req.DealIDs, ","))
	}
	if err := setFuturesTimeWindow(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if err := setFuturesPage(params, &req.FuturesPageRequest, futuresDefaultPageSizeLimit); err != nil {
		return nil, err
	}
	var resp []*FuturesFeeDeductionDetailResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresFeeDeductionDetailsEPL, http.MethodGet, "private/order/fee_details", params, nil, &resp, true)
}

// GetFuturesDiscountType reports whether the account's futures fees use the spot holdings discount or the futures
// fee deduction
func (e *Exchange) GetFuturesDiscountType(ctx context.Context) (*FuturesDiscountTypeResponse, error) {
	var resp *FuturesDiscountTypeResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresDiscountTypeEPL, http.MethodGet, "private/account/discountType", nil, nil, &resp, true)
}

// GetFuturesFeeStatistics returns the futures fees paid and saved over the last 30 days
func (e *Exchange) GetFuturesFeeStatistics(ctx context.Context) (*FuturesFeeStatisticsResponse, error) {
	var resp *FuturesFeeStatisticsResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresFeeStatisticsEPL, http.MethodGet, "private/account/asset_book/order_deal_fee/total", nil, nil, &resp, true)
}

// GetFuturesHistoricalPositions returns a page of the account's closed positions; the page size is at most 100
func (e *Exchange) GetFuturesHistoricalPositions(ctx context.Context, req *FuturesPositionsRequest) ([]*FuturesPositionResponse, error) {
	params, err := futuresPositionsParams(req)
	if err != nil {
		return nil, err
	}
	var resp pagedList[*FuturesPositionResponse]
	return resp, e.SendFuturesHTTPRequest(ctx, futuresHistoricalPositionsEPL, http.MethodGet, "private/position/list/history_positions", params, nil, &resp, true)
}

// futuresPositionsParams returns the parameters of a historical positions or funding fees request
func futuresPositionsParams(req *FuturesPositionsRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.PositionType != 0 && !validPositionType(req.PositionType) {
		return nil, fmt.Errorf("%w: %d", errInvalidPositionType, req.PositionType)
	}
	params := url.Values{}
	if !req.Symbol.IsEmpty() {
		params.Set("symbol", req.Symbol.String())
	}
	if req.PositionID != "" {
		params.Set("position_id", req.PositionID)
	}
	if req.PositionType != 0 {
		params.Set("position_type", strconv.FormatUint(uint64(req.PositionType), 10))
	}
	if err := setFuturesTimeWindow(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if err := setFuturesPage(params, &req.FuturesPageRequest, futuresDefaultPageSizeLimit); err != nil {
		return nil, err
	}
	return params, nil
}

// GetFuturesOpenPositions returns the account's open positions, of a contract when symbol is set or one position
// when positionID is
func (e *Exchange) GetFuturesOpenPositions(ctx context.Context, symbol currency.Pair, positionID string) ([]*FuturesPositionResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	if positionID != "" {
		params.Set("positionId", positionID)
	}
	var resp []*FuturesPositionResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresOpenPositionsEPL, http.MethodGet, "private/position/open_positions", params, nil, &resp, true)
}

// GetFuturesFundingFees returns a page of the funding fees the account's positions paid or received; the page size
// is at most 100
func (e *Exchange) GetFuturesFundingFees(ctx context.Context, req *FuturesPositionsRequest) ([]*FuturesFundingFeeResponse, error) {
	params, err := futuresPositionsParams(req)
	if err != nil {
		return nil, err
	}
	var resp pagedList[*FuturesFundingFeeResponse]
	return resp, e.SendFuturesHTTPRequest(ctx, futuresFundingFeesEPL, http.MethodGet, "private/position/funding_records", params, nil, &resp, true)
}

// GetFuturesRiskLimits returns the risk limits of each position side, by contract symbol, of a contract when symbol
// is set and of every contract otherwise
func (e *Exchange) GetFuturesRiskLimits(ctx context.Context, symbol currency.Pair) (map[string][]*FuturesRiskLimitResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp map[string][]*FuturesRiskLimitResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresRiskLimitsEPL, http.MethodGet, "private/account/risk_limit", params, nil, &resp, true)
}

// GetFuturesFeeDetails returns the account's futures fee rates, for a contract when symbol is set
func (e *Exchange) GetFuturesFeeDetails(ctx context.Context, symbol currency.Pair) (*FuturesFeeDetailsResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp *FuturesFeeDetailsResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresFeeDetailsEPL, http.MethodGet, "private/account/tiered_fee_rate/v2", params, nil, &resp, true)
}

// ChangeFuturesPositionMargin adds margin to or removes margin from an isolated position
func (e *Exchange) ChangeFuturesPositionMargin(ctx context.Context, req *FuturesChangeMarginRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.PositionID == 0 {
		return errPositionIDRequired
	}
	if req.Amount <= 0 {
		return fmt.Errorf("%w: %v", limits.ErrAmountBelowMin, req.Amount)
	}
	if req.Type != FuturesMarginAdd && req.Type != FuturesMarginSubtract {
		return fmt.Errorf("%w: %q", errInvalidMarginChangeType, req.Type)
	}
	return e.SendFuturesHTTPRequest(ctx, futuresChangeMarginEPL, http.MethodPost, "private/position/change_margin", nil, req, nil, true)
}

// SetFuturesAutoAddMargin turns the automatic addition of margin to an isolated position on or off
func (e *Exchange) SetFuturesAutoAddMargin(ctx context.Context, positionID uint64, enabled bool) error {
	if positionID == 0 {
		return errPositionIDRequired
	}
	body := &FuturesAutoAddMarginRequest{PositionID: positionID, IsEnabled: enabled}
	return e.SendFuturesHTTPRequest(ctx, futuresAutoAddMarginEPL, http.MethodPost, "private/position/change_auto_add_im", nil, body, nil, true)
}

// GetFuturesLeverage returns the leverage of each side of a contract's positions
func (e *Exchange) GetFuturesLeverage(ctx context.Context, symbol currency.Pair) ([]*FuturesLeverageResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp []*FuturesLeverageResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresLeverageEPL, http.MethodGet, "private/position/leverage", params, nil, &resp, true)
}

// ChangeFuturesLeverage sets the leverage of an open position, or of a contract's position side and margin mode
// when no position is open
func (e *Exchange) ChangeFuturesLeverage(ctx context.Context, req *FuturesChangeLeverageRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.Leverage == 0 {
		return errInvalidLeverage
	}
	if req.PositionID == 0 {
		if req.Symbol == "" {
			return currency.ErrSymbolStringEmpty
		}
		if req.OpenType != FuturesOpenTypeIsolated && req.OpenType != FuturesOpenTypeCross {
			return fmt.Errorf("%w: %d", errInvalidOpenType, req.OpenType)
		}
		if !validPositionType(req.PositionType) {
			return fmt.Errorf("%w: %d", errInvalidPositionType, req.PositionType)
		}
	}
	return e.SendFuturesHTTPRequest(ctx, futuresChangeLeverageEPL, http.MethodPost, "private/position/change_leverage", nil, req, nil, true)
}

// GetFuturesPositionMode returns the account's position mode, FuturesPositionModeHedge or FuturesPositionModeOneWay
func (e *Exchange) GetFuturesPositionMode(ctx context.Context) (uint8, error) {
	var resp uint8
	return resp, e.SendFuturesHTTPRequest(ctx, futuresPositionModeEPL, http.MethodGet, "private/position/position_mode", nil, nil, &resp, true)
}

// ChangeFuturesPositionMode sets the account's position mode. The venue refuses a change while any order, plan
// order or position is open, and a change to the one-way mode resets the risk limit level to 1
func (e *Exchange) ChangeFuturesPositionMode(ctx context.Context, positionMode uint8) error {
	if positionMode != FuturesPositionModeHedge && positionMode != FuturesPositionModeOneWay {
		return fmt.Errorf("%w: %d", errInvalidPositionMode, positionMode)
	}
	body := &FuturesPositionModeRequest{PositionMode: positionMode}
	return e.SendFuturesHTTPRequest(ctx, futuresChangePositionModeEPL, http.MethodPost, "private/position/change_position_mode", nil, body, nil, true)
}

// ChangeFuturesMultiAssetMode turns the multi-asset margin mode on or off
func (e *Exchange) ChangeFuturesMultiAssetMode(ctx context.Context, enabled bool) error {
	return e.sendFuturesHTTPRequest(ctx, futuresMultiAssetModeEPL, http.MethodPost, "private/multiAssets/changeMultiAssetMode/"+strconv.FormatBool(enabled), "application/x-www-form-urlencoded", nil, nil, nil, true)
}

// GetFuturesSelfTradePreventionGroups returns the market maker account's self-trade prevention groups, one when
// configName is set
func (e *Exchange) GetFuturesSelfTradePreventionGroups(ctx context.Context, configName string) ([]*FuturesSelfTradePreventionGroupResponse, error) {
	params := url.Values{}
	if configName != "" {
		params.Set("configName", configName)
	}
	var resp []*FuturesSelfTradePreventionGroupResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresSelfTradePreventionGroupsEPL, http.MethodGet, "private/market_maker/self_trade/blacklist", params, nil, &resp, true)
}

// GetFuturesSelfTradePreventionGroup returns the self-trade prevention group the account belongs to
func (e *Exchange) GetFuturesSelfTradePreventionGroup(ctx context.Context) (*FuturesSelfTradePreventionGroupResponse, error) {
	var resp *FuturesSelfTradePreventionGroupResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresSelfTradePreventionGroupEPL, http.MethodGet, "private/market_maker/self_trade/blacklist/search", nil, nil, &resp, true)
}

// CreateFuturesSelfTradePreventionGroup creates a self-trade prevention group of at least two sub-accounts and
// returns its id
func (e *Exchange) CreateFuturesSelfTradePreventionGroup(ctx context.Context, req *FuturesSelfTradePreventionGroupRequest) (int64, error) {
	if err := validateSelfTradePreventionGroup(req, true); err != nil {
		return 0, err
	}
	var resp int64
	return resp, e.SendFuturesHTTPRequest(ctx, futuresCreateSelfTradePreventionGroupEPL, http.MethodPost, "private/market_maker/self_trade/blacklist/create", nil, req, &resp, true)
}

// UpdateFuturesSelfTradePreventionGroup replaces the members of a self-trade prevention group
func (e *Exchange) UpdateFuturesSelfTradePreventionGroup(ctx context.Context, req *FuturesSelfTradePreventionGroupRequest) error {
	if err := validateSelfTradePreventionGroup(req, false); err != nil {
		return err
	}
	return e.SendFuturesHTTPRequest(ctx, futuresUpdateSelfTradePreventionGroupEPL, http.MethodPost, "private/market_maker/self_trade/blacklist/update", nil, req, nil, true)
}

// DeleteFuturesSelfTradePreventionGroup deletes a self-trade prevention group
func (e *Exchange) DeleteFuturesSelfTradePreventionGroup(ctx context.Context, configName string) error {
	if configName == "" {
		return errSelfTradePreventionGroupNameRequired
	}
	body := &FuturesSelfTradePreventionGroupRequest{ConfigName: configName}
	return e.SendFuturesHTTPRequest(ctx, futuresDeleteSelfTradePreventionGroupEPL, http.MethodPost, "private/market_maker/self_trade/blacklist/delete", nil, body, nil, true)
}

// validateSelfTradePreventionGroup checks a self-trade prevention group request; a new group needs at least two
// members
func validateSelfTradePreventionGroup(req *FuturesSelfTradePreventionGroupRequest, creating bool) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.ConfigName == "" {
		return errSelfTradePreventionGroupNameRequired
	}
	if len(req.Blacklist) == 0 {
		return errUIDRequired
	}
	if creating && len(req.Blacklist) < 2 {
		return fmt.Errorf("%w: %d given", errSelfTradePreventionUIDsShort, len(req.Blacklist))
	}
	return nil
}
