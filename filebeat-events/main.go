package main

import (
	"context"
	"fmt"
	"os"

	"filebeat-events/collector"
	"filebeat-events/config"
	"filebeat-events/metrics"
)

const (
	version = "1.0.0"
)

func main() {
	cfg, err := config.ParseFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// 创建采集器
	c, err := collector.NewCollector(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating collector: %v\n", err)
		os.Exit(1)
	}

	// 健康检查
	if err := c.HealthCheck(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
		os.Exit(1)
	}

	// 采集数据
	stats, err := c.Collect(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Collection failed: %v\n", err)
		os.Exit(1)
	}

	// 转换为 Prometheus 格式
	converter := metrics.NewConverter(c)
	promMetrics := converter.Convert(stats)

	// 输出
	fmt.Print(converter.ToPrometheusFormat(promMetrics))
}
