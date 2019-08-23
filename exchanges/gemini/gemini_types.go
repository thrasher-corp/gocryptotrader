package gemini

import "github.com/thrasher-corp/gocryptotrader/currency"

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
	// Used to store values
	BaseCurrency  string
	QuoteCurrency string
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

// NotionalVolume api call for fees, all return fee amounts are in basis points
type NotionalVolume struct {
	APIAuctionFeeBPS      int64                  `json:"api_auction_fee_bps"`
	APIMakerFeeBPS        int64                  `json:"api_maker_fee_bps"`
	APITakerFeeBPS        int64                  `json:"api_taker_fee_bps"`
	BlockMakerFeeBPS      int64                  `json:"block_maker_fee_bps"`
	BlockTakerFeeBPS      int64                  `json:"block_taker_fee_bps"`
	FixAuctionFeeBPS      int64                  `json:"fix_auction_fee_bps"`
	FixMakerFeeBPS        int64                  `json:"fix_maker_fee_bps"`
	FixTakerFeeBPS        int64                  `json:"fix_taker_fee_bps"`
	OneDayNotionalVolumes []OneDayNotionalVolume `json:"notional_1d_volume"`
	ThirtyDayVolume       float64                `json:"notional_30d_volume"`
	WebAuctionFeeBPS      int64                  `json:"web_auction_fee_bps"`
	WebMakerFeeBPS        int64                  `json:"web_maker_fee_bps"`
	WebTakerFeeBPS        int64                  `json:"web_taker_fee_bps"`
	LastedUpdated         int64                  `json:"last_updated_ms"`
	Date                  string                 `json:"date"`
}

// OneDayNotionalVolume Contains the notioanl volume for a single day
type OneDayNotionalVolume struct {
	Date             string  `json:"date"`
	NotationalVolume float64 `json:"notional_volume"`
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
	Result  string  `json:"result"`
	Reason  string  `json:"reason"`
}

