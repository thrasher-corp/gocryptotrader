package kraken

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	krakenAPIVersion       = "0"
	krakenServerTime       = "Time"
	krakenAssets           = "Assets"
	krakenAssetPairs       = "AssetPairs?"
	krakenTicker           = "Ticker"
	krakenOHLC             = "OHLC"
	krakenDepth            = "Depth"
	krakenTrades           = "Trades"
	krakenSpread           = "Spread"
	krakenBalance          = "BalanceEx"
	krakenTradeBalance     = "TradeBalance"
	krakenOpenOrders       = "OpenOrders"
	krakenClosedOrders     = "ClosedOrders"
	krakenQueryOrders      = "QueryOrders"
	krakenTradeHistory     = "TradesHistory"
	krakenQueryTrades      = "QueryTrades"
	krakenOpenPositions    = "OpenPositions"
	krakenLedgers          = "Ledgers"
	krakenQueryLedgers     = "QueryLedgers"
	krakenTradeVolume      = "TradeVolume"
	krakenOrderCancel      = "CancelOrder"
	krakenOrderPlace       = "AddOrder"
	krakenWithdrawInfo     = "WithdrawInfo"
	krakenWithdraw         = "Withdraw"
	krakenDepositMethods   = "DepositMethods"
	krakenDepositAddresses = "DepositAddresses"
	krakenWithdrawStatus   = "WithdrawStatus"
	krakenWithdrawCancel   = "WithdrawCancel"
	krakenWebsocketToken   = "GetWebSocketsToken"

	// Futures
	futuresTickers      = "/api/v3/tickers"
	futuresOrderbook    = "/api/v3/orderbook"
	futuresInstruments  = "/api/v3/instruments"
	futuresTradeHistory = "/api/v3/history"
	futuresCandles      = "charts/v1/"
	futuresPublicTrades = "history/v2/market/"

	futuresSendOrder         = "/api/v3/sendorder"
	futuresCancelOrder       = "/api/v3/cancelorder"
	futuresOrderFills        = "/api/v3/fills"
	futuresTransfer          = "/api/v3/transfer"
	futuresOpenPositions     = "/api/v3/openpositions"
	futuresBatchOrder        = "/api/v3/batchorder"
	futuresNotifications     = "/api/v3/notifications"
	futuresAccountData       = "/api/v3/accounts"
	futuresCancelAllOrders   = "/api/v3/cancelallorders"
	futuresCancelOrdersAfter = "/api/v3/cancelallordersafter"
	futuresOpenOrders        = "/api/v3/openorders"
	futuresRecentOrders      = "/api/v3/recentorders"
	futuresWithdraw          = "/api/v3/withdrawal"
	futuresTransfers         = "/api/v3/transfers"
	futuresEditOrder         = "/api/v3/editorder"

	// Rate limit consts
	krakenRateInterval = time.Second
	krakenRequestRate  = 1

	// Status consts
	statusOpen = "open"
)

var (
	assetTranslator     assetTranslatorStore
	errBadChannelSuffix = errors.New("bad websocket channel suffix")
)

// GenericResponse stores general response data for functions that only return success
type GenericResponse struct {
	Timestamp string `json:"timestamp"`
	Result    string `json:"result"`
}

type genericFuturesResponse struct {
	Result     string    `json:"result"`
	ServerTime time.Time `json:"serverTime"`
	Error      string    `json:"error"`
	Errors     []string  `json:"errors"`
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
	Wsname            string      `json:"wsname"`
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
	OrderMinimum      float64     `json:"ordermin,string"`
	TickSize          float64     `json:"tick_size,string"`
	Status            string      `json:"status"`
}

// Ticker is a standard ticker type
type Ticker struct {
	Ask                        float64
	AskSize                    float64
	Bid                        float64
	BidSize                    float64
	Last                       float64
	Volume                     float64
	VolumeWeightedAveragePrice float64
	Trades                     int64
	Low                        float64
	High                       float64
	Open                       float64
}

// Tickers stores a map of tickers
type Tickers map[string]Ticker

