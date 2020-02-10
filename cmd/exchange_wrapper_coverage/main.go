package main

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
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
	exchanges := engine.GetExchanges()
	for x := range exchanges {
		wg.Add(1)
		go func(num int) {
			name := exchanges[num].GetName()
			results[name] = testWrappers(exchanges[num])
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
	assetType := asset.Spot
	if !e.SupportsAsset(assetType) {
		assets := e.GetAssetTypes()
		rand.Seed(time.Now().Unix())
		assetType = assets[rand.Intn(len(assets))]
	}

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

	_, err = e.FetchTradablePairs(asset.Spot)
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "FetchTradablePairs")
	}

	err = e.UpdateTradablePairs(false)
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "UpdateTradablePairs")
	}

	_, err = e.FetchAccountInfo()
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

	s := &order.Submit{
		Pair:      p,
		OrderSide: order.Buy,
		OrderType: order.Limit,
		Amount:    1000000,
		Price:     10000000000,
		ClientID:  "meow",
	}
	_, err = e.SubmitOrder(s)
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "SubmitOrder")
	}

	_, err = e.ModifyOrder(&order.Modify{})
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "ModifyOrder")
	}

	err = e.CancelOrder(&order.Cancel{})
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "CancelOrder")
	}

	_, err = e.CancelAllOrders(&order.Cancel{})
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "CancelAllOrders")
	}

	_, err = e.GetOrderInfo("1")
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "GetOrderInfo")
	}

	_, err = e.GetOrderHistory(&order.GetOrdersRequest{})
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "GetOrderHistory")
	}

	_, err = e.GetActiveOrders(&order.GetOrdersRequest{})
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "GetActiveOrders")
	}

	_, err = e.GetDepositAddress(currency.BTC, "")
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "GetDepositAddress")
	}

	_, err = e.WithdrawCryptocurrencyFunds(&withdraw.Request{})
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "WithdrawCryptocurrencyFunds")
	}

	_, err = e.WithdrawFiatFunds(&withdraw.Request{})
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "WithdrawFiatFunds")
	}
	_, err = e.WithdrawFiatFundsToInternationalBank(&withdraw.Request{})
	if err == common.ErrNotYetImplemented {
		funcs = append(funcs, "WithdrawFiatFundsToInternationalBank")
	}

	return funcs
}
