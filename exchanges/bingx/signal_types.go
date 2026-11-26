package bingx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CreateSignalRequest holds the parameters of Create Signal, a USDT-M perpetual futures signal, and is sent as its JSON
// body; Symbol, Side and EntryPrice are required
type CreateSignalRequest struct {
	Symbol     currency.Pair `json:"symbol"`
	Side       SignalSide    `json:"side"`
	EntryPrice types.Number  `json:"entryPrice"`
	// TakeProfitPrice is a single price
	TakeProfitPrice types.Number `json:"tpPrice,omitempty"`
	StopLossPrice   types.Number `json:"slPrice,omitempty"`
	// OrderType is the type of the take profit and stop loss orders, LIMIT or MARKET; BingX defaults to LIMIT
	OrderType OrderType `json:"orderType,omitempty"`
	// Leverage is 20 when zero
	Leverage uint64 `json:"leverage,omitempty"`
	// SizePercentage is the percentage of the available balance to use, such as 10 for 10%
	SizePercentage uint64 `json:"sizePct,omitempty"`
}

// CreateSignalResponse holds a created signal
type CreateSignalResponse struct {
	SignalID string `json:"signalId"`
	// ShortURL is the signal's sharing link
	ShortURL string `json:"shortUrl"`
}
