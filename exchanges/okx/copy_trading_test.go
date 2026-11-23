package okx

import (
	"errors"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	leadTraderUniqueID string
	loadLeadTraderOnce sync.Once
	errSyncLeadTrader  error

	leadTraderSubPositionsUniqueID string
	loadLeadTraderSubPositionsOnce sync.Once
	errSyncLeadTraderSubPositions  error
)

// leadTraderLookupRequest is the ranking the lead trader lookups take their candidates from
var leadTraderLookupRequest = LeadTraderRanksRequest{InstrumentType: instTypeSwap, SortType: "pnl_ratio", HasVacancy: true, Limit: 10}

func syncLeadTraderUniqueID(t *testing.T) error {
	t.Helper()

	if useTestNet {
		t.Skip("Testnet does not support lead trader API")
	}

	loadLeadTraderOnce.Do(func() {
		result, err := e.GetLeadTraderRanks(contextGenerate(), &leadTraderLookupRequest)
		if err != nil {
			errSyncLeadTrader = fmt.Errorf("GetLeadTraderRanks failed: %w", err)
			return
		}
		if len(result.Ranks) == 0 {
			errSyncLeadTrader = errors.New("no lead trader found")
			return
		}
		leadTraderUniqueID = result.Ranks[0].UniqueCode
	})

	return errSyncLeadTrader
}

func syncLeadTraderSubPositionsUniqueID(t *testing.T) error {
	t.Helper()

	if useTestNet {
		t.Skip("Testnet does not support lead trader API")
	}

	loadLeadTraderSubPositionsOnce.Do(func() {
		result, err := e.GetLeadTraderRanks(contextGenerate(), &leadTraderLookupRequest)
		if err != nil {
			errSyncLeadTraderSubPositions = fmt.Errorf("GetLeadTraderRanks failed: %w", err)
			return
		}
		for i := range result.Ranks {
			candidate := &LeadTraderPositionsRequest{InstrumentType: instTypeSwap, UniqueCode: result.Ranks[i].UniqueCode, Limit: 1}
			if _, err := e.GetLeadTraderCurrentLeadPositions(contextGenerate(), candidate); err != nil {
				continue
			}
			if _, err := e.GetLeadTraderLeadPositionHistory(contextGenerate(), candidate); err != nil {
				continue
			}
			leadTraderSubPositionsUniqueID = candidate.UniqueCode
			return
		}
		errSyncLeadTraderSubPositions = errors.New("no lead trader found with working subposition endpoints")
	})

	return errSyncLeadTraderSubPositions
}

// requireCopyTradingBody runs call against a mock exchange which checks that it posts body to path
func requireCopyTradingBody(t *testing.T, path, body string, reply any, call func(*Exchange) error) {
	t.Helper()
	require.NoError(t, call(newRequestPinExchange(t, "/"+path, body, reply)), "the request must not error")
}

func TestGetExistingLeadPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetExistingLeadPositions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetExistingLeadPositions(contextGenerate(), &LeadPositionsRequest{InstrumentType: instTypeSpot, InstrumentID: mainPair.String(), Before: "649920301388038143", Limit: 2})
	require.NoError(t, err, "GetExistingLeadPositions must not error")
	if mockTests {
		exp := []LeadPositionResponse{
			{
				InstrumentID:                 "BTC-USDT",
				SubPositionID:                "649945658862370816",
				PositionSide:                 "net",
				MarginMode:                   "isolated",
				Leverage:                     types.Number(3),
				OpenOrderID:                  "649945658862370810",
				OpenAveragePrice:             types.Number(37925.1),
				OpenTime:                     types.Time(time.UnixMilli(1701231120479)),
				SubPosition:                  types.Number(1),
				TakeProfitTriggerPrice:       types.Number(40000),
				StopLossTriggerPrice:         types.Number(36000),
				AlgoID:                       "649945658862370820",
				InstrumentType:               instTypeSpot,
				TakeProfitOrderPrice:         types.Number(-1),
				StopLossOrderPrice:           types.Number(-1),
				Margin:                       types.Number(12.6417),
				UnrealisedProfitAndLoss:      types.Number(0.2807),
				UnrealisedProfitAndLossRatio: types.Number(0.0222042921442527),
				UniqueCode:                   "25CD5A80241D6FE6",
				MarginCurrency:               currency.USDT,
				AvailableSubPosition:         types.Number(1),
			},
			{
				InstrumentID:                 "BTC-USDT",
				SubPositionID:                "649920301388038144",
				PositionSide:                 "net",
				MarginMode:                   "isolated",
				Leverage:                     types.Number(3),
				OpenAveragePrice:             types.Number(37879),
				OpenTime:                     types.Time(time.UnixMilli(1701225074786)),
				SubPosition:                  types.Number(1),
				InstrumentType:               instTypeSpot,
				Margin:                       types.Number(12.6263333333333333),
				UnrealisedProfitAndLoss:      types.Number(0.3268),
				UnrealisedProfitAndLossRatio: types.Number(0.0258824150584758),
				UniqueCode:                   "25CD5A80241D6FE6",
				MarginCurrency:               currency.USDT,
				AvailableSubPosition:         types.Number(1),
			},
		}
		assert.Equal(t, exp, result, "GetExistingLeadPositions should decode every field")
	} else {
		assert.NotNil(t, result, "GetExistingLeadPositions should return a list")
	}
}

func TestGetLeadPositionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeadPositionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLeadPositionHistory(contextGenerate(), &LeadPositionsRequest{InstrumentType: instTypeSwap, InstrumentID: perpetualSwapPair.String(), After: "649750700213698562", Limit: 2})
	require.NoError(t, err, "GetLeadPositionHistory must not error")
	if mockTests {
		exp := []LeadPositionHistoryResponse{
			{
				InstrumentID:        perpetualSwapPair.String(),
				SubPositionID:       "649750700213698561",
				PositionSide:        "net",
				MarginMode:          "isolated",
				Leverage:            types.Number(3),
				OpenOrderID:         "649750700213698500",
				OpenAveragePrice:    types.Number(37410),
				OpenTime:            types.Time(time.UnixMilli(1701184638702)),
				SubPosition:         types.Number(3),
				CloseTime:           types.Time(time.UnixMilli(1701188587950)),
				CloseAveragePrice:   types.Number(37617.5),
				ProfitAndLoss:       types.Number(0.6225),
				ProfitAndLossRatio:  types.Number(0.0166399358460306),
				InstrumentType:      instTypeSwap,
				Margin:              types.Number(37.41),
				Currency:            currency.USDT,
				MarkPrice:           types.Number(38203.4),
				UniqueCode:          "25CD5A80241D6FE6",
				ProfitSharingAmount: types.Number(0.0407967),
				ClosedSubPosition:   types.Number(2),
				CloseType:           "1",
			},
			{
				InstrumentID:        perpetualSwapPair.String(),
				SubPositionID:       "649750686292803585",
				PositionSide:        "net",
				MarginMode:          "isolated",
				Leverage:            types.Number(3),
				OpenAveragePrice:    types.Number(37410),
				OpenTime:            types.Time(time.UnixMilli(1701184635381)),
				SubPosition:         types.Number(2),
				CloseTime:           types.Time(time.UnixMilli(1701188587950)),
				CloseAveragePrice:   types.Number(37617.5),
				ProfitAndLoss:       types.Number(0.415),
				ProfitAndLossRatio:  types.Number(0.0166399358460306),
				InstrumentType:      instTypeSwap,
				Margin:              types.Number(24.94),
				Currency:            currency.USDT,
				MarkPrice:           types.Number(38203.4),
				UniqueCode:          "25CD5A80241D6FE6",
				ProfitSharingAmount: types.Number(0.0271978),
				ClosedSubPosition:   types.Number(2),
				CloseType:           "2",
			},
		}
		assert.Equal(t, exp, result, "GetLeadPositionHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetLeadPositionHistory should return a list")
	}
}

