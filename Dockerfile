FROM golang
COPY config-example.json config.json
RUN go build .
CMD gocryptotrader

