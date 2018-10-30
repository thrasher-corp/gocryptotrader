package main

import (
	"log"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/engine"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
)

const (
	totalWrappers = 20
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

	log.Printf("Loading exchanges..")
	var wg sync.WaitGroup
	for x := range exchange.Exchanges {
		name := exchange.Exchanges[x]
		err := engine.LoadExchange(name, true, &wg)
		if err != nil {
			log.Printf("Failed to load exchange %s. Err: %s", name, err)
			continue
		}
	}
	wg.Wait()
	log.Println("Done.")

	log.Printf("Testing exchange wrappers..")
	results := make(map[string][]string)
	wg = sync.WaitGroup{}
	for x := range engine.Bot.Exchanges {
		wg.Add(1)
		go func(num int) {
			name := engine.Bot.Exchanges[num].GetName()
			results[name] = testWrappers(engine.Bot.Exchanges[num])
			wg.Done()
		}(x)
	}
	wg.Wait()
	log.Println("Done.")

	log.Println()
	for name, funcs := range results {
		pct := float64(totalWrappers-len(funcs)) / float64(totalWrappers) * 100
		log.Printf("Exchange %s wrapper coverage [%d/%d - %.2f%%] | Total missing: %d", name, totalWrappers-len(funcs), totalWrappers, pct, len(funcs))
		log.Printf("\t Wrappers not implemented:")

		for x := range funcs {
			log.Printf("\t - %s", funcs[x])
		}
		log.Println()
	}
}

func testWrappers(e exchange.IBotExchange) []string {
	p := currency.NewPair(currency.BTC, currency.USD)
	assetType := assets.AssetTypeSpot
	var funcs []string

	_, err := e.FetchTicker(p, assetType)
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "FetchTicker")
	}

	_, err = e.UpdateTicker(p, assetType)
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "UpdateTicker")
	}

	_, err = e.FetchOrderbook(p, assetType)
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "FetchOrderbook")
	}

	_, err = e.UpdateOrderbook(p, assetType)
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "UpdateOrderbook")
	}

	_, err = e.FetchTradablePairs(assets.AssetTypeSpot)
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "FetchTradablePairs")
	}

	err = e.UpdateTradablePairs(false)
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "UpdateTradablePairs")
	}

	_, err = e.GetAccountInfo()
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "GetAccountInfo")
	}

	_, err = e.GetExchangeHistory(p, assetType)
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "GetExchangeHistory")
	}

	_, err = e.GetFundingHistory()
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "GetFundingHistory")
	}

	_, err = e.SubmitOrder(p, exchange.BuyOrderSide, exchange.LimitOrderType, 1000000, 10000000000, "meow")
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "SubmitOrder")
	}

	_, err = e.ModifyOrder(&exchange.ModifyOrder{})
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "ModifyOrder")
	}

	err = e.CancelOrder(&exchange.OrderCancellation{})
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "CancelOrder")
	}

	_, err = e.CancelAllOrders(&exchange.OrderCancellation{})
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "CancelAllOrders")
	}

	_, err = e.GetOrderInfo("1")
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "GetOrderInfo")
	}

	_, err = e.GetOrderHistory(&exchange.GetOrdersRequest{})
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "GetOrderHistory")
	}

	_, err = e.GetActiveOrders(&exchange.GetOrdersRequest{})
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "GetActiveOrders")
	}

	_, err = e.GetDepositAddress(currency.BTC, "")
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "GetDepositAddress")
	}

	_, err = e.WithdrawCryptocurrencyFunds(&exchange.WithdrawRequest{})
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "WithdrawCryptocurrencyFunds")
	}

	_, err = e.WithdrawFiatFunds(&exchange.WithdrawRequest{})
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "WithdrawFiatFunds")
	}
	_, err = e.WithdrawFiatFundsToInternationalBank(&exchange.WithdrawRequest{})
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "WithdrawFiatFundsToInternationalBank")
	}

	return funcs
}
