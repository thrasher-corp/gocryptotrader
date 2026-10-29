package coinbase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	testNewPortfolioName     = "GCT test portfolio"
	testRenamedPortfolioName = "GCT renamed portfolio"
	// testNewPortfolioUUID is the portfolio the mock fixtures create, rename and delete, the docs' example UUID
	testNewPortfolioUUID = "8bfc20d7-f7c6-4422-bf07-8243ca4169fe"
	// testSandboxDefaultPortfolioUUID and testSandboxINTXPortfolioUUID are the sandbox's portfolios
	testSandboxDefaultPortfolioUUID = "1fb3c56a-489f-418c-8b5c-ce04268be5a6"
	testSandboxINTXPortfolioUUID    = "018c4b12-9e43-7ae1-9577-997e7df8c14d"
	// testMovePortfolioFundsAmount is how much of testMovePortfolioFundsCurrency the live test moves out and back
	testMovePortfolioFundsAmount = 1
)

// testMovePortfolioFundsCurrency is the currency the portfolio funds tests move
var testMovePortfolioFundsCurrency = currency.USDC

// testPortfoliosDefaultUUID returns the account's default portfolio, the sandbox's for mock tests
func testPortfoliosDefaultUUID(t *testing.T) string {
	t.Helper()
	if mockTests {
		return testSandboxDefaultPortfolioUUID
	}
	portfolios, err := e.ListPortfolios(t.Context(), PortfolioTypeDefault)
	require.NoError(t, err, "ListPortfolios must not error")
	require.NotEmpty(t, portfolios, "ListPortfolios must return the default portfolio")
	return portfolios[0].UUID
}

func TestCreatePortfolio(t *testing.T) {
	t.Parallel()
	_, err := e.CreatePortfolio(t.Context(), "")
	require.ErrorIs(t, err, errPortfolioNameRequired, "CreatePortfolio must require a name")

	skipLiveUnlessCanManipulate(t)
	resp, err := e.CreatePortfolio(t.Context(), testNewPortfolioName)
	require.NoError(t, err, "CreatePortfolio must not error")
	if mockTests {
		exp := &Portfolio{
			Name: "GCT test portfolio",
			UUID: "8bfc20d7-f7c6-4422-bf07-8243ca4169fe",
			Type: "CONSUMER",
		}
		assert.Equal(t, exp, resp, "CreatePortfolio should decode every field")
	} else {
		assert.Equal(t, testNewPortfolioName, resp.Name, "CreatePortfolio should name the portfolio")
		assert.NoError(t, e.DeletePortfolio(t.Context(), resp.UUID), "DeletePortfolio should delete the created portfolio")
	}

	_, err = newResponseServerExchange(t, `null`).CreatePortfolio(t.Context(), testNewPortfolioName)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreatePortfolio should error for a null response")
	_, err = newResponseServerExchange(t, `{"portfolio":null}`).CreatePortfolio(t.Context(), testNewPortfolioName)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreatePortfolio should error for a null portfolio")
	_, err = newUnavailableServerExchange(t).CreatePortfolio(t.Context(), testNewPortfolioName)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreatePortfolio should return the request failure")
}

func TestDeletePortfolio(t *testing.T) {
	t.Parallel()
	err := e.DeletePortfolio(t.Context(), "")
	require.ErrorIs(t, err, errPortfolioIDRequired, "DeletePortfolio must require a portfolio UUID")

	skipLiveUnlessCanManipulate(t)
	portfolioUUID := testNewPortfolioUUID
	if !mockTests {
		portfolio, err := e.CreatePortfolio(t.Context(), testNewPortfolioName)
		require.NoError(t, err, "CreatePortfolio must create a portfolio to delete")
		portfolioUUID = portfolio.UUID
	}
	require.NoError(t, e.DeletePortfolio(t.Context(), portfolioUUID), "DeletePortfolio must not error")

	err = newUnavailableServerExchange(t).DeletePortfolio(t.Context(), testNewPortfolioUUID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "DeletePortfolio should return the request failure")
}

