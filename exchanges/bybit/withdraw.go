package bybit

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// forceChainUID is the forceChain value that withdraws to a Bybit UID, the only mode that needs no chain
const forceChainUID = 2

var errLegalTypeRequired = errors.New("legal type required")

// GetWithdrawalAddressList returns the withdrawal addresses in the address book; the API key needs the withdrawal
// permission (Get Withdrawal Address List)
func (e *Exchange) GetWithdrawalAddressList(ctx context.Context, arg *WithdrawalAddressListRequest) (*WithdrawalAddressListResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if arg.Coin != "" {
		params.Set("coin", arg.Coin)
	}
	if arg.Chain != "" {
		params.Set("chain", arg.Chain)
	}
	if arg.AddressType > 0 {
		params.Set("addressType", strconv.FormatUint(arg.AddressType, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Cursor != "" {
		params.Set("cursor", arg.Cursor)
	}
	var resp *WithdrawalAddressListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/withdraw/query-address", params, nil, &resp, getWithdrawalAddressListEPL)
}

// GetWithdrawalRecords returns withdrawals, using the master UID's API key only. Without a window the API returns the
// last 30 days, and a window spans less than 30 days (Get Withdrawal Records)
func (e *Exchange) GetWithdrawalRecords(ctx context.Context, arg *WithdrawalRecordsRequest) (*WithdrawalRecordsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.WithdrawID != "" {
		params.Set("withdrawID", arg.WithdrawID)
	}
	if arg.TransactionID != "" {
		params.Set("txID", arg.TransactionID)
	}
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.Upper().String())
	}
	if arg.WithdrawType > 0 {
		params.Set("withdrawType", strconv.FormatUint(arg.WithdrawType, 10))
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
	var resp *WithdrawalRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/withdraw/query-record", params, nil, &resp, getWithdrawalRecordsEPL)
}

// GetAvailableVASPs returns the virtual asset service providers a Travel Rule withdrawal can name, for the compliance
// zone of the API key's UID (Get available VASPs)
func (e *Exchange) GetAvailableVASPs(ctx context.Context) (*AvailableVASPsResponse, error) {
	var resp *AvailableVASPsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/withdraw/vasp/list", nil, nil, &resp, getAvailableVASPsEPL)
}

// WithdrawCurrency withdraws a coin to an address in the address book, or to a Bybit UID; a Bybit address is paid
// off-chain without a blockchain fee. It needs the master UID's API key, and a chain and coin pair can withdraw once
// every 10 seconds (Withdraw)
func (e *Exchange) WithdrawCurrency(ctx context.Context, arg *WithdrawalRequest) (*WithdrawalResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Chain == "" && arg.ForceChain != forceChainUID {
		return nil, errMissingChainInformation
	}
	if arg.Address == "" {
		return nil, errMissingAddressInfo
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.AccountType == "" {
		return nil, errMissingAccountType
	}
	timestamp := arg.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	body := make(map[string]any)
	body["coin"] = arg.Coin.Upper().String()
	body["address"] = arg.Address
	body["amount"] = strconv.FormatFloat(arg.Amount, 'f', -1, 64)
	body["timestamp"] = timestamp.UnixMilli()
	body["accountType"] = arg.AccountType
	if arg.Chain != "" {
		body["chain"] = arg.Chain
	}
	if arg.Tag != "" {
		body["tag"] = arg.Tag
	}
	if arg.ForceChain != 0 {
		body["forceChain"] = arg.ForceChain
	}
	if arg.FeeType != 0 {
		body["feeType"] = arg.FeeType
	}
	if arg.RequestID != "" {
		body["requestId"] = arg.RequestID
	}
	if arg.TransactionPurpose != "" {
		body["transactionPurpose"] = arg.TransactionPurpose
	}
	if arg.Questionnaire != nil {
		questionnaire, err := arg.Questionnaire.encode()
		if err != nil {
			return nil, err
		}
		body["questionnaire"] = questionnaire
	}
	if arg.Beneficiary != nil {
		body["beneficiary"] = arg.Beneficiary.payload()
	}
	var resp *WithdrawalResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/withdraw/create", nil, body, &resp, withdrawCurrencyEPL)
}

// payload returns the beneficiary object the Withdraw endpoint takes, with the keys of the fields that are set
func (b *WithdrawalBeneficiary) payload() map[string]any {
	fields := make(map[string]any)
	for key, value := range map[string]string{
		"beneficiaryTransactionPurpose":      b.TransactionPurpose,
		"beneficiaryRepresentativeFirstName": b.RepresentativeFirstName,
		"beneficiaryRepresentativeLastName":  b.RepresentativeLastName,
		"vaspEntityId":                       b.VASPEntityID,
		"beneficiaryName":                    b.Name,
		"beneficiaryLegalType":               b.LegalType,
		"beneficiaryWalletType":              b.WalletType,
		"beneficiaryUnhostedWalletType":      b.UnhostedWalletType,
		"beneficiaryPoiNumber":               b.POINumber,
		"beneficiaryPoiType":                 b.POIType,
		"beneficiaryPoiIssuingCountry":       b.POIIssuingCountry,
		"beneficiaryAddressCountry":          b.AddressCountry,
		"beneficiaryAddressState":            b.AddressState,
		"beneficiaryAddressCity":             b.AddressCity,
		"beneficiaryAddressBuilding":         b.AddressBuilding,
		"beneficiaryAddressStreet":           b.AddressStreet,
		"beneficiaryAddressPostalCode":       b.AddressPostalCode,
		"beneficiaryDateOfBirth":             b.DateOfBirth,
		"beneficiaryPlaceOfBirth":            b.PlaceOfBirth,
	} {
		if value != "" {
			fields[key] = value
		}
	}
	if !b.POIExpiredDate.IsZero() {
		fields["beneficiaryPoiExpiredDate"] = b.POIExpiredDate.Format(time.DateOnly)
	}
	return fields
}

// CancelWithdrawal cancels a withdrawal (Cancel Withdrawal)
func (e *Exchange) CancelWithdrawal(ctx context.Context, id string) (*StatusResponse, error) {
	if id == "" {
		return nil, errMissingWithdrawalID
	}
	body := map[string]any{"id": id}
	var resp *StatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/withdraw/cancel", nil, body, &resp, cancelWithdrawalEPL)
}

// encode returns the questionnaire as the JSON string the Withdraw and Submit Deposit Originator Info endpoints take
// (Questionnaire)
func (q *TravelRuleQuestionnaire) encode() (string, error) {
	if q.LegalType == "" {
		return "", errLegalTypeRequired
	}
	fields := map[string]any{
		"walletType": q.WalletType,
		"legalType":  q.LegalType,
	}
	for key, value := range map[string]string{
		"vaspCode":                q.VASPCode,
		"firstName":               q.FirstName,
		"lastName":                q.LastName,
		"companyName":             q.CompanyName,
		"transactionPurpose":      q.TransactionPurpose,
		"poiType":                 q.POIType,
		"poiNumber":               q.POINumber,
		"poiIssuingCountry":       q.POIIssuingCountry,
		"representativeFirstName": q.RepresentativeFirstName,
		"representativeLastName":  q.RepresentativeLastName,
	} {
		if value != "" {
			fields[key] = value
		}
	}
	if q.IsSelfWallet != nil {
		fields["isSelfWallet"] = *q.IsSelfWallet
	}
	if !q.POIExpiredDate.IsZero() {
		fields["poiExpiredDate"] = q.POIExpiredDate.Format(time.DateOnly)
	}
	encoded, err := json.Marshal(fields)
	return string(encoded), err
}
