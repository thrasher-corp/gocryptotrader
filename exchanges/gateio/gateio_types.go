package gateio

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// TimeInterval Interval represents interval enum.
type TimeInterval int

const (
	UnderscoreDelimiter = "_"

	// Order book depth intervals

	OrderbookIntervalZero        = "0" //  means no aggregation is applied. default to 0
	OrderbookIntervalZeroPt1     = "0.1"
	OrderbookIntervalZeroPtZero1 = "0.01"
)

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

// *************************************************************

// CurrencyInfo represents currency details with permission.
type CurrencyInfo struct {
	Currency         string `json:"currency"`
	Delisted         bool   `json:"delisted"`
	WithdrawDisabled bool   `json:"withdraw_disabled"`
	WithdrawDelayed  bool   `json:"withdraw_delayed"`
	DepositDisabled  bool   `json:"deposit_disabled"`
	TradeDisabled    bool   `json:"trade_disabled"`
	FixedFeeRate     string `json:"fixed_rate,omitempty"`
	Chain            string `json:"chain"`
}

// CurrencyPairDetail represents a single currency pair detail.
type CurrencyPairDetail struct {
	ID              string  `json:"id"`
	Base            string  `json:"base"`
	Quote           string  `json:"quote"`
	Fee             float64 `json:"fee"`
	MinBaseAmount   float64 `json:"min_base_amount"`
	MinQuoteAmount  float64 `json:"min_quote_amount"`
	AmountPrecision int     `json:"amount_precision"`
	Precision       int     `json:"precision"`
	TradeStatus     string  `json:"trade_status"`
	SellStart       int     `json:"sell_start"`
	BuyStart        int     `json:"buy_start"`
}

// Ticker holds detail ticker information for a currency pair
type Ticker struct {
	CurrencyPair     string    `json:"currency_pair"`
	Last             string    `json:"last"`
	LowestAsk        float64   `json:"lowest_ask"`
	HighestBid       float64   `json:"highest_bid"`
	ChangePercentage string    `json:"change_percentage"`
	ChangeUtc0       string    `json:"change_utc0"`
	ChangeUtc8       string    `json:"change_utc8"`
	BaseVolume       float64   `json:"base_volume"`
	QuoteVolume      float64   `json:"quote_volume"`
	High24H          float64   `json:"high_24h"`
	Low24H           float64   `json:"low_24h"`
	EtfNetValue      string    `json:"etf_net_value"`
	EtfPreNetValue   string    `json:"etf_pre_net_value"`
	EtfPreTimestamp  time.Time `json:"etf_pre_timestamp"`
	EtfLeverage      float64   `json:"etf_leverage"`
}

// OrderbookData holds orderbook ask and bid datas.
type OrderbookData struct {
	ID      int         `json:"id"`
	Current time.Time   `json:"current"` // The timestamp of the response data being generated (in milliseconds)
	Update  time.Time   `json:"update"`  // The timestamp of when the orderbook last changed (in milliseconds)
	Asks    [][2]string `json:"asks"`
	Bids    [][2]string `json:"bids"`
}

// FuturesOrderbookData holds orderbook ask and bid datas for futures.
type FuturesOrderbookData struct {
	ID      int                 `json:"id"`
	Current time.Time           `json:"current"` // The timestamp of the response data being generated (in milliseconds)
	Update  time.Time           `json:"update"`  // The timestamp of when the orderbook last changed (in milliseconds)
	Asks    []map[string]string `json:"asks"`
	Bids    []map[string]string `json:"bids"`
}

// MakeOrderbook parse Orderbook asks/bids Price and Amount and create an Orderbook Instance with asks and bids data in []OrderbookItem.
func (o *OrderbookData) MakeOrderbook() (*Orderbook, error) {
	ob := &Orderbook{
		ID:      o.ID,
		Current: o.Current,
		Update:  o.Update,
	}
	asks := make([]OrderbookItem, len(o.Asks))
	bids := make([]OrderbookItem, len(o.Bids))
	for x := range o.Asks {
		price, er := strconv.ParseFloat(o.Asks[x][0], 64)
		if er != nil {
			return nil, er
		}
		amount, er := strconv.ParseFloat(o.Asks[x][1], 64)
		if er != nil {
			return nil, er
		}
		asks[x] = OrderbookItem{
			Price:  price,
			Amount: amount,
		}
	}
	for x := range o.Bids {
		price, er := strconv.ParseFloat(o.Bids[x][0], 64)
		if er != nil {
			return nil, er
		}
		amount, er := strconv.ParseFloat(o.Bids[x][1], 64)
		if er != nil {
			return nil, er
		}
		bids[x] = OrderbookItem{
			Price:  price,
			Amount: amount,
		}
	}
	ob.Asks = asks
	ob.Bids = bids
	return ob, nil
}

// OrderbookItem stores an orderbook item
type OrderbookItem struct {
	Price  float64
	Amount float64
}

// Orderbook stores the orderbook data
type Orderbook struct {
	ID      int             `json:"id"`
	Current time.Time       `json:"current"` // The timestamp of the response data being generated (in milliseconds)
	Update  time.Time       `json:"update"`  // The timestamp of when the orderbook last changed (in milliseconds)
	Bids    []OrderbookItem `json:"asks"`
	Asks    []OrderbookItem `json:"bids"`
}

// Trade represents market trade.
type Trade struct {
	ID           string    `json:"id"`
	TradingTime  time.Time `json:"create_time"`
	CreateTimeMs time.Time `json:"create_time_ms"`
	OrderID      string    `json:"order_id"`
	Side         string    `json:"side"`
	Role         string    `json:"role"`
	Amount       float64   `json:"amount,string"`
	Price        float64   `json:"price,string"`
	Fee          float64   `json:"fee,string"`
	FeeCurrency  string    `json:"fee_currency"`
	PointFee     string    `json:"point_fee"`
	GtFee        string    `json:"gt_fee"`
}

