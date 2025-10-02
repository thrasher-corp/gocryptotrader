# GoCryptoTrader Backtester: Config package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/config)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This config package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Config package overview
This readme contains details for both the GoCryptoTrader Backtester config structure along with the strategy config structure

## GoCryptoTrader Backtester Config overview
Below are the details for the GoCryptoTrader Backtester _application_ config. Strategy config overview is below this section

| Key                     | Description                                                                                                                                              | Example                          |
|-------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------|
| print-logo              | Whether to print the GoCryptoTrader Backtester logo on startup. Recommended because it looks good                                                        | `true`                           |
| verbose                 | Whether to receive verbose output. If running a GRPC server, it outputs to the server, not to the client                                                 | `false`                          |
| log-subheaders          | Whether log output contains a descriptor of what area the log is coming from, for example `STRATEGY`. Helpful for debugging                              | `true`                           |
| stop-all-tasks-on-close | When closing the application, the Backtester will attempt to stop all active tasks                                                                       | `true`                           |
| plugin-path             | When using custom strategy plugins, you can enter the path here to automatically load the plugin                                                         | `true`                           |
| report                  | Contains details on the output report after a successful backtesting run                                                                                 | See Report table below           |
| grpc                    | Contains GRPC server details                                                                                                                             | See GRPC table below             |
| use-cmd-colours         | If enabled, will output pretty colours of your choosing when running the application                                                                     | `true`                           |
| cmd-colours             | Contains details on what the colour definitions are                                                                                                      | See Colours table below          |

### Backtester Config Report overview

| Key            | Description                                                          | Example                         |
|----------------|----------------------------------------------------------------------|---------------------------------|
| output-report  | Whether or not to output a report after a successful backtesting run | `true`                          |
| template-path  | The path for the template to use when generating a report            | `/backtester/report/tpl.gohtml` |
| output-path    | The path where report output is saved                                | `/backtester/results`           |
| dark-mode      | Whether or not the report defaults to using dark mode                | `true`                          |

### Backtester Config GRPC overview

| Key                    | Description                                                                                                                                 | Example                        |
|------------------------|---------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------|
| username               | Your username to negotiate a successful connection with the server                                                                          | `rpcuser`                      |
| password               | Your password to negotiate a successful connection with the server                                                                          | `helloImTheDefaultPassword`    |
| enabled                | Whether the server is enabled. Setting this to `false` and `SingleRun` to `false` would be inadvisable                                      | `true`                         |
| listenAddress          | The listen address for the GRPC server                                                                                                      | `localhost:9054`               |
| grpcProxyEnabled       | If enabled, creates a proxy server to interact with the GRPC server via HTTP commands                                                       | `true`                         |
| grpcProxyListenAddress | The address for the proxy to listen on                                                                                                      | `localhost:9053`               |
| tls-dir                | The directory for holding your TLS certifications to make connections to the server. Will be generated by default on startup if not present | `/backtester/config/location/` |


### Backtester Config Colours overview

| Key      | Description                                                         | Example        |
|----------|---------------------------------------------------------------------|----------------|
| Default  | The colour definition for default text output                       |`[0m`        |
| Green    | The colour definition for when green is warranted, such as the logo |`[38;5;157m` |
| White    | The colour definition for when white is warranted such as the logo  |`[38;5;255m` |
| Grey     | The colour definition for grey                                      | `[38;5;240m`|
| DarkGrey | The colour definition for dark grey                                 | `[38;5;243m`|
| H1       | The colour definition for main headers                              | `[38;5;33m` |
| H2       | The colour definition for sub headers                               | `[38;5;39m` |
| H3       | The colour definition for sub sub headers                           | `[38;5;45m` |
| H4       | The colour definition for sub sub sub headers                       | `[38;5;51m` |
| Success  | The colour definition for successful operations                     | `[38;5;40m` |
| Info     | The colour definition for when informing you of something           | `[32m`      |
| Debug    | The colour definition for debug output such as verbose              | `[34m`      |
| Warn     | The colour definition for when a warning occurs                     | `[33m`      |
| Error    | The colour definition for when an error occurs                      | `[38;5;196m`|


## Strategy Config overview

### What does the config package do?
The config package contains a set of structs which allow for the customisation of the GoCryptoTrader Backtester when running.
The GoCryptoTrader Backtester runs from reading config files (`.strat` files by default under `/examples`).


