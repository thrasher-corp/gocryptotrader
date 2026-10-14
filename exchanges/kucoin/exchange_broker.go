package kucoin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Exchange Broker request validation errors
var (
	errBrokerClientUIDRequired      = errors.New("client UID is required")
	errBrokerFirstNameRequired      = errors.New("first name is required")
	errBrokerLastNameRequired       = errors.New("last name is required")
	errBrokerIssueCountryRequired   = errors.New("issue country is required")
	errBrokerIdentityTypeRequired   = errors.New("identity type is required")
	errBrokerIdentityNumberRequired = errors.New("identity number is required")
	errBrokerFacePhotoRequired      = errors.New("face photo is required")
	errBrokerAPIKeyLabelRequired    = errors.New("API key label is required")
	errBrokerDepositHashRequired    = errors.New("deposit hash is required")
	errBrokerWithdrawalIDRequired   = errors.New("withdrawal ID is required")
	errBrokerEffectiveTypeRequired  = errors.New("mark-up effective type is required")
)

// GetExchangeBrokerRebate returns a link, valid for one day, to a CSV file of the Exchange Broker's rebate orders of a
// trade type between two days at most six months apart (Get Broker Rebate)
func (e *Exchange) GetExchangeBrokerRebate(ctx context.Context, arg *GetExchangeBrokerRebateRequest) (*BrokerRebateDownloadResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := brokerPeriodParams(arg.Begin, arg.End, arg.TradeType)
	if err != nil {
		return nil, err
	}
	var resp *BrokerRebateDownloadResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, getExchangeBrokerRebateEPL, http.MethodGet, common.EncodeURLValues("/v1/broker/nd/rebate/download", params), nil, &resp)
}

// GetExchangeBrokerRebateV3 returns a link to a CSV file of the Exchange Broker's rebate orders of a trade type between
// two days, which adds the site and mark-up columns to GetExchangeBrokerRebate's file (Get Broker RebateV3)
func (e *Exchange) GetExchangeBrokerRebateV3(ctx context.Context, arg *GetExchangeBrokerRebateV3Request) (*BrokerRebateDownloadResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := brokerPeriodParams(arg.Begin, arg.End, arg.TradeType)
	if err != nil {
		return nil, err
	}
	var resp *BrokerRebateDownloadResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, getExchangeBrokerRebateV3EPL, http.MethodGet, common.EncodeURLValues("/v3/broker/nd/rebate/download", params), nil, &resp)
}

// SubmitExchangeBrokerKYC submits the KYC information of an Exchange Broker sub-account (Submit KYC)
func (e *Exchange) SubmitExchangeBrokerKYC(ctx context.Context, arg *SubmitExchangeBrokerKYCRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	switch {
	case arg.ClientUID == "":
		return errBrokerClientUIDRequired
	case arg.FirstName == "":
		return errBrokerFirstNameRequired
	case arg.LastName == "":
		return errBrokerLastNameRequired
	case arg.IssueCountry == "":
		return errBrokerIssueCountryRequired
	case arg.BirthDate.IsZero():
		return fmt.Errorf("birth %w", common.ErrDateUnset)
	case arg.IdentityType == "":
		return errBrokerIdentityTypeRequired
	case arg.IdentityNumber == "":
		return errBrokerIdentityNumberRequired
	case arg.ExpireDate.IsZero():
		return fmt.Errorf("expire %w", common.ErrDateUnset)
	case arg.FacePhoto == "":
		return errBrokerFacePhotoRequired
	}
	body := make(map[string]any)
	body["clientUid"] = arg.ClientUID
	body["firstName"] = arg.FirstName
	body["lastName"] = arg.LastName
	body["issueCountry"] = arg.IssueCountry
	body["birthDate"] = arg.BirthDate.UTC().Format("2006-01-02")
	body["identityType"] = arg.IdentityType
	body["identityNumber"] = arg.IdentityNumber
	body["expireDate"] = arg.ExpireDate.UTC().Format("2006-01-02")
	body["facePhoto"] = arg.FacePhoto
	if arg.FrontPhoto != "" {
		body["frontPhoto"] = arg.FrontPhoto
	}
	if arg.BackendPhoto != "" {
		body["backendPhoto"] = arg.BackendPhoto
	}
	// The result is typed as a string, null in the docs' example, and carries no information
	var resp string
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, submitExchangeBrokerKYCEPL, http.MethodPost, "/kyc/ndBroker/proxyClient/submit", body, &resp)
}

