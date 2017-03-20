package liqui

type LiquiTicker struct {
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

type LiquiOrderbook struct {
	Asks [][]float64 `json:"asks"`
	Bids [][]float64 `json:"bids"`
}

type LiquiTrades struct {
	Type      string  `json:"type"`
	Price     float64 `json:"bid"`
	Amount    float64 `json:"amount"`
	TID       int64   `json:"tid"`
	Timestamp int64   `json:"timestamp"`
}

type LiquiResponse struct {
	Return  interface{} `json:"return"`
	Success int         `json:"success"`
	Error   string      `json:"error"`
}

type LiquiPair struct {
	DecimalPlaces int     `json:"decimal_places"`
	MinPrice      float64 `json:"min_price"`
	MaxPrice      float64 `json:"max_price"`
	MinAmount     float64 `json:"min_amount"`
	Hidden        int     `json:"hidden"`
	Fee           float64 `json:"fee"`
}

type LiquiInfo struct {
	ServerTime int64                `json:"server_time"`
	Pairs      map[string]LiquiPair `json:"pairs"`
}

type LiquiAccountInfo struct {
	Funds  map[string]float64 `json:"funds"`
	Rights struct {
		Info     bool `json:"info"`
		Trade    bool `json:"trade"`
		Withdraw bool `json:"withdraw"`
	} `json:"rights"`
	ServerTime       float64 `json:"server_time"`
	TransactionCount int     `json:"transaction_count"`
	OpenOrders       int     `json:"open_orders"`
}

type LiquiTrade struct {
	Received float64            `json:"received"`
	Remains  float64            `json:"remains"`
	OrderID  float64            `json:"order_id"`
	Funds    map[string]float64 `json:"funds"`
}

type LiquiActiveOrders struct {
	Pair             string  `json:"pair"`
	Type             string  `json:"sell"`
	Amount           float64 `json:"amount"`
	Rate             float64 `json:"rate"`
	TimestampCreated float64 `json:"time_created"`
	Status           int     `json:"status"`
}

type LiquiOrderInfo struct {
	Pair             string  `json:"pair"`
	Type             string  `json:"sell"`
	StartAmount      float64 `json:"start_amount"`
	Amount           float64 `json:"amount"`
	Rate             float64 `json:"rate"`
	TimestampCreated float64 `json:"time_created"`
	Status           int     `json:"status"`
}

type LiquiCancelOrder struct {
	OrderID float64            `json:"order_id"`
	Funds   map[string]float64 `json:"funds"`
}

type LiquiTradeHistory struct {
	Pair      string  `json:"pair"`
	Type      string  `json:"type"`
	Amount    float64 `json:"amount"`
	Rate      float64 `json:"rate"`
	OrderID   float64 `json:"order_id"`
	MyOrder   int     `json:"is_your_order"`
	Timestamp float64 `json:"timestamp"`
}

type LiquiWithdrawCoins struct {
	TID        int64              `json:"tId"`
	AmountSent float64            `json:"amountSent"`
	Funds      map[string]float64 `json:"funds"`
}
