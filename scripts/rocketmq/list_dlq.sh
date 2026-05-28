#!/usr/bin/env bash
set -euo pipefail

ROCKETMQ_BROKER_CONTAINER="${ROCKETMQ_BROKER_CONTAINER:-rmqbroker}"
ROCKETMQ_NAMESRV="${ROCKETMQ_NAMESRV:-rmqnamesrv:9876}"
ROCKETMQ_GROUP="${ROCKETMQ_GROUP:-matching-service}"
ROCKETMQ_HOME="${ROCKETMQ_HOME:-/home/rocketmq/rocketmq-5.3.2}"

# 打印本服务消费组的死信消息。
docker exec "${ROCKETMQ_BROKER_CONTAINER}" sh -lc "cd ${ROCKETMQ_HOME}/bin && ./mqadmin printMsg -n ${ROCKETMQ_NAMESRV} -t '%DLQ%${ROCKETMQ_GROUP}' -d true"
