# service

`service` 是对外接口适配层。

## 当前职责

- gRPC 接口适配，把 proto 请求转换成 `biz` 入参。
- RocketMQ 消息适配，把消息 body 转换成业务入参。
- 健康检查响应转换。
- 指标查询响应转换。

## 边界

- 这里不启动 gRPC、RocketMQ 或后台任务，启动入口放在 `internal/server`。
- 这里不写撮合算法，撮合算法放在 `internal/engine`。
- 这里不直接操作数据库，业务编排放在 `internal/biz`。
