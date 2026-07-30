package htx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWSHandleUSDTMarginedPrivateMessage(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		channel  string
		expected any
	}{
		{name: "isolated orders", channel: subscription.MyOrdersChannel, expected: &SwapWsSubOrderData{}},
		{name: "cross orders", channel: wsCrossOrdersChannel, expected: &SwapWsSubOrderData{}},
		{name: "isolated matches", channel: subscription.MyTradesChannel, expected: &SwapWsSubMatchOrderData{}},
		{name: "cross matches", channel: wsCrossTradesChannel, expected: &SwapWsSubMatchOrderData{}},
		{name: "isolated accounts", channel: subscription.MyAccountChannel, expected: &SwapWsSubEquityData{}},
		{name: "cross accounts", channel: wsCrossAccountsChannel, expected: &SwapWsSubEquityData{}},
		{name: "isolated positions", channel: wsPositionsChannel, expected: &SwapWsSubPositionUpdates{}},
		{name: "cross positions", channel: wsCrossPositionsChannel, expected: &SwapWsSubPositionUpdates{}},
		{name: "isolated triggers", channel: wsTriggerOrdersChannel, expected: &SwapWsSubTriggerOrderUpdates{}},
		{name: "cross triggers", channel: wsCrossTriggersChannel, expected: &SwapWsSubTriggerOrderUpdates{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			sub := &subscription.Subscription{Asset: asset.USDTMarginedFutures, Channel: tt.channel, Authenticated: true}
			raw := []byte(`{"op":"notify","topic":"private.*","ts":1603878749908,"symbol":"BTC","contract_code":"BTC-USDT","data":[]}`)
			require.NoError(t, h.wsHandleUSDTMarginedPrivateMessage(t.Context(), sub, raw), "private USDT-margined notification must be decoded")
			message := <-h.Websocket.DataHandler.C
			assert.IsType(t, tt.expected, message.Data, "notification should use its dedicated response type")
		})
	}

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	err := h.wsHandleUSDTMarginedPrivateMessage(t.Context(), &subscription.Subscription{Channel: "unsupported"}, []byte(`{}`))
	require.ErrorIs(t, err, common.ErrNotYetImplemented, "unknown private USDT-margined channels must be rejected")
	err = h.wsHandleUSDTMarginedPrivateMessage(t.Context(), &subscription.Subscription{Channel: subscription.MyOrdersChannel}, []byte(`{`))
	require.Error(t, err, "malformed private USDT-margined notifications must be rejected")
	err = h.wsHandleUSDTMarginedPrivateMessage(t.Context(), nil, []byte(`{}`))
	require.ErrorIs(t, err, common.ErrNilPointer, "nil private USDT-margined subscriptions must be rejected")

	h = new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	raw := []byte(`{
		"op":"notify",
		"topic":"accounts_cross",
		"ts":1640756528985,
		"event":"snapshot",
		"data":[{
			"margin_mode":"cross",
			"margin_account":"USDT",
			"margin_balance":20.6034,
			"contract_detail":[{
				"contract_code":"BTC-USDT",
				"pair":"BTC-USDT",
				"business_type":"swap"
			}]
		}]
	}`)
	sub := &subscription.Subscription{Asset: asset.USDTMarginedFutures, Channel: wsCrossAccountsChannel, Authenticated: true}
	require.NoError(t, h.wsHandleUSDTMarginedPrivateMessage(t.Context(), sub, raw), "cross-account notification must be decoded")
	message := <-h.Websocket.DataHandler.C
	accountUpdate, ok := message.Data.(*SwapWsSubEquityData)
	require.True(t, ok, "cross-account notification must use the account response type")
	require.Len(t, accountUpdate.Data, 1, "cross-account notification must contain its account data")
	assert.Equal(t, "USDT", accountUpdate.Data[0].MarginAccount, "margin account should be retained")
	require.Len(t, accountUpdate.Data[0].ContractDetail, 1, "cross-account notification must contain contract detail")
	assert.Equal(t, "BTC-USDT", accountUpdate.Data[0].ContractDetail[0].ContractCode, "contract code should be retained")
}
