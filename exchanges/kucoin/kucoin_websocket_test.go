package kucoin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
)

// ConnectionFixture is a websocket connection mock, this is used in test that previously used the actual websocket connection.
// The current design makes it difficult to connect and subscribe using the actual websocket connection in tests, so
// this mock is used to simulate the connection behaviour.
type ConnectionFixture struct {
	websocket.Connection
	messageResponse string
	messageError    error
	sendFn          func(any) ([]byte, error)
	sentRequests    []WsSubscriptionInput
}

func (c *ConnectionFixture) SendMessageReturnResponse(_ context.Context, _ request.EndpointLimit, _, req any) ([]byte, error) {
	if input, ok := req.(WsSubscriptionInput); ok {
		c.sentRequests = append(c.sentRequests, input)
	}
	if c.messageError != nil {
		return nil, c.messageError
	}
	if c.sendFn != nil {
		return c.sendFn(req)
	}
	return []byte(c.messageResponse), nil
}

func expectedPerPairSubscriptions(channel string, a asset.Item, pairs currency.Pairs, qualifiedPrefix string, interval kline.Interval, suffixFn func(currency.Pair) string) subscription.List {
	if suffixFn == nil {
		suffixFn = func(pair currency.Pair) string {
			return pair.String()
		}
	}

	resp := make(subscription.List, 0, len(pairs))
	for _, pair := range pairs {
		resp = append(resp, &subscription.Subscription{
			Channel:          channel,
			Asset:            a,
			Pairs:            currency.Pairs{pair},
			QualifiedChannel: qualifiedPrefix + ":" + suffixFn(pair),
			Interval:         interval,
		})
	}
	return resp
}

func TestGetInstanceServers(t *testing.T) {
	t.Parallel()
	result, err := e.GetInstanceServers(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAuthenticatedServersInstances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAuthenticatedInstanceServers(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPushData(t *testing.T) {
	t.Parallel()

	e := testInstance(t)
	// Isolate global orderbook keying by exchange name to avoid cross-test contamination
	// when Kucoin websocket tests run in parallel.
	e.Name += "-TestPushData"
	e.SetCredentials("mock", "test", "test", "", "", "", "", "", "")
	e.API.AuthenticatedSupport = true
	e.API.AuthenticatedWebsocketSupport = true

	e.wsOBUpdateMgr = buffer.NewUpdateManager(&buffer.UpdateManagerParams{
		BufferInstance: &e.Websocket.Orderbook,
		CheckPendingUpdate: func(_, _ int64, _ *orderbook.Update) (skip bool, err error) {
			return false, nil
		},
		FetchDeadline: buffer.DefaultWSOrderbookUpdateDeadline,
		FetchOrderbook: func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
			if p.Equal(currency.NewBTCUSDT()) && a == asset.Spot {
				return &orderbook.Book{
					Pair:        p,
					Asset:       a,
					Exchange:    e.Name,
					Bids:        []orderbook.Level{{Amount: 1, Price: 100}},
					Asks:        []orderbook.Level{{Amount: 1, Price: 100}},
					LastUpdated: time.Now(),
				}, nil
			}
			if p.Equal(currency.NewPair(currency.ETH, currency.USDCM)) && a == asset.Futures {
				return &orderbook.Book{
					Pair:        p,
					Asset:       a,
					Exchange:    e.Name,
					Bids:        []orderbook.Level{{Amount: 1, Price: 100}},
					Asks:        []orderbook.Level{{Amount: 1, Price: 100}},
					LastUpdated: time.Now(),
				}, nil
			}
			panic("test update manager unexpected pair or asset")
		},
	})

	fErrs := testexch.FixtureToDataHandlerWithErrors(t, "testdata/wsHandleData.json", func(ctx context.Context, r []byte) error {
		if bytes.Contains(r, []byte("FANGLE-ACCOUNTS")) {
			hold := e.Accounts
			e.Accounts = nil
			defer func() { e.Accounts = hold }()
		}
		return e.wsHandleData(ctx, nil, r)
	})

	require.Eventually(t, func() bool { return len(e.Websocket.DataHandler.C) == 31 }, time.Second, time.Millisecond*10, "must receive 31 messages")
	require.Len(t, fErrs, 1, "Must get exactly one error message")
	assert.ErrorContains(t, fErrs[0].Err, "cannot save holdings: nil pointer: *accounts.Accounts")
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	ku := testInstance(t)

	// Pairs overlap for spot/margin tests:
	// Only in Spot: BTC-USDT, ETH-USDT
	// In Both: ETH-BTC, LTC-USDT
	// Only in Margin: TRX-BTC, SOL-USDC
	pairs := map[string]currency.Pairs{}
	for a, ss := range map[string][]string{
		"spot":    {"BTC-USDT", "ETH-BTC", "ETH-USDT", "LTC-USDT"},
		"margin":  {"ETH-BTC", "LTC-USDT", "SOL-USDC", "TRX-BTC"},
		"futures": {"ETHUSDCM", "SOLUSDTM", "XBTUSDCM"},
	} {
		for _, s := range ss {
			p, err := currency.NewPairFromString(s)
			require.NoError(t, err, "NewPairFromString must not error")
			pairs[a] = pairs[a].Add(p)
		}
	}
	pairs["both"] = common.SortStrings(pairs["spot"].Add(pairs["margin"]...))

	exp := append(subscription.List{}, expectedPerPairSubscriptions(subscription.TickerChannel, asset.Spot, pairs["both"], marketTickerChannel, 0, nil)...)
	exp = append(exp, expectedPerPairSubscriptions(subscription.TickerChannel, asset.Futures, pairs["futures"], futuresTickerChannel, 0, nil)...)
	exp = append(exp, expectedPerPairSubscriptions(subscription.OrderbookChannel, asset.Spot, pairs["both"], marketOrderbookDepth5Channel, kline.HundredMilliseconds, nil)...)
	exp = append(exp, expectedPerPairSubscriptions(subscription.OrderbookChannel, asset.Futures, pairs["futures"], futuresOrderbookDepth5Channel, kline.HundredMilliseconds, nil)...)
	exp = append(exp, expectedPerPairSubscriptions(subscription.AllTradesChannel, asset.Spot, pairs["both"], marketMatchChannel, 0, nil)...)

	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	testsubs.EqualLists(t, exp, subs)

	ku.Websocket.SetCanUseAuthenticatedEndpoints(true)

	var loanPairs currency.Pairs
	loanCurrs := common.SortStrings(pairs["both"].GetCurrencies())
	for _, c := range loanCurrs {
		loanPairs = append(loanPairs, currency.Pair{Base: c})
	}

	exp = append(exp, subscription.List{
		{Asset: asset.Futures, Channel: futuresTradeOrderChannel, QualifiedChannel: "/contractMarket/tradeOrders", Pairs: pairs["futures"]},
		{Asset: asset.Futures, Channel: futuresStopOrdersLifecycleEventChannel, QualifiedChannel: "/contractMarket/advancedOrders", Pairs: pairs["futures"]},
		{Asset: asset.Futures, Channel: futuresAccountBalanceEventChannel, QualifiedChannel: "/contractAccount/wallet", Pairs: pairs["futures"]},
		{Asset: asset.Margin, Channel: marginPositionChannel, QualifiedChannel: "/margin/position", Pairs: pairs["margin"]},
		{Asset: asset.Margin, Channel: marginLoanChannel, QualifiedChannel: "/margin/loan:" + loanCurrs.Join(), Pairs: loanPairs},
		{Channel: accountBalanceChannel, QualifiedChannel: "/account/balance"},
	}...)

	subs, err = ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions with Auth must not error")
	testsubs.EqualLists(t, exp, subs)
}

func TestGenerateTickerAllSub(t *testing.T) {
	t.Parallel()

	ku := testInstance(t)
	avail, err := ku.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error")
	err = ku.CurrencyPairs.StorePairs(asset.Spot, avail[:11], true)
	require.NoError(t, err, "StorePairs must not error")

	ku.Features.Subscriptions = subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Spot}}
	exp := subscription.List{
		{Channel: subscription.TickerChannel, Asset: asset.Spot, QualifiedChannel: "/market/ticker:all", Pairs: avail[:11]},
	}
	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions with Auth must not error")
	testsubs.EqualLists(t, exp, subs)
}

