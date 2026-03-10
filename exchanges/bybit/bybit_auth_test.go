package bybit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestGetAuthV5Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		response    *RestResponse
		expectError bool
		errContains string
	}{
		{
			name:        "success",
			response:    &RestResponse{RetCode: 0, RetMsg: "OK"},
			expectError: false,
		},
		{
			name:        "retCode with retMsg",
			response:    &RestResponse{RetCode: 10001, RetMsg: "PARAMS_ERROR"},
			expectError: true,
			errContains: "code: 10001 message: PARAMS_ERROR",
		},
		{
			name: "retCode with retExtInfo list",
			response: &RestResponse{
				RetCode:    10001,
				RetExtInfo: retExtInfo{List: []ErrorMessage{{Code: 170130, Message: "invalid symbol"}}},
			},
			expectError: true,
			errContains: "code: 170130 message: invalid symbol",
		},
		{
			name: "retCode with empty retMsg and zero retExtInfo codes",
			response: &RestResponse{
				RetCode:    10001,
				RetExtInfo: retExtInfo{List: []ErrorMessage{{Code: 0, Message: "OK"}}},
			},
			expectError: true,
			errContains: "code: 10001",
		},
		{
			name:        "retCode with no retMsg and no retExtInfo",
			response:    &RestResponse{RetCode: 10001},
			expectError: true,
			errContains: "code: 10001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := getAuthV5Error(tt.response)
			if !tt.expectError {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.ErrorIs(t, err, request.ErrAuthRequestFailed)
			assert.ErrorContains(t, err, tt.errContains)
		})
	}
}

func TestGetAuthV5ErrorStringRetExtInfo(t *testing.T) {
	t.Parallel()

	var response RestResponse
	err := json.Unmarshal([]byte(`{"retCode":10001,"retMsg":"","result":{},"retExtInfo":"","time":1700000000000}`), &response)
	require.NoError(t, err)

	authErr := getAuthV5Error(&response)
	require.Error(t, authErr)
	assert.ErrorIs(t, authErr, request.ErrAuthRequestFailed)
	assert.ErrorContains(t, authErr, "code: 10001")
}
