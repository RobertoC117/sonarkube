package probe

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestPingStatsRenderTable(t *testing.T) {
	tests := []struct {
		name  string
		stats PingStats
		want  string
	}{
		{
			name: "all replies succeed",
			stats: PingStats{
				Target:   "192.168.1.1",
				Sent:     2,
				Received: 2,
				Replies: []PingReply{
					{Seq: 1, Success: true, LatencyMs: 12},
					{Seq: 2, Success: true, LatencyMs: 15},
				},
			},
			want: "Reply from 192.168.1.1: seq=1 time=12ms\n" +
				"Reply from 192.168.1.1: seq=2 time=15ms\n" +
				"\n--- 192.168.1.1 ping statistics ---\n" +
				"2 packets transmitted, 2 received, 0.0% packet loss\n",
		},
		{
			name: "all replies fail",
			stats: PingStats{
				Target:   "10.0.0.99",
				Sent:     2,
				Received: 0,
				Replies: []PingReply{
					{Seq: 1, Success: false, Err: errors.New("i/o timeout")},
					{Seq: 2, Success: false, Err: errors.New("i/o timeout")},
				},
			},
			want: "Request seq=1 failed: i/o timeout\n" +
				"Request seq=2 failed: i/o timeout\n" +
				"\n--- 10.0.0.99 ping statistics ---\n" +
				"2 packets transmitted, 0 received, 100.0% packet loss\n",
		},
		{
			name: "mix of successes and failures",
			stats: PingStats{
				Target:   "example.com",
				Sent:     4,
				Received: 1,
				Replies: []PingReply{
					{Seq: 1, Success: true, LatencyMs: 5},
					{Seq: 2, Success: false, Err: errors.New("i/o timeout")},
					{Seq: 3, Success: false, Err: errors.New("i/o timeout")},
					{Seq: 4, Success: false, Err: errors.New("i/o timeout")},
				},
			},
			want: "Reply from example.com: seq=1 time=5ms\n" +
				"Request seq=2 failed: i/o timeout\n" +
				"Request seq=3 failed: i/o timeout\n" +
				"Request seq=4 failed: i/o timeout\n" +
				"\n--- example.com ping statistics ---\n" +
				"4 packets transmitted, 1 received, 75.0% packet loss\n",
		},
		{
			// Edge case: Sent == 0 exercises the "if s.Sent > 0" guard that
			// avoids dividing by zero when computing loss percentage.
			name: "no packets sent",
			stats: PingStats{
				Target:   "host",
				Sent:     0,
				Received: 0,
				Replies:  nil,
			},
			want: "\n--- host ping statistics ---\n" +
				"0 packets transmitted, 0 received, 0.0% packet loss\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.stats.RenderTable()
			if err != nil {
				t.Errorf("RenderTable() error =\n%q\nwant:\n%q", err, tt.want)
			}
			if got != tt.want {
				t.Errorf("RenderTable() =\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestMakePing(t *testing.T) {
	tests := []struct {
		name         string
		target       string
		count        int
		timeout      int64
		wantErr      bool
		minReceived  int
		wantReceived int // used when an exact value is expected (e.g. 0)
	}{
		{
			name:        "success case",
			target:      "google.com",
			count:       2,
			timeout:     2000,
			wantErr:     false,
			minReceived: 1,
		},
		{
			name:    "no real domain",
			target:  "nonexistent.domain.local",
			count:   1,
			timeout: 2000,
			wantErr: true,
		},
		{
			// A 1ms perPacketTimeout gives no reply time to arrive. MakePing
			// does not treat this as an error: it records each attempt as a
			// PingReply{Success:false} and continues.
			name:         "per-packet timeout too short",
			target:       "google.com",
			count:        1,
			timeout:      1,
			wantErr:      false,
			wantReceived: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			stats, err := MakePing(ctx, tt.target, tt.count, time.Duration(tt.timeout)*time.Millisecond)

			if errors.Is(err, os.ErrPermission) {
				t.Skip("sending ICMP requires administrator privileges (run with sudo)")
			}

			if (err != nil) != tt.wantErr {
				t.Fatalf("expected err=%v, got %v", tt.wantErr, err)
			}
			if tt.wantErr {
				return
			}

			if tt.minReceived > 0 && stats.Received < tt.minReceived {
				t.Errorf("expected at least %d received replies, got %d", tt.minReceived, stats.Received)
			}
			if tt.name == "per-packet timeout too short" && stats.Received != tt.wantReceived {
				t.Errorf("expected %d received replies, got %d", tt.wantReceived, stats.Received)
			}
		})
	}
}

func TestMakePing_ContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := MakePing(ctx, "google.com", 1, time.Second)

	if errors.Is(err, os.ErrPermission) {
		t.Skip("sending ICMP requires administrator privileges (run with sudo)")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
