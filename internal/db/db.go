package db

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/muhlemmer/count/internal/timer"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/rs/zerolog"
)

type multiError []error

func (errs multiError) Error() string {
	s := make([]string, len(errs))
	for i, err := range errs {
		s[i] = err.Error()
	}

	return fmt.Sprintf("multiple errors: %s", strings.Join(s, ", "))
}

type DB struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*DB, error) {
	pool, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}

	return &DB{pool}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) execRetry(ctx context.Context, min, max time.Duration, sql string, args ...interface{}) error {
	logger := zerolog.Ctx(ctx).Sample(zerolog.Often)
	var errs multiError

retry:
	for {
		// fail-fast wrapper function for isolated context and cancelation.
		err := func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			_, err := db.pool.Exec(ctx, sql, args...)
			return err
		}(ctx)

		if err == nil {
			return nil
		}

		logger.Err(err).Msg("db exec retry")
		errs = append(errs, err)

		select {
		case <-ctx.Done():
			break retry
		case <-timer.RandomTimer(min, max):
			// retry
		}
	}

	if len(errs) == 1 {
		return errs[0]
	}

	return errs
}

//go:embed queries/insert_method_request.sql
var insertMethodRequestSQL string

// InsertMethodRequest inserts a request for a certain method and path.
// Inserts are retried untill the operation succeeds without error
// or when the passed context expires.
func (db *DB) InsertMethodRequest(ctx context.Context, method countv1.Method, path string, requestTS time.Time) error {
	return db.execRetry(ctx, time.Second, 10*time.Second, insertMethodRequestSQL, method.String(), path, requestTS)
}
