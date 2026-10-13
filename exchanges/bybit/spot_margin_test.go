package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetVIPMarginData(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		vipLevel string
		coin     currency.Code
		// mockOnly marks the unfiltered query, about 1.6 MB, which Bybit intermittently refuses with retCode 10006
		mockOnly bool
		exp      *VIPMarginDataResponse
	}{
		{name: "every VIP level", mockOnly: true, exp: &VIPMarginDataResponse{
			VIPCoinList: []VIPMarginLevel{
				{
					List: []VIPMarginCoin{
						{
							Borrowable:         true,
							CollateralRatio:    1,
							Currency:           currency.USDT,
							HourlyBorrowRate:   0.000004455751,
							LiquidationOrder:   1,
							MarginCollateral:   true,
							MaxBorrowingAmount: 8000000,
						},
						{
							CollateralRatio:  0.999,
							Currency:         currency.NewCode("BYUSDT"),
							LiquidationOrder: 2,
							MarginCollateral: true,
						},
					},
					VIPLevel: "No VIP",
				},
				{
					List: []VIPMarginCoin{
						{
							Borrowable:         true,
							CollateralRatio:    1,
							Currency:           currency.USDT,
							HourlyBorrowRate:   0.0000043608435037,
							LiquidationOrder:   1,
							MarginCollateral:   true,
							MaxBorrowingAmount: 11000000,
						},
						{
							CollateralRatio:  0.999,
							Currency:         currency.NewCode("BYUSDT"),
							LiquidationOrder: 2,
							MarginCollateral: true,
						},
					},
					VIPLevel: "VIP-1",
				},
			},
		}},
		{name: "one VIP level and coin", vipLevel: "No VIP", coin: currency.BTC, exp: &VIPMarginDataResponse{
			VIPCoinList: []VIPMarginLevel{
				{
					List: []VIPMarginCoin{
						{
							Borrowable:         true,
							CollateralRatio:    0.98,
							Currency:           currency.BTC,
							HourlyBorrowRate:   0.000000486539,
							LiquidationOrder:   3,
							MarginCollateral:   true,
							MaxBorrowingAmount: 300,
						},
					},
					VIPLevel: "No VIP",
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.mockOnly && !mockTests {
				t.Skip("Bybit intermittently refuses the unfiltered query with retCode 10006")
			}
			result, err := e.GetVIPMarginData(t.Context(), tc.vipLevel, tc.coin)
			require.NoError(t, err, "GetVIPMarginData must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetVIPMarginData should decode every field")
				return
			}
			require.NotEmpty(t, result.VIPCoinList, "GetVIPMarginData must return VIP levels")
			assert.NotEmpty(t, result.VIPCoinList[0].List, "GetVIPMarginData should return coins")
		})
	}
}

func TestGetTieredCollateralRatio(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		coin currency.Code
		exp  *TieredCollateralRatioResponse
	}{
		{name: "every coin", exp: &TieredCollateralRatioResponse{
			List: []TieredCollateralRatio{
				{
					Currency: currency.NewCode("AGLD"),
					CollateralRatioList: []CollateralRatioTier{
						{
							MaxQuantity:     9430,
							CollateralRatio: 0.7,
						},
						{
							MaxQuantity:     11300,
							MinQuantity:     9430,
							CollateralRatio: 0.6,
						},
						{
							MinQuantity: 99999999999,
						},
					},
				},
				{
					Currency: currency.NewCode("SCR"),
					CollateralRatioList: []CollateralRatioTier{
						{
							MaxQuantity:     380000,
							CollateralRatio: 0.7,
						},
						{
							MaxQuantity:     770000,
							MinQuantity:     380000,
							CollateralRatio: 0.6,
						},
						{
							MinQuantity: 99999999999,
						},
					},
				},
			},
		}},
		{name: "one coin", coin: currency.BTC, exp: &TieredCollateralRatioResponse{
			List: []TieredCollateralRatio{
				{
					Currency: currency.BTC,
					CollateralRatioList: []CollateralRatioTier{
						{
							CollateralRatio: 0.98,
						},
					},
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetTieredCollateralRatio(t.Context(), tc.coin)
			require.NoError(t, err, "GetTieredCollateralRatio must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetTieredCollateralRatio should decode every field")
				return
			}
			require.NotEmpty(t, result.List, "GetTieredCollateralRatio must return coins")
			assert.NotEmpty(t, result.List[0].CollateralRatioList, "GetTieredCollateralRatio should return tiers")
		})
	}
}

