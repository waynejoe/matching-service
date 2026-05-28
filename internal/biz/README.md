# biz

业务用例层，负责把内存撮合引擎和数据库事务串起来。

## 当前职责

- 创建出金篮子。
- 从数据库加载等待撮合的篮子。
- 提交入金并调用撮合引擎。
- 把撮合结果落库。
- 维护撮合、消费、过期、重试等运行指标。

## 边界

- 撮合算法放在 `internal/engine`。
- 数据库读写放在 `internal/data`。
- RocketMQ 消费启动放在 `internal/server`，消息处理入口放在 `internal/service`。
- gRPC 接口适配放在 `internal/service`。
- Prometheus 指标监听放在 `internal/server`。
