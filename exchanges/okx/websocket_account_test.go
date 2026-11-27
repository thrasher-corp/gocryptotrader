package okx

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// enableFillsFeed turns the fills feed on, as a config with fillsFeed set does
func enableFillsFeed(ex *Exchange) {
	ex.SetFillsFeedStatus(true)
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
}

// newAccountStreamExchange returns an exchange with credentials, whose account pushes save to its balance store
func newAccountStreamExchange(t *testing.T, creds *accounts.Credentials) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(creds)
	return ex
}

// accountStreamConnection is a connection whose login a test records
type accountStreamConnection struct {
	websocket.Connection
}

// newLoggedInAccountStream returns newAccountStreamExchange's exchange and a
// connection logged in with its credentials, recorded as a login records them
func newLoggedInAccountStream(t *testing.T, creds *accounts.Credentials) (*Exchange, websocket.Connection) {
	t.Helper()
	ex := newAccountStreamExchange(t, creds)
	login, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	conn := new(accountStreamConnection)
	ex.recordWebsocketLogin(conn, login)
	return ex, conn
}

// accountEventSnapshot is the event type of an account channel snapshot page
const accountEventSnapshot = "snapshot"

// Page fields of account channel snapshot pushes
const (
	onlyAccountPage       = `"curPage":1,"lastPage":true,`
	firstAccountPage      = `"curPage":1,"lastPage":false,`
	secondLastAccountPage = `"curPage":2,"lastPage":true,`
)

// accountPush returns an account channel push for UID 17 of an event type and
// page fields, whose account row lists details
func accountPush(eventType, page string, details ...string) []byte {
	return []byte(`{"arg":{"channel":"account","uid":"17"},"eventType":"` + eventType + `",` + page +
		`"data":[{"uTime":"1795400001000","totalEq":"160","details":[` + strings.Join(details, ",") + `]}]}`)
}

