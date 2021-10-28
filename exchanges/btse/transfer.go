package btse

import (
	"encoding/json"
	"errors"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bank"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fee"
)

var (
	errBelowMinimumAmount    = errors.New("amount is less than minimum amount")
	errCurrencyCodeIsEmpty   = errors.New("currency code is empty")
	errCannotCompare         = errors.New("cannot compare")
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
var transferFees = map[asset.Item]map[currency.Code]fee.Transfer{
	asset.Spot: {
		currency.AAVE:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.4003), Withdrawal: fee.Convert(0.1003)},
		currency.ADA:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(8), Withdrawal: fee.Convert(1)},
		currency.ATOM:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(.02), Withdrawal: fee.Convert(.01)},
		currency.BAL:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2.889), Withdrawal: fee.Convert(1.389)},
		currency.BAND:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(5.7), Withdrawal: fee.Convert(2.85)},
		currency.BCB:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(.1)},
		currency.BNB:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(.008), Withdrawal: fee.Convert(.0005)},
		currency.BNT:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(15.885), Withdrawal: fee.Convert(7.885)},
		currency.BRZ:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(23), Withdrawal: fee.Convert(22)},
		currency.BTC:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.001), Withdrawal: fee.Convert(0.0005)},
		currency.BTSE:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(12.461), Withdrawal: fee.Convert(2.461)},
		currency.BUSD:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(50), Withdrawal: fee.Convert(25)},
		currency.COMP:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1.01243), Withdrawal: fee.Convert(0.01243)},
		currency.CRV:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(20), Withdrawal: fee.Convert(10)},
		currency.DAI:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(39.97), Withdrawal: fee.Convert(29.97)},
		currency.DOGE:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2), Withdrawal: fee.Convert(0.82)},
		currency.DOT:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2), Withdrawal: fee.Convert(0.1)},
		currency.ETH:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.042118), Withdrawal: fee.Convert(0.002118)},
		currency.FIL:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.01), Withdrawal: fee.Convert(0.001)},
		currency.FLY:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(168), Withdrawal: fee.Convert(118)},
		currency.FRM:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(100), Withdrawal: fee.Convert(80)},
		currency.FTT:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(.5)},
		currency.HT:    {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(5.718), Withdrawal: fee.Convert(3.718)},
		currency.HXRO:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(150), Withdrawal: fee.Convert(50)},
		currency.JST:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(350), Withdrawal: fee.Convert(250)},
		currency.LEO:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(19.22), Withdrawal: fee.Convert(10.22)},
		currency.LINK:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2.138), Withdrawal: fee.Convert(1.138)},
		currency.LTC:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.002), Withdrawal: fee.Convert(0.001)},
		currency.MATIC: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.2), Withdrawal: fee.Convert(0.1)},
		currency.MBM:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(300), Withdrawal: fee.Convert(200)},
		currency.MKR:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.02), Withdrawal: fee.Convert(0.01)},
		currency.PAX:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(39.99), Withdrawal: fee.Convert(29.99)},
		currency.PHNX:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(150), Withdrawal: fee.Convert(140)},
		currency.SFI:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.1), Withdrawal: fee.Convert(0.001)},
		currency.SHIB:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(3953154), Withdrawal: fee.Convert(2305154)},
		currency.STAKE: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2.979), Withdrawal: fee.Convert(1.979)},
		currency.SUSHI: {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(5.892), Withdrawal: fee.Convert(2.892)},
		currency.SWRV:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(5), Withdrawal: fee.Convert(4)},
		currency.TRX:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2), Withdrawal: fee.Convert(1)},
		currency.TRYB:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(2), Withdrawal: fee.Convert(1.4)},
		currency.TUSD:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(40.97), Withdrawal: fee.Convert(29.97)},
		currency.UNI:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(3.202), Withdrawal: fee.Convert(1.202)},
		currency.USDC:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(40.97), Withdrawal: fee.Convert(29.97)},
		currency.USDP:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(.1)},
		currency.USDT:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(59.96), Withdrawal: fee.Convert(29.96)},
		currency.WAUD:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(13)},
		currency.WCAD:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(12)},
		currency.WCHF:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(9)},
		currency.WEUR:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(8)},
		currency.WGBP:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(7)},
		currency.WHKD:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(77)},
		currency.WINR:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(729)},
		currency.WJPY:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(1050)},
		currency.WMYR:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(40)},
		currency.WOO:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(133.29), Withdrawal: fee.Convert(33.29)},
		currency.WSGD:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(13)},
		currency.WUSD:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(10)},
		currency.WXMR:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(0.06)},
		currency.XAUT:  {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.02303), Withdrawal: fee.Convert(0.01703)},
		currency.XMR:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.002), Withdrawal: fee.Convert(0.001)},
		currency.XRP:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(1), Withdrawal: fee.Convert(.25)},
		currency.YFI:   {Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.0014953), Withdrawal: fee.Convert(0.0009953)},
	},
	// TODO: ADD IN NETWORK HANDLING
	// currency.BUSD:{Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(10), Withdrawal: fee.Convert(.5)}, // BEP20
	// currency.ETH:{Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(0.02), Withdrawal: fee.Convert(0.01)}, // TRC20
	// currency.USDT:{Deposit: fee.Convert(0), MinimumWithdrawal: fee.Convert(10), Withdrawal: fee.Convert(1)}, // TRC20
}

