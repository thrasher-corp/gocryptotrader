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

func TestValidateContact(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		retail    *RetailInfo
		corporate *CorporateInfo
		err       error
	}{
		{name: "no details", err: errContactInfoRequired},
		{name: "retail", retail: &RetailInfo{FirstName: "John", LastName: "Doe"}},
		{name: "retail without a first name", retail: &RetailInfo{LastName: "Doe"}, err: errContactNameRequired},
		{name: "retail without a last name", retail: &RetailInfo{FirstName: "John"}, err: errContactNameRequired},
		{name: "corporate", corporate: &CorporateInfo{CompanyName: "Good company"}},
		{name: "corporate without a company name", corporate: &CorporateInfo{LEI: "5493001KJTIIGC8Y1R12"}, err: errCompanyNameRequired},
		{name: "retail and corporate", retail: &RetailInfo{FirstName: "John", LastName: "Doe"}, corporate: &CorporateInfo{CompanyName: "Good company"}},
		{name: "named retail with unnamed corporate", retail: &RetailInfo{FirstName: "John", LastName: "Doe"}, corporate: &CorporateInfo{}, err: errCompanyNameRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateContact(tc.retail, tc.corporate)
			if tc.err == nil {
				assert.NoError(t, err, "validateContact should not error")
				return
			}
			assert.ErrorIs(t, err, tc.err, "validateContact should return the correct error")
		})
	}
}

func TestValidateIdentifiedCustomer(t *testing.T) {
	t.Parallel()
	born := time.Date(1990, 1, 31, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		info *CustomerInfo
		err  error
	}{
		{name: "no details", info: &CustomerInfo{}, err: errContactInfoRequired},
		{name: "unnamed retail", info: &CustomerInfo{RetailInfo: &RetailInfo{FirstName: "John", DateOfBirth: born, PlaceOfBirth: "Sydney"}}, err: errContactNameRequired},
		{name: "retail by birth", info: &CustomerInfo{RetailInfo: &RetailInfo{FirstName: "John", LastName: "Doe", DateOfBirth: born, PlaceOfBirth: "Sydney"}}},
		{name: "retail without a place of birth", info: &CustomerInfo{RetailInfo: &RetailInfo{FirstName: "John", LastName: "Doe", DateOfBirth: born}}, err: errIdentificationRequired},
		{name: "retail without a date of birth", info: &CustomerInfo{RetailInfo: &RetailInfo{FirstName: "John", LastName: "Doe", PlaceOfBirth: "Sydney"}}, err: errIdentificationRequired},
		{name: "retail by document", info: &CustomerInfo{RetailInfo: &RetailInfo{FirstName: "John", LastName: "Doe", IDType: IDTypePassport, IDNumber: "PA1234567"}}},
		{name: "retail without a document number", info: &CustomerInfo{RetailInfo: &RetailInfo{FirstName: "John", LastName: "Doe", IDType: IDTypePassport}}, err: errIdentificationRequired},
		{name: "retail without a document type", info: &CustomerInfo{RetailInfo: &RetailInfo{FirstName: "John", LastName: "Doe", IDNumber: "PA1234567"}}, err: errIdentificationRequired},
		{name: "corporate by LEI", info: &CustomerInfo{CorporateInfo: &CorporateInfo{CompanyName: "Good company", LEI: "5493001KJTIIGC8Y1R12"}}},
		{name: "corporate by company ID", info: &CustomerInfo{CorporateInfo: &CorporateInfo{CompanyName: "Good company", CompanyID: "123"}}},
		{name: "unidentified corporate", info: &CustomerInfo{CorporateInfo: &CorporateInfo{CompanyName: "Good company"}}, err: errIdentificationRequired},
		{name: "unnamed corporate", info: &CustomerInfo{CorporateInfo: &CorporateInfo{LEI: "5493001KJTIIGC8Y1R12"}}, err: errCompanyNameRequired},
		{name: "identified retail with unidentified corporate", info: &CustomerInfo{RetailInfo: &RetailInfo{FirstName: "John", LastName: "Doe", IDType: IDTypePassport, IDNumber: "PA1234567"}, CorporateInfo: &CorporateInfo{CompanyName: "Good company"}}, err: errIdentificationRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateIdentifiedCustomer(tc.info)
			if tc.err == nil {
				assert.NoError(t, err, "validateIdentifiedCustomer should not error")
				return
			}
			assert.ErrorIs(t, err, tc.err, "validateIdentifiedCustomer should return the correct error")
		})
	}
}

