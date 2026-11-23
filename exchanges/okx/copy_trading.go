package okx

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errSubPositionIDRequired                    = errors.New("sub position ID is required")
	errSubPositionCloseTypeRequired             = errors.New("sub position close type is required")
	errUniqueCodeRequired                       = errors.New("unique code is required")
	errLastDaysRequired                         = errors.New("last days is required")
	errCopyMarginModeRequired                   = errors.New("copy margin mode is required")
	errCopyInstrumentIDTypeRequired             = errors.New("copy instrument ID type is required")
	errCopyAmountRequired                       = errors.New("copy amount is required")
	errCopyRatioRequired                        = errors.New("copy ratio is required")
	errTakeProfitOrStopLossTriggerPriceRequired = errors.New("take profit or stop loss trigger price is required")
	errInvalidProfitSharingRatio                = errors.New("invalid profit sharing ratio")
)

const (
	copyInstrumentIDTypeCustom = "custom"
	copyModeFixedAmount        = "fixed_amount"
	copyModeRatioCopy          = "ratio_copy"
)

// GetExistingLeadPositions retrieves the lead trader's lead positions that are not closed, newest first
func (e *Exchange) GetExistingLeadPositions(ctx context.Context, req *LeadPositionsRequest) ([]LeadPositionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	var resp []LeadPositionResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getExistingLeadingPositionsEPL, http.MethodGet, common.EncodeURLValues("copytrading/current-subpositions", req.values()), nil, &resp, request.AuthenticatedRequest)
}

// GetLeadPositionHistory retrieves the lead trader's lead positions closed in the last 3 months, newest first
func (e *Exchange) GetLeadPositionHistory(ctx context.Context, req *LeadPositionsRequest) ([]LeadPositionHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	var resp []LeadPositionHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getLeadingPositionHistoryEPL, http.MethodGet, common.EncodeURLValues("copytrading/subpositions-history", req.values()), nil, &resp, request.AuthenticatedRequest)
}

func (r *LeadPositionsRequest) values() url.Values {
	params := url.Values{}
	if r.InstrumentType != "" {
		params.Set("instType", r.InstrumentType)
	}
	if r.InstrumentID != "" {
		params.Set("instId", r.InstrumentID)
	}
	if r.After != "" {
		params.Set("after", r.After)
	}
	if r.Before != "" {
		params.Set("before", r.Before)
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	return params
}

// PlaceLeadStopOrder sets the take-profit and stop-loss of an open lead position
func (e *Exchange) PlaceLeadStopOrder(ctx context.Context, req *PlaceLeadStopOrderRequest) (*LeadPositionActionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.SubPositionID == "" {
		return nil, errSubPositionIDRequired
	}
	if req.TakeProfitTriggerPrice == 0 && req.StopLossTriggerPrice == 0 {
		return nil, errTakeProfitOrStopLossTriggerPriceRequired
	}
	var resp *LeadPositionActionResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, placeLeadingStopOrderEPL, http.MethodPost, "copytrading/algo-order", req, &resp, request.AuthenticatedRequest)
}

// CloseLeadPosition closes a lead position; only one lead position can be closed per request
func (e *Exchange) CloseLeadPosition(ctx context.Context, req *CloseLeadPositionRequest) (*LeadPositionActionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.SubPositionID == "" {
		return nil, errSubPositionIDRequired
	}
	var resp *LeadPositionActionResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, closeLeadingPositionEPL, http.MethodPost, "copytrading/close-subposition", req, &resp, request.AuthenticatedRequest)
}

// GetLeadingInstruments retrieves the instruments the platform supports for lead trading and whether the lead
// trader leads each one
func (e *Exchange) GetLeadingInstruments(ctx context.Context, instrumentType string) ([]LeadingInstrumentResponse, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []LeadingInstrumentResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getLeadingInstrumentsEPL, http.MethodGet, common.EncodeURLValues("copytrading/instruments", params), nil, &resp, request.AuthenticatedRequest)
}

// AmendLeadingInstruments sets the instruments the lead trader leads; an instrument that stops being led must have
// no position or pending order
func (e *Exchange) AmendLeadingInstruments(ctx context.Context, req *AmendLeadingInstrumentsRequest) ([]LeadingInstrumentResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	var resp []LeadingInstrumentResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, amendLeadingInstrumentsEPL, http.MethodPost, "copytrading/set-instruments", req, &resp, request.AuthenticatedRequest)
}

