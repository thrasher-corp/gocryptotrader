package zb

import "time"

//-------------Accounts

type AccountsResponseCoin struct {
	Freez       string `json:"freez"`       //冻结资产
	EnName      string `json:"enName"`      //币种英文名
	UnitDecimal int    `json:"unitDecimal"` //保留小数位
	UnName      string `json:"cnName"`      //币种中文名
	UnitTag     string `json:"unitTag"`     //币种符号
	Available   string `json:"available"`   //可用资产
	Key         string `json:"key"`         //币种
}

type AccountsBaseResponse struct {
	UserName             string `json:"username"`               //用户名
	TradePasswordEnabled bool   `json:"trade_password_enabled"` //是否开通交易密码
	AuthGoogleEnabled    bool   `json:"auth_google_enabled"`    //是否开通谷歌验证
	AuthMobileEnabled    bool   `json:"auth_mobile_enabled"`    //是否开通手机验证
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

//---------------Markets

type MarketResponseItem struct {
	AmountScale float64 `json:"amountScale"`
	PriceScale  float64 `json:"priceScale"`
}

//-------------Ticket

type TicketResponse struct {
	Date   string              `json:"date"`
	Ticket TicketChildResponse `json:"ticker"`
}

type TicketChildResponse struct {
	Vol  string //成交量(最近的24小时)
	Last string //最新成交价
	Sell string //卖一价
	Buy  string //买一价
	High string //最高价
	Low  string //最低价
}

//-------------Trade Place Order

// SpotNewOrderRequestParamsType ZB 交易类型
type SpotNewOrderRequestParamsType string

var (
	// SpotNewOrderRequestParamsTypeBuy 买
	SpotNewOrderRequestParamsTypeBuy = SpotNewOrderRequestParamsType("1")
	// SpotNewOrderRequestParamsTypeSell 卖
	SpotNewOrderRequestParamsTypeSell = SpotNewOrderRequestParamsType("0")
)

type SpotNewOrderRequestParams struct {
	Amount float64                       `json:"amount"`    // 交易数量
	Price  float64                       `json:"price"`     // 下单价格,
	Symbol string                        `json:"currency"`  // 交易对, btcusdt, bccbtc......
	Type   SpotNewOrderRequestParamsType `json:"tradeType"` // 订单类型, buy-market: 市价买, sell-market: 市价卖, buy-limit: 限价买, sell-limit: 限价卖
}

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

var (
	TimeInterval_Minute         = TimeInterval("1min")
	TimeInterval_ThreeMinutes   = TimeInterval("3min")
	TimeInterval_FiveMinutes    = TimeInterval("5min")
	TimeInterval_FifteenMinutes = TimeInterval("15min")
	TimeInterval_ThirtyMinutes  = TimeInterval("30min")
	TimeInterval_Hour           = TimeInterval("1hour")
	TimeInterval_TwoHours       = TimeInterval("2hour")
	TimeInterval_FourHours      = TimeInterval("4hour")
	TimeInterval_SixHours       = TimeInterval("6hour")
	TimeInterval_TwelveHours    = TimeInterval("12hour")
	TimeInterval_Day            = TimeInterval("1day")
	TimeInterval_ThreeDays      = TimeInterval("3day")
	TimeInterval_Week           = TimeInterval("1week")
)
