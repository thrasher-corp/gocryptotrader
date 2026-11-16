package gateio

import (
	"testing"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func TestWebsocketSubmitOrderSpotRequestQuantities(t *testing.T) {
	t.Parallel()
	ex := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, msg []byte, c *gws.Conn) error {
		var req WebsocketRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			return err
		}
		if req.Channel != "spot.order_place" {
			return nil
		}
		return c.WriteMessage(gws.TextMessage, []byte(`{"request_id":"`+req.Payload.RequestID+`","header":{"status":"200","channel":"spot.order_place","event":"api"},"data":{"result":{"id":"1","text":"t-1","create_time_ms":"1605175506123","update_time_ms":"1605175506123","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amount":"0.002","price":"60000","time_in_force":"gtc","left":"0.002","finish_as":"open"}}}`))
	}))
	got, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{
		Exchange:    ex.Name,
		Pair:        currency.NewBTCUSDT(),
		AssetType:   asset.Spot,
		Side:        order.Buy,
		Type:        order.Limit,
		Price:       60000,
		Amount:      0.002,
		QuoteAmount: 120,
	})
	require.NoError(t, err, "WebsocketSubmitOrder must not error")
	assert.Equal(t, 0.002, got.Amount, "Amount should be the submitted base quantity")
	assert.Equal(t, 120.0, got.QuoteAmount, "QuoteAmount should be the submitted quote quantity, as SubmitOrder reports it")
}
