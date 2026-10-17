package kraken

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

// CalculateFundingFees calls Calculate Funding Fees, quoting the fee a funding method charges on an amount. A
// withdrawal quote includes a fee token, which pins the quoted fee rate for any withdrawal made within 5 minutes
func (e *Exchange) CalculateFundingFees(ctx context.Context, req *FundingFeesRequest) (*FundingFeesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.MethodID == "" {
		return nil, errFundingMethodIDEmpty
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: amount must be positive", limits.ErrAmountBelowMin)
	}
	params := fundingParams(req.AccountID)
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	// Kraken documents no default for fee_included, so false is sent rather than left to the server
	params.Set("fee_included", strconv.FormatBool(req.FeeIncluded))
	if req.WithdrawalFeeToken != "" {
		params.Set("withdrawal_fee_token", req.WithdrawalFeeToken)
	}
	if req.RebaseMultiplier != "" {
		params.Set("rebase_multiplier", req.RebaseMultiplier)
	}
	var resp *FundingFeesResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodGet, "/funding/v1/fees/"+url.PathEscape(req.MethodID), params, nil, &resp)
}

// ClaimFundingDepositAddress calls Claim Funding Deposit Address, reserving a deposit address for a funding method, or
// returning the address it shares with another method on its network. A method limiting address generation holds up
// to its limit of unused addresses, so ListFundingClaimedAddresses should be checked first
func (e *Exchange) ClaimFundingDepositAddress(ctx context.Context, req *FundingDepositAddressRequest) (*FundingDepositAddressResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.MethodID == "" {
		return nil, errFundingMethodIDEmpty
	}
	var resp *FundingDepositAddressResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodPut, "/funding/v1/deposit/address", fundingParams(req.AccountID), map[string]string{"method_id": req.MethodID}, &resp)
}

// CreateFundingAddress calls Create Funding Address, saving a crypto withdrawal address. Addresses saved through the
// API are verified at once, without an email confirmation
func (e *Exchange) CreateFundingAddress(ctx context.Context, req *FundingAddressRequest) (*FundingAddressResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := req.Scope.validate(true, true); err != nil {
		return nil, err
	}
	if req.Address == "" {
		return nil, errFundingAddressEmpty
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, errFundingAddressNameEmpty
	}
	crypto := map[string]string{"address": req.Address}
	if req.Tag != "" {
		crypto["tag"] = req.Tag
	}
	if req.Memo != "" {
		crypto["memo"] = req.Memo
	}
	body := map[string]any{
		"scope":           req.Scope,
		"address_details": map[string]any{"crypto": crypto},
		"name":            req.Name,
	}
	if req.Description != "" {
		body["description"] = req.Description
	}
	var resp *FundingAddressResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodPost, "/funding/v1/addresses", fundingParams(req.AccountID), body, &resp)
}

