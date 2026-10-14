# GoCryptoTrader package Kucoin

<img src="../../common/gctlogo.png" alt="GoCryptoTrader logo" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/kucoin)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)

This kucoin package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Kucoin Exchange

### Current Features

+ REST Support: the Classic API and the Unified Trading Account (UTA) API V2
+ Websocket Support: the Classic spot and margin, Classic futures and UTA V2 connections
+ Websocket order entry: the Classic order API (add and cancel for spot, margin and futures) and the UTA trade
  connection (add, cancel and amend). The standard order methods (`SubmitOrder`, `ModifyOrder`, `CancelOrder`,
  `CancelBatchOrders`, `CancelAllOrders`) use REST only; websocket orders are placed through the `Ws` and `WS` methods

### Subscriptions

Default Public Subscriptions:

+ Ticker for spot, margin and futures
+ Orderbook for spot, margin and futures
+ All trades for spot and margin

Futures ticker subscriptions also subscribe Ticker V1 and the symbol snapshot, which carry the last fill and the 24 hour statistics that Ticker V2 leaves out.

When authenticated websocket support is enabled, the default orderbook subscription uses the realtime spot and futures feeds, which are calibrated against Get Full OrderBook: the spot one needs authentication, the futures one is public. Without authenticated websocket support it uses the public depth-5 feeds, even when REST authentication is enabled; this includes configurations that had enabled the legacy realtime entries described below, which move from full depth to depth-5.

Legacy `/market/level2` and `/contractMarket/level2` subscription entries are removed during setup, and so are `/margin/loan` entries, whose margin lending service KuCoin has retired. Existing generic orderbook subscriptions and their explicit pair restrictions are kept, except that when a legacy entry is replaced without websocket authentication, an unrestricted authenticated generic for the same asset is made public so migrated coverage remains active. Enabled legacy entries are replaced with generic subscriptions covering the same assets and pairs; unrestricted legacy entries remain dynamic wildcards. During expansion, wildcard subscriptions exclude pairs already carried by more specific subscriptions, and shared spot and margin feeds assign each pair to one effective subscription. Explicit depth-5 and depth-50 channel subscriptions remain available and are not upgraded by authentication; the realtime upgrade leaves the pairs they carry on those channels.

Generic orderbook pairs are formatted for KuCoin requests during expansion, including explicitly configured pairs. Authenticated orderbooks that differ only in interval or depth are coalesced after those fields are cleared for the realtime feed. Duplicate orderbooks that already have the same key before normalisation remain invalid.

Default Authenticated Subscriptions:

+ Order updates for futures
+ Stop order lifecycle events for futures
+ Account balance events for spot, margin and futures
+ Margin position updates

Subscriptions are subject to enabled assets and pairs.

Margin subscriptions for ticker, orderbook and All trades are merged into Spot subscriptions because duplicates are not allowed,
unless Spot subscription does not exist, i.e. Spot asset not enabled, or subscription configured only for Margin

Limitations:

+ 100 topics per subscription request
+ 400 topics per spot and margin connection

Due to these limitations, if more than 10 spot symbols are enabled, ticker will subscribe to ticker:all.

### Unified Trading Account websocket

UTA V2 channels are configured with a `uta.` prefix on KuCoin's channel name:

+ Public: `uta.ticker`, `uta.kline`, `uta.trade`, `uta.obu`, `uta.mark-price`, `uta.funding-fee`,
  `uta.funding-fee-all-symbols`, `uta.callAuctionInfo`
+ Private: `uta.orderAll`, `uta.order`, `uta.execution.lite`, `uta.execution`, `uta.balance`, `uta.positionAll`,
  `uta.position`, `uta.lw` (liquidation warning), `uta.leverage`

The UTA connections stay down unless UTA channels are configured.

### Websocket order connections

The Classic order API and UTA trade connections need no subscriptions, so they are dialled whenever they are set up,
and they sign in with the account's keys. They are set up only when authenticated websocket support is enabled with
credentials, and the UTA trade connection only when UTA channels are configured as well.

## Donations

<img src="../../docs/assets/donate.png" alt="Donate to GoCryptoTrader" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

`bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc`
