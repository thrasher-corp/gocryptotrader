package bithumb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
)

var (
	wsTickerResp    = []byte(`{"type":"ticker","content":{"tickType":"24H","date":"20210811","time":"132017","openPrice":"33400","closePrice":"34010","lowPrice":"32660","highPrice":"34510","value":"45741663716.89916828275244531","volume":"1359398.496892086826189907","sellVolume":"198021.237915860451480504","buyVolume":"1161377.258976226374709403","prevClosePrice":"33530","chgRate":"1.83","chgAmt":"610","volumePower":"500","symbol":"UNI_KRW"}}`)
	wsTransResp     = []byte(`{"type":"transaction","content":{"list":[{"buySellGb":"1","contPrice":"1166","contQty":"125.2400","contAmt":"146029.8400","contDtm":"2021-08-13 15:23:42.911273","updn":"dn","symbol":"DAI_KRW"}]}}`)
	wsOrderbookResp = []byte(`{"type":"orderbookdepth","content":{"list":[{"symbol":"XLM_KRW","orderType":"ask","price":"401.2","quantity":"0","total":"0"},{"symbol":"XLM_KRW","orderType":"ask","price":"401.6","quantity":"21277.735","total":"1"},{"symbol":"XLM_KRW","orderType":"ask","price":"403.3","quantity":"4000","total":"1"},{"symbol":"XLM_KRW","orderType":"bid","price":"399.5","quantity":"0","total":"0"},{"symbol":"XLM_KRW","orderType":"bid","price":"398.2","quantity":"0","total":"0"},{"symbol":"XLM_KRW","orderType":"bid","price":"399.8","quantity":"31416.8779","total":"1"},{"symbol":"XLM_KRW","orderType":"bid","price":"398.5","quantity":"34328.387","total":"1"}],"datetime":"1628835823604483"}}`)
)

func TestWsHandleData(t *testing.T) {
	t.Parallel()

	pairs := currency.Pairs{currency.NewBTCUSDT()}

	dummy := Exchange{
		location: time.Local,
		Base: exchange.Base{
			Name: "dummy",
			Features: exchange.Features{
				Enabled: exchange.FeaturesEnabled{SaveTradeData: true},
			},
			CurrencyPairs: currency.PairsManager{
				Pairs: map[asset.Item]*currency.PairStore{
					asset.Spot: {
						Available: pairs,
						Enabled:   pairs,
						ConfigFormat: &currency.PairFormat{
							Uppercase: true,
							Delimiter: currency.DashDelimiter,
						},
					},
				},
			},
			Websocket: websocket.NewManager(),
		},
	}

	dummy.setupOrderbookManager(t.Context())
	dummy.API.Endpoints = e.NewEndpoints()

	welcomeMsg := []byte(`{"status":"0000","resmsg":"Connected Successfully"}`)
	err := dummy.wsHandleData(t.Context(), welcomeMsg)
	require.NoError(t, err)

	err = dummy.wsHandleData(t.Context(), []byte(`{"status":"1336","resmsg":"Failed"}`))
	require.ErrorIs(t, err, websocket.ErrSubscriptionFailure)

	err = dummy.wsHandleData(t.Context(), wsTransResp)
	require.NoError(t, err)

	err = dummy.wsHandleData(t.Context(), wsOrderbookResp)
	require.NoError(t, err)

	err = dummy.wsHandleData(t.Context(), wsTickerResp)
	require.NoError(t, err)
	assert.IsType(t, new(ticker.Price), (<-dummy.Websocket.DataHandler.C).Data, "ticker should send a price to the DataHandler")
}

func TestSubToReq(t *testing.T) {
	t.Parallel()
	p := currency.Pairs{currency.NewPairWithDelimiter("BTC", "KRW", "_"), currency.NewPairWithDelimiter("ETH", "KRW", "_")}
	r := subToReq(&subscription.Subscription{Channel: subscription.AllTradesChannel}, p)
	assert.Equal(t, "transaction", r.Type)
	assert.True(t, p.Equal(r.Symbols))
	r = subToReq(&subscription.Subscription{Channel: subscription.OrderbookChannel}, p)
	assert.Equal(t, "orderbookdepth", r.Type)
	assert.True(t, p.Equal(r.Symbols))
	r = subToReq(&subscription.Subscription{Channel: subscription.TickerChannel, Interval: kline.OneHour}, p)
	assert.Equal(t, "ticker", r.Type)
	assert.True(t, p.Equal(r.Symbols))
	assert.Equal(t, []string{"1H"}, r.TickTypes)
	assert.PanicsWithError(t,
		"subscription channel not supported: myTrades",
		func() { subToReq(&subscription.Subscription{Channel: subscription.MyTradesChannel}, p) },
		"should panic on invalid channel",
	)
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	p := currency.Pairs{currency.NewPairWithDelimiter("BTC", "KRW", "_"), currency.NewPairWithDelimiter("ETH", "KRW", "_")}
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, p, false))
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, p, true))
	subs, err := e.generateSubscriptions()
	require.NoError(t, err)
	exp := subscription.List{
		{Asset: asset.Spot, Channel: subscription.AllTradesChannel, Pairs: p, QualifiedChannel: `{"type":"transaction","symbols":["BTC_KRW","ETH_KRW"]}`},
		{Asset: asset.Spot, Channel: subscription.OrderbookChannel, Pairs: p, QualifiedChannel: `{"type":"orderbookdepth","symbols":["BTC_KRW","ETH_KRW"]}`},
		{
			Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: p, Interval: kline.ThirtyMin,
			QualifiedChannel: `{"type":"ticker","symbols":["BTC_KRW","ETH_KRW"],"tickTypes":["30M"]}`,
		},
	}
	testsubs.EqualLists(t, exp, subs)
}
