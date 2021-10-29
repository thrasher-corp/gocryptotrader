package gateio

import (
	"encoding/json"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fee"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// TimeInterval Interval represents interval enum.
type TimeInterval int

// TimeInterval vars
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

// MarketInfoResponse holds the market info data
type MarketInfoResponse struct {
	Result string                    `json:"result"`
	Pairs  []MarketInfoPairsResponse `json:"pairs"`
}

// MarketInfoPairsResponse holds the market info response data
type MarketInfoPairsResponse struct {
	Symbol string
	// DecimalPlaces symbol price accuracy
	DecimalPlaces float64
	// MinAmount minimum order amount
	MinAmount float64
	// Fee transaction fee
	Fee float64
}

// BalancesResponse holds the user balances
type BalancesResponse struct {
	Result    string      `json:"result"`
	Available interface{} `json:"available"`
	Locked    interface{} `json:"locked"`
}

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol   string // Required field; example LTCBTC,BTCUSDT
	HourSize int    // How many hours of data
	GroupSec string
}

// KLineResponse holds the kline response data
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

// TickerResponse  holds the ticker response data
type TickerResponse struct {
	Period      int64   `json:"period"`
	BaseVolume  float64 `json:"baseVolume,string"`
	Change      float64 `json:"change,string"`
	Close       float64 `json:"close,string"`
	High        float64 `json:"high,string"`
	Last        float64 `json:"last,string"`
	Low         float64 `json:"low,string"`
	Open        float64 `json:"open,string"`
	QuoteVolume float64 `json:"quoteVolume,string"`
}

// OrderbookResponse stores the orderbook data
type OrderbookResponse struct {
	Result  string `json:"result"`
	Elapsed string `json:"elapsed"`
	Asks    [][]string
	Bids    [][]string
}

// OrderbookItem stores an orderbook item
type OrderbookItem struct {
	Price  float64
	Amount float64
}

// Orderbook stores the orderbook data
type Orderbook struct {
	Result  string
	Elapsed string
	Bids    []OrderbookItem
	Asks    []OrderbookItem
}

// SpotNewOrderRequestParams Order params
type SpotNewOrderRequestParams struct {
	Amount float64 `json:"amount"` // Order quantity
	Price  float64 `json:"price"`  // Order price
	Symbol string  `json:"symbol"` // Trading pair; btc_usdt, eth_btc......
	Type   string  `json:"type"`   // Order type (buy or sell),
}

// SpotNewOrderResponse Order response
type SpotNewOrderResponse struct {
	OrderNumber  int64       `json:"orderNumber"`         // OrderID number
	Price        float64     `json:"rate,string"`         // Order price
	LeftAmount   float64     `json:"leftAmount,string"`   // The remaining amount to fill
	FilledAmount float64     `json:"filledAmount,string"` // The filled amount
	Filledrate   interface{} `json:"filledRate"`          // FilledPrice. if we send a market order, the exchange returns float64.
	//			  if we set a limit order, which will remain in the order book, the exchange will return the string
}

// OpenOrdersResponse the main response from GetOpenOrders
type OpenOrdersResponse struct {
	Code    int         `json:"code"`
	Elapsed string      `json:"elapsed"`
	Message string      `json:"message"`
	Orders  []OpenOrder `json:"orders"`
	Result  string      `json:"result"`
}

// OpenOrder details each open order
type OpenOrder struct {
	Amount        float64 `json:"amount,string"`
	CurrencyPair  string  `json:"currencyPair"`
	FilledAmount  float64 `json:"filledAmount,string"`
	FilledRate    float64 `json:"filledRate"`
	InitialAmount float64 `json:"initialAmount"`
	InitialRate   float64 `json:"initialRate"`
	OrderNumber   string  `json:"orderNumber"`
	Rate          float64 `json:"rate"`
	Status        string  `json:"status"`
	Timestamp     int64   `json:"timestamp"`
	Total         float64 `json:"total,string"`
	Type          string  `json:"type"`
}

// TradHistoryResponse The full response for retrieving all user trade history
type TradHistoryResponse struct {
	Code    int              `json:"code,omitempty"`
	Elapsed string           `json:"elapsed,omitempty"`
	Message string           `json:"message"`
	Trades  []TradesResponse `json:"trades"`
	Result  string           `json:"result"`
}

// TradesResponse details trade history
type TradesResponse struct {
	ID       int64   `json:"tradeID"`
	OrderID  int64   `json:"orderNumber"`
	Pair     string  `json:"pair"`
	Type     string  `json:"type"`
	Side     string  `json:"side"`
	Rate     float64 `json:"rate,string"`
	Amount   float64 `json:"amount,string"`
	Total    float64 `json:"total"`
	Time     string  `json:"date"`
	TimeUnix int64   `json:"time_unix"`
}

