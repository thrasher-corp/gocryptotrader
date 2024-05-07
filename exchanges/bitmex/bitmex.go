package bitmex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Bitmex is the overarching type across this package
type Bitmex struct {
	exchange.Base
}

const (
	bitmexAPIVersion    = "v1"
	bitmexAPIURL        = "https://www.bitmex.com/api/v1"
	bitmexAPItestnetURL = "https://testnet.bitmex.com/api/v1"
	tradeBaseURL        = "https://www.bitmex.com/app/trade/"

	// Public endpoints
	bitmexEndpointAnnouncement              = "/announcement"
	bitmexEndpointAnnouncementUrgent        = "/announcement/urgent"
	bitmexEndpointOrderbookL2               = "/orderBook/L2"
	bitmexEndpointTrollbox                  = "/chat"
	bitmexEndpointTrollboxChannels          = "/chat/channels"
	bitmexEndpointTrollboxConnected         = "/chat/connected"
	bitmexEndpointFundingHistory            = "/funding?"
	bitmexEndpointInstruments               = "/instrument"
	bitmexEndpointActiveInstruments         = "/instrument/active"
	bitmexEndpointActiveAndIndexInstruments = "/instrument/activeAndIndices"
	bitmexEndpointActiveIntervals           = "/instrument/activeIntervals"
	bitmexEndpointCompositeIndex            = "/instrument/compositeIndex"
	bitmexEndpointIndices                   = "/instrument/indices"
	bitmexEndpointInsuranceHistory          = "/insurance"
	bitmexEndpointLiquidation               = "/liquidation"
	bitmexEndpointLeader                    = "/leaderboard"
	bitmexEndpointAlias                     = "/leaderboard/name"
	bitmexEndpointQuote                     = "/quote"
	bitmexEndpointQuoteBucketed             = "/quote/bucketed"
	bitmexEndpointSettlement                = "/settlement"
	bitmexEndpointStats                     = "/stats"
	bitmexEndpointStatsHistory              = "/stats/history"
	bitmexEndpointStatsSummary              = "/stats/historyUSD"
	bitmexEndpointTrade                     = "/trade"
	bitmexEndpointTradeBucketed             = "/trade/bucketed"
	bitmexEndpointUserCheckReferralCode     = "/user/checkReferralCode"
	bitmexEndpointUserMinWithdrawalFee      = "/user/minWithdrawalFee"

	// Authenticated endpoints
	bitmexEndpointAPIkeys               = "/apiKey"
	bitmexEndpointDisableAPIkey         = "/apiKey/disable" //nolint:gosec // false positive
	bitmexEndpointEnableAPIkey          = "/apiKey/enable"  //nolint:gosec // false positive
	bitmexEndpointTrollboxSend          = "/chat"
	bitmexEndpointExecution             = "/execution"
	bitmexEndpointExecutionTradeHistory = "/execution/tradeHistory"
	bitmexEndpointNotifications         = "/notification"
	bitmexEndpointOrder                 = "/order"
	bitmexEndpointCancelAllOrders       = "/order/all"
	bitmexEndpointBulk                  = "/order/bulk"
	bitmexEndpointCancelOrderAfter      = "/order/cancelAllAfter"
	bitmexEndpointClosePosition         = "/order/closePosition"
	bitmexEndpointPosition              = "/position"
	bitmexEndpointIsolatePosition       = "/position/isolate"
	bitmexEndpointLeveragePosition      = "/position/leverage"
	bitmexEndpointAdjustRiskLimit       = "/position/riskLimit"
	bitmexEndpointTransferMargin        = "/position/transferMargin"
	bitmexEndpointUser                  = "/user"
	bitmexEndpointUserAffiliate         = "/user/affiliateStatus"
	bitmexEndpointUserCancelWithdraw    = "/user/cancelWithdrawal"
	bitmexEndpointUserCommision         = "/user/commission"
	bitmexEndpointUserConfirmEmail      = "/user/confirmEmail"
	bitmexEndpointUserConfirmTFA        = "/user/confirmEnableTFA"
	bitmexEndpointUserConfirmWithdrawal = "/user/confirmWithdrawal"
	bitmexEndpointUserDepositAddress    = "/user/depositAddress"
	bitmexEndpointUserDisableTFA        = "/user/disableTFA"
	bitmexEndpointUserLogout            = "/user/logout"
	bitmexEndpointUserLogoutAll         = "/user/logoutAll"
	bitmexEndpointUserMargin            = "/user/margin"
	bitmexEndpointUserPreferences       = "/user/preferences"
	bitmexEndpointUserRequestTFA        = "/user/requestEnableTFA"
	bitmexEndpointUserWallet            = "/user/wallet"
	bitmexEndpointUserWalletHistory     = "/user/walletHistory"
	bitmexEndpointUserWalletSummary     = "/user/walletSummary"
	bitmexEndpointUserRequestWithdraw   = "/user/requestWithdrawal"

	constSatoshiBTC = 1e-08

	// ContractPerpetual perpetual contract type
	ContractPerpetual = iota
	// ContractFutures futures contract type
	ContractFutures
	// ContractDownsideProfit downside profit contract type
	ContractDownsideProfit
	// ContractUpsideProfit upside profit contract type
	ContractUpsideProfit

	perpetualContractID         = "FFWCSX"
	spotID                      = "IFXXXP"
	futuresID                   = "FFCCSX"
	bitMEXBasketIndexID         = "MRBXXX"
	bitMEXPriceIndexID          = "MRCXXX"
	bitMEXLendingPremiumIndexID = "MRRXXX"
	bitMEXVolatilityIndexID     = "MRIXXX"
)

