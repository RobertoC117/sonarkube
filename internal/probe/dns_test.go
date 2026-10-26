package probe

import (
	"context"
	"testing"
	"time"
)

func TestMakeDNSQuery(t *testing.T) {

	tests := []struct {
		name    string
		target  string
		timeout int64
		wantErr bool
	}{
		{
			name:    "Success case",
			target:  "google.com",
			timeout: 5000,
			wantErr: false,
		},
		{
			name:    "No real domain",
			target:  "nonexistent.domain.local",
			timeout: 5000,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(tt.timeout)*time.Millisecond)
			defer cancel()
			result, err := ResolveDNS(ctx, tt.target)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("did not expect an error but got: %v", err)
				}
				if result.Target != tt.target {
					t.Fatalf("expected target %s but got %s", tt.target, result.Target)
				}
				if result.Success != !tt.wantErr {
					t.Fatalf("expected success %v but got %v", !tt.wantErr, result.Success)
				}
			}
		})
	}
}

// Deterministically verifies ResolveDNS aborts when the context is cancelled.
// Uses an already-cancelled context instead of a tiny timeout (like "1ms"),
// which depended on network timing and resolver cache and failed intermittently.
func TestResolveDNS_RespectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled before the call: result does not depend on the network

	_, err := ResolveDNS(ctx, "google.com")
	if err == nil {
		t.Fatal("expected an error from the cancelled context, got nil")
	}
}
