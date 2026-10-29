package bitstamp

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestSetTravelRuleParams(t *testing.T) {
	t.Parallel()
	params := url.Values{}
	require.NoError(t, setTravelRuleParams(params, &travelRuleParams{}), "setTravelRuleParams must not error without details")
	assert.Empty(t, params, "setTravelRuleParams should not set unset parameters")

	require.NoError(t, setTravelRuleParams(params, &travelRuleParams{
		OriginatorInfo:                  &CustomerInfo{RetailInfo: &RetailInfo{FirstName: "John", LastName: "Doe", DateOfBirth: time.Date(1990, 1, 31, 0, 0, 0, 0, time.UTC), PlaceOfBirth: "Sydney", PostalCode: "2000"}},
		BeneficiaryInfo:                 &CustomerInfo{CorporateInfo: &CorporateInfo{CompanyName: "Good company"}},
		BeneficiaryThirdParty:           new(true),
		BeneficiaryID:                   "1",
		VirtualAssetServiceProviderUUID: "48feffe5-a5d0-44fe-bf4c-734e7d721249",
	}), "setTravelRuleParams must not error")
	assert.Equal(t, url.Values{
		"originator_info":        {`{"retail_info":{"first_name":"John","last_name":"Doe","date_of_birth":"1990-01-31","place_of_birth":"Sydney","zip":"2000"}}`},
		"beneficiary_info":       {`{"corporate_info":{"company_name":"Good company"}}`},
		"beneficiary_thirdparty": {trueValue},
		"beneficiary_id":         {"1"},
		"vasp_uuid":              {"48feffe5-a5d0-44fe-bf4c-734e7d721249"},
	}, params, "setTravelRuleParams should JSON encode contact details")

	params = url.Values{}
	require.NoError(t, setTravelRuleParams(params, &travelRuleParams{BeneficiaryThirdParty: new(false)}), "setTravelRuleParams must not error for an own address")
	assert.Equal(t, url.Values{"beneficiary_thirdparty": {falseValue}}, params, "setTravelRuleParams should send a false third party flag")
}

