LDFLAGS = -ldflags "-w -s"
GCTPKG = github.com/thrasher-corp/gocryptotrader
GOLANGCI_LINT_VERSION := $(strip $(shell cat .golangci-lint-version))
LINTPKG = github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
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

.PHONY: all lint lint_docker markdownlint workflow_lint proto proto_check misc_checks check test build install fmt gofumpt update_deps sonic udecimal decimal_bench decimal_bench_shopspring decimal_bench_udecimal bench bench_update bench_apply bench_pkg bench_compare check_bench_pkgs check_bench_env check_bench_goflags

# Edit benchmarks/packages.txt to change which packages are benchmarked.
BENCH_PKGS = $(shell $(BENCH_ENV) go run ./cmd/benchcheck -list)

# BENCH_ENV and BENCH_FLAGS must stay identical between bench and bench_update. BENCH_SAMPLES must be
# odd, since budgets compare the median, and -cpu a single value above 1. -v reports the skips a
# prune needs to see. go test's -timeout does not apply to benchmarks, so the workflow bounds the run.
BENCH_SAMPLES = 7
BENCH_FLAGS = -run '^$$' -bench . -benchmem -benchtime 100ms -count $(BENCH_SAMPLES) -cpu 4 -p 1 -v
# BENCH_ENV clears what changes a measurement without showing in go test's output, and what can make go
# run exit zero without running benchcheck, such as GOFLAGS=-exec=/bin/true.
BENCH_ENV = GOFLAGS= GOEXPERIMENT= GOGC= GOMEMLIMIT=

all: check build

lint:
	go install $(LINTPKG)
	$(LINTBIN) run --verbose

lint_docker:
	@command -v docker >/dev/null 2>&1 || (echo "Docker not found. Please install Docker to run this target." && exit 1)
	docker run --rm -t -v $(CURDIR):/app -w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run --verbose

misc_checks:
	bash ./scripts/misc_checks.sh

markdownlint:
	@if ! command -v npx >/dev/null 2>&1; then \
		echo "npx not found: install Node.js to run Markdown lint" >&2; exit 1; \
	fi; \
	npx --yes markdownlint-cli2@0.23.2 "**/*.md" "cmd/documentation/**/*.tmpl"

