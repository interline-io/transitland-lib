package jobs

import (
	"encoding/json"
	"errors"
)

///////////

type JobMapper struct {
	jobFns map[string]JobFn
}

func NewJobMapper() *JobMapper {
	return &JobMapper{jobFns: map[string]JobFn{}}
}

func (j *JobMapper) AddJobType(jobFn JobFn) error {
	jw := jobFn()
	j.jobFns[jw.Kind()] = jobFn
	return nil
}

func (j *JobMapper) GetRunner(jobType string, jobArgs JobArgs) (JobWorker, error) {
	jobFn, ok := j.jobFns[jobType]
	if !ok {
		return nil, errors.New("unknown job type")
	}
	runner := jobFn()
	jw, err := json.Marshal(jobArgs)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(jw, runner); err != nil {
		return nil, err
	}
	return runner, nil
}
