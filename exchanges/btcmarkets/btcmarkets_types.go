package btcmarkets

type BTCMarketsTicker struct {
	BestBID    float64
	BestAsk    float64
	LastPrice  float64
	Currency   string
	Instrument string
	Timestamp  int64
}

type BTCMarketsTrade struct {
	TradeID int64   `json:"tid"`
	Amount  float64 `json:"amount"`
	Price   float64 `json:"price"`
	Date    int64   `json:"date"`
}

type BTCMarketsOrderbook struct {
	Currency   string      `json:"currency"`
	Instrument string      `json:"instrument"`
	Timestamp  int64       `json:"timestamp"`
	Asks       [][]float64 `json:"asks"`
	Bids       [][]float64 `json:"bids"`
}

type BTCMarketsTradeResponse struct {
	ID           int64   `json:"id"`
	CreationTime float64 `json:"creationTime"`
	Description  string  `json:"description"`
	Price        float64 `json:"price"`
	Volume       float64 `json:"volume"`
	Fee          float64 `json:"fee"`
}

type BTCMarketsOrder struct {
	ID              int64                     `json:"id"`
	Currency        string                    `json:"currency"`
	Instrument      string                    `json:"instrument"`
	OrderSide       string                    `json:"orderSide"`
	OrderType       string                    `json:"ordertype"`
	CreationTime    float64                   `json:"creationTime"`
	Status          string                    `json:"status"`
	ErrorMessage    string                    `json:"errorMessage"`
	Price           float64                   `json:"price"`
	Volume          float64                   `json:"volume"`
	OpenVolume      float64                   `json:"openVolume"`
	ClientRequestId string                    `json:"clientRequestId"`
	Trades          []BTCMarketsTradeResponse `json:"trades"`
}

type BTCMarketsAccountBalance struct {
	Balance      float64 `json:"balance"`
	PendingFunds float64 `json:"pendingFunds"`
	Currency     string  `json:"currency"`
}
