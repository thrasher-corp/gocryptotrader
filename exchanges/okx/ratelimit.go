package okx

import (
	"context"
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	// oneSecondInterval
	oneSecondInterval = time.Second
	// twoSecondInterval
	twoSecondsInterval = 2 * time.Second
	// threeSecondInterval
	threeSecondsInterval = 3 * time.Second
	// fiveSecondsInterval
	fiveSecondsInterval = 5 * time.Second
	// tenSecondsInterval
	tenSecondsInterval = 10 * time.Second
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	// Trade Endpoints
	PlaceOrder             *rate.Limiter
	PlaceMultipleOrders    *rate.Limiter
	CancelOrder            *rate.Limiter
	CancelMultipleOrders   *rate.Limiter
	AmendOrder             *rate.Limiter
	AmendMultipleOrders    *rate.Limiter
	CloseDposit            *rate.Limiter
	GetOrderDetails        *rate.Limiter
	GetOrderList           *rate.Limiter
	GetOrderHistory        *rate.Limiter
	GetTrasactionDetails   *rate.Limiter
	PlaceAlgoOrder         *rate.Limiter
	CancelAlgoOrder        *rate.Limiter
	CancelAdvanceAlgoOrder *rate.Limiter
	GetAlgoOrderList       *rate.Limiter
	GetAlgoOrderhistory    *rate.Limiter
	// Block Trading endpoints
	GetCounterparties    *rate.Limiter
	CreateRfq            *rate.Limiter
	CancelRfq            *rate.Limiter
	CancelMultipleRfq    *rate.Limiter
	CancelAllRfqs        *rate.Limiter
	ExecuteQuote         *rate.Limiter
	CreateQuote          *rate.Limiter
	CancelQuote          *rate.Limiter
	CancelMultipleQuotes *rate.Limiter
	CancelAllQuotes      *rate.Limiter
	GetRfqs              *rate.Limiter
	GetQuotes            *rate.Limiter
	GetTrades            *rate.Limiter
	GetPublicTrades      *rate.Limiter
	// Funding
	GetCurrencies            *rate.Limiter
	GetBalance               *rate.Limiter
	GetAccountAssetValuation *rate.Limiter
	FundsTransfer            *rate.Limiter
	GetFundsTransferState    *rate.Limiter
	AssetBillsDetails        *rate.Limiter
	LigntningDeposits        *rate.Limiter
	GetDepositAddress        *rate.Limiter
	GetDepositHistory        *rate.Limiter
	Withdrawal               *rate.Limiter
	LightningWithdrawals     *rate.Limiter
	CancelWithdrawal         *rate.Limiter
	GetWithdrawalHistory     *rate.Limiter
	SmallAssetsConvert       *rate.Limiter
	GetSavingBalance         *rate.Limiter
	SavingsPurchaseRedemp    *rate.Limiter
	SetLendingRate           *rate.Limiter
	GetLendinghistory        *rate.Limiter
	GetPublicBorrowInfo      *rate.Limiter
	GetPublicBorrowHistory   *rate.Limiter
	// Convert
	GetConvertCurrencies   *rate.Limiter
	GetConvertCurrencyPair *rate.Limiter
	EstimateQuote          *rate.Limiter
	ConvertTrade           *rate.Limiter
	GetConvertHistory      *rate.Limiter

	// Account

	GetAccountBalance                 *rate.Limiter
	GetPositions                      *rate.Limiter
	GetPositionsHistory               *rate.Limiter
	GetAccountAndPositionRisk         *rate.Limiter
	GetBillsDetails                   *rate.Limiter
	GetAccountConfiguration           *rate.Limiter
	SetPositionMode                   *rate.Limiter
	SetLeverate                       *rate.Limiter
	GetMaximumBuyOrSellAmount         *rate.Limiter
	GetMaximumAvailableTradableAmount *rate.Limiter
	IncreaseOrDecreaseMargin          *rate.Limiter
	GetLeverate                       *rate.Limiter
	GetTheMaximumLoanOfInstrument     *rate.Limiter
	GetFeeRates                       *rate.Limiter
	GetInterestAccruedData            *rate.Limiter
	GetInterestRate                   *rate.Limiter
	SetGeeks                          *rate.Limiter
	IsolatedMarginTradingSettings     *rate.Limiter
	GetMaximumWithdrawals             *rate.Limiter
	GetAccountRiskState               *rate.Limiter
	VipLoansBorrowAnsRepay            *rate.Limiter
	GetBorrowAnsRepayHistoryHistory   *rate.Limiter
	GetBorrowInterestAndLimit         *rate.Limiter
	PositionBuilder                   *rate.Limiter
	GetGeeks                          *rate.Limiter
	// Sub Account Endpoints
	ViewSubaccountList                             *rate.Limiter
	GetSubaccountTradingBalance                    *rate.Limiter
	GetSubaccountFundingBalance                    *rate.Limiter
	HistoryOfSubaccountTransfer                    *rate.Limiter
	MasterAccountsManageTransfersBetweenSubaccount *rate.Limiter
	SetPermissingOfTransferOut                     *rate.Limiter
	GetCustoryTradingSubaccountList                *rate.Limiter
	GridTrading                                    *rate.Limiter
	AmendGridAlgoOrder                             *rate.Limiter
	StopGridAlgoOrder                              *rate.Limiter
	GetGridAlgoOrderList                           *rate.Limiter
	GetGridAlgoOrderHistory                        *rate.Limiter
	GetGridAlgoOrderDetails                        *rate.Limiter
	GetGridAlgoSubOrders                           *rate.Limiter
	GetGridAlgoOrderPositions                      *rate.Limiter
	SpotGridWithdrawIncome                         *rate.Limiter
	// Market Data
	GetTickers               *rate.Limiter
	GetIndexTickers          *rate.Limiter
	GetOrderBook             *rate.Limiter
	GetCandlesticks          *rate.Limiter
	GetCandlesticksHistory   *rate.Limiter
	GetIndexCandlesticks     *rate.Limiter
	GetMarkPriceCandlesticks *rate.Limiter
	GetTradesRequest         *rate.Limiter
	GetTradesHistory         *rate.Limiter
	Get24HTotalVolume        *rate.Limiter
	GetOracle                *rate.Limiter
	GetExchangeRateRequest   *rate.Limiter
	GetINdexComponents       *rate.Limiter
	GetBlockTickers          *rate.Limiter
	GetBlockTrades           *rate.Limiter
	// Public Data Endpoints
	GetInstruments                         *rate.Limiter
	GetDeliveryExerciseHistory             *rate.Limiter
	GetOpenInterest                        *rate.Limiter
	GetFunding                             *rate.Limiter
	GetFundingRateHistory                  *rate.Limiter
	GetLimitPrice                          *rate.Limiter
	GetOptionMarketDate                    *rate.Limiter
	GetEstimatedDeliveryExercisePriice     *rate.Limiter
	GetDIscountRateAndInterestFreeQuota    *rate.Limiter
	GetSystemTime                          *rate.Limiter
	GetLiquidationOrders                   *rate.Limiter
	GetMarkPrice                           *rate.Limiter
	GetPositionTiers                       *rate.Limiter
	GetInterestRateAndLoanQuota            *rate.Limiter
	GetInterestRateAndLoanQuoteForVIPLoans *rate.Limiter
	GetUnderlying                          *rate.Limiter
	GetInsuranceFund                       *rate.Limiter
	UnitConvert                            *rate.Limiter
	// Trading Data Endpoints
	GetSupportCoin                    *rate.Limiter
	GetTakerVolume                    *rate.Limiter
	GetMarginLendingRatio             *rate.Limiter
	GetLongShortRatio                 *rate.Limiter
	GetContractsOpeninterestAndVolume *rate.Limiter
	GetOptionsOpenInterestAndVolume   *rate.Limiter
	GetPutCallRatio                   *rate.Limiter
	GetOpenInterestAndVolume          *rate.Limiter
	GetopenInterestAndVolume          *rate.Limiter
	GetTakerFlow                      *rate.Limiter
	// Status Endpoints
	GetEventStatus *rate.Limiter
}

