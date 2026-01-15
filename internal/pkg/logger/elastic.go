package logger

import (
	"bytes"
	"io"
	log "log/slog"
	"net/http"
	"time"
)

type ESTransport struct {
	Transport http.RoundTripper
}

func (t *ESTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	var reqBody []byte
	if req.Body != nil {
		reqBody, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
	}

	resp, err := t.Transport.RoundTrip(req)
	elapsed := time.Since(start)

	fields := []any{
		log.String("method", req.Method),
		log.String("url", req.URL.String()),
		log.Duration("latency", elapsed),
	}

	limit := 1000
	reqStr := string(reqBody)
	if len(reqStr) > limit {
		reqStr = reqStr[:limit] + "...[truncated]"
	}
	fields = append(fields, log.String("req_body", reqStr))

	if err != nil {
		log.ErrorContext(req.Context(), "ES_QUERY_ERROR", append(fields, log.Any("err", err))...)
		return nil, err
	}

	var resBody []byte
	if resp.Body != nil {
		resBody, _ = io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewBuffer(resBody))
	}

	resStr := string(resBody)
	if len(resStr) > limit {
		resStr = resStr[:limit] + "...[truncated]"
	}
	fields = append(fields, log.Int("status", resp.StatusCode), log.String("res_body", resStr))

	if elapsed > 500*time.Millisecond {
		log.WarnContext(req.Context(), "ES_QUERY_SLOW", fields...)
	} else {
		log.InfoContext(req.Context(), "ES_QUERY", fields...)
	}

	return resp, nil
}
