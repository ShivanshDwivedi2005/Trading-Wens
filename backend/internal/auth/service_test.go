package auth

import (
	"context"
	"testing"
	"time"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

type credentialStoreStub struct {
	user     domain.User
	username string
	hash     string
}

func (s *credentialStoreStub) CreateCredentialUser(_ context.Context, user domain.User, username, hash string) error {
	s.user, s.username, s.hash = user, username, hash
	return nil
}

func (s *credentialStoreStub) CredentialUser(_ context.Context, identity string) (domain.User, string, error) {
	if identity != s.username && identity != s.user.Email {
		return domain.User{}, "", domain.ErrInvalidCredentials
	}
	return s.user, s.hash, nil
}

func TestSignupLoginAndSession(t *testing.T) {
	store := &credentialStoreStub{}
	service, err := NewService(store, "01234567890123456789012345678901", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	service.bcryptCost = 4
	user, err := service.Signup(context.Background(), "market.user", "Market@Example.com", "a-secure-password", "Market User")
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "market@example.com" || store.username != "market.user" {
		t.Fatalf("unexpected registered user: %#v", user)
	}
	loggedIn, err := service.Login(context.Background(), "market.user", "a-secure-password")
	if err != nil || loggedIn.ID != user.ID {
		t.Fatalf("login failed: %#v %v", loggedIn, err)
	}
	token, err := service.CreateSession(user)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.User(context.Background(), token); err != nil {
		t.Fatalf("session failed: %v", err)
	}
}

func TestSignupValidationAndInvalidLogin(t *testing.T) {
	service, _ := NewService(&credentialStoreStub{}, "01234567890123456789012345678901", time.Hour)
	service.bcryptCost = 4
	if _, err := service.Signup(context.Background(), "x", "invalid", "short", ""); err != ErrInvalidSignup {
		t.Fatalf("expected signup validation error, got %v", err)
	}
	if _, err := service.Login(context.Background(), "missing", "password"); err != domain.ErrInvalidCredentials {
		t.Fatalf("expected generic login error, got %v", err)
	}
}
