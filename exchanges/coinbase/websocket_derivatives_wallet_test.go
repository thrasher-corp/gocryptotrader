package coinbase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebsocketAPISubmitTransferBetweenSubaccounts(t *testing.T) {
	t.Parallel()
	t.Run("refuses a transfer of more than the account holds", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the mock records a transfer between the account's subaccounts, which the next case checks")
		}
		skipLiveUnlessCanManipulate(t)
		ex, _ := newDerivativesWebsocketTestExchange(t)
		_, err := ex.WebsocketAPI().SubmitTransferBetweenSubaccounts(t.Context(), derivativesWalletLiveRefusedTransfer())
		requireDerivativesRefusedOnMerits(t, err, "WebsocketAPI.SubmitTransferBetweenSubaccounts", "destination", "amount")
	})
	t.Run("transfers between subaccounts", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("live tests have no subaccounts to transfer between, so the case before runs live instead")
		}
		ex, _ := newDerivativesWebsocketTestExchange(t)
		resp, err := ex.WebsocketAPI().SubmitTransferBetweenSubaccounts(t.Context(), testDerivativesTransferRequest())
		require.NoError(t, err, "WebsocketAPI.SubmitTransferBetweenSubaccounts must not error")
		assert.Equal(t, testDerivativesTransfer(), resp, "WebsocketAPI.SubmitTransferBetweenSubaccounts should decode every field")
	})

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err := unauthenticated.WebsocketAPI().SubmitTransferBetweenSubaccounts(t.Context(), testDerivativesTransferRequest())
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.SubmitTransferBetweenSubaccounts should refuse an unauthenticated connection")
}
