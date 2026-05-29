package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	"matching-service/internal/conf"
)

// SetTracerProvider 注册全局 OTLP trace exporter。
func SetTracerProvider(ctx context.Context, c *conf.Trace) error {
	if c == nil || c.Endpoint == "" {
		return nil
	}
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(c.Endpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithHeaders(map[string]string{"Authentication": c.Token}),
	)
	if err != nil {
		return err
	}
	ratio := float64(c.SampleRatio)
	if ratio <= 0 {
		ratio = 1
	}
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.ParentBased(tracesdk.TraceIDRatioBased(ratio))),
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewSchemaless(
			semconv.ServiceNameKey.String("matching-service"),
		)),
	)
	otel.SetTracerProvider(tp)
	return nil
}

// TraceHeaderMiddleware 在响应头写入 trace id。
func TraceHeaderMiddleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				tr.ReplyHeader().Set("X-Trace-ID", trace.SpanContextFromContext(ctx).TraceID().String())
			}
			return handler(ctx, req)
		}
	}
}