func TestLeadPositionsRequestValues(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *LeadPositionsRequest
		exp  url.Values
	}{
		{name: "no filter", arg: &LeadPositionsRequest{}, exp: url.Values{}},
		{name: "instrument type", arg: &LeadPositionsRequest{InstrumentType: instTypeSwap}, exp: url.Values{"instType": {"SWAP"}}},
		{name: "instrument ID", arg: &LeadPositionsRequest{InstrumentID: perpetualSwapPair.String()}, exp: url.Values{"instId": {"BTC-USDT-SWAP"}}},
		{name: "after", arg: &LeadPositionsRequest{After: "649750700213698562"}, exp: url.Values{"after": {"649750700213698562"}}},
		{name: "before", arg: &LeadPositionsRequest{Before: "649750700213698500"}, exp: url.Values{"before": {"649750700213698500"}}},
		{name: "limit", arg: &LeadPositionsRequest{Limit: 10}, exp: url.Values{"limit": {"10"}}},
		{
			name: "every filter",
			arg:  &LeadPositionsRequest{InstrumentType: instTypeSwap, InstrumentID: perpetualSwapPair.String(), After: "649750700213698562", Before: "649750700213698500", Limit: 10},
			exp:  url.Values{"instType": {"SWAP"}, "instId": {"BTC-USDT-SWAP"}, "after": {"649750700213698562"}, "before": {"649750700213698500"}, "limit": {"10"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, tc.arg.values(), "values should encode every set filter")
		})
	}
}

func TestPlaceLeadStopOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceLeadStopOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.PlaceLeadStopOrder(t.Context(), &PlaceLeadStopOrderRequest{})
	require.ErrorIs(t, err, errSubPositionIDRequired)
	_, err = e.PlaceLeadStopOrder(t.Context(), &PlaceLeadStopOrderRequest{Tag: "1235454"})
	require.ErrorIs(t, err, errSubPositionIDRequired)
	_, err = e.PlaceLeadStopOrder(t.Context(), &PlaceLeadStopOrderRequest{SubPositionID: "1235454"})
	require.ErrorIs(t, err, errTakeProfitOrStopLossTriggerPriceRequired)

	for _, tc := range []struct {
		name string
		req  *PlaceLeadStopOrderRequest
		body string
	}{
		{
			name: "every parameter",
			req: &PlaceLeadStopOrderRequest{
				InstrumentType:             instTypeSwap,
				SubPositionID:              "518560559046594560",
				TakeProfitTriggerPrice:     43000,
				StopLossTriggerPrice:       36000,
				TakeProfitOrderPrice:       -1,
				StopLossOrderPrice:         35900.5,
				TakeProfitTriggerPriceType: "mark",
				StopLossTriggerPriceType:   "index",
				Tag:                        "leadstop1",
			},
			body: `{"instType":"SWAP","subPosId":"518560559046594560","tpTriggerPx":"43000","slTriggerPx":"36000","tpOrdPx":"-1","slOrdPx":"35900.5","tpTriggerPxType":"mark","slTriggerPxType":"index","tag":"leadstop1"}`,
		},
		{
			name: "required parameters only",
			req:  &PlaceLeadStopOrderRequest{SubPositionID: "518560559046594560", StopLossTriggerPrice: 36000},
			body: `{"subPosId":"518560559046594560","slTriggerPx":"36000"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requireCopyTradingBody(t, "copytrading/algo-order", tc.body, []map[string]string{{"subPosId": tc.req.SubPositionID, "tag": tc.req.Tag}}, func(ex *Exchange) error {
				_, err := ex.PlaceLeadStopOrder(t.Context(), tc.req)
				return err
			})
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceLeadStopOrder(contextGenerate(), &PlaceLeadStopOrderRequest{
		SubPositionID:          "1235454",
		TakeProfitTriggerPrice: 123455,
	})
	require.NoError(t, err, "PlaceLeadStopOrder must not error")
	if mockTests {
		assert.Equal(t, &LeadPositionActionResponse{SubPositionID: "1235454"}, result, "PlaceLeadStopOrder should decode every field")
	} else {
		assert.NotEmpty(t, result.SubPositionID, "PlaceLeadStopOrder should return the lead position ID")
	}
}

func TestCloseLeadPosition(t *testing.T) {
	t.Parallel()
	_, err := e.CloseLeadPosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.CloseLeadPosition(t.Context(), &CloseLeadPositionRequest{})
	require.ErrorIs(t, err, errSubPositionIDRequired)
	_, err = e.CloseLeadPosition(t.Context(), &CloseLeadPositionRequest{Tag: "tag-here"})
	require.ErrorIs(t, err, errSubPositionIDRequired)

	for _, tc := range []struct {
		name string
		req  *CloseLeadPositionRequest
		body string
	}{
		{
			name: "every parameter",
			req:  &CloseLeadPositionRequest{InstrumentType: instTypeSpot, SubPositionID: "518541406042591232", OrderType: "limit", Price: 42000.5, Tag: "leadclose1"},
			body: `{"instType":"SPOT","subPosId":"518541406042591232","ordType":"limit","px":"42000.5","tag":"leadclose1"}`,
		},
		{
			name: "required parameters only",
			req:  &CloseLeadPositionRequest{SubPositionID: "518541406042591232"},
			body: `{"subPosId":"518541406042591232"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requireCopyTradingBody(t, "copytrading/close-subposition", tc.body, []map[string]string{{"subPosId": tc.req.SubPositionID, "tag": tc.req.Tag}}, func(ex *Exchange) error {
				_, err := ex.CloseLeadPosition(t.Context(), tc.req)
				return err
			})
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CloseLeadPosition(contextGenerate(), &CloseLeadPositionRequest{
		SubPositionID: "518541406042591232",
		Tag:           "copytag1",
	})
	require.NoError(t, err, "CloseLeadPosition must not error")
	if mockTests {
		assert.Equal(t, &LeadPositionActionResponse{SubPositionID: "518541406042591232", Tag: "copytag1"}, result, "CloseLeadPosition should decode every field")
	} else {
		assert.NotEmpty(t, result.SubPositionID, "CloseLeadPosition should return the lead position ID")
	}
}

func TestGetLeadingInstruments(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLeadingInstruments(contextGenerate(), instTypeSwap)
	require.NoError(t, err, "GetLeadingInstruments must not error")
	if mockTests {
		exp := []LeadingInstrumentResponse{
			{InstrumentID: "BTC-USDT-SWAP", Enabled: true},
			{InstrumentID: "ETH-USDT-SWAP", Enabled: true},
			{InstrumentID: "ADA-USDT-SWAP"},
		}
		assert.Equal(t, exp, result, "GetLeadingInstruments should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetLeadingInstruments should return the instruments")
	}
}

func TestAmendLeadingInstruments(t *testing.T) {
	t.Parallel()
	_, err := e.AmendLeadingInstruments(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.AmendLeadingInstruments(t.Context(), &AmendLeadingInstrumentsRequest{})
	require.ErrorIs(t, err, errMissingInstrumentID)

	for _, tc := range []struct {
		name string
		req  *AmendLeadingInstrumentsRequest
		body string
	}{
		{
			name: "every parameter",
			req:  &AmendLeadingInstrumentsRequest{InstrumentType: instTypeSwap, InstrumentID: "BTC-USDT-SWAP,ETH-USDT-SWAP"},
			body: `{"instType":"SWAP","instId":"BTC-USDT-SWAP,ETH-USDT-SWAP"}`,
		},
		{
			name: "required parameters only",
			req:  &AmendLeadingInstrumentsRequest{InstrumentID: "BTC-USDT-SWAP"},
			body: `{"instId":"BTC-USDT-SWAP"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requireCopyTradingBody(t, "copytrading/set-instruments", tc.body, []map[string]any{{"instId": "BTC-USDT-SWAP", "enabled": true}}, func(ex *Exchange) error {
				_, err := ex.AmendLeadingInstruments(t.Context(), tc.req)
				return err
			})
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AmendLeadingInstruments(contextGenerate(), &AmendLeadingInstrumentsRequest{InstrumentID: perpetualSwapPair.String()})
	require.NoError(t, err, "AmendLeadingInstruments must not error")
	if mockTests {
		assert.Equal(t, []LeadingInstrumentResponse{{InstrumentID: perpetualSwapPair.String(), Enabled: true}}, result, "AmendLeadingInstruments should decode every field")
	} else {
		assert.NotEmpty(t, result, "AmendLeadingInstruments should return the instruments")
	}
}

func TestGetProfitSharingDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetProfitSharingDetails(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetProfitSharingDetails(contextGenerate(), &ProfitSharingDetailsRequest{After: "149", Before: "19", Limit: 2})
	require.NoError(t, err, "GetProfitSharingDetails must not error")
	if mockTests {
		exp := []ProfitSharingDetailResponse{
			{
				Currency:            currency.USDT,
				ProfitSharingAmount: types.Number(0.00536),
				NickName:            "Potato",
				ProfitSharingID:     "148",
				InstrumentType:      instTypeSwap,
				PortraitLink:        "https://static.okx.com/cdn/okex/users/headimages/20231010/fd31f45e99fe41f7bb219c0b53ae0ada",
				Timestamp:           types.Time(time.UnixMilli(1723392000000)),
			},
			{
				Currency:            currency.USDT,
				ProfitSharingAmount: types.Number(0.00336),
				NickName:            "Apple",
				ProfitSharingID:     "20",
				InstrumentType:      instTypeSwap,
				Timestamp:           types.Time(time.UnixMilli(1723392000000)),
			},
		}
		assert.Equal(t, exp, result, "GetProfitSharingDetails should decode every field")
	} else {
		assert.NotNil(t, result, "GetProfitSharingDetails should return a list")
	}
}

func TestGetTotalProfitSharing(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTotalProfitSharing(contextGenerate(), instTypeSwap)
	require.NoError(t, err, "GetTotalProfitSharing must not error")
	if mockTests {
		exp := []TotalProfitSharingResponse{{Currency: currency.USDT, TotalProfitSharingAmount: types.Number(0.6584928), InstrumentType: instTypeSwap}}
		assert.Equal(t, exp, result, "GetTotalProfitSharing should decode every field")
	} else {
		assert.NotNil(t, result, "GetTotalProfitSharing should return a list")
	}
}

func TestGetUnrealisedProfitSharingDetails(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUnrealisedProfitSharingDetails(contextGenerate(), instTypeSwap)
	require.NoError(t, err, "GetUnrealisedProfitSharingDetails must not error")
	if mockTests {
		exp := []UnrealisedProfitSharingDetailResponse{
			{
				Currency:                      currency.USDT,
				UnrealisedProfitSharingAmount: types.Number(0.455472),
				NickName:                      "Potato",
				InstrumentType:                instTypeSwap,
				PortraitLink:                  "https://static.okx.com/cdn/okex/users/headimages/20231010/fd31f45e99fe41f7bb219c0b53ae0ada",
				Timestamp:                     types.Time(time.UnixMilli(1669901824779)),
			},
			{
				Currency:                      currency.USDT,
				UnrealisedProfitSharingAmount: types.Number(0.033608),
				NickName:                      "Apple",
				InstrumentType:                instTypeSwap,
				Timestamp:                     types.Time(time.UnixMilli(1669460210113)),
			},
		}
		assert.Equal(t, exp, result, "GetUnrealisedProfitSharingDetails should decode every field")
	} else {
		assert.NotNil(t, result, "GetUnrealisedProfitSharingDetails should return a list")
	}
}

func TestGetTotalUnrealisedProfitSharing(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTotalUnrealisedProfitSharing(contextGenerate(), instTypeSwap)
	require.NoError(t, err, "GetTotalUnrealisedProfitSharing must not error")
	if mockTests {
		exp := &TotalUnrealisedProfitSharingResponse{
			ProfitSharingTimestamp:             types.Time(time.UnixMilli(1705852800000)),
			TotalUnrealisedProfitSharingAmount: types.Number(0.114402985553185),
		}
		assert.Equal(t, exp, result, "GetTotalUnrealisedProfitSharing should decode every field")
	} else {
		assert.NotNil(t, result, "GetTotalUnrealisedProfitSharing should return the total")
	}
}

func TestAmendProfitSharingRatio(t *testing.T) {
	t.Parallel()
	_, err := e.AmendProfitSharingRatio(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.AmendProfitSharingRatio(t.Context(), &AmendProfitSharingRatioRequest{ProfitSharingRatio: -0.1})
	require.ErrorIs(t, err, errInvalidProfitSharingRatio)

	for _, tc := range []struct {
		name string
		req  *AmendProfitSharingRatioRequest
		body string
	}{
		{
			name: "every parameter",
			req:  &AmendProfitSharingRatioRequest{InstrumentType: instTypeSwap, ProfitSharingRatio: 0.1},
			body: `{"instType":"SWAP","profitSharingRatio":"0.1"}`,
		},
		{
			name: "required parameters only with a zero ratio",
			req:  &AmendProfitSharingRatioRequest{},
			body: `{"profitSharingRatio":"0"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requireCopyTradingBody(t, "copytrading/amend-profit-sharing-ratio", tc.body, []map[string]bool{{"result": true}}, func(ex *Exchange) error {
				_, err := ex.AmendProfitSharingRatio(t.Context(), tc.req)
				return err
			})
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AmendProfitSharingRatio(contextGenerate(), &AmendProfitSharingRatioRequest{InstrumentType: instTypeSwap, ProfitSharingRatio: 0.1})
	require.NoError(t, err, "AmendProfitSharingRatio must not error")
	assert.Equal(t, &CopyTradingResultResponse{Result: true}, result, "AmendProfitSharingRatio should report the result")
}

func TestGetCopyTradingAccountConfiguration(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCopyTradingAccountConfiguration(contextGenerate())
	require.NoError(t, err, "GetCopyTradingAccountConfiguration must not error")
	if mockTests {
		exp := &CopyTradingAccountConfigurationResponse{
			UniqueCode:   "5506D3681454A304",
			NickName:     "155***9957",
			PortraitLink: "https://static.okx.com/cdn/okex/users/headimages/20231010/fd31f45e99fe41f7bb219c0b53ae0ada",
			Details: []CopyTradingRole{
				{InstrumentType: instTypeSwap, RoleType: "1", ProfitSharingRatio: types.Number(0.1), MaxCopyTraderNumber: types.Number(100), CopyTraderNumber: types.Number(1)},
				{InstrumentType: instTypeSpot, RoleType: "0"},
			},
		}
		assert.Equal(t, exp, result, "GetCopyTradingAccountConfiguration should decode every field")
	} else {
		assert.NotEmpty(t, result.UniqueCode, "GetCopyTradingAccountConfiguration should return the account's unique code")
	}
}

// copySettingsValidationCases holds the copy settings requests validateCopySettings rejects, with their errors
var copySettingsValidationCases = []struct {
	name string
	req  *CopySettingsRequest
	err  error
}{
	{name: "nil request", err: common.ErrNilPointer},
	{name: "empty request", req: &CopySettingsRequest{}, err: errUniqueCodeRequired},
	{name: "missing unique code", req: &CopySettingsRequest{CopyMode: copyModeRatioCopy}, err: errUniqueCodeRequired},
	{name: "missing copy margin mode", req: &CopySettingsRequest{CopyMode: copyModeRatioCopy, UniqueCode: "25CD5A80241D6FE6"}, err: errCopyMarginModeRequired},
	{name: "missing copy instrument ID type", req: &CopySettingsRequest{CopyMode: copyModeRatioCopy, UniqueCode: "25CD5A80241D6FE6", CopyMarginMode: "cross"}, err: errCopyInstrumentIDTypeRequired},
	{name: "custom copy instruments without instrument IDs", req: &CopySettingsRequest{CopyMode: copyModeRatioCopy, UniqueCode: "25CD5A80241D6FE6", CopyMarginMode: "cross", CopyInstrumentIDType: copyInstrumentIDTypeCustom}, err: errMissingInstrumentID},
	{name: "missing copy total amount", req: &CopySettingsRequest{CopyMode: copyModeRatioCopy, UniqueCode: "25CD5A80241D6FE6", CopyMarginMode: "cross", CopyInstrumentIDType: "copy"}, err: limits.ErrAmountBelowMin},
	{name: "default fixed amount mode without copy amount", req: &CopySettingsRequest{UniqueCode: "25CD5A80241D6FE6", CopyMarginMode: "cross", CopyInstrumentIDType: "copy", CopyTotalAmount: 500}, err: errCopyAmountRequired},
	{name: "fixed amount mode without copy amount", req: &CopySettingsRequest{CopyMode: copyModeFixedAmount, UniqueCode: "25CD5A80241D6FE6", CopyMarginMode: "cross", CopyInstrumentIDType: "copy", CopyTotalAmount: 500}, err: errCopyAmountRequired},
	{name: "ratio copy mode without copy ratio", req: &CopySettingsRequest{CopyMode: copyModeRatioCopy, UniqueCode: "25CD5A80241D6FE6", CopyMarginMode: "cross", CopyInstrumentIDType: "copy", CopyTotalAmount: 500}, err: errCopyRatioRequired},
	{name: "missing sub position close type", req: &CopySettingsRequest{CopyMode: copyModeRatioCopy, UniqueCode: "25CD5A80241D6FE6", CopyMarginMode: "cross", CopyInstrumentIDType: "copy", CopyTotalAmount: 500, CopyRatio: 1}, err: errSubPositionCloseTypeRequired},
}

// copySettingsBodyCases holds copy settings requests and the bodies SetFirstCopySettings and AmendCopySettings send
var copySettingsBodyCases = []struct {
	name string
	req  *CopySettingsRequest
	body string
}{
	{
		name: "every parameter",
		req: &CopySettingsRequest{
			InstrumentType:       instTypeSwap,
			UniqueCode:           "25CD5A80241D6FE6",
			CopyMarginMode:       "isolated",
			CopyInstrumentIDType: copyInstrumentIDTypeCustom,
			InstrumentID:         "BTC-USDT-SWAP,ETH-USDT-SWAP",
			CopyMode:             copyModeFixedAmount,
			CopyTotalAmount:      500,
			CopyAmount:           20,
			CopyRatio:            0.5,
			TakeProfitRatio:      0.1,
			StopLossRatio:        0.05,
			StopLossTotalAmount:  100,
			SubPositionCloseType: "market_close",
			Tag:                  "copytag1",
		},
		body: `{"instType":"SWAP","uniqueCode":"25CD5A80241D6FE6","copyMgnMode":"isolated","copyInstIdType":"custom","instId":"BTC-USDT-SWAP,ETH-USDT-SWAP","copyMode":"fixed_amount","copyTotalAmt":"500","copyAmt":"20","copyRatio":"0.5","tpRatio":"0.1","slRatio":"0.05","slTotalAmt":"100","subPosCloseType":"market_close","tag":"copytag1"}`,
	},
	{
		name: "required parameters only",
		req:  &CopySettingsRequest{UniqueCode: "25CD5A80241D6FE6", CopyMarginMode: "cross", CopyInstrumentIDType: "copy", CopyTotalAmount: 500, CopyAmount: 20, SubPositionCloseType: "copy_close"},
		body: `{"uniqueCode":"25CD5A80241D6FE6","copyMgnMode":"cross","copyInstIdType":"copy","copyTotalAmt":"500","copyAmt":"20","subPosCloseType":"copy_close"}`,
	},
}

func TestValidateCopySettings(t *testing.T) {
	t.Parallel()
	for _, tc := range copySettingsValidationCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validateCopySettings(tc.req), tc.err, "validateCopySettings should return the expected error")
		})
	}
	for _, tc := range copySettingsBodyCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, validateCopySettings(tc.req), "validateCopySettings should accept the request")
		})
	}
}

