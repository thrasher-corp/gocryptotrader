package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var errMissingExpiryTimeParameter = errors.New("missing expiry date parameter")

// GetSupportCoins retrieves the currencies the trading statistics endpoints support
func (e *Exchange) GetSupportCoins(ctx context.Context) (*SupportCoinResponse, error) {
	var resp *SupportCoinResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSupportCoinEPL, http.MethodGet, "rubik/stat/trading-data/support-coin", nil, &resp, request.UnauthenticatedRequest)
}

// GetContractOpenInterestHistory retrieves a futures or perpetual swap contract's open interest history
func (e *Exchange) GetContractOpenInterestHistory(ctx context.Context, arg *ContractStatisticsRequest) ([]ContractOpenInterestHistoryResponse, error) {
	params, err := contractStatisticsParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []ContractOpenInterestHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, rubikGetContractOpenInterestHistoryEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/contracts/open-interest-history", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetTakerVolume retrieves a currency's taker sell and buy volumes
func (e *Exchange) GetTakerVolume(ctx context.Context, arg *TakerVolumeRequest) ([]TakerVolumeResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentType == "" {
		return nil, errInvalidInstrumentType
	}
	params, err := currencyStatisticsParams(arg.Currency, arg.Begin, arg.End, arg.Period)
	if err != nil {
		return nil, err
	}
	params.Set("instType", arg.InstrumentType)
	var resp []TakerVolumeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getTakerVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/taker-volume", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetContractTakerVolume retrieves a futures or perpetual swap contract's taker sell and buy volumes
func (e *Exchange) GetContractTakerVolume(ctx context.Context, arg *ContractTakerVolumeRequest) ([]TakerVolumeResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params, err := contractStatisticsParams(&ContractStatisticsRequest{InstrumentID: arg.InstrumentID, Period: arg.Period, Begin: arg.Begin, End: arg.End, Limit: arg.Limit})
	if err != nil {
		return nil, err
	}
	if arg.Unit != "" {
		params.Set("unit", arg.Unit)
	}
	var resp []TakerVolumeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, rubikContractTakerVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/taker-volume-contract", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetMarginLendingRatio retrieves the ratio of a currency's cumulative margin loans of quote currency to base currency
func (e *Exchange) GetMarginLendingRatio(ctx context.Context, arg *CurrencyStatisticsRequest) ([]MarginLendingRatioResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params, err := currencyStatisticsParams(arg.Currency, arg.Begin, arg.End, arg.Period)
	if err != nil {
		return nil, err
	}
	var resp []MarginLendingRatioResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMarginLendingRatioEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/margin/loan-ratio", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetTopTradersContractLongShortAccountRatio retrieves a contract's net long to short account ratio of its top traders,
// the 5% of traders with the largest open position value
func (e *Exchange) GetTopTradersContractLongShortAccountRatio(ctx context.Context, arg *ContractStatisticsRequest) ([]LongShortRatioResponse, error) {
	return e.getContractLongShortRatio(ctx, "rubik/stat/contracts/long-short-account-ratio-contract-top-trader", rubikTopTradersContractLongShortRatioEPL, arg)
}

// GetTopTradersContractLongShortPositionRatio retrieves a contract's long to short position ratio of its top traders,
// the 5% of traders with the largest open position value
func (e *Exchange) GetTopTradersContractLongShortPositionRatio(ctx context.Context, arg *ContractStatisticsRequest) ([]LongShortRatioResponse, error) {
	return e.getContractLongShortRatio(ctx, "rubik/stat/contracts/long-short-position-ratio-contract-top-trader", getTopTradersContractLongShortPositionRatioEPL, arg)
}

// GetContractLongShortAccountRatio retrieves a contract's long to short account ratio of all traders
func (e *Exchange) GetContractLongShortAccountRatio(ctx context.Context, arg *ContractStatisticsRequest) ([]LongShortRatioResponse, error) {
	return e.getContractLongShortRatio(ctx, "rubik/stat/contracts/long-short-account-ratio-contract", getContractLongShortAccountRatioEPL, arg)
}

func (e *Exchange) getContractLongShortRatio(ctx context.Context, path string, epl request.EndpointLimit, arg *ContractStatisticsRequest) ([]LongShortRatioResponse, error) {
	params, err := contractStatisticsParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []LongShortRatioResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodGet, common.EncodeURLValues(path, params), nil, &resp, request.UnauthenticatedRequest)
}

// GetLongShortRatio retrieves the ratio of users with net long to net short expiry futures and perpetual swap positions
// of a currency
func (e *Exchange) GetLongShortRatio(ctx context.Context, arg *CurrencyStatisticsRequest) ([]LongShortRatioResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params, err := currencyStatisticsParams(arg.Currency, arg.Begin, arg.End, arg.Period)
	if err != nil {
		return nil, err
	}
	var resp []LongShortRatioResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getLongShortRatioEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/contracts/long-short-account-ratio", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetContractsOpenInterestAndVolume retrieves a currency's expiry futures and perpetual swap open interest and trading
// volume, in USD
func (e *Exchange) GetContractsOpenInterestAndVolume(ctx context.Context, arg *CurrencyStatisticsRequest) ([]OpenInterestVolumeResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params, err := currencyStatisticsParams(arg.Currency, arg.Begin, arg.End, arg.Period)
	if err != nil {
		return nil, err
	}
	var resp []OpenInterestVolumeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getContractsOpenInterestAndVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/contracts/open-interest-volume", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetOptionsOpenInterestAndVolume retrieves a currency's option open interest and trading volume, in that currency
func (e *Exchange) GetOptionsOpenInterestAndVolume(ctx context.Context, arg *OptionStatisticsRequest) ([]OpenInterestVolumeResponse, error) {
	params, err := optionStatisticsParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []OpenInterestVolumeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getOptionsOpenInterestAndVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/option/open-interest-volume", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetPutCallRatio retrieves a currency's call to put option open interest and trading volume ratios
func (e *Exchange) GetPutCallRatio(ctx context.Context, arg *OptionStatisticsRequest) ([]PutCallRatioResponse, error) {
	params, err := optionStatisticsParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []PutCallRatioResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getPutCallRatioEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/option/open-interest-volume-ratio", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetOpenInterestAndVolumeExpiry retrieves a currency's call and put open interest and trading volume of each
// upcoming option expiry
func (e *Exchange) GetOpenInterestAndVolumeExpiry(ctx context.Context, arg *OptionStatisticsRequest) ([]OpenInterestVolumeExpiryResponse, error) {
	params, err := optionStatisticsParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []OpenInterestVolumeExpiryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getOpenInterestAndVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/option/open-interest-volume-expiry", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetOpenInterestAndVolumeStrike retrieves a currency's call and put open interest and trading volume of each strike
// of an option expiry
func (e *Exchange) GetOpenInterestAndVolumeStrike(ctx context.Context, arg *OpenInterestVolumeStrikeRequest) ([]OpenInterestVolumeStrikeResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.ExpiryTime.IsZero() {
		return nil, errMissingExpiryTimeParameter
	}
	params, err := optionStatisticsParams(&OptionStatisticsRequest{Currency: arg.Currency, Period: arg.Period})
	if err != nil {
		return nil, err
	}
	params.Set("expTime", arg.ExpiryTime.UTC().Format(expiryDateLayout))
	var resp []OpenInterestVolumeStrikeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getOpenInterestAndVolumeStrikeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/option/open-interest-volume-strike", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetTakerFlow retrieves a currency's option taker buy and sell volumes, which show whether traders are bullish or
// bearish on price and volatility
func (e *Exchange) GetTakerFlow(ctx context.Context, arg *OptionStatisticsRequest) (*TakerFlowResponse, error) {
	params, err := optionStatisticsParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *TakerFlowResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getTakerFlowEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/option/taker-block-volume", params), nil, &resp, request.UnauthenticatedRequest)
}

func contractStatisticsParams(arg *ContractStatisticsRequest) (url.Values, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if !arg.Begin.IsZero() && !arg.End.IsZero() && arg.Begin.After(arg.End) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	params.Set("instId", arg.InstrumentID)
	if err := setStatisticsPeriod(params, arg.Period, true); err != nil {
		return nil, err
	}
	setBeginEndParams(params, arg.Begin, arg.End)
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	return params, nil
}

func currencyStatisticsParams(ccy currency.Code, begin, end time.Time, period kline.Interval) (url.Values, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if !begin.IsZero() && !end.IsZero() && begin.After(end) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	params.Set("ccy", ccy.String())
	setBeginEndParams(params, begin, end)
	if err := setStatisticsPeriod(params, period, false); err != nil {
		return nil, err
	}
	return params, nil
}

func optionStatisticsParams(arg *OptionStatisticsRequest) (url.Values, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("ccy", arg.Currency.String())
	if err := setStatisticsPeriod(params, arg.Period, false); err != nil {
		return nil, err
	}
	return params, nil
}

// setStatisticsPeriod sets the period of a statistics query; appendUTC selects the UTC opening variant the contract
// endpoints offer for periods of six hours and longer
func setStatisticsPeriod(params url.Values, period kline.Interval, appendUTC bool) error {
	if period == 0 {
		return nil
	}
	periodString := IntervalFromString(period, appendUTC)
	if periodString == "" {
		return fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, period)
	}
	params.Set("period", periodString)
	return nil
}

func setBeginEndParams(params url.Values, begin, end time.Time) {
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
	}
}
