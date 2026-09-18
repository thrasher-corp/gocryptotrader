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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	hyperliquidAPIURL              = "https://api.hyperliquid.xyz"
	hyperliquidWebsocketURL        = "wss://api.hyperliquid.xyz/ws"
	hyperliquidTestnetAPIURL       = "https://api.hyperliquid-testnet.xyz"
	hyperliquidTestnetWebsocketURL = "wss://api.hyperliquid-testnet.xyz/ws"
	infoTypePerpetualDEXs          = "perpDexs"
	infoTypeMetadata               = "meta"
	perpetualQuoteCurrency         = "USDC"
	builderPerpetualAssetIDBase    = 100000
	builderPerpetualDEXAssetStride = 10000
	maximumCandleCount             = 5000
	maximumFundingHistoryCount     = 500
	maximumUserLedgerHistoryCount  = 500
	pairMappingMissCacheDuration   = time.Minute
	perpetualMakerBaseFeeRate      = 0.00015
	perpetualTakerBaseFeeRate      = 0.00045
	spotMakerBaseFeeRate           = 0.0004
	spotTakerBaseFeeRate           = 0.0007
)

var (
	errAssetContextNotFound      = errors.New("asset context not found")
	errAccountAbstractionInvalid = errors.New("invalid account abstraction mode")
	errCoinRequired              = errors.New("coin is required")
	errEndpointEnvironment       = errors.New("endpoint does not match the configured sandbox environment")
	errInvalidBookLevelCount     = errors.New("invalid orderbook level side count")
	errInvalidMantissa           = errors.New("mantissa must be 1, 2, or 5 and requires 5 significant figures")
	errInvalidSignificantFigures = errors.New("significant figures must be 2, 3, 4, or 5")
	errInvalidSpotTokenCount     = errors.New("spot market must reference exactly two tokens")
	errInvalidPerpetualDEX       = errors.New("invalid perpetual DEX registry or metadata")
	errPairMappingNotFound       = errors.New("pair mapping not found")
	errSpotTokenNotFound         = errors.New("spot token metadata not found")
	errUnexpectedResponseLength  = errors.New("unexpected response length")
)

type infoRequest struct {
	Type      string  `json:"type"`
	DEX       string  `json:"dex,omitempty"`
	Coin      string  `json:"coin,omitempty"`
	User      string  `json:"user,omitempty"`
	Vault     string  `json:"vaultAddress,omitempty"`
	OrderID   any     `json:"oid,omitempty"`
	StartTime int64   `json:"startTime,omitempty"`
	EndTime   int64   `json:"endTime,omitempty"`
	NSigFigs  *uint64 `json:"nSigFigs,omitempty"`
	Mantissa  *uint64 `json:"mantissa,omitempty"`
	Request   any     `json:"req,omitempty"`
}

