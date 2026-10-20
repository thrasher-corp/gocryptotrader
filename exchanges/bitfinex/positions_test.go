package bitfinex

import (
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// newBodyCheckExchange returns an exchange whose REST requests must carry bodies, in order, each answered with
// response, for request shapes a recorded fixture cannot show, such as the exact text of a number
func newBodyCheckExchange(t *testing.T, response string, bodies ...string) *Exchange {
	t.Helper()
	var mu sync.Mutex
	var requests int
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, len(bodies), requests, "Every expected request should be sent")
	})
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "Reading the request body should not error")
		mu.Lock()
		if assert.Less(t, requests, len(bodies), "Only the expected requests should be sent") {
			assert.Equal(t, bodies[requests], string(body), "The request should carry the expected body")
		}
		requests++
		mu.Unlock()
		_, err = w.Write([]byte(response))
		assert.NoError(t, err, "Writing the response should not error")
	}))
}

// activePositionOfType returns one of the account's active positions of a type, skipping a live test when it has none
func activePositionOfType(t *testing.T, positionType uint64) *PositionResponse {
	t.Helper()
	positions, err := e.GetActivePositions(t.Context())
	require.NoError(t, err, "GetActivePositions must not error")
	for i := range positions {
		if positions[i].Type == positionType {
			return &positions[i]
		}
	}
	t.Skipf("The account has no active position of type %d", positionType)
	return nil
}

