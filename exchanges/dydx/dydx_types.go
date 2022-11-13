package dydx

import "time"

// InstrumentDatas metadata about each retrieved market.
type InstrumentDatas struct {
	Markets map[string]struct {
		Market                           string    `json:"market"`
		Status                           string    `json:"status"`
		BaseAsset                        string    `json:"baseAsset"`
		QuoteAsset                       string    `json:"quoteAsset"`
		StepSize                         string    `json:"stepSize"`
		TickSize                         string    `json:"tickSize"`
		IndexPrice                       string    `json:"indexPrice"`
		OraclePrice                      string    `json:"oraclePrice"`
		PriceChange24H                   string    `json:"priceChange24H"`
		NextFundingRate                  string    `json:"nextFundingRate"`
		NextFundingAt                    time.Time `json:"nextFundingAt"`
		MinOrderSize                     string    `json:"minOrderSize"`
		Type                             string    `json:"type"`
		InitialMarginFraction            string    `json:"initialMarginFraction"`
		MaintenanceMarginFraction        string    `json:"maintenanceMarginFraction"`
		BaselinePositionSize             string    `json:"baselinePositionSize"`
		IncrementalPositionSize          string    `json:"incrementalPositionSize"`
		IncrementalInitialMarginFraction string    `json:"incrementalInitialMarginFraction"`
		Volume24H                        string    `json:"volume24H"`
		Trades24H                        string    `json:"trades24H"`
		OpenInterest                     string    `json:"openInterest"`
		MaxPositionSize                  string    `json:"maxPositionSize"`
		AssetResolution                  string    `json:"assetResolution"`
		SyntheticAssetID                 string    `json:"syntheticAssetId"`
	} `json:"markets"`
}

// MarketOrderbook represents  bids and asks that are fillable are returned.
type MarketOrderbook struct {
	Bids []OrderbookData `json:"bids"`
	Asks []OrderbookData `json:"asks"`
}

// OrderbookData represents asks and bids price and size data.
type OrderbookData struct {
	Price float64 `json:"price,string"`
	Size  float64 `json:"size,string"`
}

// MarketTrades represents trade informations for specific market(instrument).
type MarketTrades struct {
	Trades []MarketTrade `json:"trades"`
}

// MarketTrade represents a market trade item.
type MarketTrade struct {
	Side        string    `json:"side"`
	Size        float64   `json:"size,string"`
	Price       float64   `json:"price,string"`
	CreatedAt   time.Time `json:"createdAt"`
	Liquidation bool      `json:"liquidation"`
}
