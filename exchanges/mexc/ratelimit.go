package mexc

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	tenSecondsInterval  = time.Second * 10
	fiveSecondsInterval = time.Second * 5
	twoSecondsInterval  = time.Second * 2
	oneSecondInterval   = time.Second
)

const (
	systemTimeEPL request.EndpointLimit = iota
	defaultSymbolsEPL
	getSymbolsEPL
	orderbooksEPL
	recentTradesListEPL
	aggregatedTradesEPL
	candlestickEPL
	currentAveragePriceEPL
	symbolTickerPriceChangeStatEPL
	symbolsTickerPriceChangeStatEPL
	symbolPriceTickerEPL
	symbolsPriceTickerEPL
	symbolOrderbookTickerEPL
	createSubAccountEPL
	subAccountListEPL
	createAPIKeyForSubAccountEPL
	getSubAccountAPIKeyEPL
	deleteSubAccountAPIKeyEPL
	subAccountUniversalTransferEPL
	getSubaccUnversalTransfersEPL
	getSubAccountAssetEPL
	getKYCStatusEPL
	selfSymbolsEPL
	newOrderEPL
	createBatchOrdersEPL
	cancelTradeOrderEPL
	cancelAllOpenOrdersBySymbolEPL
	cancelAllOrdersEPL
	getOrderByIDEPL
	getOpenOrdersEPL
	allOrdersEPL
	accountInformationEPL
	accountTradeListEPL
	enableMXDeductEPL
	getMXDeductStatusEPL
	getSymbolTradingFeeEPL
	getCurrencyInformationEPL
	withdrawCapitalEPL
	cancelWithdrawalEPL
	getFundDepositHistoryEPL
	getWithdrawalHistoryEPL
	generateDepositAddressEPL
	getDepositAddressEPL
	getWithdrawalAddressEPL
	userUniversalTransferEPL
	getUniversalTransferDetailByIDEPL
	getAssetConvertedMXEPL
	dustConvertEPL
	dustLogEPL
	internalTransferEPL
	getInternalTransferHistoryEPL
	capitalWithdrawalEPL

	getUniversalTransferhistoryEPL
	getUserRebateHistoryEPL
	getRebateRecordsDetailEPL
	selfRebateRecordsDetailsEPL
	getReferCodeEPL
	getAffilateCommissionRecordEPL
	getAffilateWithdrawRecordEPL
	getAffiliateConnissionDetailEPL
	affiliateCampaignDataEPL
	affiliateReferralDataEPL
	subAffiliateDataEPL
	getUIDEPL
	getAPIKeyInfoEPL
	setAPIKeyInfoEPL
	offlineSymbolsEPL
	announcementsEPL
	createSTPGroupEPL
	getSTPGroupEPL
	deleteSTPGroupEPL
	addSTPGroupUIDsEPL
	deleteSTPGroupUIDsEPL
)

