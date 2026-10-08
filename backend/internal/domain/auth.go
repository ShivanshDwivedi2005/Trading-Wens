package domain

import "context"

type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
}

type authenticatedUserKey struct{}

func WithAuthenticatedUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, authenticatedUserKey{}, user)
}

func AuthenticatedUser(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(authenticatedUserKey{}).(User)
	return user, ok && user.ID != ""
}
