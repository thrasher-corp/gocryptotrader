package ftx

// MarketData stores market data
type MarketData struct {
	Name           string  `json:"name"`
	BaseCurrency   string  `json:"baseCurrency"`
	QuoteCurrency  string  `json:"quoteCurrency"`
	MarketType     string  `json:"type"`
	Underlying     string  `json:"underlying"`
	Enabled        bool    `json:"enabled"`
	Ask            float64 `json:"ask"`
	Bid            float64 `json:"bid"`
	Last           float64 `json:"last"`
	PriceIncrement float64 `json:"priceIncrement"`
	SizeIncrement  float64 `json:"sizeIncrement"`
}

// Markets stores all markets data
type Markets struct {
	Success bool         `json:"success"`
	Result  []MarketData `json:"result"`
}

// Market stores data for a given market
type Market struct {
	Success bool       `json:"success"`
	Result  MarketData `json:"result"`
}

// OrderData stores orderdata in orderbook
type OrderData struct {
	Price float64
	Size  float64
}

// OrderbookData stores orderbook data
type OrderbookData struct {
	MarketName string
	Asks       []OrderData
	Bids       []OrderData
}

// TempOrderbook stores order book
type TempOrderbook struct {
	Success bool       `json:"success"`
	Result  TempOBData `json:"result"`
}

// TempOBData stores orderbook data temporarily
type TempOBData struct {
	Asks [][2]float64 `json:"asks"`
	Bids [][2]float64 `json:"bids"`
}

// TradeData stores data from trades
type TradeData struct {
	ID          int64   `json:"id"`
	Liquidation bool    `json:"liquidation"`
	Price       float64 `json:"price"`
	Side        string  `json:"side"`
	Size        float64 `json:"size"`
	Time        string  `json:"time"`
}

// Trades stores data for multiple trades
type Trades struct {
	Success bool        `json:"success"`
	Result  []TradeData `json:"result"`
}

// OHLCVData stores historical OHLCV data
type OHLCVData struct {
	Close     float64 `json:"close"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Open      float64 `json:"open"`
	StartTime string  `json:"startTime"`
	Volume    float64 `json:"volume"`
}

// HistoricalData stores historical OHLCVData
type HistoricalData struct {
	Success bool        `json:"success"`
	Result  []OHLCVData `json:"result"`
}

// FuturesData stores data for futures
type FuturesData struct {
	Ask            float64 `json:"ask"`
	Bid            float64 `json:"bid"`
	Change1h       float64 `json:"change1h"`
	Change24h      float64 `json:"change24h"`
	ChangeBod      float64 `json:"changeBod"`
	VolumeUSD24h   float64 `json:"volumeUsd24h"`
	Volume         float64 `json:"volume"`
	Description    string  `json:"description"`
	Enabled        bool    `json:"enabled"`
	Expired        bool    `json:"expired"`
	Expiry         string  `json:"expiry"`
	Index          float64 `json:"index"`
	Last           float64 `json:"last"`
	LowerBound     float64 `json:"lowerBound"`
	Mark           float64 `json:"mark"`
	Name           string  `json:"name"`
	Perpetual      bool    `json:"perpetual"`
	PostOnly       bool    `json:"postOnly"`
	PriceIncrement float64 `json:"priceIncrement"`
	SizeIncrement  float64 `json:"sizeIncrement"`
	Underlying     string  `json:"underlying"`
	UpperBound     float64 `json:"upperBound"`
	FutureType     string  `json:"type"`
}

// Futures stores futures data
type Futures struct {
	Success bool          `json:"success"`
	Result  []FuturesData `json:"result"`
}

// Future stores data for a singular future
type Future struct {
	Success bool        `json:"success"`
	Result  FuturesData `json:"result"`
}

// FutureStatsData stores data on futures stats
type FutureStatsData struct {
	Volume                   float64 `json:"volume"`
	NextFundingRate          float64 `json:"nextFundingRate"`
	NextFundingTime          string  `json:"nextFundingTime"`
	ExpirationPrice          float64 `json:"expirationPrice"`
	PredictedExpirationPrice float64 `json:"predictedExpirationPrice"`
	OpenInterest             float64 `json:"openInterest"`
	StrikePrice              float64 `json:"strikePrice"`
}

// FutureStats stores future stats
type FutureStats struct {
	Success bool            `json:"success"`
	Result  FutureStatsData `json:"result"`
}

// FundingRatesData stores data on funding rates
type FundingRatesData struct {
	Future string  `json:"future"`
	Rate   float64 `json:"rate"`
	Time   string  `json:"time"`
}

// FundingRates stores data on funding rates
type FundingRates struct {
	Success bool               `json:"success"`
	Result  []FundingRatesData `json:"result"`
}

// PositionData stores data of an open position
type PositionData struct {
	Cost                         float64 `json:"cost"`
	EntryPrice                   float64 `json:"entryPrice"`
	Future                       string  `json:"future"`
	InitialMarginRequirement     float64 `json:"initialMarginRequirement"`
	LongOrderSize                float64 `json:"longOrderSize"`
	MaintenanceMarginRequirement float64 `json:"maintenanceMarginRequirement"`
	NetSize                      float64 `json:"netSize"`
	OpenSize                     float64 `json:"openSize"`
	RealisedPnL                  float64 `json:"realisedPnL"`
	ShortOrderSide               float64 `json:"shortOrderSide"`
	Side                         string  `json:"side"`
	UnrealisedPnL                float64 `json:"unrealisedPnL"`
}

// AccountInfoData stores account data
type AccountInfoData struct {
	BackstopProvider             bool           `json:"backstopProvider"`
	Collateral                   float64        `json:"collateral"`
	FreeCollateral               float64        `json:"freeCollateral"`
	InitialMarginRequirement     float64        `json:"initialMarginRequirement"`
	Leverage                     float64        `json:"float64"`
	Liquidating                  bool           `json:"liquidating"`
	MaintenanceMarginRequirement float64        `json:"maintenanceMarginRequirement"`
	MakerFee                     float64        `json:"makerFee"`
	MarginFraction               float64        `json:"marginFraction"`
	OpenMarginFraction           float64        `json:"openMarginFraction"`
	TakerFee                     float64        `json:"takerFee"`
	TotalAccountValue            float64        `json:"totalAccountValue"`
	TotalPositionSize            float64        `json:"totalPositionSize"`
	Username                     string         `json:"username"`
	Positions                    []PositionData `json:"positions"`
}

// AccountData stores account data
type AccountData struct {
	Success bool            `json:"success"`
	Result  AccountInfoData `json:"result"`
}
