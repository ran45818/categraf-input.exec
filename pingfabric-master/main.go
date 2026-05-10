package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	workerCount = 200
	queueSize   = 500

	dbHost  = "llmdms.master.db.bigdata.com"
	dbPort  = 3309
	dbUser  = "pingfabric_ro"
	dbName  = "metadata"
	dbTable = "cmdb_vip"
	dbPass  = "RDcMlfGFEEg13RxkqiV$"

	vipTypeInternal = "internal"
	vipTypeExternal = "external"

	networkTypeIPv4 = "ipv4"
	networkTypeIPv6 = "ipv6"
	networkTypeAll  = "all"

	protocolTCP = "tcp"
	protocolUDP = "udp"
	protocolAll = "all"

	dbVIPTypeInternal = "内网"
	dbVIPTypeExternal = "外网"
	activeVIPStatus   = "使用中"
	selectedVIPEnv    = "生产"
	selectedSource    = "f5"
	probeTypeLabel    = "pingfabric"
	probeTimeout      = 2 * time.Second
)

var (
	selectedVIPdepartment = []string{"业务支持中心集团分析部", "创新业务事业群创新业务部", "创新业务事业群Choice业务部"}
	selectedVIPidc        = []string{"周浦", "浦江SH16"}
)

const (
	resultSuccess uint64 = iota
	resultTimeout
	resultConnectionFailed
)

type cmdbVIP struct {
	ID      int64  `gorm:"column:id"`
	VIPPort string `gorm:"column:vip_port"`
	VIP     string `gorm:"column:vip"`
	VPort   int    `gorm:"column:vport"`
	Status  string `gorm:"column:status"`
	Type    string `gorm:"column:type"`
	Deleted int    `gorm:"column:deleted"`
}

type vipJoinRow struct {
	ID          int64  `gorm:"column:id"`
	VIPPort     string `gorm:"column:vip_port"`
	VIP         string `gorm:"column:vip"`
	VPort       int    `gorm:"column:vport"`
	Protocol    string `gorm:"column:protocol"`
	Status      string `gorm:"column:status"`
	Type        string `gorm:"column:type"`
	Deleted     int    `gorm:"column:deleted"`
	BackendHost string `gorm:"column:backend_host"`
	BackendPort int    `gorm:"column:backend_port"`
	AppName     string `gorm:"column:app_name"`
}

type probeTarget struct {
	Endpoint string
	Protocol string
	AppName  string
}

type probeResult struct {
	Target       probeTarget
	ResponseTime float64
	ResultCode   uint64
}

type endpointLoader struct {
	db *gorm.DB
}

type config struct {
	workers     int
	vipType     string
	networkType string
	protocol    string
	timeout     time.Duration
}

func main() {
	cfg := mustParseConfig()
	startedAt := time.Now()
	metricSet := metrics.NewSet()

	targets, err := loadTargets(cfg)
	if err != nil {
		log.Fatalf("load probe targets from mysql: %v", err)
	}
	if len(targets) == 0 {
		log.Fatalf("no probe endpoints found in %s.%s", dbName, dbTable)
	}

	load := metricSet.NewGauge(`pingfabric_used_time{stage="load_record"}`, nil)
	load.Set(float64(time.Since(startedAt).Seconds()))

	results := runProbes(targets, cfg.workers, cfg.timeout)
	registerProbeMetrics(metricSet, results)

	total := metricSet.NewGauge(`pingfabric_total_time{stage="total"}`, nil)
	total.Set(float64(time.Since(startedAt).Seconds()))

	totalEndpoint := metricSet.NewGauge(`pingfabric_total_endpoint`, nil)
	totalEndpoint.Set(float64(len(targets)))

	metricSet.WritePrometheus(os.Stdout)
}

