package apexpro

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/internal/utils/starkex"
	"github.com/thrasher-corp/gocryptotrader/types"
	"golang.org/x/exp/rand"
)

// Exchange is the overarching type across this package
type Exchange struct {
	exchange.Base

	// SymbolsConfig represents all symbols configuration.
	SymbolsConfig *AllSymbolsV1Config

	StarkConfig       *starkex.StarkConfig
	UserAccountDetail *UserAccountV2
	NetworkID         int
}

const (
	apexproAPIURL     = "https://pro.apex.exchange/api/"
	apexproTestAPIURL = "https://testnet.pro.apex.exchange/api/"

	apexProOmniAPIURL = "https://omni.apex.exchange/api/"
)

var (
	errL2KeyMissing                  = errors.New("l2 Key is required")
	errEthereumAddressMissing        = errors.New("ethereum address is missing")
	errInvalidEthereumAddress        = errors.New("invalid ethereum address")
	errChainIDMissing                = errors.New("chain ID is missing")
	errOrderbookLevelIsRequired      = errors.New("orderbook level is required")
	errInvalidTimestamp              = errors.New("err invalid timestamp")
	errZeroKnowledgeAccountIDMissing = errors.New("zero knowledge account id is required")
	errInitialMarginRateRequired     = errors.New("initial margin rate required")
	errUserIDRequired                = errors.New("user ID is required")
	errDeviceTypeIsRequired          = errors.New("device type is required")
	errLimitFeeRequired              = errors.New("limit fee is required")
	errClientIDMissing               = errors.New("client ID is required")
)

// Start implementing public and private exchange API funcs below

// GetSystemTimeV3 retrieves V3 system time.
func (e *Exchange) GetSystemTimeV3(ctx context.Context) (time.Time, error) {
	return e.getSystemTime(ctx, "v3/time")
}

// GetSystemTimeV2 retrieves V2 system time.
func (e *Exchange) GetSystemTimeV2(ctx context.Context) (time.Time, error) {
	return e.getSystemTime(ctx, "v2/time")
}

// GetSystemTimeV1 retrieves V1 system time.
func (e *Exchange) GetSystemTimeV1(ctx context.Context) (time.Time, error) {
	return e.getSystemTime(ctx, "v1/time")
}

func (e *Exchange) getSystemTime(ctx context.Context, path string) (time.Time, error) {
	resp := &struct {
		Time types.Time `json:"time"`
	}{}
	return resp.Time.Time(), e.SendHTTPRequest(ctx, exchange.RestSpot, path, request.UnAuth, &resp)
}

// GetAllConfigDataV3 retrieves all symbols and asset configurations.
func (e *Exchange) GetAllConfigDataV3(ctx context.Context) (*AllSymbolsConfigs, error) {
	var resp *AllSymbolsConfigs
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "v3/symbols", request.UnAuth, &resp)
}

// GetAllSymbolsConfigDataV1 retrieves all symbols and asset configurations from the V1 API.
func (e *Exchange) GetAllSymbolsConfigDataV1(ctx context.Context) (*AllSymbolsV1Config, error) {
	var resp *AllSymbolsV1Config
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "v1/symbols", request.UnAuth, &resp, true)
}

// GetMarketDepthV3 retrieve all active orderbook for one symbol, include all bids and asks.
func (e *Exchange) GetMarketDepthV3(ctx context.Context, symbol string, limit int64) (*MarketDepthV3, error) {
	return e.getMarketDepth(ctx, symbol, "v3/depth", limit)
}

// GetMarketDepthV2 retrieve all active orderbook for one symbol, include all bids and asks.
func (e *Exchange) GetMarketDepthV2(ctx context.Context, symbol string, limit int64) (*MarketDepthV3, error) {
	return e.getMarketDepth(ctx, symbol, "v2/depth", limit)
}

// GetMarketDepthV1 retrieve all active orderbook for one symbol, include all bids and asks.
func (e *Exchange) GetMarketDepthV1(ctx context.Context, symbol string, limit int64) (*MarketDepthV3, error) {
	return e.getMarketDepth(ctx, symbol, "v1/depth", limit)
}

func (e *Exchange) getMarketDepth(ctx context.Context, symbol, path string, limit int64) (*MarketDepthV3, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)

	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *MarketDepthV3
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(path, params), request.UnAuth, &resp)
}

// GetNewestTradingDataV3 retrieve trading data.
func (e *Exchange) GetNewestTradingDataV3(ctx context.Context, symbol string, limit int64) ([]NewTradingData, error) {
	return e.getNewestTradingData(ctx, symbol, "v3/trades", limit)
}

// GetNewestTradingDataV2 retrieve trading data.
func (e *Exchange) GetNewestTradingDataV2(ctx context.Context, symbol string, limit int64) ([]NewTradingData, error) {
	return e.getNewestTradingData(ctx, symbol, "v2/trades", limit)
}

// GetNewestTradingDataV1 retrieve trading data.
func (e *Exchange) GetNewestTradingDataV1(ctx context.Context, symbol string, limit int64) ([]NewTradingData, error) {
	return e.getNewestTradingData(ctx, symbol, "v1/trades", limit)
}

func (e *Exchange) getNewestTradingData(ctx context.Context, symbol, path string, limit int64) ([]NewTradingData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []NewTradingData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(path, params), request.UnAuth, &resp)
}

var intervalToStringMap = map[kline.Interval]string{kline.OneMin: "1", kline.FiveMin: "5", kline.FifteenMin: "15", kline.ThirtyMin: "30", kline.OneHour: "60", kline.TwoHour: "120", kline.FourHour: "240", kline.SixHour: "360", kline.SevenHour: "720", kline.OneDay: "D", kline.OneMonth: "M", kline.OneWeek: "W"}

func intervalToString(interval kline.Interval) (string, error) {
	intervalString, okay := intervalToStringMap[interval]
	if !okay {
		return "", kline.ErrUnsupportedInterval
	}
	return intervalString, nil
}

// GetCandlestickChartDataV3 etrieves all candlestick chart data.
// Candlestick chart time indicators: Numbers represent minutes, D for Days, M for Month and W for Week — 1 5 15 30 60 120 240 360 720 "D" "M" "W"
func (e *Exchange) GetCandlestickChartDataV3(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) (map[string][]CandlestickData, error) {
	return e.getCandlestickChartData(ctx, symbol, "v3/klines", interval, startTime, endTime, limit)
}

// GetCandlestickChartDataV2 retrieves v2 all candlestick chart data.
func (e *Exchange) GetCandlestickChartDataV2(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) (map[string][]CandlestickData, error) {
	return e.getCandlestickChartData(ctx, symbol, "v2/klines", interval, startTime, endTime, limit)
}

// GetCandlestickChartDataV1 retrieves v1 all candlestick chart data.
func (e *Exchange) GetCandlestickChartDataV1(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) (map[string][]CandlestickData, error) {
	return e.getCandlestickChartData(ctx, symbol, "v1/klines", interval, startTime, endTime, limit)
}