// TestGenerateOtherSubscriptions exercises non-default subscriptions
func TestGenerateOtherSubscriptions(t *testing.T) {
	t.Parallel()

	ku := testInstance(t)
	spotPairs, err := ku.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	spotPairs = common.SortStrings(spotPairs)
	interval, err := IntervalToString(kline.FourHour)
	require.NoError(t, err, "IntervalToString must not error")

	subs := subscription.List{
		{Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: kline.FourHour},
		{Channel: marketSnapshotChannel, Asset: asset.Spot},
	}

	for _, s := range subs {
		ku.Features.Subscriptions = subscription.List{s}
		got, err := ku.generateSubscriptions()
		require.NoError(t, err, "generateSubscriptions must not error")

		var exp subscription.List
		switch s.Channel {
		case subscription.CandlesChannel:
			exp = expectedPerPairSubscriptions(subscription.CandlesChannel, asset.Spot, spotPairs, marketCandlesChannel, kline.FourHour, func(pair currency.Pair) string {
				return pair.String() + "_" + interval
			})
		case marketSnapshotChannel:
			exp = expectedPerPairSubscriptions(marketSnapshotChannel, asset.Spot, spotPairs, marketSnapshotChannel, 0, nil)
		default:
			t.Fatalf("unexpected test channel %s", s.Channel)
		}

		testsubs.EqualLists(t, exp, got)
	}
}

// TestGenerateMarginSubscriptions is a regression test for #1755 and ensures margin subscriptions work without spot subs
func TestGenerateMarginSubscriptions(t *testing.T) {
	t.Parallel()

	ku := testInstance(t)

	spotAvail, err := ku.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error for spot pairs")
	spotAvail = common.SortStrings(spotAvail)
	marginAvail, err := ku.GetAvailablePairs(asset.Margin)
	require.NoError(t, err, "GetAvailablePairs must not error for margin pairs")
	marginAvail = common.SortStrings(marginAvail)
	require.GreaterOrEqual(t, len(marginAvail), 6, "Margin available pairs must include at least 6 pairs")
	require.GreaterOrEqual(t, len(spotAvail), 3, "Spot available pairs must include at least 3 pairs")

	err = ku.CurrencyPairs.StorePairs(asset.Margin, marginAvail[:6], true)
	require.NoError(t, err, "StorePairs must not error storing margin pairs")
	err = ku.CurrencyPairs.StorePairs(asset.Spot, spotAvail[:3], true)
	require.NoError(t, err, "StorePairs must not error storing spot pairs")

	ku.Features.Subscriptions = subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Margin}}
	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	testsubs.EqualLists(t, expectedPerPairSubscriptions(subscription.TickerChannel, asset.Margin, marginAvail[:6], marketTickerChannel, 0, nil), subs)

	require.NoError(t, ku.CurrencyPairs.SetAssetEnabled(asset.Margin, false), "SetAssetEnabled Spot must not error")
	require.NoError(t, err, "SetAssetEnabled must not error")
	ku.Features.Subscriptions = subscription.List{{Channel: subscription.TickerChannel, Asset: asset.All}}
	subs, err = ku.generateSubscriptions()
	require.NoError(t, err, "mergeMarginPairs must not cause errAssetRecords by adding an empty asset when Margin is disabled")
	require.NotEmpty(t, subs, "generateSubscriptions must return some subs")

	require.NoError(t, ku.CurrencyPairs.SetAssetEnabled(asset.Margin, true), "SetAssetEnabled Margin must not error")
	require.NoError(t, ku.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "SetAssetEnabled Spot must not error")
	require.NoError(t, ku.CurrencyPairs.SetAssetEnabled(asset.Futures, false), "SetAssetEnabled Futures must not error")
	ku.Features.Subscriptions = subscription.List{{Channel: subscription.TickerChannel, Asset: asset.All}}
	subs, err = ku.generateSubscriptions()
	require.NoError(t, err, "mergeMarginPairs must not cause errAssetRecords by adding an empty asset when Spot is disabled")
	require.NotEmpty(t, subs, "generateSubscriptions must return some subs")
}

