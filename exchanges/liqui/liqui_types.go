package liqui

import "github.com/kempeng/gocryptotrader/decimal"

// Info holds the current pair information as well as server time
type Info struct {
	ServerTime int64               `json:"server_time"`
	Pairs      map[string]PairData `json:"pairs"`
	Success    int                 `json:"success"`
	Error      string              `json:"error"`
}

// PairData is a sub-type for Info
type PairData struct {
	DecimalPlaces int             `json:"decimal_places"`
	MinPrice      decimal.Decimal `json:"min_price"`
	MaxPrice      decimal.Decimal `json:"max_price"`
	MinAmount     decimal.Decimal `json:"min_amount"`
	Hidden        int             `json:"hidden"`
	Fee           decimal.Decimal `json:"fee"`
}

// Ticker contains ticker information
type Ticker struct {
	High           decimal.Decimal
	Low            decimal.Decimal
	Avg            decimal.Decimal
	Vol            decimal.Decimal
	VolumeCurrency decimal.Decimal
	Last           decimal.Decimal
	Buy            decimal.Decimal
	Sell           decimal.Decimal
	Updated        int64
}

// Orderbook references both ask and bid sides
type Orderbook struct {
	Asks [][]decimal.Decimal `json:"asks"`
	Bids [][]decimal.Decimal `json:"bids"`
}

// Trades contains trade information
type Trades struct {
	Type      string          `json:"type"`
	Price     decimal.Decimal `json:"bid"`
	Amount    decimal.Decimal `json:"amount"`
	TID       int64           `json:"tid"`
	Timestamp int64           `json:"timestamp"`
}

// AccountInfo contains full account details information
type AccountInfo struct {
	Funds  map[string]decimal.Decimal `json:"funds"`
	Rights struct {
		Info     bool `json:"info"`
		Trade    bool `json:"trade"`
		Withdraw bool `json:"withdraw"`
	} `json:"rights"`
	ServerTime       decimal.Decimal `json:"server_time"`
	TransactionCount int             `json:"transaction_count"`
	OpenOrders       int             `json:"open_orders"`
	Success          int             `json:"success"`
	Error            string          `json:"error"`
}

// ActiveOrders holds active order information
type ActiveOrders struct {
	Pair             string          `json:"pair"`
	Type             string          `json:"sell"`
	Amount           decimal.Decimal `json:"amount"`
	Rate             decimal.Decimal `json:"rate"`
	TimestampCreated decimal.Decimal `json:"time_created"`
	Status           int             `json:"status"`
	Success          int             `json:"success"`
	Error            string          `json:"error"`
}

// OrderInfo holds specific order information
type OrderInfo struct {
	Pair             string          `json:"pair"`
	Type             string          `json:"sell"`
	StartAmount      decimal.Decimal `json:"start_amount"`
	Amount           decimal.Decimal `json:"amount"`
	Rate             decimal.Decimal `json:"rate"`
	TimestampCreated decimal.Decimal `json:"time_created"`
	Status           int             `json:"status"`
	Success          int             `json:"success"`
	Error            string          `json:"error"`
}

// CancelOrder holds cancelled order information
type CancelOrder struct {
	OrderID decimal.Decimal            `json:"order_id"`
	Funds   map[string]decimal.Decimal `json:"funds"`
	Success int                        `json:"success"`
	Error   string                     `json:"error"`
}

// Trade holds trading information
type Trade struct {
	Received decimal.Decimal            `json:"received"`
	Remains  decimal.Decimal            `json:"remains"`
	OrderID  decimal.Decimal            `json:"order_id"`
	Funds    map[string]decimal.Decimal `json:"funds"`
	Success  int                        `json:"success"`
	Error    string                     `json:"error"`
}

// TradeHistory contains trade history data
type TradeHistory struct {
	Pair      string          `json:"pair"`
	Type      string          `json:"type"`
	Amount    decimal.Decimal `json:"amount"`
	Rate      decimal.Decimal `json:"rate"`
	OrderID   decimal.Decimal `json:"order_id"`
	MyOrder   int             `json:"is_your_order"`
	Timestamp decimal.Decimal `json:"timestamp"`
	Success   int             `json:"success"`
	Error     string          `json:"error"`
}

// Response is a generalized return type
type Response struct {
	Return  interface{} `json:"return"`
	Success int         `json:"success"`
	Error   string      `json:"error"`
}

// WithdrawCoins shows the amount of coins withdrawn from liqui not yet available
type WithdrawCoins struct {
	TID        int64                      `json:"tId"`
	AmountSent decimal.Decimal            `json:"amountSent"`
	Funds      map[string]decimal.Decimal `json:"funds"`
	Success    int                        `json:"success"`
	Error      string                     `json:"error"`
}
