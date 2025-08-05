package bitstamp

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Transaction types
const (
	Deposit = iota
	Withdrawal
	MarketTrade
	SubAccountTransfer = 14
)

// Order side type
const (
	BuyOrder = iota
	SellOrder
)

// Ticker holds ticker information
type Ticker struct {
	Last            float64    `json:"last,string"`
	High            float64    `json:"high,string"`
	Low             float64    `json:"low,string"`
	Vwap            float64    `json:"vwap,string"`
	Volume          float64    `json:"volume,string"`
	Bid             float64    `json:"bid,string"`
	Ask             float64    `json:"ask,string"`
	Timestamp       types.Time `json:"timestamp"`
	Open            float64    `json:"open,string"`
	Open24          float64    `json:"open_24,string"`
	Side            orderSide  `json:"side,string"`
	PercentChange24 float64    `json:"percent_change_24,string"`
}

// OrderbookBase holds singular price information
type OrderbookBase struct {
	Price  float64
	Amount float64
}

// Orderbook holds orderbook information
type Orderbook struct {
	Timestamp time.Time
	Bids      []OrderbookBase
	Asks      []OrderbookBase
}

// TradingPair holds trading pair information
type TradingPair struct {
	Name            string `json:"name"`
	URLSymbol       string `json:"url_symbol"`
	BaseDecimals    int    `json:"base_decimals"`
	CounterDecimals int    `json:"counter_decimals"`
	MinimumOrder    float64
	Trading         string `json:"trading"`
	Description     string `json:"description"`
}

// Transactions holds transaction data
type Transactions struct {
	Date    types.Time `json:"date"`
	TradeID int64      `json:"tid,string"`
	Price   float64    `json:"price,string"`
	Type    int        `json:"type,string"`
	Amount  float64    `json:"amount,string"`
}

// EURUSDConversionRate holds buy sell conversion rate information
type EURUSDConversionRate struct {
	Buy  float64 `json:"buy,string"`
	Sell float64 `json:"sell,string"`
}

// TradingFees holds trading fee information
type TradingFees struct {
	Symbol string         `json:"currency_pair"`
	Fees   MakerTakerFees `json:"fees"`
}

// MakerTakerFees holds maker and taker fee information
type MakerTakerFees struct {
	Maker float64 `json:"maker,string"`
	Taker float64 `json:"taker,string"`
}

// Balance stores the balance info
type Balance struct {
	Available     float64
	Balance       float64
	Reserved      float64
	WithdrawalFee float64
}

// Balances holds full balance information with the supplied APIKEYS
type Balances map[string]Balance

// UserTransactions holds user transaction information
type UserTransactions struct {
	Date          types.DateTime `json:"datetime"`
	TransactionID int64          `json:"id"`
	Type          int64          `json:"type,string"`
	USD           types.Number   `json:"usd"`
	EUR           types.Number   `json:"eur"`
	BTC           types.Number   `json:"btc"`
	XRP           types.Number   `json:"xrp"`
	BTCUSD        types.Number   `json:"btc_usd"`
	Fee           types.Number   `json:"fee"`
	OrderID       int64          `json:"order_id"`
}

// Order holds current open order data
type Order struct {
	ID             int64          `json:"id,string"`
	DateTime       types.DateTime `json:"datetime"`
	Type           int64          `json:"type,string"`
	Price          float64        `json:"price,string"`
	Amount         float64        `json:"amount,string"`
	AmountAtCreate float64        `json:"amount_at_create,string"`
	Currency       string         `json:"currency_pair"`
	LimitPrice     float64        `json:"limit_price,string"`
	ClientOrderID  string         `json:"client_order_id"`
	Market         string         `json:"market"`
}

// OrderStatus holds order status information
type OrderStatus struct {
	AmountRemaining float64        `json:"amount_remaining,string"`
	Type            int64          `json:"type"`
	ID              string         `json:"id"`
	DateTime        types.DateTime `json:"datetime"`
	Status          string         `json:"status"`
	ClientOrderID   string         `json:"client_order_id"`
	Market          string         `json:"market"`
	Transactions    []struct {
		TradeID      int64          `json:"tid"`
		FromCurrency float64        `json:"{from_currency},string"`
		ToCurrency   float64        `json:"{to_currency},string"`
		Price        float64        `json:"price,string"`
		Fee          float64        `json:"fee,string"`
		DateTime     types.DateTime `json:"datetime"`
		Type         int64          `json:"type"`
	}
}