// TestCheckSubscriptions ensures checkSubscriptions upgrades user config correctly
func TestCheckSubscriptions(t *testing.T) {
	t.Parallel()

	ku := &Exchange{
		Base: exchange.Base{
			Config: &config.Exchange{
				Features: &config.FeaturesConfig{
					Subscriptions: subscription.List{
						{Enabled: true, Channel: "ticker"},
						{Enabled: true, Channel: "allTrades"},
						{Enabled: true, Channel: "orderbook", Interval: kline.HundredMilliseconds},
						{Enabled: true, Channel: "/contractMarket/tickerV2:%s"},
						{Enabled: true, Channel: "/contractMarket/level2Depth50:%s"},
						{Enabled: true, Channel: "/margin/fundingBook:%s", Authenticated: true},
						{Enabled: true, Channel: "/account/balance", Authenticated: true},
						{Enabled: true, Channel: "/margin/position", Authenticated: true},
						{Enabled: true, Channel: "/margin/loan:%s", Authenticated: true},
						{Enabled: true, Channel: "/contractMarket/tradeOrders", Authenticated: true},
						{Enabled: true, Channel: "/contractMarket/advancedOrders", Authenticated: true},
						{Enabled: true, Channel: "/contractAccount/wallet", Authenticated: true},
						{Enabled: true, Channel: "/contractMarket/level2", Asset: asset.Futures},
						{Enabled: true, Channel: "/market/level2", Asset: asset.Spot, Authenticated: true},
					},
				},
			},
			Features: exchange.Features{},
		},
	}

	ku.checkSubscriptions()
	testsubs.EqualLists(t, defaultSubscriptions, ku.Features.Subscriptions)
	testsubs.EqualLists(t, defaultSubscriptions, ku.Config.Features.Subscriptions)
}

func TestProcessOrderbook(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ku := testInstance(t)
		pair, err := currency.NewPairFromString("ETH-BTC")
		require.NoError(t, err, "NewPairFromString must not error")
		assets, err := ku.CalculateAssets(marketOrderbookDepth50Channel, pair)
		require.NoError(t, err, "CalculateAssets must not error")
		require.NotEmpty(t, assets, "must resolve at least one asset for the orderbook pair")

		err = ku.processOrderbook([]byte(`{"asks":[["0.0500","1.5"],["0.0500","0.5"],["0.0600","2"]],"bids":[["0.0400","3"],["0.0400","1"],["0.0300","4"]],"timestamp":1700555340197}`), pair.String(), marketOrderbookDepth50Channel)
		require.NoError(t, err, "processOrderbook must not error")

		for _, a := range assets {
			book, err := ku.Websocket.Orderbook.GetOrderbook(pair, a)
			require.NoErrorf(t, err, "GetOrderbook must not error for asset %s", a)
			require.Len(t, book.Asks, 2, "must collapse duplicate ask levels")
			require.Len(t, book.Bids, 2, "must collapse duplicate bid levels")
			assert.Equal(t, time.UnixMilli(1700555340197), book.LastUpdated, "LastUpdated should match the snapshot timestamp")
			assert.Equal(t, pair, book.Pair, "Pair should match the processed orderbook symbol")
			assert.Equal(t, a, book.Asset, "Asset should match the calculated asset")
			assert.Equal(t, 0.05, book.Asks[0].Price, "First ask price should match payload")
			assert.InDelta(t, 2.0, book.Asks[0].Amount, 1e-12, "First ask amount should merge duplicate rounded levels")
			assert.Equal(t, 0.04, book.Bids[0].Price, "First bid price should match payload")
			assert.InDelta(t, 4.0, book.Bids[0].Amount, 1e-12, "First bid amount should merge duplicate rounded levels")
		}

		err = ku.wsHandleData(t.Context(), nil, []byte(`{"type":"message","topic":"/spotMarket/level2Depth50:ETH-BTC","subject":"level2","data":{"asks":[["0.0700","1.25"]],"bids":[["0.0200","2.5"]],"timestamp":1700555342007}}`))
		require.NoError(t, err, "wsHandleData must not error for orderbook payloads")

		for _, a := range assets {
			book, err := ku.Websocket.Orderbook.GetOrderbook(pair, a)
			require.NoErrorf(t, err, "GetOrderbook must not error for asset %s after wsHandleData", a)
			require.Len(t, book.Asks, 1, "must replace asks on snapshot reload")
			require.Len(t, book.Bids, 1, "must replace bids on snapshot reload")
			assert.Equal(t, time.UnixMilli(1700555342007), book.LastUpdated, "LastUpdated should update from websocket dispatch")
			assert.Equal(t, 0.07, book.Asks[0].Price, "Ask price should match websocket dispatch payload")
			assert.InDelta(t, 1.25, book.Asks[0].Amount, 1e-12, "Ask amount should match websocket dispatch payload")
			assert.Equal(t, 0.02, book.Bids[0].Price, "Bid price should match websocket dispatch payload")
			assert.InDelta(t, 2.5, book.Bids[0].Amount, 1e-12, "Bid amount should match websocket dispatch payload")
		}
	})

	t.Run("last_updated_fallback", func(t *testing.T) {
		t.Parallel()

		ku := testInstance(t)
		pair, err := currency.NewPairFromString("ETH-BTC")
		require.NoError(t, err, "NewPairFromString must not error")
		assets, err := ku.CalculateAssets(marketOrderbookDepth50Channel, pair)
		require.NoError(t, err, "CalculateAssets must not error")
		require.NotEmpty(t, assets, "must resolve at least one asset for the orderbook pair")

		before := time.Now()
		err = ku.processOrderbook([]byte(`{"asks":[["0.0500","1"]],"bids":[["0.0400","1"]]}`), pair.String(), marketOrderbookDepth50Channel)
		after := time.Now()
		require.NoError(t, err, "processOrderbook must not error when timestamp is absent")

		for _, a := range assets {
			book, err := ku.Websocket.Orderbook.GetOrderbook(pair, a)
			require.NoErrorf(t, err, "GetOrderbook must not error for asset %s", a)
			assert.False(t, book.LastUpdated.Before(before), "LastUpdated should not be before the fallback window")
			assert.False(t, book.LastUpdated.After(after), "LastUpdated should not be after the fallback window")
		}
	})

	t.Run("error_paths", func(t *testing.T) {
		t.Parallel()
		t.Run("invalid_json", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)
			err := ku.processOrderbook([]byte(`{"asks":`), "ETH-BTC", marketOrderbookDepth50Channel)
			require.Error(t, err)
		})

		t.Run("invalid_symbol", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)
			err := ku.processOrderbook([]byte(`{"asks":[["0.0500","1"]],"bids":[["0.0400","1"]],"timestamp":1700555340197}`), "a", marketOrderbookDepth50Channel)
			require.ErrorIs(t, err, currency.ErrCreatingPair)
		})

		t.Run("calculate_assets", func(t *testing.T) {
			t.Parallel()
			ku := new(Exchange)
			err := ku.processOrderbook([]byte(`{"asks":[["0.0500","1"]],"bids":[["0.0400","1"]],"timestamp":1700555340197}`), "ETH-BTC", marketOrderbookDepth50Channel)
			require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised)
		})

		t.Run("load_snapshot", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)
			ku.Name = ""
			err := ku.processOrderbook([]byte(`{"asks":[["0.0500","1"]],"bids":[["0.0400","1"]],"timestamp":1700555340197}`), "ETH-BTC", marketOrderbookDepth50Channel)
			require.ErrorIs(t, err, common.ErrExchangeNameNotSet)
		})
	})
}

