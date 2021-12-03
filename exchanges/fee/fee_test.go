package fee

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bank"
)

var one = decimal.NewFromInt(1)
var two = decimal.NewFromInt(2)

func TestNewFeeSchedule(t *testing.T) {
	t.Parallel()
	if NewFeeSchedule() == nil {
		t.Fatal("unexpected value")
	}
}

func TestLoadDynamicFeeRate(t *testing.T) {
	t.Parallel()
	err := (*Schedule)(nil).LoadDynamicFeeRate(0, 0, asset.Spot, OmitPair)
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	err = (&Schedule{}).LoadDynamicFeeRate(30, 12, asset.Spot, OmitPair)
	if !errors.Is(err, errMakerBiggerThanTaker) {
		t.Fatalf("received: %v but expected: %v", err, errMakerBiggerThanTaker)
	}

	err = (&Schedule{}).LoadDynamicFeeRate(10, 10, asset.Spot, OmitPair)
	if !errors.Is(err, errMakerInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errMakerInvalid)
	}

	err = (&Schedule{}).LoadDynamicFeeRate(0, 10, asset.Spot, OmitPair)
	if !errors.Is(err, errTakerInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errTakerInvalid)
	}

	err = (&Schedule{}).LoadDynamicFeeRate(.002, .002, "bruh", OmitPair)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v but expected: %v", err, asset.ErrNotSupported)
	}

	err = (&Schedule{}).LoadDynamicFeeRate(.002, .002, asset.Spot, OmitPair)
	if !errors.Is(err, errCommissionRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errCommissionRateNotFound)
	}

	d := &Schedule{
		pairCommissions: make(map[asset.Item]map[*currency.Item]map[*currency.Item]*CommissionInternal),
	}
	err = d.LoadDynamicFeeRate(.002, .002, asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestLoadStaticFees(t *testing.T) {
	t.Parallel()
	err := (*Schedule)(nil).LoadStaticFees(Options{})
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	d := &Schedule{
		globalCommissions: make(map[asset.Item]*CommissionInternal),
		pairCommissions:   make(map[asset.Item]map[*currency.Item]map[*currency.Item]*CommissionInternal),
		chainTransfer:     make(map[*currency.Item]map[string]*transfer),
		bankTransfer:      make(map[bank.Transfer]map[*currency.Item]*transfer),
	}
	err = d.LoadStaticFees(Options{
		GlobalCommissions: map[asset.Item]Commission{
			asset.Spot: {Maker: -1},
		},
	}) // Validate coverage
	if !errors.Is(err, errMakerInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errMakerInvalid)
	}

	err = d.LoadStaticFees(Options{
		GlobalCommissions: map[asset.Item]Commission{
			asset.Spot: {},
		},
		PairCommissions: map[asset.Item]map[currency.Pair]Commission{
			asset.Spot: {
				currency.NewPair(currency.BTC, currency.USDT): {},
			},
		},
		ChainTransfer: []Transfer{
			{Currency: currency.BTC},
		},
		BankTransfer: []Transfer{
			{BankTransfer: bank.WireTransfer, Currency: currency.BTC},
		},
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestGetcommission(t *testing.T) {
	t.Parallel()
	d := &Schedule{
		pairCommissions: map[asset.Item]map[*currency.Item]map[*currency.Item]*CommissionInternal{
			asset.Spot: {
				currency.BTC.Item: {
					currency.USDT.Item: &CommissionInternal{
						maker: decimal.NewFromFloat(0.002),
					},
				},
			},
		},
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {maker: decimal.NewFromFloat(0.002)},
		},
	}

	_, err := d.getCommission(asset.Spot, currency.NewPair(currency.BTC, currency.POOLZ))
	if !errors.Is(err, errCommissionRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errCommissionRateNotFound)
	}

	_, err = d.getCommission(asset.Spot, OmitPair)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	com, err := d.getCommission(asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Fatal(err)
	}

	if !com.maker.Equal(decimal.NewFromFloat(0.002)) {
		t.Fatal("unexpected result", com.maker)
	}
}

func TestCalculateMaker(t *testing.T) {
	t.Parallel()

	_, err := (*Schedule)(nil).CalculateMaker(50000, 1, asset.Spot, OmitPair)
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	d := &Schedule{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {maker: decimal.NewFromFloat(0.01)},
		},
	}

	_, err = d.CalculateMaker(50000, 1, asset.Futures, OmitPair)
	if !errors.Is(err, errCommissionRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errCommissionRateNotFound)
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

	_, err := (*Schedule)(nil).CalculateWorstCaseMaker(50000, 1, asset.Spot, OmitPair)
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	d := &Schedule{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {worstCaseMaker: decimal.NewFromFloat(0.01)},
		},
	}

	_, err = d.CalculateWorstCaseMaker(50000, 1, asset.Futures, OmitPair)
	if !errors.Is(err, errCommissionRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errCommissionRateNotFound)
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
	t.Parallel()
	_, _, err := (*Schedule)(nil).GetMaker(asset.Spot, OmitPair)
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	d := &Schedule{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {maker: decimal.NewFromFloat(0.01)},
		},
	}

	_, _, err = d.GetMaker(asset.Futures, OmitPair)
	if !errors.Is(err, errCommissionRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errCommissionRateNotFound)
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

	_, err := (*Schedule)(nil).CalculateTaker(50000, 1, asset.Spot, OmitPair)
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	d := &Schedule{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {taker: decimal.NewFromFloat(0.01)},
		},
	}

	_, err = d.CalculateTaker(50000, 1, asset.Futures, OmitPair)
	if !errors.Is(err, errCommissionRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errCommissionRateNotFound)
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

	_, err := (*Schedule)(nil).CalculateWorstCaseTaker(50000, 1, asset.Spot, OmitPair)
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	d := &Schedule{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {worstCaseTaker: decimal.NewFromFloat(0.01)},
		},
	}

	_, err = d.CalculateWorstCaseTaker(50000, 1, asset.Futures, OmitPair)
	if !errors.Is(err, errCommissionRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errCommissionRateNotFound)
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
	_, _, err := (*Schedule)(nil).GetTaker(asset.Spot, OmitPair)
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	d := &Schedule{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {taker: decimal.NewFromFloat(0.01)},
		},
	}

	_, _, err = d.GetTaker(asset.Futures, OmitPair)
	if !errors.Is(err, errCommissionRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errCommissionRateNotFound)
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
	t.Parallel()
	_, err := (*Schedule)(nil).CalculateDeposit(currency.Code{}, "", 0)
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	_, err = (&Schedule{}).CalculateDeposit(currency.BTC, "", 0)
	if !errors.Is(err, errTransferFeeNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errTransferFeeNotFound)
	}

	d := &Schedule{
		chainTransfer: map[*currency.Item]map[string]*transfer{
			currency.BTC.Item: {
				"": {Deposit: Convert(0.01)},
			},
		},
	}

	fee, err := d.CalculateDeposit(currency.BTC, "", 10)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if fee != 0.01 {
		t.Fatal("unexpected fee value")
	}
}

