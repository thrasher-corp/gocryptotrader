// Package mexc_proto_types holds the protobuf types of MEXC's spot websocket push data.
//
// The .proto files beside this one are MEXC's, from github.com/mexcdevelop/websocket-proto at commit
// 0c9c4f35dd0fadc3a46a350e909a93379d81e811, with a go_package option added and formatted by buf format.
// The Go code is generated from them with protoc v29.3 and protoc-gen-go v1.36.6; to update the types,
// replace the .proto files and run go generate in this directory.
//
// The .proto sources are Apache-2.0 licensed by MEXC; see https://github.com/mexcdevelop/websocket-proto/blob/main/LICENSE
package mexc_proto_types //nolint:staticcheck // ST1003: the name is the one the generated code carries

//go:generate protoc --go_out=. --go_opt=paths=source_relative PrivateAccountV3Api.proto PrivateDealsV3Api.proto PrivateOrdersV3Api.proto PublicAggreBookTickerV3Api.proto PublicAggreDealsV3Api.proto PublicAggreDepthsV3Api.proto PublicBookTickerBatchV3Api.proto PublicBookTickerV3Api.proto PublicDealsV3Api.proto PublicIncreaseDepthsBatchV3Api.proto PublicIncreaseDepthsV3Api.proto PublicLimitDepthsV3Api.proto PublicMiniTickerV3Api.proto PublicMiniTickersV3Api.proto PublicSpotKlineV3Api.proto PushDataV3ApiWrapper.proto