// Candlestick represents candlestick data point detail.
type Candlestick struct {
	Timestamp      time.Time
	QuoteCcyVolume float64
	ClosePrice     float64
	HighestPrice   float64
	LowestPrice    float64
	OpenPrice      float64
	BaseCcyAmount  float64
}

// TradingFeeRate represents
type TradingFeeRate struct {
	UserID          int    `json:"user_id"`
	TakerFee        string `json:"taker_fee"`
	MakerFee        string `json:"maker_fee"`
	FuturesTakerFee string `json:"futures_taker_fee"`
	FuturesMakerFee string `json:"futures_maker_fee"`
	GtDiscount      bool   `json:"gt_discount"`
	GtTakerFee      string `json:"gt_taker_fee"`
	GtMakerFee      string `json:"gt_maker_fee"`
	LoanFee         string `json:"loan_fee"`
	PointType       string `json:"point_type"`
}

// CurrencyChain currency chain detail.
type CurrencyChain struct {
	Chain              string `json:"chain"`
	ChineseChainName   string `json:"name_cn"`
	ChainName          string `json:"name_en"`
	IsDisabled         int    `json:"is_disabled"`
	IsDepositDisabled  int    `json:"is_deposit_disabled"`
	IsWithdrawDisabled int    `json:"is_withdraw_disabled"`
}

// MarginCurrencyPairInfo represents margin currency pair detailed info.
type MarginCurrencyPairInfo struct {
	ID             string  `json:"id"`
	Base           string  `json:"base"`
	Quote          string  `json:"quote"`
	Leverage       int     `json:"leverage"`
	MinBaseAmount  float64 `json:"min_base_amount,string"`
	MinQuoteAmount float64 `json:"min_quote_amount,string"`
	MaxQuoteAmount float64 `json:"max_quote_amount,string"`
	Status         int     `json:"status"`
}

// OrderbookOfLendingLoan represents order book of lending loans
type OrderbookOfLendingLoan struct {
	Rate   float64 `json:"rate,string"`
	Amount float64 `json:"amount,string"`
	Days   int     `json:"days"`
}

// FuturesContract represents futures contract detailed data.
type FuturesContract struct {
	Name                  string    `json:"name"`
	Type                  string    `json:"type"`
	QuantoMultiplier      string    `json:"quanto_multiplier"`
	RefDiscountRate       string    `json:"ref_discount_rate"`
	OrderPriceDeviate     string    `json:"order_price_deviate"`
	MaintenanceRate       string    `json:"maintenance_rate"`
	MarkType              string    `json:"mark_type"`
	LastPrice             string    `json:"last_price"`
	MarkPrice             string    `json:"mark_price"`
	IndexPrice            string    `json:"index_price"`
	FundingRateIndicative string    `json:"funding_rate_indicative"`
	MarkPriceRound        string    `json:"mark_price_round"`
	FundingOffset         int       `json:"funding_offset"`
	InDelisting           bool      `json:"in_delisting"`
	RiskLimitBase         string    `json:"risk_limit_base"`
	InterestRate          string    `json:"interest_rate"`
	OrderPriceRound       string    `json:"order_price_round"`
	OrderSizeMin          int       `json:"order_size_min"`
	RefRebateRate         string    `json:"ref_rebate_rate"`
	FundingInterval       int       `json:"funding_interval"`
	RiskLimitStep         string    `json:"risk_limit_step"`
	LeverageMin           string    `json:"leverage_min"`
	LeverageMax           string    `json:"leverage_max"`
	RiskLimitMax          string    `json:"risk_limit_max"`
	MakerFeeRate          float64   `json:"maker_fee_rate,string"`
	TakerFeeRate          float64   `json:"taker_fee_rate,string"`
	FundingRate           float64   `json:"funding_rate,string"`
	OrderSizeMax          int       `json:"order_size_max"`
	FundingNextApply      time.Time `json:"funding_next_apply"`
	ConfigChangeTime      time.Time `json:"config_change_time"`
	ShortUsers            int       `json:"short_users"`
	TradeSize             int64     `json:"trade_size"`
	PositionSize          int       `json:"position_size"`
	LongUsers             int       `json:"long_users"`
	FundingImpactValue    string    `json:"funding_impact_value"`
	OrdersLimit           int       `json:"orders_limit"`
	TradeID               int       `json:"trade_id"`
	OrderbookID           int       `json:"orderbook_id"`
}

// TradingHistoryItem represents futures trading history item.
type TradingHistoryItem struct {
	ID         int       `json:"id"`
	CreateTime time.Time `json:"create_time"`
	Contract   string    `json:"contract"`
	Size       float64   `json:"size"`
	Price      float64   `json:"price,string"`
}

// FuturesCandlestick represents futures candlestick data
type FuturesCandlestick struct {
	Timestamp    time.Time `json:"t"`
	Volume       int64     `json:"v"`
	ClosePrice   float64   `json:"c,string"`
	HighestPrice float64   `json:"h,string"`
	LowestPrice  float64   `json:"l,string"`
	OpenPrice    float64   `json:"o,string"`
}

// FuturesTicker represents futures ticker data.
type FuturesTicker struct {
	Contract              string  `json:"contract"`
	Last                  string  `json:"last"`
	Low24H                float64 `json:"low_24h,string"`
	High24H               float64 `json:"high_24h,string"`
	ChangePercentage      string  `json:"change_percentage"`
	TotalSize             float64 `json:"total_size,string"`
	Volume24H             float64 `json:"volume_24h,string"`
	Volume24HBtc          float64 `json:"volume_24h_btc,string"`
	Volume24HUsd          float64 `json:"volume_24h_usd,string"`
	Volume24HBase         float64 `json:"volume_24h_base,string"`
	Volume24HQuote        float64 `json:"volume_24h_quote,string"`
	Volume24HSettle       float64 `json:"volume_24h_settle,string"`
	MarkPrice             float64 `json:"mark_price,string"`
	FundingRate           float64 `json:"funding_rate,string"`
	FundingRateIndicative string  `json:"funding_rate_indicative"`
	IndexPrice            string  `json:"index_price"`
}

