package okx

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// TradeModeCross trade mode, cross
const TradeModeCross = "cross"

// TradeModeIsolated trade mode, isolated
const TradeModeIsolated = "isolated"

// TradeModeCash trade mode, cash
const TradeModeCash = "cash"

// TradeModeSpotIsolated trade mode, spot_isolated, for SPOT lead trading
const TradeModeSpotIsolated = "spot_isolated"

// Position Side for placing order
const positionSideLong = "long"

const positionSideShort = "short"

const positionSideNet = "net"

// order types, margin balance types, and instrument types constants
const (
	orderLimit                            = "limit"
	orderMarket                           = "market"
	orderPostOnly                         = "post_only"
	orderFOK                              = "fok"
	orderIOC                              = "ioc"
	orderOptimalLimitIOC                  = "optimal_limit_ioc"
	orderConditional                      = "conditional"
	orderMoveOrderStop                    = "move_order_stop"
	orderChase                            = "chase"
	orderTWAP                             = "twap"
	orderTrigger                          = "trigger"
	orderMarketMakerProtectionAndPostOnly = "mmp_and_post_only"
	orderMarketMakerProtection            = "mmp"
	orderOCO                              = "oco"
	orderRPI                              = "rpi"
	orderELP                              = "elp"
	orderOptionFOK                        = "op_fok"

	// represents a margin balance type
	marginBalanceReduce = "reduce"
	marginBalanceAdd    = "add"

	// Instrument Types ( Asset Types )

	instTypeFutures  = "FUTURES"   // instrument type "futures"
	instTypeANY      = "ANY"       // instrument type ""
	instTypeSpot     = "SPOT"      // instrument type "spot"
	instTypeSwap     = "SWAP"      // instrument type "swap"
	instTypeOption   = "OPTION"    // instrument type "option"
	instTypeMargin   = "MARGIN"    // instrument type "margin"
	instTypeContract = "CONTRACTS" // instrument type "contract"
	instTypeEvents   = "EVENTS"    // instrument type "events"

	operationSubscribe   = "subscribe"
	operationUnsubscribe = "unsubscribe"
	operationLogin       = "login"

	// orderListPageSize is OKX's maximum, and default, records per pending
	// order list request, used to page through accounts holding more open
	// orders than a single page.
	orderListPageSize = 100
)

var errFeeRateUnavailable = errors.New("applicable trading fee rate unavailable")

var errMissingInstrumentID = errors.New("missing instrument ID")

var errMissingInstrumentIDCode = errors.New("missing instrument ID code")

var errEitherInstIDOrCcyIsRequired = errors.New("either parameter instId or ccy is required")

var errInvalidTradeModeValue = errors.New("invalid trade mode value")

var errCurrencyQuantityTypeRequired = errors.New("only base_ccy and quote_ccy quantity types are supported")

var errInvalidPriceLimit = errors.New("invalid price limit value")

var errMissingIntervalValue = errors.New("missing interval value")

var errMissingSizeLimit = errors.New("missing required parameter 'szLimit'")

var errAlgoIDRequired = errors.New("algo ID is required")

var errInstrumentFamilyRequired = errors.New("instrument family is required")

var errCountdownTimeoutRequired = errors.New("countdown timeout is required")

var errInvalidInstrumentType = errors.New("invalid instrument type")

var errInvalidLeverage = errors.New("invalid leverage value")

var errInvalidAPIKey = errors.New("invalid api key")

var errInvalidAlgoOrderType = errors.New("invalid algo order type")

var errExceedLimit = errors.New("limit exceeded")

var errOnlyThreeMonthsSupported = errors.New("only three months of trade data retrieval supported")

var errOnlyOneResponseExpected = errors.New("one response item expected")

var errAmbiguousPosition = errors.New("more than one position matches the request")

var errPositionModeSwitchedDuringFetch = errors.New("the account position mode switched while it was being fetched")

var errAccountConfigurationSwitchedDuringFetch = errors.New("the account configuration switched while it was being fetched")

var errInvalidAccountLevel = errors.New("invalid account level")

// OKX's account levels (acctLv): Spot mode, Futures mode, Multi-currency margin and Portfolio margin
const (
	accountLevelSpot = iota + 1
	accountLevelFutures
	accountLevelMultiCurrencyMargin
	accountLevelPortfolioMargin
)

var errPriceTrackingNotSet = errors.New("price tracking value not set")

var errFeeTypeUnsupported = errors.New("fee type is not supported")

// testNetKey this key is designed for using the testnet endpoints
// setting context.WithValue(ctx, testNetKey("testnet"), useTestNet)
// will ensure the appropriate headers are sent to OKx to use the testnet
type testNetKey string

var testNetVal = testNetKey("testnet")

// InstrumentsRequest holds the parameters for GetInstruments
type InstrumentsRequest struct {
	InstrumentType   string
	SeriesID         string // required for EVENTS
	InstrumentFamily string
	InstrumentID     string
	// Underlying filters by uly, which OKX no longer documents but still honours; an underlying such as BTC-USD spans
	// several instrument families
	Underlying string
}

