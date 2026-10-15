package htx

import (
	"bytes"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WsTradeID preserves large integer identifiers without floating-point conversion.
type WsTradeID string

// UnmarshalJSON accepts the numeric and quoted identifier representations used by HTX feeds.
func (i *WsTradeID) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return errInvalidTradeID
	}
	if data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		*i = WsTradeID(value)
		return nil
	}
	for _, character := range data {
		if character < '0' || character > '9' {
			return fmt.Errorf("%w %q", errInvalidTradeID, data)
		}
	}
	*i = WsTradeID(data)
	return nil
}

// wsSubReq is a request to subscribe to or unubscribe from a topic for public channels (private channels use generic wsReq)
type wsSubReq struct {
	ID    string `json:"id,omitempty"`
	Sub   string `json:"sub,omitempty"`
	Unsub string `json:"unsub,omitempty"`
}

// WsHeartBeat defines a heartbeat request
type WsHeartBeat struct {
	ClientNonce int64 `json:"ping"`
}

// WsDepth defines market depth websocket response
type WsDepth struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		Bids      [][]any    `json:"bids"`
		Asks      [][]any    `json:"asks"`
		Timestamp types.Time `json:"ts"`
		Version   int64      `json:"version"`
	} `json:"tick"`
}

// WsKline defines market kline websocket response
type WsKline struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		ID     int64   `json:"id"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		Low    float64 `json:"low"`
		High   float64 `json:"high"`
		Amount float64 `json:"amount"`
		Volume float64 `json:"vol"`
		Count  int64   `json:"count"`
	} `json:"tick"`
}

// WsTick stores websocket ticker data
type WsTick struct {
	Channel   string     `json:"ch"`
	Rep       string     `json:"rep"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
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
	} `json:"tick"`
}

// WsTrade defines market trade websocket response
type WsTrade struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		ID        int64      `json:"id"`
		Timestamp types.Time `json:"ts"`
		Data      []struct {
			Amount    float64    `json:"amount"`
			Timestamp types.Time `json:"ts"`
			ID        WsTradeID  `json:"id"`
			TradeID   WsTradeID  `json:"tradeId"`
			Price     float64    `json:"price"`
			Direction string     `json:"direction"`
		} `json:"data"`
	}
}
