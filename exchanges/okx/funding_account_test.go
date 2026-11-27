package okx

import (
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// newFundingRequestExchange returns a mock exchange whose handler asserts each request's method, path, query and JSON
// body, then answers with data
func newFundingRequestExchange(t *testing.T, method, path, query, body string, data any) *Exchange {
	t.Helper()
	return newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, method, r.Method, "request method should match")
		assert.Equal(t, path, r.URL.Path, "request path should match")
		wantQuery, err := url.ParseQuery(query)
		if assert.NoError(t, err, "the expected query should parse") {
			assert.Equal(t, wantQuery, r.URL.Query(), "request query should match")
		}
		gotBody, err := io.ReadAll(r.Body)
		if assert.NoError(t, err, "reading the request body should not error") {
			if body == "" {
				assert.Empty(t, gotBody, "request body should be empty")
			} else {
				assert.JSONEq(t, body, string(gotBody), "request body should match")
			}
		}
		writeOKXData(t, w, data)
	}))
}

func TestGetFundingCurrencies(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingCurrencies(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundingCurrencies must reject a nil request")
	_, err = e.GetFundingCurrencies(t.Context(), &FundingCurrenciesRequest{Currencies: currency.Currencies{currency.BTC, currency.EMPTYCODE}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFundingCurrencies must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFundingCurrencies(contextGenerate(), &FundingCurrenciesRequest{Currencies: currency.Currencies{currency.BTC, currency.USDT}})
	require.NoError(t, err, "GetFundingCurrencies must not error")
	if mockTests {
		exp := []FundingCurrencyResponse{
			{
				Currency:                             currency.BTC,
				Name:                                 "Bitcoin",
				LogoLink:                             "https://static.coinall.ltd/cdn/oksupport/asset/currency/icon/btc20230419112752.png",
				Chain:                                "BTC-Bitcoin",
				CanDeposit:                           true,
				CanWithdraw:                          true,
				CanInternalTransfer:                  true,
				MinimumDeposit:                       0.0005,
				MinimumWithdrawal:                    0.0005,
				MinimumInternalTransfer:              0.0001,
				MaximumWithdrawal:                    500,
				WithdrawalTickSize:                   8,
				WithdrawalQuota:                      10000000,
				Fee:                                  0.00005,
				MinimumFee:                           0.00005,
				MaximumFee:                           0.00005,
				MainNet:                              true,
				MinimumDepositArrivalConfirmations:   1,
				MinimumWithdrawalUnlockConfirmations: 2,
			},
			{
				Currency:                             currency.USDT,
				Name:                                 "Tether",
				LogoLink:                             "https://static.coinall.ltd/cdn/oksupport/asset/currency/icon/usdt.png",
				Chain:                                "USDT-TON",
				ContractAddress:                      "EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs",
				DepositEstimatedOpenTime:             types.Time(time.UnixMilli(1714564800000)),
				WithdrawalEstimatedOpenTime:          types.Time(time.UnixMilli(1714568400000)),
				MinimumDeposit:                       0.00000001,
				MinimumWithdrawal:                    2,
				MinimumInternalTransfer:              0.01,
				MaximumWithdrawal:                    1000000,
				WithdrawalTickSize:                   6,
				WithdrawalQuota:                      10000000,
				UsedWithdrawalQuota:                  150,
				Fee:                                  1,
				MinimumFee:                           1,
				MaximumFee:                           1,
				MinimumFeeForContractAddress:         1.5,
				MaximumFeeForContractAddress:         2,
				BurningFeeRate:                       0.05,
				NeedTag:                              true,
				MinimumDepositArrivalConfirmations:   3,
				MinimumWithdrawalUnlockConfirmations: 6,
				DepositQuotaFixed:                    10000,
				UsedDepositQuotaFixed:                2500,
				DepositQuotaDailyLayer2:              50000,
			},
		}
		assert.Equal(t, exp, result, "GetFundingCurrencies should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetFundingCurrencies should return currencies")
	}
}

func TestGetFundingBalances(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *FundingBalancesRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty currency", req: &FundingBalancesRequest{Currencies: currency.Currencies{currency.EMPTYCODE}}, err: currency.ErrCurrencyCodeEmpty},
		{name: "more than 20 currencies", req: &FundingBalancesRequest{Currencies: slices.Repeat(currency.Currencies{currency.BTC}, 21)}, err: errExceedLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetFundingBalances(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetFundingBalances should reject the request")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFundingBalances(contextGenerate(), &FundingBalancesRequest{Currencies: currency.Currencies{currency.BTC, currency.ETH}})
	require.NoError(t, err, "GetFundingBalances must not error")
	if mockTests {
		exp := []FundingBalanceResponse{
			{Currency: currency.BTC, Balance: 0.6, FrozenBalance: 0.1, AvailableBalance: 0.5},
			{Currency: currency.ETH, Balance: 37.11827078, AvailableBalance: 37.11827078},
		}
		assert.Equal(t, exp, result, "GetFundingBalances should decode every field")
	} else {
		assert.NotNil(t, result, "GetFundingBalances should return balances")
	}
}

func TestGetNonTradableAssets(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *NonTradableAssetsRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty currency", req: &NonTradableAssetsRequest{Currencies: currency.Currencies{currency.EMPTYCODE}}, err: currency.ErrCurrencyCodeEmpty},
		{name: "more than 20 currencies", req: &NonTradableAssetsRequest{Currencies: slices.Repeat(currency.Currencies{currency.BTC}, 21)}, err: errExceedLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetNonTradableAssets(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetNonTradableAssets should reject the request")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetNonTradableAssets(contextGenerate(), &NonTradableAssetsRequest{Currencies: currency.Currencies{currency.CELT, currency.MEME}})
	require.NoError(t, err, "GetNonTradableAssets must not error")
	if mockTests {
		exp := []NonTradableAssetResponse{
			{
				Currency:           currency.CELT,
				LogoLink:           "https://static.coinall.ltd/cdn/assets/imgs/221/460DA8A592400393.png",
				Balance:            989.84719571,
				CanWithdraw:        true,
				Chain:              "CELT-OKTC",
				MinimumWithdrawal:  0.1,
				Fee:                2,
				FeeCurrency:        currency.USDT,
				BurningFeeRate:     0.05,
				ContractAddress:    "f403fb",
				WithdrawalTickSize: 8,
				NeedTag:            true,
			},
			{
				Currency:           currency.MEME,
				Name:               "MEME Inu",
				LogoLink:           "https://static.coinall.ltd/cdn/assets/imgs/207/2E664E470103C613.png",
				Balance:            0.001,
				CanWithdraw:        true,
				Chain:              "MEME-ERC20",
				MinimumWithdrawal:  0.001,
				WithdrawAll:        true,
				Fee:                5,
				FeeCurrency:        currency.USDT,
				ContractAddress:    "09b760",
				WithdrawalTickSize: 8,
			},
		}
		assert.Equal(t, exp, result, "GetNonTradableAssets should decode every field")
	} else {
		assert.NotNil(t, result, "GetNonTradableAssets should return assets")
	}
}

func TestGetAccountAssetValuation(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountAssetValuation(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccountAssetValuation must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountAssetValuation(contextGenerate(), &AccountAssetValuationRequest{Currency: currency.USDT})
	require.NoError(t, err, "GetAccountAssetValuation must not error")
	if mockTests {
		exp := &AccountAssetValuationResponse{
			TotalBalance: 3790.09,
			Timestamp:    types.Time(time.UnixMilli(1637566660769)),
			Details:      AssetValuationDetails{Funding: 0.09, Trading: 2544.28, Classic: 124.6, Earn: 1122.73},
		}
		assert.Equal(t, exp, result, "GetAccountAssetValuation should decode every field")
	} else {
		assert.NotZero(t, result.Timestamp, "GetAccountAssetValuation should return a valuation time")
	}
}

func TestFundingTransfer(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *FundingTransferRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty request", req: &FundingTransferRequest{}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing currency", req: &FundingTransferRequest{Amount: 12, RemittingAccountType: "18", BeneficiaryAccountType: "6"}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing amount", req: &FundingTransferRequest{Currency: currency.BTC, RemittingAccountType: "18", BeneficiaryAccountType: "6"}, err: limits.ErrAmountBelowMin},
		{name: "missing remitting account", req: &FundingTransferRequest{Currency: currency.BTC, Amount: 12, BeneficiaryAccountType: "6"}, err: errRemittingAccountTypeRequired},
		{name: "missing beneficiary account", req: &FundingTransferRequest{Currency: currency.BTC, Amount: 12, RemittingAccountType: "18"}, err: errBeneficiaryAccountTypeRequired},
		{name: "sub-account transfer without its name", req: &FundingTransferRequest{TransferType: "1", Currency: currency.BTC, Amount: 12, RemittingAccountType: "6", BeneficiaryAccountType: "6"}, err: errSubAccountNameRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.FundingTransfer(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "FundingTransfer should reject the request")
		})
	}

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *FundingTransferRequest
			body string
		}{
			{
				name: "required parameters only",
				req:  &FundingTransferRequest{Currency: currency.USDT, Amount: 1.5, RemittingAccountType: "6", BeneficiaryAccountType: "18"},
				body: `{"ccy":"USDT","amt":"1.5","from":"6","to":"18"}`,
			},
			{
				name: "every parameter",
				req: &FundingTransferRequest{
					TransferType: "1", Currency: currency.USDT, Amount: 1.5, RemittingAccountType: "6", BeneficiaryAccountType: "6",
					SubAccountName: "mini", LoanTransfer: true, OmitPositionRisk: true, ClientID: "transfer1",
				},
				body: `{"type":"1","ccy":"USDT","amt":"1.5","from":"6","to":"6","subAcct":"mini","loanTrans":true,"omitPosRisk":"true","clientId":"transfer1"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFundingRequestExchange(t, http.MethodPost, "/asset/transfer", "", tc.body, []map[string]string{{"transId": "754147"}})
				_, err := ex.FundingTransfer(t.Context(), tc.req)
				require.NoError(t, err, "FundingTransfer must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.FundingTransfer(contextGenerate(), &FundingTransferRequest{
		TransferType:           "1",
		Currency:               currency.USDT,
		Amount:                 1.5,
		RemittingAccountType:   "6",
		BeneficiaryAccountType: "6",
		SubAccountName:         "mini",
		LoanTransfer:           true,
		OmitPositionRisk:       true,
		ClientID:               "transfer1",
	})
	require.NoError(t, err, "FundingTransfer must not error")
	if mockTests {
		exp := &FundingTransferResponse{
			TransferID:             "754147",
			ClientID:               "transfer1",
			Currency:               currency.USDT,
			RemittingAccountType:   "6",
			Amount:                 1.5,
			BeneficiaryAccountType: "6",
		}
		assert.Equal(t, exp, result, "FundingTransfer should decode every field")
	} else {
		assert.NotEmpty(t, result.TransferID, "FundingTransfer should return a transfer ID")
	}
}

func TestGetFundsTransferState(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundsTransferState(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundsTransferState must reject a nil request")
	_, err = e.GetFundsTransferState(t.Context(), &FundsTransferStateRequest{TransferType: "1"})
	require.ErrorIs(t, err, errTransferIDOrClientIDRequired, "GetFundsTransferState must reject a request without an ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFundsTransferState(contextGenerate(), &FundsTransferStateRequest{TransferID: "1", ClientID: "transfer1", TransferType: "1"})
	require.NoError(t, err, "GetFundsTransferState must not error")
	if mockTests {
		exp := []FundsTransferStateResponse{{
			TransferID:              "1",
			ClientID:                "transfer1",
			Currency:                currency.USDT,
			Amount:                  1.5,
			TransferType:            "1",
			RemittingAccountType:    "18",
			BeneficiaryAccountType:  "6",
			SubAccountName:          "test",
			InstrumentID:            "BTC-USDT",
			BeneficiaryInstrumentID: "BTC-USDT-SWAP",
			State:                   "success",
		}}
		assert.Equal(t, exp, result, "GetFundsTransferState should decode every field")
	} else {
		assert.NotNil(t, result, "GetFundsTransferState should return the transfer state")
	}
}

func TestGetAssetBillsDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetAssetBillsDetails(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAssetBillsDetails must reject a nil request")
	_, err = e.GetAssetBillsDetails(t.Context(), &AssetBillsDetailsRequest{StartTime: time.UnixMilli(1597100000000), EndTime: time.UnixMilli(1597000000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAssetBillsDetails must reject a start after the end")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name  string
			req   *AssetBillsDetailsRequest
			query string
		}{
			{name: "no filters", req: &AssetBillsDetailsRequest{}},
			{
				name: "every filter",
				req: &AssetBillsDetailsRequest{
					Currency: currency.BTC, BillType: "284", ThirdPartyType: "2", ClientID: "transfer1",
					StartTime: time.UnixMilli(1597000000000), EndTime: time.UnixMilli(1597100000000), Limit: 5,
				},
				query: "ccy=BTC&type=284&thirdPartyType=2&clientId=transfer1&after=1597100000000&before=1597000000000&limit=5",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFundingRequestExchange(t, http.MethodGet, "/asset/bills", tc.query, "", []any{})
				_, err := ex.GetAssetBillsDetails(t.Context(), tc.req)
				require.NoError(t, err, "GetAssetBillsDetails must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAssetBillsDetails(contextGenerate(), &AssetBillsDetailsRequest{
		Currency:  currency.BTC,
		BillType:  "20",
		ClientID:  "transfer1",
		StartTime: time.UnixMilli(1597000000000),
		EndTime:   time.UnixMilli(1597100000000),
		Limit:     1,
	})
	require.NoError(t, err, "GetAssetBillsDetails must not error")
	if mockTests {
		exp := []AssetBillResponse{{
			BillID:        "12344",
			Currency:      currency.BTC,
			ClientID:      "transfer1",
			BalanceChange: -2,
			Balance:       10,
			BillType:      "20",
			Notes:         "To sub-account mini",
			Timestamp:     types.Time(time.UnixMilli(1597026383085)),
		}}
		assert.Equal(t, exp, result, "GetAssetBillsDetails should decode every field")
	} else {
		assert.NotNil(t, result, "GetAssetBillsDetails should return bills")
	}
}

func TestGetAssetBillsHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *AssetBillsHistoryRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "after bill ID with end time", req: &AssetBillsHistoryRequest{AfterBillID: "12345", EndTime: time.UnixMilli(1597100000000)}, err: errAfterBillIDAndEndTimeSet},
		{name: "start after end", req: &AssetBillsHistoryRequest{StartTime: time.UnixMilli(1597100000000), EndTime: time.UnixMilli(1597000000000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetAssetBillsHistory(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetAssetBillsHistory should reject the request")
		})
	}

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name  string
			req   *AssetBillsHistoryRequest
			query string
		}{
			{name: "no filters", req: &AssetBillsHistoryRequest{}},
			{
				name: "timestamp paging",
				req: &AssetBillsHistoryRequest{
					Currency: currency.BTC, BillType: "284", ThirdPartyType: "5", ClientID: "transfer1",
					StartTime: time.UnixMilli(1597000000000), EndTime: time.UnixMilli(1597100000000), Limit: 5,
				},
				query: "ccy=BTC&type=284&thirdPartyType=5&clientId=transfer1&after=1597100000000&before=1597000000000&limit=5",
			},
			{
				name:  "bill ID paging",
				req:   &AssetBillsHistoryRequest{AfterBillID: "12345", StartTime: time.UnixMilli(1597000000000)},
				query: "after=12345&pagingType=2&before=1597000000000",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFundingRequestExchange(t, http.MethodGet, "/asset/bills-history", tc.query, "", []any{})
				_, err := ex.GetAssetBillsHistory(t.Context(), tc.req)
				require.NoError(t, err, "GetAssetBillsHistory must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAssetBillsHistory(contextGenerate(), &AssetBillsHistoryRequest{Currency: currency.BTC, BillType: "1", AfterBillID: "12345", Limit: 1})
	require.NoError(t, err, "GetAssetBillsHistory must not error")
	if mockTests {
		exp := []AssetBillResponse{{
			BillID:        "12344",
			Currency:      currency.BTC,
			BalanceChange: 2,
			Balance:       12,
			BillType:      "1",
			Timestamp:     types.Time(time.UnixMilli(1597026383085)),
		}}
		assert.Equal(t, exp, result, "GetAssetBillsHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetAssetBillsHistory should return bills")
	}
}

func TestGetCurrencyDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyDepositAddress(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCurrencyDepositAddress must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCurrencyDepositAddress(contextGenerate(), currency.BTC)
	require.NoError(t, err, "GetCurrencyDepositAddress must not error")
	if mockTests {
		exp := []DepositAddressResponse{
			{
				Address:                "39XNxK1Ryqgg3Bsyn6HzoqV4Xji25pNkv6",
				Currency:               currency.BTC,
				Chain:                  "BTC-Bitcoin",
				BeneficiaryAccountType: "6",
				VerifiedName:           "John Corner",
				Selected:               true,
			},
			{
				Address:                "0x66d0edc2e63b6b992381ee668fbcb01f20ae0428",
				Tag:                    "123456",
				Memo:                   "889900",
				PaymentID:              "pmt-1234567890",
				AddressAttachment:      map[string]string{"comment": "123456"},
				Currency:               currency.BTC,
				Chain:                  "BTC-OKC",
				BeneficiaryAccountType: "6",
				VerifiedName:           "John Corner",
			},
			{
				Address:                "0x66d0edc2e63b6b992381ee668fbcb01f20ae0428",
				Currency:               currency.BTC,
				Chain:                  "BTC-ERC20",
				BeneficiaryAccountType: "18",
				VerifiedName:           "John Corner",
				Selected:               true,
				ContractAddress:        "5807cf",
			},
		}
		assert.Equal(t, exp, result, "GetCurrencyDepositAddress should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetCurrencyDepositAddress should return addresses")
	}
}

func TestGetCurrencyDepositHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyDepositHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCurrencyDepositHistory must reject a nil request")
	_, err = e.GetCurrencyDepositHistory(t.Context(), &DepositHistoryRequest{StartTime: time.UnixMilli(1674040000000), EndTime: time.UnixMilli(1674030000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCurrencyDepositHistory must reject a start after the end")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name  string
			req   *DepositHistoryRequest
			query string
		}{
			{name: "no filters", req: &DepositHistoryRequest{}},
			{
				name: "every filter",
				req: &DepositHistoryRequest{
					Currency: currency.USDT, DepositID: "88883333", FromWithdrawalID: "15447422", TransactionID: "fee235b3e812",
					DepositType: "4", State: "2", StartTime: time.UnixMilli(1674030000000), EndTime: time.UnixMilli(1674040000000), Limit: 5,
				},
				query: "ccy=USDT&depId=88883333&fromWdId=15447422&txId=fee235b3e812&type=4&state=2&after=1674040000000&before=1674030000000&limit=5",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFundingRequestExchange(t, http.MethodGet, "/asset/deposit-history", tc.query, "", []any{})
				_, err := ex.GetCurrencyDepositHistory(t.Context(), tc.req)
				require.NoError(t, err, "GetCurrencyDepositHistory must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCurrencyDepositHistory(contextGenerate(), &DepositHistoryRequest{
		State:     "2",
		StartTime: time.UnixMilli(1674030000000),
		EndTime:   time.UnixMilli(1674040000000),
		Limit:     2,
	})
	require.NoError(t, err, "GetCurrencyDepositHistory must not error")
	if mockTests {
		exp := []DepositHistoryResponse{
			{
				Currency:                        currency.USDT,
				Chain:                           "USDT-TRC20",
				Amount:                          1,
				To:                              "TN4hGjVXMzy5jPmK3fCbYd8pVqzXo9b4N1aGizqs",
				TransactionID:                   "fee235b3e8121c0bd9a3f4c62e7a51d9857d36bb0426917f0df1802",
				Timestamp:                       types.Time(time.UnixMilli(1674038705000)),
				State:                           "2",
				DepositID:                       "88883333",
				ActualDepositBlockConfirmations: 2,
			},
			{
				Currency:          currency.NewCode("TSLAX"),
				Chain:             "TSLAX-Solana",
				Amount:            10,
				From:              "156****359",
				AreaCodeFrom:      "86",
				Timestamp:         types.Time(time.UnixMilli(1674038800000)),
				State:             "2",
				DepositID:         "88883334",
				FromWithdrawalID:  "15447422",
				XStockMultiplier:  1.002,
				XStockShareAmount: 10.02,
				XStockShareName:   "TSLA",
			},
		}
		assert.Equal(t, exp, result, "GetCurrencyDepositHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetCurrencyDepositHistory should return deposits")
	}
}

func TestGetLockedTravelRuleDeposits(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLockedTravelRuleDeposits(contextGenerate())
	require.NoError(t, err, "GetLockedTravelRuleDeposits must not error")
	if mockTests {
		exp := []LockedTravelRuleDepositResponse{{
			DepositID:     "98765",
			TransactionID: "0xabc123def456789",
			Address:       "0xdef456abc789012",
			Currency:      currency.USDT,
			Chain:         "ERC20",
			Amount:        100,
			Timestamp:     types.Time(time.UnixMilli(1714564800000)),
		}}
		assert.Equal(t, exp, result, "GetLockedTravelRuleDeposits should decode every field")
	} else {
		assert.NotNil(t, result, "GetLockedTravelRuleDeposits should return a list")
	}
}

func TestUnlockTravelRuleDeposit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *UnlockTravelRuleDepositRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing deposit ID", req: &UnlockTravelRuleDepositRequest{Questionnaire: &TravelRuleQuestionnaire{PersonType: "1", WalletType: "private"}}, err: errDepositIDRequired},
		{name: "missing questionnaire", req: &UnlockTravelRuleDepositRequest{DepositID: "1234567890"}, err: errTravelRuleQuestionnaireRequired},
		{name: "missing person type", req: &UnlockTravelRuleDepositRequest{DepositID: "1234567890", Questionnaire: &TravelRuleQuestionnaire{WalletType: "private"}}, err: errPersonTypeRequired},
		{name: "missing wallet type", req: &UnlockTravelRuleDepositRequest{DepositID: "1234567890", Questionnaire: &TravelRuleQuestionnaire{PersonType: "1"}}, err: errWalletTypeRequired},
		{name: "exchange wallet without exchange ID", req: &UnlockTravelRuleDepositRequest{DepositID: "1234567890", Questionnaire: &TravelRuleQuestionnaire{PersonType: "1", WalletType: "exchange"}}, err: errExchangeIDRequired},
		{name: "unlisted exchange without its name", req: &UnlockTravelRuleDepositRequest{DepositID: "1234567890", Questionnaire: &TravelRuleQuestionnaire{PersonType: "1", WalletType: "exchange", ExchangeID: "0"}}, err: errExchangeNameRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.UnlockTravelRuleDeposit(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "UnlockTravelRuleDeposit should reject the request")
		})
	}

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *UnlockTravelRuleDepositRequest
			body string
		}{
			{
				name: "required parameters only",
				req:  &UnlockTravelRuleDepositRequest{DepositID: "1234567890", Questionnaire: &TravelRuleQuestionnaire{PersonType: "1", WalletType: "private"}},
				body: `{"depId":"1234567890","questionnaire":{"personType":"1","walletType":"private"}}`,
			},
			{
				name: "every parameter",
				req: &UnlockTravelRuleDepositRequest{DepositID: "1234567890", Questionnaire: &TravelRuleQuestionnaire{
					PersonType: "2", WalletType: "exchange", ExchangeID: "0", ExchangeName: "Example Exchange",
					OriginatorFirstName: "Alex", OriginatorLastName: "Chen", OriginatorCompanyName: "Example Pte Ltd",
					OriginatorCountry: "SG", OriginatorGeographicAddress: "123 Main St, Singapore 018956",
					OriginatorDateOfBirth: "1990-01-01", OriginatorPlaceOfBirth: "SG", OriginatorNationality: "SG",
					BrazilClassificationCode: "A1",
				}},
				body: `{"depId":"1234567890","questionnaire":{"personType":"2","walletType":"exchange","exchId":"0","exchName":"Example Exchange","origFirstName":"Alex","origLastName":"Chen","origCompanyName":"Example Pte Ltd","origCountry":"SG","origGeoAddress":"123 Main St, Singapore 018956","origDob":"1990-01-01","origPob":"SG","origNationality":"SG","brazilClassificationCode":"A1"}}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFundingRequestExchange(t, http.MethodPost, "/asset/travel-rule/unlock", "", tc.body, []map[string]bool{{"result": true}})
				_, err := ex.UnlockTravelRuleDeposit(t.Context(), tc.req)
				require.NoError(t, err, "UnlockTravelRuleDeposit must not error")
			})
		}
	})

	t.Run("unlock failed", func(t *testing.T) {
		t.Parallel()
		ex := newFundingRequestExchange(t, http.MethodPost, "/asset/travel-rule/unlock", "", `{"depId":"1234567890","questionnaire":{"personType":"1","walletType":"private"}}`, []any{})
		_, err := ex.UnlockTravelRuleDeposit(t.Context(), &UnlockTravelRuleDepositRequest{DepositID: "1234567890", Questionnaire: &TravelRuleQuestionnaire{PersonType: "1", WalletType: "private"}})
		assert.ErrorIs(t, err, errTravelRuleDepositUnlockFailed, "UnlockTravelRuleDeposit should report an empty result as a failed unlock")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UnlockTravelRuleDeposit(contextGenerate(), &UnlockTravelRuleDepositRequest{
		DepositID: "1234567890",
		Questionnaire: &TravelRuleQuestionnaire{
			PersonType:                  "2",
			WalletType:                  "exchange",
			ExchangeID:                  "0",
			ExchangeName:                "Example Exchange",
			OriginatorFirstName:         "Alex",
			OriginatorLastName:          "Chen",
			OriginatorCountry:           "SG",
			OriginatorGeographicAddress: "123 Main St, Singapore 018956",
			OriginatorDateOfBirth:       "1990-01-01",
			OriginatorPlaceOfBirth:      "SG",
		},
	})
	require.NoError(t, err, "UnlockTravelRuleDeposit must not error")
	if mockTests {
		assert.Equal(t, &UnlockTravelRuleDepositResponse{Result: true}, result, "UnlockTravelRuleDeposit should decode every field")
	} else {
		assert.True(t, result.Result, "UnlockTravelRuleDeposit should accept the unlock")
	}
}

func TestWithdrawal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *WithdrawalRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty request", req: &WithdrawalRequest{}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing currency", req: &WithdrawalRequest{Amount: 0.1, Destination: "4", ToAddress: core.BitcoinDonationAddress}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing amount", req: &WithdrawalRequest{Currency: currency.BTC, Destination: "4", ToAddress: core.BitcoinDonationAddress}, err: limits.ErrAmountBelowMin},
		{name: "missing destination", req: &WithdrawalRequest{Currency: currency.BTC, Amount: 0.1, ToAddress: core.BitcoinDonationAddress}, err: errWithdrawalDestinationRequired},
		{name: "missing address", req: &WithdrawalRequest{Currency: currency.BTC, Amount: 0.1, Destination: "4"}, err: errAddressRequired},
		{name: "recipient without wallet type", req: &WithdrawalRequest{Currency: currency.BTC, Amount: 0.1, Destination: "4", ToAddress: core.BitcoinDonationAddress, RecipientInformation: &WithdrawalRecipientInformation{ReceiverFirstName: "Bruce"}}, err: errWalletTypeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.Withdrawal(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "Withdrawal should reject the request")
		})
	}

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *WithdrawalRequest
			body string
		}{
			{
				name: "required parameters only",
				req:  &WithdrawalRequest{Currency: currency.BTC, Amount: 0.1, Destination: "4", ToAddress: core.BitcoinDonationAddress},
				body: `{"ccy":"BTC","amt":"0.1","dest":"4","toAddr":"` + core.BitcoinDonationAddress + `"}`,
			},
			{
				name: "every parameter",
				req: &WithdrawalRequest{
					Currency: currency.BTC, Amount: 0.1, Destination: "4", ToAddress: core.BitcoinDonationAddress, ToAddressType: "1",
					Chain: "BTC-Bitcoin", AreaCode: "86", ClientID: "withdrawal1",
					RecipientInformation: &WithdrawalRecipientInformation{
						WalletType: "exchange", ExchangeID: "did:ethr:0xfeb4f99829a9acdf52979abee87e83addf22a7e1",
						ReceiverFirstName: "Bruce", ReceiverLastName: "Wayne", ReceiverCountry: "United States",
						ReceiverCountrySubDivision: "California", ReceiverTownName: "San Jose", ReceiverStreetName: "Clementi Avenue 1",
					},
				},
				body: `{"ccy":"BTC","amt":"0.1","dest":"4","toAddr":"` + core.BitcoinDonationAddress + `","toAddrType":"1","chain":"BTC-Bitcoin","areaCode":"86","clientId":"withdrawal1","rcvrInfo":{"walletType":"exchange","exchId":"did:ethr:0xfeb4f99829a9acdf52979abee87e83addf22a7e1","rcvrFirstName":"Bruce","rcvrLastName":"Wayne","rcvrCountry":"United States","rcvrCountrySubDivision":"California","rcvrTownName":"San Jose","rcvrStreetName":"Clementi Avenue 1"}}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFundingRequestExchange(t, http.MethodPost, "/asset/withdrawal", "", tc.body, []map[string]string{{"wdId": "67485"}})
				_, err := ex.Withdrawal(t.Context(), tc.req)
				require.NoError(t, err, "Withdrawal must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.Withdrawal(contextGenerate(), &WithdrawalRequest{
		Currency:      currency.BTC,
		Amount:        0.1,
		Destination:   "4",
		ToAddress:     core.BitcoinDonationAddress,
		ToAddressType: "1",
		Chain:         "BTC-Bitcoin",
		ClientID:      "withdrawal1",
	})
	require.NoError(t, err, "Withdrawal must not error")
	if mockTests {
		exp := &WithdrawalResponse{Currency: currency.BTC, Chain: "BTC-Bitcoin", Amount: 0.1, WithdrawalID: "67485", ClientID: "withdrawal1"}
		assert.Equal(t, exp, result, "Withdrawal should decode every field")
	} else {
		assert.NotEmpty(t, result.WithdrawalID, "Withdrawal should return a withdrawal ID")
	}
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.CancelWithdrawal(t.Context(), "")
	require.ErrorIs(t, err, errWithdrawalIDRequired, "CancelWithdrawal must reject an empty withdrawal ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelWithdrawal(contextGenerate(), "1123456")
	require.NoError(t, err, "CancelWithdrawal must not error")
	if mockTests {
		assert.Equal(t, &CancelWithdrawalResponse{WithdrawalID: "1123456"}, result, "CancelWithdrawal should decode every field")
	} else {
		assert.NotEmpty(t, result.WithdrawalID, "CancelWithdrawal should return the cancelled withdrawal ID")
	}
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawalHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWithdrawalHistory must reject a nil request")
	_, err = e.GetWithdrawalHistory(t.Context(), &WithdrawalHistoryRequest{StartTime: time.UnixMilli(1655260000000), EndTime: time.UnixMilli(1655250000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetWithdrawalHistory must reject a start after the end")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name  string
			req   *WithdrawalHistoryRequest
			query string
		}{
			{name: "no filters", req: &WithdrawalHistoryRequest{}},
			{
				name: "every filter",
				req: &WithdrawalHistoryRequest{
					Currency: currency.ETH, WithdrawalID: "15447421", ClientID: "withdrawal1", TransactionID: "0x35c3f6ed1ba68b360a174d",
					WithdrawalType: "4", State: "2", StartTime: time.UnixMilli(1655250000000), EndTime: time.UnixMilli(1655260000000), Limit: 5,
				},
				query: "ccy=ETH&wdId=15447421&clientId=withdrawal1&txId=0x35c3f6ed1ba68b360a174d&type=4&state=2&after=1655260000000&before=1655250000000&limit=5",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFundingRequestExchange(t, http.MethodGet, "/asset/withdrawal-history", tc.query, "", []any{})
				_, err := ex.GetWithdrawalHistory(t.Context(), tc.req)
				require.NoError(t, err, "GetWithdrawalHistory must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWithdrawalHistory(contextGenerate(), &WithdrawalHistoryRequest{
		Currency:  currency.ETH,
		State:     "2",
		StartTime: time.UnixMilli(1655250000000),
		EndTime:   time.UnixMilli(1655260000000),
		Limit:     2,
	})
	require.NoError(t, err, "GetWithdrawalHistory must not error")
	if mockTests {
		exp := []WithdrawalHistoryResponse{
			{
				Currency:          currency.ETH,
				Chain:             "ETH-Ethereum",
				Amount:            0.029809,
				Timestamp:         types.Time(time.UnixMilli(1655251200000)),
				From:              "156****359",
				AreaCodeFrom:      "86",
				To:                "0xa30d1fab9fa5d0e5c7ef2e6f6ae7cf18c7b6c579",
				ToAddressType:     "1",
				Tag:               "123456",
				PaymentID:         "pmt-1234567890",
				Memo:              "889900",
				AddressAttachment: map[string]string{"comment": "123456"},
				TransactionID:     "0x35c3f6ed1ba68b360a174d",
				Fee:               0.007,
				FeeCurrency:       currency.ETH,
				State:             "2",
				WithdrawalID:      "15447421",
				ClientID:          "withdrawal1",
				Note:              "Payout",
			},
			{
				Currency:         currency.ETH,
				Chain:            "ETH-Ethereum",
				NonTradableAsset: true,
				Amount:           1.5,
				Timestamp:        types.Time(time.UnixMilli(1655251300000)),
				From:             "156****359",
				AreaCodeFrom:     "86",
				To:               "139****8888",
				AreaCodeTo:       "86",
				ToAddressType:    "2",
				FeeCurrency:      currency.ETH,
				State:            "2",
				WithdrawalID:     "15447422",
				ClientID:         "withdrawal2",
				Note:             "Internal",
			},
		}
		assert.Equal(t, exp, result, "GetWithdrawalHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetWithdrawalHistory should return withdrawals")
	}
}

func TestGetDepositWithdrawalStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *DepositWithdrawalStatusRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "neither ID", req: &DepositWithdrawalStatusRequest{}, err: errWithdrawalIDOrTransactionIDRequired},
		{name: "both IDs", req: &DepositWithdrawalStatusRequest{WithdrawalID: "1244", TransactionID: "1244"}, err: errWithdrawalIDOrTransactionIDRequired},
		{name: "deposit without currency", req: &DepositWithdrawalStatusRequest{TransactionID: "1244", ToAddress: "1672734730284", Chain: "USDT-ERC20"}, err: currency.ErrCurrencyCodeEmpty},
		{name: "deposit without address", req: &DepositWithdrawalStatusRequest{TransactionID: "1244", Currency: currency.USDT, Chain: "USDT-ERC20"}, err: errAddressRequired},
		{name: "deposit without chain", req: &DepositWithdrawalStatusRequest{TransactionID: "1244", Currency: currency.USDT, ToAddress: "1672734730284"}, err: errChainRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetDepositWithdrawalStatus(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetDepositWithdrawalStatus should reject the request")
		})
	}

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex := newFundingRequestExchange(t, http.MethodGet, "/asset/deposit-withdraw-status", "txId=1244&ccy=USDT&to=1672734730284&chain=USDT-ERC20", "", []any{})
		_, err := ex.GetDepositWithdrawalStatus(t.Context(), &DepositWithdrawalStatusRequest{TransactionID: "1244", Currency: currency.USDT, ToAddress: "1672734730284", Chain: "USDT-ERC20"})
		require.NoError(t, err, "GetDepositWithdrawalStatus must not error for a deposit")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDepositWithdrawalStatus(contextGenerate(), &DepositWithdrawalStatusRequest{WithdrawalID: "200045249"})
	require.NoError(t, err, "GetDepositWithdrawalStatus must not error")
	if mockTests {
		exp := []DepositWithdrawalStatusResponse{{
			EstimatedCompleteTime: EstimatedCompleteTime(time.Date(2023, time.January, 9, 20, 10, 48, 0, estCompleteTimeLocation)),
			State:                 "Pending withdrawal: Wallet is under maintenance, please wait.",
			TransactionID:         "16f3638329xxxxxx42d988f97",
			WithdrawalID:          "200045249",
		}}
		assert.Equal(t, exp, result, "GetDepositWithdrawalStatus should decode every field")
	} else {
		assert.NotNil(t, result, "GetDepositWithdrawalStatus should return the status")
	}
}

func TestGetPublicExchangeList(t *testing.T) {
	t.Parallel()
	result, err := e.GetPublicExchangeList(contextGenerate())
	require.NoError(t, err, "GetPublicExchangeList must not error")
	if mockTests {
		exp := []ExchangeResponse{
			{ExchangeName: "ABC", ExchangeID: "abc"},
			{ExchangeName: "AUSTRALIAN HOTCOIN GLOBAL EXCHANGE PTY LTD", ExchangeID: "australian-hotcoin-global-exchange-pty-ltd"},
			{ExchangeName: "Batonex", ExchangeID: "pointex-llc"},
		}
		assert.Equal(t, exp, result, "GetPublicExchangeList should decode every field")
	} else {
		require.NotEmpty(t, result, "GetPublicExchangeList must return exchanges")
		assert.NotEmpty(t, result[0].ExchangeID, "GetPublicExchangeList should return exchange IDs")
	}
}

func TestApplyForMonthlyStatement(t *testing.T) {
	t.Parallel()
	_, err := e.ApplyForMonthlyStatement(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ApplyForMonthlyStatement must reject a nil request")
	_, err = e.ApplyForMonthlyStatement(t.Context(), &ApplyForMonthlyStatementRequest{Month: 13})
	require.ErrorIs(t, err, errInvalidMonth, "ApplyForMonthlyStatement must reject an invalid month")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *ApplyForMonthlyStatementRequest
			body string
		}{
			{name: "last month by default", req: &ApplyForMonthlyStatementRequest{}, body: `{}`},
			{name: "every parameter", req: &ApplyForMonthlyStatementRequest{Month: time.December}, body: `{"month":"Dec"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFundingRequestExchange(t, http.MethodPost, "/asset/monthly-statement", "", tc.body, []map[string]string{{"ts": "1646892328000"}})
				_, err := ex.ApplyForMonthlyStatement(t.Context(), tc.req)
				require.NoError(t, err, "ApplyForMonthlyStatement must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ApplyForMonthlyStatement(contextGenerate(), &ApplyForMonthlyStatementRequest{Month: time.January})
	require.NoError(t, err, "ApplyForMonthlyStatement must not error")
	if mockTests {
		assert.Equal(t, &ApplyForMonthlyStatementResponse{Timestamp: types.Time(time.UnixMilli(1646892328000))}, result, "ApplyForMonthlyStatement should decode every field")
	} else {
		assert.NotZero(t, result.Timestamp, "ApplyForMonthlyStatement should return the download link time")
	}
}

func TestApplyForMonthlyStatementRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		month time.Month
		body  string
		err   error
	}{
		{name: "last month by default", body: `{}`},
		{name: "January", month: time.January, body: `{"month":"Jan"}`},
		{name: "September", month: time.September, body: `{"month":"Sep"}`},
		{name: "invalid month", month: 13, err: errInvalidMonth},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body, err := json.Marshal(&ApplyForMonthlyStatementRequest{Month: tc.month})
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "MarshalJSON should reject the month")
				return
			}
			require.NoError(t, err, "MarshalJSON must not error")
			assert.JSONEq(t, tc.body, string(body), "MarshalJSON should send the month's abbreviation")
		})
	}
}

