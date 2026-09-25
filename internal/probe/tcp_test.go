package probe

import (
	"context"
	"testing"
	"time"
)

func TestEstablishTCPConnection(t *testing.T) {
	tests := []struct {
		name        string
		address     string
		timeout     int64
		wantSuccess bool
		wantErr     bool
	}{
		{
			name:        "Success case",
			address:     "google.com:80",
			timeout:     5000,
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name:        "No real address",
			address:     "nonexistent.domain.local:80",
			timeout:     5000,
			wantSuccess: false,
			wantErr:     true,
		},
		{
			name:        "Timeout case",
			address:     "google.com:80",
			timeout:     1,
			wantSuccess: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(tt.timeout)*time.Millisecond)
			defer cancel()
			result, err := EstablishTCPConnection(ctx, tt.address)
			if result.Success != tt.wantSuccess {
				t.Errorf("expected success %v but got %v", tt.wantSuccess, result.Success)
			}
			if err != nil != tt.wantErr {
				t.Errorf("expected error %v but got %v", tt.wantErr, err != nil)
			}
		})
	}
}
