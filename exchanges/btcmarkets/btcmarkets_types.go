package btcmarkets

// Response is the genralized response type
type Response struct {
	Success      bool   `json:"success"`
	ErrorCode    int    `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
	ID           int    `json:"id"`
	Responses    []struct {
		Success      bool   `json:"success"`
		ErrorCode    int    `json:"errorCode"`
		ErrorMessage string `json:"errorMessage"`
		ID           int64  `json:"id"`
	}
	ClientRequestID string  `json:"clientRequestId"`
	Orders          []Order `json:"orders"`
	Status          string  `json:"status"`
}

// Ticker holds ticker information
type Ticker struct {
	BestBID    float64 `json:"bestBid"`
	BestAsk    float64 `json:"bestAsk"`
	LastPrice  float64 `json:"lastPrice"`
	Currency   string  `json:"currency"`
	Instrument string  `json:"instrument"`
	Timestamp  int64   `json:"timestamp"`
	Volume     float64 `json:"volume24h"`
}

// Orderbook holds current orderbook information returned from the exchange
type Orderbook struct {
	Currency   string      `json:"currency"`
	Instrument string      `json:"instrument"`
	Timestamp  int64       `json:"timestamp"`
	Asks       [][]float64 `json:"asks"`
	Bids       [][]float64 `json:"bids"`
}

// Trade holds trade information
type Trade struct {
	TradeID int64   `json:"tid"`
	Amount  float64 `json:"amount"`
	Price   float64 `json:"price"`
	Date    int64   `json:"date"`
}

// OrderToGo holds order information to be sent to the exchange
type OrderToGo struct {
	Currency        string `json:"currency"`
	Instrument      string `json:"instrument"`
	Price           int64  `json:"price"`
	Volume          int64  `json:"volume"`
	OrderSide       string `json:"orderSide"`
	OrderType       string `json:"ordertype"`
	ClientRequestID string `json:"clientRequestId"`
}

// Order holds order information
type Order struct {
	ID              int64           `json:"id"`
	Currency        string          `json:"currency"`
	Instrument      string          `json:"instrument"`
	OrderSide       string          `json:"orderSide"`
	OrderType       string          `json:"ordertype"`
	CreationTime    float64         `json:"creationTime"`
	Status          string          `json:"status"`
	ErrorMessage    string          `json:"errorMessage"`
	Price           float64         `json:"price"`
	Volume          float64         `json:"volume"`
	OpenVolume      float64         `json:"openVolume"`
	ClientRequestID string          `json:"clientRequestId"`
	Trades          []TradeResponse `json:"trades"`
}

// TradeResponse holds trade information
type TradeResponse struct {
	ID           int64   `json:"id"`
	CreationTime float64 `json:"creationTime"`
	Description  string  `json:"description"`
	Price        float64 `json:"price"`
	Volume       float64 `json:"volume"`
	Fee          float64 `json:"fee"`
}

// AccountBalance holds account balance details
type AccountBalance struct {
	Balance      float64 `json:"balance"`
	PendingFunds float64 `json:"pendingFunds"`
	Currency     string  `json:"currency"`
}

// WithdrawRequestCrypto is a generalized withdraw request type
type WithdrawRequestCrypto struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Address  string `json:"address"`
}

// WithdrawRequestAUD is a generalized withdraw request type
type WithdrawRequestAUD struct {
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	AccountName   string `json:"accountName"`
	AccountNumber string `json:"accountNumber"`
	BankName      string `json:"bankName"`
	BSBNumber     string `json:"bsbNumber"`
}
