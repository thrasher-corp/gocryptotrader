package bitfinex

import (
	"bytes"
	"fmt"
	"time"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// PlatformStatusResponse holds whether the platform is operative
type PlatformStatusResponse struct {
	// Operative is false during maintenance, when orders can be neither placed nor cancelled
	Operative types.Boolean `tuple:"0"`
}

// UnmarshalJSON decodes a platform status array
func (p *PlatformStatusResponse) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, p) }

// Ticker is a trading pair's ticker; its changes, volume, high and low cover the last 24 hours
type Ticker struct {
	Bid     float64 `tuple:"0"`
	BidSize float64 `tuple:"1"`
	Ask     float64 `tuple:"2"`
	AskSize float64 `tuple:"3"`
	// DailyChange is how far the last price has moved
	DailyChange float64 `tuple:"4"`
	// DailyChangeRelative is DailyChange relative to the price it moved from; 0.01 is one percent
	DailyChangeRelative float64 `tuple:"5"`
	LastPrice           float64 `tuple:"6"`
	Volume              float64 `tuple:"7"`
	High                float64 `tuple:"8"`
	Low                 float64 `tuple:"9"`
	// FirstTradeTime is when the pair first traded, not when the ticker was last updated
	FirstTradeTime types.Time `tuple:"10"`
}

// UnmarshalJSON decodes a trading pair ticker array
func (t *Ticker) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, t) }

// FundingTicker is a funding currency's ticker; its rates are daily, and its changes, volume, high and low cover the
// last 24 hours
type FundingTicker struct {
	// FlashReturnRate is the average rate of all fixed rate funding over the last hour
	FlashReturnRate float64 `tuple:"0"`
	Bid             float64 `tuple:"1"`
	// BidPeriod is in days
	BidPeriod uint64  `tuple:"2"`
	BidSize   float64 `tuple:"3"`
	Ask       float64 `tuple:"4"`
	// AskPeriod is in days
	AskPeriod uint64  `tuple:"5"`
	AskSize   float64 `tuple:"6"`
	// DailyChange is how far the last rate has moved
	DailyChange float64 `tuple:"7"`
	// DailyChangeRelative is DailyChange relative to the rate it moved from; 0.01 is one percent
	DailyChangeRelative float64 `tuple:"8"`
	LastRate            float64 `tuple:"9"`
	Volume              float64 `tuple:"10"`
	High                float64 `tuple:"11"`
	Low                 float64 `tuple:"12"`
	// FlashReturnRateAmountAvailable is the funding available at the flash return rate
	FlashReturnRateAmountAvailable float64 `tuple:"15"`
	// FirstTradeTime is when the currency was first funded, not when the ticker was last updated
	FirstTradeTime types.Time `tuple:"16"`
}

// UnmarshalJSON decodes a funding currency ticker array
func (t *FundingTicker) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, t) }

// TickerEntry is a trading pair's ticker as GetTickers returns it, prefixed with its symbol
type TickerEntry struct {
	Symbol string `tuple:"0"`
	Ticker Ticker `tuple:"1,inline"`
}

// UnmarshalJSON decodes a symbol prefixed trading pair ticker array
func (t *TickerEntry) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, t) }

// FundingTickerEntry is a funding currency's ticker as GetTickers returns it, prefixed with its symbol
type FundingTickerEntry struct {
	Symbol string        `tuple:"0"`
	Ticker FundingTicker `tuple:"1,inline"`
}

// UnmarshalJSON decodes a symbol prefixed funding currency ticker array
func (t *FundingTickerEntry) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, t) }

// TickersResponse holds the tickers GetTickers returns, the trading pairs' apart from the funding currencies', as
// their arrays differ
type TickersResponse struct {
	Trading []TickerEntry
	Funding []FundingTickerEntry
}

