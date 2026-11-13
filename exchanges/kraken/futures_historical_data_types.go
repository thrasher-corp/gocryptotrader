package kraken

import (
	"time"
)

// FuturesFillsRequest holds the parameters of Get your fills
type FuturesFillsRequest struct {
	// LastFillTime returns the 100 fills before it, for paging back with the previous response's last FillTime
	LastFillTime time.Time
}

// FuturesFillsResponse holds the account's fills, newest first
type FuturesFillsResponse struct {
	Fills      []FuturesFill `json:"fills"`
	ServerTime time.Time     `json:"serverTime"`
}

// FuturesFill is a fill of one of the account's orders
type FuturesFill struct {
	FillID        string    `json:"fill_id"`
	OrderID       string    `json:"order_id"`
	ClientOrderID string    `json:"cliOrdId"`
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"`
	Size          float64   `json:"size"`
	Price         float64   `json:"price"`
	FillTime      time.Time `json:"fillTime"`
	// FillType is maker, taker, liquidation, partialLiquidation, assignor, assignee, takerAfterEdit, unwindBankrupt or
	// unwindCounterparty
	FillType string `json:"fillType"`
	// RealisedPNL is signed from the account's perspective and zero for a fill opening or increasing a position. Kraken
	// sends none for fills recorded before it introduced the field, or for a request with LastFillTime
	RealisedPNL float64 `json:"realized_pnl"`
	// SequenceID numbers a market's executions from 1 without gaps, for gap detection; zero for fills recorded before
	// Kraken introduced it
	SequenceID uint64 `json:"sequence_id,string"`
}
