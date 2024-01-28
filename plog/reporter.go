package plog

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"google.golang.org/grpc/codes"
	"time"
)

type Reporter struct {
	reg                        *prometheus.Registry
	config                     *ReporterConfig
	serverEventTotal           *prometheus.CounterVec
	clientEventTotal           *prometheus.CounterVec
	serverEventDurationSeconds *prometheus.HistogramVec
	clientEventDurationSeconds *prometheus.HistogramVec
}

type ReporterConfig struct {
	ReportSvr      string
	ReportInst     string
	Component      string
	GlobalLabels   prometheus.Labels
	ExtraLabels    []string
	ExtraExtractor func(c context.Context) []string
}

func NewReporter() *Reporter {
	r := &Reporter{
		reg:    prometheus.NewRegistry(),
		config: new(ReporterConfig),
	}
	r.init()
	return r
}

type MonitorLevel string

const (
	MonitorLevelErr  MonitorLevel = "err"
	MonitorLevelWar  MonitorLevel = "warn"
	MonitorLevelInfo MonitorLevel = "info"
)

func (lvl MonitorLevel) String() string {
	return string(lvl)
}

func (r *Reporter) init() {
	reg := r.reg
	r.serverEventTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: r.metricName("server_event_total"),
	}, r.requestLabels("lvl", "method", "event"))
	reg.MustRegister(r.serverEventTotal)

	r.clientEventTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: r.metricName("client_event_total"),
	}, r.requestLabels("lvl", "svr", "method", "event"))
	reg.MustRegister(r.clientEventTotal)

	r.serverEventDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: r.metricName("server_event_duration_seconds"),
	}, []string{"method"})
	reg.MustRegister(r.serverEventDurationSeconds)

	r.clientEventDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: r.metricName("client_event_duration_seconds"),
	}, []string{"svr", "method"})
	reg.MustRegister(r.clientEventDurationSeconds)
}

func (r *Reporter) requestLabels(labels ...string) []string {
	result := append([]string{}, r.config.ExtraLabels...)
	return append(result, labels...)
}

func (r *Reporter) metricName(name string) string {
	return fmt.Sprintf("%s_%s", "evo", name)
}

func (r *Reporter) ReportEvent(c context.Context, event string, extra ...string) {
	r.ReportEventWithLevel(c, MonitorLevelInfo, event, extra...)
}

func (r *Reporter) ReportErrEvent(c context.Context, event string, extra ...string) {
	r.ReportEventWithLevel(c, MonitorLevelErr, event, extra...)
}

func (r *Reporter) ReportRequest(c context.Context, code codes.Code, event string, extra ...string) {
	lvl := GetMonitorLevel(code)
	r.ReportEventWithLevel(c, lvl, "REQ_END:"+code.String()+":"+event, extra...)
}

func (r *Reporter) ReportEventWithLevel(
	c context.Context, lvl MonitorLevel, event string, extra ...string) {
	lc := GetLogContext(c)
	extra = r.extractExtra(c, extra...)
	lvs := append(extra, lvl.String(), lc.GetMethod(), event)
	r.serverEventTotal.WithLabelValues(lvs...).Inc()
}

func (r *Reporter) extractExtra(c context.Context, extra ...string) []string {
	if len(extra) == len(r.config.ExtraLabels) {
		return extra
	}
	if r.config.ExtraExtractor != nil {
		extra = r.config.ExtraExtractor(c)
	}
	if len(extra) == len(r.config.ExtraLabels) {
		return extra
	}

	extra = []string{}
	for i := 0; i < len(r.config.ExtraLabels); i++ {
		extra = append(extra, "")
	}
	return extra
}

func (r *Reporter) ReportDuration(c context.Context, duration time.Duration) {
	lc := GetLogContext(c)
	r.serverEventDurationSeconds.WithLabelValues(lc.method).Observe(duration.Seconds())
}

func (r *Reporter) ReportClientRequest(c context.Context, code codes.Code, svr, event string, extra ...string) {
	lvl := GetMonitorLevel(code)
	r.ReportClientEventWithLevel(c, lvl, svr, "REQ_END:"+code.String()+":"+event, extra...)
}

func (r *Reporter) ReportClientEventWithLevel(c context.Context,
	lvl MonitorLevel, svr, event string, extra ...string) {
	lc := GetLogContext(c)
	extra = r.extractExtra(c, extra...)
	lvs := append(extra, lvl.String(), svr, lc.GetMethod(), event)
	r.clientEventTotal.WithLabelValues(lvs...).Inc()
}

func (r *Reporter) ReportClientDuration(c context.Context, svr, method string, duration time.Duration) {
	r.clientEventDurationSeconds.WithLabelValues(svr, method).Observe(duration.Seconds())
}

// https://prometheus.io/docs/instrumenting/writing_clientlibs/
// https://prometheus.io/docs/instrumenting/exposition_formats/

func (r *Reporter) Gather() ([]*dto.MetricFamily, error) {
	return r.reg.Gather()
}

func GetMonitorLevel(code codes.Code) MonitorLevel {
	switch code {
	case codes.OK, codes.Canceled, codes.InvalidArgument,
		codes.NotFound, codes.AlreadyExists, codes.PermissionDenied,
		codes.FailedPrecondition, codes.OutOfRange, codes.Unauthenticated:
		return MonitorLevelInfo
	default:
		return MonitorLevelErr
	}
}