// InstrumentResponse represents an instrument with open contracts
type InstrumentResponse struct {
	InstrumentType                   string                     `json:"instType"`
	SeriesID                         string                     `json:"seriesId"`
	InstrumentID                     currency.Pair              `json:"instId"`
	Underlying                       string                     `json:"uly"`
	TradingFeeGroupID                string                     `json:"groupId"`
	InstrumentFamily                 string                     `json:"instFamily"`
	Category                         string                     `json:"category"`
	BaseCurrency                     currency.Code              `json:"baseCcy"`
	QuoteCurrency                    currency.Code              `json:"quoteCcy"`
	SettlementCurrency               currency.Code              `json:"settleCcy"`
	ContractValue                    types.Number               `json:"ctVal"`
	ContractMultiplier               types.Number               `json:"ctMult"`
	ContractValueCurrency            currency.Code              `json:"ctValCcy"`
	OptionType                       string                     `json:"optType"`
	StrikePrice                      types.Number               `json:"stk"`
	ListTime                         types.Time                 `json:"listTime"`
	AuctionEndTime                   types.Time                 `json:"auctionEndTime"`
	ContinuousTradingSwitchTime      types.Time                 `json:"contTdSwTime"`
	PreMarketSwitchTime              types.Time                 `json:"preMktSwTime"`
	OpenType                         string                     `json:"openType"`
	ExpiryTime                       types.Time                 `json:"expTime"`
	MaxLeverage                      types.Number               `json:"lever"`
	TickSize                         types.Number               `json:"tickSz"`
	LotSize                          types.Number               `json:"lotSz"`
	MinimumOrderSize                 types.Number               `json:"minSz"`
	ContractType                     string                     `json:"ctType"`
	Alias                            string                     `json:"alias"`
	State                            string                     `json:"state"`
	RuleType                         string                     `json:"ruleType"`
	MaxLimitSize                     types.Number               `json:"maxLmtSz"`
	MaxMarketSize                    types.Number               `json:"maxMktSz"`
	MaxLimitAmountUSD                types.Number               `json:"maxLmtAmt"`
	MaxMarketAmountUSD               types.Number               `json:"maxMktAmt"`
	MaxTWAPSize                      types.Number               `json:"maxTwapSz"`
	MaxIcebergSize                   types.Number               `json:"maxIcebergSz"`
	MaxTriggerSize                   types.Number               `json:"maxTriggerSz"`
	MaxStopSize                      types.Number               `json:"maxStopSz"`
	FutureSettlement                 bool                       `json:"futureSettlement"`
	TradableQuoteCurrencies          []currency.Code            `json:"tradeQuoteCcyList"`
	InstrumentIDCode                 uint64                     `json:"instIdCode"`
	InstrumentCategory               string                     `json:"instCategory"`
	InitialPriceLimitPercent         types.Number               `json:"initPxLmtPct"`
	FloatingPriceLimitPercent        types.Number               `json:"floatPxLmtPct"`
	MaximumPriceLimitPercent         types.Number               `json:"maxPxLmtPct"`
	RPIMinimumLevel                  types.Number               `json:"rpiMinLevel"`
	RPIMinimumPriceBand              types.Number               `json:"rpiMinPxBand"`
	PositionLimitAmount              types.Number               `json:"posLmtAmt"`
	PositionLimitPercent             types.Number               `json:"posLmtPct"`
	MaxPlatformOpenInterestLimit     types.Number               `json:"maxPlatOILmt"`
	MaxPlatformOpenInterestCoinLimit types.Number               `json:"maxPlatOICoinLmt"`
	LongPositionRemainingQuota       types.Number               `json:"longPosRemainingQuota"`
	ShortPositionRemainingQuota      types.Number               `json:"shortPosRemainingQuota"`
	Frequency                        string                     `json:"freq"`
	SettlementMethod                 string                     `json:"method"`
	UpcomingChanges                  []InstrumentUpcomingChange `json:"upcChg"`
}

// InstrumentUpcomingChange represents a documented parameter change scheduled
// for an instrument.
type InstrumentUpcomingChange struct {
	Parameter     string     `json:"param"`
	NewValue      string     `json:"newValue"`
	EffectiveTime types.Time `json:"effTime"`
}

// OpenInterestResponse holds an instrument's total open interest
type OpenInterestResponse struct {
	InstrumentType       string       `json:"instType"`
	InstrumentID         string       `json:"instId"`
	OpenInterest         types.Number `json:"oi"` // in contracts
	OpenInterestCurrency types.Number `json:"oiCcy"`
	OpenInterestUSD      types.Number `json:"oiUsd"`
	Timestamp            types.Time   `json:"ts"`
}

// FundingRateResponse holds a perpetual swap's or X-Perps futures contract's current or historical funding rate.
// A nil realised rate indicates that settlement evidence was not supplied.
type FundingRateResponse struct {
	InstrumentType        string        `json:"instType"`
	InstrumentID          string        `json:"instId"`
	FundingRateMethod     string        `json:"method"`
	FormulaType           string        `json:"formulaType"`
	FundingRate           types.Number  `json:"fundingRate"`
	RealisedRate          *types.Number `json:"realizedRate"`
	NextFundingRate       types.Number  `json:"nextFundingRate"`
	FundingTime           types.Time    `json:"fundingTime"`
	NextFundingTime       types.Time    `json:"nextFundingTime"`
	PreviousFundingTime   types.Time    `json:"prevFundingTime"`
	MinFundingRate        types.Number  `json:"minFundingRate"`
	MaxFundingRate        types.Number  `json:"maxFundingRate"`
	InterestRate          types.Number  `json:"interestRate"`
	ImpactValue           types.Number  `json:"impactValue"`
	SettlementState       string        `json:"settState"`
	SettlementFundingRate types.Number  `json:"settFundingRate"`
	Premium               types.Number  `json:"premium"`
	Timestamp             types.Time    `json:"ts"`
}

