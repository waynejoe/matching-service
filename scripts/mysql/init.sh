#!/usr/bin/env bash
set -euo pipefail

MYSQL_CONTAINER="${MYSQL_CONTAINER:-mysql}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-123456}"

# mysql 在本地 Docker MySQL 容器内执行。
mysql_exec() {
	docker exec -i "${MYSQL_CONTAINER}" mysql -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "$@"
}

mysql_exec < scripts/mysql/001_init_schema.sql
mysql_exec syntra_match < scripts/mysql/002_match_short_amount.sql
