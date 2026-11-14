package binanceus

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// custodialTestRail is the custodial partner the custodial tests and their recorded responses use
const custodialTestRail = "ANCHORAGE"

func TestGetCustodialAccountBalance(t *testing.T) {
	t.Parallel()
	_, err := e.GetCustodialAccountBalance(t.Context(), "")
	require.ErrorIs(t, err, errRailRequired, "GetCustodialAccountBalance must require a rail")

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/custodian/balance", url.Values{"rail": {custodialTestRail}}, `{}`)
		_, err := ex.GetCustodialAccountBalance(t.Context(), custodialTestRail)
		require.NoError(t, err, "GetCustodialAccountBalance must not error")
	})

	t.Run("returns the balances", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetCustodialAccountBalance(t.Context(), custodialTestRail)
		require.NoError(t, err, "GetCustodialAccountBalance must not error")
		require.NotNil(t, result, "GetCustodialAccountBalance must return a result")
		if mockTests {
			exp := &CustodialAccountBalanceResponse{
				ExchangeWalletBalance: []CustodialBalance{
					{Asset: currency.BNB, Free: 0.0083686, LastUpdatedTime: types.Time(time.UnixMilli(1662535206000))},
				},
				CustodialAccountBalance: []CustodialBalance{
					{Asset: currency.BTC, Free: 9.02933865, Locked: 0.5, LastUpdatedTime: types.Time(time.UnixMilli(1658478839000))},
					{Asset: currency.NewCode("ETHT"), Free: 61.00129998, InSettlement: 25.17, LastUpdatedTime: types.Time(time.UnixMilli(1663752210000))},
				},
			}
			assert.Equal(t, exp, result, "GetCustodialAccountBalance should decode the balances")
		}
	})
}

func TestGetCustodialSupportedAssetList(t *testing.T) {
	t.Parallel()
	_, err := e.GetCustodialSupportedAssetList(t.Context(), "")
	require.ErrorIs(t, err, errRailRequired, "GetCustodialSupportedAssetList must require a rail")

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/custodian/supportedAssetList", url.Values{"rail": {custodialTestRail}}, `{}`)
		_, err := ex.GetCustodialSupportedAssetList(t.Context(), custodialTestRail)
		require.NoError(t, err, "GetCustodialSupportedAssetList must not error")
	})

	t.Run("returns the supported assets", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetCustodialSupportedAssetList(t.Context(), custodialTestRail)
		require.NoError(t, err, "GetCustodialSupportedAssetList must not error")
		require.NotNil(t, result, "GetCustodialSupportedAssetList must return a result")
		if mockTests {
			assets := []CustodialSupportedAsset{
				{Asset: currency.BTC, Precision: 8, Network: []string{"BTC"}},
				{Asset: currency.ETH, Precision: 8, Network: []string{"ETH"}},
			}
			exp := &CustodialSupportedAssetListResponse{TransferEligible: assets, SettlementEligible: assets}
			assert.Equal(t, exp, result, "GetCustodialSupportedAssetList should decode the supported assets")
		}
	})
}

func TestTransferFromExchangeWallet(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CustodialAssetTransferRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no rail", req: &CustodialAssetTransferRequest{Asset: currency.BTC, Amount: 1}, err: errRailRequired},
		{name: "no asset", req: &CustodialAssetTransferRequest{Rail: custodialTestRail, Amount: 1}, err: currency.ErrCurrencyCodeEmpty},
		{name: "no amount", req: &CustodialAssetTransferRequest{Rail: custodialTestRail, Asset: currency.BTC}, err: order.ErrAmountIsInvalid},
	} {
		_, err := e.TransferFromExchangeWallet(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "TransferFromExchangeWallet must refuse a request with %s", tc.name)
	}

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *CustodialAssetTransferRequest
			exp  url.Values
		}{
			{
				name: "generated client order ID",
				req:  &CustodialAssetTransferRequest{Rail: custodialTestRail, Asset: currency.NewCode("btc"), Amount: 0.00012345},
				exp:  url.Values{"rail": {custodialTestRail}, "asset": {"BTC"}, "amount": {"0.00012345"}},
			},
			{
				name: "client order ID",
				req:  &CustodialAssetTransferRequest{Rail: custodialTestRail, Asset: currency.BTC, Amount: 1.5, ClientOrderID: "1663752208086"},
				exp:  url.Values{"rail": {custodialTestRail}, "asset": {"BTC"}, "amount": {"1.5"}, "clientOrderId": {"1663752208086"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodPost, "/sapi/v1/custodian/walletTransfer", tc.exp, `{}`)
				_, err := ex.TransferFromExchangeWallet(t.Context(), tc.req)
				require.NoError(t, err, "TransferFromExchangeWallet must not error")
			})
		}
	})

	t.Run("transfers the asset", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.TransferFromExchangeWallet(t.Context(), &CustodialAssetTransferRequest{Rail: custodialTestRail, Asset: currency.BTC, Amount: 1, ClientOrderID: "1663752208086"})
		require.NoError(t, err, "TransferFromExchangeWallet must not error")
		require.NotNil(t, result, "TransferFromExchangeWallet must return a result")
		if mockTests {
			exp := &TransferFromExchangeWalletResponse{
				Asset:         currency.BTC,
				Amount:        1,
				ClientOrderID: "1663752208086",
				TransferID:    "2022092709232937542366",
				Status:        "SUCCESS",
				CreateTime:    types.Time(time.UnixMilli(1664276400000)),
			}
			assert.Equal(t, exp, result, "TransferFromExchangeWallet should decode the transfer")
		}
	})
}

