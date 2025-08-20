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

const (
	orderbookFunc       = "orderbook"
	tickerFunc          = "ticker"
	exchangesFunc       = "exchanges"
	pairsFunc           = "pairs"
	accountBalancesFunc = "accountbalances"
	depositAddressFunc  = "depositaddress"
	orderQueryFunc      = "orderquery"
	orderCancelFunc     = "ordercancel"
	orderSubmitFunc     = "ordersubmit"
	withdrawCryptoFunc  = "withdrawcrypto"
	withdrawFiatFunc    = "withdrawfiat"
	ohlcvFunc           = "ohlcv"
)

var exchangeModule = map[string]objects.Object{
	orderbookFunc:       &objects.UserFunction{Name: orderbookFunc, Value: ExchangeOrderbook},
	tickerFunc:          &objects.UserFunction{Name: tickerFunc, Value: ExchangeTicker},
	exchangesFunc:       &objects.UserFunction{Name: exchangesFunc, Value: ExchangeExchanges},
	pairsFunc:           &objects.UserFunction{Name: pairsFunc, Value: ExchangePairs},
	accountBalancesFunc: &objects.UserFunction{Name: accountBalancesFunc, Value: ExchangeAccountBalances},
	depositAddressFunc:  &objects.UserFunction{Name: depositAddressFunc, Value: ExchangeDepositAddress},
	orderQueryFunc:      &objects.UserFunction{Name: orderQueryFunc, Value: ExchangeOrderQuery},
	orderCancelFunc:     &objects.UserFunction{Name: orderCancelFunc, Value: ExchangeOrderCancel},
	orderSubmitFunc:     &objects.UserFunction{Name: orderSubmitFunc, Value: ExchangeOrderSubmit},
	withdrawCryptoFunc:  &objects.UserFunction{Name: withdrawCryptoFunc, Value: ExchangeWithdrawCrypto},
	withdrawFiatFunc:    &objects.UserFunction{Name: withdrawFiatFunc, Value: ExchangeWithdrawFiat},
	ohlcvFunc:           &objects.UserFunction{Name: ohlcvFunc, Value: exchangeOHLCV},
}

// ExchangeOrderbook returns orderbook for requested exchange & currencypair
func ExchangeOrderbook(args ...objects.Object) (objects.Object, error) {
	if len(args) != 5 {
		return nil, objects.ErrWrongNumArguments
	}

	scriptCtx, ok := objects.ToInterface(args[0]).(*Context)
	if !ok {
		return nil, constructRuntimeError(1, orderbookFunc, "*gct.Context", args[0])
	}
	exchangeName, ok := objects.ToString(args[1])
	if !ok {
		return nil, constructRuntimeError(2, orderbookFunc, "string", args[1])
	}
	currencyPair, ok := objects.ToString(args[2])
	if !ok {
		return nil, constructRuntimeError(3, orderbookFunc, "string", args[2])
	}
	delimiter, ok := objects.ToString(args[3])
	if !ok {
		return nil, constructRuntimeError(4, orderbookFunc, "string", args[3])
	}
	assetTypeParam, ok := objects.ToString(args[4])
	if !ok {
		return nil, constructRuntimeError(5, orderbookFunc, "string", args[4])
	}

	pair, err := currency.NewPairDelimiter(currencyPair, delimiter)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	assetType, err := asset.New(assetTypeParam)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	ctx := processScriptContext(scriptCtx)
	ob, err := wrappers.GetWrapper().Orderbook(ctx, exchangeName, pair, assetType)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	asks := objects.Array{Value: make([]objects.Object, len(ob.Asks))}
	for x := range ob.Asks {
		temp := make(map[string]objects.Object, 2)
		temp["amount"] = &objects.Float{Value: ob.Asks[x].Amount}
		temp["price"] = &objects.Float{Value: ob.Asks[x].Price}
		asks.Value[x] = &objects.Map{Value: temp}
	}

	bids := objects.Array{Value: make([]objects.Object, len(ob.Bids))}
	for x := range ob.Bids {
		temp := make(map[string]objects.Object, 2)
		temp["amount"] = &objects.Float{Value: ob.Bids[x].Amount}
		temp["price"] = &objects.Float{Value: ob.Bids[x].Price}
		bids.Value[x] = &objects.Map{Value: temp}
	}

	data := make(map[string]objects.Object, 5)
	data["exchange"] = &objects.String{Value: ob.Exchange}
	data["pair"] = &objects.String{Value: ob.Pair.String()}
	data["asks"] = &asks
	data["bids"] = &bids
	data["asset"] = &objects.String{Value: ob.Asset.String()}

	return &objects.Map{Value: data}, nil
}

