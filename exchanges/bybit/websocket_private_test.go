package bybit

import (
	"bytes"
	"strings"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// livePrivateSubscription authenticates a private stream connection with the test credentials and subscribes to
// topics, skipping without credentials. Private pushes follow account activity, so only the subscription is checked
func livePrivateSubscription(t *testing.T, topics ...string) {
	t.Helper()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	conn, resp, err := gws.DefaultDialer.DialContext(t.Context(), websocketPrivate, nil)
	if resp != nil {
		resp.Body.Close()
	}
	require.NoError(t, err, "Dial must not error")
	defer conn.Close()
	read := make(chan []byte, 16)
	go func() {
		defer close(read)
		for {
			_, p, err := conn.ReadMessage()
			if err != nil {
				return
			}
			select {
			case read <- p:
			default:
			}
		}
	}()
	auth, err := e.GetAuthenticationPayload(accounts.DeployCredentialsToContext(t.Context(), apiCredentials), "auth")
	require.NoError(t, err, "GetAuthenticationPayload must not error")
	for _, req := range []any{auth, &SubscriptionRequest{RequestID: "subscribe", Operation: subscribeOperation, Arguments: topics}} {
		require.NoError(t, conn.WriteJSON(req), "WriteJSON must not error")
		select {
		case p, ok := <-read:
			require.True(t, ok, "stream must stay open")
			var reply SubscriptionResponse
			require.NoError(t, json.Unmarshal(p, &reply), "reply must decode")
			require.Truef(t, reply.Success, "request must succeed: %s", p)
		case <-time.After(10 * time.Second):
			require.Fail(t, "request must be answered")
		}
	}
}

func TestWebsocketHandleAuthenticatedDataTopics(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	err := ex.wsHandleAuthenticatedData(t.Context(), nil, []byte(`{"id":"1","topic":"dcp.future","creationTime":1672364262482,"data":[]}`))
	require.ErrorIs(t, err, errUnhandledStreamData, "wsHandleAuthenticatedData must error for a topic without documented pushes")
	assert.ErrorContains(t, err, "dcp.future", "error should name the topic")
	err = ex.wsHandleAuthenticatedData(t.Context(), nil, []byte(`{"topic":1}`))
	require.Error(t, err, "wsHandleAuthenticatedData must error for a topic that is not a string")
	conn := &FixtureConnection{match: websocket.NewMatch()}
	err = ex.wsHandleAuthenticatedData(t.Context(), conn, []byte(`{"req_id":"test","op":"pong","args":["1675418560633"],"conn_id":"cfcb4ocsvfriu23r3er0-1b"}`))
	require.NoError(t, err, "wsHandleAuthenticatedData must handle a pong")
	for _, topic := range []string{"position", "execution", "execution.fast", "order", "wallet", "greeks", "strategy"} {
		err = ex.wsHandleAuthenticatedData(t.Context(), conn, []byte(`{"topic":"`+topic+`","data":"bad"}`))
		assert.Errorf(t, err, "wsHandleAuthenticatedData should error for an undecodable %s push", topic)
	}

	// Spread, RFQ and event contract pushes share the private connection and are relayed as their own frames; event
	// executions and orders also arrive on the generic execution and order topics
	for _, tc := range []struct {
		fixture string
		exp     []any // the frame each push should be relayed as, decoded from the push
	}{
		{fixture: "testdata/wsSpreadPrivate.json", exp: []any{new(WebsocketSpreadOrderUpdate), new(WebsocketSpreadExecutionUpdate)}},
		{fixture: "testdata/wsRFQPrivate.json", exp: []any{new(WebsocketRFQQuoteUpdate), new(WebsocketRFQQuoteUpdate), new(WebsocketRFQTradeUpdate), new(WebsocketRFQInquiryUpdate)}},
		{fixture: "testdata/wsEventPrivate.json", exp: []any{new(WebsocketEventPositionUpdate), new(WebsocketEventExecutionUpdate), new(WebsocketEventExecutionUpdate), new(WebsocketEventOrderUpdate), new(WebsocketEventOrderUpdate)}},
	} {
		frames := fixtureFrames(t, tc.fixture)
		require.Lenf(t, frames, len(tc.exp), "%s must hold one push per expected frame", tc.fixture)
		ex := websocketTestInstance(t)
		for i, frame := range frames {
			require.NoErrorf(t, json.Unmarshal(frame, tc.exp[i]), "Unmarshal must not error for %s push %d", tc.fixture, i)
			require.NoErrorf(t, ex.wsHandleAuthenticatedData(t.Context(), nil, frame), "wsHandleAuthenticatedData must not error for %s push %d", tc.fixture, i)
		}
		assert.Equalf(t, tc.exp, relayed(ex), "wsHandleAuthenticatedData should relay each %s push as its topic's frame", tc.fixture)
	}
}

func TestWebsocketPosition(t *testing.T) {
	t.Parallel()
	if !mockTests {
		livePrivateSubscription(t, "position")
		return
	}
	ex := streamTestInstance(t)
	frames := fixtureFrames(t, "testdata/wsPrivatePosition.json")
	pushes := decodeFramesStrictly[WebsocketPositionUpdate](t, frames)
	expPushes := []WebsocketPositionUpdate{
		{
			ID:           "1003076014fb7eedb-c7e6-45d6-a8c1-270f0169171a",
			Topic:        "position",
			CreationTime: types.Time(time.UnixMilli(1697682317044)),
			Data: []WebsocketPosition{{
				Category:                               "linear",
				Symbol:                                 "BTCUSDT",
				Side:                                   "Buy",
				Size:                                   preciseNumber(t, "0.15"),
				PositionIndex:                          1,
				PositionValue:                          preciseNumber(t, "12361.545"),
				RiskID:                                 1,
				RiskLimitValue:                         preciseNumber(t, "2000000"),
				EntryPrice:                             preciseNumber(t, "82410.3"),
				MarkPrice:                              preciseNumber(t, "82559.7"),
				Leverage:                               preciseNumber(t, "10"),
				BreakEvenPrice:                         preciseNumber(t, "82455.6"),
				AutoAddMargin:                          true,
				PositionInitialMargin:                  preciseNumber(t, "1243.0204"),
				PositionMaintenanceMargin:              preciseNumber(t, "68.71"),
				LiquidationPrice:                       preciseNumber(t, "74535.8"),
				TakeProfit:                             preciseNumber(t, "90000"),
				StopLoss:                               preciseNumber(t, "78000"),
				TrailingStop:                           preciseNumber(t, "500"),
				UnrealisedPNL:                          preciseNumber(t, "22.41"),
				CurrentRealisedPNL:                     preciseNumber(t, "-6.79884975"),
				SessionAveragePrice:                    preciseNumber(t, "82410.3"),
				NetDeltaRatio:                          preciseNumber(t, "0.25"),
				CumulativeRealisedPNL:                  preciseNumber(t, "-25.06579337"),
				PositionStatus:                         "Normal",
				ADLRankIndicator:                       2,
				IsReduceOnly:                           true,
				CreatedTime:                            types.Time(time.UnixMilli(1694402496913)),
				UpdatedTime:                            types.Time(time.UnixMilli(1697682317038)),
				OpenTime:                               types.Time(time.UnixMilli(1697600017038)),
				Sequence:                               8327597863,
				MaintenanceMarginRateSystemUpdatedTime: types.Time(time.UnixMilli(1697682300000)),
				LeverageSystemUpdatedTime:              types.Time(time.UnixMilli(1697682300000)),
				PositionInitialMarginByMarkPrice:       preciseNumber(t, "1243.0204"),
				PositionMaintenanceMarginByMarkPrice:   preciseNumber(t, "68.71"),
				TPSLMode:                               "Full",
				BustPrice:                              preciseNumber(t, "74169.3"),
				PositionBalance:                        preciseNumber(t, "1236.1545"),
			}},
		},
		{
			ID:           "1003076014fb7eedb-c7e6-45d6-a8c1-270f01691720",
			Topic:        "position.option",
			CreationTime: types.Time(time.UnixMilli(1697682318044)),
			Data: []WebsocketPosition{{
				Category:              "option",
				Symbol:                "BTC-24SEP27-170000-P-USDT",
				Side:                  "Sell",
				Size:                  preciseNumber(t, "0.5"),
				PositionValue:         preciseNumber(t, "42067.19"),
				RiskLimitValue:        preciseNumber(t, "0"),
				EntryPrice:            preciseNumber(t, "84130"),
				MarkPrice:             preciseNumber(t, "84134.38654221"),
				TakeProfit:            preciseNumber(t, "0"),
				StopLoss:              preciseNumber(t, "0"),
				TrailingStop:          preciseNumber(t, "0"),
				UnrealisedPNL:         preciseNumber(t, "-2.19"),
				CurrentRealisedPNL:    preciseNumber(t, "-1.25"),
				Delta:                 preciseNumber(t, "0.45963833"),
				Gamma:                 preciseNumber(t, "-0.00000207"),
				Vega:                  preciseNumber(t, "-63.75695203"),
				Theta:                 preciseNumber(t, "3.87428383"),
				CumulativeRealisedPNL: preciseNumber(t, "-1.25"),
				PositionStatus:        "Normal",
				CreatedTime:           types.Time(time.UnixMilli(1697682317044)),
				UpdatedTime:           types.Time(time.UnixMilli(1697682318040)),
				OpenTime:              types.Time(time.UnixMilli(1697682317044)),
				Sequence:              -1,
				TPSLMode:              "Full",
				PositionBalance:       preciseNumber(t, "0"),
				TradeMode:             1,
			}},
		},
	}
	assert.Equal(t, expPushes, pushes, "fixture frames should decode every position field")
	for i, frame := range frames {
		require.NoErrorf(t, ex.wsHandleAuthenticatedData(t.Context(), nil, frame), "wsHandleAuthenticatedData must not error for frame %d", i)
	}
	exp := []any{
		[]futures.Position{{
			Exchange:                     ex.Name,
			Asset:                        asset.USDTMarginedFutures,
			Pair:                         currency.NewBTCUSDT(),
			Underlying:                   currency.BTC,
			CollateralCurrency:           currency.USDT,
			Leverage:                     decimal.MustFromString("10"),
			NotionalSize:                 decimal.MustFromString("12361.545"),
			PositionMargin:               decimal.MustFromString("1243.0204"),
			InitialMarginRequirement:     decimal.MustFromString("1243.0204"),
			MaintenanceMarginRequirement: decimal.MustFromString("68.71"),
			EstimatedLiquidationPrice:    decimal.MustFromString("74535.8"),
			UpdateID:                     8327597863,
			RealisedPNL:                  decimal.MustFromString("-6.79884975"),
			UnrealisedPNL:                decimal.MustFromString("22.41"),
			Status:                       order.Open,
			OpeningDate:                  time.UnixMilli(1697600017038),
			OpeningPrice:                 decimal.MustFromString("82410.3"),
			OpeningDirection:             order.Long,
			LatestPrice:                  decimal.MustFromString("82559.7"),
			LatestSize:                   decimal.MustFromString("0.15"),
			LatestDirection:              order.Long,
			LastUpdated:                  time.UnixMilli(1697682317038),
		}},
		[]futures.Position{{
			Exchange:                     ex.Name,
			Asset:                        asset.Options,
			Pair:                         fixtureOptionPair,
			Underlying:                   currency.BTC,
			CollateralCurrency:           currency.USDT,
			Leverage:                     decimal.Zero,
			NotionalSize:                 decimal.MustFromString("42067.19"),
			PositionMargin:               decimal.Zero,
			InitialMarginRequirement:     decimal.Zero,
			MaintenanceMarginRequirement: decimal.Zero,
			EstimatedLiquidationPrice:    decimal.Zero,
			UpdateID:                     -1,
			RealisedPNL:                  decimal.MustFromString("-1.25"),
			UnrealisedPNL:                decimal.MustFromString("-2.19"),
			Status:                       order.Open,
			OpeningDate:                  time.UnixMilli(1697682317044),
			OpeningPrice:                 decimal.MustFromString("84130"),
			OpeningDirection:             order.Short,
			LatestPrice:                  decimal.MustFromString("84134.38654221"),
			LatestSize:                   decimal.MustFromString("0.5"),
			LatestDirection:              order.Short,
			LastUpdated:                  time.UnixMilli(1697682318040),
		}},
	}
	assert.Equal(t, exp, relayed(ex), "wsHandleAuthenticatedData should relay each position as a futures position")
}

// preciseNumber returns the PreciseNumber s decodes to
func preciseNumber(t *testing.T, s string) types.PreciseNumber {
	t.Helper()
	n, err := types.NewPreciseNumberFromString(s)
	require.NoErrorf(t, err, "NewPreciseNumberFromString must not error for %s", s)
	return n
}

func TestWebsocketPositionStates(t *testing.T) {
	t.Parallel()
	payload := `{"id":"74199870_position_1787197420650","topic":"position","creationTime":1787197420650,"data":[{"positionIdx":0,"tradeMode":0,"riskId":1,"riskLimitValue":"25000","symbol":"BBUSDT","side":"Sell","size":"883","entryPrice":"0.01105794","sessionAvgPrice":"","leverage":"1","positionValue":"7.403955","positionBalance":"0","markPrice":"0.008385","positionIM":"7.41469557","positionMM":"0.15881967","positionIMByMp":"7.41469557","positionMMByMp":"0.15881967","takeProfit":"0","stopLoss":"0","trailingStop":"0","unrealisedPnl":"2.360203","cumRealisedPnl":"0.00281412","curRealisedPnl":"0.00281412","createdTime":"1728875391204","updatedTime":"1787197420648","tpslMode":"Full","liqPrice":"0.20927497","bustPrice":"","category":"linear","positionStatus":"Normal","adlRankIndicator":2,"autoAddMargin":0,"leverageSysUpdatedTime":"","mmrSysUpdatedTime":"","seq":169498457174,"openTime":1786983352599,"breakEvenPrice":"0.01105504","isReduceOnly":false}]}`
	pair := currency.NewPair(currency.NewCode("BB"), currency.USDT)
	for _, tt := range []struct {
		name                string
		replacements        []string
		expectedStatus      order.Status
		expectedRiskStatus  order.Status
		expectedDirection   order.Side
		expectedCloseDate   time.Time
		expectedOpeningDate time.Time
	}{
		{name: "sell", expectedStatus: order.Open, expectedDirection: order.Short, expectedOpeningDate: time.UnixMilli(1786983352599)},
		{name: "buy", replacements: []string{`"side":"Sell"`, `"side":"Buy"`}, expectedStatus: order.Open, expectedDirection: order.Long, expectedOpeningDate: time.UnixMilli(1786983352599)},
		{name: "flat", replacements: []string{`"side":"Sell"`, `"side":""`, `"size":"883"`, `"size":"0"`}, expectedStatus: order.Closed, expectedDirection: order.UnknownSide, expectedCloseDate: time.UnixMilli(1787197420648), expectedOpeningDate: time.UnixMilli(1786983352599)},
		{name: "closed long hedge leg", replacements: []string{`"positionIdx":0`, `"positionIdx":1`, `"side":"Sell"`, `"side":""`, `"size":"883"`, `"size":"0"`}, expectedStatus: order.Closed, expectedDirection: order.Long, expectedCloseDate: time.UnixMilli(1787197420648), expectedOpeningDate: time.UnixMilli(1786983352599)},
		{name: "closed short hedge leg", replacements: []string{`"positionIdx":0`, `"positionIdx":2`, `"side":"Sell"`, `"side":""`, `"size":"883"`, `"size":"0"`}, expectedStatus: order.Closed, expectedDirection: order.Short, expectedCloseDate: time.UnixMilli(1787197420648), expectedOpeningDate: time.UnixMilli(1786983352599)},
		{name: "liquidating", replacements: []string{`"positionStatus":"Normal"`, `"positionStatus":"Liq"`}, expectedStatus: order.Open, expectedRiskStatus: order.Liquidated, expectedDirection: order.Short, expectedOpeningDate: time.UnixMilli(1786983352599)},
		{name: "auto deleveraging", replacements: []string{`"positionStatus":"Normal"`, `"positionStatus":"Adl"`}, expectedStatus: order.Open, expectedRiskStatus: order.AutoDeleverage, expectedDirection: order.Short, expectedOpeningDate: time.UnixMilli(1786983352599)},
		{name: "liquidated flat", replacements: []string{`"side":"Sell"`, `"side":""`, `"size":"883"`, `"size":"0"`, `"positionStatus":"Normal"`, `"positionStatus":"Liq"`}, expectedStatus: order.Closed, expectedRiskStatus: order.Liquidated, expectedDirection: order.UnknownSide, expectedCloseDate: time.UnixMilli(1787197420648), expectedOpeningDate: time.UnixMilli(1786983352599)},
		{name: "auto deleveraged flat", replacements: []string{`"side":"Sell"`, `"side":""`, `"size":"883"`, `"size":"0"`, `"positionStatus":"Normal"`, `"positionStatus":"Adl"`}, expectedStatus: order.Closed, expectedRiskStatus: order.AutoDeleverage, expectedDirection: order.UnknownSide, expectedCloseDate: time.UnixMilli(1787197420648), expectedOpeningDate: time.UnixMilli(1786983352599)},
		{name: "missing open time", replacements: []string{`"openTime":1786983352599`, `"openTime":0`}, expectedStatus: order.Open, expectedDirection: order.Short},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ex := websocketTestInstance(t)
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDTMarginedFutures, currency.Pairs{pair}, false), "StorePairs must store the captured position pair")
			require.NoError(t, ex.wsProcessPosition(t.Context(), []byte(strings.NewReplacer(tt.replacements...).Replace(payload))), "wsProcessPosition must process the captured position")
			require.Len(t, ex.Websocket.DataHandler.C, 1, "wsProcessPosition must relay the position")
			message := <-ex.Websocket.DataHandler.C
			positions, ok := message.Data.([]futures.Position)
			require.True(t, ok, "position handler must emit canonical futures positions")
			require.Len(t, positions, 1, "position handler must emit one position")
			assert.Equal(t, tt.expectedStatus, positions[0].Status, "position status should be canonical")
			assert.Equal(t, tt.expectedRiskStatus, positions[0].RiskStatus, "position risk status should preserve liquidation state")
			assert.Equal(t, positions[0].LatestSize.IsZero(), positions[0].Status.IsInactive(), "position lifecycle should agree with its latest size")
			assert.Equal(t, tt.expectedDirection, positions[0].OpeningDirection, "opening direction should be canonical")
			assert.Equal(t, tt.expectedDirection, positions[0].LatestDirection, "latest direction should be canonical")
			assert.Equal(t, tt.expectedCloseDate, positions[0].CloseDate, "position close time should be correct")
			assert.Equal(t, tt.expectedOpeningDate, positions[0].OpeningDate, "position opening time should use only Bybit's open time")
			assert.Equal(t, "0.00281412", positions[0].RealisedPNL.String(), "position realised PNL should use the current position value")
			assert.Equal(t, "0.20927497", positions[0].EstimatedLiquidationPrice.String(), "position liquidation price should be correct")
			assert.Equal(t, "7.41469557", positions[0].PositionMargin.String(), "position margin should preserve Bybit's initial margin")
			assert.Equal(t, positions[0].PositionMargin, positions[0].InitialMarginRequirement, "initial margin requirement should match Bybit's position margin")
			assert.Equal(t, "0.15881967", positions[0].MaintenanceMarginRequirement.String(), "maintenance margin requirement should preserve Bybit's absolute margin")
			assert.True(t, positions[0].MaintenanceMarginFraction.IsZero(), "maintenance margin fraction should remain unset without Bybit risk tier data")
		})
	}
	ex := websocketTestInstance(t)
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDTMarginedFutures, currency.Pairs{pair}, false), "StorePairs must store the captured position pair")
	err := ex.wsProcessPosition(t.Context(), []byte(strings.Replace(payload, `"side":"Sell"`, `"side":"Any"`, 1)))
	require.ErrorIs(t, err, order.ErrPositionSideUnsupported, "wsProcessPosition must reject a non-directional position side")
	err = ex.wsProcessPosition(t.Context(), []byte(strings.Replace(payload, `"side":"Sell"`, `"side":"Hold"`, 1)))
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "wsProcessPosition must reject an unknown position side")
	err = ex.wsProcessPosition(t.Context(), []byte(strings.Replace(payload, `"symbol":"BBUSDT"`, `"symbol":"ZZZQUSDT"`, 1)))
	require.ErrorIs(t, err, currency.ErrPairNotFound, "wsProcessPosition must reject an unknown pair")
	err = ex.wsProcessPosition(t.Context(), []byte(strings.NewReplacer(`"category":"linear"`, `"category":"spot"`, `"symbol":"BBUSDT"`, `"symbol":"BTCUSDT"`).Replace(payload)))
	require.ErrorIs(t, err, asset.ErrNotSupported, "wsProcessPosition must reject a spot position")
	assert.Empty(t, relayed(ex), "wsProcessPosition should relay nothing for a push whose only position it rejects")
	require.NoError(t, ex.wsProcessPosition(t.Context(), []byte(`{"topic":"position","data":[]}`)), "wsProcessPosition must not error for a push without positions")
	assert.Empty(t, relayed(ex), "wsProcessPosition should not relay an empty batch")

	// A push can carry several positions, such as both legs of a hedge mode position, and every one of them is relayed
	position := payload[strings.IndexByte(payload, '[')+1 : strings.LastIndexByte(payload, ']')]
	shortLeg := strings.Replace(position, `"positionIdx":0`, `"positionIdx":2`, 1)
	longLeg := strings.NewReplacer(`"positionIdx":0`, `"positionIdx":1`, `"side":"Sell"`, `"side":"Buy"`).Replace(position)
	require.NoError(t, ex.wsProcessPosition(t.Context(), []byte(strings.Replace(payload, position, shortLeg+","+longLeg, 1))), "wsProcessPosition must process both legs of a hedge mode position")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "wsProcessPosition must relay the push")
	positions, ok := (<-ex.Websocket.DataHandler.C).Data.([]futures.Position)
	require.True(t, ok, "wsProcessPosition must relay canonical futures positions")
	require.Len(t, positions, 2, "wsProcessPosition must relay every position of a push")
	assert.Equal(t, order.Short, positions[0].LatestDirection, "wsProcessPosition should relay the short leg first, as it was pushed")
	assert.Equal(t, order.Long, positions[1].LatestDirection, "wsProcessPosition should relay the long leg second, as it was pushed")

	// A position that cannot be read is reported without holding back the push's other positions
	require.NoError(t, ex.wsProcessPosition(t.Context(), []byte(payload)), "wsProcessPosition must process the captured position")
	exp := relayed(ex)
	require.Len(t, exp, 1, "wsProcessPosition must relay the captured position")
	unknownPair := strings.Replace(position, `"symbol":"BBUSDT"`, `"symbol":"BTCUSDT-26SEP25"`, 1)
	unknownSide := strings.NewReplacer(`"positionIdx":0`, `"positionIdx":1`, `"side":"Sell"`, `"side":"Hold"`).Replace(position)
	spotPosition := strings.NewReplacer(`"category":"linear"`, `"category":"spot"`, `"symbol":"BBUSDT"`, `"symbol":"BTCUSDT"`, `"positionIdx":0`, `"positionIdx":2`).Replace(position)
	undecodable := strings.NewReplacer(`"positionIdx":0`, `"positionIdx":2`, `"size":"883"`, `"size":"broken"`).Replace(position)
	mixed := []byte(strings.Replace(payload, position, unknownPair+","+position+","+spotPosition+","+unknownSide+","+undecodable, 1))
	err = ex.wsProcessPosition(t.Context(), mixed)
	require.ErrorIs(t, err, currency.ErrPairNotFound, "wsProcessPosition must report a position whose symbol is not an available pair")
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "wsProcessPosition must report every position it cannot read")
	require.ErrorIs(t, err, asset.ErrNotSupported, "wsProcessPosition must report a position without a collateral currency")
	assert.ErrorContains(t, err, "linear BTCUSDT-26SEP25 position index 0", "error should identify the position of the unknown pair")
	assert.ErrorContains(t, err, "linear BBUSDT position index 1", "error should identify the position of the unknown side")
	assert.ErrorContains(t, err, "spot BTCUSDT position index 2", "error should identify the position without a collateral currency")
	assert.ErrorContains(t, err, "linear BBUSDT position index 2", "error should identify the position that cannot be decoded")
	assert.Equal(t, exp, relayed(ex), "wsProcessPosition should relay the push's other positions")

	errBufferFull := fillDataHandler(t, ex)
	require.ErrorIs(t, ex.wsProcessPosition(t.Context(), []byte(payload)), errBufferFull, "wsProcessPosition must return the data handler's error")
	err = ex.wsProcessPosition(t.Context(), mixed)
	require.ErrorIs(t, err, errBufferFull, "wsProcessPosition must return the data handler's error for a push with a failed position")
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "wsProcessPosition should still report the failed position when the relay fails")
}

