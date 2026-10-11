# GoCryptoTrader package Hyperliquid

<img src="../../common/gctlogo.png" alt="GoCryptoTrader logo" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/hyperliquid)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)

This hyperliquid package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Hyperliquid Exchange

### Current Features

+ Spot, perpetual and builder-deployed (HIP-3) perpetual DEX markets
+ REST and websocket tickers, L2 orderbooks, trades and candles
+ Account balances, orders, fills, TWAP history and slice fills, funding payments and ledger history
+ Exchange status, and vault, staking, referral, borrow/lend, outcome market and template, HIP-3\* DEX approval
  state and deployment information
+ Signed orders, cancels, modifications, TWAP and trailing stop orders, scheduled cancels, and leverage and margin
  updates, with orders, cancels and modifications also available over the websocket
+ Transfers between spot, perpetual DEXs and accounts, USDC bridge withdrawals, HyperEVM sends, HIP-3 backstop
  liquidator transfers, staking, and API wallet and builder fee approvals
+ Outcome splits, merges and negations, account abstraction settings, reward claims, request weight reservation,
  pending nonce invalidation, and gossip priority bids and auction status
+ Funding rates, open interest and fee estimates

### How to enable

+ [Enable via configuration](../../config/README.md#enable-exchange-via-config-example)

### Credentials

Hyperliquid accounts are EVM addresses, so the credentials hold addresses and private keys rather than API keys:

+ `Key` is the account address, whose data is queried
+ `Secret` is optional, and is the private key of the account or of one of its approved API wallets
+ `SubAccount` is optional, and is a subaccount or vault the account controls, whose data is then queried and on whose
  behalf trading actions are signed

Account data and account websocket feeds need only `Key`. Signed actions need `Secret`, and an approved API wallet is
recommended for trading. Transfers, withdrawals, staking, approvals and abstraction settings are user-signed, so they
need `Secret` to be the account's own private key, except `AgentSendAsset`, `AgentEnableDEXAbstraction`,
`AgentSetUserAbstraction`, `TransferVaultUSDC` and `TransferHIP3BackstopLiquidator`, which an API wallet can sign.
`SendAsset`, `AgentSendAsset`, `TransferUSDCBetweenSpotAndPerp`, `SetUserDEXAbstraction` and `SetUserAbstraction` act
for a configured subaccount and are refused when `SubAccount` is a vault, while the other user-signed actions,
including `WithdrawCryptocurrencyFunds`, are refused whenever `SubAccount` is set. The remaining actions act for the
account itself. The account, signer and subaccount or vault are checked against Hyperliquid's records before the first
signed action for each combination of them, and again after Hyperliquid rejects the signer; checked combinations are
remembered, so alternating subaccounts or credentials does not repeat the check.

Signed actions use millisecond nonces, which only increase within one exchange instance, so the same signer must not be
used by several instances or processes at once.

### Testnet

Set `useSandbox` to `true` to sign for testnet. Endpoints left at their production defaults then move to testnet, while
custom endpoints are kept, so `useSandbox` must match the environment behind a custom endpoint.

### Orders

+ Orders are sized in the base currency, so `QuoteAmount` is not supported.
+ Market orders require a `SlippageTolerance` above zero and below one, and are sent as IOC limit orders bounded by it,
  so they accept only IOC or no time in force.
+ Trigger orders are supported on perpetual markets only, accept only GTC or no time in force, as Hyperliquid places a
  triggered order as GTC, and need not be reduce-only. Hyperliquid triggers on the mark price, so trigger orders and
  take-profit or stop-loss children must set `TriggerPriceType` to `order.MarkPrice`.
+ `ModifyOrder` replaces resting limit orders only. Hyperliquid replaces an order only when the replacement rests, so a
  GTC replacement that would match is rejected and trigger and IOC replacements are not supported.
+ Fee estimates follow Hyperliquid's fee formula, from the account's rates and referral discount when credentials are
  set, and the base tier rates otherwise. A spot pair between two spot quote assets, the tokens that quote a listed spot
  market, pays a fifth of the rate. A builder-deployed perpetual market's deployer fee scale raises its fees, and growth
  mode cuts its fees and rebates to a tenth. The referral discount cuts fees but not rebates. Aligned quote asset
  discounts are not applied, as Hyperliquid documents none on mainnet.
+ `GetActiveOrders` reads open orders only from the DEXs of the requested perpetual pairs, or from every DEX when the
  request names no pairs or a pair whose market is not cached. Each DEX's orders come from `frontendOpenOrders`, which
  returns every open order of most accounts but only 100, not necessarily the newest, for an account with very many,
  such as one of HLP's child vaults, so a DEX at that count has the rest added from `openOrders`, without their order
  type, time in force or trigger price. Position TP/SL orders, which Hyperliquid lists without a size, take the size of
  the position they close.
+ HIP-4 outcome orders, which the first perpetual DEX lists beside spot and perpetual orders, are skipped by
  `GetActiveOrders` and `GetOrderHistory` and left open by `CancelAllOrders`, as no supported asset type covers them.
+ `GetOrderHistory` covers the account's 2000 most recent order status changes, and returns each order once, with its
  latest status and, for a triggered order, the trigger price and placement time of its earlier changes. `GetOrderInfo`
  reads only the latest status, so a trigger order that has triggered and filled is returned without its trigger price
  and dated when it triggered.
+ Delisted perpetual markets are neither tradable nor subscribable, but their orders and fills still convert.

### Balances

Unified and portfolio margin accounts hold one balance across spot and perpetuals, which is stored under spot. Other
accounts hold a balance per perpetual DEX, in that DEX's collateral token, stored under a subaccount ID of the account
address and DEX name. Portfolio margin accounts report a spot hold beside a hold that can be negative, so a spot
balance's hold is the larger of the two, from zero up to its total, and the rest of the total is free.

### Websocket

The standard channels use these Hyperliquid feeds: tickers `activeAssetCtx`, orderbooks `l2Book`, trades `trades`,
candles `candle`, account orders `orderUpdates` and account trades `userEvents`. Any other feed, such as `bbo`,
`allMids` or `webData3`, is subscribed by using its name as the channel. Market feeds take one pair, account feeds must
be authenticated, and other feeds take no asset. Subscription parameters set the feed options:

+ `dex` selects a builder-deployed perpetual DEX for `allMids`, `clearinghouseState`, `openOrders` and `twapStates`
+ `nSigFigs` and `mantissa` aggregate an orderbook, and 5 levels select the fast orderbook, which Hyperliquid pushes
  about every half second instead of every 5 seconds
+ `aggregateByTime` combines an order's partial fills on `userFills`, and `ignorePortfolioMargin` omits portfolio margin
  amounts from `spotState`

Subscriptions differing only in these parameters select separate feeds and are held side by side, except that a
market's orderbook and the account's `userFills` take one variant at a time: two orderbook variants would replace each
other's stored book, and two `userFills` variants would relay each fill twice.

Hyperliquid closes the connection, dropping every feed on it, when a subscription fails validation, so coins, DEXs and
options are validated first. Subscriptions complete when Hyperliquid acknowledges them, and are rolled back when it
rejects them or the connection drops. When a `trades`, `userFills` or `userTwapSliceFills` subscription opens,
Hyperliquid replays recent entries, and only those newer than the last entry relayed from that feed are relayed, so a
first subscription relays none and resubscribing after a reconnect relays what the replay holds of the gap. Account
trades use `userEvents`, which includes TWAP slice fills but replays nothing, so the fills it misses while disconnected
are not recovered from the websocket.

`orderUpdates` sends a row only when an order's status changes, so fills from `userEvents`, and from `userFills` and
`userTwapSliceFills` when subscribed, are also relayed to the order stream as trades of their orders; a fill both feeds
deliver is relayed there once. Once an order's fills are known to include all its executions, because its first row
showed it unfilled or a row's executed size matched its fills, each fill reports the order's executed and remaining
size, cost and average price, and its final row carries the cost and average price. Otherwise a fill carries the trade
alone, so it never lowers an executed size learnt elsewhere. Fills from a `userFills` feed with `aggregateByTime` go only
to the fills feed, as each combines an order's fills within a block under one of their trade IDs. Orders are remembered
for a day after their last row or fill, 10000 at most, and outcome (HIP-4) orders and fills are skipped.

`WsPostInfo` sends an info request, and `WsPlaceOrders`, `WsCancelOrders`, `WsCancelOrdersByClientOrderID`,
`WsModifySingleOrder` and `WsModifyOrders` send the same signed actions as their namesakes without the `Ws` prefix,
over the websocket instead of REST.

### Withdrawals

`WithdrawCryptocurrencyFunds` withdraws USDC through the Arbitrum bridge (Arbitrum Sepolia on testnet), whose fee
Hyperliquid deducts from the amount. Hyperliquid has deprecated this legacy bridge in favour of minting USDC natively
through Circle's CCTP, which the package does not wrap. Setting `InternalTransfer` sends USDC to another Hyperliquid
account instead. Both are refused while `SubAccount` is set. Bridge deposits are credited to the address that sends
them, so there are no deposit addresses, and a deposit below the bridge's 5 USDC minimum is not credited and is lost.

### Not supported

Token, market and outcome deployment, validator actions and the other platform administration actions are not wrapped.
`SendSignedAction` signs and sends any L1 action, including one the package does not wrap, and its `L1ActionOptions`
set the batch length that weighs the action against the rate limit, whether it acts for the configured subaccount or
vault, its expiry and nonce, and whether it is sent over the websocket instead of REST. The signature hashes an
action's fields in declaration order, so an action type the package does not define must declare its fields, with
`json` and `msgpack` tags, in the order Hyperliquid's SDKs send them.

### How to do REST public/private calls

When enabled through configuration, Hyperliquid is available through the `exchange.IBotExchange` interface:

```go
var h exchange.IBotExchange

for i := range bot.Exchanges {
	if bot.Exchanges[i].GetName() == "Hyperliquid" {
		h = bot.Exchanges[i]
	}
}

// Public calls
tick, err := h.UpdateTicker(ctx, pair, asset.Spot)
if err != nil {
	// Handle error
}

book, err := h.UpdateOrderbook(ctx, pair, asset.PerpetualContract)
if err != nil {
	// Handle error
}

trades, err := h.GetRecentTrades(ctx, pair, asset.PerpetualContract)
if err != nil {
	// Handle error
}

// Private calls, which need Key, and Secret for signed actions
balances, err := h.UpdateAccountBalances(ctx, asset.PerpetualContract)
if err != nil {
	// Handle error
}
```

The rest of the API is available through the package's own methods:

```go
hl, ok := h.(*hyperliquid.Exchange)
if !ok {
	// Handle error
}

funding, err := hl.GetUserFunding(ctx, &hyperliquid.UserFundingRequest{
	User:      address,
	StartTime: time.Now().Add(-24 * time.Hour),
})
if err != nil {
	// Handle error
}
```

### How to do Websocket public/private calls

Feeds are subscribed as the Websocket section describes. Orders, cancels and modifications can also be sent over the
websocket:

```go
statuses, err := hl.WsPlaceOrders(ctx, &hyperliquid.PlaceOrdersRequest{
	Orders: []hyperliquid.OrderRequest{
		{
			Asset: 0, // BTC perpetual on mainnet
			IsBuy: true,
			Price: price,
			Size:  size,
			Limit: &hyperliquid.LimitOrderType{TimeInForce: hyperliquid.TimeInForceGTC},
		},
	},
})
if err != nil {
	// Handle error
}
```

## Donations

<img src="../../docs/assets/donate.png" alt="Donate to GoCryptoTrader" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

`bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc`