// UnmarshalJSON preserves whether OKX supplied a realised settlement rate.
func (r *FundingRateResponse) UnmarshalJSON(data []byte) error {
	type rate FundingRateResponse
	type response struct {
		*rate
		RealisedRate OptionalNumber `json:"realizedRate"`
	}
	decoded := response{rate: (*rate)(r)}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	decoded.rate.RealisedRate = decoded.RealisedRate.Value
	return nil
}

// MarkPriceResponse holds an instrument's mark price
type MarkPriceResponse struct {
	InstrumentType string       `json:"instType"`
	InstrumentID   string       `json:"instId"`
	MarkPrice      types.Number `json:"markPx"`
	Timestamp      types.Time   `json:"ts"`
}

// PlaceOrderRequest requesting parameter for placing an order
type PlaceOrderRequest struct {
	AssetType               asset.Item               `json:"-"`
	InstrumentID            string                   `json:"instId,omitempty"`     // Required for REST order placement
	InstrumentIDCode        uint64                   `json:"instIdCode,omitempty"` // Instrument ID code, required for WS order placement
	TradeMode               string                   `json:"tdMode"`
	Currency                currency.Code            `json:"ccy,omitzero"`
	ClientOrderID           string                   `json:"clOrdId,omitempty"`
	OrderTag                string                   `json:"tag,omitempty"`
	Side                    string                   `json:"side"`
	PositionSide            string                   `json:"posSide,omitempty"`
	OrderType               string                   `json:"ordType"`
	Size                    float64                  `json:"sz,string"`
	Price                   float64                  `json:"px,string,omitempty"`
	Outcome                 string                   `json:"outcome,omitempty"`
	PriceUSD                float64                  `json:"pxUsd,omitempty,string"`
	PriceVolatility         float64                  `json:"pxVol,omitempty,string"`
	ReduceOnly              bool                     `json:"reduceOnly,omitempty"`
	TargetCurrency          string                   `json:"tgtCcy,omitempty"` // base_ccy or quote_ccy
	BanAmend                bool                     `json:"banAmend,omitempty"`
	PriceAmendType          string                   `json:"pxAmendType,omitempty"`
	TradeQuoteCurrency      currency.Code            `json:"tradeQuoteCcy,omitzero"`
	SlippagePercentage      float64                  `json:"slippagePct,omitempty,string"`
	SelfTradePreventionMode string                   `json:"stpMode,omitempty"`
	RPITakerAccess          bool                     `json:"rpiTakerAccess,omitempty"`
	RPIPriceRound           bool                     `json:"rpiPxRound,omitempty"`
	AttachAlgoOrders        []AttachAlgoOrderRequest `json:"attachAlgoOrds,omitempty"`
}