// FuturesFundingRate represents futures funding rate response.
type FuturesFundingRate struct {
	Timestamp time.Time `json:"t"`
	Rate      float64   `json:"r"`
}

// InsuranceBalance
type InsuranceBalance struct {
	Timestamp time.Time `json:"t"`
	Balance   float64   `json:"b"`
}

// ContractStat represents futures stats
type ContractStat struct {
	Time                  time.Time `json:"time"`
	LongShortTaker        float64   `json:"lsr_taker"`
	LongShortAccount      float64   `json:"lsr_account"`
	LongLiqSize           float64   `json:"long_liq_size"`
	ShortLiqudiationSize  float64   `json:"short_liq_size"`
	OpenInterest          float64   `json:"open_interest"`
	ShortLiquidationUsd   float64   `json:"short_liq_usd"`
	MarkPrice             float64   `json:"mark_price"`
	TopLongShortSize      float64   `json:"top_lsr_size"`
	ShortLiqudationAmount float64   `json:"short_liq_amount"`
	LongLiqudiationAmount float64   `json:"long_liq_amount"`
	OpenInterestUsd       float64   `json:"open_interest_usd"`
	TopLongShortAccount   float64   `json:"top_lsr_account"`
	LongLiqudationUsd     float64   `json:"long_liq_usd"`
}

// IndexConstituent represents index constituents
type IndexConstituent struct {
	Index        string `json:"index"`
	Constituents []struct {
		Exchange string   `json:"exchange"`
		Symbols  []string `json:"symbols"`
	} `json:"constituents"`
}

// LiquidationHistory represents  liquidation history for a specifies settle.
type LiquidationHistory struct {
	Time             time.Time `json:"time"`
	Contract         string    `json:"contract"`
	Size             int       `json:"size"`
	Leverage         string    `json:"leverage"`
	Margin           string    `json:"margin"`
	EntryPrice       string    `json:"entry_price"`
	LiquidationPrice string    `json:"liq_price"`
	MarkPrice        string    `json:"mark_price"`
	OrderID          int       `json:"order_id"`
	OrderPrice       string    `json:"order_price"`
	FillPrice        string    `json:"fill_price"`
	Left             int       `json:"left"`
}

type DeliveryContract struct {
	Name                string    `json:"name"`
	Underlying          string    `json:"underlying"`
	Cycle               string    `json:"cycle"`
	Type                string    `json:"type"`
	QuantoMultiplier    string    `json:"quanto_multiplier"`
	MarkType            string    `json:"mark_type"`
	LastPrice           string    `json:"last_price"`
	MarkPrice           string    `json:"mark_price"`
	IndexPrice          string    `json:"index_price"`
	BasisRate           string    `json:"basis_rate"`
	BasisValue          string    `json:"basis_value"`
	BasisImpactValue    string    `json:"basis_impact_value"`
	SettlePrice         string    `json:"settle_price"`
	SettlePriceInterval int       `json:"settle_price_interval"`
	SettlePriceDuration int       `json:"settle_price_duration"`
	SettleFeeRate       string    `json:"settle_fee_rate"`
	OrderPriceRound     string    `json:"order_price_round"`
	MarkPriceRound      string    `json:"mark_price_round"`
	LeverageMin         string    `json:"leverage_min"`
	LeverageMax         string    `json:"leverage_max"`
	MaintenanceRate     string    `json:"maintenance_rate"`
	RiskLimitBase       string    `json:"risk_limit_base"`
	RiskLimitStep       string    `json:"risk_limit_step"`
	RiskLimitMax        string    `json:"risk_limit_max"`
	MakerFeeRate        string    `json:"maker_fee_rate"`
	TakerFeeRate        string    `json:"taker_fee_rate"`
	RefDiscountRate     string    `json:"ref_discount_rate"`
	RefRebateRate       string    `json:"ref_rebate_rate"`
	OrderPriceDeviate   string    `json:"order_price_deviate"`
	OrderSizeMin        int       `json:"order_size_min"`
	OrderSizeMax        int       `json:"order_size_max"`
	OrdersLimit         int       `json:"orders_limit"`
	OrderbookID         int       `json:"orderbook_id"`
	TradeID             int       `json:"trade_id"`
	TradeSize           int       `json:"trade_size"`
	PositionSize        int       `json:"position_size"`
	ExpireTime          time.Time `json:"expire_time"`
	ConfigChangeTime    time.Time `json:"config_change_time"`
	InDelisting         bool      `json:"in_delisting"`
}

// DeliveryTradingHistory represents futures trading history
type DeliveryTradingHistory struct {
	ID         int64     `json:"id"`
	CreateTime time.Time `json:"create_time"`
	Contract   string    `json:"contract"`
	Size       float64   `json:"size"`
	Price      float64   `json:"price,string"`
}

// OptionUnderlying represents option underlying and it's index price.
type OptionUnderlying struct {
	Name       string  `json:"name"`
	IndexPrice float64 `json:"index_price,string"`
}

// OptionContract represents an option contract detail.
type OptionContract struct {
	Name              string    `json:"name"`
	Tag               string    `json:"tag"`
	IsCall            bool      `json:"is_call"`
	StrikePrice       string    `json:"strike_price"`
	LastPrice         string    `json:"last_price"`
	MarkPrice         string    `json:"mark_price"`
	OrderbookID       int       `json:"orderbook_id"`
	TradeID           int       `json:"trade_id"`
	TradeSize         int       `json:"trade_size"`
	PositionSize      int       `json:"position_size"`
	Underlying        string    `json:"underlying"`
	UnderlyingPrice   string    `json:"underlying_price"`
	Multiplier        string    `json:"multiplier"`
	OrderPriceRound   string    `json:"order_price_round"`
	MarkPriceRound    string    `json:"mark_price_round"`
	MakerFeeRate      string    `json:"maker_fee_rate"`
	TakerFeeRate      string    `json:"taker_fee_rate"`
	PriceLimitFeeRate string    `json:"price_limit_fee_rate"`
	RefDiscountRate   string    `json:"ref_discount_rate"`
	RefRebateRate     string    `json:"ref_rebate_rate"`
	OrderPriceDeviate string    `json:"order_price_deviate"`
	OrderSizeMin      int       `json:"order_size_min"`
	OrderSizeMax      int       `json:"order_size_max"`
	OrdersLimit       int       `json:"orders_limit"`
	CreateTime        time.Time `json:"create_time"`
	ExpirationTime    time.Time `json:"expiration_time"`
}

