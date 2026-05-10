# Filebeat Events Exporter

Filebeat 指标采集工具，用于将 Filebeat Stats API 的数据转换为 Prometheus 格式，通过 Categraf 采集后推送到 N9e 展示。

## 功能特性

| 功能 | 说明 |
|------|------|
| **标签扩展** | 支持自定义标签（cluster、env 等） |
| **指标过滤** | 通过正则表达式过滤需要的指标 |
| **失败重试** | API 调用失败时的自动重试机制 |
| **健康检查** | 采集前检查 Filebeat 服务状态 |

## 项目结构1

```
filebeat-events/
├── main.go              # 程序入口
├── go.mod               # Go 依赖
├── config/
│   └── config.go        # 配置解析
├── collector/
│   └── filebeat.go      # 数据采集（含重试、健康检查）
└── metrics/
    └── converter.go     # 指标转换（含过滤）
```

## 使用方式

### 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-endpoint` | `http://localhost:5066/stats` | Filebeat Stats API 地址 |
| `-labels` | `` | 自定义标签，格式：`key1=value1,key2=value2` |
| `-filter` | `` | 指标过滤正则表达式 |
| `-max-retries` | `3` | 最大重试次数 |
| `-retry-delay` | `2s` | 重试延迟 |
| `-timeout` | `10s` | 请求超时时间 |
| `-debug` | `false` | 启用调试模式 |

### 环境变量

| 环境变量 | 说明 |
|----------|------|
| `FILEBEAT_ENDPOINT` | Filebeat Stats API 地址 |
| `FILEBEAT_LABELS` | 自定义标签 |
| `FILEBEAT_FILTER` | 指标过滤正则表达式 |

### 使用示例

```bash
# 基础使用
./filebeat-events

# 指定 endpoint
./filebeat-events -endpoint http://192.168.1.100:5066/stats

# 添加自定义标签
./filebeat-events -labels "cluster=prod,env=us-east"

# 过滤指标（只采集 events 相关）
./filebeat-events -filter "^filebeat_events_"

# 调整重试策略
./filebeat-events -max-retries 5 -retry-delay 3s

# 调试模式
./filebeat-events -debug
```

## Categraf 配置

在 `conf/input.exec/filebeat.toml` 中配置：

```toml
[[instances]]
## 采集程序路径
command = "/path/to/filebeat-events -endpoint=http://filebeat:5066 -labels=cluster=prod"

## 数据格式
data_format = "prometheus"

## 采集间隔
interval = "15s"

## 超时设置
timeout = "10s"

## 标签
labels = { source = "filebeat" }
```

## 输出示例

```
# HELP filebeat_events_published Total number of events published
# TYPE filebeat_events_published counter
filebeat_events_published{cluster="prod",env="us-east",host="filebeat-1"} 123456

# HELP filebeat_events_publish_failed Total number of events failed to publish
# TYPE filebeat_events_publish_failed counter
filebeat_events_publish_failed{cluster="prod",env="us-east",host="filebeat-1"} 12

# HELP filebeat_events_active Number of active events
# TYPE filebeat_events_active gauge
filebeat_events_active{cluster="prod",env="us-east",host="filebeat-1"} 42
```

## Filebeat 配置

确保 Filebeat 启用 HTTP API：

```yaml
# filebeat.yml
http.enabled: true
http.host: "0.0.0.0"
http.port: 5066
```

## 构建

```bash
# 编译
go build -o filebeat-events

# 交叉编译 Linux
GOOS=linux GOARCH=amd64 go build -o filebeat-events
go env -w GOOS=linux GOARCH=amd64 && go build -o filebeat-events && go env -w GOOS=windows GOARCH=amd64
# 交叉编译 Windows
GOOS=windows GOARCH=amd64 go build -o filebeat-events.exe
```

## 指标列表

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `filebeat_events_published` | counter | 已发布事件总数 |
| `filebeat_events_published_bytes` | counter | 已发布事件字节数 |
| `filebeat_events_failed` | counter | 失败事件总数 |
| `filebeat_events_publish_failed` | counter | 发布失败事件数 |
| `filebeat_events_retried` | counter | 重试事件数 |
| `filebeat_events_duplicated` | counter | 重复事件数 |
| `filebeat_events_active` | gauge | 当前活跃事件数 |
| `filebeat_events_acked` | counter | 已确认事件数 |
| `filebeat_events_not_acked` | gauge | 未确认事件数 |
| `filebeat_libbeat_module_running` | gauge | 运行中的模块数 |
| `filebeat_libbeat_pipeline_events_published` | counter | 管道已发布事件数 |
| `filebeat_libbeat_pipeline_events_active` | gauge | 管道活跃事件数 |
| `filebeat_libbeat_output_events_total` | counter | 输出事件总数 |
| `filebeat_libbeat_output_events_failed` | counter | 输出失败事件数 |
| `filebeat_libbeat_output_events_successful` | counter | 输出成功事件数 |
| `filebeat_system_cpu_percent` | gauge | CPU 使用率百分比 |
| `filebeat_system_memory_alloc_bytes` | gauge | 内存分配字节数 |
| `filebeat_system_memory_sys_bytes` | gauge | 系统内存字节数 |