func TestGetDeposit(t *testing.T) {
	t.Parallel()
	_, _, err := (*Schedule)(nil).GetDeposit(currency.Code{}, "")
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	_, _, err = (&Schedule{}).GetDeposit(currency.BTC, "")
	if !errors.Is(err, errTransferFeeNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errTransferFeeNotFound)
	}

	d := &Schedule{
		chainTransfer: map[*currency.Item]map[string]*transfer{
			currency.BTC.Item: {
				"": {Deposit: Convert(0.01)},
			},
		},
	}

	fee, percentage, err := d.GetDeposit(currency.BTC, "")
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
	t.Parallel()
	_, err := (*Schedule)(nil).CalculateWithdrawal(currency.Code{}, "", 0)
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	_, err = (&Schedule{}).CalculateWithdrawal(currency.BTC, "", 0)
	if !errors.Is(err, errTransferFeeNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errTransferFeeNotFound)
	}

	d := &Schedule{
		chainTransfer: map[*currency.Item]map[string]*transfer{
			currency.BTC.Item: {
				"": {Withdrawal: Convert(0.01)},
			},
		},
	}

	fee, err := d.CalculateWithdrawal(currency.BTC, "", 10)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if fee != 0.01 {
		t.Fatal("unexpected fee value")
	}
}

