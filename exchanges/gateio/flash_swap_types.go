package gateio

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FlashSwapCurrencyPairResponse represents a supported flash swap pair.
type FlashSwapCurrencyPairResponse struct {
	CurrencyPair  currency.Pair `json:"currency_pair"`
	SellCurrency  currency.Code `json:"sell_currency"`
	BuyCurrency   currency.Code `json:"buy_currency"`
	SellMinAmount types.Number  `json:"sell_min_amount"`
	SellMaxAmount types.Number  `json:"sell_max_amount"`
	BuyMinAmount  types.Number  `json:"buy_min_amount"`
	BuyMaxAmount  types.Number  `json:"buy_max_amount"`
}

// FlashSwapOrderRequest represents create flash swap order request parameters.
type FlashSwapOrderRequest struct {
	PreviewID    string        `json:"preview_id"`
	SellCurrency currency.Code `json:"sell_currency"`
	SellAmount   types.Number  `json:"sell_amount,omitempty"`
	BuyCurrency  currency.Code `json:"buy_currency"`
	BuyAmount    types.Number  `json:"buy_amount,omitempty"`
}

// FlashSwapPreviewRequest holds a flash swap to preview; set either SellAmount or BuyAmount
type FlashSwapPreviewRequest struct {
	SellCurrency currency.Code `json:"sell_currency"`
	SellAmount   types.Number  `json:"sell_amount,omitempty"`
	BuyCurrency  currency.Code `json:"buy_currency"`
	BuyAmount    types.Number  `json:"buy_amount,omitempty"`
}

// FlashSwapOrderResponse represents create flash swap order response
type FlashSwapOrderResponse struct {
	ID           uint64        `json:"id"`
	CreateTime   types.Time    `json:"create_time"`
	UpdateTime   types.Time    `json:"update_time"`
	UserID       uint64        `json:"user_id"`
	SellCurrency currency.Code `json:"sell_currency"`
	SellAmount   types.Number  `json:"sell_amount"`
	BuyCurrency  currency.Code `json:"buy_currency"`
	BuyAmount    types.Number  `json:"buy_amount"`
	Price        types.Number  `json:"price"`
	Status       uint64        `json:"status"`
}

// InitFlashSwapOrderPreviewResponse represents the order preview for flash order
type InitFlashSwapOrderPreviewResponse struct {
	PreviewID    string        `json:"preview_id"`
	SellCurrency currency.Code `json:"sell_currency"`
	SellAmount   types.Number  `json:"sell_amount"`
	BuyCurrency  currency.Code `json:"buy_currency"`
	BuyAmount    types.Number  `json:"buy_amount"`
	Price        types.Number  `json:"price"`
}
