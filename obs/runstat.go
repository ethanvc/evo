package obs

import "github.com/prometheus/client_golang/prometheus"

func EnableRunStat(reg prometheus.Registerer) {
	evoGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "evo_gauge_info",
		Help: "unified gauge for obs",
	}, []string{"key"})

}
