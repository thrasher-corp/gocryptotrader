package kucoin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetCrossMarginSymbols(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		symbol string
		exp    *CrossMarginSymbolsResponse
	}{
		{
			name: "all symbols",
			exp: &CrossMarginSymbolsResponse{
				Timestamp: types.Time(time.UnixMilli(1791898085186)),
				Items: []CrossMarginSymbol{
					{
						Symbol:         "AAVE-USDT",
						Name:           "AAVE-USDT",
						EnableTrading:  true,
						Market:         "DeFi",
						BaseCurrency:   currency.AAVE,
						QuoteCurrency:  currency.USDT,
						BaseIncrement:  0.0001,
						BaseMinSize:    0.001,
						QuoteIncrement: 0.001,
						QuoteMinSize:   0.1,
						BaseMaxSize:    10000000000,
						QuoteMaxSize:   99999999,
						PriceIncrement: 0.001,
						FeeCurrency:    currency.USDT,
						PriceLimitRate: 0.03,
						MinFunds:       0.1,
					},
					{
						Symbol:         "ETH-BTC",
						Name:           "ETH-BTC",
						EnableTrading:  true,
						Market:         "BTC",
						BaseCurrency:   currency.ETH,
						QuoteCurrency:  currency.BTC,
						BaseIncrement:  0.0000001,
						BaseMinSize:    0.0001,
						QuoteIncrement: 0.00001,
						QuoteMinSize:   0.00001,
						BaseMaxSize:    10000000000,
						QuoteMaxSize:   999999999,
						PriceIncrement: 0.00001,
						FeeCurrency:    currency.BTC,
						PriceLimitRate: 0.03,
						MinFunds:       0.000001,
					},
				},
			},
		},
		{
			name:   "one symbol",
			symbol: "BTC-USDT",
			exp: &CrossMarginSymbolsResponse{
				Timestamp: types.Time(time.UnixMilli(1791898079912)),
				Items: []CrossMarginSymbol{
					{
						Symbol:         "BTC-USDT",
						Name:           "BTC-USDT",
						EnableTrading:  true,
						Market:         "USDS",
						BaseCurrency:   currency.BTC,
						QuoteCurrency:  currency.USDT,
						BaseIncrement:  0.00000001,
						BaseMinSize:    0.00001,
						QuoteIncrement: 0.000001,
						QuoteMinSize:   0.1,
						BaseMaxSize:    10000000000,
						QuoteMaxSize:   99999999,
						PriceIncrement: 0.1,
						FeeCurrency:    currency.USDT,
						PriceLimitRate: 0.01,
						MinFunds:       0.1,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetCrossMarginSymbols(t.Context(), tc.symbol)
			require.NoError(t, err, "GetCrossMarginSymbols must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetCrossMarginSymbols should decode every field")
				return
			}
			require.NotEmpty(t, result.Items, "GetCrossMarginSymbols must return symbols")
			if tc.symbol != "" {
				assert.Equal(t, tc.symbol, result.Items[0].Symbol, "GetCrossMarginSymbols should return the symbol requested")
			}
		})
	}
}

