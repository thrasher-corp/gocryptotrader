package bybit

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	web3PEPEAddress    = "0x6982508145454ce325ddbe47a25d4ec3d2311933"
	web3BONKAddress    = "DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263"
	web3WIFAddress     = "EKpQGSJtjMFqKZ9KQanSqYXRcF8fBopzLHYxdM65zcjm"
	web3SOLUSDTAddress = "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"
	web3ETHUSDTAddress = "0xdac17f958d2ee523a2206206994597c13d831ec7"
	web3PoolAddress    = "0x1234567890abcdef"
)

func TestGetWeb3TradeQuote(t *testing.T) {
	t.Parallel()
	_, err := e.GetWeb3TradeQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWeb3TradeQuote must reject a nil request")
	r := &Web3TradeQuoteRequest{}
	_, err = e.GetWeb3TradeQuote(t.Context(), r)
	require.ErrorIs(t, err, errTradeTypeRequired, "GetWeb3TradeQuote must require a trade type")
	r.TradeType = 1
	_, err = e.GetWeb3TradeQuote(t.Context(), r)
	require.ErrorIs(t, err, errTokenCodeRequired, "GetWeb3TradeQuote must require a from token code")
	r.FromTokenCode = "CEX_1"
	_, err = e.GetWeb3TradeQuote(t.Context(), r)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "GetWeb3TradeQuote must require a positive from token amount")
	r.FromTokenAmount = 100
	_, err = e.GetWeb3TradeQuote(t.Context(), r)
	require.ErrorIs(t, err, errTokenCodeRequired, "GetWeb3TradeQuote must require a to token code")
	r.ToTokenCode = "DEX_123"
	r.QuoteMode = 1

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWeb3TradeQuote(t.Context(), r)
	require.NoError(t, err, "GetWeb3TradeQuote must not error")
	if mockTests {
		exp := &Web3TradeQuoteResponse{
			TradeType:          1,
			FromTokenCode:      "CEX_1",
			FromTokenAmount:    100,
			FromTokenAmountUSD: 100,
			ToTokenCode:        "DEX_123",
			ToTokenAmount:      12500000,
			ToTokenAmountUSD:   99.5,
			MinToTokenAmount:   12375000,
			Slippage:           0.005,
			Gas:                "0.000300",
			GasUSD:             0.3,
			PlatformFee:        0.2,
			PlatformFeeUSD:     0.2,
			SwapRate:           125000,
			LossRate:           0.005,
			QuoteData:          "eyJhbGciOiJIUzI1NiJ9",
			CorrectingCode:     "a1b2c3d4e5f6",
			QuoteMode:          1,
			QuoteDataID:        "QD_20240101_001",
			ExpireTime:         types.Time(time.Unix(1704067230, 0)),
			Timestamp:          types.Time(time.Unix(1704067200, 0)),
			ChargeAmount:       0.2,
			ModeEstimations: []Web3QuoteModeEstimation{
				{
					EstimatedGas:      12500000,
					EstimatedGasUSD:   0.3,
					EstimatedSlippage: 0.005,
				},
				{
					QuoteMode:         2,
					EstimatedGas:      12400000,
					EstimatedGasUSD:   0.35,
					EstimatedSlippage: 0.003,
				},
			},
		}
		assert.Equal(t, exp, result, "GetWeb3TradeQuote should decode every field")
		return
	}
	assert.NotEmpty(t, result.QuoteData, "GetWeb3TradeQuote should return quote data")
}

