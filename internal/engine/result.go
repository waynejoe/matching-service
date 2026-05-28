package engine

// MatchResult 是持久化之前的内存撮合结果。
type MatchResult struct {
	Matched       bool     // Matched 表示是否撮合成功
	Attached      bool     // Attached 表示入金是否已挂入篮子
	BasketNo      string   // BasketNo 是撮合成功的篮子单号
	WithdrawNo    string   // WithdrawNo 是撮合成功的出金单号
	Channel       string   // Channel 是支付渠道
	Currency      string   // Currency 是币种
	DepositNos    []string // DepositNos 是参与本次撮合的入金单号列表
	TargetAmount  int64    // TargetAmount 是篮子目标金额
	MatchedAmount int64    // MatchedAmount 是实际撮合金额
	ShortAmount   int64    // ShortAmount 是少发金额
	Pending       bool     // Pending 表示入金是否进入待处理状态
	FailedReason  string   // FailedReason 是撮合失败原因
}