// GetAnnouncement returns the general announcements from Bitmex
func (b *Bitmex) GetAnnouncement(ctx context.Context) ([]Announcement, error) {
	var announcement []Announcement

	return announcement, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointAnnouncement,
		nil,
		&announcement)
}

// GetUrgentAnnouncement returns an urgent announcement for your account
func (b *Bitmex) GetUrgentAnnouncement(ctx context.Context) ([]Announcement, error) {
	var announcement []Announcement

	return announcement, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointAnnouncementUrgent,
		nil,
		&announcement)
}

// GetAPIKeys returns the APIkeys from bitmex
func (b *Bitmex) GetAPIKeys(ctx context.Context) ([]APIKey, error) {
	var keys []APIKey

	return keys, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointAPIkeys,
		nil,
		&keys)
}

// RemoveAPIKey removes an Apikey from the bitmex trading engine
func (b *Bitmex) RemoveAPIKey(ctx context.Context, params APIKeyParams) (bool, error) {
	var keyDeleted bool

	return keyDeleted, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete,
		bitmexEndpointAPIkeys,
		&params,
		&keyDeleted)
}

// DisableAPIKey disables an Apikey from the bitmex trading engine
func (b *Bitmex) DisableAPIKey(ctx context.Context, params APIKeyParams) (APIKey, error) {
	var keyInfo APIKey

	return keyInfo, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointDisableAPIkey,
		&params,
		&keyInfo)
}

// EnableAPIKey enables an Apikey from the bitmex trading engine
func (b *Bitmex) EnableAPIKey(ctx context.Context, params APIKeyParams) (APIKey, error) {
	var keyInfo APIKey

	return keyInfo, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointEnableAPIkey,
		&params,
		&keyInfo)
}

// GetTrollboxMessages returns messages from the bitmex trollbox
func (b *Bitmex) GetTrollboxMessages(ctx context.Context, params ChatGetParams) ([]Chat, error) {
	var messages []Chat

	return messages, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointTrollbox, &params, &messages)
}

// SendTrollboxMessage sends a message to the bitmex trollbox
func (b *Bitmex) SendTrollboxMessage(ctx context.Context, params ChatSendParams) ([]Chat, error) {
	var messages []Chat

	return messages, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointTrollboxSend,
		&params,
		&messages)
}

// GetTrollboxChannels the channels from the bitmex trollbox
func (b *Bitmex) GetTrollboxChannels(ctx context.Context) ([]ChatChannel, error) {
	var channels []ChatChannel

	return channels, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointTrollboxChannels,
		nil,
		&channels)
}

// GetTrollboxConnectedUsers the channels from the bitmex trollbox
func (b *Bitmex) GetTrollboxConnectedUsers(ctx context.Context) (ConnectedUsers, error) {
	var users ConnectedUsers

	return users, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointTrollboxConnected, nil, &users)
}

