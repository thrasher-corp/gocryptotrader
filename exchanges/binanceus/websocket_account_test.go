package binanceus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

func TestWebsocketAPIGetAccount(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true)
	result, err := ex.WebsocketAPI().GetAccount(t.Context())
	require.NoError(t, err, "GetAccount must not error")
	require.NotNil(t, result, "GetAccount must return a result")
	if !mockTests {
		assert.NotEmpty(t, result.AccountType, "GetAccount should return the account type")
		return
	}
	exp := &AccountResponse{
		MakerCommissionInBasisPoints: 15,
		TakerCommissionInBasisPoints: 15,
		CommissionRates:              CommissionRates{Maker: 0.0015, Taker: 0.0015},
		CanTrade:                     true,
		CanWithdraw:                  true,
		CanDeposit:                   true,
		UpdateTime:                   types.Time(time.UnixMilli(1660801833000)),
		AccountType:                  "SPOT",
		Balances: []Balance{
			{Asset: currency.BNB, Free: decimal.Zero, Locked: decimal.Zero},
			{Asset: currency.BTC, Free: decimal.MustFromString("1.3447112"), Locked: decimal.MustFromString("0.086")},
			{Asset: currency.USDT, Free: decimal.MustFromString("1021.21"), Locked: decimal.Zero},
		},
		Permissions: []Permission{PermissionSpot},
		UID:         123456789,
	}
	require.Len(t, result.Balances, len(exp.Balances), "GetAccount must return every balance")
	for i := range exp.Balances {
		assert.Truef(t, exp.Balances[i].Free.Equal(result.Balances[i].Free), "GetAccount should decode balance %d's free amount exactly", i)
		assert.Truef(t, exp.Balances[i].Locked.Equal(result.Balances[i].Locked), "GetAccount should decode balance %d's locked amount exactly", i)
		exp.Balances[i].Free, exp.Balances[i].Locked = result.Balances[i].Free, result.Balances[i].Locked
	}
	assert.Equal(t, exp, result, "GetAccount should decode every field")
}

func TestWebsocketAPIGetMyFilters(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true)
	result, err := ex.WebsocketAPI().GetMyFilters(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "GetMyFilters must not error")
	require.NotNil(t, result, "GetMyFilters must return a result")
	if !mockTests {
		return
	}
	exp := &MyFiltersResponse{
		ExchangeFilters: []ExchangeFilter{{FilterType: "EXCHANGE_MAX_NUM_ORDERS", MaxNumberOrders: 1000}},
		SymbolFilters: []SymbolFilter{
			{FilterType: "MAX_NUM_ORDER_LISTS", MaxNumberOrderLists: 20},
			{FilterType: "NOTIONAL", MinNotional: 10, ApplyMinToMarket: true, MaxNotional: 10000, ApplyMaxToMarket: true, AveragePriceMinutes: 5},
			{FilterType: "MAX_NUM_ICEBERG_ORDERS", MaxNumberIcebergOrders: 5},
			{FilterType: "MAX_POSITION", MaxPosition: 10},
		},
		AssetFilters: []AssetFilter{{FilterType: "MAX_ASSET", Asset: currency.USD, Limit: 1000000}},
	}
	assert.Equal(t, exp, result, "GetMyFilters should decode every field")
}
