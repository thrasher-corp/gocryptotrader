package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errInvalidQuantityLimit                = errors.New("invalid quantity limit")
	errMissingValidGreeksType              = errors.New("missing valid greeks type")
	errMissingIsolatedMarginTradingSetting = errors.New("isolated margin trading setting is required")
	errInvalidPositionMode                 = errors.New("invalid position mode")
	errModeSwitchMismatch                  = fmt.Errorf("%w: the switch confirms another setting than it requested", common.ErrInvalidResponse)
	errQuarterValueRequired                = errors.New("quarter is required")
	errYearRequired                        = errors.New("year is required")
	errLendingSideRequired                 = errors.New("lending side is required")
	errTradeFeeGroupIDConflict             = errors.New("trading fee group ID cannot be combined with an instrument ID or family")
	errFeeTypeRequired                     = errors.New("fee type is required")
	errPositionQuantityRequired            = errors.New("position quantity is required")
	errAveragePriceRequired                = errors.New("average price is required")
	errGraphTypeRequired                   = errors.New("graph type is required")
	errFeatureRequired                     = errors.New("feature is required")
	errAccountLevelRequired                = errors.New("account level is required")
	errCurrencyListRequired                = errors.New("currency list is required")
	errSourceAccountRequired               = errors.New("source account is required")
	errDestinationAccountRequired          = errors.New("destination account is required")
	errPositionIDRequired                  = errors.New("position ID is required")
	errAutoEarnActionRequired              = errors.New("auto earn action is required")
	errTradingConfigTypeRequired           = errors.New("trading config type is required")
	errStrategyTypeRequired                = errors.New("strategy type is required")
	errAdjustmentTypeRequired              = errors.New("adjustment type is required")
	errAdjustmentsRequired                 = errors.New("at least one adjustment is required")
	errDuplicateCurrency                   = errors.New("duplicate currency")
	errProgramRequired                     = errors.New("program is required")
	errBusinessIDRequired                  = errors.New("business ID is required")
)

// GetAccountInstruments retrieves the instruments available to the account
func (e *Exchange) GetAccountInstruments(ctx context.Context, arg *AccountInstrumentsRequest) ([]AccountInstrumentResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	instrumentType := strings.ToUpper(arg.InstrumentType)
	switch instrumentType {
	case "":
		return nil, fmt.Errorf("%w: instrument type is required", errInvalidInstrumentType)
	case instTypeOption:
		if arg.InstrumentFamily == "" {
			return nil, errInstrumentFamilyRequired
		}
	case instTypeEvents:
		if arg.SeriesID == "" {
			return nil, errSeriesIDRequired
		}
	}
	params := url.Values{}
	params.Set("instType", instrumentType)
	if arg.SeriesID != "" {
		params.Set("seriesId", arg.SeriesID)
	}
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	var resp []AccountInstrumentResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAccountInstrumentsEPL, http.MethodGet, common.EncodeURLValues("account/instruments", params), nil, &resp, request.AuthenticatedRequest)
}

// GetAccountBalance retrieves the assets with a non-zero balance, their remaining balance and available amount in the
// trading account. A null account row in the reply is refused, as it holds no account to report.
func (e *Exchange) GetAccountBalance(ctx context.Context, arg *AccountBalanceRequest) ([]AccountBalanceResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateEntryCount("currencies", len(arg.Currencies), 20); err != nil {
		return nil, err
	}
	params := url.Values{}
	if len(arg.Currencies) > 0 {
		params.Set("ccy", joinCurrencies(arg.Currencies))
	}
	var rows []*AccountBalanceResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getAccountBalanceEPL, http.MethodGet, common.EncodeURLValues("account/balance", params), nil, &rows, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	resp := make([]AccountBalanceResponse, len(rows))
	for i, row := range rows {
		if row == nil {
			return nil, fmt.Errorf("%w: account balance reply row %d is null", common.ErrInvalidResponse, i+1)
		}
		resp[i] = *row
	}
	return resp, nil
}

// GetPositions retrieves positions: net positions in net mode, and long or short positions in long/short mode
func (e *Exchange) GetPositions(ctx context.Context, arg *PositionsRequest) ([]AccountPositionResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateEntryCount("instrument IDs", len(arg.InstrumentIDs), 10); err != nil {
		return nil, err
	}
	if err := validateEntryCount("position IDs", len(arg.PositionIDs), 20); err != nil {
		return nil, err
	}
	params := url.Values{}
	if arg.InstrumentType != "" {
		params.Set("instType", strings.ToUpper(arg.InstrumentType))
	}
	if len(arg.InstrumentIDs) > 0 {
		params.Set("instId", strings.Join(arg.InstrumentIDs, ","))
	}
	if len(arg.PositionIDs) > 0 {
		params.Set("posId", strings.Join(arg.PositionIDs, ","))
	}
	var resp []AccountPositionResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getPositionsEPL, http.MethodGet, common.EncodeURLValues("account/positions", params), nil, &resp, request.AuthenticatedRequest)
}

