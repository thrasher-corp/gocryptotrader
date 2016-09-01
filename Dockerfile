FROM golang
COPY config-example.json config.json
RUN go build .
EXPOSE 9050
CMD gocryptotrader

