package okx

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var instrumentTypeToAssetTypeMap = map[string]struct {
	AssetType asset.Item
	Error     error
}{
	instTypeSwap:     {AssetType: asset.PerpetualSwap},
	instTypeContract: {AssetType: asset.PerpetualSwap},
	instTypeSpot:     {AssetType: asset.Spot},
	instTypeMargin:   {AssetType: asset.Margin},
	instTypeFutures:  {AssetType: asset.Futures},
	instTypeOption:   {AssetType: asset.Options},
	"":               {AssetType: asset.Empty},
	"lol":            {AssetType: asset.Empty, Error: asset.ErrNotSupported},
}

func TestAssetTypeFromInstrumentType(t *testing.T) {
	t.Parallel()
	for k, v := range instrumentTypeToAssetTypeMap {
		assetItem, err := assetTypeFromInstrumentType(k)
		require.ErrorIs(t, err, v.Error)
		assert.Equal(t, v.AssetType, assetItem)
	}
}

func TestGetPairFromInstrumentID(t *testing.T) {
	t.Parallel()
	instruments := []string{
		mainPair.String(),
		perpetualSwapPair.String(),
		"BTC-USDT-ER33234",
	}
	dPair, err := e.GetPairFromInstrumentID(instruments[0])
	require.NoError(t, err)
	require.NotNil(t, dPair)
	dPair, err = e.GetPairFromInstrumentID(instruments[1])
	require.NoError(t, err)
	require.NotNil(t, dPair)
	dPair, err = e.GetPairFromInstrumentID(instruments[2])
	require.NoError(t, err)
	assert.NotNil(t, dPair)
}

