package model

// 通用状态
const (
	StatusWaiting  int32 = 1 // 等待撮合或处理
	StatusLocked   int32 = 2 // 已锁定，不能继续参与普通撮合
	StatusMatched  int32 = 3 // 已撮合
	StatusFailed   int32 = 4 // 失败
	StatusExpired  int32 = 5 // 已过期
	StatusCanceled int32 = 6 // 已取消
)

// 篮子入金明细状态
const (
	BasketDepositStatusAttached int32 = 1 // 已挂入篮子
	BasketDepositStatusMatched  int32 = 2 // 已形成撮合结果
	BasketDepositStatusReleased int32 = 3 // 已释放
)

// 撮合结果状态
const (
	MatchStatusCreated     int32 = 1 // 已创建
	MatchStatusPaymentSent int32 = 2 // 已发送支付
	MatchStatusPaying      int32 = 3 // 支付中
	MatchStatusSucceeded   int32 = 4 // 支付成功
	MatchStatusFailed      int32 = 5 // 支付失败
)

// RocketMQ 消费幂等状态
const (
	EventStatusProcessing int32 = 1 // 处理中
	EventStatusSucceeded  int32 = 2 // 成功
	EventStatusFailed     int32 = 3 // 失败
)
