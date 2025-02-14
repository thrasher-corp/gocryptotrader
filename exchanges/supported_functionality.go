package exchange

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// GenerateSupportedFunctionality analyzes the exchange's supported asset types and protocols, returning a set of dynamically
// discovered features (ProtocolCapabilities) that the exchange supports. This process is based on the exchange's
// specific implementation of wrapper functions.
func GenerateSupportedFunctionality(exch IBotExchange) protocol.FunctionalitySet {
	if exch == nil {
		return nil
	}

	// TODO: Explicit differentiation between protocol methods on IBOTExchange
	set := protocol.FunctionalitySet{}

	// Use a context with a deadline to ensure that the pre-flight check does not
	// send any outbound requests.
	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	cancel()

	for _, a := range exch.GetAssetTypes(false) {
		var restTrading OrderManagement
		restTradingType := reflect.TypeOf(&restTrading).Elem() // Get the type of the interface

		functionality := make(protocol.Functionality)
		for i := 0; i < restTradingType.NumMethod(); i++ {
			restTradingmethod := restTradingType.Method(i)

			methodName := restTradingmethod.Name
			method, ok := restTradingType.MethodByName(methodName)
			if !ok {
				fmt.Println("Method not found")
				log.Warnf(log.Global, "generate supported functionality: method %s not found for %s", methodName, exch.GetName())
				continue
			}
			methodValue := reflect.ValueOf(exch).MethodByName(methodName)

			args, err := generateArgs(ctx, a, method)
			if err != nil {
				fmt.Println("Args generation failed")
				log.Warnf(log.Global, "generate supported functionality: method %s args generation failed for %s: %v", methodName, exch.GetName(), err)
				continue
			}
			reflectArgs := make([]reflect.Value, len(args))
			for i, val := range args {
				reflectArgs[i] = reflect.ValueOf(val)
			}

			result := methodValue.Call(reflectArgs)
			err, _ = result[len(result)-1].Interface().(error)

			isFunctional := err != nil && !errors.Is(err, common.ErrFunctionNotSupported) && !errors.Is(err, common.ErrNotYetImplemented) && !errors.Is(err, asset.ErrNotSupported)

			functionality[methodName] = isFunctional
		}

		set[protocol.Target{Asset: a, Protocol: protocol.REST}] = functionality
	}

	return set
}

// generateArgs generates the args function based on the method's parameter types
func generateArgs(ctx context.Context, a asset.Item, method reflect.Method) ([]interface{}, error) {
	// Create a slice to hold the arguments
	args := make([]interface{}, 0)

	// Iterate over the method's input parameters
	for j := 0; j < method.Type.NumIn(); j++ {
		paramType := method.Type.In(j)

		// Handle specific parameter types
		switch paramType {
		case reflect.TypeOf((*context.Context)(nil)).Elem():
			args = append(args, ctx)
		case reflect.TypeOf((*order.Submit)(nil)):
			args = append(args, &order.Submit{Exchange: "preflight", AssetType: a, Pair: currency.NewBTCUSD(), Side: order.Buy, Type: order.Market, Price: 1, Amount: 1})
		case reflect.TypeOf((*order.Modify)(nil)):
			args = append(args, &order.Modify{Exchange: "preflight", AssetType: a, Pair: currency.NewBTCUSD(), Side: order.Buy, Type: order.Market, Price: 1, Amount: 1})
		case reflect.TypeOf((*order.Cancel)(nil)):
			args = append(args, &order.Cancel{Exchange: "preflight", AssetType: a, Pair: currency.NewBTCUSD(), OrderID: "bruh"})
		case reflect.TypeOf(([]order.Cancel)(nil)):
			args = append(args, []order.Cancel{{Exchange: "preflight", AssetType: a, Pair: currency.NewBTCUSD(), OrderID: "bruh"}})
		case reflect.TypeOf((*order.MultiOrderRequest)(nil)):
			args = append(args, &order.MultiOrderRequest{AssetType: a, Pairs: currency.Pairs{currency.NewBTCUSD()}, Side: order.Buy, Type: order.Limit})
		case reflect.TypeOf(""):
			args = append(args, "someID")
		case reflect.TypeOf(currency.Pair{}):
			args = append(args, currency.NewBTCUSD())
		case reflect.TypeOf(asset.Item(0)):
			args = append(args, a)
		default:
			return nil, errors.New("unsupported parameter type")
		}
	}
	return args, nil
}
