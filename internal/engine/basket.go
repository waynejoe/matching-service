package engine

import (
	"time"

	"matching-service/internal/model"
)

// Basket 是撮合引擎使用的内存出金篮子。
// 它由 model.WithdrawBasket 重建，并补充已经挂入的入金单号。
type Basket struct {
	BasketNo      string    // BasketNo 是篮子单号
	WithdrawNo    string    // WithdrawNo 是出金单号
	Channel       string    // Channel 是支付渠道
	Currency      string    // Currency 是币种
	TargetAmount  int64     // TargetAmount 是出金目标金额
	CurrentAmount int64     // CurrentAmount 是当前已经凑到的金额
	DepositNos    []string  // DepositNos 是已经挂入篮子的入金单号列表
	Status        int32     // Status 是篮子当前状态
	ExpireAt      time.Time // ExpireAt 是篮子过期时间
}

// NewBasketFromModel 根据数据库模型创建内存篮子。
func NewBasketFromModel(in *model.WithdrawBasket) *Basket {
	return &Basket{
		BasketNo:      in.BasketNo,
		WithdrawNo:    in.WithdrawNo,
		Channel:       in.Channel,
		Currency:      in.Currency,
		TargetAmount:  in.TargetAmount,
		CurrentAmount: in.CurrentAmount,
		Status:        in.Status,
		ExpireAt:      in.ExpireAt,
	}
}

// NeedAmount 返回篮子还差多少钱才能达到出金目标。
func (b *Basket) NeedAmount() int64 {
	need := b.TargetAmount - b.CurrentAmount
	if need < 0 {
		return 0
	}
	return need
}

// RoomAmount 返回篮子当前最多还能接收多少钱。
func (b *Basket) RoomAmount() int64 {
	room := b.TargetAmount - b.CurrentAmount
	if room < 0 {
		return 0
	}
	return room
}

// CanAccept 判断篮子是否能接收这笔入金金额。
func (b *Basket) CanAccept(amount int64) bool {
	return b.Status == model.StatusWaiting && amount <= b.RoomAmount()
}

// CanComplete 判断这笔入金是否能让篮子直接撮合成功。
func (b *Basket) CanComplete(amount int64) bool {
	next := b.CurrentAmount + amount
	return b.Status == model.StatusWaiting &&
		next == b.TargetAmount
}

// AttachDeposit 把一笔入金放入篮子。
func (b *Basket) AttachDeposit(deposit *Deposit) {
	b.CurrentAmount += deposit.Amount
	b.DepositNos = append(b.DepositNos, deposit.DepositNo)
}