// transferFees the large list of predefined transfer fees. Prone to change.
var transferFees = []fee.Transfer{
	{Currency: currency.GT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.25)},
	{Currency: currency.USDT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(40)},
	{Currency: currency.BTC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.001)},
	{Currency: currency.BSV, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.012)},
	{Currency: currency.ETC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.038)},
	{Currency: currency.XRP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.8)},
	{Currency: currency.ZEC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.017)},
	{Currency: currency.QTUM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.15)},
	{Currency: currency.DOGE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.BTM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(39)},
	{Currency: currency.ONT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.BAT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(51)},
	{Currency: currency.BTM_ETH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(690)},
	{Currency: currency.REQ, Deposit: fee.Convert(0), Withdrawal: fee.Convert(160)},
	{Currency: currency.KNC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(23)},
	{Currency: currency.CDT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(380)},
	{Currency: currency.INK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2700)},
	{Currency: currency.WTC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.2)},
	{Currency: currency.RCN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2000)},
	{Currency: currency.BNT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(14)},
	{Currency: currency.MCO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
	{Currency: currency.FUN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2200)},
	{Currency: currency.RLC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(9.5)},
	{Currency: currency.WINGS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
	{Currency: currency.KICK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
	{Currency: currency.HSR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.05)},
	{Currency: currency.QBT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(280)},
	{Currency: currency.BCD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.86)},
	{Currency: currency.QASH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(490)},
	{Currency: currency.GNT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.SBTC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.4)},
	{Currency: currency.GOD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
	{Currency: currency.SMT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(780)},
	{Currency: currency.IOTA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
	{Currency: currency.NAS_ETH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(94)},
	{Currency: currency.ADA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.BIFI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1300)},
	{Currency: currency.BNTY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(51000)},
	{Currency: currency.LEND, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.ELF, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.1)},
	{Currency: currency.FUEL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(83000)},
	{Currency: currency.GTC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(14000)},
	{Currency: currency.QUN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
	{Currency: currency.DDD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(14000)},
	{Currency: currency.BTO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1600)},
	{Currency: currency.THETA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.34)},
	{Currency: currency.ZIL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(22)},
	{Currency: currency.TNC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1800)},
	{Currency: currency.ZPT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2300)},
	{Currency: currency.GXS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.3)},
	{Currency: currency.BLZ, Deposit: fee.Convert(0), Withdrawal: fee.Convert(160)},
	{Currency: currency.DADI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
	{Currency: currency.LEDU, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.XLM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
	{Currency: currency.NEO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.DBC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(320)},
	{Currency: currency.MKR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.015)},
	{Currency: currency.LRC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(91)},
	{Currency: currency.ZRX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(35)},
	{Currency: currency.TNT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.DNT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(500)},
	{Currency: currency.BCDN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(35000)},
	{Currency: currency.OMG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.4)},
	{Currency: currency.EON, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.EOSDAC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1300)},
	{Currency: currency.CNC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
	{Currency: currency.XMR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0071)},
	{Currency: currency.XTC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
	{Currency: currency.USDG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(42)},
	{Currency: currency.SYS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.NYZO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.ETH2, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.005)},
	{Currency: currency.KAVA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.34)},
	{Currency: currency.ANT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.5)},
	{Currency: currency.STPT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(200)},
	{Currency: currency.RSV, Deposit: fee.Convert(0), Withdrawal: fee.Convert(42)},
	{Currency: currency.CTSI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(57)},
	{Currency: currency.OCEAN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(64)},
	{Currency: currency.KSM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0077)},
	{Currency: currency.DOT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
	{Currency: currency.MTRG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
	{Currency: currency.COTI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(70)},
	{Currency: currency.DIGG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.00083)},
	{Currency: currency.YAMV1, Deposit: fee.Convert(0), Withdrawal: fee.Convert(500)},
	{Currency: currency.LUNA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.13)},
	{Currency: currency.FSN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.4)},
	{Currency: currency.BZRX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(320)},
	{Currency: currency.YAMV2, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
	{Currency: currency.BOX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.35)},
	{Currency: currency.UNI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.7)},
	{Currency: currency.AAVE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.44)},
	{Currency: currency.ERG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.5)},
	{Currency: currency.KPHA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.KAR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.25)},
	{Currency: currency.RMRK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.3)},
	{Currency: currency.CRING, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.PICA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.XRT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.TEER, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.SGB, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.KPN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.CSM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.KAZE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.SASHIMI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1300)},
	{Currency: currency.SWRV, Deposit: fee.Convert(0), Withdrawal: fee.Convert(55)},
	{Currency: currency.AUCTION, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.3)},
	{Currency: currency.OIN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
	{Currency: currency.ADEL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1000)},
	{Currency: currency.KIMCHI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1100)},
	{Currency: currency.RING, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1000)},
	{Currency: currency.CREAM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.29)},
	{Currency: currency.DEGO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.SFG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(160)},
	{Currency: currency.CORE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.NU, Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
	{Currency: currency.ARNX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(820)},
	{Currency: currency.ROSE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.COVER, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.38)},
	{Currency: currency.BASE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
	{Currency: currency.HEGIC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(220)},
	{Currency: currency.DUSK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(250)},
	{Currency: currency.UNFI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.8)},
	{Currency: currency.GHST, Deposit: fee.Convert(0), Withdrawal: fee.Convert(57)},
	{Currency: currency.ACH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(610)},
	{Currency: currency.GRT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(52)},
	{Currency: currency.ALEPH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(65)},
	{Currency: currency.FXS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(11)},
	{Currency: currency.BORING, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3900)},
	{Currency: currency.BAC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(980)},
	{Currency: currency.LON, Deposit: fee.Convert(0), Withdrawal: fee.Convert(16)},
	{Currency: currency.WOZX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(94)},
	{Currency: currency.POND, Deposit: fee.Convert(0), Withdrawal: fee.Convert(410)},
	{Currency: currency.DSD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3400)},
	{Currency: currency.SHARE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4000)},
	{Currency: currency.ONC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(130)},
	{Currency: currency.ZKS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(68)},
	{Currency: currency.MIS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(30)},
	{Currency: currency.ONX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(69)},
	{Currency: currency.RIF, Deposit: fee.Convert(0), Withdrawal: fee.Convert(8)},
	{Currency: currency.PROPS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2900)},
	{Currency: currency.LAYER, Deposit: fee.Convert(0), Withdrawal: fee.Convert(48)},
	{Currency: currency.QNT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.12)},
	{Currency: currency.YOP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
	{Currency: currency.BONDED, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1300)},
	{Currency: currency.ROOM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.BUSD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(46)},
	{Currency: currency.UNISTAKE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1600)},
	{Currency: currency.FXF, Deposit: fee.Convert(0), Withdrawal: fee.Convert(130)},
	{Currency: currency.TORN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.58)},
	{Currency: currency.UMB, Deposit: fee.Convert(0), Withdrawal: fee.Convert(78)},
	{Currency: currency.JASMY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(490)},
	{Currency: currency.BEL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.SAND, Deposit: fee.Convert(0), Withdrawal: fee.Convert(46)},
	{Currency: currency.AMP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1900)},
	{Currency: currency.BONDLY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(560)},
	{Currency: currency.BMI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(130)},
	{Currency: currency.SUPER, Deposit: fee.Convert(0), Withdrawal: fee.Convert(52)},
	{Currency: currency.RAY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.52)},
	{Currency: currency.POLIS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.62)},
	{Currency: currency.WAG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.2)},
	{Currency: currency.CYS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.9)},
	{Currency: currency.SLRS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.4)},
	{Currency: currency.LIKE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(12)},
	{Currency: currency.PRT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(450)},
	{Currency: currency.SUNNY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(170)},
	{Currency: currency.MNGO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(19)},
	{Currency: currency.STEP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
	{Currency: currency.FIDA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.8)},
	{Currency: currency.AQT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(9.9)},
	{Currency: currency.PBR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
	{Currency: currency.HOPR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(340)},
	{Currency: currency.PROM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.9)},
	{Currency: currency.TVK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(200)},
	{Currency: currency.A5T, Deposit: fee.Convert(0), Withdrawal: fee.Convert(690)},
	{Currency: currency.CUDOS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1500)},
	{Currency: currency.COMBO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
	{Currency: currency.DOWS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
	{Currency: currency.KYL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(170)},
	{Currency: currency.EXRD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(17)},
	{Currency: currency.UTK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(96)},
	{Currency: currency.ETHA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(240)},
	{Currency: currency.ALN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(730)},
	{Currency: currency.AUDIO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.CHR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(99)},
	{Currency: currency.HAPI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.56)},
	{Currency: currency.BLANK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(190)},
	{Currency: currency.ERN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.4)},
	{Currency: currency.KINE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
	{Currency: currency.FET, Deposit: fee.Convert(0), Withdrawal: fee.Convert(50)},
	{Currency: currency.UOS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(70)},
	{Currency: currency.ALICE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.1)},
	{Currency: currency.ZEE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(84)},
	{Currency: currency.POLC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(340)},
	{Currency: currency.XED, Deposit: fee.Convert(0), Withdrawal: fee.Convert(89)},
	{Currency: currency.RLY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(65)},
	{Currency: currency.ANC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(9.7)},
	{Currency: currency.DAFI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(830)},
	{Currency: currency.FIRE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2100)},
	{Currency: currency.TARA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(7500)},
	{Currency: currency.PCNT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(620)},
	{Currency: currency.DG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.14)},
	{Currency: currency.SPI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.81)},
	{Currency: currency.BANK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.57)},
	{Currency: currency.ATD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(39)},
	{Currency: currency.UMX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(38)},
	{Currency: currency.TIDAL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(8400)},
	{Currency: currency.LABS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5300)},
	{Currency: currency.OGN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(69)},
	{Currency: currency.BLES, Deposit: fee.Convert(0), Withdrawal: fee.Convert(420)},
	{Currency: currency.OVR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(76)},
	{Currency: currency.HGET, Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
	{Currency: currency.NOIA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
	{Currency: currency.COOK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(8100)},
	{Currency: currency.CFI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.5)},
	{Currency: currency.MTL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.FST, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.4)},
	{Currency: currency.AME, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2400)},
	{Currency: currency.STN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(79)},
	{Currency: currency.SHOPX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(380)},
	{Currency: currency.SHFT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(40)},
	{Currency: currency.RBC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
	{Currency: currency.VAI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(98)},
	{Currency: currency.FEI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(40)},
	{Currency: currency.XEND, Deposit: fee.Convert(0), Withdrawal: fee.Convert(150)},
	{Currency: currency.ADX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(68)},
	{Currency: currency.SUKU, Deposit: fee.Convert(0), Withdrawal: fee.Convert(70)},
	{Currency: currency.LTO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(140)},
	{Currency: currency.TOTM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(250)},
	{Currency: currency.BLY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1200)},
	{Currency: currency.LKR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(540)},
	{Currency: currency.PNK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(700)},
	{Currency: currency.RAZE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(310)},
	{Currency: currency.DUCK2, Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
	{Currency: currency.CEL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.4)},
	{Currency: currency.DDIM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.1)},
	{Currency: currency.FLY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1600)},
	{Currency: currency.TLM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(190)},
	{Currency: currency.DDOS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(81)},
	{Currency: currency.GS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(190)},
	{Currency: currency.RAGE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1500)},
	{Currency: currency.AKITA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(22000000)},
	{Currency: currency.FORTH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.1)},
	{Currency: currency.CARDS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.7)},
	{Currency: currency.HORD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(260)},
	{Currency: currency.WBTC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.00065)},
	{Currency: currency.ARES, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1600)},
	{Currency: currency.BOA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(550)},
	{Currency: currency.SUSD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
	{Currency: currency.SFI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.068)},
	{Currency: currency.TCP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(320)},
	{Currency: currency.BLACK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1200)},
	{Currency: currency.EZ, Deposit: fee.Convert(0), Withdrawal: fee.Convert(11)},
	{Currency: currency.VSO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(19)},
	{Currency: currency.XAVA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
	{Currency: currency.PNG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
	{Currency: currency.LOCG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(270)},
	{Currency: currency.WSIENNA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.4)},
	{Currency: currency.WEMIX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
	{Currency: currency.STBU, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1200)},
	{Currency: currency.DFND, Deposit: fee.Convert(0), Withdrawal: fee.Convert(31000)},
	{Currency: currency.FTT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.97)},
	{Currency: currency.GDT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
	{Currency: currency.GUM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(340)},
	{Currency: currency.PRARE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(710)},
	{Currency: currency.GYEN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1500)},
	{Currency: currency.METIS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.8)},
	{Currency: currency.BZZ, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.TENSET, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)}, // 10SET
	{Currency: currency.PDEX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.5)},
	{Currency: currency.FEAR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(28)},
	{Currency: currency.ELON, Deposit: fee.Convert(0), Withdrawal: fee.Convert(210000000)},
	{Currency: currency.NOA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3900)},
	{Currency: currency.DOP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(27)},
	{Currency: currency.NAOS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(28)},
	{Currency: currency.GITCOIN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.8)},
	{Currency: currency.XCAD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(17)},
	{Currency: currency.LSS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
	{Currency: currency.CVX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.9)},
	{Currency: currency.PHTR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(150)},
	{Currency: currency.APN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(960)},
	{Currency: currency.DFYN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(70)},
	{Currency: currency.LIME, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5600)},
	{Currency: currency.FORM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(320)},
	{Currency: currency.KEX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
	{Currency: currency.DLTA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(410)},
	{Currency: currency.DPR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(730)},
	{Currency: currency.CQT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(43)},
	{Currency: currency.OLY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(6400)},
	{Currency: currency.FUSE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(750)},
	{Currency: currency.MLN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.43)},
	{Currency: currency.SRK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(14000)},
	{Currency: currency.MM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.4)},
	{Currency: currency.BURP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(460)},
	{Currency: currency.CART, Deposit: fee.Convert(0), Withdrawal: fee.Convert(63)},
	{Currency: currency.C98, Deposit: fee.Convert(0), Withdrawal: fee.Convert(11)},
	{Currency: currency.DNXC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(85)},
	{Currency: currency.DERC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(21)},
	{Currency: currency.PLA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(41)},
	{Currency: currency.EFI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(53)},
	{Currency: currency.GAME, Deposit: fee.Convert(0), Withdrawal: fee.Convert(26)},
	{Currency: currency.HMT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(39)},
	{Currency: currency.SKT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(240)},
	{Currency: currency.SPHRI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(290)},
	{Currency: currency.BIT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(36)},
	{Currency: currency.RARE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(18)},
	{Currency: currency.ZLW, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.SKYRIM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(490)},
	{Currency: currency.OCT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(22)},
	{Currency: currency.ATA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(39)},
	{Currency: currency.PUSH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(24)},
	{Currency: currency.REVO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(130)},
	{Currency: currency.VENT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(93)},
	{Currency: currency.LDO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(36)},
	{Currency: currency.GEL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(17)},
	{Currency: currency.CTRC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(310)},
	{Currency: currency.ITGR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(140)},
	{Currency: currency.HOTCROSS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
	{Currency: currency.MOT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(540)},
	{Currency: currency.FOREX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(290)},
	{Currency: currency.OPUL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(34)},
	{Currency: currency.EVA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1300)},
	{Currency: currency.POLI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1700)},
	{Currency: currency.TAUR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1600)},
	{Currency: currency.EQX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
	{Currency: currency.RBN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(9.4)},
	{Currency: currency.PHM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(14000)},
	{Currency: currency.FLOKI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1000000)},
	{Currency: currency.CIRUS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(74)},
	{Currency: currency.DYDX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(8.6)},
	{Currency: currency.RGT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.6)},
	{Currency: currency.AGLD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
	{Currency: currency.DOGNFT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5100)},
	{Currency: currency.SOV, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.6)},
	{Currency: currency.URUS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.41)},
	{Currency: currency.CFG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(25)},
	{Currency: currency.TBTC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0027)},
	{Currency: currency.WOM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(140)},
	{Currency: currency.NFTX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.75)},
	{Currency: currency.ORAI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.3)},
	{Currency: currency.LIT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(14)},
	{Currency: currency.POOLZ, Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.2)},
	{Currency: currency.DODO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(36)},
	{Currency: currency.IPAD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(73)},
	{Currency: currency.OPIUM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(23)},
	{Currency: currency.REEF, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2300)},
	{Currency: currency.MAPS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(62)},
	{Currency: currency.MIR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(18)},
	{Currency: currency.ZCN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(130)},
	{Currency: currency.BAO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(190000)},
	{Currency: currency.DIS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.2)},
	{Currency: currency.PBTC35A, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.68)},
	{Currency: currency.NORD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(28)},
	{Currency: currency.FLOW, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.11)},
	{Currency: currency.ENJ, Deposit: fee.Convert(0), Withdrawal: fee.Convert(34)},
	{Currency: currency.FIN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(240)},
	{Currency: currency.PRQ, Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
	{Currency: currency.INJ, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3)},
	{Currency: currency.ROOBEE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(7100)},
	{Currency: currency.KP3R, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.16)},
	{Currency: currency.HYVE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(93)},
	{Currency: currency.RAMP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(190)},
	{Currency: currency.RARI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.8)},
	{Currency: currency.MPH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2600)},
	{Currency: currency.CVP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(29)},
	{Currency: currency.VALUE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(57)},
	{Currency: currency.YFII, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.013)},
	{Currency: currency.SXP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(24)},
	{Currency: currency.BAND, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.TROY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2800)},
	{Currency: currency.SPA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(700)},
	{Currency: currency.FOR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(640)},
	{Currency: currency.DIA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(30)},
	{Currency: currency.TRB, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.2)},
	{Currency: currency.PEARL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.059)},
	{Currency: currency.NFT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1500000)},
	{Currency: currency.SLM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.2)},
	{Currency: currency.TAI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(14)},
	{Currency: currency.JFI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.083)},
	{Currency: currency.YFI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0015)},
	{Currency: currency.DKA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(450)},
	{Currency: currency.DOS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1800)},
	{Currency: currency.SRM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.5)},
	{Currency: currency.LBK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4000)},
	{Currency: currency.ASD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(92)},
	{Currency: currency.SWOP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.13)},
	{Currency: currency.WEST, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.5)},
	{Currency: currency.HYDRA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.116)},
	{Currency: currency.OLT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(200)},
	{Currency: currency.XYM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.8)},
	{Currency: currency.LAT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(14)},
	{Currency: currency.STC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(17)},
	{Currency: currency.BU, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1100)},
	{Currency: currency.HNT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.15)},
	{Currency: currency.AKT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.66)},
	{Currency: currency.BTC3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.6)},
	{Currency: currency.COTI3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.3)},
	{Currency: currency.XCH3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
	{Currency: currency.IOST3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.3)},
	{Currency: currency.BZZ3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.3)},
	{Currency: currency.TRIBE3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.RAY3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.1)},
	{Currency: currency.AR3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.3)},
	{Currency: currency.ONE3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.HBAR3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.9)},
	{Currency: currency.CSPR3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.4)},
	{Currency: currency.SXP3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.2)},
	{Currency: currency.XEC3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.3)},
	{Currency: currency.LIT3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.4)},
	{Currency: currency.MINA3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.1)},
	{Currency: currency.GALA3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.7)},
	{Currency: currency.FTT3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.6)},
	{Currency: currency.C983L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.7)},
	{Currency: currency.DYDX3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.3)},
	{Currency: currency.MTL3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5.1)},
	{Currency: currency.FTM3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.SAND3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.4)},
	{Currency: currency.LUNA3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.5)},
	{Currency: currency.ALPHA3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.1)},
	{Currency: currency.RUNE3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.2)},
	{Currency: currency.ICP3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(8)},
	{Currency: currency.SHIB3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.ACH3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(12)},
	{Currency: currency.ALICE3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(11)},
	{Currency: currency.AXS3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.1)},
	{Currency: currency.MATIC3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
	{Currency: currency.BTC5L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.BCH5L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(21)},
	{Currency: currency.DOT5L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(39)},
	{Currency: currency.XRP5L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(17)},
	{Currency: currency.BSV5L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(27)},
	{Currency: currency.LTC5L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(23)},
	{Currency: currency.EOS5L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(25)},
	{Currency: currency.ETH5L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.9)},
	{Currency: currency.LINK3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(91)},
	{Currency: currency.KAVA3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(39)},
	{Currency: currency.EGLD3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5.7)},
	{Currency: currency.CHZ3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.5)},
	{Currency: currency.MKR3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(31)},
	{Currency: currency.LRC3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.9)},
	{Currency: currency.BAL3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(93)},
	{Currency: currency.JST3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(8)},
	{Currency: currency.SERO3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(14)},
	{Currency: currency.VET3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(12)},
	{Currency: currency.THETA3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(17)},
	{Currency: currency.ZIL3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(260)},
	{Currency: currency.GRIN3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
	{Currency: currency.BEAM3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(37)},
	{Currency: currency.SOL3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.SKL3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
	{Currency: currency.ONEINCH3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(8.3)}, // 1INCH3L
	{Currency: currency.LON3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(160)},
	{Currency: currency.DOGE3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.1)},
	{Currency: currency.GRT3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(250)},
	{Currency: currency.BNB3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.TRX3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(9.2)},
	{Currency: currency.ATOM3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(16)},
	{Currency: currency.AVAX3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.NEAR3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(28)},
	{Currency: currency.ROSE3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(25)},
	{Currency: currency.ZEN3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(25)},
	{Currency: currency.QTUM3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
	{Currency: currency.XLM3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(34)},
	{Currency: currency.XRP3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(58)},
	{Currency: currency.CFX3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.3)},
	{Currency: currency.OMG3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.ALGO3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.9)},
	{Currency: currency.WAVES3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.8)},
	{Currency: currency.NEO3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(26)},
	{Currency: currency.ONT3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(74)},
	{Currency: currency.ETC3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.8)},
	{Currency: currency.CVC3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(32)},
	{Currency: currency.SNX3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(93)},
	{Currency: currency.ADA3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.DASH3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(26)},
	{Currency: currency.AAVE3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(12)},
	{Currency: currency.SRM3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(14)},
	{Currency: currency.KSM3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.8)},
	{Currency: currency.BTM3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(320)},
	{Currency: currency.ZEC3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(75)},
	{Currency: currency.XMR3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(11)},
	{Currency: currency.AMPL3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.4)},
	{Currency: currency.CRV3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
	{Currency: currency.COMP3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(23)},
	{Currency: currency.YFII3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
	{Currency: currency.YFI3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
	{Currency: currency.HT3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(53)},
	{Currency: currency.OKB3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(19)},
	{Currency: currency.UNI3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(33)},
	{Currency: currency.DOT3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.3)},
	{Currency: currency.FIL3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5.9)},
	{Currency: currency.SUSHI3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.8)},
	{Currency: currency.ETH3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.1)},
	{Currency: currency.EOS3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.7)},
	{Currency: currency.BSV3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(130)},
	{Currency: currency.BCH3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
	{Currency: currency.LTC3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
	{Currency: currency.XTZ3L, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.9)},
	{Currency: currency.RVN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
	{Currency: currency.AR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.5)},
	{Currency: currency.SNK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.25)},
	{Currency: currency.NSDX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(8)},
	{Currency: currency.DCR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.02)},
	{Currency: currency.XMC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.4)},
	{Currency: currency.VELO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.6)},
	{Currency: currency.PPS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
	{Currency: currency.XPR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(70)},
	{Currency: currency.HIVE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.5)},
	{Currency: currency.BCHA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.051)},
	{Currency: currency.FLUX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.NAX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1300)},
	{Currency: currency.NBOT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2000)},
	{Currency: currency.PLY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.BEAM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.7)},
	{Currency: currency.IOST, Deposit: fee.Convert(0), Withdrawal: fee.Convert(37)},
	{Currency: currency.MINA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.47)},
	{Currency: currency.REP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
	{Currency: currency.STAR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.9)},
	{Currency: currency.ABBC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(18)},
	{Currency: currency.FIC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(30)},
	{Currency: currency.STOX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4400)},
	{Currency: currency.VIDYX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(35)},
	{Currency: currency.ARN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
	{Currency: currency.TFUEL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.CS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(50)},
	{Currency: currency.REM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(60000)},
	{Currency: currency.INSTAR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.ONG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.7)},
	{Currency: currency.BFT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3000)},
	{Currency: currency.SENC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(40000)},
	{Currency: currency.ELEC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(26000)},
	{Currency: currency.TFD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.HUR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
	{Currency: currency.SWTH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(140)},
	{Currency: currency.SOUL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
	{Currency: currency.ADD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.DOCK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(25)},
	{Currency: currency.RATING, Deposit: fee.Convert(0), Withdrawal: fee.Convert(160000)},
	{Currency: currency.HIT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(830000)},
	{Currency: currency.CNNS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(21000)},
	{Currency: currency.MBL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(190)},
	{Currency: currency.MIX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(9300)},
	{Currency: currency.LEO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(50)},
	{Currency: currency.BTCBEAR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(520000)},
	{Currency: currency.ETHBULL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.032)},
	{Currency: currency.EOSBEAR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2500000)},
	{Currency: currency.XRPBULL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4200)},
	{Currency: currency.WGRT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(580)},
	{Currency: currency.CORAL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(9.3)},
	{Currency: currency.KGC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(220000)},
	{Currency: currency.RUNE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.3)},
	{Currency: currency.CBK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(12)},
	{Currency: currency.OPA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1300)},
	{Currency: currency.MCRN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.72)},
	{Currency: currency.KABY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(34)},
	{Currency: currency.BP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.8)},
	{Currency: currency.SFUND, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.7)},
	{Currency: currency.ASTRO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(22)},
	{Currency: currency.ARV, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4500)},
	{Currency: currency.ROSN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(33)},
	{Currency: currency.CPHR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(280)},
	{Currency: currency.KWS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(66)},
	{Currency: currency.CTT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.BEEFI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0054)},
	{Currency: currency.BLIN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.8)},
	{Currency: currency.PING, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4300)},
	{Currency: currency.XPNET, Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
	{Currency: currency.BABY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.8)},
	{Currency: currency.OPS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(40)},
	{Currency: currency.RACA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1300)},
	{Currency: currency.HOD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
	{Currency: currency.OLYMPUS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(51000000)},
	{Currency: currency.BMON, Deposit: fee.Convert(0), Withdrawal: fee.Convert(32)},
	{Currency: currency.PVU, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.1)},
	{Currency: currency.FAN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(84)},
	{Currency: currency.SKILL, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.19)},
	{Currency: currency.SPS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(5.9)},
	{Currency: currency.HERO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(87)},
	{Currency: currency.FEVR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(410)},
	{Currency: currency.WEX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1900)},
	{Currency: currency.KALM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.1)},
	{Currency: currency.KPAD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(220)},
	{Currency: currency.BABYDOGE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.PIG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.FINE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
	{Currency: currency.BSCS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(16)},
	{Currency: currency.SAFEMARS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
	{Currency: currency.PSG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.3)},
	{Currency: currency.PET, Deposit: fee.Convert(0), Withdrawal: fee.Convert(41)},
	{Currency: currency.ALPACA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(7)},
	{Currency: currency.BRY, Deposit: fee.Convert(0), Withdrawal: fee.Convert(19)},
	{Currency: currency.CTK, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.1)},
	{Currency: currency.TOOLS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(30)},
	{Currency: currency.JULD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
	{Currency: currency.CAKE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.15)},
	{Currency: currency.BAKE, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.5)},
	{Currency: currency.FRA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(90)},
	{Currency: currency.TWT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.9)},
	{Currency: currency.CRO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(200)},
	{Currency: currency.WIN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(9300)},
	{Currency: currency.MTV, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2900)},
	{Currency: currency.ARPA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(430)},
	{Currency: currency.ALGO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.2)},
	{Currency: currency.CKB, Deposit: fee.Convert(0), Withdrawal: fee.Convert(130)},
	{Currency: currency.BXC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(89000)},
	{Currency: currency.USDC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
	{Currency: currency.GARD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(15000)},
	{Currency: currency.HPB, Deposit: fee.Convert(0), Withdrawal: fee.Convert(25)},
	{Currency: currency.FTI, Deposit: fee.Convert(0), Withdrawal: fee.Convert(21000)},
	{Currency: currency.LEMO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(6300)},
	{Currency: currency.PUNDIX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(35)},
	{Currency: currency.IOTX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(29)},
	{Currency: currency.LBA, Deposit: fee.Convert(0), Withdrawal: fee.Convert(14000)},
	{Currency: currency.OPEN, Deposit: fee.Convert(0), Withdrawal: fee.Convert(14000)},
	{Currency: currency.SKM, Deposit: fee.Convert(0), Withdrawal: fee.Convert(23000)},
	{Currency: currency.NANO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.UPP, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.TMT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.EDG, Deposit: fee.Convert(0), Withdrawal: fee.Convert(230)},
	{Currency: currency.EGLD, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.052)},
	{Currency: currency.CSPR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(18)},
	{Currency: currency.FIS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
	{Currency: currency.GO, Deposit: fee.Convert(0), Withdrawal: fee.Convert(63)},
	{Currency: currency.MDX, Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.2)},
	{Currency: currency.WAR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(46)},
	{Currency: currency.XNFT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(53)},
	{Currency: currency.BXH, Deposit: fee.Convert(0), Withdrawal: fee.Convert(56)},
	{Currency: currency.BAGS, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.25)},
	{Currency: currency.BNB, Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0043)},
	{Currency: currency.EDR, Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
	{Currency: currency.TCT, Deposit: fee.Convert(0), Withdrawal: fee.Convert(1200)},
	{Currency: currency.MXC, Deposit: fee.Convert(0), Withdrawal: fee.Convert(710)},
}

