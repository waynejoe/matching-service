package biz

import "sync/atomic"

// Metrics 保存服务运行期内存指标。
type Metrics struct {
	depositConsumeSuccess  atomic.Int64 // depositConsumeSuccess 是入金消息消费成功数
	depositConsumeFailed   atomic.Int64 // depositConsumeFailed 是入金消息消费失败数
	withdrawConsumeSuccess atomic.Int64 // withdrawConsumeSuccess 是出金消息消费成功数
	withdrawConsumeFailed  atomic.Int64 // withdrawConsumeFailed 是出金消息消费失败数
	matchSuccess           atomic.Int64 // matchSuccess 是撮合成功数
	shortMatchSuccess      atomic.Int64 // shortMatchSuccess 是少发撮合成功数
	expiredWaitingDeposits atomic.Int64 // expiredWaitingDeposits 是过期待撮合入金数
	expiredLockedDeposits  atomic.Int64 // expiredLockedDeposits 是过期已挂入入金数
	expiredBaskets         atomic.Int64 // expiredBaskets 是过期篮子数
	eventFailed            atomic.Int64 // eventFailed 是事件处理失败数
	eventRetried           atomic.Int64 // eventRetried 是失败事件重试数
}

// Snapshot 是指标快照。
type Snapshot struct {
	DepositConsumeSuccess  int64 // DepositConsumeSuccess 是入金消息消费成功数
	DepositConsumeFailed   int64 // DepositConsumeFailed 是入金消息消费失败数
	WithdrawConsumeSuccess int64 // WithdrawConsumeSuccess 是出金消息消费成功数
	WithdrawConsumeFailed  int64 // WithdrawConsumeFailed 是出金消息消费失败数
	MatchSuccess           int64 // MatchSuccess 是撮合成功数
	ShortMatchSuccess      int64 // ShortMatchSuccess 是少发撮合成功数
	ExpiredWaitingDeposits int64 // ExpiredWaitingDeposits 是过期待撮合入金数
	ExpiredLockedDeposits  int64 // ExpiredLockedDeposits 是过期已挂入入金数
	ExpiredBaskets         int64 // ExpiredBaskets 是过期篮子数
	EventFailed            int64 // EventFailed 是事件处理失败数
	EventRetried           int64 // EventRetried 是失败事件重试数
}

// NewMetrics 创建运行指标。
func NewMetrics() *Metrics {
	return &Metrics{}
}

// IncDepositConsumeSuccess 增加入金消息消费成功数。
func (m *Metrics) IncDepositConsumeSuccess() {
	m.depositConsumeSuccess.Add(1)
}

// IncDepositConsumeFailed 增加入金消息消费失败数。
func (m *Metrics) IncDepositConsumeFailed() {
	m.depositConsumeFailed.Add(1)
}

// IncWithdrawConsumeSuccess 增加出金消息消费成功数。
func (m *Metrics) IncWithdrawConsumeSuccess() {
	m.withdrawConsumeSuccess.Add(1)
}

// IncWithdrawConsumeFailed 增加出金消息消费失败数。
func (m *Metrics) IncWithdrawConsumeFailed() {
	m.withdrawConsumeFailed.Add(1)
}

// IncMatchSuccess 增加撮合成功数。
func (m *Metrics) IncMatchSuccess() {
	m.matchSuccess.Add(1)
}

// IncShortMatchSuccess 增加少发撮合成功数。
func (m *Metrics) IncShortMatchSuccess() {
	m.shortMatchSuccess.Add(1)
}

// AddExpiredWaitingDeposits 增加过期待撮合入金数。
func (m *Metrics) AddExpiredWaitingDeposits(n int) {
	m.expiredWaitingDeposits.Add(int64(n))
}

// AddExpiredLockedDeposits 增加过期已挂入入金数。
func (m *Metrics) AddExpiredLockedDeposits(n int) {
	m.expiredLockedDeposits.Add(int64(n))
}

// AddExpiredBaskets 增加过期篮子数。
func (m *Metrics) AddExpiredBaskets(n int) {
	m.expiredBaskets.Add(int64(n))
}

// IncEventFailed 增加事件处理失败数。
func (m *Metrics) IncEventFailed() {
	m.eventFailed.Add(1)
}

// IncEventRetried 增加失败事件重试数。
func (m *Metrics) IncEventRetried() {
	m.eventRetried.Add(1)
}

// Snapshot 返回当前指标快照。
func (m *Metrics) Snapshot() Snapshot {
	return Snapshot{
		DepositConsumeSuccess:  m.depositConsumeSuccess.Load(),
		DepositConsumeFailed:   m.depositConsumeFailed.Load(),
		WithdrawConsumeSuccess: m.withdrawConsumeSuccess.Load(),
		WithdrawConsumeFailed:  m.withdrawConsumeFailed.Load(),
		MatchSuccess:           m.matchSuccess.Load(),
		ShortMatchSuccess:      m.shortMatchSuccess.Load(),
		ExpiredWaitingDeposits: m.expiredWaitingDeposits.Load(),
		ExpiredLockedDeposits:  m.expiredLockedDeposits.Load(),
		ExpiredBaskets:         m.expiredBaskets.Load(),
		EventFailed:            m.eventFailed.Load(),
		EventRetried:           m.eventRetried.Load(),
	}
}
