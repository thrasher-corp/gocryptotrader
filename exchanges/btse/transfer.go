package btse

import (
	"encoding/json"
	"errors"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bank"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fee"
)

var (
	errBelowMinimumAmount    = errors.New("amount is less than minimum amount")
	errCurrencyCodeIsEmpty   = errors.New("currency code is empty")
	errInvalidMinimumInUSD   = errors.New("invalid minimum in USD")
	errInvalidPercentageRate = errors.New("invalid percentage rate")
	errInvalidMinimumCharge  = errors.New("invalid minimum charge")

	minimumUSDCharge     = decimal.NewFromFloat(25)
	minimumAmountInUSD   = decimal.NewFromFloat(100)
	standardRate         = decimal.NewFromFloat(0.001)
	minimumDepositCharge = decimal.NewFromInt(3)
)

// transferFees defines exchange crypto currency transfer fees, subject to
// change.
// NOTE: https://www.btse.com/en/deposit-withdrawal-fees
var defaultTransferFees = []fee.Transfer{
	{Currency: currency.AAVE, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.4003), Withdrawal: fee.Convert(0.1003)},
	{Currency: currency.ADA, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(8), Withdrawal: fee.Convert(1)},
	{Currency: currency.ATOM, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(.02), Withdrawal: fee.Convert(.01)},
	{Currency: currency.BAL, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2.889), Withdrawal: fee.Convert(1.389)},
	{Currency: currency.BAND, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(5.7), Withdrawal: fee.Convert(2.85)},
	{Currency: currency.BCB, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(.1)},
	{Currency: currency.BNB, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(.008), Withdrawal: fee.Convert(.0005)},
	{Currency: currency.BNT, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(15.885), Withdrawal: fee.Convert(7.885)},
	{Currency: currency.BRZ, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(23), Withdrawal: fee.Convert(22)},
	{Currency: currency.BTC, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.001), Withdrawal: fee.Convert(0.0005)},
	{Currency: currency.BTSE, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(12.461), Withdrawal: fee.Convert(2.461)},
	{Currency: currency.BUSD, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(50), Withdrawal: fee.Convert(25), Chain: "ERC20"},
	{Currency: currency.BUSD, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(10), Withdrawal: fee.Convert(.5), Chain: "BEP20"},
	{Currency: currency.COMP, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1.01243), Withdrawal: fee.Convert(0.01243)},
	{Currency: currency.CRV, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(20), Withdrawal: fee.Convert(10)},
	{Currency: currency.DAI, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(39.97), Withdrawal: fee.Convert(29.97)},
	{Currency: currency.DOGE, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2), Withdrawal: fee.Convert(0.82)},
	{Currency: currency.DOT, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2), Withdrawal: fee.Convert(0.1)},
	{Currency: currency.ETH, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.042118), Withdrawal: fee.Convert(0.002118)},
	{Currency: currency.ETH, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.02), Withdrawal: fee.Convert(0.01), Chain: "TRC20"},
	{Currency: currency.FIL, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.01), Withdrawal: fee.Convert(0.001)},
	{Currency: currency.FLY, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(168), Withdrawal: fee.Convert(118)},
	{Currency: currency.FRM, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(100), Withdrawal: fee.Convert(80)},
	{Currency: currency.FTT, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(.5)},
	{Currency: currency.HT, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(5.718), Withdrawal: fee.Convert(3.718)},
	{Currency: currency.HXRO, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(150), Withdrawal: fee.Convert(50)},
	{Currency: currency.JST, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(350), Withdrawal: fee.Convert(250)},
	{Currency: currency.LEO, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(19.22), Withdrawal: fee.Convert(10.22)},
	{Currency: currency.LINK, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2.138), Withdrawal: fee.Convert(1.138)},
	{Currency: currency.LTC, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.002), Withdrawal: fee.Convert(0.001)},
	{Currency: currency.MATIC, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.2), Withdrawal: fee.Convert(0.1)},
	{Currency: currency.MBM, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(300), Withdrawal: fee.Convert(200)},
	{Currency: currency.MKR, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.02), Withdrawal: fee.Convert(0.01)},
	{Currency: currency.PAX, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(39.99), Withdrawal: fee.Convert(29.99)},
	{Currency: currency.PHNX, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(150), Withdrawal: fee.Convert(140)},
	{Currency: currency.SFI, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.1), Withdrawal: fee.Convert(0.001)},
	{Currency: currency.SHIB, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(3953154), Withdrawal: fee.Convert(2305154)},
	{Currency: currency.STAKE, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2.979), Withdrawal: fee.Convert(1.979)},
	{Currency: currency.SUSHI, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(5.892), Withdrawal: fee.Convert(2.892)},
	{Currency: currency.SWRV, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(5), Withdrawal: fee.Convert(4)},
	{Currency: currency.TRX, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2), Withdrawal: fee.Convert(1)},
	{Currency: currency.TRYB, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2), Withdrawal: fee.Convert(1.4)},
	{Currency: currency.TUSD, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(40.97), Withdrawal: fee.Convert(29.97)},
	{Currency: currency.UNI, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(3.202), Withdrawal: fee.Convert(1.202)},
	{Currency: currency.USDC, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(40.97), Withdrawal: fee.Convert(29.97)},
	{Currency: currency.USDP, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(.1)},
	{Currency: currency.USDT, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(59.96), Withdrawal: fee.Convert(29.96), Chain: "ERC20"},
	{Currency: currency.USDT, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(10), Withdrawal: fee.Convert(1), Chain: "TRC20"},
	{Currency: currency.WAUD, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(13)},
	{Currency: currency.WCAD, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(12)},
	{Currency: currency.WCHF, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(9)},
	{Currency: currency.WEUR, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(8)},
	{Currency: currency.WGBP, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(7)},
	{Currency: currency.WHKD, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(77)},
	{Currency: currency.WINR, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(729)},
	{Currency: currency.WJPY, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(1050)},
	{Currency: currency.WMYR, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(40)},
	{Currency: currency.WOO, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(133.29), Withdrawal: fee.Convert(33.29)},
	{Currency: currency.WSGD, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(13)},
	{Currency: currency.WUSD, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(10)},
	{Currency: currency.WXMR, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(0.06)},
	{Currency: currency.XAUT, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.02303), Withdrawal: fee.Convert(0.01703)},
	{Currency: currency.XMR, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.002), Withdrawal: fee.Convert(0.001)},
	{Currency: currency.XRP, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(.25)},
	{Currency: currency.YFI, Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.0014953), Withdrawal: fee.Convert(0.0009953)},
}

