package es

import (
	"Cornerstone/internal/api/config"
	"Cornerstone/internal/pkg/logger"
	"context"
	log "log/slog"
	"net/http"

	"github.com/elastic/go-elasticsearch/v8"
)

var (
	UserIndex string
	PostIndex string
)

const (
	NotFoundCode = 404
	ConflictCode = 409
)

// InitClient 初始化 Elasticsearch 客户端
func InitClient() (*elasticsearch.TypedClient, error) {
	elasticCfg := config.Cfg.Elastic

	UserIndex = elasticCfg.Indices.UserIndex
	PostIndex = elasticCfg.Indices.PostIndex

	cfg := elasticsearch.Config{
		Addresses: []string{elasticCfg.Address},
		Username:  elasticCfg.Username,
		Password:  elasticCfg.Password,
		Transport: &logger.ESTransport{
			Transport: http.DefaultTransport,
		},
	}

	var err error
	client, err := elasticsearch.NewTypedClient(cfg)
	if err != nil {
		log.Error("Cannot Connect to Elasticsearch", "err", err)
		return nil, err
	}

	info, err := client.Info().Do(context.Background())
	if err != nil {
		log.Error("Cannot Connect to Elasticsearch", "err", err)
		return nil, err
	}

	log.Info("Connected to Elasticsearch", "version", info.Version.Int)
	return client, nil
}
