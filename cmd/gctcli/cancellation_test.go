package main

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWriteCancelAllOrdersResponse(t *testing.T) {
	t.Parallel()
	response := &gctrpc.CancelAllOrdersResponse{Orders: []*gctrpc.Orders{{Exchange: "mock", OrderStatus: map[string]string{"1": "Cancelled"}}}, Count: 1}
	errConnection := errors.New("connection failed")
	failed := status.New(codes.Unavailable, "second batch failed")
	partial, err := failed.WithDetails(response)
	require.NoError(t, err, "partial results must attach")
	unrelated, err := failed.WithDetails(&gctrpc.GenericResponse{})
	require.NoError(t, err, "unrelated details must attach")
	for _, tc := range []struct {
		name         string
		response     *gctrpc.CancelAllOrdersResponse
		callErr      error
		wantErr      error
		output       bool
		writeFailure bool
	}{
		{name: "success", response: response, output: true},
		{name: "partial failure", callErr: partial.Err(), wantErr: partial.Err(), output: true},
		{name: "failure", callErr: failed.Err(), wantErr: failed.Err()},
		{name: "unrelated details", callErr: unrelated.Err(), wantErr: unrelated.Err()},
		{name: "missing response", wantErr: common.ErrInvalidResponse},
		{name: "plain error", callErr: errConnection},
		{name: "output failure", response: response, writeFailure: true, wantErr: io.ErrClosedPipe},
		{name: "partial output failure", callErr: partial.Err(), writeFailure: true, wantErr: io.ErrClosedPipe},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			var writer io.Writer = &buf
			if tc.writeFailure {
				r, w := io.Pipe()
				require.NoError(t, r.Close(), "pipe reader must close")
				t.Cleanup(func() { assert.NoError(t, w.Close(), "pipe writer should close") })
				writer = w
			}
			err := writeCancelAllOrdersResponse(writer, tc.response, tc.callErr)
			if tc.writeFailure {
				require.ErrorIs(t, err, io.ErrClosedPipe, "output failure must propagate")
			}
			if tc.callErr != nil {
				require.ErrorIs(t, err, tc.callErr, "original failure must remain available")
			} else {
				require.ErrorIs(t, err, tc.wantErr, "expected error must be returned")
			}
			if !tc.output {
				assert.Empty(t, buf.String(), "failure without results should not print a response")
				return
			}
			var got gctrpc.CancelAllOrdersResponse
			require.NoError(t, json.Unmarshal(buf.Bytes(), &got), "output must be valid JSON")
			assert.Equal(t, response.Count, got.Count, "retained result count should be printed")
			assert.Equal(t, response.Orders[0].OrderStatus, got.Orders[0].OrderStatus, "retained order statuses should be printed")
		})
	}
}