// bankTransferFees defines bank transfer fees between exchange and bank. Subject
// to change.
// NOTE: https://support.btse.com/en/support/solutions/articles/43000588188#Fiat
var bankTransferFees = []fee.Transfer{
	{Currency: currency.USD, BankTransfer: bank.Swift, Deposit: getDeposit(currency.USD), Withdrawal: getWithdrawal(currency.USD, standardRate, minimumUSDCharge, true)},
	{Currency: currency.EUR, BankTransfer: bank.Swift, Deposit: getDeposit(currency.EUR), Withdrawal: getWithdrawal(currency.EUR, standardRate, minimumUSDCharge, true)},
	{Currency: currency.GBP, BankTransfer: bank.Swift, Deposit: getDeposit(currency.GBP), Withdrawal: getWithdrawal(currency.GBP, standardRate, minimumUSDCharge, true)},
	{Currency: currency.SGD, BankTransfer: bank.Swift, Deposit: getDeposit(currency.SGD), Withdrawal: getWithdrawal(currency.SGD, standardRate, minimumUSDCharge, true)},
	{Currency: currency.JPY, BankTransfer: bank.Swift, Deposit: getDeposit(currency.JPY), Withdrawal: getWithdrawal(currency.JPY, standardRate, minimumUSDCharge, true)},
	{Currency: currency.AUD, BankTransfer: bank.Swift, Deposit: getDeposit(currency.AUD), Withdrawal: getWithdrawal(currency.AUD, standardRate, minimumUSDCharge, true)},
	{Currency: currency.AED, BankTransfer: bank.Swift, Deposit: getDeposit(currency.AED), Withdrawal: getWithdrawal(currency.AED, standardRate, minimumUSDCharge, true)},
	{Currency: currency.CAD, BankTransfer: bank.Swift, Deposit: getDeposit(currency.CAD), Withdrawal: getWithdrawal(currency.CAD, standardRate, minimumUSDCharge, true)},

	{Currency: currency.CAD, BankTransfer: bank.FasterPaymentService, Deposit: getDeposit(currency.GBP), Withdrawal: getWithdrawal(currency.GBP, decimal.NewFromFloat(.0009), decimal.NewFromInt(3), false)},

	{Currency: currency.EUR, BankTransfer: bank.SEPA, Deposit: getDeposit(currency.EUR), Withdrawal: getWithdrawal(currency.EUR, standardRate, decimal.NewFromInt(3), false)},
}

func getWithdrawal(c currency.Code, percentageRate, minimumCharge decimal.Decimal, usdValuedMinCharge bool) fee.Value {
	return &TransferWithdrawalFee{
		Code:              c,
		MinimumInUSD:      minimumAmountInUSD, // $100 USD value.
		PercentageRate:    percentageRate,     // 0.1% fee
		MinimumCharge:     minimumCharge,      // $25 USD value
		USDValueMinCharge: usdValuedMinCharge,
	}
}

// TransferWithdrawalFee defines a data structure that implements the fee.Value
// interface. This overrides the default implementation so it can have minimum
// charge in USD terms and use foreign exchange rates.
type TransferWithdrawalFee struct {
	Code              currency.Code   `json:"code"`
	MinimumInUSD      decimal.Decimal `json:"minimumInUSD"`
	PercentageRate    decimal.Decimal `json:"percentageRate"`
	MinimumCharge     decimal.Decimal `json:"minimumCharge"`
	USDValueMinCharge bool            `json:"usdValueMinCharge"`
}