const (
	// Trade Endpoints

	placeOrderRate           = 60
	placeMultipleOrdersRate  = 300
	cancelOrderRate          = 60
	cancelMultipleOrdersRate = 300
	amendOrderRate           = 60
	amendMultipleOrdersRate  = 300
	closeDepositions         = 20
	getOrderDetails          = 60
	getOrderListRate         = 20
	getOrderHistoryRate      = 40
	getTrasactionDetailsRate = 60
	placeAlgoOrderRate       = 20
	cancelAlgoOrderRate      = 20
	// cancelAdvanceAlgoOrderRate = 20
	getAlgoOrderListRate    = 20
	getAlgoOrderhistoryRate = 20

	// Block Trading endpoints

	getCounterpartiesRate    = 5
	createRfqRate            = 5
	cancelRfqRate            = 5
	cancelMultipleRfqRate    = 2
	cancelAllRfqsRate        = 2
	executeQuoteRate         = 2
	createQuoteRate          = 50
	cancelQuoteRate          = 50
	cancelMultipleQuotesRate = 2
	cancelAllQuotes          = 2
	getRfqsRate              = 2
	getQuotesRate            = 2
	getTradesRate            = 5
	getPublicTradesRate      = 5

	// Funding

	getCurrenciesRate            = 6
	getBalanceRate               = 6
	getAccountAssetValuationRate = 1
	fundsTransferRate            = 1
	getFundsTransferStateRate    = 1
	assetBillsDetailsRate        = 6
	ligntningDepositsRate        = 2
	getDepositAddressRate        = 6
	getDepositHistoryRate        = 6
	withdrawalRate               = 6
	lightningWithdrawalsRate     = 2
	cancelWithdrawalRate         = 6
	getWithdrawalHistoryRate     = 6
	smallAssetsConvertRate       = 1
	getSavingBalanceRate         = 6
	savingsPurchaseRedemption    = 6
	setLendingRateRate           = 6
	getLendinghistoryRate        = 6
	getPublicBorrowInfoRate      = 6
	// getPublicBorrowHistoryRate   = 6

	// Convert

	getConvertCurrenciesRate   = 6
	getConvertCurrencyPairRate = 6
	estimateQuoteRate          = 2
	convertTradeRate           = 2
	getConvertHistoryRate      = 6

	// Account

	getAccountBalanceRate                 = 10
	getPositionsRate                      = 10
	getPositionsHistoryRate               = 1
	getAccountAndPositionRiskRate         = 10
	getBillsDetailsRate                   = 6
	getAccountConfigurationRate           = 5
	setPositionModeRate                   = 5
	setLeverateRate                       = 20
	getMaximumBuyOrSellAmountRate         = 20
	getMaximumAvailableTradableAmountRate = 20
	increaseOrDecreaseMarginRate          = 20
	getLeverateRate                       = 20
	getTheMaximumLoanOfInstrumentRate     = 20
	getFeeRatesRate                       = 5
	getInterestAccruedDataRate            = 5
	getInterestRateRate                   = 5
	setGeeksRate                          = 5
	isolatedMarginTradingSettingsRate     = 5
	getMaximumWithdrawalsRate             = 20
	getAccountRiskStateRate               = 10
	vipLoansBorrowAnsRepayRate            = 6
	getBorrowAnsRepayHistoryHistoryRate   = 5
	getBorrowInterestAndLimitRate         = 5
	positionBuilderRate                   = 2
	getGeeksRate                          = 10

	// Sub Account Endpoints

	viewSubaccountListRate                             = 2
	getSubaccountTradingBalanceRate                    = 2
	getSubaccountFundingBalanceRate                    = 2
	historyOfSubaccountTransferRate                    = 6
	masterAccountsManageTransfersBetweenSubaccountRate = 1
	setPermissingOfTransferOutRate                     = 1
	getCustoryTradingSubaccountListRate                = 1
	gridTradingRate                                    = 20
	amendGridAlgoOrderRate                             = 20
	stopGridAlgoOrderRate                              = 20
	getGridAlgoOrderListRate                           = 20
	getGridAlgoOrderHistoryRate                        = 20
	getGridAlgoOrderDetailsRate                        = 20
	getGridAlgoSubOrdersRate                           = 20
	getGridAlgoOrderPositionsRate                      = 20
	spotGridWithdrawIncomeRate                         = 20

	// Market Data

	getTickersRate               = 20
	getIndexTickersRate          = 20
	getOrderBookRate             = 20
	getCandlesticksRate          = 40
	getCandlesticksHistoryRate   = 20
	getIndexCandlesticksRate     = 20
	getMarkPriceCandlesticksRate = 20
	getTradesRequestRate         = 20
	getTradesHistoryRate         = 10
	get24HTotalVolumeRate        = 2
	getOracleRate                = 1
	getExchangeRateRequestRate   = 1
	getINdexComponentsRate       = 20
	getBlockTickersRate          = 20
	getBlockTradesRate           = 20

	// Public Data Endpoints

	getInstrumentsRate                         = 20
	getDeliveryExerciseHistoryRate             = 40
	getOpenInterestRate                        = 20
	getFundingRate                             = 20
	getFundingRateHistoryRate                  = 20
	getLimitPriceRate                          = 20
	getOptionMarketDateRate                    = 20
	getEstimatedDeliveryExercisePriiceRate     = 10
	getDIscountRateAndInterestFreeQuotaRate    = 2
	getSystemTimeRate                          = 10
	getLiquidationOrdersRate                   = 2
	getMarkPriceRate                           = 10
	getPositionTiersRate                       = 10
	getInterestRateAndLoanQuotaRate            = 2
	getInterestRateAndLoanQuoteForVIPLoansRate = 2
	getUnderlyingRate                          = 20
	getInsuranceFundRate                       = 10
	unitConvertRate                            = 10

	// Trading Data Endpoints

	getSupportCoinRate                    = 5
	getTakerVolumeRate                    = 5
	getMarginLendingRatioRate             = 5
	getLongShortRatioRate                 = 5
	getContractsOpeninterestAndVolumeRate = 5
	getOptionsOpenInterestAndVolumeRate   = 5
	getPutCallRatioRate                   = 5
	getOpenInterestAndVolumeRate          = 5
	getopenInterestAndVolumeRate          = 5
	getTakerFlowRate                      = 5

	// Status Endpoints

	getEventStatusRate = 1
)

