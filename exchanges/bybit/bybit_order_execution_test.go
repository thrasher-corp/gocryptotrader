package bybit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestDeriveWebsocketSubmitResponse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                string
		assetType           asset.Item
		amount              float64
		details             string
		executedAmount      float64
		executedQuoteAmount float64
		fee                 float64
		feeAsset            currency.Code
	}{
		{
			name:                "spot",
			assetType:           asset.Spot,
			amount:              1,
			details:             `{"orderId":"1","orderStatus":"PartiallyFilledCanceled","timeInForce":"IOC","cumExecQty":"0.5","cumExecValue":"30000.07","avgPrice":"60000.1","cumExecFee":"9","cumFeeDetail":{"BTC":"0.0005"}}`,
			executedAmount:      0.5,
			executedQuoteAmount: 30000.07,
			fee:                 0.0005,
			feeAsset:            currency.BTC,
		},
		{
			name:                "linear",
			assetType:           asset.USDTMarginedFutures,
			amount:              3,
			details:             `{"orderId":"1","orderStatus":"Cancelled","timeInForce":"IOC","cumExecQty":"2","cumExecValue":"120000.3","avgPrice":"60000.1","cumExecFee":"9","cumFeeDetail":{"USDT":"0.066"}}`,
			executedAmount:      2,
			executedQuoteAmount: 120000.3,
			fee:                 0.066,
			feeAsset:            currency.USDT,
		},
		{
			name:           "inverse",
			assetType:      asset.CoinMarginedFutures,
			amount:         400,
			details:        `{"orderId":"1","orderStatus":"Cancelled","timeInForce":"IOC","cumExecQty":"300","cumExecValue":"0.005","avgPrice":"60000.1","cumExecFee":"0.000003"}`,
			executedAmount: 300,
			fee:            0.000003,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var details WebsocketOrderDetails
			require.NoError(t, json.Unmarshal([]byte(tc.details), &details), "order details must unmarshal")
			got, err := deriveWebsocketSubmitResponse(&order.Submit{AssetType: tc.assetType, Side: order.Buy, Type: order.Limit, Price: 61000, Amount: tc.amount}, &details)
			require.NoError(t, err, "deriveWebsocketSubmitResponse must not error")
			assert.Equal(t, tc.executedAmount, got.ExecutedAmount, "ExecutedAmount should be the cumulative executed quantity")
			assert.Equal(t, 60000.1, got.AverageExecutedPrice, "AverageExecutedPrice should be the reported average")
			assert.Equal(t, tc.executedQuoteAmount, got.ExecutedQuoteAmount, "ExecutedQuoteAmount should hold only a quote-denominated value")
			assert.Equal(t, tc.fee, got.Fee, "Fee should be the reported cumulative fee")
			assert.Equal(t, tc.feeAsset, got.FeeAsset, "FeeAsset should be the reported fee currency")
		})
	}
}

func TestConstructOrderDetailsFee(t *testing.T) {
	t.Parallel()
	var tradeOrders []TradeOrder
	require.NoError(t, json.Unmarshal([]byte(`[{"orderId":"1","symbol":"ETHUSDT","side":"Buy","orderType":"Limit","orderStatus":"PartiallyFilled","timeInForce":"GTC","price":"1600","qty":"0.1","leavesQty":"0.09","cumExecQty":"0.01","cumExecValue":"16","avgPrice":"1600","cumExecFee":"0.5","cumFeeDetail":{"ETH":"0.00001"}}]`), &tradeOrders), "trade orders must unmarshal")
	orders, err := e.ConstructOrderDetails(tradeOrders, asset.Spot, currency.EMPTYPAIR, currency.Pairs{})
	require.NoError(t, err, "ConstructOrderDetails must not error")
	require.Len(t, orders, 1, "ConstructOrderDetails must return one order")
	assert.Equal(t, 0.00001, orders[0].Fee, "Fee should come from cumFeeDetail")
	assert.Equal(t, currency.ETH, orders[0].FeeAsset, "FeeAsset should come from cumFeeDetail")
}

