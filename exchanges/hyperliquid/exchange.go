package hyperliquid

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// outcomeAssetIDBase offsets outcome encodings into the asset ID space
	outcomeAssetIDBase                             = 1e8
	maximumPriorityRate                            = 1e8
	maximumPerpetualBuilderFee TenthsOfABasisPoint = 100
	maximumSpotBuilderFee      TenthsOfABasisPoint = 1000
	minimumScheduleCancelDelay                     = 5 * time.Second
	minimumTWAPMinutes                             = 5
	maximumTWAPMinutes                             = 1440
	// liquidatorTransferStep is 1000 quote tokens with 6 decimals, the unit of HIP-3 backstop liquidator transfers
	liquidatorTransferStep = 1000e6
	maximumAgentNameLength = 16
	maximumAgentValidity   = 180 * 24 * time.Hour
	// maximumTokenWeiDecimals caps the decimals a token transfer amount is formatted with, whatever a token's metadata
	// reports
	maximumTokenWeiDecimals = 18
)

// Nonce field names of user-signed actions; older actions carry their nonce as time
const (
	nonceFieldNonce = "nonce"
	nonceFieldTime  = "time"
)

// userSignedAmountField names the amount field that user-signed transfer actions sign
const userSignedAmountField = "amount"

var (
	errActionStatusCount           = errors.New("unexpected exchange action status count")
	errActivationPriceInvalid      = errors.New("trailing stop activation price must not be negative")
	errAddressEncodingInvalid      = errors.New("address encoding must be hex or base58")
	errAgentNameInvalid            = errors.New("agent name must be at most 16 characters, and is required for an expiring agent")
	errAgentValidUntilInvalid      = errors.New("agent expiry must be in the future and at most 180 days ahead")
	errBuilderFeeInvalid           = errors.New("builder fee exceeds its maximum of 100 tenths of a basis point on perpetuals or 1000 on spot")
	errClientOrderIDInvalid        = errors.New("client order ID must be a 16-byte hexadecimal value")
	errDestinationRecipientInvalid = errors.New("invalid destination recipient")
	errExpiresAfterPassed          = errors.New("action expiry must be in the future")
	errGossipIPInvalid             = errors.New("invalid gossip priority IP address")
	errGossipSlotInvalid           = errors.New("gossip priority slot must be 0 or 1")
	errInvalidGrouping             = errors.New("invalid order grouping")
	errInvalidLeverage             = errors.New("leverage must be a positive whole number within the market maximum")
	errInvalidTriggerKind          = errors.New("trigger kind must be tp or sl")
	errLiquidatorTransferInvalid   = errors.New("liquidator transfer must be a positive multiple of 1000 quote tokens")
	errMarginChangeInvalid         = errors.New("isolated margin change must not be zero")
	errMaxGasInvalid               = errors.New("maximum gas in wei must be greater than zero")
	errNoActionItems               = errors.New("at least one item is required")
	errNonceRequired               = errors.New("nonce is required")
	errOrderPriceInvalid           = errors.New("order price must be greater than zero")
	errOrderTypeRequired           = errors.New("an order must set exactly one of a limit or a trigger type")
	errOutcomeAmountInvalid        = errors.New("outcome amount must be greater than zero")
	errOwnedSubAccountRequired     = errors.New("action requires a subaccount owned by the configured account")
	errPriorityGroupingInvalid     = errors.New("priority grouping requires ungrouped orders that are all IOC, or all non-reduce-only ALO, on markets other than outcomes")
	errPriorityRateInvalid         = errors.New("priority rate must be at most 100000000")
	errRequestWeightInvalid        = errors.New("reserved request weight must be greater than zero")
	errRetracementInvalid          = errors.New("set exactly one positive retracement percentage or price")
	errScheduleCancelTimeInvalid   = errors.New("scheduled cancel time must be at least 5 seconds ahead")
	errStakeAmountInvalid          = errors.New("stake amount in wei must be greater than zero")
	errSubAccountUnsupported       = errors.New("action supports the main account only, not a configured subaccount or vault")
	errTargetLeverageInvalid       = errors.New("target leverage must be greater than zero")
	errTransferAmountInvalid       = errors.New("transfer amount must be greater than zero")
	errTransferDEXInvalid          = errors.New("invalid transfer DEX")
	errTransferTokenInvalid        = errors.New("invalid transfer token")
	errTriggerPriceRequired        = errors.New("trigger price must be set")
	errTWAPIDRequired              = errors.New("TWAP order ID is required")
	errTWAPMinutesInvalid          = errors.New("TWAP duration must be 5 to 1440 minutes")
)

// PlaceOrders places a batch of orders as one signed order action and returns each order's status, in request order
func (e *Exchange) PlaceOrders(ctx context.Context, arg *PlaceOrdersRequest) ([]OrderActionStatusResponse, error) {
	return e.placeOrders(ctx, arg, false)
}

// placeOrders places a batch of orders through the exchange endpoint, or over the websocket when overWebsocket is set
func (e *Exchange) placeOrders(ctx context.Context, arg *PlaceOrdersRequest, overWebsocket bool) ([]OrderActionStatusResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if len(arg.Orders) == 0 {
		return nil, fmt.Errorf("%w: orders", errNoActionItems)
	}
	var grouping any
	switch arg.Grouping {
	case "", GroupingNone:
		grouping = GroupingNone
	case GroupingNormalTPSL, GroupingPositionTPSL:
		grouping = arg.Grouping
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidGrouping, arg.Grouping)
	}
	if arg.PriorityRate != 0 {
		if err := validatePriorityGrouping(arg); err != nil {
			return nil, err
		}
		grouping = PriorityGroupingWire{PriorityRate: arg.PriorityRate}
	}
	wires := make([]OrderWire, len(arg.Orders))
	for i := range arg.Orders {
		wire, err := arg.Orders[i].toWire(arg.Grouping == GroupingPositionTPSL && arg.Orders[i].Trigger != nil)
		if err != nil {
			return nil, fmt.Errorf("order %d: %w", i, err)
		}
		wires[i] = wire
	}
	action := OrderAction{Type: "order", Orders: wires, Grouping: grouping}
	if arg.Builder != nil {
		builder, err := builderFeeWire(arg.Builder, arg.Orders)
		if err != nil {
			return nil, err
		}
		action.Builder = &builder
	}
	response, err := e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: uint64(len(wires)), ActForVault: true, ExpiresAfter: arg.ExpiresAfter, Websocket: overWebsocket})
	if err != nil {
		return nil, err
	}
	return parseActionStatuses[OrderActionStatusResponse](response, len(wires))
}

// CancelOrders cancels a batch of orders by order ID and returns each cancel's status, in request order
func (e *Exchange) CancelOrders(ctx context.Context, arg *CancelOrdersRequest) ([]CancelActionStatusResponse, error) {
	return e.cancelOrders(ctx, arg, false)
}

// cancelOrders cancels a batch of orders by order ID through the exchange endpoint, or over the websocket when
// overWebsocket is set
func (e *Exchange) cancelOrders(ctx context.Context, arg *CancelOrdersRequest, overWebsocket bool) ([]CancelActionStatusResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if len(arg.Cancels) == 0 {
		return nil, fmt.Errorf("%w: cancels", errNoActionItems)
	}
	for i := range arg.Cancels {
		if arg.Cancels[i].OrderID == 0 {
			return nil, fmt.Errorf("cancel %d: %w", i, order.ErrOrderIDNotSet)
		}
	}
	action := CancelAction{Type: "cancel", Cancels: arg.Cancels, Fast: arg.Fast}
	response, err := e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: uint64(len(arg.Cancels)), ActForVault: true, ExpiresAfter: arg.ExpiresAfter, Websocket: overWebsocket})
	if err != nil {
		return nil, err
	}
	return parseActionStatuses[CancelActionStatusResponse](response, len(arg.Cancels))
}

