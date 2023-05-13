package coinmarketcap

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/log"
)

var c Coinmarketcap

// Please set API keys to test endpoint
const (
	apikey              = ""
	apiAccountPlanLevel = ""
)

// Checks credentials but also checks to see if the function can take the
// required account plan level
func areAPICredtionalsSet(minAllowable uint8) bool {
	if apiAccountPlanLevel != "" && apikey != "" {
		if err := c.CheckAccountPlan(minAllowable); err != nil {
			log.Warnln(log.Global, "coinmarketpcap test suite - account plan not allowed for function, please review or upgrade plan to test")
			return false
		}
		return true
	}
	return false
}

func TestSetDefaults(_ *testing.T) {
	c.SetDefaults()
}

func TestSetup(t *testing.T) {
	c.SetDefaults()

	cfg := Settings{}
	cfg.APIKey = apikey
	cfg.AccountPlan = apiAccountPlanLevel
	cfg.Enabled = true
	cfg.AccountPlan = "basic"

	if err := c.Setup(cfg); err != nil {
		t.Error(err)
	}
}

func TestCheckAccountPlan(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)

	if areAPICredtionalsSet(Basic) {
		err := c.CheckAccountPlan(Enterprise)
		if err == nil {
			t.Error("CheckAccountPlan() error cannot be nil")
		}

		err = c.CheckAccountPlan(Professional)
		if err == nil {
			t.Error("CheckAccountPlan() error cannot be nil")
		}

		err = c.CheckAccountPlan(Standard)
		if err == nil {
			t.Error("CheckAccountPlan() error cannot be nil")
		}

		err = c.CheckAccountPlan(Hobbyist)
		if err == nil {
			t.Error("CheckAccountPlan() error cannot be nil")
		}

		err = c.CheckAccountPlan(Startup)
		if err == nil {
			t.Error("CheckAccountPlan() error cannot be nil")
		}

		err = c.CheckAccountPlan(Basic)
		if err != nil {
			t.Error("CheckAccountPlan() error", err)
		}
	}
}

func TestGetCryptocurrencyInfo(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyInfo(1)
	if areAPICredtionalsSet(Basic) {
		if err != nil {
			t.Error("GetCryptocurrencyInfo() error", err)
		}
	} else {
		if err == nil {
			t.Error("GetCryptocurrencyInfo() error cannot be nil")
		}
	}
}

func TestGetCryptocurrencyIDMap(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyIDMap()
	if areAPICredtionalsSet(Basic) {
		if err != nil {
			t.Error("GetCryptocurrencyIDMap() error", err)
		}
	} else {
		if err == nil {
			t.Error("GetCryptocurrencyIDMap() error cannot be nil")
		}
	}
}

func TestGetCryptocurrencyHistoricalListings(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyHistoricalListings()
	if err == nil {
		t.Error("GetCryptocurrencyHistoricalListings() error cannot be nil")
	}
}

func TestGetCryptocurrencyLatestListing(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyLatestListing(0, 0)
	if areAPICredtionalsSet(Basic) {
		if err != nil {
			t.Error("GetCryptocurrencyLatestListing() error", err)
		}
	} else {
		if err == nil {
			t.Error("GetCryptocurrencyLatestListing() error cannot be nil")
		}
	}
}

