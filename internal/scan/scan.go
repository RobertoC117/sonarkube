package scan

import (
	"context"

	"github.com/RobertoC117/sonarkube/internal/probe"
	"golang.org/x/sync/errgroup"
)

// PathFinderFunc es la "sonda": recibe un destino y devuelve su resultado.
// Scan la recibe como parámetro para poder inyectar sondas reales (tcp, dns,
// icmp) o falsas en los tests.
type PathFinderFunc func(ctx context.Context, target string) (probe.ConnectionResult, error)

// Scan sondea todos los targets en paralelo, con un máximo de `workers` sondeos
// simultáneos. Devuelve un resultado por destino, en el mismo orden que la
// entrada. Si el contexto se cancela (timeout o Ctrl-C), corta cuanto antes y
// devuelve el error del contexto junto con los resultados parciales.
func Scan(ctx context.Context, targets []string, concurrency int, pathfinder PathFinderFunc) (probe.ConnectionResults, error) {
	// gctx es el contexto derivado del grupo; lo pasamos a cada sonda para que
	// la cancelación se propague. OJO: errgroup cancela gctx apenas Wait()
	// retorna, así que NO sirve para saber si hubo cancelación "real".
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	results := make(probe.ConnectionResults, len(targets))

	for i, target := range targets {
		g.Go(func() error {
			// Un sondeo fallido NO debe cancelar a los demás: el fallo ya
			// viaja dentro del ConnectionResult, así que devolvemos nil.
			// Cada goroutine escribe en su propio índice, sin pisar a otras.
			results[i], _ = pathfinder(gctx, target)
			return nil
		})
	}

	_ = g.Wait() // los workers siempre devuelven nil, así que Wait no da error

	// Consultamos el contexto ORIGINAL (no el derivado): su Err() es nil si el
	// llamador no canceló, o el motivo (timeout / Ctrl-C) si sí lo hizo.
	return results, ctx.Err()
}