// Validate checks the PlaceOrderRequest for REST placement, which identifies the instrument by its instrument ID
func (arg *PlaceOrderRequest) Validate() error {
	if arg == nil {
		return fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.InstrumentID == "" {
		return errMissingInstrumentID
	}
	return arg.validateOrder()
}

// validateOrder checks the fields REST and websocket placement share, apart from the instrument, which each transport
// identifies its own way
func (arg *PlaceOrderRequest) validateOrder() error {
	if arg.AssetType == asset.Empty || arg.AssetType == asset.Spot || arg.AssetType == asset.Margin ||
		arg.AssetType == asset.Futures || arg.AssetType == asset.PerpetualSwap || arg.AssetType == asset.Options {
		arg.Side = strings.ToLower(arg.Side)
		if arg.Side != order.Buy.Lower() && arg.Side != order.Sell.Lower() {
			return fmt.Errorf("%w %s", order.ErrSideIsInvalid, arg.Side)
		}
	}
	if arg.TradeMode == "" {
		return fmt.Errorf("%w: the trade mode is required", errInvalidTradeModeValue)
	}
	if arg.AssetType == asset.Futures || arg.AssetType == asset.PerpetualSwap {
		arg.PositionSide = strings.ToLower(arg.PositionSide)
		// The account's position mode decides the required value, which only
		// the caller knows: net mode defaults to net and may omit the field
		// entirely, so empty is accepted too.
		if !slices.Contains([]string{"", positionSideNet, positionSideLong, positionSideShort}, arg.PositionSide) {
			return fmt.Errorf("%w: %q", order.ErrSideIsInvalid, arg.PositionSide)
		}
	}
	arg.OrderType = strings.ToLower(arg.OrderType)
	if arg.OrderType == "" {
		return fmt.Errorf("%w: the order type is required", order.ErrTypeIsInvalid)
	}
	if arg.Size <= 0 {
		return limits.ErrAmountBelowMin
	}
	if !slices.Contains([]string{"", "base_ccy", "quote_ccy"}, arg.TargetCurrency) {
		return errCurrencyQuantityTypeRequired
	}
	return nil
}

// OrderActionResponse response message for place, cancel, and amend an order requests.
type OrderActionResponse struct {
	OrderID       string     `json:"ordId"`
	RequestID     string     `json:"reqId"`
	ClientOrderID string     `json:"clOrdId"`
	Tag           string     `json:"tag"`
	StatusCode    uint64     `json:"sCode,string"` // Anything above 0 is an error with an attached message
	StatusMessage string     `json:"sMsg"`
	SubCode       string     `json:"subCode"`
	Timestamp     types.Time `json:"ts"`

	statusCodeMissing bool
}

// UnmarshalJSON decodes an order action's reply row, noting a row without the sCode stating its execution result,
// whose status code would otherwise read as success
func (o *OrderActionResponse) UnmarshalJSON(data []byte) error {
	type row OrderActionResponse
	decoded := struct {
		*row
		StatusCode *string `json:"sCode"`
	}{row: (*row)(o)}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var err error
	o.StatusCode, o.statusCodeMissing, err = decodeStatusCode(decoded.StatusCode)
	return err
}

// Error returns the order action's status error, nil when it succeeded
func (o *OrderActionResponse) Error() error {
	return replyRowStatusError(o.StatusCode, o.StatusMessage, o.statusCodeMissing)
}

// errMissingStatusCode reports an order operation's reply row without the sCode stating its execution result
var errMissingStatusCode = fmt.Errorf("%w: reply row has no status code", common.ErrInvalidResponse)

// decodeStatusCode returns the execution result code a reply row's sCode states, which OKX sends as a quoted integer,
// and whether the row carried none
func decodeStatusCode(sCode *string) (code uint64, missing bool, err error) {
	if sCode == nil {
		return 0, true, nil
	}
	code, err = strconv.ParseUint(*sCode, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("status code %q: %w", *sCode, err)
	}
	return code, false, nil
}

// replyRowStatusError returns the error an order operation's reply row states, nil for success
func replyRowStatusError(statusCode uint64, statusMessage string, statusCodeMissing bool) error {
	if statusCodeMissing {
		return errMissingStatusCode
	}
	return getStatusError(statusCode, statusMessage)
}

// RESTResponse contains the common REST envelope; endpoints decode Data into their own response types.
type RESTResponse struct {
	Code    types.Number    `json:"code"`
	Message string          `json:"msg"`
	Data    json.RawMessage `json:"data"`
	// TotalPage is sent beside data by the affiliate list endpoints
	TotalPage types.Number `json:"totalPage"`
}

// ResultResponse holds responses having a status result value
type ResultResponse struct {
	Result        bool   `json:"result"`
	StatusCode    uint64 `json:"sCode,string"`
	StatusMessage string `json:"sMsg"`
}

func (r *ResultResponse) Error() error {
	return getStatusError(r.StatusCode, r.StatusMessage)
}

// PlaceAlgoOrderRequest holds the parameters to place a trigger, OCO, chase,
// conditional, TWAP, trailing stop or smart iceberg algo order
type PlaceAlgoOrderRequest struct {
	InstrumentID       string        `json:"instId"`
	TradeMode          string        `json:"tdMode"`
	Currency           currency.Code `json:"ccy,omitzero"`
	Side               string        `json:"side"`
	PositionSide       string        `json:"posSide,omitempty"`
	OrderType          string        `json:"ordType"`
	Size               float64       `json:"sz,omitempty,string"`
	OrderTag           string        `json:"tag,omitempty"`
	TargetCurrency     string        `json:"tgtCcy,omitempty"` // base_ccy or quote_ccy
	AlgoClientOrderID  string        `json:"algoClOrdId,omitempty"`
	CloseFraction      float64       `json:"closeFraction,omitempty,string"`
	TradeQuoteCurrency currency.Code `json:"tradeQuoteCcy,omitzero"`
	ReduceOnly         bool          `json:"reduceOnly,omitempty"`

	// Take profit and stop loss orders
	TakeProfitTriggerPrice     float64 `json:"tpTriggerPx,omitempty,string"`
	TakeProfitTriggerPriceType string  `json:"tpTriggerPxType,omitempty"`
	TakeProfitOrderPrice       float64 `json:"tpOrdPx,omitempty,string"`
	TakeProfitOrderKind        string  `json:"tpOrdKind,omitempty"`
	StopLossTriggerPrice       float64 `json:"slTriggerPx,omitempty,string"`
	StopLossTriggerPriceType   string  `json:"slTriggerPxType,omitempty"`
	StopLossOrderPrice         float64 `json:"slOrdPx,omitempty,string"`
	CancelOnClosePosition      bool    `json:"cxlOnClosePos,omitempty"`

	// Chase orders
	ChaseType     string  `json:"chaseType,omitempty"`
	ChaseValue    float64 `json:"chaseVal,omitempty,string"`
	MaxChaseType  string  `json:"maxChaseType,omitempty"`
	MaxChaseValue float64 `json:"maxChaseVal,omitempty,string"`

	// Trigger orders; an order price of -1 places a market order when triggered
	TriggerPrice            float64                   `json:"triggerPx,omitempty,string"`
	OrderPrice              float64                   `json:"orderPx,omitempty,string"`
	AdvanceOrderType        string                    `json:"advanceOrdType,omitempty"`
	AdvancedChaseParameters []AdvancedChaseParameters `json:"advChaseParams,omitempty"`
	TriggerPriceType        string                    `json:"triggerPxType,omitempty"`
	AttachAlgoOrders        []AttachAlgoOrderRequest  `json:"attachAlgoOrds,omitempty"`

	// Trailing stop orders
	CallbackRatio  float64 `json:"callbackRatio,omitempty,string"`
	CallbackSpread float64 `json:"callbackSpread,omitempty,string"`
	ActivePrice    float64 `json:"activePx,omitempty,string"`

	// TWAP and smart iceberg orders
	PriceVariance    float64 `json:"pxVar,omitempty,string"`
	PriceSpread      float64 `json:"pxSpread,omitempty,string"`
	SizeLimit        float64 `json:"szLimit,omitempty,string"`
	PriceLimit       float64 `json:"pxLimit,omitempty,string"`
	TimeInterval     uint64  `json:"timeInterval,omitempty,string"` // In seconds
	LimitOrderNumber uint64  `json:"lmtOrderNumber,omitempty,string"`
	Aggressiveness   string  `json:"aggressiveness,omitempty"`

	TriggerParameters []IcebergTriggerRequest `json:"triggerParams,omitempty"`
}

// AdvancedChaseParameters holds the chase limit order a trigger order places
// once triggered
type AdvancedChaseParameters struct {
	ChaseType     string       `json:"chaseType,omitempty"`
	ChaseValue    types.Number `json:"chaseVal,omitempty"`
	MaxChaseType  string       `json:"maxChaseType,omitempty"`
	MaxChaseValue types.Number `json:"maxChaseVal,omitempty"`
}

// IcebergTriggerRequest holds a condition that starts a smart iceberg order
type IcebergTriggerRequest struct {
	TriggerAction    string  `json:"triggerAction"`
	TriggerStrategy  string  `json:"triggerStrategy"`
	TriggerPrice     float64 `json:"triggerPx,omitempty,string"`
	TriggerCondition string  `json:"triggerCond,omitempty"`
	Timeframe        string  `json:"timeframe,omitempty"`
	Threshold        float64 `json:"thold,omitempty,string"`
	TimePeriod       uint64  `json:"timePeriod,omitempty,string"`
}

// AlgoOrderActionResponse holds the result of placing or cancelling an algo
// order
type AlgoOrderActionResponse struct {
	AlgoID            string `json:"algoId"`
	ClientOrderID     string `json:"clOrdId"`
	AlgoClientOrderID string `json:"algoClOrdId"`
	StatusCode        uint64 `json:"sCode,string"`
	StatusMessage     string `json:"sMsg"`
	Tag               string `json:"tag"`

	statusCodeMissing bool
}

// UnmarshalJSON decodes an algo order action's reply row, noting a row without the sCode stating its execution
// result, whose status code would otherwise read as success
func (a *AlgoOrderActionResponse) UnmarshalJSON(data []byte) error {
	type row AlgoOrderActionResponse
	decoded := struct {
		*row
		StatusCode *string `json:"sCode"`
	}{row: (*row)(a)}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var err error
	a.StatusCode, a.statusCodeMissing, err = decodeStatusCode(decoded.StatusCode)
	return err
}

// Error returns the algo order action's status error, nil when it succeeded
func (a *AlgoOrderActionResponse) Error() error {
	return replyRowStatusError(a.StatusCode, a.StatusMessage, a.statusCodeMissing)
}

// AlgoTriggerRequest represents a trigger condition of an advanced algo order:
// a smart-iceberg trigger, or a grid algo trigger with the grid-specific
// fields.
type AlgoTriggerRequest struct {
	TriggerAction    string       `json:"triggerAction"`
	TriggerStrategy  string       `json:"triggerStrategy"`
	DelaySeconds     types.Number `json:"delaySeconds"`
	TriggerTime      types.Time   `json:"triggerTime"`
	TriggerType      string       `json:"triggerType"`
	TriggerPrice     types.Number `json:"triggerPx"`
	TriggerCondition string       `json:"triggerCond"`
	Timeframe        string       `json:"timeframe"`
	Threshold        types.Number `json:"thold"`
	TimePeriod       string       `json:"timePeriod"`
	StopType         string       `json:"stopType"`
}

// AttachedAlgoOrder represents the attached TP/SL or trailing stop order info
// returned beside an order or algo order row.
type AttachedAlgoOrder struct {
	AttachAlgoID               string       `json:"attachAlgoId"`
	AttachAlgoClientOrderID    string       `json:"attachAlgoClOrdId"`
	TakeProfitTriggerPrice     types.Number `json:"tpTriggerPx"`
	TakeProfitTriggerRatio     types.Number `json:"tpTriggerRatio"`
	TakeProfitTriggerPriceType string       `json:"tpTriggerPxType"`
	TakeProfitOrderPrice       types.Number `json:"tpOrdPx"`
	TakeProfitOrderKind        string       `json:"tpOrdKind"`
	StopLossTriggerPrice       types.Number `json:"slTriggerPx"`
	StopLossTriggerRatio       types.Number `json:"slTriggerRatio"`
	StopLossTriggerPriceType   string       `json:"slTriggerPxType"`
	StopLossOrderPrice         types.Number `json:"slOrdPx"`
	Size                       types.Number `json:"sz"`
	AmendPriceOnTriggerType    string       `json:"amendPxOnTriggerType"`
	CallbackRatio              types.Number `json:"callbackRatio"`
	CallbackSpread             types.Number `json:"callbackSpread"`
	ActivePrice                types.Number `json:"activePx"`
	FailCode                   string       `json:"failCode"`
	FailReason                 string       `json:"failReason"`
}

// EstimatedCompleteTime represents the estimated completion timestamp of the
// deposit-withdraw-status endpoint, documented as a wall-clock
// MM/dd/yyyy, h:mm:ss AM/PM string in UTC+8.
type EstimatedCompleteTime time.Time

// estCompleteTimeLayout is the documented timestamp layout of
// estCompleteTime. The hour slot is the non-zero-padded 12-hour form.
const estCompleteTimeLayout = "01/02/2006, 3:04:05 PM"

// estCompleteTimeLocation holds the UTC+8 location the endpoint documents.
var estCompleteTimeLocation = time.FixedZone("UTC+8", 8*60*60)

// UnmarshalJSON deserialises the documented wall-clock format, tolerating the
// null and empty values the endpoint returns when no estimate is available.
func (t *EstimatedCompleteTime) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("%w error decoding %s into estCompleteTime: %w", types.ErrInvalidTimestampFormat, data, err)
	}
	if value == "" || value == "null" {
		return nil
	}
	parsed, err := time.ParseInLocation(estCompleteTimeLayout, value, estCompleteTimeLocation)
	if err != nil {
		return fmt.Errorf("%w error parsing %q into estCompleteTime: %w", types.ErrInvalidTimestampFormat, value, err)
	}
	*t = EstimatedCompleteTime(parsed)
	return nil
}

