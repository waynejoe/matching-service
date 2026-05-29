# pkg/toolbox

与 [server/toolbox](https://github.com/shengshi/backend/tree/master/toolbox) 对齐的 vendored 工具库，后续可独立成模块。

## 目录

| 包 | 说明 | 对标 server/toolbox |
|----|------|---------------------|
| `logx` | Kratos 日志封装 | logx |
| `kratox` | Sentry 等 Kratos 集成 | kratox |
| `mqx` | RocketMQ 5.x 生产/消费 | mqx |
| `idx` | 业务单号（时间戳+序号） | idx（简版；server 另有 snowflake / DB 发号） |
| `redisx` | Redis 客户端 + 分片分布式锁 | redisx |

## 使用

```go
import (
    "matching-service/pkg/toolbox/idx"
    "matching-service/pkg/toolbox/redisx"
)
```

- 单号：`idx.New("B")` / `idx.New("D")` / `idx.New("M")`
- 分片锁：Wire 注入 `*redisx.Lock`，或 `redisx.NewShardLockProvider(cfg)`

## 优先补充（相对 server/toolbox）

| 优先级 | 包 | 原因 |
|--------|-----|------|
| P0 | — | 当前撮合必需能力已覆盖（mqx / redisx / idx / logx / kratox） |
| P1 | `errorx` | 统一错误栈与 gRPC 错误码；与 `pkg/api/.../error_reason` 配套 |
| P1 | `kratox/log` | 统一 logger 构造（对齐 mome `kratox.NewLogger`） |
| P2 | `gormx` | GORM 日志级别与连接辅助（`data` 层已部分自管） |
| P2 | `helpx` | 上下文 trace/device 等（接 AWS X-Ray 时可补） |
| P3 | `httpx` | 出站 HTTP 客户端（当前无外部 HTTP 依赖） |
| P3 | `utils` | 时间/集合等小工具，按需拷贝 |

暂不需要：`i18n`、`search`、`geo`、`email`、`authz`、`claim`、`utmx` 等业务向包。

## 迁移说明

- 原 `pkg/idgen` → `pkg/toolbox/idx`
- 原 `pkg/lock` → `pkg/toolbox/redisx`（类型 `RedisLock` 更名为 `Lock`）
