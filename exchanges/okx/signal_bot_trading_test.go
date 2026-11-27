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

func TestCreateSignal(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSignal(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.CreateSignal(t.Context(), &CreateSignalRequest{SignalChannelDescription: "this is the first version"})
	require.ErrorIs(t, err, errSignalChannelNameRequired)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *CreateSignalRequest
			body string
		}{
			{"every parameter", &CreateSignalRequest{SignalChannelName: "long short", SignalChannelDescription: "this is the first version"}, `{"signalChanName":"long short","signalChanDesc":"this is the first version"}`},
			{"required parameters only", &CreateSignalRequest{SignalChannelName: "long short"}, `{"signalChanName":"long short"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/signal/create-signal", tc.body, []map[string]string{{"signalChanId": "1"}}).CreateSignal(t.Context(), tc.arg)
				require.NoError(t, err, "CreateSignal must not error")
			})
		}
	})

	t.Run("create signal", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.CreateSignal(contextGenerate(), &CreateSignalRequest{SignalChannelName: "long short", SignalChannelDescription: "this is the first version"})
		require.NoError(t, err, "CreateSignal must not error")
		if mockTests {
			assert.Equal(t, &CreateSignalResponse{SignalChannelID: "572112109", SignalChannelToken: "dojuckew331lkx"}, result, "CreateSignal should decode every field")
		} else {
			assert.NotEmpty(t, result.SignalChannelID, "CreateSignal should return a signal channel ID")
		}
	})
}

func TestGetSignals(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *SignalsRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing signal source type", &SignalsRequest{SignalChannelID: "623833708424069120"}, errSignalSourceTypeRequired},
		{"before later than after", &SignalsRequest{SignalSourceType: "1", After: time.UnixMilli(1695005546360), Before: time.UnixMilli(1697029422313)}, common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSignals(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetSignals should return the expected error")
		})
	}

	t.Run("query with every parameter", func(t *testing.T) {
		t.Parallel()
		exp := url.Values{"signalSourceType": {"2"}, "signalChanId": {"623833708424069120"}, "after": {"1697029422313"}, "before": {"1695005546360"}, "limit": {"10"}}
		ex := newQueryPinExchange(t, "/tradingBot/signal/signals", exp, []map[string]string{{"signalChanId": "1"}})
		_, err := ex.GetSignals(t.Context(), &SignalsRequest{SignalSourceType: "2", SignalChannelID: "623833708424069120", After: time.UnixMilli(1697029422313), Before: time.UnixMilli(1695005546360), Limit: 10})
		require.NoError(t, err, "GetSignals must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSignals(contextGenerate(), &SignalsRequest{SignalSourceType: "1"})
	require.NoError(t, err, "GetSignals must not error")
	if mockTests {
		exp := []SignalResponse{{SignalChannelID: "623833708424069120", SignalChannelName: "test", SignalChannelDescription: "test", SignalChannelToken: "test", SignalSourceType: "1"}}
		assert.Equal(t, exp, result, "GetSignals should decode every field")
	} else {
		assert.NotNil(t, result, "GetSignals should return a list")
	}
}

func TestCreateSignalBot(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CreateSignalBotRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing signal channel ID", &CreateSignalBotRequest{Leverage: 10, InvestmentAmount: 100, SubOrderType: "9"}, errSignalChannelIDRequired},
		{"missing leverage", &CreateSignalBotRequest{SignalChannelID: "627921182788161536", InvestmentAmount: 100, SubOrderType: "9"}, errInvalidLeverage},
		{"missing investment amount", &CreateSignalBotRequest{SignalChannelID: "627921182788161536", Leverage: 10, SubOrderType: "9"}, order.ErrAmountMustBeSet},
		{"missing sub order type", &CreateSignalBotRequest{SignalChannelID: "627921182788161536", Leverage: 10, InvestmentAmount: 100}, errMissingSubOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CreateSignalBot(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "CreateSignalBot should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *CreateSignalBotRequest
			body string
		}{
			{
				name: "every parameter",
				arg: &CreateSignalBotRequest{
					SignalChannelID: "627921182788161536", Leverage: 10, InvestmentAmount: 100, SubOrderType: "1", IncludeAll: true,
					InstrumentIDs: []string{"BTC-USDT-SWAP", "ETH-USDT-SWAP"}, Ratio: 0.002,
					EntrySetting: &SignalBotEntrySetting{AllowMultipleEntry: new(false), EntryType: "2", Amount: 20, Ratio: 0.1},
					ExitSetting:  &SignalBotExitSetting{TakeProfitStopLossType: "pnl", TakeProfitPercentage: 0.2, StopLossPercentage: 0.1},
				},
				body: `{"signalChanId":"627921182788161536","lever":"10","investAmt":"100","subOrdType":"1","includeAll":true,"instIds":["BTC-USDT-SWAP","ETH-USDT-SWAP"],"ratio":"0.002","entrySettingParam":{"allowMultipleEntry":false,"entryType":"2","amt":"20","ratio":"0.1"},"exitSettingParam":{"tpSlType":"pnl","tpPct":"0.2","slPct":"0.1"}}`,
			},
			{"required parameters only", &CreateSignalBotRequest{SignalChannelID: "627921182788161536", Leverage: 10, InvestmentAmount: 100, SubOrderType: "9"}, `{"signalChanId":"627921182788161536","lever":"10","investAmt":"100","subOrdType":"9"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/signal/order-algo", tc.body, tradingBotAlgoIDData).CreateSignalBot(t.Context(), tc.arg)
				require.NoError(t, err, "CreateSignalBot must not error")
			})
		}
	})

	t.Run("create signal bot", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.CreateSignalBot(contextGenerate(), &CreateSignalBotRequest{
			SignalChannelID: "627921182788161536", Leverage: 10, InvestmentAmount: 100, SubOrderType: "9",
			InstrumentIDs: []string{"BTC-USDT-SWAP", "ETH-USDT-SWAP", "LTC-USDT-SWAP"},
			EntrySetting:  &SignalBotEntrySetting{AllowMultipleEntry: new(true), EntryType: "1"},
			ExitSetting:   &SignalBotExitSetting{TakeProfitStopLossType: "pnl"},
		})
		require.NoError(t, err, "CreateSignalBot must not error")
		if mockTests {
			assert.Equal(t, &TradingBotOrderResponse{AlgoID: "447053782921515008"}, result, "CreateSignalBot should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "CreateSignalBot should return an algo ID")
		}
	})
}