func TestGetIsolatedMarginSymbols(t *testing.T) {
	t.Parallel()
	result, err := e.GetIsolatedMarginSymbols(t.Context())
	require.NoError(t, err, "GetIsolatedMarginSymbols must not error")
	if mockTests {
		exp := []IsolatedMarginSymbolResponse{
			{
				Symbol:                     "KCS-USDT",
				SymbolName:                 "KCS-USDT",
				BaseCurrency:               currency.KCS,
				QuoteCurrency:              currency.USDT,
				MaxLeverage:                10,
				ForcedLiquidationDebtRatio: 0.95,
				TradeEnable:                true,
				AutoRenewMaxDebtRatio:      0.94,
				QuoteBorrowEnable:          true,
				BaseTransferInEnable:       true,
				QuoteTransferInEnable:      true,
				BaseBorrowCoefficient:      1,
				QuoteBorrowCoefficient:     1,
			},
			{
				Symbol:                     "BTC-USDT",
				SymbolName:                 "BTC-USDT",
				BaseCurrency:               currency.BTC,
				QuoteCurrency:              currency.USDT,
				MaxLeverage:                10,
				ForcedLiquidationDebtRatio: 0.95,
				TradeEnable:                true,
				AutoRenewMaxDebtRatio:      0.94,
				BaseBorrowEnable:           true,
				QuoteBorrowEnable:          true,
				BaseTransferInEnable:       true,
				QuoteTransferInEnable:      true,
				BaseBorrowCoefficient:      1,
				QuoteBorrowCoefficient:     1,
			},
		}
		assert.Equal(t, exp, result, "GetIsolatedMarginSymbols should decode every field")
		return
	}
	require.NotEmpty(t, result, "GetIsolatedMarginSymbols must return symbols")
	assert.Positive(t, result[0].MaxLeverage, "GetIsolatedMarginSymbols should return a maximum leverage")
}

func TestGetMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPrice(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetMarkPrice must reject an empty symbol")

	result, err := e.GetMarkPrice(t.Context(), "USDT-BTC")
	require.NoError(t, err, "GetMarkPrice must not error")
	if mockTests {
		exp := &MarginMarkPriceResponse{
			Symbol:      "USDT-BTC",
			Granularity: 1000,
			TimePoint:   types.Time(time.UnixMilli(1791898112000)),
			Value:       0.000011986231177,
		}
		assert.Equal(t, exp, result, "GetMarkPrice should decode every field")
		return
	}
	assert.Equal(t, "USDT-BTC", result.Symbol, "GetMarkPrice should return the symbol requested")
	assert.Positive(t, result.Value, "GetMarkPrice should return a mark price")
}

func TestGetMarginConfiguration(t *testing.T) {
	t.Parallel()
	result, err := e.GetMarginConfiguration(t.Context())
	require.NoError(t, err, "GetMarginConfiguration must not error")
	if mockTests {
		exp := &MarginConfigurationResponse{
			CurrencyList:         []currency.Code{currency.NewCode("TAO"), currency.PAXG, currency.APT},
			MaxLeverage:          5,
			WarningDebtRatio:     0.92,
			LiquidationDebtRatio: 0.95,
		}
		assert.Equal(t, exp, result, "GetMarginConfiguration should decode every field")
		return
	}
	assert.NotEmpty(t, result.CurrencyList, "GetMarginConfiguration should return the margin currencies")
	assert.Positive(t, result.MaxLeverage, "GetMarginConfiguration should return a maximum leverage")
}

func TestGetAllMarginTradingPairsMarkPrices(t *testing.T) {
	t.Parallel()
	result, err := e.GetAllMarginTradingPairsMarkPrices(t.Context())
	require.NoError(t, err, "GetAllMarginTradingPairsMarkPrices must not error")
	if mockTests {
		exp := []MarginMarkPriceResponse{
			{
				Symbol:      "ETH-BTC",
				Granularity: 1000,
				TimePoint:   types.Time(time.UnixMilli(1791898100000)),
				Value:       0.030095724145701,
			},
			{
				Symbol:      "USDT-BTC",
				Granularity: 1000,
				TimePoint:   types.Time(time.UnixMilli(1791898100000)),
				Value:       0.000011985744156,
			},
		}
		assert.Equal(t, exp, result, "GetAllMarginTradingPairsMarkPrices should decode every field")
		return
	}
	require.NotEmpty(t, result, "GetAllMarginTradingPairsMarkPrices must return mark prices")
	assert.Positive(t, result[0].Value, "GetAllMarginTradingPairsMarkPrices should return a mark price")
}

func TestGetMarginCollateralRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginCollateralRatio(t.Context(), []currency.Code{currency.BTC, currency.EMPTYCODE})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMarginCollateralRatio must reject an empty currency")

	for _, tc := range []struct {
		name       string
		currencies []currency.Code
		exp        []MarginCollateralRatioResponse
	}{
		{
			name: "all currencies",
			exp: []MarginCollateralRatioResponse{
				{
					CurrencyList: []currency.Code{currency.TIA, currency.KCS, currency.NewCode("NXPC")},
					Items: []MarginCollateralRatioItem{
						{UpperLimit: 50000, CollateralRatio: 0.95},
						{LowerLimit: 50000, UpperLimit: 100000, CollateralRatio: 0.8},
						{LowerLimit: 500000, UpperLimit: 2000000000},
					},
				},
				{
					CurrencyList: []currency.Code{currency.PEPE, currency.ARB, currency.HBAR},
					Items: []MarginCollateralRatioItem{
						{UpperLimit: 200000, CollateralRatio: 0.97},
						{LowerLimit: 200000, UpperLimit: 500000, CollateralRatio: 0.9},
						{LowerLimit: 3000000, UpperLimit: 2000000000},
					},
				},
			},
		},
		{
			name:       "some currencies",
			currencies: []currency.Code{currency.BTC, currency.USDT},
			exp: []MarginCollateralRatioResponse{
				{
					CurrencyList: []currency.Code{currency.BTC},
					Items: []MarginCollateralRatioItem{
						{UpperLimit: 5000000, CollateralRatio: 1},
						{LowerLimit: 5000000, UpperLimit: 20000000, CollateralRatio: 0.95},
						{LowerLimit: 1000000000, UpperLimit: 2000000000},
					},
				},
				{
					CurrencyList: []currency.Code{currency.USDT},
					Items: []MarginCollateralRatioItem{
						{UpperLimit: 100000000, CollateralRatio: 1},
						{LowerLimit: 100000000, UpperLimit: 2000000000},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetMarginCollateralRatio(t.Context(), tc.currencies)
			require.NoError(t, err, "GetMarginCollateralRatio must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetMarginCollateralRatio should decode every field")
				return
			}
			require.NotEmpty(t, result, "GetMarginCollateralRatio must return collateral ratios")
			assert.NotEmpty(t, result[0].Items, "GetMarginCollateralRatio should return collateral ratio tiers")
		})
	}
}

func TestGetMarketAvailableInventory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		ccy  currency.Code
		exp  []MarketAvailableInventoryResponse
	}{
		{
			name: "all currencies",
			exp: []MarketAvailableInventoryResponse{
				{Currency: currency.USDT, BorrowableAmount: 9066139.89392282214},
				{Currency: currency.USDC, BorrowableAmount: 269973.07992858573},
			},
		},
		{
			name: "one currency",
			ccy:  currency.BTC,
			exp:  []MarketAvailableInventoryResponse{{Currency: currency.BTC, BorrowableAmount: 10.47245668039}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetMarketAvailableInventory(t.Context(), tc.ccy)
			require.NoError(t, err, "GetMarketAvailableInventory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetMarketAvailableInventory should decode every field")
				return
			}
			require.NotEmpty(t, result, "GetMarketAvailableInventory must return inventory")
			if !tc.ccy.IsEmpty() {
				assert.Equal(t, tc.ccy, result[0].Currency, "GetMarketAvailableInventory should return the currency requested")
			}
		})
	}
}

func TestCurrencyListParam(t *testing.T) {
	t.Parallel()
	list, err := currencyListParam([]currency.Code{currency.BTC, currency.NewCode("eth")})
	require.NoError(t, err, "currencyListParam must not error")
	assert.Equal(t, "BTC,ETH", list, "currencyListParam should join the currencies in upper case")

	_, err = currencyListParam([]currency.Code{currency.BTC, currency.EMPTYCODE})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "currencyListParam should reject an empty currency")
}
