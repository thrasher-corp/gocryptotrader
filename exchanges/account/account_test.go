package account

import (
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestHoldings(t *testing.T) {
	err := dispatch.Start(dispatch.DefaultMaxWorkers, dispatch.DefaultJobsLimit)
	if err != nil {
		t.Fatal(err)
	}
	err = Process(nil)
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = Process(&Holdings{})
	if err == nil {
		t.Error("error cannot be nil")
	}

	holdings := Holdings{
		Exchange: "Test",
	}

	err = Process(&holdings)
	if err != nil {
		t.Error(err)
	}

	err = Process(&Holdings{
		Exchange: "Test",
		Accounts: []SubAccount{{
			AssetType: asset.Spot,
			ID:        "1337",
			Currencies: []Balance{
				{
					CurrencyName: currency.BTC,
					TotalValue:   100,
					Hold:         20,
				},
			},
		}},
	})
	if err != nil {
		t.Error(err)
	}

	_, err = GetHoldings("", asset.Spot)
	if err == nil {
		t.Error("error cannot be nil")
	}

	_, err = GetHoldings("bla", asset.Spot)
	if err == nil {
		t.Error("error cannot be nil")
	}

	_, err = GetHoldings("bla", asset.Item("hi"))
	if err == nil {
		t.Error("error cannot be nil since an invalid asset type is provided")
	}

	u, err := GetHoldings("Test", asset.Spot)
	if err != nil {
		t.Error(err)
	}

	if u.Accounts[0].ID != "1337" {
		t.Errorf("expecting 1337 but received %s", u.Accounts[0].ID)
	}

	if u.Accounts[0].Currencies[0].CurrencyName != currency.BTC {
		t.Errorf("expecting BTC but received %s",
			u.Accounts[0].Currencies[0].CurrencyName)
	}

	if u.Accounts[0].Currencies[0].TotalValue != 100 {
		t.Errorf("expecting 100 but received %f",
			u.Accounts[0].Currencies[0].TotalValue)
	}

	if u.Accounts[0].Currencies[0].Hold != 20 {
		t.Errorf("expecting 20 but received %f",
			u.Accounts[0].Currencies[0].Hold)
	}

	_, err = SubscribeToExchangeAccount("nonsense")
	if err == nil {
		t.Fatal("error cannot be nil")
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
			case <-p.C:
			case <-c.C:
			}
		}

		wg.Done()
	}(p, &wg)

	err = Process(&Holdings{
		Exchange: "Test",
		Accounts: []SubAccount{{
			ID: "1337",
			Currencies: []Balance{
				{
					CurrencyName: currency.BTC,
					TotalValue:   100000,
					Hold:         20,
				},
			},
		}},
	})
	if err != nil {
		t.Error(err)
	}

	wg.Wait()
}
