LDFLAGS = -ldflags "-w -s"
GCTPKG = github.com/thrasher-corp/gocryptotrader
LINTPKG = github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2
GOPATH ?= $(shell go env GOPATH)
LINTBIN = $(GOPATH)/bin/golangci-lint
GOFUMPTBIN = $(GOPATH)/bin/gofumpt
GCTLISTENPORT=9050
GCTPROFILERLISTENPORT=8085
GO_FILES_TO_FORMAT := $(shell find . -type f -name '*.go' 	-not -path "./database/models/*" 	-not -path "./vendor/*" 	-not -name "*.pb.go" 	-not -name "*.pb.gw.go")
DRIVER ?= psql
RACE_FLAG := $(if $(NO_RACE_TEST),,-race)
CONFIG_FLAG = $(if $(CONFIG),-config $(CONFIG),)
DECIMAL_BENCH_COUNT ?= 5
DECIMAL_BENCH_TIME ?= 500ms
DECIMAL_BENCH_FLAGS = -run '^$$' -bench . -benchmem -benchtime $(DECIMAL_BENCH_TIME) -count $(DECIMAL_BENCH_COUNT)

.PHONY: all lint lint_docker markdownlint misc_checks check test build install fmt gofumpt update_deps sonic udecimal decimal_bench decimal_bench_shopspring decimal_bench_udecimal bench bench_update bench_apply bench_pkg check_bench_pkgs

# Edit benchmarks/packages.txt to change which packages are benchmarked.
BENCH_PKGS = $(shell go run ./cmd/benchcheck -list)

# BENCH_FLAGS must stay identical between bench and bench_update, so recorded and compared values
# are produced the same way.
#
# BENCH_SAMPLES is the -count, and benchcheck's -samples, which rejects any benchmark reporting a
# different number. It must be odd: benchcheck compares the median sample and refuses an even
# count, which has no middle. -cpu and -p pin what a scheduler-sensitive benchmark measures, so CI
# and a developer's machine agree; keep -cpu single valued, or `-cpu 1,4` doubles every count.
BENCH_SAMPLES = 7
BENCH_FLAGS = -run '^$$' -bench . -benchmem -benchtime 100ms -count $(BENCH_SAMPLES) -cpu 4 -p 1 -timeout 20m

all: check build

lint:
	go install $(LINTPKG)
	$(LINTBIN) run --verbose

lint_docker:
	@command -v docker >/dev/null 2>&1 || (echo "Docker not found. Please install Docker to run this target." && exit 1)
	docker run --rm -t -v $(CURDIR):/app -w /app golangci/golangci-lint:v2.13.2 golangci-lint run --verbose

misc_checks:
	bash ./scripts/misc_checks.sh

markdownlint:
	@if ! command -v npx >/dev/null 2>&1; then \
		if [ -n "$$CI" ]; then echo "npx not found: Markdown lint cannot run in CI"; exit 1; fi; \
		echo "npx not found: skipping Markdown lint, which CI still runs"; exit 0; \
	fi; \
	npx --yes markdownlint-cli2@0.23.2 "**/*.md" "cmd/documentation/**/*.tmpl"

check: lint misc_checks markdownlint test

# Runs the same -list BENCH_PKGS does, where its error can be seen: $(shell) swallows the message and
# the exit status alike, and an empty BENCH_PKGS makes go test benchmark the current directory.
check_bench_pkgs:
	@go run ./cmd/benchcheck -list > /dev/null

# go test writes to a file rather than a pipe so a failing run aborts the target; through a pipe
# benchcheck reads partial output and, with -update, saves a baseline before go test's exit status
# is known. The PID keeps two terminals in one checkout off the same file, which is removed on
# success and left for inspection on failure. Setting BENCH_OUT keeps it at that path instead, which
# is how CI uploads it. .gitignore covers both.
bench: check_bench_pkgs
	out=$(or $(BENCH_OUT),.bench-output-$@-$$$$.txt) && \
		go test $(BENCH_FLAGS) $(BENCH_PKGS) > $$out && \
		go run ./cmd/benchcheck -samples $(BENCH_SAMPLES) < $$out$(if $(BENCH_OUT),, && rm -f $$out)

# Measures one package with the gate's exact flags, for auditing a package before listing it.
# Usage: make bench_pkg PKG=./currency/
bench_pkg:
	@test -n "$(PKG)" || { echo "set PKG, e.g. make bench_pkg PKG=./currency/"; exit 1; }
	go test $(BENCH_FLAGS) $(PKG)

bench_update: check_bench_pkgs
	out=$(or $(BENCH_OUT),.bench-output-$@-$$$$.txt) && \
		go test $(BENCH_FLAGS) $(BENCH_PKGS) > $$out && \
		go run ./cmd/benchcheck -samples $(BENCH_SAMPLES) -update -prune < $$out$(if $(BENCH_OUT),, && rm -f $$out)

# Folds output that has already been measured into the baseline, such as the bench-output artifact
# the benchmarks workflow uploads. benchcheck only records budgets from linux/amd64 output, so this
# is how a baseline is updated from any other machine.
# Usage: make bench_apply BENCH_OUT=path/to/bench-output.txt
bench_apply:
	@test -n "$(BENCH_OUT)" || { echo "set BENCH_OUT, e.g. make bench_apply BENCH_OUT=bench-output.txt"; exit 1; }
	go run ./cmd/benchcheck -samples $(BENCH_SAMPLES) -update -prune < $(BENCH_OUT)

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

sonic:
	go build $(LDFLAGS) -tags "sonic_on" 

udecimal:
	go build $(LDFLAGS) -tags "udecimal_on"

decimal_bench: decimal_bench_shopspring decimal_bench_udecimal

decimal_bench_shopspring:
	@printf '\nshopspring/decimal backend\n'
	go test ./types/decimal $(DECIMAL_BENCH_FLAGS)

decimal_bench_udecimal:
	@printf '\nquagmt/udecimal backend\n'
	go test -tags "udecimal_on" ./types/decimal $(DECIMAL_BENCH_FLAGS)
