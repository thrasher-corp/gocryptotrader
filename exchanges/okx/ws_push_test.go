package okx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// TestWSPushSchemasDecode guards the websocket push schemas against the
// documented response fields: price-limit must decode into its own limit
// fields, withdrawal-info and deposit-info must decode into their structs,
// the account and rfqs envelopes must carry their documented pagination and
// timestamp forms, and the grid-positions, orders, algo-order, recurring-buy
// and spread-ticker pushes must carry their documented fields.
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
	var accountPushes []*WsAccountChannelPushData
	var requestForQuotePushes []*WsRequestForQuote
	var liquidations []*WsLiquidationOrders
	var algoOrderPushes []*WsAlgoOrder
	var advancedAlgoOrderPushes []*WsAdvancedAlgoOrder
	var recurringBuyPushes []*struct {
		Arguments SubscriptionInfo    `json:"arg"`
		Data      []RecurringBuyOrder `json:"data"`
	}
	var spreadTickerBatches [][]ticker.Price

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
		case *WsAccountChannelPushData:
			accountPushes = append(accountPushes, v)
		case *WsRequestForQuote:
			requestForQuotePushes = append(requestForQuotePushes, v)
		case *WsLiquidationOrders:
			liquidations = append(liquidations, v)
		case *WsAlgoOrder:
			algoOrderPushes = append(algoOrderPushes, v)
		case *WsAdvancedAlgoOrder:
			advancedAlgoOrderPushes = append(advancedAlgoOrderPushes, v)
		case *struct {
			Arguments SubscriptionInfo    `json:"arg"`
			Data      []RecurringBuyOrder `json:"data"`
		}:
			recurringBuyPushes = append(recurringBuyPushes, v)
		case []ticker.Price:
			spreadTickerBatches = append(spreadTickerBatches, v)
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
		assert.Equal(t, "USDT", row.Currency.String(), "the margin currency should decode")
		assert.Equal(t, 12.5, row.UnrealizedProfitAndLoss.Float64(), "the unrealised PnL should decode")
		assert.Equal(t, 0.012, row.UnrealizedProfitAndLossRatio.Float64(), "the unrealised PnL ratio should decode")
		assert.Equal(t, 35.0, row.Position.Float64(), "the position size should decode")
	})

	t.Run("account snapshot decodes the pagination fields", func(t *testing.T) {
		t.Parallel()
		require.Len(t, accountPushes, 1, "the account push must decode into the account push struct")
		require.Len(t, accountPushes[0].Data, 1, "the account row must decode")
		assert.Equal(t, "snapshot", accountPushes[0].EventType, "the documented eventType should decode")
		assert.Equal(t, uint64(1), accountPushes[0].CurrentPage, "the documented curPage integer should decode")
		assert.True(t, accountPushes[0].LastPage, "the documented lastPage boolean should decode")
	})

	t.Run("rfqs push decodes the millisecond timestamps", func(t *testing.T) {
		t.Parallel()
		require.Len(t, requestForQuotePushes, 1, "the rfqs push must decode into the RFQ push struct")
		require.Len(t, requestForQuotePushes[0].Data, 1, "the rfq row must decode")
		row := requestForQuotePushes[0].Data[0]
		assert.Equal(t, int64(1611033737572), row.CreationTime.Time().UnixMilli(), "the documented cTime millisecond string should decode")
		assert.Equal(t, int64(1611033737572), row.UpdateTime.Time().UnixMilli(), "the documented uTime millisecond string should decode")
		assert.Equal(t, int64(1611033857557), row.ValidUntil.Time().UnixMilli(), "the documented validUntil millisecond string should decode")
		assert.Equal(t, "active", row.State, "the documented state should decode")
		assert.Equal(t, "22534", row.RequestForQuoteID, "the documented rfqId should decode")
	})

	t.Run("liquidation orders decode the envelope data", func(t *testing.T) {
		t.Parallel()
		require.Len(t, liquidations, 1, "the liquidation-orders push must decode into the envelope struct")
		require.Len(t, liquidations[0].Data, 1, "the liquidation order row must decode")
		row := liquidations[0].Data[0]
		assert.Equal(t, "IOST-USDT-SWAP", row.InstrumentID, "the instrument id should decode")
		require.Len(t, row.Details, 1, "the liquidation detail must decode")
		assert.Equal(t, 0.007831, row.Details[0].BankruptcyPrice.Float64(), "the bankruptcy price should decode")
	})

	t.Run("orders-algo push decodes the reduceOnly string", func(t *testing.T) {
		t.Parallel()
		require.Len(t, algoOrderPushes, 1, "the orders-algo push must decode into the algo order struct")
		require.Len(t, algoOrderPushes[0].Data, 1, "the algo order row must decode")
		assert.Equal(t, "true", algoOrderPushes[0].Data[0].ReduceOnly, "the documented reduceOnly string should decode")
		assert.Equal(t, 62916.5, algoOrderPushes[0].Data[0].LastPrice.Float64(), "the last filled price should decode")
	})

	t.Run("algo-advance push decodes the empty reduceOnly form", func(t *testing.T) {
		t.Parallel()
		require.Len(t, advancedAlgoOrderPushes, 1, "the algo-advance push must decode into the advanced algo order struct")
		require.Len(t, advancedAlgoOrderPushes[0].Data, 1, "the advanced algo order row must decode")
		assert.Empty(t, advancedAlgoOrderPushes[0].Data[0].ReduceOnly, "the documented empty reduceOnly form should decode")
	})

	t.Run("algo-recurring-buy push decodes the funding source array", func(t *testing.T) {
		t.Parallel()
		require.Len(t, recurringBuyPushes, 1, "the algo-recurring-buy push must decode into the recurring buy struct")
		require.Len(t, recurringBuyPushes[0].Data, 1, "the recurring buy row must decode")
		row := recurringBuyPushes[0].Data[0]
		assert.Equal(t, []string{"1"}, row.Source, "the documented funding source array should decode")
		assert.Equal(t, "USDT", row.TradeQuoteCurrency.String(), "the documented tradeQuoteCcy should decode")
		require.Len(t, row.RecurringList, 1, "the recurring list must decode")
		assert.Equal(t, 30000.0, row.RecurringList[0].MinimumPrice.Float64(), "the documented minPx should decode")
		assert.Equal(t, 50000.0, row.RecurringList[0].MaximumPrice.Float64(), "the documented maxPx should decode")
	})

	t.Run("sprd-tickers push maps the 24 hour figures into the relayed ticker", func(t *testing.T) {
		t.Parallel()
		require.Len(t, spreadTickerBatches, 1, "the sprd-tickers push must relay a processed ticker batch")
		require.Len(t, spreadTickerBatches[0], 1, "the spread ticker row must relay")
		row := spreadTickerBatches[0][0]
		assert.Equal(t, 4.0, row.Open, "the documented open24h should map into the relayed ticker")
		assert.Equal(t, 14.5, row.High, "the documented high24h should map into the relayed ticker")
		assert.Equal(t, -2.2, row.Low, "the documented low24h should map into the relayed ticker")
		assert.Equal(t, 14.5, row.Last, "the last price should relay")
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
		assert.Equal(t, "last", row.PriceType, "the documented pxType should decode")
		assert.True(t, row.ReduceOnly, "the reduceOnly flag should decode from its quoted wire form")
	})
}

