#!/usr/bin/env bash
set -euo pipefail

CONF="${CONF:-./configs/config.yaml}"
GOCACHE="${GOCACHE:-/tmp/go-build}"
KIND="${KIND:-}"
BODY_FILE="${BODY_FILE:-}"
EVENT_ID="${EVENT_ID:-}"
DELAY_LEVEL="${DELAY_LEVEL:-0}"

# 重投 RocketMQ 死信或人工补偿消息。
GOCACHE="${GOCACHE}" go run ./cmd/matching-tools replay \
	-conf "${CONF}" \
	-kind "${KIND}" \
	-body-file "${BODY_FILE}" \
	-event-id "${EVENT_ID}" \
	-delay-level "${DELAY_LEVEL}"
