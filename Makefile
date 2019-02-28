LDFLAGS = -ldflags "-w -s"
GCTPKG = github.com/thrasher-/gocryptotrader
LINTPKG = github.com/golangci/golangci-lint/cmd/golangci-lint@v1.15.0
LINTBIN = $(GOPATH)/bin/golangci-lint

get:
	GO111MODULE=on go get $(GCTPKG)

linter:
	GO111MODULE=on go get $(GCTPKG)
	GO111MODULE=on go get $(LINTPKG)
	$(LINTBIN) run --verbose | tee /dev/stderr

check: linter test

test:
	go test -race -coverprofile=coverage.txt -covermode=atomic  ./...

build:
	GO111MODULE=on go build $(LDFLAGS)

install:
	GO111MODULE=on go install $(LDFLAGS)

fmt:
	gofmt -l -w -s $(shell find . -type f -name '*.go')

update_deps:
	GO111MODULE=on go mod verify
	GO111MODULE=on go mod tidy
	rm -rf vendor
	GO111MODULE=on go mod vendor