func getRESTEndpoint(a asset.Item) (exchange.URL, error) {
	switch a {
	case asset.Spot:
		return exchange.RestSpot, nil
	case asset.PerpetualContract:
		return exchange.RestFutures, nil
	default:
		return 0, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
}

// SendHTTPRequest sends an unauthenticated request to Hyperliquid's info endpoint.
func (e *Exchange) SendHTTPRequest(ctx context.Context, endpointType exchange.URL, f request.EndpointLimit, payload, result any) error {
	if e == nil || e.Requester == nil || e.API.Endpoints == nil {
		return common.ErrNilPointer
	}
	endpoint, err := e.API.Endpoints.GetURL(endpointType)
	if err != nil {
		return err
	}
	return e.SendPayload(ctx, f, func() (*request.Item, error) {
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
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

// GetPerpetualMetadata returns metadata for the default Hyperliquid perpetual DEX.
func (e *Exchange) GetPerpetualMetadata(ctx context.Context) (*PerpetualMetadata, error) {
	return e.GetPerpetualMetadataForDEX(ctx, "")
}

// GetPerpetualMetadataForDEX returns metadata for one perpetual DEX.
func (e *Exchange) GetPerpetualMetadataForDEX(ctx context.Context, dex string) (*PerpetualMetadata, error) {
	var resp *PerpetualMetadata
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, infoStandardEPL, &infoRequest{Type: infoTypeMetadata, DEX: dex}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNilPointer
	}
	return resp, nil
}

// GetPerpetualDEXs returns the ordered perpetual DEX registry. Index zero is
// the default DEX and is represented by null.
func (e *Exchange) GetPerpetualDEXs(ctx context.Context) ([]*PerpetualDEX, error) {
	var resp []*PerpetualDEX
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, infoStandardEPL, &infoRequest{Type: infoTypePerpetualDEXs}, &resp); err != nil {
		return nil, err
	}
	if len(resp) == 0 || resp[0] != nil {
		return nil, fmt.Errorf("%w: perpetual DEX registry must start with the default DEX", errUnexpectedResponseLength)
	}
	return resp, nil
}

// GetSpotMetadata returns metadata for all Hyperliquid spot markets.
func (e *Exchange) GetSpotMetadata(ctx context.Context) (*SpotMetadata, error) {
	var resp *SpotMetadata
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, infoStandardEPL, &infoRequest{Type: "spotMeta"}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNilPointer
	}
	return resp, nil
}

// GetPerpetualMetadataAndAssetContexts returns perpetual metadata and aligned current market contexts.
func (e *Exchange) GetPerpetualMetadataAndAssetContexts(ctx context.Context) (*PerpetualMetadataAndAssetContexts, error) {
	return e.GetPerpetualMetadataAndAssetContextsForDEX(ctx, "")
}

// GetPerpetualMetadataAndAssetContextsForDEX returns aligned metadata and
// current market contexts for one perpetual DEX.
func (e *Exchange) GetPerpetualMetadataAndAssetContextsForDEX(ctx context.Context, dex string) (*PerpetualMetadataAndAssetContexts, error) {
	var raw []json.RawMessage
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, infoStandardEPL, &infoRequest{Type: "metaAndAssetCtxs", DEX: dex}, &raw); err != nil {
		return nil, err
	}
	if len(raw) != 2 {
		return nil, fmt.Errorf("%w: expected 2 entries, got %d", errUnexpectedResponseLength, len(raw))
	}
	var resp PerpetualMetadataAndAssetContexts
	if err := json.Unmarshal(raw[0], &resp.Metadata); err != nil {
		return nil, fmt.Errorf("error decoding perpetual metadata: %w", err)
	}
	if err := json.Unmarshal(raw[1], &resp.AssetContexts); err != nil {
		return nil, fmt.Errorf("error decoding perpetual asset contexts: %w", err)
	}
	return &resp, nil
}

// GetFundingHistory returns up to 500 hourly funding records for one perpetual
// market over an inclusive time range.
func (e *Exchange) GetFundingHistory(ctx context.Context, coin string, start, end time.Time) ([]FundingRateRecord, error) {
	coin = strings.TrimSpace(coin)
	if coin == "" {
		return nil, errCoinRequired
	}
	if err := common.StartEndTimeCheck(start, end); err != nil {
		return nil, err
	}
	var resp []FundingRateRecord
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, infoFundingHistoryEPL, &infoRequest{
		Type:      "fundingHistory",
		Coin:      coin,
		StartTime: start.UnixMilli(),
		EndTime:   end.UnixMilli(),
	}, &resp)
	return resp, err
}

// GetUserNonFundingLedgerUpdates returns up to 500 non-funding account ledger
// updates over an inclusive time range.
func (e *Exchange) GetUserNonFundingLedgerUpdates(
	ctx context.Context,
	user string,
	start,
	end time.Time,
) ([]UserLedgerUpdate, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	if err := common.StartEndTimeCheck(start, end); err != nil {
		return nil, err
	}
	var resp []UserLedgerUpdate
	err = e.SendHTTPRequest(ctx, exchange.RestSpot, infoUserLedgerEPL, &infoRequest{
		Type:      "userNonFundingLedgerUpdates",
		User:      user,
		StartTime: start.UnixMilli(),
		EndTime:   end.UnixMilli(),
	}, &resp)
	return resp, err
}

// GetUserFees returns the effective maker and taker rates for an address.
func (e *Exchange) GetUserFees(ctx context.Context, user string) (*UserFees, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp *UserFees
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, infoStandardEPL, &infoRequest{Type: "userFees", User: user}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNilPointer
	}
	return resp, nil
}

// GetActiveAssetData returns account-specific leverage and trading limits for a
// perpetual market.
func (e *Exchange) GetActiveAssetData(ctx context.Context, user, coin string) (*ActiveAssetData, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	coin = strings.TrimSpace(coin)
	if coin == "" {
		return nil, errCoinRequired
	}
	var resp *ActiveAssetData
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, infoStandardEPL, &infoRequest{Type: "activeAssetData", User: user, Coin: coin}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNilPointer
	}
	return resp, nil
}

