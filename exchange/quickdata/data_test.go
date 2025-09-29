package quickdata

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

func TestHandleWSAccountChange(t *testing.T) {
	t.Parallel()
	q := mustQuickData(t, AccountHoldingsFocusType)
	require.ErrorIs(t, q.handleWSAccountChange(nil), common.ErrNilPointer)

	d := &account.Change{
		AssetType: q.key.Asset,
		Balance: &account.Balance{
			Currency:               currency.BTC,
			Total:                  1337,
			Hold:                   1337,
			Free:                   1337,
			AvailableWithoutBorrow: 1337,
			Borrowed:               1337,
			UpdatedAt:              time.Now(),
		},
	}
	require.NoError(t, q.handleWSAccountChange(d))
	require.Len(t, q.data.AccountBalance, 1)
	assert.Equal(t, d.Balance, &q.data.AccountBalance[0])

	d2 := &account.Change{
		AssetType: asset.Binary,
		Balance: &account.Balance{
			Currency:               currency.BTC,
			Total:                  1,
			Hold:                   1,
			Free:                   1,
			AvailableWithoutBorrow: 1,
			Borrowed:               1,
			UpdatedAt:              time.Now(),
		},
	}
	require.NoError(t, q.handleWSAccountChange(d2))
	require.Len(t, q.data.AccountBalance, 1)
	assert.NotEqual(t, d2.Balance, &q.data.AccountBalance[0])
}

func TestHandleWSAccountChanges(t *testing.T) {
	t.Parallel()
	q := mustQuickData(t, AccountHoldingsFocusType)
	require.NoError(t, q.handleWSAccountChanges(nil))

	d := account.Change{
		AssetType: q.key.Asset,
		Balance: &account.Balance{
			Currency:               currency.BTC,
			Total:                  1337,
			Hold:                   1337,
			Free:                   1337,
			AvailableWithoutBorrow: 1337,
			Borrowed:               1337,
			UpdatedAt:              time.Now(),
		},
	}
	require.NoError(t, q.handleWSAccountChanges([]account.Change{d}))
	require.Len(t, q.data.AccountBalance, 1)
	assert.Equal(t, d.Balance, &q.data.AccountBalance[0])

	d2 := account.Change{
		AssetType: asset.Binary,
		Balance: &account.Balance{
			Currency:               currency.BTC,
			Total:                  1,
			Hold:                   1,
			Free:                   1,
			AvailableWithoutBorrow: 1,
			Borrowed:               1,
			UpdatedAt:              time.Now(),
		},
	}
	require.NoError(t, q.handleWSAccountChanges([]account.Change{d2}))
	require.Len(t, q.data.AccountBalance, 1)
	assert.NotEqual(t, d2.Balance, &q.data.AccountBalance[0])
}

func TestHandleWSOrderDetail(t *testing.T) {
	t.Parallel()
	q := mustQuickData(t, ActiveOrdersFocusType)
	require.ErrorIs(t, q.handleWSOrderDetail(nil), common.ErrNilPointer)

	d := &order.Detail{
		AssetType: q.key.Asset,
		Amount:    1337,
		Pair:      q.key.Pair(),
	}
	require.NoError(t, q.handleWSOrderDetail(d))
	require.Len(t, q.data.Orders, 1)
	assert.Equal(t, d.Amount, q.data.Orders[0].Amount)

	d2 := &order.Detail{
		AssetType: asset.Binary,
		Amount:    1,
		Pair:      currency.NewBTCUSDT(),
	}
	require.NoError(t, q.handleWSOrderDetail(d2))
	require.Len(t, q.data.Orders, 1)
	assert.NotEqual(t, d2.Amount, q.data.Orders[0].Amount)
}

func TestHandleWSOrderDetails(t *testing.T) {
	t.Parallel()
	q := mustQuickData(t, ActiveOrdersFocusType)

	d := []order.Detail{
		{
			AssetType: q.key.Asset,
			Amount:    1337,
			Pair:      q.key.Pair(),
		},
	}
	require.NoError(t, q.handleWSOrderDetails(d))
	require.Len(t, q.data.Orders, 1)
	assert.Equal(t, d[0].Amount, q.data.Orders[0].Amount)

	d2 := []order.Detail{
		{
			AssetType: asset.Binary,
			Amount:    1,
			Pair:      currency.NewBTCUSDT(),
		},
	}
	require.NoError(t, q.handleWSOrderDetails(d2))
	require.Len(t, q.data.Orders, 1)
	assert.NotEqual(t, d2[0].Amount, q.data.Orders[0].Amount)
}

func TestAccountHoldingsFocusType(t *testing.T) {
	t.Parallel()
	if apiKey == "abc" || apiSecret == "123" {
		t.Skip("API credentials not set; skipping test that requires them")
	}
	qs := mustQuickData(t, AccountHoldingsFocusType)
	f, err := qs.GetFocusByKey(AccountHoldingsFocusType)
	require.NoError(t, err)
	require.NotNil(t, f)
	ctx := account.DeployCredentialsToContext(t.Context(), &account.Credentials{
		Key:    apiKey,
		Secret: apiSecret,
	})
	require.NoError(t, qs.handleFocusType(ctx, f.focusType, f))
	require.NotEmpty(t, qs.data.AccountBalance)
}