// TestOrderTypeString guards the ordType an order places. The post-only rows
// read PostOnly from a time in force that combines PostOnly with other flags,
// where an unlisted type combination is still refused.
func TestOrderTypeString(t *testing.T) {
	t.Parallel()
	type OrderTypeWithTIF struct {
		OrderType order.Type
		TIF       order.TimeInForce
	}
	orderTypesToStringMap := map[OrderTypeWithTIF]struct {
		Expected string
		Error    error
	}{
		{OrderType: order.Market, TIF: order.UnknownTIF}:                {Expected: orderMarket},
		{OrderType: order.Limit, TIF: order.UnknownTIF}:                 {Expected: orderLimit},
		{OrderType: order.Limit, TIF: order.PostOnly}:                   {Expected: orderPostOnly},
		{OrderType: order.Limit, TIF: order.FillOrKill}:                 {Expected: orderFOK},
		{OrderType: order.Limit, TIF: order.ImmediateOrCancel}:          {Expected: orderIOC},
		{OrderType: order.LimitMaker, TIF: order.UnknownTIF}:            {Expected: orderPostOnly},
		{OrderType: order.LimitMaker, TIF: order.ImmediateOrCancel}:     {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.LimitMaker, TIF: order.FillOrKill}:            {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.Stop, TIF: order.ImmediateOrCancel}:           {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.StopLimit, TIF: order.FillOrKill}:             {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.StopMarket, TIF: order.FillOrKill}:            {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.TakeProfit, TIF: order.FillOrKill}:            {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.TakeProfitMarket, TIF: order.FillOrKill}:      {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.TrailingStopLimit, TIF: order.FillOrKill}:     {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.Bracket, TIF: order.FillOrKill}:               {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.Liquidation, TIF: order.FillOrKill}:           {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.Market, TIF: order.FillOrKill}:                {Expected: orderFOK},
		{OrderType: order.Market, TIF: order.ImmediateOrCancel}:         {Expected: orderIOC},
		{OrderType: order.OptimalLimit, TIF: order.ImmediateOrCancel}:   {Expected: orderOptimalLimitIOC},
		{OrderType: order.MarketMakerProtection, TIF: order.UnknownTIF}: {Expected: orderMarketMakerProtection},
		{OrderType: order.MarketMakerProtection, TIF: order.PostOnly}:   {Expected: orderMarketMakerProtectionAndPostOnly},
		{OrderType: order.Liquidation, TIF: order.UnknownTIF}:           {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.OCO, TIF: order.UnknownTIF}:                   {Expected: orderOCO},
		{OrderType: order.TrailingStop, TIF: order.UnknownTIF}:          {Expected: orderMoveOrderStop},
		{OrderType: order.Chase, TIF: order.UnknownTIF}:                 {Expected: orderChase},
		{OrderType: order.TWAP, TIF: order.UnknownTIF}:                  {Expected: orderTWAP},
		{OrderType: order.ConditionalStop, TIF: order.UnknownTIF}:       {Expected: orderConditional},
		{OrderType: order.Chase, TIF: order.GoodTillCancel}:             {Expected: orderChase},
		{OrderType: order.TWAP, TIF: order.ImmediateOrCancel}:           {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.ConditionalStop, TIF: order.GoodTillDay}:      {Expected: orderConditional},
		{OrderType: order.Trigger, TIF: order.UnknownTIF}:               {Expected: orderTrigger},
		{OrderType: order.UnknownType, TIF: order.PostOnly}:             {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.UnknownType, TIF: order.FillOrKill}:           {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.UnknownType, TIF: order.ImmediateOrCancel}:    {Error: order.ErrUnsupportedOrderType},
		// A time in force the order type cannot honour is refused rather than
		// dropped by an order which ignores it, as is a type OKX has no order for.
		{OrderType: order.AnyType, TIF: order.ImmediateOrCancel}:                {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.AnyType, TIF: order.UnknownTIF}:                       {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.Market, TIF: order.PostOnly}:                          {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.Market, TIF: order.GoodTillCancel | order.PostOnly}:   {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.Market, TIF: order.GoodTillCrossing | order.PostOnly}: {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.Market, TIF: order.GoodTillCancel}:                    {Expected: orderMarket},
		{OrderType: order.OptimalLimit, TIF: order.UnknownTIF}:                  {Expected: orderOptimalLimitIOC},
		{OrderType: order.OptimalLimit, TIF: order.PostOnly}:                    {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.OptimalLimit, TIF: order.FillOrKill}:                  {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.OptimalLimit, TIF: order.GoodTillCancel}:              {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.MarketMakerProtection, TIF: order.ImmediateOrCancel}:  {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.MarketMakerProtection, TIF: order.FillOrKill}:         {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.MarketMakerProtection, TIF: order.GoodTillCancel}:     {Expected: orderMarketMakerProtection},
		{OrderType: order.Trigger, TIF: order.FillOrKill}:                       {Expected: orderTrigger},
		{OrderType: order.Trigger, TIF: order.ImmediateOrCancel}:                {Expected: orderTrigger},
		{OrderType: order.Trigger, TIF: order.PostOnly}:                         {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.Trigger, TIF: order.GoodTillCancel | order.PostOnly}:  {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.Chase, TIF: order.PostOnly}:                           {Expected: orderChase},
		{OrderType: order.Chase, TIF: order.ImmediateOrCancel}:                  {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.Chase, TIF: order.FillOrKill}:                         {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.OCO, TIF: order.PostOnly}:                             {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.OCO, TIF: order.GoodTillCancel}:                       {Expected: orderOCO},
		{OrderType: order.ConditionalStop, TIF: order.FillOrKill}:               {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.TrailingStop, TIF: order.ImmediateOrCancel}:           {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.TWAP, TIF: order.GoodTillCrossing | order.PostOnly}:   {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.Limit, TIF: order.ImmediateOrCancel | order.PostOnly}: {Error: order.ErrInvalidTimeInForce},
		{OrderType: order.Limit, TIF: order.PostOnly | 1<<15}:                   {Error: order.ErrInvalidTimeInForce},
		{OrderType: order.Market, TIF: order.FillOrKill | order.GoodTillCancel}: {Error: order.ErrInvalidTimeInForce},
		// Only Limit and MarketMakerProtection read PostOnly from a combined
		// time in force; an unlisted type combination is still refused.
		{OrderType: order.TakeProfit | order.Limit, TIF: order.GoodTillCancel | order.PostOnly}: {Error: order.ErrUnsupportedOrderType},
	}
	for tc, val := range orderTypesToStringMap {
		t.Run(tc.OrderType.String()+"/"+tc.TIF.String(), func(t *testing.T) {
			t.Parallel()
			orderTypeString, err := orderTypeString(tc.OrderType, tc.TIF)
			require.ErrorIs(t, err, val.Error)
			assert.Equal(t, val.Expected, orderTypeString)
		})
	}

	t.Run("post-only is read from combined time in forces", func(t *testing.T) {
		t.Parallel()
		// A time in force that combines PostOnly with other flags, which
		// TimeInForce.IsValid allows, must still place as post-only.
		flags := []order.TimeInForce{order.GoodTillCancel, order.GoodTillDay, order.GoodTillTime, order.GoodTillCrossing, order.StopOrReduce}
		for mask := range 1 << len(flags) {
			tif := order.PostOnly
			for i, f := range flags {
				if mask&(1<<i) != 0 {
					tif |= f
				}
			}
			require.Truef(t, tif.IsValid(), "%s must be a valid time in force", tif)
			got, err := orderTypeString(order.Limit, tif)
			require.NoErrorf(t, err, "orderTypeString must not error for a limit order with %s", tif)
			assert.Equalf(t, orderPostOnly, got, "a limit order with %s should place as post_only", tif)
			got, err = orderTypeString(order.MarketMakerProtection, tif)
			require.NoErrorf(t, err, "orderTypeString must not error for an MMP order with %s", tif)
			assert.Equalf(t, orderMarketMakerProtectionAndPostOnly, got, "an MMP order with %s should place as mmp_and_post_only", tif)
		}
	})
}

func TestTriggerAdvanceOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		tif        order.TimeInForce
		orderPrice float64
		exp        string
		err        error
	}{
		{name: "no time in force", orderPrice: 100},
		{name: "good till cancel", tif: order.GoodTillCancel, orderPrice: 100},
		{name: "fill or kill", tif: order.FillOrKill, orderPrice: 100, exp: orderFOK},
		{name: "immediate or cancel", tif: order.ImmediateOrCancel, orderPrice: 100, exp: orderIOC},
		{name: "fill or kill at market", tif: order.FillOrKill, orderPrice: marketOrderPrice, err: order.ErrUnsupportedTimeInForce},
		{name: "immediate or cancel at market", tif: order.ImmediateOrCancel, orderPrice: marketOrderPrice, err: order.ErrUnsupportedTimeInForce},
		{name: "no time in force at market", orderPrice: marketOrderPrice},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			advanceOrderType, err := triggerAdvanceOrderType(tc.tif, tc.orderPrice)
			require.ErrorIs(t, err, tc.err, "triggerAdvanceOrderType must return the expected error")
			assert.Equal(t, tc.exp, advanceOrderType, "triggerAdvanceOrderType should return the advanceOrdType the time in force places with")
		})
	}
}

func TestTriggerTimeInForce(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name             string
		advanceOrderType string
		exp              order.TimeInForce
	}{
		{"no advance order type", "", order.UnknownTIF},
		{"fill or kill", orderFOK, order.FillOrKill},
		{"immediate or cancel", orderIOC, order.ImmediateOrCancel},
		{"fill or kill in upper case", "FOK", order.FillOrKill},
		{"immediate or cancel in upper case", "IOC", order.ImmediateOrCancel},
		{"chase", orderChase, order.UnknownTIF},
		{"an advance order type OKX has not documented", "post_only", order.UnknownTIF},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, triggerTimeInForce(tc.advanceOrderType), "triggerTimeInForce should return the time in force of the order the trigger order places")
		})
	}
	for _, tif := range []order.TimeInForce{order.FillOrKill, order.ImmediateOrCancel} {
		advanceOrderType, err := triggerAdvanceOrderType(tif, 100)
		require.NoErrorf(t, err, "triggerAdvanceOrderType must place a %s trigger order", tif)
		assert.Equalf(t, tif, triggerTimeInForce(advanceOrderType), "a %s trigger order should read back with the time in force it was placed with", tif)
	}
}

