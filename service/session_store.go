package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strconv"
	"sync"
	"time"

	infraredis "github.com/austin72905/go-infra/redis"
	goredis "github.com/redis/go-redis/v9"
)

const (
	SessionKindUser  = "user"
	SessionKindAdmin = "admin"
)

var ErrSessionNotFound = errors.New("session not found")

type SessionStore interface {
	Create(ctx context.Context, kind string, subjectID int64, ttl time.Duration) (string, error)
	Get(ctx context.Context, kind, token string) (int64, error)
	Delete(ctx context.Context, kind, token string) error
}

type MemorySessionStore struct {
	mu       sync.Mutex
	sessions map[string]memorySession
}

type memorySession struct {
	SubjectID int64
	ExpiresAt time.Time
}

func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: make(map[string]memorySession),
	}
}

func (s *MemorySessionStore) Create(ctx context.Context, kind string, subjectID int64, ttl time.Duration) (string, error) {
	token, err := generateSessionToken()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionKey(kind, token)] = memorySession{
		SubjectID: subjectID,
		ExpiresAt: time.Now().Add(ttl),
	}

	return token, nil
}

func (s *MemorySessionStore) Get(ctx context.Context, kind, token string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := sessionKey(kind, token)
	session, ok := s.sessions[key]
	if !ok {
		return 0, ErrSessionNotFound
	}
	if !session.ExpiresAt.After(time.Now()) {
		delete(s.sessions, key)
		return 0, ErrSessionNotFound
	}

	return session.SubjectID, nil
}

func (s *MemorySessionStore) Delete(ctx context.Context, kind, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionKey(kind, token))
	return nil
}

type RedisSessionStore struct {
	client *goredis.Client
}

func NewRedisSessionStore(client *infraredis.Client) *RedisSessionStore {
	return &RedisSessionStore{client: client.Raw()}
}

func (s *RedisSessionStore) Create(ctx context.Context, kind string, subjectID int64, ttl time.Duration) (string, error) {
	token, err := generateSessionToken()
	if err != nil {
		return "", err
	}

	if err := s.client.Set(ctx, sessionKey(kind, token), strconv.FormatInt(subjectID, 10), ttl).Err(); err != nil {
		return "", err
	}

	return token, nil
}

func (s *RedisSessionStore) Get(ctx context.Context, kind, token string) (int64, error) {
	value, err := s.client.Get(ctx, sessionKey(kind, token)).Result()
	if err == goredis.Nil {
		return 0, ErrSessionNotFound
	}
	if err != nil {
		return 0, err
	}

	subjectID, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, ErrSessionNotFound
	}

	return subjectID, nil
}

func (s *RedisSessionStore) Delete(ctx context.Context, kind, token string) error {
	return s.client.Del(ctx, sessionKey(kind, token)).Err()
}

func generateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func sessionKey(kind, token string) string {
	return "session:" + kind + ":" + token
}
