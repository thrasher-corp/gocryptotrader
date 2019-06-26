package exchange

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/currency"
)

func TestValidate(t *testing.T) {
	testPair := currency.NewPair(currency.BTC, currency.LTC)
	tester := []struct {
		Pair        currency.Pair
		Side        OrderSide
		Type        OrderType
		Amount      float64
		Price       float64
		ExpectedErr error
	}{
		{
			ExpectedErr: ErrOrderPairIsEmpty,
		}, // empty pair
		{
			Pair:        testPair,
			ExpectedErr: ErrOrderSideIsInvalid,
		}, // valid pair but invalid order side
		{
			Pair:        testPair,
			Side:        BuyOrderSide,
			ExpectedErr: ErrOrderTypeIsInvalid,
		}, // valid pair and order side but invalid order type
		{
			Pair:        testPair,
			Side:        SellOrderSide,
			ExpectedErr: ErrOrderTypeIsInvalid,
		}, // valid pair and order side but invalid order type
		{
			Pair:        testPair,
			Side:        BidOrderSide,
			ExpectedErr: ErrOrderTypeIsInvalid,
		}, // valid pair and order side but invalid order type
		{
			Pair:        testPair,
			Side:        AskOrderSide,
			ExpectedErr: ErrOrderTypeIsInvalid,
		}, // valid pair and order side but invalid order type
		{
			Pair:        testPair,
			Side:        AskOrderSide,
			Type:        MarketOrderType,
			ExpectedErr: ErrOrderAmountIsInvalid,
		}, // valid pair, order side, type but invalid amount
		{
			Pair:        testPair,
			Side:        AskOrderSide,
			Type:        LimitOrderType,
			Amount:      1,
			ExpectedErr: ErrOrderPriceMustBeSetIfLimitOrder,
		}, // valid pair, order side, type, amount but invalid price
		{
			Pair:        testPair,
			Side:        AskOrderSide,
			Type:        LimitOrderType,
			Amount:      1,
			Price:       1000,
			ExpectedErr: nil,
		}, // valid order!
	}

	for x := range tester {
		s := OrderSubmission{
			Pair:      tester[x].Pair,
			OrderSide: tester[x].Side,
			OrderType: tester[x].Type,
			Amount:    tester[x].Amount,
			Price:     tester[x].Price,
		}
		if err := s.Validate(); err != tester[x].ExpectedErr {
			t.Errorf("Unexpected result. Got: %s, want: %s", err, tester[x].ExpectedErr)
		}
	}
}
