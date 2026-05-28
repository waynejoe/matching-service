package biz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"matching-service/internal/conf"
	"matching-service/internal/data"
	"matching-service/internal/engine"
	"matching-service/internal/model"
	"matching-service/pkg/idgen"
	"matching-service/pkg/lock"
)

var errBasketChanged = errors.New("篮子状态已变化")

// ExpireResult 是超时处理结果。
type ExpireResult struct {
	WaitingDeposits int // WaitingDeposits 是过期的待撮合入金数量
	LockedDeposits  int // LockedDeposits 是过期的已挂入入金数量
	Baskets         int // Baskets 是过期的出金篮子数量
}

// MatchDetail 是撮合结果查询详情。
type MatchDetail struct {
	Record   *model.MatchRecord          // Record 是撮合主记录
	Deposits []*model.MatchRecordDeposit // Deposits 是撮合入金明细
}

// MatchingUsecase 负责编排撮合业务。
type MatchingUsecase struct {
	data            *data.Data               // data 是数据层入口
	eventRepo       *data.EventInboxRepo     // eventRepo 是事件幂等仓库
	logRepo         *data.StateLogRepo       // logRepo 是状态流水仓库
	locker          ShardLocker              // locker 是跨实例分片锁
	lockTTL         time.Duration            // lockTTL 是分片锁过期时间
	minCompleteRate int64                    // minCompleteRate 是最低成交比例
	metrics         *Metrics                 // metrics 是运行指标
	shards          map[string]*engine.Shard // shards 保存内存撮合分片
}

// ShardLocker 是分片锁接口。
type ShardLocker interface {
	WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error
}

// NewMatchingUsecase 根据配置创建撮合用例并恢复活跃篮子。
func NewMatchingUsecase(ctx context.Context, cfg *conf.Bootstrap, d *data.Data, shardLock *lock.RedisLock, metric *Metrics) (*MatchingUsecase, error) {
	uc := &MatchingUsecase{
		data:      d,
		eventRepo: data.NewEventInboxRepo(d),
		logRepo:   data.NewStateLogRepo(d),
		metrics:   metric,
		shards:    make(map[string]*engine.Shard),
	}
	uc.SetShardLocker(shardLock, time.Duration(cfg.Match.ShardLockTTLSecond)*time.Second)
	uc.minCompleteRate = cfg.Match.MinCompleteRate
	if err := uc.RecoverActiveBaskets(ctx, cfg.Match.BasketLimit); err != nil {
		return nil, err
	}
	return uc, nil
}

// SetShardLocker 设置跨实例分片锁。
func (uc *MatchingUsecase) SetShardLocker(locker ShardLocker, ttl time.Duration) {
	uc.locker = locker
	uc.lockTTL = ttl
}

// CreateBasket 创建出金篮子并加入内存分片。
func (uc *MatchingUsecase) CreateBasket(ctx context.Context, basket *model.WithdrawBasket) error {
	if basket == nil {
		return errors.New("出金篮子为空")
	}
	if err := validateBasket(basket); err != nil {
		return err
	}
	return uc.withShardLock(ctx, basket.Channel, basket.Currency, func() error {
		return uc.createBasketLocked(ctx, basket)
	})
}

// createBasketLocked 在分片锁内创建出金篮子。
func (uc *MatchingUsecase) createBasketLocked(ctx context.Context, basket *model.WithdrawBasket) error {
	if basket.BasketNo == "" {
		basket.BasketNo = idgen.New("B")
	}
	if basket.Status == 0 {
		basket.Status = model.StatusWaiting
	}
	err := uc.data.Transaction(ctx, func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Create(basket).Error; err != nil {
			return err
		}
		return createStateLogTx(ctx, tx, basket.BasketNo, "basket", nil, basket.Status, "创建出金篮子", "biz")
	})
	if err != nil {
		return err
	}
	uc.getShard(basket.Channel, basket.Currency).AddBasket(engine.NewBasketFromModel(basket))
	return nil
}