func TestTransferFromCustodian(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CustodialAssetTransferRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no rail", req: &CustodialAssetTransferRequest{Asset: currency.BTC, Amount: 1}, err: errRailRequired},
		{name: "no asset", req: &CustodialAssetTransferRequest{Rail: custodialTestRail, Amount: 1}, err: currency.ErrCurrencyCodeEmpty},
		{name: "no amount", req: &CustodialAssetTransferRequest{Rail: custodialTestRail, Asset: currency.BTC}, err: order.ErrAmountIsInvalid},
	} {
		_, err := e.TransferFromCustodian(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "TransferFromCustodian must refuse a request with %s", tc.name)
	}

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *CustodialAssetTransferRequest
			exp  url.Values
		}{
			{
				name: "generated client order ID",
				req:  &CustodialAssetTransferRequest{Rail: custodialTestRail, Asset: currency.NewCode("btc"), Amount: 0.00012345},
				exp:  url.Values{"rail": {custodialTestRail}, "asset": {"BTC"}, "amount": {"0.00012345"}},
			},
			{
				name: "client order ID",
				req:  &CustodialAssetTransferRequest{Rail: custodialTestRail, Asset: currency.BTC, Amount: 1.5, ClientOrderID: "1663752208086"},
				exp:  url.Values{"rail": {custodialTestRail}, "asset": {"BTC"}, "amount": {"1.5"}, "clientOrderId": {"1663752208086"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodPost, "/sapi/v1/custodian/custodianTransfer", tc.exp, `{}`)
				_, err := ex.TransferFromCustodian(t.Context(), tc.req)
				require.NoError(t, err, "TransferFromCustodian must not error")
			})
		}
	})

	t.Run("transfers the asset", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.TransferFromCustodian(t.Context(), &CustodialAssetTransferRequest{Rail: custodialTestRail, Asset: currency.BTC, Amount: 1, ClientOrderID: "1663752208086"})
		require.NoError(t, err, "TransferFromCustodian must not error")
		require.NotNil(t, result, "TransferFromCustodian must return a result")
		if mockTests {
			exp := &TransferFromCustodianResponse{
				Asset:              currency.BTC,
				Amount:             1,
				ClientOrderID:      "1663752208086",
				TransferID:         "2022092709232937542366",
				CustodyAccountID:   "custodyAccountId",
				CustodyAccountName: "custodyAccountName",
				Status:             "Transfer request received ",
				CreateTime:         types.Time(time.UnixMilli(1664276400000)),
			}
			assert.Equal(t, exp, result, "TransferFromCustodian should decode the transfer")
		}
	})
}

func TestUndoCustodialTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.UndoCustodialTransfer(t.Context(), "", "2022092709232937542366")
	require.ErrorIs(t, err, errRailRequired, "UndoCustodialTransfer must require a rail")
	_, err = e.UndoCustodialTransfer(t.Context(), custodialTestRail, "")
	require.ErrorIs(t, err, errOriginTransferIDRequired, "UndoCustodialTransfer must require the transfer's ID")

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodPost, "/sapi/v1/custodian/undoTransfer", url.Values{"rail": {custodialTestRail}, "originTransferId": {"2022092709232937542366"}}, `{}`)
		_, err := ex.UndoCustodialTransfer(t.Context(), custodialTestRail, "2022092709232937542366")
		require.NoError(t, err, "UndoCustodialTransfer must not error")
	})

	t.Run("undoes the transfer", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.UndoCustodialTransfer(t.Context(), custodialTestRail, "2022092709232937542366")
		require.NoError(t, err, "UndoCustodialTransfer must not error")
		require.NotNil(t, result, "UndoCustodialTransfer must return a result")
		if mockTests {
			assert.Equal(t, &UndoCustodialTransferResponse{TransferID: "2022092709232937542366", Asset: currency.BTC, Amount: 1}, result, "UndoCustodialTransfer should decode the undone transfer")
		}
	})
}

func TestGetExchangeWalletTransfer(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1664276000000), time.UnixMilli(1664277000000)
	for _, tc := range []struct {
		name string
		req  *ExchangeWalletTransferRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no rail", req: &ExchangeWalletTransferRequest{}, err: errRailRequired},
		{name: "a reversed time range", req: &ExchangeWalletTransferRequest{Rail: custodialTestRail, StartTime: end, EndTime: start}, err: common.ErrStartAfterEnd},
	} {
		_, err := e.GetExchangeWalletTransfer(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "GetExchangeWalletTransfer must refuse a request with %s", tc.name)
	}

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *ExchangeWalletTransferRequest
			exp  url.Values
		}{
			{name: "rail only", req: &ExchangeWalletTransferRequest{Rail: custodialTestRail}, exp: url.Values{"rail": {custodialTestRail}}},
			{
				name: "every filter",
				req: &ExchangeWalletTransferRequest{
					Rail:          custodialTestRail,
					TransferID:    "2022092709232937542366",
					ClientOrderID: "1663752208086",
					Asset:         currency.NewCode("btc"),
					StartTime:     start,
					EndTime:       end,
					Page:          2,
					Limit:         100,
				},
				exp: url.Values{
					"rail":          {custodialTestRail},
					"transferId":    {"2022092709232937542366"},
					"clientOrderId": {"1663752208086"},
					"asset":         {"BTC"},
					"startTime":     {"1664276000000"},
					"endTime":       {"1664277000000"},
					"page":          {"2"},
					"limit":         {"100"},
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/custodian/walletTransferHistory", tc.exp, `{}`)
				_, err := ex.GetExchangeWalletTransfer(t.Context(), tc.req)
				require.NoError(t, err, "GetExchangeWalletTransfer must not error")
			})
		}
	})

	t.Run("returns the transfers", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetExchangeWalletTransfer(t.Context(), &ExchangeWalletTransferRequest{Rail: custodialTestRail, TransferID: "2022092709232937542366"})
		require.NoError(t, err, "GetExchangeWalletTransfer must not error")
		require.NotNil(t, result, "GetExchangeWalletTransfer must return a result")
		if mockTests {
			exp := &ExchangeWalletTransferResponse{
				Data: []ExchangeWalletTransfer{{
					TransferID:    "2022092709232937542366",
					ClientOrderID: "1663752208086",
					Asset:         currency.BTC,
					Amount:        1,
					Status:        "SUCCESS",
					CreateTime:    types.Time(time.UnixMilli(1664276400000)),
					UpdateTime:    types.Time(time.UnixMilli(1664276400000)),
				}},
				Total: 1,
			}
			assert.Equal(t, exp, result, "GetExchangeWalletTransfer should decode the transfers")
		}
	})
}

