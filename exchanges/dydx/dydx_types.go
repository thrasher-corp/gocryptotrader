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

// WithdrawalLiquidityResponse represents accounts that have available funds for fast withdrawals.
type WithdrawalLiquidityResponse struct {
	LiquidityProviders map[string]LiquidityProvider `json:"liquidityProviders"`
}

// LiquidityProvider represents a liquidation provider item data
type LiquidityProvider struct {
	AvailableFunds string      `json:"availableFunds"`
	StarkKey       string      `json:"starkKey"`
	Quote          interface{} `json:"quote"`
}

// TickerDatas represents market's statistics data.
type TickerDatas struct {
	Markets map[string]TickerData `json:"markets"`
}

// TickerData represents ticker data for a market.
type TickerData struct {
	Market      string `json:"market"`
	Open        string `json:"open"`
	Close       string `json:"close"`
	High        string `json:"high"`
	Low         string `json:"low"`
	BaseVolume  string `json:"baseVolume"`
	QuoteVolume string `json:"quoteVolume"`
	Type        string `json:"type"`
	Fees        string `json:"fees"`
}

// HistoricFundingResponse represents a historic funding response data.
type HistoricFundingResponse struct {
	HistoricalFundings []HistoricalFunding `json:"historicalFunding"`
}

// HistoricalFunding represents historical funding rates for a market.
type HistoricalFunding struct {
	Market      string    `json:"market"`
	Rate        string    `json:"rate"`
	Price       string    `json:"price"`
	EffectiveAt time.Time `json:"effectiveAt"`
}

// MarketCandlesResponse represents response data for market candlestick data.
type MarketCandlesResponse struct {
	Candles []MarketCandle `json:"candles"`
}

