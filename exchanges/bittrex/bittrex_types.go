package bittrex

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// CancelOrderRequest holds request data for CancelOrder
type CancelOrderRequest struct {
	OrderID int64 `json:"orderId,string"`
}

// TimeInForce defines timeInForce types
type TimeInForce string

// All order status types
const (
	GoodTilCancelled         TimeInForce = "GOOD_TIL_CANCELLED"
	ImmediateOrCancel        TimeInForce = "IMMEDIATE_OR_CANCEL"
	FillOrKill               TimeInForce = "FILL_OR_KILL"
	PostOnlyGoodTilCancelled TimeInForce = "POST_ONLY_GOOD_TIL_CANCELLED"
	BuyNow                   TimeInForce = "BUY_NOW"
)

// OrderData holds order data
type OrderData struct {
	ID            string    `json:"id"`
	MarketSymbol  string    `json:"marketSymbol"`
	Direction     string    `json:"direction"`
	Type          string    `json:"type"`
	Quantity      float64   `json:"quantity,string"`
	Limit         float64   `json:"limit,string"`
	Ceiling       float64   `json:"ceiling,string"`
	TimeInForce   string    `json:"timeInForce"`
	ClientOrderID string    `json:"clientOrderId"`
	FillQuantity  float64   `json:"fillQuantity,string"`
	Commission    float64   `json:"commission,string"`
	Proceeds      float64   `json:"proceeds,string"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	ClosedAt      time.Time `json:"closedAt"`
	OrderToCancel struct {
		Type string `json:"type,string"`
		ID   string `json:"id,string"`
	} `json:"orderToCancel"`
}

// BulkCancelResultData holds the result of a bulk cancel action
type BulkCancelResultData struct {
	ID         string    `json:"id"`
	StatusCode string    `json:"statusCode"`
	Result     OrderData `json:"result"`
}

// MarketData stores market data
type MarketData struct {
	Symbol              string    `json:"symbol"`
	BaseCurrencySymbol  string    `json:"baseCurrencySymbol"`
	QuoteCurrencySymbol string    `json:"quoteCurrencySymbol"`
	MinTradeSize        float64   `json:"minTradeSize,string"`
	Precision           int32     `json:"precision"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"createdAt"`
	Notice              string    `json:"notice"`
	ProhibitedIn        []string  `json:"prohibitedIn"`
}

// TickerData stores ticker data
type TickerData struct {
	Symbol        string  `json:"symbol"`
	LastTradeRate float64 `json:"lastTradeRate,string"`
	BidRate       float64 `json:"bidRate,string"`
	AskRate       float64 `json:"askRate,string"`
}

// TradeData stores trades data
type TradeData struct {
	ID         string    `json:"id"`
	ExecutedAt time.Time `json:"executedAt"`
	Quantity   float64   `json:"quantity,string"`
	Rate       float64   `json:"rate,string"`
	TakerSide  string    `json:"takerSide"`
}

// MarketSummaryData stores market summary data
type MarketSummaryData struct {
	Symbol        string    `json:"symbol"`
	High          float64   `json:"high,string"`
	Low           float64   `json:"low,string"`
	Volume        float64   `json:"volume,string"`
	QuoteVolume   float64   `json:"quoteVolume,string"`
	PercentChange float64   `json:"percentChange,string"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// OrderbookData holds the order book data
type OrderbookData struct {
	Bid []OrderbookEntryData `json:"bid"`
	Ask []OrderbookEntryData `json:"ask"`
}

// OrderbookEntryData holds an order book entry
type OrderbookEntryData struct {
	Quantity float64 `json:"quantity,string"`
	Rate     float64 `json:"rate,string"`
}

// BalanceData holds balance data
type BalanceData struct {
	CurrencySymbol string    `json:"currencySymbol"`
	Total          float64   `json:"total,string"`
	Available      float64   `json:"available,string"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// AddressData holds address data
// Status is REQUESTED or PROVISIONED
type AddressData struct {
	Status           string `json:"status"`
	CurrencySymbol   string `json:"currencySymbol"`
	CryptoAddress    string `json:"cryptoAddress"`
	CryptoAddressTag string `json:"cryptoAddressTag"`
}

// CurrencyData holds currency data
// Status is ONLINE or OFFLINE
type CurrencyData struct {
	Symbol           string   `json:"symbol"`
	Name             string   `json:"name"`
	CoinType         string   `json:"coinType"`
	Status           string   `json:"status"`
	MinConfirmations int32    `json:"minConfirmations"`
	Notice           string   `json:"notice"`
	TxFee            float64  `json:"txFee,string"`
	LogoURL          string   `json:"logoUrl"`
	ProhibitedIn     []string `json:"prohibitedIn"`
}

// WithdrawalData holds withdrawal data
type WithdrawalData struct {
	ID                 string    `json:"id"`
	CurrencySymbol     string    `json:"currencySymbol"`
	Quantity           float64   `json:"quantity,string"`
	CryptoAddress      string    `json:"cryptoAddress"`
	CryptoAddressTag   string    `json:"cryptoAddressTag"`
	TxCost             float64   `json:"txCost,string"`
	TxID               string    `json:"txId"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"createdAt"`
	CompletedAt        time.Time `json:"completedAt"`
	ClientWithdrawalID string    `json:"clientWithdrawalId"`
}

// DepositData holds deposit data
type DepositData struct {
	ID               string    `json:"id"`
	CurrencySymbol   string    `json:"currencySymbol"`
	Quantity         float64   `json:"quantity,string"`
	CryptoAddress    string    `json:"cryptoAddress"`
	CryptoAddressTag string    `json:"cryptoAddressTag"`
	TxID             string    `json:"txId"`
	Confirmations    int32     `json:"confirmations"`
	UpdatedAt        time.Time `json:"updatedAt"`
	CompletedAt      time.Time `json:"completedAt"`
	Status           string    `json:"status"`
	Source           string    `json:"source"`
}

// CandleData holds candle data
type CandleData struct {
	StartsAt    time.Time `json:"startsAt"`
	Open        float64   `json:"open,string"`
	High        float64   `json:"high,string"`
	Low         float64   `json:"low,string"`
	Close       float64   `json:"close,string"`
	Volume      float64   `json:"volume,string"`
	QuoteVolume float64   `json:"quoteVolume,string"`
}

// WsSignalRHandshakeData holds data for the SignalR websocket wrapper handshake
type WsSignalRHandshakeData struct {
	URL                     string  `json:"Url"`                     // Path to the SignalR endpoint
	ConnectionToken         string  `json:"ConnectionToken"`         // Connection token assigned by the server
	ConnectionID            string  `json:"ConnectionId"`            // The ID of the connection
	KeepAliveTimeout        float64 `json:"KeepAliveTimeout"`        // Representing the amount of time to wait before sending a keep alive packet over an idle connection
	DisconnectTimeout       float64 `json:"DisconnectTimeout"`       // Represents the amount of time to wait after a connection goes away before raising the disconnect event
	ConnectionTimeout       float64 `json:"ConnectionTimeout"`       // Represents the amount of time to leave a connection open before timing out
	TryWebSockets           bool    `json:"TryWebSockets"`           // Whether the server supports websockets
	ProtocolVersion         string  `json:"ProtocolVersion"`         // The version of the protocol used for communication
	TransportConnectTimeout float64 `json:"TransportConnectTimeout"` // The maximum amount of time the client should try to connect to the server using a given transport
	LongPollDelay           float64 `json:"LongPollDelay"`           // The time to tell the browser to wait before reestablishing a long poll connection after data is sent from the server.
}

// WsEventRequest holds data on websocket requests
type WsEventRequest struct {
	Hub          string      `json:"H"`
	Method       string      `json:"M"`
	Arguments    interface{} `json:"A"`
	InvocationID int64       `json:"I"`
}

// WsEventStatus holds data on the websocket event status
type WsEventStatus struct {
	Success   bool   `json:"Success"`
	ErrorCode string `json:"ErrorCode"`
}

// WsEventResponse holds data on the websocket response
type WsEventResponse struct {
	C            string      `json:"C"`
	S            int         `json:"S"`
	G            string      `json:"G"`
	Response     interface{} `json:"R"`
	InvocationID int64       `json:"I,string"`
	Message      []struct {
		Hub       string   `json:"H"`
		Method    string   `json:"M"`
		Arguments []string `json:"A"`
	} `json:"M"`
}

// WsSubscriptionResponse holds data on the websocket response
type WsSubscriptionResponse struct {
	C            string          `json:"C"`
	S            int             `json:"S"`
	G            string          `json:"G"`
	Response     []WsEventStatus `json:"R"`
	InvocationID int64           `json:"I,string"`
	Message      []struct {
		Hub       string   `json:"H"`
		Method    string   `json:"M"`
		Arguments []string `json:"A"`
	} `json:"M"`
}

// WsAuthResponse holds data on the websocket response
type WsAuthResponse struct {
	C            string        `json:"C"`
	S            int           `json:"S"`
	G            string        `json:"G"`
	Response     WsEventStatus `json:"R"`
	InvocationID int64         `json:"I,string"`
	Message      []struct {
		Hub       string   `json:"H"`
		Method    string   `json:"M"`
		Arguments []string `json:"A"`
	} `json:"M"`
}

// OrderbookUpdateMessage holds websocket orderbook update messages
type OrderbookUpdateMessage struct {
	MarketSymbol string               `json:"marketSymbol"`
	Depth        int                  `json:"depth"`
	Sequence     int64                `json:"sequence"`
	BidDeltas    []OrderbookEntryData `json:"bidDeltas"`
	AskDeltas    []OrderbookEntryData `json:"askDeltas"`
}

// OrderUpdateMessage holds websocket order update messages
type OrderUpdateMessage struct {
	AccountID string    `json:"accountId"`
	Sequence  int       `json:"int,string"`
	Delta     OrderData `json:"delta"`
}

// WsPendingRequest holds pending requests
type WsPendingRequest struct {
	WsEventRequest
	ChannelsToSubscribe *[]stream.ChannelSubscription
}

// orderbookManager defines a way of managing and maintaining synchronisation
// across connections and assets.
type orderbookManager struct {
	state map[currency.Code]map[currency.Code]map[asset.Item]*update
	sync.Mutex

	jobs chan job
}

type update struct {
	buffer       chan *OrderbookUpdateMessage
	fetchingBook bool
	initialSync  bool
}

// job defines a synchonisation job that tells a go routine to fetch an
// orderbook via the REST protocol
type job struct {
	Pair currency.Pair
}
