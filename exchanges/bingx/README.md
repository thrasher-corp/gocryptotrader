# GoCryptoTrader package Bingx

<img src="../../common/gctlogo.png" alt="GoCryptoTrader logo" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/bingx)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)

This bingx package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## BingX Exchange

### Current Features

+ Spot, USDT-M perpetual futures (`usdtmarginedfutures`, which also lists BingX's USDC-settled perpetuals) and coin-M
  perpetual futures (`coinmarginedfutures`)
+ REST market data: tickers, order books, recent and historical trades, candles, mark price candles, funding rates,
  open interest, contract details, trading rules and non-crypto trading sessions
+ Websocket market data: tickers, trades, candles and order books, with last price, book ticker and mark price channels
+ Account balances, positions, fund flows and commission rates, and account order and balance streams
+ Orders on every asset, with batch orders, cancel-replace, countdown cancellation, OCO orders on spot and TWAP orders,
  position closing and reversal, leverage, margin type, position mode and multi-assets mode on USDT-M
+ Deposits, withdrawals, transfers between accounts and sub-account management
+ Agent, copy trading, dual-currency wealth products, signals and platform activities

### How to enable

+ [Enable via configuration](../../config/README.md#enable-exchange-via-config-example)

### Subscriptions

Default subscriptions, for each enabled asset:

+ Ticker
+ All trades
+ Candles, one minute
+ Orderbook, 50 levels: each push is a complete snapshot of the top of the book; 5, 10, 20, 50 and 100 levels are
  accepted
+ My orders and my account, when authenticated websocket support is enabled

The `lastPrice` and `bookTicker` channels, and `markPrice` for perpetuals, can be subscribed as exchange-specific
channels. BingX allows 200 topics on a connection, and further topics open further connections. A subscription or
unsubscription BingX leaves unanswered retires its connection, as BingX may still hold the topic, and the websocket
reconnects and subscribes to what is wanted.

Account streams connect with a listen key, which is created through REST, extended every 30 minutes and never deleted,
as other clients of the account may use it. While listen key requests are failing the public streams connect without
the account streams, which follow once a key is available. Credentials BingX refuses turn authenticated websocket
support off. Perpetual balance events carry wallet balances alone, so a perpetual account's balances are refreshed
over REST after them, and whenever its account connection comes up, which fetches the changes made while it was down.

The spot and coin-M candle streams follow UTC+8 periods for intervals of six hours and longer, while candles fetched
over REST follow UTC periods.

### Orders

+ Spot market buys spend `QuoteAmount` and spot sells trade `Amount`.
+ Coin-M orders trade contracts: `Amount` is a contract count, and order books and tickers count contracts too.
+ BingX takes a perpetual order without a position side as one on the long side, so every perpetual order carries one.
  USDT-M orders follow the account's position mode: one-way mode trades the single position, with `ReduceOnly` as
  given, and hedge mode trades the position the order opens, or with `ReduceOnly` the one it closes. BingX has no
  coin-M position mode query, so coin-M orders use hedge mode sides until BingX refuses one for its position side; the
  order is then placed once more for the other mode, which later orders follow. Coin-M orders have no reduce only flag,
  so a `ReduceOnly` coin-M order is refused in one-way mode.
+ USDT-M orders take a `ClientOrderID` only as limit or market orders, as BingX documents, so a USDT-M order of another
  type carrying one is refused before anything is sent.
+ Orders carry no margin type: a perpetual order takes its symbol's, which `SetMarginType` sets. `SubmitOrder` refuses a
  `MarginType` other than unset, or `NoMargin` on spot, and a `ReduceOnly` spot order, before anything is sent.
+ `ModifyOrder` takes `Amount` as the order's desired total quantity, as the order manager fills it in. A USDT-M limit
  order whose amount alone changes is amended in place. Other USDT-M orders and spot orders are cancelled, and a
  replacement for the desired total is placed only once the order is closed; its order ID is returned. Only an order
  that has executed none of its quantity is replaced, as the order manager keeps the executed quantity and fee of the
  order it modifies on the replacement: one its lookup shows partly executed is refused before anything is cancelled,
  and one its cancellation shows executed places nothing and returns an error saying the order was cancelled. The
  cancellation is sent once, and one that fails is checked by looking the order up again. A cancellation that was
  refused shows it cancelled nothing, whether BingX refused it with an error code, an HTTP 4xx status without one turned
  it away, as a firewall or rate limit page does, or it was never sent, as when the rate limiter refuses a wait past the
  context's deadline: a filled order returns an error saying it executed, one closed otherwise an error saying it is no
  longer open, and one still open the cancellation's error, each placing nothing. A cancellation without an answer
  showing its outcome, such as one that timed out, one answered with an HTTP 5xx status such as a gateway timeout, or
  one BingX answered with a code reporting a busy, internal or network failure (100500, 109500, 109701 or 110500), may
  have cancelled the order, so an order closed without executing any of its quantity is replaced, one closed after
  executing any returns an error saying it executed, and one still open an error saying the outcome is unknown, as the
  cancellation may still land. A failed lookup returns both errors. Modifications of one order run one at a time,
  whether they name it by its order ID or its client order ID, so one made while another replaces the order finds it
  closed. A negative, NaN or infinite price or trigger price, an order its lookup shows closed, and a `NewClientOrderID`
  for a USDT-M order other than a limit or market order are refused before anything is cancelled. A USDT-M replacement
  of an order that triggers or trails triggers on the modification's `TriggerPriceType` when it names one other than the
  last price, and on the order's otherwise. Coin-M orders cannot be modified.
+ `CancelAllOrders` cancels every open order of the pair, or of every symbol without one. With a `Side` or `Type` it
  cancels, by order ID, the open orders `GetActiveOrders` reports with that side and type.
+ `GetHistoricTrades` is supported for spot only: BingX's USDT-M trade endpoints return their latest 1000 trades
  whatever is asked, and coin-M has no trade endpoints.
+ `GetFeeByType` uses the account's commission rates when credentials are set, and BingX's standard spot rate otherwise.

### Not supported

+ Fiat withdrawals
+ The fund flow export, which BingX returns as a spreadsheet file
+ Standard contracts, which BingX's API does not offer

## Donations

<img src="../../docs/assets/donate.png" alt="Donate to GoCryptoTrader" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

`bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc`
