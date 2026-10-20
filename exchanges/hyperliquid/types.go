package hyperliquid

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// zonelessTimeLayout is the ISO 8601 layout of Hyperliquid's metadata timestamps, which omit their UTC zone offset
const zonelessTimeLayout = "2006-01-02T15:04:05.999999999"

var (
	errInvalidTupleLength  = errors.New("invalid tuple length")
	errInvalidZonelessTime = errors.New("invalid zone-less timestamp")
)

// Exchange implements exchange.IBotExchange and provides Hyperliquid API access
type Exchange struct {
	exchange.Base
	pairMappingsMu      sync.RWMutex
	pairMappingsFetchMu sync.Mutex
	pairMappings        map[asset.Item][]pairMapping
	pairMappingMisses   map[string]time.Time
	// delistedPairMappings holds delisted perpetual markets, resolved only by coin; pairMappingsMu guards it
	delistedPairMappings []pairMapping
	// pairMappingsRefreshed is when a coin lookup miss last refetched every market; pairMappingsFetchMu guards it
	pairMappingsRefreshed time.Time
	authorityValidationMu sync.Mutex
	// authorityValidations holds every authority validated since setup, so callers alternating subaccounts or credentials
	// do not revalidate before each signed action; authorityValidationMu guards it
	authorityValidations map[authorityValidationKey]struct{}
	websocketPendingMu   sync.Mutex
	websocketPending     map[websocketPendingKey]*websocketPendingOperation
	// websocketTradeReplays holds the coins whose next trades message replays recent trades; websocketPendingMu guards it
	websocketTradeReplays map[string]struct{}
	// websocketWatermarks holds how far the websocket has relayed feeds that repeat entries: the newest trade per coin, the
	// newest fill per account fills feed and each recent order's fills; websocketPendingMu guards it
	websocketWatermarks websocketWatermarks
	// websocketAccountFeeds holds a key for each account feed the account connection subscribes when it connects, noted
	// when the market feeds are generated so that they leave room for them; websocketPendingMu guards it
	websocketAccountFeeds  []*websocketSubscriptionKey
	websocketPostID        atomic.Uint64
	lastNonce              atomic.Uint64
	websocketConnectionsMu sync.Mutex
	// websocketPostConnection is the public connection WebsocketAPI posts use, held apart from the manager so a post
	// never waits on the manager's lock, and websocketAccountConnection the account connection; a connection's disconnect
	// clears only its own entry; websocketConnectionsMu guards both
	websocketPostConnection    websocket.Connection
	websocketAccountConnection websocket.Connection
}

// TenthsOfABasisPoint is a fee rate in tenths of a basis point, the unit of Hyperliquid's builder fees: 1 is 0.001% of
// the notional, and 10 is one basis point
type TenthsOfABasisPoint uint64

// Rate returns the fee rate as a fraction of the notional, such as 0.0001 for one basis point
func (t TenthsOfABasisPoint) Rate() float64 {
	return float64(t) / 100000
}

// Percentage returns the fee rate as a percentage of the notional, such as 0.01 for one basis point
func (t TenthsOfABasisPoint) Percentage() float64 {
	return float64(t) / 1000
}

// percentageText returns the fee rate as an exact decimal percentage, such as 0.001 for one tenth of a basis point
func (t TenthsOfABasisPoint) percentageText() string {
	text := strconv.FormatUint(uint64(t)/1000, 10)
	if fraction := uint64(t) % 1000; fraction != 0 {
		text += strings.TrimRight(fmt.Sprintf(".%03d", fraction), "0")
	}
	return text
}

// ZonelessTime is a UTC timestamp decoded from an ISO 8601 string without a zone offset
type ZonelessTime time.Time

