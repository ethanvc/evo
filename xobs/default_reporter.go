package xobs

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type DefaultReporter struct {
	totalLabelNames   []string
	secondsLabelNames []string
	requestTotal      *prometheus.CounterVec
	durationSeconds   *prometheus.HistogramVec
}

func newDefaultReporter() *DefaultReporter {
	reporter := &DefaultReporter{}
	reporter.init()
	return reporter
}

func (r *DefaultReporter) init() {
	r.totalLabelNames = []string{"method", "lvl"}
	r.secondsLabelNames = []string{"method"}
	r.requestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "xobs_request_total",
		Help: "Total number of requests",
	}, r.totalLabelNames)
	r.durationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "xobs_duration_seconds",
		Help: "Duration of requests",
	}, r.secondsLabelNames)
	prometheus.MustRegister(r.requestTotal, r.durationSeconds)
}

func (r *DefaultReporter) Report(ctx context.Context, lvl Level, event string, labels ...KV) {
	r.requestTotal.WithLabelValues(getLabelValues(r.totalLabelNames, labels...)...).Inc()
}

func (r *DefaultReporter) ReportDuration(ctx context.Context, duration time.Duration, labels ...KV) {
	r.durationSeconds.WithLabelValues(getLabelValues(r.secondsLabelNames, labels...)...).Observe(duration.Seconds())
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
