package plog

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewReporter(t *testing.T) {
	r := NewReporter()
	c := WithLogContext(nil, &LogContextConfig{
		Method: "Test",
	})
	r.ReportEvent(c, "Test")
	metricsChan := make(chan prometheus.Metric, 50)
	r.reg.Collect(metricsChan)
	metrics := consumeChanALl(metricsChan)
	require.Equal(t, 1, len(metrics))
	families, err := r.reg.Gather()
	require.NoError(t, err)
	require.Equal(t, 1, len(families))
}

func consumeChanALl[T any](ch chan T) []T {
	var result []T
	for {
		select {
		case element := <-ch:
			result = append(result, element)
		default:
			return result
		}
	}
}
