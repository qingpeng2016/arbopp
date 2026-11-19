package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy"
	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/constants"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	conf2 "github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	"github.com/gph-tech/fgmm-strategy-bitfinex/interfaces/rest"
	"github.com/shopspring/decimal"
	"go.uber.org/dig"
	"go.uber.org/zap"
)

func main() {
	flag.Parse()
	decimal.DivisionPrecision = 18

	// 设置应用时区为 UTC（与数据库服务器时区一致）
	// 这样所有 time.Now() 都会返回 UTC 时间，与数据库时间一致
	// 如果数据库使用其他时区，可以修改为对应的时区，例如：
	// time.Local = constants.BjLocation  // 使用北京时间
	time.Local = constants.UtcLocation // 使用 UTC 时区（与数据库一致）

	container := boot.BuildContainer()

	if conf2.IsRunBot() {
		// 脚本
		var function any = func(bot *strategy.Entry) error {
			go func() {
				if err := bot.Start(); err != nil {
					notification.SendErrorLog(context.Background(), "bot start failed", zap.Error(err))
				}
			}()
			return nil
		}
		if err := container.Invoke(function); err != nil {
			notification.SendErrorLog(context.Background(), "run robot data core-service failed", zap.String("error", err.Error()))
			panic(err)
		}
	} else {
		// web
		var function any = func(router *rest.Router) {
			router.Run(conf2.GetServeConf())
		}
		if err := container.Invoke(function); err != nil {
			notification.SendErrorLog(context.Background(), "run chain data core-service failed", zap.String("error", err.Error()))
			panic(err)
		}
	}

	gracefulExit(container)
}

func gracefulExit(container *dig.Container) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, os.Kill)
	_ = <-signalChan

	logger.InfoZ(context.Background(), "graceful-shutdown-start")

	// 根据运行模式决定停止哪些服务
	if conf2.IsRunBot() {
		var function any = func(bot *strategy.Entry) error {
			err := bot.Stop()
			return err
		}
		if err := container.Invoke(function); err != nil {
			notification.SendErrorLog(context.Background(), "run robot data core-service failed", zap.String("error", err.Error()))
			panic(err)
		}

	} else {
		// Web 模式：停止 HTTP 服务器
		logger.InfoZ(context.Background(), "stopping http server")
		var webFunction any = func(router *rest.Router) {
			router.Close()
		}
		if err := container.Invoke(webFunction); err != nil {
			notification.SendErrorLog(context.Background(), "stop http server failed", zap.String("error", err.Error()))
		} else {
			logger.InfoZ(context.Background(), "http server stopped successfully")
		}
	}

	logger.InfoZ(context.Background(), "graceful-shutdown-completed")
}