// CancelOrdersByClientOrderID cancels a batch of orders by client order ID and returns each cancel's status, in request
// order
func (e *Exchange) CancelOrdersByClientOrderID(ctx context.Context, arg *CancelOrdersByClientOrderIDRequest) ([]CancelActionStatusResponse, error) {
	return e.cancelOrdersByClientOrderID(ctx, arg, false)
}

// cancelOrdersByClientOrderID cancels a batch of orders by client order ID through the exchange endpoint, or over the
// websocket when overWebsocket is set
func (e *Exchange) cancelOrdersByClientOrderID(ctx context.Context, arg *CancelOrdersByClientOrderIDRequest, overWebsocket bool) ([]CancelActionStatusResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if len(arg.Cancels) == 0 {
		return nil, fmt.Errorf("%w: cancels", errNoActionItems)
	}
	normalised := make([]CancelByClientOrderIDRequest, len(arg.Cancels))
	for i := range arg.Cancels {
		if err := validateClientOrderID(arg.Cancels[i].ClientOrderID); err != nil {
			return nil, fmt.Errorf("cancel %d: %w", i, err)
		}
		normalised[i] = CancelByClientOrderIDRequest{Asset: arg.Cancels[i].Asset, ClientOrderID: strings.ToLower(arg.Cancels[i].ClientOrderID)}
	}
	action := CancelByClientOrderIDAction{Type: "cancelByCloid", Cancels: normalised, Fast: arg.Fast}
	response, err := e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: uint64(len(normalised)), ActForVault: true, ExpiresAfter: arg.ExpiresAfter, Websocket: overWebsocket})
	if err != nil {
		return nil, err
	}
	return parseActionStatuses[CancelActionStatusResponse](response, len(normalised))
}

// ScheduleCancel schedules a cancellation of every open order, or clears the schedule when the time is zero
// Hyperliquid allows 10 scheduled cancellations a day, reset at 00:00 UTC, once the account has traded enough volume
func (e *Exchange) ScheduleCancel(ctx context.Context, arg *ScheduleCancelRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	action := ScheduleCancelAction{Type: "scheduleCancel"}
	if !arg.Time.IsZero() {
		if arg.Time.Before(time.Now().Add(minimumScheduleCancelDelay)) {
			return fmt.Errorf("%w: %s", errScheduleCancelTimeInvalid, arg.Time)
		}
		action.Time = new(unixMilli(arg.Time))
	}
	_, err := e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ActForVault: true, ExpiresAfter: arg.ExpiresAfter})
	return err
}

// ModifySingleOrder replaces one order with the modify action, which reports no status for the replacement; ModifyOrders
// returns one. A reduce-only trigger replacement without a size covers the whole position. Hyperliquid tries to cancel
// the order before it places the replacement, so an error does not mean the order still rests
func (e *Exchange) ModifySingleOrder(ctx context.Context, arg *ModifyOrderRequest) error {
	return e.modifySingleOrder(ctx, arg, false)
}

// modifySingleOrder replaces one order through the exchange endpoint, or over the websocket when overWebsocket is set
func (e *Exchange) modifySingleOrder(ctx context.Context, arg *ModifyOrderRequest, overWebsocket bool) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	orderID, err := modifyOrderID(arg.OrderID, arg.ClientOrderID)
	if err != nil {
		return err
	}
	wire, err := arg.Order.toWire(arg.Order.Trigger != nil && arg.Order.ReduceOnly)
	if err != nil {
		return err
	}
	action := ModifyAction{Type: "modify", OrderID: orderID, Order: wire, AlwaysPlace: arg.AlwaysPlace}
	_, err = e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ActForVault: true, ExpiresAfter: arg.ExpiresAfter, Websocket: overWebsocket})
	return err
}

// ModifyOrders replaces a batch of orders and returns each replacement's status, in request order. A reduce-only trigger
// replacement without a size covers the whole position. Hyperliquid tries to cancel an order before it places the
// replacement, so a replacement's error does not mean the order still rests
func (e *Exchange) ModifyOrders(ctx context.Context, arg *ModifyOrdersRequest) ([]OrderActionStatusResponse, error) {
	return e.modifyOrders(ctx, arg, false)
}

// modifyOrders replaces a batch of orders through the exchange endpoint, or over the websocket when overWebsocket is set
func (e *Exchange) modifyOrders(ctx context.Context, arg *ModifyOrdersRequest, overWebsocket bool) ([]OrderActionStatusResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if len(arg.Modifies) == 0 {
		return nil, fmt.Errorf("%w: modifies", errNoActionItems)
	}
	wires := make([]ModifyWire, len(arg.Modifies))
	for i := range arg.Modifies {
		orderID, err := modifyOrderID(arg.Modifies[i].OrderID, arg.Modifies[i].ClientOrderID)
		if err != nil {
			return nil, fmt.Errorf("modify %d: %w", i, err)
		}
		wire, err := arg.Modifies[i].Order.toWire(arg.Modifies[i].Order.Trigger != nil && arg.Modifies[i].Order.ReduceOnly)
		if err != nil {
			return nil, fmt.Errorf("modify %d: %w", i, err)
		}
		wires[i] = ModifyWire{OrderID: orderID, Order: wire}
	}
	action := BatchModifyAction{Type: "batchModify", Modifies: wires, AlwaysPlace: arg.AlwaysPlace}
	response, err := e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: uint64(len(wires)), ActForVault: true, ExpiresAfter: arg.ExpiresAfter, Websocket: overWebsocket})
	if err != nil {
		return nil, err
	}
	return parseActionStatuses[OrderActionStatusResponse](response, len(wires))
}

// UpdateLeverage sets a perpetual market's margin mode and leverage
func (e *Exchange) UpdateLeverage(ctx context.Context, arg *UpdateLeverageRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.Leverage == 0 {
		return errInvalidLeverage
	}
	action := UpdateLeverageAction{Type: "updateLeverage", Asset: arg.Asset, IsCross: arg.IsCross, Leverage: arg.Leverage}
	_, err := e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ActForVault: true, ExpiresAfter: arg.ExpiresAfter})
	return err
}

// UpdateIsolatedMargin adds margin to, or removes it from, an isolated position
func (e *Exchange) UpdateIsolatedMargin(ctx context.Context, arg *UpdateIsolatedMarginRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.SignedNotional == 0 {
		return errMarginChangeInvalid
	}
	action := UpdateIsolatedMarginAction{Type: "updateIsolatedMargin", Asset: arg.Asset, IsBuy: true, SignedNotional: arg.SignedNotional}
	_, err := e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ActForVault: true, ExpiresAfter: arg.ExpiresAfter})
	return err
}

