package service

import (
	"context"

	v1 "matching-service/pkg/api/matching/v1"
)

// HealthComponent 是单个依赖健康状态。
type HealthComponent struct {
	Name    string // Name 是组件名称
	OK      bool   // OK 表示组件是否可用
	Message string // Message 是检查结果说明
}

// HealthResult 是整体健康检查结果。
type HealthResult struct {
	OK         bool              // OK 表示全部组件是否可用
	Components []HealthComponent // Components 是组件检查结果
}

// HealthChecker 是健康检查接口。
type HealthChecker interface {
	Check(ctx context.Context) HealthResult
}

// CheckHealth 检查服务依赖健康状态。
func (s *MatchingService) CheckHealth(ctx context.Context, req *v1.CheckHealthRequest) (*v1.CheckHealthReply, error) {
	result := s.checker.Check(ctx)
	return &v1.CheckHealthReply{
		Ok:         result.OK,
		Components: healthComponentsToPB(result.Components),
	}, nil
}

// healthComponentsToPB 把健康检查组件列表转换成 protobuf 响应。
func healthComponentsToPB(in []HealthComponent) []*v1.HealthComponent {
	out := make([]*v1.HealthComponent, 0, len(in))
	for _, item := range in {
		out = append(out, healthComponentToPB(item))
	}
	return out
}

// healthComponentToPB 把单个健康检查组件转换成 protobuf 响应。
func healthComponentToPB(in HealthComponent) *v1.HealthComponent {
	return &v1.HealthComponent{
		Name:    in.Name,
		Ok:      in.OK,
		Message: in.Message,
	}
}
