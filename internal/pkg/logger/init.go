package logger

import (
	"Cornerstone/internal/api/config"
	"io"
	log "log/slog"
	"net"
	"os"
)

var LogWriter io.Writer

func InitLogger() {
	cfg := config.Cfg.Logstash

	hStdout := log.NewJSONHandler(os.Stdout, &log.HandlerOptions{Level: log.LevelInfo})

	var finalHandler log.Handler = hStdout

	conn, err := net.Dial("tcp", cfg.Address)
	if err == nil {
		hRemote := log.NewJSONHandler(conn, &log.HandlerOptions{Level: log.LevelInfo}).
			WithAttrs([]log.Attr{
				log.String("target_index", cfg.Index),
				log.String("log_token", "raqtpie_secret_2026"),
			})

		filterRemote := &RemoteFilterHandler{next: hRemote}

		finalHandler = &TeeHandler{
			handlers: []log.Handler{hStdout, filterRemote},
		}

		LogWriter = conn
	} else {
		LogWriter = os.Stdout
		log.Warn("Failed to connect to Logstash, logging to stdout only", "err", err)
	}

	logger := log.New(&ContextHandler{finalHandler})
	log.SetDefault(logger)
}