func TestCancelSignalBots(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  []string
		err  error
	}{
		{"no algo IDs", nil, common.ErrEmptyParams},
		{"more than 10 algo IDs", make([]string, 11), errExceedLimit},
		{"empty algo ID", []string{"448965992920907776", ""}, errAlgoIDRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CancelSignalBots(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "CancelSignalBots should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/tradingBot/signal/stop-order-algo", `[{"algoId":"448965992920907776"},{"algoId":"448965992920907777"}]`, tradingBotAlgoIDData)
		_, err := ex.CancelSignalBots(t.Context(), []string{"448965992920907776", "448965992920907777"})
		require.NoError(t, err, "CancelSignalBots must not error")
	})

	t.Run("partial failure returns the rows", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"code":"2","msg":"Bulk operation partially succeeded.","data":[{"algoId":"448965992920907776","sCode":"0","sMsg":"","algoClOrdId":""},{"algoId":"448965992920907777","sCode":"51000","sMsg":"Parameter algoId error","algoClOrdId":""}]}`))
		}))
		result, err := ex.CancelSignalBots(t.Context(), []string{"448965992920907776", "448965992920907777"})
		require.ErrorIs(t, err, errPartialSuccess, "CancelSignalBots must return the partial success")
		assert.ErrorContains(t, err, "Parameter algoId error", "CancelSignalBots should report the failed row's status message")
		exp := []TradingBotOrderResponse{{AlgoID: "448965992920907776"}, {AlgoID: "448965992920907777", StatusCode: 51000, StatusMessage: "Parameter algoId error"}}
		assert.Equal(t, exp, result, "CancelSignalBots should return every row beside the error")
	})

	t.Run("cancel signal bots", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.CancelSignalBots(contextGenerate(), []string{"448965992920907776"})
		require.NoError(t, err, "CancelSignalBots must not error")
		if mockTests {
			assert.Equal(t, []TradingBotOrderResponse{{AlgoID: "448965992920907776"}}, result, "CancelSignalBots should decode every field")
		} else {
			assert.NotEmpty(t, result, "CancelSignalBots should return the cancelled bots")
		}
	})
}

func TestAdjustSignalBotMarginBalance(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AdjustSignalBotMarginBalanceRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &AdjustSignalBotMarginBalanceRequest{AdjustMarginBalanceType: marginBalanceAdd, Amount: 10}, errAlgoIDRequired},
		{"missing adjustment type", &AdjustSignalBotMarginBalanceRequest{AlgoID: "123456", Amount: 10}, errMarginBalanceTypeRequired},
		{"missing amount", &AdjustSignalBotMarginBalanceRequest{AlgoID: "123456", AdjustMarginBalanceType: marginBalanceAdd}, order.ErrAmountMustBeSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AdjustSignalBotMarginBalance(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "AdjustSignalBotMarginBalance should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *AdjustSignalBotMarginBalanceRequest
			body string
		}{
			{"every parameter", &AdjustSignalBotMarginBalanceRequest{AlgoID: "123456", AdjustMarginBalanceType: marginBalanceAdd, Amount: 10, AllowReinvest: true}, `{"algoId":"123456","type":"add","amt":"10","allowReinvest":true}`},
			{"required parameters only", &AdjustSignalBotMarginBalanceRequest{AlgoID: "123456", AdjustMarginBalanceType: marginBalanceReduce, Amount: 10}, `{"algoId":"123456","type":"reduce","amt":"10"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/signal/margin-balance", tc.body, tradingBotAlgoIDData).AdjustSignalBotMarginBalance(t.Context(), tc.arg)
				require.NoError(t, err, "AdjustSignalBotMarginBalance must not error")
			})
		}
	})

	t.Run("adjust margin balance", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.AdjustSignalBotMarginBalance(contextGenerate(), &AdjustSignalBotMarginBalanceRequest{AlgoID: "123456", AdjustMarginBalanceType: marginBalanceAdd, Amount: 10})
		require.NoError(t, err, "AdjustSignalBotMarginBalance must not error")
		if mockTests {
			assert.Equal(t, &TradingBotAlgoIDResponse{AlgoID: "123456"}, result, "AdjustSignalBotMarginBalance should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "AdjustSignalBotMarginBalance should return an algo ID")
		}
	})
}

