# pingfabric

`pingfabric` 是一个从 MySQL 读取 VIP 列表并输出 Prometheus 指标的连通性探测工具。当前实现会读取 `metadata.cmdb_vip` 及其关联的 `cmdb_vip_rserver`，按协议和端口做批量拨测，适合接入文本采集链路或定时探活任务。

## 功能特点

- 从 `cmdb_vip` 读取 VIP，并通过 `LEFT JOIN cmdb_vip_rserver` 补充 `app_name`
- 支持 `tcp` / `udp` 探测
- 默认 200 个并发 worker
- `dial` 超时时间可配置，默认 `2s`
- 支持按内外网、协议、网络类型筛选
- 输出 Prometheus 文本格式指标

## 项目结构

- `main.go`：程序入口、参数解析、数据库查询、并发探测、指标输出
- `db.sql`：`cmdb_vip` 建表语句
- `rserver.sql`：`cmdb_vip_rserver` 建表语句

## 环境要求

- Go 1.24+

## 数据源与固定筛选

程序固定连接以下数据库：

- 地址：`llmdms.master.db.bigdata.com:3309`
- 数据库：`metadata`
- 主表：`cmdb_vip`
- 关联表：`cmdb_vip_rserver`
- 用户：`pingfabric_ro`

除命令行参数外，当前代码还固定附加以下筛选条件：

- `vip.status = '使用中'`
- `vip.env = '生产'`
- `vip.source = 'f5'`
- `vip.deleted = 0`
- `vip.idc IN ('周浦', '浦江SH16')`
- `vip.department IN ('业务支持中心集团分析部', '创新业务事业群创新业务部', '创新业务事业群Choice业务部')`
- `rs.deleted = 0`
- `rs.status = 'up'`

`--type` 会映射到数据库里的中文枚举：

- `internal -> 内网`
- `external -> 外网`

`--network_type` 的行为：

- `all`：不追加 `ip_type` 条件
- `ipv4`：追加 `vip.ip_type = 'ipv4'`
- `ipv6`：追加 `vip.ip_type = 'ipv6'`

`--protocol` 的行为：

- 默认值是 `tcp`
- `all`：不追加 `vip.protocol` 条件
- `tcp` / `udp`：追加对应的 `vip.protocol` 条件
- 实际拨测时使用数据库记录里的 `vip.protocol`，不是命令行参数本身

## 参数说明

运行 `go run . --help` 或 `./pingfabric --help` 可查看帮助。

当前支持的参数：

- `--workers`：并发 worker 数，默认 `200`
- `--type`：VIP 类型，`internal` 或 `external`，默认 `external`
- `--network_type`：网络类型，`ipv4`、`ipv6` 或 `all`，默认 `all`
- `--protocol`：协议过滤，`tcp`、`udp` 或 `all`，默认 `tcp`
- `--timeout`：拨测超时时间，默认 `2s`
- `-h` / `--help`：打印帮助

## 快速开始

直接运行：

```bash
go run .
```

带参数运行：

```bash
go run . --type external --network_type all --protocol tcp --timeout 2s
go run . --type internal --network_type ipv6 --protocol udp --workers 100 --timeout 5s
```

构建：

```bash
go build -o pingfabric .
```

测试：

```bash
go test ./...
```

## 输出指标

程序当前会输出以下指标：

- `net_response_result_code{target="...",type="pingfabric",protocol="...",application="..."}`
- `net_response_response_time{target="...",type="pingfabric",protocol="...",application="..."}`
- `pingfabric_used_time{stage="load_record"}`
- `pingfabric_total_time{stage="total"}`
- `pingfabric_total_endpoint`

说明：

- `application` label 仅在 `app_name` 非空时输出
- 如果数据库中的 `app_name` 是逗号分隔字符串，只取第一个非空值
- `protocol` label 来自数据库中的 `vip.protocol`

结果码定义：

- `0`：连接成功
- `1`：连接超时
- `2`：连接失败

示例输出：

```txt
net_response_result_code{target="1.2.3.4:443",type="pingfabric",protocol="tcp",application="choice"} 0
net_response_response_time{target="1.2.3.4:443",type="pingfabric",protocol="tcp",application="choice"} 0.012
pingfabric_total_endpoint 128
```

## Categraf 采集

建议通过 Categraf 的 `input.exec` 插件采集本程序输出，采集间隔按你的探测频率配置。

示例配置：

```toml
[[instances]]
interval = "300s"

[[instances.commands]]
command = "/path/to/pingfabric --type external --network_type all --protocol tcp --timeout 2s"
timeout = "60s"
data_format = "prometheus"
```

## 注意事项

- 当前实现依赖数据库在线可达；本地无法访问线上环境时，只能做编译校验，不能做真实拨测验证
- 当前仓库还没有测试文件，新增行为时建议补充 `*_test.go`
- 不要提交本地构建产物，例如 `pingfabric`
