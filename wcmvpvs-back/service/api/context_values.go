package api

import "context"

type hashedIPContextKey struct{}
type bypassCodeContextKey struct{}

func withHashedIP(ctx context.Context, hash string) context.Context {
	if hash == "" {
		return ctx
	}
	return context.WithValue(ctx, hashedIPContextKey{}, hash)
}

func getHashedIP(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(hashedIPContextKey{}).(string); ok {
		return v
	}
	return ""
}

func withBypassCode(ctx context.Context, code string) context.Context {
	if code == "" {
		return ctx
	}
	return context.WithValue(ctx, bypassCodeContextKey{}, code)
}

func getBypassCode(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(bypassCodeContextKey{}).(string); ok {
		return v
	}
	return ""
}
