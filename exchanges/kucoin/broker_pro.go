package kucoin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// ApplyForFastWithdrawal submits a Fast API withdrawal to an external address or internal account, which needs the
// Third-party withdrawals API permission (Apply for Fast Withdrawal). The first request returns the 2FA factors the
// user must verify; repeating it with Validation set completes the withdrawal and returns its ID
func (e *Exchange) ApplyForFastWithdrawal(ctx context.Context, arg *ApplyForFastWithdrawalRequest) (*ApplyForFastWithdrawalResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Address == "" {
		return nil, errAddressRequired
	}
	body := map[string]any{
		"currency": arg.Currency.Upper().String(),
		"amount":   arg.Amount, // documented as a JSON number, unlike most decimals
		"address":  arg.Address,
		"memo":     arg.Memo,
		"remark":   arg.Remark,
	}
	if arg.Chain != "" {
		body["chain"] = arg.Chain
	}
	if arg.WithdrawType != "" {
		body["withdrawType"] = arg.WithdrawType
	}
	if arg.FeeDeductType != "" {
		body["feeDeductType"] = arg.FeeDeductType
	}
	if arg.IsInner {
		body["isInner"] = true
	}
	if arg.Wallet != "" {
		body["wallet"] = arg.Wallet
	}
	if arg.TIN != "" {
		body["tin"] = arg.TIN
	}
	if arg.Country != "" {
		body["country"] = arg.Country
	}
	if arg.FirstName != "" {
		body["firstName"] = arg.FirstName
	}
	if arg.LastName != "" {
		body["lastName"] = arg.LastName
	}
	if arg.IdentityNumber != "" {
		body["identityNo"] = arg.IdentityNumber
	}
	if arg.IdentityType != "" {
		body["identityType"] = arg.IdentityType
	}
	if arg.CompanyName != "" {
		body["companyName"] = arg.CompanyName
	}
	if arg.ReceiverType != "" {
		body["receiverType"] = arg.ReceiverType
	}
	if arg.Questionnaire != "" {
		body["questionnaire"] = arg.Questionnaire
	}
	if arg.ReasonOfTransfer != "" {
		body["reasonOfTransfer"] = arg.ReasonOfTransfer
	}
	if arg.Validation != nil {
		factors := make([]map[string]any, len(arg.Validation.Factors))
		for i := range arg.Validation.Factors {
			factors[i] = map[string]any{
				"factorType":  arg.Validation.Factors[i].FactorType,
				"factorValue": arg.Validation.Factors[i].FactorValue,
			}
		}
		body["validation"] = map[string]any{
			"factors":       factors,
			"transactionId": arg.Validation.TransactionID,
		}
	}
	var resp *ApplyForFastWithdrawalResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, applyForFastWithdrawalEPL, http.MethodPost, "/v2/broker/withdrawal", body, &resp)
}

// GetBrokerProRebate returns a link, valid for one day, to a CSV file of the Broker Pro rebate orders of a trade type
// between two days at most six months apart (Get Broker Rebate)
func (e *Exchange) GetBrokerProRebate(ctx context.Context, arg *GetBrokerProRebateRequest) (*BrokerRebateDownloadResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := brokerPeriodParams(arg.Begin, arg.End, arg.TradeType)
	if err != nil {
		return nil, err
	}
	var resp *BrokerRebateDownloadResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getBrokerProRebateEPL, http.MethodGet, common.EncodeURLValues("/v2/broker/api/rebate/download", params), nil, &resp)
}

// GetBrokerProCommission returns the Broker Pro's commission per calculation period (Get Commission)
func (e *Exchange) GetBrokerProCommission(ctx context.Context, arg *GetBrokerProCommissionRequest) (*BrokerProCommissionResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartAt.IsZero() && !arg.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartAt, arg.EndAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.SiteType != "" {
		params.Set("siteType", arg.SiteType)
	}
	if arg.TradeType != "" {
		params.Set("tradeType", arg.TradeType)
	}
	if arg.RebateType != "" {
		params.Set("rebateType", arg.RebateType)
	}
	if !arg.StartAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(arg.StartAt.UnixMilli(), 10))
	}
	if !arg.EndAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(arg.EndAt.UnixMilli(), 10))
	}
	if arg.Page != 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	var resp *BrokerProCommissionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getBrokerProCommissionEPL, http.MethodGet, common.EncodeURLValues("/v2/broker/queryMyCommission", params), nil, &resp)
}

// GetBrokerProUserList returns the users trading through the Broker Pro, with the volume, fees and commission each
// contributed (Get User List)
func (e *Exchange) GetBrokerProUserList(ctx context.Context, arg *GetBrokerProUserListRequest) (*BrokerProUserListResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartAt.IsZero() && !arg.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartAt, arg.EndAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.TradeType != "" {
		params.Set("tradeType", arg.TradeType)
	}
	if arg.UID != "" {
		params.Set("uid", arg.UID)
	}
	if arg.ReferralCode != "" {
		params.Set("rcode", arg.ReferralCode)
	}
	if arg.Tag != "" {
		params.Set("tag", arg.Tag)
	}
	if !arg.StartAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(arg.StartAt.UnixMilli(), 10))
	}
	if !arg.EndAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(arg.EndAt.UnixMilli(), 10))
	}
	if arg.Page != 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	var resp *BrokerProUserListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getBrokerProUserListEPL, http.MethodGet, common.EncodeURLValues("/v2/broker/queryUser", params), nil, &resp)
}

// GetBrokerProUserTransactions returns the transactions of the users trading through the Broker Pro, with the volume,
// fees and commission each contributed; a page continues from the LastID of the previous page's last transaction (Get
// User Transactions)
func (e *Exchange) GetBrokerProUserTransactions(ctx context.Context, arg *GetBrokerProUserTransactionsRequest) ([]BrokerProUserTransactionResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartAt.IsZero() && !arg.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartAt, arg.EndAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.TradeType != "" {
		params.Set("tradeType", arg.TradeType)
	}
	if arg.UID != "" {
		params.Set("uid", arg.UID)
	}
	if !arg.StartAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(arg.StartAt.UnixMilli(), 10))
	}
	if !arg.EndAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(arg.EndAt.UnixMilli(), 10))
	}
	if arg.LastID != "" {
		params.Set("lastId", arg.LastID)
	}
	if arg.Direction != "" {
		params.Set("direction", arg.Direction)
	}
	if arg.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	var resp []BrokerProUserTransactionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getBrokerProUserTransactionsEPL, http.MethodGet, common.EncodeURLValues("/v2/broker/queryDetailByUid", params), nil, &resp)
}

// brokerPeriodParams returns the begin, end and tradeType query parameters, all required, of the broker rebate and
// broker info endpoints. Their begin and end are days, formatted YYYYMMDD, so a single day is a valid period
func brokerPeriodParams(begin, end time.Time, tradeType string) (url.Values, error) {
	if begin.IsZero() {
		return nil, fmt.Errorf("begin %w", common.ErrDateUnset)
	}
	if end.IsZero() {
		return nil, fmt.Errorf("end %w", common.ErrDateUnset)
	}
	beginDay, endDay := begin.UTC().Format("20060102"), end.UTC().Format("20060102")
	if beginDay > endDay {
		return nil, common.ErrStartAfterEnd
	}
	if tradeType == "" {
		return nil, errTradeTypeMissing
	}
	params := url.Values{}
	params.Set("begin", beginDay)
	params.Set("end", endDay)
	params.Set("tradeType", tradeType)
	return params, nil
}
