package kraken

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// GetFuturesAssignmentPrograms calls List assignment programs, returning the account's active assignment preferences
func (e *Exchange) GetFuturesAssignmentPrograms(ctx context.Context) (*FuturesAssignmentProgramsResponse, error) {
	var resp *FuturesAssignmentProgramsResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/assignmentprogram/current", nil, nil, &resp)
}

// AddFuturesAssignmentPreference calls Add assignment preference, volunteering the account to take over the positions
// of liquidated accounts at a price the engine sets. Assignments arrive as fills whenever the preference is active, and
// cannot be declined. The account must first accept the Assignment Program terms on Kraken Pro
func (e *Exchange) AddFuturesAssignmentPreference(ctx context.Context, req *FuturesAddAssignmentPreferenceRequest) (*FuturesAssignmentPreferenceResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ContractType == "" {
		return nil, errFuturesContractTypeEmpty
	}
	if req.TimeFrame == "" {
		return nil, errFuturesAssignmentTimeFrameEmpty
	}
	if !req.Contract.IsEmpty() && !req.Pair.IsEmpty() {
		return nil, errFuturesAssignmentScopeConflict
	}
	if req.Pair.Base.IsEmpty() != req.Pair.Quote.IsEmpty() {
		return nil, fmt.Errorf("%w: a currency pair needs a base and a quote", currency.ErrCurrencyCodeEmpty)
	}
	if req.MaxSize < 0 {
		return nil, fmt.Errorf("%w: max size %v", errFuturesNegativeLimit, req.MaxSize)
	}
	if req.MaxPosition != nil && *req.MaxPosition < 0 {
		return nil, fmt.Errorf("%w: max position %v", errFuturesNegativeLimit, *req.MaxPosition)
	}
	params := url.Values{}
	params.Set("contractType", req.ContractType)
	if !req.Contract.IsEmpty() {
		contract, err := e.FormatSymbol(req.Contract, asset.Futures)
		if err != nil {
			return nil, err
		}
		params.Set("contract", contract)
	}
	if !req.Pair.IsEmpty() {
		params.Set("baseCurrency", req.Pair.Base.Upper().String())
		params.Set("quoteCurrency", req.Pair.Quote.Upper().String())
	}
	if req.MaxSize != 0 {
		params.Set("maxSize", strconv.FormatFloat(req.MaxSize, 'f', -1, 64))
	}
	if req.MaxPosition != nil {
		params.Set("maxPosition", strconv.FormatFloat(*req.MaxPosition, 'f', -1, 64))
	}
	params.Set("acceptLong", strconv.FormatBool(req.AcceptLong))
	params.Set("acceptShort", strconv.FormatBool(req.AcceptShort))
	params.Set("timeFrame", req.TimeFrame)
	params.Set("enabled", strconv.FormatBool(req.Enabled))
	if req.CashAccountOpenPositionMaxNotional != nil {
		params.Set("cashAccountOpenPositionMaxNotional", strconv.FormatFloat(*req.CashAccountOpenPositionMaxNotional, 'f', -1, 64))
	}
	if req.MinimumProfitabilityLongBasisPoints != nil {
		params.Set("minimumProfitabilityPerAssignmentLongBps", strconv.FormatUint(*req.MinimumProfitabilityLongBasisPoints, 10))
	}
	if req.MinimumProfitabilityShortBasisPoints != nil {
		params.Set("minimumProfitabilityPerAssignmentShortBps", strconv.FormatUint(*req.MinimumProfitabilityShortBasisPoints, 10))
	}
	var resp *FuturesAssignmentPreferenceResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v3/assignmentprogram/add", params, nil, &resp)
}

// DeleteFuturesAssignmentPreference calls Deletes assignment preference, withdrawing the preference with the ID
// GetFuturesAssignmentPrograms lists
func (e *Exchange) DeleteFuturesAssignmentPreference(ctx context.Context, id uint64) (*FuturesAssignmentPreferenceResponse, error) {
	if id == 0 {
		return nil, errFuturesAssignmentIDEmpty
	}
	params := url.Values{}
	params.Set("id", strconv.FormatUint(id, 10))
	var resp *FuturesAssignmentPreferenceResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v3/assignmentprogram/delete", params, nil, &resp)
}

// GetFuturesAssignmentPreferencesHistory calls List assignment preferences history, returning every change to the
// account's assignment preferences, deletions included
func (e *Exchange) GetFuturesAssignmentPreferencesHistory(ctx context.Context) (*FuturesAssignmentPreferencesHistoryResponse, error) {
	var resp *FuturesAssignmentPreferencesHistoryResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/assignmentprogram/history", nil, nil, &resp)
}
