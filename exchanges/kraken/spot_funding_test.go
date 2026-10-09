package kraken

import (
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

func TestGetDepositMethods(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositMethods(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDepositMethods must reject a nil request")
	_, err = e.GetDepositMethods(t.Context(), &DepositMethodsRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositMethods must reject an empty asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDepositMethods(t.Context(), &DepositMethodsRequest{Asset: currency.XBT, AssetClass: "currency", RebaseMultiplier: "rebased"})
	require.NoError(t, err, "GetDepositMethods must not error")
	if mockTests {
		exp := []DepositMethod{
			{Method: "Bitcoin", Limit: DepositLimit{Unlimited: true}, Fee: 0.00001, AddressSetupFee: 0.00002, GenerateAddress: true, Minimum: 0.0001},
			{Method: "Bitcoin Lightning", Limit: DepositLimit{Amount: 0.5}, FeePercentage: 0.1, Minimum: 0.00001},
		}
		assert.Equal(t, exp, result, "GetDepositMethods should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetDepositMethods should return methods")
}

func TestDepositLimitUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		exp  DepositLimit
	}{
		{name: "no limit", data: `false`, exp: DepositLimit{Unlimited: true}},
		{name: "quoted amount", data: `"0.50000000"`, exp: DepositLimit{Amount: 0.5}},
		{name: "bare amount", data: `2.5`, exp: DepositLimit{Amount: 2.5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			limit := DepositLimit{Amount: 1, Unlimited: true}
			require.NoError(t, limit.UnmarshalJSON([]byte(tc.data)), "UnmarshalJSON must not error")
			assert.Equal(t, tc.exp, limit, "UnmarshalJSON should replace the whole limit")
		})
	}
	var limit DepositLimit
	assert.Error(t, limit.UnmarshalJSON([]byte(`true`)), "UnmarshalJSON should reject true, which is neither an amount nor no limit")
}

func TestGetDepositAddresses(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddresses(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDepositAddresses must reject a nil request")
	_, err = e.GetDepositAddresses(t.Context(), &DepositAddressesRequest{Method: "Bitcoin"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddresses must reject an empty asset")
	_, err = e.GetDepositAddresses(t.Context(), &DepositAddressesRequest{Asset: currency.XBT})
	require.ErrorIs(t, err, errDepositMethodRequired, "GetDepositAddresses must reject an empty method")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *DepositAddressesRequest
		exp  []DepositAddress
	}{
		{
			name: "tagged addresses",
			req:  &DepositAddressesRequest{Asset: currency.XRP, AssetClass: "currency", Method: "Ripple XRP"},
			exp: []DepositAddress{
				{Address: "rLHzPsX3oXdzU2qP17kHCH2G4csZv1rAJh", New: true, Tag: "1361101127"},
				{Address: "rLHzPsX3oXdzU2qP17kHCH2G4csZv1rAJh", Tag: "2730947812"},
			},
		},
		{
			name: "memo address",
			req:  &DepositAddressesRequest{Asset: currency.EOS, Method: "EOS"},
			exp:  []DepositAddress{{Address: "krakenkraken", Memo: "4150096490"}},
		},
		{
			name: "new Lightning invoice",
			req:  &DepositAddressesRequest{Asset: currency.XBT, Method: "Bitcoin Lightning", New: true, Amount: 0.0005},
			exp: []DepositAddress{
				{
					Address:    "lnbc500u1p5kr4kenpp5q8m2z7d3x9c4v6b8n0l2k4j6h8g0f2d4s6a8p0o2i4u6y8t0r2e4w6q8",
					ExpireTime: types.Time(time.Unix(1791508202, 0)),
					New:        true,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests && tc.req.New {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
			}
			result, err := e.GetDepositAddresses(t.Context(), tc.req)
			require.NoError(t, err, "GetDepositAddresses must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetDepositAddresses should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetDepositAddresses should return addresses")
		})
	}
}

func TestGetRecentDepositsStatus(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	_, err := e.GetRecentDepositsStatus(t.Context(), &RecentTransfersStatusRequest{Start: end, End: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetRecentDepositsStatus must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRecentDepositsStatus(t.Context(), &RecentTransfersStatusRequest{
		Asset:            currency.ETH,
		AssetClass:       "currency",
		Method:           "Ether (Hex)",
		Start:            start,
		End:              end,
		Limit:            2,
		RebaseMultiplier: "rebased",
	})
	require.NoError(t, err, "GetRecentDepositsStatus must not error")
	if mockTests {
		exp := &RecentDepositsStatusResponse{
			Deposits: []RecentDeposit{
				{
					Method:         "Ether (Hex)",
					AssetClass:     "currency",
					Asset:          currency.XETH,
					ReferenceID:    "FTQcuak-V6Za8qrPnhsTx47yYLz8Tg",
					TransactionID:  "0x339c505eba389bf2c6bebb982cc30c6d82d0bd6a37521fa292890b6b180affc0",
					Information:    "0xca210f4121dc891c9154026c3ae3d1832a005048",
					Amount:         0.1383862742,
					Fee:            0.00005,
					Time:           types.Time(time.Unix(1790328722, 0)),
					Status:         "Settled",
					StatusProperty: "onhold",
					Originators: []string{
						"0x70b6343b104785574db2c1474b3acb3937ab5de7346a5b857a78ee26954e0e2d",
						"0x5b32f6f792904a446226b17f607850d0f2f7533cdc35845bfe432b5b99f55b66",
					},
				},
				{
					Method:         "Ether (Hex)",
					AssetClass:     "currency",
					Asset:          currency.XETH,
					ReferenceID:    "FTQcuak-V6Za8qrWnhzTx67yYHz8Tg",
					TransactionID:  "0x6544b41b607d8b2512baf801755a3a87b6890eacdb451be8a94059fb11f0a8d9",
					Information:    "0x8a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d",
					Amount:         0.78125,
					Time:           types.Time(time.Unix(1789899122, 0)),
					Status:         "Success",
					StatusProperty: "return",
				},
			},
		}
		assert.Equal(t, exp, result, "GetRecentDepositsStatus should decode every field of an unpaginated response")
	} else {
		assert.NotNil(t, result, "GetRecentDepositsStatus should return deposits")
	}

	result, err = e.GetRecentDepositsStatus(t.Context(), &RecentTransfersStatusRequest{Paginate: true, Limit: 1})
	require.NoError(t, err, "GetRecentDepositsStatus must not error for the first page")
	if mockTests {
		exp := &RecentDepositsStatusResponse{
			Deposits: []RecentDeposit{
				{
					Method:        "Bitcoin",
					AssetClass:    "currency",
					Asset:         currency.XXBT,
					ReferenceID:   "FTRbJ2s-Wc8pQ4rKmzVy56xTGa7Lh",
					TransactionID: "6544b41b607d8b2512baf801755a3a87b6890eacdb451be8a94059fb11f0a8d9",
					Information:   "2Myd4eaAW96ojk38A2uDK4FbioCayvkEgVq",
					Amount:        0.78125,
					Fee:           0.00001,
					Time:          types.Time(time.Unix(1791031122, 0)),
					Status:        "Success",
				},
			},
			NextCursor: "HgAAAAAAAABGVFJiSjJzLVdjOHBRNHJLbXpWeTU2eFRHYTdMaAEBAfYDAQEAAAABAAAAAAAAAAEAAAAAAAAAEAAAAAAAAAA=",
		}
		assert.Equal(t, exp, result, "GetRecentDepositsStatus should decode every field of the first page")
	}
	if result.NextCursor == "" {
		return
	}

	result, err = e.GetRecentDepositsStatus(t.Context(), &RecentTransfersStatusRequest{Cursor: result.NextCursor, Limit: 1})
	require.NoError(t, err, "GetRecentDepositsStatus must not error for the next page")
	if mockTests {
		exp := &RecentDepositsStatusResponse{
			Deposits: []RecentDeposit{
				{
					Method:        "Bitcoin",
					AssetClass:    "currency",
					Asset:         currency.XXBT,
					ReferenceID:   "FTQpL0z-Ab3cD5eFgHi7JkLmNoPq8",
					TransactionID: "9b2d0c5e3f1a7b4c8d6e2f0a1b3c5d7e9f1a2b4c6d8e0f1a3b5c7d9e1f2a4b6c",
					Information:   "bc1qm32pq7xz3ewt0j37s2g9kd5k6v0r8y4h2n7f3w",
					Amount:        0.125,
					Fee:           0.000005,
					Time:          types.Time(time.Unix(1790943522, 0)),
					Status:        "Pending",
				},
			},
			NextCursor: "HgAAAAAAAABGVFFwTDB6LUFiM2NENWVGZ0hpN0prTG1Ob1BxOAEBAfYDAQEAAAABAAAAAAAAAAEAAAAAAAAAEAAAAAAAAAA=",
		}
		assert.Equal(t, exp, result, "GetRecentDepositsStatus should decode every field of the next page")
		return
	}
	assert.NotNil(t, result, "GetRecentDepositsStatus should return the next page")
}

func TestRecentTransfersBody(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	_, err := recentTransfersBody(&RecentTransfersStatusRequest{Start: end, End: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "recentTransfersBody must reject a start after the end")

	for _, tc := range []struct {
		name string
		req  *RecentTransfersStatusRequest
		exp  map[string]any
	}{
		{name: "nil request", exp: map[string]any{}},
		{
			name: "every filter",
			req:  &RecentTransfersStatusRequest{Asset: currency.XBT, AssetClass: "currency", Method: "Bitcoin", Start: start, End: end, Limit: 50, RebaseMultiplier: "base"},
			exp:  map[string]any{"asset": "XBT", "aclass": "currency", "method": "Bitcoin", "start": "1788220800", "end": "1790812800", "limit": uint64(50), "rebase_multiplier": "base"},
		},
		{name: "first page", req: &RecentTransfersStatusRequest{Paginate: true}, exp: map[string]any{"cursor": true}},
		{name: "cursor beside paginate", req: &RecentTransfersStatusRequest{Paginate: true, Cursor: "next"}, exp: map[string]any{"cursor": "next"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body, err := recentTransfersBody(tc.req)
			require.NoError(t, err, "recentTransfersBody must not error")
			assert.Equal(t, tc.exp, body, "recentTransfersBody should encode the request as documented")
		})
	}
}

func TestRecentTransfersStatusResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, data := range []string{`[{"refid":"FTQcuak-V6Za8qrWnhzTx67yYHz8Tg"}]`, `{"deposit":[{"refid":"FTQcuak-V6Za8qrWnhzTx67yYHz8Tg"}]}`} {
		deposits := RecentDepositsStatusResponse{Deposits: []RecentDeposit{{Method: "Bitcoin"}, {Method: "Ether"}}, NextCursor: "stale"}
		require.NoErrorf(t, deposits.UnmarshalJSON([]byte(data)), "UnmarshalJSON must not error for deposits %s", data)
		exp := RecentDepositsStatusResponse{Deposits: []RecentDeposit{{ReferenceID: "FTQcuak-V6Za8qrWnhzTx67yYHz8Tg"}}}
		assert.Equalf(t, exp, deposits, "UnmarshalJSON should replace the whole response with deposits %s", data)
	}
	for _, data := range []string{`[{"refid":"BSNFZU2-MEFN4G-J3NEZV"}]`, `{"withdrawals":[{"refid":"BSNFZU2-MEFN4G-J3NEZV"}]}`} {
		withdrawals := RecentWithdrawalsStatusResponse{Withdrawals: []RecentWithdrawal{{Method: "Bitcoin"}, {Method: "Ether"}}, NextCursor: "stale"}
		require.NoErrorf(t, withdrawals.UnmarshalJSON([]byte(data)), "UnmarshalJSON must not error for withdrawals %s", data)
		exp := RecentWithdrawalsStatusResponse{Withdrawals: []RecentWithdrawal{{ReferenceID: "BSNFZU2-MEFN4G-J3NEZV"}}}
		assert.Equalf(t, exp, withdrawals, "UnmarshalJSON should replace the whole response with withdrawals %s", data)
	}
	for _, data := range []string{`[1]`, `{"deposit":{},"withdrawals":{}}`, `true`} {
		var deposits RecentDepositsStatusResponse
		assert.Errorf(t, deposits.UnmarshalJSON([]byte(data)), "UnmarshalJSON should reject deposits %s", data)
		var withdrawals RecentWithdrawalsStatusResponse
		assert.Errorf(t, withdrawals.UnmarshalJSON([]byte(data)), "UnmarshalJSON should reject withdrawals %s", data)
	}
}

func TestGetWithdrawalMethods(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *WithdrawalMethodsRequest
		exp  []WithdrawalMethod
	}{
		{
			name: "every method",
			exp: []WithdrawalMethod{
				{
					Asset:     currency.XXBT,
					Method:    "Bitcoin",
					MethodID:  "12fca2ad-edae-4d8c-acbb-4a424c1fbdeb",
					Network:   "Bitcoin",
					NetworkID: "ee9d686d-aeb6-4e61-9d83-448e3a7511f3",
					Minimum:   0.0004,
					Fee:       WithdrawalMethodFee{AssetClass: "currency", Asset: currency.XXBT, Fee: 0.000015},
					Limits: []WithdrawalLimit{
						{
							Description: "Maximum withdrawal amount",
							LimitType:   "amount",
							Windows: map[uint64]WithdrawalLimitWindow{
								86400:   {Maximum: 10, Remaining: 9.27515, Used: 0.72485},
								2592000: {Maximum: 100, Remaining: 98.5503, Used: 1.4497},
							},
						},
					},
				},
				{
					Asset:     currency.XXBT,
					Method:    "Bitcoin Lightning",
					MethodID:  "1ac8da36-8eec-4ed0-977a-82e4b50f2e49",
					Network:   "Lightning",
					NetworkID: "03ed9b12-29f1-4dc3-8d05-bd58e3798518",
					Minimum:   0.00001,
					Fee:       WithdrawalMethodFee{AssetClass: "currency", Asset: currency.XXBT, FeePercentage: 0.1},
					Limits: []WithdrawalLimit{
						{
							Description: "Maximum withdrawal count",
							LimitType:   "count",
							Windows:     map[uint64]WithdrawalLimitWindow{86400: {Maximum: 50, Remaining: 48, Used: 2}},
						},
					},
				},
			},
		},
		{
			name: "filtered",
			req:  &WithdrawalMethodsRequest{Asset: currency.XBT, AssetClass: "currency", Network: "Bitcoin", RebaseMultiplier: "rebased"},
			exp: []WithdrawalMethod{
				{
					Asset:     currency.XXBT,
					Method:    "Bitcoin",
					MethodID:  "12fca2ad-edae-4d8c-acbb-4a424c1fbdeb",
					Network:   "Bitcoin",
					NetworkID: "ee9d686d-aeb6-4e61-9d83-448e3a7511f3",
					Minimum:   0.0004,
					Fee:       WithdrawalMethodFee{AssetClass: "currency", Asset: currency.XXBT, Fee: 0.000015},
					Limits: []WithdrawalLimit{
						{
							Description: "Maximum withdrawal amount",
							LimitType:   "amount",
							Windows:     map[uint64]WithdrawalLimitWindow{86400: {Maximum: 10, Remaining: 9.27515, Used: 0.72485}},
						},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetWithdrawalMethods(t.Context(), tc.req)
			require.NoError(t, err, "GetWithdrawalMethods must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetWithdrawalMethods should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetWithdrawalMethods should return methods")
		})
	}
}

func TestGetWithdrawalAddresses(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *WithdrawalAddressesRequest
		exp  []WithdrawalAddress
	}{
		{
			name: "every address",
			exp: []WithdrawalAddress{
				{Address: "bc1qxdsh4sdd29h6ldehz0se5c61asq8cgwyjf2y3z", Asset: currency.XBT, Method: "Bitcoin", Key: "btc-wallet-1", Verified: true},
				{Address: "rLHzPsX3oXdzU2qP17kHCH2G4csZv1rAJh", Asset: currency.XRP, Method: "Ripple XRP", Key: "xrp-wallet-1", Tag: "1361101127"},
			},
		},
		{
			name: "filtered",
			req:  &WithdrawalAddressesRequest{Asset: currency.XBT, AssetClass: "currency", Method: "Bitcoin", Key: "btc-wallet-1", Verified: new(true)},
			exp: []WithdrawalAddress{
				{Address: "bc1qxdsh4sdd29h6ldehz0se5c61asq8cgwyjf2y3z", Asset: currency.XBT, Method: "Bitcoin", Key: "btc-wallet-1", Verified: true},
			},
		},
		{
			name: "unverified",
			req:  &WithdrawalAddressesRequest{Verified: new(false)},
			exp: []WithdrawalAddress{
				{Address: "rLHzPsX3oXdzU2qP17kHCH2G4csZv1rAJh", Asset: currency.XRP, Method: "Ripple XRP", Key: "xrp-wallet-1", Tag: "1361101127"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetWithdrawalAddresses(t.Context(), tc.req)
			require.NoError(t, err, "GetWithdrawalAddresses must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetWithdrawalAddresses should decode every field")
				return
			}
			assert.NotNil(t, result, "GetWithdrawalAddresses should return addresses")
		})
	}
}

func TestGetWithdrawalInformation(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawalInformation(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWithdrawalInformation must reject a nil request")
	_, err = e.GetWithdrawalInformation(t.Context(), &WithdrawalInformationRequest{Key: "btc_testnet", Amount: 0.725})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetWithdrawalInformation must reject an empty asset")
	_, err = e.GetWithdrawalInformation(t.Context(), &WithdrawalInformationRequest{Asset: currency.XBT, Amount: 0.725})
	require.ErrorIs(t, err, errWithdrawalKeyRequired, "GetWithdrawalInformation must reject an empty key")
	_, err = e.GetWithdrawalInformation(t.Context(), &WithdrawalInformationRequest{Asset: currency.XBT, Key: "btc_testnet"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "GetWithdrawalInformation must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWithdrawalInformation(t.Context(), &WithdrawalInformationRequest{Asset: currency.XBT, Key: "btc_testnet", Amount: 0.725})
	require.NoError(t, err, "GetWithdrawalInformation must not error")
	if mockTests {
		exp := &WithdrawalInformationResponse{Method: "Bitcoin", Limit: 332.00956139, Amount: 0.72485, Fee: 0.0002}
		assert.Equal(t, exp, result, "GetWithdrawalInformation should decode every field")
		return
	}
	assert.Positive(t, result.Amount.Float64(), "GetWithdrawalInformation should return the net amount")
}

func TestWithdrawFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFunds(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WithdrawFunds must reject a nil request")
	_, err = e.WithdrawFunds(t.Context(), &WithdrawFundsRequest{Key: "btc_2709", Amount: 0.725})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "WithdrawFunds must reject an empty asset")
	_, err = e.WithdrawFunds(t.Context(), &WithdrawFundsRequest{Asset: currency.XBT, Amount: 0.725})
	require.ErrorIs(t, err, errWithdrawalKeyRequired, "WithdrawFunds must reject an empty key")
	_, err = e.WithdrawFunds(t.Context(), &WithdrawFundsRequest{Asset: currency.XBT, Key: "btc_2709"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "WithdrawFunds must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.WithdrawFunds(t.Context(), &WithdrawFundsRequest{
		Asset:            currency.XBT,
		AssetClass:       "currency",
		Key:              "btc_2709",
		Address:          "bc1kar0ssrr7xf3vy5l6d3lydnwkre5og2zz3f5ldq",
		Amount:           0.725,
		MaxFee:           0.0002,
		RebaseMultiplier: "rebased",
	})
	require.NoError(t, err, "WithdrawFunds must not error")
	if mockTests {
		assert.Equal(t, &WithdrawFundsResponse{ReferenceID: "AGBSO6T-UFMTTQ-I7KGS6"}, result, "WithdrawFunds should decode every field")
		return
	}
	assert.NotEmpty(t, result.ReferenceID, "WithdrawFunds should return a reference ID")
}

func TestGetRecentWithdrawalsStatus(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	_, err := e.GetRecentWithdrawalsStatus(t.Context(), &RecentTransfersStatusRequest{Start: end, End: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetRecentWithdrawalsStatus must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRecentWithdrawalsStatus(t.Context(), &RecentTransfersStatusRequest{
		Asset:            currency.XBT,
		AssetClass:       "currency",
		Method:           "Bitcoin",
		Start:            start,
		End:              end,
		Limit:            2,
		RebaseMultiplier: "rebased",
	})
	require.NoError(t, err, "GetRecentWithdrawalsStatus must not error")
	if mockTests {
		exp := &RecentWithdrawalsStatusResponse{
			Withdrawals: []RecentWithdrawal{
				{
					Method:         "Bitcoin",
					Network:        "Bitcoin",
					AssetClass:     "currency",
					Asset:          currency.XXBT,
					ReferenceID:    "FTQcuak-V6Za8qrWnhzTx67yYHz8Tg",
					TransactionID:  "29323ce235cee8dae22503caba7a4a5a2ae8ad3a506879a03b1e87992923d804",
					Information:    "bc1qm32pq4zk7ldx3ewt0j37s2gk8r6dv5e3hq9ka0",
					Amount:         0.72485,
					Fee:            0.0002,
					Time:           types.Time(time.Unix(1790269786, 0)),
					Status:         "Pending",
					StatusProperty: "cancel-pending",
					Key:            "btc-wallet-1",
				},
				{
					Method:         "Bitcoin",
					Network:        "Bitcoin",
					AssetClass:     "currency",
					Asset:          currency.XXBT,
					ReferenceID:    "FTQcuak-V6Za8qrPnhsTx47yYLz8Tg",
					TransactionID:  "29323ce212ceb2daf81255cbea8a5e1fd2ad7a626471e05e1f82929501e82934",
					Information:    "bc1qa35lsx7r4d2tw0e9ufk63egf0872h3wq6n8yp5",
					Amount:         0.5,
					Fee:            0.00015,
					Time:           types.Time(time.Unix(1790270623, 0)),
					Status:         "Failure",
					StatusProperty: "canceled",
					Key:            "btc-wallet-2",
				},
			},
		}
		assert.Equal(t, exp, result, "GetRecentWithdrawalsStatus should decode every field of an unpaginated response")
	} else {
		assert.NotNil(t, result, "GetRecentWithdrawalsStatus should return withdrawals")
	}

	result, err = e.GetRecentWithdrawalsStatus(t.Context(), &RecentTransfersStatusRequest{Paginate: true, Limit: 1})
	require.NoError(t, err, "GetRecentWithdrawalsStatus must not error for the first page")
	if mockTests {
		exp := &RecentWithdrawalsStatusResponse{
			Withdrawals: []RecentWithdrawal{
				{
					Method:        "Tether USD (TRC20)",
					Network:       "Tron",
					AssetClass:    "currency",
					Asset:         currency.USDT,
					ReferenceID:   "BSNFZU2-MEFN4G-J3NEZV",
					TransactionID: "1c7a642fb7387bbc2c6a2c509fd1ae146937f4cf793b4079a4f0715e3a02615a",
					Information:   "TQmdxSuC16EhFg8FZWtYgrfFRosoRF7bCp",
					Amount:        1996.5,
					Fee:           2.5,
					Time:          types.Time(time.Unix(1791421057, 0)),
					Status:        "Success",
					Key:           "poloniex",
				},
			},
			NextCursor: "HgAAAAAAAABGVFRSd3k1LVF4Y0JQY05Gd0xRY0NxenFndHpybkwBAQH2AwEBAAAAAQAAAAAAAAABAAAAAAAZAAAAAAAAAA==",
		}
		assert.Equal(t, exp, result, "GetRecentWithdrawalsStatus should decode every field of the first page")
	}
	if result.NextCursor == "" {
		return
	}

	result, err = e.GetRecentWithdrawalsStatus(t.Context(), &RecentTransfersStatusRequest{Cursor: result.NextCursor, Limit: 1})
	require.NoError(t, err, "GetRecentWithdrawalsStatus must not error for the next page")
	if mockTests {
		exp := &RecentWithdrawalsStatusResponse{
			Withdrawals: []RecentWithdrawal{
				{
					Method:         "Ether",
					Network:        "Ethereum",
					AssetClass:     "currency",
					Asset:          currency.XETH,
					ReferenceID:    "A2BF34S-O7LBNQ-UE4Y4O",
					TransactionID:  "0x288b83c6b0904d8400ef44e1c9e2187b5c8f7ea3d838222d53f701a15b5c274d",
					Information:    "0x7cb275a5e07ba943fee972e165d80daa67cb2dd0",
					Amount:         9.995,
					Fee:            0.005,
					Time:           types.Time(time.Unix(1791416477, 0)),
					Status:         "Success",
					StatusProperty: "onhold",
					Key:            "eth-wallet-1",
				},
			},
			NextCursor: "HgAAAAAAAABCU05GWlUyLU1FRk40Ry1KM05FWlYBAQH2AwEBAAAAAQAAAAAAAAABAAAAAAAaAAAAAAAAAA==",
		}
		assert.Equal(t, exp, result, "GetRecentWithdrawalsStatus should decode every field of the next page")
		return
	}
	assert.NotNil(t, result, "GetRecentWithdrawalsStatus should return the next page")
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.CancelWithdrawal(t.Context(), currency.EMPTYCODE, "FTQcuak-V6Za8qrWnhzTx67yYHz8Tg")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelWithdrawal must reject an empty asset")
	_, err = e.CancelWithdrawal(t.Context(), currency.XBT, "")
	require.ErrorIs(t, err, errWithdrawalReferenceIDRequired, "CancelWithdrawal must reject an empty reference ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelWithdrawal(t.Context(), currency.XBT, "FTQcuak-V6Za8qrWnhzTx67yYHz8Tg")
	require.NoError(t, err, "CancelWithdrawal must not error")
	assert.True(t, result, "CancelWithdrawal should report the cancellation succeeded")
}

func TestWalletTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.WalletTransfer(t.Context(), currency.EMPTYCODE, 2.54)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "WalletTransfer must reject an empty asset")
	_, err = e.WalletTransfer(t.Context(), currency.XBT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "WalletTransfer must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.WalletTransfer(t.Context(), currency.XBT, 2.54)
	require.NoError(t, err, "WalletTransfer must not error")
	if mockTests {
		assert.Equal(t, &WalletTransferResponse{ReferenceID: "FTQcuak-V6Za8qrWnhzTx67yYHz8Tg"}, result, "WalletTransfer should decode every field")
		return
	}
	assert.NotEmpty(t, result.ReferenceID, "WalletTransfer should return a reference ID")
}
