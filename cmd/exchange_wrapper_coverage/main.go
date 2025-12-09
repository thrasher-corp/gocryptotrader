package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

func main() {
	var err error
	engine.Bot, err = engine.New()
	if err != nil {
		log.Fatalf("Failed to initialise engine. Err: %s", err)
	}

	engine.Bot.Settings = engine.Settings{
		CoreSettings: engine.CoreSettings{EnableDryRun: true},
		ExchangeTuningSettings: engine.ExchangeTuningSettings{
			DisableExchangeAutoPairUpdates: true,
		},
	}

	engine.Bot.Config.PurgeExchangeAPICredentials()
	engine.Bot.ExchangeManager = engine.NewExchangeManager()

	log.Printf("Loading exchanges..")
	var wg sync.WaitGroup
	for i := range exchange.Exchanges {
		name := exchange.Exchanges[i]
		wg.Go(func() {
			if err := engine.Bot.LoadExchange(name); err != nil {
				log.Printf("Failed to load exchange %s. Err: %s", name, err)
			}
		})
	}
	wg.Wait()
	log.Println("Done.")

	log.Printf("Testing exchange wrappers..")
	results := make(map[string][]string)
	var mtx sync.Mutex

	exchanges := engine.Bot.GetExchanges()
	for x := range exchanges {
		wg.Add(1)
		go func(exch exchange.IBotExchange) {
			strResults, err := testWrappers(exch)
			if err != nil {
				log.Printf("Failed to test wrappers for %s. Err: %s", exch.GetName(), err)
			}
			mtx.Lock()
			results[exch.GetName()] = strResults
			mtx.Unlock()
			wg.Done()
		}(exchanges[x])
	}
	wg.Wait()
	log.Println("Done.")

	totalWrappers := reflect.TypeFor[exchange.IBotExchange]().NumMethod()

	log.Println()
	for name, funcs := range results {
		pct := float64(totalWrappers-len(funcs)) / float64(totalWrappers) * 100
		log.Printf("Exchange %s wrapper coverage [%d/%d - %.2f%%] | Total missing: %d",
			name,
			totalWrappers-len(funcs),
			totalWrappers,
			pct,
			len(funcs))
		log.Printf("\t Wrappers not implemented:")

		for x := range funcs {
			log.Printf("\t - %s", funcs[x])
		}
		log.Println()
	}
}

// testWrappers executes and checks each IBotExchange's function return for the
// error common.ErrNotYetImplemented to verify whether the wrapper function has
// been implemented yet.
func testWrappers(e exchange.IBotExchange) ([]string, error) {
	iExchange := reflect.TypeFor[exchange.IBotExchange]()
	actualExchange := reflect.ValueOf(e)
	errType := reflect.TypeOf(common.ErrNotYetImplemented)

	contextParam := reflect.TypeFor[context.Context]()

	var funcs []string
	for x := range iExchange.NumMethod() {
		name := iExchange.Method(x).Name
		method := actualExchange.MethodByName(name)
		inputs := make([]reflect.Value, method.Type().NumIn())

		for y := range method.Type().NumIn() {
			input := method.Type().In(y)

			if input.Implements(contextParam) {
				// Need to deploy a context.Context value as nil value is not
				// checked throughout codebase. Cancelled to minimise external
				// calls and speed up operation.
				cancelled, cancelfn := context.WithTimeout(context.Background(), 0)
				cancelfn()
				inputs[y] = reflect.ValueOf(cancelled)
				continue
			}
			inputs[y] = reflect.Zero(input)
		}

		outputs := method.Call(inputs)
		if method.Type().NumIn() == 0 {
			// Some empty functions will reset the exchange struct to defaults,
			// so turn off verbosity.
			e.GetBase().Verbose = false
		}

		for y := range outputs {
			incoming := outputs[y].Interface()
			if reflect.TypeOf(incoming) != errType {
				continue
			}
			err, ok := incoming.(error)
			if !ok {
				return nil, fmt.Errorf("%s type assertion failure for %v", name, incoming)
			}
			if errors.Is(err, common.ErrNotYetImplemented) {
				funcs = append(funcs, name)
			}
			// found error; there should not be another error in this slice.
			break
		}
	}
	return funcs, nil
}
