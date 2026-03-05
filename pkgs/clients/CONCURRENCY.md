# Client Simulation — Concurrency & Design Notes

> Reference document for the `pkgs/clients` package and `cmd/client` binary.
> Read this before touching the goroutine/context wiring.

---

## 1. The Context Tree

```
context.Background()
    └── context.WithCancel()        → rootCtx  (owned by main)
            └── context.WithValue() → rootCtx  (URL_KEY injected)
                    ├── runConstantSpawner(rootCtx, ...)
                    ├── runBurstSpawner(rootCtx, ...)
                    └── every client goroutine receives rootCtx
```

### Why one root context?

A single cancellable root means **one call to `cancel()`** — triggered by the
user typing `q` — tears down every spawner and every in-flight client at once.
No manual signalling, no channel fan-out, no extra bookkeeping.

### Why `context.WithValue` for the URL?

The URL is an immutable piece of configuration that every client needs.
Passing it through `WithValue` means `LaunchClients`, `RunConstantSpawner`,
and any future launcher can all read it without the caller having to thread
an extra parameter through every intermediate function.

**Rule:** only package-level keys (typed `contextKey`) go into this context.
Never store mutable state or large objects in a context.

---

## 2. Goroutine Lifecycle

### Who creates goroutines?

| Goroutine              | Created by          | Ends when                        |
|------------------------|---------------------|----------------------------------|
| `runConstantSpawner`   | `main`              | `ctx.Done()` fires               |
| `runBurstSpawner`      | `main`              | `ctx.Done()` or user types `q`   |
| Each client (e.g. `realClient`) | `LaunchClients` | client logic completes **or** `ctx.Done()` fires |
| Batch monitor          | `LaunchClients`     | all clients in batch finish      |

### Why does `LaunchClients` return immediately?

`LaunchClients` is called from the burst spawner's input loop.  If it
blocked until all clients finished, the input loop would freeze — you couldn't
type more commands while a burst was running.

The internal `sync.WaitGroup` is monitored by a background goroutine that
logs when the batch finishes.  The caller is never blocked.

### The `go clients.LaunchClients(ctx, total)` pattern in main

Even though `LaunchClients` is itself non-blocking, wrapping it in `go` is
still correct here because:
1. It makes the intent explicit — this is a fire-and-forget burst.
2. If the implementation ever becomes briefly blocking (e.g. rate-limiting),
   the input loop won't stutter.

---

## 3. Context Cancellation in Clients

Every client function **must** check `ctx.Done()` at natural pause points.

### Pattern A — select in a loop (realClient, streamClient)

```go
for {
    select {
    case <-ctx.Done():
        return          // clean exit
    case <-ticker.C:
        // do work
    }
}
```

### Pattern B — select before a blocking call (realClient sleep)

```go
select {
case <-ctx.Done():
    return
case <-time.After(sleep):
    // continue after sleep
}
```

Never use `time.Sleep` directly inside a client loop — it ignores context
cancellation and the goroutine will linger until the sleep expires.

### Pattern C — dialWithContext

```go
var d net.Dialer
return d.DialContext(ctx, "tcp", url)
```

`net.Dialer.DialContext` aborts the TCP handshake immediately if the context
is cancelled during dial.  Plain `net.Dial` does not.

---

## 4. The Weighted Client Roster

Defined in `launcher.go`:

```go
var clientRoster = []weightedClient{
    {name: "realClient",    fn: realClient,    weight: 50},
    {name: "fireAndForget", fn: fireAndForget, weight: 20},
    {name: "streamClient",  fn: streamClient,  weight: 15},
    {name: "idleClient",    fn: idleClient,    weight: 10},
    {name: "slowClient",    fn: slowClient,    weight:  5},
}
```

`pickClient()` does a weighted random draw.  To change the mix, change the
weights here — **no other code needs to change**.  To add a new client type:

1. Write the function in `clients.go` with signature `clientFunc`.
2. Add a row to `clientRoster`.

### Client type reference

| Type           | Sends data | Reads response | Duration          | Models                          |
|----------------|------------|----------------|-------------------|---------------------------------|
| `realClient`   | yes        | yes            | short (1-9 reqs)  | Typical HTTP keep-alive user    |
| `fireAndForget`| yes        | no             | instant           | Health-check probe, UDP-like    |
| `streamClient` | yes (loop) | no             | until ctx cancel  | Log shipper, metrics agent      |
| `idleClient`   | no         | no             | until ctx cancel  | Zombie / leaked connection      |
| `slowClient`   | byte-by-byte| no            | slow              | Slowloris, bad-network client   |

---

## 5. The Two Spawners

### Constant spawner (`runConstantSpawner`)

- Ticks every second via `time.NewTicker`.
- Reads the current rate from an `atomic.Int64` — safe to update from the
  burst goroutine without a mutex.
- Calls `clients.LaunchClients(ctx, n)` each tick.

### Burst spawner (`runBurstSpawner`)

- Blocks on `bufio.Reader.ReadString('\n')` — this is intentionally in its
  own goroutine so main is never blocked.
- Parses commands:
  - `<number>` → burst `number × 100` clients via `go clients.LaunchClients`
  - `r<number>` → atomically updates the constant-spawner rate
  - `q` → calls `cancel()`, which cascades to every goroutine

---

## 6. Atomic Rate vs Mutex

The constant-spawner rate is a single `int64`.  We use `sync/atomic` instead
of a `sync.Mutex` because:

- The value is a plain integer (not a struct).
- Only one goroutine writes (`runBurstSpawner`), one reads (`runConstantSpawner`).
- Atomic ops are cheaper and have no lock-ordering risk.

If the shared state ever grows (e.g. a config struct with multiple fields),
replace the atomic with a `sync.RWMutex`-protected struct.

---

## 7. Why `contextKey` Is an Unexported Named Type

```go
type contextKey string
const URL_KEY contextKey = "url"
```

If `URL_KEY` were a plain `string`, any other package that also stored the
string `"url"` in the same context would silently collide.  A named unexported
type makes the key unique to this package — collision is impossible even if
another package uses the same string literal.

---

## 8. Shutdown Sequence

```
user types "q"
    → runBurstSpawner calls cancel()
        → rootCtx.Done() closes
            → runConstantSpawner sees Done(), returns
            → every client goroutine sees Done() at its next select, returns
            → dialWithContext aborts any in-progress dials
    → main unblocks from <-rootCtx.Done()
    → deferred cancel() is a no-op (already called)
    → 200ms grace sleep lets goroutines log shutdown messages
    → process exits
```

No goroutine leaks: every goroutine either has a `<-ctx.Done()` arm or a
function that returns naturally after finite work.

---

## 9. Future Work / TODOs

- [ ] Add `sync.WaitGroup` to main so we wait for ALL client goroutines
      before exit (currently just a 200ms sleep).
- [ ] Expose per-type counters (e.g. `expvar` or Prometheus) so you can see
      live how many of each client type are active.
- [ ] Add `--rate` and `--url` CLI flags (via `flag` package) so the tool
      is scriptable without interactive prompts.
- [ ] Add a reconnect loop to `streamClient` and `idleClient` for
      resilience testing.
- [ ] Consider a `rampClient` that slowly increases send rate — useful for
      finding the server's saturation point.
