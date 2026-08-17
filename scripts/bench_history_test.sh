#!/usr/bin/env bash
# bench_history_test.sh — exercises scripts/bench_history.sh against a throwaway local repository.
#
# The publish sequence is the part of the trend job most likely to break and the part a unit test
# cannot reach, so it is tested here against real git rather than only by the scheduled run.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HISTORY_SH="$SCRIPT_DIR/bench_history.sh"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
FAILURES=0

pass() { echo -e "  ${GREEN}PASS${NC}: $1"; }
fail() { echo -e "  ${RED}FAIL${NC}: $1"; FAILURES=$((FAILURES + 1)); }
info() { echo -e "${YELLOW}----${NC} $1"; }

check() { # check <description> <expected> <actual>
    if [[ "$2" == "$3" ]]; then pass "$1"; else fail "$1 (expected '$2', got '$3')"; fi
}

# must runs a step that is expected to succeed and fails the suite if it does not. Without it a
# publish that pushes correctly and then exits non-zero still satisfies the content assertions that
# follow, so the harness would report success for a run the workflow would have failed. Silent on
# success: these are preconditions, not the properties under test.
must() { # must <description> <command...>
    local desc="$1" rc=0
    shift
    "$@" >/dev/null 2>&1 || rc=$?
    [[ "$rc" -eq 0 ]] || fail "$desc (exited $rc)"
}

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

export BENCH_HISTORY_REMOTE="$TMP/remote.git"
export BENCH_HISTORY_BRANCH="benchmarks-history"
export BENCH_HISTORY_PATH="benchmarks/series.jsonl"
export GIT_AUTHOR_NAME="test"; export GIT_AUTHOR_EMAIL="test@example.com"
export GIT_COMMITTER_NAME="test"; export GIT_COMMITTER_EMAIL="test@example.com"

git init --quiet --bare "$BENCH_HISTORY_REMOTE"

# A working repository standing in for the checked-out project. It deliberately carries tracked
# files and a .gitignore covering the series path, because the real repository has both: an empty
# fixture has no tree to leak onto the history branch and no ignore rule to trip over, so the
# orphan-index cleanup and the forced add could both be deleted with every check still passing.
WORK="$TMP/work"
git init --quiet -b master "$WORK"
mkdir -p "$WORK/cmd/thing"
echo "package main" > "$WORK/cmd/thing/main.go"
echo "# project" > "$WORK/README.md"
printf 'benchmarks/series.jsonl\n' > "$WORK/.gitignore"
git -C "$WORK" add -A
git -C "$WORK" commit --quiet -m "initial"
# Every publish below runs against the current directory. A failed cd would leave them operating on
# the real repository, creating worktrees and branches in it, so this must be fatal.
cd "$WORK" || { echo "cannot cd to $WORK" >&2; exit 1; }

record() { echo "{\"ts\":\"2026-07-29T0$1:00:00Z\",\"bench\":\"BenchmarkX\",\"ns_median\":$2}"; }

# ---------------------------------------------------------------------------
info "1. first run creates the branch"
# ---------------------------------------------------------------------------
must "initial fetch" "$HISTORY_SH" fetch series.jsonl
check "fetch with no branch yields an empty series" "0" "$(wc -l < series.jsonl | tr -d ' ')"
record 1 100 >> series.jsonl
must "seeding publish" "$HISTORY_SH" publish series.jsonl
check "branch now exists" "1" "$(git ls-remote --heads "$BENCH_HISTORY_REMOTE" "$BENCH_HISTORY_BRANCH" | wc -l | tr -d ' ')"
check "one record published" "1" "$(git -C "$BENCH_HISTORY_REMOTE" show "$BENCH_HISTORY_BRANCH:$BENCH_HISTORY_PATH" | wc -l | tr -d ' ')"
# Without the orphan-index cleanup the whole master tree rides along onto the history branch, and
# without the forced add the ignored series file is refused and nothing is published at all.
check "history branch holds only the series file" "$BENCH_HISTORY_PATH" \
    "$(git -C "$BENCH_HISTORY_REMOTE" ls-tree -r --name-only "$BENCH_HISTORY_BRANCH" | tr '\n' ' ' | sed 's/ $//')"

