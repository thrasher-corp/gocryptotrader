package hyperliquid

import (
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WsSubscriptionRequest subscribes to or unsubscribes from one feed
type WsSubscriptionRequest struct {
	Method       string         `json:"method"`
	Subscription WsSubscription `json:"subscription"`
}

// WsSubscription identifies a feed; the server echoes it in a canonical form, with the user lower case and every
// option present, and each option's zero value is its default, so the request and its echo decode to equal values
type WsSubscription struct {
	Type     string `json:"type"`
	Coin     string `json:"coin,omitempty"`
	Interval string `json:"interval,omitempty"`
	User     string `json:"user,omitempty"`
	// DEX selects a perpetual DEX; empty is the default DEX, whose allMids also include spot and outcome coins
	DEX string `json:"dex,omitempty"`
	// NumberOfSignificantFigures aggregates an l2Book to 2 to 5 significant figures; zero is full precision
	NumberOfSignificantFigures uint64 `json:"nSigFigs,omitempty"`
	// Mantissa further aggregates a 5 significant figure l2Book by 1, 2 or 5; zero is unaggregated
	Mantissa uint64 `json:"mantissa,omitempty"`
	// Fast selects the 5 level l2Book pushed about every half second, rather than 20 levels about every 5 seconds
	Fast bool `json:"fast,omitempty"`
	// AggregateByTime combines the partial fills of an order matched within one block
	AggregateByTime bool `json:"aggregateByTime,omitempty"`
	// IgnorePortfolioMargin omits portfolio margin collateral and available amounts from spotState; the server ignores the
	// isPortfolioMargin option the docs name
	IgnorePortfolioMargin bool `json:"ignorePortfolioMargin,omitempty"`
}

// WsSubscriptionResponse acknowledges a subscribe or unsubscribe request
type WsSubscriptionResponse struct {
	Method       string         `json:"method"`
	Subscription WsSubscription `json:"subscription"`
}

// WsQuotedRequest identifies the request a parse failure quotes: its method, a post's ID and a subscription request's
// subscription
type WsQuotedRequest struct {
	Method       string         `json:"method"`
	ID           uint64         `json:"id"`
	Subscription WsSubscription `json:"subscription"`
}

// websocketSubscriptionKey keys a subscription by the feed it selects: its Hyperliquid subscription type with the
// options its interval, levels and parameters select, and a market feed's asset and pair. Subscriptions selecting
// different feeds never match, so a variant can be held beside another and an unheld variant is never taken for a held
// one; a stored subscription's key also keeps the feed it subscribed, which its unsubscribe repeats
type websocketSubscriptionKey struct {
	*subscription.Subscription
	// options holds the subscription type and its options as its payload sends them, without the market's coin and the
	// account's address
	options WsSubscription
	// valid is false for a subscription naming no supported type or holding an invalid parameter, which matches nothing
	valid bool
	feed  WsSubscription
}

var _ subscription.MatchableKey = (*websocketSubscriptionKey)(nil)

// GetSubscription returns the subscription the key identifies
func (k *websocketSubscriptionKey) GetSubscription() *subscription.Subscription {
	return k.Subscription
}

// Match reports whether a key selects the same feed; a key of another type is compared by the feed its subscription
// selects
func (k *websocketSubscriptionKey) Match(other subscription.MatchableKey) bool {
	if other == nil {
		return false
	}
	theirs, ok := other.(*websocketSubscriptionKey)
	if !ok {
		sub := other.GetSubscription()
		if sub == nil {
			return false
		}
		theirs = newWebsocketSubscriptionKey(sub)
	}
	if theirs == nil || !k.valid || !theirs.valid || k.options != theirs.options {
		return false
	}
	if !websocketFeeds[k.options.Type].coin {
		return true
	}
	return k.Asset == theirs.Asset && k.Pairs.Equal(theirs.Pairs)
}

// String names the subscription and the options that set its feed apart from the subscription type's default feed
func (k *websocketSubscriptionKey) String() string {
	var options []string
	if k.options.DEX != "" {
		options = append(options, "dex="+k.options.DEX)
	}
	if k.options.NumberOfSignificantFigures != 0 {
		options = append(options, "nSigFigs="+strconv.FormatUint(k.options.NumberOfSignificantFigures, 10))
	}
	if k.options.Mantissa != 0 {
		options = append(options, "mantissa="+strconv.FormatUint(k.options.Mantissa, 10))
	}
	if k.options.Fast {
		options = append(options, "fast")
	}
	if k.options.AggregateByTime {
		options = append(options, "aggregateByTime")
	}
	if k.options.IgnorePortfolioMargin {
		options = append(options, "ignorePortfolioMargin")
	}
	name := subscription.ExactKey{Subscription: k.Subscription}.String()
	if len(options) == 0 {
		return name
	}
	return name + " [" + strings.Join(options, " ") + "]"
}

// websocketTemplateExpander expands subscription templates without validating them beside the held subscriptions: the
// feeds to unsubscribe would refuse an unheld variant of a held feed as exclusive with it rather than report it as not
// found, and a configuration replacing the held feeds would conflict with the variants a flush is about to remove
type websocketTemplateExpander struct {
	subscription.IExchange
}

// websocketPendingKey identifies a subscription operation awaiting its acknowledgement on one connection
type websocketPendingKey struct {
	authenticated bool
	subscription  WsSubscription
}

// websocketPendingOperation is a subscription operation awaiting its acknowledgement
type websocketPendingOperation struct {
	method        string
	connection    websocket.Connection
	subscription  *subscription.Subscription
	previousState subscription.State
	done          chan error
}

// websocketPosition is where an entry of a replaying feed sits: its time, then its trade ID, which orders the entries of
// one millisecond
type websocketPosition struct {
	time    time.Time
	tradeID uint64
}

// after reports whether the position comes after another
func (p websocketPosition) after(other websocketPosition) bool {
	return p.time.After(other.time) || p.time.Equal(other.time) && p.tradeID > other.tradeID
}

// websocketFillsFeed identifies an account's fills feed that replays a snapshot, userFills or userTwapSliceFills
type websocketFillsFeed struct {
	channel string
	user    string
}

// websocketWatermarks holds how far the websocket has relayed feeds that repeat entries: the newest entry of each coin's
// trades and of each account fills feed, which replay recent entries on subscribing, and each recent order's fills,
// which userEvents, userFills and userTwapSliceFills can each deliver
type websocketWatermarks struct {
	trades map[string]websocketPosition
	fills  map[websocketFillsFeed]websocketPosition
	orders websocketOrders
}

// websocketOrders holds what the account feeds have shown of each recent order, so its fills can report its cumulative
// execution; it holds at most maximumWebsocketOrders, forgetting the least recently touched first, and forgets an order
// untouched for websocketOrderLifetime, as an account without orderUpdates subscribed sends no row ending an order
type websocketOrders struct {
	byOrderID map[uint64]*websocketOrder
	// oldest and newest end the list of orders from the least to the most recently touched
	oldest, newest *websocketOrder
}

// websocketOrder is what an order's orderUpdates rows and fills have shown
type websocketOrder struct {
	orderID       uint64
	older, newer  *websocketOrder
	touched       time.Time
	rowSeen       bool
	originalSize  float64
	limitPrice    float64
	placed        time.Time
	sizeTolerance float64
	// covered is set once the order's fills are known to include every execution of the order
	covered  bool
	executed float64
	cost     float64
	tradeIDs map[uint64]struct{}
	// final is the order's final row when it arrived before the fills that complete its executed size
	final *order.Detail
}

// WsResponse is the outer frame of every server message
type WsResponse struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

// WsAllMids contains the mid price of every coin on one perpetual DEX; DEX is empty for the default DEX, which also
// includes spot and outcome coins
type WsAllMids struct {
	DEX  string                  `json:"dex"`
	Mids map[string]types.Number `json:"mids"`
}

// WsNotification contains a notification the frontend displays to a user, such as a fill message
type WsNotification struct {
	Notification string `json:"notification"`
}

// WsWebData3 contains the aggregate account information the frontend displays
type WsWebData3 struct {
	UserState WsWebData3UserState `json:"userState"`
	// PerpetualDEXStates contains one state per perpetual DEX
	PerpetualDEXStates []WsPerpetualDEXState `json:"perpDexStates"`
}

// WsWebData3UserState contains a user's API wallet, ledger and account mode; a vault has no account mode
type WsWebData3UserState struct {
	AgentAddress          string             `json:"agentAddress"`
	AgentValidUntil       types.Time         `json:"agentValidUntil"`
	CumulativeLedger      types.Number       `json:"cumLedger"`
	ServerTime            types.Time         `json:"serverTime"`
	IsVault               bool               `json:"isVault"`
	User                  string             `json:"user"`
	OptOutOfSpotDusting   bool               `json:"optOutOfSpotDusting"`
	DEXAbstractionEnabled bool               `json:"dexAbstractionEnabled"`
	Abstraction           AccountAbstraction `json:"abstraction"`
}

// WsPerpetualDEXState contains one perpetual DEX's vault equity, capped markets and the vaults a user leads
type WsPerpetualDEXState struct {
	TotalVaultEquity            types.Number     `json:"totalVaultEquity"`
	PerpetualsAtOpenInterestCap []string         `json:"perpsAtOpenInterestCap"`
	LeadingVaults               []WsLeadingVault `json:"leadingVaults"`
}

// WsLeadingVault contains a vault a user leads
type WsLeadingVault struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

// WsTWAPStates contains a user's active TWAP orders on one perpetual DEX
type WsTWAPStates struct {
	DEX    string           `json:"dex"`
	User   string           `json:"user"`
	States []TWAPStateEntry `json:"states"`
}

// TWAPStateEntry contains an active TWAP order and its ID
type TWAPStateEntry struct {
	TWAPID uint64
	State  TWAPState
}

// UnmarshalJSON decodes a [twapId, state] tuple
func (t *TWAPStateEntry) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &t.TWAPID, &t.State)
}

