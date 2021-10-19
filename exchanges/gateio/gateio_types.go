package gateio

import (
	"encoding/json"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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
var transferFees = map[asset.Item]map[currency.Code]fee.Transfer{
	asset.Spot: {
		currency.GT:        {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.25)},
		currency.USDT:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(40)},
		currency.BTC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.001)},
		currency.BSV:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.012)},
		currency.ETC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.038)},
		currency.XRP:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.8)},
		currency.ZEC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.017)},
		currency.QTUM:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.15)},
		currency.DOGE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.BTM:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(39)},
		currency.ONT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.BAT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(51)},
		currency.BTM_ETH:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(690)},
		currency.REQ:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(160)},
		currency.KNC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(23)},
		currency.CDT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(380)},
		currency.INK:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2700)},
		currency.WTC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.2)},
		currency.RCN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2000)},
		currency.BNT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(14)},
		currency.MCO:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
		currency.FUN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2200)},
		currency.RLC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(9.5)},
		currency.WINGS:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
		currency.KICK:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
		currency.HSR:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.05)},
		currency.QBT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(280)},
		currency.BCD:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.86)},
		currency.QASH:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(490)},
		currency.GNT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.SBTC:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.4)},
		currency.GOD:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
		currency.SMT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(780)},
		currency.IOTA:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
		currency.NAS_ETH:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(94)},
		currency.ADA:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.BIFI:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1300)},
		currency.BNTY:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(51000)},
		currency.LEND:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.ELF:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.1)},
		currency.FUEL:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(83000)},
		currency.GTC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(14000)},
		currency.QUN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
		currency.DDD:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(14000)},
		currency.BTO:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1600)},
		currency.THETA:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.34)},
		currency.ZIL:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(22)},
		currency.TNC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1800)},
		currency.ZPT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2300)},
		currency.GXS:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.3)},
		currency.BLZ:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(160)},
		currency.DADI:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
		currency.LEDU:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.XLM:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
		currency.NEO:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.DBC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(320)},
		currency.MKR:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.015)},
		currency.LRC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(91)},
		currency.ZRX:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(35)},
		currency.TNT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.DNT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(500)},
		currency.BCDN:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(35000)},
		currency.OMG:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.4)},
		currency.EON:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.EOSDAC:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1300)},
		currency.CNC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
		currency.XMR:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0071)},
		currency.XTC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
		currency.USDG:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(42)},
		currency.SYS:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.NYZO:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.ETH2:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.005)},
		currency.KAVA:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.34)},
		currency.ANT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.5)},
		currency.STPT:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(200)},
		currency.RSV:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(42)},
		currency.CTSI:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(57)},
		currency.OCEAN:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(64)},
		currency.KSM:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0077)},
		currency.DOT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
		currency.MTRG:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
		currency.COTI:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(70)},
		currency.DIGG:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.00083)},
		currency.YAMV1:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(500)},
		currency.LUNA:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.13)},
		currency.FSN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.4)},
		currency.BZRX:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(320)},
		currency.YAMV2:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.1)},
		currency.BOX:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.35)},
		currency.UNI:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.7)},
		currency.AAVE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.44)},
		currency.ERG:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.5)},
		currency.KPHA:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.KAR:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.25)},
		currency.RMRK:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.3)},
		currency.CRING:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.PICA:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.XRT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.TEER:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.SGB:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.KPN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.CSM:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.KAZE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.SASHIMI:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1300)},
		currency.SWRV:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(55)},
		currency.AUCTION:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.3)},
		currency.OIN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
		currency.ADEL:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1000)},
		currency.KIMCHI:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1100)},
		currency.RING:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1000)},
		currency.CREAM:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.29)},
		currency.DEGO:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.SFG:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(160)},
		currency.CORE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.NU:        {Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
		currency.ARNX:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(820)},
		currency.ROSE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.COVER:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.38)},
		currency.BASE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
		currency.HEGIC:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(220)},
		currency.DUSK:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(250)},
		currency.UNFI:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.8)},
		currency.GHST:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(57)},
		currency.ACH:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(610)},
		currency.GRT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(52)},
		currency.ALEPH:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(65)},
		currency.FXS:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(11)},
		currency.BORING:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3900)},
		currency.BAC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(980)},
		currency.LON:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(16)},
		currency.WOZX:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(94)},
		currency.POND:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(410)},
		currency.DSD:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3400)},
		currency.SHARE:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4000)},
		currency.ONC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(130)},
		currency.ZKS:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(68)},
		currency.MIS:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(30)},
		currency.ONX:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(69)},
		currency.RIF:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(8)},
		currency.PROPS:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2900)},
		currency.LAYER:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(48)},
		currency.QNT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.12)},
		currency.YOP:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
		currency.BONDED:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1300)},
		currency.ROOM:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.BUSD:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(46)},
		currency.UNISTAKE:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1600)},
		currency.FXF:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(130)},
		currency.TORN:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.58)},
		currency.UMB:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(78)},
		currency.JASMY:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(490)},
		currency.BEL:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.SAND:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(46)},
		currency.AMP:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1900)},
		currency.BONDLY:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(560)},
		currency.BMI:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(130)},
		currency.SUPER:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(52)},
		currency.RAY:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.52)},
		currency.POLIS:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.62)},
		currency.WAG:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.2)},
		currency.CYS:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.9)},
		currency.SLRS:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.4)},
		currency.LIKE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(12)},
		currency.PRT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(450)},
		currency.SUNNY:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(170)},
		currency.MNGO:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(19)},
		currency.STEP:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
		currency.FIDA:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.8)},
		currency.AQT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(9.9)},
		currency.PBR:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
		currency.HOPR:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(340)},
		currency.PROM:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.9)},
		currency.TVK:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(200)},
		currency.A5T:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(690)},
		currency.CUDOS:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1500)},
		currency.COMBO:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
		currency.DOWS:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
		currency.KYL:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(170)},
		currency.EXRD:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(17)},
		currency.UTK:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(96)},
		currency.ETHA:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(240)},
		currency.ALN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(730)},
		currency.AUDIO:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.CHR:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(99)},
		currency.HAPI:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.56)},
		currency.BLANK:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(190)},
		currency.ERN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.4)},
		currency.KINE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
		currency.FET:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(50)},
		currency.UOS:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(70)},
		currency.ALICE:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.1)},
		currency.ZEE:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(84)},
		currency.POLC:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(340)},
		currency.XED:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(89)},
		currency.RLY:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(65)},
		currency.ANC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(9.7)},
		currency.DAFI:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(830)},
		currency.FIRE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2100)},
		currency.TARA:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(7500)},
		currency.PCNT:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(620)},
		currency.DG:        {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.14)},
		currency.SPI:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.81)},
		currency.BANK:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.57)},
		currency.ATD:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(39)},
		currency.UMX:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(38)},
		currency.TIDAL:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(8400)},
		currency.LABS:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5300)},
		currency.OGN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(69)},
		currency.BLES:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(420)},
		currency.OVR:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(76)},
		currency.HGET:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
		currency.NOIA:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
		currency.COOK:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(8100)},
		currency.CFI:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.5)},
		currency.MTL:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.FST:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.4)},
		currency.AME:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2400)},
		currency.STN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(79)},
		currency.SHOPX:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(380)},
		currency.SHFT:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(40)},
		currency.RBC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
		currency.VAI:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(98)},
		currency.FEI:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(40)},
		currency.XEND:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(150)},
		currency.ADX:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(68)},
		currency.SUKU:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(70)},
		currency.LTO:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(140)},
		currency.TOTM:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(250)},
		currency.BLY:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1200)},
		currency.LKR:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(540)},
		currency.PNK:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(700)},
		currency.RAZE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(310)},
		currency.DUCK2:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
		currency.CEL:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.4)},
		currency.DDIM:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.1)},
		currency.FLY:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1600)},
		currency.TLM:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(190)},
		currency.DDOS:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(81)},
		currency.GS:        {Deposit: fee.Convert(0), Withdrawal: fee.Convert(190)},
		currency.RAGE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1500)},
		currency.AKITA:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(22000000)},
		currency.FORTH:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.1)},
		currency.CARDS:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.7)},
		currency.HORD:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(260)},
		currency.WBTC:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.00065)},
		currency.ARES:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1600)},
		currency.BOA:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(550)},
		currency.SUSD:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
		currency.SFI:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.068)},
		currency.TCP:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(320)},
		currency.BLACK:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1200)},
		currency.EZ:        {Deposit: fee.Convert(0), Withdrawal: fee.Convert(11)},
		currency.VSO:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(19)},
		currency.XAVA:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
		currency.PNG:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
		currency.LOCG:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(270)},
		currency.WSIENNA:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.4)},
		currency.WEMIX:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
		currency.STBU:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1200)},
		currency.DFND:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(31000)},
		currency.FTT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.97)},
		currency.GDT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
		currency.GUM:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(340)},
		currency.PRARE:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(710)},
		currency.GYEN:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1500)},
		currency.METIS:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.8)},
		currency.BZZ:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.TENSET:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)}, // 10SET
		currency.PDEX:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.5)},
		currency.FEAR:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(28)},
		currency.ELON:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(210000000)},
		currency.NOA:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3900)},
		currency.DOP:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(27)},
		currency.NAOS:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(28)},
		currency.GITCOIN:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.8)},
		currency.XCAD:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(17)},
		currency.LSS:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
		currency.CVX:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.9)},
		currency.PHTR:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(150)},
		currency.APN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(960)},
		currency.DFYN:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(70)},
		currency.LIME:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5600)},
		currency.FORM:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(320)},
		currency.KEX:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
		currency.DLTA:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(410)},
		currency.DPR:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(730)},
		currency.CQT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(43)},
		currency.OLY:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(6400)},
		currency.FUSE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(750)},
		currency.MLN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.43)},
		currency.SRK:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(14000)},
		currency.MM:        {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.4)},
		currency.BURP:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(460)},
		currency.CART:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(63)},
		currency.C98:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(11)},
		currency.DNXC:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(85)},
		currency.DERC:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(21)},
		currency.PLA:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(41)},
		currency.EFI:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(53)},
		currency.GAME:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(26)},
		currency.HMT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(39)},
		currency.SKT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(240)},
		currency.SPHRI:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(290)},
		currency.BIT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(36)},
		currency.RARE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(18)},
		currency.ZLW:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.SKYRIM:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(490)},
		currency.OCT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(22)},
		currency.ATA:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(39)},
		currency.PUSH:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(24)},
		currency.REVO:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(130)},
		currency.VENT:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(93)},
		currency.LDO:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(36)},
		currency.GEL:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(17)},
		currency.CTRC:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(310)},
		currency.ITGR:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(140)},
		currency.HOTCROSS:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
		currency.MOT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(540)},
		currency.FOREX:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(290)},
		currency.OPUL:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(34)},
		currency.EVA:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1300)},
		currency.POLI:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1700)},
		currency.TAUR:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1600)},
		currency.EQX:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
		currency.RBN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(9.4)},
		currency.PHM:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(14000)},
		currency.FLOKI:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1000000)},
		currency.CIRUS:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(74)},
		currency.DYDX:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(8.6)},
		currency.RGT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.6)},
		currency.AGLD:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
		currency.DOGNFT:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5100)},
		currency.SOV:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.6)},
		currency.URUS:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.41)},
		currency.CFG:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(25)},
		currency.TBTC:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0027)},
		currency.WOM:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(140)},
		currency.NFTX:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.75)},
		currency.ORAI:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.3)},
		currency.LIT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(14)},
		currency.POOLZ:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.2)},
		currency.DODO:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(36)},
		currency.IPAD:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(73)},
		currency.OPIUM:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(23)},
		currency.REEF:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2300)},
		currency.MAPS:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(62)},
		currency.MIR:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(18)},
		currency.ZCN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(130)},
		currency.BAO:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(190000)},
		currency.DIS:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.2)},
		currency.PBTC35A:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.68)},
		currency.NORD:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(28)},
		currency.FLOW:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.11)},
		currency.ENJ:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(34)},
		currency.FIN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(240)},
		currency.PRQ:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
		currency.INJ:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3)},
		currency.ROOBEE:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(7100)},
		currency.KP3R:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.16)},
		currency.HYVE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(93)},
		currency.RAMP:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(190)},
		currency.RARI:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.8)},
		currency.MPH:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2600)},
		currency.CVP:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(29)},
		currency.VALUE:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(57)},
		currency.YFII:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.013)},
		currency.SXP:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(24)},
		currency.BAND:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.TROY:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2800)},
		currency.SPA:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(700)},
		currency.FOR:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(640)},
		currency.DIA:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(30)},
		currency.TRB:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.2)},
		currency.PEARL:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.059)},
		currency.NFT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1500000)},
		currency.SLM:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.2)},
		currency.TAI:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(14)},
		currency.JFI:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.083)},
		currency.YFI:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0015)},
		currency.DKA:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(450)},
		currency.DOS:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1800)},
		currency.SRM:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.5)},
		currency.LBK:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4000)},
		currency.ASD:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(92)},
		currency.SWOP:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.13)},
		currency.WEST:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.5)},
		currency.HYDRA:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.116)},
		currency.OLT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(200)},
		currency.XYM:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.8)},
		currency.LAT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(14)},
		currency.STC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(17)},
		currency.BU:        {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1100)},
		currency.HNT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.15)},
		currency.AKT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.66)},
		currency.BTC3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.6)},
		currency.COTI3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.3)},
		currency.XCH3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
		currency.IOST3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.3)},
		currency.BZZ3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.3)},
		currency.TRIBE3L:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.RAY3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.1)},
		currency.AR3L:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.3)},
		currency.ONE3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.HBAR3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.9)},
		currency.CSPR3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.4)},
		currency.SXP3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.2)},
		currency.XEC3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.3)},
		currency.LIT3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.4)},
		currency.MINA3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.1)},
		currency.GALA3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.7)},
		currency.FTT3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.6)},
		currency.C983L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.7)},
		currency.DYDX3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.3)},
		currency.MTL3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5.1)},
		currency.FTM3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.SAND3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.4)},
		currency.LUNA3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.5)},
		currency.ALPHA3L:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.1)},
		currency.RUNE3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.2)},
		currency.ICP3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(8)},
		currency.SHIB3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.ACH3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(12)},
		currency.ALICE3L:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(11)},
		currency.AXS3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.1)},
		currency.MATIC3L:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4)},
		currency.BTC5L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.BCH5L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(21)},
		currency.DOT5L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(39)},
		currency.XRP5L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(17)},
		currency.BSV5L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(27)},
		currency.LTC5L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(23)},
		currency.EOS5L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(25)},
		currency.ETH5L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.9)},
		currency.LINK3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(91)},
		currency.KAVA3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(39)},
		currency.EGLD3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5.7)},
		currency.CHZ3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.5)},
		currency.MKR3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(31)},
		currency.LRC3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.9)},
		currency.BAL3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(93)},
		currency.JST3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(8)},
		currency.SERO3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(14)},
		currency.VET3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(12)},
		currency.THETA3L:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(17)},
		currency.ZIL3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(260)},
		currency.GRIN3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
		currency.BEAM3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(37)},
		currency.SOL3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.SKL3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(100)},
		currency.ONEINCH3L: {Deposit: fee.Convert(0), Withdrawal: fee.Convert(8.3)}, // 1INCH3L
		currency.LON3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(160)},
		currency.DOGE3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.1)},
		currency.GRT3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(250)},
		currency.BNB3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.TRX3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(9.2)},
		currency.ATOM3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(16)},
		currency.AVAX3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.NEAR3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(28)},
		currency.ROSE3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(25)},
		currency.ZEN3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(25)},
		currency.QTUM3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
		currency.XLM3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(34)},
		currency.XRP3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(58)},
		currency.CFX3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.3)},
		currency.OMG3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.ALGO3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.9)},
		currency.WAVES3L:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.8)},
		currency.NEO3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(26)},
		currency.ONT3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(74)},
		currency.ETC3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.8)},
		currency.CVC3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(32)},
		currency.SNX3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(93)},
		currency.ADA3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.DASH3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(26)},
		currency.AAVE3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(12)},
		currency.SRM3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(14)},
		currency.KSM3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.8)},
		currency.BTM3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(320)},
		currency.ZEC3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(75)},
		currency.XMR3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(11)},
		currency.AMPL3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.4)},
		currency.CRV3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
		currency.COMP3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(23)},
		currency.YFII3L:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
		currency.YFI3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
		currency.HT3L:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(53)},
		currency.OKB3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(19)},
		currency.UNI3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(33)},
		currency.DOT3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.3)},
		currency.FIL3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5.9)},
		currency.SUSHI3L:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.8)},
		currency.ETH3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.1)},
		currency.EOS3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.7)},
		currency.BSV3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(130)},
		currency.BCH3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
		currency.LTC3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
		currency.XTZ3L:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.9)},
		currency.RVN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1)},
		currency.AR:        {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.5)},
		currency.SNK:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.25)},
		currency.NSDX:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(8)},
		currency.DCR:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.02)},
		currency.XMC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(6.4)},
		currency.VELO:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(7.6)},
		currency.PPS:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.01)},
		currency.XPR:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(70)},
		currency.HIVE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.5)},
		currency.BCHA:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.051)},
		currency.FLUX:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.NAX:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1300)},
		currency.NBOT:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2000)},
		currency.PLY:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.BEAM:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.7)},
		currency.IOST:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(37)},
		currency.MINA:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.47)},
		currency.REP:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
		currency.STAR:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.9)},
		currency.ABBC:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(18)},
		currency.FIC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(30)},
		currency.STOX:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4400)},
		currency.VIDYX:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(35)},
		currency.ARN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
		currency.TFUEL:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.CS:        {Deposit: fee.Convert(0), Withdrawal: fee.Convert(50)},
		currency.REM:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(60000)},
		currency.INSTAR:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.ONG:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.7)},
		currency.BFT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3000)},
		currency.SENC:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(40000)},
		currency.ELEC:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(26000)},
		currency.TFD:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.HUR:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(10)},
		currency.SWTH:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(140)},
		currency.SOUL:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5)},
		currency.ADD:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.DOCK:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(25)},
		currency.RATING:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(160000)},
		currency.HIT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(830000)},
		currency.CNNS:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(21000)},
		currency.MBL:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(190)},
		currency.MIX:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(9300)},
		currency.LEO:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(50)},
		currency.BTCBEAR:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(520000)},
		currency.ETHBULL:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.032)},
		currency.EOSBEAR:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2500000)},
		currency.XRPBULL:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4200)},
		currency.WGRT:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(580)},
		currency.CORAL:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(9.3)},
		currency.KGC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(220000)},
		currency.RUNE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.3)},
		currency.CBK:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(12)},
		currency.OPA:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1300)},
		currency.MCRN:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.72)},
		currency.KABY:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(34)},
		currency.BP:        {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.8)},
		currency.SFUND:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.7)},
		currency.ASTRO:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(22)},
		currency.ARV:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4500)},
		currency.ROSN:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(33)},
		currency.CPHR:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(280)},
		currency.KWS:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(66)},
		currency.CTT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.BEEFI:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0054)},
		currency.BLIN:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.8)},
		currency.PING:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4300)},
		currency.XPNET:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
		currency.BABY:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(4.8)},
		currency.OPS:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(40)},
		currency.RACA:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1300)},
		currency.HOD:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(120)},
		currency.OLYMPUS:   {Deposit: fee.Convert(0), Withdrawal: fee.Convert(51000000)},
		currency.BMON:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(32)},
		currency.PVU:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.1)},
		currency.FAN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(84)},
		currency.SKILL:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.19)},
		currency.SPS:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(5.9)},
		currency.HERO:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(87)},
		currency.FEVR:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(410)},
		currency.WEX:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1900)},
		currency.KALM:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2.1)},
		currency.KPAD:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(220)},
		currency.BABYDOGE:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.PIG:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.FINE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(15)},
		currency.BSCS:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(16)},
		currency.SAFEMARS:  {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0)},
		currency.PSG:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.3)},
		currency.PET:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(41)},
		currency.ALPACA:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(7)},
		currency.BRY:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(19)},
		currency.CTK:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.1)},
		currency.TOOLS:     {Deposit: fee.Convert(0), Withdrawal: fee.Convert(30)},
		currency.JULD:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(110)},
		currency.CAKE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.15)},
		currency.BAKE:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.5)},
		currency.FRA:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(90)},
		currency.TWT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.9)},
		currency.CRO:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(200)},
		currency.WIN:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(9300)},
		currency.MTV:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2900)},
		currency.ARPA:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(430)},
		currency.ALGO:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1.2)},
		currency.CKB:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(130)},
		currency.BXC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(89000)},
		currency.USDC:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
		currency.GARD:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(15000)},
		currency.HPB:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(25)},
		currency.FTI:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(21000)},
		currency.LEMO:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(6300)},
		currency.PUNDIX:    {Deposit: fee.Convert(0), Withdrawal: fee.Convert(35)},
		currency.IOTX:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(29)},
		currency.LBA:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(14000)},
		currency.OPEN:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(14000)},
		currency.SKM:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(23000)},
		currency.NANO:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.UPP:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.TMT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.EDG:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(230)},
		currency.EGLD:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.052)},
		currency.CSPR:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(18)},
		currency.FIS:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(2)},
		currency.GO:        {Deposit: fee.Convert(0), Withdrawal: fee.Convert(63)},
		currency.MDX:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(3.2)},
		currency.WAR:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(46)},
		currency.XNFT:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(53)},
		currency.BXH:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(56)},
		currency.BAGS:      {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.25)},
		currency.BNB:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(0.0043)},
		currency.EDR:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(20)},
		currency.TCT:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(1200)},
		currency.MXC:       {Deposit: fee.Convert(0), Withdrawal: fee.Convert(710)},
	},
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
