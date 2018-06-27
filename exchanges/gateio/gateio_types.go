package gateio

import "time"

// SpotNewOrderRequestParamsType 交易类型
type SpotNewOrderRequestParamsType string

var (
	// SpotNewOrderRequestParamsTypeBuy 买
	SpotNewOrderRequestParamsTypeBuy = SpotNewOrderRequestParamsType("buy")

	// SpotNewOrderRequestParamsTypeSell 卖
	SpotNewOrderRequestParamsTypeSell = SpotNewOrderRequestParamsType("sell")
)

// TimeInterval Interval represents interval enum.
type TimeInterval int

var (
	TimeIntervalMinute         = TimeInterval(60)
	TimeIntervalThreeMinutes   = TimeInterval(60 * 3)
	TimeIntervalFiveMinutes    = TimeInterval(60 * 5)
	TimeIntervalFifteenMinutes = TimeInterval(60 * 15)
	TimeIntervalThirtyMinutes  = TimeInterval(60 * 30)
	TimeIntervalHour           = TimeInterval(60 * 60)
	TimeIntervalTwoHours       = TimeInterval(2 * 60 * 60)
	TimeIntervalFourHours      = TimeInterval(4 * 60 * 60)
	TimeIntervalSixHours       = TimeInterval(6 * 60 * 60)
	TimeIntervalDay            = TimeInterval(60 * 60 * 24)
)

//------------Market Info

// MarketInfoResponse 交易市场的参数信息
type MarketInfoResponse struct {
	Result string                    `json:"result"`
	Pairs  []MarketInfoPairsResponse `json:"pairs"`
}

// MarketInfoPairsResponse 交易市场的参数信息-交易对
type MarketInfoPairsResponse struct {
	Symbol string
	// DecimalPlaces 价格精度
	DecimalPlaces float64
	// MinAmount 最小下单量
	MinAmount float64
	// Fee 交易费
	Fee float64
}

//------------Balances

// BalancesResponse 用户资产
type BalancesResponse struct {
	Result    string            `json:"result"`
	Available map[string]string `json:"available"`
	Locked    map[string]string `json:"locked"`
}

//------------Kline

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol   string //必填项，交易对:LTCBTC,BTCUSDT
	HourSize int    //多少个小时内的数据
	GroupSec TimeInterval
}

// KLineResponse K线返回类型
type KLineResponse struct {
	ID        float64
	KlineTime time.Time
	Open      float64
	Time      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	Amount    float64 `db:"amount"`
}

// TickerResponse  获取单项交易行情有请求返回值
type TickerResponse struct {
	Result        string `json:"result"`
	Volume        string `json:"baseVolume"`    //交易量
	High          string `json:"high24hr"`      // 24小时最高价
	Open          string `json:"highestBid"`    // 买方最高价
	Last          string `json:"last"`          // 最新成交价
	Low           string `json:"low24hr"`       // 24小时最低价
	Close         string `json:"lowestAsk"`     // 卖方最低价
	PercentChange string `json:"percentChange"` // 涨跌百分比
	QuoteVolume   string `json:"quoteVolume"`   // 兑换货币交易量

}

// SpotNewOrderRequestParams 下单买入/卖出请求参数
type SpotNewOrderRequestParams struct {
	Amount float64                       `json:"amount"` // 下单数量
	Price  float64                       `json:"price"`  // 下单价格
	Symbol string                        `json:"symbol"` // 交易对, btc_usdt, eth_btc......
	Type   SpotNewOrderRequestParamsType `json:"type"`   // 订单类型,
}

// SpotNewOrderResponse 下单买入/卖出返回的类型
type SpotNewOrderResponse struct {
	OrderNumber  int64  `json:"orderNumber"`  //订单单号
	Price        string `json:"rate"`         //下单价格
	LeftAmount   string `json:"leftAmount"`   //剩余数量
	FilledAmount string `json:"filledAmount"` //成交数量
	// FilledPrice  string `json:"filledRate"`   //成交价格
}
