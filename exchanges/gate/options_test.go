package gate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetAllOptionsUnderlyings(t *testing.T) {
	t.Parallel()
	underlyings, err := e.GetAllOptionsUnderlyings(t.Context())
	require.NoError(t, err, "GetAllOptionsUnderlyings must not error")
	if mockTests {
		require.Len(t, underlyings, 5, "GetAllOptionsUnderlyings must return every recorded underlying")
		exp := &OptionUnderlyingResponse{Name: "BTC_USDT", IndexPrice: 67504.01, IndexTime: types.Time(time.Unix(1775423976, 0))}
		assert.Equal(t, exp, underlyings[0], "GetAllOptionsUnderlyings should decode every field")
		return
	}
	assert.NotEmpty(t, underlyings, "GetAllOptionsUnderlyings should return underlyings")
}

func TestGetExpirationTimes(t *testing.T) {
	t.Parallel()
	_, err := e.GetExpirationTimes(t.Context(), "")
	require.ErrorIs(t, err, errInvalidUnderlying, "GetExpirationTimes must reject an empty underlying")

	result, err := e.GetExpirationTimes(t.Context(), getPair(t, asset.Spot).String())
	require.NoError(t, err, "GetExpirationTimes must not error")
	if mockTests {
		exp := make([]time.Time, 0, 10)
		for _, ts := range []int64{1775635200, 1775548800, 1775808000, 1798185600, 1777017600, 1775462400, 1780041600, 1782460800, 1776412800, 1790323200} {
			exp = append(exp, time.Unix(ts, 0))
		}
		assert.Equal(t, exp, result, "GetExpirationTimes should return every expiration in the order listed")
		return
	}
	assert.NotEmpty(t, result, "GetExpirationTimes should return expirations")
}

func TestGetAllContractOfUnderlyingWithinExpiryDate(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllContractOfUnderlyingWithinExpiryDate(t.Context(), "", time.Time{})
	require.ErrorIs(t, err, errInvalidUnderlying, "GetAllContractOfUnderlyingWithinExpiryDate must reject an empty underlying")

	result, err := e.GetAllContractOfUnderlyingWithinExpiryDate(t.Context(), "BTC_USDT", time.Time{})
	require.NoError(t, err, "GetAllContractOfUnderlyingWithinExpiryDate must not error")
	require.NotEmpty(t, result, "GetAllContractOfUnderlyingWithinExpiryDate must return contracts")
	exp := &OptionContractResponse{
		Name:              "BTC_USDT-20260408-73000-P",
		Tag:               "day",
		StrikePrice:       73000,
		MarkPrice:         5533.2,
		OrderbookID:       34336,
		Underlying:        "BTC_USDT",
		UnderlyingPrice:   67488.22,
		Multiplier:        0.01,
		OrderPriceRound:   1,
		MarkPriceRound:    0.1,
		MakerFeeRate:      0.0003,
		TakerFeeRate:      0.0003,
		PriceLimitFeeRate: 0.125,
		OrderPriceDeviate: 0.15,
		OrderSizeMin:      1,
		OrderSizeMax:      10000,
		OrdersLimit:       10,
		CreateTime:        types.Time(time.Unix(1775375281, 0)),
		ExpirationTime:    types.Time(time.Unix(1775635200, 0)),
	}
	if mockTests {
		require.Len(t, result, 6, "GetAllContractOfUnderlyingWithinExpiryDate must return every recorded contract")
		assert.Equal(t, exp, result[0], "GetAllContractOfUnderlyingWithinExpiryDate should decode every field")
	}

	expirations, err := e.GetExpirationTimes(t.Context(), "BTC_USDT")
	require.NoError(t, err, "GetExpirationTimes must not error")
	require.NotEmpty(t, expirations, "GetExpirationTimes must return expirations")
	result, err = e.GetAllContractOfUnderlyingWithinExpiryDate(t.Context(), "BTC_USDT", expirations[0])
	require.NoError(t, err, "GetAllContractOfUnderlyingWithinExpiryDate must not error for an expiration")
	if mockTests {
		exp.IndexPrice = 67503.71
		assert.Equal(t, []*OptionContractResponse{exp}, result, "GetAllContractOfUnderlyingWithinExpiryDate should decode every field")
		return
	}
	require.NotEmpty(t, result, "GetAllContractOfUnderlyingWithinExpiryDate must return contracts for an expiration")
	for _, contract := range result {
		assert.Truef(t, contract.ExpirationTime.Time().Equal(expirations[0]), "contract %s should expire at the requested expiration", contract.Name)
	}
}

func TestGetOptionsSpecifiedContractDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionsSpecifiedContractDetail(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOptionsSpecifiedContractDetail must reject an empty contract")

	result, err := e.GetOptionsSpecifiedContractDetail(t.Context(), getPair(t, asset.Options))
	require.NoError(t, err, "GetOptionsSpecifiedContractDetail must not error")
	if mockTests {
		exp := &OptionContractResponse{
			Name:              "BTC_USDT-20260925-75000-P",
			Tag:               "month",
			StrikePrice:       75000,
			LastPrice:         12988,
			MarkPrice:         12368.9,
			OrderbookID:       3087433,
			TradeID:           3,
			TradeSize:         3,
			PositionSize:      2,
			Underlying:        "BTC_USDT",
			UnderlyingPrice:   68398.81,
			Multiplier:        0.01,
			OrderPriceRound:   1,
			MarkPriceRound:    0.1,
			MakerFeeRate:      0.0003,
			TakerFeeRate:      0.0003,
			PriceLimitFeeRate: 0.125,
			OrderPriceDeviate: 0.15,
			OrderSizeMin:      1,
			OrderSizeMax:      10000,
			OrdersLimit:       10,
			CreateTime:        types.Time(time.Unix(1770450500, 0)),
			ExpirationTime:    types.Time(time.Unix(1790323200, 0)),
		}
		assert.Equal(t, exp, result, "GetOptionsSpecifiedContractDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionsSpecifiedContractDetail should return a contract")
}

func TestGetSettlementHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSettlementHistory(t.Context(), "", 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidUnderlying, "GetSettlementHistory must reject an empty underlying")

	startTime, endTime := getTime()
	_, err = e.GetSettlementHistory(t.Context(), "BTC_USDT", 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSettlementHistory must reject a start after the end")

	result, err := e.GetSettlementHistory(t.Context(), "BTC_USDT", 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err, "GetSettlementHistory must not error")
	if mockTests {
		require.Len(t, result, 5, "GetSettlementHistory must return every recorded settlement")
		exp := &OptionSettlementResponse{
			Timestamp:   types.Time(time.Unix(1775376000, 0)),
			SettlePrice: 66774.71479977753,
			Contract:    "BTC_USDT-20260405-60000-P",
			StrikePrice: 60000,
		}
		assert.Equal(t, exp, result[0], "GetSettlementHistory should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetSettlementHistory should return settlements")
}

func TestGetOptionsSpecifiedContractsSettlement(t *testing.T) {
	t.Parallel()
	contract := currency.NewPairWithDelimiter("BTC", "USDT-20260405-60000-P", currency.UnderscoreDelimiter)
	_, err := e.GetOptionsSpecifiedContractsSettlement(t.Context(), contract, "", time.Unix(1775376000, 0))
	require.ErrorIs(t, err, errInvalidUnderlying, "GetOptionsSpecifiedContractsSettlement must reject an empty underlying")

	_, err = e.GetOptionsSpecifiedContractsSettlement(t.Context(), currency.EMPTYPAIR, "BTC_USDT", time.Unix(1775376000, 0))
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOptionsSpecifiedContractsSettlement must reject an empty contract")

	_, err = e.GetOptionsSpecifiedContractsSettlement(t.Context(), contract, "BTC_USDT", time.Time{})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetOptionsSpecifiedContractsSettlement must reject an unset settlement time")

	underlying := "BTC_USDT"
	optionsSettlement, err := e.GetSettlementHistory(t.Context(), underlying, 0, 1, time.Time{}, time.Time{})
	require.NoError(t, err, "GetSettlementHistory must not error")
	require.NotEmpty(t, optionsSettlement, "GetSettlementHistory must return a settlement")

	cp, err := currency.NewPairFromString(optionsSettlement[0].Contract)
	require.NoError(t, err, "NewPairFromString must not error")

	result, err := e.GetOptionsSpecifiedContractsSettlement(t.Context(), cp, underlying, optionsSettlement[0].Timestamp.Time())
	require.NoError(t, err, "GetOptionsSpecifiedContractsSettlement must not error")
	if mockTests {
		exp := &OptionSettlementResponse{
			Timestamp:   types.Time(time.Unix(1774944000, 0)),
			Profit:      73.431430405781,
			SettlePrice: 67343.1430405781,
			Contract:    "BTC_USDT-20260405-60000-P",
			StrikePrice: 60000,
		}
		assert.Equal(t, exp, result, "GetOptionsSpecifiedContractsSettlement should decode every field")
		return
	}
	assert.Equal(t, optionsSettlement[0].Contract, result.Contract, "GetOptionsSpecifiedContractsSettlement should return the requested contract")
}

func TestGetMyOptionsSettlements(t *testing.T) {
	t.Parallel()
	_, err := e.GetMyOptionsSettlements(t.Context(), "", currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidUnderlying, "GetMyOptionsSettlements must reject an empty underlying")

	startTime, endTime := getTime()
	_, err = e.GetMyOptionsSettlements(t.Context(), "BTC_USDT", currency.EMPTYPAIR, 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMyOptionsSettlements must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMyOptionsSettlements(t.Context(), "BTC_USDT", currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err, "GetMyOptionsSettlements must not error")
	if mockTests {
		exp := []*MyOptionSettlementResponse{{
			Size:        -1,
			Contract:    currency.NewPairWithDelimiter("BTC", "USDT-20220624-26000-C", currency.UnderscoreDelimiter),
			StrikePrice: 26000,
			Time:        types.Time(time.Unix(1656057600, 0)),
			SettlePrice: 20917.461281337048,
			Underlying:  "BTC_USDT",
			RealisedPNL: -0.00116042,
			Fee:         0.0001,
		}}
		assert.Equal(t, exp, result, "GetMyOptionsSettlements should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMyOptionsSettlements should return a list")
}

func TestGetOptionAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOptionAccount(t.Context())
	require.NoError(t, err, "GetOptionAccount must not error")
	if mockTests {
		exp := &OptionAccountResponse{
			UserID:                       666,
			Total:                        1650.443022,
			PositionValue:                -40.1136,
			Equity:                       1610.329422,
			ShortEnabled:                 true,
			MarketMakerProtectionEnabled: true,
			MarginMode:                   1,
			UnrealisedPNL:                -0.7811,
			InitialMargin:                0.5,
			MaintenanceMargin:            135.541485,
			OrderMargin:                  139.74496,
			AskOrderMargin:               139.74496,
			BidOrderMargin:               0.25,
			Available:                    1514.901537,
			Point:                        0.1,
			Currency:                     currency.USDT,
			MaxOrderLimit:                10,
			PositionNotionalLimit:        1000000,
		}
		assert.Equal(t, exp, result, "GetOptionAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionAccount should return an account")
}

func TestGetAccountChangingHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetAccountChangingHistory(t.Context(), 0, 0, endTime, startTime, "")
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAccountChangingHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountChangingHistory(t.Context(), 1, 100, startTime, endTime, "prem")
	require.NoError(t, err, "GetAccountChangingHistory must not error")
	if mockTests {
		exp := []*AccountBookResponse{{
			ChangeTime:    types.Time(time.Unix(1744110000, 0)),
			AccountChange: -0.16,
			Balance:       7378.189,
			CustomText:    "BTC_USDT-20260925-75000-P:25",
			ChangingType:  "prem",
		}}
		assert.Equal(t, exp, result, "GetAccountChangingHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAccountChangingHistory should return a list")
}

func TestGetUsersPositionSpecifiedUnderlying(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUsersPositionSpecifiedUnderlying(t.Context(), "")
	require.NoError(t, err, "GetUsersPositionSpecifiedUnderlying must not error for every underlying")
	if mockTests {
		withoutCloseOrder := optionsPositionFixture(currency.NewPairWithDelimiter("BTC", "USDT-20260925-80000-C", currency.UnderscoreDelimiter))
		withoutCloseOrder.Size = -2
		withoutCloseOrder.PendingOrders = 0
		withoutCloseOrder.CloseOrder = CloseOrder{}
		exp := []*UsersPositionForUnderlyingResponse{optionsPositionFixture(getPair(t, asset.Options)), withoutCloseOrder}
		assert.Equal(t, exp, result, "GetUsersPositionSpecifiedUnderlying should decode every position, including one without a close order")
	}

	result, err = e.GetUsersPositionSpecifiedUnderlying(t.Context(), "BTC_USDT")
	require.NoError(t, err, "GetUsersPositionSpecifiedUnderlying must not error")
	if mockTests {
		assert.Equal(t, []*UsersPositionForUnderlyingResponse{optionsPositionFixture(getPair(t, asset.Options))}, result, "GetUsersPositionSpecifiedUnderlying should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUsersPositionSpecifiedUnderlying should return a list")
}

func TestGetSpecifiedContractPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpecifiedContractPosition(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpecifiedContractPosition must reject an empty contract")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpecifiedContractPosition(t.Context(), getPair(t, asset.Options))
	require.NoError(t, err, "GetSpecifiedContractPosition must not error")
	if mockTests {
		assert.Equal(t, optionsPositionFixture(getPair(t, asset.Options)), result, "GetSpecifiedContractPosition should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpecifiedContractPosition should return a position")
}

// optionsPositionFixture returns the options position recorded in testdata/http.json
func optionsPositionFixture(contract currency.Pair) *UsersPositionForUnderlyingResponse {
	return &UsersPositionForUnderlyingResponse{
		User:                  11027586,
		Underlying:            "BTC_USDT",
		UnderlyingPrice:       70000,
		Contract:              contract,
		Size:                  10,
		EntryPrice:            1234,
		MarkPrice:             6000,
		MarkImpliedVolatility: 0.9638,
		RealisedPNL:           120,
		UnrealisedPNL:         -320,
		PendingOrders:         1,
		CloseOrder:            CloseOrder{ID: 232323, Price: 5779, IsLiquidation: true},
		Delta:                 -0.0046,
		Gamma:                 0.0001,
		Vega:                  2.87656,
		Theta:                 -1.00247,
	}
}

func TestGetUsersLiquidationHistoryForSpecifiedUnderlying(t *testing.T) {
	t.Parallel()
	_, err := e.GetUsersLiquidationHistoryForSpecifiedUnderlying(t.Context(), "", currency.EMPTYPAIR)
	require.ErrorIs(t, err, errInvalidUnderlying, "GetUsersLiquidationHistoryForSpecifiedUnderlying must reject an empty underlying")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUsersLiquidationHistoryForSpecifiedUnderlying(t.Context(), "BTC_USDT", currency.EMPTYPAIR)
	require.NoError(t, err, "GetUsersLiquidationHistoryForSpecifiedUnderlying must not error")
	if mockTests {
		exp := []*ContractClosePositionResponse{{
			PositionCloseTime: types.Time(time.Unix(1631764800, 0)),
			Contract:          "BTC_USDT-20210916-5000-C",
			Side:              "short",
			PNL:               -42914.291,
			CloseOrderText:    "settled",
			SettleSize:        -10001,
		}}
		assert.Equal(t, exp, result, "GetUsersLiquidationHistoryForSpecifiedUnderlying should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUsersLiquidationHistoryForSpecifiedUnderlying should return a list")
}

func TestPlaceOptionOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceOptionOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceOptionOrder must reject a nil request")

	arg := &OptionOrderRequest{
		OrderSize:   -1,
		Iceberg:     0,
		Text:        "-",
		TimeInForce: "gtc",
		Price:       100,
	}
	_, err = e.PlaceOptionOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceOptionOrder must reject an empty contract")

	pair := getPair(t, asset.Options)
	_, err = e.PlaceOptionOrder(t.Context(), &OptionOrderRequest{Contract: pair, Price: 100})
	require.ErrorIs(t, err, errInvalidOrderSize, "PlaceOptionOrder must reject a zero size without closing the position")

	_, err = e.PlaceOptionOrder(t.Context(), &OptionOrderRequest{Contract: pair, OrderSize: 1, Close: true})
	require.ErrorIs(t, err, errInvalidOrderSize, "PlaceOptionOrder must reject a non-zero size when closing the position")

	_, err = e.PlaceOptionOrder(t.Context(), &OptionOrderRequest{Contract: pair, OrderSize: 1, Price: -1})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "PlaceOptionOrder must reject a negative price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.Contract = pair
	result, err := e.PlaceOptionOrder(t.Context(), arg)
	require.NoError(t, err, "PlaceOptionOrder must not error")
	if !mockTests {
		assert.NotZero(t, result.OptionOrderID, "PlaceOptionOrder should return an order ID")
		return
	}
	assert.Equal(t, optionsOrderFixture(pair, 2), result, "PlaceOptionOrder should decode every field")

	// The recorded request body only matches a market order sending its zero price as "0"
	result, err = e.PlaceOptionOrder(t.Context(), &OptionOrderRequest{Contract: pair, OrderSize: 1, TimeInForce: iocTIF})
	require.NoError(t, err, "PlaceOptionOrder must not error for a market order")
	exp := &OptionOrderResponse{
		OptionOrderID: 3,
		User:          666,
		CreateTime:    types.Time(time.Unix(1631763361, 0)),
		FinishTime:    types.Time(time.Unix(1631763362, 0)),
		FinishAs:      "filled",
		Status:        statusFinished,
		Contract:      "BTC_USDT-20260925-75000-P",
		Size:          1,
		TimeInForce:   iocTIF,
		FillPrice:     12988,
		Text:          "api",
		TakerFeeRate:  0.0003,
		MakerFeeRate:  0.0003,
	}
	assert.Equal(t, exp, result, "PlaceOptionOrder should decode a filled market order")
}

func TestOptionOrderRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	pair := currency.NewPairWithDelimiter("BTC", "USDT-20260925-75000-P", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name    string
		request *OptionOrderRequest
		exp     string
	}{
		{
			name:    "market order",
			request: &OptionOrderRequest{Contract: pair, OrderSize: 1, TimeInForce: iocTIF},
			exp:     `{"contract":"BTC_USDT-20260925-75000-P","size":1,"price":"0","tif":"ioc"}`,
		},
		{
			name:    "close position",
			request: &OptionOrderRequest{Contract: pair, Close: true, TimeInForce: iocTIF},
			exp:     `{"contract":"BTC_USDT-20260925-75000-P","size":0,"price":"0","close":true,"tif":"ioc"}`,
		},
		{
			name: "limit order",
			request: &OptionOrderRequest{
				Contract:              pair,
				OrderSize:             -2,
				Iceberg:               1,
				Price:                 100.5,
				ReduceOnly:            true,
				MarketMakerProtection: true,
				TimeInForce:           pocTIF,
				Text:                  "t-my-custom-id",
			},
			exp: `{"contract":"BTC_USDT-20260925-75000-P","size":-2,"iceberg":1,"price":"100.5","reduce_only":true,"mmp":true,"tif":"poc","text":"t-my-custom-id"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.request)
			require.NoError(t, err, "Marshal must not error")
			assert.JSONEq(t, tc.exp, string(got), "Marshal should send the documented encoding")
		})
	}
}

func TestGetOptionFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionFuturesOrders(t.Context(), currency.EMPTYPAIR, "", "", 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetOptionFuturesOrders must reject an empty status")

	startTime, endTime := getTime()
	_, err = e.GetOptionFuturesOrders(t.Context(), currency.EMPTYPAIR, "", statusOpen, 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOptionFuturesOrders must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOptionFuturesOrders(t.Context(), currency.EMPTYPAIR, "", statusOpen, 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err, "GetOptionFuturesOrders must not error")
	if mockTests {
		exp := optionsOrderFixture(getPair(t, asset.Options), 2)
		exp.IsMarketMakerProtection = true
		assert.Equal(t, []*OptionOrderResponse{exp}, result, "GetOptionFuturesOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionFuturesOrders should return a list")
}

func TestCancelMultipleOptionOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleOptionOpenOrders(t.Context(), getPair(t, asset.Options), "", "buy")
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CancelMultipleOptionOpenOrders must reject a side other than ask or bid")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMultipleOptionOpenOrders(t.Context(), getPair(t, asset.Options), "", "")
	require.NoError(t, err, "CancelMultipleOptionOpenOrders must not error")
	if mockTests {
		assert.Equal(t, []*OptionOrderResponse{optionsCancelledOrderFixture()}, result, "CancelMultipleOptionOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelMultipleOptionOpenOrders should return a list")
}

func TestAmendOptionOrder(t *testing.T) {
	t.Parallel()
	pair := getPair(t, asset.Options)
	_, err := e.AmendOptionOrder(t.Context(), "", &AmendOptionOrderRequest{Contract: pair, Price: 1661, Size: 10})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendOptionOrder must reject an empty order ID")

	_, err = e.AmendOptionOrder(t.Context(), "1234", nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendOptionOrder must reject a nil request")

	_, err = e.AmendOptionOrder(t.Context(), "1234", &AmendOptionOrderRequest{Price: 1661, Size: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "AmendOptionOrder must reject an empty contract")

	_, err = e.AmendOptionOrder(t.Context(), "1234", &AmendOptionOrderRequest{Contract: pair, Size: 10})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "AmendOptionOrder must reject a zero price")

	_, err = e.AmendOptionOrder(t.Context(), "1234", &AmendOptionOrderRequest{Contract: pair, Price: 1661})
	require.ErrorIs(t, err, errInvalidOrderSize, "AmendOptionOrder must reject a zero size")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AmendOptionOrder(t.Context(), "1234", &AmendOptionOrderRequest{Contract: pair, Price: 1661, Size: 10})
	require.NoError(t, err, "AmendOptionOrder must not error")
	if mockTests {
		exp := &OptionOrderResponse{
			OptionOrderID:   1234,
			User:            666,
			CreateTime:      types.Time(time.Unix(1631763361, 0)),
			Status:          statusOpen,
			Contract:        "BTC_USDT-20260925-75000-P",
			Size:            10,
			Price:           1661,
			TimeInForce:     gtcTIF,
			RemainingAmount: 10,
			Text:            "t-amended",
			TakerFeeRate:    0.0003,
			MakerFeeRate:    0.0003,
		}
		assert.Equal(t, exp, result, "AmendOptionOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "AmendOptionOrder should return an order")
}

func TestGetSingleOptionOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleOptionOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSingleOptionOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSingleOptionOrder(t.Context(), "1234")
	require.NoError(t, err, "GetSingleOptionOrder must not error")
	if mockTests {
		assert.Equal(t, optionsOrderFixture(getPair(t, asset.Options), 1234), result, "GetSingleOptionOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleOptionOrder should return an order")
}

func TestCancelOptionSingleOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOptionSingleOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelOptionSingleOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelOptionSingleOrder(t.Context(), "1234")
	require.NoError(t, err, "CancelOptionSingleOrder must not error")
	if mockTests {
		assert.Equal(t, optionsCancelledOrderFixture(), result, "CancelOptionSingleOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelOptionSingleOrder should return an order")
}

// optionsOrderFixture returns the open options order recorded in testdata/http.json
func optionsOrderFixture(contract currency.Pair, id uint64) *OptionOrderResponse {
	return &OptionOrderResponse{
		OptionOrderID:   id,
		User:            666,
		CreateTime:      types.Time(time.Unix(1631763361, 0)),
		Status:          statusOpen,
		Contract:        contract.String(),
		Size:            -1,
		Price:           100,
		TimeInForce:     gtcTIF,
		RemainingAmount: -1,
		Text:            "-",
		TakerFeeRate:    0.0003,
		MakerFeeRate:    0.0003,
	}
}

// optionsCancelledOrderFixture returns the cancelled options order recorded in testdata/http.json
func optionsCancelledOrderFixture() *OptionOrderResponse {
	return &OptionOrderResponse{
		OptionOrderID:   1234,
		User:            666,
		CreateTime:      types.Time(time.Unix(1631763361, 0)),
		FinishTime:      types.Time(time.Unix(1631763397, 0)),
		FinishAs:        "cancelled",
		Status:          statusFinished,
		Contract:        "BTC_USDT-20260925-75000-P",
		Size:            -1,
		Price:           100,
		TimeInForce:     gtcTIF,
		RemainingAmount: -1,
		Text:            "-",
		TakerFeeRate:    0.0003,
		MakerFeeRate:    0.0003,
		ReferenceUserID: 123,
		ReferrerRebate:  0.1,
	}
}

func TestCountdownCancelOptionOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CountdownCancelOptionOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CountdownCancelOptionOrders must reject a nil request")

	for _, timeout := range []time.Duration{-time.Second, 4 * time.Second, 5500 * time.Millisecond} {
		_, err = e.CountdownCancelOptionOrders(t.Context(), &CountdownCancelOptionOrdersRequest{Countdown: types.DurationSeconds(timeout)})
		require.ErrorIsf(t, err, errInvalidTimeout, "CountdownCancelOptionOrders must reject a timeout of %s", timeout)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CountdownCancelOptionOrders(t.Context(), &CountdownCancelOptionOrdersRequest{Countdown: types.DurationSeconds(30 * time.Second), Underlying: "BTC_USDT"})
	require.NoError(t, err, "CountdownCancelOptionOrders must not error")
	if mockTests {
		assert.Equal(t, &TriggerTimeResponse{TriggerTime: types.Time(time.UnixMilli(1660039145000))}, result, "CountdownCancelOptionOrders should decode the trigger time")
	} else {
		assert.NotZero(t, result.TriggerTime, "CountdownCancelOptionOrders should return a trigger time")
	}

	result, err = e.CountdownCancelOptionOrders(t.Context(), &CountdownCancelOptionOrdersRequest{})
	require.NoError(t, err, "CountdownCancelOptionOrders must not error when clearing the countdown")
	assert.Zero(t, result.TriggerTime, "CountdownCancelOptionOrders should return no trigger time once cleared")
}

func TestGetMyOptionsTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMyOptionsTradingHistory(t.Context(), "", currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidUnderlying, "GetMyOptionsTradingHistory must reject an empty underlying")

	startTime, endTime := getTime()
	_, err = e.GetMyOptionsTradingHistory(t.Context(), "BTC_USDT", currency.EMPTYPAIR, 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMyOptionsTradingHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMyOptionsTradingHistory(t.Context(), "BTC_USDT", currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err, "GetMyOptionsTradingHistory must not error")
	if mockTests {
		exp := []*OptionTradingHistoryResponse{{
			ID:              1,
			UnderlyingPrice: 48000,
			Size:            1,
			Contract:        getPair(t, asset.Options),
			TradeRole:       "taker",
			CreateTime:      types.Time(time.Unix(1631763397, 0)),
			OrderID:         4,
			Price:           100,
		}}
		assert.Equal(t, exp, result, "GetMyOptionsTradingHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMyOptionsTradingHistory should return a list")
}

func TestSetOptionsMarketMakerProtection(t *testing.T) {
	t.Parallel()
	_, err := e.SetOptionsMarketMakerProtection(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetOptionsMarketMakerProtection must reject a nil request")

	newRequest := func() *OptionsMarketMakerProtectionRequest {
		return &OptionsMarketMakerProtectionRequest{
			Underlying:    "BTC_USDT",
			Window:        5 * time.Second,
			FrozenPeriod:  200 * time.Millisecond,
			QuantityLimit: 10,
			DeltaLimit:    10,
		}
	}
	for _, tc := range []struct {
		name   string
		modify func(*OptionsMarketMakerProtectionRequest)
		err    error
	}{
		{name: "underlying", modify: func(r *OptionsMarketMakerProtectionRequest) { r.Underlying = "" }, err: errInvalidUnderlying},
		{name: "negative window", modify: func(r *OptionsMarketMakerProtectionRequest) { r.Window = -time.Millisecond }, err: errInvalidMMPWindow},
		{name: "window", modify: func(r *OptionsMarketMakerProtectionRequest) { r.Window = 5*time.Second + time.Millisecond }, err: errInvalidMMPWindow},
		{name: "fractional window", modify: func(r *OptionsMarketMakerProtectionRequest) { r.Window = 500 * time.Microsecond }, err: errInvalidMMPWindow},
		{name: "negative frozen period", modify: func(r *OptionsMarketMakerProtectionRequest) { r.FrozenPeriod = -time.Millisecond }, err: errInvalidMMPFrozenPeriod},
		{name: "fractional frozen period", modify: func(r *OptionsMarketMakerProtectionRequest) { r.FrozenPeriod = 1500 * time.Microsecond }, err: errInvalidMMPFrozenPeriod},
		{name: "quantity limit", modify: func(r *OptionsMarketMakerProtectionRequest) { r.QuantityLimit = 0 }, err: errInvalidMMPLimit},
		{name: "delta limit", modify: func(r *OptionsMarketMakerProtectionRequest) { r.DeltaLimit = -1 }, err: errInvalidMMPLimit},
	} {
		arg := newRequest()
		tc.modify(arg)
		_, err = e.SetOptionsMarketMakerProtection(t.Context(), arg)
		require.ErrorIsf(t, err, tc.err, "SetOptionsMarketMakerProtection must reject an invalid %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetOptionsMarketMakerProtection(t.Context(), newRequest())
	require.NoError(t, err, "SetOptionsMarketMakerProtection must not error")
	if mockTests {
		assert.Equal(t, optionsMarketMakerProtectionFixture(), result, "SetOptionsMarketMakerProtection should decode every field")
		return
	}
	assert.Equal(t, "BTC_USDT", result.Underlying, "SetOptionsMarketMakerProtection should return the underlying's settings")
}

func TestOptionsMarketMakerProtectionRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		request *OptionsMarketMakerProtectionRequest
		exp     string
	}{
		{
			name: "enabled",
			request: &OptionsMarketMakerProtectionRequest{
				Underlying:    "BTC_USDT",
				Window:        5 * time.Second,
				FrozenPeriod:  200 * time.Millisecond,
				QuantityLimit: 10,
				DeltaLimit:    2.5,
			},
			exp: `{"underlying":"BTC_USDT","window":5000,"frozen_period":200,"qty_limit":"10","delta_limit":"2.5"}`,
		},
		{
			name:    "disabled until reset",
			request: &OptionsMarketMakerProtectionRequest{Underlying: "BTC_USDT", QuantityLimit: 1, DeltaLimit: 1},
			exp:     `{"underlying":"BTC_USDT","window":0,"frozen_period":0,"qty_limit":"1","delta_limit":"1"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.request)
			require.NoError(t, err, "Marshal must not error")
			assert.JSONEq(t, tc.exp, string(got), "Marshal should send the documented encoding")
		})
	}
}

func TestOptionsMarketMakerProtectionResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result OptionsMarketMakerProtectionResponse
	err := json.Unmarshal([]byte(`{"underlying":"BTC_USDT","window":5000,"frozen_period":200,"qty_limit":"10","delta_limit":"2.5","trigger_time_ms":1744110000123,"frozen_until_ms":1744110000323}`), &result)
	require.NoError(t, err, "Unmarshal must not error")
	exp := OptionsMarketMakerProtectionResponse{
		Underlying:    "BTC_USDT",
		Window:        types.DurationMilliseconds(5 * time.Second),
		FrozenPeriod:  types.DurationMilliseconds(200 * time.Millisecond),
		QuantityLimit: 10,
		DeltaLimit:    2.5,
		TriggerTime:   types.Time(time.UnixMilli(1744110000123)),
		FrozenUntil:   types.Time(time.UnixMilli(1744110000323)),
	}
	assert.Equal(t, exp, result, "Unmarshal should decode the window and frozen period from milliseconds")

	err = json.Unmarshal([]byte(`{"window":"5s"}`), &result)
	assert.Error(t, err, "Unmarshal should reject a window that is not a number of milliseconds")
}

func TestGetOptionsMarketMakerProtection(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOptionsMarketMakerProtection(t.Context(), "BTC_USDT")
	require.NoError(t, err, "GetOptionsMarketMakerProtection must not error")
	if mockTests {
		exp := optionsMarketMakerProtectionFixture()
		exp.TriggerTime = types.Time(time.UnixMilli(1744110000123))
		exp.FrozenUntil = types.Time(time.UnixMilli(1744110000323))
		assert.Equal(t, []*OptionsMarketMakerProtectionResponse{exp}, result, "GetOptionsMarketMakerProtection should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionsMarketMakerProtection should return a list")
}

func TestResetOptionsMarketMakerProtection(t *testing.T) {
	t.Parallel()
	_, err := e.ResetOptionsMarketMakerProtection(t.Context(), "")
	require.ErrorIs(t, err, errInvalidUnderlying, "ResetOptionsMarketMakerProtection must reject an empty underlying")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ResetOptionsMarketMakerProtection(t.Context(), "BTC_USDT")
	require.NoError(t, err, "ResetOptionsMarketMakerProtection must not error")
	if mockTests {
		assert.Equal(t, optionsMarketMakerProtectionFixture(), result, "ResetOptionsMarketMakerProtection should decode every field")
		return
	}
	assert.Equal(t, "BTC_USDT", result.Underlying, "ResetOptionsMarketMakerProtection should return the underlying's settings")
}

// optionsMarketMakerProtectionFixture returns the untriggered market maker protection recorded in testdata/http.json
func optionsMarketMakerProtectionFixture() *OptionsMarketMakerProtectionResponse {
	return &OptionsMarketMakerProtectionResponse{
		Underlying:    "BTC_USDT",
		Window:        types.DurationMilliseconds(5 * time.Second),
		FrozenPeriod:  types.DurationMilliseconds(200 * time.Millisecond),
		QuantityLimit: 10,
		DeltaLimit:    10,
	}
}

func TestGetOptionsOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionsOrderbook(t.Context(), currency.EMPTYPAIR, "0.1", 9, true)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOptionsOrderbook must reject an empty contract")

	result, err := e.GetOptionsOrderbook(t.Context(), getPair(t, asset.Options), "0.1", 9, true)
	require.NoError(t, err, "GetOptionsOrderbook must not error")
	if mockTests {
		exp := &OrderbookResponse{
			ID:           2836984,
			ResponseTime: types.Time(time.UnixMilli(1774970474039)),
			UpdateTime:   types.Time(time.UnixMilli(1774970470308)),
			Bids:         OrderbookLevels{{Price: 13588, Amount: 102}, {Price: 13487, Amount: 144}, {Price: 11639, Amount: 244}, {Price: 11611, Amount: 182}, {Price: 11584, Amount: 173}},
			Asks:         OrderbookLevels{{Price: 13676, Amount: 600}, {Price: 13817, Amount: 102}, {Price: 14577, Amount: 214}, {Price: 14605, Amount: 230}, {Price: 14632, Amount: 197}},
		}
		assert.Equal(t, exp, result, "GetOptionsOrderbook should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionsOrderbook should return an orderbook")
}

func TestGetOptionsTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionsTickers(t.Context(), "")
	require.ErrorIs(t, err, errInvalidUnderlying, "GetOptionsTickers must reject an empty underlying")

	result, err := e.GetOptionsTickers(t.Context(), "BTC_USDT")
	require.NoError(t, err, "GetOptionsTickers must not error")
	if mockTests {
		require.Len(t, result, 5, "GetOptionsTickers must return every recorded ticker")
		exp := &OptionsTickerResponse{
			Name:                  currency.NewPairWithDelimiter("BTC", "USDT-20260408-73000-P", currency.UnderscoreDelimiter),
			MarkPrice:             5532.7,
			IndexPrice:            67503.71,
			Ask1Size:              2000,
			Ask1Price:             5672,
			Bid1Size:              2000,
			Bid1Price:             5348,
			Vega:                  2.92983,
			Theta:                 -28.33781,
			Gamma:                 0.00002,
			Delta:                 -0.97772,
			MarkImpliedVolatility: 0.4729,
			AskImpliedVolatility:  0.7037,
			Leverage:              11.926376189058,
		}
		assert.Equal(t, exp, result[0], "GetOptionsTickers should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetOptionsTickers should return tickers")
}

func TestGetOptionUnderlyingTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionUnderlyingTickers(t.Context(), "")
	require.ErrorIs(t, err, errInvalidUnderlying, "GetOptionUnderlyingTickers must reject an empty underlying")

	result, err := e.GetOptionUnderlyingTickers(t.Context(), "BTC_USDT")
	require.NoError(t, err, "GetOptionUnderlyingTickers must not error")
	if mockTests {
		assert.Equal(t, &OptionsUnderlyingTickerResponse{TradePut: 278, TradeCall: 1345, IndexPrice: 67503.71}, result, "GetOptionUnderlyingTickers should decode every field")
		return
	}
	assert.Positive(t, result.IndexPrice.Float64(), "GetOptionUnderlyingTickers should return an index price")
}

func TestGetOptionFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetOptionFuturesCandlesticks(t.Context(), currency.EMPTYPAIR, kline.ThirtyMin, startTime, endTime, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOptionFuturesCandlesticks must reject an empty contract")

	_, err = e.GetOptionFuturesCandlesticks(t.Context(), getPair(t, asset.Options), kline.ThirtyMin, endTime, startTime, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOptionFuturesCandlesticks must reject a start after the end")

	_, err = e.GetOptionFuturesCandlesticks(t.Context(), getPair(t, asset.Options), kline.Interval(5), startTime, endTime, 100)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetOptionFuturesCandlesticks must reject an unsupported interval")

	result, err := e.GetOptionFuturesCandlesticks(t.Context(), getPair(t, asset.Options), kline.ThirtyMin, startTime, endTime, 1000)
	require.NoError(t, err, "GetOptionFuturesCandlesticks must not error")
	if mockTests {
		assert.Empty(t, result, "GetOptionFuturesCandlesticks should return the recorded empty range")
		return
	}
	assert.NotNil(t, result, "GetOptionFuturesCandlesticks should return a list")
}

func TestGetOptionFuturesMarkPriceCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionFuturesMarkPriceCandlesticks(t.Context(), "", 0, time.Time{}, time.Time{}, kline.OneMonth)
	require.ErrorIs(t, err, errInvalidUnderlying, "GetOptionFuturesMarkPriceCandlesticks must reject an empty underlying")

	startTime, endTime := getTime()
	_, err = e.GetOptionFuturesMarkPriceCandlesticks(t.Context(), "BTC_USDT", 0, endTime, startTime, kline.OneMonth)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOptionFuturesMarkPriceCandlesticks must reject a start after the end")

	result, err := e.GetOptionFuturesMarkPriceCandlesticks(t.Context(), "BTC_USDT", 0, time.Time{}, time.Time{}, kline.OneMonth)
	require.NoError(t, err, "GetOptionFuturesMarkPriceCandlesticks must not error")
	if mockTests {
		exp := []*FuturesCandlestickResponse{
			{Timestamp: types.Time(time.Unix(1770336000, 0)), ClosePrice: 67264.82, HighestPrice: 74027.87, LowestPrice: 65056.81, OpenPrice: 67126.36},
			{Timestamp: types.Time(time.Unix(1772928000, 0)), ClosePrice: 67116.1, HighestPrice: 75991.85, LowestPrice: 65005.28, OpenPrice: 67264.15},
		}
		assert.Equal(t, exp, result, "GetOptionFuturesMarkPriceCandlesticks should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetOptionFuturesMarkPriceCandlesticks should return candles")
}

func TestGetOptionsTradeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetOptionsTradeHistory(t.Context(), getPair(t, asset.Options), "C", 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOptionsTradeHistory must reject a start after the end")

	_, err = e.GetOptionsTradeHistory(t.Context(), getPair(t, asset.Options), "C", 0, 0, startTime, endTime)
	require.NoError(t, err, "GetOptionsTradeHistory must not error for a time range")

	pair := getPair(t, asset.Options)
	result, err := e.GetOptionsTradeHistory(t.Context(), pair, "", 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err, "GetOptionsTradeHistory must not error")
	if mockTests {
		require.Len(t, result, 3, "GetOptionsTradeHistory must return every recorded trade")
		exp := &OptionTradeResponse{
			ID:         3,
			CreateTime: types.Time(time.Unix(1775386206, 0)),
			Contract:   pair,
			Size:       -1,
			Price:      12988,
		}
		assert.Equal(t, exp, result[0], "GetOptionsTradeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionsTradeHistory should return a list")
}
