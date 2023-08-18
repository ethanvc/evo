package evolog

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

type Reporter struct {
	svr                          string
	instance                     string
	serverRequestTotal           *prometheus.CounterVec
	clientRequestTotal           *prometheus.CounterVec
	serverRequestDurationSeconds *prometheus.HistogramVec
	clientRequestDurationSeconds *prometheus.HistogramVec
	register                     prometheus.Registerer
}

func NewReporter(conf *ReporterConfig) *Reporter {
	r := &Reporter{
		svr:      conf.Svr,
		instance: conf.Instance,
	}
	r.init()
	return r
}

func (r *Reporter) init() {
	r.register = prometheus.NewRegistry()
	constLabels := prometheus.Labels{
		"svr":  r.svr,
		"inst": r.instance,
	}
	r.serverRequestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        "evo_server_request_total",
		ConstLabels: constLabels,
	}, []string{"method", "event"})
	r.register.MustRegister(r.serverRequestTotal)

	r.clientRequestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        "evo_client_request_total",
		ConstLabels: prometheus.Labels{"inst": r.instance},
	}, []string{"from_svr", "svr", "method", "event"})
	r.register.MustRegister(r.serverRequestTotal)

	r.serverRequestDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "evo_server_request_duration_seconds",
		ConstLabels: constLabels,
	}, []string{"method"})
	r.register.MustRegister(r.serverRequestDurationSeconds)

	r.clientRequestDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "evo_client_request_duration_seconds",
		ConstLabels: prometheus.Labels{"inst": r.instance},
	}, []string{"from_svr", "svr", "method"})
	r.register.MustRegister(r.clientRequestDurationSeconds)
}

func (r *Reporter) ReportServerRequest(c context.Context, event string) {
	lc := GetLogContext(c)
	r.serverRequestTotal.WithLabelValues(lc.GetMethod(), event).Inc()
}

func (r *Reporter) ReportClientRequest(c context.Context, svr, method, event string) {
	r.clientRequestTotal.WithLabelValues(r.svr, svr, method, event).Inc()
}

func (r *Reporter) ReportServerDuration(c context.Context, duration time.Duration) {
	lc := GetLogContext(c)
	r.serverRequestDurationSeconds.WithLabelValues(lc.method).Observe(duration.Seconds())
}

func (r *Reporter) ReportClientDuration(c context.Context, svr, method string, duration time.Duration) {
	r.clientRequestDurationSeconds.WithLabelValues(r.svr, svr, method).Observe(duration.Seconds())
}

func ReportServerRequest(c context.Context, event string) {
	DefaultReporter().ReportServerRequest(c, event)
}

type ReporterConfig struct {
	Svr      string
	Instance string
}