// TWAPState contains a TWAP order's parameters and progress
type TWAPState struct {
	Coin             string       `json:"coin"`
	User             string       `json:"user"`
	Side             string       `json:"side"`
	Size             types.Number `json:"sz"`
	ExecutedSize     types.Number `json:"executedSz"`
	ExecutedNotional types.Number `json:"executedNtl"`
	Minutes          uint64       `json:"minutes"`
	ReduceOnly       bool         `json:"reduceOnly"`
	Randomise        bool         `json:"randomize"`
	Timestamp        types.Time   `json:"timestamp"`
	// Trigger is set for a TWAP order that starts once the price crosses a level
	Trigger   *TWAPTrigger `json:"trigger"`
	StopPrice types.Number `json:"stopPx"`
}

// TWAPTrigger contains the price a waiting TWAP order starts at; Above is true when it starts once the price rises
// above Price
type TWAPTrigger struct {
	Price types.Number `json:"px"`
	Above bool         `json:"above"`
}

// WsClearinghouseState contains a user's perpetual margin state on one DEX; DEX is empty for the default DEX
type WsClearinghouseState struct {
	DEX                string                     `json:"dex"`
	User               string                     `json:"user"`
	ClearinghouseState ClearinghouseStateResponse `json:"clearinghouseState"`
}