// GetSpotMetadataAndAssetContexts returns spot metadata and current market contexts.
func (e *Exchange) GetSpotMetadataAndAssetContexts(ctx context.Context) (*SpotMetadataAndAssetContexts, error) {
	var raw []json.RawMessage
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, infoStandardEPL, &infoRequest{Type: "spotMetaAndAssetCtxs"}, &raw); err != nil {
		return nil, err
	}
	if len(raw) != 2 {
		return nil, fmt.Errorf("%w: expected 2 entries, got %d", errUnexpectedResponseLength, len(raw))
	}
	var resp SpotMetadataAndAssetContexts
	if err := json.Unmarshal(raw[0], &resp.Metadata); err != nil {
		return nil, fmt.Errorf("error decoding spot metadata: %w", err)
	}
	if err := json.Unmarshal(raw[1], &resp.AssetContexts); err != nil {
		return nil, fmt.Errorf("error decoding spot asset contexts: %w", err)
	}
	return &resp, nil
}

// GetAllMids returns mid prices for all actively traded markets on a perpetual DEX.
func (e *Exchange) GetAllMids(ctx context.Context, dex string) (map[string]types.Number, error) {
	resp := make(map[string]types.Number)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, infoLightEPL, &infoRequest{Type: "allMids", DEX: dex}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNilPointer
	}
	return resp, nil
}

// GetL2Book returns a complete L2 orderbook snapshot with optional price aggregation.
func (e *Exchange) GetL2Book(ctx context.Context, a asset.Item, arg *L2BookRequest) (*L2Book, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if strings.TrimSpace(arg.Coin) == "" {
		return nil, errCoinRequired
	}
	endpoint, err := getRESTEndpoint(a)
	if err != nil {
		return nil, err
	}
	if arg.SignificantFigures != nil {
		switch *arg.SignificantFigures {
		case 2, 3, 4, 5:
		default:
			return nil, errInvalidSignificantFigures
		}
	}
	if arg.Mantissa != nil {
		if arg.SignificantFigures == nil || *arg.SignificantFigures != 5 {
			return nil, errInvalidMantissa
		}
		switch *arg.Mantissa {
		case 1, 2, 5:
		default:
			return nil, errInvalidMantissa
		}
	}
	var resp *L2Book
	err = e.SendHTTPRequest(ctx, endpoint, infoLightEPL, &infoRequest{
		Type:     "l2Book",
		Coin:     arg.Coin,
		NSigFigs: arg.SignificantFigures,
		Mantissa: arg.Mantissa,
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNilPointer
	}
	if len(resp.Levels) != 2 {
		return nil, fmt.Errorf("%w: expected 2 sides, got %d", errInvalidBookLevelCount, len(resp.Levels))
	}
	return resp, nil
}

// GetRecentTradesForCoin returns the most recent public trades for a Hyperliquid market identifier.
func (e *Exchange) GetRecentTradesForCoin(ctx context.Context, coin string, a asset.Item) ([]RecentTrade, error) {
	if strings.TrimSpace(coin) == "" {
		return nil, errCoinRequired
	}
	endpoint, err := getRESTEndpoint(a)
	if err != nil {
		return nil, err
	}
	var resp []RecentTrade
	if err := e.SendHTTPRequest(ctx, endpoint, infoRecentTradesEPL, &infoRequest{Type: "recentTrades", Coin: coin}, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetCandles returns an OHLCV snapshot for a Hyperliquid market.
func (e *Exchange) GetCandles(ctx context.Context, a asset.Item, arg *CandleRequest) ([]Candle, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if strings.TrimSpace(arg.Coin) == "" {
		return nil, errCoinRequired
	}
	endpoint, err := getRESTEndpoint(a)
	if err != nil {
		return nil, err
	}
	interval, err := formatInterval(arg.Interval)
	if err != nil {
		return nil, err
	}
	if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	payload := &infoRequest{
		Type: "candleSnapshot",
		Request: struct {
			Coin      string `json:"coin"`
			Interval  string `json:"interval"`
			StartTime int64  `json:"startTime"`
			EndTime   int64  `json:"endTime"`
		}{
			Coin:      arg.Coin,
			Interval:  interval,
			StartTime: arg.StartTime.UnixMilli(),
			// Hyperliquid treats endTime as inclusive; GCT candle ranges are half-open.
			EndTime: arg.EndTime.Add(-time.Millisecond).UnixMilli(),
		},
	}
	count := kline.TotalCandlesPerInterval(arg.StartTime, arg.EndTime, arg.Interval)
	var resp []Candle
	if err := e.SendHTTPRequest(ctx, endpoint, candleEndpointLimit(count), payload, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetUserRole returns the on-chain role for an address.
func (e *Exchange) GetUserRole(ctx context.Context, user string) (*UserRole, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp *UserRole
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, infoUserRoleEPL, &infoRequest{Type: "userRole", User: user}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNilPointer
	}
	return resp, nil
}

// GetVaultDetails returns ownership details for a vault address.
func (e *Exchange) GetVaultDetails(ctx context.Context, vault, user string) (*VaultDetails, error) {
	vault, _, err := normaliseAddress(vault)
	if err != nil {
		return nil, err
	}
	if user != "" {
		user, _, err = normaliseAddress(user)
		if err != nil {
			return nil, err
		}
	}
	var resp *VaultDetails
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, infoStandardEPL, &infoRequest{Type: "vaultDetails", Vault: vault, User: user}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNilPointer
	}
	return resp, nil
}

// GetSpotClearinghouseState returns spot balances for an address.
func (e *Exchange) GetSpotClearinghouseState(ctx context.Context, user string) (*SpotClearinghouseState, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp *SpotClearinghouseState
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, infoLightEPL, &infoRequest{Type: "spotClearinghouseState", User: user}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNilPointer
	}
	return resp, nil
}

// GetUserAbstraction returns the account abstraction mode for an address.
func (e *Exchange) GetUserAbstraction(ctx context.Context, user string) (AccountAbstraction, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return "", err
	}
	var resp AccountAbstraction
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, infoStandardEPL, &infoRequest{Type: "userAbstraction", User: user}, &resp); err != nil {
		return "", err
	}
	switch resp {
	case AccountAbstractionDefault,
		AccountAbstractionDisabled,
		AccountAbstractionDEX,
		AccountAbstractionUnified,
		AccountAbstractionPortfolio:
		return resp, nil
	default:
		return "", fmt.Errorf("%w: %q", errAccountAbstractionInvalid, resp)
	}
}