// bankTransferFees defines bank transfer fees between exchange and bank. Subject
// to change.
// NOTE: https://support.btse.com/en/support/solutions/articles/43000588188#Fiat
var bankTransferFees = map[bank.Transfer]map[currency.Code]fee.Transfer{
	bank.Swift: {
		currency.USD: {
			Deposit:    getDeposit(currency.USD),
			Withdrawal: getWithdrawal(currency.USD, standardRate, minimumUSDCharge, true)},
		currency.EUR: {
			Deposit:    getDeposit(currency.EUR),
			Withdrawal: getWithdrawal(currency.EUR, standardRate, minimumUSDCharge, true)},
		currency.GBP: {
			Deposit:    getDeposit(currency.GBP),
			Withdrawal: getWithdrawal(currency.GBP, standardRate, minimumUSDCharge, true)},
		currency.HKD: {
			Deposit:    getDeposit(currency.HKD),
			Withdrawal: getWithdrawal(currency.HKD, standardRate, minimumUSDCharge, true)},
		currency.SGD: {
			Deposit:    getDeposit(currency.SGD),
			Withdrawal: getWithdrawal(currency.SGD, standardRate, minimumUSDCharge, true)},
		currency.JPY: {
			Deposit:    getDeposit(currency.JPY),
			Withdrawal: getWithdrawal(currency.JPY, standardRate, minimumUSDCharge, true)},
		currency.AUD: {
			Deposit:    getDeposit(currency.AUD),
			Withdrawal: getWithdrawal(currency.AUD, standardRate, minimumUSDCharge, true)},
		currency.AED: {
			Deposit:    getDeposit(currency.AED),
			Withdrawal: getWithdrawal(currency.AED, standardRate, minimumUSDCharge, true)},
		currency.CAD: {
			Deposit:    getDeposit(currency.CAD),
			Withdrawal: getWithdrawal(currency.CAD, standardRate, minimumUSDCharge, true)},
	},
	bank.FasterPaymentService: {
		currency.GBP: {
			Deposit:    getDeposit(currency.GBP),
			Withdrawal: getWithdrawal(currency.GBP, decimal.NewFromFloat(.0009), decimal.NewFromInt(3), false)},
	},
	bank.SEPA: {
		currency.EUR: {
			Deposit:    getDeposit(currency.EUR),
			Withdrawal: getWithdrawal(currency.EUR, standardRate, decimal.NewFromInt(3), false)},
	},
}

func getWithdrawal(c currency.Code, percentageRate, minimumCharge decimal.Decimal, usdValuedMinCharge bool) fee.Value {
	return &Withdrawal{
		Code:              c,
		MinimumInUSD:      minimumAmountInUSD, // $100 USD value.
		PercentageRate:    percentageRate,     // 0.1% fee
		MinimumCharge:     minimumCharge,      // $25 USD value
		USDValueMinCharge: usdValuedMinCharge,
	}
}

// Withdrawal defines a value structure that implements the fee.Value interface.
// Can have minimum charge in USD terms.
type Withdrawal struct {
	Code              currency.Code   `json:"code"`
	MinimumInUSD      decimal.Decimal `json:"minimumInUSD"`
	PercentageRate    decimal.Decimal `json:"percentageRate"`
	MinimumCharge     decimal.Decimal `json:"minimumCharge"`
	USDValueMinCharge bool            `json:"usdValueMinCharge"`
}

