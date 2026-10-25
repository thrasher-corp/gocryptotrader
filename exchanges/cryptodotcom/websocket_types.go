package cryptodotcom

import (
	"fmt"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Crypto.com's market data channels besides GoCryptoTrader's ticker, orderbook, trades and candles channels, named as
// Crypto.com names them, for configured subscriptions: the index channel is subscribed for a derivative's underlying
// index, the funding channels for perpetual swaps and the settlement channel for futures
const (
	IndexPriceChannel           = "index"
	MarkPriceChannel            = "mark"
	SettlementPriceChannel      = "settlement"
	FundingRateChannel          = "funding"
	EstimatedFundingRateChannel = "estimatedfunding"
)

// Crypto.com's user channels besides GoCryptoTrader's myOrders, myTrades and myAccount channels, which are its
// user.order, user.trade and user.balance channels, named as Crypto.com names them, for configured subscriptions, which
// must be authenticated. The order and trade channels serve every instrument, or each pair of the asset a subscription
// names
const (
	AdvancedOrdersChannel   = "user.advance.order"
	TransactionsChannel     = "user.transactions"
	PositionsChannel        = "user.positions"
	AccountRiskChannel      = "user.account_risk"
	PositionBalanceChannel  = "user.position_balance"
	OTCQuoteRequestsChannel = "user.otc_qr.requests"
	OTCQuotesChannel        = "user.otc_qr.quotes"
	OTCDealsChannel         = "user.otc.deals"
)

// Book subscription types: a full snapshot of the book every 500 milliseconds, or one snapshot followed by deltas every
// 10 or 100 milliseconds
const (
	BookSubscriptionTypeSnapshot          = "SNAPSHOT"
	BookSubscriptionTypeSnapshotAndUpdate = "SNAPSHOT_AND_UPDATE"
)

// cancelOnDisconnectRequest is the params of a set-cancel-on-disconnect request
type cancelOnDisconnectRequest struct {
	Scope string `json:"scope"`
}

// CancelOnDisconnectResponse is the scope of a connection's cancel on disconnect
type CancelOnDisconnectResponse struct {
	Scope string `json:"scope"`
}

// websocketChannelPush is a channel's push: the instrument it is for, the subscription it answers, the channel it comes
// from, the candlestick channel's interval, the account the user.account_risk channel reports, and its data
type websocketChannelPush struct {
	InstrumentName string          `json:"instrument_name"`
	Subscription   string          `json:"subscription"`
	Channel        string          `json:"channel"`
	Interval       string          `json:"interval"`
	AccountID      string          `json:"account_id"`
	Data           json.RawMessage `json:"data"`
}

// WebsocketBookLevels is one side of an order book
type WebsocketBookLevels orderbook.Levels

// UnmarshalJSON decodes a side of a book from its levels' [price, quantity, order count] arrays, whose values are quoted
func (l *WebsocketBookLevels) UnmarshalJSON(data []byte) error {
	var levels [][3]types.Number
	if err := json.Unmarshal(data, &levels); err != nil {
		return err
	}
	*l = make(WebsocketBookLevels, len(levels))
	for i := range levels {
		(*l)[i] = orderbook.Level{Price: levels[i][0].Float64(), Amount: levels[i][1].Float64(), OrderCount: levels[i][2].Int64()}
	}
	return nil
}

// WebsocketBook is an order book from the book channel: a snapshot of each side's levels, or a delta, whose
// Update holds the levels changed since the update PreviousUpdateID names; a delta's level of zero quantity is deleted
type WebsocketBook struct {
	Asks   WebsocketBookLevels  `json:"asks"`
	Bids   WebsocketBookLevels  `json:"bids"`
	Update *WebsocketBookUpdate `json:"update"`
	// PublishTime is when the book was published and UpdateTime when it was last updated, a little before
	PublishTime types.Time `json:"t"`
	UpdateTime  types.Time `json:"tt"`
	// UpdateID is an int64, the type GoCryptoTrader's order books keep update IDs in
	UpdateID Integer `json:"u"`
	// PreviousUpdateID is the update a delta follows, an int64 to be compared with the update ID a book keeps
	PreviousUpdateID Integer `json:"pu"`
	// Checksum is a signed 32-bit checksum of the book; Crypto.com does not document its algorithm, and the common
	// checksums of the book's levels in the layouts tried do not reproduce it, so it is not verified
	Checksum Integer `json:"cs"`
}

// Integer is an int64 Crypto.com's schema gives as a quoted integer and its pushes send as a bare one; it is decoded
// from either without passing through a float, which holds integers exactly only up to 2^53
type Integer int64

// UnmarshalJSON decodes an integer from a JSON string or a bare JSON integer, leaving it unchanged for null
func (i *Integer) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	s := string(data)
	if len(data) != 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("%w: %w", errInvalidInteger, err)
		}
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: %w", errInvalidInteger, err)
	}
	*i = Integer(v)
	return nil
}

// WebsocketBookUpdate is the levels a book delta changes
type WebsocketBookUpdate struct {
	Asks WebsocketBookLevels `json:"asks"`
	Bids WebsocketBookLevels `json:"bids"`
}