// TopUpIsolatedOnlyMargin sets an isolated-only position's margin to reach a target leverage
func (e *Exchange) TopUpIsolatedOnlyMargin(ctx context.Context, arg *TopUpIsolatedOnlyMarginRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.Leverage <= 0 {
		return errTargetLeverageInvalid
	}
	leverage, err := floatToWire(arg.Leverage)
	if err != nil {
		return fmt.Errorf("%w: %w", errTargetLeverageInvalid, err)
	}
	action := TopUpIsolatedOnlyMarginAction{Type: "topUpIsolatedOnlyMargin", Asset: arg.Asset, Leverage: leverage}
	_, err = e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ActForVault: true, ExpiresAfter: arg.ExpiresAfter})
	return err
}

// SendAsset transfers a token between perpetual DEX balances, spot, users and owned subaccounts, and returns the
// action's nonce
func (e *Exchange) SendAsset(ctx context.Context, arg *SendAssetRequest) (uint64, error) {
	if arg == nil {
		return 0, common.ErrNilPointer
	}
	destination, err := normaliseRecipientAddress(arg.Destination)
	if err != nil {
		return 0, err
	}
	if err := validateTransferAmount(arg.Amount); err != nil {
		return 0, err
	}
	if arg.Token.IsEmpty() {
		return 0, fmt.Errorf("%w: %w", errTransferTokenInvalid, currency.ErrCurrencyCodeEmpty)
	}
	credentials, subAccount, err := e.getUserSigningCredentials(ctx)
	if err != nil {
		return 0, err
	}
	if err := e.validateUserSignedSubAccount(ctx, credentials, subAccount); err != nil {
		return 0, err
	}
	sourceDEX, destinationDEX, token, tokenMetadata, err := e.validateSendAssetRoute(ctx, arg.SourceDEX, arg.DestinationDEX, arg.Token)
	if err != nil {
		return 0, err
	}
	amount, err := formatTokenTransferAmount(arg.Amount, tokenMetadata)
	if err != nil {
		return 0, err
	}
	return e.sendUserSignedAction(ctx, credentials, &SendAssetAction{
		Destination:    destination,
		SourceDEX:      sourceDEX,
		DestinationDEX: destinationDEX,
		Token:          token,
		Amount:         amount,
		FromSubAccount: subAccount,
	})
}

// AgentSendAsset transfers a token as SendAsset does, but as an L1 action that an approved API wallet can sign, and
// returns the action's nonce; the destination must be the account or one of its subaccounts
func (e *Exchange) AgentSendAsset(ctx context.Context, arg *AgentSendAssetRequest) (uint64, error) {
	if arg == nil {
		return 0, common.ErrNilPointer
	}
	destination, err := normaliseRecipientAddress(arg.Destination)
	if err != nil {
		return 0, err
	}
	if err := validateTransferAmount(arg.Amount); err != nil {
		return 0, err
	}
	if arg.Token.IsEmpty() {
		return 0, fmt.Errorf("%w: %w", errTransferTokenInvalid, currency.ErrCurrencyCodeEmpty)
	}
	// The subaccount and route checks below send info requests, so a passed expiry is rejected first
	if _, err := expiresAfterMilli(arg.ExpiresAfter); err != nil {
		return 0, err
	}
	credentials, subAccount, err := e.getSigningCredentials(ctx)
	if err != nil {
		return 0, err
	}
	if err := e.validateUserSignedSubAccount(ctx, credentials, subAccount); err != nil {
		return 0, err
	}
	sourceDEX, destinationDEX, token, tokenMetadata, err := e.validateSendAssetRoute(ctx, arg.SourceDEX, arg.DestinationDEX, arg.Token)
	if err != nil {
		return 0, err
	}
	amount, err := formatTokenTransferAmount(arg.Amount, tokenMetadata)
	if err != nil {
		return 0, err
	}
	action := &AgentSendAssetAction{
		Type:           "agentSendAsset",
		Destination:    destination,
		SourceDEX:      sourceDEX,
		DestinationDEX: destinationDEX,
		Token:          token,
		Amount:         amount,
		FromSubAccount: subAccount,
	}
	if _, err := e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ExpiresAfter: arg.ExpiresAfter, SetNonce: func(nonce uint64) { action.Nonce = nonce }}); err != nil {
		return 0, err
	}
	return action.Nonce, nil
}

// SendToEVMWithData transfers a token from HyperCore to a HyperEVM contract with a data payload, and returns the action's
// nonce
func (e *Exchange) SendToEVMWithData(ctx context.Context, arg *SendToEVMWithDataRequest) (uint64, error) {
	if arg == nil {
		return 0, common.ErrNilPointer
	}
	if err := validateTransferAmount(arg.Amount); err != nil {
		return 0, err
	}
	if arg.Token.IsEmpty() {
		return 0, fmt.Errorf("%w: %w", errTransferTokenInvalid, currency.ErrCurrencyCodeEmpty)
	}
	recipient := strings.TrimSpace(arg.DestinationRecipient)
	switch arg.AddressEncoding {
	case AddressEncodingHex:
		if len(recipient) <= 2 || !strings.EqualFold(recipient[:2], "0x") {
			return 0, fmt.Errorf("%w: expected 0x-prefixed hexadecimal", errDestinationRecipientInvalid)
		}
		if _, err := hex.DecodeString(recipient[2:]); err != nil {
			return 0, fmt.Errorf("%w: %w", errDestinationRecipientInvalid, err)
		}
		recipient = strings.ToLower(recipient)
	case AddressEncodingBase58:
		if recipient == "" {
			return 0, errDestinationRecipientInvalid
		}
	default:
		return 0, fmt.Errorf("%w: %q", errAddressEncodingInvalid, arg.AddressEncoding)
	}
	credentials, err := e.getAccountSigningCredentials(ctx)
	if err != nil {
		return 0, err
	}
	sourceDEX, err := e.resolveTransferDEX(ctx, arg.SourceDEX)
	if err != nil {
		return 0, err
	}
	tokenMetadata, token, err := e.resolveTransferToken(ctx, arg.Token)
	if err != nil {
		return 0, err
	}
	amount, err := formatTokenTransferAmount(arg.Amount, tokenMetadata)
	if err != nil {
		return 0, err
	}
	return e.sendUserSignedAction(ctx, credentials, &SendToEVMWithDataAction{
		Token:                token,
		Amount:               amount,
		SourceDEX:            sourceDEX,
		DestinationRecipient: recipient,
		AddressEncoding:      arg.AddressEncoding,
		DestinationChainID:   arg.DestinationChainID,
		GasLimit:             arg.GasLimit,
		Data:                 "0x" + hex.EncodeToString(arg.Data),
	})
}

// SendCoreUSDC sends USDC from the default perpetual DEX balance to another address without using the EVM bridge, and
// returns the action's nonce
func (e *Exchange) SendCoreUSDC(ctx context.Context, arg *SendCoreUSDCRequest) (uint64, error) {
	if arg == nil {
		return 0, common.ErrNilPointer
	}
	destination, amountText, err := prepareTransfer(arg.Destination, arg.Amount)
	if err != nil {
		return 0, err
	}
	return e.sendAccountSignedAction(ctx, &USDSendAction{Destination: destination, Amount: amountText})
}