// UnmarshalJSON decodes a zone-less ISO 8601 timestamp as UTC
func (z *ZonelessTime) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("%w: %w", errInvalidZonelessTime, err)
	}
	if text == "" {
		*z = ZonelessTime{}
		return nil
	}
	parsed, err := time.Parse(zonelessTimeLayout, text)
	if err != nil {
		return fmt.Errorf("%w: %w", errInvalidZonelessTime, err)
	}
	*z = ZonelessTime(parsed)
	return nil
}

// Time returns the timestamp as a time.Time
func (z ZonelessTime) Time() time.Time {
	return time.Time(z)
}

// AccountAbstraction identifies how Hyperliquid shares balances across spot and perpetual DEXs
type AccountAbstraction string

// Hyperliquid account abstraction modes returned by the info API
const (
	AccountAbstractionDefault         AccountAbstraction = "default"
	AccountAbstractionDisabled        AccountAbstraction = "disabled"
	AccountAbstractionDEXAbstraction  AccountAbstraction = "dexAbstraction"
	AccountAbstractionUnifiedAccount  AccountAbstraction = "unifiedAccount"
	AccountAbstractionPortfolioMargin AccountAbstraction = "portfolioMargin"
)

// InfoRequest is the body of every info endpoint request; its type field selects the response schema
type InfoRequest struct {
	Type                       string              `json:"type"`
	DEX                        string              `json:"dex,omitempty"`
	Coin                       string              `json:"coin,omitempty"`
	User                       string              `json:"user,omitempty"`
	VaultAddress               string              `json:"vaultAddress,omitempty"`
	OrderID                    any                 `json:"oid,omitempty"`
	StartTime                  uint64              `json:"startTime,omitempty"`
	EndTime                    uint64              `json:"endTime,omitempty"`
	NumberOfSignificantFigures uint64              `json:"nSigFigs,omitempty"`
	Mantissa                   uint64              `json:"mantissa,omitempty"`
	Request                    *CandleSnapshotWire `json:"req,omitempty"`
	AggregateByTime            bool                `json:"aggregateByTime,omitempty"`
	Builder                    string              `json:"builder,omitempty"`
	// Token is a pointer because token 0, USDC, must be sent and Hyperliquid rejects a request that omits it
	Token   *uint64 `json:"token,omitempty"`
	TokenID string  `json:"tokenId,omitempty"`
	Venue   string  `json:"venue,omitempty"`
	// Outcome is a pointer because outcome 0 exists and Hyperliquid rejects a request that omits it
	Outcome *uint64 `json:"outcome,omitempty"`
}

// DEXInfoRequest is the body of info requests that reject an omitted DEX, although an empty name selects the first
// perpetual DEX
type DEXInfoRequest struct {
	Type string `json:"type"`
	DEX  string `json:"dex"`
}

// CandleSnapshotWire is the nested req object of a candle snapshot request
type CandleSnapshotWire struct {
	Coin      string `json:"coin"`
	Interval  string `json:"interval"`
	StartTime uint64 `json:"startTime"`
	// EndTime is omitted when zero, which Hyperliquid answers with the candles up to now; it rejects an endTime of 0
	EndTime uint64 `json:"endTime,omitempty"`
}

// BasicOrder contains the order fields that every order response includes
type BasicOrder struct {
	Coin          string       `json:"coin"`
	Side          string       `json:"side"`
	LimitPrice    types.Number `json:"limitPx"`
	Size          types.Number `json:"sz"`
	OrderID       uint64       `json:"oid"`
	Timestamp     types.Time   `json:"timestamp"`
	OriginalSize  types.Number `json:"origSz"`
	ClientOrderID string       `json:"cloid"`
	// ReduceOnly is sent by openOrders; FrontendOpenOrder's own ReduceOnly takes the frontendOpenOrders value
	ReduceOnly bool `json:"reduceOnly"`
}

