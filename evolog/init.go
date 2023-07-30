package evolog

func init() {
	initTraceIdSeed()
	globalLogContext = &LogContext{
		TraceId: NewTraceId(),
		Method:  "Global",
	}
}
