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
	PlaceOrder                  *rate.Limiter
	PlaceMultipleOrders         *rate.Limiter
	CancelOrder                 *rate.Limiter
	CancelMultipleOrders        *rate.Limiter
	AmendOrder                  *rate.Limiter
	AmendMultipleOrders         *rate.Limiter
	CloseDeposit                *rate.Limiter
	GetOrderDetails             *rate.Limiter
	GetOrderList                *rate.Limiter
	GetOrderHistory7Days        *rate.Limiter
	GetOrderHistory3Months      *rate.Limiter
	GetTransactionDetail3Days   *rate.Limiter
	GetTransactionDetail3Months *rate.Limiter
	PlaceAlgoOrder              *rate.Limiter
	CancelAlgoOrder             *rate.Limiter
	CancelAdvanceAlgoOrder      *rate.Limiter
	GetAlgoOrderList            *rate.Limiter
	GetAlgoOrderHistory         *rate.Limiter
	GetEasyConvertCurrencyList  *rate.Limiter
	PlaceEasyConvert            *rate.Limiter
	GetEasyConvertHistory       *rate.Limiter
	GetOneClickRepayHistory     *rate.Limiter
	OneClickRepayCurrencyList   *rate.Limiter
	TradeOneClickRepay          *rate.Limiter
	// Block Trading endpoints
	GetCounterparties    *rate.Limiter
	CreateRfq            *rate.Limiter
	CancelRfq            *rate.Limiter
	CancelMultipleRfq    *rate.Limiter
	CancelAllRfqs        *rate.Limiter
	ExecuteQuote         *rate.Limiter
	SetQuoteProducts     *rate.Limiter
	RestMMPStatus        *rate.Limiter
	CreateQuote          *rate.Limiter
	CancelQuote          *rate.Limiter
	CancelMultipleQuotes *rate.Limiter
	CancelAllQuotes      *rate.Limiter
	GetRfqs              *rate.Limiter
	GetQuotes            *rate.Limiter
	GetTrades            *rate.Limiter
	GetTradesHistory     *rate.Limiter
	GetPublicTrades      *rate.Limiter
	// Funding
	GetCurrencies            *rate.Limiter
	GetBalance               *rate.Limiter
	GetAccountAssetValuation *rate.Limiter
	FundsTransfer            *rate.Limiter
	GetFundsTransferState    *rate.Limiter
	AssetBillsDetails        *rate.Limiter
	LightningDeposits        *rate.Limiter
	GetDepositAddress        *rate.Limiter
	GetDepositHistory        *rate.Limiter
	Withdrawal               *rate.Limiter
	LightningWithdrawals     *rate.Limiter
	CancelWithdrawal         *rate.Limiter
	GetWithdrawalHistory     *rate.Limiter
	SmallAssetsConvert       *rate.Limiter
	// Savings
	GetSavingBalance       *rate.Limiter
	SavingsPurchaseRedempt *rate.Limiter
	SetLendingRate         *rate.Limiter
	GetLendingHistory      *rate.Limiter
	GetPublicBorrowInfo    *rate.Limiter
	GetPublicBorrowHistory *rate.Limiter
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
	SetLeverage                       *rate.Limiter
	GetMaximumBuyOrSellAmount         *rate.Limiter
	GetMaximumAvailableTradableAmount *rate.Limiter
	IncreaseOrDecreaseMargin          *rate.Limiter
	GetLeverage                       *rate.Limiter
	GetTheMaximumLoanOfInstrument     *rate.Limiter
	GetFeeRates                       *rate.Limiter
	GetInterestAccruedData            *rate.Limiter
	GetInterestRate                   *rate.Limiter
	SetGreeks                         *rate.Limiter
	IsolatedMarginTradingSettings     *rate.Limiter
	GetMaximumWithdrawals             *rate.Limiter
	GetAccountRiskState               *rate.Limiter
	VipLoansBorrowAnsRepay            *rate.Limiter
	GetBorrowAnsRepayHistoryHistory   *rate.Limiter
	GetBorrowInterestAndLimit         *rate.Limiter
	PositionBuilder                   *rate.Limiter
	GetGreeks                         *rate.Limiter
	GetPMLimitation                   *rate.Limiter
	// Sub Account Endpoints
	ViewSubaccountList                             *rate.Limiter
	ResetSubAccountAPIKey                          *rate.Limiter
	GetSubaccountTradingBalance                    *rate.Limiter
	GetSubaccountFundingBalance                    *rate.Limiter
	HistoryOfSubaccountTransfer                    *rate.Limiter
	MasterAccountsManageTransfersBetweenSubaccount *rate.Limiter
	SetPermissionOfTransferOut                     *rate.Limiter
	GetCustodyTradingSubaccountList                *rate.Limiter
	GridTrading                                    *rate.Limiter
	AmendGridAlgoOrder                             *rate.Limiter
	StopGridAlgoOrder                              *rate.Limiter
	GetGridAlgoOrderList                           *rate.Limiter
	GetGridAlgoOrderHistory                        *rate.Limiter
	GetGridAlgoOrderDetails                        *rate.Limiter
	GetGridAlgoSubOrders                           *rate.Limiter
	GetGridAlgoOrderPositions                      *rate.Limiter
	SpotGridWithdrawIncome                         *rate.Limiter
	ComputeMarginBalance                           *rate.Limiter
	AdjustMarginBalance                            *rate.Limiter
	GetGridAIParameter                             *rate.Limiter
	// Earn
	GetOffer                   *rate.Limiter
	Purchase                   *rate.Limiter
	Redeem                     *rate.Limiter
	CancelPurchaseOrRedemption *rate.Limiter
	GetEarnActiveOrders        *rate.Limiter
	GetFundingOrderHistory     *rate.Limiter
	// Market Data
	GetTickers               *rate.Limiter
	GetIndexTickers          *rate.Limiter
	GetOrderBook             *rate.Limiter
	GetCandlesticks          *rate.Limiter
	GetCandlesticksHistory   *rate.Limiter
	GetIndexCandlesticks     *rate.Limiter
	GetMarkPriceCandlesticks *rate.Limiter
	GetTradesRequest         *rate.Limiter
	Get24HTotalVolume        *rate.Limiter
	GetOracle                *rate.Limiter
	GetExchangeRateRequest   *rate.Limiter
	GetIndexComponents       *rate.Limiter
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
	GetEstimatedDeliveryExercisePrice      *rate.Limiter
	GetDiscountRateAndInterestFreeQuota    *rate.Limiter
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
	GetContractsOpenInterestAndVolume *rate.Limiter
	GetOptionsOpenInterestAndVolume   *rate.Limiter
	GetPutCallRatio                   *rate.Limiter
	GetOpenInterestAndVolume          *rate.Limiter
	GetTakerFlow                      *rate.Limiter
	// Status Endpoints
	GetEventStatus *rate.Limiter
}