// GetPositionsHistory retrieves the updated position data for the last 3 months
func (e *Exchange) GetPositionsHistory(ctx context.Context, arg *PositionsHistoryRequest) ([]PositionHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateTimeWindow(arg.Before, arg.After); err != nil {
		return nil, err
	}
	if arg.Limit > 100 {
		return nil, fmt.Errorf("%w: %d", errLimitValueExceedsMaxOf100, arg.Limit)
	}
	params := url.Values{}
	if arg.InstrumentType != "" {
		params.Set("instType", strings.ToUpper(arg.InstrumentType))
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.MarginMode != "" {
		params.Set("mgnMode", arg.MarginMode)
	}
	if arg.CloseType != "" {
		params.Set("type", arg.CloseType)
	}
	if arg.PositionID != "" {
		params.Set("posId", arg.PositionID)
	}
	if !arg.After.IsZero() {
		params.Set("after", strconv.FormatInt(arg.After.UnixMilli(), 10))
	}
	if !arg.Before.IsZero() {
		params.Set("before", strconv.FormatInt(arg.Before.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []PositionHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getPositionsHistoryEPL, http.MethodGet, common.EncodeURLValues("account/positions-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetAccountAndPositionRisk retrieves account and position information from the same snapshot
func (e *Exchange) GetAccountAndPositionRisk(ctx context.Context, arg *AccountPositionRiskRequest) ([]AccountPositionRiskResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.InstrumentType != "" {
		params.Set("instType", strings.ToUpper(arg.InstrumentType))
	}
	var resp []AccountPositionRiskResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAccountAndPositionRiskEPL, http.MethodGet, common.EncodeURLValues("account/account-position-risk", params), nil, &resp, request.AuthenticatedRequest)
}

// GetBillsDetailLast7Days retrieves the account bills of the last 7 days, the most recent first
func (e *Exchange) GetBillsDetailLast7Days(ctx context.Context, arg *BillsDetailRequest) ([]BillsDetailResponse, error) {
	return e.getBillsDetail(ctx, arg, "account/bills", getBillsDetailsEPL)
}

// GetBillsDetail3Months retrieves the account bills of the last 3 months, the most recent first
func (e *Exchange) GetBillsDetail3Months(ctx context.Context, arg *BillsDetailRequest) ([]BillsDetailResponse, error) {
	return e.getBillsDetail(ctx, arg, "account/bills-archive", getBillsDetailArchiveEPL)
}

func (e *Exchange) getBillsDetail(ctx context.Context, arg *BillsDetailRequest, route string, epl request.EndpointLimit) ([]BillsDetailResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateTimeWindow(arg.BeginTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.Limit > 100 {
		return nil, fmt.Errorf("%w: %d", errLimitValueExceedsMaxOf100, arg.Limit)
	}
	params := url.Values{}
	if arg.InstrumentType != "" {
		params.Set("instType", strings.ToUpper(arg.InstrumentType))
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.Upper().String())
	}
	if arg.MarginMode != "" {
		params.Set("mgnMode", arg.MarginMode)
	}
	if arg.ContractType != "" {
		params.Set("ctType", arg.ContractType)
	}
	if arg.BillType != 0 {
		params.Set("type", strconv.FormatUint(arg.BillType, 10))
	}
	if arg.BillSubType != 0 {
		params.Set("subType", strconv.FormatUint(arg.BillSubType, 10))
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if !arg.BeginTime.IsZero() {
		params.Set("begin", strconv.FormatInt(arg.BeginTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("end", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []BillsDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodGet, common.EncodeURLValues(route, params), nil, &resp, request.AuthenticatedRequest)
}

// ApplyBillDetails applies for the bills of a quarter since 1 February 2021, except for the current quarter
func (e *Exchange) ApplyBillDetails(ctx context.Context, arg *BillsHistoryArchiveRequest) (*ApplyBillDetailsResponse, error) {
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var resp *ApplyBillDetailsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, billHistoryArchiveEPL, http.MethodPost, "account/bills-history-archive", arg, &resp, request.AuthenticatedRequest)
}

// GetBillsHistoryArchive retrieves the download link of the bills of a quarter applied for with ApplyBillDetails
func (e *Exchange) GetBillsHistoryArchive(ctx context.Context, arg *BillsHistoryArchiveRequest) ([]BillsHistoryArchiveResponse, error) {
	if err := arg.validate(); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("year", strconv.FormatUint(arg.Year, 10))
	params.Set("quarter", arg.Quarter)
	if arg.BillTypes != "" {
		params.Set("type", arg.BillTypes)
	}
	var resp []BillsHistoryArchiveResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getBillHistoryArchiveEPL, http.MethodGet, common.EncodeURLValues("account/bills-history-archive", params), nil, &resp, request.AuthenticatedRequest)
}

func (r *BillsHistoryArchiveRequest) validate() error {
	if r == nil {
		return common.ErrNilPointer
	}
	if r.Year == 0 {
		return errYearRequired
	}
	if r.Quarter == "" {
		return errQuarterValueRequired
	}
	return nil
}

// GetBillTypes retrieves the bill types and their sub-types
func (e *Exchange) GetBillTypes(ctx context.Context, arg *BillTypesRequest) ([]BillTypeResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.BillTypes != "" {
		params.Set("type", arg.BillTypes)
	}
	var resp []BillTypeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getBillTypesEPL, http.MethodGet, common.EncodeURLValues("account/subtypes", params), nil, &resp, request.AuthenticatedRequest)
}

// GetAccountConfiguration retrieves the current account configuration
func (e *Exchange) GetAccountConfiguration(ctx context.Context) (*AccountConfigurationResponse, error) {
	var resp *AccountConfigurationResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAccountConfigurationEPL, http.MethodGet, "account/config", nil, &resp, request.AuthenticatedRequest)
}

// SetPositionMode sets the position mode of FUTURES and SWAP: long_short_mode holds long and short positions, net_mode one direction only
func (e *Exchange) SetPositionMode(ctx context.Context, positionMode string) (*PositionModeResponse, error) {
	if positionMode != positionModeLongShort && positionMode != positionModeNet {
		return nil, fmt.Errorf("%w: %q", errInvalidPositionMode, positionMode)
	}
	ctx, err := e.withAccountCredentials(ctx)
	if err != nil {
		return nil, err
	}
	// The key is taken before the request, so a confirmed mode is cached for
	// the credentials that switched it.
	apiKey := e.positionModeKey(ctx)
	if err := e.lockAccountModeSwitch(ctx); err != nil {
		return nil, err
	}
	defer e.unlockAccountModeSwitch()
	var resp *PositionModeResponse
	err = e.SendHTTPRequest(ctx, exchange.RestSpot, setPositionModeEPL, http.MethodPost, "account/set-position-mode", &SetPositionModeRequest{
		PositionMode: positionMode,
	}, &resp, request.AuthenticatedRequest)
	switch {
	case err != nil:
		if switchOutcomeUnknown(err) {
			e.forgetAccountModes(apiKey, false)
		}
		return nil, err
	case resp == nil:
		err = fmt.Errorf("%w setting the account position mode", common.ErrNoResponse)
	case resp.PositionMode != positionModeLongShort && resp.PositionMode != positionModeNet:
		err = fmt.Errorf("%w %q", errInvalidPositionMode, resp.PositionMode)
	case resp.PositionMode != positionMode:
		err = fmt.Errorf("%w: position mode %q for a switch to %q", errModeSwitchMismatch, resp.PositionMode, positionMode)
	}
	if err != nil {
		// A success reply which confirms no mode, or another mode, leaves the
		// switch's outcome unknown
		e.forgetAccountModes(apiKey, false)
		return nil, err
	}
	// The mode switch succeeded, so the cached mode the order placement
	// branches on is refreshed with the mode OKX confirms.
	e.accountPositionModeMu.Lock()
	// Another API key can belong to the same account, so every other key
	// fetches its mode again rather than keep one this switch made stale.
	clear(e.accountPositionModes)
	e.accountPositionModes[apiKey] = resp.PositionMode
	e.accountPositionModeSwitches++
	e.accountPositionModeMu.Unlock()
	return resp, nil
}

// SetLeverageRate sets the leverage of an instrument, or of a currency for cross MARGIN
func (e *Exchange) SetLeverageRate(ctx context.Context, arg *SetLeverageRequest) (*SetLeverageResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if *arg == (SetLeverageRequest{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.InstrumentID == "" && arg.Currency.IsEmpty() {
		return nil, errEitherInstIDOrCcyIsRequired
	}
	if arg.Leverage <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidLeverage, arg.Leverage)
	}
	switch arg.MarginMode {
	case TradeModeCross, TradeModeIsolated:
	default:
		return nil, fmt.Errorf("%w: %q", margin.ErrMarginTypeUnsupported, arg.MarginMode)
	}
	if arg.InstrumentID == "" && arg.MarginMode != TradeModeCross {
		// OKX uses instId when both are sent, so a currency-scoped setting is
		// always cross margin.
		return nil, fmt.Errorf("%w: a currency-scoped leverage requires %q margin, got %q", margin.ErrMarginTypeUnsupported, TradeModeCross, arg.MarginMode)
	}
	arg.PositionSide = strings.ToLower(arg.PositionSide)
	switch arg.PositionSide {
	case "", positionSideLong, positionSideShort, positionSideNet:
		// OKX requires posSide only for isolated futures and perpetual swap
		// leverage in long/short mode, which this layer cannot see.
	default:
		return nil, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, arg.PositionSide)
	}
	var resp *SetLeverageResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, setLeverageEPL, http.MethodPost, "account/set-leverage", arg, &resp, request.AuthenticatedRequest)
}

// GetMaximumBuySellAmountOrOpenAmount retrieves the maximum quantity to buy or sell, which corresponds to an order's size
func (e *Exchange) GetMaximumBuySellAmountOrOpenAmount(ctx context.Context, arg *MaximumOrderQuantityRequest) ([]MaximumOrderQuantityResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if len(arg.InstrumentIDs) == 0 {
		return nil, errMissingInstrumentID
	}
	if err := validateEntryCount("instrument IDs", len(arg.InstrumentIDs), 5); err != nil {
		return nil, err
	}
	if arg.TradeMode == "" {
		return nil, fmt.Errorf("%w: trade mode is required", errInvalidTradeModeValue)
	}
	params := url.Values{}
	params.Set("instId", strings.Join(arg.InstrumentIDs, ","))
	params.Set("tdMode", arg.TradeMode)
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.Upper().String())
	}
	if arg.Price != 0 {
		params.Set("px", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.Leverage != 0 {
		params.Set("leverage", strconv.FormatFloat(arg.Leverage, 'f', -1, 64))
	}
	if !arg.TradeQuoteCurrency.IsEmpty() {
		params.Set("tradeQuoteCcy", arg.TradeQuoteCurrency.Upper().String())
	}
	if arg.Outcome != "" {
		params.Set("outcome", arg.Outcome)
	}
	var resp []MaximumOrderQuantityResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMaximumBuyOrSellAmountEPL, http.MethodGet, common.EncodeURLValues("account/max-size", params), nil, &resp, request.AuthenticatedRequest)
}

// GetMaximumAvailableTradableAmount retrieves the available balance for isolated margin positions and SPOT, or the available equity for cross margin positions
func (e *Exchange) GetMaximumAvailableTradableAmount(ctx context.Context, arg *MaximumAvailableBalanceRequest) ([]MaximumAvailableBalanceResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if len(arg.InstrumentIDs) == 0 {
		return nil, errMissingInstrumentID
	}
	if err := validateEntryCount("instrument IDs", len(arg.InstrumentIDs), 5); err != nil {
		return nil, err
	}
	if arg.TradeMode == "" {
		return nil, fmt.Errorf("%w: trade mode is required", errInvalidTradeModeValue)
	}
	params := url.Values{}
	params.Set("instId", strings.Join(arg.InstrumentIDs, ","))
	params.Set("tdMode", arg.TradeMode)
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.Upper().String())
	}
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if arg.Price != 0 {
		params.Set("px", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if !arg.TradeQuoteCurrency.IsEmpty() {
		params.Set("tradeQuoteCcy", arg.TradeQuoteCurrency.Upper().String())
	}
	var resp []MaximumAvailableBalanceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMaximumAvailableTradableAmountEPL, http.MethodGet, common.EncodeURLValues("account/max-avail-size", params), nil, &resp, request.AuthenticatedRequest)
}

// IncreaseDecreaseMargin increases or decreases the margin of an isolated position; a reduction may change the actual leverage
func (e *Exchange) IncreaseDecreaseMargin(ctx context.Context, arg *IncreaseDecreaseMarginRequest) (*IncreaseDecreaseMarginResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if *arg == (IncreaseDecreaseMarginRequest{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if !slices.Contains([]string{positionSideLong, positionSideShort, positionSideNet}, arg.PositionSide) {
		return nil, fmt.Errorf("%w, position side is required", order.ErrSideIsInvalid)
	}
	if arg.MarginBalanceType != marginBalanceAdd && arg.MarginBalanceType != marginBalanceReduce {
		return nil, fmt.Errorf("%w, missing valid 'type', 'add': add margin 'reduce': reduce margin are allowed", order.ErrTypeIsInvalid)
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var resp *IncreaseDecreaseMarginResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, increaseOrDecreaseMarginEPL, http.MethodPost, "account/position/margin-balance", arg, &resp, request.AuthenticatedRequest)
}

// GetLeverageRate retrieves the leverage of instruments, or of currencies for cross MARGIN
func (e *Exchange) GetLeverageRate(ctx context.Context, arg *LeverageInfoRequest) ([]LeverageResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if len(arg.InstrumentIDs) == 0 && len(arg.Currencies) == 0 {
		return nil, errEitherInstIDOrCcyIsRequired
	}
	if err := validateEntryCount("instrument IDs", len(arg.InstrumentIDs), 20); err != nil {
		return nil, err
	}
	if err := validateEntryCount("currencies", len(arg.Currencies), 20); err != nil {
		return nil, err
	}
	switch arg.MarginMode {
	case TradeModeCross, TradeModeIsolated:
	default:
		return nil, fmt.Errorf("%w: %q", margin.ErrMarginTypeUnsupported, arg.MarginMode)
	}
	params := url.Values{}
	if len(arg.InstrumentIDs) > 0 {
		params.Set("instId", strings.Join(arg.InstrumentIDs, ","))
	}
	if len(arg.Currencies) > 0 {
		params.Set("ccy", joinCurrencies(arg.Currencies))
	}
	params.Set("mgnMode", arg.MarginMode)
	var resp []LeverageResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getLeverageEPL, http.MethodGet, common.EncodeURLValues("account/leverage-info", params), nil, &resp, request.AuthenticatedRequest)
}

// GetLeverageEstimatedInfo retrieves the estimated margin, maximum amounts and liquidation price under a leverage
func (e *Exchange) GetLeverageEstimatedInfo(ctx context.Context, arg *LeverageEstimatedInfoRequest) ([]LeverageEstimatedInfoResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	instrumentType := strings.ToUpper(arg.InstrumentType)
	if instrumentType == "" {
		return nil, fmt.Errorf("%w: instrument type is required", errInvalidInstrumentType)
	}
	switch arg.MarginMode {
	case TradeModeCross, TradeModeIsolated:
	default:
		return nil, fmt.Errorf("%w: %q", margin.ErrMarginTypeUnsupported, arg.MarginMode)
	}
	if arg.Leverage <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidLeverage, arg.Leverage)
	}
	if arg.InstrumentID == "" && (instrumentType == instTypeSwap || instrumentType == instTypeFutures) {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instType", instrumentType)
	params.Set("mgnMode", arg.MarginMode)
	params.Set("lever", strconv.FormatFloat(arg.Leverage, 'f', -1, 64))
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.Upper().String())
	}
	if arg.PositionSide != "" {
		params.Set("posSide", arg.PositionSide)
	}
	var resp []LeverageEstimatedInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getLeverateEstimatedInfoEPL, http.MethodGet, common.EncodeURLValues("account/adjust-leverage-info", params), nil, &resp, request.AuthenticatedRequest)
}

