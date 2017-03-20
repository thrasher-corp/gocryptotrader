package gdax

type GDAXTicker struct {
	TradeID int64   `json:"trade_id"`
	Price   float64 `json:"price,string"`
	Size    float64 `json:"size,string"`
	Time    string  `json:"time"`
}

type GDAXProduct struct {
	ID             string  `json:"id"`
	BaseCurrency   string  `json:"base_currency"`
	QuoteCurrency  string  `json:"quote_currency"`
	BaseMinSize    float64 `json:"base_min_size,string"`
	BaseMaxSize    int64   `json:"base_max_size,string"`
	QuoteIncrement float64 `json:"quote_increment,string"`
	DisplayName    string  `json:"string"`
}

type GDAXOrderL1L2 struct {
	Price     float64
	Amount    float64
	NumOrders float64
}

type GDAXOrderL3 struct {
	Price   float64
	Amount  float64
	OrderID string
}

type GDAXOrderbookL1L2 struct {
	Sequence int64             `json:"sequence"`
	Bids     [][]GDAXOrderL1L2 `json:"asks"`
	Asks     [][]GDAXOrderL1L2 `json:"asks"`
}

type GDAXOrderbookL3 struct {
	Sequence int64           `json:"sequence"`
	Bids     [][]GDAXOrderL3 `json:"asks"`
	Asks     [][]GDAXOrderL3 `json:"asks"`
}

type GDAXOrderbookResponse struct {
	Sequence int64           `json:"sequence"`
	Bids     [][]interface{} `json:"bids"`
	Asks     [][]interface{} `json:"asks"`
}

type GDAXTrade struct {
	TradeID int64   `json:"trade_id"`
	Price   float64 `json:"price,string"`
	Size    float64 `json:"size,string"`
	Time    string  `json:"time"`
	Side    string  `json:"side"`
}

type GDAXStats struct {
	Open   float64 `json:"open,string"`
	High   float64 `json:"high,string"`
	Low    float64 `json:"low,string"`
	Volume float64 `json:"volume,string"`
}

type GDAXCurrency struct {
	ID      string
	Name    string
	MinSize float64 `json:"min_size,string"`
}

type GDAXHistory struct {
	Time   int64
	Low    float64
	High   float64
	Open   float64
	Close  float64
	Volume float64
}

type GDAXAccountResponse struct {
	ID        string  `json:"id"`
	Balance   float64 `json:"balance,string"`
	Hold      float64 `json:"hold,string"`
	Available float64 `json:"available,string"`
	Currency  string  `json:"currency"`
}

type GDAXAccountLedgerResponse struct {
	ID        string      `json:"id"`
	CreatedAt string      `json:"created_at"`
	Amount    float64     `json:"amount,string"`
	Balance   float64     `json:"balance,string"`
	Type      string      `json:"type"`
	details   interface{} `json:"details"`
}

type GDAXAccountHolds struct {
	ID        string  `json:"id"`
	AccountID string  `json:"account_id"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	Amount    float64 `json:"amount,string"`
	Type      string  `json:"type"`
	Reference string  `json:"ref"`
}

type GDAXOrdersResponse struct {
	ID         string  `json:"id"`
	Size       float64 `json:"size,string"`
	Price      float64 `json:"price,string"`
	ProductID  string  `json:"product_id"`
	Status     string  `json:"status"`
	FilledSize float64 `json:"filled_size,string"`
	FillFees   float64 `json:"fill_fees,string"`
	Settled    bool    `json:"settled"`
	Side       string  `json:"side"`
	CreatedAt  string  `json:"created_at"`
}

type GDAXOrderResponse struct {
	ID         string  `json:"id"`
	Size       float64 `json:"size,string"`
	Price      float64 `json:"price,string"`
	DoneReason string  `json:"done_reason"`
	Status     string  `json:"status"`
	Settled    bool    `json:"settled"`
	FilledSize float64 `json:"filled_size,string"`
	ProductID  string  `json:"product_id"`
	FillFees   float64 `json:"fill_fees,string"`
	Side       string  `json:"side"`
	CreatedAt  string  `json:"created_at"`
	DoneAt     string  `json:"done_at"`
}

type GDAXFillResponse struct {
	TradeID   int     `json:"trade_id"`
	ProductID string  `json:"product_id"`
	Price     float64 `json:"price,string"`
	Size      float64 `json:"size,string"`
	OrderID   string  `json:"order_id"`
	CreatedAt string  `json:"created_at"`
	Liquidity string  `json:"liquidity"`
	Fee       float64 `json:"fee,string"`
	Settled   bool    `json:"settled"`
	Side      string  `json:"side"`
}

type GDAXReportResponse struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	CompletedAt string `json:"completed_at"`
	ExpiresAt   string `json:"expires_at"`
	FileURL     string `json:"file_url"`
	Params      struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	} `json:params"`
}

type GDAXWebsocketSubscribe struct {
	Type      string `json:"type"`
	ProductID string `json:"product_id"`
}

type GDAXWebsocketReceived struct {
	Type     string  `json:"type"`
	Time     string  `json:"time"`
	Sequence int     `json:"sequence"`
	OrderID  string  `json:"order_id"`
	Size     float64 `json:"size,string"`
	Price    float64 `json:"price,string"`
	Side     string  `json:"side"`
}

type GDAXWebsocketOpen struct {
	Type          string  `json:"type"`
	Time          string  `json:"time"`
	Sequence      int     `json:"sequence"`
	OrderID       string  `json:"order_id"`
	Price         float64 `json:"price,string"`
	RemainingSize float64 `json:"remaining_size,string"`
	Side          string  `json:"side"`
}

type GDAXWebsocketDone struct {
	Type          string  `json:"type"`
	Time          string  `json:"time"`
	Sequence      int     `json:"sequence"`
	Price         float64 `json:"price,string"`
	OrderID       string  `json:"order_id"`
	Reason        string  `json:"reason"`
	Side          string  `json:"side"`
	RemainingSize float64 `json:"remaining_size,string"`
}

type GDAXWebsocketMatch struct {
	Type         string  `json:"type"`
	TradeID      int     `json:"trade_id"`
	Sequence     int     `json:"sequence"`
	MakerOrderID string  `json:"maker_order_id"`
	TakerOrderID string  `json:"taker_order_id"`
	Time         string  `json:"time"`
	Size         float64 `json:"size,string"`
	Price        float64 `json:"price,string"`
	Side         string  `json:"side"`
}

type GDAXWebsocketChange struct {
	Type     string  `json:"type"`
	Time     string  `json:"time"`
	Sequence int     `json:"sequence"`
	OrderID  string  `json:"order_id"`
	NewSize  float64 `json:"new_size,string"`
	OldSize  float64 `json:"old_size,string"`
	Price    float64 `json:"price,string"`
	Side     string  `json:"side"`
}
