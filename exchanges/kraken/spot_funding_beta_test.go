package kraken

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	fundingTestAccountID   = "WX6V-JUKW-KKPB-QE36"
	fundingTestEVMGroup    = "f95acdb7-48fb-4441-b5b4-843d3bf60e61"
	fundingTestSolanaGroup = "6c2e9f1a-8d4b-4f7e-a3c5-1b9d0e7f2a64"
	fundingTestFeeToken    = "AAAAAAAAAAHG33Wc1eES6QpeGgMok_gUnlZC7Y5niezI2,MBriLCyg8oqCX7SH1bSqY6SfmC2ZLf_SgTxNZj4Cd1RLPaTyZE"
)

// newFundingRequestServer returns an exchange whose server checks a signed Funding (Beta) request's method, path,
// query and JSON body, then replies with response. The VCR server cannot check these requests, as it matches neither
// the query of a request carrying a JSON body nor a nested body
func newFundingRequestServer(t *testing.T, method, path, query, body, response string) *Exchange {
	t.Helper()
	return newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		sent := checkSpotSignedRequest(t, r, func([]byte) string { return r.Header.Get("API-Nonce") })
		assert.Equal(t, method, r.Method, "request method should match")
		assert.Equal(t, path, r.URL.Path, "request path should match")
		assert.Equal(t, query, r.URL.RawQuery, "request query should match")
		assert.JSONEq(t, body, string(sent), "request body should match")
		_, _ = w.Write([]byte(response))
	})
}

func fundingTestAmount(class string, name currency.Code, amount types.Number) FundingAmount {
	return FundingAmount{Asset: FundingAsset{Class: class, Name: name}, Amount: amount}
}

func fundingTestUSDC(amount types.Number) FundingAmount {
	return fundingTestAmount("currency", currency.USDC, amount)
}

