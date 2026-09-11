package currencystate

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestNewCurrencyStates(t *testing.T) {
	if NewCurrencyStates() == nil {
		t.Fatal("unexpected value")
	}
}

func TestGetCurrencyStateSnapshot(t *testing.T) {
	t.Parallel()
	snapshots, err := (*States)(nil).GetCurrencyStateSnapshot()
	require.ErrorIs(t, err, errNilStates, "GetCurrencyStateSnapshot must error correctly for nil states")
	assert.Nil(t, snapshots, "GetCurrencyStateSnapshot should return nil snapshots for nil states")

	snapshots, err = NewCurrencyStates().GetCurrencyStateSnapshot()
	require.NoError(t, err, "GetCurrencyStateSnapshot must not error for empty states")
	assert.Nil(t, snapshots, "GetCurrencyStateSnapshot should preserve a nil snapshot for empty states")

	states := &States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {currency.BTC.Item: &Currency{
				withdrawals: true,
				deposits:    true,
				trading:     false,
			}},
			asset.Futures: {currency.BTC.Item: &Currency{
				withdrawals: true,
				deposits:    false,
				trading:     true,
			}},
		},
	}
	snapshots, err = states.GetCurrencyStateSnapshot()
	require.NoError(t, err, "GetCurrencyStateSnapshot must not error for populated states")
	require.Len(t, snapshots, 2, "GetCurrencyStateSnapshot must return every state")

	type snapshotKey struct {
		asset asset.Item
		code  *currency.Item
	}
	snapshotByKey := make(map[snapshotKey]Snapshot, len(snapshots))
	for i := range snapshots {
		snapshotByKey[snapshotKey{asset: snapshots[i].Asset, code: snapshots[i].Code.Item}] = snapshots[i]
	}
	spotBTC, ok := snapshotByKey[snapshotKey{asset: asset.Spot, code: currency.BTC.Item}]
	require.True(t, ok, "GetCurrencyStateSnapshot must include spot BTC")
	assert.True(t, *spotBTC.Withdraw, "GetCurrencyStateSnapshot should preserve spot BTC withdrawals")
	assert.True(t, *spotBTC.Deposit, "GetCurrencyStateSnapshot should preserve spot BTC deposits")
	assert.False(t, *spotBTC.Trade, "GetCurrencyStateSnapshot should preserve spot BTC trading")
	futuresBTC, ok := snapshotByKey[snapshotKey{asset: asset.Futures, code: currency.BTC.Item}]
	require.True(t, ok, "GetCurrencyStateSnapshot must include futures BTC")
	assert.True(t, *futuresBTC.Withdraw, "GetCurrencyStateSnapshot should preserve futures BTC withdrawals")
	assert.False(t, *futuresBTC.Deposit, "GetCurrencyStateSnapshot should preserve futures BTC deposits")
	assert.True(t, *futuresBTC.Trade, "GetCurrencyStateSnapshot should preserve futures BTC trading")

	*spotBTC.Withdraw = false
	assert.True(t, *spotBTC.Deposit, "GetCurrencyStateSnapshot should return independent option fields")
	assert.False(t, *spotBTC.Trade, "GetCurrencyStateSnapshot should return independent option fields")
	assert.True(t, *futuresBTC.Withdraw, "GetCurrencyStateSnapshot should return independent snapshots")
	btcState := states.m[asset.Spot][currency.BTC.Item].GetState()
	assert.True(t, *btcState.Withdraw, "GetCurrencyStateSnapshot should not expose internal state pointers")
}

func TestCanTradePair(t *testing.T) {
	t.Parallel()
	err := (*States)(nil).CanTradePair(currency.EMPTYPAIR, asset.Empty)
	require.ErrorIs(t, err, errNilStates)

	err = (&States{}).CanTradePair(currency.EMPTYPAIR, asset.Empty)
	require.ErrorIs(t, err, errEmptyCurrency)

	cp := currency.NewBTCUSD()
	err = (&States{}).CanTradePair(cp, asset.Empty)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = (&States{}).CanTradePair(cp, asset.Spot)
	require.NoError(t, err)
	// not found but default to operational

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: {trading: true},
				currency.USD.Item: {trading: true},
			},
		},
	}).CanTradePair(cp, asset.Spot)
	require.NoError(t, err)

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: {trading: false},
				currency.USD.Item: {trading: true},
			},
		},
	}).CanTradePair(cp, asset.Spot)
	require.ErrorIs(t, err, errTradingNotAllowed)

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: {trading: true},
				currency.USD.Item: {trading: false},
			},
		},
	}).CanTradePair(cp, asset.Spot)
	require.ErrorIs(t, err, errTradingNotAllowed)

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: {trading: false},
				currency.USD.Item: {trading: false},
			},
		},
	}).CanTradePair(cp, asset.Spot)
	require.ErrorIs(t, err, errTradingNotAllowed)
}

