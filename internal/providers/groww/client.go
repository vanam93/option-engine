package groww

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Client is a reusable HTTP client for Groww APIs.
type Client struct {
	httpClient *http.Client
	baseURL    string
	limiter    *rateLimiter
	cfg        Config
	metrics    *healthMetrics
}

func newClient(cfg Config, metrics *healthMetrics) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        32,
				MaxIdleConnsPerHost: 16,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		baseURL: cfg.BaseURL,
		limiter: newRateLimiter(cfg.RequestsPerSecond),
		cfg:     cfg,
		metrics: metrics,
	}
}

func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}

func (c *Client) GetJSON(ctx context.Context, path string, params map[string]string, out any, token string) error {
	return c.do(ctx, http.MethodGet, path, params, nil, out, token)
}

func (c *Client) PostJSON(ctx context.Context, path string, body any, out any, useAPIKey bool) error {
	token := ""
	if !useAPIKey {
		token = c.cfg.AccessToken
	}
	return c.do(ctx, http.MethodPost, path, nil, body, out, token, useAPIKey)
}

func (c *Client) do(ctx context.Context, method, path string, params map[string]string, body any, out any, token string, useAPIKey ...bool) error {
	withAPIKey := len(useAPIKey) > 0 && useAPIKey[0]
	authToken := token
	if withAPIKey {
		authToken = c.cfg.APIKey
	}

	var lastErr error
	attempts := c.cfg.RetryAttempts + 1
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			backoff := c.cfg.RetryBackoff * time.Duration(1<<(attempt-1))
			c.metrics.recordRetry()
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}

		if err := c.limiter.Wait(ctx); err != nil {
			return err
		}

		start := time.Now()
		c.metrics.recordRequestStart()
		err := c.executeOnce(ctx, method, path, params, body, out, authToken)
		c.metrics.recordRequestEnd(time.Since(start), err)

		if err == nil {
			return nil
		}
		lastErr = err

		var httpErr *HTTPError
		if !errorsAsHTTP(err, &httpErr) {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			continue
		}
		switch httpErr.StatusCode {
		case 429, 500, 502, 503, 504:
			continue
		default:
			return err
		}
	}
	return lastErr
}

func (c *Client) executeOnce(ctx context.Context, method, path string, params map[string]string, body any, out any, token string) error {
	url := c.baseURL + path
	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-VERSION", apiVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if params != nil {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		slog.Warn("groww request failed", "path", path, "error", err)
		return fmt.Errorf("%w: %v", ErrTimeout, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var env APIEnvelope
		_ = json.Unmarshal(raw, &env)
		code, message := "", string(raw)
		if env.Error != nil {
			code = env.Error.Code
			message = env.Error.Message
		}
		return classifyHTTP(resp.StatusCode, code, message)
	}

	if out == nil {
		return nil
	}

	var env APIEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	if env.Status == "FAILURE" && env.Error != nil {
		return classifyHTTP(resp.StatusCode, env.Error.Code, env.Error.Message)
	}
	if env.Payload != nil {
		payloadRaw, err := json.Marshal(env.Payload)
		if err != nil {
			return err
		}
		return json.Unmarshal(payloadRaw, out)
	}
	return json.Unmarshal(raw, out)
}

func errorsAsHTTP(err error, target **HTTPError) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*HTTPError); ok {
		*target = e
		return true
	}
	return false
}
