package evolog

import (
	"testing"

	"golang.org/x/exp/slog"
)

func Test_GlobalLog(t *testing.T) {
	slog.Info("hello")
}
