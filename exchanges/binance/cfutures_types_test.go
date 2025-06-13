package binance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestFuturesNewOrderRequest_Unmarshal(t *testing.T) {
	const inp = `
{
  "orderId": 18662274680,
  "symbol": "ETHUSD_PERP",
  "pair": "ETHUSD",
  "status": "NEW",
  "clientOrderId": "customID",
  "price": "4096",
  "avgPrice": "2.00",
  "origQty": "8",
  "executedQty": "4",
  "cumQty": "32",
  "cumBase": "16",
  "timeInForce": "GTC",
  "type": "LIMIT",
  "reduceOnly": true,
  "closePosition": true,
  "side": "BUY",
  "positionSide": "BOTH",
  "stopPrice": "2048",
  "workingType": "CONTRACT_PRICE",
  "priceProtect": true,
  "origType": "MARKET",
  "updateTime": 1635931801320,
  "activatePrice": "64",
  "priceRate": "32"
}
`

	var x FuturesOrderPlaceData
	require.NoError(t, json.Unmarshal([]byte(inp), &x))
	exp := FuturesOrderPlaceData{
		OrderID:       18662274680,
		Symbol:        "ETHUSD_PERP",
		Pair:          "ETHUSD",
		Status:        "NEW",
		ClientOrderID: "customID",
		Price:         4096.0,
		AvgPrice:      2.0,
		OrigQty:       8.0,
		ExecuteQty:    4.0,
		CumQty:        32.0,
		CumBase:       16.0,
		TimeInForce:   "GTC",
		OrderType:     cfuturesLimit,
		ReduceOnly:    true,
		ClosePosition: true,
		StopPrice:     2048.0,
		Side:          "BUY",
		PositionSide:  "BOTH",
		WorkingType:   "CONTRACT_PRICE",
		PriceProtect:  true,
		OrigType:      cfuturesMarket,
		UpdateTime:    types.Time(time.UnixMilli(1635931801320)),
		ActivatePrice: 64.0,
		PriceRate:     32.0,
	}
	assert.Equal(t, exp, x)
}
