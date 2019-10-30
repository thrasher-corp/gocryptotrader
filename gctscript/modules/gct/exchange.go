package gct

import (
	"fmt"
	"strings"

	"github.com/d5/tengo/objects"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

var exchangeModule = map[string]objects.Object{
	"orderbook":   &objects.UserFunction{Name: "orderbook", Value: exchangeOrderbook},
	"ticker":      &objects.UserFunction{Name: "ticker", Value: exchangeTicker},
	"exchanges":   &objects.UserFunction{Name: "exchanges", Value: exchangeExchanges},
	"pairs":       &objects.UserFunction{Name: "pairs", Value: exchangePairs},
	"accountinfo": &objects.UserFunction{Name: "accountinfo", Value: exchangeAccountInfo},
	"order":       &objects.UserFunction{Name: "order", Value: exchangeOrderQuery},
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

func exchangePairs(args ...objects.Object) (ret objects.Object, err error) {
	if len(args) != 3 {
		err = objects.ErrWrongNumArguments
		return
	}

	exchangeName, _ := objects.ToString(args[0])
	enabledOnly, _ := objects.ToBool(args[1])
	assetTypeParam, _ := objects.ToString(args[2])
	assetType := asset.Item(strings.ToLower(assetTypeParam))

	rtnValue, err := modules.Wrapper.Pairs(exchangeName, enabledOnly, assetType)
	if err != nil {
		return nil, err
	}

	r := objects.Array{}
	for x := range rtnValue {
		r.Value = append(r.Value, &objects.String{Value: rtnValue[x].String()})
	}

	return &r, nil
}

func exchangeAccountInfo(args ...objects.Object) (ret objects.Object, err error) {
	if len(args) != 1 {
		err = objects.ErrWrongNumArguments
		return
	}

	exchangeName, _ := objects.ToString(args[0])
	rtnValue, err := modules.Wrapper.AccountInformation(exchangeName)
	if err != nil {
		return nil, err
	}

	fmt.Println(rtnValue)

	return nil, nil
}

func exchangeOrderQuery(args ...objects.Object) (ret objects.Object, err error) {
	if len(args) != 2 {
		err = objects.ErrWrongNumArguments
		return
	}

	exchangeName, _ := objects.ToString(args[0])
	orderID, _ := objects.ToString(args[1])

	orderDetails, err := modules.Wrapper.QueryOrder(exchangeName, orderID)
	if err != nil {
		return nil, err
	}

	fmt.Printf("%+v", *orderDetails)

	return nil, nil
}

/*
type Order struct {
	ID              int64           `json:"id"`
	Currency        string          `json:"currency"`
	Instrument      string          `json:"instrument"`
	OrderSide       string          `json:"orderSide"`
	OrderType       string          `json:"ordertype"`
	CreationTime    float64         `json:"creationTime"`
	Status          string          `json:"status"`
	ErrorMessage    string          `json:"errorMessage"`
	Price           float64         `json:"price"`
	Volume          float64         `json:"volume"`
	OpenVolume      float64         `json:"openVolume"`
	ClientRequestID string          `json:"clientRequestId"`
	Trades          []TradeResponse `json:"trades"`
}
*/
