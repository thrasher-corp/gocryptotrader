package versions

import (
	"context"

	"github.com/buger/jsonparser"
	v0 "github.com/thrasher-corp/gocryptotrader/config/versions/v0"
	v1 "github.com/thrasher-corp/gocryptotrader/config/versions/v1"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// Version1 is an ExchangeVersion to upgrade currency pair format for exchanges
type Version1 struct{}

func init() {
	Manager.registerVersion(1, &Version1{})
}

// Exchanges returns all exchanges: "*"
func (v *Version1) Exchanges() []string { return []string{"*"} }

// UpgradeExchange will upgrade currency pair format
func (v *Version1) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if _, d, _, err := jsonparser.Get(e, "currencyPairs"); err == nil && d == jsonparser.Object {
		return e, nil
	}

	d := &v0.Exchange{}
	if err := json.Unmarshal(e, d); err != nil {
		return e, err
	}

	p := &v1.PairsManager{
		UseGlobalFormat: true,
		LastUpdated:     d.PairsLastUpdated,
		ConfigFormat:    d.ConfigCurrencyPairFormat,
		RequestFormat:   d.RequestCurrencyPairFormat,
		Pairs: v1.FullStore{
			"spot": {
				Available: d.AvailablePairs,
				Enabled:   d.EnabledPairs,
			},
		},
	}
	j, err := json.Marshal(p)
	if err != nil {
		return e, err
	}
	for _, f := range []string{"pairsLastUpdated", "configCurrencyPairFormat", "requestCurrencyPairFormat", "assetTypes", "availablePairs", "enabledPairs"} {
		e = jsonparser.Delete(e, f)
	}
	return jsonparser.Set(e, j, "currencyPairs")
}

// DowngradeExchange doesn't do anything for v1; There's no downgrade path since the original state is lossy and v1 was before versioning
func (v *Version1) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	return e, nil
}