// ErrorCapture is a generlized error response from the server
type ErrorCapture struct {
	Result  string `json:"result"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// WsResponse generic response
type WsResponse struct {
	Type string `json:"type"`
}

// WsMarketUpdateResponse defines the main response type
type WsMarketUpdateResponse struct {
	Type           string  `json:"type"`
	EventID        int64   `json:"eventId"`
	Timestamp      int64   `json:"timestamp"`
	TimestampMS    int64   `json:"timestampms"`
	SocketSequence int64   `json:"socket_sequence"`
	Events         []Event `json:"events"`
}

// Event defines orderbook and trade data
type Event struct {
	Type      string  `json:"type"`
	Reason    string  `json:"reason"`
	Price     float64 `json:"price,string"`
	Delta     float64 `json:"delta,string"`
	Remaining float64 `json:"remaining,string"`
	Side      string  `json:"side"`
	MakerSide string  `json:"makerSide"`
	Amount    float64 `json:"amount,string"`
}

// ReadData defines read data from the websocket connection
type ReadData struct {
	Raw      []byte
	Currency currency.Pair
}

// WsRequestPayload Request info to subscribe to a WS enpoint
type WsRequestPayload struct {
	Request string `json:"request"`
	Nonce   int64  `json:"nonce"`
}

// WsSubscriptionAcknowledgementResponse The first message you receive acknowledges your subscription
type WsSubscriptionAcknowledgementResponse struct {
	Type             string   `json:"type"`
	AccountID        int64    `json:"accountId"`
	SubscriptionID   string   `json:"subscriptionId"`
	SymbolFilter     []string `json:"symbolFilter"`
	APISessionFilter []string `json:"apiSessionFilter"`
	EventTypeFilter  []string `json:"eventTypeFilter"`
}

// WsHeartbeatResponse Gemini will send a heartbeat every five seconds so you'll know your WebSocket connection is active.
type WsHeartbeatResponse struct {
	Type           string `json:"type"`
	Timestampms    int64  `json:"timestampms"`
	Sequence       int64  `json:"sequence"`
	TraceID        string `json:"trace_id"`
	SocketSequence int64  `json:"socket_sequence"`
}

// WsActiveOrdersResponse contains active orders
type WsActiveOrdersResponse struct {
	Type              string        `json:"type"`
	OrderID           string        `json:"order_id"`
	APISession        string        `json:"api_session"`
	Symbol            currency.Pair `json:"symbol"`
	Side              string        `json:"side"`
	OrderType         string        `json:"order_type"`
	Timestamp         string        `json:"timestamp"`
	Timestampms       int64         `json:"timestampms"`
	IsLive            bool          `json:"is_live"`
	IsCancelled       bool          `json:"is_cancelled"`
	IsHidden          bool          `json:"is_hidden"`
	AvgExecutionPrice float64       `json:"avg_execution_price,string"`
	ExecutedAmount    float64       `json:"executed_amount,string"`
	RemainingAmount   float64       `json:"remaining_amount,string"`
	OriginalAmount    float64       `json:"original_amount,string"`
	Price             float64       `json:"price,string"`
	SocketSequence    int64         `json:"socket_sequence"`
}

// WsOrderRejectedResponse ws response
type WsOrderRejectedResponse struct {
	Type           string        `json:"type"`
	OrderID        string        `json:"order_id"`
	EventID        string        `json:"event_id"`
	Reason         string        `json:"reason"`
	APISession     string        `json:"api_session"`
	Symbol         currency.Pair `json:"symbol"`
	Side           string        `json:"side"`
	OrderType      string        `json:"order_type"`
	Timestamp      string        `json:"timestamp"`
	Timestampms    int64         `json:"timestampms"`
	IsLive         bool          `json:"is_live"`
	OriginalAmount float64       `json:"original_amount,string"`
	Price          float64       `json:"price,string"`
	SocketSequence int64         `json:"socket_sequence"`
}

// WsOrderBookedResponse ws response
type WsOrderBookedResponse struct {
	Type              string        `json:"type"`
	OrderID           string        `json:"order_id"`
	EventID           string        `json:"event_id"`
	APISession        string        `json:"api_session"`
	Symbol            currency.Pair `json:"symbol"`
	Side              string        `json:"side"`
	OrderType         string        `json:"order_type"`
	Timestamp         string        `json:"timestamp"`
	Timestampms       int64         `json:"timestampms"`
	IsLive            bool          `json:"is_live"`
	IsCancelled       bool          `json:"is_cancelled"`
	IsHidden          bool          `json:"is_hidden"`
	AvgExecutionPrice float64       `json:"avg_execution_price,string"`
	ExecutedAmount    float64       `json:"executed_amount,string"`
	RemainingAmount   float64       `json:"remaining_amount,string"`
	OriginalAmount    float64       `json:"original_amount,string"`
	Price             float64       `json:"price,string"`
	SocketSequence    int64         `json:"socket_sequence"`
}

// WsOrderFilledResponse ws response
type WsOrderFilledResponse struct {
	Type              string            `json:"type"`
	OrderID           string            `json:"order_id"`
	APISession        string            `json:"api_session"`
	Symbol            currency.Pair     `json:"symbol"`
	Side              string            `json:"side"`
	OrderType         string            `json:"order_type"`
	Timestamp         string            `json:"timestamp"`
	Timestampms       int64             `json:"timestampms"`
	IsLive            bool              `json:"is_live"`
	IsCancelled       bool              `json:"is_cancelled"`
	IsHidden          bool              `json:"is_hidden"`
	AvgExecutionPrice float64           `json:"avg_execution_price,string"`
	ExecutedAmount    float64           `json:"executed_amount,string"`
	RemainingAmount   float64           `json:"remaining_amount,string"`
	OriginalAmount    float64           `json:"original_amount,string"`
	Price             float64           `json:"price,string"`
	Fill              WsOrderFilledData `json:"fill"`
	SocketSequence    int64             `json:"socket_sequence"`
}

// WsOrderFilledData ws response data
type WsOrderFilledData struct {
	TradeID     string  `json:"trade_id"`
	Liquidity   string  `json:"liquidity"`
	Price       float64 `json:"price,string"`
	Amount      float64 `json:"amount,string"`
	Fee         float64 `json:"fee,string"`
	FeeCurrency string  `json:"fee_currency"`
}

// WsOrderCancelledResponse ws response
type WsOrderCancelledResponse struct {
	Type              string        `json:"type"`
	OrderID           string        `json:"order_id"`
	EventID           string        `json:"event_id"`
	CancelCommandID   string        `json:"cancel_command_id,omitempty"`
	Reason            string        `json:"reason"`
	APISession        string        `json:"api_session"`
	Symbol            currency.Pair `json:"symbol"`
	Side              string        `json:"side"`
	OrderType         string        `json:"order_type"`
	Timestamp         string        `json:"timestamp"`
	Timestampms       int64         `json:"timestampms"`
	IsLive            bool          `json:"is_live"`
	IsCancelled       bool          `json:"is_cancelled"`
	IsHidden          bool          `json:"is_hidden"`
	AvgExecutionPrice float64       `json:"avg_execution_price,string"`
	ExecutedAmount    float64       `json:"executed_amount,string"`
	RemainingAmount   float64       `json:"remaining_amount,string"`
	OriginalAmount    float64       `json:"original_amount,string"`
	Price             float64       `json:"price,string"`
	SocketSequence    int64         `json:"socket_sequence"`
}

// WsOrderCancellationRejectedResponse ws response
type WsOrderCancellationRejectedResponse struct {
	Type              string        `json:"type"`
	OrderID           string        `json:"order_id"`
	EventID           string        `json:"event_id"`
	CancelCommandID   string        `json:"cancel_command_id"`
	Reason            string        `json:"reason"`
	APISession        string        `json:"api_session"`
	Symbol            currency.Pair `json:"symbol"`
	Side              string        `json:"side"`
	OrderType         string        `json:"order_type"`
	Timestamp         string        `json:"timestamp"`
	Timestampms       int64         `json:"timestampms"`
	IsLive            bool          `json:"is_live"`
	IsCancelled       bool          `json:"is_cancelled"`
	IsHidden          bool          `json:"is_hidden"`
	AvgExecutionPrice float64       `json:"avg_execution_price,string"`
	ExecutedAmount    float64       `json:"executed_amount,string"`
	RemainingAmount   float64       `json:"remaining_amount,string"`
	OriginalAmount    float64       `json:"original_amount,string"`
	Price             float64       `json:"price,string"`
	SocketSequence    int64         `json:"socket_sequence"`
}

// WsOrderClosedResponse ws response
type WsOrderClosedResponse struct {
	Type              string        `json:"type"`
	OrderID           string        `json:"order_id"`
	EventID           string        `json:"event_id"`
	APISession        string        `json:"api_session"`
	Symbol            currency.Pair `json:"symbol"`
	Side              string        `json:"side"`
	OrderType         string        `json:"order_type"`
	Timestamp         string        `json:"timestamp"`
	Timestampms       int64         `json:"timestampms"`
	IsLive            bool          `json:"is_live"`
	IsCancelled       bool          `json:"is_cancelled"`
	IsHidden          bool          `json:"is_hidden"`
	AvgExecutionPrice float64       `json:"avg_execution_price,string"`
	ExecutedAmount    float64       `json:"executed_amount,string"`
	RemainingAmount   float64       `json:"remaining_amount,string"`
	OriginalAmount    float64       `json:"original_amount,string"`
	Price             float64       `json:"price,string"`
	SocketSequence    int64         `json:"socket_sequence"`
}