// TestSpreadOrderTypeString pins the ordType a spread order places: the
// spread endpoints document only market, limit, post_only and ioc, so fok
// and trigger-style types are rejected instead of downgraded.
func TestSpreadOrderTypeString(t *testing.T) {
	t.Parallel()
	type OrderTypeWithTIF struct {
		OrderType order.Type
		TIF       order.TimeInForce
	}
	spreadOrderTypesToStringMap := map[OrderTypeWithTIF]struct {
		Expected string
		Error    error
	}{
		{OrderType: order.Market, TIF: order.UnknownTIF}:                          {Expected: orderMarket},
		{OrderType: order.Market, TIF: order.ImmediateOrCancel}:                   {Expected: orderMarket},
		{OrderType: order.Market, TIF: order.FillOrKill}:                          {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.Market, TIF: order.GoodTillCancel}:                      {Expected: orderMarket},
		{OrderType: order.Market, TIF: order.PostOnly}:                            {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.Market, TIF: order.GoodTillCrossing | order.PostOnly}:   {Error: order.ErrUnsupportedTimeInForce},
		{OrderType: order.Limit, TIF: order.UnknownTIF}:                           {Expected: orderLimit},
		{OrderType: order.Limit, TIF: order.GoodTillCancel}:                       {Expected: orderLimit},
		{OrderType: order.Limit, TIF: order.GoodTillDay}:                          {Expected: orderLimit},
		{OrderType: order.Limit, TIF: order.PostOnly}:                             {Expected: orderPostOnly},
		{OrderType: order.Limit, TIF: order.GoodTillCancel | order.PostOnly}:      {Expected: orderPostOnly},
		{OrderType: order.Limit, TIF: order.GoodTillDay | order.PostOnly}:         {Expected: orderPostOnly},
		{OrderType: order.Limit, TIF: order.GoodTillTime | order.PostOnly}:        {Expected: orderPostOnly},
		{OrderType: order.Limit, TIF: order.ImmediateOrCancel | order.PostOnly}:   {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.Limit, TIF: order.PostOnly | 1<<15}:                     {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.Limit, TIF: order.ImmediateOrCancel}:                    {Expected: orderIOC},
		{OrderType: order.Limit, TIF: order.FillOrKill}:                           {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.LimitMaker, TIF: order.UnknownTIF}:                      {Expected: orderPostOnly},
		{OrderType: order.LimitMaker, TIF: order.GoodTillCancel}:                  {Expected: orderPostOnly},
		{OrderType: order.LimitMaker, TIF: order.GoodTillDay}:                     {Expected: orderPostOnly},
		{OrderType: order.LimitMaker, TIF: order.PostOnly}:                        {Expected: orderPostOnly},
		{OrderType: order.LimitMaker, TIF: order.GoodTillCancel | order.PostOnly}: {Expected: orderPostOnly},
		{OrderType: order.LimitMaker, TIF: order.GoodTillTime | order.PostOnly}:   {Expected: orderPostOnly},
		{OrderType: order.LimitMaker, TIF: order.FillOrKill | order.PostOnly}:     {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.LimitMaker, TIF: order.PostOnly | 1<<15}:                {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.LimitMaker, TIF: order.ImmediateOrCancel}:               {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.LimitMaker, TIF: order.FillOrKill}:                      {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.Stop, TIF: order.UnknownTIF}:                            {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.ConditionalStop, TIF: order.UnknownTIF}:                 {Error: order.ErrUnsupportedOrderType},
		// A market order's time in force must be one OKX can honour.
		{OrderType: order.Market, TIF: order.ImmediateOrCancel | order.GoodTillCancel}: {Error: order.ErrUnsupportedOrderType},
	}
	for tc, val := range spreadOrderTypesToStringMap {
		t.Run(tc.OrderType.String()+"/"+tc.TIF.String(), func(t *testing.T) {
			t.Parallel()
			spreadOrderType, err := spreadOrderTypeString(tc.OrderType, tc.TIF)
			require.ErrorIs(t, err, val.Error)
			assert.Equal(t, val.Expected, spreadOrderType)
		})
	}
}