// TickerResponse holds ticker information before its put into the Ticker struct
type TickerResponse struct {
	Ask                        [3]types.Number `json:"a"`
	Bid                        [3]types.Number `json:"b"`
	Last                       [2]types.Number `json:"c"`
	Volume                     [2]types.Number `json:"v"`
	VolumeWeightedAveragePrice [2]types.Number `json:"p"`
	Trades                     [2]int64        `json:"t"`
	Low                        [2]types.Number `json:"l"`
	High                       [2]types.Number `json:"h"`
	Open                       types.Number    `json:"o"`
}

// OpenHighLowClose contains ticker event information
type OpenHighLowClose struct {
	Time                       time.Time
	Open                       float64
	High                       float64
	Low                        float64
	Close                      float64
	VolumeWeightedAveragePrice float64
	Volume                     float64
	Count                      float64
}

// RecentTradesResponse holds recent trade data
type RecentTradesResponse struct {
	Trades map[string][]RecentTradeResponseItem
	Last   types.Time
}

// UnmarshalJSON unmarshals the recent trades response
func (r *RecentTradesResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.Trades = make(map[string][]RecentTradeResponseItem)
	for key, raw := range raw {
		if key == "last" {
			if err := json.Unmarshal(raw, &r.Last); err != nil {
				return err
			}
		} else {
			var trades []RecentTradeResponseItem
			if err := json.Unmarshal(raw, &trades); err != nil {
				return err
			}
			r.Trades[key] = trades
		}
	}
	return nil
}

// RecentTradeResponseItem holds a single recent trade response item
type RecentTradeResponseItem struct {
	Price         types.Number
	Volume        types.Number
	Time          types.Time
	BuyOrSell     string
	MarketOrLimit string
	Miscellaneous any
	TradeID       types.Number
}

// UnmarshalJSON unmarshals the recent trade response item
func (r *RecentTradeResponseItem) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&r.Price, &r.Volume, &r.Time, &r.BuyOrSell, &r.MarketOrLimit, &r.Miscellaneous, &r.TradeID})
}

// OrderbookBase stores the orderbook price and amount data
type OrderbookBase struct {
	Price     types.Number
	Amount    types.Number
	Timestamp time.Time
}

// Orderbook stores the bids and asks orderbook data
type Orderbook struct {
	Bids []OrderbookBase
	Asks []OrderbookBase
}

// SpreadItem holds the spread between trades
type SpreadItem struct {
	Time types.Time
	Bid  types.Number
	Ask  types.Number
}

// UnmarshalJSON unmarshals the spread item
func (s *SpreadItem) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[3]any{&s.Time, &s.Bid, &s.Ask})
}

// SpreadResponse holds the spread response data
type SpreadResponse struct {
	Spreads map[string][]SpreadItem
	Last    types.Time
}

// UnmarshalJSON unmarshals the spread response
func (s *SpreadResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	s.Spreads = make(map[string][]SpreadItem)
	for key, raw := range raw {
		if key == "last" {
			if err := json.Unmarshal(raw, &s.Last); err != nil {
				return err
			}
		} else {
			var spreads []SpreadItem
			if err := json.Unmarshal(raw, &spreads); err != nil {
				return err
			}
			s.Spreads[key] = spreads
		}
	}
	return nil
}