// Time returns the time.Time instance.
func (t EstimatedCompleteTime) Time() time.Time { return time.Time(t) }

// String returns a string representation of the time.
func (t EstimatedCompleteTime) String() string { return t.Time().String() }

// MarshalJSON serialises the time to json using RFC 3339 format.
func (t EstimatedCompleteTime) MarshalJSON() ([]byte, error) { return t.Time().MarshalJSON() }

// AccountBalanceResponse holds the trading account balance and its per currency details
type AccountBalanceResponse struct {
	UpdateTime                   types.Time             `json:"uTime"`
	TotalEquity                  types.Number           `json:"totalEq"`
	IsolatedMarginEquity         types.Number           `json:"isoEq"`
	AdjustedEquity               types.Number           `json:"adjEq"`
	AvailableEquity              types.Number           `json:"availEq"`
	OrderFrozen                  types.Number           `json:"ordFroz"`
	InitialMarginRequirement     types.Number           `json:"imr"`
	MaintenanceMarginRequirement types.Number           `json:"mmr"`
	BorrowFrozen                 types.Number           `json:"borrowFroz"`
	MarginRatio                  types.Number           `json:"mgnRatio"`
	NotionalUSD                  types.Number           `json:"notionalUsd"`
	NotionalUSDForBorrow         types.Number           `json:"notionalUsdForBorrow"`
	NotionalUSDForSwap           types.Number           `json:"notionalUsdForSwap"`
	NotionalUSDForFutures        types.Number           `json:"notionalUsdForFutures"`
	NotionalUSDForOption         types.Number           `json:"notionalUsdForOption"`
	UnrealisedProfitAndLoss      types.Number           `json:"upl"`
	Delta                        types.Number           `json:"delta"`
	DeltaLeverage                types.Number           `json:"deltaLever"`
	DeltaNeutralStatus           string                 `json:"deltaNeutralStatus"`
	Details                      []AccountBalanceDetail `json:"details"`
}