// TestWSGridOrderSchemasDecode pins the grid order push schemas: the quoted
// trade count decodes into an unsigned integer, the quoted decimals decode
// into numbers and the unset trigger prices decode from the empty string.
func TestWSGridOrderSchemasDecode(t *testing.T) {
	t.Parallel()
	var spot WsSpotGridAlgoOrder
	require.NoError(t, json.Unmarshal([]byte(`{"arg": {"channel": "grid-orders-spot","instType": "ANY"},"data": [{"algoId": "448965992920907776","algoOrdType": "grid","annualizedRate": "0","arbitrageNum": "0","baseSz": "0","cTime": "1653313834104","cancelType": "0","curBaseSz": "0.001776289214","curQuoteSz": "46.801755866","floatProfit": "-0.4953878967772","gridNum": "6","gridProfit": "0","instId": "BTC-USDC","instType": "SPOT","investment": "100","maxPx": "33444.8","minPx": "24323.5","pTime": "1653476023742","perMaxProfitRate": "0.060375293181491054543","perMinProfitRate": "0.0455275366818586","pnlRatio": "0","quoteSz": "100","runPx": "30478.1","runType": "1","singleAmt": "0.00059261","slTriggerPx": "","state": "running","stopResult": "0","stopType": "0","totalAnnualizedRate": "-0.9643551057262827","totalPnl": "-0.4953878967772","tpTriggerPx": "","tradeNum": "3","triggerTime": "1653378736894","uTime": "1653378736894"}]}`), &spot), "the spot grid push must decode")
	require.Len(t, spot.Data, 1, "the spot grid row must decode")
	spotRow := spot.Data[0]
	assert.Equal(t, uint64(3), spotRow.TradeNumber, "the documented tradeNum quoted count should decode")
	assert.Equal(t, 100.0, spotRow.Investment.Float64(), "the documented investment should decode")
	assert.Equal(t, 30478.1, spotRow.RunPrice.Float64(), "the documented runPx should decode")
	assert.Equal(t, -0.9643551057262827, spotRow.TotalAnnualisedRate.Float64(), "the documented totalAnnualizedRate should decode")
	assert.Zero(t, spotRow.StopLossTriggerPrice.Float64(), "the empty slTriggerPx should decode as zero")

	var contract WsContractGridAlgoOrder
	require.NoError(t, json.Unmarshal([]byte(`{"arg": {"channel": "grid-orders-contract","instType": "ANY"},"data": [{"actualLever": "1.02","algoId": "449327675342323712","algoOrdType": "contract_grid","annualizedRate": "0.7572437878956523","arbitrageNum": "1","basePos": true,"cTime": "1653400065912","cancelType": "0","direction": "long","eq": "10129.419829834853","floatProfit": "109.537858234853","gridNum": "50","gridProfit": "19.8819716","instId": "BTC-USDT-SWAP","instType": "SWAP","investment": "10000","lever": "5","liqPx": "603.2149534767834","maxPx": "100000","minPx": "10","pTime": "1653484573918","perMaxProfitRate": "995.7080916791230692","perMinProfitRate": "0.0946277854875634","pnlRatio": "0.0129419829834853","runPx": "29216.3","runType": "1","singleAmt": "1","slTriggerPx": "","state": "running","stopType": "0","sz": "10000","tag": "","totalAnnualizedRate": "4.929207431970923","totalPnl": "129.419829834853","tpTriggerPx": "","tradeNum": "37","triggerTime": "1653400066940","uTime": "1653484573589","uly": "BTC-USDT"}]}`), &contract), "the contract grid push must decode")
	require.Len(t, contract.Data, 1, "the contract grid row must decode")
	contractRow := contract.Data[0]
	assert.Equal(t, uint64(37), contractRow.TradeNumber, "the documented tradeNum quoted count should decode")
	assert.Equal(t, 1.02, contractRow.ActualLever.Float64(), "the documented actualLever should decode")
	assert.Equal(t, 5.0, contractRow.Leverage.Float64(), "the documented lever should decode")
	assert.Equal(t, 10000.0, contractRow.Investment.Float64(), "the documented investment should decode")
	assert.Zero(t, contractRow.StopLossTriggerPrice.Float64(), "the empty slTriggerPx should decode as zero")
}
