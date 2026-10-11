package hyperliquid

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	apiURL              = "https://api.hyperliquid.xyz"
	websocketURL        = "wss://api.hyperliquid.xyz/ws"
	testnetAPIURL       = "https://api.hyperliquid-testnet.xyz"
	testnetWebsocketURL = "wss://api.hyperliquid-testnet.xyz/ws"

	maximumCandleCount            = 5000
	maximumFundingHistoryCount    = 500
	maximumUserLedgerHistoryCount = 2000
	maximumUserFillsCount         = 2000
	maximumUserFundingCount       = 500
)

var (
	errAccountAbstractionInvalid = errors.New("invalid account abstraction mode")
	errActionResponse            = errors.New("exchange action failed")
	errCoinRequired              = errors.New("coin is required")
	errInvalidBookLevelCount     = errors.New("invalid orderbook level side count")
	errInvalidMantissa           = errors.New("mantissa must be 2 or 5 and requires 5 significant figures")
	errInvalidSignificantFigures = errors.New("significant figures must be 2, 3, 4 or 5")
	errOrderIdentifiersConflict  = errors.New("set either an order ID or a client order ID, not both")
	errUnexpectedResponseLength  = errors.New("unexpected response length")
	errActionBatchTooLarge       = errors.New("signed action batch is too large")
	errSignerRejected            = errors.New("exchange rejected the signer")
)

// SendHTTPRequest sends an unauthenticated request to the info endpoint
func (e *Exchange) SendHTTPRequest(ctx context.Context, epl request.EndpointLimit, payload, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return e.SendPayload(ctx, epl, func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodPost,
			Path:                   endpoint + "/info",
			Headers:                map[string]string{"Content-Type": "application/json"},
			Body:                   bytes.NewReader(body),
			Result:                 result,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.UnauthenticatedRequest)
}

// SendSignedAction signs an L1 action with the configured key and returns the response payload of an accepted action
// The action is any L1 action payload, such as OrderAction or CancelAction, and nil options send an unbatched action
// for the account itself
func (e *Exchange) SendSignedAction(ctx context.Context, action any, opts *L1ActionOptions) (json.RawMessage, error) {
	if opts == nil {
		opts = &L1ActionOptions{}
	}
	if opts.BatchLength > maximumActionBatchSize {
		return nil, fmt.Errorf("%w: maximum %d, got %d", errActionBatchTooLarge, maximumActionBatchSize, opts.BatchLength)
	}
	expiresAfter, err := expiresAfterMilli(opts.ExpiresAfter)
	if err != nil {
		return nil, err
	}
	credentials, vaultAddress, err := e.getSigningCredentials(ctx)
	if err != nil {
		return nil, err
	}
	if !opts.ActForVault {
		vaultAddress = ""
	}
	validationKey, err := e.validateCachedAuthority(ctx, credentials, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, err)
	}
	nonce := opts.Nonce
	if nonce == 0 {
		nonce = e.nextNonce()
	}
	if opts.SetNonce != nil {
		opts.SetNonce(nonce)
	}
	signature, err := signL1Action(credentials.Secret, action, vaultAddress, nonce, expiresAfter, e.isMainnetEnvironment())
	if err != nil {
		return nil, err
	}
	payload := &SignedActionRequest{
		Action:       action,
		Nonce:        nonce,
		Signature:    signature,
		VaultAddress: vaultAddress,
		ExpiresAfter: expiresAfter,
	}
	var response json.RawMessage
	if opts.Websocket {
		response, err = e.sendWebsocketAction(ctx, payload)
	} else {
		response, err = e.sendExchangeRequest(ctx, exchangeActionEndpointLimit(opts.BatchLength), payload)
	}
	if err != nil {
		if errors.Is(err, errSignerRejected) {
			e.invalidateAuthority(&validationKey)
		}
		return nil, err
	}
	return response, nil
}