// GetMaximumLoanOfInstrument retrieves the maximum loan of instruments, or the maximum manual borrow of a currency in Spot mode
func (e *Exchange) GetMaximumLoanOfInstrument(ctx context.Context, arg *MaximumLoanRequest) ([]MaximumLoanResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.MarginMode == "" {
		return nil, margin.ErrInvalidMarginType
	}
	if len(arg.InstrumentIDs) == 0 && arg.Currency.IsEmpty() {
		return nil, errEitherInstIDOrCcyIsRequired
	}
	if err := validateEntryCount("instrument IDs", len(arg.InstrumentIDs), 5); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("mgnMode", arg.MarginMode)
	if len(arg.InstrumentIDs) > 0 {
		params.Set("instId", strings.Join(arg.InstrumentIDs, ","))
	}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.Upper().String())
	}
	if !arg.MarginCurrency.IsEmpty() {
		params.Set("mgnCcy", arg.MarginCurrency.Upper().String())
	}
	if !arg.TradeQuoteCurrency.IsEmpty() {
		params.Set("tradeQuoteCcy", arg.TradeQuoteCurrency.Upper().String())
	}
	var resp []MaximumLoanResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getTheMaximumLoanOfInstrumentEPL, http.MethodGet, common.EncodeURLValues("account/max-loan", params), nil, &resp, request.AuthenticatedRequest)
}