// WebsocketRequest defines the initial request in JSON
type WebsocketRequest struct {
	ID       int64                        `json:"id"`
	Method   string                       `json:"method"`
	Params   []interface{}                `json:"params"`
	Channels []stream.ChannelSubscription `json:"-"` // used for tracking associated channel subs on batched requests
}

// WebsocketResponse defines a websocket response from gateio
type WebsocketResponse struct {
	Time    int64             `json:"time"`
	Channel string            `json:"channel"`
	Error   WebsocketError    `json:"error"`
	Result  json.RawMessage   `json:"result"`
	ID      int64             `json:"id"`
	Method  string            `json:"method"`
	Params  []json.RawMessage `json:"params"`
}

// WebsocketError defines a websocket error type
type WebsocketError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

// WebsocketTicker defines ticker data
type WebsocketTicker struct {
	Period      int64   `json:"period"`
	Open        float64 `json:"open,string"`
	Close       float64 `json:"close,string"`
	High        float64 `json:"high,string"`
	Low         float64 `json:"Low,string"`
	Last        float64 `json:"last,string"`
	Change      float64 `json:"change,string"`
	QuoteVolume float64 `json:"quoteVolume,string"`
	BaseVolume  float64 `json:"baseVolume,string"`
}

// WebsocketTrade defines trade data
type WebsocketTrade struct {
	ID     int64   `json:"id"`
	Time   float64 `json:"time"`
	Price  float64 `json:"price,string"`
	Amount float64 `json:"amount,string"`
	Type   string  `json:"type"`
}

