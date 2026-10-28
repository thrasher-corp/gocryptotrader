package kucoin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

func TestUTAWebsocketOrderDecoding(t *testing.T) {
	t.Parallel()
	exp := []UTAWebsocketOrder{
		{
			OrderID:           "409225265957388288",
			ClientOrderID:     "39350f44-56ef-4b91-9788-708d9ed83e95",
			OrderStatus:       3,
			EventType:         "FILL",
			TradeType:         "SPOT",
			Symbol:            "BTC-USDT",
			Side:              "BUY",
			OrderType:         "MARKET",
			LiquidityRole:     "TAKER",
			OrderSource:       "USER",
			TradeID:           "21105572357619712",
			Quantity:          0.00001,
			QuantityUnit:      "BASECCY",
			FilledSize:        0.00001,
			LastFilledSize:    0.00001,
			LastFilledPrice:   66421,
			AveragePrice:      66421,
			Fee:               0.00066421,
			FeeCurrency:       currency.USDT,
			Tax:               0.0000066,
			TimeInForce:       "GTC",
			CreationTimestamp: types.Time(time.Unix(0, 1770380108442909926)),
			UpdateTimestamp:   types.Time(time.Unix(0, 1770380108526000000)),
			BrokerTag:         "KuCoinPartner",
		},
		{
			OrderID:             "409225796465532928",
			ClientOrderID:       "fd8e9080-6c52-4d1e-9ae6-7f845ca19ab3",
			OrderStatus:         2,
			EventType:           "OPEN",
			TradeType:           "SPOT",
			Symbol:              "BTC-USDT",
			Side:                "SELL",
			OrderType:           "LIMIT",
			OrderSource:         "USER",
			Price:               66350,
			Quantity:            0.00001,
			QuantityUnit:        "BASECCY",
			FeeCurrency:         currency.USDT,
			RemainingSize:       0.00001,
			SelfTradePrevention: "DC",
			TimeInForce:         "GTC",
			PostOnly:            true,
			CreationTimestamp:   types.Time(time.Unix(0, 1770380234925178051)),
			UpdateTimestamp:     types.Time(time.Unix(0, 1770380234925510461)),
		},
		{
			OrderID:           "409092444108747642",
			EventType:         "OPEN",
			TradeType:         "FUTURES",
			Symbol:            "XBTUSDTM",
			Side:              "SELL",
			OrderType:         "MARKET",
			OrderSource:       "USER",
			Quantity:          1,
			QuantityUnit:      "UNIT",
			FeeCurrency:       currency.USDT,
			RemainingSize:     1,
			TriggerDirection:  "UP",
			TriggerPrice:      64570,
			TriggerPriceType:  "TP",
			TriggerOrderID:    "409092443980505088",
			ReduceOnly:        true,
			TimeInForce:       "GTC",
			CreationTimestamp: types.Time(time.Unix(0, 1770348441250511762)),
			UpdateTimestamp:   types.Time(time.Unix(0, 1770348441251034550)),
		},
		{
			OrderID:                    "409094459008032768",
			ClientOrderID:              "8c0c0f02-0c53-4e75-92f6-4bdfacb34496",
			OrderStatus:                2,
			EventType:                  "OPEN",
			TradeType:                  "FUTURES",
			Symbol:                     "XBTUSDTM",
			Side:                       "BUY",
			PositionSide:               "BOTH",
			OrderType:                  "MARKET",
			OrderSource:                "USER",
			MarginMode:                 "CROSS",
			Quantity:                   1,
			QuantityUnit:               "UNIT",
			FeeCurrency:                currency.USDT,
			RemainingSize:              1,
			TakeProfitTriggerPrice:     64880,
			TakeProfitTriggerPriceType: "TP",
			TakeProfitOrderPrice:       64890,
			StopLossTriggerPrice:       64800,
			StopLossTriggerPriceType:   "TP",
			StopLossOrderPrice:         64790,
			TimeInForce:                "GTC",
			CreationTimestamp:          types.Time(time.Unix(0, 1770348921635950111)),
			UpdateTimestamp:            types.Time(time.Unix(0, 1770348921636284837)),
		},
		{
			OrderID:           "409094459027223421",
			OrderStatus:       5,
			EventType:         "CANCEL",
			TradeType:         "FUTURES",
			Symbol:            "XBTUSDTM",
			Side:              "SELL",
			OrderType:         "MARKET",
			OrderSource:       "USER",
			Quantity:          1,
			QuantityUnit:      "UNIT",
			FeeCurrency:       currency.USDT,
			CancelReason:      "SYSTEM_CANCEL",
			CancelledSize:     1,
			TriggerDirection:  "DOWN",
			TriggerPrice:      64800,
			TriggerPriceType:  "TP",
			TriggerOrderID:    "409094459008032768",
			ReduceOnly:        true,
			TimeInForce:       "GTC",
			CreationTimestamp: types.Time(time.Unix(0, 1770348921643622286)),
			UpdateTimestamp:   types.Time(time.Unix(0, 1770348960158129346)),
		},
		{
			OrderID:           "433439467777417216",
			ClientOrderID:     "273edcef-532d-46ba-9e0c-14a8da03ca9b",
			OrderStatus:       2,
			EventType:         "OPEN",
			TradeType:         "MARGIN",
			Symbol:            "XRP-USDT",
			Side:              "BUY",
			OrderType:         "MARKET",
			OrderSource:       "USER",
			MarginMode:        "CROSS",
			Quantity:          10,
			QuantityUnit:      "QUOTECCY",
			FeeCurrency:       currency.USDT,
			RemainingSize:     10,
			TimeInForce:       "GTC",
			CreationTimestamp: types.Time(time.Unix(0, 1776153224034091350)),
			UpdateTimestamp:   types.Time(time.Unix(0, 1776153224036325316)),
		},
	}
	assert.Equal(t, exp, utaFixtureData[UTAWebsocketOrder](t, "testdata/wsUTAOrder.jsonl"), "order pushes should decode every field, telling keys apart by case")
}