// OptionSettlement list settlement history
type OptionSettlement struct {
	Time        time.Time `json:"time"`
	Profit      string    `json:"profit"`
	Fee         string    `json:"fee"`
	SettlePrice string    `json:"settle_price"`
	Contract    string    `json:"contract"`
	StrikePrice string    `json:"strike_price"`
}

// SwapCurrencies represents Flash Swap supported currencies
type SwapCurrencies struct {
	Currency  string   `json:"currency"`
	MinAmount float64  `json:"min_amount,string"`
	MaxAmount float64  `json:"max_amount,string"`
	Swappable []string `json:"swappable"`
}

// MyOptionSettlement represents option private settlement
type MyOptionSettlement struct {
	Size         float64   `json:"size"`
	SettleProfit float64   `json:"settle_profit,string"`
	Contract     string    `json:"contract"`
	StrikePrice  float64   `json:"strike_price,string"`
	Time         time.Time `json:"time"`
	SettlePrice  float64   `json:"settle_price,string"`
	Underlying   string    `json:"underlying"`
	RealisedPnl  string    `json:"realised_pnl"`
	Fee          float64   `json:"fee,string"`
}

// OptionsTicker represents  tickers of options contracts
type OptionsTicker struct {
	Name                  string `json:"name"`
	LastPrice             string `json:"last_price"`
	MarkPrice             string `json:"mark_price"`
	PositionSize          int    `json:"position_size"`
	Ask1Size              int    `json:"ask1_size"`
	Ask1Price             string `json:"ask1_price"`
	Bid1Size              int    `json:"bid1_size"`
	Bid1Price             string `json:"bid1_price"`
	Vega                  string `json:"vega"`
	Theta                 string `json:"theta"`
	Rho                   string `json:"rho"`
	Gamma                 string `json:"gamma"`
	Delta                 string `json:"delta"`
	MarkImpliedVolatility string `json:"mark_iv"`
	BidImpliedVolatility  string `json:"bid_iv"`
	AskImpliedVolatility  string `json:"ask_iv"`
	Leverage              string `json:"leverage"`
}

// OptionsUnderlyingTicker represents underlying ticker
type OptionsUnderlyingTicker struct {
	TradePut   float64 `json:"trade_put"`
	TradeCall  float64 `json:"trade_call"`
	IndexPrice float64 `json:"index_price,string"`
}

// OptionAccount represents option account.
type OptionAccount struct {
	User          int64  `json:"user"`
	Currency      string `json:"currency"`
	ShortEnabled  bool   `json:"short_enabled"`
	Total         string `json:"total"`
	UnrealisedPnl string `json:"unrealised_pnl"`
	InitMargin    string `json:"init_margin"`
	MaintMargin   string `json:"maint_margin"`
	OrderMargin   string `json:"order_margin"`
	Available     string `json:"available"`
	Point         string `json:"point"`
}

// AccountBook represents account changing history item
type AccountBook struct {
	ChangeTime    time.Time `json:"time"`
	AccountChange float64   `json:"change,string"`
	Balance       float64   `json:"balance,string"`
	CustomText    string    `json:"text"`
	ChangingType  string    `json:"type"`
}

// UsersPositionForUnderlying represents user's position for specified underlying.
type UsersPositionForUnderlying struct {
	User          int     `json:"user"`
	Contract      string  `json:"contract"`
	Size          int     `json:"size"`
	EntryPrice    float64 `json:"entry_price,string"`
	RealisedPnl   float64 `json:"realised_pnl,string"`
	MarkPrice     float64 `json:"mark_price,string"`
	UnrealisedPnl float64 `json:"unrealised_pnl,string"`
	PendingOrders int     `json:"pending_orders"`
	CloseOrder    struct {
		ID    int    `json:"id"`
		Price string `json:"price"`
		IsLiq bool   `json:"is_liq"`
	} `json:"close_order"`
}

// ContractClosePosition represents user's liquidation history
type ContractClosePosition struct {
	PositionCloseTime time.Time `json:"time"`
	Pnl               string    `json:"pnl"`
	SettleSize        string    `json:"settle_size"`
	Side              string    `json:"side"` // Position side, long or short
	FuturesContract   string    `json:"contract"`
	CloseOrderText    string    `json:"text"`
}

// OptionOrderParam represents option order request body
type OptionOrderParam struct {
	OrderSize   float64 `json:"size"`              //** [[Note]] Order size. Specify positive number to make a bid, and negative number to ask
	Iceberg     float64 `json:"iceberg,omitempty"` // Display size for iceberg order. 0 for non-iceberg. Note that you will have to pay the taker fee for the hidden size
	Contract    string  `json:"contract"`
	Text        string  `json:"text,omitempty"`
	TimeInForce string  `json:"tif,omitempty"`
	Price       float64 `json:"price,string,omitempty"`
	// Close Set as true to close the position, with size set to 0
	Close      bool `json:"close,omitempty"`
	ReduceOnly bool `json:"reduce_only,omitempty"`
}

