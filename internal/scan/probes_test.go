package scan

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RobertoC117/sonarkube/internal/probe"
)

func TestWithTimeout_AbortsSlowProbe(t *testing.T) {
	// Fake probe that hangs until the context is cancelled.
	slow := func(ctx context.Context, target string) (probe.ConnectionResult, error) {
		<-ctx.Done()
		return probe.ConnectionResult{Target: target, Err: ctx.Err()}, ctx.Err()
	}

	wrapped := withTimeout(slow, 20*time.Millisecond)

	start := time.Now()
	_, err := wrapped(context.Background(), "1.1.1.1")
	elapsed := time.Since(start)

	if elapsed > 200*time.Millisecond {
		t.Errorf("did not abort in time: took %v", elapsed)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

func TestWithTimeout_DoesNotAbortFastProbe(t *testing.T) {
	fast := func(ctx context.Context, target string) (probe.ConnectionResult, error) {
		return probe.ConnectionResult{Target: target, Success: true}, nil
	}

	wrapped := withTimeout(fast, 50*time.Millisecond)

	got, err := wrapped(context.Background(), "1.1.1.1")
	if err != nil {
		t.Errorf("did not expect an error: %v", err)
	}
	if !got.Success {
		t.Error("esperaba Success=true")
	}
}

func TestSummarizePing(t *testing.T) {
	cases := []struct {
		name        string
		stats       probe.PingStats
		err         error
		wantSuccess bool
		wantErr     bool
	}{
		{
			name: "all reply",
			stats: probe.PingStats{
				Target:   "1.1.1.1",
				Sent:     3,
				Received: 3,
				Replies: []probe.PingReply{
					{Seq: 1, Success: true, Latency: 10 * time.Millisecond},
					{Seq: 2, Success: true, Latency: 20 * time.Millisecond},
					{Seq: 3, Success: true, Latency: 30 * time.Millisecond},
				},
			},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name: "partial loss (1 of 3)",
			stats: probe.PingStats{
				Target:   "1.1.1.1",
				Sent:     3,
				Received: 1,
				Replies: []probe.PingReply{
					{Seq: 1, Success: true, Latency: 10 * time.Millisecond},
					{Seq: 2, Success: false},
					{Seq: 3, Success: false},
				},
			},
			wantSuccess: true, // at least one answered -> alive
			wantErr:     false,
		},
		{
			name: "total loss",
			stats: probe.PingStats{
				Target:   "1.1.1.1",
				Sent:     3,
				Received: 0,
				Replies: []probe.PingReply{
					{Seq: 1, Success: false},
					{Seq: 2, Success: false},
					{Seq: 3, Success: false},
				},
			},
			wantSuccess: false,
			wantErr:     true,
		},
		{
			name:        "fatal error (e.g. missing sudo)",
			stats:       probe.PingStats{Target: "1.1.1.1"},
			err:         errors.New("requires sudo"),
			wantSuccess: false,
			wantErr:     true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizePing(tt.stats, tt.err)

			if got.Success != tt.wantSuccess {
				t.Errorf("Success: expected %v, got %v", tt.wantSuccess, got.Success)
			}
			if (got.Err != nil) != tt.wantErr {
				t.Errorf("Err: expected error=%v, got %v", tt.wantErr, got.Err)
			}
			if got.Target != tt.stats.Target {
				t.Errorf("Target: expected %v, got %v", tt.stats.Target, got.Target)
			}
		})
	}
}

func TestSummarizePing_AverageLatency(t *testing.T) {
	stats := probe.PingStats{
		Target:   "1.1.1.1",
		Sent:     2,
		Received: 2,
		Replies: []probe.PingReply{
			{Seq: 1, Success: true, Latency: 10 * time.Millisecond},
			{Seq: 2, Success: true, Latency: 20 * time.Millisecond},
		},
	}

	got := summarizePing(stats, nil)

	if got.LatencyMs != 15 { // average of 10ms and 20ms
		t.Errorf("average latency: expected 15ms, got %vms", got.LatencyMs)
	}
}