# zizmor needs --config: it looks for its config from the nearest .git directory,
# which from a linked worktree, whose .git is a file, can be an unrelated repository
workflow_lint:
	@for tool in shellcheck pipx; do \
		if ! command -v $$tool >/dev/null 2>&1; then \
			echo "$$tool not found: install it to run workflow lint" >&2; exit 1; \
		fi; \
	done; \
	go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 && \
	shellcheck --severity=warning scripts/*.sh .devcontainer/*.sh && \
	pipx run zizmor==1.30.1 --offline --format plain --config .github/zizmor.yml .github/workflows

proto:
	bash ./scripts/proto.sh

proto_check:
	bash ./scripts/proto.sh --check

check: lint misc_checks markdownlint workflow_lint proto_check test

# $(shell) ignores the exit status of BENCH_PKGS's -list, and an empty BENCH_PKGS benchmarks the
# current directory.
check_bench_pkgs: check_bench_env
	@$(BENCH_ENV) go run ./cmd/benchcheck -list > /dev/null

# Refuses a Go release other than go.mod's major and minor version, which go test's output does not show.
check_bench_env: check_bench_goflags
	@want=go$$(awk '$$1 == "go" { split($$2, v, "."); print v[1] "." v[2]; exit }' go.mod); \
		have=$$($(BENCH_ENV) go env GOVERSION); \
		case "$$have" in "$$want" | "$$want".*) ;; \
		*) echo "budgets are measured with $$want, as go.mod names, but go is $$have; select a $$want release with GOTOOLCHAIN before measuring" >&2; exit 1 ;; \
		esac

# BENCH_ENV cannot clear GOFLAGS or GOEXPERIMENT set with 'go env -w'.
check_bench_goflags:
	@if [ -n "$$($(BENCH_ENV) go env GOFLAGS GOEXPERIMENT | tr -d '\n')" ]; then \
		echo "GOFLAGS or GOEXPERIMENT is set in $$(go env GOENV); remove it with 'go env -u GOFLAGS GOEXPERIMENT'" >&2; \
		exit 1; \
	fi

# go test writes to a file, not a pipe, so its failure stops the target before benchcheck reads partial
# output. The file is kept and named when anything fails, and BENCH_OUT keeps it at that path. Paths
# are quoted, so a leading ~/ is expanded by hand.
bench: check_bench_pkgs
	out="$(or $(BENCH_OUT),.bench-output-$@-$$$$.txt)"; case $$out in "~/"*) out="$$HOME/$${out#"~/"}";; esac; \
		{ $(BENCH_ENV) go test $(BENCH_FLAGS) $(BENCH_PKGS) > "$$out" || \
			{ cat "$$out"; echo "go test failed; its output is kept in $$out" >&2; exit 1; }; } && \
		{ $(BENCH_ENV) go run ./cmd/benchcheck -samples $(BENCH_SAMPLES) < "$$out" || \
			{ echo "go test's output is kept in $$out" >&2; exit 1; }; }$(if $(BENCH_OUT),, && rm -f "$$out")

# Measures one package with the gate's flags, for auditing it before listing it.
# Usage: make bench_pkg PKG=./currency/
bench_pkg: check_bench_env
	@test -n "$(PKG)" || { echo "set PKG, e.g. make bench_pkg PKG=./currency/"; exit 1; }
	$(BENCH_ENV) go test $(BENCH_FLAGS) $(PKG)

bench_update: check_bench_pkgs
	out="$(or $(BENCH_OUT),.bench-output-$@-$$$$.txt)"; case $$out in "~/"*) out="$$HOME/$${out#"~/"}";; esac; \
		{ $(BENCH_ENV) go test $(BENCH_FLAGS) $(BENCH_PKGS) > "$$out" || \
			{ cat "$$out"; echo "go test failed; its output is kept in $$out" >&2; exit 1; }; } && \
		{ $(BENCH_ENV) go run ./cmd/benchcheck -samples $(BENCH_SAMPLES) -update -prune < "$$out" || \
			{ echo "go test's output is kept in $$out" >&2; exit 1; }; }$(if $(BENCH_OUT),, && rm -f "$$out")

# Folds output already measured, such as CI's bench-output artifact, into the baseline, which is how a
# machine other than linux/amd64 updates it.
# Usage: make bench_apply BENCH_OUT=path/to/bench-output.txt
bench_apply: check_bench_goflags
	@test -n "$(BENCH_OUT)" || { echo "set BENCH_OUT, e.g. make bench_apply BENCH_OUT=bench-output.txt"; exit 1; }
	out="$(BENCH_OUT)"; case $$out in "~/"*) out="$$HOME/$${out#"~/"}";; esac; \
		$(BENCH_ENV) go run ./cmd/benchcheck -samples $(BENCH_SAMPLES) -update -prune < "$$out"

# Measures BASE and the working tree in alternating rounds, one sample per process, and reports their
# timings and allocations side by side, saving each revision's raw output and the report in
# BENCH_COMPARE_OUT. It is advisory: only a failure to measure fails it.
# Usage: make bench_compare BASE=upstream/master
BENCH_COMPARE_OUT ?= .bench-compare
bench_compare: check_bench_pkgs
	@test -n "$(BASE)" || { echo "set BASE to the commit to compare with, e.g. make bench_compare BASE=upstream/master"; exit 1; }
	out="$(BENCH_COMPARE_OUT)"; case $$out in "~/"*) out="$$HOME/$${out#"~/"}";; esac; \
		$(BENCH_ENV) go run ./cmd/benchcheck -compare "$(BASE)" -samples $(BENCH_SAMPLES) -out "$$out" -- $(BENCH_FLAGS)

test:
	go test $(RACE_FLAG) -count=1 -coverprofile=coverage.txt -covermode=atomic  ./...

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