// OptionOrderResponse represents option order response detail
type OptionOrderResponse struct {
	Status               string    `json:"status"`
	Size                 int       `json:"size"`
	OptionOrderID        int       `json:"id"`
	Iceberg              int       `json:"iceberg"`
	IsOrderLiquidation   bool      `json:"is_liq"`
	IsOrderPositionClose bool      `json:"is_close"`
	Contract             string    `json:"contract"`
	Text                 string    `json:"text"`
	FillPrice            string    `json:"fill_price"`
	FinishAs             string    `json:"finish_as"` //  finish_as 	filled, cancelled, liquidated, ioc, auto_deleveraged, reduce_only, position_closed, reduce_out
	Left                 int       `json:"left"`
	TimeInForce          string    `json:"tif"`
	IsReduceOnly         bool      `json:"is_reduce_only"`
	CreateTime           time.Time `json:"create_time"`
	FinishTime           time.Time `json:"finish_time"`
	Price                float64   `json:"price,string"`

	TakerFee        string `json:"tkrf,omitempty"`
	MakerFee        string `json:"mkrf,omitempty"`
	ReferenceUserID string `json:"refu"`
}

// OptionTradingHistory list personal trading history
type OptionTradingHistory struct {
	UnderlyingPrice string    `json:"underlying_price"`
	Size            int       `json:"size"`
	Contract        string    `json:"contract"`
	ID              int       `json:"id"`
	TradeRole       string    `json:"role"`
	CreateTime      time.Time `json:"create_time"`
	OrderID         int       `json:"order_id"`
	Price           string    `json:"price"`
}

// WithdrawalResponse represents withdrawal response
type WithdrawalResponse struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	Currency      string    `json:"currency"`
	Address       string    `json:"address"`
	Transactionid string    `json:"txid"`
	Amount        string    `json:"amount"`
	Memo          string    `json:"memo"`
	Status        string    `json:"status"`
	Chain         string    `json:"chain"`
}

// WithdrawalRequestParam represents currency withdrawal request param.
type WithdrawalRequestParam struct {
	Currency currency.Code `json:"currency"`
	Address  string        `json:"address"`
	Amount   float64       `json:"amount,string"`
	Memo     string        `json:"memo"`
	Chain    string        `json:"chain"`
}

// CurrencyDepositAddressInfo represents a crypto deposit address
type CurrencyDepositAddressInfo struct {
	Currency            string `json:"currency"`
	Address             string `json:"address"`
	MultichainAddresses []struct {
		Chain        string `json:"chain"`
		Address      string `json:"address"`
		PaymentID    string `json:"payment_id"`
		PaymentName  string `json:"payment_name"`
		ObtainFailed int    `json:"obtain_failed"`
	} `json:"multichain_addresses"`
}

// DepositRecord represents deposit record item
type DepositRecord struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	Currency      string    `json:"currency"`
	Address       string    `json:"address"`
	TransactionID string    `json:"txid"`
	Amount        float64   `json:"amount,string"`
	Memo          string    `json:"memo"`
	Status        string    `json:"status"`
	Chain         string    `json:"chain"`
}

