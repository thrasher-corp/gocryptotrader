package fixer

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/livetest"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/standin"
)

// Please set API key and apikey subscription level for correct due diligence
// testing - NOTE please be aware tests will diminish your monthly API calls

var f Fixer

var isSetup bool

// missingAccessKeyReply is apilayer's reply to a request without an access key, captured live
const missingAccessKeyReply = `{
  "success": false,
  "error": {
    "code": 101,
    "type": "missing_access_key",
    "info": "You have not supplied an API Access Key. [Required format: access_key=YOUR_ACCESS_KEY]"
  }
}`

// noAccessKeyReply answers requests the way Fixer answers a client without an access key
func noAccessKeyReply(r *http.Request) (standin.Reply, error) {
	if r.URL.Host != "data.fixer.io" {
		return standin.Reply{}, fmt.Errorf("%w for host %q", standin.ErrNoReply, r.URL.Host)
	}
	return standin.Reply{Status: http.StatusUnauthorized, Body: missingAccessKeyReply}, nil
}

func setup(t *testing.T) {
	t.Helper()
	if !isSetup {
		err := f.Setup(base.Settings{})
		if err != nil {
			t.Fatal("Setup error", err)
		}
		if livetest.Mock {
			require.NoError(t, f.Requester.SetHTTPClient(standin.Client(noAccessKeyReply)), "SetHTTPClient must not error")
		}
		isSetup = true
	}
}

func TestGetRates(t *testing.T) {
	setup(t)
	_, err := f.GetRates("EUR", "AUD")
	if err == nil {
		t.Error("fixer GetRates() Expected error")
	}
}

func TestGetLatestRates(t *testing.T) {
	setup(t)
	_, err := f.GetLatestRates("EUR", "AUD")
	if err == nil {
		t.Error("fixer GetLatestRates() Expected error")
	}
}

func TestGetHistoricalRates(t *testing.T) {
	setup(t)
	_, err := f.GetHistoricalRates("2013-12-24", "EUR", []string{"AUD,KRW"})
	if err == nil {
		t.Error("fixer GetHistoricalRates() Expected error")
	}
}

func TestConvertCurrency(t *testing.T) {
	setup(t)
	_, err := f.ConvertCurrency("AUD", "EUR", "", 1337)
	if err == nil {
		t.Error("fixer ConvertCurrency() Expected error")
	}
}

func TestGetTimeSeriesData(t *testing.T) {
	setup(t)
	_, err := f.GetTimeSeriesData("2013-12-24", "2013-12-25", "EUR", []string{"AUD,KRW"})
	if err == nil {
		t.Error("fixer GetTimeSeriesData() Expected error")
	}
}

func TestGetFluctuationData(t *testing.T) {
	setup(t)
	_, err := f.GetFluctuationData("2013-12-24", "2013-12-25", "EUR", []string{"AUD,KRW"})
	if err == nil {
		t.Error("fixer GetFluctuationData() Expected error")
	}
}
