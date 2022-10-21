package service

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/muhlemmer/count/internal/db"
	"github.com/muhlemmer/count/internal/db/migrations"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/muhlemmer/count/pkg/date"
	"github.com/rs/zerolog"
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
			args:    args{errCTX, &countv1.CountDailyTotalsRequest{Date: date.Date(begin)}},
			wantErr: true,
		},
		{
			name:    "empty date",
			args:    args{testCTX, &countv1.CountDailyTotalsRequest{}},
			wantErr: true,
		},
		{
			name: "success",
			args: args{testCTX, &countv1.CountDailyTotalsRequest{Date: date.Date(begin)}},
			want: &countv1.CountDailyTotalsResponse{
				MethodCounts: []*countv1.MethodCount{
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

func TestCountServer_ListDailyTotals(t *testing.T) {
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
				StartDate: date.Date(day1),
			}},
			wantErr: true,
		},
		{
			name: "empty EndDate",
			args: args{testCTX, &countv1.ListDailyTotalsRequest{
				EndDate: date.Date(day2),
			}},
			wantErr: true,
		},
		{
			name: "context error",
			args: args{errCTX, &countv1.ListDailyTotalsRequest{
				StartDate: date.Date(day1),
				EndDate:   date.Date(day2),
			}},
			wantErr: true,
		},
		{
			name: "not found",
			args: args{testCTX, &countv1.ListDailyTotalsRequest{
				StartDate: date.Today(),
				EndDate:   date.Today(),
			}},
			wantErr: true,
		},
		{
			name: "success",
			args: args{testCTX, &countv1.ListDailyTotalsRequest{
				StartDate: date.Date(day1),
				EndDate:   date.Date(day2),
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