func (e *Exchange) getCandlestickChartData(ctx context.Context, symbol, path string, interval kline.Interval, startTime, endTime time.Time, limit int64) (map[string][]CandlestickData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if interval != kline.Interval(0) {
		intervalString, err := intervalToString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	if !startTime.IsZero() {
		params.Set("start", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("end", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp map[string][]CandlestickData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(path, params), request.UnAuth, &resp)
}

// GetTickerDataV3 get the latest data on symbol tickers.
func (e *Exchange) GetTickerDataV3(ctx context.Context, symbol string) ([]TickerData, error) {
	return e.getTickerData(ctx, symbol, "v3/ticker")
}

// GetTickerDataV2 get the latest data on symbol tickers.
func (e *Exchange) GetTickerDataV2(ctx context.Context, symbol string) ([]TickerData, error) {
	return e.getTickerData(ctx, symbol, "v2/ticker")
}

func (e *Exchange) getTickerData(ctx context.Context, symbol, path string) ([]TickerData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp []TickerData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(path, params), request.UnAuth, &resp)
}

// GetFundingHistoryRateV3 retrieves a funding history rate.
func (e *Exchange) GetFundingHistoryRateV3(ctx context.Context, symbol string, beginTime, endTime time.Time, page, limit int64) (*FundingRateHistory, error) {
	return e.getFundingHistoryRate(ctx, symbol, "v3/history-funding", beginTime, endTime, page, limit)
}

// GetFundingHistoryRateV2 retrieves a funding history rate.
func (e *Exchange) GetFundingHistoryRateV2(ctx context.Context, symbol string, beginTime, endTime time.Time, page, limit int64) (*FundingRateHistory, error) {
	return e.getFundingHistoryRate(ctx, symbol, "v2/history-funding", beginTime, endTime, page, limit)
}

// GetFundingHistoryRateV1 retrieves a funding history rate.
func (e *Exchange) GetFundingHistoryRateV1(ctx context.Context, symbol string, beginTime, endTime time.Time, page, limit int64) (*FundingRateHistory, error) {
	return e.getFundingHistoryRate(ctx, symbol, "v2/history-funding", beginTime, endTime, page, limit)
}

func (e *Exchange) getFundingHistoryRate(ctx context.Context, symbol, path string, beginTime, endTime time.Time, page, limit int64) (*FundingRateHistory, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !beginTime.IsZero() {
		params.Set("beginTimeInclusive", strconv.FormatInt(beginTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTimeExclusive", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	var resp *FundingRateHistory
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(path, params), request.UnAuth, &resp)
}

// GetAllConfigDataV2 retrieves USDC and USDT config
func (e *Exchange) GetAllConfigDataV2(ctx context.Context) (*V2ConfigData, error) {
	var resp *V2ConfigData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "v2/symbols", request.UnAuth, &resp, true)
}

// GetCheckIfUserExistsV2 checks existence of a persion using the ethereum Address
func (e *Exchange) GetCheckIfUserExistsV2(ctx context.Context, ethAddress string) (bool, error) {
	return e.getCheckIfUserExists(ctx, ethAddress, "v2/check-user-exist")
}

// GetCheckIfUserExistsV1 checks existence of a persion using the ethereum Address
func (e *Exchange) GetCheckIfUserExistsV1(ctx context.Context, ethAddress string) (bool, error) {
	return e.getCheckIfUserExists(ctx, ethAddress, "v1/check-user-exist")
}

func (e *Exchange) getCheckIfUserExists(ctx context.Context, ethAddress, path string) (bool, error) {
	if ethAddress == "" {
		return false, errEthereumAddressMissing
	}
	params := url.Values{}
	params.Set("ethAddress", ethAddress)
	var resp bool
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(path, params), request.UnAuth, &resp)
}

// ----------------------------------------------------------------     Authenticated Endpoints ----------------------------------------------------------------

// GenerateNonceV3 generate and obtain nonce before registration. The nonce is used to assemble the signature field upon registration.
func (e *Exchange) GenerateNonceV3(ctx context.Context, l2Key, ethereumAddress, chainID string) (*NonceResponse, error) {
	return e.generateNonce(ctx, l2Key, ethereumAddress, chainID, "v3/generate-nonce")
}

// GenerateNonceV2 before registering, generate and obtain a nonce. The nonce serves the purpose of assembling the signature field during the registration process.
func (e *Exchange) GenerateNonceV2(ctx context.Context, l2Key, ethereumAddress, chainID string) (*NonceResponse, error) {
	return e.generateNonce(ctx, l2Key, ethereumAddress, chainID, "v2/generate-nonce")
}

// GenerateNonceV1 before registering, generate and obtain a nonce.
func (e *Exchange) GenerateNonceV1(ctx context.Context, l2Key, ethereumAddress, chainID string) (*NonceResponse, error) {
	return e.generateNonce(ctx, l2Key, ethereumAddress, chainID, "v1/generate-nonce")
}

func (e *Exchange) generateNonce(ctx context.Context, l2Key, ethereumAddress, chainID, path string) (*NonceResponse, error) {
	if l2Key == "" {
		return nil, errL2KeyMissing
	}
	if ethereumAddress == "" {
		return nil, errEthereumAddressMissing
	}
	if chainID == "" {
		return nil, errChainIDMissing
	}
	arg := &struct {
		L2Key           string `json:"l2Key"`
		EthereumAddress string `json:"ethAddress"`
		ChainID         string `json:"chainId"`
	}{
		L2Key:           l2Key,
		EthereumAddress: ethereumAddress,
		ChainID:         chainID,
	}
	var resp *NonceResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, path, request.UnAuth, nil, arg, &resp)
}

// GetUsersDataV3 retrieves an account users information.
func (e *Exchange) GetUsersDataV3(ctx context.Context) (*UserData, error) {
	return e.getUsersData(ctx, "v3/user")
}

// GetUsersDataV2 retrieves an account users information through the V2 API
func (e *Exchange) GetUsersDataV2(ctx context.Context) (*UserData, error) {
	return e.getUsersData(ctx, "v2/user")
}

// GetUsersDataV1 retrieves an account users information through the V1 API
func (e *Exchange) GetUsersDataV1(ctx context.Context) (*UserData, error) {
	return e.getUsersData(ctx, "v1/user")
}

func (e *Exchange) getUsersData(ctx context.Context, path string) (*UserData, error) {
	var resp *UserData
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, path, request.Unset, nil, nil, &resp)
}

// EditUserDataV3 edits user's data.
func (e *Exchange) EditUserDataV3(ctx context.Context, arg *EditUserDataParams) (*UserDataResponse, error) {
	return e.editUserData(ctx, arg, "v3/modify-user")
}

// EditUserDataV2 edits user's data through the V2 API.
func (e *Exchange) EditUserDataV2(ctx context.Context, arg *EditUserDataParams) (*UserDataResponse, error) {
	return e.editUserData(ctx, arg, "v2/modify-user")
}

// EditUserDataV1 edits user's data through the V1 API.
func (e *Exchange) EditUserDataV1(ctx context.Context, arg *EditUserDataParams) (*UserDataResponse, error) {
	return e.editUserData(ctx, arg, "v1/modify-user")
}

func (e *Exchange) editUserData(ctx context.Context, arg *EditUserDataParams, path string) (*UserDataResponse, error) {
	if *arg == (EditUserDataParams{}) {
		return nil, common.ErrNilPointer
	}
	var resp *UserDataResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, path, request.UnAuth, nil, arg, &resp)
}

// GetUserAccountDataV3 get an account for a user by id. Using the client, the id will be generated with client information and an Ethereum address.
func (e *Exchange) GetUserAccountDataV3(ctx context.Context) (*UserAccountDetail, error) {
	var resp *UserAccountDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "v3/account", request.UnAuth, nil, nil, &resp)
}

// GetUserAccountDataV2 get a user account detail through the V2 API.
func (e *Exchange) GetUserAccountDataV2(ctx context.Context) (*UserAccountV2, error) {
	var resp *UserAccountV2
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "v2/account", request.UnAuth, nil, nil, &resp)
}

// GetUserAccountDataV1 get an account for a user by id
func (e *Exchange) GetUserAccountDataV1(ctx context.Context) (*UserAccountDetailV1, error) {
	var resp *UserAccountDetailV1
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "v1/account", request.UnAuth, nil, nil, &resp)
}

