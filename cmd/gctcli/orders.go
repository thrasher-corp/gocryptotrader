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
	errSubmitOrderAmountRequired = errors.New("amount or quote amount must be set")
	errSubmitOrderAmountConflict = errors.New("amount and quote amount must not both be set")
)

// submitOrderRequest validates command flags and keeps account and client order identifiers distinct.
func submitOrderRequest(c *cli.Context) (*gctrpc.SubmitOrderRequest, error) {
	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return nil, errInvalidPair
	}

	orderSide := c.String(sideFlagName)
	if orderSide == "" {
		return nil, errSubmitOrderSideRequired
	}

	orderType := c.String("type")
	if orderType == "" {
		return nil, errSubmitOrderTypeRequired
	}

	amount, quoteAmount := c.Float64(amountFlagName), c.Float64(quoteAmountFlagName)
	if amount == 0 && quoteAmount == 0 {
		return nil, errSubmitOrderAmountRequired
	}
	// A server that predates quote amounts would ignore this one and submit the amount alone, a different quantity
	if amount != 0 && quoteAmount != 0 {
		return nil, errSubmitOrderAmountConflict
	}

	assetType := strings.ToLower(c.String(assetFlagName))
	if !validAsset(assetType) {
		return nil, errInvalidAsset
	}

	marginType := strings.ToLower(c.String(marginTypeFlagName))
	if marginType != "" && !margin.IsValidString(marginType) {
		return nil, margin.ErrInvalidMarginType
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return nil, err
	}

	return &gctrpc.SubmitOrderRequest{
		Exchange: c.String(exchangeFlagName),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		Side:          orderSide,
		OrderType:     orderType,
		Amount:        amount,
		QuoteAmount:   quoteAmount,
		Price:         c.Float64(priceFlagName),
		ClientId:      c.String("client_id"),
		ClientOrderId: c.String("client_order_id"),
		AssetType:     assetType,
		MarginType:    marginType,
	}, nil
}
