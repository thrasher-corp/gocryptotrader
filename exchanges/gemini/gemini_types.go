package gemini

type GeminiOrderbookEntry struct {
	Price  float64 `json:"price,string"`
	Amount float64 `json:"amount,string"`
}

type GeminiOrderbook struct {
	Bids []GeminiOrderbookEntry `json:"bids"`
	Asks []GeminiOrderbookEntry `json:"asks"`
}

type GeminiTrade struct {
	Timestamp int64   `json:"timestamp"`
	TID       int64   `json:"tid"`
	Price     float64 `json:"price"`
	Amount    float64 `json:"amount"`
	Side      string  `json:"taker"`
}

type GeminiOrder struct {
	OrderID           int64   `json:"order_id"`
	ClientOrderID     string  `json:"client_order_id"`
	Symbol            string  `json:"symbol"`
	Exchange          string  `json:"exchange"`
	Price             float64 `json:"price,string"`
	AvgExecutionPrice float64 `json:"avg_execution_price,string"`
	Side              string  `json:"side"`
	Type              string  `json:"type"`
	Timestamp         int64   `json:"timestamp"`
	TimestampMS       int64   `json:"timestampms"`
	IsLive            bool    `json:"is_live"`
	IsCancelled       bool    `json:"is_cancelled"`
	WasForced         bool    `json:"was_forced"`
	ExecutedAmount    float64 `json:"executed_amount,string"`
	RemainingAmount   float64 `json:"remaining_amount,string"`
	OriginalAmount    float64 `json:"original_amount,string"`
}

type GeminiOrderResult struct {
	Result bool `json:"result"`
}

type GeminiTradeHistory struct {
	Price         float64 `json:"price"`
	Amount        float64 `json:"amount"`
	Timestamp     int64   `json:"timestamp"`
	TimestampMS   int64   `json:"timestampms"`
	Type          string  `json:"type"`
	FeeCurrency   string  `json:"fee_currency"`
	FeeAmount     float64 `json:"fee_amount"`
	TID           int64   `json:"tid"`
	OrderID       int64   `json:"order_id"`
	ClientOrderID string  `json:"client_order_id"`
}

type GeminiBalance struct {
	Currency  string  `json:"currency"`
	Amount    float64 `json:"amount"`
	Available float64 `json:"available"`
}

type GeminiTicker struct {
	Ask    float64 `json:"ask,string"`
	Bid    float64 `json:"bid,string"`
	Last   float64 `json:"last,string"`
	Volume struct {
		Currency  float64
		USD       float64
		Timestamp int64
	}
}

type GeminiAuction struct {
	LastAuctionPrice    float64 `json:"last_auction_price,string"`
	LastAuctionQuantity float64 `json:"last_auction_quantity,string"`
	LastHighestBidPrice float64 `json:"last_highest_bid_price,string"`
	LastLowestAskPrice  float64 `json:"last_lowest_ask_price,string"`
	NextUpdateMS        int64   `json:"next_update_ms"`
	NextAuctionMS       int64   `json:"next_auction_ms"`
	LastAuctionEID      int64   `json:"last_auction_eid"`
}
type GeminiAuctionHistory struct {
	AuctionID       int64   `json:"auction_id"`
	AuctionPrice    float64 `json:"auction_price,string"`
	AuctionQuantity float64 `json:"auction_quantity,string"`
	EID             int64   `json:"eid"`
	HighestBidPrice float64 `json:"highest_bid_price,string"`
	LowestAskPrice  float64 `json:"lowest_ask_price,string"`
	AuctionResult   string  `json:"auction_result"`
	Timestamp       int64   `json:"timestamp"`
	TimestampMS     int64   `json:"timestampms"`
	EventType       string  `json:"event_type"`
}