// TransferCurrencyParam represents currency transfer.
type TransferCurrencyParam struct {
	Currency     currency.Code `json:"currency"`
	From         asset.Item    `json:"from"`
	To           asset.Item    `json:"to"`
	Amount       float64       `json:"amount,string"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	Settle       string        `json:"settle"`
}

// TransactionIDResponse represents transaction ID
type TransactionIDResponse struct {
	TransactionID int64 `json:"tx_id"`
}

// SubAccountTransferParam represents currency subaccount transfer request param
type SubAccountTransferParam struct {
	Currency       currency.Code `json:"currency"`
	SubAccount     string        `json:"sub_account"`
	Direction      string        `json:"direction"`
	Amount         float64       `json:"amount,string"`
	SubAccountType asset.Item    `json:"sub_account_type"`
}

// SubAccountTransferResponse represents transfer records between main and sub accounts
type SubAccountTransferResponse struct {
	UID            string    `json:"uid"`
	Timestamp      time.Time `json:"timest"`
	Source         string    `json:"source"`
	Currency       string    `json:"currency"`
	SubAccount     string    `json:"sub_account"`
	Direction      string    `json:"direction"`
	Amount         float64   `json:"amount,string"`
	SubAccountType string    `json:"sub_account_type"`
}

// WithdrawalStatus represents currency withdrawal status
type WithdrawalStatus struct {
	Currency               string            `json:"currency"`
	CurrencyName           string            `json:"name"`
	CurrencyNameChinese    string            `json:"name_cn"`
	Deposit                float64           `json:"deposit,string"`
	WithdrawPercent        string            `json:"withdraw_percent"`
	WithdrawFix            string            `json:"withdraw_fix"`
	WithdrawDayLimit       string            `json:"withdraw_day_limit"`
	WithdrawDayLimitRemain string            `json:"withdraw_day_limit_remain"`
	WithdrawAmountMini     string            `json:"withdraw_amount_mini"`
	WithdrawEachtimeLimit  string            `json:"withdraw_eachtime_limit"`
	WithdrawFixOnChains    map[string]string `json:"withdraw_fix_on_chains"`
	AdditionalProperties   string            `json:"additionalProperties"`
}

// SubAccountBalance represents sub account balance for specific sub account and several currencies
type SubAccountBalance struct {
	UID       string            `json:"uid"`
	Available map[string]string `json:"available"`
}

// SubAccountMarginBalance represents sub account margin balance for specific sub account and several currencies
type SubAccountMarginBalance struct {
	UID       string              `json:"uid"`
	Available []MarginAccountItem `json:"available"`
}

// MarginAccountItem margin account item
type MarginAccountItem struct {
	Locked       bool   `json:"locked"`
	CurrencyPair string `json:"currency_pair"`
	Risk         string `json:"risk"`
	Base         struct {
		Available string `json:"available"`
		Borrowed  string `json:"borrowed"`
		Interest  string `json:"interest"`
		Currency  string `json:"currency"`
		Locked    string `json:"locked"`
	} `json:"base"`
	Quote struct {
		Available string `json:"available"`
		Borrowed  string `json:"borrowed"`
		Interest  string `json:"interest"`
		Currency  string `json:"currency"`
		Locked    string `json:"locked"`
	} `json:"quote"`
}

// MarginAccountBalanceChangeInfo represents margin account balance
type MarginAccountBalanceChangeInfo struct {
	ID           string    `json:"id"`
	Time         time.Time `json:"time"`
	TimeMs       time.Time `json:"time_ms"`
	Currency     string    `json:"currency"`
	CurrencyPair string    `json:"currency_pair"`
	Change       string    `json:"change"`
	Balance      string    `json:"balance"`
}

// MarginFundingAccountItem represents funding account list item.
type MarginFundingAccountItem struct {
	Currency  string `json:"currency"`
	Available string `json:"available"`
	Locked    string `json:"locked"`
	Lent      string `json:"lent"`
	TotalLent string `json:"total_lent"`
}

// MarginLoanRequestParam represents margin lend or borrow request param
type MarginLoanRequestParam struct {
	Side         string        `json:"side"`
	Currency     currency.Code `json:"currency"`
	Rate         float64       `json:"rate,string,omitempty"`
	Amount       float64       `json:"amount,string,omitempty"`
	Days         int           `json:"days,omitempty"`
	AutoRenew    bool          `json:"auto_renew,omitempty"`
	CurrencyPair currency.Pair `json:"currency_pair,omitempty"`
	FeeRate      float64       `json:"fee_rate,string,omitempty"`
	OrigID       string        `json:"orig_id,omitempty"`
	Text         string        `json:"text,omitempty"`
}

// MarginLoanResponse represents lending or borrow response.
type MarginLoanResponse struct {
	Side         string `json:"side"`
	Currency     string `json:"currency"`
	Amount       string `json:"amount"`
	Rate         string `json:"rate,omitempty"`
	Days         int    `json:"days,omitempty"`
	AutoRenew    bool   `json:"auto_renew,omitempty"`
	CurrencyPair string `json:"currency_pair,omitempty"`
	FeeRate      string `json:"fee_rate,omitempty"`
	OrigID       string `json:"orig_id,omitempty"`
	Text         string `json:"text,omitempty"`
}

// SubAccountCrossMarginInfo represents subaccount's cross_margin account info
type SubAccountCrossMarginInfo struct {
	UID       string `json:"uid"`
	Available struct {
		UserID                     int    `json:"user_id"`
		Locked                     bool   `json:"locked"`
		Total                      string `json:"total"`
		Borrowed                   string `json:"borrowed"`
		Interest                   string `json:"interest"`
		BorrowedNet                string `json:"borrowed_net"`
		Net                        string `json:"net"`
		Leverage                   string `json:"leverage"`
		Risk                       string `json:"risk"`
		TotalInitialMargin         string `json:"total_initial_margin"`
		TotalMarginBalance         string `json:"total_margin_balance"`
		TotalMaintenanceMargin     string `json:"total_maintenance_margin"`
		TotalInitialMarginRate     string `json:"total_initial_margin_rate"`
		TotalMaintenanceMarginRate string `json:"total_maintenance_margin_rate"`
		TotalAvailableMargin       string `json:"total_available_margin"`
		Balances                   map[string]struct {
			Available string `json:"available"`
			Freeze    string `json:"freeze"`
			Borrowed  string `json:"borrowed"`
			Interest  string `json:"interest"`
		} `json:"balances"`
	} `json:"available"`
}

// WalletSavedAddress represents currency saved address
type WalletSavedAddress struct {
	Currency string `json:"currency"`
	Chain    string `json:"chain"`
	Address  string `json:"address"`
	Name     string `json:"name"`
	Tag      string `json:"tag"`
	Verified string `json:"verified"`
}

// PersonalTradingFee represents personal trading fee for specific currency pair
type PersonalTradingFee struct {
	UserID          int    `json:"user_id"`
	TakerFee        string `json:"taker_fee"`
	MakerFee        string `json:"maker_fee"`
	FuturesTakerFee string `json:"futures_taker_fee"`
	FuturesMakerFee string `json:"futures_maker_fee"`
	GtDiscount      bool   `json:"gt_discount"`
	GtTakerFee      string `json:"gt_taker_fee"`
	GtMakerFee      string `json:"gt_maker_fee"`
	LoanFee         string `json:"loan_fee"`
	PointType       string `json:"point_type"`
}

// UsersAllAccountBalance represents user all account balances.
type UsersAllAccountBalance struct {
	Details map[string]CurrencyBalanceAmount `json:"details"`
	Total   CurrencyBalanceAmount            `json:"total"`
}

// CurrencyBalanceAmount represents currency and its amount.
type CurrencyBalanceAmount struct {
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
}

// SpotTradingFeeRate user trading fee rates
type SpotTradingFeeRate struct {
	UserID          int    `json:"user_id"`
	TakerFee        string `json:"taker_fee"`
	MakerFee        string `json:"maker_fee"`
	FuturesTakerFee string `json:"futures_taker_fee"`
	FuturesMakerFee string `json:"futures_maker_fee"`
	GtDiscount      bool   `json:"gt_discount"`
	GtTakerFee      string `json:"gt_taker_fee"`
	GtMakerFee      string `json:"gt_maker_fee"`
	LoanFee         string `json:"loan_fee"`
	PointType       string `json:"point_type"`
}

// SpotAccount represents spot account
type SpotAccount struct {
	Currency  string `json:"currency"`
	Available string `json:"available"`
	Locked    string `json:"locked"`
}

// CreateOrderRequestData represents a single order creation param.
type CreateOrderRequestData struct {
	Text         string        `json:"text,omitempty"`
	CurrencyPair currency.Pair `json:"currency_pair,omitempty"`
	Type         string        `json:"type,omitempty"`
	Account      asset.Item    `json:"account,omitempty"`
	Side         string        `json:"side,omitempty"`
	Iceberg      string        `json:"iceberg,omitempty"`
	Amount       float64       `json:"amount,string,omitempty"`
	Price        float64       `json:"price,string,omitempty"`
	TimeInForce  string        `json:"time_in_force,omitempty"`
	AutoBorrow   bool          `json:"auto_borrow,omitempty"`
}

// SpotOrder represents create order response.
type SpotOrder struct {
	ID                 string    `json:"id,omitempty"`
	Text               string    `json:"text,omitempty"`
	Succeeded          bool      `json:"succeeded,omitempty"`
	Label              string    `json:"label,omitempty"`
	Message            string    `json:"message,omitempty"`
	CreateTime         time.Time `json:"create_time,omitempty"`
	UpdateTime         time.Time `json:"update_time,omitempty"`
	CreateTimeMs       time.Time `json:"create_time_ms,omitempty"`
	UpdateTimeMs       time.Time `json:"update_time_ms,omitempty"`
	CurrencyPair       string    `json:"currency_pair,omitempty"`
	Status             string    `json:"status,omitempty"`
	Type               string    `json:"type,omitempty"`
	Account            string    `json:"account,omitempty"`
	Side               string    `json:"side,omitempty"`
	Amount             string    `json:"amount,omitempty"`
	Price              string    `json:"price,omitempty"`
	TimeInForce        string    `json:"time_in_force,omitempty"`
	Iceberg            string    `json:"iceberg,omitempty"`
	Left               string    `json:"left,omitempty"`
	FilledTotal        string    `json:"filled_total,omitempty"`
	Fee                string    `json:"fee,omitempty"`
	FeeCurrency        string    `json:"fee_currency,omitempty"`
	PointFee           string    `json:"point_fee,omitempty"`
	GtFee              string    `json:"gt_fee,omitempty"`
	GtDiscount         bool      `json:"gt_discount,omitempty"`
	RebatedFee         string    `json:"rebated_fee,omitempty"`
	RebatedFeeCurrency string    `json:"rebated_fee_currency,omitempty"`
}

// SpotOrdersDetail represents list of orders for specific currency pair
type SpotOrdersDetail struct {
	CurrencyPair string      `json:"currency_pair"`
	Total        int         `json:"total"`
	Orders       []SpotOrder `json:"orders"`
}

// ClosePositionRequestParam represents close position when cross currency is disable.
type ClosePositionRequestParam struct {
	Text         string        `json:"text"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	Amount       float64       `json:"amount,string"`
	Price        float64       `json:"price,string"`
}