// GetAccountExecutions returns all raw transactions, which includes order
// opening and cancellation, and order status changes. It can be quite noisy.
// More focused information is available at /execution/tradeHistory.
func (b *Bitmex) GetAccountExecutions(ctx context.Context, params *GenericRequestParams) ([]Execution, error) {
	var executionList []Execution

	return executionList, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointExecution,
		params,
		&executionList)
}

// GetAccountExecutionTradeHistory returns all balance-affecting executions.
// This includes each trade, insurance charge, and settlement.
func (b *Bitmex) GetAccountExecutionTradeHistory(ctx context.Context, params *GenericRequestParams) ([]Execution, error) {
	var tradeHistory []Execution

	return tradeHistory, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointExecutionTradeHistory,
		params,
		&tradeHistory)
}

// GetFullFundingHistory returns funding history
func (b *Bitmex) GetFullFundingHistory(ctx context.Context, symbol, count, filter, columns, start string, reverse bool, startTime, endTime time.Time) ([]Funding, error) {
	var fundingHistory []Funding
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if count != "" {
		params.Set("count", count)
	}
	if filter != "" {
		params.Set("filter", filter)
	}
	if columns != "" {
		params.Set("columns", columns)
	}
	if !startTime.IsZero() {
		params.Set("start", start)
	}
	params.Set("reverse", "true")
	if !reverse {
		params.Set("reverse", "false")
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", startTime.Format(time.RFC3339))
		params.Set("endTime", endTime.Format(time.RFC3339))
	}
	return fundingHistory, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointFundingHistory+params.Encode(),
		nil,
		&fundingHistory)
}

// GetInstrument returns instrument data
func (b *Bitmex) GetInstrument(ctx context.Context, params *GenericRequestParams) ([]Instrument, error) {
	var instruments []Instrument

	return instruments, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointInstruments,
		params,
		&instruments)
}

// GetInstruments returns instrument data
func (b *Bitmex) GetInstruments(ctx context.Context, params *GenericRequestParams) ([]Instrument, error) {
	var instruments []Instrument

	return instruments, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointInstruments,
		params,
		&instruments)
}

// GetActiveInstruments returns active instruments
func (b *Bitmex) GetActiveInstruments(ctx context.Context, params *GenericRequestParams) ([]Instrument, error) {
	var activeInstruments []Instrument

	return activeInstruments, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointActiveInstruments,
		params,
		&activeInstruments)
}

// GetActiveAndIndexInstruments returns all active instruments and all indices
func (b *Bitmex) GetActiveAndIndexInstruments(ctx context.Context) ([]Instrument, error) {
	var activeAndIndices []Instrument

	return activeAndIndices,
		b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointActiveAndIndexInstruments,
			nil,
			&activeAndIndices)
}

// GetActiveIntervals returns funding history
func (b *Bitmex) GetActiveIntervals(ctx context.Context) (InstrumentInterval, error) {
	var interval InstrumentInterval

	return interval, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointActiveIntervals,
		nil,
		&interval)
}

// GetCompositeIndex returns composite index
func (b *Bitmex) GetCompositeIndex(ctx context.Context, symbol, count, filter, columns, start, reverse string, startTime, endTime time.Time) ([]IndexComposite, error) {
	var compositeIndices []IndexComposite
	params := url.Values{}
	params.Set("symbol", symbol)
	if count != "" {
		params.Set("count", count)
	}
	if filter != "" {
		params.Set("filter", filter)
	}
	if columns != "" {
		params.Set("columns", columns)
	}
	if start != "" {
		params.Set("start", start)
	}
	if reverse != "" {
		params.Set("reverse", "true")
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", startTime.Format(time.RFC3339))
		params.Set("endTime", endTime.Format(time.RFC3339))
	}
	return compositeIndices, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointCompositeIndex+"?"+params.Encode(),
		nil,
		&compositeIndices)
}

// GetIndices returns all price indices
func (b *Bitmex) GetIndices(ctx context.Context) ([]Instrument, error) {
	var indices []Instrument

	return indices, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointIndices, nil, &indices)
}

// GetInsuranceFundHistory returns insurance fund history
func (b *Bitmex) GetInsuranceFundHistory(ctx context.Context, params *GenericRequestParams) ([]Insurance, error) {
	var history []Insurance

	return history, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointIndices, params, &history)
}

