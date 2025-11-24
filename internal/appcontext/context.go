package appcontext

import (
	"context"
)

type contextKey string

const (
	userEmailKey  contextKey = "userEmail"
	isLoggedInKey contextKey = "isLoggedIn"
)

func WithUser(ctx context.Context, email string, loggedIn bool) context.Context {
	ctx = context.WithValue(ctx, userEmailKey, email)
	ctx = context.WithValue(ctx, isLoggedInKey, loggedIn)
	return ctx
}

func GetUser(ctx context.Context) (string, bool) {
	email, _ := ctx.Value(userEmailKey).(string)
	loggedIn, _ := ctx.Value(isLoggedInKey).(bool)
	return email, loggedIn
}