// SendCoreSpot sends a spot token to another address without using the EVM bridge, and returns the action's nonce
func (e *Exchange) SendCoreSpot(ctx context.Context, arg *SendCoreSpotRequest) (uint64, error) {
	if arg == nil {
		return 0, common.ErrNilPointer
	}
	destination, err := normaliseRecipientAddress(arg.Destination)
	if err != nil {
		return 0, err
	}
	if err := validateTransferAmount(arg.Amount); err != nil {
		return 0, err
	}
	if arg.Token.IsEmpty() {
		return 0, fmt.Errorf("%w: %w", errTransferTokenInvalid, currency.ErrCurrencyCodeEmpty)
	}
	credentials, err := e.getAccountSigningCredentials(ctx)
	if err != nil {
		return 0, err
	}
	tokenMetadata, tokenIdentifier, err := e.resolveTransferToken(ctx, arg.Token)
	if err != nil {
		return 0, err
	}
	amountText, err := formatTokenTransferAmount(arg.Amount, tokenMetadata)
	if err != nil {
		return 0, err
	}
	return e.sendUserSignedAction(ctx, credentials, &SpotSendAction{Destination: destination, Token: tokenIdentifier, Amount: amountText})
}

// TransferUSDCBetweenSpotAndPerp transfers USDC between the spot and default perpetual DEX balances of the account or its
// configured subaccount, and returns the action's nonce
func (e *Exchange) TransferUSDCBetweenSpotAndPerp(ctx context.Context, arg *TransferUSDCBetweenSpotAndPerpRequest) (uint64, error) {
	if arg == nil {
		return 0, common.ErrNilPointer
	}
	amountText, err := formatTransferAmount(arg.Amount)
	if err != nil {
		return 0, err
	}
	credentials, subAccount, err := e.getUserSigningCredentials(ctx)
	if err != nil {
		return 0, err
	}
	if err := e.validateUserSignedSubAccount(ctx, credentials, subAccount); err != nil {
		return 0, err
	}
	if subAccount != "" {
		// The action has no subaccount field, so Hyperliquid reads the subaccount from a suffix on the amount
		amountText += " subaccount:" + subAccount
	}
	return e.sendUserSignedAction(ctx, credentials, &USDClassTransferAction{Amount: amountText, ToPerpetual: arg.ToPerpetual})
}

// DepositIntoStaking moves HYPE, in wei of 8 decimals, from spot into staking, and returns the action's nonce
func (e *Exchange) DepositIntoStaking(ctx context.Context, amountInWei uint64) (uint64, error) {
	if amountInWei == 0 {
		return 0, errStakeAmountInvalid
	}
	return e.sendAccountSignedAction(ctx, &CDepositAction{AmountInWei: amountInWei})
}

// WithdrawFromStaking moves HYPE, in wei of 8 decimals, from staking back to spot through a 7 day unstaking queue, and
// returns the action's nonce
func (e *Exchange) WithdrawFromStaking(ctx context.Context, amountInWei uint64) (uint64, error) {
	if amountInWei == 0 {
		return 0, errStakeAmountInvalid
	}
	return e.sendAccountSignedAction(ctx, &CWithdrawAction{AmountInWei: amountInWei})
}

// DelegateStake delegates staked HYPE to, or undelegates it from, a validator, and returns the action's nonce
// Delegations are locked for a day
func (e *Exchange) DelegateStake(ctx context.Context, arg *DelegateStakeRequest) (uint64, error) {
	if arg == nil {
		return 0, common.ErrNilPointer
	}
	validator, err := normaliseRecipientAddress(arg.Validator)
	if err != nil {
		return 0, err
	}
	if arg.AmountInWei == 0 {
		return 0, errStakeAmountInvalid
	}
	return e.sendAccountSignedAction(ctx, &TokenDelegateAction{Validator: validator, AmountInWei: arg.AmountInWei, IsUndelegate: arg.IsUndelegate})
}

// TransferVaultUSDC deposits USDC into, or withdraws it from, a vault
func (e *Exchange) TransferVaultUSDC(ctx context.Context, arg *VaultTransferRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	vault, err := normaliseRecipientAddress(arg.VaultAddress)
	if err != nil {
		return err
	}
	if arg.AmountInMicroUSDC == 0 {
		return errTransferAmountInvalid
	}
	action := VaultTransferAction{Type: "vaultTransfer", VaultAddress: vault, IsDeposit: arg.IsDeposit, AmountInMicroUSDC: arg.AmountInMicroUSDC}
	_, err = e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ExpiresAfter: arg.ExpiresAfter})
	return err
}

// TransferHIP3BackstopLiquidator deposits into, or withdraws from, a HIP-3 DEX's backstop liquidator
func (e *Exchange) TransferHIP3BackstopLiquidator(ctx context.Context, arg *HIP3LiquidatorTransferRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.NotionalInMicroQuoteTokens == 0 || arg.NotionalInMicroQuoteTokens%liquidatorTransferStep != 0 {
		return fmt.Errorf("%w: got %d", errLiquidatorTransferInvalid, arg.NotionalInMicroQuoteTokens)
	}
	if dex := strings.TrimSpace(arg.DEX); dex == "" || strings.EqualFold(dex, "spot") {
		return fmt.Errorf("%w: %q is not a HIP-3 DEX", errTransferDEXInvalid, arg.DEX)
	}
	// Resolving the DEX sends an info request, so a passed expiry is rejected first
	if _, err := expiresAfterMilli(arg.ExpiresAfter); err != nil {
		return err
	}
	dex, err := e.resolveTransferDEX(ctx, arg.DEX)
	if err != nil {
		return err
	}
	action := HIP3LiquidatorTransferAction{Type: "hip3LiquidatorTransfer", DEX: dex, NotionalInMicroQuoteTokens: arg.NotionalInMicroQuoteTokens, IsDeposit: arg.IsDeposit}
	_, err = e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ExpiresAfter: arg.ExpiresAfter})
	return err
}

// ApproveAgent approves an API wallet to trade for the account, and returns the action's nonce
func (e *Exchange) ApproveAgent(ctx context.Context, arg *ApproveAgentRequest) (uint64, error) {
	if arg == nil {
		return 0, common.ErrNilPointer
	}
	agent, err := normaliseRecipientAddress(arg.AgentAddress)
	if err != nil {
		return 0, err
	}
	name := arg.AgentName
	if utf8.RuneCountInString(name) > maximumAgentNameLength {
		return 0, fmt.Errorf("%w: %q", errAgentNameInvalid, name)
	}
	if !arg.ValidUntil.IsZero() {
		if name == "" {
			return 0, errAgentNameInvalid
		}
		now := time.Now()
		if !arg.ValidUntil.After(now) || arg.ValidUntil.After(now.Add(maximumAgentValidity)) {
			return 0, fmt.Errorf("%w: %s", errAgentValidUntilInvalid, arg.ValidUntil)
		}
		name += " valid_until " + strconv.FormatUint(unixMilli(arg.ValidUntil), 10)
	}
	return e.sendAccountSignedAction(ctx, &ApproveAgentAction{AgentAddress: agent, AgentName: name})
}

// ApproveBuilderFee approves the maximum fee a builder may charge the account, and returns the action's nonce
func (e *Exchange) ApproveBuilderFee(ctx context.Context, arg *ApproveBuilderFeeRequest) (uint64, error) {
	if arg == nil {
		return 0, common.ErrNilPointer
	}
	builder, err := normaliseRecipientAddress(arg.Builder)
	if err != nil {
		return 0, err
	}
	return e.sendAccountSignedAction(ctx, &ApproveBuilderFeeAction{MaxFeeRate: arg.MaxFee.percentageText() + "%", Builder: builder})
}

