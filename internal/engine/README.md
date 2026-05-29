# engine

撮合引擎内存结构层，只放高性能撮合需要的内存结构和索引。

## 核心关系

- `Deposit`：内存入金单，是 `model.DepositOrder` 的轻量投影。
- `Basket`：内存出金篮子，是 `model.WithdrawBasket` 的轻量投影。
- `NeedIndex`：按“还差金额”查找候选篮子。
- `Shard`：撮合分片，一个分片建议由一个 goroutine 顺序处理。
- `MatchResult`：持久化之前的内存撮合结果。

## 边界

- 这里的结构体不带 `gorm` 标签。
- 服务重启后，内存结构可以从 `pkg/model` 的数据库模型重建。
