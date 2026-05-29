package service

import (
	"context"
	"encoding/json"

	"matching-service/internal/biz"
	"matching-service/internal/model"
	v1 "matching-service/api/matching/v1"
)

// MatchingConsumer 是 RocketMQ 消息处理入口。
type MatchingConsumer struct {
	uc      *biz.MatchingUsecase // uc 是撮合业务用例
	metrics *biz.Metrics         // metrics 是运行指标
}

// NewMatchingConsumer 创建撮合消息处理入口。
func NewMatchingConsumer(uc *biz.MatchingUsecase, metric *biz.Metrics) *MatchingConsumer {
	return &MatchingConsumer{uc: uc, metrics: metric}
}

// HandleDepositMessage 处理入金消息。
func (c *MatchingConsumer) HandleDepositMessage(ctx context.Context, body []byte) error {
	var msg v1.DepositEventMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		c.metrics.IncDepositConsumeFailed()
		return err
	}
	deposit := depositPBToModel(msg.GetData())
	if _, _, err := c.uc.ProcessDepositEvent(ctx, msg.GetEventId(), msg.GetTopic(), deposit); err != nil {
		c.metrics.IncDepositConsumeFailed()
		return err
	}
	c.metrics.IncDepositConsumeSuccess()
	return nil
}

// HandleWithdrawMessage 处理出金消息。
func (c *MatchingConsumer) HandleWithdrawMessage(ctx context.Context, body []byte) error {
	var msg v1.WithdrawEventMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		c.metrics.IncWithdrawConsumeFailed()
		return err
	}
	basket := withdrawPBToModel(msg.GetData())
	if _, err := c.uc.ProcessWithdrawEvent(ctx, msg.GetEventId(), msg.GetTopic(), basket); err != nil {
		c.metrics.IncWithdrawConsumeFailed()
		return err
	}
	c.metrics.IncWithdrawConsumeSuccess()
	return nil
}

// depositPBToModel 把 proto 入金单转换成 model 入金单。
func depositPBToModel(in *v1.DepositOrder) *model.DepositOrder {
	if in == nil {
		return nil
	}
	return &model.DepositOrder{
		DepositNo:       in.GetDepositNo(),
		MerchantID:      in.GetMerchantId(),
		Channel:         in.GetChannel(),
		Currency:        in.GetCurrency(),
		Amount:          in.GetAmount(),
		Status:          in.GetStatus(),
		ExpireAt:        timestampToTime(in.GetExpireAt()),
		MatchedBasketNo: in.GetMatchedBasketNo(),
		MatchNo:         in.GetMatchNo(),
		CreatedAt:       timestampToTime(in.GetCreatedAt()),
		UpdatedAt:       timestampToTime(in.GetUpdatedAt()),
	}
}

// withdrawPBToModel 把 proto 出金篮子转换成 model 出金篮子。
func withdrawPBToModel(in *v1.WithdrawBasket) *model.WithdrawBasket {
	if in == nil {
		return nil
	}
	return &model.WithdrawBasket{
		BasketNo:      in.GetBasketNo(),
		WithdrawNo:    in.GetWithdrawNo(),
		MerchantID:    in.GetMerchantId(),
		Channel:       in.GetChannel(),
		Currency:      in.GetCurrency(),
		TargetAmount:  in.GetTargetAmount(),
		CurrentAmount: in.GetCurrentAmount(),
		Status:        in.GetStatus(),
		ExpireAt:      timestampToTime(in.GetExpireAt()),
		Version:       in.GetVersion(),
		CreatedAt:     timestampToTime(in.GetCreatedAt()),
		UpdatedAt:     timestampToTime(in.GetUpdatedAt()),
	}
}