### What does Simultaneous Processing mean?
GoCryptoTrader Backtester config files may contain multiple `ExchangeSettings` which defined exchange, asset and currency pairs to iterate through a period of time.

If there are multiple entries to `ExchangeSettings` and SimultaneousProcessing is disabled, then each individual exchange, asset and currency pair candle event is evaluated individually and does not know about other exchange, asset and currency pair data events. It is a way to test a singular strategy against multiple assets simultaneously. But it isn't defined as Simultaneous Processing
Simultaneous Signal Processing is a setting which allows multiple `ExchangeSettings` data events for a candle event to be considered simultaneously. This means that you can check if the price of BTC-USDT is 5% greater on Binance than it is on Kraken and choose to make signal a BUY event for Kraken and not Binance.

It allows for complex strategical decisions to be made when you consider the scope of the entire market at a given time, rather than in a vacuum when SimultaneousSignalProcessing is disabled.

### How do I customise the GoCryptoTrader Backtester?
See below for a set of tables and fields, expected values and what they can do

#### Config

| Key                | Description                                                                                                                                                                                                                                    |
|--------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| nickname           | A nickname for the specific config. When running multiple variants of the same strategy, use the nickname to help differentiate between runs                                                                                                   |
| goal               | A description of what you would hope the outcome to be. When verifying output, you can review and confirm whether the strategy met that goal                                                                                                   |
| strategy-settings  | Select which strategy to run, what custom settings to load and whether the strategy can assess multiple currencies at once to make more in-depth decisions                                                                                     |
| funding-settings   | Defines whether individual funding settings can be used. Defines the funding exchange, asset, currencies at an individual level                                                                                                                |
| currency-settings  | Currency settings is an array of settings for each individual currency you wish to run the strategy against                                                                                                                                    |
| data-settings      | Holds data retrieval settings. Determines how the GoCryptoTraderBacktester will fetch data and in what format                                                                                                                                  |
| portfolio-settings | Contains a list of global rules for the portfolio manager. CurrencySettings contain their own rules on things like how big a position is allowable, the portfolio manager rules are the same, but override any individual currency's settings  |
| statistic-settings | Contains settings that impact statistics calculation. Such as the risk-free rate for the sharpe ratio                                                                                                                                          |

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

#### Currency Settings