// sendExchangeRequest posts a signed action to the exchange endpoint and unwraps its status envelope
func (e *Exchange) sendExchangeRequest(ctx context.Context, epl request.EndpointLimit, payload *SignedActionRequest) (json.RawMessage, error) {
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var resp *ExchangeActionResponse
	if err := e.SendPayload(ctx, epl, func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodPost,
			Path:                   endpoint + "/exchange",
			Headers:                map[string]string{"Content-Type": "application/json"},
			Body:                   bytes.NewReader(body),
			Result:                 &resp,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return actionResponseData(resp)
}

// actionResponseData returns the data of an accepted action's status envelope, or the reason the action was rejected
// A rejection naming an unknown signer or an account without deposits is what an invalid signature produces, so it
// also wraps errSignerRejected
func actionResponseData(resp *ExchangeActionResponse) (json.RawMessage, error) {
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	if resp.Status == "ok" {
		return resp.Response, nil
	}
	var message string
	if err := json.Unmarshal(resp.Response, &message); err != nil {
		message = strings.TrimSpace(string(resp.Response))
	}
	if message == "" || message == "null" {
		message = resp.Status
	}
	if strings.Contains(message, "User or API Wallet ") || strings.HasPrefix(message, "Must deposit before performing actions") {
		return nil, fmt.Errorf("%w: %w: %s", errActionResponse, errSignerRejected, message)
	}
	return nil, fmt.Errorf("%w: %s", errActionResponse, message)
}

// GetAllMids returns mid prices for every actively traded coin on a perpetual DEX; the first DEX also includes spot coins
func (e *Exchange) GetAllMids(ctx context.Context, dex string) (map[string]types.Number, error) {
	var resp map[string]types.Number
	if err := e.SendHTTPRequest(ctx, infoLightEPL, &InfoRequest{Type: "allMids", DEX: dex}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOpenOrders returns every open order of an account on a perpetual DEX, newest first; the first DEX also includes
// spot orders. It omits the order type, time in force and trigger fields of GetFrontendOpenOrders, which may return only
// 100 of the orders of an account with very many
func (e *Exchange) GetOpenOrders(ctx context.Context, user, dex string) ([]BasicOrder, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp []BasicOrder
	return resp, e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "openOrders", User: user, DEX: dex}, &resp)
}

// GetFrontendOpenOrders returns an account's open orders on a perpetual DEX with frontend fields; the first DEX also
// includes spot orders. It returns every open order of most accounts, but only 100, not necessarily the newest, of an
// account with very many, such as one of HLP's child vaults; GetOpenOrders returns every open order without the
// frontend fields
func (e *Exchange) GetFrontendOpenOrders(ctx context.Context, user, dex string) ([]FrontendOpenOrder, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp []FrontendOpenOrder
	return resp, e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "frontendOpenOrders", User: user, DEX: dex}, &resp)
}

// GetUserFills returns an account's 2000 most recent fills across every perpetual DEX and spot, newest first; with
// aggregateByTime the partial fills of an order matched within one block are combined. TWAP slice fills are only
// returned by GetUserTWAPSliceFills
func (e *Exchange) GetUserFills(ctx context.Context, user string, aggregateByTime bool) ([]Fill, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp []Fill
	return resp, e.SendHTTPRequest(ctx, infoUserFillsEPL, &InfoRequest{Type: "userFills", User: user, AggregateByTime: aggregateByTime}, &resp)
}

// GetUserFillsByTime returns up to 2000 of an account's fills over an inclusive time range, oldest first; only its 10000
// most recent fills are available, and TWAP slice fills are not returned
// Fills of one millisecond can straddle the 2000 fill limit, so callers paging by the last fill's time must de-duplicate
// fills by trade ID
func (e *Exchange) GetUserFillsByTime(ctx context.Context, arg *UserFillsByTimeRequest) ([]Fill, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	user, _, err := normaliseAddress(arg.User)
	if err != nil {
		return nil, err
	}
	startTime, endTime, err := timeRangeMilli(arg.StartTime, arg.EndTime)
	if err != nil {
		return nil, err
	}
	var resp []Fill
	return resp, e.SendHTTPRequest(ctx, infoUserFillsEPL, &InfoRequest{
		Type:            "userFillsByTime",
		User:            user,
		StartTime:       startTime,
		EndTime:         endTime,
		AggregateByTime: arg.AggregateByTime,
	}, &resp)
}

