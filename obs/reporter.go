package obs

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Reporter struct {
	eventLabelNames        []string
	eventSecondsLabelNames []string
	eventTotal             *prometheus.CounterVec
	eventDurationSeconds   *prometheus.HistogramVec
}

func newReporter() *Reporter {
	reporter := &Reporter{}
	reporter.init()
	return reporter
}

func (r *Reporter) init() {
	r.eventLabelNames = []string{"method", "lvl"}
	r.eventSecondsLabelNames = []string{"method"}
	r.eventTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "evo_event_total",
		Help: "Total number of events",
	}, r.eventLabelNames)
	r.eventDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "evo_event_duration_seconds",
		Help: "Duration of events",
	}, r.eventSecondsLabelNames)
	prometheus.MustRegister(r.eventTotal, r.eventDurationSeconds)
}

func (r *Reporter) Report(ctx context.Context, lvl Level, event string, labels ...KV) {
	r.eventTotal.WithLabelValues(getLabelValues(r.eventLabelNames, labels...)...).Inc()
}

func (r *Reporter) ReportDuration(ctx context.Context, duration time.Duration, labels ...KV) {
	r.eventDurationSeconds.WithLabelValues(getLabelValues(r.eventSecondsLabelNames, labels...)...).Observe(duration.Seconds())
}

func getLabelValues(labelNames []string, labels ...KV) []string {
	labelValues := make([]string, len(labelNames))
	for i, labelName := range labelNames {
		for _, label := range labels {
			if label.Key == labelName {
				labelValues[i] = label.Val
			}
		}
	}
	return labelValues
}