// CreateFundingWithdrawal calls Create Funding Withdrawal, withdrawing to a saved address. A withdrawal needing an
// approval waits for it, and is cancelled if it is refused
func (e *Exchange) CreateFundingWithdrawal(ctx context.Context, req *FundingWithdrawalRequest) (*FundingWithdrawalResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := req.Scope.validate(true, false); err != nil {
		return nil, err
	}
	if req.AddressID == "" {
		return nil, errFundingAddressIDEmpty
	}
	if err := req.Asset.validate(true); err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: amount must be positive", limits.ErrAmountBelowMin)
	}
	if req.MaximumFee < 0 {
		return nil, fmt.Errorf("%w: maximum fee cannot be negative", limits.ErrAmountBelowMin)
	}
	if req.FeeToken != "" && req.MaximumFee != 0 {
		return nil, errFundingFeeConflict
	}
	amount := map[string]any{"asset_amount": fundingAmountBody(req.Asset, req.Amount)}
	// Kraken documents no default for fee_included, so the fee options are always sent, choosing the current fee when no
	// token pins a quoted one
	fee := map[string]any{"fee_included": req.FeeIncluded}
	switch {
	case req.FeeToken != "":
		quotedFee := map[string]any{"token": req.FeeToken}
		if req.RebaseMultiplier != "" {
			quotedFee["rebase_multiplier"] = req.RebaseMultiplier
		}
		fee["quoted_fee"] = quotedFee
	case req.MaximumFee != 0:
		maximumFee := map[string]any{"asset_amount": fundingAmountBody(req.Asset, req.MaximumFee)}
		if req.RebaseMultiplier != "" {
			maximumFee["rebase_multiplier"] = req.RebaseMultiplier
		}
		fee["current_fee"] = map[string]any{"max_fee": maximumFee}
	default:
		fee["current_fee"] = map[string]any{}
	}
	if req.RebaseMultiplier != "" {
		amount["rebase_multiplier"] = req.RebaseMultiplier
	}
	body := map[string]any{
		"scope":      req.Scope,
		"address_id": req.AddressID,
		"amount":     amount,
		"fee":        fee,
	}
	if req.ExpectedAddress != "" {
		body["expected_address"] = req.ExpectedAddress
	}
	var resp *FundingWithdrawalResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodPost, "/funding/v1/withdrawals", fundingParams(req.AccountID), body, &resp)
}

// DeleteFundingAddress calls Delete Funding Address, deleting a saved withdrawal address
func (e *Exchange) DeleteFundingAddress(ctx context.Context, req *FundingAddressDeletionRequest) (*FundingAddressDeletionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AddressID == "" {
		return nil, errFundingAddressIDEmpty
	}
	var resp *FundingAddressDeletionResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodDelete, "/funding/v1/addresses/"+url.PathEscape(req.AddressID), fundingParams(req.AccountID), nil, &resp)
}

// ListFundingAddresses calls List Funding Addresses, returning a page of saved withdrawal addresses. Filtered by a
// funding method, the page holds every address a withdrawal by that method may use. A nil request returns the first
// page of every address
func (e *Exchange) ListFundingAddresses(ctx context.Context, req *FundingAddressesRequest) (*FundingAddressesResponse, error) {
	params := url.Values{}
	if req != nil {
		if err := req.Scope.validate(false, true); err != nil {
			return nil, err
		}
		var err error
		if params, err = fundingPageParams(req.Cursor, req.Limit, 500, !req.Scope.isEmpty(), req.AccountID); err != nil {
			return nil, err
		}
		req.Scope.setQuery(params)
	}
	var resp *FundingAddressesResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodGet, "/funding/v1/addresses", params, nil, &resp)
}

// ListFundingAssets calls List Funding Assets, returning the assets that can be deposited or withdrawn
func (e *Exchange) ListFundingAssets(ctx context.Context, req *FundingAssetsRequest) (*FundingAssetsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Direction != "deposit" && req.Direction != "withdraw" {
		return nil, fmt.Errorf("%w: %q is not deposit or withdraw", errInvalidFundingDirection, req.Direction)
	}
	params := fundingParams(req.AccountID)
	if req.AssetClass != "" {
		params.Set("asset_class", req.AssetClass)
	}
	var resp *FundingAssetsResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodGet, "/funding/v1/assets/"+req.Direction, params, nil, &resp)
}

// ListFundingClaimedAddresses calls List Funding Claimed Addresses (v2), returning a page of claimed deposit
// addresses. A nil request returns the first page of every claimed address
func (e *Exchange) ListFundingClaimedAddresses(ctx context.Context, req *FundingClaimedAddressesRequest) (*FundingClaimedAddressesResponse, error) {
	params := url.Values{}
	if req != nil {
		if err := req.Scope.validate(false, false); err != nil {
			return nil, err
		}
		if err := req.Asset.validate(false); err != nil {
			return nil, err
		}
		var err error
		if params, err = fundingPageParams(req.Cursor, req.Limit, 500, !req.Scope.isEmpty() || !req.Asset.isEmpty(), req.AccountID); err != nil {
			return nil, err
		}
		req.Scope.setQuery(params)
		req.Asset.setQuery(params, "asset")
	}
	var resp *FundingClaimedAddressesResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodGet, "/funding/v2/deposit/addresses", params, nil, &resp)
}

