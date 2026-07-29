LDFLAGS = -ldflags "-w -s"
GCTPKG = github.com/thrasher-corp/gocryptotrader
LINTPKG = github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0
GOPATH ?= $(shell go env GOPATH)
LINTBIN = $(GOPATH)/bin/golangci-lint
GOFUMPTBIN = $(GOPATH)/bin/gofumpt
GCTLISTENPORT=9050
GCTPROFILERLISTENPORT=8085
GO_FILES_TO_FORMAT := $(shell find . -type f -name '*.go' 	-not -path "./database/models/*" 	-not -path "./vendor/*" 	-not -name "*.pb.go" 	-not -name "*.pb.gw.go")
DRIVER ?= psql
RACE_FLAG := $(if $(NO_RACE_TEST),,-race)
CONFIG_FLAG = $(if $(CONFIG),-config $(CONFIG),)

.PHONY: all lint lint_docker misc_checks check test build install fmt gofumpt update_deps bench bench_update bench_trend check_bench_pkgs bench_pkg

# Edit benchmarks/packages.txt to change which packages are benchmarked.
BENCH_PKGS = $(shell go run ./cmd/benchcheck -list)

# BENCH_FLAGS gates allocations on every PR and must stay identical between bench and bench_update
# so recorded and compared values are produced the same way. It is deliberately cheap: allocs/op is
# per-iteration and reproducible, so a longer benchtime or a higher count buys the gate nothing.
#
# -cpu 4 pins GOMAXPROCS inside each test binary so a benchmark measures the same parallelism on CI
# as on a developer's machine. It does not pin -p, the number of package binaries run concurrently,
# which still defaults to the host's core count, so a scheduler-sensitive benchmark could still vary
# between machines; -p 1 closes that gap by running one package binary at a time.
BENCH_FLAGS = -run '^$$' -bench . -benchmem -benchtime 100ms -count 6 -cpu 4 -p 1 -timeout 20m

# BENCH_TREND_FLAGS feed the scheduled ns/op history, where tight confidence intervals do matter and
# a long runtime is affordable. Never use these for the PR gate; they take over ten minutes.
BENCH_TREND_FLAGS = -run '^$$' -bench . -benchmem -benchtime 1s -count 15 -cpu 4 -p 1 -timeout 60m
BENCH_SERIES = benchmarks/series.jsonl

# go test writes to a file rather than a pipe so a failing run aborts the target. In a pipeline the
# two processes run concurrently, so benchcheck could read partial output and, with -update, save a
# pruned baseline before the shell ever saw go test's exit status.
#
# Named per target ($@): a single shared path lets `make -j2 bench bench_trend`, or two terminals,
# truncate and read the same file at once, so one run parses the other's partial output.
BENCH_OUT = .bench-output-$@.txt

all: check build

lint:
	go install $(LINTPKG)
	$(LINTBIN) run --verbose

lint_docker:
	@command -v docker >/dev/null 2>&1 || (echo "Docker not found. Please install Docker to run this target." && exit 1)
	docker run --rm -t -v $(CURDIR):/app -w /app golangci/golangci-lint:v2.9.0 golangci-lint run --verbose

misc_checks:
	bash ./scripts/misc_checks.sh

check: lint misc_checks test

# $(shell) swallows the exit status, so a benchcheck that fails to build or a missing packages.txt
# would leave BENCH_PKGS empty. go test would then benchmark the current directory instead and the
# run would fail later with a misleading "no benchmark results".
check_bench_pkgs:
	@test -n "$(BENCH_PKGS)" || { \
		echo "no benchmark packages resolved; check benchmarks/packages.txt and 'go run ./cmd/benchcheck -list'"; \
		exit 1; \
	}

# BENCHCHECK_FLAGS lets CI add -warn without restating BENCH_FLAGS, so the flags have one home.
BENCHCHECK_FLAGS ?=

bench: check_bench_pkgs
	go test $(BENCH_FLAGS) $(BENCH_PKGS) > $(BENCH_OUT)
	go run ./cmd/benchcheck $(BENCHCHECK_FLAGS) < $(BENCH_OUT)

# Measures one package with the gate's exact flags, for auditing a package before gating it.
# Usage: make bench_pkg PKG=./currency/
bench_pkg:
	@test -n "$(PKG)" || { echo "set PKG, e.g. make bench_pkg PKG=./currency/"; exit 1; }
	go test $(BENCH_FLAGS) $(PKG)

bench_update: check_bench_pkgs
	go test $(BENCH_FLAGS) $(BENCH_PKGS) > $(BENCH_OUT)
	go run ./cmd/benchcheck -update -prune < $(BENCH_OUT)

bench_trend: check_bench_pkgs
	go test $(BENCH_TREND_FLAGS) $(BENCH_PKGS) > $(BENCH_OUT)
	sha=$$(git rev-parse HEAD) && \
		go run ./cmd/benchcheck -series $(BENCH_SERIES) -commit "$$sha" -warn < $(BENCH_OUT)

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
