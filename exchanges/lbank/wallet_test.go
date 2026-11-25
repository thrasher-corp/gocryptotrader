package lbank

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetSystemStatus(t *testing.T) {
	t.Parallel()
	result, err := e.GetSystemStatus(t.Context())
	require.NoError(t, err, "GetSystemStatus must not error")
	if mockTests {
		exp := &SystemStatusResponse{Status: "1"}
		assert.Equal(t, exp, result, "GetSystemStatus should decode the status")
		return
	}
	assert.Contains(t, []string{"0", "1"}, result.Status, "GetSystemStatus should return a documented status")
}

func TestGetAllCoinInformation(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetAllCoinInformation(t.Context())
	require.NoError(t, err, "GetAllCoinInformation must not error")
	if mockTests {
		doge, usdt := currency.NewCode("doge"), currency.NewCode("usdt")
		exp := []CoinInformationResponse{
			{
				Coin: doge, TotalAmount: 4302.693, AvailableAmount: 4302.693,
				Networks: []CoinNetwork{
					{IsDefault: true, Coin: doge, Name: "bep20(bsc)", Network: "bsc", FeeType: 1, WithdrawFee: 9.7059, FeeAssetCode: doge, MinimumWithdrawal: 10, MinimumTransfer: 0.0001, MinimumDeposit: 0.0001},
					{Coin: doge, Name: "dogecoin", Network: "dogecoin", FeeType: 1, WithdrawFee: 9.7059, FeeAssetCode: doge, MinimumWithdrawal: 10, MinimumTransfer: 0.0001, MinimumDeposit: 10},
				},
			},
			{
				Coin: usdt, TotalAmount: 15.36, AvailableAmount: 14.36, FrozenAmount: 1,
				Networks: []CoinNetwork{
					{IsDefault: true, Coin: usdt, Name: "trc20", Network: "tron", FeeType: 1, WithdrawFee: 1.5, FeeAssetCode: usdt, MinimumWithdrawal: 5, MinimumTransfer: 0.0001, MinimumDeposit: 0.01},
					{Coin: usdt, Name: "erc20", Network: "eth", FeeType: 3, WithdrawFee: 1, WithdrawFeeRate: 0.001, FeeAssetCode: usdt, MinimumWithdrawal: 30, MinimumTransfer: 0.0001, MinimumDeposit: 20},
				},
			},
		}
		assert.Equal(t, exp, result, "GetAllCoinInformation should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllCoinInformation should return the account's coins")
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *WithdrawRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no address", arg: &WithdrawRequest{Coin: currency.USDT, Amount: 1}, err: errAddressRequired},
		{name: "no coin", arg: &WithdrawRequest{Address: "TDxtabCC8iQwaxUUrPcE4WL2jArGAfvQ5A", Amount: 1}, err: currency.ErrCurrencyCodeEmpty},
		{name: "no amount", arg: &WithdrawRequest{Address: "TDxtabCC8iQwaxUUrPcE4WL2jArGAfvQ5A", Coin: currency.USDT}, err: limits.ErrAmountBelowMin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.Withdraw(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "Withdraw should refuse the request before sending it")
		})
	}

	// A withdrawal cannot be undone, so the live case sends less than LBank's minimum to no address
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("The mock records no reply to a request built to be refused")
		}
		skipLiveUnlessCanManipulate(t)
		_, err := e.Withdraw(t.Context(), &WithdrawRequest{Address: liveMalformedAddress, Coin: currency.USDT, Amount: liveWithdrawalAmount, NetworkName: "trc20"})
		requireRefusedOnMerits(t, err, "Withdraw", withdrawalRefusalCodes)
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("A withdrawal LBank accepts moves funds off LBank, which a live test cannot undo")
		}
		t.Run("network withdrawal", func(t *testing.T) {
			result, err := e.Withdraw(t.Context(), &WithdrawRequest{
				Address:         "TDxtabCC8iQwaxUUrPcE4WL2jArGAfvQ5A",
				Coin:            currency.USDT,
				Amount:          10,
				Fee:             1.5,
				NetworkName:     "trc20",
				Mark:            "gct",
				WithdrawOrderID: "gct-withdrawal-1",
			})
			require.NoError(t, err, "Withdraw must not error")
			exp := &WithdrawResponse{WithdrawID: 1900377, Fee: 1.5}
			assert.Equal(t, exp, result, "Withdraw should decode the withdrawal")
		})
		t.Run("internal transfer", func(t *testing.T) {
			result, err := e.Withdraw(t.Context(), &WithdrawRequest{
				Address:          "LBA0B12679",
				Coin:             currency.USDT,
				Amount:           10,
				Memo:             "gct-memo",
				AddressName:      "gct-book",
				InternalTransfer: true,
			})
			require.NoError(t, err, "Withdraw must not error for an internal transfer")
			exp := &WithdrawResponse{WithdrawID: 1900379}
			assert.Equal(t, exp, result, "Withdraw should decode an internal transfer")
		})
	})
}

