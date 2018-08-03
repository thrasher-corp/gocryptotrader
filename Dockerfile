FROM golang:1.10 as build
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
WORKDIR /go/src/github.com/thrasher-/gocryptotrader
COPY Gopkg.* ./
RUN dep ensure -vendor-only
COPY . .
RUN mv -vn config_example.json config.json \
 && GOARCH=386 GOOS=linux CGO_ENABLED=0 go install -v \
 && mv /go/bin/linux_386 /go/bin/gocryptotrader

FROM alpine:latest
RUN apk update && apk add --no-cache ca-certificates
COPY --from=build /go/bin/gocryptotrader /app/
COPY --from=build /go/src/github.com/thrasher-/gocryptotrader/config.json /app/
EXPOSE 9050
CMD ["/app/gocryptotrader"]