func TestEditPortfolio(t *testing.T) {
	t.Parallel()
	_, err := e.EditPortfolio(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "EditPortfolio must reject a nil request")
	_, err = e.EditPortfolio(t.Context(), &EditPortfolioRequest{Name: testRenamedPortfolioName})
	require.ErrorIs(t, err, errPortfolioIDRequired, "EditPortfolio must require a portfolio UUID")
	_, err = e.EditPortfolio(t.Context(), &EditPortfolioRequest{PortfolioUUID: testNewPortfolioUUID})
	require.ErrorIs(t, err, errPortfolioNameRequired, "EditPortfolio must require a name")

	skipLiveUnlessCanManipulate(t)
	portfolioUUID := testNewPortfolioUUID
	if !mockTests {
		portfolio, err := e.CreatePortfolio(t.Context(), testNewPortfolioName)
		require.NoError(t, err, "CreatePortfolio must create a portfolio to rename")
		portfolioUUID = portfolio.UUID
		t.Cleanup(func() {
			// The test's context ends before its cleanups run
			assert.NoError(t, e.DeletePortfolio(context.WithoutCancel(t.Context()), portfolioUUID), "DeletePortfolio should delete the renamed portfolio")
		})
	}
	resp, err := e.EditPortfolio(t.Context(), &EditPortfolioRequest{PortfolioUUID: portfolioUUID, Name: testRenamedPortfolioName})
	require.NoError(t, err, "EditPortfolio must not error")
	if mockTests {
		exp := &Portfolio{
			Name: "GCT renamed portfolio",
			UUID: "8bfc20d7-f7c6-4422-bf07-8243ca4169fe",
			Type: "CONSUMER",
		}
		assert.Equal(t, exp, resp, "EditPortfolio should decode every field")
	} else {
		assert.Equal(t, testRenamedPortfolioName, resp.Name, "EditPortfolio should rename the portfolio")
	}

	_, err = newResponseServerExchange(t, `null`).EditPortfolio(t.Context(), &EditPortfolioRequest{PortfolioUUID: testNewPortfolioUUID, Name: testRenamedPortfolioName})
	assert.ErrorIs(t, err, common.ErrNoResponse, "EditPortfolio should error for a null response")
	_, err = newResponseServerExchange(t, `{"portfolio":null}`).EditPortfolio(t.Context(), &EditPortfolioRequest{PortfolioUUID: testNewPortfolioUUID, Name: testRenamedPortfolioName})
	assert.ErrorIs(t, err, common.ErrNoResponse, "EditPortfolio should error for a null portfolio")
	_, err = newUnavailableServerExchange(t).EditPortfolio(t.Context(), &EditPortfolioRequest{PortfolioUUID: testNewPortfolioUUID, Name: testRenamedPortfolioName})
	assert.ErrorIs(t, err, request.ErrBadStatus, "EditPortfolio should return the request failure")
}

