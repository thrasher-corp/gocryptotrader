package lbank

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// ContractInstrumentResponse holds a contract's specification
type ContractInstrumentResponse struct {
	Symbol      string `json:"symbol"`
	SymbolName  string `json:"symbolName"`
	SymbolAlias string `json:"symbolAlias"`
	ExchangeID  string `json:"exchangeID"`
	// BaseCurrency is the contract's underlying, PriceCurrency the currency it is priced in and ClearCurrency the
	// currency it settles in
	BaseCurrency       currency.Code `json:"baseCurrency"`
	PriceCurrency      currency.Code `json:"priceCurrency"`
	ClearCurrency      currency.Code `json:"clearCurrency"`
	PriceTick          float64       `json:"priceTick"`
	VolumeTick         float64       `json:"volumeTick"`
	VolumeMultiple     float64       `json:"volumeMultiple"`
	MinimumOrderVolume types.Number  `json:"minOrderVolume"`
	MaximumOrderVolume types.Number  `json:"maxOrderVolume"`
	MinimumOrderCost   types.Number  `json:"minOrderCost"`
	// PriceLimitUpperValue and PriceLimitLowerValue bound an order's price as a fraction above and below the index
	// price: 0.05 is 5%
	PriceLimitUpperValue float64       `json:"priceLimitUpperValue"`
	PriceLimitLowerValue float64       `json:"priceLimitLowerValue"`
	IndexPrice           types.Number  `json:"indexPrice"`
	DefaultLeverage      float64       `json:"defaultLeverage"`
	MaximumLeverage      float64       `json:"maxLeverage"`
	NeedSuspend          types.Boolean `json:"needSuspend"`
}

// ContractMarketDataResponse holds a contract's 24 hour statistics, mark price and funding rate
type ContractMarketDataResponse struct {
	Symbol           string       `json:"symbol"`
	InstrumentStatus string       `json:"instrumentStatus"`
	LastPrice        types.Number `json:"lastPrice"`
	MarkPrice        types.Number `json:"markedPrice"`
	UnderlyingPrice  types.Number `json:"underlyingPrice"`
	OpenPrice24Hour  types.Number `json:"openPrice"`
	HighPrice24Hour  types.Number `json:"highestPrice"`
	LowPrice24Hour   types.Number `json:"lowestPrice"`
	Volume24Hour     types.Number `json:"volume"`
	Turnover24Hour   types.Number `json:"turnover"`
	// FundingRate and PositionFeeRate carry the same rate in live replies
	FundingRate     types.PreciseNumber `json:"fundingRate"`
	PositionFeeRate types.PreciseNumber `json:"positionFeeRate"`
	// PreviousPositionFeeRate is documented as the funding fee but is absent from live replies
	PreviousPositionFeeRate types.PreciseNumber `json:"prePositionFeeRate"`
	// FundingIntervalSeconds is the time between the contract's funding settlements in seconds
	FundingIntervalSeconds uint64     `json:"positionFeeTime"`
	NextFundingTime        types.Time `json:"nextFeeTime"`
	LastTime               types.Time `json:"lastTime"`
}

// ContractOrderbookResponse holds a contract's order book
type ContractOrderbookResponse struct {
	Symbol string                   `json:"symbol"`
	Asks   []ContractOrderbookLevel `json:"asks"`
	Bids   []ContractOrderbookLevel `json:"bids"`
}

// ContractOrderbookLevel holds an order book price level
type ContractOrderbookLevel struct {
	Price      types.Number `json:"price"`
	Volume     types.Number `json:"volume"`
	OrderCount types.Number `json:"orders"`
}