// GetUserRateLimit returns an account's address-based action rate limit and its usage
func (e *Exchange) GetUserRateLimit(ctx context.Context, user string) (*UserRateLimitResponse, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp *UserRateLimitResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "userRateLimit", User: user}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOrderStatus returns an order's latest status by exchange order ID or client order ID
func (e *Exchange) GetOrderStatus(ctx context.Context, arg *OrderStatusRequest) (*OrderStatusResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	user, _, err := normaliseAddress(arg.User)
	if err != nil {
		return nil, err
	}
	body := &InfoRequest{Type: "orderStatus", User: user}
	switch {
	case arg.OrderID != 0 && arg.ClientOrderID != "":
		return nil, errOrderIdentifiersConflict
	case arg.OrderID != 0:
		body.OrderID = arg.OrderID
	case arg.ClientOrderID != "":
		if err := validateClientOrderID(arg.ClientOrderID); err != nil {
			return nil, err
		}
		body.OrderID = strings.ToLower(arg.ClientOrderID)
	default:
		return nil, order.ErrOrderIDNotSet
	}
	var resp *OrderStatusResponse
	if err := e.SendHTTPRequest(ctx, infoLightEPL, body, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetL2Book returns an L2 book snapshot of up to 20 levels per side, optionally aggregated
func (e *Exchange) GetL2Book(ctx context.Context, arg *L2BookRequest) (*L2BookResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if strings.TrimSpace(arg.Coin) == "" {
		return nil, errCoinRequired
	}
	if err := validateBookAggregation(arg.NumberOfSignificantFigures, arg.Mantissa); err != nil {
		return nil, err
	}
	var resp *L2BookResponse
	if err := e.SendHTTPRequest(ctx, infoLightEPL, &InfoRequest{
		Type:                       "l2Book",
		Coin:                       arg.Coin,
		NumberOfSignificantFigures: arg.NumberOfSignificantFigures,
		Mantissa:                   arg.Mantissa,
	}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	if len(resp.Levels) != 2 {
		return nil, fmt.Errorf("%w: expected 2 sides, got %d", errInvalidBookLevelCount, len(resp.Levels))
	}
	return resp, nil
}

// validateBookAggregation checks an L2 book's aggregation: 2 to 5 significant figures, or zero for full precision, and a
// mantissa of 2 or 5 only with 5 significant figures
func validateBookAggregation(significantFigures, mantissa uint64) error {
	switch significantFigures {
	case 0, 2, 3, 4, 5:
	default:
		return errInvalidSignificantFigures
	}
	switch mantissa {
	case 0:
	// The documentation also lists a mantissa of 1, which Hyperliquid rejects with HTTP 500 (verified live 2026-10-10)
	case 2, 5:
		if significantFigures != 5 {
			return errInvalidMantissa
		}
	default:
		return errInvalidMantissa
	}
	return nil
}

// GetCandleSnapshot returns candles over an inclusive time range, or up to now when the end time is zero; only the most
// recent 5000 candles are available
func (e *Exchange) GetCandleSnapshot(ctx context.Context, arg *CandleSnapshotRequest) ([]Candle, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if strings.TrimSpace(arg.Coin) == "" {
		return nil, errCoinRequired
	}
	interval, err := formatInterval(arg.Interval)
	if err != nil {
		return nil, err
	}
	startTime, endTime, err := timeRangeMilli(arg.StartTime, arg.EndTime)
	if err != nil {
		return nil, err
	}
	weightEnd := arg.EndTime
	if weightEnd.IsZero() {
		weightEnd = time.Now()
	}
	var resp []Candle
	return resp, e.SendHTTPRequest(ctx, candleEndpointLimit(kline.TotalCandlesPerInterval(arg.StartTime, weightEnd, arg.Interval)), &InfoRequest{
		Type: "candleSnapshot",
		Request: &CandleSnapshotWire{
			Coin:      arg.Coin,
			Interval:  interval,
			StartTime: startTime,
			EndTime:   endTime,
		},
	}, &resp)
}

// GetMaxBuilderFee returns the maximum fee an account has approved for a builder, in tenths of a basis point; zero means
// the builder is not approved
func (e *Exchange) GetMaxBuilderFee(ctx context.Context, user, builder string) (uint64, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return 0, err
	}
	if builder, _, err = normaliseAddress(builder); err != nil {
		return 0, err
	}
	var resp *uint64
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "maxBuilderFee", User: user, Builder: builder}, &resp); err != nil {
		return 0, err
	}
	if resp == nil {
		return 0, common.ErrNoResponse
	}
	return *resp, nil
}