| Key                          | Description                                                                                                                                                                                                                                                            | Example                         |
|------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------|
| exchange-name                | The exchange to load. See [here](https://github.com/thrasher-corp/gocryptotrader/blob/master/README.md) for a list of supported exchanges                                                                                                                              | `Binance`                       |
| asset                        | The asset type. Typically, this will be `spot`, however, see [this package](https://github.com/thrasher-corp/gocryptotrader/blob/master/exchanges/asset/asset.go) for the various asset types GoCryptoTrader supports                                                  | `spot`                          |
| base                         | The base of a currency                                                                                                                                                                                                                                                 | `BTC`                           |
| quote                        | The quote of a currency                                                                                                                                                                                                                                                | `USDT`                          |
| spot-details                 | An optional field which contains initial funding data for SPOT currency pairs                                                                                                                                                                                          | See SpotSettings table below    |
| future-detailss              | An optional field which contains leverage data for FUTURES currency pairs                                                                                                                                                                                              | See FuturesSettings table below |
| buy-side                     | This struct defines the buying side rules this specific currency setting must abide by such as maximum purchase amount                                                                                                                                                 |--                               |
| sell-side                    | This struct defines the selling side rules this specific currency setting must abide by such as maximum selling amount                                                                                                                                                 |--                               |
| min-slippage-percent         | Is the lower bounds in a random number generated that make purchases more expensive, or sell events less valuable. If this value is 90, then the most a price can be affected is 10%                                                                                   | `90`                            |
| max-slippage-percent         | Is the upper bounds in a random number generated that make purchases more expensive, or sell events less valuable. If this value is 99, then the least a price can be affected is 1%. Set both upper and lower to 100 to have no randomness applied to purchase events | `100`                           |
| maker-fee-override           | The fee to use when sizing and purchasing currency. If `nil`, will lookup an exchange's fee details                                                                                                                                                                    | `0.001`                         |
| taker-fee-override           | Unused fee for when an order is placed in the orderbook, rather than taken from the orderbook. If `nil`, will lookup an exchange's fee details                                                                                                                         | `0.002`                         |
| maximum-holdings-ratio       | When multiple currency settings are used, you may set a maximum holdings ratio to prevent having too large a stake in a single currency                                                                                                                                | `0.5`                           |
| skip-candle-volume-fitting   | When placing orders, by default the BackTester will shrink an order's size to fit the candle data's volume so as to not rewrite history. Set this to `true` to ignore this and to set order size at what the portfolio manager prescribes                              | `false`                         |
| use-exchange-order-limits    | Will lookup exchange rules around purchase sizing eg minimum order increments of 0.0005. Note: Will retrieve up-to-date rules which may not have existed for the data you are using. Best to use this when considering to use this strategy live                       | `false`                         |
| use-exchange-pnl-calculation | Instead of simulating the exchange's own way of calculating PNL, use a default method which calculates the value of an asset                                                                                                                                           | `false`                         |

##### SpotSettings

| Key                 | Description                                                                                                                                                | Example |
|---------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| initial-base-funds  | The funds that the GoCryptoTraderBacktester has for the base currency. This is only required if the strategy setting `UseExchangeLevelFunding` is `false`  | `2`     |
| initial-quote-funds | The funds that the GoCryptoTraderBacktester has for the quote currency. This is only required if the strategy setting `UseExchangeLevelFunding` is `false` | `10000` |

##### FuturesSettings

| Key      | Description                                                                              | Example |
|----------|------------------------------------------------------------------------------------------|---------|
| leverage | This struct defines the leverage rules that this specific currency setting must abide by | `1`     |

### DataSettings
| Key                       | Description                                                                                            | Example       |
|---------------------------|--------------------------------------------------------------------------------------------------------|---------------|
| interval                  | The candle interval in `time.Duration` format eg set as`15000000000` for a value of `time.Second * 15` | `15000000000` |
| data-type                 | Choose whether `candle` or `trade` data is used. If trades are used, they will be converted to candles | `trade`       |
| verbose-exchange-requests | When retrieving candle data from an exchange, print verbose request/response details                   | `false`       |
| api-data                  | Holds API data settings. See table `APIData`                                                           |               |
| database-data             | Holds database data settings. See table `DatabaseData`                                                 |               |
| live-data                 | Holds API data settings. See table `LiveData`                                                          |               |
| csv-data                  | Holds CSV data settings. See table `CSVData`                                                           |               |

#### APIData

| Key                | Description                                                                                                                                                                                                | Example                     |
|--------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------|
| start-date         | The start date to retrieve data                                                                                                                                                                            | `2021-01-23T11:00:00+11:00` |
| end-date           | The end date to retrieve data                                                                                                                                                                              | `2021-01-24T11:00:00+11:00` |
| inclusive-end-date | When enabled, the end date's candle is included in the results. ie `2021-01-24T11:00:00+11:00` with a one hour candle, the final candle will be `2021-01-24T11:00:00+11:00` to `2021-01-24T12:00:00+11:00` | `false`                     |

#### CSVData

| Key       | Description      | Example                  |
|-----------|------------------|--------------------------|
| full-path | The file to load | `/data/exchangelist.csv` |

#### DatabaseData

| Key                | Description                                                                                                                                                                                                | Example                     |
|--------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------|
| start-date         | The start date to retrieve data                                                                                                                                                                            | `2021-01-23T11:00:00+11:00` |
| end-date           | The end date to retrieve data                                                                                                                                                                              | `2021-01-24T11:00:00+11:00` |
| config             | This is the same struct used as your GoCryptoTrader database config. See below tables for breakdown                                                                                                        | `see below`                 |
| path               | If using SQLite, the path to the directory, not the file. Leaving blank will use GoCryptoTrader's default database path                                                                                    | ``                          |
| inclusive-end-date | When enabled, the end date's candle is included in the results. ie `2021-01-24T11:00:00+11:00` with a one hour candle, the final candle will be `2021-01-24T11:00:00+11:00` to `2021-01-24T12:00:00+11:00` | `false`                     |

##### database

| Config            | Description                                                                | Example  |
|-------------------|----------------------------------------------------------------------------|----------|
| enabled           | Enabled or disables the database connection subsystem                      | `true`   |
| verbose           | Displays more information to the logger which can be helpful for debugging | `false`  |
| driver            | The SQL driver to use. Can be `postgres` or `sqlite`                       | `sqlite` |
| connectionDetails | See below                                                                  |          |

##### connectionDetails

| Config   | Description                                                     | Example       |
|----------|-----------------------------------------------------------------|---------------|
| host     | The host address of the database                                | `localhost`   |
| port     | The port used to connect to the database                        | `5432`        |
| username | An optional username to connect to the database                 | `username`    |
| password | An optional password to connect to the database                 | `password`    |
| database | The name of the database                                        | `database.db` |
| sslmode  | The connection type of the database for Postgres databases only | `disable`     |

#### LiveData

| Key                          | Description                                                                                                                                     | Example       |
|------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------|---------------|
| new-event-timeout            | The time allowed to wait for new data before exiting the strategy. Ensures new data is always coming in                                         | `60000000000` |
| data-check-timer             | The interval in which to check exchange API's for new data                                                                                      | `1000000000`  |
| real-orders                  | Whether to place real orders with real money. Its likely you should never want to set this to true                                              | `false`       |
| close-positions-on-stop      | As live trading doesn't stop until you tell it to, you can trigger a close of your position(s) when you stop the strategy                       | `true`        |
| data-request-retry-tolerance | Rather than immediately closing a strategy on failure to retrieve candle data, having a retry tolerance allows multiple attempts to return data | `3`           |
| data-request-retry-wait-time | How long to wait in between request retries                                                                                                     | `500000000`   |
| exchange-credentials         | A list of exchange credentials. See table named `ExchangeCredentials`                                                                           |               |

##### ExchangeCredentials Settings

| Key         | Description                                               | Example   |
|-------------|-----------------------------------------------------------|-----------|
| exchange    | The exchange to apply credentials to                      | `binance` |
| credentials | The API credentials to use. See table named `Credentials` |           |

##### Credentials Settings

| Key             | Description                                                                                 | Example      |
|-----------------|---------------------------------------------------------------------------------------------|--------------|
| Key             | Will set the GoCryptoTrader exchange to use the following API Key                           | `1234`       |
| Secret          | Will set the GoCryptoTrader exchange to use the following API Secret                        | `5678`       |
| ClientID        | Will set the GoCryptoTrader exchange to use the following API Client ID                     | `9012`       |
| PEMKey          | Private key for certain API requests. If you don't know it, you probably don't need it      | `hello-moto` |
| SubAccount      | Will set the GoCryptoTrader exchange to use the following subaccount on supported exchanges | `subzero`    |
| OneTimePassword | Will set the GoCryptoTrader exchange to use the following 2FA seed                          | `subzero`    |

#### PortfolioSettings

| Key       | Description                                                                                                            |
|-----------|------------------------------------------------------------------------------------------------------------------------|
| leverage  | This struct defines the leverage rules that this specific currency setting must abide by                               |
| buy-side  | This struct defines the buying side rules this specific currency setting must abide by such as maximum purchase amount |
| sell-side | This struct defines the selling side rules this specific currency setting must abide by such as maximum selling amount |

##### Leverage Settings

| Key                                | Description                   | Example |
|------------------------------------|-------------------------------|---------|
| can-use-leverage                   | Allows the use of leverage    | `false` |
| maximum-orders-with-leverage-ratio | currently unused              | `0.5`   |
| maximum-leverage-rate              | currently unused              | `100`   |
| maximum-collateral-leverage-rate   | currently unused              | `100`   |

##### Buy/Sell Settings

| Key           | Description                                                                                                      | Example |
|---------------|------------------------------------------------------------------------------------------------------------------|---------|
| minimum-size  | If the order's quantity is below this, the order cannot be placed                                                | `0.1`   |
| maximum-size  | If the order's quantity is over this amount, it cannot be placed and will be reduced to the maximum amount       | `10`    |
| maximum-total | If the order's price * amount exceeds this number, the order cannot be placed and will be reduced to this figure | `1337`  |


#### StatisticsSettings

| Key            | Description                                                             | Example |
|----------------|-------------------------------------------------------------------------|---------|
| risk-free-rate | The risk free rate used in the calculation of sharpe and sortino ratios | `0.03`  |

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
