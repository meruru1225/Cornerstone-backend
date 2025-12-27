package main

import (
	"Cornerstone/internal/api/config"
	"Cornerstone/internal/pkg/database"
	"Cornerstone/internal/pkg/llm"
	"Cornerstone/internal/pkg/minio"
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
	InitLogger()

	if err := config.LoadConfig(); err != nil {
		log.Error("Fatal error: failed to load configuration", "err", err)
		panic(err)
	}
	cfg := config.Cfg

	dbCfg := cfg.DB
	db, err := database.NewGormDB(&dbCfg)
	if err != nil {
		log.Error("Fatal error: failed to create database connection", "err", err)
		panic(err)
	}

	redisCfg := config.Cfg.Redis
	err = redis.InitRedis(redisCfg)
	if err != nil {
		log.Error("Fatal error: failed to create redis connection", "err", err)
		panic(err)
	}

	err = minio.Init()
	if err != nil {
		log.Error("Fatal error: failed to initialize MinIO", "err", err)
		panic(err)
	}

	app, err := wire.BuildApplication(db, cfg)
	if err != nil {
		log.Error("Fatal error: failed to create application", "err", err)
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	// HTTP 服务器
	srv := &http.Server{
		Addr:    ":8080",
		Handler: app.Router,
	}

	g.Go(func() error {
		log.Info("HTTP Server starting...")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	// Kafka 消费者
	g.Go(func() error {
		log.Info("Kafka Consumers starting...")
		return app.KafkaManager.Start(ctx, cfg)
	})

	// llm 模型初始化
	g.Go(func() error {
		log.Info("LLM Models starting...")
		return llm.InitLLM()
	})

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
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("HTTP Server shutdown failed", "err", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("App exited with error", "err", err)
	}
	log.Info("App exited successfully.")
}

func InitLogger() {
	handler := log.NewJSONHandler(os.Stdout, &log.HandlerOptions{
		Level: log.LevelDebug,
	})

	logger := log.New(handler)
	log.SetDefault(logger)
}
