package service

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/muhlemmer/count/internal/db"
	"github.com/muhlemmer/count/internal/tester"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/muhlemmer/count/pkg/datepb"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	R          *tester.Resources
	testServer *CountServer
)

func TestMain(m *testing.M) {
	os.Exit(
		tester.Run(5*time.Minute, func(r *tester.Resources) int {
			R = r
			testServer = &CountServer{
				db: db.Wrap(R.Pool),
			}
			return m.Run()
		}),
	)
}

func TestNewCountService(t *testing.T) {
	NewCountService(grpc.NewServer(), db.Wrap(R.Pool))
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
	/*
		shortCTX, cancel := context.WithTimeout(R.CTX, time.Second)
		defer cancel()
	*/

	tests := []struct {
		name    string
		args    countv1.CountService_AddServer
		wantErr bool
	}{
		{
			name: "context error",
			args: &mockAddServer{
				ctx:    R.ErrCTX,
				stream: testStream,
			},
			wantErr: true,
		},
		/*
			{
				name:   "db error",
				args: &mockAddServer{
					ctx:    shortCTX,
					stream: testStream,
				},
				wantErr: true,
			},
		*/
		{
			name: "recv error",
			args: &mockAddServer{
				ctx:     R.CTX,
				stream:  testStream,
				recvErr: errors.New("foobars"),
			},
			wantErr: true,
		},
		{
			name: "send error",
			args: &mockAddServer{
				ctx: R.CTX,
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
			name: "success",
			args: &mockAddServer{
				ctx: R.CTX,
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
			if err := testServer.Add(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("CountServer.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCountServer_CountDailyTotals(t *testing.T) {
	//pick a spot in the middle
	date := R.RequestBegin.Add(24 * time.Hour)

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
			args:    args{R.ErrCTX, &countv1.CountDailyTotalsRequest{Date: datepb.Date(date)}},
			wantErr: true,
		},
		{
			name:    "empty date",
			args:    args{R.CTX, &countv1.CountDailyTotalsRequest{}},
			wantErr: true,
		},
		{
			name: "success",
			args: args{R.CTX, &countv1.CountDailyTotalsRequest{Date: datepb.Date(date)}},
			want: &countv1.CountDailyTotalsResponse{
				MethodCounts: []*countv1.MethodCount{
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := testServer.CountDailyTotals(tt.args.ctx, tt.args.req)
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
			args:    args{R.CTX, nil},
			wantErr: true,
		},
		{
			name: "empty from",
			args: args{R.CTX, &countv1.ListDailyTotalsRequest{
				StartDate: datepb.Date(day1),
			}},
			wantErr: true,
		},
		{
			name: "empty EndDate",
			args: args{R.CTX, &countv1.ListDailyTotalsRequest{
				EndDate: datepb.Date(day2),
			}},
			wantErr: true,
		},
		{
			name: "context error",
			args: args{R.ErrCTX, &countv1.ListDailyTotalsRequest{
				StartDate: datepb.Date(day1),
				EndDate:   datepb.Date(day2),
			}},
			wantErr: true,
		},
		{
			name: "not found",
			args: args{R.CTX, &countv1.ListDailyTotalsRequest{
				StartDate: datepb.Today(),
				EndDate:   datepb.Today(),
			}},
			wantErr: true,
		},
		{
			name: "success",
			args: args{R.CTX, &countv1.ListDailyTotalsRequest{
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
			got, err := testServer.ListDailyTotals(tt.args.ctx, tt.args.req)
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
			args:    args{R.CTX, &countv1.GetPeriodTotalsRequest{}},
			wantErr: true,
		},
		{
			name: "context error",
			args: args{R.ErrCTX, &countv1.GetPeriodTotalsRequest{Period: &date.Date{
				Year: 1986,
			}}},
			wantErr: true,
		},
		{
			name: "not found",
			args: args{R.CTX, &countv1.GetPeriodTotalsRequest{Period: &date.Date{
				Year: 1977,
			}}},
			wantErr: true,
		},
		{
			name: "year",
			args: args{R.CTX, &countv1.GetPeriodTotalsRequest{Period: &date.Date{
				Year: 1986,
			}}},
			want: &countv1.GetPeriodTotalsResponse{
				MethodCounts: []*countv1.MethodCount{
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
		},
		{
			name: "month",
			args: args{R.CTX, &countv1.GetPeriodTotalsRequest{Period: &date.Date{
				Year:  1986,
				Month: 3,
			}}},
			want: &countv1.GetPeriodTotalsResponse{
				MethodCounts: []*countv1.MethodCount{
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
		},
		{
			name: "day",
			args: args{R.CTX, &countv1.GetPeriodTotalsRequest{Period: &date.Date{
				Year:  1986,
				Month: 3,
				Day:   25,
			}}},
			want: &countv1.GetPeriodTotalsResponse{
				MethodCounts: []*countv1.MethodCount{
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := testServer.GetPeriodTotals(tt.args.ctx, tt.args.req)
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
