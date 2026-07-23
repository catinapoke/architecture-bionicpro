package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"

	"bionicpro-report/internal/handlers"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) ReportByUsername(ctx context.Context, username string) (handlers.Report, error) {
	const query = `
SELECT username, name, created_at, prothesis_ids, signals_count
FROM aggregated_data
WHERE username = $1`

	var report handlers.Report
	var createdAt time.Time
	if err := s.db.QueryRowContext(ctx, query, username).Scan(
		&report.Username,
		&report.Name,
		&createdAt,
		pq.Array(&report.ProthesisIDs),
		&report.SignalsCount,
	); errors.Is(err, sql.ErrNoRows) {
		return handlers.Report{}, handlers.ErrReportNotFound
	} else if err != nil {
		return handlers.Report{}, err
	}

	report.CreatedAt = &createdAt
	return report, nil
}