func TestExecuteWeb3Purchase(t *testing.T) {
	t.Parallel()
	_, err := e.ExecuteWeb3Purchase(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ExecuteWeb3Purchase must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ExecuteWeb3Purchase(t.Context(), &Web3TradeRequest{
		FromTokenCode:   "CEX_1",
		FromTokenAmount: 100,
		ToTokenCode:     "DEX_123",
		Slippage:        0.01,
		QuoteData:       "eyJhbGciOiJIUzI1NiJ9",
		Gas:             "0.0003",
		QuoteMode:       1,
		CorrectingCode:  "a1b2c3d4e5f6",
	})
	require.NoError(t, err, "ExecuteWeb3Purchase must not error")
	if mockTests {
		exp := &Web3OrderResponse{
			OrderNumber: "ORD_20240101_001",
		}
		assert.Equal(t, exp, result, "ExecuteWeb3Purchase should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderNumber, "ExecuteWeb3Purchase should return an order number")
}

func TestExecuteWeb3Redeem(t *testing.T) {
	t.Parallel()
	_, err := e.ExecuteWeb3Redeem(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ExecuteWeb3Redeem must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ExecuteWeb3Redeem(t.Context(), &Web3TradeRequest{
		FromTokenCode:   "DEX_123",
		FromTokenAmount: 10000000,
		ToTokenCode:     "CEX_1",
		Slippage:        0.01,
		QuoteData:       "eyJhbGciOiJIUzI1NiJ9",
		Gas:             "0.0003",
		CorrectingCode:  "f6e5d4c3b2a1",
		Tenant:          "BYBIT",
	})
	require.NoError(t, err, "ExecuteWeb3Redeem must not error")
	if mockTests {
		exp := &Web3OrderResponse{
			OrderNumber: "ORD_20240101_002",
		}
		assert.Equal(t, exp, result, "ExecuteWeb3Redeem should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderNumber, "ExecuteWeb3Redeem should return an order number")
}

func TestWeb3TradeBody(t *testing.T) {
	t.Parallel()
	_, err := web3TradeBody(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "web3TradeBody must reject a nil request")
	r := &Web3TradeRequest{}
	_, err = web3TradeBody(r)
	require.ErrorIs(t, err, errTokenCodeRequired, "web3TradeBody must require a from token code")
	r.FromTokenCode = "CEX_1"
	_, err = web3TradeBody(r)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "web3TradeBody must require a positive from token amount")
	r.FromTokenAmount = 100
	_, err = web3TradeBody(r)
	require.ErrorIs(t, err, errTokenCodeRequired, "web3TradeBody must require a to token code")
	r.ToTokenCode = "DEX_123"
	_, err = web3TradeBody(r)
	require.ErrorIs(t, err, errQuoteDataRequired, "web3TradeBody must require quote data")
	r.QuoteData = "eyJhbGciOiJIUzI1NiJ9"
	_, err = web3TradeBody(r)
	require.ErrorIs(t, err, errCorrectingCodeRequired, "web3TradeBody must require a correcting code")
	r.CorrectingCode = "a1b2c3d4e5f6"
	_, err = web3TradeBody(r)
	require.ErrorIs(t, err, errGasRequired, "web3TradeBody must require the quote's gas")
	// The trailing zeros would not survive a float64, so they show the quote's gas is passed on unchanged
	r.Gas = "0.000300"

	body, err := web3TradeBody(r)
	require.NoError(t, err, "web3TradeBody must not error")
	exp := map[string]any{
		"fromTokenCode":   "CEX_1",
		"fromTokenAmount": "100",
		"toTokenCode":     "DEX_123",
		"slippage":        "0",
		"quoteData":       "eyJhbGciOiJIUzI1NiJ9",
		"gas":             "0.000300",
		"quoteMode":       uint64(0),
		"correctingCode":  "a1b2c3d4e5f6",
	}
	assert.Equal(t, exp, body, "web3TradeBody should send the quote's gas unchanged and the required zero values, and leave out an unset tenant")
}

func TestGetWeb3PaymentTokenList(t *testing.T) {
	t.Parallel()
	_, err := e.GetWeb3PaymentTokenList(t.Context(), "", web3SOLUSDTAddress)
	require.ErrorIs(t, err, errChainCodeRequired, "GetWeb3PaymentTokenList must require a chain code")
	_, err = e.GetWeb3PaymentTokenList(t.Context(), "SOL", "")
	require.ErrorIs(t, err, errTokenAddressRequired, "GetWeb3PaymentTokenList must require a token address")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWeb3PaymentTokenList(t.Context(), "SOL", web3SOLUSDTAddress)
	require.NoError(t, err, "GetWeb3PaymentTokenList must not error")
	if mockTests {
		exp := []Web3PaymentTokenResponse{
			{
				TokenCode:         "CEX_1",
				Symbol:            currency.USDT,
				TokenDecimals:     6,
				TokenIconURLDay:   "https://example.com/usdt-day.png",
				TokenIconURLNight: "https://example.com/usdt-night.png",
				Limit:             50000,
				SupportChains: []string{
					"ETH",
					"BSC",
					"SOL",
					"TRX",
				},
			},
			{
				TokenCode:         "CEX_2",
				Symbol:            currency.USDC,
				TokenDecimals:     6,
				TokenIconURLDay:   "https://example.com/usdc-day.png",
				TokenIconURLNight: "https://example.com/usdc-night.png",
				Limit:             50000,
				SupportChains: []string{
					"ETH",
					"SOL",
					"BASE",
				},
			},
		}
		assert.Equal(t, exp, result, "GetWeb3PaymentTokenList should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetWeb3PaymentTokenList should return payment tokens")
}

func TestGetWeb3OrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetWeb3OrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWeb3OrderList must reject a nil request")
	r := &Web3OrderListRequest{}
	_, err = e.GetWeb3OrderList(t.Context(), r)
	require.ErrorIs(t, err, errPageLimitRequired, "GetWeb3OrderList must require a page limit")
	r.Limit = 20
	_, err = e.GetWeb3OrderList(t.Context(), r)
	require.ErrorIs(t, err, errPageIndexRequired, "GetWeb3OrderList must require a page index")
	r.PageIndex = 1
	r.TradeType = 1
	r.TokenCode = "DEX_123"
	r.OrderStatus = []uint64{2, 3}
	r.Days = 7
	r.Direction = "next"

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWeb3OrderList(t.Context(), r)
	require.NoError(t, err, "GetWeb3OrderList must not error")
	if mockTests {
		exp := &Web3OrderListResponse{
			Total:     2,
			PageIndex: 1,
			Orders: []Web3Order{
				{
					OrderType:             1,
					TradeType:             1,
					OrderNumber:           "ORD_20240101_001",
					OrderStatus:           2,
					FromTokenCode:         "CEX_1",
					FromTokenAmount:       100,
					FromTokenSymbol:       currency.USDT,
					FromTokenDecimals:     6,
					FromTokenIconURLDay:   "https://example.com/usdt-day.png",
					FromTokenIconURLNight: "https://example.com/usdt-night.png",
					FromChainCode:         "ETH",
					FromChainIconURL:      "https://example.com/eth.png",
					ToTokenCode:           "DEX_123",
					ToTokenAmount:         12450000,
					ToTokenSymbol:         currency.NewCode("PEPE"),
					ToTokenDecimals:       18,
					ToTokenIconURLDay:     "https://example.com/pepe-day.png",
					ToTokenIconURLNight:   "https://example.com/pepe-night.png",
					ToChainCode:           "ETH",
					ToChainIconURL:        "https://example.com/eth.png",
					GasTokenSymbol:        currency.ETH,
					GasOnchain:            0.0003,
					GasUSD:                0.3,
					PlatformFee:           0.2,
					PlatformFeeUSD:        0.2,
					CreateTime:            types.Time(time.Unix(1704067200, 0)),
					ExecutionTime:         types.Time(time.Unix(1704067230, 0)),
					Source:                "OPENAPI",
					SwapRate:              124500,
					ActualFromTokenAmount: 100,
				},
				{
					OrderType:             2,
					TradeType:             2,
					OrderNumber:           "ORD_20240101_002",
					OrderStatus:           3,
					FromTokenCode:         "DEX_123",
					FromTokenAmount:       10000000,
					FromTokenSymbol:       currency.NewCode("PEPE"),
					FromTokenDecimals:     18,
					FromTokenIconURLDay:   "https://example.com/pepe-day.png",
					FromTokenIconURLNight: "https://example.com/pepe-night.png",
					FromChainCode:         "ETH",
					FromChainIconURL:      "https://example.com/eth.png",
					ToTokenCode:           "CEX_1",
					ToTokenSymbol:         currency.USDT,
					ToTokenDecimals:       6,
					ToTokenIconURLDay:     "https://example.com/usdt-day.png",
					ToTokenIconURLNight:   "https://example.com/usdt-night.png",
					ToChainCode:           "ETH",
					ToChainIconURL:        "https://example.com/eth.png",
					GasTokenSymbol:        currency.ETH,
					QuoteMode:             2,
					CreateTime:            types.Time(time.Unix(1704070800, 0)),
					ExecutionTime:         types.Time(time.Unix(1704070860, 0)),
					FailureReasonCode:     "ERR106",
					Source:                "OPENAPI",
					SwapRate:              0.0000079,
				},
			},
		}
		assert.Equal(t, exp, result, "GetWeb3OrderList should decode every field")
		return
	}
	assert.NotZero(t, result.PageIndex, "GetWeb3OrderList should return the page index")
}

func TestGetWeb3BizTokenList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWeb3BizTokenList(t.Context(), 2)
	require.NoError(t, err, "GetWeb3BizTokenList must not error")
	if mockTests {
		exp := []Web3BizTokenResponse{
			{
				TokenCode:         "DEX_123",
				ChainCode:         "ETH",
				ChainIconURL:      "https://example.com/eth.png",
				TokenAddress:      "0x6982508145454ce325ddbe47a25d4ec3d2311933",
				Symbol:            currency.NewCode("PEPE"),
				TokenDecimals:     18,
				TokenIconURLDay:   "https://example.com/pepe-day.png",
				TokenIconURLNight: "https://example.com/pepe-night.png",
				CreateTime:        types.Time(time.Unix(1704067200, 0)),
				CreateTimeOnchain: types.Time(time.Unix(1682000000, 0)),
				MinOrderQuantity:  1,
				MaxOrderQuantity:  50000,
				TokenTags: []uint64{
					1,
					2,
				},
				PayTokenCodes: []string{
					"CEX_1",
					"CEX_2",
				},
			},
			{
				TokenCode:         "DEX_456",
				ChainCode:         "SOL",
				ChainIconURL:      "https://example.com/sol.png",
				TokenAddress:      "DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263",
				Symbol:            currency.NewCode("BONK"),
				TokenDecimals:     5,
				TokenIconURLDay:   "https://example.com/bonk-day.png",
				TokenIconURLNight: "https://example.com/bonk-night.png",
				CreateTime:        types.Time(time.Unix(1704153600, 0)),
				CreateTimeOnchain: types.Time(time.Unix(1671408000, 0)),
				RiskFlag:          true,
				MinOrderQuantity:  0.5,
				MaxOrderQuantity:  20000,
				TokenTags: []uint64{
					2,
				},
				PayTokenCodes: []string{
					"CEX_1",
				},
			},
		}
		assert.Equal(t, exp, result, "GetWeb3BizTokenList should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetWeb3BizTokenList should return tokens")
}

func TestGetWeb3TokenPriceList(t *testing.T) {
	t.Parallel()
	_, err := e.GetWeb3TokenPriceList(t.Context(), nil)
	require.ErrorIs(t, err, errTokenAddressRequired, "GetWeb3TokenPriceList must require a token")
	_, err = e.GetWeb3TokenPriceList(t.Context(), []Web3TokenAddress{{TokenAddress: web3PEPEAddress}})
	require.ErrorIs(t, err, errChainCodeRequired, "GetWeb3TokenPriceList must require each token's chain code")
	_, err = e.GetWeb3TokenPriceList(t.Context(), []Web3TokenAddress{{ChainCode: "ETH"}})
	require.ErrorIs(t, err, errTokenAddressRequired, "GetWeb3TokenPriceList must require each token's address")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWeb3TokenPriceList(t.Context(), []Web3TokenAddress{
		{ChainCode: "ETH", TokenAddress: web3PEPEAddress},
		{ChainCode: "SOL", TokenAddress: web3BONKAddress},
	})
	require.NoError(t, err, "GetWeb3TokenPriceList must not error")
	if mockTests {
		exp := &Web3TokenPriceListResponse{
			TokenPriceInfoList: []Web3TokenPriceInfo{
				{
					ChainCode:    "ETH",
					TokenAddress: "0x6982508145454ce325ddbe47a25d4ec3d2311933",
					Price:        0.00001,
					Change24Hour: 0.15,
					Volume24Hour: 125000000,
					MarketCap:    4200000000,
					Liquidity:    50000000,
					Holders:      250000,
				},
				{
					ChainCode:    "SOL",
					TokenAddress: "DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263",
					Price:        0.000021,
					Change24Hour: -0.04,
					Volume24Hour: 98000000,
					MarketCap:    1600000000,
					Liquidity:    21000000,
					Holders:      910000,
				},
			},
		}
		assert.Equal(t, exp, result, "GetWeb3TokenPriceList should decode every field")
		return
	}
	assert.NotEmpty(t, result.TokenPriceInfoList, "GetWeb3TokenPriceList should return prices")
}

func TestGetWeb3TokenDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetWeb3TokenDetails(t.Context(), "", web3PEPEAddress)
	require.ErrorIs(t, err, errChainCodeRequired, "GetWeb3TokenDetails must require a chain code")
	_, err = e.GetWeb3TokenDetails(t.Context(), "ETH", "")
	require.ErrorIs(t, err, errTokenAddressRequired, "GetWeb3TokenDetails must require a token address")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWeb3TokenDetails(t.Context(), "ETH", web3PEPEAddress)
	require.NoError(t, err, "GetWeb3TokenDetails must not error")
	if mockTests {
		exp := &Web3TokenDetailsResponse{
			TokenCode:           "DEX_123",
			ChainCode:           "ETH",
			ChainIconURL:        "https://example.com/eth.png",
			TokenAddress:        "0x6982508145454ce325ddbe47a25d4ec3d2311933",
			Symbol:              currency.NewCode("PEPE"),
			TokenDecimals:       18,
			TokenIconURLDay:     "https://example.com/pepe-day.png",
			TokenIconURLNight:   "https://example.com/pepe-night.png",
			MinOrderQuantity:    1,
			MaxOrderQuantity:    50000,
			MaxPositionQuantity: 100000,
			TokenDescription:    "PEPE is a meme token on Ethereum.",
			XURL:                "https://x.com/pepecoineth",
			OfficialURL:         "https://www.pepe.vip",
			WhitePaperURL:       "https://example.com/pepe-whitepaper.pdf",
			TokenTag:            2,
			RiskFlag:            true,
			CreateTimeOnchain:   types.Time(time.Unix(1682000000, 0)),
			Status:              2,
			TokenTags: []uint64{
				1,
				2,
			},
			ShowMessage: true,
			Content:     "PEPE will be delisted from Web3 trading.",
			LinkName:    "Announcement",
			LinkAddress: "https://example.com/announcement",
		}
		assert.Equal(t, exp, result, "GetWeb3TokenDetails should decode every field")
		return
	}
	assert.NotEmpty(t, result.TokenCode, "GetWeb3TokenDetails should return the token code")
}

func TestGetWeb3AssetList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWeb3AssetList(t.Context())
	require.NoError(t, err, "GetWeb3AssetList must not error")
	if mockTests {
		exp := &Web3AssetListResponse{
			TotalAssetUSD: 1250.5,
			AssetList: []Web3Asset{
				{
					ChainCode:         "ETH",
					ChainIconURL:      "https://example.com/eth.png",
					TokenAddress:      "0x6982508145454ce325ddbe47a25d4ec3d2311933",
					TokenCode:         "DEX_123",
					TokenSymbol:       currency.NewCode("PEPE"),
					TokenDecimals:     18,
					TokenIconURLDay:   "https://example.com/pepe-day.png",
					TokenIconURLNight: "https://example.com/pepe-night.png",
					TokenAmount:       50000000,
					TokenAmountUSD:    500,
					TradeFlag:         true,
					PNL:               75,
					PNLRatio:          0.15,
					CostPrice:         0.0000085,
					LastPrice:         0.00001,
					CostTotalValue:    425,
				},
				{
					ChainCode:            "SOL",
					ChainIconURL:         "https://example.com/sol.png",
					TokenAddress:         "EKpQGSJtjMFqKZ9KQanSqYXRcF8fBopzLHYxdM65zcjm",
					TokenCode:            "DEX_789",
					TokenSymbol:          currency.NewCode("WIF"),
					TokenDecimals:        6,
					TokenIconURLDay:      "https://example.com/wif-day.png",
					TokenIconURLNight:    "https://example.com/wif-night.png",
					TokenAmount:          200,
					TokenAmountUSD:       350,
					TradeFlag:            true,
					PNL:                  -50,
					PNLRatio:             -0.125,
					CostPrice:            2,
					LastPrice:            1.75,
					CostTotalValue:       400,
					AssetStatus:          1,
					AnnouncementURL:      "https://example.com/announcement/wif",
					EstimatedOfflineTime: types.Time(time.Unix(1704153600, 0)),
				},
				{
					ChainCode:         "SOL",
					ChainIconURL:      "https://example.com/sol.png",
					TokenAddress:      "DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263",
					TokenCode:         "DEX_456",
					TokenSymbol:       currency.NewCode("BONK"),
					TokenDecimals:     5,
					TokenIconURLDay:   "https://example.com/bonk-day.png",
					TokenIconURLNight: "https://example.com/bonk-night.png",
					TokenAmount:       19047619,
					TokenAmountUSD:    400.5,
					LastPrice:         0.000021,
					AssetStatus:       2,
					AnnouncementURL:   "https://example.com/announcement/bonk",
					DelistingTime:     types.Time(time.Unix(1704157200, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetWeb3AssetList should decode every field")
		return
	}
	assert.NotNil(t, result.AssetList, "GetWeb3AssetList should return an asset list")
}

func TestGetWeb3AssetDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetWeb3AssetDetail(t.Context(), "", web3BONKAddress)
	require.ErrorIs(t, err, errChainCodeRequired, "GetWeb3AssetDetail must require a chain code")
	_, err = e.GetWeb3AssetDetail(t.Context(), "SOL", "")
	require.ErrorIs(t, err, errTokenAddressRequired, "GetWeb3AssetDetail must require a token address")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name         string
		tokenAddress string
		exp          *Web3AssetDetailResponse
	}{
		{
			name:         "delisting soon",
			tokenAddress: web3WIFAddress,
			exp: &Web3AssetDetailResponse{
				AssetList: []Web3Asset{
					{
						ChainCode:            "SOL",
						ChainIconURL:         "https://example.com/sol.png",
						TokenAddress:         "EKpQGSJtjMFqKZ9KQanSqYXRcF8fBopzLHYxdM65zcjm",
						TokenCode:            "DEX_789",
						TokenSymbol:          currency.NewCode("WIF"),
						TokenDecimals:        6,
						TokenIconURLDay:      "https://example.com/wif-day.png",
						TokenIconURLNight:    "https://example.com/wif-night.png",
						TokenAmount:          200,
						TokenAmountUSD:       350,
						TradeFlag:            true,
						PNL:                  -50,
						PNLRatio:             -0.125,
						CostPrice:            2,
						LastPrice:            1.75,
						CostTotalValue:       400,
						AssetStatus:          1,
						AnnouncementURL:      "https://example.com/announcement/wif",
						EstimatedOfflineTime: types.Time(time.Unix(1704153600, 0)),
					},
				},
			},
		},
		{
			name:         "delisted",
			tokenAddress: web3BONKAddress,
			exp: &Web3AssetDetailResponse{
				AssetList: []Web3Asset{
					{
						ChainCode:         "SOL",
						ChainIconURL:      "https://example.com/sol.png",
						TokenAddress:      "DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263",
						TokenCode:         "DEX_456",
						TokenSymbol:       currency.NewCode("BONK"),
						TokenDecimals:     5,
						TokenIconURLDay:   "https://example.com/bonk-day.png",
						TokenIconURLNight: "https://example.com/bonk-night.png",
						TokenAmount:       19047619,
						TokenAmountUSD:    400.5,
						LastPrice:         0.000021,
						AssetStatus:       2,
						AnnouncementURL:   "https://example.com/announcement/bonk",
						DelistingTime:     types.Time(time.Unix(1704157200, 0)),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetWeb3AssetDetail(t.Context(), "SOL", tc.tokenAddress)
			require.NoError(t, err, "GetWeb3AssetDetail must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetWeb3AssetDetail should decode every field")
				return
			}
			assert.NotNil(t, result.AssetList, "GetWeb3AssetDetail should return an asset list")
		})
	}
}

func TestExecuteWeb3LPStake(t *testing.T) {
	t.Parallel()
	_, err := e.ExecuteWeb3LPStake(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ExecuteWeb3LPStake must reject a nil request")
	r := &Web3LPStakeRequest{}
	_, err = e.ExecuteWeb3LPStake(t.Context(), r)
	require.ErrorIs(t, err, errPoolAddressRequired, "ExecuteWeb3LPStake must require a pool address")
	r.PoolAddress = web3PoolAddress
	_, err = e.ExecuteWeb3LPStake(t.Context(), r)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "ExecuteWeb3LPStake must require a positive pay token amount")
	r.PayTokenAmount = 1000
	_, err = e.ExecuteWeb3LPStake(t.Context(), r)
	require.ErrorIs(t, err, errTokenCodeRequired, "ExecuteWeb3LPStake must require a pay token code")
	r.PayTokenCode = "CEX_1"
	r.RangeUpper, r.PriceUpper = 2000, 2000
	_, err = e.ExecuteWeb3LPStake(t.Context(), r)
	require.ErrorIs(t, err, errRangeAndPriceBoundsBothSet, "ExecuteWeb3LPStake must reject a range and a price upper bound")
	r.PriceUpper = 0
	r.RangeLower, r.PriceLower = 1800, 1800
	_, err = e.ExecuteWeb3LPStake(t.Context(), r)
	require.ErrorIs(t, err, errRangeAndPriceBoundsBothSet, "ExecuteWeb3LPStake must reject a range and a price lower bound")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *Web3LPStakeRequest
		exp  *Web3LPStakeResponse
	}{
		{
			name: "new position by range",
			req:  &Web3LPStakeRequest{PoolAddress: web3PoolAddress, PayTokenAmount: 1000, PayTokenCode: "CEX_1", RangeUpper: 2000, RangeLower: 1800},
			exp:  &Web3LPStakeResponse{PositionID: 12345, OrderNumber: "LP_ORD_20240101_001"},
		},
		{
			name: "existing position by price",
			req:  &Web3LPStakeRequest{PositionID: 12345, PoolAddress: web3PoolAddress, PayTokenAmount: 250, PayTokenCode: "CEX_2", PriceUpper: 2100.5, PriceLower: 1750.25},
			exp:  &Web3LPStakeResponse{PositionID: 12345, OrderNumber: "LP_ORD_20240101_002"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ExecuteWeb3LPStake(t.Context(), tc.req)
			require.NoError(t, err, "ExecuteWeb3LPStake must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ExecuteWeb3LPStake should decode every field")
				return
			}
			assert.NotEmpty(t, result.OrderNumber, "ExecuteWeb3LPStake should return an order number")
		})
	}
}

func TestExecuteWeb3LPRedeem(t *testing.T) {
	t.Parallel()
	_, err := e.ExecuteWeb3LPRedeem(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ExecuteWeb3LPRedeem must reject a nil request")
	r := &Web3LPRedeemRequest{}
	_, err = e.ExecuteWeb3LPRedeem(t.Context(), r)
	require.ErrorIs(t, err, errLPPositionIDRequired, "ExecuteWeb3LPRedeem must require a position ID")
	r.PositionID = 12345
	_, err = e.ExecuteWeb3LPRedeem(t.Context(), r)
	require.ErrorIs(t, err, errPoolAddressRequired, "ExecuteWeb3LPRedeem must require a pool address")
	r.PoolAddress = web3PoolAddress
	_, err = e.ExecuteWeb3LPRedeem(t.Context(), r)
	require.ErrorIs(t, err, errDercRatioInvalid, "ExecuteWeb3LPRedeem must require a positive redeem ratio")
	r.DercRatio = 1.01
	_, err = e.ExecuteWeb3LPRedeem(t.Context(), r)
	require.ErrorIs(t, err, errDercRatioInvalid, "ExecuteWeb3LPRedeem must reject a redeem ratio above 1")
	// Redeeming the whole position is a ratio of 1
	whole := *r
	whole.DercRatio = 1
	redeemed, err := restReplyTestInstance(t, map[string]string{"/v5/alpha/lp/redeem": `{"retCode":0,"retMsg":"OK","result":{"orderNo":"LP_REDEEM_1"},"retExtInfo":{},"time":1760000000000}`}).ExecuteWeb3LPRedeem(t.Context(), &whole)
	require.NoError(t, err, "ExecuteWeb3LPRedeem must accept a redeem ratio of 1")
	assert.Equal(t, &Web3OrderResponse{OrderNumber: "LP_REDEEM_1"}, redeemed, "ExecuteWeb3LPRedeem should return the redemption's order number")
	r.DercRatio = 0.5
	r.ReceiveTokenCode = "CEX_1"

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ExecuteWeb3LPRedeem(t.Context(), r)
	require.NoError(t, err, "ExecuteWeb3LPRedeem must not error")
	if mockTests {
		exp := &Web3OrderResponse{
			OrderNumber: "LP_REDEEM_20240101_001",
		}
		assert.Equal(t, exp, result, "ExecuteWeb3LPRedeem should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderNumber, "ExecuteWeb3LPRedeem should return an order number")
}

func TestGetWeb3LPOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetWeb3LPOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWeb3LPOrderList must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWeb3LPOrderList(t.Context(), &Web3LPOrderListRequest{
		OrderType:   1,
		TokenCode:   "CEX_1",
		OrderStatus: []uint64{2, 3},
		Days:        7,
		Limit:       20,
		PageIndex:   1,
		PoolAddress: web3PoolAddress,
	})
	require.NoError(t, err, "GetWeb3LPOrderList must not error")
	if mockTests {
		exp := &Web3LPOrderListResponse{
			Total:     2,
			PageIndex: 1,
			Orders: []Web3LPOrder{
				{
					OrderType:         1,
					OrderNumber:       "LP_ORD_20240101_001",
					OrderStatus:       2,
					PoolAddress:       "0x1234567890abcdef",
					PoolName:          "ETH-USDC Pool",
					PositionID:        12345,
					TokenCode:         "CEX_1",
					TokenSymbol:       currency.USDT,
					TokenIconURLDay:   "https://example.com/usdt-day.png",
					TokenIconURLNight: "https://example.com/usdt-night.png",
					Amount:            1000,
					ChainCode:         "ETH",
					ChainIconURL:      "https://example.com/eth.png",
					GasTokenSymbol:    currency.ETH,
					GasOnchain:        0.0004,
					GasUSD:            1.2,
					PlatformFee:       1,
					PlatformFeeUSD:    1,
					CreateTime:        types.Time(time.Unix(1704067200, 0)),
					ExecutionTime:     types.Time(time.Unix(1704067230, 0)),
				},
				{
					OrderType:         2,
					OrderNumber:       "LP_REDEEM_20240101_001",
					OrderStatus:       3,
					PoolAddress:       "0x1234567890abcdef",
					PoolName:          "ETH-USDC Pool",
					PositionID:        12345,
					TokenCode:         "CEX_1",
					TokenSymbol:       currency.USDT,
					TokenIconURLDay:   "https://example.com/usdt-day.png",
					TokenIconURLNight: "https://example.com/usdt-night.png",
					Amount:            500,
					ChainCode:         "ETH",
					ChainIconURL:      "https://example.com/eth.png",
					GasTokenSymbol:    currency.ETH,
					CreateTime:        types.Time(time.Unix(1704153600, 0)),
					ExecutionTime:     types.Time(time.Unix(1704153660, 0)),
					FailureReason:     "On-chain execution failed",
					DercRatio:         0.5,
				},
			},
		}
		assert.Equal(t, exp, result, "GetWeb3LPOrderList should decode every field")
		return
	}
	assert.NotZero(t, result.PageIndex, "GetWeb3LPOrderList should return the page index")
}

func TestGetWeb3LPPayTokenList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWeb3LPPayTokenList(t.Context(), "ETH", web3ETHUSDTAddress)
	require.NoError(t, err, "GetWeb3LPPayTokenList must not error")
	if mockTests {
		exp := &Web3LPPayTokenListResponse{
			Tokens: []Web3LPPayToken{
				{
					TokenCode:         "CEX_1",
					TokenSymbol:       currency.USDT,
					ChainCode:         "ETH",
					ChainIconURL:      "https://example.com/eth.png",
					Decimals:          6,
					AvailableBalance:  1000,
					TokenIconURLDay:   "https://example.com/usdt-day.png",
					TokenIconURLNight: "https://example.com/usdt-night.png",
					MinStakeAmount:    10,
					MaxStakeAmount:    100000,
				},
			},
		}
		assert.Equal(t, exp, result, "GetWeb3LPPayTokenList should decode every field")
		return
	}
	assert.NotEmpty(t, result.Tokens, "GetWeb3LPPayTokenList should return tokens")
}

func TestGetWeb3LPPayTokenPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetWeb3LPPayTokenPrice(t.Context(), nil, "ETH")
	require.ErrorIs(t, err, errTokenCodeRequired, "GetWeb3LPPayTokenPrice must require a token code")
	_, err = e.GetWeb3LPPayTokenPrice(t.Context(), []string{"CEX_1", ""}, "ETH")
	require.ErrorIs(t, err, errTokenCodeRequired, "GetWeb3LPPayTokenPrice must reject an empty token code")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWeb3LPPayTokenPrice(t.Context(), []string{"CEX_1", "CEX_2"}, "ETH")
	require.NoError(t, err, "GetWeb3LPPayTokenPrice must not error")
	if mockTests {
		exp := &Web3LPPayTokenPriceResponse{
			Prices: []Web3LPPayTokenPrice{
				{
					TokenCode:   "CEX_1",
					TokenSymbol: currency.USDT,
					PriceUSD:    1,
					ChainCode:   "ETH",
					UpdateTime:  types.Time(time.Unix(1704067200, 0)),
				},
				{
					TokenCode:   "CEX_2",
					TokenSymbol: currency.USDC,
					PriceUSD:    0.9998,
					ChainCode:   "ETH",
					UpdateTime:  types.Time(time.Unix(1704067200, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetWeb3LPPayTokenPrice should decode every field")
		return
	}
	assert.NotEmpty(t, result.Prices, "GetWeb3LPPayTokenPrice should return prices")
}

func TestGetWeb3LPPoolInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetWeb3LPPoolInfo(t.Context(), "")
	require.ErrorIs(t, err, errPoolAddressRequired, "GetWeb3LPPoolInfo must require a pool address")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWeb3LPPoolInfo(t.Context(), web3PoolAddress)
	require.NoError(t, err, "GetWeb3LPPoolInfo must not error")
	if mockTests {
		exp := &Web3LPPoolInfoResponse{
			PoolAddress:     "0x1234567890abcdef",
			PoolName:        "ETH-USDC Pool",
			APY:             12.5,
			TVL:             1000000,
			FeeRate:         0.003,
			Token0Symbol:    currency.ETH,
			Token0Reserve:   500,
			Token1Symbol:    currency.USDC,
			Token1Reserve:   1000000,
			PriceRangeLower: 1800,
			PriceRangeUpper: 2200,
			CurrentPrice:    2000,
		}
		assert.Equal(t, exp, result, "GetWeb3LPPoolInfo should decode every field")
		return
	}
	assert.NotEmpty(t, result.PoolAddress, "GetWeb3LPPoolInfo should return the pool address")
}

func TestGetWeb3LPPoolList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWeb3LPPoolList(t.Context(), currency.ETH)
	require.NoError(t, err, "GetWeb3LPPoolList must not error")
	if mockTests {
		exp := &Web3LPPoolListResponse{
			Pools: []Web3LPPool{
				{
					PoolAddress:        "0x1234567890abcdef",
					PoolName:           "ETH-USDC Pool",
					PoolTag:            "Trending",
					APY:                12.5,
					TVL:                1000000,
					Token0Symbol:       currency.ETH,
					Token0IconURLDay:   "https://example.com/eth-day.png",
					Token0IconURLNight: "https://example.com/eth-night.png",
					Token1Symbol:       currency.USDC,
					Token1IconURLDay:   "https://example.com/usdc-day.png",
					Token1IconURLNight: "https://example.com/usdc-night.png",
					ChainCode:          "ETH",
					ChainIconURL:       "https://example.com/eth.png",
				},
			},
		}
		assert.Equal(t, exp, result, "GetWeb3LPPoolList should decode every field")
		return
	}
	assert.NotNil(t, result.Pools, "GetWeb3LPPoolList should return pools")
}

func TestGetWeb3LPPositionList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWeb3LPPositionList(t.Context())
	require.NoError(t, err, "GetWeb3LPPositionList must not error")
	if mockTests {
		exp := &Web3LPPositionListResponse{
			Positions: []Web3LPPosition{
				{
					PositionID:        12345,
					PoolAddress:       "0x1234567890abcdef",
					PoolName:          "ETH-USDC Pool",
					StakedAmount:      1000,
					StakedTokenCode:   "CEX_1",
					StakedTokenSymbol: currency.USDT,
					CurrentValueUSD:   1050.25,
					EarnedRewardsUSD:  50.25,
					UnrealisedPNL:     30,
					RealisedPNL:       20.25,
					APY:               12.5,
					Token0Amount:      0.3,
					Token0Symbol:      currency.ETH,
					Token1Amount:      600,
					Token1Symbol:      currency.USDC,
					RangeUpper:        2000,
					RangeLower:        1800,
					CreateTime:        types.Time(time.Unix(1704067200, 0)),
					UpdateTime:        types.Time(time.Unix(1704067260, 0)),
					Status:            1,
				},
			},
		}
		assert.Equal(t, exp, result, "GetWeb3LPPositionList should decode every field")
		return
	}
	assert.NotNil(t, result.Positions, "GetWeb3LPPositionList should return positions")
}

func TestGetPredictionEngineStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPredictionEngineStatus(t.Context())
	require.NoError(t, err, "GetPredictionEngineStatus must not error")
	if mockTests {
		exp := &PredictionEngineStatusResponse{
			Available: true,
		}
		assert.Equal(t, exp, result, "GetPredictionEngineStatus should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPredictionEngineStatus should return a status")
}

func TestGetPredictionPaymentTokenList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPredictionPaymentTokenList(t.Context())
	require.NoError(t, err, "GetPredictionPaymentTokenList must not error")
	if mockTests {
		exp := []PredictionPaymentTokenResponse{
			{
				TokenCode:         "USDC",
				Symbol:            currency.USDC,
				TokenDecimals:     6,
				TokenIconURLDay:   "https://example.com/usdc-day.png",
				TokenIconURLNight: "https://example.com/usdc-night.png",
				SupportChains: []string{
					"POLYGON",
				},
			},
		}
		assert.Equal(t, exp, result, "GetPredictionPaymentTokenList should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetPredictionPaymentTokenList should return payment tokens")
}

func TestGetPredictionEventDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetPredictionEventDetail(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPredictionEventDetail must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPredictionEventDetail(t.Context(), &PredictionEventDetailRequest{
		EventID:        "event_123",
		Slug:           "will-argentina-win-world-cup-2026",
		HasMoreMarkets: true,
	})
	require.NoError(t, err, "GetPredictionEventDetail must not error")
	if mockTests {
		exp := &PredictionEventDetailResponse{
			EventID:     "event_123",
			Slug:        "will-argentina-win-world-cup-2026",
			Title:       "Will Argentina win the 2026 World Cup?",
			Description: "Argentina must win the FIFA 2026 World Cup final.",
			Category:    "Sports",
			EndDate:     types.Time(time.Unix(1750000000, 0)),
			Resolved:    true,
			Volume:      1500000,
			Liquidity:   500000,
			Markets: []PredictionEventMarket{
				{
					TokenID:   "token_yes_123",
					Outcome:   "YES",
					Price:     0.65,
					Side:      1,
					Volume:    900000,
					Liquidity: 300000,
				},
				{
					TokenID:   "token_no_123",
					Outcome:   "NO",
					Price:     0.35,
					Side:      2,
					Volume:    600000,
					Liquidity: 200000,
				},
			},
		}
		assert.Equal(t, exp, result, "GetPredictionEventDetail should decode every field")
		return
	}
	assert.NotEmpty(t, result.EventID, "GetPredictionEventDetail should return the event ID")
}

func TestGetPredictionOrderEstimate(t *testing.T) {
	t.Parallel()
	_, err := e.GetPredictionOrderEstimate(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPredictionOrderEstimate must reject a nil request")
	r := &PredictionOrderEstimateRequest{}
	_, err = e.GetPredictionOrderEstimate(t.Context(), r)
	require.ErrorIs(t, err, errOutcomeTokenIDRequired, "GetPredictionOrderEstimate must require a token ID")
	r.TokenID = "token_yes_123"
	_, err = e.GetPredictionOrderEstimate(t.Context(), r)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetPredictionOrderEstimate must require a side")
	r.Side = 1
	_, err = e.GetPredictionOrderEstimate(t.Context(), r)
	require.ErrorIs(t, err, errEventIDRequired, "GetPredictionOrderEstimate must require an event ID")
	r.EventID = "event_123"
	_, err = e.GetPredictionOrderEstimate(t.Context(), r)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "GetPredictionOrderEstimate must require a positive amount")
	r.Amount = 100
	_, err = e.GetPredictionOrderEstimate(t.Context(), r)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "GetPredictionOrderEstimate must require an order type")
	r.OrderType = 1
	_, err = e.GetPredictionOrderEstimate(t.Context(), r)
	require.ErrorIs(t, err, errTokenCodeRequired, "GetPredictionOrderEstimate must require a pay token code to buy")
	r.PayTokenCode = "USDC"

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPredictionOrderEstimate(t.Context(), r)
	require.NoError(t, err, "GetPredictionOrderEstimate must not error")
	if mockTests {
		exp := &PredictionOrderEstimateResponse{
			AveragePrice:     0.65,
			EstimatedCost:    100,
			EstimatedReceive: 153.84,
			ToWin:            153.84,
			FeeAmount:        1,
			FeeDetail: PredictionFeeDetail{
				ServerFee:     0.7,
				PolymarketFee: 0.3,
			},
			Slippage: 0.05,
		}
		assert.Equal(t, exp, result, "GetPredictionOrderEstimate should decode every field")
		return
	}
	assert.NotZero(t, result.EstimatedCost, "GetPredictionOrderEstimate should return an estimated cost")
}

func TestExecutePredictionBuy(t *testing.T) {
	t.Parallel()
	_, err := e.ExecutePredictionBuy(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ExecutePredictionBuy must reject a nil request")
	r := &PredictionBuyRequest{}
	_, err = e.ExecutePredictionBuy(t.Context(), r)
	require.ErrorIs(t, err, errOutcomeTokenIDRequired, "ExecutePredictionBuy must require a token ID")
	r.TokenID = "token_yes_123"
	_, err = e.ExecutePredictionBuy(t.Context(), r)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "ExecutePredictionBuy must require a positive amount")
	r.Amount = 100
	_, err = e.ExecutePredictionBuy(t.Context(), r)
	require.ErrorIs(t, err, errTokenCodeRequired, "ExecutePredictionBuy must require a pay token code")
	r.PayTokenCode = "USDC"
	_, err = e.ExecutePredictionBuy(t.Context(), r)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "ExecutePredictionBuy must require an order type")
	r.OrderType = 1
	_, err = e.ExecutePredictionBuy(t.Context(), r)
	require.ErrorIs(t, err, errEventIDRequired, "ExecutePredictionBuy must require an event ID")
	r.EventID = "event_123"
	r.Slippage = 0.05

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ExecutePredictionBuy(t.Context(), r)
	require.NoError(t, err, "ExecutePredictionBuy must not error")
	if mockTests {
		exp := &Web3OrderResponse{
			OrderNumber: "ord_buy_20240101_001",
		}
		assert.Equal(t, exp, result, "ExecutePredictionBuy should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderNumber, "ExecutePredictionBuy should return an order number")
}

func TestExecutePredictionSell(t *testing.T) {
	t.Parallel()
	_, err := e.ExecutePredictionSell(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ExecutePredictionSell must reject a nil request")
	r := &PredictionSellRequest{}
	_, err = e.ExecutePredictionSell(t.Context(), r)
	require.ErrorIs(t, err, errOutcomeTokenIDRequired, "ExecutePredictionSell must require a token ID")
	r.TokenID = "token_yes_123"
	_, err = e.ExecutePredictionSell(t.Context(), r)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "ExecutePredictionSell must require a positive size")
	r.Size = 50
	_, err = e.ExecutePredictionSell(t.Context(), r)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "ExecutePredictionSell must require an order type")
	r.OrderType = 1
	_, err = e.ExecutePredictionSell(t.Context(), r)
	require.ErrorIs(t, err, errEventIDRequired, "ExecutePredictionSell must require an event ID")
	r.EventID = "event_123"
	r.Slippage = 0.05
	r.ToTokenCode = "USDC"

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ExecutePredictionSell(t.Context(), r)
	require.NoError(t, err, "ExecutePredictionSell must not error")
	if mockTests {
		exp := &Web3OrderResponse{
			OrderNumber: "ord_sell_20240101_001",
		}
		assert.Equal(t, exp, result, "ExecutePredictionSell should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderNumber, "ExecutePredictionSell should return an order number")
}

func TestGetPredictionOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetPredictionOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPredictionOrderList must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPredictionOrderList(t.Context(), &PredictionOrderListRequest{
		Status:    2,
		TokenID:   "token_yes_123",
		EventID:   "event_123",
		Side:      1,
		Days:      7,
		Limit:     20,
		PageIndex: 1,
	})
	require.NoError(t, err, "GetPredictionOrderList must not error")
	if mockTests {
		exp := &PredictionOrderListResponse{
			Orders: []PredictionOrder{
				{
					OrderNumber: "ord_buy_20240101_001",
					TokenID:     "token_yes_123",
					EventID:     "event_123",
					Side:        1,
					OrderType:   1,
					OrderAmount: 100,
					FilledSize:  153.84,
					FilledPrice: 0.65,
					TotalFee:    1,
					Status:      2,
					CreatedAt:   types.Time(time.Unix(1704067200, 0)),
					UpdatedAt:   types.Time(time.Unix(1704067205, 0)),
				},
			},
			Total:     1,
			PageIndex: 1,
		}
		assert.Equal(t, exp, result, "GetPredictionOrderList should decode every field")
		return
	}
	assert.NotZero(t, result.PageIndex, "GetPredictionOrderList should return the page index")
}

func TestGetPredictionOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetPredictionOrderBook(t.Context(), nil)
	require.ErrorIs(t, err, errOutcomeTokenIDRequired, "GetPredictionOrderBook must require a token ID")
	_, err = e.GetPredictionOrderBook(t.Context(), []string{""})
	require.ErrorIs(t, err, errOutcomeTokenIDRequired, "GetPredictionOrderBook must reject an empty token ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPredictionOrderBook(t.Context(), []string{"token_yes_123"})
	require.NoError(t, err, "GetPredictionOrderBook must not error")
	if mockTests {
		exp := []PredictionOrderBookResponse{
			{
				TokenID: "token_yes_123",
				Bids: []PredictionOrderBookLevel{
					{
						Price: 0.64,
						Size:  1000,
					},
					{
						Price: 0.63,
						Size:  2500,
					},
				},
				Asks: []PredictionOrderBookLevel{
					{
						Price: 0.66,
						Size:  800,
					},
					{
						Price: 0.67,
						Size:  1500,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetPredictionOrderBook should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetPredictionOrderBook should return order books")
}

func TestGetPredictionTokenPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetPredictionTokenPrice(t.Context(), nil)
	require.ErrorIs(t, err, errOutcomeTokenIDRequired, "GetPredictionTokenPrice must require a token ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPredictionTokenPrice(t.Context(), []string{"token_yes_123", "token_no_123"})
	require.NoError(t, err, "GetPredictionTokenPrice must not error")
	if mockTests {
		exp := []PredictionTokenPriceResponse{
			{
				TokenID:      "token_yes_123",
				BestBid:      0.64,
				BestAsk:      0.66,
				LastPrice:    0.65,
				Volume24Hour: 50000,
				UpdateTime:   types.Time(time.Unix(1704067200, 0)),
			},
			{
				TokenID:      "token_no_123",
				BestBid:      0.34,
				BestAsk:      0.36,
				LastPrice:    0.35,
				Volume24Hour: 48000,
				UpdateTime:   types.Time(time.Unix(1704067200, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetPredictionTokenPrice should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetPredictionTokenPrice should return prices")
}

func TestPredictionTokenIDsBody(t *testing.T) {
	t.Parallel()
	_, err := predictionTokenIDsBody(nil)
	require.ErrorIs(t, err, errOutcomeTokenIDRequired, "predictionTokenIDsBody must require a token ID")
	_, err = predictionTokenIDsBody([]string{"token_yes_123", ""})
	require.ErrorIs(t, err, errOutcomeTokenIDRequired, "predictionTokenIDsBody must reject an empty token ID")
	body, err := predictionTokenIDsBody([]string{"token_yes_123", "token_no_123"})
	require.NoError(t, err, "predictionTokenIDsBody must not error")
	assert.Equal(t, map[string]any{"tokenIds": []string{"token_yes_123", "token_no_123"}}, body, "predictionTokenIDsBody should send the token IDs")
}

func TestGetPredictionPriceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPredictionPriceHistory(t.Context(), "", "1H")
	require.ErrorIs(t, err, errOutcomeTokenIDRequired, "GetPredictionPriceHistory must require a token ID")
	_, err = e.GetPredictionPriceHistory(t.Context(), "token_yes_123", "")
	require.ErrorIs(t, err, errPriceIntervalRequired, "GetPredictionPriceHistory must require an interval")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPredictionPriceHistory(t.Context(), "token_yes_123", "1H")
	require.NoError(t, err, "GetPredictionPriceHistory must not error")
	if mockTests {
		exp := &PredictionPriceHistoryResponse{
			TokenID:  "token_yes_123",
			Interval: "1H",
			List: []PredictionCandle{
				{
					OpenTime: types.Time(time.Unix(1704067200, 0)),
					Open:     0.63,
					High:     0.68,
					Low:      0.62,
					Close:    0.65,
					Volume:   12500,
				},
				{
					OpenTime: types.Time(time.Unix(1704063600, 0)),
					Open:     0.6,
					High:     0.64,
					Low:      0.59,
					Close:    0.63,
					Volume:   9800,
				},
			},
		}
		assert.Equal(t, exp, result, "GetPredictionPriceHistory should decode every field")
		return
	}
	assert.NotEmpty(t, result.TokenID, "GetPredictionPriceHistory should return the token ID")
}

func TestGetPredictionPositionList(t *testing.T) {
	t.Parallel()
	_, err := e.GetPredictionPositionList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPredictionPositionList must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPredictionPositionList(t.Context(), &PredictionPositionListRequest{
		TokenID:   "token_yes_123",
		EventID:   "event_123",
		Limit:     20,
		PageIndex: 1,
	})
	require.NoError(t, err, "GetPredictionPositionList must not error")
	if mockTests {
		exp := &PredictionPositionListResponse{
			Positions: []PredictionPosition{
				{
					PositionID:        "pos_001",
					TokenID:           "token_yes_123",
					EventID:           "event_123",
					OutcomeName:       "YES",
					Shares:            153.84,
					Cost:              100,
					AveragePrice:      0.65,
					CurrentPrice:      0.72,
					Value:             110.77,
					UnrealisedPNL:     10.77,
					UnrealisedPNLRate: 0.107,
					CreatedAt:         types.Time(time.Unix(1704067200, 0)),
					Finished:          true,
				},
			},
			Total:         1,
			TotalAssetUSD: 110.77,
		}
		assert.Equal(t, exp, result, "GetPredictionPositionList should decode every field")
		return
	}
	assert.NotNil(t, result.Positions, "GetPredictionPositionList should return positions")
}

func TestGetPredictionPositionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPredictionPositionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPredictionPositionHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPredictionPositionHistory(t.Context(), &PredictionPositionHistoryRequest{
		TokenID:   "token_yes_456",
		EventID:   "event_456",
		Result:    1,
		Days:      30,
		Limit:     20,
		PageIndex: 1,
	})
	require.NoError(t, err, "GetPredictionPositionHistory must not error")
	if mockTests {
		exp := &PredictionPositionHistoryResponse{
			Positions: []PredictionClosedPosition{
				{
					PositionID:      "pos_456",
					TokenID:         "token_yes_456",
					EventID:         "event_456",
					OutcomeName:     "YES",
					Shares:          200,
					Cost:            140,
					AveragePrice:    0.7,
					ExitPrice:       1,
					RealisedPNL:     60,
					RealisedPNLRate: 0.429,
					Result:          1,
					ClosedAt:        types.Time(time.Unix(1704153600, 0)),
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetPredictionPositionHistory should decode every field")
		return
	}
	assert.NotNil(t, result.Positions, "GetPredictionPositionHistory should return positions")
}

func TestGetPredictionPortfolioSummary(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPredictionPortfolioSummary(t.Context(), 1)
	require.NoError(t, err, "GetPredictionPortfolioSummary must not error")
	if mockTests {
		exp := &PredictionPortfolioSummaryResponse{
			PositionValue:    580,
			PositionValueUSD: 580,
			BiggestWin:       120,
			WinRate:          0.67,
			WinCount:         10,
			LossCount:        5,
		}
		assert.Equal(t, exp, result, "GetPredictionPortfolioSummary should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPredictionPortfolioSummary should return a summary")
}

func TestGetPredictionSideMarketList(t *testing.T) {
	t.Parallel()
	_, err := e.GetPredictionSideMarketList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPredictionSideMarketList must reject a nil request")
	_, err = e.GetPredictionSideMarketList(t.Context(), &PredictionSideMarketListRequest{})
	require.ErrorIs(t, err, errEventIDRequired, "GetPredictionSideMarketList must require an event ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPredictionSideMarketList(t.Context(), &PredictionSideMarketListRequest{
		EventID:    "event_123",
		SortBy:     "volume",
		MarketType: 1,
		Limit:      20,
		PageIndex:  1,
	})
	require.NoError(t, err, "GetPredictionSideMarketList must not error")
	if mockTests {
		exp := &PredictionSideMarketListResponse{
			List: []PredictionSideMarket{
				{
					MarketID:   "market_001",
					EventID:    "event_123",
					MarketType: 1,
					Question:   "Will Argentina win the 2026 FIFA World Cup?",
					Tokens: []PredictionSideMarketToken{
						{
							TokenID:   "token_yes_123",
							Outcome:   "YES",
							Price:     0.65,
							Liquidity: 100000,
						},
						{
							TokenID:   "token_no_123",
							Outcome:   "NO",
							Price:     0.35,
							Liquidity: 95000,
						},
					},
					Volume24Hour: 50000,
					EndDate:      types.Time(time.Unix(1750000000, 0)),
					Status:       1,
				},
			},
			Total:     1,
			PageIndex: 1,
			PageSize:  20,
		}
		assert.Equal(t, exp, result, "GetPredictionSideMarketList should decode every field")
		return
	}
	assert.NotZero(t, result.PageIndex, "GetPredictionSideMarketList should return the page index")
}

func TestGetPredictionSportsMatchList(t *testing.T) {
	t.Parallel()
	_, err := e.GetPredictionSportsMatchList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPredictionSportsMatchList must reject a nil request")
	r := &PredictionSportsMatchListRequest{}
	_, err = e.GetPredictionSportsMatchList(t.Context(), r)
	require.ErrorIs(t, err, errEventTypeRequired, "GetPredictionSportsMatchList must require an event type")
	r.EventType = 1
	r.StartTime, r.EndTime = time.UnixMilli(1751500800000), time.UnixMilli(1748995200000)
	_, err = e.GetPredictionSportsMatchList(t.Context(), r)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPredictionSportsMatchList must reject a reversed window")
	r.StartTime, r.EndTime = r.EndTime, r.StartTime
	r.Status = 2
	r.StageCode = "Groups"
	r.Limit = 20
	r.PageIndex = 1

	// Upcoming matches start in the future, so a window that starts after now is sent
	var body string
	upcoming := newTransportTestExchange(t, func(w http.ResponseWriter, req *http.Request) {
		b, err := io.ReadAll(req.Body)
		assert.NoError(t, err, "reading the request body should not error")
		body = string(b)
		_, err = fmt.Fprint(w, `{"retCode":0,"retMsg":"success","result":{"list":[],"total":0,"pageIndex":1,"pageSize":20},"retExtInfo":{},"time":1792143027943}`)
		assert.NoError(t, err, "writing the response should not error")
	})
	start, end := time.Now().Add(24*time.Hour), time.Now().Add(8*24*time.Hour)
	_, err = upcoming.GetPredictionSportsMatchList(t.Context(), &PredictionSportsMatchListRequest{EventType: 1, Status: 2, StartTime: start, EndTime: end})
	require.NoError(t, err, "GetPredictionSportsMatchList must send a window of upcoming matches")
	expBody := `{"endTime":` + strconv.FormatInt(end.UnixMilli(), 10) + `,"eventType":1,"startTime":` + strconv.FormatInt(start.UnixMilli(), 10) + `,"status":2}`
	assert.Equal(t, expBody, body, "GetPredictionSportsMatchList should send both bounds of a window of upcoming matches")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPredictionSportsMatchList(t.Context(), r)
	require.NoError(t, err, "GetPredictionSportsMatchList must not error")
	if mockTests {
		exp := &PredictionSportsMatchListResponse{
			List: []PredictionSportsMatch{
				{
					MatchID:   "match_001",
					EventType: 1,
					StageCode: "Groups",
					HomeTeam:  "Argentina",
					AwayTeam:  "Brazil",
					MatchTime: types.Time(time.Unix(1750000000, 0)),
					Status:    2,
					EventID:   "event_123",
				},
			},
			Total:     1,
			PageIndex: 1,
			PageSize:  20,
		}
		assert.Equal(t, exp, result, "GetPredictionSportsMatchList should decode every field")
		return
	}
	assert.NotZero(t, result.PageIndex, "GetPredictionSportsMatchList should return the page index")
}

func TestGetPredictionSportsTimelineStages(t *testing.T) {
	t.Parallel()
	_, err := e.GetPredictionSportsTimelineStages(t.Context(), 0)
	require.ErrorIs(t, err, errEventTypeRequired, "GetPredictionSportsTimelineStages must require an event type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPredictionSportsTimelineStages(t.Context(), 1)
	require.NoError(t, err, "GetPredictionSportsTimelineStages must not error")
	if mockTests {
		exp := &PredictionSportsTimelineStagesResponse{
			EventType: 1,
			Stages: []PredictionSportsStage{
				{
					StageCode:  "Groups",
					StageName:  "Group Stage",
					Status:     1,
					MatchCount: 48,
					StartTime:  types.Time(time.Unix(1748995200, 0)),
					EndTime:    types.Time(time.Unix(1751500800, 0)),
				},
				{
					StageCode:  "R32",
					StageName:  "Round of 32",
					Status:     2,
					MatchCount: 16,
					StartTime:  types.Time(time.Unix(1751587200, 0)),
					EndTime:    types.Time(time.Unix(1752192000, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetPredictionSportsTimelineStages should decode every field")
		return
	}
	assert.NotEmpty(t, result.Stages, "GetPredictionSportsTimelineStages should return stages")
}

func TestGetPredictionSportsGroupStageDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetPredictionSportsGroupStageDetail(t.Context(), 0, "A")
	require.ErrorIs(t, err, errEventTypeRequired, "GetPredictionSportsGroupStageDetail must require an event type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPredictionSportsGroupStageDetail(t.Context(), 1, "A")
	require.NoError(t, err, "GetPredictionSportsGroupStageDetail must not error")
	if mockTests {
		exp := &PredictionSportsGroupStageDetailResponse{
			Groups: []PredictionSportsGroup{
				{
					GroupName: "A",
					Standings: []PredictionSportsStanding{
						{
							Rank:           1,
							TeamName:       "Argentina",
							Played:         3,
							Won:            3,
							GoalsFor:       7,
							GoalsAgainst:   1,
							GoalDifference: 6,
							Points:         9,
						},
						{
							Rank:           2,
							TeamName:       "Brazil",
							Played:         3,
							Won:            1,
							Drawn:          1,
							Lost:           1,
							GoalsFor:       3,
							GoalsAgainst:   5,
							GoalDifference: -2,
							Points:         4,
						},
					},
					Matches: []PredictionSportsGroupMatch{
						{
							MatchID:   "match_001",
							HomeTeam:  "Argentina",
							AwayTeam:  "Brazil",
							MatchTime: types.Time(time.Unix(1750000000, 0)),
							Status:    3,
							HomeScore: 2,
							AwayScore: 1,
							EventID:   "event_123",
						},
						{
							MatchID:   "match_002",
							HomeTeam:  "Brazil",
							AwayTeam:  "Argentina",
							MatchTime: types.Time(time.Unix(1750600000, 0)),
							Status:    2,
							HomeScore: -1,
							AwayScore: -1,
							EventID:   "event_124",
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetPredictionSportsGroupStageDetail should decode every field")
		return
	}
	assert.NotEmpty(t, result.Groups, "GetPredictionSportsGroupStageDetail should return groups")
}
