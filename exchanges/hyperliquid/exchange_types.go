package hyperliquid

import (
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Order time in force values
const (
	// TimeInForceALO adds liquidity only and is cancelled instead of matching immediately
	TimeInForceALO = "Alo"
	// TimeInForceIOC fills immediately and cancels any unfilled remainder
	TimeInForceIOC = "Ioc"
	// TimeInForceGTC rests until filled or cancelled
	TimeInForceGTC = "Gtc"
)

// Order grouping values
const (
	// GroupingNone places independent orders
	GroupingNone = "na"
	// GroupingNormalTPSL places a parent order followed by take-profit and stop-loss children sized to it
	GroupingNormalTPSL = "normalTpsl"
	// GroupingPositionTPSL places take-profit and stop-loss orders sized to the open position
	GroupingPositionTPSL = "positionTpsl"
)

// Trigger order kinds
const (
	TriggerTakeProfit = "tp"
	TriggerStopLoss   = "sl"
)

// Waiting statuses of grouped TP/SL children whose parent has not filled or triggered
const (
	orderStatusWaitingForFill    = "waitingForFill"
	orderStatusWaitingForTrigger = "waitingForTrigger"
)

// Address encodings of a SendToEVMWithData recipient
const (
	AddressEncodingHex    = "hex"
	AddressEncodingBase58 = "base58"
)

var (
	errActionStatusMalformed = errors.New("malformed exchange action status")
	errUnknownCancelStatus   = errors.New("unknown cancel status")
)

// L1ActionOptionsRequest controls how SendSignedAction signs and sends an L1 action
type L1ActionOptionsRequest struct {
	// BatchLength is the length of the action's order, cancel or modify array, which sets its rate limit weight
	BatchLength uint64
	// ActForVault signs and sends the action for the configured vault or subaccount; only trading actions accept one
	ActForVault bool
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
	// Nonce, when set, signs with this nonce instead of the next one, as invalidating a pending nonce requires
	Nonce uint64
	// SetNonce writes the nonce into an action that also carries it, before the action is hashed
	SetNonce func(nonce uint64)
	// Websocket posts the action over the websocket connection instead of the exchange endpoint
	Websocket bool
}

// SignedActionRequest is the body of every exchange endpoint request
type SignedActionRequest struct {
	Action       any             `json:"action"`
	Nonce        uint64          `json:"nonce"`
	Signature    ActionSignature `json:"signature"`
	VaultAddress string          `json:"vaultAddress,omitempty"`
	ExpiresAfter *uint64         `json:"expiresAfter,omitempty"`
}

// ExchangeActionResponse is the envelope of every exchange endpoint response; Response holds an error message unless
// Status is ok
type ExchangeActionResponse struct {
	Status   string          `json:"status"`
	Response json.RawMessage `json:"response"`
}

// ActionStatusesResponse is the response payload of an action that reports one status per batched item
type ActionStatusesResponse[T any] struct {
	Type string                `json:"type"`
	Data ActionStatusesData[T] `json:"data"`
}

// ActionStatusesData contains the statuses of a batched action, in request order
type ActionStatusesData[T any] struct {
	Statuses []T `json:"statuses"`
}

// ActionStatusResponse is the response payload of an action that reports a single status
type ActionStatusResponse[T any] struct {
	Type string              `json:"type"`
	Data ActionStatusData[T] `json:"data"`
}

// ActionStatusData contains the status of an unbatched action
type ActionStatusData[T any] struct {
	Status T `json:"status"`
}

// EmptyAction is the payload of an action without parameters, such as noop or claimRewards
type EmptyAction struct {
	Type string `json:"type" msgpack:"type"`
}

// OrderRequest contains one order of an order or modify action
type OrderRequest struct {
	// Asset is the market's asset ID: its perpetual universe index, 10000 plus its spot index, a builder DEX offset, or
	// 100000000 plus an outcome encoding, which is 10 times the outcome plus its side
	Asset      uint64
	IsBuy      bool
	Price      float64
	Size       float64
	ReduceOnly bool
	// Limit and Trigger are mutually exclusive; exactly one must be set
	Limit         *LimitOrderType
	Trigger       *TriggerOrderType
	ClientOrderID string
}

// LimitOrderType contains a limit order's time in force: TimeInForceALO, TimeInForceIOC or TimeInForceGTC
type LimitOrderType struct {
	TimeInForce string
}

// TriggerOrderType contains a take-profit or stop-loss trigger, which Hyperliquid evaluates against the mark price
type TriggerOrderType struct {
	IsMarket     bool
	TriggerPrice float64
	// TakeProfitStopLoss is TriggerTakeProfit or TriggerStopLoss
	TakeProfitStopLoss string
}

// PlaceOrdersRequest contains a batch of orders to place as one signed action
type PlaceOrdersRequest struct {
	Orders []OrderRequest
	// Grouping is GroupingNone, GroupingNormalTPSL or GroupingPositionTPSL; empty defaults to GroupingNone
	Grouping string
	// PriorityRate sends a priority grouping instead, which charges PriorityRate/1e8 of each order's filled or resting
	// notional from the undelegated staking balance; every order must then be IOC, or every order a non-reduce-only ALO,
	// on a market other than an outcome
	PriorityRate uint64
	// Builder pays a builder an additional fee on each order, which the account must have approved
	Builder *BuilderFee
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// BuilderFee pays a builder an additional fee on each order of an order action
type BuilderFee struct {
	// Builder is the address that receives the fee
	Builder string
	// Fee is at most 100, 0.1%, on perpetuals and 1000, 1%, on spot
	Fee TenthsOfABasisPoint
}

// OrderAction is the order action payload
type OrderAction struct {
	Type   string      `json:"type"   msgpack:"type"`
	Orders []OrderWire `json:"orders" msgpack:"orders"`
	// Grouping is a grouping string or a PriorityGroupingWire
	Grouping any             `json:"grouping"          msgpack:"grouping"`
	Builder  *BuilderFeeWire `json:"builder,omitempty" msgpack:"builder,omitempty"`
}

// OrderWire is an order as an order or modify action signs it; L1 actions are hashed as msgpack in field declaration
// order, so the wire types' field order must match Hyperliquid's own encoding
type OrderWire struct {
	Asset         uint64        `json:"a"           msgpack:"a"`
	IsBuy         bool          `json:"b"           msgpack:"b"`
	Price         string        `json:"p"           msgpack:"p"`
	Size          string        `json:"s"           msgpack:"s"`
	ReduceOnly    bool          `json:"r"           msgpack:"r"`
	Type          OrderTypeWire `json:"t"           msgpack:"t"`
	ClientOrderID string        `json:"c,omitempty" msgpack:"c,omitempty"`
}

// OrderTypeWire contains an order's limit or trigger type; exactly one is set
type OrderTypeWire struct {
	Limit   *LimitOrderTypeWire   `json:"limit,omitempty"   msgpack:"limit,omitempty"`
	Trigger *TriggerOrderTypeWire `json:"trigger,omitempty" msgpack:"trigger,omitempty"`
}

// LimitOrderTypeWire contains a limit order's time in force
type LimitOrderTypeWire struct {
	TimeInForce string `json:"tif" msgpack:"tif"`
}

// TriggerOrderTypeWire contains a trigger order's market flag, trigger price and take-profit or stop-loss kind
type TriggerOrderTypeWire struct {
	IsMarket           bool   `json:"isMarket"  msgpack:"isMarket"`
	TriggerPrice       string `json:"triggerPx" msgpack:"triggerPx"`
	TakeProfitStopLoss string `json:"tpsl"      msgpack:"tpsl"`
}

// PriorityGroupingWire is the priority grouping of an order action
type PriorityGroupingWire struct {
	PriorityRate uint64 `json:"p" msgpack:"p"`
}

// BuilderFeeWire is the builder fee of an order action
type BuilderFeeWire struct {
	Builder string              `json:"b" msgpack:"b"`
	Fee     TenthsOfABasisPoint `json:"f" msgpack:"f"`
}

// OrderActionStatusResponse contains the outcome of one order in an order or batchModify action; exactly one field is set
type OrderActionStatusResponse struct {
	Resting *RestingOrderStatus `json:"resting"`
	Filled  *FilledOrderStatus  `json:"filled"`
	Error   string              `json:"error"`
	// Waiting is set for a grouped TP/SL child that awaits its parent's fill or trigger
	Waiting string `json:"-"`
}

// UnmarshalJSON decodes either a waiting status string or a resting, filled or error object
func (o *OrderActionStatusResponse) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var waiting string
		if err := json.Unmarshal(data, &waiting); err != nil {
			return err
		}
		switch waiting {
		case orderStatusWaitingForFill, orderStatusWaitingForTrigger:
			*o = OrderActionStatusResponse{Waiting: waiting}
			return nil
		default:
			return fmt.Errorf("%w: unknown status %q", errActionStatusMalformed, waiting)
		}
	}
	type alias OrderActionStatusResponse
	var status alias
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}
	variants := 0
	if status.Resting != nil {
		if status.Resting.OrderID == 0 {
			return fmt.Errorf("%w: resting order ID is zero", errActionStatusMalformed)
		}
		variants++
	}
	if status.Filled != nil {
		if status.Filled.OrderID == 0 {
			return fmt.Errorf("%w: filled order ID is zero", errActionStatusMalformed)
		}
		variants++
	}
	if status.Error != "" {
		variants++
	}
	if variants != 1 {
		return fmt.Errorf("%w: %s", errActionStatusMalformed, data)
	}
	*o = OrderActionStatusResponse(status)
	return nil
}

