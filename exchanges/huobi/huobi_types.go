package huobi

// Response stores the Huobi response information
type Response struct {
	Status       string `json:"status"`
	Channel      string `json:"ch"`
	Timestamp    int64  `json:"ts"`
	ErrorCode    string `json:"err-code"`
	ErrorMessage string `json:"err-msg"`
}

// KlineItem stores a kline item
type KlineItem struct {
	ID     int     `json:"id"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	Low    float64 `json:"low"`
	High   float64 `json:"high"`
	Amount float64 `json:"amount"`
	Vol    float64 `json:"vol"`
	Count  int     `json:"count"`
}

// Klines stores tan array of kline items
type Klines struct {
	Klines []KlineItem `json:"data"`
}

// DetailMerged stores the ticker detail merged data
type DetailMerged struct {
	Detail
	Version int       `json:"version"`
	Ask     []float64 `json:"ask"`
	Bid     []float64 `json:"bid"`
}

// Orderbook stores the orderbook data
type Orderbook struct {
	ID         int64       `json:"id"`
	Timetstamp int64       `json:"ts"`
	Bids       [][]float64 `json:"bids"`
	Asks       [][]float64 `json:"asks"`
}

// Trade stores the trade data
type Trade struct {
	ID        float64 `json:"id"`
	Price     float64 `json:"price"`
	Amount    float64 `json:"amount"`
	Direction string  `json:"direction"`
	Timestamp int64   `json:"ts"`
}

// TradeHistory stores the the trade history data
type TradeHistory struct {
	ID        int64   `json:"id"`
	Timestamp int64   `json:"ts"`
	Trades    []Trade `json:"data"`
}

// Detail stores the ticker detail data
type Detail struct {
	Amount    float64 `json:"amount"`
	Open      float64 `json:"open"`
	Close     float64 `json:"close"`
	High      float64 `json:"high"`
	Timestamp int64   `json:"id"`
	ID        int     `json:"id"`
	Count     int     `json:"count"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"vol"`
}

// Symbol stores the symbol data
type Symbol struct {
	BaseCurrency    string `json:"base-currency"`
	QuoteCurrency   string `json:"quote-currency"`
	PricePrecision  int    `json:"price-precision"`
	AmountPrecision int    `json:"amount-precision"`
	SymbolPartition string `json:"symbol-partition"`
}

// Account stores the account data
type Account struct {
	ID     int64  `json:"id"`
	Type   string `json:"type"`
	State  string `json:"working"`
	UserID int64  `json:"user-id"`
}

// AccountBalance stores the user account balance
type AccountBalance struct {
	Currency string  `json:"currency"`
	Type     string  `json:"type"`
	Balance  float64 `json:"balance,string"`
}

// CancelOrderBatch stores the cancel order batch data
type CancelOrderBatch struct {
	Success []string `json:"success"`
	Failed  []struct {
		OrderID      int64  `json:"order-id,string"`
		ErrorCode    string `json:"err-code"`
		ErrorMessage string `json:"err-msg"`
	} `json:"failed"`
}

// OrderInfo stores the order info
type OrderInfo struct {
	ID              int    `json:"id"`
	Symbol          string `json:"symbol"`
	AccountID       int    `json:"account-id"`
	Amount          string `json:"amount"`
	Price           string `json:"price"`
	CreatedAt       int64  `json:"created-at"`
	Type            string `json:"type"`
	FieldAmount     string `json:"field-amount"`
	FieldCashAmount string `json:"field-cash-amount"`
	FieldFees       string `json:"field-fees"`
	FinishedAt      int64  `json:"finished-at"`
	UserID          int    `json:"user-id"`
	Source          string `json:"source"`
	State           string `json:"state"`
	CanceledAt      int    `json:"canceled-at"`
	Exchange        string `json:"exchange"`
	Batch           string `json:"batch"`
}

// OrderMatchInfo stores the order match info
type OrderMatchInfo struct {
	ID           int    `json:"id"`
	OrderID      int    `json:"order-id"`
	MatchID      int    `json:"match-id"`
	Symbol       string `json:"symbol"`
	Type         string `json:"type"`
	Source       string `json:"source"`
	Price        string `json:"price"`
	FilledAmount string `json:"filled-amount"`
	FilledFees   string `json:"filled-fees"`
	CreatedAt    int64  `json:"created-at"`
}

// MarginOrder stores the margin order info
type MarginOrder struct {
	Currency        string `json:"currency"`
	Symbol          string `json:"symbol"`
	AccruedAt       int64  `json:"accrued-at"`
	LoanAmount      string `json:"loan-amount"`
	LoanBalance     string `json:"loan-balance"`
	InterestBalance string `json:"interest-balance"`
	CreatedAt       int64  `json:"created-at"`
	InterestAmount  string `json:"interest-amount"`
	InterestRate    string `json:"interest-rate"`
	AccountID       int    `json:"account-id"`
	UserID          int    `json:"user-id"`
	UpdatedAt       int64  `json:"updated-at"`
	ID              int    `json:"id"`
	State           string `json:"state"`
}

// MarginAccountBalance stores the margin account balance info
type MarginAccountBalance struct {
	ID       int              `json:"id"`
	Type     string           `json:"type"`
	State    string           `json:"state"`
	Symbol   string           `json:"symbol"`
	FlPrice  string           `json:"fl-price"`
	FlType   string           `json:"fl-type"`
	RiskRate string           `json:"risk-rate"`
	List     []AccountBalance `json:"list"`
}
