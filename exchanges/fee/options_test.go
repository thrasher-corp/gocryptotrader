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
		GlobalCommissions: map[asset.Item]Commission{
			asset.Spot: {Maker: 0, Taker: -1},
		},
	}).validate()
	if !errors.Is(err, errTakerInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errTakerInvalid)
	}

	err = (&Options{
		GlobalCommissions: map[asset.Item]Commission{
			asset.Spot: {Maker: 10, Taker: 1},
		},
	}).validate()
	if !errors.Is(err, errMakerBiggerThanTaker) {
		t.Fatalf("received: %v but expected: %v", err, errMakerBiggerThanTaker)
	}

	err = (&Options{
		ChainTransfer: []Transfer{
			{Currency: currency.BTC, Withdrawal: Convert(-1)},
		},
	}).validate()
	if !errors.Is(err, errDepositIsInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errDepositIsInvalid)
	}

	err = (&Options{
		ChainTransfer: []Transfer{
			{Currency: currency.BTC, Withdrawal: Convert(-1)},
		},
	}).validate()
	if !errors.Is(err, errWithdrawalIsInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errWithdrawalIsInvalid)
	}

	err = (&Options{
		BankTransfer: []Transfer{
			{BankTransfer: 255, Currency: currency.BTC, Deposit: Convert(-1)},
		},
	}).validate()
	if !errors.Is(err, bank.ErrUnknownTransfer) {
		t.Fatalf("received: %v but expected: %v", err, bank.ErrUnknownTransfer)
	}

	err = (&Options{
		BankTransfer: []Transfer{
			{BankTransfer: bank.WireTransfer, Currency: currency.BTC, Deposit: Convert(-1)},
		},
	}).validate()
	if !errors.Is(err, errDepositIsInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errDepositIsInvalid)
	}

	err = (&Options{
		BankTransfer: []Transfer{
			{BankTransfer: bank.WireTransfer, Currency: currency.BTC, Withdrawal: Convert(-1)},
		},
	}).validate()
	if !errors.Is(err, errWithdrawalIsInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errWithdrawalIsInvalid)
	}
}