// ListFundingDepositLimits calls List Funding Deposit Limits, returning an asset's deposit limits for each funding
// method
func (e *Exchange) ListFundingDepositLimits(ctx context.Context, req *FundingLimitsRequest) (*FundingDepositLimitsResponse, error) {
	path, params, err := fundingLimitsRequest("/funding/v1/limits/deposit/", req)
	if err != nil {
		return nil, err
	}
	var resp *FundingDepositLimitsResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodGet, path, params, nil, &resp)
}

// ListFundingDeposits calls List Funding Deposits, returning a page of deposits, newest first. A nil request returns
// the first page of every deposit
func (e *Exchange) ListFundingDeposits(ctx context.Context, req *FundingDepositsRequest) (*FundingDepositsResponse, error) {
	params := url.Values{}
	if req != nil {
		if err := req.Scope.validate(false, true); err != nil {
			return nil, err
		}
		if err := req.Asset.validate(false); err != nil {
			return nil, err
		}
		if len(req.Statuses) != 0 && (req.StatusFrom != "" || req.StatusTo != "") {
			return nil, errFundingStatusFilterConflict
		}
		if !req.StartTime.IsZero() && !req.EndTime.IsZero() && req.StartTime.After(req.EndTime) {
			return nil, common.ErrStartAfterEnd
		}
		filtered := !req.Scope.isEmpty() || !req.Asset.isEmpty() || len(req.Statuses) != 0 || req.StatusFrom != "" || req.StatusTo != "" ||
			!req.StartTime.IsZero() || !req.EndTime.IsZero() || req.RebaseMultiplier != ""
		var err error
		if params, err = fundingPageParams(req.Cursor, req.Limit, 500, filtered, req.AccountID); err != nil {
			return nil, err
		}
		req.Scope.setQuery(params)
		req.Asset.setQuery(params, "asset")
		for i := range req.Statuses {
			params.Set("status[list]["+strconv.Itoa(i)+"]", req.Statuses[i])
		}
		if req.StatusFrom != "" {
			params.Set("status[range][start]", req.StatusFrom)
		}
		if req.StatusTo != "" {
			params.Set("status[range][end]", req.StatusTo)
		}
		setFundingTimes(params, req.StartTime, req.EndTime)
		if req.RebaseMultiplier != "" {
			params.Set("rebase_multiplier", req.RebaseMultiplier)
		}
	}
	var resp *FundingDepositsResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodGet, "/funding/v1/deposits", params, nil, &resp)
}

// ListFundingMethods calls List Funding Methods, returning a page of the methods available to deposit or withdraw
func (e *Exchange) ListFundingMethods(ctx context.Context, req *FundingMethodsRequest) (*FundingMethodsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Direction != "deposit" && req.Direction != "withdraw" {
		return nil, fmt.Errorf("%w: %q is not deposit or withdraw", errInvalidFundingDirection, req.Direction)
	}
	if err := req.Asset.validate(false); err != nil {
		return nil, err
	}
	params, err := fundingPageParams(req.Cursor, req.Limit, 10000, !req.Asset.isEmpty() || req.RebaseMultiplier != "", req.AccountID)
	if err != nil {
		return nil, err
	}
	req.Asset.setQuery(params, "asset")
	if req.RebaseMultiplier != "" {
		params.Set("rebase_multiplier", req.RebaseMultiplier)
	}
	var resp *FundingMethodsResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodGet, "/funding/v1/methods/"+req.Direction, params, nil, &resp)
}

// ListFundingNetworks calls List Funding Networks, returning the networks available for funding and the groups of
// networks sharing an address format. A nil request lists the default wallet account's networks
func (e *Exchange) ListFundingNetworks(ctx context.Context, req *FundingNetworksRequest) (*FundingNetworksResponse, error) {
	params := url.Values{}
	if req != nil {
		params = fundingParams(req.AccountID)
	}
	var resp *FundingNetworksResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodGet, "/funding/v1/networks", params, nil, &resp)
}

