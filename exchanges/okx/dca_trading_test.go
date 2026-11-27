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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestPlaceDCAAlgoOrder(t *testing.T) {
	t.Parallel()
	valid := func() *PlaceDCAAlgoOrderRequest {
		return &PlaceDCAAlgoOrderRequest{
			InstrumentID: perpetualSwapPair.String(), AlgoOrderType: AlgoOrdTypeContractDCA, InitialOrderAmount: 50, MaxSafetyOrders: 2,
			SafetyOrderAmount: 10, PriceSteps: 0.01, PriceStepsMultiplier: 1, VolumeMultiplier: 1, TakeProfitPercentage: 0.05, Leverage: 2,
			TriggerParameters: []TradingBotTriggerParameter{{TriggerAction: "start", TriggerStrategy: "instant"}},
		}
	}
	for _, tc := range []struct {
		name string
		arg  func() *PlaceDCAAlgoOrderRequest
		err  error
	}{
		{"nil request", func() *PlaceDCAAlgoOrderRequest { return nil }, common.ErrNilPointer},
		{"empty request", func() *PlaceDCAAlgoOrderRequest { return &PlaceDCAAlgoOrderRequest{} }, errMissingInstrumentID},
		{"missing algo order type", func() *PlaceDCAAlgoOrderRequest { a := valid(); a.AlgoOrderType = ""; return a }, errInvalidAlgoOrderType},
		{"missing initial order amount", func() *PlaceDCAAlgoOrderRequest { a := valid(); a.InitialOrderAmount = 0; return a }, order.ErrAmountMustBeSet},
		{"safety orders without amount", func() *PlaceDCAAlgoOrderRequest { a := valid(); a.SafetyOrderAmount = 0; return a }, errSafetyOrderSettingRequired},
		{"safety orders without price steps", func() *PlaceDCAAlgoOrderRequest { a := valid(); a.PriceSteps = 0; return a }, errSafetyOrderSettingRequired},
		{"safety orders without price steps multiplier", func() *PlaceDCAAlgoOrderRequest { a := valid(); a.PriceStepsMultiplier = 0; return a }, errSafetyOrderSettingRequired},
		{"safety orders without volume multiplier", func() *PlaceDCAAlgoOrderRequest { a := valid(); a.VolumeMultiplier = 0; return a }, errSafetyOrderSettingRequired},
		{"missing take profit percentage", func() *PlaceDCAAlgoOrderRequest { a := valid(); a.TakeProfitPercentage = 0; return a }, errTakeProfitPercentageRequired},
		{"contract DCA without leverage", func() *PlaceDCAAlgoOrderRequest { a := valid(); a.Leverage = 0; return a }, errInvalidLeverage},
		{"missing trigger parameters", func() *PlaceDCAAlgoOrderRequest { a := valid(); a.TriggerParameters = nil; return a }, errTriggerParametersRequired},
		{"trigger without action", func() *PlaceDCAAlgoOrderRequest {
			a := valid()
			a.TriggerParameters = []TradingBotTriggerParameter{{TriggerStrategy: "instant"}}
			return a
		}, errTriggerActionRequired},
		{"trigger without strategy", func() *PlaceDCAAlgoOrderRequest {
			a := valid()
			a.TriggerParameters = []TradingBotTriggerParameter{{TriggerAction: "start"}}
			return a
		}, errTriggerStrategyRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.PlaceDCAAlgoOrder(t.Context(), tc.arg())
			assert.ErrorIs(t, err, tc.err, "PlaceDCAAlgoOrder should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *PlaceDCAAlgoOrderRequest
			body string
		}{
			{
				name: "contract DCA with every parameter",
				arg: &PlaceDCAAlgoOrderRequest{
					InstrumentID: perpetualSwapPair.String(), AlgoOrderType: AlgoOrdTypeContractDCA, InitialOrderAmount: 50, AllowReinvest: new(false),
					SafetyOrderAmount: 10, MaxSafetyOrders: 3, PriceSteps: 0.01, PriceStepsMultiplier: 1.5, VolumeMultiplier: 2, TakeProfitPercentage: 0.05,
					StopLossPercentage: 0.1, StopLossMode: "market", Direction: positionSideShort, Leverage: 2,
					TriggerParameters:  []TradingBotTriggerParameter{{TriggerAction: "start", TriggerStrategy: "price", TriggerPrice: 41000}},
					ProfitSharingRatio: 0.1, TrackingMode: "sync", Tag: "dcaTag1", AlgoClientOrderID: "dcaClient1",
				},
				body: `{"instId":"BTC-USDT-SWAP","algoOrdType":"contract_dca","initOrdAmt":"50","allowReinvest":false,"safetyOrdAmt":"10","maxSafetyOrds":"3","pxSteps":"0.01","pxStepsMult":"1.5","volMult":"2","tpPct":"0.05","slPct":"0.1","slMode":"market","direction":"short","lever":"2","triggerParams":[{"triggerAction":"start","triggerStrategy":"price","triggerPx":"41000"}],"profitSharingRatio":"0.1","trackingMode":"sync","tag":"dcaTag1","algoClOrdId":"dcaClient1"}`,
			},
			{
				name: "spot DCA with every parameter",
				arg: &PlaceDCAAlgoOrderRequest{
					InstrumentID: mainPair.String(), AlgoOrderType: AlgoOrdTypeSpotDCA, InitialOrderAmount: 100, SafetyOrderAmount: 50, MaxSafetyOrders: 5,
					PriceSteps: 0.02, PriceStepsMultiplier: 1, VolumeMultiplier: 1, TakeProfitPercentage: 0.03, StopLossPercentage: 0.2, StopLossMode: "limit",
					Direction: positionSideLong, TradeQuoteCurrency: currency.USDT,
					TriggerParameters: []TradingBotTriggerParameter{{TriggerAction: "start", TriggerStrategy: "rsi", Timeframe: "15m", Threshold: 30, TriggerCondition: "cross_down", TimePeriod: 14}},
				},
				body: `{"instId":"BTC-USDT","algoOrdType":"spot_dca","initOrdAmt":"100","safetyOrdAmt":"50","maxSafetyOrds":"5","pxSteps":"0.02","pxStepsMult":"1","volMult":"1","tpPct":"0.03","slPct":"0.2","slMode":"limit","direction":"long","triggerParams":[{"triggerAction":"start","triggerStrategy":"rsi","timeframe":"15m","thold":"30","triggerCond":"cross_down","timePeriod":"14"}],"tradeQuoteCcy":"USDT"}`,
			},
			{
				name: "required parameters only",
				arg: &PlaceDCAAlgoOrderRequest{
					InstrumentID: mainPair.String(), AlgoOrderType: AlgoOrdTypeSpotDCA, InitialOrderAmount: 100, TakeProfitPercentage: 0.03,
					TriggerParameters: []TradingBotTriggerParameter{{TriggerAction: "start", TriggerStrategy: "instant"}},
				},
				body: `{"instId":"BTC-USDT","algoOrdType":"spot_dca","initOrdAmt":"100","maxSafetyOrds":"0","tpPct":"0.03","triggerParams":[{"triggerAction":"start","triggerStrategy":"instant"}]}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/dca/create", tc.body, tradingBotAlgoIDData).PlaceDCAAlgoOrder(t.Context(), tc.arg)
				require.NoError(t, err, "PlaceDCAAlgoOrder must not error")
			})
		}
	})

	t.Run("place order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.PlaceDCAAlgoOrder(contextGenerate(), &PlaceDCAAlgoOrderRequest{
			InstrumentID: perpetualSwapPair.String(), AlgoOrderType: AlgoOrdTypeContractDCA, Direction: positionSideLong, Leverage: 2, InitialOrderAmount: 50,
			SafetyOrderAmount: 10, PriceSteps: 0.01, TakeProfitPercentage: 0.05, Tag: "dcaTag1", AlgoClientOrderID: "dcaClient1",
			TriggerParameters: []TradingBotTriggerParameter{{TriggerAction: "start", TriggerStrategy: "rsi", Timeframe: "30m", Threshold: 10, TriggerCondition: "cross", TimePeriod: 14}},
		})
		require.NoError(t, err, "PlaceDCAAlgoOrder must not error")
		if mockTests {
			assert.Equal(t, &TradingBotOrderResponse{AlgoID: "447053782921515008", AlgoClientOrderID: "dcaClient1", Tag: "dcaTag1"}, result, "PlaceDCAAlgoOrder should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "PlaceDCAAlgoOrder should return an algo ID")
		}
	})
}

func TestAmendSpotDCABasicParameters(t *testing.T) {
	t.Parallel()
	valid := func() *AmendSpotDCABasicParametersRequest {
		return &AmendSpotDCABasicParametersRequest{
			AlgoID: "532177187189760000", PriceSteps: 0.02, PriceStepsMultiplier: 2, VolumeMultiplier: 2, TakeProfitPercentage: 0.05, StopLossPercentage: 0.2,
			InitialOrderAmount: 100, SafetyOrderAmount: 50, MaxSafetyOrders: 5, ReserveFunds: true,
			TriggerParameters: []TradingBotTriggerParameter{{TriggerAction: "start", TriggerStrategy: "instant"}},
		}
	}
	for _, tc := range []struct {
		name string
		arg  func() *AmendSpotDCABasicParametersRequest
		err  error
	}{
		{"nil request", func() *AmendSpotDCABasicParametersRequest { return nil }, common.ErrNilPointer},
		{"missing algo ID", func() *AmendSpotDCABasicParametersRequest { a := valid(); a.AlgoID = ""; return a }, errAlgoIDRequired},
		{"missing safety order amount", func() *AmendSpotDCABasicParametersRequest { a := valid(); a.SafetyOrderAmount = 0; return a }, errSafetyOrderSettingRequired},
		{"missing price steps", func() *AmendSpotDCABasicParametersRequest { a := valid(); a.PriceSteps = 0; return a }, errSafetyOrderSettingRequired},
		{"missing price steps multiplier", func() *AmendSpotDCABasicParametersRequest { a := valid(); a.PriceStepsMultiplier = 0; return a }, errSafetyOrderSettingRequired},
		{"missing volume multiplier", func() *AmendSpotDCABasicParametersRequest { a := valid(); a.VolumeMultiplier = 0; return a }, errSafetyOrderSettingRequired},
		{"missing take profit percentage", func() *AmendSpotDCABasicParametersRequest { a := valid(); a.TakeProfitPercentage = 0; return a }, errTakeProfitPercentageRequired},
		{"missing stop loss percentage", func() *AmendSpotDCABasicParametersRequest { a := valid(); a.StopLossPercentage = 0; return a }, errStopLossPercentageRequired},
		{"missing initial order amount", func() *AmendSpotDCABasicParametersRequest { a := valid(); a.InitialOrderAmount = 0; return a }, order.ErrAmountMustBeSet},
		{"missing trigger parameters", func() *AmendSpotDCABasicParametersRequest { a := valid(); a.TriggerParameters = nil; return a }, errTriggerParametersRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AmendSpotDCABasicParameters(t.Context(), tc.arg())
			assert.ErrorIs(t, err, tc.err, "AmendSpotDCABasicParameters should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		arg := valid()
		arg.ReserveFunds = false
		arg.TriggerParameters = []TradingBotTriggerParameter{{TriggerAction: "start", TriggerStrategy: "rsi", Timeframe: "4H", Threshold: 25, TriggerCondition: "below", TimePeriod: 14}}
		ex := newRequestPinExchange(t, "/tradingBot/dca/amend-order-algo", `{"algoId":"532177187189760000","pxSteps":"0.02","pxStepsMult":"2","volMult":"2","tpPct":"0.05","slPct":"0.2","initOrdAmt":"100","safetyOrdAmt":"50","maxSafetyOrds":"5","reserveFunds":false,"triggerParams":[{"triggerAction":"start","triggerStrategy":"rsi","timeframe":"4H","thold":"25","triggerCond":"below","timePeriod":"14"}]}`, tradingBotAlgoIDData)
		_, err := ex.AmendSpotDCABasicParameters(t.Context(), arg)
		require.NoError(t, err, "AmendSpotDCABasicParameters must not error")
	})

	t.Run("amend order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.AmendSpotDCABasicParameters(contextGenerate(), valid())
		require.NoError(t, err, "AmendSpotDCABasicParameters must not error")
		if mockTests {
			assert.Equal(t, &TradingBotOrderResponse{AlgoID: "532177187189760000"}, result, "AmendSpotDCABasicParameters should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "AmendSpotDCABasicParameters should return an algo ID")
		}
	})
}

func TestValidateDCASafetyOrderSettings(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                                                                  string
		safetyOrderAmount, priceSteps, priceStepsMultiplier, volumeMultiplier float64
		missing                                                               string
	}{
		{name: "every setting", safetyOrderAmount: 10, priceSteps: 0.5, priceStepsMultiplier: 1.2, volumeMultiplier: 1.5},
		{name: "no settings", missing: "safety order amount"},
		{name: "no safety order amount", priceSteps: 0.5, priceStepsMultiplier: 1.2, volumeMultiplier: 1.5, missing: "safety order amount"},
		{name: "negative safety order amount", safetyOrderAmount: -10, priceSteps: 0.5, priceStepsMultiplier: 1.2, volumeMultiplier: 1.5, missing: "safety order amount"},
		{name: "no price steps", safetyOrderAmount: 10, priceStepsMultiplier: 1.2, volumeMultiplier: 1.5, missing: "price steps"},
		{name: "negative price steps", safetyOrderAmount: 10, priceSteps: -0.5, priceStepsMultiplier: 1.2, volumeMultiplier: 1.5, missing: "price steps"},
		{name: "no price steps multiplier", safetyOrderAmount: 10, priceSteps: 0.5, volumeMultiplier: 1.5, missing: "price steps multiplier"},
		{name: "negative price steps multiplier", safetyOrderAmount: 10, priceSteps: 0.5, priceStepsMultiplier: -1.2, volumeMultiplier: 1.5, missing: "price steps multiplier"},
		{name: "no volume multiplier", safetyOrderAmount: 10, priceSteps: 0.5, priceStepsMultiplier: 1.2, missing: "volume multiplier"},
		{name: "negative volume multiplier", safetyOrderAmount: 10, priceSteps: 0.5, priceStepsMultiplier: 1.2, volumeMultiplier: -1.5, missing: "volume multiplier"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateDCASafetyOrderSettings(tc.safetyOrderAmount, tc.priceSteps, tc.priceStepsMultiplier, tc.volumeMultiplier)
			if tc.missing == "" {
				assert.NoError(t, err, "validateDCASafetyOrderSettings should accept positive settings")
				return
			}
			require.ErrorIs(t, err, errSafetyOrderSettingRequired, "validateDCASafetyOrderSettings must reject a setting that is not positive")
			assert.EqualError(t, err, errSafetyOrderSettingRequired.Error()+": "+tc.missing, "the error should name the first setting that is not positive")
		})
	}
}

func TestStopDCAAlgoOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *StopDCAAlgoOrderRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &StopDCAAlgoOrderRequest{AlgoOrderType: AlgoOrdTypeContractDCA, StopType: "1"}, errAlgoIDRequired},
		{"missing algo order type", &StopDCAAlgoOrderRequest{AlgoID: "448965992920907776", StopType: "1"}, errInvalidAlgoOrderType},
		{"missing stop type", &StopDCAAlgoOrderRequest{AlgoID: "448965992920907776", AlgoOrderType: AlgoOrdTypeContractDCA}, errStopTypeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.StopDCAAlgoOrder(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "StopDCAAlgoOrder should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/tradingBot/dca/stop", `{"algoId":"448965992920907776","algoOrdType":"spot_dca","stopType":"2"}`, tradingBotAlgoIDData)
		_, err := ex.StopDCAAlgoOrder(t.Context(), &StopDCAAlgoOrderRequest{AlgoID: "448965992920907776", AlgoOrderType: AlgoOrdTypeSpotDCA, StopType: "2"})
		require.NoError(t, err, "StopDCAAlgoOrder must not error")
	})

	t.Run("stop order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.StopDCAAlgoOrder(contextGenerate(), &StopDCAAlgoOrderRequest{AlgoID: "448965992920907776", AlgoOrderType: AlgoOrdTypeContractDCA, StopType: "1"})
		require.NoError(t, err, "StopDCAAlgoOrder must not error")
		if mockTests {
			assert.Equal(t, &TradingBotOrderResponse{AlgoID: "448965992920907776"}, result, "StopDCAAlgoOrder should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "StopDCAAlgoOrder should return an algo ID")
		}
	})
}

func TestGetDCAAlgoOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetDCAAlgoOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetDCAAlgoOrderList(t.Context(), &DCAAlgoOrderListRequest{Limit: 20})
	require.ErrorIs(t, err, errInvalidAlgoOrderType)

	for _, tc := range []struct {
		name string
		arg  *DCAAlgoOrderListRequest
		exp  []DCAAlgoOrderResponse
	}{
		{
			name: "contract DCA",
			arg:  &DCAAlgoOrderListRequest{AlgoOrderType: AlgoOrdTypeContractDCA, Limit: 20},
			exp: []DCAAlgoOrderResponse{
				{
					AlgoID:               "565849588675117056",
					AlgoOrderType:        AlgoOrdTypeContractDCA,
					InstrumentID:         "BTC-USDT-SWAP",
					CopyType:             "0",
					State:                "running",
					Direction:            positionSideLong,
					Leverage:             3,
					InitialOrderAmount:   100,
					SafetyOrderAmount:    200,
					MaxSafetyOrders:      5,
					PriceSteps:           0.02,
					PriceStepsMultiplier: 1,
					VolumeMultiplier:     1,
					TakeProfitPriceRange: 45000,
					StopLossPercentage:   0.05,
					StopLossMode:         orderMarket,
					AllowReinvest:        true,
					TotalPNL:             12.5,
					PNLRatio:             0.05,
					TotalFundingFee:      -0.5,
					InvestmentAmount:     500,
					InvestmentCurrency:   currency.USDT,
					ArbitragePNL:         2.1,
					TransferInMargin:     500,
					TriggerParameters:    []TradingBotTriggerParameter{{TriggerAction: "start", TriggerStrategy: "instant"}},
					CreationTime:         types.Time(time.UnixMilli(1597026383085)),
					UpdateTime:           types.Time(time.UnixMilli(1597026383085)),
					ContractValue:        0.01,
				},
			},
		},
		{
			name: "spot DCA",
			arg:  &DCAAlgoOrderListRequest{AlgoOrderType: AlgoOrdTypeSpotDCA},
			exp: []DCAAlgoOrderResponse{
				{
					AlgoID:               "565849588675117057",
					AlgoOrderType:        AlgoOrdTypeSpotDCA,
					InstrumentID:         "BTC-USDT",
					CopyType:             "0",
					State:                "running",
					Direction:            positionSideLong,
					InitialOrderAmount:   100,
					SafetyOrderAmount:    200,
					MaxSafetyOrders:      5,
					PriceSteps:           0.02,
					PriceStepsMultiplier: 1.5,
					VolumeMultiplier:     2,
					StopLossPercentage:   0.1,
					StopLossMode:         orderLimit,
					TotalPNL:             3.2,
					PNLRatio:             0.0064,
					InvestmentAmount:     500,
					InvestmentCurrency:   currency.USDC,
					ArbitragePNL:         1.4,
					TriggerParameters:    []TradingBotTriggerParameter{{TriggerAction: "start", TriggerStrategy: "rsi", Timeframe: "1H", Threshold: 30, TriggerCondition: "cross_down", TimePeriod: 14}},
					CreationTime:         types.Time(time.UnixMilli(1597026383085)),
					UpdateTime:           types.Time(time.UnixMilli(1597026390085)),
					TradeQuoteCurrency:   currency.USDT,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			result, err := e.GetDCAAlgoOrderList(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetDCAAlgoOrderList must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetDCAAlgoOrderList should decode every field")
			} else {
				assert.NotNil(t, result, "GetDCAAlgoOrderList should return a list")
			}
		})
	}
}

func TestGetDCAAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDCAAlgoOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetDCAAlgoOrderHistory(t.Context(), &DCAAlgoOrderListRequest{})
	require.ErrorIs(t, err, errInvalidAlgoOrderType)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDCAAlgoOrderHistory(contextGenerate(), &DCAAlgoOrderListRequest{AlgoOrderType: AlgoOrdTypeContractDCA})
	require.NoError(t, err, "GetDCAAlgoOrderHistory must not error")
	if mockTests {
		exp := []DCAAlgoOrderResponse{
			{
				AlgoID:               "12345689",
				AlgoOrderType:        AlgoOrdTypeContractDCA,
				InstrumentID:         "BTC-USDT-SWAP",
				CopyType:             "3",
				State:                "stopped",
				CancelType:           "1",
				Direction:            positionSideLong,
				Leverage:             3,
				InitialOrderAmount:   100,
				SafetyOrderAmount:    200,
				MaxSafetyOrders:      5,
				PriceSteps:           0.02,
				PriceStepsMultiplier: 1,
				VolumeMultiplier:     1,
				AllowReinvest:        true,
				TotalPNL:             12.5,
				PNLRatio:             0.05,
				FundingFee:           -0.5,
				InvestmentAmount:     500,
				InvestmentCurrency:   currency.USDT,
				ArbitragePNL:         2.1,
				TransferInMargin:     500,
				ProfitSharingRatio:   0.1,
				TrackingMode:         "sync",
				TriggerParameters:    []TradingBotTriggerParameter{{TriggerAction: "start", TriggerStrategy: "price", TriggerPrice: 41000}},
				CreationTime:         types.Time(time.UnixMilli(1597026383085)),
				UpdateTime:           types.Time(time.UnixMilli(1597026383085)),
				ContractValue:        0.01,
			},
		}
		assert.Equal(t, exp, result, "GetDCAAlgoOrderHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetDCAAlgoOrderHistory should return a list")
	}
}

func TestDCAAlgoOrderListParams(t *testing.T) {
	t.Parallel()
	_, err := dcaAlgoOrderListParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = dcaAlgoOrderListParams(&DCAAlgoOrderListRequest{})
	require.ErrorIs(t, err, errInvalidAlgoOrderType)

	params, err := dcaAlgoOrderListParams(&DCAAlgoOrderListRequest{AlgoOrderType: AlgoOrdTypeSpotDCA, AlgoID: "565849588675117057", After: "565849588675117057", Before: "565849588675117000", Limit: 100})
	require.NoError(t, err, "dcaAlgoOrderListParams must not error")
	exp := url.Values{"algoOrdType": {"spot_dca"}, "algoId": {"565849588675117057"}, "after": {"565849588675117057"}, "before": {"565849588675117000"}, "limit": {"100"}}
	assert.Equal(t, exp, params, "dcaAlgoOrderListParams should set every parameter")
}

func TestGetDCASubOrders(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *DCASubOrdersRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &DCASubOrdersRequest{AlgoOrderType: AlgoOrdTypeContractDCA}, errAlgoIDRequired},
		{"missing algo order type", &DCASubOrdersRequest{AlgoID: "2833925189933756416"}, errInvalidAlgoOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetDCASubOrders(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetDCASubOrders should return the expected error")
		})
	}

	t.Run("query with every parameter", func(t *testing.T) {
		t.Parallel()
		exp := url.Values{"algoId": {"2833925189933756416"}, "algoOrdType": {"contract_dca"}, "cycleId": {"9876543"}, "after": {"570627699870375936"}, "before": {"570627699870375900"}, "limit": {"50"}}
		ex := newQueryPinExchange(t, "/tradingBot/dca/orders", exp, tradingBotAlgoIDData)
		_, err := ex.GetDCASubOrders(t.Context(), &DCASubOrdersRequest{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA, CycleID: "9876543", After: "570627699870375936", Before: "570627699870375900", Limit: 50})
		require.NoError(t, err, "GetDCASubOrders must not error")
	})

	for _, tc := range []struct {
		name string
		arg  *DCASubOrdersRequest
		exp  []DCASubOrderResponse
	}{
		{
			name: "contract DCA",
			arg:  &DCASubOrdersRequest{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA},
			exp: []DCASubOrderResponse{
				{
					CycleID:          "9876543",
					OrderID:          "570627699870375936",
					AverageFillPrice: 41500,
					Direction:        positionSideLong,
					Side:             "buy",
					OrderType:        "init_order",
					Price:            41000,
					Size:             10,
					FilledSize:       10,
					State:            "filled",
					Fee:              -0.2,
					Rebate:           0.01,
					RebateCurrency:   currency.USDT,
					Leverage:         3,
					InstrumentID:     "BTC-USDT-SWAP",
					ContractValue:    0.01,
					FillTime:         types.Time(time.UnixMilli(1597026383085)),
					CreationTime:     types.Time(time.UnixMilli(1597026383085)),
					UpdateTime:       types.Time(time.UnixMilli(1597026383085)),
				},
			},
		},
		{
			name: "spot DCA cycle",
			arg:  &DCASubOrdersRequest{AlgoID: "2833925189933756417", AlgoOrderType: AlgoOrdTypeSpotDCA, CycleID: "9876544"},
			exp: []DCASubOrderResponse{
				{
					CycleID:            "9876544",
					OrderID:            "570627699870375937",
					AverageFillPrice:   40180,
					Direction:          positionSideLong,
					Side:               "buy",
					OrderType:          "safety_order",
					Price:              40180,
					Size:               0.0025,
					FilledSize:         0.0025,
					State:              "filled",
					Fee:                -0.0000025,
					RebateCurrency:     currency.BTC,
					InstrumentID:       "BTC-USDT",
					FillTime:           types.Time(time.UnixMilli(1597026390085)),
					CreationTime:       types.Time(time.UnixMilli(1597026389085)),
					UpdateTime:         types.Time(time.UnixMilli(1597026390085)),
					TradeQuoteCurrency: currency.USDT,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			result, err := e.GetDCASubOrders(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetDCASubOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetDCASubOrders should decode every field")
			} else {
				assert.NotNil(t, result, "GetDCASubOrders should return a list")
			}
		})
	}
}

func TestAddDCAInvestment(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AddDCAInvestmentRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &AddDCAInvestmentRequest{AlgoOrderType: AlgoOrdTypeContractDCA, Price: 41000, Amount: 100}, errAlgoIDRequired},
		{"missing algo order type", &AddDCAInvestmentRequest{AlgoID: "2833925189933756416", Price: 41000, Amount: 100}, errInvalidAlgoOrderType},
		{"missing price", &AddDCAInvestmentRequest{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA, Amount: 100}, limits.ErrPriceBelowMin},
		{"missing amount", &AddDCAInvestmentRequest{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA, Price: 41000}, order.ErrAmountMustBeSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AddDCAInvestment(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "AddDCAInvestment should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *AddDCAInvestmentRequest
			body string
		}{
			{"every parameter", &AddDCAInvestmentRequest{AlgoID: "2833925189933756417", AlgoOrderType: AlgoOrdTypeSpotDCA, Price: 41000, Amount: 100, OrderType: orderMarket, TradeQuoteCurrency: currency.USDT}, `{"algoId":"2833925189933756417","algoOrdType":"spot_dca","price":"41000","amt":"100","ordType":"market","tradeQuoteCcy":"USDT"}`},
			{"required parameters only", &AddDCAInvestmentRequest{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA, Price: 41000, Amount: 100}, `{"algoId":"2833925189933756416","algoOrdType":"contract_dca","price":"41000","amt":"100"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/dca/orders/manual-buy", tc.body, tradingBotAlgoIDData).AddDCAInvestment(t.Context(), tc.arg)
				require.NoError(t, err, "AddDCAInvestment must not error")
			})
		}
	})

	t.Run("rejected investment returns the row", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"code":"1","msg":"Operation failed.","data":[{"algoId":"2833925189933756416","algoClOrdId":"","algoOrdType":"contract_dca","tag":"","diffAmount":"","sCode":"51008","sMsg":"Order failed. Insufficient USDT balance in account."}]}`))
		}))
		result, err := ex.AddDCAInvestment(t.Context(), &AddDCAInvestmentRequest{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA, Price: 41000, Amount: 100})
		require.ErrorContains(t, err, "Operation failed.", "AddDCAInvestment must report OKX's error")
		exp := &AddDCAInvestmentResponse{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA, StatusCode: 51008, StatusMessage: "Order failed. Insufficient USDT balance in account."}
		assert.Equal(t, exp, result, "AddDCAInvestment should return the rejected row beside the error")
	})

	t.Run("add investment", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.AddDCAInvestment(contextGenerate(), &AddDCAInvestmentRequest{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA, Price: 41000, Amount: 100})
		require.NoError(t, err, "AddDCAInvestment must not error")
		if mockTests {
			exp := &AddDCAInvestmentResponse{AlgoID: "2833925189933756416", AlgoClientOrderID: "dcaClient2", AlgoOrderType: AlgoOrdTypeContractDCA, Tag: "dcaTag2", DiffAmount: 100}
			assert.Equal(t, exp, result, "AddDCAInvestment should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "AddDCAInvestment should return an algo ID")
		}
	})
}

func TestAmendDCAReinvestment(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AmendDCAReinvestmentRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &AmendDCAReinvestmentRequest{AlgoOrderType: AlgoOrdTypeContractDCA}, errAlgoIDRequired},
		{"missing algo order type", &AmendDCAReinvestmentRequest{AlgoID: "2833925189933756416"}, errInvalidAlgoOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AmendDCAReinvestment(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "AmendDCAReinvestment should return the expected error")
		})
	}

	t.Run("request body disabling reinvestment", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/tradingBot/dca/settings/reinvestment", `{"algoId":"2833925189933756416","algoOrdType":"contract_dca","allowReinvest":false}`, tradingBotAlgoIDData)
		_, err := ex.AmendDCAReinvestment(t.Context(), &AmendDCAReinvestmentRequest{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA})
		require.NoError(t, err, "AmendDCAReinvestment must not error")
	})

	t.Run("amend reinvestment", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.AmendDCAReinvestment(contextGenerate(), &AmendDCAReinvestmentRequest{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA, AllowReinvest: true})
		require.NoError(t, err, "AmendDCAReinvestment must not error")
		if mockTests {
			assert.Equal(t, &DCASettingResponse{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA}, result, "AmendDCAReinvestment should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "AmendDCAReinvestment should return an algo ID")
		}
	})
}

func TestAmendDCATakeProfit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AmendDCATakeProfitRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &AmendDCATakeProfitRequest{AlgoOrderType: AlgoOrdTypeContractDCA, TakeProfitPrice: 43500}, errAlgoIDRequired},
		{"missing algo order type", &AmendDCATakeProfitRequest{AlgoID: "2833925189933756416", TakeProfitPrice: 43500}, errInvalidAlgoOrderType},
		{"missing take profit price", &AmendDCATakeProfitRequest{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA}, limits.ErrPriceBelowMin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AmendDCATakeProfit(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "AmendDCATakeProfit should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/tradingBot/dca/settings/take-profit", `{"algoId":"2833925189933756416","algoOrdType":"contract_dca","tpPrice":"43500.5"}`, tradingBotAlgoIDData)
		_, err := ex.AmendDCATakeProfit(t.Context(), &AmendDCATakeProfitRequest{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA, TakeProfitPrice: 43500.5})
		require.NoError(t, err, "AmendDCATakeProfit must not error")
	})

	t.Run("rejected amendment returns the row", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"code":"1","msg":"Operation failed.","data":[{"algoId":"2833925189933756416","algoOrdType":"contract_dca","sCode":"51000","sMsg":"Parameter tpPrice error"}]}`))
		}))
		result, err := ex.AmendDCATakeProfit(t.Context(), &AmendDCATakeProfitRequest{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA, TakeProfitPrice: 43500})
		require.ErrorContains(t, err, "Operation failed.", "AmendDCATakeProfit must report OKX's error")
		exp := &DCASettingResponse{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA, StatusCode: 51000, StatusMessage: "Parameter tpPrice error"}
		assert.Equal(t, exp, result, "AmendDCATakeProfit should return the rejected row beside the error")
	})

	t.Run("amend take profit", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.AmendDCATakeProfit(contextGenerate(), &AmendDCATakeProfitRequest{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA, TakeProfitPrice: 43500})
		require.NoError(t, err, "AmendDCATakeProfit must not error")
		if mockTests {
			assert.Equal(t, &DCASettingResponse{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA}, result, "AmendDCATakeProfit should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "AmendDCATakeProfit should return an algo ID")
		}
	})
}