func TestProcessSpotOrderbookWithDepth(t *testing.T) {
	t.Parallel()

	t.Run("error_paths", func(t *testing.T) {
		t.Parallel()
		t.Run("invalid_instrument", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)
			err := ku.processSpotOrderbookWithDepth(t.Context(), []byte(`{"data":{"changes":{"asks":[["18906","0.00331","14103845"]],"bids":[["18891.9","0.15688","14103847"]]},"sequenceEnd":14103847,"sequenceStart":14103844,"symbol":"BTC-USDT","time":1663747970273}}`), "a")
			require.ErrorIs(t, err, currency.ErrCreatingPair)
		})

		t.Run("invalid_json", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)
			err := ku.processSpotOrderbookWithDepth(t.Context(), []byte(`{"data":`), "BTC-USDT")
			require.Error(t, err)
		})
	})
}

func TestProcessFuturesOrderbookLevel2(t *testing.T) {
	t.Parallel()

	validPayload := []byte(`{"sequence":18,"change":"5000.0,buy,83","timestamp":1551770400000}`)

	t.Run("buy", func(t *testing.T) {
		t.Parallel()
		ku := testInstance(t)
		ku.Name += "-TestProcessFuturesOrderbookLevel2"
		require.False(t, futuresTradablePair.IsEmpty(), "futuresTradablePair must be initialised")

		const updateID = int64(18)
		ku.wsOBUpdateMgr = buffer.NewUpdateManager(&buffer.UpdateManagerParams{
			FetchDelay:    0,
			FetchDeadline: buffer.DefaultWSOrderbookUpdateDeadline,
			FetchOrderbook: func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
				if !p.Equal(futuresTradablePair) {
					return nil, fmt.Errorf("unexpected pair %s", p)
				}
				if a != asset.Futures {
					return nil, fmt.Errorf("unexpected asset %s", a)
				}
				return &orderbook.Book{
					Exchange:     ku.Name,
					Pair:         futuresTradablePair,
					Asset:        asset.Futures,
					Bids:         orderbook.Levels{{Price: 4990, Amount: 1, ID: updateID - 1}},
					Asks:         orderbook.Levels{{Price: 5010, Amount: 1, ID: updateID - 1}},
					LastUpdated:  time.UnixMilli(1551770399000),
					LastPushed:   time.UnixMilli(1551770399000),
					LastUpdateID: updateID - 1,
				}, nil
			},
			CheckPendingUpdate: checkPendingUpdate,
			BufferInstance:     &ku.Websocket.Orderbook,
		})

		err := ku.processFuturesOrderbookLevel2(t.Context(), validPayload, futuresTradablePair.String())
		require.NoError(t, err, "processFuturesOrderbookLevel2 must not error for buy updates")

		require.Eventually(t, func() bool {
			id, err := ku.Websocket.Orderbook.LastUpdateID(futuresTradablePair, asset.Futures)
			return err == nil && id == updateID
		}, time.Second*5, time.Millisecond*50, "futures orderbook buy update must eventually sync")

		book, err := ku.Websocket.Orderbook.GetOrderbook(futuresTradablePair, asset.Futures)
		require.NoError(t, err, "GetOrderbook must not error for futures buy updates")
		require.NotEmpty(t, book.Bids, "bids must not be empty after processing a buy update")
		assert.Equal(t, updateID, book.LastUpdateID, "LastUpdateID should be updated from the websocket sequence")
		assert.Equal(t, 5000.0, book.Bids[0].Price, "Highest bid price should match the buy update")
		assert.InDelta(t, 83.0, book.Bids[0].Amount, 1e-12, "Highest bid amount should match the buy update")
	})

	t.Run("sell", func(t *testing.T) {
		t.Parallel()
		ku := testInstance(t)
		ku.Name += "-TestProcessFuturesOrderbookLevel2Sell"
		require.False(t, futuresTradablePair.IsEmpty(), "futuresTradablePair must be initialised")

		const updateID = int64(18)
		ku.wsOBUpdateMgr = buffer.NewUpdateManager(&buffer.UpdateManagerParams{
			FetchDelay:    0,
			FetchDeadline: buffer.DefaultWSOrderbookUpdateDeadline,
			FetchOrderbook: func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
				if !p.Equal(futuresTradablePair) {
					return nil, fmt.Errorf("unexpected pair %s", p)
				}
				if a != asset.Futures {
					return nil, fmt.Errorf("unexpected asset %s", a)
				}
				return &orderbook.Book{
					Exchange:     ku.Name,
					Pair:         futuresTradablePair,
					Asset:        asset.Futures,
					Bids:         orderbook.Levels{{Price: 4990, Amount: 1, ID: updateID - 1}},
					Asks:         orderbook.Levels{{Price: 5010, Amount: 1, ID: updateID - 1}},
					LastUpdated:  time.UnixMilli(1551770399000),
					LastPushed:   time.UnixMilli(1551770399000),
					LastUpdateID: updateID - 1,
				}, nil
			},
			CheckPendingUpdate: checkPendingUpdate,
			BufferInstance:     &ku.Websocket.Orderbook,
		})

		err := ku.processFuturesOrderbookLevel2(t.Context(), []byte(`{"sequence":18,"change":"5000.0,sell,83","timestamp":1551770400000}`), futuresTradablePair.String())
		require.NoError(t, err, "processFuturesOrderbookLevel2 must not error for sell updates")

		require.Eventually(t, func() bool {
			id, err := ku.Websocket.Orderbook.LastUpdateID(futuresTradablePair, asset.Futures)
			return err == nil && id == updateID
		}, time.Second*5, time.Millisecond*50, "futures orderbook sell update must eventually sync")

		book, err := ku.Websocket.Orderbook.GetOrderbook(futuresTradablePair, asset.Futures)
		require.NoError(t, err, "GetOrderbook must not error for futures sell updates")
		require.NotEmpty(t, book.Asks, "asks must not be empty after processing a sell update")
		assert.Equal(t, updateID, book.LastUpdateID, "LastUpdateID should be updated from the websocket sequence")
		assert.Equal(t, 5000.0, book.Asks[0].Price, "Lowest ask price should match the sell update")
		assert.InDelta(t, 83.0, book.Asks[0].Amount, 1e-12, "Lowest ask amount should match the sell update")
	})

	t.Run("error_paths", func(t *testing.T) {
		t.Parallel()
		t.Run("pair_match", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)
			err := ku.processFuturesOrderbookLevel2(t.Context(), validPayload, "NOTAREALPAIR")
			require.ErrorIs(t, err, currency.ErrPairNotFound)
		})

		t.Run("invalid_json", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)
			err := ku.processFuturesOrderbookLevel2(t.Context(), []byte(`{"sequence":`), futuresTradablePair.String())
			require.Error(t, err)
		})

		t.Run("invalid_change_format", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)
			err := ku.processFuturesOrderbookLevel2(t.Context(), []byte(`{"sequence":18,"change":"5000.0,buy","timestamp":1551770400000}`), futuresTradablePair.String())
			require.ErrorContains(t, err, "unexpected orderbook change format")
		})

		t.Run("invalid_price", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)
			err := ku.processFuturesOrderbookLevel2(t.Context(), []byte(`{"sequence":18,"change":"bad,buy,83","timestamp":1551770400000}`), futuresTradablePair.String())
			require.ErrorContains(t, err, "invalid syntax")
		})

		t.Run("invalid_amount", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)
			err := ku.processFuturesOrderbookLevel2(t.Context(), []byte(`{"sequence":18,"change":"5000.0,buy,bad","timestamp":1551770400000}`), futuresTradablePair.String())
			require.ErrorContains(t, err, "invalid syntax")
		})

		t.Run("invalid_side", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)
			err := ku.processFuturesOrderbookLevel2(t.Context(), []byte(`{"sequence":18,"change":"5000.0,hold,83","timestamp":1551770400000}`), futuresTradablePair.String())
			require.ErrorContains(t, err, "unexpected orderbook side")
		})
	})
}

