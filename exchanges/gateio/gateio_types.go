package gateio

import (
	"time"

	"github.com/thrasher-/gocryptotrader/currency/symbol"
)

// SpotNewOrderRequestParamsType order type (buy or sell)
type SpotNewOrderRequestParamsType string

var (
	// SpotNewOrderRequestParamsTypeBuy buy order
	SpotNewOrderRequestParamsTypeBuy = SpotNewOrderRequestParamsType("buy")

	// SpotNewOrderRequestParamsTypeSell sell order
	SpotNewOrderRequestParamsTypeSell = SpotNewOrderRequestParamsType("sell")
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
	Result    string            `json:"result"`
	Available map[string]string `json:"available"`
	Locked    map[string]string `json:"locked"`
}

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol   string // Required field; example LTCBTC,BTCUSDT
	HourSize int    // How many hours of data
	GroupSec TimeInterval
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
	Result        string  `json:"result"`
	Volume        float64 `json:"baseVolume,string"`    // Trading volume
	High          float64 `json:"high24hr,string"`      // 24 hour high price
	Open          float64 `json:"highestBid,string"`    // Openening price
	Last          float64 `json:"last,string"`          // Last price
	Low           float64 `json:"low24hr,string"`       // 24 hour low price
	Close         float64 `json:"lowestAsk,string"`     // Closing price
	PercentChange float64 `json:"percentChange,string"` // Percentage change
	QuoteVolume   float64 `json:"quoteVolume,string"`   // Quote currency volume
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
	Amount float64                       `json:"amount"` // Order quantity
	Price  float64                       `json:"price"`  // Order price
	Symbol string                        `json:"symbol"` // Trading pair; btc_usdt, eth_btc......
	Type   SpotNewOrderRequestParamsType `json:"type"`   // Order type (buy or sell),
}