func TestGetDCAPositionDetails(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *DCAAlgoOrderRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &DCAAlgoOrderRequest{AlgoOrderType: AlgoOrdTypeContractDCA}, errAlgoIDRequired},
		{"missing algo order type", &DCAAlgoOrderRequest{AlgoID: "2833925189933756416"}, errInvalidAlgoOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetDCAPositionDetails(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetDCAPositionDetails should return the expected error")
		})
	}

	for _, tc := range []struct {
		name string
		arg  *DCAAlgoOrderRequest
		exp  *DCAPositionDetailsResponse
	}{
		{
			name: "contract DCA",
			arg:  &DCAAlgoOrderRequest{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA},
			exp: &DCAPositionDetailsResponse{
				AlgoID:             "2833925189933756416",
				AlgoClientOrderID:  "dcaClient3",
				AlgoOrderType:      AlgoOrdTypeContractDCA,
				InstrumentID:       "BTC-USDT-SWAP",
				CurrentCycleID:     "3",
				StartTime:          types.Time(time.UnixMilli(1597026383085)),
				FilledManualOrders: 1,
				FilledSafetyOrders: 2,
				FundingFee:         -0.05,
				InitialPrice:       43200,
				NotionalUSD:        5000,
				AveragePrice:       43000,
				UnrealisedPNL:      12.5,
				LiquidationPrice:   38000,
				Size:               2,
				StopLossPrice:      40000,
				TakeProfitPrice:    45000,
				Fee:                -0.2,
			},
		},
		{
			name: "spot DCA",
			arg:  &DCAAlgoOrderRequest{AlgoID: "2833925189933756417", AlgoOrderType: AlgoOrdTypeSpotDCA},
			exp: &DCAPositionDetailsResponse{
				AlgoID:             "2833925189933756417",
				AlgoOrderType:      AlgoOrdTypeSpotDCA,
				InstrumentID:       "BTC-USDT",
				CurrentCycleID:     "5",
				StartTime:          types.Time(time.UnixMilli(1597026383085)),
				FilledSafetyOrders: 1,
				InitialPrice:       43200,
				AveragePrice:       42600,
				UnrealisedPNL:      -0.8,
				BaseSize:           0.0047,
				QuoteSize:          99.8,
				StopLossPrice:      38340,
				TakeProfitPrice:    44730,
				Fee:                -0.0000047,
				TradeQuoteCurrency: currency.USDT,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			result, err := e.GetDCAPositionDetails(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetDCAPositionDetails must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetDCAPositionDetails should decode every field")
			} else {
				assert.NotEmpty(t, result.AlgoID, "GetDCAPositionDetails should return an algo ID")
			}
		})
	}
}

