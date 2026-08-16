#!/usr/bin/env bash
set -e

mkdir -p bin

if command -v go >/dev/null 2>&1; then
    echo "[*] Building Go CLI directly on host..."
    (cd cli && go build -o ../bin/cli .)
    echo "[✓] Go CLI built successfully at ./bin/cli"
else
    echo "[*] Go not found on host. Building Go CLI inside Docker container..."
    docker run --rm \
        -v "$(pwd)":/usr/src/app \
        -w /usr/src/app/cli \
        golang:1.22-alpine \
        go build -o /usr/src/app/bin/cli .
    echo "[✓] Go CLI built via Docker at ./bin/cli"
fi
