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

// withdrawalFees the large list of predefined withdrawal fees. Prone to change.
var withdrawalFees = map[asset.Item]map[currency.Code]fee.Transfer{
	asset.Spot: {
		currency.USDT:     {Withdrawal: 10},
		currency.USDT_ETH: {Withdrawal: 10},
		currency.BTC:      {Withdrawal: 0.001},
		currency.BCH:      {Withdrawal: 0.0006},
		currency.BTG:      {Withdrawal: 0.002},
		currency.LTC:      {Withdrawal: 0.002},
		currency.ZEC:      {Withdrawal: 0.001},
		currency.ETH:      {Withdrawal: 0.003},
		currency.ETC:      {Withdrawal: 0.01},
		currency.DASH:     {Withdrawal: 0.02},
		currency.QTUM:     {Withdrawal: 0.1},
		currency.QTUM_ETH: {Withdrawal: 0.1},
		currency.DOGE:     {Withdrawal: 50},
		currency.REP:      {Withdrawal: 0.1},
		currency.BAT:      {Withdrawal: 10},
		currency.SNT:      {Withdrawal: 30},
		currency.BTM:      {Withdrawal: 10},
		currency.BTM_ETH:  {Withdrawal: 10},
		currency.CVC:      {Withdrawal: 5},
		currency.REQ:      {Withdrawal: 20},
		currency.RDN:      {Withdrawal: 1},
		currency.STX:      {Withdrawal: 3},
		currency.KNC:      {Withdrawal: 1},
		currency.LINK:     {Withdrawal: 8},
		currency.FIL:      {Withdrawal: 0.1},
		currency.CDT:      {Withdrawal: 20},
		currency.AE:       {Withdrawal: 1},
		currency.INK:      {Withdrawal: 10},
		currency.BOT:      {Withdrawal: 5},
		currency.POWR:     {Withdrawal: 5},
		currency.WTC:      {Withdrawal: 0.2},
		currency.VET:      {Withdrawal: 10},
		currency.RCN:      {Withdrawal: 5},
		currency.PPT:      {Withdrawal: 0.1},
		currency.ARN:      {Withdrawal: 2},
		currency.BNT:      {Withdrawal: 0.5},
		currency.VERI:     {Withdrawal: 0.005},
		currency.MCO:      {Withdrawal: 0.1},
		currency.MDA:      {Withdrawal: 0.5},
		currency.FUN:      {Withdrawal: 50},
		currency.DATA:     {Withdrawal: 10},
		currency.RLC:      {Withdrawal: 1},
		currency.ZSC:      {Withdrawal: 20},
		currency.WINGS:    {Withdrawal: 2},
		currency.GVT:      {Withdrawal: 0.2},
		currency.KICK:     {Withdrawal: 5},
		currency.CTR:      {Withdrawal: 1},
		currency.HC:       {Withdrawal: 0.2},
		currency.QBT:      {Withdrawal: 5},
		currency.QSP:      {Withdrawal: 5},
		currency.BCD:      {Withdrawal: 0.02},
		currency.MED:      {Withdrawal: 100},
		currency.QASH:     {Withdrawal: 1},
		currency.DGD:      {Withdrawal: 0.05},
		currency.GNT:      {Withdrawal: 10},
		currency.MDS:      {Withdrawal: 20},
		currency.SBTC:     {Withdrawal: 0.05},
		currency.MANA:     {Withdrawal: 50},
		currency.GOD:      {Withdrawal: 0.1},
		currency.BCX:      {Withdrawal: 30},
		currency.SMT:      {Withdrawal: 50},
		currency.BTF:      {Withdrawal: 0.1},
		currency.IOTA:     {Withdrawal: 0.1},
		currency.NAS:      {Withdrawal: 0.5},
		currency.NAS_ETH:  {Withdrawal: 0.5},
		currency.TSL:      {Withdrawal: 10},
		currency.ADA:      {Withdrawal: 1},
		currency.LSK:      {Withdrawal: 0.1},
		currency.WAVES:    {Withdrawal: 0.1},
		currency.BIFI:     {Withdrawal: 0.2},
		currency.XTZ:      {Withdrawal: 0.1},
		currency.BNTY:     {Withdrawal: 10},
		currency.ICX:      {Withdrawal: 0.5},
		currency.LEND:     {Withdrawal: 20},
		currency.LUN:      {Withdrawal: 0.2},
		currency.ELF:      {Withdrawal: 2},
		currency.SALT:     {Withdrawal: 0.2},
		currency.FUEL:     {Withdrawal: 2},
		currency.DRGN:     {Withdrawal: 2},
		currency.GTC:      {Withdrawal: 2},
		currency.MDT:      {Withdrawal: 2},
		currency.QUN:      {Withdrawal: 2},
		currency.GNX:      {Withdrawal: 2},
		currency.DDD:      {Withdrawal: 10},
		currency.OST:      {Withdrawal: 4},
		currency.BTO:      {Withdrawal: 10},
		currency.TIO:      {Withdrawal: 10},
		currency.THETA:    {Withdrawal: 10},
		currency.SNET:     {Withdrawal: 10},
		currency.OCN:      {Withdrawal: 10},
		currency.ZIL:      {Withdrawal: 10},
		currency.RUFF:     {Withdrawal: 10},
		currency.TNC:      {Withdrawal: 10},
		currency.COFI:     {Withdrawal: 10},
		currency.ZPT:      {Withdrawal: 0.1},
		currency.JNT:      {Withdrawal: 10},
		currency.GXS:      {Withdrawal: 1},
		currency.MTN:      {Withdrawal: 10},
		currency.BLZ:      {Withdrawal: 2},
		currency.GEM:      {Withdrawal: 2},
		currency.DADI:     {Withdrawal: 2},
		currency.ABT:      {Withdrawal: 2},
		currency.LEDU:     {Withdrawal: 10},
		currency.RFR:      {Withdrawal: 10},
		currency.XLM:      {Withdrawal: 1},
		currency.MOBI:     {Withdrawal: 1},
		currency.ONT:      {Withdrawal: 1},
		currency.NEO:      {Withdrawal: 0},
		currency.GAS:      {Withdrawal: 0.02},
		currency.DBC:      {Withdrawal: 10},
		currency.QLC:      {Withdrawal: 10},
		currency.MKR:      {Withdrawal: 0.003},
		currency.MKR_OLD:  {Withdrawal: 0.003},
		currency.DAI:      {Withdrawal: 2},
		currency.LRC:      {Withdrawal: 10},
		currency.OAX:      {Withdrawal: 10},
		currency.ZRX:      {Withdrawal: 10},
		currency.PST:      {Withdrawal: 5},
		currency.TNT:      {Withdrawal: 20},
		currency.LLT:      {Withdrawal: 10},
		currency.DNT:      {Withdrawal: 1},
		currency.DPY:      {Withdrawal: 2},
		currency.BCDN:     {Withdrawal: 20},
		currency.STORJ:    {Withdrawal: 3},
		currency.OMG:      {Withdrawal: 0.2},
		currency.PAY:      {Withdrawal: 1},
		currency.EOS:      {Withdrawal: 0.1},
		currency.EON:      {Withdrawal: 20},
		currency.IQ:       {Withdrawal: 20},
		currency.EOSDAC:   {Withdrawal: 20},
		currency.TIPS:     {Withdrawal: 100},
		currency.XRP:      {Withdrawal: 1},
		currency.CNC:      {Withdrawal: 0.1},
		currency.TIX:      {Withdrawal: 0.1},
		currency.XMR:      {Withdrawal: 0.05},
		currency.BTS:      {Withdrawal: 1},
		currency.XTC:      {Withdrawal: 10},
		currency.BU:       {Withdrawal: 0.1},
		currency.DCR:      {Withdrawal: 0.02},
		currency.BCN:      {Withdrawal: 10},
		currency.XMC:      {Withdrawal: 0.05},
		currency.PPS:      {Withdrawal: 0.01},
		currency.BOE:      {Withdrawal: 5},
		currency.PLY:      {Withdrawal: 10},
		currency.MEDX:     {Withdrawal: 100},
		currency.TRX:      {Withdrawal: 0.1},
		currency.SMT_ETH:  {Withdrawal: 50},
		currency.CS:       {Withdrawal: 10},
		currency.MAN:      {Withdrawal: 10},
		currency.REM:      {Withdrawal: 10},
		currency.LYM:      {Withdrawal: 10},
		currency.INSTAR:   {Withdrawal: 10},
		currency.BFT:      {Withdrawal: 10},
		currency.IHT:      {Withdrawal: 10},
		currency.SENC:     {Withdrawal: 10},
		currency.TOMO:     {Withdrawal: 10},
		currency.ELEC:     {Withdrawal: 10},
		currency.SHIP:     {Withdrawal: 10},
		currency.TFD:      {Withdrawal: 10},
		currency.HAV:      {Withdrawal: 10},
		currency.HUR:      {Withdrawal: 10},
		currency.LST:      {Withdrawal: 10},
		currency.LINO:     {Withdrawal: 10},
		currency.SWTH:     {Withdrawal: 5},
		currency.NKN:      {Withdrawal: 5},
		currency.SOUL:     {Withdrawal: 5},
		currency.GALA_NEO: {Withdrawal: 5},
		currency.LRN:      {Withdrawal: 5},
		currency.ADD:      {Withdrawal: 20},
		currency.MEETONE:  {Withdrawal: 5},
		currency.DOCK:     {Withdrawal: 20},
		currency.GSE:      {Withdrawal: 20},
		currency.RATING:   {Withdrawal: 20},
		currency.HSC:      {Withdrawal: 100},
		currency.HIT:      {Withdrawal: 100},
		currency.DX:       {Withdrawal: 100},
		currency.BXC:      {Withdrawal: 100},
		currency.PAX:      {Withdrawal: 5},
		currency.GARD:     {Withdrawal: 100},
		currency.FTI:      {Withdrawal: 100},
		currency.SOP:      {Withdrawal: 100},
		currency.LEMO:     {Withdrawal: 20},
		currency.NPXS:     {Withdrawal: 40},
		currency.QKC:      {Withdrawal: 20},
		currency.IOTX:     {Withdrawal: 20},
		currency.RED:      {Withdrawal: 20},
		currency.LBA:      {Withdrawal: 20},
		currency.KAN:      {Withdrawal: 20},
		currency.OPEN:     {Withdrawal: 20},
		currency.MITH:     {Withdrawal: 20},
		currency.SKM:      {Withdrawal: 20},
		currency.XVG:      {Withdrawal: 20},
		currency.NANO:     {Withdrawal: 20},
		currency.NBAI:     {Withdrawal: 20},
		currency.UPP:      {Withdrawal: 20},
		currency.ATMI:     {Withdrawal: 20},
		currency.TMT:      {Withdrawal: 20},
		currency.HT:       {Withdrawal: 1},
		currency.BNB:      {Withdrawal: 0.3},
		currency.BBK:      {Withdrawal: 20},
		currency.EDR:      {Withdrawal: 20},
		currency.MET:      {Withdrawal: 0.3},
		currency.TCT:      {Withdrawal: 20},
		currency.EXC:      {Withdrawal: 10},
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
