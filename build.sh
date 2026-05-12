#!/bin/bash

set -euo pipefail

if [ "${1:-}" = "--clean" ]; then
    go clean -cache -modcache 2>/dev/null || true
    make clean
fi

go mod tidy
make build
make sync
