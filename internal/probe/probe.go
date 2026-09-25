package probe

import (
	"fmt"
	"time"
)

type ConnectionResult struct {
	Target    string        `json:"target"`
	Success   bool          `json:"success"`
	Err       error         `json:"-"`
	ErrMsg    string        `json:"error,omitempty"`
	Latency   time.Duration `json:"-"`
	LatencyMs int64         `json:"latency_ms"`
	Detail    string        `json:"detail,omitempty"`
}

func (r ConnectionResult) RenderTable() (string, error) {
	if !r.Success {
		return fmt.Sprintf("Target: %s\nSuccess: false\nError: %v\n", r.Target, r.Err), nil
	}
	s := fmt.Sprintf("Target: %s\nSuccess: true\nLatency in ms: %d\n", r.Target, r.Latency.Milliseconds())
	if r.Detail != "" {
		s += fmt.Sprintf("Detail: %s\n", r.Detail)
	}
	return s, nil
}