const (
	placeOrderEPL request.EndpointLimit = iota
	placeMultipleOrdersEPL
	cancelOrderEPL
	cancelMultipleOrdersEPL
	amendOrderEPL
	amendMultipleOrdersEPL
	closePositionEPL
	getOrderDetEPL
	getOrderListEPL
	getOrderHistoryEPL
	getTrasactionDetailsEPL
	placeAlgoOrderEPL
	cancelAlgoOrderEPL
	cancelAdvanceAlgoOrderEPL
	getAlgoOrderListEPL
	getAlgoOrderhistoryEPL
	getCounterpartiesEPL
	createRfqEPL
	cancelRfqEPL
	cancelMultipleRfqEPL
	cancelAllRfqsEPL
	executeQuoteEPL
	createQuoteEPL
	cancelQuoteEPL
	cancelMultipleQuotesEPL
	cancelAllQuotesEPL
	getRfqsEPL
	getQuotesEPL
	getTradesEPL
	getPublicTradesEPL
	getCurrenciesEPL
	getBalanceEPL
	getAccountAssetValuationEPL
	fundsTransferEPL
	getFundsTransferStateEPL
	assetBillsDetailsEPL
	ligntningDepositsEPL
	getDepositAddressEPL
	getDepositHistoryEPL
	withdrawalEPL
	lightningWithdrawalsEPL
	cancelWithdrawalEPL
	getWithdrawalHistoryEPL
	smallAssetsConvertEPL
	getSavingBalanceEPL
	savingsPurchaseRedemptionEPL
	setLendingRateEPL
	getLendinghistoryEPL
	getPublicBorrowInfoEPL
	getPublicBorrowHistoryEPL
	getConvertCurrenciesEPL
	getConvertCurrencyPairEPL
	estimateQuoteEPL
	convertTradeEPL
	getConvertHistoryEPL
	getAccountBalanceEPL
	getPositionsEPL
	getPositionsHistoryEPL
	getAccountAndPositionRiskEPL
	getBillsDetailsEPL
	getAccountConfigurationEPL
	setPositionModeEPL
	setLeverateEPL
	getMaximumBuyOrSellAmountEPL
	getMaximumAvailableTradableAmountEPL
	increaseOrDecreaseMarginEPL
	getLeverateEPL
	getTheMaximumLoanOfInstrumentEPL
	getFeeRatesEPL
	getInterestAccruedDataEPL
	getInterestRateEPL
	setGeeksEPL
	isolatedMarginTradingSettingsEPL
	getMaximumWithdrawalsEPL
	getAccountRiskStateEPL
	vipLoansBorrowAnsRepayEPL
	getBorrowAnsRepayHistoryHistoryEPL
	getBorrowInterestAndLimitEPL
	positionBuilderEPL
	getGeeksEPL
	viewSubaccountListEPL
	getSubaccountTradingBalanceEPL
	getSubaccountFundingBalanceEPL
	historyOfSubaccountTransferEPL
	masterAccountsManageTransfersBetweenSubaccountEPL
	setPermissingOfTransferOutEPL
	getCustoryTradingSubaccountListEPL
	gridTradingEPL
	amendGridAlgoOrderEPL
	stopGridAlgoOrderEPL
	// getGridAlgoOrderListEPL
	getGridAlgoOrderHistoryEPL
	getGridAlgoOrderDetailsEPL
	getGridAlgoSubOrdersEPL
	getGridAlgoOrderPositionsEPL
	spotGridWithdrawIncomeEPL
	getTickersEPL
	getIndexTickersEPL
	getOrderBookEPL
	getCandlesticksEPL
	// getCandlesticksHistoryEPL
	// getIndexCandlesticksEPL
	// getMarkPriceCandlesticksEPL
	getTradesRequestEPL
	// getTradesHistoryEPL
	get24HTotalVolumeEPL
	getOracleEPL
	getExchangeRateRequestEPL
	getINdexComponentsEPL
	getBlockTickersEPL
	getBlockTradesEPL
	getInstrumentsEPL
	getDeliveryExerciseHistoryEPL
	getOpenInterestEPL
	getFundingEPL
	getFundingRateHistoryEPL
	getLimitPriceEPL
	getOptionMarketDateEPL
	getEstimatedDeliveryPriceEPL
	getDiscountRateAndInterestFreeQuotaEPL
	getSystemTimeEPL
	getLiquidationOrdersEPL
	getMarkPriceEPL
	getPositionTiersEPL
	getInterestRateAndLoanQuotaEPL
	getInterestRateAndLoanQuoteForVIPLoansEPL
	getUnderlyingEPL
	getInsuranceFundEPL
	unitConvertEPL
	getSupportCoinEPL
	getTakerVolumeEPL
	getMarginLendingRatioEPL
	getLongShortRatioEPL
	getContractsOpeninterestAndVolumeEPL
	getOptionsOpenInterestAndVolumeEPL
	getPutCallRatioEPL
	getOpenInterestAndVolumeEPL
	getopenInterestAndVolumeEPL
	getTakerFlowEPL
	getEventStatusEPL
)