func TestPositionCollateralCurrency(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		asset    asset.Item
		pair     currency.Pair
		symbol   string
		expected currency.Code
		err      error
	}{
		{name: "USDT margined", asset: asset.USDTMarginedFutures, pair: currency.NewPair(currency.BTC, currency.NewCode("USDT-04SEP26")), symbol: "BTC-USDT-04SEP26", expected: currency.USDT},
		{name: "USDC margined", asset: asset.USDCMarginedFutures, pair: currency.NewPair(currency.BTC, currency.NewCode("PERP")), symbol: "BTC-PERP", expected: currency.USDC},
		{name: "coin margined", asset: asset.CoinMarginedFutures, pair: currency.NewBTCUSD(), symbol: "BTCUSD", expected: currency.BTC},
		{name: "USDT option", asset: asset.Options, pair: currency.NewPair(currency.SOL, currency.NewCode("4MAR26-85-C-USDT")), symbol: "SOL-4MAR26-85-C-USDT", expected: currency.USDT},
		{name: "USDC option", asset: asset.Options, pair: currency.NewPair(currency.BTC, currency.NewCode("29DEC23-70000-P")), symbol: "BTC-29DEC23-70000-P", expected: currency.USDC},
		{name: "unsupported", asset: asset.Spot, pair: currency.NewBTCUSDT(), symbol: "BTCUSDT", err: asset.ErrNotSupported},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual, err := positionCollateralCurrency(tt.asset, tt.pair, tt.symbol)
			require.ErrorIs(t, err, tt.err, "positionCollateralCurrency must return the expected error")
			assert.Equal(t, tt.expected, actual, "positionCollateralCurrency should return the settlement currency")
		})
	}
}