// CancelOrderByIDParam represents cancel order by id request param.
type CancelOrderByIDParam struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	ID           string        `json:"id"`
}

// CancelOrderByIDResponse represents calcel order response when deleted by id.
type CancelOrderByIDResponse struct {
	CurrencyPair string      `json:"currency_pair"`
	ID           string      `json:"id"`
	Succeeded    bool        `json:"succeeded"`
	Label        interface{} `json:"label"`
	Message      interface{} `json:"message"`
}

// SpotPersonalTradeHistory represents personal trading history.
type SpotPersonalTradeHistory struct {
	ID           string    `json:"id"`
	CreateTime   time.Time `json:"create_time"`
	CreateTimeMs time.Time `json:"create_time_ms"`
	OrderID      string    `json:"order_id"`
	Side         string    `json:"side"`
	Role         string    `json:"role"`
	Amount       float64   `json:"amount,string"`
	Price        float64   `json:"price,string"`
	Fee          string    `json:"fee"`
	FeeCurrency  string    `json:"fee_currency"`
	PointFee     string    `json:"point_fee"`
	GtFee        string    `json:"gt_fee"`
}

// CountdownCancelOrderParam represents countdown cancel order params
type CountdownCancelOrderParam struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	Timeout      int64         `json:"timeout"` // timeout: Countdown time, in seconds At least 5 seconds, 0 means cancel the countdown
}

// TriggerTimeResponse represents trigger time as a response for countdown candle order response
type TriggerTimeResponse struct {
	TriggerTime time.Time `json:"trigger_time"`
}

// PriceTriggeredOrderParam represents price triggered order request.
type PriceTriggeredOrderParam struct {
	Trigger TriggerPriceInfo `json:"trigger"`
	Put     PutOrderData     `json:"put"`
	Market  currency.Pair    `json:"market"`
}

// TriggerPriceInfo represents a trigger price and related information for Price triggered order
type TriggerPriceInfo struct {
	Price      float64 `json:"price,string"`
	Rule       string  `json:"rule"`
	Expiration int     `json:"expiration,omitempty"`
}

// PutOrderData represents order detail for price triggered order request
type PutOrderData struct {
	Type        string  `json:"type"`
	Side        string  `json:"side"`
	Price       float64 `json:"price,string"`
	Amount      float64 `json:"amount,string"`
	Account     string  `json:"account"`
	TimeInForce string  `json:"time_in_force,omitempty"`
}

// OrderID represents order creation ID response.
type OrderID struct {
	ID int64 `json:"id"`
}

// SpotPriceTriggeredOrder represents spot price triggered order response data.
type SpotPriceTriggeredOrder struct {
	Trigger      TriggerPriceInfo `json:"trigger"`
	Put          PutOrderData     `json:"put"`
	ID           int64            `json:"id"`
	User         int64            `json:"user"`
	CreationTime time.Time        `json:"ctime"`
	FireTime     time.Time        `json:"ftime"`
	FiredOrderID int64            `json:"fired_order_id"`
	Status       string           `json:"status,omitempty"`
	Reason       string           `json:"reason,omitempty"`
	Market       string           `json:"market,omitempty"`
}

