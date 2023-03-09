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
	GetSavingBalance         *rate.Limiter
	SavingsPurchaseRedemp    *rate.Limiter
	SetLendingRate           *rate.Limiter
	GetLendingHistory        *rate.Limiter
	GetPublicBorrowInfo      *rate.Limiter
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
	getSavingBalanceRate         = 6
	savingsPurchaseRedemption    = 6
	setLendingRateRate           = 6
	getLendingHistoryRate        = 6
	getPublicBorrowInfoRate      = 6
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
)

// Limit executes rate limiting for Okx exchange given the context and EndpointLimit
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) (*rate.Limiter, int, error) {
	switch f {
	case placeOrderEPL:
		return r.PlaceOrder, 1, nil
	case placeMultipleOrdersEPL:
		return r.PlaceMultipleOrders, 1, nil
	case cancelOrderEPL:
		return r.CancelOrder, 1, nil
	case cancelMultipleOrdersEPL:
		return r.CancelMultipleOrders, 1, nil
	case amendOrderEPL:
		return r.AmendOrder, 1, nil
	case amendMultipleOrdersEPL:
		return r.AmendMultipleOrders, 1, nil
	case closePositionEPL:
		return r.CloseDeposit, 1, nil
	case getOrderDetEPL:
		return r.GetOrderDetails, 1, nil
	case getOrderListEPL:
		return r.GetOrderList, 1, nil
	case getOrderHistory7DaysEPL:
		return r.GetOrderHistory7Days, 1, nil
	case getOrderHistory3MonthsEPL:
		return r.GetOrderHistory3Months, 1, nil
	case getTransactionDetail3DaysEPL:
		return r.GetTransactionDetail3Days, 1, nil
	case getTransactionDetail3MonthsEPL:
		return r.GetTransactionDetail3Months, 1, nil
	case placeAlgoOrderEPL:
		return r.PlaceAlgoOrder, 1, nil
	case cancelAlgoOrderEPL:
		return r.CancelAlgoOrder, 1, nil
	case cancelAdvanceAlgoOrderEPL:
		return r.CancelAdvanceAlgoOrder, 1, nil
	case getAlgoOrderListEPL:
		return r.GetAlgoOrderList, 1, nil
	case getAlgoOrderHistoryEPL:
		return r.GetAlgoOrderHistory, 1, nil
	case getEasyConvertCurrencyListEPL:
		return r.GetEasyConvertCurrencyList, 1, nil
	case placeEasyConvertEPL:
		return r.PlaceEasyConvert, 1, nil
	case getEasyConvertHistoryEPL:
		return r.GetEasyConvertHistory, 1, nil
	case getOneClickRepayHistoryEPL:
		return r.GetOneClickRepayHistory, 1, nil
	case oneClickRepayCurrencyListEPL:
		return r.OneClickRepayCurrencyList, 1, nil
	case tradeOneClickRepayEPL:
		return r.TradeOneClickRepay, 1, nil
	case getCounterpartiesEPL:
		return r.GetCounterparties, 1, nil
	case createRfqEPL:
		return r.CreateRfq, 1, nil
	case cancelRfqEPL:
		return r.CancelRfq, 1, nil
	case cancelMultipleRfqEPL:
		return r.CancelMultipleRfq, 1, nil
	case cancelAllRfqsEPL:
		return r.CancelAllRfqs, 1, nil
	case executeQuoteEPL:
		return r.ExecuteQuote, 1, nil
	case setQuoteProductsEPL:
		return r.SetQuoteProducts, 1, nil
	case restMMPStatusEPL:
		return r.RestMMPStatus, 1, nil
	case createQuoteEPL:
		return r.CreateQuote, 1, nil
	case cancelQuoteEPL:
		return r.CancelQuote, 1, nil
	case cancelMultipleQuotesEPL:
		return r.CancelMultipleQuotes, 1, nil
	case cancelAllQuotesEPL:
		return r.CancelAllQuotes, 1, nil
	case getRfqsEPL:
		return r.GetRfqs, 1, nil
	case getQuotesEPL:
		return r.GetQuotes, 1, nil
	case getTradesEPL:
		return r.GetTrades, 1, nil
	case getTradesHistoryEPL:
		return r.GetTradesHistory, 1, nil
	case getPublicTradesEPL:
		return r.GetPublicTrades, 1, nil
	case getCurrenciesEPL:
		return r.GetCurrencies, 1, nil
	case getBalanceEPL:
		return r.GetBalance, 1, nil
	case getAccountAssetValuationEPL:
		return r.GetAccountAssetValuation, 1, nil
	case fundsTransferEPL:
		return r.FundsTransfer, 1, nil
	case getFundsTransferStateEPL:
		return r.GetFundsTransferState, 1, nil
	case assetBillsDetailsEPL:
		return r.AssetBillsDetails, 1, nil
	case lightningDepositsEPL:
		return r.LightningDeposits, 1, nil
	case getDepositAddressEPL:
		return r.GetDepositAddress, 1, nil
	case getDepositHistoryEPL:
		return r.GetDepositHistory, 1, nil
	case withdrawalEPL:
		return r.Withdrawal, 1, nil
	case lightningWithdrawalsEPL:
		return r.LightningWithdrawals, 1, nil
	case cancelWithdrawalEPL:
		return r.CancelWithdrawal, 1, nil
	case getWithdrawalHistoryEPL:
		return r.GetWithdrawalHistory, 1, nil
	case smallAssetsConvertEPL:
		return r.SmallAssetsConvert, 1, nil
	case getSavingBalanceEPL:
		return r.GetSavingBalance, 1, nil
	case savingsPurchaseRedemptionEPL:
		return r.SavingsPurchaseRedemp, 1, nil
	case setLendingRateEPL:
		return r.SetLendingRate, 1, nil
	case getLendingHistoryEPL:
		return r.GetLendingHistory, 1, nil
	case getPublicBorrowInfoEPL:
		return r.GetPublicBorrowInfo, 1, nil
	case getConvertCurrenciesEPL:
		return r.GetConvertCurrencies, 1, nil
	case getConvertCurrencyPairEPL:
		return r.GetConvertCurrencyPair, 1, nil
	case estimateQuoteEPL:
		return r.EstimateQuote, 1, nil
	case convertTradeEPL:
		return r.ConvertTrade, 1, nil
	case getConvertHistoryEPL:
		return r.GetConvertHistory, 1, nil
	case getAccountBalanceEPL:
		return r.GetAccountBalance, 1, nil
	case getPositionsEPL:
		return r.GetPositions, 1, nil
	case getPositionsHistoryEPL:
		return r.GetPositionsHistory, 1, nil
	case getAccountAndPositionRiskEPL:
		return r.GetAccountAndPositionRisk, 1, nil
	case getBillsDetailsEPL:
		return r.GetBillsDetails, 1, nil
	case getAccountConfigurationEPL:
		return r.GetAccountConfiguration, 1, nil
	case setPositionModeEPL:
		return r.SetPositionMode, 1, nil
	case setLeverageEPL:
		return r.SetLeverage, 1, nil
	case getMaximumBuyOrSellAmountEPL:
		return r.GetMaximumBuyOrSellAmount, 1, nil
	case getMaximumAvailableTradableAmountEPL:
		return r.GetMaximumAvailableTradableAmount, 1, nil
	case increaseOrDecreaseMarginEPL:
		return r.IncreaseOrDecreaseMargin, 1, nil
	case getLeverageEPL:
		return r.GetLeverage, 1, nil
	case getTheMaximumLoanOfInstrumentEPL:
		return r.GetTheMaximumLoanOfInstrument, 1, nil
	case getFeeRatesEPL:
		return r.GetFeeRates, 1, nil
	case getInterestAccruedDataEPL:
		return r.GetInterestAccruedData, 1, nil
	case getInterestRateEPL:
		return r.GetInterestRate, 1, nil
	case setGreeksEPL:
		return r.SetGreeks, 1, nil
	case isolatedMarginTradingSettingsEPL:
		return r.IsolatedMarginTradingSettings, 1, nil
	case getMaximumWithdrawalsEPL:
		return r.GetMaximumWithdrawals, 1, nil
	case getAccountRiskStateEPL:
		return r.GetAccountRiskState, 1, nil
	case vipLoansBorrowAnsRepayEPL:
		return r.VipLoansBorrowAnsRepay, 1, nil
	case getBorrowAnsRepayHistoryHistoryEPL:
		return r.GetBorrowAnsRepayHistoryHistory, 1, nil
	case getBorrowInterestAndLimitEPL:
		return r.GetBorrowInterestAndLimit, 1, nil
	case positionBuilderEPL:
		return r.PositionBuilder, 1, nil
	case getGreeksEPL:
		return r.GetGreeks, 1, nil
	case getPMLimitationEPL:
		return r.GetPMLimitation, 1, nil
	case viewSubaccountListEPL:
		return r.ViewSubaccountList, 1, nil
	case resetSubAccountAPIKeyEPL:
		return r.ResetSubAccountAPIKey, 1, nil
	case getSubaccountTradingBalanceEPL:
		return r.GetSubaccountTradingBalance, 1, nil
	case getSubaccountFundingBalanceEPL:
		return r.GetSubaccountFundingBalance, 1, nil
	case historyOfSubaccountTransferEPL:
		return r.HistoryOfSubaccountTransfer, 1, nil
	case masterAccountsManageTransfersBetweenSubaccountEPL:
		return r.MasterAccountsManageTransfersBetweenSubaccount, 1, nil
	case setPermissionOfTransferOutEPL:
		return r.SetPermissionOfTransferOut, 1, nil
	case getCustodyTradingSubaccountListEPL:
		return r.GetCustodyTradingSubaccountList, 1, nil
	case gridTradingEPL:
		return r.GridTrading, 1, nil
	case amendGridAlgoOrderEPL:
		return r.AmendGridAlgoOrder, 1, nil
	case stopGridAlgoOrderEPL:
		return r.StopGridAlgoOrder, 1, nil
	case getGridAlgoOrderListEPL:
		return r.GetGridAlgoOrderList, 1, nil
	case getGridAlgoOrderHistoryEPL:
		return r.GetGridAlgoOrderHistory, 1, nil
	case getGridAlgoOrderDetailsEPL:
		return r.GetGridAlgoOrderDetails, 1, nil
	case getGridAlgoSubOrdersEPL:
		return r.GetGridAlgoSubOrders, 1, nil
	case getGridAlgoOrderPositionsEPL:
		return r.GetGridAlgoOrderPositions, 1, nil
	case spotGridWithdrawIncomeEPL:
		return r.SpotGridWithdrawIncome, 1, nil
	case computeMarginBalanceEPL:
		return r.ComputeMarginBalance, 1, nil
	case adjustMarginBalanceEPL:
		return r.AdjustMarginBalance, 1, nil
	case getGridAIParameterEPL:
		return r.GetGridAIParameter, 1, nil
	case getOfferEPL:
		return r.GetOffer, 1, nil
	case purchaseEPL:
		return r.Purchase, 1, nil
	case redeemEPL:
		return r.Redeem, 1, nil
	case cancelPurchaseOrRedemptionEPL:
		return r.CancelPurchaseOrRedemption, 1, nil
	case getEarnActiveOrdersEPL:
		return r.GetEarnActiveOrders, 1, nil
	case getFundingOrderHistoryEPL:
		return r.GetFundingOrderHistory, 1, nil
	case getTickersEPL:
		return r.GetTickers, 1, nil
	case getIndexTickersEPL:
		return r.GetIndexTickers, 1, nil
	case getOrderBookEPL:
		return r.GetOrderBook, 1, nil
	case getCandlesticksEPL:
		return r.GetCandlesticks, 1, nil
	case getTradesRequestEPL:
		return r.GetTradesRequest, 1, nil
	case get24HTotalVolumeEPL:
		return r.Get24HTotalVolume, 1, nil
	case getOracleEPL:
		return r.GetOracle, 1, nil
	case getExchangeRateRequestEPL:
		return r.GetExchangeRateRequest, 1, nil
	case getIndexComponentsEPL:
		return r.GetIndexComponents, 1, nil
	case getBlockTickersEPL:
		return r.GetBlockTickers, 1, nil
	case getBlockTradesEPL:
		return r.GetBlockTrades, 1, nil
	case getInstrumentsEPL:
		return r.GetInstruments, 1, nil
	case getDeliveryExerciseHistoryEPL:
		return r.GetDeliveryExerciseHistory, 1, nil
	case getOpenInterestEPL:
		return r.GetOpenInterest, 1, nil
	case getFundingEPL:
		return r.GetFunding, 1, nil
	case getFundingRateHistoryEPL:
		return r.GetFundingRateHistory, 1, nil
	case getLimitPriceEPL:
		return r.GetLimitPrice, 1, nil
	case getOptionMarketDateEPL:
		return r.GetOptionMarketDate, 1, nil
	case getEstimatedDeliveryPriceEPL:
		return r.GetEstimatedDeliveryExercisePrice, 1, nil
	case getDiscountRateAndInterestFreeQuotaEPL:
		return r.GetDiscountRateAndInterestFreeQuota, 1, nil
	case getSystemTimeEPL:
		return r.GetSystemTime, 1, nil
	case getLiquidationOrdersEPL:
		return r.GetLiquidationOrders, 1, nil
	case getMarkPriceEPL:
		return r.GetMarkPrice, 1, nil
	case getPositionTiersEPL:
		return r.GetPositionTiers, 1, nil
	case getInterestRateAndLoanQuotaEPL:
		return r.GetInterestRateAndLoanQuota, 1, nil
	case getInterestRateAndLoanQuoteForVIPLoansEPL:
		return r.GetInterestRateAndLoanQuoteForVIPLoans, 1, nil
	case getUnderlyingEPL:
		return r.GetUnderlying, 1, nil
	case getInsuranceFundEPL:
		return r.GetInsuranceFund, 1, nil
	case unitConvertEPL:
		return r.UnitConvert, 1, nil
	case getSupportCoinEPL:
		return r.GetSupportCoin, 1, nil
	case getTakerVolumeEPL:
		return r.GetTakerVolume, 1, nil
	case getMarginLendingRatioEPL:
		return r.GetMarginLendingRatio, 1, nil
	case getLongShortRatioEPL:
		return r.GetLongShortRatio, 1, nil
	case getContractsOpenInterestAndVolumeEPL:
		return r.GetContractsOpenInterestAndVolume, 1, nil
	case getOptionsOpenInterestAndVolumeEPL:
		return r.GetOptionsOpenInterestAndVolume, 1, nil
	case getPutCallRatioEPL:
		return r.GetPutCallRatio, 1, nil
	case getOpenInterestAndVolumeEPL:
		return r.GetOpenInterestAndVolume, 1, nil
	case getTakerFlowEPL:
		return r.GetTakerFlow, 1, nil
	case getEventStatusEPL:
		return r.GetEventStatus, 1, nil
	default:
		return nil, 0, errors.New("endpoint rate limit functionality not found")
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
		SavingsPurchaseRedemp:    request.NewRateLimit(oneSecondInterval, savingsPurchaseRedemption),
		SetLendingRate:           request.NewRateLimit(oneSecondInterval, setLendingRateRate),
		GetLendingHistory:        request.NewRateLimit(oneSecondInterval, getLendingHistoryRate),
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
