package bybit

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestSetDepositAccount(t *testing.T) {
	t.Parallel()
	_, err := e.SetDepositAccount(t.Context(), "")
	require.ErrorIs(t, err, errMissingAccountType, "SetDepositAccount must reject an empty account type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetDepositAccount(t.Context(), "FUND")
	require.NoError(t, err, "SetDepositAccount must not error")
	if mockTests {
		exp := &StatusResponse{
			Status: true,
		}
		assert.Equal(t, exp, result, "SetDepositAccount should decode every field")
		return
	}
	assert.True(t, result.Status.Bool(), "SetDepositAccount should succeed")
}

func TestGetDepositRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDepositRecords must reject a nil request")
	_, err = e.GetDepositRecords(t.Context(), &DepositRecordsRequest{StartTime: time.UnixMilli(1742728163000), EndTime: time.UnixMilli(1742700000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDepositRecords must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDepositRecords(t.Context(), &DepositRecordsRequest{})
	require.NoError(t, err, "GetDepositRecords must not error")
	t.Run("every filter", func(t *testing.T) {
		t.Parallel()
		next, err := e.GetDepositRecords(t.Context(), &DepositRecordsRequest{
			ID:            "160239001",
			TransactionID: "0x5d2e8f1a9c3b4d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6",
			Coin:          currency.NewCode("usdt"),
			StartTime:     time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
			EndTime:       time.Date(2026, time.September, 8, 0, 0, 0, 0, time.UTC),
			Limit:         1,
			Cursor:        result.NextPageCursor,
		})
		require.NoError(t, err, "GetDepositRecords must not error")
		if mockTests {
			exp := &DepositRecordsResponse{
				Rows: []DepositRecord{
					{
						ID:                  "160239001",
						Coin:                currency.USDT,
						Chain:               "ETH",
						Amount:              1500,
						TransactionID:       "0x5d2e8f1a9c3b4d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6",
						Status:              3,
						ToAddress:           "0xE3De6d711e0951d34777b5Cd93c827F822ee8514",
						SuccessAt:           types.Time(time.UnixMilli(1788310000000)),
						Confirmations:       64,
						TransactionIndex:    "87",
						BlockHash:           "0x4c3b2a1908f7e6d5c4b3a29180f7e6d5c4b3a29180f7e6d5c4b3a29180f7e6d5",
						BatchReleaseLimit:   -1,
						DepositType:         "0",
						FromAddress:         "XXXXXX",
						TaxDepositRecordsID: "0",
					},
				},
			}
			assert.Equal(t, exp, next, "GetDepositRecords should send every filter and decode the matching page")
			return
		}
		assert.NotNil(t, next, "GetDepositRecords should return a page")
	})
	if mockTests {
		exp := &DepositRecordsResponse{
			Rows: []DepositRecord{
				{
					ID:                  "160237231",
					Coin:                currency.USDT,
					Chain:               "TRX",
					Amount:              999.0496,
					TransactionID:       "04bf3fbad2fc85b107a42cfdc5ff83110092b606ca754efa0f032f8b94b3262e",
					Status:              3,
					ToAddress:           "TDGYpm5zPacnEqKV34TJPuhJhHom9hcXAy",
					SuccessAt:           types.Time(time.Unix(1742728163, 0)),
					Confirmations:       50,
					TransactionIndex:    "0",
					BlockHash:           "000000000436ab4dabc8a4a87beb2262d2d87f6761a825494c4f1d5ae11b27e8",
					BatchReleaseLimit:   -1,
					DepositType:         "0",
					FromAddress:         "TJ7hhYhVhaxNx6BPyq7yFpqZrQULL3JSdb",
					TaxDepositRecordsID: "0",
				},
				{
					ID:                  "160237245",
					Coin:                currency.XRP,
					Chain:               "XRP",
					Amount:              250.5,
					TransactionID:       "7f35e3a8d1c4b2e690a1f4c3b2d1e0f9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d3",
					Status:              2,
					ToAddress:           "rXXXXXX",
					Tag:                 "501234",
					DepositFee:          0.25,
					Confirmations:       1,
					TransactionIndex:    "3",
					BlockHash:           "8D9F4A0B1C2E3D4F5A6B7C8D9E0F1A2B3C4D5E6F7A8B9C0D1E2F3A4B5C6D7E8F",
					BatchReleaseLimit:   200000,
					DepositType:         "20",
					TaxDepositRecordsID: "7001",
					TaxStatus:           1,
					TravelRuleStatus:    1,
				},
			},
			NextPageCursor: "eyJtaW5JRCI6MTYwMjM3MjMxLCJtYXhJRCI6MTYwMjM3MjMxfQ==",
		}
		assert.Equal(t, exp, result, "GetDepositRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDepositRecords should return a page")
}

func TestGetSubDepositRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubDepositRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubDepositRecords must reject a nil request")
	_, err = e.GetSubDepositRecords(t.Context(), &SubDepositRecordsRequest{})
	require.ErrorIs(t, err, errMemberIDRequired, "GetSubDepositRecords must reject an empty sub member ID")
	_, err = e.GetSubDepositRecords(t.Context(), &SubDepositRecordsRequest{SubMemberID: "12345", StartTime: time.UnixMilli(1742738340000), EndTime: time.UnixMilli(1742700000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSubDepositRecords must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubDepositRecords(t.Context(), &SubDepositRecordsRequest{SubMemberID: "12345"})
	require.NoError(t, err, "GetSubDepositRecords must not error")
	t.Run("every filter", func(t *testing.T) {
		t.Parallel()
		next, err := e.GetSubDepositRecords(t.Context(), &SubDepositRecordsRequest{
			ID:            "160239002",
			TransactionID: "c7f5a2b3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f",
			SubMemberID:   "12345",
			Coin:          currency.NewCode("usdt"),
			StartTime:     time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
			EndTime:       time.Date(2026, time.September, 8, 0, 0, 0, 0, time.UTC),
			Limit:         1,
			Cursor:        result.NextPageCursor,
		})
		require.NoError(t, err, "GetSubDepositRecords must not error")
		if mockTests {
			exp := &DepositRecordsResponse{
				Rows: []DepositRecord{
					{
						ID:                  "160239002",
						Coin:                currency.USDT,
						Chain:               "TON",
						Amount:              75,
						TransactionID:       "c7f5a2b3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f",
						Status:              3,
						ToAddress:           "EQXXXXXX",
						Tag:                 "100000123",
						DepositFee:          0.05,
						SuccessAt:           types.Time(time.UnixMilli(1788500000000)),
						Confirmations:       2,
						TransactionIndex:    "4",
						BlockHash:           "1f2e3d4c5b6a79881f2e3d4c5b6a79881f2e3d4c5b6a79881f2e3d4c5b6a7988",
						BatchReleaseLimit:   500000,
						DepositType:         "0",
						TaxDepositRecordsID: "0",
					},
				},
			}
			assert.Equal(t, exp, next, "GetSubDepositRecords should send every filter and decode the matching page")
			return
		}
		assert.NotNil(t, next, "GetSubDepositRecords should return a page")
	})
	if mockTests {
		exp := &DepositRecordsResponse{
			Rows: []DepositRecord{
				{
					ID:                  "160237290",
					Coin:                currency.USDT,
					Chain:               "ETH",
					Amount:              120.5,
					TransactionID:       "0x3d7bddb797f0e86420c982c0723653b8b728fd0ec9953b6b354445848d83a185",
					Status:              3,
					ToAddress:           "0xE3De6d711e0951d34777b5Cd93c827F822ee8514",
					SuccessAt:           types.Time(time.Unix(1742738340, 0)),
					Confirmations:       64,
					TransactionIndex:    "112",
					BlockHash:           "0x6e0b8c7f3b1b4c46d2d1f4c0a5e8b1d2c3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8",
					BatchReleaseLimit:   -1,
					DepositType:         "0",
					FromAddress:         "0x99ced129603abc771c0dabe935c326ff6c86645d",
					TaxDepositRecordsID: "0",
				},
				{
					ID:                  "160237301",
					Coin:                currency.NewCode("TON"),
					Chain:               "TON",
					Amount:              40,
					TransactionID:       "b6e4f1a2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e",
					Status:              10011,
					ToAddress:           "EQXXXXXX",
					Tag:                 "100000123",
					DepositFee:          0.05,
					Confirmations:       2,
					TransactionIndex:    "1",
					BlockHash:           "9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d3e2f1a0b9c8d7e6f5a4b3c2d1e0f9a8b",
					BatchReleaseLimit:   50000,
					DepositType:         "10",
					TaxDepositRecordsID: "7002",
					TaxStatus:           2,
					TravelRuleStatus:    2,
				},
			},
			NextPageCursor: "eyJtaW5JRCI6MTYwMjM3MjkwLCJtYXhJRCI6MTYwMjM3MzAxfQ==",
		}
		assert.Equal(t, exp, result, "GetSubDepositRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubDepositRecords should return a page")
}

func TestGetInternalDepositRecordsOffChain(t *testing.T) {
	t.Parallel()
	_, err := e.GetInternalDepositRecordsOffChain(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetInternalDepositRecordsOffChain must reject a nil request")
	_, err = e.GetInternalDepositRecordsOffChain(t.Context(), &InternalDepositRecordsRequest{StartTime: time.Unix(1705393280, 0), EndTime: time.Unix(1705300000, 0)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetInternalDepositRecordsOffChain must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInternalDepositRecordsOffChain(t.Context(), &InternalDepositRecordsRequest{Coin: currency.ETH, Limit: 8})
	require.NoError(t, err, "GetInternalDepositRecordsOffChain must not error")
	t.Run("every filter", func(t *testing.T) {
		t.Parallel()
		next, err := e.GetInternalDepositRecordsOffChain(t.Context(), &InternalDepositRecordsRequest{
			TransactionID: "9a1e4c2d-7b3f-4e8a-b6d5-c4f3e2d1a0b9",
			StartTime:     time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
			EndTime:       time.Date(2026, time.September, 8, 0, 0, 0, 0, time.UTC),
			Coin:          currency.ETH,
			Cursor:        result.NextPageCursor,
			Limit:         8,
		})
		require.NoError(t, err, "GetInternalDepositRecordsOffChain must not error")
		if mockTests {
			exp := &InternalDepositRecordsResponse{
				Rows: []InternalDepositRecord{
					{
						ID:                  "1187",
						Type:                1,
						Coin:                currency.ETH,
						Amount:              0.25,
						Status:              2,
						Address:             "test@example.com",
						CreatedTime:         types.Time(time.Unix(1788400000, 0)),
						FromMemberID:        "100000002",
						TransactionID:       "9a1e4c2d-7b3f-4e8a-b6d5-c4f3e2d1a0b9",
						TaxDepositRecordsID: "0",
					},
				},
			}
			assert.Equal(t, exp, next, "GetInternalDepositRecordsOffChain should send every filter and decode the matching page")
			return
		}
		assert.NotNil(t, next, "GetInternalDepositRecordsOffChain should return a page")
	})
	if mockTests {
		exp := &InternalDepositRecordsResponse{
			Rows: []InternalDepositRecord{
				{
					ID:                  "1103",
					Type:                1,
					Coin:                currency.ETH,
					Amount:              0.1,
					Status:              2,
					Address:             "test@example.com",
					CreatedTime:         types.Time(time.Unix(1705393280, 0)),
					FromMemberID:        "100000002",
					TransactionID:       "77c37e5c-d9fa-41e5-bd13-c9b59d95",
					ComplianceStatus:    1,
					TaxDepositRecordsID: "7003",
					TaxStatus:           1,
				},
			},
			NextPageCursor: "eyJtaW5JRCI6MTEwMywibWF4SUQiOjExMDN9",
		}
		assert.Equal(t, exp, result, "GetInternalDepositRecordsOffChain should decode every field")
		return
	}
	assert.NotNil(t, result, "GetInternalDepositRecordsOffChain should return a page")
}

func TestGetMasterDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetMasterDepositAddress(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMasterDepositAddress must reject an empty coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMasterDepositAddress(t.Context(), currency.USDT, "")
	require.NoError(t, err, "GetMasterDepositAddress must not error")
	if mockTests {
		exp := &DepositAddressResponse{
			Coin: currency.USDT,
			Chains: []DepositAddressChain{
				{
					ChainType:         "Ethereum (ERC20)",
					AddressDeposit:    "XXXXXX",
					Chain:             "ETH",
					BatchReleaseLimit: -1,
					ContractAddress:   "831ec7",
				},
				{
					ChainType:         "TON",
					AddressDeposit:    "XXXXXX",
					TagDeposit:        "100000123",
					Chain:             "TON",
					BatchReleaseLimit: 500000,
					ContractAddress:   "Id_sDs",
				},
			},
		}
		assert.Equal(t, exp, result, "GetMasterDepositAddress should decode every field")
		return
	}
	assert.True(t, result.Coin.Equal(currency.USDT), "GetMasterDepositAddress should return the requested coin")
}

func TestGetSubDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubDepositAddress(t.Context(), currency.EMPTYCODE, "TON", "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSubDepositAddress must reject an empty coin")
	_, err = e.GetSubDepositAddress(t.Context(), currency.USDT, "", "12345")
	require.ErrorIs(t, err, errMissingChainType, "GetSubDepositAddress must reject an empty chain type")
	_, err = e.GetSubDepositAddress(t.Context(), currency.USDT, "TON", "")
	require.ErrorIs(t, err, errMemberIDRequired, "GetSubDepositAddress must reject an empty sub member ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubDepositAddress(t.Context(), currency.USDT, "TON", "12345")
	require.NoError(t, err, "GetSubDepositAddress must not error")
	if mockTests {
		exp := &DepositAddressResponse{
			Coin: currency.USDT,
			Chains: []DepositAddressChain{
				{
					ChainType:         "TON",
					AddressDeposit:    "XXXXXX",
					TagDeposit:        "100000124",
					Chain:             "TON",
					BatchReleaseLimit: 500000,
					ContractAddress:   "Id_sDs",
				},
			},
		}
		assert.Equal(t, exp, result, "GetSubDepositAddress should decode every field")
		return
	}
	assert.True(t, result.Coin.Equal(currency.USDT), "GetSubDepositAddress should return the requested coin")
}

func TestDepositAddressChainsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := DepositAddressChains{{ChainType: "TON", AddressDeposit: "XXXXXX", TagDeposit: "100000124", Chain: "TON", BatchReleaseLimit: 500000, ContractAddress: "Id_sDs"}}
	for _, tc := range []struct {
		name string
		data string
	}{
		{"object", `{"chainType":"TON","addressDeposit":"XXXXXX","tagDeposit":"100000124","chain":"TON","batchReleaseLimit":"500000","contractAddress":"Id_sDs"}`},
		{"array", `[{"chainType":"TON","addressDeposit":"XXXXXX","tagDeposit":"100000124","chain":"TON","batchReleaseLimit":"500000","contractAddress":"Id_sDs"}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got DepositAddressChains
			require.NoError(t, got.UnmarshalJSON([]byte(tc.data)), "UnmarshalJSON must not error")
			assert.Equal(t, exp, got, "UnmarshalJSON should decode the chains")
		})
	}
	var got DepositAddressChains
	assert.Error(t, got.UnmarshalJSON([]byte(`{"batchReleaseLimit":true}`)), "UnmarshalJSON should reject an invalid chain object")
	assert.Error(t, got.UnmarshalJSON([]byte(`"TON"`)), "UnmarshalJSON should reject a string")
}

func TestSubmitDepositOriginatorInfo(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitDepositOriginatorInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubmitDepositOriginatorInfo must reject a nil request")
	arg := &DepositOriginatorInfoRequest{}
	_, err = e.SubmitDepositOriginatorInfo(t.Context(), arg)
	require.ErrorIs(t, err, errDepositIDRequired, "SubmitDepositOriginatorInfo must reject an empty deposit ID")
	arg.DepositID = 1234567890
	_, err = e.SubmitDepositOriginatorInfo(t.Context(), arg)
	require.ErrorIs(t, err, errQuestionnaireRequired, "SubmitDepositOriginatorInfo must reject a missing questionnaire")
	arg.Questionnaire = &TravelRuleQuestionnaire{VASPCode: "BINANCEUS_VASP", FirstName: "John", LastName: "Smith"}
	_, err = e.SubmitDepositOriginatorInfo(t.Context(), arg)
	require.ErrorIs(t, err, errLegalTypeRequired, "SubmitDepositOriginatorInfo must reject a questionnaire without a legal type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.SubAccountID = 100000002
	arg.Questionnaire.LegalType = "individual"
	arg.Questionnaire.TransactionPurpose = "Personal investment in long-term holdings"
	result, err := e.SubmitDepositOriginatorInfo(t.Context(), arg)
	require.NoError(t, err, "SubmitDepositOriginatorInfo must not error")
	if mockTests {
		exp := &DepositOriginatorInfoResponse{
			TravelRuleStatus: 2,
		}
		assert.Equal(t, exp, result, "SubmitDepositOriginatorInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "SubmitDepositOriginatorInfo should return the travel rule status")
}

func TestDepositOriginatorInfoResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		exp  uint64
	}{
		{"documented key", `{"travelRuleStatus":2}`, 2},
		{"sampled key", `{"travel_rule_status":2}`, 2},
		{"documented zero before sampled key", `{"travelRuleStatus":0,"travel_rule_status":1}`, 0},
		{"documented zero after sampled key", `{"travel_rule_status":1,"travelRuleStatus":0}`, 0},
		{"null documented key", `{"travelRuleStatus":null,"travel_rule_status":2}`, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got DepositOriginatorInfoResponse
			require.NoError(t, got.UnmarshalJSON([]byte(tc.data)), "UnmarshalJSON must not error")
			assert.Equal(t, DepositOriginatorInfoResponse{TravelRuleStatus: tc.exp}, got, "UnmarshalJSON should decode the travel rule status")
		})
	}
	got := DepositOriginatorInfoResponse{TravelRuleStatus: 3}
	require.NoError(t, got.UnmarshalJSON([]byte(`{}`)), "UnmarshalJSON must not error without either key")
	assert.Equal(t, DepositOriginatorInfoResponse{TravelRuleStatus: 3}, got, "UnmarshalJSON should keep the status without either key")
	// An approval, 0, under the sampled key replaces a status decoded before
	require.NoError(t, got.UnmarshalJSON([]byte(`{"travel_rule_status":0}`)), "UnmarshalJSON must not error for an approval under the sampled key")
	assert.Equal(t, DepositOriginatorInfoResponse{}, got, "UnmarshalJSON should replace an earlier status with an approval under the sampled key")
	assert.Error(t, got.UnmarshalJSON([]byte(`{"travel_rule_status":"pending"}`)), "UnmarshalJSON should reject a non-numeric sampled status")
}

func TestGetAllowedDepositCoinInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllowedDepositCoinInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllowedDepositCoinInfo must reject a nil request")
	// The route answers a coin or a chain alone: coin=USDT lists the coin's chains and chain=ETH the chain's coins
	for _, tc := range []struct {
		req   *AllowedDepositCoinInfoRequest
		query string
	}{
		{&AllowedDepositCoinInfoRequest{Coin: currency.USDT}, "coin=USDT"},
		{&AllowedDepositCoinInfoRequest{Chain: "ETH"}, "chain=ETH"},
	} {
		ex := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, tc.query, r.URL.RawQuery, "GetAllowedDepositCoinInfo should send a lone filter by itself")
			_, err := fmt.Fprint(w, `{"retCode":0,"retMsg":"success","result":{"configList":[{"coin":"USDT","chain":"ETH","coinShowName":"USDT","chainType":"Ethereum (ERC20)","blockConfirmNumber":6,"minDepositAmount":"0.005"}],"nextPageCursor":"eyJwYWdlIjoyLCJsaW1pdCI6MTB9"},"retExtInfo":{},"time":1791971157539}`)
			assert.NoError(t, err, "writing the reply should not error")
		})
		_, err = ex.GetAllowedDepositCoinInfo(t.Context(), tc.req)
		require.NoErrorf(t, err, "GetAllowedDepositCoinInfo must accept the lone filter %s", tc.query)
	}

	// The route lists chains closed to deposits as well as open ones, such as USDT on Omni beside USDT on Ethereum
	listing := newTransportTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := fmt.Fprint(w, `{"retCode":0,"retMsg":"success","result":{"configList":[{"coin":"USDT","chain":"ETH","coinShowName":"USDT","chainType":"Ethereum (ERC20)","blockConfirmNumber":6,"minDepositAmount":"0.005","depositSwitch":"1"},{"coin":"USDT","chain":"OMNI","coinShowName":"USDT","chainType":"OMNI","blockConfirmNumber":1,"minDepositAmount":"0.0001","depositSwitch":"0"}],"nextPageCursor":"eyJwYWdlIjoyLCJsaW1pdCI6MTB9"},"retExtInfo":{},"time":1792143293772}`)
		assert.NoError(t, err, "writing the reply should not error")
	})
	chains, err := listing.GetAllowedDepositCoinInfo(t.Context(), &AllowedDepositCoinInfoRequest{Coin: currency.USDT})
	require.NoError(t, err, "GetAllowedDepositCoinInfo must not error")
	assert.Equal(t, []AllowedDepositCoin{
		{Coin: currency.USDT, Chain: "ETH", CoinShowName: "USDT", ChainType: "Ethereum (ERC20)", BlockConfirmNumber: 6, MinDepositAmount: 0.005, DepositSwitch: "1"},
		{Coin: currency.USDT, Chain: "OMNI", CoinShowName: "USDT", ChainType: "OMNI", BlockConfirmNumber: 1, MinDepositAmount: 0.0001, DepositSwitch: "0"},
	}, chains.ConfigList, "GetAllowedDepositCoinInfo should tell a chain open to deposits from one closed to them")

	result, err := e.GetAllowedDepositCoinInfo(t.Context(), &AllowedDepositCoinInfoRequest{Coin: currency.ETH, Chain: "ETH"})
	require.NoError(t, err, "GetAllowedDepositCoinInfo must not error")
	t.Run("next page", func(t *testing.T) {
		t.Parallel()
		// Live, the route returns the same configuration on every page and only advances the cursor
		next, err := e.GetAllowedDepositCoinInfo(t.Context(), &AllowedDepositCoinInfoRequest{Coin: currency.ETH, Chain: "ETH", Limit: 10, Cursor: result.NextPageCursor})
		require.NoError(t, err, "GetAllowedDepositCoinInfo must not error")
		if mockTests {
			exp := &AllowedDepositCoinInfoResponse{
				ConfigList: []AllowedDepositCoin{
					{
						Coin:               currency.ETH,
						Chain:              "ETH",
						CoinShowName:       "ETH",
						ChainType:          "Ethereum (ERC20)",
						BlockConfirmNumber: 6,
						MinDepositAmount:   0.00005,
						DepositSwitch:      "1",
					},
				},
				NextPageCursor: "eyJwYWdlIjozLCJsaW1pdCI6MTB9",
			}
			assert.Equal(t, exp, next, "GetAllowedDepositCoinInfo should send the first page's cursor and decode the next page")
			return
		}
		require.NotEmpty(t, next.ConfigList, "GetAllowedDepositCoinInfo must return the coin")
		assert.True(t, next.ConfigList[0].Coin.Equal(currency.ETH), "GetAllowedDepositCoinInfo should return the requested coin")
		assert.NotEmpty(t, next.NextPageCursor, "GetAllowedDepositCoinInfo should return a further cursor")
	})
	if mockTests {
		exp := &AllowedDepositCoinInfoResponse{
			ConfigList: []AllowedDepositCoin{
				{
					Coin:               currency.ETH,
					Chain:              "ETH",
					CoinShowName:       "ETH",
					ChainType:          "Ethereum (ERC20)",
					BlockConfirmNumber: 6,
					MinDepositAmount:   0.00005,
					DepositSwitch:      "1",
				},
			},
			NextPageCursor: "eyJwYWdlIjoyLCJsaW1pdCI6MTB9",
		}
		assert.Equal(t, exp, result, "GetAllowedDepositCoinInfo should decode every field")
		return
	}
	require.NotEmpty(t, result.ConfigList, "GetAllowedDepositCoinInfo must return the coin")
	assert.True(t, result.ConfigList[0].Coin.Equal(currency.ETH), "GetAllowedDepositCoinInfo should return the requested coin")
	assert.Equal(t, "ETH", result.ConfigList[0].Chain, "GetAllowedDepositCoinInfo should return the requested chain")
}
