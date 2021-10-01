package fee

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var one = decimal.NewFromInt(1)
var two = decimal.NewFromInt(2)

func TestLoadDynamic(t *testing.T) {
	t.Parallel()
	err := (*Definitions)(nil).LoadDynamic(0, 0, asset.Spot, OmitPair)
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	err = (&Definitions{}).LoadDynamic(-1, 0, asset.Spot, OmitPair)
	if !errors.Is(err, errMakerInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errMakerInvalid)
	}

	err = (&Definitions{}).LoadDynamic(0, -1, asset.Spot, OmitPair)
	if !errors.Is(err, errTakerInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errTakerInvalid)
	}

	err = (&Definitions{}).LoadDynamic(30, 12, asset.Spot, OmitPair)
	if !errors.Is(err, errMakerBiggerThanTaker) {
		t.Fatalf("received: %v but expected: %v", err, errMakerBiggerThanTaker)
	}

	err = (&Definitions{}).LoadDynamic(1, 1, "bruh", OmitPair)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v but expected: %v", err, asset.ErrNotSupported)
	}

	err = (&Definitions{}).LoadDynamic(1, 1, asset.Spot, OmitPair)
	if !errors.Is(err, errCommissionRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errCommissionRateNotFound)
	}

	d := &Definitions{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {},
		},
	}
	err = d.LoadDynamic(1, 1, asset.Spot, OmitPair)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestLoadStatic(t *testing.T) {
	err := (*Definitions)(nil).LoadStatic(Options{})
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	d := &Definitions{
		globalCommissions: make(map[asset.Item]*CommissionInternal),
		transfers:         make(map[asset.Item]map[*currency.Item]*transfer),
		bankingTransfers:  make(map[BankTransaction]map[*currency.Item]*transfer),
	}
	err = d.LoadStatic(Options{
		GlobalCommissions: map[asset.Item]Commission{
			asset.Spot: {Maker: -1},
		},
	}) // Validate coverage
	if !errors.Is(err, errMakerInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errMakerInvalid)
	}

	err = d.LoadStatic(Options{
		GlobalCommissions: map[asset.Item]Commission{
			asset.Spot: {},
		},
		Transfer: map[asset.Item]map[currency.Code]Transfer{
			asset.Spot: {currency.BTC: {}},
		},
		BankingTransfer: map[BankTransaction]map[currency.Code]Transfer{
			WireTransfer: {currency.BTC: {}},
		},
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestCalculateMaker(t *testing.T) {
	t.Parallel()

	_, err := (*Definitions)(nil).CalculateMaker(50000, 1, asset.Spot, OmitPair)
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	d := &Definitions{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {maker: decimal.NewFromFloat(0.01)},
		},
	}

	_, err = d.CalculateMaker(50000, 1, asset.Futures, OmitPair)
	if !errors.Is(err, errRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errRateNotFound)
	}

	val, err := d.CalculateMaker(50000, 1, asset.Spot, OmitPair)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if val != 500 {
		t.Fatalf("received: %v but expected %v", val, 500)
	}
}

func TestCalculateWorstCaseMaker(t *testing.T) {
	t.Parallel()

	_, err := (*Definitions)(nil).CalculateWorstCaseMaker(50000, 1, asset.Spot, OmitPair)
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	d := &Definitions{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {worstCaseMaker: decimal.NewFromFloat(0.01)},
		},
	}

	_, err = d.CalculateWorstCaseMaker(50000, 1, asset.Futures, OmitPair)
	if !errors.Is(err, errRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errRateNotFound)
	}

	val, err := d.CalculateWorstCaseMaker(50000, 1, asset.Spot, OmitPair)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if val != 500 {
		t.Fatalf("received: %v but expected %v", val, 500)
	}
}

func TestGetMaker(t *testing.T) {
	_, _, err := (*Definitions)(nil).GetMaker(asset.Spot, OmitPair)
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	d := &Definitions{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {maker: decimal.NewFromFloat(0.01)},
		},
	}

	_, _, err = d.GetMaker(asset.Futures, OmitPair)
	if !errors.Is(err, errRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errRateNotFound)
	}

	fee, isSetAmount, err := d.GetMaker(asset.Spot, OmitPair)
	if err != nil {
		t.Fatal(err)
	}
	if isSetAmount {
		t.Fatal("unexpected, should be percentage")
	}
	if fee != 0.01 {
		t.Fatal("unexpected maker value")
	}
}

