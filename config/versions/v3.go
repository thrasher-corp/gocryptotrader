package versions

import (
	"context"
	"encoding/json"
	"time"

	"github.com/buger/jsonparser"
)

// Version3 is an ExchangeVersion to remove the publishPeriod from the exchange's orderbook config
type Version3 struct{}

func init() {
	Manager.registerVersion(3, &Version3{})
}

// Exchanges returns all exchanges: "*"
func (v *Version3) Exchanges() []string { return []string{"*"} }

// UpgradeExchange will remove the publishPeriod from the exchange's orderbook config
func (v *Version3) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	e = jsonparser.Delete(e, "orderbook", "publishPeriod")
	return e, nil
}

const defaultOrderbookPublishPeriod = time.Second * 10

// DowngradeExchange will downgrade the exchange's config by setting the default orderbook publish period
func (v *Version3) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if _, _, _, err := jsonparser.Get(e, "orderbook"); err != nil {
		return e, nil //nolint:nilerr // No error, just return the original config
	}
	out, err := json.Marshal(defaultOrderbookPublishPeriod)
	if err != nil {
		return e, err
	}
	return jsonparser.Set(e, out, "orderbook", "publishPeriod")
}
