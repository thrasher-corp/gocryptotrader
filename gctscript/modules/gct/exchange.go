package gct

import (
	"fmt"
	"time"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

var exchangeModule = map[string]objects.Object{
	"orderbook":      &objects.UserFunction{Name: "orderbook", Value: ExchangeOrderbook},
	"ticker":         &objects.UserFunction{Name: "ticker", Value: ExchangeTicker},
	"exchanges":      &objects.UserFunction{Name: "exchanges", Value: ExchangeExchanges},
	"pairs":          &objects.UserFunction{Name: "pairs", Value: ExchangePairs},
	"accountinfo":    &objects.UserFunction{Name: "accountinfo", Value: ExchangeAccountInfo},
	"depositaddress": &objects.UserFunction{Name: "depositaddress", Value: ExchangeDepositAddress},
	"orderquery":     &objects.UserFunction{Name: "orderquery", Value: ExchangeOrderQuery},
	"ordercancel":    &objects.UserFunction{Name: "ordercancel", Value: ExchangeOrderCancel},
	"ordersubmit":    &objects.UserFunction{Name: "ordersubmit", Value: ExchangeOrderSubmit},
	"withdrawcrypto": &objects.UserFunction{Name: "withdrawcrypto", Value: ExchangeWithdrawCrypto},
	"withdrawfiat":   &objects.UserFunction{Name: "withdrawfiat", Value: ExchangeWithdrawFiat},
	"ohlcv":          &objects.UserFunction{Name: "ohlcv", Value: exchangeOHLCV},
}

// ExchangeOrderbook returns orderbook for requested exchange & currencypair
func ExchangeOrderbook(args ...objects.Object) (objects.Object, error) {
	if len(args) != 4 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, ok := objects.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, exchangeName)
	}
	currencyPair, ok := objects.ToString(args[1])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, currencyPair)
	}
	delimiter, ok := objects.ToString(args[2])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, delimiter)
	}
	assetTypeParam, ok := objects.ToString(args[3])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, assetTypeParam)
	}

	pair, err := currency.NewPairDelimiter(currencyPair, delimiter)
	if err != nil {
		return nil, err
	}

	assetType, err := asset.New(assetTypeParam)
	if err != nil {
		return nil, err
	}

	ob, err := wrappers.GetWrapper().Orderbook(exchangeName, pair, assetType)
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

	data := make(map[string]objects.Object, 5)
	data["exchange"] = &objects.String{Value: ob.Exchange}
	data["pair"] = &objects.String{Value: ob.Pair.String()}
	data["asks"] = &asks
	data["bids"] = &bids
	data["asset"] = &objects.String{Value: ob.Asset.String()}

	return &objects.Map{
		Value: data,
	}, nil
}

// ExchangeTicker returns ticker data for requested exchange and currency pair
func ExchangeTicker(args ...objects.Object) (objects.Object, error) {
	if len(args) != 4 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, ok := objects.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, exchangeName)
	}
	currencyPair, ok := objects.ToString(args[1])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, currencyPair)
	}
	delimiter, ok := objects.ToString(args[2])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, delimiter)
	}
	assetTypeParam, ok := objects.ToString(args[3])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, assetTypeParam)
	}

	pair, err := currency.NewPairDelimiter(currencyPair, delimiter)
	if err != nil {
		return nil, err
	}

	assetType, err := asset.New(assetTypeParam)
	if err != nil {
		return nil, err
	}

	tx, err := wrappers.GetWrapper().Ticker(exchangeName, pair, assetType)
	if err != nil {
		return nil, err
	}

	data := make(map[string]objects.Object, 14)
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

	return &objects.Map{
		Value: data,
	}, nil
}

// ExchangeExchanges returns list of exchanges either enabled or all
func ExchangeExchanges(args ...objects.Object) (objects.Object, error) {
	if len(args) != 1 {
		return nil, objects.ErrWrongNumArguments
	}

	enabledOnly, ok := objects.ToBool(args[0])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, enabledOnly)
	}
	rtnValue := wrappers.GetWrapper().Exchanges(enabledOnly)

	r := objects.Array{}
	for x := range rtnValue {
		r.Value = append(r.Value, &objects.String{Value: rtnValue[x]})
	}

	return &r, nil
}

