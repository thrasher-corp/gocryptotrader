package ftx

import (
	"fmt"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

func parseRaw(t *testing.T, input string) interface{} {
	t.Helper()
	pairs := currency.Pairs{
		currency.Pair{
			Base:  currency.BTC,
			Quote: currency.USDT,
		},
	}

	dataC := make(chan interface{}, 1)

	fills := fill.Fills{}
	fills.Setup(true, dataC)

	x := FTX{
		exchange.Base{
			Name: "FTX",
			Features: exchange.Features{
				Enabled: exchange.FeaturesEnabled{
					FillsFeed: true,
				},
			},
			CurrencyPairs: currency.PairsManager{
				Pairs: map[asset.Item]*currency.PairStore{
					asset.Spot: {
						Available: pairs,
						Enabled:   pairs,
						ConfigFormat: &currency.PairFormat{
							Delimiter: "^",
							Uppercase: true,
						},
					},
				},
			},
			Websocket: &stream.Websocket{
				DataHandler: dataC,
				Fills:       fills,
			},
		},
		CollateralWeightHolder{},
	}

	if err := x.wsHandleData([]byte(input)); err != nil {
		t.Fatal(err)
	}

	var ret interface{}
	select {
	case ret = <-x.Websocket.DataHandler:
	default:
		t.Error(fmt.Errorf("timed out waiting for channel data"))
	}

	return ret
}

func TestFTX_wsHandleData_Details(t *testing.T) {
	const inputPartiallyCancelled = `{
            "channel": "orders",
            "type": "update",
            "data": {
                "id": 69350095302,
                "clientId": "192ab87ae99970b79f624ef8bd783351",
                "market": "BTC/USDT",
                "type": "limit",
                "side": "sell",
                "price": 65536,
                "size": 12,
                "status": "closed",
                "filledSize": 4,
                "remainingSize": 8,
                "reduceOnly": false,
                "liquidation": false,
                "avgFillPrice": 32768,
                "postOnly": true,
                "ioc": true,
                "createdAt": "2021-08-08T10:35:02.649437+00:00"
            }
        }`

	p := parseRaw(t, inputPartiallyCancelled)
	x, ok := p.(*order.Detail)
	if !ok {
		t.Fatalf("have %T, want *order.Detail", p)
	}
	// "reduceOnly" and "liquidation" do not have corresponding fields in
	// order.Detail.
	if x.ID != "69350095302" ||
		x.ClientOrderID != "192ab87ae99970b79f624ef8bd783351" ||
		x.Pair.Base.Item.Symbol != "BTC" ||
		x.Pair.Quote.Item.Symbol != "USDT" ||
		x.Type != order.Limit ||
		x.Side != order.Sell ||
		x.Price != 65536 ||
		x.Amount != 12 ||
		x.Status != order.PartiallyCancelled ||
		x.ExecutedAmount != 4 ||
		x.RemainingAmount != 8 ||
		x.AverageExecutedPrice != 32768 ||
		!x.PostOnly ||
		!x.Date.Equal(time.Unix(1628418902, 649437000).UTC()) {
		t.Error("parsed values do not match")
	}

	const inputFilled = `{
            "channel": "orders",
            "type": "update",
            "data": {
                "id": 69350095302,
                "clientId": "192ab87ae99970b79f624ef8bd783351",
                "market": "BTC/USDT",
                "type": "limit",
                "side": "sell",
                "price": 65536,
                "size": 12,
                "status": "closed",
                "filledSize": 12,
                "remainingSize": 0,
                "reduceOnly": false,
                "liquidation": false,
                "avgFillPrice": 32768,
                "postOnly": true,
                "ioc": true,
                "createdAt": "2021-08-08T10:35:02.649437+00:00"
            }
        }`
	if status := parseRaw(t, inputFilled).(*order.Detail).Status; status != order.Filled {
		t.Errorf("have %s, want %s", status, order.Filled)
	}

	const inputCancelled = `{
            "channel": "orders",
            "type": "update",
            "data": {
                "id": 69350095302,
                "clientId": "192ab87ae99970b79f624ef8bd783351",
                "market": "BTC/USDT",
                "type": "limit",
                "side": "sell",
                "price": 65536,
                "size": 12,
                "status": "closed",
                "filledSize": 0,
                "remainingSize": 12,
                "reduceOnly": false,
                "liquidation": false,
                "avgFillPrice": 32768,
                "postOnly": true,
                "ioc": true,
                "createdAt": "2021-08-08T10:35:02.649437+00:00"
            }
        }`
	if status := parseRaw(t, inputCancelled).(*order.Detail).Status; status != order.Cancelled {
		t.Errorf("have %s, want %s", status, order.Cancelled)
	}
}

func TestFTX_wsHandleData_wsFills(t *testing.T) {
	const input = `{
           "channel": "fills",
           "type": "update",
           "data": {
               "id": 1234567890,
               "market": "BTC-USDT",
               "type": "order",
               "side": "sell",
               "price": 32768,
               "size": 2,
               "orderId": 23456789012,
               "time": "2021-08-07T14:32:42.373010+00:00",
               "tradeId": 3456789012,
               "feeRate": 8,
               "fee": 16,
               "feeCurrency": "FTT",
               "liquidity": "maker"
           }
        }`
	p := parseRaw(t, input)
	x, ok := p.([]fill.Data)
	if !ok {
		t.Fatalf("have %T, want []fill.Data", p)
	}

	if x[0].Exchange != "FTX" ||
		x[0].ID != "1234567890" ||
		x[0].OrderID != "23456789012" ||
		x[0].CurrencyPair.Base.String() != "BTC" ||
		x[0].CurrencyPair.Quote.String() != "USDT" ||
		x[0].Side != order.Sell ||
		x[0].TradeID != "3456789012" ||
		x[0].Price != 32768 ||
		x[0].Amount != 2 ||
		!x[0].Timestamp.Equal(time.Unix(1628346762, 373010000).UTC()) {
		t.Errorf("parsed values do not match, x: %#v", x)
	}
}

func TestFTX_wsHandleData_Price(t *testing.T) {
	const input = `{
		"channel": "ticker", 
		"market": "BTC/USDT", 
		"type": "update", 
		"data": {
			"bid": 16.0, 
			"ask": 32.0, 
			"bidSize": 64.0, 
			"askSize": 128.0, 
			"last": 256.0, 
			"time": 1073741824.0
		}
	}`

	p := parseRaw(t, input)
	x, ok := p.(*ticker.Price)

	if !ok {
		t.Fatalf("have %T, want *ticker.Price", p)
	}

	if x.AssetType != asset.Spot ||
		!x.Pair.Equal(currency.NewPair(currency.BTC, currency.USDT)) ||
		x.Bid != 16 ||
		x.BidSize != 64 ||
		x.Ask != 32 ||
		x.AskSize != 128 ||
		x.Last != 256 ||
		!x.LastUpdated.Equal(time.Unix(1073741824, 0)) {
		t.Error("parsed values do not match")
	}
}

func TestParsingOrders(t *testing.T) {
	t.Parallel()
	data := []byte(`{
		  "channel": "fills",
		  "data": {
			"id": 24852229,
			"clientId": null,
			"market": "XRP-PERP",
			"type": "limit",
			"side": "buy",
			"size": 42353.0,
			"price": 0.2977,
			"reduceOnly": false,
			"ioc": false,
			"postOnly": false,
			"status": "closed",
			"filledSize": 0.0,
			"remainingSize": 0.0,
			"avgFillPrice": 0.2978
		  },
		  "type": "update"
		}`)
	if err := f.wsHandleData(data); err != nil {
		t.Error(err)
	}
}

func TestParsingWSTradesData(t *testing.T) {
	t.Parallel()
	data := []byte(`{
		"channel": "trades",
		"market": "BTC-PERP",
		"type": "update",
		"data": [
			{
				"id": 44200173,
				"price": 9761.0,
				"size": 0.0008,
				"side": "buy",
				"liquidation": false,
				"time": "2020-05-15T01:10:04.369194+00:00"
			}
		]
	}`)
	if err := f.wsHandleData(data); err != nil {
		t.Error(err)
	}
}

func TestParsingWSTickerData(t *testing.T) {
	t.Parallel()
	data := []byte(`{
		"channel": "ticker", 
		"market": "BTC-PERP", 
		"type": "update", 
		"data": {
			"bid": 9760.5, 
			"ask": 9761.0, 
			"bidSize": 3.36, 
			"askSize": 71.8484, 
			"last": 9761.0, 
			"time": 1589505004.4237103
		}
	}`)
	if err := f.wsHandleData(data); err != nil {
		t.Error(err)
	}
}

func TestParsingWSOrdersData(t *testing.T) {
	t.Parallel()
	data := []byte(`{
		"channel": "orders",
		"data": {
		  "id": 24852229,
		  "clientId": null,
		  "market": "BTC-PERP",
		  "type": "limit",
		  "side": "buy",
		  "size": 42353.0,
		  "price": 0.2977,
		  "reduceOnly": false,
		  "ioc": false,
		  "postOnly": false,
		  "status": "closed",
		  "filledSize": 0.0,
		  "remainingSize": 0.0,
		  "avgFillPrice": 0.2978
		},
		"type": "update"
	  }`)
	if err := f.wsHandleData(data); err != nil {
		t.Error(err)
	}
}

func TestParsingMarketsData(t *testing.T) {
	t.Parallel()
	data := []byte(`{"channel": "markets",
	 	"type": "partial",
		"data": {
			"ADA-0626": {
			"name": "ADA-0626",
			"enabled": true,
			"priceIncrement": 5e-06,
			"sizeIncrement": 1.0,
			"type": "future",
			"baseCurrency": null,
			"quoteCurrency": null,
			"restricted": false,
			"underlying": "ADA",
			"future": {
				"name": "ADA-0626",
				"underlying": "ADA",
				"description": "Cardano June 2020 FuturesTracker",
				"type": "future", "expiry": "2020-06-26T003:00:00+00:00", 
				"perpetual": false, 
				"expired": false, 
				"enabled": true, 
				"postOnly": false, 
				"imfFactor": 4e-05, 
				"underlyingDescription": "Cardano", 
				"expiryDescription": "June 2020", 
				"moveStart": null, "positionLimitWeight": 10.0, 
				"group": "quarterly"}}},
		"action": "partial"
	  }`)
	if err := f.wsHandleData(data); err != nil {
		t.Error(err)
	}
}

func TestParsingWSOBData(t *testing.T) {
	data := []byte(`{"channel": "orderbook", "market": "BTC-PERP", "type": "partial", "data": {"time": 1589855831.4606245, "checksum": 225973019, "bids": [[9602.0, 3.2903], [9601.5, 3.11], [9601.0, 2.1356], [9600.5, 3.0991], [9600.0, 8.014], [9599.5, 4.1571], [9599.0, 79.1846], [9598.5, 3.099], [9598.0, 3.985], [9597.5, 3.999], [9597.0, 16.4335], [9596.5, 4.006], [9596.0, 3.2596], [9595.0, 6.334], [9594.0, 3.5685], [9593.0, 14.2717], [9592.5, 0.5], [9591.0, 2.181], [9590.5, 40.4246], [9590.0, 1.0], [9589.0, 1.357], [9588.5, 0.4738], [9587.5, 0.15], [9587.0, 16.811], [9586.5, 1.2], [9586.0, 0.2], [9585.5, 1.0], [9584.5, 0.002], [9584.0, 1.51], [9583.5, 0.01], [9583.0, 1.4], [9582.5, 0.1], [9582.0, 24.7921], [9581.0, 2.087], [9580.5, 2.0], [9580.0, 0.1], [9579.0, 1.1588], [9578.0, 0.9477], [9577.5, 22.216], [9576.0, 0.2], [9574.0, 22.0], [9573.5, 1.0], [9572.0, 0.203], [9570.0, 0.1026], [9565.5, 5.5332], [9565.0, 27.5243], [9563.5, 2.6], [9562.0, 0.0175], [9561.0, 2.0085], [9552.0, 1.6], [9550.5, 27.3399], [9550.0, 0.1046], [9548.0, 0.0175], [9544.0, 4.8197], [9542.5, 26.5754], [9542.0, 0.003], [9541.0, 0.0549], [9540.0, 0.1984], [9537.5, 0.0008], [9535.5, 0.0105], [9535.0, 1.514], [9534.5, 36.5858], [9532.5, 4.7798], [9531.0, 40.6564], [9525.0, 0.001], [9523.5, 1.6], [9522.0, 0.0894], [9521.0, 0.315], [9520.5, 5.4525], [9520.0, 0.07], [9518.0, 0.034], [9517.5, 4.0], [9513.0, 0.0175], [9512.5, 15.6016], [9512.0, 32.7882], [9511.5, 0.0482], [9510.5, 0.0482], [9510.0, 0.2999], [9509.0, 2.0], [9508.5, 0.0482], [9506.0, 0.0416], [9505.5, 0.0492], [9505.0, 0.2], [9502.5, 0.01], [9502.0, 0.01], [9501.5, 0.0592], [9501.0, 0.001], [9500.0, 3.4913], [9499.5, 39.8683], [9498.0, 4.6108], [9497.0, 0.0481], [9492.0, 41.3559], [9490.0, 1.1104], [9488.0, 0.0105], [9486.0, 5.4443], [9485.5, 0.0482], [9484.0, 4.0], [9482.0, 0.25], [9481.5, 2.0], [9481.0, 8.1572]], "asks": [[9602.5, 3.0], [9603.0, 2.8979], [9603.5, 54.49], [9604.0, 5.9982], [9604.5, 3.028], [9605.0, 4.657], [9606.5, 5.2512], [9607.0, 4.003], [9607.5, 4.011], [9608.0, 13.7505], [9608.5, 3.994], [9609.0, 2.974], [9609.5, 3.002], [9612.0, 10.298], [9612.5, 13.455], [9613.5, 3.013], [9614.0, 2.02], [9614.5, 3.359], [9615.0, 21.2429], [9616.0, 0.5], [9616.5, 0.01], [9617.0, 2.182], [9617.5, 23.0223], [9618.0, 0.0623], [9618.5, 1.5795], [9619.0, 0.3065], [9620.0, 3.9], [9621.0, 1.5], [9622.0, 1.5], [9622.5, 1.216], [9625.0, 1.0], [9625.5, 0.9477], [9626.0, 0.05], [9628.5, 1.1588], [9629.0, 1.4], [9630.0, 4.2332], [9630.5, 1.228], [9631.0, 1.5], [9631.5, 0.0104], [9632.5, 26.7529], [9633.0, 0.25], [9638.0, 1.0], [9640.0, 0.2], [9641.0, 1.001], [9642.0, 0.0175], [9643.0, 0.25], [9643.5, 1.6], [9644.0, 31.4166], [9646.5, 41.6609], [9649.5, 0.2], [9653.5, 1.5], [9656.5, 1.6], [9657.0, 0.2], [9658.0, 1.5], [9659.5, 4.7804], [9660.5, 43.3405], [9665.5, 40.6564], [9670.0, 0.1034], [9671.5, 4.9098], [9674.0, 0.25], [9678.0, 15.6016], [9678.5, 1.5], [9681.0, 34.9683], [9683.0, 0.2], [9683.5, 5.3845], [9684.5, 5.087], [9685.0, 0.1032], [9686.5, 0.0075], [9689.0, 1.6], [9691.0, 34.7472], [9692.0, 0.001], [9694.0, 0.5], [9695.0, 0.0109], [9696.5, 4.825], [9700.0, 1.0595], [9701.5, 2.0], [9702.0, 0.011], [9702.5, 0.01], [9706.0, 1.2], [9708.0, 0.0175], [9710.0, 39.153], [9712.0, 48.6163], [9712.5, 1.5], [9713.0, 8.1572], [9715.5, 0.5021], [9716.5, 2.0], [9719.0, 0.0245], [9721.0, 0.5], [9724.0, 0.251], [9726.0, 0.12], [9727.5, 0.5075], [9730.0, 0.015], [9732.0, 58.5394], [9733.0, 0.001], [9734.0, 20.0], [9743.0, 0.06], [9750.0, 9.5], [9755.0, 52.4404], [9757.0, 48.6121], [9764.0, 0.015]], "action": "partial"}}`)
	err := f.wsHandleData(data)
	if err != nil {
		t.Error(err)
	}
	data = []byte(`{"channel": "orderbook", "market": "BTC-PERP", "type": "update", "data": {"time": 1589855831.5128105, "checksum": 365946911, "bids": [[9596.0, 4.2656], [9512.0, 32.7912]], "asks": [[9613.5, 4.012], [9702.0, 0.021]], "action": "update"}}`)
	err = f.wsHandleData(data)
	if err != nil {
		t.Error(err)
	}
}

func TestParsingWSOBData2(t *testing.T) {
	t.Parallel()
	data := []byte(`{"channel": "orderbook", "market": "PRIVBEAR/USD", "type": "partial", "data": {"time": 1593498757.0915809, "checksum": 87356415, "bids": [[1389.5, 5.1019], [1384.5, 16.6318], [1371.5, 23.5531], [1365.5, 23.3001], [1354.0, 26.758], [1352.5, 24.6891], [1337.5, 30.3091], [1333.5, 24.9583], [1323.0, 30.9597], [1302.0, 40.9241], [1282.5, 38.0319], [1272.5, 39.1436], [1084.5, 1.8934], [1080.0, 2.0595], [1075.0, 2.0527], [1069.0, 1.8077], [1053.5, 1.855], [1.0, 2.0]], "asks": [[1403.5, 6.8077], [1407.5, 17.6482], [1417.0, 14.6401], [1418.5, 22.6664], [1426.0, 20.3936], [1430.5, 34.2797], [1435.0, 30.6073], [1443.0, 20.2036], [1471.5, 35.5789], [1494.5, 29.2815], [1505.0, 30.9842], [1511.5, 39.4325], [1799.5, 1.7529], [1810.5, 2.0379], [1813.5, 2.0423], [1817.5, 2.0393], [1821.0, 1.7148], [86347.5, 9e-05], [94982.5, 0.0001], [104480.0, 0.0001], [114930.0, 0.00011], [126420.0, 0.00011], [139065.0, 0.00011], [152970.0, 0.00012], [168267.5, 0.00012], [185092.5, 0.00012], [223962.5, 0.00013], [246360.0, 0.00014], [270995.0, 0.00017], [1203602.5, 0.00013]], "action": "partial"}}`)
	err := f.wsHandleData(data)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(`{"channel": "orderbook", "market": "DOGE-PERP", "type": "partial", "data": {"time": 1593395710.072698, "checksum": 2591057682, "bids": [[0.0023085, 507742.0], [0.002308, 7000.0], [0.0023075, 100000.0], [0.0023065, 324770.0], [0.002305, 46000.0], [0.0023035, 879600.0], [0.002303, 49000.0], [0.0023025, 1076421.0], [0.002296, 30511800.0], [0.002293, 3006300.0], [0.0022925, 1256349.0], [0.0022895, 11855700.0], [0.0022855, 1008960.0], [0.0022775, 1047578.0], [0.0022745, 3070200.0], [0.00227, 2939100.0], [0.002269, 1599711.0], [0.00226, 1671504.0], [0.00225, 1957119.0], [0.00224, 5225404.0], [0.0022395, 250.0], [0.002233, 2994000.0], [0.002229, 2336857.0], [0.002218, 2144227.0], [0.002205, 2101662.0], [0.0021985, 7406099.0], [0.0021915, 2470187.0], [0.0021775, 2690545.0], [0.0021755, 250.0], [0.002162, 2997201.0], [0.00215, 11464856.0], [0.002148, 16178857.0], [0.0021255, 11063510.0], [0.002119, 164239.0], [0.0020435, 19124572.0], [0.0020395, 18376430.0], [0.0020125, 1250.0], [0.0019655, 50.0], [0.001958, 97012.0], [0.001942, 50000.0], [0.001899, 50000.0], [0.001895, 1250.0], [0.001712, 2500.0], [0.0012075, 70190.0], [0.00112, 22321.0], [1.65e-05, 31889.0]], "asks": [[0.0023145, 359557.0], [0.0023155, 222497.0], [0.0023175, 40000.0], [0.002319, 879600.0], [0.0023195, 50000.0], [0.0023205, 1067334.0], [0.0023215, 45000.0], [0.002326, 33518100.0], [0.0023265, 1113997.0], [0.0023285, 1170756.0], [0.002331, 11855700.0], [0.002336, 1105442.0], [0.002344, 1244804.0], [0.002348, 3070200.0], [0.0023525, 1546561.0], [0.0023555, 2939100.0], [0.0023575, 2928000.0], [0.002362, 1509707.0], [0.0023725, 1786697.0], [0.002374, 5710.0], [0.0023795, 151098.0], [0.0023835, 1747428.0], [0.002385, 2994000.0], [0.002395, 1721532.0], [0.0024015, 5710.0], [0.002408, 2552142.0], [0.002422, 2188855.0], [0.002429, 5710.0], [0.0024295, 8441953.0], [0.002437, 2196750.0], [0.002445, 122574.0], [0.002454, 1974273.0], [0.0024565, 5710.0], [0.0024715, 2864643.0], [0.00248, 15238408.0], [0.002484, 5710.0], [0.002497, 16343646.0], [0.0025025, 12177084.0], [0.0025115, 5710.0], [0.002539, 5710.0], [0.002566, 16643688.0], [0.0025665, 5710.0], [0.002594, 5710.0], [0.002617, 50.0], [0.002623, 10.0], [0.0027685, 20825893.0], [0.003178, 50000.0], [0.003811, 68952.0], [0.0074, 41460.0]], "action": "partial"}}`)
	err = f.wsHandleData(data)
	if err != nil {
		t.Error(err)
	}
	data = []byte(`{"channel": "orderbook", "market": "BTC-PERP", "type": "partial", "data": {"time": 1589855831.4606245, "checksum": 225973019, "bids": [[9602.0, 3.2903], [9601.5, 3.11], [9601.0, 2.1356], [9600.5, 3.0991], [9600.0, 8.014], [9599.5, 4.1571], [9599.0, 79.1846], [9598.5, 3.099], [9598.0, 3.985], [9597.5, 3.999], [9597.0, 16.4335], [9596.5, 4.006], [9596.0, 3.2596], [9595.0, 6.334], [9594.0, 3.5685], [9593.0, 14.2717], [9592.5, 0.5], [9591.0, 2.181], [9590.5, 40.4246], [9590.0, 1.0], [9589.0, 1.357], [9588.5, 0.4738], [9587.5, 0.15], [9587.0, 16.811], [9586.5, 1.2], [9586.0, 0.2], [9585.5, 1.0], [9584.5, 0.002], [9584.0, 1.51], [9583.5, 0.01], [9583.0, 1.4], [9582.5, 0.1], [9582.0, 24.7921], [9581.0, 2.087], [9580.5, 2.0], [9580.0, 0.1], [9579.0, 1.1588], [9578.0, 0.9477], [9577.5, 22.216], [9576.0, 0.2], [9574.0, 22.0], [9573.5, 1.0], [9572.0, 0.203], [9570.0, 0.1026], [9565.5, 5.5332], [9565.0, 27.5243], [9563.5, 2.6], [9562.0, 0.0175], [9561.0, 2.0085], [9552.0, 1.6], [9550.5, 27.3399], [9550.0, 0.1046], [9548.0, 0.0175], [9544.0, 4.8197], [9542.5, 26.5754], [9542.0, 0.003], [9541.0, 0.0549], [9540.0, 0.1984], [9537.5, 0.0008], [9535.5, 0.0105], [9535.0, 1.514], [9534.5, 36.5858], [9532.5, 4.7798], [9531.0, 40.6564], [9525.0, 0.001], [9523.5, 1.6], [9522.0, 0.0894], [9521.0, 0.315], [9520.5, 5.4525], [9520.0, 0.07], [9518.0, 0.034], [9517.5, 4.0], [9513.0, 0.0175], [9512.5, 15.6016], [9512.0, 32.7882], [9511.5, 0.0482], [9510.5, 0.0482], [9510.0, 0.2999], [9509.0, 2.0], [9508.5, 0.0482], [9506.0, 0.0416], [9505.5, 0.0492], [9505.0, 0.2], [9502.5, 0.01], [9502.0, 0.01], [9501.5, 0.0592], [9501.0, 0.001], [9500.0, 3.4913], [9499.5, 39.8683], [9498.0, 4.6108], [9497.0, 0.0481], [9492.0, 41.3559], [9490.0, 1.1104], [9488.0, 0.0105], [9486.0, 5.4443], [9485.5, 0.0482], [9484.0, 4.0], [9482.0, 0.25], [9481.5, 2.0], [9481.0, 8.1572]], "asks": [[9602.5, 3.0], [9603.0, 2.8979], [9603.5, 54.49], [9604.0, 5.9982], [9604.5, 3.028], [9605.0, 4.657], [9606.5, 5.2512], [9607.0, 4.003], [9607.5, 4.011], [9608.0, 13.7505], [9608.5, 3.994], [9609.0, 2.974], [9609.5, 3.002], [9612.0, 10.298], [9612.5, 13.455], [9613.5, 3.013], [9614.0, 2.02], [9614.5, 3.359], [9615.0, 21.2429], [9616.0, 0.5], [9616.5, 0.01], [9617.0, 2.182], [9617.5, 23.0223], [9618.0, 0.0623], [9618.5, 1.5795], [9619.0, 0.3065], [9620.0, 3.9], [9621.0, 1.5], [9622.0, 1.5], [9622.5, 1.216], [9625.0, 1.0], [9625.5, 0.9477], [9626.0, 0.05], [9628.5, 1.1588], [9629.0, 1.4], [9630.0, 4.2332], [9630.5, 1.228], [9631.0, 1.5], [9631.5, 0.0104], [9632.5, 26.7529], [9633.0, 0.25], [9638.0, 1.0], [9640.0, 0.2], [9641.0, 1.001], [9642.0, 0.0175], [9643.0, 0.25], [9643.5, 1.6], [9644.0, 31.4166], [9646.5, 41.6609], [9649.5, 0.2], [9653.5, 1.5], [9656.5, 1.6], [9657.0, 0.2], [9658.0, 1.5], [9659.5, 4.7804], [9660.5, 43.3405], [9665.5, 40.6564], [9670.0, 0.1034], [9671.5, 4.9098], [9674.0, 0.25], [9678.0, 15.6016], [9678.5, 1.5], [9681.0, 34.9683], [9683.0, 0.2], [9683.5, 5.3845], [9684.5, 5.087], [9685.0, 0.1032], [9686.5, 0.0075], [9689.0, 1.6], [9691.0, 34.7472], [9692.0, 0.001], [9694.0, 0.5], [9695.0, 0.0109], [9696.5, 4.825], [9700.0, 1.0595], [9701.5, 2.0], [9702.0, 0.011], [9702.5, 0.01], [9706.0, 1.2], [9708.0, 0.0175], [9710.0, 39.153], [9712.0, 48.6163], [9712.5, 1.5], [9713.0, 8.1572], [9715.5, 0.5021], [9716.5, 2.0], [9719.0, 0.0245], [9721.0, 0.5], [9724.0, 0.251], [9726.0, 0.12], [9727.5, 0.5075], [9730.0, 0.015], [9732.0, 58.5394], [9733.0, 0.001], [9734.0, 20.0], [9743.0, 0.06], [9750.0, 9.5], [9755.0, 52.4404], [9757.0, 48.6121], [9764.0, 0.015]], "action": "partial"}}`)
	err = f.wsHandleData(data)
	if err != nil {
		t.Error(err)
	}
	data = []byte(`{"channel": "orderbook", "market": "BTC-PERP", "type": "update", "data": {"time": 1589855831.5128105, "checksum": 365946911, "bids": [[9596.0, 4.2656], [9512.0, 32.7912]], "asks": [[9613.5, 4.012], [9702.0, 0.021]], "action": "update"}}`)
	err = f.wsHandleData(data)
	if err != nil {
		t.Error(err)
	}
}