func TestWebsocketExecution(t *testing.T) {
	t.Parallel()
	if !mockTests {
		livePrivateSubscription(t, "execution")
		return
	}
	ex := streamTestInstance(t)
	frames := fixtureFrames(t, "testdata/wsPrivateExecution.json")
	pushes := decodeFramesStrictly[WebsocketExecutionUpdate](t, frames)
	expPushes := []WebsocketExecutionUpdate{
		{
			ID:           "386825804_BTCUSDT_140612148849382",
			Topic:        "execution",
			CreationTime: types.Time(time.UnixMilli(1746270400355)),
			Data: []WebsocketExecution{
				{
					Category:          "linear",
					Symbol:            "BTCUSDT",
					IsLeverage:        "0",
					OrderID:           "9aac161b-8ed6-450d-9cab-c5cc67c21784",
					OrderLinkID:       "linear-close-001",
					Side:              "Sell",
					OrderPrice:        94942.5,
					OrderQuantity:     0.7,
					LeavesQuantity:    0.2,
					CreateType:        "CreateByUser",
					OrderType:         "Market",
					StopOrderType:     "UNKNOWN",
					ExecutionFee:      26.3725275,
					ExecutionID:       "0ab1bdf7-4219-438b-b30a-32ec863018f7",
					ExecutionPrice:    95900.1,
					ExecutionQuantity: 0.5,
					ExecutionPNL:      0.05,
					ExecutionType:     "Trade",
					ExecutionValue:    47950.05,
					ExecutionTime:     types.Time(time.UnixMilli(1746270400353)),
					FeeRate:           0.00055,
					MarkPrice:         95901.48,
					ClosedSize:        0.5,
					ExtraFees: []ExtraFee{
						{
							FeeCoin:    currency.USDT,
							FeeType:    "GST",
							SubFeeType: "IND_GST",
							FeeRate:    0.0000675,
							Fee:        0.006403779,
						},
					},
					Sequence:    140612148849382,
					FeeCurrency: currency.USDT,
				},
			},
		},
		{
			ID:           "386825804_BTCUSDT_115179399571",
			Topic:        "execution.spot",
			CreationTime: types.Time(time.UnixMilli(1791535765240)),
			Data: []WebsocketExecution{
				{
					Category:          "spot",
					Symbol:            "BTCUSDT",
					IsLeverage:        "1",
					OrderID:           "2290000000123456789",
					OrderLinkID:       "spot-buy-001",
					Side:              "Buy",
					OrderPrice:        82600,
					OrderQuantity:     0.000351,
					OrderType:         "Limit",
					ExecutionFee:      0.00000035,
					ExecutionID:       "2290000001228958155",
					ExecutionPrice:    82590.5,
					ExecutionQuantity: 0.000351,
					ExecutionType:     "Trade",
					ExecutionValue:    28.9892655,
					ExecutionTime:     types.Time(time.UnixMilli(1791535765237)),
					IsMaker:           true,
					FeeRate:           0.001,
					Sequence:          115179399571,
					MarketUnit:        "baseCoin",
					FeeCurrency:       currency.BTC,
				},
			},
		},
		{
			ID:           "386825804_option_77634219074",
			Topic:        "execution.option",
			CreationTime: types.Time(time.UnixMilli(1791535798095)),
			Data: []WebsocketExecution{
				{
					Category:               "option",
					Symbol:                 "BTC-24SEP27-170000-P-USDT",
					OrderID:                "5cf98598-39a7-459e-97bf-76ca765ee020",
					OrderLinkID:            "option-sell-001",
					Side:                   "Sell",
					OrderPrice:             84100,
					OrderQuantity:          0.01,
					OrderType:              "Limit",
					ExecutionFee:           1.25,
					ExecutionID:            "20350fa2-6942-57bc-8eb1-07705dd4bab3",
					ExecutionPrice:         84130,
					ExecutionQuantity:      0.01,
					ExecutionPNL:           -1.25,
					ExecutionType:          "Trade",
					ExecutionValue:         841.3,
					ExecutionTime:          types.Time(time.UnixMilli(1791535798090)),
					FeeRate:                0.0003,
					TradeImpliedVolatility: 0.4249,
					MarkImpliedVolatility:  0.4254,
					MarkPrice:              84134.38654221,
					IndexPrice:             82611.83292604,
					UnderlyingPrice:        87001.2,
					BlockTradeID:           "block-7e2f",
					Sequence:               77634219074,
					FeeCurrency:            currency.USDT,
				},
			},
		},
	}
	assert.Equal(t, expPushes, pushes, "fixture frames should decode every execution field")
	for i, frame := range frames {
		require.NoErrorf(t, ex.wsHandleAuthenticatedData(t.Context(), nil, frame), "wsHandleAuthenticatedData must not error for frame %d", i)
	}
	exp := []any{
		[]fill.Data{
			{
				ID:            "0ab1bdf7-4219-438b-b30a-32ec863018f7",
				Timestamp:     time.UnixMilli(1746270400353),
				Exchange:      "TestWebsocketExecution",
				AssetType:     asset.USDTMarginedFutures,
				CurrencyPair:  currency.NewBTCUSDT(),
				Side:          order.Sell,
				OrderID:       "9aac161b-8ed6-450d-9cab-c5cc67c21784",
				ClientOrderID: "linear-close-001",
				Price:         95900.1,
				Amount:        0.5,
			},
		},
		[]fill.Data{
			{
				ID:            "2290000001228958155",
				Timestamp:     time.UnixMilli(1791535765237),
				Exchange:      "TestWebsocketExecution",
				AssetType:     asset.Spot,
				CurrencyPair:  currency.NewBTCUSDT(),
				Side:          order.Buy,
				OrderID:       "2290000000123456789",
				ClientOrderID: "spot-buy-001",
				Price:         82590.5,
				Amount:        0.000351,
			},
		},
		[]fill.Data{
			{
				ID:            "20350fa2-6942-57bc-8eb1-07705dd4bab3",
				Timestamp:     time.UnixMilli(1791535798090),
				Exchange:      "TestWebsocketExecution",
				AssetType:     asset.Options,
				CurrencyPair:  currency.NewPairWithDelimiter("BTC", "24SEP27-170000-P-USDT", "-"),
				Side:          order.Sell,
				OrderID:       "5cf98598-39a7-459e-97bf-76ca765ee020",
				ClientOrderID: "option-sell-001",
				Price:         84130,
				Amount:        0.01,
			},
		},
	}
	assert.Equal(t, exp, relayed(ex), "wsHandleAuthenticatedData should relay each execution as a fill")
	for _, execType := range []string{"Funding", "Settle", "Delivery", "ForwardSplitOpen"} {
		require.NoErrorf(t, ex.wsHandleAuthenticatedData(t.Context(), nil, []byte(`{"topic":"execution","data":[{"category":"linear","symbol":"BTCUSDT","side":"Sell","execType":"`+execType+`"}]}`)), "wsHandleAuthenticatedData must not error for a %s execution", execType)
		assert.Emptyf(t, relayed(ex), "wsHandleAuthenticatedData should not relay a %s execution as a fill", execType)
	}
	err := ex.wsHandleAuthenticatedData(t.Context(), nil, []byte(`{"topic":"execution","data":[{"category":"linear","symbol":"ZZZQUSDT","side":"Sell","execType":"Trade"}]}`))
	require.ErrorIs(t, err, currency.ErrPairNotFound, "wsHandleAuthenticatedData must error for an execution of an unknown pair")
	err = ex.wsHandleAuthenticatedData(t.Context(), nil, []byte(`{"topic":"execution","data":[{"category":"linear","symbol":"BTCUSDT","side":"Hold","execType":"Trade"}]}`))
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "wsHandleAuthenticatedData must error for an execution of an unknown side")
}

