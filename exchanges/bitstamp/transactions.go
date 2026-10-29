package bitstamp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetUserTransactions returns the account's transactions
// Since and Until may only be up to 30 days ago, while older transactions can be paged through with SinceID
func (e *Exchange) GetUserTransactions(ctx context.Context, req *UserTransactionsRequest) ([]UserTransactionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := validatePaging(req.Limit, req.Offset); err != nil {
		return nil, err
	}
	if !req.Since.IsZero() && !req.Until.IsZero() && req.Since.After(req.Until) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if req.Offset != 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Sort != "" {
		params.Set("sort", req.Sort)
	}
	if !req.Since.IsZero() {
		params.Set("since_timestamp", strconv.FormatInt(req.Since.Unix(), 10))
	}
	if !req.Until.IsZero() {
		params.Set("until_timestamp", strconv.FormatInt(req.Until.Unix(), 10))
	}
	if req.SinceID != 0 {
		params.Set("since_id", strconv.FormatUint(req.SinceID, 10))
	}
	path := "/v2/user_transactions/"
	if !req.Pair.IsEmpty() {
		path += formatMarketSymbol(req.Pair) + "/"
	}
	var resp []UserTransactionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, params, &resp)
}

// GetCryptoTransactions returns the main account's cryptocurrency deposits and withdrawals
// Since and Until may only be up to 30 days ago
func (e *Exchange) GetCryptoTransactions(ctx context.Context, req *CryptoTransactionsRequest) (*CryptoTransactionsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := validatePaging(req.Limit, req.Offset); err != nil {
		return nil, err
	}
	if !req.Since.IsZero() && !req.Until.IsZero() && req.Since.After(req.Until) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Offset != 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	setTrue(params, "include_ious", req.IncludeIOUs)
	if !req.Since.IsZero() {
		params.Set("since_timestamp", strconv.FormatInt(req.Since.Unix(), 10))
	}
	if !req.Until.IsZero() {
		params.Set("until_timestamp", strconv.FormatInt(req.Until.Unix(), 10))
	}
	var resp *CryptoTransactionsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/crypto-transactions/", nil, params, &resp)
}

// GetCryptoDeposits returns the account's cryptocurrency deposits with their review status
func (e *Exchange) GetCryptoDeposits(ctx context.Context, req *CryptoDepositsRequest) ([]CryptoDepositResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := validatePaging(req.Limit, req.Offset); err != nil {
		return nil, err
	}
	if !req.Since.IsZero() && !req.Until.IsZero() && req.Since.After(req.Until) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if req.Offset != 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if !req.Since.IsZero() {
		params.Set("since_timestamp", strconv.FormatInt(req.Since.Unix(), 10))
	}
	if !req.Until.IsZero() {
		params.Set("until_timestamp", strconv.FormatInt(req.Until.Unix(), 10))
	}
	if req.Status != "" {
		params.Set("status", req.Status)
	}
	var resp []CryptoDepositResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/crypto-transactions/deposits/", params, nil, &resp)
}

// UpdateCryptoDepositOriginator submits Travel Rule originator details for a pending cryptocurrency deposit
func (e *Exchange) UpdateCryptoDepositOriginator(ctx context.Context, depositID uint64, req *DepositOriginatorRequest) error {
	if depositID == 0 {
		return errDepositIDRequired
	}
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.OriginatorThirdParty != nil && *req.OriginatorThirdParty && req.OriginatorID == "" && req.OriginatorInfo == nil {
		return errOriginatorRequired
	}
	if req.OriginatorInfo != nil {
		if err := validateContact(req.OriginatorInfo.RetailInfo, req.OriginatorInfo.CorporateInfo); err != nil {
			return fmt.Errorf("originator: %w", err)
		}
	}
	// The beneficiary details describe the account holder receiving the deposit, so they must identify them
	if req.BeneficiaryInfo != nil {
		if err := validateIdentifiedCustomer(req.BeneficiaryInfo); err != nil {
			return fmt.Errorf("beneficiary: %w", err)
		}
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/crypto-transactions/deposits/"+strconv.FormatUint(depositID, 10)+"/", nil, req, nil)
}

// RejectCryptoDeposit rejects a pending cryptocurrency deposit
// Rejected funds are not automatically returned to the originator address
func (e *Exchange) RejectCryptoDeposit(ctx context.Context, depositID uint64) error {
	if depositID == 0 {
		return errDepositIDRequired
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/crypto-transactions/deposits/"+strconv.FormatUint(depositID, 10)+"/reject/", nil, nil, nil)
}