func TestWebsocketProcessBalanceAndPosition(t *testing.T) {
	t.Parallel()

	t.Run("relays the push without saving its cash", func(t *testing.T) {
		t.Parallel()
		creds := &accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret", ClientID: "mock-passphrase"}
		ex := newAccountStreamExchange(t, creds)
		handleFixture(t, ex, "balance_and_position")
		pushed := types.Time(time.UnixMilli(1597026383085))
		contract := currency.NewPairWithDelimiter("BTC", "USD-261127", currency.DashDelimiter)
		exp := []any{
			&WebsocketBalanceAndPositionPush{
				Argument: SubscriptionArgument{Channel: channelBalanceAndPosition, UserID: "77982378738415879"},
				Data: []WebsocketBalanceAndPosition{{
					PushTime:  pushed,
					EventType: "snapshot",
					Balances:  []WebsocketBalance{{Currency: currency.BTC, CashBalance: 1, UpdateTime: pushed}},
					Positions: []WebsocketBalancePosition{{
						PositionID: "1111111111", TradeID: "2", InstrumentID: contract, InstrumentType: instTypeFutures, MarginMode: TradeModeCross,
						AveragePrice: 3320, Currency: currency.BTC, PositionSide: positionSideLong, Position: 10, BaseBalance: 0.15, QuoteBalance: 9000,
						PositionCurrency: currency.BTC, NonSettlementAveragePrice: 61400.5, SettledProfitAndLoss: 0.0004, UpdateTime: pushed,
					}},
					Trades: []WebsocketBalanceAndPositionTrade{{InstrumentID: contract, TradeID: "2"}},
				}},
			},
		}
		assert.Equal(t, exp, relayed(ex), "the push should be relayed as it arrives")
		_, err := ex.Accounts.SubAccounts(creds, asset.Spot)
		assert.ErrorIs(t, err, accounts.ErrNoSubAccounts, "the push's cash should not be saved")
	})

	t.Run("leaves a stored balance as it was", func(t *testing.T) {
		t.Parallel()
		creds := &accounts.Credentials{Key: "fixture-key", Secret: "fixture-secret", ClientID: "fixture-passphrase"}
		ex := newAccountStreamExchange(t, creds)
		held := accounts.NewSubAccount(asset.Spot, "")
		held.Balances.Set(currency.USDT, accounts.Balance{Total: 100, Free: 60, Hold: 40, Borrowed: 5, UpdatedAt: time.UnixMilli(1795400000000)})
		require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{held}, true), "Save must store the earlier balance")
		frame := []byte(`{"arg":{"channel":"balance_and_position","uid":"17"},"data":[{"pTime":"1795400001000","eventType":"transferred",` +
			`"balData":[{"ccy":"USDT","cashBal":"101","uTime":"1795400001000"},{"ccy":"BTC","cashBal":"0.5","uTime":"1795400001000"}],"posData":[],"trades":[]}]}`)
		require.NoError(t, ex.wsHandleData(t.Context(), nil, frame), "wsHandleData must not error")
		push := new(WebsocketBalanceAndPositionPush)
		require.NoError(t, json.Unmarshal(frame, push), "the push must decode")
		assert.Equal(t, []any{push}, relayed(ex), "a cash update should be relayed as it arrives")
		exp := accounts.SubAccounts{{
			AssetType: asset.Spot,
			Balances: accounts.CurrencyBalances{
				currency.USDT: {Currency: currency.USDT, Total: 100, Free: 60, Hold: 40, Borrowed: 5, UpdatedAt: time.UnixMilli(1795400000000)},
			},
		}}
		stored, err := ex.Accounts.SubAccounts(creds, asset.Spot)
		require.NoError(t, err, "SubAccounts must not error")
		assert.Equal(t, exp, stored, "a cash update should leave the stored balances as they were")
	})

	t.Run("relays a push carrying balances, positions or trades", func(t *testing.T) {
		t.Parallel()
		const cash = `[{"ccy":"USDT","cashBal":"100","uTime":"1795400000000"}]`
		const positions = `[{"posId":"p8","instType":"SWAP","instId":"BTC-USDT-SWAP","mgnMode":"isolated","posSide":"long","pos":"10","ccy":"USDT","uTime":"1795400000000"}]`
		const trades = `[{"instId":"BTC-USDT-SWAP","tradeId":"7"}]`
		row := func(eventType, balances, positions, trades string) string {
			return `{"pTime":"1795400000000","eventType":"` + eventType + `","balData":` + balances + `,"posData":` + positions + `,"trades":` + trades + `}`
		}
		for _, tc := range []struct {
			name, data string
			expPush    bool
		}{
			{name: "positions alone", data: row("set_leverage", "[]", positions, "[]"), expPush: true},
			{name: "trades alone", data: row("filled", "[]", "[]", trades), expPush: true},
			{name: "balances alone", data: row("transferred", cash, "[]", "[]"), expPush: true},
			{name: "balances and positions", data: row("filled", cash, positions, trades), expPush: true},
			{name: "a second row carrying balances", data: row("snapshot", "[]", "[]", "[]") + "," + row("transferred", cash, "[]", "[]"), expPush: true},
			{name: "nothing", data: row("snapshot", "[]", "[]", "[]")},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				creds := &accounts.Credentials{Key: "fixture-key", Secret: "fixture-secret", ClientID: "fixture-passphrase"}
				ex := newAccountStreamExchange(t, creds)
				frame := []byte(`{"arg":{"channel":"balance_and_position","uid":"4"},"data":[` + tc.data + `]}`)
				require.NoError(t, ex.wsHandleData(t.Context(), nil, frame), "wsHandleData must not error")
				var exp []any
				if tc.expPush {
					push := new(WebsocketBalanceAndPositionPush)
					require.NoError(t, json.Unmarshal(frame, push), "the push must decode")
					exp = append(exp, push)
				}
				assert.Equal(t, exp, relayed(ex), "the push should be relayed only when it carries balances, positions or trades")
				_, err := ex.Accounts.SubAccounts(creds, asset.Spot)
				assert.ErrorIs(t, err, accounts.ErrNoSubAccounts, "the push should leave the balance store untouched")
			})
		}
	})

	t.Run("needs no credentials", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, frame string
		}{
			{
				name:  "balances",
				frame: `{"arg":{"channel":"balance_and_position","uid":"77982378738415880"},"data":[{"pTime":"1597026383085","eventType":"snapshot","balData":[{"ccy":"BTC","cashBal":"1","uTime":"1597026383085"}],"posData":[]}]}`,
			},
			{
				name: "balances and positions",
				frame: `{"arg":{"channel":"balance_and_position","uid":"4"},"data":[{"pTime":"1795400000000","eventType":"filled","balData":[{"ccy":"USDT","cashBal":"100","uTime":"1795400000000"}],` +
					`"posData":[{"posId":"p8","instType":"SWAP","instId":"BTC-USDT-SWAP","posSide":"long","pos":"10","uTime":"1795400000000"}]}]}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				ex.API.AuthenticatedSupport = false
				ex.API.AuthenticatedWebsocketSupport = false
				require.NoError(t, ex.wsProcessBalanceAndPosition(t.Context(), []byte(tc.frame)), "wsProcessBalanceAndPosition must not need credentials")
				push := new(WebsocketBalanceAndPositionPush)
				require.NoError(t, json.Unmarshal([]byte(tc.frame), push), "the push must decode")
				assert.Equal(t, []any{push}, relayed(ex), "the push should be relayed without credentials")
			})
		}
	})

	t.Run("leaves the balance store to the account channel", func(t *testing.T) {
		t.Parallel()
		cash := func(amount, millis string) []byte {
			return []byte(`{"arg":{"channel":"balance_and_position","uid":"17"},"data":[{"pTime":"` + millis + `","eventType":"transferred",` +
				`"balData":[{"ccy":"USDT","cashBal":"` + amount + `","uTime":"` + millis + `"}],"posData":[],"trades":[]}]}`)
		}
		for _, tc := range []struct {
			name   string
			pushes [][]byte
			exp    accounts.Balance
		}{
			{
				name: "an account update arriving after a later cash push",
				pushes: [][]byte{
					accountPush(accountEventSnapshot, onlyAccountPage, `{"ccy":"USDT","eq":"100","cashBal":"100","availBal":"60","frozenBal":"40","uTime":"1795400000000"}`),
					cash("101", "1795400000020"),
					accountPush("event_update", "", `{"ccy":"USDT","eq":"100","cashBal":"100","availBal":"10","frozenBal":"90","uTime":"1795400000010"}`),
				},
				exp: accounts.Balance{Currency: currency.USDT, Total: 100, Hold: 90, Free: 10, UpdatedAt: time.UnixMilli(1795400000010)},
			},
			{
				name: "a transfer out before its account update",
				pushes: [][]byte{
					accountPush(accountEventSnapshot, onlyAccountPage, `{"ccy":"USDT","eq":"100","cashBal":"100","availBal":"100","frozenBal":"0","uTime":"1795400000000"}`),
					cash("10", "1795400000010"),
				},
				exp: accounts.Balance{Currency: currency.USDT, Total: 100, Free: 100, UpdatedAt: time.UnixMilli(1795400000000)},
			},
			{
				name: "a transfer out and its account update",
				pushes: [][]byte{
					accountPush(accountEventSnapshot, onlyAccountPage, `{"ccy":"USDT","eq":"100","cashBal":"100","availBal":"100","frozenBal":"0","uTime":"1795400000000"}`),
					cash("10", "1795400000010"),
					accountPush("event_update", "", `{"ccy":"USDT","eq":"10","cashBal":"10","availBal":"10","frozenBal":"0","uTime":"1795400000010"}`),
				},
				exp: accounts.Balance{Currency: currency.USDT, Total: 10, Free: 10, UpdatedAt: time.UnixMilli(1795400000010)},
			},
			{
				name: "unchanged cash beside an unrealised profit",
				pushes: [][]byte{
					accountPush(accountEventSnapshot, onlyAccountPage, `{"ccy":"USDT","eq":"120","cashBal":"100","availBal":"60","frozenBal":"40","upl":"20","uTime":"1795400000000"}`),
					cash("100", "1795400000010"),
				},
				exp: accounts.Balance{Currency: currency.USDT, Total: 120, Hold: 40, Free: 60, UpdatedAt: time.UnixMilli(1795400000000)},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				creds := &accounts.Credentials{Key: "fixture-key", Secret: "fixture-secret", ClientID: "fixture-passphrase"}
				ex, conn := newLoggedInAccountStream(t, creds)
				for i, push := range tc.pushes {
					require.NoErrorf(t, ex.wsHandleData(t.Context(), conn, push), "wsHandleData must not error on push %d", i+1)
				}
				got, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.USDT)
				require.NoError(t, err, "GetBalance must find the account's balance")
				assert.Equal(t, tc.exp, got, "the stored balance should be the account channel's latest breakdown, whatever cash the balance and position channel pushed")
			})
		}
	})

	t.Run("malformed push", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		require.Error(t, ex.wsProcessBalanceAndPosition(t.Context(), []byte(`{"data":`)), "a malformed push must error")
	})
}

func TestWebsocketProcessAccount(t *testing.T) {
	t.Parallel()

	creds := &accounts.Credentials{Key: "fixture-key", Secret: "fixture-secret", ClientID: "fixture-passphrase"}
	const usdt = `{"ccy":"USDT","eq":"102","cashBal":"100","availBal":"70","frozenBal":"30","liab":"5","uTime":"1795400001000"}`
	const btc = `{"ccy":"BTC","eq":"0.001","cashBal":"0.001","availBal":"0.001","frozenBal":"0","liab":"0","uTime":"1795400001000"}`
	storedUSDT := accounts.Balance{Currency: currency.USDT, Total: 102, Free: 70, Hold: 30, Borrowed: 5, UpdatedAt: time.UnixMilli(1795400001000)}
	storedBTC := accounts.Balance{Currency: currency.BTC, Total: 0.001, Free: 0.001, UpdatedAt: time.UnixMilli(1795400001000)}

	t.Run("saves the account's balances and relays the push", func(t *testing.T) {
		t.Parallel()
		ex, conn := newLoggedInAccountStream(t, creds)
		held := accounts.NewSubAccount(asset.Spot, "")
		held.Balances.Set(currency.USDT, accounts.Balance{Total: 100, Free: 100, UpdatedAt: time.UnixMilli(1795400000000)})
		require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{held}, true), "Save must store the earlier balance")
		frame := accountPush(accountEventSnapshot, onlyAccountPage, usdt, btc)
		require.NoError(t, ex.wsHandleData(t.Context(), conn, frame), "wsHandleData must not error")
		push := new(WebsocketAccountPush)
		require.NoError(t, json.Unmarshal(frame, push), "the push must decode")
		assert.Equal(t, []any{push}, relayed(ex), "the account push should be relayed")
		exp := accounts.SubAccounts{{AssetType: asset.Spot, Balances: accounts.CurrencyBalances{currency.USDT: storedUSDT, currency.BTC: storedBTC}}}
		stored, err := ex.Accounts.SubAccounts(creds, asset.Spot)
		require.NoError(t, err, "SubAccounts must not error")
		assert.Equal(t, exp, stored, "the account push should save each currency's equity, available, frozen and borrowed amounts as the trading account")
	})

	t.Run("merges the pages of a snapshot", func(t *testing.T) {
		t.Parallel()
		ex, conn := newLoggedInAccountStream(t, creds)
		for _, frame := range [][]byte{accountPush(accountEventSnapshot, firstAccountPage, usdt), accountPush(accountEventSnapshot, secondLastAccountPage, btc)} {
			require.NoError(t, ex.wsHandleData(t.Context(), conn, frame), "wsHandleData must not error")
		}
		exp := accounts.SubAccounts{{AssetType: asset.Spot, Balances: accounts.CurrencyBalances{currency.USDT: storedUSDT, currency.BTC: storedBTC}}}
		stored, err := ex.Accounts.SubAccounts(creds, asset.Spot)
		require.NoError(t, err, "SubAccounts must not error")
		assert.Equal(t, exp, stored, "a snapshot's pages should each add their currencies to the store")
	})

	t.Run("saves a push under the account its connection logged in as", func(t *testing.T) {
		t.Parallel()
		own := &accounts.Credentials{Key: "own-key", Secret: "own-secret", ClientID: "own-passphrase", SubAccount: "sub1", OneTimePassword: "123456"}
		other := &accounts.Credentials{Key: "other-key", Secret: "other-secret", ClientID: "other-passphrase"}
		ex, conn := newLoggedInAccountStream(t, own)
		ex.SetCredentials(other)
		require.NoError(t, ex.wsHandleData(t.Context(), conn, accountPush(accountEventSnapshot, onlyAccountPage, usdt)), "wsHandleData must not error")
		exp := accounts.SubAccounts{{AssetType: asset.Spot, Balances: accounts.CurrencyBalances{currency.USDT: storedUSDT}}}
		stored, err := ex.Accounts.SubAccounts(own, asset.Spot)
		require.NoError(t, err, "SubAccounts must find the login's account")
		assert.Equal(t, exp, stored, "the push should be saved under the credentials its connection logged in with")
		_, err = ex.Accounts.SubAccounts(other, asset.Spot)
		assert.ErrorIs(t, err, accounts.ErrNoSubAccounts, "the exchange's current credentials should not receive the push")
	})

	t.Run("the private reader saves a push under its connection's login after the credentials change", func(t *testing.T) {
		t.Parallel()
		release := make(chan struct{})
		srv := &websocketTestServer{push: func(path string, _ json.RawMessage) []string {
			if path != "/ws/v5/private" {
				return nil
			}
			select {
			case <-release:
				return []string{string(accountPush(accountEventSnapshot, onlyAccountPage, usdt))}
			case <-time.After(5 * time.Second):
				return nil
			}
		}}
		ex := newWebsocketTestExchange(t, srv, true, subscription.List{{Enabled: true, Channel: subscription.MyAccountChannel}})
		own, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		other := &accounts.Credentials{Key: "other-key", Secret: "other-secret", ClientID: "other-passphrase"}
		ex.SetCredentials(other)
		close(release)
		exp := accounts.SubAccounts{{AssetType: asset.Spot, Balances: accounts.CurrencyBalances{currency.USDT: storedUSDT}}}
		assert.Eventually(t, func() bool {
			stored, err := ex.Accounts.SubAccounts(own, asset.Spot)
			return err == nil && assert.ObjectsAreEqual(exp, stored)
		}, 5*time.Second, 10*time.Millisecond, "the push should be saved under the credentials its connection logged in with")
		_, err = ex.Accounts.SubAccounts(other, asset.Spot)
		assert.ErrorIs(t, err, accounts.ErrNoSubAccounts, "the exchange's current credentials should not receive the push")
	})

	t.Run("saves the trading account REST saves", func(t *testing.T) {
		t.Parallel()
		own := &accounts.Credentials{Key: "own-key", Secret: "own-secret", ClientID: "own-passphrase", SubAccount: "sub1"}
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/account/balance", r.URL.Path, "the balances should come from the trading account")
			writeOKXData(t, w, []map[string]any{{"uTime": "1795400001000", "details": []map[string]string{
				{"ccy": "USDT", "eq": "102", "cashBal": "100", "availBal": "70", "frozenBal": "30", "liab": "5", "uTime": "1795400001000"},
			}}})
		}))
		ex.SetCredentials(own)
		_, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "UpdateAccountBalances must not error")
		login, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		conn := new(accountStreamConnection)
		ex.recordWebsocketLogin(conn, login)
		require.NoError(t, ex.wsHandleData(t.Context(), conn, accountPush(accountEventSnapshot, onlyAccountPage, usdt)), "wsHandleData must not error")
		balances, err := ex.GetCachedCurrencyBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "GetCachedCurrencyBalances must not error")
		assert.Equal(t, accounts.CurrencyBalances{currency.USDT: storedUSDT}, balances, "the trading account should count once")
		stored, err := ex.Accounts.SubAccounts(own, asset.Spot)
		require.NoError(t, err, "SubAccounts must not error")
		exp := accounts.SubAccounts{{AssetType: asset.Spot, Balances: accounts.CurrencyBalances{currency.USDT: storedUSDT}}}
		assert.Equal(t, exp, stored, "REST and the account channel should save the trading account as one sub-account")
	})

	const (
		btc0  = `{"ccy":"BTC","eq":"1","cashBal":"1","availBal":"1","frozenBal":"0","uTime":"1795400000000"}`
		usdt0 = `{"ccy":"USDT","eq":"100","cashBal":"100","availBal":"60","frozenBal":"40","uTime":"1795400000000"}`
		eth1  = `{"ccy":"ETH","eq":"2","cashBal":"2","availBal":"2","frozenBal":"0","uTime":"1795400001000"}`
		xrp1  = `{"ccy":"XRP","eq":"5","cashBal":"5","availBal":"5","frozenBal":"0","uTime":"1795400001000"}`
		btc1  = `{"ccy":"BTC","eq":"4","cashBal":"4","availBal":"4","frozenBal":"0","uTime":"1795400001000"}`
		btc2  = `{"ccy":"BTC","eq":"3","cashBal":"3","availBal":"3","frozenBal":"0","uTime":"1795400002000"}`
	)
	balanceBTC0 := accounts.Balance{Currency: currency.BTC, Total: 1, Free: 1, UpdatedAt: time.UnixMilli(1795400000000)}
	balanceUSDT0 := accounts.Balance{Currency: currency.USDT, Total: 100, Hold: 40, Free: 60, UpdatedAt: time.UnixMilli(1795400000000)}
	balanceETH1 := accounts.Balance{Currency: currency.ETH, Total: 2, Free: 2, UpdatedAt: time.UnixMilli(1795400001000)}
	balanceXRP1 := accounts.Balance{Currency: currency.XRP, Total: 5, Free: 5, UpdatedAt: time.UnixMilli(1795400001000)}
	balanceBTC2 := accounts.Balance{Currency: currency.BTC, Total: 3, Free: 3, UpdatedAt: time.UnixMilli(1795400002000)}
	accountRows := func(page, data string) []byte {
		return []byte(`{"arg":{"channel":"account","uid":"17"},"eventType":"snapshot",` + page + `"data":` + data + `}`)
	}

	t.Run("keeps the balances a REST refresh made between a snapshot's pages", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/account/balance", r.URL.Path, "the balances should come from the trading account")
			writeOKXData(t, w, []map[string]any{{"uTime": "1795400001000", "details": []map[string]string{
				{"ccy": "ETH", "eq": "2", "cashBal": "2", "availBal": "2", "frozenBal": "0", "uTime": "1795400001000"},
				{"ccy": "USDT", "eq": "100", "cashBal": "100", "availBal": "60", "frozenBal": "40", "uTime": "1795400000000"},
			}}})
		}))
		ex.SetCredentials(creds)
		conn := new(accountStreamConnection)
		ex.recordWebsocketLogin(conn, creds)
		require.NoError(t, ex.wsHandleData(t.Context(), conn, accountPush(accountEventSnapshot, firstAccountPage, btc0)), "the snapshot's first page must be saved")
		_, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "UpdateAccountBalances must not error")
		require.NoError(t, ex.wsHandleData(t.Context(), conn, accountPush(accountEventSnapshot, secondLastAccountPage, usdt0)), "the snapshot's last page must be saved")
		stored, err := ex.Accounts.SubAccounts(creds, asset.Spot)
		require.NoError(t, err, "SubAccounts must not error")
		exp := accounts.SubAccounts{{AssetType: asset.Spot, Balances: accounts.CurrencyBalances{currency.ETH: balanceETH1, currency.USDT: balanceUSDT0}}}
		assert.Equal(t, exp, stored, "the snapshot's last page should merge into the balances REST refreshed, leaving out the currency REST removed")
	})

	type accountStep struct {
		frame   []byte
		refused bool
	}
	// usdtSubscriptionUpdate returns an update pushed to a subscription to USDT alone, holding 90 of 100 USDT
	usdtSubscriptionUpdate := func(updated string) []byte {
		return []byte(`{"arg":{"channel":"account","ccy":"USDT","uid":"17"},"eventType":"event_update","data":[{"uTime":"` + updated +
			`","details":[{"ccy":"USDT","eq":"100","cashBal":"100","availBal":"10","frozenBal":"90","uTime":"` + updated + `"}]}]}`)
	}
	for _, tc := range []struct {
		name  string
		steps []accountStep
		exp   accounts.CurrencyBalances
	}{
		{
			name:  "one page leaving a currency out",
			steps: []accountStep{{frame: accountPush(accountEventSnapshot, onlyAccountPage, usdt0)}},
			exp:   accounts.CurrencyBalances{currency.BTC: balanceBTC0, currency.USDT: balanceUSDT0},
		},
		{
			name:  "two pages leaving a currency out",
			steps: []accountStep{{frame: accountPush(accountEventSnapshot, firstAccountPage, btc0)}, {frame: accountPush(accountEventSnapshot, secondLastAccountPage, eth1)}},
			exp:   accounts.CurrencyBalances{currency.BTC: balanceBTC0, currency.USDT: balanceUSDT0, currency.ETH: balanceETH1},
		},
		{
			name:  "an empty account",
			steps: []accountStep{{frame: accountPush(accountEventSnapshot, onlyAccountPage)}},
			exp:   accounts.CurrencyBalances{currency.BTC: balanceBTC0, currency.USDT: balanceUSDT0},
		},
		{
			name:  "an update",
			steps: []accountStep{{frame: accountPush("event_update", "", eth1)}},
			exp:   accounts.CurrencyBalances{currency.BTC: balanceBTC0, currency.USDT: balanceUSDT0, currency.ETH: balanceETH1},
		},
		{
			name: "an update of another currency between the pages",
			steps: []accountStep{
				{frame: accountPush(accountEventSnapshot, firstAccountPage, btc0)},
				{frame: accountPush("event_update", "", eth1)},
				{frame: accountPush(accountEventSnapshot, secondLastAccountPage, xrp1)},
			},
			exp: accounts.CurrencyBalances{currency.BTC: balanceBTC0, currency.USDT: balanceUSDT0, currency.ETH: balanceETH1, currency.XRP: balanceXRP1},
		},
		{
			name: "an update of a listed currency between the pages",
			steps: []accountStep{
				{frame: accountPush(accountEventSnapshot, firstAccountPage, btc0)},
				{frame: accountPush("event_update", "", btc2)},
				{frame: accountPush(accountEventSnapshot, secondLastAccountPage, xrp1)},
			},
			exp: accounts.CurrencyBalances{currency.BTC: balanceBTC2, currency.USDT: balanceUSDT0, currency.XRP: balanceXRP1},
		},
		{
			name: "an update at the time of a page's balance",
			steps: []accountStep{
				{frame: accountPush(accountEventSnapshot, firstAccountPage, xrp1)},
				{frame: accountPush("event_update", "", `{"ccy":"XRP","eq":"6","cashBal":"6","availBal":"6","frozenBal":"0","uTime":"1795400001000"}`)},
				{frame: accountPush(accountEventSnapshot, secondLastAccountPage, eth1)},
			},
			exp: accounts.CurrencyBalances{
				currency.BTC:  balanceBTC0,
				currency.USDT: balanceUSDT0,
				currency.ETH:  balanceETH1,
				currency.XRP:  {Currency: currency.XRP, Total: 6, Free: 6, UpdatedAt: time.UnixMilli(1795400001000)},
			},
		},
		{
			name: "an update older than a page's balance",
			steps: []accountStep{
				{frame: accountPush(accountEventSnapshot, firstAccountPage, btc2)},
				{frame: accountPush("event_update", "", btc1), refused: true},
				{frame: accountPush(accountEventSnapshot, secondLastAccountPage, xrp1)},
			},
			exp: accounts.CurrencyBalances{currency.BTC: balanceBTC2, currency.USDT: balanceUSDT0, currency.XRP: balanceXRP1},
		},
		{
			name: "one currency's update between the pages at the time of a page's balance",
			steps: []accountStep{
				{frame: accountPush(accountEventSnapshot, firstAccountPage, usdt0)},
				{frame: usdtSubscriptionUpdate("1795400000000")},
				{frame: accountPush(accountEventSnapshot, secondLastAccountPage, btc0)},
			},
			exp: accounts.CurrencyBalances{
				currency.BTC:  balanceBTC0,
				currency.USDT: {Currency: currency.USDT, Total: 100, Hold: 90, Free: 10, UpdatedAt: time.UnixMilli(1795400000000)},
			},
		},
		{
			name: "one currency's newer update between the pages",
			steps: []accountStep{
				{frame: accountPush(accountEventSnapshot, firstAccountPage, usdt0)},
				{frame: usdtSubscriptionUpdate("1795400001000")},
				{frame: accountPush(accountEventSnapshot, secondLastAccountPage, btc0)},
			},
			exp: accounts.CurrencyBalances{
				currency.BTC:  balanceBTC0,
				currency.USDT: {Currency: currency.USDT, Total: 100, Hold: 90, Free: 10, UpdatedAt: time.UnixMilli(1795400001000)},
			},
		},
		{
			name:  "a null account row",
			steps: []accountStep{{frame: accountRows(onlyAccountPage, `[null]`)}},
			exp:   accounts.CurrencyBalances{currency.BTC: balanceBTC0, currency.USDT: balanceUSDT0},
		},
		{
			name:  "no account row",
			steps: []accountStep{{frame: accountRows(onlyAccountPage, `[]`)}},
			exp:   accounts.CurrencyBalances{currency.BTC: balanceBTC0, currency.USDT: balanceUSDT0},
		},
		{
			name:  "an account row without its currencies",
			steps: []accountStep{{frame: accountRows(onlyAccountPage, `[{"uTime":"1795400001000","totalEq":"160"}]`)}},
			exp:   accounts.CurrencyBalances{currency.BTC: balanceBTC0, currency.USDT: balanceUSDT0},
		},
		{
			name:  "a null currency row",
			steps: []accountStep{{frame: accountPush(accountEventSnapshot, onlyAccountPage, eth1, "null")}},
			exp:   accounts.CurrencyBalances{currency.BTC: balanceBTC0, currency.USDT: balanceUSDT0, currency.ETH: balanceETH1},
		},
		{
			name:  "one currency's subscription",
			steps: []accountStep{{frame: []byte(`{"arg":{"channel":"account","ccy":"ETH","uid":"17"},"eventType":"snapshot",` + onlyAccountPage + `"data":[{"uTime":"1795400001000","details":[` + eth1 + `]}]}`)}},
			exp:   accounts.CurrencyBalances{currency.BTC: balanceBTC0, currency.USDT: balanceUSDT0, currency.ETH: balanceETH1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, conn := newLoggedInAccountStream(t, creds)
			require.NoError(t, ex.wsHandleData(t.Context(), conn, accountPush(accountEventSnapshot, onlyAccountPage, btc0, usdt0)), "the first snapshot must be saved")
			for i, step := range tc.steps {
				err := ex.wsHandleData(t.Context(), conn, step.frame)
				if step.refused {
					require.ErrorContainsf(t, err, "out of sequence", "wsHandleData must refuse push %d, older than the stored balance", i+1)
					continue
				}
				require.NoErrorf(t, err, "wsHandleData must not error on push %d", i+1)
			}
			stored, err := ex.Accounts.SubAccounts(creds, asset.Spot)
			require.NoError(t, err, "SubAccounts must not error")
			assert.Equal(t, accounts.SubAccounts{{AssetType: asset.Spot, Balances: tc.exp}}, stored, "each push should merge its currencies into the account's balances")
		})
	}

	t.Run("relays a push without currencies without saving it", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name  string
			frame []byte
		}{
			{name: "a snapshot without its page", frame: accountPush(accountEventSnapshot, "")},
			{name: "an empty account's complete snapshot", frame: accountPush(accountEventSnapshot, onlyAccountPage)},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				require.NoError(t, ex.wsProcessAccount(t.Context(), nil, tc.frame), "a push without currencies must need no login")
				push := new(WebsocketAccountPush)
				require.NoError(t, json.Unmarshal(tc.frame, push), "the push must decode")
				assert.Equal(t, []any{push}, relayed(ex), "a push without currencies should still be relayed")
			})
		}
	})

	t.Run("saving needs the connection's login", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name       string
			connection bool
			frame      []byte
		}{
			{name: "no connection", frame: accountPush(accountEventSnapshot, onlyAccountPage, usdt)},
			{name: "a connection which has not logged in", connection: true, frame: accountPush(accountEventSnapshot, onlyAccountPage, usdt)},
			{name: "an update", connection: true, frame: accountPush("event_update", "", usdt)},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newAccountStreamExchange(t, creds)
				var conn websocket.Connection
				if tc.connection {
					conn = new(accountStreamConnection)
				}
				err := ex.wsHandleData(t.Context(), conn, tc.frame)
				require.ErrorIs(t, err, errUnknownPushAccount, "a push on a connection with no login must not be saved")
				assert.Empty(t, relayed(ex), "an unsaved push should not be relayed")
				_, err = ex.Accounts.SubAccounts(creds, asset.Spot)
				assert.ErrorIs(t, err, accounts.ErrNoSubAccounts, "a push on a connection with no login should leave the balance store untouched")
			})
		}
	})

	t.Run("malformed push", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		require.Error(t, ex.wsProcessAccount(t.Context(), nil, []byte(`{"data":`)), "a malformed push must error")
	})
}