// UnmarshalJSON decodes a list of tickers, telling trading pairs from funding currencies by their symbols' prefixes
func (t *TickersResponse) UnmarshalJSON(data []byte) error {
	return eachElement(data, func(n int, raw []byte, _ jsonparser.ValueType) error {
		symbol, err := jsonparser.GetString(raw, "[0]")
		if err != nil {
			return fmt.Errorf("%w: ticker %d has no symbol: %w", errTupleElementType, n, err)
		}
		switch {
		case isTradingSymbol(symbol):
			var entry TickerEntry
			if err := entry.UnmarshalJSON(raw); err != nil {
				return err
			}
			t.Trading = append(t.Trading, entry)
		case isFundingSymbol(symbol):
			var entry FundingTickerEntry
			if err := entry.UnmarshalJSON(raw); err != nil {
				return err
			}
			t.Funding = append(t.Funding, entry)
		default:
			return fmt.Errorf("%w: %q", errUnknownSymbolType, symbol)
		}
		return nil
	})
}

// TickersHistoryRequest selects the hourly history of trading pairs' best bid and ask, which goes back a year
type TickersHistoryRequest struct {
	// Symbols are trading pairs, such as tBTCUSD; none selects every pair. Funding currencies have no history
	Symbols []string
	Start   time.Time
	End     time.Time
	// Limit defaults to 100 and is at most 250
	Limit uint64
	// Sort is 1 for oldest first, or -1, the default, for newest first
	Sort int64
}

// HistoricTicker is a trading pair's best bid and ask at an hour
type HistoricTicker struct {
	Symbol string     `tuple:"0"`
	Bid    float64    `tuple:"1"`
	Ask    float64    `tuple:"3"`
	Time   types.Time `tuple:"12"`
}

// UnmarshalJSON decodes a historic ticker array
func (h *HistoricTicker) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, h) }

// PublicTradesRequest selects a trading pair's or funding currency's trades
type PublicTradesRequest struct {
	// Symbol is a trading pair, such as tBTCUSD, for GetPublicTrades, or a funding currency, such as fUSD, for
	// GetPublicFundingTrades
	Symbol string
	Start  time.Time
	End    time.Time
	// Limit defaults to 120 and is at most 10000
	Limit uint64
	// Sort is 1 for oldest first, or -1, the default, for newest first
	Sort int64
}

// PublicTrade is a trading pair's trade
type PublicTrade struct {
	ID   uint64     `tuple:"0"`
	Time types.Time `tuple:"1"`
	// Amount is positive for a buy and negative for a sell
	Amount float64 `tuple:"2"`
	Price  float64 `tuple:"3"`
}

// UnmarshalJSON decodes a trading pair trade array
func (p *PublicTrade) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, p) }

// PublicFundingTrade is a funding currency's trade
type PublicFundingTrade struct {
	ID   uint64     `tuple:"0"`
	Time types.Time `tuple:"1"`
	// Amount is positive for funding provided and negative for funding taken
	Amount float64 `tuple:"2"`
	// Rate is daily; 0.01 is one percent
	Rate float64 `tuple:"3"`
	// Period is in days
	Period uint64 `tuple:"4"`
}

// UnmarshalJSON decodes a funding currency trade array
func (p *PublicFundingTrade) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, p) }

// BookRequest selects a trading pair's or funding currency's book, aggregated by price or rate
type BookRequest struct {
	// Symbol is a trading pair, such as tBTCUSD, for GetBook, or a funding currency, such as fUSD, for GetFundingBook
	Symbol string
	// Precision, 0 to 4 for Bitfinex's P0 to P4, aggregates prices or rates to 5 down to 1 significant figures
	Precision uint64
	// Length is the levels a side, 1, 25, 100 or 250; it defaults to 25
	Length uint64
}

// RawBookRequest selects a trading pair's book of orders or a funding currency's book of offers
type RawBookRequest struct {
	// Symbol is a trading pair, such as tBTCUSD, for GetRawBook, or a funding currency, such as fUSD, for
	// GetRawFundingBook
	Symbol string
	// Length is the orders or offers a side, 1, 25, 100 or 250; it defaults to 25
	Length uint64
}