// GetHistoricalOrders returns an account's 2000 most recent order status changes, newest first; an order is listed once
// per status change
func (e *Exchange) GetHistoricalOrders(ctx context.Context, user string) ([]HistoricalOrder, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp []HistoricalOrder
	return resp, e.SendHTTPRequest(ctx, infoHistoricalOrdersEPL, &InfoRequest{Type: "historicalOrders", User: user}, &resp)
}

// GetUserTWAPSliceFills returns an account's 2000 most recent TWAP slice fills, newest first
func (e *Exchange) GetUserTWAPSliceFills(ctx context.Context, user string) ([]TWAPSliceFill, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp []TWAPSliceFill
	return resp, e.SendHTTPRequest(ctx, infoUserFillsEPL, &InfoRequest{Type: "userTwapSliceFills", User: user}, &resp)
}

// GetSubAccounts returns a master account's subaccounts with their first perpetual DEX and spot states; an account
// without subaccounts returns none
func (e *Exchange) GetSubAccounts(ctx context.Context, user string) ([]SubAccount, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp []SubAccount
	return resp, e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "subAccounts", User: user}, &resp)
}

// GetVaultDetails returns a vault's details; an optional user populates that account's follower state
func (e *Exchange) GetVaultDetails(ctx context.Context, vaultAddress, user string) (*VaultDetailsResponse, error) {
	vaultAddress, _, err := normaliseAddress(vaultAddress)
	if err != nil {
		return nil, err
	}
	if user != "" {
		if user, _, err = normaliseAddress(user); err != nil {
			return nil, err
		}
	}
	var resp *VaultDetailsResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "vaultDetails", VaultAddress: vaultAddress, User: user}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetUserVaultEquities returns an account's equity in each vault it has deposited into
func (e *Exchange) GetUserVaultEquities(ctx context.Context, user string) ([]UserVaultEquity, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp []UserVaultEquity
	return resp, e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "userVaultEquities", User: user}, &resp)
}

// GetUserRole returns whether an address is missing, a user, an agent, a vault or a subaccount
func (e *Exchange) GetUserRole(ctx context.Context, user string) (*UserRoleResponse, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp *UserRoleResponse
	if err := e.SendHTTPRequest(ctx, infoUserRoleEPL, &InfoRequest{Type: "userRole", User: user}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetPortfolio returns an account's account value and PNL history and its volume over each portfolio period
func (e *Exchange) GetPortfolio(ctx context.Context, user string) ([]PortfolioPeriod, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp []PortfolioPeriod
	return resp, e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "portfolio", User: user}, &resp)
}