// GetTradeFee retrieves the fee rates of an instrument type
func (e *Exchange) GetTradeFee(ctx context.Context, arg *TradeFeeRequest) ([]TradeFeeRateResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentType == "" {
		return nil, fmt.Errorf("%w: instrument type is required", errInvalidInstrumentType)
	}
	if arg.GroupID != "" && (arg.InstrumentID != "" || arg.InstrumentFamily != "") {
		return nil, errTradeFeeGroupIDConflict
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(arg.InstrumentType))
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	if arg.GroupID != "" {
		params.Set("groupId", arg.GroupID)
	}
	var resp []TradeFeeRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getFeeRatesEPL, http.MethodGet, common.EncodeURLValues("account/trade-fee", params), nil, &resp, request.AuthenticatedRequest)
}

// GetInterestAccruedData retrieves the interest accrued over the past year
func (e *Exchange) GetInterestAccruedData(ctx context.Context, arg *InterestAccruedRequest) ([]InterestAccruedResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateTimeWindow(arg.Before, arg.After); err != nil {
		return nil, err
	}
	if arg.Limit > 100 {
		return nil, fmt.Errorf("%w: %d", errLimitValueExceedsMaxOf100, arg.Limit)
	}
	params := url.Values{}
	if arg.LoanType != "" {
		params.Set("type", arg.LoanType)
	}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.Upper().String())
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.MarginMode != "" {
		params.Set("mgnMode", strings.ToLower(arg.MarginMode))
	}
	if !arg.After.IsZero() {
		params.Set("after", strconv.FormatInt(arg.After.UnixMilli(), 10))
	}
	if !arg.Before.IsZero() {
		params.Set("before", strconv.FormatInt(arg.Before.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []InterestAccruedResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getInterestAccruedDataEPL, http.MethodGet, common.EncodeURLValues("account/interest-accrued", params), nil, &resp, request.AuthenticatedRequest)
}

// GetInterestRate retrieves the current hourly market borrowing interest rate of leveraged currencies
func (e *Exchange) GetInterestRate(ctx context.Context, arg *InterestRateRequest) ([]InterestRateResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.Upper().String())
	}
	var resp []InterestRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getInterestRateEPL, http.MethodGet, common.EncodeURLValues("account/interest-rate", params), nil, &resp, request.AuthenticatedRequest)
}

// SetFeeType sets the currency spot trading fees are charged in: 0 charges the currency received, 1 always charges the quote currency
func (e *Exchange) SetFeeType(ctx context.Context, feeType string) (*FeeTypeResponse, error) {
	if feeType == "" {
		return nil, errFeeTypeRequired
	}
	var resp *FeeTypeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, setFeeTypeEPL, http.MethodPost, "account/set-fee-type", &SetFeeTypeRequest{FeeType: feeType}, &resp, request.AuthenticatedRequest)
}

// SetGreeks sets the display type of greeks: PA in coins, or BS for Black-Scholes in dollars
func (e *Exchange) SetGreeks(ctx context.Context, greeksType string) (*GreeksTypeResponse, error) {
	if greeksType == "" {
		return nil, errMissingValidGreeksType
	}
	var resp *GreeksTypeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, setGreeksEPL, http.MethodPost, "account/set-greeks", &SetGreeksRequest{GreeksType: strings.ToUpper(greeksType)}, &resp, request.AuthenticatedRequest)
}

// SetIsolatedMarginTradingSettings sets the isolated margin trading mode of MARGIN or of futures and perpetual contracts
func (e *Exchange) SetIsolatedMarginTradingSettings(ctx context.Context, arg *IsolatedMarginTradingSettingsRequest) (*IsolatedMarginTradingSettingsResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if *arg == (IsolatedMarginTradingSettingsRequest{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.IsolatedMode == "" {
		return nil, errMissingIsolatedMarginTradingSetting
	}
	if arg.InstrumentType == "" {
		return nil, fmt.Errorf("%w: instrument type is required", errInvalidInstrumentType)
	}
	var resp *IsolatedMarginTradingSettingsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, isolatedMarginTradingSettingsEPL, http.MethodPost, "account/set-isolated-mode", arg, &resp, request.AuthenticatedRequest)
}

// GetMaximumWithdrawals retrieves the maximum amounts transferable from the trading account to the funding account
func (e *Exchange) GetMaximumWithdrawals(ctx context.Context, arg *MaximumWithdrawalsRequest) ([]MaximumWithdrawalResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateEntryCount("currencies", len(arg.Currencies), 20); err != nil {
		return nil, err
	}
	params := url.Values{}
	if len(arg.Currencies) > 0 {
		params.Set("ccy", joinCurrencies(arg.Currencies))
	}
	var resp []MaximumWithdrawalResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMaximumWithdrawalsEPL, http.MethodGet, common.EncodeURLValues("account/max-withdrawal", params), nil, &resp, request.AuthenticatedRequest)
}

