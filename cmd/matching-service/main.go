package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"
)

// main 是撮合服务启动入口。
func main() {
	confPath := flag.String("conf", "../../configs/config.yaml", "config path")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := newApplication(ctx, *confPath)
	if err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
	defer app.Close()

	app.Run(ctx, stop)
	<-ctx.Done()
}