func TestGetMarginInfoBase(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetMarginInfoBase(t.Context())
	require.NoError(t, err, "GetMarginInfoBase must not error")
	if mockTests {
		exp := &MarginInfoBaseResponse{
			Key: "base",
			Info: MarginInfoBase{
				UserProfitLoss: -3631.798995959999,
				UserSwaps:      5.6376,
				MarginBalance:  299933.09380832,
				MarginNet:      296301.29481236,
				MarginMinimum:  4975.291049394,
			},
		}
		assert.Equal(t, exp, result, "GetMarginInfoBase should decode every field")
	} else {
		assert.Equal(t, "base", result.Key, "GetMarginInfoBase should return the base margin info")
	}

	_, err = newResponseServerExchange(t, `null`).GetMarginInfoBase(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetMarginInfoBase should error for a null response")
	_, err = newUnavailableServerExchange(t).GetMarginInfoBase(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetMarginInfoBase should return the request failure")
}

func TestGetMarginInfoBySymbol(t *testing.T) {
	t.Parallel()
	for _, symbol := range []string{"", "t", "fUSD", "base", "sym_all"} {
		_, err := e.GetMarginInfoBySymbol(t.Context(), symbol)
		assert.ErrorIsf(t, err, errTradingSymbolRequired, "GetMarginInfoBySymbol should refuse symbol %q", symbol)
	}

	skipLiveWithoutCredentials(t)
	result, err := e.GetMarginInfoBySymbol(t.Context(), "tBTCUSD")
	require.NoError(t, err, "GetMarginInfoBySymbol must not error")
	if mockTests {
		exp := &MarginInfoSymbolResponse{
			Key:    "sym",
			Symbol: "tBTCUSD",
			Info: MarginInfoSymbol{
				TradableBalance: 854774.7939165855,
				GrossBalance:    909999.9796645334,
				BuyAmount:       52.08334097688924,
				SellAmount:      62.676711921295684,
			},
		}
		assert.Equal(t, exp, result, "GetMarginInfoBySymbol should decode every field")
	} else {
		assert.Equal(t, "tBTCUSD", result.Symbol, "GetMarginInfoBySymbol should return the pair's margin info")
	}

	_, err = newResponseServerExchange(t, `null`).GetMarginInfoBySymbol(t.Context(), "tBTCUSD")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetMarginInfoBySymbol should error for a null response")
	_, err = newUnavailableServerExchange(t).GetMarginInfoBySymbol(t.Context(), "tBTCUSD")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetMarginInfoBySymbol should return the request failure")
}

func TestGetMarginInfoAllSymbols(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetMarginInfoAllSymbols(t.Context())
	require.NoError(t, err, "GetMarginInfoAllSymbols must not error")
	if mockTests {
		exp := []MarginInfoSymbolResponse{{
			Key:    "sym",
			Symbol: "tBTCUSD",
			Info: MarginInfoSymbol{
				TradableBalance: 909998.8707081091,
				GrossBalance:    909998.8707081091,
				BuyAmount:       54.66115273354812,
				SellAmount:      61.63633640667225,
			},
		}}
		assert.Equal(t, exp, result, "GetMarginInfoAllSymbols should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetMarginInfoAllSymbols should return every margin pair's info")
	}

	_, err = newUnavailableServerExchange(t).GetMarginInfoAllSymbols(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetMarginInfoAllSymbols should return the request failure")
}

func TestGetActivePositions(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetActivePositions(t.Context())
	require.NoError(t, err, "GetActivePositions must not error")
	if mockTests {
		exp := []PositionResponse{{
			Symbol:               "tBTCUSD",
			Status:               "ACTIVE",
			Amount:               0.0195,
			BasePrice:            8565.0267019,
			ProfitLoss:           -0.33455568705000516,
			ProfitLossPercentage: -0.0003117550117425625,
			LiquidationPrice:     7045.876419249083,
			Leverage:             3.0673001895895604,
			ID:                   142355652,
			CreatedAt:            types.Time(time.UnixMilli(1574002216000)),
			UpdatedAt:            types.Time(time.UnixMilli(1574002216000)),
			Meta:                 json.RawMessage(`{"reason":"TRADE","order_id":34271018124,"liq_stage":null,"trade_price":"8565.0267019","trade_amount":"0.0195","order_id_oppo":34277498022}`),
		}, {
			Symbol:               "tBTCF0:USTF0",
			Status:               "ACTIVE",
			Amount:               0.2,
			BasePrice:            80000,
			ProfitLoss:           -12.5,
			ProfitLossPercentage: -0.0625,
			LiquidationPrice:     79400,
			Leverage:             80,
			ID:                   142355653,
			CreatedAt:            types.Time(time.UnixMilli(1792022400000)),
			UpdatedAt:            types.Time(time.UnixMilli(1792026000000)),
			Type:                 PositionTypeDerivative,
			Collateral:           200,
			MinimumCollateral:    160,
			Meta:                 json.RawMessage(`{"reason":"TRADE","order_id":1747566430,"liq_stage":null,"trade_price":"80000","trade_amount":"0.2","order_id_oppo":1747566431}`),
		}}
		assert.Equal(t, exp, result, "GetActivePositions should decode every field")
	} else {
		for i := range result {
			assert.Truef(t, isTradingSymbol(result[i].Symbol), "GetActivePositions should return position %d's trading pair", i)
		}
	}

	_, err = newUnavailableServerExchange(t).GetActivePositions(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetActivePositions should return the request failure")
}

func TestClaimPosition(t *testing.T) {
	t.Parallel()
	_, err := e.ClaimPosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ClaimPosition must reject a nil request")
	_, err = e.ClaimPosition(t.Context(), &ClaimPositionRequest{Amount: 0.001})
	require.ErrorIs(t, err, errPositionIDRequired, "ClaimPosition must require a position ID")

	if !mockTests {
		t.Skip("ClaimPosition settles a position, which a test cannot undo")
	}
	result, err := e.ClaimPosition(t.Context(), &ClaimPositionRequest{PositionID: 142031891, Amount: 0.001})
	require.NoError(t, err, "ClaimPosition must not error")
	exp := &Notification[PositionResponse]{
		Time: types.Time(time.UnixMilli(1568650414540)),
		Type: "pm-req",
		Data: PositionResponse{
			Symbol:    "tBTCUSD",
			Status:    "ACTIVE",
			Amount:    -0.001,
			BasePrice: 10119,
			ID:        142031891,
			CreatedAt: types.Time(time.UnixMilli(1568650294000)),
			UpdatedAt: types.Time(time.UnixMilli(1568650294000)),
			// The docs give this endpoint's meta as a string holding JSON, where the others give an object
			Meta: json.RawMessage(`"{\"reason\": \"TRADE\", \"order_id\": 31089337681, \"liq_stage\": null, \"trade_price\": \"10119.0\", \"trade_amount\": \"-0.001\", \"user_id_oppo\": 183923, \"order_id_oppo\": 31089692490}"`),
		},
		Status: NotificationStatusSuccess,
	}
	assert.Equal(t, exp, result, "ClaimPosition should decode every field")

	_, err = newResponseServerExchange(t, `null`).ClaimPosition(t.Context(), &ClaimPositionRequest{PositionID: 142031891})
	assert.ErrorIs(t, err, common.ErrNoResponse, "ClaimPosition should error for a null response")
	_, err = newUnavailableServerExchange(t).ClaimPosition(t.Context(), &ClaimPositionRequest{PositionID: 142031891})
	assert.ErrorIs(t, err, request.ErrBadStatus, "ClaimPosition should return the request failure")
}

func TestIncreasePosition(t *testing.T) {
	t.Parallel()
	_, err := e.IncreasePosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "IncreasePosition must reject a nil request")
	for _, symbol := range []string{"", "fUSD"} {
		_, err = e.IncreasePosition(t.Context(), &IncreasePositionRequest{Symbol: symbol, Amount: 1})
		require.ErrorIsf(t, err, errTradingSymbolRequired, "IncreasePosition must refuse symbol %q", symbol)
	}
	_, err = e.IncreasePosition(t.Context(), &IncreasePositionRequest{Symbol: "tBTCUSD"})
	require.ErrorIs(t, err, errInvalidAmount, "IncreasePosition must refuse a zero amount")

	if !mockTests {
		t.Skip("IncreasePosition opens or grows a position, which a test cannot undo")
	}
	result, err := e.IncreasePosition(t.Context(), &IncreasePositionRequest{Symbol: "tBTCUSD", Amount: 1})
	require.NoError(t, err, "IncreasePosition must not error")
	exp := &Notification[PositionResponse]{
		Time:   types.Time(time.UnixMilli(1612447904081)),
		Type:   "pmi-req",
		Data:   PositionResponse{Symbol: "tBTCUSD", Amount: 1, BasePrice: 100},
		Status: NotificationStatusSuccess,
		Text:   "Submitting position increase.",
	}
	assert.Equal(t, exp, result, "IncreasePosition should decode every field")

	_, err = newResponseServerExchange(t, `null`).IncreasePosition(t.Context(), &IncreasePositionRequest{Symbol: "tBTCUSD", Amount: 1})
	assert.ErrorIs(t, err, common.ErrNoResponse, "IncreasePosition should error for a null response")
	_, err = newUnavailableServerExchange(t).IncreasePosition(t.Context(), &IncreasePositionRequest{Symbol: "tBTCUSD", Amount: 1})
	assert.ErrorIs(t, err, request.ErrBadStatus, "IncreasePosition should return the request failure")
}

func TestGetIncreasePositionInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetIncreasePositionInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetIncreasePositionInfo must reject a nil request")
	for _, symbol := range []string{"", "fUSD"} {
		_, err = e.GetIncreasePositionInfo(t.Context(), &IncreasePositionInfoRequest{Symbol: symbol})
		require.ErrorIsf(t, err, errTradingSymbolRequired, "GetIncreasePositionInfo must refuse symbol %q", symbol)
	}

	skipLiveWithoutCredentials(t)
	req := &IncreasePositionInfoRequest{Symbol: "tBTCUSD", Amount: 1.5}
	if !mockTests {
		req.Amount = 0.001
	}
	result, err := e.GetIncreasePositionInfo(t.Context(), req)
	require.NoError(t, err, "GetIncreasePositionInfo must not error")
	if mockTests {
		exp := &IncreasePositionInfoResponse{
			Position: IncreasePositionInfoPosition{MaximumPosition: 10, CurrentPosition: 0.1},
			Balance: IncreasePositionInfoBalance{
				BaseCurrencyBalance: 1.09965743,
				TradableBalance: IncreasePositionInfoTradableBalance{
					QuoteCurrency:      516734.60048924293,
					QuoteCurrencyTotal: 518415.66409424297,
					BaseCurrency:       11.5085657124553,
					BaseCurrencyTotal:  11.702877580552862,
				},
				FundingAvailable: 6843080.04050819,
			},
			Funding:         IncreasePositionInfoFunding{Value: 1.5, Required: 63345},
			FundingCurrency: IncreasePositionInfoFundingCurrency{ValueCurrency: currency.BTC, RequiredCurrency: currency.USD},
		}
		assert.Equal(t, exp, result, "GetIncreasePositionInfo should decode every field")
	} else {
		assert.False(t, result.FundingCurrency.RequiredCurrency.IsEmpty(), "GetIncreasePositionInfo should return the funding's currency")
	}

	_, err = newResponseServerExchange(t, `null`).GetIncreasePositionInfo(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetIncreasePositionInfo should error for a null response")
	_, err = newUnavailableServerExchange(t).GetIncreasePositionInfo(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetIncreasePositionInfo should return the request failure")
}

func TestGetPositionsHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositionsHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPositionsHistory must reject a nil request")
	_, err = e.GetPositionsHistory(t.Context(), &PositionsHistoryRequest{Start: time.UnixMilli(1574100000000), End: time.UnixMilli(1574000000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPositionsHistory must refuse a reversed window")

	skipLiveWithoutCredentials(t)
	req := &PositionsHistoryRequest{Start: time.UnixMilli(1574000000000), End: time.UnixMilli(1574100000000), Limit: 50, PositionID: 142355652}
	if !mockTests {
		req = &PositionsHistoryRequest{Start: time.Now().AddDate(0, -1, 0), End: time.Now(), Limit: 50}
	}
	result, err := e.GetPositionsHistory(t.Context(), req)
	require.NoError(t, err, "GetPositionsHistory must not error")
	if mockTests {
		exp := []PositionResponse{{
			Symbol:    "tBTCUSD",
			Status:    "CLOSED",
			BasePrice: 8639.50216298,
			ID:        142355652,
			CreatedAt: types.Time(time.UnixMilli(1574002216000)),
			UpdatedAt: types.Time(time.UnixMilli(1574003099000)),
		}}
		assert.Equal(t, exp, result, "GetPositionsHistory should decode every field")
	} else {
		for i := range result {
			assert.Truef(t, isTradingSymbol(result[i].Symbol), "GetPositionsHistory should return position %d's trading pair", i)
		}
	}

	_, err = newUnavailableServerExchange(t).GetPositionsHistory(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPositionsHistory should return the request failure")
}

func TestGetPositionsSnapshot(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositionsSnapshot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPositionsSnapshot must reject a nil request")
	_, err = e.GetPositionsSnapshot(t.Context(), &PositionsSnapshotRequest{Start: time.UnixMilli(1579600000000), End: time.UnixMilli(1579500000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPositionsSnapshot must refuse a reversed window")

	skipLiveWithoutCredentials(t)
	req := &PositionsSnapshotRequest{Start: time.UnixMilli(1579500000000), End: time.UnixMilli(1579600000000), Limit: 500, PositionID: 142661142}
	if !mockTests {
		req = &PositionsSnapshotRequest{Start: time.Now().AddDate(0, -1, 0), End: time.Now(), Limit: 50}
	}
	result, err := e.GetPositionsSnapshot(t.Context(), req)
	require.NoError(t, err, "GetPositionsSnapshot must not error")
	if mockTests {
		exp := []PositionResponse{{
			Symbol:    "tETHUSD",
			Status:    "ACTIVE",
			Amount:    -0.2,
			BasePrice: 167.01,
			ID:        142661142,
			CreatedAt: types.Time(time.UnixMilli(1579552390000)),
			UpdatedAt: types.Time(time.UnixMilli(1579552390000)),
		}}
		assert.Equal(t, exp, result, "GetPositionsSnapshot should decode every field")
	} else {
		for i := range result {
			assert.Truef(t, isTradingSymbol(result[i].Symbol), "GetPositionsSnapshot should return snapshot %d's trading pair", i)
		}
	}

	_, err = newUnavailableServerExchange(t).GetPositionsSnapshot(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPositionsSnapshot should return the request failure")
}

func TestGetPositionsAudit(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositionsAudit(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPositionsAudit must reject a nil request")
	_, err = e.GetPositionsAudit(t.Context(), &PositionsAuditRequest{Start: time.UnixMilli(1574100000000), End: time.UnixMilli(1574000000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPositionsAudit must refuse a reversed window")

	skipLiveWithoutCredentials(t)
	req := &PositionsAuditRequest{PositionIDs: []uint64{142358085}, Start: time.UnixMilli(1574000000000), End: time.UnixMilli(1574100000000), Limit: 250}
	if !mockTests {
		req = &PositionsAuditRequest{Start: time.Now().AddDate(0, -1, 0), End: time.Now(), Limit: 25}
	}
	result, err := e.GetPositionsAudit(t.Context(), req)
	require.NoError(t, err, "GetPositionsAudit must not error")
	if mockTests {
		exp := []PositionResponse{{
			Symbol:    "tETHUSD",
			Status:    "ACTIVE",
			Amount:    0.9682,
			BasePrice: 182.76,
			ID:        142358085,
			CreatedAt: types.Time(time.UnixMilli(1574088504000)),
			UpdatedAt: types.Time(time.UnixMilli(1574088504000)),
			Meta:      json.RawMessage(`{"reason":"TRADE","order_id":34319724780,"liq_stage":null,"trade_price":"182.76","trade_amount":"0.9682","user_id_oppo":411107,"order_id_oppo":34326729830}`),
		}}
		assert.Equal(t, exp, result, "GetPositionsAudit should decode every field")
	} else {
		for i := range result {
			assert.Truef(t, isTradingSymbol(result[i].Symbol), "GetPositionsAudit should return entry %d's trading pair", i)
		}
	}

	_, err = newUnavailableServerExchange(t).GetPositionsAudit(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPositionsAudit should return the request failure")
}

func TestUpdatePositionFundingType(t *testing.T) {
	t.Parallel()
	_, err := e.UpdatePositionFundingType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdatePositionFundingType must reject a nil request")
	for _, symbol := range []string{"", "fUSD"} {
		_, err = e.UpdatePositionFundingType(t.Context(), &UpdatePositionFundingTypeRequest{Symbol: symbol})
		require.ErrorIsf(t, err, errTradingSymbolRequired, "UpdatePositionFundingType must refuse symbol %q", symbol)
	}
	_, err = e.UpdatePositionFundingType(t.Context(), &UpdatePositionFundingTypeRequest{Symbol: "tBTCUSD", FundingType: 2})
	require.ErrorIs(t, err, errInvalidPositionFundingType, "UpdatePositionFundingType must refuse an undocumented funding type")

	// Daily funding is type 0, which must be sent rather than left out
	_, err = newBodyCheckExchange(t, `[1753796568064,"puft-req",null,null,null,null,"SUCCESS",null]`,
		`{"symbol":"tBTCUSD","type":0}`,
	).UpdatePositionFundingType(t.Context(), &UpdatePositionFundingTypeRequest{Symbol: "tBTCUSD", FundingType: PositionFundingTypeDaily})
	require.NoError(t, err, "UpdatePositionFundingType must not error for daily funding")

	skipLiveUnlessCanManipulate(t)
	req := &UpdatePositionFundingTypeRequest{Symbol: "tBTCUSD", FundingType: PositionFundingTypeTerm}
	if !mockTests {
		// Setting a position's own funding type leaves the account unchanged
		position := activePositionOfType(t, PositionTypeMargin)
		req = &UpdatePositionFundingTypeRequest{Symbol: position.Symbol, FundingType: position.MarginFundingType}
	}
	result, err := e.UpdatePositionFundingType(t.Context(), req)
	require.NoError(t, err, "UpdatePositionFundingType must not error")
	if mockTests {
		exp := &Notification[json.RawMessage]{
			Time:   types.Time(time.UnixMilli(1753796568064)),
			Type:   "puft-req",
			Status: NotificationStatusSuccess,
			Text:   "Submitting position funding type update.",
		}
		assert.Equal(t, exp, result, "UpdatePositionFundingType should decode every field")
	} else {
		assert.Equal(t, NotificationStatusSuccess, result.Status, "UpdatePositionFundingType should succeed")
	}

	_, err = newResponseServerExchange(t, `null`).UpdatePositionFundingType(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "UpdatePositionFundingType should error for a null response")
	_, err = newUnavailableServerExchange(t).UpdatePositionFundingType(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "UpdatePositionFundingType should return the request failure")
}

func TestSetDerivativePositionCollateral(t *testing.T) {
	t.Parallel()
	_, err := e.SetDerivativePositionCollateral(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetDerivativePositionCollateral must reject a nil request")
	for _, symbol := range []string{"", "fUSD"} {
		_, err = e.SetDerivativePositionCollateral(t.Context(), &DerivativePositionCollateralRequest{Symbol: symbol, Collateral: 1})
		require.ErrorIsf(t, err, errTradingSymbolRequired, "SetDerivativePositionCollateral must refuse symbol %q", symbol)
	}
	for _, collateral := range []float64{0, -1} {
		_, err = e.SetDerivativePositionCollateral(t.Context(), &DerivativePositionCollateralRequest{Symbol: "tBTCF0:USTF0", Collateral: collateral})
		require.ErrorIsf(t, err, errInvalidPositionCollateral, "SetDerivativePositionCollateral must refuse collateral %v", collateral)
	}

	// The docs take the collateral as a number, which encoding a float would send in exponent notation
	_, err = newBodyCheckExchange(t, `[[1]]`,
		`{"symbol":"tBTCF0:USTF0","collateral":0.0000001}`,
	).SetDerivativePositionCollateral(t.Context(), &DerivativePositionCollateralRequest{Symbol: "tBTCF0:USTF0", Collateral: 1e-7})
	require.NoError(t, err, "SetDerivativePositionCollateral must not error for a small collateral")

	skipLiveUnlessCanManipulate(t)
	req := &DerivativePositionCollateralRequest{Symbol: "tBTCF0:USTF0", Collateral: 1150.61}
	if !mockTests {
		// Setting a position's own collateral leaves the account unchanged
		position := activePositionOfType(t, PositionTypeDerivative)
		if position.Collateral == 0 {
			t.Skip("The derivative position reports no collateral to set")
		}
		req = &DerivativePositionCollateralRequest{Symbol: position.Symbol, Collateral: position.Collateral}
	}
	result, err := e.SetDerivativePositionCollateral(t.Context(), req)
	require.NoError(t, err, "SetDerivativePositionCollateral must not error")
	assert.Equal(t, &DerivativePositionCollateralResponse{Success: true}, result, "SetDerivativePositionCollateral should report the collateral set")
	if mockTests {
		_, err = e.SetDerivativePositionCollateral(t.Context(), &DerivativePositionCollateralRequest{Symbol: "tBTCF0:USTF0", Collateral: 1e9})
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "SetDerivativePositionCollateral must return the error Bitfinex reports")
		assert.Equal(t, &APIError{Code: 10020, Message: "collateral: insufficient"}, apiErr, "SetDerivativePositionCollateral should decode the reported error")
	}

	for _, response := range []string{`null`, `[]`} {
		_, err = newResponseServerExchange(t, response).SetDerivativePositionCollateral(t.Context(), req)
		assert.ErrorIsf(t, err, common.ErrNoResponse, "SetDerivativePositionCollateral should error for response %s", response)
	}
	_, err = newUnavailableServerExchange(t).SetDerivativePositionCollateral(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SetDerivativePositionCollateral should return the request failure")
}

func TestGetDerivativePositionCollateralLimits(t *testing.T) {
	t.Parallel()
	for _, symbol := range []string{"", "t", "fUSD"} {
		_, err := e.GetDerivativePositionCollateralLimits(t.Context(), symbol)
		assert.ErrorIsf(t, err, errTradingSymbolRequired, "GetDerivativePositionCollateralLimits should refuse symbol %q", symbol)
	}

	skipLiveWithoutCredentials(t)
	symbol := "tBTCF0:USTF0"
	if !mockTests {
		// Bitfinex calculates the limits only for a pair with an active position
		symbol = activePositionOfType(t, PositionTypeDerivative).Symbol
	}
	result, err := e.GetDerivativePositionCollateralLimits(t.Context(), symbol)
	require.NoError(t, err, "GetDerivativePositionCollateralLimits must not error")
	if mockTests {
		exp := &DerivativePositionCollateralLimitsResponse{MinimumCollateral: 18.289, MaximumCollateral: 896.67400419}
		assert.Equal(t, exp, result, "GetDerivativePositionCollateralLimits should decode every field")
	} else {
		assert.LessOrEqual(t, result.MinimumCollateral, result.MaximumCollateral, "GetDerivativePositionCollateralLimits should return a minimum within the maximum")
	}

	_, err = newResponseServerExchange(t, `null`).GetDerivativePositionCollateralLimits(t.Context(), symbol)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetDerivativePositionCollateralLimits should error for a null response")
	_, err = newUnavailableServerExchange(t).GetDerivativePositionCollateralLimits(t.Context(), symbol)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetDerivativePositionCollateralLimits should return the request failure")
}

// TestPositionsRecordsEveryPosition decodes each Positions record from an array with a distinct value at every tagged
// position, so a field tagged with the wrong position or type cannot pass on a null or zero element
func TestPositionsRecordsEveryPosition(t *testing.T) {
	t.Parallel()
	testTupleDecode(t, `[[0.5,1.5],[10.5,[110.5,111.5,112.5,113.5],12.5],null,null,[40.5,41.5],["BTC","USD"]]`,
		IncreasePositionInfoResponse{
			Position: IncreasePositionInfoPosition{MaximumPosition: 0.5, CurrentPosition: 1.5},
			Balance: IncreasePositionInfoBalance{
				BaseCurrencyBalance: 10.5,
				TradableBalance:     IncreasePositionInfoTradableBalance{QuoteCurrency: 110.5, QuoteCurrencyTotal: 111.5, BaseCurrency: 112.5, BaseCurrencyTotal: 113.5},
				FundingAvailable:    12.5,
			},
			Funding:         IncreasePositionInfoFunding{Value: 40.5, Required: 41.5},
			FundingCurrency: IncreasePositionInfoFundingCurrency{ValueCurrency: currency.BTC, RequiredCurrency: currency.USD},
		})
	testTupleDecode(t, `[1]`, DerivativePositionCollateralResponse{Success: true})
	testTupleDecode(t, `[0.5,1.5]`, DerivativePositionCollateralLimitsResponse{MinimumCollateral: 0.5, MaximumCollateral: 1.5})
}