// GetUserAccountBalance retrieves user account balance information.
func (e *Exchange) GetUserAccountBalance(ctx context.Context) (*UserAccountBalanceResponse, error) {
	var resp *UserAccountBalanceResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "v3/account-balance", request.UnAuth, nil, nil, &resp)
}

// GetUserAccountBalanceV2 retrieves user account balance information through the V2 API.
func (e *Exchange) GetUserAccountBalanceV2(ctx context.Context) (*UserAccountBalanceV2Response, error) {
	var resp *UserAccountBalanceV2Response
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "v2/account-balance", request.UnAuth, nil, nil, &resp)
}

// GetUserAccountBalanceV1 retrieve user account balance
func (e *Exchange) GetUserAccountBalanceV1(ctx context.Context) (*UserAccountBalanceResponse, error) {
	var resp *UserAccountBalanceResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "v1/account-balance", request.UnAuth, nil, nil, &resp)
}

// UserWithdrawalsV2

// GetUserTransferDataV2 retrieves user's asset transfer information.
func (e *Exchange) GetUserTransferDataV2(ctx context.Context, ccy currency.Code, startTime, endTime time.Time, transferType string, chainIDs []string, limit, page int64) (*UserWithdrawalsV2, error) {
	return e.getUserTransferData(ctx, ccy, startTime, endTime, transferType, "v2/transfers", chainIDs, limit, page)
}

// GetUserTransferDataV1 retrieves user's deposit data.
func (e *Exchange) GetUserTransferDataV1(ctx context.Context, ccy currency.Code, startTime, endTime time.Time, transferType string, chainIDs []string, limit, page int64) (*UserWithdrawalsV2, error) {
	return e.getUserTransferData(ctx, ccy, startTime, endTime, transferType, "v1/transfers", chainIDs, limit, page)
}

func (e *Exchange) getUserTransferData(ctx context.Context, ccy currency.Code, startTime, endTime time.Time, transferType, path string, chainIDs []string, limit, page int64) (*UserWithdrawalsV2, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currencyId", ccy.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if !startTime.IsZero() {
		params.Set("beginTimeInclusive", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTimeExclusive", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if len(chainIDs) > 0 {
		params.Set("chainIds", strings.Join(chainIDs, ","))
	}
	if transferType != "" {
		params.Set("transferType", transferType)
	}
	var resp *UserWithdrawalsV2
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, request.UnAuth, params, nil, &resp)
}

// GetUserWithdrawalListV2 retrieves asset withdrawal list.
func (e *Exchange) GetUserWithdrawalListV2(ctx context.Context, transferType string, startTime, endTime time.Time, page, limit int64) (*WithdrawalsV2, error) {
	return e.getUserWithdrawalList(ctx, transferType, "v2/withdraw-list", startTime, endTime, page, limit)
}

// GetUserWithdrawalListV1 returns the user withdrawal list.
func (e *Exchange) GetUserWithdrawalListV1(ctx context.Context, transferType string, startTime, endTime time.Time, page, limit int64) (*WithdrawalsV2, error) {
	return e.getUserWithdrawalList(ctx, transferType, "v1/withdraw-list", startTime, endTime, page, limit)
}

func (e *Exchange) getUserWithdrawalList(ctx context.Context, transferType, path string, startTime, endTime time.Time, page, limit int64) (*WithdrawalsV2, error) {
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("beginTimeInclusive", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTimeExclusive", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if transferType != "" {
		params.Set("transferType", transferType)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	var resp *WithdrawalsV2
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, request.UnAuth, params, nil, &resp)
}

// GetFastAndCrossChainWithdrawalFeesV2 retrieves fee information of fast and cross-chain withdrawal transactions.
func (e *Exchange) GetFastAndCrossChainWithdrawalFeesV2(ctx context.Context, amount float64, chainID string, token currency.Code) (*FastAndCrossChainWithdrawalFees, error) {
	if token.IsEmpty() {
		return nil, fmt.Errorf("%w, token is required", currency.ErrCurrencyCodeEmpty)
	}
	return e.getFastAndCrossChainWithdrawalFees(ctx, amount, chainID, "v2/uncommon-withdraw-fee", token.String())
}

// GetFastAndCrossChainWithdrawalFeesV1 retrieves fee information of fast and cross-chain withdrawals.
func (e *Exchange) GetFastAndCrossChainWithdrawalFeesV1(ctx context.Context, amount float64, chainID string) (*FastAndCrossChainWithdrawalFees, error) {
	return e.getFastAndCrossChainWithdrawalFees(ctx, amount, chainID, "v1/uncommon-withdraw-fee", "")
}

func (e *Exchange) getFastAndCrossChainWithdrawalFees(ctx context.Context, amount float64, chainID, path, token string) (*FastAndCrossChainWithdrawalFees, error) {
	params := url.Values{}
	params.Set("token", token)
	if amount > 0 {
		params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}
	if chainID != "" {
		params.Set("chainId", chainID)
	}
	var resp *FastAndCrossChainWithdrawalFees
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, request.UnAuth, params, nil, &resp)
}

// GetAssetWithdrawalAndTransferLimitV2 retrieves an asset withdrawal and transfer limit per interval.
func (e *Exchange) GetAssetWithdrawalAndTransferLimitV2(ctx context.Context, currencyID currency.Code) (*TransferAndWithdrawalLimit, error) {
	return e.getAssetWithdrawalAndTransferLimit(ctx, currencyID, "v2/transfer-limit")
}

// GetAssetWithdrawalAndTransferLimitV1 retrieves an asset withdrawal and transfer limit per interval.
func (e *Exchange) GetAssetWithdrawalAndTransferLimitV1(ctx context.Context, currencyID currency.Code) (*TransferAndWithdrawalLimit, error) {
	return e.getAssetWithdrawalAndTransferLimit(ctx, currencyID, "v1/transfer-limit")
}

func (e *Exchange) getAssetWithdrawalAndTransferLimit(ctx context.Context, currencyID currency.Code, path string) (*TransferAndWithdrawalLimit, error) {
	if currencyID.IsEmpty() {
		return nil, fmt.Errorf("%w, currencyID is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("currencyId", currencyID.String())
	var resp *TransferAndWithdrawalLimit
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, request.UnAuth, params, nil, &resp)
}

// GetUserTransferData retrieves user's asset transfer data.
// Direction: possible values are 'NEXT' and 'PREVIOUS'
// TransfersType: possible values are 'DEPOSIT', 'WITHDRAW' ,'FAST_WITHDRAW' ,'OMNI_TO_PERP' for spot account -> contract account,'OMNI_FROM_PERP' for spot account <- contract account,'AFFILIATE_REBATE' affliate rebate,'REFERRAL_REBATE' for referral rebate,'BROKER_REBATE' for broker rebate
func (e *Exchange) GetUserTransferData(ctx context.Context, id, limit int64, tokenID, transferType, subAccountID, direction string, startAt, endAt time.Time, chainIDs []string) (*UserWithdrawals, error) {
	if startAt.IsZero() {
		return nil, fmt.Errorf("%w, startTime is required", errInvalidTimestamp)
	}
	if endAt.IsZero() {
		return nil, fmt.Errorf("%w, endTime is required", errInvalidTimestamp)
	}
	arg := make(map[string]interface{})
	params := url.Values{}
	if limit > 0 {
		arg["limit"] = strconv.FormatInt(limit, 10)
	}
	if id != 0 {
		arg["id"] = strconv.FormatInt(id, 10)
	}
	if transferType != "" {
		arg["transferType"] = transferType
	}
	if tokenID != "" {
		params.Set("tokenId", tokenID)
	}
	if subAccountID != "" {
		params.Set("subAccountId", subAccountID)
	}
	if direction != "" {
		params.Set("direction", direction)
	}
	if len(chainIDs) > 0 {
		params.Add("chainIds", "1")
	}
	params.Set("endTimeExclusive", strconv.FormatInt(endAt.UnixMilli(), 10))
	params.Set("beginTimeInclusive", strconv.FormatInt(startAt.UnixMilli(), 10))
	var resp *UserWithdrawals
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "v3/transfers", request.UnAuth, params, nil, &resp)
}