// WsOpenOrders contains a user's open orders on one perpetual DEX, newest first; for an account with very many open
// orders it can be limited to 100, as frontendOpenOrders is
type WsOpenOrders struct {
	DEX    string              `json:"dex"`
	User   string              `json:"user"`
	Orders []FrontendOpenOrder `json:"orders"`
}

// WsL2Book contains an L2 book snapshot from the l2Book feed; Fast is set on the 5 level fast book
type WsL2Book struct {
	L2BookResponse
	Fast bool `json:"fast"`
}

// WsOrder contains an order update from the orderUpdates feed, which omits the order type and time in force
type WsOrder struct {
	Order           BasicOrder `json:"order"`
	Status          string     `json:"status"`
	StatusTimestamp types.Time `json:"statusTimestamp"`
}

// WsUserEvent contains one event from the userEvents feed, sent on the user channel; exactly one field is set
type WsUserEvent struct {
	Fills          []Fill            `json:"fills"`
	Funding        *WsUserFunding    `json:"funding"`
	Liquidation    *WsLiquidation    `json:"liquidation"`
	NonUserCancel  []WsNonUserCancel `json:"nonUserCancel"`
	TWAPSliceFills []TWAPSliceFill   `json:"twapSliceFills"`
}

// WsUserFunding contains one funding payment; USDC is negative when the user paid funding
type WsUserFunding struct {
	Time        types.Time   `json:"time"`
	Coin        string       `json:"coin"`
	USDC        types.Number `json:"usdc"`
	SignedSize  types.Number `json:"szi"`
	FundingRate types.Number `json:"fundingRate"`
	// NumberOfSamples is the number of hourly payments a daily aggregate holds, and zero for an hourly payment
	NumberOfSamples uint64 `json:"nSamples"`
}

