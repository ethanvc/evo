package evolog

func init() {
	initTraceIdSeed()
	globalLogContext = &LogContext{
		traceId: NewTraceId(),
		method:  "Global",
	}
}
