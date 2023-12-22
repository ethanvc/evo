package plog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTraceId(t *testing.T) {
	id := NewTraceId()
	require.Equal(t, 32, len(id))
}

func TestGetLocalIp(t *testing.T) {
	ip := GetLocalIp()
	require.Truef(t, len(ip) > 0, ip)
}