// Balance represents account asset balances
type Balance struct {
	Total float64 `json:"balance,string"`
	Hold  float64 `json:"hold_trade,string"`
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
	RefID       string     `json:"refid"`
	UserRef     int32      `json:"userref"`
	Status      string     `json:"status"`
	OpenTime    types.Time `json:"opentm"`
	CloseTime   types.Time `json:"closetm"`
	StartTime   types.Time `json:"starttm"`
	ExpireTime  types.Time `json:"expiretm"`
	Description struct {
		Pair      string  `json:"pair"`
		Type      string  `json:"type"`
		OrderType string  `json:"ordertype"`
		Price     float64 `json:"price,string"`
		Price2    float64 `json:"price2,string"`
		Leverage  string  `json:"leverage"`
		Order     string  `json:"order"`
		Close     string  `json:"close"`
	} `json:"descr"`
	Volume         float64  `json:"vol,string"`
	VolumeExecuted float64  `json:"vol_exec,string"`
	Cost           float64  `json:"cost,string"`
	Fee            float64  `json:"fee,string"`
	Price          float64  `json:"price,string"`
	StopPrice      float64  `json:"stopprice,string"`
	LimitPrice     float64  `json:"limitprice,string"`
	Misc           string   `json:"misc"`
	OrderFlags     string   `json:"oflags"`
	Trades         []string `json:"trades"`
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
	OrderTxID                  string     `json:"ordertxid"`
	Pair                       string     `json:"pair"`
	Time                       types.Time `json:"time"`
	Type                       string     `json:"type"`
	OrderType                  string     `json:"ordertype"`
	Price                      float64    `json:"price,string"`
	Cost                       float64    `json:"cost,string"`
	Fee                        float64    `json:"fee,string"`
	Volume                     float64    `json:"vol,string"`
	Margin                     float64    `json:"margin,string"`
	Misc                       string     `json:"misc"`
	PosTxID                    string     `json:"postxid"`
	ClosedPositionAveragePrice float64    `json:"cprice,string"`
	ClosedPositionFee          float64    `json:"cfee,string"`
	ClosedPositionVolume       float64    `json:"cvol,string"`
	ClosedPositionMargin       float64    `json:"cmargin,string"`
	Trades                     []string   `json:"trades"`
	PosStatus                  string     `json:"posstatus"`
}

// Position holds the opened position
type Position struct {
	Ordertxid      string     `json:"ordertxid"`
	Pair           string     `json:"pair"`
	Time           types.Time `json:"time"`
	Type           string     `json:"type"`
	OrderType      string     `json:"ordertype"`
	Cost           float64    `json:"cost,string"`
	Fee            float64    `json:"fee,string"`
	Volume         float64    `json:"vol,string"`
	VolumeClosed   float64    `json:"vol_closed,string"`
	Margin         float64    `json:"margin,string"`
	RolloverTime   int64      `json:"rollovertm,string"`
	Misc           string     `json:"misc"`
	OrderFlags     string     `json:"oflags"`
	PositionStatus string     `json:"posstatus"`
	Net            string     `json:"net"`
	Terms          string     `json:"terms"`
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
	Refid   string     `json:"refid"`
	Time    types.Time `json:"time"`
	Type    string     `json:"type"`
	Aclass  string     `json:"aclass"`
	Asset   string     `json:"asset"`
	Amount  float64    `json:"amount,string"`
	Fee     float64    `json:"fee,string"`
	Balance float64    `json:"balance,string"`
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
	TransactionIDs []string         `json:"txid"`
}

// WithdrawInformation Used to check withdrawal fees
type WithdrawInformation struct {
	Method string  `json:"method"`
	Limit  float64 `json:"limit,string"`
	Fee    float64 `json:"fee,string"`
}

// DepositMethods Used to check deposit fees
type DepositMethods struct {
	Method          string  `json:"method"`
	Limit           any     `json:"limit"` // If no limit amount, this comes back as boolean
	Fee             float64 `json:"fee,string"`
	AddressSetupFee float64 `json:"address-setup-fee,string"`
}

// OrderDescription represents an orders description
type OrderDescription struct {
	Close string `json:"close"`
	Order string `json:"order"`
}

// AddOrderOptions represents the AddOrder options
type AddOrderOptions struct {
	UserRef        int32
	OrderFlags     string
	StartTm        string
	ExpireTm       string
	CloseOrderType string
	ClosePrice     float64
	ClosePrice2    float64
	Validate       bool
	TimeInForce    string
}

// CancelOrderResponse type
type CancelOrderResponse struct {
	Count   int64 `json:"count"`
	Pending any   `json:"pending"`
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
	ExpireTime any    `json:"expiretm"` // this is an int when new is specified
	Tag        string `json:"tag"`
	New        bool   `json:"new"`
}

