package kraken

import "github.com/thrasher-/gocryptotrader/currency"

// TimeResponse type
type TimeResponse struct {
	Unixtime int64  `json:"unixtime"`
	Rfc1123  string `json:"rfc1123"`
}

// Asset holds asset information
type Asset struct {
	Altname         string `json:"altname"`
	AclassBase      string `json:"aclass_base"`
	Decimals        int    `json:"decimals"`
	DisplayDecimals int    `json:"display_decimals"`
}

// AssetPairs holds asset pair information
type AssetPairs struct {
	Altname           string      `json:"altname"`
	AclassBase        string      `json:"aclass_base"`
	Base              string      `json:"base"`
	AclassQuote       string      `json:"aclass_quote"`
	Quote             string      `json:"quote"`
	Lot               string      `json:"lot"`
	PairDecimals      int         `json:"pair_decimals"`
	LotDecimals       int         `json:"lot_decimals"`
	LotMultiplier     int         `json:"lot_multiplier"`
	LeverageBuy       []int       `json:"leverage_buy"`
	LeverageSell      []int       `json:"leverage_sell"`
	Fees              [][]float64 `json:"fees"`
	FeesMaker         [][]float64 `json:"fees_maker"`
	FeeVolumeCurrency string      `json:"fee_volume_currency"`
	MarginCall        int         `json:"margin_call"`
	MarginStop        int         `json:"margin_stop"`
}

// Ticker is a standard ticker type
type Ticker struct {
	Ask    float64
	Bid    float64
	Last   float64
	Volume float64
	VWAP   float64
	Trades int64
	Low    float64
	High   float64
	Open   float64
}

// Tickers stores a map of tickers
type Tickers map[string]Ticker

// TickerResponse holds ticker information before its put into the Ticker struct
type TickerResponse struct {
	Ask    []string `json:"a"`
	Bid    []string `json:"b"`
	Last   []string `json:"c"`
	Volume []string `json:"v"`
	VWAP   []string `json:"p"`
	Trades []int64  `json:"t"`
	Low    []string `json:"l"`
	High   []string `json:"h"`
	Open   string   `json:"o"`
}

// OpenHighLowClose contains ticker event information
type OpenHighLowClose struct {
	Time   float64
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Vwap   float64
	Volume float64
	Count  float64
}

// RecentTrades holds recent trade data
type RecentTrades struct {
	Price         float64
	Volume        float64
	Time          float64
	BuyOrSell     string
	MarketOrLimit string
	Miscellaneous interface{}
}

// OrderbookBase stores the orderbook price and amount data
type OrderbookBase struct {
	Price  float64
	Amount float64
}

// Orderbook stores the bids and asks orderbook data
type Orderbook struct {
	Bids []OrderbookBase
	Asks []OrderbookBase
}

// Spread holds the spread between trades
type Spread struct {
	Time float64
	Bid  float64
	Ask  float64
}

// TradeBalanceOptions type
type TradeBalanceOptions struct {
	Aclass string
	Asset  string
}

// TradeBalanceInfo type
type TradeBalanceInfo struct {
	EquivalentBalance float64 `json:"eb,string"` // combined balance of all currencies
	TradeBalance      float64 `json:"tb,string"` // combined balance of all equity currencies
	MarginAmount      float64 `json:"m,string"`  // margin amount of open positions
	Net               float64 `json:"n,string"`  // unrealized net profit/loss of open positions
	Equity            float64 `json:"e,string"`  // trade balance + unrealized net profit/loss
	FreeMargin        float64 `json:"mf,string"` // equity - initial margin (maximum margin available to open new positions)
	MarginLevel       float64 `json:"ml,string"` // (equity / initial margin) * 100
}

// OrderInfo type
type OrderInfo struct {
	RefID    string  `json:"refid"`
	UserRef  int32   `json:"userref"`
	Status   string  `json:"status"`
	OpenTm   float64 `json:"opentm"`
	StartTm  float64 `json:"starttm"`
	ExpireTm float64 `json:"expiretm"`
	Descr    struct {
		Pair      string  `json:"pair"`
		Type      string  `json:"type"`
		OrderType string  `json:"ordertype"`
		Price     float64 `json:"price,string"`
		Price2    float64 `json:"price2,string"`
		Leverage  string  `json:"leverage"`
		Order     string  `json:"order"`
		Close     string  `json:"close"`
	} `json:"descr"`
	Vol        float64  `json:"vol,string"`
	VolExec    float64  `json:"vol_exec,string"`
	Cost       float64  `json:"cost,string"`
	Fee        float64  `json:"fee,string"`
	Price      float64  `json:"price,string"`
	StopPrice  float64  `json:"stopprice,string"`
	LimitPrice float64  `json:"limitprice,string"`
	Misc       string   `json:"misc"`
	Oflags     string   `json:"oflags"`
	Trades     []string `json:"trades"`
}

// OpenOrders type
type OpenOrders struct {
	Open  map[string]OrderInfo `json:"open"`
	Count int64                `json:"count"`
}

