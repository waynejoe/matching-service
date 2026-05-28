package data

import (
	"context"

	"gorm.io/gorm"

	"matching-service/internal/model"
)

// StateLogRepo 负责状态流水数据访问。
type StateLogRepo struct {
	db *gorm.DB // db 是数据库连接
}

// NewStateLogRepo 创建状态流水仓库。
func NewStateLogRepo(data *Data) *StateLogRepo {
	return &StateLogRepo{db: data.DB}
}

// Create 创建状态流水。
func (r *StateLogRepo) Create(ctx context.Context, in *model.StateLog) error {
	return r.db.WithContext(ctx).Create(in).Error
}

// ListByBizNo 查询业务单号对应的状态流水。
func (r *StateLogRepo) ListByBizNo(ctx context.Context, bizNo string) ([]*model.StateLog, error) {
	var out []*model.StateLog
	err := r.db.WithContext(ctx).Where("biz_no = ?", bizNo).Order("id ASC").Find(&out).Error
	return out, err
}
