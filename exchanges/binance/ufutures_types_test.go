package binance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestAssetIndexResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	index := AssetIndex{
		Symbol:                "BTCUSD",
		Time:                  types.Time(time.UnixMilli(1791283500001)),
		Index:                 86168.37311999,
		BidBuffer:             0.05,
		AskBuffer:             0.05,
		BidRate:               81859.95446399,
		AskRate:               90476.79177599,
		AutoExchangeBidBuffer: 0.025,
		AutoExchangeAskBuffer: 0.025,
		AutoExchangeBidRate:   84014.16379199,
		AutoExchangeAskRate:   88322.58244799,
	}
	const object = `{"symbol":"BTCUSD","time":1791283500001,"index":"86168.37311999","bidBuffer":"0.05000000","askBuffer":"0.05000000","bidRate":"81859.95446399","askRate":"90476.79177599","autoExchangeBidBuffer":"0.02500000","autoExchangeAskBuffer":"0.02500000","autoExchangeBidRate":"84014.16379199","autoExchangeAskRate":"88322.58244799"}`
	var got AssetIndexResponse
	require.NoError(t, json.Unmarshal([]byte(object), &got), "Unmarshal must not error for an object")
	assert.Equal(t, AssetIndexResponse{index}, got, "AssetIndexResponse should decode a single object into one element")

	got = nil
	require.NoError(t, json.Unmarshal([]byte("["+object+","+object+"]"), &got), "Unmarshal must not error for an array")
	assert.Equal(t, AssetIndexResponse{index, index}, got, "AssetIndexResponse should decode an array")

	assert.Error(t, json.Unmarshal([]byte(`"oops"`), &got), "AssetIndexResponse should reject a value that is neither an object nor an array")
}

func TestUKlineUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var got UKline
	require.NoError(t, json.Unmarshal([]byte(`[1744156800000,"76298.00","83554.90","74578.50","82588.00","560649.862",1744243199999,"44072322884.09750",8799174,"279060.371","21941548090.15170","0"]`), &got), "Unmarshal must not error")
	exp := UKline{
		OpenTime:                 types.Time(time.UnixMilli(1744156800000)),
		Open:                     76298,
		High:                     83554.9,
		Low:                      74578.5,
		Close:                    82588,
		Volume:                   560649.862,
		CloseTime:                types.Time(time.UnixMilli(1744243199999)),
		QuoteAssetVolume:         44072322884.0975,
		NumberOfTrades:           8799174,
		TakerBuyBaseAssetVolume:  279060.371,
		TakerBuyQuoteAssetVolume: 21941548090.1517,
	}
	assert.Equal(t, exp, got, "UKline should decode every element")
	assert.Error(t, json.Unmarshal([]byte(`{"openTime":1744156800000}`), &got), "UKline should reject an object")
	assert.Error(t, json.Unmarshal([]byte(`[1744156800000,"x"]`), &got), "UKline should reject an invalid element")
}

func TestUPriceKlineUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var got UPriceKline
	require.NoError(t, json.Unmarshal([]byte(`[1744156800000,"76322.96800000","83567.47466667","74626.60222222","82611.44511111","0",1744243199999,"0",86399,"0","0","0"]`), &got), "Unmarshal must not error")
	exp := UPriceKline{
		OpenTime:  types.Time(time.UnixMilli(1744156800000)),
		Open:      76322.968,
		High:      83567.47466667,
		Low:       74626.60222222,
		Close:     82611.44511111,
		CloseTime: types.Time(time.UnixMilli(1744243199999)),
	}
	assert.Equal(t, exp, got, "UPriceKline should decode the price and time elements")
	assert.Error(t, json.Unmarshal([]byte(`{"openTime":1744156800000}`), &got), "UPriceKline should reject an object")
	assert.Error(t, json.Unmarshal([]byte(`[1744156800000,"x"]`), &got), "UPriceKline should reject an invalid element")
}

func TestUDownloadLinkResponseUnmarshal(t *testing.T) {
	t.Parallel()
	// A file still being prepared has no link yet: its expiry is -1 and isExpired is null
	var got UDownloadLinkResponse
	require.NoError(t, json.Unmarshal([]byte(`{"downloadId":"545923594199212032","status":"processing","url":"","notified":false,"expirationTimestamp":-1,"isExpired":null}`), &got), "Unmarshal must not error")
	exp := UDownloadLinkResponse{
		DownloadID:          "545923594199212032",
		Status:              "processing",
		ExpirationTimestamp: -1,
	}
	assert.Equal(t, exp, got, "UDownloadLinkResponse should decode a file that is still processing")
}

func TestUAccountTradeHistoryUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `
{
  "buyer": true,
  "commission": "-0.078125",
  "commissionAsset": "USDT",
  "id": 698759,
  "maker": true,
  "orderId": 25851813,
  "price": "7819.25",
  "qty": "0.0625",
  "baseQty": "0.03125",
  "quoteQty": "15.640625",
  "realizedPnl": "-0.9375",
  "side": "SELL",
  "positionSide": "SHORT",
  "marginAsset": "USDT",
  "symbol": "BTCUSDT",
  "pair": "BTCUSDT",
  "time": 1569514978020
}
`

	var x UAccountTradeHistory
	require.NoError(t, json.Unmarshal([]byte(inp), &x), "Unmarshal must not error")
	exp := UAccountTradeHistory{
		Buyer:           true,
		Commission:      -0.078125,
		CommissionAsset: currency.USDT,
		ID:              698759,
		Maker:           true,
		OrderID:         25851813,
		Price:           7819.25,
		Quantity:        0.0625,
		BaseQuantity:    0.03125,
		QuoteQuantity:   15.640625,
		RealizedPNL:     -0.9375,
		Side:            "SELL",
		MarginAsset:     currency.USDT,
		PositionSide:    "SHORT",
		Symbol:          "BTCUSDT",
		Pair:            "BTCUSDT",
		Time:            types.Time(time.UnixMilli(1569514978020)),
	}
	assert.Equal(t, exp, x, "UAccountTradeHistory should unmarshal correctly")
}

func TestUPublicTradesDataUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `
{
  "id": 8027591209,
  "price": "79673.50",
  "qty": "0.0625",
  "quoteQty": "318.6875",
  "time": 1787893510850,
  "isBuyerMaker": true,
  "isRPITrade": true
}
`

	var x UPublicTradesData
	require.NoError(t, json.Unmarshal([]byte(inp), &x), "Unmarshal must not error")
	exp := UPublicTradesData{
		ID:            8027591209,
		Price:         79673.50,
		Quantity:      0.0625,
		QuoteQuantity: 318.6875,
		Time:          types.Time(time.UnixMilli(1787893510850)),
		IsBuyerMaker:  true,
		IsRPITrade:    true,
	}
	assert.Equal(t, exp, x, "UPublicTradesData should unmarshal correctly")
}

func TestUFuturesOrderDataUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `
{
  "avgPrice": "4096.5",
  "clientOrderId": "customID",
  "cumQuote": "16.5",
  "cumBase": "8.25",
  "executedQty": "4.125",
  "orderId": 1573346959,
  "origQty": "32.75",
  "origType": "MARKET",
  "price": "2048.25",
  "reduceOnly": true,
  "side": "BUY",
  "positionSide": "SHORT",
  "status": "NEW",
  "stopPrice": "1024.75",
  "closePosition": true,
  "symbol": "BTCUSDT",
  "pair": "BTCUSDT",
  "time": 1579276756075,
  "timeInForce": "GTC",
  "type": "LIMIT",
  "activatePrice": "512.5",
  "priceRate": "0.35",
  "updateTime": 1635931801320,
  "workingType": "CONTRACT_PRICE",
  "priceProtect": true,
  "priceMatch": "OPPONENT",
  "selfTradePreventionMode": "EXPIRE_MAKER",
  "goodTillDate": 1693207680000
}
`

	var x UFuturesOrderData
	require.NoError(t, json.Unmarshal([]byte(inp), &x), "Unmarshal must not error")
	exp := UFuturesOrderData{
		ClientOrderID:           "customID",
		ExecutedQuantity:        4.125,
		OrderID:                 1573346959,
		OriginalQuantity:        32.75,
		Price:                   2048.25,
		ReduceOnly:              true,
		Side:                    "BUY",
		PositionSide:            "SHORT",
		Status:                  "NEW",
		StopPrice:               1024.75,
		ClosePosition:           true,
		Symbol:                  "BTCUSDT",
		TimeInForce:             "GTC",
		Type:                    "LIMIT",
		OriginalType:            "MARKET",
		UpdateTime:              types.Time(time.UnixMilli(1635931801320)),
		WorkingType:             "CONTRACT_PRICE",
		PriceProtect:            true,
		PriceMatch:              "OPPONENT",
		SelfTradePreventionMode: "EXPIRE_MAKER",
		GoodTillDate:            types.Time(time.UnixMilli(1693207680000)),
		AveragePrice:            4096.5,
		CumulativeQuote:         16.5,
		Time:                    types.Time(time.UnixMilli(1579276756075)),
		ActivatePrice:           512.5,
		PriceRate:               0.35,
		CumulativeBase:          8.25,
		Pair:                    "BTCUSDT",
	}
	assert.Equal(t, exp, x, "UFuturesOrderData should unmarshal correctly")
}