func TestCalculateTaker(t *testing.T) {
	t.Parallel()

	_, err := (*Definitions)(nil).CalculateTaker(50000, 1, asset.Spot, OmitPair)
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	d := &Definitions{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {taker: decimal.NewFromFloat(0.01)},
		},
	}

	_, err = d.CalculateTaker(50000, 1, asset.Futures, OmitPair)
	if !errors.Is(err, errRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errRateNotFound)
	}

	val, err := d.CalculateTaker(50000, 1, asset.Spot, OmitPair)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if val != 500 {
		t.Fatalf("received: %v but expected %v", val, 500)
	}
}

func TestCalculateWorstCaseTaker(t *testing.T) {
	t.Parallel()

	_, err := (*Definitions)(nil).CalculateWorstCaseTaker(50000, 1, asset.Spot, OmitPair)
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	d := &Definitions{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {worstCaseTaker: decimal.NewFromFloat(0.01)},
		},
	}

	_, err = d.CalculateWorstCaseTaker(50000, 1, asset.Futures, OmitPair)
	if !errors.Is(err, errRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errRateNotFound)
	}

	val, err := d.CalculateWorstCaseTaker(50000, 1, asset.Spot, OmitPair)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if val != 500 {
		t.Fatalf("received: %v but expected %v", val, 500)
	}
}

func TestGetTaker(t *testing.T) {
	t.Parallel()
	_, _, err := (*Definitions)(nil).GetTaker(asset.Spot, OmitPair)
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	d := &Definitions{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {taker: decimal.NewFromFloat(0.01)},
		},
	}

	_, _, err = d.GetTaker(asset.Futures, OmitPair)
	if !errors.Is(err, errRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errRateNotFound)
	}

	fee, isSetAmount, err := d.GetTaker(asset.Spot, OmitPair)
	if err != nil {
		t.Fatal(err)
	}
	if isSetAmount {
		t.Fatal("unexpected, should be a percentage")
	}
	if fee != 0.01 {
		t.Fatal("unexpected taker value")
	}
}

func TestCalculateDeposit(t *testing.T) {
	_, err := (*Definitions)(nil).CalculateDeposit(currency.Code{}, "", 0)
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	_, err = (&Definitions{}).CalculateDeposit(currency.BTC, asset.Spot, 0)
	if !errors.Is(err, errTransferFeeNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errTransferFeeNotFound)
	}

	d := &Definitions{
		transfers: map[asset.Item]map[*currency.Item]*transfer{
			asset.Spot: {
				currency.BTC.Item: {Deposit: Convert(0.01)},
			},
		},
	}

	fee, err := d.CalculateDeposit(currency.BTC, asset.Spot, 10)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if fee != 0.01 {
		t.Fatal("unexpected fee value")
	}
}

func TestGetDeposit(t *testing.T) {
	_, _, err := (*Definitions)(nil).GetDeposit(currency.Code{}, "")
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	_, _, err = (&Definitions{}).GetDeposit(currency.BTC, asset.Spot)
	if !errors.Is(err, errTransferFeeNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errTransferFeeNotFound)
	}

	d := &Definitions{
		transfers: map[asset.Item]map[*currency.Item]*transfer{
			asset.Spot: {
				currency.BTC.Item: {Deposit: Convert(0.01)},
			},
		},
	}

	fee, percentage, err := d.GetDeposit(currency.BTC, asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if percentage {
		t.Fatal("unexpected percentage value")
	}

	if !fee.(Standard).Decimal.Equal(decimal.NewFromFloat(0.01)) {
		t.Fatal("unexpected fee value")
	}
}

func TestCalculateWithdrawal(t *testing.T) {
	_, err := (*Definitions)(nil).CalculateWithdrawal(currency.Code{}, "", 0)
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	_, err = (&Definitions{}).CalculateWithdrawal(currency.BTC, asset.Spot, 0)
	if !errors.Is(err, errTransferFeeNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errTransferFeeNotFound)
	}

	d := &Definitions{
		transfers: map[asset.Item]map[*currency.Item]*transfer{
			asset.Spot: {
				currency.BTC.Item: {Withdrawal: Convert(0.01)},
			},
		},
	}

	fee, err := d.CalculateWithdrawal(currency.BTC, asset.Spot, 10)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if fee != 0.01 {
		t.Fatal("unexpected fee value")
	}
}