// WithdrawStatusResponse defines a withdrawal status response
type WithdrawStatusResponse struct {
	Method string     `json:"method"`
	Aclass string     `json:"aclass"`
	Asset  string     `json:"asset"`
	Refid  string     `json:"refid"`
	TxID   string     `json:"txid"`
	Info   string     `json:"info"`
	Amount float64    `json:"amount,string"`
	Fee    float64    `json:"fee,string"`
	Time   types.Time `json:"time"`
	Status string     `json:"status"`
}

// WebsocketSubRequest contains request data for Subscribe/Unsubscribe to channels
type WebsocketSubRequest struct {
	Event        string                    `json:"event"`
	RequestID    int64                     `json:"reqid,omitempty"`
	Pairs        []string                  `json:"pair,omitempty"`
	Subscription WebsocketSubscriptionData `json:"subscription"`
}

// WebsocketSubscriptionData contains details on WS channel
type WebsocketSubscriptionData struct {
	Name     string `json:"name,omitempty"`     // ticker|ohlc|trade|book|spread|*, * for all (ohlc interval value is 1 if all channels subscribed)
	Interval int    `json:"interval,omitempty"` // Optional - Timeframe for candles subscription in minutes; default 1. Valid: 1|5|15|30|60|240|1440|10080|21600
	Depth    int    `json:"depth,omitempty"`    // Optional - Depth associated with orderbook; default 10. Valid: 10|25|100|500|1000
	Token    string `json:"token,omitempty"`    // Optional - Token for authenticated channels
}