// GetAccountRiskState retrieves the account risk state; only applicable to Portfolio margin accounts
func (e *Exchange) GetAccountRiskState(ctx context.Context) ([]AccountRiskStateResponse, error) {
	var resp []AccountRiskStateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAccountRiskStateEPL, http.MethodGet, "account/risk-state", nil, &resp, request.AuthenticatedRequest)
}

// GetBorrowInterestAndLimit retrieves borrow interest and limits
func (e *Exchange) GetBorrowInterestAndLimit(ctx context.Context, arg *BorrowInterestAndLimitRequest) ([]BorrowInterestAndLimitResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.LoanType != "" {
		params.Set("type", arg.LoanType)
	}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.Upper().String())
	}
	var resp []BorrowInterestAndLimitResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getBorrowInterestAndLimitEPL, http.MethodGet, common.EncodeURLValues("account/interest-limits", params), nil, &resp, request.AuthenticatedRequest)
}

// ManualBorrowOrRepay borrows or repays a currency; only applicable to Spot mode with borrowing enabled
func (e *Exchange) ManualBorrowOrRepay(ctx context.Context, arg *ManualBorrowRepayRequest) (*ManualBorrowRepayResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Side == "" {
		return nil, errLendingSideRequired
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var resp *ManualBorrowRepayResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, manualBorrowOrRepayEPL, http.MethodPost, "account/spot-manual-borrow-repay", arg, &resp, request.AuthenticatedRequest)
}

// SetAutoRepay turns auto repay on or off; only applicable to Spot mode with borrowing enabled
func (e *Exchange) SetAutoRepay(ctx context.Context, autoRepay bool) (*AutoRepayResponse, error) {
	var resp *AutoRepayResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, setAutoRepayEPL, http.MethodPost, "account/set-auto-repay", &SetAutoRepayRequest{AutoRepay: autoRepay}, &resp, request.AuthenticatedRequest)
}

// GetBorrowRepayHistory retrieves the borrow and repay history in Spot mode
func (e *Exchange) GetBorrowRepayHistory(ctx context.Context, arg *BorrowRepayHistoryRequest) ([]BorrowRepayHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateTimeWindow(arg.Before, arg.After); err != nil {
		return nil, err
	}
	if arg.Limit > 100 {
		return nil, fmt.Errorf("%w: %d", errLimitValueExceedsMaxOf100, arg.Limit)
	}
	params := url.Values{}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.Upper().String())
	}
	if arg.EventType != "" {
		params.Set("type", arg.EventType)
	}
	if !arg.After.IsZero() {
		params.Set("after", strconv.FormatInt(arg.After.UnixMilli(), 10))
	}
	if !arg.Before.IsZero() {
		params.Set("before", strconv.FormatInt(arg.Before.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []BorrowRepayHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getBorrowRepayHistoryEPL, http.MethodGet, common.EncodeURLValues("account/spot-borrow-repay-history", params), nil, &resp, request.AuthenticatedRequest)
}

// NewPositionBuilder calculates portfolio margin information for simulated positions and assets, or the current positions of the account
func (e *Exchange) NewPositionBuilder(ctx context.Context, arg *PositionBuilderRequest) (*PositionBuilderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateEntryCount("simulated positions", len(arg.SimulatedPositions), 200); err != nil {
		return nil, err
	}
	if err := validateEntryCount("simulated assets", len(arg.SimulatedAssets), 200); err != nil {
		return nil, err
	}
	if err := validateSimulations(arg.SimulatedPositions, arg.SimulatedAssets); err != nil {
		return nil, err
	}
	var resp *PositionBuilderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, newPositionBuilderEPL, http.MethodPost, "account/position-builder", arg, &resp, request.AuthenticatedRequest)
}

// GetPositionBuilderTrendGraph calculates the maintenance margin requirement of simulated positions and assets, or the current positions of the account, across price changes
func (e *Exchange) GetPositionBuilderTrendGraph(ctx context.Context, arg *PositionBuilderTrendGraphRequest) (*PositionBuilderTrendGraphResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.GraphType == "" {
		return nil, errGraphTypeRequired
	}
	if err := validateSimulations(arg.SimulatedPositions, arg.SimulatedAssets); err != nil {
		return nil, err
	}
	var resp *PositionBuilderTrendGraphResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, positionBuilderTrendGraphEPL, http.MethodPost, "account/position-builder-graph", arg, &resp, request.AuthenticatedRequest)
}

// SetRiskOffsetAmount sets the spot risk offset amount defined by the user, which is not the spot risk offset amount in use; only applicable to Portfolio margin
func (e *Exchange) SetRiskOffsetAmount(ctx context.Context, arg *SetRiskOffsetAmountRequest) (*RiskOffsetAmountResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.ClientSpotInUseAmount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var resp *RiskOffsetAmountResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, setRiskOffsetAmountEPL, http.MethodPost, "account/set-riskOffset-amt", arg, &resp, request.AuthenticatedRequest)
}

// GetGreeks retrieves the greeks of the assets in the account
func (e *Exchange) GetGreeks(ctx context.Context, arg *GreeksRequest) ([]GreeksResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.Upper().String())
	}
	var resp []GreeksResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getGreeksEPL, http.MethodGet, common.EncodeURLValues("account/greeks", params), nil, &resp, request.AuthenticatedRequest)
}

// GetPMPositionLimitation retrieves the cross position limitation of SWAP, FUTURES and OPTION in Portfolio margin mode
func (e *Exchange) GetPMPositionLimitation(ctx context.Context, arg *PMPositionLimitationRequest) ([]PMLimitationResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentType == "" {
		return nil, fmt.Errorf("%w: instrument type is required", errInvalidInstrumentType)
	}
	if len(arg.InstrumentFamilies) == 0 {
		return nil, errInstrumentFamilyRequired
	}
	if err := validateEntryCount("instrument families", len(arg.InstrumentFamilies), 5); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(arg.InstrumentType))
	params.Set("instFamily", strings.Join(arg.InstrumentFamilies, ","))
	var resp []PMLimitationResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getPMLimitationEPL, http.MethodGet, common.EncodeURLValues("account/position-tiers", params), nil, &resp, request.AuthenticatedRequest)
}