const (
	// Trade Endpoints
	placeOrderRate                  = 60
	placeMultipleOrdersRate         = 300
	cancelOrderRate                 = 60
	cancelMultipleOrdersRate        = 300
	amendOrderRate                  = 60
	amendMultipleOrdersRate         = 300
	closeDepositions                = 20
	getOrderDetails                 = 60
	getOrderListRate                = 60
	getOrderHistory7DaysRate        = 40
	getOrderHistory3MonthsRate      = 20
	getTransactionDetail3DaysRate   = 60
	getTransactionDetail3MonthsRate = 10
	placeAlgoOrderRate              = 20
	cancelAlgoOrderRate             = 20
	cancelAdvanceAlgoOrderRate      = 20
	getAlgoOrderListRate            = 20
	getAlgoOrderHistoryRate         = 20
	getEasyConvertCurrencyListRate  = 1
	placeEasyConvert                = 1
	getEasyConvertHistory           = 1
	oneClickRepayCurrencyList       = 1
	tradeOneClickRepay              = 1
	getOneClickRepayHistory         = 1
	// Block Trading endpoints
	getCounterpartiesRate    = 5
	createRfqRate            = 5
	cancelRfqRate            = 5
	cancelMultipleRfqRate    = 2
	cancelAllRfqsRate        = 2
	executeQuoteRate         = 2
	setQuoteProducts         = 5
	restMMPStatus            = 5
	createQuoteRate          = 50
	cancelQuoteRate          = 50
	cancelMultipleQuotesRate = 2
	cancelAllQuotes          = 2
	getRfqsRate              = 2
	getQuotesRate            = 2
	getTradesRate            = 5
	getTradesHistoryRate     = 10
	getPublicTradesRate      = 5
	// Funding
	getCurrenciesRate            = 6
	getBalanceRate               = 6
	getAccountAssetValuationRate = 1
	fundsTransferRate            = 1
	getFundsTransferStateRate    = 1
	assetBillsDetailsRate        = 6
	lightningDepositsRate        = 2
	getDepositAddressRate        = 6
	getDepositHistoryRate        = 6
	withdrawalRate               = 6
	lightningWithdrawalsRate     = 2
	cancelWithdrawalRate         = 6
	getWithdrawalHistoryRate     = 6
	smallAssetsConvertRate       = 1
	// Savings
	getSavingBalanceRate          = 6
	savingsPurchaseRedemptionRate = 6
	setLendingRateRate            = 6
	getLendingHistoryRate         = 6
	getPublicBorrowInfoRate       = 6
	getPublicBorrowHistoryRate    = 6
	// Convert
	getConvertCurrenciesRate   = 6
	getConvertCurrencyPairRate = 6
	estimateQuoteRate          = 10
	convertTradeRate           = 10
	getConvertHistoryRate      = 6
	// Account
	getAccountBalanceRate                 = 10
	getPositionsRate                      = 10
	getPositionsHistoryRate               = 1
	getAccountAndPositionRiskRate         = 10
	getBillsDetailsRate                   = 6
	getAccountConfigurationRate           = 5
	setPositionModeRate                   = 5
	setLeverageRate                       = 20
	getMaximumBuyOrSellAmountRate         = 20
	getMaximumAvailableTradableAmountRate = 20
	increaseOrDecreaseMarginRate          = 20
	getLeverageRate                       = 20
	getTheMaximumLoanOfInstrumentRate     = 20
	getFeeRatesRate                       = 5
	getInterestAccruedDataRate            = 5
	getInterestRateRate                   = 5
	setGreeksRate                         = 5
	isolatedMarginTradingSettingsRate     = 5
	getMaximumWithdrawalsRate             = 20
	getAccountRiskStateRate               = 10
	vipLoansBorrowAndRepayRate            = 6
	getBorrowAnsRepayHistoryHistoryRate   = 5
	getBorrowInterestAndLimitRate         = 5
	positionBuilderRate                   = 2
	getGreeksRate                         = 10
	getPMLimitation                       = 10
	// Sub Account Endpoints
	viewSubaccountListRate                             = 2
	resetSubAccountAPIKey                              = 1
	getSubaccountTradingBalanceRate                    = 2
	getSubaccountFundingBalanceRate                    = 2
	historyOfSubaccountTransferRate                    = 6
	masterAccountsManageTransfersBetweenSubaccountRate = 1
	setPermissionOfTransferOutRate                     = 1
	getCustodyTradingSubaccountListRate                = 1
	gridTradingRate                                    = 20
	amendGridAlgoOrderRate                             = 20
	stopGridAlgoOrderRate                              = 20
	getGridAlgoOrderListRate                           = 20
	getGridAlgoOrderHistoryRate                        = 20
	getGridAlgoOrderDetailsRate                        = 20
	getGridAlgoSubOrdersRate                           = 20
	getGridAlgoOrderPositionsRate                      = 20
	spotGridWithdrawIncomeRate                         = 20
	computeMarginBalance                               = 20
	adjustMarginBalance                                = 20
	getGridAIParameter                                 = 20
	// Earn
	getOffer                   = 3
	purchase                   = 2
	redeem                     = 2
	cancelPurchaseOrRedemption = 2
	getEarnActiveOrders        = 3
	getFundingOrderHistory     = 3
	// Market Data
	getTickersRate               = 20
	getIndexTickersRate          = 20
	getOrderBookRate             = 20
	getCandlesticksRate          = 40
	getCandlesticksHistoryRate   = 20
	getIndexCandlesticksRate     = 20
	getMarkPriceCandlesticksRate = 20
	getTradesRequestRate         = 100
	get24HTotalVolumeRate        = 2
	getOracleRate                = 1
	getExchangeRateRequestRate   = 1
	getIndexComponentsRate       = 20
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
	getEstimatedDeliveryExercisePriceRate      = 10
	getDiscountRateAndInterestFreeQuotaRate    = 2
	getSystemTimeRate                          = 10
	getLiquidationOrdersRate                   = 40
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
	getContractsOpenInterestAndVolumeRate = 5
	getOptionsOpenInterestAndVolumeRate   = 5
	getPutCallRatioRate                   = 5
	getOpenInterestAndVolumeRate          = 5
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
	getOrderHistory7DaysEPL
	getOrderHistory3MonthsEPL
	getTransactionDetail3DaysEPL
	getTransactionDetail3MonthsEPL
	placeAlgoOrderEPL
	cancelAlgoOrderEPL
	cancelAdvanceAlgoOrderEPL
	getAlgoOrderListEPL
	getAlgoOrderHistoryEPL
	getEasyConvertCurrencyListEPL
	placeEasyConvertEPL
	getEasyConvertHistoryEPL
	getOneClickRepayHistoryEPL
	oneClickRepayCurrencyListEPL
	tradeOneClickRepayEPL
	getCounterpartiesEPL
	createRfqEPL
	cancelRfqEPL
	cancelMultipleRfqEPL
	cancelAllRfqsEPL
	executeQuoteEPL
	setQuoteProductsEPL
	restMMPStatusEPL
	createQuoteEPL
	cancelQuoteEPL
	cancelMultipleQuotesEPL
	cancelAllQuotesEPL
	getRfqsEPL
	getQuotesEPL
	getTradesEPL
	getTradesHistoryEPL
	getPublicTradesEPL
	getCurrenciesEPL
	getBalanceEPL
	getAccountAssetValuationEPL
	fundsTransferEPL
	getFundsTransferStateEPL
	assetBillsDetailsEPL
	lightningDepositsEPL
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
	getLendingHistoryEPL
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
	setLeverageEPL
	getMaximumBuyOrSellAmountEPL
	getMaximumAvailableTradableAmountEPL
	increaseOrDecreaseMarginEPL
	getLeverageEPL
	getTheMaximumLoanOfInstrumentEPL
	getFeeRatesEPL
	getInterestAccruedDataEPL
	getInterestRateEPL
	setGreeksEPL
	isolatedMarginTradingSettingsEPL
	getMaximumWithdrawalsEPL
	getAccountRiskStateEPL
	vipLoansBorrowAnsRepayEPL
	getBorrowAnsRepayHistoryHistoryEPL
	getBorrowInterestAndLimitEPL
	positionBuilderEPL
	getGreeksEPL
	getPMLimitationEPL
	viewSubaccountListEPL
	resetSubAccountAPIKeyEPL
	getSubaccountTradingBalanceEPL
	getSubaccountFundingBalanceEPL
	historyOfSubaccountTransferEPL
	masterAccountsManageTransfersBetweenSubaccountEPL
	setPermissionOfTransferOutEPL
	getCustodyTradingSubaccountListEPL
	gridTradingEPL
	amendGridAlgoOrderEPL
	stopGridAlgoOrderEPL
	getGridAlgoOrderListEPL
	getGridAlgoOrderHistoryEPL
	getGridAlgoOrderDetailsEPL
	getGridAlgoSubOrdersEPL
	getGridAlgoOrderPositionsEPL
	spotGridWithdrawIncomeEPL
	computeMarginBalanceEPL
	adjustMarginBalanceEPL
	getGridAIParameterEPL
	getOfferEPL
	purchaseEPL
	redeemEPL
	cancelPurchaseOrRedemptionEPL
	getEarnActiveOrdersEPL
	getFundingOrderHistoryEPL
	getTickersEPL
	getIndexTickersEPL
	getOrderBookEPL
	getCandlesticksEPL
	getTradesRequestEPL
	get24HTotalVolumeEPL
	getOracleEPL
	getExchangeRateRequestEPL
	getIndexComponentsEPL
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
	getContractsOpenInterestAndVolumeEPL
	getOptionsOpenInterestAndVolumeEPL
	getPutCallRatioEPL
	getOpenInterestAndVolumeEPL
	getTakerFlowEPL
	getEventStatusEPL
	getCandlestickHistoryEPL
	getIndexCandlesticksEPL
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
		return r.CloseDeposit.Wait(ctx)
	case getOrderDetEPL:
		return r.GetOrderDetails.Wait(ctx)
	case getOrderListEPL:
		return r.GetOrderList.Wait(ctx)
	case getOrderHistory7DaysEPL:
		return r.GetOrderHistory7Days.Wait(ctx)
	case getOrderHistory3MonthsEPL:
		return r.GetOrderHistory3Months.Wait(ctx)
	case getTransactionDetail3DaysEPL:
		return r.GetTransactionDetail3Days.Wait(ctx)
	case getTransactionDetail3MonthsEPL:
		return r.GetTransactionDetail3Months.Wait(ctx)
	case placeAlgoOrderEPL:
		return r.PlaceAlgoOrder.Wait(ctx)
	case cancelAlgoOrderEPL:
		return r.CancelAlgoOrder.Wait(ctx)
	case cancelAdvanceAlgoOrderEPL:
		return r.CancelAdvanceAlgoOrder.Wait(ctx)
	case getAlgoOrderListEPL:
		return r.GetAlgoOrderList.Wait(ctx)
	case getAlgoOrderHistoryEPL:
		return r.GetAlgoOrderHistory.Wait(ctx)
	case getEasyConvertCurrencyListEPL:
		return r.GetEasyConvertCurrencyList.Wait(ctx)
	case placeEasyConvertEPL:
		return r.PlaceEasyConvert.Wait(ctx)
	case getEasyConvertHistoryEPL:
		return r.GetEasyConvertHistory.Wait(ctx)
	case getOneClickRepayHistoryEPL:
		return r.GetOneClickRepayHistory.Wait(ctx)
	case oneClickRepayCurrencyListEPL:
		return r.OneClickRepayCurrencyList.Wait(ctx)
	case tradeOneClickRepayEPL:
		return r.TradeOneClickRepay.Wait(ctx)
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
	case setQuoteProductsEPL:
		return r.SetQuoteProducts.Wait(ctx)
	case restMMPStatusEPL:
		return r.RestMMPStatus.Wait(ctx)
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
	case getTradesHistoryEPL:
		return r.GetTradesHistory.Wait(ctx)
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
	case lightningDepositsEPL:
		return r.LightningDeposits.Wait(ctx)
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
		return r.SavingsPurchaseRedempt.Wait(ctx)
	case setLendingRateEPL:
		return r.SetLendingRate.Wait(ctx)
	case getLendingHistoryEPL:
		return r.GetLendingHistory.Wait(ctx)
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
	case setLeverageEPL:
		return r.SetLeverage.Wait(ctx)
	case getMaximumBuyOrSellAmountEPL:
		return r.GetMaximumBuyOrSellAmount.Wait(ctx)
	case getMaximumAvailableTradableAmountEPL:
		return r.GetMaximumAvailableTradableAmount.Wait(ctx)
	case increaseOrDecreaseMarginEPL:
		return r.IncreaseOrDecreaseMargin.Wait(ctx)
	case getLeverageEPL:
		return r.GetLeverage.Wait(ctx)
	case getTheMaximumLoanOfInstrumentEPL:
		return r.GetTheMaximumLoanOfInstrument.Wait(ctx)
	case getFeeRatesEPL:
		return r.GetFeeRates.Wait(ctx)
	case getInterestAccruedDataEPL:
		return r.GetInterestAccruedData.Wait(ctx)
	case getInterestRateEPL:
		return r.GetInterestRate.Wait(ctx)
	case setGreeksEPL:
		return r.SetGreeks.Wait(ctx)
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
	case getGreeksEPL:
		return r.GetGreeks.Wait(ctx)
	case getPMLimitationEPL:
		return r.GetPMLimitation.Wait(ctx)
	case viewSubaccountListEPL:
		return r.ViewSubaccountList.Wait(ctx)
	case resetSubAccountAPIKeyEPL:
		return r.ResetSubAccountAPIKey.Wait(ctx)
	case getSubaccountTradingBalanceEPL:
		return r.GetSubaccountTradingBalance.Wait(ctx)
	case getSubaccountFundingBalanceEPL:
		return r.GetSubaccountFundingBalance.Wait(ctx)
	case historyOfSubaccountTransferEPL:
		return r.HistoryOfSubaccountTransfer.Wait(ctx)
	case masterAccountsManageTransfersBetweenSubaccountEPL:
		return r.MasterAccountsManageTransfersBetweenSubaccount.Wait(ctx)
	case setPermissionOfTransferOutEPL:
		return r.SetPermissionOfTransferOut.Wait(ctx)
	case getCustodyTradingSubaccountListEPL:
		return r.GetCustodyTradingSubaccountList.Wait(ctx)
	case gridTradingEPL:
		return r.GridTrading.Wait(ctx)
	case amendGridAlgoOrderEPL:
		return r.AmendGridAlgoOrder.Wait(ctx)
	case stopGridAlgoOrderEPL:
		return r.StopGridAlgoOrder.Wait(ctx)
	case getGridAlgoOrderListEPL:
		return r.GetGridAlgoOrderList.Wait(ctx)
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
	case computeMarginBalanceEPL:
		return r.ComputeMarginBalance.Wait(ctx)
	case adjustMarginBalanceEPL:
		return r.AdjustMarginBalance.Wait(ctx)
	case getGridAIParameterEPL:
		return r.GetGridAIParameter.Wait(ctx)
	case getOfferEPL:
		return r.GetOffer.Wait(ctx)
	case purchaseEPL:
		return r.Purchase.Wait(ctx)
	case redeemEPL:
		return r.Redeem.Wait(ctx)
	case cancelPurchaseOrRedemptionEPL:
		return r.CancelPurchaseOrRedemption.Wait(ctx)
	case getEarnActiveOrdersEPL:
		return r.GetEarnActiveOrders.Wait(ctx)
	case getFundingOrderHistoryEPL:
		return r.GetFundingOrderHistory.Wait(ctx)
	case getTickersEPL:
		return r.GetTickers.Wait(ctx)
	case getIndexTickersEPL:
		return r.GetIndexTickers.Wait(ctx)
	case getOrderBookEPL:
		return r.GetOrderBook.Wait(ctx)
	case getCandlesticksEPL:
		return r.GetCandlesticks.Wait(ctx)
	case getCandlestickHistoryEPL:
		return r.GetCandlesticksHistory.Wait(ctx)
	case getIndexCandlesticksEPL:
		return r.GetIndexCandlesticks.Wait(ctx)
	case getTradesRequestEPL:
		return r.GetTradesRequest.Wait(ctx)
	case get24HTotalVolumeEPL:
		return r.Get24HTotalVolume.Wait(ctx)
	case getOracleEPL:
		return r.GetOracle.Wait(ctx)
	case getExchangeRateRequestEPL:
		return r.GetExchangeRateRequest.Wait(ctx)
	case getIndexComponentsEPL:
		return r.GetIndexComponents.Wait(ctx)
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
		return r.GetEstimatedDeliveryExercisePrice.Wait(ctx)
	case getDiscountRateAndInterestFreeQuotaEPL:
		return r.GetDiscountRateAndInterestFreeQuota.Wait(ctx)
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
	case getContractsOpenInterestAndVolumeEPL:
		return r.GetContractsOpenInterestAndVolume.Wait(ctx)
	case getOptionsOpenInterestAndVolumeEPL:
		return r.GetOptionsOpenInterestAndVolume.Wait(ctx)
	case getPutCallRatioEPL:
		return r.GetPutCallRatio.Wait(ctx)
	case getOpenInterestAndVolumeEPL:
		return r.GetOpenInterestAndVolume.Wait(ctx)
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
		PlaceOrder:                  request.NewRateLimit(twoSecondsInterval, placeOrderRate),
		PlaceMultipleOrders:         request.NewRateLimit(twoSecondsInterval, placeMultipleOrdersRate),
		CancelOrder:                 request.NewRateLimit(twoSecondsInterval, cancelOrderRate),
		CancelMultipleOrders:        request.NewRateLimit(twoSecondsInterval, cancelMultipleOrdersRate),
		AmendOrder:                  request.NewRateLimit(twoSecondsInterval, amendOrderRate),
		AmendMultipleOrders:         request.NewRateLimit(twoSecondsInterval, amendMultipleOrdersRate),
		CloseDeposit:                request.NewRateLimit(twoSecondsInterval, closeDepositions),
		GetOrderDetails:             request.NewRateLimit(twoSecondsInterval, getOrderDetails),
		GetOrderList:                request.NewRateLimit(twoSecondsInterval, getOrderListRate),
		GetOrderHistory7Days:        request.NewRateLimit(twoSecondsInterval, getOrderHistory7DaysRate),
		GetOrderHistory3Months:      request.NewRateLimit(twoSecondsInterval, getOrderHistory3MonthsRate),
		GetTransactionDetail3Days:   request.NewRateLimit(twoSecondsInterval, getTransactionDetail3DaysRate),
		GetTransactionDetail3Months: request.NewRateLimit(twoSecondsInterval, getTransactionDetail3MonthsRate),
		PlaceAlgoOrder:              request.NewRateLimit(twoSecondsInterval, placeAlgoOrderRate),
		CancelAlgoOrder:             request.NewRateLimit(twoSecondsInterval, cancelAlgoOrderRate),
		CancelAdvanceAlgoOrder:      request.NewRateLimit(twoSecondsInterval, cancelAdvanceAlgoOrderRate),
		GetAlgoOrderList:            request.NewRateLimit(twoSecondsInterval, getAlgoOrderListRate),
		GetAlgoOrderHistory:         request.NewRateLimit(twoSecondsInterval, getAlgoOrderHistoryRate),
		GetEasyConvertCurrencyList:  request.NewRateLimit(twoSecondsInterval, getEasyConvertCurrencyListRate),
		PlaceEasyConvert:            request.NewRateLimit(twoSecondsInterval, placeEasyConvert),
		GetEasyConvertHistory:       request.NewRateLimit(twoSecondsInterval, getEasyConvertHistory),
		GetOneClickRepayHistory:     request.NewRateLimit(twoSecondsInterval, getOneClickRepayHistory),
		OneClickRepayCurrencyList:   request.NewRateLimit(twoSecondsInterval, oneClickRepayCurrencyList),
		TradeOneClickRepay:          request.NewRateLimit(twoSecondsInterval, tradeOneClickRepay),
		// Block Trading endpoints
		GetCounterparties:    request.NewRateLimit(twoSecondsInterval, getCounterpartiesRate),
		CreateRfq:            request.NewRateLimit(twoSecondsInterval, createRfqRate),
		CancelRfq:            request.NewRateLimit(twoSecondsInterval, cancelRfqRate),
		CancelMultipleRfq:    request.NewRateLimit(twoSecondsInterval, cancelMultipleRfqRate),
		CancelAllRfqs:        request.NewRateLimit(twoSecondsInterval, cancelAllRfqsRate),
		ExecuteQuote:         request.NewRateLimit(threeSecondsInterval, executeQuoteRate),
		SetQuoteProducts:     request.NewRateLimit(twoSecondsInterval, setQuoteProducts),
		RestMMPStatus:        request.NewRateLimit(twoSecondsInterval, restMMPStatus),
		CreateQuote:          request.NewRateLimit(twoSecondsInterval, createQuoteRate),
		CancelQuote:          request.NewRateLimit(twoSecondsInterval, cancelQuoteRate),
		CancelMultipleQuotes: request.NewRateLimit(twoSecondsInterval, cancelMultipleQuotesRate),
		CancelAllQuotes:      request.NewRateLimit(twoSecondsInterval, cancelAllQuotes),
		GetRfqs:              request.NewRateLimit(twoSecondsInterval, getRfqsRate),
		GetQuotes:            request.NewRateLimit(twoSecondsInterval, getQuotesRate),
		GetTrades:            request.NewRateLimit(twoSecondsInterval, getTradesRate),
		GetTradesHistory:     request.NewRateLimit(twoSecondsInterval, getTradesHistoryRate),
		GetPublicTrades:      request.NewRateLimit(twoSecondsInterval, getPublicTradesRate),
		// Funding
		GetCurrencies:            request.NewRateLimit(oneSecondInterval, getCurrenciesRate),
		GetBalance:               request.NewRateLimit(oneSecondInterval, getBalanceRate),
		GetAccountAssetValuation: request.NewRateLimit(twoSecondsInterval, getAccountAssetValuationRate),
		FundsTransfer:            request.NewRateLimit(oneSecondInterval, fundsTransferRate),
		GetFundsTransferState:    request.NewRateLimit(oneSecondInterval, getFundsTransferStateRate),
		AssetBillsDetails:        request.NewRateLimit(oneSecondInterval, assetBillsDetailsRate),
		LightningDeposits:        request.NewRateLimit(oneSecondInterval, lightningDepositsRate),
		GetDepositAddress:        request.NewRateLimit(oneSecondInterval, getDepositAddressRate),
		GetDepositHistory:        request.NewRateLimit(oneSecondInterval, getDepositHistoryRate),
		Withdrawal:               request.NewRateLimit(oneSecondInterval, withdrawalRate),
		LightningWithdrawals:     request.NewRateLimit(oneSecondInterval, lightningWithdrawalsRate),
		CancelWithdrawal:         request.NewRateLimit(oneSecondInterval, cancelWithdrawalRate),
		GetWithdrawalHistory:     request.NewRateLimit(oneSecondInterval, getWithdrawalHistoryRate),
		SmallAssetsConvert:       request.NewRateLimit(oneSecondInterval, smallAssetsConvertRate),
		GetSavingBalance:         request.NewRateLimit(oneSecondInterval, getSavingBalanceRate),
		SavingsPurchaseRedempt:   request.NewRateLimit(oneSecondInterval, savingsPurchaseRedemptionRate),
		SetLendingRate:           request.NewRateLimit(oneSecondInterval, setLendingRateRate),
		GetLendingHistory:        request.NewRateLimit(oneSecondInterval, getLendingHistoryRate),
		GetPublicBorrowInfo:      request.NewRateLimit(oneSecondInterval, getPublicBorrowInfoRate),
		GetPublicBorrowHistory:   request.NewRateLimit(oneSecondInterval, getPublicBorrowHistoryRate),
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
		SetLeverage:                       request.NewRateLimit(twoSecondsInterval, setLeverageRate),
		GetMaximumBuyOrSellAmount:         request.NewRateLimit(twoSecondsInterval, getMaximumBuyOrSellAmountRate),
		GetMaximumAvailableTradableAmount: request.NewRateLimit(twoSecondsInterval, getMaximumAvailableTradableAmountRate),
		IncreaseOrDecreaseMargin:          request.NewRateLimit(twoSecondsInterval, increaseOrDecreaseMarginRate),
		GetLeverage:                       request.NewRateLimit(twoSecondsInterval, getLeverageRate),
		GetTheMaximumLoanOfInstrument:     request.NewRateLimit(twoSecondsInterval, getTheMaximumLoanOfInstrumentRate),
		GetFeeRates:                       request.NewRateLimit(twoSecondsInterval, getFeeRatesRate),
		GetInterestAccruedData:            request.NewRateLimit(twoSecondsInterval, getInterestAccruedDataRate),
		GetInterestRate:                   request.NewRateLimit(twoSecondsInterval, getInterestRateRate),
		SetGreeks:                         request.NewRateLimit(twoSecondsInterval, setGreeksRate),
		IsolatedMarginTradingSettings:     request.NewRateLimit(twoSecondsInterval, isolatedMarginTradingSettingsRate),
		GetMaximumWithdrawals:             request.NewRateLimit(twoSecondsInterval, getMaximumWithdrawalsRate),
		GetAccountRiskState:               request.NewRateLimit(twoSecondsInterval, getAccountRiskStateRate),
		VipLoansBorrowAnsRepay:            request.NewRateLimit(oneSecondInterval, vipLoansBorrowAndRepayRate),
		GetBorrowAnsRepayHistoryHistory:   request.NewRateLimit(twoSecondsInterval, getBorrowAnsRepayHistoryHistoryRate),
		GetBorrowInterestAndLimit:         request.NewRateLimit(twoSecondsInterval, getBorrowInterestAndLimitRate),
		PositionBuilder:                   request.NewRateLimit(twoSecondsInterval, positionBuilderRate),
		GetGreeks:                         request.NewRateLimit(twoSecondsInterval, getGreeksRate),
		GetPMLimitation:                   request.NewRateLimit(twoSecondsInterval, getPMLimitation),
		// Sub Account Endpoints

		ViewSubaccountList:                             request.NewRateLimit(twoSecondsInterval, viewSubaccountListRate),
		ResetSubAccountAPIKey:                          request.NewRateLimit(oneSecondInterval, resetSubAccountAPIKey),
		GetSubaccountTradingBalance:                    request.NewRateLimit(twoSecondsInterval, getSubaccountTradingBalanceRate),
		GetSubaccountFundingBalance:                    request.NewRateLimit(twoSecondsInterval, getSubaccountFundingBalanceRate),
		HistoryOfSubaccountTransfer:                    request.NewRateLimit(oneSecondInterval, historyOfSubaccountTransferRate),
		MasterAccountsManageTransfersBetweenSubaccount: request.NewRateLimit(oneSecondInterval, masterAccountsManageTransfersBetweenSubaccountRate),
		SetPermissionOfTransferOut:                     request.NewRateLimit(oneSecondInterval, setPermissionOfTransferOutRate),
		GetCustodyTradingSubaccountList:                request.NewRateLimit(oneSecondInterval, getCustodyTradingSubaccountListRate),
		// Grid Trading Endpoints

		GridTrading:               request.NewRateLimit(twoSecondsInterval, gridTradingRate),
		AmendGridAlgoOrder:        request.NewRateLimit(twoSecondsInterval, amendGridAlgoOrderRate),
		StopGridAlgoOrder:         request.NewRateLimit(twoSecondsInterval, stopGridAlgoOrderRate),
		GetGridAlgoOrderList:      request.NewRateLimit(twoSecondsInterval, getGridAlgoOrderListRate),
		GetGridAlgoOrderHistory:   request.NewRateLimit(twoSecondsInterval, getGridAlgoOrderHistoryRate),
		GetGridAlgoOrderDetails:   request.NewRateLimit(twoSecondsInterval, getGridAlgoOrderDetailsRate),
		GetGridAlgoSubOrders:      request.NewRateLimit(twoSecondsInterval, getGridAlgoSubOrdersRate),
		GetGridAlgoOrderPositions: request.NewRateLimit(twoSecondsInterval, getGridAlgoOrderPositionsRate),
		SpotGridWithdrawIncome:    request.NewRateLimit(twoSecondsInterval, spotGridWithdrawIncomeRate),
		ComputeMarginBalance:      request.NewRateLimit(twoSecondsInterval, computeMarginBalance),
		AdjustMarginBalance:       request.NewRateLimit(twoSecondsInterval, adjustMarginBalance),
		GetGridAIParameter:        request.NewRateLimit(twoSecondsInterval, getGridAIParameter),
		// Earn
		GetOffer:                   request.NewRateLimit(oneSecondInterval, getOffer),
		Purchase:                   request.NewRateLimit(oneSecondInterval, purchase),
		Redeem:                     request.NewRateLimit(oneSecondInterval, redeem),
		CancelPurchaseOrRedemption: request.NewRateLimit(oneSecondInterval, cancelPurchaseOrRedemption),
		GetEarnActiveOrders:        request.NewRateLimit(oneSecondInterval, getEarnActiveOrders),
		GetFundingOrderHistory:     request.NewRateLimit(oneSecondInterval, getFundingOrderHistory),
		// Market Data
		GetTickers:               request.NewRateLimit(twoSecondsInterval, getTickersRate),
		GetIndexTickers:          request.NewRateLimit(twoSecondsInterval, getIndexTickersRate),
		GetOrderBook:             request.NewRateLimit(twoSecondsInterval, getOrderBookRate),
		GetCandlesticks:          request.NewRateLimit(twoSecondsInterval, getCandlesticksRate),
		GetCandlesticksHistory:   request.NewRateLimit(twoSecondsInterval, getCandlesticksHistoryRate),
		GetIndexCandlesticks:     request.NewRateLimit(twoSecondsInterval, getIndexCandlesticksRate),
		GetMarkPriceCandlesticks: request.NewRateLimit(twoSecondsInterval, getMarkPriceCandlesticksRate),
		GetTradesRequest:         request.NewRateLimit(twoSecondsInterval, getTradesRequestRate),
		Get24HTotalVolume:        request.NewRateLimit(twoSecondsInterval, get24HTotalVolumeRate),
		GetOracle:                request.NewRateLimit(fiveSecondsInterval, getOracleRate),
		GetExchangeRateRequest:   request.NewRateLimit(twoSecondsInterval, getExchangeRateRequestRate),
		GetIndexComponents:       request.NewRateLimit(twoSecondsInterval, getIndexComponentsRate),
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
		GetEstimatedDeliveryExercisePrice:      request.NewRateLimit(twoSecondsInterval, getEstimatedDeliveryExercisePriceRate),
		GetDiscountRateAndInterestFreeQuota:    request.NewRateLimit(twoSecondsInterval, getDiscountRateAndInterestFreeQuotaRate),
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
		GetContractsOpenInterestAndVolume: request.NewRateLimit(twoSecondsInterval, getContractsOpenInterestAndVolumeRate),
		GetOptionsOpenInterestAndVolume:   request.NewRateLimit(twoSecondsInterval, getOptionsOpenInterestAndVolumeRate),
		GetPutCallRatio:                   request.NewRateLimit(twoSecondsInterval, getPutCallRatioRate),
		GetOpenInterestAndVolume:          request.NewRateLimit(twoSecondsInterval, getOpenInterestAndVolumeRate),
		GetTakerFlow:                      request.NewRateLimit(twoSecondsInterval, getTakerFlowRate),

		// Status Endpoints

		GetEventStatus: request.NewRateLimit(fiveSecondsInterval, getEventStatusRate),
	}
}
