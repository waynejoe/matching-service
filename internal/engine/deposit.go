package engine

import (
	"time"

	"matching-service/internal/model"
)

// Deposit 是撮合引擎使用的内存入金单。
// 它是 model.DepositOrder 的轻量投影。
type Deposit struct {
	DepositNo string    // DepositNo 是入金单号
	Channel   string    // Channel 是支付渠道
	Currency  string    // Currency 是币种
	Amount    int64     // Amount 是入金金额，使用最小货币单位
	Status    int32     // Status 是入金单状态
	ExpireAt  time.Time // ExpireAt 是入金单过期时间
}

// NewDepositFromModel 根据数据库模型创建内存入金单。
func NewDepositFromModel(in *model.DepositOrder) *Deposit {
	return &Deposit{
		DepositNo: in.DepositNo,
		Channel:   in.Channel,
		Currency:  in.Currency,
		Amount:    in.Amount,
		Status:    in.Status,
		ExpireAt:  in.ExpireAt,
	}
}
