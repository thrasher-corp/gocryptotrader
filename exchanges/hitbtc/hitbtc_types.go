package hitbtc

import "time"

type Ticker struct {
	Last        float64   `json:"last,string"`        // Last trade price
	Ask         float64   `json:"ask,string"`         // Best ask price
	Bid         float64   `json:"bid,string"`         // Best bid price
	Timestamp   time.Time `json:"timestamp,string"`   // Last update or refresh ticker timestamp
	Volume      float64   `json:"volume,string"`      // Total trading amount within 24 hours in base currency
	VolumeQuote float64   `json:"volumeQuote,string"` // Total trading amount within 24 hours in quote currency
	Symbol      string    `json:"symbol"`
	High        float64   `json:"high,string"` // Highest trade price within 24 hours
	Low         float64   `json:"low,string"`  // Lowest trade price within 24 hours
	Open        float64   `json:"open,string"` // Last trade price 24 hours ago
}

type Symbol struct {
	Id                   string  `json:"id"` // Symbol identifier. In the future, the description will simply use the symbol
	BaseCurrency         string  `json:"baseCurrency"`
	QuoteCurrency        string  `json:"quoteCurrency"`
	QuantityIncrement    float64 `json:"quantityIncrement,string"`
	TickSize             float64 `json:"tickSize,string"`
	TakeLiquidityRate    float64 `json:"takeLiquidityRate,string"`    // Default fee rate
	ProvideLiquidityRate float64 `json:"provideLiquidityRate,string"` // Default fee rate for market making trades
	FeeCurrency          string  `json:"feeCurrency"`                 // Default fee rate for market making trades
}

type OrderbookResponse struct {
	Asks []OrderbookItem `json:"ask"` // Ask side array of levels
	Bids []OrderbookItem `json:"bid"` // Bid side array of levels
}

type OrderbookItem struct {
	Price  float64 `json:"price,string"` // Price level
	Amount float64 `json:"size,string"`  // Total volume of orders with the specified price
}

type Orderbook struct {
	Asks []OrderbookItem `json:"asks"`
	Bids []OrderbookItem `json:"bids"`
}

type TradeHistory struct {
	Id        int64   `json:"id"`              // Trade id
	Timestamp string  `json:"timestamp"`       // Trade timestamp
	Side      string  `json:"side"`            // Trade side sell or buy
	Price     float64 `json:"price,string"`    // Trade price
	Quantity  float64 `json:"quantity,string"` // Trade quantity
}

type ChartData struct {
	Timestamp   time.Time `json:"timestamp,string"`
	Max         float64   `json:"max,string"`         // Max price
	Min         float64   `json:"min,string"`         // Min price
	Open        float64   `json:"open,string"`        // Open price
	Close       float64   `json:"close,string"`       // Close price
	Volume      float64   `json:"volume,string"`      // Volume in base currency
	VolumeQuote float64   `json:"volumeQuote,string"` // Volume in quote currency
}

type Currencies struct {
	Id                 string `json:"id"`                 // Currency identifier.
	FullName           string `json:"fullName"`           // Currency full name
	Crypto             bool   `json:"crypto,boolean"`     // Is currency belongs to blockchain (false for ICO and fiat, like EUR)
	PayinEnabled       bool   `json:"payinEnabled"`       // Is allowed for deposit (false for ICO)
	PayinPaymentId     bool   `json:"payinPaymentId"`     // Is required to provide additional information other than the address for deposit
	PayinConfirmations int64  `json:"payinConfirmations"` // Blocks confirmations count for deposit
	PayoutEnabled      bool   `json:"payoutEnabled"`      // Is allowed for withdraw (false for ICO)
	PayoutIsPaymentId  bool   `json:"payoutIsPaymentId"`  // Is allowed to provide additional information for withdraw
	TransferEnabled    bool   `json:"transferEnabled"`    // Is allowed to transfer between trading and account (may be disabled on maintain)
}

type LoanOrder struct {
	Rate     float64 `json:"rate,string"`
	Amount   float64 `json:"amount,string"`
	RangeMin int     `json:"rangeMin"`
	RangeMax int     `json:"rangeMax"`
}

type LoanOrders struct {
	Offers  []LoanOrder `json:"offers"`
	Demands []LoanOrder `json:"demands"`
}

type Balance struct {
	Currency  string  `json:"currency"`
	Available float64 `json:"available,string"` // Amount available for trading or transfer to main account
	Reserved  float64 `json:"reserved,string"`  // Amount reserved for active orders or incomplete transfers to main account

}

type DepositCryptoAddresses struct {
	Address   string `json:"address"`   // Address for deposit
	PaymentId string `json:"paymentId"` // Optional additional parameter. Required for deposit if persist
}

type Order struct {
	Id            int64  `json:"id,string"`     //  Unique identifier for Order as assigned by exchange
	ClientOrderId string `json:"clientOrderId"` // Unique identifier for Order as assigned by trader. Uniqueness must be
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

type OpenOrdersResponseAll struct {
	Data map[string][]Order
}

type OpenOrdersResponse struct {
	Data []Order
}

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

type AuthenticatedTradeHistoryAll struct {
	Data map[string][]AuthentictedTradeHistory
}

type AuthenticatedTradeHistoryResponse struct {
	Data []AuthentictedTradeHistory
}

type ResultingTrades struct {
	Amount  float64 `json:"amount,string"`
	Date    string  `json:"date"`
	Rate    float64 `json:"rate,string"`
	Total   float64 `json:"total,string"`
	TradeID int64   `json:"tradeID,string"`
	Type    string  `json:"type"`
}

type OrderResponse struct {
	OrderNumber int64             `json:"orderNumber,string"`
	Trades      []ResultingTrades `json:"resultingTrades"`
}

type GenericResponse struct {
	Success int    `json:"success"`
	Error   string `json:"error"`
}

type MoveOrderResponse struct {
	Success     int                          `json:"success"`
	Error       string                       `json:"error"`
	OrderNumber int64                        `json:"orderNumber,string"`
	Trades      map[string][]ResultingTrades `json:"resultingTrades"`
}

type Withdraw struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

type Fee struct {
	TakeLiquidityRate    float64 `json:"takeLiquidityRate,string"`    // Taker
	ProvideLiquidityRate float64 `json:"provideLiquidityRate,string"` // Maker
}

type Margin struct {
	TotalValue    float64 `json:"totalValue,string"`
	ProfitLoss    float64 `json:"pl,string"`
	LendingFees   float64 `json:"lendingFees,string"`
	NetValue      float64 `json:"netValue,string"`
	BorrowedValue float64 `json:"totalBorrowedValue,string"`
	CurrentMargin float64 `json:"currentMargin,string"`
}

type MarginPosition struct {
	Amount            float64 `json:"amount,string"`
	Total             float64 `json:"total,string"`
	BasePrice         float64 `json:"basePrice,string"`
	LiquidiationPrice float64 `json:"liquidiationPrice"`
	ProfitLoss        float64 `json:"pl,string"`
	LendingFees       float64 `json:"lendingFees,string"`
	Type              string  `json:"type"`
}

type LoanOffer struct {
	ID        int64   `json:"id"`
	Rate      float64 `json:"rate,string"`
	Amount    float64 `json:"amount,string"`
	Duration  int     `json:"duration"`
	AutoRenew bool    `json:"autoRenew,int"`
	Date      string  `json:"date"`
}

type ActiveLoans struct {
	Provided []LoanOffer `json:"provided"`
	Used     []LoanOffer `json:"used"`
}

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
