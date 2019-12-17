package gct

import (
	"fmt"
	"strings"

	"github.com/d5/tengo/objects"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/withdraw"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
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
	"withdrawfiat": &objects.UserFunction{Name: "withdrawfiat", Value: ExchangeWithdrawFiat},
}

// ExchangeOrderbook returns orderbook for requested exchange & currencypair
func ExchangeOrderbook(args ...objects.Object) (objects.Object, error) {
	if len(args) != 4 {
		return nil, objects.ErrWrongNumArguments
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

	data := make(map[string]objects.Object, 5)
	data["exchange"] = &objects.String{Value: ob.ExchangeName}
	data["pair"] = &objects.String{Value: ob.Pair.String()}
	data["asks"] = &asks
	data["bids"] = &bids
	data["asset"] = &objects.String{Value: ob.AssetType.String()}

	return &objects.Map{
		Value: data,
	}, nil
}

// ExchangeTicker returns ticker data for requested exchange and currency pair
func ExchangeTicker(args ...objects.Object) (objects.Object, error) {
	if len(args) != 4 {
		return nil, objects.ErrWrongNumArguments
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

	fmt.Println(tx)

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

	enabledOnly, _ := objects.ToBool(args[0])
	rtnValue := modules.Wrapper.Exchanges(enabledOnly)

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

	exchangeName, _ := objects.ToString(args[0])
	enabledOnly, _ := objects.ToBool(args[1])
	assetTypeParam, _ := objects.ToString(args[2])
	assetType := asset.Item(strings.ToLower(assetTypeParam))

	rtnValue, err := modules.Wrapper.Pairs(exchangeName, enabledOnly, assetType)
	if err != nil {
		return nil, err
	}

	r := objects.Array{}
	for x := range rtnValue.Slice() {
		r.Value = append(r.Value, &objects.String{Value: rtnValue.Slice()[x].String()})
	}
	return &r, nil
}

// ExchangeAccountInfo returns account information for requested exchange
func ExchangeAccountInfo(args ...objects.Object) (objects.Object, error) {
	if len(args) != 1 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, _ := objects.ToString(args[0])
	rtnValue, err := modules.Wrapper.AccountInformation(exchangeName)
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
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, _ := objects.ToString(args[0])
	orderID, _ := objects.ToString(args[1])

	orderDetails, err := modules.Wrapper.QueryOrder(exchangeName, orderID)
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
	data["currencypair"] = &objects.String{Value: orderDetails.CurrencyPair.String()}
	data["price"] = &objects.Float{Value: orderDetails.Price}
	data["amount"] = &objects.Float{Value: orderDetails.Amount}
	data["amountexecuted"] = &objects.Float{Value: orderDetails.ExecutedAmount}
	data["amountremaining"] = &objects.Float{Value: orderDetails.RemainingAmount}
	data["fee"] = &objects.Float{Value: orderDetails.Fee}
	data["side"] = &objects.String{Value: orderDetails.OrderSide.String()}
	data["type"] = &objects.String{Value: orderDetails.OrderType.String()}
	data["date"] = &objects.String{Value: orderDetails.OrderDate.String()}
	data["status"] = &objects.String{Value: orderDetails.Status.String()}
	data["trades"] = &tradeHistory

	return &objects.Map{
		Value: data,
	}, nil
}

// ExchangeOrderCancel cancels order on requested exchange
func ExchangeOrderCancel(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, _ := objects.ToString(args[0])
	orderID, _ := objects.ToString(args[1])

	rtn, err := modules.Wrapper.CancelOrder(exchangeName, orderID)
	if err != nil {
		return nil, err
	}

	if rtn {
		return objects.TrueValue, nil
	}
	return objects.FalseValue, nil
}

// ExchangeOrderSubmit submit order on exchange
func ExchangeOrderSubmit(args ...objects.Object) (objects.Object, error) {
	if len(args) != 8 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, _ := objects.ToString(args[0])
	currencyPair, _ := objects.ToString(args[1])
	delimiter, _ := objects.ToString(args[2])
	orderType, _ := objects.ToString(args[3])
	orderSide, _ := objects.ToString(args[4])
	orderPrice, _ := objects.ToFloat64(args[5])
	orderAmount, _ := objects.ToFloat64(args[6])
	orderClientID, _ := objects.ToString(args[7])

	pair := currency.NewPairDelimiter(currencyPair, delimiter)

	tempSubmit := &order.Submit{
		Pair:      pair,
		OrderType: order.Type(orderType),
		OrderSide: order.Side(orderSide),
		Price:     orderPrice,
		Amount:    orderAmount,
		ClientID:  orderClientID,
	}

	err := tempSubmit.Validate()
	if err != nil {
		return nil, err
	}

	rtn, err := modules.Wrapper.SubmitOrder(exchangeName, tempSubmit)
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
	if len(args) != 3 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, _ := objects.ToString(args[0])
	currencyCode, _ := objects.ToString(args[1])
	accountID, _ := objects.ToString(args[2])
	currCode := currency.NewCode(currencyCode)

	rtn, err := modules.Wrapper.DepositAddress(exchangeName, currCode, accountID)
	if err != nil {
		return nil, err
	}

	return &objects.String{Value: rtn}, nil
}

func ExchangeWithdrawCrypto(args ...objects.Object) (objects.Object, error) {
	if len(args) != 8 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, _ := objects.ToString(args[0])
	cur,_ := objects.ToString(args[1])
	address, _ := objects.ToString(args[2])
	addressTag, _ := objects.ToString(args[3])
	amount, _ := objects.ToFloat64(args[4])
	feeAmount, _ := objects.ToFloat64(args[5])
	tradePassword, _ := objects.ToString(args[6])
	onetimePassword, _ := objects.ToInt64(args[7])

	withdrawRequest := &withdraw.CryptoRequest{
		GenericInfo: withdraw.GenericInfo{
			Currency:        currency.NewCode(cur),
			Description:     "",
			OneTimePassword: onetimePassword,
			AccountID:       "",
			PIN:             0,
			TradePassword:   tradePassword,
			Amount:          amount,
		},
		Address:     address,
		AddressTag:  addressTag,
		FeeAmount:   feeAmount,
	}
	rtn, err := modules.Wrapper.WithdrawalCryptoFunds(exchangeName, withdrawRequest)
	if err != nil {
		return nil, err
	}

	return &objects.String{Value: rtn}, nil
}

func ExchangeWithdrawFiat(args ...objects.Object) (objects.Object, error) {
	if len(args) != 20 {
		return nil, objects.ErrWrongNumArguments
	}

	exchangeName, _ := objects.ToString(args[0])
	cur,_ := objects.ToString(args[1])
	description, _ := objects.ToString(args[2])
	bankAccountName, _ := objects.ToString(args[3])
	bankAccountNumber, _ := objects.ToString(args[4])
	bankName, _ := objects.ToString(args[5])
	bankAddress, _ := objects.ToString(args[6])
	bankCity, _ := objects.ToString(args[7])
	bankCountry, _ := objects.ToString(args[8])
	bankPostalCode, _ := objects.ToString(args[9])
	BSB, _ := objects.ToString(args[10])
	swiftCode, _ := objects.ToString(args[11])
	IBAN, _ := objects.ToString(args[12])
	bankCode, _ := objects.ToFloat64(args[13])
	isExpressWire, _ := objects.ToBool(args[14])
	amount, _ := objects.ToFloat64(args[15])
	pin,_ := objects.ToInt64(args[16])
	tradePassword, _ := objects.ToString(args[17])
	onetimePassword, _ := objects.ToInt64(args[18])
	
	withdrawRequest := &withdraw.FiatRequest{
		GenericInfo: withdraw.GenericInfo{
			Currency:        currency.NewCode(cur),
			Description:     description,
			OneTimePassword: onetimePassword,
			AccountID:       "",
			PIN:             pin,
			TradePassword:   tradePassword,
			Amount:          amount,
		},
		BankAccountName:               bankAccountName,
		BankAccountNumber:             bankAccountNumber,
		BankName:                      bankName,
		BankAddress:                   bankAddress,
		BankCity:                      bankCity,
		BankCountry:                   bankCountry,
		BankPostalCode:                bankPostalCode,
		BSB:                           BSB,
		SwiftCode:                     swiftCode,
		IBAN:                          IBAN,
		BankCode:                      bankCode,
		IsExpressWire:                 isExpressWire,
		RequiresIntermediaryBank:      false,
		IntermediaryBankAccountNumber: 0,
		IntermediaryBankName:          "",
		IntermediaryBankAddress:       "",
		IntermediaryBankCity:          "",
		IntermediaryBankCountry:       "",
		IntermediaryBankPostalCode:    "",
		IntermediarySwiftCode:         "",
		IntermediaryBankCode:          0,
		IntermediaryIBAN:              "",
		WireCurrency:                  "",
	}

	rtn, err := modules.Wrapper.WithdrawalFiatFunds(exchangeName, withdrawRequest)
	if err != nil {
		return nil, err
	}

	return &objects.String{Value: rtn}, nil
}