func TestProcessUTAOrder(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	for i, msg := range utaFixtureMessages(t, "testdata/wsUTAOrder.jsonl") {
		require.NoErrorf(t, ex.processUTAOrder(t.Context(), msg), "processUTAOrder must not error for fixture line %d", i+1)
	}
	xrpPair := currency.NewPairWithDelimiter("XRP", "USDT", currency.DashDelimiter)
	exp := []any{
		&order.Detail{TimeInForce: order.GoodTillCancel, AverageExecutedPrice: 66421, Amount: 0.00001, ExecutedAmount: 0.00001, Fee: 0.00066421, FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "409225265957388288", ClientOrderID: "39350f44-56ef-4b91-9788-708d9ed83e95", Type: order.Market, Side: order.Buy, Status: order.Filled, AssetType: asset.Spot, Date: time.Unix(0, 1770380108442909926), LastUpdated: time.Unix(0, 1770380108526000000), Pair: utaSpotTestPair},
		&order.Detail{TimeInForce: order.GoodTillCancel | order.PostOnly, Price: 66350, Amount: 0.00001, RemainingAmount: 0.00001, FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "409225796465532928", ClientOrderID: "fd8e9080-6c52-4d1e-9ae6-7f845ca19ab3", Type: order.Limit, Side: order.Sell, Status: order.Open, AssetType: asset.Spot, Date: time.Unix(0, 1770380234925178051), LastUpdated: time.Unix(0, 1770380234925510461), Pair: utaSpotTestPair},
		&order.Detail{TimeInForce: order.GoodTillCancel, ReduceOnly: true, TriggerPrice: 64570, Amount: 1, ContractAmount: 1, RemainingAmount: 1, FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "409092444108747642", Type: order.Market, Side: order.Sell, Status: order.Pending, AssetType: asset.Futures, Date: time.Unix(0, 1770348441250511762), LastUpdated: time.Unix(0, 1770348441251034550), Pair: utaFuturesTestPair},
		&order.Detail{TimeInForce: order.GoodTillCancel, Amount: 1, ContractAmount: 1, RemainingAmount: 1, FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "409094459008032768", ClientOrderID: "8c0c0f02-0c53-4e75-92f6-4bdfacb34496", Type: order.Market, Side: order.Buy, Status: order.Open, AssetType: asset.Futures, Date: time.Unix(0, 1770348921635950111), LastUpdated: time.Unix(0, 1770348921636284837), Pair: utaFuturesTestPair, MarginType: margin.Multi},
		&order.Detail{TimeInForce: order.GoodTillCancel, ReduceOnly: true, TriggerPrice: 64800, Amount: 1, ContractAmount: 1, FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "409094459027223421", Type: order.Market, Side: order.Sell, Status: order.Cancelled, AssetType: asset.Futures, Date: time.Unix(0, 1770348921643622286), LastUpdated: time.Unix(0, 1770348960158129346), Pair: utaFuturesTestPair},
		&order.Detail{TimeInForce: order.GoodTillCancel, QuoteAmount: 10, CostAsset: currency.USDT, FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "433439467777417216", ClientOrderID: "273edcef-532d-46ba-9e0c-14a8da03ca9b", Type: order.Market, Side: order.Buy, Status: order.Open, AssetType: asset.Margin, Date: time.Unix(0, 1776153224034091350), LastUpdated: time.Unix(0, 1776153224036325316), Pair: xrpPair, MarginType: margin.Multi},
	}
	assert.Equal(t, exp, drainUTARelay(ex), "processUTAOrder should relay each push as an order detail, its quantities in their unit")

	err := ex.processUTAOrder(t.Context(), utaMessage(t, `{"T":"orderAll.UNIFIED","P":1776153224046666053,"d":{"tT":"MARGIN","oi":"433439467777417216","os":3,"s":"XRP-USDT","S":"BUY","oT":"MARKET","mM":"CROSS","q":"10","qU":"QUOTECCY","fS":"10","rS":"0","aP":"1.01972","tIF":"GTC","O":1776153224034091350,"U":1776153224046325316}}`)) // codespell:ignore ot
	require.NoError(t, err, "processUTAOrder must not error for a filled quote currency order")
	exp = []any{&order.Detail{TimeInForce: order.GoodTillCancel, AverageExecutedPrice: 1.01972, QuoteAmount: 10, Cost: 10, CostAsset: currency.USDT, Exchange: ex.Name, OrderID: "433439467777417216", Type: order.Market, Side: order.Buy, Status: order.Filled, AssetType: asset.Margin, Date: time.Unix(0, 1776153224034091350), LastUpdated: time.Unix(0, 1776153224046325316), Pair: xrpPair, MarginType: margin.Multi}}
	assert.Equal(t, exp, drainUTARelay(ex), "processUTAOrder should report a quote currency order's filled size as its cost")

	for _, tc := range []struct {
		name  string
		data  string
		expIs error
	}{
		{name: "an unknown trade type", data: `{"tT":"OPTIONS","s":"BTC-USDT"}`, expIs: asset.ErrNotSupported},
		{name: "an unknown futures contract", data: `{"tT":"FUTURES","s":"NOTACONTRACTM"}`, expIs: currency.ErrPairNotFound},
		{name: "an empty spot symbol", data: `{"tT":"SPOT","s":""}`, expIs: currency.ErrCreatingPair},
		{name: "an unknown side", data: `{"tT":"SPOT","s":"BTC-USDT","S":"UP"}`, expIs: order.ErrSideIsInvalid},
		{name: "an unknown order type", data: `{"tT":"SPOT","s":"BTC-USDT","S":"BUY","oT":"ICEBERG"}`, expIs: order.ErrUnrecognisedOrderType},                                 // codespell:ignore ot
		{name: "an unknown status", data: `{"tT":"SPOT","s":"BTC-USDT","S":"BUY","oT":"LIMIT","os":7}`, expIs: order.ErrUnsupportedStatusType},                                // codespell:ignore ot
		{name: "an unknown time in force", data: `{"tT":"SPOT","s":"BTC-USDT","S":"BUY","oT":"LIMIT","os":2,"tIF":"DAY"}`, expIs: order.ErrInvalidTimeInForce},                // codespell:ignore ot
		{name: "an unknown margin mode", data: `{"tT":"SPOT","s":"BTC-USDT","S":"BUY","oT":"LIMIT","os":2,"tIF":"GTC","mM":"PORTFOLIO"}`, expIs: margin.ErrInvalidMarginType}, // codespell:ignore ot
	} {
		err := ex.processUTAOrder(t.Context(), utaMessage(t, `{"T":"orderAll.UNIFIED","P":1770380108536802787,"d":`+tc.data+`}`))
		assert.ErrorIsf(t, err, tc.expIs, "processUTAOrder should error for %s", tc.name)
	}
	assert.Error(t, ex.processUTAOrder(t.Context(), utaMessage(t, `{"T":"orderAll.UNIFIED","P":1770380108536802787,"d":[]}`)), "processUTAOrder should error for malformed data")
}