// BookLevel is a price level of a trading pair's book
type BookLevel struct {
	Price float64 `tuple:"0"`
	// Count is the number of orders at the price; on the websocket, 0 removes the level
	Count uint64 `tuple:"1"`
	// Amount is positive for a bid and negative for an ask
	Amount float64 `tuple:"2"`
}

// UnmarshalJSON decodes a trading pair book level array
func (b *BookLevel) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, b) }

// FundingBookLevel is a rate and period level of a funding currency's book
type FundingBookLevel struct {
	// Rate is daily; 0.01 is one percent
	Rate float64 `tuple:"0"`
	// Period is in days
	Period uint64 `tuple:"1"`
	// Count is the number of offers at the rate and period; on the websocket, 0 removes the level
	Count uint64 `tuple:"2"`
	// Amount is positive for an offer and negative for a bid, the reverse of a trading pair's book
	Amount float64 `tuple:"3"`
}

// UnmarshalJSON decodes a funding currency book level array
func (b *FundingBookLevel) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, b) }

// RawBookLevel is an order in a trading pair's raw book
type RawBookLevel struct {
	OrderID uint64 `tuple:"0"`
	// Price is zero on the websocket when the order leaves the book
	Price float64 `tuple:"1"`
	// Amount is positive for a bid and negative for an ask
	Amount float64 `tuple:"2"`
}

// UnmarshalJSON decodes a trading pair raw book level array
func (b *RawBookLevel) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, b) }

// RawFundingBookLevel is an offer in a funding currency's raw book
type RawFundingBookLevel struct {
	OfferID uint64 `tuple:"0"`
	// Period is in days
	Period uint64 `tuple:"1"`
	// Rate is daily; 0.01 is one percent. It is zero on the websocket when the offer leaves the book
	Rate float64 `tuple:"2"`
	// Amount is positive for an offer and negative for a bid, the reverse of a trading pair's book
	Amount float64 `tuple:"3"`
}

// UnmarshalJSON decodes a funding currency raw book level array
func (b *RawFundingBookLevel) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, b) }

// Stats keys, the statistics GetLastStat and GetStatsHistory return
const (
	// StatsKeyPositionSize is the total of a trading pair's long or short positions, in its base currency
	StatsKeyPositionSize = "pos.size"
	// StatsKeyFundingSize is a currency's active funding
	StatsKeyFundingSize = "funding.size"
	// StatsKeyCreditsSize is a currency's funding that positions use
	StatsKeyCreditsSize = "credits.size"
	// StatsKeyCreditsSizeByPair is a currency's funding that positions on one trading pair use
	StatsKeyCreditsSizeByPair = "credits.size.sym" //nolint:gosec // G101 takes the credits in the key's name for credentials
	// StatsKeyVolume1Day is the platform's trading volume over the last day
	StatsKeyVolume1Day = "vol.1d"
	// StatsKeyVolume7Day is the platform's trading volume over the last 7 days
	StatsKeyVolume7Day = "vol.7d"
	// StatsKeyVolume30Day is the platform's trading volume over the last 30 days
	StatsKeyVolume30Day = "vol.30d"
	// StatsKeyVolumeWeightedAveragePrice is a trading pair's volume weighted average price over each day
	StatsKeyVolumeWeightedAveragePrice = "vwap"
)

// StatsPlatform is the symbol the volume stats keys take, as they cover the whole platform
const StatsPlatform = "BFX"

// Position sides the position size statistic takes
const (
	StatsSideLong  = "long"
	StatsSideShort = "short"
)

// StatsRequest selects a statistic. The position size and volume weighted average price keys take a trading pair, such
// as tBTCUSD, the funding and credits keys a funding currency, such as fUSD, and the volume keys StatsPlatform
type StatsRequest struct {
	// Key is one of the stats keys, such as StatsKeyPositionSize
	Key    string
	Symbol string
	// Side is long or short, and only StatsKeyPositionSize takes it
	Side string
	// Pair is the trading pair, such as tBTCUSD, whose positions use the funding; only StatsKeyCreditsSizeByPair takes it
	Pair string
}