// LoadWaitingBaskets 从数据库加载等待撮合的篮子到内存分片。
func (uc *MatchingUsecase) LoadWaitingBaskets(ctx context.Context, channel, currency string, limit int) error {
	var baskets []*model.WithdrawBasket
	if err := uc.data.DB.WithContext(ctx).
		Where("channel = ? AND currency = ? AND status = ?", channel, currency, model.StatusWaiting).
		Order("expire_at ASC, id ASC").
		Limit(limit).
		Find(&baskets).Error; err != nil {
		return err
	}
	for _, basket := range baskets {
		if err := uc.restoreBasket(ctx, basket); err != nil {
			return err
		}
	}
	return nil
}

// RecoverActiveBaskets 从数据库恢复全部活跃篮子到内存分片。
func (uc *MatchingUsecase) RecoverActiveBaskets(ctx context.Context, limit int) error {
	var baskets []*model.WithdrawBasket
	query := uc.data.DB.WithContext(ctx).
		Where("status = ?", model.StatusWaiting).
		Order("channel ASC, currency ASC, expire_at ASC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&baskets).Error; err != nil {
		return err
	}
	for _, basket := range baskets {
		if err := uc.restoreBasket(ctx, basket); err != nil {
			return err
		}
	}
	return nil
}

// SubmitDeposit 提交入金并尝试撮合。
func (uc *MatchingUsecase) SubmitDeposit(ctx context.Context, deposit *model.DepositOrder) (engine.MatchResult, error) {
	if deposit == nil {
		return engine.MatchResult{}, errors.New("入金单为空")
	}
	normalizeDeposit(deposit)
	if err := validateDeposit(deposit); err != nil {
		return engine.MatchResult{}, err
	}
	var result engine.MatchResult
	err := uc.withShardLock(ctx, deposit.Channel, deposit.Currency, func() error {
		var err error
		result, err = uc.submitDepositLocked(ctx, deposit)
		return err
	})
	return result, err
}

// GetDeposit 查询入金单。
func (uc *MatchingUsecase) GetDeposit(ctx context.Context, depositNo string) (*model.DepositOrder, error) {
	return data.NewDepositRepo(uc.data).GetByNo(ctx, depositNo)
}

// GetBasket 查询出金篮子。
func (uc *MatchingUsecase) GetBasket(ctx context.Context, basketNo string) (*model.WithdrawBasket, error) {
	return data.NewBasketRepo(uc.data).GetByNo(ctx, basketNo)
}

// GetMatch 查询撮合结果和入金明细。
func (uc *MatchingUsecase) GetMatch(ctx context.Context, matchNo, basketNo, withdrawNo string) (*MatchDetail, error) {
	repo := data.NewMatchRecordRepo(uc.data)
	var record *model.MatchRecord
	var err error
	switch {
	case matchNo != "":
		record, err = repo.GetByNo(ctx, matchNo)
	case basketNo != "":
		record, err = repo.GetByBasketNo(ctx, basketNo)
	case withdrawNo != "":
		record, err = repo.GetByWithdrawNo(ctx, withdrawNo)
	default:
		return nil, errors.New("撮合查询条件为空")
	}
	if err != nil {
		return nil, err
	}
	deposits, err := repo.ListDeposits(ctx, record.MatchNo)
	if err != nil {
		return nil, err
	}
	return &MatchDetail{Record: record, Deposits: deposits}, nil
}

// Metrics 返回当前运行指标快照。
func (uc *MatchingUsecase) Metrics() Snapshot {
	return uc.metrics.Snapshot()
}

// submitDepositLocked 在分片锁内提交入金并尝试撮合。
func (uc *MatchingUsecase) submitDepositLocked(ctx context.Context, deposit *model.DepositOrder) (engine.MatchResult, error) {
	if deposit.Status == 0 {
		deposit.Status = model.StatusWaiting
	}
	if err := uc.saveDepositIfNeeded(ctx, deposit); err != nil {
		return engine.MatchResult{}, err
	}
	result, err := uc.getShard(deposit.Channel, deposit.Currency).MatchDepositWithCommit(
		engine.NewDepositFromModel(deposit),
		func(result engine.MatchResult) error {
			return uc.persistMatchResult(ctx, deposit, result)
		},
	)
	if err != nil {
		return engine.MatchResult{}, err
	}
	return result, nil
}