func TestAmendSignalBotTakeProfitStopLoss(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AmendSignalBotTakeProfitStopLossRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &AmendSignalBotTakeProfitStopLossRequest{ExitSetting: SignalBotExitSetting{TakeProfitStopLossType: "pnl"}}, errAlgoIDRequired},
		{"missing take profit and stop loss type", &AmendSignalBotTakeProfitStopLossRequest{AlgoID: "637039348240277504", ExitSetting: SignalBotExitSetting{TakeProfitPercentage: 0.01}}, errTakeProfitStopLossTypeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AmendSignalBotTakeProfitStopLoss(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "AmendSignalBotTakeProfitStopLoss should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *AmendSignalBotTakeProfitStopLossRequest
			body string
		}{
			{"every parameter", &AmendSignalBotTakeProfitStopLossRequest{AlgoID: "637039348240277504", ExitSetting: SignalBotExitSetting{TakeProfitStopLossType: "price", TakeProfitPercentage: 0.02, StopLossPercentage: 0.01}}, `{"algoId":"637039348240277504","exitSettingParam":{"tpSlType":"price","tpPct":"0.02","slPct":"0.01"}}`},
			{"required parameters only", &AmendSignalBotTakeProfitStopLossRequest{AlgoID: "637039348240277504", ExitSetting: SignalBotExitSetting{TakeProfitStopLossType: "pnl"}}, `{"algoId":"637039348240277504","exitSettingParam":{"tpSlType":"pnl"}}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/signal/amendTPSL", tc.body, tradingBotAlgoIDData).AmendSignalBotTakeProfitStopLoss(t.Context(), tc.arg)
				require.NoError(t, err, "AmendSignalBotTakeProfitStopLoss must not error")
			})
		}
	})

	t.Run("amend take profit and stop loss", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.AmendSignalBotTakeProfitStopLoss(contextGenerate(), &AmendSignalBotTakeProfitStopLossRequest{AlgoID: "637039348240277504", ExitSetting: SignalBotExitSetting{TakeProfitStopLossType: "pnl", TakeProfitPercentage: 0.01, StopLossPercentage: 0.01}})
		require.NoError(t, err, "AmendSignalBotTakeProfitStopLoss must not error")
		if mockTests {
			assert.Equal(t, &TradingBotAlgoIDResponse{AlgoID: "637039348240277504"}, result, "AmendSignalBotTakeProfitStopLoss should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "AmendSignalBotTakeProfitStopLoss should return an algo ID")
		}
	})
}

func TestSetSignalBotInstruments(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *SetSignalBotInstrumentsRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &SetSignalBotInstrumentsRequest{InstrumentIDs: []string{"ETH-USDT-SWAP"}}, errAlgoIDRequired},
		{"missing instrument IDs", &SetSignalBotInstrumentsRequest{AlgoID: "637039348240277504"}, errMissingInstrumentID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.SetSignalBotInstruments(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "SetSignalBotInstruments should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *SetSignalBotInstrumentsRequest
			body string
		}{
			{"every parameter", &SetSignalBotInstrumentsRequest{AlgoID: "637039348240277504", InstrumentIDs: []string{"SHIB-USDT-SWAP"}, IncludeAll: true}, `{"algoId":"637039348240277504","instIds":["SHIB-USDT-SWAP"],"includeAll":true}`},
			{"every USDT margined contract", &SetSignalBotInstrumentsRequest{AlgoID: "637039348240277504", IncludeAll: true}, `{"algoId":"637039348240277504","includeAll":true}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := newRequestPinExchange(t, "/tradingBot/signal/set-instruments", tc.body, tradingBotAlgoIDData).SetSignalBotInstruments(t.Context(), tc.arg)
				require.NoError(t, err, "SetSignalBotInstruments must not error")
			})
		}
	})

	t.Run("set instruments", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.SetSignalBotInstruments(contextGenerate(), &SetSignalBotInstrumentsRequest{AlgoID: "637039348240277504", InstrumentIDs: []string{"SHIB-USDT-SWAP", "ETH-USDT-SWAP"}})
		require.NoError(t, err, "SetSignalBotInstruments must not error")
		if mockTests {
			assert.Equal(t, &TradingBotAlgoIDResponse{AlgoID: "637039348240277504"}, result, "SetSignalBotInstruments should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "SetSignalBotInstruments should return an algo ID")
		}
	})
}