// Limit executes rate limiting for Okx exchange given the context and EndpointLimit
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) error {
	switch f {
	case placeOrderEPL:
		return r.PlaceOrder.Wait(ctx)
	case placeMultipleOrdersEPL:
		return r.PlaceMultipleOrders.Wait(ctx)
	case cancelOrderEPL:
		return r.CancelOrder.Wait(ctx)
	case cancelMultipleOrdersEPL:
		return r.CancelMultipleOrders.Wait(ctx)
	case amendOrderEPL:
		return r.AmendOrder.Wait(ctx)
	case amendMultipleOrdersEPL:
		return r.AmendMultipleOrders.Wait(ctx)
	case closePositionEPL:
		return r.CloseDposit.Wait(ctx)
	case getOrderDetEPL:
		return r.GetOrderDetails.Wait(ctx)
	case getOrderListEPL:
		return r.GetOrderList.Wait(ctx)
	case getOrderHistoryEPL:
		return r.GetOrderHistory.Wait(ctx)
	case getTrasactionDetailsEPL:
		return r.GetTrasactionDetails.Wait(ctx)
	case placeAlgoOrderEPL:
		return r.PlaceAlgoOrder.Wait(ctx)
	case cancelAlgoOrderEPL:
		return r.CancelAlgoOrder.Wait(ctx)
	case cancelAdvanceAlgoOrderEPL:
		return r.CancelAdvanceAlgoOrder.Wait(ctx)
	case getAlgoOrderListEPL:
		return r.GetAlgoOrderList.Wait(ctx)
	case getAlgoOrderhistoryEPL:
		return r.GetAlgoOrderhistory.Wait(ctx)
	case getCounterpartiesEPL:
		return r.GetCounterparties.Wait(ctx)
	case createRfqEPL:
		return r.CreateRfq.Wait(ctx)
	case cancelRfqEPL:
		return r.CancelRfq.Wait(ctx)
	case cancelMultipleRfqEPL:
		return r.CancelMultipleRfq.Wait(ctx)
	case cancelAllRfqsEPL:
		return r.CancelAllRfqs.Wait(ctx)
	case executeQuoteEPL:
		return r.ExecuteQuote.Wait(ctx)
	case createQuoteEPL:
		return r.CreateQuote.Wait(ctx)
	case cancelQuoteEPL:
		return r.CancelQuote.Wait(ctx)
	case cancelMultipleQuotesEPL:
		return r.CancelMultipleQuotes.Wait(ctx)
	case cancelAllQuotesEPL:
		return r.CancelAllQuotes.Wait(ctx)
	case getRfqsEPL:
		return r.GetRfqs.Wait(ctx)
	case getQuotesEPL:
		return r.GetQuotes.Wait(ctx)
	case getTradesEPL:
		return r.GetTrades.Wait(ctx)
	case getPublicTradesEPL:
		return r.GetPublicTrades.Wait(ctx)
	case getCurrenciesEPL:
		return r.GetCurrencies.Wait(ctx)
	case getBalanceEPL:
		return r.GetBalance.Wait(ctx)
	case getAccountAssetValuationEPL:
		return r.GetAccountAssetValuation.Wait(ctx)
	case fundsTransferEPL:
		return r.FundsTransfer.Wait(ctx)
	case getFundsTransferStateEPL:
		return r.GetFundsTransferState.Wait(ctx)
	case assetBillsDetailsEPL:
		return r.AssetBillsDetails.Wait(ctx)
	case ligntningDepositsEPL:
		return r.LigntningDeposits.Wait(ctx)
	case getDepositAddressEPL:
		return r.GetDepositAddress.Wait(ctx)
	case getDepositHistoryEPL:
		return r.GetDepositHistory.Wait(ctx)
	case withdrawalEPL:
		return r.Withdrawal.Wait(ctx)
	case lightningWithdrawalsEPL:
		return r.LightningWithdrawals.Wait(ctx)
	case cancelWithdrawalEPL:
		return r.CancelWithdrawal.Wait(ctx)
	case getWithdrawalHistoryEPL:
		return r.GetWithdrawalHistory.Wait(ctx)
	case smallAssetsConvertEPL:
		return r.SmallAssetsConvert.Wait(ctx)
	case getSavingBalanceEPL:
		return r.GetSavingBalance.Wait(ctx)
	case savingsPurchaseRedemptionEPL:
		return r.SavingsPurchaseRedemp.Wait(ctx)
	case setLendingRateEPL:
		return r.SetLendingRate.Wait(ctx)
	case getLendinghistoryEPL:
		return r.GetLendinghistory.Wait(ctx)
	case getPublicBorrowInfoEPL:
		return r.GetPublicBorrowInfo.Wait(ctx)
	case getPublicBorrowHistoryEPL:
		return r.GetPublicBorrowHistory.Wait(ctx)
	case getConvertCurrenciesEPL:
		return r.GetConvertCurrencies.Wait(ctx)
	case getConvertCurrencyPairEPL:
		return r.GetConvertCurrencyPair.Wait(ctx)
	case estimateQuoteEPL:
		return r.EstimateQuote.Wait(ctx)
	case convertTradeEPL:
		return r.ConvertTrade.Wait(ctx)
	case getConvertHistoryEPL:
		return r.GetConvertHistory.Wait(ctx)
	case getAccountBalanceEPL:
		return r.GetAccountBalance.Wait(ctx)
	case getPositionsEPL:
		return r.GetPositions.Wait(ctx)
	case getPositionsHistoryEPL:
		return r.GetPositionsHistory.Wait(ctx)
	case getAccountAndPositionRiskEPL:
		return r.GetAccountAndPositionRisk.Wait(ctx)
	case getBillsDetailsEPL:
		return r.GetBillsDetails.Wait(ctx)
	case getAccountConfigurationEPL:
		return r.GetAccountConfiguration.Wait(ctx)
	case setPositionModeEPL:
		return r.SetPositionMode.Wait(ctx)
	case setLeverateEPL:
		return r.SetLeverate.Wait(ctx)
	case getMaximumBuyOrSellAmountEPL:
		return r.GetMaximumBuyOrSellAmount.Wait(ctx)
	case getMaximumAvailableTradableAmountEPL:
		return r.GetMaximumAvailableTradableAmount.Wait(ctx)
	case increaseOrDecreaseMarginEPL:
		return r.IncreaseOrDecreaseMargin.Wait(ctx)
	case getLeverateEPL:
		return r.GetLeverate.Wait(ctx)
	case getTheMaximumLoanOfInstrumentEPL:
		return r.GetTheMaximumLoanOfInstrument.Wait(ctx)
	case getFeeRatesEPL:
		return r.GetFeeRates.Wait(ctx)
	case getInterestAccruedDataEPL:
		return r.GetInterestAccruedData.Wait(ctx)
	case getInterestRateEPL:
		return r.GetInterestRate.Wait(ctx)
	case setGeeksEPL:
		return r.SetGeeks.Wait(ctx)
	case isolatedMarginTradingSettingsEPL:
		return r.IsolatedMarginTradingSettings.Wait(ctx)
	case getMaximumWithdrawalsEPL:
		return r.GetMaximumWithdrawals.Wait(ctx)
	case getAccountRiskStateEPL:
		return r.GetAccountRiskState.Wait(ctx)
	case vipLoansBorrowAnsRepayEPL:
		return r.VipLoansBorrowAnsRepay.Wait(ctx)
	case getBorrowAnsRepayHistoryHistoryEPL:
		return r.GetBorrowAnsRepayHistoryHistory.Wait(ctx)
	case getBorrowInterestAndLimitEPL:
		return r.GetBorrowInterestAndLimit.Wait(ctx)
	case positionBuilderEPL:
		return r.PositionBuilder.Wait(ctx)
	case getGeeksEPL:
		return r.GetGeeks.Wait(ctx)
	case viewSubaccountListEPL:
		return r.ViewSubaccountList.Wait(ctx)
	case getSubaccountTradingBalanceEPL:
		return r.GetSubaccountTradingBalance.Wait(ctx)
	case getSubaccountFundingBalanceEPL:
		return r.GetSubaccountFundingBalance.Wait(ctx)
	case historyOfSubaccountTransferEPL:
		return r.HistoryOfSubaccountTransfer.Wait(ctx)
	case masterAccountsManageTransfersBetweenSubaccountEPL:
		return r.MasterAccountsManageTransfersBetweenSubaccount.Wait(ctx)
	case setPermissingOfTransferOutEPL:
		return r.SetPermissingOfTransferOut.Wait(ctx)
	case getCustoryTradingSubaccountListEPL:
		return r.GetCustoryTradingSubaccountList.Wait(ctx)
	case gridTradingEPL:
		return r.GridTrading.Wait(ctx)
	case amendGridAlgoOrderEPL:
		return r.AmendGridAlgoOrder.Wait(ctx)
	case stopGridAlgoOrderEPL:
		return r.StopGridAlgoOrder.Wait(ctx)
	case getGridAlgoOrderHistoryEPL:
		return r.GetGridAlgoOrderHistory.Wait(ctx)
	case getGridAlgoOrderDetailsEPL:
		return r.GetGridAlgoOrderDetails.Wait(ctx)
	case getGridAlgoSubOrdersEPL:
		return r.GetGridAlgoSubOrders.Wait(ctx)
	case getGridAlgoOrderPositionsEPL:
		return r.GetGridAlgoOrderPositions.Wait(ctx)
	case spotGridWithdrawIncomeEPL:
		return r.SpotGridWithdrawIncome.Wait(ctx)
	case getTickersEPL:
		return r.GetTickers.Wait(ctx)
	case getIndexTickersEPL:
		return r.GetIndexTickers.Wait(ctx)
	case getOrderBookEPL:
		return r.GetOrderBook.Wait(ctx)
	case getCandlesticksEPL:
		return r.GetCandlesticks.Wait(ctx)
	case getTradesRequestEPL:
		return r.GetTradesRequest.Wait(ctx)
	case get24HTotalVolumeEPL:
		return r.Get24HTotalVolume.Wait(ctx)
	case getOracleEPL:
		return r.GetOracle.Wait(ctx)
	case getExchangeRateRequestEPL:
		return r.GetExchangeRateRequest.Wait(ctx)
	case getINdexComponentsEPL:
		return r.GetINdexComponents.Wait(ctx)
	case getBlockTickersEPL:
		return r.GetBlockTickers.Wait(ctx)
	case getBlockTradesEPL:
		return r.GetBlockTrades.Wait(ctx)
	case getInstrumentsEPL:
		return r.GetInstruments.Wait(ctx)
	case getDeliveryExerciseHistoryEPL:
		return r.GetDeliveryExerciseHistory.Wait(ctx)
	case getOpenInterestEPL:
		return r.GetOpenInterest.Wait(ctx)
	case getFundingEPL:
		return r.GetFunding.Wait(ctx)
	case getFundingRateHistoryEPL:
		return r.GetFundingRateHistory.Wait(ctx)
	case getLimitPriceEPL:
		return r.GetLimitPrice.Wait(ctx)
	case getOptionMarketDateEPL:
		return r.GetOptionMarketDate.Wait(ctx)
	case getEstimatedDeliveryPriceEPL:
		return r.GetEstimatedDeliveryExercisePriice.Wait(ctx)
	case getDiscountRateAndInterestFreeQuotaEPL:
		return r.GetDIscountRateAndInterestFreeQuota.Wait(ctx)
	case getSystemTimeEPL:
		return r.GetSystemTime.Wait(ctx)
	case getLiquidationOrdersEPL:
		return r.GetLiquidationOrders.Wait(ctx)
	case getMarkPriceEPL:
		return r.GetMarkPrice.Wait(ctx)
	case getPositionTiersEPL:
		return r.GetPositionTiers.Wait(ctx)
	case getInterestRateAndLoanQuotaEPL:
		return r.GetInterestRateAndLoanQuota.Wait(ctx)
	case getInterestRateAndLoanQuoteForVIPLoansEPL:
		return r.GetInterestRateAndLoanQuoteForVIPLoans.Wait(ctx)
	case getUnderlyingEPL:
		return r.GetUnderlying.Wait(ctx)
	case getInsuranceFundEPL:
		return r.GetInsuranceFund.Wait(ctx)
	case unitConvertEPL:
		return r.UnitConvert.Wait(ctx)
	case getSupportCoinEPL:
		return r.GetSupportCoin.Wait(ctx)
	case getTakerVolumeEPL:
		return r.GetTakerVolume.Wait(ctx)
	case getMarginLendingRatioEPL:
		return r.GetMarginLendingRatio.Wait(ctx)
	case getLongShortRatioEPL:
		return r.GetLongShortRatio.Wait(ctx)
	case getContractsOpeninterestAndVolumeEPL:
		return r.GetContractsOpeninterestAndVolume.Wait(ctx)
	case getOptionsOpenInterestAndVolumeEPL:
		return r.GetOptionsOpenInterestAndVolume.Wait(ctx)
	case getPutCallRatioEPL:
		return r.GetPutCallRatio.Wait(ctx)
	case getOpenInterestAndVolumeEPL:
		return r.GetOpenInterestAndVolume.Wait(ctx)
	case getopenInterestAndVolumeEPL:
		return r.GetopenInterestAndVolume.Wait(ctx)
	case getTakerFlowEPL:
		return r.GetTakerFlow.Wait(ctx)
	case getEventStatusEPL:
		return r.GetEventStatus.Wait(ctx)
	default:
		return errors.New("endpoint rate limit functionality not found")
	}
}