// ProcessDepositEvent 幂等处理入金事件。
func (uc *MatchingUsecase) ProcessDepositEvent(ctx context.Context, eventID, topic string, deposit *model.DepositOrder) (engine.MatchResult, bool, error) {
	if deposit == nil {
		return engine.MatchResult{}, false, errors.New("入金单为空")
	}
	canProcess, err := uc.beginEvent(ctx, eventID, topic, deposit.DepositNo)
	if err != nil || !canProcess {
		return engine.MatchResult{}, false, err
	}
	result, err := uc.SubmitDeposit(ctx, deposit)
	if err != nil {
		_ = uc.eventRepo.MarkFailed(ctx, eventID, err.Error())
		uc.metrics.IncEventFailed()
		return engine.MatchResult{}, true, err
	}
	return result, true, uc.eventRepo.MarkSucceeded(ctx, eventID)
}

// ProcessWithdrawEvent 幂等处理出金事件。
func (uc *MatchingUsecase) ProcessWithdrawEvent(ctx context.Context, eventID, topic string, basket *model.WithdrawBasket) (bool, error) {
	if basket == nil {
		return false, errors.New("出金篮子为空")
	}
	canProcess, err := uc.beginEvent(ctx, eventID, topic, basket.WithdrawNo)
	if err != nil || !canProcess {
		return false, err
	}
	if err := uc.CreateBasket(ctx, basket); err != nil {
		_ = uc.eventRepo.MarkFailed(ctx, eventID, err.Error())
		uc.metrics.IncEventFailed()
		return true, err
	}
	return true, uc.eventRepo.MarkSucceeded(ctx, eventID)
}

// ExpireTimeouts 处理过期入金和过期出金篮子。
func (uc *MatchingUsecase) ExpireTimeouts(ctx context.Context, now time.Time, limit int) (ExpireResult, error) {
	var out ExpireResult
	waitingDeposits, err := uc.expireWaitingDeposits(ctx, now, limit)
	if err != nil {
		return out, err
	}
	out.WaitingDeposits = waitingDeposits
	uc.metrics.AddExpiredWaitingDeposits(waitingDeposits)
	lockedDeposits, err := uc.expireLockedDeposits(ctx, now, limit)
	if err != nil {
		return out, err
	}
	out.LockedDeposits = lockedDeposits
	uc.metrics.AddExpiredLockedDeposits(lockedDeposits)
	baskets, err := uc.expireBaskets(ctx, now, limit)
	if err != nil {
		return out, err
	}
	out.Baskets = baskets
	uc.metrics.AddExpiredBaskets(baskets)
	return out, nil
}

// beginEvent 开始处理一个幂等事件。
func (uc *MatchingUsecase) beginEvent(ctx context.Context, eventID, topic, bizNo string) (bool, error) {
	if eventID == "" {
		return false, errors.New("事件 ID 为空")
	}
	if bizNo == "" {
		bizNo = eventID
	}
	created, err := uc.eventRepo.TryCreateProcessing(ctx, &model.EventInbox{
		EventID: eventID,
		Topic:   topic,
		BizNo:   bizNo,
	})
	if err != nil || created {
		return created, err
	}
	event, err := uc.eventRepo.GetByEventID(ctx, eventID)
	if err != nil {
		return false, err
	}
	if event.Status == model.EventStatusFailed {
		retried, err := uc.eventRepo.RetryFailed(ctx, eventID)
		if retried {
			uc.metrics.IncEventRetried()
		}
		return retried, err
	}
	return false, nil
}