// PlaceTWAPOrder places a time-weighted average price order and returns its running status
func (e *Exchange) PlaceTWAPOrder(ctx context.Context, arg *TWAPOrderRequest) (*TWAPOrderStatusResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Minutes < minimumTWAPMinutes || arg.Minutes > maximumTWAPMinutes {
		return nil, fmt.Errorf("%w: got %d", errTWAPMinutesInvalid, arg.Minutes)
	}
	if arg.Size <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	size, err := floatToWire(arg.Size)
	if err != nil {
		return nil, err
	}
	action := TWAPOrderAction{
		Type: "twapOrder",
		TWAP: TWAPWire{Asset: arg.Asset, IsBuy: arg.IsBuy, Size: size, ReduceOnly: arg.ReduceOnly, Minutes: arg.Minutes, Randomise: arg.Randomise},
	}
	response, err := e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ActForVault: true, ExpiresAfter: arg.ExpiresAfter})
	if err != nil {
		return nil, err
	}
	status, err := parseActionStatus[TWAPOrderStatusResponse](response)
	if err != nil {
		return nil, err
	}
	if status.Error != "" {
		return nil, fmt.Errorf("%w: %s", errActionResponse, status.Error)
	}
	return status, nil
}

// CancelTWAPOrder cancels a running TWAP order
func (e *Exchange) CancelTWAPOrder(ctx context.Context, arg *TWAPCancelRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.TWAPID == 0 {
		return errTWAPIDRequired
	}
	action := TWAPCancelAction{Type: "twapCancel", Asset: arg.Asset, TWAPID: arg.TWAPID}
	response, err := e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ActForVault: true, ExpiresAfter: arg.ExpiresAfter})
	if err != nil {
		return err
	}
	status, err := parseActionStatus[CancelActionStatusResponse](response)
	if err != nil {
		return err
	}
	if status.Error != "" {
		return fmt.Errorf("%w: %s", errActionResponse, status.Error)
	}
	return nil
}

// PlaceTrailingStopOrder places a trailing stop order, whose trigger follows the mark price, and returns its order ID
func (e *Exchange) PlaceTrailingStopOrder(ctx context.Context, arg *TrailingStopRequest) (uint64, error) {
	if arg == nil {
		return 0, common.ErrNilPointer
	}
	if arg.Size <= 0 {
		return 0, order.ErrAmountIsInvalid
	}
	size, err := floatToWire(arg.Size)
	if err != nil {
		return 0, err
	}
	var retracement RetracementWire
	switch {
	case arg.RetracementPercent > 0 && arg.RetracementPrice == 0:
		percent, err := floatToWire(arg.RetracementPercent)
		if err != nil {
			return 0, fmt.Errorf("%w: %w", errRetracementInvalid, err)
		}
		retracement.Percent = percent + "%"
	case arg.RetracementPrice > 0 && arg.RetracementPercent == 0:
		if retracement.Price, err = floatToWire(arg.RetracementPrice); err != nil {
			return 0, fmt.Errorf("%w: %w", errRetracementInvalid, err)
		}
	default:
		return 0, errRetracementInvalid
	}
	action := TrailingStopAction{Type: "trailingStop", Asset: arg.Asset, IsBuy: arg.IsBuy, Size: size, ReduceOnly: arg.ReduceOnly, Retracement: retracement}
	switch {
	case arg.ActivationPrice < 0:
		return 0, errActivationPriceInvalid
	case arg.ActivationPrice != 0:
		activation, err := floatToWire(arg.ActivationPrice)
		if err != nil {
			return 0, fmt.Errorf("%w: %w", errActivationPriceInvalid, err)
		}
		action.ActivationPrice = &activation
	}
	response, err := e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ActForVault: true, ExpiresAfter: arg.ExpiresAfter})
	if err != nil {
		return 0, err
	}
	var resp *TrailingStopResponse
	if err := json.Unmarshal(response, &resp); err != nil {
		return 0, fmt.Errorf("decoding trailing stop response: %w", err)
	}
	if resp == nil || resp.Data.OrderID == 0 {
		return 0, fmt.Errorf("%w: trailing stop order ID is missing", errActionStatusMalformed)
	}
	return resp.Data.OrderID, nil
}

// ReserveRequestWeight buys additional address-based action capacity for the account or another existing user
func (e *Exchange) ReserveRequestWeight(ctx context.Context, arg *ReserveRequestWeightRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.Weight == 0 {
		return errRequestWeightInvalid
	}
	action := ReserveRequestWeightAction{Type: "reserveRequestWeight", Weight: arg.Weight}
	if strings.TrimSpace(arg.Destination) != "" {
		var err error
		if action.Destination, err = normaliseRecipientAddress(arg.Destination); err != nil {
			return err
		}
	}
	_, err := e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ExpiresAfter: arg.ExpiresAfter})
	return err
}

// InvalidatePendingNonce signs a noop with the nonce of an action that has not yet been processed, so that action is
// rejected
func (e *Exchange) InvalidatePendingNonce(ctx context.Context, arg *InvalidatePendingNonceRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.Nonce == 0 {
		return errNonceRequired
	}
	_, err := e.SendSignedAction(ctx, EmptyAction{Type: "noop"}, &L1ActionOptionsRequest{BatchLength: 1, Nonce: arg.Nonce, ExpiresAfter: arg.ExpiresAfter})
	return err
}

// SetUserDEXAbstraction enables or disables HIP-3 DEX abstraction for the account or one of its subaccounts, and returns
// the action's nonce; SetUserAbstraction supersedes it
func (e *Exchange) SetUserDEXAbstraction(ctx context.Context, arg *UserDEXAbstractionRequest) (uint64, error) {
	if arg == nil {
		return 0, common.ErrNilPointer
	}
	credentials, subAccount, err := e.getUserSigningCredentials(ctx)
	if err != nil {
		return 0, err
	}
	user, err := e.abstractionUser(ctx, credentials, subAccount, arg.User)
	if err != nil {
		return 0, err
	}
	return e.sendUserSignedAction(ctx, credentials, &UserDEXAbstractionAction{User: user, Enabled: arg.Enabled})
}

// AgentEnableDEXAbstraction enables HIP-3 DEX abstraction for an account that has never set it, as an L1 action that an
// approved API wallet can sign; AgentSetUserAbstraction supersedes it
func (e *Exchange) AgentEnableDEXAbstraction(ctx context.Context) error {
	_, err := e.SendSignedAction(ctx, EmptyAction{Type: "agentEnableDexAbstraction"}, &L1ActionOptionsRequest{BatchLength: 1})
	return err
}

// SetUserAbstraction sets how the account or one of its subaccounts shares balances across spot and perpetual DEXs, and
// returns the action's nonce
func (e *Exchange) SetUserAbstraction(ctx context.Context, arg *SetUserAbstractionRequest) (uint64, error) {
	if arg == nil {
		return 0, common.ErrNilPointer
	}
	switch arg.Abstraction {
	case AccountAbstractionDisabled, AccountAbstractionUnifiedAccount, AccountAbstractionPortfolioMargin:
	default:
		return 0, fmt.Errorf("%w: %q", errAccountAbstractionInvalid, arg.Abstraction)
	}
	credentials, subAccount, err := e.getUserSigningCredentials(ctx)
	if err != nil {
		return 0, err
	}
	user, err := e.abstractionUser(ctx, credentials, subAccount, arg.User)
	if err != nil {
		return 0, err
	}
	return e.sendUserSignedAction(ctx, credentials, &UserSetAbstractionAction{User: user, Abstraction: arg.Abstraction})
}

