package kraken

import (
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errInvalidDepth     = errors.New("invalid depth")
	errInvalidGrouping  = errors.New("invalid grouping")
	errInvalidCount     = errors.New("invalid count")
	errUnexpectedLength = errors.New("unexpected array length")
)

// ServerTimeResponse holds Kraken's server time
type ServerTimeResponse struct {
	UnixTime types.Time `json:"unixtime"`
	// RFC1123 is the time in RFC 1123 format with a two digit year, such as "Fri, 09 Oct 26 00:10:02 +0000"
	RFC1123 string `json:"rfc1123"`
}

// SystemStatusResponse holds the trading engine's status and any maintenance or incident advisories. An advisory
// explains why a status is happening or coming; the status itself is set by the trading engine alone
type SystemStatusResponse struct {
	Status              string             `json:"status"`
	Timestamp           time.Time          `json:"timestamp"`
	UpcomingMaintenance []MaintenanceEvent `json:"upcoming_maintenance"`
	Emergency           []EmergencyEvent   `json:"emergency"`
}

// MaintenanceEvent is a scheduled maintenance advisory
type MaintenanceEvent struct {
	EventID       uint64    `json:"event_id"`
	Title         string    `json:"title"`
	ExpectedStart time.Time `json:"expected_start_utc"`
	ExpectedEnd   time.Time `json:"expected_end_utc"`
	// TimeToStartSeconds is evaluated when the response is served, and responses are cached, so a live countdown should
	// be derived from ExpectedStart
	TimeToStartSeconds int64     `json:"time_to_start_s"`
	Phase              string    `json:"phase"`
	AffectedServices   []string  `json:"affected_services"`
	OrderSubmission    string    `json:"order_submission"`
	RecommendedAction  string    `json:"recommended_action"`
	CancelBefore       time.Time `json:"cancel_before_utc"`
	SourceURL          string    `json:"source_url"`
}

// EmergencyEvent is an unresolved incident advisory
type EmergencyEvent struct {
	EventID          uint64              `json:"event_id"`
	Title            string              `json:"title"`
	IncidentStatus   string              `json:"incident_status"`
	Impact           string              `json:"impact"`
	AffectedServices []string            `json:"affected_services"`
	StartedAt        time.Time           `json:"started_at_utc"`
	NextSteps        []EmergencyNextStep `json:"next_steps"`
	SourceURL        string              `json:"source_url"`
}

// EmergencyNextStep is an operational action Kraken forecasts during an incident
type EmergencyNextStep struct {
	AppliesTo  []string  `json:"applies_to"`
	Type       string    `json:"type"`
	ExpectedAt time.Time `json:"expected_at_utc"`
}

// MaintenanceScheduleResponse holds the maintenance scheduled in the next 7 days
type MaintenanceScheduleResponse struct {
	Events []MaintenanceEvent `json:"events"`
}

// AssetsRequest holds the parameters of Get Asset Info
type AssetsRequest struct {
	// Assets limits the response to these assets; every asset is returned when it is empty
	Assets []currency.Code
	// AssetClass is currency, the default, or tokenized_asset for xStocks
	AssetClass string
	// DisplayNames keys the response by display names, such as BTC, rather than internal names, such as XXBT
	DisplayNames bool
}

// AssetInfo holds an asset's details
type AssetInfo struct {
	AssetClass string `json:"aclass"`
	// AlternativeName is the name GoCryptoTrader uses, such as XBT for XXBT
	AlternativeName string       `json:"altname"`
	Decimals        uint64       `json:"decimals"`
	DisplayDecimals uint64       `json:"display_decimals"`
	CollateralValue float64      `json:"collateral_value"`
	Status          string       `json:"status"`
	MarginRate      types.Number `json:"margin_rate"`
}

// AssetPairsRequest holds the parameters of Get Tradable Asset Pairs
type AssetPairsRequest struct {
	// Pairs limits the response to these pairs; every pair is returned when it is empty
	Pairs currency.Pairs
	// BaseAssetClass is currency, the default, or tokenized_asset for xStocks
	BaseAssetClass string
	// Info selects the details returned: info, the default, leverage, fees or margin
	Info string
	// CountryCode limits the response to the pairs available in an ISO 3166-1 alpha-2 country or region
	CountryCode string
	// ExecutionVenues limits the response to pairs listed on these venues, international by default
	ExecutionVenues []string
	// DisplayNames keys the response by display names, such as BTC/USD, and names the base, quote and fee volume
	// currency by display names too
	DisplayNames bool
}