func TestUTATradeTypeAsset(t *testing.T) {
	t.Parallel()
	for tradeType, exp := range map[string]asset.Item{"SPOT": asset.Spot, "MARGIN": asset.Margin, "FUTURES": asset.Futures} {
		a, err := utaTradeTypeAsset(tradeType)
		require.NoErrorf(t, err, "utaTradeTypeAsset must not error for %s", tradeType)
		assert.Equalf(t, exp, a, "utaTradeTypeAsset should return the asset %s names", tradeType)
	}
	_, err := utaTradeTypeAsset("UNIFIED")
	assert.ErrorIs(t, err, asset.ErrNotSupported, "utaTradeTypeAsset should error for a trade type naming no asset")
}

func TestUTASymbolPair(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	pair, err := ex.utaSymbolPair("BTC-USDT", asset.Margin)
	require.NoError(t, err, "utaSymbolPair must not error for a margin symbol")
	assert.Equal(t, utaSpotTestPair, pair, "utaSymbolPair should split a margin symbol")
	pair, err = ex.utaSymbolPair("XBTUSDTM", asset.Futures)
	require.NoError(t, err, "utaSymbolPair must not error for a futures symbol")
	assert.Equal(t, utaFuturesTestPair, pair, "utaSymbolPair should match a futures symbol")
	_, err = ex.utaSymbolPair("NOTACONTRACTM", asset.Futures)
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "utaSymbolPair should error for an unknown futures contract")
	_, err = ex.utaSymbolPair("", asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCreatingPair, "utaSymbolPair should error for an empty spot symbol")
}

func TestUTAOrderStatus(t *testing.T) {
	t.Parallel()
	for status, exp := range map[uint64]order.Status{
		0: order.Pending,
		1: order.Closed,
		2: order.Open,
		3: order.Filled,
		4: order.PartiallyFilled,
		5: order.Cancelled,
		6: order.PartiallyFilledCancelled,
	} {
		got, err := utaOrderStatus(status)
		require.NoErrorf(t, err, "utaOrderStatus must not error for %d", status)
		assert.Equalf(t, exp, got, "utaOrderStatus should convert %d", status)
	}
	_, err := utaOrderStatus(7)
	assert.ErrorIs(t, err, order.ErrUnsupportedStatusType, "utaOrderStatus should error for an unknown status")
}

