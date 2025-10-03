# GoCryptoTrader Backtester: Funding package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/funding)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This funding package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Funding package overview

### What does the funding package do?
The funding package is responsible for keeping track of all funds across all events during a backtesting run. It is backwards compatible with all existing backtesting strategies

### What is the funding manager?
The funding manager is responsible for holding all funding Items and Pairs over the course of a backtesting run. It prevents funds from being overwritten and maintains relationships of currency pairs.

Consider the following example: Exchange Level Funding is disabled and Simultaneous Processing is disabled, so each currency in the `.strat` config will execute a strategy individually. If the pairs BTC-USDT, BNB-USDT and LTC-BTC are present, then the funding manager will ensure that none of the funds are shared between each of the currencies, even if they all share base or quote values.

Conversely, Exchange level funding and Simultaneous Processing is enabled, so each currency in the `.strat` file will be processed in one step per time interval. The pairs BTC-USDT, BNB-USDT and BTC-LTC can all share the same base or quote level funds and can make complex decisions on how that funding is used, such as allowing BTC-LTC to make a purchase if an indicator is strongest for that pair.

### What is a funding Item?
A funding item holds the initial funding, current funding, reserved funding and transfer fees associated with an exchange, asset and currency. If it is a Pair, then the Item will be linked to the paired Item.

### What is a funding Pair?
A funding Pair consists of two funding Items, the Base and Quote. If Exchange Level Funding is disabled, the Base and Quote are linked to each other and the funds cannot be shared with other Pairs or Items. If Exchange Level Funding is enabled, the pair can access the same funds as every other currency that shares the exchange and asset type.

### What is a collateral Pair?
A collateral Pair consists of two funding Items, the Contract and Collateral. These are exclusive to FUTURES asset type and help track how much money there is, along with how many contract holdings there are

### What does Exchange Level Funding mean?
Exchange level funding allows funds to be shared during a backtesting run. If the strategy contains the two pairs BTC-USDT and BNB-USDT and the strategy sells 3 BTC for $100,000 USDT, then BNB-USDT can use that $100,000 USDT to make a purchase of $20,000 BNB.
It is restricted to an exchange and asset type, so BTC used in spot, cannot be used in a futures contract (futures backtesting is not currently supported). However, the funding manager can transfer funds between exchange and asset types.

Having funding at the exchange level also allows for a finer degree of control while also being more realistic for strategic execution.
A user can create a strategy with many pairs, such as BTC-USDT, LTC-BTC, DOGE-XRP and XRP-USDT, but only creating funding for USDT and still see the purchase of LTC or DOGE.
Another strategy could start with funding in the Base currency. So an RSI strategy for BTC-USDT that has 3 BTC as funding will then start by selling rather than buying.

#### Why is Simultaneous Processing a prerequisite of Exchange Level Funding?
Simultaneous Processing allows a strategy to process multiple data signals for a single time period to be processed in one step. The reason Simultaneous Processing is required for Exchange Level Funding is that if it is disabled, all events are handled in a sequence.
If any funding was to be shared in such a scenario, the first currency to be processed will always get the choice share of funding. Simultaneous Processing ensures the decision to spend funds for BTC-USDT over BNB-USDT is a measured decision, and not done by the order of currencies in a strategy config.

### Can I transfer funds from one place to another?
Yes! Though it does use some things to consider.
- It is handled at the strategy execution level, so when creating a strategy, you design the conditions in which funding may be transferred from one place to another.
  - For example, if an indicator is very strong on one exchange, but not another, you may wish to transfer funds to the strongest exchange to act upon
- It comes with the assumption that a transfer is actually possible in the candle timeframe your strategy runs on.
  - For example, a 1 minute candle strategy likely would not be able to process a transfer of funds and have another exchange use it in that timeframe. So any positive results from such a strategy may not be reflected in real-world scenarios
- You can only transfer to the same currency eg BTC from Binance to Kraken, no conversions
- You set the transfer fee in your config

### Do I need to add funding settings to my config if Exchange Level Funding is disabled?
No. The already existing `CurrencySettings` will populate the funding manager with initial funds if Exchange Level Funding is disabled.

#### Strategy Settings

| Key                                | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | Example                                                                   |
|------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------|
| name                               | The strategy to use                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            | `rsi`                                                                     |
| use-simultaneous-signal-processing | This denotes whether multiple currencies are processed simultaneously with the strategy function `OnSimultaneousSignals`. Eg If you have multiple CurrencySettings and only wish to purchase BTC-USDT when XRP-DOGE is 1337, this setting is useful as you can analyse both signal events to output a purchase call for BTC                                                                                                                                                                                                                                                                                                    | `true`                                                                    |
| disable-usd-tracking               | If `false`, will track all currencies used in your strategy against USD equivalent candles. For example, if you are running a strategy for BTC/XRP, then the GoCryptoTrader Backtester will also retrieve candles data for BTC/USD and XRP/USD to then track strategy performance against a single currency. This also tracks against USDT and other USD tracked stablecoins, so one exchange supporting USDT and another BUSD will still allow unified strategy performance analysis. If disabled, will not track against USD, this can be especially helpful when running strategies under live, database and CSV based data | `false`                                                                   |
| custom-settings                    | This is a map where you can enter custom settings for a strategy. The RSI strategy allows for customisation of the upper, lower and length variables to allow you to change them from 70, 30 and 14 respectively to 69, 36, 12                                                                                                                                                                                                                                                                                                                                                                                                 | `"custom-settings": { "rsi-high": 70, "rsi-low": 30, "rsi-period": 14 } ` |

#### Funding Config Settings

| Key                        | Description                                                                                                                                                                                                                           | Example |
|----------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| use-exchange-level-funding | Allows shared funding at an exchange asset level. You can set funding for `USDT` and all pairs that feature `USDT` will have access to those funds when making orders. See [this](/backtester/funding/README.md) for more information | `false` |
| exchange-level-funding     | An array of exchange level funding settings.  See below, or [this](/backtester/funding/README.md) for more information                                                                                                                | `[]`    |

##### Funding Item Config Settings

| Key           | Description                                                                                                                                                                                                                        | Example   |
|---------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------|
| exchange-name | The exchange to set funds. See [here](https://github.com/thrasher-corp/gocryptotrader/blob/master/README.md) for a list of supported exchanges                                                                                     | `Binance` |
| asset         | The asset type to set funds. Typically, this will be `spot`, however, see [this package](https://github.com/thrasher-corp/gocryptotrader/blob/master/exchanges/asset/asset.go) for the various asset types GoCryptoTrader supports | `spot`    |
| currency      | The currency to set funds                                                                                                                                                                                                          | `BTC`     |
| initial-funds | The initial funding for the currency                                                                                                                                                                                               | `1337`    |
| transfer-fee  | If your strategy utilises transferring of funds via the Funding Manager, this is deducted upon doing so                                                                                                                            | `0.005`   |

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
