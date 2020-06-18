package httpx

import (
	"context"
	"net/http"

	horizonContext "github.com/stellar/go/services/horizon/internal/context"
)

func RequestFromContext(ctx context.Context) *http.Request {
	found, _ := ctx.Value(&horizonContext.RequestContextKey).(*http.Request)
	return found
}

// RequestContext returns a context representing the provided http action.
func RequestContext(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
	if r == nil {
		panic("Cannot bind nil *http.Request to context tree")
	}

	return context.WithValue(ctx, &horizonContext.RequestContextKey, r)
}
