package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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
SELECT
	username,
	name,
	created_at,
	groupUniqArrayMerge(prothesis_ids_state) AS prothesis_ids,
	countMerge(signals_count_state) AS signals_count
FROM aggregated_data
WHERE username = ?
GROUP BY username, name, created_at`

	var report handlers.Report
	var createdAt time.Time
	var prothesisIDs stringArray
	var signalsCount uint64
	if err := s.db.QueryRowContext(ctx, query, username).Scan(
		&report.Username,
		&report.Name,
		&createdAt,
		&prothesisIDs,
		&signalsCount,
	); errors.Is(err, sql.ErrNoRows) {
		return handlers.Report{}, handlers.ErrReportNotFound
	} else if err != nil {
		return handlers.Report{}, err
	}

	report.CreatedAt = &createdAt
	report.ProthesisIDs = []string(prothesisIDs)
	report.SignalsCount = int64(signalsCount)
	return report, nil
}

type stringArray []string

func (a *stringArray) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*a = nil
	case []string:
		*a = append((*a)[:0], v...)
	case []byte:
		*a = parseArrayString(string(v))
	case string:
		*a = parseArrayString(v)
	default:
		return fmt.Errorf("scan string array: unsupported type %T", src)
	}
	return nil
}

func parseArrayString(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "{}[]")
	if value == "" {
		return []string{}
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"'`)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