// WebsocketEventResponse holds all data response types
type WebsocketEventResponse struct {
	Event        string                            `json:"event"`
	Status       string                            `json:"status"`
	Pair         currency.Pair                     `json:"pair"`
	RequestID    int64                             `json:"reqid,omitempty"`
	Subscription WebsocketSubscriptionResponseData `json:"subscription"`
	ChannelName  string                            `json:"channelName,omitempty"`
	WebsocketSubscriptionEventResponse
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

// WebsocketErrorResponse defines a websocket error response
type WebsocketErrorResponse struct {
	ErrorMessage string `json:"errorMessage"`
}

// WsTokenResponse holds the WS auth token
type WsTokenResponse struct {
	Expires int64  `json:"expires"`
	Token   string `json:"token"`
}

type wsSystemStatus struct {
	ConnectionID float64 `json:"connectionID"`
	Event        string  `json:"event"`
	Status       string  `json:"status"`
	Version      string  `json:"version"`
}

// WsOpenOrder contains all open order data from ws feed
type WsOpenOrder struct {
	UserReferenceID int64      `json:"userref"`
	ExpireTime      types.Time `json:"expiretm"`
	LastUpdated     types.Time `json:"lastupdated"`
	OpenTime        types.Time `json:"opentm"`
	StartTime       types.Time `json:"starttm"`
	Fee             float64    `json:"fee,string"`
	LimitPrice      float64    `json:"limitprice,string"`
	StopPrice       float64    `json:"stopprice,string"`
	Volume          float64    `json:"vol,string"`
	ExecutedVolume  float64    `json:"vol_exec,string"`
	Cost            float64    `json:"cost,string"`
	AveragePrice    float64    `json:"avg_price,string"`
	Misc            string     `json:"misc"`
	OFlags          string     `json:"oflags"`
	RefID           string     `json:"refid"`
	Status          string     `json:"status"`
	Description     struct {
		Close     string  `json:"close"`
		Price     float64 `json:"price,string"`
		Price2    float64 `json:"price2,string"`
		Leverage  float64 `json:"leverage,string"`
		Order     string  `json:"order"`
		OrderType string  `json:"ordertype"`
		Pair      string  `json:"pair"`
		Type      string  `json:"type"`
	} `json:"descr"`
}

// WsOwnTrade ws auth owntrade data
type WsOwnTrade struct {
	Cost               float64    `json:"cost,string"`
	Fee                float64    `json:"fee,string"`
	Margin             float64    `json:"margin,string"`
	OrderTransactionID string     `json:"ordertxid"`
	OrderType          string     `json:"ordertype"`
	Pair               string     `json:"pair"`
	PostTransactionID  string     `json:"postxid"`
	Price              float64    `json:"price,string"`
	Time               types.Time `json:"time"`
	Type               string     `json:"type"`
	Vol                float64    `json:"vol,string"`
}

// WsOpenOrders ws auth open order data
type WsOpenOrders struct {
	Cost           float64                `json:"cost,string"`
	Description    WsOpenOrderDescription `json:"descr"`
	ExpireTime     types.Time             `json:"expiretm"`
	Fee            float64                `json:"fee,string"`
	LimitPrice     float64                `json:"limitprice,string"`
	Misc           string                 `json:"misc"`
	OFlags         string                 `json:"oflags"`
	OpenTime       types.Time             `json:"opentm"`
	Price          float64                `json:"price,string"`
	RefID          string                 `json:"refid"`
	StartTime      types.Time             `json:"starttm"`
	Status         string                 `json:"status"`
	StopPrice      float64                `json:"stopprice,string"`
	UserReference  float64                `json:"userref"`
	Volume         float64                `json:"vol,string"`
	ExecutedVolume float64                `json:"vol_exec,string"`
}

// WsOpenOrderDescription additional data for WsOpenOrders
type WsOpenOrderDescription struct {
	Close     string  `json:"close"`
	Leverage  string  `json:"leverage"`
	Order     string  `json:"order"`
	OrderType string  `json:"ordertype"`
	Pair      string  `json:"pair"`
	Price     float64 `json:"price,string"`
	Price2    float64 `json:"price2,string"`
	Type      string  `json:"type"`
}

// WsAddOrderRequest request type for ws adding order
type WsAddOrderRequest struct {
	Event           string  `json:"event"`
	Token           string  `json:"token"`
	RequestID       int64   `json:"reqid,omitempty"` // Optional, client originated ID reflected in response message.
	OrderType       string  `json:"ordertype"`
	OrderSide       string  `json:"type"`
	Pair            string  `json:"pair"`
	Price           float64 `json:"price,string,omitempty"`  // optional
	Price2          float64 `json:"price2,string,omitempty"` // optional
	Volume          float64 `json:"volume,string,omitempty"`
	Leverage        float64 `json:"leverage,omitempty"`         // optional
	OFlags          string  `json:"oflags,omitempty"`           // optional
	StartTime       string  `json:"starttm,omitempty"`          // optional
	ExpireTime      string  `json:"expiretm,omitempty"`         // optional
	UserReferenceID string  `json:"userref,omitempty"`          // optional
	Validate        string  `json:"validate,omitempty"`         // optional
	CloseOrderType  string  `json:"close[ordertype],omitempty"` // optional
	ClosePrice      float64 `json:"close[price],omitempty"`     // optional
	ClosePrice2     float64 `json:"close[price2],omitempty"`    // optional
	TimeInForce     string  `json:"timeinforce,omitempty"`      // optional
}

// WsAddOrderResponse response data for ws order
type WsAddOrderResponse struct {
	Event         string `json:"event"`
	RequestID     int64  `json:"reqid"`
	Status        string `json:"status"`
	TransactionID string `json:"txid"`
	Description   string `json:"descr"`
	ErrorMessage  string `json:"errorMessage"`
}

// WsCancelOrderRequest request for ws cancel order
type WsCancelOrderRequest struct {
	Event          string   `json:"event"`
	Token          string   `json:"token"`
	TransactionIDs []string `json:"txid,omitempty"`
	RequestID      int64    `json:"reqid,omitempty"` // Optional, client originated ID reflected in response message.
}

// WsCancelOrderResponse response data for ws cancel order and ws cancel all orders
type WsCancelOrderResponse struct {
	Event        string `json:"event"`
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage"`
	RequestID    int64  `json:"reqid"`
	Count        int64  `json:"count"`
}

// OrderVars stores side, status and type for any order/trade
type OrderVars struct {
	Side      order.Side
	Status    order.Status
	OrderType order.Type
	Fee       float64
}

type genericRESTResponse struct {
	Error  errorResponse `json:"error"`
	Result any           `json:"result"`
}

type errorResponse struct {
	warnings []string
	errors   error
}

func (e *errorResponse) UnmarshalJSON(data []byte) error {
	var errInterface any
	if err := json.Unmarshal(data, &errInterface); err != nil {
		return err
	}

	switch d := errInterface.(type) {
	case string:
		if d[0] == 'E' {
			e.errors = common.AppendError(e.errors, errors.New(d))
		} else {
			e.warnings = append(e.warnings, d)
		}
	case []any:
		for x := range d {
			errStr, ok := d[x].(string)
			if !ok {
				return fmt.Errorf("unable to convert %v to string", d[x])
			}
			if errStr[0] == 'E' {
				e.errors = common.AppendError(e.errors, errors.New(errStr))
			} else {
				e.warnings = append(e.warnings, errStr)
			}
		}
	default:
		return fmt.Errorf("unhandled error response type %T", errInterface)
	}
	return nil
}

// Errors returns one or many errors as an error
func (e errorResponse) Errors() error {
	return e.errors
}

// Warnings returns a string of warnings
func (e errorResponse) Warnings() string {
	return strings.Join(e.warnings, ", ")
}

type wsTicker struct {
	Ask                        [3]types.Number `json:"a"`
	Bid                        [3]types.Number `json:"b"`
	Last                       [2]types.Number `json:"c"`
	Volume                     [2]types.Number `json:"v"`
	VolumeWeightedAveragePrice [2]types.Number `json:"p"`
	Trades                     [2]int64        `json:"t"`
	Low                        [2]types.Number `json:"l"`
	High                       [2]types.Number `json:"h"`
	Open                       [2]types.Number `json:"o"`
}

type wsSpread struct {
	Bid       types.Number
	Ask       types.Number
	Time      types.Time
	BidVolume types.Number
	AskVolume types.Number
}

func (w *wsSpread) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[5]any{&w.Bid, &w.Ask, &w.Time, &w.BidVolume, &w.AskVolume})
}

