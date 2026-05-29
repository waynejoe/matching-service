# data

数据访问层，只负责 MySQL 连接、事务和表级 Repo。

## 当前内容

- `Data`：数据库连接和事务入口。
- `DepositRepo`：入金单访问。
- `BasketRepo`：出金篮子访问。
- `BasketDepositRepo`：篮子入金明细访问。
- `MatchRecordRepo`：撮合结果访问。
- `StateLogRepo`：状态流水访问。
- `EventInboxRepo`：RocketMQ 消费幂等访问。
- Redis 分布式锁放在 `pkg/toolbox/redisx`，不放在 data 层。

## 边界

- 这里不写撮合算法。
- 这里不维护内存索引。
- 撮合逻辑放在 `internal/engine`。
- 业务编排放在 `internal/biz`。
