package cryptodotcom

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetFiatAccess(t *testing.T) {
	t.Parallel()
	ex, sent := newSignedRecorder(t, "private/fiat/fiat-access", `{"accesses":[]}`)
	_, err := ex.GetFiatAccess(t.Context())
	require.NoError(t, err, "GetFiatAccess must not error")
	params := sent()
	require.Len(t, params, 1, "GetFiatAccess must send one request")
	assert.JSONEq(t, `{}`, params[0], "GetFiatAccess should send empty params")

	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/fiat/fiat-access","code":0,"result":null}`).GetFiatAccess(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFiatAccess should error for a null result")
	_, err = newUnavailableServerExchange(t).GetFiatAccess(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFiatAccess should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetFiatAccess(t.Context())
	require.NoError(t, err, "GetFiatAccess must not error")
	if mockTests {
		exp := []FiatAccessResponse{
			{Currency: currency.USD, PaymentNetwork: PaymentNetworkUSACH, DepositEnabled: true, WithdrawalEnabled: true},
			{Currency: currency.EUR, PaymentNetwork: PaymentNetworkOpenPaydSEPA, DepositEnabled: true},
		}
		assert.Equal(t, exp, result, "GetFiatAccess should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].PaymentNetwork, "GetFiatAccess should name access %d's payment network", i)
		}
	}
}

func TestGetFiatDepositInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatDepositInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFiatDepositInfo must refuse a nil request")

	ex, sent := newSignedRecorder(t, "private/fiat/fiat-deposit-info", `{"deposit_info_list":[]}`)
	_, err = ex.GetFiatDepositInfo(t.Context(), &FiatPaymentNetworksRequest{PaymentNetworks: []string{PaymentNetworkUSDCubix, PaymentNetworkUSDSWIFT}})
	require.NoError(t, err, "GetFiatDepositInfo must not error with every param")
	_, err = ex.GetFiatDepositInfo(t.Context(), &FiatPaymentNetworksRequest{})
	require.NoError(t, err, "GetFiatDepositInfo must not error without params")
	params := sent()
	require.Len(t, params, 2, "GetFiatDepositInfo must send two requests")
	assert.JSONEq(t, `{"payment_networks":"usd_cubix, usd_swift"}`, params[0], "GetFiatDepositInfo should send the payment networks as the docs' example lists them")
	assert.JSONEq(t, `{}`, params[1], "GetFiatDepositInfo should send no params when none are set")

	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/fiat/fiat-deposit-info","code":0,"result":null}`).GetFiatDepositInfo(t.Context(), &FiatPaymentNetworksRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFiatDepositInfo should error for a null result")
	_, err = newUnavailableServerExchange(t).GetFiatDepositInfo(t.Context(), &FiatPaymentNetworksRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFiatDepositInfo should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetFiatDepositInfo(t.Context(), &FiatPaymentNetworksRequest{PaymentNetworks: []string{PaymentNetworkUSDCubix, PaymentNetworkUSDSWIFT, PaymentNetworkOpenPaydSEPA, PaymentNetworkUKFPS}})
	require.NoError(t, err, "GetFiatDepositInfo must not error")
	if mockTests {
		exp := []FiatDepositInfoResponse{
			{
				PaymentNetwork: PaymentNetworkUSDCubix,
				Currency:       currency.USD,
				BankDetails: FiatDepositBankDetails{
					CubixPartnerName:   "Crypto.com",
					CubixAccountName:   "Foris Dax Inc",
					CubixAccountID:     "48e8431d-2026-41d4-a872-b1ed00db8626",
					CubixAccountNumber: "5859",
					Meta:               json.RawMessage(`null`),
				},
			},
			{
				PaymentNetwork: PaymentNetworkUSDSWIFT,
				Currency:       currency.USD,
				BankDetails: FiatDepositBankDetails{
					BankName:           "Customers Bank",
					BankAddress:        "701 Reading Avenue, West Reading, Pennsylvania 19611",
					RoutingNumber:      "031302971",
					AccountNumber:      "5415859",
					RecipientName:      "FORIS DAX LIMITED",
					RecipientAddress:   "P.O. BOX 31910, 20 GENESIS CLOSE GRAND CAYMAN",
					BankIdentifierCode: "CUESUS33",
					ReferenceCode:      "676938157158298",
					Meta:               json.RawMessage(`null`),
				},
			},
			{
				PaymentNetwork: PaymentNetworkOpenPaydSEPA,
				Currency:       currency.EUR,
				BankDetails: FiatDepositBankDetails{
					AccountHolderName:              "Example Holdings Malta Limited",
					BankName:                       "Example Bank Malta",
					BankAddress:                    "1 Example Square, Valletta VLT 1000",
					BankCountry:                    "MT",
					BankIdentifierCode:             "EXMPMTMT",
					InternationalBankAccountNumber: "MT00EXMP00000000000000012345678",
					ReferenceCode:                  "CDCX1234567",
					AccountType:                    "IBAN",
					Meta:                           json.RawMessage(`{"note":"Include the reference code"}`),
				},
			},
			{
				PaymentNetwork: PaymentNetworkUKFPS,
				Currency:       currency.GBP,
				BankDetails: FiatDepositBankDetails{
					AccountHolderName: "Example Holdings UK Limited",
					BankName:          "Example Bank UK",
					BankAddress:       "1 Example Lane, London EC1A 1AA",
					BankCountry:       "GB",
					AccountNumber:     "12345678",
					ReferenceCode:     "CDCX7654321",
					SortCode:          "00-00-00",
					AccountType:       "Current",
					Meta:              json.RawMessage(`null`),
				},
			},
		}
		assert.Equal(t, exp, result, "GetFiatDepositInfo should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].PaymentNetwork, "GetFiatDepositInfo should name deposit details %d's payment network", i)
		}
	}
}