func TestUTATimeInForce(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		timeInForce string
		postOnly    bool
		exp         order.TimeInForce
	}{
		{timeInForce: "GTC", exp: order.GoodTillCancel},
		{timeInForce: "GTC", postOnly: true, exp: order.GoodTillCancel | order.PostOnly},
		{timeInForce: "IOC", exp: order.ImmediateOrCancel},
		{timeInForce: "FOK", exp: order.FillOrKill},
		{timeInForce: "GTT", exp: order.GoodTillTime},
		{timeInForce: "RPI", exp: order.UnknownTIF},
		{timeInForce: "", postOnly: true, exp: order.PostOnly},
	} {
		got, err := utaTimeInForce(tc.timeInForce, tc.postOnly)
		require.NoErrorf(t, err, "utaTimeInForce must not error for %q", tc.timeInForce)
		assert.Equalf(t, tc.exp, got, "utaTimeInForce should convert %q with post-only %t", tc.timeInForce, tc.postOnly)
	}
	_, err := utaTimeInForce("DAY", false)
	assert.ErrorIs(t, err, order.ErrInvalidTimeInForce, "utaTimeInForce should error for an unknown time in force")
}

func TestUTAWebsocketExecutionDecoding(t *testing.T) {
	t.Parallel()
	exp := []UTAWebsocketExecution{
		{TradeID: "22075878662488064", Symbol: "BTC-USDT", Price: 67396.6, Size: 0.00001483, Side: "BUY", OrderID: "427992448220876800", ClientOrderID: "9b5620d9-ef1c-4ab5-89d1-717b2602d445", OrderType: "MARKET", LiquidityRole: "TAKER", ExecutionTimestamp: types.Time(time.Unix(0, 1774854553397000000))},
		{TradeID: "1928560862264", Symbol: "XBTUSDTM", Price: 67391.6, Size: 1, Side: "BUY", OrderID: "427993476303507456", OrderType: "MARKET", LiquidityRole: "TAKER", ExecutionTimestamp: types.Time(time.Unix(0, 1774854798514000000))},
		{
			TradeID:            "22221471895799808",
			Symbol:             "XRP-USDT",
			Price:              1.01972,
			Size:               0.9806,
			Side:               "BUY",
			OrderID:            "476907831686172672",
			ClientOrderID:      "9b5620d9-ef1c-4ab5-89d1-717b2602d445",
			OrderType:          "MARKET",
			LiquidityRole:      "TAKER",
			ExecutionTimestamp: types.Time(time.Unix(0, 1786516889638000000)),
			Fee:                0.000999937432,
			FeeCurrency:        currency.USDT,
			FillType:           "NORMAL",
		},
		{
			TradeID:            "1780500473345",
			Symbol:             "XBTUSDTM",
			Price:              63962.4799219875,
			Size:               8,
			Side:               "BUY",
			OrderID:            "0",
			LiquidityRole:      "TAKER",
			ExecutionTimestamp: types.Time(time.Unix(0, 1785393975227000000)),
			FeeCurrency:        currency.USDT,
			FillType:           "LIQUID",
			FillPNL:            -12.5398,
		},
	}
	assert.Equal(t, exp, utaFixtureData[UTAWebsocketExecution](t, "testdata/wsUTAExecution.jsonl"), "execution pushes should decode every field, whether the trade ID is quoted or not")
}

func TestProcessUTAExecutionLite(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	newUTAWebsocket(t, ex)
	msgs := utaFixtureMessages(t, "testdata/wsUTAExecution.jsonl")[:2]
	for i, msg := range msgs {
		require.NoErrorf(t, ex.processUTAExecutionLite(msg), "processUTAExecutionLite must not error for fixture line %d while fills are not fed", i+1)
	}
	assert.Empty(t, drainUTARelay(ex), "processUTAExecutionLite should relay nothing while fills are not fed")

	ex.SetFillsFeedStatus(true)
	for i, msg := range msgs {
		require.NoErrorf(t, ex.processUTAExecutionLite(msg), "processUTAExecutionLite must not error for fixture line %d", i+1)
	}
	exp := []any{
		[]fill.Data{{Timestamp: time.Unix(0, 1774854553397000000), Exchange: ex.Name, AssetType: asset.Spot, CurrencyPair: utaSpotTestPair, Side: order.Buy, OrderID: "427992448220876800", ClientOrderID: "9b5620d9-ef1c-4ab5-89d1-717b2602d445", TradeID: "22075878662488064", Price: 67396.6, Amount: 0.00001483}},
		[]fill.Data{{Timestamp: time.Unix(0, 1774854798514000000), Exchange: ex.Name, AssetType: asset.Futures, CurrencyPair: utaFuturesTestPair, Side: order.Buy, OrderID: "427993476303507456", TradeID: "1928560862264", Price: 67391.6, Amount: 1}},
	}
	assert.Equal(t, exp, drainUTARelay(ex), "processUTAExecutionLite should relay each push as a fill")

	err := ex.processUTAExecutionLite(utaMessage(t, `{"T":"execution.lite.UNIFIED","P":1774854798514737265,"d":{"s":"NOTACONTRACTM","S":"BUY"}}`))
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "processUTAExecutionLite should error for an unknown futures contract")
	assert.Error(t, ex.processUTAExecutionLite(utaMessage(t, `{"T":"execution.lite.UNIFIED","P":1774854798514737265,"d":[]}`)), "processUTAExecutionLite should error for malformed data")
}