func TestGetDepositHistory(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1792195200000), time.UnixMilli(1792800000000)
	t.Run("start after the end", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetDepositHistory(t.Context(), &DepositHistoryRequest{StartTime: end, EndTime: start})
		require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDepositHistory must reject a start after the end")
	})

	// These cases share the package exchange and run in order
	t.Run("page of deposits", func(t *testing.T) {
		skipLiveWithoutCredentials(t)
		result, err := e.GetDepositHistory(t.Context(), nil)
		require.NoError(t, err, "GetDepositHistory must not error")
		if !mockTests {
			assert.NotNil(t, result, "GetDepositHistory should return a page")
			return
		}
		exp := &DepositHistoryResponse{
			Total: 1, PageLength: 20, CurrentPage: 1,
			Deposits: []DepositRecord{{
				Coin: currency.NewCode("usdt"), Amount: 9, Address: "TYASr5UV6HEcXatwdFQfmLVUqQQQMUxHLS", NetworkName: "trc20",
				TransactionID: "081e4e9351dd0274922168da5f2d14ea6c495b1c3b440244f4a6dd9fe196bf2b", Status: "2", InsertTime: types.Time(time.UnixMilli(1649012310000)),
			}},
		}
		assert.Equal(t, exp, result, "GetDepositHistory should decode a page of deposits")
	})
	t.Run("bare list of deposits", func(t *testing.T) {
		if !mockTests {
			t.Skip("The expected reply is only recorded in the mock")
		}
		// LBank's docs show the deposits as a bare list, which a filtered request's fixture holds
		result, err := e.GetDepositHistory(t.Context(), &DepositHistoryRequest{Status: 2, Coin: currency.NewCode("lbk"), StartTime: start, EndTime: end})
		require.NoError(t, err, "GetDepositHistory must not error with filters")
		exp := &DepositHistoryResponse{
			Deposits: []DepositRecord{{
				Coin: currency.NewCode("lbk"), Amount: 443, Address: "QiEik5W3szxqtjJnzp14apfrY7McUdX9sa", NetworkName: "lbk20",
				TransactionID: "0x4baf730c242fe6d85559edb40399daaf8dca16255382a5ce36934f0ff964a776", Status: "2", InsertTime: types.Time(time.UnixMilli(1644378926000)),
			}},
		}
		assert.Equal(t, exp, result, "GetDepositHistory should decode a bare list of deposits")
	})
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1792195200000), time.UnixMilli(1792800000000)
	t.Run("start after the end", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetWithdrawalHistory(t.Context(), &WithdrawalHistoryRequest{StartTime: end, EndTime: start})
		require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetWithdrawalHistory must reject a start after the end")
	})

	// These cases share the package exchange and run in order
	t.Run("page of withdrawals", func(t *testing.T) {
		skipLiveWithoutCredentials(t)
		result, err := e.GetWithdrawalHistory(t.Context(), nil)
		require.NoError(t, err, "GetWithdrawalHistory must not error")
		if !mockTests {
			assert.NotNil(t, result, "GetWithdrawalHistory should return a page")
			return
		}
		exp := &WithdrawalHistoryResponse{
			Total: 1, PageLength: 20, CurrentPage: 1,
			Withdrawals: []WithdrawalRecord{{
				ID: 1902194, Coin: currency.NewCode("usdt"), Amount: 2, Fee: 1, Address: "TBjrW5JHDyPZjFc5nrRMhRWUDaJmhGhmD6", NetworkName: "trc20",
				TransferType: "数字资产提现", TransactionID: "47eeee2763ad49b8817524dacfa7d092fb58f8b0ab7e5d25473314df1a793c3d", Status: "4",
				ApplyTime: types.Time(time.UnixMilli(1649014002000)),
			}},
		}
		assert.Equal(t, exp, result, "GetWithdrawalHistory should decode a page of withdrawals")
	})
	t.Run("bare list of withdrawals", func(t *testing.T) {
		if !mockTests {
			t.Skip("The expected reply is only recorded in the mock")
		}
		result, err := e.GetWithdrawalHistory(t.Context(), &WithdrawalHistoryRequest{Status: 1, Coin: currency.NewCode("lbk"), WithdrawOrderID: "werqwerq123425353", StartTime: start, EndTime: end})
		require.NoError(t, err, "GetWithdrawalHistory must not error with filters")
		exp := &WithdrawalHistoryResponse{
			Withdrawals: []WithdrawalRecord{{
				ID: 93182, Coin: currency.NewCode("lbk"), Amount: 234, Fee: 2, Address: "34252342", NetworkName: "lbk30",
				TransferType: "Digital Asset Withdrawal", WithdrawOrderID: "werqwerq123425353", Status: "1", ApplyTime: types.Time(time.UnixMilli(1644476930000)),
			}},
		}
		assert.Equal(t, exp, result, "GetWithdrawalHistory should decode a bare list of withdrawals")
	})
}

