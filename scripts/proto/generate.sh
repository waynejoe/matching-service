#!/usr/bin/env bash
set -euo pipefail

PROTO_INCLUDE="${PROTO_INCLUDE:-/opt/homebrew/Cellar/protobuf/33.4_1/include}"
PROTO_FILES=("$@")

if [ "${#PROTO_FILES[@]}" -eq 0 ]; then
	mapfile -t PROTO_FILES < <(find api -name '*.proto')
fi

# 生成 gRPC protobuf 代码。
protoc --proto_path=. \
	--proto_path="${PROTO_INCLUDE}" \
	--go_out=. \
	--go_opt=module=matching-service \
	--go-grpc_out=. \
	--go-grpc_opt=module=matching-service \
	"${PROTO_FILES[@]}"