// GetExchangeBrokerKYCStatus returns the KYC status of Exchange Broker sub-accounts (Get KYC Status)
func (e *Exchange) GetExchangeBrokerKYCStatus(ctx context.Context, clientUIDs []string) ([]ExchangeBrokerKYCStatus, error) {
	if len(clientUIDs) == 0 || slices.Contains(clientUIDs, "") {
		return nil, errBrokerClientUIDRequired
	}
	params := url.Values{}
	params.Set("clientUids", strings.Join(clientUIDs, ","))
	var resp []ExchangeBrokerKYCStatus
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, getExchangeBrokerKYCStatusEPL, http.MethodGet, common.EncodeURLValues("/kyc/ndBroker/proxyClient/status/list", params), nil, &resp)
}

// GetExchangeBrokerKYCStatusList returns a page of the KYC statuses of the Exchange Broker's sub-accounts; pageSize is
// at most 100, the API's default (Get KYC Status List)
func (e *Exchange) GetExchangeBrokerKYCStatusList(ctx context.Context, pageNumber, pageSize uint64) (*ExchangeBrokerKYCStatusListResponse, error) {
	params := url.Values{}
	if pageNumber != 0 {
		params.Set("pageNumber", strconv.FormatUint(pageNumber, 10))
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(pageSize, 10))
	}
	var resp *ExchangeBrokerKYCStatusListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, getExchangeBrokerKYCStatusListEPL, http.MethodGet, common.EncodeURLValues("/kyc/ndBroker/proxyClient/status/page", params), nil, &resp)
}

// GetExchangeBrokerInfo returns the Exchange Broker's level and sub-account quota (Get Broker Info)
func (e *Exchange) GetExchangeBrokerInfo(ctx context.Context, arg *GetExchangeBrokerInfoRequest) (*ExchangeBrokerInfoResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := brokerPeriodParams(arg.Begin, arg.End, arg.TradeType)
	if err != nil {
		return nil, err
	}
	var resp *ExchangeBrokerInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, getExchangeBrokerInfoEPL, http.MethodGet, common.EncodeURLValues("/v1/broker/nd/info", params), nil, &resp)
}

// AddExchangeBrokerSubAccount creates an Exchange Broker sub-account, whose name must be unique across the exchange
// (Add sub-account)
func (e *Exchange) AddExchangeBrokerSubAccount(ctx context.Context, accountName string) (*AddExchangeBrokerSubAccountResponse, error) {
	if accountName == "" {
		return nil, errInvalidSubAccountName
	}
	var resp *AddExchangeBrokerSubAccountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, addExchangeBrokerSubAccountEPL, http.MethodPost, "/v1/broker/nd/account", map[string]any{"accountName": accountName}, &resp)
}

// GetExchangeBrokerSubAccounts returns a page of the sub-accounts the Exchange Broker created (Get sub-account)
func (e *Exchange) GetExchangeBrokerSubAccounts(ctx context.Context, arg *GetExchangeBrokerSubAccountsRequest) (*ExchangeBrokerSubAccountsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if arg.UID != "" {
		params.Set("uid", arg.UID)
	}
	if arg.CurrentPage != 0 {
		params.Set("currentPage", strconv.FormatUint(arg.CurrentPage, 10))
	}
	if arg.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	var resp *ExchangeBrokerSubAccountsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, getExchangeBrokerSubAccountsEPL, http.MethodGet, common.EncodeURLValues("/v1/broker/nd/account", params), nil, &resp)
}

// AddExchangeBrokerSubAccountAPI creates an API key for an Exchange Broker sub-account; the response is the only one
// that carries the key's secret (Add sub-account API)
func (e *Exchange) AddExchangeBrokerSubAccountAPI(ctx context.Context, arg *AddExchangeBrokerSubAccountAPIRequest) (*AddExchangeBrokerSubAccountAPIResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.UID == "" {
		return nil, errSubUserIDRequired
	}
	if arg.Passphrase == "" {
		return nil, errInvalidPassPhraseInstance
	}
	if arg.Label == "" {
		return nil, errBrokerAPIKeyLabelRequired
	}
	body := make(map[string]any)
	body["uid"] = arg.UID
	body["passphrase"] = arg.Passphrase
	body["label"] = arg.Label
	if len(arg.IPWhitelist) != 0 {
		body["ipWhitelist"] = arg.IPWhitelist
	}
	if len(arg.Permissions) != 0 {
		body["permissions"] = arg.Permissions
	}
	var resp *AddExchangeBrokerSubAccountAPIResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, addExchangeBrokerSubAccountAPIEPL, http.MethodPost, "/v1/broker/nd/account/apikey", body, &resp)
}

