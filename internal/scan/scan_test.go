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

// okProbe es una sonda falsa sencilla: no toca la red y siempre devuelve
// un resultado exitoso para el destino que recibe.
func okProbe(ctx context.Context, target string) (probe.ConnectionResult, error) {
	return probe.ConnectionResult{
		Target:  target,
		Success: true,
	}, nil
}

// countingProbe es una sonda falsa con estado: cuenta cuántos sondeos corren
// a la vez y recuerda el pico máximo, para validar el límite de concurrencia.
type countingProbe struct {
	inFlight atomic.Int32
	maxSeen  atomic.Int32
}

func (c *countingProbe) fakePathFinder(ctx context.Context, target string) (probe.ConnectionResult, error) {
	current := c.inFlight.Add(1)
	defer c.inFlight.Add(-1)

	// Guardar el pico de forma segura entre goroutines.
	for {
		old := c.maxSeen.Load()
		if current <= old {
			break
		}
		if c.maxSeen.CompareAndSwap(old, current) {
			break
		}
	}

	time.Sleep(10 * time.Millisecond) // simula trabajo para que los sondeos se solapen

	return probe.ConnectionResult{
		Target:  target,
		Success: true,
	}, nil
}

// blockingProbe es una sonda falsa que RESPETA el ctx: "trabaja" un tiempo
// largo, pero abandona de inmediato si el contexto se cancela. Es la clave
// para probar que la cancelación corta el escaneo.
func blockingProbe(ctx context.Context, target string) (probe.ConnectionResult, error) {
	select {
	case <-ctx.Done():
		// Me cancelaron: devuelvo ya, con el motivo dentro del resultado.
		return probe.ConnectionResult{Target: target, Success: false, Err: ctx.Err()}, ctx.Err()
	case <-time.After(500 * time.Millisecond):
		// Alcancé a "trabajar" todo mi tiempo sin que me cancelen.
		return probe.ConnectionResult{Target: target, Success: true}, nil
	}
}

func TestScan_DevuelveUnResultadoPorTarget(t *testing.T) {
	targets := []string{
		"www.google.com",
		"www.youtube.com",
		"www.github.com",
	}

	got, err := Scan(context.Background(), targets, 50, okProbe)
	if err != nil {
		t.Fatalf("no esperaba error pero recibí: %v", err)
	}

	if len(got) != len(targets) {
		t.Errorf("esperaba %v resultados pero recibí %v", len(targets), len(got))
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

func TestScan_RespetaLimiteDeConcurrencia(t *testing.T) {
	counter := &countingProbe{}
	workers := 5

	targets := make([]string, 20)
	for i := range targets {
		targets[i] = fmt.Sprintf("target-%v", i)
	}

	if _, err := Scan(context.Background(), targets, workers, counter.fakePathFinder); err != nil {
		t.Fatalf("no esperaba error pero recibí: %v", err)
	}

	max := counter.maxSeen.Load()

	if max > int32(workers) {
		t.Errorf("se superó el límite de workers: pico %v, permitido %v", max, workers)
	}

	if max <= 1 {
		t.Errorf("sin paralelismo: pico en vuelo = %v", max)
	}
}

func TestScan_CancelaAlCancelarElContexto(t *testing.T) {
	targets := make([]string, 20)
	for i := range targets {
		targets[i] = fmt.Sprintf("target-%v", i)
	}

	// El contexto se cancela solo a los 50ms.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := Scan(ctx, targets, 5, blockingProbe)
	elapsed := time.Since(start)

	// Sin cancelación, 20 destinos de 500ms con 5 workers tardarían ~2s.
	// Con la cancelación a los 50ms, Scan debe cortar mucho antes.
	if elapsed > 200*time.Millisecond {
		t.Errorf("Scan no respetó la cancelación: tardó %v", elapsed)
	}

	if err == nil {
		t.Error("esperaba un error de cancelación, pero Scan devolvió nil")
	}
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