func TestCalculateFundingFees(t *testing.T) {
	t.Parallel()
	_, err := e.CalculateFundingFees(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CalculateFundingFees must reject a nil request")
	_, err = e.CalculateFundingFees(t.Context(), &FundingFeesRequest{Amount: 5})
	require.ErrorIs(t, err, errFundingMethodIDEmpty, "CalculateFundingFees must reject an empty method ID")
	_, err = e.CalculateFundingFees(t.Context(), &FundingFeesRequest{MethodID: "d4ec4d52-b159-428e-ba64-f45455a978a1"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CalculateFundingFees must reject an empty amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	aapl := currency.NewCode("AAPLx")
	for _, tc := range []struct {
		name string
		req  *FundingFeesRequest
		exp  *FundingFeesResponse
	}{
		{
			name: "withdrawal quote",
			req: &FundingFeesRequest{
				MethodID:           "d4ec4d52-b159-428e-ba64-f45455a978a1",
				Amount:             5,
				FeeIncluded:        true,
				WithdrawalFeeToken: fundingTestFeeToken,
				AccountID:          fundingTestAccountID,
			},
			exp: &FundingFeesResponse{
				Fee:                fundingTestUSDC(1.025),
				GrossAmount:        fundingTestUSDC(5),
				NetAmount:          fundingTestUSDC(3.975),
				FeeDetails:         FundingFeeDetails{BaseFee: fundingTestUSDC(1), FeePercentage: 0.5},
				WithdrawalFeeToken: "AAAAAAAAAAHG33Wc1eES6QpeGgMok_gUnlZC7Y5niezI2,NCsjMDzh9prDY8TI2cTrZ7TgnD3aMg_ThUyOak5De2SMQbUzaFEUmz2Qgc9YK8X1",
			},
		},
		{
			name: "tokenised deposit quote",
			req:  &FundingFeesRequest{MethodID: "0f6e7d4c-3b2a-4198-8a7b-6c5d4e3f2a10", Amount: 2.5, RebaseMultiplier: "base"},
			exp: &FundingFeesResponse{
				Fee:         fundingTestAmount("tokenized_asset", aapl, 0.0125),
				GrossAmount: fundingTestAmount("tokenized_asset", aapl, 2.5125),
				NetAmount:   fundingTestAmount("tokenized_asset", aapl, 2.5),
				FeeDetails:  FundingFeeDetails{BaseFee: fundingTestAmount("tokenized_asset", aapl, 0.01), FeePercentage: 0.1},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.CalculateFundingFees(t.Context(), tc.req)
			require.NoError(t, err, "CalculateFundingFees must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CalculateFundingFees should decode every field")
				return
			}
			assert.Positive(t, result.GrossAmount.Amount.Float64(), "CalculateFundingFees should return a gross amount")
		})
	}
}

func TestClaimFundingDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.ClaimFundingDepositAddress(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ClaimFundingDepositAddress must reject a nil request")
	_, err = e.ClaimFundingDepositAddress(t.Context(), &FundingDepositAddressRequest{AccountID: fundingTestAccountID})
	require.ErrorIs(t, err, errFundingMethodIDEmpty, "ClaimFundingDepositAddress must reject an empty method ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name, methodID, accountID, query, response string
		exp                                        *FundingDepositAddressResponse
	}{
		{
			name:      "tagged crypto address",
			methodID:  "e1b7c3d9-5f2a-4c8e-a6b4-7d0f9e2c5a38",
			accountID: fundingTestAccountID,
			query:     "account_id=" + fundingTestAccountID,
			response:  `{"address_details":{"crypto":{"address":"rPEPPER7kfTD9w2To4CQk6UCfuHM9c6GDY","tag":"2059931772"}}}`,
			exp: &FundingDepositAddressResponse{AddressDetails: FundingDepositAddressDetails{
				Crypto: &FundingDepositCryptoAddress{Address: "rPEPPER7kfTD9w2To4CQk6UCfuHM9c6GDY", Tag: "2059931772"},
			}},
		},
		{
			name:     "crypto address with a memo",
			methodID: "f6a2d8c4-9e3b-4d1f-8c7a-2b5e0f9d3a16",
			response: `{"address_details":{"crypto":{"address":"GA5XIGA5C7QTPTWXQHY6MCJRMTRZDOSHR6EFIBNDQTCQHG262N4GGKTM","memo":"7731005"}}}`,
			exp: &FundingDepositAddressResponse{AddressDetails: FundingDepositAddressDetails{
				Crypto: &FundingDepositCryptoAddress{Address: "GA5XIGA5C7QTPTWXQHY6MCJRMTRZDOSHR6EFIBNDQTCQHG262N4GGKTM", Memo: "7731005"},
			}},
		},
		{
			name:     "fiat bank details",
			methodID: "5e0b1c7a-2f4d-4e8b-9c6a-3d1f0e2b4a57",
			response: `{"address_details":{"fiat":{"account":"87654321","account_type":"savings","address":"2 Sample Square, Dublin",` +
				`"bank":"Example Deposit Bank","bank_code":"EXDB","bic":"EXDBIE2D","branch":"Grand Canal","bank_address":"5 Bank Quay, Dublin 2",` +
				`"bsb":"082-001","iban":"IE29AIBK93115212345678","memo":"KRAKEN-REF-8812","name_on_account":"Example Exchange Ltd",` +
				`"routing":"011000015","sort":"93-11-52","swift":"EXDBIE2DXXX","tag":"4471","transit":"00212"}}}`,
			exp: &FundingDepositAddressResponse{AddressDetails: FundingDepositAddressDetails{Fiat: fundingTestDepositFiatAddress()}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := e
			if mockTests {
				ex = newFundingRequestServer(t, http.MethodPut, "/funding/v1/deposit/address", tc.query, `{"method_id":"`+tc.methodID+`"}`, tc.response)
			}
			result, err := ex.ClaimFundingDepositAddress(t.Context(), &FundingDepositAddressRequest{MethodID: tc.methodID, AccountID: tc.accountID})
			require.NoError(t, err, "ClaimFundingDepositAddress must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ClaimFundingDepositAddress should decode every field")
				return
			}
			assert.True(t, result.AddressDetails.Crypto != nil || result.AddressDetails.Fiat != nil, "ClaimFundingDepositAddress should return an address")
		})
	}
}

func fundingTestDepositFiatAddress() *FundingDepositFiatAddress {
	return &FundingDepositFiatAddress{
		AccountNumber: "87654321",
		AccountType:   "savings",
		Address:       "2 Sample Square, Dublin",
		BankName:      "Example Deposit Bank",
		BankCode:      "EXDB",
		BIC:           "EXDBIE2D",
		Branch:        "Grand Canal",
		BankAddress:   "5 Bank Quay, Dublin 2",
		BSB:           "082-001",
		IBAN:          "IE29AIBK93115212345678",
		Memo:          "KRAKEN-REF-8812",
		NameOnAccount: "Example Exchange Ltd",
		RoutingNumber: "011000015",
		SortCode:      "93-11-52",
		SWIFTCode:     "EXDBIE2DXXX",
		Tag:           "4471",
		TransitNumber: "00212",
	}
}

func TestCreateFundingAddress(t *testing.T) {
	t.Parallel()
	valid := FundingAddressRequest{
		Scope:   FundingScope{NetworkGroupID: fundingTestEVMGroup},
		Address: "0x5d7347ff6cd27a96c58e1426d45710c6d1535f92",
		Name:    "EVM Desk",
	}
	_, err := e.CreateFundingAddress(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateFundingAddress must reject a nil request")
	for _, tc := range []struct {
		name   string
		modify func(*FundingAddressRequest)
		err    error
	}{
		{"an empty scope", func(r *FundingAddressRequest) { r.Scope = FundingScope{} }, errFundingScopeEmpty},
		{"a scope setting two IDs", func(r *FundingAddressRequest) { r.Scope.NetworkID = "d9d375da-44b7-4be1-8a00-8b281acfe366" }, errFundingScopeAmbiguous},
		{"an empty address", func(r *FundingAddressRequest) { r.Address = "" }, errFundingAddressEmpty},
		{"a blank name", func(r *FundingAddressRequest) { r.Name = " \t" }, errFundingAddressNameEmpty},
	} {
		invalid := valid
		tc.modify(&invalid)
		_, err = e.CreateFundingAddress(t.Context(), &invalid)
		require.ErrorIsf(t, err, tc.err, "CreateFundingAddress must reject %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name                  string
		req                   *FundingAddressRequest
		query, body, response string
		exp                   *FundingAddressResponse
	}{
		{
			name: "network scope with a tag",
			req: &FundingAddressRequest{
				Scope:       FundingScope{NetworkID: "5a8e2c4f-7b1d-4e3a-9f6c-0d2b8e4a7c19"},
				Address:     "rLHzPsX6oXkzU2qL12kHCH8G8cnZv1rBJh",
				Tag:         "1830271466",
				Name:        "Exchange Deposit",
				Description: "Tagged XRP address at another exchange",
				AccountID:   fundingTestAccountID,
			},
			query: "account_id=" + fundingTestAccountID,
			body: `{"scope":{"network_id":"5a8e2c4f-7b1d-4e3a-9f6c-0d2b8e4a7c19"},"address_details":{"crypto":{"address":"rLHzPsX6oXkzU2qL12kHCH8G8cnZv1rBJh",` +
				`"tag":"1830271466"}},"name":"Exchange Deposit","description":"Tagged XRP address at another exchange"}`,
			response: `{"address_id":"AB7J4FF-BGM7G-V2JMIH","verified":true}`,
			exp:      &FundingAddressResponse{AddressID: "AB7J4FF-BGM7G-V2JMIH", Verified: true},
		},
		{
			name:     "network group scope",
			req:      &valid,
			body:     `{"scope":{"network_group_id":"` + fundingTestEVMGroup + `"},"address_details":{"crypto":{"address":"0x5d7347ff6cd27a96c58e1426d45710c6d1535f92"}},"name":"EVM Desk"}`,
			response: `{"address_id":"ABR6SXP-SF6CY-VJMONY","verified":true}`,
			exp:      &FundingAddressResponse{AddressID: "ABR6SXP-SF6CY-VJMONY", Verified: true},
		},
		{
			name: "method scope with a memo",
			req: &FundingAddressRequest{
				Scope:   FundingScope{MethodID: "b8e4f2a6-1d9c-4b7e-8f3a-6c0d2e5b9a14"},
				Address: "GCKFBEIYV2U22IO2BJ4KVJOIP7XPWQGQFKKWXR6DOSJBV7STMAQSMTGG",
				Memo:    "414792",
				Name:    "Stellar Wallet",
			},
			body: `{"scope":{"method_id":"b8e4f2a6-1d9c-4b7e-8f3a-6c0d2e5b9a14"},"address_details":{"crypto":` +
				`{"address":"GCKFBEIYV2U22IO2BJ4KVJOIP7XPWQGQFKKWXR6DOSJBV7STMAQSMTGG","memo":"414792"}},"name":"Stellar Wallet"}`,
			response: `{"address_id":"ABM3Q8R-K5T2W-PZ7NXC","verified":false}`,
			exp:      &FundingAddressResponse{AddressID: "ABM3Q8R-K5T2W-PZ7NXC"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := e
			if mockTests {
				ex = newFundingRequestServer(t, http.MethodPost, "/funding/v1/addresses", tc.query, tc.body, tc.response)
			}
			result, err := ex.CreateFundingAddress(t.Context(), tc.req)
			require.NoError(t, err, "CreateFundingAddress must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CreateFundingAddress should decode every field")
				return
			}
			assert.NotEmpty(t, result.AddressID, "CreateFundingAddress should return the address ID")
		})
	}
}

func TestCreateFundingWithdrawal(t *testing.T) {
	t.Parallel()
	valid := FundingWithdrawalRequest{
		Scope:     FundingScope{MethodID: "d4ec4d52-b159-428e-ba64-f45455a978a1"},
		AddressID: "ABR6SXP-SF6CY-VJMONY",
		Asset:     FundingAsset{Class: "currency", Name: currency.USDC},
		Amount:    5,
	}
	_, err := e.CreateFundingWithdrawal(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateFundingWithdrawal must reject a nil request")
	for _, tc := range []struct {
		name   string
		modify func(*FundingWithdrawalRequest)
		err    error
	}{
		{"an empty scope", func(r *FundingWithdrawalRequest) { r.Scope = FundingScope{} }, errFundingScopeEmpty},
		{"a scope setting two IDs", func(r *FundingWithdrawalRequest) { r.Scope.NetworkID = "bc7562cf-1e51-4308-b52b-d062aaa6aa3b" }, errFundingScopeAmbiguous},
		{"a network group scope", func(r *FundingWithdrawalRequest) { r.Scope = FundingScope{NetworkGroupID: fundingTestEVMGroup} }, errFundingNetworkGroupScope},
		{"an empty address ID", func(r *FundingWithdrawalRequest) { r.AddressID = "" }, errFundingAddressIDEmpty},
		{"an empty asset class", func(r *FundingWithdrawalRequest) { r.Asset.Class = "" }, errFundingAssetClassEmpty},
		{"an empty asset name", func(r *FundingWithdrawalRequest) { r.Asset.Name = currency.EMPTYCODE }, currency.ErrCurrencyCodeEmpty},
		{"an empty amount", func(r *FundingWithdrawalRequest) { r.Amount = 0 }, limits.ErrAmountBelowMin},
		{"a negative maximum fee", func(r *FundingWithdrawalRequest) { r.MaximumFee = -1 }, limits.ErrAmountBelowMin},
		{"a fee token with a maximum fee", func(r *FundingWithdrawalRequest) { r.FeeToken, r.MaximumFee = fundingTestFeeToken, 1 }, errFundingFeeConflict},
	} {
		invalid := valid
		tc.modify(&invalid)
		_, err = e.CreateFundingWithdrawal(t.Context(), &invalid)
		require.ErrorIsf(t, err, tc.err, "CreateFundingWithdrawal must reject %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	aapl, tsla := currency.NewCode("AAPLx"), currency.NewCode("TSLAx")
	for _, tc := range []struct {
		name                  string
		req                   *FundingWithdrawalRequest
		query, body, response string
		exp                   *FundingWithdrawalResponse
	}{
		{
			name: "quoted fee for a tokenised asset",
			req: &FundingWithdrawalRequest{
				Scope:            FundingScope{MethodID: "3c9d1e5a-7b2f-4e8c-a1d6-9f4b2c7e0a35"},
				AddressID:        "ABVXBF7-RC3Z5-WYUBPD",
				Asset:            FundingAsset{Class: "tokenized_asset", Name: aapl},
				Amount:           2.5,
				FeeIncluded:      true,
				FeeToken:         fundingTestFeeToken,
				RebaseMultiplier: "rebased",
				ExpectedAddress:  "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU",
				AccountID:        fundingTestAccountID,
			},
			query: "account_id=" + fundingTestAccountID,
			body: `{"scope":{"method_id":"3c9d1e5a-7b2f-4e8c-a1d6-9f4b2c7e0a35"},"address_id":"ABVXBF7-RC3Z5-WYUBPD",` +
				`"amount":{"asset_amount":{"asset":{"class":"tokenized_asset","name":"AAPLX"},"amount":"2.5"},"rebase_multiplier":"rebased"},` +
				`"fee":{"quoted_fee":{"token":"` + fundingTestFeeToken + `","rebase_multiplier":"rebased"},"fee_included":true},` +
				`"expected_address":"7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU"}`,
			response: `{"withdrawal_id":"FTh5Jw2-Kx8Lz3Mb6Nc9Pd1Qf4Rg7S","net_amount":{"asset_amount":{"asset":{"class":"tokenized_asset","name":"AAPLx"},` +
				`"amount":"2.48750000"},"rebase_multiplier":"rebased"},"gross_amount":{"asset_amount":{"asset":{"class":"tokenized_asset","name":"AAPLx"},` +
				`"amount":"2.50000000"},"rebase_multiplier":"rebased"},"fee":{"asset_amount":{"asset":{"class":"tokenized_asset","name":"AAPLx"},` +
				`"amount":"0.01250000"},"rebase_multiplier":"rebased"},"approval_request_id":"WAR-7Q2KD-5TBNE"}`,
			exp: &FundingWithdrawalResponse{
				WithdrawalID:      "FTh5Jw2-Kx8Lz3Mb6Nc9Pd1Qf4Rg7S",
				NetAmount:         FundingWithdrawalAmount{AssetAmount: fundingTestAmount("tokenized_asset", aapl, 2.4875), RebaseMultiplier: "rebased"},
				GrossAmount:       FundingWithdrawalAmount{AssetAmount: fundingTestAmount("tokenized_asset", aapl, 2.5), RebaseMultiplier: "rebased"},
				Fee:               FundingWithdrawalAmount{AssetAmount: fundingTestAmount("tokenized_asset", aapl, 0.0125), RebaseMultiplier: "rebased"},
				ApprovalRequestID: "WAR-7Q2KD-5TBNE",
			},
		},
		{
			name: "capped current fee",
			req: &FundingWithdrawalRequest{
				Scope:            FundingScope{NetworkID: "8b1f4d7a-2c6e-4a9b-b3d5-0e7f1a4c8d26"},
				AddressID:        "ABT4N6Q-Z2X8C-L5V7BM",
				Asset:            FundingAsset{Class: "tokenized_asset", Name: tsla},
				Amount:           1.3,
				MaximumFee:       0.02,
				RebaseMultiplier: "base",
			},
			body: `{"scope":{"network_id":"8b1f4d7a-2c6e-4a9b-b3d5-0e7f1a4c8d26"},"address_id":"ABT4N6Q-Z2X8C-L5V7BM",` +
				`"amount":{"asset_amount":{"asset":{"class":"tokenized_asset","name":"TSLAX"},"amount":"1.3"},"rebase_multiplier":"base"},` +
				`"fee":{"current_fee":{"max_fee":{"asset_amount":{"asset":{"class":"tokenized_asset","name":"TSLAX"},"amount":"0.02"},` +
				`"rebase_multiplier":"base"}},"fee_included":false}}`,
			response: `{"withdrawal_id":"FTa55AO-u7dazcZpBR1nS3Uk42k6LI","net_amount":{"asset_amount":{"asset":{"class":"tokenized_asset","name":"TSLAx"},` +
				`"amount":"1.30000000"},"rebase_multiplier":"base"},"gross_amount":{"asset_amount":{"asset":{"class":"tokenized_asset","name":"TSLAx"},` +
				`"amount":"1.31300000"},"rebase_multiplier":"base"},"fee":{"asset_amount":{"asset":{"class":"tokenized_asset","name":"TSLAx"},` +
				`"amount":"0.01300000"},"rebase_multiplier":"base"}}`,
			exp: &FundingWithdrawalResponse{
				WithdrawalID: "FTa55AO-u7dazcZpBR1nS3Uk42k6LI",
				NetAmount:    FundingWithdrawalAmount{AssetAmount: fundingTestAmount("tokenized_asset", tsla, 1.3), RebaseMultiplier: "base"},
				GrossAmount:  FundingWithdrawalAmount{AssetAmount: fundingTestAmount("tokenized_asset", tsla, 1.313), RebaseMultiplier: "base"},
				Fee:          FundingWithdrawalAmount{AssetAmount: fundingTestAmount("tokenized_asset", tsla, 0.013), RebaseMultiplier: "base"},
			},
		},
		{
			name: "current fee",
			req:  &valid,
			body: `{"scope":{"method_id":"d4ec4d52-b159-428e-ba64-f45455a978a1"},"address_id":"ABR6SXP-SF6CY-VJMONY",` +
				`"amount":{"asset_amount":{"asset":{"class":"currency","name":"USDC"},"amount":"5"}},"fee":{"current_fee":{},"fee_included":false}}`,
			response: `{"withdrawal_id":"FTVZiTI-e02T84mm87JmibnObWNdnW","net_amount":{"asset_amount":{"asset":{"class":"currency","name":"USDC"},` +
				`"amount":"5.00000000"}},"gross_amount":{"asset_amount":{"asset":{"class":"currency","name":"USDC"},"amount":"6.00000000"}},` +
				`"fee":{"asset_amount":{"asset":{"class":"currency","name":"USDC"},"amount":"1.00000000"}}}`,
			exp: &FundingWithdrawalResponse{
				WithdrawalID: "FTVZiTI-e02T84mm87JmibnObWNdnW",
				NetAmount:    FundingWithdrawalAmount{AssetAmount: fundingTestUSDC(5)},
				GrossAmount:  FundingWithdrawalAmount{AssetAmount: fundingTestUSDC(6)},
				Fee:          FundingWithdrawalAmount{AssetAmount: fundingTestUSDC(1)},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := e
			if mockTests {
				ex = newFundingRequestServer(t, http.MethodPost, "/funding/v1/withdrawals", tc.query, tc.body, tc.response)
			}
			result, err := ex.CreateFundingWithdrawal(t.Context(), tc.req)
			require.NoError(t, err, "CreateFundingWithdrawal must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CreateFundingWithdrawal should decode every field")
				return
			}
			assert.NotEmpty(t, result.WithdrawalID, "CreateFundingWithdrawal should return the withdrawal ID")
		})
	}
}

func TestDeleteFundingAddress(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteFundingAddress(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "DeleteFundingAddress must reject a nil request")
	_, err = e.DeleteFundingAddress(t.Context(), &FundingAddressDeletionRequest{AccountID: fundingTestAccountID})
	require.ErrorIs(t, err, errFundingAddressIDEmpty, "DeleteFundingAddress must reject an empty address ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.DeleteFundingAddress(t.Context(), &FundingAddressDeletionRequest{AddressID: "ABSXEMA-3FXQS-T7Q2FG", AccountID: fundingTestAccountID})
	require.NoError(t, err, "DeleteFundingAddress must not error")
	assert.Equal(t, &FundingAddressDeletionResponse{Deleted: true}, result, "DeleteFundingAddress should report the address deleted")
}

func TestListFundingAddresses(t *testing.T) {
	t.Parallel()
	_, err := e.ListFundingAddresses(t.Context(), &FundingAddressesRequest{Scope: FundingScope{MethodID: "67b765fd-4efd-42dc-8ce7-63352971d566", NetworkGroupID: fundingTestEVMGroup}})
	require.ErrorIs(t, err, errFundingScopeAmbiguous, "ListFundingAddresses must reject a scope setting two IDs")
	_, err = e.ListFundingAddresses(t.Context(), &FundingAddressesRequest{Limit: 501})
	require.ErrorIs(t, err, errInvalidFundingLimit, "ListFundingAddresses must reject a limit above 500")
	_, err = e.ListFundingAddresses(t.Context(), &FundingAddressesRequest{Scope: FundingScope{NetworkGroupID: fundingTestEVMGroup}, Cursor: "gqNrZXm-QUJTWEVNQS0zRlhRUy1UN1EyRkc"})
	require.ErrorIs(t, err, errFundingCursorWithFilters, "ListFundingAddresses must reject a scope with a cursor")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *FundingAddressesRequest
		exp  *FundingAddressesResponse
	}{
		{
			name: "nil request",
			exp: &FundingAddressesResponse{
				Addresses: []FundingAddress{
					{
						AddressID:      "ABQ5W2E-R7T9Y-U3I6OP",
						Scope:          FundingScope{NetworkID: "d9d375da-44b7-4be1-8a00-8b281acfe366"},
						AddressDetails: FundingWithdrawalAddressDetails{Crypto: &FundingWithdrawalCryptoAddress{Address: "0xBef7B36845cA31045E86D0B46DBCac4e675291c"}},
						Name:           "Personal Wallet",
						Description:    "My Ethereum hardware wallet address",
						Verified:       true,
					},
				},
				NextCursor: "gqNrZXm-QUJRNVcyRS1SN1Q5WS1VM0k2T1A",
			},
		},
		{
			name: "every address",
			req:  &FundingAddressesRequest{Limit: 2, AccountID: fundingTestAccountID},
			exp: &FundingAddressesResponse{
				Addresses: []FundingAddress{
					{
						AddressID: "AB7J4FF-BGM7G-V2JMIH",
						Scope:     FundingScope{NetworkID: "5a8e2c4f-7b1d-4e3a-9f6c-0d2b8e4a7c19"},
						AddressDetails: FundingWithdrawalAddressDetails{Crypto: &FundingWithdrawalCryptoAddress{
							Address: "rLHzPsX6oXkzU2qL12kHCH8G8cnZv1rBJh",
							Tag:     "1830271466",
							Beneficiary: FundingBeneficiary{
								Recipient:            "other",
								Type:                 "individual",
								FirstName:            "Jane",
								LastName:             "Citizen",
								Country:              "AU",
								StateOrProvince:      "NSW",
								AddressLine1:         "Level 4, 1 Example Street",
								AddressLine2:         "Unit 12",
								City:                 "Sydney",
								PostalCode:           "2000",
								CounterpartyVASPName: "Example Custody Pty Ltd",
							},
						}},
						Name:        "Exchange Deposit",
						Description: "Tagged XRP address at another exchange",
						Modifier:    "third_party",
						Verified:    true,
					},
					{
						AddressID:      "ABSXEMA-3FXQS-T7Q2FG",
						Scope:          FundingScope{MethodID: "9e4b2d7f-3c1a-4f8e-b6d0-5a2c8e1f4b73"},
						AddressDetails: FundingWithdrawalAddressDetails{Fiat: fundingTestWithdrawalFiatAddress()},
						Name:           "Settlement Account",
						Description:    "Fiat settlement",
					},
				},
				NextCursor: "gqNrZXm-QUJTWEVNQS0zRlhRUy1UN1EyRkc",
			},
		},
		{
			name: "network group scope",
			req:  &FundingAddressesRequest{Scope: FundingScope{NetworkGroupID: fundingTestEVMGroup}},
			exp: &FundingAddressesResponse{
				Addresses: []FundingAddress{
					{
						AddressID:      "ABR6SXP-SF6CY-VJMONY",
						Scope:          FundingScope{NetworkGroupID: fundingTestEVMGroup},
						AddressDetails: FundingWithdrawalAddressDetails{Crypto: &FundingWithdrawalCryptoAddress{Address: "0x5d7347ff6cd27a96c58e1426d45710c6d1535f92"}},
						Name:           "EVM Desk",
						Verified:       true,
					},
				},
			},
		},
		{
			name: "next page",
			req:  &FundingAddressesRequest{Cursor: "gqNrZXm-QUJTWEVNQS0zRlhRUy1UN1EyRkc", Limit: 2},
			exp: &FundingAddressesResponse{
				Addresses: []FundingAddress{
					{
						AddressID: "ABM3Q8R-K5T2W-PZ7NXC",
						Scope:     FundingScope{NetworkID: "2d7b9e3a-6c1f-4a8d-b5e2-9f0c3a6d8b41"},
						AddressDetails: FundingWithdrawalAddressDetails{Crypto: &FundingWithdrawalCryptoAddress{
							Address: "GCKFBEIYV2U22IO2BJ4KVJOIP7XPWQGQFKKWXR6DOSJBV7STMAQSMTGG",
							Memo:    "414792",
						}},
						Description: "Stellar wallet with a memo",
						Verified:    true,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ListFundingAddresses(t.Context(), tc.req)
			require.NoError(t, err, "ListFundingAddresses must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ListFundingAddresses should decode every field")
				return
			}
			assert.NotNil(t, result.Addresses, "ListFundingAddresses should return addresses")
		})
	}
}

func fundingTestWithdrawalFiatAddress() *FundingWithdrawalFiatAddress {
	return &FundingWithdrawalFiatAddress{
		AccountNumber: "12345678",
		AccountType:   "checking",
		Address:       "1 Sample Plaza, New York, NY 10004",
		BankName:      "Example Bank",
		BankCode:      "EXBK",
		BIC:           "EXBKUS33",
		Branch:        "Wall Street",
		BranchCode:    "0042",
		BankAddress:   "100 Example Avenue, New York, NY 10005",
		BSB:           "062-000",
		Beneficiary: FundingBeneficiary{
			Recipient:            "sender",
			Type:                 "business",
			FirstName:            "John",
			LastName:             "Smith",
			Country:              "GB",
			StateOrProvince:      "Greater London",
			AddressLine1:         "10 Sample Road",
			AddressLine2:         "Floor 3",
			City:                 "London",
			PostalCode:           "EC1A 1BB",
			CounterpartyVASPName: "Example Payments Ltd",
		},
		CardNumber:                "4111111111111111",
		CardNumberLastFour:        "1111",
		CardType:                  "visa",
		CardVerificationCode:      "123",
		ExpiryMonth:               "08",
		ExpiryYear:                "2029",
		IBAN:                      "GB33BUKB20201555555555",
		MaskedIBAN:                "GB33****5555",
		IntermediaryBankAddress:   "200 Correspondent Lane, London",
		IntermediaryBankName:      "Example Correspondent Bank",
		IntermediaryBranch:        "City",
		IntermediaryRoutingNumber: "021000021",
		IntermediarySWIFTCode:     "EXCBGB2L",
		Memo:                      "Invoice 88",
		MerchantReference:         "MREF-5521",
		NameOnAccount:             "Example Trading Ltd",
		Notes:                     "Settlement account",
		Password:                  "example-password",
		RoutingNumber:             "026009593",
		SortCode:                  "20-20-15",
		SWIFTCode:                 "EXBKUS3NXXX",
		Tag:                       "8812",
		ThirdPartyEmail:           "payments@example.com",
		TransactionID:             "TX-7781-2026",
		TransitNumber:             "00011",
		Username:                  "example-user",
		Signature:                 "3045022100f1e2d3c4",
	}
}

func TestListFundingAssets(t *testing.T) {
	t.Parallel()
	_, err := e.ListFundingAssets(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ListFundingAssets must reject a nil request")
	_, err = e.ListFundingAssets(t.Context(), &FundingAssetsRequest{Direction: "withdrawal"})
	require.ErrorIs(t, err, errInvalidFundingDirection, "ListFundingAssets must reject an undocumented direction")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *FundingAssetsRequest
		exp  *FundingAssetsResponse
	}{
		{
			name: "deposit",
			req:  &FundingAssetsRequest{Direction: "deposit", AccountID: fundingTestAccountID},
			exp: &FundingAssetsResponse{
				Currencies:      []FundingAvailableAsset{{Name: currency.USDC}, {Name: currency.BTC}},
				TokenisedAssets: []FundingAvailableAsset{{Name: currency.NewCode("TONXx")}, {Name: currency.NewCode("NVDAx")}},
			},
		},
		{
			name: "tokenised withdrawals",
			req:  &FundingAssetsRequest{Direction: "withdraw", AssetClass: "tokenized_asset"},
			exp:  &FundingAssetsResponse{TokenisedAssets: []FundingAvailableAsset{{Name: currency.NewCode("AAPLx")}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ListFundingAssets(t.Context(), tc.req)
			require.NoError(t, err, "ListFundingAssets must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ListFundingAssets should decode every field")
				return
			}
			assert.NotNil(t, result, "ListFundingAssets should return assets")
		})
	}
}

func TestListFundingClaimedAddresses(t *testing.T) {
	t.Parallel()
	_, err := e.ListFundingClaimedAddresses(t.Context(), &FundingClaimedAddressesRequest{Scope: FundingScope{NetworkGroupID: fundingTestEVMGroup}})
	require.ErrorIs(t, err, errFundingNetworkGroupScope, "ListFundingClaimedAddresses must reject a network group scope")
	_, err = e.ListFundingClaimedAddresses(t.Context(), &FundingClaimedAddressesRequest{Asset: FundingAsset{Name: currency.USDC}})
	require.ErrorIs(t, err, errFundingAssetClassEmpty, "ListFundingClaimedAddresses must reject an asset name without its class")
	_, err = e.ListFundingClaimedAddresses(t.Context(), &FundingClaimedAddressesRequest{Limit: 501})
	require.ErrorIs(t, err, errInvalidFundingLimit, "ListFundingClaimedAddresses must reject a limit above 500")
	_, err = e.ListFundingClaimedAddresses(t.Context(), &FundingClaimedAddressesRequest{Asset: FundingAsset{Class: "currency"}, Cursor: "gqNrZXm-RlRrM1J6OC1XbTVOcDJRczdUdjRYeTlCYzZEZjFH"})
	require.ErrorIs(t, err, errFundingCursorWithFilters, "ListFundingClaimedAddresses must reject an asset filter with a cursor")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	sharedEVMAddress := FundingClaimedAddress{
		MethodID:       "a001231f-488e-48c0-b36c-6e0d2c1ee247",
		ID:             "FTb7N2c-Qx4Zk9Lm2Pw8Rt6Vy1Ns3H",
		LastUsed:       time.Date(2026, 9, 30, 8, 12, 45, 0, time.UTC),
		ExpiryTime:     time.Date(2026, 10, 7, 8, 12, 45, 0, time.UTC),
		AddressDetails: FundingDepositAddressDetails{Crypto: &FundingDepositCryptoAddress{Address: "0x0d5cff23d40bcc6b98537c7ad5a839a247f546e1"}},
	}
	sharedForUSDC := sharedEVMAddress
	sharedForUSDC.MethodID = "27ede8db-804b-4d91-8e25-46b7b9668730"
	sharedForUSDC.SharesAddressesWithMethodID = "a001231f-488e-48c0-b36c-6e0d2c1ee247"
	for _, tc := range []struct {
		name string
		req  *FundingClaimedAddressesRequest
		exp  *FundingClaimedAddressesResponse
	}{
		{
			name: "nil request",
			exp: &FundingClaimedAddressesResponse{
				Addresses: []FundingClaimedAddress{
					{
						MethodID:       "3e7f8072-cc6d-4394-982a-5f4ca6ab27dd",
						ID:             "FTw3Ex5-Rc7Tv9Yb2Un4Im6Ok8Pl1A",
						LastUsed:       time.Date(2026, 10, 1, 6, 30, 0, 0, time.UTC),
						ExpiryTime:     time.Date(2026, 10, 31, 6, 30, 0, 0, time.UTC),
						AddressDetails: FundingDepositAddressDetails{Crypto: &FundingDepositCryptoAddress{Address: "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh"}},
					},
				},
			},
		},
		{
			name: "every address",
			req:  &FundingClaimedAddressesRequest{Limit: 2, AccountID: fundingTestAccountID},
			exp: &FundingClaimedAddressesResponse{
				Addresses: []FundingClaimedAddress{
					sharedEVMAddress,
					{
						MethodID:       "5e0b1c7a-2f4d-4e8b-9c6a-3d1f0e2b4a57",
						ID:             "FTk3Rz8-Wm5Np2Qs7Tv4Xy9Bc6Df1G",
						LastUsed:       time.Date(2026, 9, 28, 16, 40, 0, 0, time.UTC),
						ExpiryTime:     time.Date(2026, 10, 28, 16, 40, 0, 0, time.UTC),
						AddressDetails: FundingDepositAddressDetails{Fiat: fundingTestDepositFiatAddress()},
					},
				},
				NextCursor: "gqNrZXm-RlRrM1J6OC1XbTVOcDJRczdUdjRYeTlCYzZEZjFH",
			},
		},
		{
			name: "method and asset filters",
			req: &FundingClaimedAddressesRequest{
				Scope: FundingScope{MethodID: "27ede8db-804b-4d91-8e25-46b7b9668730"},
				Asset: FundingAsset{Class: "currency", Name: currency.USDC},
			},
			exp: &FundingClaimedAddressesResponse{Addresses: []FundingClaimedAddress{sharedForUSDC}},
		},
		{
			name: "next page",
			req:  &FundingClaimedAddressesRequest{Cursor: "gqNrZXm-RlRrM1J6OC1XbTVOcDJRczdUdjRYeTlCYzZEZjFH", Limit: 2},
			exp: &FundingClaimedAddressesResponse{
				Addresses: []FundingClaimedAddress{
					{
						MethodID:       "e1b7c3d9-5f2a-4c8e-a6b4-7d0f9e2c5a38",
						ID:             "FTp4Lw9-Hs2Kd7Mq5Nr8Tx3Vz6Bf1J",
						LastUsed:       time.Date(2026, 9, 12, 11, 5, 30, 0, time.UTC),
						ExpiryTime:     time.Date(2026, 10, 12, 11, 5, 30, 0, time.UTC),
						AddressDetails: FundingDepositAddressDetails{Crypto: &FundingDepositCryptoAddress{Address: "rPEPPER7kfTD9w2To4CQk6UCfuHM9c6GDY", Tag: "2059931772"}},
					},
					{
						MethodID:   "f6a2d8c4-9e3b-4d1f-8c7a-2b5e0f9d3a16",
						ID:         "FTr6Tz1-Ay3Bx5Cw7Dv9Eu2Ft4Gs6H",
						LastUsed:   time.Date(2026, 9, 9, 7, 45, 12, 0, time.UTC),
						ExpiryTime: time.Date(2026, 10, 9, 7, 45, 12, 0, time.UTC),
						AddressDetails: FundingDepositAddressDetails{Crypto: &FundingDepositCryptoAddress{
							Address: "GA5XIGA5C7QTPTWXQHY6MCJRMTRZDOSHR6EFIBNDQTCQHG262N4GGKTM",
							Memo:    "7731005",
						}},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ListFundingClaimedAddresses(t.Context(), tc.req)
			require.NoError(t, err, "ListFundingClaimedAddresses must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ListFundingClaimedAddresses should decode every field")
				return
			}
			assert.NotNil(t, result, "ListFundingClaimedAddresses should return a page")
		})
	}
}

func TestListFundingDepositLimits(t *testing.T) {
	t.Parallel()
	_, err := e.ListFundingDepositLimits(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ListFundingDepositLimits must reject a nil request")
	_, err = e.ListFundingDepositLimits(t.Context(), &FundingLimitsRequest{Asset: FundingAsset{Class: "crypto", Name: currency.USDC}})
	require.ErrorIs(t, err, errInvalidFundingAssetClass, "ListFundingDepositLimits must reject an undocumented asset class")
	_, err = e.ListFundingDepositLimits(t.Context(), &FundingLimitsRequest{Asset: FundingAsset{Class: "currency"}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "ListFundingDepositLimits must reject an empty asset")
	_, err = e.ListFundingDepositLimits(t.Context(), &FundingLimitsRequest{Asset: FundingAsset{Class: "currency", Name: currency.USDC}, PreferredAsset: FundingAsset{Class: "currency"}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "ListFundingDepositLimits must reject a preferred asset class without its name")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.ListFundingDepositLimits(t.Context(), &FundingLimitsRequest{
		Asset:          FundingAsset{Class: "currency", Name: currency.USDC},
		PreferredAsset: FundingAsset{Class: "currency", Name: currency.EUR},
		AccountID:      fundingTestAccountID,
	})
	require.NoError(t, err, "ListFundingDepositLimits must not error")
	if !mockTests {
		assert.NotNil(t, result.DepositLimits, "ListFundingDepositLimits should return limits")
		return
	}
	maximumAmount := fundingTestUSDC(99995.5013)
	exp := &FundingDepositLimitsResponse{
		DepositLimits: []FundingDepositLimit{
			{
				MethodID:      "742656ee-ecc8-4287-a301-5a472ca16c24",
				MaximumAmount: &maximumAmount,
				Limits: []FundingLimit{
					{
						TimeWindow: 86400,
						Limit: FundingLimitValues{
							LimitType: "equiv_amount_usd",
							Remaining: fundingTestLimitAmounts(99995.5013, 99995.5014, fundingTestAmount("currency", currency.EUR, 85996.13)),
							Maximum:   fundingTestLimitAmounts(100000, 100000.0001, fundingTestAmount("currency", currency.EUR, 86000)),
							Used:      fundingTestLimitAmounts(4.4987, 4.4988, fundingTestAmount("currency", currency.EUR, 3.87)),
						},
					},
					{
						TimeWindow: 2592000,
						Limit: FundingLimitValues{
							LimitType: "attempt",
							Remaining: FundingLimitValue{Count: 18},
							Maximum:   FundingLimitValue{Count: 20},
							Used:      FundingLimitValue{Count: 2},
						},
					},
				},
			},
			{MethodID: "98de2213-0e3c-422c-9b08-81078557ad3a", Limits: []FundingLimit{}},
		},
	}
	assert.Equal(t, exp, result, "ListFundingDepositLimits should decode every field")
}

func fundingTestLimitAmounts(policy, usd types.Number, requested FundingAmount) FundingLimitValue {
	return FundingLimitValue{
		PolicyAssetAmount:    fundingTestAmount("currency", currency.USD, policy),
		USDAmount:            fundingTestAmount("currency", currency.USD, usd),
		RequestedAssetAmount: requested,
	}
}

func TestListFundingDeposits(t *testing.T) {
	t.Parallel()
	_, err := e.ListFundingDeposits(t.Context(), &FundingDepositsRequest{Scope: FundingScope{MethodID: "3e7f8072-cc6d-4394-982a-5f4ca6ab27dd", NetworkID: "b336ce74-8d60-42b8-8714-b1095e06b711"}})
	require.ErrorIs(t, err, errFundingScopeAmbiguous, "ListFundingDeposits must reject a scope setting two IDs")
	_, err = e.ListFundingDeposits(t.Context(), &FundingDepositsRequest{Asset: FundingAsset{Name: currency.USDC}})
	require.ErrorIs(t, err, errFundingAssetClassEmpty, "ListFundingDeposits must reject an asset name without its class")
	_, err = e.ListFundingDeposits(t.Context(), &FundingDepositsRequest{Statuses: []string{"pending"}, StatusTo: "success"})
	require.ErrorIs(t, err, errFundingStatusFilterConflict, "ListFundingDeposits must reject a status list with a status range")
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC)
	_, err = e.ListFundingDeposits(t.Context(), &FundingDepositsRequest{StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "ListFundingDeposits must reject a start after the end")
	_, err = e.ListFundingDeposits(t.Context(), &FundingDepositsRequest{Limit: 501})
	require.ErrorIs(t, err, errInvalidFundingLimit, "ListFundingDeposits must reject a limit above 500")
	_, err = e.ListFundingDeposits(t.Context(), &FundingDepositsRequest{Statuses: []string{"pending"}, Cursor: "gqNrZXm-RlRBc3hlMS1QSHQ4TmlOb2R2dE9FdzhGQnZMQUo"})
	require.ErrorIs(t, err, errFundingCursorWithFilters, "ListFundingDeposits must reject a status filter with a cursor")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	twentyUSDC, halfUSDC, depositUSDC := fundingTestUSDC(20), fundingTestUSDC(0.5), fundingTestUSDC(5555)
	pagedUSDC, pagedFee := fundingTestUSDC(75), fundingTestUSDC(0.25)
	settledUSDC, settledFee := fundingTestUSDC(150), fundingTestUSDC(0.75)
	for _, tc := range []struct {
		name string
		req  *FundingDepositsRequest
		exp  *FundingDepositsResponse
	}{
		{
			name: "nil request",
			exp: &FundingDepositsResponse{
				Deposits: []FundingDeposit{
					{
						DepositID:    "FTz8Xc4-Vb6Nm1Qw3Er5Ty7Ui9Op2S",
						MethodID:     "3e7f8072-cc6d-4394-982a-5f4ca6ab27dd",
						NetworkID:    "b336ce74-8d60-42b8-8714-b1095e06b711",
						Status:       "settled",
						Amount:       &settledUSDC,
						Fee:          &settledFee,
						CreationTime: time.Date(2026, 9, 2, 17, 20, 44, 0, time.UTC),
					},
				},
				NextCursor: "gqNrZXm-RlR6OFhjNC1WYjZObTFRdzNFcjVUeTdVaTlPcDJT",
			},
		},
		{
			name: "status list",
			req: &FundingDepositsRequest{
				Asset:     FundingAsset{Class: "currency", Name: currency.USDC},
				Scope:     FundingScope{NetworkID: "b336ce74-8d60-42b8-8714-b1095e06b711"},
				Statuses:  []string{"pending", "success"},
				StartTime: start,
				EndTime:   end,
				Limit:     20,
				AccountID: fundingTestAccountID,
			},
			exp: &FundingDepositsResponse{
				Deposits: []FundingDeposit{
					{
						DepositID:    "FTcQ4qW-fWGQbQwUfqdnZo4dsMn1ao",
						MethodID:     "3e7f8072-cc6d-4394-982a-5f4ca6ab27dd",
						NetworkID:    "b336ce74-8d60-42b8-8714-b1095e06b711",
						Status:       "success",
						Amount:       &twentyUSDC,
						Fee:          &halfUSDC,
						CreationTime: time.Date(2026, 7, 1, 8, 31, 33, 0, time.UTC),
					},
					{
						DepositID:    "FTAsxe1-PHt8Ni7NodvtOEw8FBvLAJ",
						MethodID:     "a897bb0b-e7bc-4eff-969f-433bc811e4ad",
						NetworkID:    "b336ce74-8d60-42b8-8714-b1095e06b711",
						Status:       "pending",
						Amount:       &depositUSDC,
						CreationTime: time.Date(2026, 6, 23, 8, 41, 17, 0, time.UTC),
					},
				},
				NextCursor: "gqNrZXm-RlRBc3hlMS1QSHQ4TmlOb2R2dE9FdzhGQnZMQUo",
			},
		},
		{
			name: "status range",
			req: &FundingDepositsRequest{
				Asset:            FundingAsset{Class: "tokenized_asset"},
				Scope:            FundingScope{NetworkGroupID: fundingTestSolanaGroup},
				StatusFrom:       "initial",
				StatusTo:         "settled",
				RebaseMultiplier: "base",
			},
			exp: &FundingDepositsResponse{
				Deposits: []FundingDeposit{
					{
						DepositID:    "FTn8Gv2-Jk4Lp6Qr1St3Uw5Xz7Ab9C",
						MethodID:     "7d2c5e8f-1a4b-4c6d-9e0f-2b3a5c7d9e1f",
						NetworkID:    "3f8a1c6e-9b2d-4e7f-8a0c-5d1e3b7f9a2c",
						Status:       "initial",
						CreationTime: time.Date(2026, 8, 14, 19, 2, 51, 0, time.UTC),
					},
				},
			},
		},
		{
			name: "next page",
			req:  &FundingDepositsRequest{Cursor: "gqNrZXm-RlRBc3hlMS1QSHQ4TmlOb2R2dE9FdzhGQnZMQUo", Limit: 20},
			exp: &FundingDepositsResponse{
				Deposits: []FundingDeposit{
					{
						DepositID:    "FTm2Bx7-Yc9Dv4Ew6Fh1Gj3Hk5Ln8P",
						MethodID:     "3e7f8072-cc6d-4394-982a-5f4ca6ab27dd",
						NetworkID:    "b336ce74-8d60-42b8-8714-b1095e06b711",
						Status:       "success",
						Amount:       &pagedUSDC,
						Fee:          &pagedFee,
						CreationTime: time.Date(2026, 6, 20, 3, 15, 9, 0, time.UTC),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ListFundingDeposits(t.Context(), tc.req)
			require.NoError(t, err, "ListFundingDeposits must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ListFundingDeposits should decode every field")
				return
			}
			assert.NotNil(t, result, "ListFundingDeposits should return a page")
		})
	}
}

func TestListFundingMethods(t *testing.T) {
	t.Parallel()
	_, err := e.ListFundingMethods(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ListFundingMethods must reject a nil request")
	_, err = e.ListFundingMethods(t.Context(), &FundingMethodsRequest{})
	require.ErrorIs(t, err, errInvalidFundingDirection, "ListFundingMethods must reject an empty direction")
	_, err = e.ListFundingMethods(t.Context(), &FundingMethodsRequest{Direction: "deposit", Asset: FundingAsset{Name: currency.USDC}})
	require.ErrorIs(t, err, errFundingAssetClassEmpty, "ListFundingMethods must reject an asset name without its class")
	_, err = e.ListFundingMethods(t.Context(), &FundingMethodsRequest{Direction: "deposit", Limit: 10001})
	require.ErrorIs(t, err, errInvalidFundingLimit, "ListFundingMethods must reject a limit above 10000")
	_, err = e.ListFundingMethods(t.Context(), &FundingMethodsRequest{Direction: "deposit", RebaseMultiplier: "base", Cursor: "gqNrZXm-YzRhN2UyZDktNmIxZi00YTNlLThkNWMtMGY5YjJlN2E0YzE4"})
	require.ErrorIs(t, err, errFundingCursorWithFilters, "ListFundingMethods must reject a rebase multiplier with a cursor")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	debitOnly, exempt := false, true
	for _, tc := range []struct {
		name string
		req  *FundingMethodsRequest
		exp  *FundingMethodsResponse
	}{
		{
			name: "deposit methods",
			req:  &FundingMethodsRequest{Direction: "deposit", Asset: FundingAsset{Class: "currency"}, RebaseMultiplier: "rebased", Limit: 2, AccountID: fundingTestAccountID},
			exp: &FundingMethodsResponse{
				Methods: []FundingMethod{
					{
						Deposit: &FundingMethodDeposit{
							SharesAddressesWithMethodID: "a001231f-488e-48c0-b36c-6e0d2c1ee247",
							AddressSetupFee:             fundingTestUSDC(1),
							AddressGeneration:           FundingAddressGeneration{Status: "limited", Limit: 5},
						},
						Asset:         FundingAsset{Class: "currency", Name: currency.USDC},
						MethodID:      "27ede8db-804b-4d91-8e25-46b7b9668730",
						MethodName:    "Standard",
						MinimumAmount: 2,
						MaximumAmount: 1000000,
						Fees: FundingMethodFees{
							Base:       fundingTestUSDC(0.5),
							Percentage: 0.1,
							Included:   true,
							Minimum:    fundingTestUSDC(0.25),
							Maximum:    fundingTestUSDC(50),
						},
						Network: FundingMethodNetwork{
							NetworkID:          "d9d375da-44b7-4be1-8a00-8b281acfe366",
							NetworkName:        "Ethereum",
							ContractAddress:    "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							OnChainAssetSymbol: "USDC",
						},
					},
					{
						Deposit: &FundingMethodDeposit{
							AddressGeneration:      FundingAddressGeneration{Status: "unsupported"},
							AllowCreditCards:       &debitOnly,
							ExemptedWithdrawalHold: &exempt,
						},
						Asset:         FundingAsset{Class: "currency", Name: currency.USD},
						MethodID:      "c4a7e2d9-6b1f-4a3e-8d5c-0f9b2e7a4c18",
						MethodName:    "Debit Card",
						MinimumAmount: 10,
						MaximumAmount: 5000,
						Fees:          FundingMethodFees{Base: fundingTestAmount("currency", currency.USD, 0.3), Percentage: 3.75},
					},
				},
				NextCursor: "gqNrZXm-YzRhN2UyZDktNmIxZi00YTNlLThkNWMtMGY5YjJlN2E0YzE4",
			},
		},
		{
			name: "withdrawal methods by asset",
			req:  &FundingMethodsRequest{Direction: "withdraw", Asset: FundingAsset{Class: "currency", Name: currency.USDC}},
			exp: &FundingMethodsResponse{
				Methods: []FundingMethod{
					{
						Asset:         FundingAsset{Class: "currency", Name: currency.USDC},
						MethodID:      "d4ec4d52-b159-428e-ba64-f45455a978a1",
						MethodName:    "USDC - Arbitrum One",
						MinimumAmount: 0.01,
						Fees:          FundingMethodFees{Base: fundingTestUSDC(1)},
						Network: FundingMethodNetwork{
							NetworkID:          "bc7562cf-1e51-4308-b52b-d062aaa6aa3b",
							NetworkName:        "Arbitrum One",
							ContractAddress:    "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
							OnChainAssetSymbol: "USDC",
						},
					},
				},
			},
		},
		{
			name: "next page",
			req:  &FundingMethodsRequest{Direction: "deposit", Cursor: "gqNrZXm-YzRhN2UyZDktNmIxZi00YTNlLThkNWMtMGY5YjJlN2E0YzE4", Limit: 2},
			exp: &FundingMethodsResponse{
				Methods: []FundingMethod{
					{
						Deposit:       &FundingMethodDeposit{AddressGeneration: FundingAddressGeneration{Status: "unlimited"}},
						Asset:         FundingAsset{Class: "currency", Name: currency.ETH},
						MethodID:      "a001231f-488e-48c0-b36c-6e0d2c1ee247",
						MethodName:    "Ether",
						MinimumAmount: 0.002,
						Fees:          FundingMethodFees{Base: fundingTestAmount("currency", currency.ETH, 0.0001), Included: true},
						Network:       FundingMethodNetwork{NetworkID: "d9d375da-44b7-4be1-8a00-8b281acfe366", NetworkName: "Ethereum", OnChainAssetSymbol: "ETH"},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ListFundingMethods(t.Context(), tc.req)
			require.NoError(t, err, "ListFundingMethods must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ListFundingMethods should decode every field")
				return
			}
			assert.NotEmpty(t, result.Methods, "ListFundingMethods should return methods")
		})
	}
}

func TestListFundingNetworks(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *FundingNetworksRequest
		exp  *FundingNetworksResponse
	}{
		{
			name: "default account",
			exp: &FundingNetworksResponse{
				NetworkGroups: []FundingNetworkGroup{
					{NetworkGroupID: fundingTestSolanaGroup, Name: "SVM", NetworkIDs: []string{"8b1f4d7a-2c6e-4a9b-b3d5-0e7f1a4c8d26"}},
				},
				Networks: []FundingNetwork{
					{NetworkID: "8b1f4d7a-2c6e-4a9b-b3d5-0e7f1a4c8d26", Name: "Solana"},
					{NetworkID: "2d7b9e3a-6c1f-4a8d-b5e2-9f0c3a6d8b41", Name: "Stellar"},
				},
			},
		},
		{
			name: "wallet account",
			req:  &FundingNetworksRequest{AccountID: fundingTestAccountID},
			exp: &FundingNetworksResponse{
				NetworkGroups: []FundingNetworkGroup{
					{
						NetworkGroupID: fundingTestEVMGroup,
						Name:           "EVM",
						NetworkIDs:     []string{"d9d375da-44b7-4be1-8a00-8b281acfe366", "bc7562cf-1e51-4308-b52b-d062aaa6aa3b"},
					},
				},
				Networks: []FundingNetwork{
					{NetworkID: "d9d375da-44b7-4be1-8a00-8b281acfe366", Name: "Ethereum"},
					{NetworkID: "bc7562cf-1e51-4308-b52b-d062aaa6aa3b", Name: "Arbitrum One"},
					{NetworkID: "0b6b3a4e-5c1f-4c9e-9a51-2e8d7f6c4b13", Name: "Bitcoin"},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ListFundingNetworks(t.Context(), tc.req)
			require.NoError(t, err, "ListFundingNetworks must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ListFundingNetworks should decode every field")
				return
			}
			assert.NotEmpty(t, result.Networks, "ListFundingNetworks should return networks")
		})
	}
}

func TestListFundingWithdrawalLimits(t *testing.T) {
	t.Parallel()
	_, err := e.ListFundingWithdrawalLimits(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ListFundingWithdrawalLimits must reject a nil request")
	_, err = e.ListFundingWithdrawalLimits(t.Context(), &FundingLimitsRequest{Asset: FundingAsset{Name: currency.USDC}})
	require.ErrorIs(t, err, errInvalidFundingAssetClass, "ListFundingWithdrawalLimits must reject an empty asset class")
	_, err = e.ListFundingWithdrawalLimits(t.Context(), &FundingLimitsRequest{Asset: FundingAsset{Class: "currency"}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "ListFundingWithdrawalLimits must reject an empty asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.ListFundingWithdrawalLimits(t.Context(), &FundingLimitsRequest{
		Asset:          FundingAsset{Class: "currency", Name: currency.USDC},
		PreferredAsset: FundingAsset{Name: currency.USD},
		AccountID:      fundingTestAccountID,
	})
	require.NoError(t, err, "ListFundingWithdrawalLimits must not error")
	if !mockTests {
		assert.NotNil(t, result.WithdrawalLimits, "ListFundingWithdrawalLimits should return limits")
		return
	}
	exp := &FundingWithdrawalLimitsResponse{
		AvailableBalance: fundingTestUSDC(689005.55897887),
		WithdrawalLimits: []FundingWithdrawalLimit{
			{
				MethodID:      "ec344850-a2bf-4fd8-b5c4-eb902a61cd03",
				MaximumAmount: fundingTestUSDC(500155.05091629),
				MaximumReason: "limits",
				Limits: []FundingLimit{
					{
						TimeWindow: 86400,
						Limit: FundingLimitValues{
							LimitType: "equiv_amount_usd",
							Remaining: fundingTestLimitAmounts(499995.0013, 499995.0012, fundingTestAmount("currency", currency.USD, 499995.0011)),
							Maximum:   fundingTestLimitAmounts(500000, 500000.0001, fundingTestAmount("currency", currency.USD, 500000.0002)),
							Used:      fundingTestLimitAmounts(4.9987, 4.9986, fundingTestAmount("currency", currency.USD, 4.9985)),
						},
					},
					{
						TimeWindow: 604800,
						Limit: FundingLimitValues{
							LimitType: "success",
							Remaining: FundingLimitValue{Count: 4},
							Maximum:   FundingLimitValue{Count: 5},
							Used:      FundingLimitValue{Count: 1},
						},
					},
				},
			},
			{
				MethodID:      "d4ec4d52-b159-428e-ba64-f45455a978a1",
				MaximumAmount: fundingTestUSDC(689005.55897887),
				MaximumReason: "balance",
				Limits:        []FundingLimit{},
			},
		},
	}
	assert.Equal(t, exp, result, "ListFundingWithdrawalLimits should decode every field")
}

func TestListFundingWithdrawals(t *testing.T) {
	t.Parallel()
	_, err := e.ListFundingWithdrawals(t.Context(), &FundingWithdrawalsRequest{Scope: FundingScope{NetworkID: "0b6b3a4e-5c1f-4c9e-9a51-2e8d7f6c4b13", NetworkGroupID: fundingTestEVMGroup}})
	require.ErrorIs(t, err, errFundingScopeAmbiguous, "ListFundingWithdrawals must reject a scope setting two IDs")
	_, err = e.ListFundingWithdrawals(t.Context(), &FundingWithdrawalsRequest{Asset: FundingAsset{Name: currency.BTC}})
	require.ErrorIs(t, err, errFundingAssetClassEmpty, "ListFundingWithdrawals must reject an asset name without its class")
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	_, err = e.ListFundingWithdrawals(t.Context(), &FundingWithdrawalsRequest{StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "ListFundingWithdrawals must reject a start after the end")
	_, err = e.ListFundingWithdrawals(t.Context(), &FundingWithdrawalsRequest{Limit: 501})
	require.ErrorIs(t, err, errInvalidFundingLimit, "ListFundingWithdrawals must reject a limit above 500")
	_, err = e.ListFundingWithdrawals(t.Context(), &FundingWithdrawalsRequest{Status: "success", Cursor: "gqNrZXm-RlRxOVdjMy1SdjVUeDdZejJBYjRDZDZFZjhHaDFK"})
	require.ErrorIs(t, err, errFundingCursorWithFilters, "ListFundingWithdrawals must reject a status with a cursor")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	tsla := currency.NewCode("TSLAx")
	for _, tc := range []struct {
		name string
		req  *FundingWithdrawalsRequest
		exp  *FundingWithdrawalsResponse
	}{
		{
			name: "nil request",
			exp: &FundingWithdrawalsResponse{
				Withdrawals: []FundingWithdrawal{
					{
						WithdrawalID: "FTe4Rt6-Yu8Io0Pa2Sd4Fg6Hj8Kl1Z",
						Amount:       fundingTestUSDC(250),
						Fee:          fundingTestUSDC(2.5),
						MethodID:     "67b765fd-4efd-42dc-8ce7-63352971d566",
						Status:       "pending",
						CreationTime: time.Date(2026, 10, 8, 22, 5, 13, 0, time.UTC),
						AddressID:    "AB7J4FF-BGM7G-V2JMIH",
					},
				},
			},
		},
		{
			name: "filters",
			req: &FundingWithdrawalsRequest{
				Asset:     FundingAsset{Class: "currency", Name: currency.BTC},
				Scope:     FundingScope{NetworkID: "0b6b3a4e-5c1f-4c9e-9a51-2e8d7f6c4b13"},
				Status:    "success",
				StartTime: start,
				EndTime:   end,
				Limit:     20,
				AccountID: fundingTestAccountID,
			},
			exp: &FundingWithdrawalsResponse{
				Withdrawals: []FundingWithdrawal{
					{
						WithdrawalID:       "FTq9Wc3-Rv5Tx7Yz2Ab4Cd6Ef8Gh1J",
						Amount:             fundingTestAmount("currency", currency.BTC, 0.05),
						Fee:                fundingTestAmount("currency", currency.BTC, 0.00002),
						MethodID:           "8a3d6f1c-4e7b-4a2d-9c5f-1e8b3d6a9c2f",
						Status:             "success",
						CreationTime:       time.Date(2026, 8, 10, 14, 22, 5, 0, time.UTC),
						AddressID:          "ABK2M7P-Q4R8S-T3V9WX",
						OnChainTransaction: "b6f6991d03df0e2e04dafffcd6bc418aac66049e2cd74b80f14ac86db1e3f0da",
						OutputIndex:        1,
					},
				},
				NextCursor: "gqNrZXm-RlRxOVdjMy1SdjVUeDdZejJBYjRDZDZFZjhHaDFK",
			},
		},
		{
			name: "tokenised assets",
			req: &FundingWithdrawalsRequest{
				Asset:            FundingAsset{Class: "tokenized_asset"},
				Scope:            FundingScope{NetworkGroupID: fundingTestSolanaGroup},
				RebaseMultiplier: "base",
			},
			exp: &FundingWithdrawalsResponse{
				Withdrawals: []FundingWithdrawal{
					{
						WithdrawalID: "FTa55AO-u7dazcZpBR1nS3Uk42k6LI",
						Amount:       fundingTestAmount("tokenized_asset", tsla, 1.3),
						Fee:          fundingTestAmount("tokenized_asset", tsla, 0.013),
						MethodID:     "5104c28e-9da8-43c8-8194-33d9ea1fbdc0",
						Status:       "pending",
						CreationTime: time.Date(2026, 8, 12, 10, 7, 4, 0, time.UTC),
						AddressID:    "ABVXBF7-RC3Z5-WYUBPD",
					},
				},
			},
		},
		{
			name: "next page",
			req:  &FundingWithdrawalsRequest{Cursor: "gqNrZXm-RlRxOVdjMy1SdjVUeDdZejJBYjRDZDZFZjhHaDFK", Limit: 20},
			exp: &FundingWithdrawalsResponse{
				Withdrawals: []FundingWithdrawal{
					{
						WithdrawalID:       "FTVZiTI-e02T84mm87JmibnObWNdnW",
						Amount:             fundingTestAmount("currency", currency.BTC, 0.125),
						Fee:                fundingTestAmount("currency", currency.BTC, 0.00003),
						MethodID:           "8a3d6f1c-4e7b-4a2d-9c5f-1e8b3d6a9c2f",
						Status:             "success",
						CreationTime:       time.Date(2026, 8, 3, 9, 41, 27, 0, time.UTC),
						OnChainTransaction: "4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b",
						OutputIndex:        2,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ListFundingWithdrawals(t.Context(), tc.req)
			require.NoError(t, err, "ListFundingWithdrawals must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ListFundingWithdrawals should decode every field")
				return
			}
			assert.NotNil(t, result, "ListFundingWithdrawals should return a page")
		})
	}
}

func TestUpdateFundingAddress(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateFundingAddress(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdateFundingAddress must reject a nil request")
	_, err = e.UpdateFundingAddress(t.Context(), &FundingAddressUpdateRequest{Name: "Cold Wallet"})
	require.ErrorIs(t, err, errFundingAddressIDEmpty, "UpdateFundingAddress must reject an empty address ID")
	_, err = e.UpdateFundingAddress(t.Context(), &FundingAddressUpdateRequest{AddressID: "AB7J4FF-BGM7G-V2JMIH"})
	require.ErrorIs(t, err, errFundingAddressUpdateEmpty, "UpdateFundingAddress must reject a request changing nothing")
	_, err = e.UpdateFundingAddress(t.Context(), &FundingAddressUpdateRequest{AddressID: "AB7J4FF-BGM7G-V2JMIH", Name: "  "})
	require.ErrorIs(t, err, errFundingAddressNameEmpty, "UpdateFundingAddress must reject a blank name")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name                        string
		req                         *FundingAddressUpdateRequest
		path, query, body, response string
		exp                         *FundingAddressUpdateResponse
	}{
		{
			name:     "name and description",
			req:      &FundingAddressUpdateRequest{AddressID: "AB7J4FF-BGM7G-V2JMIH", Name: "Cold Wallet", Description: "Hardware wallet in the safe", AccountID: fundingTestAccountID},
			path:     "/funding/v1/addresses/AB7J4FF-BGM7G-V2JMIH",
			query:    "account_id=" + fundingTestAccountID,
			body:     `{"name":"Cold Wallet","description":"Hardware wallet in the safe"}`,
			response: `{"verified":true}`,
			exp:      &FundingAddressUpdateResponse{Verified: true},
		},
		{
			name:     "description",
			req:      &FundingAddressUpdateRequest{AddressID: "ABM3Q8R-K5T2W-PZ7NXC", Description: "Stellar wallet with a memo"},
			path:     "/funding/v1/addresses/ABM3Q8R-K5T2W-PZ7NXC",
			body:     `{"description":"Stellar wallet with a memo"}`,
			response: `{"verified":false}`,
			exp:      &FundingAddressUpdateResponse{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := e
			if mockTests {
				ex = newFundingRequestServer(t, http.MethodPut, tc.path, tc.query, tc.body, tc.response)
			}
			result, err := ex.UpdateFundingAddress(t.Context(), tc.req)
			require.NoError(t, err, "UpdateFundingAddress must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "UpdateFundingAddress should decode every field")
				return
			}
			assert.NotNil(t, result, "UpdateFundingAddress should return the address's verification")
		})
	}
}

func TestFundingLimitValueUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, data string
		exp        FundingLimitValue
	}{
		{name: "quoted count", data: `"18"`, exp: FundingLimitValue{Count: 18}},
		{name: "bare count", data: `7`, exp: FundingLimitValue{Count: 7}},
		{
			name: "amounts",
			data: `{"policy_asset_amount":{"asset":{"class":"currency","name":"USD"},"amount":"10"},` +
				`"usd_amount":{"asset":{"class":"currency","name":"USD"},"amount":"11"},` +
				`"requested_asset_amount":{"asset":{"class":"currency","name":"EUR"},"amount":"9"}}`,
			exp: FundingLimitValue{
				PolicyAssetAmount:    fundingTestAmount("currency", currency.USD, 10),
				USDAmount:            fundingTestAmount("currency", currency.USD, 11),
				RequestedAssetAmount: fundingTestAmount("currency", currency.EUR, 9),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var value FundingLimitValue
			require.NoError(t, value.UnmarshalJSON([]byte(tc.data)), "UnmarshalJSON must not error")
			assert.Equal(t, tc.exp, value, "UnmarshalJSON should decode the limit value")
		})
	}
	var value FundingLimitValue
	assert.Error(t, value.UnmarshalJSON([]byte(`true`)), "UnmarshalJSON should reject a boolean")
	assert.Error(t, value.UnmarshalJSON([]byte(`{"usd_amount":5}`)), "UnmarshalJSON should reject an amount that is not an object")
}

func TestFundingAssetValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		asset        FundingAsset
		nameRequired bool
		err          error
		empty        bool
	}{
		{name: "no asset filter", empty: true},
		{name: "no asset where one is required", nameRequired: true, err: errFundingAssetClassEmpty, empty: true},
		{name: "class filter", asset: FundingAsset{Class: "currency"}},
		{name: "class without the required name", asset: FundingAsset{Class: "currency"}, nameRequired: true, err: currency.ErrCurrencyCodeEmpty},
		{name: "name without a class", asset: FundingAsset{Name: currency.BTC}, err: errFundingAssetClassEmpty},
		{name: "class and name", asset: FundingAsset{Class: "currency", Name: currency.BTC}, nameRequired: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.asset.validate(tc.nameRequired), tc.err, "validate should return the expected error")
			assert.Equal(t, tc.empty, tc.asset.isEmpty(), "isEmpty should report whether the asset sets neither class nor name")
		})
	}
}

func TestFundingScopeValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                    string
		scope                   FundingScope
		required, networkGroups bool
		err                     error
		empty                   bool
	}{
		{name: "no optional scope", empty: true},
		{name: "no required scope", required: true, err: errFundingScopeEmpty, empty: true},
		{name: "method", scope: FundingScope{MethodID: "method"}, required: true},
		{name: "network", scope: FundingScope{NetworkID: "network"}, required: true},
		{name: "network group", scope: FundingScope{NetworkGroupID: "group"}, required: true, networkGroups: true},
		{name: "unsupported network group", scope: FundingScope{NetworkGroupID: "group"}, err: errFundingNetworkGroupScope},
		{name: "method and network", scope: FundingScope{MethodID: "method", NetworkID: "network"}, err: errFundingScopeAmbiguous},
		{name: "network and network group", scope: FundingScope{NetworkID: "network", NetworkGroupID: "group"}, networkGroups: true, err: errFundingScopeAmbiguous},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.scope.validate(tc.required, tc.networkGroups), tc.err, "validate should return the expected error")
			assert.Equal(t, tc.empty, tc.scope.isEmpty(), "isEmpty should report whether the scope sets no ID")
		})
	}
}

func TestFundingParams(t *testing.T) {
	t.Parallel()
	assert.Empty(t, fundingParams(""), "fundingParams should query no account without one")
	assert.Equal(t, url.Values{"account_id": {"AA24 N84G HSBB 7ZDO"}}, fundingParams("AA24 N84G HSBB 7ZDO"), "fundingParams should query the account")
}

func TestFundingPageParams(t *testing.T) {
	t.Parallel()
	_, err := fundingPageParams("", 101, 100, false, "")
	assert.ErrorIs(t, err, errInvalidFundingLimit, "fundingPageParams should reject a limit above the maximum")
	_, err = fundingPageParams("cursor", 0, 100, true, "")
	assert.ErrorIs(t, err, errFundingCursorWithFilters, "fundingPageParams should reject a cursor beside filters")
	params, err := fundingPageParams("", 0, 100, true, "")
	require.NoError(t, err, "fundingPageParams must not error for filters alone")
	assert.Empty(t, params, "fundingPageParams should set nothing unasked")
	params, err = fundingPageParams("eyJwYWdlIjoyfQ", 100, 100, false, "AA24 N84G HSBB 7ZDO")
	require.NoError(t, err, "fundingPageParams must not error")
	assert.Equal(t, url.Values{"cursor": {"eyJwYWdlIjoyfQ"}, "limit": {"100"}, "account_id": {"AA24 N84G HSBB 7ZDO"}}, params, "fundingPageParams should set the cursor, limit and account")
}

func TestSetFundingTimes(t *testing.T) {
	t.Parallel()
	params := url.Values{}
	setFundingTimes(params, time.Time{}, time.Time{})
	assert.Empty(t, params, "setFundingTimes should set nothing for unset times")
	aest := time.FixedZone("AEST", 10*60*60)
	setFundingTimes(params, time.Date(2026, 10, 9, 10, 0, 0, 0, aest), time.Date(2026, 10, 10, 10, 0, 0, 500, aest))
	assert.Equal(t, url.Values{"start_time": {"2026-10-09T00:00:00Z"}, "end_time": {"2026-10-10T00:00:00.0000005Z"}}, params, "setFundingTimes should set the times in UTC")
}

func TestFundingLimitsRequest(t *testing.T) {
	t.Parallel()
	usdc := FundingAsset{Class: "currency", Name: currency.USDC}
	for _, tc := range []struct {
		name string
		req  *FundingLimitsRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "asset class", req: &FundingLimitsRequest{Asset: FundingAsset{Class: "nft", Name: currency.USDC}}, err: errInvalidFundingAssetClass},
		{name: "asset name", req: &FundingLimitsRequest{Asset: FundingAsset{Class: "currency"}}, err: currency.ErrCurrencyCodeEmpty},
		{name: "preferred asset name", req: &FundingLimitsRequest{Asset: usdc, PreferredAsset: FundingAsset{Class: "currency"}}, err: currency.ErrCurrencyCodeEmpty},
	} {
		_, _, err := fundingLimitsRequest("/funding/v1/limits/deposit/", tc.req)
		assert.ErrorIsf(t, err, tc.err, "fundingLimitsRequest should reject a request by its %s", tc.name)
	}
	path, params, err := fundingLimitsRequest("/funding/v1/limits/withdrawal/", &FundingLimitsRequest{Asset: FundingAsset{Class: "tokenized_asset", Name: currency.NewCode("aaplx")}, PreferredAsset: FundingAsset{Class: "currency", Name: currency.USD}, AccountID: "AA24 N84G HSBB 7ZDO"})
	require.NoError(t, err, "fundingLimitsRequest must not error")
	assert.Equal(t, "/funding/v1/limits/withdrawal/tokenized_asset/AAPLX", path, "fundingLimitsRequest should put the asset in the path")
	assert.Equal(t, url.Values{"account_id": {"AA24 N84G HSBB 7ZDO"}, "preferred_asset[class]": {"currency"}, "preferred_asset[name]": {"USD"}}, params, "fundingLimitsRequest should query the account and preferred asset")
}

func TestFundingAmountBody(t *testing.T) {
	t.Parallel()
	exp := map[string]any{"asset": map[string]string{"class": "currency", "name": "USDC"}, "amount": "12.5"}
	assert.Equal(t, exp, fundingAmountBody(FundingAsset{Class: "currency", Name: currency.NewCode("usdc")}, 12.5), "fundingAmountBody should send the upper case asset and amount")
}

func TestFundingAssetSetQuery(t *testing.T) {
	t.Parallel()
	params := url.Values{}
	(&FundingAsset{}).setQuery(params, "asset")
	assert.Empty(t, params, "setQuery should set nothing for an empty asset")
	(&FundingAsset{Class: "currency", Name: currency.NewCode("usdc")}).setQuery(params, "asset")
	assert.Equal(t, url.Values{"asset[class]": {"currency"}, "asset[name]": {"USDC"}}, params, "setQuery should nest the class and upper case name under the key")
}

func TestFundingScopeSetQuery(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		scope FundingScope
		exp   url.Values
	}{
		{scope: FundingScope{}, exp: url.Values{}},
		{scope: FundingScope{MethodID: "method", NetworkID: "network"}, exp: url.Values{"scope[method_id]": {"method"}}},
		{scope: FundingScope{NetworkID: "network", NetworkGroupID: "group"}, exp: url.Values{"scope[network_id]": {"network"}}},
		{scope: FundingScope{NetworkGroupID: "group"}, exp: url.Values{"scope[network_group_id]": {"group"}}},
	} {
		params := url.Values{}
		tc.scope.setQuery(params)
		assert.Equalf(t, tc.exp, params, "setQuery should set the first of %+v's IDs", tc.scope)
	}
}
