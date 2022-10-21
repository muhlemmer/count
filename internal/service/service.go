package service

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/muhlemmer/count/internal/db"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CountServer struct {
	countv1.UnimplementedCountServiceServer

	db *db.DB
}

func NewCountService(s grpc.ServiceRegistrar, db *db.DB) {
	countv1.RegisterCountServiceServer(s, &CountServer{
		db: db,
	})
}

func (s *CountServer) Add(as countv1.CountService_AddServer) error {
	errChan := make(chan error, 10)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		var abort atomic.Bool
		for {
			req, err := as.Recv()
			if err == io.EOF || abort.Load() {
				if err = as.SendAndClose(&countv1.AddResponse{}); err != nil {
					errChan <- err
				}
				return
			}
			if err != nil {
				errChan <- err
				return
			}

			wg.Add(1)
			go func() {
				ctx, cancel := context.WithTimeout(as.Context(), time.Minute)
				defer cancel()

				var (
					method    = req.GetMethod()
					path      = req.GetPath()
					requestTS = req.GetRequestTimestamp().AsTime()
				)

				// add some details to the logger passed to the DB layer.
				logger := zerolog.Ctx(ctx).With().Stringer("method", method).Str("path", path).Time("request_timestamp", requestTS).Logger()
				ctx = logger.WithContext(ctx)

				err := s.db.InsertMethodRequest(ctx, method, path, requestTS)
				logger.Err(err).Msg("count service stream add request")

				if err != nil {
					errChan <- err
					abort.Store(true)
				}
				wg.Done()
			}()
		}
	}()

	// conclusion will be read only once,
	// to determine the first encountered error.
	// The errChan channel needs to be drained and
	// the surplus of errors are discarded.
	conclusion := make(chan error, 1)
	go func() {
		for err := range errChan {
			select {
			case conclusion <- err:
			default:
			}
		}
		close(conclusion)
	}()

	// wait on all go routines to be terminated
	// and close the reports channel.
	wg.Wait()
	close(errChan)
	return <-conclusion
}

func (s *CountServer) CountDailyTotals(ctx context.Context, req *countv1.CountDailyTotalsRequest) (*countv1.CountDailyTotalsResponse, error) {
	counts, err := s.db.CountDailyMethodTotals(ctx, req.GetDate().AsTime())
	if err != nil {
		return nil, err
	}

	return &countv1.CountDailyTotalsResponse{
		MethodCounts: counts,
	}, nil
}

func (s *CountServer) ListDailyTotals(ctx context.Context, req *countv1.ListDailyTotalsRequest) (*countv1.ListDailyTotalsResponse, error) {
	var (
		from = req.GetFrom()
		till = req.GetTill()
	)
	if from == nil || till == nil {
		return nil, status.Error(codes.InvalidArgument, "a valid date interval is required")
	}

	counts, err := s.db.ListDailyTotals(ctx, from.AsTime(), till.AsTime())
	if err != nil {
		return nil, err
	}
	if len(counts) == 0 {
		return nil, status.Error(codes.NotFound, "no results found")
	}

	return &countv1.ListDailyTotalsResponse{
		MethodCounts: counts,
	}, nil
}