// StatsHistoryRequest selects a statistic's history
type StatsHistoryRequest struct {
	StatsRequest

	Start time.Time
	End   time.Time
	// Limit is at most 10000
	Limit uint64
	// Sort is 1 for oldest first, or -1 for newest first
	Sort int64
}

// Stat is a statistic's value at a time
type Stat struct {
	Time  types.Time `tuple:"0"`
	Value float64    `tuple:"1"`
}

// UnmarshalJSON decodes a statistic array
func (s *Stat) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, s) }

// CandlesRequest selects a trading pair's or funding currency's candles. A funding currency's candles cover its
// funding of one period, or of a range of periods aggregated; the website's are aggregated 30 over 2 to 30 days
type CandlesRequest struct {
	// Symbol is a trading pair, such as tBTCUSD, or a funding currency, such as fUSD
	Symbol string
	// Interval is the time frame each candle covers: one of kline.OneMin, FiveMin, FifteenMin, ThirtyMin, OneHour,
	// ThreeHour, SixHour, TwelveHour, OneDay, OneWeek, TwoWeek and OneMonth
	Interval kline.Interval
	// Period is the funding period, in days, of a funding currency's candles, unless they are aggregated
	Period uint64
	// Aggregation, 10 or 30, aggregates a funding currency's candles over the periods from PeriodStart to PeriodEnd
	Aggregation uint64
	// PeriodStart is the shortest funding period, in days, aggregated candles cover
	PeriodStart uint64
	// PeriodEnd is the longest funding period, in days, aggregated candles cover
	PeriodEnd uint64
}

// CandlesHistoryRequest selects the history of a trading pair's or funding currency's candles
type CandlesHistoryRequest struct {
	CandlesRequest

	Start time.Time
	End   time.Time
	// Limit defaults to 100 and is at most 10000
	Limit uint64
	// Sort is 1 for oldest first, or -1 for newest first
	Sort int64
}

// Candle is a candle of a trading pair's prices or a funding currency's rates
type Candle struct {
	Time   types.Time `tuple:"0"`
	Open   float64    `tuple:"1"`
	Close  float64    `tuple:"2"`
	High   float64    `tuple:"3"`
	Low    float64    `tuple:"4"`
	Volume float64    `tuple:"5"`
}

// UnmarshalJSON decodes a candle array
func (c *Candle) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, c) }

// DerivativeStatus is a derivative pair's status, as its history and the websocket status channel send it
type DerivativeStatus struct {
	Time types.Time `tuple:"0"`
	// DerivativePrice is the mid price of the derivative's book
	DerivativePrice float64 `tuple:"2"`
	// SpotPrice is the mid price of the underlying spot pair's book
	SpotPrice float64 `tuple:"3"`
	// InsuranceFundBalance is what the liquidation engine has to absorb losses
	InsuranceFundBalance float64    `tuple:"5"`
	NextFundingTime      types.Time `tuple:"7"`
	// NextFundingAccrued is the funding accrued for the next 8 hour period
	NextFundingAccrued float64 `tuple:"8"`
	// NextFundingStep counts the funding accruals
	NextFundingStep uint64 `tuple:"9"`
	// CurrentFunding is the funding applied in the current 8 hour period
	CurrentFunding float64 `tuple:"11"`
	// MarkPrice is the price of the BFX Composite Index
	MarkPrice    float64 `tuple:"14"`
	OpenInterest float64 `tuple:"17"`
	// ClampMinimum is the range of the average spread within which no funding is paid
	ClampMinimum float64 `tuple:"21"`
	// ClampMaximum caps the funding payment
	ClampMaximum float64 `tuple:"22"`
}

// UnmarshalJSON decodes a derivative status array
func (d *DerivativeStatus) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, d) }

