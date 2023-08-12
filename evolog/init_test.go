package evolog

import (
	"testing"

	"log/slog"
)

func Test_GlobalLog(t *testing.T) {
	slog.Info("hello")
}
