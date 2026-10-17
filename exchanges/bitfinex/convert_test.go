package bitfinex

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	// testDerivativePair is a USDT margined perpetual contract, and testCoinMarginedPair one margined by bitcoin
	testDerivativePair   = currency.NewPair(currency.NewCode("BTCF0"), currency.NewCode("USTF0"))
	testCoinMarginedPair = currency.NewPair(currency.NewCode("ETHF0"), currency.NewCode("BTCF0"))
)

func TestSymbolFor(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a   asset.Item
		p   currency.Pair
		exp string
	}{
		{a: asset.Spot, p: currency.NewBTCUSD(), exp: "tBTCUSD"},
		{a: asset.Margin, p: currency.NewPairWithDelimiter("ada", "btc", ":"), exp: "tADABTC"},
		{a: asset.Spot, p: currency.NewPair(currency.NewCode("AVAX"), currency.USD), exp: "tAVAX:USD"},
		{a: asset.Spot, p: currency.NewPair(currency.USD, currency.NewCode("TESTUSDT")), exp: "tUSD:TESTUSDT"},
		{a: asset.USDTMarginedFutures, p: testDerivativePair, exp: "tBTCF0:USTF0"},
		{a: asset.CoinMarginedFutures, p: testCoinMarginedPair, exp: "tETHF0:BTCF0"},
		{a: asset.CoinMarginedFutures, p: currency.NewPairWithDelimiter("ETHF0", "btcf0", "-TEST-DELIM-"), exp: "tETHF0:BTCF0"},
		{a: asset.MarginFunding, p: testFundingUSD, exp: "fUSD"},
		{a: asset.MarginFunding, p: currency.NewPairWithDelimiter("avax", "", "-"), exp: "fAVAX"},
	} {
		got, err := symbolFor(tc.a, tc.p)
		require.NoErrorf(t, err, "symbolFor must not error for %s %s", tc.a, tc.p)
		assert.Equalf(t, tc.exp, got, "symbolFor should give %s %s's symbol", tc.a, tc.p)
	}
	_, err := symbolFor(asset.Options, currency.NewBTCUSD())
	assert.ErrorIs(t, err, asset.ErrNotSupported, "symbolFor should refuse an asset Bitfinex has no symbol for")
	for _, tc := range []struct {
		a asset.Item
		p currency.Pair
	}{
		{a: asset.Spot, p: testDerivativePair},
		{a: asset.Margin, p: testCoinMarginedPair},
		{a: asset.USDTMarginedFutures, p: currency.NewBTCUSD()},
		{a: asset.USDTMarginedFutures, p: testCoinMarginedPair},
		{a: asset.CoinMarginedFutures, p: testDerivativePair},
		{a: asset.CoinMarginedFutures, p: currency.NewPair(currency.NewCode("ETHF0"), currency.BTC)},
	} {
		_, err := symbolFor(tc.a, tc.p)
		assert.ErrorIsf(t, err, errPairNotOfAsset, "symbolFor should refuse %s as %s, rather than send it to another asset's market", tc.p, tc.a)
	}
}

func TestDerivativeAsset(t *testing.T) {
	t.Parallel()
	for collateral, exp := range map[string]asset.Item{
		"USTF0":      asset.USDTMarginedFutures,
		"ustf0":      asset.USDTMarginedFutures,
		"TESTUSDTF0": asset.USDTMarginedFutures,
		"BTCF0":      asset.CoinMarginedFutures,
		"TESTBTCF0":  asset.CoinMarginedFutures,
	} {
		assert.Equalf(t, exp, derivativeAsset(currency.NewCode(collateral)), "derivativeAsset should give the asset %s margins", collateral)
	}
	assert.True(t, isDerivativeAsset(asset.USDTMarginedFutures), "USDT margined futures should be a derivative asset")
	assert.True(t, isDerivativeAsset(asset.CoinMarginedFutures), "coin margined futures should be a derivative asset")
	assert.False(t, isDerivativeAsset(asset.Futures), "futures should not be a derivative asset, as Bitfinex's derivatives trade by their collateral")
	assert.Equal(t, currency.BTC, underlyingCurrency(currency.NewCode("btcf0")), "a derivative currency should follow the currency without its suffix")
}

