#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT}"

PROTO_INCLUDE="${PROTO_INCLUDE:-/opt/homebrew/include}"
if [ "$#" -gt 0 ]; then
	PROTO_FILES=("$@")
else
	PROTO_FILES=($(find pkg/api -name '*.proto'))
fi

protoc --proto_path=. \
	--proto_path=./third_party \
	--proto_path="${PROTO_INCLUDE}" \
	--go_out=. \
	--go_opt=module=matching-service \
	--go-grpc_out=. \
	--go-grpc_opt=module=matching-service \
	--go-errors_out=paths=source_relative:. \
	--validate_out=paths=source_relative,lang=go:. \
	"${PROTO_FILES[@]}"
