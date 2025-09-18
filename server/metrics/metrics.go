package metrics

import "net/http"

type ApiMetric interface {
	AddResponse(method string, responseCode int, requestSize int64, responseSize int64, responseTime float64)
}

type JobMetric interface {
	AddStartedJob(string, string)
	AddCompletedJob(string, string, bool)
}

type MetricProvider interface {
	NewApiMetric(handlerName string) ApiMetric
	NewJobMetric(queue string) JobMetric
	MetricsHandler() http.Handler
}

type Config struct {
	EnableMetrics   bool
	MetricsProvider string
}