// WebsocketTicker is an instrument's ticker from the ticker channel
type WebsocketTicker struct {
	InstrumentName string       `json:"i"`
	High24Hour     types.Number `json:"h"`
	Low24Hour      types.Number `json:"l"`
	LastPrice      types.Number `json:"a"`
	// PriceChange24Hour is the change in the last price over 24 hours, as a fraction of the price 24 hours before
	PriceChange24Hour types.Number `json:"c"`
	BestBidPrice      types.Number `json:"b"`
	BestBidSize       types.Number `json:"bs"`
	BestAskPrice      types.Number `json:"k"`
	BestAskSize       types.Number `json:"ks"`
	Volume24Hour      types.Number `json:"v"`
	// VolumeValue24Hour is the value of the volume traded over 24 hours, in USD whatever the instrument's quote currency
	VolumeValue24Hour types.Number `json:"vv"`
	OpenInterest      types.Number `json:"oi"`
	Time              types.Time   `json:"t"`
}

// WebsocketTrade is a public trade from the trade channel
type WebsocketTrade struct {
	TradeID  string       `json:"d"`
	Time     types.Time   `json:"t"`
	Price    types.Number `json:"p"`
	Quantity types.Number `json:"q"`
	// Side is the side of the trade's taker
	Side           string `json:"s"`
	InstrumentName string `json:"i"`
	// MatchID identifies the match the trade is part of, which the match's other trades share
	MatchID string `json:"m"`
}

// WebsocketCandle is a candle from the candlestick channel; its time is the start of its interval, and its
// update time when it last changed
type WebsocketCandle struct {
	Open       types.Number `json:"o"`
	High       types.Number `json:"h"`
	Low        types.Number `json:"l"`
	Close      types.Number `json:"c"`
	Volume     types.Number `json:"v"`
	Time       types.Time   `json:"t"`
	UpdateTime types.Time   `json:"ut"`
}

// WebsocketIndexPrice is an index's price from the index channel, such as BTCUSD-INDEX, the underlying index of
// the BTCUSD derivatives
type WebsocketIndexPrice struct {
	IndexName  string       `json:"-"`
	IndexPrice types.Number `json:"v"`
	Time       types.Time   `json:"t"`
}

// WebsocketMarkPrice is an instrument's mark price from the mark price channel
type WebsocketMarkPrice struct {
	Pair      currency.Pair `json:"-"`
	AssetType asset.Item    `json:"-"`
	MarkPrice types.Number  `json:"v"`
	Time      types.Time    `json:"t"`
}

// WebsocketSettlementPrice is a future's settlement price from the settlement channel, pushed when it expires
type WebsocketSettlementPrice struct {
	Pair            currency.Pair `json:"-"`
	AssetType       asset.Item    `json:"-"`
	SettlementPrice types.Number  `json:"v"`
	Time            types.Time    `json:"t"`
}

// WebsocketFundingRate is a perpetual swap's hourly funding rate from the funding channel, which is settled at
// the end of the hour
type WebsocketFundingRate struct {
	Pair        currency.Pair `json:"-"`
	AssetType   asset.Item    `json:"-"`
	FundingRate types.Number  `json:"v"`
	Time        types.Time    `json:"t"`
}

// WebsocketEstimatedFundingRate is a perpetual swap's estimated funding rate from the estimated funding channel:
// the hourly rate expected to be settled at the end of each hour of the next funding interval, the intervals being the
// four hours from 00:00, 04:00, 08:00, 12:00, 16:00 and 20:00 UTC
type WebsocketEstimatedFundingRate struct {
	Pair                 currency.Pair `json:"-"`
	AssetType            asset.Item    `json:"-"`
	EstimatedFundingRate types.Number  `json:"v"`
	Time                 types.Time    `json:"t"`
}

// WebsocketUserTrade is one of the account's trades from the user.trade channel, which also reports when the
// trade was transacted, in nanoseconds
type WebsocketUserTrade struct {
	UserTradeResponse
	TransactionTime types.Time `json:"transaction_time"`
}

// WebsocketAccountRisk is a snapshot of the account's risk from the user.account_risk channel, which Crypto.com
// pushes regularly: the account's margin totals, collateral balances and positions, and those of each isolated position
type WebsocketAccountRisk struct {
	// AccountID is the account the push names
	AccountID string `json:"-"`
	BalanceSummary
	Balances          []CollateralBalance           `json:"balances"`
	Positions         []PositionResponse            `json:"positions"`
	IsolatedPositions []AccountRiskIsolatedPosition `json:"isolated_positions"`
}

// AccountRiskIsolatedPosition is the risk of an isolated position, which is margined apart from the account
type AccountRiskIsolatedPosition struct {
	BalanceSummary
	IsolationID   string              `json:"isolation_id"`
	Leverage      types.Number        `json:"leverage"`
	IsolationType string              `json:"isolation_type"`
	Balances      []CollateralBalance `json:"balances"`
	Positions     []PositionResponse  `json:"positions"`
}

// WebsocketPositionBalance is the account's collateral balances and positions from the user.position_balance
// channel, pushed as they change; its balances carry only their instrument names, quantities and update times, and its
// positions only their core fields
type WebsocketPositionBalance struct {
	Balances          []CollateralBalance               `json:"balances"`
	Positions         []PositionResponse                `json:"positions"`
	IsolatedPositions []PositionBalanceIsolatedPosition `json:"isolated_positions"`
}

// PositionBalanceIsolatedPosition is the collateral balances and positions of an isolated position
type PositionBalanceIsolatedPosition struct {
	IsolationID   string              `json:"isolation_id"`
	Leverage      types.Number        `json:"leverage"`
	IsolationType string              `json:"isolation_type"`
	Balances      []CollateralBalance `json:"balances"`
	Positions     []PositionResponse  `json:"positions"`
}