// GetExchangeBrokerSubAccountAPIs returns the API keys of an Exchange Broker sub-account, filtered to apiKey when it is
// set (Get sub-account API)
func (e *Exchange) GetExchangeBrokerSubAccountAPIs(ctx context.Context, uid, apiKey string) ([]ExchangeBrokerSubAccountAPI, error) {
	if uid == "" {
		return nil, errSubUserIDRequired
	}
	params := url.Values{}
	params.Set("uid", uid)
	if apiKey != "" {
		params.Set("apiKey", apiKey)
	}
	var resp []ExchangeBrokerSubAccountAPI
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, getExchangeBrokerSubAccountAPIsEPL, http.MethodGet, common.EncodeURLValues("/v1/broker/nd/account/apikey", params), nil, &resp)
}

// ModifyExchangeBrokerSubAccountAPI changes the label, permissions and IP whitelist of an Exchange Broker sub-account's
// API key (Modify sub-account API)
func (e *Exchange) ModifyExchangeBrokerSubAccountAPI(ctx context.Context, arg *ModifyExchangeBrokerSubAccountAPIRequest) (*ModifyExchangeBrokerSubAccountAPIResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.UID == "" {
		return nil, errSubUserIDRequired
	}
	if arg.APIKey == "" {
		return nil, errAPIKeyRequired
	}
	if arg.Label == "" {
		return nil, errBrokerAPIKeyLabelRequired
	}
	body := make(map[string]any)
	body["uid"] = arg.UID
	body["apiKey"] = arg.APIKey
	body["label"] = arg.Label
	if len(arg.IPWhitelist) != 0 {
		body["ipWhitelist"] = arg.IPWhitelist
	}
	if len(arg.Permissions) != 0 {
		body["permissions"] = arg.Permissions
	}
	var resp *ModifyExchangeBrokerSubAccountAPIResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, modifyExchangeBrokerSubAccountAPIEPL, http.MethodPost, "/v1/broker/nd/account/update-apikey", body, &resp)
}

// DeleteExchangeBrokerSubAccountAPI deletes an API key of an Exchange Broker sub-account, returning whether it was
// deleted (Delete sub-account API)
func (e *Exchange) DeleteExchangeBrokerSubAccountAPI(ctx context.Context, uid, apiKey string) (bool, error) {
	if uid == "" {
		return false, errSubUserIDRequired
	}
	if apiKey == "" {
		return false, errAPIKeyRequired
	}
	params := url.Values{}
	params.Set("uid", uid)
	params.Set("apiKey", apiKey)
	var resp bool
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, deleteExchangeBrokerSubAccountAPIEPL, http.MethodDelete, common.EncodeURLValues("/v1/broker/nd/account/apikey", params), nil, &resp)
}

// ExchangeBrokerTransfer transfers funds between the Exchange Broker's account and one of its sub-accounts; a
// sub-account cannot withdraw, so its funds are transferred to the broker account to withdraw them (Transfer)
func (e *Exchange) ExchangeBrokerTransfer(ctx context.Context, arg *ExchangeBrokerTransferRequest) (*ExchangeBrokerTransferResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	switch {
	case arg.Currency.IsEmpty():
		return nil, currency.ErrCurrencyCodeEmpty
	case arg.Amount <= 0:
		return nil, limits.ErrAmountBelowMin
	case arg.Direction == "":
		return nil, errTransferDirectionRequired
	case arg.AccountType == "":
		return nil, fmt.Errorf("%w: accountType", errAccountTypeMissing)
	case arg.SpecialUID == "":
		return nil, errSubUserIDRequired
	case arg.SpecialAccountType == "":
		return nil, fmt.Errorf("%w: specialAccountType", errAccountTypeMissing)
	case arg.ClientOrderID == "":
		return nil, order.ErrClientOrderIDMustBeSet
	}
	body := make(map[string]any)
	body["currency"] = arg.Currency.Upper().String()
	body["amount"] = strconv.FormatFloat(arg.Amount, 'f', -1, 64)
	body["direction"] = arg.Direction
	body["accountType"] = arg.AccountType
	body["specialUid"] = arg.SpecialUID
	body["specialAccountType"] = arg.SpecialAccountType
	body["clientOid"] = arg.ClientOrderID
	var resp *ExchangeBrokerTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, exchangeBrokerTransferEPL, http.MethodPost, "/v1/broker/nd/transfer", body, &resp)
}

// GetExchangeBrokerTransferHistory returns a transfer of the Exchange Broker or one of its sub-accounts (Get Transfer
// History)
func (e *Exchange) GetExchangeBrokerTransferHistory(ctx context.Context, orderID string) (*ExchangeBrokerTransferHistoryResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("orderId", orderID)
	var resp *ExchangeBrokerTransferHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, getExchangeBrokerTransferHistoryEPL, http.MethodGet, common.EncodeURLValues("/v3/broker/nd/transfer/detail", params), nil, &resp)
}