// GetProfitSharingDetails retrieves the profit shared with the lead trader in the last 3 months
func (e *Exchange) GetProfitSharingDetails(ctx context.Context, req *ProfitSharingDetailsRequest) ([]ProfitSharingDetailResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.InstrumentType != "" {
		params.Set("instType", req.InstrumentType)
	}
	if req.After != "" {
		params.Set("after", req.After)
	}
	if req.Before != "" {
		params.Set("before", req.Before)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []ProfitSharingDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getProfitSharingLimitEPL, http.MethodGet, common.EncodeURLValues("copytrading/profit-sharing-details", params), nil, &resp, request.AuthenticatedRequest)
}

// GetTotalProfitSharing retrieves the total profit shared with the lead trader since joining the platform
func (e *Exchange) GetTotalProfitSharing(ctx context.Context, instrumentType string) ([]TotalProfitSharingResponse, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []TotalProfitSharingResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getTotalProfitSharingEPL, http.MethodGet, common.EncodeURLValues("copytrading/total-profit-sharing", params), nil, &resp, request.AuthenticatedRequest)
}

// GetUnrealisedProfitSharingDetails retrieves the profit expected to be shared with the lead trader at the next
// settlement, which updates as copy positions close
func (e *Exchange) GetUnrealisedProfitSharingDetails(ctx context.Context, instrumentType string) ([]UnrealisedProfitSharingDetailResponse, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []UnrealisedProfitSharingDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUnrealisedProfitSharingDetailsEPL, http.MethodGet, common.EncodeURLValues("copytrading/unrealized-profit-sharing-details", params), nil, &resp, request.AuthenticatedRequest)
}

// GetTotalUnrealisedProfitSharing retrieves the total profit expected to be shared with the lead trader at the next
// settlement
func (e *Exchange) GetTotalUnrealisedProfitSharing(ctx context.Context, instrumentType string) (*TotalUnrealisedProfitSharingResponse, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp *TotalUnrealisedProfitSharingResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getTotalUnrealisedProfitSharingEPL, http.MethodGet, common.EncodeURLValues("copytrading/total-unrealized-profit-sharing", params), nil, &resp, request.AuthenticatedRequest)
}

// AmendProfitSharingRatio sets the lead trader's profit sharing ratio
func (e *Exchange) AmendProfitSharingRatio(ctx context.Context, req *AmendProfitSharingRatioRequest) (*CopyTradingResultResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ProfitSharingRatio < 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidProfitSharingRatio, req.ProfitSharingRatio)
	}
	var resp *CopyTradingResultResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, amendProfitSharingRatioEPL, http.MethodPost, "copytrading/amend-profit-sharing-ratio", req, &resp, request.AuthenticatedRequest)
}

// GetCopyTradingAccountConfiguration retrieves the account's copy and lead trading configuration
func (e *Exchange) GetCopyTradingAccountConfiguration(ctx context.Context) (*CopyTradingAccountConfigurationResponse, error) {
	var resp *CopyTradingAccountConfigurationResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getCopyTradingAccountConfigurationEPL, http.MethodGet, "copytrading/config", nil, &resp, request.AuthenticatedRequest)
}

// SetFirstCopySettings sets the first copy settings for a lead trader, which is also needed after stopping copying
func (e *Exchange) SetFirstCopySettings(ctx context.Context, req *CopySettingsRequest) (*CopyTradingResultResponse, error) {
	if err := validateCopySettings(req); err != nil {
		return nil, err
	}
	var resp *CopyTradingResultResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, setFirstCopySettingsEPL, http.MethodPost, "copytrading/first-copy-settings", req, &resp, request.AuthenticatedRequest)
}

// AmendCopySettings amends the copy settings for a lead trader
func (e *Exchange) AmendCopySettings(ctx context.Context, req *CopySettingsRequest) (*CopyTradingResultResponse, error) {
	if err := validateCopySettings(req); err != nil {
		return nil, err
	}
	var resp *CopyTradingResultResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, amendFirstCopySettingsEPL, http.MethodPost, "copytrading/amend-copy-settings", req, &resp, request.AuthenticatedRequest)
}