// GetWithdrawalFees retrieves list of withdrawal fees.
// the withdrawal need zkAvailableAmount >= withdrawAmount
// the fast withdrawal needzkAvailableAmount >= withdrawAmount && fastPoolAvailableAmount>= withdrawAmount
func (e *Exchange) GetWithdrawalFees(ctx context.Context, amount float64, chainIDs []string, tokenID int64) (*WithdrawalFeeInfos, error) {
	params := url.Values{}
	if amount != 0 {
		params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}
	if len(chainIDs) > 0 {
		for a := range chainIDs {
			params.Set("chainId", chainIDs[a])
		}
	}
	if tokenID != 0 {
		params.Set("tokenId", strconv.FormatInt(tokenID, 10))
	}
	var resp *WithdrawalFeeInfos
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "v3/withdraw-fee", request.UnAuth, params, nil, &resp)
}

// GetContractAccountTransferLimits retrieves a transfer limit of a contract.
func (e *Exchange) GetContractAccountTransferLimits(ctx context.Context, ccy currency.Code) (*ContractTransferLimit, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("token", ccy.String())
	var resp *ContractTransferLimit
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "v3/contract-transfer-limit", request.UnAuth, params, nil, &resp)
}

// GetTradeHistory retrieves trade fills history
func (e *Exchange) GetTradeHistory(ctx context.Context, symbol, side, orderType string, startTime, endTime time.Time, page, limit int64) (*TradeHistory, error) {
	return e.getTradeHistory(ctx, symbol, side, orderType, "", "v3/fills", startTime, endTime, page, limit, exchange.RestFutures)
}

// GetTradeHistoryV2 retrieves trade fills history through the v2 API
func (e *Exchange) GetTradeHistoryV2(ctx context.Context, symbol, side, orderType string, token currency.Code, startTime, endTime time.Time, page, limit int64) (*TradeHistory, error) {
	if token.IsEmpty() {
		return nil, fmt.Errorf("%w, token is required", currency.ErrCurrencyCodeEmpty)
	}
	return e.getTradeHistory(ctx, symbol, side, orderType, token.String(), "v2/fills", startTime, endTime, page, limit, exchange.RestSpot)
}

// GetTradeHistoryV1 retrieves trade fills history through the v1 API
func (e *Exchange) GetTradeHistoryV1(ctx context.Context, symbol, side, orderType string, startTime, endTime time.Time, page, limit int64) (*TradeHistory, error) {
	return e.getTradeHistory(ctx, symbol, side, orderType, "", "v1/fills", startTime, endTime, page, limit, exchange.RestSpot)
}

func (e *Exchange) getTradeHistory(ctx context.Context, symbol, side, orderType, token, path string, startTime, endTime time.Time, page, limit int64, ePath exchange.URL) (*TradeHistory, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if token != "" {
		params.Set("token", token)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderType != "" {
		params.Set("orderType", orderType)
	}
	if !startTime.IsZero() {
		params.Set("beginTimeInclusive", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTimeExclusive", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *TradeHistory
	return resp, e.SendAuthenticatedHTTPRequest(ctx, ePath, http.MethodGet, path, request.UnAuth, params, nil, &resp)
}

// GetWorstPriceV3 retrieves the worst market price from orderbook
func (e *Exchange) GetWorstPriceV3(ctx context.Context, symbol, side string, amount float64) (*SymbolWorstPrice, error) {
	return e.getWorstPrice(ctx, symbol, side, "v3/get-worst-price", amount, exchange.RestFutures)
}

// GetWorstPriceV2 retrieves the worst market price from orderbook
func (e *Exchange) GetWorstPriceV2(ctx context.Context, symbol, side string, amount float64) (*SymbolWorstPrice, error) {
	return e.getWorstPrice(ctx, symbol, side, "v2/get-worst-price", amount, exchange.RestSpot)
}

// GetWorstPriceV1 retrieves the worst market price from orderbook
func (e *Exchange) GetWorstPriceV1(ctx context.Context, symbol, side string, amount float64) (*SymbolWorstPrice, error) {
	return e.getWorstPrice(ctx, symbol, side, "v1/get-worst-price", amount, exchange.RestSpot)
}

func (e *Exchange) orderCreationParamsFilter(ctx context.Context, arg *CreateOrderParams) (url.Values, error) {
	if *arg == (CreateOrderParams{}) {
		return nil, order.ErrOrderDetailIsNil
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.Size <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	signature, err := e.ProcessOrderSignature(ctx, arg)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", arg.Symbol.String())
	params.Set("side", arg.Side)
	params.Set("clientOrderId", arg.ClientOrderID)
	params.Set("type", arg.OrderType)
	params.Set("size", strconv.FormatFloat(arg.Size, 'f', -1, 64))
	params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	if arg.LimitFee != 0 {
		params.Set("limitFee", strconv.FormatFloat(arg.LimitFee, 'f', -1, 64))
	}
	params.Set("expiration", strconv.FormatInt(arg.ExpirationTime, 10))
	if arg.TimeInForce != "" {
		params.Set("timeInForce", arg.TimeInForce)
	}
	if arg.TrailingPercent > 0 {
		params.Set("trailingPercent", strconv.FormatFloat(arg.TrailingPercent, 'f', -1, 64))
	}
	if arg.TriggerPrice > 0 {
		params.Set("triggerPrice", strconv.FormatFloat(arg.TriggerPrice, 'f', -1, 64))
	}
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	params.Set("signature", signature)
	return params, nil
}

// CreateOrderV3 creates a new order
func (e *Exchange) CreateOrderV3(ctx context.Context, arg *CreateOrderParams) (*OrderDetail, error) {
	params, err := e.orderCreationParamsFilter(ctx, arg)
	if err != nil {
		return nil, err
	}
	var resp *OrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "v3/order", request.UnAuth, params, nil, &resp)
}

// CreateOrderV2 creates a new order through the v2 API
func (e *Exchange) CreateOrderV2(ctx context.Context, arg *CreateOrderParams) (*OrderDetail, error) {
	params, err := e.orderCreationParamsFilter(ctx, arg)
	if err != nil {
		return nil, err
	}
	var resp *OrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "v2/create-order", request.UnAuth, params, nil, &resp)
}

// CreateOrderV1 creates a new order through the v2 API
func (e *Exchange) CreateOrderV1(ctx context.Context, arg *CreateOrderParams) (*OrderDetail, error) {
	params, err := e.orderCreationParamsFilter(ctx, arg)
	if err != nil {
		return nil, err
	}
	var resp *OrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "v1/create-order", request.UnAuth, params, nil, &resp)
}

// FastWithdrawalV1 withdraws an asset
func (e *Exchange) FastWithdrawalV1(ctx context.Context, arg *FastWithdrawalParams) (*WithdrawalResponse, error) {
	return e.fastWithdrawal(ctx, arg, "v1/fast-withdraw")
}

// FastWithdrawalV2 withdraws an asset
func (e *Exchange) FastWithdrawalV2(ctx context.Context, arg *FastWithdrawalParams) (*WithdrawalResponse, error) {
	return e.fastWithdrawal(ctx, arg, "v2/fast-withdraw")
}

func (e *Exchange) fillWithdrawalParams(arg *FastWithdrawalParams) error {
	if *arg == (FastWithdrawalParams{}) {
		return common.ErrNilPointer
	}
	if arg.Amount <= 0 {
		return limits.ErrAmountBelowMin
	}
	if arg.Asset.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if arg.ChainID == "" {
		return errChainIDMissing
	}
	return nil
}

func (e *Exchange) fastWithdrawal(ctx context.Context, arg *FastWithdrawalParams, path string) (*WithdrawalResponse, error) {
	err := e.fillWithdrawalParams(arg)
	if err != nil {
		return nil, err
	}
	signature, err := e.ProcessConditionalTransfer(ctx, arg)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	params.Set("expiration", strconv.FormatInt(arg.Expiration, 10))
	params.Set("asset", arg.Asset.String())
	params.Set("fees", strconv.FormatFloat(arg.Fees, 'f', -1, 64))
	params.Set("chainId", arg.ChainID)
	params.Set("clientId", arg.ClientID)
	params.Set("signature", signature)
	var resp *WithdrawalResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, request.UnAuth, params, nil, &resp)
}

