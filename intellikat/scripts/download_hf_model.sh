#!/usr/bin/env bash
# Pre-pull HuggingFace weights into a target cache dir at image build time.
# Used by the `intellikat-local` Docker target so cold starts don't hit the
# network. Hosted-mode deployments skip this entirely.
#
# Usage: download_hf_model.sh <model_id> <cache_dir>
set -euo pipefail

MODEL_ID="${1:-MarieAngeA13/Sentiment-Analysis-BERT}"
CACHE_DIR="${2:-/var/cache/intellikat/hf}"

mkdir -p "${CACHE_DIR}"

python - <<PY
import os
os.environ["HF_HOME"] = "${CACHE_DIR}"
os.environ["TRANSFORMERS_CACHE"] = "${CACHE_DIR}"

from transformers import AutoModelForSequenceClassification, AutoTokenizer

print(f"pre-pulling {'${MODEL_ID}'} into ${CACHE_DIR}")
AutoTokenizer.from_pretrained("${MODEL_ID}", cache_dir="${CACHE_DIR}")
AutoModelForSequenceClassification.from_pretrained("${MODEL_ID}", cache_dir="${CACHE_DIR}")
print("done.")
PY
