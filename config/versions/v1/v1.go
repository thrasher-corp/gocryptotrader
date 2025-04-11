package v1

import (
	"context"

	"github.com/buger/jsonparser"
	v0 "github.com/thrasher-corp/gocryptotrader/config/versions/v0"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// Version is an ExchangeVersion to upgrade currency pair format for exchanges
type Version struct{}

// Exchanges returns all exchanges: "*"
func (*Version) Exchanges() []string { return []string{"*"} }

// UpgradeExchange will upgrade currency pair format
func (*Version) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if _, d, _, err := jsonparser.Get(e, "currencyPairs"); err == nil && d == jsonparser.Object {
		return e, nil
	}

	d := &v0.Exchange{}
	if err := json.Unmarshal(e, d); err != nil {
		return e, err
	}

	p := &PairsManager{
		UseGlobalFormat: true,
		LastUpdated:     d.PairsLastUpdated,
		ConfigFormat:    d.ConfigCurrencyPairFormat,
		RequestFormat:   d.RequestCurrencyPairFormat,
		Pairs: FullStore{
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
func (*Version) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	return e, nil
}
