#!/usr/bin/env bash
# Rekall — frontend pre-commit hook integration test (Linux / macOS only).
#
# Spawns a throw-away git repo, points its core.hooksPath at the real
# repo-root .husky/, and exercises three cases against the actual hook:
#
#   1. Bad TS staged (no-explicit-any)         → commit blocked, eslint diag
#   2. Good TS staged                          → commit lands, prettier formats
#   3. Bad TS staged + --no-verify             → commit lands (bypass honoured)
#
# Run via `npm run test:hook` from frontend/.

set -euo pipefail

# ── Resolve paths ────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
HUSKY_DIR="$REPO_ROOT/.husky"
FRONTEND_DIR="$REPO_ROOT/frontend"

if [ ! -f "$HUSKY_DIR/pre-commit" ]; then
  echo "FAIL: $HUSKY_DIR/pre-commit not found" >&2
  exit 1
fi

if [ ! -d "$FRONTEND_DIR/node_modules" ]; then
  echo "FAIL: frontend/node_modules missing — run 'npm install' first" >&2
  exit 1
fi

# ── Set up throw-away git repo ───────────────────────────────────────────────
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

git -C "$TMP" init -q
git -C "$TMP" config user.email "test@rekall.local"
git -C "$TMP" config user.name  "Rekall Test"
git -C "$TMP" config commit.gpgsign false

# Mirror the real layout so the hook's cd "$REPO_ROOT/frontend" lands somewhere
# real. We symlink frontend/ from the real checkout so npx, eslint, prettier,
# tsc, and the project's tsconfig / src tree are all available unchanged.
ln -s "$FRONTEND_DIR" "$TMP/frontend"
ln -s "$HUSKY_DIR"    "$TMP/.husky"

git -C "$TMP" config core.hooksPath .husky

# Seed the repo with one initial commit so refs exist.
echo "seed" > "$TMP/seed.txt"
git -C "$TMP" add seed.txt
git -C "$TMP" commit -q -m "seed" --no-verify

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

# Helper: stage a file in the temp repo. The path must live under the symlinked
# frontend/ tree; we add a sandbox subdir so we don't pollute real src/.
SANDBOX_DIR="$FRONTEND_DIR/src/__pre_commit_test_sandbox__"
mkdir -p "$SANDBOX_DIR"
cleanup_sandbox() { rm -rf "$SANDBOX_DIR"; }
trap 'cleanup_sandbox; rm -rf "$TMP"' EXIT

# ── Case 1 — known-bad TS staged: hook blocks ────────────────────────────────
echo ""
echo "Case 1 — bad TypeScript should block the commit"
BAD_REL="frontend/src/__pre_commit_test_sandbox__/bad.ts"
cat > "$REPO_ROOT/$BAD_REL" <<'EOF'
export const x: any = 1
EOF

git -C "$TMP" add -- "$BAD_REL"
set +e
COMMIT_OUT="$(git -C "$TMP" commit -m "should fail" 2>&1)"
COMMIT_EXIT=$?
set -e

if [ $COMMIT_EXIT -ne 0 ]; then assert "exit code != 0" 1; else assert "exit code != 0" 0; fi
if echo "$COMMIT_OUT" | grep -q 'no-explicit-any'; then
  assert "stderr mentions no-explicit-any" 1
else
  echo "----- captured commit output -----"
  echo "$COMMIT_OUT"
  echo "----- end -----"
  assert "stderr mentions no-explicit-any" 0
fi

# Reset working state for next case.
git -C "$TMP" reset -q HEAD -- "$BAD_REL" || true
rm -f "$REPO_ROOT/$BAD_REL"

# ── Case 2 — known-good TS staged: hook passes, prettier formats ─────────────
echo ""
echo "Case 2 — clean TypeScript should commit and be formatted"
GOOD_REL="frontend/src/__pre_commit_test_sandbox__/good.ts"
# Intentionally written WITHOUT a trailing newline so we can verify prettier
# adds one as part of its --write pass (assertion below).
printf 'export const greeting: string = "hi"' > "$REPO_ROOT/$GOOD_REL"

git -C "$TMP" add -- "$GOOD_REL"
set +e
COMMIT_OUT="$(git -C "$TMP" commit -m "good" 2>&1)"
COMMIT_EXIT=$?
set -e

if [ $COMMIT_EXIT -eq 0 ]; then assert "exit code == 0" 1; else
  echo "----- captured commit output -----"
  echo "$COMMIT_OUT"
  echo "----- end -----"
  assert "exit code == 0" 0
fi

if [ -n "$(git -C "$TMP" log --oneline -- "$GOOD_REL")" ]; then
  assert "commit landed" 1
else
  assert "commit landed" 0
fi

# Prettier should have rewritten the file: single quotes → double, missing
# trailing newline added. The .prettierrc sets singleQuote: true so the source
# stays with double quotes converted to single.
LAST_LINE_OF_GOOD="$(tail -c 1 "$REPO_ROOT/$GOOD_REL" | xxd -p)"
if [ "$LAST_LINE_OF_GOOD" = "0a" ]; then
  assert "prettier added trailing newline" 1
else
  assert "prettier added trailing newline" 0
fi

# Reset state for next case.
rm -f "$REPO_ROOT/$GOOD_REL"
git -C "$TMP" rm -q --cached -- "$GOOD_REL" 2>/dev/null || true

# ── Case 3 — bad TS staged + --no-verify: bypass honoured ────────────────────
echo ""
echo "Case 3 — --no-verify must bypass the hook"
BYPASS_REL="frontend/src/__pre_commit_test_sandbox__/bypass.ts"
cat > "$REPO_ROOT/$BYPASS_REL" <<'EOF'
export const y: any = 2
EOF

git -C "$TMP" add -- "$BYPASS_REL"
set +e
COMMIT_OUT="$(git -C "$TMP" commit -m "bypass" --no-verify 2>&1)"
COMMIT_EXIT=$?
set -e

if [ $COMMIT_EXIT -eq 0 ]; then assert "--no-verify exit 0" 1; else
  echo "----- captured commit output -----"
  echo "$COMMIT_OUT"
  echo "----- end -----"
  assert "--no-verify exit 0" 0
fi

if [ -n "$(git -C "$TMP" log --oneline -- "$BYPASS_REL")" ]; then
  assert "bypass commit landed" 1
else
  assert "bypass commit landed" 0
fi

rm -f "$REPO_ROOT/$BYPASS_REL"

# ── Summary ──────────────────────────────────────────────────────────────────
echo ""
echo "── pre-commit hook integration test summary ──"
echo "  passed: $PASS"
echo "  failed: $FAIL"

if [ $FAIL -ne 0 ]; then
  exit 1
fi

echo "All cases passed."
