package kucoin

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetUTAPositionList(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAPositionList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAPositionList must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *UTAPositionListRequest
		exp  []UTAPosition
	}{
		{
			name: "one symbol",
			req:  &UTAPositionListRequest{Symbol: "XRPUSDTM", PageNumber: 1, PageSize: 50},
			exp: []UTAPosition{
				{
					Symbol:                "XRPUSDTM",
					ID:                    "10000000000904102",
					MarginMode:            "ISOLATED",
					Size:                  1,
					EntryPrice:            1.0101,
					PositionValue:         10.1023,
					MarkPrice:             1.01023,
					Leverage:              3,
					UnrealisedPNL:         0.0013,
					RealisedPNL:           -0.0060606,
					InitialMargin:         3.3669999996633,
					MaintenanceMarginRate: 0.007,
					MaintenanceMargin:     0.0707161,
					CreationTime:          types.Time(time.UnixMilli(1786591827251)),
					LiquidationPrice:      0.67855703345425231761,
					ADLPercentage:         0.12,
					RiskRatio:             0.02279413353917406406,
					UpdateTime:            types.Time(time.UnixMilli(1786591827251)),
					PositionMargin:        3.367,
				},
			},
		},
		{
			name: "every position",
			req:  &UTAPositionListRequest{},
			exp: []UTAPosition{
				{
					Symbol:                "XRPUSDTM",
					ID:                    "10000000000904102",
					MarginMode:            "ISOLATED",
					Size:                  1,
					EntryPrice:            1.0101,
					PositionValue:         10.1023,
					MarkPrice:             1.01023,
					Leverage:              3,
					UnrealisedPNL:         0.0013,
					RealisedPNL:           -0.0060606,
					InitialMargin:         3.3669999996633,
					MaintenanceMarginRate: 0.007,
					MaintenanceMargin:     0.0707161,
					CreationTime:          types.Time(time.UnixMilli(1786591827251)),
					LiquidationPrice:      0.67855703345425231761,
					ADLPercentage:         0.12,
					RiskRatio:             0.02279413353917406406,
					UpdateTime:            types.Time(time.UnixMilli(1786591827251)),
					PositionMargin:        3.367,
				},
				{
					Symbol:                "XBTUSDTM",
					ID:                    "10000000000904388",
					MarginMode:            "CROSS",
					Size:                  -2,
					EntryPrice:            78412.1,
					PositionValue:         -156.8604,
					MarkPrice:             78430.2,
					Leverage:              20,
					UnrealisedPNL:         -0.0362,
					RealisedPNL:           -0.09409452,
					InitialMargin:         7.84302,
					MaintenanceMarginRate: 0.004,
					MaintenanceMargin:     0.6274416,
					CreationTime:          types.Time(time.UnixMilli(1788857447893)),
					LiquidationPrice:      112946.38,
					ADLPercentage:         0.47,
					UpdateTime:            types.Time(time.UnixMilli(1788881466353)),
					PositionMargin:        7.84302,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAPositionList(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAPositionList must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAPositionList should decode every field")
				return
			}
			for i := range result {
				assert.NotEmptyf(t, result[i].Symbol, "GetUTAPositionList should return the symbol of position %d", i)
			}
		})
	}
}

func TestGetUTAPositionsHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAPositionsHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAPositionsHistory must reject a nil request")
	_, err = e.GetUTAPositionsHistory(t.Context(), &UTAPositionsHistoryRequest{StartAt: time.UnixMilli(1771977600000), EndAt: time.UnixMilli(1771372800000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTAPositionsHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUTAPositionsHistory(t.Context(), &UTAPositionsHistoryRequest{
		Symbol: "XBTUSDTM", StartAt: time.UnixMilli(1771372800000), EndAt: time.UnixMilli(1771977600000), LastID: 600, PageSize: 10,
	})
	require.NoError(t, err, "GetUTAPositionsHistory must not error")
	if mockTests {
		exp := &UTAPositionsHistoryResponse{
			Items: []UTAClosedPosition{
				{
					CloseID:           "556",
					Symbol:            "XBTUSDTM",
					MarginMode:        "CROSS",
					Side:              "LONG",
					EntryPrice:        66895.7,
					ClosePrice:        62889.8,
					AverageClosePrice: 62889.8,
					MaxSize:           5,
					Leverage:          25,
					RealisedPNL:       -20.27523235,
					Fee:               0.3893565,
					Tax:               0.00038936,
					FundingFee:        0.14362415,
					CreationTime:      types.Time(time.UnixMilli(1771495960604)),
					ClosingTime:       types.Time(time.UnixMilli(1771940011060)),
				},
			},
			LastID: 556,
		}
		assert.Equal(t, exp, result, "GetUTAPositionsHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUTAPositionsHistory should return a page")
}

func TestGetUTAPrivateFundingFeeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAPrivateFundingFeeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAPrivateFundingFeeHistory must reject a nil request")
	_, err = e.GetUTAPrivateFundingFeeHistory(t.Context(), &UTAPrivateFundingFeeHistoryRequest{StartAt: time.UnixMilli(1776211200000), EndAt: time.UnixMilli(1776038400000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTAPrivateFundingFeeHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUTAPrivateFundingFeeHistory(t.Context(), &UTAPrivateFundingFeeHistoryRequest{
		Symbol: "XBTUSDTM", StartAt: time.UnixMilli(1776038400000), EndAt: time.UnixMilli(1776211200000), LastID: 2125247172539600, PageSize: 50,
	})
	require.NoError(t, err, "GetUTAPrivateFundingFeeHistory must not error")
	if mockTests {
		exp := &UTAPrivateFundingFeeHistoryResponse{
			LastID: 2125247172539515,
			Items: []UTAFundingFeeRecord{
				{
					Symbol:         "XBTUSDTM",
					MarginMode:     "CROSS",
					FundingRate:    -0.000147,
					MarkPrice:      74540.89,
					Size:           5,
					PositionValue:  372.70445,
					FundingFee:     0.05478755,
					SettleCurrency: currency.USDT,
					SettlementTime: types.Time(time.UnixMilli(1776153600000)),
				},
				{
					Symbol:         "XBTUSDTM",
					MarginMode:     "CROSS",
					FundingRate:    -0.000315,
					MarkPrice:      74403.4,
					Size:           3,
					PositionValue:  223.2102,
					FundingFee:     0.07031121,
					SettleCurrency: currency.USDT,
					SettlementTime: types.Time(time.UnixMilli(1776124800000)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetUTAPrivateFundingFeeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUTAPrivateFundingFeeHistory should return a page")
}

func TestModifyUTAMarginMode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		symbols    []string
		marginMode string
		err        error
	}{
		{name: "no symbols", marginMode: "ISOLATED", err: currency.ErrSymbolStringEmpty},
		{name: "empty symbol", symbols: []string{"ETHUSDTM", ""}, marginMode: "ISOLATED", err: currency.ErrSymbolStringEmpty},
		{name: "no margin mode", symbols: []string{"ETHUSDTM"}, err: margin.ErrInvalidMarginType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.ModifyUTAMarginMode(t.Context(), tc.symbols, tc.marginMode)
			assert.ErrorIs(t, err, tc.err, "ModifyUTAMarginMode should reject the request")
		})
	}

	ex, sent := newSigningTestInstance(t, `{"code":"200000","data":{"ts":1779370293039,"items":[]}}`)
	_, err := ex.ModifyUTAMarginMode(t.Context(), []string{"ETHUSDTM", "XRPUSDTM"}, "ISOLATED")
	require.NoError(t, err, "ModifyUTAMarginMode must not error against the local server")
	require.Len(t, *sent, 1, "the local server must receive one request")
	assert.JSONEq(t, `{"symbol":"ETHUSDTM, XRPUSDTM","marginMode":"ISOLATED"}`, (*sent)[0].body, "ModifyUTAMarginMode should list the symbols as the docs' example does")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ModifyUTAMarginMode(t.Context(), []string{"ETHUSDTM", "XRPUSDTM"}, "ISOLATED")
	require.NoError(t, err, "ModifyUTAMarginMode must not error")
	if mockTests {
		exp := &ModifyUTAMarginModeResponse{
			Timestamp: types.Time(time.UnixMilli(1779370293039)),
			Items: []UTAMarginModeResult{
				{
					Message:    "success",
					Code:       "200000",
					Symbol:     "ETHUSDTM",
					MarginMode: "ISOLATED",
				},
				{
					Message:    "success",
					Code:       "200000",
					Symbol:     "XRPUSDTM",
					MarginMode: "ISOLATED",
				},
			},
		}
		assert.Equal(t, exp, result, "ModifyUTAMarginMode should decode every field")
		return
	}
	assert.Len(t, result.Items, 2, "ModifyUTAMarginMode should return the outcome of each symbol")
}

func TestModifyUTAPositionMargin(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *ModifyUTAPositionMarginRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no type", req: &ModifyUTAPositionMarginRequest{Amount: 1, Symbol: "ETHUSDTM", TradeType: "FUTURES"}, err: errMarginChangeTypeRequired},
		{name: "no amount", req: &ModifyUTAPositionMarginRequest{Type: "DEPOSIT", Symbol: "ETHUSDTM", TradeType: "FUTURES"}, err: order.ErrAmountIsInvalid},
		{name: "negative amount", req: &ModifyUTAPositionMarginRequest{Type: "DEPOSIT", Amount: -1, Symbol: "ETHUSDTM", TradeType: "FUTURES"}, err: order.ErrAmountIsInvalid},
		{name: "no symbol", req: &ModifyUTAPositionMarginRequest{Type: "DEPOSIT", Amount: 1, TradeType: "FUTURES"}, err: currency.ErrSymbolStringEmpty},
		{name: "no trade type", req: &ModifyUTAPositionMarginRequest{Type: "DEPOSIT", Amount: 1, Symbol: "ETHUSDTM"}, err: errTradeTypeMissing},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.ModifyUTAPositionMargin(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "ModifyUTAPositionMargin should reject the request")
		})
	}

	ex, sent := newSigningTestInstance(t, `{"code":"200000","data":{"ts":1779367360372}}`)
	_, err := ex.ModifyUTAPositionMargin(t.Context(), &ModifyUTAPositionMarginRequest{Type: "DEPOSIT", Amount: 1, Symbol: "ETHUSDTM", TradeType: "FUTURES"})
	require.NoError(t, err, "ModifyUTAPositionMargin must not error against the local server")
	require.Len(t, *sent, 1, "the local server must receive one request")
	assert.JSONEq(t, `{"type":"DEPOSIT","amount":1,"symbol":"ETHUSDTM","tradeType":"FUTURES"}`, (*sent)[0].body, "ModifyUTAPositionMargin should send the amount as a bare number")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		positions, err := e.GetUTAPositionList(t.Context(), &UTAPositionListRequest{})
		require.NoError(t, err, "GetUTAPositionList must not error")
		i := slices.IndexFunc(positions, func(p UTAPosition) bool {
			return p.MarginMode == "ISOLATED" && strings.HasSuffix(p.Symbol, "USDTM")
		})
		if i < 0 {
			t.Skip("ModifyUTAPositionMargin needs an isolated USDT-margined position, which the account does not hold")
		}
		// The margin added is withdrawn again, leaving the position as it was
		for _, changeType := range []string{"DEPOSIT", "WITHDRAW"} {
			result, err := e.ModifyUTAPositionMargin(t.Context(), &ModifyUTAPositionMarginRequest{Type: changeType, Amount: 1, Symbol: positions[i].Symbol, TradeType: "FUTURES"})
			require.NoErrorf(t, err, "ModifyUTAPositionMargin must not error for %s", changeType)
			require.NotNilf(t, result, "ModifyUTAPositionMargin must return a result for %s", changeType)
			assert.Falsef(t, result.Timestamp.Time().IsZero(), "ModifyUTAPositionMargin should return when it changed the margin for %s", changeType)
		}
		return
	}
	result, err := e.ModifyUTAPositionMargin(t.Context(), &ModifyUTAPositionMarginRequest{Type: "DEPOSIT", Amount: 1, Symbol: "ETHUSDTM", TradeType: "FUTURES"})
	require.NoError(t, err, "ModifyUTAPositionMargin must not error")
	exp := &ModifyUTAPositionMarginResponse{
		Timestamp: types.Time(time.UnixMilli(1779367360372)),
	}
	assert.Equal(t, exp, result, "ModifyUTAPositionMargin should decode every field")
}

func TestGetUTAMarginMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name   string
		symbol string
		exp    *UTAMarginModeResponse
	}{
		{
			name:   "one symbol",
			symbol: "XBTUSDTM",
			exp: &UTAMarginModeResponse{
				Timestamp: types.Time(time.UnixMilli(1780630392057)),
				Items: []UTASymbolMarginMode{
					{
						Symbol:     "XBTUSDTM",
						MarginMode: "CROSS",
					},
				},
			},
		},
		{
			name: "every symbol",
			exp: &UTAMarginModeResponse{
				Timestamp: types.Time(time.UnixMilli(1780630392057)),
				Items: []UTASymbolMarginMode{
					{
						Symbol:     "XBTUSDTM",
						MarginMode: "CROSS",
					},
					{
						Symbol:     "ETHUSDTM",
						MarginMode: "ISOLATED",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAMarginMode(t.Context(), tc.symbol)
			require.NoError(t, err, "GetUTAMarginMode must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAMarginMode should decode every field")
				return
			}
			assert.NotZero(t, result.Timestamp, "GetUTAMarginMode should return a timestamp")
		})
	}
}
