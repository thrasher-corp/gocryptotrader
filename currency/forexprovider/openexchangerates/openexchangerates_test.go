package openexchangerates

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/livetest"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/standin"
)

// please set apikey and build with the mock_test_off tag for due diligence testing NOTE testing uses your allocated
// API request quota
const (
	apikey = ""
	apilvl = 2
)

var o OXR

func TestMain(m *testing.M) {
	err := o.Setup(base.Settings{
		Name:      "OpenExchangeRates",
		Enabled:   true,
		APIKey:    apikey,
		APIKeyLvl: apilvl,
	})
	if err != nil {
		log.Fatal(err)
	}
	if livetest.Mock {
		currencies, err := os.ReadFile("testdata/currencies.json")
		if err != nil {
			log.Fatal(err)
		}
		if err := o.Requester.SetHTTPClient(standin.Client(noAppIDReply(currencies))); err != nil {
			log.Fatal(err)
		}
	}
	os.Exit(m.Run())
}

// missingAppIDReply is Open Exchange Rates' reply to a request without an app ID, captured live
const missingAppIDReply = `{"error":true,"status":403,"message":"missing_app_id","description":"No App ID provided. Please sign up at https://openexchangerates.org/signup, or contact support@openexchangerates.org."}`

// noAppIDReply answers requests the way Open Exchange Rates answers a client without an app ID, which can still
// list currencies
func noAppIDReply(currencies []byte) standin.Transport {
	return func(r *http.Request) (standin.Reply, error) {
		if r.URL.Host != "openexchangerates.org" {
			return standin.Reply{}, fmt.Errorf("%w for host %q", standin.ErrNoReply, r.URL.Host)
		}
		if strings.HasSuffix(r.URL.Path, "/"+APIEndpointCurrencies) {
			return standin.Reply{Status: http.StatusOK, Body: string(currencies)}, nil
		}
		return standin.Reply{Status: http.StatusForbidden, Body: missingAppIDReply}, nil
	}
}

func TestGetRates(t *testing.T) {
	t.Parallel()
	_, err := o.GetRates("USD", "AUD")
	if err == nil {
		t.Error("GetRates() Expected error")
	}
}

func TestGetLatest(t *testing.T) {
	t.Parallel()
	_, err := o.GetLatest("USD", "AUD", false, false)
	if err == nil {
		t.Error("GetLatest() Expected error")
	}
}

func TestGetHistoricalRates(t *testing.T) {
	t.Parallel()
	_, err := o.GetHistoricalRates("2017-12-01", "USD", []string{"CNH", "AUD", "ANG"}, false, false)
	if err == nil {
		t.Error("GetRates() Expected error")
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := o.GetCurrencies(true, true, true)
	if err != nil {
		t.Error("GetCurrencies() error", err)
	}
}

func TestGetTimeSeries(t *testing.T) {
	t.Parallel()
	_, err := o.GetTimeSeries("USD", "2017-12-01", "2017-12-02", []string{"CNH", "AUD", "ANG"}, false, false)
	if err == nil {
		t.Error("GetTimeSeries() Expected error")
	}
}

func TestConvertCurrency(t *testing.T) {
	t.Parallel()
	_, err := o.ConvertCurrency(1337, "USD", "AUD")
	if err == nil {
		t.Error("ConvertCurrency() Expected error")
	}
}

func TestGetOHLC(t *testing.T) {
	t.Parallel()
	_, err := o.GetOHLC("2017-07-17T08:30:00Z", "1m", "USD", []string{"AUD"}, false)
	if err == nil {
		t.Error("GetOHLC() Expected error")
	}
}

func TestGetUsageStats(t *testing.T) {
	t.Parallel()
	if _, err := o.GetUsageStats(false); err == nil {
		t.Error("GetUsageStats() Expected error")
	}
}

func TestUsageUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `
{
  "status": 200,
  "data": {
    "app_id": "abc123",
    "status": "active",
    "plan": {
      "name": "Enterprise",
      "quota": "100,000 requests/month",
      "update_frequency": "30m",
      "features": {
        "base": true,
        "symbols": true,
        "experimental": true,
        "time-series": true,
        "convert": false
      }
    },
    "usage": {
      "requests": 54524,
      "requests_quota": 100000,
      "requests_remaining": 45476,
      "days_elapsed": 16,
      "days_remaining": 14,
      "daily_average": 3407
    }
  },
  "error": false,
  "message": "",
  "description": ""
}
`

	var x Usage
	require.NoError(t, json.Unmarshal([]byte(inp), &x), "Unmarshal must not error")
	exp := Usage{
		Status: 200,
		Data: UsageData{
			AppID:  "abc123",
			Status: "active",
			Plan: UsagePlan{
				Name:            "Enterprise",
				Quota:           "100,000 requests/month",
				UpdateFrequency: "30m",
				Features: UsagePlanFeature{
					Base:         true,
					Symbols:      true,
					Experimental: true,
					Timeseries:   true,
				},
			},
			Usages: UsageStatistics{
				Requests:          54524,
				RequestQuota:      100000,
				RequestsRemaining: 45476,
				DaysElapsed:       16,
				DaysRemaining:     14,
				DailyAverage:      3407,
			},
		},
	}
	assert.Equal(t, exp, x, "Usage should unmarshal correctly")

	const optional = `
{
  "data": {
    "plan": {
      "features": {
        "convert": true
      }
    }
  },
  "error": true,
  "message": "not_allowed",
  "description": "Access denied"
}
`

	x = Usage{}
	require.NoError(t, json.Unmarshal([]byte(optional), &x), "Unmarshal must decode non-zero optional fields")
	exp = Usage{
		Data: UsageData{
			Plan: UsagePlan{
				Features: UsagePlanFeature{
					Convert: true,
				},
			},
		},
		Error:       true,
		Message:     "not_allowed",
		Description: "Access denied",
	}
	assert.Equal(t, exp, x, "Usage should decode every optional field")
}
