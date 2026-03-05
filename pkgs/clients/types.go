package clients

import "context"

// contextKey is an unexported type for context keys in this package.
// Using a named type prevents collisions with keys from other packages
// that also use context.WithValue.
type contextKey string

const (
	// URL_KEY is the context key used to inject the target server URL.
	// Usage: ctx = context.WithValue(ctx, clients.URL_KEY, "localhost:8080")
	URL_KEY contextKey = "url"
)

// clientFunc is the function signature every client type must implement.
// All clients receive:
//   - ctx : a cancellable context — clients MUST respect ctx.Done()
//   - id  : a unique integer identifier for logging/tracing
//   - url : the target TCP address (e.g. "localhost:8080")
type clientFunc func(ctx context.Context, id int, url string)
