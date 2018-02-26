package hitbtc

import "time"

// Ticker holds ticker information
type Ticker struct {
	Last        float64
	Ask         float64
	Bid         float64
	Timestamp   time.Time
	Volume      float64
	VolumeQuote float64
	Symbol      string
	High        float64
	Low         float64
	Open        float64
}

// TickerResponse is the response type
type TickerResponse struct {
	Last        string    `json:"last"`             // Last trade price
	Ask         string    `json:"ask"`              // Best ask price
	Bid         string    `json:"bid"`              // Best bid price
	Timestamp   time.Time `json:"timestamp,string"` // Last update or refresh ticker timestamp
	Volume      string    `json:"volume"`           // Total trading amount within 24 hours in base currency
	VolumeQuote string    `json:"volumeQuote"`      // Total trading amount within 24 hours in quote currency
	Symbol      string    `json:"symbol"`
	High        string    `json:"high"` // Highest trade price within 24 hours
	Low         string    `json:"low"`  // Lowest trade price within 24 hours
	Open        string    `json:"open"` // Last trade price 24 hours ago
}

// Symbol holds symbol data
type Symbol struct {
	ID                   string  `json:"id"` // Symbol identifier. In the future, the description will simply use the symbol
	BaseCurrency         string  `json:"baseCurrency"`
	QuoteCurrency        string  `json:"quoteCurrency"`
	QuantityIncrement    float64 `json:"quantityIncrement,string"`
	TickSize             float64 `json:"tickSize,string"`
	TakeLiquidityRate    float64 `json:"takeLiquidityRate,string"`    // Default fee rate
	ProvideLiquidityRate float64 `json:"provideLiquidityRate,string"` // Default fee rate for market making trades
	FeeCurrency          string  `json:"feeCurrency"`                 // Default fee rate for market making trades
}

// OrderbookResponse is the full orderbook response
type OrderbookResponse struct {
	Asks []OrderbookItem `json:"ask"` // Ask side array of levels
	Bids []OrderbookItem `json:"bid"` // Bid side array of levels
}

// OrderbookItem is a sub type for orderbook response
type OrderbookItem struct {
	Price  float64 `json:"price,string"` // Price level
	Amount float64 `json:"size,string"`  // Total volume of orders with the specified price
}

// Orderbook contains orderbook data
type Orderbook struct {
	Asks []OrderbookItem `json:"asks"`
	Bids []OrderbookItem `json:"bids"`
}

// TradeHistory contains trade history data
type TradeHistory struct {
	ID        int64   `json:"id"`              // Trade id
	Timestamp string  `json:"timestamp"`       // Trade timestamp
	Side      string  `json:"side"`            // Trade side sell or buy
	Price     float64 `json:"price,string"`    // Trade price
	Quantity  float64 `json:"quantity,string"` // Trade quantity
}

// ChartData contains chart data
type ChartData struct {
	Timestamp   time.Time `json:"timestamp,string"`
	Max         float64   `json:"max,string"`         // Max price
	Min         float64   `json:"min,string"`         // Min price
	Open        float64   `json:"open,string"`        // Open price
	Close       float64   `json:"close,string"`       // Close price
	Volume      float64   `json:"volume,string"`      // Volume in base currency
	VolumeQuote float64   `json:"volumeQuote,string"` // Volume in quote currency
}

// Currencies hold the full range of data for a specified currency
type Currencies struct {
	ID                 string `json:"id"`                 // Currency identifier.
	FullName           string `json:"fullName"`           // Currency full name
	Crypto             bool   `json:"crypto,boolean"`     // Is currency belongs to blockchain (false for ICO and fiat, like EUR)
	PayinEnabled       bool   `json:"payinEnabled"`       // Is allowed for deposit (false for ICO)
	PayinPaymentID     bool   `json:"payinPaymentId"`     // Is required to provide additional information other than the address for deposit
	PayinConfirmations int64  `json:"payinConfirmations"` // Blocks confirmations count for deposit
	PayoutEnabled      bool   `json:"payoutEnabled"`      // Is allowed for withdraw (false for ICO)
	PayoutIsPaymentID  bool   `json:"payoutIsPaymentId"`  // Is allowed to provide additional information for withdraw
	TransferEnabled    bool   `json:"transferEnabled"`    // Is allowed to transfer between trading and account (may be disabled on maintain)
}

// LoanOrder contains information about your loans
type LoanOrder struct {
	Rate     float64 `json:"rate,string"`
	Amount   float64 `json:"amount,string"`
	RangeMin int     `json:"rangeMin"`
	RangeMax int     `json:"rangeMax"`
}

// LoanOrders holds information on the full range of loan orders
type LoanOrders struct {
	Offers  []LoanOrder `json:"offers"`
	Demands []LoanOrder `json:"demands"`
}

// Balance is a simple balance type
type Balance struct {
	Currency  string  `json:"currency"`
	Available float64 `json:"available,string"` // Amount available for trading or transfer to main account
	Reserved  float64 `json:"reserved,string"`  // Amount reserved for active orders or incomplete transfers to main account

}

// DepositCryptoAddresses contains address information
type DepositCryptoAddresses struct {
	Address   string `json:"address"`   // Address for deposit
	PaymentID string `json:"paymentId"` // Optional additional parameter. Required for deposit if persist
}

