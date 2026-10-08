package engine

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type orderLookupExchange struct {
	exchange.IBotExchange
	response *order.Detail
	err      error
	calls    int
}

func (e *orderLookupExchange) GetOrderInfo(context.Context, string, currency.Pair, asset.Item) (*order.Detail, error) {
	e.calls++
	return e.response, e.err
}
