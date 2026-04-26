#!/usr/bin/env bash
# Fetches whisper.cpp ggml-* model weights from huggingface and verifies SHA-256.
#
# Usage:  ./scripts/download_models.sh tiny.en small.en
#         ASR_MODELS_DIR=/var/lib/rekall-asr/models ./scripts/download_models.sh small.en
#
set -euo pipefail

MODELS_DIR="${ASR_MODELS_DIR:-$(cd "$(dirname "$0")/.." && pwd)/models}"
BASE_URL="https://huggingface.co/ggerganov/whisper.cpp/resolve/main"

# SHA-256 sums lifted from the upstream repo's models/download-ggml-model.sh
declare -A SHA256
SHA256[tiny.en]="921e4cf8686fdd993dcd081a5da5b6c365bfde1162e72b08d75ac75289920b1f"
SHA256[tiny]="bd577a113a864445d4c299885e0cb97d4ba92b5f"
SHA256[base.en]="137c40403d78fd54d454da0f9bd998f78703390c"
SHA256[base]="60ed5bc3dd14eea856493d334349b405782ddcaf"
SHA256[small.en]="db8a495a91d927739e50b3fc1cc4c6b8f6c2d022"
SHA256[small]="1be3a9b2063867b937e64e2ec7483364a79917e9"
SHA256[medium.en]="cb1a4dada917be01a9085b8db05bf4f3b65c9a0c"
SHA256[medium]="fd9727b6e1217c2f614f9b698455c4ffd82463b4"
SHA256[large-v3]="ad82bf6a9043ceed055076d0fd39f5f186ff8062"

verify() {
    local file="$1" expected="$2"
    if command -v sha256sum >/dev/null; then
        local actual; actual=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum >/dev/null; then
        local actual; actual=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        echo "warn: no sha256sum/shasum found; skipping verification" >&2
        return 0
    fi
    if [[ "$actual" != "$expected" ]]; then
        echo "FAIL: $file sha256 mismatch (got $actual, want $expected)" >&2
        return 1
    fi
    echo "ok:   $file sha256 verified"
}

if [[ $# -eq 0 ]]; then
    echo "usage: $0 <model_id> [<model_id>...]" >&2
    echo "models: ${!SHA256[*]}" >&2
    exit 2
fi

mkdir -p "$MODELS_DIR"
for model in "$@"; do
    if [[ -z "${SHA256[$model]:-}" ]]; then
        echo "unknown model: $model (known: ${!SHA256[*]})" >&2
        exit 2
    fi
    out="$MODELS_DIR/ggml-${model}.bin"
    if [[ -f "$out" ]]; then
        echo "skip: $out already exists"
        continue
    fi
    echo "downloading ggml-${model}.bin ..."
    curl -fSL "$BASE_URL/ggml-${model}.bin" -o "$out.tmp"
    mv "$out.tmp" "$out"
    verify "$out" "${SHA256[$model]}" || { rm -f "$out"; exit 1; }
done

echo "done. models in $MODELS_DIR"