// AgentSetUserAbstraction sets how the account shares balances across spot and perpetual DEXs, as an L1 action that an
// approved API wallet can sign
func (e *Exchange) AgentSetUserAbstraction(ctx context.Context, abstraction AccountAbstraction) error {
	var code string
	switch abstraction {
	case AccountAbstractionDisabled:
		code = "i"
	case AccountAbstractionUnifiedAccount:
		code = "u"
	case AccountAbstractionPortfolioMargin:
		code = "p"
	default:
		return fmt.Errorf("%w: %q", errAccountAbstractionInvalid, abstraction)
	}
	_, err := e.SendSignedAction(ctx, AgentSetAbstractionAction{Type: "agentSetAbstraction", Abstraction: code}, &L1ActionOptionsRequest{BatchLength: 1})
	return err
}

// SplitOutcome splits quote tokens into both sides of an outcome
func (e *Exchange) SplitOutcome(ctx context.Context, arg *SplitOutcomeRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	amount, err := formatOutcomeAmount(arg.Amount)
	if err != nil {
		return err
	}
	action := UserOutcomeAction{Type: "userOutcome", SplitOutcome: &SplitOutcomeWire{Outcome: arg.Outcome, Amount: amount}}
	_, err = e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ExpiresAfter: arg.ExpiresAfter})
	return err
}

// MergeOutcome merges both sides of an outcome back into quote tokens
func (e *Exchange) MergeOutcome(ctx context.Context, arg *MergeOutcomeRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	amount, err := formatMergeAmount(arg.Amount)
	if err != nil {
		return err
	}
	action := UserOutcomeAction{Type: "userOutcome", MergeOutcome: &MergeOutcomeWire{Outcome: arg.Outcome, Amount: amount}}
	_, err = e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ExpiresAfter: arg.ExpiresAfter})
	return err
}

// MergeQuestion merges a side 0 token of every outcome of a question back into quote tokens
func (e *Exchange) MergeQuestion(ctx context.Context, arg *MergeQuestionRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	amount, err := formatMergeAmount(arg.Amount)
	if err != nil {
		return err
	}
	action := UserOutcomeAction{Type: "userOutcome", MergeQuestion: &MergeQuestionWire{Question: arg.Question, Amount: amount}}
	_, err = e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ExpiresAfter: arg.ExpiresAfter})
	return err
}

// NegateOutcome converts an outcome's side 1 tokens into side 0 tokens of every other outcome of its question
func (e *Exchange) NegateOutcome(ctx context.Context, arg *NegateOutcomeRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	amount, err := formatOutcomeAmount(arg.Amount)
	if err != nil {
		return err
	}
	action := UserOutcomeAction{Type: "userOutcome", NegateOutcome: &NegateOutcomeWire{Question: arg.Question, Outcome: arg.Outcome, Amount: amount}}
	_, err = e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ExpiresAfter: arg.ExpiresAfter})
	return err
}

// ClaimRewards claims the account's accrued referral and builder rewards
func (e *Exchange) ClaimRewards(ctx context.Context) error {
	_, err := e.SendSignedAction(ctx, EmptyAction{Type: "claimRewards"}, &L1ActionOptionsRequest{BatchLength: 1})
	return err
}

// BidGossipPriority bids in a gossip priority auction for an IP address
func (e *Exchange) BidGossipPriority(ctx context.Context, arg *GossipPriorityBidRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.SlotID > 1 {
		return fmt.Errorf("%w: got %d", errGossipSlotInvalid, arg.SlotID)
	}
	if !arg.IP.IsValid() {
		return errGossipIPInvalid
	}
	if arg.MaxGasInWei == 0 {
		return errMaxGasInvalid
	}
	action := GossipPriorityBidAction{Type: "gossipPriorityBid", SlotID: arg.SlotID, IP: arg.IP.Unmap().String(), MaxGasInWei: arg.MaxGasInWei}
	_, err := e.SendSignedAction(ctx, action, &L1ActionOptionsRequest{BatchLength: 1, ExpiresAfter: arg.ExpiresAfter})
	return err
}

// ToWire validates an order and formats its numbers for signing
func (o *OrderRequest) ToWire() (OrderWire, error) {
	return o.toWire(false)
}

// toWire validates an order and formats its numbers for signing; wholePosition allows the zero size with which a position
// TP/SL order covers the whole position, as Hyperliquid's app sends one
func (o *OrderRequest) toWire(wholePosition bool) (OrderWire, error) {
	if o.Size < 0 || o.Size == 0 && !wholePosition {
		return OrderWire{}, order.ErrAmountIsInvalid
	}
	if o.Price <= 0 {
		return OrderWire{}, errOrderPriceInvalid
	}
	size, err := floatToWire(o.Size)
	if err != nil {
		return OrderWire{}, err
	}
	price, err := floatToWire(o.Price)
	if err != nil {
		return OrderWire{}, err
	}
	wire := OrderWire{Asset: o.Asset, IsBuy: o.IsBuy, Price: price, Size: size, ReduceOnly: o.ReduceOnly}
	if o.ClientOrderID != "" {
		if err := validateClientOrderID(o.ClientOrderID); err != nil {
			return OrderWire{}, err
		}
		wire.ClientOrderID = strings.ToLower(o.ClientOrderID)
	}
	switch {
	case o.Limit != nil && o.Trigger == nil:
		switch o.Limit.TimeInForce {
		case TimeInForceALO, TimeInForceIOC, TimeInForceGTC:
		default:
			return OrderWire{}, fmt.Errorf("%w: %q", order.ErrUnsupportedTimeInForce, o.Limit.TimeInForce)
		}
		wire.Type.Limit = &LimitOrderTypeWire{TimeInForce: o.Limit.TimeInForce}
	case o.Trigger != nil && o.Limit == nil:
		switch o.Trigger.TakeProfitStopLoss {
		case TriggerTakeProfit, TriggerStopLoss:
		default:
			return OrderWire{}, fmt.Errorf("%w: %q", errInvalidTriggerKind, o.Trigger.TakeProfitStopLoss)
		}
		if o.Trigger.TriggerPrice <= 0 {
			return OrderWire{}, errTriggerPriceRequired
		}
		triggerPrice, err := floatToWire(o.Trigger.TriggerPrice)
		if err != nil {
			return OrderWire{}, err
		}
		wire.Type.Trigger = &TriggerOrderTypeWire{
			IsMarket:           o.Trigger.IsMarket,
			TriggerPrice:       triggerPrice,
			TakeProfitStopLoss: o.Trigger.TakeProfitStopLoss,
		}
	default:
		return OrderWire{}, errOrderTypeRequired
	}
	return wire, nil
}

// validatePriorityGrouping checks a priority grouping replaces no TP/SL grouping, and that its orders are all IOC, or all
// non-reduce-only ALO, on markets other than outcomes
func validatePriorityGrouping(arg *PlaceOrdersRequest) error {
	if arg.PriorityRate > maximumPriorityRate {
		return fmt.Errorf("%w: got %d", errPriorityRateInvalid, arg.PriorityRate)
	}
	if arg.Grouping != "" && arg.Grouping != GroupingNone {
		return fmt.Errorf("%w: grouping %q", errPriorityGroupingInvalid, arg.Grouping)
	}
	var immediate, addLiquidity bool
	for i := range arg.Orders {
		o := &arg.Orders[i]
		switch {
		case o.Asset >= outcomeAssetIDBase || o.Limit == nil:
			return fmt.Errorf("%w: order %d", errPriorityGroupingInvalid, i)
		case o.Limit.TimeInForce == TimeInForceIOC:
			immediate = true
		case o.Limit.TimeInForce == TimeInForceALO && !o.ReduceOnly:
			addLiquidity = true
		default:
			return fmt.Errorf("%w: order %d", errPriorityGroupingInvalid, i)
		}
	}
	if immediate && addLiquidity {
		return fmt.Errorf("%w: IOC and ALO orders are mixed", errPriorityGroupingInvalid)
	}
	return nil
}