func TestGetSignalBotOrderDetails(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *SignalBotOrderRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo order type", &SignalBotOrderRequest{AlgoID: "623833708424069120"}, errInvalidAlgoOrderType},
		{"missing algo ID", &SignalBotOrderRequest{AlgoOrderType: "contract"}, errAlgoIDRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSignalBotOrderDetails(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetSignalBotOrderDetails should return the expected error")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSignalBotOrderDetails(contextGenerate(), &SignalBotOrderRequest{AlgoOrderType: "contract", AlgoID: "623833708424069120"})
	require.NoError(t, err, "GetSignalBotOrderDetails must not error")
	if mockTests {
		exp := &SignalBotOrderResponse{
			AlgoID:            "623833708424069120",
			AlgoClientOrderID: "signalClient1",
			InstrumentType:    "SWAP",
			InstrumentIDs:     []string{"BTC-USDT-SWAP", "ETH-USDT-SWAP"},
			CreationTime:      types.Time(time.UnixMilli(1695005546360)),
			UpdateTime:        types.Time(time.UnixMilli(1697029422313)),
			AlgoOrderType:     "contract",
			State:             "running",
			CancelType:        "0",
			TotalPNL:          -73.1757030986877733,
			TotalPNLRatio:     -0.7317570309868777,
			TotalEquity:       26.824296901312227,
			FloatPNL:          0.1279999999999927,
			RealisedPNL:       -73.303703098687766,
			FrozenBalance:     25.16816,
			AvailableBalance:  1.6561369013122267,
			Leverage:          10,
			InvestmentAmount:  100,
			SubOrderType:      "9",
			EntrySetting:      SignalBotEntrySetting{AllowMultipleEntry: new(true), EntryType: "1"},
			ExitSetting:       SignalBotExitSetting{TakeProfitStopLossType: "price"},
			SignalChannelID:   "623827579484770304",
			SignalChannelName: "testing",
			SignalSourceType:  "1",
		}
		assert.Equal(t, exp, result, "GetSignalBotOrderDetails should decode every field")
	} else {
		assert.NotEmpty(t, result.AlgoID, "GetSignalBotOrderDetails should return an algo ID")
	}
}

func TestGetActiveSignalBots(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveSignalBots(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetActiveSignalBots(t.Context(), &SignalBotListRequest{AlgoID: "623833708424069121"})
	require.ErrorIs(t, err, errInvalidAlgoOrderType)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetActiveSignalBots(contextGenerate(), &SignalBotListRequest{AlgoOrderType: "contract"})
	require.NoError(t, err, "GetActiveSignalBots must not error")
	if mockTests {
		exp := []SignalBotOrderResponse{
			{
				AlgoID:            "623833708424069121",
				InstrumentType:    "SWAP",
				InstrumentIDs:     []string{"BTC-USDT-SWAP"},
				CreationTime:      types.Time(time.UnixMilli(1695005546360)),
				UpdateTime:        types.Time(time.UnixMilli(1697029422313)),
				AlgoOrderType:     "contract",
				State:             "running",
				CancelType:        "0",
				TotalPNL:          2.05,
				TotalPNLRatio:     0.0205,
				TotalEquity:       102.05,
				FloatPNL:          0.85,
				RealisedPNL:       1.2,
				FrozenBalance:     51.75,
				AvailableBalance:  48.25,
				Leverage:          5,
				InvestmentAmount:  100,
				SubOrderType:      "1",
				Ratio:             0.002,
				EntrySetting:      SignalBotEntrySetting{AllowMultipleEntry: new(false), EntryType: "2", Amount: 20},
				ExitSetting:       SignalBotExitSetting{TakeProfitStopLossType: "pnl", TakeProfitPercentage: 0.2, StopLossPercentage: 0.1},
				SignalChannelID:   "623827579484770304",
				SignalChannelName: "my signal",
				SignalSourceType:  "2",
			},
		}
		assert.Equal(t, exp, result, "GetActiveSignalBots should decode every field")
	} else {
		assert.NotNil(t, result, "GetActiveSignalBots should return a list")
	}
}

func TestGetSignalBotHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSignalBotHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetSignalBotHistory(t.Context(), &SignalBotListRequest{AlgoOrderType: "contract"})
	require.ErrorIs(t, err, errAlgoIDRequired)
	_, err = e.GetSignalBotHistory(t.Context(), &SignalBotListRequest{AlgoID: "623833708424069120"})
	require.ErrorIs(t, err, errInvalidAlgoOrderType)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSignalBotHistory(contextGenerate(), &SignalBotListRequest{AlgoOrderType: "contract", AlgoID: "623833708424069120"})
	require.NoError(t, err, "GetSignalBotHistory must not error")
	if mockTests {
		exp := []SignalBotOrderResponse{
			{
				AlgoID:            "623833708424069120",
				InstrumentType:    "SWAP",
				InstrumentIDs:     []string{"BTC-USDT-SWAP", "ETH-USDT-SWAP"},
				CreationTime:      types.Time(time.UnixMilli(1695005546360)),
				UpdateTime:        types.Time(time.UnixMilli(1697029422313)),
				AlgoOrderType:     "contract",
				State:             "stopped",
				CancelType:        "1",
				TotalPNL:          -73.1757030986877733,
				TotalPNLRatio:     -0.7317570309868777,
				TotalEquity:       26.824296901312227,
				FloatPNL:          0.1279999999999927,
				RealisedPNL:       -73.303703098687766,
				FrozenBalance:     25.16816,
				AvailableBalance:  1.6561369013122267,
				Leverage:          10,
				InvestmentAmount:  100,
				SubOrderType:      "9",
				EntrySetting:      SignalBotEntrySetting{AllowMultipleEntry: new(true), EntryType: "4", Ratio: 0.1},
				ExitSetting:       SignalBotExitSetting{TakeProfitStopLossType: "price"},
				SignalChannelID:   "623827579484770304",
				SignalChannelName: "my signal",
				SignalSourceType:  "1",
			},
		}
		assert.Equal(t, exp, result, "GetSignalBotHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetSignalBotHistory should return a list")
	}
}

func TestSignalBotListParams(t *testing.T) {
	t.Parallel()
	_, err := signalBotListParams(&SignalBotListRequest{})
	require.ErrorIs(t, err, errInvalidAlgoOrderType)
	_, err = signalBotListParams(&SignalBotListRequest{AlgoOrderType: "contract", After: time.UnixMilli(1695005546360), Before: time.UnixMilli(1697029422313)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	params, err := signalBotListParams(&SignalBotListRequest{AlgoOrderType: "contract", AlgoID: "623833708424069120", After: time.UnixMilli(1697029422313), Before: time.UnixMilli(1695005546360), Limit: 100})
	require.NoError(t, err, "signalBotListParams must not error")
	exp := url.Values{"algoOrdType": {"contract"}, "algoId": {"623833708424069120"}, "after": {"1697029422313"}, "before": {"1695005546360"}, "limit": {"100"}}
	assert.Equal(t, exp, params, "signalBotListParams should set every parameter")
}

func TestSignalBotTimeCursorParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		after, before time.Time
		limit         uint64
		exp           url.Values
		err           error
	}{
		{name: "no parameters", exp: url.Values{}},
		{name: "every parameter", after: time.UnixMilli(1697029422313), before: time.UnixMilli(1695005546360), limit: 50, exp: url.Values{"after": {"1697029422313"}, "before": {"1695005546360"}, "limit": {"50"}}},
		{name: "after only", after: time.UnixMilli(1697029422313), exp: url.Values{"after": {"1697029422313"}}},
		{name: "before only", before: time.UnixMilli(1695005546360), exp: url.Values{"before": {"1695005546360"}}},
		{name: "equal cursors", after: time.UnixMilli(1695005546360), before: time.UnixMilli(1695005546360), exp: url.Values{"after": {"1695005546360"}, "before": {"1695005546360"}}},
		{name: "before later than after", after: time.UnixMilli(1695005546360), before: time.UnixMilli(1695005546361), err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := signalBotTimeCursorParams(tc.after, tc.before, tc.limit)
			require.ErrorIs(t, err, tc.err, "signalBotTimeCursorParams must return the expected error")
			assert.Equal(t, tc.exp, params, "signalBotTimeCursorParams should return the expected parameters")
		})
	}
}

func TestGetSignalBotOrderPositions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *SignalBotOrderRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo order type", &SignalBotOrderRequest{AlgoID: "623833708424069120"}, errInvalidAlgoOrderType},
		{"missing algo ID", &SignalBotOrderRequest{AlgoOrderType: "contract"}, errAlgoIDRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSignalBotOrderPositions(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetSignalBotOrderPositions should return the expected error")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSignalBotOrderPositions(contextGenerate(), &SignalBotOrderRequest{AlgoOrderType: "contract", AlgoID: "623833708424069120"})
	require.NoError(t, err, "GetSignalBotOrderPositions must not error")
	if mockTests {
		exp := []TradingBotPositionResponse{
			{
				AlgoID:                       "623833708424069120",
				AlgoClientOrderID:            "signalClient1",
				InstrumentType:               "SWAP",
				InstrumentID:                 "ETH-USDT-SWAP",
				CreationTime:                 types.Time(time.UnixMilli(1697502301460)),
				UpdateTime:                   types.Time(time.UnixMilli(1697502301460)),
				AveragePrice:                 1597.74,
				Currency:                     currency.USDT,
				Leverage:                     10,
				LiquidationPrice:             1438.7380360728976,
				PositionSide:                 positionSideNet,
				Position:                     15,
				MarginMode:                   TradeModeCross,
				MarginRatio:                  11.719278420807477,
				InitialMarginRequirement:     23.76495,
				MaintenanceMarginRequirement: 1.9011959999999997,
				UnrealisedPNL:                -2.0115000000000123,
				UnrealisedPNLRatio:           -0.0839310526118142,
				LastTradedPrice:              1584.34,
				NotionalUSD:                  237.75168928499997,
				AutoDeleveraging:             1,
				MarkPrice:                    1584.33,
			},
		}
		assert.Equal(t, exp, result, "GetSignalBotOrderPositions should decode every field")
	} else {
		assert.NotNil(t, result, "GetSignalBotOrderPositions should return a list")
	}
}

func TestSignalBotOrderParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *SignalBotOrderRequest
		exp  url.Values
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing algo order type", arg: &SignalBotOrderRequest{AlgoID: "623833708424069120"}, err: errInvalidAlgoOrderType},
		{name: "missing algo ID", arg: &SignalBotOrderRequest{AlgoOrderType: "contract"}, err: errAlgoIDRequired},
		{name: "algo order type and ID", arg: &SignalBotOrderRequest{AlgoOrderType: "contract", AlgoID: "623833708424069120"}, exp: url.Values{"algoOrdType": {"contract"}, "algoId": {"623833708424069120"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := signalBotOrderParams(tc.arg)
			require.ErrorIs(t, err, tc.err, "signalBotOrderParams must return the expected error")
			assert.Equal(t, tc.exp, params, "signalBotOrderParams should encode the algo order type and ID")
		})
	}
}

func TestGetSignalBotPositionHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *SignalBotPositionHistoryRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &SignalBotPositionHistoryRequest{InstrumentID: "ETH-USDT-SWAP"}, errAlgoIDRequired},
		{"before later than after", &SignalBotPositionHistoryRequest{AlgoID: "1234", After: time.UnixMilli(1704724451471), Before: time.UnixMilli(1704724456322)}, common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSignalBotPositionHistory(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetSignalBotPositionHistory should return the expected error")
		})
	}

	t.Run("query with every parameter", func(t *testing.T) {
		t.Parallel()
		exp := url.Values{"algoId": {"1234"}, "instId": {"ETH-USDT-SWAP"}, "after": {"1704724456322"}, "before": {"1704724451471"}, "limit": {"20"}}
		ex := newQueryPinExchange(t, "/tradingBot/signal/positions-history", exp, []map[string]string{{"instId": "ETH-USDT-SWAP"}})
		_, err := ex.GetSignalBotPositionHistory(t.Context(), &SignalBotPositionHistoryRequest{AlgoID: "1234", InstrumentID: "ETH-USDT-SWAP", After: time.UnixMilli(1704724456322), Before: time.UnixMilli(1704724451471), Limit: 20})
		require.NoError(t, err, "GetSignalBotPositionHistory must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSignalBotPositionHistory(contextGenerate(), &SignalBotPositionHistoryRequest{AlgoID: "1234"})
	require.NoError(t, err, "GetSignalBotPositionHistory must not error")
	if mockTests {
		exp := []SignalBotPositionHistoryResponse{
			{
				InstrumentID:      "ETH-USDT-SWAP",
				MarginMode:        TradeModeCross,
				CreationTime:      types.Time(time.UnixMilli(1704724451471)),
				UpdateTime:        types.Time(time.UnixMilli(1704724456322)),
				OpenAveragePrice:  220,
				CloseAveragePrice: 200,
				PNL:               -2.021,
				PNLRatio:          -0.4593181818181818,
				Leverage:          5,
				Direction:         positionSideNet,
				Underlying:        "ETH-USDT",
			},
		}
		assert.Equal(t, exp, result, "GetSignalBotPositionHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetSignalBotPositionHistory should return a list")
	}
}

func TestCloseSignalBotPosition(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CloseSignalBotPositionRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &CloseSignalBotPositionRequest{InstrumentID: perpetualSwapPair.String()}, errAlgoIDRequired},
		{"missing instrument ID", &CloseSignalBotPositionRequest{AlgoID: "448965992920907776"}, errMissingInstrumentID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CloseSignalBotPosition(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "CloseSignalBotPosition should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/tradingBot/signal/close-position", `{"algoId":"448965992920907776","instId":"ETH-USDT-SWAP"}`, tradingBotAlgoIDData)
		_, err := ex.CloseSignalBotPosition(t.Context(), &CloseSignalBotPositionRequest{AlgoID: "448965992920907776", InstrumentID: "ETH-USDT-SWAP"})
		require.NoError(t, err, "CloseSignalBotPosition must not error")
	})

	t.Run("close position", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.CloseSignalBotPosition(contextGenerate(), &CloseSignalBotPositionRequest{AlgoID: "448965992920907776", InstrumentID: perpetualSwapPair.String()})
		require.NoError(t, err, "CloseSignalBotPosition must not error")
		if mockTests {
			assert.Equal(t, &TradingBotAlgoIDResponse{AlgoID: "448965992920907776"}, result, "CloseSignalBotPosition should decode every field")
		} else {
			assert.NotEmpty(t, result.AlgoID, "CloseSignalBotPosition should return an algo ID")
		}
	})
}