// GetReferral returns an account's referrer, referral rewards and, once it has a code, the accounts it referred
func (e *Exchange) GetReferral(ctx context.Context, user string) (*ReferralResponse, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp *ReferralResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "referral", User: user}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetUserFees returns an account's effective fee rates, the fee schedule and its recent daily volume
func (e *Exchange) GetUserFees(ctx context.Context, user string) (*UserFeesResponse, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp *UserFeesResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "userFees", User: user}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetDelegations returns an account's staking delegations to each validator
func (e *Exchange) GetDelegations(ctx context.Context, user string) ([]Delegation, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp []Delegation
	return resp, e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "delegations", User: user}, &resp)
}

// GetDelegatorSummary returns an account's delegated, undelegated and pending withdrawal staking balances
func (e *Exchange) GetDelegatorSummary(ctx context.Context, user string) (*DelegatorSummaryResponse, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp *DelegatorSummaryResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "delegatorSummary", User: user}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetDelegatorHistory returns an account's staking deposits, delegations and withdrawals, newest first
func (e *Exchange) GetDelegatorHistory(ctx context.Context, user string) ([]DelegatorUpdate, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp []DelegatorUpdate
	return resp, e.SendHTTPRequest(ctx, infoStakingHistoryEPL, &InfoRequest{Type: "delegatorHistory", User: user}, &resp)
}

// GetDelegatorRewards returns an account's daily staking rewards by source, newest first
func (e *Exchange) GetDelegatorRewards(ctx context.Context, user string) ([]DelegatorReward, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp []DelegatorReward
	return resp, e.SendHTTPRequest(ctx, infoStakingHistoryEPL, &InfoRequest{Type: "delegatorRewards", User: user}, &resp)
}

// GetUserDEXAbstraction returns whether an account has HIP-3 DEX abstraction enabled, or nil when it has never been set;
// GetUserAbstraction supersedes it
func (e *Exchange) GetUserDEXAbstraction(ctx context.Context, user string) (*bool, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp *bool
	return resp, e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "userDexAbstraction", User: user}, &resp)
}

// GetUserAbstraction returns an account's abstraction mode
func (e *Exchange) GetUserAbstraction(ctx context.Context, user string) (AccountAbstraction, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return "", err
	}
	var resp AccountAbstraction
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "userAbstraction", User: user}, &resp); err != nil {
		return "", err
	}
	switch resp {
	case AccountAbstractionDefault, AccountAbstractionDisabled, AccountAbstractionDEXAbstraction, AccountAbstractionUnifiedAccount, AccountAbstractionPortfolioMargin:
		return resp, nil
	default:
		return "", fmt.Errorf("%w: %q", errAccountAbstractionInvalid, resp)
	}
}

// GetBorrowLendUserState returns an account's borrowed and supplied balances per token and its health
func (e *Exchange) GetBorrowLendUserState(ctx context.Context, user string) (*BorrowLendUserStateResponse, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp *BorrowLendUserStateResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "borrowLendUserState", User: user}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetBorrowLendReserveState returns the borrow/lend reserve of a spot token index; a token without a reserve returns an
// error
func (e *Exchange) GetBorrowLendReserveState(ctx context.Context, token uint64) (*BorrowLendReserveStateResponse, error) {
	var resp *BorrowLendReserveStateResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "borrowLendReserveState", Token: &token}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAllBorrowLendReserveStates returns the borrow/lend reserve of every token that has one
func (e *Exchange) GetAllBorrowLendReserveStates(ctx context.Context) ([]BorrowLendReserveStateEntry, error) {
	var resp []BorrowLendReserveStateEntry
	return resp, e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "allBorrowLendReserveStates"}, &resp)
}

// GetApprovedBuilders returns the builders an account has approved a fee for
func (e *Exchange) GetApprovedBuilders(ctx context.Context, user string) ([]string, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp []string
	return resp, e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "approvedBuilders", User: user}, &resp)
}

