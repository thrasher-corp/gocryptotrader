package account

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var happyCredentials = &Credentials{Key: "AAAAA"}

func TestCollectBalances(t *testing.T) {
	t.Parallel()
	accounts, err := CollectBalances(
		map[string][]Balance{
			"someAccountID": {
				{Currency: currency.BTC, Total: 40000, Hold: 1},
			},
		},
		asset.Spot,
	)
	subAccount := accounts[0]
	balance := subAccount.Currencies[0]
	if subAccount.ID != "someAccountID" {
		t.Error("subAccount ID not set correctly")
	}
	if subAccount.AssetType != asset.Spot {
		t.Error("subAccount AssetType not set correctly")
	}
	if balance.Currency != currency.BTC || balance.Total != 40000 || balance.Hold != 1 {
		t.Error("subAccount currency balance not set correctly")
	}
	if err != nil {
		t.Error("err is not expected")
	}

	accounts, err = CollectBalances(map[string][]Balance{}, asset.Spot)
	if len(accounts) != 0 {
		t.Error("accounts should be empty")
	}
	if err != nil {
		t.Error("err is not expected")
	}

	accounts, err = CollectBalances(nil, asset.Spot)
	if len(accounts) != 0 {
		t.Error("accounts should be empty")
	}
	if err == nil {
		t.Errorf("expecting err %s", errAccountBalancesIsNil.Error())
	}

	_, err = CollectBalances(map[string][]Balance{}, asset.Empty)
	require.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetHoldings(t *testing.T) {
	err := dispatch.Start(dispatch.DefaultMaxWorkers, dispatch.DefaultJobsLimit)
	require.NoError(t, err)
	err = Process(nil, nil)
	assert.ErrorIs(t, err, errHoldingsIsNil)

	err = Process(&Holdings{}, nil)
	assert.ErrorIs(t, err, errExchangeNameUnset)

	holdings := Holdings{Exchange: "Test"}

	err = Process(&holdings, nil)
	assert.ErrorIs(t, err, errCredentialsAreNil)

	err = Process(&holdings, happyCredentials)
	require.NoError(t, err)

	err = Process(&Holdings{
		Exchange: "Test",
		Accounts: []SubAccount{
			{
				ID: "1337",
			},
		},
	}, happyCredentials)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	err = Process(&Holdings{
		Exchange: "Test",
		Accounts: []SubAccount{
			{
				AssetType: asset.UpsideProfitContract,
				ID:        "1337",
			},
			{
				AssetType: asset.Spot,
				ID:        "1337",
				Currencies: []Balance{
					{
						Currency: currency.BTC,
						Total:    100,
						Hold:     20,
					},
				},
			},
		},
	}, happyCredentials)
	assert.NoError(t, err)

	_, err = GetHoldings("", nil, asset.Spot)
	assert.ErrorIs(t, err, errExchangeNameUnset)

	_, err = GetHoldings("bla", nil, asset.Spot)
	assert.ErrorIs(t, err, errCredentialsAreNil)

	_, err = GetHoldings("bla", happyCredentials, asset.Spot)
	assert.ErrorIs(t, err, ErrExchangeHoldingsNotFound)

	_, err = GetHoldings("bla", happyCredentials, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = GetHoldings("Test", happyCredentials, asset.UpsideProfitContract)
	assert.ErrorIs(t, err, ErrExchangeHoldingsNotFound)

	_, err = GetHoldings("Test", &Credentials{Key: "BBBBB"}, asset.Spot)
	assert.ErrorIs(t, err, errNoCredentialBalances)

	u, err := GetHoldings("Test", happyCredentials, asset.Spot)
	require.NoError(t, err)

	assert.Equal(t, "test", u.Exchange)
	require.Len(t, u.Accounts, 1)
	assert.Equal(t, "1337", u.Accounts[0].ID)
	assert.Equal(t, asset.Spot, u.Accounts[0].AssetType)
	require.Len(t, u.Accounts[0].Currencies, 1)
	assert.Equal(t, currency.BTC, u.Accounts[0].Currencies[0].Currency)
	assert.Equal(t, 100.0, u.Accounts[0].Currencies[0].Total)
	assert.Equal(t, 20.0, u.Accounts[0].Currencies[0].Hold)

	_, err = SubscribeToExchangeAccount("nonsense")
	require.NoError(t, err)

	p, err := SubscribeToExchangeAccount("Test")
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func(p dispatch.Pipe, wg *sync.WaitGroup) {
		for range 2 {
			c := time.NewTimer(time.Second)
			select {
			case <-p.Channel():
			case <-c.C:
			}
		}
		wg.Done()
	}(p, &wg)

	err = Process(&Holdings{
		Exchange: "Test",
		Accounts: []SubAccount{{
			ID:        "1337",
			AssetType: asset.MarginFunding,
			Currencies: []Balance{
				{
					Currency: currency.BTC,
					Total:    100000,
					Hold:     20,
				},
			},
		}},
	}, happyCredentials)
	assert.NoError(t, err)

	wg.Wait()
}

func TestGetBalance(t *testing.T) {
	t.Parallel()

	_, err := GetBalance("", "", nil, asset.Empty, currency.Code{})
	assert.ErrorIs(t, err, errExchangeNameUnset)

	_, err = GetBalance("bruh", "", nil, asset.Empty, currency.Code{})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = GetBalance("bruh", "", nil, asset.Spot, currency.Code{})
	assert.ErrorIs(t, err, errCredentialsAreNil)

	_, err = GetBalance("bruh", "", happyCredentials, asset.Spot, currency.Code{})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = GetBalance("bruh", "", happyCredentials, asset.Spot, currency.BTC)
	assert.ErrorIs(t, err, ErrExchangeHoldingsNotFound)

	err = Process(&Holdings{
		Exchange: "bruh",
		Accounts: []SubAccount{
			{
				AssetType: asset.Spot,
				ID:        "1337",
			},
		},
	}, happyCredentials)
	require.NoError(t, err, "process must not error")

	_, err = GetBalance("bruh", "1336", &Credentials{Key: "BBBBB"}, asset.Spot, currency.BTC)
	assert.ErrorIs(t, err, errNoCredentialBalances)

	_, err = GetBalance("bruh", "1336", happyCredentials, asset.Spot, currency.BTC)
	assert.ErrorIs(t, err, errNoExchangeSubAccountBalances)

	_, err = GetBalance("bruh", "1337", happyCredentials, asset.Futures, currency.BTC)
	assert.ErrorIs(t, err, errNoExchangeSubAccountBalances)

	err = Process(&Holdings{
		Exchange: "bruh",
		Accounts: []SubAccount{
			{
				AssetType: asset.Spot,
				ID:        "1337",
				Currencies: []Balance{
					{
						Currency: currency.BTC,
						Total:    2,
						Hold:     1,
					},
				},
			},
		},
	}, happyCredentials)
	require.NoError(t, err, "process must not error")

	bal, err := GetBalance("bruh", "1337", happyCredentials, asset.Spot, currency.BTC)
	require.NoError(t, err, "get balance must not error")

	bal.m.Lock()
	assert.Equal(t, 2.0, bal.total)
	assert.Equal(t, 1.0, bal.hold)
	bal.m.Unlock()
}

func TestBalanceInternalWait(t *testing.T) {
	t.Parallel()
	var bi *ProtectedBalance
	_, _, err := bi.Wait(0)
	require.ErrorIs(t, err, errBalanceIsNil)

	bi = &ProtectedBalance{}
	waiter, _, err := bi.Wait(time.Nanosecond)
	require.NoError(t, err)

	if !<-waiter {
		t.Fatal("should been alerted by timeout")
	}

	waiter, _, err = bi.Wait(0)
	require.NoError(t, err)

	go bi.notice.Alert()
	if <-waiter {
		t.Fatal("should have been alerted by change notice")
	}
}

func TestBalanceInternalLoad(t *testing.T) {
	t.Parallel()
	bi := &ProtectedBalance{}
	err := bi.load(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "should error nil pointer correctly")

	err = bi.load(&Balance{Total: 1, Hold: 2, Free: 3, AvailableWithoutBorrow: 4, Borrowed: 5})
	assert.ErrorIs(t, err, errUpdatedAtIsZero, "should error correctly when updatedAt is not set")

	now := time.Now()
	err = bi.load(&Balance{UpdatedAt: now, Total: 1, Hold: 2, Free: 3, AvailableWithoutBorrow: 4, Borrowed: 5})
	require.NoError(t, err)

	bi.m.Lock()
	assert.Equal(t, now, bi.updatedAt)
	assert.Equal(t, 1.0, bi.total)
	assert.Equal(t, 2.0, bi.hold)
	assert.Equal(t, 3.0, bi.free)
	assert.Equal(t, 4.0, bi.availableWithoutBorrow)
	assert.Equal(t, 5.0, bi.borrowed)
	bi.m.Unlock()

	assert.Equal(t, 3.0, bi.GetFree())

	err = bi.load(&Balance{UpdatedAt: now, Total: 2, Hold: 3, Free: 4, AvailableWithoutBorrow: 5, Borrowed: 6})
	assert.ErrorIs(t, err, errOutOfSequence, "should error correctly with same UpdatedAt")

	err = bi.load(&Balance{UpdatedAt: now.Add(time.Second), Total: 2, Hold: 3, Free: 4, AvailableWithoutBorrow: 5, Borrowed: 6})
	assert.NoError(t, err)
}

func TestGetFree(t *testing.T) {
	t.Parallel()
	var bi *ProtectedBalance
	if bi.GetFree() != 0 {
		t.Fatal("unexpected value")
	}
	bi = &ProtectedBalance{}
	bi.free = 1
	if bi.GetFree() != 1 {
		t.Fatal("unexpected value")
	}
}

func TestSave(t *testing.T) {
	t.Parallel()
	s := &Service{exchangeAccounts: make(map[string]*Accounts), mux: dispatch.GetNewMux(nil)}
	err := s.Save(nil, nil)
	assert.ErrorIs(t, err, errHoldingsIsNil)

	err = s.Save(&Holdings{}, nil)
	assert.ErrorIs(t, err, errExchangeNameUnset)

	err = s.Save(&Holdings{
		Exchange: "TeSt",
		Accounts: []SubAccount{
			{
				AssetType: 6969,
				ID:        "1337",
				Currencies: []Balance{
					{
						Currency: currency.BTC,
						Total:    100,
						Hold:     20,
					},
				},
			},
			{AssetType: asset.UpsideProfitContract, ID: "1337"},
		},
	}, happyCredentials)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	err = s.Save(&Holdings{ // No change
		Exchange: "tEsT",
		Accounts: []SubAccount{
			{
				AssetType: asset.Spot,
				ID:        "1337",
				Currencies: []Balance{
					{
						Currency: currency.BTC,
						Total:    100,
						Hold:     20,
					},
				},
			},
		},
	}, happyCredentials)
	require.NoError(t, err)

	acc, ok := s.exchangeAccounts["test"]
	require.True(t, ok)

	assets, ok := acc.subAccounts[*happyCredentials][key.SubAccountAsset{
		SubAccount: "1337",
		Asset:      asset.Spot,
	}]
	require.True(t, ok)

	b, ok := assets[currency.BTC.Item]
	require.True(t, ok)

	assert.NotEmpty(t, b.updatedAt)
	assert.Equal(t, 100.0, b.total)
	assert.Equal(t, 20.0, b.hold)

	err = s.Save(&Holdings{
		Exchange: "tEsT",
		Accounts: []SubAccount{
			{
				AssetType: asset.Spot,
				ID:        "1337",
				Currencies: []Balance{
					{
						Currency: currency.ETH,
						Total:    80,
						Hold:     20,
					},
				},
			},
		},
	}, happyCredentials)
	require.NoError(t, err)

	b, ok = assets[currency.BTC.Item]
	require.True(t, ok)
	assert.NotEmpty(t, b.updatedAt)
	assert.Zero(t, b.total)
	assert.Zero(t, b.hold)

	e, ok := assets[currency.ETH.Item]
	require.True(t, ok)
	assert.NotEmpty(t, e.updatedAt)
	assert.Equal(t, 80.0, e.total)
	assert.Equal(t, 20.0, e.hold)
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	s := &Service{exchangeAccounts: make(map[string]*Accounts), mux: dispatch.GetNewMux(nil)}
	err := s.Update("", nil, nil)
	assert.ErrorIs(t, err, errExchangeNameUnset)

	err = s.Update("test", nil, nil)
	assert.ErrorIs(t, err, errCredentialsAreNil)

	err = s.Update("test", []Change{
		{
			AssetType: 6969,
			Balance: &Balance{
				Currency: currency.BTC,
				Free:     100,
			},
		},
	}, happyCredentials)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	now := time.Now()
	err = s.Update("test", []Change{
		{
			AssetType: asset.Spot,
			Account:   "1337",
			Balance: &Balance{
				Currency:  currency.BTC,
				Total:     100,
				Free:      80,
				UpdatedAt: now,
			},
		},
	}, happyCredentials)
	require.NoError(t, err)

	acc, ok := s.exchangeAccounts["test"]
	require.True(t, ok, "Update must add the exchange")

	assets, ok := acc.subAccounts[*happyCredentials][key.SubAccountAsset{
		SubAccount: "1337",
		Asset:      asset.Spot,
	}]
	require.True(t, ok, "Update must add subAccount for the credentials")

	b, ok := assets[currency.BTC.Item]
	require.True(t, ok, "Update must add currency to the subAccount")

	assert.Equal(t, 100.0, b.total, "Update should set total correctly")
	assert.Equal(t, 80.0, b.free, "Update should set free correctly")
	assert.Equal(t, now, b.updatedAt, "Update should set updatedAt correctly")

	err = s.Update("test", []Change{
		{
			AssetType: asset.Spot,
			Account:   "1337",
			Balance: &Balance{
				Currency:  currency.BTC,
				Total:     100,
				Free:      100,
				UpdatedAt: now.Add(-1 * time.Second),
			},
		},
	}, happyCredentials)
	assert.ErrorIs(t, err, errOutOfSequence)

	err = s.Update("test", []Change{
		{
			AssetType: asset.Spot,
			Account:   "1337",
			Balance: &Balance{
				Currency:  currency.BTC,
				Total:     100,
				Free:      100,
				UpdatedAt: now.Add(1 * time.Second),
			},
		},
	}, happyCredentials)
	require.NoError(t, err)

	assert.Equal(t, 100.0, b.total)
	assert.Equal(t, 100.0, b.free)
	assert.Equal(t, now.Add(1*time.Second), b.updatedAt)
}

func TestTrackNewAccounts(t *testing.T) {
	t.Parallel()
	s := &Service{
		exchangeAccounts: make(map[string]*Accounts),
		mux:              dispatch.GetNewMux(nil),
	}

	s.mu.Lock()
	_, err := s.initAccounts("binance")
	s.mu.Unlock()
	require.NoError(t, err)

	s.mu.Lock()
	_, err = s.initAccounts("binance")
	s.mu.Unlock()
	assert.ErrorIs(t, err, errExchangeAlreadyExists)
}