func TestPairFromSymbol(t *testing.T) {
	t.Parallel()
	for symbol, exp := range map[string]currency.Pair{
		"tBTCUSD":      currency.NewBTCUSD(),
		"BTCUSD":       currency.NewBTCUSD(),
		"tAVAX:USD":    currency.NewPair(currency.NewCode("AVAX"), currency.USD),
		"tBTCF0:USTF0": testDerivativePair,
		"fUSD":         testFundingUSD,
	} {
		got, err := pairFromSymbol(symbol)
		require.NoErrorf(t, err, "pairFromSymbol must not error for %s", symbol)
		assert.Truef(t, exp.Equal(got), "pairFromSymbol should return %s for %s, got %s", exp, symbol, got)
	}
	for _, symbol := range []string{"tBTC", "tBTCUSDT", "t:USD", "tBTC:", ""} {
		_, err := pairFromSymbol(symbol)
		assert.ErrorIsf(t, err, errSymbolUnparsable, "pairFromSymbol should refuse %q", symbol)
	}
}

func TestOrderAsset(t *testing.T) {
	t.Parallel()
	assert.Equal(t, asset.USDTMarginedFutures, orderAsset(OrderTypeLimit, testDerivativePair), "a USDT margined derivative pair's order should trade USDT margined futures")
	assert.Equal(t, asset.CoinMarginedFutures, orderAsset(OrderTypeLimit, testCoinMarginedPair), "a coin margined derivative pair's order should trade coin margined futures")
	assert.Equal(t, asset.Spot, orderAsset(ExchangeOrderTypePrefix+OrderTypeLimit, currency.NewBTCUSD()), "an EXCHANGE order should trade spot")
	assert.Equal(t, asset.Margin, orderAsset(OrderTypeLimit, currency.NewBTCUSD()), "any other order should trade margin")
	assert.False(t, isDerivativePair(currency.NewPair(currency.NewCode("BTCF0"), currency.USD)), "a pair with one F0 currency should not be a derivative")
}

func TestOrderTypeFromBitfinex(t *testing.T) {
	t.Parallel()
	for orderType, exp := range map[string]struct {
		t   order.Type
		tif order.TimeInForce
	}{
		OrderTypeLimit: {t: order.Limit},
		ExchangeOrderTypePrefix + OrderTypeMarket: {t: order.Market},
		OrderTypeStop: {t: order.Stop},
		ExchangeOrderTypePrefix + OrderTypeStopLimit:      {t: order.StopLimit},
		OrderTypeTrailingStop:                             {t: order.TrailingStop},
		ExchangeOrderTypePrefix + OrderTypeFillOrKill:     {t: order.Limit, tif: order.FillOrKill},
		OrderTypeImmediateOrCancel:                        {t: order.Limit, tif: order.ImmediateOrCancel},
		ExchangeOrderTypePrefix + OrderTypeTrailingStop:   {t: order.TrailingStop},
		ExchangeOrderTypePrefix + OrderTypeLimit + "    ": {t: order.UnknownType},
	} {
		gotType, gotTIF, err := orderTypeFromBitfinex(orderType)
		if exp.t == order.UnknownType {
			assert.ErrorIsf(t, err, errUnknownOrderType, "orderTypeFromBitfinex should refuse %q", orderType)
			continue
		}
		require.NoErrorf(t, err, "orderTypeFromBitfinex must not error for %s", orderType)
		assert.Equalf(t, exp.t, gotType, "orderTypeFromBitfinex should give %s's type", orderType)
		assert.Equalf(t, exp.tif, gotTIF, "orderTypeFromBitfinex should give %s's time in force", orderType)
	}
}

func TestOrderStatusFromBitfinex(t *testing.T) {
	t.Parallel()
	for status, exp := range map[string]order.Status{
		"ACTIVE":                         order.Active,
		"EXECUTED @ 107.6(-0.2)":         order.Filled,
		"PARTIALLY FILLED @ 107.6(-0.1)": order.PartiallyFilled,
		"CANCELED":                       order.Cancelled,
		"CANCELED was: PARTIALLY FILLED @ 107.6(-0.1)":      order.PartiallyFilledCancelled,
		"POSTONLY CANCELED":                                 order.Cancelled,
		"FILLORKILL CANCELED":                               order.Cancelled,
		"IOC CANCELED was: PARTIALLY FILLED @ 1(-1)":        order.PartiallyFilledCancelled,
		"INSUFFICIENT MARGIN was: PARTIALLY FILLED @ 1(-1)": order.InsufficientBalance,
		"INSUFFICIENT BALANCE (U1)":                         order.InsufficientBalance,
		"RSN_DUST":                                          order.Cancelled,
		"RSN_PAUSE":                                         order.Cancelled,
		"FORCED EXECUTED @ 1(-1)":                           order.Liquidated,
		"NOVEL":                                             order.UnknownStatus,
	} {
		assert.Equalf(t, exp, orderStatusFromBitfinex(status), "orderStatusFromBitfinex should give %q's status", status)
	}
}