func TestGetWithdrawal(t *testing.T) {
	t.Parallel()
	_, _, err := (&Schedule{}).GetWithdrawal(currency.Code{}, "")
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCurrencyIsEmpty)
	}

	_, _, err = (&Schedule{}).GetWithdrawal(currency.BTC, "")
	if !errors.Is(err, errTransferFeeNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errTransferFeeNotFound)
	}

	d := &Schedule{
		chainTransfer: map[*currency.Item]map[string]*transfer{
			currency.BTC.Item: {
				"": {Withdrawal: Convert(0.01)},
			},
		},
	}

	fee, percentage, err := d.GetWithdrawal(currency.BTC, "")
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
	t.Parallel()
	_, err := (*Schedule)(nil).GetAllFees()
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	d := Schedule{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {},
		},
		pairCommissions: map[asset.Item]map[*currency.Item]map[*currency.Item]*CommissionInternal{
			asset.Spot: {currency.BTC.Item: {currency.USDT.Item: {}}},
		},
		chainTransfer: map[*currency.Item]map[string]*transfer{
			currency.BTC.Item: {"": {}},
		},
		bankTransfer: map[bank.Transfer]map[*currency.Item]*transfer{
			bank.WireTransfer: {currency.BTC.Item: {}},
		},
	}
	_, err = d.GetAllFees()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestGetCommissionFee(t *testing.T) {
	t.Parallel()
	_, err := (*Schedule)(nil).GetCommissionFee(asset.Spot, OmitPair)
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	_, err = (&Schedule{}).GetCommissionFee(asset.Spot, OmitPair)
	if !errors.Is(err, errCommissionRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errCommissionRateNotFound)
	}

	_, err = (&Schedule{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {},
		},
	}).GetCommissionFee(asset.Spot, OmitPair)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestSetCommissionFee(t *testing.T) {
	t.Parallel()
	err := (*Schedule)(nil).SetCommissionFee("", OmitPair, 0, 0, true)
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	err = (&Schedule{}).SetCommissionFee("", OmitPair, -1, 0, true)
	if !errors.Is(err, errMakerInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errMakerInvalid)
	}

	err = (&Schedule{}).SetCommissionFee("cows", OmitPair, 0, 0, true)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v but expected: %v", err, asset.ErrNotSupported)
	}

	err = (&Schedule{}).SetCommissionFee(asset.Spot, OmitPair, 0, 0, true)
	if !errors.Is(err, errCommissionRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errCommissionRateNotFound)
	}

	err = (&Schedule{
		globalCommissions: map[asset.Item]*CommissionInternal{
			asset.Spot: {},
		},
	}).SetCommissionFee(asset.Spot, OmitPair, 0, 0, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestGetTransferFee(t *testing.T) {
	t.Parallel()
	_, err := (*Schedule)(nil).GetTransferFee(currency.Code{}, "")
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	_, err = (&Schedule{}).GetTransferFee(currency.Code{}, "")
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCurrencyIsEmpty)
	}

	_, err = (&Schedule{}).GetTransferFee(currency.BTC, "")
	if !errors.Is(err, errRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errRateNotFound)
	}

	_, err = (&Schedule{
		chainTransfer: map[*currency.Item]map[string]*transfer{
			currency.BTC.Item: {"": {}},
		},
	}).GetTransferFee(currency.BTC, "")
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestSetTransferFee(t *testing.T) {
	t.Parallel()
	err := (*Schedule)(nil).SetTransferFee(currency.Code{}, "", 0, 0, true)
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	err = (&Schedule{}).SetTransferFee(currency.Code{}, "", -1, 0, true)
	if !errors.Is(err, errWithdrawalIsInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errWithdrawalIsInvalid)
	}

	err = (&Schedule{}).SetTransferFee(currency.Code{}, "", 0, -1, true)
	if !errors.Is(err, errDepositIsInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errDepositIsInvalid)
	}

	err = (&Schedule{}).SetTransferFee(currency.Code{}, "", 0, 0, true)
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCurrencyIsEmpty)
	}

	err = (&Schedule{}).SetTransferFee(currency.BTC, "", 0, 0, true)
	if !errors.Is(err, errTransferFeeNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errTransferFeeNotFound)
	}

	err = (&Schedule{
		chainTransfer: map[*currency.Item]map[string]*transfer{
			currency.BTC.Item: {"": {}},
		},
	}).SetTransferFee(currency.BTC, "", 0, 0, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	err = (&Schedule{
		chainTransfer: map[*currency.Item]map[string]*transfer{
			currency.BTC.Item: {"": {}},
		},
	}).SetTransferFee(currency.BTC, "", 0, 0, true)
	if !errors.Is(err, errFeeTypeMismatch) {
		t.Fatalf("received: %v but expected: %v", err, errFeeTypeMismatch)
	}
}