// AccountBalanceDetail holds the trading account balance of a currency
type AccountBalanceDetail struct {
	Currency                           currency.Code `json:"ccy"`
	EquityOfCurrency                   types.Number  `json:"eq"`
	CashBalance                        types.Number  `json:"cashBal"`
	UpdateTime                         types.Time    `json:"uTime"`
	IsolatedEquity                     types.Number  `json:"isoEq"`
	AvailableEquity                    types.Number  `json:"availEq"`
	DiscountEquity                     types.Number  `json:"disEq"`
	FixedBalance                       types.Number  `json:"fixedBal"`
	AvailableBalance                   types.Number  `json:"availBal"`
	FrozenBalance                      types.Number  `json:"frozenBal"`
	OrderFrozen                        types.Number  `json:"ordFrozen"`
	Liabilities                        types.Number  `json:"liab"`
	UnrealisedProfitAndLoss            types.Number  `json:"upl"`
	UnrealisedProfitAndLossLiabilities types.Number  `json:"uplLiab"`
	CrossLiabilities                   types.Number  `json:"crossLiab"`
	RewardBalance                      types.Number  `json:"rewardBal"`
	IsolatedLiabilities                types.Number  `json:"isoLiab"`
	MarginRatio                        types.Number  `json:"mgnRatio"`
	InitialMarginRequirement           types.Number  `json:"imr"`
	MaintenanceMarginRequirement       types.Number  `json:"mmr"`
	Interest                           types.Number  `json:"interest"`
	ForcedRepaymentRiskLevel           types.Number  `json:"twap"`
	ForcedRepaymentType                string        `json:"frpType"`
	MaxLoan                            types.Number  `json:"maxLoan"`
	EquityUSD                          types.Number  `json:"eqUsd"`
	BorrowFrozen                       types.Number  `json:"borrowFroz"`
	NotionalLeverage                   types.Number  `json:"notionalLever"`
	CoinUSDPrice                       types.Number  `json:"coinUsdPrice"` // Sent by the account websocket channel only
	StrategyEquity                     types.Number  `json:"stgyEq"`
	IsolatedUnrealisedProfitAndLoss    types.Number  `json:"isoUpl"`
	SpotInUseAmount                    types.Number  `json:"spotInUseAmt"`
	ClientSpotInUseAmount              types.Number  `json:"clSpotInUseAmt"`
	MaxSpotInUse                       types.Number  `json:"maxSpotInUse"`
	// MaxSpotInUseAmount is filled by the account channel push, which sends
	// maxSpotInUseAmt; the REST balances endpoint sends maxSpotInUse instead
	// (MaxSpotInUse above).
	MaxSpotInUseAmount               types.Number `json:"maxSpotInUseAmt"`
	SpotIsolatedBalance              types.Number `json:"spotIsoBal"`
	SmartSyncEquity                  types.Number `json:"smtSyncEq"`
	SpotCopyTradingEquity            types.Number `json:"spotCopyTradingEq"`
	SpotBalance                      types.Number `json:"spotBal"`
	OpenAveragePrice                 types.Number `json:"openAvgPx"`
	AccumulatedAveragePrice          types.Number `json:"accAvgPx"`
	SpotUnrealisedProfitAndLoss      types.Number `json:"spotUpl"`
	SpotUnrealisedProfitAndLossRatio types.Number `json:"spotUplRatio"`
	TotalProfitAndLoss               types.Number `json:"totalPnl"`
	TotalProfitAndLossRatio          types.Number `json:"totalPnlRatio"`
	CollateralRestriction            string       `json:"colRes"`
	CollateralBorrowAutoConversion   string       `json:"colBorrAutoConversion"`
	CollateralBorrowRestricted       bool         `json:"collateralRestrict"` // Deprecated, see CollateralRestriction
	CollateralEnabled                bool         `json:"collateralEnabled"`
	AutoLendStatus                   string       `json:"autoLendStatus"`
	AutoLendMatchedAmount            types.Number `json:"autoLendMtAmt"`
}