// ExchangeTicker returns ticker data for requested exchange and currency pair
func ExchangeTicker(args ...objects.Object) (objects.Object, error) {
	if len(args) != 5 {
		return nil, objects.ErrWrongNumArguments
	}

	scriptCtx, ok := objects.ToInterface(args[0]).(*Context)
	if !ok {
		return nil, constructRuntimeError(1, tickerFunc, "*gct.Context", args[0])
	}

	exchangeName, ok := objects.ToString(args[1])
	if !ok {
		return nil, constructRuntimeError(2, tickerFunc, "string", args[1])
	}
	currencyPair, ok := objects.ToString(args[2])
	if !ok {
		return nil, constructRuntimeError(3, tickerFunc, "string", args[2])
	}
	delimiter, ok := objects.ToString(args[3])
	if !ok {
		return nil, constructRuntimeError(4, tickerFunc, "string", args[3])
	}
	assetTypeParam, ok := objects.ToString(args[4])
	if !ok {
		return nil, constructRuntimeError(5, tickerFunc, "string", args[4])
	}

	pair, err := currency.NewPairDelimiter(currencyPair, delimiter)
	if err != nil {
		return nil, err
	}

	assetType, err := asset.New(assetTypeParam)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	ctx := processScriptContext(scriptCtx)
	tx, err := wrappers.GetWrapper().Ticker(ctx, exchangeName, pair, assetType)
	if err != nil {
		return errorResponsef(standardFormatting, err)
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
		return nil, constructRuntimeError(1, exchangesFunc, "bool", args[0])
	}
	rtnValue := wrappers.GetWrapper().Exchanges(enabledOnly)

	r := objects.Array{
		Value: make([]objects.Object, len(rtnValue)),
	}
	for x := range rtnValue {
		r.Value[x] = &objects.String{Value: rtnValue[x]}
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
		return nil, constructRuntimeError(1, pairsFunc, "string", args[0])
	}
	enabledOnly, ok := objects.ToBool(args[1])
	if !ok {
		return nil, constructRuntimeError(2, pairsFunc, "bool", args[1])
	}
	assetTypeParam, ok := objects.ToString(args[2])
	if !ok {
		return nil, constructRuntimeError(3, pairsFunc, "string", args[2])
	}
	assetType, err := asset.New(assetTypeParam)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	pairs, err := wrappers.GetWrapper().Pairs(exchangeName, enabledOnly, assetType)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	r := objects.Array{
		Value: make([]objects.Object, len(*pairs)),
	}

	for x := range *pairs {
		r.Value[x] = &objects.String{Value: (*pairs)[x].String()}
	}
	return &r, nil
}