// ListFundingWithdrawalLimits calls List Funding Withdrawal Limits, returning the balance available to withdraw and an
// asset's withdrawal limits for each funding method
func (e *Exchange) ListFundingWithdrawalLimits(ctx context.Context, req *FundingLimitsRequest) (*FundingWithdrawalLimitsResponse, error) {
	path, params, err := fundingLimitsRequest("/funding/v1/limits/withdrawal/", req)
	if err != nil {
		return nil, err
	}
	var resp *FundingWithdrawalLimitsResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodGet, path, params, nil, &resp)
}

// ListFundingWithdrawals calls List Funding Withdrawals, returning a page of withdrawals, newest first. A nil request
// returns the first page of every withdrawal
func (e *Exchange) ListFundingWithdrawals(ctx context.Context, req *FundingWithdrawalsRequest) (*FundingWithdrawalsResponse, error) {
	params := url.Values{}
	if req != nil {
		if err := req.Scope.validate(false, true); err != nil {
			return nil, err
		}
		if err := req.Asset.validate(false); err != nil {
			return nil, err
		}
		if !req.StartTime.IsZero() && !req.EndTime.IsZero() && req.StartTime.After(req.EndTime) {
			return nil, common.ErrStartAfterEnd
		}
		filtered := !req.Scope.isEmpty() || !req.Asset.isEmpty() || req.Status != "" || !req.StartTime.IsZero() || !req.EndTime.IsZero() ||
			req.RebaseMultiplier != ""
		var err error
		if params, err = fundingPageParams(req.Cursor, req.Limit, 500, filtered, req.AccountID); err != nil {
			return nil, err
		}
		req.Scope.setQuery(params)
		req.Asset.setQuery(params, "asset")
		if req.Status != "" {
			params.Set("status", req.Status)
		}
		setFundingTimes(params, req.StartTime, req.EndTime)
		if req.RebaseMultiplier != "" {
			params.Set("rebase_multiplier", req.RebaseMultiplier)
		}
	}
	var resp *FundingWithdrawalsResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodGet, "/funding/v1/withdrawals", params, nil, &resp)
}

// UpdateFundingAddress calls Update Funding Address, renaming or redescribing a saved withdrawal address
func (e *Exchange) UpdateFundingAddress(ctx context.Context, req *FundingAddressUpdateRequest) (*FundingAddressUpdateResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AddressID == "" {
		return nil, errFundingAddressIDEmpty
	}
	if req.Name == "" && req.Description == "" {
		return nil, errFundingAddressUpdateEmpty
	}
	body := make(map[string]string, 2)
	if req.Name != "" {
		if strings.TrimSpace(req.Name) == "" {
			return nil, errFundingAddressNameEmpty
		}
		body["name"] = req.Name
	}
	if req.Description != "" {
		body["description"] = req.Description
	}
	var resp *FundingAddressUpdateResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodPut, "/funding/v1/addresses/"+url.PathEscape(req.AddressID), fundingParams(req.AccountID), body, &resp)
}

// fundingParams returns the query parameters every Funding (Beta) endpoint shares
func fundingParams(accountID string) url.Values {
	params := url.Values{}
	if accountID != "" {
		params.Set("account_id", accountID)
	}
	return params
}

// fundingPageParams returns the query parameters a paginated Funding (Beta) list shares. A cursor carries the filters
// of the request that returned it, and Kraken asks for no other filter alongside it
func fundingPageParams(cursor string, limit, maxLimit uint64, filtered bool, accountID string) (url.Values, error) {
	if limit > maxLimit {
		return nil, fmt.Errorf("%w: %d exceeds %d", errInvalidFundingLimit, limit, maxLimit)
	}
	if cursor != "" && filtered {
		return nil, errFundingCursorWithFilters
	}
	params := fundingParams(accountID)
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	return params, nil
}