// AssetPair holds a tradable pair's details. Kraken no longer fills its deprecated fees and fees_maker fields; fees
// come from Get Trade Volume
type AssetPair struct {
	// AlternativeName is the pair as the request formats it, such as XBTUSD
	AlternativeName string `json:"altname"`
	// WebsocketName is the pair as GoCryptoTrader's configuration delimits it, such as XBT/USD
	WebsocketName     string        `json:"wsname"`
	BaseAssetClass    string        `json:"aclass_base"`
	Base              currency.Code `json:"base"`
	QuoteAssetClass   string        `json:"aclass_quote"`
	Quote             currency.Code `json:"quote"`
	ExecutionVenue    string        `json:"execution_venue"`
	Lot               string        `json:"lot"`
	PairDecimals      uint64        `json:"pair_decimals"`
	CostDecimals      uint64        `json:"cost_decimals"`
	LotDecimals       uint64        `json:"lot_decimals"`
	LotMultiplier     uint64        `json:"lot_multiplier"`
	LeverageBuy       []uint64      `json:"leverage_buy"`
	LeverageSell      []uint64      `json:"leverage_sell"`
	FeeVolumeCurrency currency.Code `json:"fee_volume_currency"`
	MarginCall        uint64        `json:"margin_call"`
	MarginStop        uint64        `json:"margin_stop"`
	// MarginLevel is the stop-out level the margin info returns in place of MarginStop
	MarginLevel        uint64       `json:"margin_level"`
	OrderMinimum       types.Number `json:"ordermin"`
	CostMinimum        types.Number `json:"costmin"`
	TickSize           types.Number `json:"tick_size"`
	Status             string       `json:"status"`
	LongPositionLimit  uint64       `json:"long_position_limit"`
	ShortPositionLimit uint64       `json:"short_position_limit"`
}

// TickerInformationRequest holds the parameters of Get Ticker Information
type TickerInformationRequest struct {
	// Pairs limits the response to these pairs; every tradable pair is returned when it is empty
	Pairs currency.Pairs
	// AssetClass is forex, the default, or tokenized_asset for xStocks
	AssetClass string
	// DisplayNames keys the response by display names, such as BTC/USD, rather than internal names, such as XXBTZUSD
	DisplayNames bool
}

// TickerInformation holds a pair's ticker. Today's values start at midnight UTC
type TickerInformation struct {
	Ask                        TickerLevel       `json:"a"`
	Bid                        TickerLevel       `json:"b"`
	LastTradeClosed            TickerLastTrade   `json:"c"`
	Volume                     TickerValues      `json:"v"`
	VolumeWeightedAveragePrice TickerValues      `json:"p"`
	NumberOfTrades             TickerTradeCounts `json:"t"`
	Low                        TickerValues      `json:"l"`
	High                       TickerValues      `json:"h"`
	OpeningPrice               types.Number      `json:"o"`
}

// TickerLevel is a ticker's best ask or bid
type TickerLevel struct {
	Price          types.Number
	WholeLotVolume types.Number
	LotVolume      types.Number
}

// UnmarshalJSON decodes a [price, whole lot volume, lot volume] array
func (t *TickerLevel) UnmarshalJSON(data []byte) error {
	fields := [3]any{&t.Price, &t.WholeLotVolume, &t.LotVolume}
	return unmarshalFixedArray(data, fields[:])
}

// TickerLastTrade is the last trade a ticker closed
type TickerLastTrade struct {
	Price     types.Number
	LotVolume types.Number
}

// UnmarshalJSON decodes a [price, lot volume] array
func (t *TickerLastTrade) UnmarshalJSON(data []byte) error {
	fields := [2]any{&t.Price, &t.LotVolume}
	return unmarshalFixedArray(data, fields[:])
}

// TickerValues holds a ticker value for today and for the last 24 hours
type TickerValues struct {
	Today       types.Number
	Last24Hours types.Number
}

// UnmarshalJSON decodes a [today, last 24 hours] array
func (t *TickerValues) UnmarshalJSON(data []byte) error {
	fields := [2]any{&t.Today, &t.Last24Hours}
	return unmarshalFixedArray(data, fields[:])
}

// TickerTradeCounts holds a ticker's number of trades today and in the last 24 hours
type TickerTradeCounts struct {
	Today       uint64
	Last24Hours uint64
}

