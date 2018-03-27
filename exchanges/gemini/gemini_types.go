package gemini

// Ticker holds returned ticker data from the exchange
type Ticker struct {
	Ask    float64 `json:"ask,string"`
	Bid    float64 `json:"bid,string"`
	Last   float64 `json:"last,string"`
	Volume struct {
		Currency  float64
		USD       float64
		BTC       float64
		ETH       float64
		Timestamp int64
	}
}

// Orderbook contains orderbook information for both bid and ask side
type Orderbook struct {
	Bids []OrderbookEntry `json:"bids"`
	Asks []OrderbookEntry `json:"asks"`
}

// OrderbookEntry subtype of orderbook information
type OrderbookEntry struct {
	Price  float64 `json:"price,string"`
	Amount float64 `json:"amount,string"`
}

// Trade holds trade history for a specific currency pair
type Trade struct {
	Timestamp   int64   `json:"timestamp"`
	Timestampms int64   `json:"timestampms"`
	TID         int64   `json:"tid"`
	Price       float64 `json:"price,string"`
	Amount      float64 `json:"amount,string"`
	Exchange    string  `json:"exchange"`
	Side        string  `json:"type"`
}

// Auction is generalized response type
type Auction struct {
	LastAuctionEID               int64   `json:"last_auction_eid"`
	ClosedUntilMs                int64   `json:"closed_until_ms"`
	LastAuctionPrice             float64 `json:"last_auction_price,string"`
	LastAuctionQuantity          float64 `json:"last_auction_quantity,string"`
	LastHighestBidPrice          float64 `json:"last_highest_bid_price,string"`
	LastLowestAskPrice           float64 `json:"last_lowest_ask_price,string"`
	NextAuctionMS                int64   `json:"next_auction_ms"`
	NextUpdateMS                 int64   `json:"next_update_ms"`
	MostRecentIndicativePrice    float64 `json:"most_recent_indicative_price,string"`
	MostRecentIndicativeQuantity float64 `json:"most_recent_indicative_quantity,string"`
	MostRecentHighestBidPrice    float64 `json:"most_recent_highest_bid_price,string"`
	MostRecentLowestAskPrice     float64 `json:"most_recent_lowest_ask_price,string"`
}

// AuctionHistory holds auction history information
type AuctionHistory struct {
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

// OrderResult holds cancelled order information
type OrderResult struct {
	Result  string `json:"result"`
	Details struct {
		CancelledOrders []string `json:"cancelledOrders"`
		CancelRejects   []string `json:"cancelRejects"`
	} `json:"details"`
	Message string `json:"message"`
}

// Order contains order information
type Order struct {
	OrderID           int64    `json:"order_id,string"`
	ID                int64    `json:"id,string"`
	ClientOrderID     string   `json:"client_order_id"`
	Symbol            string   `json:"symbol"`
	Exchange          string   `json:"exchange"`
	Price             float64  `json:"price,string"`
	AvgExecutionPrice float64  `json:"avg_execution_price,string"`
	Side              string   `json:"side"`
	Type              string   `json:"type"`
	Timestamp         int64    `json:"timestamp,string"`
	TimestampMS       int64    `json:"timestampms"`
	IsLive            bool     `json:"is_live"`
	IsCancelled       bool     `json:"is_cancelled"`
	IsHidden          bool     `json:"is_hidden"`
	Options           []string `json:"options"`
	WasForced         bool     `json:"was_forced"`
	ExecutedAmount    float64  `json:"executed_amount,string"`
	RemainingAmount   float64  `json:"remaining_amount,string"`
	OriginalAmount    float64  `json:"original_amount,string"`
	Message           string   `json:"message"`
}

// TradeHistory holds trade history information
type TradeHistory struct {
	Price           float64 `json:"price,string"`
	Amount          float64 `json:"amount,string"`
	Timestamp       int64   `json:"timestamp"`
	TimestampMS     int64   `json:"timestampms"`
	Type            string  `json:"type"`
	FeeCurrency     string  `json:"fee_currency"`
	FeeAmount       float64 `json:"fee_amount,string"`
	TID             int64   `json:"tid"`
	OrderID         int64   `json:"order_id,string"`
	Exchange        string  `json:"exchange"`
	IsAuctionFilled bool    `json:"is_auction_fill"`
	ClientOrderID   string  `json:"client_order_id"`
}

// TradeVolume holds Volume information
type TradeVolume struct {
	AccountID         int64   `json:"account_id"`
	Symbol            string  `json:"symbol"`
	BaseCurrency      string  `json:"base_currency"`
	NotionalCurrency  string  `json:"notional_currency"`
	Date              string  `json:"date_date"`
	TotalVolumeBase   float64 `json:"total_volume_base"`
	MakerBuySellRatio float64 `json:"maker_buy_sell_ratio"`
	BuyMakerBase      float64 `json:"buy_maker_base"`
	BuyMakerNotional  float64 `json:"buy_maker_notional"`
	BuyMakerCount     float64 `json:"buy_maker_count"`
	SellMakerBase     float64 `json:"sell_maker_base"`
	SellMakerNotional float64 `json:"sell_maker_notional"`
	SellMakerCount    float64 `json:"sell_maker_count"`
	BuyTakerBase      float64 `json:"buy_taker_base"`
	BuyTakerNotional  float64 `json:"buy_taker_notional"`
	BuyTakerCount     float64 `json:"buy_taker_count"`
	SellTakerBase     float64 `json:"sell_taker_base"`
	SellTakerNotional float64 `json:"sell_taker_notional"`
	SellTakerCount    float64 `json:"sell_taker_count"`
}

// Balance is a simple balance type
type Balance struct {
	Currency  string  `json:"currency"`
	Amount    float64 `json:"amount,string"`
	Available float64 `json:"available,string"`
}

// DepositAddress holds assigned deposit address for a specific currency
type DepositAddress struct {
	Currency string `json:"currency"`
	Address  string `json:"address"`
	Label    string `json:"label"`
	Message  string `json:"message"`
}

// WithdrawalAddress holds withdrawal information
type WithdrawalAddress struct {
	Address string  `json:"address"`
	Amount  float64 `json:"amount"`
	TXHash  string  `json:"txHash"`
	Message string  `json:"message"`
}

// ErrorCapture is a generlized error response from the server
type ErrorCapture struct {
	Result  string `json:"result"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}