// restoreBasket 恢复单个篮子和已挂入入金明细。
func (uc *MatchingUsecase) restoreBasket(ctx context.Context, basket *model.WithdrawBasket) error {
	memBasket := engine.NewBasketFromModel(basket)
	deposits, err := uc.listBasketDeposits(ctx, basket.BasketNo)
	if err != nil {
		return err
	}
	for _, deposit := range deposits {
		memBasket.DepositNos = append(memBasket.DepositNos, deposit.DepositNo)
	}
	uc.getShard(basket.Channel, basket.Currency).AddBasket(memBasket)
	return nil
}

// expireWaitingDeposits 过期未挂入篮子的入金单。
func (uc *MatchingUsecase) expireWaitingDeposits(ctx context.Context, now time.Time, limit int) (int, error) {
	var deposits []*model.DepositOrder
	if err := uc.data.DB.WithContext(ctx).
		Where("status = ? AND expire_at <= ?", model.StatusWaiting, now).
		Order("expire_at ASC, id ASC").
		Limit(limit).
		Find(&deposits).Error; err != nil {
		return 0, err
	}
	for _, deposit := range deposits {
		ret := uc.data.DB.WithContext(ctx).
			Model(&model.DepositOrder{}).
			Where("deposit_no = ? AND status = ?", deposit.DepositNo, model.StatusWaiting).
			Update("status", model.StatusExpired)
		if ret.Error != nil {
			return 0, ret.Error
		}
		if ret.RowsAffected > 0 {
			if err := uc.createStateLog(ctx, deposit.DepositNo, "deposit", &deposit.Status, model.StatusExpired, "入金超时", "expire_worker"); err != nil {
				return 0, err
			}
		}
	}
	return len(deposits), nil
}

// expireLockedDeposits 过期已挂入但未成交的入金单。
func (uc *MatchingUsecase) expireLockedDeposits(ctx context.Context, now time.Time, limit int) (int, error) {
	var deposits []*model.DepositOrder
	if err := uc.data.DB.WithContext(ctx).
		Where("status = ? AND expire_at <= ?", model.StatusLocked, now).
		Order("expire_at ASC, id ASC").
		Limit(limit).
		Find(&deposits).Error; err != nil {
		return 0, err
	}
	for _, deposit := range deposits {
		if err := uc.expireLockedDeposit(ctx, deposit); err != nil {
			return 0, err
		}
	}
	return len(deposits), nil
}

// expireLockedDeposit 释放一笔已挂入篮子的过期入金。
func (uc *MatchingUsecase) expireLockedDeposit(ctx context.Context, deposit *model.DepositOrder) error {
	return uc.withShardLock(ctx, deposit.Channel, deposit.Currency, func() error {
		var detail model.BasketDeposit
		if err := uc.data.Transaction(ctx, func(tx *gorm.DB) error {
			if err := tx.WithContext(ctx).
				Where("deposit_no = ? AND status = ?", deposit.DepositNo, model.BasketDepositStatusAttached).
				First(&detail).Error; err != nil {
				return err
			}
			if err := tx.WithContext(ctx).
				Model(&model.DepositOrder{}).
				Where("deposit_no = ? AND status = ?", deposit.DepositNo, model.StatusLocked).
				Update("status", model.StatusExpired).Error; err != nil {
				return err
			}
			if err := createStateLogTx(ctx, tx, deposit.DepositNo, "deposit", &deposit.Status, model.StatusExpired, "已挂入入金超时释放", "expire_worker"); err != nil {
				return err
			}
			if err := tx.WithContext(ctx).
				Model(&model.BasketDeposit{}).
				Where("deposit_no = ? AND status = ?", deposit.DepositNo, model.BasketDepositStatusAttached).
				Update("status", model.BasketDepositStatusReleased).Error; err != nil {
				return err
			}
			ret := tx.WithContext(ctx).
				Model(&model.WithdrawBasket{}).
				Where("basket_no = ? AND status = ?", detail.BasketNo, model.StatusWaiting).
				Update("current_amount", gorm.Expr("GREATEST(current_amount - ?, 0)", detail.Amount))
			if ret.Error != nil {
				return ret.Error
			}
			return nil
		}); err != nil {
			return err
		}
		uc.getShard(deposit.Channel, deposit.Currency).ReleaseDeposit(detail.BasketNo, deposit.DepositNo, detail.Amount)
		return nil
	})
}

