package queue

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/muhlemmer/count/internal/db"
	"github.com/muhlemmer/count/internal/service"
	"github.com/muhlemmer/count/internal/tester"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	R          *tester.Resources
	testClient countv1.CountServiceClient
	testServer *grpc.Server
)

func TestMain(m *testing.M) {
	os.Exit(tester.Run(2*time.Minute, func(r *tester.Resources) int {
		R = r
		testServer = grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
		service.NewCountService(testServer, db.Wrap(R.Pool))

		lis, err := net.Listen("tcp", "127.0.0.1:9999")
		if err != nil {
			panic(err)
		}

		go func() {
			zerolog.Ctx(R.CTX).Err(testServer.Serve(lis)).Msg("grpc server terminated")
		}()
		defer testServer.GracefulStop()

		cc, err := grpc.DialContext(R.CTX, "127.0.0.1:9999",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			panic(err)
		}

		testClient = countv1.NewCountServiceClient(cc)
		return m.Run()
	}))
}

func TestCountAddQueue_reconnectCountAddClientStream(t *testing.T) {
	type fields struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "context error",
			fields:  fields{R.ErrCTX},
			wantErr: true,
		},
		{
			name:   "success",
			fields: fields{R.CTX},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CountAddQueue{
				ctx:    tt.fields.ctx,
				client: testClient,
			}
			if err := c.reconnectCountAddClientStream(); (err != nil) != tt.wantErr {
				t.Errorf("CountAddQueue.reconnectCountAddClientStream() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkCountAddQueue_processQueue(b *testing.B) {
	stream, err := testClient.Add(R.CTX)
	if err != nil {
		b.Fatal(err)
	}

	c := &CountAddQueue{
		ctx:    R.CTX,
		client: testClient,
		queue:  make(chan *request),
		stream: stream,
	}

	c.wg.Add(1)
	go func() {
		c.processQueue()
		c.wg.Done()
	}()

	b.Run("bench", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			c.queue <- &request{R.CTX, &countv1.AddRequest{
				Method:           countv1.Method_DELETE,
				Path:             "/foo/bar",
				RequestTimestamp: timestamppb.Now(),
			}}
		}
	})

	close(c.queue)
	c.wg.Wait()
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

func TestNewCountAddClient(t *testing.T) {
	type args struct {
		ctx    context.Context
		client countv1.CountServiceClient
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "context error",
			args:    args{R.ErrCTX, testClient},
			wantErr: true,
		},
		{
			name: "success",
			args: args{R.CTX, testClient},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := NewCountAddClient(tt.args.ctx, tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCountAddClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for _, req := range testStream {
					q.Queue(R.CTX, req)
				}
				q.Close()
			}
		})
	}
}

func TestCountAddQueue_QueueOrDrop(t *testing.T) {
	c := &CountAddQueue{
		queue: make(chan *request, 2),
	}

	for _, req := range testStream {
		c.QueueOrDrop(R.CTX, req)
	}
}

func TestCountAddQueue_Middleware(t *testing.T) {
	c := &CountAddQueue{
		queue: make(chan *request),
	}

	r := httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	w := httptest.NewRecorder()

	const want = "Hello, world!"

	c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, world!")
	}))(w, r)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want %d", res.StatusCode, http.StatusOK)
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if got := string(b); got != want {
		t.Errorf("CountAddQueue.Middleware = %s, want %s", got, want)
	}
}

func TestCountAddQueue_UnaryInterceptor(t *testing.T) {
	c := &CountAddQueue{
		queue: make(chan *request),
	}

	var handlerCalled bool
	c.UnaryInterceptor()(R.CTX, nil, &grpc.UnaryServerInfo{FullMethod: "foo.bar"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return nil, nil
	})

	if !handlerCalled {
		t.Error("CountAddQueue.UnaryInterceptor handler not called")
	}
}
