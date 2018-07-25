package gemini

import "github.com/kempeng/gocryptotrader/decimal"

// Ticker holds returned ticker data from the exchange
type Ticker struct {
	Ask    decimal.Decimal `json:"ask,string"`
	Bid    decimal.Decimal `json:"bid,string"`
	Last   decimal.Decimal `json:"last,string"`
	Volume struct {
		Currency  decimal.Decimal
		USD       decimal.Decimal
		BTC       decimal.Decimal
		ETH       decimal.Decimal
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
	Price  decimal.Decimal `json:"price,string"`
	Amount decimal.Decimal `json:"amount,string"`
}

// Trade holds trade history for a specific currency pair
type Trade struct {
	Timestamp   int64           `json:"timestamp"`
	Timestampms int64           `json:"timestampms"`
	TID         int64           `json:"tid"`
	Price       decimal.Decimal `json:"price,string"`
	Amount      decimal.Decimal `json:"amount,string"`
	Exchange    string          `json:"exchange"`
	Side        string          `json:"type"`
}

// Auction is generalized response type
type Auction struct {
	LastAuctionEID               int64           `json:"last_auction_eid"`
	ClosedUntilMs                int64           `json:"closed_until_ms"`
	LastAuctionPrice             decimal.Decimal `json:"last_auction_price,string"`
	LastAuctionQuantity          decimal.Decimal `json:"last_auction_quantity,string"`
	LastHighestBidPrice          decimal.Decimal `json:"last_highest_bid_price,string"`
	LastLowestAskPrice           decimal.Decimal `json:"last_lowest_ask_price,string"`
	NextAuctionMS                int64           `json:"next_auction_ms"`
	NextUpdateMS                 int64           `json:"next_update_ms"`
	MostRecentIndicativePrice    decimal.Decimal `json:"most_recent_indicative_price,string"`
	MostRecentIndicativeQuantity decimal.Decimal `json:"most_recent_indicative_quantity,string"`
	MostRecentHighestBidPrice    decimal.Decimal `json:"most_recent_highest_bid_price,string"`
	MostRecentLowestAskPrice     decimal.Decimal `json:"most_recent_lowest_ask_price,string"`
}

// AuctionHistory holds auction history information
type AuctionHistory struct {
	AuctionID       int64           `json:"auction_id"`
	AuctionPrice    decimal.Decimal `json:"auction_price,string"`
	AuctionQuantity decimal.Decimal `json:"auction_quantity,string"`
	EID             int64           `json:"eid"`
	HighestBidPrice decimal.Decimal `json:"highest_bid_price,string"`
	LowestAskPrice  decimal.Decimal `json:"lowest_ask_price,string"`
	AuctionResult   string          `json:"auction_result"`
	Timestamp       int64           `json:"timestamp"`
	TimestampMS     int64           `json:"timestampms"`
	EventType       string          `json:"event_type"`
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
	OrderID           int64           `json:"order_id,string"`
	ID                int64           `json:"id,string"`
	ClientOrderID     string          `json:"client_order_id"`
	Symbol            string          `json:"symbol"`
	Exchange          string          `json:"exchange"`
	Price             decimal.Decimal `json:"price,string"`
	AvgExecutionPrice decimal.Decimal `json:"avg_execution_price,string"`
	Side              string          `json:"side"`
	Type              string          `json:"type"`
	Timestamp         int64           `json:"timestamp,string"`
	TimestampMS       int64           `json:"timestampms"`
	IsLive            bool            `json:"is_live"`
	IsCancelled       bool            `json:"is_cancelled"`
	IsHidden          bool            `json:"is_hidden"`
	Options           []string        `json:"options"`
	WasForced         bool            `json:"was_forced"`
	ExecutedAmount    decimal.Decimal `json:"executed_amount,string"`
	RemainingAmount   decimal.Decimal `json:"remaining_amount,string"`
	OriginalAmount    decimal.Decimal `json:"original_amount,string"`
	Message           string          `json:"message"`
}

// TradeHistory holds trade history information
type TradeHistory struct {
	Price           decimal.Decimal `json:"price,string"`
	Amount          decimal.Decimal `json:"amount,string"`
	Timestamp       int64           `json:"timestamp"`
	TimestampMS     int64           `json:"timestampms"`
	Type            string          `json:"type"`
	FeeCurrency     string          `json:"fee_currency"`
	FeeAmount       decimal.Decimal `json:"fee_amount,string"`
	TID             int64           `json:"tid"`
	OrderID         int64           `json:"order_id,string"`
	Exchange        string          `json:"exchange"`
	IsAuctionFilled bool            `json:"is_auction_fill"`
	ClientOrderID   string          `json:"client_order_id"`
}

// TradeVolume holds Volume information
type TradeVolume struct {
	AccountID         int64           `json:"account_id"`
	Symbol            string          `json:"symbol"`
	BaseCurrency      string          `json:"base_currency"`
	NotionalCurrency  string          `json:"notional_currency"`
	Date              string          `json:"date_date"`
	TotalVolumeBase   decimal.Decimal `json:"total_volume_base"`
	MakerBuySellRatio decimal.Decimal `json:"maker_buy_sell_ratio"`
	BuyMakerBase      decimal.Decimal `json:"buy_maker_base"`
	BuyMakerNotional  decimal.Decimal `json:"buy_maker_notional"`
	BuyMakerCount     decimal.Decimal `json:"buy_maker_count"`
	SellMakerBase     decimal.Decimal `json:"sell_maker_base"`
	SellMakerNotional decimal.Decimal `json:"sell_maker_notional"`
	SellMakerCount    decimal.Decimal `json:"sell_maker_count"`
	BuyTakerBase      decimal.Decimal `json:"buy_taker_base"`
	BuyTakerNotional  decimal.Decimal `json:"buy_taker_notional"`
	BuyTakerCount     decimal.Decimal `json:"buy_taker_count"`
	SellTakerBase     decimal.Decimal `json:"sell_taker_base"`
	SellTakerNotional decimal.Decimal `json:"sell_taker_notional"`
	SellTakerCount    decimal.Decimal `json:"sell_taker_count"`
}

// Balance is a simple balance type
type Balance struct {
	Currency  string          `json:"currency"`
	Amount    decimal.Decimal `json:"amount,string"`
	Available decimal.Decimal `json:"available,string"`
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
	Address string          `json:"address"`
	Amount  decimal.Decimal `json:"amount"`
	TXHash  string          `json:"txHash"`
	Message string          `json:"message"`
}

// ErrorCapture is a generlized error response from the server
type ErrorCapture struct {
	Result  string `json:"result"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}