// WsLiquidation contains a liquidation of a user's account
type WsLiquidation struct {
	LiquidationID              uint64       `json:"lid"`
	Liquidator                 string       `json:"liquidator"`
	LiquidatedUser             string       `json:"liquidated_user"`
	LiquidatedNotionalPosition types.Number `json:"liquidated_ntl_pos"`
	LiquidatedAccountValue     types.Number `json:"liquidated_account_value"`
}

// WsNonUserCancel identifies an order the exchange cancelled
type WsNonUserCancel struct {
	Coin    string `json:"coin"`
	OrderID uint64 `json:"oid"`
}

// WsUserFills contains fills from the userFills feed; the first message is a snapshot of recent fills
type WsUserFills struct {
	IsSnapshot bool   `json:"isSnapshot"`
	User       string `json:"user"`
	Fills      []Fill `json:"fills"`
}

// WsUserFundings contains funding payments from the userFundings feed; the first message is a snapshot of recent
// payments
type WsUserFundings struct {
	IsSnapshot bool            `json:"isSnapshot"`
	User       string          `json:"user"`
	Fundings   []WsUserFunding `json:"fundings"`
}

// WsUserNonFundingLedgerUpdates contains ledger updates other than funding payments; the first message is a snapshot of
// recent updates
type WsUserNonFundingLedgerUpdates struct {
	IsSnapshot              bool               `json:"isSnapshot"`
	User                    string             `json:"user"`
	NonFundingLedgerUpdates []UserLedgerUpdate `json:"nonFundingLedgerUpdates"`
}

// WsPerpetualAssetContext contains a perpetual market's context from the activeAssetCtx feed
type WsPerpetualAssetContext struct {
	Coin    string                `json:"coin"`
	Context PerpetualAssetContext `json:"ctx"`
}

// WsSpotAssetContext contains a spot or outcome market's context from the activeAssetCtx feed, sent on the
// activeSpotAssetCtx channel
type WsSpotAssetContext struct {
	Coin    string           `json:"coin"`
	Context SpotAssetContext `json:"ctx"`
}

// WsUserTWAPSliceFills contains TWAP slice fills from the userTwapSliceFills feed; the first message is a snapshot of
// recent slice fills
type WsUserTWAPSliceFills struct {
	IsSnapshot     bool            `json:"isSnapshot"`
	User           string          `json:"user"`
	TWAPSliceFills []TWAPSliceFill `json:"twapSliceFills"`
}

// WsUserTWAPHistory contains TWAP status changes from the userTwapHistory feed; the first message is a snapshot of
// recent changes
type WsUserTWAPHistory struct {
	IsSnapshot bool               `json:"isSnapshot"`
	User       string             `json:"user"`
	History    []TWAPHistoryEntry `json:"history"`
}

// TWAPHistoryEntry contains a TWAP order's state when its status changed
type TWAPHistoryEntry struct {
	// Time is sent in seconds, unlike the state's millisecond timestamp
	Time   types.Time `json:"time"`
	State  TWAPState  `json:"state"`
	Status TWAPStatus `json:"status"`
	TWAPID uint64     `json:"twapId"`
}