// ClosedOrders type
type ClosedOrders struct {
	Closed map[string]OrderInfo `json:"closed"`
	Count  int64                `json:"count"`
}

// GetClosedOrdersOptions type
type GetClosedOrdersOptions struct {
	Trades    bool
	UserRef   int32
	Start     string
	End       string
	Ofs       int64
	CloseTime string
}

// OrderInfoOptions type
type OrderInfoOptions struct {
	Trades  bool
	UserRef int32
}

// GetTradesHistoryOptions type
type GetTradesHistoryOptions struct {
	Type   string
	Trades bool
	Start  string
	End    string
	Ofs    int64
}

// TradesHistory type
type TradesHistory struct {
	Trades map[string]TradeInfo `json:"trades"`
	Count  int64                `json:"count"`
}

// TradeInfo type
type TradeInfo struct {
	OrderTxID string   `json:"ordertxid"`
	Pair      string   `json:"pair"`
	Time      float64  `json:"time"`
	Type      string   `json:"type"`
	OrderType string   `json:"ordertype"`
	Price     float64  `json:"price,string"`
	Cost      float64  `json:"cost,string"`
	Fee       float64  `json:"fee,string"`
	Vol       float64  `json:"vol,string"`
	Margin    float64  `json:"margin,string"`
	Misc      string   `json:"misc"`
	PosTxID   string   `json:"postxid"`
	Cprice    float64  `json:"cprice,string"`
	Cfee      float64  `json:"cfee,string"`
	Cvol      float64  `json:"cvol,string"`
	Cmargin   float64  `json:"cmargin,string"`
	Trades    []string `json:"trades"`
	PosStatus string   `json:"posstatus"`
}

// Position holds the opened position
type Position struct {
	Ordertxid  string  `json:"ordertxid"`
	Pair       string  `json:"pair"`
	Time       float64 `json:"time"`
	Type       string  `json:"type"`
	OrderType  string  `json:"ordertype"`
	Cost       float64 `json:"cost,string"`
	Fee        float64 `json:"fee,string"`
	Vol        float64 `json:"vol,string"`
	VolClosed  float64 `json:"vol_closed,string"`
	Margin     float64 `json:"margin,string"`
	Rollovertm int64   `json:"rollovertm,string"`
	Misc       string  `json:"misc"`
	Oflags     string  `json:"oflags"`
	PosStatus  string  `json:"posstatus"`
	Net        string  `json:"net"`
	Terms      string  `json:"terms"`
}

// GetLedgersOptions type
type GetLedgersOptions struct {
	Aclass string
	Asset  string
	Type   string
	Start  string
	End    string
	Ofs    int64
}

// Ledgers type
type Ledgers struct {
	Ledger map[string]LedgerInfo `json:"ledger"`
	Count  int64                 `json:"count"`
}

// LedgerInfo type
type LedgerInfo struct {
	Refid   string  `json:"refid"`
	Time    float64 `json:"time"`
	Type    string  `json:"type"`
	Aclass  string  `json:"aclass"`
	Asset   string  `json:"asset"`
	Amount  float64 `json:"amount,string"`
	Fee     float64 `json:"fee,string"`
	Balance float64 `json:"balance,string"`
}

// TradeVolumeResponse type
type TradeVolumeResponse struct {
	Currency  string                    `json:"currency"`
	Volume    float64                   `json:"volume,string"`
	Fees      map[string]TradeVolumeFee `json:"fees"`
	FeesMaker map[string]TradeVolumeFee `json:"fees_maker"`
}

// TradeVolumeFee type
type TradeVolumeFee struct {
	Fee        float64 `json:"fee,string"`
	MinFee     float64 `json:"minfee,string"`
	MaxFee     float64 `json:"maxfee,string"`
	NextFee    float64 `json:"nextfee,string"`
	NextVolume float64 `json:"nextvolume,string"`
	TierVolume float64 `json:"tiervolume,string"`
}

// AddOrderResponse type
type AddOrderResponse struct {
	Description    OrderDescription `json:"descr"`
	TransactionIds []string         `json:"txid"`
}

// WithdrawInformation Used to check withdrawal fees
type WithdrawInformation struct {
	Method string  `json:"method"`
	Limit  float64 `json:"limit,string"`
	Fee    float64 `json:"fee,string"`
}

// DepositMethods Used to check deposit fees
type DepositMethods struct {
	Method          string      `json:"method"`
	Limit           interface{} `json:"limit"` // If no limit amount, this comes back as boolean
	Fee             float64     `json:"fee,string"`
	AddressSetupFee float64     `json:"address-setup-fee,string"`
}

// OrderDescription represents an orders description
type OrderDescription struct {
	Close string `json:"close"`
	Order string `json:"order"`
}

// AddOrderOptions represents the AddOrder options
type AddOrderOptions struct {
	UserRef        int32
	Oflags         string
	StartTm        string
	ExpireTm       string
	CloseOrderType string
	ClosePrice     float64
	ClosePrice2    float64
	Validate       bool
}