// builderFeeWire validates a builder fee against the maximum of the markets it applies to: 1000 tenths of a basis point
// when every order is on spot, otherwise 100
func builderFeeWire(builder *BuilderFee, orders []OrderRequest) (BuilderFeeWire, error) {
	address, err := normaliseRecipientAddress(builder.Builder)
	if err != nil {
		return BuilderFeeWire{}, err
	}
	maximum := maximumSpotBuilderFee
	if slices.ContainsFunc(orders, func(o OrderRequest) bool { return o.Asset < spotAssetIDBase || o.Asset >= builderPerpetualAssetIDBase }) {
		maximum = maximumPerpetualBuilderFee
	}
	if builder.Fee > maximum {
		return BuilderFeeWire{}, fmt.Errorf("%w: maximum %d, got %d", errBuilderFeeInvalid, maximum, builder.Fee)
	}
	return BuilderFeeWire{Builder: address, Fee: builder.Fee}, nil
}

// modifyOrderID returns a modification's target: its order ID, or its client order ID lower-cased
func modifyOrderID(orderID uint64, clientOrderID string) (any, error) {
	switch {
	case orderID != 0 && clientOrderID != "":
		return nil, errOrderIdentifiersConflict
	case orderID != 0:
		return orderID, nil
	case clientOrderID != "":
		if err := validateClientOrderID(clientOrderID); err != nil {
			return nil, err
		}
		return strings.ToLower(clientOrderID), nil
	default:
		return nil, order.ErrOrderIDNotSet
	}
}

// parseActionStatuses decodes the per-item statuses of a batched action
// Payload validation rejects a whole batch with one status, so a lone status is applied to every item
func parseActionStatuses[T any](response json.RawMessage, expected int) ([]T, error) {
	var resp *ActionStatusesResponse[T]
	if err := json.Unmarshal(response, &resp); err != nil {
		return nil, fmt.Errorf("decoding action statuses: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("%w: missing statuses", errActionStatusMalformed)
	}
	statuses := resp.Data.Statuses
	if len(statuses) == 1 && expected > 1 {
		statuses = slices.Repeat(statuses, expected)
	}
	if len(statuses) != expected {
		return nil, fmt.Errorf("%w: expected %d, got %d", errActionStatusCount, expected, len(statuses))
	}
	return statuses, nil
}

// parseActionStatus decodes the single status of an unbatched action
func parseActionStatus[T any](response json.RawMessage) (*T, error) {
	var resp *ActionStatusResponse[json.RawMessage]
	if err := json.Unmarshal(response, &resp); err != nil {
		return nil, fmt.Errorf("decoding action status response: %w", err)
	}
	if resp == nil || len(resp.Data.Status) == 0 || string(resp.Data.Status) == "null" {
		return nil, fmt.Errorf("%w: missing status", errActionStatusMalformed)
	}
	status := new(T)
	if err := json.Unmarshal(resp.Data.Status, status); err != nil {
		return nil, fmt.Errorf("decoding action status: %w", err)
	}
	return status, nil
}

// validateClientOrderID checks a client order ID is a 0x-prefixed 128-bit hexadecimal value
func validateClientOrderID(clientOrderID string) error {
	if !isHex128(clientOrderID) {
		return errClientOrderIDInvalid
	}
	return nil
}

// formatOutcomeAmount formats a positive outcome token amount
func formatOutcomeAmount(amount float64) (string, error) {
	if amount <= 0 {
		return "", errOutcomeAmountInvalid
	}
	formatted, err := floatToWire(amount)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errOutcomeAmountInvalid, err)
	}
	return formatted, nil
}

// formatMergeAmount formats a merge amount, or returns nil for zero, which merges the maximum
func formatMergeAmount(amount float64) (*string, error) {
	if amount == 0 {
		return nil, nil
	}
	formatted, err := formatOutcomeAmount(amount)
	if err != nil {
		return nil, err
	}
	return &formatted, nil
}

// expiresAfterMilli converts an action's expiry to Unix milliseconds, or returns nil when it is unset
// A passed expiry is rejected before signing, as Hyperliquid charges five times the weight for a stale action
func expiresAfterMilli(expiresAfter time.Time) (*uint64, error) {
	if expiresAfter.IsZero() {
		return nil, nil
	}
	if !expiresAfter.After(time.Now()) {
		return nil, fmt.Errorf("%w: %s", errExpiresAfterPassed, expiresAfter)
	}
	return new(unixMilli(expiresAfter)), nil
}

// unixMilli converts a time after the Unix epoch, as every caller validates, to Unix milliseconds
func unixMilli(t time.Time) uint64 {
	return uint64(t.UnixMilli())
}

// sendUserSignedAction signs a user-signed action with the given credentials, which must already be checked to hold the
// account's own key, and returns the action's nonce
func (e *Exchange) sendUserSignedAction(ctx context.Context, credentials *accounts.Credentials, action userSignedPayload) (uint64, error) {
	validationKey, err := e.validateCachedAuthority(ctx, credentials, false)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, err)
	}
	nonce := e.nextNonce()
	chain := "Testnet"
	if e.isMainnetEnvironment() {
		chain = "Mainnet"
	}
	action.setEnvelope(chain, nonce)
	primaryType, fields := action.signingFields()
	signature, err := signUserSignedAction(credentials.Secret, primaryType, fields)
	if err != nil {
		return 0, err
	}
	if _, err := e.sendExchangeRequest(ctx, exchangeActionEndpointLimit(1), &SignedActionRequest{
		Action:    action,
		Nonce:     nonce,
		Signature: signature,
	}); err != nil {
		if errors.Is(err, errSignerRejected) {
			e.invalidateAuthority(&validationKey)
		}
		return 0, err
	}
	return nonce, nil
}

// userSignedFields lists a user-signed action's EIP-712 fields in type order: the chain, the action's own fields, then
// its nonce
func userSignedFields(chain, nonceField string, nonce uint64, fields ...eip712Field) []eip712Field {
	signing := make([]eip712Field, 0, len(fields)+2)
	signing = append(signing, eip712Field{Name: "hyperliquidChain", Type: eip712TypeString, Value: chain})
	signing = append(signing, fields...)
	return append(signing, eip712Field{Name: nonceField, Type: eip712TypeUint64, Value: nonce})
}

// sendAccountSignedAction signs a user-signed action with the account's own key, for an action that cannot act through
// a configured subaccount, and returns the action's nonce
func (e *Exchange) sendAccountSignedAction(ctx context.Context, action userSignedPayload) (uint64, error) {
	credentials, err := e.getAccountSigningCredentials(ctx)
	if err != nil {
		return 0, err
	}
	return e.sendUserSignedAction(ctx, credentials, action)
}

// getAccountSigningCredentials returns signing credentials for a user-signed action that has no subaccount field, which
// a configured subaccount could not act through
func (e *Exchange) getAccountSigningCredentials(ctx context.Context) (*accounts.Credentials, error) {
	credentials, subAccount, err := e.getUserSigningCredentials(ctx)
	if err != nil {
		return nil, err
	}
	if subAccount != "" {
		return nil, errSubAccountUnsupported
	}
	return credentials, nil
}