// FrontendOpenOrder contains an order with the additional fields the frontend displays
type FrontendOpenOrder struct {
	BasicOrder
	TriggerCondition string              `json:"triggerCondition"`
	IsTrigger        bool                `json:"isTrigger"`
	TriggerPrice     types.Number        `json:"triggerPx"`
	Children         []FrontendOpenOrder `json:"children"`
	IsPositionTPSL   bool                `json:"isPositionTpsl"`
	ReduceOnly       bool                `json:"reduceOnly"`
	OrderType        string              `json:"orderType"`
	TimeInForce      string              `json:"tif"`
}

// Fill contains one fill of an account's order
type Fill struct {
	Coin          string       `json:"coin"`
	Price         types.Number `json:"px"`
	Size          types.Number `json:"sz"`
	Side          string       `json:"side"`
	Time          types.Time   `json:"time"`
	StartPosition types.Number `json:"startPosition"`
	// Direction describes the fill's effect on the position for display, such as Open Long or Close Short
	Direction string       `json:"dir"`
	ClosedPNL types.Number `json:"closedPnl"`
	Hash      string       `json:"hash"`
	OrderID   uint64       `json:"oid"`
	// Crossed is true when the order crossed the spread as the taker
	Crossed bool `json:"crossed"`
	// Fee includes any builder fee, and is negative for a rebate
	Fee           types.Number     `json:"fee"`
	TradeID       uint64           `json:"tid"`
	ClientOrderID string           `json:"cloid"`
	Liquidation   *FillLiquidation `json:"liquidation"`
	FeeToken      currency.Code    `json:"feeToken"`
	BuilderFee    types.Number     `json:"builderFee"`
	TWAPID        uint64           `json:"twapId"`
}

// FillLiquidation contains the liquidation that caused a fill
type FillLiquidation struct {
	LiquidatedUser string       `json:"liquidatedUser"`
	MarkPrice      types.Number `json:"markPx"`
	// Method is market or backstop
	Method string `json:"method"`
}

// UserFillsByTimeRequest contains the parameters of a fills by time request
type UserFillsByTimeRequest struct {
	User      string
	StartTime time.Time
	// EndTime may be zero, which Hyperliquid treats as now
	EndTime time.Time
	// AggregateByTime combines the partial fills of an order matched within one block
	AggregateByTime bool
}

// UserRateLimitResponse contains an account's address-based action rate limit, which allows 10000 requests plus one
// per USDC traded
type UserRateLimitResponse struct {
	CumulativeVolume     types.Number `json:"cumVlm"`
	NumberOfRequestsUsed uint64       `json:"nRequestsUsed"`
	NumberOfRequestsCap  uint64       `json:"nRequestsCap"`
	// NumberOfRequestsSurplus is the reserved request weight not yet used
	NumberOfRequestsSurplus uint64 `json:"nRequestsSurplus"`
}

// OrderStatusRequest identifies an order by its exchange order ID or by its client order ID
type OrderStatusRequest struct {
	User          string
	OrderID       uint64
	ClientOrderID string
}

// OrderStatusResponse contains an order's latest status; Status is unknownOid when the order does not exist
type OrderStatusResponse struct {
	Status string           `json:"status"`
	Order  *HistoricalOrder `json:"order"`
}

// HistoricalOrder contains an order and its latest status
type HistoricalOrder struct {
	Order           FrontendOpenOrder `json:"order"`
	Status          string            `json:"status"`
	StatusTimestamp types.Time        `json:"statusTimestamp"`
}

// L2BookRequest contains the coin and optional price aggregation of an L2 book request
type L2BookRequest struct {
	Coin string
	// NumberOfSignificantFigures aggregates levels to 2, 3, 4 or 5 significant figures; zero requests full precision
	NumberOfSignificantFigures uint64
	// Mantissa further aggregates five significant figure books by 2 or 5; zero leaves them unaggregated. The
	// documentation also lists 1, which Hyperliquid rejects
	Mantissa uint64
}

