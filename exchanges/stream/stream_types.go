package stream

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Streamer defines functionality for different exchange streaming services
type Streamer interface {
	Setup()
	Connect() error
	Disconnect() error
	GenerateAuthSubscriptions()
	GenerateMarketDataSubscriptions()
	Subscribe()
	UnSubscribe()
	Refresh()
	GetName() string
	SetProxyAddress(string) error
	GetProxy() string
}

// Connection defines a streaming services connection
type Connection interface {
	Dial(*websocket.Dialer, http.Header) error
	ReadMessage() (Response, error)
	SendJSONMessage(interface{}) error
	SetupPingHandler(WebsocketPingHandler)
	IsIDWaitingForResponse(id int64) bool
	SetResponseIDAndData(id int64, data []byte)
	GenerateMessageID(useNano bool) int64
	SendMessageReturnResponse(id int64, request interface{}) ([]byte, error)
	SendRawMessage(messageType int, message []byte) error
}

// Response defines generalised data from the stream connection
type Response struct {
	Type int
	Raw  []byte
}

// type Manager interface {
// 	Connector
// 	Disconnector
// }

// type Connector interface{}

// type Disconnector interface{}

// // Websocket defines a websocket connection
// type Websocket struct{}

// WebsocketPingHandler container for ping handler settings
type WebsocketPingHandler struct {
	UseGorillaHandler bool
	MessageType       int
	Message           []byte
	Delay             time.Duration
}

// TradeData defines trade data
type TradeData struct {
	Timestamp    time.Time
	CurrencyPair currency.Pair
	AssetType    asset.Item
	Exchange     string
	EventType    order.Type
	Price        float64
	Amount       float64
	Side         order.Side
}

// FundingData defines funding data
type FundingData struct {
	Timestamp    time.Time
	CurrencyPair currency.Pair
	AssetType    asset.Item
	Exchange     string
	Amount       float64
	Rate         float64
	Period       int64
	Side         order.Side
}

// KlineData defines kline feed
type KlineData struct {
	Timestamp  time.Time
	Pair       currency.Pair
	AssetType  asset.Item
	Exchange   string
	StartTime  time.Time
	CloseTime  time.Time
	Interval   string
	OpenPrice  float64
	ClosePrice float64
	HighPrice  float64
	LowPrice   float64
	Volume     float64
}

// WebsocketPositionUpdated reflects a change in orders/contracts on an exchange
type WebsocketPositionUpdated struct {
	Timestamp time.Time
	Pair      currency.Pair
	AssetType asset.Item
	Exchange  string
}
