package backtest

import (
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestNewFromConfig(t *testing.T) {
	_, err := NewFromConfig(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}

	cfg := &config.Config{}
	_, err = NewFromConfig(cfg)
	if err == nil {
		t.Error("expected error for nil config")
	}
	if err != nil && err.Error() != "expected at least one currency in the config" {
		t.Error(err)
	}

	cfg.CurrencySettings = []config.CurrencySettings{
		{
			ExchangeName: "test",
			Asset:        "test",
			Base:         "test",
			Quote:        "test",
		},
	}
	_, err = NewFromConfig(cfg)
	if err != nil && err.Error() != "exchange not found" {
		t.Error(err)
	}

	cfg.CurrencySettings[0].ExchangeName = "binance"
	_, err = NewFromConfig(cfg)
	if err != nil && !strings.Contains(err.Error(), "cannot create new asset") {
		t.Error(err)
	}

	cfg.CurrencySettings[0].Asset = asset.Spot.String()
	cfg.CurrencySettings[0].Base = "BTC"
	cfg.CurrencySettings[0].Quote = "USDT"
	_, err = NewFromConfig(cfg)
	if err != nil && !strings.Contains(err.Error(), "initial funds unset") {
		t.Error(err)
	}

	cfg.CurrencySettings[0].InitialFunds = 1337

	_, err = NewFromConfig(cfg)
	if err != nil && err.Error() != "no data settings set in config" {
		t.Error(err)
	}

	cfg.APIData = &config.APIData{
		DataType:  "",
		Interval:  0,
		StartDate: time.Time{},
		EndDate:   time.Time{},
	}

	_, err = NewFromConfig(cfg)
	if err != nil && err.Error() != "api data start and end dates must be set" {
		t.Error(err)
	}

	cfg.APIData.StartDate = time.Now().Add(-time.Hour)
	cfg.APIData.EndDate = time.Now()
	_, err = NewFromConfig(cfg)
	if err != nil && err.Error() != "api data interval unset" {
		t.Error(err)
	}

	cfg.APIData.Interval = gctkline.FifteenMin.Duration()
	_, err = NewFromConfig(cfg)
	if err != nil && err.Error() != "unrecognised api datatype received: ''" {
		t.Error(err)
	}

	cfg.APIData.DataType = common.CandleStr
	_, err = NewFromConfig(cfg)
	if err != nil && err.Error() != "strategy '' not found" {
		t.Error(err)
	}

	cfg.StrategySettings = config.StrategySettings{
		Name: dollarcostaverage.Name,
		CustomSettings: map[string]interface{}{
			"hello": "moto",
		},
	}
	cfg.CurrencySettings[0].MakerFee = 1337
	cfg.CurrencySettings[0].TakerFee = 1337
	_, err = NewFromConfig(cfg)
	if err != nil {
		t.Error(err)
	}
}

func TestLoadDatabaseData(t *testing.T) {
	cp := currency.NewPair(currency.BTC, currency.USDT)
	_, err := loadDatabaseData(nil, "", cp, "")
	if err != nil && !strings.Contains(err.Error(), "nil config data received") {
		t.Error(err)
	}
	cfg := &config.Config{DatabaseData: &config.DatabaseData{
		DataType:       "",
		Interval:       0,
		StartDate:      time.Time{},
		EndDate:        time.Time{},
		ConfigOverride: nil,
	}}
	_, err = loadDatabaseData(cfg, "", cp, "")
	if err != nil && !strings.Contains(err.Error(), "database data start and end dates must be set") {
		t.Error(err)
	}
	cfg.DatabaseData.StartDate = time.Now().Add(-time.Hour)
	cfg.DatabaseData.EndDate = time.Now()
	_, err = loadDatabaseData(cfg, "", cp, "")
	if err != nil && !strings.Contains(err.Error(), "unexpected database datatype: ''") {
		t.Error(err)
	}

	cfg.DatabaseData.DataType = common.CandleStr
	_, err = loadDatabaseData(cfg, "", cp, "")
	if err != nil && !strings.Contains(err.Error(), "exchange, base, quote, asset, interval, start & end cannot be empty") {
		t.Error(err)
	}
	cfg.DatabaseData.Interval = gctkline.OneDay.Duration()
	_, err = loadDatabaseData(cfg, "binance", cp, asset.Spot)
	if err != nil && !strings.Contains(err.Error(), "database support is disabled") {
		t.Error(err)
	}
}

func TestLoadLiveData(t *testing.T) {
	err := loadLiveData(nil, nil)
	if err != nil && err.Error() != "received nil argument(s)" {
		t.Error(err)
	}
	cfg := &config.Config{}
	err = loadLiveData(cfg, nil)
	if err != nil && err.Error() != "received nil argument(s)" {
		t.Error(err)
	}
	b := &gctexchange.Base{
		Name: "binance",
		API: gctexchange.API{
			AuthenticatedSupport:          false,
			AuthenticatedWebsocketSupport: false,
			PEMKeySupport:                 false,
			Endpoints: struct {
				URL                 string
				URLDefault          string
				URLSecondary        string
				URLSecondaryDefault string
				WebsocketURL        string
			}{},
			Credentials: struct {
				Key      string
				Secret   string
				ClientID string
				PEMKey   string
			}{},
			CredentialsValidator: struct {
				RequiresPEM                bool
				RequiresKey                bool
				RequiresSecret             bool
				RequiresClientID           bool
				RequiresBase64DecodeSecret bool
			}{
				RequiresPEM:                true,
				RequiresKey:                true,
				RequiresSecret:             true,
				RequiresClientID:           true,
				RequiresBase64DecodeSecret: true,
			},
		},
	}
	err = loadLiveData(cfg, b)
	if err != nil && err.Error() != "received nil argument(s)" {
		t.Error(err)
	}
	cfg.LiveData = &config.LiveData{
		Interval:   gctkline.OneDay.Duration(),
		DataType:   common.CandleStr,
		RealOrders: true,
	}
	err = loadLiveData(cfg, b)
	if err != nil {
		t.Error(err)
	}

	cfg.LiveData.APIKeyOverride = "1234"
	cfg.LiveData.APISecretOverride = "1234"
	cfg.LiveData.APIClientIDOverride = "1234"
	cfg.LiveData.API2FAOverride = "1234"
	err = loadLiveData(cfg, b)
	if err != nil {
		t.Error(err)
	}
}
