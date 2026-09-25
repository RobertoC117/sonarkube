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
			name: "todos los replies exitosos",
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
			name: "todos los replies fallidos",
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
			name: "mezcla de exitosos y fallidos",
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
			// Caso borde: Sent == 0 ejercita la guarda "if s.Sent > 0" que evita
			// dividir por cero al calcular el porcentaje de perdida.
			name: "sin paquetes enviados",
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
		wantReceived int // usado cuando se espera un valor exacto (ej. 0)
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
			// Un perPacketTimeout de 1ms no le da tiempo a ningun reply de
			// llegar. MakePing no trata esto como error: solo registra cada
			// intento como un PingReply{Success:false} y sigue.
			name:         "per-packet timeout muy corto",
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
				t.Skip("enviar ICMP requiere privilegios de administrador (correr con sudo)")
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
			if tt.name == "per-packet timeout muy corto" && stats.Received != tt.wantReceived {
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
		t.Skip("enviar ICMP requiere privilegios de administrador (correr con sudo)")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
