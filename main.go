package main

import (
	"fmt"
	"time"

	"github.com/idoall/gocryptotrader/communications"
	"github.com/idoall/gocryptotrader/config"
	exchange "github.com/idoall/gocryptotrader/exchanges"
	"github.com/idoall/gocryptotrader/exchanges/bitmex"
	"github.com/idoall/gocryptotrader/portfolio"
)

// Bot contains configuration, portfolio, exchange & ticker data and is the
// overarching type across this code base.
type Bot struct {
	config     *config.Config
	portfolio  *portfolio.Base
	exchanges  []exchange.IBotExchange
	comms      *communications.Communications
	shutdown   chan bool
	dryRun     bool
	configFile string
}

const banner = `
   ______        ______                     __        ______                  __
  / ____/____   / ____/_____ __  __ ____   / /_ ____ /_  __/_____ ______ ____/ /___   _____
 / / __ / __ \ / /    / ___// / / // __ \ / __// __ \ / /  / ___// __  // __  // _ \ / ___/
/ /_/ // /_/ // /___ / /   / /_/ // /_/ // /_ / /_/ // /  / /   / /_/ // /_/ //  __// /
\____/ \____/ \____//_/    \__, // .___/ \__/ \____//_/  /_/    \__,_/ \__,_/ \___//_/
                          /____//_/
`

var bot Bot

// getDefaultConfig 获取默认配置
func getDefaultConfig() config.ExchangeConfig {
	return config.ExchangeConfig{
		Name:                    "Bitmex",
		Enabled:                 true,
		Verbose:                 true,
		Websocket:               true,
		BaseAsset:               "btc",
		QuoteAsset:              "usdt",
		RESTPollingDelay:        10,
		HTTPTimeout:             3 * time.Second,
		AuthenticatedAPISupport: true,
		APIKey:                  "X0X8_5ugrifK6dcAjRFY_UsN",
		APISecret:               "DRKlKBwvHPVsRZGhfckzE272EYvUpYovZ5pgwiPx46J9c5j7",
	}
}

