package validator

import (
	"github.com/interline-io/transitland-lib/copier"
)

// Result contains a validation report result,
type Result struct {
	copier.Result        // add to copier result:
	Success       bool   `json:"success"`
	FailureReason string `json:"failure_reason"`
}
