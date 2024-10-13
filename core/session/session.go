package session

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/redis/go-redis/v9"
	"log"
	"time"
)

type SessionManager struct {
	cache         *redis.Client
	db            *pgxpool.Pool
	accessSecret  []byte
	refreshSecret []byte
}

type Session struct {
	ID           string    `json:"id"`
	UserID       int64     `json:"user_id"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expirecop	s_at"`
}

type JWTClaims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func NewSessionManager(cache *redis.Client, db *pgxpool.Pool, accessSecretHex, refreshSecretHex string) (*SessionManager, error) {
	accessSecret, err := decodeSecret(accessSecretHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode access secret: %v", err)
	}

	refreshSecret, err := decodeSecret(refreshSecretHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode refresh secret: %v", err)
	}

	return &SessionManager{
		cache:         cache,
		db:            db,
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
	}, nil
}

func (sm *SessionManager) CreateSession(ctx context.Context, userID int64) (*Session, string, string, error) {
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)

	refreshToken, err := sm.CreateRefreshToken(userID)
	if err != nil {
		log.Printf("Failed to create refresh token: %v", err)
		return nil, "", "", fmt.Errorf("failed to create refresh token: %v", err)
	}

	accessToken, err := sm.CreateAccessToken(userID)
	if err != nil {
		log.Printf("Failed to create access token: %v", err)
		return nil, "", "", fmt.Errorf("failed to create access token: %v", err)
	}

	session := &Session{
		ID:           sessionID,
		UserID:       userID,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}

	// Store in cache
	if sm.cache != nil {
		err := sm.storeSessionInCache(ctx, session)
		if err != nil {
			log.Printf("Failed to store session in cache: %v", err)
		}
	} else {
		log.Printf("Cache is not initialized")
	}

	// Store in database
	if sm.db != nil {
		err := sm.storeSessionInDB(ctx, session)
		if err != nil {
			log.Printf("Failed to store session in database: %v", err)
			return nil, "", "", fmt.Errorf("failed to store session in database: %v", err)
		}
	} else {
		log.Printf("Database is not initialized")
		return nil, "", "", fmt.Errorf("database is not initialized")
	}

	return session, accessToken, refreshToken, nil
}

func (sm *SessionManager) CreateAccessToken(userID int64) (string, error) {
	expirationTime := time.Now().Add(15 * time.Minute)
	claims := &JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(sm.accessSecret)
}

func (sm *SessionManager) CreateRefreshToken(userID int64) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(sm.refreshSecret)
}

func (sm *SessionManager) VerifyAccessToken(tokenString string) (*JWTClaims, error) {
	return sm.verifyToken(tokenString, sm.accessSecret)
}