// L2BookResponse contains an L2 book snapshot, with bids first and asks second
type L2BookResponse struct {
	Coin   string      `json:"coin"`
	Time   types.Time  `json:"time"`
	Levels [][]L2Level `json:"levels"`
	// Spread is only returned for books aggregated by significant figures
	Spread types.Number `json:"spread"`
}

// L2Level contains one aggregated orderbook price level
type L2Level struct {
	Price          types.Number `json:"px"`
	Size           types.Number `json:"sz"`
	NumberOfOrders uint64       `json:"n"`
}

// CandleSnapshotRequest contains the parameters of a candle snapshot request
type CandleSnapshotRequest struct {
	Coin      string
	Interval  kline.Interval
	StartTime time.Time
	EndTime   time.Time
}

// Candle contains one OHLCV interval
type Candle struct {
	OpenTime       types.Time   `json:"t"`
	CloseTime      types.Time   `json:"T"`
	Coin           string       `json:"s"`
	Interval       string       `json:"i"`
	Open           types.Number `json:"o"`
	Close          types.Number `json:"c"`
	High           types.Number `json:"h"`
	Low            types.Number `json:"l"`
	Volume         types.Number `json:"v"`
	NumberOfTrades uint64       `json:"n"`
}

// TWAPSliceFill contains one fill of a TWAP order's slice
type TWAPSliceFill struct {
	Fill   Fill   `json:"fill"`
	TWAPID uint64 `json:"twapId"`
}

// SubAccount contains a subaccount and its first perpetual DEX and spot states
type SubAccount struct {
	Name               string                         `json:"name"`
	SubAccountUser     string                         `json:"subAccountUser"`
	Master             string                         `json:"master"`
	ClearinghouseState ClearinghouseStateResponse     `json:"clearinghouseState"`
	SpotState          SpotClearinghouseStateResponse `json:"spotState"`
}

// VaultDetailsResponse contains a vault's configuration, performance history and followers
type VaultDetailsResponse struct {
	Name                  string            `json:"name"`
	VaultAddress          string            `json:"vaultAddress"`
	Leader                string            `json:"leader"`
	Description           string            `json:"description"`
	Portfolio             []PortfolioPeriod `json:"portfolio"`
	AnnualPercentageRate  float64           `json:"apr"`
	FollowerState         *VaultFollower    `json:"followerState"`
	LeaderFraction        float64           `json:"leaderFraction"`
	LeaderCommission      float64           `json:"leaderCommission"`
	Followers             []VaultFollower   `json:"followers"`
	MaxDistributable      float64           `json:"maxDistributable"`
	MaxWithdrawable       float64           `json:"maxWithdrawable"`
	IsClosed              bool              `json:"isClosed"`
	Relationship          VaultRelationship `json:"relationship"`
	AllowDeposits         bool              `json:"allowDeposits"`
	AlwaysCloseOnWithdraw bool              `json:"alwaysCloseOnWithdraw"`
}

// PortfolioPeriod contains an account's or a vault's history over one period, such as day, week or perpAllTime
type PortfolioPeriod struct {
	Period  string
	History PortfolioHistory
}

// UnmarshalJSON decodes a [period, history] tuple
func (p *PortfolioPeriod) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &p.Period, &p.History)
}

// PortfolioHistory contains the account value and PNL samples of one portfolio period
type PortfolioHistory struct {
	AccountValueHistory []PortfolioValue `json:"accountValueHistory"`
	PNLHistory          []PortfolioValue `json:"pnlHistory"`
	Volume              types.Number     `json:"vlm"`
}

// PortfolioValue contains one timestamped portfolio sample
type PortfolioValue struct {
	Time  types.Time
	Value types.Number
}

// UnmarshalJSON decodes a [timestamp, value] tuple
func (p *PortfolioValue) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &p.Time, &p.Value)
}

