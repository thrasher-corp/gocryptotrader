#!/usr/bin/env bash
# misc_checks.sh — Single source of truth for miscellaneous CI checks.
# Called by .github/workflows/misc.yml and runnable locally on Linux/macOS.
# macOS compatibility: BSD grep lacks -P (PCRE), so we use perl for those patterns.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# CI detection: emit GitHub Actions annotations when running in CI.
if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
    CI_MODE=1
else
    CI_MODE=0
fi

GREP_BIN='grep'
# FORCE_GREP_FALLBACK skips GNU grep detection and uses the non-GNU fallback path.
if [[ "${FORCE_GREP_FALLBACK:-0}" == "1" ]]; then
    HAS_GNU_GREP=0
elif command -v ggrep >/dev/null 2>&1 && ggrep --version 2>/dev/null | grep -q 'GNU grep'; then
    HAS_GNU_GREP=1
    GREP_BIN='ggrep'
elif grep --version 2>/dev/null | grep -q 'GNU grep'; then
    HAS_GNU_GREP=1
else
    HAS_GNU_GREP=0
fi

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

FAILURES=0
GROUP_OPEN=0

end_group() {
    if [[ "$CI_MODE" -eq 1 && "$GROUP_OPEN" -eq 1 ]]; then
        echo "::endgroup::"
        GROUP_OPEN=0
    fi
}

trap end_group EXIT

pass() {
    if [[ "$CI_MODE" -eq 1 ]]; then
        echo "PASS: $1"
        end_group
    else
        echo -e "${GREEN}PASS${NC}: $1"
    fi
}

fail() {
    FAILURES=$((FAILURES + 1))
    if [[ "$CI_MODE" -eq 1 ]]; then
        echo "::error::$1"
        end_group
        exit 1
    else
        echo -e "${RED}FAIL${NC}: $1"
    fi
}

info() {
    if [[ "$CI_MODE" -eq 1 ]]; then
        end_group
        echo "::group::$1"
        GROUP_OPEN=1
    else
        echo -e "${YELLOW}----${NC} $1"
    fi
}

