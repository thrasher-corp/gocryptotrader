package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WebsocketOrderbookUpdate is a push from the orderbook.{depth}.{symbol} and orderbook.full.{symbol} topics (Orderbook,
// Full Orderbook)
type WebsocketOrderbookUpdate struct {
	Topic     string     `json:"topic"`
	Type      string     `json:"type"`
	Timestamp types.Time `json:"ts"`
	// ID is set on option pushes only
	ID                string             `json:"id"`
	Data              WebsocketOrderbook `json:"data"`
	CreationTimestamp types.Time         `json:"cts"` // Matching engine time, comparable with T on public trades
}

// WebsocketOrderbook is an orderbook snapshot or delta; a size of 0 in a delta removes the price level
type WebsocketOrderbook struct {
	Symbol string                           `json:"s"`
	Bids   orderbook.LevelsArrayPriceAmount `json:"b"`
	Asks   orderbook.LevelsArrayPriceAmount `json:"a"`
	// UpdateID restarts at 1 when Bybit restarts the service, which always comes with a snapshot
	UpdateID uint64 `json:"u"`
	Sequence uint64 `json:"seq"`
}

// WebsocketRPIOrderbookUpdate is a push from the orderbook.rpi.{symbol} topic (RPI Orderbook)
type WebsocketRPIOrderbookUpdate struct {
	Topic             string                `json:"topic"`
	Type              string                `json:"type"`
	Timestamp         types.Time            `json:"ts"`
	Data              WebsocketRPIOrderbook `json:"data"`
	CreationTimestamp types.Time            `json:"cts"` // Matching engine time, comparable with T on public trades
}

// WebsocketRPIOrderbook is an orderbook snapshot or delta that includes Retail Price Improvement quotes
type WebsocketRPIOrderbook struct {
	Symbol   string              `json:"s"`
	Bids     []RPIOrderbookLevel `json:"b"`
	Asks     []RPIOrderbookLevel `json:"a"`
	UpdateID uint64              `json:"u"`
	Sequence uint64              `json:"seq"`
}

// WebsocketPublicTradeUpdate is a push from the publicTrade.{symbol} topic, which options subscribe to by base coin
// (Trade)
type WebsocketPublicTradeUpdate struct {
	// ID is set on option pushes only
	ID        string                 `json:"id"`
	Topic     string                 `json:"topic"`
	Type      string                 `json:"type"`
	Timestamp types.Time             `json:"ts"`
	Data      []WebsocketPublicTrade `json:"data"`
}

// WebsocketPublicTrade is a public trade
type WebsocketPublicTrade struct {
	OrderFillTimestamp types.Time   `json:"T"`
	Symbol             string       `json:"s"`
	Side               string       `json:"S"` // Side of the taker
	Size               types.Number `json:"v"`
	Price              types.Number `json:"p"`
	// TickDirection is set for perpetuals and futures only
	TickDirection         string       `json:"L"`
	TradeID               string       `json:"i"`
	BlockTrade            bool         `json:"BT"`
	RPI                   bool         `json:"RPI"`
	Sequence              uint64       `json:"seq"`
	MarkPrice             types.Number `json:"mP"`  // Options only
	IndexPrice            types.Number `json:"iP"`  // Options only
	MarkImpliedVolatility types.Number `json:"mIv"` // Options only
	ImpliedVolatility     types.Number `json:"iv"`  // Options only
}

// WebsocketTickerUpdate is a push from the tickers.{symbol} topic (Ticker). Linear and inverse tickers send a snapshot
// followed by deltas that hold only the fields that changed; spot and option tickers send snapshots only
type WebsocketTickerUpdate struct {
	Topic         string     `json:"topic"`
	Type          string     `json:"type"`
	Timestamp     types.Time `json:"ts"`
	CrossSequence uint64     `json:"cs"`
	// ID is set on option pushes only
	ID   string          `json:"id"`
	Data WebsocketTicker `json:"data"`
}

// WebsocketTicker holds the ticker fields of every category; each category sets its own subset
type WebsocketTicker struct {
	Symbol                  string       `json:"symbol"`
	TickDirection           string       `json:"tickDirection"`
	Price24HourPercent      types.Number `json:"price24hPcnt"`
	LastPrice               types.Number `json:"lastPrice"`
	PreviousPrice24Hour     types.Number `json:"prevPrice24h"`
	HighPrice24Hour         types.Number `json:"highPrice24h"`
	LowPrice24Hour          types.Number `json:"lowPrice24h"`
	PreviousPrice1Hour      types.Number `json:"prevPrice1h"`
	MarkPrice               types.Number `json:"markPrice"`
	IndexPrice              types.Number `json:"indexPrice"`
	OpenInterest            types.Number `json:"openInterest"`      // Both sides
	OpenInterestValue       types.Number `json:"openInterestValue"` // Both sides
	SingleOpenInterest      types.Number `json:"singleOpenInterest"`
	SingleOpenInterestValue types.Number `json:"singleOpenInterestValue"`
	// Turnover24Hour and Volume24Hour are in the quote and base currency, except on inverse contracts, where they swap
	Turnover24Hour  types.Number `json:"turnover24h"`
	Volume24Hour    types.Number `json:"volume24h"`
	NextFundingTime types.Time   `json:"nextFundingTime"`
	FundingRate     types.Number `json:"fundingRate"`
	Bid1Price       types.Number `json:"bid1Price"`
	Bid1Size        types.Number `json:"bid1Size"`
	Ask1Price       types.Number `json:"ask1Price"`
	Ask1Size        types.Number `json:"ask1Size"`
	// DeliveryTime is set for futures only
	DeliveryTime           time.Time    `json:"deliveryTime"`
	BasisRate              types.Number `json:"basisRate"`
	DeliveryFeeRate        types.Number `json:"deliveryFeeRate"`
	PredictedDeliveryPrice types.Number `json:"predictedDeliveryPrice"`
	// Basis is not documented but sent for futures
	Basis                      types.Number `json:"basis"`
	PreOpenPrice               types.Number `json:"preOpenPrice"`
	PreQuantity                types.Number `json:"preQty"`
	CurrentPreListingPhase     string       `json:"curPreListingPhase"`
	FundingIntervalHour        types.Number `json:"fundingIntervalHour"`
	FundingCap                 types.Number `json:"fundingCap"`
	BasisRateYear              types.Number `json:"basisRateYear"`
	BidPrice                   types.Number `json:"bidPrice"`
	BidSize                    types.Number `json:"bidSize"`
	BidImpliedVolatility       types.Number `json:"bidIv"`
	AskPrice                   types.Number `json:"askPrice"`
	AskSize                    types.Number `json:"askSize"`
	AskImpliedVolatility       types.Number `json:"askIv"`
	MarkPriceImpliedVolatility types.Number `json:"markPriceIv"`
	UnderlyingPrice            types.Number `json:"underlyingPrice"`
	TotalVolume                types.Number `json:"totalVolume"`
	TotalTurnover              types.Number `json:"totalTurnover"`
	Delta                      types.Number `json:"delta"`
	Gamma                      types.Number `json:"gamma"`
	Vega                       types.Number `json:"vega"`
	Theta                      types.Number `json:"theta"`
	Change24Hour               types.Number `json:"change24h"`
	USDIndexPrice              types.Number `json:"usdIndexPrice"`
}

// WebsocketKlineUpdate is a push from the kline.{interval}.{symbol} topic (Kline)
type WebsocketKlineUpdate struct {
	Topic     string           `json:"topic"`
	Type      string           `json:"type"`
	Timestamp types.Time       `json:"ts"`
	Data      []WebsocketKline `json:"data"`
}

// WebsocketKline is a candle; it is still open until Confirm is true
type WebsocketKline struct {
	Start     types.Time   `json:"start"`
	End       types.Time   `json:"end"`
	Interval  string       `json:"interval"`
	Open      types.Number `json:"open"`
	Close     types.Number `json:"close"`
	High      types.Number `json:"high"`
	Low       types.Number `json:"low"`
	Volume    types.Number `json:"volume"`
	Turnover  types.Number `json:"turnover"`
	Confirm   bool         `json:"confirm"`
	Timestamp types.Time   `json:"timestamp"` // Time of the last trade in the candle
}

// WebsocketAllLiquidationUpdate is a push from the allLiquidation.{symbol} topic (All Liquidation)
type WebsocketAllLiquidationUpdate struct {
	Topic     string                 `json:"topic"`
	Type      string                 `json:"type"`
	Timestamp types.Time             `json:"ts"`
	Data      []WebsocketLiquidation `json:"data"`
}

// WebsocketLiquidation is a liquidation; a Side of Buy means a long position was liquidated
type WebsocketLiquidation struct {
	UpdatedTime     types.Time   `json:"T"`
	Symbol          string       `json:"s"`
	Side            string       `json:"S"`
	Size            types.Number `json:"v"`
	BankruptcyPrice types.Number `json:"p"`
}

// WebsocketInsurancePoolUpdate is a push from the insurance.{USDT|USDC|inverse} topic (Insurance Pool)
type WebsocketInsurancePoolUpdate struct {
	Topic     string                   `json:"topic"`
	Type      string                   `json:"type"`
	Timestamp types.Time               `json:"ts"`
	Data      []WebsocketInsurancePool `json:"data"`
}

// WebsocketInsurancePool is the balance of an isolated insurance pool
type WebsocketInsurancePool struct {
	Coin currency.Code `json:"coin"`
	// Symbols lists, comma separated, the contracts the pool covers
	Symbols    string       `json:"symbols"`
	Balance    types.Number `json:"balance"`
	UpdateTime types.Time   `json:"updateTime"`
}

// WebsocketPriceLimitUpdate is a push from the priceLimit.{symbol} topic (Order Price Limit)
type WebsocketPriceLimitUpdate struct {
	Topic     string     `json:"topic"`
	Timestamp types.Time `json:"ts"`
	// Data is an object, although the docs table calls it an array
	Data WebsocketPriceLimit `json:"data"`
}

// WebsocketPriceLimit is the highest bid and lowest ask price an order may have
type WebsocketPriceLimit struct {
	Symbol    string       `json:"symbol"`
	BuyLimit  types.Number `json:"buyLmt"`
	SellLimit types.Number `json:"sellLmt"`
}

// WebsocketADLAlertUpdate is a push from the adlAlert.{USDT|USDC|inverse} topic (ADL Alert)
type WebsocketADLAlertUpdate struct {
	Topic     string              `json:"topic"`
	Type      string              `json:"type"`
	Timestamp types.Time          `json:"ts"`
	Data      []WebsocketADLAlert `json:"data"`
}

// WebsocketADLAlert holds a contract's insurance pool balance and the drawdown ratios that trigger and stop
// auto-deleveraging. The values are documented as strings but the docs' example sends bare numbers
type WebsocketADLAlert struct {
	Coin   currency.Code `json:"c"`
	Symbol string        `json:"s"`
	// Balance of a shared insurance pool refreshes daily at 00:00 UTC
	Balance types.Number `json:"b"`
	// MaxBalance is deprecated and always empty
	MaxBalance          types.Number `json:"mb"`
	InsurancePNLRatio   types.Number `json:"i_pr"`
	PNLRatio            types.Number `json:"pr"`
	ADLTriggerThreshold types.Number `json:"adl_tt"`
	ADLStopRatio        types.Number `json:"adl_sr"`
}
