package gateio

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func getTimeWithInterval(interval kline.Interval) (startTime, endTime time.Time) {
	if mockTests {
		startTime = time.UnixMilli(1744103854944)
		return startTime, startTime.Add(time.Duration(interval))
	}
	startTime = time.Now()
	return startTime.Add(-time.Hour * 48), startTime
}

// futuresFixtureAccount returns the account the futures fixtures record for a settlement currency
func futuresFixtureAccount(settle currency.Code) *FuturesAccountResponse {
	return &FuturesAccountResponse{
		User:                          1666,
		Currency:                      settle,
		Total:                         9707.803567115145,
		UnrealisedPNL:                 3371.248828,
		PositionMargin:                38.712189181,
		OrderMargin:                   0.5,
		Available:                     9669.091377934145,
		Point:                         1,
		Bonus:                         2,
		EnabledCredit:                 true,
		InDualMode:                    true,
		PositionInitialMargin:         10.5,
		MaintenanceMargin:             2.25,
		MarginMode:                    1,
		EnabledEvolvedClassic:         true,
		CrossOrderMargin:              3,
		CrossInitialMargin:            61855.56788525,
		CrossMaintenanceMargin:        682.04678105,
		CrossUnrealisedPNL:            1501.178222634128,
		CrossAvailable:                27549.406108813951,
		CrossMarginBalance:            10371.77306201952,
		CrossMaintenanceMarginRate:    797.2134,
		CrossInitialMarginRate:        116.6097,
		IsolatedPositionMargin:        4,
		EnableNewDualMode:             true,
		EnableTieredMaintenanceMargin: true,
		EnableDualPlus:                true,
		PositionMode:                  "dual_plus",
		History: FuturesAccountHistory{
			DepositAndWithdrawal:      10000,
			ProfitAndLoss:             68.3685,
			Fee:                       -1.645812875,
			ReferrerRebate:            5,
			Funding:                   -358.919120009855,
			PointDepositAndWithdrawal: 6,
			PointFee:                  7,
			PointReferrerRebate:       8,
			BonusDepositAndWithdrawal: 9,
			BonusOffset:               10,
			CrossSettle:               12.5,
		},
	}
}

// futuresFixtureOrder returns the order the futures fixtures record for a contract
func futuresFixtureOrder(contract currency.Pair) *FuturesOrderResponse {
	return &FuturesOrderResponse{
		ID:                        15675394,
		User:                      "100000",
		Contract:                  contract,
		CreateTime:                types.Time(time.Unix(1546569968, 0)),
		UpdateTime:                types.Time(time.UnixMilli(1546569970123)),
		Size:                      6024,
		RemainingAmount:           6024,
		Price:                     3765,
		MakerFeeRate:              -0.00025,
		TakerFeeRate:              0.00075,
		TimeInForce:               "gtc",
		Text:                      "t-my-custom-id",
		Status:                    "finished",
		FinishTime:                types.Time(time.Unix(1514764900, 0)),
		FinishAs:                  "cancelled",
		SelfTradePreventionAction: "-",
		AmendText:                 "-",
		OrderValue:                64112.2099,
		TradeValue:                64112.2099,
		MarketOrderSlipRatio:      0.03,
		PositionMarginMode:        "isolated",
		ActionMode:                "FULL",
		TakeProfitTriggerPrice:    3800,
		StopLossTriggerPrice:      3700,
	}
}

// futuresFixtureTrailOrder returns the trail order the futures fixtures record
func futuresFixtureTrailOrder() *TrailOrderResponse {
	return &TrailOrderResponse{
		ID:                 "63585",
		UserID:             "2124438083",
		User:               "2124438083",
		Contract:           currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
		Settle:             "usdt",
		Amount:             10,
		IsGreaterThanEqual: true,
		ActivationPrice:    50000,
		PriceType:          1,
		PriceOffset:        "5%",
		Text:               "apiv4",
		ReduceOnly:         true,
		PositionRelated:    true,
		CreatedAt:          types.Time(time.Unix(1769569837, 0)),
		ActivatedAt:        types.Time(time.Unix(1769569838, 0)),
		FinishedAt:         types.Time(time.Unix(1769578529, 0)),
		CreateTime:         types.Time(time.Unix(1769569837, 0)),
		ActiveTime:         types.Time(time.Unix(1769569838, 0)),
		FinishTime:         types.Time(time.Unix(1769578529, 0)),
		Reason:             "success",
		SuborderText:       "apiv4-auto-trail-o-1d29",
		IsDualMode:         true,
		TriggerPrice:       91047.4,
		SuborderID:         "94294117233225616",
		SideLabel:          "buy",
		OriginalStatus:     4,
		Status:             "finished",
		PositionSideOutput: "buy",
		UpdatedAt:          types.Time(time.Unix(1769578529, 0)),
		ExtremumPrice:      86711.9,
		StatusCode:         "success",
		CreatedAtPrecise:   types.Time(time.UnixMicro(1769569837778000)),
		FinishedAtPrecise:  types.Time(time.UnixMicro(1769578529853294)),
		ActivatedAtPrecise: types.Time(time.UnixMicro(1769569837976010)),
		StatusLabel:        "finished",
		PositionMarginMode: "cross",
		PositionMode:       "single",
		Leverage:           10,
	}
}

// futuresFixtureChaseOrder returns the chase order the futures fixtures record
func futuresFixtureChaseOrder() *ChaseOrderResponse {
	return &ChaseOrderResponse{
		ID:                 "1001",
		User:               "2124438083",
		Contract:           currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
		Settle:             "usdt",
		Amount:             1,
		PriceLimit:         68000,
		ReduceOnly:         true,
		Text:               "t-chase",
		CreateTime:         types.Time(time.Unix(1774991579, 0)),
		FinishTime:         types.Time(time.Unix(1774991679, 0)),
		OriginalStatus:     3,
		Status:             "finished",
		Reason:             "filled",
		FillAmount:         1,
		AverageFillPrice:   67900.1,
		SuborderID:         "94294117233225616",
		IsDualMode:         true,
		SideLabel:          "buy",
		PositionSideOutput: "buy",
		ChasePrice:         67900.1,
		IntervalSeconds:    5,
		UpdatedAt:          types.Time(time.Unix(1774991679, 0)),
		SuborderPrice:      67900.1,
		SuborderFinishAs:   "filled",
		PriceType:          2,
		PriceGapType:       "1",
		PriceGapValue:      0.5,
		StatusCode:         "success",
		CreateTimePrecise:  types.Time(time.UnixMicro(1774991579123456)),
		FinishTimePrecise:  types.Time(time.UnixMicro(1774991679654321)),
		PositionMarginMode: "cross",
		PositionMode:       "dual",
		Leverage:           10,
	}
}

// futuresFixturePriceTriggeredOrder returns the price triggered order the futures fixtures record
func futuresFixturePriceTriggeredOrder() *PriceTriggeredOrderResponse {
	return &PriceTriggeredOrderResponse{
		Initial: PriceTriggeredOrderInitial{
			Contract:     currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
			Size:         2,
			Amount:       2,
			Price:        1234,
			TimeInForce:  "gtc",
			Text:         "api",
			ReduceOnly:   true,
			IsReduceOnly: true,
			IsClose:      true,
		},
		Trigger: PriceTriggeredOrderTrigger{
			PriceType:   1,
			Price:       3000,
			Rule:        1,
			ExpireAfter: types.DurationSeconds(24 * time.Hour),
		},
		ID:                 1432329,
		IDString:           "1432329",
		User:               1666,
		CreateTime:         types.Time(time.Unix(1514764800, 0)),
		FinishTime:         types.Time(time.Unix(1514764900, 0)),
		TradeID:            13566,
		Status:             "finished",
		FinishAs:           "succeeded",
		Reason:             "triggered",
		OrderType:          "close-short-position",
		MatchingOrderID:    15675394,
		PositionMarginMode: "isolated",
	}
}

