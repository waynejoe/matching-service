package engine

import (
	"errors"
	"sort"
	"sync"

	"matching-service/internal/model"
)

// CommitFunc 是撮合结果持久化回调。
type CommitFunc func(result MatchResult) error

// Shard 是一个内存撮合分片。
// 一个分片建议由一个 goroutine 顺序处理，避免篮子并发竞争。
type Shard struct {
	mu              sync.Mutex          // mu 保证当前分片内的篮子修改是串行的
	Channel         string              // Channel 是分片负责的支付渠道
	Currency        string              // Currency 是分片负责的币种
	Baskets         BasketMap           // Baskets 保存活跃出金篮子
	NeedIndex       NeedIndex           // NeedIndex 按还差金额索引活跃篮子
	PendingDeposits map[string]*Deposit // PendingDeposits 保存暂时无法撮合的入金单
}

// NewShard 创建撮合分片。
func NewShard(channel, currency string) *Shard {
	return &Shard{
		Channel:         channel,
		Currency:        currency,
		Baskets:         make(BasketMap),
		NeedIndex:       NewBucketNeedIndex(),
		PendingDeposits: make(map[string]*Deposit),
	}
}

// AddBasket 把出金篮子加入分片。
func (s *Shard) AddBasket(basket *Basket) {
	if basket == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Baskets[basket.BasketNo] = basket
	s.NeedIndex.Add(basket)
}

// RemoveBasket 从分片中移除出金篮子。
func (s *Shard) RemoveBasket(basketNo string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	basket := s.Baskets[basketNo]
	if basket == nil {
		return
	}
	s.NeedIndex.Remove(basket)
	delete(s.Baskets, basketNo)
}

// ReleaseDeposit 从活跃篮子中释放一笔入金。
func (s *Shard) ReleaseDeposit(basketNo, depositNo string, amount int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	basket := s.Baskets[basketNo]
	if basket == nil {
		return
	}
	oldNeed := basket.NeedAmount()
	if basket.CurrentAmount >= amount {
		basket.CurrentAmount -= amount
	} else {
		basket.CurrentAmount = 0
	}
	basket.DepositNos = removeDepositNo(basket.DepositNos, depositNo)
	s.NeedIndex.Update(oldNeed, basket)
}

// MatchDeposit 处理一笔入金撮合。
func (s *Shard) MatchDeposit(deposit *Deposit) MatchResult {
	result, _ := s.MatchDepositWithCommit(deposit, nil)
	return result
}

// MatchDepositWithCommit 先计算撮合结果，提交成功后再修改内存。
func (s *Shard) MatchDepositWithCommit(deposit *Deposit, commit CommitFunc) (MatchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if deposit == nil {
		return MatchResult{FailedReason: "入金为空"}, nil
	}
	if deposit.Channel != s.Channel || deposit.Currency != s.Currency {
		return MatchResult{FailedReason: "入金渠道或币种不属于当前分片"}, nil
	}
	if deposit.Status != model.StatusWaiting {
		return MatchResult{FailedReason: "入金状态不是待撮合"}, nil
	}
	if basket := s.bestCompletable(deposit.Amount); basket != nil {
		result := s.buildCompleteResult(basket, deposit)
		if err := runCommit(commit, result); err != nil {
			return result, err
		}
		s.applyComplete(basket, deposit)
		return result, nil
	}
	if basket := s.bestAcceptable(deposit.Amount); basket != nil {
		result := s.buildWaitResult(basket, deposit)
		if err := runCommit(commit, result); err != nil {
			return result, err
		}
		s.applyWait(basket, deposit)
		return result, nil
	}
	s.PendingDeposits[deposit.DepositNo] = deposit
	return MatchResult{Pending: true, FailedReason: "没有合适篮子"}, nil
}

// bestCompletable 选择能被当前入金刚好凑满的最优篮子。
func (s *Shard) bestCompletable(amount int64) *Basket {
	candidates := s.NeedIndex.FindCompletable(amount, 32)
	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if !left.ExpireAt.Equal(right.ExpireAt) {
			return left.ExpireAt.Before(right.ExpireAt)
		}
		return len(left.DepositNos) < len(right.DepositNos)
	})
	if len(candidates) == 0 {
		return nil
	}
	return candidates[0]
}

// bestAcceptable 选择能先接收当前入金但不会直接凑满的最优篮子。
func (s *Shard) bestAcceptable(amount int64) *Basket {
	candidates := s.NeedIndex.FindAcceptable(amount, 32)
	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		leftNeedAfter := left.TargetAmount - (left.CurrentAmount + amount)
		rightNeedAfter := right.TargetAmount - (right.CurrentAmount + amount)
		if leftNeedAfter != rightNeedAfter {
			return leftNeedAfter < rightNeedAfter
		}
		if !left.ExpireAt.Equal(right.ExpireAt) {
			return left.ExpireAt.Before(right.ExpireAt)
		}
		return len(left.DepositNos) < len(right.DepositNos)
	})
	if len(candidates) == 0 {
		return nil
	}
	return candidates[0]
}

// buildCompleteResult 生成刚好凑满的撮合结果。
func (s *Shard) buildCompleteResult(basket *Basket, deposit *Deposit) MatchResult {
	matchedAmount := basket.CurrentAmount + deposit.Amount
	depositNos := append([]string(nil), basket.DepositNos...)
	depositNos = append(depositNos, deposit.DepositNo)
	return MatchResult{
		Matched:       true,
		Attached:      true,
		BasketNo:      basket.BasketNo,
		WithdrawNo:    basket.WithdrawNo,
		Channel:       basket.Channel,
		Currency:      basket.Currency,
		DepositNos:    depositNos,
		TargetAmount:  basket.TargetAmount,
		MatchedAmount: matchedAmount,
		ShortAmount:   0,
		FailedReason:  "",
	}
}

// buildWaitResult 生成挂入等待的撮合结果。
func (s *Shard) buildWaitResult(basket *Basket, deposit *Deposit) MatchResult {
	matchedAmount := basket.CurrentAmount + deposit.Amount
	depositNos := append([]string(nil), basket.DepositNos...)
	depositNos = append(depositNos, deposit.DepositNo)
	return MatchResult{
		Attached:      true,
		BasketNo:      basket.BasketNo,
		WithdrawNo:    basket.WithdrawNo,
		Channel:       basket.Channel,
		Currency:      basket.Currency,
		DepositNos:    depositNos,
		TargetAmount:  basket.TargetAmount,
		MatchedAmount: matchedAmount,
	}
}

// applyComplete 把凑满结果应用到内存池。
func (s *Shard) applyComplete(basket *Basket, deposit *Deposit) {
	basket.AttachDeposit(deposit)
	basket.Status = model.StatusMatched
	s.NeedIndex.Remove(basket)
	delete(s.Baskets, basket.BasketNo)
}

// applyWait 把挂入等待结果应用到内存池。
func (s *Shard) applyWait(basket *Basket, deposit *Deposit) {
	oldNeed := basket.NeedAmount()
	basket.AttachDeposit(deposit)
	s.NeedIndex.Update(oldNeed, basket)
}

// runCommit 执行撮合结果提交回调。
func runCommit(commit CommitFunc, result MatchResult) error {
	if commit == nil {
		return nil
	}
	if !result.Attached {
		return errors.New("未挂入结果不需要提交")
	}
	return commit(result)
}

// removeDepositNo 从入金单号列表中移除指定单号。
func removeDepositNo(in []string, depositNo string) []string {
	out := in[:0]
	for _, item := range in {
		if item != depositNo {
			out = append(out, item)
		}
	}
	return out
}