func TestGetCustodianTransfer(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1664276000000), time.UnixMilli(1664277000000)
	for _, tc := range []struct {
		name string
		req  *CustodianTransferRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no rail", req: &CustodianTransferRequest{}, err: errRailRequired},
		{name: "a reversed time range", req: &CustodianTransferRequest{Rail: custodialTestRail, StartTime: end, EndTime: start}, err: common.ErrStartAfterEnd},
	} {
		_, err := e.GetCustodianTransfer(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "GetCustodianTransfer must refuse a request with %s", tc.name)
	}

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *CustodianTransferRequest
			exp  url.Values
		}{
			{name: "rail only", req: &CustodianTransferRequest{Rail: custodialTestRail}, exp: url.Values{"rail": {custodialTestRail}}},
			{
				name: "every filter",
				req: &CustodianTransferRequest{
					Rail:                 custodialTestRail,
					TransferID:           "2022092709232937542366",
					ClientOrderID:        "1663752208086",
					ExpressTradeTransfer: true,
					Asset:                currency.NewCode("btc"),
					StartTime:            start,
					EndTime:              end,
					Page:                 2,
					Limit:                100,
				},
				exp: url.Values{
					"rail":                 {custodialTestRail},
					"transferId":           {"2022092709232937542366"},
					"clientOrderId":        {"1663752208086"},
					"expressTradeTransfer": {"true"},
					"asset":                {"BTC"},
					"startTime":            {"1664276000000"},
					"endTime":              {"1664277000000"},
					"page":                 {"2"},
					"limit":                {"100"},
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/custodian/custodianTransferHistory", tc.exp, `{}`)
				_, err := ex.GetCustodianTransfer(t.Context(), tc.req)
				require.NoError(t, err, "GetCustodianTransfer must not error")
			})
		}
	})

	t.Run("returns the transfers", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetCustodianTransfer(t.Context(), &CustodianTransferRequest{Rail: custodialTestRail, TransferID: "2022092709232937542366"})
		require.NoError(t, err, "GetCustodianTransfer must not error")
		require.NotNil(t, result, "GetCustodianTransfer must return a result")
		if mockTests {
			exp := &CustodianTransferResponse{
				Data: []CustodianTransfer{{
					TransferID:    "2022092709232937542366",
					ClientOrderID: "1663752208086",
					Asset:         currency.BTC,
					Amount:        1,
					Status:        "SUCCESS",
					ExpressTrade:  true,
					CreateTime:    types.Time(time.UnixMilli(1664276400000)),
					UpdateTime:    types.Time(time.UnixMilli(1664276400000)),
				}},
				Total: 1,
			}
			assert.Equal(t, exp, result, "GetCustodianTransfer should decode the transfers")
		}
	})
}

func TestCustodialNewOrder(t *testing.T) {
	t.Parallel()
	ethtusd := currency.NewPair(currency.NewCode("ETHT"), currency.USD)
	for _, tc := range []struct {
		name string
		req  *CustodialNewOrderRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no rail", req: &CustodialNewOrderRequest{Symbol: ethtusd, Side: SideBuy, Type: OrderTypeMarket}, err: errRailRequired},
		{name: "no symbol", req: &CustodialNewOrderRequest{Rail: custodialTestRail, Side: SideBuy, Type: OrderTypeMarket}, err: currency.ErrCurrencyPairEmpty},
		{name: "no side", req: &CustodialNewOrderRequest{Rail: custodialTestRail, Symbol: ethtusd, Type: OrderTypeMarket}, err: order.ErrSideIsInvalid},
		{name: "no type", req: &CustodialNewOrderRequest{Rail: custodialTestRail, Symbol: ethtusd, Side: SideBuy}, err: order.ErrTypeIsInvalid},
	} {
		_, err := e.CustodialNewOrder(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "CustodialNewOrder must refuse a request with %s", tc.name)
	}

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *CustodialNewOrderRequest
			exp  url.Values
		}{
			{
				name: "market order by quantity",
				req:  &CustodialNewOrderRequest{Rail: custodialTestRail, Symbol: currency.NewBTCUSD(), Side: SideSell, Type: OrderTypeMarket, Quantity: 0.00012345},
				exp:  url.Values{"rail": {custodialTestRail}, "symbol": {"BTCUSD"}, "side": {"SELL"}, "type": {"MARKET"}, "quantity": {"0.00012345"}},
			},
			{
				name: "market order by quote quantity",
				req:  &CustodialNewOrderRequest{Rail: custodialTestRail, Symbol: currency.NewBTCUSD(), Side: SideBuy, Type: OrderTypeMarket, QuoteOrderQuantity: 25.5},
				exp:  url.Values{"rail": {custodialTestRail}, "symbol": {"BTCUSD"}, "side": {"BUY"}, "type": {"MARKET"}, "quoteOrderQty": {"25.5"}},
			},
			{
				name: "stop loss limit iceberg order with an ExpressTrade",
				req: &CustodialNewOrderRequest{
					Rail:              custodialTestRail,
					Symbol:            currency.NewBTCUSD(),
					Side:              SideBuy,
					Type:              OrderTypeStopLossLimit,
					TimeInForce:       TimeInForceGoodTillCancel,
					Quantity:          0.5,
					Price:             60000.01,
					StopPrice:         59000.5,
					IcebergQuantity:   0.1,
					Asset:             currency.NewCode("usd"),
					AllowExpressTrade: true,
				},
				exp: url.Values{
					"rail":              {custodialTestRail},
					"symbol":            {"BTCUSD"},
					"side":              {"BUY"},
					"type":              {"STOP_LOSS_LIMIT"},
					"timeInForce":       {"GTC"},
					"quantity":          {"0.5"},
					"price":             {"60000.01"},
					"stopPrice":         {"59000.5"},
					"icebergQty":        {"0.1"},
					"asset":             {"USD"},
					"allowExpressTrade": {"true"},
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodPost, "/sapi/v1/custodian/order", tc.exp, `{}`)
				_, err := ex.CustodialNewOrder(t.Context(), tc.req)
				require.NoError(t, err, "CustodialNewOrder must not error")
			})
		}
	})

	t.Run("places the order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.CustodialNewOrder(t.Context(), &CustodialNewOrderRequest{
			Rail:              custodialTestRail,
			Symbol:            ethtusd,
			Side:              SideBuy,
			Type:              OrderTypeLimit,
			TimeInForce:       TimeInForceGoodTillCancel,
			Quantity:          1,
			Price:             18,
			Asset:             currency.USD,
			AllowExpressTrade: true,
		})
		require.NoError(t, err, "CustodialNewOrder must not error")
		require.NotNil(t, result, "CustodialNewOrder must return a result")
		if mockTests {
			exp := &CustodialNewOrderResponse{
				Symbol:                  "ETHTUSD",
				OrderID:                 70,
				OrderListID:             -1,
				ClientOrderID:           "7c713b92cce34358b23fcf1fcd9953bc",
				Price:                   18,
				OriginalQuantity:        1,
				ExecutedQuantity:        0.5,
				CumulativeQuoteQuantity: 9,
				Status:                  "PARTIALLY_FILLED",
				TimeInForce:             TimeInForceGoodTillCancel,
				Type:                    OrderTypeLimit,
				Side:                    SideBuy,
				ExpressTradeFlag:        true,
			}
			assert.Equal(t, exp, result, "CustodialNewOrder should decode the order")
		}
	})
}

