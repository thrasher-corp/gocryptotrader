get:
	dep ensure

build:
	go build -o bin/gocryptotrader .

install:
	go install