func validateCopySettings(req *CopySettingsRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.UniqueCode == "" {
		return errUniqueCodeRequired
	}
	if req.CopyMarginMode == "" {
		return errCopyMarginModeRequired
	}
	if req.CopyInstrumentIDType == "" {
		return errCopyInstrumentIDTypeRequired
	}
	if req.CopyInstrumentIDType == copyInstrumentIDTypeCustom && req.InstrumentID == "" {
		return fmt.Errorf("%w: custom copy instruments need instrument IDs", errMissingInstrumentID)
	}
	if req.CopyTotalAmount <= 0 {
		return fmt.Errorf("%w: copy total amount %v", limits.ErrAmountBelowMin, req.CopyTotalAmount)
	}
	switch req.CopyMode {
	case "", copyModeFixedAmount:
		if req.CopyAmount <= 0 {
			return errCopyAmountRequired
		}
	case copyModeRatioCopy:
		if req.CopyRatio <= 0 {
			return errCopyRatioRequired
		}
	}
	if req.SubPositionCloseType == "" {
		return errSubPositionCloseTypeRequired
	}
	return nil
}

// StopCopying stops copying a lead trader
func (e *Exchange) StopCopying(ctx context.Context, req *StopCopyingRequest) (*CopyTradingResultResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.UniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	if req.SubPositionCloseType == "" {
		return nil, errSubPositionCloseTypeRequired
	}
	var resp *CopyTradingResultResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, stopCopyingEPL, http.MethodPost, "copytrading/stop-copy-trading", req, &resp, request.AuthenticatedRequest)
}

// GetCopySettings retrieves the account's copy settings for a lead trader
func (e *Exchange) GetCopySettings(ctx context.Context, req *LeadTraderRequest) (*CopySettingsResponse, error) {
	params, err := req.values()
	if err != nil {
		return nil, err
	}
	var resp *CopySettingsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getCopySettingsEPL, http.MethodGet, common.EncodeURLValues("copytrading/copy-settings", params), nil, &resp, request.AuthenticatedRequest)
}

func (r *LeadTraderRequest) values() (url.Values, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.UniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", r.UniqueCode)
	if r.InstrumentType != "" {
		params.Set("instType", r.InstrumentType)
	}
	return params, nil
}

// GetMyLeadTraders retrieves the lead traders the account copies
func (e *Exchange) GetMyLeadTraders(ctx context.Context, instrumentType string) ([]MyLeadTraderResponse, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []MyLeadTraderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMyLeadTradersEPL, http.MethodGet, common.EncodeURLValues("copytrading/current-lead-traders", params), nil, &resp, request.AuthenticatedRequest)
}

