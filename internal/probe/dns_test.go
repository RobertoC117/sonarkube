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

// TestResolveDNS_RespetaCancelacion verifica, de forma determinista, que
// ResolveDNS aborta cuando el contexto está cancelado. Usamos un contexto YA
// cancelado en vez de un timeout chico (como "1ms"), porque ese enfoque dependía
// del tiempo de red y del caché del resolver: si la respuesta llegaba rápido, no
// había error y el test fallaba de forma intermitente.
func TestResolveDNS_RespetaCancelacion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelado antes de llamar: el resultado no depende de la red

	_, err := ResolveDNS(ctx, "google.com")
	if err == nil {
		t.Fatal("esperaba un error por contexto cancelado, pero obtuve nil")
	}
}