# ---------------------------------------------------------------------------
info "2. second run appends and preserves commit history"
# ---------------------------------------------------------------------------
rm -f series.jsonl
must "second fetch" "$HISTORY_SH" fetch series.jsonl
check "fetch returns the existing record" "1" "$(wc -l < series.jsonl | tr -d ' ')"
record 2 90 >> series.jsonl
must "appending publish" "$HISTORY_SH" publish series.jsonl
check "both records present" "2" "$(git -C "$BENCH_HISTORY_REMOTE" show "$BENCH_HISTORY_BRANCH:$BENCH_HISTORY_PATH" | wc -l | tr -d ' ')"
check "history has two commits, not a re-orphaned root" "2" "$(git -C "$BENCH_HISTORY_REMOTE" rev-list --count "$BENCH_HISTORY_BRANCH")"
check "earliest record survived" "1" "$(git -C "$BENCH_HISTORY_REMOTE" show "$BENCH_HISTORY_BRANCH:$BENCH_HISTORY_PATH" | grep -c 'T01:')"

# ---------------------------------------------------------------------------
info "3. a series built from a stale fetch is refused, not silently published"
# ---------------------------------------------------------------------------
# publish re-fetches before committing, so a stale run's push is still a clean fast-forward and git
# raises nothing. The danger is therefore not a rejected push but an accepted one: the file is
# copied wholesale, so records the stale run never saw would be dropped without any error.
STALE="$TMP/stale"
git clone --quiet "$WORK" "$STALE"
# Series as it would look to a run that fetched before record 2 landed
printf '%s\n%s\n' "$(record 1 100)" "$(record 9 999)" > "$STALE/series.jsonl"
( cd "$STALE" && "$HISTORY_SH" publish series.jsonl ) >/dev/null 2>&1
stale_rc=$?
check "stale publish exits non-zero" "1" "$([[ $stale_rc -ne 0 ]] && echo 1 || echo 0)"
published="$(git -C "$BENCH_HISTORY_REMOTE" show "$BENCH_HISTORY_BRANCH:$BENCH_HISTORY_PATH")"
check "remote still holds both real records" "2" "$(echo "$published" | wc -l | tr -d ' ')"
check "the record the stale run never saw survived" "1" "$(echo "$published" | grep -c 'T02:')"
check "the stale run's record never landed" "0" "$(echo "$published" | grep -c 'T09:')"

# A run that legitimately extends the published history is still accepted
rm -f series.jsonl && must "fetch before genuine append" "$HISTORY_SH" fetch series.jsonl
record 3 80 >> series.jsonl
must "genuine append publish" "$HISTORY_SH" publish series.jsonl
check "a genuine append is still accepted" "3" "$(git -C "$BENCH_HISTORY_REMOTE" show "$BENCH_HISTORY_BRANCH:$BENCH_HISTORY_PATH" | wc -l | tr -d ' ')"

# ---------------------------------------------------------------------------
info "4. an unreachable remote is fatal, not treated as a first run"
# ---------------------------------------------------------------------------
out="$(BENCH_HISTORY_REMOTE="$TMP/does-not-exist.git" "$HISTORY_SH" fetch probe.jsonl 2>&1)"
probe_rc=$?
check "fetch against a broken remote exits non-zero" "1" "$([[ $probe_rc -ne 0 ]] && echo 1 || echo 0)"
check "and says so rather than reporting a new history" "1" "$(echo "$out" | grep -c 'could not determine')"

# ---------------------------------------------------------------------------
info "4b. a history branch without the series file reads as an empty history"
# ---------------------------------------------------------------------------
# publish already tolerates this, so fetch must too; otherwise the scheduled job aborts here on
# every run until someone fixes the branch by hand.
git init --quiet --bare "$TMP/nopath.git"
NP="$TMP/nopath"
git init --quiet -b master "$NP"
git -C "$NP" commit --quiet --allow-empty -m init
git -C "$NP" checkout --quiet --orphan h
echo placeholder > "$NP/README"
git -C "$NP" add README
git -C "$NP" commit --quiet -m "branch without the series path"
git -C "$NP" push --quiet "$TMP/nopath.git" h:h
# Back to master: git refuses to fetch a branch that is currently checked out, which the real job
# never hits because it runs from master
git -C "$NP" checkout --quiet master
nopath_rc=0
( cd "$NP" && BENCH_HISTORY_REMOTE="$TMP/nopath.git" BENCH_HISTORY_BRANCH=h \
    "$HISTORY_SH" fetch fetched.jsonl ) >/dev/null 2>&1 || nopath_rc=$?
