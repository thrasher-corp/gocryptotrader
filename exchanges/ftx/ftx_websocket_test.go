package ftx

import (
	"testing"
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

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
	f := FTX{
		exchange.Base{
			Websocket: &stream.Websocket{
				DataHandler: make(chan interface{}),
			},
		},
	}

	go func() {
		f.wsHandleData([]byte(input))
	}()

	// Give the channel limited time to yield the data.
	ticker := time.NewTicker(1 * time.Second)
	select {
	case <-ticker.C:
		t.Fatal("timeout")
	case p := <-f.Websocket.DataHandler:
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
}