// DerivativeStatusEntry is a derivative pair's status as GetDerivativesStatus returns it, prefixed with its key
type DerivativeStatusEntry struct {
	Key    string           `tuple:"0"`
	Status DerivativeStatus `tuple:"1,inline"`
}

// UnmarshalJSON decodes a keyed derivative status array
func (d *DerivativeStatusEntry) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, d) }

// DerivativesStatusHistoryRequest selects a derivative pair's status history
type DerivativesStatusHistoryRequest struct {
	// Key is a derivative pair, such as tBTCF0:USTF0
	Key   string
	Start time.Time
	End   time.Time
	// Limit is at most 5000
	Limit uint64
	// Sort is 1 for oldest first, or -1 for newest first
	Sort int64
}

// LiquidationsRequest selects a window of liquidations
type LiquidationsRequest struct {
	Start time.Time
	End   time.Time
	// Limit is at most 500
	Limit uint64
	// Sort is 1 for oldest first, or -1 for newest first
	Sort int64
}

// Liquidation is a position's liquidation
type Liquidation struct {
	// Type is pos
	Type       string     `tuple:"0"`
	PositionID uint64     `tuple:"1"`
	Time       types.Time `tuple:"2"`
	Symbol     string     `tuple:"4"`
	// Amount is the position's size: positive for a long position, negative for a short one
	Amount float64 `tuple:"5"`
	// BasePrice is the price the position was entered at
	BasePrice float64 `tuple:"6"`
	// IsMatch is false for the liquidation's trigger and true for its market execution
	IsMatch types.Boolean `tuple:"8"`
	// IsMarketSold is true when the position was sold into the market and false when the system acquired it, as the
	// REST docs give it; the websocket docs give the reverse
	IsMarketSold types.Boolean `tuple:"9"`
	// PriceAcquired is the price the position was acquired or sold at
	PriceAcquired float64 `tuple:"11"`
}

// UnmarshalJSON decodes a liquidation array
func (l *Liquidation) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, l) }

// liquidationsResponse holds the liquidations GetLiquidations returns, which Bitfinex sends each in an array of its own
type liquidationsResponse []Liquidation

// UnmarshalJSON decodes a list of liquidations, each sent alone in an array; any more an array holds are kept too, and
// null holds none, as it would for a plain slice
func (l *liquidationsResponse) UnmarshalJSON(data []byte) error {
	*l = (*l)[:0]
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}
	return eachElement(data, func(n int, raw []byte, dataType jsonparser.ValueType) error {
		if dataType == jsonparser.Null {
			return nil
		}
		return eachElement(raw, func(_ int, element []byte, elementType jsonparser.ValueType) error {
			if elementType == jsonparser.Null {
				return nil
			}
			var liquidation Liquidation
			if err := liquidation.UnmarshalJSON(element); err != nil {
				return fmt.Errorf("liquidation %d: %w", n, err)
			}
			*l = append(*l, liquidation)
			return nil
		})
	})
}

// Leaderboard keys
const (
	// LeaderboardKeyUnrealisedProfitPeriodDelta ranks the unrealised profit made over the time frame
	LeaderboardKeyUnrealisedProfitPeriodDelta = "plu_diff"
	// LeaderboardKeyUnrealisedProfitInception ranks the unrealised profit made since inception
	LeaderboardKeyUnrealisedProfitInception = "plu"
	// LeaderboardKeyVolume ranks volume
	LeaderboardKeyVolume = "vol"
	// LeaderboardKeyRealisedProfit ranks realised profit
	LeaderboardKeyRealisedProfit = "plr"
)

// LeaderboardSymbolGlobal is the symbol of the leaderboards across every pair
const LeaderboardSymbolGlobal = "tGLOBAL:USD"

