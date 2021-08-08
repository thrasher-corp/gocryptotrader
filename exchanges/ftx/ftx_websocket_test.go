package ftx

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

func parseRaw(t *testing.T, input string) interface{} {
	t.Helper()
	pairs := currency.Pairs{
		currency.Pair{
			Base:  currency.NewCode("BTC"),
			Quote: currency.NewCode("USDT"),
		},
	}
	f := FTX{
		exchange.Base{
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
				DataHandler: make(chan interface{}),
			},
		},
	}
	go func() {
		f.wsHandleData([]byte(input))
	}()
	// Give the channel limited time to yield the data.
	ticker := time.NewTicker(100 * time.Second)
	select {
	case <-ticker.C:
		t.Fatal("timeout")
	case p := <-f.Websocket.DataHandler:
		return p
	}
	return nil
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
		t.Fatalf("have %T, want order.Detail", p)
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
		//
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
               "market": "MARKET",
               "future": "FUTURE",
               "baseCurrency": "BTC",
               "quoteCurrency": "USDT",
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
	x, ok := p.(WsFills)
	if !ok {
		t.Fatalf("have %T, want ftx.WsFills", p)
	}
	if x.ID != 1234567890 ||
		x.Market != "MARKET" ||
		x.Future != "FUTURE" ||
		x.BaseCurrency != "BTC" ||
		x.QuoteCurrency != "USDT" ||
		x.Type != "order" ||
		x.Side != "sell" ||
		x.Price != 32768 ||
		x.Size != 2 ||
		x.OrderID != 23456789012 ||
		!x.Time.Equal(time.Unix(1628346762, 373010000).UTC()) ||
		x.TradeID != 3456789012 ||
		x.FeeRate != 8 ||
		x.Fee != 16 ||
		x.FeeCurrency != "FTT" ||
		x.Liquidity != "maker" {
		//
		t.Error("parsed values do not match")
	}
}