func TestGetWithdrawalRequests(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *WithdrawalRequestsRequest
		err  error
	}{
		{name: "nil request", req: nil, err: common.ErrNilPointer},
		{name: "time delta above the maximum", req: &WithdrawalRequestsRequest{TimeDelta: maxWithdrawalRequestsTimeDelta + time.Second}, err: errInvalidTimeDelta},
		{name: "negative time delta", req: &WithdrawalRequestsRequest{TimeDelta: -time.Second}, err: errInvalidTimeDelta},
		{name: "time delta below a second", req: &WithdrawalRequestsRequest{TimeDelta: time.Second - time.Nanosecond}, err: errInvalidTimeDelta},
		{name: "limit above the maximum", req: &WithdrawalRequestsRequest{Limit: 1001}, err: errInvalidLimit},
		{name: "offset above the maximum", req: &WithdrawalRequestsRequest{Offset: maxResultOffset + 1}, err: errInvalidOffset},
	} {
		// Serial and in order, as the request below goes through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.GetWithdrawalRequests(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetWithdrawalRequests should return the correct error")
		})
	}
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		skipIfLiveWithoutCredentials(t)
		requests, err := e.GetWithdrawalRequests(t.Context(), &WithdrawalRequestsRequest{TimeDelta: 24 * time.Hour, Limit: 10})
		require.NoError(t, err, "GetWithdrawalRequests must not error")
		if !mockTests {
			return
		}
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
	})
}

func TestOpenBankWithdrawal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *BankWithdrawalRequest
		err  error
	}{
		{name: "nil request", req: nil, err: common.ErrNilPointer},
		{name: "without an amount", req: &BankWithdrawalRequest{}, err: order.ErrAmountIsInvalid},
		{name: "without a currency", req: &BankWithdrawalRequest{Amount: 1}, err: currency.ErrCurrencyCodeEmpty},
		{name: "without bank details", req: &BankWithdrawalRequest{Amount: 1, AccountCurrency: currency.EUR}, err: errBankDetailsRequired},
	} {
		// Serial and in order, as the request below goes through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.OpenBankWithdrawal(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "OpenBankWithdrawal should return the correct error")
		})
	}
	// Valid requests are mocked by TestWithdrawFiatFunds and TestWithdrawFiatFundsToInternationalBank
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("The request is built for the live API to refuse")
		}
		skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
		_, err := e.OpenBankWithdrawal(t.Context(), &BankWithdrawalRequest{
			Amount:          1e12,
			AccountCurrency: currency.EUR,
			Name:            "Satoshi Nakamoto",
			IBAN:            invalidIBAN,
			BIC:             "CTBAAU2S",
			Address:         "123 Fake St",
			PostalCode:      "2088",
			City:            "Tarry Town",
			Country:         "AU",
			Type:            BankWithdrawalTypeSEPA,
		})
		requireRefusedOnMerits(t, err, invalidChoice, unsupportedBankWithdrawalType)
	})
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
		name string
		req  *CryptoWithdrawalRequest
		err  error
	}{
		{name: "nil request", req: nil, err: common.ErrNilPointer},
		{name: "without a currency", req: &CryptoWithdrawalRequest{}, err: currency.ErrCurrencyCodeEmpty},
		{name: "without an amount", req: &CryptoWithdrawalRequest{Currency: currency.BTC}, err: order.ErrAmountIsInvalid},
		{name: "without an address", req: &CryptoWithdrawalRequest{Currency: currency.BTC, Amount: 1}, err: common.ErrAddressIsEmptyOrInvalid},
	} {
		// Serial and in order, as the requests below go through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.CryptoWithdrawal(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "CryptoWithdrawal should return the correct error")
		})
	}
	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("The mocked withdrawal is not sent live, to avoid withdrawing funds")
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
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("The request is built for the live API to refuse")
		}
		skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
		// More bitcoin than will ever exist, so no account holds it
		_, err := e.CryptoWithdrawal(t.Context(), &CryptoWithdrawalRequest{Currency: currency.BTC, Network: "bitcoin", Amount: 21000001, Address: core.BitcoinDonationAddress})
		requireRefusedOnMerits(t, err, invalidAddressLength, belowMinimumAmount)
	})
}

func TestRippleIOUWithdrawal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *RippleIOUWithdrawalRequest
		err  error
	}{
		{name: "nil request", req: nil, err: common.ErrNilPointer},
		{name: "without a currency", req: &RippleIOUWithdrawalRequest{}, err: currency.ErrCurrencyCodeEmpty},
		{name: "without an amount", req: &RippleIOUWithdrawalRequest{Currency: currency.USD}, err: order.ErrAmountIsInvalid},
		{name: "without an address", req: &RippleIOUWithdrawalRequest{Currency: currency.USD, Amount: 1}, err: common.ErrAddressIsEmptyOrInvalid},
	} {
		// Serial and in order, as the requests below go through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.RippleIOUWithdrawal(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "RippleIOUWithdrawal should return the correct error")
		})
	}
	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("The mocked withdrawal is not sent live, to avoid withdrawing funds")
		}
		resp, err := e.RippleIOUWithdrawal(t.Context(), &RippleIOUWithdrawalRequest{Currency: currency.USD, Amount: 123, Address: "rvYAfWj5gh67oV6fW32ZzP3Aw4Eubs59B"})
		require.NoError(t, err, "RippleIOUWithdrawal must not error")
		assert.Equal(t, &CryptoWithdrawalResponse{ID: 2}, resp, "RippleIOUWithdrawal should return the withdrawal")
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("The request is built for the live API to refuse")
		}
		skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
		_, err := e.RippleIOUWithdrawal(t.Context(), &RippleIOUWithdrawalRequest{Currency: currency.USD, Amount: 1e12, Address: "unlisted"})
		requireRefusedOnMerits(t, err, belowMinimumAmount)
	})
}
