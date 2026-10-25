package lbank

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// mustPreciseNumber returns the PreciseNumber of a numeric string
func mustPreciseNumber(t *testing.T, s string) types.PreciseNumber {
	t.Helper()
	n, err := types.NewPreciseNumberFromString(s)
	require.NoErrorf(t, err, "NewPreciseNumberFromString must not error for %q", s)
	return n
}

func TestGetContractServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetContractServerTime(t.Context())
	require.NoError(t, err, "GetContractServerTime must not error")
	if mockTests {
		exp := time.UnixMilli(1792890789640)
		assert.Equal(t, exp, result, "GetContractServerTime should decode the server time")
		return
	}
	assert.WithinDuration(t, time.Now(), result, time.Minute, "GetContractServerTime should return the current time")
}

func TestGetContractInstruments(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractInstruments(t.Context(), "")
	require.ErrorIs(t, err, errProductGroupRequired, "GetContractInstruments must require a product group")

	result, err := e.GetContractInstruments(t.Context(), ProductGroupUSDTSwap)
	require.NoError(t, err, "GetContractInstruments must not error")
	if mockTests {
		btc, eth, usdt := currency.NewCode("BTC"), currency.NewCode("ETH"), currency.NewCode("USDT")
		exp := []ContractInstrumentResponse{
			{
				Symbol: "BTCUSDT", SymbolName: "BTCUSDT", SymbolAlias: "BTC", ExchangeID: "Exchange", BaseCurrency: btc, PriceCurrency: usdt, ClearCurrency: usdt,
				PriceTick: 0.1, VolumeTick: 0.0001, VolumeMultiple: 1, MinimumOrderVolume: 0.0001, MaximumOrderVolume: 150, MinimumOrderCost: 1,
				PriceLimitUpperValue: 0.05, PriceLimitLowerValue: 0.05, IndexPrice: 80796.6, DefaultLeverage: 20, MaximumLeverage: 200,
			},
			{
				Symbol: "ETHUSDT", SymbolName: "ETHUSDT", SymbolAlias: "ETH", ExchangeID: "Exchange", BaseCurrency: eth, PriceCurrency: usdt, ClearCurrency: usdt,
				PriceTick: 0.01, VolumeTick: 0.001, VolumeMultiple: 1, MinimumOrderVolume: 0.001, MaximumOrderVolume: 1000, MinimumOrderCost: 5,
				PriceLimitUpperValue: 0.05, PriceLimitLowerValue: 0.05, IndexPrice: 2356.98, DefaultLeverage: 20, MaximumLeverage: 200,
			},
		}
		assert.Equal(t, exp, result, "GetContractInstruments should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetContractInstruments should return contracts")
}

func TestGetContractMarketData(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractMarketData(t.Context(), "")
	require.ErrorIs(t, err, errProductGroupRequired, "GetContractMarketData must require a product group")

	result, err := e.GetContractMarketData(t.Context(), ProductGroupUSDTSwap)
	require.NoError(t, err, "GetContractMarketData must not error")
	if mockTests {
		exp := []ContractMarketDataResponse{
			{
				Symbol: "ETHUSDT", InstrumentStatus: "2", LastPrice: 2355.9, MarkPrice: 2355.9, UnderlyingPrice: 2356.98, OpenPrice24Hour: 2376.3,
				HighPrice24Hour: 2389.44, LowPrice24Hour: 2281.59, Volume24Hour: 104654.774, Turnover24Hour: 244946973.57545501,
				FundingRate: mustPreciseNumber(t, "0.00004605"), PositionFeeRate: mustPreciseNumber(t, "0.00004605"), FundingIntervalSeconds: 28800,
				NextFundingTime: types.Time(time.UnixMilli(1792915200000)), LastTime: types.Time(time.Unix(1792890789, 0)),
			},
			{
				Symbol: "BTCUSDT", InstrumentStatus: "2", LastPrice: 80766, MarkPrice: 80766, UnderlyingPrice: 80796.6, OpenPrice24Hour: 80710,
				HighPrice24Hour: 81073.2, LowPrice24Hour: 79195.9, Volume24Hour: 3718.8446, Turnover24Hour: 298925536.6276579,
				FundingRate: mustPreciseNumber(t, "0.00009783"), PositionFeeRate: mustPreciseNumber(t, "0.00009783"), FundingIntervalSeconds: 28800,
				NextFundingTime: types.Time(time.UnixMilli(1792915200000)), LastTime: types.Time(time.Unix(1792890789, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetContractMarketData should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetContractMarketData should return contracts")
}

func TestGetContractOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractOrderbook(t.Context(), futuresTestPair, 0)
	require.ErrorIs(t, err, errLimitOutOfRange, "GetContractOrderbook must require a depth")
	_, err = e.GetContractOrderbook(t.Context(), currency.EMPTYPAIR, 5)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetContractOrderbook must require a pair")

	result, err := e.GetContractOrderbook(t.Context(), futuresTestPair, 5)
	require.NoError(t, err, "GetContractOrderbook must not error")
	if mockTests {
		exp := &ContractOrderbookResponse{
			Symbol: "BTCUSDT",
			Asks: []ContractOrderbookLevel{
				{Price: 80766.1, Volume: 26.0079, OrderCount: 2},
				{Price: 80766.2, Volume: 0.4896, OrderCount: 1},
				{Price: 80766.3, Volume: 0.9742, OrderCount: 1},
				{Price: 80766.4, Volume: 1.0028, OrderCount: 1},
				{Price: 80766.5, Volume: 0.6086, OrderCount: 1},
			},
			Bids: []ContractOrderbookLevel{
				{Price: 80766, Volume: 21.944, OrderCount: 1},
				{Price: 80765.9, Volume: 0.3735, OrderCount: 1},
				{Price: 80765.8, Volume: 0.522, OrderCount: 1},
				{Price: 80765.7, Volume: 0.3777, OrderCount: 1},
				{Price: 80765.6, Volume: 0.372, OrderCount: 1},
			},
		}
		assert.Equal(t, exp, result, "GetContractOrderbook should decode the order book")
		return
	}
	assert.Equal(t, "BTCUSDT", result.Symbol, "GetContractOrderbook should return the contract asked for")
}
