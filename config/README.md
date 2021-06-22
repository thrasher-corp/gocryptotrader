# GoCryptoTrader package Config

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/config)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This config package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## Current Features for config

 + Handling of config encryption and verification of "configuration".json data.

 + Contains configurations for:

	- Exchanges for utilisation of a broad or minimal amount of enabled
	exchanges [Example](#enable-exchange-via-config-example) for
	enabling an exchange.

	- Bank accounts for withdrawal and depositing FIAT between exchange and
	your personal accounts [Example](#enable-bank-accounts-via-config-example).

	- Portfolio to monitor online and offline accounts [Example](#enable-portfolio-via-config-example).

	- Currency configurations to set your foreign exchange provider accounts,
	your preferred display currency, suitable FIAT currency and suitable
	cryptocurrency [Example](#enable-currency-via-config-example).

	- Communication for utilisation of supported communication mediums e.g.
	email events direct to your personal account [Example](#enable-communications-via-config-example).

# Config Examples

#### Basic examples for enabling features on the GoCryptoTrader platform

+ Linux example for quickly creating and testing configuration file
```sh
cd ~/go/src/github.com/thrasher-corp/gocryptotrader
cp config_example.json config.json
# Test config
go build
./gocryptotrader
```

+ or custom config, can also pass in absolute path to "configuration".json file.

```sh
cd ~/go/src/github.com/thrasher-corp/gocryptotrader
cp config_example.json custom.json
# Test config
go build
./gocryptotrader -config custom.json
```

## Enable Exchange Via Config Example

+ To enable or disable an exchange via config proceed through the
"configuration".json file to exchanges and to the supported exchange e.g see
below. "Enabled" set to true or false will enable and disable the exchange,
if you set "APIKey" && "APISecret" you must set "AuthenticatedAPISupport" to
true or the bot will not be able to send authenticated http requests. If needed
you can set the exchanges bank details for depositing FIAT options. Some banks
have multiple deposit accounts for different FIAT deposit currencies.

```js
"Exchanges": [
 {
  "Name": "Bitfinex",
  "Enabled": true,
  "Verbose": false,
  "Websocket": false,
  "UseSandbox": false,
  "RESTPollingDelay": 10,
  "websocketResponseCheckTimeout": 30000000,
  "websocketResponseMaxLimit": 7000000000,
  "httpTimeout": 15000000000,
  "APIKey": "Key",
  "APISecret": "Secret",
  "AvailablePairs": "ATENC_GBP,ATENC_NZD,BTC_AUD,BTC_SGD,LTC_BTC,START_GBP,...",
  "EnabledPairs": "BTC_USD,BTC_HKD,BTC_EUR,BTC_CAD,BTC_AUD,BTC_SGD,BTC_JPY,...",
  "BaseCurrencies": "USD,HKD,EUR,CAD,AUD,SGD,JPY,GBP,NZD",
  "AssetTypes": "SPOT",
  "SupportsAutoPairUpdates": true,
  "ConfigCurrencyPairFormat": {
   "Uppercase": true,
   "Delimiter": "_"
  },
  "RequestCurrencyPairFormat": {
   "Uppercase": true
  },
  "BankAccounts": [
   {
    "BankName": "",
    "BankAddress": "",
    "AccountName": "",
    "AccountNumber": "",
    "SWIFTCode": "",
    "IBAN": "",
    "SupportedCurrencies": "AUD,USD,EUR"
   }
  ]
 },
```

## Enable Bank Accounts Via Config Example

+ To enable bank accounts simply proceed through "configuration".json file to
"BankAccounts" and input your account information example below.

```js
"BankAccounts": [
 {
  "BankName": "test",
  "BankAddress": "test",
  "AccountName": "TestAccount",
  "AccountNumber": "0234",
  "SWIFTCode": "91272837",
  "IBAN": "98218738671897",
  "SupportedCurrencies": "USD",
  "SupportedExchanges": "Kraken,Bitstamp"
 }
]
```

## Enable Portfolio Via Config Example

+ To enable the GoCryptoTrader platform to monitor your addresses please
specify, "configuration".json file example below.

```js
"PortfolioAddresses": {
 "Addresses": [
  {
   "Address": "1JCe8z4jJVNXSjohjM4i9Hh813dLCNx2Sy",
   "CoinType": "BTC",
   "Balance": 53000.01310358,
   "Description": ""
  },
  {
   "Address": "3Nxwenay9Z8Lc9JBiywExpnEFiLp6Afp8v",
   "CoinType": "BTC",
   "Balance": 101848.28376405,
   "Description": ""
  }
 ]
```

## Enable Currency Via Config Example

+ To Enable foreign exchange providers set "Enabled" to true and add in your
account API keys example below.

```js
"ForexProviders": [
 {
  "Name": "CurrencyConverter",
  "Enabled": true,
  "Verbose": false,
  "RESTPollingDelay": 600,
  "APIKey": "",
  "APIKeyLvl": -1,
  "PrimaryProvider": true
 },
]
```

+ To define the cryptocurrency you want the platform to use set them here
example below.

```js
"Cryptocurrencies": "BTC,LTC,ETH,XRP,NMC,NVC,PPC,XBT,DOGE,DASH",
```

+ To define the currency you want to everything to be valued against example
below.

```js
"FiatDisplayCurrency": "USD"
```

## Enable Communications Via Config Example

+ To set the desired platform communication medium proceed to "Communications"
in the "configuration".json file and set your account details to the preferred
comm method and add in your contact list if available.

```js
"SMSGlobal": {
 "Name": "SMSGlobal",
 "Enabled": false,
 "Verbose": false,
 "Username": "Username",
 "Password": "Password",
 "Contacts": [
  {
   "Name": "Bob",
   "Number": "12345",
   "Enabled": false
  }
 ]
},
```


## Configure Network Time Server 

+ To configure and enable a NTP server you need to set the "enabled" field to one of 3 values -1 is disabled 0 is enabled and alert at start up 1 is enabled and warn at start up
servers are configured by the pool array and attempted first to last allowedDifference and allowedNegativeDifference are how far ahead and behind is acceptable for the time to be out in nanoseconds

```js
 "ntpclient": {
  "enabled": 0,
  "pool": [
   "pool.ntp.org:123"
  ],
  "allowedDifference": 0,
  "allowedNegativeDifference": 0
 },
 ```

### Please click GoDocs chevron above to view current GoDoc information for this package

## Contribution

Please feel free to submit any pull requests or suggest any desired features to be added.

When submitting a PR, please abide by our coding guidelines:

+ Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
+ Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
+ Code must adhere to our [coding style](https://github.com/thrasher-corp/gocryptotrader/blob/master/doc/coding_style.md).
+ Pull requests need to be based on and opened against the `master` branch.

## Donations

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/donate.png?raw=true" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
