package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	"matching-service/internal/conf"
	appmw "matching-service/internal/middleware"
	"matching-service/internal/server"
	"matching-service/pkg/toolbox/kratox"
	"matching-service/pkg/toolbox/mqx"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	Name     = "matching-service"
	Version  string
	flagconf string
	id, _    = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs/config.yaml", "config path, eg: -conf config.yaml")
}

func newApp(
	logger log.Logger,
	gs *grpc.Server,
	ms *http.Server,
	mqc *mqx.ConsumerManager,
	mqp *mqx.Producer,
	ew *server.ExpireWorker,
) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Logger(logger),
		kratos.Server(gs, ms, mqc, mqp, ew),
	)
}

// main 是撮合服务启动入口。
func main() {
	time.Local = time.UTC
	flag.Parse()

	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
	)
	log.SetLogger(logger)

	c := config.New(config.WithSource(file.NewSource(flagconf)))
	defer func() { _ = c.Close() }()
	if err := c.Load(); err != nil {
		panic(err)
	}
	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	ctx := context.Background()
	if err := appmw.SetTracerProvider(ctx, bc.Trace); err != nil {
		panic(err)
	}
	if err := kratox.InitSentry(ctx, &kratox.SentryConfig{
		Dsn:     bc.GetSentry().GetDsn(),
		Env:     bc.GetSentry().GetEnv(),
		SlsAddr: bc.GetSentry().GetSlsAddr(),
	}); err != nil {
		panic(err)
	}
	defer sentry.Flush(2 * time.Second)

	app, cleanup, err := wireApp(&bc, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	if err := app.Run(); err != nil {
		panic(err)
	}
}
