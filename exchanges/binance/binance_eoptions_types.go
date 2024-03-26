package binance

import (
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// EOptionExchangeInfo represents an exchange information.
type EOptionExchangeInfo struct {
	Timezone        string               `json:"timezone"`
	ServerTime      convert.ExchangeTime `json:"serverTime"`
	OptionContracts []struct {
		ID          int64  `json:"id"`
		BaseAsset   string `json:"baseAsset"`
		QuoteAsset  string `json:"quoteAsset"`
		Underlying  string `json:"underlying"`
		SettleAsset string `json:"settleAsset"`
	} `json:"optionContracts"`
	OptionAssets []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"optionAssets"`
	OptionSymbols []struct {
		ContractID int64                `json:"contractId"`
		ExpiryDate convert.ExchangeTime `json:"expiryDate"`
		Filters    []struct {
			FilterType string       `json:"filterType"`
			MinPrice   types.Number `json:"minPrice,omitempty"`
			MaxPrice   types.Number `json:"maxPrice,omitempty"`
			TickSize   types.Number `json:"tickSize,omitempty"`
			MinQty     types.Number `json:"minQty,omitempty"`
			MaxQty     types.Number `json:"maxQty,omitempty"`
			StepSize   types.Number `json:"stepSize,omitempty"`
		} `json:"filters"`
		ID                   int64        `json:"id"`
		Symbol               string       `json:"symbol"`
		Side                 string       `json:"side"`
		StrikePrice          types.Number `json:"strikePrice"`
		Underlying           string       `json:"underlying"`
		Unit                 int          `json:"unit"`
		MakerFeeRate         string       `json:"makerFeeRate"`
		TakerFeeRate         string       `json:"takerFeeRate"`
		MinQty               string       `json:"minQty"`
		MaxQty               string       `json:"maxQty"`
		InitialMargin        string       `json:"initialMargin"`
		MaintenanceMargin    string       `json:"maintenanceMargin"`
		MinInitialMargin     string       `json:"minInitialMargin"`
		MinMaintenanceMargin string       `json:"minMaintenanceMargin"`
		PriceScale           float64      `json:"priceScale"`
		QuantityScale        float64      `json:"quantityScale"`
		QuoteAsset           string       `json:"quoteAsset"`
	} `json:"optionSymbols"`
	RateLimits []struct {
		RateLimitType string `json:"rateLimitType"`
		Interval      string `json:"interval"`
		IntervalNum   int64  `json:"intervalNum"`
		Limit         int64  `json:"limit"`
	} `json:"rateLimits"`
}

// EOptionsOrderbook represents an european orderbook option information.
type EOptionsOrderbook struct {
	TransactionTime convert.ExchangeTime `json:"T"`
	UpdateID        int64                `json:"u"`
	Asks            [][2]types.Number    `json:"asks"`
	Bids            [][2]types.Number    `json:"bids"` // [][Price, Quantity]
}

// EOptionsTradeItem represents a recent trade information
type EOptionsTradeItem struct {
	ID       string               `json:"id"`
	Symbol   string               `json:"symbol"`
	Price    types.Number         `json:"price"`
	Quantity types.Number         `json:"qty"`
	QuoteQty types.Number         `json:"quoteQty"`
	Side     tradeSide            `json:"side"` // Completed trade direction（-1 Sell，1 Buy）
	Time     convert.ExchangeTime `json:"time"`
}

type tradeSide int64

const (
	sellSide tradeSide = -1
	buySide  tradeSide = 1
)

// String returns a string representation of side value in eoptions recent trades.
func (s tradeSide) String() string {
	switch s {
	case sellSide:
		return "Sell"
	case buySide:
		return "Buy"
	default:
		return ""
	}
}

// EOptionsCandlestick represents candlestick bar for options symbols.
type EOptionsCandlestick struct {
	Open        types.Number         `json:"open"`
	High        types.Number         `json:"high"`
	Low         types.Number         `json:"low"`
	Close       types.Number         `json:"close"`
	Volume      types.Number         `json:"volume"`
	Amount      types.Number         `json:"amount"`
	Interval    string               `json:"interval"`
	TradeCount  int64                `json:"tradeCount"`
	TakerVolume types.Number         `json:"takerVolume"`
	TakerAmount types.Number         `json:"takerAmount"`
	OpenTime    convert.ExchangeTime `json:"openTime"`
	CloseTime   convert.ExchangeTime `json:"closeTime"`
}

// OptionMarkPrice represents an option mark price
type OptionMarkPrice struct {
	Symbol         string       `json:"symbol"`
	MarkPrice      types.Number `json:"markPrice"`
	BidIV          types.Number `json:"bidIV"`
	AskIV          types.Number `json:"askIV"`
	MarkIV         types.Number `json:"markIV"`
	Delta          types.Number `json:"delta"`
	Theta          types.Number `json:"theta"`
	Gamma          types.Number `json:"gamma"`
	Vega           types.Number `json:"vega"`
	HighPriceLimit types.Number `json:"highPriceLimit"`
	LowPriceLimit  types.Number `json:"lowPriceLimit"`
}

// EOptionTicker represents a ticker information for an european option instance.
type EOptionTicker struct {
	Symbol             string               `json:"symbol"`
	PriceChange        types.Number         `json:"priceChange"`
	PriceChangePercent types.Number         `json:"priceChangePercent"`
	LastPrice          types.Number         `json:"lastPrice"`
	LastQty            types.Number         `json:"lastQty"`
	Open               types.Number         `json:"open"`
	High               types.Number         `json:"high"`
	Low                types.Number         `json:"low"`
	Volume             types.Number         `json:"volume"`
	Amount             types.Number         `json:"amount"`
	BidPrice           types.Number         `json:"bidPrice"`
	AskPrice           types.Number         `json:"askPrice"`
	OpenTime           convert.ExchangeTime `json:"openTime"`
	CloseTime          convert.ExchangeTime `json:"closeTime"`
	FirstTradeID       int64                `json:"firstTradeId"`
	TradeCount         int64                `json:"tradeCount"`
	StrikePrice        types.Number         `json:"strikePrice"`
	ExercisePrice      types.Number         `json:"exercisePrice"`
}

// EOptionIndexSymbolPriceTicker represents spot index price
type EOptionIndexSymbolPriceTicker struct {
	Time       convert.ExchangeTime `json:"time"`
	IndexPrice string               `json:"indexPrice"`
}

// ExerciseHistoryItem represents an exercise history
// REALISTIC_VALUE_STRICKEN -> Exercised
// EXTRINSIC_VALUE_EXPIRED -> Expired OTM
type ExerciseHistoryItem struct {
	Symbol          string       `json:"symbol"`
	StrikePrice     types.Number `json:"strikePrice"`
	RealStrikePrice types.Number `json:"realStrikePrice"`
	ExpiryDate      int64        `json:"expiryDate"`
	StrikeResult    string       `json:"strikeResult"`
}

// OpenInterest represents an instance open interest
type OpenInterest struct {
	Symbol             string               `json:"symbol"`
	SumOpenInterest    types.Number         `json:"sumOpenInterest"`
	SumOpenInterestUSD types.Number         `json:"sumOpenInterestUsd"`
	Timestamp          convert.ExchangeTime `json:"timestamp"`
}
