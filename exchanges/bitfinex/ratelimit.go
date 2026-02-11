package bitfinex

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Ratelimit intervals.
const (
	oneMinuteInterval = time.Minute
	// Bitfinex rate limits - Public
	platformStatusReqRate     = 15
	tickerBatchReqRate        = 30
	tickerReqRate             = 90
	tradeReqRate              = 30
	orderbookReqRate          = 240
	statsReqRate              = 90
	candleReqRate             = 10
	configsReqRate            = 15
	statusReqRate             = 90
	liquidReqRate             = 15 // This is not specified just inputted WCS
	leaderBoardReqRate        = 90
	marketAveragePriceReqRate = 90
	fxReqRate                 = 90

	// Bitfinex rate limits - Authenticated
	// Wallets -
	accountWalletBalanceReqRate = 45
	accountWalletHistoryReqRate = 45
	// Orders -
	retrieveOrderReqRate  = 45
	submitOrderReqRate    = 45 // This is not specified just inputted above
	updateOrderReqRate    = 45 // This is not specified just inputted above
	cancelOrderReqRate    = 45 // This is not specified just inputted above
	orderBatchReqRate     = 45 // This is not specified just inputted above
	cancelBatchReqRate    = 45 // This is not specified just inputted above
	orderHistoryReqRate   = 45
	getOrderTradesReqRate = 45
	getTradesReqRate      = 45
	getLedgersReqRate     = 45
	// Positions -
	getAccountMarginInfoReqRate       = 45
	getActivePositionsReqRate         = 45
	claimPositionReqRate              = 45 // This is not specified just inputted above
	getPositionHistoryReqRate         = 45
	getPositionAuditReqRate           = 45
	updateCollateralOnPositionReqRate = 45 // This is not specified just inputted above
	// Margin funding -
	getMarginInfoRate               = 90
	getActiveFundingOffersReqRate   = 45
	submitFundingOfferReqRate       = 45 // This is not specified just inputted above
	cancelFundingOfferReqRate       = 45
	cancelAllFundingOfferReqRate    = 45 // This is not specified just inputted above
	closeFundingReqRate             = 45 // This is not specified just inputted above
	fundingAutoRenewReqRate         = 45 // This is not specified just inputted above
	keepFundingReqRate              = 45 // This is not specified just inputted above
	getOffersHistoryReqRate         = 45
	getFundingLoansReqRate          = 45
	getFundingLoanHistoryReqRate    = 45
	getFundingCreditsReqRate        = 45
	getFundingCreditsHistoryReqRate = 45
	getFundingTradesReqRate         = 45
	getFundingInfoReqRate           = 45
	// Account actions
	getUserInfoReqRate               = 45
	transferBetweenWalletsReqRate    = 45 // This is not specified just inputted above
	getDepositAddressReqRate         = 45 // This is not specified just inputted above
	withdrawalReqRate                = 45 // This is not specified just inputted above
	getMovementsReqRate              = 45
	getAlertListReqRate              = 45
	setPriceAlertReqRate             = 45 // This is not specified just inputted above
	deletePriceAlertReqRate          = 45 // This is not specified just inputted above
	getBalanceForOrdersOffersReqRate = 30
	userSettingsWriteReqRate         = 45 // This is not specified just inputted general count
	userSettingsReadReqRate          = 45
	userSettingsDeleteReqRate        = 45 // This is not specified just inputted above
	// Account V1 endpoints
	getAccountFeesReqRate    = 5
	getWithdrawalFeesReqRate = 5
	getAccountSummaryReqRate = 5 // This is not specified just inputted above
	newDepositAddressReqRate = 5 // This is not specified just inputted above
	getKeyPermissionsReqRate = 5 // This is not specified just inputted above
	getMarginInfoReqRate     = 5 // This is not specified just inputted above
	getAccountBalanceReqRate = 10
	walletTransferReqRate    = 10 // This is not specified just inputted above
	withdrawV1ReqRate        = 10 // This is not specified just inputted above
	orderV1ReqRate           = 10 // This is not specified just inputted above
	orderMultiReqRate        = 10 // This is not specified just inputted above
	statsV1ReqRate           = 10
	fundingbookReqRate       = 15
	lendsReqRate             = 15

	// Rate limit endpoint functionality declaration
	platformStatus request.EndpointLimit = iota
	tickerBatch
	tickerFunction
	tradeRateLimit
	orderbookFunction
	stats
	candle
	configs
	status
	liquid
	leaderBoard
	marketAveragePrice
	fx

	// Bitfinex rate limits - Authenticated
	// Wallets -
	accountWalletBalance
	accountWalletHistory
	// Orders -
	retrieveOrder
	submitOrder
	updateOrder
	cancelOrder
	orderBatch
	cancelBatch
	orderHistory
	getOrderTrades
	getTrades
	getLedgers
	// Positions -
	getAccountMarginInfo
	getActivePositions
	claimPosition
	getPositionHistory
	getPositionAudit
	updateCollateralOnPosition
	// Margin funding -
	getActiveFundingOffers
	submitFundingOffer
	cancelFundingOffer
	cancelAllFundingOffer
	closeFunding
	fundingAutoRenew
	keepFunding
	getOffersHistory
	getFundingLoans
	getFundingLoanHistory
	getFundingCredits
	getFundingCreditsHistory
	getFundingTrades
	getFundingInfo
	// Account actions
	getUserInfo
	transferBetweenWallets
	getDepositAddress
	withdrawal
	getMovements
	getAlertList
	setPriceAlert
	deletePriceAlert
	getBalanceForOrdersOffers
	userSettingsWrite
	userSettingsRead
	userSettingsDelete
	// Account V1 endpoints
	getAccountFees
	getWithdrawalFees
	getAccountSummary
	newDepositAddress
	getKeyPermissions
	getMarginInfo
	getAccountBalance
	walletTransfer
	withdrawV1
	orderV1
	orderMulti
	statsV1
	fundingbook
	lends
)

