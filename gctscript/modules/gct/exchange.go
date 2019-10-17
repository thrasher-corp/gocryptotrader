package gct

import (
	"github.com/d5/tengo/objects"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

var exchangeModule = map[string]objects.Object{
	"orderbook": &objects.UserFunction{Name: "orderbook", Value: exchangeOrderbook},
	"ticker":    &objects.UserFunction{Name: "ticker", Value: exchangeTicker},
	"exchanges": &objects.UserFunction{Name: "exchanges", Value: exchangeExchanges},
}

func exchangeOrderbook(args ...objects.Object) (ret objects.Object, err error) {
	if len(args) != 4 {
		err = objects.ErrWrongNumArguments
		return
	}

	exchangeName, _ := objects.ToString(args[0])
	currencyPair, _ := objects.ToString(args[1])
	delimiter, _ := objects.ToString(args[2])
	assetTypeParam, _ := objects.ToString(args[3])

	pairs := currency.NewPairDelimiter(currencyPair, delimiter)
	assetType := asset.Item(assetTypeParam)

	ob, err := modules.Wrapper.Orderbook(exchangeName, pairs, assetType)
	if err != nil {
		return nil, err
	}

	var asks, bids objects.Array

	for x := range ob.Asks {
		temp := make(map[string]objects.Object, 2)
		temp["amount"] = &objects.Float{Value: ob.Asks[x].Amount}
		temp["price"] = &objects.Float{Value: ob.Asks[x].Price}
		asks.Value = append(asks.Value, &objects.Map{Value: temp})
	}

	for x := range ob.Bids {
		temp := make(map[string]objects.Object, 2)
		temp["amount"] = &objects.Float{Value: ob.Bids[x].Amount}
		temp["price"] = &objects.Float{Value: ob.Bids[x].Price}
		bids.Value = append(bids.Value, &objects.Map{Value: temp})
	}

	data := make(map[string]objects.Object, 13)
	data["exchange"] = &objects.String{Value: ob.ExchangeName}
	data["pair"] = &objects.String{Value: ob.Pair.String()}
	data["asks"] = &asks
	data["bids"] = &bids
	data["asset"] = &objects.String{Value: ob.AssetType.String()}

	r := objects.Map{
		Value: data,
	}

	return &r, nil
}

func exchangeTicker(args ...objects.Object) (ret objects.Object, err error) {
	if len(args) != 4 {
		err = objects.ErrWrongNumArguments
		return
	}

	exchangeName, _ := objects.ToString(args[0])
	currencyPair, _ := objects.ToString(args[1])
	delimiter, _ := objects.ToString(args[2])
	assetTypeParam, _ := objects.ToString(args[3])

	pairs := currency.NewPairDelimiter(currencyPair, delimiter)
	assetType := asset.Item(assetTypeParam)

	tx, err := modules.Wrapper.Ticker(exchangeName, pairs, assetType)
	if err != nil {
		return nil, err
	}

	data := make(map[string]objects.Object, 13)
	data["exchange"] = &objects.String{Value: tx.ExchangeName}
	data["last"] = &objects.Float{Value: tx.Last}
	data["High"] = &objects.Float{Value: tx.High}
	data["Low"] = &objects.Float{Value: tx.Low}
	data["bid"] = &objects.Float{Value: tx.Bid}
	data["ask"] = &objects.Float{Value: tx.Ask}
	data["volume"] = &objects.Float{Value: tx.Volume}
	data["quotevolume"] = &objects.Float{Value: tx.QuoteVolume}
	data["priceath"] = &objects.Float{Value: tx.PriceATH}
	data["open"] = &objects.Float{Value: tx.Open}
	data["close"] = &objects.Float{Value: tx.Close}
	data["pair"] = &objects.String{Value: tx.Pair.String()}
	data["asset"] = &objects.String{Value: tx.AssetType.String()}
	data["updated"] = &objects.Time{Value: tx.LastUpdated}

	r := objects.Map{
		Value: data,
	}

	return &r, nil
}

func exchangeExchanges(args ...objects.Object) (ret objects.Object, err error) {
	if len(args) != 1 {
		err = objects.ErrWrongNumArguments
		return
	}

	enabledOnly, _ := objects.ToBool(args[0])
	rtnValue := modules.Wrapper.Exchanges(enabledOnly)

	r := objects.Array{}
	for x := range rtnValue {
		r.Value = append(r.Value, &objects.String{Value: rtnValue[x]})
	}

	return &r, nil
}
