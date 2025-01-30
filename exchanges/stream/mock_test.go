package stream

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestNewMockWebsocketConnection(t *testing.T) {
	t.Parallel()
	got := newMockWebsocketConnection()
	require.NotNil(t, got)
	require.Panics(t, func() { got.SendMessageReturnResponse(context.Background(), 0, nil, nil) })
	resp, err := got.SendMessageReturnResponsesWithInspector(context.Background(), 0, nil, nil, 0, nil)
	require.NoError(t, err)
	require.Nil(t, resp)
	singleResp, err := got.SendMessageReturnResponse(request.WithMockResponse(context.Background(), []byte("test")), 0, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, singleResp)
	resp, err = got.SendMessageReturnResponsesWithInspector(request.WithMockResponse(context.Background(), []byte("test")), 0, nil, nil, 0, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
}