// VaultFollower contains one depositor's position in a vault
type VaultFollower struct {
	User           string       `json:"user"`
	VaultEquity    types.Number `json:"vaultEquity"`
	PNL            types.Number `json:"pnl"`
	AllTimePNL     types.Number `json:"allTimePnl"`
	DaysFollowing  uint64       `json:"daysFollowing"`
	VaultEntryTime types.Time   `json:"vaultEntryTime"`
	LockupUntil    types.Time   `json:"lockupUntil"`
}

// VaultRelationship describes whether a vault is a parent of other vaults, a child vault or neither
type VaultRelationship struct {
	Type string                `json:"type"`
	Data VaultRelationshipData `json:"data"`
}

// VaultRelationshipData contains the child vaults of a parent vault
type VaultRelationshipData struct {
	ChildAddresses []string `json:"childAddresses"`
}

// UserVaultEquity contains an account's equity in one vault
type UserVaultEquity struct {
	VaultAddress string       `json:"vaultAddress"`
	Equity       types.Number `json:"equity"`
	// LockedUntilTimestamp is when the deposit's lockup ends, and may be in the past
	LockedUntilTimestamp types.Time `json:"lockedUntilTimestamp"`
}

// UserRoleResponse contains an address's role and the account it is linked to
type UserRoleResponse struct {
	Role string       `json:"role"`
	Data UserRoleData `json:"data"`
}

// UserRoleData contains the account an agent signs for, or the master account of a subaccount
type UserRoleData struct {
	User   string `json:"user"`
	Master string `json:"master"`
}

// ReferralResponse contains an account's referral state; its reward totals are in USDC, with per-token totals in
// TokenToState
type ReferralResponse struct {
	ReferralTokenState
	ReferredBy    *ReferredBy   `json:"referredBy"`
	ReferrerState ReferrerState `json:"referrerState"`
	// RewardHistory contains legacy rewards; claimed rewards are now reported as ledger updates
	RewardHistory []ReferralReward          `json:"rewardHistory"`
	TokenToState  []ReferralTokenStateEntry `json:"tokenToState"`
}

// ReferralTokenState contains an account's volume and referral rewards in one token
type ReferralTokenState struct {
	CumulativeVolume types.Number `json:"cumVlm"`
	UnclaimedRewards types.Number `json:"unclaimedRewards"`
	ClaimedRewards   types.Number `json:"claimedRewards"`
	BuilderRewards   types.Number `json:"builderRewards"`
}

// ReferralTokenStateEntry contains an account's referral totals in one token
type ReferralTokenStateEntry struct {
	Token uint64
	State ReferralTokenState
}

// UnmarshalJSON decodes a [token, state] tuple
func (r *ReferralTokenStateEntry) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &r.Token, &r.State)
}

// ReferredBy contains the referrer and code an account signed up with
type ReferredBy struct {
	Referrer string `json:"referrer"`
	Code     string `json:"code"`
}

// ReferrerState contains an account's progress towards and use of its own referral code
type ReferrerState struct {
	// Stage is ready, needToCreateCode or needToTrade
	Stage string            `json:"stage"`
	Data  ReferrerStateData `json:"data"`
}

// ReferrerStateData contains the code and referred accounts at the ready stage, or the volume still required at the
// needToTrade stage
type ReferrerStateData struct {
	Code              string          `json:"code"`
	NumberOfReferrals uint64          `json:"nReferrals"`
	ReferralStates    []ReferralState `json:"referralStates"`
	Required          types.Number    `json:"required"`
}

// ReferralState contains one referred account's volume and the fees it generated
type ReferralState struct {
	ReferralStateTokenState
	TimeJoined   types.Time                `json:"timeJoined"`
	User         string                    `json:"user"`
	TokenToState []ReferralStateTokenEntry `json:"tokenToState"`
}

// ReferralStateTokenState contains a referred account's volume and the fees it generated in one token
type ReferralStateTokenState struct {
	CumulativeVolume                    types.Number `json:"cumVlm"`
	CumulativeRewardedFeesSinceReferred types.Number `json:"cumRewardedFeesSinceReferred"`
	CumulativeFeesRewardedToReferrer    types.Number `json:"cumFeesRewardedToReferrer"`
}

