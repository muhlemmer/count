// Package tester provides a testing framework for the complete project.
package tester

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/zerologadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/muhlemmer/count/internal/db/migrations"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/rs/zerolog"
)

// Database configuration
const (
	MigrDriverEnvKey  = "MIGRATION_DRIVER"
	DefaultMigrDriver = "pgx"
	DSNEnvKey         = "DB_URL"
	DefaultDSN        = "postgresql://muhlemmer@db:5432/muhlemmer?sslmode=disable"
)

// Resources caries data for testing.
type Resources struct {
	CTX    context.Context
	ErrCTX context.Context

	DSN  string
	Pool *pgxpool.Pool

	RequestBegin     time.Time
	RequestsEnd      time.Time
	DailyTotalsBegin time.Time
	DailyTotalsEnd   time.Time

	MethodIDs []pgtype.Int8
}

func (r *Resources) methodsData(ctx context.Context) {
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

	err := r.Pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		sd, err := tx.Prepare(ctx, "MethodsData", insertMethodsSQL)
		if err != nil {
			return err
		}

		for _, method := range methods {
			for _, path := range paths {
				var dst pgtype.Int8

				err = tx.QueryRow(ctx, sd.Name,
					pgtype.Varchar{
						String: method.String(),
						Status: pgtype.Present,
					},
					pgtype.Varchar{
						String: path,
						Status: pgtype.Present,
					},
				).Scan(&dst)
				if err != nil {
					return err
				}

				r.MethodIDs = append(r.MethodIDs, dst)
			}
		}
		return nil
	})

	zerolog.Ctx(ctx).Err(err).Int("inserted", len(r.MethodIDs)).Msg("tester methods data insert")
	if err != nil {
		panic(fmt.Errorf("tester MethodsData: %w", err))
	}
}

func (r *Resources) requestsData(ctx context.Context, multiplier int) {
	step := r.RequestsEnd.Sub(r.RequestBegin) / time.Duration(len(r.MethodIDs)*multiplier)

	var inserted int64

	err := r.Pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		sd, err := tx.Prepare(ctx, "RequestsData", insertRequestsSQL)
		if err != nil {
			return err
		}

		current := r.RequestBegin

		for i := 0; i < multiplier; i++ {
			for _, mid := range r.MethodIDs {
				ct, err := tx.Exec(ctx, sd.Name,
					mid,
					pgtype.Timestamptz{
						Time:   current,
						Status: pgtype.Present,
					},
				)
				if err != nil {
					return err
				}

				if ct.Insert() {
					inserted += ct.RowsAffected()
				}
				current = current.Add(step)
			}
		}
		return nil
	})

	zerolog.Ctx(ctx).Err(err).Int64("inserted", inserted).Msg("tester requests data insert")
	if err != nil {
		panic(fmt.Errorf("tester RequestsData: %w", err))
	}
}

func (r *Resources) dailyMethodTotalsData(ctx context.Context) {
	source := rand.New(rand.NewSource(2))
	var inserted int64

	err := r.Pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		sd, err := tx.Prepare(ctx, "DailyMethodTotalsData", insertDailyMethodTotalsSQL)
		if err != nil {
			return err
		}

		for current := r.DailyTotalsBegin; current.Before(r.DailyTotalsEnd); current = current.Add(24 * time.Hour) {
			for _, mid := range r.MethodIDs {
				ct, err := tx.Exec(ctx, sd.Name,
					pgtype.Date{
						Time:   current,
						Status: pgtype.Present,
					},
					mid,
					pgtype.Int8{
						Int:    source.Int63n(1000),
						Status: pgtype.Present,
					},
				)
				if err != nil {
					return err
				}

				if ct.Insert() {
					inserted += ct.RowsAffected()
				}

			}

		}

		return nil
	})

	zerolog.Ctx(ctx).Err(err).Int64("inserted", inserted).Msg("tester daily method totals data insert")
	if err != nil {
		panic(fmt.Errorf("tester DailyMethodTotalsData: %w", err))
	}
}

// Run resets the database by migrating Down and Up.
//
// The run function is meant to iniate tests and supply
// them with Resources as required, returning the value
// from testing.M.Run().
//
// Database configuration is taken from the environment.
// See package constants for more details.
func Run(timeout time.Duration, run func(r *Resources) int) int {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, NoColor: true}).With().Timestamp().Logger()
	ctx = logger.WithContext(ctx)
	errCTX, cancel := context.WithCancel(ctx)
	cancel()

	migrDriver, ok := os.LookupEnv(MigrDriverEnvKey)
	if !ok {
		migrDriver = DefaultMigrDriver
	}
	dsn, ok := os.LookupEnv(DSNEnvKey)
	if !ok {
		dsn = DefaultDSN
	}

	migrDSN := strings.Replace(dsn, "postgresql", migrDriver, 1)

	migrations.Down(migrDSN)
	migrations.Up(migrDSN)

	conf, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		panic(err)
	}

	conf.ConnConfig.Logger = zerologadapter.NewLogger(logger)

	db, err := pgxpool.ConnectConfig(ctx, conf)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	r := &Resources{
		CTX:    ctx,
		ErrCTX: errCTX,
		DSN:    dsn,
		Pool:   db,
	}

	return run(r)
}

// RunWithData resets the database by migrating Down and Up and generating
// testdata in all tables.
// The limits of the generated data can be found in Resources,
// passed to the run function.
// The run function is meant to iniate tests and supply
// them with Resources as required, returning the value
// from testing.M.Run().
//
// Database configuration is taken from the environment.
// See package constants for more details.
func RunWithData(timeout time.Duration, run func(r *Resources) int) int {
	return Run(timeout, func(r *Resources) int {
		r.RequestBegin = time.Date(2022, time.October, 16, 0, 0, 0, 0, time.UTC)
		r.RequestsEnd = time.Date(2022, time.October, 18, 0, 0, 0, -1, time.UTC)
		r.DailyTotalsBegin = time.Date(1986, time.March, 15, 0, 0, 0, 0, time.UTC)
		r.DailyTotalsEnd = time.Date(1986, time.April, 16, 0, 0, 0, -1, time.UTC)

		r.methodsData(r.CTX)
		r.requestsData(r.CTX, 100)
		r.dailyMethodTotalsData(r.CTX)

		return run(r)
	})
}