func TestHandleWSTickers(t *testing.T) {
	t.Parallel()
	q := mustQuickData(t, TickerFocusType)

	require.NoError(t, q.handleWSTickers(nil))
	assert.Nil(t, q.data.Ticker)
	require.NoError(t, q.handleWSTickers([]ticker.Price{}))
	assert.Nil(t, q.data.Ticker)

	solo := ticker.Price{
		AssetType:    q.key.Asset,
		Pair:         q.key.Pair(),
		Last:         100,
		ExchangeName: q.key.Exchange,
	}
	require.NoError(t, q.handleWSTickers([]ticker.Price{solo}))
	require.NotNil(t, q.data.Ticker)
	assert.Equal(t, solo.Last, q.data.Ticker.Last)

	mismatch := ticker.Price{
		AssetType: asset.Binary,
		Pair:      currency.NewBTCUSDT(),
		Last:      1,
	}
	match := ticker.Price{
		AssetType:    q.key.Asset,
		Pair:         q.key.Pair(),
		Last:         200,
		ExchangeName: q.key.Exchange,
	}
	require.NoError(t, q.handleWSTickers([]ticker.Price{mismatch, match}))
	require.NotNil(t, q.data.Ticker)
	assert.Equal(t, match.Last, q.data.Ticker.Last)
	assert.NotEqual(t, mismatch.Last, q.data.Ticker.Last)

	prev := *q.data.Ticker
	noMatch1 := ticker.Price{AssetType: asset.Binary, Pair: currency.NewBTCUSDT(), Last: 300}
	noMatch2 := ticker.Price{AssetType: asset.Binary, Pair: currency.NewPair(currency.BTC, currency.USDC), Last: 400}
	require.NoError(t, q.handleWSTickers([]ticker.Price{noMatch1, noMatch2}))
	require.NotNil(t, q.data.Ticker)
	assert.Equal(t, prev.Last, q.data.Ticker.Last)
}

// Newly added websocket handler tests
func TestHandleWSTicker(t *testing.T) {
	t.Parallel()
	q := mustQuickData(t, TickerFocusType)
	require.ErrorIs(t, q.handleWSTicker(nil), common.ErrNilPointer)
	p := &ticker.Price{AssetType: q.key.Asset, Pair: q.key.Pair(), Last: 999}
	require.NoError(t, q.handleWSTicker(p))
	require.NotNil(t, q.data.Ticker)
	assert.Equal(t, 999.0, q.data.Ticker.Last)
}

func TestHandleWSOrderbook(t *testing.T) {
	q := mustQuickData(t, OrderBookFocusType)
	require.ErrorIs(t, q.handleWSOrderbook(nil), common.ErrNilPointer)
	id, _ := uuid.NewV4()
	depth := orderbook.NewDepth(id)
	bk := &orderbook.Book{
		Bids:        orderbook.Levels{{Price: 10, Amount: 1}},
		Asks:        orderbook.Levels{{Price: 11, Amount: 2}},
		Exchange:    q.key.Exchange,
		Asset:       q.key.Asset,
		Pair:        q.key.Pair(),
		LastUpdated: time.Now(),
	}
	depth.AssignOptions(bk)
	require.NoError(t, depth.LoadSnapshot(bk))
	require.NoError(t, q.handleWSOrderbook(depth))
	require.NotNil(t, q.data.Orderbook)
	assert.Len(t, q.data.Orderbook.Bids, 1)
}

func TestHandleWSTrade(t *testing.T) {
	q := mustQuickData(t, TradesFocusType)
	require.ErrorIs(t, q.handleWSTrade(nil), common.ErrNilPointer)
	trd := &trade.Data{
		Exchange:     q.key.Exchange,
		CurrencyPair: q.key.Pair(),
		AssetType:    q.key.Asset,
		Price:        123.45,
		Amount:       0.5,
		Timestamp:    time.Now(),
	}
	require.NoError(t, q.handleWSTrade(trd))
	require.Len(t, q.data.Trades, 1)
	assert.Equal(t, 123.45, q.data.Trades[0].Price)
}

func TestHandleWSTrades(t *testing.T) {
	q := mustQuickData(t, TradesFocusType)
	require.NoError(t, q.handleWSTrades(nil))
	require.Empty(t, q.data.Trades)
	trs := []trade.Data{
		{
			Exchange:     q.key.Exchange,
			CurrencyPair: q.key.Pair(),
			AssetType:    q.key.Asset,
			Price:        1,
			Amount:       1,
			Timestamp:    time.Now(),
		},
		{
			Exchange:     q.key.Exchange,
			CurrencyPair: q.key.Pair(),
			AssetType:    q.key.Asset,
			Price:        2,
			Amount:       2,
			Timestamp:    time.Now(),
		},
	}
	require.NoError(t, q.handleWSTrades(trs))
	require.Len(t, q.data.Trades, 2)
	assert.Equal(t, 2.0, q.data.Trades[1].Price)
}