func TestSetFirstCopySettings(t *testing.T) {
	t.Parallel()
	for _, tc := range copySettingsValidationCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.SetFirstCopySettings(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "SetFirstCopySettings should return the expected error")
		})
	}
	for _, tc := range copySettingsBodyCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requireCopyTradingBody(t, "copytrading/first-copy-settings", tc.body, []map[string]bool{{"result": true}}, func(ex *Exchange) error {
				_, err := ex.SetFirstCopySettings(t.Context(), tc.req)
				return err
			})
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetFirstCopySettings(contextGenerate(), &CopySettingsRequest{
		InstrumentType:       instTypeSwap,
		UniqueCode:           "25CD5A80241D6FE6",
		CopyMarginMode:       "cross",
		CopyInstrumentIDType: "copy",
		CopyMode:             copyModeRatioCopy,
		CopyRatio:            1,
		CopyTotalAmount:      500,
		SubPositionCloseType: "copy_close",
	})
	require.NoError(t, err, "SetFirstCopySettings must not error")
	assert.Equal(t, &CopyTradingResultResponse{Result: true}, result, "SetFirstCopySettings should report the result")
}

func TestAmendCopySettings(t *testing.T) {
	t.Parallel()
	for _, tc := range copySettingsValidationCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AmendCopySettings(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "AmendCopySettings should return the expected error")
		})
	}
	for _, tc := range copySettingsBodyCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requireCopyTradingBody(t, "copytrading/amend-copy-settings", tc.body, []map[string]bool{{"result": true}}, func(ex *Exchange) error {
				_, err := ex.AmendCopySettings(t.Context(), tc.req)
				return err
			})
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AmendCopySettings(contextGenerate(), &CopySettingsRequest{
		InstrumentType:       instTypeSwap,
		UniqueCode:           "25CD5A80241D6FE6",
		CopyMarginMode:       "cross",
		CopyInstrumentIDType: "copy",
		CopyMode:             copyModeRatioCopy,
		CopyRatio:            1,
		CopyTotalAmount:      500,
		SubPositionCloseType: "copy_close",
	})
	require.NoError(t, err, "AmendCopySettings must not error")
	assert.Equal(t, &CopyTradingResultResponse{Result: true}, result, "AmendCopySettings should report the result")
}

