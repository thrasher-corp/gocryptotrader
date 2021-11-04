package binance

import (
	"encoding/json"
	"testing"
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

	if err := json.Unmarshal([]byte(inp), &x); err != nil {
		t.Error(err)
	}

	if x.OrderID != 18662274680 ||
		x.Symbol != "ETHUSD_PERP" ||
		x.Pair != "ETHUSD" ||
		x.Status != "NEW" ||
		x.ClientOrderID != "customID" ||
		x.Price != 4096 ||
		x.AvgPrice != 2 ||
		x.OrigQty != 8 ||
		x.ExecuteQty != 4 ||
		x.CumQty != 32 ||
		x.CumBase != 16 ||
		x.TimeInForce != "GTC" ||
		x.OrderType != cfuturesLimit ||
		!x.ReduceOnly ||
		!x.ClosePosition ||
		x.StopPrice != 2048 ||
		x.WorkingType != "CONTRACT_PRICE" ||
		!x.PriceProtect ||
		x.OrigType != cfuturesMarket ||
		x.UpdateTime != 1635931801320 ||
		x.ActivatePrice != 64 ||
		x.PriceRate != 32 {
		// If any of these values isn't set as expected, mark test as failed.
		t.Errorf("unmarshaling failed: %v", x)
	}
}