func TestGetCryptocurrencyLatestMarketPairs(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyLatestMarketPairs(1, 0, 0)
	if areAPICredtionalsSet(Standard) {
		if err != nil {
			t.Error("GetCryptocurrencyLatestMarketPairs() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("GetCryptocurrencyLatestMarketPairs() error cannot be nil")
		}
	}
}

func TestGetCryptocurrencyOHLCHistorical(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyOHLCHistorical(1, time.Now(), time.Now())
	if areAPICredtionalsSet(Standard) {
		if err != nil {
			t.Error("GetCryptocurrencyOHLCHistorical() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("GetCryptocurrencyOHLCHistorical() error cannot be nil")
		}
	}
}

func TestGetCryptocurrencyOHLCLatest(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyOHLCLatest(1)
	if areAPICredtionalsSet(Startup) {
		if err != nil {
			t.Error("GetCryptocurrencyOHLCLatest() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("GetCryptocurrencyOHLCLatest() error cannot be nil")
		}
	}
}

func TestGetCryptocurrencyLatestQuotes(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyLatestQuotes(1)
	if areAPICredtionalsSet(Basic) {
		if err != nil {
			t.Error("GetCryptocurrencyLatestQuotes() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("GetCryptocurrencyLatestQuotes() error cannot be nil")
		}
	}
}

func TestGetCryptocurrencyHistoricalQuotes(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyHistoricalQuotes(1, time.Now(), time.Now())
	if areAPICredtionalsSet(Standard) {
		if err != nil {
			t.Error("GetCryptocurrencyHistoricalQuotes() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("GetCryptocurrencyHistoricalQuotes() error cannot be nil")
		}
	}
}

func TestGetExchangeInfo(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetExchangeInfo(1)
	if areAPICredtionalsSet(Startup) {
		if err != nil {
			t.Error("GetExchangeInfo() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("GetExchangeInfo() error cannot be nil")
		}
	}
}

func TestGetExchangeMap(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetExchangeMap(0, 0)
	if areAPICredtionalsSet(Startup) {
		if err != nil {
			t.Error("GetExchangeMap() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("GetExchangeMap() error cannot be nil")
		}
	}
}

func TestGetExchangeHistoricalListings(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetExchangeHistoricalListings()
	if err == nil {
		// TODO: update this once the feature above is implemented
		t.Error("GetExchangeHistoricalListings() error cannot be nil")
	}
}

func TestGetExchangeLatestListings(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetExchangeLatestListings()
	if err == nil {
		// TODO: update this once the feature above is implemented
		t.Error("GetExchangeHistoricalListings() error cannot be nil")
	}
}

func TestGetExchangeLatestMarketPairs(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetExchangeLatestMarketPairs(1, 0, 0)
	if areAPICredtionalsSet(Standard) {
		if err != nil {
			t.Error("GetExchangeLatestMarketPairs() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("GetExchangeLatestMarketPairs() error cannot be nil")
		}
	}
}

func TestGetExchangeLatestQuotes(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetExchangeLatestQuotes(1)
	if areAPICredtionalsSet(Standard) {
		if err != nil {
			t.Error("GetExchangeLatestQuotes() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("GetExchangeLatestQuotes() error cannot be nil")
		}
	}
}

func TestGetExchangeHistoricalQuotes(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetExchangeHistoricalQuotes(1, time.Now(), time.Now())
	if areAPICredtionalsSet(Standard) {
		if err != nil {
			t.Error("GetExchangeHistoricalQuotes() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("GetExchangeHistoricalQuotes() error cannot be nil")
		}
	}
}

func TestGetGlobalMeticLatestQuotes(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetGlobalMeticLatestQuotes()
	if areAPICredtionalsSet(Basic) {
		if err != nil {
			t.Error("GetGlobalMeticLatestQuotes() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("GetGlobalMeticLatestQuotes() error cannot be nil")
		}
	}
}

func TestGetGlobalMeticHistoricalQuotes(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetGlobalMeticHistoricalQuotes(time.Now(), time.Now())
	if areAPICredtionalsSet(Standard) {
		if err != nil {
			t.Error("GetGlobalMeticHistoricalQuotes() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("GetGlobalMeticHistoricalQuotes() error cannot be nil")
		}
	}
}

func TestGetPriceConversion(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetPriceConversion(0, 1, time.Now())
	if areAPICredtionalsSet(Hobbyist) {
		if err != nil {
			t.Error("GetPriceConversion() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("GetPriceConversion() error cannot be nil")
		}
	}
}

func TestSetAccountPlan(t *testing.T) {
	accPlans := []string{"basic", "startup", "hobbyist", "standard", "professional", "enterprise"}
	for _, plan := range accPlans {
		err := c.SetAccountPlan(plan)
		if err != nil {
			t.Error("SetAccountPlan() error", err)
		}

		switch plan {
		case "basic":
			if c.Plan != Basic {
				t.Error("SetAccountPlan() error basic plan not set correctly")
			}
		case "startup":
			if c.Plan != Startup {
				t.Error("SetAccountPlan() error startup plan not set correctly")
			}
		case "hobbyist":
			if c.Plan != Hobbyist {
				t.Error("SetAccountPlan() error hobbyist plan not set correctly")
			}
		case "standard":
			if c.Plan != Standard {
				t.Error("SetAccountPlan() error standard plan not set correctly")
			}
		case "professional":
			if c.Plan != Professional {
				t.Error("SetAccountPlan() error professional plan not set correctly")
			}
		case "enterprise":
			if c.Plan != Enterprise {
				t.Error("SetAccountPlan() error enterprise plan not set correctly")
			}
		}
	}
}