func (e *Exchange) getWorstPrice(ctx context.Context, symbol, side, path string, amount float64, ePath exchange.URL) (*SymbolWorstPrice, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("size", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("side", side)
	params.Set("symbol", symbol)
	var resp *SymbolWorstPrice
	return resp, e.SendAuthenticatedHTTPRequest(ctx, ePath, http.MethodGet, path, request.UnAuth, params, nil, &resp)
}

// CancelPerpOrder cancels a perpetual contract order cancellation.
func (e *Exchange) CancelPerpOrder(ctx context.Context, orderID string) (types.Number, error) {
	return e.cancelOrderByID(ctx, orderID, "v3/delete-order")
}

// CancelPerpOrderByClientOrderID cancels a perpetual contract order by client order ID.
func (e *Exchange) CancelPerpOrderByClientOrderID(ctx context.Context, clientOrderID string) (types.Number, error) {
	return e.cancelOrderByID(ctx, clientOrderID, "v3/delete-client-order-id")
}

func (e *Exchange) cancelOrderByID(ctx context.Context, id, path string) (types.Number, error) {
	if id == "" {
		return 0, order.ErrOrderIDNotSet
	}
	var resp types.Number
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, path, request.UnAuth, nil, map[string]interface{}{"id": id}, &resp)
}

// CancelAllOpenOrdersV3 cancels all open orders
func (e *Exchange) CancelAllOpenOrdersV3(ctx context.Context, symbols []string) error {
	var symbolString string
	if len(symbols) > 0 {
		symbolString = strings.Join(symbols, ",")
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "v3/delete-open-orders", request.UnAuth, nil, map[string]string{"symbol": symbolString}, nil)
}

// CancelPerpOrderV2 cancels a perpetual contract futures order.
func (e *Exchange) CancelPerpOrderV2(ctx context.Context, orderID string, token currency.Code) (types.Number, error) {
	if orderID == "" {
		return 0, order.ErrOrderIDNotSet
	}
	if token.IsEmpty() {
		return 0, fmt.Errorf("%w, token is required", currency.ErrCurrencyCodeEmpty)
	}
	arg := &map[string]string{
		"id":    orderID,
		"token": token.String(),
	}
	var resp types.Number
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "v2/delete-order", request.UnAuth, nil, arg, &resp)
}

// GetOpenOrders retrieves an active orders
func (e *Exchange) GetOpenOrders(ctx context.Context) ([]OrderDetail, error) {
	var resp []OrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "v3/open-orders", request.UnAuth, nil, nil, &resp)
}

// GetOpenOrdersV2 retrieves an active orders
func (e *Exchange) GetOpenOrdersV2(ctx context.Context, token currency.Code) ([]OrderDetail, error) {
	if token.IsEmpty() {
		return nil, fmt.Errorf("%w, token is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("token", token.String())
	var resp []OrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "v2/open-orders", request.UnAuth, params, nil, &resp)
}

// GetOpenOrdersV1 retrieves an active orders
func (e *Exchange) GetOpenOrdersV1(ctx context.Context) ([]OrderDetail, error) {
	var resp []OrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "v1/open-orders", request.UnAuth, nil, nil, &resp)
}

// GetAllOrderHistory retrieves all order history
// possible ordersKind are "ACTIVE","CONDITION", and "HISTORY"
func (e *Exchange) GetAllOrderHistory(ctx context.Context, symbol, side, orderType, orderStatus, ordersKind string, startTime, endTime time.Time, page, limit int64) (*OrderHistoryResponse, error) {
	return e.getAllOrderHistory(ctx, symbol, side, orderType, orderStatus, ordersKind, "", "v3/history-orders", startTime, endTime, page, limit)
}

// GetAllOrderHistoryV2 retrieves all order history
// possible ordersKind are "ACTIVE","CONDITION", and "HISTORY"
func (e *Exchange) GetAllOrderHistoryV2(ctx context.Context, token currency.Code, symbol, side, orderType, orderStatus, ordersKind string, startTime, endTime time.Time, page, limit int64) (*OrderHistoryResponse, error) {
	if token.IsEmpty() {
		return nil, fmt.Errorf("%w, token is required", currency.ErrCurrencyCodeEmpty)
	}
	return e.getAllOrderHistory(ctx, symbol, side, orderType, orderStatus, ordersKind, token.String(), "v2/history-orders", startTime, endTime, page, limit)
}

// GetAllOrderHistoryV1 retrieves all order history
// possible ordersKind are "ACTIVE","CONDITION", and "HISTORY"
func (e *Exchange) GetAllOrderHistoryV1(ctx context.Context, symbol, side, orderType, orderStatus, ordersKind string, startTime, endTime time.Time, page, limit int64) (*OrderHistoryResponse, error) {
	return e.getAllOrderHistory(ctx, symbol, side, orderType, orderStatus, ordersKind, "", "v1/history-orders", startTime, endTime, page, limit)
}