// WebsocketBalance holds a slice of WebsocketBalanceCurrency
type WebsocketBalance struct {
	Currency []WebsocketBalanceCurrency
}

// WebsocketBalanceCurrency contains currency name funds available and frozen
type WebsocketBalanceCurrency struct {
	Currency  string
	Available string `json:"available"`
	Locked    string `json:"freeze"`
}

// WebSocketOrderQueryResult data returned from a websocket ordre query holds slice of WebSocketOrderQueryRecords
type WebSocketOrderQueryResult struct {
	Error                      WebsocketError               `json:"error"`
	Limit                      int                          `json:"limit"`
	Offset                     int                          `json:"offset"`
	Total                      int                          `json:"total"`
	WebSocketOrderQueryRecords []WebSocketOrderQueryRecords `json:"records"`
}

// WebSocketOrderQueryRecords contains order information from a order.query websocket request
type WebSocketOrderQueryRecords struct {
	ID           int64   `json:"id"`
	Market       string  `json:"market"`
	User         int64   `json:"user"`
	Ctime        float64 `json:"ctime"`
	Mtime        float64 `json:"mtime"`
	Price        float64 `json:"price,string"`
	Amount       float64 `json:"amount,string"`
	Left         float64 `json:"left,string"`
	DealFee      float64 `json:"dealFee,string"`
	OrderType    int64   `json:"orderType"`
	Type         int64   `json:"type"`
	FilledAmount float64 `json:"filledAmount,string"`
	FilledTotal  float64 `json:"filledTotal,string"`
}