func TestUAccountBalanceUnmarshal(t *testing.T) {
	t.Parallel()
	want := UAccountBalance{
		AccountAlias:       "test-account",
		Asset:              currency.USDT,
		Balance:            125.5,
		CrossWalletBalance: 120.25,
		CrossUnrealizedPNL: -0.5,
		AvailableBalance:   115.75,
		MaxWithdrawAmount:  110.25,
	}
	for _, tc := range []struct {
		name  string
		input string
		want  UAccountBalance
	}{
		{
			name: "quoted numbers",
			input: `[
  {
    "accountAlias": "test-account",
    "asset": "USDT",
    "balance": "125.5",
    "crossWalletBalance": "120.25",
    "crossUnPnl": "-0.5",
    "availableBalance": "115.75",
    "maxWithdrawAmount": "110.25"
  }
]`,
			want: want,
		},
		{
			name: "bare numbers",
			input: `[
  {
    "accountAlias": "test-account",
    "asset": "USDT",
    "balance": 125.5,
    "crossWalletBalance": 120.25,
    "crossUnPnl": -0.5,
    "availableBalance": 115.75,
    "maxWithdrawAmount": 110.25
  }
]`,
			want: want,
		},
		{
			name: "empty numbers",
			input: `[
  {
    "accountAlias": "test-account",
    "asset": "USDT",
    "balance": "",
    "crossWalletBalance": "",
    "crossUnPnl": "",
    "availableBalance": "",
    "maxWithdrawAmount": ""
  }
]`,
			want: UAccountBalance{
				AccountAlias: "test-account",
				Asset:        currency.USDT,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got []UAccountBalance
			require.NoError(t, json.Unmarshal([]byte(tc.input), &got), "Unmarshal must not error")
			assert.Equal(t, []UAccountBalance{tc.want}, got, "UAccountBalance should decode every field")
		})
	}
}

func TestUOrderResponseUnmarshal(t *testing.T) {
	t.Parallel()
	base := UOrderBase{
		ClientOrderID:           "testOrder",
		OrderID:                 22542179,
		Side:                    "BUY",
		PositionSide:            "SHORT",
		Status:                  "NEW",
		Symbol:                  "BTCUSDT",
		TimeInForce:             "GTC",
		Type:                    "TRAILING_STOP_MARKET",
		OriginalType:            "TRAILING_STOP_MARKET",
		UpdateTime:              types.Time(time.UnixMilli(1566818724722)),
		WorkingType:             "CONTRACT_PRICE",
		PriceProtect:            true,
		PriceMatch:              "NONE",
		SelfTradePreventionMode: "NONE",
		GoodTillDate:            types.Time(time.UnixMilli(1693207680000)),
	}
	filled := base
	filled.ExecutedQuantity = 0.5
	filled.OriginalQuantity = 1.25
	filled.Price = 19999.25
	filled.StopPrice = 18000.5
	want := UOrderResponse{
		UOrderBase:      filled,
		AveragePrice:    20001.5,
		CumulativeQuote: 10000.75,
		Time:            types.Time(time.UnixMilli(1579276756075)),
		ActivatePrice:   9020.5,
		PriceRate:       0.3,
	}
	const shared = `
  "clientOrderId": "testOrder",
  "orderId": 22542179,
  "side": "BUY",
  "positionSide": "SHORT",
  "status": "NEW",
  "symbol": "BTCUSDT",
  "timeInForce": "GTC",
  "type": "TRAILING_STOP_MARKET",
  "origType": "TRAILING_STOP_MARKET",
  "time": 1579276756075,
  "updateTime": 1566818724722,
  "workingType": "CONTRACT_PRICE",
  "priceProtect": true,
  "priceMatch": "NONE",
  "selfTradePreventionMode": "NONE",
  "goodTillDate": 1693207680000
}`

	for _, tc := range []struct {
		name  string
		input string
		want  UOrderResponse
	}{
		{
			name: "quoted numbers",
			input: `{
  "cumQuote": "10000.75",
  "executedQty": "0.5",
  "avgPrice": "20001.5",
  "origQty": "1.25",
  "price": "19999.25",
  "stopPrice": "18000.5",
  "activatePrice": "9020.5",
  "priceRate": "0.3",` + shared,
			want: want,
		},
		{
			// Binance sends avgPrice and priceRate unquoted on some order types, which a float64 field with the string
			// option could not decode
			name: "bare numbers",
			input: `{
  "cumQuote": 10000.75,
  "executedQty": 0.5,
  "avgPrice": 20001.5,
  "origQty": 1.25,
  "price": 19999.25,
  "stopPrice": 18000.5,
  "activatePrice": 9020.5,
  "priceRate": 0.3,` + shared,
			want: want,
		},
		{
			// and an empty string where a trailing stop field does not apply
			name: "empty numbers",
			input: `{
  "cumQuote": "",
  "executedQty": "",
  "avgPrice": "",
  "origQty": "",
  "price": "",
  "stopPrice": "",
  "activatePrice": "",
  "priceRate": "",` + shared,
			want: UOrderResponse{
				UOrderBase: base,
				Time:       types.Time(time.UnixMilli(1579276756075)),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got UOrderResponse
			require.NoError(t, json.Unmarshal([]byte(tc.input), &got), "Unmarshal must not error")
			assert.Equal(t, tc.want, got, "UOrderResponse should decode every field")
		})
	}
}
