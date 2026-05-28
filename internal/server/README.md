# server

`server` 是服务运行入口层。

## 当前职责

- 启动 gRPC 服务，并注册 `MatchingService`。
- 启动 RocketMQ 顺序消费。
- 启动超时扫描后台任务。
- 启动 Prometheus `/metrics` 指标服务。
- 检查 MySQL、Redis、RocketMQ 健康状态。

## 边界

- 这里只负责服务监听和后台任务启动。
- gRPC、MQ、Job 的请求转换放在 `internal/service`。
- 业务编排放在 `internal/biz`。