func TestWebsocketProcessOrders(t *testing.T) {
	t.Parallel()

	t.Run("relays the order", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		handleFixture(t, ex, "orders")
		exp := []any{&order.Detail{
			Price:                31527.1,
			Amount:               0.001,
			AverageExecutedPrice: 31527.1,
			ExecutedAmount:       0.001,
			Cost:                 31527.1 * 0.001,
			CostAsset:            currency.USDT,
			Fee:                  0.02522168,
			FeeAsset:             currency.USDT,
			Exchange:             ex.Name,
			OrderID:              "452197707845865472",
			ClientOrderID:        "b15",
			Type:                 order.Limit,
			Side:                 order.Sell,
			Status:               order.Filled,
			AssetType:            asset.Spot,
			ReduceOnly:           true,
			Leverage:             3,
			Date:                 time.UnixMilli(1654084334977),
			LastUpdated:          time.UnixMilli(1654084353264),
			CloseTime:            time.UnixMilli(1654084353263),
			Pair:                 mainPair,
		}}
		assert.Equal(t, exp, relayed(ex), "the order push should relay the order's details")
	})

	t.Run("reports each order as OKX states it", func(t *testing.T) {
		t.Parallel()
		for _, tc := range statedOrderCases("partially_filled", order.PartiallyFilled) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				push, err := json.Marshal(map[string]any{"arg": map[string]string{"channel": "orders", "instType": tc.row["instType"]}, "data": []map[string]string{tc.row}})
				require.NoError(t, err, "the push must encode")
				require.NoError(t, ex.wsProcessOrders(t.Context(), push), "wsProcessOrders must not error")
				exp := tc.exp
				exp.Exchange = ex.Name
				assert.Equal(t, []any{&exp}, relayed(ex), "the order push should report the order's sizes, trade currency and reduce only flag as the REST readers do")
			})
		}
	})

	t.Run("reports a maker rebate as a negative fee", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		push, err := json.Marshal(map[string]any{"arg": map[string]string{"channel": "orders", "instType": instTypeSpot}, "data": []map[string]string{makerRebateOrderRow("filled", "2")}})
		require.NoError(t, err, "the push must encode")
		require.NoError(t, ex.wsProcessOrders(t.Context(), push), "wsProcessOrders must not error")
		exp := []any{&order.Detail{
			Price: 100, Amount: 2, AverageExecutedPrice: 100, ExecutedAmount: 2, Cost: 200, CostAsset: currency.USDT,
			Fee: -0.5, FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "ORD-1", ClientOrderID: "client-1",
			Type: order.Limit, Side: order.Buy, Status: order.Filled, AssetType: asset.Spot,
			Date: time.UnixMilli(1700000000000), LastUpdated: time.UnixMilli(1700000001000), Pair: mainPair,
		}}
		assert.Equal(t, exp, relayed(ex), "the order push should net the maker rebate into the fee")
	})

	t.Run("a fill merged into the stored order keeps its rebate, cost currency and no remainder", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		push, err := json.Marshal(map[string]any{"arg": map[string]string{"channel": "orders", "instType": instTypeSpot}, "data": []map[string]string{makerRebateOrderRow("filled", "2")}})
		require.NoError(t, err, "the push must encode")
		require.NoError(t, ex.wsProcessOrders(t.Context(), push), "wsProcessOrders must not error")
		got := relayed(ex)
		require.Len(t, got, 1, "the order push must reach the data handler")
		filled, ok := got[0].(*order.Detail)
		require.True(t, ok, "the order push must reach the data handler as an order detail")
		live := order.Detail{
			Exchange: ex.Name, OrderID: "ORD-1", ClientOrderID: "client-1", Pair: mainPair, AssetType: asset.Spot, Type: order.Limit,
			Side: order.Buy, Status: order.Active, Price: 100, Amount: 2, RemainingAmount: 2, Date: time.UnixMilli(1700000000000),
			LastUpdated: time.UnixMilli(1700000000000),
		}
		require.NoError(t, live.UpdateOrderFromDetail(filled), "the order store's merge must not error")
		exp := order.Detail{
			Exchange: ex.Name, OrderID: "ORD-1", ClientOrderID: "client-1", Pair: mainPair, AssetType: asset.Spot, Type: order.Limit,
			Side: order.Buy, Status: order.Filled, Price: 100, Amount: 2, AverageExecutedPrice: 100, ExecutedAmount: 2, Cost: 200,
			CostAsset: currency.USDT, Fee: -0.5, FeeAsset: currency.USDT, Date: time.UnixMilli(1700000000000),
			LastUpdated: time.UnixMilli(1700000001000),
		}
		assert.Equal(t, exp, live, "the stored order should take the fill's rebate, cost currency and lack of a remainder")
	})

	t.Run("reports the cost of the fills as the REST readers do", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			instType  string
			instID    string
			cost      float64
			costAsset currency.Code
		}{
			{instTypeSpot, "BTC-USDT", 60, currency.USDT},
			{instTypeMargin, "BTC-USDT", 60, currency.USDT},
			{instTypeSwap, "BTC-USDT-SWAP", 0, currency.EMPTYCODE},
		} {
			t.Run(tc.instType, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				push := []byte(`{"arg":{"channel":"orders","instType":"` + tc.instType + `"},"data":[{"instId":"` + tc.instID + `","ordId":"1","ordType":"limit","side":"buy","state":"partially_filled","sz":"4","accFillSz":"3","avgPx":"20","px":"20","cTime":"1700000000000"}]}`)
				require.NoError(t, ex.wsProcessOrders(t.Context(), push), "wsProcessOrders must not error")
				got := relayed(ex)
				require.Len(t, got, 1, "the order push must reach the data handler")
				d, ok := got[0].(*order.Detail)
				require.True(t, ok, "the order push must reach the data handler as an order detail")
				assert.Equal(t, tc.cost, d.Cost, "the cost should be the average price times the accumulated fill on spot and margin only")
				assert.Equal(t, tc.costAsset, d.CostAsset, "the cost asset should be the pair quote on spot and margin only")
			})
		}
	})

	t.Run("parses every order type", func(t *testing.T) {
		t.Parallel()
		for ordType, exp := range map[string]struct {
			oType order.Type
			tif   order.TimeInForce
		}{
			orderPostOnly:                         {order.Limit, order.PostOnly},
			orderFOK:                              {order.Limit, order.FillOrKill},
			orderIOC:                              {order.Limit, order.ImmediateOrCancel},
			orderOptionFOK:                        {order.Limit, order.FillOrKill},
			orderOptimalLimitIOC:                  {order.OptimalLimit, order.ImmediateOrCancel},
			orderMarketMakerProtection:            {order.MarketMakerProtection, order.UnknownTIF},
			orderMarketMakerProtectionAndPostOnly: {order.MarketMakerProtection, order.PostOnly},
			orderRPI:                              {order.Limit, order.UnknownTIF},
			orderELP:                              {order.Limit, order.UnknownTIF},
		} {
			t.Run(ordType, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
				push := []byte(`{"arg":{"channel":"orders","instType":"SPOT"},"data":[{"instId":"BTC-USDT","ordId":"1","ordType":"` + ordType + `","side":"buy","state":"live","sz":"1","accFillSz":"0","cTime":"1700000000000"}]}`)
				require.NoErrorf(t, ex.wsProcessOrders(t.Context(), push), "a %s order push must not error", ordType)
				got := relayed(ex)
				require.Len(t, got, 1, "the order push must reach the data handler")
				d, ok := got[0].(*order.Detail)
				require.True(t, ok, "the order push must reach the data handler as an order detail")
				assert.Equal(t, exp.oType, d.Type, "the push should read the order type")
				assert.Equal(t, exp.tif, d.TimeInForce, "the push should preserve the time in force")
			})
		}
	})

	for _, tc := range []struct {
		name  string
		frame string
	}{
		{"unknown instrument type", `{"arg":{"channel":"orders","instType":"EVENTS"},"data":[]}`},
		{"unknown order type", `{"arg":{"channel":"orders","instType":"SPOT"},"data":[{"instId":"BTC-USDT","ordId":"1","ordType":"unknown","side":"buy","state":"live"}]}`},
		{"unknown state", `{"arg":{"channel":"orders","instType":"SPOT"},"data":[{"instId":"BTC-USDT","ordId":"1","ordType":"limit","side":"buy","state":"unknown"}]}`},
		{"malformed push", `{"data":`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			require.Error(t, ex.wsProcessOrders(t.Context(), []byte(tc.frame)), "wsProcessOrders must error")
			assert.Empty(t, ex.Websocket.DataHandler.C, "nothing should be relayed")
		})
	}
}

