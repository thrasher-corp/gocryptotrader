package exchange

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"

	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

type Exchange struct{}

// Exchanges returns slice of all current exchanges
func (e Exchange) Exchanges(enabledOnly bool) []string {
	return engine.GetExchanges(enabledOnly)
}

// GetExchange returns IBotExchange for exchange or error if exchange is not found
func (e Exchange) GetExchange(exch string) (exchange.IBotExchange, error) {
	ex := engine.GetExchangeByName(exch)

	if ex == nil {
		return nil, fmt.Errorf("%v exchange not found", exch)
	}

	return ex, nil
}

func (e Exchange) IsEnabled(exch string) (rtn bool) {
	ex, err := e.GetExchange(exch)

	if err != nil {
		return
	}

	return ex.IsEnabled()
}

func (e Exchange) Orderbook(exch string, pair currency.Pair, item asset.Item) (*orderbook.Base, error) {
	ex, err := e.GetExchange(exch)
	if err != nil {
		return nil, err
	}

	ob, err := ex.FetchOrderbook(pair, item)
	if err != nil {
		return nil, err
	}
	return &ob, nil
}

// Ticker returns ticker for provided currency pair & asset type
func (e Exchange) Ticker(exch string, pair currency.Pair, item asset.Item) (*ticker.Price, error) {
	ex, err := e.GetExchange(exch)
	if err != nil {
		return nil, err
	}

	tx, err := ex.FetchTicker(pair, item)
	if err != nil {
		return nil, err
	}

	return &tx, nil
}

func (e Exchange) Pairs(exch string, enabledOnly bool, item asset.Item) (currency.Pairs, error) {
	x, err := engine.Bot.Config.GetExchangeConfig(exch)
	if err != nil {
		return nil, err
	}

	if enabledOnly {
		return x.CurrencyPairs.Get(item).Enabled, nil
	}
	return x.CurrencyPairs.Get(item).Available, nil
}

func (e Exchange) QueryOrder() error {

	return nil
}

func (e Exchange) SubmitOrder() error {
	return nil
}

func (e Exchange) CancelOrder() error {
	return nil
}

func (e Exchange) AccountInformation(exch string) (modules.AccountInfo, error) {
	ex, err := e.GetExchange(exch)
	if err != nil {
		return modules.AccountInfo{}, err
	}

	r, _ := ex.GetAccountInfo()

	fmt.Println(r)

	return modules.AccountInfo{}, nil
}
