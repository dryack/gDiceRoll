package user

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/dryack/gDiceRoll/core/crypto"
	"github.com/jackc/pgx/v4/pgxpool"
)

type UserType string

const (
	Guest UserType = "guest"
	User  UserType = "user"
	Admin UserType = "admin"
)

type UserStruct struct {
	ID               int64
	Username         string
	PasswordHash     string
	UserType         UserType
	TwoFactorSecret  sql.NullString
	TwoFactorEnabled bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type UserManager struct {
	db *pgxpool.Pool
}

func NewUserManager(db *pgxpool.Pool) *UserManager {
	return &UserManager{db: db}
}

func (um *UserManager) CreateUser(ctx context.Context, username, password string, userType UserType) (*UserStruct, error) {
	params := &crypto.Params{
		Memory:      256 * 1024,
		Iterations:  5,
		Parallelism: 2,
		SaltLength:  32,
		KeyLength:   32,
	}

	passwordHash, err := crypto.GenerateFromPassword(password, params)
	if err != nil {
		return nil, err
	}

	var user UserStruct
	err = um.db.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, user_type, two_factor_enabled)
         VALUES ($1, $2, $3, $4)
         RETURNING id, created_at, updated_at`,
		username, passwordHash, userType, false).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	user.Username = username
	user.UserType = userType
	user.TwoFactorEnabled = false

	return &user, nil
}

func (um *UserManager) GetUserByUsername(ctx context.Context, username string) (*UserStruct, error) {
	var user UserStruct
	err := um.db.QueryRow(ctx,
		`SELECT id, username, password_hash, user_type, two_factor_secret, two_factor_enabled, created_at, updated_at
         FROM users WHERE username = $1`,
		username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.UserType,
		&user.TwoFactorSecret, &user.TwoFactorEnabled, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func (um *UserManager) VerifyPassword(user *UserStruct, password string) (bool, error) {
	return crypto.ComparePasswordAndHash(password, user.PasswordHash)
}

func (um *UserManager) CreateInitialAdminUser(ctx context.Context) error {
	_, err := um.GetUserByUsername(ctx, "admin")
	if err == nil {
		// Admin user already exists
		return nil
	}

	_, err = um.CreateUser(ctx, "admin", "12345trust-albion-shocking", Admin)
	return err
}
