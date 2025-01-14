package versions

import (
	"context"
	"encoding/json"

	"github.com/buger/jsonparser"
)

// Version3 is an ExchangeVersion to add the websocketMetricsLogging field
type Version3 struct {
}

func init() {
	Manager.registerVersion(3, &Version3{})
}

// Exchanges returns all exchanges: "*"
func (v *Version3) Exchanges() []string { return []string{"*"} }

// UpgradeExchange will upgrade the exchange config with the websocketMetricsLogging field
func (v *Version3) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if len(e) == 0 {
		return e, nil
	}
	if _, _, _, err := jsonparser.Get(e, "websocketMetricsLogging"); err == nil {
		return e, nil
	}
	val, err := json.Marshal(false)
	if err != nil {
		return nil, err
	}
	return jsonparser.Set(e, val, "websocketMetricsLogging")
}

// DowngradeExchange will downgrade the exchange config by removing the websocketMetricsLogging field
func (v *Version3) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	return jsonparser.Delete(e, "websocketMetricsLogging"), nil
}
