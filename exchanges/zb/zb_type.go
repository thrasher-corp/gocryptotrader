package zb

import "time"
import "github.com/thrasher-/gocryptotrader/currency/symbol"

// OrderbookResponse holds the orderbook data for a symbol
type OrderbookResponse struct {
	Timestamp int64       `json:"timestamp"`
	Asks      [][]float64 `json:"asks"`
	Bids      [][]float64 `json:"bids"`
}

// AccountsResponseCoin holds the accounts coin details
type AccountsResponseCoin struct {
	Freez       string `json:"freez"`       //冻结资产
	EnName      string `json:"enName"`      //币种英文名
	UnitDecimal int    `json:"unitDecimal"` //保留小数位
	UnName      string `json:"cnName"`      //币种中文名
	UnitTag     string `json:"unitTag"`     //币种符号
	Available   string `json:"available"`   //可用资产
	Key         string `json:"key"`         //币种
}

// AccountsBaseResponse holds basic account details
type AccountsBaseResponse struct {
	UserName             string `json:"username"`               //用户名
	TradePasswordEnabled bool   `json:"trade_password_enabled"` //是否开通交易密码
	AuthGoogleEnabled    bool   `json:"auth_google_enabled"`    //是否开通谷歌验证
	AuthMobileEnabled    bool   `json:"auth_mobile_enabled"`    //是否开通手机验证
}

type UnfinishedOpenOrder struct {
	Currency    string  `json:"currency"`
	ID          int64   `json:"id"`
	Price       int     `json:"price"`
	Status      int     `json:"status"`
	TotalAmount float64 `json:"total_amount"`
	TradeAmount int     `json:"trade_amount"`
	TradeDate   int     `json:"trade_date"`
	TradeMoney  int     `json:"trade_money"`
	Type        int     `json:"type"`
}

// AccountsResponse 用户基本信息
type AccountsResponse struct {
	Result struct {
		Coins []AccountsResponseCoin `json:"coins"`
		Base  AccountsBaseResponse   `json:"base"`
	} `json:"result"` //用户名
	AssetPerm   bool `json:"assetPerm"`   //是否开通交易密码
	LeverPerm   bool `json:"leverPerm"`   //是否开通谷歌验证
	EntrustPerm bool `json:"entrustPerm"` //是否开通手机验证
	MoneyPerm   bool `json:"moneyPerm"`   // 资产列表
}

// MarketResponseItem stores market data
type MarketResponseItem struct {
	AmountScale float64 `json:"amountScale"`
	PriceScale  float64 `json:"priceScale"`
}

// TickerResponse holds the ticker response data
type TickerResponse struct {
	Date   string              `json:"date"`
	Ticker TickerChildResponse `json:"ticker"`
}

// TickerChildResponse holds the ticker child response data
type TickerChildResponse struct {
	Vol  float64 `json:"vol,string"`  //成交量(最近的24小时)
	Last float64 `json:"last,string"` //最新成交价
	Sell float64 `json:"sell,string"` //卖一价
	Buy  float64 `json:"buy,string"`  //买一价
	High float64 `json:"high,string"` //最高价
	Low  float64 `json:"low,string"`  //最低价
}

// SpotNewOrderRequestParamsType ZB 交易类型
type SpotNewOrderRequestParamsType string

var (
	// SpotNewOrderRequestParamsTypeBuy 买
	SpotNewOrderRequestParamsTypeBuy = SpotNewOrderRequestParamsType("1")
	// SpotNewOrderRequestParamsTypeSell 卖
	SpotNewOrderRequestParamsTypeSell = SpotNewOrderRequestParamsType("0")
)

// SpotNewOrderRequestParams is the params used for placing an order
type SpotNewOrderRequestParams struct {
	Amount float64                       `json:"amount"`    // 交易数量
	Price  float64                       `json:"price"`     // 下单价格,
	Symbol string                        `json:"currency"`  // 交易对, btcusdt, bccbtc......
	Type   SpotNewOrderRequestParamsType `json:"tradeType"` // 订单类型, buy-market: 市价买, sell-market: 市价卖, buy-limit: 限价买, sell-limit: 限价卖
}

// SpotNewOrderResponse stores the new order response data
type SpotNewOrderResponse struct {
	Code    int    `json:"code"`    //返回代码
	Message string `json:"message"` //提示信息
	ID      string `json:"id"`      //委托挂单号
}