// LeaderboardsRequest selects a leaderboard's standings. Not every key, time frame and symbol combine;
// GetLeaderboardCompetitions lists the leaderboards that run, such as plu:3h:tBTCUSD
type LeaderboardsRequest struct {
	// Key is one of the leaderboard keys, such as LeaderboardKeyVolume
	Key string
	// TimeFrame is kline.ThreeHour, kline.OneWeek or kline.OneMonth
	TimeFrame kline.Interval
	// Symbol is a trading pair, such as tBTCUSD, or LeaderboardSymbolGlobal
	Symbol string
	Start  time.Time
	End    time.Time
	// Limit defaults to 125 and is at most 10000
	Limit uint64
	// Sort is 1 for oldest first, or -1, the default, for newest first
	Sort int64
}

// LeaderboardStanding is a user's standing on a leaderboard
type LeaderboardStanding struct {
	Time     types.Time `tuple:"0"`
	Username string     `tuple:"2"`
	Ranking  uint64     `tuple:"3"`
	// Value is the volume, or unrealised or realised profit, the leaderboard ranks
	Value float64 `tuple:"6"`
	// TwitterHandle is set when the user has shared one
	TwitterHandle string `tuple:"9"`
}

// UnmarshalJSON decodes a leaderboard standing array
func (l *LeaderboardStanding) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, l) }

// FundingStatisticsRequest selects a funding currency's statistics, which Bitfinex keeps hourly
type FundingStatisticsRequest struct {
	// Symbol is a funding currency, such as fUSD
	Symbol string
	Start  time.Time
	End    time.Time
	// Limit is at most 250
	Limit uint64
}

// FundingStatistic is a funding currency's funding at a time
type FundingStatistic struct {
	Time types.Time `tuple:"0"`
	// FlashReturnRate is the daily flash return rate divided by 365: multiplied by 365 it gives the daily rate, and by
	// 365, 365 and 100 the yearly rate in percent
	FlashReturnRate float64 `tuple:"3"`
	// AveragePeriod is the average period, in days, of the funding provided
	AveragePeriod float64 `tuple:"4"`
	// FundingAmount is the funding provided
	FundingAmount float64 `tuple:"7"`
	// FundingAmountUsed is the funding provided that positions use
	FundingAmountUsed float64 `tuple:"8"`
	// FundingBelowThreshold is the sum of open funding offers below 0.75%
	FundingBelowThreshold float64 `tuple:"11"`
}

// UnmarshalJSON decodes a funding statistic array
func (f *FundingStatistic) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, f) }

// CurrencySymbol is a currency and the symbol it is better known by, such as DSH and DASH
type CurrencySymbol struct {
	Currency currency.Code `tuple:"0"`
	Symbol   string        `tuple:"1"`
}

// UnmarshalJSON decodes a currency symbol array
func (c *CurrencySymbol) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, c) }

// CurrencyLabel is a currency and its name, such as BTC and Bitcoin
type CurrencyLabel struct {
	Currency currency.Code `tuple:"0"`
	Label    string        `tuple:"1"`
}

// UnmarshalJSON decodes a currency label array
func (c *CurrencyLabel) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, c) }

// CurrencyUnit is a currency and the unit its amounts are in
type CurrencyUnit struct {
	Currency currency.Code `tuple:"0"`
	// Unit is the unit's symbol and name, such as Mi|MegaIOTA
	Unit string `tuple:"1"`
}

// UnmarshalJSON decodes a currency unit array
func (c *CurrencyUnit) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, c) }

// CurrencyUnderlying is a derivative currency and the currency it tracks, such as BTCF0 and BTC
type CurrencyUnderlying struct {
	Currency   currency.Code `tuple:"0"`
	Underlying currency.Code `tuple:"1"`
}

// UnmarshalJSON decodes a currency underlying array
func (c *CurrencyUnderlying) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, c) }

// CurrencyPool is a currency and the network it is carried on, such as BAT on ETH
type CurrencyPool struct {
	Currency currency.Code `tuple:"0"`
	// Pool is the network's currency, such as ETH, TRX or ARBETH for Arbitrum
	Pool currency.Code `tuple:"1"`
}

// UnmarshalJSON decodes a currency pool array
func (c *CurrencyPool) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, c) }

