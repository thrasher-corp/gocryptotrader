package binance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestFuturesOrderPlaceDataUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `
{
  "orderId": 18662274680,
  "symbol": "ETHUSD_PERP",
  "pair": "ETHUSD",
  "status": "NEW",
  "clientOrderId": "customID",
  "price": "4096.5",
  "origQty": "8.25",
  "executedQty": "4.125",
  "cumQty": "32.75",
  "timeInForce": "GTC",
  "type": "LIMIT",
  "reduceOnly": true,
  "closePosition": true,
  "side": "BUY",
  "positionSide": "BOTH",
  "stopPrice": "2048.5",
  "workingType": "CONTRACT_PRICE",
  "priceProtect": true,
  "origType": "MARKET",
  "updateTime": 1635931801320,
  "activatePrice": "64.25",
  "priceRate": "32.5",
  "priceMatch": "OPPONENT",
  "selfTradePreventionMode": "EXPIRE_MAKER"
}
`

	var x FuturesOrderPlaceData
	require.NoError(t, json.Unmarshal([]byte(inp), &x), "Unmarshal must not error")
	exp := FuturesOrderPlaceData{
		OrderID:                 18662274680,
		Symbol:                  "ETHUSD_PERP",
		Pair:                    "ETHUSD",
		Status:                  "NEW",
		ClientOrderID:           "customID",
		Price:                   4096.5,
		OriginalQuantity:        8.25,
		ExecutedQuantity:        4.125,
		CumulativeQuantity:      32.75,
		TimeInForce:             "GTC",
		OrderType:               cfuturesLimit,
		ReduceOnly:              true,
		ClosePosition:           true,
		StopPrice:               2048.5,
		Side:                    "BUY",
		PositionSide:            "BOTH",
		WorkingType:             "CONTRACT_PRICE",
		PriceProtect:            true,
		OriginalType:            cfuturesMarket,
		UpdateTime:              types.Time(time.UnixMilli(1635931801320)),
		ActivatePrice:           64.25,
		PriceRate:               32.5,
		PriceMatch:              "OPPONENT",
		SelfTradePreventionMode: "EXPIRE_MAKER",
	}
	assert.Equal(t, exp, x, "FuturesOrderPlaceData should unmarshal correctly")

	var rejected FuturesOrderPlaceData
	require.NoError(t, json.Unmarshal([]byte(`{"code":-2022,"msg":"ReduceOnly Order is rejected."}`), &rejected), "Unmarshal must not error for a rejected batch entry")
	assert.Equal(t, FuturesOrderPlaceData{Code: -2022, Message: "ReduceOnly Order is rejected."}, rejected, "a rejected batch entry should be distinguishable from a zero-valued success")
}

func TestNotionalBracketDataUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `
{
  "pair": "BTCUSD",
  "brackets": [
    {
      "bracket": 1,
      "initialLeverage": 125,
      "qtyCap": 50.5,
      "qtylFloor": 2.25,
      "maintMarginRatio": 0.004,
      "cum": 8.75
    }
  ]
}
`

	var x NotionalBracketData
	require.NoError(t, json.Unmarshal([]byte(inp), &x), "Unmarshal must not error")
	exp := NotionalBracketData{
		Pair: "BTCUSD",
		Brackets: []NotionalBracket{{
			Bracket:          1,
			InitialLeverage:  125,
			QtyCap:           50.5,
			QtylFloor:        2.25,
			MaintMarginRatio: 0.004,
			Cumulative:       8.75,
		}},
	}
	assert.Equal(t, exp, x, "NotionalBracketData should unmarshal correctly")
}

func TestBatchCancelOrderDataUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `
[
  {
    "clientOrderId": "myOrder1",
    "cumQty": "3.5",
    "executedQty": "4.25",
    "orderId": 283194212,
    "origQty": "11.75",
    "price": "9100.5",
    "reduceOnly": true,
    "side": "BUY",
    "positionSide": "BOTH",
    "status": "CANCELED",
    "stopPrice": "9300.25",
    "closePosition": true,
    "symbol": "BTCUSD_200925",
    "pair": "BTCUSD",
    "timeInForce": "GTC",
    "type": "TRAILING_STOP_MARKET",
    "origType": "TRAILING_STOP_MARKET",
    "activatePrice": "9020.125",
    "priceRate": "0.3",
    "updateTime": 1571110484038,
    "workingType": "CONTRACT_PRICE",
    "priceProtect": true,
    "priceMatch": "NONE",
    "selfTradePreventionMode": "EXPIRE_MAKER"
  },
  {
    "code": -2011,
    "msg": "Unknown order sent."
  }
]
`

	var x []BatchCancelOrderData
	require.NoError(t, json.Unmarshal([]byte(inp), &x), "Unmarshal must not error")
	exp := []BatchCancelOrderData{
		{
			ClientOrderID:           "myOrder1",
			CumulativeQuantity:      3.5,
			ExecutedQuantity:        4.25,
			OrderID:                 283194212,
			OriginalQuantity:        11.75,
			Price:                   9100.5,
			ReduceOnly:              true,
			Side:                    "BUY",
			PositionSide:            "BOTH",
			Status:                  "CANCELED",
			StopPrice:               9300.25,
			ClosePosition:           true,
			Symbol:                  "BTCUSD_200925",
			Pair:                    "BTCUSD",
			TimeInForce:             "GTC",
			OrderType:               "TRAILING_STOP_MARKET",
			OriginalType:            "TRAILING_STOP_MARKET",
			ActivatePrice:           9020.125,
			PriceRate:               0.3,
			UpdateTime:              types.Time(time.UnixMilli(1571110484038)),
			WorkingType:             "CONTRACT_PRICE",
			PriceProtect:            true,
			PriceMatch:              "NONE",
			SelfTradePreventionMode: "EXPIRE_MAKER",
		},
		{
			Code:    -2011,
			Message: "Unknown order sent.",
		},
	}
	assert.Equal(t, exp, x, "BatchCancelOrderData should unmarshal correctly")
}

func TestFuturesAccountTradeListUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `
{
  "symbol": "BTCUSD_200626",
  "id": 6,
  "orderId": 28,
  "pair": "BTCUSD",
  "side": "SELL",
  "price": "8800",
  "qty": "1",
  "realizedPnl": "-0.04274782",
  "marginAsset": "BTC",
  "baseQty": "0.01136364",
  "quoteQty": "100",
  "commission": "0.00000454",
  "commissionAsset": "BTC",
  "time": 1590743483586,
  "positionSide": "BOTH",
  "buyer": true,
  "maker": true
}
`

	var x FuturesAccountTradeList
	require.NoError(t, json.Unmarshal([]byte(inp), &x), "Unmarshal must not error")
	exp := FuturesAccountTradeList{
		Symbol:          "BTCUSD_200626",
		ID:              6,
		OrderID:         28,
		Pair:            "BTCUSD",
		Side:            "SELL",
		Price:           8800,
		Quantity:        1,
		RealizedPNL:     -0.04274782,
		MarginAsset:     currency.BTC,
		BaseQuantity:    0.01136364,
		QuoteQuantity:   100,
		Commission:      0.00000454,
		CommissionAsset: currency.BTC,
		Timestamp:       types.Time(time.UnixMilli(1590743483586)),
		PositionSide:    "BOTH",
		Buyer:           true,
		Maker:           true,
	}
	assert.Equal(t, exp, x, "FuturesAccountTradeList should unmarshal correctly")
}

func TestForcedOrdersDataUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `
{
  "orderId": 6071832819,
  "symbol": "BTCUSD_200925",
  "pair": "BTCUSD",
  "status": "FILLED",
  "clientOrderId": "autoclose-1596107620040000020",
  "price": "10871.09",
  "avgPrice": "10913.21000",
  "origQty": "2",
  "executedQty": "1",
  "cumBase": "0.00009159",
  "cumQuote": "99.75",
  "timeInForce": "IOC",
  "type": "LIMIT",
  "reduceOnly": true,
  "closePosition": true,
  "side": "SELL",
  "positionSide": "BOTH",
  "stopPrice": "10800",
  "workingType": "CONTRACT_PRICE",
  "priceProtect": true,
  "origType": "LIMIT",
  "time": 1596107620044,
  "updateTime": 1596107620087,
  "goodTillDate": 1596107620099
}
`

	var x ForcedOrdersData
	require.NoError(t, json.Unmarshal([]byte(inp), &x), "Unmarshal must not error")
	exp := ForcedOrdersData{
		OrderID:          6071832819,
		Symbol:           "BTCUSD_200925",
		Pair:             "BTCUSD",
		Status:           "FILLED",
		ClientOrderID:    "autoclose-1596107620040000020",
		Price:            10871.09,
		AveragePrice:     10913.21,
		OriginalQuantity: 2,
		ExecutedQuantity: 1,
		CumulativeBase:   0.00009159,
		CumulativeQuote:  99.75,
		TimeInForce:      "IOC",
		OrderType:        "LIMIT",
		ReduceOnly:       true,
		ClosePosition:    true,
		Side:             "SELL",
		PositionSide:     "BOTH",
		StopPrice:        10800,
		WorkingType:      "CONTRACT_PRICE",
		PriceProtect:     true,
		OriginalType:     "LIMIT",
		Time:             types.Time(time.UnixMilli(1596107620044)),
		UpdateTime:       types.Time(time.UnixMilli(1596107620087)),
		GoodTillDate:     types.Time(time.UnixMilli(1596107620099)),
	}
	assert.Equal(t, exp, x, "ForcedOrdersData should unmarshal correctly")
}

func TestGetPositionMarginChangeHistoryDataUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `
{
  "amount": "23.36332311",
  "asset": "BTC",
  "symbol": "BTCUSD_200925",
  "time": 1578047897183,
  "type": 1,
  "positionSide": "BOTH"
}
`

	var x GetPositionMarginChangeHistoryData
	require.NoError(t, json.Unmarshal([]byte(inp), &x), "Unmarshal must not error")
	exp := GetPositionMarginChangeHistoryData{
		Amount:           23.36332311,
		Asset:            currency.BTC,
		Symbol:           "BTCUSD_200925",
		Timestamp:        types.Time(time.UnixMilli(1578047897183)),
		MarginChangeType: 1,
		PositionSide:     "BOTH",
	}
	assert.Equal(t, exp, x, "GetPositionMarginChangeHistoryData should unmarshal correctly")
}

func TestFuturesOrderGetDataUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `
{
  "avgPrice": "4096.5",
  "clientOrderId": "customID",
  "cumBase": "8.25",
  "cumQty": "16.5",
  "executedQty": "4.125",
  "orderId": 1573346959,
  "origQty": "32.75",
  "origType": "MARKET",
  "price": "2048.25",
  "reduceOnly": true,
  "side": "BUY",
  "positionSide": "SHORT",
  "status": "NEW",
  "stopPrice": "1024.75",
  "closePosition": true,
  "symbol": "BTCUSD_200925",
  "pair": "BTCUSD",
  "time": 1579276756075,
  "timeInForce": "GTC",
  "type": "LIMIT",
  "activatePrice": "512.5",
  "priceRate": "0.35",
  "updateTime": 1635931801320,
  "workingType": "CONTRACT_PRICE",
  "priceProtect": true,
  "priceMatch": "OPPONENT",
  "selfTradePreventionMode": "EXPIRE_MAKER"
}
`

	var x FuturesOrderGetData
	require.NoError(t, json.Unmarshal([]byte(inp), &x), "Unmarshal must not error")
	exp := FuturesOrderGetData{
		AveragePrice:            4096.5,
		ClientOrderID:           "customID",
		CumulativeBase:          8.25,
		CumulativeQuantity:      16.5,
		ExecutedQuantity:        4.125,
		OrderID:                 1573346959,
		OriginalQuantity:        32.75,
		OriginalType:            cfuturesMarket,
		Price:                   2048.25,
		ReduceOnly:              true,
		Side:                    "BUY",
		PositionSide:            "SHORT",
		Status:                  "NEW",
		StopPrice:               1024.75,
		ClosePosition:           true,
		Symbol:                  "BTCUSD_200925",
		Pair:                    "BTCUSD",
		Time:                    types.Time(time.UnixMilli(1579276756075)),
		TimeInForce:             "GTC",
		OrderType:               cfuturesLimit,
		ActivatePrice:           512.5,
		PriceRate:               0.35,
		UpdateTime:              types.Time(time.UnixMilli(1635931801320)),
		WorkingType:             "CONTRACT_PRICE",
		PriceProtect:            true,
		PriceMatch:              "OPPONENT",
		SelfTradePreventionMode: "EXPIRE_MAKER",
	}
	assert.Equal(t, exp, x, "FuturesOrderGetData should unmarshal correctly")
}

func TestFuturesOrderDataUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `
{
  "avgPrice": "4096.5",
  "clientOrderId": "customID",
  "cumBase": "8.25",
  "cumQuote": "16.5",
  "executedQty": "4.125",
  "orderId": 1573346959,
  "origQty": "32.75",
  "origType": "MARKET",
  "price": "2048.25",
  "reduceOnly": true,
  "side": "BUY",
  "positionSide": "SHORT",
  "status": "NEW",
  "stopPrice": "1024.75",
  "closePosition": true,
  "symbol": "BTCUSD_200925",
  "pair": "BTCUSD",
  "time": 1579276756075,
  "timeInForce": "GTC",
  "type": "LIMIT",
  "activatePrice": "512.5",
  "priceRate": "0.35",
  "updateTime": 1635931801320,
  "workingType": "CONTRACT_PRICE",
  "priceProtect": true,
  "priceMatch": "OPPONENT",
  "selfTradePreventionMode": "EXPIRE_MAKER",
  "goodTillDate": 1693207680000
}
`

	var x FuturesOrderData
	require.NoError(t, json.Unmarshal([]byte(inp), &x), "Unmarshal must not error")
	exp := FuturesOrderData{
		AveragePrice:            4096.5,
		ClientOrderID:           "customID",
		CumulativeBase:          8.25,
		CumulativeQuote:         16.5,
		ExecutedQuantity:        4.125,
		OrderID:                 1573346959,
		OriginalQuantity:        32.75,
		OriginalType:            cfuturesMarket,
		Price:                   2048.25,
		ReduceOnly:              true,
		Side:                    "BUY",
		PositionSide:            "SHORT",
		Status:                  "NEW",
		StopPrice:               1024.75,
		ClosePosition:           true,
		Symbol:                  "BTCUSD_200925",
		Pair:                    "BTCUSD",
		Time:                    types.Time(time.UnixMilli(1579276756075)),
		TimeInForce:             "GTC",
		OrderType:               cfuturesLimit,
		ActivatePrice:           512.5,
		PriceRate:               0.35,
		UpdateTime:              types.Time(time.UnixMilli(1635931801320)),
		WorkingType:             "CONTRACT_PRICE",
		PriceProtect:            true,
		PriceMatch:              "OPPONENT",
		SelfTradePreventionMode: "EXPIRE_MAKER",
		GoodTillDate:            types.Time(time.UnixMilli(1693207680000)),
	}
	assert.Equal(t, exp, x, "FuturesOrderData should unmarshal correctly")
}

func TestFuturesPublicTradesDataUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `{"id":1150414473,"price":"79822.4","qty":"2","baseQty":"0.00250556","time":1787898309227,"isBuyerMaker":true}`
	var x FuturesPublicTradesData
	require.NoError(t, json.Unmarshal([]byte(inp), &x), "Unmarshal must not error")
	exp := FuturesPublicTradesData{
		ID:           1150414473,
		Price:        79822.4,
		Quantity:     2,
		BaseQuantity: 0.00250556,
		Time:         types.Time(time.UnixMilli(1787898309227)),
		IsBuyerMaker: true,
	}
	assert.Equal(t, exp, x, "FuturesPublicTradesData should unmarshal correctly")
}