// GetRateLimit returns a RateLimit instance, which implements the request.Limiter interface.
func GetRateLimit() request.RateLimitDefinitions {
	// IP-weighted endpoints share 300 weight per 10 seconds; the order-placement and cancel
	// endpoints are documented as a shared 12-requests-per-second budget, which is the binding
	// constraint on them (12/s is well inside the UID pool they also sit in), so they draw from one
	// per-second limiter rather than a weighted pool.
	ipModeRate := request.NewRateLimit(tenSecondsInterval, 300)
	orderRate := request.NewRateLimit(oneSecondInterval, 12)
	// Announcements are limited on their own at 5 requests per 2 seconds, outside the weighted pool.
	announcementsRate := request.NewRateLimit(twoSecondsInterval, 5)

	return request.RateLimitDefinitions{
		systemTimeEPL:          request.GetRateLimiterWithWeight(ipModeRate, 1),
		defaultSymbolsEPL:      request.GetRateLimiterWithWeight(ipModeRate, 1),
		getSymbolsEPL:          request.GetRateLimiterWithWeight(ipModeRate, 25),
		orderbooksEPL:          request.GetRateLimiterWithWeight(ipModeRate, 3),
		recentTradesListEPL:    request.GetRateLimiterWithWeight(ipModeRate, 5),
		aggregatedTradesEPL:    request.GetRateLimiterWithWeight(ipModeRate, 1),
		candlestickEPL:         request.GetRateLimiterWithWeight(ipModeRate, 1),
		currentAveragePriceEPL: request.GetRateLimiterWithWeight(ipModeRate, 1),

		// The 24hr ticker costs 25 whether one symbol or the whole catalogue is requested.
		symbolTickerPriceChangeStatEPL:  request.GetRateLimiterWithWeight(ipModeRate, 25),
		symbolsTickerPriceChangeStatEPL: request.GetRateLimiterWithWeight(ipModeRate, 25),

		symbolPriceTickerEPL:              request.GetRateLimiterWithWeight(ipModeRate, 10),
		symbolsPriceTickerEPL:             request.GetRateLimiterWithWeight(ipModeRate, 10),
		symbolOrderbookTickerEPL:          request.GetRateLimiterWithWeight(ipModeRate, 10),
		createSubAccountEPL:               request.GetRateLimiterWithWeight(ipModeRate, 1),
		subAccountListEPL:                 request.GetRateLimiterWithWeight(ipModeRate, 1),
		createAPIKeyForSubAccountEPL:      request.GetRateLimiterWithWeight(ipModeRate, 1),
		getSubAccountAPIKeyEPL:            request.GetRateLimiterWithWeight(ipModeRate, 1),
		deleteSubAccountAPIKeyEPL:         request.GetRateLimiterWithWeight(ipModeRate, 1),
		subAccountUniversalTransferEPL:    request.GetRateLimiterWithWeight(ipModeRate, 50),
		getSubaccUnversalTransfersEPL:     request.GetRateLimiterWithWeight(ipModeRate, 1),
		getSubAccountAssetEPL:             request.GetRateLimiterWithWeight(ipModeRate, 1),
		getKYCStatusEPL:                   request.GetRateLimiterWithWeight(ipModeRate, 1),
		selfSymbolsEPL:                    request.GetRateLimiterWithWeight(ipModeRate, 1),
		newOrderEPL:                       request.GetRateLimiterWithWeight(orderRate, 1),
		createBatchOrdersEPL:              request.GetRateLimiterWithWeight(orderRate, 1),
		cancelTradeOrderEPL:               request.GetRateLimiterWithWeight(orderRate, 1),
		cancelAllOpenOrdersBySymbolEPL:    request.GetRateLimiterWithWeight(orderRate, 1),
		cancelAllOrdersEPL:                request.GetRateLimiterWithWeight(orderRate, 1),
		getOrderByIDEPL:                   request.GetRateLimiterWithWeight(ipModeRate, 2),
		getOpenOrdersEPL:                  request.GetRateLimiterWithWeight(ipModeRate, 3),
		allOrdersEPL:                      request.GetRateLimiterWithWeight(ipModeRate, 10),
		accountInformationEPL:             request.GetRateLimiterWithWeight(ipModeRate, 10),
		accountTradeListEPL:               request.GetRateLimiterWithWeight(ipModeRate, 10),
		enableMXDeductEPL:                 request.GetRateLimiterWithWeight(ipModeRate, 1),
		getMXDeductStatusEPL:              request.GetRateLimiterWithWeight(ipModeRate, 1),
		getSymbolTradingFeeEPL:            request.GetRateLimiterWithWeight(ipModeRate, 20),
		getCurrencyInformationEPL:         request.GetRateLimiterWithWeight(ipModeRate, 10),
		withdrawCapitalEPL:                request.GetRateLimiterWithWeight(ipModeRate, 1),
		cancelWithdrawalEPL:               request.GetRateLimiterWithWeight(ipModeRate, 1),
		getFundDepositHistoryEPL:          request.GetRateLimiterWithWeight(ipModeRate, 10),
		getWithdrawalHistoryEPL:           request.GetRateLimiterWithWeight(ipModeRate, 1),
		generateDepositAddressEPL:         request.GetRateLimiterWithWeight(ipModeRate, 1),
		getDepositAddressEPL:              request.GetRateLimiterWithWeight(ipModeRate, 10),
		getWithdrawalAddressEPL:           request.GetRateLimiterWithWeight(ipModeRate, 10),
		userUniversalTransferEPL:          request.GetRateLimiterWithWeight(ipModeRate, 50),
		getUniversalTransferDetailByIDEPL: request.GetRateLimiterWithWeight(ipModeRate, 1),
		getAssetConvertedMXEPL:            request.GetRateLimiterWithWeight(ipModeRate, 1),
		dustConvertEPL:                    request.GetRateLimiterWithWeight(ipModeRate, 10),
		dustLogEPL:                        request.GetRateLimiterWithWeight(ipModeRate, 1),
		internalTransferEPL:               request.GetRateLimiterWithWeight(ipModeRate, 1),
		getInternalTransferHistoryEPL:     request.GetRateLimiterWithWeight(ipModeRate, 1),
		capitalWithdrawalEPL:              request.GetRateLimiterWithWeight(ipModeRate, 10),

		getUniversalTransferhistoryEPL:  request.GetRateLimiterWithWeight(ipModeRate, 1),
		getUserRebateHistoryEPL:         request.GetRateLimiterWithWeight(ipModeRate, 1),
		getRebateRecordsDetailEPL:       request.GetRateLimiterWithWeight(ipModeRate, 1),
		selfRebateRecordsDetailsEPL:     request.GetRateLimiterWithWeight(ipModeRate, 1),
		getReferCodeEPL:                 request.GetRateLimiterWithWeight(ipModeRate, 1),
		getAffilateCommissionRecordEPL:  request.GetRateLimiterWithWeight(ipModeRate, 1),
		getAffilateWithdrawRecordEPL:    request.GetRateLimiterWithWeight(ipModeRate, 1),
		getAffiliateConnissionDetailEPL: request.GetRateLimiterWithWeight(ipModeRate, 1),
		affiliateCampaignDataEPL:        request.GetRateLimiterWithWeight(ipModeRate, 1),
		affiliateReferralDataEPL:        request.GetRateLimiterWithWeight(ipModeRate, 1),
		subAffiliateDataEPL:             request.GetRateLimiterWithWeight(ipModeRate, 1),

		getUIDEPL:             request.GetRateLimiterWithWeight(ipModeRate, 1),
		getAPIKeyInfoEPL:      request.GetRateLimiterWithWeight(ipModeRate, 1),
		setAPIKeyInfoEPL:      request.GetRateLimiterWithWeight(ipModeRate, 1),
		offlineSymbolsEPL:     request.GetRateLimiterWithWeight(ipModeRate, 10),
		announcementsEPL:      request.GetRateLimiterWithWeight(announcementsRate, 1),
		createSTPGroupEPL:     request.GetRateLimiterWithWeight(ipModeRate, 20),
		getSTPGroupEPL:        request.GetRateLimiterWithWeight(ipModeRate, 20),
		deleteSTPGroupEPL:     request.GetRateLimiterWithWeight(ipModeRate, 20),
		addSTPGroupUIDsEPL:    request.GetRateLimiterWithWeight(ipModeRate, 20),
		deleteSTPGroupUIDsEPL: request.GetRateLimiterWithWeight(ipModeRate, 20),
	}
}
