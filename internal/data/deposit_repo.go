package data

import (
	"context"

	"gorm.io/gorm"

	"matching-service/pkg/model"
)

// DepositRepo 负责入金单数据访问。
type DepositRepo struct {
	db *gorm.DB // db 是数据库连接
}

// NewDepositRepo 创建入金单仓库。
func NewDepositRepo(data *Data) *DepositRepo {
	return &DepositRepo{db: data.DB}
}

// Create 创建入金单。
func (r *DepositRepo) Create(ctx context.Context, in *model.DepositOrder) error {
	return r.db.WithContext(ctx).Create(in).Error
}

// GetByNo 根据入金单号查询入金单。
func (r *DepositRepo) GetByNo(ctx context.Context, depositNo string) (*model.DepositOrder, error) {
	var out model.DepositOrder
	if err := r.db.WithContext(ctx).Where("deposit_no = ?", depositNo).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

// Save 保存入金单。
func (r *DepositRepo) Save(ctx context.Context, in *model.DepositOrder) error {
	return r.db.WithContext(ctx).Save(in).Error
}

// MarkMatched 标记入金单已撮合。
func (r *DepositRepo) MarkMatched(ctx context.Context, depositNo, basketNo, matchNo string) error {
	return r.db.WithContext(ctx).
		Model(&model.DepositOrder{}).
		Where("deposit_no = ? AND status = ?", depositNo, model.StatusWaiting).
		Updates(map[string]any{
			"status":            model.StatusMatched,
			"matched_basket_no": basketNo,
			"match_no":          matchNo,
		}).Error
}