func (sm *SessionManager) VerifyRefreshToken(tokenString string) (*JWTClaims, error) {
	// Check if the token is in the blacklist
	if sm.cache != nil {
		exists, err := sm.cache.Exists(context.Background(), "blacklist:"+tokenString).Result()
		if err != nil {
			log.Printf("Error checking refresh token blacklist: %v", err)
		} else if exists == 1 {
			return nil, fmt.Errorf("refresh token has been invalidated")
		}
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return sm.refreshSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

func (sm *SessionManager) verifyToken(tokenString string, secret []byte) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

func (sm *SessionManager) RevokeSession(ctx context.Context, sessionID string) error {
	// Remove from cache
	if sm.cache != nil {
		err := sm.cache.Del(ctx, sessionID).Err()
		if err != nil && err != redis.Nil {
			fmt.Printf("Failed to remove session from cache: %v\n", err)
		}
	}

	// Remove from database
	if sm.db != nil {
		_, err := sm.db.Exec(ctx, "DELETE FROM sessions WHERE id = $1", sessionID)
		if err != nil {
			return fmt.Errorf("failed to remove session from database: %v", err)
		}
	}

	return nil
}

func (sm *SessionManager) storeSessionInCache(ctx context.Context, session *Session) error {
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %v", err)
	}

	// Store the session with an expiration time
	err = sm.cache.Set(ctx, session.ID, string(sessionJSON), time.Until(session.ExpiresAt)).Err()
	if err != nil {
		return fmt.Errorf("failed to store session in cache: %v", err)
	}

	return nil
}

func (sm *SessionManager) storeSessionInDB(ctx context.Context, session *Session) error {
	_, err := sm.db.Exec(ctx,
		`INSERT INTO sessions (id, user_id, refresh_token, expires_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (id) DO UPDATE
		 SET user_id = $2, refresh_token = $3, expires_at = $4`,
		session.ID, session.UserID, session.RefreshToken, session.ExpiresAt)

	if err != nil {
		return fmt.Errorf("failed to store session in database: %v", err)
	}

	return nil
}

func (sm *SessionManager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	// Try to get from cache first
	if sm.cache != nil {
		session, err := sm.getSessionFromCache(ctx, sessionID)
		if err == nil {
			return session, nil
		}
		// If error is not a cache miss, log it
		if err != redis.Nil {
			fmt.Printf("Error retrieving session from cache: %v\n", err)
		}
	}

	// If not in cache or cache is unavailable, try database
	if sm.db != nil {
		return sm.getSessionFromDB(ctx, sessionID)
	}

	return nil, fmt.Errorf("session not found")
}

func (sm *SessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	// Delete from cache
	if sm.cache != nil {
		err := sm.cache.Del(ctx, sessionID).Err()
		if err != nil && err != redis.Nil {
			fmt.Printf("Error deleting session from cache: %v\n", err)
		}
	}

	// Delete from database
	if sm.db != nil {
		_, err := sm.db.Exec(ctx, "DELETE FROM sessions WHERE id = $1", sessionID)
		if err != nil {
			return fmt.Errorf("failed to delete session from database: %v", err)
		}
	}

	return nil
}

func (sm *SessionManager) StartCleanupTask(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err := sm.cleanupExpiredSessions(ctx)
				if err != nil {
					fmt.Printf("Error during session cleanup: %v\n", err)
				}
			}
		}
	}()
}

func (sm *SessionManager) cleanupExpiredSessions(ctx context.Context) error {
	if sm.db == nil {
		return fmt.Errorf("database is not available")
	}

	_, err := sm.db.Exec(ctx, "DELETE FROM sessions WHERE expires_at < NOW()")
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %v", err)
	}

	return nil
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
		"SELECT id, user_id, refresh_token, expires_at FROM sessions WHERE id = $1",
		sessionID).Scan(&session.ID, &session.UserID, &session.RefreshToken, &session.ExpiresAt)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (sm *SessionManager) RefreshSession(ctx context.Context, oldRefreshToken string) (*Session, string, string, error) {
	claims, err := sm.VerifyRefreshToken(oldRefreshToken)
	if err != nil {
		return nil, "", "", fmt.Errorf("invalid refresh token: %v", err)
	}

	// Create a new session
	session, accessToken, newRefreshToken, err := sm.CreateSession(ctx, claims.UserID)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to create new session: %v", err)
	}

	// Invalidate the old refresh token
	err = sm.InvalidateRefreshToken(ctx, oldRefreshToken)
	if err != nil {
		log.Printf("Failed to invalidate old refresh token: %v", err)
		// We don't return here because the new session has been created successfully
	}

	return session, accessToken, newRefreshToken, nil
}

func (sm *SessionManager) InvalidateRefreshToken(ctx context.Context, refreshToken string) error {
	// Parse the token to get the claims
	claims, err := sm.VerifyRefreshToken(refreshToken)
	if err != nil {
		return fmt.Errorf("invalid refresh token: %v", err)
	}

	// Remove the session from the cache
	if sm.cache != nil {
		err = sm.cache.Del(ctx, claims.ID).Err()
		if err != nil && err != redis.Nil {
			log.Printf("Failed to remove session from cache: %v", err)
		}
	}

	// Remove the session from the database
	if sm.db != nil {
		_, err = sm.db.Exec(ctx, "DELETE FROM sessions WHERE refresh_token = $1", refreshToken)
		if err != nil {
			log.Printf("Failed to remove session from database: %v", err)
		}
	}

	// Add the token to a blacklist in the cache
	// The blacklist entry will expire after the token's original expiration time
	if sm.cache != nil {
		expirationTime := time.Unix(claims.ExpiresAt.Unix(), 0)
		err = sm.cache.Set(ctx, "blacklist:"+refreshToken, "1", time.Until(expirationTime)).Err()
		if err != nil {
			log.Printf("Failed to add refresh token to blacklist: %v", err)
		}
	}

	return nil
}