// fiatHistoryValidationCases are the fiat history requests both fiat history methods must refuse
var fiatHistoryValidationCases = []struct {
	req *FiatHistoryRequest
	err error
}{
	{req: nil, err: common.ErrNilPointer},
	{req: &FiatHistoryRequest{}, err: errPageSizeRequired},
	{req: &FiatHistoryRequest{PageSize: 10, EndTime: time.UnixMilli(1781127400000)}, err: common.ErrDateUnset},
	{req: &FiatHistoryRequest{PageSize: 10, StartTime: time.UnixMilli(1781127400000), EndTime: time.UnixMilli(1781126400000)}, err: common.ErrStartAfterEnd},
	{req: &FiatHistoryRequest{PageSize: 10, StartTime: time.UnixMilli(-1)}, err: errTimeBeforeUnixEpoch},
	{req: &FiatHistoryRequest{PageSize: 10, StartTime: time.UnixMilli(-2), EndTime: time.UnixMilli(-1)}, err: errTimeBeforeUnixEpoch},
}

func TestGetFiatDepositHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range fiatHistoryValidationCases {
		_, err := e.GetFiatDepositHistory(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "GetFiatDepositHistory must refuse %+v", tc.req)
	}

	ex, sent := newSignedRecorder(t, "private/fiat/fiat-deposit-history", `{"page":"0","page_size":"10","total":"0","transaction_history_list":[]}`)
	_, err := ex.GetFiatDepositHistory(t.Context(), &FiatHistoryRequest{Page: 3, PageSize: 10, StartTime: time.UnixMilli(1781126400000), EndTime: time.UnixMilli(1781127400000), PaymentNetworks: []string{PaymentNetworkUSDCubix}})
	require.NoError(t, err, "GetFiatDepositHistory must not error with every param")
	_, err = ex.GetFiatDepositHistory(t.Context(), &FiatHistoryRequest{PageSize: 10})
	require.NoError(t, err, "GetFiatDepositHistory must not error with only the required params")
	params := sent()
	require.Len(t, params, 2, "GetFiatDepositHistory must send two requests")
	assert.JSONEq(t, `{"page":"3","page_size":"10","start_time":"1781126400000","end_time":"1781127400000","payment_networks":"usd_cubix"}`, params[0], "GetFiatDepositHistory should send every param")
	assert.JSONEq(t, `{"page":"0","page_size":"10"}`, params[1], "GetFiatDepositHistory should send the first page, which is required, when no page is set")

	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/fiat/fiat-deposit-history","code":0,"result":null}`).GetFiatDepositHistory(t.Context(), &FiatHistoryRequest{PageSize: 10})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFiatDepositHistory should error for a null result")
	_, err = newUnavailableServerExchange(t).GetFiatDepositHistory(t.Context(), &FiatHistoryRequest{PageSize: 10})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFiatDepositHistory should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetFiatDepositHistory(t.Context(), &FiatHistoryRequest{PageSize: 2, StartTime: time.UnixMilli(1778140000000), EndTime: time.UnixMilli(1778141000000), PaymentNetworks: []string{PaymentNetworkUSACH}})
	require.NoError(t, err, "GetFiatDepositHistory must not error")
	if mockTests {
		exp := &FiatTransactionHistoryResponse{
			PageSize: 2,
			Total:    41,
			Transactions: []FiatTransaction{
				{
					ID:                  "c2669ca7-35a2-464c-8114-2313912c6d76",
					Currency:            currency.USD,
					Amount:              preciseNumber("100.0"),
					FeeCurrency:         currency.USD,
					FeeAmount:           preciseNumber("0.5"),
					Fee:                 preciseNumber("0.5"),
					PaymentNetwork:      PaymentNetworkUSACH,
					Status:              FiatTransactionStatusCompleted,
					CreatedAt:           types.Time(time.UnixMilli(1778140550953)),
					UpdatedAt:           types.Time(time.UnixMilli(1778140573431)),
					CompletedAt:         types.Time(time.UnixMilli(1778140573431)),
					VendorTransactionID: "N89940DKZD",
					Sender:              &FiatTransactionParty{AccountIdentifierValue: "****6789"},
				},
				{
					ID:                  "f5527db9-c0ca-4d82-b3f8-4470c3398384",
					Currency:            currency.USD,
					Amount:              preciseNumber("100.0"),
					FeeCurrency:         currency.USD,
					FeeAmount:           preciseNumber("0.0"),
					PaymentNetwork:      PaymentNetworkUSACH,
					Status:              FiatTransactionStatusCompleted,
					CreatedAt:           types.Time(time.UnixMilli(1778140412428)),
					UpdatedAt:           types.Time(time.UnixMilli(1778140429199)),
					CompletedAt:         types.Time(time.UnixMilli(1778140429199)),
					VendorTransactionID: "F6KSNMQV0B",
				},
			},
		}
		assert.Equal(t, exp, result, "GetFiatDepositHistory should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Transactions), 2, "GetFiatDepositHistory should return at most a page of deposits")
	}
}

func TestGetFiatWithdrawHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range fiatHistoryValidationCases {
		_, err := e.GetFiatWithdrawHistory(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "GetFiatWithdrawHistory must refuse %+v", tc.req)
	}

	ex, sent := newSignedRecorder(t, "private/fiat/fiat-withdraw-history", `{"page":"0","page_size":"10","total":"0","transaction_history_list":[]}`)
	_, err := ex.GetFiatWithdrawHistory(t.Context(), &FiatHistoryRequest{Page: 3, PageSize: 10, StartTime: time.UnixMilli(1781126400000), EndTime: time.UnixMilli(1781127400000), PaymentNetworks: []string{PaymentNetworkUSDCubix, PaymentNetworkUSDSWIFT}})
	require.NoError(t, err, "GetFiatWithdrawHistory must not error with every param")
	_, err = ex.GetFiatWithdrawHistory(t.Context(), &FiatHistoryRequest{PageSize: 10})
	require.NoError(t, err, "GetFiatWithdrawHistory must not error with only the required params")
	params := sent()
	require.Len(t, params, 2, "GetFiatWithdrawHistory must send two requests")
	assert.JSONEq(t, `{"page":"3","page_size":"10","start_time":"1781126400000","end_time":"1781127400000","payment_networks":"usd_cubix, usd_swift"}`, params[0], "GetFiatWithdrawHistory should send every param")
	assert.JSONEq(t, `{"page":"0","page_size":"10"}`, params[1], "GetFiatWithdrawHistory should send the first page, which is required, when no page is set")

	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/fiat/fiat-withdraw-history","code":0,"result":null}`).GetFiatWithdrawHistory(t.Context(), &FiatHistoryRequest{PageSize: 10})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFiatWithdrawHistory should error for a null result")
	_, err = newUnavailableServerExchange(t).GetFiatWithdrawHistory(t.Context(), &FiatHistoryRequest{PageSize: 10})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFiatWithdrawHistory should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetFiatWithdrawHistory(t.Context(), &FiatHistoryRequest{PageSize: 2, StartTime: time.UnixMilli(1778200000000), EndTime: time.UnixMilli(1778600000000)})
	require.NoError(t, err, "GetFiatWithdrawHistory must not error")
	if mockTests {
		exp := &FiatTransactionHistoryResponse{
			PageSize: 2,
			Total:    132,
			Transactions: []FiatTransaction{
				{
					ID:                 "a6a41d92-1cdc-4dd3-b845-85fa203405a2",
					AccountID:          "6871391474021396480",
					Currency:           currency.EUR,
					Amount:             preciseNumber("10.0"),
					AmountInUSD:        preciseNumber("10.87"),
					FeeCurrency:        currency.EUR,
					FeeAmount:          preciseNumber("1.0"),
					FeeAmountInUSD:     preciseNumber("1.09"),
					PaymentNetwork:     PaymentNetworkOpenPaydSEPA,
					Status:             FiatTransactionStatusPending,
					CreatedAt:          types.Time(time.UnixMilli(1778553420579)),
					UpdatedAt:          types.Time(time.UnixMilli(1778553420643)),
					ClientWithdrawalID: "c1d2e3f4-a5b6-7890-cdef-123456789abc",
					Beneficiary:        &FiatTransactionParty{AccountIdentifierValue: "****6789"},
				},
				{
					ID:                  "fb372838-be75-4699-92c8-0db0efee6506",
					Currency:            currency.USD,
					Amount:              preciseNumber("100.0"),
					FeeCurrency:         currency.USD,
					FeeAmount:           preciseNumber("1.0"),
					PaymentNetwork:      PaymentNetworkUSACH,
					Status:              FiatTransactionStatusCompleted,
					CreatedAt:           types.Time(time.UnixMilli(1778212310417)),
					UpdatedAt:           types.Time(time.UnixMilli(1778212506975)),
					CompletedAt:         types.Time(time.UnixMilli(1778212506975)),
					VendorTransactionID: "db0fee89-6ab7-4431-b715-798655891555",
				},
			},
		}
		assert.Equal(t, exp, result, "GetFiatWithdrawHistory should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Transactions), 2, "GetFiatWithdrawHistory should return at most a page of withdrawals")
	}
}

