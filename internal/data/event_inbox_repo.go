package data

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"matching-service/internal/model"
)

// EventInboxRepo 负责 RocketMQ 消费幂等记录数据访问。
type EventInboxRepo struct {
	db *gorm.DB // db 是数据库连接
}

// NewEventInboxRepo 创建 RocketMQ 消费幂等仓库。
func NewEventInboxRepo(data *Data) *EventInboxRepo {
	return &EventInboxRepo{db: data.DB}
}

// CreateProcessing 创建处理中事件记录。
func (r *EventInboxRepo) CreateProcessing(ctx context.Context, in *model.EventInbox) error {
	in.Status = model.EventStatusProcessing
	return r.db.WithContext(ctx).Create(in).Error
}

// TryCreateProcessing 尝试创建处理中事件记录。
func (r *EventInboxRepo) TryCreateProcessing(ctx context.Context, in *model.EventInbox) (bool, error) {
	err := r.CreateProcessing(ctx, in)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return false, nil
	}
	return false, err
}

// GetByEventID 根据事件 ID 查询幂等记录。
func (r *EventInboxRepo) GetByEventID(ctx context.Context, eventID string) (*model.EventInbox, error) {
	var out model.EventInbox
	if err := r.db.WithContext(ctx).Where("event_id = ?", eventID).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

// RetryFailed 把失败事件重新标记为处理中。
func (r *EventInboxRepo) RetryFailed(ctx context.Context, eventID string) (bool, error) {
	ret := r.db.WithContext(ctx).
		Model(&model.EventInbox{}).
		Where("event_id = ? AND status = ?", eventID, model.EventStatusFailed).
		Updates(map[string]any{
			"status":    model.EventStatusProcessing,
			"error_msg": "",
		})
	return ret.RowsAffected > 0, ret.Error
}

// MarkSucceeded 标记事件处理成功。
func (r *EventInboxRepo) MarkSucceeded(ctx context.Context, eventID string) error {
	return r.db.WithContext(ctx).
		Model(&model.EventInbox{}).
		Where("event_id = ?", eventID).
		Update("status", model.EventStatusSucceeded).Error
}

// MarkFailed 标记事件处理失败。
func (r *EventInboxRepo) MarkFailed(ctx context.Context, eventID string, errMsg string) error {
	return r.db.WithContext(ctx).
		Model(&model.EventInbox{}).
		Where("event_id = ?", eventID).
		Updates(map[string]any{
			"status":    model.EventStatusFailed,
			"error_msg": errMsg,
		}).Error
}
