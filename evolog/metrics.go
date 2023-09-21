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
	register                   prometheus.Registerer
}

func NewReporter(conf *ReporterConfig) *Reporter {
	r := &Reporter{
		config: conf,
	}
	r.init()
	return r
}

func (r *Reporter) init() {
	r.register = prometheus.NewRegistry()
	globalLabels := prometheus.Labels{
		"report_svr":  r.config.ReportSvr,
		"report_inst": r.config.ReportInst,
	}
	for k, v := range r.config.GlobalLabels {
		globalLabels[k] = v
	}
	r.serverEventTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        "evo_server_event_total",
		ConstLabels: globalLabels,
	}, []string{"method", "event"})
	r.register.MustRegister(r.serverEventTotal)

	r.clientEventTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        "evo_client_event_total",
		ConstLabels: globalLabels,
	}, []string{"svr", "method", "event"})
	r.register.MustRegister(r.clientEventTotal)

	r.serverEventDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "evo_server_event_duration_seconds",
		ConstLabels: globalLabels,
	}, []string{"method"})
	r.register.MustRegister(r.serverEventDurationSeconds)

	r.clientEventDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "evo_client_event_duration_seconds",
		ConstLabels: globalLabels,
	}, []string{"svr", "method"})
	r.register.MustRegister(r.clientEventDurationSeconds)
}

func (r *Reporter) ReportEvent(c context.Context, event string) {
	lc := GetLogContext(c)
	r.serverEventTotal.WithLabelValues(lc.GetMethod(), event).Inc()
}

func (r *Reporter) ReportErrEvent(c context.Context, event string) {
	r.ReportEvent(c, "ERR:"+event)
}

func (r *Reporter) ReportRequestEvent(c context.Context, code codes.Code, event string) {
	r.ReportEvent(c, fmt.Sprintf("%s:%s", code.String(), event))
}

func (r *Reporter) ReportRequest(c context.Context, event string) {
	r.ReportEvent(c, "REQ:"+event)
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

func ReportServerRequest(c context.Context, event string) {
	DefaultReporter().ReportEvent(c, event)
}

type ReporterConfig struct {
	ReportSvr    string
	ReportInst   string
	GlobalLabels prometheus.Labels
}