// ReferralStateTokenEntry contains a referred account's volume and fees in one token
type ReferralStateTokenEntry struct {
	Token uint64
	State ReferralStateTokenState
}

// UnmarshalJSON decodes a [token, state] tuple
func (r *ReferralStateTokenEntry) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &r.Token, &r.State)
}

// ReferralReward contains one legacy referral reward
type ReferralReward struct {
	Earned         types.Number `json:"earned"`
	Volume         types.Number `json:"vlm"`
	ReferralVolume types.Number `json:"referralVlm"`
	Time           types.Time   `json:"time"`
}

// UserFeesResponse contains an account's effective fee rates, the fee schedule and recent volume
type UserFeesResponse struct {
	DailyUserVolume        []DailyUserVolume `json:"dailyUserVlm"`
	FeeSchedule            FeeSchedule       `json:"feeSchedule"`
	UserTakerFeeRate       types.Number      `json:"userCrossRate"`
	UserMakerFeeRate       types.Number      `json:"userAddRate"`
	UserSpotTakerFeeRate   types.Number      `json:"userSpotCrossRate"`
	UserSpotMakerFeeRate   types.Number      `json:"userSpotAddRate"`
	ActiveReferralDiscount types.Number      `json:"activeReferralDiscount"`
	// Trial's shape is undocumented; Hyperliquid has only been seen sending null
	Trial json.RawMessage `json:"trial"`
	// FeeTrialEscrow is the field the documentation calls feeTrialReward, which Hyperliquid does not send
	FeeTrialEscrow              types.Number        `json:"feeTrialEscrow"`
	NextTrialAvailableTimestamp types.Time          `json:"nextTrialAvailableTimestamp"`
	StakingLink                 *StakingLink        `json:"stakingLink"`
	ActiveStakingDiscount       StakingDiscountTier `json:"activeStakingDiscount"`
}

// DailyUserVolume contains an account's taker and maker volume for one day, alongside the exchange's total volume
type DailyUserVolume struct {
	Date            string       `json:"date"`
	UserTakerVolume types.Number `json:"userCross"`
	UserMakerVolume types.Number `json:"userAdd"`
	ExchangeVolume  types.Number `json:"exchange"`
}

// FeeSchedule contains the base fee rates and discount tiers; Hyperliquid calls taker rates cross rates and maker rates
// add rates
type FeeSchedule struct {
	TakerFeeRate         types.Number          `json:"cross"`
	MakerFeeRate         types.Number          `json:"add"`
	SpotTakerFeeRate     types.Number          `json:"spotCross"`
	SpotMakerFeeRate     types.Number          `json:"spotAdd"`
	Tiers                FeeTiers              `json:"tiers"`
	ReferralDiscount     types.Number          `json:"referralDiscount"`
	StakingDiscountTiers []StakingDiscountTier `json:"stakingDiscountTiers"`
}

// FeeTiers contains the volume and market maker fee tiers
type FeeTiers struct {
	VIP         []VIPFeeTier         `json:"vip"`
	MarketMaker []MarketMakerFeeTier `json:"mm"`
}

// VIPFeeTier contains the fee rates that apply above a notional volume cutoff
type VIPFeeTier struct {
	NotionalCutoff   types.Number `json:"ntlCutoff"`
	TakerFeeRate     types.Number `json:"cross"`
	MakerFeeRate     types.Number `json:"add"`
	SpotTakerFeeRate types.Number `json:"spotCross"`
	SpotMakerFeeRate types.Number `json:"spotAdd"`
}

// MarketMakerFeeTier contains the maker rate that applies above a maker volume share cutoff
type MarketMakerFeeTier struct {
	MakerFractionCutoff types.Number `json:"makerFractionCutoff"`
	MakerFeeRate        types.Number `json:"add"`
}

