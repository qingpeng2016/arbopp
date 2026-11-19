package rest

import (
	"context"
	"errors"
	"net/http"
	"strings"

	ginMiddleware "github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/gin/middleware"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	conf2 "github.com/gph-tech/fgmm-strategy-bitfinex/conf"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	gs "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// Router 路由（prediction handler 依赖的 core-service 包缺失，暂不挂业务路由）
type Router struct {
	httpServer *http.Server
	setting    *conf2.Config
}

func NewRouter(setting *conf2.Config) *Router {
	return &Router{setting: setting}
}

func (r *Router) setupRouters() *gin.Engine {
	engine := gin.Default()
	engine.Use(ginMiddleware.TraceRequestLog, ginMiddleware.CORSMiddleware, ginMiddleware.AuthKeyMiddleware)
	engine.GET("/swagger/*any", gs.WrapHandler(swaggerFiles.Handler))
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return engine
}

func (r *Router) Run(setting *conf2.Server) {
	gin.SetMode(strings.ToLower(setting.RunMode))
	go func() {
		r.httpServer = &http.Server{Addr: setting.Port, Handler: r.setupRouters()}
		if err := r.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			notification.SendErrorLog(context.Background(), "run http server failed", zap.String("error", err.Error()))
			panic(err)
		}
	}()
}

func (r *Router) Close() {
	if r.httpServer == nil {
		return
	}
	if err := r.httpServer.Shutdown(context.Background()); err != nil {
		notification.SendErrorLog(context.Background(), "stop http server failed", zap.String("error", err.Error()))
	}
}
