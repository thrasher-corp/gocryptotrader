package mexc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWsHandle(t *testing.T) {
	t.Parallel()
	pushDataMap := map[string]string{
		"spot@public.aggre.deals.v3.api.pb":          `{"channel": "spot@public.aggre.deals.v3.api.pb@100ms@BTCUSDT", "publicdeals": { "dealsList": [ { "price": "93220.00", "quantity": "0.04438243", "tradetype": 2, "time": 1736409765051 } ], "eventtype": "spot@public.aggre.deals.v3.api.pb@100ms" }, "symbol": "BTCUSDT", "sendtime": 1736409765052 }`,
		"spot@public.kline.v3.api.pb":                `{"channel": "spot@public.kline.v3.api.pb@BTCUSDT@Min15", "publicspotkline": { "interval": "Min15", "windowstart": 1736410500, "openingprice": "92925", "closingprice": "93158.47", "highestprice": "93158.47", "lowestprice": "92800", "volume": "36.83803224", "amount": "3424811.05", "windowend": 1736411400 }, "symbol": "BTCUSDT", "symbolid": "2fb942154ef44a4ab2ef98c8afb6a4a7", "createtime": 1736410707571}`,
		"spot@public.aggre.depth.v3.api.pb":          `{"channel": "spot@public.aggre.depth.v3.api.pb@100ms@BTCUSDT", "publicincreasedepths": { "asksList": [], "bidsList": [ { "price": "92877.58", "quantity": "0.00000000" } ], "eventtype": "spot@public.aggre.depth.v3.api.pb@100ms", "version": "36913293511" }, "symbol": "BTCUSDT", "sendtime": 1736411507002}`,
		"spot@public.increase.depth.batch.v3.api.pb": `{"channel" : "spot@public.increase.depth.batch.v3.api.pb@BTCUSDT", "symbol" : "BTCUSDT", "sendTime" : "1739502064578", "publicIncreaseDepthsBatch" : { "items" : [ { "asks" : [ ], "bids" : [ { "price" : "96578.48", "quantity" : "0.00000000" } ], "eventType" : "", "version" : "39003145507" }, { "asks" : [ ], "bids" : [ { "price" : "96578.90", "quantity" : "0.00000000" } ], "eventType" : "", "version" : "39003145508" }, { "asks" : [ ], "bids" : [ { "price" : "96579.31", "quantity" : "0.00000000" } ], "eventType" : "", "version" : "39003145509" }, { "asks" : [ ], "bids" : [ { "price" : "96579.84", "quantity" : "0.00000000" } ], "eventType" : "", "version" : "39003145510" }, { "asks" : [ ], "bids" : [ { "price" : "96576.69", "quantity" : "4.88725694" } ], "eventType" : "", "version" : "39003145511" } ], "eventType" : "spot@public.increase.depth.batch.v3.api.pb"}}`,
		"spot@public.limit.depth.v3.api.pb":          `{"channel": "spot@public.limit.depth.v3.api.pb@BTCUSDT@5", "publiclimitdepths": { "asksList": [ { "price": "93180.18", "quantity": "0.21976424" } ], "bidsList": [ { "price": "93179.98", "quantity": "2.82651000" } ], "eventtype": "spot@public.limit.depth.v3.api.pb", "version": "36913565463" }, "symbol": "BTCUSDT", "sendtime": 1736411838730}`,
		"spot@public.aggre.bookTicker.v3.api.pb":     `{"channel": "spot@public.aggre.bookTicker.v3.api.pb@100ms@BTCUSDT", "publicbookticker": { "bidprice": "93387.28", "bidquantity": "3.73485", "askprice": "93387.29", "askquantity": "7.669875" }, "symbol": "BTCUSDT", "sendtime": 1736412092433 }`,
		"spot@public.bookTicker.batch.v3.api.pb":     `{"channel" : "spot@public.bookTicker.batch.v3.api.pb@BTCUSDT", "symbol" : "BTCUSDT", "sendTime" : "1739503249114", "publicBookTickerBatch" : { "items" : [ { "bidPrice" : "96567.37", "bidQuantity" : "3.362925", "askPrice" : "96567.38", "askQuantity":"1.545255"}]}}`,
		"spot@private.deals.v3.api.pb":               `{channel: "spot@private.deals.v3.api.pb", symbol: "MXUSDT", sendTime: 1736417034332, privateDeals { price: "3.6962", quantity: "1", amount: "3.6962", tradeType: 2, tradeId: "505979017439002624X1", orderId: "C02__505979017439002624115", feeAmount: "0.0003998377369698171", feeCurrency: "MX", time: 1736417034280}}`,
		"spot@private.orders.v3.api.pb":              `{channel: "spot@private.orders.v3.api.pb", symbol: "MXUSDT", sendTime: 1736417034281, privateOrders { id: "C02__505979017439002624115", price: "3.5121", quantity: "1", amount: "0", avgPrice: "3.6962", orderType: 5, tradeType: 2, remainAmount: "0", remainQuantity: "0", lastDealQuantity: "1", cumulativeQuantity: "1", cumulativeAmount: "3.6962", status: 2, createTime: 1736417034259}}`,
		"spot@private.account.v3.api.pb":             `{channel: "spot@private.account.v3.api.pb", createTime: 1736417034305, sendTime: 1736417034307, privateAccount { vcoinName: "USDT", coinId: "128f589271cb4951b03e71e6323eb7be", balanceAmount: "21.94210356004384", balanceAmountChange: "10", frozenAmount: "0", frozenAmountChange: "0", type: "CONTRACT_TRANSFER", time: 1736416910000}}`,
	}
	for elem := range pushDataMap {
		err := e.WsHandleData([]byte(pushDataMap[elem]))
		assert.NoErrorf(t, err, "%v: %s", err, elem)
	}
}

