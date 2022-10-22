package db

import (
	"context"
	_ "embed"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/log/zerologadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/muhlemmer/count/internal/db/migrations"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/muhlemmer/count/pkg/datepb"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
)

var (
	testCTX context.Context
	errCTX  context.Context
	testDSN string
	testDB  *DB
)

const (
	defaultMigrDriver = "pgx"
	defaultDSN        = "postgresql://muhlemmer@db:5432/muhlemmer?sslmode=disable"
)

func testMain(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, NoColor: true}).With().Timestamp().Logger()
	testCTX = logger.WithContext(ctx)
	errCTX, cancel = context.WithCancel(testCTX)
	cancel()

	migrDriver, ok := os.LookupEnv("MIGRATION_DRIVER")
	if !ok {
		migrDriver = defaultMigrDriver
	}
	testDSN, ok = os.LookupEnv("DB_URL")
	if !ok {
		testDSN = defaultDSN
	}

	migrDSN := strings.Replace(testDSN, "postgresql", migrDriver, 1)

	migrations.Down(migrDSN)
	migrations.Up(migrDSN)

	conf, err := pgxpool.ParseConfig(testDSN)
	if err != nil {
		panic(err)
	}

	conf.ConnConfig.Logger = zerologadapter.NewLogger(logger)

	db, err := pgxpool.ConnectConfig(testCTX, conf)
	if err != nil {
		panic(err)
	}

	testDB = &DB{pool: db}

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func Test_multiError_Error(t *testing.T) {
	const want = "multiple errors: foo, bar"
	errs := multiError{errors.New("foo"), errors.New("bar")}

	if got := errs.Error(); got != want {
		t.Errorf("multiError.Error() = %s, want %s", got, want)
	}
}