func TestWebsocketProcessExecution(t *testing.T) {
	t.Parallel()
	frames := fixtureFrames(t, "testdata/wsPrivateExecution.json")
	require.Len(t, frames, 3, "the execution fixture must hold a linear, a spot and an option push")
	ex := fixtureStreamInstance(t)
	for i, frame := range frames {
		require.NoErrorf(t, ex.wsProcessExecution(t.Context(), frame), "wsProcessExecution must not error for frame %d", i)
	}
	exp := []any{
		[]fill.Data{
			{
				ID:            "0ab1bdf7-4219-438b-b30a-32ec863018f7",
				Timestamp:     time.UnixMilli(1746270400353),
				Exchange:      "TestWebsocketProcessExecution",
				AssetType:     asset.USDTMarginedFutures,
				CurrencyPair:  currency.NewBTCUSDT(),
				Side:          order.Sell,
				OrderID:       "9aac161b-8ed6-450d-9cab-c5cc67c21784",
				ClientOrderID: "linear-close-001",
				Price:         95900.1,
				Amount:        0.5,
			},
		},
		[]fill.Data{
			{
				ID:            "2290000001228958155",
				Timestamp:     time.UnixMilli(1791535765237),
				Exchange:      "TestWebsocketProcessExecution",
				AssetType:     asset.Spot,
				CurrencyPair:  currency.NewBTCUSDT(),
				Side:          order.Buy,
				OrderID:       "2290000000123456789",
				ClientOrderID: "spot-buy-001",
				Price:         82590.5,
				Amount:        0.000351,
			},
		},
		[]fill.Data{
			{
				ID:            "20350fa2-6942-57bc-8eb1-07705dd4bab3",
				Timestamp:     time.UnixMilli(1791535798090),
				Exchange:      "TestWebsocketProcessExecution",
				AssetType:     asset.Options,
				CurrencyPair:  fixtureOptionPair,
				Side:          order.Sell,
				OrderID:       "5cf98598-39a7-459e-97bf-76ca765ee020",
				ClientOrderID: "option-sell-001",
				Price:         84130,
				Amount:        0.01,
			},
		},
	}
	assert.Equal(t, exp, relayed(ex), "wsProcessExecution should relay each trade execution as a fill")

	// The execution topic also pushes funding payments, settlements and deliveries, which are not fills
	funding := `{"category":"linear","symbol":"BTCUSDT","execFee":"0.52","execId":"6f3c2b1a-9d8e-4f7a-b6c5-d4e3f2a1b0c9","execPrice":"95910.2","execQty":"0.2","execType":"Funding","execValue":"19182.04","feeRate":"0.0000271","orderId":"","orderLinkId":"","orderType":"UNKNOWN","side":"Sell","execTime":"1746273600000","seq":140612148849390,"feeCurrency":"USDT"}`
	trade := `{"category":"linear","symbol":"BTCUSDT","execFee":"10.549011","execId":"9b2e7c4d-1a3f-4e5b-8c6d-7e8f9a0b1c2d","execPrice":"95900.1","execQty":"0.2","execType":"Trade","execValue":"19180.02","feeRate":"0.00055","orderId":"9aac161b-8ed6-450d-9cab-c5cc67c21784","orderLinkId":"linear-close-001","orderType":"Market","side":"Sell","execTime":"1746273600353","seq":140612148849391,"feeCurrency":"USDT"}`
	require.NoError(t, ex.wsProcessExecution(t.Context(), []byte(`{"topic":"execution","id":"386825804_BTCUSDT_140612148849391","creationTime":1746273600360,"data":[`+funding+`,`+trade+`]}`)), "wsProcessExecution must not error for a push mixing a funding payment and a trade")
	exp = []any{
		[]fill.Data{
			{
				ID:            "9b2e7c4d-1a3f-4e5b-8c6d-7e8f9a0b1c2d",
				Timestamp:     time.UnixMilli(1746273600353),
				Exchange:      "TestWebsocketProcessExecution",
				AssetType:     asset.USDTMarginedFutures,
				CurrencyPair:  currency.NewBTCUSDT(),
				Side:          order.Sell,
				OrderID:       "9aac161b-8ed6-450d-9cab-c5cc67c21784",
				ClientOrderID: "linear-close-001",
				Price:         95900.1,
				Amount:        0.2,
			},
		},
	}
	assert.Equal(t, exp, relayed(ex), "wsProcessExecution should relay only the trades of a push")
	require.NoError(t, ex.wsProcessExecution(t.Context(), []byte(`{"topic":"execution","id":"386825804_BTCUSDT_140612148849390","creationTime":1746273600010,"data":[`+funding+`]}`)), "wsProcessExecution must not error for a push without trades")
	assert.Empty(t, relayed(ex), "wsProcessExecution should not relay an empty batch")

	// An execution that cannot be read is reported without holding back the push's other executions
	unknownPair := `{"category":"linear","symbol":"BTCUSDT-26SEP25","execId":"5d1f0e3a-7b2c-4d9e-8f6a-1c3b5d7e9f0a","execType":"Trade","side":"Buy"}`
	unknownSide := `{"category":"linear","symbol":"BTCUSDT","execId":"8c4a2e6f-0b1d-4f3a-9e5c-7d2b4a6c8e1f","execType":"Trade","side":"Hold"}`
	undecodable := `{"category":"linear","symbol":"BTCUSDT","execId":"3e7d9b1f-5a2c-4c8e-9d6b-0f1a2b3c4d5e","execType":"Trade","side":"Buy","execQty":"broken"}`
	buy := `{"category":"linear","symbol":"BTCUSDT","execId":"7a6b5c4d-3e2f-4a1b-9c8d-e7f6a5b4c3d2","execPrice":"95899.9","execQty":"0.1","execType":"Trade","orderId":"c3d2e1f0-a9b8-4c7d-6e5f-4a3b2c1d0e9f","orderLinkId":"linear-open-002","side":"Buy","execTime":"1746273600361"}`
	mixed := []byte(`{"topic":"execution","id":"386825804_BTCUSDT_140612148849392","creationTime":1746273600370,"data":[` + unknownPair + `,` + trade + `,` + undecodable + `,` + unknownSide + `,` + buy + `]}`)
	err := ex.wsProcessExecution(t.Context(), mixed)
	require.ErrorIs(t, err, currency.ErrPairNotFound, "wsProcessExecution must report an execution whose symbol is not an available pair")
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "wsProcessExecution must report every execution it cannot read")
	assert.ErrorContains(t, err, "linear BTCUSDT-26SEP25 execution 5d1f0e3a-7b2c-4d9e-8f6a-1c3b5d7e9f0a", "error should identify the execution of the unknown pair")
	assert.ErrorContains(t, err, "linear BTCUSDT execution 8c4a2e6f-0b1d-4f3a-9e5c-7d2b4a6c8e1f", "error should identify the execution of the unknown side")
	assert.ErrorContains(t, err, "linear BTCUSDT execution 3e7d9b1f-5a2c-4c8e-9d6b-0f1a2b3c4d5e", "error should identify the execution that cannot be decoded")
	tradeFills, ok := exp[0].([]fill.Data)
	require.True(t, ok, "the trade must have been relayed as fills")
	exp = []any{append(tradeFills, fill.Data{
		ID:            "7a6b5c4d-3e2f-4a1b-9c8d-e7f6a5b4c3d2",
		Timestamp:     time.UnixMilli(1746273600361),
		Exchange:      "TestWebsocketProcessExecution",
		AssetType:     asset.USDTMarginedFutures,
		CurrencyPair:  currency.NewBTCUSDT(),
		Side:          order.Buy,
		OrderID:       "c3d2e1f0-a9b8-4c7d-6e5f-4a3b2c1d0e9f",
		ClientOrderID: "linear-open-002",
		Price:         95899.9,
		Amount:        0.1,
	})}
	assert.Equal(t, exp, relayed(ex), "wsProcessExecution should relay the push's other trades in turn")

	for _, tc := range []struct {
		name  string
		frame []byte
		err   error
	}{
		{name: "an unknown pair", frame: bytes.Replace(frames[0], []byte(`"symbol":"BTCUSDT"`), []byte(`"symbol":"ZZZQUSDT"`), 1), err: currency.ErrPairNotFound},
		{name: "a category the topic does not serve", frame: bytes.Replace(frames[0], []byte(`"category":"linear"`), []byte(`"category":"event"`), 1), err: errUnsupportedCategory},
		{name: "an unknown side", frame: bytes.Replace(frames[0], []byte(`"side":"Sell"`), []byte(`"side":"Hold"`), 1), err: order.ErrSideIsInvalid},
		{name: "an undecodable push", frame: []byte(`{"topic":"execution","data":"bad"}`)},
	} {
		err := ex.wsProcessExecution(t.Context(), tc.frame)
		if tc.err == nil {
			require.Errorf(t, err, "wsProcessExecution must error for %s", tc.name)
		} else {
			require.ErrorIsf(t, err, tc.err, "wsProcessExecution must error for %s", tc.name)
		}
		assert.Emptyf(t, relayed(ex), "wsProcessExecution should relay nothing for %s", tc.name)
	}

	errBufferFull := fillDataHandler(t, ex)
	require.ErrorIs(t, ex.wsProcessExecution(t.Context(), frames[0]), errBufferFull, "wsProcessExecution must return the data handler's error")
	err = ex.wsProcessExecution(t.Context(), mixed)
	require.ErrorIs(t, err, errBufferFull, "wsProcessExecution must return the data handler's error for a push with a failed execution")
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "wsProcessExecution should still report the failed execution when the relay fails")
}

func TestIsTradeExecution(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		execType string
		exp      bool
	}{
		{execType: "Trade", exp: true},
		{execType: "AdlTrade", exp: true},
		{execType: "BustTrade", exp: true},
		{execType: "BlockTrade", exp: true},
		{execType: "MovePosition", exp: true},
		{execType: "FutureSpread", exp: true},
		{execType: "Funding"},
		{execType: "Delivery"},
		{execType: "Settle"},
		{execType: "ForwardSplitSettle"},
		{execType: "ForwardSplitOpen"},
		{execType: "ReverseSplitSettle"},
		{execType: "ReverseSplitOpen"},
		{execType: "UNKNOWN"},
		{execType: ""},
	} {
		assert.Equalf(t, tc.exp, isTradeExecution(tc.execType), "isTradeExecution should report whether a %q execution is a trade", tc.execType)
	}
}

func TestExtraFeesUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := ExtraFees{{FeeCoin: currency.USDT, FeeType: "GST", SubFeeType: "IND_GST", FeeRate: 0.0000675, Fee: 0.006403779}}
	for _, data := range []string{
		`[{"feeCoin":"USDT","feeType":"GST","subFeeType":"IND_GST","feeRate":"0.0000675","fee":"0.006403779"}]`,
		`"[{\"feeCoin\":\"USDT\",\"feeType\":\"GST\",\"subFeeType\":\"IND_GST\",\"feeRate\":\"0.0000675\",\"fee\":\"0.006403779\"}]"`,
	} {
		var fees ExtraFees
		require.NoErrorf(t, fees.UnmarshalJSON([]byte(data)), "UnmarshalJSON must not error for %s", data)
		assert.Equalf(t, exp, fees, "UnmarshalJSON should decode %s", data)
	}
	for _, data := range []string{`""`, `null`} {
		fees := ExtraFees{{}}
		require.NoErrorf(t, fees.UnmarshalJSON([]byte(data)), "UnmarshalJSON must not error for %s", data)
		assert.Nilf(t, fees, "UnmarshalJSON should decode %s as no fees", data)
	}
	var fees ExtraFees
	require.ErrorIs(t, fees.UnmarshalJSON([]byte(`1`)), errInvalidExtraFees, "UnmarshalJSON must reject a number")
	require.ErrorIs(t, fees.UnmarshalJSON([]byte(`"fees"`)), errInvalidExtraFees, "UnmarshalJSON must reject a string that holds no list")
	require.Error(t, fees.UnmarshalJSON([]byte(`"unterminated`)), "UnmarshalJSON must reject a broken string")
	require.Error(t, fees.UnmarshalJSON([]byte(`[{"fee":true}]`)), "UnmarshalJSON must reject a broken fee")
}

func TestWebsocketFastExecution(t *testing.T) {
	t.Parallel()
	if !mockTests {
		livePrivateSubscription(t, "execution.fast")
		return
	}
	ex := streamTestInstance(t)
	frames := fixtureFrames(t, "testdata/wsPrivateFastExecution.json")
	pushes := decodeFramesStrictly[WebsocketFastExecutionUpdate](t, frames)
	expPushes := []WebsocketFastExecutionUpdate{
		{
			Topic:        "execution.fast",
			CreationTime: types.Time(time.UnixMilli(1716800399338)),
			Data: []WebsocketFastExecution{
				{
					Category:          "linear",
					Symbol:            "BTCUSDT",
					OrderID:           "443d63fa-b4c3-4297-b7b1-23bca88b04dc",
					OrderLinkID:       "test-00001",
					ExecutionID:       "3510f361-0add-5c7b-a2e7-9679810944fc",
					ExecutionPrice:    82560.9,
					ExecutionQuantity: 0.003,
					Side:              "Sell",
					ExecutionTime:     types.Time(time.UnixMilli(1716800399334)),
					Sequence:          34771365464,
				},
				{
					Category:          "linear",
					Symbol:            "BTCUSDT",
					OrderID:           "b1d0e2f4-7c3a-4a51-9f6e-2d8b0c1a3e57",
					IsMaker:           true,
					ExecutionID:       "6e3f9a27-4b0c-5d1e-9f2a-81c7d4e0b5aa",
					ExecutionPrice:    82561,
					ExecutionQuantity: 0.001,
					Side:              "Buy",
					ExecutionTime:     types.Time(time.UnixMilli(1716800399335)),
					Sequence:          34771365465,
				},
			},
		},
	}
	assert.Equal(t, expPushes, pushes, "fixture frames should decode every fast execution field")
	for i, frame := range frames {
		require.NoErrorf(t, ex.wsHandleAuthenticatedData(t.Context(), nil, frame), "wsHandleAuthenticatedData must not error for frame %d", i)
	}
	exp := []any{
		[]fill.Data{
			{
				ID:            "3510f361-0add-5c7b-a2e7-9679810944fc",
				Timestamp:     time.UnixMilli(1716800399334),
				Exchange:      "TestWebsocketFastExecution",
				AssetType:     asset.USDTMarginedFutures,
				CurrencyPair:  currency.NewBTCUSDT(),
				Side:          order.Sell,
				OrderID:       "443d63fa-b4c3-4297-b7b1-23bca88b04dc",
				ClientOrderID: "test-00001",
				Price:         82560.9,
				Amount:        0.003,
			},
			{
				ID:           "6e3f9a27-4b0c-5d1e-9f2a-81c7d4e0b5aa",
				Timestamp:    time.UnixMilli(1716800399335),
				Exchange:     "TestWebsocketFastExecution",
				AssetType:    asset.USDTMarginedFutures,
				CurrencyPair: currency.NewBTCUSDT(),
				Side:         order.Buy,
				OrderID:      "b1d0e2f4-7c3a-4a51-9f6e-2d8b0c1a3e57",
				Price:        82561,
				Amount:       0.001,
			},
		},
	}
	assert.Equal(t, exp, relayed(ex), "wsHandleAuthenticatedData should relay each fast execution as a fill")
	err := ex.wsHandleAuthenticatedData(t.Context(), nil, []byte(`{"topic":"execution.fast.linear","data":[{"category":"linear","symbol":"ZZZQUSDT","side":"Sell"}]}`))
	require.ErrorIs(t, err, currency.ErrPairNotFound, "wsHandleAuthenticatedData must error for a fast execution of an unknown pair")
}

