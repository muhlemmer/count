package service

import (
	"testing"

	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

func Test_serverStreamCtx_Context(t *testing.T) {
	s := serverStreamCtx{ctx: testCTX}
	if err := s.Context().Err(); err != nil {
		t.Fatal(err)
	}
}

func TestStreamLogInterceptor(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	interceptor := StreamLogInterceptor(logger)
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		s := &CountServer{db: testDB}
		return s.Add(stream.(*serverStreamCtx).ServerStream.(countv1.CountService_AddServer))
	}
	mock := &mockAddServer{
		ctx:    testCTX,
		stream: testStream,
	}

	if err := interceptor(nil, mock, nil, handler); err != nil {
		t.Fatal(err)
	}
}
