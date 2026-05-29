package logx

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

func Infof(ctx context.Context, format string, args ...any) {
	log.Context(ctx).Info(fmt.Sprintf(format, args...))
}

func Errorf(ctx context.Context, format string, args ...any) {
	log.Context(ctx).Error(fmt.Sprintf(format, args...))
}

func Fatalf(ctx context.Context, format string, args ...any) {
	log.Context(ctx).Fatal(fmt.Sprintf(format, args...))
}