func (e *Exchange) getAllOrderHistory(ctx context.Context, symbol, side, orderType, orderStatus, ordersKind, token, path string, startTime, endTime time.Time, page, limit int64) (*OrderHistoryResponse, error) {
	params := url.Values{}
	if token != "" {
		params.Set("token", token)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if ordersKind != "" {
		params.Set("orderType", ordersKind)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if orderStatus != "" {
		params.Set("status", orderStatus)
	}
	if !startTime.IsZero() {
		params.Set("beginTimeInclusive", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTimeExclusive", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	var resp *OrderHistoryResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, path, request.UnAuth, params, nil, &resp)
}

// GetOrderID retrieves a single order by ID.
func (e *Exchange) GetOrderID(ctx context.Context, orderID string) (*OrderDetail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	return e.getOrderID(ctx, orderID, "v3/order")
}

// GetSingleOrderByOrderIDV2 retrieves a single order detail by ID through the V2 API
func (e *Exchange) GetSingleOrderByOrderIDV2(ctx context.Context, orderID string, token currency.Code) (*OrderDetail, error) {
	if token.IsEmpty() {
		return nil, fmt.Errorf("%w, token is required", currency.ErrCurrencyCodeEmpty)
	}
	return e.getSingleOrder(ctx, orderID, "v2/get-order", token)
}

// GetSingleOrderByClientOrderIDV2 retrieves a single order detail by client supplied order ID through the V2 API
func (e *Exchange) GetSingleOrderByClientOrderIDV2(ctx context.Context, orderID string, token currency.Code) (*OrderDetail, error) {
	if token.IsEmpty() {
		return nil, fmt.Errorf("%w, token is required", currency.ErrCurrencyCodeEmpty)
	}
	return e.getSingleOrder(ctx, orderID, "v2/order-by-client-order-id", token)
}

// GetSingleOrderByOrderIDV1 retrieves a single order detail by ID through the V1 API
func (e *Exchange) GetSingleOrderByOrderIDV1(ctx context.Context, orderID string) (*OrderDetail, error) {
	return e.getSingleOrder(ctx, orderID, "v1/get-order", currency.EMPTYCODE)
}

// GetSingleOrderByClientOrderIDV1 retrieves a single order detail by client supplied order ID through the V1 API
func (e *Exchange) GetSingleOrderByClientOrderIDV1(ctx context.Context, orderID string) (*OrderDetail, error) {
	return e.getSingleOrder(ctx, orderID, "v1/order-by-client-order-id", currency.EMPTYCODE)
}

func (e *Exchange) getSingleOrder(ctx context.Context, orderID, path string, token currency.Code) (*OrderDetail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("id", orderID)
	params.Set("token", token.String())
	var resp *OrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, request.UnAuth, params, nil, &resp)
}

// GetVerificationEmailLink retrieves a link to the verification email
func (e *Exchange) GetVerificationEmailLink(ctx context.Context, userID string, token currency.Code) error {
	params := url.Values{}
	if userID == "" {
		return errUserIDRequired
	}
	params.Set("userId", userID)
	if !token.IsEmpty() {
		params.Set("token", token.String())
	}
	return e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("v1/verify-email", params), request.UnAuth, nil)
}

// LinkDevice bind a device to an account.
// possible device type values: 1 (ios_firebase), 2 (android_firebase)
func (e *Exchange) LinkDevice(ctx context.Context, deviceToken currency.Code, deviceType string) error {
	if deviceToken.IsEmpty() {
		return fmt.Errorf("%w, device token is required", currency.ErrCurrencyCodeEmpty)
	}
	if deviceType == "" {
		return errDeviceTypeIsRequired
	}
	arg := &map[string]string{
		"deviceToken": deviceType,
		"deviceType":  deviceType,
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "v1/bind-device", request.UnAuth, nil, arg, nil)
}

// GetOrderByClientOrderID retrieves a single order by client order ID.
func (e *Exchange) GetOrderByClientOrderID(ctx context.Context, clientOrderID string) (*OrderDetail, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	return e.getOrderID(ctx, clientOrderID, "v3/order-by-client-order-id")
}

func (e *Exchange) getOrderID(ctx context.Context, id, path string) (*OrderDetail, error) {
	params := url.Values{}
	params.Set("id", id)
	var resp *OrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, path, request.UnAuth, params, nil, &resp)
}

// GetFundingRateV3 retrieves a funding rate information.
func (e *Exchange) GetFundingRateV3(ctx context.Context, symbol, side, status string, startTime, endTime time.Time, limit, page int64) (*FundingRateResponse, error) {
	return e.getFundingRate(ctx, symbol, side, status, "", "v3/funding", startTime, endTime, limit, page, exchange.RestFutures)
}

// GetFundingRateV2 retrieves a funding rate info for a contract.
func (e *Exchange) GetFundingRateV2(ctx context.Context, token currency.Code, symbol, side, status string, startTime, endTime time.Time, limit, page int64) (*FundingRateResponse, error) {
	if token.IsEmpty() {
		return nil, fmt.Errorf("%w, token is required", currency.ErrCurrencyCodeEmpty)
	}
	return e.getFundingRate(ctx, symbol, side, status, token.String(), "v2/funding", startTime, endTime, limit, page, exchange.RestSpot)
}

// GetFundingRateV1 retrieves a funding rate information.
func (e *Exchange) GetFundingRateV1(ctx context.Context, symbol, side, status string, startTime, endTime time.Time, limit, page int64) (*FundingRateResponse, error) {
	return e.getFundingRate(ctx, symbol, side, status, "", "v1/funding", startTime, endTime, limit, page, exchange.RestSpot)
}

func (e *Exchange) getFundingRate(ctx context.Context, symbol, side, status, token, path string, startTime, endTime time.Time, limit, page int64, ePath exchange.URL) (*FundingRateResponse, error) {
	params := url.Values{}
	if token != "" {
		params.Set("token", token)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if status != "" {
		params.Set("status", status)
	}
	if !startTime.IsZero() {
		params.Set("beginTimeInclusive", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTimeExclusive", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *FundingRateResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, ePath, http.MethodGet, path, request.UnAuth, params, nil, &resp)
}

// GetUserHistorialProfitAndLoss retrieves a profit and loss history of order positions
func (e *Exchange) GetUserHistorialProfitAndLoss(ctx context.Context, symbol, positionType string, startTime, endTime time.Time, page, limit int64) (*PNLHistory, error) {
	return e.getUserHistorialProfitAndLoss(ctx, symbol, positionType, "", "v3/historical-pnl", startTime, endTime, page, limit, exchange.RestFutures)
}

// GetUserHistorialProfitAndLossV1 retrieves a profit and loss history of order positions through the V1 API endpoint.
func (e *Exchange) GetUserHistorialProfitAndLossV1(ctx context.Context, symbol, positionType string, startTime, endTime time.Time, page, limit int64) (*PNLHistory, error) {
	return e.getUserHistorialProfitAndLoss(ctx, symbol, positionType, "", "v1/historical-pnl", startTime, endTime, page, limit, exchange.RestSpot)
}

// GetUserHistorialProfitAndLossV2 retrieves a profit and loss history of order positions.
func (e *Exchange) GetUserHistorialProfitAndLossV2(ctx context.Context, token currency.Code, symbol, positionType string, startTime, endTime time.Time, page, limit int64) (*PNLHistory, error) {
	if token.IsEmpty() {
		return nil, fmt.Errorf("%w, token is required", currency.ErrCurrencyCodeEmpty)
	}
	return e.getUserHistorialProfitAndLoss(ctx, symbol, positionType, token.String(), "v2/historical-pnl", startTime, endTime, page, limit, exchange.RestSpot)
}

func (e *Exchange) getUserHistorialProfitAndLoss(ctx context.Context, symbol, positionType, token, path string, startTime, endTime time.Time, page, limit int64, ePath exchange.URL) (*PNLHistory, error) {
	params := url.Values{}
	if token != "" {
		params.Set("token", token)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if positionType != "" {
		params.Set("type", positionType)
	}
	if !startTime.IsZero() {
		params.Set("beginTimeInclusive", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTimeExclusive", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *PNLHistory
	return resp, e.SendAuthenticatedHTTPRequest(ctx, ePath, http.MethodGet, path, request.UnAuth, params, nil, &resp)
}

// GetYesterdaysPNL retrieves yesterdays profit and loss(PNL)
func (e *Exchange) GetYesterdaysPNL(ctx context.Context) (types.Number, error) {
	var resp types.Number
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "v3/yesterday-pnl", request.UnAuth, nil, nil, &resp)
}

// GetYesterdaysPNLV1 retrieves yesterdays profit and loss(PNL)
func (e *Exchange) GetYesterdaysPNLV1(ctx context.Context) (types.Number, error) {
	var resp types.Number
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "v1/yesterday-pnl", request.UnAuth, nil, nil, &resp)
}

// GetYesterdaysPNLV2 retrieves yesterdays profit and loss(PNL)
func (e *Exchange) GetYesterdaysPNLV2(ctx context.Context, token currency.Code) (types.Number, error) {
	if token.IsEmpty() {
		return 0, fmt.Errorf("%w, token is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("token", token.String())
	var resp types.Number
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "v2/yesterday-pnl", request.UnAuth, params, nil, &resp)
}

// GetHistoricalAssetValue retrieves a historical asset value
func (e *Exchange) GetHistoricalAssetValue(ctx context.Context, startTime, endTime time.Time) (*AssetValueHistory, error) {
	return e.getHistoricalAssetValue(ctx, "", "v3/history-value", startTime, endTime, exchange.RestFutures)
}

// GetHistoricalAssetValueV1 retrieves a historical asset value through the V1 APi endpoints.
func (e *Exchange) GetHistoricalAssetValueV1(ctx context.Context, startTime, endTime time.Time) (*AssetValueHistory, error) {
	return e.getHistoricalAssetValue(ctx, "", "v1/history-value", startTime, endTime, exchange.RestSpot)
}

// GetHistoricalAssetValueV2 retrieves a historical asset value
func (e *Exchange) GetHistoricalAssetValueV2(ctx context.Context, token currency.Code, startTime, endTime time.Time) (*AssetValueHistory, error) {
	if token.IsEmpty() {
		return nil, fmt.Errorf("%w, token is required", currency.ErrCurrencyCodeEmpty)
	}
	return e.getHistoricalAssetValue(ctx, token.String(), "v2/history-value", startTime, endTime, exchange.RestSpot)
}

func (e *Exchange) getHistoricalAssetValue(ctx context.Context, token, path string, startTime, endTime time.Time, ePath exchange.URL) (*AssetValueHistory, error) {
	params := url.Values{}
	if token != "" {
		params.Set("token", token)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp *AssetValueHistory
	return resp, e.SendAuthenticatedHTTPRequest(ctx, ePath, http.MethodGet, path, request.UnAuth, params, nil, &resp)
}

// SetInitialMarginRateInfo sets an initial margin rate
func (e *Exchange) SetInitialMarginRateInfo(ctx context.Context, symbol string, initialMarginRate float64) error {
	return e.setInitialMarginRateInfo(ctx, symbol, "v3/set-initial-margin-rate", initialMarginRate, exchange.RestFutures)
}

// GetRepaymentPrice retrieves repayment prices for tokens
func (e *Exchange) GetRepaymentPrice(ctx context.Context, repaymentPriceTokens []RepaymentTokenAndAmount, clientID string) (*LoanRepaymentRates, error) {
	if len(repaymentPriceTokens) == 0 {
		return nil, common.ErrEmptyParams
	}
	if clientID == "" {
		return nil, errClientIDMissing
	}
	params := url.Values{}
	var paramString string
	for a := range repaymentPriceTokens {
		if repaymentPriceTokens[a].Token.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		if repaymentPriceTokens[a].Amount <= 0 {
			return nil, limits.ErrAmountBelowMin
		}
		paramString += repaymentPriceTokens[a].Token.String() + "|" + strconv.FormatFloat(repaymentPriceTokens[a].Amount, 'f', -1, 64) + ","
	}
	paramString = strings.Trim(paramString, ",")
	params.Set("repaymentPriceTokens", paramString)
	params.Set("clientId", clientID)
	var resp *LoanRepaymentRates
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "v3/repayment-price", request.Auth, params, nil, &resp)
}

// UserManualRepayment sends a user manual repayment request
func (e *Exchange) UserManualRepayment(ctx context.Context, arg *UserManualRepaymentParams) (*IDResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.ClientID == "" {
		return nil, errClientIDMissing
	}
	if len(arg.LoanRepaymentTokenAndAmount) == 0 {
		return nil, fmt.Errorf("%w: LoanRepaymentTokenAndAmount detail is required", common.ErrEmptyParams)
	}
	if len(arg.PoolRepaymentTokensDetail) == 0 {
		return nil, fmt.Errorf("%w: PoolRepaymentTokensDetail detail is required", common.ErrEmptyParams)
	}
	if arg.ExpiryTime.IsZero() {
		return nil, errExpirationTimeRequired
	}
	loanRepaymentTokensByteData, err := json.Marshal(arg.LoanRepaymentTokenAndAmount)
	if err != nil {
		return nil, err
	}
	poolTokensByteData, err := json.Marshal(arg.PoolRepaymentTokensDetail)
	if err != nil {
		return nil, err
	}
	argParam := &map[string]string{
		"repaymentTokens":     string(loanRepaymentTokensByteData),
		"poolRepaymentTokens": string(poolTokensByteData),
		"clientId":            arg.ClientID,
		"expireTime":          strconv.FormatInt(arg.ExpiryTime.UnixMilli(), 10),
	}
	var resp *IDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "v3/manual-create-repayment", request.Auth, nil, argParam, &resp)
}

// SetInitialMarginRateInfoV1 sets an initial margin rate
func (e *Exchange) SetInitialMarginRateInfoV1(ctx context.Context, symbol string, initialMarginRate float64) error {
	return e.setInitialMarginRateInfo(ctx, symbol, "v1/set-initial-margin-rate", initialMarginRate, exchange.RestSpot)
}

// SetInitialMarginRateInfoV2 sets an initial margin rate
func (e *Exchange) SetInitialMarginRateInfoV2(ctx context.Context, symbol string, initialMarginRate float64) error {
	return e.setInitialMarginRateInfo(ctx, symbol, "v2/set-initial-margin-rate", initialMarginRate, exchange.RestSpot)
}

func (e *Exchange) setInitialMarginRateInfo(ctx context.Context, symbol, path string, initialMarginRate float64, ePath exchange.URL) error {
	if symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	if initialMarginRate <= 0 {
		return errInitialMarginRateRequired
	}
	arg := &map[string]interface{}{
		"symbol":            symbol,
		"initialMarginRate": strconv.FormatFloat(initialMarginRate, 'f', -1, 64),
	}
	return e.SendAuthenticatedHTTPRequest(ctx, ePath, http.MethodPost, path, request.UnAuth, nil, arg, nil)
}

// WithdrawAsset posts an asset withdrawal
func (e *Exchange) WithdrawAsset(ctx context.Context, arg *AssetWithdrawalParams) (*WithdrawalResponse, error) {
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.ClientWithdrawID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	if arg.Timestamp.IsZero() {
		return nil, errInvalidTimestamp
	}
	creds, err := e.GetCredentials(context.Background())
	if err != nil {
		return nil, err
	}
	if arg.EthereumAddress == "" && creds.SubAccount == "" {
		return nil, errEthereumAddressMissing
	} else if arg.EthereumAddress == "" {
		arg.EthereumAddress = creds.SubAccount
	}
	if arg.L2Key == "" && creds.L2Key == "" {
		return nil, errL2KeyMissing
	} else if arg.L2Key == "" {
		arg.L2Key = creds.L2Key
	}
	if arg.ToChainID == "" {
		return nil, fmt.Errorf("%w, toChainID is required", errChainIDMissing)
	}
	if arg.L2SourceTokenID.IsEmpty() {
		return nil, fmt.Errorf("%w, l2SourceTokenId is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.L1TargetTokenID.IsEmpty() {
		return nil, fmt.Errorf("%w, l1TargetTokenId is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Nonce == "" {
		arg.Nonce = strconv.FormatInt(time.Now().UnixMilli(), 10)
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	params.Set("clientWithdrawId", arg.ClientWithdrawID)
	params.Set("timestamp", strconv.FormatInt(arg.Timestamp.UnixMilli(), 10))
	params.Set("ethAddress", arg.EthereumAddress)
	params.Set("subAccountId", arg.SubAccountID)
	params.Set("l2Key", arg.L2Key)
	params.Set("toChainId", arg.ToChainID)
	params.Set("l2SourceTokenId", arg.L2SourceTokenID.String())
	params.Set("l1TargetTokenId", arg.L1TargetTokenID.String())
	if arg.Fee != 0 {
		params.Set("fee", strconv.FormatFloat(arg.Fee, 'f', -1, 64))
	}
	params.Set("isFastWithdraw", strconv.FormatBool(arg.IsFastWithdraw))
	params.Set("nonce", arg.Nonce)
	signature, err := e.ProcessWithdrawalToAddressSignatureV3(ctx, arg)
	if err != nil {
		return nil, err
	}
	if arg.ZKAccountID == "" {
		return nil, errZeroKnowledgeAccountIDMissing
	}
	params.Set("zkAccountId", arg.ZKAccountID)
	params.Set("signature", signature)
	var resp *WithdrawalResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "v3/withdrawal", request.UnAuth, params, nil, &resp)
}

// ----------------------------------------------------- Private V2 Endpoints --------------------------------------------------------------------------------

// UserWithdrawalV2 withdraws an asset
func (e *Exchange) UserWithdrawalV2(ctx context.Context, arg *WithdrawalParams) (*WithdrawalResponse, error) {
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.EthereumAddress == "" {
		return nil, errEthereumAddressMissing
	}
	if arg.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	signature, err := e.ProcessWithdrawalSignature(ctx, arg)
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	params.Set("clientId", arg.ClientID)
	params.Set("expiration", strconv.FormatInt(arg.ExpEpoch, 10))
	params.Set("asset", arg.Asset.String())
	if err != nil {
		return nil, err
	}
	params.Set("signature", signature)
	params.Set("ethAddress", arg.EthereumAddress)
	var resp *WithdrawalResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "v2/create-withdrawal", request.UnAuth, params, nil, &resp)
}

// WithdrawalToAddressV1 withdraws as asset to an ethereum address
func (e *Exchange) WithdrawalToAddressV1(ctx context.Context, arg *WithdrawalToAddressParams) (*WithdrawalResponse, error) {
	return e.withdrawalToAddress(ctx, arg, "v1/create-withdrawal-to-address")
}

// WithdrawalToAddressV2 withdraws as asset to an ethereum address
func (e *Exchange) WithdrawalToAddressV2(ctx context.Context, arg *WithdrawalToAddressParams) (*WithdrawalResponse, error) {
	return e.withdrawalToAddress(ctx, arg, "v2/create-withdrawal-to-address")
}

func (e *Exchange) withdrawalToAddress(ctx context.Context, arg *WithdrawalToAddressParams, path string) (*WithdrawalResponse, error) {
	if *arg == (WithdrawalToAddressParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Asset.IsEmpty() {
		return nil, fmt.Errorf("%w, asset is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.EthereumAddress == "" {
		return nil, errEthereumAddressMissing
	}
	var err error
	signature, err := e.ProcessWithdrawalToAddressSignature(ctx, arg)
	if err != nil {
		return nil, err
	}
	if arg.ClientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	params.Set("asset", arg.Asset.String())
	params.Set("expiration", strconv.FormatInt(arg.ExpEpoch, 10))
	params.Set("clientId", arg.ClientOrderID)
	params.Set("ethAddress", arg.EthereumAddress)
	params.Set("signature", signature)
	var resp *WithdrawalResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, path, request.UnAuth, params, nil, &resp)
}

// CrossChainWithdrawalsV1 withdraws an asset through different chains
func (e *Exchange) CrossChainWithdrawalsV1(ctx context.Context, arg *FastWithdrawalParams) (*WithdrawalResponse, error) {
	return e.crossChainWithdrawals(ctx, arg, "v1/cross-chain-withdraw")
}

// CrossChainWithdrawalsV2 withdraaws an asse tthrough the v2 api
func (e *Exchange) CrossChainWithdrawalsV2(ctx context.Context, arg *FastWithdrawalParams) (*WithdrawalResponse, error) {
	return e.crossChainWithdrawals(ctx, arg, "v2/cross-chain-withdraw")
}

func (e *Exchange) crossChainWithdrawals(ctx context.Context, arg *FastWithdrawalParams, path string) (*WithdrawalResponse, error) {
	err := e.fillWithdrawalParams(arg)
	if err != nil {
		return nil, err
	}
	// TODO: signature validation and testing
	signature, err := e.ProcessConditionalTransfer(ctx, arg)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	params.Set("expiration", strconv.FormatInt(arg.Expiration, 10))
	params.Set("asset", arg.Asset.String())
	params.Set("fees", strconv.FormatFloat(arg.Fees, 'f', -1, 64))
	params.Set("chainId", arg.ChainID)
	params.Set("signature", signature)
	var resp *WithdrawalResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, path, request.UnAuth, params, nil, &resp)
}

// SendHTTPRequest sends an unauthenticated request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result interface{}, useAsItIs ...bool) error {
	endpointPath, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	var response interface{}
	if len(useAsItIs) > 0 && useAsItIs[0] {
		response = result
	} else {
		response = &struct {
			Data interface{} `json:"data"`
		}{
			Data: result,
		}
	}
	return e.SendPayload(ctx, f, func() (*request.Item, error) {
		return &request.Item{
			Method:        http.MethodGet,
			Path:          endpointPath + path,
			Result:        response,
			Verbose:       e.Verbose,
			HTTPDebugging: e.HTTPDebugging,
			HTTPRecording: e.HTTPRecording,
		}, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request.
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, f request.EndpointLimit, params url.Values, arg, result interface{}, timestamps ...int64) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpointPath, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	response := &UserResponse{
		Data: result,
	}
	if params != nil {
		path = common.EncodeURLValues(path, params)
	}
	var body io.Reader
	var payload []byte
	var dataString string
	if arg != nil {
		payload, err = json.Marshal(arg)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(payload)
	} else if method == http.MethodPost && params != nil {
		body = bytes.NewBuffer([]byte(params.Encode()))
		dataString = params.Encode()
	}
	err = e.SendPayload(ctx, f, func() (*request.Item, error) {
		timestamp := time.Now().UTC().UnixMilli()
		if len(timestamps) > 0 && timestamps[0] != 0 {
			timestamp = timestamps[0]
		}
		message := strconv.FormatInt(timestamp, 10) + method + ("/api/" + path) + dataString
		encodedSecret := base64.StdEncoding.EncodeToString([]byte(creds.Secret))
		var hmacSigned []byte
		hmacSigned, err = crypto.GetHMAC(crypto.HashSHA256,
			[]byte(message),
			[]byte(encodedSecret))
		if err != nil {
			return nil, err
		}
		headers := make(map[string]string)
		headers["APEX-API-KEY"] = creds.Key
		headers["APEX-SIGNATURE"] = base64.StdEncoding.EncodeToString(hmacSigned)
		headers["APEX-TIMESTAMP"] = strconv.FormatInt(timestamp, 10)
		headers["APEX-PASSPHRASE"] = creds.ClientID
		reqItem := &request.Item{
			Method:        method,
			Path:          endpointPath + path,
			Headers:       headers,
			Result:        response,
			Body:          body,
			Verbose:       e.Verbose,
			HTTPDebugging: e.HTTPDebugging,
			HTTPRecording: e.HTTPRecording,
		}
		return reqItem, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return err
	}
	if response.Code != 0 {
		return fmt.Errorf("code: %d msg: %q", response.Code, response.Message)
	}
	return nil
}

func randomClientID() string {
	rand.Seed(uint64(time.Now().UnixNano()))
	return strconv.FormatFloat(rand.Float64(), 'f', -1, 64)[2:]
}

func nonceFromClientID(clientID string) *big.Int {
	hasher := sha256.New()
	hasher.Write([]byte(clientID))
	hashBytes := hasher.Sum(nil)
	hashHex := hex.EncodeToString(hashBytes)
	nonce, _ := strconv.ParseUint(hashHex[0:8], 16, 64)
	return big.NewInt(int64(nonce))
}