func TestGetDCACycleList(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *DCACycleListRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &DCACycleListRequest{AlgoOrderType: AlgoOrdTypeContractDCA}, errAlgoIDRequired},
		{"missing algo order type", &DCACycleListRequest{AlgoID: "2833925189933756416"}, errInvalidAlgoOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetDCACycleList(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetDCACycleList should return the expected error")
		})
	}

	t.Run("query with every parameter", func(t *testing.T) {
		t.Parallel()
		exp := url.Values{"algoId": {"2833925189933756416"}, "algoOrdType": {"contract_dca"}, "instId": {"BTC-USDT-SWAP"}, "after": {"9876543"}, "before": {"9876500"}, "limit": {"10"}}
		ex := newQueryPinExchange(t, "/tradingBot/dca/cycle-list", exp, tradingBotAlgoIDData)
		_, err := ex.GetDCACycleList(t.Context(), &DCACycleListRequest{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA, InstrumentID: perpetualSwapPair.String(), After: "9876543", Before: "9876500", Limit: 10})
		require.NoError(t, err, "GetDCACycleList must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDCACycleList(contextGenerate(), &DCACycleListRequest{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA})
	require.NoError(t, err, "GetDCACycleList must not error")
	if mockTests {
		exp := []DCACycleResponse{
			{
				AlgoID:          "2833925189933756416",
				CycleID:         "9876543",
				CurrentCycle:    true,
				RealisedPNL:     12.5,
				StartTime:       types.Time(time.UnixMilli(1597026383085)),
				Fee:             -0.3,
				AveragePrice:    41500,
				TakeProfitPrice: 43000,
			},
			{
				AlgoID:            "2833925189933756416",
				AlgoClientOrderID: "dcaClient4",
				CycleID:           "9876542",
				RealisedPNL:       8.75,
				StartTime:         types.Time(time.UnixMilli(1596939983085)),
				EndTime:           types.Time(time.UnixMilli(1597026383080)),
				Fee:               -0.25,
				AveragePrice:      41200,
				TakeProfitPrice:   42400,
			},
		}
		assert.Equal(t, exp, result, "GetDCACycleList should decode every field")
	} else {
		assert.NotNil(t, result, "GetDCACycleList should return a list")
	}
}

func TestAddDCAMargin(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *DCAMarginRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &DCAMarginRequest{Amount: 50}, errAlgoIDRequired},
		{"missing amount", &DCAMarginRequest{AlgoID: "2833925189933756416"}, order.ErrAmountMustBeSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AddDCAMargin(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "AddDCAMargin should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/tradingBot/dca/margin/add", `{"algoId":"2833925189933756416","amt":"50.5"}`, tradingBotAlgoIDData)
		_, err := ex.AddDCAMargin(t.Context(), &DCAMarginRequest{AlgoID: "2833925189933756416", Amount: 50.5})
		require.NoError(t, err, "AddDCAMargin must not error")
	})

	t.Run("add margin", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.AddDCAMargin(contextGenerate(), &DCAMarginRequest{AlgoID: "2833925189933756416", Amount: 50})
		require.NoError(t, err, "AddDCAMargin must not error")
		if mockTests {
			assert.Equal(t, &DCASettingResponse{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA}, result, "AddDCAMargin should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "AddDCAMargin should return an algo ID")
		}
	})
}

