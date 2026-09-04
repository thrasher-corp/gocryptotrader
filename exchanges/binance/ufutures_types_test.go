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

func TestUAccountTradeHistoryUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `
{
  "buyer": true,
  "commission": "-0.078125",
  "commissionAsset": "USDT",
  "id": 698759,
  "maker": true,
  "orderId": 25851813,
  "price": "7819.25",
  "qty": "0.0625",
  "baseQty": "0.03125",
  "quoteQty": "15.640625",
  "realizedPnl": "-0.9375",
  "side": "SELL",
  "positionSide": "SHORT",
  "marginAsset": "USDT",
  "symbol": "BTCUSDT",
  "pair": "BTCUSDT",
  "time": 1569514978020
}
`

	var x UAccountTradeHistory
	require.NoError(t, json.Unmarshal([]byte(inp), &x), "Unmarshal must not error")
	exp := UAccountTradeHistory{
		Buyer:           true,
		Commission:      -0.078125,
		CommissionAsset: currency.USDT,
		ID:              698759,
		Maker:           true,
		OrderID:         25851813,
		Price:           7819.25,
		Quantity:        0.0625,
		BaseQuantity:    0.03125,
		QuoteQuantity:   15.640625,
		RealizedPNL:     -0.9375,
		Side:            "SELL",
		MarginAsset:     currency.USDT,
		PositionSide:    "SHORT",
		Symbol:          "BTCUSDT",
		Pair:            "BTCUSDT",
		Time:            types.Time(time.UnixMilli(1569514978020)),
	}
	assert.Equal(t, exp, x, "UAccountTradeHistory should unmarshal correctly")
}

func TestUPublicTradesDataUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `
{
  "id": 8027591209,
  "price": "79673.50",
  "qty": "0.0625",
  "quoteQty": "318.6875",
  "time": 1787893510850,
  "isBuyerMaker": true,
  "isRPITrade": true
}
`

	var x UPublicTradesData
	require.NoError(t, json.Unmarshal([]byte(inp), &x), "Unmarshal must not error")
	exp := UPublicTradesData{
		ID:            8027591209,
		Price:         79673.50,
		Quantity:      0.0625,
		QuoteQuantity: 318.6875,
		Time:          types.Time(time.UnixMilli(1787893510850)),
		IsBuyerMaker:  true,
		IsRPITrade:    true,
	}
	assert.Equal(t, exp, x, "UPublicTradesData should unmarshal correctly")

	const coinMargined = `{"id":1150414473,"price":"79822.4","qty":"2","baseQty":"0.00250556","time":1787898309227,"isBuyerMaker":true}`
	var cm UPublicTradesData
	require.NoError(t, json.Unmarshal([]byte(coinMargined), &cm), "Unmarshal must not error for coin margined futures")
	expCM := UPublicTradesData{
		ID:           1150414473,
		Price:        79822.4,
		Quantity:     2,
		BaseQuantity: 0.00250556,
		Time:         types.Time(time.UnixMilli(1787898309227)),
		IsBuyerMaker: true,
	}
	assert.Equal(t, expCM, cm, "UPublicTradesData should unmarshal the coin margined shape correctly")
}

func TestUFuturesOrderDataUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `
{
  "avgPrice": "4096.5",
  "clientOrderId": "customID",
  "cumQuote": "16.5",
  "cumBase": "8.25",
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
  "symbol": "BTCUSDT",
  "pair": "BTCUSDT",
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

	var x UFuturesOrderData
	require.NoError(t, json.Unmarshal([]byte(inp), &x), "Unmarshal must not error")
	exp := UFuturesOrderData{
		AveragePrice:            4096.5,
		ClientOrderID:           "customID",
		CumulativeQuote:         16.5,
		CumulativeBase:          8.25,
		ExecutedQuantity:        4.125,
		OrderID:                 1573346959,
		OriginalQuantity:        32.75,
		OriginalType:            "MARKET",
		Price:                   2048.25,
		ReduceOnly:              true,
		Side:                    "BUY",
		PositionSide:            "SHORT",
		Status:                  "NEW",
		StopPrice:               1024.75,
		ClosePosition:           true,
		Symbol:                  "BTCUSDT",
		Pair:                    "BTCUSDT",
		Time:                    types.Time(time.UnixMilli(1579276756075)),
		TimeInForce:             "GTC",
		OrderType:               "LIMIT",
		ActivatePrice:           512.5,
		PriceRate:               0.35,
		UpdateTime:              types.Time(time.UnixMilli(1635931801320)),
		WorkingType:             "CONTRACT_PRICE",
		PriceProtect:            true,
		PriceMatch:              "OPPONENT",
		SelfTradePreventionMode: "EXPIRE_MAKER",
		GoodTillDate:            types.Time(time.UnixMilli(1693207680000)),
	}
	assert.Equal(t, exp, x, "UFuturesOrderData should unmarshal correctly")
}
