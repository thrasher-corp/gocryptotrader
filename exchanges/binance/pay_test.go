package binance

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

func TestGetPayTradeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetPayTradeHistory(t.Context(), endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPayTradeHistory must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPayTradeHistory(t.Context(), startTime, endTime, 10)
	require.NoError(t, err, "GetPayTradeHistory must not error")
	if mockTests {
		exp := &PayTradeHistoryResponse{
			Code:    "000000",
			Message: "success",
			Data: []PayTrade{
				{
					OrderType:       "C2C",
					TransactionID:   "M_P_71505104267788288",
					TransactionTime: types.Time(time.Unix(1744150000, 0)),
					Amount:          1.2,
					Currency:        currency.USDT,
					WalletType:      1,
					WalletTypes: []uint64{
						1,
						2,
					},
					FundsDetail: []PayFundsDetail{
						{
							Currency: currency.USDT,
							Amount:   1.2,
							WalletAssetCost: map[uint64]types.Number{
								1: 0.6,
								2: 0.6,
							},
						},
					},
					PayerInfo: PayPayerInfo{
						Name:      "Jack",
						Type:      "USER",
						BinanceID: "12345678",
					},
					ReceiverInfo: PayReceiverInfo{
						Name:        "Alan",
						Type:        "USER",
						Email:       "test@example.com",
						BinanceID:   "34355667",
						AccountID:   "21326891",
						CountryCode: "1",
						PhoneNumber: "2025550123",
						MobileCode:  "US",
					},
				},
				{
					OrderType:       "REMITTANCE",
					TransactionID:   "M_R_71505104267788289",
					TransactionTime: types.Time(time.Unix(1744160000, 0)),
					Amount:          -50,
					Currency:        currency.USDT,
					WalletType:      2,
					WalletTypes: []uint64{
						2,
					},
					FundsDetail: []PayFundsDetail{
						{
							Currency: currency.USDT,
							Amount:   50,
							WalletAssetCost: map[uint64]types.Number{
								2: 50,
							},
						},
					},
					PayerInfo: PayPayerInfo{
						Type:      "USER",
						BinanceID: "12345678",
					},
					ReceiverInfo: PayReceiverInfo{
						Name: "Alan",
						Type: "USER",
						Extend: PayReceiverExtend{
							InstitutionName: "Example Bank",
							CardNumber:      "4111********1111",
							DigitalWalletID: "wallet-0001",
						},
					},
				},
			},
			Success: true,
		}
		assert.Equal(t, exp, result, "GetPayTradeHistory should decode every field")
		return
	}
	assert.True(t, result.Success, "GetPayTradeHistory should succeed")
}
