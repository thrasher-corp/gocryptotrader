package main

import (
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
		DisableExchangeAutoPairUpdates: true,
	}

	engine.Bot.ExchangeManager = engine.SetupExchangeManager()

	log.Printf("Loading exchanges..")
	var wg sync.WaitGroup
	for x := range exchange.Exchanges {
		err = engine.Bot.LoadExchange(exchange.Exchanges[x], &wg)
		if err != nil {
			log.Printf("Failed to load exchange %s. Err: %s",
				exchange.Exchanges[x],
				err)
			continue
		}
	}
	wg.Wait()
	log.Println("Done.")

	log.Printf("Testing exchange wrappers..")
	results := make(map[string][]string)
	wg = sync.WaitGroup{}
	exchanges := engine.Bot.GetExchanges()
	for x := range exchanges {
		exch := exchanges[x]
		wg.Add(1)
		go func(e exchange.IBotExchange) {
			results[e.GetName()], err = testWrappers(e)
			if err != nil {
				fmt.Printf("failed to test wrappers for %s %s", e.GetName(), err)
			}
			wg.Done()
		}(exch)
	}
	wg.Wait()
	log.Println("Done.")

	var dummyInterface exchange.IBotExchange
	totalWrappers := reflect.TypeOf(&dummyInterface).Elem().NumMethod()

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

// errValue extract reflection type
var errValue = reflect.TypeOf(errors.New(""))

// testWrappers searches the functions returns for common.ErrNotYetImplemented
// for checking if the wrapper function has been implemented yet. This uses
// reflection so it can dynamically scale to GCT's exchange interface, IBotExchange.
func testWrappers(e exchange.IBotExchange) ([]string, error) {
	iExchange := reflect.TypeOf(&e).Elem()
	actualExchange := reflect.ValueOf(e)

	var funcs []string
	for x := 0; x < iExchange.NumMethod(); x++ {
		name := iExchange.Method(x).Name
		method := actualExchange.MethodByName(name)
		inputs := make([]reflect.Value, method.Type().NumIn())
		for y := 0; y < method.Type().NumIn(); y++ {
			input := method.Type().In(y)
			// Initial stage of fuzzing wrapper functions in core GCT
			// functionality, for now, zero value all inputs.
			inputs[y] = reflect.Zero(input)
		}

		outputs := method.Call(inputs)
		for y := range outputs {
			incoming := outputs[y].Interface()
			if reflect.TypeOf(incoming) == errValue {
				err, ok := incoming.(error)
				if !ok {
					return nil, fmt.Errorf("%s type assertion failure for %v", name, incoming)
				}
				if errors.Is(err, common.ErrNotYetImplemented) {
					funcs = append(funcs, name)
				}
				break // found error; there should not be another error in this slice.
			}
		}
	}
	return funcs, nil
}
