package binance

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// RebateHistoryResponse holds a page of spot rebates
type RebateHistoryResponse struct {
	Status string            `json:"status"`
	Type   string            `json:"type"`
	Code   string            `json:"code"`
	Data   RebateHistoryPage `json:"data"`
}

// RebateHistoryPage is a page of spot rebates
type RebateHistoryPage struct {
	Page            uint64         `json:"page"`
	TotalRecords    uint64         `json:"totalRecords"`
	TotalPageNumber uint64         `json:"totalPageNum"`
	Data            []RebateRecord `json:"data"`
}

// RebateRecord is a spot rebate
type RebateRecord struct {
	Asset currency.Code `json:"asset"`
	// Type is 1 for a commission rebate and 2 for a referral kickback
	Type       uint64       `json:"type"`
	Amount     types.Number `json:"amount"`
	UpdateTime types.Time   `json:"updateTime"`
}
