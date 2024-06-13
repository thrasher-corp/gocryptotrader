package poloniex

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own APIKEYS here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var p = &Poloniex{}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.LTC.String(),
			currency.BTC.String(),
			"-"),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()
	var feeBuilder = setFeeBuilder()
	result, err := p.GetFeeByType(context.Background(), feeBuilder)
	require.NoError(t, err)
	if !sharedtestvalues.AreAPICredentialsSet(p) {
		require.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType)
	} else {
		require.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		require.NotNil(t, result)
	}
}

// TODO: update
func TestGetFee(t *testing.T) {
	t.Parallel()
	var feeBuilder = setFeeBuilder()

	if sharedtestvalues.AreAPICredentialsSet(p) || mockTests {
		// CryptocurrencyTradeFee Basic
		result, err := p.GetFee(context.Background(), feeBuilder)
		require.NoError(t, err)
		require.NotNil(t, result)

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		result, err = p.GetFee(context.Background(), feeBuilder)
		require.NoError(t, err)
		require.NotNil(t, result)

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		result, err = p.GetFee(context.Background(), feeBuilder)
		require.NoError(t, err)
		require.NotNil(t, result)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	result, err := p.GetFee(context.Background(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	result, err = p.GetFee(context.Background(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	result, err = p.GetFee(context.Background(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	result, err = p.GetFee(context.Background(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	result, err = p.GetFee(context.Background(), feeBuilder)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetActiveOrders(context.Background(), &order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}
	result, err := p.GetOrderHistory(context.Background(), &getOrdersRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	var orderSubmission = &order.Submit{
		Exchange: p.Name,
		Pair: currency.Pair{
			Delimiter: currency.UnderscoreDelimiter,
			Base:      currency.BTC,
			Quote:     currency.LTC,
		},
		Side:      order.Buy,
		Type:      order.Market,
		Price:     10,
		Amount:    10000000,
		ClientID:  "hi",
		AssetType: asset.Spot,
	}
	result, err := p.SubmitOrder(context.Background(), orderSubmission)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.SubmitOrder(context.Background(), &order.Submit{
		Exchange: p.Name,
		Pair: currency.Pair{
			Delimiter: currency.UnderscoreDelimiter,
			Base:      currency.BTC,
			Quote:     currency.LTC,
		},
		Side:         order.Buy,
		Type:         order.Market,
		TriggerPrice: 11,
		Price:        10,
		Amount:       10000000,
		ClientID:     "hi",
		AssetType:    asset.Spot,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		AssetType:     asset.Spot,
	}
	err := p.CancelOrder(context.Background(), orderCancellation)
	assert.NoError(t, err)
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		AssetType:     asset.Spot,
	}

	result, err := p.CancelAllOrders(context.Background(), orderCancellation)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.ModifyOrder(context.Background(), &order.Modify{
		OrderID:   "1337",
		Price:     1337,
		AssetType: asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	withdrawCryptoRequest := withdraw.Request{
		Exchange: p.Name,
		Crypto: withdraw.CryptoRequest{
			Address:   core.BitcoinDonationAddress,
			FeeAmount: 0,
		},
		Amount:        1,
		Currency:      currency.LTC,
		Description:   "WITHDRAW IT ALL",
		TradePassword: "Password",
	}
	result, err := p.WithdrawCryptocurrencyFunds(context.Background(), &withdrawCryptoRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.UpdateAccountInfo(context.Background(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	var withdrawFiatRequest withdraw.Request
	_, err := p.WithdrawFiatFunds(context.Background(), &withdrawFiatRequest)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := p.WithdrawFiatFundsToInternationalBank(context.Background(), &withdraw.Request{})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	var start, end time.Time
	if mockTests {
		start = time.Unix(1588741402, 0)
		end = time.Unix(1588745003, 0)
	} else {
		start = time.Now().Add(-time.Hour * 2)
		end = time.Now()
	}
	result, err := p.GetHistoricCandles(context.Background(), pair, asset.Spot, kline.FiveMin, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	var start, end time.Time
	if mockTests {
		start = time.Unix(1588741402, 0)
		end = time.Unix(1588745003, 0)
	} else {
		start = time.Now().Add(-time.Hour)
		end = time.Now()
	}
	result, err := p.GetHistoricCandlesExtended(context.Background(), pair, asset.Spot, kline.FiveMin, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC_XMR")
	require.NoError(t, err)
	if mockTests {
		t.Skip("relies on time.Now()")
	}
	result, err := p.GetRecentTrades(context.Background(), currencyPair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC_XMR")
	require.NoError(t, err)
	tStart := time.Date(2020, 6, 6, 0, 0, 0, 0, time.UTC)
	tEnd := time.Date(2020, 6, 6, 1, 0, 0, 0, time.UTC)
	if !mockTests {
		tmNow := time.Now()
		tStart = time.Date(tmNow.Year(), tmNow.Month()-3, 6, 0, 0, 0, 0, time.UTC)
		tEnd = time.Date(tmNow.Year(), tmNow.Month()-3, 7, 0, 0, 0, 0, time.UTC)
	}
	result, err := p.GetHistoricTrades(context.Background(),
		currencyPair, asset.Spot, tStart, tEnd)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	result, err := p.UpdateTicker(context.Background(), cp, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := p.UpdateTickers(context.Background(), asset.Spot)
	assert.NoError(t, err)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	result, err := p.GetAvailableTransferChains(context.Background(), currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetAccountFundingHistory(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip("could not be mock tested")
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CancelBatchOrders(context.Background(), []order.Cancel{
		{
			OrderID:   "1234",
			AssetType: asset.Spot,
			Pair:      currency.NewPair(currency.BTC, currency.USD),
		},
		{
			OrderID:   "134",
			AssetType: asset.Spot,
			Pair:      currency.NewPair(currency.BTC, currency.USD),
		},
		{
			OrderID:   "234",
			AssetType: asset.Spot,
			Pair:      currency.NewPair(currency.BTC, currency.USD),
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := p.GetServerTime(context.Background(), asset.Spot)
	require.NoError(t, err)
	assert.Zero(t, st)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	result, err := p.FetchTradablePairs(context.Background(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolInformation(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("ETH_USDT")
	require.NoError(t, err)

	result, err := p.GetSymbolInformation(context.Background(), pair)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.GetSymbolInformation(context.Background(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyInformations(t *testing.T) {
	t.Parallel()
	result, err := p.GetCurrencyInformations(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyInformation(t *testing.T) {
	t.Parallel()
	result, err := p.GetCurrencyInformation(context.Background(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV2CurrencyInformations(t *testing.T) {
	t.Parallel()
	result, err := p.GetV2CurrencyInformations(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV2CurrencyInformation(t *testing.T) {
	t.Parallel()
	result, err := p.GetV2CurrencyInformation(context.Background(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSystemTimestamp(t *testing.T) {
	t.Parallel()
	result, err := p.GetSystemTimestamp(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarketPrices(t *testing.T) {
	t.Parallel()
	result, err := p.GetMarketPrices(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarketPrice(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("TRX_USDC")
	require.NoError(t, err)
	result, err := p.GetMarketPrice(context.Background(), pair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPrices(t *testing.T) {
	t.Parallel()
	result, err := p.GetMarkPrices(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPrice(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	result, err := p.GetMarkPrice(context.Background(), pair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarkPriceComponents(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	result, err := p.MarkPriceComponents(context.Background(), pair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	result, err := p.GetOrderbook(context.Background(), pair, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	result, err := p.UpdateOrderbook(context.Background(), pair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	result, err := p.GetCandlesticks(context.Background(), pair, kline.FiveMin, time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	result, err := p.GetTrades(context.Background(), pair, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	result, err := p.GetTickers(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	result, err := p.GetTicker(context.Background(), pair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCollateralInfos(t *testing.T) {
	t.Parallel()
	result, err := p.GetCollateralInfos(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCollateralInfo(t *testing.T) {
	t.Parallel()
	result, err := p.GetCollateralInfo(context.Background(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowRateInfo(t *testing.T) {
	t.Parallel()
	result, err := p.GetBorrowRateInfo(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetAccountInformation(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetAllBalances(context.Background(), "")
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.GetAllBalances(context.Background(), "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetAllBalance(context.Background(), "219961623421431808", "")
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.GetAllBalance(context.Background(), "219961623421431808", "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllAccountActivities(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetAllAccountActivities(context.Background(), time.Time{}, time.Time{}, 0, 0, 0, "", currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAccountsTransfer(t *testing.T) {
	t.Parallel()
	_, err := p.AccountsTransfer(context.Background(), nil)
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.AccountsTransfer(context.Background(), &AccountTransferParams{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = p.AccountsTransfer(context.Background(), &AccountTransferParams{
		Ccy: currency.BTC,
	})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)
	_, err = p.AccountsTransfer(context.Background(), &AccountTransferParams{
		Amount:      1,
		Ccy:         currency.BTC,
		FromAccount: "219961623421431808",
	})
	require.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.AccountsTransfer(context.Background(), &AccountTransferParams{
		Amount:      1,
		Ccy:         currency.BTC,
		FromAccount: "219961623421431808",
		ToAccount:   "219961623421431890",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountTransferRecords(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetAccountTransferRecords(context.Background(), time.Time{}, time.Time{}, "", currency.BTC, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountTransferRecord(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetAccountTransferRecord(context.Background(), "23123123120")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFeeInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetFeeInfo(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInterestHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetInterestHistory(context.Background(), time.Time{}, time.Time{}, "", 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountInformations(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetSubAccountInformations(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetSubAccountBalances(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetSubAccountBalance(context.Background(), "2d45301d-5f08-4a2b-a763-f9199778d854")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubAccountTransfer(t *testing.T) {
	t.Parallel()
	_, err := p.SubAccountTransfer(context.Background(), nil)
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.SubAccountTransfer(context.Background(), &SubAccountTransferParam{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = p.SubAccountTransfer(context.Background(), &SubAccountTransferParam{
		Currency: currency.BTC,
	})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)
	_, err = p.SubAccountTransfer(context.Background(), &SubAccountTransferParam{
		Currency: currency.BTC,
		Amount:   1,
	})
	require.ErrorIs(t, err, errAccountIDRequired)
	_, err = p.SubAccountTransfer(context.Background(), &SubAccountTransferParam{
		Currency:      currency.BTC,
		Amount:        1,
		FromAccountID: "1234568",
		ToAccountID:   "1234567",
	})
	require.ErrorIs(t, err, errAccountTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.SubAccountTransfer(context.Background(), &SubAccountTransferParam{
		Currency:        currency.BTC,
		Amount:          1,
		FromAccountID:   "1234568",
		ToAccountID:     "1234567",
		FromAccountType: "SPOT",
		ToAccountType:   "SPOT",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountTransferRecords(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetSubAccountTransferRecords(context.Background(), currency.BTC, time.Time{}, time.Now(), "", "", "", "", "", 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountTransferRecord(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetSubAccountTransferRecord(context.Background(), "1234567")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddresses(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetDepositAddresses(context.Background(), currency.LTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	result, err := p.GetOrderInfo(context.Background(), "1234", pair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetDepositAddress(context.Background(), currency.LTC, "", "USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWalletActivity(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	var start, end time.Time
	if mockTests {
		start = time.UnixMilli(1693741163970)
		end = time.UnixMilli(1693748363970)
	} else {
		start = time.Now().Add(-time.Hour * 2)
		end = time.Now()
	}
	result, err := p.WalletActivity(context.Background(), start, end, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewCurrencyDepoditAddress(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	result, err := p.NewCurrencyDepositAddress(context.Background(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawCurrency(t *testing.T) {
	t.Parallel()
	_, err := p.WithdrawCurrency(context.Background(), nil)
	assert.ErrorIs(t, err, errNilArgument)
	_, err = p.WithdrawCurrency(context.Background(), &WithdrawCurrencyParam{
		Currency: currency.BTC,
	})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = p.WithdrawCurrency(context.Background(), &WithdrawCurrencyParam{
		Currency: currency.BTC,
		Amount:   1,
	})
	require.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.WithdrawCurrency(context.Background(), &WithdrawCurrencyParam{
		Currency: currency.BTC,
		Amount:   1,
		Address:  "0xbb8d0d7c346daecc2380dabaa91f3ccf8ae232fb4",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawCurrencyV2(t *testing.T) {
	t.Parallel()
	_, err := p.WithdrawCurrencyV2(context.Background(), nil)
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.WithdrawCurrencyV2(context.Background(), &WithdrawCurrencyV2Param{
		Coin: currency.BTC})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = p.WithdrawCurrencyV2(context.Background(), &WithdrawCurrencyV2Param{Coin: currency.BTC, Amount: 1})
	require.ErrorIs(t, err, errInvalidWithdrawalChain)
	_, err = p.WithdrawCurrencyV2(context.Background(), &WithdrawCurrencyV2Param{
		Coin: currency.BTC, Amount: 1, Network: "BTC"})
	require.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.WithdrawCurrencyV2(context.Background(), &WithdrawCurrencyV2Param{
		Network: "BTC", Coin: currency.BTC, Amount: 1, Address: "0xbb8d0d7c346daecc2380dabaa91f3ccf8ae232fb4"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountMarginInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetAccountMarginInformation(context.Background(), "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetBorrowStatus(context.Background(), currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMaximumBuySellAmount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	result, err := p.MaximumBuySellAmount(context.Background(), pair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	_, err := p.PlaceOrder(context.Background(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = p.PlaceOrder(context.Background(), &PlaceOrderParams{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	_, err = p.PlaceOrder(context.Background(), &PlaceOrderParams{
		Symbol: pair,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.PlaceOrder(context.Background(), &PlaceOrderParams{
		Symbol:        pair,
		Side:          order.Buy.String(),
		Type:          order.Market.String(),
		Quantity:      100,
		Price:         40000.50000,
		TimeInForce:   "GTC",
		ClientOrderID: "1234Abc",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceBatchOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	_, err := p.PlaceBatchOrders(context.Background(), nil)
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.PlaceBatchOrders(context.Background(), []PlaceOrderParams{{}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	require.NotEmpty(t, pair)

	_, err = p.PlaceBatchOrders(context.Background(), []PlaceOrderParams{
		{
			Symbol: pair,
		},
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	getPairFromString := func(pairString string) currency.Pair {
		pair, err = currency.NewPairFromString(pairString)
		if err != nil {
			return currency.EMPTYPAIR
		}
		return pair
	}
	result, err := p.PlaceBatchOrders(context.Background(), []PlaceOrderParams{
		{
			Symbol:        pair,
			Side:          order.Buy.String(),
			Type:          order.Market.String(),
			Quantity:      100,
			Price:         40000.50000,
			TimeInForce:   "GTC",
			ClientOrderID: "1234Abc",
		},
		{
			Symbol: getPairFromString("BTC_USDT"),
			Amount: 100,
			Side:   "BUY",
		},
		{
			Symbol:        getPairFromString("BTC_USDT"),
			Type:          "LIMIT",
			Quantity:      100,
			Side:          "BUY",
			Price:         40000.50000,
			TimeInForce:   "IOC",
			ClientOrderID: "1234Abc",
		},
		{
			Symbol: getPairFromString("ETH_USDT"),
			Amount: 1000,
			Side:   "BUY",
		},
		{
			Symbol:        getPairFromString("TRX_USDT"),
			Type:          "LIMIT",
			Quantity:      15000,
			Side:          "SELL",
			Price:         0.0623423423,
			TimeInForce:   "IOC",
			ClientOrderID: "456Xyz",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelReplaceOrder(t *testing.T) {
	t.Parallel()
	_, err := p.CancelReplaceOrder(context.Background(), &CancelReplaceOrderParam{})
	require.ErrorIs(t, err, errNilArgument)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CancelReplaceOrder(context.Background(), &CancelReplaceOrderParam{
		orderID:       "29772698821328896",
		ClientOrderID: "1234Abc",
		Price:         18000,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	result, err := p.GetOpenOrders(context.Background(), pair, "", "NEXT", "", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetOrderDetail(context.Background(), "12345536545645", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrderByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CancelOrderByID(context.Background(), "12345536545645")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMultipleOrdersByIDs(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	result, err := p.CancelMultipleOrdersByIDs(context.Background(), &OrderCancellationParams{OrderIDs: []string{"1234"}, ClientOrderIDs: []string{"5678"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllTradeOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.CancelAllTradeOrders(context.Background(), []string{"BTC_USDT", "ETH_USDT"}, []string{"SPOT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestKillSwitch(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.KillSwitch(context.Background(), "30")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetKillSwitchStatus(t *testing.T) {
	t.Parallel()
	result, err := p.GetKillSwitchStatus(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSmartOrder(t *testing.T) {
	t.Parallel()
	_, err := p.CreateSmartOrder(context.Background(), &SmartOrderRequestParam{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.CreateSmartOrder(context.Background(), &SmartOrderRequestParam{
		Side: "BUY",
	})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	require.NotEmpty(t, pair)

	_, err = p.CreateSmartOrder(context.Background(), &SmartOrderRequestParam{
		Symbol: pair,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CreateSmartOrder(context.Background(), &SmartOrderRequestParam{
		Symbol:        pair,
		Side:          "BUY",
		Type:          orderTypeString(order.StopLimit),
		Quantity:      100,
		Price:         40000.50000,
		TimeInForce:   "GTC",
		ClientOrderID: "1234Abc",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelReplaceSmartOrder(t *testing.T) {
	t.Parallel()
	_, err := p.CancelReplaceSmartOrder(context.Background(), &CancelReplaceSmartOrderParam{})
	require.ErrorIs(t, err, errNilArgument)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CancelReplaceSmartOrder(context.Background(), &CancelReplaceSmartOrderParam{
		orderID:       "29772698821328896",
		ClientOrderID: "1234Abc",
		Price:         18000,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSmartOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetSmartOpenOrders(context.Background(), 10)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGetSmartOrderDetail(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetSmartOrderDetail(context.Background(), "123313413", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelSmartOrderByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CancelSmartOrderByID(context.Background(), "123313413", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMultipleSmartOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	result, err := p.CancelMultipleSmartOrders(context.Background(), &OrderCancellationParams{OrderIDs: []string{"1234"}, ClientOrderIDs: []string{"5678"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllSmartOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CancelAllSmartOrders(context.Background(), []string{"BTC_USDT", "ETH_USDT"}, []string{"SPOT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrdersHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	require.NotEmpty(t, pair)
	result, err := p.GetOrdersHistory(context.Background(), pair, "SPOT", "", "", "", "", 0, 10, time.Time{}, time.Time{}, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSmartOrderHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	require.NotEmpty(t, pair)

	result, err := p.GetSmartOrderHistory(context.Background(), pair, "SPOT", "", "", "", "", 0, 10, time.Time{}, time.Time{}, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)

	result, err := p.GetTradeHistory(context.Background(), currency.Pairs{pair}, "", 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeOrderID(t *testing.T) {
	t.Parallel()
	result, err := p.GetTradesByOrderID(context.Background(), "13123242323")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateDefaultSubscriptions(t *testing.T) {
	result, err := p.GenerateDefaultSubscriptions()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestHandlePayloads(t *testing.T) {
	t.Parallel()
	subscriptions, err := p.GenerateDefaultSubscriptions()
	require.NoError(t, err)
	result, err := p.handleSubscriptions("subscribe", subscriptions)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

var pushMessages = map[string]string{
	"AccountBalance": `{ "channel": "balances", "data": [{ "changeTime": 1657312008411, "accountId": "1234", "accountType": "SPOT", "eventType": "place_order", "available": "9999999983.668", "currency": "BTC", "id": 60018450912695040, "userId": 12345, "hold": "16.332", "ts": 1657312008443 }] }`,
	"Orders":         `{ "channel": "orders", "data": [ { "symbol": "BTC_USDT", "type": "LIMIT", "quantity": "1", "orderId": "32471407854219264", "tradeFee": "0", "clientOrderId": "", "accountType": "SPOT", "feeCurrency": "", "eventType": "place", "source": "API", "side": "BUY", "filledQuantity": "0", "filledAmount": "0", "matchRole": "MAKER", "state": "NEW", "tradeTime": 0, "tradeAmount": "0", "orderAmount": "0", "createTime": 1648708186922, "price": "47112.1", "tradeQty": "0", "tradePrice": "0", "tradeId": "0", "ts": 1648708187469 } ] }`,
	"Candles":        `{"channel":"candles_minute_5","data":[{"symbol":"BTC_USDT","open":"25143.19","high":"25148.58","low":"25138.76","close":"25144.55","quantity":"0.860454","amount":"21635.20983974","tradeCount":20,"startTime":1694469000000,"closeTime":1694469299999,"ts":1694469049867}]}`,
	"BooksLV2":       `{"channel":"book_lv2","data":[{"symbol":"BTC_USDT","createTime":1694469187745,"asks":[],"bids":[["25148.81","0.02158"],["25088.11","0"]],"lastId":598273385,"id":598273386,"ts":1694469187760}],"action":"update"}`,
	"Books":          `{"channel":"book","data":[{"symbol":"BTC_USDT","createTime":1694469187686,"asks":[["25157.24","0.444294"],["25157.25","0.024357"],["25157.26","0.003204"],["25163.39","0.039476"],["25163.4","0.110047"]],"bids":[["25148.8","0.00692"],["25148.61","0.021581"],["25148.6","0.034504"],["25148.59","0.065405"],["25145.52","0.79537"]],"id":598273384,"ts":1694469187733}]}`,
	"Tickers":        `{"channel":"ticker","data":[{"symbol":"BTC_USDT","startTime":1694382780000,"open":"25866.3","high":"26008.47","low":"24923.65","close":"25153.02","quantity":"1626.444884","amount":"41496808.63699303","tradeCount":37124,"dailyChange":"-0.0276","markPrice":"25154.9","closeTime":1694469183664,"ts":1694469187081}]}`,
	"Trades":         `{"channel":"trades","data":[{"symbol":"BTC_USDT","amount":"52.821342","quantity":"0.0021","takerSide":"sell","createTime":1694469183664,"price":"25153.02","id":"71076055","ts":1694469183673}]}`,
	"Currencies":     `{"channel":"currencies","data":[[{"currency":"BTC","id":28,"name":"Bitcoin","description":"BTC Clone","type":"address","withdrawalFee":"0.0008","minConf":2,"depositAddress":null,"blockchain":"BTC","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["BTCTRON"]},{"currency":"XRP","id":243,"name":"XRP","description":"Payment ID","type":"address-payment-id","withdrawalFee":"0.2","minConf":2,"depositAddress":"rwU8rAiE2eyEPz3sikfbHuqCuiAtdXqa2v","blockchain":"XRP","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":false,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":[]},{"currency":"ETH","id":267,"name":"Ethereum","description":"Sweep to Main Account","type":"address","withdrawalFee":"0.00197556","minConf":64,"depositAddress":null,"blockchain":"ETH","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["ETHTRON"]},{"currency":"USDT","id":214,"name":"Tether USD","description":"Sweep to Main Account","type":"address","withdrawalFee":"0","minConf":2,"depositAddress":null,"blockchain":"OMNI","delisted":false,"tradingState":"NORMAL","walletState":"DISABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["USDTETH","USDTTRON"]},{"currency":"DOGE","id":59,"name":"Dogecoin","description":"BTC Clone","type":"address","withdrawalFee":"20","minConf":6,"depositAddress":null,"blockchain":"DOGE","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["DOGETRON"]},{"currency":"LTC","id":125,"name":"Litecoin","description":"BTC Clone","type":"address","withdrawalFee":"0.001","minConf":4,"depositAddress":null,"blockchain":"LTC","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["LTCTRON"]},{"currency":"DASH","id":60,"name":"Dash","description":"BTC Clone","type":"address","withdrawalFee":"0.01","minConf":20,"depositAddress":null,"blockchain":"DASH","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":false,"isChildChain":false,"supportCollateral":false,"supportBorrow":false,"childChains":[]}]],"action":"snapshot"}`,
	"Symbols":        `{"channel":"symbols","data":[[{"symbol":"BTC_USDT","baseCurrencyName":"BTC","quoteCurrencyName":"USDT","displayName":"BTC/USDT","state":"NORMAL","visibleStartTime":1659018819512,"tradableStartTime":1659018819512,"crossMargin":{"supportCrossMargin":true,"maxLeverage":"3"},"symbolTradeLimit":{"symbol":"BTC_USDT","priceScale":2,"quantityScale":6,"amountScale":2,"minQuantity":"0.000001","minAmount":"1","highestBid":"0","lowestAsk":"0"}}]],"action":"snapshot"}`,
}

func TestWsPushData(t *testing.T) {
	t.Parallel()
	for key, value := range pushMessages {
		err := p.wsHandleData([]byte(value))
		require.NoErrorf(t, err, "%s error %s: %v", p.Name, key, err)
	}
}

func setupWS() {
	if !p.Websocket.IsEnabled() {
		return
	}
	if !sharedtestvalues.AreAPICredentialsSet(p) {
		p.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	err := p.WsConnect()
	if err != nil {
		log.Fatal(err)
	}
}

func TestWsCreateOrder(t *testing.T) {
	t.Parallel()
	_, err := p.WsCreateOrder(nil)
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.WsCreateOrder(&PlaceOrderParams{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	pair, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	_, err = p.WsCreateOrder(&PlaceOrderParams{
		Symbol: pair,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.WsCreateOrder(&PlaceOrderParams{
		Symbol:        pair,
		Side:          order.Buy.String(),
		Type:          order.Market.String(),
		Amount:        1232432,
		Quantity:      100,
		Price:         40000.50000,
		TimeInForce:   "GTC",
		ClientOrderID: "1234Abc",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelMultipleOrdersByIDs(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	result, err := p.WsCancelMultipleOrdersByIDs(&OrderCancellationParams{OrderIDs: []string{"1234"}, ClientOrderIDs: []string{"5678"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelAllTradeOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	result, err := p.WsCancelAllTradeOrders([]string{"BTC_USDT", "ETH_USDT"}, []string{"SPOT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := p.UpdateOrderExecutionLimits(context.Background(), asset.Spot)
	require.NoError(t, err)
	instruments, err := p.GetSymbolInformation(context.Background(), currency.EMPTYPAIR)
	require.NoError(t, err)
	require.Len(t, instruments, 1)
	cp, err := currency.NewPairFromString(instruments[0].Symbol)
	require.NoError(t, err)
	limits, err := p.GetOrderExecutionLimits(asset.Spot, cp)
	require.NoErrorf(t, err, "Asset: %s Pair: %s Err: %v", asset.Spot, cp, err)
	require.Equal(t, limits.PriceStepIncrementSize, instruments[0].SymbolTradeLimit.PriceScale)
	require.Equal(t, limits.MinimumBaseAmount, instruments[0].SymbolTradeLimit.MinQuantity.Float64())
	assert.Equal(t, limits.MinimumQuoteAmount, instruments[0].SymbolTradeLimit.MinAmount.Float64())
}

// ---- Futures endpoints ---

func TestGetOpenContractList(t *testing.T) {
	t.Parallel()
	result, err := p.GetOpenContractList(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderInfoOfTheContract(t *testing.T) {
	t.Parallel()
	result, err := p.GetOrderInfoOfTheContract(context.Background(), "BTCUSDTPERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRealTimeTicker(t *testing.T) {
	t.Parallel()
	result, err := p.GetRealTimeTicker(context.Background(), "BTCUSDTPERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRealTimeTickersOfSymbols(t *testing.T) {
	t.Parallel()
	result, err := p.TestGetRealTimeTickersOfSymbols(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFullOrderbookLevel2(t *testing.T) {
	t.Parallel()
	result, err := p.GetFullOrderbookLevel2(context.Background(), "BTCUSDTPERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPartialOrderbookLevel2(t *testing.T) {
	t.Parallel()
	result, err := p.GetPartialOrderbookLevel2(context.Background(), "BTCUSDTPERP", "depth20")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLevel2PullingMessages(t *testing.T) {
	t.Parallel()
	result, err := p.Level2PullingMessages(context.Background(), "BTCUSDTPERP", 6, 400)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFullOrderBookLevel3(t *testing.T) {
	t.Parallel()
	result, err := p.GetFullOrderBookLevel3(context.Background(), "BTCUSDTPERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLevel3PullingMessages(t *testing.T) {
	t.Parallel()
	result, err := p.Level3PullingMessages(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTransactionHistory(t *testing.T) {
	t.Parallel()
	result, err := p.GetTransactionHistory(context.Background(), "BTCUSDTPERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInterestRateList(t *testing.T) {
	t.Parallel()
	result, err := p.GetInterestRateList(context.Background(), "BTCUSDTPERP", time.Time{}, time.Now(), false, true, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexList(t *testing.T) {
	t.Parallel()
	result, err := p.GetIndexList(context.Background(), "BTCUSDTPERP", time.Time{}, time.Now(), false, true, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentMarkPrice(t *testing.T) {
	t.Parallel()
	result, err := p.GetCurrentMarkPrice(context.Background(), "BTCUSDTPERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPremiumIndex(t *testing.T) {
	t.Parallel()
	result, err := p.GetPremiumIndex(context.Background(), "BTCUSDTPERP", time.Time{}, time.Now(), false, true, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentFundingRate(t *testing.T) {
	t.Parallel()
	result, err := p.GetCurrentFundingRate(context.Background(), "BTCUSDTPERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesServerTime(t *testing.T) {
	t.Parallel()
	result, err := p.GetFuturesServerTime(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetServiceStatus(t *testing.T) {
	t.Parallel()
	result, err := p.GetServiceStatus(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetKlineDataOfContract(t *testing.T) {
	t.Parallel()
	result, err := p.GetKlineDataOfContract(context.Background(), "BTCUSDTPERP", 123, time.Time{}, time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPublicFuturesWebsocketServerInstances(t *testing.T) {
	t.Parallel()
	result, err := p.GetPublicFuturesWebsocketServerInstances(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPrivateFuturesWebsocketServerInstances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetPrivateFuturesWebsocketServerInstances(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, p)
	for _, a := range p.GetAssetTypes(false) {
		pairs, err := p.CurrencyPairs.GetPairs(a, false)
		require.NoError(t, err, "cannot get pairs for %s", a)
		require.NotEmpty(t, pairs, "no pairs for %s", a)
		resp, err := p.GetCurrencyTradeURL(context.Background(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

func TestWsConnect(t *testing.T) {
	t.Parallel()
	err := p.WsConnect()
	require.NoError(t, err)
	time.Sleep(time.Second * 23)
}

func TestWsFuturesConnect(t *testing.T) {
	t.Parallel()
	p.Verbose = true
	err := p.WsFuturesConnect()
	require.NoError(t, err)
	time.Sleep(time.Second * 60)
}