// ExchangePairs returns currency pairs for requested exchange
func ExchangePairs(args ...objects.Object) (objects.Object, error) {
	if len(args) != 3 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, ok := objects.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, exchangeName)
	}
	enabledOnly, ok := objects.ToBool(args[1])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, enabledOnly)
	}
	assetTypeParam, ok := objects.ToString(args[2])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, assetTypeParam)
	}
	assetType, err := asset.New(assetTypeParam)
	if err != nil {
		return nil, err
	}

	rtnValue, err := wrappers.GetWrapper().Pairs(exchangeName, enabledOnly, assetType)
	if err != nil {
		return nil, err
	}

	r := objects.Array{}
	pairs := *(*[]currency.Pair)(rtnValue)
	for x := range pairs {
		r.Value = append(r.Value, &objects.String{Value: pairs[x].String()})
	}
	return &r, nil
}

// ExchangeAccountInfo returns account information for requested exchange
func ExchangeAccountInfo(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, ok := objects.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, exchangeName)
	}
	assetString, ok := objects.ToString(args[1])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, assetString)
	}
	assetType, err := asset.New(assetString)
	if err != nil {
		return nil, err
	}
	rtnValue, err := wrappers.GetWrapper().AccountInformation(exchangeName, assetType)
	if err != nil {
		return nil, err
	}

	var funds objects.Array
	for x := range rtnValue.Accounts {
		for y := range rtnValue.Accounts[x].Currencies {
			temp := make(map[string]objects.Object, 3)
			temp["name"] = &objects.String{Value: rtnValue.Accounts[x].Currencies[y].CurrencyName.String()}
			temp["total"] = &objects.Float{Value: rtnValue.Accounts[x].Currencies[y].TotalValue}
			temp["hold"] = &objects.Float{Value: rtnValue.Accounts[x].Currencies[y].Hold}
			funds.Value = append(funds.Value, &objects.Map{Value: temp})
		}
	}

	data := make(map[string]objects.Object, 2)
	data["exchange"] = &objects.String{Value: rtnValue.Exchange}
	data["currencies"] = &funds

	return &objects.Map{
		Value: data,
	}, nil
}

// ExchangeOrderQuery query order on exchange
func ExchangeOrderQuery(args ...objects.Object) (objects.Object, error) {
	if len(args) < 2 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, ok := objects.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, exchangeName)
	}
	orderID, ok := objects.ToString(args[1])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, orderID)
	}

	var pair currency.Pair
	assetTypeString := asset.Spot.String()

	switch len(args) {
	case 4:
		assetTypeString, ok = objects.ToString(args[3])
		if !ok {
			return nil, fmt.Errorf(ErrParameterConvertFailed, assetTypeString)
		}
		fallthrough
	case 3:
		currencyPairString, isOk := objects.ToString(args[2])
		if !isOk {
			return nil, fmt.Errorf(ErrParameterConvertFailed, currencyPairString)
		}

		var err error
		pair, err = currency.NewPairFromString(currencyPairString)
		if err != nil {
			return nil, fmt.Errorf(ErrParameterConvertFailed, currencyPairString)
		}
	}

	assetType, err := asset.New(assetTypeString)
	if err != nil {
		return nil, err
	}

	orderDetails, err := wrappers.GetWrapper().QueryOrder(exchangeName, orderID, pair, assetType)
	if err != nil {
		return nil, err
	}

	var tradeHistory objects.Array
	for x := range orderDetails.Trades {
		temp := make(map[string]objects.Object, 7)
		temp["timestamp"] = &objects.Time{Value: orderDetails.Trades[x].Timestamp}
		temp["price"] = &objects.Float{Value: orderDetails.Trades[x].Price}
		temp["fee"] = &objects.Float{Value: orderDetails.Trades[x].Fee}
		temp["amount"] = &objects.Float{Value: orderDetails.Trades[x].Amount}
		temp["type"] = &objects.String{Value: orderDetails.Trades[x].Type.String()}
		temp["side"] = &objects.String{Value: orderDetails.Trades[x].Side.String()}
		temp["description"] = &objects.String{Value: orderDetails.Trades[x].Description}
		tradeHistory.Value = append(tradeHistory.Value, &objects.Map{Value: temp})
	}

	data := make(map[string]objects.Object, 14)
	data["exchange"] = &objects.String{Value: orderDetails.Exchange}
	data["id"] = &objects.String{Value: orderDetails.ID}
	data["accountid"] = &objects.String{Value: orderDetails.AccountID}
	data["currencypair"] = &objects.String{Value: orderDetails.Pair.String()}
	data["price"] = &objects.Float{Value: orderDetails.Price}
	data["amount"] = &objects.Float{Value: orderDetails.Amount}
	data["amountexecuted"] = &objects.Float{Value: orderDetails.ExecutedAmount}
	data["amountremaining"] = &objects.Float{Value: orderDetails.RemainingAmount}
	data["fee"] = &objects.Float{Value: orderDetails.Fee}
	data["side"] = &objects.String{Value: orderDetails.Side.String()}
	data["type"] = &objects.String{Value: orderDetails.Type.String()}
	data["date"] = &objects.String{Value: orderDetails.Date.String()}
	data["status"] = &objects.String{Value: orderDetails.Status.String()}
	data["trades"] = &tradeHistory

	return &objects.Map{
		Value: data,
	}, nil
}