func TestGetWithdrawal(t *testing.T) {
	_, _, err := (&Definitions{}).GetWithdrawal(currency.Code{}, "")
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCurrencyIsEmpty)
	}

	_, _, err = (&Definitions{}).GetWithdrawal(currency.BTC, "")
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v but expected: %v", err, asset.ErrNotSupported)
	}

	_, _, err = (&Definitions{}).GetWithdrawal(currency.BTC, asset.Spot)
	if !errors.Is(err, errTransferFeeNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errTransferFeeNotFound)
	}

	d := &Definitions{
		transfers: map[asset.Item]map[*currency.Item]*transfer{
			asset.Spot: {
				currency.BTC.Item: {Withdrawal: Convert(0.01)},
			},
		},
	}

	fee, percentage, err := d.GetWithdrawal(currency.BTC, asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if percentage {
		t.Fatal("unexpected percentage value")
	}

	if !fee.(Standard).Decimal.Equal(decimal.NewFromFloat(0.01)) {
		t.Fatal("unexpected fee value")
	}
}

func TestGetAllFees(t *testing.T) {
	_, err := (*Definitions)(nil).GetAllFees()
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	d := Definitions{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {},
		},
		transfers: map[asset.Item]map[*currency.Item]*transfer{
			asset.Spot: {currency.BTC.Item: {}},
		},
		bankingTransfers: map[BankTransaction]map[*currency.Item]*transfer{
			WireTransfer: {currency.BTC.Item: {}},
		},
	}
	_, err = d.GetAllFees()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestGetCommissionFee(t *testing.T) {
	_, err := (*Definitions)(nil).GetCommissionFee(asset.Spot, OmitPair)
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	_, err = (&Definitions{}).GetCommissionFee(asset.Spot, OmitPair)
	if !errors.Is(err, errRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errRateNotFound)
	}

	_, err = (&Definitions{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {},
		},
	}).GetCommissionFee(asset.Spot, OmitPair)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestSetCommissionFee(t *testing.T) {
	err := (*Definitions)(nil).SetCommissionFee("", OmitPair, 0, 0, true)
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	err = (&Definitions{}).SetCommissionFee("", OmitPair, -1, 0, true)
	if !errors.Is(err, errMakerInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errMakerInvalid)
	}

	err = (&Definitions{}).SetCommissionFee("", OmitPair, 0, -1, true)
	if !errors.Is(err, errTakerInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errTakerInvalid)
	}

	err = (&Definitions{}).SetCommissionFee("", OmitPair, 0, 0, true)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v but expected: %v", err, asset.ErrNotSupported)
	}

	err = (&Definitions{}).SetCommissionFee(asset.Spot, OmitPair, 0, 0, true)
	if !errors.Is(err, errRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errRateNotFound)
	}

	err = (&Definitions{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {},
		},
	}).SetCommissionFee(asset.Spot, OmitPair, 0, 0, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestGetTransferFee(t *testing.T) {
	_, err := (*Definitions)(nil).GetTransferFee(currency.Code{}, "")
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	_, err = (&Definitions{}).GetTransferFee(currency.Code{}, "")
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCurrencyIsEmpty)
	}

	_, err = (&Definitions{}).GetTransferFee(currency.BTC, "")
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v but expected: %v", err, asset.ErrNotSupported)
	}

	_, err = (&Definitions{}).GetTransferFee(currency.BTC, asset.Spot)
	if !errors.Is(err, errRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errRateNotFound)
	}

	_, err = (&Definitions{
		transfers: map[asset.Item]map[*currency.Item]*transfer{
			asset.Spot: {currency.BTC.Item: {}},
		},
	}).GetTransferFee(currency.BTC, asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestSetTransferFee(t *testing.T) {
	err := (*Definitions)(nil).SetTransferFee(currency.Code{}, "", 0, 0, true)
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	err = (&Definitions{}).SetTransferFee(currency.Code{}, "", -1, 0, true)
	if !errors.Is(err, errWithdrawalIsInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errWithdrawalIsInvalid)
	}

	err = (&Definitions{}).SetTransferFee(currency.Code{}, "", 0, -1, true)
	if !errors.Is(err, errDepositIsInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errDepositIsInvalid)
	}

	err = (&Definitions{}).SetTransferFee(currency.Code{}, "", 0, 0, true)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v but expected: %v", err, asset.ErrNotSupported)
	}

	err = (&Definitions{}).SetTransferFee(currency.Code{}, asset.Spot, 0, 0, true)
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCurrencyIsEmpty)
	}

	err = (&Definitions{}).SetTransferFee(currency.BTC, asset.Spot, 0, 0, true)
	if !errors.Is(err, errTransferFeeNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errTransferFeeNotFound)
	}

	err = (&Definitions{
		transfers: map[asset.Item]map[*currency.Item]*transfer{
			asset.Spot: {currency.BTC.Item: {}},
		},
	}).SetTransferFee(currency.BTC, asset.Spot, 0, 0, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	err = (&Definitions{
		transfers: map[asset.Item]map[*currency.Item]*transfer{
			asset.Spot: {currency.BTC.Item: {}},
		},
	}).SetTransferFee(currency.BTC, asset.Spot, 0, 0, true)
	if !errors.Is(err, errFeeTypeMismatch) {
		t.Fatalf("received: %v but expected: %v", err, errFeeTypeMismatch)
	}
}

