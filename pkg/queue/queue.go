// Package queue provides client side queueing, HTTP middleware and gRPC interceptor
// for request submission to a count gRPC server.
package queue

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/muhlemmer/count/internal/timer"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type request struct {
	ctx context.Context
	msg *countv1.AddRequest
}

// CountAddQueue provides a countv1.CountService_AddClient stream queue,
// with automatic reconnect on errors.
type CountAddQueue struct {
	ctx    context.Context
	client countv1.CountServiceClient
	opts   []grpc.CallOption

	wg     sync.WaitGroup
	queue  chan *request
	stream countv1.CountService_AddClient
}

func (c *CountAddQueue) reconnectCountAddClientStream() error {
	for {
		stream, err := func() (countv1.CountService_AddClient, error) {
			ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
			defer cancel()

			return c.client.Add(ctx, c.opts...)
		}()
		zerolog.Ctx(c.ctx).Err(err).Msg("count reconnect stream")
		if err == nil {
			c.stream = stream
			return nil
		}

		select {
		case <-timer.RandomTimer(time.Second/5, time.Second*5):
		case <-c.ctx.Done():
			return c.ctx.Err()
		}
	}
}

const countMsgDroppedFmt = "count message dropped: %s"

func (c *CountAddQueue) processQueue() {
	for entry := range c.queue {
		logger := zerolog.Ctx(c.ctx).With().Stringer("msg", entry.msg).Logger()

		if err := c.stream.Send(entry.msg); err != nil {
			if err = c.reconnectCountAddClientStream(); err != nil {
				logger.Warn().Err(err).Msgf(countMsgDroppedFmt, "reconnect failure")
				return
			}

			if err := c.stream.Send(entry.msg); err != nil {
				logger.Warn().Err(err).Msgf(countMsgDroppedFmt, "persistent send failure")
			} else {
				logger.Debug().Stringer("msg", entry.msg).Msg("count message sent")
			}
		}
	}
	c.stream.CloseSend()
}

// NewCountAddClient initiates a new CountServiceClient.Add stream on the ClientConn.
// The returned CountAddClient can be used to queue and send countv1.AddRequest messages.
// A seperate go routine is started for queue processing and automatic reconnection on failure.
//
// The context needs to remain available for automatic reconnection.
// When the context is expired or canceled, automatic reconnection will fail.
// However, existing entries in the queue will still be processed, as long as the stream does not break.
func NewCountAddClient(ctx context.Context, cc *grpc.ClientConn, opts ...grpc.CallOption) (*CountAddQueue, error) {
	client := countv1.NewCountServiceClient(cc)
	stream, err := client.Add(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("middleware; %w", err)
	}

	c := &CountAddQueue{
		ctx:    ctx,
		client: client,
		opts:   opts,
		queue:  make(chan *request, 1024),
		stream: stream,
	}

	c.wg.Add(1)
	go func() {
		c.processQueue()
		c.wg.Done()
	}()

	return c, nil
}

// Close the stream. Blocks untill the queue is emptied.
func (c *CountAddQueue) Close() {
	close(c.queue)
	c.wg.Wait()
}

// Queue a AddRequest. Blocks if the queue is full untill space is available.
// The context is used for logging only.
// Dropped messages are reported on the logger in context, using the Warn loglevel.
func (c *CountAddQueue) Queue(ctx context.Context, req *countv1.AddRequest) {
	c.queue <- &request{
		ctx: ctx,
		msg: req,
	}
}

// QueueOrDrop a AddRequest. Req is dropped if the queue is full,
// so QueueOrDrop is always a non-blocking action.
// The context is used for logging only.
// Dropped messages are reported on the logger in context, using the Warn loglevel.
func (c *CountAddQueue) QueueOrDrop(ctx context.Context, req *countv1.AddRequest) {
	select {
	case c.queue <- &request{
		ctx: ctx,
		msg: req,
	}:
	default:
		zerolog.Ctx(ctx).Warn().Msgf(countMsgDroppedFmt, "queue full")
	}
}

// Middleware for net/http which queues request data.
// The middleware never blocks. If the queue is full,
// the request message is dropped instead.
// Dropped messages are reported on the logger in the request context,
// using the Warn loglevel.
func (c *CountAddQueue) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.QueueOrDrop(r.Context(), &countv1.AddRequest{
			Method:           countv1.Method(countv1.Method_value[r.Method]),
			Path:             r.URL.Path,
			RequestTimestamp: timestamppb.Now(),
		})

		next.ServeHTTP(w, r)
	})
}

// UnaryInterceptor for gRPC, which queues request data.
// The interceptor never blocks. If the queue is full,
// the request message is dropped instead.
// Dropped messages are reported on the logger in the request context,
// using the Warn loglevel.
func (c *CountAddQueue) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		c.QueueOrDrop(ctx, &countv1.AddRequest{
			Method:           countv1.Method_GRPC,
			Path:             info.FullMethod,
			RequestTimestamp: timestamppb.Now(),
		})

		return handler(ctx, req)
	}
}
