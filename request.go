package digo

import (
	"context"

	"github.com/google/uuid"
)

// RequestScope attaches a request id to ctx (reuses existing non-empty id) and
// returns end(), which shuts down only that request's managed instances.
func RequestScope(parent context.Context) (ctx context.Context, end func() error, err error) {
	return getContainer().RequestScope(parent)
}

// RequestScope is the container-scoped variant of RequestScope.
func (c *Container) RequestScope(parent context.Context) (ctx context.Context, end func() error, err error) {
	if parent == nil {
		parent = context.Background()
	}
	id, ok := RequestID(parent)
	if !ok {
		id = uuid.NewString()
		parent = WithRequestID(parent, id)
	}
	ctx = withContainer(parent, c)
	ended := false
	end = func() error {
		if ended {
			return nil
		}
		ended = true
		return c.EndRequest(id)
	}
	return ctx, end, nil
}
