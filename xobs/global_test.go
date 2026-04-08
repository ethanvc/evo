package xobs

import "time"

func init() {
	sNow = func() time.Time {
		return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	SetGenerateTraceIdFunc(func() string {
		return "1234567890"
	})
	SetGenerateSpanIdFunc(func(rootSpan bool) string {
		return "1234567890"
	})
}