func TestCustodialNewOneCancelsTheOtherOrder(t *testing.T) {
	t.Parallel()
	ltcbtc := currency.NewPair(currency.LTC, currency.BTC)
	valid := CustodialNewOneCancelsTheOtherOrderRequest{Rail: custodialTestRail, Symbol: ltcbtc, Side: SideBuy, Quantity: 0.624363, Price: 0.036435, StopPrice: 0.960664}
	for _, tc := range []struct {
		name   string
		modify func(*CustodialNewOneCancelsTheOtherOrderRequest)
		err    error
	}{
		{name: "no rail", modify: func(r *CustodialNewOneCancelsTheOtherOrderRequest) { r.Rail = "" }, err: errRailRequired},
		{name: "no symbol", modify: func(r *CustodialNewOneCancelsTheOtherOrderRequest) { r.Symbol = currency.EMPTYPAIR }, err: currency.ErrCurrencyPairEmpty},
		{name: "no side", modify: func(r *CustodialNewOneCancelsTheOtherOrderRequest) { r.Side = "" }, err: order.ErrSideIsInvalid},
		{name: "no quantity", modify: func(r *CustodialNewOneCancelsTheOtherOrderRequest) { r.Quantity = 0 }, err: order.ErrAmountIsInvalid},
		{name: "no price", modify: func(r *CustodialNewOneCancelsTheOtherOrderRequest) { r.Price = 0 }, err: order.ErrPriceMustBeSetIfLimitOrder},
		{name: "no stop price", modify: func(r *CustodialNewOneCancelsTheOtherOrderRequest) { r.StopPrice = 0 }, err: errStopPriceRequired},
		{name: "a stop limit price without its time in force", modify: func(r *CustodialNewOneCancelsTheOtherOrderRequest) { r.StopLimitPrice = 0.97 }, err: errStopLimitTimeInForceRequired},
	} {
		req := valid
		tc.modify(&req)
		_, err := e.CustodialNewOneCancelsTheOtherOrder(t.Context(), &req)
		require.ErrorIsf(t, err, tc.err, "CustodialNewOneCancelsTheOtherOrder must refuse a request with %s", tc.name)
	}
	_, err := e.CustodialNewOneCancelsTheOtherOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CustodialNewOneCancelsTheOtherOrder must refuse a nil request")

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *CustodialNewOneCancelsTheOtherOrderRequest
			exp  url.Values
		}{
			{
				name: "mandatory parameters",
				req:  &valid,
				exp:  url.Values{"rail": {custodialTestRail}, "symbol": {"LTCBTC"}, "side": {"BUY"}, "quantity": {"0.624363"}, "price": {"0.036435"}, "stopPrice": {"0.960664"}},
			},
			{
				name: "every parameter",
				req: &CustodialNewOneCancelsTheOtherOrderRequest{
					Rail:                 custodialTestRail,
					Symbol:               ltcbtc,
					Side:                 SideSell,
					Quantity:             0.00012345,
					LimitClientOrderID:   "limit-1",
					Price:                0.05,
					LimitIcebergQuantity: 0.00002,
					StopClientOrderID:    "stop-1",
					StopPrice:            0.03,
					StopLimitPrice:       0.0299,
					StopIcebergQuantity:  0.00003,
					StopLimitTimeInForce: TimeInForceFillOrKill,
					Asset:                currency.NewCode("ltc"),
					AllowExpressTrade:    true,
				},
				exp: url.Values{
					"rail":                 {custodialTestRail},
					"symbol":               {"LTCBTC"},
					"side":                 {"SELL"},
					"quantity":             {"0.00012345"},
					"limitClientOrderId":   {"limit-1"},
					"price":                {"0.05"},
					"limitIcebergQty":      {"0.00002"},
					"stopClientOrderId":    {"stop-1"},
					"stopPrice":            {"0.03"},
					"stopLimitPrice":       {"0.0299"},
					"stopIcebergQty":       {"0.00003"},
					"stopLimitTimeInForce": {"FOK"},
					"asset":                {"LTC"},
					"allowExpressTrade":    {"true"},
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodPost, "/sapi/v1/custodian/ocoOrder", tc.exp, `{}`)
				_, err := ex.CustodialNewOneCancelsTheOtherOrder(t.Context(), tc.req)
				require.NoError(t, err, "CustodialNewOneCancelsTheOtherOrder must not error")
			})
		}
	})

	t.Run("places the order list", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		req := valid
		req.LimitClientOrderID = "xTXKaGYd4bluPVp78IVRvl"
		req.StopClientOrderID = "Kk7sqHb9J6mJWTMDVW7Vos"
		req.Asset = currency.BTC
		req.AllowExpressTrade = true
		result, err := e.CustodialNewOneCancelsTheOtherOrder(t.Context(), &req)
		require.NoError(t, err, "CustodialNewOneCancelsTheOtherOrder must not error")
		require.NotNil(t, result, "CustodialNewOneCancelsTheOtherOrder must return a result")
		if mockTests {
			exp := &CustodialOneCancelsTheOtherOrderResponse{
				ContingencyType:   "OCO",
				ListStatusType:    "EXEC_STARTED",
				ListOrderStatus:   "EXECUTING",
				ListClientOrderID: "JYVpp3F0f5CAG15DhtrqLp",
				TransactionTime:   types.Time(time.UnixMilli(1563417480525)),
				Symbol:            "LTCBTC",
				ExpressTradeFlag:  true,
				Orders: []CustodialOrderListOrder{
					{Symbol: "LTCBTC", OrderID: 2, ClientOrderID: "Kk7sqHb9J6mJWTMDVW7Vos"},
					{Symbol: "LTCBTC", OrderID: 3, ClientOrderID: "xTXKaGYd4bluPVp78IVRvl"},
				},
				OrderReports: []CustodialOrderReport{
					{
						Symbol:           "LTCBTC",
						OrderID:          2,
						ClientOrderID:    "Kk7sqHb9J6mJWTMDVW7Vos",
						TransactTime:     types.Time(time.UnixMilli(1563417480525)),
						OriginalQuantity: 0.624363,
						Status:           "NEW",
						TimeInForce:      TimeInForceGoodTillCancel,
						Type:             OrderTypeStopLoss,
						Side:             SideBuy,
						StopPrice:        0.960664,
					},
					{
						Symbol:           "LTCBTC",
						OrderID:          3,
						ClientOrderID:    "xTXKaGYd4bluPVp78IVRvl",
						TransactTime:     types.Time(time.UnixMilli(1563417480525)),
						Price:            0.036435,
						OriginalQuantity: 0.624363,
						Status:           "NEW",
						TimeInForce:      TimeInForceGoodTillCancel,
						Type:             OrderTypeLimitMaker,
						Side:             SideBuy,
					},
				},
			}
			assert.Equal(t, exp, result, "CustodialNewOneCancelsTheOtherOrder should decode the order list")
		}
	})
}

func TestGetCustodialOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetCustodialOpenOrders(t.Context(), "", currency.EMPTYPAIR)
	require.ErrorIs(t, err, errRailRequired, "GetCustodialOpenOrders must require a rail")

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			symbol currency.Pair
			exp    url.Values
		}{
			{name: "every symbol", exp: url.Values{"rail": {custodialTestRail}}},
			{name: "one symbol", symbol: currency.NewBTCUSD(), exp: url.Values{"rail": {custodialTestRail}, "symbol": {"BTCUSD"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/custodian/openOrders", tc.exp, `[]`)
				_, err := ex.GetCustodialOpenOrders(t.Context(), custodialTestRail, tc.symbol)
				require.NoError(t, err, "GetCustodialOpenOrders must not error")
			})
		}
	})

	t.Run("returns the open orders", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetCustodialOpenOrders(t.Context(), custodialTestRail, currency.NewPair(currency.LTC, currency.BTC))
		require.NoError(t, err, "GetCustodialOpenOrders must not error")
		if mockTests {
			exp := []CustodialOrderResponse{{
				Symbol:           "LTCBTC",
				OrderID:          1,
				OrderListID:      -1,
				ClientOrderID:    "myOrder1",
				Price:            0.1,
				OriginalQuantity: 1,
				Status:           "NEW",
				TimeInForce:      TimeInForceGoodTillCancel,
				Type:             OrderTypeLimit,
				Side:             SideBuy,
				Time:             types.Time(time.UnixMilli(1499827319559)),
				UpdateTime:       types.Time(time.UnixMilli(1499827319559)),
				IsWorking:        true,
				ExpressTradeFlag: true,
			}}
			assert.Equal(t, exp, result, "GetCustodialOpenOrders should decode the open orders")
		}
	})
}

func TestGetCustodialOrder(t *testing.T) {
	t.Parallel()
	ltcbtc := currency.NewPair(currency.LTC, currency.BTC)
	for _, tc := range []struct {
		name string
		req  *CustodialOrderRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no rail", req: &CustodialOrderRequest{Symbol: ltcbtc, OrderID: 1}, err: errRailRequired},
		{name: "no symbol", req: &CustodialOrderRequest{Rail: custodialTestRail, OrderID: 1}, err: currency.ErrCurrencyPairEmpty},
		{name: "no order ID", req: &CustodialOrderRequest{Rail: custodialTestRail, Symbol: ltcbtc}, err: order.ErrOrderIDNotSet},
	} {
		_, err := e.GetCustodialOrder(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "GetCustodialOrder must refuse a request with %s", tc.name)
	}

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/custodian/order", url.Values{"rail": {custodialTestRail}, "symbol": {"BTCUSD"}, "orderId": {"18446744073709551615"}}, `{}`)
		_, err := ex.GetCustodialOrder(t.Context(), &CustodialOrderRequest{Rail: custodialTestRail, Symbol: currency.NewBTCUSD(), OrderID: 18446744073709551615})
		require.NoError(t, err, "GetCustodialOrder must not error")
	})

	t.Run("returns the order", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetCustodialOrder(t.Context(), &CustodialOrderRequest{Rail: custodialTestRail, Symbol: ltcbtc, OrderID: 1})
		require.NoError(t, err, "GetCustodialOrder must not error")
		require.NotNil(t, result, "GetCustodialOrder must return a result")
		if mockTests {
			exp := &CustodialOrderResponse{
				Symbol:                     "LTCBTC",
				OrderID:                    1,
				OrderListID:                -1,
				ClientOrderID:              "myOrder1",
				Price:                      0.1,
				OriginalQuantity:           1,
				Status:                     "NEW",
				TimeInForce:                TimeInForceGoodTillCancel,
				Type:                       OrderTypeLimit,
				Side:                       SideBuy,
				IcebergQuantity:            0.5,
				Time:                       types.Time(time.UnixMilli(1499827319559)),
				UpdateTime:                 types.Time(time.UnixMilli(1499827319559)),
				IsWorking:                  true,
				OriginalQuoteOrderQuantity: 0.1,
				ExpressTradeFlag:           true,
			}
			assert.Equal(t, exp, result, "GetCustodialOrder should decode the order")
		}
	})
}

func TestGetCustodialOrderHistory(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1565776000000), time.UnixMilli(1565777000000)
	for _, tc := range []struct {
		name string
		req  *CustodialOrderHistoryRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no rail", req: &CustodialOrderHistoryRequest{}, err: errRailRequired},
		{name: "a reversed time range", req: &CustodialOrderHistoryRequest{Rail: custodialTestRail, StartTime: end, EndTime: start}, err: common.ErrStartAfterEnd},
	} {
		_, err := e.GetCustodialOrderHistory(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "GetCustodialOrderHistory must refuse a request with %s", tc.name)
	}

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *CustodialOrderHistoryRequest
			exp  url.Values
		}{
			{name: "every symbol", req: &CustodialOrderHistoryRequest{Rail: custodialTestRail}, exp: url.Values{"rail": {custodialTestRail}}},
			{
				name: "every filter",
				req:  &CustodialOrderHistoryRequest{Rail: custodialTestRail, Symbol: currency.NewBTCUSD(), StartTime: start, EndTime: end, FromID: 11, Limit: 50},
				exp:  url.Values{"rail": {custodialTestRail}, "symbol": {"BTCUSD"}, "startTime": {"1565776000000"}, "endTime": {"1565777000000"}, "fromId": {"11"}, "limit": {"50"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/custodian/orderHistory", tc.exp, `[]`)
				_, err := ex.GetCustodialOrderHistory(t.Context(), tc.req)
				require.NoError(t, err, "GetCustodialOrderHistory must not error")
			})
		}
	})

	t.Run("returns the orders", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetCustodialOrderHistory(t.Context(), &CustodialOrderHistoryRequest{Rail: custodialTestRail})
		require.NoError(t, err, "GetCustodialOrderHistory must not error")
		if mockTests {
			exp := []CustodialOrderResponse{{
				Symbol:                  "1565776717369",
				OrderID:                 11,
				OrderListID:             -1,
				ClientOrderID:           "mhUTILjXKS0BHTaBG0is1p",
				OriginalQuantity:        5,
				ExecutedQuantity:        5,
				CumulativeQuoteQuantity: 9.5,
				Status:                  "FILLED",
				TimeInForce:             TimeInForceGoodTillCancel,
				Type:                    OrderTypeTakeProfit,
				Side:                    SideBuy,
				StopPrice:               2,
				Time:                    types.Time(time.UnixMilli(1565776718390)),
				UpdateTime:              types.Time(time.UnixMilli(1565776718779)),
				IsWorking:               true,
			}}
			assert.Equal(t, exp, result, "GetCustodialOrderHistory should decode the orders")
		}
	})
}