// CancelOrderResponse type
type CancelOrderResponse struct {
	Count   int64       `json:"count"`
	Pending interface{} `json:"pending"`
}

// DepositFees the large list of predefined deposit fees
// Prone to change
var DepositFees = map[currency.Code]float64{
	currency.XTZ: 0.05,
}

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change
var WithdrawalFees = map[currency.Code]float64{
	currency.ZUSD: 5,
	currency.ZEUR: 5,
	currency.USD:  5,
	currency.EUR:  5,
	currency.REP:  0.01,
	currency.XXBT: 0.0005,
	currency.BTC:  0.0005,
	currency.XBT:  0.0005,
	currency.BCH:  0.0001,
	currency.ADA:  0.3,
	currency.DASH: 0.005,
	currency.XDG:  2,
	currency.EOS:  0.05,
	currency.ETH:  0.005,
	currency.ETC:  0.005,
	currency.GNO:  0.005,
	currency.ICN:  0.2,
	currency.LTC:  0.001,
	currency.MLN:  0.003,
	currency.XMR:  0.05,
	currency.QTUM: 0.01,
	currency.XRP:  0.02,
	currency.XLM:  0.00002,
	currency.USDT: 5,
	currency.XTZ:  0.05,
	currency.ZEC:  0.0001,
}

// DepositAddress defines a deposit address
type DepositAddress struct {
	Address    string `json:"address"`
	ExpireTime int64  `json:"expiretm,string"`
	New        bool   `json:"new"`
}

// WithdrawStatusResponse defines a withdrawal status response
type WithdrawStatusResponse struct {
	Method string  `json:"method"`
	Aclass string  `json:"aclass"`
	Asset  string  `json:"asset"`
	Refid  string  `json:"refid"`
	TxID   string  `json:"txid"`
	Info   string  `json:"info"`
	Amount float64 `json:"amount,string"`
	Fee    float64 `json:"fee,string"`
	Time   float64 `json:"time"`
	Status string  `json:"status"`
}

// WebsocketSubscriptionEventRequest handles WS subscription events
type WebsocketSubscriptionEventRequest struct {
	Event        string                    `json:"event"`           // subscribe
	RequestID    int64                     `json:"reqid,omitempty"` // Optional, client originated ID reflected in response message.
	Pairs        []string                  `json:"pair"`            // Array of currency pairs (pair1,pair2,pair3).
	Subscription WebsocketSubscriptionData `json:"subscription,omitempty"`
}

// WebsocketUnsubscribeByChannelIDEventRequest  handles WS unsubscribe events
type WebsocketUnsubscribeByChannelIDEventRequest struct {
	Event     string   `json:"event"`           // unsubscribe
	RequestID int64    `json:"reqid,omitempty"` // Optional, client originated ID reflected in response message.
	Pairs     []string `json:"pair,omitempty"`  // Array of currency pairs (pair1,pair2,pair3).
	ChannelID int64    `json:"channelID,omitempty"`
}

// WebsocketSubscriptionData contains details on WS channel
type WebsocketSubscriptionData struct {
	Name     string `json:"name,omitempty"`     // ticker|ohlc|trade|book|spread|*, * for all (ohlc interval value is 1 if all channels subscribed)
	Interval int64  `json:"interval,omitempty"` // Optional - Time interval associated with ohlc subscription in minutes. Default 1. Valid Interval values: 1|5|15|30|60|240|1440|10080|21600
	Depth    int64  `json:"depth,omitempty"`    // Optional - depth associated with book subscription in number of levels each side, default 10. Valid Options are: 10, 25, 100, 500, 1000
}

// WebsocketEventResponse holds all data response types
type WebsocketEventResponse struct {
	Event        string                            `json:"event"`
	Status       string                            `json:"status"`
	Pair         currency.Pair                     `json:"pair,omitempty"`
	RequestID    int64                             `json:"reqid,omitempty"` // Optional, client originated ID reflected in response message.
	Subscription WebsocketSubscriptionResponseData `json:"subscription,omitempty"`
	WebsocketSubscriptionEventResponse
	WebsocketStatusResponse
	WebsocketErrorResponse
}

// WebsocketSubscriptionEventResponse defines a websocket socket event response
type WebsocketSubscriptionEventResponse struct {
	ChannelID int64 `json:"channelID"`
}

// WebsocketSubscriptionResponseData defines a websocket subscription response
type WebsocketSubscriptionResponseData struct {
	Name string `json:"name"`
}

// WebsocketStatusResponse defines a websocket status response
type WebsocketStatusResponse struct {
	ConnectionID float64 `json:"connectionID"`
	Version      string  `json:"version"`
}

// WebsocketDataResponse defines a websocket data type
type WebsocketDataResponse []interface{}

// WebsocketErrorResponse defines a websocket error response
type WebsocketErrorResponse struct {
	ErrorMessage string `json:"errorMessage"`
}

// WebsocketChannelData Holds relevant data for channels to identify what we're
// doing
type WebsocketChannelData struct {
	Subscription string
	Pair         currency.Pair
	ChannelID    int64
}
