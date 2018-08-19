package bitstamp

import "github.com/shopspring/decimal"

// Ticker holds ticker information
type Ticker struct {
	Last      decimal.Decimal `json:"last,string"`
	High      decimal.Decimal `json:"high,string"`
	Low       decimal.Decimal `json:"low,string"`
	Vwap      decimal.Decimal `json:"vwap,string"`
	Volume    decimal.Decimal `json:"volume,string"`
	Bid       decimal.Decimal `json:"bid,string"`
	Ask       decimal.Decimal `json:"ask,string"`
	Timestamp int64           `json:"timestamp,string"`
	Open      decimal.Decimal `json:"open,string"`
}

// OrderbookBase holds singular price information
type OrderbookBase struct {
	Price  decimal.Decimal
	Amount decimal.Decimal
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
	Date    int64           `json:"date,string"`
	TradeID int64           `json:"tid,string"`
	Price   decimal.Decimal `json:"price,string"`
	Type    int             `json:"type,string"`
	Amount  decimal.Decimal `json:"amount,string"`
}

// EURUSDConversionRate holds buy sell conversion rate information
type EURUSDConversionRate struct {
	Buy  decimal.Decimal `json:"buy,string"`
	Sell decimal.Decimal `json:"sell,string"`
}

// Balances holds full balance information with the supplied APIKEYS
type Balances struct {
	USDBalance   decimal.Decimal `json:"usd_balance,string"`
	BTCBalance   decimal.Decimal `json:"btc_balance,string"`
	EURBalance   decimal.Decimal `json:"eur_balance,string"`
	XRPBalance   decimal.Decimal `json:"xrp_balance,string"`
	USDReserved  decimal.Decimal `json:"usd_reserved,string"`
	BTCReserved  decimal.Decimal `json:"btc_reserved,string"`
	EURReserved  decimal.Decimal `json:"eur_reserved,string"`
	XRPReserved  decimal.Decimal `json:"xrp_reserved,string"`
	USDAvailable decimal.Decimal `json:"usd_available,string"`
	BTCAvailable decimal.Decimal `json:"btc_available,string"`
	EURAvailable decimal.Decimal `json:"eur_available,string"`
	XRPAvailable decimal.Decimal `json:"xrp_available,string"`
	BTCUSDFee    decimal.Decimal `json:"btcusd_fee,string"`
	BTCEURFee    decimal.Decimal `json:"btceur_fee,string"`
	EURUSDFee    decimal.Decimal `json:"eurusd_fee,string"`
	XRPUSDFee    decimal.Decimal `json:"xrpusd_fee,string"`
	XRPEURFee    decimal.Decimal `json:"xrpeur_fee,string"`
	XRPBTCFee    decimal.Decimal `json:"xrpbtc_fee,string"`
	Fee          decimal.Decimal `json:"fee,string"`
}

// UserTransactions holds user transaction information
type UserTransactions struct {
	Date    string          `json:"datetime"`
	TransID int64           `json:"id"`
	Type    int             `json:"type,string"`
	USD     decimal.Decimal `json:"usd"`
	EUR     decimal.Decimal `json:"eur"`
	BTC     decimal.Decimal `json:"btc"`
	XRP     decimal.Decimal `json:"xrp"`
	BTCUSD  decimal.Decimal `json:"btc_usd"`
	Fee     decimal.Decimal `json:"fee,string"`
	OrderID int64           `json:"order_id"`
}

// Order holds current open order data
type Order struct {
	ID     int64           `json:"id"`
	Date   string          `json:"datetime"`
	Type   int             `json:"type"`
	Price  decimal.Decimal `json:"price"`
	Amount decimal.Decimal `json:"amount"`
}

// OrderStatus holds order status information
type OrderStatus struct {
	Status       string
	Transactions []struct {
		TradeID int64           `json:"tid"`
		USD     decimal.Decimal `json:"usd,string"`
		Price   decimal.Decimal `json:"price,string"`
		Fee     decimal.Decimal `json:"fee,string"`
		BTC     decimal.Decimal `json:"btc,string"`
	}
}

// WithdrawalRequests holds request information on withdrawals
type WithdrawalRequests struct {
	OrderID       int64           `json:"id"`
	Date          string          `json:"datetime"`
	Type          int             `json:"type"`
	Amount        decimal.Decimal `json:"amount,string"`
	Status        int             `json:"status"`
	Data          interface{}
	Address       string `json:"address"`        // Bitcoin withdrawals only
	TransactionID string `json:"transaction_id"` // Bitcoin withdrawals only
}

// UnconfirmedBTCTransactions holds address information about unconfirmed
// transactions
type UnconfirmedBTCTransactions struct {
	Amount        decimal.Decimal `json:"amount,string"`
	Address       string          `json:"address"`
	Confirmations int             `json:"confirmations"`
}

// CaptureError is used to capture unmarshalled errors
type CaptureError struct {
	Status interface{} `json:"status"`
	Reason interface{} `json:"reason"`
	Code   interface{} `json:"code"`
	Error  interface{} `json:"error"`
}
