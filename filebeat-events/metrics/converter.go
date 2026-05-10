package metrics

import (
	"fmt"
	"strings"

	"filebeat-events/collector"
)

// Metric 指标定义
type Metric struct {
	Name   string
	Type   string // counter, gauge
	Value  uint64
	Help   string
	Labels map[string]string
}

// Converter 指标转换器
type Converter struct {
	collector *collector.Collector
}

// NewConverter 创建转换器
func NewConverter(c *collector.Collector) *Converter {
	return &Converter{collector: c}
}

// Convert 转换为 Prometheus 格式
func (cv *Converter) Convert(stats *collector.Stats) []Metric {
	var metrics []Metric
	labels := cv.collector.GetLabels()

	// Events 指标
	metrics = append(metrics, []Metric{
		{
			Name:   "filebeat_events_published",
			Type:   "counter",
			Value:  stats.Events.Published,
			Help:   "Total number of events published",
			Labels: labels,
		},
		{
			Name:   "filebeat_events_published_bytes",
			Type:   "counter",
			Value:  stats.Events.PublishedBytes,
			Help:   "Total bytes of events published",
			Labels: labels,
		},
		{
			Name:   "filebeat_events_failed",
			Type:   "counter",
			Value:  stats.Events.Failed,
			Help:   "Total number of events failed",
			Labels: labels,
		},
		{
			Name:   "filebeat_events_publish_failed",
			Type:   "counter",
			Value:  stats.Events.PublishFails,
			Help:   "Total number of events failed to publish",
			Labels: labels,
		},
		{
			Name:   "filebeat_events_retried",
			Type:   "counter",
			Value:  stats.Events.Retried,
			Help:   "Total number of events retried",
			Labels: labels,
		},
		{
			Name:   "filebeat_events_duplicated",
			Type:   "counter",
			Value:  stats.Events.Duplicated,
			Help:   "Total number of duplicated events",
			Labels: labels,
		},
		{
			Name:   "filebeat_events_active",
			Type:   "gauge",
			Value:  stats.Events.Active,
			Help:   "Number of active events",
			Labels: labels,
		},
		{
			Name:   "filebeat_events_acked",
			Type:   "counter",
			Value:  stats.Events.Acked,
			Help:   "Total number of events acknowledged",
			Labels: labels,
		},
		{
			Name:   "filebeat_events_not_acked",
			Type:   "gauge",
			Value:  stats.Events.NotAcked,
			Help:   "Number of events not acknowledged",
			Labels: labels,
		},
	}...)

	// Libbeat 指标
	metrics = append(metrics, []Metric{
		{
			Name:   "filebeat_libbeat_module_running",
			Type:   "gauge",
			Value:  stats.Libbeat.Config.Module.Running,
			Help:   "Number of running modules",
			Labels: labels,
		},
		{
			Name:   "filebeat_libbeat_pipeline_events_published",
			Type:   "counter",
			Value:  stats.Libbeat.Pipeline.Events.Published,
			Help:   "Total pipeline events published",
			Labels: labels,
		},
		{
			Name:   "filebeat_libbeat_pipeline_events_active",
			Type:   "gauge",
			Value:  stats.Libbeat.Pipeline.Events.Active,
			Help:   "Number of active pipeline events",
			Labels: labels,
		},
		{
			Name:   "filebeat_libbeat_output_type_info",
			Type:   "gauge",
			Value:  1,
			Help:   "Filebeat output type",
			Labels: mergeLabels(labels, map[string]string{"type": stats.Libbeat.Output.Type}),
		},
		{
			Name:   "filebeat_libbeat_output_events_total",
			Type:   "counter",
			Value:  stats.Libbeat.Output.Events.Total,
			Help:   "Total output events",
			Labels: labels,
		},
		{
			Name:   "filebeat_libbeat_output_events_failed",
			Type:   "counter",
			Value:  stats.Libbeat.Output.Events.Failed,
			Help:   "Total output events failed",
			Labels: labels,
		},
		{
			Name:   "filebeat_libbeat_output_events_successful",
			Type:   "counter",
			Value:  stats.Libbeat.Output.Events.Successful,
			Help:   "Total output events successful",
			Labels: labels,
		},
	}...)

	// System 指标
	metrics = append(metrics, []Metric{
		{
			Name:   "filebeat_system_cpu_percent",
			Type:   "gauge",
			Value:  uint64(stats.System.CPU.Total.Pct * 100),
			Help:   "CPU usage percentage",
			Labels: labels,
		},
		{
			Name:   "filebeat_system_memory_alloc_bytes",
			Type:   "gauge",
			Value:  stats.System.Memory.Alloc,
			Help:   "Memory allocated bytes",
			Labels: labels,
		},
		{
			Name:   "filebeat_system_memory_total_alloc_bytes",
			Type:   "counter",
			Value:  stats.System.Memory.TotalAlloc,
			Help:   "Total memory allocated bytes",
			Labels: labels,
		},
		{
			Name:   "filebeat_system_memory_sys_bytes",
			Type:   "gauge",
			Value:  stats.System.Memory.Sys,
			Help:   "System memory bytes",
			Labels: labels,
		},
	}...)

	// Registry 指标（针对每个 registry）
	for name, registry := range stats.Registries.Registries {
		registryLabels := mergeLabels(labels, map[string]string{"registry": name})
		metrics = append(metrics, []Metric{
			{
				Name:   "filebeat_registry_events_published",
				Type:   "counter",
				Value:  registry.Events.Published,
				Help:   "Registry events published",
				Labels: registryLabels,
			},
			{
				Name:   "filebeat_registry_events_failed",
				Type:   "counter",
				Value:  registry.Events.Failed,
				Help:   "Registry events failed",
				Labels: registryLabels,
			},
		}...)
	}

	// 过滤指标
	var filtered []Metric
	for _, m := range metrics {
		if cv.collector.ShouldInclude(m.Name) {
			filtered = append(filtered, m)
		}
	}

	return filtered
}

// ToPrometheusFormat 转换为 Prometheus 文本格式
func (cv *Converter) ToPrometheusFormat(metrics []Metric) string {
	var sb strings.Builder

	// 输出 HELP 和 TYPE
	for _, m := range metrics {
		sb.WriteString(fmt.Sprintf("# HELP %s %s\n", m.Name, m.Help))
		sb.WriteString(fmt.Sprintf("# TYPE %s %s\n", m.Name, m.Type))
	}

	sb.WriteString("\n")

	// 输出指标值
	for _, m := range metrics {
		sb.WriteString(fmt.Sprintf("%s%s %d\n", m.Name, cv.formatLabels(m.Labels), m.Value))
	}

	return sb.String()
}

func (cv *Converter) formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	var parts []string
	for k, v := range labels {
		// 转义标签值
		escaped := strings.ReplaceAll(v, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		escaped = strings.ReplaceAll(escaped, "\n", "\\n")
		parts = append(parts, fmt.Sprintf("%s=\"%s\"", k, escaped))
	}

	return fmt.Sprintf("{%s}", strings.Join(parts, ","))
}

func mergeLabels(base, extra map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range extra {
		result[k] = v
	}
	return result
}
