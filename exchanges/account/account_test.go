package account

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestHoldings(t *testing.T) {
	err := Process(nil)
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

	// process update
	err = Process(&Holdings{
		Exchange: "Test",
		Accounts: []SubAccount{{
			ID: "1337",
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

	_, err = GetHoldings("")
	if err == nil {
		t.Error("error cannot be nil")
	}

	_, err = GetHoldings("bla")
	if err == nil {
		t.Error("error cannot be nil")
	}

	u, err := GetHoldings("Test")
	if err != nil {
		t.Error(err)
	}

	if u.Accounts[0].ID != "1337" {
		t.Errorf("expecting 1337 but receieved %s", u.Accounts[0].ID)
	}

	if u.Accounts[0].Currencies[0].CurrencyName != currency.BTC {
		t.Errorf("expecting BTC but receieved %s",
			u.Accounts[0].Currencies[0].CurrencyName)
	}

	if u.Accounts[0].Currencies[0].TotalValue != 100 {
		t.Errorf("expecting 100 but receieved %f",
			u.Accounts[0].Currencies[0].TotalValue)
	}

	if u.Accounts[0].Currencies[0].Hold != 20 {
		t.Errorf("expecting 20 but receieved %f",
			u.Accounts[0].Currencies[0].Hold)
	}
}
