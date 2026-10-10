package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidSession = errors.New("invalid or expired session")
	ErrInvalidSignup  = errors.New("invalid signup details")
	ErrUnavailable    = errors.New("authentication service is unavailable")
	usernamePattern   = regexp.MustCompile(`^[A-Za-z0-9_.-]{3,32}$`)
)

type CredentialStore interface {
	CreateCredentialUser(ctx context.Context, user domain.User, username, passwordHash string) error
	CredentialUser(ctx context.Context, identity string) (domain.User, string, error)
}

type Service struct {
	store      CredentialStore
	sessionKey []byte
	sessionTTL time.Duration
	bcryptCost int
	dummyHash  []byte
}

func NewService(store CredentialStore, sessionSecret string, sessionTTL time.Duration) (*Service, error) {
	if store == nil {
		return nil, errors.New("credential store is required")
	}
	if len(sessionSecret) < 32 {
		return nil, errors.New("session secret must contain at least 32 characters")
	}
	if sessionTTL <= 0 {
		return nil, errors.New("session TTL must be positive")
	}
	dummyHash, err := bcrypt.GenerateFromPassword([]byte("invalid-password-placeholder"), 12)
	if err != nil {
		return nil, errors.New("password security could not be initialized")
	}
	return &Service{store: store, sessionKey: []byte(sessionSecret), sessionTTL: sessionTTL, bcryptCost: 12, dummyHash: dummyHash}, nil
}

func (s *Service) Signup(ctx context.Context, username, email, password, displayName string) (domain.User, error) {
	username = strings.TrimSpace(username)
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)
	if !usernamePattern.MatchString(username) || !validEmail(email) || len(password) < 12 || len([]byte(password)) > 72 {
		return domain.User{}, ErrInvalidSignup
	}
	if displayName == "" {
		displayName = username
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return domain.User{}, errors.New("password could not be secured")
	}
	userID, err := randomUserID()
	if err != nil {
		return domain.User{}, errors.New("user identity could not be created")
	}
	user := domain.User{ID: userID, Email: email, DisplayName: displayName}
	if err := s.store.CreateCredentialUser(ctx, user, username, string(hash)); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, identity, password string) (domain.User, error) {
	identity = strings.TrimSpace(identity)
	if identity == "" || password == "" {
		return domain.User{}, domain.ErrInvalidCredentials
	}
	user, hash, err := s.store.CredentialUser(ctx, identity)
	comparisonHash := s.dummyHash
	if err == nil {
		comparisonHash = []byte(hash)
	}
	passwordErr := bcrypt.CompareHashAndPassword(comparisonHash, []byte(password))
	if err != nil && !errors.Is(err, domain.ErrInvalidCredentials) {
		return domain.User{}, ErrUnavailable
	}
	if err != nil || passwordErr != nil {
		return domain.User{}, domain.ErrInvalidCredentials
	}
	return user, nil
}

type sessionClaims struct {
	User      domain.User `json:"user"`
	IssuedAt  int64       `json:"iat"`
	ExpiresAt int64       `json:"exp"`
}

func (s *Service) CreateSession(user domain.User) (string, error) {
	now := time.Now().UTC()
	payload, err := json.Marshal(sessionClaims{User: user, IssuedAt: now.Unix(), ExpiresAt: now.Add(s.sessionTTL).Unix()})
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + s.signature(encoded), nil
}

func (s *Service) User(_ context.Context, token string) (domain.User, error) {
	encoded, providedSignature, found := strings.Cut(token, ".")
	if !found || !hmac.Equal([]byte(providedSignature), []byte(s.signature(encoded))) {
		return domain.User{}, ErrInvalidSession
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return domain.User{}, ErrInvalidSession
	}
	var claims sessionClaims
	if err := json.Unmarshal(payload, &claims); err != nil || claims.User.ID == "" || claims.User.Email == "" || time.Now().Unix() >= claims.ExpiresAt {
		return domain.User{}, ErrInvalidSession
	}
	return claims.User, nil
}

func (s *Service) SessionTTL() time.Duration { return s.sessionTTL }

func (s *Service) signature(payload string) string {
	digest := hmac.New(sha256.New, s.sessionKey)
	_, _ = digest.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(digest.Sum(nil))
}

func validEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && strings.EqualFold(address.Address, value) && len(value) <= 254
}

func randomUserID() (string, error) {
	value := make([]byte, 18)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return "usr_" + base64.RawURLEncoding.EncodeToString(value), nil
}