func TestCreateFiatWithdrawal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *FiatWithdrawalRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &FiatWithdrawalRequest{Amount: 1, Currency: currency.USD, PaymentNetwork: PaymentNetworkUSDSWIFT}, err: errBankAccountIDRequired},
		{req: &FiatWithdrawalRequest{BankAccountID: "b", Currency: currency.USD, PaymentNetwork: PaymentNetworkUSDSWIFT}, err: errInvalidAmount},
		{req: &FiatWithdrawalRequest{BankAccountID: "b", Amount: -1, Currency: currency.USD, PaymentNetwork: PaymentNetworkUSDSWIFT}, err: errInvalidAmount},
		{req: &FiatWithdrawalRequest{BankAccountID: "b", Amount: 1, PaymentNetwork: PaymentNetworkUSDSWIFT}, err: currency.ErrCurrencyCodeEmpty},
		{req: &FiatWithdrawalRequest{BankAccountID: "b", Amount: 1, Currency: currency.USD}, err: errPaymentNetworkRequired},
	} {
		_, err := e.CreateFiatWithdrawal(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "CreateFiatWithdrawal must refuse %+v", tc.req)
	}

	ex, sent := newSignedRecorder(t, "private/fiat/fiat-create-withdraw", `{"id":"1"}`)
	_, err := ex.CreateFiatWithdrawal(t.Context(), &FiatWithdrawalRequest{
		ClientWithdrawalID: "c1",
		BankAccountID:      "b1",
		Amount:             105,
		Currency:           currency.NewCode("aed"),
		PaymentNetwork:     PaymentNetworkAEDIPI,
		IntermediateBank: &FiatIntermediateBank{
			BankIdentifierType:  "SWIFT",
			BankIdentifierValue: "CHASUS33",
			BankName:            "JPMorgan Chase Bank",
			AddressLine1:        "270 Park Avenue",
			AddressLine2:        "New York, NY 10017",
		},
		BeneficiaryID:          "be1",
		AuthorisationID:        "au1",
		BankIdentifierValue:    "bi1",
		AccountIdentifierValue: "ai1",
	})
	require.NoError(t, err, "CreateFiatWithdrawal must not error with every param")
	_, err = ex.CreateFiatWithdrawal(t.Context(), &FiatWithdrawalRequest{BankAccountID: "b2", Amount: 100, Currency: currency.EUR, PaymentNetwork: PaymentNetworkOpenPaydSEPA})
	require.NoError(t, err, "CreateFiatWithdrawal must not error with only the required params")
	params := sent()
	require.Len(t, params, 2, "CreateFiatWithdrawal must send two requests")
	assert.JSONEq(t, `{
		"client_wid": "c1",
		"account_id": "b1",
		"amount": "105",
		"currency": "AED",
		"payment_network": "aed_ipi",
		"intermediate_bank": {
			"bank_identifier_type": "SWIFT",
			"bank_identifier_value": "CHASUS33",
			"bank_name": "JPMorgan Chase Bank",
			"address_1": "270 Park Avenue",
			"address_2": "New York, NY 10017"
		},
		"beneficiary_id": "be1",
		"authorization_id": "au1",
		"bank_identifier_value": "bi1",
		"account_identifier_value": "ai1"
	}`, params[0], "CreateFiatWithdrawal should send every param, the currency in upper case")
	assert.JSONEq(t, `{"account_id":"b2","amount":"100","currency":"EUR","payment_network":"openpayd_exchange_sepa"}`, params[1], "CreateFiatWithdrawal should send only the required params when the others are unset")

	req := &FiatWithdrawalRequest{
		ClientWithdrawalID: "c1d2e3f4-a5b6-7890-cdef-123456789abc",
		BankAccountID:      "7d1e5a3b-2c4f-4e8a-9b6d-0f1a2b3c4d5e",
		Amount:             500,
		Currency:           currency.USD,
		PaymentNetwork:     PaymentNetworkUSDSWIFT,
		IntermediateBank: &FiatIntermediateBank{
			BankIdentifierType:  "SWIFT",
			BankIdentifierValue: "CHASUS33",
			BankName:            "JPMorgan Chase Bank",
			AddressLine1:        "270 Park Avenue",
			AddressLine2:        "New York, NY 10017",
		},
		BeneficiaryID:   "b5d7c1e2-3f4a-4b6c-8d9e-0a1b2c3d4e5f",
		AuthorisationID: "a9e8f7d6-c5b4-4a39-8281-7f6e5d4c3b2a",
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/fiat/fiat-create-withdraw","code":0,"result":null}`).CreateFiatWithdrawal(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateFiatWithdrawal should error for a null result")
	_, err = newUnavailableServerExchange(t).CreateFiatWithdrawal(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateFiatWithdrawal should return the request's failure")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		// Live, the least Crypto.com allows goes to the first linked bank account, on its first payment network
		bankAccounts, err := e.GetFiatBankAccounts(t.Context(), &FiatPaymentNetworksRequest{})
		require.NoError(t, err, "GetFiatBankAccounts must not error")
		if len(bankAccounts) == 0 || len(bankAccounts[0].SupportedPaymentNetworks) == 0 {
			t.Skip("no bank account is linked for fiat withdrawals")
		}
		quota, err := e.GetFiatTransactionQuota(t.Context(), bankAccounts[0].SupportedPaymentNetworks[0])
		require.NoError(t, err, "GetFiatTransactionQuota must not error")
		require.NotNil(t, quota.MinimumWithdrawalAmount.Amount, "the payment network must have a minimum withdrawal amount")
		req = &FiatWithdrawalRequest{BankAccountID: bankAccounts[0].ID, Amount: *quota.MinimumWithdrawalAmount.Amount, Currency: bankAccounts[0].Currency, PaymentNetwork: bankAccounts[0].SupportedPaymentNetworks[0]}
	}
	result, err := e.CreateFiatWithdrawal(t.Context(), req)
	require.NoError(t, err, "CreateFiatWithdrawal must not error")
	if mockTests {
		exp := &FiatWithdrawalResponse{
			ID:                 "27f3b781-e8e4-4ae9-871e-0edd7f9deba9",
			ClientWithdrawalID: "c1d2e3f4-a5b6-7890-cdef-123456789abc",
			BankAccountID:      "7d1e5a3b-2c4f-4e8a-9b6d-0f1a2b3c4d5e",
			Status:             FiatTransactionStatusDebitProcessing,
			PaymentNetwork:     PaymentNetworkUSDSWIFT,
			Currency:           currency.USD,
			Amount:             preciseNumber("500.0"),
			AmountInUSD:        preciseNumber("500.0"),
			FeeCurrency:        currency.USD,
			FeeAmount:          preciseNumber("45.0"),
			BeneficiaryID:      "b5d7c1e2-3f4a-4b6c-8d9e-0a1b2c3d4e5f",
			AuthorisationID:    "a9e8f7d6-c5b4-4a39-8281-7f6e5d4c3b2a",
		}
		assert.Equal(t, exp, result, "CreateFiatWithdrawal should decode every field")
	} else {
		assert.NotEmpty(t, result.ID, "CreateFiatWithdrawal should return the withdrawal's ID")
	}
}