func mustParseConfig() config {
	cfg := config{}
	showHelp := false

	flag.Usage = func() {
		printHelp()
	}
	flag.IntVar(&cfg.workers, "workers", workerCount, "number of concurrent probe workers")
	flag.StringVar(&cfg.vipType, "type", vipTypeExternal, "cmdb_vip type to probe: internal or external")
	flag.StringVar(&cfg.networkType, "network_type", networkTypeAll, "network type filter: ipv4, ipv6, or all")
	flag.StringVar(&cfg.protocol, "protocol", protocolTCP, "protocol filter: tcp, udp, or all")
	flag.DurationVar(&cfg.timeout, "timeout", probeTimeout, "dial timeout, e.g. 2s or 500ms")
	flag.BoolVar(&showHelp, "h", false, "show help")
	flag.BoolVar(&showHelp, "help", false, "show help")
	flag.Parse()
	if showHelp {
		printHelp()
		os.Exit(0)
	}

	cfg.vipType = strings.ToLower(strings.TrimSpace(cfg.vipType))
	cfg.networkType = strings.ToLower(strings.TrimSpace(cfg.networkType))
	cfg.protocol = strings.ToLower(strings.TrimSpace(cfg.protocol))
	if cfg.workers <= 0 {
		log.Fatalf("invalid --workers value %d: must be greater than 0", cfg.workers)
	}
	if cfg.vipType != vipTypeInternal && cfg.vipType != vipTypeExternal {
		log.Fatalf(`invalid --type value %q: must be "%s" or "%s"`, cfg.vipType, vipTypeInternal, vipTypeExternal)
	}
	if cfg.networkType != networkTypeIPv4 && cfg.networkType != networkTypeIPv6 && cfg.networkType != networkTypeAll {
		log.Fatalf(`invalid --network_type value %q: must be "%s", "%s", or "%s"`, cfg.networkType, networkTypeIPv4, networkTypeIPv6, networkTypeAll)
	}
	if cfg.protocol != protocolTCP && cfg.protocol != protocolUDP && cfg.protocol != protocolAll {
		log.Fatalf(`invalid --protocol value %q: must be "%s", "%s", or "%s"`, cfg.protocol, protocolTCP, protocolUDP, protocolAll)
	}
	if cfg.timeout <= 0 {
		log.Fatalf("invalid --timeout value %s: must be greater than 0", cfg.timeout)
	}

	return cfg
}

func printHelp() {
	name := filepath.Base(os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n\n", name)
	fmt.Fprintln(flag.CommandLine.Output(), "Options:")
	flag.PrintDefaults()
	fmt.Fprintln(flag.CommandLine.Output())
	fmt.Fprintf(flag.CommandLine.Output(), "Examples:\n")
	fmt.Fprintf(flag.CommandLine.Output(), "  %s --type external --network_type all --protocol tcp --timeout 2s\n", name)
	fmt.Fprintf(flag.CommandLine.Output(), "  %s --type internal --network_type ipv6 --protocol udp --workers 100 --timeout 5s\n", name)
}

func loadTargets(cfg config) ([]probeTarget, error) {
	db, err := openDB()
	if err != nil {
		return nil, err
	}

	loader := endpointLoader{db: db}
	rows, err := loader.loadProbeTargetRows(cfg.vipType, cfg.protocol, cfg.networkType)
	if err != nil {
		return nil, err
	}

	return buildProbeTargets(rows), nil
}

func openDB() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser,
		dbPass,
		dbHost,
		dbPort,
		dbName,
	)

	return gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

