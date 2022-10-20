package db

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgtype"
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

//go:embed queries/insert_method_request.sql
var insertMethodRequestSQL string

// InsertMethodRequest inserts a request for a certain method and path.
// Inserts are retried untill the operation succeeds without error
// or when the passed context expires.
func (db *DB) InsertMethodRequest(ctx context.Context, method countv1.Method, path string, requestTS time.Time) error {
	return statusError(
		db.execRetry(ctx, time.Second, 10*time.Second, insertMethodRequestSQL, method.String(), path, requestTS),
		"insert method request",
	)

}

// InsertMethodRequestTestdata generates pseudo-random entries in the count.requests table.
// The generated data is deterministic for a given amount, begin and end values.
// This function is mainly used for unit testing.
func (db *DB) InsertMethodRequestTestdata(ctx context.Context, amount int, begin, end time.Time) error {
	source := rand.New(rand.NewSource(22)) // for deterministic output

	var (
		methods = []countv1.Method{
			countv1.Method_GET,
			countv1.Method_POST,
			countv1.Method_DELETE,
			countv1.Method_GRPC,
		}
		paths = []string{
			"/users",
			"/items",
			"/actions",
		}
	)

	beginN := begin.UnixNano()
	endN := end.UnixNano()

	for i := 0; i < amount; i++ {
		ts := time.Unix(0, source.Int63n(endN-beginN)+beginN)
		method := methods[int(source.Int63n(int64(len(methods))))]
		path := paths[int(source.Int63n(int64(len(paths))))]

		err := func() error {
			ctx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()

			return db.InsertMethodRequest(ctx, method, path, ts)
		}()
		zerolog.Ctx(ctx).Err(err).Stringer("method", method).Str("path", path).Time("ts", ts).Msg("insert method request")

		if err != nil {
			return err
		}
	}
	return nil
}

//go:embed queries/count_daily_method_totals.sql
var countDailyMethodTotalsSQL string

// CountDailyMethodTotals deletes entries from count.requests for the given day.
// Deleted entries are counted for each method and path pair and inserted in the
// count.daily_method_totals table.
// The resulting count enties are returned.
func (db *DB) CountDailyMethodTotals(ctx context.Context, day time.Time) (results []*countv1.MethodCount, err error) {
	const errDesc = "count daily method totals"

	rows, err := db.pool.Query(ctx, countDailyMethodTotalsSQL, pgtype.Date{
		Time:   day,
		Status: pgtype.Present,
	})
	if err = statusError(err, errDesc); err != nil {
		return nil, err
	}
	defer func() {
		rows.Close()
		if rerr := rows.Err(); rerr != nil && err == nil {
			err = statusError(rerr, errDesc)
		}
	}()

	for rows.Next() {
		var (
			method pgtype.Varchar
			path   pgtype.Varchar
			total  pgtype.Int4
		)

		if err = statusError(rows.Scan(&method, &path, &total), errDesc); err != nil {
			return nil, err
		}

		results = append(results, &countv1.MethodCount{
			Method: countv1.Method(countv1.Method_value[method.String]),
			Path:   path.String,
			Count:  total.Int,
		})
	}

	return results, nil
}
