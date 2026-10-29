package coinbase

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	testFuturesPositionProductID = "BIT-30OCT26-CDE"
	// testFuturesSweepAmount is the USD the sweep tests sweep
	testFuturesSweepAmount = 100
)

// futuresSweepTestMu serialises the live tests that schedule and cancel sweeps, as Coinbase holds one pending sweep at
// a time
var futuresSweepTestMu sync.Mutex

func TestCancelPendingFuturesSweep(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		futuresSweepTestMu.Lock()
		defer futuresSweepTestMu.Unlock()
		require.NoError(t, e.ScheduleFuturesSweep(t.Context(), testFuturesSweepAmount), "ScheduleFuturesSweep must schedule a sweep to cancel")
	}
	require.NoError(t, e.CancelPendingFuturesSweep(t.Context()), "CancelPendingFuturesSweep must not error")

	err := newResponseServerExchange(t, `{"success":false}`).CancelPendingFuturesSweep(t.Context())
	assert.ErrorIs(t, err, errFuturesSweepNotCancelled, "CancelPendingFuturesSweep should error when Coinbase cancels nothing")
	err = newResponseServerExchange(t, `null`).CancelPendingFuturesSweep(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelPendingFuturesSweep should error for a null response")
	err = newUnavailableServerExchange(t).CancelPendingFuturesSweep(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelPendingFuturesSweep should return the request failure")
}

func TestGetCurrentMarginWindow(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrentMarginWindow(t.Context(), "MARGIN_PROFILE_TYPE_RETAIL_CASH")
	require.ErrorIs(t, err, errInvalidMarginProfileType, "GetCurrentMarginWindow must refuse an unknown margin profile type")

	skipLiveWithoutCredentials(t)
	resp, err := e.GetCurrentMarginWindow(t.Context(), MarginProfileTypeRetailRegular)
	require.NoError(t, err, "GetCurrentMarginWindow must not error")
	if mockTests {
		exp := &CurrentMarginWindowResponse{
			MarginWindow: MarginWindow{
				MarginWindowType: "MARGIN_WINDOW_TYPE_OVERNIGHT",
				EndTime:          time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC),
			},
		}
		assert.Equal(t, exp, resp, "GetCurrentMarginWindow should decode every field")
	} else {
		assert.NotEmpty(t, resp.MarginWindow.MarginWindowType, "GetCurrentMarginWindow should return the margin window")
	}

	_, err = newResponseServerExchange(t, `null`).GetCurrentMarginWindow(t.Context(), "")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetCurrentMarginWindow should error for a null response")
	_, err = newUnavailableServerExchange(t).GetCurrentMarginWindow(t.Context(), "")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetCurrentMarginWindow should return the request failure")
}

