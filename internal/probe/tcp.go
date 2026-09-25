package probe

import (
	"context"
	"log"
	"net"
	"time"
)

func EstablishTCPConnection(ctx context.Context, address string) (ConnectionResult, error) {
	start := time.Now()
	// conn, err := net.DialTimeout("tcp", address, time.Duration(timeout_ms)*time.Millisecond)
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		elapsed := time.Since(start)
		return ConnectionResult{
			Target:    address,
			Success:   false,
			Err:       err,
			ErrMsg:    err.Error(),
			Latency:   elapsed,
			LatencyMs: elapsed.Milliseconds(),
		}, err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("error closing connection: %v", err)
		}
	}()
	elapsed := time.Since(start)
	return ConnectionResult{
		Target:    address,
		Success:   true,
		Err:       nil,
		Latency:   elapsed,
		LatencyMs: elapsed.Milliseconds(),
	}, nil
}
