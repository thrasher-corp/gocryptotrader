package coinbase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebsocketAPISubmitTransferBetweenSubaccounts(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("WebsocketAPI.SubmitTransferBetweenSubaccounts needs subaccounts, which live tests do not have")
	}
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().SubmitTransferBetweenSubaccounts(t.Context(), testDerivativesTransferRequest())
	require.NoError(t, err, "WebsocketAPI.SubmitTransferBetweenSubaccounts must not error")
	assert.Equal(t, testDerivativesTransfer(), resp, "WebsocketAPI.SubmitTransferBetweenSubaccounts should decode every field")

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().SubmitTransferBetweenSubaccounts(t.Context(), testDerivativesTransferRequest())
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.SubmitTransferBetweenSubaccounts should refuse an unauthenticated connection")
}
