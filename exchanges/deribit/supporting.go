package deribit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// testExpectedResultException makes public/test return an error, to exercise a client's error handling
const testExpectedResultException = "exception"

var (
	errClientNameRequired    = errors.New("client name is required")
	errClientVersionRequired = errors.New("client version is required")
	errInvalidExpectedResult = errors.New("invalid expected result")
)

// GetTime returns Deribit's current time, which shows the clock skew between the client and Deribit
func (e *Exchange) GetTime(ctx context.Context) (time.Time, error) {
	return e.getTime(ctx, false)
}

func (e *Exchange) getTime(ctx context.Context, overWebsocket bool) (time.Time, error) {
	var resp types.Time
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_time", nil, &resp); err != nil {
		return time.Time{}, err
	}
	if resp.Time().IsZero() {
		return time.Time{}, common.ErrNoResponse
	}
	return resp.Time(), nil
}

// GetStatus returns whether the platform is locked in every currency, some or none, and the locked indices
func (e *Exchange) GetStatus(ctx context.Context) (*StatusResponse, error) {
	return e.getStatus(ctx, false)
}

func (e *Exchange) getStatus(ctx context.Context, overWebsocket bool) (*StatusResponse, error) {
	var resp *StatusResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/status", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// TestConnection checks the API is reachable and returns its version; an expectedResult of "exception" makes Deribit
// return an error instead, to exercise a client's error handling
func (e *Exchange) TestConnection(ctx context.Context, expectedResult string) (*TestConnectionResponse, error) {
	return e.testConnection(ctx, expectedResult, false)
}

func (e *Exchange) testConnection(ctx context.Context, expectedResult string, overWebsocket bool) (*TestConnectionResponse, error) {
	if expectedResult != "" && expectedResult != testExpectedResultException {
		return nil, fmt.Errorf("%w: %q", errInvalidExpectedResult, expectedResult)
	}
	// A typed nil pointer would be sent as null params, so only a set parameter is assigned
	var params any
	if expectedResult != "" {
		params = &testConnectionParams{ExpectedResult: expectedResult}
	}
	var resp *TestConnectionResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/test", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}
