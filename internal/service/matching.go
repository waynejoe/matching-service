package service

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	v1 "matching-service/pkg/api/matching/v1"
	"matching-service/internal/biz"
	"matching-service/internal/engine"
	"matching-service/pkg/model"
)

// MatchingService 是撮合 gRPC 服务。
type MatchingService struct {
	v1.UnimplementedMatchingServiceServer
	uc      *biz.MatchingUsecase // uc 是撮合业务用例
	checker HealthChecker        // checker 是健康检查器
}

// NewMatchingService 创建撮合服务。
func NewMatchingService(uc *biz.MatchingUsecase, checker HealthChecker) *MatchingService {
	return &MatchingService{uc: uc, checker: checker}
}

// CreateBasket 创建出金篮子。
func (s *MatchingService) CreateBasket(ctx context.Context, req *v1.CreateBasketRequest) (*v1.CreateBasketReply, error) {
	basket := &model.WithdrawBasket{
		BasketNo:     req.GetBasketNo(),
		WithdrawNo:   req.GetWithdrawNo(),
		Channel:      req.GetChannel(),
		Currency:     req.GetCurrency(),
		TargetAmount: req.GetTargetAmount(),
		ExpireAt:     timestampToTime(req.GetExpireAt()),
	}
	if err := s.uc.CreateBasket(ctx, basket); err != nil {
		return nil, toGRPCError(err)
	}
	return &v1.CreateBasketReply{BasketNo: basket.BasketNo}, nil
}

// SubmitDeposit 提交入金并返回撮合结果。
func (s *MatchingService) SubmitDeposit(ctx context.Context, req *v1.SubmitDepositRequest) (*v1.SubmitDepositReply, error) {
	result, err := s.uc.SubmitDeposit(ctx, &model.DepositOrder{
		DepositNo: req.GetDepositNo(),
		Channel:   req.GetChannel(),
		Currency:  req.GetCurrency(),
		Amount:    req.GetAmount(),
		ExpireAt:  timestampToTime(req.GetExpireAt()),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &v1.SubmitDepositReply{Result: matchResultToPB(result)}, nil
}

// ExpireTimeouts 手动触发一次超时扫描。
func (s *MatchingService) ExpireTimeouts(ctx context.Context, req *v1.ExpireTimeoutsRequest) (*v1.ExpireTimeoutsReply, error) {
	now := time.Now()
	if req.GetNowUnix() > 0 {
		now = time.Unix(req.GetNowUnix(), 0)
	}
	result, err := s.uc.ExpireTimeouts(ctx, now, int(req.GetLimit()))
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &v1.ExpireTimeoutsReply{
		WaitingDeposits: int32(result.WaitingDeposits),
		LockedDeposits:  int32(result.LockedDeposits),
		Baskets:         int32(result.Baskets),
	}, nil
}

// GetDeposit 查询入金单状态。
func (s *MatchingService) GetDeposit(ctx context.Context, req *v1.GetDepositRequest) (*v1.GetDepositReply, error) {
	deposit, err := s.uc.GetDeposit(ctx, req.GetDepositNo())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &v1.GetDepositReply{Deposit: depositToPB(deposit)}, nil
}

// GetBasket 查询出金篮子状态。
func (s *MatchingService) GetBasket(ctx context.Context, req *v1.GetBasketRequest) (*v1.GetBasketReply, error) {
	basket, err := s.uc.GetBasket(ctx, req.GetBasketNo())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &v1.GetBasketReply{Basket: basketToPB(basket)}, nil
}

// GetMatch 查询撮合结果。
func (s *MatchingService) GetMatch(ctx context.Context, req *v1.GetMatchRequest) (*v1.GetMatchReply, error) {
	detail, err := s.uc.GetMatch(ctx, req.GetMatchNo(), req.GetBasketNo(), req.GetWithdrawNo())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &v1.GetMatchReply{
		Match:    matchRecordToPB(detail.Record),
		Deposits: matchDepositsToPB(detail.Deposits),
	}, nil
}

// toGRPCError 把业务错误转换成 gRPC 错误。
func toGRPCError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return status.Error(codes.NotFound, "记录不存在")
	}
	return status.Error(codes.InvalidArgument, err.Error())
}