// ModifyLoanRequestParam represents request parameters for modify loan request
type ModifyLoanRequestParam struct {
	Currency     currency.Code `json:"currency"`
	Side         string        `json:"side"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	AutoRenew    bool          `json:"auto_renew"`
	LoanID       string        `json:"loan_id,omitempty"`
}

// RepayLoanRequestParam represents loan repay request parameters
type RepayLoanRequestParam struct {
	CurrencyPair currency.Pair `json:"currency_pair"`
	Currency     currency.Code `json:"currency"`
	Mode         string        `json:"mode"`
	Amount       float64       `json:"amount,string"`
}

// LoanRepaymentRecord represents loan repayment history record item.
type LoanRepaymentRecord struct {
	ID         string    `json:"id"`
	CreateTime time.Time `json:"create_time"`
	Principal  string    `json:"principal"`
	Interest   string    `json:"interest"`
}

// LoanRecord represents loan repayment specific record
type LoanRecord struct {
	ID             string    `json:"id"`
	LoanID         string    `json:"loan_id"`
	CreateTime     time.Time `json:"create_time"`
	ExpireTime     time.Time `json:"expire_time"`
	Status         string    `json:"status"`
	BorrowUserID   string    `json:"borrow_user_id"`
	Currency       string    `json:"currency"`
	Rate           float64   `json:"rate,string"`
	Amount         float64   `json:"amount,string"`
	Days           int       `json:"days"`
	AutoRenew      bool      `json:"auto_renew"`
	Repaid         float64   `json:"repaid,string"`
	PaidInterest   string    `json:"paid_interest"`
	UnpaidInterest string    `json:"unpaid_interest"`
}

// OnOffStatus represents on or off status response status
type OnOffStatus struct {
	Status string `json:"status"`
}

// MaxTransferAndLoanAmount represents the maximum amount to transfer, borrow, or lend for specific currency and currency pair
type MaxTransferAndLoanAmount struct {
	Currency     currency.Code `json:"currency"`
	CurrencyPair currency.Pair `json:"currency_pair"`
	Amount       float64       `json:"amount,string"`
}

// CrossMarginCurrencies represents a currency supported by cross margin
type CrossMarginCurrencies struct {
	Name                 string  `json:"name"`
	Rate                 float64 `json:"rate,string"`
	Precesion            float64 `json:"prec,string"`
	Discount             string  `json:"discount"`
	MinBorrowAmount      float64 `json:"min_borrow_amount,string"`
	UserMaxBorrowAmount  float64 `json:"user_max_borrow_amount,string"`
	TotalMaxBorrowAmount float64 `json:"total_max_borrow_amount,string"`
	Price                float64 `json:"price,string"`
	Status               int     `json:"status"`
}

// CrossMarginCurrencyBalance represents the currency detailed balance information for cross margin
type CrossMarginCurrencyBalance struct {
	Available string `json:"available"`
	Freeze    string `json:"freeze"`
	Borrowed  string `json:"borrowed"`
	Interest  string `json:"interest"`
}

// CrossMarginAccount represents the account detail for cross margin account balance
type CrossMarginAccount struct {
	UserID                     int                                   `json:"user_id"`
	Locked                     bool                                  `json:"locked"`
	Balances                   map[string]CrossMarginCurrencyBalance `json:"balances"`
	Total                      float64                               `json:"total,string"`
	Borrowed                   float64                               `json:"borrowed,string"`
	Interest                   float64                               `json:"interest,string"`
	Risk                       float64                               `json:"risk,string"`
	TotalInitialMargin         string                                `json:"total_initial_margin"`
	TotalMarginBalance         string                                `json:"total_margin_balance"`
	TotalMaintenanceMargin     string                                `json:"total_maintenance_margin"`
	TotalInitialMarginRate     string                                `json:"total_initial_margin_rate"`
	TotalMaintenanceMarginRate string                                `json:"total_maintenance_margin_rate"`
	TotalAvailableMargin       string                                `json:"total_available_margin"`
}

// CrossMarginAccountHistoryItem represents a cross margin account change history item
type CrossMarginAccountHistoryItem struct {
	ID       string    `json:"id"`
	Time     time.Time `json:"time"`
	Currency string    `json:"currency"`
	Change   string    `json:"change"`
	Balance  float64   `json:"balance,string"`
	Type     string    `json:"type"`
}

// CrossMarginBorrowLoanParams represents a cross margin borrow loan parameters
type CrossMarginBorrowLoanParams struct {
	Currency currency.Code `json:"currency"`
	Amount   float64       `json:"amount"`
	Text     string        `json:"text"`
}

// CrossMarginLoanResponse represents a cross margin borrow loan response
type CrossMarginLoanResponse struct {
	ID             string    `json:"id"`
	CreateTime     time.Time `json:"create_time"`
	UpdateTime     time.Time `json:"update_time"`
	Currency       string    `json:"currency"`
	Amount         float64   `json:"amount,string"`
	Text           string    `json:"text"`
	Status         int       `json:"status"`
	Repaid         string    `json:"repaid"`
	RepaidInterest float64   `json:"repaid_interest,string"`
	UnpaidInterest float64   `json:"unpaid_interest,string"`
}

// CurrencyAndAmount represents request parameters for repayment
type CurrencyAndAmount struct {
	Currency currency.Code `json:"currency"`
	Amount   float64       `json:"amount,string"`
}

// RepaymentHistoryItem represents an item in a repayment history.
type RepaymentHistoryItem struct {
	ID         string    `json:"id"`
	CreateTime time.Time `json:"create_time"`
	LoanID     string    `json:"loan_id"`
	Currency   string    `json:"currency"`
	Principal  float32   `json:"principal,string"`
	Interest   float32   `json:"interest,string"`
}

// FlashSwapOrderParams represents create flash swap order request parameters.
type FlashSwapOrderParams struct {
	PreviewID    string        `json:"preview_id"`
	SellCurrency currency.Code `json:"sell_currency"`
	SellAmount   float64       `json:"sell_amount,string,omitempty"`
	BuyCurrency  currency.Code `json:"buy_currency"`
	BuyAmount    float64       `json:"buy_amount,string,omitempty"`
}

// FlashSwapOrderResponse represents create flash swap order response
type FlashSwapOrderResponse struct {
	ID           int       `json:"id"`
	CreateTime   time.Time `json:"create_time"`
	UpdateTime   time.Time `json:"update_time"`
	UserID       int       `json:"user_id"`
	SellCurrency string    `json:"sell_currency"`
	SellAmount   float64   `json:"sell_amount,string"`
	BuyCurrency  string    `json:"buy_currency"`
	BuyAmount    float64   `json:"buy_amount,string"`
	Price        float64   `json:"price,string"`
	Status       int       `json:"status"`
}
