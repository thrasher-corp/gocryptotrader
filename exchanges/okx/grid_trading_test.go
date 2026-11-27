package okx

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestPlaceGridAlgoOrder(t *testing.T) {
	t.Parallel()
	contractGrid := func() *PlaceGridAlgoOrderRequest {
		return &PlaceGridAlgoOrderRequest{InstrumentID: mainPair.String(), AlgoOrderType: AlgoOrdTypeContractGrid, MaxPrice: 1000, MinPrice: 1200, GridNumber: 123, Size: 123, Direction: positionSideLong, Leverage: 2}
	}
	for _, tc := range []struct {
		name string
		arg  func() *PlaceGridAlgoOrderRequest
		err  error
	}{
		{"nil request", func() *PlaceGridAlgoOrderRequest { return nil }, common.ErrNilPointer},
		{"empty request", func() *PlaceGridAlgoOrderRequest { return &PlaceGridAlgoOrderRequest{BasePosition: true} }, errMissingInstrumentID},
		{"missing algo order type", func() *PlaceGridAlgoOrderRequest { a := contractGrid(); a.AlgoOrderType = ""; return a }, errInvalidAlgoOrderType},
		{"missing max price", func() *PlaceGridAlgoOrderRequest { a := contractGrid(); a.MaxPrice = 0; return a }, limits.ErrPriceBelowMin},
		{"missing min price", func() *PlaceGridAlgoOrderRequest { a := contractGrid(); a.MinPrice = 0; return a }, limits.ErrPriceBelowMin},
		{"missing grid number", func() *PlaceGridAlgoOrderRequest { a := contractGrid(); a.GridNumber = 0; return a }, errInvalidGridNumber},
		{"spot grid without an investment", func() *PlaceGridAlgoOrderRequest { a := contractGrid(); a.AlgoOrderType = AlgoOrdTypeGrid; return a }, order.ErrAmountMustBeSet},
		{"contract grid without size", func() *PlaceGridAlgoOrderRequest { a := contractGrid(); a.Size = 0; return a }, order.ErrAmountMustBeSet},
		{"contract grid without direction", func() *PlaceGridAlgoOrderRequest { a := contractGrid(); a.Direction = ""; return a }, errMissingRequiredArgumentDirection},
		{"contract grid without leverage", func() *PlaceGridAlgoOrderRequest { a := contractGrid(); a.Leverage = 0; return a }, errInvalidLeverage},
		{"trigger without action", func() *PlaceGridAlgoOrderRequest {
			a := contractGrid()
			a.TriggerParameters = []TradingBotTriggerParameter{{TriggerStrategy: "instant"}}
			return a
		}, errTriggerActionRequired},
		{"trigger without strategy", func() *PlaceGridAlgoOrderRequest {
			a := contractGrid()
			a.TriggerParameters = []TradingBotTriggerParameter{{TriggerAction: "start"}}
			return a
		}, errTriggerStrategyRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.PlaceGridAlgoOrder(t.Context(), tc.arg())
			assert.ErrorIs(t, err, tc.err, "PlaceGridAlgoOrder should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *PlaceGridAlgoOrderRequest
			body string
		}{
			{
				name: "contract grid with every parameter",
				arg: &PlaceGridAlgoOrderRequest{
					InstrumentID: perpetualSwapPair.String(), AlgoOrderType: AlgoOrdTypeContractGrid, MaxPrice: 5000, MinPrice: 400, GridNumber: 10, RunType: "1",
					TakeProfitTriggerPrice: 6000, StopLossTriggerPrice: 300, AlgoClientOrderID: "gridClient1", Tag: "gridTag1", ProfitSharingRatio: 0.1,
					TriggerParameters: []TradingBotTriggerParameter{
						{TriggerAction: "start", TriggerStrategy: "rsi", DelaySeconds: 60, Timeframe: "30m", Threshold: 10, TriggerCondition: "cross", TimePeriod: 14},
						{TriggerAction: "stop", TriggerStrategy: "price", TriggerPrice: 1000, StopType: "2"},
					},
					Size: 200, Direction: positionSideLong, Leverage: 2, BasePosition: true, TakeProfitRatio: 0.1, StopLossRatio: 0.05,
				},
				body: `{"instId":"BTC-USDT-SWAP","algoOrdType":"contract_grid","maxPx":"5000","minPx":"400","gridNum":"10","runType":"1","tpTriggerPx":"6000","slTriggerPx":"300","algoClOrdId":"gridClient1","tag":"gridTag1","profitSharingRatio":"0.1","triggerParams":[{"triggerAction":"start","triggerStrategy":"rsi","delaySeconds":"60","timeframe":"30m","thold":"10","triggerCond":"cross","timePeriod":"14"},{"triggerAction":"stop","triggerStrategy":"price","triggerPx":"1000","stopType":"2"}],"sz":"200","direction":"long","lever":"2","basePos":true,"tpRatio":"0.1","slRatio":"0.05"}`,
			},
			{
				name: "spot grid with every parameter",
				arg: &PlaceGridAlgoOrderRequest{
					InstrumentID: mainPair.String(), AlgoOrderType: AlgoOrdTypeGrid, MaxPrice: 5000, MinPrice: 400, GridNumber: 10, RunType: "2",
					TakeProfitTriggerPrice: 6000, StopLossTriggerPrice: 300, AlgoClientOrderID: "gridClient2", Tag: "gridTag2", ProfitSharingRatio: 0.2,
					TriggerParameters: []TradingBotTriggerParameter{{TriggerAction: "stop", TriggerStrategy: "price", TriggerPrice: 1000}},
					QuoteSize:         25, BaseSize: 0.001, TradeQuoteCurrency: currency.USDT,
				},
				body: `{"instId":"BTC-USDT","algoOrdType":"grid","maxPx":"5000","minPx":"400","gridNum":"10","runType":"2","tpTriggerPx":"6000","slTriggerPx":"300","algoClOrdId":"gridClient2","tag":"gridTag2","profitSharingRatio":"0.2","triggerParams":[{"triggerAction":"stop","triggerStrategy":"price","triggerPx":"1000"}],"quoteSz":"25","baseSz":"0.001","tradeQuoteCcy":"USDT"}`,
			},
			{
				name: "spot grid with required parameters only",
				arg:  &PlaceGridAlgoOrderRequest{InstrumentID: mainPair.String(), AlgoOrderType: AlgoOrdTypeGrid, MaxPrice: 5000, MinPrice: 400, GridNumber: 10, QuoteSize: 25},
				body: `{"instId":"BTC-USDT","algoOrdType":"grid","maxPx":"5000","minPx":"400","gridNum":"10","quoteSz":"25"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/grid/order-algo", tc.body, tradingBotAlgoIDData).PlaceGridAlgoOrder(t.Context(), tc.arg)
				require.NoError(t, err, "PlaceGridAlgoOrder must not error")
			})
		}
	})

	t.Run("place order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.PlaceGridAlgoOrder(contextGenerate(), &PlaceGridAlgoOrderRequest{
			InstrumentID: "BTC-USD-SWAP", AlgoOrderType: AlgoOrdTypeContractGrid, MaxPrice: 5000, MinPrice: 400, GridNumber: 10, RunType: "1",
			Size: 200, Direction: positionSideLong, Leverage: 2,
		})
		require.NoError(t, err, "PlaceGridAlgoOrder must not error")
		if mockTests {
			assert.Equal(t, &TradingBotOrderResponse{AlgoID: "447053782921515008"}, result, "PlaceGridAlgoOrder should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "PlaceGridAlgoOrder should return an algo ID")
		}
	})
}

func TestAmendGridAlgoOrderBasicParameters(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AmendGridAlgoOrderBasicParametersRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &AmendGridAlgoOrderBasicParametersRequest{MinPrice: 10, MaxPrice: 100, GridNumber: 5}, errAlgoIDRequired},
		{"missing min price", &AmendGridAlgoOrderBasicParametersRequest{AlgoID: "448965992920907776", MaxPrice: 100, GridNumber: 5}, limits.ErrPriceBelowMin},
		{"missing max price", &AmendGridAlgoOrderBasicParametersRequest{AlgoID: "448965992920907776", MinPrice: 10, GridNumber: 5}, limits.ErrPriceBelowMin},
		{"missing grid number", &AmendGridAlgoOrderBasicParametersRequest{AlgoID: "448965992920907776", MinPrice: 10, MaxPrice: 100}, errInvalidGridNumber},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AmendGridAlgoOrderBasicParameters(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "AmendGridAlgoOrderBasicParameters should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *AmendGridAlgoOrderBasicParametersRequest
			body string
		}{
			{"every parameter", &AmendGridAlgoOrderBasicParametersRequest{AlgoID: "448965992920907776", MinPrice: 10, MaxPrice: 100, GridNumber: 5, TopUpAmount: 123.45}, `{"algoId":"448965992920907776","minPx":"10","maxPx":"100","gridNum":"5","topupAmount":"123.45"}`},
			{"required parameters only", &AmendGridAlgoOrderBasicParametersRequest{AlgoID: "448965992920907776", MinPrice: 10, MaxPrice: 100, GridNumber: 5}, `{"algoId":"448965992920907776","minPx":"10","maxPx":"100","gridNum":"5"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/grid/amend-algo-basic-param", tc.body, tradingBotAlgoIDData).AmendGridAlgoOrderBasicParameters(t.Context(), tc.arg)
				require.NoError(t, err, "AmendGridAlgoOrderBasicParameters must not error")
			})
		}
	})

	t.Run("top up required alongside an error", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"code":"55186","msg":"Due to market fluctuations, your investment amount is too large to apply these modifications.","data":[{"algoId":"4283223775520665600","maxTopupAmount":"12456.78","requiredTopupAmount":"12.34"}]}`))
		}))
		result, err := ex.AmendGridAlgoOrderBasicParameters(t.Context(), &AmendGridAlgoOrderBasicParametersRequest{AlgoID: "4283223775520665600", MinPrice: 10, MaxPrice: 100, GridNumber: 5})
		require.ErrorContains(t, err, "your investment amount is too large", "AmendGridAlgoOrderBasicParameters must report OKX's error")
		exp := &AmendGridAlgoOrderBasicParametersResponse{AlgoID: "4283223775520665600", RequiredTopUpAmount: 12.34, MaxTopUpAmount: 12456.78}
		assert.Equal(t, exp, result, "AmendGridAlgoOrderBasicParameters should return the top up OKX requires beside the error")
	})

	t.Run("amend order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.AmendGridAlgoOrderBasicParameters(contextGenerate(), &AmendGridAlgoOrderBasicParametersRequest{AlgoID: "448965992920907776", MinPrice: 10, MaxPrice: 100, GridNumber: 5, TopUpAmount: 123.45})
		require.NoError(t, err, "AmendGridAlgoOrderBasicParameters must not error")
		if mockTests {
			exp := &AmendGridAlgoOrderBasicParametersResponse{AlgoID: "448965992920907776", RequiredTopUpAmount: 12.34, MaxTopUpAmount: 12456.78}
			assert.Equal(t, exp, result, "AmendGridAlgoOrderBasicParameters should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "AmendGridAlgoOrderBasicParameters should return an algo ID")
		}
	})
}

func TestAmendGridAlgoOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AmendGridAlgoOrderRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"empty request", &AmendGridAlgoOrderRequest{}, errAlgoIDRequired},
		{"missing algo ID", &AmendGridAlgoOrderRequest{TakeProfitTriggerPrice: new(types.Number(1234.5))}, errAlgoIDRequired},
		{"missing instrument ID", &AmendGridAlgoOrderRequest{AlgoID: "560472804207104000"}, errMissingInstrumentID},
		{"trigger without action", &AmendGridAlgoOrderRequest{AlgoID: "560472804207104000", InstrumentID: mainPair.String(), TriggerParameters: []TradingBotTriggerParameter{{TriggerStrategy: "price"}}}, errTriggerActionRequired},
		{"trigger without strategy", &AmendGridAlgoOrderRequest{AlgoID: "560472804207104000", InstrumentID: mainPair.String(), TriggerParameters: []TradingBotTriggerParameter{{TriggerAction: "stop"}}}, errTriggerStrategyRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AmendGridAlgoOrder(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "AmendGridAlgoOrder should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *AmendGridAlgoOrderRequest
			body string
		}{
			{
				name: "every parameter, cancelling the take profit trigger",
				arg: &AmendGridAlgoOrderRequest{
					AlgoID: "448965992920907776", InstrumentID: perpetualSwapPair.String(), StopLossTriggerPrice: new(types.Number(1200)), TakeProfitTriggerPrice: new(types.Number(0)),
					TakeProfitRatio: new(types.Number(0.1)), StopLossRatio: new(types.Number(0.05)), TopUpAmount: 10,
					TriggerParameters: []TradingBotTriggerParameter{{TriggerAction: "stop", TriggerStrategy: "price", TriggerPrice: 1000, StopType: "1"}},
				},
				body: `{"algoId":"448965992920907776","instId":"BTC-USDT-SWAP","slTriggerPx":"1200","tpTriggerPx":"","tpRatio":"0.1","slRatio":"0.05","topUpAmt":"10","triggerParams":[{"triggerAction":"stop","triggerStrategy":"price","triggerPx":"1000","stopType":"1"}]}`,
			},
			{"required parameters only", &AmendGridAlgoOrderRequest{AlgoID: "448965992920907776", InstrumentID: perpetualSwapPair.String()}, `{"algoId":"448965992920907776","instId":"BTC-USDT-SWAP"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/grid/amend-order-algo", tc.body, tradingBotAlgoIDData).AmendGridAlgoOrder(t.Context(), tc.arg)
				require.NoError(t, err, "AmendGridAlgoOrder must not error")
			})
		}
	})

	t.Run("status message of a failed amendment", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"code":"1","msg":"Operation failed.","data":[{"algoId":"","sCode":"51000","sMsg":"Parameter algoId error"}]}`))
		}))
		_, err := ex.AmendGridAlgoOrder(t.Context(), &AmendGridAlgoOrderRequest{AlgoID: "1", InstrumentID: perpetualSwapPair.String(), StopLossTriggerPrice: new(types.Number(1))})
		assert.ErrorContains(t, err, "Parameter algoId error", "AmendGridAlgoOrder should report OKX's status message")
	})

	t.Run("amend order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.AmendGridAlgoOrder(contextGenerate(), &AmendGridAlgoOrderRequest{AlgoID: "448965992920907776", InstrumentID: mainPair.String(), StopLossTriggerPrice: new(types.Number(1200)), TakeProfitTriggerPrice: new(types.Number(0))})
		require.NoError(t, err, "AmendGridAlgoOrder must not error")
		if mockTests {
			assert.Equal(t, &TradingBotOrderResponse{AlgoID: "448965992920907776"}, result, "AmendGridAlgoOrder should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "AmendGridAlgoOrder should return an algo ID")
		}
	})
}

func TestStopGridAlgoOrder(t *testing.T) {
	t.Parallel()
	valid := StopGridAlgoOrderRequest{AlgoID: "198273485", InstrumentID: mainPair.String(), AlgoOrderType: AlgoOrdTypeGrid, StopType: "1"}
	for _, tc := range []struct {
		name string
		arg  []StopGridAlgoOrderRequest
		err  error
	}{
		{"no orders", []StopGridAlgoOrderRequest{}, common.ErrEmptyParams},
		{"more than 10 orders", make([]StopGridAlgoOrderRequest, 11), errExceedLimit},
		{"empty order", []StopGridAlgoOrderRequest{{}}, errAlgoIDRequired},
		{"missing algo ID", []StopGridAlgoOrderRequest{{InstrumentID: mainPair.String(), AlgoOrderType: AlgoOrdTypeGrid, StopType: "1"}}, errAlgoIDRequired},
		{"missing instrument ID", []StopGridAlgoOrderRequest{{AlgoID: "algo_id", AlgoOrderType: AlgoOrdTypeGrid, StopType: "1"}}, errMissingInstrumentID},
		{"missing algo order type", []StopGridAlgoOrderRequest{{AlgoID: "algo_id", InstrumentID: mainPair.String(), StopType: "1"}}, errInvalidAlgoOrderType},
		{"missing stop type", []StopGridAlgoOrderRequest{{AlgoID: "algo_id", InstrumentID: mainPair.String(), AlgoOrderType: AlgoOrdTypeGrid}}, errStopTypeRequired},
		{"second order invalid", []StopGridAlgoOrderRequest{valid, {AlgoID: "algo_id"}}, errMissingInstrumentID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.StopGridAlgoOrder(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "StopGridAlgoOrder should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/tradingBot/grid/stop-order-algo", `[{"algoId":"198273485","instId":"BTC-USDT","algoOrdType":"grid","stopType":"1"},{"algoId":"571039869070475264","instId":"BTC-USDT-SWAP","algoOrdType":"contract_grid","stopType":"2"}]`, tradingBotAlgoIDData)
		_, err := ex.StopGridAlgoOrder(t.Context(), []StopGridAlgoOrderRequest{valid, {AlgoID: "571039869070475264", InstrumentID: perpetualSwapPair.String(), AlgoOrderType: AlgoOrdTypeContractGrid, StopType: "2"}})
		require.NoError(t, err, "StopGridAlgoOrder must not error")
	})

	t.Run("partial failure returns the rows", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"code":"2","msg":"Bulk operation partially succeeded.","data":[{"algoClOrdId":"","algoId":"198273485","sCode":"0","sMsg":"","tag":""},{"algoClOrdId":"","algoId":"198273486","sCode":"51000","sMsg":"Parameter algoId error","tag":""}]}`))
		}))
		result, err := ex.StopGridAlgoOrder(t.Context(), []StopGridAlgoOrderRequest{valid, {AlgoID: "198273486", InstrumentID: mainPair.String(), AlgoOrderType: AlgoOrdTypeGrid, StopType: "1"}})
		require.ErrorIs(t, err, errPartialSuccess, "StopGridAlgoOrder must return the partial success")
		assert.ErrorContains(t, err, "Parameter algoId error", "StopGridAlgoOrder should report the failed row's status message")
		exp := []TradingBotOrderResponse{{AlgoID: "198273485"}, {AlgoID: "198273486", StatusCode: 51000, StatusMessage: "Parameter algoId error"}}
		assert.Equal(t, exp, result, "StopGridAlgoOrder should return every row beside the error")
	})

	t.Run("stop order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.StopGridAlgoOrder(contextGenerate(), []StopGridAlgoOrderRequest{valid})
		require.NoError(t, err, "StopGridAlgoOrder must not error")
		if mockTests {
			assert.Equal(t, []TradingBotOrderResponse{{AlgoID: "198273485"}}, result, "StopGridAlgoOrder should decode every field")
		} else {
			assert.NotEmpty(t, result, "StopGridAlgoOrder should return the stopped orders")
		}
	})
}

