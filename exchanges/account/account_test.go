package account

import (
	"errors"
	"sync"
	"testing"
	"time"

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
	if err != nil {
		t.Fatal(err)
	}
	err = Process(nil, nil)
	if !errors.Is(err, errHoldingsIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errHoldingsIsNil)
	}

	err = Process(&Holdings{}, nil)
	if !errors.Is(err, errExchangeNameUnset) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeNameUnset)
	}

	holdings := Holdings{
		Exchange: "Test",
	}

	err = Process(&holdings, nil)
	if !errors.Is(err, errCredentialsAreNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errCredentialsAreNil)
	}

	err = Process(&holdings, happyCredentials)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = Process(&Holdings{
		Exchange: "Test",
		Accounts: []SubAccount{
			{
				ID: "1337",
			}},
	}, happyCredentials)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

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
	if err != nil {
		t.Error(err)
	}

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
	if err != nil {
		t.Error(err)
	}

	_, err = GetHoldings("", nil, asset.Spot)
	if !errors.Is(err, errExchangeNameUnset) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeNameUnset)
	}

	_, err = GetHoldings("bla", nil, asset.Spot)
	if !errors.Is(err, errCredentialsAreNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errCredentialsAreNil)
	}

	_, err = GetHoldings("bla", happyCredentials, asset.Spot)
	if !errors.Is(err, errExchangeHoldingsNotFound) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeHoldingsNotFound)
	}

	_, err = GetHoldings("bla", happyCredentials, asset.Empty)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	_, err = GetHoldings("Test", happyCredentials, asset.UpsideProfitContract)
	if !errors.Is(err, errAssetHoldingsNotFound) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errAssetHoldingsNotFound)
	}

	_, err = GetHoldings("Test", &Credentials{Key: "BBBBB"}, asset.Spot)
	if !errors.Is(err, errNoCredentialBalances) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoCredentialBalances)
	}

	u, err := GetHoldings("Test", happyCredentials, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	if u.Accounts[0].ID != "1337" {
		t.Errorf("expecting 1337 but received %s", u.Accounts[0].ID)
	}

	if !u.Accounts[0].Currencies[0].Currency.Equal(currency.BTC) {
		t.Errorf("expecting BTC but received %s",
			u.Accounts[0].Currencies[0].Currency)
	}

	if u.Accounts[0].Currencies[0].Total != 100 {
		t.Errorf("expecting 100 but received %f",
			u.Accounts[0].Currencies[0].Total)
	}

	if u.Accounts[0].Currencies[0].Hold != 20 {
		t.Errorf("expecting 20 but received %f",
			u.Accounts[0].Currencies[0].Hold)
	}

	_, err = SubscribeToExchangeAccount("nonsense")
	if !errors.Is(err, errExchangeAccountsNotFound) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeAccountsNotFound)
	}

	p, err := SubscribeToExchangeAccount("Test")
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func(p dispatch.Pipe, wg *sync.WaitGroup) {
		for i := 0; i < 2; i++ {
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
	if err != nil {
		t.Error(err)
	}

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
	if !errors.Is(err, errExchangeHoldingsNotFound) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeHoldingsNotFound)
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
	if !errors.Is(err, errAssetHoldingsNotFound) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errAssetHoldingsNotFound)
	}

	_, err = GetBalance("bruh", "1337", happyCredentials, asset.Spot, currency.BTC)
	if !errors.Is(err, errNoBalanceFound) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoBalanceFound)
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

	b, ok := acc.SubAccounts[Credentials{Key: "AAAAA"}]["1337"][asset.Spot][currency.BTC.Item]
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
