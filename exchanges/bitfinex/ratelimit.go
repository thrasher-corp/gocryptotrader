package bitfinex

import (
	"context"
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
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

// RateLimit implements the rate.Limiter interface
type RateLimit struct {
	PlatformStatus       *rate.Limiter
	TickerBatch          *rate.Limiter
	Ticker               *rate.Limiter
	Trade                *rate.Limiter
	Orderbook            *rate.Limiter
	Stats                *rate.Limiter
	Candle               *rate.Limiter
	Configs              *rate.Limiter
	Status               *rate.Limiter
	Liquid               *rate.Limiter
	LeaderBoard          *rate.Limiter
	MarketAveragePrice   *rate.Limiter
	Fx                   *rate.Limiter
	AccountWalletBalance *rate.Limiter
	AccountWalletHistory *rate.Limiter
	// Orders -
	RetrieveOrder  *rate.Limiter
	SubmitOrder    *rate.Limiter
	UpdateOrder    *rate.Limiter
	CancelOrder    *rate.Limiter
	OrderBatch     *rate.Limiter
	CancelBatch    *rate.Limiter
	OrderHistory   *rate.Limiter
	GetOrderTrades *rate.Limiter
	GetTrades      *rate.Limiter
	GetLedgers     *rate.Limiter
	// Positions -
	GetAccountMarginInfo       *rate.Limiter
	GetActivePositions         *rate.Limiter
	ClaimPosition              *rate.Limiter
	GetPositionHistory         *rate.Limiter
	GetPositionAudit           *rate.Limiter
	UpdateCollateralOnPosition *rate.Limiter
	// Margin funding -
	GetActiveFundingOffers   *rate.Limiter
	SubmitFundingOffer       *rate.Limiter
	CancelFundingOffer       *rate.Limiter
	CancelAllFundingOffer    *rate.Limiter
	CloseFunding             *rate.Limiter
	FundingAutoRenew         *rate.Limiter
	KeepFunding              *rate.Limiter
	GetOffersHistory         *rate.Limiter
	GetFundingLoans          *rate.Limiter
	GetFundingLoanHistory    *rate.Limiter
	GetFundingCredits        *rate.Limiter
	GetFundingCreditsHistory *rate.Limiter
	GetFundingTrades         *rate.Limiter
	GetFundingInfo           *rate.Limiter
	// Account actions
	GetUserInfo               *rate.Limiter
	TransferBetweenWallets    *rate.Limiter
	GetDepositAddress         *rate.Limiter
	Withdrawal                *rate.Limiter
	GetMovements              *rate.Limiter
	GetAlertList              *rate.Limiter
	SetPriceAlert             *rate.Limiter
	DeletePriceAlert          *rate.Limiter
	GetBalanceForOrdersOffers *rate.Limiter
	UserSettingsWrite         *rate.Limiter
	UserSettingsRead          *rate.Limiter
	UserSettingsDelete        *rate.Limiter
	// Account V1 endpoints
	GetAccountFees    *rate.Limiter
	GetWithdrawalFees *rate.Limiter
	GetAccountSummary *rate.Limiter
	NewDepositAddress *rate.Limiter
	GetKeyPermissions *rate.Limiter
	GetMarginInfo     *rate.Limiter
	GetAccountBalance *rate.Limiter
	WalletTransfer    *rate.Limiter
	WithdrawV1        *rate.Limiter
	OrderV1           *rate.Limiter
	OrderMulti        *rate.Limiter
	StatsV1           *rate.Limiter
	Fundingbook       *rate.Limiter
	Lends             *rate.Limiter
}

// Limit limits outbound requests
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) (*rate.Limiter, int, error) {
	switch f {
	case platformStatus:
		return r.PlatformStatus, 1, nil
	case tickerBatch:
		return r.TickerBatch, 1, nil
	case tickerFunction:
		return r.Ticker, 1, nil
	case tradeRateLimit:
		return r.Trade, 1, nil
	case orderbookFunction:
		return r.Orderbook, 1, nil
	case stats:
		return r.Stats, 1, nil
	case candle:
		return r.Candle, 1, nil
	case configs:
		return r.Configs, 1, nil
	case status:
		return r.Stats, 1, nil
	case liquid:
		return r.Liquid, 1, nil
	case leaderBoard:
		return r.LeaderBoard, 1, nil
	case marketAveragePrice:
		return r.MarketAveragePrice, 1, nil
	case fx:
		return r.Fx, 1, nil
	case accountWalletBalance:
		return r.AccountWalletBalance, 1, nil
	case accountWalletHistory:
		return r.AccountWalletHistory, 1, nil
	case retrieveOrder:
		return r.RetrieveOrder, 1, nil
	case submitOrder:
		return r.SubmitOrder, 1, nil
	case updateOrder:
		return r.UpdateOrder, 1, nil
	case cancelOrder:
		return r.CancelOrder, 1, nil
	case orderBatch:
		return r.OrderBatch, 1, nil
	case cancelBatch:
		return r.CancelBatch, 1, nil
	case orderHistory:
		return r.OrderHistory, 1, nil
	case getOrderTrades:
		return r.GetOrderTrades, 1, nil
	case getTrades:
		return r.GetTrades, 1, nil
	case getLedgers:
		return r.GetLedgers, 1, nil
	case getAccountMarginInfo:
		return r.GetAccountMarginInfo, 1, nil
	case getActivePositions:
		return r.GetActivePositions, 1, nil
	case claimPosition:
		return r.ClaimPosition, 1, nil
	case getPositionHistory:
		return r.GetPositionHistory, 1, nil
	case getPositionAudit:
		return r.GetPositionAudit, 1, nil
	case updateCollateralOnPosition:
		return r.UpdateCollateralOnPosition, 1, nil
	case getActiveFundingOffers:
		return r.GetActiveFundingOffers, 1, nil
	case submitFundingOffer:
		return r.SubmitFundingOffer, 1, nil
	case cancelFundingOffer:
		return r.CancelFundingOffer, 1, nil
	case cancelAllFundingOffer:
		return r.CancelAllFundingOffer, 1, nil
	case closeFunding:
		return r.CloseFunding, 1, nil
	case fundingAutoRenew:
		return r.FundingAutoRenew, 1, nil
	case keepFunding:
		return r.KeepFunding, 1, nil
	case getOffersHistory:
		return r.GetOffersHistory, 1, nil
	case getFundingLoans:
		return r.GetFundingLoans, 1, nil
	case getFundingLoanHistory:
		return r.GetFundingLoanHistory, 1, nil
	case getFundingCredits:
		return r.GetFundingCredits, 1, nil
	case getFundingCreditsHistory:
		return r.GetFundingCreditsHistory, 1, nil
	case getFundingTrades:
		return r.GetFundingTrades, 1, nil
	case getFundingInfo:
		return r.GetFundingInfo, 1, nil
	case getUserInfo:
		return r.GetUserInfo, 1, nil
	case transferBetweenWallets:
		return r.TransferBetweenWallets, 1, nil
	case getDepositAddress:
		return r.GetDepositAddress, 1, nil
	case withdrawal:
		return r.Withdrawal, 1, nil
	case getMovements:
		return r.GetMovements, 1, nil
	case getAlertList:
		return r.GetAlertList, 1, nil
	case setPriceAlert:
		return r.SetPriceAlert, 1, nil
	case deletePriceAlert:
		return r.DeletePriceAlert, 1, nil
	case getBalanceForOrdersOffers:
		return r.GetBalanceForOrdersOffers, 1, nil
	case userSettingsWrite:
		return r.UserSettingsWrite, 1, nil
	case userSettingsRead:
		return r.UserSettingsRead, 1, nil
	case userSettingsDelete:
		return r.UserSettingsDelete, 1, nil
		//  Bitfinex V1 API
	case getAccountFees:
		return r.GetAccountFees, 1, nil
	case getWithdrawalFees:
		return r.GetWithdrawalFees, 1, nil
	case getAccountSummary:
		return r.GetAccountSummary, 1, nil
	case newDepositAddress:
		return r.NewDepositAddress, 1, nil
	case getKeyPermissions:
		return r.GetKeyPermissions, 1, nil
	case getMarginInfo:
		return r.GetMarginInfo, 1, nil
	case getAccountBalance:
		return r.GetAccountBalance, 1, nil
	case walletTransfer:
		return r.WalletTransfer, 1, nil
	case withdrawV1:
		return r.WithdrawV1, 1, nil
	case orderV1:
		return r.OrderV1, 1, nil
	case orderMulti:
		return r.OrderMulti, 1, nil
	case statsV1:
		return r.Stats, 1, nil
	case fundingbook:
		return r.Fundingbook, 1, nil
	case lends:
		return r.Lends, 1, nil
	default:
		return nil, 0, errors.New("endpoint rate limit functionality not found")
	}
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		PlatformStatus:       request.NewRateLimit(requestLimitInterval, platformStatusReqRate),
		TickerBatch:          request.NewRateLimit(requestLimitInterval, tickerBatchReqRate),
		Ticker:               request.NewRateLimit(requestLimitInterval, tickerReqRate),
		Trade:                request.NewRateLimit(requestLimitInterval, tradeReqRate),
		Orderbook:            request.NewRateLimit(requestLimitInterval, orderbookReqRate),
		Stats:                request.NewRateLimit(requestLimitInterval, statsReqRate),
		Candle:               request.NewRateLimit(requestLimitInterval, candleReqRate),
		Configs:              request.NewRateLimit(requestLimitInterval, configsReqRate),
		Status:               request.NewRateLimit(requestLimitInterval, statusReqRate),
		Liquid:               request.NewRateLimit(requestLimitInterval, liquidReqRate),
		LeaderBoard:          request.NewRateLimit(requestLimitInterval, leaderBoardReqRate),
		MarketAveragePrice:   request.NewRateLimit(requestLimitInterval, marketAveragePriceReqRate),
		Fx:                   request.NewRateLimit(requestLimitInterval, fxReqRate),
		AccountWalletBalance: request.NewRateLimit(requestLimitInterval, accountWalletBalanceReqRate),
		AccountWalletHistory: request.NewRateLimit(requestLimitInterval, accountWalletHistoryReqRate),
		// Orders -
		RetrieveOrder:  request.NewRateLimit(requestLimitInterval, retrieveOrderReqRate),
		SubmitOrder:    request.NewRateLimit(requestLimitInterval, submitOrderReqRate),
		UpdateOrder:    request.NewRateLimit(requestLimitInterval, updateOrderReqRate),
		CancelOrder:    request.NewRateLimit(requestLimitInterval, cancelOrderReqRate),
		OrderBatch:     request.NewRateLimit(requestLimitInterval, orderBatchReqRate),
		CancelBatch:    request.NewRateLimit(requestLimitInterval, cancelBatchReqRate),
		OrderHistory:   request.NewRateLimit(requestLimitInterval, orderHistoryReqRate),
		GetOrderTrades: request.NewRateLimit(requestLimitInterval, getOrderTradesReqRate),
		GetTrades:      request.NewRateLimit(requestLimitInterval, getTradesReqRate),
		GetLedgers:     request.NewRateLimit(requestLimitInterval, getLedgersReqRate),
		// Positions -
		GetAccountMarginInfo:       request.NewRateLimit(requestLimitInterval, getAccountMarginInfoReqRate),
		GetActivePositions:         request.NewRateLimit(requestLimitInterval, getActivePositionsReqRate),
		ClaimPosition:              request.NewRateLimit(requestLimitInterval, claimPositionReqRate),
		GetPositionHistory:         request.NewRateLimit(requestLimitInterval, getPositionAuditReqRate),
		GetPositionAudit:           request.NewRateLimit(requestLimitInterval, getPositionAuditReqRate),
		UpdateCollateralOnPosition: request.NewRateLimit(requestLimitInterval, updateCollateralOnPositionReqRate),
		// Margin funding -
		GetActiveFundingOffers:   request.NewRateLimit(requestLimitInterval, getActiveFundingOffersReqRate),
		SubmitFundingOffer:       request.NewRateLimit(requestLimitInterval, submitFundingOfferReqRate),
		CancelFundingOffer:       request.NewRateLimit(requestLimitInterval, cancelFundingOfferReqRate),
		CancelAllFundingOffer:    request.NewRateLimit(requestLimitInterval, cancelAllFundingOfferReqRate),
		CloseFunding:             request.NewRateLimit(requestLimitInterval, closeFundingReqRate),
		FundingAutoRenew:         request.NewRateLimit(requestLimitInterval, fundingAutoRenewReqRate),
		KeepFunding:              request.NewRateLimit(requestLimitInterval, keepFundingReqRate),
		GetOffersHistory:         request.NewRateLimit(requestLimitInterval, getOffersHistoryReqRate),
		GetFundingLoans:          request.NewRateLimit(requestLimitInterval, getOffersHistoryReqRate),
		GetFundingLoanHistory:    request.NewRateLimit(requestLimitInterval, getFundingLoanHistoryReqRate),
		GetFundingCredits:        request.NewRateLimit(requestLimitInterval, getFundingCreditsReqRate),
		GetFundingCreditsHistory: request.NewRateLimit(requestLimitInterval, getFundingCreditsHistoryReqRate),
		GetFundingTrades:         request.NewRateLimit(requestLimitInterval, getFundingTradesReqRate),
		GetFundingInfo:           request.NewRateLimit(requestLimitInterval, getFundingInfoReqRate),
		// Account actions
		GetUserInfo:               request.NewRateLimit(requestLimitInterval, getUserInfoReqRate),
		TransferBetweenWallets:    request.NewRateLimit(requestLimitInterval, transferBetweenWalletsReqRate),
		GetDepositAddress:         request.NewRateLimit(requestLimitInterval, getDepositAddressReqRate),
		Withdrawal:                request.NewRateLimit(requestLimitInterval, withdrawalReqRate),
		GetMovements:              request.NewRateLimit(requestLimitInterval, getMovementsReqRate),
		GetAlertList:              request.NewRateLimit(requestLimitInterval, getAlertListReqRate),
		SetPriceAlert:             request.NewRateLimit(requestLimitInterval, setPriceAlertReqRate),
		DeletePriceAlert:          request.NewRateLimit(requestLimitInterval, deletePriceAlertReqRate),
		GetBalanceForOrdersOffers: request.NewRateLimit(requestLimitInterval, getBalanceForOrdersOffersReqRate),
		UserSettingsWrite:         request.NewRateLimit(requestLimitInterval, userSettingsWriteReqRate),
		UserSettingsRead:          request.NewRateLimit(requestLimitInterval, userSettingsReadReqRate),
		UserSettingsDelete:        request.NewRateLimit(requestLimitInterval, userSettingsDeleteReqRate),
		// Account V1 endpoints
		GetAccountFees:    request.NewRateLimit(requestLimitInterval, getAccountFeesReqRate),
		GetWithdrawalFees: request.NewRateLimit(requestLimitInterval, getWithdrawalFeesReqRate),
		GetAccountSummary: request.NewRateLimit(requestLimitInterval, getAccountSummaryReqRate),
		NewDepositAddress: request.NewRateLimit(requestLimitInterval, newDepositAddressReqRate),
		GetKeyPermissions: request.NewRateLimit(requestLimitInterval, getKeyPermissionsReqRate),
		GetMarginInfo:     request.NewRateLimit(requestLimitInterval, getMarginInfoReqRate),
		GetAccountBalance: request.NewRateLimit(requestLimitInterval, getAccountBalanceReqRate),
		WalletTransfer:    request.NewRateLimit(requestLimitInterval, walletTransferReqRate),
		WithdrawV1:        request.NewRateLimit(requestLimitInterval, withdrawV1ReqRate),
		OrderV1:           request.NewRateLimit(requestLimitInterval, orderV1ReqRate),
		OrderMulti:        request.NewRateLimit(requestLimitInterval, orderMultiReqRate),
		StatsV1:           request.NewRateLimit(requestLimitInterval, statsV1ReqRate),
		Fundingbook:       request.NewRateLimit(requestLimitInterval, fundingbookReqRate),
		Lends:             request.NewRateLimit(requestLimitInterval, lendsReqRate),
	}
}
