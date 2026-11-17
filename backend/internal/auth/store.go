// Package auth handles accounts and sign-in.
//
// How a session works:
//  1. On login we create 32 random bytes (the token) and send them to the
//     browser in an httpOnly cookie. Page JavaScript can't read that cookie.
//  2. The database stores only the SHA-256 hash of the token, so someone who
//     reads the sessions table still can't use the values as cookies.
//  3. On each protected request we hash the cookie again and look it up.
//  4. Logout deletes the row, so the cookie stops working immediately
//     (unlike a JWT, which stays valid until it expires).
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sessionLifetime = 30 * 24 * time.Hour

var (
	ErrEmailTaken = errors.New("email already registered")
	ErrNoSession  = errors.New("no valid session")
)

// User is what the API returns about an account. The password hash never
// leaves this package.
type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store { return &Store{db: db} }

// CreateUser inserts a user. The email column is citext (case-insensitive),
// so "Demo@Eneba.test" and "demo@eneba.test" count as the same address.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash, displayName string) (User, error) {
	var u User
	err := s.db.QueryRow(ctx, `
        INSERT INTO users (email, password_hash, display_name)
        VALUES ($1, $2, $3)
        RETURNING id, email, display_name, created_at`,
		email, passwordHash, displayName,
	).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation on email
		return u, ErrEmailTaken
	}
	if err != nil {
		return u, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// UserByEmail returns the user and their password hash, for login.
// found is false when no account has that email.
func (s *Store) UserByEmail(ctx context.Context, email string) (u User, passwordHash string, found bool, err error) {
	err = s.db.QueryRow(ctx, `
        SELECT id, email, display_name, created_at, password_hash
        FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, "", false, nil
	}
	if err != nil {
		return u, "", false, fmt.Errorf("user by email: %w", err)
	}
	return u, passwordHash, true, nil
}

// CreateSession stores a new session for userID and returns the raw token
// (for the cookie) and its expiry. Old expired sessions of the same user are
// cleaned up at the same time.
func (s *Store) CreateSession(ctx context.Context, userID string) (token string, expires time.Time, err error) {
	raw := make([]byte, 32) // 256 random bits: impossible to guess
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, fmt.Errorf("random token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	expires = time.Now().Add(sessionLifetime)

	_, err = s.db.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1 AND expires_at < now()`, userID)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("clean sessions: %w", err)
	}
	_, err = s.db.Exec(ctx, `INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		hashToken(token), userID, expires)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("create session: %w", err)
	}
	return token, expires, nil
}

// UserBySession returns the signed-in user for a cookie token, or
// ErrNoSession if the token is unknown or expired.
func (s *Store) UserBySession(ctx context.Context, token string) (User, error) {
	var u User
	err := s.db.QueryRow(ctx, `
        SELECT u.id, u.email, u.display_name, u.created_at
        FROM sessions s
        JOIN users u ON u.id = s.user_id
        WHERE s.token_hash = $1 AND s.expires_at > now()`, hashToken(token),
	).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNoSession
	}
	if err != nil {
		return u, fmt.Errorf("user by session: %w", err)
	}
	return u, nil
}

// DeleteSession ends a session (logout). Unknown tokens are not an error.
func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hashToken(token))
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
