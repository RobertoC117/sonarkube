package scan

import (
	"context"
	"fmt"
	"go.uber.org/goleak"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RobertoC117/sonarkube/internal/probe"
)

// okProbe is a simple fake probe: it never touches the network and always
// returns a successful result for the target it receives.
func okProbe(ctx context.Context, target string) (probe.ConnectionResult, error) {
	return probe.ConnectionResult{
		Target:  target,
		Success: true,
	}, nil
}

// countingProbe is a stateful fake probe: it counts how many probes run at once
// and remembers the peak, to validate the concurrency limit.
type countingProbe struct {
	inFlight atomic.Int32
	maxSeen  atomic.Int32
}

func (c *countingProbe) fakePathFinder(ctx context.Context, target string) (probe.ConnectionResult, error) {
	current := c.inFlight.Add(1)
	defer c.inFlight.Add(-1)

	// Store the peak safely across goroutines.
	for {
		old := c.maxSeen.Load()
		if current <= old {
			break
		}
		if c.maxSeen.CompareAndSwap(old, current) {
			break
		}
	}

	time.Sleep(10 * time.Millisecond) // simulate work so probes overlap

	return probe.ConnectionResult{
		Target:  target,
		Success: true,
	}, nil
}

// blockingProbe is a fake probe that RESPECTS ctx: it "works" for a long time
// but returns immediately if the context is cancelled. Key to proving that
// cancellation cuts the scan short.
func blockingProbe(ctx context.Context, target string) (probe.ConnectionResult, error) {
	select {
	case <-ctx.Done():
		// Cancelled: return now, with the reason inside the result.
		return probe.ConnectionResult{Target: target, Success: false, Err: ctx.Err()}, ctx.Err()
	case <-time.After(500 * time.Millisecond):
		// Finished all my "work" without being cancelled.
		return probe.ConnectionResult{Target: target, Success: true}, nil
	}
}

func TestScan_ReturnsOneResultPerTarget(t *testing.T) {
	targets := []string{
		"www.google.com",
		"www.youtube.com",
		"www.github.com",
	}

	got, err := Scan(context.Background(), targets, 50, okProbe)
	if err != nil {
		t.Fatalf("did not expect an error, got: %v", err)
	}

	if len(got) != len(targets) {
		t.Errorf("expected %v results, got %v", len(targets), len(got))
	}

	scannedTargets := make(map[string]bool)
	for _, res := range got {
		scannedTargets[res.Target] = true
	}

	for _, target := range targets {
		if !scannedTargets[target] {
			t.Errorf("el destino %v no fue escaneado", target)
		}
	}
}

func TestScan_RespectsConcurrencyLimit(t *testing.T) {
	counter := &countingProbe{}
	workers := 5

	targets := make([]string, 20)
	for i := range targets {
		targets[i] = fmt.Sprintf("target-%v", i)
	}

	if _, err := Scan(context.Background(), targets, workers, counter.fakePathFinder); err != nil {
		t.Fatalf("did not expect an error, got: %v", err)
	}

	max := counter.maxSeen.Load()

	if max > int32(workers) {
		t.Errorf("exceeded the worker limit: peak %v, allowed %v", max, workers)
	}

	if max <= 1 {
		t.Errorf("sin paralelismo: pico en vuelo = %v", max)
	}
}

func TestScan_CancelsOnContextCancellation(t *testing.T) {
	targets := make([]string, 20)
	for i := range targets {
		targets[i] = fmt.Sprintf("target-%v", i)
	}

	// The context auto-cancels after 50ms.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := Scan(ctx, targets, 5, blockingProbe)
	elapsed := time.Since(start)

	// Without cancellation, 20 targets of 500ms with 5 workers would take ~2s.
	// With cancellation at 50ms, Scan must cut off much sooner.
	if elapsed > 200*time.Millisecond {
		t.Errorf("Scan did not honor cancellation: took %v", elapsed)
	}

	if err == nil {
		t.Error("expected a cancellation error, but Scan returned nil")
	}
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