func TestWebsocketProcessFills(t *testing.T) {
	t.Parallel()

	t.Run("relays the fills to the fills feed", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketHandlerExchange(t)
		enableFillsFeed(ex)
		handleFixture(t, ex, "fills")
		exp := []any{[]fill.Data{{
			ID:            "12345",
			Timestamp:     time.UnixMilli(1705449605015),
			Exchange:      ex.Name,
			AssetType:     asset.PerpetualSwap,
			CurrencyPair:  perpetualSwapPair,
			Side:          order.Buy,
			OrderID:       "680800019749904384",
			ClientOrderID: "1234567890",
			TradeID:       "12345",
			Price:         70000,
			Amount:        100,
		}}}
		assert.Equal(t, exp, relayed(ex), "the fill should be relayed to the fills feed")
	})

	t.Run("relays a spot instrument's fill once", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketHandlerExchange(t)
		enableFillsFeed(ex)
		require.NoError(t, ex.wsProcessFills([]byte(`{"arg":{"channel":"fills"},"data":[{"instId":"BTC-USDT","fillSz":"0.5","fillPx":"61000","side":"sell","ts":"1705449605015","ordId":"1","clOrdId":"","tradeId":"2","execType":"M","count":"1"}]}`)), "wsProcessFills must not error")
		exp := []any{[]fill.Data{{
			ID: "2", Timestamp: time.UnixMilli(1705449605015), Exchange: ex.Name, AssetType: asset.Spot, CurrencyPair: mainPair,
			Side: order.Sell, OrderID: "1", TradeID: "2", Price: 61000, Amount: 0.5,
		}}}
		assert.Equal(t, exp, relayed(ex), "a fill of an instrument enabled on spot and margin should be relayed once")
	})

	t.Run("fills feed off", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketHandlerExchange(t)
		handleFixture(t, ex, "fills")
		assert.Empty(t, ex.Websocket.DataHandler.C, "no fill should be relayed with the fills feed off")
	})

	t.Run("instrument not enabled", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		enableFillsFeed(ex)
		err := ex.wsProcessFills([]byte(`{"arg":{"channel":"fills"},"data":[{"instId":"XRP-USDT","fillSz":"1","fillPx":"1","side":"buy","ts":"1705449605015","ordId":"1","tradeId":"1"}]}`))
		require.ErrorIs(t, err, asset.ErrNotEnabled, "a fill of an instrument which is not enabled must error")
		assert.Empty(t, relayed(ex), "a push with no fill of an enabled instrument should relay nothing")
	})

	t.Run("relays the fills of enabled instruments beside one that is not", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketHandlerExchange(t)
		enableFillsFeed(ex)
		err := ex.wsProcessFills([]byte(`{"arg":{"channel":"fills","uid":"17"},"data":[` +
			`{"instId":"BTC-USDT","fillSz":"1","fillPx":"50000","side":"buy","ts":"1795413500000","ordId":"1","tradeId":"101"},` +
			`{"instId":"XRP-USDT","fillSz":"1","fillPx":"1","side":"buy","ts":"1795413500001","ordId":"2","tradeId":"102"},` +
			`{"instId":"BTC-USDT","fillSz":"2","fillPx":"50001","side":"sell","ts":"1795413500002","ordId":"3","tradeId":"103"}]}`))
		require.ErrorIs(t, err, asset.ErrNotEnabled, "the fill of an instrument which is not enabled must be reported")
		assert.ErrorContains(t, err, "102", "the error should name the fill it could not relay")
		exp := []any{[]fill.Data{
			{
				ID: "101", Timestamp: time.UnixMilli(1795413500000), Exchange: ex.Name, AssetType: asset.Spot, CurrencyPair: mainPair,
				Side: order.Buy, OrderID: "1", TradeID: "101", Price: 50000, Amount: 1,
			},
			{
				ID: "103", Timestamp: time.UnixMilli(1795413500002), Exchange: ex.Name, AssetType: asset.Spot, CurrencyPair: mainPair,
				Side: order.Sell, OrderID: "3", TradeID: "103", Price: 50001, Amount: 2,
			},
		}}
		assert.Equal(t, exp, relayed(ex), "the fills of enabled instruments should be relayed in one batch")
	})

	t.Run("malformed push", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		enableFillsFeed(ex)
		require.Error(t, ex.wsProcessFills([]byte(`{"data":`)), "a malformed push must error")
	})
}