func TestClosePositionForContractGrid(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *ClosePositionForContractGridRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"empty request", &ClosePositionForContractGridRequest{}, errAlgoIDRequired},
		{"missing algo ID", &ClosePositionForContractGridRequest{MarketCloseAllPositions: true}, errAlgoIDRequired},
		{"partial close without size", &ClosePositionForContractGridRequest{AlgoID: "448965992920907776"}, order.ErrAmountMustBeSet},
		{"partial close without price", &ClosePositionForContractGridRequest{AlgoID: "448965992920907776", Size: 123}, limits.ErrPriceBelowMin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.ClosePositionForContractGrid(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "ClosePositionForContractGrid should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *ClosePositionForContractGridRequest
			body string
		}{
			{"partial close with every parameter", &ClosePositionForContractGridRequest{AlgoID: "448965992920907776", Size: 2, Price: 30000}, `{"algoId":"448965992920907776","mktClose":false,"sz":"2","px":"30000"}`},
			{"market close with required parameters only", &ClosePositionForContractGridRequest{AlgoID: "448965992920907776", MarketCloseAllPositions: true}, `{"algoId":"448965992920907776","mktClose":true}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/grid/close-position", tc.body, tradingBotAlgoIDData).ClosePositionForContractGrid(t.Context(), tc.arg)
				require.NoError(t, err, "ClosePositionForContractGrid must not error")
			})
		}
	})

	t.Run("close position", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.ClosePositionForContractGrid(contextGenerate(), &ClosePositionForContractGridRequest{AlgoID: "448965992920907776", MarketCloseAllPositions: true})
		require.NoError(t, err, "ClosePositionForContractGrid must not error")
		if mockTests {
			exp := &ContractGridCloseOrderResponse{AlgoID: "448965992920907776", AlgoClientOrderID: "gridClose1", Tag: "gridTag1"}
			assert.Equal(t, exp, result, "ClosePositionForContractGrid should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "ClosePositionForContractGrid should return an algo ID")
		}
	})
}

func TestCancelClosePositionOrderForContractGrid(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CancelClosePositionOrderRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"empty request", &CancelClosePositionOrderRequest{}, errAlgoIDRequired},
		{"missing algo ID", &CancelClosePositionOrderRequest{OrderID: "570627699870375936"}, errAlgoIDRequired},
		{"missing order ID", &CancelClosePositionOrderRequest{AlgoID: "448965992920907776"}, order.ErrOrderIDNotSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CancelClosePositionOrderForContractGrid(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "CancelClosePositionOrderForContractGrid should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/tradingBot/grid/cancel-close-order", `{"algoId":"448965992920907776","ordId":"570627699870375936"}`, tradingBotAlgoIDData)
		_, err := ex.CancelClosePositionOrderForContractGrid(t.Context(), &CancelClosePositionOrderRequest{AlgoID: "448965992920907776", OrderID: "570627699870375936"})
		require.NoError(t, err, "CancelClosePositionOrderForContractGrid must not error")
	})

	t.Run("cancel order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.CancelClosePositionOrderForContractGrid(contextGenerate(), &CancelClosePositionOrderRequest{AlgoID: "448965992920907776", OrderID: "570627699870375936"})
		require.NoError(t, err, "CancelClosePositionOrderForContractGrid must not error")
		if mockTests {
			exp := &ContractGridCloseOrderResponse{AlgoID: "448965992920907776", OrderID: "570627699870375936"}
			assert.Equal(t, exp, result, "CancelClosePositionOrderForContractGrid should decode every field")
		} else {
			assert.NotEmpty(t, result.OrderID, "CancelClosePositionOrderForContractGrid should return the order ID")
		}
	})
}

func TestInstantTriggerGridAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.InstantTriggerGridAlgoOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.InstantTriggerGridAlgoOrder(t.Context(), &InstantTriggerGridAlgoOrderRequest{TopUpAmount: 50})
	require.ErrorIs(t, err, errAlgoIDRequired)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *InstantTriggerGridAlgoOrderRequest
			body string
		}{
			{"every parameter", &InstantTriggerGridAlgoOrderRequest{AlgoID: "123456789", TopUpAmount: 50}, `{"algoId":"123456789","topUpAmt":"50"}`},
			{"required parameters only", &InstantTriggerGridAlgoOrderRequest{AlgoID: "123456789"}, `{"algoId":"123456789"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/grid/order-instant-trigger", tc.body, tradingBotAlgoIDData).InstantTriggerGridAlgoOrder(t.Context(), tc.arg)
				require.NoError(t, err, "InstantTriggerGridAlgoOrder must not error")
			})
		}
	})

	t.Run("trigger order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.InstantTriggerGridAlgoOrder(contextGenerate(), &InstantTriggerGridAlgoOrderRequest{AlgoID: "123456789"})
		require.NoError(t, err, "InstantTriggerGridAlgoOrder must not error")
		if mockTests {
			assert.Equal(t, &InstantTriggerGridAlgoOrderResponse{AlgoID: "123456789", AlgoClientOrderID: "gridTrigger1"}, result, "InstantTriggerGridAlgoOrder should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "InstantTriggerGridAlgoOrder should return an algo ID")
		}
	})
}