func TestGetPortfolioBreakdown(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioBreakdown(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPortfolioBreakdown must reject a nil request")
	_, err = e.GetPortfolioBreakdown(t.Context(), &PortfolioBreakdownRequest{Currency: currency.USD})
	require.ErrorIs(t, err, errPortfolioIDRequired, "GetPortfolioBreakdown must require a portfolio UUID")

	skipLiveWithoutCredentials(t)
	resp, err := e.GetPortfolioBreakdown(t.Context(), &PortfolioBreakdownRequest{PortfolioUUID: testPortfoliosDefaultUUID(t), Currency: currency.USD})
	require.NoError(t, err, "GetPortfolioBreakdown must not error")
	if mockTests {
		exp := &PortfolioBreakdown{
			Portfolio: Portfolio{
				Name: "DEFAULT",
				UUID: "1fb3c56a-489f-418c-8b5c-ce04268be5a6",
				Type: "DEFAULT",
			},
			PortfolioBalances: PortfolioBalances{
				TotalBalance: Amount{
					Value:    42082.31,
					Currency: currency.USD,
				},
				TotalFuturesBalance: Amount{
					Value:    250,
					Currency: currency.USD,
				},
				TotalCashEquivalentBalance: Amount{
					Value:    200,
					Currency: currency.USD,
				},
				TotalCryptoBalance: Amount{
					Value:    41032.31,
					Currency: currency.USD,
				},
				FuturesUnrealisedPNL: Amount{
					Value:    12.2,
					Currency: currency.USD,
				},
				PerpetualUnrealisedPNL: Amount{
					Currency: currency.USD,
				},
				TotalEquitiesBalance: Amount{
					Value:    600,
					Currency: currency.USD,
				},
			},
			SpotPositions: []PortfolioSpotPosition{
				{
					Asset:                currency.BTC,
					AccountUUID:          "31e698c7-edd1-43b7-a0db-5c87fc15ea52",
					TotalBalanceFiat:     41032.31,
					TotalBalanceCrypto:   0.5,
					AvailableToTradeFiat: 41032.31,
					Allocation:           0.9750486,
					CostBasis: Amount{
						Value:    30000,
						Currency: currency.USD,
					},
					AssetImageURL: "https://dynamic-assets.coinbase.com/e785e0181f1a23a30d9476038d9be91e9f6c63959b538eabbc51a1abc8898940383291eede695c3b8dfaa1829a9b57f5a2d0a16b0523580346c6b8fab67af14b/asset_icons/b57ac673f06a4b0338a596817eb0a50ce16e2059f327dc117744449a47915cb2.png",
					AverageEntryPrice: Amount{
						Value:    60000,
						Currency: currency.USD,
					},
					AssetUUID:                 "5b71fc48-3dd3-540c-809b-f8c94d0e68b5",
					AvailableToTradeCrypto:    0.5,
					UnrealisedPNL:             11032.31,
					AvailableToTransferFiat:   41032.31,
					AvailableToTransferCrypto: 0.5,
					AssetColour:               "#F7931A",
					AccountType:               "ACCOUNT_TYPE_WALLET",
					AvailableToSendFiat:       41032.31,
					AvailableToSendCrypto:     0.5,
				},
				{
					Asset:                currency.USD,
					AccountUUID:          "81b5dca0-5775-4a21-b872-fc850046c99b",
					TotalBalanceFiat:     100,
					TotalBalanceCrypto:   100,
					AvailableToTradeFiat: 100,
					Allocation:           0.0023763,
					CostBasis: Amount{
						Value:    100,
						Currency: currency.USD,
					},
					IsCash: true,
					AverageEntryPrice: Amount{
						Value:    1,
						Currency: currency.USD,
					},
					AvailableToTradeCrypto:    100,
					AvailableToTransferFiat:   100,
					AvailableToTransferCrypto: 100,
					AccountType:               "ACCOUNT_TYPE_FIAT",
					AvailableToSendFiat:       100,
					AvailableToSendCrypto:     100,
				},
			},
			PerpetualPositions: []PortfolioPerpetualPosition{},
			FuturesPositions: []PortfolioFuturesPosition{
				{
					ProductID:         "BIT-30OCT26-CDE",
					ContractSize:      0.01,
					Side:              "FUTURES_POSITION_SIDE_LONG",
					Amount:            2,
					AverageEntryPrice: 81500,
					CurrentPrice:      82110,
					UnrealisedPNL:     12.2,
					Expiry:            time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC),
					UnderlyingAsset:   currency.BTC,
					ProductName:       "BTC 30 OCT 26",
					Venue:             "cde",
					NotionalValue:     1642.2,
					AssetColour:       "#F7931A",
					LastTradedAt:      time.Date(2026, 10, 18, 21, 14, 3, 0, time.UTC),
					PriceIncrement:    5,
				},
			},
			PredictionMarketsPositions: []PortfolioUnifiedPosition{
				{
					ProductType: "PRODUCT_TYPE_PREDICTION_MARKETS",
					PredictionMarkets: &PortfolioPredictionMarketPosition{
						Side:         "PREDICTION_MARKETS_POSITION_SIDE_LONG",
						LastTradedAt: time.Date(2026, 10, 17, 15, 30, 0, 0, time.UTC),
						AverageEntryPrice: Amount{
							Value:    0.42,
							Currency: currency.USD,
						},
						CurrentPrice: Amount{
							Value:    0.55,
							Currency: currency.USD,
						},
						UnrealisedPNL: Amount{
							Value:    1.3,
							Currency: currency.USD,
						},
						NotionalValue: Amount{
							Value:    5.5,
							Currency: currency.USD,
						},
						CostBasis: Amount{
							Value:    4.2,
							Currency: currency.USD,
						},
						ContractsOwned: 10,
					},
				},
			},
			EquityPositions: []PortfolioEquityPosition{
				{
					AccountUUID: "a8c3b1d2-6f4e-4c5b-9a7d-2e1f0b3c4d5e",
					TotalBalanceFiat: Amount{
						Value:    600,
						Currency: currency.USD,
					},
					TotalBalanceEquity: Amount{
						Value:    2,
						Currency: currency.NewCode("COIN"),
					},
					AvailableToTradeEquity: Amount{
						Value:    2,
						Currency: currency.NewCode("COIN"),
					},
					AvailableToTradeFiat: Amount{
						Value:    600,
						Currency: currency.USD,
					},
					AccountType: "ACCOUNT_TYPE_CCM_EQUITY",
					Allocation:  0.0142578,
					CostBasis: Amount{
						Value:    500,
						Currency: currency.USD,
					},
					AverageEntryPrice: Amount{
						Value:    250,
						Currency: currency.USD,
					},
					UnrealisedPNL: Amount{
						Value:    100,
						Currency: currency.USD,
					},
				},
			},
		}
		assert.Equal(t, exp, resp, "GetPortfolioBreakdown should decode every field")
	} else {
		assert.Equal(t, PortfolioTypeDefault, resp.Portfolio.Type, "GetPortfolioBreakdown should break the default portfolio down")
	}

	_, err = newResponseServerExchange(t, `null`).GetPortfolioBreakdown(t.Context(), &PortfolioBreakdownRequest{PortfolioUUID: testSandboxDefaultPortfolioUUID})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPortfolioBreakdown should error for a null response")
	_, err = newResponseServerExchange(t, `{"breakdown":null}`).GetPortfolioBreakdown(t.Context(), &PortfolioBreakdownRequest{PortfolioUUID: testSandboxDefaultPortfolioUUID})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPortfolioBreakdown should error for a null breakdown")
	_, err = newUnavailableServerExchange(t).GetPortfolioBreakdown(t.Context(), &PortfolioBreakdownRequest{PortfolioUUID: testSandboxDefaultPortfolioUUID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPortfolioBreakdown should return the request failure")
}

func TestListPortfolios(t *testing.T) {
	t.Parallel()
	_, err := e.ListPortfolios(t.Context(), "UNDEFINED")
	require.ErrorIs(t, err, errInvalidPortfolioType, "ListPortfolios must refuse an unknown portfolio type")

	skipLiveWithoutCredentials(t)
	if mockTests {
		for portfolioType, exp := range map[string][]Portfolio{
			"": {
				{
					Name: "DEFAULT",
					UUID: "1fb3c56a-489f-418c-8b5c-ce04268be5a6",
					Type: "DEFAULT",
				},
				{
					Name: "INTX Portfolio",
					UUID: "018c4b12-9e43-7ae1-9577-997e7df8c14d",
					Type: "INTX",
				},
			},
			PortfolioTypeDefault: {
				{
					Name: "DEFAULT",
					UUID: "1fb3c56a-489f-418c-8b5c-ce04268be5a6",
					Type: "DEFAULT",
				},
			},
			PortfolioTypeConsumer: {},
			PortfolioTypeINTX: {
				{
					Name: "INTX Portfolio",
					UUID: "018c4b12-9e43-7ae1-9577-997e7df8c14d",
					Type: "INTX",
				},
			},
		} {
			resp, err := e.ListPortfolios(t.Context(), portfolioType)
			require.NoErrorf(t, err, "ListPortfolios must not error for %q", portfolioType)
			assert.Equalf(t, exp, resp, "ListPortfolios should decode every field for %q", portfolioType)
		}
	} else {
		resp, err := e.ListPortfolios(t.Context(), "")
		require.NoError(t, err, "ListPortfolios must not error")
		assert.NotEmpty(t, resp, "ListPortfolios should return the account's portfolios")
	}

	_, err = newUnavailableServerExchange(t).ListPortfolios(t.Context(), "")
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListPortfolios should return the request failure")
}

func TestMovePortfolioFunds(t *testing.T) {
	t.Parallel()
	_, err := e.MovePortfolioFunds(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "MovePortfolioFunds must reject a nil request")
	req := &MovePortfolioFundsRequest{TargetPortfolioUUID: testSandboxINTXPortfolioUUID, Currency: testMovePortfolioFundsCurrency, Amount: testMovePortfolioFundsAmount}
	_, err = e.MovePortfolioFunds(t.Context(), req)
	require.ErrorIs(t, err, errPortfolioIDRequired, "MovePortfolioFunds must require a source portfolio")
	req = &MovePortfolioFundsRequest{SourcePortfolioUUID: testSandboxDefaultPortfolioUUID, Currency: testMovePortfolioFundsCurrency, Amount: testMovePortfolioFundsAmount}
	_, err = e.MovePortfolioFunds(t.Context(), req)
	require.ErrorIs(t, err, errPortfolioIDRequired, "MovePortfolioFunds must require a target portfolio")
	req = &MovePortfolioFundsRequest{SourcePortfolioUUID: testSandboxDefaultPortfolioUUID, TargetPortfolioUUID: testSandboxINTXPortfolioUUID, Amount: testMovePortfolioFundsAmount}
	_, err = e.MovePortfolioFunds(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "MovePortfolioFunds must require a currency")
	for _, amount := range []float64{0, -1} {
		req = &MovePortfolioFundsRequest{SourcePortfolioUUID: testSandboxDefaultPortfolioUUID, TargetPortfolioUUID: testSandboxINTXPortfolioUUID, Currency: testMovePortfolioFundsCurrency, Amount: amount}
		_, err = e.MovePortfolioFunds(t.Context(), req)
		require.ErrorIsf(t, err, errInvalidAmount, "MovePortfolioFunds must refuse an amount of %v", amount)
	}

	skipLiveUnlessCanManipulate(t)
	if mockTests {
		resp, err := e.MovePortfolioFunds(t.Context(), &MovePortfolioFundsRequest{SourcePortfolioUUID: testSandboxDefaultPortfolioUUID, TargetPortfolioUUID: testSandboxINTXPortfolioUUID, Currency: testMovePortfolioFundsCurrency, Amount: 100})
		require.NoError(t, err, "MovePortfolioFunds must not error")
		exp := &MovePortfolioFundsResponse{
			SourcePortfolioUUID: "1fb3c56a-489f-418c-8b5c-ce04268be5a6",
			TargetPortfolioUUID: "018c4b12-9e43-7ae1-9577-997e7df8c14d",
		}
		assert.Equal(t, exp, resp, "MovePortfolioFunds should decode every field")
	} else {
		portfolio, err := e.CreatePortfolio(t.Context(), testNewPortfolioName)
		require.NoError(t, err, "CreatePortfolio must create a portfolio to move funds to")
		t.Cleanup(func() {
			assert.NoError(t, e.DeletePortfolio(context.WithoutCancel(t.Context()), portfolio.UUID), "DeletePortfolio should delete the emptied portfolio")
		})
		defaultUUID := testPortfoliosDefaultUUID(t)
		resp, err := e.MovePortfolioFunds(t.Context(), &MovePortfolioFundsRequest{SourcePortfolioUUID: defaultUUID, TargetPortfolioUUID: portfolio.UUID, Currency: testMovePortfolioFundsCurrency, Amount: testMovePortfolioFundsAmount})
		require.NoError(t, err, "MovePortfolioFunds must not error")
		assert.Equal(t, portfolio.UUID, resp.TargetPortfolioUUID, "MovePortfolioFunds should move the funds to the target portfolio")
		_, err = e.MovePortfolioFunds(t.Context(), &MovePortfolioFundsRequest{SourcePortfolioUUID: portfolio.UUID, TargetPortfolioUUID: defaultUUID, Currency: testMovePortfolioFundsCurrency, Amount: testMovePortfolioFundsAmount})
		require.NoError(t, err, "MovePortfolioFunds must move the funds back")
	}

	req = &MovePortfolioFundsRequest{SourcePortfolioUUID: testSandboxDefaultPortfolioUUID, TargetPortfolioUUID: testSandboxINTXPortfolioUUID, Currency: testMovePortfolioFundsCurrency, Amount: testMovePortfolioFundsAmount}
	_, err = newResponseServerExchange(t, `null`).MovePortfolioFunds(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "MovePortfolioFunds should error for a null response")
	_, err = newUnavailableServerExchange(t).MovePortfolioFunds(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "MovePortfolioFunds should return the request failure")
}
