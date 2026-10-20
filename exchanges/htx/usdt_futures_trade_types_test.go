package htx

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestV5CancelBatchOrdersRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		request *V5CancelBatchOrdersRequest
		wantErr error
	}{
		{name: "nil", wantErr: common.ErrNilPointer},
		{name: "empty", request: &V5CancelBatchOrdersRequest{}, wantErr: order.ErrOrderIDNotSet},
		{name: "mixed", request: &V5CancelBatchOrdersRequest{OrderIDs: []string{"1"}, ClientOrderIDs: []string{"2"}}, wantErr: errMixedOrderIdentifiers},
		{name: "empty order ID", request: &V5CancelBatchOrdersRequest{OrderIDs: []string{""}}, wantErr: order.ErrOrderIDNotSet},
		{name: "empty client ID", request: &V5CancelBatchOrdersRequest{ClientOrderIDs: []string{""}}, wantErr: order.ErrOrderIDNotSet},
		{name: "order IDs", request: &V5CancelBatchOrdersRequest{OrderIDs: []string{"1"}}},
		{name: "client IDs", request: &V5CancelBatchOrdersRequest{ClientOrderIDs: []string{"2"}}},
		{name: "ten orders", request: &V5CancelBatchOrdersRequest{OrderIDs: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}}},
		{name: "eleven orders", request: &V5CancelBatchOrdersRequest{OrderIDs: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}}, wantErr: errCancelOrderLimitExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.request.Validate()
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "Validate must return the expected validation error")
				return
			}
			require.NoError(t, err, "Validate must accept one identifier kind within the limit")
		})
	}
}
