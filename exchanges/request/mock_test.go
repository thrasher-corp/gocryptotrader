package request

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMockResponse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	require.False(t, IsMockResponse(ctx))
	require.Nil(t, GetMockResponse(ctx))
	require.Panics(t, func() { getRESTResponseFromMock(ctx) })
	mockCtx := WithMockResponse(ctx, []byte("test"))
	require.True(t, IsMockResponse(mockCtx))
	require.NotNil(t, GetMockResponse(mockCtx))
	got := getRESTResponseFromMock(mockCtx)
	require.NotNil(t, got)
	require.Equal(t, 200, got.StatusCode)
	hotBod, err := io.ReadAll(got.Body)
	require.NoError(t, err)
	require.Equal(t, []byte("test"), hotBod)
}