func TestGetCustodialTradeHistory(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1662027000000), time.UnixMilli(1662030000000)
	for _, tc := range []struct {
		name string
		req  *CustodialTradeHistoryRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no rail", req: &CustodialTradeHistoryRequest{}, err: errRailRequired},
		{name: "a reversed time range", req: &CustodialTradeHistoryRequest{Rail: custodialTestRail, StartTime: end, EndTime: start}, err: common.ErrStartAfterEnd},
	} {
		_, err := e.GetCustodialTradeHistory(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "GetCustodialTradeHistory must refuse a request with %s", tc.name)
	}

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *CustodialTradeHistoryRequest
			exp  url.Values
		}{
			{name: "every symbol", req: &CustodialTradeHistoryRequest{Rail: custodialTestRail}, exp: url.Values{"rail": {custodialTestRail}}},
			{
				name: "every filter",
				req:  &CustodialTradeHistoryRequest{Rail: custodialTestRail, Symbol: currency.NewBTCUSD(), OrderID: 115, StartTime: start, EndTime: end, FromID: 7, Limit: 50},
				exp:  url.Values{"rail": {custodialTestRail}, "symbol": {"BTCUSD"}, "orderId": {"115"}, "startTime": {"1662027000000"}, "endTime": {"1662030000000"}, "fromId": {"7"}, "limit": {"50"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/custodian/tradeHistory", tc.exp, `[]`)
				_, err := ex.GetCustodialTradeHistory(t.Context(), tc.req)
				require.NoError(t, err, "GetCustodialTradeHistory must not error")
			})
		}
	})

	t.Run("returns the trades", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetCustodialTradeHistory(t.Context(), &CustodialTradeHistoryRequest{Rail: custodialTestRail, Symbol: currency.NewBTCUSD()})
		require.NoError(t, err, "GetCustodialTradeHistory must not error")
		if mockTests {
			exp := []CustodialTradeResponse{
				{
					Symbol:          "BTCUSD",
					Price:           10000,
					Quantity:        1,
					QuoteQuantity:   10000,
					Time:            types.Time(time.UnixMilli(1662027374093)),
					IsBuyer:         true,
					IsBestMatch:     true,
					OrderID:         115,
					OrderListID:     -1,
					Commission:      0.001,
					CommissionAsset: currency.BTC,
				},
				{
					Symbol:          "BTCUSD",
					Price:           5000,
					Quantity:        0.0001,
					QuoteQuantity:   0.5,
					Time:            types.Time(time.UnixMilli(1662029127602)),
					IsMaker:         true,
					IsBestMatch:     true,
					OrderID:         111,
					OrderListID:     -1,
					CommissionAsset: currency.USD,
				},
			}
			assert.Equal(t, exp, result, "GetCustodialTradeHistory should decode the trades")
		}
	})
}

func TestCancelCustodialOrder(t *testing.T) {
	t.Parallel()
	ltcbtc := currency.NewPair(currency.LTC, currency.BTC)
	for _, tc := range []struct {
		name string
		req  *CancelCustodialOrderRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no rail", req: &CancelCustodialOrderRequest{Symbol: ltcbtc, OrderID: 4}, err: errRailRequired},
		{name: "no symbol", req: &CancelCustodialOrderRequest{Rail: custodialTestRail, OrderID: 4}, err: currency.ErrCurrencyPairEmpty},
		{name: "neither order ID", req: &CancelCustodialOrderRequest{Rail: custodialTestRail, Symbol: ltcbtc, NewClientOrderID: "cancelMyOrder1"}, err: order.ErrOrderIDNotSet},
	} {
		_, err := e.CancelCustodialOrder(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "CancelCustodialOrder must refuse a request with %s", tc.name)
	}

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *CancelCustodialOrderRequest
			exp  url.Values
		}{
			{
				name: "order ID",
				req:  &CancelCustodialOrderRequest{Rail: custodialTestRail, Symbol: currency.NewBTCUSD(), OrderID: 4},
				exp:  url.Values{"rail": {custodialTestRail}, "symbol": {"BTCUSD"}, "orderId": {"4"}},
			},
			{
				name: "client order ID with a new client order ID",
				req:  &CancelCustodialOrderRequest{Rail: custodialTestRail, Symbol: currency.NewBTCUSD(), OriginalClientOrderID: "myOrder1", NewClientOrderID: "cancelMyOrder1"},
				exp:  url.Values{"rail": {custodialTestRail}, "symbol": {"BTCUSD"}, "origClientOrderId": {"myOrder1"}, "newClientOrderId": {"cancelMyOrder1"}},
			},
			{
				name: "both order IDs",
				req:  &CancelCustodialOrderRequest{Rail: custodialTestRail, Symbol: currency.NewBTCUSD(), OrderID: 4, OriginalClientOrderID: "myOrder1"},
				exp:  url.Values{"rail": {custodialTestRail}, "symbol": {"BTCUSD"}, "orderId": {"4"}, "origClientOrderId": {"myOrder1"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodDelete, "/sapi/v1/custodian/cancelOrder", tc.exp, `{}`)
				_, err := ex.CancelCustodialOrder(t.Context(), tc.req)
				require.NoError(t, err, "CancelCustodialOrder must not error")
			})
		}
	})

	t.Run("cancels the order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.CancelCustodialOrder(t.Context(), &CancelCustodialOrderRequest{Rail: custodialTestRail, Symbol: ltcbtc, OrderID: 4, NewClientOrderID: "cancelMyOrder1"})
		require.NoError(t, err, "CancelCustodialOrder must not error")
		require.NotNil(t, result, "CancelCustodialOrder must return a result")
		if mockTests {
			exp := &CancelCustodialOrderResponse{
				Symbol:                  "LTCBTC",
				OriginalClientOrderID:   "myOrder1",
				OrderID:                 4,
				OrderListID:             -1,
				ClientOrderID:           "cancelMyOrder1",
				Price:                   2,
				OriginalQuantity:        1,
				ExecutedQuantity:        0.5,
				CumulativeQuoteQuantity: 1,
				Status:                  "CANCELED",
				TimeInForce:             TimeInForceGoodTillCancel,
				Type:                    OrderTypeLimit,
				Side:                    SideBuy,
			}
			assert.Equal(t, exp, result, "CancelCustodialOrder should decode the cancelled order")
		}
	})
}