// ExchangeAccountBalances returns account balances for requested exchange
func ExchangeAccountBalances(args ...objects.Object) (objects.Object, error) {
	if len(args) != 3 {
		return nil, objects.ErrWrongNumArguments
	}

	scriptCtx, ok := objects.ToInterface(args[0]).(*Context)
	if !ok {
		return nil, constructRuntimeError(1, accountBalancesFunc, "*gct.Context", args[0])
	}
	exchangeName, ok := objects.ToString(args[1])
	if !ok {
		return nil, constructRuntimeError(2, accountBalancesFunc, "string", args[1])
	}
	assetString, ok := objects.ToString(args[2])
	if !ok {
		return nil, constructRuntimeError(3, accountBalancesFunc, "string", args[2])
	}
	assetType, err := asset.New(assetString)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	ctx := processScriptContext(scriptCtx)
	rtnValue, err := wrappers.GetWrapper().AccountBalances(ctx, exchangeName, assetType)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	var funds objects.Array
	for i := range rtnValue {
		for curr, bal := range rtnValue[i].Balances {
			funds.Value = append(funds.Value, &objects.Map{Value: map[string]objects.Object{
				"name":  &objects.String{Value: curr.String()},
				"total": &objects.Float{Value: bal.Total},
				"hold":  &objects.Float{Value: bal.Hold},
			}})
		}
	}

	data := make(map[string]objects.Object, 2)
	data["exchange"] = &objects.String{Value: exchangeName}
	data["currencies"] = &funds
	return &objects.Map{Value: data}, nil
}

// ExchangeOrderQuery query order on exchange
func ExchangeOrderQuery(args ...objects.Object) (objects.Object, error) {
	if len(args) < 3 {
		return nil, objects.ErrWrongNumArguments
	}

	scriptCtx, ok := objects.ToInterface(args[0]).(*Context)
	if !ok {
		return nil, constructRuntimeError(1, orderQueryFunc, "*gct.Context", args[0])
	}
	exchangeName, ok := objects.ToString(args[1])
	if !ok {
		return nil, constructRuntimeError(2, orderQueryFunc, "string", args[1])
	}
	orderID, ok := objects.ToString(args[2])
	if !ok {
		return nil, constructRuntimeError(3, orderQueryFunc, "string", args[2])
	}

	var pair currency.Pair
	assetTypeString := asset.Spot.String()

	switch len(args) {
	case 5:
		assetTypeString, ok = objects.ToString(args[4])
		if !ok {
			return nil, constructRuntimeError(5, orderQueryFunc, "string", args[4])
		}
		fallthrough
	case 4:
		currencyPairString, isOk := objects.ToString(args[3])
		if !isOk {
			return nil, constructRuntimeError(4, orderQueryFunc, "string", args[3])
		}

		var err error
		pair, err = currency.NewPairFromString(currencyPairString)
		if err != nil {
			return errorResponsef(standardFormatting, err)
		}
	}

	assetType, err := asset.New(assetTypeString)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	ctx := processScriptContext(scriptCtx)
	orderDetails, err := wrappers.GetWrapper().
		QueryOrder(ctx, exchangeName, orderID, pair, assetType)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	var tradeHistory objects.Array
	tradeHistory.Value = make([]objects.Object, len(orderDetails.Trades))
	for x := range orderDetails.Trades {
		temp := make(map[string]objects.Object, 7)
		temp["timestamp"] = &objects.Time{Value: orderDetails.Trades[x].Timestamp}
		temp["price"] = &objects.Float{Value: orderDetails.Trades[x].Price}
		temp["fee"] = &objects.Float{Value: orderDetails.Trades[x].Fee}
		temp["amount"] = &objects.Float{Value: orderDetails.Trades[x].Amount}
		temp["type"] = &objects.String{Value: orderDetails.Trades[x].Type.String()}
		temp["side"] = &objects.String{Value: orderDetails.Trades[x].Side.String()}
		temp["description"] = &objects.String{Value: orderDetails.Trades[x].Description}
		tradeHistory.Value[x] = &objects.Map{Value: temp}
	}

	data := make(map[string]objects.Object, 14)
	data["exchange"] = &objects.String{Value: orderDetails.Exchange}
	data["id"] = &objects.String{Value: orderDetails.OrderID}
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

	return &objects.Map{Value: data}, nil
}

