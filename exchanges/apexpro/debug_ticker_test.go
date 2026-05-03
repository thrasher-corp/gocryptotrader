package apexpro

import (
	"net/url"
	"strconv"
	"testing"
	"time"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestDebugTickerSymbol(t *testing.T) {
	e.Verbose = true
	
	// Check OMNI ticker
	var resp []TickerData
	err := e.SendHTTPRequest(t.Context(), exchange.RestFutures, "v3/ticker?symbol=BTC-USDC", request.UnAuth, &resp)
	t.Logf("OMNI v3/ticker BTC-USDC: len=%d, err=%v", len(resp), err)

	// Check OMNI klines with time range
	params := url.Values{}
	params.Set("symbol", "BTC-USDC")
	params.Set("interval", "15")
	params.Set("start", strconv.FormatInt(time.Now().Add(-time.Hour*2).UnixMilli(), 10))
	params.Set("end", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("limit", "10")
	var klines map[string][]CandlestickData
	err2 := e.SendHTTPRequest(t.Context(), exchange.RestFutures, "v3/klines?"+params.Encode(), request.UnAuth, &klines)
	t.Logf("OMNI v3/klines 2h: len=%d, err=%v", len(klines), err2)

	// Check what happens with v3/klines without symbol
	var resp3 map[string][]CandlestickData
	err3 := e.SendHTTPRequest(t.Context(), exchange.RestSpot, "v3/klines", request.UnAuth, &resp3)
	t.Logf("PRO v3/klines (no params): len=%d, err=%v", len(resp3), err3)
}