func TestProcessMarketSnapshot(t *testing.T) {
	t.Parallel()
	ku := testInstance(t)
	testexch.FixtureToDataHandler(t, "testdata/wsMarketSnapshot.json", func(ctx context.Context, b []byte) error { return ku.wsHandleData(ctx, nil, b) })
	ku.Websocket.DataHandler.Close()
	assert.Len(t, ku.Websocket.DataHandler.C, 4, "Should see 4 tickers")
	seenAssetTypes := map[asset.Item]int{}
	for resp := range ku.Websocket.DataHandler.C {
		switch v := resp.Data.(type) {
		case *ticker.Price:
			switch len(ku.Websocket.DataHandler.C) {
			case 3:
				assert.Equal(t, asset.Margin, v.AssetType, "AssetType")
				assert.Equal(t, time.UnixMilli(1700555342007), v.LastUpdated, "datetime")
				assert.Equal(t, 0.004445, v.High, "high")
				assert.Equal(t, 0.004415, v.Last, "lastTradedPrice")
				assert.Equal(t, 0.004191, v.Low, "low")
				assert.Equal(t, currency.NewPairWithDelimiter("TRX", "BTC", "-"), v.Pair, "symbol")
				assert.Equal(t, 13097.3357, v.Volume, "volume")
				assert.Equal(t, 57.44552981, v.QuoteVolume, "volValue")
			case 2, 1:
				assert.Equal(t, time.UnixMilli(1700555340197), v.LastUpdated, "datetime")
				assert.Contains(t, []asset.Item{asset.Spot, asset.Margin}, v.AssetType, "AssetType is Spot or Margin")
				seenAssetTypes[v.AssetType]++
				assert.Equal(t, 1, seenAssetTypes[v.AssetType], "Each Asset Type is sent only once per unique snapshot")
				assert.Equal(t, 0.054846, v.High, "high")
				assert.Equal(t, 0.053778, v.Last, "lastTradedPrice")
				assert.Equal(t, 0.05364, v.Low, "low")
				assert.Equal(t, currency.NewPairWithDelimiter("ETH", "BTC", "-"), v.Pair, "symbol")
				assert.Equal(t, 2958.3139116, v.Volume, "volume")
				assert.Equal(t, 160.7847672784213, v.QuoteVolume, "volValue")
			case 0:
				assert.Equal(t, asset.Spot, v.AssetType, "AssetType")
				assert.Equal(t, time.UnixMilli(1700555342151), v.LastUpdated, "datetime")
				assert.Equal(t, 37750.0, v.High, "high")
				assert.Equal(t, 37366.8, v.Last, "lastTradedPrice")
				assert.Equal(t, 36700.0, v.Low, "low")
				assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USDT", "-"), v.Pair, "symbol")
				assert.Equal(t, 2900.37846402, v.Volume, "volume")
				assert.Equal(t, 108210331.34015164, v.QuoteVolume, "volValue")
			}
		case error:
			t.Error(v)
		default:
			t.Errorf("Got unexpected data: %T %v", v, v)
		}
	}
}

