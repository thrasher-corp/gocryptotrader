# GoCryptoTrader Backtester: Currencystatistics package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatistics)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This currencystatistics package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## Currencystatistics package overview

Currency Statistics is an important package to verify the effectiveness of your strategies.
It can calculate the following:
- Calmar ratio
- Information ratio
- Sharpe ratio
- Sortino ratio
- CAGR
- Drawdowns, both the biggest and longest
- Whether the strategy outperformed the market
- If the strategy made a profit

## Ratios

| Ratio | Description | A good range |
| ----- | ----------- | ------------ |
| Calmar ratio |  It is a function of the fund's average compounded annual rate of return versus its maximum drawdown. The higher the Calmar ratio, the better it performed on a risk-adjusted basis during the given time frame, which is mostly commonly set at 36 months. | 3.0 to 5.0 |
| Information ratio| It is a measurement of portfolio returns beyond the returns of a benchmark, usually an index, compared to the volatility of those returns. The ratio is often used as a measure of a portfolio manager's level of skill and ability to generate excess returns relative to a benchmark | 0.40-0.60. Any positive number means that it has beaten the benchmark |
| Sharpe ratio | The Sharpe Ratio is a financial metric often used by investors when assessing the performance of investment management products and professionals. It consists of taking the excess return of the portfolio, relative to the risk-free rate, and dividing it by the standard deviation of the portfolio's excess returns | Any Sharpe ratio greater than 1.0 is good. Higher than 2.0 is very good. 3.0 or higher is excellent. Under 1.0 is sub-optimal |
| Sortino ratio | The Sortino ratio measures the risk-adjusted return of an investment asset, portfolio, or strategy. It is a modification of the Sharpe ratio but penalizes only those returns falling below a user-specified target or required rate of return, while the Sharpe ratio penalizes both upside and downside volatility equally. | The higher the better, but > 2 is considered good. |
| Compound annual growth rate | Compound annual growth rate is the rate of return that would be required for an investment to grow from its beginning balance to its ending balance, assuming the profits were reinvested at the end of each year of the investment’s lifespan | Any positive number |

## Arithmetic or versus geometric?
Both! We calculate ratios where an average is required using both types. The reasoning for using either is debated by finance and mathematicians. [This](https://www.investopedia.com/ask/answers/06/geometricmean.asp) is a good breakdown of both, but here is an extra simple table

| Average type | A reason to use it |
| ------------ | ------------------ |
| Arithmetic | The arithmetic mean is the average of a sum of numbers, which reflects the central tendency of the position of the numbers |
| Geometric | The geometric mean differs from the arithmetic average, or arithmetic mean, in how it is calculated because it takes into account the compounding that occurs from period to period. Because of this, investors usually consider the geometric mean a more accurate measure of returns than the arithmetic mean. |



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
