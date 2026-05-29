package data

import (
	"context"

	"gorm.io/gorm"

	"matching-service/pkg/model"
)

// BasketDepositRepo 负责篮子入金明细数据访问。
type BasketDepositRepo struct {
	db *gorm.DB // db 是数据库连接
}

// NewBasketDepositRepo 创建篮子入金明细仓库。
func NewBasketDepositRepo(data *Data) *BasketDepositRepo {
	return &BasketDepositRepo{db: data.DB}
}

// Create 创建篮子入金明细。
func (r *BasketDepositRepo) Create(ctx context.Context, in *model.BasketDeposit) error {
	return r.db.WithContext(ctx).Create(in).Error
}

// ListByBasketNo 查询篮子下的入金明细。
func (r *BasketDepositRepo) ListByBasketNo(ctx context.Context, basketNo string) ([]*model.BasketDeposit, error) {
	var out []*model.BasketDeposit
	err := r.db.WithContext(ctx).Where("basket_no = ?", basketNo).Order("id ASC").Find(&out).Error
	return out, err
}

// MarkMatchedByBasketNo 标记篮子下的入金明细已形成撮合结果。
func (r *BasketDepositRepo) MarkMatchedByBasketNo(ctx context.Context, basketNo string) error {
	return r.db.WithContext(ctx).
		Model(&model.BasketDeposit{}).
		Where("basket_no = ? AND status = ?", basketNo, model.BasketDepositStatusAttached).
		Update("status", model.BasketDepositStatusMatched).Error
}