func TestPlaceSignalBotSubOrder(t *testing.T) {
	t.Parallel()
	valid := func() *PlaceSignalBotSubOrderRequest {
		return &PlaceSignalBotSubOrderRequest{InstrumentID: perpetualSwapPair.String(), AlgoID: "1222", Side: order.Buy.Lower(), OrderType: orderLimit, Size: 2, Price: 2.15}
	}
	for _, tc := range []struct {
		name string
		arg  func() *PlaceSignalBotSubOrderRequest
		err  error
	}{
		{"nil request", func() *PlaceSignalBotSubOrderRequest { return nil }, common.ErrNilPointer},
		{"missing instrument ID", func() *PlaceSignalBotSubOrderRequest { a := valid(); a.InstrumentID = ""; return a }, errMissingInstrumentID},
		{"missing algo ID", func() *PlaceSignalBotSubOrderRequest { a := valid(); a.AlgoID = ""; return a }, errAlgoIDRequired},
		{"missing side", func() *PlaceSignalBotSubOrderRequest { a := valid(); a.Side = ""; return a }, order.ErrSideIsInvalid},
		{"missing order type", func() *PlaceSignalBotSubOrderRequest { a := valid(); a.OrderType = ""; return a }, order.ErrTypeIsInvalid},
		{"missing size", func() *PlaceSignalBotSubOrderRequest { a := valid(); a.Size = 0; return a }, order.ErrAmountMustBeSet},
		{"limit order without price", func() *PlaceSignalBotSubOrderRequest { a := valid(); a.Price = 0; return a }, limits.ErrPriceBelowMin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := e.PlaceSignalBotSubOrder(t.Context(), tc.arg())
			assert.ErrorIs(t, err, tc.err, "PlaceSignalBotSubOrder should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *PlaceSignalBotSubOrderRequest
			body string
		}{
			{"every parameter", &PlaceSignalBotSubOrderRequest{InstrumentID: perpetualSwapPair.String(), AlgoID: "1222", Side: order.Sell.Lower(), OrderType: orderLimit, Size: 2, Price: 2.15, ReduceOnly: true}, `{"instId":"BTC-USDT-SWAP","algoId":"1222","side":"sell","ordType":"limit","sz":"2","px":"2.15","reduceOnly":true}`},
			{"market order with required parameters only", &PlaceSignalBotSubOrderRequest{InstrumentID: perpetualSwapPair.String(), AlgoID: "1222", Side: order.Buy.Lower(), OrderType: orderMarket, Size: 2}, `{"instId":"BTC-USDT-SWAP","algoId":"1222","side":"buy","ordType":"market","sz":"2"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				err := newRequestPinExchange(t, "/tradingBot/signal/sub-order", tc.body, []any{}).PlaceSignalBotSubOrder(t.Context(), tc.arg)
				require.NoError(t, err, "PlaceSignalBotSubOrder must not error")
			})
		}
	})

	t.Run("place sub order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		require.NoError(t, e.PlaceSignalBotSubOrder(contextGenerate(), valid()), "PlaceSignalBotSubOrder must not error")
	})
}

func TestCancelSignalBotSubOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CancelSignalBotSubOrderRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &CancelSignalBotSubOrderRequest{InstrumentID: perpetualSwapPair.String(), SignalOrderID: "590908157585625111"}, errAlgoIDRequired},
		{"missing instrument ID", &CancelSignalBotSubOrderRequest{AlgoID: "91664", SignalOrderID: "590908157585625111"}, errMissingInstrumentID},
		{"missing signal order ID", &CancelSignalBotSubOrderRequest{AlgoID: "91664", InstrumentID: perpetualSwapPair.String()}, errSignalOrderIDRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CancelSignalBotSubOrder(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "CancelSignalBotSubOrder should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/tradingBot/signal/cancel-sub-order", `{"algoId":"91664","instId":"BTC-USDT-SWAP","signalOrdId":"590908157585625111"}`, []map[string]string{{"signalOrdId": "590908157585625111", "sCode": "0"}})
		_, err := ex.CancelSignalBotSubOrder(t.Context(), &CancelSignalBotSubOrderRequest{AlgoID: "91664", InstrumentID: perpetualSwapPair.String(), SignalOrderID: "590908157585625111"})
		require.NoError(t, err, "CancelSignalBotSubOrder must not error")
	})

	t.Run("rejected cancellation returns the row", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"code":"1","msg":"Operation failed.","data":[{"signalOrdId":"590908157585625111","sCode":"51400","sMsg":"Cancellation failed as the order has been filled, canceled or does not exist."}]}`))
		}))
		result, err := ex.CancelSignalBotSubOrder(t.Context(), &CancelSignalBotSubOrderRequest{AlgoID: "91664", InstrumentID: perpetualSwapPair.String(), SignalOrderID: "590908157585625111"})
		require.ErrorContains(t, err, "Operation failed.", "CancelSignalBotSubOrder must report OKX's error")
		exp := &CancelSignalBotSubOrderResponse{SignalOrderID: "590908157585625111", StatusCode: 51400, StatusMessage: "Cancellation failed as the order has been filled, canceled or does not exist."}
		assert.Equal(t, exp, result, "CancelSignalBotSubOrder should return the rejected row beside the error")
	})

	t.Run("cancel sub order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.CancelSignalBotSubOrder(contextGenerate(), &CancelSignalBotSubOrderRequest{AlgoID: "91664", InstrumentID: perpetualSwapPair.String(), SignalOrderID: "590908157585625111"})
		require.NoError(t, err, "CancelSignalBotSubOrder must not error")
		if mockTests {
			assert.Equal(t, &CancelSignalBotSubOrderResponse{SignalOrderID: "590908157585625111"}, result, "CancelSignalBotSubOrder should decode every field")
		} else {
			assert.NotEmpty(t, result.SignalOrderID, "CancelSignalBotSubOrder should return the sub order ID")
		}
	})
}