// GetCopyTradingConfiguration retrieves the limits that apply to copy settings
func (e *Exchange) GetCopyTradingConfiguration(ctx context.Context, instrumentType string) (*CopyTradingConfigurationResponse, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp *CopyTradingConfigurationResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getCopyTradingConfigurationEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-config", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetLeadTraderRanks retrieves a page of the lead trader ranking
func (e *Exchange) GetLeadTraderRanks(ctx context.Context, req *LeadTraderRanksRequest) (*LeadTraderRanksResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.InstrumentType != "" {
		params.Set("instType", req.InstrumentType)
	}
	if req.SortType != "" {
		params.Set("sortType", req.SortType)
	}
	if req.HasVacancy {
		params.Set("state", "1")
	}
	if req.MinLeadDays != 0 {
		params.Set("minLeadDays", strconv.FormatUint(req.MinLeadDays, 10))
	}
	if req.MinAssets > 0 {
		params.Set("minAssets", strconv.FormatFloat(req.MinAssets, 'f', -1, 64))
	}
	if req.MaxAssets > 0 {
		params.Set("maxAssets", strconv.FormatFloat(req.MaxAssets, 'f', -1, 64))
	}
	if req.MinAssetsUnderManagement > 0 {
		params.Set("minAum", strconv.FormatFloat(req.MinAssetsUnderManagement, 'f', -1, 64))
	}
	if req.MaxAssetsUnderManagement > 0 {
		params.Set("maxAum", strconv.FormatFloat(req.MaxAssetsUnderManagement, 'f', -1, 64))
	}
	if req.DataVersion != "" {
		params.Set("dataVer", req.DataVersion)
	}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *LeadTraderRanksResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderRanksEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-lead-traders", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetLeadTraderWeeklyProfitAndLoss retrieves a lead trader's weekly profit and loss, newest first
func (e *Exchange) GetLeadTraderWeeklyProfitAndLoss(ctx context.Context, req *LeadTraderRequest) ([]LeadTraderProfitAndLossResponse, error) {
	params, err := req.values()
	if err != nil {
		return nil, err
	}
	var resp []LeadTraderProfitAndLossResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderWeeklyProfitAndLossEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-weekly-pnl", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetLeadTraderDailyProfitAndLoss retrieves a lead trader's profit and loss accumulated to each day, newest first
func (e *Exchange) GetLeadTraderDailyProfitAndLoss(ctx context.Context, req *LeadTraderPerformanceRequest) ([]LeadTraderProfitAndLossResponse, error) {
	params, err := req.values()
	if err != nil {
		return nil, err
	}
	var resp []LeadTraderProfitAndLossResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderDailyProfitAndLossEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-pnl", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetLeadTraderStats retrieves key data of a lead trader's performance
func (e *Exchange) GetLeadTraderStats(ctx context.Context, req *LeadTraderPerformanceRequest) (*LeadTraderStatsResponse, error) {
	params, err := req.values()
	if err != nil {
		return nil, err
	}
	var resp *LeadTraderStatsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderStatsEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-stats", params), nil, &resp, request.UnauthenticatedRequest)
}

func (r *LeadTraderPerformanceRequest) values() (url.Values, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.UniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	if r.LastDays == "" {
		return nil, errLastDaysRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", r.UniqueCode)
	params.Set("lastDays", r.LastDays)
	if r.InstrumentType != "" {
		params.Set("instType", r.InstrumentType)
	}
	return params, nil
}

// GetLeadTraderCurrencyPreferences retrieves the currencies a lead trader trades most, largest share first
func (e *Exchange) GetLeadTraderCurrencyPreferences(ctx context.Context, req *LeadTraderRequest) ([]LeadTraderCurrencyPreferenceResponse, error) {
	params, err := req.values()
	if err != nil {
		return nil, err
	}
	var resp []LeadTraderCurrencyPreferenceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderCurrencyPreferencesEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-preference-currency", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetLeadTraderCurrentLeadPositions retrieves a lead trader's open lead positions
func (e *Exchange) GetLeadTraderCurrentLeadPositions(ctx context.Context, req *LeadTraderPositionsRequest) ([]LeadPositionResponse, error) {
	params, err := req.values()
	if err != nil {
		return nil, err
	}
	var resp []LeadPositionResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getTraderCurrentLeadPositionsEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-current-subpositions", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetLeadTraderLeadPositionHistory retrieves a lead trader's lead positions closed in the last 3 months, newest first
func (e *Exchange) GetLeadTraderLeadPositionHistory(ctx context.Context, req *LeadTraderPositionsRequest) ([]LeadPositionHistoryResponse, error) {
	params, err := req.values()
	if err != nil {
		return nil, err
	}
	var resp []LeadPositionHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderLeadPositionHistoryEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-subpositions-history", params), nil, &resp, request.UnauthenticatedRequest)
}

func (r *LeadTraderPositionsRequest) values() (url.Values, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.UniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", r.UniqueCode)
	if r.InstrumentType != "" {
		params.Set("instType", r.InstrumentType)
	}
	if r.After != "" {
		params.Set("after", r.After)
	}
	if r.Before != "" {
		params.Set("before", r.Before)
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	return params, nil
}

// GetCopyTraders retrieves a lead trader's copy traders, highest profit and loss first
func (e *Exchange) GetCopyTraders(ctx context.Context, req *CopyTradersRequest) (*CopyTradersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.UniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", req.UniqueCode)
	if req.InstrumentType != "" {
		params.Set("instType", req.InstrumentType)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *CopyTradersResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getCopyTradersEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-copy-traders", params), nil, &resp, request.UnauthenticatedRequest)
}