func TestGetGridAlgoOrdersList(t *testing.T) {
	t.Parallel()
	_, err := e.GetGridAlgoOrdersList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetGridAlgoOrdersList(t.Context(), &GridAlgoOrderListRequest{Limit: 1})
	require.ErrorIs(t, err, errInvalidAlgoOrderType)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetGridAlgoOrdersList(contextGenerate(), &GridAlgoOrderListRequest{AlgoOrderType: AlgoOrdTypeGrid, Limit: 1})
	require.NoError(t, err, "GetGridAlgoOrdersList must not error")
	if mockTests {
		exp := []GridAlgoOrderResponse{
			{
				AlgoID:          "568028283477164032",
				InstrumentType:  "SPOT",
				InstrumentID:    "BTC-USDT",
				CreationTime:    types.Time(time.UnixMilli(1681700496249)),
				UpdateTime:      types.Time(time.UnixMilli(1682062564350)),
				AlgoOrderType:   AlgoOrdTypeGrid,
				State:           "running",
				RebateTransfers: []GridRebateTransfer{{RebateCurrency: currency.BTC}, {RebateCurrency: currency.USDT}},
				TriggerParameters: []TradingBotTriggerParameter{
					{TriggerAction: "start", TriggerStrategy: "instant", TriggerType: "auto"},
					{TriggerAction: "stop", TriggerStrategy: "instant", TriggerPrice: 1000, StopType: "1", TriggerType: "manual"},
				},
				MaxPrice:           5000,
				MinPrice:           400,
				GridNumber:         10,
				RunType:            "1",
				Investment:         25,
				CancelType:         "0",
				QuoteSize:          25,
				Underlying:         "BTC-USDT",
				CopyType:           "0",
				TradeQuoteCurrency: currency.USDT,
			},
		}
		assert.Equal(t, exp, result, "GetGridAlgoOrdersList should decode every field")
	} else {
		assert.NotNil(t, result, "GetGridAlgoOrdersList should return a list")
	}
}

func TestGetGridAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetGridAlgoOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetGridAlgoOrderHistory(t.Context(), &GridAlgoOrderListRequest{Limit: 1})
	require.ErrorIs(t, err, errInvalidAlgoOrderType)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetGridAlgoOrderHistory(contextGenerate(), &GridAlgoOrderListRequest{AlgoOrderType: AlgoOrdTypeContractGrid, Limit: 1})
	require.NoError(t, err, "GetGridAlgoOrderHistory must not error")
	if mockTests {
		exp := []GridAlgoOrderResponse{
			{
				AlgoID:            "571039869070475264",
				AlgoClientOrderID: "gridLead1",
				InstrumentType:    "SWAP",
				InstrumentID:      "BTC-USDT-SWAP",
				CreationTime:      types.Time(time.UnixMilli(1682418514204)),
				UpdateTime:        types.Time(time.UnixMilli(1682492552257)),
				AlgoOrderType:     AlgoOrdTypeContractGrid,
				State:             "stopped",
				RebateTransfers:   []GridRebateTransfer{{Rebate: 0.0012, RebateCurrency: currency.USDT}},
				TriggerParameters: []TradingBotTriggerParameter{
					{TriggerAction: "start", TriggerStrategy: "price", TriggerPrice: 27000, TriggerTime: types.Time(time.UnixMilli(1682418561497)), TriggerType: "manual"},
					{TriggerAction: "stop", TriggerStrategy: "instant", StopType: "1", TriggerTime: types.Time(time.UnixMilli(1682492552250)), TriggerType: "manual"},
				},
				MaxPrice:               36437.3,
				MinPrice:               26931.9,
				GridNumber:             10,
				RunType:                "1",
				TakeProfitTriggerPrice: 40000,
				StopLossTriggerPrice:   25000,
				ArbitrageNumber:        3,
				TotalPNL:               10.0445652387089673,
				PNLRatio:               0.0879456523870897,
				Investment:             100,
				GridProfit:             1.25,
				FloatProfit:            8.7945652387089673,
				CancelType:             "2",
				StopType:               "1",
				StopResult:             "0",
				Direction:              positionSideLong,
				BasePosition:           true,
				Size:                   100,
				Leverage:               5,
				ActualLeverage:         2.3481494635276649,
				LiquidationPrice:       16370.482143120824,
				Underlying:             "BTC-USDT",
				InstrumentFamily:       "BTC-USDT",
				OrderFrozen:            5.38638,
				AvailableEquity:        52.3015392887089673,
				Tag:                    "gridLeadTag",
				ProfitSharingRatio:     0.1,
				CopyType:               "3",
				Fee:                    -0.2,
				FeeCurrency:            currency.USDT,
				FundingFee:             -0.3,
			},
		}
		assert.Equal(t, exp, result, "GetGridAlgoOrderHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetGridAlgoOrderHistory should return a list")
	}
}

func TestGridAlgoOrderListParams(t *testing.T) {
	t.Parallel()
	_, err := gridAlgoOrderListParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = gridAlgoOrderListParams(&GridAlgoOrderListRequest{})
	require.ErrorIs(t, err, errInvalidAlgoOrderType)

	params, err := gridAlgoOrderListParams(&GridAlgoOrderListRequest{AlgoOrderType: AlgoOrdTypeGrid, AlgoID: "448965992920907776", InstrumentID: mainPair.String(), InstrumentType: instTypeSpot, After: "448965992920907776", Before: "448965992920907700", Limit: 100})
	require.NoError(t, err, "gridAlgoOrderListParams must not error")
	exp := url.Values{"algoOrdType": {"grid"}, "algoId": {"448965992920907776"}, "instId": {"BTC-USDT"}, "instType": {"SPOT"}, "after": {"448965992920907776"}, "before": {"448965992920907700"}, "limit": {"100"}}
	assert.Equal(t, exp, params, "gridAlgoOrderListParams should set every parameter")
}

func TestGetGridAlgoOrderDetails(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *GridAlgoOrderRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo order type", &GridAlgoOrderRequest{AlgoID: "7878"}, errInvalidAlgoOrderType},
		{"missing algo ID", &GridAlgoOrderRequest{AlgoOrderType: AlgoOrdTypeGrid}, errAlgoIDRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetGridAlgoOrderDetails(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetGridAlgoOrderDetails should return the expected error")
		})
	}

	for _, tc := range []struct {
		name string
		arg  *GridAlgoOrderRequest
		exp  *GridAlgoOrderResponse
	}{
		{
			name: "spot grid",
			arg:  &GridAlgoOrderRequest{AlgoOrderType: AlgoOrdTypeGrid, AlgoID: "7878"},
			exp: &GridAlgoOrderResponse{
				AlgoID:            "7878",
				AlgoClientOrderID: "gridClient7878",
				InstrumentType:    "SPOT",
				InstrumentID:      "BTC-USDT",
				CreationTime:      types.Time(time.UnixMilli(1681181054927)),
				UpdateTime:        types.Time(time.UnixMilli(1681181186496)),
				AlgoOrderType:     AlgoOrdTypeGrid,
				State:             "stopped",
				RebateTransfers:   []GridRebateTransfer{{RebateCurrency: currency.BTC}, {RebateCurrency: currency.USDT}},
				TriggerParameters: []TradingBotTriggerParameter{
					{TriggerAction: "start", TriggerStrategy: "rsi", DelaySeconds: 60, Timeframe: "15m", Threshold: 30, TriggerCondition: "cross_up", TimePeriod: 14, TriggerTime: types.Time(time.UnixMilli(1681181114927)), TriggerType: "auto"},
					{TriggerAction: "stop", TriggerStrategy: "instant", StopType: "1", TriggerTime: types.Time(time.UnixMilli(1681181186484)), TriggerType: "manual"},
				},
				MaxPrice:            5000,
				MinPrice:            400,
				GridNumber:          10,
				RunType:             "1",
				TradeNumber:         9,
				ArbitrageNumber:     2,
				SingleAmount:        0.00101214,
				PerMinProfitRate:    0.0991200440528634356837,
				PerMaxProfitRate:    1.14570215,
				RunPrice:            30089.7,
				TotalPNL:            0.2,
				PNLRatio:            0.008,
				Investment:          25,
				GridProfit:          0.35,
				FloatProfit:         -0.15,
				TotalAnnualisedRate: 38.52019574554529,
				AnnualisedRate:      12.5,
				CancelType:          "1",
				StopType:            "1",
				ActiveOrderNumber:   8,
				QuoteSize:           25,
				BaseSize:            0.0005,
				CurrentQuoteSize:    24.5,
				CurrentBaseSize:     0.0002,
				Profit:              0.35,
				StopResult:          "1",
				CopyType:            "0",
				TradeQuoteCurrency:  currency.USDT,
			},
		},
		{
			name: "contract grid",
			arg:  &GridAlgoOrderRequest{AlgoOrderType: AlgoOrdTypeContractGrid, AlgoID: "571039869070475264"},
			exp: &GridAlgoOrderResponse{
				AlgoID:              "571039869070475264",
				InstrumentType:      "SWAP",
				InstrumentID:        "BTC-USDT-SWAP",
				CreationTime:        types.Time(time.UnixMilli(1682418514204)),
				UpdateTime:          types.Time(time.UnixMilli(1682492552257)),
				AlgoOrderType:       AlgoOrdTypeContractGrid,
				State:               "running",
				RebateTransfers:     []GridRebateTransfer{{RebateCurrency: currency.USDT}},
				TriggerParameters:   []TradingBotTriggerParameter{{TriggerAction: "start", TriggerStrategy: "price", TriggerPrice: 1, TriggerTime: types.Time(time.UnixMilli(1682418561497)), TriggerType: "manual"}},
				MaxPrice:            36437.3,
				MinPrice:            26931.9,
				GridNumber:          10,
				RunType:             "1",
				TradeNumber:         9,
				SingleAmount:        1,
				PerMinProfitRate:    0.1263869357706788,
				PerMaxProfitRate:    0.1687494513302446,
				RunPrice:            27306.9,
				TotalPNL:            8.7945652387089673,
				PNLRatio:            0.0879456523870897,
				Investment:          100,
				FloatProfit:         8.7945652387089673,
				TotalAnnualisedRate: 38.52019574554529,
				CancelType:          "0",
				StopType:            "0",
				ActiveOrderNumber:   10,
				Direction:           positionSideLong,
				BasePosition:        true,
				Size:                100,
				Leverage:            5,
				ActualLeverage:      2.3481494635276649,
				LiquidationPrice:    16370.482143120824,
				Underlying:          "BTC-USDT",
				InstrumentFamily:    "BTC-USDT",
				OrderFrozen:         5.38638,
				AvailableEquity:     52.3015392887089673,
				Equity:              108.7945652387089673,
				CopyType:            "0",
				TakeProfitRatio:     0.1,
				StopLossRatio:       0.05,
				Fee:                 -0.2,
				FeeCurrency:         currency.USDT,
				FundingFee:          -0.3,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			result, err := e.GetGridAlgoOrderDetails(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetGridAlgoOrderDetails must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetGridAlgoOrderDetails should decode every field")
			} else {
				assert.NotEmpty(t, result.AlgoID, "GetGridAlgoOrderDetails should return an algo ID")
			}
		})
	}
}