func main() {
	// new(binance.Binance).WebsocketClient()
	// exchange := gateio.Gateio{}
	// exchange := bitfinex.Bitfinex{}
	// exchange := okex.OKEX{}
	// exchange := huobi.HUOBI{}
	// exchange := zb.ZB{}
	// exchange := binance.Binance{}

	exchange := bitmex.Bitmex{}
	defaultConfig := getDefaultConfig()
	exchange.SetDefaults()
	fmt.Println("----------setup-------")
	exchange.Setup(defaultConfig)
	//bitmex.GenericRequestParams{}

	err := exchange.WebsocketKline()
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Duration(1) * time.Hour)

	//--------------批量创建新订单
	// list, err := exchange.CreateBulkOrders(bitmex.OrderNewBulkParams{
	// 	[]bitmex.OrderNewParams{
	// 		bitmex.OrderNewParams{
	// 			Symbol:   "XBTUSD",
	// 			Side:     "Buy",
	// 			Price:    6520,
	// 			ClOrdID:  "test/idoall1",
	// 			OrderQty: 200,
	// 		},
	// 	},
	// })
	// if err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	for _, v := range list {
	// 		b, _ := commonutils.JSONEncode(v)
	// 		fmt.Printf("%s\n", b)
	// 	}
	// }

	//------------------平仓
	// res, err := exchange.CreateOrder(bitmex.OrderNewParams{
	// 	Symbol: "XBTUSD",
	// 	// Side:     "Buy",
	// 	Price:    6210,
	// 	ExecInst: "Close",
	// 	// ClOrdID:  "test/idoall",
	// 	// OrderQty: 10,
	// })
	// if err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	fmt.Printf("%v\n", res)
	// }

	//-----------------取消订单，平仓或者发布中未成交的都可以取消
	// list, err := exchange.CancelOrders(bitmex.OrderCancelParams{
	// 	OrderID: "94673f58-3edc-46a2-d2e3-7d201ddacffe",
	// })

	// // list, err := exchange.GetStats()
	// if err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	for _, v := range list {
	// 		b, _ := commonutils.JSONEncode(v)
	// 		fmt.Printf("%s\n", b)
	// 	}
	// }

	// ---------------获取K线,每个时间段的统计，都是向前推5分钟，例如1小时的是从5分开始到一个小时的0分
	// list, err := exchange.GetPreviousTrades(bitmex.TradeGetBucketedParams{
	// 	BinSize:   string(bitmex.TimeIntervalMinute),
	// 	Symbol:    "XBT",
	// 	Reverse:   false, //如果为true，从endTime开始是倒排序
	// 	Partial:   true,  //当前时段在更新的数据也发送过来
	// 	Count:     10,
	// 	StartTime: "2017-01-01T00:00:00.000Z",
	// 	EndTime:   "2018-10-26T13:00:00.000Z",
	// })

	// // list, err := exchange.GetStats()
	// if err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	for k, v := range list {
	// 		t, _ := time.ParseInLocation("2006-01-02T15:04:05.000Z", v.Timestamp, time.Local)
	// 		t = t.Add(time.Duration(8) * time.Hour)
	// 		b, _ := commonutils.JSONEncode(v)
	// 		fmt.Printf("index:%d %s %s\n", k, t.Format("2006-01-02 15:04:05"), b)
	// 	}
	// }

	// list, err := exchange.CancelOrders(bitmex.OrderCancelParams{
	// 	OrderID: "76049cb1-abba-efef-a918-373d151ee892",
	// })

	//----------BM调整杠杆
	// res, err := exchange.LeveragePosition(bitmex.PositionUpdateLeverageParams{Leverage: 10, Symbol: "XBTUSD"})

	// // list, err := exchange.GetStats()
	// if err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	fmt.Printf("%+v\n", res)

	// 	// for _, v := range list {
	// 	// 	b, _ := commonutils.JSONEncode(v)
	// 	// 	fmt.Printf("%s\n", b)
	// 	// }

	// }

	// ch := make(chan *binance.KlineStream)
	// done := make(chan struct{})
	// timeIntervals := []binance.TimeInterval{
	// 	binance.TimeIntervalFiveMinutes,
	// 	binance.TimeIntervalMinute,
	// 	binance.TimeIntervalDay,
	// 	binance.TimeIntervalHour,
	// 	binance.TimeIntervalTwoHours,
	// }

	// go exchange.WebsocketKline(ch, timeIntervals, done)
	// for {
	// 	fmt.Fprintln(os.Stdout, gocolorize.NewColor("green").Paint("接收....."))
	// 	kline := <-ch
	// 	log.Println("Kline received", "value:", kline.Kline.Interval, kline.Symbol, kline.EventTime, kline.Kline.HighPrice, kline.Kline.LowPrice)
	// }
	// res, err := exchange.GetExchangeInfo()

	// if err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	// fmt.Printf("%v\n", res)

	// 	for _, v := range res.Symbols {
	// 		if v.BaseAsset == "BTC" {
	// 			b, _ := commonutils.JSONEncode(v)
	// 			fmt.Printf("%s\n", b)
	// 		}
	// 	}

	// }

	// sh1 := common.GetHMAC(common.MD5New, []byte("accesskey=6d8f62fd-3086-46e3-a0ba-c66a929c24e2&method=getAccountInfo"), []byte(common.Sha1ToHex("48939bbc-8d49-402b-b731-adadf2ea9628")))
	// fmt.Println(common.HexEncodeToString((sh1)))
	// arg := huobi.SpotNewOrderRequestParams{
	// 	Symbol:    exchange.GetSymbol(),
	// 	AccountID: 3838465,
	// 	Amount:    0.01,
	// 	Price:     10.1,
	// 	Type:      huobi.SpotNewOrderRequestTypeBuyLimit,
	// }
	// fmt.Println(exchange.SpotNewOrder(arg))

	// res, err := exchange.SpotNewOrder(okex.SpotNewOrderRequestParams{
	// 	Symbol: exchange.GetSymbol(),
	// 	Amount: 1.1,
	// 	Price:  10.1,
	// 	Type:   okex.SpotNewOrderRequestTypeBuy,
	// })
	// if err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	fmt.Println(res)
	// }

	// fmt.Println(exchange.GetKline("btcusdt", "1min", ""))
	// fmt.Println(exchange.GetKline("btcusdt", "1min", ""))
	// fmt.Println(exchange.GetKline("btcusdt", "15min", ""))
	// fmt.Println(exchange.GetKline("btcusdt", "1hour", ""))
	// fmt.Println(exchange.GetKline("btcusdt", "1day", ""))

	// list, err := exchange.GetAccountInfo()
	// if err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	for k, v := range list {
	// 		// b, _ := json.Marshal(v)
	// 		fmt.Printf("%s:%v \n", k, v)
	// 	}
	// }

	// fmt.Println(exchange.CancelOrder(917591554, exchange.GetSymbol()))

	//获取交易所的规则和交易对信息
	// getExchangeInfo(exchange)

	//获取交易深度
	// getOrderBook(exchange)

	//获取最近的交易记录
	// getRecentTrades(exchange)

	//获取 k 线数据
	// getCandleStickData(exchange)

	//获取最新价格
	// getLatestSpotPrice(exchange)

	//新订单
	// newOrder(exchange)

	//取消订单
	// cancelOrder(exchange, 82584683)

	// fmt.Println(exchange.GetAccount())

	// fmt.Println(exchange.GetSymbol())

}
