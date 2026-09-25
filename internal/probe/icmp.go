package probe

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type PingReply struct {
	Seq       int           `json:"seq"`
	Success   bool          `json:"success"`
	Latency   time.Duration `json:"-"`
	LatencyMs int64         `json:"latency_ms,omitempty"`
	Err       error         `json:"-"`
	ErrMsg    string        `json:"error,omitempty"`
}

type PingStats struct {
	Target   string      `json:"target"`
	Sent     int         `json:"sent"`
	Received int         `json:"received"`
	Replies  []PingReply `json:"replies"`
}

func (s PingStats) RenderTable() (string, error) {
	var b strings.Builder
	for _, r := range s.Replies {
		if r.Success {
			_, err := fmt.Fprintf(&b, "Reply from %v: seq=%d time=%dms\n", s.Target, r.Seq, r.LatencyMs)
			if err != nil {
				return b.String(), err
			}
		} else {
			_, err := fmt.Fprintf(&b, "Request seq=%d failed: %v\n", r.Seq, r.Err)
			if err != nil {
				return b.String(), err
			}
		}
	}

	lossPct := 0.0
	if s.Sent > 0 {
		lossPct = float64(s.Sent-s.Received) / float64(s.Sent) * 100
	}
	_, err := fmt.Fprintf(&b, "\n--- %v ping statistics ---\n", s.Target)
	if err != nil {
		return b.String(), err
	}
	_, err = fmt.Fprintf(&b, "%d packets transmitted, %d received, %.1f%% packet loss\n", s.Sent, s.Received, lossPct)
	if err != nil {
		return b.String(), err
	}
	return b.String(), nil
}

func MakePing(ctx context.Context, host string, count int, perPacketTimeout time.Duration) (PingStats, error) {
	stats := PingStats{Target: host}

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return stats, fmt.Errorf("sending ICMP requires administrator privileges (try with sudo): %w", err)
		}
		return stats, err
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			log.Printf("error closing ICMP connection: %v", cerr)
		}
	}()

	dst, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return stats, err
	}

	id := os.Getpid() & 0xffff

	for seq := 1; seq <= count; seq++ {
		select {
		case <-ctx.Done():
			return stats, ctx.Err()
		default:
		}

		start := time.Now()

		// Deadline propio por paquete: un eco perdido no debe consumir
		// el presupuesto de tiempo de los siguientes.
		if err := conn.SetDeadline(start.Add(perPacketTimeout)); err != nil {
			return stats, err
		}

		msg := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   id,
				Seq:  seq,
				Data: []byte("sonarkube-ping"),
			},
		}

		wb, err := msg.Marshal(nil)
		if err != nil {
			stats.Replies = append(stats.Replies, PingReply{Seq: seq, Success: false, Err: err, ErrMsg: err.Error()})
			continue
		}

		if _, err := conn.WriteTo(wb, dst); err != nil {
			stats.Replies = append(stats.Replies, PingReply{Seq: seq, Success: false, Err: err, ErrMsg: err.Error()})
			continue
		}
		stats.Sent++

		rm, err := waitForEchoReply(conn, id, seq)
		latency := time.Since(start)
		if err != nil {
			stats.Replies = append(stats.Replies, PingReply{
				Seq: seq, Success: false, Latency: latency, LatencyMs: latency.Milliseconds(),
				Err: err, ErrMsg: err.Error(),
			})
			continue
		}
		if rm.Type != ipv4.ICMPTypeEchoReply {
			unexpectedErr := fmt.Errorf("unexpected ICMP response: %v", rm.Type)
			stats.Replies = append(stats.Replies, PingReply{
				Seq: seq, Success: false, Latency: latency, LatencyMs: latency.Milliseconds(),
				Err: unexpectedErr, ErrMsg: unexpectedErr.Error(),
			})
			continue
		}

		stats.Received++
		stats.Replies = append(stats.Replies, PingReply{Seq: seq, Success: true, Latency: latency, LatencyMs: latency.Milliseconds()})
	}

	return stats, nil
}

// waitForEchoReply lee paquetes hasta encontrar una respuesta que corresponda
// a este ID+Seq (ignora respuestas de otras sondas ICMP que puedan llegar),
// o hasta que se cumpla el deadline seteado en el conn.
func waitForEchoReply(conn *icmp.PacketConn, id, seq int) (*icmp.Message, error) {
	buf := make([]byte, 1500)
	for {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			return nil, err
		}

		rm, err := icmp.ParseMessage(1, buf[:n]) // 1 = número de protocolo IANA para ICMPv4
		if err != nil {
			return nil, err
		}

		echo, ok := rm.Body.(*icmp.Echo)
		if !ok || echo.ID != id || echo.Seq != seq {
			continue
		}
		return rm, nil
	}
}