// expireBaskets 关闭过期出金篮子。
func (uc *MatchingUsecase) expireBaskets(ctx context.Context, now time.Time, limit int) (int, error) {
	var baskets []*model.WithdrawBasket
	if err := uc.data.DB.WithContext(ctx).
		Where("status = ? AND expire_at <= ?", model.StatusWaiting, now).
		Order("expire_at ASC, id ASC").
		Limit(limit).
		Find(&baskets).Error; err != nil {
		return 0, err
	}
	for _, basket := range baskets {
		if err := uc.expireBasket(ctx, basket); err != nil {
			return 0, err
		}
	}
	return len(baskets), nil
}

// expireBasket 处理单个过期出金篮子。
func (uc *MatchingUsecase) expireBasket(ctx context.Context, basket *model.WithdrawBasket) error {
	return uc.withShardLock(ctx, basket.Channel, basket.Currency, func() error {
		changed := false
		if uc.canCompletePartial(basket) {
			completed, err := uc.completeExpiredBasket(ctx, basket)
			if err != nil || completed {
				return err
			}
		}
		if err := uc.data.Transaction(ctx, func(tx *gorm.DB) error {
			ret := tx.WithContext(ctx).
				Model(&model.WithdrawBasket{}).
				Where("basket_no = ? AND status = ?", basket.BasketNo, model.StatusWaiting).
				Update("status", model.StatusExpired)
			if ret.Error != nil {
				return ret.Error
			}
			if ret.RowsAffected == 0 {
				return nil
			}
			changed = true
			if err := createStateLogTx(ctx, tx, basket.BasketNo, "basket", &basket.Status, model.StatusExpired, "出金篮子超时", "expire_worker"); err != nil {
				return err
			}
			if err := tx.WithContext(ctx).
				Model(&model.BasketDeposit{}).
				Where("basket_no = ? AND status = ?", basket.BasketNo, model.BasketDepositStatusAttached).
				Update("status", model.BasketDepositStatusReleased).Error; err != nil {
				return err
			}
			if err := tx.WithContext(ctx).
				Model(&model.DepositOrder{}).
				Where("matched_basket_no = ? AND status = ?", basket.BasketNo, model.StatusLocked).
				Update("status", model.StatusExpired).Error; err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
		if changed {
			uc.getShard(basket.Channel, basket.Currency).RemoveBasket(basket.BasketNo)
		}
		return nil
	})
}

// canCompletePartial 判断过期篮子是否达到最低成交金额。
func (uc *MatchingUsecase) canCompletePartial(basket *model.WithdrawBasket) bool {
	return basket.CurrentAmount > 0 && basket.CurrentAmount >= uc.minCompleteAmount(basket.TargetAmount)
}

// completeExpiredBasket 按当前已凑金额完成过期篮子。
func (uc *MatchingUsecase) completeExpiredBasket(ctx context.Context, basket *model.WithdrawBasket) (bool, error) {
	var changed bool
	err := uc.data.Transaction(ctx, func(tx *gorm.DB) error {
		deposits, err := uc.listBasketDepositsTx(ctx, tx, basket.BasketNo)
		if err != nil {
			return err
		}
		if len(deposits) == 0 {
			return nil
		}
		result := uc.buildPartialMatchResult(basket, deposits)
		if result.MatchedAmount < uc.minCompleteAmount(result.TargetAmount) {
			return nil
		}
		if err := uc.persistCompletedMatch(ctx, tx, result); err != nil {
			return err
		}
		changed = true
		return nil
	})
	if err != nil {
		return false, err
	}
	if changed {
		uc.getShard(basket.Channel, basket.Currency).RemoveBasket(basket.BasketNo)
	}
	return changed, nil
}

// withShardLock 在分片锁内执行函数。
func (uc *MatchingUsecase) withShardLock(ctx context.Context, channel, currency string, fn func() error) error {
	if uc.locker == nil {
		return fn()
	}
	return uc.locker.WithLock(ctx, shardKey(channel, currency), uc.lockTTL, fn)
}

// getShard 获取或创建内存撮合分片。
func (uc *MatchingUsecase) getShard(channel, currency string) *engine.Shard {
	key := shardKey(channel, currency)
	if uc.shards[key] == nil {
		uc.shards[key] = engine.NewShard(channel, currency)
	}
	return uc.shards[key]
}

// saveDepositIfNeeded 保存入金单。
func (uc *MatchingUsecase) saveDepositIfNeeded(ctx context.Context, deposit *model.DepositOrder) error {
	var exists model.DepositOrder
	err := uc.data.DB.WithContext(ctx).Where("deposit_no = ?", deposit.DepositNo).First(&exists).Error
	if err == nil {
		deposit.ID = exists.ID
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return uc.data.DB.WithContext(ctx).Create(deposit).Error
}

// persistMatchResult 持久化撮合结果。
func (uc *MatchingUsecase) persistMatchResult(ctx context.Context, deposit *model.DepositOrder, result engine.MatchResult) error {
	return uc.data.Transaction(ctx, func(tx *gorm.DB) error {
		if err := uc.createBasketDeposit(ctx, tx, deposit, result); err != nil {
			return err
		}
		if result.Matched {
			return uc.persistCompletedMatch(ctx, tx, result)
		}
		return uc.persistAttachedDeposit(ctx, tx, deposit, result)
	})
}

// createBasketDeposit 创建篮子入金明细。
func (uc *MatchingUsecase) createBasketDeposit(ctx context.Context, tx *gorm.DB, deposit *model.DepositOrder, result engine.MatchResult) error {
	if err := tx.WithContext(ctx).Create(&model.BasketDeposit{
		BasketNo:   result.BasketNo,
		WithdrawNo: result.WithdrawNo,
		DepositNo:  deposit.DepositNo,
		Amount:     deposit.Amount,
		Status:     model.BasketDepositStatusAttached,
	}).Error; err != nil {
		return err
	}
	return createStateLogTx(ctx, tx, deposit.DepositNo, "deposit", &deposit.Status, model.StatusLocked, "入金挂入篮子", "match")
}

// persistAttachedDeposit 持久化已挂入但未凑满的入金。
func (uc *MatchingUsecase) persistAttachedDeposit(ctx context.Context, tx *gorm.DB, deposit *model.DepositOrder, result engine.MatchResult) error {
	ret := tx.WithContext(ctx).
		Model(&model.WithdrawBasket{}).
		Where("basket_no = ? AND status = ?", result.BasketNo, model.StatusWaiting).
		Update("current_amount", result.MatchedAmount)
	if ret.Error != nil {
		return ret.Error
	}
	if ret.RowsAffected == 0 {
		return errBasketChanged
	}
	if err := tx.WithContext(ctx).
		Model(&model.DepositOrder{}).
		Where("deposit_no = ? AND status = ?", deposit.DepositNo, model.StatusWaiting).
		Updates(map[string]any{
			"status":            model.StatusLocked,
			"matched_basket_no": result.BasketNo,
		}).Error; err != nil {
		return err
	}
	return createStateLogTx(ctx, tx, deposit.DepositNo, "deposit", &deposit.Status, model.StatusLocked, "入金锁定", "match")
}

// persistCompletedMatch 持久化已经凑满的撮合结果。
func (uc *MatchingUsecase) persistCompletedMatch(ctx context.Context, tx *gorm.DB, result engine.MatchResult) error {
	matchNo := idgen.New("M")
	if err := tx.WithContext(ctx).Create(&model.MatchRecord{
		MatchNo:       matchNo,
		BasketNo:      result.BasketNo,
		WithdrawNo:    result.WithdrawNo,
		Channel:       result.Channel,
		Currency:      result.Currency,
		TargetAmount:  result.TargetAmount,
		MatchedAmount: result.MatchedAmount,
		ShortAmount:   result.ShortAmount,
		Status:        model.MatchStatusCreated,
	}).Error; err != nil {
		return err
	}
	if err := createStateLogTx(ctx, tx, matchNo, "match", nil, model.MatchStatusCreated, "创建撮合结果", "match"); err != nil {
		return err
	}
	for _, depositNo := range result.DepositNos {
		if err := uc.persistMatchedDeposit(ctx, tx, matchNo, result, depositNo); err != nil {
			return err
		}
	}
	ret := tx.WithContext(ctx).
		Model(&model.WithdrawBasket{}).
		Where("basket_no = ? AND status = ?", result.BasketNo, model.StatusWaiting).
		Updates(map[string]any{
			"status":         model.StatusMatched,
			"current_amount": result.MatchedAmount,
		})
	if ret.Error != nil {
		return ret.Error
	}
	if ret.RowsAffected == 0 {
		return errBasketChanged
	}
	if err := createStateLogTx(ctx, tx, result.BasketNo, "basket", ptrStatus(model.StatusWaiting), model.StatusMatched, "篮子撮合成功", "match"); err != nil {
		return err
	}
	if err := tx.WithContext(ctx).
		Model(&model.BasketDeposit{}).
		Where("basket_no = ? AND status = ?", result.BasketNo, model.BasketDepositStatusAttached).
		Update("status", model.BasketDepositStatusMatched).Error; err != nil {
		return err
	}
	uc.metrics.IncMatchSuccess()
	if result.ShortAmount > 0 {
		uc.metrics.IncShortMatchSuccess()
	}
	return nil
}

// persistMatchedDeposit 持久化撮合结果中的一笔入金。
func (uc *MatchingUsecase) persistMatchedDeposit(ctx context.Context, tx *gorm.DB, matchNo string, result engine.MatchResult, depositNo string) error {
	var deposit model.DepositOrder
	if err := tx.WithContext(ctx).Where("deposit_no = ?", depositNo).First(&deposit).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Create(&model.MatchRecordDeposit{
		MatchNo:    matchNo,
		WithdrawNo: result.WithdrawNo,
		DepositNo:  depositNo,
		Amount:     deposit.Amount,
	}).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).
		Model(&model.DepositOrder{}).
		Where("deposit_no = ? AND status IN ?", depositNo, []int32{model.StatusWaiting, model.StatusLocked}).
		Updates(map[string]any{
			"status":            model.StatusMatched,
			"matched_basket_no": result.BasketNo,
			"match_no":          matchNo,
		}).Error; err != nil {
		return err
	}
	return createStateLogTx(ctx, tx, depositNo, "deposit", &deposit.Status, model.StatusMatched, "入金撮合成功", "match")
}

// listBasketDeposits 查询篮子已经挂入的入金明细。
func (uc *MatchingUsecase) listBasketDeposits(ctx context.Context, basketNo string) ([]*model.BasketDeposit, error) {
	var out []*model.BasketDeposit
	err := uc.data.DB.WithContext(ctx).
		Where("basket_no = ? AND status IN ?", basketNo, []int32{model.BasketDepositStatusAttached, model.BasketDepositStatusMatched}).
		Order("id ASC").
		Find(&out).Error
	return out, err
}

// listBasketDepositsTx 在事务内查询篮子已经挂入的入金明细。
func (uc *MatchingUsecase) listBasketDepositsTx(ctx context.Context, tx *gorm.DB, basketNo string) ([]*model.BasketDeposit, error) {
	var out []*model.BasketDeposit
	err := tx.WithContext(ctx).
		Where("basket_no = ? AND status = ?", basketNo, model.BasketDepositStatusAttached).
		Order("id ASC").
		Find(&out).Error
	return out, err
}

// buildPartialMatchResult 生成少发成交的撮合结果。
func (uc *MatchingUsecase) buildPartialMatchResult(basket *model.WithdrawBasket, deposits []*model.BasketDeposit) engine.MatchResult {
	depositNos := make([]string, 0, len(deposits))
	var matchedAmount int64
	for _, deposit := range deposits {
		depositNos = append(depositNos, deposit.DepositNo)
		matchedAmount += deposit.Amount
	}
	return engine.MatchResult{
		Matched:       true,
		Attached:      true,
		BasketNo:      basket.BasketNo,
		WithdrawNo:    basket.WithdrawNo,
		Channel:       basket.Channel,
		Currency:      basket.Currency,
		DepositNos:    depositNos,
		TargetAmount:  basket.TargetAmount,
		MatchedAmount: matchedAmount,
		ShortAmount:   basket.TargetAmount - matchedAmount,
	}
}

// minCompleteAmount 返回出金篮子的最低可成交金额。
func (uc *MatchingUsecase) minCompleteAmount(targetAmount int64) int64 {
	if targetAmount <= 0 {
		return 0
	}
	rate := uc.cfgMinCompleteRate()
	return (targetAmount*rate + 9999) / 10000
}

// cfgMinCompleteRate 返回最低成交比例。
func (uc *MatchingUsecase) cfgMinCompleteRate() int64 {
	if uc.minCompleteRate <= 0 {
		return 3000
	}
	if uc.minCompleteRate > 10000 {
		return 10000
	}
	return uc.minCompleteRate
}

// shardKey 生成撮合分片 key。
func shardKey(channel, currency string) string {
	return fmt.Sprintf("%s:%s", channel, currency)
}

// createStateLog 创建状态流水。
func (uc *MatchingUsecase) createStateLog(ctx context.Context, bizNo, bizType string, fromStatus *int32, toStatus int32, reason, operator string) error {
	return uc.logRepo.Create(ctx, &model.StateLog{
		BizNo:      bizNo,
		BizType:    bizType,
		FromStatus: fromStatus,
		ToStatus:   toStatus,
		Reason:     reason,
		Operator:   operator,
	})
}

// createStateLogTx 在事务内创建状态流水。
func createStateLogTx(ctx context.Context, tx *gorm.DB, bizNo, bizType string, fromStatus *int32, toStatus int32, reason, operator string) error {
	return tx.WithContext(ctx).Create(&model.StateLog{
		BizNo:      bizNo,
		BizType:    bizType,
		FromStatus: fromStatus,
		ToStatus:   toStatus,
		Reason:     reason,
		Operator:   operator,
	}).Error
}

// ptrStatus 返回状态指针。
func ptrStatus(status int32) *int32 {
	return &status
}

// normalizeDeposit 填充入金单默认值。
func normalizeDeposit(deposit *model.DepositOrder) {
	if deposit.DepositNo == "" {
		deposit.DepositNo = idgen.New("D")
	}
	if deposit.ExpireAt.IsZero() {
		deposit.ExpireAt = time.Now().Add(30 * time.Minute)
	}
}

// validateBasket 校验出金篮子参数。
func validateBasket(basket *model.WithdrawBasket) error {
	if basket.WithdrawNo == "" {
		return errors.New("出金单号为空")
	}
	if basket.Channel == "" {
		return errors.New("支付渠道为空")
	}
	if basket.Currency == "" {
		return errors.New("币种为空")
	}
	if basket.TargetAmount <= 0 {
		return errors.New("出金目标金额必须大于 0")
	}
	if basket.ExpireAt.IsZero() {
		return errors.New("出金过期时间为空")
	}
	if !basket.ExpireAt.After(time.Now()) {
		return errors.New("出金过期时间必须大于当前时间")
	}
	return nil
}

// validateDeposit 校验入金单参数。
func validateDeposit(deposit *model.DepositOrder) error {
	if deposit.Channel == "" {
		return errors.New("支付渠道为空")
	}
	if deposit.Currency == "" {
		return errors.New("币种为空")
	}
	if deposit.Amount <= 0 {
		return errors.New("入金金额必须大于 0")
	}
	if deposit.ExpireAt.IsZero() {
		return errors.New("入金过期时间为空")
	}
	if !deposit.ExpireAt.After(time.Now()) {
		return errors.New("入金过期时间必须大于当前时间")
	}
	return nil
}
