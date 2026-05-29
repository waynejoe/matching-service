//go:build integration

package integration

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"matching-service/internal/biz"
	"matching-service/internal/conf"
	"matching-service/internal/data"
	"matching-service/pkg/model"
	"matching-service/pkg/toolbox/redisx"
)

// testEnv 保存集成测试共享资源。
type testEnv struct {
	cfg  *conf.Bootstrap      // cfg 是测试配置
	data *data.Data           // data 是数据库资源
	uc   *biz.MatchingUsecase // uc 是撮合用例
	lock *redisx.Lock         // lock 是 Redis 分片锁
	m    *biz.Metrics         // m 是运行指标
}

// TestPartialMatchSucceeded 验证出金超时后达到最低比例可以少发成交。
func TestPartialMatchSucceeded(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t, ctx)
	defer env.close()
	env.cleanup(t, "IT_PARTIAL_OK")

	expireAt := time.Now().Add(time.Hour)
	if err := env.uc.CreateBasket(ctx, &model.WithdrawBasket{
		BasketNo:     "IT_PARTIAL_OK_B",
		WithdrawNo:   "IT_PARTIAL_OK_W",
		Channel:      "IT_PARTIAL_OK",
		Currency:     "TST",
		TargetAmount: 1000,
		ExpireAt:     expireAt,
	}); err != nil {
		t.Fatalf("创建出金篮子失败: %v", err)
	}
	result, err := env.uc.SubmitDeposit(ctx, &model.DepositOrder{
		DepositNo: "IT_PARTIAL_OK_D",
		Channel:   "IT_PARTIAL_OK",
		Currency:  "TST",
		Amount:    300,
		ExpireAt:  expireAt.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("提交入金失败: %v", err)
	}
	if !result.Attached || result.Matched {
		t.Fatalf("期望先挂入未成交，实际 attached=%v matched=%v", result.Attached, result.Matched)
	}
	if _, err := env.uc.ExpireTimeouts(ctx, expireAt.Add(time.Second), 100); err != nil {
		t.Fatalf("触发超时失败: %v", err)
	}

	var record model.MatchRecord
	if err := env.data.DB.WithContext(ctx).Where("basket_no = ?", "IT_PARTIAL_OK_B").First(&record).Error; err != nil {
		t.Fatalf("查询撮合记录失败: %v", err)
	}
	if record.MatchedAmount != 300 || record.ShortAmount != 700 {
		t.Fatalf("期望成交 300 少发 700，实际成交 %d 少发 %d", record.MatchedAmount, record.ShortAmount)
	}
	assertBasketStatus(t, env, "IT_PARTIAL_OK_B", model.StatusMatched)
	assertDepositStatus(t, env, "IT_PARTIAL_OK_D", model.StatusMatched)
}

// TestPartialMatchFailed 验证出金超时后低于最低比例会过期失败。
func TestPartialMatchFailed(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t, ctx)
	defer env.close()
	env.cleanup(t, "IT_PARTIAL_FAIL")

	expireAt := time.Now().Add(time.Hour)
	if err := env.uc.CreateBasket(ctx, &model.WithdrawBasket{
		BasketNo:     "IT_PARTIAL_FAIL_B",
		WithdrawNo:   "IT_PARTIAL_FAIL_W",
		Channel:      "IT_PARTIAL_FAIL",
		Currency:     "TST",
		TargetAmount: 1000,
		ExpireAt:     expireAt,
	}); err != nil {
		t.Fatalf("创建出金篮子失败: %v", err)
	}
	if _, err := env.uc.SubmitDeposit(ctx, &model.DepositOrder{
		DepositNo: "IT_PARTIAL_FAIL_D",
		Channel:   "IT_PARTIAL_FAIL",
		Currency:  "TST",
		Amount:    299,
		ExpireAt:  expireAt.Add(time.Hour),
	}); err != nil {
		t.Fatalf("提交入金失败: %v", err)
	}
	if _, err := env.uc.ExpireTimeouts(ctx, expireAt.Add(time.Second), 100); err != nil {
		t.Fatalf("触发超时失败: %v", err)
	}

	var count int64
	env.data.DB.WithContext(ctx).Model(&model.MatchRecord{}).Where("basket_no = ?", "IT_PARTIAL_FAIL_B").Count(&count)
	if count != 0 {
		t.Fatalf("低于最低比例不应生成撮合记录，实际 %d", count)
	}
	assertBasketStatus(t, env, "IT_PARTIAL_FAIL_B", model.StatusExpired)
	assertDepositStatus(t, env, "IT_PARTIAL_FAIL_D", model.StatusExpired)
}

