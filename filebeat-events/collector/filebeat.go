package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"filebeat-events/config"
)

// Stats Filebeat Stats API 返回的数据结构
type Stats struct {
	Events struct {
		Published      uint64 `json:"published"`
		PublishedBytes uint64 `json:"published_bytes"`
		Failed         uint64 `json:"failed"`
		PublishFails   uint64 `json:"publish_failed"`
		Retried        uint64 `json:"retried"`
		Duplicated     uint64 `json:"duplicated"`
		Active         uint64 `json:"active"`
		Acked          uint64 `json:"acked"`
		NotAcked       uint64 `json:"not_acked"`
	} `json:"events"`

	Libbeat struct {
		Config struct {
			Module struct {
				Running uint64 `json:"running"`
			} `json:"module"`
		} `json:"config"`
		Pipeline struct {
			Events struct {
				Published uint64 `json:"published"`
				Active    uint64 `json:"active"`
			} `json:"events"`
		} `json:"pipeline"`
		Output struct {
			Type  string `json:"type"`
			Events struct {
				Total      uint64 `json:"total"`
				Failed     uint64 `json:"failed"`
				Successful uint64 `json:"successful"`
			} `json:"events"`
		} `json:"output"`
	} `json:"libbeat"`

	Registries struct {
		Registries map[string]struct {
			Events struct {
				Published uint64 `json:"published"`
				Failed    uint64 `json:"failed"`
			} `json:"events"`
		} `json:"registries"`
	} `json:"registries"`

	System struct {
		CPU struct {
			Total struct {
				Pct  float64 `json:"pct"`
				Norm float64 `json:"norm"`
				Ticks float64 `json:"ticks"`
			} `json:"total"`
		} `json:"cpu"`
		Memory struct {
			Alloc     uint64 `json:"alloc"`
			TotalAlloc uint64 `json:"total_alloc"`
			Sys      uint64 `json:"sys"`
		} `json:"memory"`
	} `json:"system"`
}

// Collector 数据采集器
type Collector struct {
	cfg       *config.Config
	client    *http.Client
	filter    *regexp.Regexp
	hostLabel string
}

// NewCollector 创建新的采集器
func NewCollector(cfg *config.Config) (*Collector, error) {
	c := &Collector{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		hostLabel: extractHost(cfg.Endpoint),
	}

	// 编译过滤器
	if cfg.FilterRegex != "" {
		var err error
		c.filter, err = regexp.Compile(cfg.FilterRegex)
		if err != nil {
			return nil, fmt.Errorf("invalid filter regex: %w", err)
		}
	}

	return c, nil
}

// extractHost 从 endpoint 提取主机标识
func extractHost(endpoint string) string {
	parts := strings.Split(strings.TrimPrefix(endpoint, "http://"), ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

// HealthCheck 检查 Filebeat 健康状态
func (c *Collector) HealthCheck(ctx context.Context) error {
	url := strings.Replace(c.cfg.Endpoint, "/stats", "/health", 1)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Collect 采集数据
func (c *Collector) Collect(ctx context.Context) (*Stats, error) {
	var lastErr error

	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			if c.cfg.Debug {
				fmt.Fprintf(os.Stderr, "Retry attempt %d/%d...\n", attempt, c.cfg.MaxRetries)
			}
			select {
			case <-time.After(c.cfg.RetryDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		stats, err := c.fetch(ctx)
		if err == nil {
			return stats, nil
		}

		lastErr = err
		if c.cfg.Debug {
			fmt.Fprintf(os.Stderr, "Attempt %d failed: %v\n", attempt+1, err)
		}
	}

	return nil, fmt.Errorf("after %d attempts, last error: %w", c.cfg.MaxRetries+1, lastErr)
}

// fetch 执行 HTTP 请求获取数据
func (c *Collector) fetch(ctx context.Context) (*Stats, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.cfg.Endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var stats Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("decode json failed: %w", err)
	}

	return &stats, nil
}

// ShouldInclude 判断指标是否应该包含（基于过滤规则）
func (c *Collector) ShouldInclude(metricName string) bool {
	if c.filter == nil {
		return true
	}
	return c.filter.MatchString(metricName)
}

// GetHostLabel 获取主机标签
func (c *Collector) GetHostLabel() string {
	return c.hostLabel
}

// GetLabels 获取所有标签
func (c *Collector) GetLabels() map[string]string {
	labels := make(map[string]string)
	for k, v := range c.cfg.Labels {
		labels[k] = v
	}
	labels["host"] = c.hostLabel
	return labels
}