func TestOrderTypeFromString(t *testing.T) {
	t.Parallel()
	orderTypeStrings := map[string]struct {
		OType order.Type
		TIF   order.TimeInForce
		Error error
	}{
		"market":            {OType: order.Market},
		"LIMIT":             {OType: order.Limit},
		"limit":             {OType: order.Limit},
		"post_only":         {OType: order.Limit, TIF: order.PostOnly},
		"fok":               {OType: order.Limit, TIF: order.FillOrKill},
		"ioc":               {OType: order.Limit, TIF: order.ImmediateOrCancel},
		"op_fok":            {OType: order.Limit, TIF: order.FillOrKill},
		"rpi":               {OType: order.Limit},
		"elp":               {OType: order.Limit},
		"optimal_limit_ioc": {OType: order.OptimalLimit, TIF: order.ImmediateOrCancel},
		"mmp":               {OType: order.MarketMakerProtection},
		"mmp_and_post_only": {OType: order.MarketMakerProtection, TIF: order.PostOnly},
		"trigger":           {OType: order.UnknownType, Error: order.ErrTypeIsInvalid},
		"chase":             {OType: order.Chase},
		"move_order_stop":   {OType: order.TrailingStop},
		"twap":              {OType: order.TWAP},
		"abcd":              {OType: order.UnknownType, Error: order.ErrTypeIsInvalid},
	}
	for s, exp := range orderTypeStrings {
		t.Run(s, func(t *testing.T) {
			t.Parallel()
			oType, tif, err := orderTypeFromString(s)
			require.ErrorIs(t, err, exp.Error)
			assert.Equal(t, exp.OType, oType)
			assert.Equal(t, exp.TIF.String(), tif.String())
		})
	}
}

func TestOrderTypeFilter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType order.Type
		tif       order.TimeInForce
		exp       string
		err       error
	}{
		{order.Limit, order.UnknownTIF, "limit,post_only,fok,ioc,op_fok,rpi", nil},
		{order.Limit, order.PostOnly, orderPostOnly, nil},
		{order.Limit, order.FillOrKill, "fok,op_fok", nil},
		{order.Limit, order.ImmediateOrCancel, orderIOC, nil},
		{order.Limit, order.GoodTillCancel, "limit,rpi", nil},
		{order.MarketMakerProtection, order.UnknownTIF, "mmp,mmp_and_post_only", nil},
		{order.MarketMakerProtection, order.PostOnly, orderMarketMakerProtectionAndPostOnly, nil},
		{order.Limit, order.GoodTillCancel | order.PostOnly, orderPostOnly, nil},
		{order.MarketMakerProtection, order.GoodTillCancel | order.PostOnly, orderMarketMakerProtectionAndPostOnly, nil},
		{order.Market, order.UnknownTIF, orderMarket, nil},
		{order.Trigger, order.UnknownTIF, "trigger", nil},
		{order.Stop, order.UnknownTIF, "", order.ErrUnsupportedOrderType},
		{order.Limit, order.GoodTillDay, "", order.ErrUnsupportedTimeInForce},
		{order.Limit, order.GoodTillTime, "", order.ErrUnsupportedTimeInForce},
		{order.Limit, order.StopOrReduce, "", order.ErrUnsupportedTimeInForce},
		{order.Limit, order.PostOnly | 1<<15, "", order.ErrUnsupportedTimeInForce},
		{order.Market, order.ImmediateOrCancel, "", order.ErrUnsupportedTimeInForce},
		{order.Market, order.FillOrKill, "", order.ErrUnsupportedTimeInForce},
		{order.MarketMakerProtection, order.ImmediateOrCancel, "", order.ErrUnsupportedTimeInForce},
		{order.OptimalLimit, order.UnknownTIF, orderOptimalLimitIOC, nil},
		{order.OptimalLimit, order.ImmediateOrCancel, orderOptimalLimitIOC, nil},
		{order.OptimalLimit, order.GoodTillCancel, "", order.ErrUnsupportedTimeInForce},
		{order.LimitMaker, order.UnknownTIF, orderPostOnly, nil},
		{order.LimitMaker, order.PostOnly, orderPostOnly, nil},
		{order.LimitMaker, order.GoodTillCancel, orderPostOnly, nil},
		{order.LimitMaker, order.GoodTillCancel | order.PostOnly, orderPostOnly, nil},
		{order.LimitMaker, order.GoodTillDay, "", order.ErrUnsupportedTimeInForce},
		{order.LimitMaker, order.FillOrKill, "", order.ErrUnsupportedOrderType},
	} {
		t.Run(fmt.Sprintf("%s %s", tc.orderType, tc.tif), func(t *testing.T) {
			t.Parallel()
			got, err := orderTypeFilter(tc.orderType, tc.tif)
			require.ErrorIs(t, err, tc.err, "orderTypeFilter must refuse a time in force no pending order of the type reads back with")
			assert.Equal(t, tc.exp, got, "orderTypeFilter should select the OKX order types read back as the type and time in force")
		})
	}
}