func TestStopCopying(t *testing.T) {
	t.Parallel()
	_, err := e.StopCopying(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.StopCopying(t.Context(), &StopCopyingRequest{})
	require.ErrorIs(t, err, errUniqueCodeRequired)
	_, err = e.StopCopying(t.Context(), &StopCopyingRequest{
		InstrumentType:       instTypeSwap,
		SubPositionCloseType: "manual_close",
	})
	require.ErrorIs(t, err, errUniqueCodeRequired)
	_, err = e.StopCopying(t.Context(), &StopCopyingRequest{
		InstrumentType: instTypeSwap,
		UniqueCode:     "25CD5A80241D6FE6",
	})
	require.ErrorIs(t, err, errSubPositionCloseTypeRequired)

	for _, tc := range []struct {
		name string
		req  *StopCopyingRequest
		body string
	}{
		{
			name: "every parameter",
			req:  &StopCopyingRequest{InstrumentType: instTypeSwap, UniqueCode: "25CD5A80241D6FE6", SubPositionCloseType: "market_close"},
			body: `{"instType":"SWAP","uniqueCode":"25CD5A80241D6FE6","subPosCloseType":"market_close"}`,
		},
		{
			name: "required parameters only",
			req:  &StopCopyingRequest{UniqueCode: "25CD5A80241D6FE6", SubPositionCloseType: "copy_close"},
			body: `{"uniqueCode":"25CD5A80241D6FE6","subPosCloseType":"copy_close"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requireCopyTradingBody(t, "copytrading/stop-copy-trading", tc.body, []map[string]bool{{"result": true}}, func(ex *Exchange) error {
				_, err := ex.StopCopying(t.Context(), tc.req)
				return err
			})
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.StopCopying(contextGenerate(), &StopCopyingRequest{
		InstrumentType:       instTypeSwap,
		UniqueCode:           "25CD5A80241D6FE6",
		SubPositionCloseType: "manual_close",
	})
	require.NoError(t, err, "StopCopying must not error")
	assert.Equal(t, &CopyTradingResultResponse{Result: true}, result, "StopCopying should report the result")
}

func TestGetCopySettings(t *testing.T) {
	t.Parallel()
	_, err := e.GetCopySettings(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetCopySettings(t.Context(), &LeadTraderRequest{InstrumentType: instTypeSwap})
	require.ErrorIs(t, err, errUniqueCodeRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCopySettings(contextGenerate(), &LeadTraderRequest{InstrumentType: instTypeSwap, UniqueCode: "213E8C92DC61EFAC"})
	require.NoError(t, err, "GetCopySettings must not error")
	if mockTests {
		exp := &CopySettingsResponse{
			CopyMode:             copyModeRatioCopy,
			CopyAmount:           types.Number(20),
			CopyRatio:            types.Number(1),
			CopyTotalAmount:      types.Number(500),
			TakeProfitRatio:      types.Number(0.8),
			StopLossRatio:        types.Number(0.5),
			CopyInstrumentIDType: "copy",
			InstrumentIDs: []CopySettingsInstrument{
				{InstrumentID: "ADA-USDT-SWAP", Enabled: true},
				{InstrumentID: "YFII-USDT-SWAP"},
			},
			StopLossTotalAmount:  types.Number(100),
			SubPositionCloseType: "copy_close",
			CopyMarginMode:       "isolated",
			Currency:             currency.USDT,
			CopyState:            "1",
			Tag:                  "copytag1",
		}
		assert.Equal(t, exp, result, "GetCopySettings should decode every field")
	} else {
		assert.NotNil(t, result, "GetCopySettings should return the settings")
	}
}

func TestLeadTraderRequestValues(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *LeadTraderRequest
		exp  url.Values
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing unique code", arg: &LeadTraderRequest{InstrumentType: instTypeSwap}, err: errUniqueCodeRequired},
		{name: "unique code only", arg: &LeadTraderRequest{UniqueCode: "25CD5A80241D6FE6"}, exp: url.Values{"uniqueCode": {"25CD5A80241D6FE6"}}},
		{name: "every parameter", arg: &LeadTraderRequest{UniqueCode: "25CD5A80241D6FE6", InstrumentType: instTypeSwap}, exp: url.Values{"uniqueCode": {"25CD5A80241D6FE6"}, "instType": {"SWAP"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := tc.arg.values()
			require.ErrorIs(t, err, tc.err, "values must return the expected error")
			assert.Equal(t, tc.exp, params, "values should encode every set parameter")
		})
	}
}

func TestGetMyLeadTraders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMyLeadTraders(contextGenerate(), instTypeSwap)
	require.NoError(t, err, "GetMyLeadTraders must not error")
	if mockTests {
		exp := []MyLeadTraderResponse{{
			PortraitLink:            "https://static.okx.com/cdn/okex/users/headimages/20231010/fd31f45e99fe41f7bb219c0b53ae0ada",
			NickName:                "Trader9527",
			Margin:                  types.Number(1.89395),
			CopyTotalAmount:         types.Number(500),
			CopyTotalProfitAndLoss:  types.Number(1.5),
			UniqueCode:              "25CD5A80241D6FE6",
			Currency:                currency.USDT,
			ProfitSharingRatio:      types.Number(0.08),
			BeginCopyTime:           types.Time(time.UnixMilli(1701224821936)),
			UnrealisedProfitAndLoss: types.Number(-0.3),
			TodayProfitAndLoss:      types.Number(0.25),
			LeadMode:                "public",
		}}
		assert.Equal(t, exp, result, "GetMyLeadTraders should decode every field")
	} else {
		assert.NotNil(t, result, "GetMyLeadTraders should return a list")
	}
}

func TestGetCopyTradingConfiguration(t *testing.T) {
	t.Parallel()
	result, err := e.GetCopyTradingConfiguration(contextGenerate(), instTypeSwap)
	require.NoError(t, err, "GetCopyTradingConfiguration must not error")
	if mockTests {
		exp := &CopyTradingConfigurationResponse{
			MaxCopyAmount:      types.Number(100000),
			MinCopyAmount:      types.Number(10),
			MaxCopyTotalAmount: types.Number(2000000),
			MinCopyRatio:       types.Number(0.01),
			MaxCopyRatio:       types.Number(100),
			MaxTakeProfitRatio: types.Number(1.5),
			MaxStopLossRatio:   types.Number(0.75),
		}
		assert.Equal(t, exp, result, "GetCopyTradingConfiguration should decode every field")
	} else {
		assert.Positive(t, result.MaxCopyAmount.Float64(), "GetCopyTradingConfiguration should return the maximum copy amount")
	}
}

func TestGetLeadTraderRanks(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeadTraderRanks(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	result, err := e.GetLeadTraderRanks(contextGenerate(), &LeadTraderRanksRequest{
		InstrumentType:           instTypeSwap,
		SortType:                 "aum",
		HasVacancy:               true,
		MinLeadDays:              2,
		MinAssets:                100,
		MaxAssets:                10000000,
		MinAssetsUnderManagement: 1000,
		MaxAssetsUnderManagement: 100000000,
		DataVersion:              "20261123110001",
		Page:                     1,
		Limit:                    2,
	})
	require.NoError(t, err, "GetLeadTraderRanks must not error")
	if mockTests {
		exp := &LeadTraderRanksResponse{
			DataVersion: "20261123110001",
			TotalPage:   types.Number(31),
			Ranks: []LeadTraderRank{
				{
					AssetsUnderManagement:       types.Number(1271682.8642544275204409),
					CopyState:                   "0",
					MaxCopyTraderNumber:         types.Number(300),
					CopyTraderNumber:            types.Number(43),
					AccumulatedCopyTraderNumber: types.Number(247),
					PortraitLink:                "https://static.okx.com/cdn/okex/users/headimages/20250225/3894a47f5c4045dabf4474a695dd0e89",
					NickName:                    "kxiandaoyan",
					Currency:                    currency.USDT,
					UniqueCode:                  "C419F165132C0C57",
					WinRatio:                    types.Number(0.4778),
					LeadDays:                    types.Number(489),
					TraderInstruments:           []string{"SOL-USDT-SWAP", "BTC-USDT-SWAP"},
					ProfitAndLoss:               types.Number(-54488.636097714201674),
					ProfitAndLossRatio:          types.Number(-0.6784),
					ProfitAndLossRatios: []LeadTraderProfitAndLossRatio{
						{BeginTimestamp: types.Time(time.UnixMilli(1795363200000)), ProfitAndLossRatio: types.Number(-0.6784)},
						{BeginTimestamp: types.Time(time.UnixMilli(1794931200000)), ProfitAndLossRatio: types.Number(-0.6121)},
					},
				},
				{
					AssetsUnderManagement:       types.Number(776437.3136628856254239),
					CopyState:                   "0",
					MaxCopyTraderNumber:         types.Number(1200),
					CopyTraderNumber:            types.Number(669),
					AccumulatedCopyTraderNumber: types.Number(10827),
					PortraitLink:                "https://static.okx.com/cdn/okex/users/headimages/20220717/22d0e209ae55427ba0c4c878f9778500",
					NickName:                    "To the Moon Merchant",
					Currency:                    currency.USDT,
					UniqueCode:                  "9C2CB3B2306B28EA",
					WinRatio:                    types.Number(0.5244),
					LeadDays:                    types.Number(1178),
					TraderInstruments:           []string{"AVAX-USDT-SWAP", "MU-USDT-SWAP"},
					ProfitAndLoss:               types.Number(-74839.6117401909603075),
					ProfitAndLossRatio:          types.Number(-0.3996),
					ProfitAndLossRatios: []LeadTraderProfitAndLossRatio{
						{BeginTimestamp: types.Time(time.UnixMilli(1795363200000)), ProfitAndLossRatio: types.Number(-0.4112)},
						{BeginTimestamp: types.Time(time.UnixMilli(1794931200000)), ProfitAndLossRatio: types.Number(-0.2138)},
					},
					TraditionalFinanceTrader: true,
				},
			},
		}
		assert.Equal(t, exp, result, "GetLeadTraderRanks should decode every field")
	} else {
		assert.NotEmpty(t, result.Ranks, "GetLeadTraderRanks should return lead traders")
	}
}

func TestGetLeadTraderWeeklyProfitAndLoss(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeadTraderWeeklyProfitAndLoss(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetLeadTraderWeeklyProfitAndLoss(t.Context(), &LeadTraderRequest{})
	require.ErrorIs(t, err, errUniqueCodeRequired)

	require.NoError(t, syncLeadTraderUniqueID(t), "syncLeadTraderUniqueID must not error")
	result, err := e.GetLeadTraderWeeklyProfitAndLoss(contextGenerate(), &LeadTraderRequest{UniqueCode: leadTraderUniqueID})
	require.NoError(t, err, "GetLeadTraderWeeklyProfitAndLoss must not error")
	if mockTests {
		exp := []LeadTraderProfitAndLossResponse{
			{BeginTimestamp: types.Time(time.UnixMilli(1795363200000)), ProfitAndLoss: types.Number(10995.996799999975), ProfitAndLossRatio: types.Number(0.0185)},
			{BeginTimestamp: types.Time(time.UnixMilli(1794758400000)), ProfitAndLoss: types.Number(91123.7601331987467240), ProfitAndLossRatio: types.Number(0.1853)},
		}
		assert.Equal(t, exp, result, "GetLeadTraderWeeklyProfitAndLoss should decode every field")
	} else {
		assert.NotNil(t, result, "GetLeadTraderWeeklyProfitAndLoss should return a list")
	}
}

func TestGetLeadTraderDailyProfitAndLoss(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeadTraderDailyProfitAndLoss(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetLeadTraderDailyProfitAndLoss(t.Context(), &LeadTraderPerformanceRequest{InstrumentType: instTypeSwap, LastDays: "2"})
	require.ErrorIs(t, err, errUniqueCodeRequired)
	_, err = e.GetLeadTraderDailyProfitAndLoss(t.Context(), &LeadTraderPerformanceRequest{InstrumentType: instTypeSwap, UniqueCode: "WOOF"})
	require.ErrorIs(t, err, errLastDaysRequired)

	require.NoError(t, syncLeadTraderUniqueID(t), "syncLeadTraderUniqueID must not error")
	result, err := e.GetLeadTraderDailyProfitAndLoss(contextGenerate(), &LeadTraderPerformanceRequest{InstrumentType: instTypeSwap, UniqueCode: leadTraderUniqueID, LastDays: "2"})
	require.NoError(t, err, "GetLeadTraderDailyProfitAndLoss must not error")
	if mockTests {
		exp := []LeadTraderProfitAndLossResponse{
			{BeginTimestamp: types.Time(time.UnixMilli(1795363200000)), ProfitAndLoss: types.Number(478378.3278409821676115), ProfitAndLossRatio: types.Number(3.8572)},
			{BeginTimestamp: types.Time(time.UnixMilli(1795276800000)), ProfitAndLoss: types.Number(505336.3550409821926115), ProfitAndLossRatio: types.Number(4.0746)},
		}
		assert.Equal(t, exp, result, "GetLeadTraderDailyProfitAndLoss should decode every field")
	} else {
		assert.NotNil(t, result, "GetLeadTraderDailyProfitAndLoss should return a list")
	}
}

func TestGetLeadTraderStats(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeadTraderStats(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetLeadTraderStats(t.Context(), &LeadTraderPerformanceRequest{InstrumentType: instTypeSwap, LastDays: "2"})
	require.ErrorIs(t, err, errUniqueCodeRequired)
	_, err = e.GetLeadTraderStats(t.Context(), &LeadTraderPerformanceRequest{InstrumentType: instTypeSwap, UniqueCode: "RAWR"})
	require.ErrorIs(t, err, errLastDaysRequired)

	require.NoError(t, syncLeadTraderUniqueID(t), "syncLeadTraderUniqueID must not error")
	result, err := e.GetLeadTraderStats(contextGenerate(), &LeadTraderPerformanceRequest{InstrumentType: instTypeSwap, UniqueCode: leadTraderUniqueID, LastDays: "2"})
	require.NoError(t, err, "GetLeadTraderStats must not error")
	if mockTests {
		exp := &LeadTraderStatsResponse{
			WinRatio:                       types.Number(0.4667),
			ProfitDays:                     types.Number(14),
			LossDays:                       types.Number(16),
			CurrentCopyTraderProfitAndLoss: types.Number(331.3616121354253194),
			AverageSubPositionNotional:     types.Number(176486.9828),
			InvestmentAmount:               types.Number(124021.6495764985906101),
			Currency:                       currency.USDT,
		}
		assert.Equal(t, exp, result, "GetLeadTraderStats should decode every field")
	} else {
		assert.NotNil(t, result, "GetLeadTraderStats should return the stats")
	}
}

func TestLeadTraderPerformanceRequestValues(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *LeadTraderPerformanceRequest
		exp  url.Values
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing unique code", arg: &LeadTraderPerformanceRequest{LastDays: "1"}, err: errUniqueCodeRequired},
		{name: "missing last days", arg: &LeadTraderPerformanceRequest{UniqueCode: "25CD5A80241D6FE6"}, err: errLastDaysRequired},
		{name: "required parameters only", arg: &LeadTraderPerformanceRequest{UniqueCode: "25CD5A80241D6FE6", LastDays: "1"}, exp: url.Values{"uniqueCode": {"25CD5A80241D6FE6"}, "lastDays": {"1"}}},
		{
			name: "every parameter",
			arg:  &LeadTraderPerformanceRequest{UniqueCode: "25CD5A80241D6FE6", LastDays: "4", InstrumentType: instTypeSwap},
			exp:  url.Values{"uniqueCode": {"25CD5A80241D6FE6"}, "lastDays": {"4"}, "instType": {"SWAP"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := tc.arg.values()
			require.ErrorIs(t, err, tc.err, "values must return the expected error")
			assert.Equal(t, tc.exp, params, "values should encode every set parameter")
		})
	}
}

func TestGetLeadTraderCurrencyPreferences(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeadTraderCurrencyPreferences(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetLeadTraderCurrencyPreferences(t.Context(), &LeadTraderRequest{InstrumentType: instTypeSwap})
	require.ErrorIs(t, err, errUniqueCodeRequired)

	require.NoError(t, syncLeadTraderUniqueID(t), "syncLeadTraderUniqueID must not error")
	result, err := e.GetLeadTraderCurrencyPreferences(contextGenerate(), &LeadTraderRequest{InstrumentType: instTypeSwap, UniqueCode: leadTraderUniqueID})
	require.NoError(t, err, "GetLeadTraderCurrencyPreferences must not error")
	if mockTests {
		exp := []LeadTraderCurrencyPreferenceResponse{
			{Currency: currency.ETH, Ratio: types.Number(0.744)},
			{Currency: currency.DOGE, Ratio: types.Number(0.123)},
			{Currency: currency.SOL, Ratio: types.Number(0.0575)},
			{Currency: currency.NewCode("Other"), Ratio: types.Number(0.03380000000000005)},
			{Currency: currency.BTC, Ratio: types.Number(0.0238)},
			{Currency: currency.OKB, Ratio: types.Number(0.0179)},
		}
		assert.Equal(t, exp, result, "GetLeadTraderCurrencyPreferences should decode every field")
	} else {
		assert.NotNil(t, result, "GetLeadTraderCurrencyPreferences should return a list")
	}
}

func TestGetLeadTraderCurrentLeadPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeadTraderCurrentLeadPositions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetLeadTraderCurrentLeadPositions(t.Context(), &LeadTraderPositionsRequest{InstrumentType: instTypeSwap, Limit: 10})
	require.ErrorIs(t, err, errUniqueCodeRequired)

	if err := syncLeadTraderSubPositionsUniqueID(t); err != nil {
		t.Skipf("OKX did not expose a lead trader with working current/history subposition endpoints: %v", err)
	}
	result, err := e.GetLeadTraderCurrentLeadPositions(contextGenerate(), &LeadTraderPositionsRequest{InstrumentType: instTypeSwap, UniqueCode: leadTraderSubPositionsUniqueID, Limit: 10})
	require.NoError(t, err, "GetLeadTraderCurrentLeadPositions must not error")
	if mockTests {
		exp := []LeadPositionResponse{
			{
				InstrumentID:                 "XRP-USDT-SWAP",
				SubPositionID:                "4123748858336980992",
				PositionSide:                 "short",
				MarginMode:                   "cross",
				Leverage:                     types.Number(100),
				OpenAveragePrice:             types.Number(0.9168123471999999),
				OpenTime:                     types.Time(time.UnixMilli(1795399692922)),
				SubPosition:                  types.Number(2500),
				InstrumentType:               instTypeSwap,
				Margin:                       types.Number(2292.03086799999975),
				UnrealisedProfitAndLoss:      types.Number(-3921.913200000025),
				UnrealisedProfitAndLossRatio: types.Number(-1.71110836889481),
				MarkPrice:                    types.Number(0.9325),
				UniqueCode:                   "CEB867A1A00B6682",
				MarginCurrency:               currency.USDT,
			},
			{
				InstrumentID:                 "SOL-USDT-SWAP",
				SubPositionID:                "4121550235318693888",
				PositionSide:                 "short",
				MarginMode:                   "cross",
				Leverage:                     types.Number(100),
				OpenAveragePrice:             types.Number(69.20811064),
				OpenTime:                     types.Time(time.UnixMilli(1795334168850)),
				SubPosition:                  types.Number(5000),
				InstrumentType:               instTypeSwap,
				Margin:                       types.Number(3460.405532),
				UnrealisedProfitAndLoss:      types.Number(-10659.4468),
				UnrealisedProfitAndLossRatio: types.Number(-3.08040392995187),
				MarkPrice:                    types.Number(71.34),
				UniqueCode:                   "CEB867A1A00B6682",
				MarginCurrency:               currency.USDT,
			},
		}
		assert.Equal(t, exp, result, "GetLeadTraderCurrentLeadPositions should decode every field")
	}
	// No live test validation of positions performed as the lead trader may not have any positions open
}

func TestGetLeadTraderLeadPositionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeadTraderLeadPositionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetLeadTraderLeadPositionHistory(t.Context(), &LeadTraderPositionsRequest{InstrumentType: instTypeSwap, Limit: 10})
	require.ErrorIs(t, err, errUniqueCodeRequired)

	if err := syncLeadTraderSubPositionsUniqueID(t); err != nil {
		t.Skipf("OKX did not expose a lead trader with working current/history subposition endpoints: %v", err)
	}
	result, err := e.GetLeadTraderLeadPositionHistory(contextGenerate(), &LeadTraderPositionsRequest{InstrumentType: instTypeSwap, UniqueCode: leadTraderSubPositionsUniqueID, Limit: 10})
	require.NoError(t, err, "GetLeadTraderLeadPositionHistory must not error")
	if mockTests {
		exp := []LeadPositionHistoryResponse{
			{
				InstrumentID:       "PIEVERSE-USDT-SWAP",
				SubPositionID:      "4121760231167963137",
				PositionSide:       "short",
				MarginMode:         "isolated",
				Leverage:           types.Number(10),
				OpenAveragePrice:   types.Number(4.9022),
				OpenTime:           types.Time(time.UnixMilli(1795340427218)),
				SubPosition:        types.Number(625),
				CloseTime:          types.Time(time.UnixMilli(1795342353699)),
				CloseAveragePrice:  types.Number(5.2561336),
				ProfitAndLoss:      types.Number(-2228.5071131),
				ProfitAndLossRatio: types.Number(-0.7273492270735588),
				InstrumentType:     instTypeSwap,
				Margin:             types.Number(3063.875),
				Currency:           currency.USDT,
				UniqueCode:         "CEB867A1A00B6682",
			},
			{
				InstrumentID:       "MMT-USDT-SWAP",
				SubPositionID:      "4110105491732508672",
				PositionSide:       "short",
				MarginMode:         "isolated",
				Leverage:           types.Number(10),
				OpenAveragePrice:   types.Number(0.2561674117647058),
				OpenTime:           types.Time(time.UnixMilli(1794993088914)),
				SubPosition:        types.Number(4250),
				CloseTime:          types.Time(time.UnixMilli(1795101919229)),
				CloseAveragePrice:  types.Number(0.2789309411764705),
				ProfitAndLoss:      types.Number(-979.8625152108254486),
				ProfitAndLossRatio: types.Number(-0.9000203591225277),
				InstrumentType:     instTypeSwap,
				Margin:             types.Number(1088.71149999999965),
				Currency:           currency.USDT,
				UniqueCode:         "CEB867A1A00B6682",
			},
		}
		assert.Equal(t, exp, result, "GetLeadTraderLeadPositionHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetLeadTraderLeadPositionHistory should return a list")
	}
}

func TestLeadTraderPositionsRequestValues(t *testing.T) {
	t.Parallel()
	uniqueCode := url.Values{"uniqueCode": {"25CD5A80241D6FE6"}}
	with := func(key, value string) url.Values {
		params := url.Values{"uniqueCode": {"25CD5A80241D6FE6"}}
		params.Set(key, value)
		return params
	}
	for _, tc := range []struct {
		name string
		arg  *LeadTraderPositionsRequest
		exp  url.Values
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing unique code", arg: &LeadTraderPositionsRequest{InstrumentType: instTypeSwap, Limit: 10}, err: errUniqueCodeRequired},
		{name: "unique code only", arg: &LeadTraderPositionsRequest{UniqueCode: "25CD5A80241D6FE6"}, exp: uniqueCode},
		{name: "instrument type", arg: &LeadTraderPositionsRequest{UniqueCode: "25CD5A80241D6FE6", InstrumentType: instTypeSwap}, exp: with("instType", "SWAP")},
		{name: "after", arg: &LeadTraderPositionsRequest{UniqueCode: "25CD5A80241D6FE6", After: "649750700213698562"}, exp: with("after", "649750700213698562")},
		{name: "before", arg: &LeadTraderPositionsRequest{UniqueCode: "25CD5A80241D6FE6", Before: "649750700213698500"}, exp: with("before", "649750700213698500")},
		{name: "limit", arg: &LeadTraderPositionsRequest{UniqueCode: "25CD5A80241D6FE6", Limit: 10}, exp: with("limit", "10")},
		{
			name: "every parameter",
			arg:  &LeadTraderPositionsRequest{UniqueCode: "25CD5A80241D6FE6", InstrumentType: instTypeSwap, After: "649750700213698562", Before: "649750700213698500", Limit: 10},
			exp:  url.Values{"uniqueCode": {"25CD5A80241D6FE6"}, "instType": {"SWAP"}, "after": {"649750700213698562"}, "before": {"649750700213698500"}, "limit": {"10"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := tc.arg.values()
			require.ErrorIs(t, err, tc.err, "values must return the expected error")
			assert.Equal(t, tc.exp, params, "values should encode every set parameter")
		})
	}
}

func TestGetCopyTraders(t *testing.T) {
	t.Parallel()
	_, err := e.GetCopyTraders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetCopyTraders(t.Context(), &CopyTradersRequest{InstrumentType: instTypeSwap})
	require.ErrorIs(t, err, errUniqueCodeRequired)

	require.NoError(t, syncLeadTraderUniqueID(t), "syncLeadTraderUniqueID must not error")
	result, err := e.GetCopyTraders(contextGenerate(), &CopyTradersRequest{InstrumentType: instTypeSwap, UniqueCode: leadTraderUniqueID, Limit: 3})
	require.NoError(t, err, "GetCopyTraders must not error")
	if mockTests {
		exp := &CopyTradersResponse{
			CopyTotalProfitAndLoss:      types.Number(-1025.98587123792816),
			Currency:                    currency.USDT,
			CopyTraderNumberChange:      types.Number(220),
			CopyTraderNumberChangeRatio: types.Number(3.4921),
			CopyTraders: []CopyTrader{
				{
					BeginCopyTime: types.Time(time.UnixMilli(1794430968000)),
					NickName:      "Vital2111",
					PortraitLink:  "https://static.okx.com/cdn/okex/users/headimages/predefined/0028.png",
					ProfitAndLoss: types.Number(685.2177703402046336),
				},
				{
					BeginCopyTime: types.Time(time.UnixMilli(1795087193000)),
					PortraitLink:  "https://static.okx.com/cdn/okex/users/headimages/predefined/0039.png",
					ProfitAndLoss: types.Number(352.9227838613325306),
				},
			},
		}
		assert.Equal(t, exp, result, "GetCopyTraders should decode every field")
	} else {
		assert.NotNil(t, result, "GetCopyTraders should return the copy traders")
	}
}
