package request

import (
	"net/http"
	"sync"
	"testing"
)

var (
	wg         sync.WaitGroup
	bitfinex   *Handler
	BTCMarkets *Handler
)

func TestSetRequestHandler(t *testing.T) {
	bitfinex = new(Handler)
	err := bitfinex.SetRequestHandler("bitfinex", 1000, 1000, new(http.Client))
	if err != nil {
		t.Error("Test failed - request SetRequestHandler()", err)
	}
	err = bitfinex.SetRequestHandler("bitfinex", 1000, 1000, new(http.Client))
	if err == nil {
		t.Error("Test failed - request SetRequestHandler()", err)
	}
	err = bitfinex.SetRequestHandler("bla", 1000, 1000, new(http.Client))
	if err == nil {
		t.Error("Test failed - request SetRequestHandler()", err)
	}
	BTCMarkets = new(Handler)
	BTCMarkets.SetRequestHandler("btcmarkets", 1000, 1000, new(http.Client))

	if len(request.exchangeHandlers) != 2 {
		t.Error("test failed - request GetRequestHandler() error")
	}
	wg.Add(2)
}

func TestSetRateLimit(t *testing.T) {
	bitfinex.SetRateLimit(0, 0)
	BTCMarkets.SetRateLimit(0, 0)
}

func TestSend(t *testing.T) {
	for i := 0; i < 1; i++ {
		go func() {
			var v interface{}
			err := bitfinex.SendPayload("GET",
				"https://api.bitfinex.com/v1/pubticker/BTCUSD",
				nil,
				nil,
				&v,
				false,
				false,
			)
			if err != nil {
				t.Error("test failed - send error", err)
			}
			wg.Done()
		}()
		go func() {
			var v interface{}
			err := BTCMarkets.SendPayload("GET",
				"https://api.btcmarkets.net/market/BTC/AUD/tick",
				nil,
				nil,
				&v,
				false,
				false,
			)
			if err != nil {
				t.Error("test failed - send error", err)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	newHandler := new(Handler)
	err := newHandler.SendPayload("GET", "https://api.bitfinex.com/v1/pubticker/BTCUSD",
		nil, nil, nil, false, false)
	if err == nil {
		t.Error("test failed - request Send() error", err)
	}
}