func TestListingOrderTypeFilter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		orderType order.Type
		tif       order.TimeInForce
		exp       string
		err       error
	}{
		{name: "market", orderType: order.Market, exp: orderMarket},
		{name: "market immediate or cancel", orderType: order.Market, tif: order.ImmediateOrCancel, err: order.ErrUnsupportedTimeInForce},
		{name: "market fill or kill", orderType: order.Market, tif: order.FillOrKill, err: order.ErrUnsupportedTimeInForce},
		{name: "market post only", orderType: order.Market, tif: order.PostOnly, err: order.ErrUnsupportedTimeInForce},
		{name: "market good till cancel", orderType: order.Market, tif: order.GoodTillCancel, err: order.ErrUnsupportedTimeInForce},
		{name: "limit", orderType: order.Limit, exp: "limit,post_only,fok,ioc,op_fok,rpi"},
		{name: "limit immediate or cancel", orderType: order.Limit, tif: order.ImmediateOrCancel, exp: orderIOC},
		{name: "limit good till day", orderType: order.Limit, tif: order.GoodTillDay, err: order.ErrUnsupportedTimeInForce},
		{name: "market maker protection", orderType: order.MarketMakerProtection, exp: "mmp,mmp_and_post_only"},
		{name: "limit maker", orderType: order.LimitMaker, exp: orderPostOnly},
		{name: "unsupported order type", orderType: order.Stop, err: order.ErrUnsupportedOrderType},
		{name: "any type post only", orderType: order.AnyType, tif: order.PostOnly, exp: "post_only,mmp_and_post_only"},
		{name: "any type good till cancel", orderType: order.AnyType, tif: order.GoodTillCancel, exp: "limit,mmp,rpi"},
		{name: "any type good till cancel post only", orderType: order.AnyType, tif: order.GoodTillCancel | order.PostOnly, exp: "post_only,mmp_and_post_only"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			filter, err := listingOrderTypeFilter(tc.orderType, tc.tif)
			require.ErrorIs(t, err, tc.err, "listingOrderTypeFilter must return the expected error")
			assert.Equal(t, tc.exp, filter, "listingOrderTypeFilter should narrow any order type by its time in force and otherwise filter as orderTypeFilter")
		})
	}
}

func TestFilterTimeInForceMatches(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		readType  order.Type
		readBack  order.TimeInForce
		requested order.TimeInForce
		exp       bool
	}{
		{"no requested time in force", order.Limit, order.FillOrKill, order.UnknownTIF, true},
		{"the same time in force", order.Limit, order.PostOnly, order.PostOnly, true},
		{"another time in force", order.Limit, order.PostOnly, order.ImmediateOrCancel, false},
		{"good till cancel on a resting order", order.Limit, order.UnknownTIF, order.GoodTillCancel, true},
		{"good till cancel on a market order", order.Market, order.UnknownTIF, order.GoodTillCancel, false},
		{"good till cancel on an immediate order", order.Limit, order.ImmediateOrCancel, order.GoodTillCancel, false},
		{"good till cancel with post only on a post only order", order.Limit, order.PostOnly, order.GoodTillCancel | order.PostOnly, true},
		{"good till cancel with post only on a resting order", order.Limit, order.UnknownTIF, order.GoodTillCancel | order.PostOnly, false},
		{"good till cancel with stop or reduce", order.Limit, order.UnknownTIF, order.GoodTillCancel | order.StopOrReduce, false},
		{"an invalid combination", order.Limit, order.FillOrKill, order.GoodTillCancel | order.FillOrKill, false},
		{"good till day on a resting order", order.Limit, order.UnknownTIF, order.GoodTillDay, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, filterTimeInForceMatches(tc.readType, tc.readBack, tc.requested), "filterTimeInForceMatches should match a filter's time in force to what an order reads back with")
		})
	}
}

