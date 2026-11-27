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

func TestGetFuturesCurrentFundingRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesCurrentFundingRate(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesCurrentFundingRate must reject an empty symbol")

	result, err := e.GetFuturesCurrentFundingRate(t.Context(), "XBTUSDTM")
	require.NoError(t, err, "GetFuturesCurrentFundingRate must not error")
	if mockTests {
		exp := &FuturesFundingRateResponse{
			Symbol:              ".XBTUSDTMFPI8H",
			Granularity:         28800000,
			TimePoint:           types.Time(time.Unix(1791878400, 0)),
			Value:               0.000052,
			PredictedValue:      0.000048,
			DailyInterestRate:   0.0003,
			FundingRateCap:      0.003,
			FundingRateFloor:    -0.003,
			Period:              1,
			FundingTime:         types.Time(time.Unix(1791907200, 0)),
			LastTimeFundingRate: 0.000072,
		}
		assert.Equal(t, exp, result, "GetFuturesCurrentFundingRate should decode every field")
		return
	}
	require.NotNil(t, result, "GetFuturesCurrentFundingRate must return a funding rate")
	assert.Equal(t, ".XBTUSDTMFPI8H", result.Symbol, "GetFuturesCurrentFundingRate should return the contract's funding rate symbol")
	assert.True(t, result.FundingTime.Time().After(result.TimePoint.Time()), "FundingTime should follow TimePoint")
}

func TestGetPublicFundingRate(t *testing.T) {
	t.Parallel()
	from, to := time.UnixMilli(1791763200000), time.UnixMilli(1791820800000)
	_, err := e.GetPublicFundingRate(t.Context(), "", from, to)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetPublicFundingRate must reject an empty symbol")
	_, err = e.GetPublicFundingRate(t.Context(), "XBTUSDTM", time.Time{}, to)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetPublicFundingRate must reject an unset start")
	_, err = e.GetPublicFundingRate(t.Context(), "XBTUSDTM", to, from)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPublicFundingRate must reject a start after the end")

	result, err := e.GetPublicFundingRate(t.Context(), "XBTUSDTM", from, to)
	require.NoError(t, err, "GetPublicFundingRate must not error")
	if mockTests {
		exp := []FuturesPublicFundingHistoryResponse{
			{
				Symbol:      "XBTUSDTM",
				FundingRate: 0.000067,
				Timepoint:   types.Time(time.Unix(1791820800, 0)),
			},
			{
				Symbol:      "XBTUSDTM",
				FundingRate: 0.000031,
				Timepoint:   types.Time(time.Unix(1791792000, 0)),
			},
			{
				Symbol:      "XBTUSDTM",
				FundingRate: 0.000035,
				Timepoint:   types.Time(time.Unix(1791763200, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetPublicFundingRate should decode every field")
		return
	}
	require.NotEmpty(t, result, "GetPublicFundingRate must return funding rates")
	for i := range result {
		assert.Equal(t, "XBTUSDTM", result[i].Symbol, "GetPublicFundingRate should return the requested contract")
		assert.WithinRange(t, result[i].Timepoint.Time(), from, to, "Timepoint should fall within the requested window")
	}
}

func TestGetFuturesFundingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesFundingHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesFundingHistory must reject a nil request")
	_, err = e.GetFuturesFundingHistory(t.Context(), &FuturesHistoryRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesFundingHistory must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesFundingHistory(t.Context(), &FuturesHistoryRequest{Symbol: "XBTUSDTM", Reverse: new(true), Forward: new(true)})
	require.NoError(t, err, "GetFuturesFundingHistory must not error")
	if mockTests {
		exp := &FuturesPrivateFundingHistoryResponse{
			DataList: []FuturesPrivateFundingHistory{
				{
					ID:               1472387374042586,
					Symbol:           "XBTUSDTM",
					TimePoint:        types.Time(time.Unix(1731470400, 0)),
					FundingRate:      0.000641,
					MarkPrice:        87139.92,
					PositionQuantity: 1,
					PositionCost:     87.13992,
					Funding:          -0.05585669,
					SettleCurrency:   currency.USDT,
					Context:          "{\"marginMode\": \"ISOLATED\", \"positionSide\": \"BOTH\"}",
					MarginMode:       "ISOLATED",
				},
				{
					ID:               1472387374042587,
					Symbol:           "XBTUSDTM",
					TimePoint:        types.Time(time.Unix(1731441600, 0)),
					FundingRate:      0.000412,
					MarkPrice:        86904.1,
					PositionQuantity: -2,
					PositionCost:     -173.8082,
					Funding:          0.07160897,
					SettleCurrency:   currency.USDT,
					Context:          "{\"marginMode\": \"CROSS\", \"positionSide\": \"BOTH\"}",
					MarginMode:       "CROSS",
				},
			},
			HasMore: true,
		}
		assert.Equal(t, exp, result, "GetFuturesFundingHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesFundingHistory should return a page")
}