// timestampToTime 把 protobuf 时间转换成 Go 时间。
func timestampToTime(in *timestamppb.Timestamp) time.Time {
	if in == nil {
		return time.Time{}
	}
	return in.AsTime()
}

// matchResultToPB 把内存撮合结果转换成 protobuf 响应。
func matchResultToPB(in engine.MatchResult) *v1.MatchResult {
	return &v1.MatchResult{
		Matched:       in.Matched,
		Attached:      in.Attached,
		BasketNo:      in.BasketNo,
		WithdrawNo:    in.WithdrawNo,
		Channel:       in.Channel,
		Currency:      in.Currency,
		DepositNos:    append([]string(nil), in.DepositNos...),
		TargetAmount:  in.TargetAmount,
		MatchedAmount: in.MatchedAmount,
		ShortAmount:   in.ShortAmount,
		Pending:       in.Pending,
		FailedReason:  in.FailedReason,
	}
}

// depositToPB 把入金单转换成 protobuf 响应。
func depositToPB(in *model.DepositOrder) *v1.DepositOrder {
	if in == nil {
		return nil
	}
	return &v1.DepositOrder{
		DepositNo:       in.DepositNo,
		MerchantId:      in.MerchantID,
		Channel:         in.Channel,
		Currency:        in.Currency,
		Amount:          in.Amount,
		Status:          in.Status,
		ExpireAt:        timeToTimestamp(in.ExpireAt),
		MatchedBasketNo: in.MatchedBasketNo,
		MatchNo:         in.MatchNo,
		CreatedAt:       timeToTimestamp(in.CreatedAt),
		UpdatedAt:       timeToTimestamp(in.UpdatedAt),
	}
}

// basketToPB 把出金篮子转换成 protobuf 响应。
func basketToPB(in *model.WithdrawBasket) *v1.WithdrawBasket {
	if in == nil {
		return nil
	}
	return &v1.WithdrawBasket{
		BasketNo:      in.BasketNo,
		WithdrawNo:    in.WithdrawNo,
		MerchantId:    in.MerchantID,
		Channel:       in.Channel,
		Currency:      in.Currency,
		TargetAmount:  in.TargetAmount,
		CurrentAmount: in.CurrentAmount,
		Status:        in.Status,
		ExpireAt:      timeToTimestamp(in.ExpireAt),
		Version:       in.Version,
		CreatedAt:     timeToTimestamp(in.CreatedAt),
		UpdatedAt:     timeToTimestamp(in.UpdatedAt),
	}
}

// matchRecordToPB 把撮合结果转换成 protobuf 响应。
func matchRecordToPB(in *model.MatchRecord) *v1.MatchRecord {
	if in == nil {
		return nil
	}
	return &v1.MatchRecord{
		MatchNo:       in.MatchNo,
		BasketNo:      in.BasketNo,
		WithdrawNo:    in.WithdrawNo,
		TargetAmount:  in.TargetAmount,
		MatchedAmount: in.MatchedAmount,
		ShortAmount:   in.ShortAmount,
		Channel:       in.Channel,
		Currency:      in.Currency,
		Status:        in.Status,
		CreatedAt:     timeToTimestamp(in.CreatedAt),
		UpdatedAt:     timeToTimestamp(in.UpdatedAt),
	}
}

// matchDepositsToPB 把撮合入金明细转换成 protobuf 响应。
func matchDepositsToPB(in []*model.MatchRecordDeposit) []*v1.MatchRecordDeposit {
	out := make([]*v1.MatchRecordDeposit, 0, len(in))
	for _, item := range in {
		out = append(out, matchDepositToPB(item))
	}
	return out
}

// matchDepositToPB 把单条撮合入金明细转换成 protobuf 响应。
func matchDepositToPB(in *model.MatchRecordDeposit) *v1.MatchRecordDeposit {
	if in == nil {
		return nil
	}
	return &v1.MatchRecordDeposit{
		MatchNo:    in.MatchNo,
		WithdrawNo: in.WithdrawNo,
		DepositNo:  in.DepositNo,
		Amount:     in.Amount,
		CreatedAt:  timeToTimestamp(in.CreatedAt),
	}
}

// timeToTimestamp 把 Go 时间转换成 protobuf 时间。
func timeToTimestamp(in time.Time) *timestamppb.Timestamp {
	if in.IsZero() {
		return nil
	}
	return timestamppb.New(in)
}