func TestTimeInForceFilter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		tif  order.TimeInForce
		exp  string
		err  error
	}{
		{name: "post only", tif: order.PostOnly, exp: "post_only,mmp_and_post_only"},
		{name: "good till cancel post only", tif: order.GoodTillCancel | order.PostOnly, exp: "post_only,mmp_and_post_only"},
		{name: "fill or kill", tif: order.FillOrKill, exp: "fok,op_fok"},
		{name: "immediate or cancel", tif: order.ImmediateOrCancel, exp: "ioc,optimal_limit_ioc"},
		{name: "good till cancel", tif: order.GoodTillCancel, exp: "limit,mmp,rpi"},
		{name: "good till day", tif: order.GoodTillDay, err: order.ErrUnsupportedTimeInForce},
		{name: "good till time", tif: order.GoodTillTime, err: order.ErrUnsupportedTimeInForce},
		{name: "post only with an unknown flag", tif: order.PostOnly | 1<<15, err: order.ErrUnsupportedTimeInForce},
		{name: "no time in force", tif: order.UnknownTIF, exp: "market,limit,mmp,rpi"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			filter, err := timeInForceFilter(tc.tif)
			require.ErrorIs(t, err, tc.err, "timeInForceFilter must refuse a time in force no pending order reads back with")
			assert.Equal(t, tc.exp, filter, "timeInForceFilter should select every OKX order type read back with the time in force")
		})
	}
}

func TestSpreadOrderTypeFilter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType order.Type
		tif       order.TimeInForce
		exp       string
	}{
		// limit, post_only and ioc all read back as Limit: no filter is sent
		// and the request filter narrows by type instead.
		{order.Limit, order.UnknownTIF, ""},
		{order.Limit, order.GoodTillCancel, orderLimit},
		{order.Limit, order.PostOnly, orderPostOnly},
		{order.Limit, order.GoodTillCancel | order.PostOnly, orderPostOnly},
		{order.Limit, order.ImmediateOrCancel, orderIOC},
		{order.Market, order.UnknownTIF, orderMarket},
		// A market order with ImmediateOrCancel places as market.
		{order.Market, order.ImmediateOrCancel, orderMarket},
	} {
		got, err := spreadOrderTypeFilter(tc.orderType, tc.tif)
		require.NoErrorf(t, err, "spreadOrderTypeFilter must not error for %s %s", tc.orderType, tc.tif)
		assert.Equalf(t, tc.exp, got, "spreadOrderTypeFilter should select a documented spread ordType for %s %s", tc.orderType, tc.tif)
	}
	_, err := spreadOrderTypeFilter(order.Market, order.FillOrKill)
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "spreadOrderTypeFilter should reject the fok a fill-or-kill market order maps to")
	_, err = spreadOrderTypeFilter(order.Limit, order.FillOrKill)
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "spreadOrderTypeFilter should reject the fok a fill-or-kill limit order maps to")
	_, err = spreadOrderTypeFilter(order.Trigger, order.UnknownTIF)
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "spreadOrderTypeFilter should reject an order type the spread endpoints do not list")

	t.Run("any order type narrows by its time in force", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			tif  order.TimeInForce
			exp  string
			err  error
		}{
			{name: "post only", tif: order.PostOnly, exp: orderPostOnly},
			{name: "good till cancel post only", tif: order.GoodTillCancel | order.PostOnly, exp: orderPostOnly},
			{name: "immediate or cancel", tif: order.ImmediateOrCancel, exp: orderIOC},
			{name: "good till cancel", tif: order.GoodTillCancel, exp: orderLimit},
			{name: "fill or kill", tif: order.FillOrKill, err: order.ErrUnsupportedOrderType},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				got, err := spreadOrderTypeFilter(order.AnyType, tc.tif)
				require.ErrorIs(t, err, tc.err, "spreadOrderTypeFilter must return the expected error")
				assert.Equal(t, tc.exp, got, "spreadOrderTypeFilter should narrow any order type as a limit listing with the time in force")
			})
		}
	})
}

