FROM golang:1.12 as build
WORKDIR /go/src/github.com/thrasher-/gocryptotrader
COPY . .
RUN GO111MODULE=on go mod vendor
RUN mv -vn config_example.json config.json \
 && GOARCH=386 GOOS=linux CGO_ENABLED=0 go build . \
 && mv gocryptotrader /go/bin/gocryptotrader

FROM alpine:latest
RUN apk update && apk add --no-cache ca-certificates
COPY --from=build /go/bin/gocryptotrader /app/
COPY --from=build /go/src/github.com/thrasher-/gocryptotrader/config.json /app/
EXPOSE 9050-9053
CMD ["/app/gocryptotrader"]