// GetClearinghouseState returns default-DEX perpetual account state for an address.
func (e *Exchange) GetClearinghouseState(ctx context.Context, user string) (*ClearinghouseState, error) {
	return e.GetClearinghouseStateForDEX(ctx, user, "")
}

// GetClearinghouseStateForDEX returns perpetual account state for one DEX.
func (e *Exchange) GetClearinghouseStateForDEX(ctx context.Context, user, dex string) (*ClearinghouseState, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp *ClearinghouseState
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, infoLightEPL, &infoRequest{Type: "clearinghouseState", User: user, DEX: dex}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNilPointer
	}
	return resp, nil
}

// GetOpenOrdersForUser returns open spot and default-DEX perpetual orders for an address.
func (e *Exchange) GetOpenOrdersForUser(ctx context.Context, user string) ([]OpenOrder, error) {
	return e.GetOpenOrdersForUserForDEX(ctx, user, "")
}

// GetOpenOrdersForUserForDEX returns open orders for one perpetual DEX. The
// default DEX response also includes spot orders.
func (e *Exchange) GetOpenOrdersForUserForDEX(ctx context.Context, user, dex string) ([]OpenOrder, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp []OpenOrder
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, infoStandardEPL, &infoRequest{Type: "frontendOpenOrders", User: user, DEX: dex}, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetHistoricalOrdersForUser returns up to the latest 2000 orders for an address.
func (e *Exchange) GetHistoricalOrdersForUser(ctx context.Context, user string) ([]HistoricalOrder, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp []HistoricalOrder
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, infoHistoricalOrdersEPL, &infoRequest{Type: "historicalOrders", User: user}, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOrderStatusForUser returns order status by numeric order ID or client order ID.
func (e *Exchange) GetOrderStatusForUser(ctx context.Context, user string, orderID any) (*OrderStatusResponse, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	switch id := orderID.(type) {
	case uint64:
		if id == 0 {
			return nil, order.ErrOrderIDNotSet
		}
	case string:
		if err := validateClientOrderID(id); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w: expected uint64 or client order ID string", order.ErrOrderIDNotSet)
	}
	var resp *OrderStatusResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, infoLightEPL, &infoRequest{Type: "orderStatus", User: user, OrderID: orderID}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNilPointer
	}
	return resp, nil
}
