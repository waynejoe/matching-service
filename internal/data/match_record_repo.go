package data

import (
	"context"

	"gorm.io/gorm"

	"matching-service/pkg/model"
)

// MatchRecordRepo 负责撮合结果数据访问。
type MatchRecordRepo struct {
	db *gorm.DB // db 是数据库连接
}

// NewMatchRecordRepo 创建撮合结果仓库。
func NewMatchRecordRepo(data *Data) *MatchRecordRepo {
	return &MatchRecordRepo{db: data.DB}
}

// Create 创建撮合结果。
func (r *MatchRecordRepo) Create(ctx context.Context, in *model.MatchRecord) error {
	return r.db.WithContext(ctx).Create(in).Error
}

// CreateDeposit 创建撮合结果入金明细。
func (r *MatchRecordRepo) CreateDeposit(ctx context.Context, in *model.MatchRecordDeposit) error {
	return r.db.WithContext(ctx).Create(in).Error
}

// GetByNo 根据撮合结果单号查询撮合结果。
func (r *MatchRecordRepo) GetByNo(ctx context.Context, matchNo string) (*model.MatchRecord, error) {
	var out model.MatchRecord
	if err := r.db.WithContext(ctx).Where("match_no = ?", matchNo).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

// GetByBasketNo 根据篮子单号查询撮合结果。
func (r *MatchRecordRepo) GetByBasketNo(ctx context.Context, basketNo string) (*model.MatchRecord, error) {
	var out model.MatchRecord
	if err := r.db.WithContext(ctx).Where("basket_no = ?", basketNo).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

// GetByWithdrawNo 根据出金单号查询撮合结果。
func (r *MatchRecordRepo) GetByWithdrawNo(ctx context.Context, withdrawNo string) (*model.MatchRecord, error) {
	var out model.MatchRecord
	if err := r.db.WithContext(ctx).Where("withdraw_no = ?", withdrawNo).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

// ListDeposits 查询撮合结果下的入金明细。
func (r *MatchRecordRepo) ListDeposits(ctx context.Context, matchNo string) ([]*model.MatchRecordDeposit, error) {
	var out []*model.MatchRecordDeposit
	err := r.db.WithContext(ctx).Where("match_no = ?", matchNo).Order("id ASC").Find(&out).Error
	return out, err
}
