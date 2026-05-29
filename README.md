# matching-service

撮合服务代码库，当前已包含数据模型、内存撮合引擎、业务用例、启动恢复、幂等处理、Redis 分片锁、RocketMQ 顺序消费、超时扫描、健康检查、Prometheus 指标和 gRPC 接口。

## 目录说明

```text
matching-service/
├── api/
│   └── matching/
│       └── v1/                 # proto 定义及生成代码
├── cmd/
│   └── matching-service/       # 服务启动入口
├── configs/                    # 配置文件
├── internal/
│   ├── biz/                    # 业务用例层
│   ├── conf/                   # 配置结构
│   ├── data/                   # MySQL 表级访问层
│   ├── engine/                 # 撮合引擎核心逻辑
│   ├── model/                  # GORM 数据模型
│   ├── server/                 # gRPC、RocketMQ、Prometheus、后台任务等运行入口
│   └── service/                # gRPC 接口实现、MQ 消息处理
├── pkg/
│   ├── idgen/                  # 可复用 ID 生成工具
│   └── lock/                   # 可复用 Redis 分布式锁
├── scripts/
│   ├── mysql/                  # MySQL 初始化和变更脚本
│   └── rocketmq/               # RocketMQ 本地管理脚本
├── test/
│   └── integration/            # 本服务集成测试
├── go.mod
└── go.sum
```

## gRPC 接口

proto 文件：

```text
api/matching/v1/matching.proto
```

当前接口：

- `CreateBasket`：创建出金篮子。
- `SubmitDeposit`：提交入金并触发撮合。
- `ExpireTimeouts`：手动触发一次超时扫描。
- `GetDeposit`：查询入金单状态。
- `GetBasket`：查询出金篮子状态。
- `GetMatch`：查询撮合结果和入金明细。
- `CheckHealth`：检查 MySQL、Redis、RocketMQ 是否可用。
- `GetMetrics`：查询当前进程内存指标。

## 健康检查和指标

- `CheckHealth` 返回整体可用状态和每个组件状态。
- `GetMetrics` 返回消费成功/失败、撮合成功、少发成交、过期处理、事件失败和事件重试计数。
- Prometheus 指标地址：`http://127.0.0.1:8203/metrics`。
- Prometheus 暴露 Go 运行时、进程指标和 `matching_*` 业务指标。
- 指标保存在当前服务进程内存里，服务重启后重新计数。

## 并发保护

- 单进程内：每个内存分片有互斥锁。
- 多实例间：同一个 `channel:currency` 会先抢 Redis 分片锁。
- 数据库兜底：更新篮子时要求 `status = waiting`，否则事务失败。

## 超时处理

- 待撮合入金过期：入金状态改为 `expired`。
- 已挂入入金过期：释放篮子明细，扣减篮子已凑金额，同步内存索引。
- 出金篮子过期：已凑金额达到 `min_complete_rate` 时少发成交；否则篮子过期并释放已挂入入金。

## 撮合规则

- 入金只能挂入 `current_amount + amount <= target_amount` 的出金篮子。
- 刚好达到 `target_amount` 时立即成交。
- 超过 `target_amount` 的入金不会挂入该篮子。
- 出金超时后，已凑金额达到最低成交比例时按当前金额成交。

## 测试命令

```bash
make db-init
make rocketmq-topic
make test
make integration
```

`make integration` 会连接本地 MySQL、Redis 和 RocketMQ，验证少发成交、低于最低比例失败、超过剩余金额不挂入、MQ 消息入口处理、失败事件重试，以及 RocketMQ 真实收发消费链路。

## 数据库初始化

本地 Docker MySQL 创建库表：

```bash
make db-init
```

执行内容：

- `scripts/mysql/001_init_schema.sql`：完整建库建表。
- `scripts/mysql/002_match_short_amount.sql`：兼容旧表结构，把超发字段迁移为少发字段。

## RocketMQ 消息设计

- 主队列：`match_deposit` 和 `match_withdraw`。
- 延迟重试：顺序消费失败后返回 `SuspendCurrentQueueAMoment`，按 `suspend_queue_millis` 暂停当前队列后重试。
- 死信队列：超过 `max_reconsume_times` 后进入 RocketMQ 消费组 DLQ，默认类似 `%DLQ%matching-service`。
- 分片顺序：生产方应按 `channel:currency` 作为 sharding key 投递到同一队列，同一队列由 RocketMQ 顺序消费保证串行。
- 发送工具：`internal/server/RocketMQProducer` 已使用 `HashQueueSelector` 和 `WithShardingKey`。
- 延迟消息：发送时传 `delayLevel > 0`，RocketMQ 到期后再投递到主队列。

## RocketMQ 初始化

本地 Docker 环境创建或更新 topic：

```bash
make rocketmq-topic
```

创建内容：

- `matching-service`：消费组，开启顺序消费，最大重试 16 次。
- `match_deposit`：入金主队列，8 读 / 8 写。
- `match_withdraw`：出金主队列，8 读 / 8 写。
- `%RETRY%matching-service`：重试队列，1 读 / 1 写。
- `%DLQ%matching-service`：死信队列，1 读 / 1 写。

常用检查：

```bash
make rocketmq-topic-list
make rocketmq-dlq-list
```

## RocketMQ 死信补偿

查看 DLQ：

```bash
make rocketmq-dlq-list
```

把需要补偿的消息 body 保存成 JSON 文件后重投：

```bash
make rocketmq-replay KIND=deposit BODY_FILE=/tmp/deposit-dlq.json
make rocketmq-replay KIND=withdraw BODY_FILE=/tmp/withdraw-dlq.json
```

说明：

- `KIND=deposit` 会重投到 `match_deposit`。
- `KIND=withdraw` 会重投到 `match_withdraw`。
- 不传 `EVENT_ID` 时保留原事件 ID，适合原事件状态为失败后的重试。
- 传 `EVENT_ID=xxx` 时会覆盖消息体里的事件 ID，适合人工确认后重新发起一笔补偿事件。
- 重投工具会使用消息 `data.channel` 和 `data.currency` 作为 RocketMQ sharding key，保证同一业务分片顺序。

## RocketMQ 消息格式

消息 envelope 和 data 统一由 proto 定义（`api/matching/v1/matching.proto`），JSON 序列化使用 proto3 标准的 snake_case 字段名。

入金 topic：

```json
{
  "event_id": "deposit-event-001",
  "topic": "match_deposit",
  "data": {
    "deposit_no": "D001",
    "channel": "bank",
    "currency": "CNY",
    "amount": 10000,
    "expire_at": "2026-05-27T12:00:00Z"
  }
}
```

出金 topic：

```json
{
  "event_id": "withdraw-event-001",
  "topic": "match_withdraw",
  "data": {
    "basket_no": "B001",
    "withdraw_no": "W001",
    "channel": "bank",
    "currency": "CNY",
    "target_amount": 50000,
    "expire_at": "2026-05-27T12:30:00Z"
  }
}
```
