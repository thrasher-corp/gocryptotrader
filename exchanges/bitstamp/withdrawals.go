package bitstamp

import (
	"context"
	"fmt"
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

// GetWithdrawalRequests returns the account's withdrawal requests
func (e *Exchange) GetWithdrawalRequests(ctx context.Context, req *WithdrawalRequestsRequest) ([]WithdrawalRequestResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.TimeDelta != 0 && (req.TimeDelta < time.Second || req.TimeDelta > maxWithdrawalRequestsTimeDelta) {
		return nil, fmt.Errorf("%w: %s", errInvalidTimeDelta, req.TimeDelta)
	}
	if err := validatePaging(req.Limit, req.Offset); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.ID != 0 {
		params.Set("id", strconv.FormatUint(req.ID, 10))
	}
	if req.TimeDelta != 0 {
		params.Set("timedelta", strconv.FormatInt(int64(req.TimeDelta/time.Second), 10))
	}
	// The limit and offset parameters must be sent together
	if req.Limit != 0 || req.Offset != 0 {
		limit := req.Limit
		if limit == 0 {
			limit = maxResultLimit
		}
		params.Set("limit", strconv.FormatUint(limit, 10))
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	var resp []WithdrawalRequestResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/withdrawal-requests/", nil, params, &resp)
}

// OpenBankWithdrawal opens a SEPA or international bank withdrawal request from the main account
func (e *Exchange) OpenBankWithdrawal(ctx context.Context, req *BankWithdrawalRequest) (*BankWithdrawalResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.AccountCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Name == "" || req.IBAN == "" || req.BIC == "" || req.Address == "" || req.PostalCode == "" || req.City == "" || req.Country == "" || req.Type == "" {
		return nil, errBankDetailsRequired
	}
	params := url.Values{}
	setFloat(params, "amount", req.Amount)
	params.Set("account_currency", req.AccountCurrency.Upper().String())
	params.Set("name", req.Name)
	params.Set("iban", req.IBAN)
	params.Set("bic", req.BIC)
	params.Set("address", req.Address)
	params.Set("postal_code", req.PostalCode)
	params.Set("city", req.City)
	params.Set("country", req.Country)
	params.Set("type", req.Type)
	for k, v := range map[string]string{
		"comment":                     req.Comment,
		"bank_name":                   req.BankName,
		"bank_address":                req.BankAddress,
		"bank_postal_code":            req.BankPostalCode,
		"bank_city":                   req.BankCity,
		"bank_country":                req.BankCountry,
		"intermed_routing_num_or_bic": req.IntermediaryRoutingNumberOrBIC,
	} {
		if v != "" {
			params.Set(k, v)
		}
	}
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	var resp *BankWithdrawalResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/withdrawal/open/", nil, params, &resp)
}

// CancelWithdrawal cancels a bank or cryptocurrency withdrawal request from the main account
func (e *Exchange) CancelWithdrawal(ctx context.Context, withdrawalID uint64) (*CancelWithdrawalResponse, error) {
	if withdrawalID == 0 {
		return nil, errWithdrawalIDRequired
	}
	params := url.Values{}
	params.Set("id", strconv.FormatUint(withdrawalID, 10))
	var resp *CancelWithdrawalResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/withdrawal/cancel/", nil, params, &resp)
}

// GetFiatWithdrawalStatus returns the status of a bank withdrawal request from the main account
func (e *Exchange) GetFiatWithdrawalStatus(ctx context.Context, withdrawalID uint64) (*WithdrawalStatusResponse, error) {
	if withdrawalID == 0 {
		return nil, errWithdrawalIDRequired
	}
	params := url.Values{}
	params.Set("id", strconv.FormatUint(withdrawalID, 10))
	var resp *WithdrawalStatusResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/withdrawal/status/", nil, params, &resp)
}

// CryptoWithdrawal requests a cryptocurrency withdrawal
func (e *Exchange) CryptoWithdrawal(ctx context.Context, req *CryptoWithdrawalRequest) (*CryptoWithdrawalResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.Address == "" {
		return nil, common.ErrAddressIsEmptyOrInvalid
	}
	travelRule := &travelRuleParams{
		OriginatorInfo:                  req.OriginatorInfo,
		BeneficiaryInfo:                 req.BeneficiaryInfo,
		BeneficiaryThirdParty:           req.BeneficiaryThirdParty,
		BeneficiaryID:                   req.BeneficiaryID,
		VirtualAssetServiceProviderUUID: req.VirtualAssetServiceProviderUUID,
	}
	if err := validateTravelRuleParams(travelRule); err != nil {
		return nil, err
	}
	params := url.Values{}
	setFloat(params, "amount", req.Amount)
	params.Set("address", req.Address)
	if req.Network != "" {
		params.Set("network", req.Network)
	}
	if req.MemoID != "" {
		params.Set("memo_id", req.MemoID)
	}
	if req.DestinationTag != "" {
		params.Set("destination_tag", req.DestinationTag)
	}
	if req.TransferID != 0 {
		params.Set("transfer_id", strconv.FormatUint(req.TransferID, 10))
	}
	if err := setTravelRuleParams(params, travelRule); err != nil {
		return nil, err
	}
	var resp *CryptoWithdrawalResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/"+req.Currency.Lower().String()+"_withdrawal/", nil, params, &resp)
}

// RippleIOUWithdrawal requests a USD, BTC, EUR or ETH IOU withdrawal on the XRP Ledger
func (e *Exchange) RippleIOUWithdrawal(ctx context.Context, req *RippleIOUWithdrawalRequest) (*CryptoWithdrawalResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.Address == "" {
		return nil, common.ErrAddressIsEmptyOrInvalid
	}
	travelRule := &travelRuleParams{
		OriginatorInfo:                  req.OriginatorInfo,
		BeneficiaryInfo:                 req.BeneficiaryInfo,
		BeneficiaryThirdParty:           req.BeneficiaryThirdParty,
		BeneficiaryID:                   req.BeneficiaryID,
		VirtualAssetServiceProviderUUID: req.VirtualAssetServiceProviderUUID,
	}
	if err := validateTravelRuleParams(travelRule); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("currency", req.Currency.Upper().String())
	setFloat(params, "amount", req.Amount)
	params.Set("address", req.Address)
	if err := setTravelRuleParams(params, travelRule); err != nil {
		return nil, err
	}
	var resp *CryptoWithdrawalResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/ripple_withdrawal/", nil, params, &resp)
}

// validateTravelRuleParams checks the Travel Rule details of a cryptocurrency withdrawal
// The originator details describe the sender and must identify them, and a third party beneficiary must be identified by
// a contact ID or details
func validateTravelRuleParams(p *travelRuleParams) error {
	if p.OriginatorInfo != nil {
		if err := validateIdentifiedCustomer(p.OriginatorInfo); err != nil {
			return fmt.Errorf("originator: %w", err)
		}
	}
	if p.BeneficiaryInfo != nil {
		if err := validateContact(p.BeneficiaryInfo.RetailInfo, p.BeneficiaryInfo.CorporateInfo); err != nil {
			return fmt.Errorf("beneficiary: %w", err)
		}
	}
	if p.BeneficiaryThirdParty != nil && *p.BeneficiaryThirdParty && p.BeneficiaryID == "" && p.BeneficiaryInfo == nil {
		return errBeneficiaryRequired
	}
	return nil
}

// validateContact checks that Travel Rule contact details hold the names the API requires
func validateContact(retail *RetailInfo, corporate *CorporateInfo) error {
	if retail == nil && corporate == nil {
		return errContactInfoRequired
	}
	if retail != nil && (retail.FirstName == "" || retail.LastName == "") {
		return errContactNameRequired
	}
	if corporate != nil && corporate.CompanyName == "" {
		return errCompanyNameRequired
	}
	return nil
}

// validateIdentifiedCustomer checks Travel Rule details which must also identify the customer: a retail customer by
// date and place of birth or by identity document, and a corporate customer by LEI or company ID
func validateIdentifiedCustomer(info *CustomerInfo) error {
	if err := validateContact(info.RetailInfo, info.CorporateInfo); err != nil {
		return err
	}
	if r := info.RetailInfo; r != nil && (r.DateOfBirth.IsZero() || r.PlaceOfBirth == "") && (r.IDType == "" || r.IDNumber == "") {
		return fmt.Errorf("%w: date and place of birth or identity document type and number", errIdentificationRequired)
	}
	if c := info.CorporateInfo; c != nil && c.LEI == "" && c.CompanyID == "" {
		return fmt.Errorf("%w: LEI or company ID", errIdentificationRequired)
	}
	return nil
}

func setTravelRuleParams(params url.Values, p *travelRuleParams) error {
	// Contact details are submitted as JSON encoded form values
	for k, v := range map[string]*CustomerInfo{"originator_info": p.OriginatorInfo, "beneficiary_info": p.BeneficiaryInfo} {
		if v == nil {
			continue
		}
		info, err := json.Marshal(v)
		if err != nil {
			return err
		}
		params.Set(k, string(info))
	}
	if p.BeneficiaryThirdParty != nil {
		params.Set("beneficiary_thirdparty", formatBool(*p.BeneficiaryThirdParty))
	}
	if p.BeneficiaryID != "" {
		params.Set("beneficiary_id", p.BeneficiaryID)
	}
	if p.VirtualAssetServiceProviderUUID != "" {
		params.Set("vasp_uuid", p.VirtualAssetServiceProviderUUID)
	}
	return nil
}