func TestGetAssetsFromInstrumentID(t *testing.T) {
	t.Parallel()
	newExchange := func(t *testing.T, format *currency.PairFormat, enabled map[asset.Item]currency.Pairs) *Exchange {
		t.Helper()
		ex := new(Exchange)
		ex.CurrencyPairs.UseGlobalFormat = true
		ex.CurrencyPairs.RequestFormat = format
		for a, pairs := range enabled {
			require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, pairs, true), "StorePairs must not error for %s", a)
		}
		return ex
	}
	dashFormat := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	pair := func(base, quote string) currency.Pair {
		return currency.NewPairWithDelimiter(base, quote, currency.DashDelimiter)
	}
	enabled := newExchange(t, dashFormat, map[asset.Item]currency.Pairs{
		asset.Spot:          {mainPair, pair("ETH", "USDT")},
		asset.Margin:        {mainPair, pair("OKB", "USDT")},
		asset.PerpetualSwap: {perpetualSwapPair},
		asset.Futures:       {pair("BTC", "USD-261225")},
		asset.Options:       {pair("BTC", "USD-261225-100000-C"), pair("BTC", "USD-261225-100000-P")},
	})
	spotOnly := newExchange(t, dashFormat, map[asset.Item]currency.Pairs{asset.Spot: {mainPair}})
	for _, tc := range []struct {
		name         string
		ex           *Exchange
		instrumentID string
		exp          []asset.Item
		err          error
	}{
		{name: "empty instrument ID", ex: enabled, err: errMissingInstrumentID},
		{name: "instrument ID without the delimiter", ex: enabled, instrumentID: "BTCUSDT", err: currency.ErrCurrencyNotSupported},
		{name: "spot and margin pair", ex: enabled, instrumentID: "BTC-USDT", exp: []asset.Item{asset.Spot, asset.Margin}},
		{name: "lower case spot and margin pair", ex: enabled, instrumentID: "btc-usdt", exp: []asset.Item{asset.Spot, asset.Margin}},
		{name: "spot pair", ex: enabled, instrumentID: "ETH-USDT", exp: []asset.Item{asset.Spot}},
		{name: "margin pair", ex: enabled, instrumentID: "OKB-USDT", exp: []asset.Item{asset.Margin}},
		{name: "pair enabled for neither spot nor margin", ex: enabled, instrumentID: "LTC-USDT", err: asset.ErrNotEnabled},
		{name: "perpetual swap", ex: enabled, instrumentID: "BTC-USDT-SWAP", exp: []asset.Item{asset.PerpetualSwap}},
		{name: "lower case perpetual swap", ex: enabled, instrumentID: "btc-usdt-swap", exp: []asset.Item{asset.PerpetualSwap}},
		{name: "perpetual swap not enabled", ex: enabled, instrumentID: "ETH-USDT-SWAP", err: asset.ErrNotEnabled},
		{name: "futures", ex: enabled, instrumentID: "BTC-USD-261225", exp: []asset.Item{asset.Futures}},
		{name: "futures not enabled", ex: enabled, instrumentID: "BTC-USD-270326", err: asset.ErrNotEnabled},
		{name: "call option", ex: enabled, instrumentID: "BTC-USD-261225-100000-C", exp: []asset.Item{asset.Options}},
		{name: "lower case put option", ex: enabled, instrumentID: "btc-usd-261225-100000-p", exp: []asset.Item{asset.Options}},
		{name: "option not enabled", ex: enabled, instrumentID: "BTC-USD-261225-90000-C", err: asset.ErrNotEnabled},
		{name: "no request format", ex: newExchange(t, nil, nil), instrumentID: "BTC-USDT", err: currency.ErrPairFormatIsNil},
		{name: "no pair stores", ex: newExchange(t, dashFormat, nil), instrumentID: "BTC-USDT", err: currency.ErrPairManagerNotInitialised},
		{name: "no margin pair store", ex: spotOnly, instrumentID: "BTC-USDT", err: currency.ErrAssetNotFound},
		{name: "no perpetual swap pair store", ex: spotOnly, instrumentID: "BTC-USDT-SWAP", err: currency.ErrAssetNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assets, err := tc.ex.getAssetsFromInstrumentID(tc.instrumentID)
			require.ErrorIs(t, err, tc.err, "getAssetsFromInstrumentID must return the expected error")
			assert.Equal(t, tc.exp, assets, "getAssetsFromInstrumentID should return every asset the instrument is enabled for")
		})
	}

	t.Run("empty delimiter", func(t *testing.T) {
		t.Parallel()
		ex := newExchange(t, &currency.PairFormat{Uppercase: true}, map[asset.Item]currency.Pairs{asset.Spot: {mainPair}})
		assets, err := ex.getAssetsFromInstrumentID("BTC-USDT")
		assert.ErrorContains(t, err, "delimiter cannot be empty", "getAssetsFromInstrumentID should reject a pair format without a delimiter")
		assert.Nil(t, assets, "getAssetsFromInstrumentID should return no assets beside the error")
	})
}
