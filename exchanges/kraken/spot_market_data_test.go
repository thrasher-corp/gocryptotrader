package kraken

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetCurrentServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetCurrentServerTime(t.Context())
	require.NoError(t, err, "GetCurrentServerTime must not error")
	if mockTests {
		exp := &ServerTimeResponse{
			UnixTime: types.Time(time.Unix(1791504602, 0)),
			RFC1123:  "Fri, 09 Oct 26 00:10:02 +0000",
		}
		assert.Equal(t, exp, result, "GetCurrentServerTime should decode every field")
		return
	}
	assert.NotZero(t, result.UnixTime, "GetCurrentServerTime should return a time")
}

func TestGetSystemStatus(t *testing.T) {
	t.Parallel()
	result, err := e.GetSystemStatus(t.Context())
	require.NoError(t, err, "GetSystemStatus must not error")
	if mockTests {
		exp := &SystemStatusResponse{
			Status:    "cancel_only",
			Timestamp: time.Date(2026, 5, 8, 14, 28, 0, 0, time.UTC),
			UpcomingMaintenance: []MaintenanceEvent{
				{
					EventID:            21,
					Title:              "Scheduled Maintenance - Website",
					ExpectedStart:      time.Date(2026, 5, 11, 9, 0, 0, 0, time.UTC),
					ExpectedEnd:        time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC),
					TimeToStartSeconds: 1740,
					Phase:              "approaching_30m",
					AffectedServices:   []string{"spot_trading"},
					OrderSubmission:    "allowed",
					RecommendedAction:  "reduce_activity",
					CancelBefore:       time.Date(2026, 5, 11, 8, 55, 0, 0, time.UTC),
					SourceURL:          "https://status.kraken.com/incidents/b7k2r9wqmn41",
				},
			},
			Emergency: []EmergencyEvent{
				{
					EventID:          4821,
					Title:            "Elevated API error rates",
					IncidentStatus:   "identified",
					Impact:           "critical",
					AffectedServices: []string{"spot_ws", "spot_rest", "spot_fix"},
					StartedAt:        time.Date(2026, 5, 8, 14, 23, 11, 0, time.UTC),
					NextSteps: []EmergencyNextStep{
						{
							AppliesTo:  []string{"spot_trading"},
							Type:       "expected_restart",
							ExpectedAt: time.Date(2026, 5, 8, 14, 30, 0, 0, time.UTC),
						},
					},
					SourceURL: "https://stspg.io/c378t4f7rh0n",
				},
			},
		}
		assert.Equal(t, exp, result, "GetSystemStatus should decode every field")
		return
	}
	assert.NotEmpty(t, result.Status, "GetSystemStatus should return a status")
}

func TestGetMaintenanceSchedule(t *testing.T) {
	t.Parallel()
	result, err := e.GetMaintenanceSchedule(t.Context())
	require.NoError(t, err, "GetMaintenanceSchedule must not error")
	if !mockTests {
		return
	}
	exp := &MaintenanceScheduleResponse{
		Events: []MaintenanceEvent{
			{
				EventID:            21,
				Title:              "Scheduled Maintenance - Website",
				ExpectedStart:      time.Date(2026, 5, 11, 9, 0, 0, 0, time.UTC),
				ExpectedEnd:        time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC),
				TimeToStartSeconds: 1740,
				Phase:              "approaching_30m",
				AffectedServices:   []string{"spot_trading"},
				OrderSubmission:    "allowed",
				RecommendedAction:  "reduce_activity",
				CancelBefore:       time.Date(2026, 5, 11, 8, 55, 0, 0, time.UTC),
				SourceURL:          "https://status.kraken.com/incidents/b7k2r9wqmn41",
			},
			{
				EventID:            34,
				Title:              "REST API infrastructure upgrade",
				ExpectedStart:      time.Date(2026, 5, 14, 2, 0, 0, 0, time.UTC),
				ExpectedEnd:        time.Date(2026, 5, 14, 2, 30, 0, 0, time.UTC),
				TimeToStartSeconds: 236940,
				Phase:              "announced",
				AffectedServices:   []string{"spot_rest", "spot_ws"},
				OrderSubmission:    "allowed",
				RecommendedAction:  "continue",
				CancelBefore:       time.Date(2026, 5, 14, 1, 55, 0, 0, time.UTC),
				SourceURL:          "https://status.kraken.com/incidents/d3f8h1xpqv62",
			},
		},
	}
	assert.Equal(t, exp, result, "GetMaintenanceSchedule should decode every field")
}

