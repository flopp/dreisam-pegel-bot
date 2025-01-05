#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)

"${SCRIPT_DIR}/bot-linux"  \
    "${SCRIPT_DIR}/production-config.json" \
    "${SCRIPT_DIR}/.data"