// billTypeFundingFee filters account bills down to funding fee entries,
// bill type 8 per the BillsDetailQueryParameter.BillType mapping
const billTypeFundingFee = 8

// TradeFeeRateResponse holds trade fee rate information for a given instrument type.
// Nil fee rates represent omitted, null or empty API values.
type TradeFeeRateResponse struct {
	FeeRateLevel      string          `json:"level"`
	FeeGroup          []TradeFeeGroup `json:"feeGroup"`
	DeliveryFeeRate   types.Number    `json:"delivery"`
	ExerciseFeeRate   types.Number    `json:"exercise"`
	InstrumentType    string          `json:"instType"`
	Timestamp         types.Time      `json:"ts"`
	FeeRateTaker      *types.Number   `json:"taker"`     // Deprecated
	FeeRateMaker      *types.Number   `json:"maker"`     // Deprecated
	FeeRateTakerUSDT  *types.Number   `json:"takerU"`    // Deprecated
	FeeRateMakerUSDT  *types.Number   `json:"makerU"`    // Deprecated
	FeeRateTakerUSDC  *types.Number   `json:"takerUSDC"` // Deprecated
	FeeRateMakerUSDC  *types.Number   `json:"makerUSDC"` // Deprecated
	RuleType          string          `json:"ruleType"`  // Deprecated
	Category          string          `json:"category"`  // Deprecated
	Fiat              []FiatItemInfo  `json:"fiat"`      // Deprecated
	SettlementFeeRate types.Number    `json:"settle"`    // Only applicable to EVENTS
}

// TradeFeeGroup represents a trading fee group's maker and taker rates.
// Nil fee rates represent omitted, null or empty API values.
type TradeFeeGroup struct {
	GroupID      string        `json:"groupId"`
	MakerRate    *types.Number `json:"maker"`
	TakerRate    *types.Number `json:"taker"`
	RPIMakerRate types.Number  `json:"rpiMaker"`
}

// FiatItemInfo represents fiat currency with taker and maker fee details
type FiatItemInfo struct {
	Currency currency.Code `json:"ccy"`
	Taker    types.Number  `json:"taker"`
	Maker    types.Number  `json:"maker"`
}

// BorrowAndRepay manual holds manual borrow and repay in quick margin mode
type BorrowAndRepay struct {
	Amount       float64       `json:"amt,string"`
	InstrumentID string        `json:"instId"`
	LoanCurrency currency.Code `json:"ccy"`
	Side         string        `json:"side"` // possible values: 'borrow' and 'repay'
}

// BorrowRepayHistoryItem holds borrow or repay history item information
type BorrowRepayHistoryItem struct {
	InstrumentID    string        `json:"instId"`
	Currency        currency.Code `json:"ccy"`
	Side            string        `json:"side"`
	AccBorrowAmount types.Number  `json:"accBorrowed"`
	Amount          types.Number  `json:"amt"`
	RefID           string        `json:"refId"`
	Timestamp       types.Time    `json:"ts"`
}

// QuoteLeg holds one leg of a quote
type QuoteLeg struct {
	InstrumentID       string        `json:"instId"`
	TradeMode          string        `json:"tdMode"`
	Currency           currency.Code `json:"ccy"`
	Size               types.Number  `json:"sz"`
	Price              types.Number  `json:"px"`
	Side               string        `json:"side"`
	PositionSide       string        `json:"posSide"`
	TargetCurrency     string        `json:"tgtCcy"` // base_ccy or quote_ccy, the unit of Size
	TradeQuoteCurrency currency.Code `json:"tradeQuoteCcy"`
}

// PlaceSpreadOrderRequest holds parameters for spread orders. Price is the
// differential between the spread's legs, which can be zero or negative, and
// is sent with every order type but market.
type PlaceSpreadOrderRequest struct {
	SpreadID      string  `json:"sprdId,omitempty"`
	ClientOrderID string  `json:"clOrdId,omitempty"`
	Side          string  `json:"side"`    // Order side, buy sell
	OrderType     string  `json:"ordType"` // Order type  'limit': Limit order  'post_only': Post-only order 'ioc': Immediate-or-cancel order
	Size          float64 `json:"sz,string"`
	Price         float64 `json:"-"`
	Tag           string  `json:"tag,omitempty"`
}

