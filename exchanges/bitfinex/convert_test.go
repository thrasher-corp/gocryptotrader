package bitfinex

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
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
		"ACTIVE (note:POSCLOSE)":         order.Active,
		"ACTIVE (note: POSCLOSE)":        order.Active,
		"EXECUTED @ 107.6(-0.2)":         order.Filled,
		"PARTIALLY FILLED @ 107.6(-0.1)": order.PartiallyFilled,
		"CANCELED":                       order.Cancelled,
		"CANCELED was: PARTIALLY FILLED @ 107.6(-0.1)":      order.PartiallyFilledCancelled,
		"POSTONLY CANCELED":                                 order.Cancelled,
		"FILLORKILL CANCELED":                               order.Cancelled,
		"IOC CANCELED was: PARTIALLY FILLED @ 1(-1)":        order.PartiallyFilledCancelled,
		"INSUFFICIENT MARGIN was: PARTIALLY FILLED @ 1(-1)": order.InsufficientBalance,
		"EXECUTED @ 140.0(0.02) was ACTIVE (note:POSCLOSE)": order.Filled,
		"RSN_DUST was: ACTIVE (note: POSCLOSE)":             order.Cancelled,
		"INSUFFICIENT BALANCE (U1)":                         order.InsufficientBalance,
		"RSN_DUST":                                          order.Cancelled,
		"RSN_PAUSE":                                         order.Cancelled,
		"FORCED EXECUTED @ 1(-1)":                           order.Liquidated,
		"NOVEL":                                             order.UnknownStatus,
		// An order closing a position keeps its note ahead of its fills, while open and once closed
		"ACTIVE (note:POSCLOSE), PARTIALLY FILLED @ 0.008049(-0.41291886)":                                    order.PartiallyFilled,
		"ACTIVE (note: POSCLOSE), PARTIALLY FILLED @ 0.008049(-0.2), PARTIALLY FILLED @ 0.00805(-0.21291886)": order.PartiallyFilled,
		"INSUFFICIENT BALANCE (G1) was: ACTIVE (note:POSCLOSE), PARTIALLY FILLED @ 0.008049(-0.41291886)":     order.InsufficientBalance,
		"CANCELED was: ACTIVE (note:POSCLOSE), PARTIALLY FILLED @ 0.008049(-0.2)":                             order.PartiallyFilledCancelled,
		// Bitfinex cuts a status listing many fills short, keeping its end, and earlier cut one without marking it
		"... 2365.3(53.94146517), PARTIALLY FILLED @ 2365.3(45.58642898), PARTIALLY FILLED @ 2365.3(100.0)":    order.PartiallyFilled,
		"...LLED @ 2365.3(45.58642898), PARTIALLY FILLED @ 2365.3(100.0)":                                      order.PartiallyFilled,
		"CANCELED was: ...LLED @ 2365.3(45.58642898), PARTIALLY FILLED @ 2365.3(100.0)":                        order.PartiallyFilledCancelled,
		"EXECUTED @ 2365.3(594.14534185): was ...LLED @ 2365.3(45.58642898), PARTIALLY FILLED @ 2365.3(100.0)": order.Filled,
		"(-41.37139577), PARTIALLY FILLED @ 12.58(-15.53), PARTIALLY FILLED @ 12.58(-41.34952975)":             order.PartiallyFilled,
		"LLED @ 12.58(-15.53), PARTIALLY FILLED @ 12.58(-41.34952975)":                                         order.PartiallyFilled,
		"CANCELED was:  12.58(-15.53)":  order.PartiallyFilledCancelled,
		"...LLED @ 2365.3(45.58642898)": order.UnknownStatus,
	} {
		assert.Equalf(t, exp, orderStatusFromBitfinex(status), "orderStatusFromBitfinex should give %q's status", status)
	}
}

func TestClosedOrderStatus(t *testing.T) {
	t.Parallel()
	for status, exp := range map[order.Status]order.Status{
		order.PartiallyFilled:          order.PartiallyFilledCancelled,
		order.PartiallyFilledCancelled: order.PartiallyFilledCancelled,
		order.Filled:                   order.Filled,
		order.Cancelled:                order.Cancelled,
		order.Active:                   order.Active,
		order.InsufficientBalance:      order.InsufficientBalance,
	} {
		assert.Equalf(t, exp, closedOrderStatus(status), "closedOrderStatus should report a closed %s order %s", status, exp)
	}
}