// package level rate limits for REST API
var packageRateLimits = request.RateLimitDefinitions{
	// Public
	platformStatus:       rateLimitPerMinute(platformStatusReqRate),
	tickerBatch:          rateLimitPerMinute(tickerBatchReqRate),
	tickerFunction:       rateLimitPerMinute(tickerReqRate),
	tradeRateLimit:       rateLimitPerMinute(tradeReqRate),
	orderbookFunction:    rateLimitPerMinute(orderbookReqRate),
	stats:                rateLimitPerMinute(statsReqRate),
	candle:               rateLimitPerMinute(candleReqRate),
	configs:              rateLimitPerMinute(configsReqRate),
	status:               rateLimitPerMinute(statusReqRate),
	liquid:               rateLimitPerMinute(liquidReqRate),
	leaderBoard:          rateLimitPerMinute(leaderBoardReqRate),
	marketAveragePrice:   rateLimitPerMinute(marketAveragePriceReqRate),
	fx:                   rateLimitPerMinute(fxReqRate),
	accountWalletBalance: rateLimitPerMinute(accountWalletBalanceReqRate),
	accountWalletHistory: rateLimitPerMinute(accountWalletHistoryReqRate),
	// Orders
	retrieveOrder:  rateLimitPerMinute(retrieveOrderReqRate),
	submitOrder:    rateLimitPerMinute(submitOrderReqRate),
	updateOrder:    rateLimitPerMinute(updateOrderReqRate),
	cancelOrder:    rateLimitPerMinute(cancelOrderReqRate),
	orderBatch:     rateLimitPerMinute(orderBatchReqRate),
	cancelBatch:    rateLimitPerMinute(cancelBatchReqRate),
	orderHistory:   rateLimitPerMinute(orderHistoryReqRate),
	getOrderTrades: rateLimitPerMinute(getOrderTradesReqRate),
	getTrades:      rateLimitPerMinute(getTradesReqRate),
	getLedgers:     rateLimitPerMinute(getLedgersReqRate),
	// Positions
	getAccountMarginInfo:       rateLimitPerMinute(getAccountMarginInfoReqRate),
	getActivePositions:         rateLimitPerMinute(getActivePositionsReqRate),
	claimPosition:              rateLimitPerMinute(claimPositionReqRate),
	getPositionHistory:         rateLimitPerMinute(getPositionHistoryReqRate),
	getPositionAudit:           rateLimitPerMinute(getPositionAuditReqRate),
	updateCollateralOnPosition: rateLimitPerMinute(updateCollateralOnPositionReqRate),
	// Margin funding
	getActiveFundingOffers:   rateLimitPerMinute(getActiveFundingOffersReqRate),
	submitFundingOffer:       rateLimitPerMinute(submitFundingOfferReqRate),
	cancelFundingOffer:       rateLimitPerMinute(cancelFundingOfferReqRate),
	cancelAllFundingOffer:    rateLimitPerMinute(cancelAllFundingOfferReqRate),
	closeFunding:             rateLimitPerMinute(closeFundingReqRate),
	fundingAutoRenew:         rateLimitPerMinute(fundingAutoRenewReqRate),
	keepFunding:              rateLimitPerMinute(keepFundingReqRate),
	getOffersHistory:         rateLimitPerMinute(getOffersHistoryReqRate),
	getFundingLoans:          rateLimitPerMinute(getFundingLoansReqRate),
	getFundingLoanHistory:    rateLimitPerMinute(getFundingLoanHistoryReqRate),
	getFundingCredits:        rateLimitPerMinute(getFundingCreditsReqRate),
	getFundingCreditsHistory: rateLimitPerMinute(getFundingCreditsHistoryReqRate),
	getFundingTrades:         rateLimitPerMinute(getFundingTradesReqRate),
	getFundingInfo:           rateLimitPerMinute(getFundingInfoReqRate),
	// Account actions
	getUserInfo:               rateLimitPerMinute(getUserInfoReqRate),
	transferBetweenWallets:    rateLimitPerMinute(transferBetweenWalletsReqRate),
	getDepositAddress:         rateLimitPerMinute(getDepositAddressReqRate),
	withdrawal:                rateLimitPerMinute(withdrawalReqRate),
	getMovements:              rateLimitPerMinute(getMovementsReqRate),
	getAlertList:              rateLimitPerMinute(getAlertListReqRate),
	setPriceAlert:             rateLimitPerMinute(setPriceAlertReqRate),
	deletePriceAlert:          rateLimitPerMinute(deletePriceAlertReqRate),
	getBalanceForOrdersOffers: rateLimitPerMinute(getBalanceForOrdersOffersReqRate),
	userSettingsWrite:         rateLimitPerMinute(userSettingsWriteReqRate),
	userSettingsRead:          rateLimitPerMinute(userSettingsReadReqRate),
	userSettingsDelete:        rateLimitPerMinute(userSettingsDeleteReqRate),
	// Account V1 endpoints
	getAccountFees:    rateLimitPerMinute(getAccountFeesReqRate),
	getWithdrawalFees: rateLimitPerMinute(getWithdrawalFeesReqRate),
	getAccountSummary: rateLimitPerMinute(getAccountSummaryReqRate),
	newDepositAddress: rateLimitPerMinute(newDepositAddressReqRate),
	getKeyPermissions: rateLimitPerMinute(getKeyPermissionsReqRate),
	getMarginInfo:     rateLimitPerMinute(getMarginInfoReqRate),
	getAccountBalance: rateLimitPerMinute(getAccountBalanceReqRate),
	walletTransfer:    rateLimitPerMinute(walletTransferReqRate),
	withdrawV1:        rateLimitPerMinute(withdrawV1ReqRate),
	orderV1:           rateLimitPerMinute(orderV1ReqRate),
	orderMulti:        rateLimitPerMinute(orderMultiReqRate),
	statsV1:           rateLimitPerMinute(statsV1ReqRate),
	fundingbook:       rateLimitPerMinute(fundingbookReqRate),
	lends:             rateLimitPerMinute(lendsReqRate),
}

func rateLimitPerMinute(limit int) *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(oneMinuteInterval, limit, 1)
}

// GetRateLimit returns the rate limit for the exchange
func GetRateLimit() request.RateLimitDefinitions {
	return packageRateLimits
}
