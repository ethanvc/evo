package obs

import (
	"github.com/ethanvc/evo/runstat"
	"github.com/prometheus/client_golang/prometheus"
)

const cpuUsageCounterKey = "cpu_usage_seconds_total"

var cpuUsageCounterDesc = prometheus.NewDesc(
	"evo_cpu_usage_seconds_total",
	"Cumulative CPU usage seconds for the runtime environment",
	nil,
	nil,
)

func cpuStatSampler() (map[string]float64, map[string]float64) {
	info, err := runstat.GetCPU()
	if err != nil {
		return nil, nil
	}
	return map[string]float64{
		"cpu_limit_cores": info.LimitCores,
	}, map[string]float64{
		cpuUsageCounterKey: info.UsageSeconds,
	}
}