func TestCancelCustodialOpenOrdersForSymbol(t *testing.T) {
	t.Parallel()
	_, err := e.CancelCustodialOpenOrdersForSymbol(t.Context(), "", currency.NewBTCUSDT())
	require.ErrorIs(t, err, errRailRequired, "CancelCustodialOpenOrdersForSymbol must require a rail")
	_, err = e.CancelCustodialOpenOrdersForSymbol(t.Context(), custodialTestRail, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelCustodialOpenOrdersForSymbol must require a symbol")

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodDelete, "/sapi/v1/custodian/cancelOrdersBySymbol", url.Values{"rail": {custodialTestRail}, "symbol": {"BTCUSD"}}, `[]`)
		_, err := ex.CancelCustodialOpenOrdersForSymbol(t.Context(), custodialTestRail, currency.NewBTCUSD())
		require.NoError(t, err, "CancelCustodialOpenOrdersForSymbol must not error")
	})

	t.Run("cancels the open orders", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.CancelCustodialOpenOrdersForSymbol(t.Context(), custodialTestRail, currency.NewBTCUSDT())
		require.NoError(t, err, "CancelCustodialOpenOrdersForSymbol must not error")
		if mockTests {
			exp := []CancelCustodialOrderResponse{{
				Symbol:                "BTCUSDT",
				OriginalClientOrderID: "KZJijIsAFf5BU5oxkWTAU3",
				OrderListID:           -1,
				ClientOrderID:         "8epSIUMvNCknntXcSzWs7H",
				Price:                 0.1,
				OriginalQuantity:      1,
				Status:                "CANCELED",
				TimeInForce:           TimeInForceGoodTillCancel,
				Type:                  OrderTypeLimit,
				Side:                  SideBuy,
			}}
			assert.Equal(t, exp, result, "CancelCustodialOpenOrdersForSymbol should decode the cancelled orders")
		}
	})
}

func TestCancelCustodialOneCancelsTheOtherOrder(t *testing.T) {
	t.Parallel()
	ltcbtc := currency.NewPair(currency.LTC, currency.BTC)
	for _, tc := range []struct {
		name string
		req  *CancelCustodialOneCancelsTheOtherOrderRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no rail", req: &CancelCustodialOneCancelsTheOtherOrderRequest{Symbol: ltcbtc, OrderListID: new(uint64(1))}, err: errRailRequired},
		{name: "no symbol", req: &CancelCustodialOneCancelsTheOtherOrderRequest{Rail: custodialTestRail, OrderListID: new(uint64(1))}, err: currency.ErrCurrencyPairEmpty},
		{name: "a list client order ID instead of an order list ID", req: &CancelCustodialOneCancelsTheOtherOrderRequest{Rail: custodialTestRail, Symbol: ltcbtc, ListClientOrderID: "C3wyj4WVEktd7u9aVBRXcN"}, err: order.ErrOrderIDNotSet},
	} {
		_, err := e.CancelCustodialOneCancelsTheOtherOrder(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "CancelCustodialOneCancelsTheOtherOrder must refuse a request with %s", tc.name)
	}

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *CancelCustodialOneCancelsTheOtherOrderRequest
			exp  url.Values
		}{
			{
				name: "order list ID",
				req:  &CancelCustodialOneCancelsTheOtherOrderRequest{Rail: custodialTestRail, Symbol: currency.NewBTCUSD(), OrderListID: new(uint64(6))},
				exp:  url.Values{"rail": {custodialTestRail}, "symbol": {"BTCUSD"}, "orderListId": {"6"}},
			},
			{
				name: "order list ID 0",
				req:  &CancelCustodialOneCancelsTheOtherOrderRequest{Rail: custodialTestRail, Symbol: currency.NewBTCUSD(), OrderListID: new(uint64(0))},
				exp:  url.Values{"rail": {custodialTestRail}, "symbol": {"BTCUSD"}, "orderListId": {"0"}},
			},
			{
				name: "every parameter",
				req:  &CancelCustodialOneCancelsTheOtherOrderRequest{Rail: custodialTestRail, Symbol: currency.NewBTCUSD(), OrderListID: new(uint64(6)), ListClientOrderID: "C3wyj4WVEktd7u9aVBRXcN", NewClientOrderID: "unfWT8ig8i0uj6lPuYLez6"},
				exp:  url.Values{"rail": {custodialTestRail}, "symbol": {"BTCUSD"}, "orderListId": {"6"}, "listClientOrderId": {"C3wyj4WVEktd7u9aVBRXcN"}, "newClientOrderId": {"unfWT8ig8i0uj6lPuYLez6"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodDelete, "/sapi/v1/custodian/cancelOcoOrder", tc.exp, `{}`)
				_, err := ex.CancelCustodialOneCancelsTheOtherOrder(t.Context(), tc.req)
				require.NoError(t, err, "CancelCustodialOneCancelsTheOtherOrder must not error")
			})
		}
	})

	t.Run("cancels the order list", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.CancelCustodialOneCancelsTheOtherOrder(t.Context(), &CancelCustodialOneCancelsTheOtherOrderRequest{Rail: custodialTestRail, Symbol: ltcbtc, OrderListID: new(uint64(1)), NewClientOrderID: "unfWT8ig8i0uj6lPuYLez6"})
		require.NoError(t, err, "CancelCustodialOneCancelsTheOtherOrder must not error")
		require.NotNil(t, result, "CancelCustodialOneCancelsTheOtherOrder must return a result")
		if mockTests {
			exp := &CustodialOneCancelsTheOtherOrderResponse{
				OrderListID:       1,
				ContingencyType:   "OCO",
				ListStatusType:    "ALL_DONE",
				ListOrderStatus:   "ALL_DONE",
				ListClientOrderID: "C3wyj4WVEktd7u9aVBRXcN",
				TransactionTime:   types.Time(time.UnixMilli(1574040868128)),
				Symbol:            "LTCBTC",
				Orders: []CustodialOrderListOrder{
					{Symbol: "LTCBTC", OrderID: 2, ClientOrderID: "pO9ufTiFGg3nw2fOdgeOXa"},
					{Symbol: "LTCBTC", OrderID: 3, ClientOrderID: "TXOvglzXuaubXAaENpaRCB"},
				},
				OrderReports: []CustodialOrderReport{
					{
						Symbol:                "LTCBTC",
						OriginalClientOrderID: "pO9ufTiFGg3nw2fOdgeOXa",
						OrderID:               2,
						OrderListID:           1,
						ClientOrderID:         "unfWT8ig8i0uj6lPuYLez6",
						Price:                 1,
						OriginalQuantity:      10,
						Status:                "CANCELED",
						TimeInForce:           TimeInForceGoodTillCancel,
						Type:                  OrderTypeStopLossLimit,
						Side:                  SideSell,
						StopPrice:             1,
					},
					{
						Symbol:                  "LTCBTC",
						OriginalClientOrderID:   "TXOvglzXuaubXAaENpaRCB",
						OrderID:                 3,
						OrderListID:             1,
						ClientOrderID:           "unfWT8ig8i0uj6lPuYLez6",
						Price:                   3,
						OriginalQuantity:        10,
						ExecutedQuantity:        5,
						CumulativeQuoteQuantity: 15,
						Status:                  "CANCELED",
						TimeInForce:             TimeInForceGoodTillCancel,
						Type:                    OrderTypeLimitMaker,
						Side:                    SideSell,
					},
				},
			}
			assert.Equal(t, exp, result, "CancelCustodialOneCancelsTheOtherOrder should decode the cancelled order list")
		}
	})
}

