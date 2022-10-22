package service

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/muhlemmer/count/internal/db"
	"github.com/muhlemmer/count/internal/db/migrations"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/muhlemmer/count/pkg/datepb"
	"github.com/rs/zerolog"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	testCTX context.Context
	errCTX  context.Context
	testDB  *db.DB
	errDB   *db.DB
)

const (
	dsn = "postgresql://muhlemmer@db:5432/muhlemmer?sslmode=disable"
)

func testMain(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, NoColor: true}).With().Timestamp().Logger()
	testCTX = logger.WithContext(ctx)
	errCTX, cancel = context.WithCancel(testCTX)
	cancel()

	migrDSN := strings.Replace(dsn, "postgresql", "cockroachdb", 1)

	migrations.Down(migrDSN)
	migrations.Up(migrDSN)

	var err error
	testDB, err = db.New(testCTX, dsn)
	if err != nil {
		panic(err)
	}
	defer testDB.Close()

	errDB, err = db.New(testCTX, dsn)
	if err != nil {
		panic(err)
	}
	errDB.Close()

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func TestNewCountService(t *testing.T) {
	NewCountService(grpc.NewServer(), testDB)
}

type mockAddServer struct {
	grpc.ServerStream

	ctx     context.Context
	stream  []*countv1.AddRequest
	pos     int
	recvErr error
	sendErr error
}

func (s *mockAddServer) SendAndClose(*countv1.AddResponse) error { return s.sendErr }

func (s *mockAddServer) Recv() (*countv1.AddRequest, error) {
	if s.recvErr != nil {
		return nil, s.recvErr
	}

	if s.pos >= len(s.stream) {
		return nil, io.EOF
	}

	req := s.stream[s.pos]
	s.pos++

	return req, nil
}

func (s *mockAddServer) Context() context.Context {
	return s.ctx
}

var testStream = []*countv1.AddRequest{
	{
		Method:           countv1.Method_GET,
		Path:             "/foo/bar",
		RequestTimestamp: timestamppb.New(time.Unix(123, 0)),
	},
	{
		Method:           countv1.Method_POST,
		Path:             "/items/new",
		RequestTimestamp: timestamppb.New(time.Unix(456, 0)),
	},
	{
		Method:           countv1.Method_PUT,
		Path:             "/items/update",
		RequestTimestamp: timestamppb.New(time.Unix(789, 0)),
	},
}

