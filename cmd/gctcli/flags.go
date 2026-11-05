package main

import (
	"time"

	"github.com/urfave/cli/v2"
)

const (
	exchangeFlagName                  = "exchange"
	assetFlagName                     = "asset"
	pairFlagName                      = "pair"
	startFlagName                     = "start"
	endFlagName                       = "end"
	amountFlagName                    = "amount"
	sideFlagName                      = "side"
	enableFlagName                    = "enable"
	idFlagName                        = "id"
	subsystemFlagName                 = "subsystem"
	descriptionFlagName               = "description"
	priceFlagName                     = "price"
	addressFlagName                   = "address"
	coinTypeFlagName                  = "coin_type"
	orderIDFlagName                   = "order_id"
	accountIDFlagName                 = "account_id"
	cryptocurrencyFlagName            = "cryptocurrency"
	chainFlagName                     = "chain"
	currencyFlagName                  = "currency"
	limitFlagName                     = "limit"
	loggerFlagName                    = "logger"
	granularityFlagName               = "granularity"
	intervalFlagName                  = "interval"
	fillMissingDataWithTradesFlagName = "fillmissingdatawithtrades"
	marginTypeFlagName                = "margintype"
	prerequisiteJobNicknameFlagName   = "prerequisite_job_nickname"
	includeOrderDetailsFlagName       = "includeorderdetails"
	getFundingDataFlagName            = "getfundingdata"
	includeFundingEntriesFlagName     = "includefundingentries"
	includePredictedRateFlagName      = "includepredictedrate"
	includePredictedFlagName          = "includepredicted"
	orderSideFlagName                 = "orderside"
	underlyingPairFlagName            = "underlyingpair"
	exchangeStyleFlagName             = "exchangestyle"
	depthLimitFlagName                = "depthlimit"
	pairsFlagName                     = "pairs"
	requestSizeLimitFlagName          = "request_size_limit"
	syncFlagName                      = "sync"
	forceFlagName                     = "force"
	startDateFlagName                 = "start_date"
	endDateFlagName                   = "end_date"
	percentFlagName                   = "percent"
)

const (
	exchangeUsage                  = "the exchange to act on"
	pairUsage                      = "the currency pair"
	assetUsage                     = "the asset type of the currency pair"
	futuresAssetUsage              = "the asset type of the currency pair, must be a futures type"
	sideUsage                      = "the order side to use (BUY OR SELL)"
	orderIDUsage                   = "the order ID"
	subsystemUsage                 = "the subsystem to act on"
	cryptocurrencyUsage            = "the cryptocurrency"
	chainUsage                     = "the transfer chain to use"
	underlyingPairUsage            = "the underlying currency of a futures pair eg pair is BTCUSD-1337-C, the underlying pair could be BTC-USD, or if pair is LTCUSD-PERP the underlying pair could be LTC-USD"
	prerequisiteJobNicknameUsage   = "the prerequisite job nickname; the job will only run when its prerequisite is complete - use command 'removeprerequisite' to remove a prerequisite"
	orderAmountUsage               = "the amount for the order"
	withdrawalAmountUsage          = "amount of funds to withdraw"
	currencyUsage                  = "the currency"
	marginTypeUsage                = "the margin type, such as 'isolated', 'multi' or 'cross'"
	orderPriceUsage                = "the price for the order"
	startUsage                     = "the start date"
	endUsage                       = "the end date"
	startRoundedUsage              = "the start date, rounded down to the nearest hour"
	endRoundedUsage                = "the end date, rounded down to the nearest hour"
	addressUsage                   = "the portfolio address"
	coinTypeUsage                  = "the coin type e.g ('BTC')"
	portfolioDescriptionUsage      = "description of the address"
	accountIDUsage                 = "the account id"
	withdrawalDescriptionUsage     = "description to submit with request"
	idUsage                        = "withdrawal id"
	limitUsage                     = "max number of withdrawals to return"
	loggerUsage                    = "the logger name"
	fillMissingDataWithTradesUsage = "will create candles for missing intervals using stored trade data <true/false>"
	forceUsage                     = "will overwrite any conflicting candle data on save <true/false>"
	startDateUsage                 = "formatted as: " + time.DateTime
	endDateUsage                   = "formatted as: " + time.DateTime
	includeOrderDetailsUsage       = "includes all orders that make up a position in the response"
	getFundingDataUsage            = "if true, will return funding rate summary"
	includeFundingEntriesUsage     = "if true, will return all funding rate entries - requires --getfundingdata"
	includePredictedRateUsage      = "if true, will return the predicted funding rate - requires --getfundingdata"
	includePredictedUsage          = "optional - include the predicted next funding rate"
	orderSideUsage                 = "optional - some exchanges distinguish between order side"
	percentUsage                   = "the max percentage slip you wish to occur e.g. 1 = 1% and 100 = 100%. Note: If selling base/hitting the bids you can only have a max value of 100%"
	exchangeStyleUsage             = "optional - renders the books like on an exchange website"
	pairsUsage                     = "either a single currency pair string or comma delimiter string of pairs e.g. \"BTC-USD,XRP-USD\""
)

