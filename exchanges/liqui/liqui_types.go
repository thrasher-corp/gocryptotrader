package liqui

// Info holds the current pair information as well as server time
type Info struct {
	ServerTime int64               `json:"server_time"`
	Pairs      map[string]PairData `json:"pairs"`
	Success    int                 `json:"success"`
	Error      string              `json:"error"`
}

// PairData is a sub-type for Info
type PairData struct {
	DecimalPlaces int     `json:"decimal_places"`
	MinPrice      float64 `json:"min_price"`
	MaxPrice      float64 `json:"max_price"`
	MinAmount     float64 `json:"min_amount"`
	Hidden        int     `json:"hidden"`
	Fee           float64 `json:"fee"`
}

// Ticker contains ticker information
type Ticker struct {
	High           float64
	Low            float64
	Avg            float64
	Vol            float64
	VolumeCurrency float64
	Last           float64
	Buy            float64
	Sell           float64
	Updated        int64
}

// Orderbook references both ask and bid sides
type Orderbook struct {
	Asks [][]float64 `json:"asks"`
	Bids [][]float64 `json:"bids"`
}

// Trades contains trade information
type Trades struct {
	Type      string  `json:"type"`
	Price     float64 `json:"bid"`
	Amount    float64 `json:"amount"`
	TID       int64   `json:"tid"`
	Timestamp int64   `json:"timestamp"`
}

// AccountInfo contains full account details information
type AccountInfo struct {
	Funds  map[string]float64 `json:"funds"`
	Rights struct {
		Info     bool `json:"info"`
		Trade    bool `json:"trade"`
		Withdraw bool `json:"withdraw"`
	} `json:"rights"`
	ServerTime       float64 `json:"server_time"`
	TransactionCount int     `json:"transaction_count"`
	OpenOrders       int     `json:"open_orders"`
	Success          int     `json:"success"`
	Error            string  `json:"error"`
}

// ActiveOrders holds active order information
type ActiveOrders struct {
	Pair             string  `json:"pair"`
	Type             string  `json:"sell"`
	Amount           float64 `json:"amount"`
	Rate             float64 `json:"rate"`
	TimestampCreated float64 `json:"time_created"`
	Status           int     `json:"status"`
	Success          int     `json:"success"`
	Error            string  `json:"error"`
}

// OrderInfo holds specific order information
type OrderInfo struct {
	Pair             string  `json:"pair"`
	Type             string  `json:"sell"`
	StartAmount      float64 `json:"start_amount"`
	Amount           float64 `json:"amount"`
	Rate             float64 `json:"rate"`
	TimestampCreated float64 `json:"time_created"`
	Status           int     `json:"status"`
	Success          int     `json:"success"`
	Error            string  `json:"error"`
}

// CancelOrder holds cancelled order information
type CancelOrder struct {
	OrderID float64            `json:"order_id"`
	Funds   map[string]float64 `json:"funds"`
	Success int                `json:"success"`
	Error   string             `json:"error"`
}

// Trade holds trading information
type Trade struct {
	Received float64            `json:"received"`
	Remains  float64            `json:"remains"`
	OrderID  float64            `json:"order_id"`
	Funds    map[string]float64 `json:"funds"`
	Success  int                `json:"success"`
	Error    string             `json:"error"`
}

// TradeHistory contains trade history data
type TradeHistory struct {
	Pair      string  `json:"pair"`
	Type      string  `json:"type"`
	Amount    float64 `json:"amount"`
	Rate      float64 `json:"rate"`
	OrderID   float64 `json:"order_id"`
	MyOrder   int     `json:"is_your_order"`
	Timestamp float64 `json:"timestamp"`
	Success   int     `json:"success"`
	Error     string  `json:"error"`
}

// Response is a generalized return type
type Response struct {
	Return  interface{} `json:"return"`
	Success int         `json:"success"`
	Error   string      `json:"error"`
}

// WithdrawCoins shows the amount of coins withdrawn from liqui not yet available
type WithdrawCoins struct {
	TID        int64              `json:"tId"`
	AmountSent float64            `json:"amountSent"`
	Funds      map[string]float64 `json:"funds"`
	Success    int                `json:"success"`
	Error      string             `json:"error"`
}
