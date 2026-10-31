package hyperliquid

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
)

var errDEXRequired = errors.New("perpetual DEX name is required")

// GetPerpetualDEXs returns the perpetual DEX registry; the first entry is the default DEX and is always nil
func (e *Exchange) GetPerpetualDEXs(ctx context.Context) ([]*PerpetualDEXResponse, error) {
	var resp []*PerpetualDEXResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "perpDexs"}, &resp); err != nil {
		return nil, err
	}
	if len(resp) == 0 || resp[0] != nil {
		return nil, fmt.Errorf("%w: perpetual DEX registry must start with the default DEX", errUnexpectedResponseLength)
	}
	return resp, nil
}

// GetPerpetualMetadata returns the markets and margin tables of a perpetual DEX; an empty dex selects the default DEX
func (e *Exchange) GetPerpetualMetadata(ctx context.Context, dex string) (*PerpetualMetadataResponse, error) {
	var resp *PerpetualMetadataResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "meta", DEX: dex}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetPerpetualMetadataAndAssetContexts returns a perpetual DEX's metadata and the current context of each market
func (e *Exchange) GetPerpetualMetadataAndAssetContexts(ctx context.Context, dex string) (*PerpetualMetadataAndAssetContextsResponse, error) {
	var resp *PerpetualMetadataAndAssetContextsResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "metaAndAssetCtxs", DEX: dex}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	if len(resp.Metadata.Universe) != len(resp.AssetContexts) {
		return nil, fmt.Errorf("%w: %d perpetual markets but %d contexts", errUnexpectedResponseLength, len(resp.Metadata.Universe), len(resp.AssetContexts))
	}
	return resp, nil
}

// GetClearinghouseState returns an account's perpetual margin summary and positions on one DEX
// Unified and portfolio margin accounts report their trading balance through GetSpotClearinghouseState instead
func (e *Exchange) GetClearinghouseState(ctx context.Context, user, dex string) (*ClearinghouseStateResponse, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp *ClearinghouseStateResponse
	if err := e.SendHTTPRequest(ctx, infoLightEPL, &InfoRequest{Type: "clearinghouseState", User: user, DEX: dex}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetUserFunding returns up to 500 of an account's funding payments across every perpetual DEX over an inclusive time
// range, oldest first; a zero end time ends the range now
// Payments at one funding instant share a timestamp, so callers paging by the last returned time must drop repeated
// payments, and older history is returned as daily aggregates with NumberOfSamples set
func (e *Exchange) GetUserFunding(ctx context.Context, arg *UserFundingRequest) ([]UserFundingUpdateResponse, error) {
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
	var resp []UserFundingUpdateResponse
	return resp, e.SendHTTPRequest(ctx, infoUserFundingEPL, &InfoRequest{
		Type:      "userFunding",
		User:      user,
		StartTime: startTime,
		EndTime:   endTime,
	}, &resp)
}

// GetUserNonFundingLedgerUpdates returns up to 2000 of an account's non-funding ledger updates, such as deposits,
// transfers and withdrawals, over an inclusive time range; a zero end time ends the range now
func (e *Exchange) GetUserNonFundingLedgerUpdates(ctx context.Context, arg *UserNonFundingLedgerUpdatesRequest) ([]UserLedgerUpdateResponse, error) {
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
	var resp []UserLedgerUpdateResponse
	return resp, e.SendHTTPRequest(ctx, infoUserLedgerEPL, &InfoRequest{
		Type:      "userNonFundingLedgerUpdates",
		User:      user,
		StartTime: startTime,
		EndTime:   endTime,
	}, &resp)
}

// GetFundingHistory returns up to 500 of a perpetual market's hourly funding rates over an inclusive time range; a zero
// end time ends the range now
func (e *Exchange) GetFundingHistory(ctx context.Context, arg *FundingHistoryRequest) ([]FundingRateHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if strings.TrimSpace(arg.Coin) == "" {
		return nil, errCoinRequired
	}
	startTime, endTime, err := timeRangeMilli(arg.StartTime, arg.EndTime)
	if err != nil {
		return nil, err
	}
	var resp []FundingRateHistoryResponse
	return resp, e.SendHTTPRequest(ctx, infoFundingHistoryEPL, &InfoRequest{
		Type:      "fundingHistory",
		Coin:      arg.Coin,
		StartTime: startTime,
		EndTime:   endTime,
	}, &resp)
}

// GetPredictedFundings returns each default perpetual DEX market's predicted funding on Hyperliquid and other venues
func (e *Exchange) GetPredictedFundings(ctx context.Context) ([]PredictedFundingResponse, error) {
	var resp []PredictedFundingResponse
	return resp, e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "predictedFundings"}, &resp)
}

