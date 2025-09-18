package local

import (
	"net/http"

	"github.com/interline-io/transitland-lib/server/metrics"
)

type LocalMetric struct{}

func NewLocalMetric() *LocalMetric {
	return &LocalMetric{}
}

func (m *LocalMetric) NewJobMetric(queue string) metrics.JobMetric {
	return &LocalMetric{}
}

func (m *LocalMetric) NewApiMetric(handlerName string) metrics.ApiMetric {
	return &LocalMetric{}
}

func (m *LocalMetric) MetricsHandler() http.Handler {
	return nil
}

func (m *LocalMetric) AddStartedJob(queueName string, jobType string) {
}

func (m *LocalMetric) AddCompletedJob(queueName string, jobType string, success bool) {
}

func (m *LocalMetric) AddResponse(method string, responseCode int, requestSize int64, responseSize int64, responseTime float64) {
}