func TestGetFuturesBalanceSummary(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	resp, err := e.GetFuturesBalanceSummary(t.Context())
	require.NoError(t, err, "GetFuturesBalanceSummary must not error")
	if mockTests {
		exp := &FuturesBalanceSummary{
			FuturesBuyingPower: Amount{
				Value:    1100.5,
				Currency: currency.USD,
			},
			TotalUSDBalance: Amount{
				Value:    1350.5,
				Currency: currency.USD,
			},
			CBIUSDBalance: Amount{
				Value:    1100.5,
				Currency: currency.USD,
			},
			CFMUSDBalance: Amount{
				Value:    250,
				Currency: currency.USD,
			},
			TotalOpenOrdersHoldAmount: Amount{
				Currency: currency.USD,
			},
			UnrealisedPNL: Amount{
				Value:    16.15,
				Currency: currency.USD,
			},
			DailyRealisedPNL: Amount{
				Value:    -1.2,
				Currency: currency.USD,
			},
			InitialMargin: Amount{
				Value:    197.74,
				Currency: currency.USD,
			},
			AvailableMargin: Amount{
				Value:    1366.65,
				Currency: currency.USD,
			},
			LiquidationThreshold: Amount{
				Value:    98.87,
				Currency: currency.USD,
			},
			LiquidationBufferAmount: Amount{
				Value:    1267.78,
				Currency: currency.USD,
			},
			LiquidationBufferPercentage: 1282.27,
			IntradayMarginWindowMeasure: &MarginWindowMeasure{
				MarginWindowType:   "FCM_MARGIN_WINDOW_TYPE_INTRADAY",
				MarginLevel:        "MARGIN_LEVEL_TYPE_BASE",
				InitialMargin:      98.87,
				MaintenanceMargin:  49.44,
				LiquidationBuffer:  1317.21,
				FuturesBuyingPower: 1267.78,
			},
			OvernightMarginWindowMeasure: &MarginWindowMeasure{
				MarginWindowType:   "FCM_MARGIN_WINDOW_TYPE_OVERNIGHT",
				MarginLevel:        "MARGIN_LEVEL_TYPE_BASE",
				InitialMargin:      197.74,
				MaintenanceMargin:  98.87,
				LiquidationBuffer:  1267.78,
				FuturesBuyingPower: 1168.91,
			},
			TotalPendingTransfersAmount: Amount{
				Currency: currency.USD,
			},
			FundingPNL: Amount{
				Currency: currency.USD,
			},
		}
		assert.Equal(t, exp, resp, "GetFuturesBalanceSummary should decode every field")
	} else {
		assert.NotEmpty(t, resp.TotalUSDBalance.Currency, "GetFuturesBalanceSummary should return the USD balance")
	}

	_, err = newResponseServerExchange(t, `null`).GetFuturesBalanceSummary(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFuturesBalanceSummary should error for a null response")
	_, err = newResponseServerExchange(t, `{"balance_summary":null}`).GetFuturesBalanceSummary(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFuturesBalanceSummary should error for a null balance summary")
	_, err = newUnavailableServerExchange(t).GetFuturesBalanceSummary(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFuturesBalanceSummary should return the request failure")
}

func TestGetFuturesPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPosition(t.Context(), "")
	require.ErrorIs(t, err, errProductIDRequired, "GetFuturesPosition must require a product ID")

	skipLiveWithoutCredentials(t)
	productID := testFuturesPositionProductID
	if !mockTests {
		positions, err := e.ListFuturesPositions(t.Context())
		require.NoError(t, err, "ListFuturesPositions must not error")
		if len(positions) == 0 {
			t.Skip("the account holds no US derivatives position")
		}
		productID = positions[0].ProductID
	}
	resp, err := e.GetFuturesPosition(t.Context(), productID)
	require.NoError(t, err, "GetFuturesPosition must not error")
	if mockTests {
		exp := &FuturesPosition{
			ProductID:         "BIT-30OCT26-CDE",
			ExpirationTime:    time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC),
			Side:              "LONG",
			NumberOfContracts: 2,
			CurrentPrice:      82110,
			AverageEntryPrice: 81500,
			UnrealisedPNL:     12.2,
		}
		assert.Equal(t, exp, resp, "GetFuturesPosition should decode every field")
	} else {
		assert.Equal(t, productID, resp.ProductID, "GetFuturesPosition should return the product's position")
	}

	_, err = newResponseServerExchange(t, `null`).GetFuturesPosition(t.Context(), testFuturesPositionProductID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFuturesPosition should error for a null response")
	_, err = newResponseServerExchange(t, `{"position":null}`).GetFuturesPosition(t.Context(), testFuturesPositionProductID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFuturesPosition should error for a null position")
	_, err = newUnavailableServerExchange(t).GetFuturesPosition(t.Context(), testFuturesPositionProductID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFuturesPosition should return the request failure")
}

func TestGetIntradayMarginSetting(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	resp, err := e.GetIntradayMarginSetting(t.Context())
	require.NoError(t, err, "GetIntradayMarginSetting must not error")
	if mockTests {
		assert.Equal(t, IntradayMarginSettingStandard, resp, "GetIntradayMarginSetting should decode the setting")
	} else {
		assert.NotEmpty(t, resp, "GetIntradayMarginSetting should return the setting")
	}

	_, err = newResponseServerExchange(t, `null`).GetIntradayMarginSetting(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetIntradayMarginSetting should error for a null response")
	_, err = newUnavailableServerExchange(t).GetIntradayMarginSetting(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetIntradayMarginSetting should return the request failure")
}

func TestListFuturesPositions(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	resp, err := e.ListFuturesPositions(t.Context())
	require.NoError(t, err, "ListFuturesPositions must not error")
	if mockTests {
		exp := []FuturesPosition{
			{
				ProductID:         "BIT-30OCT26-CDE",
				ExpirationTime:    time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC),
				Side:              "LONG",
				NumberOfContracts: 2,
				CurrentPrice:      82110,
				AverageEntryPrice: 81500,
				UnrealisedPNL:     12.2,
			},
			{
				ProductID:         "ET-30OCT26-CDE",
				ExpirationTime:    time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC),
				Side:              "SHORT",
				NumberOfContracts: 1,
				CurrentPrice:      2410.5,
				AverageEntryPrice: 2450,
				UnrealisedPNL:     3.95,
				DailyRealisedPNL:  -1.2,
			},
		}
		assert.Equal(t, exp, resp, "ListFuturesPositions should decode every field")
	} else {
		for i := range resp {
			assert.NotEmptyf(t, resp[i].ProductID, "ListFuturesPositions should return position %d's product", i)
		}
	}

	_, err = newUnavailableServerExchange(t).ListFuturesPositions(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListFuturesPositions should return the request failure")
}

func TestListFuturesSweeps(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	resp, err := e.ListFuturesSweeps(t.Context())
	require.NoError(t, err, "ListFuturesSweeps must not error")
	if mockTests {
		exp := []FuturesSweep{
			{
				ID: "9c86a6f3-8a55-4a2e-9e6c-0e1c3d5b7f20",
				RequestedAmount: Amount{
					Value:    100,
					Currency: currency.USD,
				},
				Status:        "PENDING",
				ScheduledTime: time.Date(2026, 10, 18, 22, 5, 11, 204000000, time.UTC),
			},
		}
		assert.Equal(t, exp, resp, "ListFuturesSweeps should decode every field")
	} else {
		for i := range resp {
			assert.NotEmptyf(t, resp[i].ID, "ListFuturesSweeps should return sweep %d's ID", i)
		}
	}

	_, err = newUnavailableServerExchange(t).ListFuturesSweeps(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListFuturesSweeps should return the request failure")
}

func TestScheduleFuturesSweep(t *testing.T) {
	t.Parallel()
	err := e.ScheduleFuturesSweep(t.Context(), -1)
	require.ErrorIs(t, err, errFuturesSweepAmount, "ScheduleFuturesSweep must refuse a negative amount")
	// An amount that is not a number would otherwise sweep every excess fund, as a zero amount does
	for _, amount := range []float64{math.NaN(), math.Inf(1)} {
		require.ErrorIsf(t, e.ScheduleFuturesSweep(t.Context(), amount), errFuturesSweepAmount, "ScheduleFuturesSweep must refuse %v", amount)
	}

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		futuresSweepTestMu.Lock()
		defer futuresSweepTestMu.Unlock()
	}
	require.NoError(t, e.ScheduleFuturesSweep(t.Context(), testFuturesSweepAmount), "ScheduleFuturesSweep must not error")
	if mockTests {
		// The mock only answers the body it recorded, so this proves a zero amount leaves usd_amount out
		assert.NoError(t, e.ScheduleFuturesSweep(t.Context(), 0), "ScheduleFuturesSweep should sweep every excess fund for a zero amount")
	} else {
		assert.NoError(t, e.CancelPendingFuturesSweep(t.Context()), "CancelPendingFuturesSweep should cancel the scheduled sweep")
	}

	err = newResponseServerExchange(t, `{"success":false}`).ScheduleFuturesSweep(t.Context(), testFuturesSweepAmount)
	assert.ErrorIs(t, err, errFuturesSweepNotScheduled, "ScheduleFuturesSweep should error when Coinbase schedules nothing")
	err = newResponseServerExchange(t, `null`).ScheduleFuturesSweep(t.Context(), testFuturesSweepAmount)
	assert.ErrorIs(t, err, common.ErrNoResponse, "ScheduleFuturesSweep should error for a null response")
	err = newUnavailableServerExchange(t).ScheduleFuturesSweep(t.Context(), testFuturesSweepAmount)
	assert.ErrorIs(t, err, request.ErrBadStatus, "ScheduleFuturesSweep should return the request failure")
}

func TestSetIntradayMarginSetting(t *testing.T) {
	t.Parallel()
	for _, setting := range []string{"", "INTRADAY_MARGIN_SETTING_UNSPECIFIED"} {
		err := e.SetIntradayMarginSetting(t.Context(), setting)
		require.ErrorIsf(t, err, errInvalidIntradayMarginSetting, "SetIntradayMarginSetting must refuse %q", setting)
	}

	skipLiveUnlessCanManipulate(t)
	setting := IntradayMarginSettingIntraday
	if !mockTests {
		current, err := e.GetIntradayMarginSetting(t.Context())
		require.NoError(t, err, "GetIntradayMarginSetting must not error")
		// Standard margin is what an account that never opted in has, so only a setting the account chose is flipped
		setting = IntradayMarginSettingStandard
		if current == IntradayMarginSettingStandard {
			setting = IntradayMarginSettingIntraday
		}
		if current == IntradayMarginSettingStandard || current == IntradayMarginSettingIntraday {
			t.Cleanup(func() {
				// The test's context ends before its cleanups run
				assert.NoError(t, e.SetIntradayMarginSetting(context.WithoutCancel(t.Context()), current), "SetIntradayMarginSetting should restore the setting")
			})
		}
	}
	require.NoError(t, e.SetIntradayMarginSetting(t.Context(), setting), "SetIntradayMarginSetting must not error")

	err := newUnavailableServerExchange(t).SetIntradayMarginSetting(t.Context(), IntradayMarginSettingStandard)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SetIntradayMarginSetting should return the request failure")
}