// ExchangeOrderCancel cancels order on requested exchange
func ExchangeOrderCancel(args ...objects.Object) (objects.Object, error) {
	if len(args) < 3 || len(args) > 5 {
		return nil, objects.ErrWrongNumArguments
	}

	scriptCtx, ok := objects.ToInterface(args[0]).(*Context)
	if !ok {
		return nil, constructRuntimeError(1, orderCancelFunc, "*gct.Context", args[0])
	}
	exchangeName, ok := objects.ToString(args[1])
	if !ok {
		return nil, constructRuntimeError(2, orderCancelFunc, "string", args[1])
	}
	if exchangeName == "" {
		return nil, fmt.Errorf(ErrEmptyParameter, "exchange name")
	}
	var orderID string
	orderID, ok = objects.ToString(args[2])
	if !ok {
		return nil, constructRuntimeError(3, orderCancelFunc, "string", args[2])
	}
	if orderID == "" {
		return nil, fmt.Errorf(ErrEmptyParameter, "orderID")
	}
	var err error
	var cp currency.Pair
	if len(args) > 3 {
		var currencyPair string
		currencyPair, ok = objects.ToString(args[3])
		if !ok {
			return nil, constructRuntimeError(4, orderCancelFunc, "string", args[3])
		}
		cp, err = currency.NewPairFromString(currencyPair)
		if err != nil {
			return errorResponsef(standardFormatting, err)
		}
	}
	var a asset.Item
	if len(args) > 4 {
		var assetType string
		assetType, ok = objects.ToString(args[4])
		if !ok {
			return nil, constructRuntimeError(5, orderCancelFunc, "string", args[4])
		}
		a, err = asset.New(assetType)
		if err != nil {
			return errorResponsef(standardFormatting, err)
		}
	}

	ctx := processScriptContext(scriptCtx)
	isCancelled, err := wrappers.GetWrapper().
		CancelOrder(ctx, exchangeName, orderID, cp, a)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	if isCancelled {
		return objects.TrueValue, nil
	}
	return objects.FalseValue, nil
}

// ExchangeOrderSubmit submit order on exchange
func ExchangeOrderSubmit(args ...objects.Object) (objects.Object, error) {
	if len(args) != 10 {
		return nil, objects.ErrWrongNumArguments
	}

	scriptCtx, ok := objects.ToInterface(args[0]).(*Context)
	if !ok {
		return nil, constructRuntimeError(1, orderSubmitFunc, "*gct.Context", args[0])
	}
	exchangeName, ok := objects.ToString(args[1])
	if !ok {
		return nil, constructRuntimeError(2, orderSubmitFunc, "string", args[1])
	}
	currencyPair, ok := objects.ToString(args[2])
	if !ok {
		return nil, constructRuntimeError(3, orderSubmitFunc, "string", args[2])
	}
	delimiter, ok := objects.ToString(args[3])
	if !ok {
		return nil, constructRuntimeError(4, orderSubmitFunc, "string", args[3])
	}
	orderType, ok := objects.ToString(args[4])
	if !ok {
		return nil, constructRuntimeError(5, orderSubmitFunc, "string", args[4])
	}
	orderSide, ok := objects.ToString(args[5])
	if !ok {
		return nil, constructRuntimeError(6, orderSubmitFunc, "string", args[5])
	}
	orderPrice, ok := objects.ToFloat64(args[6])
	if !ok {
		return nil, constructRuntimeError(7, orderSubmitFunc, "float64", args[6])
	}
	orderAmount, ok := objects.ToFloat64(args[7])
	if !ok {
		return nil, constructRuntimeError(8, orderSubmitFunc, "float64", args[7])
	}
	orderClientID, ok := objects.ToString(args[8])
	if !ok {
		return nil, constructRuntimeError(9, orderSubmitFunc, "string", args[8])
	}
	assetType, ok := objects.ToString(args[9])
	if !ok {
		return nil, constructRuntimeError(10, orderSubmitFunc, "string", args[9])
	}

	pair, err := currency.NewPairDelimiter(currencyPair, delimiter)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	a, err := asset.New(assetType)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	side, err := order.StringToOrderSide(orderSide)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	oType, err := order.StringToOrderType(orderType)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	tempSubmit := &order.Submit{
		Pair:      pair,
		Type:      oType,
		Side:      side,
		Price:     orderPrice,
		Amount:    orderAmount,
		ClientID:  orderClientID,
		AssetType: a,
		Exchange:  exchangeName,
	}

	ctx := processScriptContext(scriptCtx)
	rtn, err := wrappers.GetWrapper().SubmitOrder(ctx, tempSubmit)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	data := make(map[string]objects.Object, 2)
	data["orderid"] = &objects.String{Value: rtn.OrderID}
	data["isorderplaced"] = objects.TrueValue

	return &objects.Map{Value: data}, nil
}