func TestGetAssets(t *testing.T) {
	t.Parallel()
	_, err := e.GetAssets(t.Context(), &AssetsRequest{Assets: []currency.Code{currency.EMPTYCODE}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAssets must reject an empty asset")

	result, err := e.GetAssets(t.Context(), &AssetsRequest{Assets: []currency.Code{currency.XBT, currency.ETH}})
	require.NoError(t, err, "GetAssets must not error")
	exp := map[string]AssetInfoResponse{
		"XETH": {AssetClass: "currency", AlternativeName: "ETH", Decimals: 10, DisplayDecimals: 5, CollateralValue: 0.99, Status: "enabled", MarginRate: 0.02},
		"XXBT": {AssetClass: "currency", AlternativeName: "XBT", Decimals: 10, DisplayDecimals: 5, CollateralValue: 0.99, Status: "enabled", MarginRate: 0.01},
	}
	if mockTests {
		assert.Equal(t, exp, result, "GetAssets should decode every field")
	} else {
		assert.Len(t, result, 2, "GetAssets should return the requested assets")
	}

	result, err = e.GetAssets(t.Context(), &AssetsRequest{Assets: []currency.Code{currency.XBT}, AssetClass: "currency", DisplayNames: true})
	require.NoError(t, err, "GetAssets must not error with display names")
	require.Contains(t, result, "BTC", "GetAssets must key the asset by its display name")
	assert.Equal(t, "XBT", result["BTC"].AlternativeName, "GetAssets should keep the alternative name")
}

func TestGetAssetPairs(t *testing.T) {
	t.Parallel()
	_, err := e.GetAssetPairs(t.Context(), &AssetPairsRequest{Pairs: currency.Pairs{currency.EMPTYPAIR}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetAssetPairs must reject an empty pair")

	result, err := e.GetAssetPairs(t.Context(), &AssetPairsRequest{Pairs: currency.Pairs{spotTestPair}})
	require.NoError(t, err, "GetAssetPairs must not error")
	require.Contains(t, result, "XXBTZUSD", "GetAssetPairs must key the pair by its internal name")
	if mockTests {
		exp := AssetPairResponse{
			AlternativeName:    "XBTUSD",
			WebsocketName:      "XBT/USD",
			BaseAssetClass:     "currency",
			Base:               currency.XXBT,
			QuoteAssetClass:    "currency",
			Quote:              currency.ZUSD,
			ExecutionVenue:     "international",
			Lot:                "unit",
			PairDecimals:       1,
			CostDecimals:       5,
			LotDecimals:        8,
			LotMultiplier:      1,
			LeverageBuy:        []uint64{2, 3, 4, 5},
			LeverageSell:       []uint64{2, 3, 4},
			FeeVolumeCurrency:  currency.ZUSD,
			MarginCall:         80,
			MarginStop:         40,
			OrderMinimum:       0.00005,
			CostMinimum:        0.5,
			TickSize:           0.1,
			Status:             "online",
			LongPositionLimit:  350,
			ShortPositionLimit: 250,
		}
		assert.Equal(t, exp, result["XXBTZUSD"], "GetAssetPairs should decode every field")
	}

	result, err = e.GetAssetPairs(t.Context(), &AssetPairsRequest{
		Pairs:           currency.Pairs{spotTestPair},
		BaseAssetClass:  "currency",
		Info:            "info",
		CountryCode:     "GB",
		ExecutionVenues: []string{"international"},
		DisplayNames:    true,
	})
	require.NoError(t, err, "GetAssetPairs must not error with every parameter")
	require.Contains(t, result, "BTC/USD", "GetAssetPairs must key the pair by its display name")
	assert.Equal(t, currency.BTC, result["BTC/USD"].Base, "GetAssetPairs should name the base by its display name")
	assert.Equal(t, "XBT/USD", result["BTC/USD"].WebsocketName, "GetAssetPairs should keep the websocket name")

	result, err = e.GetAssetPairs(t.Context(), &AssetPairsRequest{Pairs: currency.Pairs{spotTestPair}, Info: "margin"})
	require.NoError(t, err, "GetAssetPairs must not error for margin info")
	require.Contains(t, result, "XXBTZUSD", "GetAssetPairs must return the margin info's pair")
	if mockTests {
		assert.Equal(t, AssetPairResponse{MarginCall: 80, MarginLevel: 40}, result["XXBTZUSD"], "GetAssetPairs should decode the margin info")
		return
	}
	assert.NotZero(t, result["XXBTZUSD"].MarginLevel, "GetAssetPairs should return the margin info's stop-out level")
}

func TestGetTickerInformation(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickerInformation(t.Context(), &TickerInformationRequest{Pairs: currency.Pairs{currency.EMPTYPAIR}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetTickerInformation must reject an empty pair")

	result, err := e.GetTickerInformation(t.Context(), &TickerInformationRequest{Pairs: currency.Pairs{spotTestPair}})
	require.NoError(t, err, "GetTickerInformation must not error")
	require.Contains(t, result, "XXBTZUSD", "GetTickerInformation must key the ticker by its internal pair name")
	if mockTests {
		exp := TickerInformationResponse{
			Ask:                        TickerLevel{Price: 81720.1, WholeLotVolume: 1, LotVolume: 1},
			Bid:                        TickerLevel{Price: 81720, WholeLotVolume: 2, LotVolume: 2},
			LastTradeClosed:            TickerLastTrade{Price: 81720.1, LotVolume: 0.00029992},
			Volume:                     TickerValues{Today: 3.5524723, Last24Hours: 4011.48934847},
			VolumeWeightedAveragePrice: TickerValues{Today: 81717.59466, Last24Hours: 81876.00203},
			NumberOfTrades:             TickerTradeCounts{Today: 1115, Last24Hours: 196973},
			Low:                        TickerValues{Today: 81656.4, Last24Hours: 80328.6},
			High:                       TickerValues{Today: 81771.5, Last24Hours: 83467.7},
			OpeningPrice:               81683.8,
		}
		assert.Equal(t, exp, result["XXBTZUSD"], "GetTickerInformation should decode every field")
	} else {
		assert.Positive(t, result["XXBTZUSD"].Ask.Price.Float64(), "GetTickerInformation should return an ask")
	}

	result, err = e.GetTickerInformation(t.Context(), &TickerInformationRequest{Pairs: currency.Pairs{spotTestPair}, AssetClass: "forex", DisplayNames: true})
	require.NoError(t, err, "GetTickerInformation must not error with display names")
	assert.Contains(t, result, "BTC/USD", "GetTickerInformation should key the ticker by its display name")
}

func TestGetOHLCData(t *testing.T) {
	t.Parallel()
	_, err := e.GetOHLCData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOHLCData must reject a nil request")
	_, err = e.GetOHLCData(t.Context(), &OHLCDataRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOHLCData must reject an empty pair")
	_, err = e.GetOHLCData(t.Context(), &OHLCDataRequest{Pair: spotTestPair, Interval: kline.TwoHour})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetOHLCData must reject an unsupported interval")

	result, err := e.GetOHLCData(t.Context(), &OHLCDataRequest{Pair: spotTestPair, Interval: kline.OneDay, Since: time.Unix(1791400000, 0)})
	require.NoError(t, err, "GetOHLCData must not error")
	if mockTests {
		exp := &OHLCDataResponse{
			Candles: map[string][]Candle{
				"XXBTZUSD": {
					{Time: types.Time(time.Unix(1791417600, 0)), Open: 83276.1, High: 83467.7, Low: 80328.6, Close: 81683.8, VolumeWeightedAveragePrice: 81878.5, Volume: 4015.12325849, TradeCount: 196775},
					{Time: types.Time(time.Unix(1791504000, 0)), Open: 81683.8, High: 81771.5, Low: 81656.4, Close: 81723.4, VolumeWeightedAveragePrice: 81717.5, Volume: 3.55298125, TradeCount: 1117},
				},
			},
			Last: types.Time(time.Unix(1791417600, 0)),
		}
		assert.Equal(t, exp, result, "GetOHLCData should decode every field")
		return
	}
	assert.NotEmpty(t, result.Candles, "GetOHLCData should return candles")
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderBook must reject a nil request")
	_, err = e.GetOrderBook(t.Context(), &OrderBookRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderBook must reject an empty pair")
	_, err = e.GetOrderBook(t.Context(), &OrderBookRequest{Pair: spotTestPair, Count: 501})
	require.ErrorIs(t, err, errInvalidCount, "GetOrderBook must reject a count above 500")

	result, err := e.GetOrderBook(t.Context(), &OrderBookRequest{Pair: spotTestPair, Count: 2})
	require.NoError(t, err, "GetOrderBook must not error")
	require.Contains(t, result, "XXBTZUSD", "GetOrderBook must key the book by its internal pair name")
	if mockTests {
		exp := OrderBookResponse{
			Asks: []OrderBookLevel{
				{Price: 81720.1, Volume: 0.021, Timestamp: types.Time(time.Unix(1791504605, 0))},
				{Price: 81720.2, Volume: 0.025, Timestamp: types.Time(time.Unix(1791504603, 0))},
			},
			Bids: []OrderBookLevel{
				{Price: 81720, Volume: 0.345, Timestamp: types.Time(time.Unix(1791504605, 0))},
				{Price: 81719.7, Volume: 0.001, Timestamp: types.Time(time.Unix(1791504597, 0))},
			},
		}
		assert.Equal(t, exp, result["XXBTZUSD"], "GetOrderBook should decode every field")
		return
	}
	assert.Len(t, result["XXBTZUSD"].Asks, 2, "GetOrderBook should return the requested number of asks")
}

func TestGetLevel3OrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetLevel3OrderBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLevel3OrderBook must reject a nil request")
	_, err = e.GetLevel3OrderBook(t.Context(), &Level3OrderBookRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetLevel3OrderBook must reject an empty pair")
	for _, pair := range partialTestPairs {
		_, err = e.GetLevel3OrderBook(t.Context(), &Level3OrderBookRequest{Pair: pair})
		require.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "GetLevel3OrderBook must reject %q, which lacks a base or a quote", pair)
	}
	_, err = e.GetLevel3OrderBook(t.Context(), &Level3OrderBookRequest{Pair: spotTestPair, Depth: 11})
	require.ErrorIs(t, err, errInvalidDepth, "GetLevel3OrderBook must reject an undocumented depth")
	_, err = e.GetLevel3OrderBook(t.Context(), &Level3OrderBookRequest{Pair: spotTestPair, Depth: 10, FullBook: true})
	require.ErrorIs(t, err, errInvalidDepth, "GetLevel3OrderBook must reject a depth with the full book")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLevel3OrderBook(t.Context(), &Level3OrderBookRequest{Pair: spotTestPair, Depth: 10})
	require.NoError(t, err, "GetLevel3OrderBook must not error")
	if mockTests {
		exp := &Level3OrderBookResponse{
			Pair: "XBTUSD",
			Bids: []Level3Order{
				{Price: 81720, Quantity: 0.296658, OrderID: "O5KJU4-IEQTM-NDMS6W", Timestamp: types.Time(time.Unix(0, 1791504605594292000))},
				{Price: 81719.7, Quantity: 0.139174, OrderID: "OERRY6-MXYER-6EQKNY", Timestamp: types.Time(time.Unix(0, 1791504597396903000))},
			},
			Asks: []Level3Order{
				{Price: 81720.1, Quantity: 0.00278335, OrderID: "ORAWGV-N5L4J-LBA3WH", Timestamp: types.Time(time.Unix(0, 1791504605499456000))},
			},
		}
		assert.Equal(t, exp, result, "GetLevel3OrderBook should decode every field")
		return
	}
	assert.NotEmpty(t, result.Bids, "GetLevel3OrderBook should return bids")
}

func TestGetGroupedOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetGroupedOrderBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetGroupedOrderBook must reject a nil request")
	_, err = e.GetGroupedOrderBook(t.Context(), &GroupedOrderBookRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetGroupedOrderBook must reject an empty pair")
	for _, pair := range partialTestPairs {
		_, err = e.GetGroupedOrderBook(t.Context(), &GroupedOrderBookRequest{Pair: pair})
		require.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "GetGroupedOrderBook must reject %q, which lacks a base or a quote", pair)
	}
	_, err = e.GetGroupedOrderBook(t.Context(), &GroupedOrderBookRequest{Pair: spotTestPair, Depth: 11})
	require.ErrorIs(t, err, errInvalidDepth, "GetGroupedOrderBook must reject an undocumented depth")
	_, err = e.GetGroupedOrderBook(t.Context(), &GroupedOrderBookRequest{Pair: spotTestPair, Grouping: 2})
	require.ErrorIs(t, err, errInvalidGrouping, "GetGroupedOrderBook must reject an undocumented grouping")

	result, err := e.GetGroupedOrderBook(t.Context(), &GroupedOrderBookRequest{Pair: spotTestPair, Depth: 10, Grouping: 1000})
	require.NoError(t, err, "GetGroupedOrderBook must not error")
	if mockTests {
		exp := &GroupedOrderBookResponse{
			Pair:     "XBTUSD",
			Grouping: 1000,
			Bids:     []GroupedOrderLevel{{Price: 81700, Quantity: 19.76152972}, {Price: 81600, Quantity: 65.47726206}},
			Asks:     []GroupedOrderLevel{{Price: 81800, Quantity: 70.45672131}, {Price: 81900, Quantity: 62.58250439}},
		}
		assert.Equal(t, exp, result, "GetGroupedOrderBook should decode every field")
		return
	}
	assert.Equal(t, uint64(1000), result.Grouping, "GetGroupedOrderBook should return the requested grouping")
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTrades must reject a nil request")
	_, err = e.GetTrades(t.Context(), &RecentTradesRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetTrades must reject an empty pair")
	_, err = e.GetTrades(t.Context(), &RecentTradesRequest{Pair: spotTestPair, Count: 1001})
	require.ErrorIs(t, err, errInvalidCount, "GetTrades must reject a count above 1000")

	result, err := e.GetTrades(t.Context(), &RecentTradesRequest{Pair: spotTestPair, Since: time.Unix(1791504600, 0), Count: 2})
	require.NoError(t, err, "GetTrades must not error")
	if mockTests {
		exp := &RecentTradesResponse{
			Trades: map[string][]RecentTrade{
				"XXBTZUSD": {
					{Price: 81720.1, Volume: 0.00029992, Time: types.Time(time.Unix(1791504601, 92063400)), Side: "b", OrderType: "l", TradeID: 110760685},
					{Price: 81723.4, Volume: 0.00005006, Time: types.Time(time.Unix(1791504605, 893529400)), Side: "s", OrderType: "m", TradeID: 110760687},
				},
			},
			Last: types.Time(time.Unix(0, 1791504605893529482)),
		}
		assert.Equal(t, exp, result, "GetTrades should decode every field")
		return
	}
	assert.NotEmpty(t, result.Trades, "GetTrades should return trades")
}

func TestGetRecentSpreads(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentSpreads(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRecentSpreads must reject a nil request")
	_, err = e.GetRecentSpreads(t.Context(), &RecentSpreadsRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetRecentSpreads must reject an empty pair")

	result, err := e.GetRecentSpreads(t.Context(), &RecentSpreadsRequest{Pair: spotTestPair, Since: time.Unix(1791504000, 0)})
	require.NoError(t, err, "GetRecentSpreads must not error")
	if mockTests {
		exp := &RecentSpreadsResponse{
			Spreads: map[string][]Spread{
				"XXBTZUSD": {
					{Time: types.Time(time.Unix(1791504435, 0)), Bid: 81694.1, Ask: 81697.2},
					{Time: types.Time(time.Unix(1791504446, 0)), Bid: 81688.7, Ask: 81691},
				},
			},
			Last: types.Time(time.Unix(1791504446, 0)),
		}
		assert.Equal(t, exp, result, "GetRecentSpreads should decode every field")
		return
	}
	assert.NotEmpty(t, result.Spreads, "GetRecentSpreads should return spreads")
}

func TestUnmarshalFixedArray(t *testing.T) {
	t.Parallel()
	var level OrderBookLevel
	fields := []any{&level.Price, &level.Volume, &level.Timestamp}
	require.NoError(t, unmarshalFixedArray([]byte(`["81720.10000","0.021",1791504605]`), fields), "unmarshalFixedArray must not error for one element per field")
	assert.Equal(t, OrderBookLevel{Price: 81720.1, Volume: 0.021, Timestamp: types.Time(time.Unix(1791504605, 0))}, level, "unmarshalFixedArray should decode each element into its field")
	assert.ErrorIs(t, unmarshalFixedArray([]byte(`["81720.10000","0.021"]`), fields), errUnexpectedLength, "unmarshalFixedArray should reject a short array")
	assert.ErrorIs(t, unmarshalFixedArray([]byte(`["81720.10000","0.021",1791504605,1]`), fields), errUnexpectedLength, "unmarshalFixedArray should reject a long array")
	assert.Error(t, unmarshalFixedArray([]byte(`{"price":"81720.10000"}`), fields[:1]), "unmarshalFixedArray should reject an object")
	assert.Error(t, unmarshalFixedArray([]byte(`[{}]`), fields[:1]), "unmarshalFixedArray should reject an element its field cannot hold")
}

func TestFixedArrayUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var level TickerLevel
	require.NoError(t, level.UnmarshalJSON([]byte(`["81720.10000","1","1.000"]`)), "TickerLevel.UnmarshalJSON must not error")
	assert.Equal(t, TickerLevel{Price: 81720.1, WholeLotVolume: 1, LotVolume: 1}, level, "TickerLevel.UnmarshalJSON should decode each element")
	assert.ErrorIs(t, level.UnmarshalJSON([]byte(`[]`)), errUnexpectedLength, "TickerLevel.UnmarshalJSON should reject an empty array")

	var lastTrade TickerLastTrade
	require.NoError(t, lastTrade.UnmarshalJSON([]byte(`["81720.10000","0.00029992"]`)), "TickerLastTrade.UnmarshalJSON must not error")
	assert.Equal(t, TickerLastTrade{Price: 81720.1, LotVolume: 0.00029992}, lastTrade, "TickerLastTrade.UnmarshalJSON should decode each element")
	assert.ErrorIs(t, lastTrade.UnmarshalJSON([]byte(`[]`)), errUnexpectedLength, "TickerLastTrade.UnmarshalJSON should reject an empty array")

	var values TickerValues
	require.NoError(t, values.UnmarshalJSON([]byte(`["3.55247230","4011.48934847"]`)), "TickerValues.UnmarshalJSON must not error")
	assert.Equal(t, TickerValues{Today: 3.5524723, Last24Hours: 4011.48934847}, values, "TickerValues.UnmarshalJSON should decode each element")
	assert.ErrorIs(t, values.UnmarshalJSON([]byte(`[]`)), errUnexpectedLength, "TickerValues.UnmarshalJSON should reject an empty array")

	var counts TickerTradeCounts
	require.NoError(t, counts.UnmarshalJSON([]byte(`[1115,196973]`)), "TickerTradeCounts.UnmarshalJSON must not error")
	assert.Equal(t, TickerTradeCounts{Today: 1115, Last24Hours: 196973}, counts, "TickerTradeCounts.UnmarshalJSON should decode each element")
	assert.ErrorIs(t, counts.UnmarshalJSON([]byte(`[]`)), errUnexpectedLength, "TickerTradeCounts.UnmarshalJSON should reject an empty array")

	var candle Candle
	require.NoError(t, candle.UnmarshalJSON([]byte(`[1791417600,"83276.1","83467.7","80328.6","81683.8","81878.5","4015.12325849",196775]`)), "Candle.UnmarshalJSON must not error")
	exp := Candle{
		Time:                       types.Time(time.Unix(1791417600, 0)),
		Open:                       83276.1,
		High:                       83467.7,
		Low:                        80328.6,
		Close:                      81683.8,
		VolumeWeightedAveragePrice: 81878.5,
		Volume:                     4015.12325849,
		TradeCount:                 196775,
	}
	assert.Equal(t, exp, candle, "Candle.UnmarshalJSON should decode each element")
	assert.ErrorIs(t, candle.UnmarshalJSON([]byte(`[]`)), errUnexpectedLength, "Candle.UnmarshalJSON should reject an empty array")

	var bookLevel OrderBookLevel
	require.NoError(t, bookLevel.UnmarshalJSON([]byte(`["81720.10000","0.021",1791504605]`)), "OrderBookLevel.UnmarshalJSON must not error")
	assert.Equal(t, OrderBookLevel{Price: 81720.1, Volume: 0.021, Timestamp: types.Time(time.Unix(1791504605, 0))}, bookLevel, "OrderBookLevel.UnmarshalJSON should decode each element")
	assert.ErrorIs(t, bookLevel.UnmarshalJSON([]byte(`[]`)), errUnexpectedLength, "OrderBookLevel.UnmarshalJSON should reject an empty array")

	var trade RecentTrade
	require.NoError(t, trade.UnmarshalJSON([]byte(`["81720.10000","0.00029992",1791504601.0920634,"b","l","",110760685]`)), "RecentTrade.UnmarshalJSON must not error")
	assert.Equal(t, RecentTrade{Price: 81720.1, Volume: 0.00029992, Time: types.Time(time.Unix(1791504601, 92063400)), Side: "b", OrderType: "l", TradeID: 110760685}, trade, "RecentTrade.UnmarshalJSON should decode each element")
	assert.ErrorIs(t, trade.UnmarshalJSON([]byte(`[]`)), errUnexpectedLength, "RecentTrade.UnmarshalJSON should reject an empty array")

	var spread Spread
	require.NoError(t, spread.UnmarshalJSON([]byte(`[1791504435,"81694.10000","81697.20000"]`)), "Spread.UnmarshalJSON must not error for a whole spread")
	assert.Equal(t, Spread{Time: types.Time(time.Unix(1791504435, 0)), Bid: 81694.1, Ask: 81697.2}, spread, "Spread.UnmarshalJSON should decode each element")
	assert.ErrorIs(t, spread.UnmarshalJSON([]byte(`[1791504435,"81694.10000"]`)), errUnexpectedLength, "Spread.UnmarshalJSON should reject a short array")
	assert.ErrorIs(t, spread.UnmarshalJSON([]byte(`[1791504435,"81694.10000","81697.20000","1"]`)), errUnexpectedLength, "Spread.UnmarshalJSON should reject a long array")
}

func TestPairKeyedResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var ohlc OHLCDataResponse
	require.NoError(t, ohlc.UnmarshalJSON([]byte(`{"XXBTZUSD":[[1791417600,"83276.1","83467.7","80328.6","81683.8","81878.5","4015.12325849",196775]],"last":1791417600}`)), "OHLCDataResponse.UnmarshalJSON must not error")
	expOHLC := OHLCDataResponse{
		Candles: map[string][]Candle{"XXBTZUSD": {{
			Time:                       types.Time(time.Unix(1791417600, 0)),
			Open:                       83276.1,
			High:                       83467.7,
			Low:                        80328.6,
			Close:                      81683.8,
			VolumeWeightedAveragePrice: 81878.5,
			Volume:                     4015.12325849,
			TradeCount:                 196775,
		}}},
		Last: types.Time(time.Unix(1791417600, 0)),
	}
	assert.Equal(t, expOHLC, ohlc, "OHLCDataResponse.UnmarshalJSON should key the candles by pair beside the last time")

	var trades RecentTradesResponse
	require.NoError(t, trades.UnmarshalJSON([]byte(`{"XXBTZUSD":[["81720.10000","0.00029992",1791504601.0920634,"b","l","",110760685]],"last":"1791504605893529482"}`)), "RecentTradesResponse.UnmarshalJSON must not error")
	expTrades := RecentTradesResponse{
		Trades: map[string][]RecentTrade{"XXBTZUSD": {{Price: 81720.1, Volume: 0.00029992, Time: types.Time(time.Unix(1791504601, 92063400)), Side: "b", OrderType: "l", TradeID: 110760685}}},
		Last:   types.Time(time.Unix(0, 1791504605893529482)),
	}
	assert.Equal(t, expTrades, trades, "RecentTradesResponse.UnmarshalJSON should key the trades by pair beside the last time")

	var spreads RecentSpreadsResponse
	require.NoError(t, spreads.UnmarshalJSON([]byte(`{"XXBTZUSD":[[1791504435,"81694.10000","81697.20000"]],"last":1791504446}`)), "RecentSpreadsResponse.UnmarshalJSON must not error")
	expSpreads := RecentSpreadsResponse{
		Spreads: map[string][]Spread{"XXBTZUSD": {{Time: types.Time(time.Unix(1791504435, 0)), Bid: 81694.1, Ask: 81697.2}}},
		Last:    types.Time(time.Unix(1791504446, 0)),
	}
	assert.Equal(t, expSpreads, spreads, "RecentSpreadsResponse.UnmarshalJSON should key the spreads by pair beside the last time")

	for _, data := range []string{`[]`, `{"last":"soon"}`, `{"XXBTZUSD":{}}`} {
		assert.Errorf(t, ohlc.UnmarshalJSON([]byte(data)), "OHLCDataResponse.UnmarshalJSON should reject %s", data)
		assert.Errorf(t, trades.UnmarshalJSON([]byte(data)), "RecentTradesResponse.UnmarshalJSON should reject %s", data)
		assert.Errorf(t, spreads.UnmarshalJSON([]byte(data)), "RecentSpreadsResponse.UnmarshalJSON should reject %s", data)
	}
}

func TestSpotPairParams(t *testing.T) {
	t.Parallel()
	_, err := e.spotPairParams(currency.EMPTYPAIR, "", false)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "spotPairParams should reject an empty pair")
	for _, pair := range partialTestPairs {
		_, err = e.spotPairParams(pair, "", false)
		assert.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "spotPairParams should reject %q, which lacks a base or a quote", pair)
	}
	params, err := e.spotPairParams(spotTestPair, "", false)
	require.NoError(t, err, "spotPairParams must not error")
	assert.Equal(t, url.Values{"pair": {"XBTUSD"}}, params, "spotPairParams should set the pair")
	params, err = e.spotPairParams(spotTestPair, "tokenized_asset", true)
	require.NoError(t, err, "spotPairParams must not error with an asset class and display names")
	assert.Equal(t, url.Values{"pair": {"XBTUSD"}, "asset_class": {"tokenized_asset"}, "assetVersion": {"1"}}, params, "spotPairParams should set the asset class and ask for display names")
}

func TestFormatSpotPairs(t *testing.T) {
	t.Parallel()
	_, err := e.formatSpotPairs(currency.Pairs{spotTestPair, currency.EMPTYPAIR})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "formatSpotPairs should reject an empty pair")
	for _, pair := range partialTestPairs {
		_, err = e.formatSpotPairs(currency.Pairs{spotTestPair, pair})
		assert.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "formatSpotPairs should reject %q, which lacks a base or a quote", pair)
	}
	symbols, err := e.formatSpotPairs(currency.Pairs{spotTestPair, currency.NewPairWithDelimiter("ETH", "XBT", currency.UnderscoreDelimiter)})
	require.NoError(t, err, "formatSpotPairs must not error")
	assert.Equal(t, "XBTUSD,ETHXBT", symbols, "formatSpotPairs should join the pairs' alternative names")
	symbols, err = e.formatSpotPairs(nil)
	require.NoError(t, err, "formatSpotPairs must not error without pairs")
	assert.Empty(t, symbols, "formatSpotPairs should return nothing without pairs")
}
