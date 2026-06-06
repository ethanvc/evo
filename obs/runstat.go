package obs

import "github.com/prometheus/client_golang/prometheus"

type statSampler func() (gauges map[string]float64, counters map[string]float64)

type evoRunStatCollector struct {
	gaugeDesc   *prometheus.Desc
	counterDesc *prometheus.Desc
	samplers    []statSampler
}

func newEvoRunStatCollector() *evoRunStatCollector {
	c := &evoRunStatCollector{
		gaugeDesc: prometheus.NewDesc(
			"evo_gauge_info",
			"unified gauge for obs",
			[]string{"key"},
			nil,
		),
		counterDesc: cpuUsageCounterDesc,
	}
	c.samplers = append(c.samplers, memoryStatSampler, cpuStatSampler)
	return c
}

func (c *evoRunStatCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.gaugeDesc
	ch <- c.counterDesc
}

func (c *evoRunStatCollector) Collect(ch chan<- prometheus.Metric) {
	for _, sampler := range c.samplers {
		gauges, counters := sampler()
		for key, val := range gauges {
			ch <- prometheus.MustNewConstMetric(c.gaugeDesc, prometheus.GaugeValue, val, key)
		}
		if val, ok := counters[cpuUsageCounterKey]; ok {
			ch <- prometheus.MustNewConstMetric(c.counterDesc, prometheus.CounterValue, val)
		}
	}
}

// EnableRunStat registers runtime gauges and counters collected on each prometheus scrape.
func EnableRunStat(reg prometheus.Registerer) {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	reg.MustRegister(newEvoRunStatCollector())
}