// StakingDiscountTier contains the fee discount for staking a share of the native token's maximum supply
type StakingDiscountTier struct {
	BPSOfMaxSupply types.Number `json:"bpsOfMaxSupply"`
	Discount       types.Number `json:"discount"`
}

// StakingLink contains the staking account linked to a trading account for fee discounts
type StakingLink struct {
	Type        string `json:"type"`
	StakingUser string `json:"stakingUser"`
}

// Delegation contains an account's stake delegated to one validator
type Delegation struct {
	Validator string       `json:"validator"`
	Amount    types.Number `json:"amount"`
	// LockedUntilTimestamp is one day after the account's last delegation to the validator
	LockedUntilTimestamp types.Time `json:"lockedUntilTimestamp"`
}

// DelegatorSummaryResponse contains an account's staking balances in HYPE
type DelegatorSummaryResponse struct {
	Delegated                  types.Number `json:"delegated"`
	Undelegated                types.Number `json:"undelegated"`
	TotalPendingWithdrawal     types.Number `json:"totalPendingWithdrawal"`
	NumberOfPendingWithdrawals uint64       `json:"nPendingWithdrawals"`
}

// DelegatorUpdate contains one staking history entry
type DelegatorUpdate struct {
	Time types.Time `json:"time"`
	// Hash is zero for finalised withdrawals
	Hash  string         `json:"hash"`
	Delta DelegatorDelta `json:"delta"`
}

// DelegatorDelta contains one staking change; exactly one field is set for each change Hyperliquid reports
type DelegatorDelta struct {
	Delegate   *DelegateDelta   `json:"delegate"`
	CDeposit   *CDepositDelta   `json:"cDeposit"`
	Withdrawal *WithdrawalDelta `json:"withdrawal"`
}

// DelegateDelta contains a delegation to or an undelegation from a validator
type DelegateDelta struct {
	Validator    string       `json:"validator"`
	Amount       types.Number `json:"amount"`
	IsUndelegate bool         `json:"isUndelegate"`
}

// CDepositDelta contains a transfer from spot into staking
type CDepositDelta struct {
	Amount types.Number `json:"amount"`
}

// WithdrawalDelta contains a step of a withdrawal from staking to spot
type WithdrawalDelta struct {
	Amount types.Number `json:"amount"`
	// Phase is initiated or finalized
	Phase string `json:"phase"`
}

// DelegatorReward contains one day's staking reward from one source
type DelegatorReward struct {
	Time types.Time `json:"time"`
	// Source is delegation, or commission for validators
	Source      string       `json:"source"`
	TotalAmount types.Number `json:"totalAmount"`
}

// BorrowLendUserStateResponse contains an account's borrow/lend positions and health
type BorrowLendUserStateResponse struct {
	TokenToState []BorrowLendTokenStateEntry `json:"tokenToState"`
	Health       string                      `json:"health"`
	// HealthFactor is zero when Hyperliquid returns null, which it does for accounts without borrows
	HealthFactor types.Number `json:"healthFactor"`
}

// BorrowLendTokenStateEntry contains an account's borrow/lend position in one token
type BorrowLendTokenStateEntry struct {
	Token uint64
	State BorrowLendTokenState
}

// UnmarshalJSON decodes a [token, state] tuple
func (b *BorrowLendTokenStateEntry) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &b.Token, &b.State)
}

// BorrowLendTokenState contains an account's borrowed and supplied amounts of one token
type BorrowLendTokenState struct {
	Borrow BorrowLendBalance `json:"borrow"`
	Supply BorrowLendBalance `json:"supply"`
}

// BorrowLendBalance contains a borrowed or supplied amount and its basis
type BorrowLendBalance struct {
	Basis types.Number `json:"basis"`
	Value types.Number `json:"value"`
}