func TestGetGridAlgoSubOrders(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *GridAlgoSubOrdersRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo order type", &GridAlgoSubOrdersRequest{Limit: 2}, errInvalidAlgoOrderType},
		{"missing algo ID", &GridAlgoSubOrdersRequest{AlgoOrderType: AlgoOrdTypeGrid, Limit: 2}, errAlgoIDRequired},
		{"missing sub order type", &GridAlgoSubOrdersRequest{AlgoOrderType: AlgoOrdTypeGrid, AlgoID: "1234", Limit: 2}, errMissingSubOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetGridAlgoSubOrders(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetGridAlgoSubOrders should return the expected error")
		})
	}

	t.Run("query with every parameter", func(t *testing.T) {
		t.Parallel()
		exp := url.Values{"algoOrdType": {"grid"}, "algoId": {"1234"}, "type": {"live"}, "groupId": {"3"}, "after": {"449109084439187456"}, "before": {"449109084439187400"}, "limit": {"2"}}
		ex := newQueryPinExchange(t, "/tradingBot/grid/sub-orders", exp, tradingBotAlgoIDData)
		_, err := ex.GetGridAlgoSubOrders(t.Context(), &GridAlgoSubOrdersRequest{AlgoOrderType: AlgoOrdTypeGrid, AlgoID: "1234", SubOrderType: "live", GroupID: "3", After: "449109084439187456", Before: "449109084439187400", Limit: 2})
		require.NoError(t, err, "GetGridAlgoSubOrders must not error")
	})

	for _, tc := range []struct {
		name string
		arg  *GridAlgoSubOrdersRequest
		exp  []GridSubOrderResponse
	}{
		{
			name: "live spot grid sub orders",
			arg:  &GridAlgoSubOrdersRequest{AlgoOrderType: AlgoOrdTypeGrid, AlgoID: "1234", SubOrderType: "live", Limit: 2},
			exp: []GridSubOrderResponse{
				{
					AlgoID:         "1234",
					InstrumentType: "SPOT",
					InstrumentID:   "BTC-USDC",
					AlgoOrderType:  AlgoOrdTypeGrid,
					GroupID:        "3",
					OrderID:        "449109084439187456",
					CreationTime:   types.Time(time.UnixMilli(1653347949771)),
					UpdateTime:     types.Time(time.UnixMilli(1653347949831)),
					TradeMode:      TradeModeCash,
					OrderType:      orderLimit,
					Size:           0.00059213,
					State:          "live",
					Side:           order.Sell,
					Price:          30404.3,
					FeeCurrency:    currency.USDC,
					RebateCurrency: currency.USDT,
					PositionSide:   positionSideNet,
				},
			},
		},
		{
			name: "filled contract grid sub orders",
			arg:  &GridAlgoSubOrdersRequest{AlgoOrderType: AlgoOrdTypeContractGrid, AlgoID: "449327675342323712", SubOrderType: "filled"},
			exp: []GridSubOrderResponse{
				{
					AlgoID:              "449327675342323712",
					AlgoClientOrderID:   "gridClient449",
					InstrumentType:      "SWAP",
					InstrumentID:        "BTC-USDT-SWAP",
					AlgoOrderType:       AlgoOrdTypeContractGrid,
					GroupID:             "-1",
					OrderID:             "449518234142904321",
					CreationTime:        types.Time(time.UnixMilli(1653445498664)),
					UpdateTime:          types.Time(time.UnixMilli(1653445498674)),
					TradeMode:           TradeModeCross,
					Currency:            currency.USDT,
					OrderType:           orderLimit,
					Size:                1,
					State:               "filled",
					Side:                order.Buy,
					Price:               28007.2,
					Fee:                 -0.140036,
					FeeCurrency:         currency.USDT,
					Rebate:              0.0014,
					RebateCurrency:      currency.USDT,
					AveragePrice:        28007.2,
					AccumulatedFillSize: 1,
					PositionSide:        positionSideNet,
					PNL:                 1.52,
					ContractValue:       0.01,
					Leverage:            5,
					Tag:                 "gridTag449",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			result, err := e.GetGridAlgoSubOrders(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetGridAlgoSubOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetGridAlgoSubOrders should decode every field")
			} else {
				assert.NotNil(t, result, "GetGridAlgoSubOrders should return a list")
			}
		})
	}
}

func TestGetGridAlgoOrderPositions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *GridAlgoOrderRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"empty request", &GridAlgoOrderRequest{}, errInvalidAlgoOrderType},
		{"missing algo ID", &GridAlgoOrderRequest{AlgoOrderType: AlgoOrdTypeContractGrid}, errAlgoIDRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetGridAlgoOrderPositions(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetGridAlgoOrderPositions should return the expected error")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetGridAlgoOrderPositions(contextGenerate(), &GridAlgoOrderRequest{AlgoOrderType: AlgoOrdTypeContractGrid, AlgoID: "448965992920907776"})
	require.NoError(t, err, "GetGridAlgoOrderPositions must not error")
	if mockTests {
		exp := []TradingBotPositionResponse{
			{
				AlgoID:                       "448965992920907776",
				AlgoClientOrderID:            "gridClient448",
				InstrumentType:               "SWAP",
				InstrumentID:                 "BTC-USDT-SWAP",
				CreationTime:                 types.Time(time.UnixMilli(1653400065917)),
				UpdateTime:                   types.Time(time.UnixMilli(1653400066938)),
				AveragePrice:                 29215.0142857142857149,
				Currency:                     currency.USDT,
				Leverage:                     5,
				LiquidationPrice:             661.1684795867162,
				PositionSide:                 positionSideNet,
				Position:                     35,
				MarginMode:                   TradeModeCross,
				MarginRatio:                  217.19370606167573,
				InitialMarginRequirement:     2045.386,
				MaintenanceMarginRequirement: 40.907720000000005,
				UnrealisedPNL:                1.674999999999818,
				UnrealisedPNLRatio:           0.0008190504784478,
				LastTradedPrice:              29206.7,
				NotionalUSD:                  10216.70307,
				AutoDeleveraging:             1,
				MarkPrice:                    29213.9,
			},
		}
		assert.Equal(t, exp, result, "GetGridAlgoOrderPositions should decode every field")
	} else {
		assert.NotNil(t, result, "GetGridAlgoOrderPositions should return a list")
	}
}

func TestGridAlgoOrderParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *GridAlgoOrderRequest
		exp  url.Values
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing algo order type", arg: &GridAlgoOrderRequest{AlgoID: "448965992920907776"}, err: errInvalidAlgoOrderType},
		{name: "missing algo ID", arg: &GridAlgoOrderRequest{AlgoOrderType: AlgoOrdTypeGrid}, err: errAlgoIDRequired},
		{name: "spot grid", arg: &GridAlgoOrderRequest{AlgoOrderType: AlgoOrdTypeGrid, AlgoID: "448965992920907776"}, exp: url.Values{"algoOrdType": {"grid"}, "algoId": {"448965992920907776"}}},
		{name: "contract grid", arg: &GridAlgoOrderRequest{AlgoOrderType: AlgoOrdTypeContractGrid, AlgoID: "448965992920907777"}, exp: url.Values{"algoOrdType": {"contract_grid"}, "algoId": {"448965992920907777"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := gridAlgoOrderParams(tc.arg)
			require.ErrorIs(t, err, tc.err, "gridAlgoOrderParams must return the expected error")
			assert.Equal(t, tc.exp, params, "gridAlgoOrderParams should encode the algo order type and ID")
		})
	}
}