func TestGetBankTransferFee(t *testing.T) {
	t.Parallel()
	_, err := (*Schedule)(nil).GetBankTransferFee(currency.Code{}, 255)
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	_, err = (&Schedule{}).GetBankTransferFee(currency.Code{}, 255)
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCurrencyIsEmpty)
	}

	_, err = (&Schedule{}).GetBankTransferFee(currency.USD, 255)
	if !errors.Is(err, bank.ErrUnknownTransfer) {
		t.Fatalf("received: %v but expected: %v", err, bank.ErrUnknownTransfer)
	}

	_, err = (&Schedule{}).GetBankTransferFee(currency.USD, bank.WireTransfer)
	if !errors.Is(err, errRateNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errRateNotFound)
	}

	_, err = (&Schedule{
		bankTransfer: map[bank.Transfer]map[*currency.Item]*transfer{
			bank.WireTransfer: {currency.USD.Item: {}},
		},
	}).GetBankTransferFee(currency.USD, bank.WireTransfer)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestSetBankTransferFee(t *testing.T) {
	t.Parallel()
	err := (*Schedule)(nil).SetBankTransferFee(currency.Code{}, 255, -1, -1, true)
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}

	err = (&Schedule{}).SetBankTransferFee(currency.Code{}, 255, -1, -1, true)
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCurrencyIsEmpty)
	}

	err = (&Schedule{}).SetBankTransferFee(currency.USD, 255, -1, -1, true)
	if !errors.Is(err, bank.ErrUnknownTransfer) {
		t.Fatalf("received: %v but expected: %v", err, bank.ErrUnknownTransfer)
	}

	err = (&Schedule{}).SetBankTransferFee(currency.USD, bank.WireTransfer, -1, -1, true)
	if !errors.Is(err, errWithdrawalIsInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errWithdrawalIsInvalid)
	}

	err = (&Schedule{}).SetBankTransferFee(currency.USD, bank.WireTransfer, 0, -1, true)
	if !errors.Is(err, errDepositIsInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errDepositIsInvalid)
	}

	err = (&Schedule{}).SetBankTransferFee(currency.USD, bank.WireTransfer, 0, 0, true)
	if !errors.Is(err, errBankTransferFeeNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errBankTransferFeeNotFound)
	}

	err = (&Schedule{
		bankTransfer: map[bank.Transfer]map[*currency.Item]*transfer{
			bank.WireTransfer: {currency.USD.Item: {}},
		},
	}).SetBankTransferFee(currency.USD, bank.WireTransfer, 0, 0, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	err = (&Schedule{
		bankTransfer: map[bank.Transfer]map[*currency.Item]*transfer{
			bank.WireTransfer: {currency.USD.Item: {}},
		},
	}).SetBankTransferFee(currency.USD, bank.WireTransfer, 0, 0, true)
	if !errors.Is(err, errFeeTypeMismatch) {
		t.Fatalf("received: %v but expected: %v", err, errFeeTypeMismatch)
	}
}

func TestLoadChainTransferFees(t *testing.T) {
	t.Parallel()
	var def *Schedule
	err := def.LoadChainTransferFees(nil)
	if !errors.Is(err, ErrScheduleIsNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrScheduleIsNil)
	}
	def = &Schedule{chainTransfer: make(map[*currency.Item]map[string]*transfer)}
	err = def.LoadChainTransferFees(nil)
	if !errors.Is(err, errNoTransferFees) {
		t.Fatalf("received: %v but expected: %v", err, errNoTransferFees)
	}

	newTransfers := []Transfer{{
		Deposit: Convert(-1), // This should fail
	}}

	err = def.LoadChainTransferFees(newTransfers)
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCurrencyIsEmpty)
	}

	newTransfers = []Transfer{{
		Currency: currency.BTC,
		Deposit:  Convert(1),
	}}

	err = def.LoadChainTransferFees(newTransfers)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	newTransfers = []Transfer{{
		Currency:     currency.BTC,
		Deposit:      Convert(2),
		IsPercentage: true, // This should fail
	}}

	err = def.LoadChainTransferFees(newTransfers)
	if !errors.Is(err, errFeeTypeMismatch) {
		t.Fatalf("received: %v but expected: %v", err, errFeeTypeMismatch)
	}
}
