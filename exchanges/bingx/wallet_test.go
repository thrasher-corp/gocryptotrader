package bingx

import (
	"io"
	"net/http"
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

func TestGetDepositRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDepositRecords must reject a nil request")
	_, err = e.GetDepositRecords(t.Context(), &DepositRecordsRequest{StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDepositRecords must reject a reversed time range")

	t.Run("in progress status", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "0", r.URL.Query().Get("status"), "a set in progress status should be sent as 0")
			_, _ = io.WriteString(w, `[]`)
		})
		result, err := ex.GetDepositRecords(t.Context(), &DepositRecordsRequest{Status: new(DepositStatusInProgress)})
		require.NoError(t, err, "GetDepositRecords must not error")
		assert.Empty(t, result, "GetDepositRecords should decode an empty list")
	})

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetDepositRecords(t.Context(), &DepositRecordsRequest{
			Coin:      currency.NewCode("USDTTRC20"),
			Status:    new(DepositStatusCompleted),
			StartTime: time.UnixMilli(1700894588000),
			EndTime:   time.UnixMilli(1702622588000),
			Offset:    1,
			Limit:     1000,
			TxID:      "60*****1d",
		})
		require.NoError(t, err, "GetDepositRecords must not error")
		if mockTests {
			exp := []DepositRecordResponse{
				{
					Amount:        49999,
					Coin:          currency.NewCode("USDTTRC20"),
					Network:       "TRC20",
					Status:        DepositStatusCompleted,
					Address:       "TP******B4v",
					TxID:          "60*****1d",
					InsertTime:    types.Time(time.UnixMilli(1701557778000)),
					UnlockConfirm: "2/2",
					ConfirmTimes:  "2/2",
				},
			}
			assert.Equal(t, exp, result, "GetDepositRecords should decode every field")
		}
	})
}

func TestGetWithdrawRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWithdrawRecords must reject a nil request")
	_, err = e.GetWithdrawRecords(t.Context(), &WithdrawRecordsRequest{StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetWithdrawRecords must reject a reversed time range")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawRecords(t.Context(), &WithdrawRecordsRequest{
		ID:              "125*****98",
		Coin:            currency.NewCode("USDTTRC20"),
		WithdrawOrderID: "abc123",
		Status:          WithdrawStatusCompleted,
		StartTime:       time.UnixMilli(1702450164000),
		EndTime:         time.UnixMilli(1702536564000),
		Offset:          1,
		Limit:           1000,
		TxID:            "b9***********b67",
	})
	require.NoError(t, err, "GetWithdrawRecords must not error")
	if mockTests {
		require.Len(t, result, 1, "GetWithdrawRecords must return the documented record")
		// The decoded time keeps the record's UTC+8 offset, which time.Local may or may not share
		result[0].ApplyTime = result[0].ApplyTime.UTC()
		exp := []WithdrawRecordResponse{
			{
				Address:        "TR****zc",
				Amount:         3500,
				ApplyTime:      time.Date(2023, 12, 13, 20, 5, 2, 0, time.UTC),
				Coin:           currency.NewCode("USDTTRC20"),
				ID:             "125*****98",
				Network:        "TRC20",
				TransactionFee: 1,
				ConfirmNumber:  2,
				TxID:           "b9***********b67",
				TransferType:   TransferTypeWithdrawal,
			},
		}
		assert.Equal(t, exp, result, "GetWithdrawRecords should decode every field")
	}
}