func TestNew(t *testing.T) {
	type args struct {
		ctx context.Context
		dsn string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "dsn error",
			args:    args{testCTX, "foo"},
			wantErr: true,
		},
		{
			name: "succes",
			args: args{testCTX, testDSN},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.ctx, tt.args.dsn)
			if got != nil {
				defer got.Close()
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got != nil) != tt.want {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDB_execRetry(t *testing.T) {
	type args struct {
		ctx  context.Context
		sql  string
		args []interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "context error",
			args:    args{errCTX, "select $1::int;", []interface{}{1}},
			wantErr: true,
		},
		{
			name:    "repeated error",
			args:    args{testCTX, "foo $1::int;", []interface{}{1}},
			wantErr: true,
		},
		{
			name: "success",
			args: args{testCTX, "select $1::int;", []interface{}{1}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(tt.args.ctx, time.Second)
			defer cancel()

			if err := testDB.execRetry(ctx, time.Microsecond, time.Second/10, tt.args.sql, tt.args.args...); (err != nil) != tt.wantErr {
				t.Errorf("DB.execRetry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDB_InsertMethodRequest(t *testing.T) {
	type args struct {
		ctx       context.Context
		method    countv1.Method
		path      string
		requestTS time.Time
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "context error",
			args:    args{errCTX, countv1.Method_GET, "/foo/bar", time.Now()},
			wantErr: true,
		},
		{
			name: "succes",
			args: args{testCTX, countv1.Method_GET, "/foo/bar", time.Now()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := testDB.InsertMethodRequest(tt.args.ctx, tt.args.method, tt.args.path, tt.args.requestTS); (err != nil) != tt.wantErr {
				t.Errorf("DB.InsertMethodRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func compareMethodCounts(t *testing.T, fname string, got, wants []*countv1.MethodCount) {
	for _, msg := range got {
		t.Log(msg)
	}
	if len(got) != len(wants) {
		t.Fatalf("%s =\n%v\nwant\n%v", fname, got, wants)
	}
	for i, want := range wants {
		if !proto.Equal(got[i], want) {
			t.Errorf("%s #%d =\n%v\nwant\n%v", fname, i, got[i], want)
		}
	}
}

func TestDB_CountDailyMethodTotals(t *testing.T) {
	begin := time.Unix(1666000000, 0)
	end := begin.Add(24 * time.Hour)

	func() {
		ctx, cancel := context.WithTimeout(testCTX, 3*time.Minute/2)
		defer cancel()

		err := testDB.InsertMethodRequestTestdata(ctx, 1000, begin, end)
		if err != nil {
			t.Fatal(err)
		}
	}()

	type args struct {
		ctx context.Context
		day time.Time
	}
	tests := []struct {
		name    string
		args    args
		want    []*countv1.MethodCount
		wantErr bool
	}{
		{
			name:    "context error",
			args:    args{errCTX, begin},
			wantErr: true,
		},
		{
			name: "success",
			args: args{testCTX, begin},
			want: []*countv1.MethodCount{
				{Method: countv1.Method_DELETE, Path: "/actions", Count: 48, Date: datepb.Date(begin)},
				{Method: countv1.Method_GET, Path: "/actions", Count: 35, Date: datepb.Date(begin)},
				{Method: countv1.Method_GRPC, Path: "/actions", Count: 52, Date: datepb.Date(begin)},
				{Method: countv1.Method_POST, Path: "/actions", Count: 54, Date: datepb.Date(begin)},
				{Method: countv1.Method_DELETE, Path: "/items", Count: 52, Date: datepb.Date(begin)},
				{Method: countv1.Method_GET, Path: "/items", Count: 41, Date: datepb.Date(begin)},
				{Method: countv1.Method_GRPC, Path: "/items", Count: 47, Date: datepb.Date(begin)},
				{Method: countv1.Method_POST, Path: "/items", Count: 47, Date: datepb.Date(begin)},
				{Method: countv1.Method_DELETE, Path: "/users", Count: 57, Date: datepb.Date(begin)},
				{Method: countv1.Method_GET, Path: "/users", Count: 51, Date: datepb.Date(begin)},
				{Method: countv1.Method_GRPC, Path: "/users", Count: 44, Date: datepb.Date(begin)},
				{Method: countv1.Method_POST, Path: "/users", Count: 52, Date: datepb.Date(begin)},
			},
		},
		{
			name:    "conflict",
			args:    args{testCTX, begin},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "conflict" {
				err := testDB.InsertMethodRequest(testCTX, countv1.Method_POST, "/items", begin)
				if err != nil {
					t.Fatal(err)
				}
			}

			begin, end := datepb.Interval(datepb.Date(tt.args.day))
			got, err := testDB.CountDailyMethodTotals(tt.args.ctx, begin, end)
			if (err != nil) != tt.wantErr {
				t.Errorf("DB.CountDailyMethodTotals() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			compareMethodCounts(t, "DB.CountDailyMethodTotals()", got, tt.want)
		})
	}
}

var (
	dailyTotalsBegin    = time.Date(1986, 3, 25, 0, 0, 0, 0, time.UTC)
	dailyTotalsEnd      = dailyTotalsBegin.Add(10 * 24 * time.Hour)
	dailyTotalsTestdata sync.Once
)

func TestDB_ListDailyTotals(t *testing.T) {
	var (
		day1 = dailyTotalsBegin.Add(24 * time.Hour)
		day2 = dailyTotalsBegin.Add(48 * time.Hour)
	)

	dailyTotalsTestdata.Do(func() {
		ctx, cancel := context.WithTimeout(testCTX, 12*time.Minute)
		defer cancel()

		err := testDB.InsertDailyTotalsTestdata(ctx, 5000, dailyTotalsBegin, dailyTotalsEnd)
		if err != nil {
			t.Fatal(err)
		}
	})

	results := []*countv1.MethodCount{
		{Date: datepb.Date(day1), Path: "/actions", Method: countv1.Method_DELETE, Count: 52},
		{Date: datepb.Date(day1), Path: "/actions", Method: countv1.Method_GET, Count: 27},
		{Date: datepb.Date(day1), Path: "/actions", Method: countv1.Method_GRPC, Count: 41},
		{Date: datepb.Date(day1), Path: "/actions", Method: countv1.Method_POST, Count: 33},
		{Date: datepb.Date(day1), Path: "/items", Method: countv1.Method_DELETE, Count: 51},
		{Date: datepb.Date(day1), Path: "/items", Method: countv1.Method_GET, Count: 48},
		{Date: datepb.Date(day1), Path: "/items", Method: countv1.Method_GRPC, Count: 35},
		{Date: datepb.Date(day1), Path: "/items", Method: countv1.Method_POST, Count: 35},
		{Date: datepb.Date(day1), Path: "/users", Method: countv1.Method_DELETE, Count: 48},
		{Date: datepb.Date(day1), Path: "/users", Method: countv1.Method_GET, Count: 45},
		{Date: datepb.Date(day1), Path: "/users", Method: countv1.Method_GRPC, Count: 27},
		{Date: datepb.Date(day1), Path: "/users", Method: countv1.Method_POST, Count: 37},
		{Date: datepb.Date(day2), Path: "/actions", Method: countv1.Method_DELETE, Count: 42},
		{Date: datepb.Date(day2), Path: "/actions", Method: countv1.Method_GET, Count: 30},
		{Date: datepb.Date(day2), Path: "/actions", Method: countv1.Method_GRPC, Count: 42},
		{Date: datepb.Date(day2), Path: "/actions", Method: countv1.Method_POST, Count: 44},
		{Date: datepb.Date(day2), Path: "/items", Method: countv1.Method_DELETE, Count: 40},
		{Date: datepb.Date(day2), Path: "/items", Method: countv1.Method_GET, Count: 41},
		{Date: datepb.Date(day2), Path: "/items", Method: countv1.Method_GRPC, Count: 39},
		{Date: datepb.Date(day2), Path: "/items", Method: countv1.Method_POST, Count: 35},
		{Date: datepb.Date(day2), Path: "/users", Method: countv1.Method_DELETE, Count: 32},
		{Date: datepb.Date(day2), Path: "/users", Method: countv1.Method_GET, Count: 50},
		{Date: datepb.Date(day2), Path: "/users", Method: countv1.Method_GRPC, Count: 39},
		{Date: datepb.Date(day2), Path: "/users", Method: countv1.Method_POST, Count: 35},
	}

	type args struct {
		ctx  context.Context
		from time.Time
		till time.Time
	}
	tests := []struct {
		name    string
		args    args
		want    []*countv1.MethodCount
		wantErr bool
	}{
		{
			name:    "context error",
			args:    args{errCTX, day1, day2},
			wantErr: true,
		},
		{
			name: "two days interval",
			args: args{testCTX, day1, day2},
			want: results,
		},
		{
			name: "single day",
			args: args{testCTX, day2, day2},
			want: results[12:],
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := testDB.ListDailyTotals(tt.args.ctx, tt.args.from, tt.args.till)
			if (err != nil) != tt.wantErr {
				t.Errorf("DB.ListDailyTotals() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			compareMethodCounts(t, "DB.ListDailyTotals()", got, tt.want)
		})
	}
}

func TestDB_GetPeriodTotals(t *testing.T) {
	dailyTotalsTestdata.Do(func() {
		ctx, cancel := context.WithTimeout(testCTX, 12*time.Minute)
		defer cancel()

		err := testDB.InsertDailyTotalsTestdata(ctx, 5000, dailyTotalsBegin, dailyTotalsEnd)
		if err != nil {
			t.Fatal(err)
		}
	})

	type args struct {
		ctx   context.Context
		start time.Time
		end   time.Time
	}
	tests := []struct {
		name    string
		args    args
		want    []*countv1.MethodCount
		wantErr bool
	}{
		{
			name: "context error",
			args: args{errCTX,
				time.Date(1986, time.March, 1, 0, 0, 0, 0, time.UTC),
				time.Date(1986, time.March, 31, 23, 59, 59, 999999999, time.UTC),
			},
			wantErr: true,
		},
		{
			name: "year",
			args: args{testCTX,
				time.Date(1986, time.January, 1, 0, 0, 0, 0, time.UTC),
				time.Date(1986, time.December, 31, 23, 59, 59, 999999999, time.UTC),
			},
			want: []*countv1.MethodCount{
				{Path: "/actions", Method: countv1.Method_DELETE, Count: 441},
				{Path: "/actions", Method: countv1.Method_GET, Count: 397},
				{Path: "/actions", Method: countv1.Method_GRPC, Count: 401},
				{Path: "/actions", Method: countv1.Method_POST, Count: 403},
				{Path: "/items", Method: countv1.Method_DELETE, Count: 422},
				{Path: "/items", Method: countv1.Method_GET, Count: 424},
				{Path: "/items", Method: countv1.Method_GRPC, Count: 428},
				{Path: "/items", Method: countv1.Method_POST, Count: 425},
				{Path: "/users", Method: countv1.Method_DELETE, Count: 428},
				{Path: "/users", Method: countv1.Method_GET, Count: 448},
				{Path: "/users", Method: countv1.Method_GRPC, Count: 395},
				{Path: "/users", Method: countv1.Method_POST, Count: 388},
			},
		},
		{
			name: "month",
			args: args{testCTX,
				time.Date(1986, time.March, 1, 0, 0, 0, 0, time.UTC),
				time.Date(1986, time.March, 31, 23, 59, 59, 999999999, time.UTC),
			},
			want: []*countv1.MethodCount{
				{Path: "/actions", Method: countv1.Method_DELETE, Count: 309},
				{Path: "/actions", Method: countv1.Method_GET, Count: 267},
				{Path: "/actions", Method: countv1.Method_GRPC, Count: 278},
				{Path: "/actions", Method: countv1.Method_POST, Count: 259},
				{Path: "/items", Method: countv1.Method_DELETE, Count: 311},
				{Path: "/items", Method: countv1.Method_GET, Count: 295},
				{Path: "/items", Method: countv1.Method_GRPC, Count: 304},
				{Path: "/items", Method: countv1.Method_POST, Count: 286},
				{Path: "/users", Method: countv1.Method_DELETE, Count: 290},
				{Path: "/users", Method: countv1.Method_GET, Count: 316},
				{Path: "/users", Method: countv1.Method_GRPC, Count: 264},
				{Path: "/users", Method: countv1.Method_POST, Count: 268},
			},
		},
		{
			name: "day",
			args: args{testCTX,
				time.Date(1986, time.March, 25, 0, 0, 0, 0, time.UTC),
				time.Date(1986, time.March, 25, 23, 59, 59, 999999999, time.UTC),
			},
			want: []*countv1.MethodCount{
				{Path: "/actions", Method: countv1.Method_DELETE, Count: 46},
				{Path: "/actions", Method: countv1.Method_GET, Count: 44},
				{Path: "/actions", Method: countv1.Method_GRPC, Count: 38},
				{Path: "/actions", Method: countv1.Method_POST, Count: 42},
				{Path: "/items", Method: countv1.Method_DELETE, Count: 50},
				{Path: "/items", Method: countv1.Method_GET, Count: 45},
				{Path: "/items", Method: countv1.Method_GRPC, Count: 35},
				{Path: "/items", Method: countv1.Method_POST, Count: 32},
				{Path: "/users", Method: countv1.Method_DELETE, Count: 46},
				{Path: "/users", Method: countv1.Method_GET, Count: 56},
				{Path: "/users", Method: countv1.Method_GRPC, Count: 45},
				{Path: "/users", Method: countv1.Method_POST, Count: 35},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := testDB.GetPeriodTotals(tt.args.ctx, tt.args.start, tt.args.end)
			if (err != nil) != tt.wantErr {
				t.Errorf("DB.GetPeriodTotals() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			compareMethodCounts(t, "DB.GetPeriodTotals()", got, tt.want)
		})
	}
}
