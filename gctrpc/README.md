# GoCryptoTrader gRPC Service

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

Then use `go get -u` to download the following packages:

```bash
go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
go get -u github.com/golang/protobuf/protoc-gen-go
```

This will place three binaries in your `$GOBIN`;

* `protoc-gen-grpc-gateway`
* `protoc-gen-swagger`
* `protoc-gen-go`

Make sure that your `$GOBIN` is in your `$PATH`.

## Usage

After the above depenancies are required, make necessary changes to the `rpc.proto`
spec file and run the generation scripts:

### Windows

Run `gen_pb_win.bat`

### Linux and macOS

Run `./gen_pb_linux.sh`
