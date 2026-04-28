#!/usr/bin/env bash
# Rekall — Go pre-commit hook integration test (Linux / macOS only).
#
# Spawns a throw-away git repo, points its core.hooksPath at the real
# repo-root .husky/, and exercises three cases against the actual hook:
#
#   G1. Bad gofmt + bad imports → auto-fixed, commit lands clean
#   G2. Vet-flagged code (Printf("%s", 42)) → commit blocked
#   G3. Bypass with --no-verify → commit lands
#
# Skips cleanly when the Go toolchain is not on $PATH — a missing tool is
# not a test failure (Requirement 10.3).

set -euo pipefail

# ── Skip if the toolchain is not present ─────────────────────────────────────
for t in gofmt goimports golangci-lint go; do
  if ! command -v "$t" >/dev/null 2>&1; then
    echo "skipping: $t not on \$PATH"
    exit 0
  fi
done

# ── Resolve paths ────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
HUSKY_DIR="$REPO_ROOT/.husky"
BACKEND_DIR="$REPO_ROOT/backend"

if [ ! -f "$HUSKY_DIR/pre-commit" ]; then
  echo "FAIL: $HUSKY_DIR/pre-commit not found" >&2
  exit 1
fi

# ── Set up throw-away git repo ───────────────────────────────────────────────
TMP="$(mktemp -d)"
SANDBOX_DIR="$BACKEND_DIR/cmd/server/__pre_commit_test_sandbox__"

cleanup() {
  rm -rf "$TMP"
  rm -rf "$SANDBOX_DIR"
}
trap cleanup EXIT

git -C "$TMP" init -q
git -C "$TMP" config user.email "test@rekall.local"
git -C "$TMP" config user.name  "Rekall Test"
git -C "$TMP" config commit.gpgsign false

# Mirror the real layout via symlinks so the hook's cd "$REPO_ROOT/backend"
# lands somewhere real with a working go.mod and node_modules etc.
ln -s "$BACKEND_DIR" "$TMP/backend"
ln -s "$REPO_ROOT/frontend" "$TMP/frontend" 2>/dev/null || true
ln -s "$HUSKY_DIR" "$TMP/.husky"

git -C "$TMP" config core.hooksPath .husky/_

# Seed the repo with one initial commit so refs exist.
echo "seed" > "$TMP/seed.txt"
git -C "$TMP" add seed.txt
git -C "$TMP" commit -q -m "seed" --no-verify

# Scope-isolate to the Go block: skip the frontend block entirely.
export REKALL_PRE_COMMIT_SKIP_FRONTEND=1

mkdir -p "$SANDBOX_DIR"

PASS=0
FAIL=0
assert() {
  local label="$1"
  local cond="$2"
  if [ "$cond" = "1" ]; then
    echo "  PASS  $label"
    PASS=$((PASS + 1))
  else
    echo "  FAIL  $label" >&2
    FAIL=$((FAIL + 1))
  fi
}

# ── Case G1 — bad gofmt → auto-fixed, commit lands ───────────────────────────
echo ""
echo "Case G1 — bad gofmt should be auto-fixed and the commit should land"
G1_REL="backend/cmd/server/__pre_commit_test_sandbox__/g1.go"
# Mis-spaced braces, missing import grouping. gofmt + goimports should fix.
cat > "$REPO_ROOT/$G1_REL" <<'EOF'
package sandbox
import "fmt"
func G1Hello(){fmt.Println("hi")}
EOF

git -C "$TMP" add -- "$G1_REL"
set +e
COMMIT_OUT="$(REKALL_PRE_COMMIT_SKIP_FRONTEND=1 git -C "$TMP" commit -m "g1" 2>&1)"
COMMIT_EXIT=$?
set -e

if [ $COMMIT_EXIT -eq 0 ]; then
  assert "G1 commit landed" 1
else
  echo "----- captured commit output -----"
  echo "$COMMIT_OUT"
  echo "----- end -----"
  assert "G1 commit landed" 0
fi

if gofmt -l "$REPO_ROOT/$G1_REL" | grep -q .; then
  assert "G1 file is gofmt-clean after commit" 0
else
  assert "G1 file is gofmt-clean after commit" 1
fi

rm -f "$REPO_ROOT/$G1_REL"

# ── Case G2 — vet-flagged code → commit blocked ──────────────────────────────
echo ""
echo "Case G2 — vet-flagged code should block the commit"
G2_REL="backend/cmd/server/__pre_commit_test_sandbox__/g2.go"
cat > "$REPO_ROOT/$G2_REL" <<'EOF'
package sandbox

import "fmt"

// G2Bad has a Printf format/arg mismatch that go vet will catch.
func G2Bad() { fmt.Printf("%s", 42) }
EOF

git -C "$TMP" add -- "$G2_REL"
set +e
COMMIT_OUT="$(REKALL_PRE_COMMIT_SKIP_FRONTEND=1 git -C "$TMP" commit -m "g2" 2>&1)"
COMMIT_EXIT=$?
set -e

if [ $COMMIT_EXIT -ne 0 ]; then
  assert "G2 commit blocked (exit != 0)" 1
else
  echo "----- captured commit output -----"
  echo "$COMMIT_OUT"
  echo "----- end -----"
  assert "G2 commit blocked (exit != 0)" 0
fi

# vet OR golangci-lint (which wraps staticcheck and others) should report it.
if echo "$COMMIT_OUT" | grep -Eiq '%s|wrong type|govet|printf'; then
  assert "G2 stderr mentions a vet/lint diagnostic" 1
else
  echo "----- captured commit output -----"
  echo "$COMMIT_OUT"
  echo "----- end -----"
  assert "G2 stderr mentions a vet/lint diagnostic" 0
fi

# Reset state for next case.
git -C "$TMP" reset -q HEAD -- "$G2_REL" || true

# ── Case G3 — --no-verify bypass honoured ────────────────────────────────────
echo ""
echo "Case G3 — --no-verify should bypass the hook"
git -C "$TMP" add -- "$G2_REL"
set +e
COMMIT_OUT="$(git -C "$TMP" commit -m "g3" --no-verify 2>&1)"
COMMIT_EXIT=$?
set -e

if [ $COMMIT_EXIT -eq 0 ]; then
  assert "G3 --no-verify exit 0" 1
else
  echo "----- captured commit output -----"
  echo "$COMMIT_OUT"
  echo "----- end -----"
  assert "G3 --no-verify exit 0" 0
fi

if [ -n "$(git -C "$TMP" log --oneline -- "$G2_REL")" ]; then
  assert "G3 bypass commit landed" 1
else
  assert "G3 bypass commit landed" 0
fi

rm -f "$REPO_ROOT/$G2_REL"

# ── Summary ──────────────────────────────────────────────────────────────────
echo ""
echo "── Go pre-commit hook integration test summary ──"
echo "  passed: $PASS"
echo "  failed: $FAIL"

if [ $FAIL -ne 0 ]; then
  exit 1
fi

echo "All cases passed."