# grep_matches: shared grep engine with GNU grep on Linux and a Perl fallback
# for macOS/BSD grep. The pattern is treated as a Perl regex in the fallback,
# so callers should pass syntax compatible with the selected grep mode.
#   $1 = grep mode flag ('E' or 'P')
#   $2 = raw regex pattern
#   $3 = find -name pattern (e.g. '*.go' or '*_test.go')
#   $4 = strip leading './' from displayed paths (0 or 1)
#   $5 = search root for GNU grep ('.' or '*')
#   $6+ = optional additional find args
grep_matches() {
    local grep_mode="$1"
    local pattern="$2"
    local name_glob="$3"
    local strip_dot_prefix="${4:-0}"
    local search_root="${5:-.}"
    shift 5

    if [[ "$HAS_GNU_GREP" -eq 1 ]]; then
        local grep_targets=(.)
        if [[ "$search_root" == "*" ]]; then
            grep_targets=(*)
        fi
        "$GREP_BIN" -r -n -I --color=always --exclude-dir=.git --exclude-dir=.idea --include="$name_glob" -"$grep_mode" "$pattern" "${grep_targets[@]}" 2>/dev/null
        return $?
    fi

    local results
    results=$(find . -name "$name_glob" -not -path './.git/*' -not -path './.idea/*' "$@" -print0 2>/dev/null \
        | xargs -0 perl -e '
            my $pattern = shift @ARGV;
            my $strip_dot_prefix = shift @ARGV;
            for my $f (@ARGV) {
                open my $fh, "<", $f or next;
                my $line = 0;
                while (<$fh>) {
                    $line++;
                    my $display = $f;
                    $display =~ s{^\./}{} if $strip_dot_prefix;
                    print "$display:$line:$_" if /$pattern/;
                }
                close $fh;
            }
        ' "$pattern" "$strip_dot_prefix" 2>/dev/null) || true
    if [[ -n "$results" ]]; then
        echo "$results"
        return 0
    fi
    return 1
}

ere_grep() {
    grep_matches 'E' "$@"
}

pcre_grep() {
    local strip_dot_prefix="${3:-0}"
    local search_root='.'
    if [[ "$strip_dot_prefix" -eq 1 ]]; then
        search_root='*'
    fi
    grep_matches 'P' "$1" "$2" "$strip_dot_prefix" "$search_root"
}

# ---------------------------------------------------------------------------
# 1. currency.NewPair(BTC, USD) usage
# ---------------------------------------------------------------------------
info "Check for currency.NewPair(BTC, USD) used instead of currency.NewBTCUSD"
if ere_grep 'currency\.NewPair\(currency\.BTC, currency\.USDT?\)' '*.go' 1 '*'; then
    fail "Replace currency.NewPair(BTC, USD*) with currency.NewBTCUSD*()"
else
    pass "No currency.NewPair(BTC, USD*) misuse found"
fi

# ---------------------------------------------------------------------------
# 2. Missing postfix `f` for testify assertions
# ---------------------------------------------------------------------------
info "Check for missing postfix f format func variant for testify assertions"
if pcre_grep '(?:assert|require)\.[A-Za-z_]\w*?(?<!f)\((?:(?!fmt\.Sprintf).)*%' '*.go' 1; then
    fail 'Replace func with the `…f` func variant (e.g. Equalf, Errorf)'
else
    pass "No missing postfix f on testify format assertions"
fi

# ---------------------------------------------------------------------------
# 3. Quoted / backticked %s in format specifier strings
# ---------------------------------------------------------------------------
info "Check for quoted and backticked %s usage in format specifier strings"
if ere_grep "[\`']%s[\`']" '*.go' 0 '.'; then
    fail "Replace '%s' or \`%s\` format specifier with %q"
else
    pass "No quoted/backticked %s usage found"
fi

# ---------------------------------------------------------------------------
# 4. require… "should" / assert… "must" message consistency
# ---------------------------------------------------------------------------
info "Check for testify require/assert message consistency"
check_failed=0

echo "Checking for 'should' in require messages..."
if ere_grep 'require\.[A-Za-z0-9_]+.*"[^"]*should[^"]*"' '*.go' 0 '.'; then
    check_failed=1
fi

echo "Checking for 'must' in assert messages..."
if ere_grep 'assert\.[A-Za-z0-9_]+.*"[^"]*must[^"]*"' '*.go' 0 '.'; then
    check_failed=1
fi

if [[ "$check_failed" -eq 1 ]]; then
    fail "Replace \"should\" in require messages and \"must\" in assert messages"
else
    pass "Testify require/assert message wording is consistent"
fi

# ---------------------------------------------------------------------------
# 5. errors.Is(err, nil) in test files
# ---------------------------------------------------------------------------
info "Check for errors.Is(err, nil) usage in tests"
if ere_grep 'errors\.Is\([^,]+, nil' '*_test.go' 0 '.'; then
    fail "Replace errors.Is(err, nil) with testify equivalents"
else
    pass "No errors.Is(err, nil) in tests"
fi

# ---------------------------------------------------------------------------
# 6. !errors.Is(err, target) in test files
# ---------------------------------------------------------------------------
info "Check for !errors.Is(err, target) usage in tests"
if pcre_grep '!errors\.Is\(\s*[^,]+\s*,\s*[^)]+\s*\)' '*_test.go' 0; then
    fail "Replace !errors.Is(err, target) with testify equivalents"
else
    pass "No !errors.Is(err, target) in tests"
fi

# ---------------------------------------------------------------------------
# 7. LLM-targeted invisible Unicode characters
# ---------------------------------------------------------------------------
info "Check for LLM targeted invisible Unicode"
# Uses \p{Cf} (format), \p{Z} (separator), \p{M} (combining mark) — same as CI's grep -P.
# Uses perl -T to skip binary files (equivalent to grep -I in the original CI).
UNICODE_PATTERN='(?!\x20)[\p{Cf}\p{Z}\p{M}]'
if [[ "$HAS_GNU_GREP" -eq 1 ]]; then
    unicode_results=$("$GREP_BIN" -r -n -I --color=always --exclude-dir=.git --exclude-dir=.idea -P "$UNICODE_PATTERN" . 2>/dev/null) || true
else
    unicode_results=$(find . -not -path './.git/*' -not -path './.idea/*' -type f -print0 2>/dev/null \
        | xargs -0 perl -e '
            for my $f (@ARGV) {
                next unless -T $f;
                open my $fh, "<:encoding(UTF-8)", $f or next;
                my $line = 0;
                while (<$fh>) {
                    $line++;
                    print "$f:$line:$_" if /'"$UNICODE_PATTERN"'/;
                }
                close $fh;
            }
        ' 2>/dev/null) || true
fi
if [[ -n "$unicode_results" ]]; then
    echo "$unicode_results"
    fail "Remove zero-width/format, separator or combining-mark characters"
else
    pass "No invisible Unicode characters found"
fi

# ---------------------------------------------------------------------------
# 8. JSON config format (sorted exchanges)
# ---------------------------------------------------------------------------
info "Check configs JSON format"
if ! command -v jq &>/dev/null; then
    fail "jq is not installed - install via: brew install jq (macOS) or apt install jq (Linux)"
else
    check_failed=0
    for file in config_example.json testdata/configtest.json; do
        if [[ ! -f "$file" ]]; then
            info "Skipping $file (not found)"
            continue
        fi

        processed_file="${file%.*}_processed.${file##*.}"
        jq '.exchanges |= sort_by(.name)' --indent 1 "$file" > "$processed_file"
        if ! diff "$file" "$processed_file" >/dev/null 2>&1; then
            diff "$file" "$processed_file" || true
            echo "jq differences found in $file! Please run 'make lint_configs'"
            rm -f "$processed_file"
            check_failed=1
        else
            rm -f "$processed_file"
            echo "No differences found in $file 🌞"
        fi
    done

    if [[ "$check_failed" -eq 1 ]]; then
        fail "JSON config files are not properly sorted"
    else
        pass "JSON config files are properly formatted"
    fi
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
if [ "$FAILURES" -eq 0 ]; then
    if [[ "$CI_MODE" -eq 0 ]]; then
        echo -e "${GREEN}All miscellaneous checks passed!${NC}"
    else
        echo "::notice::All miscellaneous checks passed."
    fi
    exit 0
else
    if [[ "$CI_MODE" -eq 0 ]]; then
        echo -e "${RED}${FAILURES} check(s) failed.${NC}"
    else
        echo "${FAILURES} check(s) failed."
    fi
    exit 1
fi
