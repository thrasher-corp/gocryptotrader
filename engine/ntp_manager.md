# GoCryptoTrader package Ntp Manager

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/engine/ntp_manager)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)


This ntp_manager package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Current Features for NTP Manager

The NTP manager compares the local clock with configured NTP servers. It is a diagnostic feature
for investigating timestamp-sensitive request failures; it does not adjust the system clock.
These settings take effect only when the engine's NTP client feature is enabled; the command-line
application enables that feature by default.

Servers are tried in configured order and the first valid response is used. The clock correction is
evaluated together with its complete root-distance interval, producing one of four results:

- `in-sync`: the complete interval is within the configured tolerances;
- `ahead`: the complete interval proves the local clock is ahead;
- `behind`: the complete interval proves the local clock is behind;
- `inconclusive`: the interval crosses a tolerance boundary, so no clock verdict is made.

At startup, the alert/warn/disable prompt is shown only for a conclusive `ahead` or `behind` result.
An inconclusive result warns without prompting and recommends a closer or lower-latency NTP server.
The interval is a diagnostic bound based on the server's reported NTP metadata, not authenticated
proof of UTC.

Periodic checks run every 30 minutes only after the `ntp_timekeeper` subsystem is explicitly
started. Normal engine startup constructs the subsystem but does not start it, including when the
startup prompt changes the setting to periodic mode.

The following settings are available under `ntpclient`:

### ntpclient

| Config | Description | Example |
| ------ | ----------- | ------- |
| enabled | `-1` disables checks, `0` checks at startup, and `1` permits periodic checks when the subsystem is explicitly started | `1` |
| pool | NTP servers attempted in configured order until the first valid response | `["0.pool.ntp.org:123","pool.ntp.org:123"]` |
| allowedDifference | Maximum permitted local-clock-behind duration, expressed as Go `time.Duration` nanoseconds | `50000000` |
| allowedNegativeDifference | Maximum permitted local-clock-ahead duration, expressed as Go `time.Duration` nanoseconds | `50000000` |

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
