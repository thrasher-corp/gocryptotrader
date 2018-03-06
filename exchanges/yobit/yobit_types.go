package yobit

// Response is a generic struct used for exchange API request result
type Response struct {
	Return  interface{} `json:"return"`
	Success int         `json:"success"`
	Error   string      `json:"error"`
}

// Info holds server time and pair information
type Info struct {
	ServerTime int64           `json:"server_time"`
	Pairs      map[string]Pair `json:"pairs"`
}

// Ticker stores the ticker information
type Ticker struct {
	High          float64 // maximal price
	Low           float64 // minimal price
	Avg           float64 // average price
	Vol           float64 // traded volume
	VolumeCurrent float64 `json:"vol_cur"` // traded volume in currency
	Last          float64 // last transaction price
	Buy           float64 // buying price
	Sell          float64 // selling price
	Updated       int64   // last cache upgrade
}

// Orderbook stores the asks and bids orderbook information
type Orderbook struct {
	Asks [][]float64 `json:"asks"` // selling orders
	Bids [][]float64 `json:"bids"` // buying orders
}

// Trades stores trade information
type Trades struct {
	Type      string  `json:"type"`
	Price     float64 `json:"bid"`
	Amount    float64 `json:"amount"`
	TID       int64   `json:"tid"`
	Timestamp int64   `json:"timestamp"`
}

// ActiveOrders stores active order information
type ActiveOrders struct {
	Pair             string  `json:"pair"`
	Type             string  `json:"type"`
	Amount           float64 `json:"amount"`
	Rate             float64 `json:"rate"`
	TimestampCreated float64 `json:"timestamp_created"`
	Status           int     `json:"status"`
}

// Pair holds pair information
type Pair struct {
	DecimalPlaces int     `json:"decimal_places"` // Quantity of permitted numbers after decimal point
	MinPrice      float64 `json:"min_price"`      // Minimal permitted price
	MaxPrice      float64 `json:"max_price"`      // Maximal permitted price
	MinAmount     float64 `json:"min_amount"`     // Minimal permitted buy or sell amount
	Hidden        int     `json:"hidden"`         // Pair is hidden (0 or 1)
	Fee           float64 `json:"fee"`            // Pair commission
}

// AccountInfo stores the account information for a user
type AccountInfo struct {
	Funds           map[string]float64 `json:"funds"`
	FundsInclOrders map[string]float64 `json:"funds_incl_orders"`
	Rights          struct {
		Info     int `json:"info"`
		Trade    int `json:"trade"`
		Withdraw int `json:"withdraw"`
	} `json:"rights"`
	TransactionCount int     `json:"transaction_count"`
	OpenOrders       int     `json:"open_orders"`
	ServerTime       float64 `json:"server_time"`
	Error            string  `json:"error"`
}

// OrderInfo stores order information
type OrderInfo struct {
	Pair             string  `json:"pair"`
	Type             string  `json:"type"`
	StartAmount      float64 `json:"start_amount"`
	Amount           float64 `json:"amount"`
	Rate             float64 `json:"rate"`
	TimestampCreated float64 `json:"timestamp_created"`
	Status           int     `json:"status"`
}

// CancelOrder is used for the CancelOrder API request response
type CancelOrder struct {
	OrderID float64            `json:"order_id"`
	Funds   map[string]float64 `json:"funds"`
	Error   string             `json:"error"`
}

// Trade stores the trade information
type Trade struct {
	Received float64            `json:"received"`
	Remains  float64            `json:"remains"`
	OrderID  float64            `json:"order_id"`
	Funds    map[string]float64 `json:"funds"`
	Error    string             `json:"error"`
}

// TradeHistory stores trade history
type TradeHistory struct {
	Pair      string  `json:"pair"`
	Type      string  `json:"type"`
	Amount    float64 `json:"amount"`
	Rate      float64 `json:"rate"`
	OrderID   float64 `json:"order_id"`
	MyOrder   int     `json:"is_your_order"`
	Timestamp float64 `json:"timestamp"`
}

// DepositAddress stores a curency deposit address
type DepositAddress struct {
	Address         string  `json:"address"`
	ProcessedAmount float64 `json:"processed_amount"`
	ServerTime      int64   `json:"server_time"`
	Error           string  `json:"error"`
}

// WithdrawCoinsToAddress stores information for a withdrawcoins request
type WithdrawCoinsToAddress struct {
	ServerTime int64  `json:"server_time"`
	Error      string `json:"error"`
}

// CreateCoupon stores information coupon information
type CreateCoupon struct {
	Coupon  string             `json:"coupon"`
	TransID int64              `json:"transID"`
	Funds   map[string]float64 `json:"funds"`
	Error   string             `json:"error"`
}

// RedeemCoupon stores redeem coupon information
type RedeemCoupon struct {
	CouponAmount   float64            `json:"couponAmount,string"`
	CouponCurrency string             `json:"couponCurrency"`
	TransID        int64              `json:"transID"`
	Funds          map[string]float64 `json:"funds"`
	Error          string             `json:"error"`
}