func TestOrderSideFromAmount(t *testing.T) {
	t.Parallel()
	side, err := orderSideFromAmount(1)
	require.NoError(t, err, "orderSideFromAmount must not error for a positive amount")
	assert.Equal(t, order.Buy, side, "a positive amount should buy")
	side, err = orderSideFromAmount(-1)
	require.NoError(t, err, "orderSideFromAmount must not error for a negative amount")
	assert.Equal(t, order.Sell, side, "a negative amount should sell")
	_, err = orderSideFromAmount(0)
	assert.ErrorIs(t, err, errOrderSideUnresolved, "a zero amount should have no side")
}

func TestOrderDetail(t *testing.T) {
	t.Parallel()
	got, err := e.orderDetail(&Order{
		ID: 1, ClientOrderID: 2, Symbol: "tBTCUSD", CreatedAt: types.Time(time.UnixMilli(1700000000001)), UpdatedAt: types.Time(time.UnixMilli(1700000000002)),
		Amount: -0.25, OriginalAmount: -1, Type: ExchangeOrderTypePrefix + OrderTypeStopLimit, TimeInForce: types.Time(time.UnixMilli(1700000000003)),
		Flags: OrderFlagHidden | OrderFlagReduceOnly | OrderFlagPostOnly, Status: "PARTIALLY FILLED @ 100(-0.75)", Price: 110, AveragePrice: 100,
		AuxiliaryLimitPrice: 105, Meta: json.RawMessage(`{"$F33":5}`),
	})
	require.NoError(t, err, "orderDetail must not error")
	assert.Equal(t, &order.Detail{
		Exchange: e.Name, OrderID: "1", ClientOrderID: "2", Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Side: order.Sell, Type: order.StopLimit,
		TimeInForce: order.PostOnly | order.GoodTillTime, Status: order.PartiallyFilled, Price: 105, TriggerPrice: 110, Amount: 1, RemainingAmount: 0.25,
		ExecutedAmount: 0.75, AverageExecutedPrice: 100, HiddenOrder: true, ReduceOnly: true, Leverage: 5, Date: time.UnixMilli(1700000000001),
		LastUpdated: time.UnixMilli(1700000000002),
	}, got, "orderDetail should convert every field, a stop limit's price being its trigger")

	stop, err := e.orderDetail(&Order{ID: 1, Symbol: "tBTCUSD", Amount: 1, OriginalAmount: 1, Type: OrderTypeStop, Price: 90, Hidden: true})
	require.NoError(t, err, "orderDetail must not error for a stop order")
	assert.Equal(t, 90.0, stop.TriggerPrice, "a stop order's price should be its trigger")
	assert.Zero(t, stop.Price, "a stop order should trade at the market once triggered")
	assert.True(t, stop.HiddenOrder, "an order Bitfinex reports hidden should be hidden")

	for _, tc := range []struct {
		o   *Order
		err error
	}{
		{o: &Order{Symbol: "tBTC", Type: OrderTypeLimit, OriginalAmount: 1}, err: errSymbolUnparsable},
		{o: &Order{Symbol: "tBTCUSD", Type: "NOVEL", OriginalAmount: 1}, err: errUnknownOrderType},
		{o: &Order{Symbol: "tBTCUSD", Type: OrderTypeLimit}, err: errOrderSideUnresolved},
	} {
		_, err := e.orderDetail(tc.o)
		assert.ErrorIsf(t, err, tc.err, "orderDetail should return the expected error for %+v", tc.o)
	}
}

func TestWalletAsset(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		walletType string
		c          currency.Code
		exp        asset.Item
	}{
		{walletType: WalletTypeExchange, c: currency.UST, exp: asset.Spot},
		{walletType: WalletTypeMargin, c: currency.USD, exp: asset.Margin},
		{walletType: "trading", c: currency.BTC, exp: asset.Margin},
		{walletType: WalletTypeMargin, c: currency.NewCode("USTF0"), exp: asset.USDTMarginedFutures},
		{walletType: "trading", c: currency.NewCode("BTCF0"), exp: asset.CoinMarginedFutures},
		{walletType: WalletTypeFunding, c: currency.USD, exp: asset.MarginFunding},
		{walletType: "deposit", c: currency.BTC, exp: asset.MarginFunding},
	} {
		got, err := walletAsset(tc.walletType, tc.c)
		require.NoErrorf(t, err, "walletAsset must not error for %s %s", tc.walletType, tc.c)
		assert.Equalf(t, tc.exp, got, "walletAsset should give the asset %s in the %s wallet funds", tc.c, tc.walletType)
	}
	_, err := walletAsset("contribution", currency.USD)
	assert.ErrorIs(t, err, errUnknownWalletType, "walletAsset should refuse an unknown wallet type")
}
