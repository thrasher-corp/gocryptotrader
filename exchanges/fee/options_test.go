package fee

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestOptionsValidate(t *testing.T) {
	err := (&Options{
		Commission: map[asset.Item]Commission{
			asset.Spot: {Maker: -1},
		},
	}).validate()
	if !errors.Is(err, errMakerInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errMakerInvalid)
	}

	err = (&Options{
		Commission: map[asset.Item]Commission{
			asset.Spot: {Maker: 0, Taker: -1},
		},
	}).validate()
	if !errors.Is(err, errTakerInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errTakerInvalid)
	}

	err = (&Options{
		Commission: map[asset.Item]Commission{
			asset.Spot: {Maker: 10, Taker: 1},
		},
	}).validate()
	if !errors.Is(err, errMakerBiggerThanTaker) {
		t.Fatalf("received: %v but expected: %v", err, errMakerBiggerThanTaker)
	}

	err = (&Options{
		Transfer: map[asset.Item]map[currency.Code]Transfer{
			asset.Spot: {currency.BTC: {Deposit: -1}},
		},
	}).validate()
	if !errors.Is(err, errDepositIsInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errDepositIsInvalid)
	}

	err = (&Options{
		Transfer: map[asset.Item]map[currency.Code]Transfer{
			asset.Spot: {currency.BTC: {Withdrawal: -1}},
		},
	}).validate()
	if !errors.Is(err, errWithdrawalIsInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errWithdrawalIsInvalid)
	}

	err = (&Options{
		BankingTransfer: map[BankTransaction]map[currency.Code]Transfer{
			255: {currency.BTC: {Deposit: -1}},
		},
	}).validate()
	if !errors.Is(err, errUnknownBankTransaction) {
		t.Fatalf("received: %v but expected: %v", err, errUnknownBankTransaction)
	}

	err = (&Options{
		BankingTransfer: map[BankTransaction]map[currency.Code]Transfer{
			WireTransfer: {currency.BTC: {Deposit: -1}},
		},
	}).validate()
	if !errors.Is(err, errDepositIsInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errDepositIsInvalid)
	}

	err = (&Options{
		BankingTransfer: map[BankTransaction]map[currency.Code]Transfer{
			WireTransfer: {currency.BTC: {Withdrawal: -1}},
		},
	}).validate()
	if !errors.Is(err, errWithdrawalIsInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errWithdrawalIsInvalid)
	}
}