// setFundingTimes adds the creation time filters of a Funding (Beta) list
func setFundingTimes(params url.Values, start, end time.Time) {
	if !start.IsZero() {
		params.Set("start_time", start.UTC().Format(time.RFC3339Nano))
	}
	if !end.IsZero() {
		params.Set("end_time", end.UTC().Format(time.RFC3339Nano))
	}
}

// fundingLimitsRequest returns the path and query parameters of List Funding Deposit Limits or List Funding
// Withdrawal Limits, whose asset is a path parameter
func fundingLimitsRequest(prefix string, req *FundingLimitsRequest) (string, url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return "", nil, err
	}
	if req.Asset.Class != "currency" && req.Asset.Class != "tokenized_asset" {
		return "", nil, fmt.Errorf("%w: %q is not currency or tokenized_asset", errInvalidFundingAssetClass, req.Asset.Class)
	}
	if req.Asset.Name.IsEmpty() {
		return "", nil, currency.ErrCurrencyCodeEmpty
	}
	if req.PreferredAsset.Class != "" && req.PreferredAsset.Name.IsEmpty() {
		return "", nil, fmt.Errorf("%w: preferred asset name is required with its class", currency.ErrCurrencyCodeEmpty)
	}
	params := fundingParams(req.AccountID)
	req.PreferredAsset.setQuery(params, "preferred_asset")
	return prefix + req.Asset.Class + "/" + url.PathEscape(req.Asset.Name.Upper().String()), params, nil
}

// fundingAmountBody returns an amount of an asset as Funding (Beta) request bodies take it
func fundingAmountBody(asset FundingAsset, amount float64) map[string]any {
	return map[string]any{
		"asset":  map[string]string{"class": asset.Class, "name": asset.Name.Upper().String()},
		"amount": strconv.FormatFloat(amount, 'f', -1, 64),
	}
}

// validate rejects an asset without a class, or, when the name is required, without a name. An asset filter takes a
// class alone, but not a name alone
func (a *FundingAsset) validate(nameRequired bool) error {
	if a.Class == "" && (nameRequired || !a.Name.IsEmpty()) {
		return errFundingAssetClassEmpty
	}
	if nameRequired && a.Name.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	return nil
}

func (a *FundingAsset) isEmpty() bool {
	return a.Class == "" && a.Name.IsEmpty()
}

// setQuery adds the asset to params as Kraken nests query objects, such as asset[class]=currency&asset[name]=USDC
func (a *FundingAsset) setQuery(params url.Values, key string) {
	if a.Class != "" {
		params.Set(key+"[class]", a.Class)
	}
	if !a.Name.IsEmpty() {
		params.Set(key+"[name]", a.Name.Upper().String())
	}
}

// validate rejects a scope setting more than one ID, an empty scope when one is required, and a network group where
// the endpoint takes none
func (s *FundingScope) validate(required, networkGroups bool) error {
	var set int
	for _, id := range []string{s.MethodID, s.NetworkID, s.NetworkGroupID} {
		if id != "" {
			set++
		}
	}
	switch {
	case set > 1:
		return errFundingScopeAmbiguous
	case set == 0 && required:
		return errFundingScopeEmpty
	case s.NetworkGroupID != "" && !networkGroups:
		return errFundingNetworkGroupScope
	}
	return nil
}

func (s *FundingScope) isEmpty() bool {
	return s.MethodID == "" && s.NetworkID == "" && s.NetworkGroupID == ""
}

// setQuery adds the scope to params as Kraken nests query objects, such as scope[method_id]
func (s *FundingScope) setQuery(params url.Values) {
	switch {
	case s.MethodID != "":
		params.Set("scope[method_id]", s.MethodID)
	case s.NetworkID != "":
		params.Set("scope[network_id]", s.NetworkID)
	case s.NetworkGroupID != "":
		params.Set("scope[network_group_id]", s.NetworkGroupID)
	}
}
