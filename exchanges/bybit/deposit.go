package bybit

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

var (
	errDepositIDRequired     = errors.New("deposit ID required")
	errQuestionnaireRequired = errors.New("questionnaire required")
	errCoinChainPairRequired = errors.New("coin and chain must be passed together")
)

// SetDepositAccount sets the account deposits are transferred to automatically: UNIFIED, FUND, or EARN, which falls
// back to FUND for coins without flexible savings. Only the master UID can set it (Set Deposit Account)
func (e *Exchange) SetDepositAccount(ctx context.Context, accountType string) (*StatusResponse, error) {
	if accountType == "" {
		return nil, errMissingAccountType
	}
	body := map[string]any{"accountType": accountType}
	var resp *StatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/deposit/deposit-to-account", nil, body, &resp, setDepositAccountEPL)
}

// GetDepositRecords returns the on-chain deposits of the API key's UID. Without a window the API returns the last 30
// days, and a window spans less than 30 days (Get Deposit Records (on-chain))
func (e *Exchange) GetDepositRecords(ctx context.Context, arg *DepositRecordsRequest) (*DepositRecordsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.ID != "" {
		params.Set("id", arg.ID)
	}
	if arg.TransactionID != "" {
		params.Set("txID", arg.TransactionID)
	}
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.Upper().String())
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Cursor != "" {
		params.Set("cursor", arg.Cursor)
	}
	var resp *DepositRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-record", params, nil, &resp, getDepositRecordsEPL)
}

// GetSubDepositRecords returns a sub UID's on-chain deposits using the master UID's API key. Without a window the API
// returns the last 30 days, and a window spans less than 30 days (Get Sub Deposit Records (on-chain))
func (e *Exchange) GetSubDepositRecords(ctx context.Context, arg *SubDepositRecordsRequest) (*DepositRecordsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.SubMemberID == "" {
		return nil, errMemberIDRequired
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.ID != "" {
		params.Set("id", arg.ID)
	}
	if arg.TransactionID != "" {
		params.Set("txID", arg.TransactionID)
	}
	params.Set("subMemberId", arg.SubMemberID)
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.Upper().String())
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Cursor != "" {
		params.Set("cursor", arg.Cursor)
	}
	var resp *DepositRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-sub-member-record", params, nil, &resp, getSubDepositRecordsEPL)
}

// GetInternalDepositRecordsOffChain returns deposits made within Bybit, which do not touch a blockchain. Without a
// window the API returns the last 30 days, and a window spans at most 30 days (Get Internal Deposit Records
// (off-chain))
func (e *Exchange) GetInternalDepositRecordsOffChain(ctx context.Context, arg *InternalDepositRecordsRequest) (*InternalDepositRecordsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.TransactionID != "" {
		params.Set("txID", arg.TransactionID)
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.Upper().String())
	}
	if arg.Cursor != "" {
		params.Set("cursor", arg.Cursor)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp *InternalDepositRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-internal-record", params, nil, &resp, getInternalDepositRecordsOffChainEPL)
}

// GetMasterDepositAddress returns the master account's deposit addresses for coin, on every chain or on the chain
// chainType names; chainType takes a chain value from GetCoinInfo (Get Master Deposit Address)
func (e *Exchange) GetMasterDepositAddress(ctx context.Context, coin currency.Code, chainType string) (*DepositAddressResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.Upper().String())
	if chainType != "" {
		params.Set("chainType", chainType)
	}
	var resp *DepositAddressResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-address", params, nil, &resp, getMasterDepositAddressEPL)
}

// GetSubDepositAddress returns a sub-account's deposit address for coin on the chain chainType names, which takes a
// chain value from GetCoinInfo. It needs the master UID's API key, and custodial sub-accounts have no address (Get Sub
// Deposit Address)
func (e *Exchange) GetSubDepositAddress(ctx context.Context, coin currency.Code, chainType, subMemberID string) (*DepositAddressResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if chainType == "" {
		return nil, errMissingChainType
	}
	if subMemberID == "" {
		return nil, errMemberIDRequired
	}
	params := url.Values{}
	params.Set("coin", coin.Upper().String())
	params.Set("chainType", chainType)
	params.Set("subMemberId", subMemberID)
	var resp *DepositAddressResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-sub-member-address", params, nil, &resp, getSubDepositAddressEPL)
}

// SubmitDepositOriginatorInfo submits the originator's Travel Rule information for a deposit whose travel rule status
// is 1, asking for counterparty information, and returns the deposit's new status (Submit Deposit Originator Info)
func (e *Exchange) SubmitDepositOriginatorInfo(ctx context.Context, arg *DepositOriginatorInfoRequest) (*DepositOriginatorInfoResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.DepositID == 0 {
		return nil, errDepositIDRequired
	}
	if arg.Questionnaire == nil {
		return nil, errQuestionnaireRequired
	}
	questionnaire, err := arg.Questionnaire.encode()
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"depositId":     arg.DepositID,
		"questionnaire": questionnaire,
	}
	if arg.SubAccountID != 0 {
		body["subAccountId"] = arg.SubAccountID
	}
	var resp *DepositOriginatorInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/travel-rule/deposit/submit", nil, body, &resp, defaultEPL)
}

// GetAllowedDepositCoinInfo returns the coins and chains that accept deposits, with their minimum deposit and
// confirmation count. Bybit removed its documentation page in March 2025 but the public route still answers; Coin and
// Chain filter together or not at all (Get Allowed Deposit Coin Info)
func (e *Exchange) GetAllowedDepositCoinInfo(ctx context.Context, arg *AllowedDepositCoinInfoRequest) (*AllowedDepositCoinInfoResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Coin.IsEmpty() != (arg.Chain == "") {
		return nil, errCoinChainPairRequired
	}
	params := url.Values{}
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.Upper().String())
		params.Set("chain", arg.Chain)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Cursor != "" {
		params.Set("cursor", arg.Cursor)
	}
	var resp *AllowedDepositCoinInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/asset/deposit/query-allowed-list", params), defaultEPL, &resp)
}
