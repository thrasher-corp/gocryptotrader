package account

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}
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
			}},
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
			}},
	}, happyCredentials)
	assert.NoError(t, err)

	// process again with no changes
	err = Process(&Holdings{
		Exchange: "Test",
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
			}},
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
	assert.ErrorIs(t, err, errExchangeAccountsNotFound)

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
	_, err := GetBalance("", "", nil, asset.Empty, currency.Code{})
	if !errors.Is(err, errExchangeNameUnset) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeNameUnset)
	}

	_, err = GetBalance("bruh", "", nil, asset.Empty, currency.Code{})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	_, err = GetBalance("bruh", "", nil, asset.Spot, currency.Code{})
	if !errors.Is(err, errCredentialsAreNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errCredentialsAreNil)
	}

	_, err = GetBalance("bruh", "", happyCredentials, asset.Spot, currency.Code{})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, currency.ErrCurrencyCodeEmpty)
	}

	_, err = GetBalance("bruh", "", happyCredentials, asset.Spot, currency.BTC)
	if !errors.Is(err, ErrExchangeHoldingsNotFound) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrExchangeHoldingsNotFound)
	}

	err = Process(&Holdings{
		Exchange: "bruh",
		Accounts: []SubAccount{
			{
				AssetType: asset.Spot,
				ID:        "1337",
			},
		},
	}, happyCredentials)
	if err != nil {
		t.Error(err)
	}

	_, err = GetBalance("bruh", "1336", &Credentials{Key: "BBBBB"}, asset.Spot, currency.BTC)
	if !errors.Is(err, errNoCredentialBalances) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoCredentialBalances)
	}

	_, err = GetBalance("bruh", "1336", happyCredentials, asset.Spot, currency.BTC)
	if !errors.Is(err, errNoExchangeSubAccountBalances) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoExchangeSubAccountBalances)
	}

	_, err = GetBalance("bruh", "1337", happyCredentials, asset.Futures, currency.BTC)
	if !errors.Is(err, errNoExchangeSubAccountBalances) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoExchangeSubAccountBalances)
	}

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
	if err != nil {
		t.Error(err)
	}

	bal, err := GetBalance("bruh", "1337", happyCredentials, asset.Spot, currency.BTC)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	bal.m.Lock()
	if bal.total != 2 {
		t.Fatal("unexpected value")
	}
	if bal.hold != 1 {
		t.Fatal("unexpected value")
	}
}

func TestBalanceInternalWait(t *testing.T) {
	t.Parallel()
	var bi *ProtectedBalance
	_, _, err := bi.Wait(0)
	if !errors.Is(err, errBalanceIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errBalanceIsNil)
	}

	bi = &ProtectedBalance{}
	waiter, _, err := bi.Wait(time.Nanosecond)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	if !<-waiter {
		t.Fatal("should been alerted by timeout")
	}

	waiter, _, err = bi.Wait(0)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	go bi.notice.Alert()
	if <-waiter {
		t.Fatal("should have been alerted by change notice")
	}
}

func TestBalanceInternalLoad(t *testing.T) {
	t.Parallel()
	bi := &ProtectedBalance{}
	bi.load(Balance{Total: 1, Hold: 2, Free: 3, AvailableWithoutBorrow: 4, Borrowed: 5})
	bi.m.Lock()
	if bi.total != 1 {
		t.Fatal("unexpected value")
	}
	if bi.hold != 2 {
		t.Fatal("unexpected value")
	}
	if bi.free != 3 {
		t.Fatal("unexpected value")
	}
	if bi.availableWithoutBorrow != 4 {
		t.Fatal("unexpected value")
	}
	if bi.borrowed != 5 {
		t.Fatal("unexpected value")
	}
	bi.m.Unlock()

	if bi.GetFree() != 3 {
		t.Fatal("unexpected value")
	}
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

func TestUpdate(t *testing.T) {
	t.Parallel()
	s := &Service{exchangeAccounts: make(map[string]*Accounts), mux: dispatch.GetNewMux(nil)}
	err := s.Update(nil, nil)
	if !errors.Is(err, errHoldingsIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errHoldingsIsNil)
	}

	err = s.Update(&Holdings{}, nil)
	if !errors.Is(err, errExchangeNameUnset) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeNameUnset)
	}

	err = s.Update(&Holdings{
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
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	err = s.Update(&Holdings{ // No change
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
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	acc, ok := s.exchangeAccounts["test"]
	if !ok {
		t.Fatal("account should be loaded")
	}

	b, ok := acc.SubAccounts[Credentials{Key: "AAAAA"}][key.SubAccountCurrencyAsset{
		SubAccount: "1337",
		Currency:   currency.BTC.Item,
		Asset:      asset.Spot,
	}]
	if !ok {
		t.Fatal("account should be loaded")
	}

	if b.total != 100 {
		t.Errorf("expecting 100 but received %f", b.total)
	}

	if b.hold != 20 {
		t.Errorf("expecting 20 but received %f", b.hold)
	}
}
