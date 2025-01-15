package exchange

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/log"
)

type setter struct {
	fn   func(f *protocol.Features) *bool
	args func(asset.Item) []interface{}
}

// AutomaticPreFlightCheck analyzes the exchange's supported asset types
// and protocols, returning a set of dynamically discovered features
// (ProtocolCapabilities) that the exchange supports. This process is
// based on the exchange's specific implementation of wrapper functions.
func AutomaticPreFlightCheck(exch IBotExchange) protocol.FeatureSet {
	if exch == nil {
		return nil
	}

	// TODO: Explicit differentiation between protocol methods on IBOTExchange
	set := protocol.FeatureSet{}

	// Use a context with a deadline to ensure that the pre-flight check does not
	// send any outbound requests.
	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	cancel()

	// TODO: Make this more dynamic and reflect off interface.go interfaces.
	methodToFeature := map[string]setter{
		"SubmitOrder": {fn: func(f *protocol.Features) *bool { return &f.SubmitOrder }, args: func(a asset.Item) []interface{} {
			return []interface{}{ctx, &order.Submit{Exchange: "preflight", AssetType: a, Pair: currency.NewBTCUSD(), Side: order.Buy, Type: order.Market, Price: 1, Amount: 1}}
		}},
		"ModifyOrder": {fn: func(f *protocol.Features) *bool { return &f.ModifyOrder }, args: func(a asset.Item) []interface{} {
			return []interface{}{ctx, &order.Modify{Exchange: "preflight", AssetType: a, Pair: currency.NewBTCUSD(), Side: order.Buy, Type: order.Market, Price: 1, Amount: 1}}
		}},
		"CancelOrder": {fn: func(f *protocol.Features) *bool { return &f.CancelOrder }, args: func(a asset.Item) []interface{} {
			return []interface{}{ctx, &order.Cancel{Exchange: "preflight", AssetType: a, Pair: currency.NewBTCUSD(), OrderID: "bruh"}}
		}},
		"CancelAllOrders": {fn: func(f *protocol.Features) *bool { return &f.CancelOrders }, args: func(a asset.Item) []interface{} {
			return []interface{}{ctx, &order.Cancel{Exchange: "preflight", AssetType: a, Pair: currency.NewBTCUSD()}}
		}},
		"GetOrderInfo": {fn: func(f *protocol.Features) *bool { return &f.GetOrder }, args: func(a asset.Item) []interface{} { return []interface{}{ctx, "someID", currency.NewBTCUSD(), a} }},
		"GetActiveOrders": {fn: func(f *protocol.Features) *bool { return &f.GetOrders }, args: func(a asset.Item) []interface{} {
			return []interface{}{ctx, &order.MultiOrderRequest{AssetType: a, Pairs: currency.Pairs{currency.NewBTCUSD()}, Side: order.Buy, Type: order.Limit}}
		}},
		"GetOrderHistory": {fn: func(f *protocol.Features) *bool { return &f.UserTradeHistory }, args: func(a asset.Item) []interface{} {
			return []interface{}{ctx, &order.MultiOrderRequest{AssetType: a, Pairs: currency.Pairs{currency.NewBTCUSD()}, Side: order.Buy, Type: order.Limit}}
		}},
	}

	for _, a := range exch.GetAssetTypes(false) {
		target := protocol.Target{
			Asset:    a,
			Protocol: protocol.REST,
		}

		features := protocol.Features{}
		for methodName, featureSetter := range methodToFeature {
			method := reflect.ValueOf(exch).MethodByName(methodName)
			if !method.IsValid() {
				log.Errorf(log.ExchangeSys, "Failed pre flight check for %s. Does not have method %s", exch.GetName(), methodName)
				return nil
			}

			args := featureSetter.args(a)
			reflectArgs := make([]reflect.Value, len(args))
			for i, val := range args {
				reflectArgs[i] = reflect.ValueOf(val)
			}

			result := method.Call(reflectArgs)
			err, _ := result[len(result)-1].Interface().(error)
			if err == nil ||
				errors.Is(err, common.ErrFunctionNotSupported) ||
				errors.Is(err, common.ErrNotYetImplemented) ||
				errors.Is(err, asset.ErrNotSupported) {
				continue
			}

			feature := featureSetter.fn(&features)
			*feature = true
		}
		set[target] = features
	}

	return set
}
