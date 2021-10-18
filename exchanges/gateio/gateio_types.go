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
		currency.USDT:     {Withdrawal: fee.Convert(10)},
		currency.USDT_ETH: {Withdrawal: fee.Convert(10)},
		currency.BTC:      {Withdrawal: fee.Convert(0.001)},
		currency.BCH:      {Withdrawal: fee.Convert(0.0006)},
		currency.BTG:      {Withdrawal: fee.Convert(0.002)},
		currency.LTC:      {Withdrawal: fee.Convert(0.002)},
		currency.ZEC:      {Withdrawal: fee.Convert(0.001)},
		currency.ETH:      {Withdrawal: fee.Convert(0.003)},
		currency.ETC:      {Withdrawal: fee.Convert(0.01)},
		currency.DASH:     {Withdrawal: fee.Convert(0.02)},
		currency.QTUM:     {Withdrawal: fee.Convert(0.1)},
		currency.QTUM_ETH: {Withdrawal: fee.Convert(0.1)},
		currency.DOGE:     {Withdrawal: fee.Convert(50)},
		currency.REP:      {Withdrawal: fee.Convert(0.1)},
		currency.BAT:      {Withdrawal: fee.Convert(10)},
		currency.SNT:      {Withdrawal: fee.Convert(30)},
		currency.BTM:      {Withdrawal: fee.Convert(10)},
		currency.BTM_ETH:  {Withdrawal: fee.Convert(10)},
		currency.CVC:      {Withdrawal: fee.Convert(5)},
		currency.REQ:      {Withdrawal: fee.Convert(20)},
		currency.RDN:      {Withdrawal: fee.Convert(1)},
		currency.STX:      {Withdrawal: fee.Convert(3)},
		currency.KNC:      {Withdrawal: fee.Convert(1)},
		currency.LINK:     {Withdrawal: fee.Convert(8)},
		currency.FIL:      {Withdrawal: fee.Convert(0.1)},
		currency.CDT:      {Withdrawal: fee.Convert(20)},
		currency.AE:       {Withdrawal: fee.Convert(1)},
		currency.INK:      {Withdrawal: fee.Convert(10)},
		currency.BOT:      {Withdrawal: fee.Convert(5)},
		currency.POWR:     {Withdrawal: fee.Convert(5)},
		currency.WTC:      {Withdrawal: fee.Convert(0.2)},
		currency.VET:      {Withdrawal: fee.Convert(10)},
		currency.RCN:      {Withdrawal: fee.Convert(5)},
		currency.PPT:      {Withdrawal: fee.Convert(0.1)},
		currency.ARN:      {Withdrawal: fee.Convert(2)},
		currency.BNT:      {Withdrawal: fee.Convert(0.5)},
		currency.VERI:     {Withdrawal: fee.Convert(0.005)},
		currency.MCO:      {Withdrawal: fee.Convert(0.1)},
		currency.MDA:      {Withdrawal: fee.Convert(0.5)},
		currency.FUN:      {Withdrawal: fee.Convert(50)},
		currency.DATA:     {Withdrawal: fee.Convert(10)},
		currency.RLC:      {Withdrawal: fee.Convert(1)},
		currency.ZSC:      {Withdrawal: fee.Convert(20)},
		currency.WINGS:    {Withdrawal: fee.Convert(2)},
		currency.GVT:      {Withdrawal: fee.Convert(0.2)},
		currency.KICK:     {Withdrawal: fee.Convert(5)},
		currency.CTR:      {Withdrawal: fee.Convert(1)},
		currency.HC:       {Withdrawal: fee.Convert(0.2)},
		currency.QBT:      {Withdrawal: fee.Convert(5)},
		currency.QSP:      {Withdrawal: fee.Convert(5)},
		currency.BCD:      {Withdrawal: fee.Convert(0.02)},
		currency.MED:      {Withdrawal: fee.Convert(100)},
		currency.QASH:     {Withdrawal: fee.Convert(1)},
		currency.DGD:      {Withdrawal: fee.Convert(0.05)},
		currency.GNT:      {Withdrawal: fee.Convert(10)},
		currency.MDS:      {Withdrawal: fee.Convert(20)},
		currency.SBTC:     {Withdrawal: fee.Convert(0.05)},
		currency.MANA:     {Withdrawal: fee.Convert(50)},
		currency.GOD:      {Withdrawal: fee.Convert(0.1)},
		currency.BCX:      {Withdrawal: fee.Convert(30)},
		currency.SMT:      {Withdrawal: fee.Convert(50)},
		currency.BTF:      {Withdrawal: fee.Convert(0.1)},
		currency.IOTA:     {Withdrawal: fee.Convert(0.1)},
		currency.NAS:      {Withdrawal: fee.Convert(0.5)},
		currency.NAS_ETH:  {Withdrawal: fee.Convert(0.5)},
		currency.TSL:      {Withdrawal: fee.Convert(10)},
		currency.ADA:      {Withdrawal: fee.Convert(1)},
		currency.LSK:      {Withdrawal: fee.Convert(0.1)},
		currency.WAVES:    {Withdrawal: fee.Convert(0.1)},
		currency.BIFI:     {Withdrawal: fee.Convert(0.2)},
		currency.XTZ:      {Withdrawal: fee.Convert(0.1)},
		currency.BNTY:     {Withdrawal: fee.Convert(10)},
		currency.ICX:      {Withdrawal: fee.Convert(0.5)},
		currency.LEND:     {Withdrawal: fee.Convert(20)},
		currency.LUN:      {Withdrawal: fee.Convert(0.2)},
		currency.ELF:      {Withdrawal: fee.Convert(2)},
		currency.SALT:     {Withdrawal: fee.Convert(0.2)},
		currency.FUEL:     {Withdrawal: fee.Convert(2)},
		currency.DRGN:     {Withdrawal: fee.Convert(2)},
		currency.GTC:      {Withdrawal: fee.Convert(2)},
		currency.MDT:      {Withdrawal: fee.Convert(2)},
		currency.QUN:      {Withdrawal: fee.Convert(2)},
		currency.GNX:      {Withdrawal: fee.Convert(2)},
		currency.DDD:      {Withdrawal: fee.Convert(10)},
		currency.OST:      {Withdrawal: fee.Convert(4)},
		currency.BTO:      {Withdrawal: fee.Convert(10)},
		currency.TIO:      {Withdrawal: fee.Convert(10)},
		currency.THETA:    {Withdrawal: fee.Convert(10)},
		currency.SNET:     {Withdrawal: fee.Convert(10)},
		currency.OCN:      {Withdrawal: fee.Convert(10)},
		currency.ZIL:      {Withdrawal: fee.Convert(10)},
		currency.RUFF:     {Withdrawal: fee.Convert(10)},
		currency.TNC:      {Withdrawal: fee.Convert(10)},
		currency.COFI:     {Withdrawal: fee.Convert(10)},
		currency.ZPT:      {Withdrawal: fee.Convert(0.1)},
		currency.JNT:      {Withdrawal: fee.Convert(10)},
		currency.GXS:      {Withdrawal: fee.Convert(1)},
		currency.MTN:      {Withdrawal: fee.Convert(10)},
		currency.BLZ:      {Withdrawal: fee.Convert(2)},
		currency.GEM:      {Withdrawal: fee.Convert(2)},
		currency.DADI:     {Withdrawal: fee.Convert(2)},
		currency.ABT:      {Withdrawal: fee.Convert(2)},
		currency.LEDU:     {Withdrawal: fee.Convert(10)},
		currency.RFR:      {Withdrawal: fee.Convert(10)},
		currency.XLM:      {Withdrawal: fee.Convert(1)},
		currency.MOBI:     {Withdrawal: fee.Convert(1)},
		currency.ONT:      {Withdrawal: fee.Convert(1)},
		currency.NEO:      {Withdrawal: fee.Convert(0)},
		currency.GAS:      {Withdrawal: fee.Convert(0.02)},
		currency.DBC:      {Withdrawal: fee.Convert(10)},
		currency.QLC:      {Withdrawal: fee.Convert(10)},
		currency.MKR:      {Withdrawal: fee.Convert(0.003)},
		currency.MKR_OLD:  {Withdrawal: fee.Convert(0.003)},
		currency.DAI:      {Withdrawal: fee.Convert(2)},
		currency.LRC:      {Withdrawal: fee.Convert(10)},
		currency.OAX:      {Withdrawal: fee.Convert(10)},
		currency.ZRX:      {Withdrawal: fee.Convert(10)},
		currency.PST:      {Withdrawal: fee.Convert(5)},
		currency.TNT:      {Withdrawal: fee.Convert(20)},
		currency.LLT:      {Withdrawal: fee.Convert(10)},
		currency.DNT:      {Withdrawal: fee.Convert(1)},
		currency.DPY:      {Withdrawal: fee.Convert(2)},
		currency.BCDN:     {Withdrawal: fee.Convert(20)},
		currency.STORJ:    {Withdrawal: fee.Convert(3)},
		currency.OMG:      {Withdrawal: fee.Convert(0.2)},
		currency.PAY:      {Withdrawal: fee.Convert(1)},
		currency.EOS:      {Withdrawal: fee.Convert(0.1)},
		currency.EON:      {Withdrawal: fee.Convert(20)},
		currency.IQ:       {Withdrawal: fee.Convert(20)},
		currency.EOSDAC:   {Withdrawal: fee.Convert(20)},
		currency.TIPS:     {Withdrawal: fee.Convert(100)},
		currency.XRP:      {Withdrawal: fee.Convert(1)},
		currency.CNC:      {Withdrawal: fee.Convert(0.1)},
		currency.TIX:      {Withdrawal: fee.Convert(0.1)},
		currency.XMR:      {Withdrawal: fee.Convert(0.05)},
		currency.BTS:      {Withdrawal: fee.Convert(1)},
		currency.XTC:      {Withdrawal: fee.Convert(10)},
		currency.BU:       {Withdrawal: fee.Convert(0.1)},
		currency.DCR:      {Withdrawal: fee.Convert(0.02)},
		currency.BCN:      {Withdrawal: fee.Convert(10)},
		currency.XMC:      {Withdrawal: fee.Convert(0.05)},
		currency.PPS:      {Withdrawal: fee.Convert(0.01)},
		currency.BOE:      {Withdrawal: fee.Convert(5)},
		currency.PLY:      {Withdrawal: fee.Convert(10)},
		currency.MEDX:     {Withdrawal: fee.Convert(100)},
		currency.TRX:      {Withdrawal: fee.Convert(0.1)},
		currency.SMT_ETH:  {Withdrawal: fee.Convert(50)},
		currency.CS:       {Withdrawal: fee.Convert(10)},
		currency.MAN:      {Withdrawal: fee.Convert(10)},
		currency.REM:      {Withdrawal: fee.Convert(10)},
		currency.LYM:      {Withdrawal: fee.Convert(10)},
		currency.INSTAR:   {Withdrawal: fee.Convert(10)},
		currency.BFT:      {Withdrawal: fee.Convert(10)},
		currency.IHT:      {Withdrawal: fee.Convert(10)},
		currency.SENC:     {Withdrawal: fee.Convert(10)},
		currency.TOMO:     {Withdrawal: fee.Convert(10)},
		currency.ELEC:     {Withdrawal: fee.Convert(10)},
		currency.SHIP:     {Withdrawal: fee.Convert(10)},
		currency.TFD:      {Withdrawal: fee.Convert(10)},
		currency.HAV:      {Withdrawal: fee.Convert(10)},
		currency.HUR:      {Withdrawal: fee.Convert(10)},
		currency.LST:      {Withdrawal: fee.Convert(10)},
		currency.LINO:     {Withdrawal: fee.Convert(10)},
		currency.SWTH:     {Withdrawal: fee.Convert(5)},
		currency.NKN:      {Withdrawal: fee.Convert(5)},
		currency.SOUL:     {Withdrawal: fee.Convert(5)},
		currency.GALA_NEO: {Withdrawal: fee.Convert(5)},
		currency.LRN:      {Withdrawal: fee.Convert(5)},
		currency.ADD:      {Withdrawal: fee.Convert(20)},
		currency.MEETONE:  {Withdrawal: fee.Convert(5)},
		currency.DOCK:     {Withdrawal: fee.Convert(20)},
		currency.GSE:      {Withdrawal: fee.Convert(20)},
		currency.RATING:   {Withdrawal: fee.Convert(20)},
		currency.HSC:      {Withdrawal: fee.Convert(100)},
		currency.HIT:      {Withdrawal: fee.Convert(100)},
		currency.DX:       {Withdrawal: fee.Convert(100)},
		currency.BXC:      {Withdrawal: fee.Convert(100)},
		currency.PAX:      {Withdrawal: fee.Convert(5)},
		currency.GARD:     {Withdrawal: fee.Convert(100)},
		currency.FTI:      {Withdrawal: fee.Convert(100)},
		currency.SOP:      {Withdrawal: fee.Convert(100)},
		currency.LEMO:     {Withdrawal: fee.Convert(20)},
		currency.NPXS:     {Withdrawal: fee.Convert(40)},
		currency.QKC:      {Withdrawal: fee.Convert(20)},
		currency.IOTX:     {Withdrawal: fee.Convert(20)},
		currency.RED:      {Withdrawal: fee.Convert(20)},
		currency.LBA:      {Withdrawal: fee.Convert(20)},
		currency.KAN:      {Withdrawal: fee.Convert(20)},
		currency.OPEN:     {Withdrawal: fee.Convert(20)},
		currency.MITH:     {Withdrawal: fee.Convert(20)},
		currency.SKM:      {Withdrawal: fee.Convert(20)},
		currency.XVG:      {Withdrawal: fee.Convert(20)},
		currency.NANO:     {Withdrawal: fee.Convert(20)},
		currency.NBAI:     {Withdrawal: fee.Convert(20)},
		currency.UPP:      {Withdrawal: fee.Convert(20)},
		currency.ATMI:     {Withdrawal: fee.Convert(20)},
		currency.TMT:      {Withdrawal: fee.Convert(20)},
		currency.HT:       {Withdrawal: fee.Convert(1)},
		currency.BNB:      {Withdrawal: fee.Convert(0.3)},
		currency.BBK:      {Withdrawal: fee.Convert(20)},
		currency.EDR:      {Withdrawal: fee.Convert(20)},
		currency.MET:      {Withdrawal: fee.Convert(0.3)},
		currency.TCT:      {Withdrawal: fee.Convert(20)},
		currency.EXC:      {Withdrawal: fee.Convert(10)},
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