func TestWebsocketProcessFastExecution(t *testing.T) {
	t.Parallel()
	frames := fixtureFrames(t, "testdata/wsPrivateFastExecution.json")
	require.Len(t, frames, 1, "the fast execution fixture must hold one push")
	ex := fixtureStreamInstance(t)
	require.NoError(t, ex.wsProcessFastExecution(t.Context(), frames[0]), "wsProcessFastExecution must not error")
	sell := fill.Data{
		ID:            "3510f361-0add-5c7b-a2e7-9679810944fc",
		Timestamp:     time.UnixMilli(1716800399334),
		Exchange:      "TestWebsocketProcessFastExecution",
		AssetType:     asset.USDTMarginedFutures,
		CurrencyPair:  currency.NewBTCUSDT(),
		Side:          order.Sell,
		OrderID:       "443d63fa-b4c3-4297-b7b1-23bca88b04dc",
		ClientOrderID: "test-00001",
		Price:         82560.9,
		Amount:        0.003,
	}
	buy := fill.Data{
		ID:           "6e3f9a27-4b0c-5d1e-9f2a-81c7d4e0b5aa",
		Timestamp:    time.UnixMilli(1716800399335),
		Exchange:     "TestWebsocketProcessFastExecution",
		AssetType:    asset.USDTMarginedFutures,
		CurrencyPair: currency.NewBTCUSDT(),
		Side:         order.Buy,
		OrderID:      "b1d0e2f4-7c3a-4a51-9f6e-2d8b0c1a3e57",
		Price:        82561,
		Amount:       0.001,
	}
	assert.Equal(t, []any{[]fill.Data{sell, buy}}, relayed(ex), "wsProcessFastExecution should relay the push's executions as fills")

	unknownPair := bytes.Replace(frames[0], []byte(`"symbol":"BTCUSDT"`), []byte(`"symbol":"ZZZQUSDT"`), 1)
	for _, tc := range []struct {
		name  string
		frame []byte
		errs  []error
		named []string // the failed executions the error names
		exp   []any
	}{
		{name: "an unknown pair", frame: unknownPair, errs: []error{currency.ErrPairNotFound}, named: []string{"linear ZZZQUSDT execution 3510f361-0add-5c7b-a2e7-9679810944fc"}, exp: []any{[]fill.Data{buy}}},
		{name: "an unknown side", frame: bytes.Replace(frames[0], []byte(`"side":"Sell"`), []byte(`"side":"Hold"`), 1), errs: []error{order.ErrSideIsInvalid}, named: []string{"linear BTCUSDT execution 3510f361-0add-5c7b-a2e7-9679810944fc"}, exp: []any{[]fill.Data{buy}}},
		{name: "no readable execution", frame: bytes.Replace(unknownPair, []byte(`"side":"Buy"`), []byte(`"side":"Hold"`), 1), errs: []error{currency.ErrPairNotFound, order.ErrSideIsInvalid}, named: []string{"linear ZZZQUSDT execution 3510f361-0add-5c7b-a2e7-9679810944fc", "linear BTCUSDT execution 6e3f9a27-4b0c-5d1e-9f2a-81c7d4e0b5aa"}},
		{name: "an execution that cannot be decoded", frame: bytes.Replace(frames[0], []byte(`"execQty":"0.003"`), []byte(`"execQty":"broken"`), 1), named: []string{"linear BTCUSDT execution 3510f361-0add-5c7b-a2e7-9679810944fc"}, exp: []any{[]fill.Data{buy}}},
		{name: "an undecodable push", frame: []byte(`{"topic":"execution.fast","data":"bad"}`)},
	} {
		err := ex.wsProcessFastExecution(t.Context(), tc.frame)
		require.Errorf(t, err, "wsProcessFastExecution must error for %s", tc.name)
		for _, expErr := range tc.errs {
			require.ErrorIsf(t, err, expErr, "wsProcessFastExecution must report each failed execution for %s", tc.name)
		}
		for _, named := range tc.named {
			assert.ErrorContainsf(t, err, named, "error should identify each failed execution for %s", tc.name)
		}
		assert.Equalf(t, tc.exp, relayed(ex), "wsProcessFastExecution should relay only the executions it reads for %s", tc.name)
	}

	require.NoError(t, ex.wsProcessFastExecution(t.Context(), []byte(`{"topic":"execution.fast","creationTime":1716800399338,"data":[]}`)), "wsProcessFastExecution must not error for a push without executions")
	assert.Empty(t, relayed(ex), "wsProcessFastExecution should not relay an empty batch")

	errBufferFull := fillDataHandler(t, ex)
	require.ErrorIs(t, ex.wsProcessFastExecution(t.Context(), frames[0]), errBufferFull, "wsProcessFastExecution must return the data handler's error")
	err := ex.wsProcessFastExecution(t.Context(), unknownPair)
	require.ErrorIs(t, err, errBufferFull, "wsProcessFastExecution must return the data handler's error for a push with a failed execution")
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "wsProcessFastExecution should still report the failed execution when the relay fails")
}

func TestExecutionFill(t *testing.T) {
	t.Parallel()
	ex := fixtureStreamInstance(t)
	for _, tc := range []struct {
		category string
		symbol   string
		side     string
		asset    asset.Item
		pair     currency.Pair
		expSide  order.Side
		err      error
	}{
		{category: cLinear, symbol: "BTCUSDT", side: "Sell", asset: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(), expSide: order.Sell},
		{category: cLinear, symbol: "BTCPERP", side: "Buy", asset: asset.USDCMarginedFutures, pair: fixtureUSDCPerpetualPair, expSide: order.Buy},
		{category: cInverse, symbol: "BTCUSD", side: "Buy", asset: asset.CoinMarginedFutures, pair: currency.NewBTCUSD(), expSide: order.Buy},
		{category: cSpot, symbol: "BTCUSDT", side: "Buy", asset: asset.Spot, pair: currency.NewBTCUSDT(), expSide: order.Buy},
		{category: cOption, symbol: "BTC-24SEP27-170000-P-USDT", side: "Sell", asset: asset.Options, pair: fixtureOptionPair, expSide: order.Sell},
		{category: "event", symbol: "BTCUSDT-15MIN-DOWN", side: "Sell", err: errUnsupportedCategory},
		{category: cLinear, symbol: "ZZZQUSDT", side: "Sell", err: currency.ErrPairNotFound},
		{category: cLinear, symbol: "BTCUSDT", side: "Hold", err: order.ErrSideIsInvalid},
	} {
		f, err := ex.executionFill(tc.category, tc.symbol, tc.side, "0ab1bdf7-4219-438b-b30a-32ec863018f7", "9aac161b-8ed6-450d-9cab-c5cc67c21784", "linear-close-001")
		require.ErrorIsf(t, err, tc.err, "executionFill must return the expected error for %s %s", tc.category, tc.symbol)
		var exp fill.Data
		if tc.err == nil {
			exp = fill.Data{
				ID:            "0ab1bdf7-4219-438b-b30a-32ec863018f7",
				Exchange:      "TestExecutionFill",
				AssetType:     tc.asset,
				CurrencyPair:  tc.pair,
				Side:          tc.expSide,
				OrderID:       "9aac161b-8ed6-450d-9cab-c5cc67c21784",
				ClientOrderID: "linear-close-001",
			}
		}
		assert.Equalf(t, exp, f, "executionFill should identify the %s %s fill", tc.category, tc.symbol)
	}
}

func TestWebsocketOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		livePrivateSubscription(t, "order")
		return
	}
	ex := streamTestInstance(t)
	frames := fixtureFrames(t, "testdata/wsPrivateOrder.json")
	pushes := decodeFramesStrictly[WebsocketOrderUpdate](t, frames)
	assert.Equal(t, fixtureWebsocketOrders(), pushes, "fixture frames should decode every order field")
	conn := &FixtureConnection{match: websocket.NewMatch()}
	for i, frame := range frames {
		require.NoErrorf(t, ex.wsHandleAuthenticatedData(t.Context(), conn, frame), "wsHandleAuthenticatedData must not error for frame %d", i)
	}
	linear := order.Detail{
		TimeInForce:          order.GoodTillCancel,
		ReduceOnly:           true,
		Price:                82600,
		Amount:               0.5,
		TriggerPrice:         82590,
		AverageExecutedPrice: 82600,
		ExecutedAmount:       0.3,
		RemainingAmount:      0.2,
		Cost:                 24780,
		CostAsset:            currency.USDT,
		Exchange:             "TestWebsocketOrder",
		OrderID:              "c1956690-b731-4191-97c0-94b00422231b",
		ClientOrderID:        "linear-tp-001",
		Type:                 order.TakeProfit,
		Side:                 order.Sell,
		Status:               order.PartiallyFilled,
		AssetType:            asset.USDTMarginedFutures,
		Date:                 time.UnixMilli(1733778525913),
		LastUpdated:          time.UnixMilli(1733778525917),
		Pair:                 currency.NewBTCUSDT(),
		SettlementCurrency:   currency.USDT,
	}
	linear.SetFees(mustFees(t, order.FeeAmount{Currency: currency.USDT, Amount: 13.629}))
	spot := order.Detail{
		TimeInForce:          order.ImmediateOrCancel,
		Price:                82650,
		Amount:               0.000351,
		QuoteAmount:          29,
		AverageExecutedPrice: 82590.5,
		ExecutedAmount:       0.000351,
		Cost:                 28.9892655,
		CostAsset:            currency.USDT,
		Exchange:             "TestWebsocketOrder",
		OrderID:              "2290000000123456789",
		ClientOrderID:        "spot-buy-001",
		Type:                 order.OCO,
		Side:                 order.Buy,
		Status:               order.Filled,
		AssetType:            asset.Spot,
		Date:                 time.UnixMilli(1791535765230),
		LastUpdated:          time.UnixMilli(1791535765238),
		Pair:                 currency.NewBTCUSDT(),
	}
	spot.SetFees(mustFees(t, order.FeeAmount{Currency: currency.BTC, Amount: 0.00000035}, order.FeeAmount{Currency: currency.MNT, Amount: 0.00242968}))
	option := order.Detail{
		TimeInForce:          order.ImmediateOrCancel,
		Price:                84100,
		Amount:               0.01,
		AverageExecutedPrice: 84130,
		ExecutedAmount:       0.01,
		Cost:                 841.3,
		CostAsset:            currency.USDT,
		Exchange:             "TestWebsocketOrder",
		OrderID:              "5cf98598-39a7-459e-97bf-76ca765ee020",
		ClientOrderID:        "option-sell-001",
		Type:                 order.Limit,
		Side:                 order.Sell,
		Status:               order.Filled,
		AssetType:            asset.Options,
		Date:                 time.UnixMilli(1791535798080),
		LastUpdated:          time.UnixMilli(1791535798091),
		Pair:                 currency.NewPairWithDelimiter("BTC", "24SEP27-170000-P-USDT", "-"),
		SettlementCurrency:   currency.USDT,
	}
	// Option orders report their fee in cumExecFee; their cumFeeDetail is empty
	option.SetFees(mustFees(t, order.FeeAmount{Currency: currency.USDT, Amount: 1.25}))
	exp := []any{[]order.Detail{linear}, []order.Detail{spot}, []order.Detail{option}}
	assert.Equal(t, exp, relayed(ex), "wsHandleAuthenticatedData should relay each order as an order detail")
	for _, tc := range []struct {
		data string
		err  error
	}{
		{data: `{"category":"linear","symbol":"ZZZQUSDT","side":"Sell","orderType":"Limit","timeInForce":"GTC"}`, err: currency.ErrPairNotFound},
		{data: `{"category":"linear","symbol":"BTCUSDT","side":"Hold","orderType":"Limit","timeInForce":"GTC"}`, err: order.ErrSideIsInvalid},
		{data: `{"category":"linear","symbol":"BTCUSDT","side":"Sell","orderType":"Iceberg","timeInForce":"GTC"}`, err: order.ErrUnrecognisedOrderType},
		{data: `{"category":"linear","symbol":"BTCUSDT","side":"Sell","orderType":"Limit","timeInForce":"Forever"}`, err: order.ErrInvalidTimeInForce},
	} {
		err := ex.wsHandleAuthenticatedData(t.Context(), conn, []byte(`{"topic":"order","data":[`+tc.data+`]}`))
		require.ErrorIsf(t, err, tc.err, "wsHandleAuthenticatedData must reject %s", tc.data)
	}
}