// prepareTransfer validates a transfer to a destination address, and returns the normalised destination and the amount
// as the action sends it
func prepareTransfer(destination string, amount float64) (normalisedDestination, amountText string, err error) {
	if normalisedDestination, err = normaliseRecipientAddress(destination); err != nil {
		return "", "", err
	}
	if amountText, err = formatTransferAmount(amount); err != nil {
		return "", "", err
	}
	return normalisedDestination, amountText, nil
}

// abstractionUser returns the address an abstraction action applies to: the requested user, else the configured
// subaccount, else the account; every non-account target must be an owned subaccount, and credentials must come from
// getSigningCredentials, which normalises the account address
func (e *Exchange) abstractionUser(ctx context.Context, credentials *accounts.Credentials, subAccount, user string) (string, error) {
	if strings.TrimSpace(user) == "" {
		if subAccount == "" {
			return credentials.Key, nil
		}
		user = subAccount
	}
	normalised, _, err := normaliseAddress(user)
	if err != nil {
		return "", err
	}
	if normalised != credentials.Key {
		if err := e.validateUserSignedSubAccount(ctx, credentials, normalised); err != nil {
			return "", err
		}
	}
	return normalised, nil
}

// getBridgeChain returns the bridge network for the configured environment
func (e *Exchange) getBridgeChain() string {
	if e.isMainnetEnvironment() {
		return "Arbitrum"
	}
	return "Arbitrum Sepolia"
}

// validateTransferAmount checks a transfer amount is positive, before the requests that resolve its token
func validateTransferAmount(amount float64) error {
	if amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return errTransferAmountInvalid
	}
	return nil
}

// formatTokenTransferAmount formats a transfer amount of a spot token with up to as many decimals as the token holds,
// eight at least, as other signed numbers take; UBTC, UETH and others hold more
func formatTokenTransferAmount(amount float64, token *SpotTokenMetadata) (string, error) {
	if err := validateTransferAmount(amount); err != nil {
		return "", err
	}
	decimals := 8
	if token.WeiDecimals > 8 {
		decimals = int(min(token.WeiDecimals, maximumTokenWeiDecimals))
		// The shortest text that reads back as the amount is the amount as written, so it is sent when it fits the token's
		// decimals; formatting to more decimals than it has would add the float's binary residue as digits never written
		if shortest := strconv.FormatFloat(amount, 'f', -1, 64); decimalPlaces(shortest) <= decimals {
			return shortest, nil
		}
	}
	formatted, err := floatToDecimals(amount, decimals)
	if err != nil {
		return "", fmt.Errorf("%s transfer amount: %w", token.Name, err)
	}
	return formatted, nil
}

// decimalPlaces returns how many digits a number's text has after its decimal point
func decimalPlaces(text string) int {
	if point := strings.IndexByte(text, '.'); point >= 0 {
		return len(text) - point - 1
	}
	return 0
}

func formatTransferAmount(amount float64) (string, error) {
	if amount <= 0 {
		return "", errTransferAmountInvalid
	}
	formatted, err := floatToWire(amount)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errTransferAmountInvalid, err)
	}
	return formatted, nil
}

// validateUserSignedSubAccount checks the account controls a configured subaccount; vaults cannot transfer this way, and
// credentials must come from getSigningCredentials, which normalises the account address
func (e *Exchange) validateUserSignedSubAccount(ctx context.Context, credentials *accounts.Credentials, subAccount string) error {
	if subAccount == "" {
		return nil
	}
	role, err := e.GetUserRole(ctx, subAccount)
	if err != nil {
		return err
	}
	if role.Role != "subAccount" {
		return fmt.Errorf("%w: role is %q", errOwnedSubAccountRequired, role.Role)
	}
	if masterAddress, _, err := normaliseAddress(role.Data.Master); err != nil || masterAddress != credentials.Key {
		return fmt.Errorf("%w: configured account does not control %s", errOwnedSubAccountRequired, subAccount)
	}
	return nil
}

// resolveTransferToken returns a spot token's metadata and the NAME:TOKEN_ID identifier transfers sign it with
func (e *Exchange) resolveTransferToken(ctx context.Context, token currency.Code) (*SpotTokenMetadata, string, error) {
	if token.IsEmpty() {
		return nil, "", fmt.Errorf("%w: %w", errTransferTokenInvalid, currency.ErrCurrencyCodeEmpty)
	}
	metadata, err := e.GetSpotMetadata(ctx)
	if err != nil {
		return nil, "", err
	}
	var found *SpotTokenMetadata
	for i := range metadata.Tokens {
		if !metadata.Tokens[i].Name.Equal(token) {
			continue
		}
		if found != nil {
			return nil, "", fmt.Errorf("%w: %s matches more than one spot token", errTransferTokenInvalid, token)
		}
		found = &metadata.Tokens[i]
	}
	if found == nil {
		return nil, "", fmt.Errorf("%w: %s is not present in spot metadata", errTransferTokenInvalid, token)
	}
	return found, found.Name.String() + ":" + found.TokenID, nil
}

// resolveTransferDEX validates a transfer DEX: empty for the default perpetual DEX, spot, or a registered DEX name
func (e *Exchange) resolveTransferDEX(ctx context.Context, dex string) (string, error) {
	dex = strings.TrimSpace(dex)
	if dex == "" {
		return "", nil
	}
	if strings.EqualFold(dex, "spot") {
		return "spot", nil
	}
	dexes, err := e.GetPerpetualDEXs(ctx)
	if err != nil {
		return "", err
	}
	if slices.ContainsFunc(dexes[1:], func(d *PerpetualDEXResponse) bool { return d != nil && d.Name == dex }) {
		return dex, nil
	}
	return "", fmt.Errorf("%w: %q", errTransferDEXInvalid, dex)
}

// validateSendAssetRoute resolves a send asset route and checks the token is the collateral of each perpetual DEX it
// touches, since only collateral can enter or leave a perpetual DEX
func (e *Exchange) validateSendAssetRoute(ctx context.Context, source, destination string, token currency.Code) (sourceDEX, destinationDEX, tokenIdentifier string, tokenMetadata *SpotTokenMetadata, err error) {
	if sourceDEX, err = e.resolveTransferDEX(ctx, source); err != nil {
		return "", "", "", nil, err
	}
	if destinationDEX, err = e.resolveTransferDEX(ctx, destination); err != nil {
		return "", "", "", nil, err
	}
	if tokenMetadata, tokenIdentifier, err = e.resolveTransferToken(ctx, token); err != nil {
		return "", "", "", nil, err
	}
	if token.Equal(currency.USDC) {
		// USDC is sent by name, as the Python SDK sends it, rather than as NAME:TOKEN_ID
		tokenIdentifier = currency.USDC.String()
	}
	for _, dex := range []string{sourceDEX, destinationDEX} {
		if dex == "spot" {
			continue
		}
		metadata, err := e.GetPerpetualMetadata(ctx, dex)
		if err != nil {
			return "", "", "", nil, err
		}
		if metadata.CollateralToken != tokenMetadata.Index {
			return "", "", "", nil, fmt.Errorf("%w: %s is not collateral for DEX %q", errTransferTokenInvalid, token, dex)
		}
	}
	return sourceDEX, destinationDEX, tokenIdentifier, tokenMetadata, nil
}
