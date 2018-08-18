package metadata

import (
	"context"
)

type metaKey struct{}

// Metadata is our way of representing request headers internally.
// They're used at the RPC level and translate back and forth
// from Transport headers.
type Metadata map[string]interface{}

func FromContext(ctx context.Context) (Metadata, bool) {
	md, ok := ctx.Value(metaKey{}).(Metadata)
	return md, ok
}

func NewContext(ctx context.Context, md Metadata) context.Context {
	// create empty context if ctx is nil
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, metaKey{}, md)
}
