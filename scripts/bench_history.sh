#!/usr/bin/env bash
# bench_history.sh — fetch and publish the benchmark ns/op history branch.
#
# Kept out of the workflow YAML so the git plumbing can be tested locally against a throwaway
# repository rather than only by waiting for the scheduled run. See scripts/bench_history_test.sh.
#
# Usage:
#   bench_history.sh fetch   <series-file>   # write existing history to <series-file>
#   bench_history.sh publish <series-file>   # commit and push <series-file> onto the history branch
#
# Environment:
#   BENCH_HISTORY_REMOTE  remote name or URL          (default: origin)
#   BENCH_HISTORY_BRANCH  branch holding the history  (default: benchmarks-history)
#   BENCH_HISTORY_PATH    path within that branch     (default: benchmarks/series.jsonl)
#   BENCH_HISTORY_LABEL   text appended to the commit subject (default: local)

set -euo pipefail

REMOTE="${BENCH_HISTORY_REMOTE:-origin}"
BRANCH="${BENCH_HISTORY_BRANCH:-benchmarks-history}"
BRANCH_PATH="${BENCH_HISTORY_PATH:-benchmarks/series.jsonl}"
LABEL="${BENCH_HISTORY_LABEL:-local}"

# Global rather than a local in cmd_publish: the EXIT trap runs after the function has returned, so
# a local would already be out of scope and `set -u` would abort inside the trap itself.
WORKTREE=""
cleanup() {
    [[ -n "$WORKTREE" ]] || return 0
    git worktree remove --force "$WORKTREE" >/dev/null 2>&1 || true
    rm -rf "$WORKTREE"
}
trap cleanup EXIT

die() {
    if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
        echo "::error::$1"
    else
        echo "bench_history: $1" >&2
    fi
    exit 1
}

# resolve_branch_state sets BRANCH_STATE to exists or absent, or dies. git ls-remote --exit-code
# returns 2 for "no matching ref" and other non-zero codes for transport failures. Collapsing the
# two would let a transient network error look like a first run, and the history would then be
# published over the real one.
#
# It assigns to a global rather than echoing a value, because a caller would have to invoke it in a
# command substitution and `exit` inside one only leaves the subshell — the transport failure would
# be swallowed and treated as "absent", which is the exact bug this guards against.
BRANCH_STATE=""
resolve_branch_state() {
    local rc=0
    git ls-remote --exit-code --heads "$REMOTE" "$BRANCH" >/dev/null 2>&1 || rc=$?
    case "$rc" in
        0) BRANCH_STATE=exists ;;
        2) BRANCH_STATE=absent ;;
        *) die "could not determine whether $BRANCH exists on $REMOTE (git ls-remote exit $rc)" ;;
    esac
}

cmd_fetch() {
    local out="$1"
    resolve_branch_state
    if [[ "$BRANCH_STATE" != exists ]]; then
        echo "bench_history: no $BRANCH branch yet, starting a new history"
        : > "$out"
        return 0
    fi
    git fetch --quiet --force "$REMOTE" "$BRANCH:$BRANCH"
    # The branch can exist without the series file, after a BENCH_HISTORY_PATH change or a branch
    # created by hand. assert_extends_published already treats that as an empty history; without the
    # same treatment here git show aborts the job, and does so on every scheduled run thereafter.
    if git cat-file -e "$BRANCH:$BRANCH_PATH" 2>/dev/null; then
        git show "$BRANCH:$BRANCH_PATH" > "$out"
    else
        echo "bench_history: $BRANCH holds no $BRANCH_PATH yet, starting a new history"
        : > "$out"
    fi
}