func TestProcessUTAExecution(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	newUTAWebsocket(t, ex)
	msgs := utaFixtureMessages(t, "testdata/wsUTAExecution.jsonl")[2:]
	executions := utaFixtureData[UTAWebsocketExecution](t, "testdata/wsUTAExecution.jsonl")[2:]
	for i, msg := range msgs {
		require.NoErrorf(t, ex.processUTAExecution(t.Context(), msg), "processUTAExecution must not error for fixture line %d while fills are not fed", i+3)
	}
	assert.Equal(t, []any{&executions[0], &executions[1]}, drainUTARelay(ex), "processUTAExecution should relay each push, with its fee and fill details, while fills are not fed")

	ex.SetFillsFeedStatus(true)
	for i, msg := range msgs {
		require.NoErrorf(t, ex.processUTAExecution(t.Context(), msg), "processUTAExecution must not error for fixture line %d", i+3)
	}
	exp := []any{
		&executions[0],
		[]fill.Data{{Timestamp: time.Unix(0, 1786516889638000000), Exchange: ex.Name, AssetType: asset.Spot, CurrencyPair: currency.NewPairWithDelimiter("XRP", "USDT", currency.DashDelimiter), Side: order.Buy, OrderID: "476907831686172672", ClientOrderID: "9b5620d9-ef1c-4ab5-89d1-717b2602d445", TradeID: "22221471895799808", Price: 1.01972, Amount: 0.9806}},
		&executions[1],
		[]fill.Data{{Timestamp: time.Unix(0, 1785393975227000000), Exchange: ex.Name, AssetType: asset.Futures, CurrencyPair: utaFuturesTestPair, Side: order.Buy, OrderID: "0", TradeID: "1780500473345", Price: 63962.4799219875, Amount: 8}},
	}
	assert.Equal(t, exp, drainUTARelay(ex), "processUTAExecution should relay each push and its fill")

	err := ex.processUTAExecution(t.Context(), utaMessage(t, `{"T":"execution.UNIFIED","P":1786516889648920721,"d":{"s":"XRP-USDT","S":"LONG-SHORT"}}`))
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "processUTAExecution should error for an unknown side")
	assert.Error(t, ex.processUTAExecution(t.Context(), utaMessage(t, `{"T":"execution.UNIFIED","P":1786516889648920721,"d":[]}`)), "processUTAExecution should error for malformed data")
	ex.Websocket.DataHandler = stream.NewRelay(1)
	require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), "filler"), "Send must not error")
	assert.Error(t, ex.processUTAExecution(t.Context(), msgs[0]), "processUTAExecution should return the error relaying a push")
}

func TestUTAFill(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	x := &UTAWebsocketExecutionLite{TradeID: "22075878662488064", Symbol: "BTC-USDT", Price: 67396.6, Size: 0.00001483, Side: "SELL", OrderID: "427992448220876800", ClientOrderID: "9b5620d9-ef1c-4ab5-89d1-717b2602d445", ExecutionTimestamp: types.Time(time.Unix(0, 1774854553397000000))}
	got, err := ex.utaFill(x)
	require.NoError(t, err, "utaFill must not error")
	exp := fill.Data{Timestamp: time.Unix(0, 1774854553397000000), Exchange: ex.Name, AssetType: asset.Spot, CurrencyPair: utaSpotTestPair, Side: order.Sell, OrderID: "427992448220876800", ClientOrderID: "9b5620d9-ef1c-4ab5-89d1-717b2602d445", TradeID: "22075878662488064", Price: 67396.6, Amount: 0.00001483}
	assert.Equal(t, exp, got, "utaFill should report a dashed symbol's fill as spot")

	_, err = ex.utaFill(&UTAWebsocketExecutionLite{Symbol: "NOTACONTRACTM", Side: "BUY"})
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "utaFill should error for an unknown futures contract")
	_, err = ex.utaFill(&UTAWebsocketExecutionLite{Symbol: "BTC-USDT", Side: "UP"})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "utaFill should error for an unknown side")
}

func TestUTAWebsocketBalanceDecoding(t *testing.T) {
	t.Parallel()
	exp := []UTAWebsocketBalance{
		{UpdateTimestamp: types.Time(time.Unix(0, 1770116995058000000)), Currency: currency.BTC, Equity: 0.0000517, Balance: 0.0000517, Available: 0.0000517, CollateralStatus: "1"},
		{
			Sequence:                 2851,
			UpdateTimestamp:          types.Time(time.Unix(0, 1770117071740000000)),
			Currency:                 currency.USDT,
			Equity:                   1203.35813275,
			Balance:                  1200.32433692,
			Available:                969.10453692,
			Hold:                     215.88,
			Liability:                50.2,
			TotalCrossMargin:         235.33,
			CrossPositionMargin:      9.7198,
			CrossOrderMargin:         215.88,
			CrossUnrealisedPNL:       3.03379583,
			IsolatedPositionMargin:   4.12,
			IsolatedOrderMargin:      1.5,
			IsolatedFundingFeeMargin: 0.02,
			IsolatedUnrealisedPNL:    -0.51,
			CollateralStatus:         "2",
		},
	}
	assert.Equal(t, exp, utaFixtureData[UTAWebsocketBalance](t, "testdata/wsUTABalance.jsonl"), "balance pushes should decode every field")
}

