package scan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/RobertoC117/sonarkube/internal/probe"
)

// ICMPProbe builds a PathFinderFunc for ICMP (ping) probes. It captures the
// packet count and per-packet timeout in a closure, since MakePing needs them
// but PathFinderFunc does not receive them. The result (PingStats) is reduced to
// a uniform ConnectionResult so Scan treats it like tcp/dns.
//
// Note: tcp and dns need no adapter, because probe.EstablishTCPConnection and
// probe.ResolveDNS already match the PathFinderFunc shape.
func ICMPProbe(count int, perPacketTimeout time.Duration) PathFinderFunc {
	return func(ctx context.Context, target string) (probe.ConnectionResult, error) {
		stats, err := probe.MakePing(ctx, target, count, perPacketTimeout)
		// Return nil error: a down host is a result, not a scan failure. The
		// reason travels folded inside the ConnectionResult.
		return summarizePing(stats, err), nil
	}
}

// DefaultTimeout is the max wait per target when none is specified. For icmp it
// is interpreted as the per-packet timeout.
const DefaultTimeout = 5 * time.Second

// TCPProbe builds a PathFinderFunc for TCP probes with a max time per target.
// probe.EstablishTCPConnection already has the right shape; we just add the
// per-call time limit.
func TCPProbe(timeout time.Duration) PathFinderFunc {
	return withTimeout(probe.EstablishTCPConnection, timeout)
}

// DNSProbe builds a PathFinderFunc for DNS resolution with a per-target timeout.
func DNSProbe(timeout time.Duration) PathFinderFunc {
	return withTimeout(probe.ResolveDNS, timeout)
}

// withTimeout wraps any probe so each call aborts if it takes longer than d. It
// derives a context with deadline from the received one, so it also respects
// upstream cancellation (e.g. Ctrl-C). If d <= 0, no limit applies.
func withTimeout(inner PathFinderFunc, d time.Duration) PathFinderFunc {
	if d <= 0 {
		return inner
	}
	return func(ctx context.Context, target string) (probe.ConnectionResult, error) {
		ctx, cancel := context.WithTimeout(ctx, d)
		defer cancel()
		return inner(ctx, target)
	}
}

// summarizePing is pure: it converts PingStats (and a possible MakePing error)
// into a ConnectionResult. Depending on neither network nor privileges, it is
// testable without sudo and without hitting the network.
func summarizePing(stats probe.PingStats, err error) probe.ConnectionResult {
	result := probe.ConnectionResult{Target: stats.Target}

	// Fatal error before pinging (host not resolved, missing sudo, etc.). This
	// error does not come inside PingStats, so we receive it separately.
	if err != nil {
		result.Success = false
		result.Err = err
		result.ErrMsg = err.Error()
		return result
	}

	lossPct := 0.0
	if stats.Sent > 0 {
		lossPct = float64(stats.Sent-stats.Received) / float64(stats.Sent) * 100
	}
	result.Detail = fmt.Sprintf("%d/%d received, %.0f%% loss", stats.Received, stats.Sent, lossPct)

	// A host is "alive" if it answered at least one ping.
	if stats.Received == 0 {
		result.Success = false
		result.Err = errors.New("100% packet loss")
		result.ErrMsg = result.Err.Error()
		return result
	}

	result.Success = true

	// Representative latency: average of the successful replies.
	var total time.Duration
	var n int
	for _, r := range stats.Replies {
		if r.Success {
			total += r.Latency
			n++
		}
	}
	if n > 0 {
		avg := total / time.Duration(n)
		result.Latency = avg
		result.LatencyMs = avg.Milliseconds()
	}

	return result
}