// SetRateLimit returns a RateLimit instance, which implements the request.Limiter interface.
func SetRateLimit() *RateLimit {
	return &RateLimit{
		// Trade Endpoints
		PlaceOrder:           request.NewRateLimit(twoSecondsInterval, placeOrderRate),
		PlaceMultipleOrders:  request.NewRateLimit(twoSecondsInterval, placeMultipleOrdersRate),
		CancelOrder:          request.NewRateLimit(twoSecondsInterval, cancelOrderRate),
		CancelMultipleOrders: request.NewRateLimit(twoSecondsInterval, cancelMultipleOrdersRate),
		AmendOrder:           request.NewRateLimit(twoSecondsInterval, amendOrderRate),
		AmendMultipleOrders:  request.NewRateLimit(twoSecondsInterval, amendMultipleOrdersRate),
		CloseDposit:          request.NewRateLimit(twoSecondsInterval, closeDepositions),
		GetOrderDetails:      request.NewRateLimit(twoSecondsInterval, getOrderDetails),
		GetOrderList:         request.NewRateLimit(twoSecondsInterval, getOrderListRate),
		GetOrderHistory:      request.NewRateLimit(twoSecondsInterval, getOrderHistoryRate),
		GetTrasactionDetails: request.NewRateLimit(twoSecondsInterval, getTrasactionDetailsRate),
		PlaceAlgoOrder:       request.NewRateLimit(twoSecondsInterval, placeAlgoOrderRate),
		CancelAlgoOrder:      request.NewRateLimit(twoSecondsInterval, cancelAlgoOrderRate),
		GetAlgoOrderList:     request.NewRateLimit(twoSecondsInterval, getAlgoOrderListRate),
		GetAlgoOrderhistory:  request.NewRateLimit(twoSecondsInterval, getAlgoOrderhistoryRate),
		// Block Trading endpoints
		GetCounterparties:    request.NewRateLimit(twoSecondsInterval, getCounterpartiesRate),
		CreateRfq:            request.NewRateLimit(twoSecondsInterval, createRfqRate),
		CancelRfq:            request.NewRateLimit(twoSecondsInterval, cancelRfqRate),
		CancelMultipleRfq:    request.NewRateLimit(twoSecondsInterval, cancelMultipleRfqRate),
		CancelAllRfqs:        request.NewRateLimit(twoSecondsInterval, cancelAllRfqsRate),
		ExecuteQuote:         request.NewRateLimit(threeSecondsInterval, executeQuoteRate),
		CreateQuote:          request.NewRateLimit(twoSecondsInterval, createQuoteRate),
		CancelQuote:          request.NewRateLimit(twoSecondsInterval, cancelQuoteRate),
		CancelMultipleQuotes: request.NewRateLimit(twoSecondsInterval, cancelMultipleQuotesRate),
		CancelAllQuotes:      request.NewRateLimit(twoSecondsInterval, cancelAllQuotes),
		GetRfqs:              request.NewRateLimit(twoSecondsInterval, getRfqsRate),
		GetQuotes:            request.NewRateLimit(twoSecondsInterval, getQuotesRate),
		GetTrades:            request.NewRateLimit(twoSecondsInterval, getTradesRate),
		GetPublicTrades:      request.NewRateLimit(twoSecondsInterval, getPublicTradesRate),
		// Funding
		GetCurrencies:            request.NewRateLimit(oneSecondInterval, getCurrenciesRate),
		GetBalance:               request.NewRateLimit(oneSecondInterval, getBalanceRate),
		GetAccountAssetValuation: request.NewRateLimit(oneSecondInterval, getAccountAssetValuationRate),
		FundsTransfer:            request.NewRateLimit(oneSecondInterval, fundsTransferRate),
		GetFundsTransferState:    request.NewRateLimit(oneSecondInterval, getFundsTransferStateRate),
		AssetBillsDetails:        request.NewRateLimit(oneSecondInterval, assetBillsDetailsRate),
		LigntningDeposits:        request.NewRateLimit(oneSecondInterval, ligntningDepositsRate),
		GetDepositAddress:        request.NewRateLimit(oneSecondInterval, getDepositAddressRate),
		GetDepositHistory:        request.NewRateLimit(oneSecondInterval, getDepositHistoryRate),
		Withdrawal:               request.NewRateLimit(oneSecondInterval, withdrawalRate),
		LightningWithdrawals:     request.NewRateLimit(oneSecondInterval, lightningWithdrawalsRate),
		CancelWithdrawal:         request.NewRateLimit(oneSecondInterval, cancelWithdrawalRate),
		GetWithdrawalHistory:     request.NewRateLimit(oneSecondInterval, getWithdrawalHistoryRate),
		SmallAssetsConvert:       request.NewRateLimit(oneSecondInterval, smallAssetsConvertRate),
		GetSavingBalance:         request.NewRateLimit(oneSecondInterval, getSavingBalanceRate),
		SavingsPurchaseRedemp:    request.NewRateLimit(oneSecondInterval, savingsPurchaseRedemption),
		SetLendingRate:           request.NewRateLimit(oneSecondInterval, setLendingRateRate),
		GetLendinghistory:        request.NewRateLimit(oneSecondInterval, getLendinghistoryRate),
		GetPublicBorrowInfo:      request.NewRateLimit(oneSecondInterval, getPublicBorrowInfoRate),
		// Convert
		GetConvertCurrencies:   request.NewRateLimit(oneSecondInterval, getConvertCurrenciesRate),
		GetConvertCurrencyPair: request.NewRateLimit(oneSecondInterval, getConvertCurrencyPairRate),
		EstimateQuote:          request.NewRateLimit(oneSecondInterval, estimateQuoteRate),
		ConvertTrade:           request.NewRateLimit(oneSecondInterval, convertTradeRate),
		GetConvertHistory:      request.NewRateLimit(oneSecondInterval, getConvertHistoryRate),

		// Account

		GetAccountBalance:                 request.NewRateLimit(twoSecondsInterval, getAccountBalanceRate),
		GetPositions:                      request.NewRateLimit(twoSecondsInterval, getPositionsRate),
		GetPositionsHistory:               request.NewRateLimit(tenSecondsInterval, getPositionsHistoryRate),
		GetAccountAndPositionRisk:         request.NewRateLimit(twoSecondsInterval, getAccountAndPositionRiskRate),
		GetBillsDetails:                   request.NewRateLimit(oneSecondInterval, getBillsDetailsRate),
		GetAccountConfiguration:           request.NewRateLimit(twoSecondsInterval, getAccountConfigurationRate),
		SetPositionMode:                   request.NewRateLimit(twoSecondsInterval, setPositionModeRate),
		SetLeverate:                       request.NewRateLimit(twoSecondsInterval, setLeverateRate),
		GetMaximumBuyOrSellAmount:         request.NewRateLimit(twoSecondsInterval, getMaximumBuyOrSellAmountRate),
		GetMaximumAvailableTradableAmount: request.NewRateLimit(twoSecondsInterval, getMaximumAvailableTradableAmountRate),
		IncreaseOrDecreaseMargin:          request.NewRateLimit(twoSecondsInterval, increaseOrDecreaseMarginRate),
		GetLeverate:                       request.NewRateLimit(twoSecondsInterval, getLeverateRate),
		GetTheMaximumLoanOfInstrument:     request.NewRateLimit(twoSecondsInterval, getTheMaximumLoanOfInstrumentRate),
		GetFeeRates:                       request.NewRateLimit(twoSecondsInterval, getFeeRatesRate),
		GetInterestAccruedData:            request.NewRateLimit(twoSecondsInterval, getInterestAccruedDataRate),
		GetInterestRate:                   request.NewRateLimit(twoSecondsInterval, getInterestRateRate),
		SetGeeks:                          request.NewRateLimit(twoSecondsInterval, setGeeksRate),
		IsolatedMarginTradingSettings:     request.NewRateLimit(twoSecondsInterval, isolatedMarginTradingSettingsRate),
		GetMaximumWithdrawals:             request.NewRateLimit(twoSecondsInterval, getMaximumWithdrawalsRate),
		GetAccountRiskState:               request.NewRateLimit(twoSecondsInterval, getAccountRiskStateRate),
		VipLoansBorrowAnsRepay:            request.NewRateLimit(oneSecondInterval, vipLoansBorrowAnsRepayRate),
		GetBorrowAnsRepayHistoryHistory:   request.NewRateLimit(twoSecondsInterval, getBorrowAnsRepayHistoryHistoryRate),
		GetBorrowInterestAndLimit:         request.NewRateLimit(twoSecondsInterval, getBorrowInterestAndLimitRate),
		PositionBuilder:                   request.NewRateLimit(twoSecondsInterval, positionBuilderRate),
		GetGeeks:                          request.NewRateLimit(twoSecondsInterval, getGeeksRate),

		// Sub Account Endpoints

		ViewSubaccountList:                             request.NewRateLimit(twoSecondsInterval, viewSubaccountListRate),
		GetSubaccountTradingBalance:                    request.NewRateLimit(twoSecondsInterval, getSubaccountTradingBalanceRate),
		GetSubaccountFundingBalance:                    request.NewRateLimit(twoSecondsInterval, getSubaccountFundingBalanceRate),
		HistoryOfSubaccountTransfer:                    request.NewRateLimit(oneSecondInterval, historyOfSubaccountTransferRate),
		MasterAccountsManageTransfersBetweenSubaccount: request.NewRateLimit(oneSecondInterval, masterAccountsManageTransfersBetweenSubaccountRate),
		SetPermissingOfTransferOut:                     request.NewRateLimit(oneSecondInterval, setPermissingOfTransferOutRate),
		GetCustoryTradingSubaccountList:                request.NewRateLimit(oneSecondInterval, getCustoryTradingSubaccountListRate),
		GridTrading:                                    request.NewRateLimit(twoSecondsInterval, gridTradingRate),
		AmendGridAlgoOrder:                             request.NewRateLimit(twoSecondsInterval, amendGridAlgoOrderRate),
		StopGridAlgoOrder:                              request.NewRateLimit(twoSecondsInterval, stopGridAlgoOrderRate),
		GetGridAlgoOrderList:                           request.NewRateLimit(twoSecondsInterval, getGridAlgoOrderListRate),
		GetGridAlgoOrderHistory:                        request.NewRateLimit(twoSecondsInterval, getGridAlgoOrderHistoryRate),
		GetGridAlgoOrderDetails:                        request.NewRateLimit(twoSecondsInterval, getGridAlgoOrderDetailsRate),
		GetGridAlgoSubOrders:                           request.NewRateLimit(twoSecondsInterval, getGridAlgoSubOrdersRate),
		GetGridAlgoOrderPositions:                      request.NewRateLimit(twoSecondsInterval, getGridAlgoOrderPositionsRate),
		SpotGridWithdrawIncome:                         request.NewRateLimit(twoSecondsInterval, spotGridWithdrawIncomeRate),

		// Market Data

		GetTickers:               request.NewRateLimit(twoSecondsInterval, getTickersRate),
		GetIndexTickers:          request.NewRateLimit(twoSecondsInterval, getIndexTickersRate),
		GetOrderBook:             request.NewRateLimit(twoSecondsInterval, getOrderBookRate),
		GetCandlesticks:          request.NewRateLimit(twoSecondsInterval, getCandlesticksRate),
		GetCandlesticksHistory:   request.NewRateLimit(twoSecondsInterval, getCandlesticksHistoryRate),
		GetIndexCandlesticks:     request.NewRateLimit(twoSecondsInterval, getIndexCandlesticksRate),
		GetMarkPriceCandlesticks: request.NewRateLimit(twoSecondsInterval, getMarkPriceCandlesticksRate),
		GetTradesRequest:         request.NewRateLimit(twoSecondsInterval, getTradesRequestRate),
		GetTradesHistory:         request.NewRateLimit(twoSecondsInterval, getTradesHistoryRate),
		Get24HTotalVolume:        request.NewRateLimit(twoSecondsInterval, get24HTotalVolumeRate),
		GetOracle:                request.NewRateLimit(fiveSecondsInterval, getOracleRate),
		GetExchangeRateRequest:   request.NewRateLimit(twoSecondsInterval, getExchangeRateRequestRate),
		GetINdexComponents:       request.NewRateLimit(twoSecondsInterval, getINdexComponentsRate),
		GetBlockTickers:          request.NewRateLimit(twoSecondsInterval, getBlockTickersRate),
		GetBlockTrades:           request.NewRateLimit(twoSecondsInterval, getBlockTradesRate),

		// Public Data Endpoints

		GetInstruments:                         request.NewRateLimit(twoSecondsInterval, getInstrumentsRate),
		GetDeliveryExerciseHistory:             request.NewRateLimit(twoSecondsInterval, getDeliveryExerciseHistoryRate),
		GetOpenInterest:                        request.NewRateLimit(twoSecondsInterval, getOpenInterestRate),
		GetFunding:                             request.NewRateLimit(twoSecondsInterval, getFundingRate),
		GetFundingRateHistory:                  request.NewRateLimit(twoSecondsInterval, getFundingRateHistoryRate),
		GetLimitPrice:                          request.NewRateLimit(twoSecondsInterval, getLimitPriceRate),
		GetOptionMarketDate:                    request.NewRateLimit(twoSecondsInterval, getOptionMarketDateRate),
		GetEstimatedDeliveryExercisePriice:     request.NewRateLimit(twoSecondsInterval, getEstimatedDeliveryExercisePriiceRate),
		GetDIscountRateAndInterestFreeQuota:    request.NewRateLimit(twoSecondsInterval, getDIscountRateAndInterestFreeQuotaRate),
		GetSystemTime:                          request.NewRateLimit(twoSecondsInterval, getSystemTimeRate),
		GetLiquidationOrders:                   request.NewRateLimit(twoSecondsInterval, getLiquidationOrdersRate),
		GetMarkPrice:                           request.NewRateLimit(twoSecondsInterval, getMarkPriceRate),
		GetPositionTiers:                       request.NewRateLimit(twoSecondsInterval, getPositionTiersRate),
		GetInterestRateAndLoanQuota:            request.NewRateLimit(twoSecondsInterval, getInterestRateAndLoanQuotaRate),
		GetInterestRateAndLoanQuoteForVIPLoans: request.NewRateLimit(twoSecondsInterval, getInterestRateAndLoanQuoteForVIPLoansRate),
		GetUnderlying:                          request.NewRateLimit(twoSecondsInterval, getUnderlyingRate),
		GetInsuranceFund:                       request.NewRateLimit(twoSecondsInterval, getInsuranceFundRate),
		UnitConvert:                            request.NewRateLimit(twoSecondsInterval, unitConvertRate),

		// Trading Data Endpoints

		GetSupportCoin:                    request.NewRateLimit(twoSecondsInterval, getSupportCoinRate),
		GetTakerVolume:                    request.NewRateLimit(twoSecondsInterval, getTakerVolumeRate),
		GetMarginLendingRatio:             request.NewRateLimit(twoSecondsInterval, getMarginLendingRatioRate),
		GetLongShortRatio:                 request.NewRateLimit(twoSecondsInterval, getLongShortRatioRate),
		GetContractsOpeninterestAndVolume: request.NewRateLimit(twoSecondsInterval, getContractsOpeninterestAndVolumeRate),
		GetOptionsOpenInterestAndVolume:   request.NewRateLimit(twoSecondsInterval, getOptionsOpenInterestAndVolumeRate),
		GetPutCallRatio:                   request.NewRateLimit(twoSecondsInterval, getPutCallRatioRate),
		GetOpenInterestAndVolume:          request.NewRateLimit(twoSecondsInterval, getOpenInterestAndVolumeRate),
		GetopenInterestAndVolume:          request.NewRateLimit(twoSecondsInterval, getopenInterestAndVolumeRate),
		GetTakerFlow:                      request.NewRateLimit(twoSecondsInterval, getTakerFlowRate),

		// Status Endpoints

		GetEventStatus: request.NewRateLimit(fiveSecondsInterval, getEventStatusRate),
	}
}