check "fetch succeeds against a branch with no series file" "0" "$nopath_rc"
check "and yields an empty series" "0" "$(wc -c < "$NP/fetched.jsonl" | tr -d ' ')"

# ---------------------------------------------------------------------------
info "5. an empty series is refused"
# ---------------------------------------------------------------------------
: > empty.jsonl
empty_rc=0
"$HISTORY_SH" publish empty.jsonl >/dev/null 2>&1 || empty_rc=$?
check "publishing an empty series exits non-zero" "1" "$([[ $empty_rc -ne 0 ]] && echo 1 || echo 0)"

# Against an existing history the prefix check rejects an empty series anyway, so that alone does
# not exercise the guard. A first publish has nothing to compare against, and is the only case where
# the guard is what stops an empty history being created.
#
# A distinct branch name is essential: reusing BENCH_HISTORY_BRANCH would hit the local branch left
# by the earlier tests, and `checkout --orphan` would refuse on the name collision instead. The test
# would then pass whether or not the guard existed.
git init --quiet --bare "$TMP/firstrun.git"
: > first.jsonl
first_rc=0
BENCH_HISTORY_REMOTE="$TMP/firstrun.git" BENCH_HISTORY_BRANCH=firstrun \
    "$HISTORY_SH" publish first.jsonl >/dev/null 2>&1 || first_rc=$?
check "an empty first publish exits non-zero" "1" "$([[ $first_rc -ne 0 ]] && echo 1 || echo 0)"
check "and creates no history branch" "0" \
    "$(git ls-remote --heads "$TMP/firstrun.git" firstrun | wc -l | tr -d ' ')"

# ---------------------------------------------------------------------------
info "6. republishing identical records is a no-op"
# ---------------------------------------------------------------------------
before="$(git -C "$BENCH_HISTORY_REMOTE" rev-list --count "$BENCH_HISTORY_BRANCH")"
rm -f series.jsonl && must "fetch before no-op publish" "$HISTORY_SH" fetch series.jsonl
must "no-op publish" "$HISTORY_SH" publish series.jsonl
check "no empty commit is created" "$before" "$(git -C "$BENCH_HISTORY_REMOTE" rev-list --count "$BENCH_HISTORY_BRANCH")"

# ---------------------------------------------------------------------------
info "7. a published file with no trailing newline still accepts an append"
# ---------------------------------------------------------------------------
# Comparing by line count rather than bytes silently miscounts such a file and would reject every
# subsequent append forever, so this pins the prefix comparison to bytes.
# Two records with no newline after the last: `wc -l` reports 1 where the file really holds 2, so a
# line-based prefix check compares the wrong amount and rejects the append. A single record would
# not discriminate, because `wc -l` returns 0 there and any `> 0` guard skips the check entirely.
NL="$TMP/nonewline"
git init --quiet --bare "$TMP/nl.git"
git init --quiet -b master "$NL"
git -C "$NL" commit --quiet --allow-empty -m init
printf '%s\n%s' "$(record 1 100)" "$(record 2 90)" > "$NL/series.jsonl"
( cd "$NL" && BENCH_HISTORY_REMOTE="$TMP/nl.git" BENCH_HISTORY_BRANCH=h "$HISTORY_SH" publish series.jsonl ) >/dev/null 2>&1
check "seeding a newline-less history succeeds" "0" "$?"

printf '%s\n%s\n%s\n' "$(record 1 100)" "$(record 2 90)" "$(record 3 80)" > "$NL/series.jsonl"
( cd "$NL" && BENCH_HISTORY_REMOTE="$TMP/nl.git" BENCH_HISTORY_BRANCH=h "$HISTORY_SH" publish series.jsonl ) >/dev/null 2>&1
check "append onto a newline-less history is accepted" "0" "$?"
check "and all three records are stored" "3" "$(git -C "$TMP/nl.git" show "h:$BENCH_HISTORY_PATH" | wc -l | tr -d ' ')"

