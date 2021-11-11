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
	{Currency: currency.ZB, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(10, 2e6, 0)},
	{Currency: currency.BTC, Deposit: fee.ConvertWithMinimumAmount(0.0001, 0), Withdrawal: fee.ConvertWithMaxAndMin(0.0005, 60, 0)},
	{Currency: currency.LTC, Deposit: fee.ConvertWithMinimumAmount(0.005, 0), Withdrawal: fee.ConvertWithMaxAndMin(0.005, 20000, 0)},
	{Currency: currency.ETH, Deposit: fee.ConvertWithMinimumAmount(0.008, 0), Withdrawal: fee.ConvertWithMaxAndMin(0.01, 1000, 0)},
	{Currency: currency.ETC, Deposit: fee.ConvertWithMinimumAmount(0.01, 0), Withdrawal: fee.ConvertWithMaxAndMin(0.01, 200000, 0)},
	{Currency: currency.BTS, Deposit: fee.ConvertWithMinimumAmount(1, 0), Withdrawal: fee.ConvertWithMaxAndMin(3, 10e6, 0)},
	{Currency: currency.EOS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 20000, 0)},
	{Currency: currency.QTUM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.01, 2e5, 0)},
	{Currency: currency.HC, Deposit: fee.ConvertWithMinimumAmount(0.011, 0), Withdrawal: fee.ConvertWithMaxAndMin(0.01, 200000, 0)},
	{Currency: currency.XRP, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 200000, 0)},
	{Currency: currency.QCASH, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(5, 2000000, 0)},
	{Currency: currency.DASH, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.002, 2000, 0)},
	{Currency: currency.BCD, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0, 4000, 0)},
	{Currency: currency.UBTC, Deposit: fee.ConvertWithMinimumAmount(0.1, 0), Withdrawal: fee.ConvertWithMaxAndMin(0.001, 200000, 0)},
	{Currency: currency.SBTC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0, 400, 0)},
	{Currency: currency.INK, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(60, 2000000, 0)},
	{Currency: currency.TV, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1, 2000000, 0)},
	{Currency: currency.BTH, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 2000000, 0)},
	{Currency: currency.BCX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0, 400, 0)},
	{Currency: currency.LBTC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.0001, 200000, 0)},
	{Currency: currency.CHAT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(2000, 2000000, 0)},
	{Currency: currency.BITCNY, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(20, 1000000, 0)},
	{Currency: currency.HLC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(100, 2000000, 0)},
	{Currency: currency.BCW, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(30, 200, 0)},
	{Currency: currency.BTP, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.001, 200000, 0)},
	{Currency: currency.TOPC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(2000, 2000000, 0)},
	{Currency: currency.ENTC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(10, 2000000, 0)},
	{Currency: currency.BAT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(40, 2000000, 0)},
	{Currency: currency.SAFE, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1, 20000, 0)},
	{Currency: currency.QUN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(200, 2000000, 0)},
	{Currency: currency.BTN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.005, 1000000, 0)},
	{Currency: currency.TRUE, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1, 1000000, 0)},
	{Currency: currency.CDC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1000, 20000000, 0)},
	{Currency: currency.DDM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.2, 20000000, 0)},
	{Currency: currency.HOTC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(150, 2000000, 0)},
	{Currency: currency.USDT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(15, 400000, 0)},
	{Currency: currency.XUC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1, 200000, 0)},
	{Currency: currency.EPC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(40, 2000000, 0)},
	{Currency: currency.BDS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(100, 10000000, 0)},
	{Currency: currency.GRAM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1000, 20000000, 0)},
	{Currency: currency.DOGE, Deposit: fee.ConvertWithMinimumAmount(10, 0), Withdrawal: fee.ConvertWithMaxAndMin(20, 20000000, 0)},
	{Currency: currency.NEO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0, 20000, 0)},
	{Currency: currency.OMG, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(3, 20000, 0)},
	{Currency: currency.BTM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(10, 400000, 0)},
	{Currency: currency.SNT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(60, 2000000, 0)},
	{Currency: currency.AE, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0, 100000, 0)},
	{Currency: currency.ICX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(3, 100000, 0)},
	{Currency: currency.ZRX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(10, 200000, 0)},
	{Currency: currency.EDO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(4, 100000, 0)},
	{Currency: currency.FUN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(250, 10000000, 0)},
	{Currency: currency.MANA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(20, 4000000, 0)},
	{Currency: currency.RCN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(70, 4000000, 0)},
	{Currency: currency.MCO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.6, 40000, 0)},
	{Currency: currency.MITH, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(500, 200000, 0)},
	{Currency: currency.KNC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(5, 200000, 0)},
	{Currency: currency.XLM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 1000000, 0)},
	{Currency: currency.GLM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(20, 200000, 0)},
	{Currency: currency.MTL, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(3, 40000, 0)},
	{Currency: currency.SUB, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(20, 200000, 0)},
	{Currency: currency.XEM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(4, 2e6, 0)},
	{Currency: currency.EOSDAC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0, 100, 0)},
	{Currency: currency.KAN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(350, 20000000, 0)},
	{Currency: currency.AAA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1500, 2000000, 0)},
	{Currency: currency.XWCC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1, 2000000, 0)},
	{Currency: currency.PDX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1300, 2000000, 0)},
	{Currency: currency.SLT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(100, 200000, 0)},
	{Currency: currency.ADA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1, 2000000, 0)},
	{Currency: currency.HPY, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(100, 10000000, 0)},
	{Currency: currency.PAX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(5, 200000, 0)},
	{Currency: currency.XTZ, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 200000, 0)},
	{Currency: currency.BRC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(5, 200000, 0)},
	{Currency: currency.BCH, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.0005, 500, 0)},
	{Currency: currency.BSV, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.001, 600, 0)},
	{Currency: currency.VSYS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1, 1000000, 0)},
	{Currency: currency.GRIN, Deposit: fee.ConvertWithMinimumAmount(0.0000001, 0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 20000, 0)},
	{Currency: currency.TRX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1, 1000000, 0)},
	{Currency: currency.TUSD, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(5, 200000, 0)},
	{Currency: currency.XMR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.001, 100, 0)},
	{Currency: currency.LEO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(5, 200000, 0)},
	{Currency: currency.B91, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(4, 200000, 0)},
	{Currency: currency.YTNB, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(8, 1000000, 0)},
	{Currency: currency.NWT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(20, 1000000, 0)},
	{Currency: currency.ETZ, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.01, 200000, 0)},
	{Currency: currency.BAR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.5, 2000000, 0)},
	{Currency: currency.ACC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 500000, 0)},
	{Currency: currency.HX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 1000000, 0)},
	{Currency: currency.LVN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(250, 1000000, 0)},
	{Currency: currency.TSR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(800, 200000, 0)},
	{Currency: currency.FN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0, 200000, 0)},
	{Currency: currency.CRO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(15, 2000000, 0)},
	{Currency: currency.XWC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.01, 200000, 0)},
	{Currency: currency.GUSD, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(5, 400000, 0)},
	{Currency: currency.USDC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(5, 400000, 0)},
	{Currency: currency.DNA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(20, 10000000, 0)},
	{Currency: currency.LUCKY, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(20, 2000000, 0)},
	{Currency: currency.HNS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.2, 200000, 0)},
	{Currency: currency.KPG, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.05, 200000, 0)},
	{Currency: currency.LTG, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.6, 10000, 0)},
	{Currency: currency.UFO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.01, 10000, 0)},
	{Currency: currency.GUCS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.01, 10000, 0)},
	{Currency: currency.VBT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(250, 2000000, 0)},
	{Currency: currency.DSF, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(55, 400000, 0)},
	{Currency: currency.GST, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.01, 50000, 0)},
	{Currency: currency.DAWN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(3, 2000000, 0)},
	{Currency: currency.DOT, Deposit: fee.ConvertWithMinimumAmount(0.1, 0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 100000, 0)},
	{Currency: currency.SWFTC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(3000, 20000000, 0)},
	{Currency: currency.UFC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(10, 1000000, 0)},
	{Currency: currency.CENNZ, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(5, 2000000, 0)},
	{Currency: currency.EP, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(2, 5000000, 0)},
	{Currency: currency.YFI, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.0005, 5, 0)},
	{Currency: currency.YFII, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.0001, 20, 0)},
	{Currency: currency.ULU, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.05, 500, 0)},
	{Currency: currency.DMD, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.03, 500, 0)},
	{Currency: currency.SUSHI, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(2, 10000, 0)},
	{Currency: currency.NBS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(5, 15000000, 0)},
	{Currency: currency.BGPT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(200, 3000000, 0)},
	{Currency: currency.DIP, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1.5, 4000000, 0)},
	{Currency: currency.SWRV, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 100, 0)},
	{Currency: currency.AAVE, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 300000, 0)},
	{Currency: currency.LINK, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.5, 20000, 0)},
	{Currency: currency.ONT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1, 200000, 0)},
	{Currency: currency.UNI, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1, 50000, 0)},
	{Currency: currency.QFIL, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.05, 1000, 0)},
	{Currency: currency.FIL, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.002, 50000, 0)},
	{Currency: currency.RTF, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.05, 3000, 0)},
	{Currency: currency.M, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.01, 100, 0)},
	{Currency: currency.FOMP, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.05, 1000, 0)},
	{Currency: currency.BDM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(5, 50000, 0)},
	{Currency: currency.ATOM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.01, 5000, 0)},
	{Currency: currency.LPT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1, 5000, 0)},
	{Currency: currency.DNT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(60, 500000, 0)},
	{Currency: currency.DORA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1.5, 10000, 0)},
	{Currency: currency.CRV, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(10, 400000, 0)},
	{Currency: currency.ANKR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(200, 10000000, 0)},
	{Currency: currency.STORJ, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(12, 500000, 0)},
	{Currency: currency.ENJ, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(5, 500000, 0)},
	{Currency: currency.KSM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.01, 300, 0)},
	{Currency: currency.GRT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(15, 100000, 0)},
	{Currency: currency.COMP, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.05, 500, 0)},
	{Currency: currency.OGN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(6, 100000, 0)},
	{Currency: currency.DENT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1400, 10000000, 0)},
	{Currency: currency.POND, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(80, 1000000, 0)},
	{Currency: currency.ONEINCH, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(3, 30000, 0)},
	{Currency: currency.NKN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(25, 200000, 0)},
	{Currency: currency.MATIC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(40, 500000, 0)},
	{Currency: currency.SAND, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(25, 200000, 0)},
	{Currency: currency.CELR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(200, 5000000, 0)},
	{Currency: currency.UZ, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 100000, 0)},
	{Currency: currency.BKH, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1, 200000, 0)},
	{Currency: currency.CHZ, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(10, 250000, 0)},
	{Currency: currency.CRU, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 2000, 0)},
	{Currency: currency.IDV, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(230, 2000000, 0)},
	{Currency: currency.NEAR, Deposit: fee.ConvertWithMinimumAmount(1, 0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 150000, 0)},
	{Currency: currency.XYM, Deposit: fee.ConvertWithMinimumAmount(1, 0), Withdrawal: fee.ConvertWithMaxAndMin(5, 400000, 0)},
	{Currency: currency.DFL, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(10, 100000, 0)},
	{Currency: currency.UMA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.8, 10000, 0)},
	{Currency: currency.RSR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(20, 2500000, 0)},
	{Currency: currency.CSPR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 5000000, 0)},
	{Currency: currency.XCH, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.0013, 5000, 0)},
	{Currency: currency.FORTH, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0, 30000, 0)},
	{Currency: currency.MIR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(2, 100000, 0)},
	{Currency: currency.SHIB, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(2600000, 20000000000, 0)},
	{Currency: currency.AKITA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(2600000, 20000000000, 0)},
	{Currency: currency.SOL, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.01, 5000, 0)},
	{Currency: currency.BED, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.5, 100000, 0)},
	{Currency: currency.SDOG, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(10000000, 20000000000, 0)},
	{Currency: currency.CFX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.5, 200000, 0)},
	{Currency: currency.ONETHOUSANDHOKK, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(20000000, 40000000000, 0)},
	{Currency: currency.ONETHOUSANDKISHU, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(4000000, 10000000000, 0)},
	{Currency: currency.CATE, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(4500000, 40000000000, 0)},
	{Currency: currency.XFLR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0, 0, 0)},
	{Currency: currency.ICP, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.01, 10000, 0)},
	{Currency: currency.BNA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(200, 3000000, 0)},
	{Currency: currency.GTC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(5, 20000, 0)},
	{Currency: currency.THETA, Deposit: fee.ConvertWithMinimumAmount(0.01, 0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 10000, 0)},
	{Currency: currency.BZZ, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1, 10000, 0)},
	{Currency: currency.DOM, Deposit: fee.ConvertWithMinimumAmount(0.01, 0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 23000, 0)},
	{Currency: currency.LAT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1, 400000, 0)},
	{Currency: currency.AMP, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(600, 1000000, 0)},
	{Currency: currency.MLN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.2, 1000, 0)},
	{Currency: currency.POLS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(20, 100000, 0)},
	{Currency: currency.POLY, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(120, 500000, 0)},
	{Currency: currency.KEEP, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(45, 200000, 0)},
	{Currency: currency.ALGO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.2, 500000, 0)},
	{Currency: currency.O3, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(5, 20000, 0)},
	{Currency: currency.BNT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(6, 50000, 0)},
	{Currency: currency.LIKE, Deposit: fee.ConvertWithMinimumAmount(0.01, 0), Withdrawal: fee.ConvertWithMaxAndMin(1, 5000000, 0)},
	{Currency: currency.DAI, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0, 150000, 0)},
	{Currency: currency.QNT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0, 900, 0)},
	{Currency: currency.BOND, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0, 5000, 0)},
	{Currency: currency.MNC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(9000, 50000000, 0)},
	{Currency: currency.KAVA, Deposit: fee.ConvertWithMinimumAmount(0.0001, 0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 30000, 0)},
	{Currency: currency.CART, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(20, 400000, 0)},
	{Currency: currency.AXS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.2, 3000, 0)},
	{Currency: currency.CLV, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(8, 100000, 0)},
	{Currency: currency.XEC, Deposit: fee.ConvertWithMinimumAmount(5000, 0), Withdrawal: fee.ConvertWithMaxAndMin(500, 2000000000, 0)},
	{Currency: currency.YGG, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(6, 70000, 0)},
	{Currency: currency.FARM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.05, 700, 0)},
	{Currency: currency.ACH, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(100, 1500000, 0)},
	{Currency: currency.EFI, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(20, 150000, 0)},
	{Currency: currency.ROSE, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.001, 1500000, 0)},
	{Currency: currency.PLA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(20, 150000, 0)},
	{Currency: currency.RAI, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(6, 50000, 0)},
	{Currency: currency.SLP, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(105, 500000, 0)},
	{Currency: currency.ORN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(2, 13000, 0)},
	{Currency: currency.QUICK, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.03, 200, 0)},
	{Currency: currency.TRU, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(26, 180000, 0)},
	{Currency: currency.REQ, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(77, 500000, 0)},
	{Currency: currency.LUNA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.01, 30000, 0)},
	{Currency: currency.SANA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(666, 4000000, 0)},
	{Currency: currency.TRIBE, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(29, 200000, 0)},
	{Currency: currency.AUDIO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(6, 50000, 0)},
	{Currency: currency.RAD, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(2, 20000, 0)},
	{Currency: currency.CELO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.001, 40000, 0)},
	{Currency: currency.RARI, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1, 4000, 0)},
	{Currency: currency.SRM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(2.5, 20000, 0)},
	{Currency: currency.RARE, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(15, 50000, 0)},
	{Currency: currency.COTI, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0, 400000, 0)},
	{Currency: currency.TLM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(60, 500000, 0)},
	{Currency: currency.RLY, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(30, 250000, 0)},
	{Currency: currency.SDN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1, 25000, 0)},
	{Currency: currency.FET, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0, 200000, 0)},
	{Currency: currency.WNCG, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(20, 70000, 0)},
	{Currency: currency.AGLD, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(10, 50000, 0)},
	{Currency: currency.IOTX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0.1, 10000000, 0)},
	{Currency: currency.DDX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(0, 16000, 0)},
	{Currency: currency.AMC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(48, 180000, 0)},
	{Currency: currency.DYDX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(5, 11000, 0)},
	{Currency: currency.OOE, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(10, 150000, 0)},
	{Currency: currency.WAXP, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(5, 400000, 0)},
	{Currency: currency.XYO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(1000, 3200000, 0)},
	{Currency: currency.RGT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(2.3, 8600, 0)},
	{Currency: currency.NU, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(75, 400000, 0)},
	{Currency: currency.GALA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(200, 1200000, 0)},
	{Currency: currency.ZKN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMaxAndMin(3, 310000, 0)},
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

// FeeInformation defines fee information
type FeeInformation struct {
	ChainName     string  `json:"chainName"`
	Fee           float64 `json:"fee"`
	MainChainName string  `json:"mainChainName"`
	CanDeposit    bool    `json:"canDeposit"`
	CanWithdraw   bool    `json:"canWithdraw"`
}

// AllFeeInformation defines fee information for all currencies
type AllFeeInformation map[string][]FeeInformation
