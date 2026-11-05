package main

import (
	"errors"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

var (
	errSubmitOrderSideRequired   = errors.New("order side must be set")
	errSubmitOrderTypeRequired   = errors.New("order type must be set")
	errSubmitOrderAmountRequired = errors.New("amount must be set")
)

// submitOrderRequest validates command flags and keeps account and client order identifiers distinct.
func submitOrderRequest(c *cli.Context) (*gctrpc.SubmitOrderRequest, error) {
	var exchangeName string
	var currencyPair string
	var orderSide string
	var orderType string
	var amount float64
	var price float64
	var clientID string
	var assetType string
	var marginType string

	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	if c.IsSet(pairFlag) {
		currencyPair = c.String(pairFlag)
	}

	if !validPair(currencyPair) {
		return nil, errInvalidPair
	}

	if c.IsSet(sideFlag) {
		orderSide = c.String(sideFlag)
	}

	if orderSide == "" {
		return nil, errSubmitOrderSideRequired
	}

	if c.IsSet("type") {
		orderType = c.String("type")
	}

	if orderType == "" {
		return nil, errSubmitOrderTypeRequired
	}

	if c.IsSet(amountFlag) {
		amount = c.Float64(amountFlag)
	}

	if amount == 0 {
		return nil, errSubmitOrderAmountRequired
	}

	// price is optional for market orders
	if c.IsSet("price") {
		price = c.Float64("price")
	}

	if c.IsSet("client_id") {
		clientID = c.String("client_id")
	}

	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	if c.IsSet("margintype") {
		marginType = c.String("margintype")
	}

	assetType = strings.ToLower(assetType)
	if !validAsset(assetType) {
		return nil, errInvalidAsset
	}

	marginType = strings.ToLower(marginType)
	if marginType != "" && !margin.IsValidString(marginType) {
		return nil, margin.ErrInvalidMarginType
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return nil, err
	}

	return &gctrpc.SubmitOrderRequest{
		Exchange: exchangeName,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		Side:          orderSide,
		OrderType:     orderType,
		Amount:        amount,
		Price:         price,
		ClientId:      clientID,
		ClientOrderId: c.String("client_order_id"),
		AssetType:     assetType,
		MarginType:    marginType,
	}, nil
}