// GetLeaderboard returns leaderboard information
func (b *Bitmex) GetLeaderboard(ctx context.Context, params LeaderboardGetParams) ([]Leaderboard, error) {
	var leader []Leaderboard

	return leader, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointLeader, params, &leader)
}

// GetAliasOnLeaderboard returns your alias on the leaderboard
func (b *Bitmex) GetAliasOnLeaderboard(ctx context.Context) (Alias, error) {
	var alias Alias

	return alias, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointAlias, nil, &alias)
}

// GetLiquidationOrders returns liquidation orders
func (b *Bitmex) GetLiquidationOrders(ctx context.Context, params *GenericRequestParams) ([]Liquidation, error) {
	var orders []Liquidation

	return orders, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointLiquidation,
		params,
		&orders)
}

// GetCurrentNotifications returns your current notifications
func (b *Bitmex) GetCurrentNotifications(ctx context.Context) ([]Notification, error) {
	var notifications []Notification

	return notifications, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointNotifications,
		nil,
		&notifications)
}

// GetOrders returns all the orders, open and closed
func (b *Bitmex) GetOrders(ctx context.Context, params *OrdersRequest) ([]Order, error) {
	var orders []Order
	return orders, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointOrder,
		params,
		&orders)
}

// AmendOrder amends the quantity or price of an open order
func (b *Bitmex) AmendOrder(ctx context.Context, params *OrderAmendParams) (Order, error) {
	var order Order

	return order, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut,
		bitmexEndpointOrder,
		params,
		&order)
}

// CreateOrder creates a new order
func (b *Bitmex) CreateOrder(ctx context.Context, params *OrderNewParams) (Order, error) {
	var orderInfo Order

	return orderInfo, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointOrder,
		params,
		&orderInfo)
}

// CancelOrders cancels one or a batch of orders on the exchange and returns
// a cancelled order list
func (b *Bitmex) CancelOrders(ctx context.Context, params *OrderCancelParams) ([]Order, error) {
	var cancelledOrders []Order

	return cancelledOrders, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete,
		bitmexEndpointOrder,
		params,
		&cancelledOrders)
}

// CancelAllExistingOrders cancels all open orders on the exchange
func (b *Bitmex) CancelAllExistingOrders(ctx context.Context, params OrderCancelAllParams) ([]Order, error) {
	var cancelledOrders []Order

	return cancelledOrders, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete,
		bitmexEndpointCancelAllOrders,
		params,
		&cancelledOrders)
}

// AmendBulkOrders amends multiple orders for the same symbol
func (b *Bitmex) AmendBulkOrders(ctx context.Context, params OrderAmendBulkParams) ([]Order, error) {
	var amendedOrders []Order

	return amendedOrders, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut,
		bitmexEndpointBulk,
		params,
		&amendedOrders)
}

// CreateBulkOrders creates multiple orders for the same symbol
func (b *Bitmex) CreateBulkOrders(ctx context.Context, params OrderNewBulkParams) ([]Order, error) {
	var orders []Order

	return orders, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointBulk,
		params,
		&orders)
}

// CancelAllOrdersAfterTime closes all positions after a certain time period
func (b *Bitmex) CancelAllOrdersAfterTime(ctx context.Context, params OrderCancelAllAfterParams) ([]Order, error) {
	var cancelledOrder []Order

	return cancelledOrder, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointCancelOrderAfter,
		params,
		&cancelledOrder)
}

// ClosePosition closes a position WARNING deprecated use /order endpoint
func (b *Bitmex) ClosePosition(ctx context.Context, params OrderClosePositionParams) ([]Order, error) {
	var closedPositions []Order

	return closedPositions, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointOrder,
		params,
		&closedPositions)
}

// GetOrderbook returns layer two orderbook data
func (b *Bitmex) GetOrderbook(ctx context.Context, params OrderBookGetL2Params) ([]OrderBookL2, error) {
	var orderBooks []OrderBookL2

	return orderBooks, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointOrderbookL2,
		params,
		&orderBooks)
}

// GetPositions returns positions
func (b *Bitmex) GetPositions(ctx context.Context, params PositionGetParams) ([]Position, error) {
	var positions []Position

	return positions, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointPosition,
		params,
		&positions)
}