func TestGetSignalBotSubOrders(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *SignalBotSubOrdersRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &SignalBotSubOrdersRequest{AlgoOrderType: "contract", SubOrderType: "filled"}, errAlgoIDRequired},
		{"missing algo order type", &SignalBotSubOrdersRequest{AlgoID: "623833708424069120", SubOrderType: "filled"}, errInvalidAlgoOrderType},
		{"missing state and sub order ID", &SignalBotSubOrdersRequest{AlgoID: "623833708424069120", AlgoOrderType: "contract"}, order.ErrOrderIDNotSet},
		{"begin later than end", &SignalBotSubOrdersRequest{AlgoID: "623833708424069120", AlgoOrderType: "contract", State: "filled", Begin: time.UnixMilli(1697024802320), End: time.UnixMilli(1697024702320)}, common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSignalBotSubOrders(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetSignalBotSubOrders should return the expected error")
		})
	}

	t.Run("query with every parameter", func(t *testing.T) {
		t.Parallel()
		exp := url.Values{
			"algoId": {"623833708424069120"}, "algoOrdType": {"contract"}, "state": {"filled"}, "signalOrdId": {"O632302662327996418"},
			"after": {"632302662351958017"}, "before": {"632302662351958000"}, "begin": {"1697024702320"}, "end": {"1697024802320"},
			"limit": {"30"}, "type": {"filled"}, "clOrdId": {"O632302662327996418"},
		}
		ex := newQueryPinExchange(t, "/tradingBot/signal/sub-orders", exp, tradingBotAlgoIDData)
		_, err := ex.GetSignalBotSubOrders(t.Context(), &SignalBotSubOrdersRequest{
			AlgoID: "623833708424069120", AlgoOrderType: "contract", State: "filled", SignalOrderID: "O632302662327996418", After: "632302662351958017", Before: "632302662351958000",
			Begin: time.UnixMilli(1697024702320), End: time.UnixMilli(1697024802320), Limit: 30, SubOrderType: "filled", ClientOrderID: "O632302662327996418",
		})
		require.NoError(t, err, "GetSignalBotSubOrders must not error")
	})

	for _, tc := range []struct {
		name string
		arg  *SignalBotSubOrdersRequest
		exp  []SignalBotSubOrderResponse
	}{
		{
			name: "filled sub orders",
			arg:  &SignalBotSubOrdersRequest{AlgoID: "623833708424069120", AlgoOrderType: "contract", SubOrderType: "filled"},
			exp: []SignalBotSubOrderResponse{
				{
					AlgoID:              "623833708424069120",
					InstrumentType:      "SWAP",
					InstrumentID:        "ETH-USDT-SWAP",
					AlgoOrderType:       "contract",
					OrderID:             "632302662351958016",
					ClientOrderID:       "O632302662327996418",
					CreationTime:        types.Time(time.UnixMilli(1697024702320)),
					UpdateTime:          types.Time(time.UnixMilli(1697024702322)),
					TradeMode:           TradeModeCross,
					OrderType:           orderMarket,
					Size:                18,
					State:               "filled",
					Side:                "buy",
					Fee:                 -0.1415529,
					FeeCurrency:         currency.USDT,
					AveragePrice:        1572.81,
					AccumulatedFillSize: 18,
					PositionSide:        positionSideNet,
					PNL:                 -2.6784,
					ContractValue:       0.01,
					Leverage:            10,
				},
			},
		},
		{
			name: "designated sub order",
			arg:  &SignalBotSubOrdersRequest{AlgoID: "623833708424069120", AlgoOrderType: "contract", SignalOrderID: "O632302662327996419"},
			exp: []SignalBotSubOrderResponse{
				{
					AlgoID:              "623833708424069120",
					AlgoClientOrderID:   "signalClient1",
					InstrumentType:      "SWAP",
					InstrumentID:        "ETH-USDT-SWAP",
					AlgoOrderType:       "contract",
					OrderID:             "632302662351958017",
					ClientOrderID:       "O632302662327996419",
					CreationTime:        types.Time(time.UnixMilli(1697024802320)),
					UpdateTime:          types.Time(time.UnixMilli(1697024802322)),
					TradeMode:           TradeModeCross,
					Currency:            currency.USDT,
					OrderType:           orderLimit,
					Size:                5,
					State:               "filled",
					Side:                "sell",
					Price:               1571.4,
					Fee:                 -0.03928,
					FeeCurrency:         currency.USDT,
					AveragePrice:        1571.4,
					AccumulatedFillSize: 5,
					PositionSide:        positionSideNet,
					ContractValue:       0.01,
					Leverage:            10,
					Tag:                 "signalTag1",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			result, err := e.GetSignalBotSubOrders(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetSignalBotSubOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSignalBotSubOrders should decode every field")
			} else {
				assert.NotNil(t, result, "GetSignalBotSubOrders should return a list")
			}
		})
	}
}

func TestGetSignalBotEventHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *SignalBotEventHistoryRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing algo ID", &SignalBotEventHistoryRequest{After: time.UnixMilli(1697453500000), Limit: 50}, errAlgoIDRequired},
		{"before later than after", &SignalBotEventHistoryRequest{AlgoID: "12345", After: time.UnixMilli(1697453400000), Before: time.UnixMilli(1697453500000)}, common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSignalBotEventHistory(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetSignalBotEventHistory should return the expected error")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSignalBotEventHistory(contextGenerate(), &SignalBotEventHistoryRequest{AlgoID: "12345", After: time.UnixMilli(1697453500000), Limit: 50})
	require.NoError(t, err, "GetSignalBotEventHistory must not error")
	if mockTests {
		exp := []SignalBotEventHistoryResponse{
			{
				AlertMessage:        `{"marketPosition":"short","prevMarketPosition":"long","action":"sell","instrument":"ETHUSDT.P","timestamp":"2023-10-16T10:50:00.000Z","maxLag":"60","investmentType":"base","amount":"2"}`,
				AlgoID:              "12345",
				EventType:           "signal_processing",
				EventCreationTime:   types.Time(time.UnixMilli(1697453400959)),
				EventUpdateTime:     types.Time(time.UnixMilli(1697453401012)),
				EventProcessMessage: "Processed reverse entry signal and placed ETH-USDT-SWAP order with all available balance",
				EventStatus:         "success",
				TriggeredOrderData:  []SignalBotTriggeredOrder{{ClientOrderID: "O634100754731765763"}, {ClientOrderID: "O634100754752737282"}},
			},
		}
		assert.Equal(t, exp, result, "GetSignalBotEventHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetSignalBotEventHistory should return a list")
	}
}
