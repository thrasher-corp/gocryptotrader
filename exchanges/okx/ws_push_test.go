package okx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// TestWSPushSchemasDecode guards the websocket push schemas against the
// documented response fields: price-limit must decode into its own limit
// fields, withdrawal-info must decode into the withdrawal struct, and the
// grid-positions and orders pushes must carry their documented fields.
func TestWSPushSchemasDecode(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")

	testexch.FixtureToDataHandler(t, "testdata/wsPushSchemas.json", func(ctx context.Context, b []byte) error {
		return e.wsHandleData(ctx, nil, b)
	})

	var priceLimits []*WsLimitPrice
	var withdrawals []*struct {
		Arguments SubscriptionInfo   `json:"arg"`
		Data      []WsWithdrawalInfo `json:"data"`
	}
	var deposits []*struct {
		Arguments SubscriptionInfo `json:"arg"`
		Data      []WsDepositInfo  `json:"data"`
	}
	var gridPositions []*WsGridPosition

	for len(e.Websocket.DataHandler.C) > 0 {
		switch v := (<-e.Websocket.DataHandler.C).Data.(type) {
		case *WsLimitPrice:
			priceLimits = append(priceLimits, v)
		case *struct {
			Arguments SubscriptionInfo   `json:"arg"`
			Data      []WsWithdrawalInfo `json:"data"`
		}:
			withdrawals = append(withdrawals, v)
		case *struct {
			Arguments SubscriptionInfo `json:"arg"`
			Data      []WsDepositInfo  `json:"data"`
		}:
			deposits = append(deposits, v)
		case *WsGridPosition:
			gridPositions = append(gridPositions, v)
		}
	}

	t.Run("price limit decodes its own fields", func(t *testing.T) {
		t.Parallel()
		require.Len(t, priceLimits, 1, "the price-limit push must decode into the limit price struct")
		require.Len(t, priceLimits[0].Data, 1, "the price limit row must decode")
		assert.Equal(t, 52000.0, priceLimits[0].Data[0].BuyLimit.Float64(), "the documented buyLmt should decode")
		assert.Equal(t, 38000.0, priceLimits[0].Data[0].SellLimit.Float64(), "the documented sellLmt should decode")
		assert.True(t, priceLimits[0].Data[0].Enabled, "the documented enabled flag should decode")
	})

	t.Run("withdrawal info decodes into the withdrawal struct", func(t *testing.T) {
		t.Parallel()
		require.Len(t, withdrawals, 1, "the withdrawal-info push must decode into the withdrawal struct")
		require.Len(t, withdrawals[0].Data, 1, "the withdrawal row must decode")
		assert.Equal(t, "2", withdrawals[0].Data[0].ToAddressType, "the documented toAddrType should decode")
		assert.Equal(t, "w1", withdrawals[0].Data[0].WithdrawalNote, "the documented note should decode")
		assert.Equal(t, "W12345", withdrawals[0].Data[0].WithdrawalID, "the withdrawal id should decode")
	})

	t.Run("deposit info decodes into the deposit struct", func(t *testing.T) {
		t.Parallel()
		require.Len(t, deposits, 1, "the deposit-info push must decode into the deposit struct")
		require.Len(t, deposits[0].Data, 1, "the deposit row must decode")
		assert.Equal(t, "D12345", deposits[0].Data[0].DepositID, "the deposit id should decode")
	})

	t.Run("grid positions decode the position fields", func(t *testing.T) {
		t.Parallel()
		require.Len(t, gridPositions, 1, "the grid-positions push must decode into the grid position struct")
		require.Len(t, gridPositions[0].Data, 1, "the grid position row must decode")
		row := gridPositions[0].Data[0]
		assert.Equal(t, "grid-client-1", row.AlgoClientOrderID, "the documented algoClOrdId should decode")
		assert.Equal(t, 29181.46, row.AveragePrice.Float64(), "the entry price should decode")
		assert.Equal(t, "USDT", row.Currency, "the margin currency should decode")
		assert.Equal(t, 12.5, row.UnrealisedPNL.Float64(), "the unrealised PnL should decode")
		assert.Equal(t, 0.012, row.UnrealisedPNLRatio.Float64(), "the unrealised PnL ratio should decode")
		assert.Equal(t, "35", row.Position, "the position size should decode")
	})

	t.Run("orders schema decodes every documented field", func(t *testing.T) {
		t.Parallel()
		frame := []byte(`{"arg":{"channel":"orders","instType":"SWAP"},"data":[{"instType":"SWAP","instId":"BTC-USDT-SWAP","ordId":"680800019749904384","clOrdId":"ord-client-1","algoId":"algo-1","algoClOrdId":"algo-client-1","attachAlgoClOrdId":"attach-client-1","state":"partially_filled","fillPnl":"12.5","fillPxVol":"0.4","fillPxUsd":"42000","fillMarkVol":"0.39","fillFwdPx":"42001","fillMarkPx":"42000.5","fillIdxPx":"41999","pxUsd":"42000","pxVol":"0.4","pxType":"last","cancelSource":"1","amendSource":"2","stpMode":"cancel_maker","lastPx":"42000.5","reduceOnly":"true","attachAlgoOrds":[{"attachAlgoClOrdId":"tp-1","tpTriggerPx":"50000","tpOrdPx":"-1"}]}]}`)
		var resp WsOrderResponse
		require.NoError(t, json.Unmarshal(frame, &resp), "the orders push schema must decode")
		row := resp.Data[0]
		assert.Equal(t, "1", row.CancelSource, "the documented cancelSource should decode")
		assert.Equal(t, "2", row.AmendSource, "the documented amendSource should decode")
		assert.Equal(t, 41999.0, row.FillIndexPrice.Float64(), "the documented fillIdxPx should decode")
		assert.Equal(t, "last", row.OptionPriceType, "the documented pxType should decode")
		assert.True(t, row.ReduceOnly, "the reduceOnly flag should decode from its quoted wire form")
	})
}