// Order contains information about an order
type Order struct {
	ID            int64  `json:"id,string"`     //  Unique identifier for Order as assigned by exchange
	ClientOrderID string `json:"clientOrderId"` // Unique identifier for Order as assigned by trader. Uniqueness must be
	// guaranteed within a single trading day, including all active orders.
	Symbol      string `json:"symbol"`      // Trading symbol
	Side        string `json:"side"`        // sell buy
	Status      string `json:"status"`      // new, suspended, partiallyFilled, filled, canceled, expired
	Type        string `json:"type"`        // Enum: limit, market, stopLimit, stopMarket
	TimeInForce string `json:"timeInForce"` // Time in force is a special instruction used when placing a trade to
	//   indicate how long an order will remain active before it is executed or expires
	// GTC - Good till cancel. GTC order won't close until it is filled.
	// IOC - An immediate or cancel order is an order to buy or sell that must be executed immediately, and any portion
	//   of the order that cannot be immediately filled is cancelled.
	// FOK - Fill or kill is a type of time-in-force designation used in securities trading that instructs a brokerage
	//   to execute a transaction immediately and completely or not at all.
	// Day - keeps the order active until the end of the trading day in UTC.
	// GTD - Good till date specified in expireTime.
	Quantity    float64   `json:"quantity,string"`    // Order quantity
	Price       float64   `json:"price,string"`       // Order price
	CumQuantity float64   `json:"cumQuantity,string"` // Cumulative executed quantity
	CreatedAt   time.Time `json:"createdAt,string"`
	UpdatedAt   time.Time `json:"updatedAt,string"`
	StopPrice   float64   `json:"stopPrice,string"`
	ExpireTime  time.Time `json:"expireTime,string"`
}

// OpenOrdersResponseAll holds the full open order response
type OpenOrdersResponseAll struct {
	Data map[string][]Order
}

// OpenOrdersResponse contains open order information
type OpenOrdersResponse struct {
	Data []Order
}

// AuthentictedTradeHistory contains trade history data
type AuthentictedTradeHistory struct {
	GlobalTradeID int64   `json:"globalTradeID"`
	TradeID       int64   `json:"tradeID,string"`
	Date          string  `json:"date"`
	Rate          float64 `json:"rate,string"`
	Amount        float64 `json:"amount,string"`
	Total         float64 `json:"total,string"`
	Fee           float64 `json:"fee,string"`
	OrderNumber   int64   `json:"orderNumber,string"`
	Type          string  `json:"type"`
	Category      string  `json:"category"`
}

// AuthenticatedTradeHistoryAll contains the full trade history
type AuthenticatedTradeHistoryAll struct {
	Data map[string][]AuthentictedTradeHistory
}

// AuthenticatedTradeHistoryResponse is the resp type for trade history
type AuthenticatedTradeHistoryResponse struct {
	Data []AuthentictedTradeHistory
}

// ResultingTrades holds resulting trade information
type ResultingTrades struct {
	Amount  float64 `json:"amount,string"`
	Date    string  `json:"date"`
	Rate    float64 `json:"rate,string"`
	Total   float64 `json:"total,string"`
	TradeID int64   `json:"tradeID,string"`
	Type    string  `json:"type"`
}

// OrderResponse holds the order response information
type OrderResponse struct {
	OrderNumber int64             `json:"orderNumber,string"`
	Trades      []ResultingTrades `json:"resultingTrades"`
}

// GenericResponse is the common response from HitBTC
type GenericResponse struct {
	Success int    `json:"success"`
	Error   string `json:"error"`
}

// MoveOrderResponse holds information about a move order
type MoveOrderResponse struct {
	Success     int                          `json:"success"`
	Error       string                       `json:"error"`
	OrderNumber int64                        `json:"orderNumber,string"`
	Trades      map[string][]ResultingTrades `json:"resultingTrades"`
}

// Withdraw holds response for a withdrawel process
type Withdraw struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

// Fee holds fee structure
type Fee struct {
	TakeLiquidityRate    float64 `json:"takeLiquidityRate,string"`    // Taker
	ProvideLiquidityRate float64 `json:"provideLiquidityRate,string"` // Maker
}

// Margin holds full margin information
type Margin struct {
	TotalValue    float64 `json:"totalValue,string"`
	ProfitLoss    float64 `json:"pl,string"`
	LendingFees   float64 `json:"lendingFees,string"`
	NetValue      float64 `json:"netValue,string"`
	BorrowedValue float64 `json:"totalBorrowedValue,string"`
	CurrentMargin float64 `json:"currentMargin,string"`
}

// MarginPosition holds information about your current margin position
type MarginPosition struct {
	Amount            float64 `json:"amount,string"`
	Total             float64 `json:"total,string"`
	BasePrice         float64 `json:"basePrice,string"`
	LiquidiationPrice float64 `json:"liquidiationPrice"`
	ProfitLoss        float64 `json:"pl,string"`
	LendingFees       float64 `json:"lendingFees,string"`
	Type              string  `json:"type"`
}

// LoanOffer holds information about your loan offers
type LoanOffer struct {
	ID        int64   `json:"id"`
	Rate      float64 `json:"rate,string"`
	Amount    float64 `json:"amount,string"`
	Duration  int     `json:"duration"`
	AutoRenew bool    `json:"autoRenew,int"`
	Date      string  `json:"date"`
}

// ActiveLoans holds information about your active loans
type ActiveLoans struct {
	Provided []LoanOffer `json:"provided"`
	Used     []LoanOffer `json:"used"`
}

// LendingHistory contains lending history data
type LendingHistory struct {
	ID       int64   `json:"id"`
	Currency string  `json:"currency"`
	Rate     float64 `json:"rate,string"`
	Amount   float64 `json:"amount,string"`
	Duration float64 `json:"duration,string"`
	Interest float64 `json:"interest,string"`
	Fee      float64 `json:"fee,string"`
	Earned   float64 `json:"earned,string"`
	Open     string  `json:"open"`
	Close    string  `json:"close"`
}