// MarshalJSON sends px with every order type that takes a price, zero
// included, and leaves it out of a market order, which takes none
func (arg *PlaceSpreadOrderRequest) MarshalJSON() ([]byte, error) {
	type alias PlaceSpreadOrderRequest
	body := &struct {
		*alias
		Price string `json:"px,omitempty"`
	}{alias: (*alias)(arg)}
	if !strings.EqualFold(arg.OrderType, orderMarket) {
		body.Price = strconv.FormatFloat(arg.Price, 'f', -1, 64)
	}
	return json.Marshal(body)
}

// Validate checks if the parameters are valid
func (arg *PlaceSpreadOrderRequest) Validate() error {
	if arg == nil {
		return fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.SpreadID == "" {
		return fmt.Errorf("%w, spread ID missing", errMissingInstrumentID)
	}
	if arg.OrderType == "" {
		return fmt.Errorf("%w spread order type is required", order.ErrTypeIsInvalid)
	}
	arg.OrderType = strings.ToLower(arg.OrderType)
	if arg.Size <= 0 {
		return limits.ErrAmountBelowMin
	}
	arg.Side = strings.ToLower(arg.Side)
	switch arg.Side {
	case order.Buy.Lower(), order.Sell.Lower():
	default:
		return fmt.Errorf("%w %s", order.ErrSideIsInvalid, arg.Side)
	}
	return nil
}

// SpreadOrderResponse represents a spread create order response
type SpreadOrderResponse struct {
	StatusCode    uint64 `json:"sCode,string"` // Anything above 0 is an error with an attached message
	StatusMessage string `json:"sMsg"`
	ClientOrderID string `json:"clOrdId"`
	OrderID       string `json:"ordId"`
	Tag           string `json:"tag"`
	// RequestID echoes the request ID of an amendment
	RequestID string `json:"reqId"`

	statusCodeMissing bool
}

// UnmarshalJSON decodes a spread order action's reply row, noting a row without the sCode stating its execution
// result, whose status code would otherwise read as success
func (arg *SpreadOrderResponse) UnmarshalJSON(data []byte) error {
	type row SpreadOrderResponse
	decoded := struct {
		*row
		StatusCode *string `json:"sCode"`
	}{row: (*row)(arg)}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var err error
	arg.StatusCode, arg.statusCodeMissing, err = decodeStatusCode(decoded.StatusCode)
	return err
}

// Error returns the spread order action's status error, nil when it succeeded
func (arg *SpreadOrderResponse) Error() error {
	return replyRowStatusError(arg.StatusCode, arg.StatusMessage, arg.statusCodeMissing)
}

// SpreadCandlestickResponse holds a spread candlestick
type SpreadCandlestickResponse struct {
	Timestamp types.Time
	Open      types.Number
	High      types.Number
	Low       types.Number
	Close     types.Number
	Volume    types.Number
	Confirm   types.Boolean // true once the candlestick is complete
}

// UnmarshalJSON decodes a spread candlestick from its array form
func (s *SpreadCandlestickResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&s.Timestamp, &s.Open, &s.High, &s.Low, &s.Close, &s.Volume, &s.Confirm})
}

// LeverageInfo holds leverage information
type LeverageInfo struct {
	Leverage     types.Number `json:"lever"`
	PositionSide string       `json:"posSide"`
}

// OptionalNumber preserves whether an API numeric field contains a value.
// Empty strings, null and omitted fields are absent; explicit zero is present.
type OptionalNumber struct {
	Value *types.Number
}

// UnmarshalJSON decodes OKX's numeric strings while retaining explicit zero values.
func (n *OptionalNumber) UnmarshalJSON(data []byte) error {
	var value types.Number
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	data = bytes.TrimSpace(data)
	n.Value = nil
	if !bytes.Equal(data, []byte(`null`)) && !bytes.Equal(data, []byte(`""`)) {
		n.Value = &value
	}
	return nil
}

// UnmarshalJSON preserves presence of legacy fee fields, including explicit zero rates.
func (r *TradeFeeRateResponse) UnmarshalJSON(data []byte) error {
	type rate TradeFeeRateResponse
	type response struct {
		*rate
		Maker     OptionalNumber `json:"maker"`
		Taker     OptionalNumber `json:"taker"`
		MakerUSDT OptionalNumber `json:"makerU"`
		TakerUSDT OptionalNumber `json:"takerU"`
		MakerUSDC OptionalNumber `json:"makerUSDC"`
		TakerUSDC OptionalNumber `json:"takerUSDC"`
	}
	decoded := response{rate: (*rate)(r)}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	decoded.FeeRateMaker = decoded.Maker.Value
	decoded.FeeRateTaker = decoded.Taker.Value
	decoded.FeeRateMakerUSDT = decoded.MakerUSDT.Value
	decoded.FeeRateTakerUSDT = decoded.TakerUSDT.Value
	decoded.FeeRateMakerUSDC = decoded.MakerUSDC.Value
	decoded.FeeRateTakerUSDC = decoded.TakerUSDC.Value
	return nil
}

// UnmarshalJSON retains explicit zero rates without treating absent fields as free trading.
func (r *TradeFeeGroup) UnmarshalJSON(data []byte) error {
	type feeGroup TradeFeeGroup
	type response struct {
		*feeGroup
		Maker OptionalNumber `json:"maker"`
		Taker OptionalNumber `json:"taker"`
	}
	decoded := response{feeGroup: (*feeGroup)(r)}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	decoded.MakerRate = decoded.Maker.Value
	decoded.TakerRate = decoded.Taker.Value
	return nil
}