func TestGetFiatTransactionQuota(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatTransactionQuota(t.Context(), "")
	require.ErrorIs(t, err, errPaymentNetworkRequired, "GetFiatTransactionQuota must refuse an empty payment network")

	ex, sent := newSignedRecorder(t, "private/fiat/fiat-transaction-quota", `{"details":{}}`)
	_, err = ex.GetFiatTransactionQuota(t.Context(), PaymentNetworkUSDSWIFT)
	require.NoError(t, err, "GetFiatTransactionQuota must not error")
	params := sent()
	require.Len(t, params, 1, "GetFiatTransactionQuota must send one request")
	assert.JSONEq(t, `{"payment_network":"usd_swift"}`, params[0], "GetFiatTransactionQuota should send the payment network")

	for _, result := range []string{`null`, `{"details":null}`} {
		_, err = newResponseServerExchange(t, `{"id":1,"method":"private/fiat/fiat-transaction-quota","code":0,"result":`+result+`}`).GetFiatTransactionQuota(t.Context(), PaymentNetworkUSDSWIFT)
		assert.ErrorIsf(t, err, common.ErrNoResponse, "GetFiatTransactionQuota should error for the result %s", result)
	}
	_, err = newUnavailableServerExchange(t).GetFiatTransactionQuota(t.Context(), PaymentNetworkUSDSWIFT)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFiatTransactionQuota should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetFiatTransactionQuota(t.Context(), PaymentNetworkOpenPaydSEPA)
	require.NoError(t, err, "GetFiatTransactionQuota must not error")
	if mockTests {
		exp := &FiatTransactionQuotaResponse{
			Currency:                                currency.EUR,
			PaymentNetwork:                          PaymentNetworkOpenPaydSEPA,
			NetworkName:                             "SEPA",
			MonthlyQuotaInUSD:                       FiatCurrencyAmount{Currency: currency.USD},
			DailyQuotaInUSD:                         FiatCurrencyAmount{Currency: currency.USD, Amount: new(types.Number(5000000))},
			UsedMonthlyQuotaInUSD:                   FiatCurrencyAmount{Currency: currency.USD, Amount: new(types.Number(6200))},
			UsedDailyQuotaInUSD:                     FiatCurrencyAmount{Currency: currency.USD, Amount: new(types.Number(1500))},
			RemainingMonthlyQuotaInUSD:              FiatCurrencyAmount{Currency: currency.USD},
			RemainingDailyQuotaInUSD:                FiatCurrencyAmount{Currency: currency.USD, Amount: new(types.Number(4998500))},
			MinimumWithdrawalAmountInUSD:            FiatCurrencyAmount{Currency: currency.USD, Amount: new(types.Number(500))},
			MonthlyQuota:                            FiatCurrencyAmount{Currency: currency.EUR},
			DailyQuota:                              FiatCurrencyAmount{Currency: currency.EUR, Amount: new(types.Number(4600000))},
			UsedMonthlyQuota:                        FiatCurrencyAmount{Currency: currency.EUR, Amount: new(types.Number(5704))},
			UsedDailyQuota:                          FiatCurrencyAmount{Currency: currency.EUR, Amount: new(types.Number(1380))},
			RemainingMonthlyQuota:                   FiatCurrencyAmount{Currency: currency.EUR},
			RemainingDailyQuota:                     FiatCurrencyAmount{Currency: currency.EUR, Amount: new(types.Number(4598620))},
			MinimumWithdrawalAmount:                 FiatCurrencyAmount{Currency: currency.EUR, Amount: new(types.Number(460))},
			CurrencyDailyQuota:                      FiatCurrencyAmount{Currency: currency.EUR, Amount: new(types.Number(9200000))},
			CurrencyMonthlyQuota:                    FiatCurrencyAmount{Currency: currency.EUR},
			CurrencyUsedDailyQuota:                  FiatCurrencyAmount{Currency: currency.EUR, Amount: new(types.Number(2300))},
			CurrencyUsedMonthlyQuota:                FiatCurrencyAmount{Currency: currency.EUR, Amount: new(types.Number(9016))},
			CurrencyRemainingDailyQuota:             FiatCurrencyAmount{Currency: currency.EUR, Amount: new(types.Number(9197700))},
			CurrencyRemainingMonthlyQuota:           FiatCurrencyAmount{Currency: currency.EUR},
			TransactionsPerDay:                      new(types.Number(10)),
			TransactionsPerMonth:                    new(types.Number(100)),
			TransactionsDailyCount:                  2,
			TransactionsMonthlyCount:                9,
			RemainingTransactionsDailyCount:         new(types.Number(8)),
			RemainingTransactionsMonthlyCount:       new(types.Number(91)),
			CurrencyTransactionsPerDay:              new(types.Number(20)),
			CurrencyTransactionsDailyCount:          3,
			CurrencyTransactionsMonthlyCount:        11,
			CurrencyRemainingTransactionsDailyCount: new(types.Number(17)),
		}
		assert.Equal(t, exp, result, "GetFiatTransactionQuota should decode every field, unlimited amounts and counts as nil")
	} else {
		assert.Equal(t, PaymentNetworkOpenPaydSEPA, result.PaymentNetwork, "GetFiatTransactionQuota should return the payment network's quota")
	}
}

func TestGetFiatTransactionLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatTransactionLimit(t.Context(), "")
	require.ErrorIs(t, err, errPaymentNetworkRequired, "GetFiatTransactionLimit must refuse an empty payment network")

	ex, sent := newSignedRecorder(t, "private/fiat/fiat-transaction-limit", `{"deposit":{},"payment":{}}`)
	_, err = ex.GetFiatTransactionLimit(t.Context(), PaymentNetworkUSDSWIFT)
	require.NoError(t, err, "GetFiatTransactionLimit must not error")
	params := sent()
	require.Len(t, params, 1, "GetFiatTransactionLimit must send one request")
	assert.JSONEq(t, `{"payment_network":"usd_swift"}`, params[0], "GetFiatTransactionLimit should send the payment network")

	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/fiat/fiat-transaction-limit","code":0,"result":null}`).GetFiatTransactionLimit(t.Context(), PaymentNetworkUSDSWIFT)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFiatTransactionLimit should error for a null result")
	_, err = newUnavailableServerExchange(t).GetFiatTransactionLimit(t.Context(), PaymentNetworkUSDSWIFT)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFiatTransactionLimit should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetFiatTransactionLimit(t.Context(), PaymentNetworkUSDCubix)
	require.NoError(t, err, "GetFiatTransactionLimit must not error")
	if mockTests {
		exp := &FiatTransactionLimitResponse{
			Deposit: FiatDepositLimits{
				MinimumDepositAmount:                   FiatCurrencyAmount{Currency: currency.USD, Amount: new(types.Number(500))},
				DailyMaximumDepositAmount:              FiatCurrencyAmount{Currency: currency.USD, Amount: new(types.Number(1000000))},
				MonthlyMaximumDepositAmount:            FiatCurrencyAmount{Currency: currency.USD},
				CurrencyDailyMaximumDepositAmount:      FiatCurrencyAmount{Currency: currency.USD, Amount: new(types.Number(2000000))},
				CurrencyMonthlyMaximumDepositAmount:    FiatCurrencyAmount{Currency: currency.USD},
				DailyQuota:                             5,
				MonthlyQuota:                           60,
				DailyTransactionCount:                  1,
				MonthlyTransactionCount:                4,
				CurrencyDailyTransactionCount:          2,
				CurrencyMonthlyTransactionCount:        6,
				FeeAmount:                              FiatCurrencyAmount{Currency: currency.USD, Amount: new(types.Number(0))},
				DailyMaximumTransactionCount:           10,
				MonthlyMaximumTransactionCount:         100,
				CurrencyDailyMaximumTransactionCount:   20,
				CurrencyMonthlyMaximumTransactionCount: 200,
			},
			Payment: FiatPaymentLimits{
				Name:                                   "CUBIX",
				FullName:                               "CUBIX",
				ReviewTimeDescription:                  "1-2 working days",
				ReviewTime:                             FiatDurationRange{Minimum: 1, Maximum: 2, Unit: "working days"},
				BankTransferTimeDescription:            "1-4 working days",
				BankTransferTime:                       FiatDurationRange{Minimum: 1, Maximum: 4, Unit: "working days"},
				MinimumPaymentAmount:                   FiatCurrencyAmount{Currency: currency.USD, Amount: new(types.Number(500))},
				DailyMaximumPaymentAmount:              FiatCurrencyAmount{Currency: currency.USD, Amount: new(types.Number(5000000))},
				MonthlyMaximumPaymentAmount:            FiatCurrencyAmount{Currency: currency.USD},
				AutoApproveMaximumPaymentAmount:        FiatCurrencyAmount{Currency: currency.USD, Amount: new(types.Number(250000))},
				CurrencyDailyMaximumPaymentAmount:      FiatCurrencyAmount{Currency: currency.USD, Amount: new(types.Number(10000000))},
				CurrencyMonthlyMaximumPaymentAmount:    FiatCurrencyAmount{Currency: currency.USD},
				FeeAmount:                              FiatCurrencyAmount{Currency: currency.USD, Amount: new(types.Number(0))},
				RefundFeeAmount:                        FiatCurrencyAmount{Currency: currency.USD, Amount: new(types.Number(25))},
				DailyTransactionCount:                  3,
				MonthlyTransactionCount:                8,
				CurrencyDailyTransactionCount:          7,
				CurrencyMonthlyTransactionCount:        12,
				DailyMaximumTransactionCount:           15,
				MonthlyMaximumTransactionCount:         150,
				CurrencyDailyMaximumTransactionCount:   30,
				CurrencyMonthlyMaximumTransactionCount: 300,
			},
		}
		assert.Equal(t, exp, result, "GetFiatTransactionLimit should decode every field, unlimited amounts as nil")
	} else {
		assert.NotEmpty(t, result.Payment.Name, "GetFiatTransactionLimit should name the payment network")
	}
}