func TestCountServer_Add(t *testing.T) {
	shortCTX, cancel := context.WithTimeout(testCTX, time.Second)
	defer cancel()

	type fields struct {
		db *db.DB
	}
	tests := []struct {
		name    string
		fields  fields
		args    countv1.CountService_AddServer
		wantErr bool
	}{
		{
			name:   "context error",
			fields: fields{testDB},
			args: &mockAddServer{
				ctx:    errCTX,
				stream: testStream,
			},
			wantErr: true,
		},
		{
			name:   "db error",
			fields: fields{errDB},
			args: &mockAddServer{
				ctx:    shortCTX,
				stream: testStream,
			},
			wantErr: true,
		},
		{
			name:   "recv error",
			fields: fields{testDB},
			args: &mockAddServer{
				ctx:     testCTX,
				stream:  testStream,
				recvErr: errors.New("foobars"),
			},
			wantErr: true,
		},
		{
			name:   "send error",
			fields: fields{testDB},
			args: &mockAddServer{
				ctx: testCTX,
				stream: []*countv1.AddRequest{
					{
						Method:           countv1.Method_GET,
						Path:             "/foo/bar",
						RequestTimestamp: timestamppb.New(time.Unix(123, 0)),
					},
					{
						Method:           countv1.Method_POST,
						Path:             "/items/new",
						RequestTimestamp: timestamppb.New(time.Unix(456, 0)),
					},
					{
						Method:           countv1.Method_PUT,
						Path:             "/items/update",
						RequestTimestamp: timestamppb.New(time.Unix(789, 0)),
					},
				},
				sendErr: errors.New("foobars"),
			},
			wantErr: true,
		},
		{
			name:   "success",
			fields: fields{testDB},
			args: &mockAddServer{
				ctx: testCTX,
				stream: []*countv1.AddRequest{
					{
						Method:           countv1.Method_GET,
						Path:             "/foo/bar",
						RequestTimestamp: timestamppb.New(time.Unix(123, 0)),
					},
					{
						Method:           countv1.Method_POST,
						Path:             "/items/new",
						RequestTimestamp: timestamppb.New(time.Unix(456, 0)),
					},
					{
						Method:           countv1.Method_PUT,
						Path:             "/items/update",
						RequestTimestamp: timestamppb.New(time.Unix(789, 0)),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &CountServer{
				db: tt.fields.db,
			}
			if err := s.Add(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("CountServer.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCountServer_CountDailyTotals(t *testing.T) {
	begin := time.Unix(1666000000, 0)
	end := begin.Add(24 * time.Hour)

	err := testDB.InsertMethodRequestTestdata(testCTX, 1000, begin, end)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx context.Context
		req *countv1.CountDailyTotalsRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *countv1.CountDailyTotalsResponse
		wantErr bool
	}{
		{
			name:    "context error",
			args:    args{errCTX, &countv1.CountDailyTotalsRequest{Date: datepb.Date(begin)}},
			wantErr: true,
		},
		{
			name:    "empty date",
			args:    args{testCTX, &countv1.CountDailyTotalsRequest{}},
			wantErr: true,
		},
		{
			name: "success",
			args: args{testCTX, &countv1.CountDailyTotalsRequest{Date: datepb.Date(begin)}},
			want: &countv1.CountDailyTotalsResponse{
				MethodCounts: []*countv1.MethodCount{
					{Method: countv1.Method_POST, Path: "/users", Count: 52, Date: datepb.Date(begin)},
					{Method: countv1.Method_POST, Path: "/items", Count: 47, Date: datepb.Date(begin)},
					{Method: countv1.Method_DELETE, Path: "/actions", Count: 48, Date: datepb.Date(begin)},
					{Method: countv1.Method_GRPC, Path: "/actions", Count: 52, Date: datepb.Date(begin)},
					{Method: countv1.Method_GET, Path: "/items", Count: 41, Date: datepb.Date(begin)},
					{Method: countv1.Method_GRPC, Path: "/users", Count: 44, Date: datepb.Date(begin)},
					{Method: countv1.Method_GET, Path: "/users", Count: 51, Date: datepb.Date(begin)},
					{Method: countv1.Method_DELETE, Path: "/users", Count: 57, Date: datepb.Date(begin)},
					{Method: countv1.Method_GET, Path: "/actions", Count: 35, Date: datepb.Date(begin)},
					{Method: countv1.Method_GRPC, Path: "/items", Count: 47, Date: datepb.Date(begin)},
					{Method: countv1.Method_DELETE, Path: "/items", Count: 52, Date: datepb.Date(begin)},
					{Method: countv1.Method_POST, Path: "/actions", Count: 54, Date: datepb.Date(begin)},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &CountServer{
				db: testDB,
			}
			got, err := s.CountDailyTotals(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CountServer.CountDailyTotals() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !proto.Equal(got, tt.want) {
				t.Errorf("CountServer.CountDailyTotals() =\n%v\nwant\n%v", got, tt.want)
			}
		})
	}
}

var (
	dailyTotalsBegin    = time.Date(1986, 3, 25, 0, 0, 0, 0, time.UTC)
	dailyTotalsEnd      = dailyTotalsBegin.Add(10 * 24 * time.Hour)
	dailyTotalsTestdata sync.Once
)

func TestCountServer_ListDailyTotals(t *testing.T) {
	var (
		day1 = dailyTotalsBegin.Add(24 * time.Hour)
		day2 = dailyTotalsBegin.Add(48 * time.Hour)
	)

	dailyTotalsTestdata.Do(func() {
		err := testDB.InsertDailyTotalsTestdata(testCTX, 5000, dailyTotalsBegin, dailyTotalsEnd)
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
		ctx context.Context
		req *countv1.ListDailyTotalsRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *countv1.ListDailyTotalsResponse
		wantErr bool
	}{
		{
			name:    "empty req",
			args:    args{testCTX, nil},
			wantErr: true,
		},
		{
			name: "empty from",
			args: args{testCTX, &countv1.ListDailyTotalsRequest{
				StartDate: datepb.Date(day1),
			}},
			wantErr: true,
		},
		{
			name: "empty EndDate",
			args: args{testCTX, &countv1.ListDailyTotalsRequest{
				EndDate: datepb.Date(day2),
			}},
			wantErr: true,
		},
		{
			name: "context error",
			args: args{errCTX, &countv1.ListDailyTotalsRequest{
				StartDate: datepb.Date(day1),
				EndDate:   datepb.Date(day2),
			}},
			wantErr: true,
		},
		{
			name: "not found",
			args: args{testCTX, &countv1.ListDailyTotalsRequest{
				StartDate: datepb.Today(),
				EndDate:   datepb.Today(),
			}},
			wantErr: true,
		},
		{
			name: "success",
			args: args{testCTX, &countv1.ListDailyTotalsRequest{
				StartDate: datepb.Date(day1),
				EndDate:   datepb.Date(day2),
			}},
			want: &countv1.ListDailyTotalsResponse{
				MethodCounts: results,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &CountServer{
				db: testDB,
			}
			got, err := s.ListDailyTotals(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CountServer.ListDailyTotals() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !proto.Equal(got, tt.want) {
				t.Errorf("CountServer.ListDailyTotals() = \n%v\nwant\n%v", got, tt.want)
			}
		})
	}
}

func TestCountServer_GetPeriodTotals(t *testing.T) {
	dailyTotalsTestdata.Do(func() {
		err := testDB.InsertDailyTotalsTestdata(testCTX, 5000, dailyTotalsBegin, dailyTotalsEnd)
		if err != nil {
			t.Fatal(err)
		}
	})

	type args struct {
		ctx context.Context
		req *countv1.GetPeriodTotalsRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *countv1.GetPeriodTotalsResponse
		wantErr bool
	}{
		{
			name:    "missing period",
			args:    args{testCTX, &countv1.GetPeriodTotalsRequest{}},
			wantErr: true,
		},
		{
			name: "context error",
			args: args{errCTX, &countv1.GetPeriodTotalsRequest{Period: &date.Date{
				Year: 1986,
			}}},
			wantErr: true,
		},
		{
			name: "not found",
			args: args{testCTX, &countv1.GetPeriodTotalsRequest{Period: &date.Date{
				Year: 1977,
			}}},
			wantErr: true,
		},
		{
			name: "year",
			args: args{testCTX, &countv1.GetPeriodTotalsRequest{Period: &date.Date{
				Year: 1986,
			}}},
			want: &countv1.GetPeriodTotalsResponse{
				MethodCounts: []*countv1.MethodCount{
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
		},
		{
			name: "month",
			args: args{testCTX, &countv1.GetPeriodTotalsRequest{Period: &date.Date{
				Year:  1986,
				Month: 3,
			}}},
			want: &countv1.GetPeriodTotalsResponse{
				MethodCounts: []*countv1.MethodCount{
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
		},
		{
			name: "day",
			args: args{testCTX, &countv1.GetPeriodTotalsRequest{Period: &date.Date{
				Year:  1986,
				Month: 3,
				Day:   25,
			}}},
			want: &countv1.GetPeriodTotalsResponse{
				MethodCounts: []*countv1.MethodCount{
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &CountServer{
				db: testDB,
			}
			got, err := s.GetPeriodTotals(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CountServer.GetPeriodTotals() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !proto.Equal(got, tt.want) {
				t.Errorf("CountServer.GetPeriodTotals() =\n%v\nwant\n%v", got, tt.want)
			}
		})
	}
}