func TestStatesCanTrade(t *testing.T) {
	t.Parallel()
	err := (*States)(nil).CanTrade(currency.EMPTYCODE, asset.Empty)
	require.ErrorIs(t, err, errNilStates)

	err = (&States{}).CanTrade(currency.EMPTYCODE, asset.Empty)
	require.ErrorIs(t, err, errEmptyCurrency)
}

func TestStatesCanWithdraw(t *testing.T) {
	t.Parallel()
	err := (*States)(nil).CanWithdraw(currency.EMPTYCODE, asset.Empty)
	require.ErrorIs(t, err, errNilStates)

	err = (&States{}).CanWithdraw(currency.EMPTYCODE, asset.Empty)
	require.ErrorIs(t, err, errEmptyCurrency)

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: {withdrawals: true},
			},
		},
	}).CanWithdraw(currency.BTC, asset.Spot)
	require.NoError(t, err)

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: {},
			},
		},
	}).CanWithdraw(currency.BTC, asset.Spot)
	require.ErrorIs(t, err, errWithdrawalsNotAllowed)
}

func TestStatesCanDeposit(t *testing.T) {
	t.Parallel()
	err := (*States)(nil).CanDeposit(currency.EMPTYCODE, asset.Empty)
	require.ErrorIs(t, err, errNilStates)

	err = (&States{}).CanDeposit(currency.EMPTYCODE, asset.Empty)
	require.ErrorIs(t, err, errEmptyCurrency)

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: {deposits: true},
			},
		},
	}).CanDeposit(currency.BTC, asset.Spot)
	require.NoError(t, err)

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: {},
			},
		},
	}).CanDeposit(currency.BTC, asset.Spot)
	require.ErrorIs(t, err, errDepositNotAllowed)
}

func TestStatesUpdateAll(t *testing.T) {
	t.Parallel()
	err := (*States)(nil).UpdateAll(asset.Empty, nil)
	require.ErrorIs(t, err, errNilStates)

	err = (&States{}).UpdateAll(asset.Empty, nil)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = (&States{}).UpdateAll(asset.Spot, nil)
	require.ErrorIs(t, err, errUpdatesAreNil)

	s := &States{
		m: map[asset.Item]map[*currency.Item]*Currency{},
	}

	err = s.UpdateAll(asset.Spot, map[currency.Code]Options{
		currency.BTC: {
			Withdraw: convert.BoolPtr(true),
			Trade:    convert.BoolPtr(true),
			Deposit:  convert.BoolPtr(true),
		},
	})
	require.NoError(t, err)

	err = s.UpdateAll(asset.Spot, map[currency.Code]Options{currency.BTC: {
		Withdraw: convert.BoolPtr(false),
		Deposit:  convert.BoolPtr(false),
		Trade:    convert.BoolPtr(false),
	}})
	require.NoError(t, err)

	c, err := s.Get(currency.BTC, asset.Spot)
	require.NoError(t, err)

	if c.CanDeposit() || c.CanTrade() || c.CanWithdraw() {
		t.Fatal()
	}
}

func TestStatesUpdate(t *testing.T) {
	t.Parallel()
	err := (*States)(nil).Update(currency.EMPTYCODE, asset.Empty, Options{})
	require.ErrorIs(t, err, errNilStates)

	err = (&States{}).Update(currency.EMPTYCODE, asset.Empty, Options{})
	require.ErrorIs(t, err, errEmptyCurrency)

	err = (&States{}).Update(currency.BTC, asset.Empty, Options{})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {currency.BTC.Item: &Currency{}},
		},
	}).Update(currency.BTC, asset.Spot, Options{})
	require.NoError(t, err)
}

func TestStatesGet(t *testing.T) {
	t.Parallel()
	_, err := (*States)(nil).Get(currency.EMPTYCODE, asset.Empty)
	require.ErrorIs(t, err, errNilStates)

	_, err = (&States{}).Get(currency.EMPTYCODE, asset.Empty)
	require.ErrorIs(t, err, errEmptyCurrency)

	_, err = (&States{}).Get(currency.BTC, asset.Empty)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = (&States{}).Get(currency.BTC, asset.Spot)
	require.ErrorIs(t, err, ErrCurrencyStateNotFound)
}

func TestCurrencyGetState(t *testing.T) {
	o := (&Currency{}).GetState()
	if *o.Deposit || *o.Trade || *o.Withdraw {
		t.Fatal("unexpected values")
	}
}

func TestAlerting(_ *testing.T) {
	c := Currency{}
	var start, finish sync.WaitGroup
	start.Add(3)
	finish.Add(3)
	go waitForAlert(c.WaitTrading(nil), &start, &finish)
	go waitForAlert(c.WaitDeposit(nil), &start, &finish)
	go waitForAlert(c.WaitWithdraw(nil), &start, &finish)
	start.Wait()
	c.update(Options{
		Trade:    convert.BoolPtr(true),
		Withdraw: convert.BoolPtr(true),
		Deposit:  convert.BoolPtr(true),
	})
	finish.Wait()
}

func waitForAlert(ch <-chan bool, start, finish *sync.WaitGroup) {
	defer finish.Done()
	start.Done()
	<-ch
}
