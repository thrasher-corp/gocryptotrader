package gateio

import (
	"encoding/json"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
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

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change
var WithdrawalFees = map[currency.Code]float64{
	currency.USDT:     10,
	currency.USDT_ETH: 10,
	currency.BTC:      0.001,
	currency.BCH:      0.0006,
	currency.BTG:      0.002,
	currency.LTC:      0.002,
	currency.ZEC:      0.001,
	currency.ETH:      0.003,
	currency.ETC:      0.01,
	currency.DASH:     0.02,
	currency.QTUM:     0.1,
	currency.QTUM_ETH: 0.1,
	currency.DOGE:     50,
	currency.REP:      0.1,
	currency.BAT:      10,
	currency.SNT:      30,
	currency.BTM:      10,
	currency.BTM_ETH:  10,
	currency.CVC:      5,
	currency.REQ:      20,
	currency.RDN:      1,
	currency.STX:      3,
	currency.KNC:      1,
	currency.LINK:     8,
	currency.FIL:      0.1,
	currency.CDT:      20,
	currency.AE:       1,
	currency.INK:      10,
	currency.BOT:      5,
	currency.POWR:     5,
	currency.WTC:      0.2,
	currency.VET:      10,
	currency.RCN:      5,
	currency.PPT:      0.1,
	currency.ARN:      2,
	currency.BNT:      0.5,
	currency.VERI:     0.005,
	currency.MCO:      0.1,
	currency.MDA:      0.5,
	currency.FUN:      50,
	currency.DATA:     10,
	currency.RLC:      1,
	currency.ZSC:      20,
	currency.WINGS:    2,
	currency.GVT:      0.2,
	currency.KICK:     5,
	currency.CTR:      1,
	currency.HC:       0.2,
	currency.QBT:      5,
	currency.QSP:      5,
	currency.BCD:      0.02,
	currency.MED:      100,
	currency.QASH:     1,
	currency.DGD:      0.05,
	currency.GNT:      10,
	currency.MDS:      20,
	currency.SBTC:     0.05,
	currency.MANA:     50,
	currency.GOD:      0.1,
	currency.BCX:      30,
	currency.SMT:      50,
	currency.BTF:      0.1,
	currency.IOTA:     0.1,
	currency.NAS:      0.5,
	currency.NAS_ETH:  0.5,
	currency.TSL:      10,
	currency.ADA:      1,
	currency.LSK:      0.1,
	currency.WAVES:    0.1,
	currency.BIFI:     0.2,
	currency.XTZ:      0.1,
	currency.BNTY:     10,
	currency.ICX:      0.5,
	currency.LEND:     20,
	currency.LUN:      0.2,
	currency.ELF:      2,
	currency.SALT:     0.2,
	currency.FUEL:     2,
	currency.DRGN:     2,
	currency.GTC:      2,
	currency.MDT:      2,
	currency.QUN:      2,
	currency.GNX:      2,
	currency.DDD:      10,
	currency.OST:      4,
	currency.BTO:      10,
	currency.TIO:      10,
	currency.THETA:    10,
	currency.SNET:     10,
	currency.OCN:      10,
	currency.ZIL:      10,
	currency.RUFF:     10,
	currency.TNC:      10,
	currency.COFI:     10,
	currency.ZPT:      0.1,
	currency.JNT:      10,
	currency.GXS:      1,
	currency.MTN:      10,
	currency.BLZ:      2,
	currency.GEM:      2,
	currency.DADI:     2,
	currency.ABT:      2,
	currency.LEDU:     10,
	currency.RFR:      10,
	currency.XLM:      1,
	currency.MOBI:     1,
	currency.ONT:      1,
	currency.NEO:      0,
	currency.GAS:      0.02,
	currency.DBC:      10,
	currency.QLC:      10,
	currency.MKR:      0.003,
	currency.MKR_OLD:  0.003,
	currency.DAI:      2,
	currency.LRC:      10,
	currency.OAX:      10,
	currency.ZRX:      10,
	currency.PST:      5,
	currency.TNT:      20,
	currency.LLT:      10,
	currency.DNT:      1,
	currency.DPY:      2,
	currency.BCDN:     20,
	currency.STORJ:    3,
	currency.OMG:      0.2,
	currency.PAY:      1,
	currency.EOS:      0.1,
	currency.EON:      20,
	currency.IQ:       20,
	currency.EOSDAC:   20,
	currency.TIPS:     100,
	currency.XRP:      1,
	currency.CNC:      0.1,
	currency.TIX:      0.1,
	currency.XMR:      0.05,
	currency.BTS:      1,
	currency.XTC:      10,
	currency.BU:       0.1,
	currency.DCR:      0.02,
	currency.BCN:      10,
	currency.XMC:      0.05,
	currency.PPS:      0.01,
	currency.BOE:      5,
	currency.PLY:      10,
	currency.MEDX:     100,
	currency.TRX:      0.1,
	currency.SMT_ETH:  50,
	currency.CS:       10,
	currency.MAN:      10,
	currency.REM:      10,
	currency.LYM:      10,
	currency.INSTAR:   10,
	currency.BFT:      10,
	currency.IHT:      10,
	currency.SENC:     10,
	currency.TOMO:     10,
	currency.ELEC:     10,
	currency.SHIP:     10,
	currency.TFD:      10,
	currency.HAV:      10,
	currency.HUR:      10,
	currency.LST:      10,
	currency.LINO:     10,
	currency.SWTH:     5,
	currency.NKN:      5,
	currency.SOUL:     5,
	currency.GALA_NEO: 5,
	currency.LRN:      5,
	currency.ADD:      20,
	currency.MEETONE:  5,
	currency.DOCK:     20,
	currency.GSE:      20,
	currency.RATING:   20,
	currency.HSC:      100,
	currency.HIT:      100,
	currency.DX:       100,
	currency.BXC:      100,
	currency.PAX:      5,
	currency.GARD:     100,
	currency.FTI:      100,
	currency.SOP:      100,
	currency.LEMO:     20,
	currency.NPXS:     40,
	currency.QKC:      20,
	currency.IOTX:     20,
	currency.RED:      20,
	currency.LBA:      20,
	currency.KAN:      20,
	currency.OPEN:     20,
	currency.MITH:     20,
	currency.SKM:      20,
	currency.XVG:      20,
	currency.NANO:     20,
	currency.NBAI:     20,
	currency.UPP:      20,
	currency.ATMI:     20,
	currency.TMT:      20,
	currency.HT:       1,
	currency.BNB:      0.3,
	currency.BBK:      20,
	currency.EDR:      20,
	currency.MET:      0.3,
	currency.TCT:      20,
	currency.EXC:      10,
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
