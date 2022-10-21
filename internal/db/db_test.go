package db

import (
	"context"
	_ "embed"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/log/zerologadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/muhlemmer/count/internal/db/migrations"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/muhlemmer/count/pkg/date"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
)

var (
	testCTX context.Context
	errCTX  context.Context
	testDB  *DB
)

const dsn = "postgresql://muhlemmer@db:5432/muhlemmer?sslmode=disable"

func testMain(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, NoColor: true}).With().Timestamp().Logger()
	testCTX = logger.WithContext(ctx)
	errCTX, cancel = context.WithCancel(testCTX)
	cancel()

	migrDSN := strings.Replace(dsn, "postgresql", "pgx", 1)

	migrations.Down(migrDSN)
	migrations.Up(migrDSN)

	conf, err := pgxpool.ParseConfig(dsn)
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
			args: args{testCTX, dsn},
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

func TestDB_CountDailyMethodTotals(t *testing.T) {
	begin := time.Unix(1666000000, 0)
	end := begin.Add(24 * time.Hour)

	err := testDB.InsertMethodRequestTestdata(testCTX, 1000, begin, end)
	if err != nil {
		t.Fatal(err)
	}

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
				{Method: countv1.Method_POST, Path: "/users", Count: 52, Date: date.Date(begin)},
				{Method: countv1.Method_POST, Path: "/items", Count: 47, Date: date.Date(begin)},
				{Method: countv1.Method_DELETE, Path: "/actions", Count: 48, Date: date.Date(begin)},
				{Method: countv1.Method_GRPC, Path: "/actions", Count: 52, Date: date.Date(begin)},
				{Method: countv1.Method_GET, Path: "/items", Count: 41, Date: date.Date(begin)},
				{Method: countv1.Method_GRPC, Path: "/users", Count: 44, Date: date.Date(begin)},
				{Method: countv1.Method_GET, Path: "/users", Count: 51, Date: date.Date(begin)},
				{Method: countv1.Method_DELETE, Path: "/users", Count: 57, Date: date.Date(begin)},
				{Method: countv1.Method_GET, Path: "/actions", Count: 35, Date: date.Date(begin)},
				{Method: countv1.Method_GRPC, Path: "/items", Count: 47, Date: date.Date(begin)},
				{Method: countv1.Method_DELETE, Path: "/items", Count: 52, Date: date.Date(begin)},
				{Method: countv1.Method_POST, Path: "/actions", Count: 54, Date: date.Date(begin)},
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

			got, err := testDB.CountDailyMethodTotals(tt.args.ctx, tt.args.day)
			if (err != nil) != tt.wantErr {
				t.Errorf("DB.CountDailyMethodTotals() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for _, msg := range got {
				t.Log(msg)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("DB.CountDailyMethodTotals() =\n%v\nwant\n%v", got, tt.want)
			}
			for i, want := range tt.want {
				if !proto.Equal(got[i], want) {
					t.Errorf("DB.CountDailyMethodTotals() #%d =\n%v\nwant\n%v", i, got[i], want)
				}
			}
		})
	}
}

func TestDB_ListDailyTotals(t *testing.T) {
	var (
		begin = time.Date(1986, 3, 25, 0, 0, 0, 0, time.UTC)
		end   = begin.Add(10 * 24 * time.Hour)
		day1  = begin.Add(24 * time.Hour)
		day2  = begin.Add(48 * time.Hour)
	)

	err := testDB.InsertDailyTotalsTestdata(testCTX, 5000, begin, end)
	if err != nil {
		t.Fatal(err)
	}

	results := []*countv1.MethodCount{
		{Date: date.Date(day1), Path: "/actions", Method: countv1.Method_DELETE, Count: 52},
		{Date: date.Date(day1), Path: "/actions", Method: countv1.Method_GET, Count: 27},
		{Date: date.Date(day1), Path: "/actions", Method: countv1.Method_GRPC, Count: 41},
		{Date: date.Date(day1), Path: "/actions", Method: countv1.Method_POST, Count: 33},
		{Date: date.Date(day1), Path: "/items", Method: countv1.Method_DELETE, Count: 51},
		{Date: date.Date(day1), Path: "/items", Method: countv1.Method_GET, Count: 48},
		{Date: date.Date(day1), Path: "/items", Method: countv1.Method_GRPC, Count: 35},
		{Date: date.Date(day1), Path: "/items", Method: countv1.Method_POST, Count: 35},
		{Date: date.Date(day1), Path: "/users", Method: countv1.Method_DELETE, Count: 48},
		{Date: date.Date(day1), Path: "/users", Method: countv1.Method_GET, Count: 45},
		{Date: date.Date(day1), Path: "/users", Method: countv1.Method_GRPC, Count: 27},
		{Date: date.Date(day1), Path: "/users", Method: countv1.Method_POST, Count: 37},
		{Date: date.Date(day2), Path: "/actions", Method: countv1.Method_DELETE, Count: 42},
		{Date: date.Date(day2), Path: "/actions", Method: countv1.Method_GET, Count: 30},
		{Date: date.Date(day2), Path: "/actions", Method: countv1.Method_GRPC, Count: 42},
		{Date: date.Date(day2), Path: "/actions", Method: countv1.Method_POST, Count: 44},
		{Date: date.Date(day2), Path: "/items", Method: countv1.Method_DELETE, Count: 40},
		{Date: date.Date(day2), Path: "/items", Method: countv1.Method_GET, Count: 41},
		{Date: date.Date(day2), Path: "/items", Method: countv1.Method_GRPC, Count: 39},
		{Date: date.Date(day2), Path: "/items", Method: countv1.Method_POST, Count: 35},
		{Date: date.Date(day2), Path: "/users", Method: countv1.Method_DELETE, Count: 32},
		{Date: date.Date(day2), Path: "/users", Method: countv1.Method_GET, Count: 50},
		{Date: date.Date(day2), Path: "/users", Method: countv1.Method_GRPC, Count: 39},
		{Date: date.Date(day2), Path: "/users", Method: countv1.Method_POST, Count: 35},
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
			for _, msg := range got {
				t.Log(msg)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("DB.ListDailyTotals() =\n%v\nwant\n%v", got, tt.want)
			}
			for i, want := range tt.want {
				if !proto.Equal(got[i], want) {
					t.Errorf("DB.ListDailyTotals() #%d =\n%v\nwant\n%v", i, got[i], want)
				}
			}
		})
	}
}
