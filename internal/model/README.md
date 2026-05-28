# model

数据库模型层，只放和 MySQL 表一一对应的结构体。

## 核心关系

- `DepositOrder`：入金单，一笔入金只能进入一个篮子。
- `WithdrawBasket`：出金篮子，一个篮子可以接收多笔入金。
- `BasketDeposit`：篮子入金明细，记录篮子里已经挂入的入金单。
- `MatchRecord`：撮合结果，一个出金篮子对应多笔入金。
- `StateLog`：状态流水，用于审计。
- `EventInbox`：RocketMQ 消费幂等记录。

## 边界

- 这里的结构体带 `gorm` 标签。
- 内存撮合结构不要放这里，放到 `internal/engine`。
