package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

var ErrGoogleProfile = errors.New("incomplete google profile")

const googleUserinfoURL = "https://openidconnect.googleapis.com/v1/userinfo"

type GoogleProfile struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

type GoogleRepository struct {
	conf   *oauth2.Config
	client *http.Client
}

func NewGoogleRepository(clientID, clientSecret, callbackURL string) *GoogleRepository {
	return &GoogleRepository{
		conf: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  callbackURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://accounts.google.com/o/oauth2/auth",
				TokenURL:  "https://oauth2.googleapis.com/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (r *GoogleRepository) AuthCodeURL(state, verifier string) string {
	return r.conf.AuthCodeURL(state,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)
}

func (r *GoogleRepository) FetchProfile(ctx context.Context, code, verifier string) (*GoogleProfile, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, r.client)
	tok, err := r.conf.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("google token exchange: %w", err)
	}
	res, err := r.conf.Client(ctx, tok).Get(googleUserinfoURL)
	if err != nil {
		return nil, fmt.Errorf("google userinfo: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo: status %d", res.StatusCode)
	}
	var p GoogleProfile
	if err := json.NewDecoder(res.Body).Decode(&p); err != nil {
		return nil, fmt.Errorf("google userinfo decode: %w", err)
	}
	if p.Sub == "" || p.Email == "" {
		return nil, ErrGoogleProfile
	}
	return &p, nil
}
