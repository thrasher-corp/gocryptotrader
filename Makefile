LDFLAGS = -ldflags "-w -s"
GCTPKG = github.com/thrasher-/gocryptotrader
LINTPKG = gopkg.in/alecthomas/gometalinter.v2
LINTBIN = $(GOPATH)/bin/gometalinter.v2
ENABLELLL = false
LINTOPTS = --disable-all \
           --enable=gofmt \
		   --enable=vet \
		   --enable=golint
ifeq ($(ENABLELLL), true)
LINTOPTS += --enable=lll \
			--line-length=80
endif
LINTOPTS += --deadline=5m ./... | \
		   	grep -v 'ALL_CAPS\|OP_' 2>&1 | \
		   	tee /dev/stderr

get:
	GO111MODULE=on go get $(GCTPKG)

linter:
	GO111MODULE=off go get -u $(LINTPKG)
	$(LINTBIN) --install
	$(LINTBIN) $(LINTOPTS)

check: linter test

test:
	go test -race -coverprofile=coverage.txt -covermode=atomic  ./...

build:
	GO111MODULE=on go build $(LDFLAGS)

install:
	GO111MODULE=on go install $(LDFLAGS)

fmt:
	gofmt -l -w -s $(shell find . -type f -name '*.go')