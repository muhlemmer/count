package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/zerologadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/muhlemmer/count/internal/timer"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type multiError []error

func (errs multiError) Error() string {
	s := make([]string, len(errs))
	for i, err := range errs {
		s[i] = err.Error()
	}

	return fmt.Sprintf("multiple errors: %s", strings.Join(s, ", "))
}

func statusError(err error, desc string) error {
	if err == nil {
		return nil
	}

	var code codes.Code

	pge := new(pgconn.PgError)
	if errors.As(err, &pge) {
		switch pge.Code {
		case pgerrcode.UniqueViolation:
			code = codes.AlreadyExists
		default:
			code = codes.Internal
		}

		return status.Errorf(code, "%s: %s", desc, pge.Detail)
	}

	return err
}

// DB provides high level query execution over
// a PGX connection pool.
type DB struct {
	pool *pgxpool.Pool
}

func Wrap(pool *pgxpool.Pool) *DB {
	return &DB{pool: pool}
}

// New configures a new PGX connection pool
// with a zerolog adapter taken from context.
func New(ctx context.Context, dsn string) (*DB, error) {
	conf, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	conf.ConnConfig.Logger = zerologadapter.NewLogger(*zerolog.Ctx(ctx))

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

// InsertMethodRequest inserts a request for a certain method and path.
// Inserts are retried untill the operation succeeds without error
// or when the passed context expires.
func (db *DB) InsertMethodRequest(ctx context.Context, method countv1.Method, path string, requestTS time.Time) error {
	return statusError(
		db.execRetry(ctx, time.Second, 10*time.Second, insertMethodRequestSQL, method.String(), path, requestTS),
		"insert method request",
	)

}

// CountDailyMethodTotals deletes entries from count.requests for the given day.
// Deleted entries are counted for each method and path pair and inserted in the
// count.daily_method_totals table.
// The resulting count enties are returned.
func (db *DB) CountDailyMethodTotals(ctx context.Context, start, end time.Time) ([]*countv1.MethodCount, error) {
	const errDesc = "count daily method totals"

	rows, err := db.pool.Query(ctx, countDailyMethodTotalsSQL,
		pgtype.Timestamptz{
			Time:   start,
			Status: pgtype.Present,
		},
		pgtype.Timestamptz{
			Time:   end,
			Status: pgtype.Present,
		},
	)
	if err = statusError(err, errDesc); err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := scanMethodCountRows(rows)
	return results, statusError(err, errDesc)
}

// dateIntervalQuery is a generalized function for queries that use a start / end date interval.
// the passed query is executed with start and end as arguments.
func (db *DB) dateIntervalQuery(ctx context.Context, query string, start, end time.Time) (pgx.Rows, error) {
	return db.pool.Query(ctx, query,
		pgtype.Date{
			Time:   start,
			Status: pgtype.Present,
		},
		pgtype.Date{
			Time:   end,
			Status: pgtype.Present,
		},
	)
}

// ListDailyTotals selects entries from count.daily_method_totals in the
// date interval of start-end inclusive.
func (db *DB) ListDailyTotals(ctx context.Context, start, end time.Time) ([]*countv1.MethodCount, error) {
	const errDesc = "list daily totals"

	rows, err := db.dateIntervalQuery(ctx, listDailyTotalsSQL, start, end)
	if err = statusError(err, errDesc); err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := scanMethodCountRows(rows)
	return results, statusError(err, errDesc)
}

// GetPeriodTotals selects entries from count.daily_method_totals and
// sums the totals columns, grouped by method and path.
// Start and end times are inclusive.
func (db *DB) GetPeriodTotals(ctx context.Context, start, end time.Time) ([]*countv1.MethodCount, error) {
	const errDesc = "get period totals"

	rows, err := db.dateIntervalQuery(ctx, getPeriodTotalsSQL, start, end)
	if err = statusError(err, errDesc); err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := scanMethodCountRows(rows)
	return results, statusError(err, errDesc)
}