func TestWsHandleFuturesData(t *testing.T) {
	t.Parallel()
	futuresWsPushDataMap := map[string]string{
		"sub.tickers":                 `{"channel": "push.tickers", "data": [ { "fairPrice": 183.01, "lastPrice": 183, "riseFallRate": -0.0708, "symbol": "BSV_USDT", "volume24": 200 }, { "fairPrice": 220.22, "lastPrice": 220.4, "riseFallRate": -0.0686, "symbol": "BCH_USDT", "volume24": 200 } ], "ts": 1587442022003}`,
		"push.ticker":                 `{"symbol":"LINK_USDT","data":{"symbol":"LINK_USDT","lastPrice":14.022,"riseFallRate":-0.0270,"fairPrice":14.022,"indexPrice":14.028,"volume24":104524120,"amount24":149228107.8277,"maxBidPrice":16.833,"minAskPrice":11.222,"lower24Price":13.967,"high24Price":14.518,"timestamp":1746351275382,"bid1":14.02,"ask1":14.021,"holdVol":14558875,"riseFallValue":-0.390,"fundingRate":-0.000045,"zone":"UTC+8","riseFallRates":[-0.0270,-0.0594,0.1172,-0.3674,0.3499,0.0065],"riseFallRatesOfTimezone":[-0.0238,-0.0153,-0.0270]},"channel":"push.ticker","ts":1746351275382}`,
		"push.deal":                   `{"symbol":"IOTA_USDT","data":[{"p":0.1834,"v":97,"T":1,"O":1,"M":2,"t":1748810708074}],"channel":"push.deal","ts":1748810708074}`,
		"sub.depth":                   `{"channel":"push.depth", "data":{ "asks":[ [ 6859.5, 3251, 1 ] ], "bids":[ ], "version":96801927 }, "symbol":"BTC_USDT", "ts":1587442022003}`,
		"push.kline":                  `{"symbol":"CHEEMS_USDT","data":{"symbol":"CHEEMS_USDT","interval":"Min15","t":1746351000,"o":0.0000015036,"c":0.0000014988,"h":0.0000015036,"l":0.0000014962,"a":1183.078,"q":79,"ro":0.0000015021,"rc":0.0000014988,"rh":0.0000015021,"rl":0.0000014962},"channel":"push.kline","ts":1746351123147}`,
		"sub.funding.rate":            `{"channel":"push.funding.rate", "data":{ "rate":0.001, "symbol":"BTC_USDT" }, "symbol":"BTC_USDT", "ts":1587442022003 }`,
		"push.index.price":            `{"symbol":"BSV_USDT","data":{"symbol":"BSV_USDT","price":36.64},"channel":"push.index.price","ts":1746351370315}`,
		"push.fair.price":             `{"symbol":"YZYSOL_USDT","data":{"symbol":"YZYSOL_USDT","price":0.00278},"channel":"push.fair.price","ts":1746351543720}`,
		"push.personal.order":         `{"channel":"push.personal.order", "data":{ "category":1, "createTime":1610005069976, "dealAvgPrice":0.731, "dealVol":1, "errorCode":0, "externalOid":"_m_95bc2b72d3784bce8f9efecbdef9fe35", "feeCurrency":"USDT", "leverage":0, "makerFee":0, "openType":1, "orderId":"102067003631907840", "orderMargin":0, "orderType":5, "positionId":1397818, "price":0.707, "profit":-0.0005, "remainVol":0, "side":4, "state":3, "symbol":"CRV_USDT", "takerFee":0.00004386, "updateTime":1610005069983, "usedMargin":0, "version":2, "vol":1 }, "ts":1610005069989}`,
		"push.personal.asset":         `{"channel":"push.personal.asset", "data":{ "availableBalance":0.7514236, "bonus":0, "currency":"USDT", "frozenBalance":0, "positionMargin":0 }, "ts":1610005070083}`,
		"push.personal.position":      `{"channel":"push.personal.position", "data":{ "autoAddIm":false, "closeAvgPrice":0.731, "closeVol":1, "frozenVol":0, "holdAvgPrice":0.736, "holdFee":0, "holdVol":0, "im":0, "leverage":15, "liquidatePrice":0, "oim":0, "openAvgPrice":0.736, "openType":1, "positionId":1397818, "positionType":1, "realised":-0.0005, "state":3, "symbol":"CRV_USDT" },"ts":1610005070157}`,
		"push.personal.adl.level":     `{"channel":"push.personal.adl.level", "data":{ "adlLevel":0, "positionId":1397818 }, "ts":1610005032231 }`,
		"push.personal.position.mode": `{"channel":"push.personal.position.mode", "data":{ "positionMode": 1 }, "ts":1610005070157}`,
		"push.fullDepth":              `{"symbol":"INIT_USDT","data":{"asks":[[0.7542,1484,1],[0.7543,4676,2],[0.7544,11626,2],[0.7545,8247,1],[0.7546,20469,1],[0.7547,10241,1],[0.7548,26518,1],[0.7549,10490,1],[0.755,21088,1],[0.7551,16653,1],[0.7552,22110,1],[0.7553,26518,1],[0.7554,26252,1],[0.7555,16962,1],[0.7556,26518,1],[0.7557,16926,1],[0.7558,18085,1],[0.7559,16484,1],[0.756,26518,1],[0.7561,9654,1]],"bids":[[0.7541,374,1],[0.754,3186,3],[0.7539,3995,1],[0.7538,10560,1],[0.7537,12689,1],[0.7536,14731,1],[0.7535,18077,1],[0.7534,11203,1],[0.7533,9609,1],[0.7532,20530,1],[0.7531,10936,1],[0.753,11492,1],[0.7529,13563,1],[0.7528,15658,1],[0.7527,10737,1],[0.7526,15113,1],[0.7525,20870,1],[0.7524,13257,1],[0.7523,16629,1],[0.7522,10854,1]],"version":197614550},"channel":"push.depth.full","ts":1748810839220}`,
	}
	for elem := range futuresWsPushDataMap {
		t.Run(elem, func(t *testing.T) {
			t.Parallel()
			err := e.WsHandleFuturesData([]byte(futuresWsPushDataMap[elem]))
			assert.NoErrorf(t, err, "%v: %s", err, elem)
		})
	}
}