// UnmarshalJSON decodes a [today, last 24 hours] array
func (t *TickerTradeCounts) UnmarshalJSON(data []byte) error {
	fields := [2]any{&t.Today, &t.Last24Hours}
	return unmarshalFixedArray(data, fields[:])
}

// OHLCDataRequest holds the parameters of Get OHLC Data
type OHLCDataRequest struct {
	Pair currency.Pair
	// Interval is one minute, the default, or 5, 15 or 30 minutes, 1 or 4 hours, 1 day, 1 week or 15 days
	Interval kline.Interval
	// Since returns the candles after it, for polling with the previous response's Last; only the 720 most recent
	// candles are served, whatever Since is
	Since time.Time
	// AssetClass is tokenized_asset for xStocks
	AssetClass   string
	DisplayNames bool
}

// OHLCDataResponse holds a pair's candles, keyed by pair name. The last candle is the current, uncommitted one
type OHLCDataResponse struct {
	Candles map[string][]Candle
	// Last is the open time of the last committed candle, to poll with as OHLCDataRequest's Since
	Last types.Time
}

// UnmarshalJSON decodes the pair keyed candles and the last field beside them
func (o *OHLCDataResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	o.Candles = make(map[string][]Candle, len(raw))
	for k, v := range raw {
		if k == "last" {
			if err := json.Unmarshal(v, &o.Last); err != nil {
				return err
			}
			continue
		}
		var candles []Candle
		if err := json.Unmarshal(v, &candles); err != nil {
			return err
		}
		o.Candles[k] = candles
	}
	return nil
}

// Candle is an OHLC candle
type Candle struct {
	Time                       types.Time
	Open                       types.Number
	High                       types.Number
	Low                        types.Number
	Close                      types.Number
	VolumeWeightedAveragePrice types.Number
	Volume                     types.Number
	TradeCount                 uint64
}

// UnmarshalJSON decodes a [time, open, high, low, close, vwap, volume, count] array
func (c *Candle) UnmarshalJSON(data []byte) error {
	fields := [8]any{&c.Time, &c.Open, &c.High, &c.Low, &c.Close, &c.VolumeWeightedAveragePrice, &c.Volume, &c.TradeCount}
	return unmarshalFixedArray(data, fields[:])
}

// OrderBookRequest holds the parameters of Get Order Book
type OrderBookRequest struct {
	Pair currency.Pair
	// Count is the number of levels per side, 1 to 500, and 100 when it is 0
	Count uint64
	// AssetClass is tokenized_asset for xStocks
	AssetClass   string
	DisplayNames bool
}

// OrderBook is a pair's level 2 order book
type OrderBook struct {
	Asks []OrderBookLevel `json:"asks"`
	Bids []OrderBookLevel `json:"bids"`
}

// OrderBookLevel is an order book price level
type OrderBookLevel struct {
	Price     types.Number
	Volume    types.Number
	Timestamp types.Time
}

// UnmarshalJSON decodes a [price, volume, timestamp] array
func (o *OrderBookLevel) UnmarshalJSON(data []byte) error {
	fields := [3]any{&o.Price, &o.Volume, &o.Timestamp}
	return unmarshalFixedArray(data, fields[:])
}

// Level3OrderBookRequest holds the parameters of Query L3 Order Book
type Level3OrderBookRequest struct {
	Pair currency.Pair
	// Depth is the number of price levels per side: 10, 25, 100, the default when it is not set, 250, 1000, or the
	// whole book when FullBook is set
	Depth    uint64
	FullBook bool
}

// Level3OrderBookResponse holds a pair's level 3 order book, which lists each order
type Level3OrderBookResponse struct {
	Pair string        `json:"pair"`
	Bids []Level3Order `json:"bids"`
	Asks []Level3Order `json:"asks"`
}

// Level3Order is an order resting in the book
type Level3Order struct {
	Price     types.Number `json:"price"`
	Quantity  types.Number `json:"qty"`
	OrderID   string       `json:"order_id"`
	Timestamp types.Time   `json:"timestamp"`
}

// GroupedOrderBookRequest holds the parameters of Get Grouped Order Book
type GroupedOrderBookRequest struct {
	Pair currency.Pair
	// Depth is the number of grouped levels per side: 10, the default when it is 0, 25, 100, 250 or 1000
	Depth uint64
	// Grouping is the number of ticks in each level: 1, the default when it is 0, 5, 10, 25, 50, 100, 250, 500 or 1000
	Grouping uint64
}

