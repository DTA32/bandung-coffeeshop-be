package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
	"golang.org/x/oauth2"
)

var (
	ErrAuthUnavailable   = errors.New("auth unavailable")
	ErrAuthStateMismatch = errors.New("oauth state mismatch")
	ErrDeactivated       = errors.New("account deactivated")
	ErrNoSession         = errors.New("no session")
)

// LoginState is round-tripped through the short-lived bdg_oauth cookie.
type LoginState struct {
	State    string `json:"s"`
	Verifier string `json:"v"`
	Redirect string `json:"r"`
}

type AuthService struct {
	users    *repository.UserRepository
	sessions *repository.SessionRepository
	google   *repository.GoogleRepository
}

// google is nil when Google credentials are not configured.
func NewAuthService(users *repository.UserRepository, sessions *repository.SessionRepository, google *repository.GoogleRepository) *AuthService {
	return &AuthService{users: users, sessions: sessions, google: google}
}

func (s *AuthService) Available(ctx context.Context) bool {
	return s.google != nil && s.sessions.Available(ctx)
}

// SafeRedirect keeps post-login redirects on our own origin: a local path
// starting with a single "/", else "/".
func SafeRedirect(raw string) string {
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || len(raw) > 2048 {
		return "/"
	}
	for _, r := range raw {
		if r == '\\' || r < 0x20 || r == 0x7f {
			return "/"
		}
	}
	return raw
}

func (s *AuthService) BeginLogin(ctx context.Context, redirect string) (string, LoginState, error) {
	if !s.Available(ctx) {
		return "", LoginState{}, ErrAuthUnavailable
	}
	state, err := randomToken()
	if err != nil {
		return "", LoginState{}, err
	}
	st := LoginState{State: state, Verifier: oauth2.GenerateVerifier(), Redirect: SafeRedirect(redirect)}
	return s.google.AuthCodeURL(st.State, st.Verifier), st, nil
}

// CompleteLogin returns the user and a fresh raw session token.
func (s *AuthService) CompleteLogin(ctx context.Context, code, state string, st LoginState) (*model.User, string, error) {
	if s.google == nil {
		return nil, "", ErrAuthUnavailable
	}
	if st.State == "" || code == "" || subtle.ConstantTimeCompare([]byte(state), []byte(st.State)) != 1 {
		return nil, "", ErrAuthStateMismatch
	}
	p, err := s.google.FetchProfile(ctx, code, st.Verifier)
	if err != nil {
		return nil, "", err
	}
	name := p.Name
	if name == "" {
		name = p.Email
	}
	var avatar *string
	if p.Picture != "" {
		avatar = &p.Picture
	}
	u, err := s.users.UpsertByProvider(ctx, repository.UpsertUserParams{
		Provider:    constants.AuthProviderGoogle,
		ProviderSub: p.Sub,
		Email:       p.Email,
		Name:        name,
		AvatarURL:   avatar,
	})
	if err != nil {
		return nil, "", err
	}
	if u.Status != constants.UserStatusActive {
		return u, "", ErrDeactivated
	}
	token, err := s.sessions.Create(ctx, u.ID)
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}

// UserFromSession resolves a session token to an active user. Sessions of
// missing or deactivated users are revoked on sight.
func (s *AuthService) UserFromSession(ctx context.Context, token string) (*model.User, error) {
	userID, err := s.sessions.Lookup(ctx, token)
	if err != nil {
		return nil, ErrNoSession
	}
	u, err := s.users.GetByID(ctx, userID)
	if errors.Is(err, repository.ErrUserNotFound) || (err == nil && u.Status != constants.UserStatusActive) {
		_ = s.sessions.DeleteHash(ctx, repository.HashSessionToken(token), userID)
		return nil, ErrNoSession
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *AuthService) Me(u *model.User) model.MeResponse {
	return model.MeResponse{ID: u.ID, Name: u.Name, Email: u.Email, AvatarURL: u.AvatarURL}
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.sessions.Delete(ctx, token)
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