func TestGetMonthlyStatement(t *testing.T) {
	t.Parallel()
	_, err := e.GetMonthlyStatement(t.Context(), 0)
	require.ErrorIs(t, err, errMonthRequired, "GetMonthlyStatement must reject a missing month")
	_, err = e.GetMonthlyStatement(t.Context(), 13)
	require.ErrorIs(t, err, errInvalidMonth, "GetMonthlyStatement must reject an invalid month")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMonthlyStatement(contextGenerate(), time.January)
	require.NoError(t, err, "GetMonthlyStatement must not error")
	if mockTests {
		exp := []MonthlyStatementResponse{{FileLink: "http://xxx", Timestamp: types.Time(time.UnixMilli(1646892328000)), State: "finished"}}
		assert.Equal(t, exp, result, "GetMonthlyStatement should decode every field")
	} else {
		assert.NotNil(t, result, "GetMonthlyStatement should return the statement")
	}
}

func TestGetConvertCurrencies(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetConvertCurrencies(contextGenerate())
	require.NoError(t, err, "GetConvertCurrencies must not error")
	if mockTests {
		exp := []ConvertCurrencyResponse{
			{Currency: currency.BTC, Minimum: 0.0001, Maximum: 100},
			{Currency: currency.ETH},
		}
		assert.Equal(t, exp, result, "GetConvertCurrencies should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetConvertCurrencies should return currencies")
	}
}

