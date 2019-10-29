package currencylayer

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
)

var c CurrencyLayer

// please set your API key here for due diligence testing NOTE be aware you will
// minimize your API calls using this test.
const (
	APIkey   = ""
	Apilevel = 0
)

var isSet bool

func setup() error {
	if !isSet {
		defaultCfg := base.Settings{
			Name:    "CurrencyLayer",
			Enabled: true,
		}

		if APIkey != "" {
			defaultCfg.APIKey = APIkey
		}

		if Apilevel > -2 && Apilevel < 4 {
			defaultCfg.APIKeyLvl = Apilevel
		}

		err := c.Setup(defaultCfg)
		if err != nil {
			return err
		}
		isSet = true
	}
	return nil
}

func areAPIKeysSet() bool {
	return APIkey != "" && Apilevel != -1
}

func TestGetRates(t *testing.T) {
	err := setup()
	if err != nil {
		t.Skip("CurrencyLayer GetRates error", err)
	}
	_, err = c.GetRates("USD", "AUD")
	if areAPIKeysSet() && err != nil {
		t.Error("test error - currencylayer GetRates() error", err)
	} else if !areAPIKeysSet() && err == nil {
		t.Error("test error - currencylayer GetRates() error cannot be nil")
	}
}

func TestGetSupportedCurrencies(t *testing.T) {
	err := setup()
	if err != nil {
		t.Fatal("CurrencyLayer GetSupportedCurrencies error", err)
	}
	_, err = c.GetSupportedCurrencies()
	if areAPIKeysSet() && err != nil {
		t.Error("test error - currencylayer GetSupportedCurrencies() error", err)
	} else if !areAPIKeysSet() && err == nil {
		t.Error("test error - currencylayer GetSupportedCurrencies() error cannot be nil")
	}
}

func TestGetliveData(t *testing.T) {
	err := setup()
	if err != nil {
		t.Fatal("CurrencyLayer GetliveData error", err)
	}
	_, err = c.GetliveData("AUD", "USD")
	if areAPIKeysSet() && err != nil {
		t.Error("test error - currencylayer GetliveData() error", err)
	} else if !areAPIKeysSet() && err == nil {
		t.Error("test error - currencylayer GetliveData() error cannot be nil")
	}
}

func TestGetHistoricalData(t *testing.T) {
	err := setup()
	if err != nil {
		t.Fatal("CurrencyLayer GetHistoricalData error", err)
	}
	_, err = c.GetHistoricalData("2016-12-15", []string{"AUD"}, "USD")
	if areAPIKeysSet() && err != nil {
		t.Error("test error - currencylayer GetHistoricalData() error", err)
	} else if !areAPIKeysSet() && err == nil {
		t.Error("test error - currencylayer GetHistoricalData() error cannot be nil")
	}
}

func TestConvert(t *testing.T) {
	err := setup()
	if err != nil {
		t.Fatal("CurrencyLayer Convert error", err)
	}
	_, err = c.Convert("USD", "AUD", "", 1)
	if areAPIKeysSet() && err != nil && c.APIKeyLvl >= AccountBasic {
		t.Error("test error - currencylayer Convert() error", err)
	} else if !areAPIKeysSet() && err == nil {
		t.Error("test error - currencylayer Convert() error cannot be nil")
	}
}

func TestQueryTimeFrame(t *testing.T) {
	err := setup()
	if err != nil {
		t.Fatal("CurrencyLayer QueryTimeFrame error", err)
	}
	_, err = c.QueryTimeFrame("2010-12-0", "2010-12-5", "USD", []string{"AUD"})
	if areAPIKeysSet() && err != nil && c.APIKeyLvl >= AccountPro {
		t.Error("test error - currencylayer QueryTimeFrame() error", err)
	} else if !areAPIKeysSet() && err == nil {
		t.Error("test error - currencylayer QueryTimeFrame() error cannot be nil")
	}
}

func TestQueryCurrencyChange(t *testing.T) {
	err := setup()
	if err != nil {
		t.Fatal("CurrencyLayer QueryCurrencyChange() error", err)
	}
	_, err = c.QueryCurrencyChange("2010-12-0", "2010-12-5", "USD", []string{"AUD"})
	if areAPIKeysSet() && err != nil && c.APIKeyLvl == AccountEnterprise {
		t.Error("test error - currencylayer QueryCurrencyChange() error", err)
	} else if !areAPIKeysSet() && err == nil {
		t.Error("test error - currencylayer QueryCurrencyChange() error cannot be nil")
	}
}
