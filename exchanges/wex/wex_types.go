package wex

type Ticker struct {
	High    float64
	Low     float64
	Avg     float64
	Vol     float64
	Vol_cur float64
	Last    float64
	Buy     float64
	Sell    float64
	Updated int64
}

type Orderbook struct {
	Asks [][]float64 `json:"asks"`
	Bids [][]float64 `json:"bids"`
}

type Trades struct {
	Type      string  `json:"type"`
	Price     float64 `json:"bid"`
	Amount    float64 `json:"amount"`
	TID       int64   `json:"tid"`
	Timestamp int64   `json:"timestamp"`
}

type Response struct {
	Return  interface{} `json:"return"`
	Success int         `json:"success"`
	Error   string      `json:"error"`
}

type Pair struct {
	DecimalPlaces int     `json:"decimal_places"`
	MinPrice      float64 `json:"min_price"`
	MaxPrice      float64 `json:"max_price"`
	MinAmount     float64 `json:"min_amount"`
	Hidden        int     `json:"hidden"`
	Fee           float64 `json:"fee"`
}

type Info struct {
	ServerTime int64           `json:"server_time"`
	Pairs      map[string]Pair `json:"pairs"`
}

type AccountInfo struct {
	Funds      map[string]float64 `json:"funds"`
	OpenOrders int                `json:"open_orders"`
	Rights     struct {
		Info     int `json:"info"`
		Trade    int `json:"trade"`
		Withdraw int `json:"withdraw"`
	} `json:"rights"`
	ServerTime       float64 `json:"server_time"`
	TransactionCount int     `json:"transaction_count"`
}

type ActiveOrders struct {
	Pair             string  `json:"pair"`
	Type             string  `json:"sell"`
	Amount           float64 `json:"amount"`
	Rate             float64 `json:"rate"`
	TimestampCreated float64 `json:"time_created"`
	Status           int     `json:"status"`
}

type OrderInfo struct {
	Pair             string  `json:"pair"`
	Type             string  `json:"sell"`
	StartAmount      float64 `json:"start_amount"`
	Amount           float64 `json:"amount"`
	Rate             float64 `json:"rate"`
	TimestampCreated float64 `json:"time_created"`
	Status           int     `json:"status"`
}

type CancelOrder struct {
	OrderID float64            `json:"order_id"`
	Funds   map[string]float64 `json:"funds"`
}

type Trade struct {
	Received float64            `json:"received"`
	Remains  float64            `json:"remains"`
	OrderID  float64            `json:"order_id"`
	Funds    map[string]float64 `json:"funds"`
}

type TransHistory struct {
	Type        int     `json:"type"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Description string  `json:"desc"`
	Status      int     `json:"status"`
	Timestamp   float64 `json:"timestamp"`
}

type TradeHistory struct {
	Pair      string  `json:"pair"`
	Type      string  `json:"type"`
	Amount    float64 `json:"amount"`
	Rate      float64 `json:"rate"`
	OrderID   float64 `json:"order_id"`
	MyOrder   int     `json:"is_your_order"`
	Timestamp float64 `json:"timestamp"`
}

type CoinDepositAddress struct {
	Address string `json:"address"`
}

type WithdrawCoins struct {
	TID        int64              `json:"tId"`
	AmountSent float64            `json:"amountSent"`
	Funds      map[string]float64 `json:"funds"`
}

type CreateCoupon struct {
	Coupon  string             `json:"coupon"`
	TransID int64              `json:"transID"`
	Funds   map[string]float64 `json:"funds"`
}

type RedeemCoupon struct {
	CouponAmount   float64 `json:"couponAmount,string"`
	CouponCurrency string  `json:"couponCurrency"`
	TransID        int64   `json:"transID"`
}
