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

func TestAddDepositAddressV3(t *testing.T) {
	t.Parallel()
	_, err := e.AddDepositAddressV3(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddDepositAddressV3 must reject a nil request")
	_, err = e.AddDepositAddressV3(t.Context(), &AddDepositAddressV3Request{Chain: "ton"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "AddDepositAddressV3 must reject an empty currency")
	_, err = e.AddDepositAddressV3(t.Context(), &AddDepositAddressV3Request{Currency: currency.NewCode("TON")})
	require.ErrorIs(t, err, errChainRequired, "AddDepositAddressV3 must reject an empty chain")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *AddDepositAddressV3Request
		exp  *AddDepositAddressV3Response
	}{
		{
			name: "memo",
			req:  &AddDepositAddressV3Request{Currency: currency.NewCode("TON"), Chain: "ton", To: "trade"},
			exp: &AddDepositAddressV3Response{
				Address:   "***",
				Memo:      "2085123456",
				ChainID:   "ton",
				To:        "TRADE",
				Currency:  currency.NewCode("TON"),
				ChainName: "TON",
				Remark:    "TON deposits need the memo",
			},
		},
		{
			name: "lightning",
			req:  &AddDepositAddressV3Request{Currency: currency.BTC, Chain: "btcln", Amount: 0.001},
			exp: &AddDepositAddressV3Response{
				Address:        "ln***",
				ChainID:        "btcln",
				To:             "MAIN",
				ExpirationDate: types.Time(time.UnixMilli(1770641912000)),
				Currency:       currency.BTC,
				ChainName:      "Lightning Network",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.AddDepositAddressV3(t.Context(), tc.req)
			require.NoError(t, err, "AddDepositAddressV3 must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "AddDepositAddressV3 should decode every field")
				return
			}
			require.NotNil(t, result, "AddDepositAddressV3 must return a result")
			assert.NotEmpty(t, result.Address, "AddDepositAddressV3 should return an address")
		})
	}
}

func TestGetDepositAddressV3(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddressV3(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddressV3 must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name  string
		ccy   currency.Code
		chain string
		exp   []DepositAddressResponse
	}{
		{
			name: "every chain",
			ccy:  currency.USDT,
			exp: []DepositAddressResponse{
				{
					Address:         "***",
					ChainID:         "trx",
					To:              "TRADE",
					Currency:        currency.USDT,
					ContractAddress: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
					ChainName:       "TRC20",
				},
				{
					Address:         "***",
					Memo:            "2085123456",
					ChainID:         "ton",
					To:              "MAIN",
					Currency:        currency.USDT,
					ContractAddress: "EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs",
					ChainName:       "TON",
					Remark:          "USDT on TON needs the memo",
				},
			},
		},
		{
			name:  "one chain",
			ccy:   currency.BTC,
			chain: "btcln",
			exp: []DepositAddressResponse{
				{
					Address:        "ln***",
					ChainID:        "btcln",
					To:             "MAIN",
					Currency:       currency.BTC,
					ChainName:      "Lightning Network",
					ExpirationDate: types.Time(time.UnixMilli(1770641912000)),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetDepositAddressV3(t.Context(), tc.ccy, tc.chain)
			require.NoError(t, err, "GetDepositAddressV3 must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetDepositAddressV3 should decode every field")
				return
			}
			for i := range result {
				assert.Truef(t, result[i].Currency.Equal(tc.ccy), "GetDepositAddressV3 should return addresses of %s", tc.ccy)
			}
		})
	}
}

func TestGetDepositHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDepositHistory must reject a nil request")
	_, err = e.GetDepositHistory(t.Context(), &DepositHistoryRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositHistory must reject an empty currency")
	_, err = e.GetDepositHistory(t.Context(), &DepositHistoryRequest{Currency: currency.USDT, StartAt: time.UnixMilli(1766000000000), EndAt: time.UnixMilli(1765900000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDepositHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *DepositHistoryRequest
		exp  *DepositHistoryResponse
	}{
		{
			name: "currency",
			req:  &DepositHistoryRequest{Currency: currency.USDT},
			exp: &DepositHistoryResponse{
				CurrentPage: 1,
				PageSize:    50,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []DepositRecord{
					{
						ID:        "69425929fc86870007ecd8b1",
						Currency:  currency.USDT,
						Chain:     "trx",
						Status:    "SUCCESS",
						Address:   "***",
						IsInner:   true,
						Amount:    3,
						CreatedAt: types.Time(time.UnixMilli(1765955881000)),
						UpdatedAt: types.Time(time.UnixMilli(1765955881000)),
						Remark:    "Study",
					},
				},
			},
		},
		{
			name: "every filter",
			req: &DepositHistoryRequest{
				Currency:    currency.USDT,
				Status:      "FAILURE",
				StartAt:     time.UnixMilli(1765900000000),
				EndAt:       time.UnixMilli(1766000000000),
				CurrentPage: 1,
				PageSize:    10,
			},
			exp: &DepositHistoryResponse{
				CurrentPage: 1,
				PageSize:    10,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []DepositRecord{
					{
						ID:            "69425356fc8d21000734603a",
						Currency:      currency.USDT,
						Chain:         "ton",
						Status:        "FAILURE",
						SubStatus:     "ROLLBACK",
						URL:           "https://tonviewer.com/transaction/5a4c3c6c9000d3c8292781a152d8cf603accb5a4f2f86d123f0ea45a393ae7fd",
						Address:       "***",
						Memo:          "2085123456",
						Amount:        1.5,
						Fee:           0.1,
						WalletTxID:    "5a4c3c6c9000d3c8292781a152d8cf603accb5a4f2f86d123f0ea45a393ae7fd",
						CreatedAt:     types.Time(time.UnixMilli(1765954390000)),
						UpdatedAt:     types.Time(time.UnixMilli(1765954450000)),
						Remark:        "Deposit",
						Arrears:       true,
						NeedSubmitWHT: "false",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetDepositHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetDepositHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetDepositHistory should decode every field")
				return
			}
			require.NotNil(t, result, "GetDepositHistory must return a result")
			assert.NotZero(t, result.PageSize, "GetDepositHistory should return a page size")
		})
	}
}
