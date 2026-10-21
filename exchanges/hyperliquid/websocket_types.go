package hyperliquid

import (
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WebsocketSubscriptionRequest subscribes to or unsubscribes from one feed
type WebsocketSubscriptionRequest struct {
	Method       string                `json:"method"`
	Subscription WebsocketSubscription `json:"subscription"`
}

// WebsocketSubscription identifies a feed; the server echoes it in a canonical form, with the user lower case and every
// option present, and each option's zero value is its default, so the request and its echo decode to equal values
type WebsocketSubscription struct {
	Type string `json:"type"`
	// Coin is the case-sensitive market name, such as BTC, kPEPE, xyz:XYZ100 or @107, rather than a currency code
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

// WebsocketSubscriptionResponse acknowledges a subscribe or unsubscribe request
type WebsocketSubscriptionResponse struct {
	Method       string                `json:"method"`
	Subscription WebsocketSubscription `json:"subscription"`
}

// WebsocketQuotedRequest identifies the request a parse failure quotes: its method, a post's ID and a subscription
// request's subscription
type WebsocketQuotedRequest struct {
	Method       string                `json:"method"`
	ID           uint64                `json:"id"`
	Subscription WebsocketSubscription `json:"subscription"`
}

// websocketSubscriptionKey keys a subscription by the feed it selects: its Hyperliquid subscription type with the
// options its interval, levels and parameters select, and a market feed's asset and pair. Subscriptions selecting
// different feeds never match, so a variant can be held beside another and an unheld variant is never taken for a held
// one; a stored subscription's key also keeps the feed it subscribed, which its unsubscribe repeats
type websocketSubscriptionKey struct {
	*subscription.Subscription
	// options holds the subscription type and its options as its payload sends them, without the market's coin and the
	// account's address
	options WebsocketSubscription
	// valid is false for a subscription naming no supported type or holding an invalid parameter, which matches nothing
	valid bool
	feed  WebsocketSubscription
}

var _ subscription.MatchableKey = (*websocketSubscriptionKey)(nil)

// websocketFeedIdentity is a comparable form of a websocketSubscriptionKey naming at most one market: two such keys match
// when their identities are equal
type websocketFeedIdentity struct {
	options     WebsocketSubscription
	assetType   asset.Item
	base, quote *currency.Item
}

// identity returns the key's comparable form, and false for a key that matches nothing or names other than one market
func (k *websocketSubscriptionKey) identity() (websocketFeedIdentity, bool) {
	if !k.valid {
		return websocketFeedIdentity{}, false
	}
	id := websocketFeedIdentity{options: k.options}
	if websocketFeeds[k.options.Type].coin {
		if len(k.Pairs) != 1 {
			return websocketFeedIdentity{}, false
		}
		id.assetType, id.base, id.quote = k.Asset, k.Pairs[0].Base.Item, k.Pairs[0].Quote.Item
	}
	return id, true
}

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

// CanUseAuthenticatedWebsocketEndpoints expands account feeds whatever the account connection's state, as subscribing
// one opens it; callers leave account feeds out when they cannot be subscribed
func (websocketTemplateExpander) CanUseAuthenticatedWebsocketEndpoints() bool {
	return true
}

// websocketPendingKey identifies a subscription operation awaiting its acknowledgement on one connection
type websocketPendingKey struct {
	connection   websocket.Connection
	subscription WebsocketSubscription
}

// websocketPendingOperation is a subscription operation awaiting its acknowledgement
type websocketPendingOperation struct {
	method        string
	connection    websocket.Connection
	subscription  *subscription.Subscription
	previousState subscription.State
	done          chan error
	// abandoned marks a subscribe whose caller gave up on it once sent, and whose subscription was rolled back; cleanup
	// marks an abandoned subscribe Hyperliquid acknowledged, whose feed is being unsubscribed. websocketPendingMu guards
	// both
	abandoned, cleanup bool
}

// awaitedMethod returns the method whose acknowledgement settles the operation; the caller holds websocketPendingMu
func (p *websocketPendingOperation) awaitedMethod() string {
	if p.cleanup {
		return wsMethodUnsubscribe
	}
	return p.method
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
	// fee is the order's fills' fees in feeAsset, which a fill in another fee token leaves unknown
	fee        float64
	feeAsset   currency.Code
	feeUnknown bool
	tradeIDs   map[uint64]struct{}
	// final is the order's final row when it arrived before the fills that complete its executed size
	final *order.Detail
}

// WebsocketMessage is the outer frame of every server message
type WebsocketMessage struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

// WebsocketAllMids contains the mid price of every coin on one perpetual DEX; DEX is empty for the default DEX, which
// also includes spot and outcome coins
type WebsocketAllMids struct {
	DEX       string                  `json:"dex"`
	MidPrices map[string]types.Number `json:"mids"`
}

// WebsocketNotification contains a notification the frontend displays to a user, such as a fill message
type WebsocketNotification struct {
	Notification string `json:"notification"`
}

// WebsocketWebData3 contains the aggregate account information the frontend displays
type WebsocketWebData3 struct {
	UserState WebsocketWebData3UserState `json:"userState"`
	// PerpetualDEXStates contains one state per perpetual DEX
	PerpetualDEXStates []WebsocketPerpetualDEXState `json:"perpDexStates"`
}

// WebsocketWebData3UserState contains a user's API wallet, ledger and account mode; a vault has no account mode
type WebsocketWebData3UserState struct {
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

// WebsocketPerpetualDEXState contains one perpetual DEX's vault equity, capped markets and the vaults a user leads
type WebsocketPerpetualDEXState struct {
	TotalVaultEquity            types.Number            `json:"totalVaultEquity"`
	PerpetualsAtOpenInterestCap []string                `json:"perpsAtOpenInterestCap"`
	LeadingVaults               []WebsocketLeadingVault `json:"leadingVaults"`
}

// WebsocketLeadingVault contains a vault a user leads
type WebsocketLeadingVault struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

// WebsocketTWAPStates contains a user's active TWAP orders on one perpetual DEX
type WebsocketTWAPStates struct {
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
	// Coin is the case-sensitive market name, such as BTC, kPEPE, xyz:XYZ100 or @107, rather than a currency code
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

// WebsocketClearinghouseState contains a user's perpetual margin state on one DEX; DEX is empty for the default DEX
type WebsocketClearinghouseState struct {
	DEX                string                     `json:"dex"`
	User               string                     `json:"user"`
	ClearinghouseState ClearinghouseStateResponse `json:"clearinghouseState"`
}

// WebsocketOpenOrders contains a user's open orders on one perpetual DEX, newest first; for an account with very many
// open orders it can be limited to 100, as frontendOpenOrders is
type WebsocketOpenOrders struct {
	DEX    string                      `json:"dex"`
	User   string                      `json:"user"`
	Orders []FrontendOpenOrderResponse `json:"orders"`
}

// WebsocketL2Book contains an L2 book snapshot from the l2Book feed; Fast is set on the 5 level fast book
type WebsocketL2Book struct {
	L2BookResponse
	Fast bool `json:"fast"`
}

// WebsocketOrder contains an order update from the orderUpdates feed, which omits the order type and time in force
type WebsocketOrder struct {
	Order           BasicOrderResponse `json:"order"`
	Status          string             `json:"status"`
	StatusTimestamp types.Time         `json:"statusTimestamp"`
}

// WebsocketUserEvent contains one event from the userEvents feed, sent on the user channel; exactly one field is set
type WebsocketUserEvent struct {
	Fills          []FillResponse           `json:"fills"`
	Funding        *WebsocketUserFunding    `json:"funding"`
	Liquidation    *WebsocketLiquidation    `json:"liquidation"`
	NonUserCancel  []WebsocketNonUserCancel `json:"nonUserCancel"`
	TWAPSliceFills []TWAPSliceFillResponse  `json:"twapSliceFills"`
}

// WebsocketUserFunding contains one funding payment; USDC is negative when the user paid funding
type WebsocketUserFunding struct {
	Time types.Time `json:"time"`
	// Coin is the perpetual's case-sensitive market name, such as BTC, kPEPE or xyz:XYZ100, rather than a currency code
	Coin        string       `json:"coin"`
	USDC        types.Number `json:"usdc"`
	SignedSize  types.Number `json:"szi"`
	FundingRate types.Number `json:"fundingRate"`
	// NumberOfSamples is the number of hourly payments a daily aggregate holds, and zero for an hourly payment
	NumberOfSamples uint64 `json:"nSamples"`
}

// WebsocketLiquidation contains a liquidation of a user's account
type WebsocketLiquidation struct {
	LiquidationID              uint64       `json:"lid"`
	Liquidator                 string       `json:"liquidator"`
	LiquidatedUser             string       `json:"liquidated_user"`
	LiquidatedNotionalPosition types.Number `json:"liquidated_ntl_pos"`
	LiquidatedAccountValue     types.Number `json:"liquidated_account_value"`
}

// WebsocketNonUserCancel identifies an order the exchange cancelled
type WebsocketNonUserCancel struct {
	// Coin is the case-sensitive market name, such as BTC, kPEPE, xyz:XYZ100 or @107, rather than a currency code
	Coin    string `json:"coin"`
	OrderID uint64 `json:"oid"`
}

// WebsocketUserFills contains fills from the userFills feed; the first message is a snapshot of recent fills
type WebsocketUserFills struct {
	IsSnapshot bool           `json:"isSnapshot"`
	User       string         `json:"user"`
	Fills      []FillResponse `json:"fills"`
}

// WebsocketUserFundings contains funding payments from the userFundings feed; the first message is a snapshot of recent
// payments
type WebsocketUserFundings struct {
	IsSnapshot bool                   `json:"isSnapshot"`
	User       string                 `json:"user"`
	Fundings   []WebsocketUserFunding `json:"fundings"`
}

// WebsocketUserNonFundingLedgerUpdates contains ledger updates other than funding payments; the first message is a
// snapshot of recent updates
type WebsocketUserNonFundingLedgerUpdates struct {
	IsSnapshot              bool                       `json:"isSnapshot"`
	User                    string                     `json:"user"`
	NonFundingLedgerUpdates []UserLedgerUpdateResponse `json:"nonFundingLedgerUpdates"`
}

// WebsocketPerpetualAssetContext contains a perpetual market's context from the activeAssetCtx feed
type WebsocketPerpetualAssetContext struct {
	// Coin is the perpetual's case-sensitive market name, such as BTC, kPEPE or xyz:XYZ100, rather than a currency code
	Coin    string                `json:"coin"`
	Context PerpetualAssetContext `json:"ctx"`
}

// WebsocketSpotAssetContext contains a spot or outcome market's context from the activeAssetCtx feed, sent on the
// activeSpotAssetCtx channel
type WebsocketSpotAssetContext struct {
	// Coin is the spot market's name, such as PURR/USDC or @107, rather than a currency code
	Coin    string           `json:"coin"`
	Context SpotAssetContext `json:"ctx"`
}

// WebsocketUserTWAPSliceFills contains TWAP slice fills from the userTwapSliceFills feed; the first message is a
// snapshot of recent slice fills
type WebsocketUserTWAPSliceFills struct {
	IsSnapshot     bool                    `json:"isSnapshot"`
	User           string                  `json:"user"`
	TWAPSliceFills []TWAPSliceFillResponse `json:"twapSliceFills"`
}

// WebsocketUserTWAPHistory contains TWAP status changes from the userTwapHistory feed; the first message is a snapshot
// of recent changes
type WebsocketUserTWAPHistory struct {
	IsSnapshot bool                       `json:"isSnapshot"`
	User       string                     `json:"user"`
	History    []TWAPHistoryEntryResponse `json:"history"`
}

// TWAPHistoryEntryResponse contains a TWAP order's state when its status changed
type TWAPHistoryEntryResponse struct {
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

// WebsocketBestBidOffer contains a coin's best bid and ask; either is nil when its side of the book is empty
type WebsocketBestBidOffer struct {
	// Coin is the case-sensitive market name, such as BTC, kPEPE, xyz:XYZ100 or @107, rather than a currency code
	Coin string     `json:"coin"`
	Time types.Time `json:"time"`
	// BestBidOffer contains the bid then the ask
	BestBidOffer [2]*L2Level `json:"bbo"`
}

// WebsocketSpotState contains a user's spot balances
type WebsocketSpotState struct {
	User      string                         `json:"user"`
	SpotState SpotClearinghouseStateResponse `json:"spotState"`
}

// WebsocketAllDEXsClearinghouseState contains a user's perpetual margin state on every DEX
type WebsocketAllDEXsClearinghouseState struct {
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

// WebsocketAllDEXsAssetContexts contains every perpetual DEX's asset contexts, each in its DEX's metadata universe
// order
type WebsocketAllDEXsAssetContexts struct {
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

// WebsocketOutcomeMetaUpdate contains one outcome or question change; exactly one field is set
type WebsocketOutcomeMetaUpdate struct {
	OutcomeCreated  *OutcomeSpecification `json:"outcomeCreated"`
	OutcomeSettled  *uint64               `json:"outcomeSettled"`
	QuestionUpdated *OutcomeQuestion      `json:"questionUpdated"`
	QuestionSettled *uint64               `json:"questionSettled"`
}

// WebsocketFastAssetContext contains a coin's changed mark and mid prices from the fastAssetCtxs feed; a field that is
// not set is unchanged, and MidPriceSet with a zero MidPrice means the book has no mid price
type WebsocketFastAssetContext struct {
	MarkPrice    types.Number
	MidPrice     types.Number
	MarkPriceSet bool
	MidPriceSet  bool
}

// UnmarshalJSON records which prices are present, so updates can be merged into the previous snapshot
func (w *WebsocketFastAssetContext) UnmarshalJSON(data []byte) error {
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

// WebsocketPostRequest sends an info request or a signed action over the websocket
type WebsocketPostRequest struct {
	Method string `json:"method"`
	// ID must be unique among the client's outstanding requests, as Hyperliquid does not check it
	ID      uint64                   `json:"id"`
	Request WebsocketPostRequestBody `json:"request"`
}

// WebsocketPostRequestBody contains a post request's type, info or action, and its payload, an info request body or a
// signed action request
type WebsocketPostRequestBody struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// WebsocketPostResponse contains a post request's ID and outcome
type WebsocketPostResponse struct {
	ID       uint64                    `json:"id"`
	Response WebsocketPostResponseBody `json:"response"`
}

// WebsocketPostResponseBody contains a post response's type, info, action or error, and its payload: a
// WebsocketPostInfoPayload for info, an exchange action response for action and an HTTP status for error
type WebsocketPostResponseBody struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// WebsocketPostInfoResponse contains an info request's type and its response, as the info endpoint returns it
type WebsocketPostInfoResponse struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