# ---------------------------------------------------------------------------
info "8. a truncated series is refused"
# ---------------------------------------------------------------------------
printf '%s\n' "$(record 1 100)" > "$WORK/truncated.jsonl"
"$HISTORY_SH" publish truncated.jsonl >/dev/null 2>&1
trunc_rc=$?
check "publishing fewer records than are published fails" "1" "$([[ $trunc_rc -ne 0 ]] && echo 1 || echo 0)"
check "the full history survives truncation" "3" "$(git -C "$BENCH_HISTORY_REMOTE" show "$BENCH_HISTORY_BRANCH:$BENCH_HISTORY_PATH" | wc -l | tr -d ' ')"

# ---------------------------------------------------------------------------
info "9. a run that loses the race between the prefix check and the push is rejected"
# ---------------------------------------------------------------------------
# A pre-push hook advances the remote after assert_extends_published has already passed, which is
# the one window the prefix check cannot cover.
#
# Note what this does and does not prove. git plans the ref update before running the hook, so the
# remote moving underneath fails git's own ref-lock compare-and-swap ("incorrect old value
# provided"). That rejection happens with or without --force, because --force overrides the
# fast-forward check and not the CAS. So this pins the end-to-end property — a run that loses the
# race does not clobber the winner — but it does not exercise the choice of an unforced push.
# Check 10 covers that separately.
cat > "$WORK/.git/hooks/pre-push" <<HOOK
#!/usr/bin/env bash
# git exports GIT_DIR and friends into hooks, which would silently redirect the nested git commands
# below onto the pushing repository instead of the clone
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_PREFIX
[[ -e "$TMP/raced" ]] && exit 0
: > "$TMP/raced"
d="\$(mktemp -d)"
git clone -q --branch "$BENCH_HISTORY_BRANCH" "$BENCH_HISTORY_REMOTE" "\$d"
echo '{"ts":"2026-07-29T08:00:00Z","bench":"BenchmarkRace"}' >> "\$d/$BENCH_HISTORY_PATH"
git -C "\$d" add -A
git -C "\$d" -c user.name=race -c user.email=race@example.com commit -qm "competing run"
git -C "\$d" push -q origin "$BENCH_HISTORY_BRANCH"
rm -rf "\$d"
exit 0
HOOK
chmod +x "$WORK/.git/hooks/pre-push"

rm -f series.jsonl && must "fetch before race" "$HISTORY_SH" fetch series.jsonl
record 4 70 >> series.jsonl
"$HISTORY_SH" publish series.jsonl >/dev/null 2>&1
race_rc=$?
rm -f "$WORK/.git/hooks/pre-push"
check "losing the push race fails rather than overwriting" "1" "$([[ $race_rc -ne 0 ]] && echo 1 || echo 0)"
check "the competing run's record survived" "1" \
    "$(git -C "$BENCH_HISTORY_REMOTE" show "$BENCH_HISTORY_BRANCH:$BENCH_HISTORY_PATH" | grep -c 'BenchmarkRace')"

# ---------------------------------------------------------------------------
info "10. the push is not forced"
# ---------------------------------------------------------------------------
# A source assertion rather than a behavioural one, and deliberately so: as check 9 shows, git's
# ref-lock CAS masks the difference in every race this harness can construct, so no black-box test
# here distinguishes a forced push from an unforced one. The property still matters — --force would
# discard a competing run's commits in any window the CAS does not cover — so it is pinned here
# where it can at least not be reintroduced silently.
# Covers --force, a bundled short flag such as -qf, and a leading + on the refspec, all of which
# force-update. Matching only "--force" would let the other two through.
check "publish does not force-push" "0" \
    "$(grep -cE 'git[^#]*push([^#]*(--force|--mirror)|[^#]*[[:space:]]-[a-zA-Z]*f|[^#]*[[:space:]]"?\+)' "$SCRIPT_DIR/bench_history.sh")"

echo ""
if [[ "$FAILURES" -eq 0 ]]; then
    echo -e "${GREEN}All bench history checks passed!${NC}"
    exit 0
fi
echo -e "${RED}${FAILURES} check(s) failed.${NC}"
exit 1
