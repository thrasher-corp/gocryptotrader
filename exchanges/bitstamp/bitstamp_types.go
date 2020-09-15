package bitstamp

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
	Last      float64 `json:"last,string"`
	High      float64 `json:"high,string"`
	Low       float64 `json:"low,string"`
	Vwap      float64 `json:"vwap,string"`
	Volume    float64 `json:"volume,string"`
	Bid       float64 `json:"bid,string"`
	Ask       float64 `json:"ask,string"`
	Timestamp int64   `json:"timestamp,string"`
	Open      float64 `json:"open,string"`
}

// OrderbookBase holds singular price information
type OrderbookBase struct {
	Price  float64
	Amount float64
}

// Orderbook holds orderbook information
type Orderbook struct {
	Timestamp int64 `json:"timestamp,string"`
	Bids      []OrderbookBase
	Asks      []OrderbookBase
}

// TradingPair holds trading pair information
type TradingPair struct {
	Name            string `json:"name"`
	URLSymbol       string `json:"url_symbol"`
	BaseDecimals    int    `json:"base_decimals"`
	CounterDecimals int    `json:"counter_decimals"`
	MinimumOrder    string `json:"minimum_order"`
	Trading         string `json:"trading"`
	Description     string `json:"description"`
}

// Transactions holds transaction data
type Transactions struct {
	Date    int64   `json:"date,string"`
	TradeID int64   `json:"tid,string"`
	Price   float64 `json:"price,string"`
	Type    int     `json:"type,string"`
	Amount  float64 `json:"amount,string"`
}

// EURUSDConversionRate holds buy sell conversion rate information
type EURUSDConversionRate struct {
	Buy  float64 `json:"buy,string"`
	Sell float64 `json:"sell,string"`
}

// Balance stores the balance info
type Balance struct {
	Available     float64
	Balance       float64
	Reserved      float64
	WithdrawalFee float64
	BTCFee        float64 // for cryptocurrency pairs
	USDFee        float64
	EURFee        float64
}

// Balances holds full balance information with the supplied APIKEYS
type Balances map[string]Balance

// UserTransactions holds user transaction information
type UserTransactions struct {
	Date          string  `json:"datetime"`
	TransactionID int64   `json:"id"`
	Type          int     `json:"type,string"`
	USD           float64 `json:"usd"`
	EUR           float64 `json:"eur"`
	BTC           float64 `json:"btc"`
	XRP           float64 `json:"xrp"`
	BTCUSD        float64 `json:"btc_usd"`
	Fee           float64 `json:"fee,string"`
	OrderID       int64   `json:"order_id"`
}

// Order holds current open order data
type Order struct {
	ID       int64   `json:"id,string"`
	DateTime string  `json:"datetime"`
	Type     int     `json:"type,string"`
	Price    float64 `json:"price,string"`
	Amount   float64 `json:"amount,string"`
	Currency string  `json:"currency_pair"`
}

// OrderStatus holds order status information
type OrderStatus struct {
	Price        float64 `json:"price,string"`
	Amount       float64 `json:"amount,string"`
	Type         int     `json:"type"`
	ID           int64   `json:"id,string"`
	DateTime     string  `json:"datetime"`
	Status       string
	Transactions []struct {
		TradeID int64   `json:"tid"`
		USD     float64 `json:"usd,string"`
		Price   float64 `json:"price,string"`
		Fee     float64 `json:"fee,string"`
		BTC     float64 `json:"btc,string"`
	}
}

// CancelOrder holds the order cancellation info
type CancelOrder struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
	Type   int     `json:"type"`
	ID     int64   `json:"id"`
}

// WithdrawalRequests holds request information on withdrawals
type WithdrawalRequests struct {
	OrderID       int64   `json:"id"`
	Date          string  `json:"datetime"`
	Type          int     `json:"type"`
	Amount        float64 `json:"amount,string"`
	Status        int     `json:"status"`
	Data          interface{}
	Address       string `json:"address"`        // Bitcoin withdrawals only
	TransactionID string `json:"transaction_id"` // Bitcoin withdrawals only
}

// CryptoWithdrawalResponse response from a crypto withdrawal request
type CryptoWithdrawalResponse struct {
	ID    string              `json:"id"`
	Error map[string][]string `json:"error"`
}

// FIATWithdrawalResponse response from a fiat withdrawal request
type FIATWithdrawalResponse struct {
	ID     string              `json:"id"`
	Status string              `json:"status"`
	Reason map[string][]string `json:"reason"`
}

// UnconfirmedBTCTransactions holds address information about unconfirmed
// transactions
type UnconfirmedBTCTransactions struct {
	Amount        float64 `json:"amount,string"`
	Address       string  `json:"address"`
	Confirmations int     `json:"confirmations"`
}

// CaptureError is used to capture unmarshalled errors
type CaptureError struct {
	Status interface{} `json:"status"`
	Reason interface{} `json:"reason"`
	Code   interface{} `json:"code"`
	Error  interface{} `json:"error"`
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
	Microtimestamp string  `json:"microtimestamp"`
	Amount         float64 `json:"amount"`
	BuyOrderID     int64   `json:"buy_order_id"`
	SellOrderID    int64   `json:"sell_order_id"`
	AmountStr      string  `json:"amount_str"`
	PriceStr       string  `json:"price_str"`
	Timestamp      int64   `json:"timestamp,string"`
	Price          float64 `json:"price"`
	Type           int     `json:"type"`
	ID             int64   `json:"id"`
}

type websocketOrderBookResponse struct {
	websocketResponse
	Data websocketOrderBook `json:"data"`
}

type websocketOrderBook struct {
	Asks           [][]string `json:"asks"`
	Bids           [][]string `json:"bids"`
	Timestamp      int64      `json:"timestamp,string"`
	Microtimestamp string     `json:"microtimestamp"`
}

// OHLCResponse holds returned candle data
type OHLCResponse struct {
	Data struct {
		Pair  string `json:"pair"`
		OHLCV []struct {
			Timestamp int64   `json:"timestamp,string"`
			Open      float64 `json:"open,string"`
			High      float64 `json:"high,string"`
			Low       float64 `json:"low,string"`
			Close     float64 `json:"close,string"`
			Volume    float64 `json:"volume,string"`
		} `json:"ohlc"`
	} `json:"data"`
}
