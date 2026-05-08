#!/bin/bash
set -e

echo "==> Running contract tests..."
cd tests/contract && go test ./... -v
echo "==> Contract tests done."