func TestWebsocketProcessOrder(t *testing.T) {
	t.Parallel()
	frames := fixtureFrames(t, "testdata/wsPrivateOrder.json")
	require.Len(t, frames, 3, "the order fixture must hold a linear, a spot and an option push")
	ex := fixtureStreamInstance(t)
	for i, frame := range frames {
		require.NoErrorf(t, ex.wsProcessOrder(t.Context(), frame), "wsProcessOrder must not error for frame %d", i)
	}
	linear := order.Detail{
		TimeInForce:          order.GoodTillCancel,
		ReduceOnly:           true,
		Price:                82600,
		Amount:               0.5,
		TriggerPrice:         82590,
		AverageExecutedPrice: 82600,
		ExecutedAmount:       0.3,
		RemainingAmount:      0.2,
		Cost:                 24780,
		CostAsset:            currency.USDT,
		Exchange:             "TestWebsocketProcessOrder",
		OrderID:              "c1956690-b731-4191-97c0-94b00422231b",
		ClientOrderID:        "linear-tp-001",
		Type:                 order.TakeProfit,
		Side:                 order.Sell,
		Status:               order.PartiallyFilled,
		AssetType:            asset.USDTMarginedFutures,
		Date:                 time.UnixMilli(1733778525913),
		LastUpdated:          time.UnixMilli(1733778525917),
		Pair:                 currency.NewBTCUSDT(),
		SettlementCurrency:   currency.USDT,
	}
	linear.SetFees(mustFees(t, order.FeeAmount{Currency: currency.USDT, Amount: 13.629}))
	// A spot market buy sized by value holds the quote amount in qty, and the base amount it bought in cumExecQty
	spot := order.Detail{
		TimeInForce:          order.ImmediateOrCancel,
		Price:                82650,
		Amount:               0.000351,
		QuoteAmount:          29,
		AverageExecutedPrice: 82590.5,
		ExecutedAmount:       0.000351,
		Cost:                 28.9892655,
		CostAsset:            currency.USDT,
		Exchange:             "TestWebsocketProcessOrder",
		OrderID:              "2290000000123456789",
		ClientOrderID:        "spot-buy-001",
		Type:                 order.OCO,
		Side:                 order.Buy,
		Status:               order.Filled,
		AssetType:            asset.Spot,
		Date:                 time.UnixMilli(1791535765230),
		LastUpdated:          time.UnixMilli(1791535765238),
		Pair:                 currency.NewBTCUSDT(),
	}
	spot.SetFees(mustFees(t, order.FeeAmount{Currency: currency.BTC, Amount: 0.00000035}, order.FeeAmount{Currency: currency.MNT, Amount: 0.00242968}))
	option := order.Detail{
		TimeInForce:          order.ImmediateOrCancel,
		Price:                84100,
		Amount:               0.01,
		AverageExecutedPrice: 84130,
		ExecutedAmount:       0.01,
		Cost:                 841.3,
		CostAsset:            currency.USDT,
		Exchange:             "TestWebsocketProcessOrder",
		OrderID:              "5cf98598-39a7-459e-97bf-76ca765ee020",
		ClientOrderID:        "option-sell-001",
		Type:                 order.Limit,
		Side:                 order.Sell,
		Status:               order.Filled,
		AssetType:            asset.Options,
		Date:                 time.UnixMilli(1791535798080),
		LastUpdated:          time.UnixMilli(1791535798091),
		Pair:                 fixtureOptionPair,
		SettlementCurrency:   currency.USDT,
	}
	// Option orders report their fee in cumExecFee, in the settle coin; their cumFeeDetail is empty
	option.SetFees(mustFees(t, order.FeeAmount{Currency: currency.USDT, Amount: 1.25}))
	assert.Equal(t, []any{[]order.Detail{linear}, []order.Detail{spot}, []order.Detail{option}}, relayed(ex), "wsProcessOrder should relay each order as an order detail")
	// A push can carry several orders, and every one of them is relayed
	spotOrders := frames[1][bytes.Index(frames[1], []byte(`"data":[`))+len(`"data":[`) : bytes.LastIndexByte(frames[1], ']')]
	twoOrders := bytes.Replace(frames[0], []byte(`"data":[`), bytes.Join([][]byte{[]byte(`"data":[`), spotOrders, []byte(",")}, nil), 1)
	require.NoError(t, ex.wsProcessOrder(t.Context(), twoOrders), "wsProcessOrder must not error for a push of two orders")
	assert.Equal(t, []any{[]order.Detail{spot, linear}}, relayed(ex), "wsProcessOrder should relay every order of a push")
	// The cost is the executed value Bybit reports, which need not be the average price times the executed quantity
	require.NoError(t, ex.wsProcessOrder(t.Context(), bytes.Replace(frames[0], []byte(`"cumExecValue":"24780"`), []byte(`"cumExecValue":"24780.15"`), 1)), "wsProcessOrder must not error for an order whose value is not its average price times its quantity")
	valued := linear
	valued.Cost = 24780.15
	assert.Equal(t, []any{[]order.Detail{valued}}, relayed(ex), "wsProcessOrder should take an order's cost from its executed value")

	// A retail price improvement order rests on the book as a post only order
	require.NoError(t, ex.wsProcessOrder(t.Context(), bytes.Replace(frames[0], []byte(`"timeInForce":"GTC"`), []byte(`"timeInForce":"RPI"`), 1)), "wsProcessOrder must not error for an RPI order")
	rpi := linear
	rpi.TimeInForce = order.PostOnly
	assert.Equal(t, []any{[]order.Detail{rpi}}, relayed(ex), "wsProcessOrder should relay an RPI order as post only")

	// A conditional market order is pushed as the market order it becomes, with its condition in stopOrderType
	stopMarket := bytes.Replace(bytes.Replace(frames[0], []byte(`"orderType":"Limit"`), []byte(`"orderType":"Market"`), 1), []byte(`"stopOrderType":"TakeProfit"`), []byte(`"stopOrderType":"Stop"`), 1)
	require.NoError(t, ex.wsProcessOrder(t.Context(), stopMarket), "wsProcessOrder must not error for a conditional market order")
	conditional := linear
	conditional.Type = order.StopMarket
	assert.Equal(t, []any{[]order.Detail{conditional}}, relayed(ex), "wsProcessOrder should relay a conditional market order as a stop market order")

	// Block trade orders, and orders Bybit names no type for, read back as GetOrderHistory reads them
	for orderType, exp := range map[string]order.Type{"Block trade-Limit": order.Limit, "Block trade Roll Back": order.UnknownType, "UNKNOWN": order.UnknownType} {
		frame := bytes.Replace(bytes.Replace(frames[0], []byte(`"orderType":"Limit"`), []byte(`"orderType":"`+orderType+`"`), 1), []byte(`"stopOrderType":"TakeProfit"`), []byte(`"stopOrderType":""`), 1)
		require.NoErrorf(t, ex.wsProcessOrder(t.Context(), frame), "wsProcessOrder must not error for order type %q", orderType)
		typed := linear
		typed.Type = exp
		assert.Equalf(t, []any{[]order.Detail{typed}}, relayed(ex), "wsProcessOrder should relay order type %q as GetOrderHistory reads it", orderType)
	}

	// Bybit sizes a spot market buy by value unless told otherwise, and reports some such buys with an empty market unit
	require.NoError(t, ex.wsProcessOrder(t.Context(), bytes.Replace(frames[1], []byte(`"marketUnit":"quoteCoin"`), []byte(`"marketUnit":""`), 1)), "wsProcessOrder must not error for a spot market buy without a market unit")
	assert.Equal(t, []any{[]order.Detail{spot}}, relayed(ex), "wsProcessOrder should read a spot market buy without a market unit as sized by value")

	// An order that cannot be read is reported without holding back the push's other orders
	linearOrders := frames[0][bytes.Index(frames[0], []byte(`"data":[`))+len(`"data":[`) : bytes.LastIndexByte(frames[0], ']')]
	unknownPair := `{"category":"linear","symbol":"BTCUSDT-26SEP25","orderId":"0f6c8a52-3e1d-4b7a-9c25-6d8e1f4a7b30","side":"Sell","orderType":"Limit","timeInForce":"GTC","orderStatus":"Cancelled"}`
	unknownSide := `{"category":"spot","symbol":"BTCUSDT","orderId":"2290000000123456790","side":"Hold","orderType":"Limit","timeInForce":"GTC","orderStatus":"New"}`
	unknownType := `{"category":"linear","symbol":"BTCUSDT","orderId":"6b5a4c3d-2e1f-4a0b-9c8d-7e6f5a4b3c21","side":"Buy","orderType":"Bogus","timeInForce":"GTC","orderStatus":"New"}`
	unknownTimeInForce := `{"category":"linear","symbol":"BTCUSDT","orderId":"6b5a4c3d-2e1f-4a0b-9c8d-7e6f5a4b3c22","side":"Buy","orderType":"Limit","timeInForce":"Bogus","orderStatus":"New"}`
	unknownFeeCoin := `{"category":"linear","symbol":"BTCUSDT","orderId":"6b5a4c3d-2e1f-4a0b-9c8d-7e6f5a4b3c23","side":"Buy","orderType":"Limit","timeInForce":"GTC","orderStatus":"New","cumFeeDetail":{"":"0.1"}}`
	undecodable := `{"category":"linear","symbol":"BTCUSDT","orderId":"6b5a4c3d-2e1f-4a0b-9c8d-7e6f5a4b3c24","side":"Buy","orderType":"Limit","timeInForce":"GTC","orderStatus":"New","qty":"broken"}`
	mixed := []byte(`{"topic":"order","data":[` + unknownPair + `,` + string(linearOrders) + `,` + unknownType + `,` + unknownTimeInForce + `,` + unknownFeeCoin + `,` + undecodable + `,` + unknownSide + `,` + string(spotOrders) + `]}`)
	err := ex.wsProcessOrder(t.Context(), mixed)
	require.ErrorIs(t, err, currency.ErrPairNotFound, "wsProcessOrder must report an order whose symbol is not an available pair")
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "wsProcessOrder must report every order it cannot read")
	require.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "wsProcessOrder must report an order of an unknown type")
	require.ErrorIs(t, err, order.ErrInvalidTimeInForce, "wsProcessOrder must report an order of an unknown time in force")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "wsProcessOrder must report an order whose fees name no coin")
	assert.ErrorContains(t, err, "linear BTCUSDT-26SEP25 order 0f6c8a52-3e1d-4b7a-9c25-6d8e1f4a7b30", "error should identify the order of the unknown pair")
	assert.ErrorContains(t, err, "spot BTCUSDT order 2290000000123456790", "error should identify the order of the unknown side")
	for _, id := range []string{"6b5a4c3d-2e1f-4a0b-9c8d-7e6f5a4b3c21", "6b5a4c3d-2e1f-4a0b-9c8d-7e6f5a4b3c22", "6b5a4c3d-2e1f-4a0b-9c8d-7e6f5a4b3c23", "6b5a4c3d-2e1f-4a0b-9c8d-7e6f5a4b3c24"} {
		assert.ErrorContainsf(t, err, "linear BTCUSDT order "+id, "error should identify order %s", id)
	}
	assert.Equal(t, []any{[]order.Detail{linear, spot}}, relayed(ex), "wsProcessOrder should relay the push's other orders in turn")
	require.NoError(t, ex.wsProcessOrder(t.Context(), []byte(`{"topic":"order","data":[]}`)), "wsProcessOrder must not error for a push without orders")
	assert.Empty(t, relayed(ex), "wsProcessOrder should not relay an empty batch")

	for _, tc := range []struct {
		name  string
		frame []byte
		err   error
	}{
		{name: "an unknown pair", frame: bytes.Replace(frames[0], []byte(`"symbol":"BTCUSDT"`), []byte(`"symbol":"ZZZQUSDT"`), 1), err: currency.ErrPairNotFound},
		{name: "an unknown side", frame: bytes.Replace(frames[0], []byte(`"side":"Sell"`), []byte(`"side":"Hold"`), 1), err: order.ErrSideIsInvalid},
		{name: "an unknown order type", frame: bytes.Replace(frames[0], []byte(`"orderType":"Limit"`), []byte(`"orderType":"Iceberg"`), 1), err: order.ErrUnrecognisedOrderType},
		{name: "an unknown time in force", frame: bytes.Replace(frames[0], []byte(`"timeInForce":"GTC"`), []byte(`"timeInForce":"Forever"`), 1), err: order.ErrInvalidTimeInForce},
		{name: "a fee without a coin", frame: bytes.Replace(frames[0], []byte(`"cumFeeDetail":{"USDT":"13.629"}`), []byte(`"cumFeeDetail":{"":"13.629"}`), 1), err: currency.ErrCurrencyCodeEmpty},
		{name: "an undecodable push", frame: []byte(`{"topic":"order","data":"bad"}`)},
	} {
		err := ex.wsProcessOrder(t.Context(), tc.frame)
		if tc.err == nil {
			require.Errorf(t, err, "wsProcessOrder must error for %s", tc.name)
		} else {
			require.ErrorIsf(t, err, tc.err, "wsProcessOrder must error for %s", tc.name)
		}
		assert.Emptyf(t, relayed(ex), "wsProcessOrder should relay nothing for %s", tc.name)
	}

	errBufferFull := fillDataHandler(t, ex)
	require.ErrorIs(t, ex.wsProcessOrder(t.Context(), frames[0]), errBufferFull, "wsProcessOrder must return the data handler's error")
	err = ex.wsProcessOrder(t.Context(), mixed)
	require.ErrorIs(t, err, errBufferFull, "wsProcessOrder must return the data handler's error for a push with a failed order")
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "wsProcessOrder should still report the failed order when the relay fails")
}

