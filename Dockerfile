FROM golang:1.25 as build
WORKDIR /go/src/github.com/thrasher-corp/gocryptotrader
COPY . .
RUN GO111MODULE=on go mod vendor
RUN mv -vn config_example.json config.json \
 && GOARCH=386 GOOS=linux go build . \
 && GOARCH=386 GOOS=linux go build ./cmd/gctcli \
 && mv gocryptotrader /go/bin/gocryptotrader \
 && mv gctcli /go/bin/gctcli

FROM alpine:latest
VOLUME /root/.gocryptotrader
RUN apk update && apk add --no-cache ca-certificates bash
COPY --from=build /go/bin/gocryptotrader /app/
COPY --from=build /go/bin/gctcli /app/
COPY --from=build /go/src/github.com/thrasher-corp/gocryptotrader/config.json /root/.gocryptotrader/
EXPOSE 9050-9053
ENTRYPOINT [ "/app/gocryptotrader" ]
