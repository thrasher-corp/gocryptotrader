package bybit

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestWsProcessWalletPushData(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		run  func(*testing.T, *Exchange)
	}
	testCases := make([]testCase, 0, 5)
	testCases = append(testCases,
		testCase{
			name: "current unified payload",
			run: func(t *testing.T, ex *Exchange) {
				t.Helper()
				payload := []byte(`{"id":"wallet-update","topic":"wallet","creationTime":1787197420650,"data":[{"accountIMRate":"0.01","accountMMRate":"0.005","totalEquity":"120","totalWalletBalance":"100","totalMarginBalance":"110","totalAvailableBalance":"75","totalPerpUPL":"10","totalInitialMargin":"25","totalMaintenanceMargin":"5","coin":[{"coin":"USDT","equity":"120","usdValue":"120","walletBalance":"100","availableToWithdraw":"","availableToBorrow":"","borrowAmount":"0","accruedInterest":"0","totalOrderIM":"5","totalPositionIM":"20","totalPositionMM":"5","unrealisedPnl":"10","cumRealisedPnl":"2","bonus":"0","spotHedgingQty":"0"}],"accountType":"UNIFIED","accountLTV":""}]}`)
				var wallet WebsocketWallet
				decoder := json.NewDecoder(bytes.NewReader(payload))
				decoder.DisallowUnknownFields()
				require.NoError(t, decoder.Decode(&wallet), "current UNIFIED wallet payload must unmarshal without unknown fields")
				require.Len(t, wallet.Data, 1, "current UNIFIED wallet payload must contain one account")
				require.Len(t, wallet.Data[0].Coin, 1, "current UNIFIED wallet payload must contain one coin")
				assert.True(t, wallet.Data[0].Coin[0].AvailableToWithdraw.Decimal().IsZero(),
					"empty deprecated withdrawal availability should decode as zero")

				seed := accounts.NewSubAccount(asset.Spot, "")
				seed.Balances.Set(currency.USDT, accounts.Balance{Total: 100, Hold: 25, Free: 75, Borrowed: 10, AvailableWithoutBorrow: 80})
				require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{seed}, true),
					"Accounts.Save must seed fields absent from the websocket mapping")
				startedAt := time.Now()
				require.NoError(t, ex.wsProcessWalletPushData(t.Context(), payload),
					"wsProcessWalletPushData must process current UNIFIED wallet payload")
				require.Len(t, ex.Websocket.DataHandler.C, 1, "wallet handler must emit one message")
				message := <-ex.Websocket.DataHandler.C
				subAccounts, ok := message.Data.(accounts.SubAccounts)
				require.True(t, ok, "wallet handler must emit canonical subaccounts")
				require.Len(t, subAccounts, 1, "wallet handler must emit one subaccount")
				balance, ok := subAccounts[0].Balances[currency.USDT]
				require.True(t, ok, "wallet handler must emit the USDT balance")
				assert.Equal(t, 100.0, balance.Total, "wallet balance should be correct")
				assert.Equal(t, 75.0, balance.Free, "wallet update should preserve spendable funds from the REST snapshot")
				assert.Equal(t, 25.0, balance.Hold, "wallet update should preserve held funds from the REST snapshot")
				assert.Equal(t, 10.0, balance.Borrowed, "wallet update should preserve borrowed funds from the REST snapshot")
				assert.Equal(t, 80.0, balance.AvailableWithoutBorrow,
					"wallet update should preserve cached availability instead of using deprecated availableToWithdraw")
				creds, err := ex.GetCredentials(t.Context())
				require.NoError(t, err, "GetCredentials must not error")
				stored, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.USDT)
				require.NoError(t, err, "GetBalance must return the stored USDT balance")
				assert.Equal(t, balance, stored, "wallet update should store the balance it emits")
				assert.False(t, balance.UpdatedAt.Before(startedAt), "wallet update should use local arrival order")
			},
		},
		testCase{
			name: "mixed account types",
			run: func(t *testing.T, ex *Exchange) {
				t.Helper()
				payload := []byte(`{"topic":"wallet","data":[{"accountType":"unified","coin":[{"coin":"USDT","walletBalance":"100","availableToWithdraw":"5"}]},{"accountType":"contract","coin":[{"coin":"BTC","walletBalance":"2","availableToWithdraw":"1.5"}]}]}`)
				require.NoError(t, ex.wsProcessWalletPushData(t.Context(), payload),
					"wsProcessWalletPushData must process a wallet payload with several account types")
				require.Len(t, ex.Websocket.DataHandler.C, 1, "wallet handler must emit one message")
				message := <-ex.Websocket.DataHandler.C
				subAccounts, ok := message.Data.(accounts.SubAccounts)
				require.True(t, ok, "wallet handler must emit canonical subaccounts")
				require.Len(t, subAccounts, 2, "wallet handler must emit one subaccount per wallet")
				assert.Equal(t, asset.Spot, subAccounts[0].AssetType, "UNIFIED wallet should be emitted under spot")
				assert.Equal(t, 100.0, subAccounts[0].Balances[currency.USDT].Total, "UNIFIED wallet balance should be emitted")
				assert.Zero(t, subAccounts[0].Balances[currency.USDT].AvailableWithoutBorrow, "UNIFIED wallet should not apply availableToWithdraw")
				assert.Equal(t, asset.CoinMarginedFutures, subAccounts[1].AssetType, "CONTRACT wallet should be emitted under coin margined futures")
				assert.Equal(t, 2.0, subAccounts[1].Balances[currency.BTC].Total, "CONTRACT wallet balance should be emitted")
				spot, err := ex.GetCachedCurrencyBalances(t.Context(), asset.Spot)
				require.NoError(t, err, "GetCachedCurrencyBalances must not error for spot")
				assert.Equal(t, 100.0, spot[currency.USDT].Total, "UNIFIED wallet balance should be stored under spot")
				assert.NotContains(t, spot, currency.BTC, "CONTRACT wallet coins should not be stored under spot")
			},
		},
		testCase{
			name: "classic contract caches",
			run: func(t *testing.T, ex *Exchange) {
				t.Helper()
				ex.account.accountType = accountTypeNormal
				derivatives := []asset.Item{asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.Options}
				seeds := make(accounts.SubAccounts, 0, 1+len(derivatives))
				seeds = append(seeds, accounts.NewSubAccount(asset.Spot, ""))
				for _, a := range derivatives {
					seeds = append(seeds, accounts.NewSubAccount(a, ""))
				}
				for _, seed := range seeds {
					seed.Balances.Set(currency.USDT, accounts.Balance{Total: 90, AvailableWithoutBorrow: 80})
				}
				require.NoError(t, ex.Accounts.Save(t.Context(), seeds, true), "Accounts.Save must seed the REST wallet caches")

				payload := []byte(`{"topic":"wallet","data":[{"accountType":"CONTRACT","coin":[{"coin":"USDT","walletBalance":"100","availableToWithdraw":"65"}]}]}`)
				require.NoError(t, ex.wsProcessWalletPushData(t.Context(), payload), "wsProcessWalletPushData must process a classic CONTRACT wallet")
				require.Len(t, ex.Websocket.DataHandler.C, 1, "wallet handler must emit one message")
				message := <-ex.Websocket.DataHandler.C
				subAccounts, ok := message.Data.(accounts.SubAccounts)
				require.True(t, ok, "wallet handler must emit canonical subaccounts")
				require.Len(t, subAccounts, len(derivatives), "wallet handler must emit every REST-backed CONTRACT asset")
				creds, err := ex.GetCredentials(t.Context())
				require.NoError(t, err, "GetCredentials must not error")
				for i, a := range derivatives {
					assert.Equal(t, a, subAccounts[i].AssetType, "CONTRACT wallet should be emitted under its REST-backed asset")
					balance, err := ex.Accounts.GetBalance("", creds, a, currency.USDT)
					require.NoError(t, err, "GetBalance must return the updated CONTRACT wallet")
					assert.Equal(t, 100.0, balance.Total, "CONTRACT wallet total should be updated")
					assert.Equal(t, 65.0, balance.AvailableWithoutBorrow, "CONTRACT wallet availability should be updated")
					assert.Equal(t, balance, subAccounts[i].Balances[currency.USDT], "stored balance should match the emitted balance")
				}
				spot, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.USDT)
				require.NoError(t, err, "GetBalance must return the untouched SPOT wallet")
				assert.Equal(t, 90.0, spot.Total, "CONTRACT push should not change SPOT total")
				assert.Equal(t, 80.0, spot.AvailableWithoutBorrow, "CONTRACT push should not change SPOT availability")
			},
		},
	)

	for _, tc := range []struct {
		accountType        string
		updated, untouched asset.Item
	}{
		{accountType: "SPOT", updated: asset.Spot, untouched: asset.CoinMarginedFutures},
		{accountType: "CONTRACT", updated: asset.CoinMarginedFutures, untouched: asset.Spot},
	} {
		testCases = append(testCases, testCase{
			name: "normal account availability/" + tc.accountType,
			run: func(t *testing.T, ex *Exchange) {
				t.Helper()
				seeds := accounts.SubAccounts{accounts.NewSubAccount(asset.Spot, ""), accounts.NewSubAccount(asset.CoinMarginedFutures, "")}
				for _, seed := range seeds {
					seed.Balances.Set(currency.USDT, accounts.Balance{Total: 90, AvailableWithoutBorrow: 80})
				}
				require.NoError(t, ex.Accounts.Save(t.Context(), seeds, true),
					"Accounts.Save must seed the previous balances")
				payload := []byte(fmt.Sprintf(`{"topic":"wallet","data":[{"coin":[{"coin":"USDT","walletBalance":"100","availableToWithdraw":"65"}],"accountType":%q}]}`, tc.accountType))
				require.NoError(t, ex.wsProcessWalletPushData(t.Context(), payload),
					"wsProcessWalletPushData must process a normal account wallet payload")
				require.Len(t, ex.Websocket.DataHandler.C, 1, "wallet handler must emit one message")
				message := <-ex.Websocket.DataHandler.C
				subAccounts, ok := message.Data.(accounts.SubAccounts)
				require.True(t, ok, "wallet handler must emit canonical subaccounts")
				require.Len(t, subAccounts, 1, "wallet handler must emit one subaccount")
				assert.Equal(t, tc.updated, subAccounts[0].AssetType, "wallet handler should use the asset REST stores this wallet under")
				balance, ok := subAccounts[0].Balances[currency.USDT]
				require.True(t, ok, "wallet handler must emit the USDT balance")
				assert.Equal(t, 100.0, balance.Total, "normal wallet update should apply walletBalance")
				assert.Equal(t, 65.0, balance.AvailableWithoutBorrow,
					"normal wallet update should apply availableToWithdraw")
				creds, err := ex.GetCredentials(t.Context())
				require.NoError(t, err, "GetCredentials must not error")
				stored, err := ex.Accounts.GetBalance("", creds, tc.updated, currency.USDT)
				require.NoError(t, err, "GetBalance must return the pushed wallet's balance")
				assert.Equal(t, balance, stored, "wallet update should store the balance it emits")
				untouched, err := ex.Accounts.GetBalance("", creds, tc.untouched, currency.USDT)
				require.NoError(t, err, "GetBalance must return the other wallet's balance")
				assert.Equal(t, 90.0, untouched.Total, "wallet update should not change another wallet's balance")
				assert.Equal(t, 80.0, untouched.AvailableWithoutBorrow, "wallet update should not change another wallet's availability")
			},
		})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := testInstance()
			ex.API.AuthenticatedSupport = true
			ex.API.AuthenticatedWebsocketSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "test", Secret: "test"})
			tc.run(t, ex)
		})
	}
}

func TestWSHandleTradeData(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input []byte
		match string
		err   error
	}{
		{input: []byte(`{"reqId":"12345"}`), match: "12345", err: nil},
		{input: []byte(`{"op":"auth"}`), match: "auth", err: nil},
		{input: []byte(`{"op":"pong"}`), err: nil},
		{input: []byte(`{"op":"pewpewpew"}`), err: errUnhandledStreamData},
	} {
		conn := &FixtureConnection{match: websocket.NewMatch()}
		var ch <-chan []byte
		if tc.match != "" {
			var err error
			ch, err = conn.match.Set(tc.match, 1)
			require.NoError(t, err, "match.Set must not error")
		}
		err := e.wsHandleTradeData(conn, tc.input)
		if tc.err != nil {
			require.ErrorIs(t, err, tc.err)
			continue
		}
		require.NoError(t, err)
		if tc.match != "" {
			require.Len(t, ch, 1, "must receive 1 message from channel")
			require.Equal(t, tc.input, <-ch, "must be correct")
		}
	}
}