// WebsocketAuthenticationResponse contains the result of a login request
type WebsocketAuthenticationResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Result struct {
		Status string `json:"status"`
	} `json:"result"`
	ID int64 `json:"id"`
}

// wsGetBalanceRequest
type wsGetBalanceRequest struct {
	ID     int64    `json:"id"`
	Method string   `json:"method"`
	Params []string `json:"params"`
}

// WsGetBalanceResponse stores WS GetBalance response
type WsGetBalanceResponse struct {
	Error  WebsocketError                      `json:"error"`
	Result map[string]WsGetBalanceResponseData `json:"result"`
	ID     int64                               `json:"id"`
}

// WsGetBalanceResponseData contains currency data
type WsGetBalanceResponseData struct {
	Available float64 `json:"available,string"`
	Freeze    float64 `json:"freeze,string"`
}

type wsBalanceSubscription struct {
	Method     string                                `json:"method"`
	Parameters []map[string]WsGetBalanceResponseData `json:"params"`
	ID         int64                                 `json:"id"`
}

type wsOrderUpdate struct {
	ID     int64         `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// TradeHistory contains trade history data
type TradeHistory struct {
	Elapsed string              `json:"elapsed"`
	Result  bool                `json:"result,string"`
	Data    []TradeHistoryEntry `json:"data"`
}

// TradeHistoryEntry contains an individual trade
type TradeHistoryEntry struct {
	Amount    float64 `json:"amount,string"`
	Date      string  `json:"date"`
	Rate      float64 `json:"rate,string"`
	Timestamp int64   `json:"timestamp,string"`
	Total     float64 `json:"total,string"`
	TradeID   string  `json:"tradeID"`
	Type      string  `json:"type"`
}

// wsOrderbook defines a websocket orderbook
type wsOrderbook struct {
	Asks [][]string `json:"asks"`
	Bids [][]string `json:"bids"`
	ID   int64      `json:"id"`
}

// DepositAddr stores the deposit address info
type DepositAddr struct {
	Result              bool   `json:"result,string"`
	Code                int    `json:"code"`
	Message             string `json:"message"`
	Address             string `json:"addr"`
	Tag                 string
	MultichainAddresses []struct {
		Chain        string `json:"chain"`
		Address      string `json:"address"`
		PaymentID    string `json:"payment_id"`
		PaymentName  string `json:"payment_name"`
		ObtainFailed uint8  `json:"obtain_failed"`
	} `json:"multichain_addresses"`
}

// AccountFees defines fees and other account related information.
type AccountFees struct {
	UserID     int64   `json:"user_id"`
	TakerFee   float64 `json:"taker_fee,string"`
	MakerFee   float64 `json:"maker_fee,string"`
	GTDiscount bool    `json:"gt_discount"`
	GTTakerFee float64 `json:"gt_taker_fee,string"`
	GTMakerFee float64 `json:"gt_maker_fee,string"`
	LoanFee    float64 `json:"loan_fee,string"`
	PointType  float64 `json:"point_type,string"`
}