// IsolatePosition enables isolated margin or cross margin per-position
func (b *Bitmex) IsolatePosition(ctx context.Context, params PositionIsolateMarginParams) (Position, error) {
	var position Position

	return position, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointIsolatePosition,
		params,
		&position)
}

// LeveragePosition chooses leverage for a position
func (b *Bitmex) LeveragePosition(ctx context.Context, params PositionUpdateLeverageParams) (Position, error) {
	var position Position

	return position, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointLeveragePosition,
		params,
		&position)
}

// UpdateRiskLimit updates risk limit on a position
func (b *Bitmex) UpdateRiskLimit(ctx context.Context, params PositionUpdateRiskLimitParams) (Position, error) {
	var position Position

	return position, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointAdjustRiskLimit,
		params,
		&position)
}

// TransferMargin transfers equity in or out of a position
func (b *Bitmex) TransferMargin(ctx context.Context, params PositionTransferIsolatedMarginParams) (Position, error) {
	var position Position

	return position, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointTransferMargin,
		params,
		&position)
}

// GetQuotes returns quotations
func (b *Bitmex) GetQuotes(ctx context.Context, params *GenericRequestParams) ([]Quote, error) {
	var quotations []Quote

	return quotations, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointQuote,
		params,
		&quotations)
}

// GetQuotesByBuckets returns previous quotes in time buckets
func (b *Bitmex) GetQuotesByBuckets(ctx context.Context, params *QuoteGetBucketedParams) ([]Quote, error) {
	var quotations []Quote

	return quotations, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointQuoteBucketed,
		params,
		&quotations)
}

// GetSettlementHistory returns settlement history
func (b *Bitmex) GetSettlementHistory(ctx context.Context, params *GenericRequestParams) ([]Settlement, error) {
	var history []Settlement

	return history, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointSettlement,
		params,
		&history)
}

// GetStats returns exchange wide per series turnover and volume statistics
func (b *Bitmex) GetStats(ctx context.Context) ([]Stats, error) {
	var stats []Stats

	return stats, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointStats, nil, &stats)
}

// GetStatsHistorical historic stats
func (b *Bitmex) GetStatsHistorical(ctx context.Context) ([]StatsHistory, error) {
	var history []StatsHistory

	return history, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointStatsHistory, nil, &history)
}

// GetStatSummary returns the stats summary in USD terms
func (b *Bitmex) GetStatSummary(ctx context.Context) ([]StatsUSD, error) {
	var summary []StatsUSD

	return summary, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointStatsSummary, nil, &summary)
}

// GetTrade returns executed trades on the desk
func (b *Bitmex) GetTrade(ctx context.Context, params *GenericRequestParams) ([]Trade, error) {
	var trade []Trade

	return trade, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointTrade, params, &trade)
}

// GetPreviousTrades previous trade history in time buckets
func (b *Bitmex) GetPreviousTrades(ctx context.Context, params *TradeGetBucketedParams) ([]Trade, error) {
	var trade []Trade

	return trade, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointTradeBucketed,
		params,
		&trade)
}

// GetUserInfo returns your user information
func (b *Bitmex) GetUserInfo(ctx context.Context) (User, error) {
	var userInfo User

	return userInfo, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUser,
		nil,
		&userInfo)
}

// UpdateUserInfo updates user information
func (b *Bitmex) UpdateUserInfo(ctx context.Context, params *UserUpdateParams) (User, error) {
	var userInfo User

	return userInfo, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut,
		bitmexEndpointUser,
		params,
		&userInfo)
}

// GetAffiliateStatus returns your affiliate status
func (b *Bitmex) GetAffiliateStatus(ctx context.Context) (AffiliateStatus, error) {
	var status AffiliateStatus

	return status, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUserAffiliate,
		nil,
		&status)
}

// CancelWithdraw cancels a current withdrawal
func (b *Bitmex) CancelWithdraw(ctx context.Context, token string) (TransactionInfo, error) {
	var info TransactionInfo

	return info, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserCancelWithdraw,
		UserTokenParams{Token: token},
		&info)
}

// CheckReferalCode checks a code, will return a percentage eg 0.1 for 10% or
// if err a 404
func (b *Bitmex) CheckReferalCode(ctx context.Context, referralCode string) (float64, error) {
	var percentage float64

	return percentage, b.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointUserCheckReferralCode,
		UserCheckReferralCodeParams{ReferralCode: referralCode},
		&percentage)
}

