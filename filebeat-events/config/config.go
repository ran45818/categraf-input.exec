package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// Config 配置结构
type Config struct {
	Endpoint    string            // Filebeat Stats API 地址
	Labels      map[string]string // 自定义标签
	FilterRegex string            // 指标过滤正则表达式
	MaxRetries  int               // 最大重试次数
	RetryDelay  time.Duration     // 重试延迟
	Timeout     time.Duration     // 请求超时
	Debug       bool              // 调试模式
}

// ParseFlags 解析命令行参数
func ParseFlags() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.Endpoint, "endpoint", "http://localhost:5066/stats", "Filebeat Stats API endpoint")
	flag.StringVar(&cfg.FilterRegex, "filter", "", "Regex pattern to filter metrics")
	flag.IntVar(&cfg.MaxRetries, "max-retries", 3, "Max retry times on failure")
	flag.DurationVar(&cfg.RetryDelay, "retry-delay", 2*time.Second, "Delay between retries")
	flag.DurationVar(&cfg.Timeout, "timeout", 10*time.Second, "HTTP request timeout")
	flag.BoolVar(&cfg.Debug, "debug", false, "Enable debug mode")

	labels := flag.String("labels", "", "Custom labels in format: key1=value1,key2=value2")

	flag.Parse()

	// 解析标签
	cfg.Labels = make(map[string]string)
	if *labels != "" {
		pairs := strings.Split(*labels, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) != 2 {
				return nil, fmt.Errorf("invalid label format: %s", pair)
			}
			cfg.Labels[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	// 从环境变量读取
	if envEndpoint := os.Getenv("FILEBEAT_ENDPOINT"); envEndpoint != "" {
		cfg.Endpoint = envEndpoint
	}
	if envLabels := os.Getenv("FILEBEAT_LABELS"); envLabels != "" {
		cfg.Labels = parseLabels(envLabels)
	}
	if envFilter := os.Getenv("FILEBEAT_FILTER"); envFilter != "" {
		cfg.FilterRegex = envFilter
	}

	return cfg, nil
}

func parseLabels(s string) map[string]string {
	labels := make(map[string]string)
	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			labels[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return labels
}
