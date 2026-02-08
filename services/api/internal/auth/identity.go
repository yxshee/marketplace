package auth

import "context"

// Identity is the authenticated principal bound to request context.
type Identity struct {
	UserID    string
	Role      Role
	SessionID string
	VendorID  *string
}

type identityContextKey string

const identityKey identityContextKey = "request_identity"

func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey, identity)
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey).(Identity)
	return identity, ok
}