// mustFees returns the fee snapshot of amounts
func mustFees(tb testing.TB, amounts ...order.FeeAmount) order.Fees {
	tb.Helper()
	fees, err := order.NewFees(amounts...)
	require.NoError(tb, err, "NewFees must not error")
	return fees
}

// fixtureWebsocketOrders returns the order pushes testdata/wsPrivateOrder.json holds
func fixtureWebsocketOrders() []WebsocketOrderUpdate {
	return []WebsocketOrderUpdate{
		{
			ID:           "5923240c6880ab-c59f-420b-9adb-3639adc9dd90",
			Topic:        "order",
			CreationTime: types.Time(time.UnixMilli(1672364262474)),
			Data: []WebsocketOrder{
				{
					Category:                   "linear",
					OrderID:                    "c1956690-b731-4191-97c0-94b00422231b",
					OrderLinkID:                "linear-tp-001",
					ParentOrderLinkID:          "linear-entry-001",
					BlockTradeID:               "block-c195",
					Symbol:                     "BTCUSDT",
					Price:                      82600,
					BrokerOrderPrice:           82600.5,
					Quantity:                   0.5,
					Side:                       "Sell",
					PositionIndex:              2,
					OrderStatus:                "PartiallyFilled",
					CreateType:                 "CreateByUser",
					CancelType:                 "UNKNOWN",
					RejectReason:               "EC_NoError",
					AveragePrice:               82600,
					LeavesQuantity:             0.2,
					LeavesValue:                16520,
					CumulativeExecutedQuantity: 0.3,
					CumulativeExecutedValue:    24780,
					CumulativeExecutedFee:      13.629,
					ClosedPNL:                  4.5,
					TimeInForce:                "GTC",
					OrderType:                  "Limit",
					StopOrderType:              "TakeProfit",
					OCOTriggerBy:               "OcoTriggerByUnknown",
					SlippageToleranceType:      "UNKNOWN",
					TriggerPrice:               82590,
					TakeProfit:                 83000,
					StopLoss:                   82000,
					TPSLMode:                   "Partial",
					TPLimitPrice:               83010,
					SLLimitPrice:               81990,
					TPTriggerBy:                "MarkPrice",
					SLTriggerBy:                "IndexPrice",
					TriggerDirection:           1,
					TriggerBy:                  "LastPrice",
					LastPriceOnCreated:         82561,
					ReduceOnly:                 true,
					CloseOnTrigger:             true,
					SMPType:                    "CancelMaker",
					SMPGroup:                   12,
					SMPOrderID:                 "smp-9f1c",
					CreatedTime:                types.Time(time.UnixMilli(1733778525913)),
					UpdatedTime:                types.Time(time.UnixMilli(1733778525917)),
					CumulativeFeeDetail: map[currency.Code]types.Number{
						currency.USDT: 13.629,
					},
					RPITakerAccess:     true,
					RPIMatchedQuantity: 0.1,
				},
			},
		},
		{
			ID:           "5923240c6880ab-c59f-420b-9adb-3639adc9dd91",
			Topic:        "order.spot",
			CreationTime: types.Time(time.UnixMilli(1672364262480)),
			Data: []WebsocketOrder{
				{
					Category:                   "spot",
					OrderID:                    "2290000000123456789",
					OrderLinkID:                "spot-buy-001",
					IsLeverage:                 "1",
					Symbol:                     "BTCUSDT",
					Price:                      82650,
					Quantity:                   29,
					Side:                       "Buy",
					OrderStatus:                "Filled",
					CancelType:                 "UNKNOWN",
					RejectReason:               "EC_NoError",
					AveragePrice:               82590.5,
					CumulativeExecutedQuantity: 0.000351,
					CumulativeExecutedValue:    28.9892655,
					FeeCurrency:                currency.BTC,
					TimeInForce:                "IOC",
					OrderType:                  "Market",
					StopOrderType:              "OcoOrder",
					OCOTriggerBy:               "OcoTriggerByTp",
					MarketUnit:                 "quoteCoin",
					SlippageToleranceType:      "Percent",
					SlippageTolerance:          0.5,
					LastPriceOnCreated:         82590.6,
					SMPType:                    "None",
					CreatedTime:                types.Time(time.UnixMilli(1791535765230)),
					UpdatedTime:                types.Time(time.UnixMilli(1791535765238)),
					CumulativeFeeDetail: map[currency.Code]types.Number{
						currency.BTC:            0.00000035,
						currency.NewCode("MNT"): 0.00242968,
					},
				},
			},
		},
		{
			ID:           "5923240c6880ab-c59f-420b-9adb-3639adc9dd92",
			Topic:        "order.option",
			CreationTime: types.Time(time.UnixMilli(1672364262490)),
			Data: []WebsocketOrder{
				{
					Category:                   "option",
					OrderID:                    "5cf98598-39a7-459e-97bf-76ca765ee020",
					OrderLinkID:                "option-sell-001",
					Symbol:                     "BTC-24SEP27-170000-P-USDT",
					Price:                      84100,
					Quantity:                   0.01,
					Side:                       "Sell",
					OrderStatus:                "Filled",
					CancelType:                 "UNKNOWN",
					RejectReason:               "EC_NoError",
					AveragePrice:               84130,
					CumulativeExecutedQuantity: 0.01,
					CumulativeExecutedValue:    841.3,
					CumulativeExecutedFee:      1.25,
					TimeInForce:                "IOC",
					OrderType:                  "Limit",
					OrderImpliedVolatility:     0.4249,
					SlippageToleranceType:      "UNKNOWN",
					LastPriceOnCreated:         84120,
					PlaceType:                  "iv",
					SMPType:                    "None",
					CreatedTime:                types.Time(time.UnixMilli(1791535798080)),
					UpdatedTime:                types.Time(time.UnixMilli(1791535798091)),
					CumulativeFeeDetail:        map[currency.Code]types.Number{},
				},
			},
		},
	}
}

func TestWebsocketEntries(t *testing.T) {
	t.Parallel()
	_, err := websocketEntries[WebsocketOrder]([]byte(`{"topic":"order","data":"bad"}`), "order", "orderId")
	require.Error(t, err, "websocketEntries must error for a push whose data is not a list")
	for _, creationTime := range []string{`"bad"`, `42`} {
		_, err = websocketEntries[WebsocketOrder]([]byte(`{"topic":"order","creationTime":`+creationTime+`,"data":[]}`), "order", "orderId")
		require.Errorf(t, err, "websocketEntries must error for a push whose creation time %s is not a time", creationTime)
	}
	entries, err := websocketEntries[WebsocketOrder]([]byte(`{"topic":"order","data":[]}`), "order", "orderId")
	require.NoError(t, err, "websocketEntries must not error for a push without entries")
	assert.Empty(t, entries, "websocketEntries should return no entries for a push without any")

	entries, err = websocketEntries[WebsocketOrder]([]byte(`{"topic":"order","data":[{"category":"linear","symbol":"BTCUSDT","orderId":"a","qty":"1"},{"category":"linear","symbol":"ETHUSDT","orderId":"b","qty":"broken"},{"category":"spot","symbol":"SOLUSDT","orderId":"c","qty":"2"}]}`), "order", "orderId")
	require.Error(t, err, "websocketEntries must report an entry that cannot be decoded")
	assert.ErrorContains(t, err, "linear ETHUSDT order b", "error should identify the entry that cannot be decoded")
	exp := []WebsocketOrder{{Category: "linear", Symbol: "BTCUSDT", OrderID: "a", Quantity: 1}, {Category: "spot", Symbol: "SOLUSDT", OrderID: "c", Quantity: 2}}
	assert.Equal(t, exp, entries, "websocketEntries should decode the other entries in turn")
}

func TestTimeInForceFromString(t *testing.T) {
	t.Parallel()
	for in, exp := range map[string]order.TimeInForce{"RPI": order.PostOnly, "PostOnly": order.PostOnly, "GTC": order.GoodTillCancel, "IOC": order.ImmediateOrCancel, "FOK": order.FillOrKill} {
		tif, err := timeInForceFromString(in)
		require.NoErrorf(t, err, "timeInForceFromString must not error for %s", in)
		assert.Equalf(t, exp, tif, "timeInForceFromString should map %s", in)
	}
	_, err := timeInForceFromString("Forever")
	require.ErrorIs(t, err, order.ErrInvalidTimeInForce, "timeInForceFromString must reject an unknown time in force")
}

