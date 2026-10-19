package scan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/RobertoC117/sonarkube/internal/probe"
)

// ICMPProbe arma una PathFinderFunc para sondeos ICMP (ping). Captura el número
// de paquetes y el timeout por paquete en una closure, porque MakePing los pide
// pero PathFinderFunc no los recibe. El resultado (PingStats) se resume a un
// ConnectionResult uniforme para que Scan lo trate igual que a tcp/dns.
//
// Nota: tcp y dns NO necesitan adaptador, porque probe.EstablishTCPConnection y
// probe.ResolveDNS ya tienen exactamente la forma de PathFinderFunc.
func ICMPProbe(count int, perPacketTimeout time.Duration) PathFinderFunc {
	return func(ctx context.Context, target string) (probe.ConnectionResult, error) {
		stats, err := probe.MakePing(ctx, target, count, perPacketTimeout)
		// Devolvemos nil como error: un host caído es un resultado, no un fallo
		// del escaneo. El motivo viaja plegado dentro del ConnectionResult.
		return summarizePing(stats, err), nil
	}
}

// DefaultTimeout es el tiempo máximo de espera por target cuando no se
// especifica uno. Para icmp se interpreta como timeout por paquete.
const DefaultTimeout = 5 * time.Second

// TCPProbe arma una PathFinderFunc para sondeos TCP con un tiempo máximo por
// target. probe.EstablishTCPConnection ya tiene la forma correcta; solo le
// sumamos el límite de tiempo por llamada.
func TCPProbe(timeout time.Duration) PathFinderFunc {
	return withTimeout(probe.EstablishTCPConnection, timeout)
}

// DNSProbe arma una PathFinderFunc para resolución DNS con timeout por target.
func DNSProbe(timeout time.Duration) PathFinderFunc {
	return withTimeout(probe.ResolveDNS, timeout)
}

// withTimeout envuelve cualquier sonda para que cada llamada aborte si tarda más
// de d. Deriva un contexto con deadline del que recibe, así respeta también la
// cancelación de arriba (p. ej. Ctrl-C). Si d <= 0, no aplica límite.
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

// summarizePing es pura: convierte el PingStats (y un posible error de MakePing)
// en un ConnectionResult. Al no depender de la red ni de privilegios, se testea
// sin sudo y sin salir a la red.
func summarizePing(stats probe.PingStats, err error) probe.ConnectionResult {
	result := probe.ConnectionResult{Target: stats.Target}

	// Error fatal antes de poder pingear (no resolvió el host, falta sudo, etc.).
	// Ese error no viene dentro de PingStats, por eso lo recibimos aparte.
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
	result.Detail = fmt.Sprintf("%d/%d recibidos, %.0f%% pérdida", stats.Received, stats.Sent, lossPct)

	// Un host está "vivo" si respondió al menos un ping.
	if stats.Received == 0 {
		result.Success = false
		result.Err = errors.New("100% packet loss")
		result.ErrMsg = result.Err.Error()
		return result
	}

	result.Success = true

	// Latencia representativa: promedio de las respuestas exitosas.
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