// //-------------Kline

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol string       //交易对, zb_qc,zb_usdt,zb_btc...
	Type   TimeInterval //K线类型, 1min, 3min, 15min, 30min, 1hour......
	Since  string       //从这个时间戳之后的
	Size   int          //返回数据的条数限制(默认为1000，如果返回数据多于1000条，那么只返回1000条)
}

// KLineResponseData Kline Data
type KLineResponseData struct {
	ID        float64   `json:"id"` // K线ID
	KlineTime time.Time `json:"klineTime"`
	Open      float64   `json:"open"`  // 开盘价
	Close     float64   `json:"close"` // 收盘价, 当K线为最晚的一根时, 时最新成交价
	Low       float64   `json:"low"`   // 最低价
	High      float64   `json:"high"`  // 最高价
	Volume    float64   `json:"vol"`   // 成交量
}

// KLineResponse K线返回类型
type KLineResponse struct {
	// Data      string                `json:"data"`      // 买入货币
	MoneyType string               `json:"moneyType"` // 卖出货币
	Symbol    string               `json:"symbol"`    // 内容说明
	Data      []*KLineResponseData `json:"data"`      // KLine数据
}

// TimeInterval represents interval enum.
type TimeInterval string

// TimeInterval vars
var (
	TimeIntervalMinute         = TimeInterval("1min")
	TimeIntervalThreeMinutes   = TimeInterval("3min")
	TimeIntervalFiveMinutes    = TimeInterval("5min")
	TimeIntervalFifteenMinutes = TimeInterval("15min")
	TimeIntervalThirtyMinutes  = TimeInterval("30min")
	TimeIntervalHour           = TimeInterval("1hour")
	TimeIntervalTwoHours       = TimeInterval("2hour")
	TimeIntervalFourHours      = TimeInterval("4hour")
	TimeIntervalSixHours       = TimeInterval("6hour")
	TimeIntervalTwelveHours    = TimeInterval("12hour")
	TimeIntervalDay            = TimeInterval("1day")
	TimeIntervalThreeDays      = TimeInterval("3day")
	TimeIntervalWeek           = TimeInterval("1week")
)

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change, using highest value
var WithdrawalFees = map[string]float64{
	symbol.ZB:     5,
	symbol.BTC:    0.001,
	symbol.BCH:    0.0006,
	symbol.LTC:    0.005,
	symbol.ETH:    0.01,
	symbol.ETC:    0.01,
	symbol.BTS:    3,
	symbol.EOS:    0.1,
	symbol.QTUM:   0.01,
	symbol.HC:     0.001,
	symbol.XRP:    0.1,
	symbol.QC:     5,
	symbol.DASH:   0.002,
	symbol.BCD:    0,
	symbol.UBTC:   0.001,
	symbol.SBTC:   0,
	symbol.INK:    60,
	symbol.BTH:    0.01,
	symbol.LBTC:   0.01,
	symbol.CHAT:   20,
	symbol.BITCNY: 20,
	symbol.HLC:    100,
	symbol.BTP:    0.001,
	symbol.TOPC:   200,
	symbol.ENT:    50,
	symbol.BAT:    40,
	symbol.FIRST:  30,
	symbol.SAFE:   0.001,
	symbol.QUN:    200,
	symbol.BTN:    0.005,
	symbol.TRUE:   5,
	symbol.CDC:    1,
	symbol.DDM:    1,
	symbol.HOTC:   150,
	symbol.USDT:   5,
	symbol.XUC:    1,
	symbol.EPC:    40,
	symbol.BDS:    3,
	symbol.GRAM:   5,
	symbol.DOGE:   20,
	symbol.NEO:    0,
	symbol.OMG:    0.5,
	symbol.BTM:    4,
	symbol.SNT:    60,
	symbol.AE:     3,
	symbol.ICX:    3,
	symbol.ZRX:    10,
	symbol.EDO:    4,
	symbol.FUN:    250,
	symbol.MANA:   70,
	symbol.RCN:    70,
	symbol.MCO:    0.6,
	symbol.MITH:   10,
	symbol.KNC:    5,
	symbol.XLM:    0.1,
	symbol.GNT:    20,
	symbol.MTL:    3,
	symbol.SUB:    20,
	symbol.XEM:    4,
	symbol.EOSDAC: 0,
	symbol.KAN:    350,
	symbol.AAA:    1,
	symbol.XWC:    1,
	symbol.PDX:    1,
	symbol.SLT:    100,
	symbol.ADA:    1,
	symbol.HPY:    100,
	symbol.PAX:    5,
	symbol.XTZ:    0.1,
}
