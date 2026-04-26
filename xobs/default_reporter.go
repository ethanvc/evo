

type DefaultReporter struct {
}

func (r *DefaultReporter) Report(ctx context.Context, lvl Level, event string, labels ...KV) {

}

func (r *DefaultReporter) ReportDuration(ctx context.Context, duration time.Duration, labels ...KV) {

}