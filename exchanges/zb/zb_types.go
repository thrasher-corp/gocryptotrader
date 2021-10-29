package zb

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fee"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// OrderbookResponse holds the orderbook data for a symbol
type OrderbookResponse struct {
	Timestamp int64       `json:"timestamp"`
	Asks      [][]float64 `json:"asks"`
	Bids      [][]float64 `json:"bids"`
}

// AccountsResponseCoin holds the accounts coin details
type AccountsResponseCoin struct {
	Freeze      string `json:"freez"`       // 冻结资产
	EnName      string `json:"enName"`      // 币种英文名
	UnitDecimal int    `json:"unitDecimal"` // 保留小数位
	UnName      string `json:"cnName"`      // 币种中文名
	UnitTag     string `json:"unitTag"`     // 币种符号
	Available   string `json:"available"`   // 可用资产
	Key         string `json:"key"`         // 币种
}

// AccountsBaseResponse holds basic account details
type AccountsBaseResponse struct {
	UserName             string `json:"username"`               // 用户名
	TradePasswordEnabled bool   `json:"trade_password_enabled"` // 是否开通交易密码
	AuthGoogleEnabled    bool   `json:"auth_google_enabled"`    // 是否开通谷歌验证
	AuthMobileEnabled    bool   `json:"auth_mobile_enabled"`    // 是否开通手机验证
}

// Order is the order details for retrieving all orders
type Order struct {
	Currency    string  `json:"currency"`
	ID          int64   `json:"id,string"`
	Price       float64 `json:"price"`
	Status      int     `json:"status"`
	TotalAmount float64 `json:"total_amount"`
	TradeAmount float64 `json:"trade_amount"`
	TradeDate   int     `json:"trade_date"`
	TradeMoney  float64 `json:"trade_money"`
	Type        int64   `json:"type"`
	Fees        float64 `json:"fees,omitempty"`
	TradePrice  float64 `json:"trade_price,omitempty"`
	No          int64   `json:"no,string,omitempty"`
}

// AccountsResponse 用户基本信息
type AccountsResponse struct {
	Result struct {
		Coins []AccountsResponseCoin `json:"coins"`
		Base  AccountsBaseResponse   `json:"base"`
	} `json:"result"` // 用户名
	AssetPerm   bool `json:"assetPerm"`   // 是否开通交易密码
	LeverPerm   bool `json:"leverPerm"`   // 是否开通谷歌验证
	EntrustPerm bool `json:"entrustPerm"` // 是否开通手机验证
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
	Volume float64 `json:"vol,string"`  // 成交量(最近的24小时)
	Last   float64 `json:"last,string"` // 最新成交价
	Sell   float64 `json:"sell,string"` // 卖一价
	Buy    float64 `json:"buy,string"`  // 买一价
	High   float64 `json:"high,string"` // 最高价
	Low    float64 `json:"low,string"`  // 最低价
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
	Code    int    `json:"code"`    // 返回代码
	Message string `json:"message"` // 提示信息
	ID      string `json:"id"`      // 委托挂单号
}

// //-------------Kline

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol string // 交易对, zb_qc,zb_usdt,zb_btc...
	Type   string // K线类型, 1min, 3min, 15min, 30min, 1hour......
	Since  int64  // 从这个时间戳之后的
	Size   int64  // 返回数据的条数限制(默认为1000，如果返回数据多于1000条，那么只返回1000条)
}

// KLineResponseData Kline Data
type KLineResponseData struct {
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

// UserAddress defines Users Address for depositing funds
type UserAddress struct {
	Code    int64 `json:"code"`
	Message struct {
		Description  string `json:"des"`
		IsSuccessful bool   `json:"isSuc"`
		Data         struct {
			Address string `json:"key"`
			Tag     string // custom field we populate
		} `json:"datas"`
	} `json:"message"`
}

// MultiChainDepositAddress stores an individual multichain deposit item
type MultiChainDepositAddress struct {
	Blockchain  string `json:"blockChain"`
	IsUseMemo   bool   `json:"isUseMemo"`
	Account     string `json:"account"`
	Address     string `json:"address"`
	Memo        string `json:"memo"`
	CanDeposit  bool   `json:"canDeposit"`
	CanWithdraw bool   `json:"canWithdraw"`
}

// MultiChainDepositAddressResponse stores the multichain deposit address response
type MultiChainDepositAddressResponse struct {
	Code    int64 `json:"code"`
	Message struct {
		Description  string                     `json:"des"`
		IsSuccessful bool                       `json:"isSuc"`
		Data         []MultiChainDepositAddress `json:"datas"`
	} `json:"message"`
}

// transferFees the large list of predefined transfer fees fees prone to change
var transferFees = []fee.Transfer{
	{Currency: currency.ZB, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
	{Currency: currency.BTC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.001)},
	{Currency: currency.BCH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0006)},
	{Currency: currency.LTC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.005)},
	{Currency: currency.ETH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
	{Currency: currency.ETC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
	{Currency: currency.BTS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3)},
	{Currency: currency.EOS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
	{Currency: currency.QTUM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
	{Currency: currency.HC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.001)},
	{Currency: currency.XRP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
	{Currency: currency.QC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
	{Currency: currency.DASH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.002)},
	{Currency: currency.BCD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.UBTC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.001)},
	{Currency: currency.SBTC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.INK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(60)},
	{Currency: currency.BTH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
	{Currency: currency.LBTC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
	{Currency: currency.CHAT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.BITCNY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.HLC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
	{Currency: currency.BTP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.001)},
	{Currency: currency.TOPC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(200)},
	{Currency: currency.ENT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(50)},
	{Currency: currency.BAT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(40)},
	{Currency: currency.FIRST, Deposit: fee.Convert(0), Withdrawal: fee.Convert(30)},
	{Currency: currency.SAFE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.001)},
	{Currency: currency.QUN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(200)},
	{Currency: currency.BTN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.005)},
	{Currency: currency.TRUE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
	{Currency: currency.CDC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.DDM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.HOTC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(150)},
	{Currency: currency.USDT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
	{Currency: currency.XUC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.EPC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(40)},
	{Currency: currency.BDS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3)},
	{Currency: currency.GRAM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
	{Currency: currency.DOGE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.NEO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.OMG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.5)},
	{Currency: currency.BTM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
	{Currency: currency.SNT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(60)},
	{Currency: currency.AE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3)},
	{Currency: currency.ICX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3)},
	{Currency: currency.ZRX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.EDO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
	{Currency: currency.FUN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(250)},
	{Currency: currency.MANA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(70)},
	{Currency: currency.RCN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(70)},
	{Currency: currency.MCO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.6)},
	{Currency: currency.MITH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.KNC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
	{Currency: currency.XLM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
	{Currency: currency.GNT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.MTL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3)},
	{Currency: currency.SUB, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.XEM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
	{Currency: currency.EOSDAC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.KAN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(350)},
	{Currency: currency.AAA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.XWC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.PDX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.SLT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
	{Currency: currency.ADA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.HPY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
	{Currency: currency.PAX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
	{Currency: currency.XTZ, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
}

// orderSideMap holds order type info based on Alphapoint data
var orderSideMap = map[int64]order.Side{
	0: order.Buy,
	1: order.Sell,
}

// TradeHistory defines a slice of historic trades
type TradeHistory []struct {
	Amount    float64 `json:"amount,string"`
	Date      int64   `json:"date"`
	Price     float64 `json:"price,string"`
	Tid       int64   `json:"tid"`
	TradeType string  `json:"trade_type"`
	Type      string  `json:"type"`
}