var (
	requiredSubsystemFlag = &cli.StringFlag{
		Name:     subsystemFlagName,
		Required: true,
		Usage:    subsystemUsage,
	}
	requiredExchangeFlag = &cli.StringFlag{
		Name:     exchangeFlagName,
		Required: true,
		Usage:    exchangeUsage,
	}
	requiredPairFlag = &cli.StringFlag{
		Name:     pairFlagName,
		Required: true,
		Usage:    pairUsage,
	}
	requiredAssetFlag = &cli.StringFlag{
		Name:     assetFlagName,
		Required: true,
		Usage:    assetUsage,
	}
	requiredPortfolioAddressFlag = &cli.StringFlag{
		Name:     addressFlagName,
		Required: true,
		Usage:    addressUsage,
	}
	requiredCoinTypeFlag = &cli.StringFlag{
		Name:     coinTypeFlagName,
		Required: true,
		Usage:    coinTypeUsage,
	}
	requiredOrderIDFlag = &cli.StringFlag{
		Name:     orderIDFlagName,
		Required: true,
		Usage:    orderIDUsage,
	}
	requiredSideFlag = &cli.StringFlag{
		Name:     sideFlagName,
		Required: true,
		Usage:    sideUsage,
	}
	requiredOrderAmountFlag = &cli.Float64Flag{
		Name:     amountFlagName,
		Required: true,
		Usage:    orderAmountUsage,
	}
	requiredCryptocurrencyFlag = &cli.StringFlag{
		Name:     cryptocurrencyFlagName,
		Required: true,
		Usage:    cryptocurrencyUsage,
	}
	requiredCurrencyFlag = &cli.StringFlag{
		Name:     currencyFlagName,
		Required: true,
		Usage:    currencyUsage,
	}
	requiredWithdrawalAmountFlag = &cli.Float64Flag{
		Name:     amountFlagName,
		Required: true,
		Usage:    withdrawalAmountUsage,
	}
	requiredWithdrawalIDFlag = &cli.StringFlag{
		Name:     idFlagName,
		Required: true,
		Usage:    idUsage,
	}
	requiredLoggerFlag = &cli.StringFlag{
		Name:     loggerFlagName,
		Required: true,
		Usage:    loggerUsage,
	}
	requiredExchangeAliasFlag = &cli.StringFlag{
		Name:     exchangeFlagName,
		Required: true,
		Aliases:  []string{"e"},
		Usage:    exchangeUsage,
	}
	requiredPairAliasFlag = &cli.StringFlag{
		Name:     pairFlagName,
		Required: true,
		Aliases:  []string{"p"},
		Usage:    pairUsage,
	}
	requiredAssetAliasFlag = &cli.StringFlag{
		Name:     assetFlagName,
		Required: true,
		Aliases:  []string{"a"},
		Usage:    assetUsage,
	}
	requiredFuturesAssetFlag = &cli.StringFlag{
		Name:     assetFlagName,
		Required: true,
		Aliases:  []string{"a"},
		Usage:    futuresAssetUsage,
	}
	requiredPairsFlag = &cli.StringFlag{
		Name:     pairsFlagName,
		Required: true,
		Usage:    pairsUsage,
	}
)
