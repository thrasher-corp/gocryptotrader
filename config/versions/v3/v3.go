package v3

import (
	"context"
	"time"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// Version is an ExchangeVersion to remove the publishPeriod from the exchange's orderbook config
type Version struct{}

// Exchanges returns all exchanges: "*"
func (*Version) Exchanges() []string { return []string{"*"} }

// UpgradeExchange will remove the publishPeriod from the exchange's orderbook config
func (*Version) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	e = jsonparser.Delete(e, "orderbook", "publishPeriod")
	return e, nil
}

const defaultOrderbookPublishPeriod = time.Second * 10

// DowngradeExchange will downgrade the exchange's config by setting the default orderbook publish period
func (*Version) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if _, _, _, err := jsonparser.Get(e, "orderbook"); err != nil {
		return e, nil //nolint:nilerr // No error, just return the original config
	}
	out, err := json.Marshal(defaultOrderbookPublishPeriod)
	if err != nil {
		return e, err
	}
	return jsonparser.Set(e, out, "orderbook", "publishPeriod")
}