// GroupedOrderBookResponse holds a pair's order book grouped into levels of several ticks. Volume between levels
// counts towards the nearest passive level, so asks round up and bids round down
type GroupedOrderBookResponse struct {
	Pair     string              `json:"pair"`
	Grouping uint64              `json:"grouping"`
	Bids     []GroupedOrderLevel `json:"bids"`
	Asks     []GroupedOrderLevel `json:"asks"`
}

// GroupedOrderLevel is a grouped order book level
type GroupedOrderLevel struct {
	Price    types.Number `json:"price"`
	Quantity types.Number `json:"qty"`
}

// RecentTradesRequest holds the parameters of Get Recent Trades
type RecentTradesRequest struct {
	Pair currency.Pair
	// Since returns the trades after it, for polling with the previous response's Last
	Since time.Time
	// Count is the number of trades, up to 1000, and 1000 when it is 0
	Count uint64
	// AssetClass is tokenized_asset for xStocks
	AssetClass   string
	DisplayNames bool
}

// RecentTradesResponse holds a pair's recent trades, keyed by pair name
type RecentTradesResponse struct {
	Trades map[string][]RecentTrade
	// Last is the time of the last trade returned, to poll with as RecentTradesRequest's Since
	Last types.Time
}

// UnmarshalJSON decodes the pair keyed trades and the last field beside them
func (r *RecentTradesResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Trades = make(map[string][]RecentTrade, len(raw))
	for k, v := range raw {
		if k == "last" {
			if err := json.Unmarshal(v, &r.Last); err != nil {
				return err
			}
			continue
		}
		var trades []RecentTrade
		if err := json.Unmarshal(v, &trades); err != nil {
			return err
		}
		r.Trades[k] = trades
	}
	return nil
}

// RecentTrade is a public trade
type RecentTrade struct {
	Price  types.Number
	Volume types.Number
	Time   types.Time
	// Side is b for a buy or s for a sell
	Side string
	// OrderType is m for a market order or l for a limit order
	OrderType     string
	Miscellaneous string
	TradeID       uint64
}

// UnmarshalJSON decodes a [price, volume, time, side, order type, miscellaneous, trade ID] array
func (r *RecentTrade) UnmarshalJSON(data []byte) error {
	fields := [7]any{&r.Price, &r.Volume, &r.Time, &r.Side, &r.OrderType, &r.Miscellaneous, &r.TradeID}
	return unmarshalFixedArray(data, fields[:])
}

// RecentSpreadsRequest holds the parameters of Get Recent Spreads
type RecentSpreadsRequest struct {
	Pair currency.Pair
	// Since returns the spreads after it, for polling with the previous response's Last
	Since time.Time
	// AssetClass is tokenized_asset for xStocks
	AssetClass   string
	DisplayNames bool
}

// RecentSpreadsResponse holds a pair's last ~200 top of book spreads, keyed by pair name
type RecentSpreadsResponse struct {
	Spreads map[string][]Spread
	// Last is the time of the last spread returned, to poll with as RecentSpreadsRequest's Since
	Last types.Time
}

// UnmarshalJSON decodes the pair keyed spreads and the last field beside them
func (r *RecentSpreadsResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Spreads = make(map[string][]Spread, len(raw))
	for k, v := range raw {
		if k == "last" {
			if err := json.Unmarshal(v, &r.Last); err != nil {
				return err
			}
			continue
		}
		var spreads []Spread
		if err := json.Unmarshal(v, &spreads); err != nil {
			return err
		}
		r.Spreads[k] = spreads
	}
	return nil
}

// Spread is a top of book spread
type Spread struct {
	Time types.Time
	Bid  types.Number
	Ask  types.Number
}

// UnmarshalJSON decodes a [time, bid, ask] array
func (s *Spread) UnmarshalJSON(data []byte) error {
	fields := [3]any{&s.Time, &s.Bid, &s.Ask}
	return unmarshalFixedArray(data, fields[:])
}

// unmarshalFixedArray decodes a JSON array into fields, requiring exactly one element per field, so a payload of a
// different shape is reported rather than leaving fields unset
func unmarshalFixedArray(data []byte, fields []any) error {
	var elements []json.RawMessage
	if err := json.Unmarshal(data, &elements); err != nil {
		return err
	}
	if len(elements) != len(fields) {
		return fmt.Errorf("%w: got %d elements, expected %d", errUnexpectedLength, len(elements), len(fields))
	}
	for i := range elements {
		if err := json.Unmarshal(elements[i], fields[i]); err != nil {
			return err
		}
	}
	return nil
}