func TestGetCustodialSettlementSettings(t *testing.T) {
	t.Parallel()
	_, err := e.GetCustodialSettlementSettings(t.Context(), "")
	require.ErrorIs(t, err, errRailRequired, "GetCustodialSettlementSettings must require a rail")

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/custodian/settlementSetting", url.Values{"rail": {custodialTestRail}}, `{}`)
		_, err := ex.GetCustodialSettlementSettings(t.Context(), custodialTestRail)
		require.NoError(t, err, "GetCustodialSettlementSettings must not error")
	})

	t.Run("returns the settlement settings", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetCustodialSettlementSettings(t.Context(), custodialTestRail)
		require.NoError(t, err, "GetCustodialSettlementSettings must not error")
		require.NotNil(t, result, "GetCustodialSettlementSettings must return a result")
		if mockTests {
			exp := &CustodialSettlementSettingsResponse{SettlementActive: true, FrequencyInHours: 24, NextTriggerTime: types.Time(time.UnixMilli(1646524800000))}
			assert.Equal(t, exp, result, "GetCustodialSettlementSettings should decode the settlement settings")
		}
	})
}

func TestGetCustodialSettlementHistory(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1658981047000), time.UnixMilli(1660074031000)
	for _, tc := range []struct {
		name string
		req  *CustodialSettlementHistoryRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no rail", req: &CustodialSettlementHistoryRequest{}, err: errRailRequired},
		{name: "a reversed time range", req: &CustodialSettlementHistoryRequest{Rail: custodialTestRail, StartTime: end, EndTime: start}, err: common.ErrStartAfterEnd},
	} {
		_, err := e.GetCustodialSettlementHistory(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "GetCustodialSettlementHistory must refuse a request with %s", tc.name)
	}

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *CustodialSettlementHistoryRequest
			exp  url.Values
		}{
			{name: "rail only", req: &CustodialSettlementHistoryRequest{Rail: custodialTestRail}, exp: url.Values{"rail": {custodialTestRail}}},
			{
				name: "every filter",
				req:  &CustodialSettlementHistoryRequest{Rail: custodialTestRail, StartTime: start, EndTime: end, Limit: 100, Page: 2},
				exp:  url.Values{"rail": {custodialTestRail}, "startTime": {"1658981047000"}, "endTime": {"1660074031000"}, "limit": {"100"}, "page": {"2"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/custodian/settlementHistory", tc.exp, `{}`)
				_, err := ex.GetCustodialSettlementHistory(t.Context(), tc.req)
				require.NoError(t, err, "GetCustodialSettlementHistory must not error")
			})
		}
	})

	t.Run("returns the settlements", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetCustodialSettlementHistory(t.Context(), &CustodialSettlementHistoryRequest{Rail: custodialTestRail})
		require.NoError(t, err, "GetCustodialSettlementHistory must not error")
		require.NotNil(t, result, "GetCustodialSettlementHistory must return a result")
		if mockTests {
			exp := &CustodialSettlementHistoryResponse{
				Records: []CustodialSettlementRecord{
					{
						Status:       "PROCESS",
						TriggerTime:  types.Time(time.UnixMilli(1660074030009)),
						SettlementID: "1a22baf2329845b997eff2bc9a700504",
						SettlementAssets: []CustodialSettlementAsset{
							{Asset: currency.NewCode("USDC_T"), SettleTo: "0xc80e62a4Bb86c42D1a0Cda3dCBcCb26830D92F2B", Network: "ETH", Amount: 10},
						},
					},
					{Status: "FAILURE", TriggerTime: types.Time(time.UnixMilli(1659081889724)), SettlementID: "d9e69bb7386f4ebaa1689559b549ef25", SettlementAssets: []CustodialSettlementAsset{}},
					{Status: "FAILURE", TriggerTime: types.Time(time.UnixMilli(1659079319905)), SettlementID: "58ca799c596f43a792e8add27dc42d56", SettlementAssets: []CustodialSettlementAsset{}},
					{Status: "FAILURE", TriggerTime: types.Time(time.UnixMilli(1658981047109)), SettlementID: "3ab30503bf6d43bc8ea8112f6f6b2c91", SettlementAssets: []CustodialSettlementAsset{}},
				},
				Total: 4,
			}
			assert.Equal(t, exp, result, "GetCustodialSettlementHistory should decode the settlements")
		}
	})
}

func TestCustodialParams(t *testing.T) {
	t.Parallel()
	_, err := custodialParams("")
	require.ErrorIs(t, err, errRailRequired, "custodialParams must require a rail")
	params, err := custodialParams(custodialTestRail)
	require.NoError(t, err, "custodialParams must not error")
	assert.Equal(t, url.Values{"rail": {custodialTestRail}}, params, "custodialParams should set the rail")
}

func TestCustodialAssetTransferParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CustodialAssetTransferRequest
		exp  url.Values
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no rail", req: &CustodialAssetTransferRequest{Asset: currency.BTC, Amount: 1}, err: errRailRequired},
		{name: "no asset", req: &CustodialAssetTransferRequest{Rail: custodialTestRail, Amount: 1}, err: currency.ErrCurrencyCodeEmpty},
		{name: "no amount", req: &CustodialAssetTransferRequest{Rail: custodialTestRail, Asset: currency.BTC}, err: order.ErrAmountIsInvalid},
		{name: "negative amount", req: &CustodialAssetTransferRequest{Rail: custodialTestRail, Asset: currency.BTC, Amount: -1}, err: order.ErrAmountIsInvalid},
		{
			name: "generated client order ID",
			req:  &CustodialAssetTransferRequest{Rail: custodialTestRail, Asset: currency.NewCode("btc"), Amount: 0.00012345},
			exp:  url.Values{"rail": {custodialTestRail}, "asset": {"BTC"}, "amount": {"0.00012345"}},
		},
		{
			name: "client order ID",
			req:  &CustodialAssetTransferRequest{Rail: custodialTestRail, Asset: currency.BTC, Amount: 1, ClientOrderID: "1663752208086"},
			exp:  url.Values{"rail": {custodialTestRail}, "asset": {"BTC"}, "amount": {"1"}, "clientOrderId": {"1663752208086"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := custodialAssetTransferParams(tc.req)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "custodialAssetTransferParams must refuse the request")
				return
			}
			require.NoError(t, err, "custodialAssetTransferParams must not error")
			assert.Equal(t, tc.exp, params, "custodialAssetTransferParams should set the transfer's parameters")
		})
	}
}
