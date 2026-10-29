package coinbase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errDerivativesClientNameRequired    = errors.New("client name required")
	errDerivativesClientVersionRequired = errors.New("client version required")
	errDerivativesInvalidExpectedResult = errors.New("invalid expected result, want exception")
)

// GetTime returns the gateway's current time, which shows the clock skew between the client and the gateway
func (g GlobalDerivativesAPI) GetTime(ctx context.Context) (time.Time, error) {
	return g.getTime(ctx, false)
}

func (g GlobalDerivativesAPI) getTime(ctx context.Context, overWebsocket bool) (time.Time, error) {
	var resp types.Time
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_time", nil, &resp); err != nil {
		return time.Time{}, err
	}
	if resp.Time().IsZero() {
		return time.Time{}, common.ErrNoResponse
	}
	return resp.Time(), nil
}

// GetStatus returns whether the platform is locked in every currency, some or none, and the locked indices
func (g GlobalDerivativesAPI) GetStatus(ctx context.Context) (*DerivativesStatusResponse, error) {
	return g.getStatus(ctx, false)
}

func (g GlobalDerivativesAPI) getStatus(ctx context.Context, overWebsocket bool) (*DerivativesStatusResponse, error) {
	var resp *DerivativesStatusResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/status", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// TestConnection checks the gateway is reachable and returns its API version; an expectedResult of
// DerivativesTestExpectedResultException makes the gateway return an error instead, to exercise a client's error
// handling
func (g GlobalDerivativesAPI) TestConnection(ctx context.Context, expectedResult string) (*DerivativesTestConnectionResponse, error) {
	return g.testConnection(ctx, false, expectedResult)
}

func (g GlobalDerivativesAPI) testConnection(ctx context.Context, overWebsocket bool, expectedResult string) (*DerivativesTestConnectionResponse, error) {
	if expectedResult != "" && expectedResult != DerivativesTestExpectedResultException {
		return nil, fmt.Errorf("%w: %q", errDerivativesInvalidExpectedResult, expectedResult)
	}
	// A typed nil pointer would be sent as null params, so the params are only set for an expected result
	var params any
	if expectedResult != "" {
		params = &DerivativesTestWireRequest{ExpectedResult: expectedResult}
	}
	var resp *DerivativesTestConnectionResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/test", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}