// RestingOrderStatus contains the ID of an order resting on the book
type RestingOrderStatus struct {
	OrderID       uint64 `json:"oid"`
	ClientOrderID string `json:"cloid"`
}

// FilledOrderStatus contains the fill of an order that matched on placement
type FilledOrderStatus struct {
	TotalSize     types.Number `json:"totalSz"`
	AveragePrice  types.Number `json:"avgPx"`
	OrderID       uint64       `json:"oid"`
	ClientOrderID string       `json:"cloid"`
}

// CancelOrdersRequest cancels a batch of orders by order ID as one signed action
type CancelOrdersRequest struct {
	Cancels []CancelRequest
	// Fast rejects cancels of trigger orders; Hyperliquid plans to prioritise fast cancels in the mempool
	Fast bool
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// CancelRequest identifies an order to cancel by its asset and order ID
type CancelRequest struct {
	Asset   uint64 `json:"a" msgpack:"a"`
	OrderID uint64 `json:"o" msgpack:"o"`
}

// CancelAction is the cancel action payload
type CancelAction struct {
	Type    string          `json:"type"        msgpack:"type"`
	Cancels []CancelRequest `json:"cancels"     msgpack:"cancels"`
	Fast    bool            `json:"f,omitempty" msgpack:"f,omitempty"`
}

// CancelActionStatusResponse contains the outcome of one cancel in a cancel action; Error is empty when it succeeded
type CancelActionStatusResponse struct {
	Error string `json:"error"`
}

// UnmarshalJSON decodes either a success status string or an error object
func (c *CancelActionStatusResponse) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var status string
		if err := json.Unmarshal(data, &status); err != nil {
			return err
		}
		if status != "success" {
			return fmt.Errorf("%w: %q", errUnknownCancelStatus, status)
		}
		*c = CancelActionStatusResponse{}
		return nil
	}
	type alias CancelActionStatusResponse
	var status alias
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}
	if status.Error == "" {
		return fmt.Errorf("%w: %s", errActionStatusMalformed, data)
	}
	*c = CancelActionStatusResponse(status)
	return nil
}