func TestOrderStatusExecutedAmount(t *testing.T) {
	t.Parallel()
	for status, exp := range map[string]float64{
		"ACTIVE":                 0,
		"EXECUTED @ 107.6(-0.2)": 0.2,
		"EXECUTED @ 916.2(0.78): was PARTIALLY FILLED @ 915.9(0.22)":                     1,
		"PARTIALLY FILLED @ 100.0(0.1), PARTIALLY FILLED @ 101.0(0.2)":                   0.3,
		"CANCELED was: PARTIALLY FILLED @ 0.0218(-0.3), PARTIALLY FILLED @ 0.0218(-0.2)": 0.5,
		"INSUFFICIENT BALANCE (G1) was: PARTIALLY FILLED @ 1.0(0.5)":                     0.5,
		"EXECUTED @ 9500.0(-0.5) was ACTIVE (note:POSCLOSE)":                             0.5,
		"PARTIALLY FILLED @ 1.2e3(1e-7)":                                                 0.0000001,
		"RSN_DUST (amount is less than 0.00000001)":                                      0,
		"CANCELED": 0,
		"PARTIALLY FILLED @ x(0.1), PARTIALLY FILLED @ 100(y), PARTIALLY FILLED @ 100(0.1e-2147483647)": 0,
		"... 2365.3(53.94146517), PARTIALLY FILLED @ 2365.3(45.58642898)":                               45.58642898,
	} {
		got := orderStatusExecutedAmount(status)
		assert.Truef(t, decimal.MustFromFloat(exp).Equal(got), "orderStatusExecutedAmount should add up the fills %q lists in full to %v exactly, got %s", status, exp, got)
	}
}

func TestSplitOrderStatus(t *testing.T) {
	t.Parallel()
	for status, exp := range map[string][2]string{
		"ACTIVE":                         {"ACTIVE", ""},
		"PARTIALLY FILLED @ 107.6(-0.1)": {"PARTIALLY FILLED @ 107.6(-0.1)", ""},
		"CANCELED was: PARTIALLY FILLED @ 107.6(-0.1)":               {"CANCELED", "PARTIALLY FILLED @ 107.6(-0.1)"},
		"EXECUTED @ 916.2(0.78): was PARTIALLY FILLED @ 915.9(0.22)": {"EXECUTED @ 916.2(0.78)", "PARTIALLY FILLED @ 915.9(0.22)"},
		"EXECUTED @ 9500.0(-0.5) was ACTIVE (note:POSCLOSE)":         {"EXECUTED @ 9500.0(-0.5)", "ACTIVE (note:POSCLOSE)"},
		"CANCELED was:  12.58(-15.53)":                               {"CANCELED", " 12.58(-15.53)"},
	} {
		current, previous := splitOrderStatus(status)
		assert.Equalf(t, exp, [2]string{current, previous}, "splitOrderStatus should split %q into what the order is and what it was", status)
	}
}

func TestIsFillListCut(t *testing.T) {
	t.Parallel()
	for fills, exp := range map[string]bool{
		"":                          false,
		"ACTIVE":                    false,
		"CANCELED":                  false,
		"NOVEL":                     false,
		"INSUFFICIENT BALANCE (G1)": false,
		"RSN_DUST (amount is less than 0.00000001)":        false,
		"PARTIALLY FILLED @ 12.58(-15.53)":                 false,
		"EXECUTED @ 12.58(-15.53)":                         false,
		"FORCED EXECUTED @ 12.58(-15.53)":                  false,
		"... 2365.3(53.94146517)":                          true,
		"...LLED @ 2365.3(45.58642898)":                    true,
		"(-41.37139577), PARTIALLY FILLED @ 12.58(-15.53)": true,
		" 12.58(-15.53)":                                   true,
		"LLED @ 12.58(-15.53)":                             true,
		"ARTIALLY FILLED @ 12.58(-15.53)":                  true,
		"D EXECUTED @ 12.58(-15.53)":                       true,
		"XECUTED @ 12.58(-15.53)":                          true,
	} {
		assert.Equalf(t, exp, isFillListCut(fills), "isFillListCut should report whether %q starts part way through a fill", fills)
	}
}

func TestIsOrderStatusCut(t *testing.T) {
	t.Parallel()
	for status, exp := range map[string]bool{
		"ACTIVE": false,
		"PARTIALLY FILLED @ 2365.3(53.94146517), PARTIALLY FILLED @ 2365.3(45.58642898)": false,
		"CANCELED was: PARTIALLY FILLED @ 107.6(-0.1)":                                   false,
		"... 2365.3(53.94146517), PARTIALLY FILLED @ 2365.3(45.58642898)":                true,
		"CANCELED was: ...LLED @ 2365.3(45.58642898)":                                    true,
		"EXECUTED @ 2365.3(594.14534185): was ...LLED @ 2365.3(45.58642898)":             true,
	} {
		assert.Equalf(t, exp, isOrderStatusCut(status), "isOrderStatusCut should report whether %q was cut short", status)
	}
}

