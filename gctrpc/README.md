# GoCryptoTrader gRPC Service

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/page-logo.png?raw=true" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)

A cryptocurrency trading bot supporting multiple exchanges written in Golang.

**Please note that this bot is under development and is not ready for production!**

## Community

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## Background

GoCryptoTrader utilises gRPC for client/server interaction. Authentication is done
by a self signed TLS cert, which only supports connections from localhost and also
through basic authorisation specified by the users config file.

GoCryptoTrader also supports a gRPC JSON proxy service for applications which can
be toggled on or off depending on the users preference.

## Installation

GoCryptoTrader requires a local installation of the Google protocol buffers
compiler `protoc` v3.0.0 or above. Please install this via your local package
manager or by downloading one of the releases from the official repository:

[protoc releases](https://github.com/protocolbuffers/protobuf/releases)

Then use `go get` to download the following packages:

```bash
go get \
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

### Linux / macOS

GoCryptoTrader requires a local installation of the `buf` cli tool that tries to make Protobuf handling more easier and reliable,
after [installation](https://docs.buf.build/installation) you'll need to run:

```shell
buf beta mod update
```

After previous command, make necessary changes to the `rpc.proto` spec file and run the generation command:

```shell
buf generate
```

### Windows

After the above dependencies are required, make necessary changes to the `rpc.proto`
spec file and run the generation scripts:

Run `gen_pb_win.bat`

