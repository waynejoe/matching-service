package service

import (
	"context"

	"matching-service/internal/biz"
	v1 "matching-service/pb/matching/v1"
)

// GetMetrics 返回服务运行指标。
func (s *MatchingService) GetMetrics(ctx context.Context, req *v1.GetMetricsRequest) (*v1.GetMetricsReply, error) {
	return metricsToPB(s.uc.Metrics()), nil
}

// metricsToPB 把指标快照转换成 protobuf 响应。
func metricsToPB(in biz.Snapshot) *v1.GetMetricsReply {
	return &v1.GetMetricsReply{
		DepositConsumeSuccess:  in.DepositConsumeSuccess,
		DepositConsumeFailed:   in.DepositConsumeFailed,
		WithdrawConsumeSuccess: in.WithdrawConsumeSuccess,
		WithdrawConsumeFailed:  in.WithdrawConsumeFailed,
		MatchSuccess:           in.MatchSuccess,
		ShortMatchSuccess:      in.ShortMatchSuccess,
		ExpiredWaitingDeposits: in.ExpiredWaitingDeposits,
		ExpiredLockedDeposits:  in.ExpiredLockedDeposits,
		ExpiredBaskets:         in.ExpiredBaskets,
		EventFailed:            in.EventFailed,
		EventRetried:           in.EventRetried,
	}
}
