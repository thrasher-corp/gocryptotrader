package gateio

import "time"

// GateioRequestParamsType 交易类型
type GateioRequestParamsType string

var (
	// GateioRequestParamsTypeBuy 买
	GateioRequestParamsTypeBuy = GateioRequestParamsType("buy")

	//GGateioRequestParamsTypeSell 卖
	GGateioRequestParamsTypeSell = GateioRequestParamsType("sell")
)

// GateioInterval Interval represents interval enum.
type GateioInterval int

var (
	GateioIntervalMinute         = GateioInterval(60)
	GateioIntervalThreeMinutes   = GateioInterval(60 * 3)
	GateioIntervalFiveMinutes    = GateioInterval(60 * 5)
	GateioIntervalFifteenMinutes = GateioInterval(60 * 15)
	GateioIntervalThirtyMinutes  = GateioInterval(60 * 30)
	GateioIntervalHour           = GateioInterval(60 * 60)
	GateioIntervalTwoHours       = GateioInterval(2 * 60 * 60)
	GateioIntervalFourHours      = GateioInterval(4 * 60 * 60)
	GateioIntervalSixHours       = GateioInterval(6 * 60 * 60)
	GateioIntervalDay            = GateioInterval(60 * 60 * 24)
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

// GateioBalancesResponse 用户资产
type GateioBalancesResponse struct {
	Result    string            `json:"result"`
	Available map[string]string `json:"available"`
	Locked    map[string]string `json:"locked"`
}

//------------Kline

// GateioKlinesRequestParams represents Klines request data.
type GateioKlinesRequestParams struct {
	Symbol   string //必填项，交易对:LTCBTC,BTCUSDT
	HourSize int    //多少个小时内的数据
	GroupSec GateioInterval
}

// GateioKLineResponse K线返回类型
type GateioKLineResponse struct {
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

// GateioTradeData
type GateioTradeData struct {
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

// GateioPlaceRequestParams 下单买入/卖出请求参数
type GateioPlaceRequestParams struct {
	Amount float64                 `json:"amount"` // 下单数量
	Price  float64                 `json:"price"`  // 下单价格
	Symbol string                  `json:"symbol"` // 交易对, btc_usdt, eth_btc......
	Type   GateioRequestParamsType `json:"type"`   // 订单类型,
}

// GateioPlaceResponse 下单买入/卖出返回的类型
type GateioPlaceResponse struct {
	OrderNumber  int64  `json:"orderNumber"`  //订单单号
	Price        string `json:"rate"`         //下单价格
	LeftAmount   string `json:"leftAmount"`   //剩余数量
	FilledAmount string `json:"filledAmount"` //成交数量
	// FilledPrice  string `json:"filledRate"`   //成交价格
}