// ExchangeOrderCancel cancels order on requested exchange
func ExchangeOrderCancel(args ...objects.Object) (objects.Object, error) {
	if len(args) < 2 || len(args) > 4 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, ok := objects.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, exchangeName)
	}
	if exchangeName == "" {
		return nil, fmt.Errorf(ErrEmptyParameter, "exchange name")
	}
	var orderID string
	orderID, ok = objects.ToString(args[1])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, orderID)
	}
	if orderID == "" {
		return nil, fmt.Errorf(ErrEmptyParameter, "orderID")
	}
	var err error
	var cp currency.Pair
	if len(args) > 2 {
		var currencyPair string
		currencyPair, ok = objects.ToString(args[2])
		if !ok {
			return nil, fmt.Errorf(ErrParameterConvertFailed, currencyPair)
		}
		cp, err = currency.NewPairFromString(currencyPair)
		if err != nil {
			return nil, err
		}
	}
	var a asset.Item
	if len(args) > 3 {
		var assetType string
		assetType, ok = objects.ToString(args[3])
		if !ok {
			return nil, fmt.Errorf(ErrParameterConvertFailed, assetType)
		}
		a, err = asset.New(assetType)
		if err != nil {
			return nil, err
		}
	}

	var isCancelled bool
	isCancelled, err = wrappers.GetWrapper().CancelOrder(exchangeName, orderID, cp, a)
	if err != nil {
		return nil, err
	}

	if isCancelled {
		return objects.TrueValue, nil
	}
	return objects.FalseValue, nil
}

// ExchangeOrderSubmit submit order on exchange
func ExchangeOrderSubmit(args ...objects.Object) (objects.Object, error) {
	if len(args) != 9 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, ok := objects.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, exchangeName)
	}
	currencyPair, ok := objects.ToString(args[1])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, currencyPair)
	}
	delimiter, ok := objects.ToString(args[2])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, delimiter)
	}
	pair, err := currency.NewPairDelimiter(currencyPair, delimiter)
	if err != nil {
		return nil, err
	}
	orderType, ok := objects.ToString(args[3])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, orderType)
	}
	orderSide, ok := objects.ToString(args[4])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, orderSide)
	}
	orderPrice, ok := objects.ToFloat64(args[5])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, orderPrice)
	}
	orderAmount, ok := objects.ToFloat64(args[6])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, orderAmount)
	}
	orderClientID, ok := objects.ToString(args[7])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, orderClientID)
	}
	assetType, ok := objects.ToString(args[8])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, orderClientID)
	}
	a, err := asset.New(assetType)
	if err != nil {
		return nil, err
	}

	tempSubmit := &order.Submit{
		Pair:      pair,
		Type:      order.Type(orderType),
		Side:      order.Side(orderSide),
		Price:     orderPrice,
		Amount:    orderAmount,
		ClientID:  orderClientID,
		AssetType: a,
		Exchange:  exchangeName,
	}

	rtn, err := wrappers.GetWrapper().SubmitOrder(tempSubmit)
	if err != nil {
		return nil, err
	}

	data := make(map[string]objects.Object, 2)
	data["orderid"] = &objects.String{Value: rtn.OrderID}
	if rtn.IsOrderPlaced {
		data["isorderplaced"] = objects.TrueValue
	} else {
		data["isorderplaced"] = objects.FalseValue
	}

	return &objects.Map{
		Value: data,
	}, nil
}

// ExchangeDepositAddress returns deposit address (if supported by exchange)
func ExchangeDepositAddress(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, ok := objects.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, exchangeName)
	}
	currencyCode, ok := objects.ToString(args[1])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, currencyCode)
	}

	currCode := currency.NewCode(currencyCode)

	rtn, err := wrappers.GetWrapper().DepositAddress(exchangeName, currCode)
	if err != nil {
		return nil, err
	}

	return &objects.String{Value: rtn}, nil
}