func TestWebsocketProcessSpreadOrders(t *testing.T) {
	t.Parallel()

	t.Run("relays the order", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		handleFixture(t, ex, "sprd-orders")
		exp := []any{[]order.Detail{{
			Price:                999,
			Amount:               3,
			AverageExecutedPrice: 998,
			ExecutedAmount:       1,
			RemainingAmount:      2,
			Exchange:             ex.Name,
			OrderID:              "312269865356374016",
			ClientOrderID:        "b1",
			Type:                 order.Limit,
			Side:                 order.Buy,
			Status:               order.PartiallyFilled,
			AssetType:            asset.Spread,
			Date:                 time.UnixMilli(1597026383085),
			LastUpdated:          time.UnixMilli(1597026383085),
			Pair:                 spreadPair,
		}}}
		assert.Equal(t, exp, relayed(ex), "the spread order push should relay the order's details")
	})

	t.Run("relays the push", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		require.NoError(t, ex.wsProcessSpreadOrders(t.Context(), []byte(`{"arg":{"channel":"sprd-orders","sprdId": "BTC-USDT_BTC-USDT-SWAP", "uid": "614488474791936"}, "data": [{"sprdId": "BTC-USDT_BTC-UST-SWAP", "ordId": "312269865356374016", "clOrdId": "b1", "tag": "", "px": "999", "sz": "3", "ordType": "limit", "side": "buy", "fillSz": "0", "fillPx": "", "tradeId": "", "accFillSz": "0", "pendingFillSz": "2", "pendingSettleSz": "1", "canceledSz": "1", "state": "live", "avgPx": "0", "cancelSource": "", "uTime": "1597026383085", "cTime": "1597026383085", "code": "0", "msg": "", "reqId": "", "amendResult": ""}]}`)), "wsProcessSpreadOrders must not error")
		assert.Len(t, ex.Websocket.DataHandler.C, 1, "the push should be relayed")
	})

	t.Run("reports fills from the accumulated size", func(t *testing.T) {
		t.Parallel()
		// The executed amount is the accumulated fill rather than the last
		// fill, the remainder is what is left of the size, and the order
		// type's time in force is kept
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		push := `{"arg":{"channel":"sprd-orders","sprdId":"BTC-USDT_BTC-USDT-SWAP","uid":"614488474791936"},"data":[{"sprdId":"BTC-USDT_BTC-USDT-SWAP","ordId":"312269865356374016","clOrdId":"b1","px":"999","sz":"3","ordType":"post_only","side":"buy","fillSz":"0.5","fillPx":"999","accFillSz":"2","pendingFillSz":"1","state":"partially_filled","avgPx":"998","uTime":"1597026383085","cTime":"1597026383085"}]}`
		require.NoError(t, ex.wsProcessSpreadOrders(t.Context(), []byte(push)), "wsProcessSpreadOrders must not error")
		stamp := time.UnixMilli(1597026383085)
		exp := []any{[]order.Detail{{
			AssetType: asset.Spread, Amount: 3, AverageExecutedPrice: 998, ClientOrderID: "b1", Date: stamp, Exchange: ex.Name,
			ExecutedAmount: 2, OrderID: "312269865356374016", Pair: spreadPair, Price: 999, RemainingAmount: 1,
			Side: order.Buy, Status: order.PartiallyFilled, Type: order.Limit, LastUpdated: stamp, TimeInForce: order.PostOnly,
		}}}
		assert.Equal(t, exp, relayed(ex), "the push should report the accumulated fill, the remainder and the time in force")
	})

	t.Run("reports the size still to fill", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		push := `{"arg":{"channel":"sprd-orders","sprdId":"BTC-USDT_BTC-USDT-SWAP","uid":"614488474791936"},"data":[` +
			`{"sprdId":"BTC-USDT_BTC-USDT-SWAP","ordId":"1","px":"999","sz":"3","ordType":"limit","side":"buy","accFillSz":"0","pendingFillSz":"2","pendingSettleSz":"1","canceledSz":"1","state":"live","uTime":"1597026383085","cTime":"1597026383085"},` +
			`{"sprdId":"BTC-USDT_BTC-USDT-SWAP","ordId":"2","px":"999","sz":"3","ordType":"limit","side":"buy","accFillSz":"1","pendingFillSz":"0","canceledSz":"2","state":"partially_filled","avgPx":"999","uTime":"1597026383085","cTime":"1597026383085"},` +
			`{"sprdId":"BTC-USDT_BTC-USDT-SWAP","ordId":"3","px":"999","sz":"3","ordType":"limit","side":"buy","accFillSz":"1","pendingFillSz":"","state":"partially_filled","avgPx":"999","uTime":"1597026383085","cTime":"1597026383085"}]}`
		require.NoError(t, ex.wsProcessSpreadOrders(t.Context(), []byte(push)), "wsProcessSpreadOrders must not error")
		stamp := time.UnixMilli(1597026383085)
		detail := func(id string, status order.Status, executed, remaining float64) order.Detail {
			d := order.Detail{
				AssetType: asset.Spread, Amount: 3, ExecutedAmount: executed, RemainingAmount: remaining, Date: stamp, Exchange: ex.Name, OrderID: id,
				Pair: spreadPair, Price: 999, Side: order.Buy, Status: status, Type: order.Limit, LastUpdated: stamp,
			}
			if executed > 0 {
				d.AverageExecutedPrice = 999
			}
			return d
		}
		exp := []any{[]order.Detail{
			detail("1", order.Active, 0, 2),
			detail("2", order.PartiallyFilled, 1, 0),
			detail("3", order.PartiallyFilled, 1, 2),
		}}
		assert.Equal(t, exp, relayed(ex), "the push should report the size OKX states is still to fill, and what the fills leave of an order stating none")
	})

	for _, tc := range []struct {
		name   string
		frame  string
		expErr error
	}{
		{"empty data", `{"arg":{"channel":"sprd-orders"},"data":[]}`, kline.ErrNoTimeSeriesDataToConvert},
		{"unknown state", `{"arg":{"channel":"sprd-orders"},"data":[{"sprdId":"BTC-USDT_BTC-USDT-SWAP","ordId":"1","ordType":"limit","side":"buy","state":"unknown"}]}`, nil},
		{"unknown order type", `{"arg":{"channel":"sprd-orders"},"data":[{"sprdId":"BTC-USDT_BTC-USDT-SWAP","ordId":"1","ordType":"unknown","side":"buy","state":"live"}]}`, nil},
		{"malformed push", `{"data":`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			err := ex.wsProcessSpreadOrders(t.Context(), []byte(tc.frame))
			if tc.expErr != nil {
				require.ErrorIs(t, err, tc.expErr, "wsProcessSpreadOrders must return the expected error")
			} else {
				require.Error(t, err, "wsProcessSpreadOrders must error")
			}
			assert.Empty(t, ex.Websocket.DataHandler.C, "nothing should be relayed")
		})
	}
}

