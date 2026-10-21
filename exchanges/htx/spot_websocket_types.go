package htx

import (
	"bytes"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WebsocketTradeID preserves large integer identifiers without floating-point conversion.
type WebsocketTradeID string

// UnmarshalJSON accepts the numeric and quoted identifier representations used by HTX feeds.
func (i *WebsocketTradeID) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return errInvalidTradeID
	}
	if data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		*i = WebsocketTradeID(value)
		return nil
	}
	for _, character := range data {
		if character < '0' || character > '9' {
			return fmt.Errorf("%w %q", errInvalidTradeID, data)
		}
	}
	*i = WebsocketTradeID(data)
	return nil
}

// websocketSubscriptionRequest is a request to subscribe to or unsubscribe from a topic for public channels (private channels use generic websocketRequest)
type websocketSubscriptionRequest struct {
	ID    string `json:"id,omitempty"`
	Sub   string `json:"sub,omitempty"`
	Unsub string `json:"unsub,omitempty"`
}

// WebsocketHeartBeat defines a heartbeat request
type WebsocketHeartBeat struct {
	ClientNonce int64 `json:"ping"`
}

// WebsocketDepth defines market depth websocket response
type WebsocketDepth struct {
	Channel   string             `json:"ch"`
	Timestamp types.Time         `json:"ts"`
	Tick      WebsocketDepthTick `json:"tick"`
}

// WebsocketKline defines market kline websocket response
type WebsocketKline struct {
	Channel   string             `json:"ch"`
	Timestamp types.Time         `json:"ts"`
	Tick      WebsocketKlineTick `json:"tick"`
}

// WebsocketTick stores websocket ticker data
type WebsocketTick struct {
	Channel   string              `json:"ch"`
	Rep       string              `json:"rep"`
	Timestamp types.Time          `json:"ts"`
	Tick      WebsocketTickerTick `json:"tick"`
}

// WebsocketTrade defines market trade websocket response
type WebsocketTrade struct {
	Channel   string             `json:"ch"`
	Timestamp types.Time         `json:"ts"`
	Tick      WebsocketTradeTick `json:"tick"`
}

// WebsocketDepthTick is a spot depth snapshot with its exchange timestamp and version.
type WebsocketDepthTick struct {
	Bids      [][]any    `json:"bids"`
	Asks      [][]any    `json:"asks"`
	Timestamp types.Time `json:"ts"`
	Version   uint64     `json:"version"`
}

// WebsocketKlineTick is a spot candle; ID is the start time in Unix seconds.
type WebsocketKlineTick struct {
	ID     uint64  `json:"id"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	Low    float64 `json:"low"`
	High   float64 `json:"high"`
	Amount float64 `json:"amount"`
	Volume float64 `json:"vol"`
	Count  uint64  `json:"count"`
}

// WebsocketTickerTick is a rolling market summary that also carries the best bid and ask.
type WebsocketTickerTick struct {
	Bid           []float64  `json:"bid"`
	Ask           []float64  `json:"ask"`
	Amount        float64    `json:"amount"`
	Close         float64    `json:"close"`
	Count         float64    `json:"count"`
	High          float64    `json:"high"`
	ID            float64    `json:"id"`
	Low           float64    `json:"low"`
	Open          float64    `json:"open"`
	Timestamp     types.Time `json:"ts"`
	Volume        float64    `json:"vol"`
	TradeTurnover float64    `json:"trade_turnover"`
}

// WebsocketTradeTick groups executions published together under one exchange update ID.
type WebsocketTradeTick struct {
	ID        uint64                    `json:"id"`
	Timestamp types.Time                `json:"ts"`
	Data      []WebsocketTradeExecution `json:"data"`
}

// WebsocketTradeExecution accepts either id or tradeId so execution identifiers retain their exact digits.
type WebsocketTradeExecution struct {
	Amount    float64          `json:"amount"`
	Timestamp types.Time       `json:"ts"`
	ID        WebsocketTradeID `json:"id"`
	TradeID   WebsocketTradeID `json:"tradeId"`
	Price     float64          `json:"price"`
	Direction string           `json:"direction"`
}