func TestGetCoinDepositAddress(t *testing.T) {
	t.Parallel()
	t.Run("no coin", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetCoinDepositAddress(t.Context(), currency.EMPTYCODE, "trc20")
		require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCoinDepositAddress must require a coin")
	})

	// These cases share the package exchange and run in order
	t.Run("chain asked for", func(t *testing.T) {
		skipLiveWithoutCredentials(t)
		result, err := e.GetCoinDepositAddress(t.Context(), currency.USDT, "trc20")
		require.NoError(t, err, "GetCoinDepositAddress must not error")
		if !mockTests {
			assert.NotEmpty(t, result.Address, "GetCoinDepositAddress should return an address")
			return
		}
		exp := &DepositAddressResponse{Coin: currency.NewCode("usdt"), Address: "TDxtabCC8iQwaxUUrPcE4WL2jArGAfvQ5A"}
		assert.Equal(t, exp, result, "GetCoinDepositAddress should decode the address")
	})
	t.Run("default chain", func(t *testing.T) {
		if !mockTests {
			t.Skip("The expected reply is only recorded in the mock")
		}
		result, err := e.GetCoinDepositAddress(t.Context(), currency.XRP, "")
		require.NoError(t, err, "GetCoinDepositAddress must not error on the default chain")
		exp := &DepositAddressResponse{Coin: currency.NewCode("xrp"), Address: "rLbKJjgXNjTGLFrEXbF5yBuWR3XWHF5nMa", Memo: "107824"}
		assert.Equal(t, exp, result, "GetCoinDepositAddress should decode the address and memo")
	})
}

func TestAddDepositAddress(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		coin    currency.Code
		network string
		err     error
	}{
		{name: "no coin", coin: currency.EMPTYCODE, network: "erc20", err: currency.ErrCurrencyCodeEmpty},
		{name: "no network", coin: currency.USDT, err: errNetworkRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AddDepositAddress(t.Context(), tc.coin, tc.network)
			assert.ErrorIs(t, err, tc.err, "AddDepositAddress should refuse the request before sending it")
		})
	}

	// An address added cannot be removed, so the live case asks for one on a network LBank doesn't list
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("The mock records no reply to a request built to be refused")
		}
		skipLiveUnlessCanManipulate(t)
		_, err := e.AddDepositAddress(t.Context(), currency.USDT, liveUnlistedNetwork)
		requireRefusedOnMerits(t, err, "AddDepositAddress", depositAddressRefusalCodes)
	})

	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("An address LBank adds cannot be removed by a live test")
		}
		result, err := e.AddDepositAddress(t.Context(), currency.USDT, "erc20")
		require.NoError(t, err, "AddDepositAddress must not error")
		exp := &DepositAddressResponse{Coin: currency.NewCode("usdt"), NetworkName: "erc20", Address: "0xabc123"}
		assert.Equal(t, exp, result, "AddDepositAddress should decode the new address")
	})
}

func TestGetAssetDetails(t *testing.T) { //nolint:tparallel // Cases share the package exchange and run in order
	t.Parallel()
	t.Run("every coin", func(t *testing.T) {
		skipLiveWithoutCredentials(t)
		result, err := e.GetAssetDetails(t.Context(), currency.EMPTYCODE)
		require.NoError(t, err, "GetAssetDetails must not error")
		if !mockTests {
			assert.NotEmpty(t, result, "GetAssetDetails should return coins")
			return
		}
		exp := map[currency.Code]AssetDetailResponse{
			currency.NewCode("htmoon"): {CanTransfer: true},
			currency.NewCode("phx"):    {CanDeposit: true, CanWithdraw: true, CanTransfer: true},
		}
		assert.Equal(t, exp, result, "GetAssetDetails should decode every coin")
	})
	t.Run("one coin", func(t *testing.T) {
		if !mockTests {
			t.Skip("The expected reply is only recorded in the mock")
		}
		result, err := e.GetAssetDetails(t.Context(), currency.USDT)
		require.NoError(t, err, "GetAssetDetails must not error for a coin")
		exp := map[currency.Code]AssetDetailResponse{
			currency.NewCode("usdt"): {CanDeposit: true, CanWithdraw: true, CanTransfer: true, WithdrawFee: 1.5, MinimumWithdrawAmount: 5},
		}
		assert.Equal(t, exp, result, "GetAssetDetails should decode the coin")
	})
}