// SpotNewOrderResponse Order response
type SpotNewOrderResponse struct {
	OrderNumber  int64   `json:"orderNumber"`         // OrderID number
	Price        float64 `json:"rate,string"`         // Order price
	LeftAmount   float64 `json:"leftAmount,string"`   // The remaining amount to fill
	FilledAmount float64 `json:"filledAmount,string"` // The filled amount
	Filledrate   float64 `json:"filledRate,string"`   // FilledPrice
}

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change
var WithdrawalFees = map[string]float64{
	symbol.USDT:     10,
	symbol.USDT_ETH: 10,
	symbol.BTC:      0.001,
	symbol.BCH:      0.0006,
	symbol.BTG:      0.002,
	symbol.LTC:      0.002,
	symbol.ZEC:      0.001,
	symbol.ETH:      0.003,
	symbol.ETC:      0.01,
	symbol.DASH:     0.02,
	symbol.QTUM:     0.1,
	symbol.QTUM_ETH: 0.1,
	symbol.DOGE:     50,
	symbol.REP:      0.1,
	symbol.BAT:      10,
	symbol.SNT:      30,
	symbol.BTM:      10,
	symbol.BTM_ETH:  10,
	symbol.CVC:      5,
	symbol.REQ:      20,
	symbol.RDN:      1,
	symbol.STX:      3,
	symbol.KNC:      1,
	symbol.LINK:     8,
	symbol.FIL:      0.1,
	symbol.CDT:      20,
	symbol.AE:       1,
	symbol.INK:      10,
	symbol.BOT:      5,
	symbol.POWR:     5,
	symbol.WTC:      0.2,
	symbol.VET:      10,
	symbol.RCN:      5,
	symbol.PPT:      0.1,
	symbol.ARN:      2,
	symbol.BNT:      0.5,
	symbol.VERI:     0.005,
	symbol.MCO:      0.1,
	symbol.MDA:      0.5,
	symbol.FUN:      50,
	symbol.DATA:     10,
	symbol.RLC:      1,
	symbol.ZSC:      20,
	symbol.WINGS:    2,
	symbol.GVT:      0.2,
	symbol.KICK:     5,
	symbol.CTR:      1,
	symbol.HC:       0.2,
	symbol.QBT:      5,
	symbol.QSP:      5,
	symbol.BCD:      0.02,
	symbol.MED:      100,
	symbol.QASH:     1,
	symbol.DGD:      0.05,
	symbol.GNT:      10,
	symbol.MDS:      20,
	symbol.SBTC:     0.05,
	symbol.MANA:     50,
	symbol.GOD:      0.1,
	symbol.BCX:      30,
	symbol.SMT:      50,
	symbol.BTF:      0.1,
	symbol.IOTA:     0.1,
	symbol.NAS:      0.5,
	symbol.NAS_ETH:  0.5,
	symbol.TSL:      10,
	symbol.ADA:      1,
	symbol.LSK:      0.1,
	symbol.WAVES:    0.1,
	symbol.BIFI:     0.2,
	symbol.XTZ:      0.1,
	symbol.BNTY:     10,
	symbol.ICX:      0.5,
	symbol.LEND:     20,
	symbol.LUN:      0.2,
	symbol.ELF:      2,
	symbol.SALT:     0.2,
	symbol.FUEL:     2,
	symbol.DRGN:     2,
	symbol.GTC:      2,
	symbol.MDT:      2,
	symbol.QUN:      2,
	symbol.GNX:      2,
	symbol.DDD:      10,
	symbol.OST:      4,
	symbol.BTO:      10,
	symbol.TIO:      10,
	symbol.THETA:    10,
	symbol.SNET:     10,
	symbol.OCN:      10,
	symbol.ZIL:      10,
	symbol.RUFF:     10,
	symbol.TNC:      10,
	symbol.COFI:     10,
	symbol.ZPT:      0.1,
	symbol.JNT:      10,
	symbol.GXS:      1,
	symbol.MTN:      10,
	symbol.BLZ:      2,
	symbol.GEM:      2,
	symbol.DADI:     2,
	symbol.ABT:      2,
	symbol.LEDU:     10,
	symbol.RFR:      10,
	symbol.XLM:      1,
	symbol.MOBI:     1,
	symbol.ONT:      1,
	symbol.NEO:      0,
	symbol.GAS:      0.02,
	symbol.DBC:      10,
	symbol.QLC:      10,
	symbol.MKR:      0.003,
	symbol.MKR_OLD:  0.003,
	symbol.DAI:      2,
	symbol.LRC:      10,
	symbol.OAX:      10,
	symbol.ZRX:      10,
	symbol.PST:      5,
	symbol.TNT:      20,
	symbol.LLT:      10,
	symbol.DNT:      1,
	symbol.DPY:      2,
	symbol.BCDN:     20,
	symbol.STORJ:    3,
	symbol.OMG:      0.2,
	symbol.PAY:      1,
	symbol.EOS:      0.1,
	symbol.EON:      20,
	symbol.IQ:       20,
	symbol.EOSDAC:   20,
	symbol.TIPS:     100,
	symbol.XRP:      1,
	symbol.CNC:      0.1,
	symbol.TIX:      0.1,
	symbol.XMR:      0.05,
	symbol.BTS:      1,
	symbol.XTC:      10,
	symbol.BU:       0.1,
	symbol.DCR:      0.02,
	symbol.BCN:      10,
	symbol.XMC:      0.05,
	symbol.PPS:      0.01,
	symbol.BOE:      5,
	symbol.PLY:      10,
	symbol.MEDX:     100,
	symbol.TRX:      0.1,
	symbol.SMT_ETH:  50,
	symbol.CS:       10,
	symbol.MAN:      10,
	symbol.REM:      10,
	symbol.LYM:      10,
	symbol.INSTAR:   10,
	symbol.BFT:      10,
	symbol.IHT:      10,
	symbol.SENC:     10,
	symbol.TOMO:     10,
	symbol.ELEC:     10,
	symbol.SHIP:     10,
	symbol.TFD:      10,
	symbol.HAV:      10,
	symbol.HUR:      10,
	symbol.LST:      10,
	symbol.LINO:     10,
	symbol.SWTH:     5,
	symbol.NKN:      5,
	symbol.SOUL:     5,
	symbol.GALA_NEO: 5,
	symbol.LRN:      5,
	symbol.ADD:      20,
	symbol.MEETONE:  5,
	symbol.DOCK:     20,
	symbol.GSE:      20,
	symbol.RATING:   20,
	symbol.HSC:      100,
	symbol.HIT:      100,
	symbol.DX:       100,
	symbol.BXC:      100,
	symbol.PAX:      5,
	symbol.GARD:     100,
	symbol.FTI:      100,
	symbol.SOP:      100,
	symbol.LEMO:     20,
	symbol.NPXS:     40,
	symbol.QKC:      20,
	symbol.IOTX:     20,
	symbol.RED:      20,
	symbol.LBA:      20,
	symbol.KAN:      20,
	symbol.OPEN:     20,
	symbol.MITH:     20,
	symbol.SKM:      20,
	symbol.XVG:      20,
	symbol.NANO:     20,
	symbol.NBAI:     20,
	symbol.UPP:      20,
	symbol.ATMI:     20,
	symbol.TMT:      20,
	symbol.HT:       1,
	symbol.BNB:      0.3,
	symbol.BBK:      20,
	symbol.EDR:      20,
	symbol.MET:      0.3,
	symbol.TCT:      20,
	symbol.EXC:      10,
}