// ActivateOption activates option trading
func (e *Exchange) ActivateOption(ctx context.Context) (*ActivateOptionResponse, error) {
	var resp *ActivateOptionResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, activateOptionEPL, http.MethodPost, "account/activate-option", nil, &resp, request.AuthenticatedRequest)
}

// ActivateFeature activates an account feature; feature 1 activates USDC order book trading, which OKX only requires after order placement fails with error code 54109
func (e *Exchange) ActivateFeature(ctx context.Context, feature string) error {
	if feature == "" {
		return errFeatureRequired
	}
	var resp []json.RawMessage
	return e.SendHTTPRequest(ctx, exchange.RestSpot, activateFeatureEPL, http.MethodPost, "account/activate-feature", &ActivateFeatureRequest{Feature: feature}, &resp, request.AuthenticatedRequest)
}

// SetAutoLoan turns automatic borrowing on or off; only applicable to Multi-currency margin and Portfolio margin
func (e *Exchange) SetAutoLoan(ctx context.Context, autoLoan bool) (*AutoLoanResponse, error) {
	var resp *AutoLoanResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, setAutoLoanEPL, http.MethodPost, "account/set-auto-loan", &SetAutoLoanRequest{AutoLoan: autoLoan}, &resp, request.AuthenticatedRequest)
}

// PresetAccountModeSwitch presets the information an account mode switch requires, such as the leverage of cross margin contract positions when leaving Portfolio margin
func (e *Exchange) PresetAccountModeSwitch(ctx context.Context, arg *PresetAccountModeSwitchRequest) (*PresetAccountModeSwitchResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AccountLevel == "" {
		return nil, errAccountLevelRequired
	}
	var resp *PresetAccountModeSwitchResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, presetAccountModeSwitchEPL, http.MethodPost, "account/account-level-switch-preset", arg, &resp, request.AuthenticatedRequest)
}

// PrecheckAccountModeSwitch retrieves the precheck information for switching to an account mode
func (e *Exchange) PrecheckAccountModeSwitch(ctx context.Context, accountLevel string) (*PrecheckAccountModeSwitchResponse, error) {
	if accountLevel == "" {
		return nil, errAccountLevelRequired
	}
	params := url.Values{}
	params.Set("acctLv", accountLevel)
	var resp *PrecheckAccountModeSwitchResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, precheckAccountModeSwitchEPL, http.MethodGet, common.EncodeURLValues("account/set-account-switch-precheck", params), nil, &resp, request.AuthenticatedRequest)
}

// SetAccountMode sets the account mode: 1 Spot mode, 2 Futures mode, 3 Multi-currency margin or 4 Portfolio margin; each mode must first be set on the website or app
func (e *Exchange) SetAccountMode(ctx context.Context, accountLevel string) (*AccountModeResponse, error) {
	if accountLevel == "" {
		return nil, errAccountLevelRequired
	}
	ctx, err := e.withAccountCredentials(ctx)
	if err != nil {
		return nil, err
	}
	// The key is taken before the request, so a confirmed level is cached for
	// the credentials that switched it.
	apiKey := e.positionModeKey(ctx)
	if err := e.lockAccountModeSwitch(ctx); err != nil {
		return nil, err
	}
	defer e.unlockAccountModeSwitch()
	var resp *AccountModeResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, setAccountLevelEPL, http.MethodPost, "account/set-account-level", &SetAccountModeRequest{AccountLevel: accountLevel}, &resp, request.AuthenticatedRequest); err != nil {
		if switchOutcomeUnknown(err) {
			e.forgetAccountModes(apiKey, true)
		}
		return nil, err
	}
	if resp == nil {
		e.forgetAccountModes(apiKey, true)
		return nil, fmt.Errorf("%w setting the account level", common.ErrNoResponse)
	}
	// A success reply which confirms no level, or another level, leaves the
	// switch's outcome unknown
	level, err := strconv.ParseUint(resp.AccountLevel, 10, 64)
	if err != nil || level < accountLevelSpot || level > accountLevelPortfolioMargin {
		e.forgetAccountModes(apiKey, true)
		return nil, fmt.Errorf("%w %q", errInvalidAccountLevel, resp.AccountLevel)
	}
	if requested, err := strconv.ParseUint(accountLevel, 10, 64); err != nil || requested != level {
		e.forgetAccountModes(apiKey, true)
		return nil, fmt.Errorf("%w: account level %d for a switch to %q", errModeSwitchMismatch, level, accountLevel)
	}
	// Another API key can belong to the same account, and a switch can change
	// the position mode too, Portfolio margin taking net mode only, so every
	// cached mode is fetched again apart from the level OKX confirms.
	e.accountPositionModeMu.Lock()
	clear(e.accountPositionModes)
	clear(e.accountLevels)
	e.accountLevels[apiKey] = level
	e.accountPositionModeSwitches++
	e.accountPositionModeMu.Unlock()
	return resp, nil
}

// lockAccountModeSwitch waits for the account mode switch in progress, if
// any, to complete, returning the context's error if it ends first
func (e *Exchange) lockAccountModeSwitch(ctx context.Context) error {
	e.accountModeSwitchGateOnce.Do(func() { e.accountModeSwitchGate = make(chan struct{}, 1) })
	select {
	case e.accountModeSwitchGate <- struct{}{}:
		return nil
	default:
	}
	select {
	case e.accountModeSwitchGate <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// unlockAccountModeSwitch lets the next account mode switch proceed
func (e *Exchange) unlockAccountModeSwitch() {
	<-e.accountModeSwitchGate
}

// endpointRequestTimeoutCode is the OKX error code of a request whose outcome
// OKX does not know
const endpointRequestTimeoutCode = 50004

// switchOutcomeUnknown reports whether an account setting request which failed
// may still have been applied: an OKX error reply refuses it, apart from OKX's
// endpoint timeout, which leaves the outcome unknown as a failed transport does
func switchOutcomeUnknown(err error) bool {
	var respErr *ResponseError
	return !errors.As(err, &respErr) || respErr.Code == endpointRequestTimeoutCode
}

// forgetAccountModes drops the cached account modes a switch through apiKey
// whose outcome is unknown may have changed: every key's position mode, as
// another key can belong to the same account, and apiKey's account level, or
// every key's after an account level switch. The switch is counted so that a
// fetch it overlaps does not cache what may predate it.
func (e *Exchange) forgetAccountModes(apiKey string, levelSwitch bool) {
	e.accountPositionModeMu.Lock()
	defer e.accountPositionModeMu.Unlock()
	clear(e.accountPositionModes)
	if levelSwitch {
		clear(e.accountLevels)
	} else {
		delete(e.accountLevels, apiKey)
	}
	e.accountPositionModeSwitches++
}

// SetCollateralAssets sets whether assets are used as collateral
func (e *Exchange) SetCollateralAssets(ctx context.Context, arg *SetCollateralAssetsRequest) (*CollateralAssetsResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Type == "custom" && len(arg.Currencies) == 0 {
		return nil, errCurrencyListRequired
	}
	var resp *CollateralAssetsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, setCollateralAssetsEPL, http.MethodPost, "account/set-collateral-assets", arg, &resp, request.AuthenticatedRequest)
}

// GetCollateralAssets retrieves whether assets are used as collateral
func (e *Exchange) GetCollateralAssets(ctx context.Context, arg *CollateralAssetsRequest) ([]CollateralAssetResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateEntryCount("currencies", len(arg.Currencies), 20); err != nil {
		return nil, err
	}
	params := url.Values{}
	if len(arg.Currencies) > 0 {
		params.Set("ccy", joinCurrencies(arg.Currencies))
	}
	if arg.CollateralEnabled != nil {
		params.Set("collateralEnabled", strconv.FormatBool(*arg.CollateralEnabled))
	}
	var resp []CollateralAssetResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getCollateralAssetsEPL, http.MethodGet, common.EncodeURLValues("account/collateral-assets", params), nil, &resp, request.AuthenticatedRequest)
}

