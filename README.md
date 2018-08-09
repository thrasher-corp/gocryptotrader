<img src="https://github.com/thrasher-/gocryptotrader/blob/master/web/src/assets/page-logo.png?raw=true" width="350px" height="350px" hspace="70">

[![Build Status](https://travis-ci.org/thrasher-/gocryptotrader.svg?branch=master)](https://travis-ci.org/thrasher-/gocryptotrader)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-/gocryptotrader)
[![Coverage Status](http://codecov.io/github/thrasher-/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-/gocryptotrader)

A cryptocurrency trading bot supporting multiple exchanges written in Golang.

**Please note that this bot is under development and is not ready for production!**

## Community

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://gocryptotrader.herokuapp.com/)

## Exchange Support Table

| Exchange | REST API | Streaming API | FIX API |
|----------|------|-----------|-----|
| Alphapoint | Yes  | Yes        | NA  |
| ANXPRO | Yes  | No        | NA  |
| Binance| Yes  | Yes        | NA  |
| Bitfinex | Yes  | Yes        | NA  |
| Bitflyer | Yes  | No      | NA  |
| Bithumb | Yes  | NA       | NA  |
| BitMEX | Yes | No | NA |
| Bitstamp | Yes  | Yes       | No  |
| Bittrex | Yes | No | NA |
| BTCC | Yes  | Yes     | No  |
| BTCMarkets | Yes | No       | NA  |
| COINUT | Yes | No | NA |
| Exmo | Yes | NA | NA |
| CoinbasePro | Yes | Yes | No|
| GateIO | Yes | No | NA |
| Gemini | Yes | No | No |
| HitBTC | Yes | Yes | No |
| Huobi.Pro | Yes | No | NA |
| Huobi.Hadax | Yes | No | NA |
| ItBit | Yes | NA | No |
| Kraken | Yes | NA | NA |
| LakeBTC | Yes | No | NA |
| Liqui | Yes | No | NA |
| LocalBitcoins | Yes | NA | NA |
| OKCoin China | Yes | Yes | No |
| OKCoin International | Yes | Yes | No |
| OKEX | Yes | No | No |
| Poloniex | Yes | Yes | NA |
| WEX     | Yes  | NA        | NA  |
| Yobit | Yes | NA | NA |
| ZB.COM | Yes | No | NA |

We are aiming to support the top 20 highest volume exchanges based off the [CoinMarketCap exchange data](https://coinmarketcap.com/exchanges/volume/24-hour/).

** NA means not applicable as the Exchange does not support the feature.

## Current Features

+ Support for all Exchange fiat and digital currencies, with the ability to individually toggle them on/off.
+ AES256 encrypted config file.
+ REST API support for all exchanges.
+ Websocket support for applicable exchanges.
+ Ability to turn off/on certain exchanges.
+ Ability to adjust manual polling timer for exchanges.
+ Communication packages (Slack, SMS via SMSGlobal, Telegram and SMTP)
+ HTTP rate limiter package.
+ Forex currency converter packages (CurrencyConverterAPI, CurrencyLayer, Fixer.io, OpenExchangeRates)
+ Packages for handling currency pairs, tickers and orderbooks.
+ Portfolio management tool; fetches balances from supported exchanges and allows for custom address tracking.
+ Basic event trigger system.
+ WebGUI.

## Planned Features

Planned features can be found on our [community Trello page](https://trello.com/b/ZAhMhpOy/gocryptotrader).

## Contribution

Please feel free to submit any pull requests or suggest any desired features to be added.

When submitting a PR, please abide by our coding guidelines:

+ Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
+ Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
+ Code must adhere to our [coding style](https://github.com/thrasher-/gocryptotrader/blob/master/doc/coding_style.md).
+ Pull requests need to be based on and opened against the `master` branch.

## Compiling instructions

Download and install Go from [Go Downloads](https://golang.org/dl/) for your
platform.

### Linux/OSX

We use the `dep` tool provided by Golang for managing dependencies. As it is not officially part
of the go tools package suite, you will need to manually install it if you have not already.

On MacOS you can install or upgrade to the latest released version with Homebrew:

```sh
brew install dep
brew upgrade dep
```

On linux or MacOS, you can also install it via `go get`:

```sh
go get -u github.com/golang/dep/cmd/dep
```

After `dep` is installed, please follow the instructions below:

```bash
go get github.com/thrasher-/gocryptotrader
cd $GOPATH/src/github.com/thrasher-/gocryptotrader
make get
make install
cp $GOPATH/src/github.com/thrasher-/gocryptotrader/config_example.json $GOPATH/bin/config.json
```

### Windows

```bash
go get github.com/thrasher-/gocryptotrader
cd %GOPATH%\src\github.com\thrasher-\gocryptotrader
go install
copy %GOPATH%\src\github.com\thrasher-\gocryptotrader\config_example.json %GOPATH%\bin\config.json
```

+ Make any neccessary changes to the `config.json` file.
+ Run the `gocryptotrader` binary file inside your GOPATH bin folder.

## Donations

<img src="https://github.com/thrasher-/gocryptotrader/blob/master/web/src/assets/donate.png?raw=true" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB***

## Binaries

Binaries will be published once the codebase reaches a stable condition.

## Contributor List

### A very special thank you to all who have contributed to this program:

|User|Github|Contribution Amount|
|--|--|--|
| thrasher- | https://github.com/thrasher- | 456 |
| shazbert | https://github.com/shazbert | 142 |
| gloriousCode | https://github.com/gloriousCode | 122 |
| 140am | https://github.com/140am | 8 |
| ermalguni | https://github.com/ermalguni | 4 |
| marcofranssen | https://github.com/marcofranssen | 4 |
| Betazoid | https://github.com/Betazoid | 4 |
| crackcomm | https://github.com/crackcomm | 3 |
| bretep | https://github.com/bretep | 2 |
| gam-phon | https://github.com/gam-phon | 2 |
| cornelk | https://github.com/cornelk | 2 |
| if1live | https://github.com/if1live | 2 |
| daniel-cohen | https://github.com/daniel-cohen | 1 |
| frankzougc | https://github.com/frankzougc | 1 |
| starit | https://github.com/starit | 1 |
| Jimexist | https://github.com/Jimexist | 1 |
| lookfirst | https://github.com/lookfirst | 1 |
| zeldrinn | https://github.com/zeldrinn | 1 |
| mattkanwisher | https://github.com/mattkanwisher | 1 |
| mKurrels | https://github.com/mKurrels | 1 |
| m1kola | https://github.com/m1kola | 1 |
| cavapoo2 | https://github.com/cavapoo2 | 1 |
| tongxiaofeng | https://github.com/tongxiaofeng | 1 |
| idealhack | https://github.com/idealhack | 1 |
| vyloy | https://github.com/vyloy | 1 |
| askew- | https://github.com/askew- | 1 |
| whilei | https://github.com/whilei | 1 |
| snipesjr | https://github.com/snipesjr | 1 |
