package obs

import (
	"github.com/ethanvc/evo/runstat"
	"github.com/prometheus/client_golang/prometheus"
)

type gaugeSampler func() map[string]float64

type evoGaugeCollector struct {
	desc     *prometheus.Desc
	samplers []gaugeSampler
}

func newEvoGaugeCollector() *evoGaugeCollector {
	c := &evoGaugeCollector{
		desc: prometheus.NewDesc(
			"evo_gauge_info",
			"unified gauge for obs",
			[]string{"key"},
			nil,
		),
	}
	c.samplers = append(c.samplers, memoryGaugeSampler)
	return c
}

func memoryGaugeSampler() map[string]float64 {
	info, err := runstat.GetMemory()
	if err != nil {
		return nil
	}
	return map[string]float64{
		"memory_limit":   float64(info.MaxBytes),
		"memory_current": float64(info.UsedBytes),
	}
}

func (c *evoGaugeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *evoGaugeCollector) Collect(ch chan<- prometheus.Metric) {
	for _, sampler := range c.samplers {
		for key, val := range sampler() {
			ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, val, key)
		}
	}
}

// EnableRunStat registers runtime gauges collected on each prometheus scrape.
func EnableRunStat(reg prometheus.Registerer) {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	reg.MustRegister(newEvoGaugeCollector())
}