// TestSubscribeBatches ensures that endpoints support batching, contrary to kucoin api docs
func TestSubscribeBatches(t *testing.T) {
	t.Parallel()

	ku := testInstance(t)
	ku.Features.Subscriptions = subscription.List{}
	testexch.SetupWs(t, ku)

	ku.Features.Subscriptions = subscription.List{
		{Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
		{Asset: asset.Futures, Channel: subscription.TickerChannel},
		{Asset: asset.Spot, Channel: marketSnapshotChannel},
	}

	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	spotPairs, err := ku.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error for spot pairs")
	futuresPairs, err := ku.GetEnabledPairs(asset.Futures)
	require.NoError(t, err, "GetEnabledPairs must not error for futures pairs")
	require.Len(t, subs, len(spotPairs)+len(futuresPairs)+len(spotPairs), "generateSubscriptions must return one subscription per enabled pair")

	conn := &ConnectionFixture{messageResponse: `{"id":"019ae225-c584-7b71-a634-489c7249e000","type":"ack"}`}
	err = ku.Subscribe(t.Context(), conn, subs)
	require.NoError(t, err, "Subscribe must not error for small batches")

	expectedOutbound := collapseSubscriptionList(subs)
	require.Len(t, conn.sentRequests, len(expectedOutbound), "Subscribe must send one request per collapsed batch")
	expectedTopics := make(map[string]struct{}, len(expectedOutbound))
	for _, outbound := range expectedOutbound {
		expectedTopics[outbound.QualifiedChannel] = struct{}{}
	}
	for _, req := range conn.sentRequests {
		_, ok := expectedTopics[req.Topic]
		assert.Truef(t, ok, "Request topic should match a collapsed subscription: %s", req.Topic)
	}
	assert.Len(t, ku.Websocket.GetSubscriptions(), len(subs), "Subscribe should track each original subscription")
}

func TestChannelName(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		a   asset.Item
		ch  string
		exp string
	}{
		{asset.Futures, futuresOrderbookDepth50Channel, futuresOrderbookDepth50Channel},
		{asset.Futures, subscription.OrderbookChannel, futuresOrderbookDepth5Channel},
		{asset.Futures, subscription.CandlesChannel, marketCandlesChannel},
		{asset.Futures, subscription.TickerChannel, futuresTickerChannel},
		{asset.Spot, subscription.OrderbookChannel, marketOrderbookDepth5Channel},
		{asset.Spot, subscription.AllTradesChannel, marketMatchChannel},
		{asset.Spot, subscription.CandlesChannel, marketCandlesChannel},
		{asset.Spot, subscription.TickerChannel, marketTickerChannel},
	} {
		assert.Equal(t, tt.exp, channelName(&subscription.Subscription{Channel: tt.ch}, tt.a))
	}
}

// TestSubscribeBatchLimit exercises the kucoin batch limits of 400 per connection
// Ensures batching of 100 pairs and the connection symbol limit is still 400 at Kucoin's end
func TestSubscribeBatchLimit(t *testing.T) {
	t.Parallel()

	const expectedLimit = 400

	setupSubs := func(tb testing.TB, total int) (*Exchange, subscription.List) {
		tb.Helper()
		ku := testInstance(tb)
		ku.Features.Subscriptions = subscription.List{}
		testexch.SetupWs(tb, ku)

		avail, err := ku.GetAvailablePairs(asset.Spot)
		require.NoError(tb, err, "GetAvailablePairs must not error")
		require.GreaterOrEqual(tb, len(avail), total, "GetAvailablePairs must include enough spot pairs")

		err = ku.CurrencyPairs.StorePairs(asset.Spot, avail[:total], true)
		require.NoError(tb, err, "StorePairs must not error")

		ku.Features.Subscriptions = subscription.List{{Asset: asset.Spot, Channel: subscription.AllTradesChannel}}
		subs, err := ku.generateSubscriptions()
		require.NoError(tb, err, "generateSubscriptions must not error")
		return ku, subs
	}

	t.Run("within_limit", func(t *testing.T) {
		t.Parallel()

		ku, subs := setupSubs(t, expectedLimit)
		require.Len(t, subs, expectedLimit, "generateSubscriptions must return one subscription per enabled pair")

		conn := &ConnectionFixture{messageResponse: `{"id":"019ae225-c584-7b71-a634-489c7249e000","type":"ack"}`}
		err := ku.Subscribe(t.Context(), conn, subs)
		require.NoError(t, err, "Subscribe must not error at KuCoin's session limit")
		require.Len(t, conn.sentRequests, 4, "Subscribe must send four batched requests at KuCoin's session limit")
		assert.Len(t, ku.Websocket.GetSubscriptions(), expectedLimit, "Subscribe should track each original subscription at KuCoin's session limit")

		unsubConn := &ConnectionFixture{messageResponse: `{"id":"019ae225-c584-7b71-a634-489c7249e000","type":"ack"}`}
		err = ku.Unsubscribe(t.Context(), unsubConn, subs)
		require.NoError(t, err, "Unsubscribe must not error at KuCoin's session limit")
		require.Len(t, unsubConn.sentRequests, 4, "Unsubscribe must send four batched requests at KuCoin's session limit")
		assert.Empty(t, ku.Websocket.GetSubscriptions(), "Unsubscribe should remove all tracked subscriptions")
	})

	t.Run("over_limit", func(t *testing.T) {
		t.Parallel()

		ku, subs := setupSubs(t, expectedLimit+20)
		require.Len(t, subs, expectedLimit+20, "generateSubscriptions must return one subscription per enabled pair")
		expectedOutbound := collapseSubscriptionList(subs)
		batchSizes := make(map[string]int, len(expectedOutbound))
		for assoc, outbound := range expectedOutbound {
			batchSizes[outbound.QualifiedChannel] = len(*assoc)
		}

		conn := &ConnectionFixture{}
		conn.sendFn = func(_ any) ([]byte, error) {
			if len(conn.sentRequests) == 5 {
				return []byte(`{"id":"019ae22f-4718-7da4-846d-999b085cc24a","type":"error","code":509,"data":"exceed max subscription count limitation of 400 per session"}`), nil
			}
			return []byte(`{"id":"019ae225-c584-7b71-a634-489c7249e000","type":"ack"}`), nil
		}

		err := ku.Subscribe(t.Context(), conn, subs)
		require.Error(t, err, "Subscribe must error once KuCoin's session limit is exceeded")
		require.Len(t, conn.sentRequests, 5, "Subscribe must send five batched requests when more than 400 pairs are enabled")
		assert.ErrorContains(t, err, "exceed max subscription count limitation of 400 per session", "Subscribe should return KuCoin's session limit error")
		failedBatchSize := batchSizes[conn.sentRequests[len(conn.sentRequests)-1].Topic]
		assert.Len(t, ku.Websocket.GetSubscriptions(), len(subs)-failedBatchSize, "Only acknowledged subscriptions should be tracked after KuCoin rejects one batch")
	})
}

