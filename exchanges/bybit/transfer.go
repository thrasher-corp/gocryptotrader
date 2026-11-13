package bybit

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"uuid"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// CreateInternalTransfer transfers a coin between two account types of the same UID; each account type accepts its
// own coins, which GetTransferableCoin lists (Create Internal Transfer)
func (e *Exchange) CreateInternalTransfer(ctx context.Context, arg *InternalTransferRequest) (*TransferResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.TransferID == uuid.Nil() {
		return nil, errMissingTransferID
	}
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.FromAccountType == "" {
		return nil, fmt.Errorf("%w: from account type", errMissingAccountType)
	}
	if arg.ToAccountType == "" {
		return nil, fmt.Errorf("%w: to account type", errMissingAccountType)
	}
	body := make(map[string]any)
	body["transferId"] = arg.TransferID.String()
	body["coin"] = arg.Coin.Upper().String()
	body["amount"] = strconv.FormatFloat(arg.Amount, 'f', -1, 64)
	body["fromAccountType"] = arg.FromAccountType
	body["toAccountType"] = arg.ToAccountType
	var resp *TransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/transfer/inter-transfer", nil, body, &resp, createInternalTransferEPL)
}

// GetInternalTransferRecords returns transfers between account types of the same UID. Without a window the API returns
// the last 7 days, and a window spans at most 7 days (Get Internal Transfer Records)
func (e *Exchange) GetInternalTransferRecords(ctx context.Context, arg *InternalTransferRecordsRequest) (*InternalTransferRecordsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.TransferID != "" {
		params.Set("transferId", arg.TransferID)
	}
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.Upper().String())
	}
	if arg.Status != "" {
		params.Set("status", arg.Status)
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
	var resp *InternalTransferRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-inter-transfer-list", params, nil, &resp, getInternalTransferRecordsEPL)
}

// CreateUniversalTransfer transfers a coin between the master UID and a sub UID or between two sub UIDs. A sub UID's
// API key needs the SubMemberTransferList permission and can only transfer to the master UID (Create Universal
// Transfer)
func (e *Exchange) CreateUniversalTransfer(ctx context.Context, arg *UniversalTransferRequest) (*TransferResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.TransferID == uuid.Nil() {
		return nil, errMissingTransferID
	}
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.FromMemberID == 0 {
		return nil, fmt.Errorf("%w: from member ID", errMemberIDRequired)
	}
	if arg.ToMemberID == 0 {
		return nil, fmt.Errorf("%w: to member ID", errMemberIDRequired)
	}
	if arg.FromAccountType == "" {
		return nil, fmt.Errorf("%w: from account type", errMissingAccountType)
	}
	if arg.ToAccountType == "" {
		return nil, fmt.Errorf("%w: to account type", errMissingAccountType)
	}
	body := map[string]any{
		"transferId":      arg.TransferID.String(),
		"coin":            arg.Coin.Upper().String(),
		"amount":          strconv.FormatFloat(arg.Amount, 'f', -1, 64),
		"fromMemberId":    arg.FromMemberID,
		"toMemberId":      arg.ToMemberID,
		"fromAccountType": arg.FromAccountType,
		"toAccountType":   arg.ToAccountType,
	}
	var resp *TransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/transfer/universal-transfer", nil, body, &resp, createUniversalTransferEPL)
}

// GetUniversalTransferRecords returns transfers between UIDs. The master UID's API key needs the SubMemberTransfer
// permission and a sub UID's the SubMemberTransferList permission. Without a window the API returns the last 7 days,
// and a window spans at most 7 days (Get Universal Transfer Records)
func (e *Exchange) GetUniversalTransferRecords(ctx context.Context, arg *UniversalTransferRecordsRequest) (*UniversalTransferRecordsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.TransferID != "" {
		params.Set("transferId", arg.TransferID)
	}
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.Upper().String())
	}
	if arg.Status != "" {
		params.Set("status", arg.Status)
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.FromMemberID != "" {
		params.Set("fromMemberId", arg.FromMemberID)
	}
	if arg.ToMemberID != "" {
		params.Set("toMemberId", arg.ToMemberID)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Cursor != "" {
		params.Set("cursor", arg.Cursor)
	}
	var resp *UniversalTransferRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-universal-transfer-list", params, nil, &resp, getUniversalTransferRecordsEPL)
}

// GetTransferableCoin returns the coins that can be transferred from one account type to another (Get Transferable
// Coin)
func (e *Exchange) GetTransferableCoin(ctx context.Context, fromAccountType, toAccountType string) (*TransferableCoinResponse, error) {
	if fromAccountType == "" {
		return nil, fmt.Errorf("%w: from account type", errMissingAccountType)
	}
	if toAccountType == "" {
		return nil, fmt.Errorf("%w: to account type", errMissingAccountType)
	}
	params := url.Values{}
	params.Set("fromAccountType", fromAccountType)
	params.Set("toAccountType", toAccountType)
	var resp *TransferableCoinResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-transfer-coin-list", params, nil, &resp, getTransferableCoinEPL)
}
