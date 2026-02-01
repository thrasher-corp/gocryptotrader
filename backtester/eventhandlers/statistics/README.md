# GoCryptoTrader Backtester: Statistics package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This statistics package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Statistics package overview

The statistics package is used for storing all relevant data over the course of a GoCryptoTrader Backtesting run. All types of events are tracked by exchange, asset and currency pair.
When multiple currencies are included in your strategy, the statistics package will be able to calculate which exchange asset currency pair has performed the best, along with the biggest drop downs in the market.

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
| Calmar ratio |  It is a function of the fund's average compounded annual rate of return versus its maximum drawdown. The higher the Calmar ratio, the better it performed on a risk-adjusted basis during the given time frame, which is mostly commonly set at 36 months | 3.0 to 5.0 |
| Information ratio| It is a measurement of portfolio returns beyond the returns of a benchmark, usually an index, compared to the volatility of those returns. The ratio is often used as a measure of a portfolio manager's level of skill and ability to generate excess returns relative to a benchmark | 0.40-0.60. Any positive number means that it has beaten the benchmark |
| Sharpe ratio | The Sharpe Ratio is a financial metric often used by investors when assessing the performance of investment management products and professionals. It consists of taking the excess return of the portfolio, relative to the risk-free rate, and dividing it by the standard deviation of the portfolio's excess returns | Any Sharpe ratio greater than 1.0 is good. Higher than 2.0 is very good. 3.0 or higher is excellent. Under 1.0 is sub-optimal |
| Sortino ratio | The Sortino ratio measures the risk-adjusted return of an investment asset, portfolio, or strategy. It is a modification of the Sharpe ratio but penalizes only those returns falling below a user-specified target or required rate of return, while the Sharpe ratio penalizes both upside and downside volatility equally | The higher the better, but > 2 is considered good |
| Compound annual growth rate | Compound annual growth rate is the rate of return that would be required for an investment to grow from its beginning balance to its ending balance, assuming the profits were reinvested at the end of each year of the investment’s lifespan | Any positive number |

## Arithmetic or versus geometric?
Both! We calculate ratios where an average is required using both types. The reasoning for using either is debated by finance and mathematicians. [This](https://www.investopedia.com/ask/answers/06/geometricmean.asp) is a good breakdown of both, but here is an extra simple table

| Average type | A reason to use it |
| ------------ | ------------------ |
| Arithmetic | The arithmetic mean is the average of a sum of numbers, which reflects the central tendency of the position of the numbers |
| Geometric | The geometric mean differs from the arithmetic average, or arithmetic mean, in how it is calculated because it takes into account the compounding that occurs from period to period. Because of this, investors usually consider the geometric mean a more accurate measure of returns than the arithmetic mean |

## USD total tracking
If the strategy config setting `DisableUSDTracking` is `false`, then the GoCryptoTrader Backtester will automatically retrieve USD data that matches your backtesting currencies, eg pair BTC/LTC will track BTC/USD and LTC/USD as well. This allows for tracking overall strategic performance against one currency. This can allow for much easier performance calculations and comparisons

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
