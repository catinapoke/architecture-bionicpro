package profile

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type Profile struct {
	Subject   string
	Username  string
	Email     string
	Name      string
	Issuer    string
	Provider  string
	RawClaims string
	UpdatedAt time.Time
}

type Store struct {
	db *sql.DB
}

func NewStore(databaseURL string) (*Store, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database url is empty")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open profile db: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping profile db: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Upsert(ctx context.Context, p Profile) error {
	if p.Subject == "" {
		return fmt.Errorf("profile subject is required")
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = time.Now().UTC()
	}

	_, err := s.db.ExecContext(ctx, `
INSERT INTO user_profiles (subject, username, email, name, issuer, provider, raw_claims, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (subject) DO UPDATE SET
  username = EXCLUDED.username,
  email = EXCLUDED.email,
  name = EXCLUDED.name,
  issuer = EXCLUDED.issuer,
  provider = EXCLUDED.provider,
  raw_claims = EXCLUDED.raw_claims,
  updated_at = EXCLUDED.updated_at;
`, p.Subject, p.Username, p.Email, p.Name, p.Issuer, p.Provider, p.RawClaims, p.UpdatedAt.UTC())
	if err != nil {
		return fmt.Errorf("upsert profile: %w", err)
	}
	return nil
}

func (s *Store) GetBySubject(ctx context.Context, subject string) (Profile, bool, error) {
	var p Profile
	err := s.db.QueryRowContext(ctx, `
SELECT subject, username, email, name, issuer, provider, raw_claims, updated_at
FROM user_profiles WHERE subject = $1`, subject).Scan(
		&p.Subject, &p.Username, &p.Email, &p.Name, &p.Issuer, &p.Provider, &p.RawClaims, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return Profile{}, false, nil
	}
	if err != nil {
		return Profile{}, false, fmt.Errorf("get profile: %w", err)
	}
	return p, true, nil
}
