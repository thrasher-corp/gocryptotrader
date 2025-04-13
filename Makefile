LDFLAGS = -ldflags "-w -s"
GCTPKG = github.com/thrasher-corp/gocryptotrader
LINTPKG = github.com/golangci/golangci-lint/cmd/golangci-lint@v2.0.2
LINTBIN = $(GOPATH)/bin/golangci-lint
GCTLISTENPORT=9050
GCTPROFILERLISTENPORT=8085
DRIVER ?= psql
RACE_FLAG := $(if $(NO_RACE_TEST),,-race)
CONFIG_FLAG = $(if $(CONFIG),-config $(CONFIG),)

.PHONY: get linter check test build install update_deps

all: check build

get:
	go install $(GCTPKG)

linter:
	go install $(GCTPKG)
	go install $(LINTPKG)
	test -z "$$($(LINTBIN) run --verbose | tee /dev/stderr)"

check: linter test

test:
	go test $(RACE_FLAG) -coverprofile=coverage.txt -covermode=atomic  ./...

build:
	go build $(LDFLAGS)

install:
	go install $(LDFLAGS)

fmt:
	gofmt -l -w -s $(shell find . -type f -name '*.go')

update_deps:
	go mod verify
	go mod tidy
	rm -rf vendor
	go mod vendor

.PHONY: profile_heap
profile_heap:
	go tool pprof -http "localhost:$(GCTPROFILERLISTENPORT)" 'http://localhost:$(GCTLISTENPORT)/debug/pprof/heap'

.PHONY: profile_cpu
profile_cpu:
	go tool pprof -http "localhost:$(GCTPROFILERLISTENPORT)" 'http://localhost:$(GCTLISTENPORT)/debug/pprof/profile'

.PHONY: gen_db_models
gen_db_models: target/sqlboiler.json
ifeq ($(DRIVER), psql)
	sqlboiler -c $< -o database/models/postgres -p postgres --no-auto-timestamps --wipe $(DRIVER)
else ifeq ($(DRIVER), sqlite3)
	sqlboiler -c $< -o database/models/sqlite3 -p sqlite3 --no-auto-timestamps --wipe $(DRIVER)
else
	$(error Driver '$(DRIVER)' not supported)
endif

target/sqlboiler.json:
	mkdir -p $(@D)
	go run ./cmd/gen_sqlboiler_config/main.go $(CONFIG_FLAG) -outdir $(@D)

.PHONY: lint_configs
lint_configs: check-jq
	@$(call sort-json,config_example.json)
	@$(call sort-json,testdata/configtest.json)

define sort-json
	@printf "Processing $(1)... "
	@jq '.exchanges |= sort_by(.name)' --indent 1 $(1) > $(1).temp && \
		(mv $(1).temp $(1) && printf "OK\n") || \
		(rm $(1).temp; printf "FAILED\n"; exit 1)
endef

.PHONY: check-jq
check-jq:
	@printf "Checking if jq is installed... "
	@command -v jq >/dev/null 2>&1 && { printf "OK\n"; } || { printf "FAILED. Please install jq to proceed.\n"; exit 1; }

.PHONY: sonic
sonic:
	go build $(LDFLAGS) -tags "sonic_on" 