// TestSubscribeTickerAll ensures that ticker subscriptions switch to using all and it works
func TestSubscribeTickerAll(t *testing.T) {
	t.Parallel()

	ku := testInstance(t)
	ku.Features.Subscriptions = subscription.List{}
	testexch.SetupWs(t, ku)
	done := make(chan struct{})
	go func() { // drain websocket messages when subscribed to all tickers
		for {
			select {
			case <-done:
				return
			case <-ku.Websocket.DataHandler.C:
			}
		}
	}()
	t.Cleanup(func() {
		close(done)
	})

	avail, err := ku.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error")

	require.GreaterOrEqual(t, len(avail), 500)
	err = ku.CurrencyPairs.StorePairs(asset.Spot, avail[:500], true)
	require.NoError(t, err, "StorePairs must not error")

	ku.Features.Subscriptions = subscription.List{{Asset: asset.Spot, Channel: subscription.TickerChannel}}

	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	require.Len(t, subs, 1, "Must generate one subscription")
	assert.Equal(t, "/market/ticker:all", subs[0].QualifiedChannel, "QualifiedChannel should be correct")

	err = ku.Subscribe(t.Context(), &ConnectionFixture{messageResponse: `{"id":"019ae225-c584-7b71-a634-489c7249e000","type":"ack"}`}, subs)
	assert.NoError(t, err, "Subscribe to should not error")
}

func TestManageSubscriptions(t *testing.T) {
	t.Parallel()

	baseSub := func() *subscription.Subscription {
		pair := currency.NewPairWithDelimiter("BTC", "USDT", "-")
		return &subscription.Subscription{
			Asset:            asset.Spot,
			Channel:          subscription.TickerChannel,
			Pairs:            currency.Pairs{pair},
			QualifiedChannel: marketTickerChannel + ":" + pair.String(),
		}
	}

	t.Run("error_paths", func(t *testing.T) {
		t.Parallel()
		t.Run("send_message", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)
			errExpected := errors.New("send failure")

			err := ku.manageSubscriptions(t.Context(), &ConnectionFixture{messageError: errExpected}, subscription.List{baseSub()}, "subscribe")
			require.ErrorIs(t, err, errExpected)
		})

		t.Run("invalid_json", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)

			err := ku.manageSubscriptions(t.Context(), &ConnectionFixture{messageResponse: `{"type":`}, subscription.List{baseSub()}, "subscribe")
			require.Error(t, err)
		})

		t.Run("error_payload_not_string", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)

			err := ku.manageSubscriptions(t.Context(), &ConnectionFixture{messageResponse: `{"type":"error","code":509,"data":{}}`}, subscription.List{baseSub()}, "subscribe")
			require.Error(t, err)
			assert.ErrorContains(t, err, "unknown error (509)")
		})

		t.Run("unexpected_message_type", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)

			err := ku.manageSubscriptions(t.Context(), &ConnectionFixture{messageResponse: `{"type":"mystery","code":1}`}, subscription.List{baseSub()}, "subscribe")
			require.ErrorIs(t, err, errInvalidMsgType)
			assert.ErrorContains(t, err, "mystery")
		})

		t.Run("subscribe_ack_add_error", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)
			ku.Verbose = true
			sub := baseSub()
			require.NoError(t, sub.SetState(subscription.SubscribedState), "SetState must not error")

			err := ku.manageSubscriptions(t.Context(), &ConnectionFixture{messageResponse: `{"type":"ack"}`}, subscription.List{sub}, "subscribe")
			require.ErrorIs(t, err, subscription.ErrInStateAlready)
		})

		t.Run("unsubscribe_ack_remove_error", func(t *testing.T) {
			t.Parallel()
			ku := testInstance(t)

			err := ku.manageSubscriptions(t.Context(), &ConnectionFixture{messageResponse: `{"type":"ack"}`}, subscription.List{baseSub()}, "unsubscribe")
			require.ErrorIs(t, err, subscription.ErrNotFound)
		})
	})
}