func TestProcessUTABalance(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	msgs := utaFixtureMessages(t, "testdata/wsUTABalance.jsonl")
	for i, msg := range msgs {
		require.NoErrorf(t, ex.processUTABalance(t.Context(), msg, "UNIFIED"), "processUTABalance must not error for fixture line %d", i+1)
	}
	btc := accounts.Balance{Currency: currency.BTC, Total: 0.0000517, Free: 0.0000517, UpdatedAt: time.Unix(0, 1770116995058000000)}
	usdt := accounts.Balance{Currency: currency.USDT, Total: 1200.32433692, Hold: 215.88, Free: 969.10453692, Borrowed: 50.2, UpdatedAt: time.Unix(0, 1770117071740000000)}
	exp := []any{
		accounts.SubAccounts{{ID: utaUnifiedAccountType, AssetType: asset.Spot, Balances: accounts.CurrencyBalances{currency.BTC: btc}}},
		accounts.SubAccounts{{ID: utaUnifiedAccountType, AssetType: asset.Spot, Balances: accounts.CurrencyBalances{currency.USDT: usdt}}},
	}
	assert.Equal(t, exp, drainUTARelay(ex), "processUTABalance should save and relay unified account balances under spot in the unified sub-account")
	// The docs' funding account example, which the unified account's subscription never receives
	err := ex.processUTABalance(t.Context(), utaMessage(t, `{"T":"balance.FUNDING","P":1780543242950333125,"d":{"U":"1780543242933","a":"531.42853547","b":"531.42853547","c":"USDT","h":"0"}}`), "FUNDING")
	assert.ErrorIs(t, err, errUTAUnknownTopic, "processUTABalance should reject another account's balance")
	assert.Empty(t, drainUTARelay(ex), "processUTABalance should not relay another account's balance")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	stored, err := ex.Accounts.SubAccounts(creds, asset.Spot)
	require.NoError(t, err, "SubAccounts must not error")
	assert.Equal(t, accounts.SubAccounts{{ID: utaUnifiedAccountType, AssetType: asset.Spot, Balances: accounts.CurrencyBalances{currency.BTC: btc, currency.USDT: usdt}}}, stored, "processUTABalance should save the unified account balances in the unified sub-account")

	classic := marketTestServerExchange(t, map[string]string{"/api/v1/accounts": `{"code":"200000","data":[{"id":"548674591753","currency":"USDT","type":"trade","balance":"26.66759503","available":"20.66759503","holds":"6"}]}`})
	for i, msg := range msgs {
		require.NoErrorf(t, classic.processUTABalance(t.Context(), msg, "UNIFIED"), "processUTABalance must not error for fixture line %d", i+1)
	}
	_, err = classic.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	creds, err = classic.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	held, err := classic.Accounts.CurrencyBalances(creds, asset.Spot)
	require.NoError(t, err, "CurrencyBalances must not error")
	assert.Equal(t, btc.Total, held[currency.BTC].Total, "a Classic spot snapshot should not remove the unified account's BTC")
	for _, b := range []accounts.Balance{btc, usdt} {
		got, err := classic.Accounts.GetBalance(utaUnifiedAccountType, creds, asset.Spot, b.Currency)
		require.NoErrorf(t, err, "the unified %s balance must outlast a Classic spot snapshot", b.Currency)
		assert.Equalf(t, b, got, "a Classic spot snapshot should leave the unified %s balance as pushed", b.Currency)
	}
	got, err := classic.Accounts.GetBalance("", creds, asset.Spot, currency.USDT)
	require.NoError(t, err, "the Classic spot USDT balance must be stored")
	got.UpdatedAt = time.Time{}
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 26.66759503, Hold: 6, Free: 20.66759503}, got, "unified account pushes should leave the Classic spot balance as REST saved it")
	_, err = classic.Accounts.GetBalance("", creds, asset.Spot, currency.BTC)
	assert.ErrorIs(t, err, accounts.ErrNoBalances, "the unified account's balances should not appear among the Classic spot balances")

	assert.Error(t, ex.processUTABalance(t.Context(), utaMessage(t, `{"T":"balance.UNIFIED","P":1770116995060810093,"d":[]}`), "UNIFIED"), "processUTABalance should error for malformed data")
	ex.SkipAuthCheck = false // The mock instance skips credential checks
	ex.SetCredentials(&accounts.Credentials{})
	err = ex.processUTABalance(t.Context(), msgs[0], "UNIFIED")
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "processUTABalance should return the error saving the balance")
}

