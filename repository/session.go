package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/dta32/bandung-coffeeshop-be/cache"
	"github.com/redis/go-redis/v9"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionStore    = errors.New("session store unavailable")
)

// Fixed lifetime from login; activity never extends it.
const SessionTTL = 30 * 24 * time.Hour

type SessionRepository struct {
	cache *cache.Redis
}

func NewSessionRepository(redisCache *cache.Redis) *SessionRepository {
	return &SessionRepository{cache: redisCache}
}

func HashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func sessionKey(hash string) string { return "session:" + hash }

func userSessionsKey(userID int) string { return "user_sessions:" + strconv.Itoa(userID) }

func (r *SessionRepository) Available(ctx context.Context) bool {
	return r.cache != nil && r.cache.Ping(ctx) == nil
}

// Create stores only the sha256 of the token and returns the raw token for
// the cookie.
func (r *SessionRepository) Create(ctx context.Context, userID int) (string, error) {
	if r.cache == nil {
		return "", ErrSessionStore
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	hash := HashSessionToken(token)
	err := r.cache.Pipelined(ctx, func(p redis.Pipeliner) error {
		p.Set(ctx, sessionKey(hash), userID, SessionTTL)
		p.SAdd(ctx, userSessionsKey(userID), hash)
		p.Expire(ctx, userSessionsKey(userID), SessionTTL)
		return nil
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

func (r *SessionRepository) Lookup(ctx context.Context, token string) (int, error) {
	if r.cache == nil {
		return 0, ErrSessionStore
	}
	b, err := r.cache.Get(ctx, sessionKey(HashSessionToken(token)))
	if errors.Is(err, cache.ErrMiss) {
		return 0, ErrSessionNotFound
	}
	if err != nil {
		return 0, err
	}
	id, err := strconv.Atoi(string(b))
	if err != nil {
		return 0, ErrSessionNotFound
	}
	return id, nil
}

func (r *SessionRepository) Delete(ctx context.Context, token string) error {
	hash := HashSessionToken(token)
	userID, err := r.Lookup(ctx, token)
	if errors.Is(err, ErrSessionNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return r.DeleteHash(ctx, hash, userID)
}

func (r *SessionRepository) DeleteHash(ctx context.Context, hash string, userID int) error {
	if r.cache == nil {
		return ErrSessionStore
	}
	return r.cache.Pipelined(ctx, func(p redis.Pipeliner) error {
		p.Del(ctx, sessionKey(hash))
		p.SRem(ctx, userSessionsKey(userID), hash)
		return nil
	})
}
