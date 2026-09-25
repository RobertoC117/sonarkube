package probe

import (
	"context"
	"net"
	"strings"
	"time"
)

func ResolveDNS(ctx context.Context, name string) (ConnectionResult, error) {
	start := time.Now()

	ips, err := net.DefaultResolver.LookupHost(ctx, name)
	elapsed := time.Since(start)
	if err != nil {
		return ConnectionResult{
			Target:    name,
			Success:   false,
			Err:       err,
			ErrMsg:    err.Error(),
			Latency:   elapsed,
			LatencyMs: elapsed.Milliseconds(),
		}, err
	}

	return ConnectionResult{
		Target:    name,
		Success:   true,
		Latency:   elapsed,
		LatencyMs: elapsed.Milliseconds(),
		Detail:    strings.Join(ips, ", "), // o guardá el slice completo si tu struct lo soporta
	}, nil
}