// GetFee returns the fee based off the amount requested
func (t TransferWithdrawalFee) GetFee(amount float64) (decimal.Decimal, error) {
	amt := decimal.NewFromFloat(amount)
	potentialFee := amt.Mul(t.PercentageRate)
	if t.Code.Item == currency.USD.Item {
		if amt.LessThan(t.MinimumInUSD) {
			return decimal.Zero, errBelowMinimumAmount
		}
		if potentialFee.LessThanOrEqual(t.MinimumCharge) {
			return t.MinimumCharge, nil
		}
		return potentialFee, nil
	}
	// attempt to attain correct foreign exchange value compared to USD
	fxRate, err := currency.ConvertCurrency(1, t.Code, currency.USD)
	if err != nil {
		return decimal.Zero, err
	}

	fxRateDec := decimal.NewFromFloat(fxRate)
	valueInUSD := amt.Mul(fxRateDec)
	if valueInUSD.LessThan(t.MinimumInUSD) {
		return decimal.Zero, errBelowMinimumAmount
	}

	if t.USDValueMinCharge {
		// In the event the min amount is a USD amount and you need to convert
		// the 25 USD amount to another currency ie EUR :. 25 USD =~= 21.53 EUR
		feeInUSD := valueInUSD.Mul(potentialFee)
		if feeInUSD.LessThanOrEqual(t.MinimumCharge) {
			// Return the minimum charge in the current currency
			invRate := decimal.NewFromFloat(1 / fxRate) // Gets inverse
			return t.MinimumCharge.Mul(invRate), nil
		}
	} else if potentialFee.LessThanOrEqual(t.MinimumCharge) {
		// Return the minimum charge in the current currency
		return t.MinimumCharge, nil
	}
	return potentialFee, nil
}

// Display displays current working internal data for use in RPC outputs
func (t TransferWithdrawalFee) Display() (string, error) {
	data, err := json.Marshal(t)
	return string(data), err
}

// Validate validates current values
func (t *TransferWithdrawalFee) Validate() error {
	if t.Code.IsEmpty() {
		return errCurrencyCodeIsEmpty
	}
	if t.MinimumInUSD.LessThanOrEqual(decimal.Zero) {
		return errInvalidMinimumInUSD
	}
	if t.PercentageRate.LessThanOrEqual(decimal.Zero) {
		return errInvalidPercentageRate
	}
	if t.MinimumCharge.LessThanOrEqual(decimal.Zero) {
		return errInvalidMinimumCharge
	}
	return nil
}

// LessThan implements value interface, not needed.
func (t *TransferWithdrawalFee) LessThan(_ fee.Value) (bool, error) {
	return false, fee.ErrCannotCompare
}

func getDeposit(c currency.Code) fee.Value {
	return &TransferDepositFee{
		Code:             c,
		MinimumAmountUSD: minimumAmountInUSD,
		Fee:              minimumDepositCharge,
	}
}

// TransferDepositFee defines a data structure that implements the fee.Value
// interface. This overrides the default implementation so it can have minimum
// charge in USD terms and use foreign exchange rates.
type TransferDepositFee struct {
	Code             currency.Code   `json:"code"`
	MinimumAmountUSD decimal.Decimal `json:"minimumAmountUSD"`
	Fee              decimal.Decimal `json:"fee"`
}

// GetFee returns the fee based off the amount requested
func (t TransferDepositFee) GetFee(amount float64) (decimal.Decimal, error) {
	amt := decimal.NewFromFloat(amount)
	if t.Code.Item == currency.USD.Item {
		if amt.LessThan(t.MinimumAmountUSD) {
			return t.Fee, nil
		}
		return decimal.Zero, nil
	}
	// attempt to attain correct foreign exchange value compared to USD
	fxRate, err := currency.ConvertCurrency(1, t.Code, currency.USD)
	if err != nil {
		return decimal.Zero, err
	}

	fxRateDec := decimal.NewFromFloat(fxRate)
	valueInUSD := amt.Mul(fxRateDec)
	if valueInUSD.LessThan(t.MinimumAmountUSD) {
		invRate := decimal.NewFromFloat(1 / fxRate) // Gets inverse
		return t.Fee.Mul(invRate), nil
	}
	return decimal.Zero, nil
}

// Display displays current working internal data for use in RPC outputs
func (t TransferDepositFee) Display() (string, error) {
	data, err := json.Marshal(t)
	return string(data), err
}

// Validate validates current values
func (t *TransferDepositFee) Validate() error {
	if t.Code.IsEmpty() {
		return errCurrencyCodeIsEmpty
	}
	return nil
}

// LessThan implements value interface, not needed.
func (t *TransferDepositFee) LessThan(_ fee.Value) (bool, error) {
	return false, fee.ErrCannotCompare
}
