package engine

import (
	"errors"
	"sync"
	"testing"
	"time"

	"matching-service/internal/model"
)

// TestMatchDepositCompleteBasket 验证入金可以直接凑满篮子。
func TestMatchDepositCompleteBasket(t *testing.T) {
	shard := NewShard("bank", "CNY")
	shard.AddBasket(&Basket{
		BasketNo:      "B1",
		WithdrawNo:    "W1",
		Channel:       "bank",
		Currency:      "CNY",
		TargetAmount:  50000,
		CurrentAmount: 45000,
		Status:        model.StatusWaiting,
		ExpireAt:      time.Now().Add(time.Minute),
	})

	result := shard.MatchDeposit(&Deposit{
		DepositNo: "D1",
		Channel:   "bank",
		Currency:  "CNY",
		Amount:    5000,
		Status:    model.StatusWaiting,
	})

	if !result.Matched {
		t.Fatalf("期望撮合成功")
	}
	if result.BasketNo != "B1" {
		t.Fatalf("期望篮子 B1，实际 %s", result.BasketNo)
	}
	if result.MatchedAmount != 50000 {
		t.Fatalf("期望撮合金额 50000，实际 %d", result.MatchedAmount)
	}
	if result.ShortAmount != 0 {
		t.Fatalf("期望少发金额 0，实际 %d", result.ShortAmount)
	}
	if _, ok := shard.Baskets["B1"]; ok {
		t.Fatalf("撮合成功后篮子应从活跃池移除")
	}
}

// TestMatchDepositAttachAndWait 验证入金可以先挂入未凑满的篮子。
func TestMatchDepositAttachAndWait(t *testing.T) {
	shard := NewShard("bank", "CNY")
	shard.AddBasket(&Basket{
		BasketNo:     "B1",
		WithdrawNo:   "W1",
		Channel:      "bank",
		Currency:     "CNY",
		TargetAmount: 50000,
		Status:       model.StatusWaiting,
		ExpireAt:     time.Now().Add(time.Minute),
	})

	result := shard.MatchDeposit(&Deposit{
		DepositNo: "D1",
		Channel:   "bank",
		Currency:  "CNY",
		Amount:    9000,
		Status:    model.StatusWaiting,
	})

	if result.Matched {
		t.Fatalf("不应该直接撮合成功")
	}
	if !result.Attached {
		t.Fatalf("期望入金先挂入篮子")
	}
	if shard.Baskets["B1"].CurrentAmount != 9000 {
		t.Fatalf("期望已凑金额 9000，实际 %d", shard.Baskets["B1"].CurrentAmount)
	}
}

// TestMatchDepositRejectOverTargetAmount 验证超过目标金额时不能挂入篮子。
func TestMatchDepositRejectOverTargetAmount(t *testing.T) {
	shard := NewShard("bank", "CNY")
	shard.AddBasket(&Basket{
		BasketNo:      "B1",
		WithdrawNo:    "W1",
		Channel:       "bank",
		Currency:      "CNY",
		TargetAmount:  50000,
		CurrentAmount: 49000,
		Status:        model.StatusWaiting,
		ExpireAt:      time.Now().Add(time.Minute),
	})

	result := shard.MatchDeposit(&Deposit{
		DepositNo: "D1",
		Channel:   "bank",
		Currency:  "CNY",
		Amount:    2000,
		Status:    model.StatusWaiting,
	})

	if result.Matched || result.Attached {
		t.Fatalf("超过目标金额时不应挂入或撮合")
	}
	if !result.Pending {
		t.Fatalf("未匹配入金应进入待处理")
	}
}

// TestMatchDepositPreferExactMatch 验证多个篮子可选时优先选择刚好凑满的篮子。
func TestMatchDepositPreferExactMatch(t *testing.T) {
	shard := NewShard("bank", "CNY")
	now := time.Now()
	shard.AddBasket(&Basket{
		BasketNo:      "B1",
		WithdrawNo:    "W1",
		Channel:       "bank",
		Currency:      "CNY",
		TargetAmount:  50000,
		CurrentAmount: 43000,
		Status:        model.StatusWaiting,
		ExpireAt:      now.Add(time.Minute),
	})
	shard.AddBasket(&Basket{
		BasketNo:      "B2",
		WithdrawNo:    "W2",
		Channel:       "bank",
		Currency:      "CNY",
		TargetAmount:  50000,
		CurrentAmount: 44000,
		Status:        model.StatusWaiting,
		ExpireAt:      now.Add(2 * time.Minute),
	})

	result := shard.MatchDeposit(&Deposit{
		DepositNo: "D1",
		Channel:   "bank",
		Currency:  "CNY",
		Amount:    6000,
		Status:    model.StatusWaiting,
	})

	if result.BasketNo != "B2" {
		t.Fatalf("期望选择刚好凑满的 B2，实际 %s", result.BasketNo)
	}
	if result.ShortAmount != 0 {
		t.Fatalf("期望少发金额 0，实际 %d", result.ShortAmount)
	}
}

// TestMatchDepositConcurrentOneBasket 验证并发入金时同一个篮子只能被成交一次。
func TestMatchDepositConcurrentOneBasket(t *testing.T) {
	shard := NewShard("bank", "CNY")
	shard.AddBasket(&Basket{
		BasketNo:     "B1",
		WithdrawNo:   "W1",
		Channel:      "bank",
		Currency:     "CNY",
		TargetAmount: 10000,
		Status:       model.StatusWaiting,
		ExpireAt:     time.Now().Add(time.Minute),
	})

	var wg sync.WaitGroup
	results := make(chan MatchResult, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results <- shard.MatchDeposit(&Deposit{
				DepositNo: "D" + string(rune('1'+i)),
				Channel:   "bank",
				Currency:  "CNY",
				Amount:    10000,
				Status:    model.StatusWaiting,
			})
		}(i)
	}
	wg.Wait()
	close(results)

	matchedCount := 0
	for result := range results {
		if result.Matched {
			matchedCount++
		}
	}
	if matchedCount != 1 {
		t.Fatalf("期望只成交 1 次，实际成交 %d 次", matchedCount)
	}
}

// TestMatchDepositCommitFailedKeepMemory 验证提交失败时内存池不发生变化。
func TestMatchDepositCommitFailedKeepMemory(t *testing.T) {
	shard := NewShard("bank", "CNY")
	shard.AddBasket(&Basket{
		BasketNo:     "B1",
		WithdrawNo:   "W1",
		Channel:      "bank",
		Currency:     "CNY",
		TargetAmount: 10000,
		Status:       model.StatusWaiting,
		ExpireAt:     time.Now().Add(time.Minute),
	})

	_, err := shard.MatchDepositWithCommit(&Deposit{
		DepositNo: "D1",
		Channel:   "bank",
		Currency:  "CNY",
		Amount:    10000,
		Status:    model.StatusWaiting,
	}, func(result MatchResult) error {
		return errors.New("提交失败")
	})

	if err == nil {
		t.Fatalf("期望返回提交失败")
	}
	if shard.Baskets["B1"].CurrentAmount != 0 {
		t.Fatalf("提交失败时内存金额不应变化")
	}
	if shard.Baskets["B1"].Status != model.StatusWaiting {
		t.Fatalf("提交失败时篮子状态不应变化")
	}
}
