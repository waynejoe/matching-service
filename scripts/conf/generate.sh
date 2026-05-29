#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT}"

PROTO_INCLUDE="${PROTO_INCLUDE:-/opt/homebrew/include}"

protoc --proto_path=./internal/conf \
	--proto_path=./third_party \
	--proto_path="${PROTO_INCLUDE}" \
	--go_out=paths=source_relative:./internal/conf \
	./internal/conf/conf.proto