func TestGetBankTransferFee(t *testing.T) {
	_, err := (*Definitions)(nil).GetBankTransferFee(currency.Code{}, 255)
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	_, err = (&Definitions{}).GetBankTransferFee(currency.Code{}, 255)
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCurrencyIsEmpty)
	}

	_, err = (&Definitions{}).GetBankTransferFee(currency.USD, 255)
	if !errors.Is(err, errUnknownBankTransaction) {
		t.Fatalf("received: %v but expected: %v", err, errUnknownBankTransaction)
	}

	_, err = (&Definitions{}).GetBankTransferFee(currency.USD, WireTransfer)
	if !errors.Is(err, errRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errRateNotFound)
	}

	_, err = (&Definitions{
		bankingTransfers: map[BankTransaction]map[*currency.Item]*transfer{
			WireTransfer: {currency.USD.Item: {}},
		},
	}).GetBankTransferFee(currency.USD, WireTransfer)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestSetBankTransferFee(t *testing.T) {
	err := (*Definitions)(nil).SetBankTransferFee(currency.Code{}, 255, -1, -1, true)
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	err = (&Definitions{}).SetBankTransferFee(currency.Code{}, 255, -1, -1, true)
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCurrencyIsEmpty)
	}

	err = (&Definitions{}).SetBankTransferFee(currency.USD, 255, -1, -1, true)
	if !errors.Is(err, errUnknownBankTransaction) {
		t.Fatalf("received: %v but expected: %v", err, errUnknownBankTransaction)
	}

	err = (&Definitions{}).SetBankTransferFee(currency.USD, WireTransfer, -1, -1, true)
	if !errors.Is(err, errWithdrawalIsInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errWithdrawalIsInvalid)
	}

	err = (&Definitions{}).SetBankTransferFee(currency.USD, WireTransfer, 0, -1, true)
	if !errors.Is(err, errDepositIsInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errDepositIsInvalid)
	}

	err = (&Definitions{}).SetBankTransferFee(currency.USD, WireTransfer, 0, 0, true)
	if !errors.Is(err, errBankTransferFeeNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errBankTransferFeeNotFound)
	}

	err = (&Definitions{
		bankingTransfers: map[BankTransaction]map[*currency.Item]*transfer{
			WireTransfer: {currency.USD.Item: {}},
		},
	}).SetBankTransferFee(currency.USD, WireTransfer, 0, 0, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	err = (&Definitions{
		bankingTransfers: map[BankTransaction]map[*currency.Item]*transfer{
			WireTransfer: {currency.USD.Item: {}},
		},
	}).SetBankTransferFee(currency.USD, WireTransfer, 0, 0, true)
	if !errors.Is(err, errFeeTypeMismatch) {
		t.Fatalf("received: %v but expected: %v", err, errFeeTypeMismatch)
	}
}
