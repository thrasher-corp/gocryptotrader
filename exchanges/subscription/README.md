# GoCryptoTrader package Subscription

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/subscription)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This subscription package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

# Exchange Subscriptions

Exchange Subscriptions are streams of data delivered via websocket.

GoCryptoTrader engine will subscribe automatically to configured channels.
A subset of exchanges currently support user configured channels, with the remaining using hardcoded defaults.
See configuration Features.Subscriptions for whether an exchange is configurable.

## Templating

Exchange Contributors should implement `GetSubscriptionTemplate` to return a text/template Template.

Exchanges are free to implement template caching, a map or a mono-template, inline or file templates.

The template is provided with a single context structure:
```go
  S              *subscription.Subscription
  AssetPairs     map[asset.Item]currency.Pairs
  AssetSeparator string
  PairSeparator  string
  BatchSize      string
```

Subscriptions may fan out many channels for assets and pairs, to support exchanges which require individual subscriptions.  
To allow the template to communicate how to handle its output it should use the provided directives:
- AssetSeparator should be added at the end of each section related to assets
- PairSeparator should be added at the end of each pair
- BatchSize should be added with a number directly before AssetSeparator to indicate pairs have been batched

Example:
```
{{- range $asset, $pairs := $.AssetPairs }}
    {{- range $b := batch $pairs 30 -}}
        {{- $.S.Channel -}} : {{- $b.Join -}}
        {{ $.PairSeparator }}
    {{- end -}}
    {{- $.BatchSize -}} 30
    {{- $.AssetSeparator }}
{{- end }}
```

Assets and pairs should be output in the sequence in AssetPairs since text/template range function uses an sorted order for map keys.

Template functions may modify AssetPairs to update the subscription's pairs, e.g. Filtering out margin pairs already in spot subscription.

We use separators like this because it allows mono-templates to decide at runtime whether to fan out.

See exchanges/subscription/testdata/subscriptions.tmpl for an example mono-template showcasing various features.

Templates do not need to worry about joining around separators; Trailing separators will be stripped automatically.

Template functions should panic to handle errors. They are caught by text/template and turned into errors for use in `subscription.expandTemplate`.


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
