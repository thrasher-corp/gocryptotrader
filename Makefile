LDFLAGS = -ldflags "-w -s"
GCTPKG = github.com/thrasher-corp/gocryptotrader
LINTPKG = github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0
GOPATH ?= $(shell go env GOPATH)
LINTBIN = $(GOPATH)/bin/golangci-lint
GOFUMPTBIN = $(GOPATH)/bin/gofumpt
GCTLISTENPORT=9050
GCTPROFILERLISTENPORT=8085
GO_FILES_TO_FORMAT := $(shell find . -type f -name '*.go' 	-not -path "./database/models/*" 	-not -path "./vendor/*" 	-not -name "*.pb.go" 	-not -name "*.pb.gw.go")
DRIVER ?= psql
RACE_FLAG := $(if $(NO_RACE_TEST),,-race)
CONFIG_FLAG = $(if $(CONFIG),-config $(CONFIG),)

.PHONY: all lint lint_docker check test build install fmt gofumpt update_deps modernise

all: check build

lint:
	go install $(LINTPKG)
	$(LINTBIN) run --verbose

lint_docker:
	@command -v docker >/dev/null 2>&1 || (echo "Docker not found. Please install Docker to run this target." && exit 1)
	docker run --rm -t -v $(CURDIR):/app -w /app golangci/golangci-lint:v2.4.0 golangci-lint run --verbose

check: lint test

test:
	go test $(RACE_FLAG) -coverprofile=coverage.txt -covermode=atomic  ./...

build:
	go build $(LDFLAGS)

install:
	go install $(LDFLAGS)

fmt:
	gofmt -l -w -s $(GO_FILES_TO_FORMAT)

gofumpt:
	@command -v gofumpt >/dev/null 2>&1 || go install mvdan.cc/gofumpt@latest
	$(GOFUMPTBIN) -l -w $(GO_FILES_TO_FORMAT)

modernise:
	@command -v modernize >/dev/null 2>&1 || go install golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest
	@pkgs=$$(go list ./... | grep -vE '/gctrpc$$|/backtester/btrpc$$|/database/models/'); \
		modernize -test $$pkgs

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
