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
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
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

var (
	p                                     = &Poloniex{}
	spotTradablePair, futuresTradablePair currency.Pair
)

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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
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
		Pair:          spotTradablePair,
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
		Pair:          spotTradablePair,
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
		Pair:      spotTradablePair,
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
	var start, end time.Time
	if mockTests {
		start = time.Unix(1588741402, 0)
		end = time.Unix(1588745003, 0)
	} else {
		start = time.Now().Add(-time.Hour * 2)
		end = time.Now()
	}
	result, err := p.GetHistoricCandles(context.Background(), spotTradablePair, asset.Spot, kline.FiveMin, start, end)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.GetHistoricCandles(context.Background(), futuresTradablePair, asset.Futures, kline.FiveMin, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	start := time.Unix(1718546646, 0)
	end := time.Unix(1718550246, 0)
	if !mockTests {
		start = time.Now().Add(-time.Hour)
		end = time.Now()
	}
	result, err := p.GetHistoricCandlesExtended(context.Background(), spotTradablePair, asset.Spot, kline.FiveMin, start, end)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.GetHistoricCandlesExtended(context.Background(), futuresTradablePair, asset.Futures, kline.FiveMin, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	result, err := p.GetRecentTrades(context.Background(), spotTradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = p.GetRecentTrades(context.Background(), futuresTradablePair, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	tStart := time.Date(2020, 6, 6, 0, 0, 0, 0, time.UTC)
	tEnd := time.Date(2020, 6, 6, 1, 0, 0, 0, time.UTC)
	if !mockTests {
		tmNow := time.Now()
		tStart = time.Date(tmNow.Year(), tmNow.Month()-3, 6, 0, 0, 0, 0, time.UTC)
		tEnd = time.Date(tmNow.Year(), tmNow.Month()-3, 7, 0, 0, 0, 0, time.UTC)
	}
	result, err := p.GetHistoricTrades(context.Background(),
		spotTradablePair, asset.Spot, tStart, tEnd)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.GetHistoricTrades(context.Background(),
		futuresTradablePair, asset.Futures, tStart, tEnd)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	result, err := p.UpdateTicker(context.Background(), spotTradablePair, asset.Spot)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.UpdateTicker(context.Background(), futuresTradablePair, asset.Futures)
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
	result, err := p.GetAvailableTransferChains(context.Background(), currency.NewCode("SHIT"))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetAccountFundingHistory(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
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
			Pair:      spotTradablePair,
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
	require.NotZero(t, st)

	st, err = p.GetServerTime(context.Background(), asset.Futures)
	require.NoError(t, err)
	assert.NotZero(t, st)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := p.GetFuturesContractDetails(context.Background(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	result, err := p.GetFuturesContractDetails(context.Background(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := p.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
		Asset:                asset.Spot,
		Pair:                 spotTradablePair,
		IncludePredictedRate: false,
	})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture)

	result, err := p.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
		Asset:                asset.Futures,
		Pair:                 futuresTradablePair,
		IncludePredictedRate: false,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := p.IsPerpetualFutureCurrency(asset.Spot, spotTradablePair)
	require.NoError(t, err)
	assert.False(t, is)

	is, err = p.IsPerpetualFutureCurrency(asset.Futures, futuresTradablePair)
	require.NoError(t, err)
	assert.True(t, is)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	result, err := p.FetchTradablePairs(context.Background(), asset.Spot)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.FetchTradablePairs(context.Background(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolInformation(t *testing.T) {
	t.Parallel()
	result, err := p.GetSymbolInformation(context.Background(), spotTradablePair)
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
	result, err := p.GetMarketPrice(context.Background(), spotTradablePair)
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

	result, err := p.GetMarkPrice(context.Background(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarkPriceComponents(t *testing.T) {
	t.Parallel()
	result, err := p.MarkPriceComponents(context.Background(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	result, err := p.GetOrderbook(context.Background(), spotTradablePair, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	result, err := p.UpdateOrderbook(context.Background(), spotTradablePair, asset.Spot)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.UpdateOrderbook(context.Background(), futuresTradablePair, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	result, err := p.GetCandlesticks(context.Background(), spotTradablePair, kline.FiveMin, time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	result, err := p.GetTrades(context.Background(), spotTradablePair, 10)
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
	result, err := p.GetTicker(context.Background(), spotTradablePair)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetAccountInformation(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetAllBalances(context.Background(), "")
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.GetAllBalances(context.Background(), "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetAllBalance(context.Background(), "219961623421431808", "")
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.GetAllBalance(context.Background(), "219961623421431808", "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllAccountActivities(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetAccountTransferRecords(context.Background(), time.Time{}, time.Time{}, "", currency.BTC, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountTransferRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetAccountTransferRecord(context.Background(), "23123123120")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFeeInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetFeeInfo(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInterestHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetInterestHistory(context.Background(), time.Time{}, time.Time{}, "", 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountInformations(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetSubAccountInformations(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetSubAccountBalances(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetSubAccountTransferRecords(context.Background(), currency.BTC, time.Time{}, time.Now(), "", "", "", "", "", 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountTransferRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetSubAccountTransferRecord(context.Background(), "1234567")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddresses(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetDepositAddresses(context.Background(), currency.LTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetOrderInfo(context.Background(), "1234", spotTradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetDepositAddress(context.Background(), currency.LTC, "", "USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWalletActivity(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetAccountMarginInformation(context.Background(), "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetBorrowStatus(context.Background(), currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMaximumBuySellAmount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.MaximumBuySellAmount(context.Background(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	_, err := p.PlaceOrder(context.Background(), nil)
	require.ErrorIs(t, err, errNilArgument)

	_, err = p.PlaceOrder(context.Background(), &PlaceOrderParams{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = p.PlaceOrder(context.Background(), &PlaceOrderParams{
		Symbol: spotTradablePair,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.PlaceOrder(context.Background(), &PlaceOrderParams{
		Symbol:        spotTradablePair,
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	_, err := p.PlaceBatchOrders(context.Background(), nil)
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.PlaceBatchOrders(context.Background(), []PlaceOrderParams{{}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = p.PlaceBatchOrders(context.Background(), []PlaceOrderParams{
		{
			Symbol: spotTradablePair,
		},
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	var pair currency.Pair
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetOpenOrders(context.Background(), spotTradablePair, "", "NEXT", "", 10)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CancelMultipleOrdersByIDs(context.Background(), &OrderCancellationParams{OrderIDs: []string{"1234"}, ClientOrderIDs: []string{"5678"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllTradeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.CancelAllTradeOrders(context.Background(), []string{"BTC_USDT", "ETH_USDT"}, []string{"SPOT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestKillSwitch(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
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

	_, err = p.CreateSmartOrder(context.Background(), &SmartOrderRequestParam{
		Symbol: spotTradablePair,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CreateSmartOrder(context.Background(), &SmartOrderRequestParam{
		Symbol:        spotTradablePair,
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetSmartOpenOrders(context.Background(), 10)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGetSmartOrderDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetOrdersHistory(context.Background(), spotTradablePair, "SPOT", "", "", "", "", 0, 10, time.Time{}, time.Time{}, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSmartOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetSmartOrderHistory(context.Background(), spotTradablePair, "SPOT", "", "", "", "", 0, 10, time.Time{}, time.Time{}, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetTradeHistory(context.Background(), currency.Pairs{spotTradablePair}, "", 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeOrderID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
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
	require.NotEmpty(t, subscriptions)

	result, err := p.handleSubscriptions("subscribe", subscriptions)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

var pushMessages = map[string]string{
	"AccountBalance": `{ "channel": "balances", "data": [{ "changeTime": 1657312008411, "accountId": "1234", "accountType": "SPOT", "eventType": "place_order", "available": "9999999983.668", "currency": "BTC", "id": 60018450912695040, "userId": 12345, "hold": "16.332", "ts": 1657312008443 }] }`,
	"Orders":         `{ "channel": "orders", "data": [ { "symbol": "BTC_USDC", "type": "LIMIT", "quantity": "1", "orderId": "32471407854219264", "tradeFee": "0", "clientOrderId": "", "accountType": "SPOT", "feeCurrency": "", "eventType": "place", "source": "API", "side": "BUY", "filledQuantity": "0", "filledAmount": "0", "matchRole": "MAKER", "state": "NEW", "tradeTime": 0, "tradeAmount": "0", "orderAmount": "0", "createTime": 1648708186922, "price": "47112.1", "tradeQty": "0", "tradePrice": "0", "tradeId": "0", "ts": 1648708187469 } ] }`,
	"Candles":        `{"channel":"candles_minute_5","data":[{"symbol":"BTC_USDT","open":"25143.19","high":"25148.58","low":"25138.76","close":"25144.55","quantity":"0.860454","amount":"21635.20983974","tradeCount":20,"startTime":1694469000000,"closeTime":1694469299999,"ts":1694469049867}]}`,
	"BooksLV2":       `{"channel":"book_lv2","data":[{"symbol":"BTC_USDC","createTime":1694469187745,"asks":[],"bids":[["25148.81","0.02158"],["25088.11","0"]],"lastId":598273385,"id":598273386,"ts":1694469187760}],"action":"update"}`,
	"Books":          `{"channel":"book","data":[{"symbol":"BTC_USDC","createTime":1694469187686,"asks":[["25157.24","0.444294"],["25157.25","0.024357"],["25157.26","0.003204"],["25163.39","0.039476"],["25163.4","0.110047"]],"bids":[["25148.8","0.00692"],["25148.61","0.021581"],["25148.6","0.034504"],["25148.59","0.065405"],["25145.52","0.79537"]],"id":598273384,"ts":1694469187733}]}`,
	"Tickers":        `{"channel":"ticker","data":[{"symbol":"BTC_USDC","startTime":1694382780000,"open":"25866.3","high":"26008.47","low":"24923.65","close":"25153.02","quantity":"1626.444884","amount":"41496808.63699303","tradeCount":37124,"dailyChange":"-0.0276","markPrice":"25154.9","closeTime":1694469183664,"ts":1694469187081}]}`,
	"Trades":         `{"channel":"trades","data":[{"symbol":"BTC_USDC","amount":"52.821342","quantity":"0.0021","takerSide":"sell","createTime":1694469183664,"price":"25153.02","id":"71076055","ts":1694469183673}]}`,
	"Currencies":     `{"channel":"currencies","data":[[{"currency":"BTC","id":28,"name":"Bitcoin","description":"BTC Clone","type":"address","withdrawalFee":"0.0008","minConf":2,"depositAddress":null,"blockchain":"BTC","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["BTCTRON"]},{"currency":"XRP","id":243,"name":"XRP","description":"Payment ID","type":"address-payment-id","withdrawalFee":"0.2","minConf":2,"depositAddress":"rwU8rAiE2eyEPz3sikfbHuqCuiAtdXqa2v","blockchain":"XRP","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":false,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":[]},{"currency":"ETH","id":267,"name":"Ethereum","description":"Sweep to Main Account","type":"address","withdrawalFee":"0.00197556","minConf":64,"depositAddress":null,"blockchain":"ETH","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["ETHTRON"]},{"currency":"USDT","id":214,"name":"Tether USD","description":"Sweep to Main Account","type":"address","withdrawalFee":"0","minConf":2,"depositAddress":null,"blockchain":"OMNI","delisted":false,"tradingState":"NORMAL","walletState":"DISABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["USDTETH","USDTTRON"]},{"currency":"DOGE","id":59,"name":"Dogecoin","description":"BTC Clone","type":"address","withdrawalFee":"20","minConf":6,"depositAddress":null,"blockchain":"DOGE","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["DOGETRON"]},{"currency":"LTC","id":125,"name":"Litecoin","description":"BTC Clone","type":"address","withdrawalFee":"0.001","minConf":4,"depositAddress":null,"blockchain":"LTC","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["LTCTRON"]},{"currency":"DASH","id":60,"name":"Dash","description":"BTC Clone","type":"address","withdrawalFee":"0.01","minConf":20,"depositAddress":null,"blockchain":"DASH","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":false,"isChildChain":false,"supportCollateral":false,"supportBorrow":false,"childChains":[]}]],"action":"snapshot"}`,
	"Symbols":        `{"channel":"symbols","data":[[{"symbol":"BTC_USDC","baseCurrencyName":"BTC","quoteCurrencyName":"USDT","displayName":"BTC/USDT","state":"NORMAL","visibleStartTime":1659018819512,"tradableStartTime":1659018819512,"crossMargin":{"supportCrossMargin":true,"maxLeverage":"3"},"symbolTradeLimit":{"symbol":"BTC_USDT","priceScale":2,"quantityScale":6,"amountScale":2,"minQuantity":"0.000001","minAmount":"1","highestBid":"0","lowestAsk":"0"}}]],"action":"snapshot"}`,
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

	_, err = p.WsCreateOrder(&PlaceOrderParams{
		Symbol: spotTradablePair,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.WsCreateOrder(&PlaceOrderParams{
		Symbol:        spotTradablePair,
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.WsCancelMultipleOrdersByIDs(&OrderCancellationParams{OrderIDs: []string{"1234"}, ClientOrderIDs: []string{"5678"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelAllTradeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
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
	result, err := p.GetFuturesRealTimeTickersOfSymbols(context.Background())
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

var futuresPushDataMap = map[string]string{
	"Public Ticker":                        `{"subject":"ticker", "topic": "/contractMarket/ticker:BTCUSDTPERP", "data": { "symbol": "BTCUSDTPERP", "sequence": 45, "side": "sell", "price": 3600.00, "size": 16, "tradeId": "5c9dcf4170744d6f5a3d32fb", "bestBidSize": 795, "bestBidPrice": 3200.00, "bestAskPrice": 3600.00, "bestAskSize": 284, "ts": 1553846081210004941 } }`,
	"Level2 Orderbook":                     `{"subject":"level2", "topic": "/contractMarket/level2:BTCUSDTPERP", "type": "message", "data": { "lastSequence": 8, "sequence": 18, "change": "5000.0,sell,83", "changes": ["5000.0,sell,83","5001.0,sell,3"], "timestamp": 1551770400000 } }`,
	"Public Trade Execution":               `{"topic":"/contractMarket/execution:BTCUSDTPERP", "subject": "match", "data": { "symbol": "BTCUSDTPERP", "sequence": 36, "side": "buy", "matchSize": 1, "size": 1, "price": 3200.00, "takerOrderId": "5c9dd00870744d71c43f5e25", "ts": 1553846281766256031, "makerOrderId": "5c9d852070744d0976909a0c", "tradeId": "5c9dd00970744d6f5a3d32fc" } }`,
	"Level3 V2 Received":                   `{"topic":"/contractMarket/level3v2:BTCUSDTPERP", "subject": "received", "data": { "symbol": "BTCUSDTPERP", "sequence": 3262786900, "orderId": "5c0b520032eba53a888fd02x", "clientOid": "ad123ad", "ts": 1545914149935808589 } }`,
	"Level3 V2 Open":                       `{"topic":"/contractMarket/level3v2:BTCUSDTPERP", "subject": "open", "data": { "symbol": "BTCUSDTPERP", "sequence": 3262786900, "side": "buy", "price": "3634.5", "size": "10", "orderId": "5c0b520032eba53a888fd02x", "orderTime": 1547697294838004923, "ts": 1547697294838004923} }`,
	"Level3 V2 Update":                     `{"topic":"/contractMarket/level3v2:BTCUSDTPERP", "subject": "update", "data": { "symbol": "BTCUSDTPERP", "sequence": 3262786897, "orderId": "5c0b520032eba53a888fd01f", "size": "100", "ts": 1547697294838004923 } }`,
	"Level3 V2 Match":                      `{"topic":"/contractMarket/level3v2:BTCUSDTPERP", "subject": "match", "data": { "symbol": "BTCUSDTPERP", "sequence": 3262786901, "side": "buy", "price": "3634", "size": "10", "makerOrderId": "5c0b520032eba53a888fd01e", "takerOrderId": "5c0b520032eba53a888fd01f", "tradeId": "6c23b5454353a8882d023b3o", "ts": 1547697294838004923 } }`,
	"Level3 V2 Done":                       `{"topic":"/contractMarket/level3v2:BTCUSDTPERP", "subject": "done", "data": { "symbol": "BTCUSDTPERP", "sequence": 3262786901, "reason": "filled", "orderId": "5c0b520032eba53a888fd02x", "ts": 1547697294838004923}}`,
	"Level2 Depth5":                        `{"type":"message", "topic": "/contractMarket/level2Depth5:BTCUSDTPERP", "subject": "level2", "data": { "asks":[ ["9993", "3"], ["9992", "3"], ["9991", "47"], ["9990", "32"], ["9989", "8"] ], "bids":[ ["9988", "56"], ["9987", "15"], ["9986", "100"], ["9985", "10"], ["9984", "10"] ], "ts": 1590634672060667000 } }`,
	"Level2 Depth50":                       `{"type":"message", "topic": "/contractMarket/level2Depth50:BTCUSDTPERP", "subject": "level2", "data": { "asks":[ ["9993",3], ["9992",3], ["9991",47], ["9990",32], ["9989",8] ], "bids":[ ["9988",56], ["9987",15], ["9986",100], ["9985",10], ["9984",10] ], "ts": 1590634672060667000}}`,
	"Contract Instrument":                  `{"topic":"/contract/instrument:BTCUSDTPERP", "subject": "mark.index.price", "data": { "granularity": 1000, "indexPrice": 4000.23, "markPrice": 4010.52, "timestamp": 1551770400000 } }`,
	"Funding Rate":                         `{"topic":"/contract/instrument:BTCUSDTPERP", "subject": "funding.rate", "data": { "granularity": 60000, "fundingRate": -0.002966, "timestamp": 1551770400000 } }`,
	"Start Funding Fee Settlement":         `{"topic":"/contract/announcement", "subject": "funding.begin", "data": { "symbol": "BTCUSDTPERP", "fundingTime": 1551770400000, "fundingRate": -0.002966, "timestamp": 1551770400000 } }`,
	"End Funding Fee Settlement":           `{"type":"message", "topic": "/contract/announcement", "subject": "funding.end", "data": { "symbol": "BTCUSDTPERP", "fundingTime": 1551770400000, "fundingRate": -0.002966, "timestamp": 1551770410000 } }`,
	"Transaction Statistics Timer Event":   `{"topic":"/contractMarket/snapshot:BTCUSDTPERP", "subject": "snapshot.24h", "data": { "volume": 30449670, "turnover": 845169919063, "lastPrice": 3551, "priceChgPct": 0.0043, "ts": 1547697294838004923 } }`,
	"User Private Message":                 `{"type":"message", "topic": "/contractMarket/tradeOrders", "subject": "orderChange", "channelType": "private", "data": { "orderId": "5cdfc138b21023a909e5ad55", "symbol": "BTCUSDTPERP", "type": "match", "marginType": 0, "status": "open", "matchSize": "", "matchPrice": "", "orderType": "limit", "side": "buy", "price": "3600", "size": "20000", "remainSize": "20001", "filledSize":"20000", "canceledSize": "0", "tradeId": "5ce24c16b210233c36eexxxx", "clientOid": "5ce24c16b210233c36ee321d", "orderTime": 1545914149935808589, "oldSize ": "15000", "liquidity": "maker", "ts": 1545914149935808589 } }`,
	"Stop Order Lifecycle Event":           `{"topic":"/contractMarket/advancedOrders", "subject": "stopOrder", "channelType": "private", "data": { "orderId": "5cdfc138b21023a909e5ad55", "symbol": "BTCUSDTPERP", "type": "open", "marginType": 0, "orderType":"stop", "side":"buy", "size":"1000", "orderPrice":"9000", "stop":"up", "stopPrice":"9100", "stopPriceType":"TP", "triggerSuccess": true, "error": "error.createOrder.accountBalanceInsufficient", "createdAt": 1558074652423, "ts":1558074652423004000}}`,
	"Account Order Margin Event":           `{"topic":"/contractAccount/wallet", "subject": "orderMargin.change", "channelType": "private", "data": { "orderMargin": 5923, "currency":"USDT", "timestamp": 1553842862614 } }`,
	"Available Balance Event":              `{"topic":"/contractAccount/wallet", "subject": "availableBalance.change", "channelType": "private", "data": { "availableBalance": 5923, "currency":"USDT", "timestamp": 1553842862614 } }`,
	"Position Change Caused Operation":     `{"topic":"/contract/position:BTCUSDTPERP", "subject": "position.change", "channelType": "private", "data": { "realisedGrossPnl": 0.0001, "marginType": 0, "liquidationPrice": 1000000.0, "posLoss": 0E-8, "avgEntryPrice": 7508.22, "unrealisedPnl": -0.00014735, "markPrice": 7947.83, "posMargin": 0.00266779, "riskLimit": 200, "unrealisedCost": 0.00266375, "posComm": 0.00000392, "posMaint": 0.00001724, "posCost": 0.00266375, "maintMarginReq": 0.005, "bankruptPrice": 1000000.0, "realisedCost": 0.00000271, "markValue": 0.00251640, "posInit": 0.00266375, "realisedPnl": -0.00000253, "maintMargin": 0.00252044, "realLeverage": 1.06, "currentCost": 0.00266375, "openingTimestamp": 1558433191000, "currentQty": -20, "delevPercentage": 0.52, "currentComm": 0.00000271, "realisedGrossCost": 0E-8, "isOpen": true, "posCross": 1.2E-7, "currentTimestamp": 1558506060394, "unrealisedRoePcnt": -0.0553, "unrealisedPnlPcnt": -0.0553, "settleCurrency": "XBT" } }`,
	"Position Change Caused by Mark Price": `{"topic":"/contract/position:BTCUSDTPERP", "subject": "position.change", "channelType": "private", "data": { "marginType": 0, "markPrice": 7947.83, "markValue": 0.00251640, "maintMargin": 0.00252044, "realLeverage": 10.06, "unrealisedPnl": -0.00014735, "unrealisedRoePcnt": -0.0553, "unrealisedPnlPcnt": -0.0553, "delevPercentage": 0.52, "currentTimestamp": 1558087175068, "settleCurrency": "XBT" } }`,
	"Funding Settlement":                   `{"topic":"/contract/position:BTCUSDTPERP", "subject": "position.settlement", "channelType": "private", "data": { "fundingTime": 1551770400000, "qty": 100, "markPrice": 3610.85, "fundingRate": -0.002966, "fundingFee": -296, "ts": 1547697294838004923, "settleCurrency": "XBT" } }`,
	"Close Position Information":           `{"topic":"/contract/positionCross", "subject": "positionCross.change", "channelType": "private", "data": { "maintainMargin" : 100.99, "marginAvailable" : 30.98, "maintainRate" : 0.38, "unreleaseSum" : 1.5, "fundingSum" : -0.3, "accountAvailable" : 100.1 } }`,
}

func TestWsFuturesHandleData(t *testing.T) {
	t.Parallel()
	var err error
	for title, data := range futuresPushDataMap {
		err = p.wsFuturesHandleData([]byte(data))
		require.NoErrorf(t, err, "%s: unexpected error %v", title, err)
	}
}

func populateTradablepairs() error {
	err := p.UpdateTradablePairs(context.Background(), false)
	if err != nil {
		return err
	}
	tradablePairs, err := p.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}
	spotTradablePair = tradablePairs[0]
	tradablePairs, err = p.GetEnabledPairs(asset.Futures)
	if err != nil {
		return err
	}
	futuresTradablePair = tradablePairs[0]
	return nil
}

func TestWsFuturesConnect(t *testing.T) {
	t.Parallel()
	err := p.WsFuturesConnect()
	require.NoError(t, err)
	time.Sleep(time.Second * 23)
}

func TestGetFuturesAccountOverview(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetFuturesAccountOverview(context.Background(), currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesAccountTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetFuturesAccountTransactionHistory(context.Background(), time.Now().Add(-time.Hour*50), time.Now(), "RealisedPNL", 0, 100, currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