// CancelOrder holds the order cancellation info
type CancelOrder struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
	Type   int     `json:"type"`
	ID     int64   `json:"id"`
}

// DepositAddress holds the deposit info
type DepositAddress struct {
	Address        string `json:"address"`
	DestinationTag int64  `json:"destination_tag"`
}

// WithdrawalRequests holds request information on withdrawals
type WithdrawalRequests struct {
	OrderID       int64          `json:"id"`
	Date          types.DateTime `json:"datetime"`
	Type          int64          `json:"type"`
	Amount        float64        `json:"amount,string"`
	Status        int64          `json:"status"`
	Currency      currency.Code  `json:"currency"`
	Address       string         `json:"address"`
	TransactionID string         `json:"transaction_id"`
	Network       string         `json:"network"`
	TxID          int64          `json:"txid"`
}

// CryptoWithdrawalResponse response from a crypto withdrawal request
type CryptoWithdrawalResponse struct {
	ID int64 `json:"withdrawal_id"`
}

// FIATWithdrawalResponse response from a fiat withdrawal request
type FIATWithdrawalResponse struct {
	ID int64 `json:"withdrawal_id"`
}

// UnconfirmedBTCTransactions holds address information about unconfirmed
// transactions
type UnconfirmedBTCTransactions struct {
	Address        string `json:"address"`
	DestinationTag int    `json:"destination_tag"`
	MemoID         string `json:"memo_id"`
}

// CaptureError is used to capture unmarshalled errors
type CaptureError struct {
	Status any `json:"status"`
	Reason any `json:"reason"`
	Code   any `json:"code"`
	Error  any `json:"error"`
}

const (
	sepaWithdrawal          string = "sepa"
	internationalWithdrawal string = "international"
	errStr                  string = "error"
)

type websocketEventRequest struct {
	Event string        `json:"event"`
	Data  websocketData `json:"data"`
}

type websocketData struct {
	Channel string `json:"channel"`
	Auth    string `json:"auth,omitempty"`
}

type websocketResponse struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
}

type websocketTradeResponse struct {
	websocketResponse
	Data websocketTradeData `json:"data"`
}

type websocketTradeData struct {
	Microtimestamp string     `json:"microtimestamp"`
	Amount         float64    `json:"amount"`
	BuyOrderID     int64      `json:"buy_order_id"`
	SellOrderID    int64      `json:"sell_order_id"`
	AmountStr      string     `json:"amount_str"`
	PriceStr       string     `json:"price_str"`
	Timestamp      types.Time `json:"timestamp"`
	Price          float64    `json:"price"`
	Type           int        `json:"type"`
	ID             int64      `json:"id"`
}

// WebsocketAuthResponse holds the auth token for subscribing to auth channels
type WebsocketAuthResponse struct {
	Token     string `json:"token"`
	UserID    int64  `json:"user_id"`
	ValidSecs int64  `json:"valid_sec"`
}

type websocketOrderBookResponse struct {
	websocketResponse
	Data websocketOrderBook `json:"data"`
}

type websocketOrderBook struct {
	Asks           [][2]types.Number `json:"asks"`
	Bids           [][2]types.Number `json:"bids"`
	Timestamp      types.Time        `json:"timestamp"`
	Microtimestamp types.Time        `json:"microtimestamp"`
}

// OHLCResponse holds returned candle data
type OHLCResponse struct {
	Data struct {
		Pair  string `json:"pair"`
		OHLCV []struct {
			Timestamp types.Time `json:"timestamp"`
			Open      float64    `json:"open,string"`
			High      float64    `json:"high,string"`
			Low       float64    `json:"low,string"`
			Close     float64    `json:"close,string"`
			Volume    float64    `json:"volume,string"`
		} `json:"ohlc"`
	} `json:"data"`
}

type websocketOrderResponse struct {
	websocketResponse
	Order websocketOrderData `json:"data"`
}

type websocketOrderData struct {
	ID              int64      `json:"id"`
	IDStr           string     `json:"id_str"`
	ClientOrderID   string     `json:"client_order_id"`
	RemainingAmount float64    `json:"amount"`
	ExecutedAmount  float64    `json:"amount_traded,string"` // Not Cumulative; Partial fill amount
	Amount          float64    `json:"amount_at_create,string"`
	Price           float64    `json:"price"`
	Side            orderSide  `json:"order_type"`
	Datetime        types.Time `json:"datetime"`
	Microtimestamp  types.Time `json:"microtimestamp"`
}
