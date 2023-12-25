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
	gauge                      *prometheus.GaugeVec
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

func (r *Reporter) init() {
	reg := r.reg
	r.serverEventTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: r.metricName("server_event_total"),
	}, []string{"svr", "method", "event"})
	reg.MustRegister(r.serverEventTotal)

	r.clientEventTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: r.metricName("client_event_total"),
	}, []string{"svr", "method", "event"})
	reg.MustRegister(r.clientEventTotal)

	r.serverEventDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: r.metricName("server_event_duration_seconds"),
	}, []string{"method"})
	reg.MustRegister(r.serverEventDurationSeconds)

	r.clientEventDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: r.metricName("client_event_duration_seconds"),
	}, []string{"svr", "method"})
	reg.MustRegister(r.clientEventDurationSeconds)

	r.gauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: r.metricName("xx"),
	}, []string{"type", "id"})
	reg.MustRegister(r.gauge)
}

func (r *Reporter) metricName(name string) string {
	return fmt.Sprintf("%s_%s", "evo", name)
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

func (r *Reporter) UpdateConfig(f func(conf *ReporterConfig)) {
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
}

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

func ReportEvent(c context.Context, event string) {
	DefaultReporter().ReportEvent(c, event)
}
