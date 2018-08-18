package wex

import "github.com/thrasher-/gocryptotrader/decimal"

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
	High          decimal.Decimal
	Low           decimal.Decimal
	Avg           decimal.Decimal
	Vol           decimal.Decimal
	VolumeCurrent decimal.Decimal `json:"vol_cur"`
	Last          decimal.Decimal
	Buy           decimal.Decimal
	Sell          decimal.Decimal
	Updated       int64
}

// Orderbook stores the asks and bids orderbook information
type Orderbook struct {
	Asks [][]decimal.Decimal `json:"asks"`
	Bids [][]decimal.Decimal `json:"bids"`
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
	Type             string          `json:"sell"`
	Amount           decimal.Decimal `json:"amount"`
	Rate             decimal.Decimal `json:"rate"`
	TimestampCreated decimal.Decimal `json:"time_created"`
	Status           int             `json:"status"`
}

// Pair holds pair information
type Pair struct {
	DecimalPlaces int             `json:"decimal_places"`
	MinPrice      decimal.Decimal `json:"min_price"`
	MaxPrice      decimal.Decimal `json:"max_price"`
	MinAmount     decimal.Decimal `json:"min_amount"`
	Hidden        int             `json:"hidden"`
	Fee           decimal.Decimal `json:"fee"`
}

// AccountInfo stores the account information for a user
type AccountInfo struct {
	Funds      map[string]decimal.Decimal `json:"funds"`
	OpenOrders int                        `json:"open_orders"`
	Rights     struct {
		Info     int `json:"info"`
		Trade    int `json:"trade"`
		Withdraw int `json:"withdraw"`
	} `json:"rights"`
	ServerTime       decimal.Decimal `json:"server_time"`
	TransactionCount int             `json:"transaction_count"`
	Error            string          `json:"error"`
}

// OrderInfo stores order information
type OrderInfo struct {
	Pair             string          `json:"pair"`
	Type             string          `json:"sell"`
	StartAmount      decimal.Decimal `json:"start_amount"`
	Amount           decimal.Decimal `json:"amount"`
	Rate             decimal.Decimal `json:"rate"`
	TimestampCreated decimal.Decimal `json:"time_created"`
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

// TransHistory stores transaction history
type TransHistory struct {
	Type        int             `json:"type"`
	Amount      decimal.Decimal `json:"amount"`
	Currency    string          `json:"currency"`
	Description string          `json:"desc"`
	Status      int             `json:"status"`
	Timestamp   decimal.Decimal `json:"timestamp"`
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

// CoinDepositAddress stores a curency deposit address
type CoinDepositAddress struct {
	Address string `json:"address"`
	Error   string `json:"error"`
}

// WithdrawCoins stores information for a withdrawcoins request
type WithdrawCoins struct {
	TID        int64                      `json:"tId"`
	AmountSent decimal.Decimal            `json:"amountSent"`
	Funds      map[string]decimal.Decimal `json:"funds"`
	Error      string                     `json:"error"`
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
	CouponAmount   decimal.Decimal `json:"couponAmount,string"`
	CouponCurrency string          `json:"couponCurrency"`
	TransID        int64           `json:"transID"`
	Error          string          `json:"error"`
}
