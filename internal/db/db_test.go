package db

import (
	"context"
	_ "embed"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/muhlemmer/count/internal/db/migrations"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/rs/zerolog"
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
	//hi

	migrDSN := strings.Replace(dsn, "postgresql", "cockroachdb", 1)

	migrations.Down(migrDSN)
	migrations.Up(migrDSN)

	db, err := pgxpool.Connect(testCTX, dsn)
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