// GetUserCommision returns your account's commission status.
func (b *Bitmex) GetUserCommision(ctx context.Context) (UserCommission, error) {
	var commissionInfo UserCommission

	return commissionInfo, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUserCommision,
		nil,
		&commissionInfo)
}

// ConfirmEmail confirms email address with a token
func (b *Bitmex) ConfirmEmail(ctx context.Context, token string) (ConfirmEmail, error) {
	var confirmation ConfirmEmail

	return confirmation, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUserConfirmEmail,
		UserTokenParams{Token: token},
		&confirmation)
}

// ConfirmTwoFactorAuth confirms 2FA for this account.
func (b *Bitmex) ConfirmTwoFactorAuth(ctx context.Context, token, typ string) (bool, error) {
	var working bool

	return working, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserConfirmTFA,
		UserConfirmTFAParams{Token: token, Type: typ},
		&working)
}

// ConfirmWithdrawal confirms a withdrawal
func (b *Bitmex) ConfirmWithdrawal(ctx context.Context, token string) (TransactionInfo, error) {
	var info TransactionInfo

	return info, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserCancelWithdraw,
		UserTokenParams{Token: token},
		&info)
}

// GetCryptoDepositAddress returns a deposit address for a cryptocurrency
func (b *Bitmex) GetCryptoDepositAddress(ctx context.Context, cryptoCurrency string) (string, error) {
	var address string
	if !strings.EqualFold(cryptoCurrency, currency.XBT.String()) {
		return "", fmt.Errorf("%v %w only bitcoin", cryptoCurrency, currency.ErrCurrencyNotSupported)
	}

	return address, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUserDepositAddress,
		UserCurrencyParams{Currency: "XBt"},
		&address)
}

// DisableTFA dsiables 2 factor authentication for your account
func (b *Bitmex) DisableTFA(ctx context.Context, token, typ string) (bool, error) {
	var disabled bool

	return disabled, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserDisableTFA,
		UserConfirmTFAParams{Token: token, Type: typ},
		&disabled)
}

// UserLogOut logs you out of BitMEX
func (b *Bitmex) UserLogOut(ctx context.Context) error {
	return b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserLogout,
		nil,
		nil)
}

// UserLogOutAll logs you out of all systems for BitMEX
func (b *Bitmex) UserLogOutAll(ctx context.Context) (int64, error) {
	var status int64

	return status, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserLogoutAll,
		nil,
		&status)
}

// GetUserMargin returns user margin information
func (b *Bitmex) GetUserMargin(ctx context.Context, currency string) (UserMargin, error) {
	var info UserMargin

	return info, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUserMargin,
		UserCurrencyParams{Currency: currency},
		&info)
}

// GetAllUserMargin returns user margin information
func (b *Bitmex) GetAllUserMargin(ctx context.Context) ([]UserMargin, error) {
	var info []UserMargin

	return info, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUserMargin,
		UserCurrencyParams{Currency: "all"},
		&info)
}

// GetMinimumWithdrawalFee returns minimum withdrawal fee information
func (b *Bitmex) GetMinimumWithdrawalFee(ctx context.Context, currency string) (MinWithdrawalFee, error) {
	var fee MinWithdrawalFee

	return fee, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUserMinWithdrawalFee,
		UserCurrencyParams{Currency: currency},
		&fee)
}

// GetUserPreferences returns user preferences
func (b *Bitmex) GetUserPreferences(ctx context.Context, params UserPreferencesParams) (User, error) {
	var userInfo User

	return userInfo, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserPreferences,
		params,
		&userInfo)
}

// EnableTFA enables 2 factor authentication
func (b *Bitmex) EnableTFA(ctx context.Context, typ string) (bool, error) {
	var enabled bool

	return enabled, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserRequestTFA,
		UserConfirmTFAParams{Type: typ},
		&enabled)
}

// UserRequestWithdrawal This will send a confirmation email to the email
// address on record, unless requested via an API Key with the withdraw
// permission.
func (b *Bitmex) UserRequestWithdrawal(ctx context.Context, params UserRequestWithdrawalParams) (TransactionInfo, error) {
	var info TransactionInfo

	return info, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserRequestWithdraw,
		params,
		&info)
}