func TestSpotGridWithdrawProfit(t *testing.T) {
	t.Parallel()
	_, err := e.SpotGridWithdrawProfit(t.Context(), "")
	require.ErrorIs(t, err, errAlgoIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SpotGridWithdrawProfit(contextGenerate(), "1234")
	require.NoError(t, err, "SpotGridWithdrawProfit must not error")
	if mockTests {
		assert.Equal(t, &SpotGridWithdrawIncomeResponse{AlgoID: "1234", AlgoClientOrderID: "gridClient1234", Profit: 100}, result, "SpotGridWithdrawProfit should decode every field")
	} else {
		assert.NotEmpty(t, result.AlgoID, "SpotGridWithdrawProfit should return an algo ID")
	}
}

func TestComputeGridMarginBalance(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *ComputeGridMarginBalanceRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &ComputeGridMarginBalanceRequest{AdjustMarginBalanceType: "other"}, errAlgoIDRequired},
		{"missing adjustment type", &ComputeGridMarginBalanceRequest{AlgoID: "123456"}, errMarginBalanceTypeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.ComputeGridMarginBalance(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "ComputeGridMarginBalance should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *ComputeGridMarginBalanceRequest
			body string
		}{
			{"every parameter", &ComputeGridMarginBalanceRequest{AlgoID: "123456", AdjustMarginBalanceType: marginBalanceAdd, Amount: 10}, `{"algoId":"123456","type":"add","amt":"10"}`},
			{"required parameters only", &ComputeGridMarginBalanceRequest{AlgoID: "123456", AdjustMarginBalanceType: marginBalanceReduce}, `{"algoId":"123456","type":"reduce"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/grid/compute-margin-balance", tc.body, tradingBotAlgoIDData).ComputeGridMarginBalance(t.Context(), tc.arg)
				require.NoError(t, err, "ComputeGridMarginBalance must not error")
			})
		}
	})

	t.Run("compute margin balance", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		}
		result, err := e.ComputeGridMarginBalance(contextGenerate(), &ComputeGridMarginBalanceRequest{AlgoID: "123456", AdjustMarginBalanceType: marginBalanceReduce})
		require.NoError(t, err, "ComputeGridMarginBalance must not error")
		if mockTests {
			assert.Equal(t, &ComputeGridMarginBalanceResponse{MaxAmount: 1.8309562403342999, Leverage: 0.3877200981166066}, result, "ComputeGridMarginBalance should decode every field")
		} else {
			assert.NotNil(t, result, "ComputeGridMarginBalance should return a result")
		}
	})
}

func TestAdjustGridMarginBalance(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AdjustGridMarginBalanceRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"empty request", &AdjustGridMarginBalanceRequest{}, errAlgoIDRequired},
		{"missing algo ID", &AdjustGridMarginBalanceRequest{Amount: 12345}, errAlgoIDRequired},
		{"missing adjustment type", &AdjustGridMarginBalanceRequest{AlgoID: "1234", Amount: 12345}, errMarginBalanceTypeRequired},
		{"missing amount and percent", &AdjustGridMarginBalanceRequest{AlgoID: "1234", AdjustMarginBalanceType: marginBalanceReduce}, order.ErrAmountIsInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AdjustGridMarginBalance(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "AdjustGridMarginBalance should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *AdjustGridMarginBalanceRequest
			body string
		}{
			{"every parameter", &AdjustGridMarginBalanceRequest{AlgoID: "1234", AdjustMarginBalanceType: marginBalanceAdd, Amount: 10, Percent: 0.1}, `{"algoId":"1234","type":"add","amt":"10","percent":"0.1"}`},
			{"amount only", &AdjustGridMarginBalanceRequest{AlgoID: "1234", AdjustMarginBalanceType: marginBalanceReduce, Amount: 12345}, `{"algoId":"1234","type":"reduce","amt":"12345"}`},
			{"percent only", &AdjustGridMarginBalanceRequest{AlgoID: "1234", AdjustMarginBalanceType: marginBalanceReduce, Percent: 0.5}, `{"algoId":"1234","type":"reduce","percent":"0.5"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/grid/margin-balance", tc.body, tradingBotAlgoIDData).AdjustGridMarginBalance(t.Context(), tc.arg)
				require.NoError(t, err, "AdjustGridMarginBalance must not error")
			})
		}
	})

	t.Run("adjust margin balance", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.AdjustGridMarginBalance(contextGenerate(), &AdjustGridMarginBalanceRequest{AlgoID: "1234", AdjustMarginBalanceType: marginBalanceReduce, Amount: 12345})
		require.NoError(t, err, "AdjustGridMarginBalance must not error")
		if mockTests {
			assert.Equal(t, &AdjustGridMarginBalanceResponse{AlgoID: "1234", AlgoClientOrderID: "gridClient1234"}, result, "AdjustGridMarginBalance should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "AdjustGridMarginBalance should return an algo ID")
		}
	})
}