func TestWebsocketWallet(t *testing.T) {
	t.Parallel()
	if !mockTests {
		livePrivateSubscription(t, "wallet")
		return
	}
	ex := streamTestInstance(t)
	frames := fixtureFrames(t, "testdata/wsPrivateWallet.json")
	pushes := decodeFramesStrictly[WebsocketWalletUpdate](t, frames)
	expPushes := []WebsocketWalletUpdate{
		{
			ID:           "592324d2bce751-ad38-48eb-8f42-4671d1fb4d4e",
			Topic:        "wallet",
			CreationTime: types.Time(time.UnixMilli(1700034722104)),
			Data: []WebsocketWallet{
				{
					AccountType:                             "UNIFIED",
					AccountInitialMarginRate:                0.0125,
					AccountMaintenanceMarginRate:            0.0068,
					TotalEquity:                             10262.91335023,
					TotalWalletBalance:                      9684.46297164,
					TotalMarginBalance:                      9706.87297164,
					TotalAvailableBalance:                   8463.85257164,
					TotalPerpetualUnrealisedPNL:             22.41,
					TotalInitialMargin:                      1243.0204,
					TotalMaintenanceMargin:                  68.71,
					AccountInitialMarginRateByMarkPrice:     0.0125,
					AccountMaintenanceMarginRateByMarkPrice: 0.0068,
					TotalInitialMarginByMarkPrice:           1243.0204,
					TotalMaintenanceMarginByMarkPrice:       68.71,
					AccountLTV:                              0.017,
					Coins: []WalletBalanceCoin{
						{
							Coin:                           currency.BTC,
							Equity:                         0.00102964,
							USDValue:                       36.70759517,
							WalletBalance:                  0.00102964,
							Locked:                         0.0002,
							SpotHedgingQuantity:            0.01592413,
							BorrowAmount:                   0.0001,
							AccruedInterest:                0.00000012,
							TotalOrderInitialMargin:        0.00001,
							TotalPositionInitialMargin:     0.00002,
							TotalPositionMaintenanceMargin: 0.000005,
							UnrealisedPNL:                  0.0000271,
							CumulativeRealisedPNL:          -0.00000973,
							Bonus:                          0.0001,
							CollateralSwitch:               true,
							MarginCollateral:               true,
							CollateralRestriction:          "1",
							SpotBorrow:                     0.0001,
							Free:                           0.00082964,
							AvailableToBorrow:              1.5,
							AvailableToWithdraw:            0.00102964,
						},
						{
							Coin:                  currency.USDT,
							Equity:                10226.20575506,
							USDValue:              10226.2057,
							WalletBalance:         9648,
							UnrealisedPNL:         22.41,
							CumulativeRealisedPNL: -25.06579337,
							MarginCollateral:      true,
							CollateralRestriction: "0",
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expPushes, pushes, "fixture frames should decode every wallet field")
	require.NoError(t, ex.wsHandleAuthenticatedData(t.Context(), nil, frames[0]), "wsHandleAuthenticatedData must not error")
	data := relayed(ex)
	require.Len(t, data, 1, "the wallet must be relayed")
	subAccounts, ok := data[0].(accounts.SubAccounts)
	require.True(t, ok, "the wallet must be relayed as sub accounts")
	for _, sa := range subAccounts {
		for code, balance := range sa.Balances {
			assert.False(t, balance.UpdatedAt.IsZero(), "balance arrival time should be set")
			balance.UpdatedAt = time.Time{}
			sa.Balances[code] = balance
		}
	}
	// The BTC wallet balance includes its spot borrowing, and its orders and positions hold margin
	btcWallet, btcLocked, btcOrderMargin, btcPositionMargin, btcSpotBorrow, btcBonus := 0.00102964, 0.0002, 0.00001, 0.00002, 0.0001, 0.0001
	btcHold := btcLocked + btcOrderMargin + btcPositionMargin
	exp := accounts.SubAccounts{{
		AssetType: asset.Spot,
		Balances: accounts.CurrencyBalances{
			currency.BTC:  {Currency: currency.BTC, Total: btcWallet, Hold: btcHold, Free: btcWallet - btcHold, Borrowed: 0.0001, AvailableWithoutBorrow: btcWallet - btcSpotBorrow - btcHold - btcBonus},
			currency.USDT: {Currency: currency.USDT, Total: 9648, Free: 9648, AvailableWithoutBorrow: 9648},
		},
	}}
	assert.Equal(t, exp, subAccounts, "wsHandleAuthenticatedData should store and relay each coin's wallet balance")

	ex.Accounts = nil
	err := ex.wsHandleAuthenticatedData(t.Context(), nil, frames[0])
	require.ErrorIs(t, err, common.ErrNilPointer, "wsHandleAuthenticatedData must error without account storage")
}

func TestWebsocketWalletReplacesRESTBalances(t *testing.T) {
	t.Parallel()
	payload := []byte(`{"id":"wallet-update","topic":"wallet","creationTime":1787197420650,"data":[{"accountIMRate":"0.01","accountMMRate":"0.005","totalEquity":"120","totalWalletBalance":"100","totalMarginBalance":"110","totalAvailableBalance":"75","totalPerpUPL":"10","totalInitialMargin":"25","totalMaintenanceMargin":"5","coin":[{"coin":"USDT","equity":"120","usdValue":"120","walletBalance":"100","locked":"15","spotBorrow":"20","availableToWithdraw":"","availableToBorrow":"","borrowAmount":"20","accruedInterest":"0","totalOrderIM":"5","totalPositionIM":"20","totalPositionMM":"5","unrealisedPnl":"10","cumRealisedPnl":"2","bonus":"0","spotHedgingQty":"0"}],"accountType":"UNIFIED","accountLTV":""}]}`)
	pushes := decodeFramesStrictly[WebsocketWalletUpdate](t, [][]byte{payload})
	require.Len(t, pushes[0].Data, 1, "the wallet payload must hold one account")
	require.Len(t, pushes[0].Data[0].Coins, 1, "the wallet payload must hold one coin")
	assert.Zero(t, pushes[0].Data[0].Coins[0].AvailableToWithdraw, "the deprecated empty availableToWithdraw should decode as zero")
	ex := withMockCredentials(websocketTestInstance(t))
	seed := accounts.NewSubAccount(asset.Spot, "")
	seed.Balances.Set(currency.USDT, accounts.Balance{Total: 90, Hold: 25, Free: 65, Borrowed: 10})
	require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{seed}, true), "Accounts.Save must seed a REST snapshot")
	startedAt := time.Now()
	require.NoError(t, ex.wsProcessWalletPushData(t.Context(), payload), "wsProcessWalletPushData must process current UNIFIED wallet payload")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "wsProcessWalletPushData must relay the wallet")
	message := <-ex.Websocket.DataHandler.C
	subAccounts, ok := message.Data.(accounts.SubAccounts)
	require.True(t, ok, "wallet handler must emit canonical subaccounts")
	require.Len(t, subAccounts, 1, "wallet handler must emit one subaccount")
	balance, ok := subAccounts[0].Balances[currency.USDT]
	require.True(t, ok, "wallet handler must emit the USDT balance")
	assert.False(t, balance.UpdatedAt.Before(startedAt), "wallet update should use local arrival order")
	assert.False(t, balance.UpdatedAt.After(time.Now()), "wallet update should be stamped with its arrival time")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	stored, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.USDT)
	require.NoError(t, err, "GetBalance must return the stored USDT balance")
	assert.Equal(t, balance, stored, "wallet update should store the balance it emits")
	balance.UpdatedAt = time.Time{}
	exp := accounts.Balance{Currency: currency.USDT, Total: 100, Hold: 40, Free: 60, Borrowed: 20, AvailableWithoutBorrow: 40}
	assert.Equal(t, exp, balance, "wallet update should replace the REST snapshot with the push's balances, as REST computes them")

	// A coin pushed with nothing left replaces the stored balance with nothing
	emptied := []byte(`{"id":"wallet-update-2","topic":"wallet","creationTime":1787197421650,"data":[{"accountType":"UNIFIED","coin":[{"coin":"USDT","walletBalance":"0","locked":"0","spotBorrow":"0","borrowAmount":"0","totalOrderIM":"0","totalPositionIM":"0","bonus":"0"}]}]}`)
	require.NoError(t, ex.wsProcessWalletPushData(t.Context(), emptied), "wsProcessWalletPushData must process an emptied wallet")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "wsProcessWalletPushData must relay the emptied wallet")
	<-ex.Websocket.DataHandler.C
	stored, err = ex.Accounts.GetBalance("", creds, asset.Spot, currency.USDT)
	require.NoError(t, err, "GetBalance must return the stored USDT balance")
	stored.UpdatedAt = time.Time{}
	assert.Equal(t, accounts.Balance{Currency: currency.USDT}, stored, "an emptied wallet push should replace the stored balance")
}

func TestWebsocketGreeks(t *testing.T) {
	t.Parallel()
	if !mockTests {
		livePrivateSubscription(t, "greeks")
		return
	}
	testRelayedPrivateStream[WebsocketGreeksUpdate](t, "testdata/wsPrivateGreeks.json", []WebsocketGreeksUpdate{
		{
			ID:           "592324fa945a30-2603-49a5-b865-21668c29f2a6",
			Topic:        "greeks",
			CreationTime: types.Time(time.UnixMilli(1672364262482)),
			Data: []WebsocketGreeks{
				{
					BaseCoin:   currency.ETH,
					TotalDelta: 0.06999986,
					TotalGamma: -0.00000001,
					TotalVega:  -0.00000024,
					TotalTheta: 0.00001314,
				},
			},
		},
	})
}

func TestWebsocketDCP(t *testing.T) {
	t.Parallel()
	if !mockTests {
		livePrivateSubscription(t, "dcp.future")
		return
	}
	ex := websocketTestInstance(t)
	conn := &FixtureConnection{}
	require.NoError(t, ex.authSubscribe(t.Context(), conn, subscription.List{{Channel: "dcp.future", Asset: asset.All}, {Channel: "dcp.spot", Asset: asset.All}}), "authSubscribe must not error")
	require.Len(t, conn.sent, 1, "authSubscribe must send one request")
	req, ok := conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "authSubscribe must send a SubscriptionRequest")
	assert.Equal(t, []string{"dcp.future", "dcp.spot"}, req.Arguments, "the dcp topics should be sent as named")
}

func TestWebsocketStrategy(t *testing.T) {
	t.Parallel()
	if !mockTests {
		livePrivateSubscription(t, "strategy")
		return
	}
	testRelayedPrivateStream[WebsocketStrategyUpdate](t, "testdata/wsPrivateStrategy.json", []WebsocketStrategyUpdate{
		{
			ID:           "62f79ebea4794f767cad0bd937f7ad01",
			Topic:        "strategy",
			CreationTime: types.Time(time.UnixMilli(1776734985598)),
			Data: []WebsocketStrategy{
				{
					StrategyID:           "cf7303ae-29c0-480a-8f3d-eaa9330054bc",
					StrategyType:         "iceberg",
					Category:             "UTA_USDT",
					Symbol:               "BTCUSDT",
					Size:                 0.36,
					Side:                 "Buy",
					Status:               3,
					TerminateType:        2,
					TerminateRemark:      "RunningStop",
					ExecutedDuration:     268,
					ExecutedSize:         0.36,
					ExecutedAveragePrice: 134301.53,
					ExecutedStartTime:    types.Time(time.UnixMilli(1776734716717)),
					ExecutedEndTime:      types.Time(time.UnixMilli(1776734985592)),
					CreatedTime:          types.Time(time.UnixMilli(1776734716717)),
					UpdatedTime:          types.Time(time.UnixMilli(1776734985592)),
					ReduceOnly:           true,
					LimitPrice:           134500,
					TriggerCount:         4,
					TradingCount:         12,
					ChaseDistance:        0.5,
					ChasePercentE4:       100,
					MaxChasePrice:        198000,
					ChaseOrderPrice:      135682.4,
					StrategyPrefer:       "quickExecution",
					PostOnly:             true,
					TriggerPrice:         134000,
					IsTriggered:          true,
					StrategyTP:           140000,
					StrategySL:           130000,
					OrderType:            "2",
					OrderPriceOffset:     0.001,
					PositionValue:        48348.55,
					FilledPositionValue:  48348.55,
				},
				{
					StrategyID:           "0b1f3c5e-7d9a-4e2b-8c6d-1f3a5b7c9d0e",
					StrategyType:         "twap",
					Category:             "UTA_SPOT",
					Symbol:               "BTCUSDT",
					Size:                 1.2,
					Side:                 "Sell",
					Duration:             3600,
					Status:               2,
					ExecutedDuration:     600,
					ExecutedSize:         0.2,
					ExecutedAveragePrice: 82560.4,
					ExecutedStartTime:    types.Time(time.Unix(1791535165, 0)),
					CreatedTime:          types.Time(time.Unix(1791535160, 0)),
					UpdatedTime:          types.Time(time.Unix(1791535765, 0)),
					IsRandom:             true,
					TradingCount:         10,
					StrategyPrefer:       "limit",
					Interval:             30,
					LeverageType:         1,
					OrderType:            "1",
				},
			},
		},
	})
}

// testRelayedPrivateStream feeds a fixture's frames to the private handler, checks they decode to exp and that each
// push is relayed decoded
func testRelayedPrivateStream[T any](t *testing.T, fixture string, exp []T) {
	t.Helper()
	ex := websocketTestInstance(t)
	frames := fixtureFrames(t, fixture)
	pushes := decodeFramesStrictly[T](t, frames)
	assert.Equal(t, exp, pushes, "fixture frames should decode every field")
	for i, frame := range frames {
		require.NoErrorf(t, ex.wsHandleAuthenticatedData(t.Context(), nil, frame), "wsHandleAuthenticatedData must not error for frame %d", i)
	}
	data := relayed(ex)
	require.Len(t, data, len(frames), "every push must be relayed")
	for i := range pushes {
		assert.Equalf(t, &pushes[i], data[i], "push %d should be relayed decoded", i)
	}
}

func TestGenerateAuthSubscriptions(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	subs, err := ex.generateAuthSubscriptions()
	require.NoError(t, err, "generateAuthSubscriptions must not error")
	assert.Empty(t, subs, "nothing should be generated without authenticated websocket support")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err = ex.generateAuthSubscriptions()
	require.NoError(t, err, "generateAuthSubscriptions must not error")
	exp := subscription.List{
		{Channel: chanPositions, Asset: asset.All},
		{Channel: chanExecution, Asset: asset.All},
		{Channel: chanOrder, Asset: asset.All},
		{Channel: chanWallet, Asset: asset.All},
	}
	assert.Equal(t, exp, subs, "generateAuthSubscriptions should leave dcp, fast execution, greeks and strategy to callers")

	// Setup disables a configured authenticated subscription, so generating, which can run while the configuration is
	// saved, leaves the configuration as it is
	ex.Config.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: chanWallet, Authenticated: true},
		{Enabled: true, Channel: chanOrder, Authenticated: true},
		{Enabled: true, Channel: chanPositions, Authenticated: true},
		{Enabled: true, Channel: chanExecution, Authenticated: true},
	}
	subs, err = ex.generateAuthSubscriptions()
	require.NoError(t, err, "generateAuthSubscriptions must not error")
	assert.Equal(t, exp, subs, "generateAuthSubscriptions should generate the private topics whatever the configuration holds")
	for _, s := range ex.Config.Features.Subscriptions {
		assert.Truef(t, s.Enabled, "generateAuthSubscriptions should leave the configured %s subscription enabled", s.Channel)
	}
}

func TestQualifyByChannelName(t *testing.T) {
	t.Parallel()
	subs := subscription.List{
		{Channel: subscription.MyOrdersChannel},
		{Channel: "execution.fast.linear"},
		{Channel: chanWallet, QualifiedChannel: "wallet"},
		{Channel: subscription.MyOrdersChannel, QualifiedChannel: "order.spot"},
	}
	qualified := qualifyByChannelName(subs)
	for i, exp := range []string{"order", "execution.fast.linear", "wallet", "order.spot"} {
		assert.Equalf(t, exp, qualified[i].QualifiedChannel, "qualifyByChannelName should keep topic %d if named and otherwise name it by its channel", i)
	}
}
