package biz

import (
	"testing"

	"matching-service/internal/model"
)

// TestMinCompleteAmount 验证最低成交金额按比例向上取整。
func TestMinCompleteAmount(t *testing.T) {
	uc := &MatchingUsecase{minCompleteRate: 3000}
	if got := uc.minCompleteAmount(1000); got != 300 {
		t.Fatalf("期望最低成交金额 300，实际 %d", got)
	}
	if got := uc.minCompleteAmount(1001); got != 301 {
		t.Fatalf("期望最低成交金额 301，实际 %d", got)
	}
}

// TestBuildPartialMatchResult 验证少发成交结果金额和差额。
func TestBuildPartialMatchResult(t *testing.T) {
	uc := &MatchingUsecase{}
	result := uc.buildPartialMatchResult(&model.WithdrawBasket{
		BasketNo:     "B1",
		WithdrawNo:   "W1",
		Channel:      "bank",
		Currency:     "CNY",
		TargetAmount: 1000,
	}, []*model.BasketDeposit{
		{DepositNo: "D1", Amount: 200},
		{DepositNo: "D2", Amount: 300},
	})
	if !result.Matched {
		t.Fatalf("期望少发成交结果为已成交")
	}
	if result.MatchedAmount != 500 {
		t.Fatalf("期望成交金额 500，实际 %d", result.MatchedAmount)
	}
	if result.ShortAmount != 500 {
		t.Fatalf("期望少发金额 500，实际 %d", result.ShortAmount)
	}
}