// TestOverRemainDepositPending 验证入金超过剩余金额时不会挂入篮子。
func TestOverRemainDepositPending(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t, ctx)
	defer env.close()
	env.cleanup(t, "IT_OVER_REMAIN")

	expireAt := time.Now().Add(time.Hour)
	if err := env.uc.CreateBasket(ctx, &model.WithdrawBasket{
		BasketNo:     "IT_OVER_REMAIN_B",
		WithdrawNo:   "IT_OVER_REMAIN_W",
		Channel:      "IT_OVER_REMAIN",
		Currency:     "TST",
		TargetAmount: 1000,
		ExpireAt:     expireAt,
	}); err != nil {
		t.Fatalf("创建出金篮子失败: %v", err)
	}
	if _, err := env.uc.SubmitDeposit(ctx, &model.DepositOrder{
		DepositNo: "IT_OVER_REMAIN_D1",
		Channel:   "IT_OVER_REMAIN",
		Currency:  "TST",
		Amount:    900,
		ExpireAt:  expireAt.Add(time.Hour),
	}); err != nil {
		t.Fatalf("提交第一笔入金失败: %v", err)
	}
	result, err := env.uc.SubmitDeposit(ctx, &model.DepositOrder{
		DepositNo: "IT_OVER_REMAIN_D2",
		Channel:   "IT_OVER_REMAIN",
		Currency:  "TST",
		Amount:    200,
		ExpireAt:  expireAt.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("提交第二笔入金失败: %v", err)
	}
	if !result.Pending || result.Attached || result.Matched {
		t.Fatalf("期望第二笔入金 pending，实际 pending=%v attached=%v matched=%v", result.Pending, result.Attached, result.Matched)
	}

	var basket model.WithdrawBasket
	if err := env.data.DB.WithContext(ctx).Where("basket_no = ?", "IT_OVER_REMAIN_B").First(&basket).Error; err != nil {
		t.Fatalf("查询篮子失败: %v", err)
	}
	if basket.CurrentAmount != 900 {
		t.Fatalf("超过剩余金额不应挂入，期望已凑 900，实际 %d", basket.CurrentAmount)
	}
	assertDepositStatus(t, env, "IT_OVER_REMAIN_D2", model.StatusWaiting)
}

// newTestEnv 创建集成测试环境。
func newTestEnv(t *testing.T, ctx context.Context) *testEnv {
	t.Helper()
	cfg, err := loadBootstrap(configPath(t))
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}
	dataLayer, dataCleanup, err := data.NewData(cfg)
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	t.Cleanup(dataCleanup)
	shardLock := redisx.NewShardLock(cfg)
	metric := biz.NewMetrics()
	usecase, err := biz.NewMatchingUsecase(ctx, cfg, dataLayer, shardLock, metric)
	if err != nil {
		t.Fatalf("创建撮合用例失败: %v", err)
	}
	return &testEnv{cfg: cfg, data: dataLayer, uc: usecase, lock: shardLock, m: metric}
}

// close 关闭集成测试环境资源。
func (e *testEnv) close() {
	_ = e.lock.Close()
	_ = e.data.Close()
}

// cleanup 清理指定前缀的测试数据。
func (e *testEnv) cleanup(t *testing.T, prefix string) {
	t.Helper()
	db := e.data.DB
	like := prefix + "%"
	statements := []string{
		"DELETE FROM match_record_deposit WHERE withdraw_no LIKE ? OR deposit_no LIKE ?",
		"DELETE FROM match_record WHERE basket_no LIKE ? OR withdraw_no LIKE ?",
		"DELETE FROM basket_deposit WHERE basket_no LIKE ? OR withdraw_no LIKE ? OR deposit_no LIKE ?",
		"DELETE FROM deposit_order WHERE deposit_no LIKE ? OR matched_basket_no LIKE ?",
		"DELETE FROM withdraw_basket WHERE basket_no LIKE ? OR withdraw_no LIKE ?",
		"DELETE FROM match_event_inbox WHERE biz_no LIKE ?",
		"DELETE FROM match_state_log WHERE biz_no LIKE ?",
	}
	args := [][]any{
		{like, like},
		{like, like},
		{like, like, like},
		{like, like},
		{like, like},
		{like},
		{like},
	}
	for i, statement := range statements {
		if err := db.Exec(statement, args[i]...).Error; err != nil {
			t.Fatalf("清理测试数据失败: %v", err)
		}
	}
}

// assertBasketStatus 断言篮子状态。
func assertBasketStatus(t *testing.T, env *testEnv, basketNo string, status int32) {
	t.Helper()
	var basket model.WithdrawBasket
	if err := env.data.DB.Where("basket_no = ?", basketNo).First(&basket).Error; err != nil {
		t.Fatalf("查询篮子失败: %v", err)
	}
	if basket.Status != status {
		t.Fatalf("篮子 %s 期望状态 %d，实际 %d", basketNo, status, basket.Status)
	}
}

// assertDepositStatus 断言入金状态。
func assertDepositStatus(t *testing.T, env *testEnv, depositNo string, status int32) {
	t.Helper()
	var deposit model.DepositOrder
	if err := env.data.DB.Where("deposit_no = ?", depositNo).First(&deposit).Error; err != nil {
		t.Fatalf("查询入金失败: %v", err)
	}
	if deposit.Status != status {
		t.Fatalf("入金 %s 期望状态 %d，实际 %d", depositNo, status, deposit.Status)
	}
}

// configPath 返回项目配置文件路径。
func configPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("读取当前文件路径失败")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "configs", "config.yaml")
}