func TestProcessFuturesKline(t *testing.T) {
	t.Parallel()

	ku := new(Exchange)
	require.NoError(t, testexch.Setup(ku), "Test instance Setup must not error")

	data := fmt.Sprintf(`{"symbol":%q,"candles":["1714964400","63815.1","63890.8","63928.5","63797.8","17553.0","17553"],"time":1714964823722}`, futuresTradablePair.String())
	err := ku.processFuturesKline(t.Context(), []byte(data), "1hour")
	require.NoError(t, err)

	select {
	case msg := <-ku.Websocket.DataHandler.C:
		got, ok := msg.Data.(*kline.Item)
		require.True(t, ok, "expected *kline.Item")
		assert.Equal(t, &kline.Item{
			Asset:    asset.Futures,
			Exchange: ku.Name,
			Pair:     futuresTradablePair,
			Interval: kline.OneHour,
			Candles: []kline.Candle{{
				Time:   time.Unix(1714964400, 0),
				Open:   63815.1,
				Close:  63890.8,
				High:   63928.5,
				Low:    63797.8,
				Volume: 17553,
			}},
		}, got)
	default:
		require.Fail(t, "expected websocket kline payload")
	}
}

func TestCollapseSubscriptionList(t *testing.T) {
	t.Parallel()

	const totalSubscriptions = 101

	marketSubs := make(subscription.List, 0, totalSubscriptions)
	expectedPairs := make(currency.Pairs, 0, totalSubscriptions)
	expectedSuffixes := make([]string, 0, totalSubscriptions)
	for i := range totalSubscriptions {
		base := fmt.Sprintf("COIN%03d", i)
		suffix := base + "-USDT"
		pair := currency.NewPairWithDelimiter(base, "USDT", "-")
		marketSubs = append(marketSubs, &subscription.Subscription{
			Channel:          subscription.AllTradesChannel,
			Asset:            asset.Spot,
			Pairs:            currency.Pairs{pair},
			QualifiedChannel: marketMatchChannel + ":" + suffix,
		})
		expectedPairs = append(expectedPairs, pair)
		expectedSuffixes = append(expectedSuffixes, suffix)
	}

	accountSub := &subscription.Subscription{
		Channel:          accountBalanceChannel,
		Authenticated:    true,
		QualifiedChannel: accountBalanceChannel,
	}

	subs := append(subscription.List{}, marketSubs...)
	subs = append(subs, accountSub)

	collapsed := collapseSubscriptionList(subs)
	require.Len(t, collapsed, 3, "collapseSubscriptionList must create three collapsed batches")

	type batchResult struct {
		assoc *subscription.List
		sub   *subscription.Subscription
	}

	var hundredBatch *batchResult
	var singleBatch *batchResult
	var accountBatch *batchResult

	for assoc, sub := range collapsed {
		switch {
		case sub.QualifiedChannel == accountBalanceChannel:
			accountBatch = &batchResult{assoc: assoc, sub: sub}
		case strings.HasPrefix(sub.QualifiedChannel, marketMatchChannel+":"):
			switch len(*assoc) {
			case 100:
				hundredBatch = &batchResult{assoc: assoc, sub: sub}
			case 1:
				singleBatch = &batchResult{assoc: assoc, sub: sub}
			default:
				t.Fatalf("unexpected market batch size: %d", len(*assoc))
			}
		default:
			t.Fatalf("unexpected collapsed channel: %s", sub.QualifiedChannel)
		}
	}

	require.NotNil(t, hundredBatch, "the 100-item market batch must be present")
	require.NotNil(t, singleBatch, "the single-item market batch must be present")
	require.NotNil(t, accountBatch, "the pairless account batch must be present")

	assertCollapsedBatch(t, marketSubs[:100], expectedPairs[:100], expectedSuffixes[:100], hundredBatch.assoc, hundredBatch.sub)
	assertCollapsedBatch(t, marketSubs[100:], expectedPairs[100:], expectedSuffixes[100:], singleBatch.assoc, singleBatch.sub)

	require.Len(t, *accountBatch.assoc, 1, "the pairless account batch must preserve one original subscription")
	assert.Same(t, accountSub, (*accountBatch.assoc)[0], "the pairless account batch should preserve the original subscription pointer")
	assert.Equal(t, accountBalanceChannel, accountBatch.sub.Channel, "the pairless account batch should preserve the original channel")
	assert.Equal(t, accountBalanceChannel, accountBatch.sub.QualifiedChannel, "the pairless account batch should preserve the qualified channel")
	assert.Empty(t, accountBatch.sub.Pairs, "the pairless account batch should not gain pairs")
	assert.True(t, accountBatch.sub.Authenticated, "the pairless account batch should preserve authentication state")

	assert.Len(t, marketSubs[0].Pairs, 1, "the source subscription should remain unchanged after collapsing")
	assert.Equal(t, marketMatchChannel+":"+expectedSuffixes[0], marketSubs[0].QualifiedChannel, "the source subscription should keep its original qualified channel")
}

func assertCollapsedBatch(t *testing.T, expectedOriginal subscription.List, expectedPairs currency.Pairs, expectedSuffixes []string, assoc *subscription.List, got *subscription.Subscription) {
	t.Helper()

	require.NotNil(t, assoc, "the associated subscription list must not be nil")
	require.NotNil(t, got, "the collapsed subscription must not be nil")
	require.Len(t, *assoc, len(expectedOriginal), "the associated subscription list must preserve the original subscriptions")

	for i := range expectedOriginal {
		assert.Samef(t, expectedOriginal[i], (*assoc)[i], "associated subscription %d should match the original pointer", i)
	}

	assert.Equal(t, subscription.AllTradesChannel, got.Channel, "the collapsed subscription should preserve the channel")
	assert.Equal(t, asset.Spot, got.Asset, "the collapsed subscription should preserve the asset")
	assert.Equal(t, expectedPairs, got.Pairs, "the collapsed subscription should merge pairs in order")
	assert.Equal(t, marketMatchChannel+":"+strings.Join(expectedSuffixes, ","), got.QualifiedChannel, "the collapsed subscription should join the qualified channel suffixes")
	assert.False(t, got.Authenticated, "the collapsed market subscription should remain public")
}