// CancelOrdersByClientOrderIDRequest cancels a batch of orders by client order ID as one signed action
type CancelOrdersByClientOrderIDRequest struct {
	Cancels []CancelByClientOrderIDRequest
	// Fast rejects cancels of trigger orders; Hyperliquid plans to prioritise fast cancels in the mempool
	Fast bool
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// CancelByClientOrderIDRequest identifies an order to cancel by its asset and client order ID
type CancelByClientOrderIDRequest struct {
	Asset         uint64 `json:"asset" msgpack:"asset"`
	ClientOrderID string `json:"cloid" msgpack:"cloid"`
}

// CancelByClientOrderIDAction is the cancelByCloid action payload
type CancelByClientOrderIDAction struct {
	Type    string                         `json:"type"        msgpack:"type"`
	Cancels []CancelByClientOrderIDRequest `json:"cancels"     msgpack:"cancels"`
	Fast    bool                           `json:"f,omitempty" msgpack:"f,omitempty"`
}

// ScheduleCancelRequest schedules or clears a cancellation of every open order, a dead man's switch
type ScheduleCancelRequest struct {
	// Time is when every open order is cancelled, at least 5 seconds ahead; zero clears the schedule
	Time time.Time
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// ScheduleCancelAction is the scheduleCancel action payload; a nil Time clears the schedule
type ScheduleCancelAction struct {
	Type string  `json:"type"           msgpack:"type"`
	Time *uint64 `json:"time,omitempty" msgpack:"time,omitempty"`
}

// ModifyOrderRequest replaces one order, identified by its order ID or client order ID, with the modify action
// Without AlwaysPlace, Hyperliquid only accepts a replacement that is not a trigger order and is either ALO or a GTC that
// would not execute, in which case it rests as ALO
type ModifyOrderRequest struct {
	OrderID       uint64
	ClientOrderID string
	Order         OrderRequest
	// AlwaysPlace places the replacement even when the cancel fails, allowing trigger and IOC replacements
	AlwaysPlace bool
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// ModifyAction is the modify action payload
type ModifyAction struct {
	Type string `json:"type" msgpack:"type"`
	// OrderID is a uint64 order ID or a lower-case client order ID
	OrderID     any       `json:"oid"         msgpack:"oid"`
	Order       OrderWire `json:"order"       msgpack:"order"`
	AlwaysPlace bool      `json:"a,omitempty" msgpack:"a,omitempty"`
}

// ModifyOrdersRequest replaces a batch of orders with one batchModify action
// Without AlwaysPlace, Hyperliquid only accepts replacements that are not trigger orders and are either ALO or GTCs that
// would not execute, which then rest as ALO
type ModifyOrdersRequest struct {
	Modifies []OrderModification
	// AlwaysPlace places every replacement even when its cancel fails, allowing trigger and IOC replacements
	AlwaysPlace bool
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// OrderModification identifies an order by its order ID or client order ID, and its replacement
type OrderModification struct {
	OrderID       uint64
	ClientOrderID string
	Order         OrderRequest
}

// ModifyWire is one replacement of a batchModify action
type ModifyWire struct {
	// OrderID is a uint64 order ID or a lower-case client order ID
	OrderID any       `json:"oid"   msgpack:"oid"`
	Order   OrderWire `json:"order" msgpack:"order"`
}

// BatchModifyAction is the batchModify action payload
type BatchModifyAction struct {
	Type        string       `json:"type"        msgpack:"type"`
	Modifies    []ModifyWire `json:"modifies"    msgpack:"modifies"`
	AlwaysPlace bool         `json:"a,omitempty" msgpack:"a,omitempty"`
}

// UpdateLeverageRequest sets a perpetual market's margin mode and leverage
type UpdateLeverageRequest struct {
	Asset    uint64
	IsCross  bool
	Leverage uint64
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// UpdateLeverageAction is the updateLeverage action payload
type UpdateLeverageAction struct {
	Type     string `json:"type"     msgpack:"type"`
	Asset    uint64 `json:"asset"    msgpack:"asset"`
	IsCross  bool   `json:"isCross"  msgpack:"isCross"`
	Leverage uint64 `json:"leverage" msgpack:"leverage"`
}

// UpdateIsolatedMarginRequest adds margin to, or removes it from, an isolated position
type UpdateIsolatedMarginRequest struct {
	Asset uint64
	// SignedNotional is the USDC margin change with 6 decimals: 1000000 adds 1 USDC and a negative value removes margin
	SignedNotional int64
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// UpdateIsolatedMarginAction is the updateIsolatedMargin action payload
type UpdateIsolatedMarginAction struct {
	Type  string `json:"type"  msgpack:"type"`
	Asset uint64 `json:"asset" msgpack:"asset"`
	// IsBuy is always true, as Hyperliquid ignores it until hedge mode exists
	IsBuy bool `json:"isBuy" msgpack:"isBuy"`
	// SignedNotional is the USDC margin change with 6 decimals, as in UpdateIsolatedMarginRequest; a negative value
	// removes margin
	SignedNotional int64 `json:"ntli" msgpack:"ntli"`
}

// TopUpIsolatedOnlyMarginRequest sets an isolated-only position's margin to reach a target leverage
type TopUpIsolatedOnlyMarginRequest struct {
	Asset    uint64
	Leverage float64
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// TopUpIsolatedOnlyMarginAction is the topUpIsolatedOnlyMargin action payload
type TopUpIsolatedOnlyMarginAction struct {
	Type     string `json:"type"     msgpack:"type"`
	Asset    uint64 `json:"asset"    msgpack:"asset"`
	Leverage string `json:"leverage" msgpack:"leverage"`
}

// SendAssetRequest transfers a token between perpetual DEX balances, spot, users and owned subaccounts
type SendAssetRequest struct {
	Destination string
	// SourceDEX and DestinationDEX are perpetual DEX names, empty for the default DEX or spot for the spot balance
	SourceDEX      string
	DestinationDEX string
	// Token is the perpetual DEX collateral token when either side is a perpetual DEX
	Token  currency.Code
	Amount float64
}

// SendAssetAction is the sendAsset action payload
type SendAssetAction struct {
	Type             string `json:"type"`
	SignatureChainID string `json:"signatureChainId"`
	HyperliquidChain string `json:"hyperliquidChain"`
	Destination      string `json:"destination"`
	// SourceDEX and DestinationDEX are perpetual DEX names, empty for the default DEX or spot for the spot balance
	SourceDEX      string `json:"sourceDex"`
	DestinationDEX string `json:"destinationDex"`
	// Token is USDC, or a NAME:TOKEN_ID identifier for any other token
	Token  string `json:"token"`
	Amount string `json:"amount"`
	// FromSubAccount is the subaccount the token is sent from, empty for the account
	FromSubAccount string `json:"fromSubAccount"`
	Nonce          uint64 `json:"nonce"`
}

func (a *SendAssetAction) setEnvelope(chain string, nonce uint64) {
	a.Type, a.SignatureChainID, a.HyperliquidChain, a.Nonce = "sendAsset", userSignedChainIDHex, chain, nonce
}

func (a *SendAssetAction) signingFields() (primaryType string, fields []eip712Field) {
	return "HyperliquidTransaction:SendAsset", userSignedFields(
		a.HyperliquidChain, nonceFieldNonce, a.Nonce,
		eip712Field{Name: "destination", Type: eip712TypeString, Value: a.Destination},
		eip712Field{Name: "sourceDex", Type: eip712TypeString, Value: a.SourceDEX},
		eip712Field{Name: "destinationDex", Type: eip712TypeString, Value: a.DestinationDEX},
		eip712Field{Name: "token", Type: eip712TypeString, Value: a.Token},
		eip712Field{Name: userSignedAmountField, Type: eip712TypeString, Value: a.Amount},
		eip712Field{Name: "fromSubAccount", Type: eip712TypeString, Value: a.FromSubAccount},
	)
}

// AgentSendAssetRequest transfers a token as SendAssetRequest does, through an L1 action that an approved API wallet can
// sign; the destination must be the account or one of its subaccounts
type AgentSendAssetRequest struct {
	Destination string
	// SourceDEX and DestinationDEX are perpetual DEX names, empty for the default DEX or spot for the spot balance
	SourceDEX      string
	DestinationDEX string
	// Token is the perpetual DEX collateral token when either side is a perpetual DEX
	Token  currency.Code
	Amount float64
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// AgentSendAssetAction is the agentSendAsset action payload, which an API wallet can sign; Nonce must equal the
// request nonce
type AgentSendAssetAction struct {
	Type           string `json:"type"           msgpack:"type"`
	Destination    string `json:"destination"    msgpack:"destination"`
	SourceDEX      string `json:"sourceDex"      msgpack:"sourceDex"`
	DestinationDEX string `json:"destinationDex" msgpack:"destinationDex"`
	Token          string `json:"token"          msgpack:"token"`
	Amount         string `json:"amount"         msgpack:"amount"`
	FromSubAccount string `json:"fromSubAccount" msgpack:"fromSubAccount"`
	Nonce          uint64 `json:"nonce"          msgpack:"nonce"`
}

// SendToEVMWithDataRequest transfers a token from HyperCore to a HyperEVM contract with a data payload; the token's
// linked contract must implement ICoreReceiveWithData
type SendToEVMWithDataRequest struct {
	Token  currency.Code
	Amount float64
	// SourceDEX is a perpetual DEX name, empty for the default DEX, or spot for the spot balance
	SourceDEX            string
	DestinationRecipient string
	// AddressEncoding is AddressEncodingHex or AddressEncodingBase58, the encoding of DestinationRecipient
	AddressEncoding    string
	DestinationChainID uint32
	GasLimit           uint64
	Data               []byte
}

// SendToEVMWithDataAction is the sendToEvmWithData action payload
type SendToEVMWithDataAction struct {
	Type             string `json:"type"`
	SignatureChainID string `json:"signatureChainId"`
	HyperliquidChain string `json:"hyperliquidChain"`
	// Token is a NAME:TOKEN_ID identifier
	Token  string `json:"token"`
	Amount string `json:"amount"`
	// SourceDEX is a perpetual DEX name, empty for the default DEX, or spot for the spot balance
	SourceDEX            string `json:"sourceDex"`
	DestinationRecipient string `json:"destinationRecipient"`
	// AddressEncoding is AddressEncodingHex or AddressEncodingBase58, the encoding of DestinationRecipient
	AddressEncoding    string `json:"addressEncoding"`
	DestinationChainID uint32 `json:"destinationChainId"`
	GasLimit           uint64 `json:"gasLimit"`
	// Data is 0x-prefixed hexadecimal, and 0x when empty
	Data  string `json:"data"`
	Nonce uint64 `json:"nonce"`
}

func (a *SendToEVMWithDataAction) setEnvelope(chain string, nonce uint64) {
	a.Type, a.SignatureChainID, a.HyperliquidChain, a.Nonce = "sendToEvmWithData", userSignedChainIDHex, chain, nonce
}

func (a *SendToEVMWithDataAction) signingFields() (primaryType string, fields []eip712Field) {
	return "HyperliquidTransaction:SendToEvmWithData", userSignedFields(
		a.HyperliquidChain, nonceFieldNonce, a.Nonce,
		eip712Field{Name: "token", Type: eip712TypeString, Value: a.Token},
		eip712Field{Name: userSignedAmountField, Type: eip712TypeString, Value: a.Amount},
		eip712Field{Name: "sourceDex", Type: eip712TypeString, Value: a.SourceDEX},
		eip712Field{Name: "destinationRecipient", Type: eip712TypeString, Value: a.DestinationRecipient},
		eip712Field{Name: "addressEncoding", Type: eip712TypeString, Value: a.AddressEncoding},
		eip712Field{Name: "destinationChainId", Type: eip712TypeUint32, Value: a.DestinationChainID},
		eip712Field{Name: "gasLimit", Type: eip712TypeUint64, Value: a.GasLimit},
		eip712Field{Name: "data", Type: eip712TypeBytes, Value: a.Data},
	)
}

// SendCoreUSDCRequest sends USDC from the default perpetual DEX balance to another address
type SendCoreUSDCRequest struct {
	Destination string
	Amount      float64
}

// USDSendAction is the usdSend action payload, which sends USDC from the default perpetual DEX balance to another
// address
type USDSendAction struct {
	Type             string `json:"type"`
	SignatureChainID string `json:"signatureChainId"`
	HyperliquidChain string `json:"hyperliquidChain"`
	Destination      string `json:"destination"`
	Amount           string `json:"amount"`
	// Time is the action's nonce
	Time uint64 `json:"time"`
}

func (a *USDSendAction) setEnvelope(chain string, nonce uint64) {
	a.Type, a.SignatureChainID, a.HyperliquidChain, a.Time = "usdSend", userSignedChainIDHex, chain, nonce
}

func (a *USDSendAction) signingFields() (primaryType string, fields []eip712Field) {
	return "HyperliquidTransaction:UsdSend", userSignedFields(
		a.HyperliquidChain, nonceFieldTime, a.Time,
		eip712Field{Name: "destination", Type: eip712TypeString, Value: a.Destination},
		eip712Field{Name: userSignedAmountField, Type: eip712TypeString, Value: a.Amount},
	)
}

// SendCoreSpotRequest sends a spot token to another address
type SendCoreSpotRequest struct {
	Destination string
	Token       currency.Code
	Amount      float64
}

// SpotSendAction is the spotSend action payload, which sends a spot token to another address
type SpotSendAction struct {
	Type             string `json:"type"`
	SignatureChainID string `json:"signatureChainId"`
	HyperliquidChain string `json:"hyperliquidChain"`
	Destination      string `json:"destination"`
	// Token is a NAME:TOKEN_ID identifier
	Token  string `json:"token"`
	Amount string `json:"amount"`
	// Time is the action's nonce
	Time uint64 `json:"time"`
}

func (a *SpotSendAction) setEnvelope(chain string, nonce uint64) {
	a.Type, a.SignatureChainID, a.HyperliquidChain, a.Time = "spotSend", userSignedChainIDHex, chain, nonce
}

func (a *SpotSendAction) signingFields() (primaryType string, fields []eip712Field) {
	return "HyperliquidTransaction:SpotSend", userSignedFields(
		a.HyperliquidChain, nonceFieldTime, a.Time,
		eip712Field{Name: "destination", Type: eip712TypeString, Value: a.Destination},
		eip712Field{Name: "token", Type: eip712TypeString, Value: a.Token},
		eip712Field{Name: userSignedAmountField, Type: eip712TypeString, Value: a.Amount},
	)
}

// TransferUSDCBetweenSpotAndPerpRequest transfers USDC between the spot and default perpetual DEX balances; ToPerpetual
// moves it from spot to perpetuals
type TransferUSDCBetweenSpotAndPerpRequest struct {
	Amount      float64
	ToPerpetual bool
}

// USDClassTransferAction is the usdClassTransfer action payload, which transfers USDC between the spot and default
// perpetual DEX balances
type USDClassTransferAction struct {
	Type             string `json:"type"`
	SignatureChainID string `json:"signatureChainId"`
	HyperliquidChain string `json:"hyperliquidChain"`
	// Amount ends with " subaccount:" and the subaccount's address for a transfer of a subaccount's balances
	Amount      string `json:"amount"`
	ToPerpetual bool   `json:"toPerp"`
	Nonce       uint64 `json:"nonce"`
}

func (a *USDClassTransferAction) setEnvelope(chain string, nonce uint64) {
	a.Type, a.SignatureChainID, a.HyperliquidChain, a.Nonce = "usdClassTransfer", userSignedChainIDHex, chain, nonce
}

func (a *USDClassTransferAction) signingFields() (primaryType string, fields []eip712Field) {
	return "HyperliquidTransaction:UsdClassTransfer", userSignedFields(
		a.HyperliquidChain, nonceFieldNonce, a.Nonce,
		eip712Field{Name: userSignedAmountField, Type: eip712TypeString, Value: a.Amount},
		eip712Field{Name: "toPerp", Type: eip712TypeBool, Value: a.ToPerpetual},
	)
}

// CDepositAction is the cDeposit action payload, which deposits HYPE from spot into staking
type CDepositAction struct {
	Type             string `json:"type"`
	SignatureChainID string `json:"signatureChainId"`
	HyperliquidChain string `json:"hyperliquidChain"`
	// AmountInWei is HYPE with 8 decimals
	AmountInWei uint64 `json:"wei"`
	Nonce       uint64 `json:"nonce"`
}

func (a *CDepositAction) setEnvelope(chain string, nonce uint64) {
	a.Type, a.SignatureChainID, a.HyperliquidChain, a.Nonce = "cDeposit", userSignedChainIDHex, chain, nonce
}

func (a *CDepositAction) signingFields() (primaryType string, fields []eip712Field) {
	return "HyperliquidTransaction:CDeposit", userSignedFields(
		a.HyperliquidChain, nonceFieldNonce, a.Nonce,
		eip712Field{Name: "wei", Type: eip712TypeUint64, Value: a.AmountInWei},
	)
}

// CWithdrawAction is the cWithdraw action payload, which withdraws HYPE from staking to spot
type CWithdrawAction struct {
	Type             string `json:"type"`
	SignatureChainID string `json:"signatureChainId"`
	HyperliquidChain string `json:"hyperliquidChain"`
	// AmountInWei is HYPE with 8 decimals
	AmountInWei uint64 `json:"wei"`
	Nonce       uint64 `json:"nonce"`
}

func (a *CWithdrawAction) setEnvelope(chain string, nonce uint64) {
	a.Type, a.SignatureChainID, a.HyperliquidChain, a.Nonce = "cWithdraw", userSignedChainIDHex, chain, nonce
}

func (a *CWithdrawAction) signingFields() (primaryType string, fields []eip712Field) {
	return "HyperliquidTransaction:CWithdraw", userSignedFields(
		a.HyperliquidChain, nonceFieldNonce, a.Nonce,
		eip712Field{Name: "wei", Type: eip712TypeUint64, Value: a.AmountInWei},
	)
}

// DelegateStakeRequest delegates HYPE to, or undelegates it from, a validator
type DelegateStakeRequest struct {
	Validator string
	// AmountInWei is HYPE with 8 decimals
	AmountInWei  uint64
	IsUndelegate bool
}

// TokenDelegateAction is the tokenDelegate action payload
type TokenDelegateAction struct {
	Type             string `json:"type"`
	SignatureChainID string `json:"signatureChainId"`
	HyperliquidChain string `json:"hyperliquidChain"`
	Validator        string `json:"validator"`
	// AmountInWei is HYPE with 8 decimals
	AmountInWei  uint64 `json:"wei"`
	IsUndelegate bool   `json:"isUndelegate"`
	Nonce        uint64 `json:"nonce"`
}

func (a *TokenDelegateAction) setEnvelope(chain string, nonce uint64) {
	a.Type, a.SignatureChainID, a.HyperliquidChain, a.Nonce = "tokenDelegate", userSignedChainIDHex, chain, nonce
}

func (a *TokenDelegateAction) signingFields() (primaryType string, fields []eip712Field) {
	return "HyperliquidTransaction:TokenDelegate", userSignedFields(
		a.HyperliquidChain, nonceFieldNonce, a.Nonce,
		eip712Field{Name: "validator", Type: eip712TypeAddress, Value: a.Validator},
		eip712Field{Name: "wei", Type: eip712TypeUint64, Value: a.AmountInWei},
		eip712Field{Name: "isUndelegate", Type: eip712TypeBool, Value: a.IsUndelegate},
	)
}

// VaultTransferRequest deposits USDC into, or withdraws it from, a vault
type VaultTransferRequest struct {
	VaultAddress string
	IsDeposit    bool
	// AmountInMicroUSDC is USDC with 6 decimals
	AmountInMicroUSDC uint64
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// VaultTransferAction is the vaultTransfer action payload
type VaultTransferAction struct {
	Type              string `json:"type"         msgpack:"type"`
	VaultAddress      string `json:"vaultAddress" msgpack:"vaultAddress"`
	IsDeposit         bool   `json:"isDeposit"    msgpack:"isDeposit"`
	AmountInMicroUSDC uint64 `json:"usd"          msgpack:"usd"`
}

// HIP3LiquidatorTransferRequest deposits into, or withdraws from, a HIP-3 DEX's backstop liquidator; only the deposited
// principal can be withdrawn
type HIP3LiquidatorTransferRequest struct {
	DEX string
	// NotionalInMicroQuoteTokens must be a multiple of 1000 quote tokens
	NotionalInMicroQuoteTokens uint64
	IsDeposit                  bool
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// HIP3LiquidatorTransferAction is the hip3LiquidatorTransfer action payload
type HIP3LiquidatorTransferAction struct {
	Type                       string `json:"type"      msgpack:"type"`
	DEX                        string `json:"dex"       msgpack:"dex"`
	NotionalInMicroQuoteTokens uint64 `json:"ntl"       msgpack:"ntl"`
	IsDeposit                  bool   `json:"isDeposit" msgpack:"isDeposit"`
}

// ApproveAgentRequest approves an API wallet to trade for the account; approving a new unnamed agent, or reusing a name,
// replaces the previous agent
type ApproveAgentRequest struct {
	AgentAddress string
	// AgentName is at most 16 characters; empty approves the account's unnamed agent
	AgentName string
	// ValidUntil, when set, expires the agent at most 180 days ahead, and requires a name
	ValidUntil time.Time
}

// ApproveAgentAction is the approveAgent action payload
type ApproveAgentAction struct {
	Type             string `json:"type"`
	SignatureChainID string `json:"signatureChainId"`
	HyperliquidChain string `json:"hyperliquidChain"`
	AgentAddress     string `json:"agentAddress"`
	// AgentName is empty for the unnamed agent, and an expiring agent's name ends with " valid_until " and its expiry in
	// Unix milliseconds
	AgentName string `json:"agentName"`
	Nonce     uint64 `json:"nonce"`
}

func (a *ApproveAgentAction) setEnvelope(chain string, nonce uint64) {
	a.Type, a.SignatureChainID, a.HyperliquidChain, a.Nonce = "approveAgent", userSignedChainIDHex, chain, nonce
}

func (a *ApproveAgentAction) signingFields() (primaryType string, fields []eip712Field) {
	return "HyperliquidTransaction:ApproveAgent", userSignedFields(
		a.HyperliquidChain, nonceFieldNonce, a.Nonce,
		eip712Field{Name: "agentAddress", Type: eip712TypeAddress, Value: a.AgentAddress},
		eip712Field{Name: "agentName", Type: eip712TypeString, Value: a.AgentName},
	)
}

// ApproveBuilderFeeRequest approves the maximum fee a builder may charge the account
type ApproveBuilderFeeRequest struct {
	Builder string
	// MaxFee is the highest builder fee the account allows; zero allows none
	MaxFee TenthsOfABasisPoint
}

// ApproveBuilderFeeAction is the approveBuilderFee action payload
type ApproveBuilderFeeAction struct {
	Type             string `json:"type"`
	SignatureChainID string `json:"signatureChainId"`
	HyperliquidChain string `json:"hyperliquidChain"`
	// MaxFeeRate is a percentage followed by %, such as 0.001%
	MaxFeeRate string `json:"maxFeeRate"`
	Builder    string `json:"builder"`
	Nonce      uint64 `json:"nonce"`
}

func (a *ApproveBuilderFeeAction) setEnvelope(chain string, nonce uint64) {
	a.Type, a.SignatureChainID, a.HyperliquidChain, a.Nonce = "approveBuilderFee", userSignedChainIDHex, chain, nonce
}

func (a *ApproveBuilderFeeAction) signingFields() (primaryType string, fields []eip712Field) {
	return "HyperliquidTransaction:ApproveBuilderFee", userSignedFields(
		a.HyperliquidChain, nonceFieldNonce, a.Nonce,
		eip712Field{Name: "maxFeeRate", Type: eip712TypeString, Value: a.MaxFeeRate},
		eip712Field{Name: "builder", Type: eip712TypeAddress, Value: a.Builder},
	)
}

// TWAPOrderRequest places a time-weighted average price order
type TWAPOrderRequest struct {
	Asset      uint64
	IsBuy      bool
	Size       float64
	ReduceOnly bool
	// Minutes is the order's duration, 5 to 1440
	Minutes uint64
	// Randomise varies the timing of the order's slices
	Randomise bool
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// TWAPOrderAction is the twapOrder action payload
type TWAPOrderAction struct {
	Type string   `json:"type" msgpack:"type"`
	TWAP TWAPWire `json:"twap" msgpack:"twap"`
}

// TWAPWire is a TWAP order as a twapOrder action signs it
type TWAPWire struct {
	Asset      uint64 `json:"a" msgpack:"a"`
	IsBuy      bool   `json:"b" msgpack:"b"`
	Size       string `json:"s" msgpack:"s"`
	ReduceOnly bool   `json:"r" msgpack:"r"`
	Minutes    uint64 `json:"m" msgpack:"m"`
	Randomise  bool   `json:"t" msgpack:"t"`
}

// TWAPOrderStatusResponse contains the outcome of a TWAP order; exactly one field is set
type TWAPOrderStatusResponse struct {
	Running *TWAPRunningStatus `json:"running"`
	Error   string             `json:"error"`
}

// UnmarshalJSON decodes a running or error object, rejecting a status that sets neither or both
func (t *TWAPOrderStatusResponse) UnmarshalJSON(data []byte) error {
	type alias TWAPOrderStatusResponse
	var status alias
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}
	if status.Running != nil && status.Running.TWAPID == 0 {
		return fmt.Errorf("%w: running TWAP ID is zero", errActionStatusMalformed)
	}
	if (status.Running == nil) == (status.Error == "") {
		return fmt.Errorf("%w: %s", errActionStatusMalformed, data)
	}
	*t = TWAPOrderStatusResponse(status)
	return nil
}

// TWAPRunningStatus contains the ID of a running TWAP order
type TWAPRunningStatus struct {
	TWAPID uint64 `json:"twapId"`
}

// TWAPCancelRequest cancels a running TWAP order
type TWAPCancelRequest struct {
	Asset  uint64
	TWAPID uint64
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// TWAPCancelAction is the twapCancel action payload
type TWAPCancelAction struct {
	Type   string `json:"type" msgpack:"type"`
	Asset  uint64 `json:"a"    msgpack:"a"`
	TWAPID uint64 `json:"t"    msgpack:"t"`
}

// TrailingStopRequest places a trailing stop order whose trigger follows the mark price
type TrailingStopRequest struct {
	Asset      uint64
	IsBuy      bool
	Size       float64
	ReduceOnly bool
	// RetracementPercent, such as 1.234 for 1.234%, and RetracementPrice are mutually exclusive; exactly one must be set
	RetracementPercent float64
	RetracementPrice   float64
	// ActivationPrice starts tracking once the mark price reaches it; zero tracks immediately
	ActivationPrice float64
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// TrailingStopAction is the trailingStop action payload; a nil ActivationPrice starts tracking immediately
type TrailingStopAction struct {
	Type            string          `json:"type"         msgpack:"type"`
	Asset           uint64          `json:"asset"        msgpack:"asset"`
	IsBuy           bool            `json:"isBuy"        msgpack:"isBuy"`
	Size            string          `json:"sz"           msgpack:"sz"`
	ReduceOnly      bool            `json:"reduceOnly"   msgpack:"reduceOnly"`
	Retracement     RetracementWire `json:"retracement"  msgpack:"retracement"`
	ActivationPrice *string         `json:"activationPx" msgpack:"activationPx"`
}

// RetracementWire is a trailing stop's retracement; exactly one field is set
type RetracementWire struct {
	// Percent is a percentage followed by %, such as 1.234%
	Percent string `json:"pct,omitempty" msgpack:"pct,omitempty"`
	Price   string `json:"px,omitempty"  msgpack:"px,omitempty"`
}

// TrailingStopResponse is the response payload of a trailing stop order
type TrailingStopResponse struct {
	Type string           `json:"type"`
	Data TrailingStopData `json:"data"`
}

// TrailingStopData contains a placed trailing stop order's ID
type TrailingStopData struct {
	OrderID uint64 `json:"oid"`
}

// ReserveRequestWeightRequest buys additional address-based action capacity, at 0.0005 USDC a request from the
// perpetual balance
type ReserveRequestWeightRequest struct {
	Weight uint64
	// Destination pays for another existing user; empty reserves capacity for the signer's account
	Destination string
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// ReserveRequestWeightAction is the reserveRequestWeight action payload
type ReserveRequestWeightAction struct {
	Type        string `json:"type"                  msgpack:"type"`
	Weight      uint64 `json:"weight"                msgpack:"weight"`
	Destination string `json:"destination,omitempty" msgpack:"destination,omitempty"`
}

// InvalidatePendingNonceRequest invalidates an action that has not yet been processed by signing a noop with its nonce
type InvalidatePendingNonceRequest struct {
	Nonce uint64
	// ExpiresAfter, when set, makes Hyperliquid reject the noop after this time
	ExpiresAfter time.Time
}

// UserDEXAbstractionRequest enables or disables HIP-3 DEX abstraction; SetUserAbstraction supersedes it
type UserDEXAbstractionRequest struct {
	// User is the account or one of its subaccounts; empty selects the configured subaccount, else the account
	User    string
	Enabled bool
}

// UserDEXAbstractionAction is the userDexAbstraction action payload
type UserDEXAbstractionAction struct {
	Type             string `json:"type"`
	SignatureChainID string `json:"signatureChainId"`
	HyperliquidChain string `json:"hyperliquidChain"`
	User             string `json:"user"`
	Enabled          bool   `json:"enabled"`
	Nonce            uint64 `json:"nonce"`
}

func (a *UserDEXAbstractionAction) setEnvelope(chain string, nonce uint64) {
	a.Type, a.SignatureChainID, a.HyperliquidChain, a.Nonce = "userDexAbstraction", userSignedChainIDHex, chain, nonce
}

func (a *UserDEXAbstractionAction) signingFields() (primaryType string, fields []eip712Field) {
	return "HyperliquidTransaction:UserDexAbstraction", userSignedFields(
		a.HyperliquidChain, nonceFieldNonce, a.Nonce,
		eip712Field{Name: "user", Type: eip712TypeAddress, Value: a.User},
		eip712Field{Name: "enabled", Type: eip712TypeBool, Value: a.Enabled},
	)
}

// SetUserAbstractionRequest sets how an account shares balances across spot and perpetual DEXs
type SetUserAbstractionRequest struct {
	// User is the account or one of its subaccounts; empty selects the configured subaccount, else the account
	User string
	// Abstraction is AccountAbstractionDisabled, AccountAbstractionUnifiedAccount or AccountAbstractionPortfolioMargin
	Abstraction AccountAbstraction
}

// UserSetAbstractionAction is the userSetAbstraction action payload
type UserSetAbstractionAction struct {
	Type             string             `json:"type"`
	SignatureChainID string             `json:"signatureChainId"`
	HyperliquidChain string             `json:"hyperliquidChain"`
	User             string             `json:"user"`
	Abstraction      AccountAbstraction `json:"abstraction"`
	Nonce            uint64             `json:"nonce"`
}

func (a *UserSetAbstractionAction) setEnvelope(chain string, nonce uint64) {
	a.Type, a.SignatureChainID, a.HyperliquidChain, a.Nonce = "userSetAbstraction", userSignedChainIDHex, chain, nonce
}

func (a *UserSetAbstractionAction) signingFields() (primaryType string, fields []eip712Field) {
	return "HyperliquidTransaction:UserSetAbstraction", userSignedFields(
		a.HyperliquidChain, nonceFieldNonce, a.Nonce,
		eip712Field{Name: "user", Type: eip712TypeAddress, Value: a.User},
		eip712Field{Name: "abstraction", Type: eip712TypeString, Value: string(a.Abstraction)},
	)
}

// AgentSetAbstractionAction is the agentSetAbstraction action payload, whose abstraction is i for disabled, u for a
// unified account or p for portfolio margin
type AgentSetAbstractionAction struct {
	Type        string `json:"type"        msgpack:"type"`
	Abstraction string `json:"abstraction" msgpack:"abstraction"`
}

// SplitOutcomeRequest splits quote tokens into both sides of an outcome
type SplitOutcomeRequest struct {
	Outcome uint64
	Amount  float64
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// MergeOutcomeRequest merges both sides of an outcome back into quote tokens
type MergeOutcomeRequest struct {
	Outcome uint64
	// Amount is the number of each side to merge; zero merges the maximum
	Amount float64
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// MergeQuestionRequest merges one side 0 token of every outcome of a question back into quote tokens
type MergeQuestionRequest struct {
	Question uint64
	// Amount is the number of each outcome's tokens to merge; zero merges the maximum
	Amount float64
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// NegateOutcomeRequest converts one outcome's side 1 tokens into side 0 tokens of every other outcome of its question
type NegateOutcomeRequest struct {
	Question uint64
	Outcome  uint64
	Amount   float64
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// UserOutcomeAction is the userOutcome action payload; exactly one variant is set
type UserOutcomeAction struct {
	Type          string             `json:"type"                    msgpack:"type"`
	SplitOutcome  *SplitOutcomeWire  `json:"splitOutcome,omitempty"  msgpack:"splitOutcome,omitempty"`
	MergeOutcome  *MergeOutcomeWire  `json:"mergeOutcome,omitempty"  msgpack:"mergeOutcome,omitempty"`
	MergeQuestion *MergeQuestionWire `json:"mergeQuestion,omitempty" msgpack:"mergeQuestion,omitempty"`
	NegateOutcome *NegateOutcomeWire `json:"negateOutcome,omitempty" msgpack:"negateOutcome,omitempty"`
}

// SplitOutcomeWire is the splitOutcome variant of a userOutcome action
type SplitOutcomeWire struct {
	Outcome uint64 `json:"outcome" msgpack:"outcome"`
	Amount  string `json:"amount"  msgpack:"amount"`
}

// MergeOutcomeWire is the mergeOutcome variant of a userOutcome action; a nil Amount merges the maximum
type MergeOutcomeWire struct {
	Outcome uint64  `json:"outcome" msgpack:"outcome"`
	Amount  *string `json:"amount"  msgpack:"amount"`
}

// MergeQuestionWire is the mergeQuestion variant of a userOutcome action; a nil Amount merges the maximum
type MergeQuestionWire struct {
	Question uint64  `json:"question" msgpack:"question"`
	Amount   *string `json:"amount"   msgpack:"amount"`
}

// NegateOutcomeWire is the negateOutcome variant of a userOutcome action
type NegateOutcomeWire struct {
	Question uint64 `json:"question" msgpack:"question"`
	Outcome  uint64 `json:"outcome"  msgpack:"outcome"`
	Amount   string `json:"amount"   msgpack:"amount"`
}

// GossipPriorityBidRequest bids in a gossip priority auction for an IP address; the bid is charged from the spot balance
// and burned
type GossipPriorityBidRequest struct {
	// SlotID is 0 or 1
	SlotID uint64
	// IP must exactly match the address the prioritised peer is seen from
	IP netip.Addr
	// MaxGasInWei is HYPE with 8 decimals
	MaxGasInWei uint64
	// ExpiresAfter, when set, makes Hyperliquid reject the action after this time
	ExpiresAfter time.Time
}

// GossipPriorityBidAction is the gossipPriorityBid action payload
type GossipPriorityBidAction struct {
	Type        string `json:"type"   msgpack:"type"`
	SlotID      uint64 `json:"slotId" msgpack:"slotId"`
	IP          string `json:"ip"     msgpack:"ip"`
	MaxGasInWei uint64 `json:"maxGas" msgpack:"maxGas"`
}

// userSignedPayload is the payload of an action signed with the EIP-712 user-signed scheme, which signs the same values
// it sends
type userSignedPayload interface {
	// setEnvelope sets the action's type, its signature chain ID, the Hyperliquid chain it is signed for and its nonce
	setEnvelope(chain string, nonce uint64)
	// signingFields returns the action's EIP-712 primary type and its fields in type order, from hyperliquidChain to the
	// nonce
	signingFields() (primaryType string, fields []eip712Field)
}