// TWAPStatus contains a TWAP order's status: activated, waitingForTrigger, finished, stopped, terminated or error;
// Description explains an error
type TWAPStatus struct {
	Status      string `json:"status"`
	Description string `json:"description"`
}

// WsBestBidOffer contains a coin's best bid and ask; either is nil when its side of the book is empty
type WsBestBidOffer struct {
	Coin string     `json:"coin"`
	Time types.Time `json:"time"`
	// BestBidOffer contains the bid then the ask
	BestBidOffer [2]*L2Level `json:"bbo"`
}

// WsSpotState contains a user's spot balances
type WsSpotState struct {
	User      string                         `json:"user"`
	SpotState SpotClearinghouseStateResponse `json:"spotState"`
}

// WsAllDEXsClearinghouseState contains a user's perpetual margin state on every DEX
type WsAllDEXsClearinghouseState struct {
	User                string                  `json:"user"`
	ClearinghouseStates []DEXClearinghouseState `json:"clearinghouseStates"`
}

// DEXClearinghouseState contains a user's perpetual margin state on one DEX; DEX is empty for the default DEX
type DEXClearinghouseState struct {
	DEX                string
	ClearinghouseState ClearinghouseStateResponse
}

// UnmarshalJSON decodes a [dex, state] tuple
func (d *DEXClearinghouseState) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &d.DEX, &d.ClearinghouseState)
}

// WsAllDEXsAssetContexts contains every perpetual DEX's asset contexts, each in its DEX's metadata universe order
type WsAllDEXsAssetContexts struct {
	Contexts []DEXAssetContexts `json:"ctxs"`
}

// DEXAssetContexts contains one perpetual DEX's asset contexts; DEX is empty for the default DEX
type DEXAssetContexts struct {
	DEX           string
	AssetContexts []PerpetualAssetContext
}

// UnmarshalJSON decodes a [dex, contexts] tuple
func (d *DEXAssetContexts) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &d.DEX, &d.AssetContexts)
}

// WsOutcomeMetaUpdate contains one outcome or question change; exactly one field is set
type WsOutcomeMetaUpdate struct {
	OutcomeCreated  *OutcomeSpecification `json:"outcomeCreated"`
	OutcomeSettled  *uint64               `json:"outcomeSettled"`
	QuestionUpdated *OutcomeQuestion      `json:"questionUpdated"`
	QuestionSettled *uint64               `json:"questionSettled"`
}

// WsFastAssetContext contains a coin's changed mark and mid prices from the fastAssetCtxs feed; a field that is not set
// is unchanged, and MidPriceSet with a zero MidPrice means the book has no mid price
type WsFastAssetContext struct {
	MarkPrice    types.Number
	MidPrice     types.Number
	MarkPriceSet bool
	MidPriceSet  bool
}

// UnmarshalJSON records which prices are present, so updates can be merged into the previous snapshot
func (w *WsFastAssetContext) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if value, ok := fields["markPx"]; ok {
		w.MarkPriceSet = true
		if err := json.Unmarshal(value, &w.MarkPrice); err != nil {
			return err
		}
	}
	if value, ok := fields["midPx"]; ok {
		w.MidPriceSet = true
		if err := json.Unmarshal(value, &w.MidPrice); err != nil {
			return err
		}
	}
	return nil
}

// WsPostRequest sends an info request or a signed action over the websocket
type WsPostRequest struct {
	Method string `json:"method"`
	// ID must be unique among the client's outstanding requests, as Hyperliquid does not check it
	ID      uint64            `json:"id"`
	Request WsPostRequestBody `json:"request"`
}

// WsPostRequestBody contains a post request's type, info or action, and its payload, an info request body or a signed
// action request
type WsPostRequestBody struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// WsPostResponse contains a post request's ID and outcome
type WsPostResponse struct {
	ID       uint64             `json:"id"`
	Response WsPostResponseBody `json:"response"`
}

// WsPostResponseBody contains a post response's type, info, action or error, and its payload: a WsPostInfoPayload for
// info, an exchange action response for action and an HTTP status for error
type WsPostResponseBody struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// WsPostInfoPayload contains an info request's type and its response, as the info endpoint returns it
type WsPostInfoPayload struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
