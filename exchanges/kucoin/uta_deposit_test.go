package kucoin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetUTADepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTADepositAddress(t.Context(), currency.EMPTYCODE, "trx")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUTADepositAddress must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	tron := UTADepositAddress{
		Currency:        currency.USDT,
		Chain:           "trx",
		ChainName:       "TRC20",
		Address:         "TLyiW7gFiykd2AK8qMui959d3rhcbyWrjZ",
		ToAccountType:   "FUNDING",
		ContractAddress: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
	}
	for _, tc := range []struct {
		chain string
		exp   []UTADepositAddress
	}{
		{
			exp: []UTADepositAddress{
				tron,
				{
					Currency:        currency.USDT,
					Chain:           "eth",
					ChainName:       "ERC20",
					Address:         "0xa39525c83bd1d45c866a464821050d9097194498",
					Memo:            "1234567",
					Remark:          "Default address",
					ToAccountType:   "UNIFIED",
					ContractAddress: "0xdac17f958d2ee523a2206206994597c13d831ec7",
					ExpirationDate:  types.Time(time.UnixMilli(1801550893000)),
				},
			},
		},
		{chain: "trx", exp: []UTADepositAddress{tron}},
	} {
		t.Run("chain "+tc.chain, func(t *testing.T) {
			t.Parallel()
			if !mockTests && tc.chain != "" {
				// Asking for a chain's address creates it when the account has none yet
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
			}
			result, err := e.GetUTADepositAddress(t.Context(), currency.USDT, tc.chain)
			require.NoError(t, err, "GetUTADepositAddress must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTADepositAddress should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUTADepositAddress should return a result")
		})
	}
}

func TestGetUTADepositHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTADepositHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTADepositHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	confirmed := UTADepositRecord{
		ID:                  "6a4f4955b583dd0007255c64",
		Currency:            currency.USDT,
		Chain:               "eth",
		Status:              "SUCCESS",
		Address:             "0xa39525c83bd1d45c866a464821050d9097194498",
		Amount:              4.6,
		WalletTransactionID: "0xfb7fd7eb0f717071619a255e220e7a26e93d2de951ed4e5abe4a4edd3f597b0c",
		CreatedAt:           types.Time(time.UnixMilli(1783581013000)),
		UpdatedAt:           types.Time(time.UnixMilli(1783581763000)),
		Remark:              "Deposit",
		StatusRemark:        "Deposit confirmed. Funds have been credited.",
		PreConfirms:         12,
		Confirms:            64,
		CurrentConfirms:     64,
	}
	for _, tc := range []struct {
		name string
		req  *UTAAssetHistoryRequest
		exp  *UTADepositHistoryResponse
	}{
		{
			name: "all",
			req:  &UTAAssetHistoryRequest{},
			exp: &UTADepositHistoryResponse{
				CurrentPage: 1,
				PageSize:    10,
				TotalNumber: 2,
				TotalPage:   1,
				Items: []UTADepositRecord{
					confirmed,
					{
						ID:                  "6a4f4955b583dd0007255c65",
						Currency:            currency.USDT,
						Chain:               "trx",
						Status:              "PROCESSING",
						Address:             "TLyiW7gFiykd2AK8qMui959d3rhcbyWrjZ",
						Memo:                "1234567",
						IsInner:             true,
						Amount:              10,
						Fee:                 1,
						WalletTransactionID: "4c5b6a7d8e9f0a1b2c3d4e5f60718293a4b5c6d7e8f9012a3b4c5d6e7f809123",
						CreatedAt:           types.Time(time.UnixMilli(1783582013000)),
						UpdatedAt:           types.Time(time.UnixMilli(1783582063000)),
						Remark:              "Deposit",
						Arrears:             true,
						URL:                 "https://tronscan.org/#/transaction/4c5b6a7d8e9f0a1b2c3d4e5f60718293a4b5c6d7e8f9012a3b4c5d6e7f809123",
						StatusRemark:        "Waiting for block confirmations.",
						PreConfirms:         1,
						Confirms:            20,
						CurrentConfirms:     5,
					},
				},
			},
		},
		{
			name: "filtered",
			req: &UTAAssetHistoryRequest{
				Currency:    currency.USDT,
				ID:          "6a4f4955b583dd0007255c64",
				Status:      "SUCCESS",
				StartAt:     time.UnixMilli(1783500000000),
				EndAt:       time.UnixMilli(1783600000000),
				CurrentPage: 1,
				PageSize:    50,
			},
			exp: &UTADepositHistoryResponse{CurrentPage: 1, PageSize: 50, TotalNumber: 1, TotalPage: 1, Items: []UTADepositRecord{confirmed}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTADepositHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetUTADepositHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTADepositHistory should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUTADepositHistory should return a result")
		})
	}
}

func TestUTAAssetHistoryRequestValues(t *testing.T) {
	t.Parallel()
	_, err := (*UTAAssetHistoryRequest)(nil).values()
	require.ErrorIs(t, err, common.ErrNilPointer, "values must reject a nil request")
	_, err = (&UTAAssetHistoryRequest{StartAt: time.UnixMilli(1783600000000), EndAt: time.UnixMilli(1783500000000)}).values()
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "values must reject a start after the end")

	params, err := (&UTAAssetHistoryRequest{}).values()
	require.NoError(t, err, "values must not error for an empty request")
	assert.Empty(t, params, "values should set no parameters for an empty request")

	params, err = (&UTAAssetHistoryRequest{
		Currency:    currency.NewCode("usdt"),
		ID:          "6a4f4955b583dd0007255c64",
		Status:      "SUCCESS",
		StartAt:     time.UnixMilli(1783500000000),
		EndAt:       time.UnixMilli(1783600000000),
		CurrentPage: 2,
		PageSize:    50,
	}).values()
	require.NoError(t, err, "values must not error")
	assert.Equal(t, "currency=USDT&currentPage=2&endAt=1783600000000&id=6a4f4955b583dd0007255c64&pageSize=50&startAt=1783500000000&status=SUCCESS", params.Encode(), "values should set every given parameter")
}
