package main

import (
	"fmt"
	"io"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/grpc/status"
)

func writeCancelAllOrdersResponse(w io.Writer, response *gctrpc.CancelAllOrdersResponse, callErr error) error {
	if callErr != nil {
		response = nil
		for _, detail := range status.Convert(callErr).Details() {
			if partial, ok := detail.(*gctrpc.CancelAllOrdersResponse); ok {
				response = partial
				break
			}
		}
		if response == nil {
			return callErr
		}
	}
	if response == nil {
		return common.ErrInvalidResponse
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")
	if err := encoder.Encode(response); err != nil {
		if callErr != nil {
			return fmt.Errorf("%w: writing cancellation results: %w", callErr, err)
		}
		return fmt.Errorf("writing cancellation results: %w", err)
	}
	return callErr
}
