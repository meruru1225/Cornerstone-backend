package main

import (
	"Cornerstone/internal/api/config"
	"Cornerstone/internal/pkg/cron"
	"Cornerstone/internal/pkg/database"
	"Cornerstone/internal/pkg/es"
	"Cornerstone/internal/pkg/llm"
	"Cornerstone/internal/pkg/logger"
	"Cornerstone/internal/pkg/minio"
	"Cornerstone/internal/pkg/mongo"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/wire"
	"context"
	"errors"
	log "log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

func main() {
	// 加载配置
	if err := config.LoadConfig(); err != nil {
		log.Error("Fatal error: failed to load configuration", "err", err)
		panic(err)
	}
	cfg := config.Cfg

	// 初始化日志
	logger.InitLogger()

	// 数据库连接
	dbCfg := cfg.DB
	db, err := database.NewGormDB(&dbCfg)
	if err != nil {
		log.Error("Fatal error: failed to create database connection", "err", err)
		panic(err)
	}

	// Redis 连接
	redisCfg := config.Cfg.Redis
	err = redis.InitRedis(redisCfg)
	if err != nil {
		log.Error("Fatal error: failed to create redis connection", "err", err)
		panic(err)
	}

	// Mongo 连接
	mongoCfg := config.Cfg.Mongo
	mongoConn, err := mongo.InitMongo(mongoCfg)
	if err != nil {
		log.Error("Fatal error: failed to create mongo connection", "err", err)
		panic(err)
	}

	// MinIO 连接
	err = minio.Init()
	if err != nil {
		log.Error("Fatal error: failed to initialize MinIO", "err", err)
		panic(err)
	}

	// ElasticSearch 连接
	err = es.InitClient()
	if err != nil {
		log.Error("Fatal error: failed to initialize ElasticSearch", "err", err)
		panic(err)
	}

	// llm 模型初始化
	err = llm.InitLLM()
	if err != nil {
		log.Error("Fatal error: failed to initialize llm models", "err", err)
		panic(err)
	}

	// 依赖注入
	app, err := wire.BuildApplication(db, mongoConn, cfg)
	if err != nil {
		log.Error("Fatal error: failed to create application", "err", err)
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)

	// 定时任务
	err = cron.InitCron(app.CronMgr)
	if err != nil {
		log.Error("Fatal error: failed to start cron jobs", "err", err)
		panic(err)
	}
	g.Go(func() error {
		<-ctx.Done()
		log.Info("Cron Jobs stopping...")
		app.CronMgr.Stop()
		return nil
	})

	// Kafka 消费者
	g.Go(func() error {
		log.Info("Kafka Consumers starting...")
		return app.KafkaManager.Start(ctx, cfg)
	})

	// HTTP 服务器
	srv := &http.Server{
		Addr:    ":8080",
		Handler: app.Router,
	}
	g.Go(func() error {
		log.Info("HTTP Server starting...")
		if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	// 优雅退出
	g.Go(func() error {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case sig := <-quit:
			log.Info("Received signal, shutting down...", "signal", sig)
			cancel()
		}

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err = srv.Shutdown(shutdownCtx); err != nil {
			log.Error("HTTP Server shutdown failed", "err", err)
		}
		return nil
	})

	if err = g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("App exited with error", "err", err)
	}
	log.Info("App exited successfully.")
}