func TestGetCurrencyConfig(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCurrencyConfig(t.Context(), currency.BTC, "BTC")
	require.NoError(t, err, "GetCurrencyConfig must not error")
	if mockTests {
		exp := []CurrencyConfigResponse{
			{
				Coin: currency.BTC,
				Name: "BTC",
				NetworkList: []CurrencyNetwork{
					{
						Name:            "BTC",
						Network:         "BTC",
						IsDefault:       true,
						MinimumConfirm:  2,
						WithdrawEnable:  true,
						DepositEnable:   true,
						WithdrawFee:     0.0006,
						WithdrawMaximum: 1.17522,
						WithdrawMinimum: 0.0005,
						DepositMinimum:  0.0002,
					},
					{
						Name:            "BTC",
						Network:         "BEP20",
						MinimumConfirm:  15,
						WithdrawEnable:  true,
						DepositEnable:   true,
						WithdrawFee:     0.0000066,
						WithdrawMaximum: 1.17522,
						WithdrawMinimum: 0.0000066,
						DepositMinimum:  0.0002,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetCurrencyConfig should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetCurrencyConfig should return a coin")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *WithdrawRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty coin", req: &WithdrawRequest{Address: "0x8****11", Amount: 1, WalletType: WalletTypeFund}, err: currency.ErrCurrencyCodeEmpty},
		{name: "empty address", req: &WithdrawRequest{Coin: currency.USDT, Amount: 1, WalletType: WalletTypeFund}, err: common.ErrAddressIsEmptyOrInvalid},
		{name: "zero amount", req: &WithdrawRequest{Coin: currency.USDT, Address: "0x8****11", WalletType: WalletTypeFund}, err: limits.ErrAmountBelowMin},
		{name: "no wallet type", req: &WithdrawRequest{Coin: currency.USDT, Address: "0x8****11", Amount: 1}, err: errWalletTypeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.Withdraw(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "Withdraw should return the validation error")
		})
	}

	t.Run("every parameter", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "method should be POST")
			assert.Equal(t, "/openApi/wallets/v1/capital/withdraw/apply", r.URL.Path, "path should be the endpoint's")
			q := r.URL.Query()
			for k, v := range map[string]string{
				"coin":               "USDT",
				"network":            "TRC20",
				"address":            "TR****zc",
				"addressTag":         "memo 1",
				"amount":             "12.5",
				"walletType":         "15",
				"withdrawOrderId":    "abc123",
				"vaspEntityId":       "Upbit",
				"recipientLastName":  "Kim",
				"recipientFirstName": "Minsu",
				"dateOfbirth":        "1999-09-09",
			} {
				assert.Equalf(t, v, q.Get(k), "%s should be sent", k)
			}
			_, _ = io.WriteString(w, `{"code":0,"timestamp":1702619168218,"data":{"id":"125*****4","withdrawOrderId":"abc123"}}`)
		})
		result, err := ex.Withdraw(t.Context(), &WithdrawRequest{
			Coin:               currency.NewCode("usdt"),
			Network:            "TRC20",
			Address:            "TR****zc",
			AddressTag:         "memo 1",
			Amount:             12.5,
			WalletType:         WalletTypeSpot,
			WithdrawOrderID:    "abc123",
			VASPEntityID:       VASPEntityIDUpbit,
			RecipientLastName:  "Kim",
			RecipientFirstName: "Minsu",
			DateOfBirth:        time.Date(1999, 9, 9, 0, 30, 0, 0, time.FixedZone("KST", 9*60*60)),
		})
		require.NoError(t, err, "Withdraw must not error")
		assert.Equal(t, &WithdrawResponse{ID: "125*****4", WithdrawOrderID: "abc123"}, result, "Withdraw should decode every field")
	})

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		result, err := e.Withdraw(t.Context(), &WithdrawRequest{
			Coin:            currency.USDT,
			Network:         "BEP20",
			Address:         "0x8****11",
			Amount:          4998,
			WalletType:      WalletTypeFund,
			WithdrawOrderID: "abc123",
		})
		require.NoError(t, err, "Withdraw must not error")
		require.NotNil(t, result, "Withdraw must return a result")
		if mockTests {
			assert.Equal(t, &WithdrawResponse{ID: "125*****4"}, result, "Withdraw should decode the documented example")
		}
	})
}

func TestGetMainAccountDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetMainAccountDepositAddress(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMainAccountDepositAddress must reject a nil request")
	_, err = e.GetMainAccountDepositAddress(t.Context(), &MainAccountDepositAddressRequest{Limit: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMainAccountDepositAddress must reject an empty coin")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMainAccountDepositAddress(t.Context(), &MainAccountDepositAddressRequest{Coin: currency.USDT, Offset: 1, Limit: 1000})
	require.NoError(t, err, "GetMainAccountDepositAddress must not error")
	require.NotNil(t, result, "GetMainAccountDepositAddress must return a result")
	if mockTests {
		exp := &DepositAddressListResponse{
			Data: []DepositAddress{
				{CoinID: 760, Coin: currency.USDT, Network: "ERC20", Address: "40e*****95"},
				{CoinID: 780, Coin: currency.USDT, Network: "TRC20", Address: "40e*****95"},
				{CoinID: 799, Coin: currency.USDT, Network: "BEP20", Address: "40e*****95"},
				{CoinID: 857, Coin: currency.USDT, Network: "SOL", Address: "40e*****95"},
				{CoinID: 1192, Coin: currency.USDT, Network: "POLYGON", Address: "40e*****95"},
				{CoinID: 1367, Coin: currency.USDT, Network: "ARBITRUM", Address: "40e*****95"},
				{CoinID: 1371, Coin: currency.USDT, Network: "OPTIMISM", Address: "40e*****95"},
			},
			Total: 7,
		}
		assert.Equal(t, exp, result, "GetMainAccountDepositAddress should decode every field")
	}
}

func TestGetDepositRiskControlRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDepositRiskControlRecords(t.Context())
	require.NoError(t, err, "GetDepositRiskControlRecords must not error")
	if mockTests {
		// The documented example holds one record with every field empty
		assert.Equal(t, []DepositRiskControlRecordResponse{{}}, result, "GetDepositRiskControlRecords should decode the documented example")
	}
}
