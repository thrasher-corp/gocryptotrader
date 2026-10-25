package cryptodotcom

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// stakingHistoryMaximumLimit is the most requests or rewards a staking history query returns
const stakingHistoryMaximumLimit = 500

var (
	errInvalidStakingConversionRate    = errors.New("expected conversion rate must be greater than zero")
	errInvalidStakingSlippageTolerance = errors.New("slippage tolerance must not be negative")
)

// GetStakingConversionRate returns the conversion rate between a liquid staking token, such as CDCETH, and its staked
// token, the rate ConvertStakedToken expects
func (e *Exchange) GetStakingConversionRate(ctx context.Context, instrumentName string) (*StakingConversionRateResponse, error) {
	if instrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	var resp *StakingConversionRateResponse
	// A public method sent as a POST, which Crypto.com answers unsigned
	if err := e.sendPostRequest(ctx, stakingEPL, "public/staking/get-conversion-rate", &stakingInstrumentParams{InstrumentName: instrumentName}, &resp, nil); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// Stake requests a quantity of a staking instrument be staked on-chain to earn rewards
func (e *Exchange) Stake(ctx context.Context, req *StakeRequest) (*StakeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	if err := validateStakingQuantity(req.InstrumentName, req.Quantity); err != nil {
		return nil, err
	}
	var resp *StakeResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, stakingEPL, "private/staking/stake", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// Unstake requests a quantity of a staking instrument be unstaked, which may wait on the underlying blockchain's
// unbonding period
func (e *Exchange) Unstake(ctx context.Context, req *UnstakeRequest) (*UnstakeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	if err := validateStakingQuantity(req.InstrumentName, req.Quantity); err != nil {
		return nil, err
	}
	var resp *UnstakeResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, stakingEPL, "private/staking/unstake", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetStakingPosition returns the staking positions, of one instrument or of every one
func (e *Exchange) GetStakingPosition(ctx context.Context, req *StakingPositionRequest) ([]StakingPositionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	var resp *dataResponse[StakingPositionResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, stakingEPL, "private/staking/get-staking-position", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetStakingInstruments returns the instruments that can be staked, with their minimum stake amounts and reward rates
func (e *Exchange) GetStakingInstruments(ctx context.Context) ([]StakingInstrumentResponse, error) {
	var resp *dataResponse[StakingInstrumentResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, stakingEPL, "private/staking/get-staking-instruments", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetOpenStakes returns the stake and unstake requests not yet completed or rejected
func (e *Exchange) GetOpenStakes(ctx context.Context, req *StakingHistoryRequest) ([]StakingRequestResponse, error) {
	params, err := newStakingHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *dataResponse[StakingRequestResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, stakingEPL, "private/staking/get-open-stake", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetStakeHistory returns the stake and unstake requests completed or rejected
func (e *Exchange) GetStakeHistory(ctx context.Context, req *StakingHistoryRequest) ([]StakingRequestResponse, error) {
	params, err := newStakingHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *dataResponse[StakingRequestResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, stakingEPL, "private/staking/get-stake-history", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetStakingRewardHistory returns the staking rewards paid
func (e *Exchange) GetStakingRewardHistory(ctx context.Context, req *StakingHistoryRequest) ([]StakingRewardResponse, error) {
	params, err := newStakingHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *dataResponse[StakingRewardResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, stakingEPL, "private/staking/get-reward-history", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// ConvertStakedToken requests a conversion between a staked token and its liquid staking token, such as ETH.staked and
// CDCETH
func (e *Exchange) ConvertStakedToken(ctx context.Context, req *ConvertStakedTokenRequest) (*ConvertStakedTokenResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	if req.FromInstrumentName == "" {
		return nil, fmt.Errorf("%w: from instrument", errInstrumentNameRequired)
	}
	if req.ToInstrumentName == "" {
		return nil, fmt.Errorf("%w: to instrument", errInstrumentNameRequired)
	}
	if req.ExpectedRate <= 0 {
		return nil, errInvalidStakingConversionRate
	}
	if req.FromQuantity <= 0 {
		return nil, errInvalidQuantity
	}
	if req.SlippageToleranceBasisPoints < 0 {
		return nil, errInvalidStakingSlippageTolerance
	}
	params := &convertStakedTokenParams{
		FromInstrumentName:           req.FromInstrumentName,
		ToInstrumentName:             req.ToInstrumentName,
		ExpectedRate:                 req.ExpectedRate,
		FromQuantity:                 req.FromQuantity,
		SlippageToleranceBasisPoints: strconv.FormatFloat(req.SlippageToleranceBasisPoints.Float64(), 'f', -1, 64),
	}
	var resp *ConvertStakedTokenResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, stakingEPL, "private/staking/convert", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOpenStakingConverts returns the convert requests not yet completed or rejected
func (e *Exchange) GetOpenStakingConverts(ctx context.Context, req *StakingConvertHistoryRequest) ([]StakingConvertRequestResponse, error) {
	params, err := newStakingConvertHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *dataResponse[StakingConvertRequestResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, stakingEPL, "private/staking/get-open-convert", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetStakingConvertHistory returns the convert requests completed or rejected
func (e *Exchange) GetStakingConvertHistory(ctx context.Context, req *StakingConvertHistoryRequest) ([]StakingConvertRequestResponse, error) {
	params, err := newStakingConvertHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *dataResponse[StakingConvertRequestResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, stakingEPL, "private/staking/get-convert-history", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// validateStakingQuantity checks a stake or unstake request's instrument and quantity
func validateStakingQuantity(instrumentName string, quantity types.Number) error {
	if instrumentName == "" {
		return errInstrumentNameRequired
	}
	if quantity <= 0 {
		return errInvalidQuantity
	}
	return nil
}

// newStakingHistoryParams checks a staking history query and returns its params
func newStakingHistoryParams(req *StakingHistoryRequest) (*stakingHistoryParams, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	return newStakingWindowParams(req.InstrumentName, req.StartTime, req.EndTime, req.Limit)
}

// newStakingConvertHistoryParams checks a convert history query and returns its params, which name no instrument
func newStakingConvertHistoryParams(req *StakingConvertHistoryRequest) (*stakingHistoryParams, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	return newStakingWindowParams("", req.StartTime, req.EndTime, req.Limit)
}

// newStakingWindowParams checks a staking history query's window and limit and returns its params, sending its times in
// nanoseconds, which Crypto.com recommends for paging
func newStakingWindowParams(instrumentName string, startTime, endTime time.Time, limit uint64) (*stakingHistoryParams, error) {
	if err := validateOptionalTimeWindow(startTime, endTime); err != nil {
		return nil, err
	}
	if limit > stakingHistoryMaximumLimit {
		return nil, fmt.Errorf("%w: limit %d is above %d", errInvalidCount, limit, stakingHistoryMaximumLimit)
	}
	start, err := unixNanoParam(startTime)
	if err != nil {
		return nil, err
	}
	end, err := unixNanoParam(endTime)
	if err != nil {
		return nil, err
	}
	return &stakingHistoryParams{InstrumentName: instrumentName, StartTime: start, EndTime: end, Limit: limit}, nil
}

// validateOptionalTimeWindow checks a history query's time window, either end of which may be left unset for Crypto.com
// to default
func validateOptionalTimeWindow(startTime, endTime time.Time) error {
	if startTime.IsZero() || endTime.IsZero() {
		return nil
	}
	return common.StartEndTimeCheck(startTime, endTime)
}