// GetExchangeStatus returns the exchange's server time and any special statuses it reports
// The info endpoint documentation omits exchangeStatus, but its rate limit table weighs it
func (e *Exchange) GetExchangeStatus(ctx context.Context) (*ExchangeStatusResponse, error) {
	var resp *ExchangeStatusResponse
	if err := e.SendHTTPRequest(ctx, infoLightEPL, &InfoRequest{Type: "exchangeStatus"}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetRecentTradesForCoin returns a coin's most recent public trades
// The info endpoint documentation omits recentTrades, but its rate limit table weighs it
func (e *Exchange) GetRecentTradesForCoin(ctx context.Context, coin string) ([]RecentTrade, error) {
	if strings.TrimSpace(coin) == "" {
		return nil, errCoinRequired
	}
	var resp []RecentTrade
	return resp, e.SendHTTPRequest(ctx, infoRecentTradesEPL, &InfoRequest{Type: "recentTrades", Coin: coin}, &resp)
}

// GetTWAPHistory returns an account's 2000 most recent TWAP order status changes, newest first
// The info endpoint documentation omits twapHistory, but its rate limit table weighs it
func (e *Exchange) GetTWAPHistory(ctx context.Context, user string) ([]TWAPHistoryEntry, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp []TWAPHistoryEntry
	return resp, e.SendHTTPRequest(ctx, infoTWAPHistoryEPL, &InfoRequest{Type: "twapHistory", User: user}, &resp)
}

// GetUserTWAPSliceFillsByTime returns up to 2000 of an account's TWAP slice fills over an inclusive time range, oldest
// first; slice fills of one millisecond can straddle the limit, so callers paging by the last fill's time must
// de-duplicate fills by trade ID
// The info endpoint documentation omits userTwapSliceFillsByTime, but its rate limit table weighs it
func (e *Exchange) GetUserTWAPSliceFillsByTime(ctx context.Context, arg *UserTWAPSliceFillsByTimeRequest) ([]TWAPSliceFill, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	user, _, err := normaliseAddress(arg.User)
	if err != nil {
		return nil, err
	}
	startTime, endTime, err := timeRangeMilli(arg.StartTime, arg.EndTime)
	if err != nil {
		return nil, err
	}
	var resp []TWAPSliceFill
	return resp, e.SendHTTPRequest(ctx, infoUserFillsEPL, &InfoRequest{
		Type:      "userTwapSliceFillsByTime",
		User:      user,
		StartTime: startTime,
		EndTime:   endTime,
	}, &resp)
}

// GetGossipPriorityAuctionStatus returns the gossip priority order the previous auctions set and each slot's current
// auction
func (e *Exchange) GetGossipPriorityAuctionStatus(ctx context.Context) (*GossipPriorityAuctionStatusResponse, error) {
	var resp *GossipPriorityAuctionStatusResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "gossipPriorityAuctionStatus"}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

func formatInterval(interval kline.Interval) (string, error) {
	switch interval {
	case kline.OneMin:
		return "1m", nil
	case kline.ThreeMin:
		return "3m", nil
	case kline.FiveMin:
		return "5m", nil
	case kline.FifteenMin:
		return "15m", nil
	case kline.ThirtyMin:
		return "30m", nil
	case kline.OneHour:
		return "1h", nil
	case kline.TwoHour:
		return "2h", nil
	case kline.FourHour:
		return "4h", nil
	case kline.EightHour:
		return "8h", nil
	case kline.TwelveHour:
		return "12h", nil
	case kline.OneDay:
		return "1d", nil
	case kline.ThreeDay:
		return "3d", nil
	case kline.OneWeek:
		return "1w", nil
	case kline.OneMonth:
		return "1M", nil
	default:
		return "", fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, interval)
	}
}

// parseInterval converts a Hyperliquid candle interval into a supported kline.Interval
func parseInterval(interval string) (kline.Interval, error) {
	parsed, err := kline.ParseInterval(interval)
	if err != nil {
		return 0, err
	}
	if _, err := formatInterval(parsed); err != nil {
		return 0, err
	}
	return parsed, nil
}
