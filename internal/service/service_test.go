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
			args:    args{errCTX, &countv1.CountDailyTotalsRequest{Date: timestamppb.New(begin)}},
			wantErr: true,
		},
		{
			name: "success",
			args: args{testCTX, &countv1.CountDailyTotalsRequest{Date: timestamppb.New(begin)}},
			want: &countv1.CountDailyTotalsResponse{
				MethodCounts: []*countv1.MethodCount{
					{Method: countv1.Method_POST, Path: "/users", Count: 52},
					{Method: countv1.Method_POST, Path: "/items", Count: 47},
					{Method: countv1.Method_DELETE, Path: "/actions", Count: 48},
					{Method: countv1.Method_GRPC, Path: "/actions", Count: 52},
					{Method: countv1.Method_GET, Path: "/items", Count: 41},
					{Method: countv1.Method_GRPC, Path: "/users", Count: 44},
					{Method: countv1.Method_GET, Path: "/users", Count: 51},
					{Method: countv1.Method_DELETE, Path: "/users", Count: 57},
					{Method: countv1.Method_GET, Path: "/actions", Count: 35},
					{Method: countv1.Method_GRPC, Path: "/items", Count: 47},
					{Method: countv1.Method_DELETE, Path: "/items", Count: 52},
					{Method: countv1.Method_POST, Path: "/actions", Count: 54},
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
				t.Errorf("CountServer.CountDailyTotals() = %v, want %v", got, tt.want)
			}
		})
	}
}