func TestGetOrderHistoryDerivativeExecutionMappings(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		assetType           asset.Item
		pair                currency.Pair
		row                 string
		executedQuoteAmount float64
		fee                 float64
		feeAsset            currency.Code
	}{
		{
			assetType:           asset.USDTMarginedFutures,
			pair:                currency.NewBTCUSDT(),
			row:                 `{"orderId":"1","symbol":"BTCUSDT","side":"Buy","orderType":"Limit","orderStatus":"Filled","price":"61000","qty":"2","leavesQty":"0","cumExecQty":"2","cumExecValue":"120000.3","avgPrice":"60000.1","cumExecFee":"9","cumFeeDetail":{"USDT":"0.066"},"createdTime":"1735720637000","updatedTime":"1735720638000"}`,
			executedQuoteAmount: 120000.3,
			fee:                 0.066,
			feeAsset:            currency.USDT,
		},
		{
			assetType:           asset.USDCMarginedFutures,
			pair:                currency.NewPair(currency.BTC, currency.USDC),
			row:                 `{"orderId":"1","symbol":"BTCUSDC","side":"Buy","orderType":"Limit","orderStatus":"Filled","price":"61000","qty":"2","leavesQty":"0","cumExecQty":"2","cumExecValue":"120000.3","avgPrice":"60000.1","cumExecFee":"9","cumFeeDetail":{"USDC":"0.066"},"createdTime":"1735720637000","updatedTime":"1735720638000"}`,
			executedQuoteAmount: 120000.3,
			fee:                 0.066,
			feeAsset:            currency.USDC,
		},
		{
			assetType: asset.CoinMarginedFutures,
			pair:      currency.NewBTCUSD(),
			row:       `{"orderId":"1","symbol":"BTCUSD","side":"Buy","orderType":"Limit","orderStatus":"Filled","price":"61000","qty":"300","leavesQty":"0","cumExecQty":"300","cumExecValue":"0.005","avgPrice":"60000.1","cumExecFee":"0.000003","createdTime":"1735720637000","updatedTime":"1735720638000"}`,
			fee:       0.000003,
		},
	} {
		t.Run(tc.assetType.String(), func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
			require.NoError(t, ex.CurrencyPairs.StorePairs(tc.assetType, currency.Pairs{tc.pair}, false), "StorePairs must not error for available pairs")
			require.NoError(t, ex.CurrencyPairs.StorePairs(tc.assetType, currency.Pairs{tc.pair}, true), "StorePairs must not error for enabled pairs")
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, err := w.Write([]byte(`{"retCode":0,"retMsg":"OK","result":{"list":[` + tc.row + `],"nextPageCursor":""},"time":1735720638000}`))
				assert.NoError(t, err, "mock order history response should be written")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})

			orders, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: tc.assetType, Side: order.AnySide, Type: order.AnyType})
			require.NoError(t, err, "GetOrderHistory must not error")
			require.Len(t, orders, 1, "GetOrderHistory must return one order")
			assert.Equal(t, tc.executedQuoteAmount, orders[0].ExecutedQuoteAmount, "history should hold only a quote-denominated executed value")
			assert.Equal(t, tc.fee, orders[0].Fee, "history should retain the cumulative fee")
			assert.Equal(t, tc.feeAsset, orders[0].FeeAsset, "history should retain the fee currency")
		})
	}
}

func TestWSProcessOrderExecutionMappings(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
	for a, p := range map[asset.Item]currency.Pair{asset.USDTMarginedFutures: currency.NewBTCUSDT(), asset.CoinMarginedFutures: currency.NewBTCUSD()} {
		require.NoError(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, false), "StorePairs must not error for available pairs")
		require.NoError(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, true), "StorePairs must not error for enabled pairs")
	}
	require.NoError(t, ex.wsProcessOrder(t.Context(), &WebsocketResponse{Data: json.RawMessage(`[` +
		`{"category":"linear","symbol":"BTCUSDT","orderId":"1","side":"Buy","orderType":"Limit","timeInForce":"GTC","orderStatus":"Filled","price":"61000","qty":"2","cumExecQty":"2","cumExecValue":"120000.3","avgPrice":"60000.1","cumExecFee":"9","cumFeeDetail":{"USDT":"0.066"}},` +
		`{"category":"inverse","symbol":"BTCUSD","orderId":"2","side":"Buy","orderType":"Limit","timeInForce":"GTC","orderStatus":"Filled","price":"61000","qty":"300","cumExecQty":"300","cumExecValue":"0.005","avgPrice":"60000.1","cumExecFee":"0.000003"}]`)}), "wsProcessOrder must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "wsProcessOrder must send one update")
	details, ok := (<-ex.Websocket.DataHandler.C).Data.([]order.Detail)
	require.True(t, ok, "wsProcessOrder must send order details")
	require.Len(t, details, 2, "wsProcessOrder must send both orders")
	assert.Equal(t, 120000.3, details[0].ExecutedQuoteAmount, "linear order should retain the quote-denominated executed value")
	assert.Equal(t, 0.066, details[0].Fee, "linear order fee should come from cumFeeDetail")
	assert.Equal(t, currency.USDT, details[0].FeeAsset, "linear order fee currency should come from cumFeeDetail")
	assert.Zero(t, details[1].ExecutedQuoteAmount, "inverse order should not expose its settlement-denominated executed value as a quote amount")
	assert.Equal(t, 0.000003, details[1].Fee, "inverse order fee should come from cumExecFee")
	assert.Equal(t, currency.EMPTYCODE, details[1].FeeAsset, "inverse order fee currency should stay unset when only cumExecFee is reported")
}