func TestGetSpotMarginCurrencyData(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		coin currency.Code
		exp  *SpotMarginCurrencyDataResponse
	}{
		{name: "every coin", exp: &SpotMarginCurrencyDataResponse{
			List: []SpotMarginCurrencyData{
				{
					Currency:                        currency.LTC,
					FlexibleManualBorrowable:        true,
					MinFlexibleManualBorrowQuantity: 0.4,
					FlexibleManualBorrowAccuracy:    4,
				},
				{
					Currency:                        currency.USDT,
					FlexibleManualBorrowable:        true,
					MinFlexibleManualBorrowQuantity: 20,
					FlexibleManualBorrowAccuracy:    4,
					FixedManualBorrowable:           true,
					MinFixedManualBorrowQuantity:    2000,
					FixedManualBorrowAccuracy:       4,
					FixedInterestRateAccuracy:       4,
					MinFixedInterestRate:            0.01,
					MaxFixedInterestRate:            0.8,
				},
			},
		}},
		{name: "one coin", coin: currency.BTC, exp: &SpotMarginCurrencyDataResponse{
			List: []SpotMarginCurrencyData{
				{
					Currency:                        currency.BTC,
					FlexibleManualBorrowable:        true,
					MinFlexibleManualBorrowQuantity: 0.0011,
					FlexibleManualBorrowAccuracy:    4,
					FixedManualBorrowable:           true,
					MinFixedManualBorrowQuantity:    0.2,
					FixedManualBorrowAccuracy:       8,
					FixedInterestRateAccuracy:       4,
					MinFixedInterestRate:            0.003,
					MaxFixedInterestRate:            0.4,
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSpotMarginCurrencyData(t.Context(), tc.coin)
			require.NoError(t, err, "GetSpotMarginCurrencyData must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSpotMarginCurrencyData should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetSpotMarginCurrencyData should return coins")
		})
	}
}

func TestGetSpotMarginHistoricalInterestRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotMarginHistoricalInterestRate(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotMarginHistoricalInterestRate must reject a nil request")
	_, err = e.GetSpotMarginHistoricalInterestRate(t.Context(), &SpotMarginHistoricalInterestRateRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSpotMarginHistoricalInterestRate must reject a request without a currency")
	_, err = e.GetSpotMarginHistoricalInterestRate(t.Context(), &SpotMarginHistoricalInterestRateRequest{Currency: currency.BTC, StartTime: time.UnixMilli(1788220800000)})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetSpotMarginHistoricalInterestRate must reject a start time without an end time")
	_, err = e.GetSpotMarginHistoricalInterestRate(t.Context(), &SpotMarginHistoricalInterestRateRequest{Currency: currency.BTC, StartTime: time.UnixMilli(1788231600000), EndTime: time.UnixMilli(1788220800000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotMarginHistoricalInterestRate must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *SpotMarginHistoricalInterestRateRequest
		exp  *SpotMarginHistoricalInterestRateResponse
	}{
		{
			name: "VIP level in a window",
			req:  &SpotMarginHistoricalInterestRateRequest{Currency: currency.USDC, VIPLevel: "No VIP", StartTime: time.UnixMilli(1788220800000), EndTime: time.UnixMilli(1788231600000)},
			exp: &SpotMarginHistoricalInterestRateResponse{
				List: []SpotMarginInterestRate{
					{
						Timestamp:        types.Time(time.Unix(1788231600, 0)),
						Currency:         currency.USDC,
						HourlyBorrowRate: 0.000014621596,
						VIPLevel:         "No VIP",
					},
					{
						Timestamp:        types.Time(time.Unix(1788228000, 0)),
						Currency:         currency.USDC,
						HourlyBorrowRate: 0.000014598301,
						VIPLevel:         "No VIP",
					},
				},
			},
		},
		{
			name: "account's VIP level",
			req:  &SpotMarginHistoricalInterestRateRequest{Currency: currency.BTC},
			exp: &SpotMarginHistoricalInterestRateResponse{
				List: []SpotMarginInterestRate{
					{
						Timestamp:        types.Time(time.Unix(1791532800, 0)),
						Currency:         currency.BTC,
						HourlyBorrowRate: 0.000000486539,
						VIPLevel:         "VIP-1",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSpotMarginHistoricalInterestRate(t.Context(), tc.req)
			require.NoError(t, err, "GetSpotMarginHistoricalInterestRate must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSpotMarginHistoricalInterestRate should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetSpotMarginHistoricalInterestRate should return rates")
		})
	}
}

func TestToggleMarginTrade(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		on  bool
		exp *SpotMarginModeResponse
	}{
		{on: true, exp: &SpotMarginModeResponse{
			SpotMarginMode: true,
		}},
		{on: false, exp: &SpotMarginModeResponse{}},
	} {
		result, err := e.ToggleMarginTrade(t.Context(), tc.on)
		require.NoErrorf(t, err, "ToggleMarginTrade must not error turning it %t", tc.on)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "ToggleMarginTrade should decode every field turning it %t", tc.on)
			continue
		}
		assert.Equalf(t, tc.on, result.SpotMarginMode.Bool(), "ToggleMarginTrade should turn it %t", tc.on)
	}
}

func TestSetSpotMarginTradeLeverage(t *testing.T) {
	t.Parallel()
	err := e.SetSpotMarginTradeLeverage(t.Context(), 0, currency.EMPTYCODE)
	require.ErrorIs(t, err, errInvalidLeverage, "SetSpotMarginTradeLeverage must reject a request without a leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		leverage float64
		coin     currency.Code
	}{
		{leverage: 4},
		{leverage: 3, coin: currency.BTC},
	} {
		assert.NoErrorf(t, e.SetSpotMarginTradeLeverage(t.Context(), tc.leverage, tc.coin), "SetSpotMarginTradeLeverage should not error for coin %q", tc.coin)
	}
}

func TestGetSpotMarginStatusAndLeverage(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotMarginStatusAndLeverage(t.Context())
	require.NoError(t, err, "GetSpotMarginStatusAndLeverage must not error")
	if mockTests {
		exp := &SpotMarginStatusResponse{
			SpotLeverage:      10,
			SpotMarginMode:    true,
			EffectiveLeverage: 1.25,
		}
		assert.Equal(t, exp, result, "GetSpotMarginStatusAndLeverage should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotMarginStatusAndLeverage should return the status")
}

func TestGetMaxBorrowableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaxBorrowableAmount(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMaxBorrowableAmount must reject a request without a currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMaxBorrowableAmount(t.Context(), currency.BTC)
	require.NoError(t, err, "GetMaxBorrowableAmount must not error")
	if mockTests {
		exp := &MaxBorrowableAmountResponse{
			Currency: currency.BTC,
			MaxLoan:  17.54689892,
		}
		assert.Equal(t, exp, result, "GetMaxBorrowableAmount should decode every field")
		return
	}
	assert.Equal(t, currency.BTC, result.Currency, "GetMaxBorrowableAmount should return the requested currency")
}

func TestGetSpotMarginPositionTiers(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		coin currency.Code
		exp  *SpotMarginPositionTiersResponse
	}{
		{name: "one coin", coin: currency.BTC, exp: &SpotMarginPositionTiersResponse{
			List: []SpotMarginPositionTiers{
				{
					Currency: currency.BTC,
					PositionTiersRatioList: []SpotMarginPositionTier{
						{
							Tier:                          1,
							BorrowLimit:                   390,
							PositionMaintenanceMarginRate: 0.04,
							PositionInitialMarginRate:     0.2,
							MaxLeverage:                   5,
						},
						{
							Tier:                          2,
							BorrowLimit:                   391,
							PositionMaintenanceMarginRate: 0.04,
							PositionInitialMarginRate:     0.25,
							MaxLeverage:                   4,
						},
					},
				},
			},
		}},
		{name: "every coin", exp: &SpotMarginPositionTiersResponse{
			List: []SpotMarginPositionTiers{
				{
					Currency: currency.BTC,
					PositionTiersRatioList: []SpotMarginPositionTier{
						{
							Tier:                          1,
							BorrowLimit:                   390,
							PositionMaintenanceMarginRate: 0.04,
							PositionInitialMarginRate:     0.2,
							MaxLeverage:                   5,
						},
						{
							Tier:                          2,
							BorrowLimit:                   391,
							PositionMaintenanceMarginRate: 0.04,
							PositionInitialMarginRate:     0.25,
							MaxLeverage:                   4,
						},
					},
				},
				{
					Currency: currency.ETH,
					PositionTiersRatioList: []SpotMarginPositionTier{
						{
							Tier:                          1,
							BorrowLimit:                   5000,
							PositionMaintenanceMarginRate: 0.05,
							PositionInitialMarginRate:     0.33333333,
							MaxLeverage:                   3,
						},
					},
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSpotMarginPositionTiers(t.Context(), tc.coin)
			require.NoError(t, err, "GetSpotMarginPositionTiers must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSpotMarginPositionTiers should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetSpotMarginPositionTiers should return coins")
		})
	}
}

func TestGetSpotMarginCoinState(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		coin currency.Code
		exp  *SpotMarginCoinStateResponse
	}{
		{name: "every coin", exp: &SpotMarginCoinStateResponse{
			List: []SpotMarginCoinState{
				{
					Currency:     currency.BTC,
					SpotLeverage: 3,
				},
				{
					Currency:     currency.ETH,
					SpotLeverage: 4,
				},
			},
		}},
		{name: "one coin", coin: currency.BTC, exp: &SpotMarginCoinStateResponse{
			List: []SpotMarginCoinState{
				{
					Currency:     currency.BTC,
					SpotLeverage: 3,
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSpotMarginCoinState(t.Context(), tc.coin)
			require.NoError(t, err, "GetSpotMarginCoinState must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSpotMarginCoinState should decode every field")
				return
			}
			assert.NotNil(t, result, "GetSpotMarginCoinState should return the coins")
		})
	}
}

func TestGetSpotMarginAvailableAmountToRepay(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotMarginAvailableAmountToRepay(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSpotMarginAvailableAmountToRepay must reject a request without a currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotMarginAvailableAmountToRepay(t.Context(), currency.BTC)
	require.NoError(t, err, "GetSpotMarginAvailableAmountToRepay must not error")
	if mockTests {
		exp := &SpotMarginAvailableAmountToRepayResponse{
			Currency:                currency.BTC,
			LossLessRepaymentAmount: 0.02,
		}
		assert.Equal(t, exp, result, "GetSpotMarginAvailableAmountToRepay should decode every field")
		return
	}
	assert.Equal(t, currency.BTC, result.Currency, "GetSpotMarginAvailableAmountToRepay should return the requested currency")
}

func TestSetSpotMarginAutoRepayMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		coin currency.Code
		on   bool
		exp  *SpotMarginAutoRepayModeResponse
	}{
		{name: "turn one coin on", coin: currency.ETH, on: true, exp: &SpotMarginAutoRepayModeResponse{
			Data: []SpotMarginAutoRepayMode{
				{
					Currency:      currency.ETH,
					AutoRepayMode: true,
				},
			},
		}},
		{name: "turn every coin off", exp: &SpotMarginAutoRepayModeResponse{
			Data: []SpotMarginAutoRepayMode{
				{
					Currency: currency.BTC,
				},
				{
					Currency: currency.ETH,
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.SetSpotMarginAutoRepayMode(t.Context(), tc.coin, tc.on)
			require.NoError(t, err, "SetSpotMarginAutoRepayMode must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "SetSpotMarginAutoRepayMode should decode every field")
				return
			}
			assert.NotEmpty(t, result.Data, "SetSpotMarginAutoRepayMode should return the coins it set")
		})
	}
}

func TestGetSpotMarginAutoRepayMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		coin currency.Code
		exp  *SpotMarginAutoRepayModeResponse
	}{
		{name: "one coin", coin: currency.ETH, exp: &SpotMarginAutoRepayModeResponse{
			Data: []SpotMarginAutoRepayMode{
				{
					Currency:      currency.ETH,
					AutoRepayMode: true,
				},
			},
		}},
		{name: "every coin", exp: &SpotMarginAutoRepayModeResponse{
			Data: []SpotMarginAutoRepayMode{
				{
					Currency: currency.BTC,
				},
				{
					Currency:      currency.ETH,
					AutoRepayMode: true,
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSpotMarginAutoRepayMode(t.Context(), tc.coin)
			require.NoError(t, err, "GetSpotMarginAutoRepayMode must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSpotMarginAutoRepayMode should decode every field")
				return
			}
			assert.NotEmpty(t, result.Data, "GetSpotMarginAutoRepayMode should return coins")
		})
	}
}

func TestGetSpotMarginFixedRateBorrowOrderQuote(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotMarginFixedRateBorrowOrderQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotMarginFixedRateBorrowOrderQuote must reject a nil request")
	_, err = e.GetSpotMarginFixedRateBorrowOrderQuote(t.Context(), &SpotMarginFixedRateBorrowOrderQuoteRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSpotMarginFixedRateBorrowOrderQuote must reject a request without a currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *SpotMarginFixedRateBorrowOrderQuoteRequest
		exp  *SpotMarginFixedRateBorrowOrderQuoteResponse
	}{
		{
			name: "by annual rate",
			req:  &SpotMarginFixedRateBorrowOrderQuoteRequest{OrderCurrency: currency.ETH, OrderBy: "apy", Limit: 10},
			exp: &SpotMarginFixedRateBorrowOrderQuoteResponse{
				List: []SpotMarginFixedRateBorrowOrderQuote{
					{
						OrderCurrency: currency.ETH,
						Term:          30,
						AnnualRate:    0.026,
						Quantity:      0.1,
					},
					{
						OrderCurrency: currency.ETH,
						Term:          90,
						AnnualRate:    0.038,
						Quantity:      0.1,
					},
				},
			},
		},
		{
			name: "one term by quantity, descending",
			req:  &SpotMarginFixedRateBorrowOrderQuoteRequest{OrderCurrency: currency.ETH, Term: 30, OrderBy: "quantity", Sort: 1, Limit: 2},
			exp: &SpotMarginFixedRateBorrowOrderQuoteResponse{
				List: []SpotMarginFixedRateBorrowOrderQuote{
					{
						OrderCurrency: currency.ETH,
						Term:          30,
						AnnualRate:    0.1,
						Quantity:      0.6,
					},
					{
						OrderCurrency: currency.ETH,
						Term:          30,
						AnnualRate:    0.026,
						Quantity:      0.1,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSpotMarginFixedRateBorrowOrderQuote(t.Context(), tc.req)
			require.NoError(t, err, "GetSpotMarginFixedRateBorrowOrderQuote must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSpotMarginFixedRateBorrowOrderQuote should decode every field")
				return
			}
			assert.NotNil(t, result, "GetSpotMarginFixedRateBorrowOrderQuote should return the offers")
		})
	}
}

func TestSpotMarginFixedRateBorrow(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *SpotMarginFixedRateBorrowRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no currency", req: &SpotMarginFixedRateBorrowRequest{}, err: currency.ErrCurrencyCodeEmpty},
		{name: "no amount", req: &SpotMarginFixedRateBorrowRequest{OrderCurrency: currency.BTC}, err: limits.ErrAmountBelowMin},
		{name: "no annual rate", req: &SpotMarginFixedRateBorrowRequest{OrderCurrency: currency.BTC, OrderAmount: 0.01}, err: errAnnualRateNotSet},
		{name: "no term", req: &SpotMarginFixedRateBorrowRequest{OrderCurrency: currency.BTC, OrderAmount: 0.01, AnnualRate: 0.02}, err: errTermNotSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.SpotMarginFixedRateBorrow(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "SpotMarginFixedRateBorrow should return the expected error")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *SpotMarginFixedRateBorrowRequest
		exp  *SpotMarginFixedRateBorrowResponse
	}{
		{
			name: "default repayment and fill",
			req:  &SpotMarginFixedRateBorrowRequest{OrderCurrency: currency.BTC, OrderAmount: 0.01, AnnualRate: 0.02, Term: 30},
			exp: &SpotMarginFixedRateBorrowResponse{
				OrderID: "FIXED_BORROW_4563567182f746ec9f73e4357264d8c7187",
			},
		},
		{
			name: "flexible loan when due, full fill only",
			req:  &SpotMarginFixedRateBorrowRequest{OrderCurrency: currency.USDT, OrderAmount: 1000, AnnualRate: 0.05, Term: 7, RepayType: "2", StrategyType: "FULL"},
			exp: &SpotMarginFixedRateBorrowResponse{
				OrderID: "FIXED_BORROW_a17089fc526441faa52eb99b0b9feb69185",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.SpotMarginFixedRateBorrow(t.Context(), tc.req)
			require.NoError(t, err, "SpotMarginFixedRateBorrow must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "SpotMarginFixedRateBorrow should decode every field")
				return
			}
			assert.NotEmpty(t, result.OrderID, "SpotMarginFixedRateBorrow should return the loan order ID")
		})
	}
}

func TestRenewSpotMarginFixedRateBorrow(t *testing.T) {
	t.Parallel()
	err := e.RenewSpotMarginFixedRateBorrow(t.Context(), "", 0)
	require.ErrorIs(t, err, errLoanIDNotSet, "RenewSpotMarginFixedRateBorrow must reject a request without a loan ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	if mockTests {
		err = e.RenewSpotMarginFixedRateBorrow(t.Context(), "2091991237922142464", 500)
		require.ErrorIs(t, err, errFixedRateBorrowRenewalFailed, "RenewSpotMarginFixedRateBorrow must return a failed renewal as an error")
		err = e.RenewSpotMarginFixedRateBorrow(t.Context(), "2000000000000000000", 0)
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "RenewSpotMarginFixedRateBorrow must return an API error")
	}
	assert.NoError(t, e.RenewSpotMarginFixedRateBorrow(t.Context(), "2092341042506646784", 0), "RenewSpotMarginFixedRateBorrow should not error")
}

func TestGetSpotMarginFixedRateBorrowOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotMarginFixedRateBorrowOrderInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotMarginFixedRateBorrowOrderInfo must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *SpotMarginFixedRateBorrowOrderInfoRequest
		// mockOnly marks requests with a placeholder cursor that Bybit could reject live
		mockOnly bool
		exp      *SpotMarginFixedRateBorrowOrderInfoResponse
	}{
		{
			name: "coin",
			req:  &SpotMarginFixedRateBorrowOrderInfoRequest{OrderCurrency: currency.ETH, Limit: 10},
			exp: &SpotMarginFixedRateBorrowOrderInfoResponse{
				List: []SpotMarginFixedRateBorrowOrder{
					{
						AnnualRate:     0.07,
						OrderID:        "FIXED_BORROW_4563567182f746ec9f73e4357264d8c7187",
						OrderTime:      types.Time(time.Unix(1775616125, 0)),
						FilledQuantity: 0.4,
						OrderQuantity:  1,
						OrderCurrency:  currency.ETH,
						State:          2,
						Term:           7,
						RepayType:      "1",
						StrategyType:   "PARTIAL",
					},
					{
						AnnualRate:    0.06,
						OrderID:       "FIXED_BORROW_9b0c2b8f6a4d4e1b8f3c2a1d0e9f8a7b112",
						OrderTime:     types.Time(time.Unix(1764120783, 0)),
						OrderQuantity: 2,
						OrderCurrency: currency.ETH,
						State:         1,
						Term:          30,
						RepayType:     "2",
						StrategyType:  "FULL",
					},
				},
				NextPageCursor: "30",
			},
		},
		{
			name:     "filled order of a term",
			req:      &SpotMarginFixedRateBorrowOrderInfoRequest{OrderID: "FIXED_BORROW_a17089fc526441faa52eb99b0b9feb69185", OrderCurrency: currency.USDT, State: 3, Term: 7, Limit: 1, Cursor: "30"},
			mockOnly: true,
			exp: &SpotMarginFixedRateBorrowOrderInfoResponse{
				List: []SpotMarginFixedRateBorrowOrder{
					{
						AnnualRate:     0.05,
						OrderID:        "FIXED_BORROW_a17089fc526441faa52eb99b0b9feb69185",
						OrderTime:      types.Time(time.Unix(1775616124, 0)),
						FilledQuantity: 1000,
						OrderQuantity:  1000,
						OrderCurrency:  currency.USDT,
						State:          3,
						Term:           7,
						RepayType:      "2",
						StrategyType:   "FULL",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.mockOnly && !mockTests {
				t.Skip("request carries a placeholder cursor")
			}
			result, err := e.GetSpotMarginFixedRateBorrowOrderInfo(t.Context(), tc.req)
			require.NoError(t, err, "GetSpotMarginFixedRateBorrowOrderInfo must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSpotMarginFixedRateBorrowOrderInfo should decode every field")
				return
			}
			assert.NotNil(t, result, "GetSpotMarginFixedRateBorrowOrderInfo should return a page")
		})
	}
}

func TestGetSpotMarginFixedRateBorrowContractInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotMarginFixedRateBorrowContractInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotMarginFixedRateBorrowContractInfo must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *SpotMarginFixedRateBorrowContractInfoRequest
		// mockOnly marks requests with a placeholder cursor that Bybit could reject live
		mockOnly bool
		exp      *SpotMarginFixedRateBorrowContractInfoResponse
	}{
		{
			name: "coin",
			req:  &SpotMarginFixedRateBorrowContractInfoRequest{OrderCurrency: currency.USDT, Limit: 10},
			exp: &SpotMarginFixedRateBorrowContractInfoResponse{
				List: []SpotMarginFixedRateBorrowContract{
					{
						AnnualRate:              1,
						BorrowCurrency:          currency.USDT,
						BorrowTime:              types.Time(time.Unix(1764162490, 0)),
						InterestPaid:            1.0657534246575342,
						LoanID:                  "2092341042506646784",
						OrderID:                 "FIXED_BORROW_a17089fc526441faa52eb99b0b9feb69185",
						RepaymentTime:           types.Time(time.Unix(1764288000, 0)),
						ResidualPenaltyInterest: 0.012,
						ResidualPrincipal:       400,
						Status:                  3,
						Term:                    1,
						RepayType:               "1",
						StrategyType:            "PARTIAL",
					},
					{
						AnnualRate:     1,
						BorrowCurrency: currency.USDT,
						BorrowTime:     types.Time(time.Unix(1764149170, 0)),
						InterestPaid:   0.030136986301369864,
						LoanID:         "2092229306860452864",
						OrderID:        "FIXED_BORROW_a17089fc526441faa52eb99b0b9feb69185",
						RepaymentTime:  types.Time(time.Unix(1764244800, 0)),
						Status:         2,
						Term:           1,
						RepayType:      "0",
						StrategyType:   "PARTIAL",
					},
				},
				NextPageCursor: "0",
			},
		},
		{
			name:     "loans of an order",
			req:      &SpotMarginFixedRateBorrowContractInfoRequest{OrderID: "FIXED_BORROW_4563567182f746ec9f73e4357264d8c7187", OrderCurrency: currency.ETH, Term: 7, Limit: 1, Cursor: "0"},
			mockOnly: true,
			exp: &SpotMarginFixedRateBorrowContractInfoResponse{
				List: []SpotMarginFixedRateBorrowContract{
					{
						AnnualRate:        0.07,
						BorrowCurrency:    currency.ETH,
						BorrowTime:        types.Time(time.Unix(1775616126, 0)),
						LoanID:            "2101543977219727360",
						OrderID:           "FIXED_BORROW_4563567182f746ec9f73e4357264d8c7187",
						RepaymentTime:     types.Time(time.Unix(1776220926, 0)),
						ResidualPrincipal: 0.4,
						Status:            1,
						Term:              7,
						RepayType:         "1",
						StrategyType:      "PARTIAL",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.mockOnly && !mockTests {
				t.Skip("request carries a placeholder cursor")
			}
			result, err := e.GetSpotMarginFixedRateBorrowContractInfo(t.Context(), tc.req)
			require.NoError(t, err, "GetSpotMarginFixedRateBorrowContractInfo must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSpotMarginFixedRateBorrowContractInfo should decode every field")
				return
			}
			assert.NotNil(t, result, "GetSpotMarginFixedRateBorrowContractInfo should return a page")
		})
	}
}

func TestGetSpotMarginLiability(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotMarginLiability(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSpotMarginLiability must reject a request without a currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotMarginLiability(t.Context(), currency.BTC)
	require.NoError(t, err, "GetSpotMarginLiability must not error")
	if mockTests {
		exp := &SpotMarginLiabilityResponse{
			Currency:             currency.BTC,
			TotalBorrowAmount:    0.05,
			FixedBorrowAmount:    0.02,
			FlexibleBorrowAmount: 0.03,
			SpotTotalBorrow:      0.04,
			DerivativesBorrow:    0.01,
		}
		assert.Equal(t, exp, result, "GetSpotMarginLiability should decode every field")
		return
	}
	assert.Equal(t, currency.BTC, result.Currency, "GetSpotMarginLiability should return the requested currency")
}

func TestGetSpotMarginFlexibleAvailableInventory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotMarginFlexibleAvailableInventory(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSpotMarginFlexibleAvailableInventory must reject a request without a currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotMarginFlexibleAvailableInventory(t.Context(), currency.BTC)
	require.NoError(t, err, "GetSpotMarginFlexibleAvailableInventory must not error")
	if mockTests {
		exp := &SpotMarginFlexibleAvailableInventoryResponse{
			Currency:           currency.BTC,
			AvailableInventory: 17.54689892,
			UpdateTime:         types.Time(time.UnixMilli(1756261353733)),
		}
		assert.Equal(t, exp, result, "GetSpotMarginFlexibleAvailableInventory should decode every field")
		return
	}
	assert.Equal(t, currency.BTC, result.Currency, "GetSpotMarginFlexibleAvailableInventory should return the requested currency")
}

func TestGetSpotMarginFixedRateAvailableInventory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *SpotMarginFixedRateAvailableInventoryRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no currency", req: &SpotMarginFixedRateAvailableInventoryRequest{}, err: currency.ErrCurrencyCodeEmpty},
		{name: "no term", req: &SpotMarginFixedRateAvailableInventoryRequest{Currency: currency.USDT}, err: errTermNotSet},
		{name: "no annual rate", req: &SpotMarginFixedRateAvailableInventoryRequest{Currency: currency.USDT, Term: 7}, err: errAnnualRateNotSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSpotMarginFixedRateAvailableInventory(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetSpotMarginFixedRateAvailableInventory should return the expected error")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotMarginFixedRateAvailableInventory(t.Context(), &SpotMarginFixedRateAvailableInventoryRequest{Currency: currency.USDT, Term: 7, AnnualRate: 0.02})
	require.NoError(t, err, "GetSpotMarginFixedRateAvailableInventory must not error")
	if mockTests {
		exp := &SpotMarginFixedRateAvailableInventoryResponse{
			Currency:           currency.USDT,
			Term:               7,
			AnnualRate:         0.02,
			AvailableInventory: 500000,
			UpdateTime:         types.Time(time.Unix(1699272487, 0)),
		}
		assert.Equal(t, exp, result, "GetSpotMarginFixedRateAvailableInventory should decode every field")
		return
	}
	assert.Equal(t, currency.USDT, result.Currency, "GetSpotMarginFixedRateAvailableInventory should return the requested currency")
}