// MarketCandle represents candle statistics for a specific market.
type MarketCandle struct {
	StartedAt            time.Time `json:"startedAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
	Market               string    `json:"market"`
	Resolution           string    `json:"resolution"`
	Low                  float64   `json:"low,string"`
	High                 float64   `json:"high,string"`
	Open                 float64   `json:"open,string"`
	Close                float64   `json:"close,string"`
	BaseTokenVolume      string    `json:"baseTokenVolume"`
	Trades               string    `json:"trades"`
	UsdVolume            float64   `json:"usdVolume,string"`
	StartingOpenInterest string    `json:"startingOpenInterest"`
}

// ConfigurationVariableResponse represents any configuration variables for the exchange.
type ConfigurationVariableResponse struct {
	CollateralAssetID             string `json:"collateralAssetId"`
	CollateralTokenAddress        string `json:"collateralTokenAddress"`
	DefaultMakerFee               string `json:"defaultMakerFee"`
	DefaultTakerFee               string `json:"defaultTakerFee"`
	ExchangeAddress               string `json:"exchangeAddress"`
	MaxExpectedBatchLengthMinutes string `json:"maxExpectedBatchLengthMinutes"`
	MaxFastWithdrawalAmount       string `json:"maxFastWithdrawalAmount"`
	CancelOrderRateLimiting       struct {
		MaxPointsMulti  int `json:"maxPointsMulti"`
		MaxPointsSingle int `json:"maxPointsSingle"`
		WindowSecMulti  int `json:"windowSecMulti"`
		WindowSecSingle int `json:"windowSecSingle"`
	} `json:"cancelOrderRateLimiting"`
	PlaceOrderRateLimiting struct {
		MaxPoints                 int `json:"maxPoints"`
		WindowSec                 int `json:"windowSec"`
		TargetNotional            int `json:"targetNotional"`
		MinLimitConsumption       int `json:"minLimitConsumption"`
		MinMarketConsumption      int `json:"minMarketConsumption"`
		MinTriggerableConsumption int `json:"minTriggerableConsumption"`
		MaxOrderConsumption       int `json:"maxOrderConsumption"`
	} `json:"placeOrderRateLimiting"`
}

// APIServerTime represents the server time in ISO(string) and Epoch milliseconds.
type APIServerTime struct {
	ISO   string    `json:"iso"`
	Epoch time.Time `json:"epoch"`
}

// LeaderboardPNLs represents top PNLs for a specified period and how they rank against
type LeaderboardPNLs struct {
	PrizePool         int64   `json:"prizePool"`
	NumHedgiesWinners int64   `json:"numHedgiesWinners"`
	NumPrizeWinners   int64   `json:"numPrizeWinners"`
	RatioPromoted     float64 `json:"ratioPromoted"`
	RatioDemoted      float64 `json:"ratioDemoted"`
	MinimumEquity     int64   `json:"minimumEquity"`
	MinimumDYDXTokens int64   `json:"minimumDYDXTokens"`
	SeasonNumber      int64   `json:"seasonNumber"`
	TopPnls           []struct {
		Username              string      `json:"username"`
		EthereumAddress       string      `json:"ethereumAddress"`
		PublicID              string      `json:"publicId"`
		AbsolutePnl           string      `json:"absolutePnl"`
		PercentPnl            string      `json:"percentPnl"`
		AbsoluteRank          int64       `json:"absoluteRank"`
		PercentRank           int64       `json:"percentRank"`
		SeasonExpectedOutcome string      `json:"seasonExpectedOutcome"`
		HedgieWon             interface{} `json:"hedgieWon"`
		PrizeWon              interface{} `json:"prizeWon"`
	} `json:"topPnls"`
	NumParticipants int       `json:"numParticipants"`
	UpdatedAt       time.Time `json:"updatedAt"`
	StartedAt       time.Time `json:"startedAt"`
	EndsAt          time.Time `json:"endsAt"`
}

// RetroactiveMiningReward represents retroactive mining rewards for an ethereum address.
type RetroactiveMiningReward struct {
	Allocation   string `json:"allocation"`
	TargetVolume string `json:"targetVolume"`
}

// CurrentRevealedHedgies represents hedgies for competition distribution
type CurrentRevealedHedgies struct {
	Daily struct {
		BlockNumber       int64    `json:"blockNumber,string"`
		CompetitionPeriod int64    `json:"competitionPeriod"`
		TokenIds          []string `json:"tokenIds"`
	} `json:"daily"`
	Weekly struct {
		BlockNumber       int64    `json:"blockNumber,string"`
		Competitionperiod int64    `json:"competitionperiod"`
		TokenIds          []string `json:"tokenIds"`
	} `json:"weekly"`
}

// HistoricalRevealedHedgies represents historically revealed Hedgies.
type HistoricalRevealedHedgies struct {
	HistoricalTokenIds []struct {
		BlockNumber       int64    `json:"blockNumber,string"`
		Competitionperiod int64    `json:"competitionperiod"`
		TokenIds          []string `json:"tokenIds"`
	} `json:"historicalTokenIds"`
}

// InsuranceFundBalance represents balance of the dYdX insurance fund.
type InsuranceFundBalance struct {
	Balance string `json:"balance"`
}

// PublicProfile represents the public profile of a user given their public ID.
type PublicProfile struct {
	Username           string `json:"username"`
	EthereumAddress    string `json:"ethereumAddress"`
	DYDXHoldings       string `json:"DYDXHoldings"`
	StakedDYDXHoldings string `json:"stakedDYDXHoldings"`
	HedgiesHeld        []int  `json:"hedgiesHeld"`
	TwitterHandle      string `json:"twitterHandle"`
	TradingLeagues     struct {
		CurrentLeague        string `json:"currentLeague"`
		CurrentLeagueRanking int    `json:"currentLeagueRanking"`
	} `json:"tradingLeagues"`
	TradingPnls struct {
		AbsolutePnl30D string `json:"absolutePnl30D"`
		PercentPnl30D  string `json:"percentPnl30D"`
		Volume30D      string `json:"volume30D"`
	} `json:"tradingPnls"`
	TradingRewards struct {
		CurEpoch                  string `json:"curEpoch"`
		CurEpochEstimatedRewards  int    `json:"curEpochEstimatedRewards"`
		PrevEpochEstimatedRewards int    `json:"prevEpochEstimatedRewards"`
	} `json:"tradingRewards"`
}

// WSOrderbookData represents the websocket orderbook data pushed through the channel.
type WSOrderbookData struct {
	Type         string `json:"type"`
	ConnectionID string `json:"connection_id"`
	MessageID    int    `json:"message_id"`
	ID           string `json:"id"`
	Channel      string `json:"channel"`
	Contents     struct {
		Offset string        `json:"offset"`
		Bids   [][]string    `json:"bids"`
		Asks   []interface{} `json:"asks"`
	} `json:"contents"`
}
