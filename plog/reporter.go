package plog

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/jinzhu/copier"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"google.golang.org/grpc/codes"
	"sync"
	"time"
)

type Reporter struct {
	mux                        sync.Mutex
	reg                        *prometheus.Registry
	config                     *ReporterConfig
	globalLabels               []*dto.LabelPair
	serverEventTotal           *prometheus.CounterVec
	clientEventTotal           *prometheus.CounterVec
	serverEventDurationSeconds *prometheus.HistogramVec
	clientEventDurationSeconds *prometheus.HistogramVec
}

type ReporterConfig struct {
	ReportSvr    string
	ReportInst   string
	Component    string
	GlobalLabels prometheus.Labels
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
	}, []string{"lvl", "method", "event"})
	reg.MustRegister(r.serverEventTotal)

	r.clientEventTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: r.metricName("client_event_total"),
	}, []string{"lvl", "svr", "method", "event"})
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

func (r *Reporter) metricName(name string) string {
	return fmt.Sprintf("%s_%s", "evo", name)
}

func (r *Reporter) ReportEvent(c context.Context, event string) {
	r.ReportEventWithLevel(c, MonitorLevelInfo, event)
}

func (r *Reporter) ReportErrEvent(c context.Context, event string) {
	r.ReportEventWithLevel(c, MonitorLevelErr, event)
}

func (r *Reporter) ReportRequest(c context.Context, code codes.Code, event string, extra ...string) {
	lvl := GetMonitorLevel(code)
	r.ReportEventWithLevel(c, lvl, "REQ_END:"+code.String()+":"+event, extra...)
}

func (r *Reporter) ReportEventWithLevel(
	c context.Context, lvl MonitorLevel, event string, extra ...string) {
	lc := GetLogContext(c)
	lvs := append(extra, lvl.String(), lc.GetMethod(), event)
	r.serverEventTotal.WithLabelValues(lvs...).Inc()
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
	lvs := append(extra, lvl.String(), lc.GetMethod(), event)
	r.clientEventTotal.WithLabelValues(lvs...).Inc()
}

func (r *Reporter) ReportClientDuration(c context.Context, svr, method string, duration time.Duration) {
	r.clientEventDurationSeconds.WithLabelValues(svr, method).Observe(duration.Seconds())
}

func (r *Reporter) UpdateConfig(f func(conf *ReporterConfig)) error {
	r.mux.Lock()
	defer r.mux.Lock()
	var newConfig ReporterConfig
	copier.Copy(&newConfig, r.config)
	f(&newConfig)
	var globalLabels []*dto.LabelPair
	globalLabels = append(globalLabels, &dto.LabelPair{
		Name:  proto.String("report_inst"),
		Value: proto.String(newConfig.ReportInst),
	})
	globalLabels = append(globalLabels, &dto.LabelPair{
		Name:  proto.String("report_svr"),
		Value: proto.String(newConfig.ReportSvr),
	})
	globalLabels = append(globalLabels, &dto.LabelPair{
		Name:  proto.String("component"),
		Value: proto.String(newConfig.Component),
	})
	for k, v := range newConfig.GlobalLabels {
		globalLabels = append(globalLabels, &dto.LabelPair{
			Name:  &k,
			Value: &v,
		})
	}
	r.config = &newConfig
	r.globalLabels = globalLabels
	return nil
}

// https://prometheus.io/docs/instrumenting/writing_clientlibs/
// https://prometheus.io/docs/instrumenting/exposition_formats/

func (r *Reporter) Gather() ([]*dto.MetricFamily, error) {
	globalLabels := r.globalLabels
	families, err := r.reg.Gather()
	for _, family := range families {
		for _, metric := range family.Metric {
			metric.Label = append(metric.Label, globalLabels...)
		}
	}
	return families, err
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
