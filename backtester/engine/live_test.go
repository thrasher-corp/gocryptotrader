package engine

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestLoadLiveData(t *testing.T) {
	t.Parallel()
	err := loadLiveData(nil, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Error(err)
	}
	cfg := &config.Config{}
	err = loadLiveData(cfg, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Error(err)
	}
	b := &gctexchange.Base{
		Name: testExchange,
		API: gctexchange.API{
			CredentialsValidator: gctexchange.CredentialsValidator{
				RequiresPEM:                true,
				RequiresKey:                true,
				RequiresSecret:             true,
				RequiresClientID:           true,
				RequiresBase64DecodeSecret: true,
			},
		},
	}

	err = loadLiveData(cfg, b)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Error(err)
	}
	cfg.DataSettings.LiveData = &config.LiveData{
		RealOrders: true,
	}
	cfg.DataSettings.Interval = gctkline.OneDay
	cfg.DataSettings.DataType = common.CandleStr
	err = loadLiveData(cfg, b)
	if err != nil {
		t.Error(err)
	}

	cfg.DataSettings.LiveData.APIKeyOverride = "1234"
	cfg.DataSettings.LiveData.APISecretOverride = "1234"
	cfg.DataSettings.LiveData.APIClientIDOverride = "1234"
	cfg.DataSettings.LiveData.API2FAOverride = "1234"
	cfg.DataSettings.LiveData.APISubAccountOverride = "1234"
	err = loadLiveData(cfg, b)
	if err != nil {
		t.Error(err)
	}
}