func TestAddGridInvestment(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AddGridInvestmentRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &AddGridInvestmentRequest{Amount: 12}, errAlgoIDRequired},
		{"missing amount", &AddGridInvestmentRequest{AlgoID: "448965992920907776"}, order.ErrAmountMustBeSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AddGridInvestment(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "AddGridInvestment should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *AddGridInvestmentRequest
			body string
		}{
			{"every parameter", &AddGridInvestmentRequest{AlgoID: "448965992920907776", Amount: 12, AllowReinvestProfit: new(false)}, `{"algoId":"448965992920907776","amt":"12","allowReinvestProfit":"false"}`},
			{"required parameters only", &AddGridInvestmentRequest{AlgoID: "448965992920907776", Amount: 12}, `{"algoId":"448965992920907776","amt":"12"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/grid/adjust-investment", tc.body, tradingBotAlgoIDData).AddGridInvestment(t.Context(), tc.arg)
				require.NoError(t, err, "AddGridInvestment must not error")
			})
		}
	})

	t.Run("add investment", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.AddGridInvestment(contextGenerate(), &AddGridInvestmentRequest{AlgoID: "448965992920907776", Amount: 12})
		require.NoError(t, err, "AddGridInvestment must not error")
		if mockTests {
			assert.Equal(t, &TradingBotAlgoIDResponse{AlgoID: "448965992920907776"}, result, "AddGridInvestment should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "AddGridInvestment should return an algo ID")
		}
	})
}

func TestGetGridAIParameter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *GridAIParameterRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo order type", &GridAIParameterRequest{InstrumentID: mainPair.String()}, errInvalidAlgoOrderType},
		{"missing instrument ID", &GridAIParameterRequest{AlgoOrderType: AlgoOrdTypeGrid}, errMissingInstrumentID},
		{"contract grid without direction", &GridAIParameterRequest{AlgoOrderType: AlgoOrdTypeContractGrid, InstrumentID: mainPair.String()}, errMissingRequiredArgumentDirection},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetGridAIParameter(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetGridAIParameter should return the expected error")
		})
	}

	t.Run("query with every parameter", func(t *testing.T) {
		t.Parallel()
		exp := url.Values{"algoOrdType": {"contract_grid"}, "instId": {"BTC-USDT-SWAP"}, "direction": {"long"}, "duration": {"30D"}}
		ex := newQueryPinExchange(t, "/tradingBot/grid/ai-param", exp, tradingBotAlgoIDData)
		_, err := ex.GetGridAIParameter(t.Context(), &GridAIParameterRequest{AlgoOrderType: AlgoOrdTypeContractGrid, InstrumentID: perpetualSwapPair.String(), Direction: positionSideLong, Duration: "30D"})
		require.NoError(t, err, "GetGridAIParameter must not error")
	})

	for _, tc := range []struct {
		name string
		arg  *GridAIParameterRequest
		exp  []GridAIParameterResponse
	}{
		{
			name: "spot grid",
			arg:  &GridAIParameterRequest{AlgoOrderType: AlgoOrdTypeGrid, InstrumentID: mainPair.String()},
			exp: []GridAIParameterResponse{
				{
					InstrumentID:       "BTC-USDT",
					AlgoOrderType:      AlgoOrdTypeGrid,
					Duration:           "7D",
					GridNumber:         84,
					MaxPrice:           72610.4,
					MinPrice:           49558.5,
					PerMaxProfitRate:   0.0039286752636984,
					PerMinProfitRate:   0.0021893669762316,
					PerGridProfitRatio: 4.386375616380676,
					AnnualisedRate:     0.2717,
					MinInvestment:      50.2,
					Currency:           currency.USDT,
					RunType:            "1",
				},
			},
		},
		{
			name: "contract grid",
			arg:  &GridAIParameterRequest{AlgoOrderType: AlgoOrdTypeContractGrid, InstrumentID: perpetualSwapPair.String(), Direction: positionSideLong},
			exp: []GridAIParameterResponse{
				{
					InstrumentID:     "BTC-USDT-SWAP",
					AlgoOrderType:    AlgoOrdTypeContractGrid,
					Duration:         "14D",
					GridNumber:       200,
					MaxPrice:         73101.2,
					MinPrice:         48734,
					PerMaxProfitRate: 0.0104834263696659,
					PerMinProfitRate: 0.0063368148142755,
					AnnualisedRate:   1.4978,
					MinInvestment:    377.92,
					Currency:         currency.USDT,
					RunType:          "1",
					Direction:        positionSideLong,
					Leverage:         5,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetGridAIParameter(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetGridAIParameter must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetGridAIParameter should decode every field")
			} else {
				require.NotEmpty(t, result, "GetGridAIParameter must return parameters")
				assert.Equal(t, tc.arg.InstrumentID, result[0].InstrumentID, "GetGridAIParameter should return the requested instrument")
				assert.Positive(t, result[0].GridNumber.Float64(), "GetGridAIParameter should return a grid number")
			}
		})
	}
}

func TestComputeMinInvestment(t *testing.T) {
	t.Parallel()
	contractGrid := func() *ComputeMinInvestmentRequest {
		return &ComputeMinInvestmentRequest{InstrumentID: mainPair.String(), AlgoOrderType: AlgoOrdTypeContractGrid, MaxPrice: 5000, MinPrice: 5000, GridNumber: 1234, RunType: "1", Direction: positionSideLong, Leverage: 5}
	}
	for _, tc := range []struct {
		name string
		arg  func() *ComputeMinInvestmentRequest
		err  error
	}{
		{"nil request", func() *ComputeMinInvestmentRequest { return nil }, common.ErrNilPointer},
		{"missing instrument ID", func() *ComputeMinInvestmentRequest { return &ComputeMinInvestmentRequest{RunType: "1"} }, errMissingInstrumentID},
		{"missing algo order type", func() *ComputeMinInvestmentRequest { a := contractGrid(); a.AlgoOrderType = ""; return a }, errInvalidAlgoOrderType},
		{"missing max price", func() *ComputeMinInvestmentRequest { a := contractGrid(); a.MaxPrice = 0; return a }, limits.ErrPriceBelowMin},
		{"missing min price", func() *ComputeMinInvestmentRequest { a := contractGrid(); a.MinPrice = 0; return a }, limits.ErrPriceBelowMin},
		{"missing grid number", func() *ComputeMinInvestmentRequest { a := contractGrid(); a.GridNumber = 0; return a }, errInvalidGridNumber},
		{"missing run type", func() *ComputeMinInvestmentRequest { a := contractGrid(); a.RunType = ""; return a }, errRunTypeRequired},
		{"contract grid without direction", func() *ComputeMinInvestmentRequest { a := contractGrid(); a.Direction = ""; return a }, errMissingRequiredArgumentDirection},
		{"contract grid without leverage", func() *ComputeMinInvestmentRequest { a := contractGrid(); a.Leverage = 0; return a }, errInvalidLeverage},
		{"investment without amount", func() *ComputeMinInvestmentRequest {
			a := contractGrid()
			a.InvestmentData = []InvestmentData{{Currency: currency.ETH}}
			return a
		}, limits.ErrAmountBelowMin},
		{"investment without currency", func() *ComputeMinInvestmentRequest {
			a := contractGrid()
			a.InvestmentData = []InvestmentData{{Amount: 0.01}}
			return a
		}, currency.ErrCurrencyCodeEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.ComputeMinInvestment(t.Context(), tc.arg())
			assert.ErrorIs(t, err, tc.err, "ComputeMinInvestment should return the expected error")
		})
	}

	for _, tc := range []struct {
		name string
		arg  *ComputeMinInvestmentRequest
		exp  *ComputeMinInvestmentResponse
	}{
		{
			name: "spot grid",
			arg: &ComputeMinInvestmentRequest{
				InstrumentID: mainPair.String(), AlgoOrderType: AlgoOrdTypeGrid, GridNumber: 50, MaxPrice: 5000, MinPrice: 3000, RunType: "1",
				InvestmentData: []InvestmentData{{Amount: 0.01, Currency: currency.BTC}, {Amount: 100, Currency: currency.USDT}},
			},
			exp: &ComputeMinInvestmentResponse{
				MinInvestmentData: []InvestmentData{{Amount: 10, Currency: currency.USDT}, {Amount: 0.000045, Currency: currency.BTC}},
				SingleAmount:      0.00376459,
			},
		},
		{
			name: "contract grid",
			arg:  &ComputeMinInvestmentRequest{InstrumentID: perpetualSwapPair.String(), AlgoOrderType: AlgoOrdTypeContractGrid, GridNumber: 50, MaxPrice: 72000, MinPrice: 50000, RunType: "1", Direction: positionSideLong, Leverage: 5, BasePosition: true},
			exp:  &ComputeMinInvestmentResponse{MinInvestmentData: []InvestmentData{{Amount: 99.0791, Currency: currency.USDT}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ComputeMinInvestment(contextGenerate(), tc.arg)
			require.NoError(t, err, "ComputeMinInvestment must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ComputeMinInvestment should decode every field")
			} else {
				assert.NotEmpty(t, result.MinInvestmentData, "ComputeMinInvestment should return the minimum investment")
			}
		})
	}
}

func TestRSIBackTesting(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *RSIBackTestingRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing instrument ID", &RSIBackTestingRequest{Threshold: 50, TimePeriod: 14, Timeframe: kline.FiveMin}, errMissingInstrumentID},
		{"missing timeframe", &RSIBackTestingRequest{InstrumentID: mainPair.String(), Threshold: 50, TimePeriod: 14}, errMissingIntervalValue},
		{"threshold below 1", &RSIBackTestingRequest{InstrumentID: mainPair.String(), TimePeriod: 14, Timeframe: kline.FiveMin}, errInvalidRSIThreshold},
		{"threshold above 100", &RSIBackTestingRequest{InstrumentID: mainPair.String(), Threshold: 101, TimePeriod: 14, Timeframe: kline.FiveMin}, errInvalidRSIThreshold},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.RSIBackTesting(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "RSIBackTesting should return the expected error")
		})
	}

	t.Run("query with every parameter", func(t *testing.T) {
		t.Parallel()
		exp := url.Values{"instId": {"BTC-USDT"}, "timeframe": {"1H"}, "thold": {"30"}, "timePeriod": {"14"}, "triggerCond": {"cross_up"}, "duration": {"1M"}}
		ex := newQueryPinExchange(t, "/tradingBot/public/rsi-back-testing", exp, []map[string]string{{"triggerNum": "1"}})
		_, err := ex.RSIBackTesting(t.Context(), &RSIBackTestingRequest{InstrumentID: mainPair.String(), Timeframe: kline.OneHour, Threshold: 30, TimePeriod: 14, TriggerCondition: "cross_up", Duration: "1M"})
		require.NoError(t, err, "RSIBackTesting must not error")
	})

	result, err := e.RSIBackTesting(contextGenerate(), &RSIBackTestingRequest{InstrumentID: mainPair.String(), Threshold: 50, TimePeriod: 14, Timeframe: kline.FiveMin})
	require.NoError(t, err, "RSIBackTesting must not error")
	if mockTests {
		assert.Equal(t, &RSIBackTestingResponse{TriggerNumber: 444}, result, "RSIBackTesting should decode every field")
	} else {
		assert.Positive(t, result.TriggerNumber.Float64(), "RSIBackTesting should return a trigger count")
	}
}

func TestGetMaxGridQuantity(t *testing.T) {
	t.Parallel()
	valid := func() *MaxGridQuantityRequest {
		return &MaxGridQuantityRequest{InstrumentID: perpetualSwapPair.String(), RunType: "1", AlgoOrderType: AlgoOrdTypeContractGrid, MaxPrice: 70000, MinPrice: 50000, Leverage: 5}
	}
	for _, tc := range []struct {
		name string
		arg  func() *MaxGridQuantityRequest
		err  error
	}{
		{"nil request", func() *MaxGridQuantityRequest { return nil }, common.ErrNilPointer},
		{"missing instrument ID", func() *MaxGridQuantityRequest { a := valid(); a.InstrumentID = ""; return a }, errMissingInstrumentID},
		{"missing run type", func() *MaxGridQuantityRequest { a := valid(); a.RunType = ""; return a }, errRunTypeRequired},
		{"missing algo order type", func() *MaxGridQuantityRequest { a := valid(); a.AlgoOrderType = ""; return a }, errInvalidAlgoOrderType},
		{"missing max price", func() *MaxGridQuantityRequest { a := valid(); a.MaxPrice = 0; return a }, limits.ErrPriceBelowMin},
		{"missing min price", func() *MaxGridQuantityRequest { a := valid(); a.MinPrice = 0; return a }, limits.ErrPriceBelowMin},
		{"contract grid without leverage", func() *MaxGridQuantityRequest { a := valid(); a.Leverage = 0; return a }, errInvalidLeverage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetMaxGridQuantity(t.Context(), tc.arg())
			assert.ErrorIs(t, err, tc.err, "GetMaxGridQuantity should return the expected error")
		})
	}

	t.Run("spot grid query without leverage", func(t *testing.T) {
		t.Parallel()
		exp := url.Values{"instId": {"BTC-USDT"}, "runType": {"2"}, "algoOrdType": {"grid"}, "maxPx": {"70000.5"}, "minPx": {"50000"}}
		ex := newQueryPinExchange(t, "/tradingBot/grid/grid-quantity", exp, []map[string]string{{"maxGridQty": "1"}})
		_, err := ex.GetMaxGridQuantity(t.Context(), &MaxGridQuantityRequest{InstrumentID: mainPair.String(), RunType: "2", AlgoOrderType: AlgoOrdTypeGrid, MaxPrice: 70000.5, MinPrice: 50000})
		require.NoError(t, err, "GetMaxGridQuantity must not error")
	})

	result, err := e.GetMaxGridQuantity(contextGenerate(), valid())
	require.NoError(t, err, "GetMaxGridQuantity must not error")
	if mockTests {
		assert.Equal(t, &MaxGridQuantityResponse{MaxGridQuantity: 714}, result, "GetMaxGridQuantity should decode every field")
	} else {
		assert.GreaterOrEqual(t, result.MaxGridQuantity.Float64(), 2.0, "GetMaxGridQuantity should return at least the minimum grid quantity")
	}
}

func TestCopyGridAlgoOrder(t *testing.T) {
	t.Parallel()
	valid := func() *CopyGridAlgoOrderRequest {
		return &CopyGridAlgoOrderRequest{InstrumentID: perpetualSwapPair.String(), AlgoOrderType: AlgoOrdTypeContractGrid, SourceAlgoID: "580007082221121536", Leverage: 3, AutoReserve: new(true), Size: 5000}
	}
	for _, tc := range []struct {
		name string
		arg  func() *CopyGridAlgoOrderRequest
		err  error
	}{
		{"nil request", func() *CopyGridAlgoOrderRequest { return nil }, common.ErrNilPointer},
		{"missing instrument ID", func() *CopyGridAlgoOrderRequest { a := valid(); a.InstrumentID = ""; return a }, errMissingInstrumentID},
		{"missing algo order type", func() *CopyGridAlgoOrderRequest { a := valid(); a.AlgoOrderType = ""; return a }, errInvalidAlgoOrderType},
		{"missing source algo ID", func() *CopyGridAlgoOrderRequest { a := valid(); a.SourceAlgoID = ""; return a }, errSourceAlgoIDRequired},
		{"automatic reserve without size", func() *CopyGridAlgoOrderRequest { a := valid(); a.Size = 0; return a }, order.ErrAmountMustBeSet},
		{"manual reserve without actual margin", func() *CopyGridAlgoOrderRequest { a := valid(); a.AutoReserve = new(false); return a }, order.ErrAmountMustBeSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CopyGridAlgoOrder(t.Context(), tc.arg())
			assert.ErrorIs(t, err, tc.err, "CopyGridAlgoOrder should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *CopyGridAlgoOrderRequest
			body string
		}{
			{
				name: "contract grid with every parameter",
				arg: &CopyGridAlgoOrderRequest{
					InstrumentID: perpetualSwapPair.String(), AlgoOrderType: AlgoOrdTypeContractGrid, SourceAlgoID: "580007082221121536", QuoteSize: 1000, Leverage: 3,
					AutoReserve: new(false), Size: 5000, ActualMarginSize: 4000, ExtraMarginSize: 1000, AlgoClientOrderID: "copyClient1", Tag: "copyTag1",
				},
				body: `{"instId":"BTC-USDT-SWAP","algoOrdType":"contract_grid","sourceAlgoId":"580007082221121536","quoteSz":"1000","lever":"3","autoReserve":false,"sz":"5000","actualMarginSz":"4000","extraMarginSz":"1000","algoClOrdId":"copyClient1","tag":"copyTag1"}`,
			},
			{"required parameters only", &CopyGridAlgoOrderRequest{InstrumentID: mainPair.String(), AlgoOrderType: AlgoOrdTypeGrid, SourceAlgoID: "580007082221121536"}, `{"instId":"BTC-USDT","algoOrdType":"grid","sourceAlgoId":"580007082221121536"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/grid/copy-order-algo", tc.body, tradingBotAlgoIDData).CopyGridAlgoOrder(t.Context(), tc.arg)
				require.NoError(t, err, "CopyGridAlgoOrder must not error")
			})
		}
	})

	t.Run("copy order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		arg := valid()
		arg.AlgoClientOrderID = "copyClient1"
		arg.Tag = "copyTag1"
		result, err := e.CopyGridAlgoOrder(contextGenerate(), arg)
		require.NoError(t, err, "CopyGridAlgoOrder must not error")
		if mockTests {
			assert.Equal(t, &TradingBotOrderResponse{AlgoID: "581234567890123456", AlgoClientOrderID: "copyClient1", Tag: "copyTag1"}, result, "CopyGridAlgoOrder should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "CopyGridAlgoOrder should return an algo ID")
		}
	})
}

