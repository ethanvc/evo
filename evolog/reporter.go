package evolog

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
	"time"
)

type Reporter struct {
	config                     *ReporterConfig
	serverEventTotal           *prometheus.CounterVec
	clientEventTotal           *prometheus.CounterVec
	serverEventDurationSeconds *prometheus.HistogramVec
	clientEventDurationSeconds *prometheus.HistogramVec
	gauge                      *prometheus.GaugeVec
}

type ReporterConfig struct {
	ReportSvr    string
	ReportInst   string
	Component    string
	MetricPrefix string
	GlobalLabels prometheus.Labels
}

func NewReporter(conf *ReporterConfig) *Reporter {
	if conf.MetricPrefix == "" {
		conf.MetricPrefix = "evo"
	}
	r := &Reporter{
		config: conf,
	}
	r.init()
	return r
}

func (r *Reporter) init() {
	reg := prometheus.DefaultRegisterer
	globalLabels := prometheus.Labels{
		"report_svr":  r.config.ReportSvr,
		"report_inst": r.config.ReportInst,
		"component":   r.config.Component,
	}
	for k, v := range r.config.GlobalLabels {
		globalLabels[k] = v
	}
	r.serverEventTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        r.metricName("server_event_total"),
		ConstLabels: globalLabels,
	}, []string{"svr", "method", "event"})
	reg.MustRegister(r.serverEventTotal)

	r.clientEventTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        r.metricName("client_event_total"),
		ConstLabels: globalLabels,
	}, []string{"svr", "method", "event"})
	reg.MustRegister(r.clientEventTotal)

	r.serverEventDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        r.metricName("server_event_duration_seconds"),
		ConstLabels: globalLabels,
	}, []string{"method"})
	reg.MustRegister(r.serverEventDurationSeconds)

	r.clientEventDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        r.metricName("client_event_duration_seconds"),
		ConstLabels: globalLabels,
	}, []string{"svr", "method"})
	reg.MustRegister(r.clientEventDurationSeconds)

	r.gauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: r.metricName("xx"),
	}, []string{"type", "id"})
	reg.MustRegister(r.gauge)
}

func (r *Reporter) Clean() {
	reg := prometheus.DefaultRegisterer
	reg.Unregister(r.clientEventTotal)
	reg.Unregister(r.clientEventDurationSeconds)
	reg.Unregister(r.serverEventTotal)
	reg.Unregister(r.serverEventDurationSeconds)
}

func (r *Reporter) metricName(name string) string {
	return fmt.Sprintf("%s_%s", r.config.MetricPrefix, name)
}

func (r *Reporter) ReportEvent(c context.Context, event string) {
	lc := GetLogContext(c)
	r.serverEventTotal.WithLabelValues(r.config.ReportSvr, lc.GetMethod(), event).Inc()
}

func (r *Reporter) ReportErrEvent(c context.Context, event string) {
	r.ReportEvent(c, "ERR:"+event)
}

func (r *Reporter) ReportRequest(c context.Context, code codes.Code, event string) {
	r.ReportEvent(c, fmt.Sprintf("REQ:%s:%s", code.String(), event))
}

func (r *Reporter) ReportClientEvent(c context.Context, svr, method, event string) {
	r.clientEventTotal.WithLabelValues(svr, method, event).Inc()
}

func (r *Reporter) ReportClientRequest(c context.Context, svr, method, event string) {
	r.ReportClientEvent(c, svr, method, "REQ:"+event)
}

func (r *Reporter) ReportEventDuration(c context.Context, duration time.Duration) {
	lc := GetLogContext(c)
	r.serverEventDurationSeconds.WithLabelValues(lc.method).Observe(duration.Seconds())
}

func (r *Reporter) ReportClientEventDuration(c context.Context, svr, method string, duration time.Duration) {
	r.clientEventDurationSeconds.WithLabelValues(svr, method).Observe(duration.Seconds())
}

func ReportEvent(c context.Context, event string) {
	DefaultReporter().ReportEvent(c, event)
}
