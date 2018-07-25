package yobit

import "github.com/kempeng/gocryptotrader/decimal"

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
	High          decimal.Decimal // maximal price
	Low           decimal.Decimal // minimal price
	Avg           decimal.Decimal // average price
	Vol           decimal.Decimal // traded volume
	VolumeCurrent decimal.Decimal `json:"vol_cur"` // traded volume in currency
	Last          decimal.Decimal // last transaction price
	Buy           decimal.Decimal // buying price
	Sell          decimal.Decimal // selling price
	Updated       int64           // last cache upgrade
}

// Orderbook stores the asks and bids orderbook information
type Orderbook struct {
	Asks [][]decimal.Decimal `json:"asks"` // selling orders
	Bids [][]decimal.Decimal `json:"bids"` // buying orders
}

// Trades stores trade information
type Trades struct {
	Type      string          `json:"type"`
	Price     decimal.Decimal `json:"bid"`
	Amount    decimal.Decimal `json:"amount"`
	TID       int64           `json:"tid"`
	Timestamp int64           `json:"timestamp"`
}

// ActiveOrders stores active order information
type ActiveOrders struct {
	Pair             string          `json:"pair"`
	Type             string          `json:"type"`
	Amount           decimal.Decimal `json:"amount"`
	Rate             decimal.Decimal `json:"rate"`
	TimestampCreated decimal.Decimal `json:"timestamp_created"`
	Status           int             `json:"status"`
}

// Pair holds pair information
type Pair struct {
	DecimalPlaces int             `json:"decimal_places"` // Quantity of permitted numbers after decimal point
	MinPrice      decimal.Decimal `json:"min_price"`      // Minimal permitted price
	MaxPrice      decimal.Decimal `json:"max_price"`      // Maximal permitted price
	MinAmount     decimal.Decimal `json:"min_amount"`     // Minimal permitted buy or sell amount
	Hidden        int             `json:"hidden"`         // Pair is hidden (0 or 1)
	Fee           decimal.Decimal `json:"fee"`            // Pair commission
}

// AccountInfo stores the account information for a user
type AccountInfo struct {
	Funds           map[string]decimal.Decimal `json:"funds"`
	FundsInclOrders map[string]decimal.Decimal `json:"funds_incl_orders"`
	Rights          struct {
		Info     int `json:"info"`
		Trade    int `json:"trade"`
		Withdraw int `json:"withdraw"`
	} `json:"rights"`
	TransactionCount int             `json:"transaction_count"`
	OpenOrders       int             `json:"open_orders"`
	ServerTime       decimal.Decimal `json:"server_time"`
	Error            string          `json:"error"`
}

// OrderInfo stores order information
type OrderInfo struct {
	Pair             string          `json:"pair"`
	Type             string          `json:"type"`
	StartAmount      decimal.Decimal `json:"start_amount"`
	Amount           decimal.Decimal `json:"amount"`
	Rate             decimal.Decimal `json:"rate"`
	TimestampCreated decimal.Decimal `json:"timestamp_created"`
	Status           int             `json:"status"`
}

// CancelOrder is used for the CancelOrder API request response
type CancelOrder struct {
	OrderID decimal.Decimal            `json:"order_id"`
	Funds   map[string]decimal.Decimal `json:"funds"`
	Error   string                     `json:"error"`
}

// Trade stores the trade information
type Trade struct {
	Received decimal.Decimal            `json:"received"`
	Remains  decimal.Decimal            `json:"remains"`
	OrderID  decimal.Decimal            `json:"order_id"`
	Funds    map[string]decimal.Decimal `json:"funds"`
	Error    string                     `json:"error"`
}

// TradeHistory stores trade history
type TradeHistory struct {
	Pair      string          `json:"pair"`
	Type      string          `json:"type"`
	Amount    decimal.Decimal `json:"amount"`
	Rate      decimal.Decimal `json:"rate"`
	OrderID   decimal.Decimal `json:"order_id"`
	MyOrder   int             `json:"is_your_order"`
	Timestamp decimal.Decimal `json:"timestamp"`
}

// DepositAddress stores a curency deposit address
type DepositAddress struct {
	Address         string          `json:"address"`
	ProcessedAmount decimal.Decimal `json:"processed_amount"`
	ServerTime      int64           `json:"server_time"`
	Error           string          `json:"error"`
}

// WithdrawCoinsToAddress stores information for a withdrawcoins request
type WithdrawCoinsToAddress struct {
	ServerTime int64  `json:"server_time"`
	Error      string `json:"error"`
}

// CreateCoupon stores information coupon information
type CreateCoupon struct {
	Coupon  string                     `json:"coupon"`
	TransID int64                      `json:"transID"`
	Funds   map[string]decimal.Decimal `json:"funds"`
	Error   string                     `json:"error"`
}

// RedeemCoupon stores redeem coupon information
type RedeemCoupon struct {
	CouponAmount   decimal.Decimal            `json:"couponAmount,string"`
	CouponCurrency string                     `json:"couponCurrency"`
	TransID        int64                      `json:"transID"`
	Funds          map[string]decimal.Decimal `json:"funds"`
	Error          string                     `json:"error"`
}
