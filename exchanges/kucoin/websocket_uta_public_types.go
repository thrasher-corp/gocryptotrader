package kucoin

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UTAWebsocketTicker is a UTA ticker push. The docs say it is sent when a trade occurs only, but spot pushes also arrive
// when the best bid or ask changes, carrying the last trade again
type UTAWebsocketTicker struct {
	Symbol string `json:"s"`
	// Sequence increases with each push but is not guaranteed to be continuous
	Sequence        uint64       `json:"E"`
	BestBidPrice    types.Number `json:"b"`
	BestBidSize     types.Number `json:"B"`
	BestAskPrice    types.Number `json:"a"`
	BestAskSize     types.Number `json:"A"`
	LastTradedPrice types.Number `json:"l"`
	// Side is the taker side: upper case on spot pushes and lower case on futures pushes
	Side           string       `json:"S"`
	LastTradedSize types.Number `json:"q"`
	// MatchTimestamp is when the best bid or ask changed for spot, and when the trade that changed them matched for
	// futures
	MatchTimestamp types.Time `json:"M"`
}

// UTAWebsocketKline is a UTA kline push
type UTAWebsocketKline struct {
	Interval   string       `json:"i"`
	Symbol     string       `json:"s"`
	StartTime  types.Time   `json:"O"`
	EndTime    types.Time   `json:"C"`
	OpenPrice  types.Number `json:"o"`
	ClosePrice types.Number `json:"c"`
	HighPrice  types.Number `json:"h"`
	LowPrice   types.Number `json:"l"`
	// TransactionVolume is in contracts for futures
	TransactionVolume types.Number `json:"v"`
	TransactionAmount types.Number `json:"a"`
	// StartIndicator is true on the first push of a candle's period
	StartIndicator bool `json:"S"`
}

// UTAWebsocketTrade is a UTA public trade push
type UTAWebsocketTrade struct {
	Symbol   string       `json:"s"`
	Sequence uint64       `json:"E"`
	TradeID  string       `json:"ti"`
	Price    types.Number `json:"p"`
	Size     types.Number `json:"q"`
	// Side is the taker side
	Side string `json:"S"`
	// RPI is whether a retail price improvement order filled, on futures trades
	RPI            bool       `json:"rpi"`
	MatchTimestamp types.Time `json:"M"`
}

// UTAWebsocketOrderbook is a UTA orderbook push, a snapshot or a delta as its push's update type says. The sequences
// are int64 to match GoCryptoTrader's orderbook update IDs
type UTAWebsocketOrderbook struct {
	// SequenceStart and SequenceEnd bound the changes in the push; they are equal on snapshots
	SequenceStart  int64                        `json:"O"`
	SequenceEnd    int64                        `json:"C"`
	MatchTimestamp types.Time                   `json:"M"`
	Symbol         string                       `json:"s"`
	Bids           []UTAWebsocketOrderbookLevel `json:"b"`
	Asks           []UTAWebsocketOrderbookLevel `json:"a"`
}

// UTAWebsocketOrderbookLevel is a price level of a UTA orderbook push, sent as an array of two strings, or of three when
// the subscription includes retail price improvement orders
type UTAWebsocketOrderbookLevel struct {
	Price types.Number
	// Size excludes retail price improvement orders when the push includes them
	Size    types.Number
	RPISize types.Number
}

// UnmarshalJSON decodes a level from its [price, size] or [price, size, RPI size] array
func (l *UTAWebsocketOrderbookLevel) UnmarshalJSON(data []byte) error {
	var values []types.Number
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	if len(values) != 2 && len(values) != 3 {
		return fmt.Errorf("%w: orderbook level holds %d values, expected 2 or 3", common.ErrMalformedData, len(values))
	}
	*l = UTAWebsocketOrderbookLevel{Price: values[0], Size: values[1]}
	if len(values) == 3 {
		l.RPISize = values[2]
	}
	return nil
}

// UTAWebsocketMarkPrice is a UTA mark price push, sent every second
type UTAWebsocketMarkPrice struct {
	Symbol       string       `json:"s"`
	MarkPrice    types.Number `json:"mp"`
	IndexPrice   types.Number `json:"ip"`
	OpenInterest types.Number `json:"oi"`
	// Timestamp is when the mark price was calculated
	Timestamp types.Time `json:"ts"`
}

// UTAWebsocketFundingFee is a UTA funding fee rate push, sent every minute, and an element of the all funding fee rates
// push
type UTAWebsocketFundingFee struct {
	Symbol         string       `json:"s"`
	FundingFeeRate types.Number `json:"fr"`
	// FundingTime is the last funding fee settlement
	FundingTime     types.Time `json:"ft"`
	NextFundingTime types.Time `json:"nt"`
	// Granularity is the settlement interval in milliseconds
	Granularity        uint64       `json:"gl"`
	FundingRateCap     types.Number `json:"fc"`
	FundingRateFloor   types.Number `json:"ff"`
	LastFundingFeeRate types.Number `json:"lfr"`
}

// UTAWebsocketCallAuctionInfo is a UTA call auction push for a symbol in its call auction phase. Prices and sizes
// arrive as bare numbers in the docs' example although the docs' schema types them as strings
type UTAWebsocketCallAuctionInfo struct {
	Symbol string `json:"s"`
	// EstimatedPrice is keyed eq in the docs' example, which KuCoin's SDK follows, and ep in the docs' schema
	EstimatedPrice          types.Number `json:"eq"`
	EstimatedSize           types.Number `json:"es"`
	SellOrderRangeLowPrice  types.Number `json:"slp"`
	SellOrderRangeHighPrice types.Number `json:"shp"`
	BuyOrderRangeLowPrice   types.Number `json:"blp"`
	BuyOrderRangeHighPrice  types.Number `json:"bhp"`
	Timestamp               types.Time   `json:"ts"`
}
