# model

撮合域 GORM 模型，与 MySQL 表一一对应，供多服务通过 `matching-service/pkg/model` 引用。

## 核心关系

- `DepositOrder`：入金单，一笔入金只能进入一个篮子。
- `WithdrawBasket`：出金篮子，一个篮子可接收多笔入金。
- `BasketDeposit`：篮子入金明细。
- `MatchRecord`：撮合主记录。
- `StateLog`：状态流水。
- `EventInbox`：RocketMQ 消费幂等。

## 边界

- 结构体带 `gorm` 标签，由表结构生成或维护 `*.gen.go`。
- 内存撮合结构在 `internal/engine`，不放本包。

## 多服务引用

```go
import "matching-service/pkg/model"
```

后续独立模块时可迁为 `github.com/<org>/syntra/pkg/model`。