// GetExchangeBrokerDepositList returns the deposits to the Exchange Broker's sub-accounts, newest first (Get Deposit
// List)
func (e *Exchange) GetExchangeBrokerDepositList(ctx context.Context, arg *GetExchangeBrokerDepositListRequest) ([]ExchangeBrokerDeposit, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartTimestamp.IsZero() && !arg.EndTimestamp.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTimestamp, arg.EndTimestamp); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.Upper().String())
	}
	if arg.Status != "" {
		params.Set("status", arg.Status)
	}
	if arg.Hash != "" {
		params.Set("hash", arg.Hash)
	}
	if !arg.StartTimestamp.IsZero() {
		params.Set("startTimestamp", strconv.FormatInt(arg.StartTimestamp.UnixMilli(), 10))
	}
	if !arg.EndTimestamp.IsZero() {
		params.Set("endTimestamp", strconv.FormatInt(arg.EndTimestamp.UnixMilli(), 10))
	}
	if arg.Limit != 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []ExchangeBrokerDeposit
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, getExchangeBrokerDepositListEPL, http.MethodGet, common.EncodeURLValues("/v1/asset/ndbroker/deposit/list", params), nil, &resp)
}

// GetExchangeBrokerDepositDetail returns a deposit to one of the Exchange Broker's sub-accounts, not to the broker
// account itself (Get Deposit Detail)
func (e *Exchange) GetExchangeBrokerDepositDetail(ctx context.Context, ccy currency.Code, hash string) (*ExchangeBrokerDepositDetailResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if hash == "" {
		return nil, errBrokerDepositHashRequired
	}
	params := url.Values{}
	params.Set("currency", ccy.Upper().String())
	params.Set("hash", hash)
	var resp *ExchangeBrokerDepositDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, getExchangeBrokerDepositDetailEPL, http.MethodGet, common.EncodeURLValues("/v3/broker/nd/deposit/detail", params), nil, &resp)
}

// GetExchangeBrokerWithdrawDetail returns a withdrawal from one of the Exchange Broker's sub-accounts, not from the
// broker account itself (Get Withdraw Detail)
func (e *Exchange) GetExchangeBrokerWithdrawDetail(ctx context.Context, withdrawalID string) (*ExchangeBrokerWithdrawDetailResponse, error) {
	if withdrawalID == "" {
		return nil, errBrokerWithdrawalIDRequired
	}
	params := url.Values{}
	params.Set("withdrawalId", withdrawalID)
	var resp *ExchangeBrokerWithdrawDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, getExchangeBrokerWithdrawDetailEPL, http.MethodGet, common.EncodeURLValues("/v3/broker/nd/withdraw/detail", params), nil, &resp)
}

// SetExchangeBrokerMarkupFee sets the Exchange Broker's default mark-up of a trade type, or that of the sub-accounts
// listed (Set Markup Fee)
func (e *Exchange) SetExchangeBrokerMarkupFee(ctx context.Context, arg *SetExchangeBrokerMarkupFeeRequest) (*SetExchangeBrokerMarkupFeeResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.TradeType == "" {
		return nil, errTradeTypeMissing
	}
	if arg.EffectiveType == "" {
		return nil, errBrokerEffectiveTypeRequired
	}
	// Both mark-ups are always sent as the JSON numbers KuCoin documents: the API rejects one without the other, and
	// zero cancels a mark-up
	body := make(map[string]any)
	body["tradeType"] = arg.TradeType
	body["effectiveType"] = arg.EffectiveType
	body["makerMarkUp"] = arg.MakerMarkUp
	body["takerMarkUp"] = arg.TakerMarkUp
	if len(arg.SubUIDs) != 0 {
		body["subUids"] = arg.SubUIDs
	}
	if !arg.EffectAt.IsZero() {
		body["effectAt"] = arg.EffectAt.UnixMilli()
	}
	var resp *SetExchangeBrokerMarkupFeeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, setExchangeBrokerMarkupFeeEPL, http.MethodPost, "/v1/broker/nd/mark-up", body, &resp)
}

// GetExchangeBrokerMarkupFee returns the Exchange Broker's default mark-up and those of up to 20 sub-accounts (Get
// Markup Fee)
func (e *Exchange) GetExchangeBrokerMarkupFee(ctx context.Context, subUIDs []uint64) (*ExchangeBrokerMarkupFeeResponse, error) {
	params := url.Values{}
	if len(subUIDs) != 0 {
		uids := make([]string, len(subUIDs))
		for i := range subUIDs {
			uids[i] = strconv.FormatUint(subUIDs[i], 10)
		}
		params.Set("subUids", strings.Join(uids, ","))
	}
	var resp *ExchangeBrokerMarkupFeeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, getExchangeBrokerMarkupFeeEPL, http.MethodGet, common.EncodeURLValues("/v1/broker/nd/mark-up", params), nil, &resp)
}
