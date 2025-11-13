# GoCryptoTrader Backtester: Btrpc package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/btrpc)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This btrpc package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Btrpc overview


The GoCryptoTrader Backtester utilises gRPC for client/server interaction. Authentication is done
by a self signed TLS cert, which only supports connections from localhost and also
through basic authorisation specified by the users config file.

The GoCryptoTrader Backtester also supports a gRPC JSON proxy service for applications which can
be toggled on or off depending on the users preference. This can be found in your config file
under `grpcProxyEnabled` `grpcProxyListenAddress`. See `btrpc.swagger.json` for endpoint definitions

## Installation

The GoCryptoTrader Backtester requires a local installation of the Google protocol buffers
compiler `protoc` v3.0.0 or above. Please install this via your local package
manager or by downloading one of the releases from the official repository:

[protoc releases](https://github.com/protocolbuffers/protobuf/releases)

Then use `go install` to download the following packages:

```bash
go install \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
    google.golang.org/protobuf/cmd/protoc-gen-go \
    google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

This will place the following binaries in your `$GOBIN`;

* `protoc-gen-grpc-gateway`
* `protoc-gen-openapiv2`
* `protoc-gen-go`
* `protoc-gen-go-grpc`

Make sure that your `$GOBIN` is in your `$PATH`.

### Linux / macOS / Windows

The GoCryptoTrader Backtester requires a local installation of the `buf` cli tool that tries to make Protobuf handling more easier and reliable,
after [installation](https://docs.buf.build/installation) you'll need to run:

```shell
buf mod update
```

After previous command, make necessary changes to the `rpc.proto` spec file and run the generation command:

```shell
buf generate
```

If any changes were made, ensure that the `rpc.proto` file is formatted correctly by using `buf format -w`

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