func TestGetConvertCurrencyPair(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *ConvertCurrencyPairRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing from currency", req: &ConvertCurrencyPairRequest{ToCurrency: currency.BTC}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing to currency", req: &ConvertCurrencyPairRequest{FromCurrency: currency.USDT}, err: currency.ErrCurrencyCodeEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetConvertCurrencyPair(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetConvertCurrencyPair should reject the request")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetConvertCurrencyPair(contextGenerate(), &ConvertCurrencyPairRequest{FromCurrency: currency.USDT, ToCurrency: currency.BTC, ConvertMode: "0"})
	require.NoError(t, err, "GetConvertCurrencyPair must not error")
	if mockTests {
		exp := &ConvertCurrencyPairResponse{
			InstrumentID:         "BTC-USDT",
			BaseCurrency:         currency.BTC,
			BaseCurrencyMaximum:  0.5,
			BaseCurrencyMinimum:  0.0001,
			QuoteCurrency:        currency.USDT,
			QuoteCurrencyMaximum: 10000,
			QuoteCurrencyMinimum: 1,
		}
		assert.Equal(t, exp, result, "GetConvertCurrencyPair should decode every field")
	} else {
		assert.NotEmpty(t, result.InstrumentID, "GetConvertCurrencyPair should return the pair")
	}
}

func TestEstimateQuote(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *EstimateQuoteRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty request", req: &EstimateQuoteRequest{}, err: currency.ErrCurrencyCodeEmpty},
		{name: "tag only", req: &EstimateQuoteRequest{Tag: "abcd"}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing quote currency", req: &EstimateQuoteRequest{BaseCurrency: currency.BTC}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing side", req: &EstimateQuoteRequest{BaseCurrency: currency.BTC, QuoteCurrency: currency.USDT}, err: order.ErrSideIsInvalid},
		{name: "invalid side", req: &EstimateQuoteRequest{BaseCurrency: currency.BTC, QuoteCurrency: currency.USDT, Side: "long"}, err: order.ErrSideIsInvalid},
		{name: "missing size", req: &EstimateQuoteRequest{BaseCurrency: currency.BTC, QuoteCurrency: currency.USDT, Side: order.Sell.Lower()}, err: limits.ErrAmountBelowMin},
		{name: "missing size currency", req: &EstimateQuoteRequest{BaseCurrency: currency.BTC, QuoteCurrency: currency.USDT, Side: order.Sell.Lower(), RequestForQuoteSize: 30}, err: currency.ErrCurrencyCodeEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.EstimateQuote(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "EstimateQuote should reject the request")
		})
	}

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *EstimateQuoteRequest
			body string
		}{
			{
				name: "required parameters only",
				req:  &EstimateQuoteRequest{BaseCurrency: currency.ETH, QuoteCurrency: currency.USDT, Side: "SELL", RequestForQuoteSize: 0.5, RequestForQuoteCurrency: currency.ETH},
				body: `{"baseCcy":"ETH","quoteCcy":"USDT","side":"sell","rfqSz":"0.5","rfqSzCcy":"ETH"}`,
			},
			{
				name: "every parameter",
				req: &EstimateQuoteRequest{
					BaseCurrency: currency.ETH, QuoteCurrency: currency.USDT, Side: "buy", RequestForQuoteSize: 30,
					RequestForQuoteCurrency: currency.USDT, ClientQuoteRequestID: "quote1", Tag: "broker1", ConvertMode: "1",
				},
				body: `{"baseCcy":"ETH","quoteCcy":"USDT","side":"buy","rfqSz":"30","rfqSzCcy":"USDT","clQReqId":"quote1","tag":"broker1","convertMode":"1"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFundingRequestExchange(t, http.MethodPost, "/asset/convert/estimate-quote", "", tc.body, []map[string]string{{"quoteId": "quoterETH-USDT16461885104612381", "side": "buy"}})
				_, err := ex.EstimateQuote(t.Context(), tc.req)
				require.NoError(t, err, "EstimateQuote must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.EstimateQuote(contextGenerate(), &EstimateQuoteRequest{
		BaseCurrency:            currency.ETH,
		QuoteCurrency:           currency.USDT,
		Side:                    order.Buy.Lower(),
		RequestForQuoteSize:     30,
		RequestForQuoteCurrency: currency.USDT,
		ClientQuoteRequestID:    "quote1",
		Tag:                     "broker1",
		ConvertMode:             "0",
	})
	require.NoError(t, err, "EstimateQuote must not error")
	if mockTests {
		exp := &EstimateQuoteResponse{
			QuoteTime:                   types.Time(time.UnixMilli(1646188510461)),
			TTLMilliseconds:             10000,
			ClientQuoteRequestID:        "quote1",
			QuoteID:                     "quoterETH-USDT16461885104612381",
			BaseCurrency:                currency.ETH,
			QuoteCurrency:               currency.USDT,
			Side:                        order.Buy,
			OriginalRequestForQuoteSize: 30,
			RequestForQuoteSize:         30,
			RequestForQuoteCurrency:     currency.USDT,
			ConvertPrice:                2932.40104429,
			BaseSize:                    0.01023052,
			QuoteSize:                   30,
		}
		assert.Equal(t, exp, result, "EstimateQuote should decode every field")
	} else {
		assert.NotEmpty(t, result.QuoteID, "EstimateQuote should return a quote ID")
	}
}

func TestConvertTrade(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *ConvertTradeRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty request", req: &ConvertTradeRequest{}, err: currency.ErrCurrencyCodeEmpty},
		{name: "tag only", req: &ConvertTradeRequest{Tag: "123"}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing quote currency", req: &ConvertTradeRequest{BaseCurrency: currency.BTC}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing side", req: &ConvertTradeRequest{BaseCurrency: currency.BTC, QuoteCurrency: currency.USDT}, err: order.ErrSideIsInvalid},
		{name: "missing size", req: &ConvertTradeRequest{BaseCurrency: currency.BTC, QuoteCurrency: currency.USDT, Side: order.Buy.Lower()}, err: limits.ErrAmountBelowMin},
		{name: "missing size currency", req: &ConvertTradeRequest{BaseCurrency: currency.BTC, QuoteCurrency: currency.USDT, Side: order.Buy.Lower(), Size: 2}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing quote ID", req: &ConvertTradeRequest{BaseCurrency: currency.BTC, QuoteCurrency: currency.USDT, Side: order.Buy.Lower(), Size: 2, SizeCurrency: currency.USDT}, err: errQuoteIDRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.ConvertTrade(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "ConvertTrade should reject the request")
		})
	}

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *ConvertTradeRequest
			body string
		}{
			{
				name: "required parameters only",
				req:  &ConvertTradeRequest{QuoteID: "quoterETH-USDT16461885104612381", BaseCurrency: currency.ETH, QuoteCurrency: currency.USDT, Side: "Buy", Size: 30, SizeCurrency: currency.USDT},
				body: `{"quoteId":"quoterETH-USDT16461885104612381","baseCcy":"ETH","quoteCcy":"USDT","side":"buy","sz":"30","szCcy":"USDT"}`,
			},
			{
				name: "every parameter",
				req: &ConvertTradeRequest{
					QuoteID: "quoterETH-USDT16461885104612381", BaseCurrency: currency.ETH, QuoteCurrency: currency.USDT, Side: "buy",
					Size: 30, SizeCurrency: currency.USDT, ClientTradeRequestID: "trade1", Tag: "broker1", ConvertMode: "1",
				},
				body: `{"quoteId":"quoterETH-USDT16461885104612381","baseCcy":"ETH","quoteCcy":"USDT","side":"buy","sz":"30","szCcy":"USDT","clTReqId":"trade1","tag":"broker1","convertMode":"1"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFundingRequestExchange(t, http.MethodPost, "/asset/convert/trade", "", tc.body, []map[string]string{{"tradeId": "trader16461885203381437", "side": "buy"}})
				_, err := ex.ConvertTrade(t.Context(), tc.req)
				require.NoError(t, err, "ConvertTrade must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ConvertTrade(contextGenerate(), &ConvertTradeRequest{
		QuoteID:              "quoterETH-USDT16461885104612381",
		BaseCurrency:         currency.ETH,
		QuoteCurrency:        currency.USDT,
		Side:                 order.Buy.Lower(),
		Size:                 30,
		SizeCurrency:         currency.USDT,
		ClientTradeRequestID: "trade1",
		Tag:                  "broker1",
		ConvertMode:          "0",
	})
	require.NoError(t, err, "ConvertTrade must not error")
	if mockTests {
		exp := &ConvertTradeResponse{
			TradeID:              "trader16461885203381437",
			QuoteID:              "quoterETH-USDT16461885104612381",
			ClientTradeRequestID: "trade1",
			State:                "fullyFilled",
			InstrumentID:         "ETH-USDT",
			BaseCurrency:         currency.ETH,
			QuoteCurrency:        currency.USDT,
			Side:                 order.Buy,
			FillPrice:            2932.40104429,
			FillBaseSize:         0.01023052,
			FillQuoteSize:        30,
			Timestamp:            types.Time(time.UnixMilli(1646188520338)),
		}
		assert.Equal(t, exp, result, "ConvertTrade should decode every field")
	} else {
		assert.NotEmpty(t, result.TradeID, "ConvertTrade should return a trade ID")
	}
}

func TestGetConvertHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetConvertHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetConvertHistory must reject a nil request")
	_, err = e.GetConvertHistory(t.Context(), &ConvertHistoryRequest{StartTime: time.UnixMilli(1646190000000), EndTime: time.UnixMilli(1646180000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetConvertHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetConvertHistory(contextGenerate(), &ConvertHistoryRequest{
		ClientTradeRequestID: "trade1",
		StartTime:            time.UnixMilli(1646180000000),
		EndTime:              time.UnixMilli(1646190000000),
		Limit:                1,
		Tag:                  "broker1",
	})
	require.NoError(t, err, "GetConvertHistory must not error")
	if mockTests {
		exp := []ConvertHistoryResponse{{
			TradeID:              "trader16461885203381437",
			ClientTradeRequestID: "trade1",
			State:                "fullyFilled",
			InstrumentID:         "ETH-USDT",
			BaseCurrency:         currency.ETH,
			QuoteCurrency:        currency.USDT,
			Side:                 order.Buy,
			FillPrice:            2932.401044,
			FillBaseSize:         0.01023052,
			FillQuoteSize:        30,
			Timestamp:            types.Time(time.UnixMilli(1646188520000)),
		}}
		assert.Equal(t, exp, result, "GetConvertHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetConvertHistory should return trades")
	}
}

func TestGetFiatDepositPaymentMethods(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatDepositPaymentMethods(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFiatDepositPaymentMethods must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFiatDepositPaymentMethods(contextGenerate(), currency.TRY)
	require.NoError(t, err, "GetFiatDepositPaymentMethods must not error")
	if mockTests {
		exp := []FiatPaymentMethodResponse{{
			Currency:      currency.TRY,
			PaymentMethod: "TR_BANKS",
			Limits: FiatPaymentMethodLimits{
				DailyLimit:            2147483647,
				DailyLimitRemaining:   2147483647,
				WeeklyLimit:           2147483647,
				WeeklyLimitRemaining:  2147483647,
				MonthlyLimit:          2147483647,
				MonthlyLimitRemaining: 2147483647,
				MaximumAmount:         1000000,
				MinimumAmount:         1,
				LifetimeLimit:         2147483647,
			},
			Accounts: []FiatPaymentAccount{
				{PaymentAccountID: "1", AccountNumber: "TR740001592093703829602611", RecipientName: "John Doe", BankName: "VakıfBank", BankCode: "TVBATR2AXXX", State: "active"},
				{PaymentAccountID: "2", AccountNumber: "TR740001592093703829602622", RecipientName: "John Doe", BankName: "FBHLTRISXXX", State: "active"},
			},
		}}
		assert.Equal(t, exp, result, "GetFiatDepositPaymentMethods should decode every field")
	} else {
		assert.NotNil(t, result, "GetFiatDepositPaymentMethods should return payment methods")
	}
}

func TestGetFiatWithdrawalPaymentMethods(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatWithdrawalPaymentMethods(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFiatWithdrawalPaymentMethods must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFiatWithdrawalPaymentMethods(contextGenerate(), currency.TRY)
	require.NoError(t, err, "GetFiatWithdrawalPaymentMethods must not error")
	if mockTests {
		exp := []FiatPaymentMethodResponse{{
			Currency:      currency.TRY,
			PaymentMethod: "TR_BANKS",
			FeeRate:       0.02,
			MinimumFee:    1,
			Accounts: []FiatPaymentAccount{
				{PaymentAccountID: "1", AccountNumber: "TR740001592093703829602668", RecipientName: "John Doe", BankName: "VakıfBank", BankCode: "TVBATR2AXXX", State: "active"},
				{PaymentAccountID: "2", AccountNumber: "TR740001592093703829603024", RecipientName: "John Doe", BankName: "Şekerbank", BankCode: "SEKETR2AXXX", State: "active"},
			},
		}}
		assert.Equal(t, exp, result, "GetFiatWithdrawalPaymentMethods should decode every field")
	} else {
		assert.NotNil(t, result, "GetFiatWithdrawalPaymentMethods should return payment methods")
	}
}

func TestCreateFiatWithdrawalOrder(t *testing.T) {
	t.Parallel()
	valid := CreateFiatWithdrawalOrderRequest{
		PaymentAccountID: "20",
		Currency:         currency.TRY,
		Amount:           100,
		PaymentMethod:    "TR_BANKS",
		ClientID:         "194a6975e98246538faeb0fab0d502df",
	}
	for _, tc := range []struct {
		name   string
		mutate func(*CreateFiatWithdrawalOrderRequest)
		err    error
	}{
		{name: "missing payment account ID", mutate: func(r *CreateFiatWithdrawalOrderRequest) { r.PaymentAccountID = "" }, err: errPaymentAccountIDRequired},
		{name: "missing currency", mutate: func(r *CreateFiatWithdrawalOrderRequest) { r.Currency = currency.EMPTYCODE }, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing amount", mutate: func(r *CreateFiatWithdrawalOrderRequest) { r.Amount = 0 }, err: limits.ErrAmountBelowMin},
		{name: "missing payment method", mutate: func(r *CreateFiatWithdrawalOrderRequest) { r.PaymentMethod = "" }, err: errPaymentMethodRequired},
		{name: "missing client ID", mutate: func(r *CreateFiatWithdrawalOrderRequest) { r.ClientID = "" }, err: errClientIDRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := valid
			tc.mutate(&req)
			_, err := e.CreateFiatWithdrawalOrder(t.Context(), &req)
			assert.ErrorIs(t, err, tc.err, "CreateFiatWithdrawalOrder should reject the request")
		})
	}
	_, err := e.CreateFiatWithdrawalOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateFiatWithdrawalOrder must reject a nil request")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex := newFundingRequestExchange(t, http.MethodPost, "/fiat/create-withdrawal", "",
			`{"paymentAcctId":"20","ccy":"TRY","amt":"100","paymentMethod":"TR_BANKS","clientId":"194a6975e98246538faeb0fab0d502df"}`,
			[]map[string]string{{"ordId": "124041201450544699"}})
		req := valid
		_, err := ex.CreateFiatWithdrawalOrder(t.Context(), &req)
		require.NoError(t, err, "CreateFiatWithdrawalOrder must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req := valid
	result, err := e.CreateFiatWithdrawalOrder(contextGenerate(), &req)
	require.NoError(t, err, "CreateFiatWithdrawalOrder must not error")
	if mockTests {
		exp := &FiatOrderResponse{
			OrderID:          "124041201450544699",
			ClientID:         "194a6975e98246538faeb0fab0d502df",
			Amount:           100,
			Currency:         currency.TRY,
			Fee:              2,
			PaymentAccountID: "20",
			PaymentMethod:    "TR_BANKS",
			State:            "completed",
			CreationTime:     types.Time(time.UnixMilli(1707429385000)),
			UpdateTime:       types.Time(time.UnixMilli(1707429385000)),
		}
		assert.Equal(t, exp, result, "CreateFiatWithdrawalOrder should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "CreateFiatWithdrawalOrder should return an order ID")
	}
}

func TestCancelFiatWithdrawalOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFiatWithdrawalOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelFiatWithdrawalOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelFiatWithdrawalOrder(contextGenerate(), "124041201450544699")
	require.NoError(t, err, "CancelFiatWithdrawalOrder must not error")
	if mockTests {
		assert.Equal(t, &CancelFiatWithdrawalOrderResponse{OrderID: "124041201450544699", State: "canceled"}, result, "CancelFiatWithdrawalOrder should decode every field")
	} else {
		assert.NotEmpty(t, result.State, "CancelFiatWithdrawalOrder should return the order state")
	}
}

func TestGetFiatWithdrawalOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatWithdrawalOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFiatWithdrawalOrderHistory must reject a nil request")
	_, err = e.GetFiatWithdrawalOrderHistory(t.Context(), &FiatOrderHistoryRequest{StartTime: time.UnixMilli(1707430000000), EndTime: time.UnixMilli(1707429000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFiatWithdrawalOrderHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFiatWithdrawalOrderHistory(contextGenerate(), &FiatOrderHistoryRequest{
		Currency:      currency.TRY,
		PaymentMethod: "TR_BANKS",
		State:         "completed",
		StartTime:     time.UnixMilli(1707429000000),
		EndTime:       time.UnixMilli(1707430000000),
		Limit:         2,
	})
	require.NoError(t, err, "GetFiatWithdrawalOrderHistory must not error")
	if mockTests {
		exp := []FiatOrderResponse{
			{
				OrderID:          "124041201450544699",
				ClientID:         "194a6975e98246538faeb0fab0d502df",
				Amount:           10000,
				Currency:         currency.TRY,
				PaymentAccountID: "20",
				PaymentMethod:    "TR_BANKS",
				State:            "completed",
				CreationTime:     types.Time(time.UnixMilli(1707429385000)),
				UpdateTime:       types.Time(time.UnixMilli(1707429385000)),
			},
			{
				OrderID:          "124041201450544690",
				ClientID:         "164a6975e48946538faeb0fab0d414fg",
				Amount:           5000,
				Currency:         currency.TRY,
				PaymentAccountID: "20",
				PaymentMethod:    "TR_BANKS",
				State:            "completed",
				CreationTime:     types.Time(time.UnixMilli(1707429385000)),
				UpdateTime:       types.Time(time.UnixMilli(1707429385000)),
			},
		}
		assert.Equal(t, exp, result, "GetFiatWithdrawalOrderHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetFiatWithdrawalOrderHistory should return orders")
	}
}

func TestGetFiatWithdrawalOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatWithdrawalOrderDetail(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetFiatWithdrawalOrderDetail must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFiatWithdrawalOrderDetail(contextGenerate(), "024041201450544699")
	require.NoError(t, err, "GetFiatWithdrawalOrderDetail must not error")
	if mockTests {
		exp := &FiatOrderResponse{
			OrderID:          "024041201450544699",
			ClientID:         "194a6975e98246538faeb0fab0d502df",
			Amount:           100,
			Currency:         currency.TRY,
			PaymentAccountID: "20",
			PaymentMethod:    "TR_BANKS",
			State:            "completed",
			CreationTime:     types.Time(time.UnixMilli(1707429385000)),
			UpdateTime:       types.Time(time.UnixMilli(1707429385000)),
		}
		assert.Equal(t, exp, result, "GetFiatWithdrawalOrderDetail should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "GetFiatWithdrawalOrderDetail should return the order")
	}
}

func TestGetFiatDepositOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatDepositOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFiatDepositOrderHistory must reject a nil request")
	_, err = e.GetFiatDepositOrderHistory(t.Context(), &FiatOrderHistoryRequest{StartTime: time.UnixMilli(1707430000000), EndTime: time.UnixMilli(1707429000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFiatDepositOrderHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFiatDepositOrderHistory(contextGenerate(), &FiatOrderHistoryRequest{
		Currency:      currency.TRY,
		PaymentMethod: "TR_BANKS",
		State:         "completed",
		StartTime:     time.UnixMilli(1707429000000),
		EndTime:       time.UnixMilli(1707430000000),
		Limit:         2,
	})
	require.NoError(t, err, "GetFiatDepositOrderHistory must not error")
	if mockTests {
		exp := []FiatOrderResponse{
			{
				OrderID:          "024041201450544699",
				Amount:           10000,
				Currency:         currency.TRY,
				PaymentAccountID: "20",
				PaymentMethod:    "TR_BANKS",
				State:            "completed",
				CreationTime:     types.Time(time.UnixMilli(1707429385000)),
				UpdateTime:       types.Time(time.UnixMilli(1707429385000)),
			},
			{
				OrderID:          "024041201450544690",
				Amount:           50000,
				Currency:         currency.TRY,
				PaymentAccountID: "20",
				PaymentMethod:    "TR_BANKS",
				State:            "completed",
				CreationTime:     types.Time(time.UnixMilli(1707429385000)),
				UpdateTime:       types.Time(time.UnixMilli(1707429385000)),
			},
		}
		assert.Equal(t, exp, result, "GetFiatDepositOrderHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetFiatDepositOrderHistory should return orders")
	}
}

func TestGetFiatDepositOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatDepositOrderDetail(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetFiatDepositOrderDetail must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFiatDepositOrderDetail(contextGenerate(), "024041201450544699")
	require.NoError(t, err, "GetFiatDepositOrderDetail must not error")
	if mockTests {
		exp := &FiatOrderResponse{
			OrderID:          "024041201450544699",
			Amount:           100,
			Currency:         currency.TRY,
			PaymentAccountID: "20",
			PaymentMethod:    "TR_BANKS",
			State:            "completed",
			CreationTime:     types.Time(time.UnixMilli(1707429385000)),
			UpdateTime:       types.Time(time.UnixMilli(1707429385000)),
		}
		assert.Equal(t, exp, result, "GetFiatDepositOrderDetail should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "GetFiatDepositOrderDetail should return the order")
	}
}

func TestGetBuySellCurrencies(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBuySellCurrencies(contextGenerate())
	require.NoError(t, err, "GetBuySellCurrencies must not error")
	if mockTests {
		exp := &BuySellCurrenciesResponse{
			FiatCurrencies:   []BuySellCurrency{{Currency: currency.USD}, {Currency: currency.EUR}},
			CryptoCurrencies: []BuySellCurrency{{Currency: currency.BTC}},
		}
		assert.Equal(t, exp, result, "GetBuySellCurrencies should decode every field")
	} else {
		assert.NotEmpty(t, result.CryptoCurrencies, "GetBuySellCurrencies should return crypto currencies")
	}
}

func TestGetBuySellCurrencyPair(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *BuySellCurrencyPairRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing from currency", req: &BuySellCurrencyPairRequest{ToCurrency: currency.BTC}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing to currency", req: &BuySellCurrencyPairRequest{FromCurrency: currency.USD}, err: currency.ErrCurrencyCodeEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetBuySellCurrencyPair(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetBuySellCurrencyPair should reject the request")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBuySellCurrencyPair(contextGenerate(), &BuySellCurrencyPairRequest{FromCurrency: currency.USD, ToCurrency: currency.BTC})
	require.NoError(t, err, "GetBuySellCurrencyPair must not error")
	if mockTests {
		exp := &BuySellCurrencyPairResponse{
			Side:                          "buy",
			FromCurrency:                  currency.USD,
			ToCurrency:                    currency.BTC,
			SingleTradeMaximum:            1,
			SingleTradeMinimum:            0.01,
			FixedPriceDailyLimit:          100000,
			FixedPriceRemainingDailyQuota: 75000,
			PaymentMethods:                []string{"balance"},
		}
		assert.Equal(t, exp, result, "GetBuySellCurrencyPair should decode every field")
	} else {
		assert.NotEmpty(t, result.Side, "GetBuySellCurrencyPair should return the pair's side")
	}
}

func TestGetBuySellQuote(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *BuySellQuoteRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing side", req: &BuySellQuoteRequest{FromCurrency: currency.USD, ToCurrency: currency.BTC, RequestForQuoteAmount: 30, RequestForQuoteCurrency: currency.USD}, err: order.ErrSideIsInvalid},
		{name: "missing from currency", req: &BuySellQuoteRequest{Side: "buy", ToCurrency: currency.BTC, RequestForQuoteAmount: 30, RequestForQuoteCurrency: currency.USD}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing to currency", req: &BuySellQuoteRequest{Side: "buy", FromCurrency: currency.USD, RequestForQuoteAmount: 30, RequestForQuoteCurrency: currency.USD}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing amount", req: &BuySellQuoteRequest{Side: "buy", FromCurrency: currency.USD, ToCurrency: currency.BTC, RequestForQuoteCurrency: currency.USD}, err: limits.ErrAmountBelowMin},
		{name: "missing request for quote currency", req: &BuySellQuoteRequest{Side: "buy", FromCurrency: currency.USD, ToCurrency: currency.BTC, RequestForQuoteAmount: 30}, err: currency.ErrCurrencyCodeEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetBuySellQuote(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetBuySellQuote should reject the request")
		})
	}

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex := newFundingRequestExchange(t, http.MethodPost, "/fiat/buy-sell/quote", "", `{"side":"sell","fromCcy":"BTC","toCcy":"USD","rfqAmt":"30","rfqCcy":"USD"}`, []map[string]string{{"quoteId": "quoterBTC-USD16461885104612381", "side": "sell"}})
		_, err := ex.GetBuySellQuote(t.Context(), &BuySellQuoteRequest{Side: "SELL", FromCurrency: currency.BTC, ToCurrency: currency.USD, RequestForQuoteAmount: 30, RequestForQuoteCurrency: currency.USD})
		require.NoError(t, err, "GetBuySellQuote must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.GetBuySellQuote(contextGenerate(), &BuySellQuoteRequest{
		Side:                    order.Buy.Lower(),
		FromCurrency:            currency.USD,
		ToCurrency:              currency.BTC,
		RequestForQuoteAmount:   30,
		RequestForQuoteCurrency: currency.USD,
	})
	require.NoError(t, err, "GetBuySellQuote must not error")
	if mockTests {
		exp := &BuySellQuoteResponse{
			QuoteID:                 "quoterBTC-USD16461885104612381",
			Side:                    order.Buy,
			FromCurrency:            currency.USD,
			ToCurrency:              currency.BTC,
			RequestForQuoteAmount:   30,
			RequestForQuoteCurrency: currency.USD,
			QuotePrice:              2932.40104429,
			QuoteCurrency:           currency.USD,
			QuoteFromAmount:         30,
			QuoteToAmount:           30,
			QuoteTime:               types.Time(time.UnixMilli(1646188510461)),
			TTLMilliseconds:         10000,
		}
		assert.Equal(t, exp, result, "GetBuySellQuote should decode every field")
	} else {
		assert.NotEmpty(t, result.QuoteID, "GetBuySellQuote should return a quote ID")
	}
}

func TestPlaceBuySellTrade(t *testing.T) {
	t.Parallel()
	valid := BuySellTradeRequest{
		QuoteID:                 "quoterBTC-USD16461885104612381",
		Side:                    order.Buy.Lower(),
		FromCurrency:            currency.USD,
		ToCurrency:              currency.BTC,
		RequestForQuoteAmount:   30,
		RequestForQuoteCurrency: currency.USD,
		PaymentMethod:           "balance",
		ClientOrderID:           "123456",
	}
	for _, tc := range []struct {
		name   string
		mutate func(*BuySellTradeRequest)
		err    error
	}{
		{name: "missing quote ID", mutate: func(r *BuySellTradeRequest) { r.QuoteID = "" }, err: errQuoteIDRequired},
		{name: "invalid side", mutate: func(r *BuySellTradeRequest) { r.Side = "short" }, err: order.ErrSideIsInvalid},
		{name: "missing from currency", mutate: func(r *BuySellTradeRequest) { r.FromCurrency = currency.EMPTYCODE }, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing to currency", mutate: func(r *BuySellTradeRequest) { r.ToCurrency = currency.EMPTYCODE }, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing amount", mutate: func(r *BuySellTradeRequest) { r.RequestForQuoteAmount = 0 }, err: limits.ErrAmountBelowMin},
		{name: "missing request for quote currency", mutate: func(r *BuySellTradeRequest) { r.RequestForQuoteCurrency = currency.EMPTYCODE }, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing payment method", mutate: func(r *BuySellTradeRequest) { r.PaymentMethod = "" }, err: errPaymentMethodRequired},
		{name: "missing client order ID", mutate: func(r *BuySellTradeRequest) { r.ClientOrderID = "" }, err: order.ErrClientOrderIDMustBeSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := valid
			tc.mutate(&req)
			_, err := e.PlaceBuySellTrade(t.Context(), &req)
			assert.ErrorIs(t, err, tc.err, "PlaceBuySellTrade should reject the request")
		})
	}
	_, err := e.PlaceBuySellTrade(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceBuySellTrade must reject a nil request")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex := newFundingRequestExchange(t, http.MethodPost, "/fiat/buy-sell/trade", "",
			`{"quoteId":"quoterBTC-USD16461885104612381","side":"buy","fromCcy":"USD","toCcy":"BTC","rfqAmt":"30","rfqCcy":"USD","paymentMethod":"balance","clOrdId":"123456"}`,
			[]map[string]string{{"ordId": "1234", "side": "buy"}})
		req := valid
		req.Side = "BUY"
		_, err := ex.PlaceBuySellTrade(t.Context(), &req)
		require.NoError(t, err, "PlaceBuySellTrade must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req := valid
	result, err := e.PlaceBuySellTrade(contextGenerate(), &req)
	require.NoError(t, err, "PlaceBuySellTrade must not error")
	if mockTests {
		exp := &BuySellTradeResponse{
			OrderID:                 "1234",
			ClientOrderID:           "123456",
			QuoteID:                 "quoterBTC-USD16461885104612381",
			State:                   "completed",
			Side:                    order.Buy,
			FromCurrency:            currency.USD,
			ToCurrency:              currency.BTC,
			RequestForQuoteAmount:   30,
			RequestForQuoteCurrency: currency.USD,
			FillPrice:               2932.40104429,
			FillQuoteCurrency:       currency.USD,
			FillFromAmount:          30,
			FillToAmount:            0.01,
			CreationTime:            types.Time(time.UnixMilli(1646188510461)),
		}
		assert.Equal(t, exp, result, "PlaceBuySellTrade should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "PlaceBuySellTrade should return an order ID")
	}
}

func TestGetBuySellTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetBuySellTradeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBuySellTradeHistory must reject a nil request")
	_, err = e.GetBuySellTradeHistory(t.Context(), &BuySellTradeHistoryRequest{StartTime: time.UnixMilli(1646189000000), EndTime: time.UnixMilli(1646188000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBuySellTradeHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBuySellTradeHistory(contextGenerate(), &BuySellTradeHistoryRequest{
		OrderID:       "1234",
		ClientOrderID: "123456",
		State:         "completed",
		StartTime:     time.UnixMilli(1646188000000),
		EndTime:       time.UnixMilli(1646189000000),
		Limit:         1,
	})
	require.NoError(t, err, "GetBuySellTradeHistory must not error")
	if mockTests {
		exp := []BuySellTradeHistoryResponse{{
			OrderID:                 "1234",
			ClientOrderID:           "123456",
			QuoteID:                 "quoterBTC-USD16461885104612381",
			State:                   "completed",
			Side:                    order.Buy,
			FromCurrency:            currency.USD,
			ToCurrency:              currency.BTC,
			RequestForQuoteAmount:   30,
			RequestForQuoteCurrency: currency.USD,
			FillPrice:               2932.40104429,
			FillQuoteCurrency:       currency.USD,
			FillFromAmount:          30,
			FillToAmount:            0.01,
			CreationTime:            types.Time(time.UnixMilli(1646188510461)),
			UpdateTime:              types.Time(time.UnixMilli(1646188510461)),
		}}
		assert.Equal(t, exp, result, "GetBuySellTradeHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetBuySellTradeHistory should return trades")
	}
}

func TestFormatFundingCurrencies(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		currencies currency.Currencies
		maximum    int
		exp        string
		err        error
	}{
		{name: "no currencies"},
		{name: "one currency", currencies: currency.Currencies{currency.BTC}, maximum: 20, exp: "BTC"},
		{name: "several currencies", currencies: currency.Currencies{currency.BTC, currency.ETH, currency.USDT}, exp: "BTC,ETH,USDT"},
		{name: "at the maximum", currencies: slices.Repeat(currency.Currencies{currency.BTC}, 20), maximum: 20, exp: "BTC" + strings.Repeat(",BTC", 19)},
		{name: "above the maximum", currencies: slices.Repeat(currency.Currencies{currency.BTC}, 21), maximum: 20, err: errExceedLimit},
		{name: "no maximum", currencies: slices.Repeat(currency.Currencies{currency.BTC}, 21), exp: "BTC" + strings.Repeat(",BTC", 20)},
		{name: "empty currency", currencies: currency.Currencies{currency.BTC, currency.EMPTYCODE}, err: currency.ErrCurrencyCodeEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := formatFundingCurrencies(tc.currencies, tc.maximum)
			require.ErrorIs(t, err, tc.err, "formatFundingCurrencies must return the expected error")
			assert.Equal(t, tc.exp, got, "formatFundingCurrencies should join the currencies")
		})
	}
}

func TestCheckFundingTimeWindow(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1597000000000), time.UnixMilli(1597100000000)
	for _, tc := range []struct {
		name       string
		start, end time.Time
		err        error
	}{
		{name: "no bounds"},
		{name: "start only", start: start},
		{name: "end only", end: end},
		{name: "ordered window", start: start, end: end},
		{name: "equal bounds", start: start, end: start},
		{name: "start after end", start: end, end: start, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, checkFundingTimeWindow(tc.start, tc.end), tc.err, "checkFundingTimeWindow should return the expected error")
		})
	}
}

func TestAssetBillsParams(t *testing.T) {
	t.Parallel()
	assert.Empty(t, assetBillsParams(currency.EMPTYCODE, "", "", "", 0), "assetBillsParams should send no unset filter")
	exp := url.Values{"ccy": {"BTC"}, "type": {"284"}, "thirdPartyType": {"2"}, "clientId": {"transfer1"}, "limit": {"5"}}
	assert.Equal(t, exp, assetBillsParams(currency.BTC, "284", "2", "transfer1", 5), "assetBillsParams should send every filter")
}

func TestFiatOrderHistoryParams(t *testing.T) {
	t.Parallel()
	_, err := fiatOrderHistoryParams(&FiatOrderHistoryRequest{StartTime: time.UnixMilli(1707430000000), EndTime: time.UnixMilli(1707429000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "fiatOrderHistoryParams must reject a start after the end")

	params, err := fiatOrderHistoryParams(&FiatOrderHistoryRequest{})
	require.NoError(t, err, "fiatOrderHistoryParams must not error without filters")
	assert.Empty(t, params, "fiatOrderHistoryParams should send no unset filter")

	params, err = fiatOrderHistoryParams(&FiatOrderHistoryRequest{
		Currency:      currency.TRY,
		PaymentMethod: "TR_BANKS",
		State:         "completed",
		StartTime:     time.UnixMilli(1707429000000),
		EndTime:       time.UnixMilli(1707430000000),
		Limit:         2,
	})
	require.NoError(t, err, "fiatOrderHistoryParams must not error")
	exp := url.Values{
		"ccy": {"TRY"}, "paymentMethod": {"TR_BANKS"}, "state": {"completed"},
		"after": {"1707429000000"}, "before": {"1707430000000"}, "limit": {"2"},
	}
	assert.Equal(t, exp, params, "fiatOrderHistoryParams should send the window's start as after and its end as before")
}

func TestFormatFundingSide(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, side, exp string
		err             error
	}{
		{name: "lower case buy", side: "buy", exp: "buy"},
		{name: "upper case sell", side: "SELL", exp: "sell"},
		{name: "title case buy", side: "Buy", exp: "buy"},
		{name: "empty side", err: order.ErrSideIsInvalid},
		{name: "unsupported side", side: "long", err: order.ErrSideIsInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := formatFundingSide(tc.side)
			require.ErrorIs(t, err, tc.err, "formatFundingSide must return the expected error")
			assert.Equal(t, tc.exp, got, "formatFundingSide should return the lower case side")
		})
	}
}

func TestFormatStatementMonth(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		month time.Month
		exp   string
		err   error
	}{
		{name: "January", month: time.January, exp: "Jan"},
		{name: "May", month: time.May, exp: "May"},
		{name: "June", month: time.June, exp: "Jun"},
		{name: "December", month: time.December, exp: "Dec"},
		{name: "below January", month: 0, err: errInvalidMonth},
		{name: "above December", month: 13, err: errInvalidMonth},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := formatStatementMonth(tc.month)
			require.ErrorIs(t, err, tc.err, "formatStatementMonth must return the expected error")
			assert.Equal(t, tc.exp, got, "formatStatementMonth should return the month's abbreviation")
		})
	}
}