// ResetMMPStatus unfreezes trading once MMP has been triggered; only applicable to options in Portfolio margin mode with the MMP privilege
func (e *Exchange) ResetMMPStatus(ctx context.Context, arg *ResetMMPStatusRequest) (*MMPStatusResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	var resp *MMPStatusResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, resetMMPStatusEPL, http.MethodPost, "account/mmp-reset", arg, &resp, request.AuthenticatedRequest)
}

// SetMMP sets the market maker protection configuration; only applicable to options in Portfolio margin mode with the MMP privilege
func (e *Exchange) SetMMP(ctx context.Context, arg *SetMMPRequest) (*SetMMPResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if *arg == (SetMMPRequest{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.InstrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	if arg.QuantityLimit <= 0 {
		return nil, errInvalidQuantityLimit
	}
	var resp *SetMMPResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, setMMPEPL, http.MethodPost, "account/mmp-config", arg, &resp, request.AuthenticatedRequest)
}

// GetMMPConfig retrieves the market maker protection configuration; only applicable to options in Portfolio margin mode with the MMP privilege
func (e *Exchange) GetMMPConfig(ctx context.Context, arg *MMPConfigRequest) ([]MMPConfigResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	var resp []MMPConfigResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMMPConfigEPL, http.MethodGet, common.EncodeURLValues("account/mmp-config", params), nil, &resp, request.AuthenticatedRequest)
}

// MovePositions moves positions between accounts under the same master account; only applicable to VIP6 or higher through the master account's API key
func (e *Exchange) MovePositions(ctx context.Context, arg *MovePositionsRequest) (*MovePositionsResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.FromAccount == "" {
		return nil, errSourceAccountRequired
	}
	if arg.ToAccount == "" {
		return nil, errDestinationAccountRequired
	}
	if len(arg.Legs) == 0 {
		return nil, errMissingLegs
	}
	for i := range arg.Legs {
		if arg.Legs[i].From.PositionID == "" {
			return nil, fmt.Errorf("%w: leg %d", errPositionIDRequired, i)
		}
		if arg.Legs[i].From.Size <= 0 {
			return nil, fmt.Errorf("%w: leg %d", order.ErrAmountIsInvalid, i)
		}
		if arg.Legs[i].From.Side == "" {
			return nil, fmt.Errorf("%w: leg %d", order.ErrSideIsInvalid, i)
		}
	}
	if arg.ClientID == "" {
		return nil, errClientIDRequired
	}
	var resp *MovePositionsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, movePositionsEPL, http.MethodPost, "account/move-positions", arg, &resp, request.AuthenticatedRequest)
}

// GetMovePositionsHistory retrieves positions moved in the last 3 days; only applicable to VIP6 or higher through the master account's API key
func (e *Exchange) GetMovePositionsHistory(ctx context.Context, arg *MovePositionsHistoryRequest) ([]MovePositionsHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateTimeWindow(arg.BeginTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.Limit > 100 {
		return nil, fmt.Errorf("%w: %d", errLimitValueExceedsMaxOf100, arg.Limit)
	}
	params := url.Values{}
	if arg.BlockTradeID != "" {
		params.Set("blockTdId", arg.BlockTradeID)
	}
	if arg.ClientID != "" {
		params.Set("clientId", arg.ClientID)
	}
	if !arg.BeginTime.IsZero() {
		params.Set("beginTs", strconv.FormatInt(arg.BeginTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTs", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.State != "" {
		params.Set("state", arg.State)
	}
	var resp []MovePositionsHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMovePositionsHistoryEPL, http.MethodGet, common.EncodeURLValues("account/move-positions-history", params), nil, &resp, request.AuthenticatedRequest)
}

// SetAutoEarn turns auto earn on or off for a currency
func (e *Exchange) SetAutoEarn(ctx context.Context, arg *SetAutoEarnRequest) (*AutoEarnResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Action == "" {
		return nil, errAutoEarnActionRequired
	}
	var resp *AutoEarnResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, setAutoEarnEPL, http.MethodPost, "account/set-auto-earn", arg, &resp, request.AuthenticatedRequest)
}

// SetSettleCurrency sets the settlement currency of USD-margined contracts
func (e *Exchange) SetSettleCurrency(ctx context.Context, settlementCurrency currency.Code) (*SettleCurrencyResponse, error) {
	if settlementCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *SettleCurrencyResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, setSettleCurrencyEPL, http.MethodPost, "account/set-settle-currency", &SetSettleCurrencyRequest{SettlementCurrency: settlementCurrency.Upper()}, &resp, request.AuthenticatedRequest)
}

// SetTradingConfig sets a trading configuration, such as the strategy type
func (e *Exchange) SetTradingConfig(ctx context.Context, arg *SetTradingConfigRequest) (*TradingConfigResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Type == "" {
		return nil, errTradingConfigTypeRequired
	}
	var resp *TradingConfigResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, setTradingConfigEPL, http.MethodPost, "account/set-trading-config", arg, &resp, request.AuthenticatedRequest)
}

