package data

import (
	"context"

	"gorm.io/gorm"

	"matching-service/internal/model"
)

// BasketRepo 负责出金篮子数据访问。
type BasketRepo struct {
	db *gorm.DB // db 是数据库连接
}

// NewBasketRepo 创建出金篮子仓库。
func NewBasketRepo(data *Data) *BasketRepo {
	return &BasketRepo{db: data.DB}
}

// Create 创建出金篮子。
func (r *BasketRepo) Create(ctx context.Context, in *model.WithdrawBasket) error {
	return r.db.WithContext(ctx).Create(in).Error
}

// GetByNo 根据篮子单号查询出金篮子。
func (r *BasketRepo) GetByNo(ctx context.Context, basketNo string) (*model.WithdrawBasket, error) {
	var out model.WithdrawBasket
	if err := r.db.WithContext(ctx).Where("basket_no = ?", basketNo).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

// ListWaitingByShard 查询指定渠道和币种下等待撮合的篮子。
func (r *BasketRepo) ListWaitingByShard(ctx context.Context, channel, currency string, limit int) ([]*model.WithdrawBasket, error) {
	var out []*model.WithdrawBasket
	err := r.db.WithContext(ctx).
		Where("channel = ? AND currency = ? AND status = ?", channel, currency, model.StatusWaiting).
		Order("expire_at ASC, id ASC").
		Limit(limit).
		Find(&out).Error
	return out, err
}

// Save 保存出金篮子。
func (r *BasketRepo) Save(ctx context.Context, in *model.WithdrawBasket) error {
	return r.db.WithContext(ctx).Save(in).Error
}

// UpdateAmount 更新篮子已凑金额和版本号。
func (r *BasketRepo) UpdateAmount(ctx context.Context, basketNo string, oldVersion int64, currentAmount int64) error {
	return r.db.WithContext(ctx).
		Model(&model.WithdrawBasket{}).
		Where("basket_no = ? AND version = ?", basketNo, oldVersion).
		Updates(map[string]any{
			"current_amount": currentAmount,
			"version":        oldVersion + 1,
		}).Error
}

// MarkMatched 标记篮子已撮合。
func (r *BasketRepo) MarkMatched(ctx context.Context, basketNo string) error {
	return r.db.WithContext(ctx).
		Model(&model.WithdrawBasket{}).
		Where("basket_no = ? AND status = ?", basketNo, model.StatusWaiting).
		Update("status", model.StatusMatched).Error
}
