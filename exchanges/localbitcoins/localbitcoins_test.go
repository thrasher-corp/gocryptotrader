package localbitcoins

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var l LocalBitcoins

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey    = ""
	apiSecret = ""
)

func TestSetDefaults(t *testing.T) {
	l.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	localbitcoinsConfig, err := cfg.GetExchangeConfig("LocalBitcoins")
	if err != nil {
		t.Error("Test Failed - LakeBTC Setup() init error")
	}

	localbitcoinsConfig.AuthenticatedAPISupport = true
	localbitcoinsConfig.APIKey = apiKey
	localbitcoinsConfig.APISecret = apiSecret

	l.Setup(localbitcoinsConfig)
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	if l.GetFee(false) != 0 || l.GetFee(true) != 0 {
		t.Error("Test Failed - GetFee() error")
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := l.GetAccountInfo("", true)
	if err == nil {
		t.Error("Test Failed - GetAccountInfo() error", err)
	}
	_, err = l.GetAccountInfo("bitcoinbaron", false)
	if err != nil {
		t.Error("Test Failed - GetAccountInfo() error", err)
	}
}

func TestGetads(t *testing.T) {
	t.Parallel()
	_, err := l.Getads("")
	if err == nil {
		t.Error("Test Failed - Getads() - Full list, error", err)
	}
	_, err = l.Getads("1337")
	if err == nil {
		t.Error("Test Failed - Getads() error", err)
	}
}

func TestEditAd(t *testing.T) {
	t.Parallel()
	edit := AdEdit{}
	err := l.EditAd(edit, "1337")
	if err == nil {
		t.Error("Test Failed - EditAd() error", err)
	}
}