// PrecheckSetDeltaNeutral retrieves what blocks switching to a strategy type: 0 general strategy, 1 delta neutral strategy
func (e *Exchange) PrecheckSetDeltaNeutral(ctx context.Context, strategyType string) (*PrecheckSetDeltaNeutralResponse, error) {
	if strategyType == "" {
		return nil, errStrategyTypeRequired
	}
	params := url.Values{}
	params.Set("stgyType", strategyType)
	var resp *PrecheckSetDeltaNeutralResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, precheckSetDeltaNeutralEPL, http.MethodGet, common.EncodeURLValues("account/precheck-set-delta-neutral", params), nil, &resp, request.AuthenticatedRequest)
}

// AdjustDemoAccountBalance increases or reduces the balances of a demo trading account; only applicable to demo trading, and the whole request is rejected if any currency fails
func (e *Exchange) AdjustDemoAccountBalance(ctx context.Context, arg *AdjustDemoAccountBalanceRequest) (*AdjustDemoAccountBalanceResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Type == "" {
		return nil, errAdjustmentTypeRequired
	}
	if len(arg.Adjustments) == 0 {
		return nil, errAdjustmentsRequired
	}
	seen := make(map[string]bool, len(arg.Adjustments))
	for i := range arg.Adjustments {
		if arg.Adjustments[i].Currency.IsEmpty() {
			return nil, fmt.Errorf("%w: adjustment %d", currency.ErrCurrencyCodeEmpty, i)
		}
		if arg.Adjustments[i].Amount <= 0 {
			return nil, fmt.Errorf("%w: adjustment %d", limits.ErrAmountBelowMin, i)
		}
		symbol := arg.Adjustments[i].Currency.Upper().String()
		if seen[symbol] {
			return nil, fmt.Errorf("%w: %s", errDuplicateCurrency, symbol)
		}
		seen[symbol] = true
	}
	epl := reduceDemoAccountBalanceEPL
	if arg.Type == "increase" {
		epl = increaseDemoAccountBalanceEPL
	}
	var resp *AdjustDemoAccountBalanceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodPost, "account/demo-adjust-balance", arg, &resp, request.AuthenticatedRequest)
}

// GetGLPTodayPerformance retrieves the current day and month to date Global Liquidity Program performance of every enrolled program
func (e *Exchange) GetGLPTodayPerformance(ctx context.Context) (*GLPTodayPerformanceResponse, error) {
	var resp *GLPTodayPerformanceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getGLPTodayPerformanceEPL, http.MethodGet, "users/glp/todayperformance", nil, &resp, request.AuthenticatedRequest)
}

// GetGLPHistoricalPerformance retrieves the daily Global Liquidity Program performance of a program, the newest first
func (e *Exchange) GetGLPHistoricalPerformance(ctx context.Context, arg *GLPHistoricalPerformanceRequest) ([]GLPHistoricalPerformanceResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Program == "" {
		return nil, errProgramRequired
	}
	if err := validateTimeWindow(arg.BeginTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.Limit > 100 {
		return nil, fmt.Errorf("%w: %d", errLimitValueExceedsMaxOf100, arg.Limit)
	}
	params := url.Values{}
	params.Set("program", arg.Program)
	if !arg.BeginTime.IsZero() {
		params.Set("begin", strconv.FormatInt(arg.BeginTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("end", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []GLPHistoricalPerformanceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getGLPHistoricalPerformanceEPL, http.MethodGet, common.EncodeURLValues("users/glp/historicalperformance", params), nil, &resp, request.AuthenticatedRequest)
}

// GetGLPV2Overview retrieves the active Global Liquidity Program enrollments, tiers, tier requirements and week to date performance
func (e *Exchange) GetGLPV2Overview(ctx context.Context) (*GLPOverviewResponse, error) {
	var resp *GLPOverviewResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getGLPV2OverviewEPL, http.MethodGet, "users/glp/v2/overview", nil, &resp, request.AuthenticatedRequest)
}

// GetGLPV2AssessmentDetails retrieves the daily and week to date Global Liquidity Program performance of a product and trading side in the current assessment period
func (e *Exchange) GetGLPV2AssessmentDetails(ctx context.Context, businessID string) (*GLPAssessmentDetailsResponse, error) {
	if businessID == "" {
		return nil, errBusinessIDRequired
	}
	params := url.Values{}
	params.Set("bizId", businessID)
	var resp *GLPAssessmentDetailsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getGLPV2AssessmentDetailsEPL, http.MethodGet, common.EncodeURLValues("users/glp/v2/details", params), nil, &resp, request.AuthenticatedRequest)
}

// GetGLPV2PerformanceHistory retrieves the weekly Global Liquidity Program assessment results of a product and trading side
func (e *Exchange) GetGLPV2PerformanceHistory(ctx context.Context, businessID string) (*GLPPerformanceHistoryResponse, error) {
	if businessID == "" {
		return nil, errBusinessIDRequired
	}
	params := url.Values{}
	params.Set("bizId", businessID)
	var resp *GLPPerformanceHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getGLPV2PerformanceHistoryEPL, http.MethodGet, common.EncodeURLValues("users/glp/v2/history/performance", params), nil, &resp, request.AuthenticatedRequest)
}

// validateEntryCount rejects a list longer than the maximum the endpoint documents per request
func validateEntryCount(name string, count, maximum int) error {
	if count > maximum {
		return fmt.Errorf("%w: %d %s, at most %d per request", errExceedLimit, count, name, maximum)
	}
	return nil
}

// validateTimeWindow rejects a window whose start is after its end; either bound may be unset
func validateTimeWindow(start, end time.Time) error {
	if !start.IsZero() && !end.IsZero() && start.After(end) {
		return fmt.Errorf("%w: %s is after %s", common.ErrStartAfterEnd, start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
	}
	return nil
}

// validateSimulations checks the fields OKX requires of each simulated position and asset
func validateSimulations(positions []SimulatedPosition, assets []SimulatedAsset) error {
	for i := range positions {
		if positions[i].InstrumentID == "" {
			return fmt.Errorf("%w: simulated position %d", errMissingInstrumentID, i)
		}
		if positions[i].Position == 0 {
			return fmt.Errorf("%w: simulated position %d", errPositionQuantityRequired, i)
		}
		if positions[i].AveragePrice <= 0 {
			return fmt.Errorf("%w: simulated position %d", errAveragePriceRequired, i)
		}
	}
	for i := range assets {
		if assets[i].Currency.IsEmpty() {
			return fmt.Errorf("%w: simulated asset %d", currency.ErrCurrencyCodeEmpty, i)
		}
		if assets[i].Amount == 0 {
			return fmt.Errorf("%w: simulated asset %d", limits.ErrAmountBelowMin, i)
		}
	}
	return nil
}

// joinCurrencies returns currencies as the comma separated upper case list OKX takes
func joinCurrencies(codes []currency.Code) string {
	symbols := make([]string, len(codes))
	for i := range codes {
		symbols[i] = codes[i].Upper().String()
	}
	return strings.Join(symbols, ",")
}
