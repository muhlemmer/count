package db

import (
	"context"
	_ "embed"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/muhlemmer/count/internal/tester"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/muhlemmer/count/pkg/datepb"
	"google.golang.org/protobuf/proto"
)

var (
	R      *tester.Resources
	testDB *DB
)

func TestMain(m *testing.M) {
	os.Exit(
		tester.Run(5*time.Minute, func(r *tester.Resources) int {
			R = r
			testDB = &DB{pool: r.Pool}
			return m.Run()
		}),
	)
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
			args:    args{R.CTX, "foo"},
			wantErr: true,
		},
		{
			name: "succes",
			args: args{R.CTX, R.DSN},
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
			args:    args{R.ErrCTX, "select $1::int;", []interface{}{1}},
			wantErr: true,
		},
		/*
			{
				name:    "repeated error",
				args:    args{R.CTX, "foo $1::int;", []interface{}{1}},
				wantErr: true,
			},
		*/
		{
			name: "success",
			args: args{R.CTX, "select $1::int;", []interface{}{1}},
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
			args:    args{R.ErrCTX, countv1.Method_GET, "/foo/bar", time.Now()},
			wantErr: true,
		},
		{
			name: "succes",
			args: args{R.CTX, countv1.Method_GET, "/foo/bar", time.Now()},
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
	//pick a spot in the middle
	date := R.RequestBegin.Add(24 * time.Hour)

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
			args:    args{R.ErrCTX, date},
			wantErr: true,
		},
		{
			name: "success",
			args: args{R.CTX, date},
			want: []*countv1.MethodCount{
				{Method: countv1.Method_DELETE, Path: "/actions", Count: 50, Date: datepb.Date(date)},
				{Method: countv1.Method_GET, Path: "/actions", Count: 50, Date: datepb.Date(date)},
				{Method: countv1.Method_GRPC, Path: "/actions", Count: 50, Date: datepb.Date(date)},
				{Method: countv1.Method_POST, Path: "/actions", Count: 50, Date: datepb.Date(date)},
				{Method: countv1.Method_DELETE, Path: "/items", Count: 50, Date: datepb.Date(date)},
				{Method: countv1.Method_GET, Path: "/items", Count: 50, Date: datepb.Date(date)},
				{Method: countv1.Method_GRPC, Path: "/items", Count: 50, Date: datepb.Date(date)},
				{Method: countv1.Method_POST, Path: "/items", Count: 50, Date: datepb.Date(date)},
				{Method: countv1.Method_DELETE, Path: "/users", Count: 50, Date: datepb.Date(date)},
				{Method: countv1.Method_GET, Path: "/users", Count: 49, Date: datepb.Date(date)},
				{Method: countv1.Method_GRPC, Path: "/users", Count: 50, Date: datepb.Date(date)},
				{Method: countv1.Method_POST, Path: "/users", Count: 50, Date: datepb.Date(date)},
			},
		},
		{
			name:    "conflict",
			args:    args{R.CTX, date},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "conflict" {
				err := testDB.InsertMethodRequest(R.CTX, countv1.Method_POST, "/items", date)
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

func TestDB_ListDailyTotals(t *testing.T) {
	var (
		day1 = R.DailyTotalsBegin.Add(24 * time.Hour)
		day2 = R.DailyTotalsBegin.Add(48 * time.Hour)
	)

	results := []*countv1.MethodCount{
		{Date: datepb.Date(day1), Path: "/actions", Method: countv1.Method_DELETE, Count: 217},
		{Date: datepb.Date(day1), Path: "/actions", Method: countv1.Method_GET, Count: 2},
		{Date: datepb.Date(day1), Path: "/actions", Method: countv1.Method_GRPC, Count: 510},
		{Date: datepb.Date(day1), Path: "/actions", Method: countv1.Method_POST, Count: 818},
		{Date: datepb.Date(day1), Path: "/items", Method: countv1.Method_DELETE, Count: 43},
		{Date: datepb.Date(day1), Path: "/items", Method: countv1.Method_GET, Count: 211},
		{Date: datepb.Date(day1), Path: "/items", Method: countv1.Method_GRPC, Count: 820},
		{Date: datepb.Date(day1), Path: "/items", Method: countv1.Method_POST, Count: 740},
		{Date: datepb.Date(day1), Path: "/users", Method: countv1.Method_DELETE, Count: 146},
		{Date: datepb.Date(day1), Path: "/users", Method: countv1.Method_GET, Count: 358},
		{Date: datepb.Date(day1), Path: "/users", Method: countv1.Method_GRPC, Count: 409},
		{Date: datepb.Date(day1), Path: "/users", Method: countv1.Method_POST, Count: 933},
		{Date: datepb.Date(day2), Path: "/actions", Method: countv1.Method_DELETE, Count: 185},
		{Date: datepb.Date(day2), Path: "/actions", Method: countv1.Method_GET, Count: 563},
		{Date: datepb.Date(day2), Path: "/actions", Method: countv1.Method_GRPC, Count: 404},
		{Date: datepb.Date(day2), Path: "/actions", Method: countv1.Method_POST, Count: 813},
		{Date: datepb.Date(day2), Path: "/items", Method: countv1.Method_DELETE, Count: 464},
		{Date: datepb.Date(day2), Path: "/items", Method: countv1.Method_GET, Count: 589},
		{Date: datepb.Date(day2), Path: "/items", Method: countv1.Method_GRPC, Count: 365},
		{Date: datepb.Date(day2), Path: "/items", Method: countv1.Method_POST, Count: 849},
		{Date: datepb.Date(day2), Path: "/users", Method: countv1.Method_DELETE, Count: 159},
		{Date: datepb.Date(day2), Path: "/users", Method: countv1.Method_GET, Count: 49},
		{Date: datepb.Date(day2), Path: "/users", Method: countv1.Method_GRPC, Count: 542},
		{Date: datepb.Date(day2), Path: "/users", Method: countv1.Method_POST, Count: 176},
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
			args:    args{R.ErrCTX, day1, day2},
			wantErr: true,
		},
		{
			name: "two days interval",
			args: args{R.CTX, day1, day2},
			want: results,
		},
		{
			name: "single day",
			args: args{R.CTX, day2, day2},
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
			args: args{R.ErrCTX,
				time.Date(1986, time.March, 1, 0, 0, 0, 0, time.UTC),
				time.Date(1986, time.March, 31, 23, 59, 59, 999999999, time.UTC),
			},
			wantErr: true,
		},
		{
			name: "year",
			args: args{R.CTX,
				time.Date(1986, time.January, 1, 0, 0, 0, 0, time.UTC),
				time.Date(1986, time.December, 31, 23, 59, 59, 999999999, time.UTC),
			},
			want: []*countv1.MethodCount{
				{Path: "/actions", Method: countv1.Method_DELETE, Count: 12721},
				{Path: "/actions", Method: countv1.Method_GET, Count: 19719},
				{Path: "/actions", Method: countv1.Method_GRPC, Count: 16596},
				{Path: "/actions", Method: countv1.Method_POST, Count: 16864},
				{Path: "/items", Method: countv1.Method_DELETE, Count: 15655},
				{Path: "/items", Method: countv1.Method_GET, Count: 18839},
				{Path: "/items", Method: countv1.Method_GRPC, Count: 15354},
				{Path: "/items", Method: countv1.Method_POST, Count: 14384},
				{Path: "/users", Method: countv1.Method_DELETE, Count: 15916},
				{Path: "/users", Method: countv1.Method_GET, Count: 14854},
				{Path: "/users", Method: countv1.Method_GRPC, Count: 14394},
				{Path: "/users", Method: countv1.Method_POST, Count: 16563},
			},
		},
		{
			name: "month",
			args: args{R.CTX,
				time.Date(1986, time.March, 1, 0, 0, 0, 0, time.UTC),
				time.Date(1986, time.March, 31, 23, 59, 59, 999999999, time.UTC),
			},
			want: []*countv1.MethodCount{
				{Path: "/actions", Method: countv1.Method_DELETE, Count: 6762},
				{Path: "/actions", Method: countv1.Method_GET, Count: 11360},
				{Path: "/actions", Method: countv1.Method_GRPC, Count: 9233},
				{Path: "/actions", Method: countv1.Method_POST, Count: 8438},
				{Path: "/items", Method: countv1.Method_DELETE, Count: 7759},
				{Path: "/items", Method: countv1.Method_GET, Count: 8490},
				{Path: "/items", Method: countv1.Method_GRPC, Count: 7744},
				{Path: "/items", Method: countv1.Method_POST, Count: 7931},
				{Path: "/users", Method: countv1.Method_DELETE, Count: 7220},
				{Path: "/users", Method: countv1.Method_GET, Count: 8138},
				{Path: "/users", Method: countv1.Method_GRPC, Count: 9961},
				{Path: "/users", Method: countv1.Method_POST, Count: 9510},
			},
		},
		{
			name: "day",
			args: args{R.CTX,
				time.Date(1986, time.March, 25, 0, 0, 0, 0, time.UTC),
				time.Date(1986, time.March, 25, 23, 59, 59, 999999999, time.UTC),
			},
			want: []*countv1.MethodCount{
				{Path: "/actions", Method: countv1.Method_DELETE, Count: 46},
				{Path: "/actions", Method: countv1.Method_GET, Count: 565},
				{Path: "/actions", Method: countv1.Method_GRPC, Count: 725},
				{Path: "/actions", Method: countv1.Method_POST, Count: 211},
				{Path: "/items", Method: countv1.Method_DELETE, Count: 756},
				{Path: "/items", Method: countv1.Method_GET, Count: 853},
				{Path: "/items", Method: countv1.Method_GRPC, Count: 210},
				{Path: "/items", Method: countv1.Method_POST, Count: 107},
				{Path: "/users", Method: countv1.Method_DELETE, Count: 627},
				{Path: "/users", Method: countv1.Method_GET, Count: 473},
				{Path: "/users", Method: countv1.Method_GRPC, Count: 856},
				{Path: "/users", Method: countv1.Method_POST, Count: 288},
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
