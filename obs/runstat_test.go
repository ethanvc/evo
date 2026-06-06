package obs

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnableRunStat_memoryGauges(t *testing.T) {
	reg := prometheus.NewRegistry()
	EnableRunStat(reg)

	got := gaugeValues(mustGather(t, reg), "evo_gauge_info")
	require.Contains(t, got, "memory_limit")
	require.Contains(t, got, "memory_current")
	assert.Greater(t, got["memory_limit"], float64(0))
	assert.Greater(t, got["memory_current"], float64(0))
	assert.LessOrEqual(t, got["memory_current"], got["memory_limit"])
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
