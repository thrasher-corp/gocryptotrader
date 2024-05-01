package bitfinex

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// Bitfinex rate limits - Public
	requestLimitInterval      = time.Minute
	platformStatusReqRate     = 15
	tickerBatchReqRate        = 30
	tickerReqRate             = 30
	tradeReqRate              = 30
	orderbookReqRate          = 30
	statsReqRate              = 90
	candleReqRate             = 60
	configsReqRate            = 15
	statusReqRate             = 15 // This is not specified just inputted WCS
	liquidReqRate             = 15 // This is not specified just inputted WCS
	leaderBoardReqRate        = 90
	marketAveragePriceReqRate = 20
	fxReqRate                 = 20

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
	lendsReqRate             = 30

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

// GetRateLimit returns the rate limit for the exchange
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		platformStatus:       request.NewRateLimitWithWeight(requestLimitInterval, platformStatusReqRate, 1),
		tickerBatch:          request.NewRateLimitWithWeight(requestLimitInterval, tickerBatchReqRate, 1),
		tickerFunction:       request.NewRateLimitWithWeight(requestLimitInterval, tickerReqRate, 1),
		tradeRateLimit:       request.NewRateLimitWithWeight(requestLimitInterval, tradeReqRate, 1),
		orderbookFunction:    request.NewRateLimitWithWeight(requestLimitInterval, orderbookReqRate, 1),
		stats:                request.NewRateLimitWithWeight(requestLimitInterval, statsReqRate, 1),
		candle:               request.NewRateLimitWithWeight(requestLimitInterval, candleReqRate, 1),
		configs:              request.NewRateLimitWithWeight(requestLimitInterval, configsReqRate, 1),
		status:               request.NewRateLimitWithWeight(requestLimitInterval, statusReqRate, 1),
		liquid:               request.NewRateLimitWithWeight(requestLimitInterval, liquidReqRate, 1),
		leaderBoard:          request.NewRateLimitWithWeight(requestLimitInterval, leaderBoardReqRate, 1),
		marketAveragePrice:   request.NewRateLimitWithWeight(requestLimitInterval, marketAveragePriceReqRate, 1),
		fx:                   request.NewRateLimitWithWeight(requestLimitInterval, fxReqRate, 1),
		accountWalletBalance: request.NewRateLimitWithWeight(requestLimitInterval, accountWalletBalanceReqRate, 1),
		accountWalletHistory: request.NewRateLimitWithWeight(requestLimitInterval, accountWalletHistoryReqRate, 1),
		// Orders -
		retrieveOrder:  request.NewRateLimitWithWeight(requestLimitInterval, retrieveOrderReqRate, 1),
		submitOrder:    request.NewRateLimitWithWeight(requestLimitInterval, submitOrderReqRate, 1),
		updateOrder:    request.NewRateLimitWithWeight(requestLimitInterval, updateOrderReqRate, 1),
		cancelOrder:    request.NewRateLimitWithWeight(requestLimitInterval, cancelOrderReqRate, 1),
		orderBatch:     request.NewRateLimitWithWeight(requestLimitInterval, orderBatchReqRate, 1),
		cancelBatch:    request.NewRateLimitWithWeight(requestLimitInterval, cancelBatchReqRate, 1),
		orderHistory:   request.NewRateLimitWithWeight(requestLimitInterval, orderHistoryReqRate, 1),
		getOrderTrades: request.NewRateLimitWithWeight(requestLimitInterval, getOrderTradesReqRate, 1),
		getTrades:      request.NewRateLimitWithWeight(requestLimitInterval, getTradesReqRate, 1),
		getLedgers:     request.NewRateLimitWithWeight(requestLimitInterval, getLedgersReqRate, 1),
		// Positions -
		getAccountMarginInfo:       request.NewRateLimitWithWeight(requestLimitInterval, getAccountMarginInfoReqRate, 1),
		getActivePositions:         request.NewRateLimitWithWeight(requestLimitInterval, getActivePositionsReqRate, 1),
		claimPosition:              request.NewRateLimitWithWeight(requestLimitInterval, claimPositionReqRate, 1),
		getPositionHistory:         request.NewRateLimitWithWeight(requestLimitInterval, getPositionAuditReqRate, 1),
		getPositionAudit:           request.NewRateLimitWithWeight(requestLimitInterval, getPositionAuditReqRate, 1),
		updateCollateralOnPosition: request.NewRateLimitWithWeight(requestLimitInterval, updateCollateralOnPositionReqRate, 1),
		// Margin funding -
		getActiveFundingOffers:   request.NewRateLimitWithWeight(requestLimitInterval, getActiveFundingOffersReqRate, 1),
		submitFundingOffer:       request.NewRateLimitWithWeight(requestLimitInterval, submitFundingOfferReqRate, 1),
		cancelFundingOffer:       request.NewRateLimitWithWeight(requestLimitInterval, cancelFundingOfferReqRate, 1),
		cancelAllFundingOffer:    request.NewRateLimitWithWeight(requestLimitInterval, cancelAllFundingOfferReqRate, 1),
		closeFunding:             request.NewRateLimitWithWeight(requestLimitInterval, closeFundingReqRate, 1),
		fundingAutoRenew:         request.NewRateLimitWithWeight(requestLimitInterval, fundingAutoRenewReqRate, 1),
		keepFunding:              request.NewRateLimitWithWeight(requestLimitInterval, keepFundingReqRate, 1),
		getOffersHistory:         request.NewRateLimitWithWeight(requestLimitInterval, getOffersHistoryReqRate, 1),
		getFundingLoans:          request.NewRateLimitWithWeight(requestLimitInterval, getOffersHistoryReqRate, 1),
		getFundingLoanHistory:    request.NewRateLimitWithWeight(requestLimitInterval, getFundingLoanHistoryReqRate, 1),
		getFundingCredits:        request.NewRateLimitWithWeight(requestLimitInterval, getFundingCreditsReqRate, 1),
		getFundingCreditsHistory: request.NewRateLimitWithWeight(requestLimitInterval, getFundingCreditsHistoryReqRate, 1),
		getFundingTrades:         request.NewRateLimitWithWeight(requestLimitInterval, getFundingTradesReqRate, 1),
		getFundingInfo:           request.NewRateLimitWithWeight(requestLimitInterval, getFundingInfoReqRate, 1),
		// Account actions
		getUserInfo:               request.NewRateLimitWithWeight(requestLimitInterval, getUserInfoReqRate, 1),
		transferBetweenWallets:    request.NewRateLimitWithWeight(requestLimitInterval, transferBetweenWalletsReqRate, 1),
		getDepositAddress:         request.NewRateLimitWithWeight(requestLimitInterval, getDepositAddressReqRate, 1),
		withdrawal:                request.NewRateLimitWithWeight(requestLimitInterval, withdrawalReqRate, 1),
		getMovements:              request.NewRateLimitWithWeight(requestLimitInterval, getMovementsReqRate, 1),
		getAlertList:              request.NewRateLimitWithWeight(requestLimitInterval, getAlertListReqRate, 1),
		setPriceAlert:             request.NewRateLimitWithWeight(requestLimitInterval, setPriceAlertReqRate, 1),
		deletePriceAlert:          request.NewRateLimitWithWeight(requestLimitInterval, deletePriceAlertReqRate, 1),
		getBalanceForOrdersOffers: request.NewRateLimitWithWeight(requestLimitInterval, getBalanceForOrdersOffersReqRate, 1),
		userSettingsWrite:         request.NewRateLimitWithWeight(requestLimitInterval, userSettingsWriteReqRate, 1),
		userSettingsRead:          request.NewRateLimitWithWeight(requestLimitInterval, userSettingsReadReqRate, 1),
		userSettingsDelete:        request.NewRateLimitWithWeight(requestLimitInterval, userSettingsDeleteReqRate, 1),
		// Account V1 endpoints
		getAccountFees:    request.NewRateLimitWithWeight(requestLimitInterval, getAccountFeesReqRate, 1),
		getWithdrawalFees: request.NewRateLimitWithWeight(requestLimitInterval, getWithdrawalFeesReqRate, 1),
		getAccountSummary: request.NewRateLimitWithWeight(requestLimitInterval, getAccountSummaryReqRate, 1),
		newDepositAddress: request.NewRateLimitWithWeight(requestLimitInterval, newDepositAddressReqRate, 1),
		getKeyPermissions: request.NewRateLimitWithWeight(requestLimitInterval, getKeyPermissionsReqRate, 1),
		getMarginInfo:     request.NewRateLimitWithWeight(requestLimitInterval, getMarginInfoReqRate, 1),
		getAccountBalance: request.NewRateLimitWithWeight(requestLimitInterval, getAccountBalanceReqRate, 1),
		walletTransfer:    request.NewRateLimitWithWeight(requestLimitInterval, walletTransferReqRate, 1),
		withdrawV1:        request.NewRateLimitWithWeight(requestLimitInterval, withdrawV1ReqRate, 1),
		orderV1:           request.NewRateLimitWithWeight(requestLimitInterval, orderV1ReqRate, 1),
		orderMulti:        request.NewRateLimitWithWeight(requestLimitInterval, orderMultiReqRate, 1),
		statsV1:           request.NewRateLimitWithWeight(requestLimitInterval, statsV1ReqRate, 1),
		fundingbook:       request.NewRateLimitWithWeight(requestLimitInterval, fundingbookReqRate, 1),
		lends:             request.NewRateLimitWithWeight(requestLimitInterval, lendsReqRate, 1),
	}
}