// GetFee returns the fee based off the amount requested
func (w Withdrawal) GetFee(amount float64) (decimal.Decimal, error) {
	amt := decimal.NewFromFloat(amount)
	potentialFee := amt.Mul(w.PercentageRate)
	if w.Code.Item == currency.USD.Item {
		if amt.LessThan(w.MinimumInUSD) {
			return decimal.Zero, errBelowMinimumAmount
		}
		if potentialFee.LessThanOrEqual(w.MinimumCharge) {
			return w.MinimumCharge, nil
		}
		return potentialFee, nil
	}
	// attempt to attain correct foreign exchange value compared to USD
	fxRate, err := currency.ConvertCurrency(1, w.Code, currency.USD)
	if err != nil {
		return decimal.Zero, err
	}

	fxRateDec := decimal.NewFromFloat(fxRate)
	valueInUSD := amt.Mul(fxRateDec)
	if valueInUSD.LessThan(w.MinimumInUSD) {
		return decimal.Zero, errBelowMinimumAmount
	}

	if w.USDValueMinCharge {
		// In the event the min amount is a USD amount and you need to convert
		// the 25 USD amount to another currency ie EUR :. 25 USD =~= 21.53 EUR
		feeInUSD := valueInUSD.Mul(potentialFee)
		if feeInUSD.LessThanOrEqual(w.MinimumCharge) {
			// Return the minimum charge in the current currency
			invRate := decimal.NewFromFloat(1 / fxRate) // Gets inverse
			return w.MinimumCharge.Mul(invRate), nil
		}
	} else if potentialFee.LessThanOrEqual(w.MinimumCharge) {
		// Return the minimum charge in the current currency
		return w.MinimumCharge, nil
	}
	return potentialFee, nil
}

// Display displays current working internal data for use in RPC outputs
func (w Withdrawal) Display() (string, error) {
	data, err := json.Marshal(w)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Validate validates current values
func (w *Withdrawal) Validate() error {
	if w.Code.IsEmpty() {
		return errCurrencyCodeIsEmpty
	}
	if w.MinimumInUSD.LessThanOrEqual(decimal.Zero) {
		return errInvalidMinimumInUSD
	}
	if w.PercentageRate.LessThanOrEqual(decimal.Zero) {
		return errInvalidPercentageRate
	}
	if w.MinimumCharge.LessThanOrEqual(decimal.Zero) {
		return errInvalidMinimumCharge
	}
	return nil
}

// LessThan implements value interface, not needed.
func (w *Withdrawal) LessThan(_ fee.Value) (bool, error) {
	return false, errors.New("cannot compare")
}

func getDeposit(c currency.Code) fee.Value {
	return &Deposit{
		Code:             c,
		MinimumAmountUSD: minimumAmountInUSD,
		Fee:              minimumDepositCharge,
	}
}

// Deposit defines a fee of $3 USD which will be applied to single deposits of
// less than $100 USD or its equivalent.
type Deposit struct {
	Code             currency.Code   `json:"code"`
	MinimumAmountUSD decimal.Decimal `json:"minimumAmountUSD"`
	Fee              decimal.Decimal `json:"fee"`
}

// GetFee returns the fee based off the amount requested
func (d Deposit) GetFee(amount float64) (decimal.Decimal, error) {
	amt := decimal.NewFromFloat(amount)
	if d.Code.Item == currency.USD.Item {
		if amt.LessThan(d.MinimumAmountUSD) {
			return d.Fee, nil
		}
		return decimal.Zero, nil
	}
	// attempt to attain correct foreign exchange value compared to USD
	fxRate, err := currency.ConvertCurrency(1, d.Code, currency.USD)
	if err != nil {
		return decimal.Zero, err
	}

	fxRateDec := decimal.NewFromFloat(fxRate)
	valueInUSD := amt.Mul(fxRateDec)
	if valueInUSD.LessThan(d.MinimumAmountUSD) {
		invRate := decimal.NewFromFloat(1 / fxRate) // Gets inverse
		return d.Fee.Mul(invRate), nil
	}
	return decimal.Zero, nil
}

// Display displays current working internal data for use in RPC outputs
func (w Deposit) Display() (string, error) {
	data, err := json.Marshal(w)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Validate validates current values
func (d *Deposit) Validate() error {
	if d.Code.IsEmpty() {
		return errCurrencyCodeIsEmpty
	}
	return nil
}

// LessThan implements value interface, not needed.
func (d *Deposit) LessThan(_ fee.Value) (bool, error) {
	return false, errCannotCompare
}