# assert_extends_published refuses to publish a series that does not begin with what is already on
# the branch. Publishing copies the file wholesale, so a series built from a fetch that predates
# another run's push would silently drop that run's records — the push itself would still be a clean
# fast-forward, so nothing else would catch it. The history is append-only by construction, and this
# is the invariant that makes that true.
# Compared as a byte prefix rather than by line count: a published file with no trailing newline
# makes `wc -l` undercount, which would reject a perfectly good append and keep rejecting it
# forever. Bytes also make truncation (a candidate shorter than what is published) a mismatch.
assert_extends_published() {
    local series="$1" published published_size series_size
    published="$(mktemp)"
    # A read failure must not look like an empty history: treating it as empty skips the whole
    # prefix check and publishes straight over whatever is there. Only a genuinely absent path is
    # an empty history, so the two are distinguished before deciding.
    if git cat-file -e "$BRANCH:$BRANCH_PATH" 2>/dev/null; then
        if ! git show "$BRANCH:$BRANCH_PATH" > "$published" 2>/dev/null; then
            rm -f "$published"
            die "cannot read $BRANCH:$BRANCH_PATH; refusing to publish without verifying the existing history"
        fi
    else
        : > "$published"
    fi
    published_size="$(wc -c < "$published" | tr -d ' ')"
    series_size="$(wc -c < "$series" | tr -d ' ')"

    if [[ "$published_size" -gt 0 ]]; then
        if [[ "$series_size" -lt "$published_size" ]] || \
           ! head -c "$published_size" "$series" | cmp -s - "$published"; then
            rm -f "$published"
            die "$series does not extend the published history: it diverges within the first $published_size bytes of $BRANCH. It was built from a stale fetch, and publishing it would drop already-recorded runs."
        fi
    fi
    rm -f "$published"
}

cmd_publish() {
    local series="$1"
    [[ -s "$series" ]] || die "$series is empty, refusing to publish an empty history"

    # git worktree add requires the path not to exist yet. Declared separately so a mktemp failure
    # is not masked by the exit status of the local declaration itself.
    local work staged
    work="$(mktemp -du)"

    # Committing onto the branch's own history, rather than re-creating an orphan root each run,
    # keeps every previous commit reachable and makes a stale push a fast-forward failure.
    resolve_branch_state
    if [[ "$BRANCH_STATE" == exists ]]; then
        git fetch --quiet --force "$REMOTE" "$BRANCH:$BRANCH"
        assert_extends_published "$series"
        git worktree add --quiet "$work" "$BRANCH"
    else
        git worktree add --quiet --detach "$work"
    fi
    # Recorded only once the worktree exists. mktemp -du reserves nothing, so if another process
    # won the path and the add failed, cleanup would otherwise rm -rf a directory we never created.
    WORKTREE="$work"

    if [[ "$BRANCH_STATE" != exists ]]; then
        git -C "$work" checkout --quiet --orphan "$BRANCH"
        # checkout --orphan stages the previous branch's tree; drop it so the history branch holds
        # only the series file. Assigned separately because a failing ls-files inside a test would
        # look like "no files staged", silently committing the whole master tree into the history.
        if ! staged="$(git -C "$work" ls-files)"; then
            die "cannot list the staged files in $work"
        fi
        if [[ -n "$staged" ]]; then
            git -C "$work" rm -rfq --cached .
        fi
    fi

    mkdir -p "$work/$(dirname "$BRANCH_PATH")"
    cp "$series" "$work/$BRANCH_PATH"

    # -f because the series file is gitignored for local runs; without it the add is refused
    git -C "$work" add -f "$BRANCH_PATH"
    if git -C "$work" diff --cached --quiet; then
        echo "bench_history: no new records, nothing to publish"
        return 0
    fi
    git -C "$work" -c user.name="${GIT_AUTHOR_NAME:-github-actions[bot]}" \
        -c user.email="${GIT_AUTHOR_EMAIL:-github-actions[bot]@users.noreply.github.com}" \
        commit --quiet -m "benchmarks: append history for $LABEL"

    # No --force: a concurrent run that already pushed makes this a non-fast-forward, and the job
    # fails loudly instead of silently discarding the other run's records.
    git -C "$work" push --quiet "$REMOTE" "$BRANCH:$BRANCH"
    echo "bench_history: published $(wc -l < "$series") records to $BRANCH"
}

case "${1:-}" in
    fetch)   [[ $# -eq 2 ]] || die "usage: bench_history.sh fetch <series-file>";   cmd_fetch "$2" ;;
    publish) [[ $# -eq 2 ]] || die "usage: bench_history.sh publish <series-file>"; cmd_publish "$2" ;;
    *)       die "usage: bench_history.sh {fetch|publish} <series-file>" ;;
esac
