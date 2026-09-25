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
		{
			name:    "Timeout case",
			target:  "google.com",
			timeout: 1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(tt.timeout)*time.Millisecond)
			defer cancel()
			result, err := ResolveDNS(ctx, tt.target)

			// fmt.Printf("Result: %v, Error: %v\n", result, err)
			// isTimeout := os.IsTimeout(err)
			// fmt.Printf("isTimeout: %v\n", isTimeout)

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