// CurrencyExplorer is a currency and its block explorer
type CurrencyExplorer struct {
	Currency currency.Code `tuple:"0"`
	Explorer BlockExplorer `tuple:"1"`
}

// UnmarshalJSON decodes a currency explorer array
func (c *CurrencyExplorer) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, c) }

// BlockExplorer is a block explorer's URLs; the address and transaction URLs hold VAL where the address or transaction
// ID goes
type BlockExplorer struct {
	URL            string `tuple:"0"`
	AddressURL     string `tuple:"1"`
	TransactionURL string `tuple:"2"`
}

// UnmarshalJSON decodes a block explorer array
func (b *BlockExplorer) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, b) }

// CurrencyWithdrawalFee is a currency and the fee its withdrawals cost
type CurrencyWithdrawalFee struct {
	Currency currency.Code `tuple:"0"`
	Fee      WithdrawalFee `tuple:"1"`
}

// UnmarshalJSON decodes a currency withdrawal fee array
func (c *CurrencyWithdrawalFee) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, c) }

// WithdrawalFee is a withdrawal's fee
type WithdrawalFee struct {
	// Amount is the fee, in the currency withdrawn; the element before it is undocumented
	Amount float64 `tuple:"1"`
}

// UnmarshalJSON decodes a withdrawal fee array
func (w *WithdrawalFee) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, w) }

// TransferMethod is a method API withdrawals take and the currencies it moves, such as WIRE for USD, EUR and others
type TransferMethod struct {
	// Method is the method's name, such as BITCOIN or TETHERUSE
	Method     string          `tuple:"0"`
	Currencies []currency.Code `tuple:"1"`
}

// UnmarshalJSON decodes a transfer method array
func (t *TransferMethod) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, t) }

// PairInfoEntry is a pair's market information, prefixed with the pair
type PairInfoEntry struct {
	// Pair is the pair without its t prefix, such as BTCUSD or BTCF0:USTF0
	Pair string   `tuple:"0"`
	Info PairInfo `tuple:"1"`
}

// UnmarshalJSON decodes a pair info entry array
func (p *PairInfoEntry) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, p) }

// PairInfo is a pair's market information
type PairInfo struct {
	// FirstTradeTime is unset for some pairs
	FirstTradeTime types.Time `tuple:"0"`
	// MinimumOrderSize is the smallest amount an order can have, in the base currency
	MinimumOrderSize types.Number `tuple:"3"`
	// MaximumOrderSize is the largest amount an order can have, in the base currency
	MaximumOrderSize types.Number `tuple:"4"`
	// InitialMargin is the margin opening a position needs, relative to its value; it is unset for a pair margin
	// trading does not offer
	InitialMargin float64 `tuple:"8"`
	// MinimumMargin is the margin keeping a position open needs, relative to its value
	MinimumMargin float64 `tuple:"9"`
}

// UnmarshalJSON decodes a pair info array
func (p *PairInfo) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, p) }

// TransferMethodStatus is whether a transfer method's deposits and withdrawals run, and what they need
type TransferMethodStatus struct {
	// Method is the method's name, such as BITCOIN, as GetTransferMethods lists it
	Method string `tuple:"0"`
	// DepositActive is false while the method's deposits are in maintenance
	DepositActive types.Boolean `tuple:"1"`
	// WithdrawalActive is false while the method's withdrawals are in maintenance
	WithdrawalActive           types.Boolean `tuple:"2"`
	DepositPaymentIDAllowed    types.Boolean `tuple:"7"`
	WithdrawalPaymentIDAllowed types.Boolean `tuple:"8"`
	// DepositConfirmationsRequired is the confirmations a deposit needs; it is unset for some methods
	DepositConfirmationsRequired uint64 `tuple:"11"`
}

// UnmarshalJSON decodes a transfer method status array
func (t *TransferMethodStatus) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, t) }