func TestOrderExecutedAmount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		o   *OrderResponse
		exp string
	}{
		{o: &OrderResponse{Amount: 0.25, OriginalAmount: 0.1, Status: "ACTIVE"}, exp: "0"},
		{o: &OrderResponse{Amount: -0.4, OriginalAmount: -1, Status: "PARTIALLY FILLED @ 100.0(-0.3)"}, exp: "0.3"},
		{o: &OrderResponse{Amount: 694.14534185, OriginalAmount: 1093.673236, Status: "... 2365.3(53.94146517), PARTIALLY FILLED @ 2365.3(45.58642898)"}, exp: "399.52789415"},
		{o: &OrderResponse{Amount: -0.9, OriginalAmount: -1, Status: "...LLED @ 100.0(-0.2), PARTIALLY FILLED @ 100.0(-0.2)"}, exp: "0.4"},
		{o: &OrderResponse{OriginalAmount: 1, Status: "CANCELED was: ...LLED @ 100.0(0.2)"}, exp: "1"},
	} {
		got := orderExecutedAmount(tc.o)
		assert.Truef(t, decimal.MustFromString(tc.exp).Equal(got), "orderExecutedAmount should report %s executed for an order of %v, placed for %v, %q, got %s", tc.exp, tc.o.Amount, tc.o.OriginalAmount, tc.o.Status, got)
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
	got, err := e.orderDetail(&OrderResponse{
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

	stop, err := e.orderDetail(&OrderResponse{ID: 1, Symbol: "tBTCUSD", Amount: 1, OriginalAmount: 1, Type: OrderTypeStop, Price: 90, Hidden: true})
	require.NoError(t, err, "orderDetail must not error for a stop order")
	assert.Equal(t, 90.0, stop.TriggerPrice, "a stop order's price should be its trigger")
	assert.Zero(t, stop.Price, "a stop order should trade at the market once triggered")
	assert.True(t, stop.HiddenOrder, "an order Bitfinex reports hidden should be hidden")
	trailing, err := e.orderDetail(&OrderResponse{ID: 1, Symbol: "tBTCUSD", Amount: -1, OriginalAmount: -1, Type: OrderTypeTrailingStop, Price: 95, TrailingPrice: 5})
	require.NoError(t, err, "orderDetail must not error for a trailing stop order")
	assert.Equal(t, [2]float64{95, 0}, [2]float64{trailing.TriggerPrice, trailing.Price}, "a trailing stop order's price should be the stop Bitfinex works out from its distance, its trigger")

	// Bitfinex keeps an order's original amount from before any update, as the documented update reply does: one raised
	// from 0.1 to 0.25, and one lowered from 1 to 0.4 after executing 0.3, each by an update
	for _, tc := range []struct {
		o                           *OrderResponse
		amount, executed, remaining float64
	}{
		{o: &OrderResponse{ID: 1, Symbol: "tBTCUSD", Amount: 0.25, OriginalAmount: 0.1, Type: OrderTypeLimit, Status: "ACTIVE"}, amount: 0.25, remaining: 0.25},
		{o: &OrderResponse{ID: 2, Symbol: "tBTCUSD", Amount: -0.4, OriginalAmount: -1, Type: OrderTypeLimit, Status: "PARTIALLY FILLED @ 100.0(-0.3)"}, amount: 0.7, executed: 0.3, remaining: 0.4},
		// Added up as floats, 0.1 and 0.2 make 0.30000000000000004
		{o: &OrderResponse{ID: 3, Symbol: "tBTCUSD", Amount: 0.2, OriginalAmount: 0.3, Type: OrderTypeLimit, Status: "PARTIALLY FILLED @ 100.0(0.1)"}, amount: 0.3, executed: 0.1, remaining: 0.2},
	} {
		got, err := e.orderDetail(tc.o)
		require.NoErrorf(t, err, "orderDetail must not error for updated order %d", tc.o.ID)
		assert.Equalf(t, tc.amount, got.Amount, "orderDetail should report updated order %d's whole amount as what it executed and what remains", tc.o.ID)
		assert.Equalf(t, tc.executed, got.ExecutedAmount, "orderDetail should report what updated order %d executed from the fills its status lists", tc.o.ID)
		assert.Equalf(t, tc.remaining, got.RemainingAmount, "orderDetail should report what remains of updated order %d", tc.o.ID)
	}

	for _, tc := range []struct {
		o   *OrderResponse
		err error
	}{
		{o: &OrderResponse{Symbol: "tBTC", Type: OrderTypeLimit, OriginalAmount: 1}, err: errSymbolUnparsable},
		{o: &OrderResponse{Symbol: "tBTCUSD", Type: "NOVEL", OriginalAmount: 1}, err: errUnknownOrderType},
		{o: &OrderResponse{Symbol: "tBTCUSD", Type: OrderTypeLimit}, err: errOrderSideUnresolved},
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

func TestWalletBalance(t *testing.T) {
	t.Parallel()
	available, short := 0.25, -0.5
	assert.Equal(t, accounts.Balance{Total: 1, Free: 0.25, Hold: 0.75}, walletBalance(&WalletResponse{Balance: 1, AvailableBalance: &available}), "walletBalance should hold what is not available")
	assert.Equal(t, accounts.Balance{Total: 1, Free: -0.5, Hold: 1.5}, walletBalance(&WalletResponse{Balance: 1, AvailableBalance: &short}), "walletBalance should keep a negative available balance as Bitfinex sends it")
	assert.Equal(t, accounts.Balance{Total: 1, Hold: 1}, walletBalance(&WalletResponse{Balance: 1}), "walletBalance should hold the whole balance until Bitfinex sends its available balance")
}
