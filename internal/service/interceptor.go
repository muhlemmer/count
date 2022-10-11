package service

import (
	"context"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

type serverStreamCtx struct {
	grpc.ServerStream
	ctx context.Context
}

func (ss *serverStreamCtx) Context() context.Context {
	return ss.ctx
}

func StreamLogInterceptor(logger zerolog.Logger) func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &serverStreamCtx{
			ServerStream: ss,
			ctx:          logger.WithContext(ss.Context()),
		})
	}
}