func TestUTAWebsocketPositionDecoding(t *testing.T) {
	t.Parallel()
	exp := []UTAWebsocketPosition{
		{
			PositionID:            "70000000000000010",
			Symbol:                "TRUMPUSDTM",
			MarginMode:            "CROSS",
			Size:                  -46,
			EntryPrice:            4.8855208333333333333,
			PositionValue:         19.4396,
			PositionMargin:        9.7198,
			MarkPrice:             4.226,
			LiquidationPrice:      5.2913432001,
			BankruptPrice:         5.3580141244,
			Leverage:              2,
			UnrealisedPNL:         3.03379583333333333332,
			RealisedPNL:           -0.20267048333333333332,
			InitialMargin:         9.7198,
			MaintenanceMarginRate: 0.012,
			MaintenanceMargin:     0.2332752,
			RiskRatio:             0.2001,
			ADL:                   0.46,
			UpdateTimestamp:       types.Time(time.Unix(0, 1770117071740000000)),
			CreationTimestamp:     types.Time(time.Unix(0, 1768876206493000000)),
		},
		{
			PositionID:            "70000000000000011",
			Symbol:                "XBTUSDTM",
			MarginMode:            "ISOLATED",
			Size:                  2,
			EntryPrice:            64521.3,
			PositionValue:         129.0602,
			PositionMargin:        25.80852,
			MarkPrice:             64530.1,
			LiquidationPrice:      51875.12,
			BankruptPrice:         51617.04,
			Leverage:              5,
			UnrealisedPNL:         0.0176,
			RealisedPNL:           -0.0774256,
			InitialMargin:         25.80852,
			MaintenanceMarginRate: 0.004,
			MaintenanceMargin:     0.5162408,
			RiskRatio:             0.0199,
			ADL:                   0.12,
			UpdateTimestamp:       types.Time(time.Unix(0, 1770117072840000000)),
			CreationTimestamp:     types.Time(time.Unix(0, 1770117062493000000)),
		},
		{
			PositionID:            "70000000000000011",
			Symbol:                "XBTUSDTM",
			MarginMode:            "ISOLATED",
			MarkPrice:             64528.7,
			Leverage:              5,
			RealisedPNL:           0.0611744,
			MaintenanceMarginRate: 0.004,
			UpdateTimestamp:       types.Time(time.Unix(0, 1770117073940000000)),
			CreationTimestamp:     types.Time(time.Unix(0, 1770117062493000000)),
		},
	}
	assert.Equal(t, exp, utaFixtureData[UTAWebsocketPosition](t, "testdata/wsUTAPosition.jsonl"), "position pushes should decode every field")
}

