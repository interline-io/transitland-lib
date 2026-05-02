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

func (j *JobMapper) Register(jobFn JobFn) error {
	jw := jobFn()
	j.jobFns[jw.Kind()] = jobFn
	return nil
}

func (j *JobMapper) GetRunner(kind string, args JobArgs) (JobWorker, error) {
	jobFn, ok := j.jobFns[kind]
	if !ok {
		return nil, errors.New("unknown job kind")
	}
	runner := jobFn()
	jw, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(jw, runner); err != nil {
		return nil, err
	}
	return runner, nil
}