func TestGetAllFutureContracts(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllFutureContracts(t.Context(), currency.EMPTYCODE, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAllFutureContracts must reject an empty settlement currency")

	for _, c := range []currency.Code{currency.BTC, currency.USDT} {
		result, err := e.GetAllFutureContracts(t.Context(), c, 0, 0)
		require.NoErrorf(t, err, "GetAllFutureContracts must not error for %s", c)
		assert.NotEmptyf(t, result, "GetAllFutureContracts should return contracts for %s", c)
	}

	result, err := e.GetAllFutureContracts(t.Context(), currency.USDT, 1, 1)
	require.NoError(t, err, "GetAllFutureContracts must not error with a limit and offset")
	require.Len(t, result, 1, "GetAllFutureContracts must honour the limit")
	if mockTests {
		exp := &FuturesContractResponse{
			Name:                  currency.NewPairWithDelimiter("BTC", "USDT", "_"),
			Type:                  "direct",
			QuantoMultiplier:      0.0001,
			OrderPriceDeviate:     0.5,
			MaintenanceRate:       0.005,
			MarkType:              "index",
			LastPrice:             38026,
			MarkPrice:             37985.6,
			IndexPrice:            37954.92,
			FundingRateIndicative: 0.000219,
			MarkPriceRound:        0.01,
			RiskLimitBase:         1000000,
			InterestRate:          0.0003,
			OrderPriceRound:       0.1,
			OrderSizeMin:          1,
			RefRebateRate:         0.2,
			FundingInterval:       types.DurationSeconds(8 * time.Hour),
			RiskLimitStep:         1000000,
			LeverageMin:           1,
			LeverageMax:           100,
			RiskLimitMax:          8000000,
			MakerFeeRate:          -0.00025,
			TakerFeeRate:          0.00075,
			FundingRate:           0.002053,
			OrderSizeMax:          1000000,
			FundingNextApply:      types.Time(time.Unix(1610035200, 0)),
			ShortUsers:            977,
			ConfigChangeTime:      types.Time(time.Unix(1609899548, 0)),
			TradeSize:             28530850594,
			PositionSize:          5223816,
			LongUsers:             455,
			FundingImpactValue:    60000,
			OrdersLimit:           50,
			TradeID:               10851092,
			OrderbookID:           2129638396,
			EnableBonus:           true,
			EnableCredit:          true,
			CreateTime:            types.Time(time.Unix(1669688556, 0)),
			FundingCapRatio:       0.75,
			Status:                "trading",
			LaunchTime:            types.Time(time.Unix(1609899548, 0)),
			DelistingTime:         types.Time(time.Unix(1609899549, 0)),
			DelistedTime:          types.Time(time.Unix(1609899550, 0)),
			EnableDecimal:         true,
			MarketOrderSlipRatio:  0.05,
			MarketOrderSizeMax:    500000,
			FundingRateLimit:      0.003,
			ContractType:          "indices",
			EnableCircuitBreaker:  true,
			CrossLeverageDefault:  10,
			VoucherLeverage:       2,
			IsPreMarket:           true,
		}
		assert.Equal(t, exp, result[0], "GetAllFutureContracts should decode every field")
	}
}

func TestGetAllFutureContractsIncludingDelisted(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllFutureContractsIncludingDelisted(t.Context(), currency.EMPTYCODE, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAllFutureContractsIncludingDelisted must reject an empty settlement currency")

	result, err := e.GetAllFutureContractsIncludingDelisted(t.Context(), currency.USDT, 1, 0)
	require.NoError(t, err, "GetAllFutureContractsIncludingDelisted must not error")
	require.Len(t, result, 1, "GetAllFutureContractsIncludingDelisted must honour the limit")
	if mockTests {
		exp := &FuturesContractResponse{
			Name:                  currency.NewPairWithDelimiter("LUNA", "USDT", "_"),
			Type:                  "direct",
			QuantoMultiplier:      0.0001,
			OrderPriceDeviate:     0.5,
			MaintenanceRate:       0.005,
			MarkType:              "index",
			LastPrice:             0.0001,
			MarkPrice:             0.0001,
			IndexPrice:            0.0001,
			FundingRateIndicative: 0.000219,
			MarkPriceRound:        0.01,
			Delisting:             true,
			RiskLimitBase:         1000000,
			InterestRate:          0.0003,
			OrderPriceRound:       0.1,
			OrderSizeMin:          1,
			RefRebateRate:         0.2,
			FundingInterval:       types.DurationSeconds(8 * time.Hour),
			RiskLimitStep:         1000000,
			LeverageMin:           1,
			LeverageMax:           100,
			RiskLimitMax:          8000000,
			MakerFeeRate:          -0.00025,
			TakerFeeRate:          0.00075,
			FundingRate:           0.002053,
			OrderSizeMax:          1000000,
			FundingNextApply:      types.Time(time.Unix(1610035200, 0)),
			ShortUsers:            977,
			ConfigChangeTime:      types.Time(time.Unix(1609899548, 0)),
			TradeSize:             28530850594,
			LongUsers:             455,
			FundingImpactValue:    60000,
			OrdersLimit:           50,
			TradeID:               10851092,
			OrderbookID:           2129638396,
			EnableBonus:           true,
			EnableCredit:          true,
			CreateTime:            types.Time(time.Unix(1669688556, 0)),
			FundingCapRatio:       0.75,
			Status:                "delisted",
			LaunchTime:            types.Time(time.Unix(1609899548, 0)),
			DelistingTime:         types.Time(time.Unix(1609899549, 0)),
			DelistedTime:          types.Time(time.Unix(1609899550, 0)),
			MarketOrderSlipRatio:  0.05,
			MarketOrderSizeMax:    500000,
			FundingRateLimit:      0.003,
			CrossLeverageDefault:  10,
			VoucherLeverage:       2,
		}
		assert.Equal(t, exp, result[0], "GetAllFutureContractsIncludingDelisted should decode every field")
	}
}

func TestGetFuturesContract(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContract(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesContract must reject an empty contract")

	_, err = e.GetFuturesContract(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures).String())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesContract must reject an empty settlement currency")

	result, err := e.GetFuturesContract(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures).String())
	require.NoError(t, err, "GetFuturesContract must not error for a USDT contract")
	assert.NotNil(t, result, "GetFuturesContract should return a USDT contract")

	result, err = e.GetFuturesContract(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures).String())
	require.NoError(t, err, "GetFuturesContract must not error for a BTC contract")
	if mockTests {
		exp := &FuturesContractResponse{
			Name:                  currency.NewPairWithDelimiter("BTC", "USD", "_"),
			Type:                  "inverse",
			OrderPriceDeviate:     0.5,
			MaintenanceRate:       0.005,
			MarkType:              "index",
			LastPrice:             67295.7,
			MarkPrice:             67501.33,
			IndexPrice:            67499.05,
			FundingRateIndicative: 0.0001,
			MarkPriceRound:        0.01,
			RiskLimitBase:         3,
			InterestRate:          0.0003,
			OrderPriceRound:       0.1,
			OrderSizeMin:          1,
			RefRebateRate:         0.2,
			FundingInterval:       types.DurationSeconds(8 * time.Hour),
			RiskLimitStep:         2497,
			LeverageMin:           1,
			LeverageMax:           100,
			RiskLimitMax:          2500,
			MakerFeeRate:          -0.0002,
			TakerFeeRate:          0.00075,
			FundingRate:           0.0001,
			OrderSizeMax:          600000,
			FundingNextApply:      types.Time(time.Unix(1775433600, 0)),
			ShortUsers:            224,
			ConfigChangeTime:      types.Time(time.Unix(1769580511, 0)),
			TradeSize:             61312185044,
			PositionSize:          8701256,
			LongUsers:             349,
			FundingImpactValue:    0.5,
			OrdersLimit:           50,
			TradeID:               48366685,
			OrderbookID:           5235273987,
			EnableCredit:          true,
			CreateTime:            types.Time(time.Unix(1545235200, 0)),
			FundingCapRatio:       0.75,
			Status:                "trading",
			LaunchTime:            types.Time(time.Unix(1545235200, 0)),
			MarketOrderSlipRatio:  0.02,
			MarketOrderSizeMax:    400000,
			FundingRateLimit:      0.00375,
			CrossLeverageDefault:  10,
		}
		assert.Equal(t, exp, result, "GetFuturesContract should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesContract should return a BTC contract")
}

func TestGetFuturesADLRiskStates(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesADLRiskStates(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesADLRiskStates must reject an empty settlement currency")

	result, err := e.GetFuturesADLRiskStates(t.Context(), currency.USDT)
	require.NoError(t, err, "GetFuturesADLRiskStates must not error")
	if mockTests {
		exp := &FuturesADLRiskStatesResponse{
			Settle: "usdt",
			States: map[string]*FuturesADLRiskState{
				"BTC_USDT": {
					State:        "normal",
					CalculatedAt: types.Time(time.UnixMilli(1786695185345)),
				},
				"HYPE_USDT": {
					State:        "adl_risk",
					CalculatedAt: types.Time(time.UnixMilli(1786695185346)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesADLRiskStates should decode every field")
		return
	}
	require.NotNil(t, result, "GetFuturesADLRiskStates must return a result")
	assert.NotEmpty(t, result.States, "GetFuturesADLRiskStates should return market states")
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderbook(t.Context(), currency.BTC, currency.EMPTYPAIR, "", 10, false)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesOrderbook must reject an empty contract")

	_, err = e.GetFuturesOrderbook(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), "", 10, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesOrderbook must reject an empty settlement currency")

	result, err := e.GetFuturesOrderbook(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), "", 10, false)
	require.NoError(t, err, "GetFuturesOrderbook must not error for a USDT contract")
	assert.NotNil(t, result, "GetFuturesOrderbook should return a USDT order book")

	result, err = e.GetFuturesOrderbook(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), "", 10, false)
	require.NoError(t, err, "GetFuturesOrderbook must not error for a BTC contract")
	if mockTests {
		exp := &OrderbookResponse{
			ID:           123456,
			ResponseTime: types.Time(time.UnixMilli(1623898993123)),
			UpdateTime:   types.Time(time.UnixMilli(1623898993121)),
			Bids: OrderbookLevels{
				{
					Amount: 150,
					Price:  1.17,
				},
				{
					Amount: 203,
					Price:  1.16,
				},
			},
			Asks: OrderbookLevels{
				{
					Amount: 100,
					Price:  1.52,
				},
				{
					Amount: 40,
					Price:  1.53,
				},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesOrderbook should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesOrderbook should return a BTC order book")
}

func TestGetFuturesTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTradingHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 0, 0, "", time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesTradingHistory must reject an empty settlement currency")

	_, err = e.GetFuturesTradingHistory(t.Context(), currency.BTC, currency.EMPTYPAIR, 0, 0, "", time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesTradingHistory must reject an empty contract")

	_, err = e.GetFuturesTradingHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, 0, "", time.Time{}, time.Time{})
	require.NoError(t, err, "GetFuturesTradingHistory must not error for a BTC contract")

	result, err := e.GetFuturesTradingHistory(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 0, 0, "", time.Time{}, time.Time{})
	require.NoError(t, err, "GetFuturesTradingHistory must not error for a USDT contract")
	require.NotEmpty(t, result, "GetFuturesTradingHistory must return trades")
	if mockTests {
		require.Len(t, result, 5, "GetFuturesTradingHistory must return every recorded trade")
		exp := &TradingHistoryItemResponse{
			ID:                     724267229,
			CreateTime:             types.Time(time.UnixMilli(1775423970950)),
			CreateTimeMilliseconds: types.Time(time.UnixMilli(1775423970950)),
			Contract:               "BTC_USDT",
			Size:                   -4,
			Price:                  67477.9,
		}
		assert.Equal(t, exp, result[0], "GetFuturesTradingHistory should decode every field")
	}
}

func TestGetFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesCandlesticks(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), time.Time{}, time.Time{}, 0, kline.OneWeek, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesCandlesticks must reject an empty settlement currency")

	_, err = e.GetFuturesCandlesticks(t.Context(), currency.BTC, currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, kline.OneWeek, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesCandlesticks must reject an empty contract")

	startTime, endTime := getTime()
	_, err = e.GetFuturesCandlesticks(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), startTime, endTime, 10, kline.OneWeek, "")
	require.ErrorIs(t, err, errLimitWithTimeRange, "GetFuturesCandlesticks must reject a limit combined with a time range")

	_, err = e.GetFuturesCandlesticks(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), time.Time{}, time.Time{}, 0, kline.OneWeek, "utc3")
	require.ErrorIs(t, err, errInvalidTimezone, "GetFuturesCandlesticks must reject an unsupported timezone")

	_, err = e.GetFuturesCandlesticks(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), time.Time{}, time.Time{}, 0, kline.OneWeek, "")
	require.NoError(t, err, "GetFuturesCandlesticks must not error for a BTC contract")

	result, err := e.GetFuturesCandlesticks(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), time.Time{}, time.Time{}, 0, kline.OneWeek, "")
	require.NoError(t, err, "GetFuturesCandlesticks must not error for a USDT contract")
	require.NotEmpty(t, result, "GetFuturesCandlesticks must return candles")
	if mockTests {
		require.Len(t, result, 5, "GetFuturesCandlesticks must return every recorded candle")
		exp := &FuturesCandlestickResponse{
			Timestamp:    types.Time(time.Unix(1715212800, 0)),
			Volume:       1035191038,
			ClosePrice:   66173.3,
			HighestPrice: 66398.8,
			LowestPrice:  60166.2,
			OpenPrice:    61187,
			QuoteVolume:  6443459137.427661,
		}
		assert.Equal(t, exp, result[0], "GetFuturesCandlesticks should decode every field")
	}
}

func TestPremiumIndexKline(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.PremiumIndexKline(t.Context(), currency.EMPTYCODE, contract, time.Time{}, time.Time{}, 0, kline.OneWeek)
	require.ErrorIs(t, err, errEmptyOrInvalidSettlementCurrency, "PremiumIndexKline must reject an empty settlement currency")

	_, err = e.PremiumIndexKline(t.Context(), currency.USDT, currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, kline.OneWeek)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PremiumIndexKline must reject an empty contract")

	_, err = e.PremiumIndexKline(t.Context(), currency.USDT, contract, time.Time{}, time.Time{}, 0, kline.FiveDay)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "PremiumIndexKline must reject an unsupported interval")

	startTime, endTime := getTime()
	_, err = e.PremiumIndexKline(t.Context(), currency.USDT, contract, startTime, endTime, 1, kline.OneWeek)
	require.ErrorIs(t, err, errLimitWithTimeRange, "PremiumIndexKline must reject a limit combined with a time range")

	result, err := e.PremiumIndexKline(t.Context(), currency.USDT, contract, time.Time{}, time.Time{}, 1, kline.OneWeek)
	require.NoError(t, err, "PremiumIndexKline must not error with a limit")
	if mockTests {
		exp := []*FuturesPremiumIndexKlineResponse{
			{
				UnixTimestamp: types.Time(time.Unix(1780876800, 0)),
				ClosePrice:    -0.0005988142412787689,
				HighestPrice:  -0.0005638765430773542,
				LowestPrice:   -0.0013547770658499465,
				OpenPrice:     0.0030031738532652376,
			},
			{
				UnixTimestamp: types.Time(time.Unix(1781481600, 0)),
				ClosePrice:    -0.0004602259855446214,
				HighestPrice:  -0.0007128946261382114,
				LowestPrice:   -0.0015641449473647465,
				OpenPrice:     0.00025202506014303526,
			},
		}
		assert.Equal(t, exp, result, "PremiumIndexKline should decode every field")
		return
	}
	assert.NotEmpty(t, result, "PremiumIndexKline should return data points")
}

func TestGetFutureFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetFutureFundingRates(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFutureFundingRates must reject an empty settlement currency")

	_, err = e.GetFutureFundingRates(t.Context(), currency.BTC, currency.EMPTYPAIR, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFutureFundingRates must reject an empty contract")

	startTime, endTime := getTime()
	_, err = e.GetFutureFundingRates(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFutureFundingRates must reject a start after the end")

	_, err = e.GetFutureFundingRates(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, time.Time{}, time.Time{})
	require.NoError(t, err, "GetFutureFundingRates must not error for a BTC contract")

	result, err := e.GetFutureFundingRates(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10, startTime, endTime)
	require.NoError(t, err, "GetFutureFundingRates must not error for a USDT contract within a time range")
	if mockTests {
		exp := []*FuturesFundingRateResponse{
			{
				Timestamp: types.Time(time.Unix(1744185600, 0)),
				Rate:      0.000157,
			},
			{
				Timestamp: types.Time(time.Unix(1744156800, 0)),
				Rate:      -0.000145,
			},
		}
		assert.Equal(t, exp, result, "GetFutureFundingRates should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetFutureFundingRates should return rates")
}

func TestGetBatchFuturesFundingRates(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.GetBatchFuturesFundingRates(t.Context(), currency.EMPTYCODE, currency.Pairs{contract})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetBatchFuturesFundingRates must reject an empty settlement currency")

	_, err = e.GetBatchFuturesFundingRates(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetBatchFuturesFundingRates must reject an empty contract list")

	_, err = e.GetBatchFuturesFundingRates(t.Context(), currency.USDT, currency.Pairs{currency.EMPTYPAIR})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetBatchFuturesFundingRates must reject an empty contract")

	ex := e
	if mockTests {
		ex = futuresLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "batch funding rates should be requested with POST")
			assert.Equal(t, "/api/v4/futures/usdt/funding_rates", r.URL.Path, "batch funding rates should be requested from the funding_rates endpoint")
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the request body should not error")
			assert.JSONEq(t, `{"contracts":["BTC_USDT"]}`, string(body), "batch funding rates should send the contract list in the body")
			_, err = w.Write([]byte(`[{"contract":"BTC_USDT","data":[{"t":1543968000,"r":"0.000157"},{"t":1544054400,"r":"-0.000145"}]}]`))
			assert.NoError(t, err, "writing the funding rates should not error")
		})
	}
	result, err := ex.GetBatchFuturesFundingRates(t.Context(), currency.USDT, currency.Pairs{contract})
	require.NoError(t, err, "GetBatchFuturesFundingRates must not error")
	if mockTests {
		exp := []*BatchFundingRatesResponse{
			{
				Contract: "BTC_USDT",
				Data: []*FuturesFundingRateResponse{
					{
						Timestamp: types.Time(time.Unix(1543968000, 0)),
						Rate:      0.000157,
					},
					{
						Timestamp: types.Time(time.Unix(1544054400, 0)),
						Rate:      -0.000145,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetBatchFuturesFundingRates should decode every field")
		return
	}
	require.Len(t, result, 1, "GetBatchFuturesFundingRates must return the requested contract")
	assert.NotEmpty(t, result[0].Data, "GetBatchFuturesFundingRates should return funding rates")
}

func TestGetFuturesInsuranceBalanceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesInsuranceBalanceHistory(t.Context(), currency.EMPTYCODE, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesInsuranceBalanceHistory must reject an empty settlement currency")

	result, err := e.GetFuturesInsuranceBalanceHistory(t.Context(), currency.USDT, 10)
	require.NoError(t, err, "GetFuturesInsuranceBalanceHistory must not error")
	require.NotEmpty(t, result, "GetFuturesInsuranceBalanceHistory must return balances")
	if mockTests {
		require.Len(t, result, 5, "GetFuturesInsuranceBalanceHistory must return every recorded balance")
		exp := &InsuranceBalanceResponse{
			Timestamp: types.Time(time.Unix(1775347200, 0)),
			Balance:   132672449.967292,
		}
		assert.Equal(t, exp, result[0], "GetFuturesInsuranceBalanceHistory should decode every field")
	}
}

func TestGetFutureStats(t *testing.T) {
	t.Parallel()
	_, err := e.GetFutureStats(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), time.Time{}, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFutureStats must reject an empty settlement currency")

	_, err = e.GetFutureStats(t.Context(), currency.BTC, currency.EMPTYPAIR, time.Time{}, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFutureStats must reject an empty contract")

	_, err = e.GetFutureStats(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), time.Time{}, kline.FiveDay, 0)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetFutureStats must reject an unsupported interval")

	_, err = e.GetFutureStats(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), time.Time{}, 0, 0)
	require.NoError(t, err, "GetFutureStats must not error for a BTC contract")

	result, err := e.GetFutureStats(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), time.UnixMilli(1780846985008), kline.OneMin, 100)
	require.NoError(t, err, "GetFutureStats must not error for a USDT contract")
	if mockTests {
		require.Len(t, result, 5, "GetFutureStats must return every recorded stat")
		exp := &ContractStatResponse{
			Time:                types.Time(time.Unix(1780847040, 0)),
			LongShortTaker:      0.4883055642879598,
			LongShortAccount:    1.5497060395510422,
			OpenInterest:        643307912,
			MarkPrice:           62048.59,
			TopLongShortSize:    1.060266307572198,
			OpenInterestUSD:     3991634887.544408,
			TopLongShortAccount: 0.5662650602409639,
			TopLongSize:         36769850,
			TopShortSize:        34679825,
			TopLongAccount:      47,
			TopShortAccount:     83,
			LongTakerSize:       155330,
			ShortTakerSize:      318100,
			LongUsers:           17397,
			ShortUsers:          11226,
		}
		assert.Equal(t, exp, result[0], "GetFutureStats should decode every field")
	}
}

func TestGetIndexConstituent(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexConstituent(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures).String())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetIndexConstituent must reject an empty settlement currency")

	_, err = e.GetIndexConstituent(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetIndexConstituent must reject an empty index")

	result, err := e.GetIndexConstituent(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures).String())
	require.NoError(t, err, "GetIndexConstituent must not error")
	if mockTests {
		exp := &IndexConstituentResponse{
			Index: "BTC_USDT",
			Constituents: []*IndexConstituentSources{
				{
					Exchange: "Binance",
					Symbols: []string{
						"BTC_USDT",
					},
				},
				{
					Exchange: "Bitget",
					Symbols: []string{
						"BTC_USDT",
					},
				},
				{
					Exchange: "Bybit",
					Symbols: []string{
						"BTC_USDT",
					},
				},
				{
					Exchange: "Gate",
					Symbols: []string{
						"BTC_USDT",
					},
				},
				{
					Exchange: "MEXC",
					Symbols: []string{
						"BTC_USDT",
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetIndexConstituent should decode every field")
		return
	}
	require.NotNil(t, result, "GetIndexConstituent must return a result")
	assert.NotEmpty(t, result.Constituents, "GetIndexConstituent should return constituents")
}

func TestGetLiquidationHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetLiquidationHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures), time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetLiquidationHistory must reject an empty settlement currency")

	startTime, endTime := getTimeWithInterval(kline.OneHour)
	if !mockTests {
		endTime = startTime.Add(time.Hour) // live ranges span two days, beyond this endpoint's one hour limit
	}
	_, err = e.GetLiquidationHistory(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), endTime, startTime, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLiquidationHistory must reject a start after the end")

	_, err = e.GetLiquidationHistory(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), startTime, endTime.Add(time.Second), 0)
	require.ErrorIs(t, err, errLiquidationTimeRangeTooLarge, "GetLiquidationHistory must reject a time range beyond an hour")

	result, err := e.GetLiquidationHistory(t.Context(), currency.USDT, currency.EMPTYPAIR, startTime, endTime, 10)
	require.NoError(t, err, "GetLiquidationHistory must not error without a contract")
	if mockTests {
		exp := []*LiquidationHistoryResponse{
			{
				Time:       types.Time(time.Unix(1744104000, 0)),
				Contract:   "BTC_USDT",
				Size:       600,
				OrderSize:  -600,
				OrderPrice: 3405,
				FillPrice:  3424,
			},
		}
		assert.Equal(t, exp, result, "GetLiquidationHistory should decode every field")
	}
}

func TestQueryFuturesAccount(t *testing.T) {
	t.Parallel()
	_, err := e.QueryFuturesAccount(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "QueryFuturesAccount must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.QueryFuturesAccount(t.Context(), currency.USDT)
	require.NoError(t, err, "QueryFuturesAccount must not error")
	if mockTests {
		exp := futuresFixtureAccount(currency.USDT)
		assert.Equal(t, exp, result, "QueryFuturesAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "QueryFuturesAccount should return an account")
}

func TestGetFuturesAccountBooks(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesAccountBooks(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{}, "dnw")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesAccountBooks must reject an empty settlement currency")

	startTime, endTime := getTime()
	_, err = e.GetFuturesAccountBooks(t.Context(), currency.USDT, currency.EMPTYPAIR, 0, 0, endTime, startTime, "dnw")
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesAccountBooks must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesAccountBooks(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10, 1, startTime, endTime, "fee")
	require.NoError(t, err, "GetFuturesAccountBooks must not error")
	if mockTests {
		exp := []*AccountBookItemResponse{
			{
				Time:     types.Time(time.UnixMicro(1682294400123456)),
				Change:   0.000010152188,
				Balance:  4.59316525194,
				Text:     "BTC_USDT:6086261",
				Type:     "fee",
				Contract: "BTC_USDT",
				TradeID:  "1",
				ID:       "2",
			},
		}
		assert.Equal(t, exp, result, "GetFuturesAccountBooks should decode every field")
	}
}

func TestGetAllFuturesPositionsOfUsers(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllFuturesPositionsOfUsers(t.Context(), currency.EMPTYCODE, true, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAllFuturesPositionsOfUsers must reject an empty settlement currency")

	_, err = e.GetAllFuturesPositionsOfUsers(t.Context(), currency.USDT, true, 101, 0)
	require.ErrorIs(t, err, errInvalidLimit, "GetAllFuturesPositionsOfUsers must reject a limit above 100")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllFuturesPositionsOfUsers(t.Context(), currency.USDT, true, 10, 0)
	require.NoError(t, err, "GetAllFuturesPositionsOfUsers must not error")
	if mockTests {
		exp := []*PositionResponse{futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "single", -9440)}
		assert.Equal(t, exp, result, "GetAllFuturesPositionsOfUsers should decode every field")
	}
}

func TestGetFuturesHistoricalPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesHistoricalPositions(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures), 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesHistoricalPositions must reject an empty settlement currency")

	_, err = e.GetFuturesHistoricalPositions(t.Context(), currency.USDT, currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesHistoricalPositions must reject an empty contract")

	startTime, endTime := getTime()
	_, err = e.GetFuturesHistoricalPositions(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesHistoricalPositions must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesHistoricalPositions(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10, 0, startTime, endTime)
	require.NoError(t, err, "GetFuturesHistoricalPositions must not error")
	if mockTests {
		exp := []*FuturesHistoricalPositionResponse{
			{
				Contract:                currency.NewPairWithDelimiter("BTC", "USDT", "_"),
				Size:                    -9440,
				RiskLimit:               100,
				LeverageMax:             100,
				MaintenanceRate:         0.005,
				Margin:                  4.43,
				LiquidationPrice:        99999999,
				RealisedPNL:             0.04,
				HistoryPNL:              1.5,
				LastClosePNL:            0.25,
				RealisedPNLPoint:        0.1,
				RealisedPNLHistoryPoint: 0.2,
				Mode:                    "single",
				CrossLeverageLimit:      10,
				EntryPrice:              3779.55,
				Time:                    types.Time(time.Unix(1744110000, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetFuturesHistoricalPositions should decode every field")
	}
}

func TestGetSinglePosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetSinglePosition(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSinglePosition must reject an empty settlement currency")

	_, err = e.GetSinglePosition(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSinglePosition must reject an empty contract")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSinglePosition(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures))
	require.NoError(t, err, "GetSinglePosition must not error")
	if mockTests {
		exp := futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "single", -9440)
		assert.Equal(t, exp, result, "GetSinglePosition should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSinglePosition should return a position")
}

func TestGetFuturesLeverage(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.GetFuturesLeverage(t.Context(), currency.EMPTYCODE, contract, "isolated", "dual_long")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesLeverage must reject an empty settlement currency")

	_, err = e.GetFuturesLeverage(t.Context(), currency.USDT, currency.EMPTYPAIR, "isolated", "dual_long")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesLeverage must reject an empty contract")

	_, err = e.GetFuturesLeverage(t.Context(), currency.USDT, contract, "portfolio", "dual_long")
	require.ErrorIs(t, err, errInvalidPositionMarginMode, "GetFuturesLeverage must reject an unknown position margin mode")

	_, err = e.GetFuturesLeverage(t.Context(), currency.USDT, contract, "isolated", "long")
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetFuturesLeverage must reject an unknown dual side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesLeverage(t.Context(), currency.USDT, contract, "isolated", "dual_long")
	require.NoError(t, err, "GetFuturesLeverage must not error")
	if mockTests {
		exp := &FuturesLeverageResponse{
			Leverage: 10,
		}
		assert.Equal(t, exp, result, "GetFuturesLeverage should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesLeverage should return a leverage")
}

func TestUpdateFuturesPositionMargin(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateFuturesPositionMargin(t.Context(), currency.EMPTYCODE, 0.01, getPair(t, asset.CoinMarginedFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdateFuturesPositionMargin must reject an empty settlement currency")

	_, err = e.UpdateFuturesPositionMargin(t.Context(), currency.BTC, 0.01, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateFuturesPositionMargin must reject an empty contract")

	_, err = e.UpdateFuturesPositionMargin(t.Context(), currency.BTC, 0, getPair(t, asset.CoinMarginedFutures))
	require.ErrorIs(t, err, errInvalidMarginChange, "UpdateFuturesPositionMargin must reject a zero change")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.UpdateFuturesPositionMargin(t.Context(), currency.BTC, 0.01, getPair(t, asset.CoinMarginedFutures))
	require.NoError(t, err, "UpdateFuturesPositionMargin must not error adding margin")

	result, err := e.UpdateFuturesPositionMargin(t.Context(), currency.USDT, -0.01, getPair(t, asset.USDTMarginedFutures))
	require.NoError(t, err, "UpdateFuturesPositionMargin must not error removing margin")
	if mockTests {
		exp := futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "single", -9440)
		assert.Equal(t, exp, result, "UpdateFuturesPositionMargin should decode every field")
		return
	}
	assert.NotNil(t, result, "UpdateFuturesPositionMargin should return the position")
}

func TestUpdateFuturesPositionLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateFuturesPositionLeverage(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 1, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdateFuturesPositionLeverage must reject an empty settlement currency")

	_, err = e.UpdateFuturesPositionLeverage(t.Context(), currency.BTC, currency.EMPTYPAIR, 1, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateFuturesPositionLeverage must reject an empty contract")

	_, err = e.UpdateFuturesPositionLeverage(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), -1, 0, 0)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported, "UpdateFuturesPositionLeverage must reject a negative leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UpdateFuturesPositionLeverage(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 1, 0, 0)
	require.NoError(t, err, "UpdateFuturesPositionLeverage must not error")
	if mockTests {
		exp := futuresFixturePosition(getPair(t, asset.CoinMarginedFutures), "single", -9440)
		assert.Equal(t, exp, result, "UpdateFuturesPositionLeverage should decode every field")
		return
	}
	assert.NotNil(t, result, "UpdateFuturesPositionLeverage should return the position")
}

func TestPositionReplyUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := &PositionResponse{Contract: currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter), Size: 5, Mode: "single"}
	for _, payload := range []string{
		`{"contract":"BTC_USDT","size":"5","mode":"single"}`,
		`[{"contract":"BTC_USDT","size":"5","mode":"single"}]`,
	} {
		var reply positionReply
		require.NoErrorf(t, json.Unmarshal([]byte(payload), &reply), "Unmarshal must not error for %s", payload)
		assert.Equalf(t, exp, reply.Position, "positionReply should decode the position from %s", payload)
	}
	for _, payload := range []string{`[]`, `[{"contract":"BTC_USDT"},{"contract":"ETH_USDT"}]`} {
		var reply positionReply
		assert.ErrorIsf(t, json.Unmarshal([]byte(payload), &reply), common.ErrInvalidResponse, "positionReply should reject %s, which does not hold exactly one position", payload)
	}
}

func TestSetFuturesLeverage(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.SetFuturesLeverage(t.Context(), currency.EMPTYCODE, contract, 10, "cross", "dual_long")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SetFuturesLeverage must reject an empty settlement currency")

	_, err = e.SetFuturesLeverage(t.Context(), currency.USDT, currency.EMPTYPAIR, 10, "cross", "dual_long")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SetFuturesLeverage must reject an empty contract")

	_, err = e.SetFuturesLeverage(t.Context(), currency.USDT, contract, 0, "cross", "dual_long")
	require.ErrorIs(t, err, errInvalidLeverage, "SetFuturesLeverage must reject a zero leverage")

	_, err = e.SetFuturesLeverage(t.Context(), currency.USDT, contract, 10, "CROSS", "dual_long")
	require.ErrorIs(t, err, errInvalidPositionMarginMode, "SetFuturesLeverage must reject an unknown margin mode")

	_, err = e.SetFuturesLeverage(t.Context(), currency.USDT, contract, 10, "cross", "short")
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "SetFuturesLeverage must reject an unknown dual side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetFuturesLeverage(t.Context(), currency.USDT, contract, 10, "cross", "dual_long")
	require.NoError(t, err, "SetFuturesLeverage must not error")
	if mockTests {
		exp := futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_long", -9440)
		assert.Equal(t, exp, result, "SetFuturesLeverage should decode every field")
		return
	}
	assert.NotNil(t, result, "SetFuturesLeverage should return the position")
}

func TestSwitchFuturesPositionMarginMode(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.SwitchFuturesPositionMarginMode(t.Context(), currency.EMPTYCODE, &FuturesPositionMarginModeRequest{Mode: "ISOLATED", Contract: contract})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SwitchFuturesPositionMarginMode must reject an empty settlement currency")

	_, err = e.SwitchFuturesPositionMarginMode(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SwitchFuturesPositionMarginMode must reject a nil request")

	_, err = e.SwitchFuturesPositionMarginMode(t.Context(), currency.USDT, &FuturesPositionMarginModeRequest{Mode: "isolated", Contract: contract})
	require.ErrorIs(t, err, errInvalidPositionMarginMode, "SwitchFuturesPositionMarginMode must reject a margin mode in the wrong case")

	_, err = e.SwitchFuturesPositionMarginMode(t.Context(), currency.USDT, &FuturesPositionMarginModeRequest{Mode: "ISOLATED"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SwitchFuturesPositionMarginMode must reject an empty contract")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SwitchFuturesPositionMarginMode(t.Context(), currency.USDT, &FuturesPositionMarginModeRequest{Mode: "ISOLATED", Contract: contract})
	require.NoError(t, err, "SwitchFuturesPositionMarginMode must not error")
	if mockTests {
		exp := futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "single", -9440)
		assert.Equal(t, exp, result, "SwitchFuturesPositionMarginMode should decode every field")
		return
	}
	assert.NotNil(t, result, "SwitchFuturesPositionMarginMode should return the position")
}

func TestSwitchFuturesDualModePositionMarginMode(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.SwitchFuturesDualModePositionMarginMode(t.Context(), currency.EMPTYCODE, &FuturesPositionMarginModeRequest{Mode: "CROSS", Contract: contract})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SwitchFuturesDualModePositionMarginMode must reject an empty settlement currency")

	_, err = e.SwitchFuturesDualModePositionMarginMode(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SwitchFuturesDualModePositionMarginMode must reject a nil request")

	_, err = e.SwitchFuturesDualModePositionMarginMode(t.Context(), currency.USDT, &FuturesPositionMarginModeRequest{Mode: "PORTFOLIO", Contract: contract})
	require.ErrorIs(t, err, errInvalidPositionMarginMode, "SwitchFuturesDualModePositionMarginMode must reject an unknown margin mode")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SwitchFuturesDualModePositionMarginMode(t.Context(), currency.USDT, &FuturesPositionMarginModeRequest{Mode: "CROSS", Contract: contract})
	require.NoError(t, err, "SwitchFuturesDualModePositionMarginMode must not error")
	if mockTests {
		exp := []*PositionResponse{futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_long", 10), futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_short", -10)}
		assert.Equal(t, exp, result, "SwitchFuturesDualModePositionMarginMode should decode every field")
	}
}

func TestFuturesPositionMarginModeRequestValidate(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, (*FuturesPositionMarginModeRequest)(nil).validate(), common.ErrNilPointer, "validate should reject a nil request")
	assert.ErrorIs(t, (&FuturesPositionMarginModeRequest{Mode: "cross", Contract: currency.NewBTCUSDT()}).validate(), errInvalidPositionMarginMode, "validate should reject a margin mode in the wrong case")
	assert.ErrorIs(t, (&FuturesPositionMarginModeRequest{Mode: "CROSS"}).validate(), currency.ErrCurrencyPairEmpty, "validate should reject an empty contract")
	for _, mode := range []string{"ISOLATED", "CROSS"} {
		assert.NoErrorf(t, (&FuturesPositionMarginModeRequest{Mode: mode, Contract: currency.NewBTCUSDT()}).validate(), "validate should accept the %s margin mode", mode)
	}
}

func TestEnableOrDisableDualMode(t *testing.T) {
	t.Parallel()
	_, err := e.EnableOrDisableDualMode(t.Context(), currency.EMPTYCODE, true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "EnableOrDisableDualMode must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.EnableOrDisableDualMode(t.Context(), currency.BTC, true)
	require.NoError(t, err, "EnableOrDisableDualMode must not error")
	if mockTests {
		exp := futuresFixtureAccount(currency.BTC)
		assert.Equal(t, exp, result, "EnableOrDisableDualMode should decode every field")
		return
	}
	assert.NotNil(t, result, "EnableOrDisableDualMode should return the account")
}

func TestSetFuturesPositionMode(t *testing.T) {
	t.Parallel()
	_, err := e.SetFuturesPositionMode(t.Context(), currency.EMPTYCODE, "dual_plus")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SetFuturesPositionMode must reject an empty settlement currency")

	_, err = e.SetFuturesPositionMode(t.Context(), currency.USDT, "hedge")
	require.ErrorIs(t, err, errInvalidPositionMode, "SetFuturesPositionMode must reject an unknown position mode")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetFuturesPositionMode(t.Context(), currency.USDT, "dual_plus")
	require.NoError(t, err, "SetFuturesPositionMode must not error")
	if mockTests {
		exp := futuresFixtureAccount(currency.USDT)
		assert.Equal(t, exp, result, "SetFuturesPositionMode should decode every field")
		return
	}
	assert.NotNil(t, result, "SetFuturesPositionMode should return the account")
}

func TestGetPositionDetailInDualMode(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositionDetailInDualMode(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetPositionDetailInDualMode must reject an empty settlement currency")

	_, err = e.GetPositionDetailInDualMode(t.Context(), currency.BTC, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPositionDetailInDualMode must reject an empty contract")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetPositionDetailInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures))
	require.NoError(t, err, "GetPositionDetailInDualMode must not error for a BTC contract")

	result, err := e.GetPositionDetailInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures))
	require.NoError(t, err, "GetPositionDetailInDualMode must not error for a USDT contract")
	if mockTests {
		exp := []*PositionResponse{futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_long", 10), futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_short", -10)}
		assert.Equal(t, exp, result, "GetPositionDetailInDualMode should decode every field")
	}
}

func TestUpdatePositionMarginInDualMode(t *testing.T) {
	t.Parallel()
	_, err := e.UpdatePositionMarginInDualMode(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 0.001, "dual_long")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdatePositionMarginInDualMode must reject an empty settlement currency")

	_, err = e.UpdatePositionMarginInDualMode(t.Context(), currency.BTC, currency.EMPTYPAIR, 0.001, "dual_long")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdatePositionMarginInDualMode must reject an empty contract")

	_, err = e.UpdatePositionMarginInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, "dual_long")
	require.ErrorIs(t, err, errInvalidMarginChange, "UpdatePositionMarginInDualMode must reject a zero change")

	_, err = e.UpdatePositionMarginInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0.001, "")
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "UpdatePositionMarginInDualMode must reject an empty dual side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.UpdatePositionMarginInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0.001, "dual_long")
	require.NoError(t, err, "UpdatePositionMarginInDualMode must not error for a BTC contract")

	result, err := e.UpdatePositionMarginInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 0.001, "dual_long")
	require.NoError(t, err, "UpdatePositionMarginInDualMode must not error for a USDT contract")
	if mockTests {
		exp := []*PositionResponse{futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_long", 10)}
		assert.Equal(t, exp, result, "UpdatePositionMarginInDualMode should decode every field")
	}
}

func TestUpdatePositionLeverageInDualMode(t *testing.T) {
	t.Parallel()
	_, err := e.UpdatePositionLeverageInDualMode(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures), 1, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdatePositionLeverageInDualMode must reject an empty settlement currency")

	_, err = e.UpdatePositionLeverageInDualMode(t.Context(), currency.USDT, currency.EMPTYPAIR, 1, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdatePositionLeverageInDualMode must reject an empty contract")

	_, err = e.UpdatePositionLeverageInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), -1, 0.001)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported, "UpdatePositionLeverageInDualMode must reject a negative leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.UpdatePositionLeverageInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 1, 0.001)
	require.NoError(t, err, "UpdatePositionLeverageInDualMode must not error for a BTC contract")

	result, err := e.UpdatePositionLeverageInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10, 0.001)
	require.NoError(t, err, "UpdatePositionLeverageInDualMode must not error for a USDT contract")
	if mockTests {
		exp := []*PositionResponse{futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_long", 10), futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_short", -10)}
		assert.Equal(t, exp, result, "UpdatePositionLeverageInDualMode should decode every field")
	}
}

func TestUpdatePositionRiskLimitInDualMode(t *testing.T) {
	t.Parallel()
	_, err := e.UpdatePositionRiskLimitInDualMode(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdatePositionRiskLimitInDualMode must reject an empty settlement currency")

	_, err = e.UpdatePositionRiskLimitInDualMode(t.Context(), currency.BTC, currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdatePositionRiskLimitInDualMode must reject an empty contract")

	_, err = e.UpdatePositionRiskLimitInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0)
	require.ErrorIs(t, err, errInvalidRiskLimit, "UpdatePositionRiskLimitInDualMode must reject a zero risk limit")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.UpdatePositionRiskLimitInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 20000)
	require.NoError(t, err, "UpdatePositionRiskLimitInDualMode must not error for a BTC contract")

	result, err := e.UpdatePositionRiskLimitInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 20000)
	require.NoError(t, err, "UpdatePositionRiskLimitInDualMode must not error for a USDT contract")
	if mockTests {
		exp := []*PositionResponse{futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_long", 10), futuresFixturePosition(getPair(t, asset.USDTMarginedFutures), "dual_short", -10)}
		assert.Equal(t, exp, result, "UpdatePositionRiskLimitInDualMode should decode every field")
	}
}

func TestPlaceFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceFuturesOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceFuturesOrder must reject a nil order")

	arg := &FuturesOrderCreateRequest{}
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceFuturesOrder must reject an empty contract")

	arg.Contract = getPair(t, asset.USDTMarginedFutures)
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidOrderSize, "PlaceFuturesOrder must reject an empty size")

	arg.Size = 1
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "PlaceFuturesOrder must reject a market order without an ioc or fok time in force")

	arg.TimeInForce = order.GoodTillCancel.Lower()
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "PlaceFuturesOrder must reject a market order with a gtc time in force")

	arg.Price = 123
	arg.Text = "client-order-id"
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidTextPrefix, "PlaceFuturesOrder must reject text without the t- prefix")

	arg.Text = "t-client-order-id"
	arg.AutoSize = "long"
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAutoSize, "PlaceFuturesOrder must reject an unknown auto size")

	arg.AutoSize = ""
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceFuturesOrder must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceFuturesOrder(t.Context(), &FuturesOrderCreateRequest{
		Contract:    getPair(t, asset.CoinMarginedFutures),
		Size:        6024,
		Price:       3765,
		TimeInForce: "gtc",
		Text:        "t-my-custom-id",
		Settle:      currency.BTC,
	})
	require.NoError(t, err, "PlaceFuturesOrder must not error")
	if mockTests {
		exp := futuresFixtureOrder(getPair(t, asset.CoinMarginedFutures))
		assert.Equal(t, exp, result, "PlaceFuturesOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "PlaceFuturesOrder should return the order")
}

func TestGetFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrders(t.Context(), getPair(t, asset.CoinMarginedFutures), statusOpen, "", currency.EMPTYCODE, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesOrders must reject an empty settlement currency")

	_, err = e.GetFuturesOrders(t.Context(), getPair(t, asset.CoinMarginedFutures), "", "", currency.BTC, 0, 0)
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetFuturesOrders must reject an empty status")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesOrders(t.Context(), getPair(t, asset.CoinMarginedFutures), statusOpen, "", currency.BTC, 0, 0)
	require.NoError(t, err, "GetFuturesOrders must not error")
	if mockTests {
		openOrder := futuresFixtureOrder(getPair(t, asset.CoinMarginedFutures))
		openOrder.Status = statusOpen
		exp := []*FuturesOrderResponse{openOrder}
		assert.Equal(t, exp, result, "GetFuturesOrders should decode every field")
	}
}

func TestCancelMultipleFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.CancelMultipleFuturesOpenOrders(t.Context(), currency.EMPTYCODE, &CancelFuturesOrdersRequest{Contract: contract, Side: "ask"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelMultipleFuturesOpenOrders must reject an empty settlement currency")

	_, err = e.CancelMultipleFuturesOpenOrders(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelMultipleFuturesOpenOrders must reject a nil request")

	_, err = e.CancelMultipleFuturesOpenOrders(t.Context(), currency.USDT, &CancelFuturesOrdersRequest{Contract: contract, Side: "sell"})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CancelMultipleFuturesOpenOrders must reject a side other than bid or ask")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMultipleFuturesOpenOrders(t.Context(), currency.USDT, &CancelFuturesOrdersRequest{Contract: contract, Side: "ask"})
	require.NoError(t, err, "CancelMultipleFuturesOpenOrders must not error")
	if mockTests {
		exp := []*FuturesOrderResponse{futuresFixtureOrder(getPair(t, asset.USDTMarginedFutures))}
		assert.Equal(t, exp, result, "CancelMultipleFuturesOpenOrders should decode every field")
	}
}

func TestGetFuturesOrdersByTimeRange(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrdersByTimeRange(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures), 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesOrdersByTimeRange must reject an empty settlement currency")

	startTime, endTime := getTime()
	_, err = e.GetFuturesOrdersByTimeRange(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesOrdersByTimeRange must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesOrdersByTimeRange(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10, 0, startTime, endTime)
	require.NoError(t, err, "GetFuturesOrdersByTimeRange must not error")
	if mockTests {
		exp := []*FuturesOrderResponse{futuresFixtureOrder(getPair(t, asset.USDTMarginedFutures))}
		assert.Equal(t, exp, result, "GetFuturesOrdersByTimeRange should decode every field")
	}
}

func TestPlaceBatchFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceBatchFuturesOrders(t.Context(), currency.EMPTYCODE, []FuturesOrderCreateRequest{{}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceBatchFuturesOrders must reject an empty settlement currency")

	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, errOrdersEmpty, "PlaceBatchFuturesOrders must reject an empty batch")

	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, make([]FuturesOrderCreateRequest, 11))
	require.ErrorIs(t, err, errTooManyOrderRequest, "PlaceBatchFuturesOrders must reject more than 10 orders")

	arg := FuturesOrderCreateRequest{}
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []FuturesOrderCreateRequest{arg})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceBatchFuturesOrders must reject an empty contract")

	arg.Contract = getPair(t, asset.USDTMarginedFutures)
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []FuturesOrderCreateRequest{arg})
	require.ErrorIs(t, err, errInvalidOrderSize, "PlaceBatchFuturesOrders must reject an empty size")

	arg.Size = 1
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []FuturesOrderCreateRequest{arg})
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "PlaceBatchFuturesOrders must reject a market order without an ioc or fok time in force")

	arg.TimeInForce = order.GoodTillCancel.Lower()
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []FuturesOrderCreateRequest{arg})
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "PlaceBatchFuturesOrders must reject a market order with a gtc time in force")

	arg.Price = 123
	arg.Text = "client-order-id"
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []FuturesOrderCreateRequest{arg})
	require.ErrorIs(t, err, errInvalidTextPrefix, "PlaceBatchFuturesOrders must reject text without the t- prefix")

	arg.Text = "t-client-order-id"
	arg.AutoSize = "long"
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []FuturesOrderCreateRequest{arg})
	require.ErrorIs(t, err, errInvalidAutoSize, "PlaceBatchFuturesOrders must reject an unknown auto size")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceBatchFuturesOrders(t.Context(), currency.BTC, []FuturesOrderCreateRequest{
		{
			Contract:    getPair(t, asset.CoinMarginedFutures),
			Size:        6024,
			Price:       3765,
			TimeInForce: "gtc",
			Text:        "t-my-custom-id",
			Settle:      currency.BTC,
		},
		{
			Contract:    getPair(t, asset.CoinMarginedFutures),
			Size:        232,
			Price:       376225,
			TimeInForce: "gtc",
			Text:        "t-my-custom-id",
			Settle:      currency.BTC,
		},
	})
	require.NoError(t, err, "PlaceBatchFuturesOrders must not error")
	if mockTests {
		exp := []*BatchFuturesOrderResponse{
			{Succeeded: true, FuturesOrderResponse: *futuresFixtureOrder(getPair(t, asset.CoinMarginedFutures))},
			{Label: "INSUFFICIENT_AVAILABLE", Detail: "available balance not enough"},
		}
		assert.Equal(t, exp, result, "PlaceBatchFuturesOrders should decode every field")
		return
	}
	assert.Len(t, result, 2, "PlaceBatchFuturesOrders should report each order")
}

func TestGetSingleFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleFuturesOrder(t.Context(), currency.EMPTYCODE, "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSingleFuturesOrder must reject an empty settlement currency")

	_, err = e.GetSingleFuturesOrder(t.Context(), currency.BTC, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSingleFuturesOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSingleFuturesOrder(t.Context(), currency.BTC, "12345")
	require.NoError(t, err, "GetSingleFuturesOrder must not error")
	if mockTests {
		exp := futuresFixtureOrder(getPair(t, asset.CoinMarginedFutures))
		assert.Equal(t, exp, result, "GetSingleFuturesOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleFuturesOrder should return the order")
}

func TestCancelSingleFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSingleFuturesOrder(t.Context(), currency.EMPTYCODE, "12345", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelSingleFuturesOrder must reject an empty settlement currency")

	_, err = e.CancelSingleFuturesOrder(t.Context(), currency.BTC, "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSingleFuturesOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelSingleFuturesOrder(t.Context(), currency.BTC, "12345", "ACK")
	require.NoError(t, err, "CancelSingleFuturesOrder must not error")
	if mockTests {
		exp := futuresFixtureOrder(getPair(t, asset.CoinMarginedFutures))
		assert.Equal(t, exp, result, "CancelSingleFuturesOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelSingleFuturesOrder should return the order")
}

func TestAmendFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AmendFuturesOrder(t.Context(), currency.EMPTYCODE, "1234", &AmendFuturesOrderRequest{Price: 12345.990})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "AmendFuturesOrder must reject an empty settlement currency")

	_, err = e.AmendFuturesOrder(t.Context(), currency.BTC, "", &AmendFuturesOrderRequest{Price: 12345.990})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendFuturesOrder must reject an empty order ID")

	_, err = e.AmendFuturesOrder(t.Context(), currency.BTC, "1234", nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendFuturesOrder must reject a nil amendment")

	_, err = e.AmendFuturesOrder(t.Context(), currency.BTC, "1234", &AmendFuturesOrderRequest{AmendText: "no change"})
	require.ErrorIs(t, err, errNoOrderAmendment, "AmendFuturesOrder must reject an amendment without a size or price")

	_, err = e.AmendFuturesOrder(t.Context(), currency.BTC, "1234", &AmendFuturesOrderRequest{Price: -1})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "AmendFuturesOrder must reject a negative price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AmendFuturesOrder(t.Context(), currency.BTC, "1234", &AmendFuturesOrderRequest{Price: 12345.990})
	require.NoError(t, err, "AmendFuturesOrder must not error")
	if mockTests {
		exp := futuresFixtureOrder(getPair(t, asset.CoinMarginedFutures))
		exp.Price = 12345.99
		assert.Equal(t, exp, result, "AmendFuturesOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "AmendFuturesOrder should return the order")
}

func TestGetMyFuturesTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMyFuturesTradingHistory(t.Context(), currency.EMPTYCODE, "", "", getPair(t, asset.CoinMarginedFutures), 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMyFuturesTradingHistory must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMyFuturesTradingHistory(t.Context(), currency.BTC, "", "", getPair(t, asset.CoinMarginedFutures), 0, 0)
	require.NoError(t, err, "GetMyFuturesTradingHistory must not error")
	if mockTests {
		exp := []*TradingHistoryItemResponse{
			{
				ID:         121234231,
				CreateTime: types.Time(time.UnixMilli(1514764800123)),
				Contract:   "BTC_USD",
				Text:       "t-123456",
				Size:       100,
				Price:      100.123,
				OrderID:    "21893289839",
				CloseSize:  -50,
				Fee:        0.01,
				PointFee:   0.02,
				Role:       "taker",
				TradeValue: 28601.83,
			},
		}
		assert.Equal(t, exp, result, "GetMyFuturesTradingHistory should decode every field")
	}
}

func TestGetMyFuturesTradingHistoryByTimeRange(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.GetMyFuturesTradingHistoryByTimeRange(t.Context(), currency.EMPTYCODE, contract, "", 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMyFuturesTradingHistoryByTimeRange must reject an empty settlement currency")

	_, err = e.GetMyFuturesTradingHistoryByTimeRange(t.Context(), currency.USDT, contract, "liquidity", 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidTradeRole, "GetMyFuturesTradingHistoryByTimeRange must reject a role other than maker or taker")

	startTime, endTime := getTime()
	_, err = e.GetMyFuturesTradingHistoryByTimeRange(t.Context(), currency.USDT, contract, "", 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMyFuturesTradingHistoryByTimeRange must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMyFuturesTradingHistoryByTimeRange(t.Context(), currency.USDT, contract, "taker", 10, 0, startTime, endTime)
	require.NoError(t, err, "GetMyFuturesTradingHistoryByTimeRange must not error")
	if mockTests {
		exp := []*FuturesTradeByTimeRangeResponse{
			{
				TradeID:    "121234231",
				CreateTime: types.Time(time.UnixMilli(1744110000123)),
				Contract:   "BTC_USDT",
				OrderID:    "21893289839",
				Size:       100,
				CloseSize:  -50,
				Price:      100.123,
				Role:       "taker",
				Text:       "t-123456",
				Fee:        0.01,
				PointFee:   0.02,
			},
		}
		assert.Equal(t, exp, result, "GetMyFuturesTradingHistoryByTimeRange should decode every field")
	}
}

func TestGetFuturesPositionCloseHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionCloseHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 0, 0, time.Time{}, time.Time{}, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesPositionCloseHistory must reject an empty settlement currency")

	_, err = e.GetFuturesPositionCloseHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, 0, time.Time{}, time.Time{}, "buy", "")
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetFuturesPositionCloseHistory must reject a side other than long or short")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesPositionCloseHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, 0, time.Time{}, time.Time{}, "long", "")
	require.NoError(t, err, "GetFuturesPositionCloseHistory must not error")
	if mockTests {
		exp := []*PositionCloseHistoryResponse{
			{
				Time:                  types.Time(time.UnixMilli(1546487347123)),
				ProfitAndLoss:         0.00013,
				ProfitAndLossPosition: 0.00011,
				ProfitAndLossFunding:  0.00001,
				ProfitAndLossFee:      0.00001,
				Side:                  "long",
				Contract:              "BTC_USD",
				Text:                  "web",
				MaxSize:               100,
				AccumulatedClosedSize: 100,
				FirstOpenTime:         types.Time(time.Unix(1546487300, 0)),
				LongAveragePrice:      2026.87,
				ShortAveragePrice:     2544.4,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesPositionCloseHistory should decode every field")
	}
}

func TestGetFuturesLiquidationHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesLiquidationHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 0, 0, time.Time{}, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesLiquidationHistory must reject an empty settlement currency")

	startTime, endTime := getTime()
	_, err = e.GetFuturesLiquidationHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, 0, endTime, startTime, time.Time{})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesLiquidationHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesLiquidationHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, 0, time.Time{}, time.Time{}, time.Time{})
	require.NoError(t, err, "GetFuturesLiquidationHistory must not error")
	if mockTests {
		exp := []*LiquidationHistoryItemResponse{
			{
				Time:             types.Time(time.Unix(1548654951, 0)),
				Contract:         currency.NewPairWithDelimiter("BTC", "USD", "_"),
				Size:             600,
				Leverage:         25,
				Margin:           0.006705256878,
				EntryPrice:       3536.123,
				MarkPrice:        3420.27,
				OrderPrice:       3405,
				FillPrice:        3424,
				LiquidationPrice: 3421.54,
				OrderID:          317393847,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesLiquidationHistory should decode every field")
	}
}

func TestGetFuturesAutoDeleverages(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesAutoDeleverages(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures), 0, 0, time.Time{}, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesAutoDeleverages must reject an empty settlement currency")

	startTime, endTime := getTime()
	_, err = e.GetFuturesAutoDeleverages(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 0, 0, endTime, startTime, time.Time{})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesAutoDeleverages must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesAutoDeleverages(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10, 0, startTime, endTime, time.Time{})
	require.NoError(t, err, "GetFuturesAutoDeleverages must not error")
	if mockTests {
		exp := []*FuturesAutoDeleverageResponse{
			{
				Time:               types.Time(time.Unix(1675841679, 0)),
				User:               1666,
				OrderID:            73873128,
				Contract:           currency.NewPairWithDelimiter("BTC", "USDT", "_"),
				CrossLeverageLimit: 10,
				EntryPrice:         2649.648633636364,
				FillPrice:          2790.8082,
				TradeSize:          -10,
				PositionSize:       1,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesAutoDeleverages should decode every field")
	}
}

func TestCountdownCancelFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CountdownCancelFuturesOrders(t.Context(), currency.EMPTYCODE, &CountdownRequest{Countdown: types.DurationSeconds(8 * time.Second)})
	require.ErrorIs(t, err, errEmptyOrInvalidSettlementCurrency, "CountdownCancelFuturesOrders must reject an empty settlement currency")

	_, err = e.CountdownCancelFuturesOrders(t.Context(), currency.BTC, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CountdownCancelFuturesOrders must reject a nil request")

	for _, timeout := range []time.Duration{-time.Second, time.Second, 4 * time.Second, 4999 * time.Millisecond, 5500 * time.Millisecond} {
		_, err = e.CountdownCancelFuturesOrders(t.Context(), currency.BTC, &CountdownRequest{Countdown: types.DurationSeconds(timeout)})
		require.ErrorIsf(t, err, errInvalidTimeout, "CountdownCancelFuturesOrders must reject a timeout of %s", timeout)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CountdownCancelFuturesOrders(t.Context(), currency.BTC, &CountdownRequest{Countdown: types.DurationSeconds(8 * time.Second)})
	require.NoError(t, err, "CountdownCancelFuturesOrders must not error")
	if mockTests {
		exp := &TriggerTimeResponse{
			TriggerTime: types.Time(time.Unix(1660039145, 0)),
		}
		assert.Equal(t, exp, result, "CountdownCancelFuturesOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CountdownCancelFuturesOrders should return the trigger time")
}

func TestGetFuturesTradingFees(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTradingFees(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesTradingFees must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesTradingFees(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures))
	require.NoError(t, err, "GetFuturesTradingFees must not error")
	if mockTests {
		exp := map[string]*FuturesTradingFeeResponse{
			"BTC_USDT": {
				TakerFee: 0.00075,
				MakerFee: -0.0001,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesTradingFees should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetFuturesTradingFees should return fee rates")
}

func TestCancelBatchFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchFuturesOrders(t.Context(), currency.EMPTYCODE, []string{"1"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelBatchFuturesOrders must reject an empty settlement currency")

	_, err = e.CancelBatchFuturesOrders(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, errNoOrdersToCancel, "CancelBatchFuturesOrders must reject an empty order list")

	_, err = e.CancelBatchFuturesOrders(t.Context(), currency.USDT, make([]string, 21))
	require.ErrorIs(t, err, errTooManyOrders, "CancelBatchFuturesOrders must reject more than 20 orders")

	_, err = e.CancelBatchFuturesOrders(t.Context(), currency.USDT, []string{"1", ""})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelBatchFuturesOrders must reject an empty order ID")

	ex := e
	if mockTests {
		ex = futuresLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "batch cancellation should be requested with POST")
			assert.Equal(t, "/api/v4/futures/usdt/batch_cancel_orders", r.URL.Path, "batch cancellation should be requested from the batch_cancel_orders endpoint")
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the request body should not error")
			assert.JSONEq(t, `["1","2"]`, string(body), "batch cancellation should send the order IDs as the body")
			_, err = w.Write([]byte(`[{"user_id":111,"id":"1","succeeded":true,"message":""},{"user_id":111,"id":"2","succeeded":false,"message":"ORDER_NOT_FOUND"}]`))
			assert.NoError(t, err, "writing the cancellation results should not error")
		})
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := ex.CancelBatchFuturesOrders(t.Context(), currency.USDT, []string{"1", "2"})
	require.NoError(t, err, "CancelBatchFuturesOrders must not error")
	if mockTests {
		exp := []*CancelFuturesOrderResultResponse{
			{
				UserID:    111,
				ID:        "1",
				Succeeded: true,
			},
			{
				UserID:  111,
				ID:      "2",
				Message: "ORDER_NOT_FOUND",
			},
		}
		assert.Equal(t, exp, result, "CancelBatchFuturesOrders should decode every field")
		return
	}
	assert.Len(t, result, 2, "CancelBatchFuturesOrders should report each order")
}

func TestAmendBatchFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.AmendBatchFuturesOrders(t.Context(), currency.EMPTYCODE, []*BatchAmendFuturesOrderRequest{{OrderID: 1, Price: 1}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "AmendBatchFuturesOrders must reject an empty settlement currency")

	_, err = e.AmendBatchFuturesOrders(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, errOrdersEmpty, "AmendBatchFuturesOrders must reject an empty batch")

	_, err = e.AmendBatchFuturesOrders(t.Context(), currency.USDT, make([]*BatchAmendFuturesOrderRequest, 11))
	require.ErrorIs(t, err, errTooManyOrders, "AmendBatchFuturesOrders must reject more than 10 orders")

	_, err = e.AmendBatchFuturesOrders(t.Context(), currency.USDT, []*BatchAmendFuturesOrderRequest{nil})
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendBatchFuturesOrders must reject a nil amendment")

	_, err = e.AmendBatchFuturesOrders(t.Context(), currency.USDT, []*BatchAmendFuturesOrderRequest{{Price: 1}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendBatchFuturesOrders must reject an amendment without an order ID or text")

	_, err = e.AmendBatchFuturesOrders(t.Context(), currency.USDT, []*BatchAmendFuturesOrderRequest{{OrderID: 1}})
	require.ErrorIs(t, err, errNoOrderAmendment, "AmendBatchFuturesOrders must reject an amendment without a size or price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AmendBatchFuturesOrders(t.Context(), currency.USDT, []*BatchAmendFuturesOrderRequest{{OrderID: 121212, AmendText: "batch amend text", Size: 100, Price: 54321}})
	require.NoError(t, err, "AmendBatchFuturesOrders must not error")
	if mockTests {
		amendedOrder := futuresFixtureOrder(getPair(t, asset.USDTMarginedFutures))
		amendedOrder.Size, amendedOrder.Price, amendedOrder.AmendText = 100, 54321, "batch amend text"
		exp := []*BatchFuturesOrderResponse{{Succeeded: true, FuturesOrderResponse: *amendedOrder}}
		assert.Equal(t, exp, result, "AmendBatchFuturesOrders should decode every field")
		return
	}
	assert.Len(t, result, 1, "AmendBatchFuturesOrders should report the order")
}

func TestPlaceFuturesBBOOrder(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.PlaceFuturesBBOOrder(t.Context(), currency.EMPTYCODE, &FuturesBBOOrderRequest{Contract: contract, Size: 1, Direction: "sell", Level: 8})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceFuturesBBOOrder must reject an empty settlement currency")

	_, err = e.PlaceFuturesBBOOrder(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceFuturesBBOOrder must reject a nil order")

	_, err = e.PlaceFuturesBBOOrder(t.Context(), currency.USDT, &FuturesBBOOrderRequest{Size: 1, Direction: "sell", Level: 8})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceFuturesBBOOrder must reject an empty contract")

	_, err = e.PlaceFuturesBBOOrder(t.Context(), currency.USDT, &FuturesBBOOrderRequest{Contract: contract, Direction: "sell", Level: 8})
	require.ErrorIs(t, err, errInvalidOrderSize, "PlaceFuturesBBOOrder must reject an empty size")

	_, err = e.PlaceFuturesBBOOrder(t.Context(), currency.USDT, &FuturesBBOOrderRequest{Contract: contract, Size: 1, Direction: "ask", Level: 8})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "PlaceFuturesBBOOrder must reject a direction other than buy or sell")

	for _, level := range []uint64{0, 21} {
		_, err = e.PlaceFuturesBBOOrder(t.Context(), currency.USDT, &FuturesBBOOrderRequest{Contract: contract, Size: 1, Direction: "sell", Level: level})
		require.ErrorIsf(t, err, errInvalidOrderBookLevel, "PlaceFuturesBBOOrder must reject order book level %d", level)
	}

	_, err = e.PlaceFuturesBBOOrder(t.Context(), currency.USDT, &FuturesBBOOrderRequest{Contract: contract, Size: 1, Direction: "sell", Level: 8, TimeInForce: "day"})
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "PlaceFuturesBBOOrder must reject an unknown time in force")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceFuturesBBOOrder(t.Context(), currency.USDT, &FuturesBBOOrderRequest{Contract: contract, Size: 1, Direction: "sell", Level: 8})
	require.NoError(t, err, "PlaceFuturesBBOOrder must not error")
	if mockTests {
		exp := futuresFixtureOrder(getPair(t, asset.USDTMarginedFutures))
		exp.Size, exp.RemainingAmount, exp.TimeInForce = 1, 1, pocTIF
		assert.Equal(t, exp, result, "PlaceFuturesBBOOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "PlaceFuturesBBOOrder should return the order")
}

func TestCreateTrailOrder(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.CreateTrailOrder(t.Context(), currency.EMPTYCODE, &CreateTrailOrderRequest{Contract: contract, Amount: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateTrailOrder must reject an empty settlement currency")

	_, err = e.CreateTrailOrder(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateTrailOrder must reject a nil order")

	_, err = e.CreateTrailOrder(t.Context(), currency.USDT, &CreateTrailOrderRequest{Amount: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CreateTrailOrder must reject an empty contract")

	_, err = e.CreateTrailOrder(t.Context(), currency.USDT, &CreateTrailOrderRequest{Contract: contract})
	require.ErrorIs(t, err, errInvalidOrderSize, "CreateTrailOrder must reject an empty amount")

	_, err = e.CreateTrailOrder(t.Context(), currency.USDT, &CreateTrailOrderRequest{Contract: contract, Amount: 10, PriceType: 4})
	require.ErrorIs(t, err, errInvalidTriggerPriceType, "CreateTrailOrder must reject an unknown price type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateTrailOrder(t.Context(), currency.USDT, &CreateTrailOrderRequest{
		Contract:           contract,
		Amount:             10,
		ActivationPrice:    50000,
		IsGreaterThanEqual: true,
		PriceType:          1,
		PriceOffset:        "0.1%",
		Text:               "apiv4",
	})
	require.NoError(t, err, "CreateTrailOrder must not error")
	if mockTests {
		exp := &CreateTrailOrderResponse{
			ID: NumericString("63648"),
		}
		assert.Equal(t, exp, result, "CreateTrailOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateTrailOrder should return the order ID")
}

func TestStopTrailOrder(t *testing.T) {
	t.Parallel()
	_, err := e.StopTrailOrder(t.Context(), currency.EMPTYCODE, &StopTrailOrderRequest{ID: 63585})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "StopTrailOrder must reject an empty settlement currency")

	_, err = e.StopTrailOrder(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "StopTrailOrder must reject a nil request")

	_, err = e.StopTrailOrder(t.Context(), currency.USDT, &StopTrailOrderRequest{})
	require.ErrorIs(t, err, errTrailOrderIDOrTextRequired, "StopTrailOrder must reject a request without an ID or text")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.StopTrailOrder(t.Context(), currency.USDT, &StopTrailOrderRequest{ID: 63585})
	require.NoError(t, err, "StopTrailOrder must not error")
	if mockTests {
		exp := futuresFixtureTrailOrder()
		assert.Equal(t, exp, result, "StopTrailOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "StopTrailOrder should return the order")
}

func TestStopAllTrailOrders(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.StopAllTrailOrders(t.Context(), currency.EMPTYCODE, &StopAllTrailOrdersRequest{Contract: contract})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "StopAllTrailOrders must reject an empty settlement currency")

	_, err = e.StopAllTrailOrders(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "StopAllTrailOrders must reject a nil request")

	_, err = e.StopAllTrailOrders(t.Context(), currency.USDT, &StopAllTrailOrdersRequest{RelatedPosition: 3})
	require.ErrorIs(t, err, errInvalidTrailOrderFilter, "StopAllTrailOrders must reject an unknown related position")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.StopAllTrailOrders(t.Context(), currency.USDT, &StopAllTrailOrdersRequest{Contract: contract})
	require.NoError(t, err, "StopAllTrailOrders must not error")
	if mockTests {
		exp := []*TrailOrderResponse{futuresFixtureTrailOrder()}
		assert.Equal(t, exp, result, "StopAllTrailOrders should decode every field")
	}
}

func TestGetTrailOrders(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.GetTrailOrders(t.Context(), currency.EMPTYCODE, &GetTrailOrdersRequest{Contract: contract})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTrailOrders must reject an empty settlement currency")

	_, err = e.GetTrailOrders(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTrailOrders must reject a nil request")

	for _, arg := range []*GetTrailOrdersRequest{{SortBy: 3}, {RelatedPosition: 3}, {ReduceOnly: 3}, {Side: 3}} {
		_, err = e.GetTrailOrders(t.Context(), currency.USDT, arg)
		require.ErrorIsf(t, err, errInvalidTrailOrderFilter, "GetTrailOrders must reject the filter %+v", arg)
	}

	startTime, endTime := getTime()
	_, err = e.GetTrailOrders(t.Context(), currency.USDT, &GetTrailOrdersRequest{StartAt: endTime, EndAt: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetTrailOrders must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTrailOrders(t.Context(), currency.USDT, &GetTrailOrdersRequest{Contract: contract, IsFinished: true, PageNumber: 1, PageSize: 10, SortBy: 1})
	require.NoError(t, err, "GetTrailOrders must not error")
	if mockTests {
		exp := []*TrailOrderResponse{futuresFixtureTrailOrder()}
		assert.Equal(t, exp, result, "GetTrailOrders should decode every field")
	}
}

func TestGetTrailOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrailOrderDetail(t.Context(), currency.EMPTYCODE, 63585)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTrailOrderDetail must reject an empty settlement currency")

	_, err = e.GetTrailOrderDetail(t.Context(), currency.USDT, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetTrailOrderDetail must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTrailOrderDetail(t.Context(), currency.USDT, 63585)
	require.NoError(t, err, "GetTrailOrderDetail must not error")
	if mockTests {
		exp := futuresFixtureTrailOrder()
		assert.Equal(t, exp, result, "GetTrailOrderDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetTrailOrderDetail should return the order")
}

func TestUpdateTrailOrder(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTrailOrder(t.Context(), currency.EMPTYCODE, &UpdateTrailOrderRequest{ID: 63585, Amount: 20})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdateTrailOrder must reject an empty settlement currency")

	_, err = e.UpdateTrailOrder(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdateTrailOrder must reject a nil request")

	_, err = e.UpdateTrailOrder(t.Context(), currency.USDT, &UpdateTrailOrderRequest{Amount: 20})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "UpdateTrailOrder must reject an empty order ID")

	_, err = e.UpdateTrailOrder(t.Context(), currency.USDT, &UpdateTrailOrderRequest{ID: 63585, IsGreaterThanEqual: "yes"})
	require.ErrorIs(t, err, errInvalidTrailOrderFilter, "UpdateTrailOrder must reject an is_gte_str other than true or false")

	_, err = e.UpdateTrailOrder(t.Context(), currency.USDT, &UpdateTrailOrderRequest{ID: 63585, PriceType: 4})
	require.ErrorIs(t, err, errInvalidTriggerPriceType, "UpdateTrailOrder must reject an unknown price type")

	_, err = newFixedResponseExchange(t, "/api/v4/futures/usdt/autoorder/v1/trail/update", `null`).UpdateTrailOrder(t.Context(), currency.USDT, &UpdateTrailOrderRequest{ID: 63585, Amount: 20})
	require.ErrorIs(t, err, common.ErrNoResponse, "UpdateTrailOrder must reject a response without the order")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UpdateTrailOrder(t.Context(), currency.USDT, &UpdateTrailOrderRequest{ID: 63585, Amount: 20, ActivationPrice: "51000", PriceOffset: "0.2%"})
	require.NoError(t, err, "UpdateTrailOrder must not error")
	if mockTests {
		exp := futuresFixtureTrailOrder()
		assert.Equal(t, exp, result, "UpdateTrailOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "UpdateTrailOrder should return the order")
}

func TestGetTrailOrderChangeLog(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrailOrderChangeLog(t.Context(), currency.EMPTYCODE, 63585, 1, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTrailOrderChangeLog must reject an empty settlement currency")

	_, err = e.GetTrailOrderChangeLog(t.Context(), currency.USDT, 0, 1, 10)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetTrailOrderChangeLog must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTrailOrderChangeLog(t.Context(), currency.USDT, 63585, 1, 10)
	require.NoError(t, err, "GetTrailOrderChangeLog must not error")
	if mockTests {
		exp := []*TrailOrderChangeResponse{
			{
				UpdatedAt:          types.Time(time.Unix(1546569968, 0)),
				Amount:             10,
				IsGreaterThanEqual: true,
				ActivationPrice:    50000,
				PriceType:          1,
				PriceOffset:        "0.1%",
				IsCreate:           true,
			},
			{
				UpdatedAt:       types.Time(time.Unix(1546570000, 0)),
				Amount:          20,
				ActivationPrice: 51000,
				PriceType:       2,
				PriceOffset:     "0.2%",
			},
		}
		assert.Equal(t, exp, result, "GetTrailOrderChangeLog should decode every field")
	}
}

func TestTrailOrderReplyUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := &TrailOrderResponse{ID: "63585", Contract: currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter), Amount: 10}
	for _, payload := range []string{
		`{"order":{"id":"63585","contract":"BTC_USDT","amount":"10"}}`,
		`{"id":63585,"contract":"BTC_USDT","amount":"10"}`,
	} {
		var reply trailOrderReply
		require.NoErrorf(t, json.Unmarshal([]byte(payload), &reply), "Unmarshal must not error for %s", payload)
		assert.Equalf(t, exp, reply.Order, "trailOrderReply should decode the order from %s", payload)
	}
}

func TestFuturesEmptyOrderReplies(t *testing.T) {
	t.Parallel()
	ex := futuresLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		reply := `{}`
		switch r.URL.Path {
		case "/api/v4/futures/btc/positions/BTC_USD/leverage", "/api/v4/futures/usdt/autoorder/v1/trail/stop":
			reply = `null`
		case "/api/v4/futures/usdt/autoorder/v1/trail/detail":
			reply = `{"code":0,"message":"ok","data":null,"timestamp":1769584936814}`
		}
		_, err := w.Write([]byte(reply))
		assert.NoError(t, err, "writing the empty reply should not error")
	})
	for name, call := range map[string]func() (any, error){
		"UpdateFuturesPositionLeverage": func() (any, error) {
			return ex.UpdateFuturesPositionLeverage(t.Context(), currency.BTC, currency.NewPairWithDelimiter("BTC", "USD", currency.UnderscoreDelimiter), 1, 0, 0)
		},
		"StopTrailOrder": func() (any, error) {
			return ex.StopTrailOrder(t.Context(), currency.USDT, &StopTrailOrderRequest{ID: 63585})
		},
		"GetTrailOrderDetail": func() (any, error) { return ex.GetTrailOrderDetail(t.Context(), currency.USDT, 63585) },
		"StopChaseOrder": func() (any, error) {
			return ex.StopChaseOrder(t.Context(), currency.USDT, &StopChaseOrderRequest{ID: "1001"})
		},
		"GetChaseOrderDetail": func() (any, error) { return ex.GetChaseOrderDetail(t.Context(), currency.USDT, "1001") },
	} {
		_, err := call()
		assert.ErrorIsf(t, err, common.ErrNoResponse, "%s should report an empty reply", name)
	}
}

func TestCreateChaseOrder(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.CreateChaseOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateChaseOrder must reject a nil order")

	_, err = e.CreateChaseOrder(t.Context(), &CreateChaseOrderRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateChaseOrder must reject an empty settlement currency")

	_, err = e.CreateChaseOrder(t.Context(), &CreateChaseOrderRequest{Settle: currency.USDT})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CreateChaseOrder must reject an empty contract")

	_, err = e.CreateChaseOrder(t.Context(), &CreateChaseOrderRequest{Settle: currency.USDT, Contract: contract})
	require.ErrorIs(t, err, errInvalidOrderSize, "CreateChaseOrder must reject an empty amount")

	_, err = e.CreateChaseOrder(t.Context(), &CreateChaseOrderRequest{Settle: currency.USDT, Contract: contract, Amount: 1, PriceLimit: 68000, OffsetLimit: 50})
	require.ErrorIs(t, err, errChasePriceAndOffsetLimit, "CreateChaseOrder must reject both a price limit and an offset limit")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateChaseOrder(t.Context(), &CreateChaseOrderRequest{Settle: currency.USDT, Contract: contract, Amount: 1, PriceLimit: 68000})
	require.NoError(t, err, "CreateChaseOrder must not error")
	if mockTests {
		exp := &CreateChaseOrderResponse{
			ID: NumericString("1001"),
		}
		assert.Equal(t, exp, result, "CreateChaseOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateChaseOrder should return the order ID")
}

func TestCreateChaseOrderRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(&CreateChaseOrderRequest{
		Contract:    currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
		Amount:      -2,
		OffsetLimit: 50,
		PriceType:   2,
		Settle:      currency.USDT,
	})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"contract":"BTC_USDT","amount":"-2","price_limit":"0","offset_limit":"50","price_type":2}`, string(payload), "CreateChaseOrderRequest should send a zero price limit when chasing by offset")
}

func TestStopChaseOrder(t *testing.T) {
	t.Parallel()
	_, err := e.StopChaseOrder(t.Context(), currency.EMPTYCODE, &StopChaseOrderRequest{ID: "1001"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "StopChaseOrder must reject an empty settlement currency")

	_, err = e.StopChaseOrder(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "StopChaseOrder must reject a nil request")

	_, err = e.StopChaseOrder(t.Context(), currency.USDT, &StopChaseOrderRequest{})
	require.ErrorIs(t, err, errChaseOrderIDOrTextRequired, "StopChaseOrder must reject a request without an ID or text")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.StopChaseOrder(t.Context(), currency.USDT, &StopChaseOrderRequest{ID: "1001"})
	require.NoError(t, err, "StopChaseOrder must not error")
	if mockTests {
		exp := futuresFixtureChaseOrder()
		assert.Equal(t, exp, result, "StopChaseOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "StopChaseOrder should return the order")
}

func TestStopAllChaseOrders(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	_, err := e.StopAllChaseOrders(t.Context(), currency.EMPTYCODE, &StopAllChaseOrdersRequest{Contract: contract})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "StopAllChaseOrders must reject an empty settlement currency")

	_, err = e.StopAllChaseOrders(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "StopAllChaseOrders must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.StopAllChaseOrders(t.Context(), currency.USDT, &StopAllChaseOrdersRequest{Contract: contract})
	require.NoError(t, err, "StopAllChaseOrders must not error")
	if mockTests {
		exp := []*ChaseOrderResponse{futuresFixtureChaseOrder()}
		assert.Equal(t, exp, result, "StopAllChaseOrders should decode every field")
	}
}

func TestStopAllChaseOrdersRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		arg StopAllChaseOrdersRequest
		exp string
	}{
		{arg: StopAllChaseOrdersRequest{}, exp: `{}`},
		{arg: StopAllChaseOrdersRequest{PositionMarginMode: "cross"}, exp: `{"pos_margin_mode":"cross"}`},
		{arg: StopAllChaseOrdersRequest{Contract: currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter), PositionMarginMode: "isolated"}, exp: `{"contract":"BTC_USDT","pos_margin_mode":"isolated"}`},
	} {
		payload, err := json.Marshal(&tc.arg)
		require.NoError(t, err, "Marshal must not error")
		assert.Equal(t, tc.exp, string(payload), "StopAllChaseOrdersRequest should omit an empty contract and margin mode")
	}
}

func TestGetChaseOrders(t *testing.T) {
	t.Parallel()
	contract := getPair(t, asset.USDTMarginedFutures)
	startTime, endTime := getTime()
	_, err := e.GetChaseOrders(t.Context(), currency.EMPTYCODE, &GetChaseOrdersRequest{SortBy: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetChaseOrders must reject an empty settlement currency")

	_, err = e.GetChaseOrders(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetChaseOrders must reject a nil request")

	_, err = e.GetChaseOrders(t.Context(), currency.USDT, &GetChaseOrdersRequest{})
	require.ErrorIs(t, err, errInvalidChaseSortBy, "GetChaseOrders must reject a missing sort field")

	_, err = e.GetChaseOrders(t.Context(), currency.USDT, &GetChaseOrdersRequest{SortBy: 1, IsFinished: true})
	require.ErrorIs(t, err, errChaseTimeRangeRequired, "GetChaseOrders must reject finished orders without a time range")

	_, err = e.GetChaseOrders(t.Context(), currency.USDT, &GetChaseOrdersRequest{SortBy: 1, StartAt: endTime, EndAt: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetChaseOrders must reject a start after the end")

	_, err = e.GetChaseOrders(t.Context(), currency.USDT, &GetChaseOrdersRequest{SortBy: 1, PageSize: 101})
	require.ErrorIs(t, err, errInvalidLimit, "GetChaseOrders must reject a page size above 100")

	for _, arg := range []*GetChaseOrdersRequest{{SortBy: 1, ReduceOnly: 3}, {SortBy: 1, Side: 3}} {
		_, err = e.GetChaseOrders(t.Context(), currency.USDT, arg)
		require.ErrorIsf(t, err, errInvalidChaseOrderFilter, "GetChaseOrders must reject the filter %+v", arg)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetChaseOrders(t.Context(), currency.USDT, &GetChaseOrdersRequest{Contract: contract, IsFinished: true, StartAt: startTime, EndAt: endTime, PageNumber: 1, PageSize: 100, SortBy: 1})
	require.NoError(t, err, "GetChaseOrders must not error")
	if mockTests {
		exp := []*ChaseOrderResponse{futuresFixtureChaseOrder()}
		assert.Equal(t, exp, result, "GetChaseOrders should decode every field")
	}
}

func TestGetChaseOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetChaseOrderDetail(t.Context(), currency.EMPTYCODE, "1001")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetChaseOrderDetail must reject an empty settlement currency")

	_, err = e.GetChaseOrderDetail(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetChaseOrderDetail must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetChaseOrderDetail(t.Context(), currency.USDT, "1001")
	require.NoError(t, err, "GetChaseOrderDetail must not error")
	if mockTests {
		exp := futuresFixtureChaseOrder()
		assert.Equal(t, exp, result, "GetChaseOrderDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetChaseOrderDetail should return the order")
}

func TestCreatePriceTriggeredFuturesOrder(t *testing.T) {
	t.Parallel()
	arg := &FuturesPriceTriggeredOrderRequest{
		Initial: FuturesInitial{Price: 1234, Size: 2, Contract: getPair(t, asset.USDTMarginedFutures)},
		Trigger: FuturesTrigger{Rule: 1, Price: 3000},
	}
	_, err := e.CreatePriceTriggeredFuturesOrder(t.Context(), currency.EMPTYCODE, arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreatePriceTriggeredFuturesOrder must reject an empty settlement currency")

	_, err = e.CreatePriceTriggeredFuturesOrder(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreatePriceTriggeredFuturesOrder must reject a nil order")

	for _, tc := range []struct {
		name        string
		mutate      func(*FuturesPriceTriggeredOrderRequest)
		expectedErr error
	}{
		{"empty contract", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.Contract = currency.EMPTYPAIR }, currency.ErrCurrencyPairEmpty},
		{"negative price", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.Price = -1 }, limits.ErrPriceBelowMin},
		{"fok time in force", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.TimeInForce = fokTIF }, order.ErrInvalidTimeInForce},
		{"unknown strategy type", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.StrategyType = 2 }, errInvalidTriggerStrategyType},
		{"unknown price type", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.PriceType = 3 }, errInvalidTriggerPriceType},
		{"missing trigger price", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.Price = 0 }, limits.ErrPriceBelowMin},
		{"unknown rule", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.Rule = 3 }, errInvalidTriggerRule},
		{"fractional trigger expiry", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.ExpireAfter = fractionalSeconds }, errNotWholeSeconds},
		{"unknown order type", func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "close-all" }, errInvalidAutoOrderType},
	} {
		invalid := *arg
		tc.mutate(&invalid)
		_, err = e.CreatePriceTriggeredFuturesOrder(t.Context(), currency.USDT, &invalid)
		require.ErrorIsf(t, err, tc.expectedErr, "CreatePriceTriggeredFuturesOrder must reject an order with a %s", tc.name)
	}

	ex := e
	if mockTests {
		ex = futuresLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "price triggered orders should be created with POST")
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the request body should not error")
			switch r.URL.Path {
			case "/api/v4/futures/btc/price_orders":
				assert.JSONEq(t, `{"initial":{"contract":"BTC_USD","size":2,"price":"1234"},"trigger":{"price":"3000","rule":1},"order_type":"close-short-position"}`, string(body), "the BTC price triggered order should send the documented body")
			case "/api/v4/futures/usdt/price_orders":
				assert.JSONEq(t, `{"initial":{"contract":"BTC_USDT","size":0,"price":"0","tif":"ioc","reduce_only":true,"auto_size":"close_long"},"trigger":{"price_type":1,"price":"3000","rule":2,"expiration":86400},"order_type":"plan-close-long-position","pos_margin_mode":"isolated"}`, string(body), "a market close price triggered order should send a zero size and price")
			default:
				assert.Failf(t, "unexpected price triggered order path", "price triggered orders should only be created on the BTC and USDT paths, not %s", r.URL.Path)
			}
			_, err = w.Write([]byte(`{"id":1432329,"id_string":"1432329"}`))
			assert.NoError(t, err, "writing the order ID should not error")
		})
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = ex.CreatePriceTriggeredFuturesOrder(t.Context(), currency.BTC, &FuturesPriceTriggeredOrderRequest{
		Initial:   FuturesInitial{Price: 1234, Size: 2, Contract: getPair(t, asset.CoinMarginedFutures)},
		Trigger:   FuturesTrigger{Rule: 1, Price: 3000},
		OrderType: "close-short-position",
	})
	require.NoError(t, err, "CreatePriceTriggeredFuturesOrder must not error for a BTC contract")

	result, err := ex.CreatePriceTriggeredFuturesOrder(t.Context(), currency.USDT, &FuturesPriceTriggeredOrderRequest{
		Initial:            FuturesInitial{Contract: getPair(t, asset.USDTMarginedFutures), TimeInForce: iocTIF, ReduceOnly: true, AutoSize: "close_long"},
		Trigger:            FuturesTrigger{PriceType: 1, Price: 3000, Rule: 2, ExpireAfter: types.DurationSeconds(24 * time.Hour)},
		OrderType:          "plan-close-long-position",
		PositionMarginMode: "isolated",
	})
	require.NoError(t, err, "CreatePriceTriggeredFuturesOrder must not error for a market close")
	if mockTests {
		exp := &TriggerOrderIDResponse{
			ID:       1432329,
			IDString: "1432329",
		}
		assert.Equal(t, exp, result, "CreatePriceTriggeredFuturesOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CreatePriceTriggeredFuturesOrder should return the order ID")
}

func TestListAllFuturesAutoOrders(t *testing.T) {
	t.Parallel()
	_, err := e.ListAllFuturesAutoOrders(t.Context(), "pending", currency.BTC, currency.EMPTYPAIR, 0, 0)
	require.ErrorIs(t, err, errInvalidOrderStatus, "ListAllFuturesAutoOrders must reject an unknown status")

	_, err = e.ListAllFuturesAutoOrders(t.Context(), statusOpen, currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "ListAllFuturesAutoOrders must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.ListAllFuturesAutoOrders(t.Context(), statusOpen, currency.BTC, currency.EMPTYPAIR, 0, 0)
	require.NoError(t, err, "ListAllFuturesAutoOrders must not error")
	if mockTests {
		exp := []*PriceTriggeredOrderResponse{futuresFixturePriceTriggeredOrder()}
		assert.Equal(t, exp, result, "ListAllFuturesAutoOrders should decode every field")
	}
}

func TestCancelAllFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllFuturesOpenOrders(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelAllFuturesOpenOrders must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.CancelAllFuturesOpenOrders(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures))
	require.NoError(t, err, "CancelAllFuturesOpenOrders must not error for a contract")

	result, err := e.CancelAllFuturesOpenOrders(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.NoError(t, err, "CancelAllFuturesOpenOrders must not error for every contract")
	if mockTests {
		exp := []*PriceTriggeredOrderResponse{futuresFixturePriceTriggeredOrder()}
		assert.Equal(t, exp, result, "CancelAllFuturesOpenOrders should decode every field")
	}
}

func TestGetSingleFuturesPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleFuturesPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSingleFuturesPriceTriggeredOrder must reject an empty settlement currency")

	_, err = e.GetSingleFuturesPriceTriggeredOrder(t.Context(), currency.BTC, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSingleFuturesPriceTriggeredOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSingleFuturesPriceTriggeredOrder(t.Context(), currency.BTC, "12345")
	require.NoError(t, err, "GetSingleFuturesPriceTriggeredOrder must not error")
	if mockTests {
		exp := futuresFixturePriceTriggeredOrder()
		assert.Equal(t, exp, result, "GetSingleFuturesPriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleFuturesPriceTriggeredOrder should return the order")
}

func TestCancelFuturesPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFuturesPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelFuturesPriceTriggeredOrder must reject an empty settlement currency")

	_, err = e.CancelFuturesPriceTriggeredOrder(t.Context(), currency.BTC, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelFuturesPriceTriggeredOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelFuturesPriceTriggeredOrder(t.Context(), currency.USDT, "12345")
	require.NoError(t, err, "CancelFuturesPriceTriggeredOrder must not error")
	if mockTests {
		exp := futuresFixturePriceTriggeredOrder()
		assert.Equal(t, exp, result, "CancelFuturesPriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelFuturesPriceTriggeredOrder should return the order")
}

func TestAmendFuturesPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	zeroSize, marketPrice := int64(0), 0.0
	_, err := e.AmendFuturesPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, &AmendFuturesPriceTriggeredOrderRequest{OrderID: 1432329, TriggerPrice: 3100})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "AmendFuturesPriceTriggeredOrder must reject an empty settlement currency")

	_, err = e.AmendFuturesPriceTriggeredOrder(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendFuturesPriceTriggeredOrder must reject a nil amendment")

	_, err = e.AmendFuturesPriceTriggeredOrder(t.Context(), currency.USDT, &AmendFuturesPriceTriggeredOrderRequest{TriggerPrice: 3100})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendFuturesPriceTriggeredOrder must reject an empty order ID")

	_, err = e.AmendFuturesPriceTriggeredOrder(t.Context(), currency.USDT, &AmendFuturesPriceTriggeredOrderRequest{OrderID: 1432329})
	require.ErrorIs(t, err, errNoOrderAmendment, "AmendFuturesPriceTriggeredOrder must reject an amendment that changes nothing")

	negativePrice, unknownPriceType := -1.0, int64(3)
	_, err = e.AmendFuturesPriceTriggeredOrder(t.Context(), currency.USDT, &AmendFuturesPriceTriggeredOrderRequest{OrderID: 1432329, Price: &negativePrice})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "AmendFuturesPriceTriggeredOrder must reject a negative price")

	_, err = e.AmendFuturesPriceTriggeredOrder(t.Context(), currency.USDT, &AmendFuturesPriceTriggeredOrderRequest{OrderID: 1432329, PriceType: &unknownPriceType})
	require.ErrorIs(t, err, errInvalidTriggerPriceType, "AmendFuturesPriceTriggeredOrder must reject an unknown price type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AmendFuturesPriceTriggeredOrder(t.Context(), currency.USDT, &AmendFuturesPriceTriggeredOrderRequest{OrderID: 1432329, Size: &zeroSize, Price: &marketPrice, TriggerPrice: 3100})
	require.NoError(t, err, "AmendFuturesPriceTriggeredOrder must not error")
	if mockTests {
		exp := &TriggerOrderIDResponse{
			ID:       1432329,
			IDString: "1432329",
		}
		assert.Equal(t, exp, result, "AmendFuturesPriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "AmendFuturesPriceTriggeredOrder should return the order ID")
}

func TestAmendFuturesPriceTriggeredOrderRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	price, priceType := 3050.5, int64(0)
	for _, tc := range []struct {
		name string
		arg  AmendFuturesPriceTriggeredOrderRequest
		exp  string
	}{
		{name: "unset fields are left unchanged", arg: AmendFuturesPriceTriggeredOrderRequest{OrderID: 1, TriggerPrice: 3100}, exp: `{"order_id":1,"trigger_price":"3100"}`},
		{name: "set fields are sent even when zero", arg: AmendFuturesPriceTriggeredOrderRequest{OrderID: 1, Price: &price, PriceType: &priceType, Amount: 1.5, AutoSize: "close_short", Close: true}, exp: `{"order_id":1,"amount":"1.5","price":"3050.5","price_type":0,"auto_size":"close_short","close":true}`},
	} {
		payload, err := json.Marshal(&tc.arg)
		require.NoErrorf(t, err, "Marshal must not error when %s", tc.name)
		assert.Equalf(t, tc.exp, string(payload), "AmendFuturesPriceTriggeredOrderRequest should marshal as documented when %s", tc.name)
	}
}

func TestCountdownRequestsJSON(t *testing.T) {
	t.Parallel()
	pair := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name    string
		request any
		decoded any
		exp     string
	}{
		{name: "spot", request: &CountdownCancelOrderRequest{CurrencyPair: pair, Countdown: types.DurationSeconds(10 * time.Second)}, decoded: new(CountdownCancelOrderRequest), exp: `{"currency_pair":"BTC_USDT","timeout":10}`},
		{name: "spot stop", request: &CountdownCancelOrderRequest{}, decoded: new(CountdownCancelOrderRequest), exp: `{"timeout":0}`},
		{name: "futures", request: &CountdownRequest{Contract: pair, Countdown: types.DurationSeconds(30 * time.Second)}, decoded: new(CountdownRequest), exp: `{"timeout":30,"contract":"BTC_USDT"}`},
		{name: "futures stop", request: &CountdownRequest{}, decoded: new(CountdownRequest), exp: `{"timeout":0}`},
		{
			name: "options",
			request: &CountdownCancelOptionOrdersRequest{
				Countdown:  types.DurationSeconds(30 * time.Second),
				Contract:   currency.NewPairWithDelimiter("BTC", "USDT-20260925-75000-P", currency.UnderscoreDelimiter),
				Underlying: "BTC_USDT",
			},
			decoded: new(CountdownCancelOptionOrdersRequest),
			exp:     `{"timeout":30,"contract":"BTC_USDT-20260925-75000-P","underlying":"BTC_USDT"}`,
		},
		{name: "options stop", request: &CountdownCancelOptionOrdersRequest{}, decoded: new(CountdownCancelOptionOrdersRequest), exp: `{"timeout":0}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.request)
			require.NoError(t, err, "Marshal must not error")
			assert.JSONEq(t, tc.exp, string(got), "Marshal should send the timeout in seconds and keep a zero timeout")
			require.NoError(t, json.Unmarshal(got, tc.decoded), "Unmarshal must not error")
			assert.Equal(t, tc.request, tc.decoded, "Unmarshal should read back the request")
		})
	}
}
