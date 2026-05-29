package kratox

import (
	"context"
	"fmt"
	"runtime"

	"github.com/getsentry/sentry-go"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// SentryConfig 是 Sentry 初始化配置。
type SentryConfig struct {
	Dsn     string
	Env     string
	SlsAddr string
}

// InitSentry 初始化 Sentry；dsn 为空时跳过。
func InitSentry(_ context.Context, conf *SentryConfig) error {
	if conf == nil || conf.Dsn == "" {
		return nil
	}
	return sentry.Init(sentry.ClientOptions{
		Dsn:              conf.Dsn,
		Environment:      conf.Env,
		SendDefaultPII:   true,
		TracesSampleRate: 0,
	})
}

// SentryMiddleware 捕获 panic 与 handler 错误并上报 Sentry。
func SentryMiddleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			defer func() {
				if r := recover(); r != nil {
					buf := make([]byte, 64<<10)
					n := runtime.Stack(buf, false)
					buf = buf[:n]
					log.Context(ctx).Errorf("panic: %v\n%s", r, buf)
					err = fmt.Errorf("panic: %v", r)
				}
				if err != nil {
					captureException(ctx, err)
				}
			}()
			return handler(ctx, req)
		}
	}
}

func captureException(ctx context.Context, err error) {
	if err == nil {
		return
	}
	hub := sentry.CurrentHub().Clone()
	if info, ok := transport.FromServerContext(ctx); ok {
		hub.Scope().SetTag("kind", info.Kind().String())
		hub.Scope().SetTag("operation", info.Operation())
	}
	hub.CaptureException(err)
}