func TestProcessUTAPosition(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	for i, msg := range utaFixtureMessages(t, "testdata/wsUTAPosition.jsonl") {
		require.NoErrorf(t, ex.processUTAPosition(t.Context(), msg), "processUTAPosition must not error for fixture line %d", i+1)
	}
	trumpPair := currency.NewPairWithDelimiter("TRUMP", "USDTM", currency.UnderscoreDelimiter)
	exp := []any{
		[]futures.Position{{
			Exchange:                     ex.Name,
			Asset:                        asset.Futures,
			Pair:                         trumpPair,
			Underlying:                   trumpPair.Base,
			Leverage:                     decimal.MustFromFloat(2),
			NotionalSize:                 decimal.MustFromFloat(19.4396),
			PositionMargin:               decimal.MustFromFloat(9.7198),
			InitialMarginRequirement:     decimal.MustFromFloat(9.7198),
			MaintenanceMarginRequirement: decimal.MustFromFloat(0.2332752),
			MaintenanceMarginFraction:    decimal.MustFromFloat(0.012),
			EstimatedLiquidationPrice:    decimal.MustFromFloat(5.2913432001),
			RealisedPNL:                  decimal.MustFromFloat(-0.20267048333333333332),
			UnrealisedPNL:                decimal.MustFromFloat(3.03379583333333333332),
			Status:                       order.Open,
			OpeningDate:                  time.Unix(0, 1768876206493000000),
			OpeningPrice:                 decimal.MustFromFloat(4.8855208333333333333),
			OpeningDirection:             order.Short,
			LatestPrice:                  decimal.MustFromFloat(4.226),
			LatestSize:                   decimal.MustFromFloat(46),
			LatestDirection:              order.Short,
			LastUpdated:                  time.Unix(0, 1770117071740000000),
		}},
		[]futures.Position{{
			Exchange:                     ex.Name,
			Asset:                        asset.Futures,
			Pair:                         utaFuturesTestPair,
			Underlying:                   utaFuturesTestPair.Base,
			Leverage:                     decimal.MustFromFloat(5),
			NotionalSize:                 decimal.MustFromFloat(129.0602),
			PositionMargin:               decimal.MustFromFloat(25.80852),
			InitialMarginRequirement:     decimal.MustFromFloat(25.80852),
			MaintenanceMarginRequirement: decimal.MustFromFloat(0.5162408),
			MaintenanceMarginFraction:    decimal.MustFromFloat(0.004),
			EstimatedLiquidationPrice:    decimal.MustFromFloat(51875.12),
			RealisedPNL:                  decimal.MustFromFloat(-0.0774256),
			UnrealisedPNL:                decimal.MustFromFloat(0.0176),
			Status:                       order.Open,
			OpeningDate:                  time.Unix(0, 1770117062493000000),
			OpeningPrice:                 decimal.MustFromFloat(64521.3),
			OpeningDirection:             order.Long,
			LatestPrice:                  decimal.MustFromFloat(64530.1),
			LatestSize:                   decimal.MustFromFloat(2),
			LatestDirection:              order.Long,
			LastUpdated:                  time.Unix(0, 1770117072840000000),
		}},
		[]futures.Position{{
			Exchange:                     ex.Name,
			Asset:                        asset.Futures,
			Pair:                         utaFuturesTestPair,
			Underlying:                   utaFuturesTestPair.Base,
			Leverage:                     decimal.MustFromFloat(5),
			NotionalSize:                 decimal.MustFromFloat(0),
			PositionMargin:               decimal.MustFromFloat(0),
			InitialMarginRequirement:     decimal.MustFromFloat(0),
			MaintenanceMarginRequirement: decimal.MustFromFloat(0),
			MaintenanceMarginFraction:    decimal.MustFromFloat(0.004),
			EstimatedLiquidationPrice:    decimal.MustFromFloat(0),
			RealisedPNL:                  decimal.MustFromFloat(0.0611744),
			UnrealisedPNL:                decimal.MustFromFloat(0),
			Status:                       order.Closed,
			OpeningDate:                  time.Unix(0, 1770117062493000000),
			OpeningPrice:                 decimal.MustFromFloat(0),
			LatestPrice:                  decimal.MustFromFloat(64528.7),
			LatestSize:                   decimal.MustFromFloat(0),
			LastUpdated:                  time.Unix(0, 1770117073940000000),
			CloseDate:                    time.Unix(0, 1770117073940000000),
		}},
	}
	assert.Equal(t, exp, drainUTARelay(ex), "processUTAPosition should relay each push as a position, closed once its size is zero")

	err := ex.processUTAPosition(t.Context(), utaMessage(t, `{"T":"positionAll.UNIFIED","P":1770117071746459555,"d":{"s":"NOTACONTRACTM"}}`))
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "processUTAPosition should error for an unknown futures contract")
	assert.Error(t, ex.processUTAPosition(t.Context(), utaMessage(t, `{"T":"positionAll.UNIFIED","P":1770117071746459555,"d":[]}`)), "processUTAPosition should error for malformed data")
}

func TestProcessUTALiquidationWarning(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	for i, msg := range utaFixtureMessages(t, "testdata/wsUTALiquidationWarning.jsonl") {
		require.NoErrorf(t, ex.processUTALiquidationWarning(t.Context(), msg), "processUTALiquidationWarning must not error for fixture line %d", i+1)
	}
	exp := []any{
		&UTAWebsocketLiquidationWarning{EventType: "MARGIN_CALL", RiskRatio: 0.9378, AdjustedEquity: 12456.32, InitialMargin: 58720, MaintenanceMargin: 5284.8, AvailableMargin: 1890.45, Equity: 63710.77, Liability: 51254.45, UpdateTimestamp: types.Time(time.Unix(0, 1729842192785100000))},
		&UTAWebsocketLiquidationWarning{EventType: "REDUCE_ONLY", RiskRatio: 0.8612, AdjustedEquity: 12317.08, InitialMargin: 58720, MaintenanceMargin: 5284.8, AvailableMargin: 1751.21, Equity: 63571.53, Liability: 51254.45, UpdateTimestamp: types.Time(time.Unix(0, 1729842252785100000))},
	}
	assert.Equal(t, exp, drainUTARelay(ex), "processUTALiquidationWarning should relay every field of the lw and risk topics' pushes")
	assert.Error(t, ex.processUTALiquidationWarning(t.Context(), utaMessage(t, `{"T":"lw.UNIFIED","P":1729842192785164840,"d":[]}`)), "processUTALiquidationWarning should error for malformed data")
}

func TestProcessUTALeverage(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	for i, msg := range utaFixtureMessages(t, "testdata/wsUTALeverage.jsonl") {
		require.NoErrorf(t, ex.processUTALeverage(t.Context(), msg), "processUTALeverage must not error for fixture line %d", i+1)
	}
	exp := []any{
		&UTAWebsocketLeverage{Symbol: "XBTUSDTM", Leverage: 5, MarginMode: "CROSS", TradeType: "FUTURES"},
		&UTAWebsocketLeverage{Leverage: 5, Currency: currency.BTC, MarginMode: "CROSS", TradeType: "MARGIN"},
	}
	assert.Equal(t, exp, drainUTARelay(ex), "processUTALeverage should relay every field of a futures and a cross margin push")
	assert.Error(t, ex.processUTALeverage(t.Context(), utaMessage(t, `{"T":"leverage.UNIFIED","P":1764570290237940700,"d":[]}`)), "processUTALeverage should error for malformed data")
}