func TestCreateFiatBankAccount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *FiatBankAccountRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &FiatBankAccountRequest{PlaidPublicToken: "p", Currency: currency.USD}, err: errPlaidAccountIDRequired},
		{req: &FiatBankAccountRequest{PlaidAccountID: "a", Currency: currency.USD}, err: errPlaidPublicTokenRequired},
		{req: &FiatBankAccountRequest{PlaidAccountID: "a", PlaidPublicToken: "p"}, err: currency.ErrCurrencyCodeEmpty},
	} {
		_, err := e.CreateFiatBankAccount(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "CreateFiatBankAccount must refuse %+v", tc.req)
	}

	ex, sent := newSignedRecorder(t, "private/fiat/fiat-create-bank-account", `{"bank_accounts_list":[]}`)
	_, err := ex.CreateFiatBankAccount(t.Context(), &FiatBankAccountRequest{PlaidAccountID: "a1", PlaidPublicToken: "public-sandbox-1", Currency: currency.NewCode("usd"), BankAccountID: "b1"})
	require.NoError(t, err, "CreateFiatBankAccount must not error with every param")
	_, err = ex.CreateFiatBankAccount(t.Context(), &FiatBankAccountRequest{PlaidAccountID: "a2", PlaidPublicToken: "public-sandbox-2", Currency: currency.USD})
	require.NoError(t, err, "CreateFiatBankAccount must not error with only the required params")
	params := sent()
	require.Len(t, params, 2, "CreateFiatBankAccount must send two requests")
	assert.JSONEq(t, `{"plaid_account_id":"a1","plaid_public_token":"public-sandbox-1","currency":"USD","bank_account_id":"b1"}`, params[0], "CreateFiatBankAccount should send every param, the currency in upper case")
	assert.JSONEq(t, `{"plaid_account_id":"a2","plaid_public_token":"public-sandbox-2","currency":"USD"}`, params[1], "CreateFiatBankAccount should send only the required params when the others are unset")

	req := &FiatBankAccountRequest{PlaidAccountID: "vzeNDwK7KQIm4yEog683uElbp9GRLEFXGK98D", PlaidPublicToken: "public-sandbox-0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d", Currency: currency.USD}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/fiat/fiat-create-bank-account","code":0,"result":null}`).CreateFiatBankAccount(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateFiatBankAccount should error for a null result")
	_, err = newUnavailableServerExchange(t).CreateFiatBankAccount(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateFiatBankAccount should return the request's failure")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("a Plaid public token only comes from a user completing Plaid Link")
	}
	result, err := e.CreateFiatBankAccount(t.Context(), req)
	require.NoError(t, err, "CreateFiatBankAccount must not error")
	exp := []FiatBankAccountResponse{{
		ID:                       "5b8e2f1a-9c3d-4e7f-a6b5-1c2d3e4f5a6b",
		Status:                   FiatBankAccountStatusCompleted,
		BankName:                 "Chase",
		AccountIdentifierValue:   "****6789",
		AccountHolderName:        "John Doe",
		Currency:                 currency.USD,
		SupportedPaymentNetworks: []string{PaymentNetworkUSACH},
		CreatedAt:                types.Time(time.UnixMilli(1773372503821)),
		UpdatedAt:                types.Time(time.UnixMilli(1773372503821)),
	}}
	assert.Equal(t, exp, result, "CreateFiatBankAccount should decode every field")
}

func TestGetFiatBankAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatBankAccounts(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFiatBankAccounts must refuse a nil request")

	ex, sent := newSignedRecorder(t, "private/fiat/fiat-get-bank-accounts", `{"bank_accounts_list":[]}`)
	_, err = ex.GetFiatBankAccounts(t.Context(), &FiatPaymentNetworksRequest{PaymentNetworks: []string{PaymentNetworkUSDCubix, PaymentNetworkUSDFedwire}})
	require.NoError(t, err, "GetFiatBankAccounts must not error with every param")
	_, err = ex.GetFiatBankAccounts(t.Context(), &FiatPaymentNetworksRequest{})
	require.NoError(t, err, "GetFiatBankAccounts must not error without params")
	params := sent()
	require.Len(t, params, 2, "GetFiatBankAccounts must send two requests")
	assert.JSONEq(t, `{"payment_networks":"usd_cubix, usd_fedwire"}`, params[0], "GetFiatBankAccounts should send the payment networks as the docs' example lists them")
	assert.JSONEq(t, `{}`, params[1], "GetFiatBankAccounts should send no params when none are set")

	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/fiat/fiat-get-bank-accounts","code":0,"result":null}`).GetFiatBankAccounts(t.Context(), &FiatPaymentNetworksRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFiatBankAccounts should error for a null result")
	_, err = newUnavailableServerExchange(t).GetFiatBankAccounts(t.Context(), &FiatPaymentNetworksRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFiatBankAccounts should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetFiatBankAccounts(t.Context(), &FiatPaymentNetworksRequest{PaymentNetworks: []string{PaymentNetworkUSDCubix}})
	require.NoError(t, err, "GetFiatBankAccounts must not error")
	if mockTests {
		exp := []FiatBankAccountResponse{{
			ID:                            "4c662477-a16d-4c99-a27b-a492b592ed62",
			AccountID:                     "adb80cff-f420-469c-b439-4d90272bf1a1",
			UserUUID:                      "f0857b97-0d47-5897-8b1f-e358725cf49c",
			Status:                        FiatBankAccountStatusCompleted,
			BankName:                      "Customers Bank",
			BankCountry:                   "US",
			BankIdentifierType:            "cubix_account_number_last_4_digits",
			BankIdentifierValue:           "****2565",
			AccountIdentifierType:         "cubix_account_id",
			AccountIdentifierValue:        "****cc63",
			AccountHolderName:             "Smoke Test Co",
			Currency:                      currency.USD,
			VerifiedBy:                    "Cubix",
			Reason:                        "reason",
			SupportedPaymentNetworks:      []string{PaymentNetworkUSDCubix},
			PaymentNetworkIdentifierValue: "dbbb3eda-960f-4406-ae3c-b250010e7bef",
			BankDetails: &FiatBankAccountDetails{
				InstitutionalStreetAddress: "40 General Warren Boulevard, Malvern, Pennsylvania 19355",
				BankStreetAddress:          "701 Reading Avenue, West Reading, Pennsylvania 19611",
				AccountHolderAddress:       "1 Example Street, Sydney NSW 2000",
				BankAccountAddress:         json.RawMessage(`null`),
				IntermediateBank:           json.RawMessage(`null`),
			},
			CreatedAt: types.Time(time.UnixMilli(1751126400000)),
			UpdatedAt: types.Time(time.UnixMilli(1751212800000)),
		}}
		assert.Equal(t, exp, result, "GetFiatBankAccounts should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].ID, "GetFiatBankAccounts should return bank account %d's ID", i)
		}
	}
}
