package fee

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bank"
)

func TestOptionsValidate(t *testing.T) {
	err := (&Options{
		GlobalCommissions: map[asset.Item]Commission{
			asset.Spot: {Maker: -1},
		},
	}).validate()
	if !errors.Is(err, errMakerInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errMakerInvalid)
	}

	err = (&Options{
		PairCommissions: map[asset.Item]map[currency.Pair]Commission{
			asset.Spot: {
				currency.NewPair(currency.BTC, currency.USDT): {Maker: 1},
			},
		},
	}).validate()
	if !errors.Is(err, errMakerBiggerThanTaker) {
		t.Fatalf("received: %v but expected: %v", err, errMakerBiggerThanTaker)
	}

	err = (&Options{
		ChainTransfer: []Transfer{
			{},
		},
	}).validate()
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCurrencyIsEmpty)
	}

	err = (&Options{
		BankTransfer: []Transfer{
			{},
		},
	}).validate()
	if !errors.Is(err, bank.ErrUnknownTransfer) {
		t.Fatalf("received: %v but expected: %v", err, bank.ErrUnknownTransfer)
	}

	err = (&Options{
		BankTransfer: []Transfer{
			{BankTransfer: bank.WireTransfer},
		},
	}).validate()
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCurrencyIsEmpty)
	}
}