// BorrowLendReserveStateResponse contains a token's borrow/lend reserve and rates
type BorrowLendReserveStateResponse struct {
	BorrowYearlyRate types.Number `json:"borrowYearlyRate"`
	SupplyYearlyRate types.Number `json:"supplyYearlyRate"`
	Balance          types.Number `json:"balance"`
	Utilisation      types.Number `json:"utilization"`
	OraclePrice      types.Number `json:"oraclePx"`
	LoanToValue      types.Number `json:"ltv"`
	TotalSupplied    types.Number `json:"totalSupplied"`
	TotalBorrowed    types.Number `json:"totalBorrowed"`
}

// BorrowLendReserveStateEntry contains the borrow/lend reserve of one token
type BorrowLendReserveStateEntry struct {
	Token uint64
	State BorrowLendReserveStateResponse
}

// UnmarshalJSON decodes a [token, state] tuple
func (b *BorrowLendReserveStateEntry) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &b.Token, &b.State)
}

// ExchangeStatusResponse contains the exchange's server time and any special statuses it reports
type ExchangeStatusResponse struct {
	// SpecialStatuses is empty when Hyperliquid sends null, the only value it has been seen sending
	SpecialStatuses string     `json:"specialStatuses"`
	Time            types.Time `json:"time"`
}

// RecentTrade contains one public trade
type RecentTrade struct {
	Coin    string       `json:"coin"`
	Side    string       `json:"side"`
	Price   types.Number `json:"px"`
	Size    types.Number `json:"sz"`
	Time    types.Time   `json:"time"`
	Hash    string       `json:"hash"`
	TradeID uint64       `json:"tid"`
	// Users contains the buyer's and then the seller's address
	Users []string `json:"users"`
}

// UserTWAPSliceFillsByTimeRequest contains the parameters of a TWAP slice fills by time request
type UserTWAPSliceFillsByTimeRequest struct {
	User      string
	StartTime time.Time
	// EndTime may be zero, which Hyperliquid treats as now
	EndTime time.Time
}

// isHex128 reports whether a value is a 0x-prefixed 16-byte hexadecimal value, the format of token and client order IDs
func isHex128(value string) bool {
	if len(value) != 34 || !strings.EqualFold(value[:2], "0x") {
		return false
	}
	_, err := hex.DecodeString(value[2:])
	return err == nil
}

// unmarshalTuple decodes a JSON array into exactly as many targets as are supplied
func unmarshalTuple(data []byte, targets ...any) error {
	var elements []json.RawMessage
	if err := json.Unmarshal(data, &elements); err != nil {
		return err
	}
	if len(elements) != len(targets) {
		return fmt.Errorf("%w: expected %d elements, got %d", errInvalidTupleLength, len(targets), len(elements))
	}
	for i := range elements {
		if err := json.Unmarshal(elements[i], targets[i]); err != nil {
			return err
		}
	}
	return nil
}

// GasAuctionResponse contains a Dutch auction priced in HYPE, which deployments and gossip priority use; CurrentGas is zero once
// the auction has been won, and EndGas, the winning price, is zero until then
type GasAuctionResponse struct {
	// StartTime is sent in Unix seconds
	StartTime       types.Time   `json:"startTimeSeconds"`
	DurationSeconds uint64       `json:"durationSeconds"`
	StartGas        types.Number `json:"startGas"`
	CurrentGas      types.Number `json:"currentGas"`
	EndGas          types.Number `json:"endGas"`
}

// GossipPriorityAuctionStatusResponse contains the previous auctions' winners, which form the current gossip priority
// order, and each slot's current auction
type GossipPriorityAuctionStatusResponse struct {
	// PreviousWinners contains each slot's winning IP address, in slot order
	PreviousWinners []string
	Auctions        []GasAuctionResponse
}

// UnmarshalJSON decodes a [previous winners, auctions] tuple
func (g *GossipPriorityAuctionStatusResponse) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &g.PreviousWinners, &g.Auctions)
}