// ExchangeWithdrawCrypto submit request to withdraw crypto assets
func ExchangeWithdrawCrypto(args ...objects.Object) (objects.Object, error) {
	if len(args) != 7 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, ok := objects.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, exchangeName)
	}
	cur, ok := objects.ToString(args[1])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, cur)
	}
	address, ok := objects.ToString(args[2])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, address)
	}
	addressTag, ok := objects.ToString(args[3])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, addressTag)
	}
	amount, ok := objects.ToFloat64(args[4])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, amount)
	}
	feeAmount, ok := objects.ToFloat64(args[5])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, feeAmount)
	}
	description, ok := objects.ToString(args[6])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, description)
	}

	withdrawRequest := &withdraw.Request{
		Exchange: exchangeName,
		Crypto: withdraw.CryptoRequest{
			Address:    address,
			AddressTag: addressTag,
			FeeAmount:  feeAmount,
		},
		Currency:    currency.NewCode(cur),
		Description: description,
		Amount:      amount,
	}

	rtn, err := wrappers.GetWrapper().WithdrawalCryptoFunds(withdrawRequest)
	if err != nil {
		return nil, err
	}

	return &objects.String{Value: rtn}, nil
}

// ExchangeWithdrawFiat submit request to withdraw fiat assets
func ExchangeWithdrawFiat(args ...objects.Object) (objects.Object, error) {
	if len(args) != 5 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, ok := objects.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, exchangeName)
	}
	cur, ok := objects.ToString(args[1])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, cur)
	}
	description, ok := objects.ToString(args[2])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, description)
	}
	amount, ok := objects.ToFloat64(args[3])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, amount)
	}
	bankAccountID, ok := objects.ToString(args[4])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, bankAccountID)
	}

	withdrawRequest := &withdraw.Request{
		Exchange:    exchangeName,
		Currency:    currency.NewCode(cur),
		Description: description,
		Amount:      amount,
	}

	rtn, err := wrappers.GetWrapper().WithdrawalFiatFunds(bankAccountID, withdrawRequest)
	if err != nil {
		return nil, err
	}

	return &objects.String{Value: rtn}, nil
}

// OHLCV defines a custom Open High Low Close Volume tengo object
type OHLCV struct {
	objects.Map
}

// TypeName returns the name of the custom type.
func (o *OHLCV) TypeName() string {
	return indicators.OHLCV
}

func exchangeOHLCV(args ...objects.Object) (objects.Object, error) {
	if len(args) != 7 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, ok := objects.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, exchangeName)
	}
	currencyPair, ok := objects.ToString(args[1])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, currencyPair)
	}
	delimiter, ok := objects.ToString(args[2])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, delimiter)
	}
	assetTypeParam, ok := objects.ToString(args[3])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, assetTypeParam)
	}

	startTime, ok := objects.ToTime(args[4])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, startTime)
	}

	endTime, ok := objects.ToTime(args[5])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, endTime)
	}

	intervalStr, ok := objects.ToString(args[6])
	if !ok {
		return nil, fmt.Errorf(ErrParameterConvertFailed, endTime)
	}
	interval, err := parseInterval(intervalStr)
	if err != nil {
		return nil, err
	}
	pair, err := currency.NewPairDelimiter(currencyPair, delimiter)
	if err != nil {
		return nil, err
	}
	assetType, err := asset.New(assetTypeParam)
	if err != nil {
		return nil, err
	}

	ret, err := wrappers.GetWrapper().OHLCV(exchangeName, pair, assetType, startTime, endTime, kline.Interval(interval))
	if err != nil {
		return nil, err
	}

	var candles objects.Array
	for x := range ret.Candles {
		candle := &objects.Array{}
		candle.Value = append(candle.Value, &objects.Int{Value: ret.Candles[x].Time.Unix()},
			&objects.Float{Value: ret.Candles[x].Open},
			&objects.Float{Value: ret.Candles[x].High},
			&objects.Float{Value: ret.Candles[x].Low},
			&objects.Float{Value: ret.Candles[x].Close},
			&objects.Float{Value: ret.Candles[x].Volume},
		)

		candles.Value = append(candles.Value, candle)
	}

	retValue := make(map[string]objects.Object, 5)
	retValue["exchange"] = &objects.String{Value: ret.Exchange}
	retValue["pair"] = &objects.String{Value: ret.Pair.String()}
	retValue["asset"] = &objects.String{Value: ret.Asset.String()}
	retValue["intervals"] = &objects.String{Value: ret.Interval.String()}
	retValue["candles"] = &candles

	c := new(OHLCV)
	c.Value = retValue
	return c, nil
}

// parseInterval will parse the interval param of indictors that have them and convert to time.Duration
func parseInterval(in string) (time.Duration, error) {
	if !common.StringDataContainsInsensitive(supportedDurations, in) {
		return time.Nanosecond, errInvalidInterval
	}
	switch in {
	case "1d":
		in = "24h"
	case "3d":
		in = "72h"
	case "1w":
		in = "168h"
	}
	return time.ParseDuration(in)
}