func TestValidateTravelRuleParams(t *testing.T) {
	t.Parallel()
	retail := &RetailInfo{FirstName: "John", LastName: "Doe"}
	for _, tc := range []struct {
		name string
		p    *travelRuleParams
		err  error
	}{
		{name: "empty", p: &travelRuleParams{}},
		{name: "own address", p: &travelRuleParams{BeneficiaryThirdParty: new(false)}},
		{name: "third party without beneficiary", p: &travelRuleParams{BeneficiaryThirdParty: new(true)}, err: errBeneficiaryRequired},
		{name: "third party by ID", p: &travelRuleParams{BeneficiaryThirdParty: new(true), BeneficiaryID: "1"}},
		{name: "third party by details", p: &travelRuleParams{BeneficiaryThirdParty: new(true), BeneficiaryInfo: &CustomerInfo{RetailInfo: retail}}},
		{name: "empty beneficiary", p: &travelRuleParams{BeneficiaryInfo: &CustomerInfo{}}, err: errContactInfoRequired},
		{name: "unnamed beneficiary", p: &travelRuleParams{BeneficiaryInfo: &CustomerInfo{RetailInfo: &RetailInfo{FirstName: "John"}}}, err: errContactNameRequired},
		{name: "unidentified retail originator", p: &travelRuleParams{OriginatorInfo: &CustomerInfo{RetailInfo: retail}}, err: errIdentificationRequired},
		{name: "originator without place of birth", p: &travelRuleParams{OriginatorInfo: &CustomerInfo{RetailInfo: &RetailInfo{FirstName: "John", LastName: "Doe", DateOfBirth: time.Date(1990, 1, 31, 0, 0, 0, 0, time.UTC)}}}, err: errIdentificationRequired},
		{name: "originator identified by birth", p: &travelRuleParams{OriginatorInfo: &CustomerInfo{RetailInfo: &RetailInfo{FirstName: "John", LastName: "Doe", DateOfBirth: time.Date(1990, 1, 31, 0, 0, 0, 0, time.UTC), PlaceOfBirth: "Sydney"}}}},
		{name: "originator identified by document", p: &travelRuleParams{OriginatorInfo: &CustomerInfo{RetailInfo: &RetailInfo{FirstName: "John", LastName: "Doe", IDType: IDTypePassport, IDNumber: "PA1234567"}}}},
		{name: "unidentified corporate originator", p: &travelRuleParams{OriginatorInfo: &CustomerInfo{CorporateInfo: &CorporateInfo{CompanyName: "Good company"}}}, err: errIdentificationRequired},
		{name: "unnamed corporate originator", p: &travelRuleParams{OriginatorInfo: &CustomerInfo{CorporateInfo: &CorporateInfo{LEI: "5493001KJTIIGC8Y1R12"}}}, err: errCompanyNameRequired},
		{name: "corporate originator", p: &travelRuleParams{OriginatorInfo: &CustomerInfo{CorporateInfo: &CorporateInfo{CompanyName: "Good company", CompanyID: "123"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateTravelRuleParams(tc.p)
			if tc.err == nil {
				assert.NoError(t, err, "validateTravelRuleParams should not error")
				return
			}
			assert.ErrorIs(t, err, tc.err, "validateTravelRuleParams should return the correct error")
		})
	}
}

func TestGetWithdrawalRequests(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *WithdrawalRequestsRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &WithdrawalRequestsRequest{TimeDelta: maxWithdrawalRequestsTimeDelta + time.Second}, err: errInvalidTimeDelta},
		{req: &WithdrawalRequestsRequest{TimeDelta: -time.Second}, err: errInvalidTimeDelta},
		{req: &WithdrawalRequestsRequest{TimeDelta: time.Second - time.Nanosecond}, err: errInvalidTimeDelta},
		{req: &WithdrawalRequestsRequest{Limit: 1001}, err: errInvalidLimit},
		{req: &WithdrawalRequestsRequest{Offset: maxResultOffset + 1}, err: errInvalidOffset},
	} {
		_, err := e.GetWithdrawalRequests(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "GetWithdrawalRequests should return the correct error")
	}

	skipIfLiveWithoutCredentials(t)
	requests, err := e.GetWithdrawalRequests(t.Context(), &WithdrawalRequestsRequest{TimeDelta: 24 * time.Hour, Limit: 10})
	require.NoError(t, err, "GetWithdrawalRequests must not error")
	if mockTests {
		require.Len(t, requests, 1, "GetWithdrawalRequests must return the requests")
		exp := WithdrawalRequestResponse{
			ID:                    1,
			DateTime:              types.DateTime(time.Date(2022, 1, 31, 16, 7, 32, 0, time.UTC)),
			Type:                  1,
			Currency:              currency.BTC,
			Network:               "bitcoin",
			Amount:                0.00006,
			Status:                WithdrawalStatusFinished,
			BitstampTransactionID: 1,
			Address:               core.BitcoinDonationAddress,
			TransactionID:         "NsOeFbQhRnpGzNIThWGBTkQwRJqTNOGPVhYavrVyMfkAyMUmIlUpFIwGTzSvpeOP",
		}
		assert.Equal(t, exp, requests[0], "GetWithdrawalRequests should return the request")
	}
}

func TestOpenBankWithdrawal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *BankWithdrawalRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &BankWithdrawalRequest{}, err: order.ErrAmountIsInvalid},
		{req: &BankWithdrawalRequest{Amount: 1}, err: currency.ErrCurrencyCodeEmpty},
		{req: &BankWithdrawalRequest{Amount: 1, AccountCurrency: currency.EUR}, err: errBankDetailsRequired},
	} {
		_, err := e.OpenBankWithdrawal(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "OpenBankWithdrawal should return the correct error")
	}
	// Covered with valid requests by TestWithdrawFiatFunds and TestWithdrawFiatFundsToInternationalBank
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.CancelWithdrawal(t.Context(), 0)
	assert.ErrorIs(t, err, errWithdrawalIDRequired, "CancelWithdrawal should error without an ID")

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	resp, err := e.CancelWithdrawal(t.Context(), 1)
	if !mockTests {
		assert.ErrorIs(t, err, errAPIResponse, "CancelWithdrawal should error for an unknown withdrawal")
		return
	}
	require.NoError(t, err, "CancelWithdrawal must not error")
	exp := &CancelWithdrawalResponse{ID: 1, Amount: 100, Currency: currency.EUR, AccountCurrency: currency.EUR, Type: "EU bank transfer (SEPA)"}
	assert.Equal(t, exp, resp, "CancelWithdrawal should return the cancelled withdrawal")
}

func TestGetFiatWithdrawalStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatWithdrawalStatus(t.Context(), 0)
	assert.ErrorIs(t, err, errWithdrawalIDRequired, "GetFiatWithdrawalStatus should error without an ID")

	skipIfLiveWithoutCredentials(t)
	resp, err := e.GetFiatWithdrawalStatus(t.Context(), 1)
	if !mockTests {
		assert.ErrorIs(t, err, errAPIResponse, "GetFiatWithdrawalStatus should error for an unknown withdrawal")
		return
	}
	require.NoError(t, err, "GetFiatWithdrawalStatus must not error")
	assert.Equal(t, &WithdrawalStatusResponse{Status: "Waiting to be processed"}, resp, "GetFiatWithdrawalStatus should decode the status")
}

func TestCryptoWithdrawal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *CryptoWithdrawalRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &CryptoWithdrawalRequest{}, err: currency.ErrCurrencyCodeEmpty},
		{req: &CryptoWithdrawalRequest{Currency: currency.BTC}, err: order.ErrAmountIsInvalid},
		{req: &CryptoWithdrawalRequest{Currency: currency.BTC, Amount: 1}, err: common.ErrAddressIsEmptyOrInvalid},
	} {
		_, err := e.CryptoWithdrawal(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "CryptoWithdrawal should return the correct error")
	}

	if !mockTests {
		t.Skip("CryptoWithdrawal is not tested live to avoid withdrawing funds")
	}
	resp, err := e.CryptoWithdrawal(t.Context(), &CryptoWithdrawalRequest{
		Currency:                        currency.BTC,
		Amount:                          0.002,
		Address:                         core.BitcoinDonationAddress,
		OriginatorInfo:                  &CustomerInfo{CorporateInfo: &CorporateInfo{CompanyName: "Good company", LEI: "5493001KJTIIGC8Y1R12"}},
		BeneficiaryInfo:                 &CustomerInfo{RetailInfo: &RetailInfo{FirstName: "John", LastName: "Doe"}},
		BeneficiaryThirdParty:           new(true),
		VirtualAssetServiceProviderUUID: "48feffe5-a5d0-44fe-bf4c-734e7d721249",
	})
	require.NoError(t, err, "CryptoWithdrawal must not error")
	assert.Equal(t, &CryptoWithdrawalResponse{ID: 3}, resp, "CryptoWithdrawal should return the withdrawal")
}

func TestRippleIOUWithdrawal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *RippleIOUWithdrawalRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &RippleIOUWithdrawalRequest{}, err: currency.ErrCurrencyCodeEmpty},
		{req: &RippleIOUWithdrawalRequest{Currency: currency.USD}, err: order.ErrAmountIsInvalid},
		{req: &RippleIOUWithdrawalRequest{Currency: currency.USD, Amount: 1}, err: common.ErrAddressIsEmptyOrInvalid},
	} {
		_, err := e.RippleIOUWithdrawal(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "RippleIOUWithdrawal should return the correct error")
	}

	if !mockTests {
		t.Skip("RippleIOUWithdrawal is not tested live to avoid withdrawing funds")
	}
	resp, err := e.RippleIOUWithdrawal(t.Context(), &RippleIOUWithdrawalRequest{Currency: currency.USD, Amount: 123, Address: "rvYAfWj5gh67oV6fW32ZzP3Aw4Eubs59B"})
	require.NoError(t, err, "RippleIOUWithdrawal must not error")
	assert.Equal(t, &CryptoWithdrawalResponse{ID: 2}, resp, "RippleIOUWithdrawal should return the withdrawal")
}