func (l endpointLoader) loadProbeTargetRows(vipType string, protocol string, networkType string) ([]vipJoinRow, error) {
	var rows []vipJoinRow
	dbVIPType := dbVIPTypeExternal
	if vipType == vipTypeInternal {
		dbVIPType = dbVIPTypeInternal
	}

	query := l.db.
		Table(dbTable+" vip").
		Select(`
			vip.id,
			vip.vip_port,
			vip.vip,
			vip.vport,
			vip.protocol,
			vip.status,
			vip.type,
			vip.deleted,
			rs.server AS backend_host,
			rs.port AS backend_port,
			rs.app_name
		`).
		Joins(`
			LEFT JOIN cmdb_vip_rserver rs
				ON rs.cmdb_id = vip.id
				AND rs.deleted = 0
				AND rs.status = ?
		`, "up").
		Where(
			"vip.type = ? AND vip.source = ? AND vip.status = ? AND vip.deleted = 0 AND vip.env = ? AND vip.idc IN ? AND vip.department IN ?",
			dbVIPType,
			selectedSource,
			activeVIPStatus,
			selectedVIPEnv,
			selectedVIPidc,
			selectedVIPdepartment,
		)
	if protocol != protocolAll {
		query = query.Where("vip.protocol = ?", protocol)
	}
	if networkType != networkTypeAll {
		query = query.Where("vip.ip_type = ?", networkType)
	}

	err := query.Order("vip.id ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func buildProbeTargets(rows []vipJoinRow) []probeTarget {
	targetByEndpoint := make(map[string]*probeTarget, len(rows))
	orderedEndpoints := make([]string, 0, len(rows))

	for _, row := range rows {
		vip := cmdbVIP{
			ID:      row.ID,
			VIPPort: row.VIPPort,
			VIP:     row.VIP,
			VPort:   row.VPort,
			Status:  row.Status,
			Type:    row.Type,
			Deleted: row.Deleted,
		}

		endpoint, ok := vip.Endpoint()
		if !ok {
			continue
		}

		protocol := normalizeProtocol(row.Protocol)
		if protocol == "" {
			continue
		}

		targetKey := endpoint + "|" + protocol

		target, exists := targetByEndpoint[targetKey]
		if !exists {
			target = &probeTarget{Endpoint: endpoint, Protocol: protocol}
			targetByEndpoint[targetKey] = target
			orderedEndpoints = append(orderedEndpoints, targetKey)
		}

		if target.AppName == "" {
			target.AppName = firstAppName(row.AppName)
		}
	}

	targets := make([]probeTarget, 0, len(orderedEndpoints))
	for _, targetKey := range orderedEndpoints {
		target := targetByEndpoint[targetKey]
		targets = append(targets, *target)
	}

	return targets
}

func (v cmdbVIP) Endpoint() (string, bool) {
	if endpoint := normalizeEndpoint(v.VIPPort); endpoint != "" {
		return endpoint, true
	}

	host := strings.TrimSpace(v.VIP)
	if host == "" || v.VPort <= 0 {
		return "", false
	}

	return normalizeEndpoint(net.JoinHostPort(strings.Trim(host, "[]"), strconv.Itoa(v.VPort))), true
}

func runProbes(targets []probeTarget, workers int, timeout time.Duration) []probeResult {
	jobs := make(chan probeTarget, queueSize)
	results := make(chan probeResult, len(targets))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for target := range jobs {
				results <- probe(target, timeout)
			}
		}()
	}

	for _, target := range targets {
		jobs <- target
	}
	close(jobs)

	wg.Wait()
	close(results)

	probeResults := make([]probeResult, 0, len(targets))
	for result := range results {
		probeResults = append(probeResults, result)
	}

	return probeResults
}

func probe(target probeTarget, timeout time.Duration) probeResult {
	startedAt := time.Now()
	conn, err := net.DialTimeout(target.Protocol, target.Endpoint, timeout)

	result := probeResult{
		Target:       target,
		ResponseTime: time.Since(startedAt).Seconds(),
		ResultCode:   resultConnectionFailed,
	}

	if err == nil {
		result.ResultCode = resultSuccess
		_ = conn.Close()
		return result
	}

	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		result.ResultCode = resultTimeout
	}

	return result
}

func registerProbeMetrics(metricSet *metrics.Set, results []probeResult) {
	for _, result := range results {
		result := result
		labels := prometheusLabels(result.Target)
		metricSet.NewGauge(
			fmt.Sprintf("net_response_result_code{%s}", labels),
			func() float64 { return float64(result.ResultCode) },
		)
		metricSet.NewGauge(
			fmt.Sprintf("net_response_response_time{%s}", labels),
			func() float64 { return result.ResponseTime },
		)
	}
}

func prometheusLabels(target probeTarget) string {
	labels := []string{
		fmt.Sprintf(`target="%s"`, escapeLabelValue(target.Endpoint)),
		fmt.Sprintf(`type="%s"`, probeTypeLabel),
		fmt.Sprintf(`protocol="%s"`, escapeLabelValue(target.Protocol)),
	}
	if target.AppName != "" {
		labels = append(labels, fmt.Sprintf(`service="%s"`, escapeLabelValue(target.AppName)))
	}
	return strings.Join(labels, ", ")
}

func normalizeProtocol(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case protocolTCP:
		return protocolTCP
	case protocolUDP:
		return protocolUDP
	default:
		return ""
	}
}

func firstAppName(raw string) string {
	for _, part := range strings.Split(raw, ",") {
		name := strings.TrimSpace(part)
		if name != "" {
			return name
		}
	}
	return ""
}

func escapeLabelValue(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, "\n", `\n`, `"`, `\"`)
	return replacer.Replace(value)
}

func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}

	if host, port, err := net.SplitHostPort(endpoint); err == nil {
		return net.JoinHostPort(strings.Trim(host, "[]"), port)
	}

	if strings.HasPrefix(endpoint, "[") {
		return endpoint
	}

	if strings.Count(endpoint, ":") > 1 {
		parts := strings.Split(endpoint, ":")
		host := strings.Join(parts[:len(parts)-1], ":")
		port := parts[len(parts)-1]
		if host == "" || port == "" {
			return ""
		}
		return net.JoinHostPort(host, port)
	}

	return endpoint
}