// MarginSpecificationResponse holds the collateral haircuts and margin requirements of margin trading
type MarginSpecificationResponse struct {
	// Configuration holds base settings, such as base_haircut, and each collateral currency's haircut, such as
	// IOT_haircut
	Configuration map[string]float64 `json:"conf"`
	// Initial holds each margin pair's initial margin, such as BTCUSD's, with the base and base_cap defaults and, for
	// some pairs, a _risk variant
	Initial map[string]float64 `json:"initial"`
	// Minimum holds each margin pair's minimum margin, keyed as Initial is
	Minimum map[string]float64 `json:"minimum"`
}

// DerivativesFeesResponse holds the fee settings of derivatives
type DerivativesFeesResponse struct {
	Perpetual PerpetualContractFees `json:"deriv_perp"`
}

// PerpetualContractFees holds the funding correction settings of perpetual contracts
type PerpetualContractFees struct {
	CorrectionClampMinimum   float64 `json:"correction_clamp_min"`
	CorrectionClampInsurance float64 `json:"correction_clamp_insurance"`
	CorrectionSlope          float64 `json:"correction_slope"`
}

// VirtualAssetServiceProvider is a virtual asset service provider a withdrawal can name as its recipient, for the
// Travel Rule
type VirtualAssetServiceProvider struct {
	// ID is the provider's decentralised ID, which a withdrawal's vasp_did takes
	ID   string `json:"id"`
	Name string `json:"name"`
}

// MarketAveragePriceRequest selects the amount of a trading pair whose average price GetMarketAveragePrice returns
type MarketAveragePriceRequest struct {
	// Symbol is a trading pair, such as tBTCUSD
	Symbol string
	// Amount is positive to buy, at the asks, and negative to sell, at the bids
	Amount float64
	// PriceLimit is sent as rate_limit, which the docs describe only as a limit price
	PriceLimit float64
}

// MarketAveragePriceResponse is the average price at which a trading pair's book would fill an amount
type MarketAveragePriceResponse struct {
	AveragePrice float64 `tuple:"0"`
	// Amount is the amount priced, signed as the request's
	Amount float64 `tuple:"1"`
}

// UnmarshalJSON decodes a market average price array
func (m *MarketAveragePriceResponse) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, m) }

// FundingMarketAveragePriceRequest selects the amount of a funding currency whose average rate
// GetFundingMarketAveragePrice returns
type FundingMarketAveragePriceRequest struct {
	// Symbol is a funding currency, such as fUSD
	Symbol string
	// Amount is positive to take funding, at the offers' rates, and negative to provide it, at the bids'
	Amount float64
	// Period is the longest funding period, in days, to take or provide funding for
	Period uint64
	// RateLimit is sent as rate_limit, which the docs describe only as a limit rate
	RateLimit float64
}

// FundingMarketAveragePriceResponse is the average rate at which a funding currency's book would fill an amount
type FundingMarketAveragePriceResponse struct {
	// AverageRate is daily; 0.01 is one percent
	AverageRate float64 `tuple:"0"`
	// Amount is the amount priced, signed as the request's
	Amount float64 `tuple:"1"`
}

// UnmarshalJSON decodes a funding market average price array
func (f *FundingMarketAveragePriceResponse) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, f)
}

// ForeignExchangeRateRequest selects the currencies an exchange rate is between
type ForeignExchangeRateRequest struct {
	// BaseCurrency is the currency priced, such as BTC
	BaseCurrency currency.Code
	// QuoteCurrency is the currency the price is in, such as USD
	QuoteCurrency currency.Code
}

// foreignExchangeRateBody is the body GetForeignExchangeRate sends
type foreignExchangeRateBody struct {
	BaseCurrency  string `json:"ccy1"`
	QuoteCurrency string `json:"ccy2"`
}

// ForeignExchangeRateResponse holds the exchange rate between two currencies
type ForeignExchangeRateResponse struct {
	// CurrentRate is the price of one unit of the base currency in the quote currency
	CurrentRate float64 `tuple:"0"`
}

// UnmarshalJSON decodes a foreign exchange rate array
func (f *ForeignExchangeRateResponse) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, f)
}
