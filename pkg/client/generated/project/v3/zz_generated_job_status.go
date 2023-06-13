package client

const (
	JobStatusType                         = "jobStatus"
	JobStatusFieldActive                  = "active"
	JobStatusFieldCompletedIndexes        = "completedIndexes"
	JobStatusFieldCompletionTime          = "completionTime"
	JobStatusFieldConditions              = "conditions"
	JobStatusFieldFailed                  = "failed"
	JobStatusFieldReady                   = "ready"
	JobStatusFieldStartTime               = "startTime"
	JobStatusFieldSucceeded               = "succeeded"
	JobStatusFieldUncountedTerminatedPods = "uncountedTerminatedPods"
)

type JobStatus struct {
	Active                  int64                    `json:"active,omitempty" yaml:"active,omitempty"`
	CompletedIndexes        string                   `json:"completedIndexes,omitempty" yaml:"completedIndexes,omitempty"`
	CompletionTime          string                   `json:"completionTime,omitempty" yaml:"completionTime,omitempty"`
	Conditions              []JobCondition           `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	Failed                  int64                    `json:"failed,omitempty" yaml:"failed,omitempty"`
	Ready                   *int64                   `json:"ready,omitempty" yaml:"ready,omitempty"`
	StartTime               string                   `json:"startTime,omitempty" yaml:"startTime,omitempty"`
	Succeeded               int64                    `json:"succeeded,omitempty" yaml:"succeeded,omitempty"`
	UncountedTerminatedPods *UncountedTerminatedPods `json:"uncountedTerminatedPods,omitempty" yaml:"uncountedTerminatedPods,omitempty"`
}
