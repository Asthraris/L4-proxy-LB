package clients

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net"
	"time"
)

// ────────────────────────────────────────────────────────────────────────────
// fireAndForget
//
// Behaviour : connects, fires a single PING, then immediately disconnects.
// Real-world analogy : a health-check probe or a UDP-style one-shot sender.
// ────────────────────────────────────────────────────────────────────────────
func fireAndForget(ctx context.Context, id int, url string) {
	conn, err := dialWithContext(ctx, url)
	if err != nil {
		log.Printf("[fireAndForget][%d] dial: %v", id, err)
		return
	}
	defer conn.Close()

	payload := fmt.Sprintf("PING %d\n", id)
	if _, err := fmt.Fprint(conn, payload); err != nil {
		log.Printf("[fireAndForget][%d] write: %v", id, err)
	}
	// no read — fire and forget
}

// ────────────────────────────────────────────────────────────────────────────
// realClient
//
// Behaviour : connects, sends REQ / waits for response a random number of
//             times (1-9), with a random sleep between each exchange.
// Real-world analogy : a typical HTTP keep-alive client doing several
//                      requests before closing.
// ────────────────────────────────────────────────────────────────────────────
func realClient(ctx context.Context, id int, url string) {
	conn, err := dialWithContext(ctx, url)
	if err != nil {
		log.Printf("[realClient][%d] dial: %v", id, err)
		return
	}
	defer conn.Close()

	iterations := rand.IntN(9) + 1 // 1-9  (never 0)

	for i := 0; i < iterations; i++ {
		// Respect cancellation before every exchange
		select {
		case <-ctx.Done():
			return
		default:
		}

		if _, err := fmt.Fprintf(conn, "REQ %d\n", i); err != nil {
			log.Printf("[realClient][%d] write: %v", id, err)
			return
		}

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("[realClient][%d] read: %v", id, err)
			}
			return
		}
		_ = buf[:n] // consume response

		sleep := time.Duration(rand.IntN(800)+100) * time.Millisecond
		select {
		case <-ctx.Done():
			return
		case <-time.After(sleep):
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// streamClient
//
// Behaviour : connects and continuously streams data to the server until the
//             context is cancelled.  No response is expected.
// Real-world analogy : a log-shipping agent, a metrics pusher, or a live
//                      sensor feeding data into a pipeline.
// ────────────────────────────────────────────────────────────────────────────
func streamClient(ctx context.Context, id int, url string) {
	conn, err := dialWithContext(ctx, url)
	if err != nil {
		log.Printf("[streamClient][%d] dial: %v", id, err)
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	seq := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			msg := fmt.Sprintf("STREAM %d seq=%d\n", id, seq)
			if _, err := fmt.Fprint(conn, msg); err != nil {
				log.Printf("[streamClient][%d] write: %v", id, err)
				return
			}
			seq++
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// idleClient
//
// Behaviour : connects and holds the connection open without sending anything.
//             Disconnects only when the context is cancelled.
// Real-world analogy : a zombie connection — a client that connected but
//                      stopped responding (e.g. a crashed process that left
//                      a socket open).  Tests server-side idle timeouts.
// ────────────────────────────────────────────────────────────────────────────
func idleClient(ctx context.Context, id int, url string) {
	conn, err := dialWithContext(ctx, url)
	if err != nil {
		log.Printf("[idleClient][%d] dial: %v", id, err)
		return
	}
	defer conn.Close()

	// Just hold the connection until cancelled
	<-ctx.Done()
}

// ────────────────────────────────────────────────────────────────────────────
// slowClient
//
// Behaviour : sends data one byte at a time with large delays between each
//             byte — the classic "slowloris" style misbehaving client.
// Real-world analogy : a client on an extremely poor connection, or a
//                      deliberate slowloris DoS probe.  Tests whether the
//                      server can handle slow senders without blocking others.
// ────────────────────────────────────────────────────────────────────────────
func slowClient(ctx context.Context, id int, url string) {
	conn, err := dialWithContext(ctx, url)
	if err != nil {
		log.Printf("[slowClient][%d] dial: %v", id, err)
		return
	}
	defer conn.Close()

	msg := fmt.Sprintf("SLOW %d\n", id)

	for _, b := range []byte(msg) {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(rand.IntN(300)+100) * time.Millisecond):
		}

		if _, err := conn.Write([]byte{b}); err != nil {
			log.Printf("[slowClient][%d] write: %v", id, err)
			return
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// dialWithContext is a helper that respects context cancellation during dial.
// net.Dial itself is blocking; this wraps it so a cancelled context aborts it.
// ────────────────────────────────────────────────────────────────────────────
func dialWithContext(ctx context.Context, url string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, "tcp", url)
}
