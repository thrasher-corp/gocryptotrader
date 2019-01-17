package bitstamp

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

// Balances holds full balance information with the supplied APIKEYS
type Balances struct {
	USDBalance   float64 `json:"usd_balance,string"`
	BTCBalance   float64 `json:"btc_balance,string"`
	EURBalance   float64 `json:"eur_balance,string"`
	XRPBalance   float64 `json:"xrp_balance,string"`
	USDReserved  float64 `json:"usd_reserved,string"`
	BTCReserved  float64 `json:"btc_reserved,string"`
	EURReserved  float64 `json:"eur_reserved,string"`
	XRPReserved  float64 `json:"xrp_reserved,string"`
	USDAvailable float64 `json:"usd_available,string"`
	BTCAvailable float64 `json:"btc_available,string"`
	EURAvailable float64 `json:"eur_available,string"`
	XRPAvailable float64 `json:"xrp_available,string"`
	BTCUSDFee    float64 `json:"btcusd_fee,string"`
	BTCEURFee    float64 `json:"btceur_fee,string"`
	EURUSDFee    float64 `json:"eurusd_fee,string"`
	XRPUSDFee    float64 `json:"xrpusd_fee,string"`
	XRPEURFee    float64 `json:"xrpeur_fee,string"`
	XRPBTCFee    float64 `json:"xrpbtc_fee,string"`
	Fee          float64 `json:"fee,string"`
}

// UserTransactions holds user transaction information
type UserTransactions struct {
	Date    int64   `json:"datetime"`
	TransID int64   `json:"id"`
	Type    int     `json:"type,string"`
	USD     float64 `json:"usd"`
	EUR     float64 `json:"eur"`
	BTC     float64 `json:"btc"`
	XRP     float64 `json:"xrp"`
	BTCUSD  float64 `json:"btc_usd"`
	Fee     float64 `json:"fee,string"`
	OrderID int64   `json:"order_id"`
}

// Order holds current open order data
type Order struct {
	ID       int64   `json:"id"`
	Date     int64   `json:"datetime"`
	Type     int     `json:"type"`
	Price    float64 `json:"price"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency_pair"`
}

// OrderStatus holds order status information
type OrderStatus struct {
	Status       string
	Transactions []struct {
		TradeID int64   `json:"tid"`
		USD     float64 `json:"usd,string"`
		Price   float64 `json:"price,string"`
		Fee     float64 `json:"fee,string"`
		BTC     float64 `json:"btc,string"`
	}
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
	ID    string `json:"id"`
	Error string `json:"error"`
}

// FIATWithdrawalResponse response from a fiat withdrawal request
type FIATWithdrawalResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Reason string `json:"reason"`
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
