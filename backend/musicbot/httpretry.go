package musicbot

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	discordRetries    = 2
	discordRetryDelay = 250 * time.Millisecond
)

type retryTransport struct {
	base http.RoundTripper
}

func NewDiscordHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   20 * time.Second,
		Transport: retryTransport{base: http.DefaultTransport},
	}
}

func retryableRequest(req *http.Request) bool {
	switch req.Method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	case http.MethodPost:
		return strings.HasPrefix(req.URL.Path, "/api/") && strings.Contains(req.URL.Path, "/interactions/") && strings.HasSuffix(req.URL.Path, "/callback")
	}
	return false
}

func retryableStatus(code int) bool {
	return code == http.StatusBadGateway || code == http.StatusServiceUnavailable || code == http.StatusGatewayTimeout
}

func (t retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	canRetry := retryableRequest(req) && (req.Body == nil || req.GetBody != nil)

	for attempt := 0; ; attempt++ {
		resp, err := t.base.RoundTrip(req)
		failed := err != nil || retryableStatus(resp.StatusCode)
		if !canRetry || !failed || attempt >= discordRetries || req.Context().Err() != nil {
			return resp, err
		}

		if resp != nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
			resp.Body.Close()
		}
		if req.GetBody != nil {
			body, bodyErr := req.GetBody()
			if bodyErr != nil {
				return nil, bodyErr
			}
			req.Body = body
		}

		slog.Warn("retrying discord request",
			slog.String("method", req.Method),
			slog.String("path", req.URL.Path),
			slog.Int("attempt", attempt+1),
		)

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(discordRetryDelay * time.Duration(attempt+1)):
		}
	}
}
