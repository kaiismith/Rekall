#!/usr/bin/env bash
# clang-format wrapper. Pass --check to fail when files would be reformatted.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MODE="${1:-write}"

mapfile -t FILES < <(find "$ROOT/src" "$ROOT/include" "$ROOT/tests" \
    -type f \( -name '*.cpp' -o -name '*.hpp' -o -name '*.h' -o -name '*.cc' \) \
    -not -path '*/third_party/*' -not -path '*/build/*' 2>/dev/null || true)

if [[ ${#FILES[@]} -eq 0 ]]; then
    echo "no source files found"
    exit 0
fi

case "$MODE" in
    --check|check)
        clang-format --dry-run --Werror "${FILES[@]}"
        ;;
    *)
        clang-format -i "${FILES[@]}"
        echo "formatted ${#FILES[@]} files"
        ;;
esac