func TestReduceDCAMargin(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *DCAMarginRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &DCAMarginRequest{Amount: 50}, errAlgoIDRequired},
		{"missing amount", &DCAMarginRequest{AlgoID: "2833925189933756416"}, order.ErrAmountMustBeSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.ReduceDCAMargin(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "ReduceDCAMargin should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/tradingBot/dca/margin/reduce", `{"algoId":"2833925189933756416","amt":"50.5"}`, tradingBotAlgoIDData)
		_, err := ex.ReduceDCAMargin(t.Context(), &DCAMarginRequest{AlgoID: "2833925189933756416", Amount: 50.5})
		require.NoError(t, err, "ReduceDCAMargin must not error")
	})

	t.Run("reduce margin", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.ReduceDCAMargin(contextGenerate(), &DCAMarginRequest{AlgoID: "2833925189933756416", Amount: 50})
		require.NoError(t, err, "ReduceDCAMargin must not error")
		if mockTests {
			assert.Equal(t, &DCASettingResponse{AlgoID: "2833925189933756416", AlgoOrderType: AlgoOrdTypeContractDCA}, result, "ReduceDCAMargin should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "ReduceDCAMargin should return an algo ID")
		}
	})
}

func TestValidateDCAMarginRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *DCAMarginRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing algo ID", arg: &DCAMarginRequest{Amount: 10}, err: errAlgoIDRequired},
		{name: "missing amount", arg: &DCAMarginRequest{AlgoID: "1"}, err: order.ErrAmountMustBeSet},
		{name: "negative amount", arg: &DCAMarginRequest{AlgoID: "1", Amount: -10}, err: order.ErrAmountMustBeSet},
		{name: "valid", arg: &DCAMarginRequest{AlgoID: "1", Amount: 10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validateDCAMarginRequest(tc.arg), tc.err, "validateDCAMarginRequest should return the expected error")
		})
	}
}
