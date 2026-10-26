package scan

import (
	"context"

	"github.com/RobertoC117/sonarkube/internal/probe"
	"golang.org/x/sync/errgroup"
)

// PathFinderFunc is the "probe": it takes a target and returns its result. Scan
// receives it as a parameter so real probes (tcp, dns, icmp) or fakes can be
// injected in tests.
type PathFinderFunc func(ctx context.Context, target string) (probe.ConnectionResult, error)

// Scan probes all targets in parallel, with at most `workers` simultaneous
// probes. It returns one result per target, in input order. If the context is
// cancelled (timeout or Ctrl-C), it stops as soon as possible and returns the
// context error along with partial results.
func Scan(ctx context.Context, targets []string, concurrency int, pathfinder PathFinderFunc) (probe.ConnectionResults, error) {
	// gctx is the group-derived context; we pass it to each probe so
	// cancellation propagates. Note: errgroup cancels gctx as soon as Wait()
	// returns, so it cannot tell whether a "real" cancellation happened.
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	results := make(probe.ConnectionResults, len(targets))

	for i, target := range targets {
		g.Go(func() error {
			// A failed probe must NOT cancel the others: the failure already
			// travels inside the ConnectionResult, so we return nil. Each
			// goroutine writes to its own index without clobbering others.
			results[i], _ = pathfinder(gctx, target)
			return nil
		})
	}

	_ = g.Wait() // workers always return nil, so Wait never errors

	// Query the ORIGINAL context (not the derived one): its Err() is nil if the
	// caller did not cancel, or the reason (timeout / Ctrl-C) if it did.
	return results, ctx.Err()
}