// ExchangeDepositAddress returns deposit address (if supported by exchange)
func ExchangeDepositAddress(args ...objects.Object) (objects.Object, error) {
	if len(args) != 3 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, ok := objects.ToString(args[0])
	if !ok {
		return nil, constructRuntimeError(1, depositAddressFunc, "string", args[0])
	}
	currencyCode, ok := objects.ToString(args[1])
	if !ok {
		return nil, constructRuntimeError(2, depositAddressFunc, "string", args[1])
	}
	chain, ok := objects.ToString(args[2])
	if !ok {
		return nil, constructRuntimeError(3, depositAddressFunc, "string", args[2])
	}

	currCode := currency.NewCode(currencyCode)

	rtn, err := wrappers.GetWrapper().DepositAddress(exchangeName, chain, currCode)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	data := make(map[string]objects.Object, 2)
	data["address"] = &objects.String{Value: rtn.Address}
	data["tag"] = &objects.String{Value: rtn.Tag}
	return &objects.Map{Value: data}, nil
}

// ExchangeWithdrawCrypto submit request to withdraw crypto assets
func ExchangeWithdrawCrypto(args ...objects.Object) (objects.Object, error) {
	if len(args) != 8 {
		return nil, objects.ErrWrongNumArguments
	}

	scriptCtx, ok := objects.ToInterface(args[0]).(*Context)
	if !ok {
		return nil, constructRuntimeError(1, withdrawCryptoFunc, "*gct.Context", args[0])
	}
	exchangeName, ok := objects.ToString(args[1])
	if !ok {
		return nil, constructRuntimeError(2, withdrawCryptoFunc, "string", args[1])
	}
	cur, ok := objects.ToString(args[2])
	if !ok {
		return nil, constructRuntimeError(3, withdrawCryptoFunc, "string", args[2])
	}
	address, ok := objects.ToString(args[3])
	if !ok {
		return nil, constructRuntimeError(4, withdrawCryptoFunc, "string", args[3])
	}
	addressTag, ok := objects.ToString(args[4])
	if !ok {
		return nil, constructRuntimeError(5, withdrawCryptoFunc, "string", args[4])
	}
	amount, ok := objects.ToFloat64(args[5])
	if !ok {
		return nil, constructRuntimeError(6, withdrawCryptoFunc, "float64", args[5])
	}
	feeAmount, ok := objects.ToFloat64(args[6])
	if !ok {
		return nil, constructRuntimeError(7, withdrawCryptoFunc, "float64", args[6])
	}
	description, ok := objects.ToString(args[7])
	if !ok {
		return nil, constructRuntimeError(8, withdrawCryptoFunc, "string", args[7])
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

	ctx := processScriptContext(scriptCtx)
	rtn, err := wrappers.GetWrapper().WithdrawalCryptoFunds(ctx, withdrawRequest)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	return &objects.String{Value: rtn}, nil
}

// ExchangeWithdrawFiat submit request to withdraw fiat assets
func ExchangeWithdrawFiat(args ...objects.Object) (objects.Object, error) {
	if len(args) != 6 {
		return nil, objects.ErrWrongNumArguments
	}

	scriptCtx, ok := objects.ToInterface(args[0]).(*Context)
	if !ok {
		return nil, constructRuntimeError(1, withdrawFiatFunc, "*gct.Context", args[0])
	}
	exchangeName, ok := objects.ToString(args[1])
	if !ok {
		return nil, constructRuntimeError(2, withdrawFiatFunc, "string", args[1])
	}
	cur, ok := objects.ToString(args[2])
	if !ok {
		return nil, constructRuntimeError(3, withdrawFiatFunc, "string", args[2])
	}
	description, ok := objects.ToString(args[3])
	if !ok {
		return nil, constructRuntimeError(4, withdrawFiatFunc, "string", args[3])
	}
	amount, ok := objects.ToFloat64(args[4])
	if !ok {
		return nil, constructRuntimeError(5, withdrawFiatFunc, "float64", args[4])
	}
	bankAccountID, ok := objects.ToString(args[5])
	if !ok {
		return nil, constructRuntimeError(6, withdrawFiatFunc, "string", args[5])
	}

	withdrawRequest := &withdraw.Request{
		Exchange:    exchangeName,
		Currency:    currency.NewCode(cur),
		Description: description,
		Amount:      amount,
	}

	ctx := processScriptContext(scriptCtx)
	rtn, err := wrappers.GetWrapper().
		WithdrawalFiatFunds(ctx, bankAccountID, withdrawRequest)
	if err != nil {
		return errorResponsef(standardFormatting, err)
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
	if len(args) != 8 {
		return nil, objects.ErrWrongNumArguments
	}

	scriptCtx, ok := objects.ToInterface(args[0]).(*Context)
	if !ok {
		return nil, constructRuntimeError(1, ohlcvFunc, "*gct.Context", args[0])
	}
	exchangeName, ok := objects.ToString(args[1])
	if !ok {
		return nil, constructRuntimeError(2, ohlcvFunc, "string", args[1])
	}
	currencyPair, ok := objects.ToString(args[2])
	if !ok {
		return nil, constructRuntimeError(3, ohlcvFunc, "string", args[2])
	}
	delimiter, ok := objects.ToString(args[3])
	if !ok {
		return nil, constructRuntimeError(4, ohlcvFunc, "string", args[3])
	}
	assetTypeParam, ok := objects.ToString(args[4])
	if !ok {
		return nil, constructRuntimeError(5, ohlcvFunc, "string", args[4])
	}

	startTime, ok := objects.ToTime(args[5])
	if !ok {
		return nil, constructRuntimeError(6, ohlcvFunc, "string", args[5])
	}

	endTime, ok := objects.ToTime(args[6])
	if !ok {
		return nil, constructRuntimeError(7, ohlcvFunc, "time.Time", args[6])
	}

	intervalStr, ok := objects.ToString(args[7])
	if !ok {
		return nil, constructRuntimeError(8, ohlcvFunc, "string", args[7])
	}
	interval, err := parseInterval(intervalStr)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}
	pair, err := currency.NewPairDelimiter(currencyPair, delimiter)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}
	assetType, err := asset.New(assetTypeParam)
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	ctx := processScriptContext(scriptCtx)
	ret, err := wrappers.GetWrapper().
		OHLCV(ctx,
			exchangeName,
			pair,
			assetType,
			startTime,
			endTime,
			kline.Interval(interval))
	if err != nil {
		return errorResponsef(standardFormatting, err)
	}

	candles := objects.Array{Value: make([]objects.Object, len(ret.Candles))}
	for x := range ret.Candles {
		candles.Value[x] = &objects.Array{
			Value: []objects.Object{
				&objects.Int{Value: ret.Candles[x].Time.Unix()},
				&objects.Float{Value: ret.Candles[x].Open},
				&objects.Float{Value: ret.Candles[x].High},
				&objects.Float{Value: ret.Candles[x].Low},
				&objects.Float{Value: ret.Candles[x].Close},
				&objects.Float{Value: ret.Candles[x].Volume},
			},
		}
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
	if !common.StringSliceContainsInsensitive(supportedDurations, in) {
		return time.Nanosecond, kline.ErrInvalidInterval
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