func TestValidateTradingBotTriggerParameters(t *testing.T) {
	t.Parallel()
	start := TradingBotTriggerParameter{TriggerAction: "start", TriggerStrategy: "instant"}
	stop := TradingBotTriggerParameter{TriggerAction: "stop", TriggerStrategy: "price", TriggerPrice: 60000}
	for _, tc := range []struct {
		name   string
		params []TradingBotTriggerParameter
		err    error
	}{
		{name: "no parameters"},
		{name: "one parameter", params: []TradingBotTriggerParameter{start}},
		{name: "start and stop", params: []TradingBotTriggerParameter{start, stop}},
		{name: "missing trigger action", params: []TradingBotTriggerParameter{{TriggerStrategy: "instant"}}, err: errTriggerActionRequired},
		{name: "missing trigger strategy", params: []TradingBotTriggerParameter{{TriggerAction: "start"}}, err: errTriggerStrategyRequired},
		{name: "later parameter missing its action", params: []TradingBotTriggerParameter{start, {TriggerStrategy: "price"}}, err: errTriggerActionRequired},
		{name: "later parameter missing its strategy", params: []TradingBotTriggerParameter{start, {TriggerAction: "stop"}}, err: errTriggerStrategyRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validateTradingBotTriggerParameters(tc.params), tc.err, "validateTradingBotTriggerParameters should return the expected error")
		})
	}
}
