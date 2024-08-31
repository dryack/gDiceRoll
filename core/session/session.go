package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/redis/go-redis/v9"
)

type SessionManager struct {
	cache *redis.Client
	db    *pgxpool.Pool
}

type Session struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewSessionManager(cache *redis.Client, db *pgxpool.Pool) *SessionManager {
	return &SessionManager{
		cache: cache,
		db:    db,
	}
}

func (sm *SessionManager) CreateSession(ctx context.Context, userID int64) (*Session, error) {
	if sm.cache == nil && sm.db == nil {
		return nil, fmt.Errorf("both cache and database are unavailable")
	}

	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)

	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}

	// Try to store in cache first
	if sm.cache != nil {
		err := sm.storeSessionInCache(ctx, session)
		if err == nil {
			return session, nil
		}
		// Log the cache error, but continue to try database
		fmt.Printf("Failed to store session in cache: %v\n", err)
	}

	// If cache fails or is not available, store in database
	if sm.db != nil {
		err := sm.storeSessionInDB(ctx, session)
		if err != nil {
			return nil, fmt.Errorf("failed to store session in database: %v", err)
		}
		return session, nil
	}

	return nil, fmt.Errorf("failed to store session")
}

func (sm *SessionManager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	// Try to get from cache first
	session, err := sm.getSessionFromCache(ctx, sessionID)
	if err == nil {
		return session, nil
	}

	// If not in cache, try database
	session, err = sm.getSessionFromDB(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %v", err)
	}

	// If found in database, update cache
	_ = sm.storeSessionInCache(ctx, session)

	return session, nil
}

func (sm *SessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	// Delete from cache
	if err := sm.cache.Del(ctx, sessionID).Err(); err != nil {
		// Log error but continue to delete from database
		fmt.Printf("Error deleting session from cache: %v\n", err)
	}

	// Delete from database
	_, err := sm.db.Exec(ctx, "DELETE FROM sessions WHERE id = $1", sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session from database: %v", err)
	}

	return nil
}

func (sm *SessionManager) storeSessionInCache(ctx context.Context, session *Session) error {
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return sm.cache.Set(ctx, session.ID, string(sessionJSON), time.Until(session.ExpiresAt)).Err()
}

func (sm *SessionManager) storeSessionInDB(ctx context.Context, session *Session) error {
	_, err := sm.db.Exec(ctx,
		"INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)",
		session.ID, session.UserID, session.ExpiresAt)
	return err
}

func (sm *SessionManager) getSessionFromCache(ctx context.Context, sessionID string) (*Session, error) {
	sessionJSON, err := sm.cache.Get(ctx, sessionID).Result()
	if err != nil {
		return nil, err
	}

	var session Session
	err = json.Unmarshal([]byte(sessionJSON), &session)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (sm *SessionManager) getSessionFromDB(ctx context.Context, sessionID string) (*Session, error) {
	var session Session
	err := sm.db.QueryRow(ctx,
		"SELECT id, user_id, expires_at FROM sessions WHERE id = $1",
		sessionID).Scan(&session.ID, &session.UserID, &session.ExpiresAt)
	if err != nil {
		return nil, err
	}

	return &session, nil
}
