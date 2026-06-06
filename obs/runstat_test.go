package obs

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnableRunStat_runtimeMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	EnableRunStat(reg)

	mfs := mustGather(t, reg)
	got := gaugeValues(mfs, "evo_gauge_info")
	require.Contains(t, got, "memory_limit")
	require.Contains(t, got, "memory_current")
	require.Contains(t, got, "cpu_core")
	assert.Greater(t, got["memory_limit"], float64(0))
	assert.Greater(t, got["memory_current"], float64(0))
	assert.Greater(t, got["cpu_core"], float64(0))
	assert.LessOrEqual(t, got["memory_current"], got["memory_limit"])

	counter := counterValue(mfs, "evo_cpu_usage_seconds_total")
	require.NotNil(t, counter)
	assert.GreaterOrEqual(t, *counter, float64(0))
}

func gaugeValues(mfs []*dto.MetricFamily, name string) map[string]float64 {
	out := make(map[string]float64)
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			out[labelValue(m, "key")] = m.GetGauge().GetValue()
		}
	}
	return out
}

func counterValue(mfs []*dto.MetricFamily, name string) *float64 {
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			v := m.GetCounter().GetValue()
			return &v
		}
	}
	return nil
}

func mustGather(t *testing.T, reg *prometheus.Registry) []*dto.MetricFamily {
	t.Helper()
	mfs, err := reg.Gather()
	require.NoError(t, err)
	return mfs
}

func labelValue(m *dto.Metric, name string) string {
	for _, lp := range m.GetLabel() {
		if lp.GetName() == name {
			return lp.GetValue()
		}
	}
	return ""
}