type wsTrades struct {
	Price     types.Number
	Volume    types.Number
	Time      types.Time
	Side      string
	OrderType string
	Misc      string
}

func (w *wsTrades) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[6]any{&w.Price, &w.Volume, &w.Time, &w.Side, &w.OrderType, &w.Misc})
}

type wsCandle struct {
	LastUpdateTime types.Time
	EndTime        types.Time
	Open           types.Number
	High           types.Number
	Low            types.Number
	Close          types.Number
	VWAP           types.Number
	Volume         types.Number
	Count          int64
}

func (w *wsCandle) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[9]any{&w.LastUpdateTime, &w.EndTime, &w.Open, &w.High, &w.Low, &w.Close, &w.VWAP, &w.Volume, &w.Count})
}

type wsSnapshot struct {
	Asks []wsOrderbookItem `json:"as"`
	Bids []wsOrderbookItem `json:"bs"`
}

type wsUpdate struct {
	Asks     []wsOrderbookItem `json:"a"`
	Bids     []wsOrderbookItem `json:"b"`
	Checksum uint32            `json:"c,string"`
}

type wsOrderbookItem struct {
	Price     float64
	PriceRaw  string
	Amount    float64
	AmountRaw string
	Time      types.Time
}

func (ws *wsOrderbookItem) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &[3]any{&ws.PriceRaw, &ws.AmountRaw, &ws.Time})
	if err != nil {
		return err
	}
	ws.Price, err = strconv.ParseFloat(ws.PriceRaw, 64)
	if err != nil {
		return fmt.Errorf("error parsing price: %w", err)
	}
	ws.Amount, err = strconv.ParseFloat(ws.AmountRaw, 64)
	if err != nil {
		return fmt.Errorf("error parsing amount: %w", err)
	}
	return nil
}