// GetWalletInfo returns user wallet information
func (b *Bitmex) GetWalletInfo(ctx context.Context, currency string) (WalletInfo, error) {
	var info WalletInfo

	if err := b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUserWallet,
		UserCurrencyParams{Currency: currency},
		&info); err != nil {
		return info, err
	}

	// Bitmex has an "interesting" of dealing with currencies,
	// for instance XBt is actually BTC but in Satoshi units,
	// for sanity purposes apply here a conversion to normalize
	// this
	// avoid a copy here since this is a big struct
	normalizeWalletInfo(&info)

	return info, nil
}

// GetWalletHistory returns user wallet history transaction data
func (b *Bitmex) GetWalletHistory(ctx context.Context, currency string) ([]TransactionInfo, error) {
	var info []TransactionInfo

	return info, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUserWalletHistory,
		UserCurrencyParams{Currency: currency},
		&info)
}

// GetWalletSummary returns user wallet summary
func (b *Bitmex) GetWalletSummary(ctx context.Context, currency string) ([]TransactionInfo, error) {
	var info []TransactionInfo

	return info, b.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUserWalletSummary,
		UserCurrencyParams{Currency: currency},
		&info)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (b *Bitmex) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, params Parameter, result interface{}) error {
	var respCheck interface{}
	endpoint, err := b.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	path = endpoint + path
	if params != nil && !params.IsNil() {
		path, err = params.ToURLVals(path)
		if err != nil {
			return err
		}
	}

	item := &request.Item{
		Method:        http.MethodGet,
		Path:          path,
		Result:        &respCheck,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	}

	err = b.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
	if err != nil {
		return err
	}

	return b.CaptureError(respCheck, result)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to bitmex
func (b *Bitmex) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, verb, path string, params Parameter, result interface{}) error {
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := b.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	var respCheck interface{}
	newRequest := func() (*request.Item, error) {
		expires := time.Now().Add(time.Second * 10)
		timestamp := expires.UnixNano()
		timestampStr := strconv.FormatInt(timestamp, 10)
		timestampNew := timestampStr[:13]

		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		headers["api-expires"] = timestampNew
		headers["api-key"] = creds.Key

		var payload string
		if params != nil {
			err = params.VerifyData()
			if err != nil {
				return nil, err
			}
			var data []byte
			data, err = json.Marshal(params)
			if err != nil {
				return nil, err
			}
			payload = string(data)
		}

		var hmac []byte
		hmac, err = crypto.GetHMAC(crypto.HashSHA256,
			[]byte(verb+"/api/v1"+path+timestampNew+payload),
			[]byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		headers["api-signature"] = crypto.HexEncodeToString(hmac)

		return &request.Item{
			Method:        verb,
			Path:          endpoint + path,
			Headers:       headers,
			Body:          strings.NewReader(payload),
			Result:        &respCheck,
			Verbose:       b.Verbose,
			HTTPDebugging: b.HTTPDebugging,
			HTTPRecording: b.HTTPRecording,
		}, nil
	}
	err = b.SendPayload(ctx, request.Auth, newRequest, request.AuthenticatedRequest)
	if err != nil {
		return err
	}

	return b.CaptureError(respCheck, result)
}

// CaptureError little hack that captures an error
func (b *Bitmex) CaptureError(resp, reType interface{}) error {
	var Error RequestError

	marshalled, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	err = json.Unmarshal(marshalled, &Error)
	if err == nil && Error.Error.Name != "" {
		return fmt.Errorf("bitmex error %s: %s",
			Error.Error.Name,
			Error.Error.Message)
	}

	return json.Unmarshal(marshalled, reType)
}

// GetFee returns an estimate of fee based on type of transaction
func (b *Bitmex) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	var err error
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount, feeBuilder.IsMaker)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, err
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.000750 * price * amount
}

// calculateTradingFee returns the fee for trading any currency on Bitmex
func calculateTradingFee(purchasePrice, amount float64, isMaker bool) float64 {
	var fee = 0.000750
	if isMaker {
		fee -= 0.000250
	}

	return fee * purchasePrice * amount
}

// normalizeWalletInfo converts any non-standard currencies (eg. XBt -> BTC)
func normalizeWalletInfo(w *WalletInfo) {
	if w.Currency != "XBt" {
		return
	}

	w.Currency = "BTC"
	w.Amount *= constSatoshiBTC
}