func TestGetTradingFeeRates(t *testing.T) { //nolint:tparallel // Cases share the package exchange and run in order
	t.Parallel()
	t.Run("one pair", func(t *testing.T) {
		skipLiveWithoutCredentials(t)
		result, err := e.GetTradingFeeRates(t.Context(), spotTestPair)
		require.NoError(t, err, "GetTradingFeeRates must not error")
		exp := []TradingFeeRateResponse{{Symbol: "btc_usdt", MakerCommissionPercent: 0.08, TakerCommissionPercent: 0.1}}
		if !mockTests {
			require.Len(t, result, 1, "GetTradingFeeRates must return the pair asked for")
			assert.Equal(t, "btc_usdt", result[0].Symbol, "GetTradingFeeRates should return the pair asked for")
			return
		}
		assert.Equal(t, exp, result, "GetTradingFeeRates should decode the pair's rates")
	})
	t.Run("every pair", func(t *testing.T) {
		if !mockTests {
			t.Skip("The expected reply is only recorded in the mock")
		}
		result, err := e.GetTradingFeeRates(t.Context(), currency.EMPTYPAIR)
		require.NoError(t, err, "GetTradingFeeRates must not error for every pair")
		exp := []TradingFeeRateResponse{
			{Symbol: "btc_usdt", MakerCommissionPercent: 0.08, TakerCommissionPercent: 0.1},
			{Symbol: "lbk_usdt", MakerCommissionPercent: 0.1, TakerCommissionPercent: 10},
		}
		assert.Equal(t, exp, result, "GetTradingFeeRates should decode every pair's rates")
	})
}

func TestGetAPIRestrictions(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetAPIRestrictions(t.Context())
	require.NoError(t, err, "GetAPIRestrictions must not error")
	if mockTests {
		exp := &APIRestrictionsResponse{ReadingEnabled: true, SpotTradingEnabled: true, IPRestricted: true, CreateTime: types.Time(time.UnixMilli(1643091005000))}
		assert.Equal(t, exp, result, "GetAPIRestrictions should decode every permission")
		return
	}
	assert.True(t, result.ReadingEnabled, "GetAPIRestrictions should report the key can read")
}

func TestSetMillisecondTimeRange(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1792195200123), time.UnixMilli(1792800000456)
	for _, tc := range []struct {
		name       string
		start, end time.Time
		exp        url.Values
		err        error
	}{
		{name: "both", start: start, end: end, exp: url.Values{"startTime": {"1792195200123"}, "endTime": {"1792800000456"}}},
		{name: "start only", start: start, exp: url.Values{"startTime": {"1792195200123"}}},
		{name: "end only", end: end, exp: url.Values{"endTime": {"1792800000456"}}},
		{name: "neither", exp: url.Values{}},
		{name: "start after end", start: end, end: start, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := url.Values{}
			err := setMillisecondTimeRange(params, tc.start, tc.end)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "setMillisecondTimeRange should reject the range")
				return
			}
			require.NoError(t, err, "setMillisecondTimeRange must not error")
			assert.Equal(t, tc.exp, params, "setMillisecondTimeRange should set the times in milliseconds")
		})
	}
}

func TestDepositHistoryResponseUnmarshalJSON(t *testing.T) { //nolint:tparallel // Cases decode into one response and run in order
	t.Parallel()
	var deposits DepositHistoryResponse
	t.Run("malformed bare list", func(t *testing.T) {
		assert.Error(t, deposits.UnmarshalJSON([]byte(`[{"amount":true}]`)), "DepositHistoryResponse should reject a malformed bare list")
	})
	t.Run("malformed page", func(t *testing.T) {
		assert.Error(t, deposits.UnmarshalJSON([]byte(`{"depositOrders":{}}`)), "DepositHistoryResponse should reject a malformed page")
	})
}

func TestWithdrawalHistoryResponseUnmarshalJSON(t *testing.T) { //nolint:tparallel // Cases decode into one response and run in order
	t.Parallel()
	var withdrawals WithdrawalHistoryResponse
	t.Run("malformed bare list", func(t *testing.T) {
		assert.Error(t, withdrawals.UnmarshalJSON([]byte(`[{"fee":true}]`)), "WithdrawalHistoryResponse should reject a malformed bare list")
	})
	t.Run("malformed page", func(t *testing.T) {
		assert.Error(t, withdrawals.UnmarshalJSON([]byte(`{"withdraws":{}}`)), "WithdrawalHistoryResponse should reject a malformed page")
	})
}