// GetPerpetualsAtOpenInterestCap returns the coins of a perpetual DEX whose open interest is at its cap; an empty dex
// selects the default DEX
func (e *Exchange) GetPerpetualsAtOpenInterestCap(ctx context.Context, dex string) ([]string, error) {
	var resp []string
	return resp, e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "perpsAtOpenInterestCap", DEX: dex}, &resp)
}

// GetPerpetualDeployAuctionStatus returns the gas auction for deploying a builder perpetual DEX or market
func (e *Exchange) GetPerpetualDeployAuctionStatus(ctx context.Context) (*GasAuctionResponse, error) {
	var resp *GasAuctionResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "perpDeployAuctionStatus"}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetActiveAssetData returns an account's leverage and trading capacity for one perpetual market; coin is the market's
// case-sensitive name, such as BTC, kPEPE or xyz:XYZ100, rather than a currency code
func (e *Exchange) GetActiveAssetData(ctx context.Context, user, coin string) (*ActiveAssetDataResponse, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(coin) == "" {
		return nil, errCoinRequired
	}
	var resp *ActiveAssetDataResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "activeAssetData", User: user, Coin: coin}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetPerpetualDEXLimits returns a builder-deployed perpetual DEX's open interest and transfer limits
func (e *Exchange) GetPerpetualDEXLimits(ctx context.Context, dex string) (*PerpetualDEXLimitsResponse, error) {
	if strings.TrimSpace(dex) == "" {
		return nil, errDEXRequired
	}
	var resp *PerpetualDEXLimitsResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "perpDexLimits", DEX: dex}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetPerpetualDEXStatus returns a perpetual DEX's total net deposits; an empty dex selects the default DEX
func (e *Exchange) GetPerpetualDEXStatus(ctx context.Context, dex string) (*PerpetualDEXStatusResponse, error) {
	var resp *PerpetualDEXStatusResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &DEXInfoRequest{Type: "perpDexStatus", DEX: dex}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAllPerpetualMetadata returns the metadata of every perpetual DEX in GetPerpetualDEXs order, starting with the
// default DEX; unlike GetPerpetualMetadataAndAssetContexts it carries no asset contexts
func (e *Exchange) GetAllPerpetualMetadata(ctx context.Context) ([]PerpetualMetadataResponse, error) {
	var resp []PerpetualMetadataResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "allPerpMetas"}, &resp); err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetPerpetualAnnotation returns a builder-deployed perpetual market's display annotation; default DEX markets have
// none, so they return common.ErrNoResponse. coin is the market's case-sensitive name, such as xyz:XYZ100, rather than a
// currency code
func (e *Exchange) GetPerpetualAnnotation(ctx context.Context, coin string) (*PerpetualAnnotationResponse, error) {
	if strings.TrimSpace(coin) == "" {
		return nil, errCoinRequired
	}
	var resp *PerpetualAnnotationResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "perpAnnotation", Coin: coin}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetPerpetualCategories returns the category of every annotated builder-deployed perpetual market
func (e *Exchange) GetPerpetualCategories(ctx context.Context) ([]PerpetualCategoryResponse, error) {
	var resp []PerpetualCategoryResponse
	return resp, e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "perpCategories"}, &resp)
}

// GetPerpetualConciseAnnotations returns the annotation of every annotated builder-deployed perpetual market, without
// descriptions
func (e *Exchange) GetPerpetualConciseAnnotations(ctx context.Context) ([]PerpetualConciseAnnotationResponse, error) {
	var resp []PerpetualConciseAnnotationResponse
	return resp, e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "perpConciseAnnotations"}, &resp)
}

// GetUserStarState returns an account's approval state on every HIP-3* perpetual DEX that has approved it, including
// DEXs that have since removed the approval
func (e *Exchange) GetUserStarState(ctx context.Context, user string) (*UserStarStateResponse, error) {
	user, _, err := normaliseAddress(user)
	if err != nil {
		return nil, err
	}
	var resp *UserStarStateResponse
	if err := e.SendHTTPRequest(ctx, infoStandardEPL, &InfoRequest{Type: "userStarState", User: user}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// timeRangeMilli converts an inclusive time range to Unix milliseconds; a zero end time is omitted, which Hyperliquid
// treats as now
func timeRangeMilli(start, end time.Time) (startTime, endTime uint64, err error) {
	// The start time is required and InfoRequest omits a zero one, so a start at or before the epoch counts as unset
	if start.UnixMilli() <= 0 {
		return 0, 0, common.ErrDateUnset
	}
	if start.After(time.Now()) {
		return 0, 0, common.ErrStartAfterTimeNow
	}
	if end.IsZero() {
		return unixMilli(start), 0, nil
	}
	if start.After(end) {
		return 0, 0, common.ErrStartAfterEnd
	}
	return unixMilli(start), unixMilli(end), nil
}
