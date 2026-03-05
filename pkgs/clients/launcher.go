package clients

import (
	"context"
	"log"
	"math/rand/v2"
	"sync"
	"time"
)

// ────────────────────────────────────────────────────────────────────────────
// weightedClient pairs a clientFunc with a spawn probability weight.
// The higher the weight relative to the total, the more often that type
// is chosen when picking at random.
// ────────────────────────────────────────────────────────────────────────────
type weightedClient struct {
	name   string
	fn     clientFunc
	weight int
}

// clientRoster defines every client type and how frequently it appears.
// Adjust weights here — no other code needs to change.
var clientRoster = []weightedClient{
	{name: "realClient", fn: realClient, weight: 50},     // most common — typical user
	{name: "fireAndForget", fn: fireAndForget, weight: 20}, // light probes
	{name: "streamClient", fn: streamClient, weight: 15},  // persistent streamers
	{name: "idleClient", fn: idleClient, weight: 10},      // zombie holders
	{name: "slowClient", fn: slowClient, weight: 5},       // slowloris / bad actors
}

// pickClient selects a clientFunc using weighted random selection.
func pickClient() weightedClient {
	total := 0
	for _, c := range clientRoster {
		total += c.weight
	}

	r := rand.IntN(total)
	for _, c := range clientRoster {
		r -= c.weight
		if r < 0 {
			return c
		}
	}
	return clientRoster[0] // fallback — should never be reached
}

// ────────────────────────────────────────────────────────────────────────────
// LaunchClients spawns n clients and returns immediately.
// Each client runs in its own goroutine and respects ctx cancellation.
// A WaitGroup is used internally so we can log when the whole batch finishes
// — the caller is never blocked.
// ────────────────────────────────────────────────────────────────────────────
func LaunchClients(ctx context.Context, n int) {
	url, ok := urlFromCtx(ctx)
	if !ok {
		return
	}

	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		chosen := pickClient()
		id := i

		go func() {
			defer wg.Done()
			chosen.fn(ctx, id, url)
		}()
	}

	// Monitor in background so LaunchClients itself is non-blocking
	go func() {
		wg.Wait()
		log.Printf("[launcher] batch of %d clients finished", n)
	}()
}

// ────────────────────────────────────────────────────────────────────────────
// RunConstantSpawner launches `rate` clients every second for as long as
// ctx is alive.  Call it in its own goroutine from main.
//
//	go clients.RunConstantSpawner(ctx, 5) // 5 new clients per second
func RunConstantSpawner(ctx context.Context, rate int) {
	_, ok := urlFromCtx(ctx)
	if !ok {
		return
	}

	log.Printf("[constant-spawner] starting — %d clients/sec", rate)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// globalID keeps IDs unique across all ticks
	globalID := 0

	for {
		select {
		case <-ctx.Done():
			log.Println("[constant-spawner] stopped")
			return

		case <-ticker.C:
			// Re-read url each tick (ctx cannot change, but this is defensive)
			url, ok := urlFromCtx(ctx)
			if !ok {
				return
			}
			for i := 0; i < rate; i++ {
				chosen := pickClient()
				id := globalID
				globalID++
				go chosen.fn(ctx, id, url)
			}
			log.Printf("[constant-spawner] spawned %d clients (total so far: %d)", rate, globalID)
		}
	}
}

// urlFromCtx is a small helper that extracts and validates the URL from ctx.
func urlFromCtx(ctx context.Context) (string, bool) {
	val := ctx.Value(URL_KEY)
	url, ok := val.(string)
	if !ok || url == "" {
		log.Println("[launcher] ERR: URL not found in context — did you set clients.URL_KEY?")
		return "", false
	}
	return url, true
}