func TestWebsocketProcessSpreadTrades(t *testing.T) {
	t.Parallel()

	t.Run("relays the fills to the fills feed", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		enableFillsFeed(ex)
		handleFixture(t, ex, "sprd-trades")
		exp := []any{[]fill.Data{{
			ID:            "123",
			Timestamp:     time.UnixMilli(1597026383085),
			Exchange:      ex.Name,
			AssetType:     asset.Spread,
			CurrencyPair:  spreadPair,
			Side:          order.Buy,
			OrderID:       "123445",
			ClientOrderID: "b16",
			TradeID:       "123",
			Price:         999,
			Amount:        3,
		}}}
		assert.Equal(t, exp, relayed(ex), "the spread fill should be relayed to the fills feed")
	})

	t.Run("relays the fills with trade saving on", func(t *testing.T) {
		t.Parallel()
		const push = `{"arg":{"channel":"sprd-trades","sprdId":"BTC-USDT_BTC-USDT-SWAP","uid":"614488474791936"},"data":[{"sprdId":"BTC-USDT_BTC-USDT-SWAP","tradeId":"123","ordId":"123445","clOrdId":"b16","tag":"gct","fillPx":"999","fillSz":"3","state":"filled","side":"buy","execType":"M","ts":"1597026383085","legs":[],"code":"0","msg":"Trade filled"}]}`
		for _, tc := range []struct {
			name      string
			from, to  string
			expErr    string
			expPrice  float64
			expAmount float64
		}{
			{"a fill at a zero spread price", `"fillPx":"999"`, `"fillPx":"0"`, "", 0, 3},
			{"a fill the trade buffer refuses", `"fillSz":"3"`, `"fillSz":"0"`, "received invalid trade data", 999, 0},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				frame := strings.Replace(push, tc.from, tc.to, 1)
				require.NotEqual(t, push, frame, "the replacement must find its fixture anchor")
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				ex.SetSaveTradeDataStatus(true)
				enableFillsFeed(ex)
				err := ex.wsProcessSpreadTrades([]byte(frame))
				if tc.expErr == "" {
					assert.NoError(t, err, "wsProcessSpreadTrades should save the fill")
				} else {
					assert.ErrorContains(t, err, tc.expErr, "wsProcessSpreadTrades should report the failed save")
				}
				exp := []any{[]fill.Data{{
					ID:            "123",
					Timestamp:     time.UnixMilli(1597026383085),
					Exchange:      ex.Name,
					AssetType:     asset.Spread,
					CurrencyPair:  spreadPair,
					Side:          order.Buy,
					OrderID:       "123445",
					ClientOrderID: "b16",
					TradeID:       "123",
					Price:         tc.expPrice,
					Amount:        tc.expAmount,
				}}}
				assert.Equal(t, exp, relayed(ex), "the spread fill should be relayed to the fills feed whatever its save does")
			})
		}
	})

	t.Run("fills feed off", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		handleFixture(t, ex, "sprd-trades")
		assert.Empty(t, ex.Websocket.DataHandler.C, "no fill should be relayed with the fills feed off")
	})

	t.Run("honours the save trade data gate", func(t *testing.T) {
		t.Parallel()
		// The private spread trades stream keeps the same saving gate as the
		// public one: with trade saving off a push must not reach the shared
		// trade buffer, while the decode still runs first so a malformed push
		// reports its error. The trade carries a zero fill size, which the
		// buffer rejects with an error, so that error tells whether the buffer
		// was reached.
		const push = `{"arg":{"channel":"sprd-trades", "sprdId": "BTC-USDT_BTC-USDT-SWAP", "uid": "614488474791936" }, "data":[ { "sprdId":"BTC-USDT-SWAP_BTC-USDT-200329", "tradeId":"123", "ordId":"123445", "clOrdId": "b16", "tag":"", "fillPx":"999", "fillSz":"3", "state": "filled", "side":"buy", "execType":"M", "ts":"1597026383085", "legs": [ { "instId": "BTC-USDT-SWAP", "px": "20000", "sz": "3", "szCont": "0.03", "side": "buy", "fee": "", "feeCcy": "", "tradeId": "1232342342" }, { "instId": "BTC-USDT-200329", "px": "21000", "sz": "3", "szCont": "0.03", "side": "sell", "fee": "", "feeCcy": "", "tradeId": "5345646634" } ], "code": "", "msg":""}]}`
		zeroSize := strings.Replace(push, `"fillSz":"3"`, `"fillSz":"0"`, 1)
		require.NotEqual(t, push, zeroSize, "the zero size replacement must find its fixture anchor")
		for _, tc := range []struct {
			name       string
			save       bool
			push       string
			expErr     string
			wantAnyErr bool
		}{
			{name: "private spread trades decode", save: false, push: push},
			{name: "saving off", save: false, push: zeroSize},
			{name: "saving on", save: true, push: zeroSize, expErr: "received invalid trade data"},
			// The decoder's message differs per JSON backend (stdlib vs sonic),
			// so only the presence of the error is pinned for the malformed push.
			{name: "malformed push with saving off", save: false, push: "{", wantAnyErr: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
				ex.SetSaveTradeDataStatus(tc.save)
				err := ex.wsProcessSpreadTrades([]byte(tc.push))
				switch {
				case tc.wantAnyErr:
					assert.Error(t, err, "a malformed push should still report its decode error")
				case tc.expErr == "":
					assert.NoError(t, err, "a push with trade saving off should not reach the trade buffer")
				default:
					assert.ErrorContains(t, err, tc.expErr, "the push should report the expected error")
				}
			})
		}
	})

	t.Run("malformed push", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		enableFillsFeed(ex)
		require.Error(t, ex.wsProcessSpreadTrades([]byte(`{"data":`)), "a malformed push must error")
	})
}
