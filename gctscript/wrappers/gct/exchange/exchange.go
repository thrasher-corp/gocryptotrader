package exchange

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Exchange implements all required methods for Wrapper
type Exchange struct{}

// Exchanges returns slice of all current exchanges
func (e Exchange) Exchanges(enabledOnly bool) []string {
	return engine.Bot.GetExchangeNames(enabledOnly)
}

// GetExchange returns IBotExchange for exchange or error if exchange is not found
func (e Exchange) GetExchange(exch string) (exchange.IBotExchange, error) {
	ex := engine.Bot.GetExchangeByName(exch)
	if ex == nil {
		return nil, fmt.Errorf("%v exchange not found", exch)
	}

	return ex, nil
}

// IsEnabled returns if requested exchange is enabled or disabled
func (e Exchange) IsEnabled(exch string) bool {
	ex, err := e.GetExchange(exch)
	if err != nil {
		return false
	}

	return ex.IsEnabled()
}

// Orderbook returns current orderbook requested exchange, pair and asset
func (e Exchange) Orderbook(exch string, pair currency.Pair, item asset.Item) (*orderbook.Base, error) {
	return engine.Bot.GetSpecificOrderbook(pair, exch, item)
}

// Ticker returns ticker for provided currency pair & asset type
func (e Exchange) Ticker(exch string, pair currency.Pair, item asset.Item) (*ticker.Price, error) {
	ex, err := e.GetExchange(exch)
	if err != nil {
		return nil, err
	}

	return ex.FetchTicker(pair, item)
}

// Pairs returns either all or enabled currency pairs
func (e Exchange) Pairs(exch string, enabledOnly bool, item asset.Item) (*currency.Pairs, error) {
	x, err := engine.Bot.Config.GetExchangeConfig(exch)
	if err != nil {
		return nil, err
	}

	ps, err := x.CurrencyPairs.Get(item)
	if err != nil {
		return nil, err
	}

	if enabledOnly {
		return &ps.Enabled, nil
	}
	return &ps.Available, nil
}

// QueryOrder returns details of a valid exchange order
func (e Exchange) QueryOrder(exch, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	o, err := engine.Bot.OrderManager.GetOrderInfo(exch, orderID, pair, assetType)
	if err != nil {
		return nil, err
	}

	return &o, nil
}

// SubmitOrder submit new order on exchange
func (e Exchange) SubmitOrder(submit *order.Submit) (*order.SubmitResponse, error) {
	r, err := engine.Bot.OrderManager.Submit(submit)
	if err != nil {
		return nil, err
	}

	return &r.SubmitResponse, nil
}

// CancelOrder wrapper to cancel order on exchange
func (e Exchange) CancelOrder(exch, orderID string, cp currency.Pair, a asset.Item) (bool, error) {
	orderDetails, err := e.QueryOrder(exch, orderID, cp, a)
	if err != nil {
		return false, err
	}

	cancel := &order.Cancel{
		AccountID: orderDetails.AccountID,
		ID:        orderDetails.ID,
		Pair:      orderDetails.Pair,
		Side:      orderDetails.Side,
		AssetType: orderDetails.AssetType,
		Exchange:  exch,
	}

	err = engine.Bot.OrderManager.Cancel(cancel)
	if err != nil {
		return false, err
	}
	return true, nil
}

// AccountInformation returns account information (balance etc) for requested exchange
func (e Exchange) AccountInformation(exch string, assetType asset.Item) (account.Holdings, error) {
	ex, err := e.GetExchange(exch)
	if err != nil {
		return account.Holdings{}, err
	}

	accountInfo, err := ex.FetchAccountInfo(assetType)
	if err != nil {
		return account.Holdings{}, err
	}

	return accountInfo, nil
}

// DepositAddress gets the address required to deposit funds for currency type
func (e Exchange) DepositAddress(exch string, currencyCode currency.Code) (out string, err error) {
	if currencyCode.IsEmpty() {
		err = errors.New("currency code is empty")
		return
	}
	return engine.Bot.DepositAddressManager.GetDepositAddressByExchangeAndCurrency(exch, currencyCode)
}

// WithdrawalFiatFunds withdraw funds from exchange to requested fiat source
func (e Exchange) WithdrawalFiatFunds(bankAccountID string, request *withdraw.Request) (string, error) {
	ex, err := e.GetExchange(request.Exchange)
	if err != nil {
		return "", err
	}
	var v *banking.Account
	v, err = banking.GetBankAccountByID(bankAccountID)
	if err != nil {
		v, err = ex.GetBase().GetExchangeBankAccounts(bankAccountID, request.Currency.String())
		if err != nil {
			return "", err
		}
	}

	otp, err := engine.Bot.GetExchangeOTPByName(request.Exchange)
	if err == nil {
		otpValue, errParse := strconv.ParseInt(otp, 10, 64)
		if errParse != nil {
			return "", errors.New("failed to generate OTP unable to continue")
		}
		request.OneTimePassword = otpValue
	}
	request.Fiat.Bank.AccountName = v.AccountName
	request.Fiat.Bank.AccountNumber = v.AccountNumber
	request.Fiat.Bank.BankName = v.BankName
	request.Fiat.Bank.BankAddress = v.BankAddress
	request.Fiat.Bank.BankPostalCity = v.BankPostalCity
	request.Fiat.Bank.BankCountry = v.BankCountry
	request.Fiat.Bank.BankPostalCode = v.BankPostalCode
	request.Fiat.Bank.BSBNumber = v.BSBNumber
	request.Fiat.Bank.SWIFTCode = v.SWIFTCode
	request.Fiat.Bank.IBAN = v.IBAN

	resp, err := engine.Bot.WithdrawManager.SubmitWithdrawal(request)
	if err != nil {
		return "", err
	}
	return resp.Exchange.ID, nil
}

// WithdrawalCryptoFunds withdraw funds from exchange to requested Crypto source
func (e Exchange) WithdrawalCryptoFunds(request *withdraw.Request) (string, error) {
	// Checks if exchange is enabled or not so we don't call OTP generation
	_, err := e.GetExchange(request.Exchange)
	if err != nil {
		return "", err
	}
	otp, err := engine.Bot.GetExchangeOTPByName(request.Exchange)
	if err == nil {
		v, errParse := strconv.ParseInt(otp, 10, 64)
		if errParse != nil {
			return "", errors.New("failed to generate OTP unable to continue")
		}
		request.OneTimePassword = v
	}

	resp, err := engine.Bot.WithdrawManager.SubmitWithdrawal(request)
	if err != nil {
		return "", err
	}
	return resp.Exchange.ID, nil
}

// OHLCV returns open high low close volume candles for requested exchange/pair/asset/start & end time
func (e Exchange) OHLCV(exch string, pair currency.Pair, item asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	ex, err := e.GetExchange(exch)
	if err != nil {
		return kline.Item{}, err
	}
	ret, err := ex.GetHistoricCandlesExtended(pair, item, start, end, interval)
	if err != nil {
		return kline.Item{}, err
	}

	sort.Slice(ret.Candles, func(i, j int) bool {
		return ret.Candles[i].Time.Before(ret.Candles[j].Time)
	})

	ret.FormatDates()

	return ret, nil
}
